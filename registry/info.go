package registry

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/duglin/dlog"
)

type RequestInfo struct {
	tx               *Tx
	Registry         *Registry
	BaseURL          string // host+path to root of registry
	OriginalPath     string
	OriginalRequest  *http.Request       `json:"-"`
	OriginalResponse http.ResponseWriter `json:"-"`
	RootPath         string              // rootPath or "" for registry itself
	Parts            []string
	Root             string // path into the Registry of the request
	Abstract         string
	GroupType        string
	GroupUID         string
	GroupModel       *GroupModel
	ResourceType     string
	ResourceUID      string
	ResourceModel    *ResourceModel
	VersionUID       string
	What             string            // Registry, Coll, Entity
	Flags            map[string]string // Query params (and str value, if there)
	Inlines          []*Inline
	Filters          [][]*FilterExpr // [OR][AND] filter=e,e(and) &(or) filter=e
	ShowDetails      bool            //	is $details present

	StatusCode int
	SentStatus bool
	HTTPWriter HTTPWriter `json:"-"`

	// extra stuff if we ever need to pass around data while processing
	extras map[string]any
}

var explicitInlines = []string{"capabilities", "model"}
var nonModelInlines = append([]string{"*"}, explicitInlines...)
var rootPaths = []string{"capabilities", "model", "export"}

type Inline struct {
	Path    string    // value from ?inline query param
	PP      *PropPath // PP for 'value'
	NonWild *PropPath // PP for value w/o .* if present, else nil
}

func (info *RequestInfo) AddInline(path string) error {
	// use "*" to inline all
	// path = strings.TrimLeft(path, "/.") // To be nice
	originalPath := path

	if ArrayContains(nonModelInlines, path) {
		info.Inlines = append(info.Inlines, &Inline{
			Path:    NewPPP(path).DB(),
			PP:      NewPPP(path),
			NonWild: nil,
		})
		return nil
	}

	pp, err := PropPathFromUI(path)
	if err != nil {
		return err
	}

	storeInline := &Inline{
		Path:    pp.DB(),
		PP:      pp,
		NonWild: nil,
	}

	if pp.Bottom() == "*" {
		pp = pp.RemoveLast()
		storeInline.NonWild = pp
	}

	// Check to make sure the requested inline attribute exists, else error

	hasErr := false
	for _, group := range info.Registry.Model.Groups {
		gPPP := NewPPP(group.Plural)

		if pp.Equals(gPPP) {
			info.Inlines = append(info.Inlines, storeInline)
			return nil
		}

		for _, res := range group.Resources {
			// Check for wildcard available ones first
			rPPP := gPPP.P(res.Plural)
			vPPP := rPPP.P("versions")

			// Check for ones that allow * at the end, first
			if pp.Equals(rPPP) || pp.Equals(vPPP) {
				info.Inlines = append(info.Inlines, storeInline)
				return nil
			}

			// Now look for ones that don't allow wildcards
			if pp.Equals(rPPP.P(res.Singular)) ||
				pp.Equals(rPPP.P("meta")) ||
				pp.Equals(vPPP.P(res.Singular)) {

				// We have a match, but these don't allow wildcards, so err
				// if * was in ?inline value
				if storeInline.NonWild != nil {
					hasErr = true
					break
				}

				info.Inlines = append(info.Inlines, storeInline)
				return nil
			}
		}
		if hasErr {
			break
		}
	}

	// // Convert back to UI version for the error message
	// path = pp.UI()
	path = originalPath

	// Remove Abstract value just to print a nicer error message
	if info.Abstract != "" && strings.HasPrefix(path, info.Abstract) {
		path = path[len(info.Abstract)+1:]
	}

	return fmt.Errorf("Invalid 'inline' value: %s", path)
}

func (info *RequestInfo) IsInlineSet(entityPath string) bool {
	if entityPath == "" {
		entityPath = "*"
	}
	for _, inline := range info.Inlines {
		if inline.Path == entityPath {
			return true
		}
	}
	return false
}

func (info *RequestInfo) ShouldInline(entityPath string) bool {
	// ePP is the abstract path to the prop we're checking/serializing
	// iPP is the ?inline value the the client provided
	// Note that iPP will likely end with ","
	// e.g. Inline cmp: "dirs,datas" in "dirs,files,"

	ePP, _ := PropPathFromDB(entityPath) // entity-PP

	for _, inline := range info.Inlines {
		iPP := inline.PP
		log.VPrintf(4, "Inline cmp: %q in %q",
			ePP.DB(), inline.PP.DB())

		// * doesn't include "model"... because they're special, they need to
		// be explicit if they want to include those
		// ||
		// prop == ?inline-value
		// ||
		// inline-value has prop as a prefix. Inline parents of requested value
		//     e.g. inline=endpoints.message has endpoints as prefix
		// ||
		// inline-value ends with "*", prop has inline-value as prefix
		if (iPP.Top() == "*" && !ArrayContains(explicitInlines, ePP.UI())) ||
			ePP.Equals(iPP) ||
			iPP.HasPrefix(ePP) ||
			(inline.NonWild != nil && ePP.HasPrefix(inline.NonWild)) {
			// (iPP.Len() > 1 && iPP.Bottom() == "*" && ePP.HasPrefix(iPP.RemoveLast())) {

			log.VPrintf(4, "   match: %q in %q",
				ePP.DB(), inline.PP.DB())
			return true
		}
	}
	return false
}

func (ri *RequestInfo) Write(b []byte) (int, error) {
	return ri.HTTPWriter.Write(b)
}

func (ri *RequestInfo) AddHeader(name, value string) {
	ri.HTTPWriter.AddHeader(name, value)
}

type FilterExpr struct {
	// User provided
	Path     string // endpoints.id  TODO store a PropPath?
	Value    string // myEndpoint
	Operator int    // FILTER_PRESENT, ...

	// helpers
	Abstract string
	PropName string
}

func ParseRequest(tx *Tx, w http.ResponseWriter, r *http.Request) (*RequestInfo, error) {
	path := strings.Trim(r.URL.Path, " /")
	info := &RequestInfo{
		tx: tx,

		OriginalPath:     path,
		OriginalRequest:  r,
		OriginalResponse: w,
		Registry:         GetDefaultReg(tx),
		BaseURL:          "http://" + r.Host,

		extras: map[string]any{},
	}

	if r.TLS != nil {
		info.BaseURL = "https" + info.BaseURL[4:]
	} else if tmp := r.Header.Get("Referer"); tmp != "" {
		if strings.HasPrefix(tmp, "https:") {
			info.BaseURL = "https" + info.BaseURL[4:]
		}
	}

	PanicIf(info.Registry == nil, "No default registry")

	if info.Registry != nil && tx.Registry == nil {
		tx.Registry = info.Registry
	}

	info.HTTPWriter = DefaultHTTPWriter(info)

	if log.GetVerbose() > 2 {
		defer func() { log.VPrintf(3, "Info:\n%s\n", ToJSON(info)) }()
	}

	if tmp := r.Header.Get("xRegistry~User"); tmp != "" {
		tx.User = tmp
	}

	// Notice boolean flags end up with "" as a value
	info.Flags = map[string]string{}
	params := info.OriginalRequest.URL.Query()
	for _, flag := range AllowableFlags {
		val, ok := params[flag]
		if ok {
			info.Flags[flag] = val[0]
		}
	}

	err := info.ParseRequestURL()
	if err != nil {
		return info, err
	}

	tx.IgnoreEpoch = info.HasFlag("noepoch")
	tx.IgnoreDefaultVersionSticky = info.HasFlag("nodefaultversionsticky")
	tx.IgnoreDefaultVersionID = info.HasFlag("nodefaultversionid")

	if info.HasFlag("inline") {
		// OLD: Only pick up inlining values if we're doing a GET, not write ops
		// if  strings.EqualFold(r.Method, "GET")
		for _, value := range info.GetFlagValues("inline") {
			for _, p := range strings.Split(value, ",") {
				if p == "" || p == "*" {
					p = "*"
				} else {
					// if we're not at the root then we need to twiddle
					// the inline path to add the HTTP Path as a prefix
					if info.Abstract != "" {
						// want: p = info.Abstract + "." + p  in UI format
						absPP, err := PropPathFromPath(info.Abstract)
						if err != nil {
							return info, err
						}
						pPP, err := PropPathFromUI(p)
						if err != nil {
							return info, err
						}
						p = absPP.Append(pPP).UI()
					}
				}
				if err := info.AddInline(p); err != nil {
					return info, err
				}
			}
		}
	}

	// if  strings.EqualFold(r.Method, "GET")
	err = info.ParseFilters()
	return info, err
}

func (info *RequestInfo) ParseFilters() error {
	for _, filterQ := range info.GetFlagValues("filter") {
		// ?filter=path.to.attribute[=value],* & filter=...

		filterQ = strings.TrimSpace(filterQ)
		exprs := strings.Split(filterQ, ",")
		AndFilters := ([]*FilterExpr)(nil)
		for _, expr := range exprs {
			expr = strings.TrimSpace(expr)
			if expr == "" {
				continue
			}

			filterOp := FILTER_PRESENT

			path, value, found := strings.Cut(expr, "!=")
			if found {
				// Note that "xxx!=null" is the same as "xxx"
				if value != "null" {
					filterOp = FILTER_NOT_EQUAL
				}
			} else {
				path, value, found = strings.Cut(expr, "=")
				if found {
					if value == "null" {
						filterOp = FILTER_ABSENT
					} else {
						filterOp = FILTER_EQUAL
					}
				}
				// No "=" or "!=" means FILTER_PRESENT
			}

			pp, err := PropPathFromUI(path)
			if err != nil {
				return err
			}
			path = pp.DB()

			/*
				if info.What != "Coll" && strings.Index(path, "/") < 0 {
					return fmt.Errorf("A filter with just an attribute name (%s) "+
						"isn't allowed in this context", path)
				}
			*/

			if info.Abstract != "" {
				// Want: path = abs + "," + path in DB format
				absPP, _ := PropPathFromPath(info.Abstract)
				absPP = absPP.Append(pp)
				path = absPP.DB()
			}

			filter := &FilterExpr{
				Path:     path,
				Value:    value,
				Operator: filterOp,
			}
			filter.Abstract, filter.PropName = SplitProp(info.Registry, path)

			if AndFilters == nil {
				AndFilters = []*FilterExpr{}
			}
			AndFilters = append(AndFilters, filter)
		}

		if AndFilters != nil {
			if info.Filters == nil {
				info.Filters = [][]*FilterExpr{}
			}
			info.Filters = append(info.Filters, AndFilters)
		}
	}
	return nil
}

// path.DB() -> abstract.Abstract() + propName.DB()
func SplitProp(reg *Registry, path string) (string, string) {
	pp := MustPropPathFromDB(path)

	abs := &PropPath{}

	if pp.Top() != "" {
		if gm := reg.Model.FindGroupModel(pp.Top()); gm != nil {
			abs = abs.Append(pp.First())
			pp = pp.Next()

			if rm := gm.Resources[pp.Top()]; rm != nil {
				abs = abs.Append(pp.First())
				pp = pp.Next()

				next := pp.Top()
				if next == "meta" || next == "versions" {
					abs = abs.Append(pp.First())
					pp = pp.Next()
				}
			}
		}
	}

	return abs.Abstract(), pp.DB()
}

func (info *RequestInfo) ParseRequestURL() error {
	path := strings.Trim(info.OriginalPath, " /")
	info.Parts = strings.Split(path, "/")
	if len(info.Parts) == 1 && info.Parts[0] == "" {
		info.Parts = []string{}
	}

	if len(info.Parts) > 0 && strings.HasPrefix(info.Parts[0], "reg-") {
		info.BaseURL += "/" + info.Parts[0]
		name := info.Parts[0][4:]
		info.Parts = info.Parts[1:] // shift

		reg, err := FindRegistry(info.tx, name)
		if reg == nil {
			extra := ""
			if err != nil {
				extra = ": " + err.Error()
			}
			return fmt.Errorf("Can't find registry %q%s", name, extra)
		}
		info.tx.Rollback() // Not sure why
		info.tx.Registry = reg
		info.Registry = reg
	}

	if len(info.Parts) == 0 {
		info.Parts = nil
		info.What = "Registry"
		return nil
	}

	// /???
	info.RootPath = ""
	if len(info.Parts) > 0 && ArrayContains(rootPaths, info.Parts[0]) {
		info.RootPath = info.Parts[0]
		return nil
	}

	// /GROUPs
	if strings.HasSuffix(info.Parts[0], "$details") {
		return fmt.Errorf("$details isn't allowed on %q", "/"+info.Parts[0])
	}

	gModel := (*GroupModel)(nil)
	if info.Registry.Model != nil && info.Registry.Model.Groups != nil {
		gModel = info.Registry.Model.Groups[info.Parts[0]]
	}
	if gModel == nil && (info.Parts[0] != "model" || len(info.Parts) > 1) {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Unknown Group type: %s", info.Parts[0])
	}
	info.GroupModel = gModel
	info.GroupType = info.Parts[0]
	info.Root += info.Parts[0]
	info.Abstract += info.Parts[0]

	if len(info.Parts) == 1 {
		info.What = "Coll"
		return nil
	}

	// /GROUPs/gID
	if strings.HasSuffix(info.Parts[1], "$details") {
		return fmt.Errorf("$details isn't allowed on %q",
			"/"+strings.Join(info.Parts[:2], "/"))
	}

	info.GroupUID = info.Parts[1]
	info.Root += "/" + info.Parts[1]

	if len(info.Parts) == 2 {
		info.What = "Entity"
		return nil
	}

	// /GROUPs/gID/RESOURCEs
	if strings.HasSuffix(info.Parts[2], "$details") {
		return fmt.Errorf("$details isn't allowed on %q",
			"/"+strings.Join(info.Parts[:3], "/"))
	}

	rModel := (*ResourceModel)(nil)
	if gModel.Resources != nil {
		rModel = gModel.Resources[info.Parts[2]]
	}
	if rModel == nil {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Unknown Resource type: %s", info.Parts[2])
	}
	info.ResourceModel = rModel
	info.ResourceType = info.Parts[2]
	info.Root += "/" + info.Parts[2]
	info.Abstract += "/" + info.Parts[2]

	if len(info.Parts) == 3 {
		info.What = "Coll"
		return nil
	}

	// /GROUPs/gID/RESOURCEs/rID
	info.ResourceUID, info.ShowDetails =
		strings.CutSuffix(info.Parts[3], "$details")

	info.Root += "/" + info.ResourceUID

	// GROUPs/gID/RESOURCEs/rID
	if len(info.Parts) == 4 {
		info.Parts[3] = info.ResourceUID
		info.What = "Entity"
		return nil
	}

	// GROUPs/gID/RESOURCEs/rID/???
	if info.ShowDetails {
		return fmt.Errorf("$details isn't allowed on %q",
			"/"+strings.Join(info.Parts[:4], "/"))
	}

	if strings.HasSuffix(info.Parts[4], "$details") {
		return fmt.Errorf("$details isn't allowed on %q",
			"/"+strings.Join(info.Parts[:5], "/"))
	}

	if info.Parts[4] != "versions" && info.Parts[4] != "meta" {
		info.StatusCode = http.StatusNotFound
		return fmt.Errorf("Expected \"versions\" or \"meta\", got: %s",
			info.Parts[4])
	}

	// GROUPs/gID/RESOURCEs/rID/[meta|versions]
	if len(info.Parts) >= 5 {
		if info.Parts[4] == "meta" {
			if len(info.Parts) > 5 {
				// GROUPs/gID/RESOURCEs/rID/meta/???
				info.StatusCode = http.StatusNotFound
				return fmt.Errorf("URL is too long")
			}

			// GROUPs/gID/RESOURCEs/rID/meta
			info.Root += "/meta"
			info.Abstract += "/meta"
			info.What = "Entity"
			return nil
		}

		// GROUPs/gID/RESOURCEs/rID/versions
		info.Root += "/versions"
		info.Abstract += "/versions"
		if len(info.Parts) == 5 {
			info.What = "Coll"
			return nil
		}

	}

	// GROUPs/gID/RESOURCEs/rID/versions/vID
	info.VersionUID, info.ShowDetails =
		strings.CutSuffix(info.Parts[5], "$details")

	info.Root += "/" + info.VersionUID

	if len(info.Parts) == 6 {
		if info.VersionUID == "" {
			return fmt.Errorf("Version id in URL can't be blank")
		}

		info.Parts[5] = info.VersionUID
		info.What = "Entity"
		return nil
	}

	info.StatusCode = http.StatusNotFound
	return fmt.Errorf("URL is too long")
}

// Get query parameter value
func (info *RequestInfo) GetFlag(name string) string {
	if !info.Registry.Capabilities.FlagEnabled(name) {
		return ""
	}
	return info.Flags[name]
}

func (info *RequestInfo) GetFlagValues(name string) []string {
	if !info.Registry.Capabilities.FlagEnabled(name) {
		return nil
	}
	return info.OriginalRequest.URL.Query()[name]
}

func (info *RequestInfo) HasFlag(name string) bool {
	if !info.Registry.Capabilities.FlagEnabled(name) {
		return false
	}
	_, ok := info.Flags[name]
	return ok
}

func (info *RequestInfo) DoDocView() bool {
	return info.HasFlag("doc") || info.RootPath == "export"
}
