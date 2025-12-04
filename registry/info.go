package registry

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

type RequestInfo struct {
	tx               *Tx
	Registry         *Registry
	BaseURL          string              // host+path to root of registry
	OriginalPath     string              // /GROUPs/gID...?inline...
	OriginalRequest  *http.Request       `json:"-"`
	OriginalResponse http.ResponseWriter `json:"-"`
	RootPath         string              // "", "model", "export", ...
	Parts            []string            // Split /GROUPS/gID of OriginalPath
	Root             string              // GROUPS/gID/..
	Abstract         string              // /GROUPS/RESOUCES (no IDs)
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
	SortKey          string          // [-]AttrName  - => descending

	StatusCode int
	SentStatus bool
	HTTPWriter HTTPWriter `json:"-"`

	ProxyHost string
	ProxyPath string

	// extra stuff if we ever need to pass around data while processing
	extras map[string]any
}

var explicitInlines = []string{"capabilities", "model", "modelsource"}
var nonModelInlines = append([]string{"*"}, explicitInlines...)
var rootPaths = []string{"capabilities", "capabilitiesoffered", "export",
	"model", "modelsource", "proxy"}

type Inline struct {
	Path    string    // value from ?inline query param
	PP      *PropPath // PP for 'value'
	NonWild *PropPath // PP for value w/o .* if present, else nil
}

func (info *RequestInfo) AddInline(path string) *XRError {
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
		return NewXRError("bad_inline", info.GetParts(0),
			"inline_value="+path,
			"error_detail="+err.Error())
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

		rList := group.GetResourceList()
		for _, rName := range rList {
			res := group.FindResourceModel(rName)
			PanicIf(res == nil, "Not found: %s", rName)
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

	return NewXRError("inline_noninlineable", info.GetParts(0), "name="+path)
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

func NewRequestInfo(w http.ResponseWriter, r *http.Request) *RequestInfo {
	info := &RequestInfo{
		OriginalPath:     strings.Trim(r.URL.Path, " /"),
		OriginalRequest:  r,
		OriginalResponse: w,
		BaseURL:          "http://" + r.Host,
		extras:           map[string]any{},
	}

	info.HTTPWriter = DefaultHTTPWriter(info)

	if r.TLS != nil {
		info.BaseURL = "https" + info.BaseURL[4:]
	} else if tmp := r.Header.Get("Referer"); tmp != "" {
		if strings.HasPrefix(tmp, "https:") {
			info.BaseURL = "https" + info.BaseURL[4:]
		}
	}

	return info
}

func ParseRequest(tx *Tx, w http.ResponseWriter, r *http.Request) (*RequestInfo, *XRError) {

	info := NewRequestInfo(w, r)
	info.tx = tx

	// See which registry to use and twiddle some stuff in info if needed
	xErr := info.ParseRegistryURL()
	if xErr != nil {
		return info, xErr
	}

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Methods",
		"GET, PATCH, POST, PUT, DELETE")
	w.Header().Add("Link", fmt.Sprintf("<%s>;rel=xregistry-root", info.BaseURL))

	if log.GetVerbose() > 2 {
		defer func() { log.VPrintf(3, "Info:\n%s\n", ToJSON(info)) }()
	}

	if tmp := r.Header.Get("xRegistry~User"); tmp != "" {
		info.tx.User = tmp
	}

	// Parse the incoming URL and setup more stuff in info, like Groups...
	xErr = info.ParseRequestURL()
	if xErr != nil {
		return info, xErr
	}

	return info, nil
}

func (info *RequestInfo) ParseFilters() *XRError {
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
				return NewXRError("bad_filter", info.GetParts(0),
					"filter_name="+path,
					"error_detail="+err.Error())
			}
			path = pp.DB()

			/*
				if info.What != "Coll" && strings.Index(path, "/") < 0 {
				return NewXRError("bad_filter", info.GetParts(0),
				"filter_name=" + path,
				"error_detail=" +
				fmt.Sprintf("a filter with just an attribute " +
				"name (%s) isn't allowed in this context",
				path)
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

			if rm := gm.FindResourceModel(pp.Top()); rm != nil {
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

// This will extract the "reg-xxx" part of the URL if there and choose the
// appropriate Registry to use. It'll update info's BaseURL based on reg-
// This will populate some initial stuff in the "info" struct too, like
// Registry.
func (info *RequestInfo) ParseRegistryURL() *XRError {
	path := strings.Trim(info.OriginalPath, " /")

	if len(path) > 0 && strings.HasPrefix(path, "reg-") {
		regName, rest, _ := strings.Cut(path, "/")
		info.BaseURL += "/" + regName
		info.OriginalPath = rest
		name := regName[4:] // remove leading "reg-"

		reg, xErr := FindRegistry(info.tx, name, FOR_READ)
		if xErr != nil {
			return NewXRError("server_error", info.GetParts(0)).
				SetDetail(xErr.GetTitle())
		}
		if reg == nil {
			return NewXRError("not_found", name).
				SetDetailf("Can't find registry %q", name)
		}
		info.Registry = reg
		info.tx.Registry = reg
	} else {
		info.Registry = GetDefaultReg(info.tx)
		info.tx.Registry = info.Registry
	}
	return nil
}

func (info *RequestInfo) ParseRequestURL() *XRError {
	log.VPrintf(4, "ParseRequestURL:\n%s", ToJSON(info))
	log.VPrintf(4, "Req: %#v", info.OriginalRequest.URL)

	// Notice boolean flags end up with "" as a value
	info.Flags = map[string]string{}
	params := info.OriginalRequest.URL.Query()
	for _, flag := range AllowableFlags {
		val, ok := params[flag]
		if ok {
			info.Flags[flag] = val[0]
		}
	}

	if xErr := info.ParseRequestPath(); xErr != nil {
		return xErr
	}

	// Some of these have to come after we parse the path so that the
	// group/resource info is setup - for verification

	// Let's do some query parameter stuff.
	// It's ok for tx to be nil, but only when we're doing a read operation
	if info.tx != nil {
		info.tx.IgnoreEpoch = info.HasFlag("ignoreepoch")
		info.tx.IgnoreDefaultVersionSticky = info.HasFlag("ignoredefaultversionsticky")
		info.tx.IgnoreDefaultVersionID = info.HasFlag("ignoredefaultversionid")
	}

	if info.HasFlag("inline") {
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
							return NewXRError("bad_request", info.GetParts(0),
								"error_detail="+
									fmt.Sprintf("Error processing path "+
										"(%s): %s",
										info.Abstract, err.Error()))
						}
						pPP, err := PropPathFromUI(p)
						if err != nil {
							return NewXRError("bad_inline", info.GetParts(0),
								"inline_value="+p,
								"error_detail="+err.Error())
						}
						p = absPP.Append(pPP).UI()
					}
				}
				if xErr := info.AddInline(p); xErr != nil {
					return xErr
				}
			}
		}
	}

	// Do some error checking on "collections"
	if info.HasFlag("collections") {
		if !(info.GroupType == "" ||
			(info.GroupUID != "" && info.ResourceType == "")) {
			return NewXRError("bad_flag", "/"+info.OriginalPath,
				"flag=collections").
				SetDetail("?collections is only allow on the " +
					"Registry or Group instance level.")
		}
		// Force inline=* to be on
		info.AddInline("*")
	}

	if info.HasFlag("sort") {
		if info.What != "Coll" {
			return NewXRError("sort_noncollection", info.GetParts(0))
		}

		sortStr := info.GetFlag("sort")
		name, ascDesc, _ := strings.Cut(sortStr, "=")
		if name == "" {
			return NewXRError("bad_sort", info.GetParts(0),
				"sort_value="+sortStr,
				"error_detail=missing \"sort\" attribute name")
		}
		if ascDesc != "" && ascDesc != "asc" && ascDesc != "desc" {
			return NewXRError("bad_sort", info.GetParts(0),
				"sort_value="+sortStr,
				"error_detail="+
					fmt.Sprintf("invalid \"sort\" order %q", ascDesc))
		}
		// info.SortKey = name
		pp, err := PropPathFromUI(name)
		if err != nil {
			return NewXRError("bad_sort", info.GetParts(0),
				"sort_value="+sortStr,
				"error_detail="+
					fmt.Sprintf("bad attribute name(%s): %s",
						name, err.Error()))
		}
		info.SortKey = pp.DB()
		if ascDesc == "desc" {
			info.SortKey = "-" + info.SortKey
		}
	}

	return info.ParseFilters()
}

func (info *RequestInfo) ParseRequestPath() *XRError {
	// Now process the URL path
	log.VPrintf(4, "ParseRequestPath: %q", info.OriginalPath)

	path := strings.Trim(info.OriginalPath, " /")
	info.Parts = strings.Split(path, "/")

	if len(info.Parts) == 1 && info.Parts[0] == "" {
		info.Parts = []string{}
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
		return NewXRError("bad_details", "/"+info.Parts[0])
	}

	gModel := (*GroupModel)(nil)
	if info.Registry.Model != nil && info.Registry.Model.Groups != nil {
		gModel = info.Registry.Model.Groups[info.Parts[0]]
	}
	if gModel == nil &&
		(!ArrayContains(rootPaths, info.Parts[0]) || len(info.Parts) > 1) {

		return NewXRError("not_found", info.GetParts(1)).
			SetDetailf("Unknown Group type: %s.", info.Parts[0])
	}
	info.GroupModel = gModel
	info.GroupType = info.Parts[0]
	info.Root += info.Parts[0]
	info.Abstract += info.Parts[0]

	if info.GroupType == "" {
		return NewXRError("bad_request", info.GetParts(1),
			"error_detail=Group type in URL cannot be an empty string")
	}

	if len(info.Parts) == 1 {
		info.What = "Coll"
		return nil
	}

	// /GROUPs/gID
	if strings.HasSuffix(info.Parts[1], "$details") {
		return NewXRError("bad_details", info.GetParts(2))
	}

	info.GroupUID = info.Parts[1]
	info.Root += "/" + info.Parts[1]

	if info.GroupUID == "" {
		return NewXRError("bad_request", info.GetParts(2),
			"error_detail="+
				fmt.Sprintf("\"%sid\" value in URL cannot be an empty string",
					info.GroupModel.Singular))
	}

	if len(info.Parts) == 2 {
		info.What = "Entity"
		return nil
	}

	// /GROUPs/gID/RESOURCEs
	if strings.HasSuffix(info.Parts[2], "$details") {
		return NewXRError("bad_details", info.GetParts(3))
	}

	if info.Parts[2] == "" {
		return NewXRError("bad_request", info.GetParts(3),
			"error_detail=Resource type in URL cannot be an empty string")
	}

	rModel := gModel.FindResourceModel(info.Parts[2])
	if rModel == nil {
		return NewXRError("not_found", info.GetParts(3)).
			SetDetailf("Unknown Resource type: %s.", info.Parts[2])
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

	if info.ResourceUID == "" {
		return NewXRError("bad_request", info.GetParts(4),
			"error_detail="+
				fmt.Sprintf("\"%sid\" value in URL cannot be an empty string",
					info.ResourceModel.Singular))
	}

	// GROUPs/gID/RESOURCEs/rID
	if len(info.Parts) == 4 {
		info.Parts[3] = info.ResourceUID
		info.What = "Entity"
		return nil
	}

	// GROUPs/gID/RESOURCEs/rID/???
	if info.ShowDetails {
		return NewXRError("bad_details", info.GetParts(4))
	}

	if strings.HasSuffix(info.Parts[4], "$details") {
		return NewXRError("bad_details", info.GetParts(5))
	}

	if info.Parts[4] != "versions" && info.Parts[4] != "meta" {
		return NewXRError("not_found", info.GetParts(5)).
			SetDetailf("Expected \"versions\" or \"meta\", got: %s",
				info.Parts[4])
	}

	// GROUPs/gID/RESOURCEs/rID/[meta|versions]
	if len(info.Parts) >= 5 {
		if info.Parts[4] == "meta" {
			if len(info.Parts) > 5 {
				// GROUPs/gID/RESOURCEs/rID/meta/???
				return NewXRError("not_found", info.GetParts(0))
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

	if info.VersionUID == "" {
		return NewXRError("bad_request", info.GetParts(6),
			"error_detail="+
				fmt.Sprintf("\"versionid\" value in URL cannot be an empty string"))
	}

	if len(info.Parts) == 6 {
		info.Parts[5] = info.VersionUID
		info.What = "Entity"
		return nil
	}

	return NewXRError("not_found", info.GetParts(0))
}

// Get query parameter value
func (info *RequestInfo) GetFlag(name string) string {
	if info.Registry == nil || info.Registry.Capabilities == nil ||
		!info.Registry.Capabilities.FlagEnabled(name) {
		return ""
	}
	return info.Flags[name]
}

func (info *RequestInfo) GetFlagValues(name string) []string {
	if info.Registry == nil || info.Registry.Capabilities == nil ||
		!info.Registry.Capabilities.FlagEnabled(name) {
		return nil
	}
	return info.OriginalRequest.URL.Query()[name]
}

func (info *RequestInfo) HasFlag(name string) bool {
	if info.Registry == nil || info.Registry.Capabilities == nil ||
		!info.Registry.Capabilities.FlagEnabled(name) {

		return false
	}
	_, ok := info.Flags[name]
	return ok
}

func (info *RequestInfo) FlagEnabled(name string) bool {
	if info.Registry == nil || info.Registry.Capabilities == nil ||
		!info.Registry.Capabilities.FlagEnabled(name) {

		return false
	}
	return true
}

func (info *RequestInfo) APIEnabled(name string) bool {
	if info.Registry == nil || info.Registry.Capabilities == nil ||
		!info.Registry.Capabilities.APIEnabled(name) {

		return false
	}
	return true
}

func (info *RequestInfo) DoDocView() bool {
	return info.HasFlag("doc") || info.RootPath == "export"
}

func (info *RequestInfo) GetParts(num int) string {
	PanicIf(num < 0, "Can't be %d", num)
	if num == 0 {
		return "/" + strings.TrimLeft(info.OriginalPath, "/")
	}
	PanicIf(num > len(info.Parts), "Asking for too many (%d): %s", num,
		info.OriginalPath)
	return "/" + strings.Join(info.Parts[:num], "/")
}
