package http

import (
	"bytes"
	// "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	// "os"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

type Server struct {
	XRSConfig  *Config
	Addr       string // dup of what's in XRSConfig
	Port       int    // dup of what's in XRSConfig
	HTTPServer *http.Server
}

func NewServer(xrsConfig *Config) *Server {
	// At some point we may need clone the config to avoid pollution but for
	// now just keep this as a remind of something to think about
	// xrsConfig = xrsConfig.Clone()

	addr := xrsConfig.GetAsString("http.addr")
	port := xrsConfig.GetAsInt("http.port")

	server := &Server{
		XRSConfig: xrsConfig,
		Addr:      addr,
		Port:      port,
		HTTPServer: &http.Server{
			Addr: fmt.Sprintf("%s:%d", addr, port),
		},
	}
	server.HTTPServer.Handler = server
	server.XRSConfig.Set("server", server)
	return server
}

func (s *Server) Close() {
	s.HTTPServer.Close()

	// Release the backend's shared resources, e.g. the DB connection pool
	if be, xErr := registry.NewDefaultBackend(s.XRSConfig); xErr == nil {
		be.Close(s.XRSConfig) //xrlint:ignore
	}
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
	log.VPrintf(1, "Listening on %s:%d", s.Addr, s.Port)
	err := s.HTTPServer.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Printf("Serve: %s", err)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uuid := NewUUID()
	info := registry.NewRequestInfo(uuid, s.XRSConfig, w, r)

	httpWriter := NewHttpWriter(info)
	info.Extras["HttpWriter"] = httpWriter // Dup in info.Reset()

	defer func() {
		// If we haven't written anything, this will force the HTTP status code
		// to be written and not default to 200
		httpWriter.Done()
	}()

	defer func() {
		// Unless an explicit Commit() is done, we'll always rollback. In
		// which case this rollback is a no-op
		if info.Tx != nil {
			info.Tx.Rollback() //xrlint:ignore
			info.Tx = nil
		}

		if rec := recover(); rec != nil {
			log.Printf("tx: %s panic: %s", uuid, rec)
			// ShowStack() // Down via NewXRError below now

			xErr := NewXRError("server_error", "/"+info.OriginalPath).
				SetDetail("An internal error occurred, contact the admin.")
			xErr.Instance = uuid
			HTTPWriteError(info, xErr)
		}
	}()

	saveVerbose := log.Clone()
	if tmp := r.URL.Query().Get("verbose"); tmp != "" {
		log.AddVerbose(tmp)
		defer func() {
			log.DelVerbose(tmp)
			log.Reset(saveVerbose)
		}()
	}

	// If Verbose > 2 (-vvv) or we're logging any keyword then show tx
	if log.GetLevel() > 2 || log.IsVerbose() {
		log.VPrintf(2, "tx: %s %s %s", uuid, r.Method, r.URL)
	} else {
		log.VPrintf(2, "%s %s", r.Method, r.URL)
	}

	if strings.HasPrefix(r.URL.Path, XRPROXY_PREFIX) {
		httpWriter.SentStatus = true // to avoid a dup w.Write() in defer
		HTTPXRProxy(uuid, w, r)
		return
	}

	rootApp := s.XRSConfig.GetAsString("rootapp")
	if rootApp != "ui" && rootApp != "xreg" {
		xErr := NewXRError("server_error", "/"+info.OriginalPath).
			SetDetailf("Unknown rootapp: %s.", rootApp)
		HTTPWriteError(info, xErr)
		return
	}

	path := r.URL.Path[1:] // remove leading /

	serveUI := (rootApp == "ui")

	uiSegment := s.XRSConfig.GetAsString("path.ui")
	if path == uiSegment || strings.HasPrefix(path, uiSegment+"/") {
		serveUI = true
	}

	defRegSegment := s.XRSConfig.GetAsString("path.defaultreg")
	regCollectionSegment := s.XRSConfig.GetAsString("path.regcollection")
	if path == ".xregistry" ||
		path == defRegSegment ||
		strings.HasPrefix(path, defRegSegment+"/") ||
		path == regCollectionSegment ||
		strings.HasPrefix(path, regCollectionSegment+"/") {

		serveUI = false
	}

	// Serve the new SPA UI for /ui and /ui/...
	if serveUI {
		httpWriter.SentStatus = true // to avoid a dup w.Write() in defer
		ServeUIStatic(s, uuid, w, r)
		return
	}

	// Buffer the raw request body ONCE, up front, so it can be replayed
	// if the retry loop needs to redo the whole request
	var savedBody []byte
	if r.Body != nil {
		savedBody, _ = io.ReadAll(r.Body)
		r.Body.Close()
	}

	const maxAttempts = 5

	var xErr *XRError
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		r.Body = io.NopCloser(bytes.NewReader(savedBody))

		xErr = s.serveOneAttempt(info)

		// no error, or not retryable, or we've already start to send data
		// then just exit and don't retry
		if xErr == nil || !xErr.Retryable || httpWriter.SentStatus {
			break
		}

		if attempt == maxAttempts {
			// Too many retries, send 503 instead of the the DB error
			xErr = NewXRError("server_busy", "/"+info.OriginalPath)
			xErr.Instance = info.UUID
			break
		}

		if log.GetLevel() == 1 && !log.IsVerbose() {
			log.Printf("Retrying %s %s after DB lock conflict "+
				"(attempt %d)", r.Method, r.URL, attempt)
		} else if log.GetLevel() > 1 {
			log.VPrintf(3, "tx: %s Retrying %s %s after DB lock "+
				"conflict (attempt %d)", uuid, r.Method, r.URL,
				attempt)
		}

		// Sleep a random multiplier of 40ms
		delay := time.Duration(40*attempt+rand.Int()%40) * time.Millisecond
		time.Sleep(delay)
		info.Reset()
		info.Extras["HttpWriter"] = httpWriter
		httpWriter.Reset()

		// If client connection is close, then don't bother to try again
		if info.OriginalRequest.Context().Err() != nil {
			break
		}
	}

	if xErr != nil {
		if xErr.Type == "server_error" {
			log.Printf(">>> Server Error <<<\n%s", xErr.Stack)
		}
		HTTPWriteError(info, xErr)
	}

}

// serveOneAttempt runs one full "new Tx -> parse -> dispatch by method ->
// commit/rollback" pass for a single attempt of ServeHTTP's retry loop.
// It's the caller's job to determine if we should retry on XRError
func (s *Server) serveOneAttempt(info *registry.RequestInfo) (retErr *XRError) {

	defer func() {
		if info.Tx != nil {
			// Make sure we don't leave any dangling TXs on something bad
			info.Tx.Rollback() //xrlint:ignore
			info.Tx = nil
		}

		// Converts an panicked XRError into a returned XRError insted so the
		// retry loop can handle it.
		// Shouldn't happen but there's one case (the Get() func) where we
		// don't return an xErr due to the big code impact. Maybe later
		if rec := recover(); rec != nil {
			if xErr, ok := rec.(*XRError); ok {
				retErr = xErr
			} else {
				panic(rec)
			}
		}
	}()

	be, xErr := registry.NewDefaultBackend(s.XRSConfig)
	if xErr != nil {
		return xErr
	}

	tx, xErr := registry.NewTx(info.UUID, s.XRSConfig, be)
	if xErr != nil {
		return xErr
	}

	info.Tx = tx
	tx.RequestInfo = info

	xErr = ProcessShortSelf(info)
	if xErr != nil {
		return xErr
	}

	xErr = info.ParseRequest(tx)
	if xErr != nil {
		return xErr
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
		switch info.OriginalRequest.Method {
		case "GET":
			tx.Lock()
			xErr = HTTPGet(info)
		case "PUT":
			xErr = HTTPPutPost(info)
		case "POST":
			xErr = HTTPPutPost(info)
		case "PATCH":
			xErr = HTTPPutPost(info)
		case "DELETE":
			xErr = HTTPDelete(info)
		case "OPTIONS":
			tx.Lock()
			xErr = HTTPOptions(info)
		default:
			xErr = NewXRError("action_not_supported", "/"+info.OriginalPath,
				"action="+info.OriginalRequest.Method)
		}
	}

	// Add our tx uuid to the error for debugging
	if xErr != nil && xErr.Instance == "" {
		xErr.Instance = info.UUID
	}

	if xErr != nil {
		// Ignore possible error since we'r already in error state
		tx.Rollback() //xrlint:ignore
	} else {
		xErr = tx.Commit()
	}

	info.Tx = nil

	return xErr
}

func HTTPGETCapabilities(info *registry.RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	buf := []byte(nil)
	var err error

	cap := info.Registry.Capabilities

	buf, err = json.MarshalIndent(cap, "", "  ")
	if err != nil {
		return NewXRError("server_error", "/").
			SetDetailf("Error parsing capabilities: %s.", err.Error())
	}

	httpWriter.SetHeader("Content-Type", "application/json")
	httpWriter.Write(buf)
	httpWriter.Write([]byte("\n"))
	return nil
}

func HTTPGETCapabilitiesOffered(info *registry.RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	buf := []byte(nil)
	var err error

	offered := GetOffered()
	buf, err = json.MarshalIndent(offered, "", "  ")
	if err != nil {
		return NewXRError("server_error", "/capabilitiesoffered").
			SetDetailf("Error parsing capabilitiesoffered: %s.", err.Error())
	}

	httpWriter.SetHeader("Content-Type", "application/json")
	httpWriter.Write(buf)
	httpWriter.Write([]byte("\n"))
	return nil
}

func HTTPGETModel(info *registry.RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	format := info.GetFlag("schema")
	if format == "" {
		format = "xRegistry-json"
	}

	model := info.Registry.Model
	if model == nil {
		model = &registry.Model{}
	}

	buf, xErr := model.SerializeForUser()
	if xErr != nil {
		return xErr
	}

	httpWriter.SetHeader("Content-Type", "application/json")
	httpWriter.Write(buf)
	httpWriter.Write([]byte("\n"))
	return nil
}

func HTTPGETModelSource(info *registry.RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	model := info.Registry.Model
	if model == nil {
		model = &registry.Model{}
	}

	modelSrc := model.Source
	if modelSrc == nil {
		modelSrc = []byte(`{}`)
	}

	buf, err := PrettyPrintJSON(modelSrc, "", "  ")
	if err != nil {
		return NewXRError("server_error", "/modelsource").
			SetDetailf("Error parsing modelsource: %s.", err.Error())
	}

	httpWriter.SetHeader("Content-Type", "application/json")
	httpWriter.Write(buf)
	httpWriter.Write([]byte("\n"))
	return nil
}

// HTTPGETXRegistryDiscovery serves GET .xregistry - a small, extensible,
// cross-registry discovery document. Unlike /model, /capabilities, etc.
// this isn't scoped to the single Registry the request happened to be
// routed through - it's host/server-level info, so it's exposed
// unconditionally (no capabilities.available gating) and reachable under
// ANY registry's own base URL (registry-scoped variant; a true host-root
// `GET <host>/.xregistry` variant, when a deployment controls its own
// host root, is a documented follow-up - not implemented here since this
// server always routes a request to a specific Registry first).
//
// Response shape is intentionally minimal today (just a "registries" array
// of URLs) but left open for future top-level attributes, which is why
// this got its own dedicated, dot-prefixed endpoint name (avoiding
// collisions with other tooling APIs, and avoiding ".well-known" - that
// name has strict RFC 8615 host-root-only semantics, which would be
// misleading for this registry-scoped variant).
//
// "registries" is intentionally just an array of URLs, not a map keyed
// by name - the name is server-owned info (the Registry's own "name"
// attribute, fetched from its own URL), so baking a name into this
// discovery doc would create a second, easily-stale copy of it. Clients
// that want a name should GET the registry's own URL and read it there.
func HTTPGETXRegistryDiscovery(info *registry.RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	names, xErr := registry.GetRegistryNames(info.XRSConfig)
	if xErr != nil {
		return xErr
	}

	defRegSegment := info.XRSConfig.GetAsString("path.defaultreg")
	regCollectionSegment := info.XRSConfig.GetAsString("path.regcollection")

	// Get the plain host base (scheme://host, no /xregs/<name> suffix)
	// regardless of whether THIS request itself came in via a /xregs/<name>
	// prefixed URL - every sibling registry's URL below is built from this
	// same host base, not from whatever prefix this particular request
	// happened to use.
	hostBase := info.BaseURL
	if info.Registry != nil {
		if strings.HasSuffix(hostBase,
			"/"+regCollectionSegment+"/"+info.Registry.UID) {

			hostBase = strings.TrimSuffix(hostBase,
				"/"+regCollectionSegment+"/"+info.Registry.UID)
		} else if strings.HasSuffix(hostBase, "/"+defRegSegment) {
			hostBase = strings.TrimSuffix(hostBase,
				"/"+defRegSegment)
		}
	}

	registries := make([]string, 0, len(names)+1)
	// The plain host base URL (no "/xregs/<name>" suffix) is itself a valid
	// way to reach a registry too if rootapp=xreg - it's just an alias for
	// whichever registry the server was started with as its default (see
	// GetDefaultReg()/cmds/xrserver's "Default(/): xregx/<name>" startup
	// log). Include it so clients relying solely on this discovery doc
	// don't miss that registry.

	rootApp := info.XRSConfig.GetAsString("rootapp")
	if rootApp == "xreg" {
		registries = append(registries, hostBase)
	}

	// But always add the default registry
	registries = append(registries, hostBase+"/xreg")

	for _, name := range names {
		registries = append(registries,
			hostBase+"/"+regCollectionSegment+"/"+name)
	}

	buf, err := json.MarshalIndent(map[string]any{
		"registries": registries,
	}, "", "  ")
	if err != nil {
		return NewXRError("server_error", "/.xregistry").
			SetDetailf("Error generating .xregistry doc: %s.", err.Error())
	}

	httpWriter.SetHeader("Content-Type", "application/json")
	httpWriter.Write(buf)
	httpWriter.Write([]byte("\n"))
	return nil
}

func HTTPGETContent(info *registry.RequestInfo) *XRError {
	defer log.Trace("tx: %s", info.UUID)()

	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	isV := log.IsFuncVerbose()
	XID := "/" + strings.Join(info.Parts, "/")

	iter, xErr := info.Registry.GetEntitiesFromXID(XID, true, true, false,
		FOR_READ)
	if xErr != nil {
		return xErr
	}

	entity, xErr := iter.Next()
	if xErr != nil {
		return xErr
	}

	if entity == nil {
		return NewXRError("not_found", XID)
	}

	var version *registry.Entity
	versionsCount := 0
	if info.VersionUID == "" {
		// We're on a Resource, so go find the default Version and count
		// how many versions there are for the VersionsCount attribute
		group, xErr := info.Registry.FindGroup(info.GroupType, info.GroupUID,
			false, FOR_READ)
		if xErr != nil {
			return xErr
		}
		if group == nil {
			return NewXRError("not_found", info.GetParts(2))
		}

		resource, xErr := group.FindResource(info.ResourceType,
			info.ResourceUID, false, FOR_READ)
		if xErr != nil {
			return xErr
		}
		if resource == nil {
			return NewXRError("not_found", info.GetParts(4))
		}
		meta := resource.MustFindMeta(false)

		// Count the versions and find the default one
		vID := meta.GetAsString("defaultversionid")
		for {
			v, xErr := iter.Next()
			if xErr != nil {
				return xErr
			}
			if v == nil {
				break
			}

			if v.Type == ENTITY_META {
				// Skip the "meta" subobject, but first grab the
				// "defaultversionid if we don't already have it,
				// which should only be true in the xref case
				if vID == "" {
					vID = v.GetAsString("defaultversionid")
				}
				continue
			}

			versionsCount++
			if v.UID == vID {
				version = v
			}
		}
	} else {
		version = entity
	}

	if isV {
		log.Printf("tx: %s Version: %#v", info.UUID, version)
	}

	headerIt := func(e *registry.Entity, key string, val any, attr *registry.Attribute) *XRError {
		if key[0] == '#' {
			return nil
		}

		if attr.Internals != nil && attr.Internals.NeverSerialize {
			return nil
		}

		if attr.Type == MAP && registry.IsScalar(attr.Item.Type) {
			for name, value := range val.(map[string]any) {
				// Should only ever be one of each xRegistry header
				httpWriter.SetHeader("xRegistry-"+key+"."+name,
					fmt.Sprintf("%v", value))
			}
			return nil
		}

		if !registry.IsScalar(attr.Type) {
			return nil
		}

		var headerName string
		if attr.Internals != nil && attr.Internals.HttpHeader != "" {
			headerName = attr.Internals.HttpHeader
		} else {
			headerName = "xRegistry-" + key
		}

		str := fmt.Sprintf("%v", val)
		// As above, should only ever be one of these
		httpWriter.AddHeader(headerName, str)

		return nil
	}

	xErr = entity.SerializeProps(headerIt)
	if xErr != nil {
		return (xErr)
	}

	if info.VersionUID == "" {
		httpWriter.SetHeader("xRegistry-versionscount",
			fmt.Sprintf("%d", versionsCount))
		httpWriter.SetHeader("xRegistry-versionsurl",
			info.BaseURL+entity.XID+"/versions")
	}
	httpWriter.SetHeader("Content-Location", info.BaseURL+version.XID)
	httpWriter.SetHeader("Content-Disposition",
		`inline; filename="`+info.ResourceUID+`"`)

	url := ""
	singular := info.ResourceModel.Singular
	if url = entity.GetAsString(singular + "url"); url != "" {
		// Should already be serialzied as a header
		// httpWriter.SetHeader("xRegistry-"+singular+"url", url)

		if httpWriter.StatusCode == 0 {
			// If we set it during a PUT/POST, don't override the 201
			httpWriter.StatusCode = http.StatusSeeOther
			httpWriter.SetHeader("Location", url)
		}
		return nil
	}

	url = entity.GetAsString(singular + "proxyurl")

	if isV {
		log.Printf("tx: %s %sproxyurl: %s", info.UUID, singular, url)
	}
	if url != "" {
		// Just act as a proxy and copy the remote resource as our response
		resp, err := http.Get(url)
		if err != nil {
			return NewXRError("parsing_response", "/"+info.OriginalPath,
				"error_detail="+err.Error())
		}
		if resp.StatusCode/100 != 2 {
			httpWriter.StatusCode = resp.StatusCode
			// return fmt.Error f("Remote error")
			// Let the body of the response be our body, below
		}

		// Copy all HTTP headers
		for header, value := range resp.Header {
			httpWriter.AddHeader(header, strings.Join(value, ","))
		}

		// Now copy the body
		_, err = io.Copy(httpWriter, resp.Body)
		if err != nil {
			return NewXRError("parsing_response", "/"+info.OriginalPath,
				"error_detail="+err.Error())
		}
		return nil
	}

	buf := version.Get(singular)
	if IsNil(buf) {
		// No data so just return
		/*
			if httpWriter.StatusCode == 0 {
				httpWriter.StatusCode = http.StatusNoContent
			}
		*/
		return nil
	}

	httpWriter.Write(buf.([]byte))

	return nil
}

func HTTPOptions(info *registry.RequestInfo) *XRError {
	defer log.Trace("tx: %s %s", info.UUID, info.OriginalPath)()

	// Headers will be set automatically by httpWriter.Write()
	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)
	httpWriter.StatusCode = 200

	return nil
}

func HTTPGet(info *registry.RequestInfo) *XRError {
	defer log.Trace("tx: %s %s", info.UUID, info.What)()

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
		return SerializeQuery(info, nil, "Registry")
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

	if info.RootPath == ".xregistry" {
		if !info.IsAvailable(".xregistry") {
			return NewXRError("not_available", "/.xregistry")
		}
		return HTTPGETXRegistryDiscovery(info)
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
	resXIDs := map[string][]string{
		"": []string{"/" + strings.Join(info.Parts, "/")},
	}
	return SerializeQuery(info, resXIDs, info.What)
}

func SerializeQuery(info *registry.RequestInfo, resXIDs map[string][]string,
	what string) *XRError {

	defer log.Trace("tx: %s", info.UUID)()

	// ?filter=excludeall returns nothing
	filters := info.Filters
	if len(filters) == 1 && filters[0][0].Path == "excludeall"+string(DB_IN) {
		if what != "Coll" {
			return NewXRError("bad_filter",
				info.OriginalRequest.URL.RequestURI(),
				"value=excludeall",
				"error_detail=\"excludeall\" must only be used "+
					"on collections")
		}
		return nil
	}

	sortKey := info.SortKey
	if sortKey != "" && what != "Coll" {
		return NewXRError("sort_noncollection",
			info.OriginalRequest.URL.RequestURI())
	}

	// Same for ?limit on non-collections:w
	if info.Limit != 0 && what != "Coll" {
		return NewXRError("pagination_noncollection",
			info.OriginalRequest.URL.RequestURI())
	}

	// Make sure everything is ok before we send back the results
	if xErr := info.Tx.Validate(); xErr != nil {
		return xErr
	}

	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	// resXIDs is used to group the items we want to return. In most cases
	// the items will all be part of one group where that group name doesn't
	// need to be returned - e.g. POST /schemagroup where the response will
	// just be a map of gIDs.
	// However, there are cases where we want to return multiple groupings
	// and each group has a different grouping name. For example:
	// POST / +   { "schemagroups": { "sg1"...}, "messagegroups": { "mg1"...}}
	// In this case the "XIDs" within each group are all processed as a
	// single query but the normal jwWriter stuff won't show the parent
	// group (schemagroups) because to do so would mean to also show the
	// attributes at that level (meaning the Registry attrs in this case).
	// To avoid this we pass in resXIDs which is a map of groupings for
	// each "XIDs" (IDs) we want to serialize.
	// Each key in the map becomes the grouping key/name.
	// When the key is "" then there shouldn't be any other keys and we
	// won't wrapper it at all.
	// So, "" is for things like:  POST .../rID/versions + map[vID]{version}
	// And "xxx" is for things like POST /  + map[GROUPS]map[gID]{group}
	if resXIDs == nil {
		resXIDs = map[string][]string{"": nil}
	}

	/*
		start := time.Now()

		defer func() {
			if log.GetLevel() > 3 {
				diff := time.Now().Sub(start).Truncate(time.Millisecond)
				log.Printf("tx: %s Total Time: %s", info.UUID, diff)
			}
		}()
	*/

	if info.RootPath == "export" {
		if !info.FlagEnabled("inline") || len(info.Inlines) == 0 {
			// Clear all inlines for cases where they specified them
			// but "inline" flag is not enabled
			info.Inlines = nil

			addInlines := []string{"*", "capabilities", "modelsource"}

			for _, add := range addInlines {
				if info.Registry.Capabilities.IsAvailable(add) {
					if xErr := info.AddInline(add); xErr != nil {
						return xErr
					}
				}
			}
		}
	}

	httpWriter.SetHeader("Content-Type", "application/json")
	var jw *registry.JsonWriter
	hasData := false
	keys := SortedKeys(resXIDs)
	for i, key := range keys {
		XIDs, ok := resXIDs[key]
		PanicIf(!ok, "can't find %q", key)

		var xErr *XRError
		var iter registry.EntityIterator

		// "!" is special - it means skip the query and just produce: {}
		if len(XIDs) != 1 || XIDs[0] != "!" {
			iter, xErr = info.Registry.GenerateResultEntities(what, XIDs)
			if xErr != nil {
				return xErr
			}
		}

		// Pagination Link headers MUST be added before any part of the
		// body is written
		if what == "Coll" && info.Limit != 0 {
			AddPaginationLinkHeaders(info, iter)
		}

		jw = registry.NewJsonWriter(info, httpWriter, iter)

		_, xErr = jw.NextEntity()
		if xErr != nil {
			return xErr
		}

		// Collections will need to print the {}, so don't error for them
		if what != "Coll" {
			if jw.Entity == nil {
				// Special case, if the URL is ../rID/versions/vID?doc then
				// check to see if Resource has xref set, if so then the error
				// is 400, not 404
				if info.VersionUID != "" && info.DoDocView() {
					xid := "/" +
						strings.Join(info.Parts[:len(info.Parts)-2], "/")
					xid += "/meta"
					entity, xErr := info.Registry.GetEntityFromXID(xid, false,
						false, FOR_READ)
					if xErr != nil {
						return xErr
					}

					// Assume that if we can't find the Resource's meta object
					// then the Resource doesn't exist, so a 404 really is the
					// best response in those cases, so skip the 400
					if entity != nil && !IsNil(entity.Object["xref"]) {
						return NewXRError("cannot_doc_xref",
							"/"+info.OriginalPath)
					}
				}

				return NewXRError("not_found", info.GetParts(0))
			}
		}

		// Special case, if we're doing a collection, let's make sure we didn't
		// get an empty result due to it's parent not even existing - for
		// example the user used the wrong case (or even name) in the parent's
		// XID
		if what == "Coll" && jw.Entity == nil && len(info.Parts) > 2 {
			xid := "/" + strings.Join(info.Parts[:len(info.Parts)-1], "/")

			entity, xErr := info.Registry.GetEntityFromXID(xid, false, false,
				FOR_READ)
			if xErr != nil {
				return xErr
			}

			if entity == nil {
				return NewXRError("not_found", xid)
			}
		}

		// GROUPS/GID/RESOURCES/RID/versions
		// Another special case .../rID/versions?doc when rID has xref
		if jw.Entity == nil && info.DoDocView() && len(info.Parts) == 5 &&
			info.Parts[4] == "versions" {

			// Should be our case since "versions" can never be empty except
			// when xref is set. If this is not longer true then we'll need to
			// check this Resource's xref to see if it's set.
			// Can copy the Raw EntityFromXID... stuff above
			return NewXRError("cannot_doc_xref", "/"+info.OriginalPath)
		}

		// Only do this if we're adding the extra grouping wrapper
		if key != "" {
			// Cause the jwWriter to indent since we're adding a wrapper
			jw.Indent = "  "
			if i == 0 {
				jw.Print("{\n")
			}
			jw.Printf("  %q: ", key)
		}

		if what == "Coll" {
			_, _, _, xErr = jw.WriteCollection()
		} else {
			xErr = jw.WriteEntity()
		}

		if xErr != nil {
			return xErr
		}

		if jw.HasData {
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

// AddPaginationLinkHeaders adds the pagination spec's "Link" HTTP headers
// (rel=self|first|prev|next|last) for a paginated "Coll" response. Only
// "self" carries the offset/limit/count parameters ("count" being the total
// number of top-level entities across the *entire* filtered collection). Must be called before anything is written to the response
// body, since HTTP headers can't be sent after the body has started.
func AddPaginationLinkHeaders(info *registry.RequestInfo, iter registry.EntityIterator) {
	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	limit := info.Limit
	offset := info.Offset

	total := uint64(0)
	gotTotal := false

	if iter.Total() != math.MaxUint64 {
		total = iter.Total()
		gotTotal = true
	}

	basePath := "/" + info.OriginalPath
	origQuery := info.OriginalRequest.URL.Query()

	// This will keep the original URL (and query params) and just replace
	// the pagination ones as needed. The key thing here is that we're
	// keeping others, like sort and filter
	buildLink := func(off uint64) string {
		q := url.Values{}
		for k, v := range origQuery {
			if k == "offset" || k == "limit" {
				continue
			}
			q[k] = v
		}
		q.Set("limit", strconv.FormatUint(limit, 10))
		q.Set("offset", strconv.FormatUint(off, 10))
		return fmt.Sprintf("<%s%s?%s>", info.BaseURL, basePath, q.Encode())
	}

	addLink := func(rel string, off uint64) {
		httpWriter.AddHeader("Link", fmt.Sprintf("%s;rel=%s",
			buildLink(off), rel))
	}

	// "self" is on every response and carries the pagination metadata. Per
	// the spec "offset" MUST NOT appear when the page has no records, and
	// "count" is only included when the total is actually known.
	addSelf := func(withOffset bool, withCount bool) {
		v := fmt.Sprintf("%s;rel=self", buildLink(offset))
		if withOffset {
			v += fmt.Sprintf(";offset=%d", offset)
		}
		v += fmt.Sprintf(";limit=%d", limit)
		if withCount {
			v += fmt.Sprintf(";count=%d", total)
		}
		httpWriter.AddHeader("Link", v)
	}

	hasRows := gotTotal

	if !gotTotal && offset == 0 {
		// We asked for the very first page (offset=0) and got zero rows
		// back - nothing was filtered out by the page-range WHERE clause
		// at offset 0, so the entire (filtered) collection is genuinely
		// empty. The total is just 0
		total = 0
		gotTotal = true
	}

	if !gotTotal {
		// offset > 0 and the page came back empty: the requested offset
		// is past the end of a result set whose true current size we
		// don't know (and it's not worth an extra DB round-trip to find
		// out - especially since any client can trigger this simply by
		// passing a huge "offset="). Only "first" is unambiguously correct
		// regardless of the (unknown) true total, so emit just that (plus
		// "self" without offset/count) so the client can restart.
		addSelf(false, false)
		addLink("first", 0)
		return
	}

	addSelf(hasRows, true)

	// If the very first page already contains the entire (filtered) result
	// set, there's nothing to paginate through, so only "self" is needed.
	if offset == 0 && total <= limit {
		return
	}

	// per spec: "first" MAY be included in any response.
	addLink("first", 0)

	// per spec: "prev" MUST NOT be present if this is the start of the set.
	if offset > 0 {
		prevOffset := uint64(0)
		if offset > limit {
			prevOffset = offset - limit
		}
		addLink("prev", prevOffset)
	}

	// per spec: "next" MUST NOT be present if this is the end of the set.
	if offset+limit < total {
		addLink("next", offset+limit)
	}

	// per spec: "last" MAY be included in any response.
	lastOffset := uint64(0)
	if total > 0 {
		lastOffset = ((total - 1) / limit) * limit
	}
	addLink("last", lastOffset)
}

func HTTPPutPost(info *registry.RequestInfo) *XRError {
	method := info.OriginalRequest.Method

	defer log.Trace("tx: %s %s %s", info.UUID, method, info.OriginalPath)()

	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	isNew := false
	XIDs := ([]string)(nil)
	what := "Entity"
	numParts := len(info.Parts)

	metaInBody := (info.ResourceModel == nil) ||
		(info.ResourceModel.GetHasDocument() == false || info.ShowDetails ||
			(numParts == 5 && info.Parts[4] == "meta"))

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
		return NewXRError("not_available", "/"+info.OriginalPath).
			SetDetail("Registry data is read-only.")
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
		entityType := "Resource"
		if numParts == 6 {
			entityType = "Version"
		}
		return NewXRError("details_required", "/"+info.OriginalPath).
			SetDetail("PATCH is not allowed on " + entityType + " documents.")
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
			addType := registry.ADD_UPDATE
			if method == "PATCH" {
				addType = registry.ADD_PATCH
			}
			xErr = info.Registry.Update(IncomingObj, addType)
			if xErr != nil {
				return xErr
			}

			// Return HTTP GET of Registry root
			resXIDs := map[string][]string{"": []string{""}}
			return SerializeQuery(info, resXIDs, "Registry")
		}

		// Must be POST /    + body:map[GROUPS]map[id]Group

		newGs, xErr := info.Registry.UpsertJustGroups(IncomingObj, registry.ADD_UPDATE)
		if xErr != nil {
			return xErr
		}

		resXIDs := map[string][]string{}
		// Special case - if req is {} then make response {}
		if len(newGs) == 0 {
			resXIDs = map[string][]string{"": []string{"!"}}
		} else {
			for gType, groups := range newGs {
				resXIDs[gType] = []string{}
				for _, g := range groups {
					resXIDs[gType] = append(resXIDs[gType], g.XID)
				}
				if len(resXIDs[gType]) == 0 {
					// Force an empty collection to be returned
					resXIDs[gType] = []string{"!"}
				}
			}
		}

		// Return HTTP GET of Groups created or updated
		return SerializeQuery(info, resXIDs, "Coll")
	}

	// URL: /GROUPs[/gID]...
	// ////////////////////////////////////////////////////////////////
	group := (*registry.Group)(nil)
	groupUID := info.GroupUID

	if numParts == 1 {
		// POST /GROUPs + body:map[id]Group

		objMap, xErr := IncomingObj2Map(IncomingObj, info.GetParts(0),
			"\""+info.GroupModel.Singular+"\"")
		if xErr != nil {
			return xErr
		}

		addType := registry.ADD_UPSERT
		if method == "PATCH" {
			addType = registry.ADD_PATCH
		}

		for id, obj := range objMap {
			g, _, xErr := info.Registry.UpsertGroupWithObject(info.GroupType,
				id, obj, addType)
			if xErr != nil {
				return xErr
			}
			XIDs = append(XIDs, g.XID)
		}

		if len(XIDs) == 0 {
			XIDs = []string{"!"} // Force an empty collection to be returned
		}

		// Return HTTP GET of Groups created or updated
		resXIDs := map[string][]string{"": XIDs}
		return SerializeQuery(info, resXIDs, "Coll")
	}

	if numParts == 2 {
		// PUT    /GROUPs/gID  + body: {group}
		// PATCH  /GROUPs/gID  + body: {group}
		// POST   /GROUPs/gID  + body: map[rType]map[rID]{resource}

		if method == "PUT" || method == "PATCH" {
			addType := registry.ADD_UPSERT
			if method == "PATCH" {
				addType = registry.ADD_PATCH
			}

			group, isNew, xErr := info.Registry.UpsertGroupWithObject(info.GroupType,
				info.GroupUID, IncomingObj, addType)
			if xErr != nil {
				return xErr
			}

			if isNew { // 201, else let it default to 200
				httpWriter.SetHeader("Location", info.BaseURL+group.XID)
				httpWriter.StatusCode = http.StatusCreated
			}

			// Return HTTP GET of Group
			resXIDs := map[string][]string{"": []string{group.XID}}
			return SerializeQuery(info, resXIDs, "Entity")
		}

		// Must be POST /GROUPs/gID + body: map[rType]map[rID]{resource}

		// We'll lock it later if needed
		group, xErr = info.Registry.FindGroup(info.GroupType, groupUID, false,
			FOR_READ)
		if xErr != nil {
			return xErr
		}

		if group == nil {
			group, _, xErr = info.Registry.UpsertGroup(info.GroupType, groupUID)
			if xErr != nil {
				return xErr
			}
		}

		newRs, xErr := group.UpsertJustResources(IncomingObj, registry.ADD_UPDATE)
		if xErr != nil {
			return xErr
		}

		resXIDs := map[string][]string{}

		// Special case - if req is {} then make response {}
		if len(IncomingObj) == 0 {
			resXIDs = map[string][]string{"": []string{"!"}}
		} else {
			for rType, resources := range newRs {
				resXIDs[rType] = []string{}
				for _, r := range resources {
					resXIDs[rType] = append(resXIDs[rType], r.XID)
				}
				if len(resXIDs[rType]) == 0 {
					resXIDs[rType] = []string{"!"}
				}
			}
		}

		// Return HTTP GET of Resources created or updated
		return SerializeQuery(info, resXIDs, "Coll")
	}

	// Must be PUT/POST /GROUPs/gID/...

	group, xErr = info.Registry.FindGroup(info.GroupType, groupUID, false,
		FOR_READ)
	if xErr != nil {
		return xErr
	}

	// If not found, create it
	if group == nil {
		group, _, xErr = info.Registry.UpsertGroup(info.GroupType, groupUID)
		if xErr != nil {
			return xErr
		}
	}

	// URL: /GROUPs/gID/RESOURCEs...
	// ////////////////////////////////////////////////////////////////

	// Some global vars
	resource := (*registry.Resource)(nil)
	version := (*registry.Version)(nil)
	resourceUID := info.ResourceUID
	versionUID := info.VersionUID
	setDefVerID := info.GetFlag("setdefaultversionid")
	locationVersionUID := ""
	needLocationVersionIDFromMeta := false

	// Do Resources and Versions at the same time
	// URL: /GROUPs/gID/RESOURCEs
	// URL: /GROUPs/gID/RESOURCEs/rID
	// URL: /GROUPs/gID/RESOURCEs/rID/versions[/vID]
	// ////////////////////////////////////////////////////////////////

	// If there isn't an explicit "return" then this assumes we're left with
	// a version and will return that back to the client

	if numParts == 3 {
		// POST GROUPs/gID/RESOURCEs + body:map[id]Resource

		// Check if setdefaultversionid flag is present - it's not allowed on
		// operations that allow modifying multiple Resources
		if setDefVerID != "" {
			return NewXRError("bad_flag", info.GetParts(0),
				"flag=setdefaultversionid")
		}

		objMap, xErr := IncomingObj2Map(IncomingObj,
			info.GetParts(0), "\""+info.ResourceModel.Singular+"\"")
		if xErr != nil {
			return xErr
		}

		// For each Resource in the map, upsert it and add it's XID to result
		addType := registry.ADD_UPSERT
		if method == "PATCH" {
			addType = registry.ADD_PATCH
		}

		for id, obj := range objMap {
			r, _, xErr := group.UpsertResource(&registry.ResourceUpsert{
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
			XIDs = append(XIDs, r.XID)
		}

		if len(XIDs) == 0 {
			XIDs = []string{"!"} // Force an empty collection to be returned
		}

		// Return HTTP GET of Resources created or modified
		resXIDs := map[string][]string{"": XIDs}
		return SerializeQuery(info, resXIDs, "Coll")
	}

	if numParts > 3 {
		// GROUPs/gID/RESOURCEs/rID...
		// Any write action on a Resource or version needs to lock the entire
		// tree due to the interdependencies of everything, esp. with the
		// compatibility and constraints stuff. So lock this "Find" result.
		resource, xErr = group.FindResource(info.ResourceType, resourceUID,
			false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}
	}

	if numParts == 4 && (method == "PUT" || method == "PATCH") {
		// PUT|PATCH GROUPs/gID/RESOURCEs/rID [$details]

		if resource != nil {
			// ID needs to be the version's ID, not the Resources
			// IncomingObj["id"] = version.UID

			// Create a new Resource and it's first/only/default Version
			addType := registry.ADD_UPSERT
			if method == "PATCH" || !metaInBody {
				addType = registry.ADD_PATCH
			}
			resource, _, xErr = group.UpsertResource(&registry.ResourceUpsert{
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
		} else {
			// Upsert resource's default version

			addType := registry.ADD_UPSERT
			if method == "PATCH" {
				addType = registry.ADD_PATCH
			}
			resource, isNew, xErr = group.UpsertResource(&registry.ResourceUpsert{
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
			resource, isNew, xErr = group.UpsertResource(&registry.ResourceUpsert{
				RType:            info.ResourceType,
				Id:               resourceUID,
				VID:              propsID,
				Obj:              IncomingObj,
				AddType:          registry.ADD_ADD,
				ObjIsVer:         true,
				DefaultVersionID: setDefVerID,
			})
			if xErr != nil {
				return xErr
			}

			// Grab the one and only version UID, we need it for the
			// "location" header in the response. Resource.ValidateResource()
			// (which computes/sets meta's "defaultversionid") is now
			// deferred (see Tx.AddResourceToValidate()), so we can't read
			// it yet - do it after info.Tx.Validate() runs below instead.
			needLocationVersionIDFromMeta = true
		} else {
			version, isNew, xErr = resource.UpsertVersionWithObject(&registry.VersionUpsert{
				Id:               propsID,
				Obj:              IncomingObj,
				AddType:          registry.ADD_UPSERT,
				More:             false,
				DefaultVersionID: setDefVerID,
			})
			if xErr == nil {
				locationVersionUID = version.UID
			}
		}
		if xErr != nil {
			return xErr
		}
		// Default to just returning the version
	}

	// GROUPs/gID/RESOURCEs/rID/meta
	if numParts > 4 && info.Parts[4] == "meta" {
		// PUT /GROUPs/gID/RESOURCEs/rID/meta
		addType := registry.ADD_UPSERT
		if method == "PATCH" {
			addType = registry.ADD_PATCH
		}

		if resource == nil {
			isNew = true

			// Implicitly create the resource
			resource, _, xErr = group.UpsertResource(&registry.ResourceUpsert{
				// TODO check to see if "" should be propsID
				RType:            info.ResourceType,
				Id:               resourceUID,
				VID:              "",
				Obj:              map[string]any{},
				AddType:          registry.ADD_ADD,
				ObjIsVer:         false,
				DefaultVersionID: setDefVerID,
			})
			if xErr != nil {
				return xErr
			}
		}

		// setdefaultversionid flag MUST override any values in request body
		if setDefVerID != "" {
			IncomingObj["defaultversionid"] = setDefVerID
			IncomingObj["defaultversionsticky"] = true
		}

		// Technically, this will always "update" not "insert"
		meta, _, xErr := resource.UpsertMeta(&registry.MetaUpsert{
			Obj:           IncomingObj,
			AddType:       addType,
			CreateVersion: true,
			More:          false,
		})

		if xErr != nil {
			return xErr
		}

		// Return HTTP GET of 'meta'
		if isNew { // 201, else let it default to 200
			httpWriter.SetHeader("Location", info.BaseURL+meta.XID)
			httpWriter.StatusCode = http.StatusCreated
		}

		resXIDs := map[string][]string{"": []string{meta.XID}}
		return SerializeQuery(info, resXIDs, "Entity")
	}

	// Just double-check
	if numParts > 4 {
		PanicIf(info.Parts[4] != "versions", "Not 'versions': %s",
			info.Parts[4])
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
		objMap, xErr := IncomingObj2Map(IncomingObj, "/"+info.OriginalPath,
			"version")
		if xErr != nil {
			return xErr
		}

		// ?setdefaultversionid=request w/o any versions is an error
		if len(objMap) == 0 && setDefVerID == "request" {
			return NewXRError("defaultversionid_request", "/"+info.OriginalPath)
		}

		if resource == nil {
			// Implicitly create the resource

			// Creating a Resource w/o any Versions is an error.
			// We could create a dummy Version but the user explicitly
			// left the map empty...so we have no idea what they really meant.
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

			addType := registry.ADD_UPSERT
			if method == "PATCH" {
				addType = registry.ADD_PATCH
			}

			resource, _, xErr = group.UpsertResource(&registry.ResourceUpsert{
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

			// Remove the newly created default version from objMap so we
			// won't process it again, but add it to the reuslts collection
			for id, _ := range objMap {
				XIDs = append(XIDs, "/"+strings.Join(
					[]string{info.GroupType, info.GroupUID, info.ResourceType,
						info.ResourceUID, "versions", id},
					"/"))
			}
		} else {
			meta := resource.MustFindMeta(false)

			if meta.Get("readonly") == true {
				if resource.GetRequestInfo().HasIgnore("readonly") {
					// ?ignore=readonly so just stop w/o error
					// Force an empty collection to be returned
					XIDs = []string{"!"}
					resXIDs := map[string][]string{"": XIDs}
					return SerializeQuery(info, resXIDs, "Coll")
				} else {
					return NewXRError("readonly", resource.XID)
				}
			}

			// Process the versions
			addType := registry.ADD_UPSERT
			if method == "PATCH" {
				addType = registry.ADD_PATCH
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

				v, _, xErr := resource.UpsertVersionWithObject(&registry.VersionUpsert{
					Id:               id,
					Obj:              obj,
					AddType:          addType,
					More:             more,
					DefaultVersionID: defv,
				})
				if xErr != nil {
					return xErr
				}

				XIDs = append(XIDs, v.XID)
			}
		}

		if len(XIDs) == 0 {
			XIDs = []string{"!"} // Force an empty collection to be returned
		}
		resXIDs := map[string][]string{"": XIDs}
		return SerializeQuery(info, resXIDs, "Coll")
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

		version, xErr = resource.FindVersion(versionUID, false)
		if xErr != nil {
			return xErr
		}

		if version == nil {
			// We have a Resource, so add a new Version based on IncomingObj
			version, isNew, xErr = resource.UpsertVersionWithObject(&registry.VersionUpsert{
				Id:               versionUID,
				Obj:              IncomingObj,
				AddType:          registry.ADD_UPSERT,
				More:             false,
				DefaultVersionID: setDefVerID,
			})
		} else if !isNew {
			addType := registry.ADD_UPSERT
			if method == "PATCH" || !metaInBody {
				addType = registry.ADD_PATCH
			}
			version, _, xErr = resource.UpsertVersionWithObject(&registry.VersionUpsert{
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
		locationVersionUID = version.UID
	}

	PanicIf(xErr != nil, "err should be nil")

	// Make sure everything is ok before we send back the results
	if xErr := info.Tx.Validate(); xErr != nil {
		return xErr
	}

	if needLocationVersionIDFromMeta {
		// Now that validation has run (deferred via
		// Tx.AddResourceToValidate()), meta's "defaultversionid" is
		// guaranteed to be resolved - safe to read it now.
		meta := resource.MustFindMeta(false)
		locationVersionUID = meta.GetAsString("defaultversionid")
	}

	originalLen := numParts

	// Need to setup info stuff in case we call HTTPGetContent
	info.Parts = []string{info.Parts[0], groupUID,
		info.Parts[2], resourceUID}
	info.What = "Entity"
	info.GroupUID = groupUID
	info.ResourceUID = resourceUID // needed for $details in URLs

	location := info.BaseURL + resource.XID
	if originalLen > 4 || (originalLen == 4 && method == "POST") {
		info.VersionUID = locationVersionUID
		info.Parts = append(info.Parts, "versions", info.VersionUID)
		location += "/versions/" + info.VersionUID
	}

	if info.ShowDetails { // not 100% sure this the right way/spot
		location += "$details"
	}

	if isNew { // 201, else let it default to 200
		httpWriter.SetHeader("Location", location)
		httpWriter.StatusCode = http.StatusCreated
	}

	// Return the contents of the entity instead of the xReg metadata
	if !metaInBody {
		return HTTPGETContent(info)
	}

	// Return the xReg metadata of the entity processed
	if XIDs == nil {
		XIDs = []string{"/" + strings.Join(info.Parts, "/")}
	}

	resXIDs := map[string][]string{"": XIDs}
	return SerializeQuery(info, resXIDs, what)
}

func HTTPPUTCapabilities(info *registry.RequestInfo) *XRError {
	if len(info.Parts) > 1 || !info.IsAvailable("capabilities") {
		return NewXRError("not_available", "/"+info.GetParts(0))
	}

	reqBody, err := RemoveSchema(info.Body)
	if err != nil {
		return NewXRError("parsing_data", "request JSON",
			"error_detail="+err.Error())
	}

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
			return NewXRError("parsing_data", "request JSON",
				"error_detail="+err.Error())
		}

		reqBody, _ = json.Marshal(tmp)
	} else {
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action="+info.OriginalRequest.Method)
	}

	xErr := (*XRError)(nil)
	info.Registry.Capabilities, xErr = ParseCapabilities(reqBody)
	if xErr != nil {
		return xErr
	}

	if xErr = info.Registry.SaveCapabilities(); xErr != nil {
		return xErr
	}

	return HTTPGETCapabilities(info)
}

func HTTPPUTModelSource(info *registry.RequestInfo) *XRError {
	if len(info.Parts) > 1 || !info.IsAvailable("modelsource") {
		return NewXRError("not_available", "/"+info.GetParts(0))
	}

	if info.OriginalRequest.Method != "PUT" {
		return NewXRError("action_not_supported", "/modelsource",
			"action="+info.OriginalRequest.Method)
	}

	// null and {} both mean "reset model to empty" — treat identically.
	body := bytes.TrimSpace(info.Body)

	if len(body) == 0 {
		return NewXRError("missing_body", "/"+info.OriginalPath)
	}

	if string(body) == "null" {
		body = []byte("{}")
	}

	// A Model change can impact/invalidate assumptions across the
	// entire Registry tree, so lock the whole Registry row FOR_WRITE
	// before applying the new model - same physical lock
	// Registry.Update() takes for "/" PUT/PATCH, just triggered here
	// via the /modelsource entry point instead.
	// DUG - do we need to lock everything in Entities too?
	// I wonder if a PUT to a version would be let thru and missed??
	if _, xErr := info.Registry.Lock(); xErr != nil {
		return xErr
	}

	xErr := info.Registry.SetModel(nil, string(body), true)
	if xErr != nil {
		return xErr
	}

	return HTTPGETModelSource(info)
}

func HTTPDelete(info *registry.RequestInfo) *XRError {
	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	// DELETE /...
	if len(info.Parts) == 0 {
		// DELETE /
		return NewXRError("action_not_supported", "/", "action=DELETE")
	}

	if !info.IsAvailableMutable("entities") {
		return NewXRError("not_available", "/"+info.OriginalPath)
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
		return info.Tx.Validate()
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

		if xErr := info.Tx.Validate(); xErr != nil {
			return xErr
		}

		httpWriter.StatusCode = http.StatusNoContent
		return nil
	}

	// DELETE /GROUPs/gID/RESOURCEs...
	if rm := gm.Resources[info.ResourceType]; rm == nil {
		return NewXRError("not_found", info.GetParts(3))
	}

	if len(info.Parts) == 3 {
		// DELETE /GROUPs/gID/RESOURCEs
		xErr = HTTPDeleteResources(info)
		if xErr != nil {
			return xErr
		}
		return info.Tx.Validate()
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

	meta := resource.MustFindMeta(false)

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

		if xErr := info.Tx.Validate(); xErr != nil {
			return xErr
		}

		httpWriter.StatusCode = http.StatusNoContent
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

		return info.Tx.Validate()
	}

	// DELETE /GROUPs/gID/RESOURCEs/rID/versions/vID...
	version, xErr := resource.FindVersion(info.VersionUID, false)
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

		if xErr := info.Tx.Validate(); xErr != nil {
			return xErr
		}

		httpWriter.StatusCode = http.StatusNoContent
		return nil
	}

	return NewXRError("not_found", info.GetParts(0))
}

type EpochEntry map[string]any
type EpochEntryMap map[string]EpochEntry

func LoadEpochMap(info *registry.RequestInfo) (EpochEntryMap, *XRError) {
	res := EpochEntryMap{}

	bodyStr := strings.TrimSpace(string(info.Body))

	if len(bodyStr) > 0 {
		err := Unmarshal([]byte(bodyStr), &res)
		if err != nil {
			return nil, NewXRError("parsing_data", "request JSON",
				"error_detail="+err.Error())
		}
	} else {
		// EpochEntryMap == nil mean no list at all, not same as empty list
		return nil, nil
	}

	return res, nil
}

func HTTPDeleteGroups(info *registry.RequestInfo) *XRError {
	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	list, xErr := LoadEpochMap(info)
	if xErr != nil {
		return xErr
	}

	// No list provided so get list of Groups so we can delete them all
	// TODO: optimize this to just delete it all in one shot
	if list == nil {
		list = EpochEntryMap{}
		IDs, xErr := info.Registry.GetGroupIDsByType(info.GroupType, FOR_WRITE)
		if xErr != nil {
			return xErr
		}

		for _, gID := range IDs {
			list[gID] = EpochEntry{}
		}
	}

	// Delete each Group, checking epoch first if provided
	for id, entry := range list {
		group, xErr := info.Registry.FindGroup(info.GroupType, id, false,
			FOR_WRITE)
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

	httpWriter.StatusCode = http.StatusNoContent
	return nil
}

func HTTPDeleteResources(info *registry.RequestInfo) *XRError {
	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	list, xErr := LoadEpochMap(info)
	if xErr != nil {
		return xErr
	}

	group, xErr := info.Registry.FindGroup(info.GroupType, info.GroupUID,
		false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}

	// No list provided so get list of Resources so we can delete them all
	// TODO: optimize this to just delete it all in one shot
	if list == nil {
		list = EpochEntryMap{}
		IDs, xErr := group.GetResourceIDsByType(info.ResourceType, FOR_WRITE)
		if xErr != nil {
			return xErr
		}

		for _, gID := range IDs {
			list[gID] = EpochEntry{}
		}
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

		meta := resource.MustFindMeta(false)

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

	httpWriter.StatusCode = http.StatusNoContent
	return nil
}

func HTTPDeleteVersions(info *registry.RequestInfo) *XRError {
	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	nextDefault := info.GetFlag("setdefaultversionid")

	list, xErr := LoadEpochMap(info)
	if xErr != nil {
		return xErr
	}

	group, xErr := info.Registry.FindGroup(info.GroupType, info.GroupUID,
		false, FOR_READ)
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

		VAs, xErr := resource.GetVersionIDs()
		if xErr != nil {
			return xErr
		}
		for _, va := range VAs {
			list[va.VID] = EpochEntry{}
		}
	}

	// Before we delete each one, make sure the epoch value is ok.
	// We can't check and delete at the same time before deleting one
	// might update a future one's ancestorid value which will also change
	// its epoch value - which will make the epoch check fail.
	// However, we can save the Versions for the next loop.
	vers := []*registry.Version{}
	for id, entry := range list {
		version, xErr := resource.FindVersion(id, false)
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
		version, xErr := resource.FindVersion(nextDefault, false)
		if xErr != nil {
			return xErr
		}
		xErr = resource.SetDefault(version)
		if xErr != nil {
			return xErr
		}
	}

	httpWriter.StatusCode = http.StatusNoContent
	return nil
}

func ExtractIncomingObject(info *registry.RequestInfo, body []byte) (Object, *XRError) {
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
			return nil, NewXRError("parsing_data", "request JSON",
				"error_detail="+err.Error())
		}

		// "modelsource" is sooo special! Don't parse it into a golang type
		// keep it as []byte so that we preserve the order of the map keys.
		// Check key existence (not non-nil) so explicit null is also
		// captured as json.RawMessage("null"), not left as Go nil.
		_, msExists := IncomingObj["modelsource"]

		if len(info.Parts) == 0 && msExists {
			tmpReg := struct {
				ModelSource json.RawMessage
			}{}
			if err := json.Unmarshal(body, &tmpReg); err != nil {
				return nil, NewXRError("parsing_data", "request JSON",
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

		for name, attr := range registry.SpecialAttrHeaders {
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
						fmt.Sprintf("'xRegistry-%s' isn't a valid HTTP "+
							"header", key))
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

func HTTPWriteError(info *registry.RequestInfo, errAny any) {
	var xErr *XRError
	var ok bool

	httpWriter := info.Extras["HttpWriter"].(*HttpWriter)

	if xErr, ok = errAny.(*XRError); !ok {
		xErr = NewXRError("bad_request", "/"+info.OriginalPath,
			"error_detail="+fmt.Sprintf("%v", errAny))
	}

	if xErr.IsType("action_not_supported") {
		httpWriter.AddHeader("Allow",
			strings.Join(httpWriter.GetAllowedMethods(), ", "))
	}

	// Add our tx uuid to the error for debugging
	if xErr != nil && xErr.Instance == "" {
		xErr.Instance = info.UUID
	}

	httpWriter.StatusCode = xErr.Code
	// If header not already set, set it. This will likely only happen
	// when the error happens very very very early in our processing
	if httpWriter.GetResponseHeader("Content-Type") == "" {
		httpWriter.SetHeader("Content-Type", "application/json; charset=utf-8")
	}

	httpWriter.AddRegistryRootHeader()

	httpWriter.Write([]byte(xErr.ToJSON() + "\n"))
}

func (hw *HttpWriter) AddRegistryRootHeader() {
	// Add or replace Link header with xregistry-root rel
	// Check if there's already a Link header with rel=xregistry-root

	linkValue := fmt.Sprintf("<%s>;rel=xregistry-root", hw.Info.BaseURL)

	existingLinks := hw.GetResponseHeaderValues("Link")
	for _, v := range existingLinks {
		// Check if this Link header has rel=xregistry-root
		if strings.Contains(v, "rel=xregistry-root") ||
			strings.Contains(v, "rel=\"xregistry-root\"") {
			return
		}
	}

	existingLinks = append(existingLinks, linkValue)

	// Clear all Link headers and re-add them with the updated value
	hw.DelHeader("Link")
	for _, v := range existingLinks {
		hw.AddHeader("Link", v)
	}
}

func ProcessShortSelf(info *registry.RequestInfo) *XRError {
	path := info.OriginalRequest.URL.Path
	if !strings.HasPrefix(path, "/r/") {
		return nil
	}

	ss, suffix, ok := strings.Cut(path[3:], "$")
	if ok {
		suffix = "$" + suffix
	}

	regUID, eXID, xErr := registry.MapShortSelf(info.Tx, ss)
	if xErr != nil {
		return xErr
	}

	regCollectionSegment := info.Tx.Config.GetAsString("path.regcollection")

	if regUID != "" && eXID != "" {
		// found it!
		// Note that eXID will always have a leading "/"
		newPath := "/" + regCollectionSegment + "/" + regUID + eXID + suffix

		log.FuncPrintf("tx: %s Redirect: %q -> %q", info.Tx.UUID, path, newPath)

		info.OriginalRequest.URL.Path = newPath
		return nil
	}

	return NewXRError("not_found", path)
}

type HttpWriter struct {
	Info       *registry.RequestInfo
	StatusCode int
	SentStatus bool
}

func NewHttpWriter(info *registry.RequestInfo) *HttpWriter {
	return &HttpWriter{
		Info:       info,
		StatusCode: 0,
		SentStatus: false,
	}
}

func (hw *HttpWriter) Reset() { // on each tx retry
	hw.StatusCode = 0
	hw.SentStatus = false
}

func (hw *HttpWriter) Write(b []byte) (int, error) {
	if !hw.SentStatus {
		// Set all response headers before writing status

		// CORS headers
		hw.AddHeader("Access-Control-Allow-Origin", "*")
		methods := hw.GetAllowedMethods()
		methodsStr := strings.Join(methods, ", ")
		hw.AddHeader("Access-Control-Allow-Methods", methodsStr)
		hw.AddHeader("Access-Control-Expose-Headers", "Link")

		// Reflect whatever headers the browser's preflight asked for
		// (e.g. Content-Type) so cross-origin PUT/PATCH/POST/DELETE
		// requests with a JSON body aren't blocked by CORS.
		reqHeaders := hw.Info.OriginalRequest.Header.Get(
			"Access-Control-Request-Headers")
		if reqHeaders != "" {
			hw.AddHeader("Access-Control-Allow-Headers", reqHeaders)
		} else {
			hw.AddHeader("Access-Control-Allow-Headers", "Content-Type")
		}

		// For OPTIONS requests, also set the Allow header
		if hw.Info.OriginalRequest.Method == "OPTIONS" {
			hw.AddHeader("Allow", methodsStr)
		}

		hw.AddRegistryRootHeader()

		hw.SentStatus = true
		if hw.StatusCode == 0 {
			hw.StatusCode = http.StatusOK
		}

		// If the user never set one, don't let golang add one
		if hw.GetResponseHeader("Content-Type") == "" {
			hw.Info.OriginalResponse.Header()["Content-Type"] = nil
		}

		hw.Info.OriginalResponse.WriteHeader(hw.StatusCode)
	}
	return hw.Info.OriginalResponse.Write(b)
}

func (hw *HttpWriter) DelHeader(name string) {
	hw.Info.OriginalResponse.Header().Del(name)
}

var stacks = map[string]string{}

func (hw *HttpWriter) SetHeader(name, value string) {
	// Make sure we don't add the same header more than once, that's a sign
	// we're doing something weong.
	// At some point we may need to add a new func (AddHeader) to append a
	// value to the end of the current one (if there)
	PanicIf(hw.Info.OriginalResponse.Header().Get(name) != "",
		"%s\nPrev:\n%s\n---", name, stacks[name])
	// Uncomment when we need to debug the PanicIf
	// stacks[name] = GetStackAsString()

	hw.Info.OriginalResponse.Header()[name] = []string{value}
}

func (hw *HttpWriter) AddHeader(name, value string) {
	hw.Info.OriginalResponse.Header().Add(name, value)
}

func (hw *HttpWriter) GetResponseHeader(name string) string {
	return hw.Info.OriginalResponse.Header().Get(name)
}

func (hw *HttpWriter) GetResponseHeaderValues(name string) []string {
	return hw.Info.OriginalResponse.Header()[name]
}

func (hw *HttpWriter) Done() {
	// If we haven't written anything, this will force the HTTP status code
	// to be written and not default to 200
	hw.Write(nil)
}

// GetAllowedMethods returns the list of HTTP methods allowed for the current
// request path based on capabilities
func (hw *HttpWriter) GetAllowedMethods() []string {
	methods := []string{}

	// Use original Parts/RootPath (they may have been modified during processing)
	parts := hw.Info.OriginalParts
	rootPath := hw.Info.OriginalRootPath
	numParts := len(parts)

	// Check special endpoints
	if rootPath == "capabilities" {
		if hw.Info.IsAvailable("capabilities") {
			methods = append(methods, "GET")
			if hw.Info.IsAvailableMutable("capabilities") {
				methods = append(methods, "PUT", "PATCH")
			}
		}
	} else if rootPath == "capabilitiesoffered" {
		if hw.Info.IsAvailable("capabilitiesoffered") {
			methods = append(methods, "GET")
		}
	} else if rootPath == "export" {
		if hw.Info.IsAvailable("export") {
			methods = append(methods, "GET")
		}
	} else if rootPath == "model" {
		if hw.Info.IsAvailable("model") {
			methods = append(methods, "GET")
		}
	} else if rootPath == "modelsource" {
		if hw.Info.IsAvailable("modelsource") {
			methods = append(methods, "GET")
			if hw.Info.IsAvailableMutable("modelsource") {
				methods = append(methods, "PUT")
			}
		}
	} else if rootPath == ".xregistry" {
		if hw.Info.IsAvailable(".xregistry") {
			methods = append(methods, "GET")
		}
	} else if hw.Info.IsAvailable("entities") {
		// Standard entity endpoints
		isMutable := hw.Info.IsAvailableMutable("entities")

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
