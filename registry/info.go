package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

func NewRequestInfo(uuid string, xrsConfig *Config, w http.ResponseWriter,
	r *http.Request) *RequestInfo {

	info := &RequestInfo{
		UUID:             uuid,
		XRSConfig:        xrsConfig,
		OriginalPath:     strings.Trim(r.URL.Path, " /"),
		OriginalRequest:  r,
		OriginalResponse: w,
		BaseURL:          "http://" + r.Host,
		Extras:           map[string]any{},
	}

	// Probably an empty list, but if not, save it so that if we need to
	// reset() info we'll also reset the list of out-bound headers
	hdr := w.Header()
	if hdr != nil {
		copyHdr := map[string][]string{}
		for k, v := range hdr {
			copyHdr[k] = append([]string{}, v...)
		}

		info.Extras["savedHeaders"] = copyHdr
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

// Clear everything except the stuff that should never change on a retry
func (info *RequestInfo) Reset() {
	if info.Tx != nil {
		// Should never happen, but just incase
		info.Tx.Rollback()
		info.Tx = nil
	}

	// Clear and reset outbound headers to the original list
	clear(info.OriginalResponse.Header())
	savedHeaders := info.Extras["savedHeaders"]
	if !IsNil(savedHeaders) {
		for k, v := range savedHeaders.(map[string][]string) {
			info.OriginalResponse.Header()[k] = v
		}
	}

	newInfo := NewRequestInfo(info.UUID, info.XRSConfig,
		info.OriginalResponse, info.OriginalRequest)

	*info = *newInfo

}

func (info *RequestInfo) ParseRequest(tx *Tx) *XRError {

	// Incase it changed due to shortSelf
	info.OriginalPath = strings.Trim(info.OriginalRequest.URL.Path, " /")

	// See which registry to use and twiddle some stuff in info if needed
	xErr := info.ParseRegistryURL()
	if xErr != nil {
		return xErr
	}

	// Load the request's body
	var err error
	info.Body, err = io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		return NewXRError("parsing_data", "request JSON",
			"error_detail="+err.Error())
	}
	if len(info.Body) == 0 {
		info.Body = nil
	}

	if log.IsFuncVerbose() {
		defer func() {
			log.Printf("tx: %s Info:\n%s",
				info.UUID, ToJSON(info))
		}()
	}

	// Save HTTP Header as flags before anyone asks to check them.
	info.ParseFlags()

	xErr = info.ProcessCapabilitiesModelSource()
	if xErr != nil {
		return xErr
	}

	if tmp := info.OriginalRequest.Header.Get("xRegistry~User"); tmp != "" {
		info.Tx.User = tmp
	}

	// Parse the incoming URL and setup more stuff in info, like Groups...
	xErr = info.ParseRequestURL()
	if xErr != nil {
		return xErr
	}

	// Save the original Parts and RootPath for operations that need them
	// (they may be modified during request processing)
	info.OriginalParts = make([]string, len(info.Parts))
	copy(info.OriginalParts, info.Parts)
	info.OriginalRootPath = info.RootPath

	// Get root of the overall server - before any xreg/xxx processing
	root := "http://" + info.OriginalRequest.Host
	if info.OriginalRequest.TLS != nil {
		root = "https" + info.BaseURL[4:]
	} else if tmp := info.OriginalRequest.Header.Get("Referer"); tmp != "" {
		if strings.HasPrefix(tmp, "https:") {
			root = "https" + info.BaseURL[4:]
		}
	} else if tmp := info.OriginalRequest.Header.Get("Forwarded"); tmp != "" {
		if strings.Contains(tmp, "https") {
			root = "https" + info.BaseURL[4:]
		}
	}
	info.OriginalBaseURL = root

	if log.IsFuncVerbose() {
		log.Printf("tx: %s Info: %s", info.UUID, ToJSON(info))
	}

	return nil
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

	var xErr *XRError

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

		reg, xErr := FindRegistryByUID(info.Tx, info.XRSConfig, parts[1],
			FOR_READ)
		if xErr != nil {
			return xErr
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

		info.Registry, xErr = GetDefaultReg(info.Tx)
	} else {
		info.Registry, xErr = GetDefaultReg(info.Tx)
	}

	if xErr != nil {
		return xErr
	}

	info.Tx.Registry = info.Registry

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
		log.Printf("tx: %s ParseRequestURL:\n%s", info.UUID, ToJSON(info))
		log.Printf("tx: %s Req: %#v", info.UUID, info.OriginalRequest.URL)
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
	log.FuncPrintf("tx: %s ParseRequestPath: %q", info.UUID, info.OriginalPath)

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
				if _, xErr := info.Registry.Lock(); xErr != nil {
					return xErr
				}

				xErr := info.Registry.SetModel(nil, string(rawJson), false)
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
