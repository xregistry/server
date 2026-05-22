package registry

import (
	"bytes"
	// "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	// "os"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

type Server struct {
	Port       int
	HTTPServer *http.Server
}

func NewServer(port int) *Server {
	server := &Server{
		Port: port,
		HTTPServer: &http.Server{
			Addr: fmt.Sprintf(":%d", port),
		},
	}
	server.HTTPServer.Handler = server
	return server
}

func (s *Server) Close() {
	s.HTTPServer.Close()
}

func (s *Server) Start() *Server {
	go s.Serve()
	/*
		for {
			_, err := http.Get(fmt.Sprintf("http://localhost:%d", s.Port))
			if err == nil || !strings.Contains(err.Error(), "refused") {
				break
			}
		}
	*/
	return s
}

func (s *Server) Serve() {
	log.VPrintf(1, "Listening on %d", s.Port)
	err := s.HTTPServer.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Printf("Serve: %s", err)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var info *RequestInfo
	var tx *Tx

	defer func() {
		// If we haven't written anything, this will force the HTTP status code
		// to be written and not default to 200
		if info != nil {
			info.HTTPWriter.Done()
		}
	}()

	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("Panic: %s", rec)
			// ShowStack() // Down via NewXRError below now

			// If info isn't defined yet
			if info == nil {
				info = NewRequestInfo(w, r)
			}

			xErr := NewXRError("server_error", "/"+info.OriginalPath).
				SetDetail("An internal error occurred, contact the admin.")
			HTTPWriteError(info, xErr)
		}

		// As of now we should never have more than one active Tx during
		// testing
		/*
			if (os.Getenv("TESTING") != "") && tx != nil {
				l := len(TXs)
				if (tx.tx == nil && l > 0) || (tx.tx != nil && l > 1) {
					log.Printf(">End of HTTP Request")
					log.Printf("len(TXs): %d", l)
					log.Printf("tx.tx: %p", tx.tx)
					DumpTXs()

					log.Printf("Info: %s", ToJSON(info))
					log.Printf("<Exit http req")

					panic("nested Txs")
				}
			}
		*/

		// Explicit Commit() is required, else we'll always rollback
		tx.Rollback()
	}()

	saveVerbose := log.GetVerbose()
	if tmp := r.URL.Query().Get("verbose"); tmp != "" {
		if v, err := strconv.Atoi(tmp); err == nil {
			log.SetVerbose(v)
		}
		defer log.SetVerbose(saveVerbose)
	}

	log.VPrintf(2, "%s %s", r.Method, r.URL)

	if r.URL.Path == "/proxy" {
		HTTPProxy(w, r)
		return
	}

	tx, xErr := NewTx()
	if xErr != nil {
		log.Printf("Error talking to the DB creating new Tx: %s",
			xErr.GetTitle())

		// Special one off - info isn't defined yet
		if info == nil {
			info = NewRequestInfo(w, r)
		}

		HTTPWriteError(info, xErr)

		return
	}

	info, xErr = ParseRequest(tx, w, r)
	// tx.RequestInfo = info

	if xErr != nil {
		HTTPWriteError(info, xErr)
		return
	}

	if r.URL.Query().Has("ui") { // Wrap in html page
		info.HTTPWriter = NewPageWriter(info)
	}

	if r.URL.Query().Has("html") || r.URL.Query().Has("noprops") { //HTMLify it
		info.HTTPWriter = NewBufferedWriter(info)
	}

	/*
		if sv := info.GetFlag("specversion"); sv != "" {
			if !info.Registry.Capabilities.SpecVersionEnabled(sv) {
				xErr = NewXRError("unsupported_specversion",
					"/"+info.OriginalPath,
					"specversion="+sv,
					"list="+
						strings.Join(info.Registry.Capabilities.SpecVersions, ","))
			}
		}
	*/

	if xErr == nil {
		// These should only return an error if they didn't already
		// send a response back to the client.
		switch r.Method {
		case "GET":
			xErr = HTTPGet(info)
		case "PUT":
			xErr = HTTPPutPost(info)
		case "POST":
			xErr = HTTPPutPost(info)
		case "PATCH":
			xErr = HTTPPutPost(info)
		case "DELETE":
			xErr = HTTPDelete(info)
		default:
			xErr = NewXRError("action_not_supported", "/"+info.OriginalPath,
				"action="+r.Method)
		}
	}

	Must(tx.Conditional(xErr))

	if xErr != nil {
		HTTPWriteError(info, xErr)
	}
}

type HTTPWriter interface {
	Write([]byte) (int, error)
	SetHeader(string, string)
	AddHeader(string, string)
	GetHeader(string) string
	GetHeaderValues(string) []string
	Done()
}

var _ HTTPWriter = &DefaultWriter{}
var _ HTTPWriter = &BufferedWriter{}
var _ HTTPWriter = &DiscardWriter{}
var _ HTTPWriter = &PageWriter{}

func DefaultHTTPWriter(info *RequestInfo) HTTPWriter {
	return &DefaultWriter{
		Info: info,
	}
}

type DefaultWriter struct {
	Info *RequestInfo
}

func (dw *DefaultWriter) Write(b []byte) (int, error) {
	if !dw.Info.SentStatus {
		dw.Info.SentStatus = true
		if dw.Info.StatusCode == 0 {
			dw.Info.StatusCode = http.StatusOK
		}
		dw.Info.OriginalResponse.WriteHeader(dw.Info.StatusCode)
	}
	return dw.Info.OriginalResponse.Write(b)
}

var stacks = map[string]string{}

func (dw *DefaultWriter) SetHeader(name, value string) {
	// Make sure we don't add the same header more than once, that's a sign
	// we're doing something weong.
	// At some point we may need to add a new func (AddHeader) to append a
	// value to the end of the current one (if there)
	PanicIf(dw.Info.OriginalResponse.Header().Get(name) != "", "%s\nPrev:\n%s\n---", name, stacks[name])
	// Uncomment when we need to debug the PanicIf
	// stacks[name] = GetStackAsString()

	dw.Info.OriginalResponse.Header()[name] = []string{value}
}

func (dw *DefaultWriter) AddHeader(name string, value string) {
	dw.Info.OriginalResponse.Header().Add(name, value)
}

func (dw *DefaultWriter) GetHeader(name string) string {
	return dw.Info.OriginalResponse.Header().Get(name)
}

func (dw *DefaultWriter) GetHeaderValues(name string) []string {
	return dw.Info.OriginalResponse.Header()[name]
}

func (dw *DefaultWriter) Done() {
	dw.Write(nil)
}

type BufferedWriter struct {
	Info      *RequestInfo
	OldWriter HTTPWriter
	Headers   *map[string][]string
	Buffer    *bytes.Buffer
}

func NewBufferedWriter(info *RequestInfo) *BufferedWriter {
	return &BufferedWriter{
		Info:      info,
		OldWriter: info.HTTPWriter,
		Headers:   &map[string][]string{},
		Buffer:    &bytes.Buffer{},
	}
}

func (bw *BufferedWriter) Write(b []byte) (int, error) {
	return bw.Buffer.Write(b)
}

func (bw *BufferedWriter) SetHeader(name, value string) {
	(*bw.Headers)[name] = []string{value}
}

func (bw *BufferedWriter) AddHeader(name, value string) {
	(*bw.Headers)[name] = append((*bw.Headers)[name], value)
}

func (bw *BufferedWriter) GetHeader(name string) string {
	vals := (*bw.Headers)[name]
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

func (bw *BufferedWriter) GetHeaderValues(name string) []string {
	return (*bw.Headers)[name]
}

func (bw *BufferedWriter) Done() {
	req := bw.Info.OriginalRequest
	if req.URL.Query().Has("html") {
		// Override content-type
		bw.SetHeader("Content-Type", "text/html")
	}

	for k, values := range *bw.Headers {
		for _, val := range values {
			bw.OldWriter.AddHeader(k, val)
		}
	}

	buf := bw.Buffer.Bytes()
	if req.URL.Query().Has("html") {
		bw.OldWriter.Write([]byte("<pre>\n"))
		buf = HTMLify(req, buf)
	}
	bw.OldWriter.Write(buf)
}

type DiscardWriter struct{}

func (dw *DiscardWriter) Write(b []byte) (int, error)          { return len(b), nil }
func (dw *DiscardWriter) SetHeader(name, value string)         {}
func (dw *DiscardWriter) AddHeader(name, value string)         {}
func (dw *DiscardWriter) GetHeader(name string) string         { return "" }
func (dw *DiscardWriter) GetHeaderValues(name string) []string { return nil }
func (dw *DiscardWriter) Done()                                {}

var DefaultDiscardWriter = &DiscardWriter{}

func HTTPGETCapabilities(info *RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	buf := []byte(nil)
	var err error

	cap := info.Registry.Capabilities
	capStr := info.Registry.GetAsString("#capabilities")
	if capStr != "" {
		var xErr *XRError
		cap, xErr = ParseCapabilities([]byte(capStr))
		Must(xErr)
	}

	buf, err = json.MarshalIndent(cap, "", "  ")
	if err != nil {
		return NewXRError("server_error", "/").
			SetDetailf("Error parsing capabilities: %s", err.Error())
	}

	info.SetHeader("Content-Type", "application/json")
	info.Write(buf)
	info.Write([]byte("\n"))
	return nil
}

func HTTPGETCapabilitiesOffered(info *RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	buf := []byte(nil)
	var err error

	offered := GetOffered()
	buf, err = json.MarshalIndent(offered, "", "  ")
	if err != nil {
		return NewXRError("server_error", "/capabilitiesoffered").
			SetDetailf("Error parsing capabilitiesoffered: %s", err.Error())
	}

	info.SetHeader("Content-Type", "application/json")
	info.Write(buf)
	info.Write([]byte("\n"))
	return nil
}

func HTTPGETModel(info *RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	format := info.GetFlag("schema")
	if format == "" {
		format = "xRegistry-json"
	}

	model := info.Registry.Model
	if model == nil {
		model = &Model{}
	}

	buf, xErr := model.SerializeForUser()
	if xErr != nil {
		return NewXRError("server_error", "/"+info.OriginalPath).
			SetDetail(xErr.GetTitle())
	}

	info.SetHeader("Content-Type", "application/json")
	info.Write(buf)
	info.Write([]byte("\n"))
	return nil
}

func HTTPGETModelSource(info *RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	model := info.Registry.Model
	if model == nil {
		model = &Model{}
	}
	modelSrc := model.Source
	if modelSrc == "" {
		modelSrc = "{}"
	}

	buf, err := PrettyPrintJSON([]byte(modelSrc), "", "  ")
	if err != nil {
		return NewXRError("server_error", "/modelsource").
			SetDetailf("Error parsing modelsource: %s", err.Error())
	}

	info.SetHeader("Content-Type", "application/json")
	info.Write(buf)
	info.Write([]byte("\n"))
	return nil
}

func HTTPGETContent(info *RequestInfo) *XRError {
	log.VPrintf(3, ">Enter: HTTPGetContent")
	defer log.VPrintf(3, "<Exit: HTTPGetContent")

	query := `
SELECT
  RegSID,Type,Plural,Singular,eSID,UID,PropName,PropValue,PropType,Path,Abstract
FROM FullTree WHERE RegSID=? AND `
	args := []any{info.Registry.DbSID}

	path := strings.Join(info.Parts, "/")

	// TODO consider excluding the META object from the query instead of
	// dropping it via the if-statement below in the versioncount logic
	if info.VersionUID == "" {
		query += `(Path=? OR Path LIKE ?)`
		args = append(args, path, path+"/%")
	} else {
		query += `Path=?`
		args = append(args, path)
	}
	query += " ORDER BY Path"

	log.VPrintf(3, "Query:\n%s", SubQuery(query, args))

	results := Query(info.tx, query, args...)
	defer results.Close()

	entity, xErr := readNextEntity(info.tx, results, FOR_READ)
	log.VPrintf(3, "Entity: %#v", entity)
	if entity == nil {
		if xErr != nil {
			log.Printf("Error loading entity: %s", xErr)
			return NewXRError("server_error", "/"+path).SetDetailf(
				"error loading entity: %s", xErr.GetTitle())
		} else {
			return NewXRError("not_found", "/"+path)
		}
	}

	var version *Entity
	versionsCount := 0
	if info.VersionUID == "" {
		// We're on a Resource, so go find the default Version and count
		// how many versions there are for the VersionsCount attribute
		group, err := info.Registry.FindGroup(info.GroupType, info.GroupUID, false, FOR_READ)
		if err != nil {
			return NewXRError("server_error",
				info.GetParts(2)).SetDetailf("Error finding Group: %s", err)
		}
		if group == nil {
			return NewXRError("not_found", info.GetParts(2))
		}

		resource, err := group.FindResource(info.ResourceType,
			info.ResourceUID, false, FOR_READ)
		if err != nil {
			return NewXRError("server_error", info.GetParts(4)).
				SetDetailf("Error finding Resource: %s", err)
		}
		if resource == nil {
			return NewXRError("not_found", info.GetParts(4))
		}
		meta := resource.MustFindMeta(false, FOR_READ)

		vID := meta.GetAsString("defaultversionid")
		for {
			v, err := readNextEntity(info.tx, results, FOR_READ)

			if v != nil && v.Type == ENTITY_META {
				// Skip the "meta" subobject, but first grab the
				// "defaultversionid if we don't already have it,
				// which should only be true in the xref case
				if vID == "" {
					vID = v.GetAsString("defaultversionid")
				}
				continue
			}

			if v == nil && version == nil {
				return NewXRError("server_error", resource.XID+"/versions/"+vID).
					SetDetailf("Error finding Version: %s", err)
			}
			if v == nil {
				break
			}
			versionsCount++
			if v.UID == vID {
				version = v
			}
		}
	} else {
		version = entity
	}

	log.VPrintf(3, "Version: %#v", version)

	headerIt := func(e *Entity, info *RequestInfo, key string, val any, attr *Attribute) *XRError {
		if key[0] == '#' {
			return nil
		}

		if attr.internals != nil && attr.internals.neverSerialize {
			return nil
		}

		if attr.Type == MAP && IsScalar(attr.Item.Type) {
			for name, value := range val.(map[string]any) {
				// Should only ever be one of each xRegistry header
				info.SetHeader("xRegistry-"+key+"."+name,
					fmt.Sprintf("%v", value))
			}
			return nil
		}

		if !IsScalar(attr.Type) {
			return nil
		}

		var headerName string
		if attr.internals != nil && attr.internals.httpHeader != "" {
			headerName = attr.internals.httpHeader
		} else {
			headerName = "xRegistry-" + key
		}

		str := fmt.Sprintf("%v", val)
		// As above, should only ever be one of these
		info.AddHeader(headerName, str)

		return nil
	}

	xErr = entity.SerializeProps(info, headerIt)
	if xErr != nil {
		panic(xErr)
	}

	if info.VersionUID == "" {
		info.SetHeader("xRegistry-versionscount",
			fmt.Sprintf("%d", versionsCount))
		info.SetHeader("xRegistry-versionsurl",
			info.BaseURL+"/"+entity.Path+"/versions")
	}
	info.SetHeader("Content-Location", info.BaseURL+"/"+version.Path)
	info.SetHeader("Content-Disposition", info.ResourceUID)

	url := ""
	singular := info.ResourceModel.Singular
	if url = entity.GetAsString(singular + "url"); url != "" {
		// Should already be serialzied as a header
		// info.SetHeader("xRegistry-"+singular+"url", url)

		if info.StatusCode == 0 {
			// If we set it during a PUT/POST, don't override the 201
			info.StatusCode = http.StatusSeeOther
			info.SetHeader("Location", url)
		}
		/*
			http.Redirect(info.OriginalResponse, "/"+info.OriginalPath, url,
				http.StatusSeeOther)
		*/
		return nil
	}

	url = entity.GetAsString(singular + "proxyurl")

	log.VPrintf(3, singular+"proxyurl: %s", url)
	if url != "" {
		// Just act as a proxy and copy the remote resource as our response
		resp, err := http.Get(url)
		if err != nil {
			return NewXRError("parsing_response", "/"+info.OriginalPath,
				"error_detail="+err.Error())
		}
		if resp.StatusCode/100 != 2 {
			info.StatusCode = resp.StatusCode
			// return fmt.Error f("Remote error")
			// Let the body of the response be our body, below
		}

		// Copy all HTTP headers
		for header, value := range resp.Header {
			info.AddHeader(header, strings.Join(value, ","))
		}

		// Now copy the body
		_, err = io.Copy(info, resp.Body)
		if err != nil {
			return NewXRError("parsing_response", "/"+info.OriginalPath,
				"error_detail="+err.Error())
		}
		return nil
	}

	buf := version.Get(singular)
	if buf == nil {
		// No data so just return
		/*
			if info.StatusCode == 0 {
				info.StatusCode = http.StatusNoContent
			}
		*/
		return nil
	}

	info.Write(buf.([]byte))

	return nil
}

func HTTPGet(info *RequestInfo) *XRError {
	log.VPrintf(3, ">Enter: HTTPGet(%s)", info.What)
	defer log.VPrintf(3, "<Exit: HTTPGet(%s)", info.What)

	info.Root = strings.Trim(info.Root, "/")

	if info.RootPath == "capabilities" {
		if !info.IsAvailable("capabilities") {
			return NewXRError("not_available", "/capabilities")
		}
		return HTTPGETCapabilities(info)
	}

	if info.RootPath == "capabilitiesoffered" {
		if !info.IsAvailable("capabilitiesoffered") {
			return NewXRError("not_available", "/capabilitiesoffered")
		}
		return HTTPGETCapabilitiesOffered(info)
	}

	if info.RootPath == "export" {
		if !info.IsAvailable("export") {
			return NewXRError("not_available", "/export")
		}
		return SerializeQuery(info, nil, "Registry", info.Filters)
	}

	if info.RootPath == "model" {
		if !info.IsAvailable("model") {
			return NewXRError("not_available", "/model")
		}
		return HTTPGETModel(info)
	}

	if info.RootPath == "modelsource" {
		if !info.IsAvailable("modelsource") {
			return NewXRError("not_available", "/modelsource")
		}
		return HTTPGETModelSource(info)
	}

	// 'metaInBody' tells us whether xReg metadata should be in the http
	// response body or not (meaning, the hasDoc doc)
	metaInBody := (info.ResourceModel == nil) ||
		(info.ResourceModel.GetHasDocument() == false || info.ShowDetails ||
			info.DoDocView() ||
			(len(info.Parts) == 5 && info.Parts[4] == "meta"))

	// Return the Resource's document
	if info.What == "Entity" && info.ResourceUID != "" && !metaInBody {
		return HTTPGETContent(info)
	}

	// Serialize the xReg metadata
	resPaths := map[string][]string{
		"": []string{strings.Join(info.Parts, "/")},
	}
	return SerializeQuery(info, resPaths, info.What, info.Filters)
}

func SerializeQuery(info *RequestInfo, resPaths map[string][]string,
	what string, filters [][]*FilterExpr) *XRError {

	// Make sure everything is ok before we send back the results
	info.tx.Validate(info)

	// resPaths is used to group the items we want to return. In most cases
	// the items will all be part of one group where that group name doesn't
	// need to be returned - e.g. POST /schemagroup where the response will
	// just be a map of gIDs.
	// However, there are cases where we want to return multiple groupings
	// and each group has a different grouping name. For example:
	// POST / +   { "schemagroups": { "sg1"...}, "messagegroups": { "mg1"...}}
	// In this case the "paths" within each group are all processed as a
	// single query but the normal jwWriter stuff won't show the parent
	// group (schemagroups) because to do so would mean to also so the
	// attributes at that level (meaning the Registry attrs in this case).
	// To avoid this we pass in resPaths which is a map of groupings for
	// each "paths" (IDs) we want to serialize.
	// Each key in the map becomes the grouping key/name.
	// When the key is "" then there shouldn't be any other keys and we
	// won't wrapper it at all.
	// So, "" is for things like:  POST .../rID/versions + map[vID]{version}
	// And "xxx" is for things like POST /  + map[GROUPS]map[gID]{group}
	if resPaths == nil {
		resPaths = map[string][]string{"": nil}
	}

	start := time.Now()

	defer func() {
		if log.GetVerbose() > 3 {
			diff := time.Now().Sub(start).Truncate(time.Millisecond)
			log.Printf("  Total Time: %s", diff)
		}
	}()

	if info.RootPath == "export" {
		if !info.FlagEnabled("inline") || len(info.Inlines) == 0 {
			// Clear all inlines for cases where they specified them
			// but "inline" flag is not enabled
			info.Inlines = nil

			info.AddInline("*")
			info.AddInline("capabilities")
			info.AddInline("modelsource")
		}
	}

	info.SetHeader("Content-Type", "application/json")
	var jw *JsonWriter
	hasData := false
	keys := SortedKeys(resPaths)
	for i, key := range keys {
		paths, ok := resPaths[key]
		PanicIf(!ok, "can't find %q", key)

		var xErr *XRError
		var results *Result

		// "!" is special - it means skip the query and just produce: {}
		if len(paths) != 1 || paths[0] != "!" {
			query, args, err := GenerateQuery(info.Registry, what, paths,
				filters, info.DoDocView(), info.SortKey)
			if err != nil {
				return err
			}
			results = Query(info.tx, query, args...)
			defer results.Close()

			if log.GetVerbose() > 3 {
				log.Printf("SerializeQuery: %s", SubQuery(query, args))
				diff := time.Now().Sub(start).Truncate(time.Millisecond)
				log.Printf("  Query: # results: %d (time: %s)",
					len(results.AllRows), diff)
			}
		}

		jw = NewJsonWriter(info, results)

		jw.NextEntity()

		// Collections will need to print the {}, so don't error for them
		if what != "Coll" {
			if jw.Entity == nil {
				// Special case, if the URL is ../rID/versions/vID?doc then
				// check to see if Resource has xref set, if so then the error
				// is 400, not 404
				if info.VersionUID != "" && info.DoDocView() {
					path := strings.Join(info.Parts[:len(info.Parts)-2], "/")
					path += "/meta"
					entity, err := RawEntityFromPath(info.tx,
						info.Registry.DbSID, path, false, FOR_READ)
					if err != nil {
						return err
					}

					// Assume that if we cant' find the Resource's meta object
					// then the Resource doesn't exist, so a 404 really is the
					// best response in those cases, so skip the 400
					if entity != nil && !IsNil(entity.Object["xref"]) {
						return NewXRError("cannot_doc_xref", "/"+info.OriginalPath)
					}
				}

				return NewXRError("not_found", info.GetParts(0))
			}
		}

		// Special case, if we're doing a collection, let's make sure we didn't
		// get an empty result due to it's parent not even existing - for
		// example the user used the wrong case (or even name) in the parent's
		// Path
		if what == "Coll" && jw.Entity == nil && len(info.Parts) > 2 {
			path := strings.Join(info.Parts[:len(info.Parts)-1], "/")
			entity, xErr := RawEntityFromPath(info.tx, info.Registry.DbSID,
				path, false, FOR_READ)
			if xErr != nil {
				return xErr
			}
			if IsNil(entity) {
				return NewXRError("not_found", "/"+path)
			}
		}

		// GROUPS/GID/RESOURCES/RID/versions
		// Another special case .../rID/versions?doc when rID has xref
		if jw.Entity == nil && info.DoDocView() && len(info.Parts) == 5 &&
			info.Parts[4] == "versions" {

			// Should be our case since "versions" can never be empty except
			// when xref is set. If this is not longer true then we'll need to
			// check this Resource's xref to see if it's set.
			// Can copy the RawEntityFromPath... stuff above
			return NewXRError("cannot_doc_xref", "/"+info.OriginalPath)
		}

		// Only do this if we're adding the extra grouping wrapper
		if key != "" {
			// Cause the jwWriter to indent since we're adding a wrapper
			jw.indent = "  "
			if i == 0 {
				jw.Print("{\n")
			}
			jw.Printf("  %q: ", key)
		}

		if what == "Coll" {
			_, xErr = jw.WriteCollection()
		} else {
			xErr = jw.WriteEntity()
		}

		if xErr != nil {
			return xErr
		}

		if jw.hasData {
			hasData = true
		}

		// Only do this if we're adding the extra grouping wrapper
		if key != "" {
			if i+1 < len(keys) {
				jw.Print(",\n")
			} else {
				jw.Print("\n}")
			}
		}
	}

	if hasData {
		// Add a tailing \n if there's any data, else skip it
		jw.Print("\n")
	}

	return nil
}

var specialAttrHeaders = map[string]*Attribute{}

func init() {
	// Load-up the attributes that have custom http header names
	for _, attr := range OrderedSpecProps {
		if attr.internals != nil && attr.internals.httpHeader != "" {
			specialAttrHeaders[strings.ToLower(attr.internals.httpHeader)] =
				attr
		}
	}
}

func HTTPPutPost(info *RequestInfo) *XRError {
	method := info.OriginalRequest.Method
	isNew := false
	paths := ([]string)(nil)
	what := "Entity"
	numParts := len(info.Parts)

	metaInBody := (info.ResourceModel == nil) ||
		(info.ResourceModel.GetHasDocument() == false || info.ShowDetails ||
			(numParts == 5 && info.Parts[4] == "meta"))

	log.VPrintf(3, "HTTPPutPost: %s %s", method, info.OriginalPath)

	info.Root = strings.Trim(info.Root, "/")

	// Capabilities has its own special func
	if info.RootPath == "capabilities" {
		if !info.IsAvailable("capabilities") || !info.IsAvailableMutable("capabilities") {
			return NewXRError("not_available", "/capabilities")
		}
		return HTTPPUTCapabilities(info)
	}

	if info.RootPath == "model" {
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action="+method).
			SetDetail("Use \"/modelsource\" instead of \"/model\".")
	}

	// The model has its own special func
	if info.RootPath == "modelsource" {
		if !info.IsAvailable("modelsource") || !info.IsAvailableMutable("modelsource") {
			return NewXRError("not_available", "/modelsource")
		}
		return HTTPPUTModelSource(info)
	}

	if !info.IsAvailableMutable("entities") {
		return NewXRError("not_available", "/"+info.OriginalPath)
	}

	// Check for some obvious high-level bad states up-front
	// //////////////////////////////////////////////////////
	if info.What == "Coll" && method == "PUT" {
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action="+method).SetDetail("PUT not allowed on collections.")
	}

	if numParts >= 5 && info.Parts[4] == "meta" && method == "POST" {
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action="+method).SetDetail("POST not allowed on a 'meta'.")
	}

	if numParts == 6 && method == "POST" {
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action="+method).
			SetDetail("POST not allowed on a version.")
	}

	if (numParts == 4 || numParts == 6) && !metaInBody && method == "PATCH" {
		return NewXRError("details_required", "/"+info.OriginalPath).
			SetDetail("PATCH is not allowed on Resource documents.")
	}

	// Ok, now start to deal with the incoming request
	//////////////////////////////////////////////////

	// Get the incoming Object either from the body or from xRegistry headers
	IncomingObj, xErr := ExtractIncomingObject(info, info.Body)
	if xErr != nil {
		return xErr
	}

	// Walk the PATH and process things
	///////////////////////////////////

	// URL: /
	// ////////////////////////////////////////////////////////////////
	if numParts == 0 {
		// PUT /     + body:Registry
		// PATCH /   + body:Registry
		// POST /    + body:map[GROUPS]map[id]Group

		if method == "PUT" || method == "PATCH" {
			addType := ADD_UPDATE
			if method == "PATCH" {
				addType = ADD_PATCH
			}
			xErr = info.Registry.Update(IncomingObj, addType)
			if xErr != nil {
				return xErr
			}

			// Return HTTP GET of Registry root
			resPaths := map[string][]string{"": []string{""}}
			return SerializeQuery(info, resPaths, "Registry", info.Filters)
		}

		// Must be POST /    + body:map[GROUPS]map[id]Group

		newGs, xErr := info.Registry.UpsertJustGroups(IncomingObj, ADD_UPDATE)
		if xErr != nil {
			return xErr
		}

		resPaths := map[string][]string{}
		// Special case - if req is {} then make response {}
		if len(newGs) == 0 {
			resPaths = map[string][]string{"": []string{"!"}}
		} else {
			for gType, groups := range newGs {
				resPaths[gType] = []string{}
				for _, g := range groups {
					resPaths[gType] = append(resPaths[gType], g.Path)
				}
				if len(resPaths[gType]) == 0 {
					// Force an empty collection to be returned
					resPaths[gType] = []string{"!"}
				}
			}
		}

		// Return HTTP GET of Groups created or updated
		return SerializeQuery(info, resPaths, "Coll", info.Filters)
	}

	// URL: /GROUPs[/gID]...
	// ////////////////////////////////////////////////////////////////
	group := (*Group)(nil)
	groupUID := info.GroupUID

	if numParts == 1 {
		// POST /GROUPs + body:map[id]Group

		objMap, xErr := IncomingObj2Map(IncomingObj, "Group")
		if xErr != nil {
			return xErr.SetSubject(info.GetParts(0))
		}

		addType := ADD_UPSERT
		if method == "PATCH" {
			addType = ADD_PATCH
		}

		for id, obj := range objMap {
			g, _, xErr := info.Registry.UpsertGroupWithObject(info.GroupType,
				id, obj, addType)
			if xErr != nil {
				return xErr
			}
			paths = append(paths, g.Path)
		}

		if len(paths) == 0 {
			paths = []string{"!"} // Force an empty collection to be returned
		}

		// Return HTTP GET of Groups created or updated
		resPaths := map[string][]string{"": paths}
		return SerializeQuery(info, resPaths, "Coll", info.Filters)
	}

	if numParts == 2 {
		// PUT    /GROUPs/gID  + body: {group}
		// PATCH  /GROUPs/gID  + body: {group}
		// POST   /GROUPs/gID  + body: map[rType]map[rID]{resource}

		if method == "PUT" || method == "PATCH" {
			addType := ADD_UPSERT
			if method == "PATCH" {
				addType = ADD_PATCH
			}

			group, isNew, xErr := info.Registry.UpsertGroupWithObject(info.GroupType,
				info.GroupUID, IncomingObj, addType)
			if xErr != nil {
				return xErr
			}

			if isNew { // 201, else let it default to 200
				info.SetHeader("Location", info.BaseURL+"/"+group.Path)
				info.StatusCode = http.StatusCreated
			}

			// Return HTTP GET of Group
			resPaths := map[string][]string{"": []string{group.Path}}
			return SerializeQuery(info, resPaths, "Entity", info.Filters)
		}

		// Must be POST /GROUPs/gID + body: map[rType]map[rID]{resource}
		group, xErr = info.Registry.FindGroup(info.GroupType, groupUID, false,
			FOR_WRITE)
		if xErr != nil {
			return xErr
		}

		if group == nil {
			group, _, xErr = info.Registry.UpsertGroup(info.GroupType, groupUID)
			if xErr != nil {
				return xErr
			}
		}

		newRs, xErr := group.UpsertJustResources(IncomingObj, ADD_UPDATE)
		if xErr != nil {
			return xErr
		}

		resPaths := map[string][]string{}

		// Special case - if req is {} then make response {}
		if len(IncomingObj) == 0 {
			resPaths = map[string][]string{"": []string{"!"}}
		} else {
			for rType, resources := range newRs {
				resPaths[rType] = []string{}
				for _, r := range resources {
					resPaths[rType] = append(resPaths[rType], r.Path)
				}
				if len(resPaths[rType]) == 0 {
					resPaths[rType] = []string{"!"}
				}
			}
		}

		// Return HTTP GET of Resources created or updated
		return SerializeQuery(info, resPaths, "Coll", info.Filters)
	}

	// Must be PUT/POST /GROUPs/gID/...

	// This will either find or create an empty Group as needed
	group, _, xErr = info.Registry.UpsertGroup(info.GroupType, groupUID)
	if xErr != nil {
		return xErr
	}

	// URL: /GROUPs/gID/RESOURCEs...
	// ////////////////////////////////////////////////////////////////

	// Some global vars
	resource := (*Resource)(nil)
	version := (*Version)(nil)
	resourceUID := info.ResourceUID
	versionUID := info.VersionUID
	setDefVerID := info.GetFlag("setdefaultversionid")

	// Do Resources and Versions at the same time
	// URL: /GROUPs/gID/RESOURCEs
	// URL: /GROUPs/gID/RESOURCEs/rID
	// URL: /GROUPs/gID/RESOURCEs/rID/versions[/vID]
	// ////////////////////////////////////////////////////////////////

	// If there isn't an explicit "return" then this assumes we're left with
	// a version and will return that back to the client

	if numParts == 3 {
		// POST GROUPs/gID/RESOURCEs + body:map[id]Resource

		objMap, xErr := IncomingObj2Map(IncomingObj, "Resource")
		if xErr != nil {
			return xErr.SetSubject(info.GetParts(0))
		}

		// For each Resource in the map, upsert it and add it's path to result
		addType := ADD_UPSERT
		if method == "PATCH" {
			addType = ADD_PATCH
		}

		for id, obj := range objMap {
			r, _, xErr := group.UpsertResource(&ResourceUpsert{
				RType:            info.ResourceType,
				Id:               id,
				VID:              "",
				Obj:              obj,
				AddType:          addType,
				ObjIsVer:         false,
				DefaultVersionID: "",
			})
			if xErr != nil {
				return xErr
			}
			paths = append(paths, r.Path)
		}

		if len(paths) == 0 {
			paths = []string{"!"} // Force an empty collection to be returned
		}

		// Return HTTP GET of Resources created or modified
		resPaths := map[string][]string{"": paths}
		return SerializeQuery(info, resPaths, "Coll", info.Filters)
	}

	if numParts > 3 {
		// GROUPs/gID/RESOURCEs/rID...

		resource, xErr = group.FindResource(info.ResourceType, resourceUID,
			false, FOR_READ)
		if xErr != nil {
			return xErr
		}
	}

	if numParts == 4 && (method == "PUT" || method == "PATCH") {
		// PUT GROUPs/gID/RESOURCEs/rID [$details]

		if resource != nil {
			// version, xErr = resource.GetDefault(FOR_WRITE)

			// ID needs to be the version's ID, not the Resources
			// IncomingObj["id"] = version.UID

			// Create a new Resource and it's first/only/default Version
			addType := ADD_UPSERT
			if method == "PATCH" || !metaInBody {
				addType = ADD_PATCH
			}
			resource, _, xErr = group.UpsertResource(&ResourceUpsert{
				RType:            info.ResourceType,
				Id:               resourceUID,
				VID:              "",
				Obj:              IncomingObj,
				AddType:          addType,
				ObjIsVer:         false,
				DefaultVersionID: setDefVerID,
			})
			if xErr != nil {
				return xErr
			}

			version, xErr = resource.GetDefault(FOR_WRITE)
		} else {
			// Upsert resource's default version

			addType := ADD_UPSERT
			if method == "PATCH" {
				addType = ADD_PATCH
			}
			resource, isNew, xErr = group.UpsertResource(&ResourceUpsert{
				RType:            info.ResourceType,
				Id:               resourceUID,
				VID:              "",
				Obj:              IncomingObj,
				AddType:          addType,
				ObjIsVer:         false,
				DefaultVersionID: setDefVerID,
			})
			if xErr != nil {
				return xErr
			}

			version, xErr = resource.GetDefault(FOR_WRITE)
		}
		if xErr != nil {
			return xErr
		}
	}

	if method == "POST" && numParts == 4 {
		// POST GROUPs/gID/RESOURCEs/rID[$details], body=obj or doc
		propsID := "" // versionid
		if v, ok := IncomingObj["versionid"]; ok {
			if reflect.ValueOf(v).Kind() == reflect.String {
				propsID = NotNilString(&v)
			}
		}

		if resource == nil {
			// Implicitly create the resource
			resource, isNew, xErr = group.UpsertResource(&ResourceUpsert{
				RType:            info.ResourceType,
				Id:               resourceUID,
				VID:              propsID,
				Obj:              IncomingObj,
				AddType:          ADD_ADD,
				ObjIsVer:         true,
				DefaultVersionID: setDefVerID,
			})
			if xErr != nil {
				return xErr
			}
			version, xErr = resource.GetDefault(FOR_WRITE)
		} else {
			version, isNew, xErr = resource.UpsertVersionWithObject(&VersionUpsert{
				Id:               propsID,
				Obj:              IncomingObj,
				AddType:          ADD_UPSERT,
				More:             false,
				DefaultVersionID: setDefVerID,
			})
		}
		if xErr != nil {
			return xErr
		}
		// Default to just returning the version
	}

	// GROUPs/gID/RESOURCEs/rID/meta
	if numParts > 4 && info.Parts[4] == "meta" {
		// PUT /GROUPs/gID/RESOURCEs/rID/meta
		addType := ADD_UPSERT
		if method == "PATCH" {
			addType = ADD_PATCH
		}

		if resource == nil {
			isNew = true

			// Implicitly create the resource
			resource, _, xErr = group.UpsertResource(&ResourceUpsert{
				// TODO check to see if "" should be propsID
				RType:            info.ResourceType,
				Id:               resourceUID,
				VID:              "",
				Obj:              map[string]any{},
				AddType:          ADD_ADD,
				ObjIsVer:         false,
				DefaultVersionID: setDefVerID,
			})
			if xErr != nil {
				return xErr
			}
		}

		// DUG do we still need this? I think so
		if setDefVerID != "" {
			IncomingObj["defaultversionid"] = setDefVerID
			IncomingObj["defaultversionsticky"] = true
		}

		// Technically, this will always "update" not "insert"
		meta, _, xErr := resource.UpsertMeta(&MetaUpsert{
			obj:           IncomingObj,
			addType:       addType,
			createVersion: true,
			more:          false,
		})

		if xErr != nil {
			return xErr
		}

		// Return HTTP GET of 'meta'
		if isNew { // 201, else let it default to 200
			info.SetHeader("Location", info.BaseURL+"/"+meta.Path)
			info.StatusCode = http.StatusCreated
		}

		resPaths := map[string][]string{"": []string{meta.Path}}
		return SerializeQuery(info, resPaths, "Entity", info.Filters)
	}

	// Just double-check
	if numParts > 4 {
		PanicIf(info.Parts[4] != "versions", "Not 'versions': %s", info.Parts[4])
	}

	// GROUPs/gID/RESOURCEs/rID/versions...

	if info.ShowDetails && numParts == 5 {
		// PATCH|POST GROUPs/gID/RESOURCEs/rID/versions$details - error
		// TODO add a test for this
		panic("should never get here - info should catch this")
		return NewXRError("bad_details", info.GetParts(5))
	}

	if numParts == 5 && (method == "POST" || method == "PATCH") {
		// POST GROUPs/gID/RESOURCEs/rID/versions, body=map[id]->Version
		// PATCH GROUPs/gID/RESOURCEs/rID/versions, body=map[id]->Version

		// Convert IncomingObj to a map of Objects
		objMap, xErr := IncomingObj2Map(IncomingObj, "Version")
		if xErr != nil {
			return xErr
		}

		thisVersion := (*Version)(nil)

		if resource == nil {
			// Implicitly create the resource
			if len(objMap) == 0 {
				return NewXRError("missing_versions", "/"+info.OriginalPath)
			}

			tmpVID := SortedKeys(IncomingObj)[0]
			tmpObj := map[string]any{
				// Just grab any vID from the collection to make sure we
				// don't create a version with an ID of "1" by default
				"versionid": tmpVID,
				"versions":  (map[string]any)(IncomingObj),
			}

			addType := ADD_UPSERT
			if method == "PATCH" {
				addType = ADD_PATCH
			}

			resource, _, xErr = group.UpsertResource(&ResourceUpsert{
				RType:            info.ResourceType,
				Id:               resourceUID,
				VID:              tmpVID, // setDefVerID, // was ""
				Obj:              tmpObj,
				AddType:          addType,
				ObjIsVer:         false,
				DefaultVersionID: setDefVerID,
			})

			if xErr != nil {
				return xErr
			}

			v, xErr := resource.GetDefault(FOR_WRITE)
			Must(xErr)
			thisVersion = v

			// Remove the newly created default version from objMap so we
			// won't process it again, but add it to the reuslts collection
			for id, _ := range objMap {
				paths = append(paths, strings.Join(
					[]string{info.GroupType, info.GroupUID, info.ResourceType,
						info.ResourceUID, "versions", id},
					"/"))
			}
		} else {
			meta := resource.MustFindMeta(false, FOR_WRITE)

			if meta.Get("readonly") == true {
				if resource.tx.RequestInfo.HasIgnore("readonly") {
					// ?ignore=readonly so just stop w/o error
					// Force an empty collection to be returned
					paths = []string{"!"}
					resPaths := map[string][]string{"": paths}
					return SerializeQuery(info, resPaths, "Coll", info.Filters)
				} else {
					return NewXRError("readonly", resource.XID)
				}
			}

			// Process the versions
			addType := ADD_UPSERT
			if method == "PATCH" {
				addType = ADD_PATCH
			}
			count := 0
			for id, obj := range objMap {
				count++

				defv := ""
				more := true
				if count == len(objMap) {
					more = false
					defv = setDefVerID
				}

				v, _, xErr := resource.UpsertVersionWithObject(&VersionUpsert{
					Id:               id,
					Obj:              obj,
					AddType:          addType,
					More:             more,
					DefaultVersionID: defv,
				})
				if xErr != nil {
					return xErr
				}

				paths = append(paths, v.Path)
			}
		}

		// DUG do we still need this given what we do above?
		xErr = ProcessSetDefaultVersionIDFlag(info, resource, thisVersion)
		if xErr != nil {
			return xErr
		}

		if len(paths) == 0 {
			paths = []string{"!"} // Force an empty collection to be returned
		}
		resPaths := map[string][]string{"": paths}
		return SerializeQuery(info, resPaths, "Coll", info.Filters)
	}

	if numParts == 6 {
		// PUT GROUPs/gID/RESOURCEs/rID/versions/vID [$details]
		if resource == nil {
			// Implicitly create the resource
			resource, xErr = group.AddResourceWithObject(info.ResourceType,
				resourceUID, versionUID, IncomingObj, true)
			if xErr != nil {
				return xErr
			}

			isNew = true
		}

		version, xErr = resource.FindVersion(versionUID, false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}

		if version == nil {
			// We have a Resource, so add a new Version based on IncomingObj
			version, isNew, xErr = resource.UpsertVersionWithObject(&VersionUpsert{
				Id:               versionUID,
				Obj:              IncomingObj,
				AddType:          ADD_UPSERT,
				More:             false,
				DefaultVersionID: setDefVerID,
			})
		} else if !isNew {
			addType := ADD_UPSERT
			if method == "PATCH" || !metaInBody {
				addType = ADD_PATCH
			}
			version, _, xErr = resource.UpsertVersionWithObject(&VersionUpsert{
				Id:               version.UID,
				Obj:              IncomingObj,
				AddType:          addType,
				More:             false,
				DefaultVersionID: setDefVerID,
			})
		}
		if xErr != nil {
			return xErr
		}
	}

	PanicIf(xErr != nil, "err should be nil")

	// Process any ?setdefaultversionid query parameter there might be
	xErr = ProcessSetDefaultVersionIDFlag(info, resource, version)
	if xErr != nil {
		return xErr
	}

	// Make sure everything is ok before we send back the results
	info.tx.Validate(info)

	originalLen := numParts

	// Need to setup info stuff in case we call HTTPGetContent
	info.Parts = []string{info.Parts[0], groupUID,
		info.Parts[2], resourceUID}
	info.What = "Entity"
	info.GroupUID = groupUID
	info.ResourceUID = resourceUID // needed for $details in URLs

	location := info.BaseURL + "/" + resource.Path
	if originalLen > 4 || (originalLen == 4 && method == "POST") {
		info.Parts = append(info.Parts, "versions", version.UID)
		info.VersionUID = version.UID
		location += "/versions/" + version.UID
	}

	if info.ShowDetails { // not 100% sure this the right way/spot
		location += "$details"
	}

	if isNew { // 201, else let it default to 200
		info.SetHeader("Location", location)
		info.StatusCode = http.StatusCreated
	}

	// Return the contents of the entity instead of the xReg metadata
	if !metaInBody {
		return HTTPGETContent(info)
	}

	// Return the xReg metadata of the entity processed
	if paths == nil {
		paths = []string{strings.Join(info.Parts, "/")}
	}

	resPaths := map[string][]string{"": paths}
	return SerializeQuery(info, resPaths, what, info.Filters)
}

func HTTPPUTCapabilities(info *RequestInfo) *XRError {
	if len(info.Parts) > 1 || !info.IsAvailable("capabilities") {
		return NewXRError("not_available", "/"+info.GetParts(0))
	}

	reqBody, err := RemoveSchema(info.Body)
	if err != nil {
		return NewXRError("parsing_data", info.GetParts(0),
			"error_detail="+err.Error())
	}

	cap := &Capabilities{}

	method := info.OriginalRequest.Method
	if method == "PUT" {
		// Fall thru
	} else if method == "PATCH" {
		// put current capabilities into a simple map
		tmp := map[string]any{}
		tmpJSON, _ := json.Marshal(info.Registry.Capabilities)
		Must(Unmarshal(tmpJSON, &tmp))

		// Now override wth anything new
		err := Unmarshal(reqBody, &tmp)
		if err != nil {
			return NewXRError("parsing_data", "/capabilities",
				"error_detail="+err.Error())
		}

		reqBody, _ = json.Marshal(tmp)
	} else {
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action="+info.OriginalRequest.Method)
	}

	cap, xErr := ParseCapabilities(reqBody)
	if xErr != nil {
		return xErr
	}

	xErr = cap.Validate()
	if xErr != nil {
		return xErr
	}

	if xErr = info.Registry.SetSave("#capabilities", ToJSON(cap)); xErr != nil {
		return xErr
	}

	// Now make sure all of the data in the Registry is ok. If not
	// we can't allow the capabilities to be updated
	if xErr = info.Registry.Model.Verify(); xErr != nil {
		return xErr
	}

	return HTTPGETCapabilities(info)
}

func HTTPPUTModelSource(info *RequestInfo) *XRError {
	if len(info.Parts) > 1 || !info.IsAvailable("modelsource") {
		return NewXRError("not_available", "/"+info.GetParts(0))
	}

	if info.OriginalRequest.Method != "PUT" {
		return NewXRError("action_not_supported", "/modelsource",
			"action="+info.OriginalRequest.Method)
	}

	xErr := info.Registry.Model.ApplyNewModelFromJSON(info.Body, true)
	if xErr != nil {
		return xErr
	}

	info.Registry.Touch()

	if xErr = info.Registry.ValidateAndSave(false); xErr != nil {
		return xErr
	}

	return HTTPGETModelSource(info)
}

// Process the ?setdefaultversionid query parameter
// "resource" is the resource we're processing
// "version" is the version that was processed
func ProcessSetDefaultVersionIDFlag(info *RequestInfo, resource *Resource, version *Version) *XRError {
	return nil
	vIDs := info.GetFlagValues("setdefaultversionid")
	if len(vIDs) == 0 {
		return nil
	}

	/*
		if info.ResourceModel.GetSetDefaultSticky() == false {
			return NewXRError("setdefaultversionid_not_allowed", resource.XID,
				"singular="+info.ResourceModel.Singular)
		}
	*/

	if len(info.Parts) >= 5 {
		vID := vIDs[0]

		/*
			if vID == "" {
				return NewXRError("bad_defaultversionid", resource.XID,
					"value="+`""`,
					"error_detail=value must not be empty")
			}
		*/

		// "null" and "request" have special meaning
		if vID == "null" {
			// Unstick the default version and go back to newest=default
			return resource.SetDefault(nil)
		}

		if vID == "request" {
			if version == nil {
				return NewXRError("defaultversionid_request", resource.XID)
			}
			// stick default version to current one we just processed
			return resource.SetDefault(version)
		}

		version, xErr := resource.FindVersion(vID, false, FOR_READ)
		if xErr != nil {
			return xErr
		}
		if version == nil {
			return NewXRError("unknown_id", resource.XID,
				"singular=version",
				"id="+vID)
		}
	}

	return resource.SetDefault(version)
}

func HTTPDelete(info *RequestInfo) *XRError {
	// DELETE /...
	if len(info.Parts) == 0 {
		// DELETE /
		return NewXRError("action_not_supported", "/", "action=DELETE")
	}

	var xErr *XRError
	var err error
	epochStr := info.GetFlag("epoch")
	epochInt := -1
	if epochStr != "" {
		epochInt, err = strconv.Atoi(epochStr)
		if err != nil || epochInt < 0 {
			return NewXRError("invalid_attribute", "/"+info.OriginalPath,
				"name=epoch",
				"error_detail="+
					fmt.Sprintf("value (%s) must be a uinteger", epochStr))
		}
	}

	if info.HasIgnore("epoch") {
		epochInt = -1
	}

	// DELETE /GROUPs...
	gm := info.Registry.Model.Groups[info.GroupType]
	if gm == nil {
		return NewXRError("not_found", info.GetParts(1))
	}

	if len(info.Parts) == 1 {
		// DELETE /GROUPs
		xErr = HTTPDeleteGroups(info)
		if xErr != nil {
			return xErr
		}
		info.tx.Validate(info)
		return nil
	}

	// DELETE /GROUPs/gID...
	group, xErr := info.Registry.FindGroup(info.GroupType, info.GroupUID, false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}
	if group == nil {
		return NewXRError("not_found", info.GetParts(2))
	}

	if len(info.Parts) == 2 {
		// DELETE /GROUPs/gID
		if epochInt >= 0 {
			if e := group.Get("epoch"); e != epochInt {
				return NewXRError("mismatched_epoch", group.XID,
					"bad_epoch="+epochStr,
					"epoch="+fmt.Sprintf("%d", e))
			}
		}
		if xErr = group.Delete(); xErr != nil {
			return xErr
		}

		info.tx.Validate(info)

		info.StatusCode = http.StatusNoContent
		return nil
	}

	// DELETE /GROUPs/gID/RESOURCEs...
	if rm := gm.FindResourceModel(info.ResourceType); rm == nil {
		return NewXRError("not_found", info.GetParts(3))
	}

	if len(info.Parts) == 3 {
		// DELETE /GROUPs/gID/RESOURCEs
		xErr = HTTPDeleteResources(info)
		if xErr != nil {
			return xErr
		}
		info.tx.Validate(info)
		return nil
	}

	// DELETE /GROUPs/gID/RESOURCEs/rID...
	resource, xErr := group.FindResource(info.ResourceType, info.ResourceUID,
		false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}
	if resource == nil {
		return NewXRError("not_found", info.GetParts(4))
	}

	meta := resource.MustFindMeta(false, FOR_WRITE)

	if len(info.Parts) == 4 {
		// DELETE /GROUPs/gID/RESOURCEs/rID
		if epochInt >= 0 {
			if e := meta.Get("epoch"); e != epochInt {
				return NewXRError("mismatched_epoch", meta.XID,
					"bad_epoch="+epochStr,
					"epoch="+fmt.Sprintf("%d", e))
			}
		}

		xErr = resource.Delete()
		if xErr != nil {
			return xErr
		}

		info.tx.Validate(info)

		info.StatusCode = http.StatusNoContent
		return nil
	}

	if len(info.Parts) == 5 && info.Parts[4] == "meta" {
		// DELETE /GROUPs/gID/RESOURCEs/rID/meta
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action=DELETE")
	}

	if len(info.Parts) == 5 {
		// DELETE /GROUPs/gID/RESOURCEs/rID/versions
		xErr = HTTPDeleteVersions(info)
		if xErr != nil {
			return xErr
		}

		info.tx.Validate(info)
		return nil
	}

	// DELETE /GROUPs/gID/RESOURCEs/rID/versions/vID...
	version, xErr := resource.FindVersion(info.VersionUID, false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}
	if version == nil {
		return NewXRError("not_found", info.GetParts(0))
	}

	if len(info.Parts) == 6 {
		// DELETE /GROUPs/gID/RESOURCEs/rID/versions/vID
		if epochInt >= 0 {
			if e := version.Get("epoch"); e != epochInt {
				return NewXRError("mismatched_epoch", version.XID,
					"bad_epoch="+epochStr,
					"epoch="+fmt.Sprintf("%d", e))
			}
		}
		nextDefault := info.GetFlag("setdefaultversionid")
		xErr = version.DeleteSetNextVersion(nextDefault)
		if xErr != nil {
			return xErr
		}

		info.tx.Validate(info)

		info.StatusCode = http.StatusNoContent
		return nil
	}

	return NewXRError("not_found", info.GetParts(0))
}

type EpochEntry map[string]any
type EpochEntryMap map[string]EpochEntry

func LoadEpochMap(info *RequestInfo) (EpochEntryMap, *XRError) {
	res := EpochEntryMap{}

	bodyStr := strings.TrimSpace(string(info.Body))

	if len(bodyStr) > 0 {
		err := Unmarshal([]byte(bodyStr), &res)
		if err != nil {
			return nil, NewXRError("parsing_data", info.GetParts(0),
				"error_detail="+err.Error())
		}
	} else {
		// EpochEntryMap == nil mean no list at all, not same as empty list
		return nil, nil
	}

	return res, nil
}

func HTTPDeleteGroups(info *RequestInfo) *XRError {
	list, xErr := LoadEpochMap(info)
	if xErr != nil {
		return xErr
	}

	// No list provided so get list of Groups so we can delete them all
	// TODO: optimize this to just delete it all in one shot
	if list == nil {
		list = EpochEntryMap{}
		results := Query(info.tx, `
			SELECT UID
			FROM Entities
			WHERE RegSID=? AND Abstract=?`,
			info.Registry.DbSID, info.GroupType)

		for row := results.NextRow(); row != nil; row = results.NextRow() {
			list[NotNilString(row[0])] = EpochEntry{}
		}
		defer results.Close()
	}

	// Delete each Group, checking epoch first if provided
	for id, entry := range list {
		group, xErr := info.Registry.FindGroup(info.GroupType, id, false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}
		if group == nil {
			// Silently ignore the 404
			continue
		}

		if tmp, ok := entry["epoch"]; ok && !info.HasIgnore("epoch") {
			tmpInt, err := AnyToUInt(tmp)
			if err != nil {
				return NewXRError("invalid_attribute", group.XID,
					"name=epoch",
					"error_detail=must be a uinteger")
			}
			if tmpInt != group.Get("epoch") {
				return NewXRError("mismatched_epoch", group.XID,
					"bad_epoch="+fmt.Sprintf("%v", tmp),
					"epoch="+fmt.Sprintf("%d", group.Get("epoch")))
			}
		}

		singular := group.Singular + "id"
		if tmp, ok := entry[singular]; ok && tmp != id {
			return NewXRError("mismatched_id", group.XID,
				"singular="+group.Singular,
				"invalid_id="+fmt.Sprintf("%v", tmp),
				"expected_id="+id)
		}

		xErr = group.Delete()
		if xErr != nil {
			return xErr
		}
	}

	info.StatusCode = http.StatusNoContent
	return nil
}

func HTTPDeleteResources(info *RequestInfo) *XRError {
	list, xErr := LoadEpochMap(info)
	if xErr != nil {
		return xErr
	}

	// No list provided so get list of Resources so we can delete them all
	// TODO: optimize this to just delete it all in one shot
	if list == nil {
		list = EpochEntryMap{}
		results := Query(info.tx, `
			SELECT UID
			FROM Entities
			WHERE RegSID=? AND Abstract=?`,
			info.Registry.DbSID,
			NewPPP(info.GroupType).P(info.ResourceType).Abstract())

		for row := results.NextRow(); row != nil; row = results.NextRow() {
			list[NotNilString(row[0])] = EpochEntry{}
		}
		defer results.Close()
	}

	group, xErr := info.Registry.FindGroup(info.GroupType, info.GroupUID, false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}

	// Delete each Resource, checking epoch first if provided
	for id, entry := range list {
		resource, xErr := group.FindResource(info.ResourceType, id, false,
			FOR_WRITE)
		if xErr != nil {
			return xErr
		}
		if resource == nil {
			// Silently ignore the 404
			continue
		}

		meta := resource.MustFindMeta(false, FOR_WRITE)

		singular := resource.Singular + "id"

		if metaJSON, ok := entry["meta"]; ok {
			metaMap, ok := metaJSON.(map[string]any)
			if !ok {
				return NewXRError("invalid_attribute", resource.XID,
					"name=meta",
					"error_detail="+
						fmt.Sprintf("meta needs to be an object, "+
							"not a \"%T\"", metaJSON))
			}

			if tmp, ok := metaMap[singular]; ok && tmp != id {
				return NewXRError("mismatched_id", resource.XID,
					"singular="+resource.Singular,
					"invalid_id="+fmt.Sprintf("%v", tmp),
					"expected_id="+id)
			}

			if tmp, ok := metaMap["epoch"]; ok && !info.HasIgnore("epoch") {
				tmpInt, err := AnyToUInt(tmp)
				if err != nil {
					return NewXRError("invalid_attribute", meta.XID,
						"name=epoch",
						"error_detail=must be a uinteger")
				}
				if tmpInt != meta.Get("epoch") {
					return NewXRError("mismatched_epoch", meta.XID,
						"bad_epoch="+fmt.Sprintf("%v", tmp),
						"epoch="+fmt.Sprintf("%d", meta.Get("epoch")))
				}
			}
		} else {
			if _, ok := entry["epoch"]; ok {
				return NewXRError("misplaced_epoch", resource.XID)
			}
		}

		if tmp, ok := entry[singular]; ok && tmp != id {
			return NewXRError("mismatched_id", resource.XID,
				"singular="+resource.Singular,
				"invalid_id="+fmt.Sprintf("%v", tmp),
				"expected_id="+id)
		}

		xErr = resource.Delete()
		if xErr != nil {
			return xErr
		}
	}

	info.StatusCode = http.StatusNoContent
	return nil
}

func HTTPDeleteVersions(info *RequestInfo) *XRError {
	nextDefault := info.GetFlag("setdefaultversionid")

	list, xErr := LoadEpochMap(info)
	if xErr != nil {
		return xErr
	}

	group, xErr := info.Registry.FindGroup(info.GroupType, info.GroupUID, false, FOR_READ)
	if xErr != nil {
		return xErr
	}

	resource, xErr := group.FindResource(info.ResourceType, info.ResourceUID,
		false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}

	// No list provided so get list of Versions so we can delete them all
	// TODO: optimize this to just delete it all in one shot
	if list == nil {
		list = EpochEntryMap{}
		vIDs, xErr := resource.GetVersionIDs()
		if xErr != nil {
			return xErr
		}
		for _, vID := range vIDs {
			list[vID] = EpochEntry{}
		}
	}

	// Before we delete each one, make sure the epoch value is ok.
	// We can't check and delete at the same time before deleting one
	// might update a future one's ancestor value which will also change
	// its epoch value - which will make the epoch check fail.
	// However, we can save the Versions for the next loop.
	vers := []*Version{}
	for id, entry := range list {
		version, xErr := resource.FindVersion(id, false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}
		if version == nil {
			// Silently ignore the 404
			continue
		}
		vers = append(vers, version)

		// check epoch
		if tmp, ok := entry["epoch"]; ok && !info.HasIgnore("epoch") {
			tmpInt, err := AnyToUInt(tmp)
			if err != nil {
				return NewXRError("invalid_attribute", version.XID,
					"name=epoch",
					"error_detail=value must be a uinteger")
			}
			if tmpInt != version.GetAsInt("epoch") {
				return NewXRError("mismatched_epoch", version.XID,
					"bad_epoch="+fmt.Sprintf("%v", tmp),
					"epoch="+fmt.Sprintf("%d", version.GetAsInt("epoch")))
			}
		}

		// For safety make sure the Resource's ID on the version (if there)
		// matches
		singular := version.Singular + "id"
		if tmp, ok := entry[singular]; ok && tmp != version.Get(singular) {
			return NewXRError("mismatched_id", version.XID,
				"singular=version",
				"invalid_id="+fmt.Sprintf("%v", tmp),
				"expected_id="+fmt.Sprintf("%v", version.Get(singular)))
		}
	}

	// Now we can actually delete each one
	for _, version := range vers {
		xErr = version.DeleteSetNextVersion(nextDefault)
		if xErr != nil {
			return xErr
		}
	}

	if nextDefault != "" {
		version, xErr := resource.FindVersion(nextDefault, false, FOR_READ)
		if xErr != nil {
			return xErr
		}
		xErr = resource.SetDefault(version)
		if xErr != nil {
			return xErr
		}
	}

	info.StatusCode = http.StatusNoContent
	return nil
}

func ExtractIncomingObject(info *RequestInfo, body []byte) (Object, *XRError) {
	IncomingObj := map[string]any{}

	if len(body) == 0 {
		body = nil
	}

	resSingular := ""
	hasDoc := false
	if info.ResourceModel != nil {
		resSingular = info.ResourceModel.Singular
		hasDoc = info.ResourceModel.GetHasDocument()
	}

	// Start with the assumption that we need a body, until proven otherwise
	requireBody := true
	if hasDoc && !info.ShowDetails {
		// .../rID || .../vID
		if len(info.Parts) == 4 || len(info.Parts) == 6 {
			requireBody = false
		}
	}

	if requireBody && len(body) == 0 {
		return nil, NewXRError("missing_body", "/"+info.OriginalPath)
	}

	// len=5 is a special case where we know .../versions always has the
	// metadata in the body so $details isn't needed, and in fact an error

	// GROUPS/GID/RESOURCES/RID/meta|versions/vID
	metaInBody := (info.ShowDetails ||
		len(info.Parts) == 3 ||
		len(info.Parts) == 5 ||
		(info.ResourceModel != nil && hasDoc == false))

	if len(info.Parts) < 3 || metaInBody {
		for k, _ := range info.OriginalRequest.Header {
			k := strings.ToLower(k)
			if strings.HasPrefix(k, "xregistry-") {
				if hasDoc == false {
					return nil, NewXRError("extra_xregistry_header",
						"/"+info.OriginalPath,
						"name="+k,
						"error_detail="+
							fmt.Sprintf("including \"xRegistry\" headers "+
								"for a Resource that has the model "+
								"\"hasdocument\" value of \"false\" is invalid"))
				}
				return nil, NewXRError("extra_xregistry_header",
					"/"+info.OriginalPath,
					"name="+k,
					"error_detail="+
						fmt.Sprintf("including \"xRegistry\" HTTP headers "+
							"when \"$details\" is used is not allowed"))
			}
		}

		err := Unmarshal(body, &IncomingObj)
		if err != nil {
			return nil, NewXRError("parsing_data", info.GetParts(0),
				"error_detail="+err.Error())
		}

		// "modelsource" is sooo special! Don't parse it into a golang type
		// keep it as []byte so that we preserve the order of the map keys
		if len(info.Parts) == 0 && !IsNil(IncomingObj["modelsource"]) {
			tmpReg := struct {
				ModelSource json.RawMessage
			}{}
			if err := json.Unmarshal(body, &tmpReg); err != nil {
				return nil, NewXRError("parsing_data", info.GetParts(0),
					"error_detail="+err.Error())
			}
			IncomingObj["modelsource"] = tmpReg.ModelSource
		}

		// Delete any json schema tag in there
		delete(IncomingObj, "$schema")
	}

	// xReg metadata are in headers, so move them into IncomingObj. We'll
	// copy over the existing properties later once we know what entity
	// we're dealing with
	if len(info.Parts) > 2 && !metaInBody {
		IncomingObj[resSingular] = body // save new body

		seenMaps := map[string]bool{}
		seenMetaMaps := map[string]bool{}

		for name, attr := range specialAttrHeaders {
			// TODO we may need some kind of "delete if missing" flag on
			// each HttpHeader attribute since some may want to have an
			// explicit 'null' to be erased instead of just missing (eg patch)
			vals, ok := info.OriginalRequest.Header[http.CanonicalHeaderKey(name)]
			if ok {
				val := vals[0]
				if val == "null" {
					IncomingObj[attr.Name] = nil
				} else {
					IncomingObj[attr.Name] = val
				}
			}
		}

		meta := map[string]any(nil)

		for key, value := range info.OriginalRequest.Header {
			key := strings.ToLower(key)

			if !strings.HasPrefix(key, "xregistry-") {
				continue
			}

			key = strings.TrimSpace(key[10:]) // remove xRegistry-
			if key == "" {
				return nil, NewXRError("header_error", "/"+info.OriginalPath,
					"name=xRegistry-",
					`error_detail=missing an attribute name after the "-"`)
			}

			if key == resSingular || key == resSingular+"base64" {
				return nil, NewXRError("extra_xregistry_header",
					"/"+info.OriginalPath,
					"name=xRegistry-"+key,
					"error_detail="+
						fmt.Sprintf("'xRegistry-%s' isn't allowed as "+
							"an HTTP header", key))
			}

			if key == resSingular+"url" || key == resSingular+"proxyurl" {
				if len(body) != 0 {
					return nil, NewXRError("extra_xregistry_header",
						"/"+info.OriginalPath,
						"name=xRegistry-"+key,
						"error_detail=header isn't allowed if there's a body")
				}
			}

			val := any(value[0])
			if val == "null" {
				val = nil
			}

			// If there are .'s then it's a non-scalar, convert it.
			// Note that any "." after the 1st is part of the key name for maps:
			// labels.keyName && labels."key.name"
			parts := strings.SplitN(key, ".", 2)
			if len(parts) > 1 {
				obj := IncomingObj

				// "meta.abc" is special
				if parts[0] == "meta" {
					// Add "meta" if not already there
					if mAny, ok := obj["meta"]; !ok {
						meta = map[string]any{}
						obj["meta"] = meta
					} else {
						meta = mAny.(map[string]any)
					}

					// ---

					// If there are .'s then it's a non-scalar, convert it.
					// Note that any "." after the 1st is part of the key name for maps:
					// labels.keyName && labels."key.name"
					metaParts := strings.SplitN(parts[1], ".", 2)
					if len(metaParts) > 1 {
						obj := meta

						// Must be a map
						if _, ok := seenMetaMaps[metaParts[0]]; !ok {
							// First time we've seen this map, delete old stuff
							delete(obj, parts[0])
							seenMetaMaps[metaParts[0]] = true
						}

						for i, part := range metaParts {
							if i+1 == len(metaParts) {
								// Should we just skip all of this logic if nil?
								// If we try, watch for the case where someone
								// has just xReg-label-foo:null, it should probably
								// create the empty map anyway. And watch for the
								// case mentioned below
								if val != nil {
									obj[part] = val
								}
								continue
							}

							prop, ok := obj[part]
							if !ok {
								if val == nil {
									break
								}
								tmpO := map[string]any{}
								obj[part] = tmpO
								obj = map[string]any(tmpO)
							} else {
								obj, ok = prop.(map[string]any)
								PanicIf(!ok, "Prop isn't map: %#v", prop)
							}
						}
					} else {

						// ---

						// Make sure "abc" doesn't have any dots in it
						/*
							if strings.Index(parts[1], ".") >= 0 {
								return nil, NewXRError("header_error",
									"/"+info.OriginalPath,
									"name=xRegistry-"+key,
									`error_detail="meta" attributes must only be `+
										`one level deep, "`+parts[1]+`" is invalid"`)
							}
						*/

						meta[parts[1]] = val
					}
					continue
				}

				// Must be a map
				if _, ok := seenMaps[parts[0]]; !ok {
					// First time we've seen this map, delete old stuff
					delete(IncomingObj, parts[0])
					seenMaps[parts[0]] = true
				}

				for i, part := range parts {
					if i+1 == len(parts) {
						// Should we just skip all of this logic if nil?
						// If we try, watch for the case where someone
						// has just xReg-label-foo:null, it should probably
						// create the empty map anyway. And watch for the
						// case mentioned below
						if val != nil {
							obj[part] = val
						}
						continue
					}

					prop, ok := obj[part]
					if !ok {
						if val == nil {
							break
						}
						tmpO := map[string]any{}
						obj[part] = tmpO
						obj = map[string]any(tmpO)
					} else {
						obj, ok = prop.(map[string]any)
						PanicIf(!ok, "Prop isn't map: %#v", prop)
					}
				}
			} else {
				if IsNil(val) {
					if _, ok := seenMaps[key]; ok {
						// Do nothing if we've seen keys for this map already.
						// We don't want to erase any keys we just added.
						// This is an edge/error? case where someone included
						// xReg-label:null AND xreg-label-foo:foo - keep "foo"
					} else {
						// delete(IncomingObj, key)
						IncomingObj[key] = nil
					}
				} else {
					IncomingObj[key] = val
					// However, if there's a real (non-nil) value for 'key'
					// and we've seen this key as a map, delete the fact we've
					// seen this map/key before so it'll be created if we do
					// see a map entry again
					delete(seenMaps, key)
				}
			}
		}

		// Convert all HTTP header values into their proper data types
		// since as of now they're all just strings
		attrs := info.ResourceModel.GetBaseAttributes()
		attrs.AddIfValuesAttributes(IncomingObj)
		attrs.ConvertStrings(IncomingObj)

		// Now do the same thing is there's "meta"
		if meta != nil {
			attrs := info.ResourceModel.GetBaseMetaAttributes()
			attrs.AddIfValuesAttributes(meta)
			attrs.ConvertStrings(meta)
		}
	}

	return IncomingObj, nil
}

func HTTPProxy(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host") // http://xregistry.io/xreg
	path := r.URL.Query().Get("path") // /GROUPS?inline

	host = strings.Trim(host, "/")
	path = "/" + strings.Trim(path, "/")
	data := []byte(nil)

	reg, xErr := LoadRemoteRegistry(host)
	if xErr != nil {
		data = []byte(xErr.String())
	}

	log.VPrintf(4, "Download: %s%s", host, path)

	var err error
	if data == nil {
		data, err = DownloadURL(host + path)
		if !IsNil(err) {
			// See if we can be tricky and load the index.html file ourselves
			data, err = DownloadURL(host + path + "/index.html")
			if !IsNil(err) {
				data = []byte(err.Error())
			}
		}
	}

	log.VPrintf(4, "Data:\n%s", string(data))

	r.URL, err = url.Parse(path)
	r.RequestURI = path
	if err != nil {
		data = []byte(err.Error())
	}

	info := &RequestInfo{
		OriginalPath:    r.URL.Path, // path,
		OriginalRequest: r,          // not sure this is the best option
		Registry:        reg,
		BaseURL:         host,
		ProxyHost:       host,
		ProxyPath:       path,
	}

	if reg != nil && reg.Model != nil {
		if xErr = info.ParseRequestURL(); xErr != nil {
			data = []byte(xErr.String())
		}
	}

	w.Header().Add("Content-Type", "text/html")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Methods",
		"GET, PATCH, POST, PUT, DELETE")
	w.Header().Add("Link", fmt.Sprintf("<http://%s>;rel=xregistry-root",
		r.Host))

	html := GenerateUI(info, data)
	w.Write(html)
}

func HTTPWriteError(info *RequestInfo, errAny any) {
	var xErr *XRError
	var ok bool

	if xErr, ok = errAny.(*XRError); !ok {
		xErr = NewXRError("bad_request", "/"+info.OriginalPath,
			"error_detail="+fmt.Sprintf("%v", errAny))
	}

	info.StatusCode = xErr.Code
	// If header not already set, set it. This will likely only happen
	// when the error happens very very very early in our processing
	if info.GetHeader("Content-Type") == "" {
		info.SetHeader("Content-Type", "application/json; charset=utf-8")
	}

	// Add or replace Link header with xregistry-root rel
	// Check if there's already a Link header with rel=xregistry-root
	linkValue := fmt.Sprintf("<%s>;rel=xregistry-root", info.BaseURL)
	existingLinks := info.GetHeaderValues("Link")
	hasXRegistryLink := false
	for i, v := range existingLinks {
		// Check if this Link header has rel=xregistry-root
		if strings.Contains(v, "rel=xregistry-root") ||
			strings.Contains(v, "rel=\"xregistry-root\"") {

			// Replace it with the current value
			existingLinks[i] = linkValue
			hasXRegistryLink = true
			break
		}
	}
	if hasXRegistryLink {
		// Clear all Link headers and re-add them with the updated value
		info.OriginalResponse.Header().Del("Link")
		for _, v := range existingLinks {
			info.AddHeader("Link", v)
		}
	} else {
		// No existing xregistry-root link, just add it
		info.AddHeader("Link", linkValue)
	}

	for k, v := range xErr.Headers {
		info.AddHeader(k, v)
	}

	info.Write([]byte(xErr.ToJSON(info.BaseURL) + "\n"))
}
