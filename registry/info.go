package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

func (info *RequestInfo) Write(b []byte) (int, error) {
	if !info.SentStatus {
		// Set all response headers before writing status

		// CORS headers
		info.AddHeader("Access-Control-Allow-Origin", "*")
		methods := info.GetAllowedMethods()
		methodsStr := strings.Join(methods, ", ")
		info.AddHeader("Access-Control-Allow-Methods", methodsStr)
		info.AddHeader("Access-Control-Expose-Headers", "Link")

		// Reflect whatever headers the browser's preflight asked for
		// (e.g. Content-Type) so cross-origin PUT/PATCH/POST/DELETE
		// requests with a JSON body aren't blocked by CORS.
		reqHeaders := info.OriginalRequest.Header.Get(
			"Access-Control-Request-Headers")
		if reqHeaders != "" {
			info.AddHeader("Access-Control-Allow-Headers", reqHeaders)
		} else {
			info.AddHeader("Access-Control-Allow-Headers", "Content-Type")
		}

		// For OPTIONS requests, also set the Allow header
		if info.OriginalRequest.Method == "OPTIONS" {
			info.AddHeader("Allow", methodsStr)
		}

		AddRegistryRootHeader(info)

		info.SentStatus = true
		if info.StatusCode == 0 {
			info.StatusCode = http.StatusOK
		}

		// If the user never set one, don't let golang add one
		if info.GetResponseHeader("Content-Type") == "" {
			info.OriginalResponse.Header()["Content-Type"] = nil
		}

		info.OriginalResponse.WriteHeader(info.StatusCode)
	}
	return info.OriginalResponse.Write(b)
}

func (info *RequestInfo) DelHeader(name string) {
	info.OriginalResponse.Header().Del(name)
}

var stacks = map[string]string{}

func (info *RequestInfo) SetHeader(name, value string) {
	// Make sure we don't add the same header more than once, that's a sign
	// we're doing something weong.
	// At some point we may need to add a new func (AddHeader) to append a
	// value to the end of the current one (if there)
	PanicIf(info.OriginalResponse.Header().Get(name) != "",
		"%s\nPrev:\n%s\n---", name, stacks[name])
	// Uncomment when we need to debug the PanicIf
	// stacks[name] = GetStackAsString()

	info.OriginalResponse.Header()[name] = []string{value}
}

func (info *RequestInfo) AddHeader(name, value string) {
	info.OriginalResponse.Header().Add(name, value)
}

func (info *RequestInfo) GetResponseHeader(name string) string {
	return info.OriginalResponse.Header().Get(name)
}

func (info *RequestInfo) GetResponseHeaderValues(name string) []string {
	return info.OriginalResponse.Header()[name]
}

func (info *RequestInfo) Done() {
	// If we haven't written anything, this will force the HTTP status code
	// to be written and not default to 200
	info.Write(nil)
}

func NewRequestInfo(uuid string, xrsConfig *Config, w http.ResponseWriter,
	r *http.Request) *RequestInfo {

	info := &RequestInfo{
		uuid:             uuid,
		XRSConfig:        xrsConfig,
		OriginalPath:     strings.Trim(r.URL.Path, " /"),
		OriginalRequest:  r,
		OriginalResponse: w,
		BaseURL:          "http://" + r.Host,
		extras:           map[string]any{},
	}

	if r.TLS != nil {
		info.BaseURL = "https" + info.BaseURL[4:]
	} else if tmp := r.Header.Get("Referer"); tmp != "" {
		if strings.HasPrefix(tmp, "https:") {
			info.BaseURL = "https" + info.BaseURL[4:]
		}
	} else if tmp := r.Header.Get("Forwarded"); tmp != "" {
		if strings.Contains(tmp, "https") {
			info.BaseURL = "https" + info.BaseURL[4:]
		}
	}

	return info
}

func ParseRequest(tx *Tx, w http.ResponseWriter, r *http.Request) (*RequestInfo, *XRError) {

	info := NewRequestInfo(tx.uuid, tx.Config, w, r)
	info.tx = tx
	tx.RequestInfo = info

	// See which registry to use and twiddle some stuff in info if needed
	xErr := info.ParseRegistryURL()
	if xErr != nil {
		return info, xErr
	}

	// Load the request's body
	var err error
	info.Body, err = io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		return info, NewXRError("parsing_data", "request JSON",
			"error_detail="+err.Error())
	}
	if len(info.Body) == 0 {
		info.Body = nil
	}

	if log.IsFuncVerbose() {
		defer func() {
			log.Printf("tx: %s Info:\n%s",
				info.uuid, ToJSON(info))
		}()
	}

	// Save HTTP Header as flags before anyone asks to check them.
	info.ParseFlags()

	xErr = info.ProcessCapabilitiesModelSource()
	if xErr != nil {
		return info, xErr
	}

	if tmp := r.Header.Get("xRegistry~User"); tmp != "" {
		info.tx.User = tmp
	}

	// Parse the incoming URL and setup more stuff in info, like Groups...
	xErr = info.ParseRequestURL()
	if xErr != nil {
		return info, xErr
	}

	// Save the original Parts and RootPath for operations that need them
	// (they may be modified during request processing)
	info.OriginalParts = make([]string, len(info.Parts))
	copy(info.OriginalParts, info.Parts)
	info.OriginalRootPath = info.RootPath

	// Get root of the overall server - before any xreg/xxx processing
	root := "http://" + r.Host
	if r.TLS != nil {
		root = "https" + info.BaseURL[4:]
	} else if tmp := r.Header.Get("Referer"); tmp != "" {
		if strings.HasPrefix(tmp, "https:") {
			root = "https" + info.BaseURL[4:]
		}
	} else if tmp := r.Header.Get("Forwarded"); tmp != "" {
		if strings.Contains(tmp, "https") {
			root = "https" + info.BaseURL[4:]
		}
	}
	info.OriginalBaseURL = root

	if log.IsFuncVerbose() {
		log.Printf("tx: %s Info: %s", info.uuid, ToJSON(info))
	}

	return info, nil
}

func (info *RequestInfo) ParseFilters() *XRError {
	seenExcludeAll := false

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

			if expr == "excludeall" {
				if len(info.Filters) > 0 || len(AndFilters) > 0 {
					return NewXRError("bad_filter",
						info.OriginalRequest.URL.RequestURI(),
						"value=excludeall",
						"error_detail=\"excludeall\" must not appear with "+
							"any other filter expressions")
				}
				seenExcludeAll = true
			} else if seenExcludeAll {
				return NewXRError("bad_filter",
					info.OriginalRequest.URL.RequestURI(),
					"value="+expr,
					"error_detail=\"excludeall\" must not appear with "+
						"any other filter expressions")
			}

			filterOp := FILTER_PRESENT

			path, value, found := strings.Cut(expr, "!=")
			if found {
				// Note that "xxx!=null" is the same as "xxx"
				if value != "null" {
					filterOp = FILTER_NOT_EQUAL
				}
			} else {
				path, value, found = strings.Cut(expr, "<>")
				if found {
					// "<>null" is the same as present (no operator), per spec
					if value != "null" {
						filterOp = FILTER_NOT_EQUAL
					}
				} else {
					path, value, found = strings.Cut(expr, "<=")
					if found {
						filterOp = FILTER_LESS_EQUAL
					} else {
						path, value, found = strings.Cut(expr, ">=")
						if found {
							filterOp = FILTER_GREATER_EQUAL
						} else {
							path, value, found = strings.Cut(expr, "<")
							if found {
								filterOp = FILTER_LESS
							} else {
								path, value, found = strings.Cut(expr, ">")
								if found {
									filterOp = FILTER_GREATER
								} else {
									path, value, found = strings.Cut(expr, "=")
									if found {
										if value == "null" {
											filterOp = FILTER_ABSENT
										} else {
											filterOp = FILTER_EQUAL
										}
									}
									// No operator means FILTER_PRESENT
								}
							}
						}
					}
				}
			}

			pp, err := PropPathFromUI(path)
			if err != nil {
				return NewXRError("bad_filter",
					info.OriginalRequest.URL.RequestURI(),
					"value="+path,
					"error_detail="+err.Error())
			}
			path = pp.DB()

			// Validate comparison operator constraints per spec
			if filterOp == FILTER_LESS || filterOp == FILTER_LESS_EQUAL ||
				filterOp == FILTER_GREATER || filterOp == FILTER_GREATER_EQUAL {
				if value == "null" {
					return NewXRError("bad_filter",
						info.OriginalRequest.URL.RequestURI(),
						"value="+expr,
						"error_detail=null is not allowed "+
							"with <, <=, >, >= operators")
				}
				if strings.ContainsRune(value, '*') {
					return NewXRError("bad_filter",
						info.OriginalRequest.URL.RequestURI(),
						"value="+expr,
						"error_detail=wildcards are not allowed "+
							"with <, <=, >, >= operators")
				}
			}

			/*
								if info.What != "Coll" && strings.Index(path, "/") < 0 {
								return NewXRError("bad_filter",
				                info.OriginalRequest.URL.RequestURI(),
								"value=" + path,
								"error_detail=" +
								fmt.Sprintf("a filter with just an attribute " +
								"name (%s) isn't allowed in this context",
								path)
								}
			*/

			// If the request wasn't at the root, then add in the
			// abstract path that was provided so 'path' has the full
			// path starting from the root
			if info.Abstract != "" {
				// Want: path = abs + "," + path in DB format
				absPP, _ := PropPathFromPath(info.Abstract)
				pp = absPP.Append(pp)
				path = pp.DB()
			}

			absPP, newPP := SplitProp(info.Registry, pp)

			filter := &FilterExpr{
				PP:       newPP,
				Path:     path,
				Value:    value,
				Operator: filterOp,

				Abstract: absPP.Abstract(),
				PropName: newPP.DB(),
			}

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

// pp == PP for full DB path of attribute we're looking for
// (group.resource.attrPath) e.g abstractPP + propNamePP
// Look at it and remove any Group or Resource type names
// that appear and remove them. If none match then we MUST be left with just
// the attribute path (PP) within the entity we're looking for.
func SplitProp(reg *Registry, pp *PropPath) (*PropPath, *PropPath) {
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

	return abs, pp
}

// This will extract the "xregs/xxx" part of the URL if there and choose the
// appropriate Registry to use. It'll update info's BaseURL based on xregs
// This will populate some initial stuff in the "info" struct too, like
// Registry.
func (info *RequestInfo) ParseRegistryURL() *XRError {
	path := strings.Trim(info.OriginalPath, "/")

	defRegSegment := info.XRSConfig.GetAsString("path.defaultreg")
	regCollectionSegment := info.XRSConfig.GetAsString("path.regcollection")

	// e.g. localhost:8080/xreg/...
	// e.g. localhost:8080/xregs/XXX/...
	parts := strings.Split(path, "/")
	if parts[0] == regCollectionSegment {
		// have /xregs
		if len(parts) == 1 {
			return NewXRError("bad_request", info.BaseURL).
				SetDetailf("Missing registry name in URL: %s", info.BaseURL)
		}
		// have /xregs/...
		info.BaseURL += "/" + parts[0] + "/" + parts[1]
		info.OriginalPath = strings.Join(parts[2:], "/")

		reg, xErr := FindRegistry(info.tx, info.XRSConfig, parts[1], FOR_READ)
		if xErr != nil {
			return NewXRError("server_error",
				info.OriginalRequest.URL.RequestURI()).
				SetDetail(xErr.GetTitle())
		}
		if reg == nil {
			return NewXRError("not_found", info.BaseURL).
				SetDetailf("Can't find registry %q.", info.BaseURL)
		}
		info.Registry = reg
	} else if parts[0] == defRegSegment {
		// have /xreg
		info.BaseURL += "/" + parts[0]
		info.OriginalPath = strings.Join(parts[1:], "/")

		info.Registry = GetDefaultReg(info.tx)
	} else {
		info.Registry = GetDefaultReg(info.tx)
	}

	info.tx.Registry = info.Registry

	return nil
}

func (info *RequestInfo) ParseFlags() {
	// Save HTTP Header as flags before anyone asks to check them.
	// Notice boolean flags end up with [] as a value.
	info.Flags = map[string][]string{}
	params := info.OriginalRequest.URL.Query()
	for _, flag := range SupportedFlags {
		val, ok := params[flag]
		if ok {
			info.Flags[flag] = val
		}
	}
}

func (info *RequestInfo) ParseRequestURL() *XRError {
	if log.IsFuncVerbose() {
		log.Printf("tx: %s ParseRequestURL:\n%s", info.uuid, ToJSON(info))
		log.Printf("tx: %s Req: %#v", info.uuid, info.OriginalRequest.URL)
	}

	if xErr := info.ParseRequestPath(); xErr != nil {
		return xErr
	}

	// Some of these have to come after we parse the path so that the
	// group/resource info is setup - for verification

	// Let's do some query parameter stuff.

	if info.HasFlag("ignore") {
		info.Ignores = map[string]bool{}
		for _, value := range info.GetFlagValues("ignore") {
			if value == "" {
				info.Ignores["*"] = true
			} else {
				for _, val := range strings.Split(value, ",") {
					val = strings.TrimSpace(val)
					if val == "" {
						continue
						/*
							return NewXRError("bad_ignore", "/"+info.OriginalPath,
								"value="+value,
								"error_detail="+
									fmt.Sprintf("misplaced comma(,)"))
						*/
					}
					if val != "*" && !info.Registry.Capabilities.IgnoresEnabled(val) {
						return NewXRError("bad_ignore",
							info.OriginalRequest.URL.RequestURI(),
							"value="+val,
							"error_detail="+
								fmt.Sprintf("value not supported; allowed "+
									"values: %s",
									strings.Join(info.Registry.Capabilities.Ignores,
										",")))
					}
					info.Ignores[val] = true
				}
			}
		}
	}

	if info.HasFlag("inline") {
		for _, value := range info.GetFlagValues("inline") {
			if value == "" || value == "*" {
				if xErr := info.AddInline("*"); xErr != nil {
					return xErr
				}
				continue
			}
			for _, p := range strings.Split(value, ",") {
				if p == "" {
					continue
				}
				// if we're not at the root then we need to twiddle
				// the inline path to add the HTTP Path as a prefix
				if info.Abstract != "" {
					// want: p = info.Abstract + "." + p  in UI format
					absPP, err := PropPathFromPath(info.Abstract)
					if err != nil {
						return NewXRError("bad_inline",
							info.OriginalRequest.URL.RequestURI(),
							"value="+info.Abstract,
							"error_detail="+err.Error())
					}
					pPP, err := PropPathFromUI(p)
					if err != nil {
						return NewXRError("bad_inline",
							info.OriginalRequest.URL.RequestURI(),
							"value="+p,
							"error_detail="+err.Error())
					}
					p = absPP.Append(pPP).UI()
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
			return NewXRError("sort_noncollection",
				info.OriginalRequest.URL.RequestURI())
		}

		sortStr := info.GetFlag("sort")
		name, ascDesc, _ := strings.Cut(sortStr, "=")
		if name == "" {
			return NewXRError("bad_sort",
				info.OriginalRequest.URL.RequestURI(),
				"value="+sortStr,
				"error_detail=missing \"sort\" attribute name")
		}
		if ascDesc != "" && ascDesc != "asc" && ascDesc != "desc" {
			return NewXRError("bad_sort",
				info.OriginalRequest.URL.RequestURI(),
				"value="+sortStr,
				"error_detail="+
					fmt.Sprintf("invalid \"sort\" order %q", ascDesc))
		}
		// info.SortKey = name
		pp, err := PropPathFromUI(name)
		if err != nil {
			return NewXRError("bad_sort",
				info.OriginalRequest.URL.RequestURI(),
				"value="+sortStr,
				"error_detail="+
					fmt.Sprintf("bad attribute name(%s): %s",
						name, err.Error()))
		}
		info.SortKey = pp.DB()
		if ascDesc == "desc" {
			info.SortKey = "-" + info.SortKey
		}
	}

	// Pagination ("limit"/"offset") is only meaningful when the
	// "pagination" capability is turned on. When it's off, silently
	// ignore these query params rather than erroring - they're just
	// treated as if they weren't specified at all.
	if info.Registry.Capabilities.PaginationEnabled() &&
		(info.HasFlag("limit") || info.HasFlag("offset")) {

		if info.What != "Coll" {
			return NewXRError("bad_request",
				info.OriginalRequest.URL.RequestURI(),
				"error_detail=Can't paginate a non-collection result set")
		}

		if info.HasFlag("limit") {
			limitStr := info.GetFlag("limit")
			limit, err := strconv.ParseUint(limitStr, 10, 64)
			if err != nil || limit == 0 {
				return NewXRError("bad_request",
					info.OriginalRequest.URL.RequestURI(),
					"error_detail=\"limit\" value ("+limitStr+
						") must be an unsigned integer > 0")
			}
			info.Limit = limit
		}

		if info.HasFlag("offset") {
			offsetStr := info.GetFlag("offset")
			offset, err := strconv.ParseUint(offsetStr, 10, 64)
			if err != nil {
				return NewXRError("bad_request",
					info.OriginalRequest.URL.RequestURI(),
					"error_detail=\"offset\" value ("+offsetStr+
						") must be an unsigned integer")
			}
			if info.Limit == 0 {
				return NewXRError("bad_request",
					info.OriginalRequest.URL.RequestURI(),
					"error_detail=\"offset\" can't be used without \"limit\"")
			}
			info.Offset = offset
		}
	}

	if sv := info.GetFlag("specversion"); sv != "" {
		if !info.Registry.Capabilities.SpecVersionEnabled(sv) {
			return NewXRError("unsupported_specversion",
				"/"+info.OriginalPath,
				"specversion="+sv,
				"list="+
					strings.Join(info.Registry.Capabilities.SpecVersions, ","))
		}
	}

	if info.HasFlag("setdefaultversionid") {
		if def := info.GetFlag("setdefaultversionid"); def == "" {
			return NewXRError("bad_defaultversionid",
				info.OriginalRequest.URL.RequestURI(),
				"error_detail=value must not be empty",
				"value=\"\"")
		}
	}

	return info.ParseFilters()
}

func (info *RequestInfo) ParseRequestPath() *XRError {
	// Now process the URL path
	log.FuncPrintf("tx: %s ParseRequestPath: %q", info.uuid, info.OriginalPath)

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

	rModel := gModel.Resources[info.Parts[2]]
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
			SetDetailf("Expected \"versions\" or \"meta\", got: %s.",
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

// GetAllowedMethods returns the list of HTTP methods allowed for the current
// request path based on capabilities
func (info *RequestInfo) GetAllowedMethods() []string {
	methods := []string{}

	// Use original Parts/RootPath (they may have been modified during processing)
	parts := info.OriginalParts
	rootPath := info.OriginalRootPath
	numParts := len(parts)

	// Check special endpoints
	if rootPath == "capabilities" {
		if info.IsAvailable("capabilities") {
			methods = append(methods, "GET")
			if info.IsAvailableMutable("capabilities") {
				methods = append(methods, "PUT", "PATCH")
			}
		}
	} else if rootPath == "capabilitiesoffered" {
		if info.IsAvailable("capabilitiesoffered") {
			methods = append(methods, "GET")
		}
	} else if rootPath == "export" {
		if info.IsAvailable("export") {
			methods = append(methods, "GET")
		}
	} else if rootPath == "model" {
		if info.IsAvailable("model") {
			methods = append(methods, "GET")
		}
	} else if rootPath == "modelsource" {
		if info.IsAvailable("modelsource") {
			methods = append(methods, "GET")
			if info.IsAvailableMutable("modelsource") {
				methods = append(methods, "PUT")
			}
		}
	} else if rootPath == ".xregistry" {
		if info.IsAvailable(".xregistry") {
			methods = append(methods, "GET")
		}
	} else if info.IsAvailable("entities") {
		// Standard entity endpoints
		isMutable := info.IsAvailableMutable("entities")

		// GET is always supported for entities
		methods = append(methods, "GET")

		if isMutable {
			// Determine which write methods are supported based on path
			if numParts == 0 {
				// / - Registry root
				methods = append(methods, "PUT", "PATCH", "POST")
			} else if numParts == 1 {
				// /GROUPS - Collection
				methods = append(methods, "POST", "PATCH", "DELETE")
			} else if numParts == 2 {
				// /GROUPS/gID - Group entity
				methods = append(methods, "PUT", "PATCH", "POST", "DELETE")
			} else if numParts == 3 {
				// /GROUPS/gID/RESOURCES - Resource collection
				methods = append(methods, "POST", "PATCH", "DELETE")
			} else if numParts == 4 {
				// /GROUPS/gID/RESOURCES/rID - Resource entity
				methods = append(methods, "PUT", "PATCH", "POST", "DELETE")
			} else if numParts == 5 {
				if parts[4] == "meta" {
					// /GROUPS/gID/RESOURCES/rID/meta
					methods = append(methods, "PUT", "PATCH")
				} else if parts[4] == "versions" {
					// /GROUPS/gID/RESOURCES/rID/versions
					methods = append(methods, "POST", "PATCH", "DELETE")
				}
			} else if numParts == 6 {
				// /GROUPS/gID/RESOURCES/rID/versions/vID
				methods = append(methods, "PUT", "PATCH", "DELETE")
			}
		}
	}

	// Always include OPTIONS
	methods = append(methods, "OPTIONS")

	// Sort alphabetically for consistent output
	sort.Strings(methods)

	return methods
}

// This is called prior to partsing any of the URL bits (path,query params)
// because we may need to change what features are available based on how
// the capabilities or model is changed. This is only true when we're doing
// a write to the root of the Registry and it includes the "capabilities"
// or "modelsource" attributes.
func (info *RequestInfo) ProcessCapabilitiesModelSource() *XRError {
	// Only looking for operations that deal with the Registry entity itself
	if info.OriginalPath != "" {
		return nil
	}

	// Gotta be a 'write' operation (but NOT POST since that's just for
	// updating child collections, not xreg-level attributes)
	method := info.OriginalRequest.Method
	if method != "PUT" && method != "PATCH" {
		return nil
	}

	// Body must include at least "{}", if not just leave
	if len(info.Body) < 2 {
		return nil
	}

	obj := (map[string]any)(nil)
	changed := false

	err := Unmarshal(info.Body, &obj)
	if err != nil {
		return NewXRError("parsing_data", "request JSON",
			"error_detail="+err.Error())
	}

	// Grab the ?ignore query parameter from THIS request to know if we
	// should ignore the caps/modelSrc attributes or not.
	ignores := info.GetFlagValues("ignore")
	ignores = strings.Split(strings.Join(ignores, ","), ",")

	// Note this will always be the capabilities before any possible
	// updates to the capabilities (pre tx)
	cap := info.Registry.Capabilities

	if newCap, ok := obj["capabilities"]; ok {
		delete(obj, "capabilities")
		changed = true

		if !cap.FlagEnabled("ignore") ||
			!cap.IgnoresEnabled("capabilities") ||
			!ArrayContains(ignores, "capabilities") {

			// Handle PATCH vs PUT semantics for capabilities
			// Per spec (core/http.md lines 603-608):
			// - PATCH: only update specified top-level capabilities
			// - PUT: complete replacement of all capabilities
			var capToUpdate any
			if method == "PATCH" {
				// For PATCH: merge with current capabilities (top-level only)
				tmp := map[string]any{}
				tmpJSON, _ := json.Marshal(info.Registry.Capabilities)
				Must(Unmarshal(tmpJSON, &tmp))

				// Override with new values
				if newCapMap, ok := newCap.(map[string]any); ok {
					for k, v := range newCapMap {
						tmp[k] = v
					}
					capToUpdate = tmp
				} else {
					// If newCap is nil, use as-is (will reset to defaults)
					capToUpdate = newCap
				}
			} else {
				// For PUT: use as-is (complete replacement)
				capToUpdate = newCap
			}

			// Parse and validate the capabilities
			var newCapabilities *Capabilities
			var xErr *XRError

			if !IsNil(capToUpdate) {
				valStr := ToJSON(capToUpdate)
				newCapabilities, xErr = ParseCapabilities([]byte(valStr))
				if xErr != nil {
					return xErr
				}
			} else {
				// NULL capabilities - reset to defaults per spec
				newCapabilities = DefaultCapabilities.Clone()
			}

			if xErr = newCapabilities.Validate(); xErr != nil {
				return xErr
			}

			// Set capabilities directly (like HTTPPUTCapabilities does)
			xErr = info.Registry.SetSave("#capabilities",
				ToJSON(newCapabilities))
			if xErr != nil {
				return xErr
			}

			// Update the in-memory capabilities
			info.Registry.Capabilities = newCapabilities
		}
	}

	if _, ok := obj["modelsource"]; ok {
		delete(obj, "modelsource")
		changed = true

		if !cap.FlagEnabled("ignore") ||
			!cap.IgnoresEnabled("modelsource") ||
			!ArrayContains(ignores, "modelsource") {

			// We do this special logic so that "modelsource" isn't parsed
			// into golang stuff because when serialized back as as JSON
			// we'll lose the order of the map keys. Which I want to keep.
			// I want the modelsource to look exactly like how the user
			// provided it
			tmpReg := struct {
				ModelSource json.RawMessage
			}{}
			if err := json.Unmarshal(info.Body, &tmpReg); err != nil {
				return NewXRError("parsing_data", "request JSON",
					"error_detail="+err.Error())
			}

			val := tmpReg.ModelSource
			if ok {
				// null and {} both mean "reset model to empty" — treat
				// identically. json.RawMessage stores JSON null as the
				// 4-byte string "null" (not Go nil), so check for both.
				var rawJson []byte
				if IsNil(val) || string(val) == "null" {
					rawJson = []byte("{}")
				} else {
					var err error
					rawJson = []byte(val)
					rawJson, err = RemoveSchema(rawJson)
					if err != nil {
						return NewXRError("bad_request", "/",
							"error_detail="+err.Error())
					}
				}
				// A Model change can impact/invalidate assumptions
				// across the entire Registry tree (not just the
				// Registry entity's own attrs), so this must lock
				// the whole Registry row FOR_WRITE before applying
				// the new model - same physical lock Registry.Update()
				// already takes for "/" PUT/PATCH, just triggered here
				// for the model-changing reason instead.
				// DUG do we need to lock everything in Entities too?
				// Could a lower-level entity update get thru?
				info.Registry.Lock()

				xErr := info.Registry.Model.ApplyNewModelFromJSON(
					rawJson, false)
				if xErr != nil {
					return xErr
				}
				info.Registry.SetStuff("modelchanged", true)
			}

		}
	}

	// No need to call Registry.Update for capabilities - we handled it directly above
	// This avoids confusion about ADD_PATCH semantics at the Registry level

	if changed {
		info.Body = []byte(ToJSON(obj))
	}

	return nil
}
