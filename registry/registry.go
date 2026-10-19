package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

// Default values for some server config properties
var DefAddr = ""
var DefPort = 8080
var DefDBHost = "127.0.0.1"
var DefDBPort = 3306
var DefDBName = "registry"
var DefDBUser = "root"
var DefDBPassword = "password"
var DefRegistryName = "xRegistry"

func NewXRServerConfig(fileName string) *Config {
	if fileName == "" {
		fileName = ".xrserver"
	}

	xrsConfig := NewConfig(fileName) // Doesn't load, just defines the file

	// First populate with Defaults values
	xrsConfig.Set("defaultreg", DefRegistryName)
	xrsConfig.Set("rootapp", "ui")
	xrsConfig.Set("ui.dir", "")
	xrsConfig.Set("ui.xrui.json", "")
	xrsConfig.Set("verbose", "")
	xrsConfig.Set("http.addr", DefAddr)
	xrsConfig.Set("http.port", fmt.Sprintf("%d", DefPort))
	xrsConfig.Set("db.name", DefDBName)
	xrsConfig.Set("db.host", DefDBHost)
	xrsConfig.Set("db.port", fmt.Sprintf("%d", DefDBPort))
	xrsConfig.Set("db.user", DefDBUser)
	xrsConfig.Set("db.password", DefDBPassword)
	xrsConfig.Set("path.ui", "ui")
	xrsConfig.Set("path.defaultreg", "xreg")
	xrsConfig.Set("path.regcollection", "xregs")

	return xrsConfig
}

func SetXRServerConfigFromEnvVars(xrsConfig *Config) *Config {
	xrsConfig.SetFromEnv("db.name", "DBNAME")
	xrsConfig.SetFromEnv("db.host", "DBHOST")
	xrsConfig.SetFromEnv("db.port", "DBPORT")
	xrsConfig.SetFromEnv("db.user", "DBUSER")
	xrsConfig.SetFromEnv("db.password", "DBPASSWORD")

	xrsConfig.SetFromEnv("http.addr", "XR_ADDR")
	xrsConfig.SetFromEnv("http.port", "XR_PORT")

	return xrsConfig
}

func (r *Registry) GetTx() *Tx {
	return r.Tx
}

func (r *Registry) GetXRServerConfig() *Config {
	PanicIf(r.Tx == nil, "Shouldn't be nil")
	return r.Tx.Config
}

func GetDefaultReg(tx *Tx) (*Registry, *XRError) {
	PanicIf(tx == nil, "Check me")

	regSID := tx.Config.GetAsString("DefaultRegistrySID")

	if regSID == "" {
		panic("No registry specified")
	}

	reg, xErr := FindRegistryBySID(tx, regSID, FOR_READ)
	if xErr != nil {
		return nil, xErr
	}
	PanicIf(reg == nil, "No default registry")

	if reg != nil {
		tx.Registry = reg
	}

	return reg, nil
}

func (r *Registry) Rollback() *XRError {
	if r != nil {
		return r.Tx.Rollback()
	}
	return nil
}

func (r *Registry) SaveAll() *XRError {
	if r != nil {
		if r.Model.GetChanged() {
			if xErr := r.SaveModel(false); xErr != nil {
				return xErr
			}
		}
		return r.Tx.SaveAll()
	}
	return nil
}

func (r *Registry) SaveAllAndCommit() *XRError {
	if r != nil {
		if r.Model.GetChanged() {
			if xErr := r.SaveModel(false); xErr != nil {
				return xErr
			}
		}
		return r.Tx.SaveAllAndCommit()
	}
	return nil
}

func (r *Registry) Refresh(accessMode int) *XRError {
	xErr := r.Entity.Refresh(accessMode)
	if xErr != nil {
		return xErr
	}

	if _, xErr := r.LoadCapabilities(); xErr != nil {
		return xErr
	}

	if _, xErr := r.LoadModel(); xErr != nil {
		return xErr
	}

	return nil
}

// ONLY CALL FROM TESTS - NEVER IN PROD
func (r *Registry) SaveCommitRefresh() *XRError {
	if r != nil {
		return r.Tx.SaveCommitRefresh()
	}
	return nil
}

// ONLY CALL FROM TESTS - NEVER IN PROD
func (r *Registry) AddToCache(e *Entity) {
	r.Tx.AddToCache(e)
}

func (r *Registry) Commit() *XRError {
	if r != nil {
		return r.Tx.Commit()
	}
	return nil
}

// Validate drains this Registry's Tx of any deferred Resource/Group
// validation work (Tx.ResourcesToValidate/GroupsToValidate) - this is
// registry-wide constraint/cascade checking, so it belongs here rather
// than in the Tx layer, even though the pending-work lists themselves
// are Tx-scoped bookkeeping (see those fields' doc comments in db.go).
// Called from Tx.Validate() once the Tx's own bookkeeping/sanity-checks
// are done.
func (r *Registry) Validate() *XRError {
	tx := r.Tx

	// Drain any Resources marked for (re-)validation - possibly marked
	// several times each (see AddResourceToValidate()'s doc comment).
	// Loops until the map is empty since running ValidateResource() can,
	// in theory, cause further marks (e.g. Resource.EnsureLatest() saving
	// a Meta), with a safety cap to avoid an infinite loop should that
	// ever cycle unexpectedly. Must run before GroupsToValidate since
	// constraint checks may depend on cascade-derived mirrored data
	// (e.g. isdefault).
	for i := 0; len(tx.ResourcesToValidate) > 0; i++ {
		if i >= 20 {
			return NewXRError("server_error", "/").SetDetail(
				"Resource validation worklist did not converge.")
		}
		batch := tx.ResourcesToValidate
		tx.ResourcesToValidate = map[string]*resourceValidation{}

		tx.ResourcesValidatingBatch = map[string]bool{}
		for sid := range batch {
			tx.ResourcesValidatingBatch[sid] = true
		}

		for _, entry := range batch {
			if xErr := entry.r.ValidateResource(entry.onlyMetaChanged,
				entry.force); xErr != nil {
				tx.ResourcesValidatingBatch = nil
				return xErr
			}
		}
		tx.ResourcesValidatingBatch = nil
	}

	for _, g := range tx.GroupsToValidate {
		if xErr := g.Validate(); xErr != nil {
			return xErr
		}
	}

	return nil
}

func GetRegistryNames(xrsConfig *Config) ([]string, *XRError) {
	be, xErr := NewDefaultBackend(xrsConfig)
	if xErr != nil {
		return nil, xErr
	}

	tx, xErr := NewTx(NewUUID(), xrsConfig, be)
	if xErr != nil {
		return nil, xErr
	}

	defer tx.Rollback()

	return tx.Backend.ListRegistries(tx)
}

var _ EntitySetter = &Registry{}

func (reg *Registry) Get(name string) any {
	return reg.Entity.Get(name)
}

func (reg *Registry) JustSet(name string, val any) *XRError {
	// Normally we should never call Lock() directly, however Registry is
	// kind of special because we rarely know if we want to "Find" the Registry
	// for writing until later in the process. So instead of forcing the
	// code to re-Find with FOR_WRITE, we'll just make it easy and these
	// variants of 'update' will just lock it automatically
	if _, xErr := reg.Lock(); xErr != nil {
		return xErr
	}
	return reg.Entity.eJustSetPath(name, val)
}

func (reg *Registry) Delete() *XRError {
	defer log.Trace("tx: %s %s", reg.Tx.UUID, reg.UID)()

	// Normally we should never call Lock() directly, however Registry is
	// kind of special because we rarely know if we want to "Find" the Registry
	// for writing until later in the process. So instead of forcing the
	// code to re-Find with FOR_WRITE, we'll just make it easy and these
	// variants of 'update'  will just lock it automatically
	if _, xErr := reg.Lock(); xErr != nil {
		return xErr
	}

	if xErr := reg.Tx.Backend.DeleteRegistry(reg); xErr != nil {
		return xErr
	}

	// Avoid an unbounded leak of dead entries in the in-process Model
	// cache (see registry/model.go) - the SID is never reused, but
	// there's no reason to hang on to it once the Registry is gone.
	EvictCachedModel(reg.SID)

	// Delete any pending changes so dirty check doesn't fail
	reg.NewObject = nil
	reg.Tx.EraseCache()
	reg.Tx.Registry = nil

	return nil
}

func (reg *Registry) LoadModelFromFile(file string) *XRError {
	defer log.Trace("tx: %s %s", reg.Tx.UUID, file)()

	var xErr *XRError
	var err error
	buf := []byte{}
	if strings.HasPrefix(file, "http") {
		res, err := http.Get(file)
		if err == nil {
			buf, err = io.ReadAll(res.Body)
			res.Body.Close()

			if res.StatusCode/100 != 2 {
				return NewXRError("parsing_data", file,
					"error_detail="+
						fmt.Sprintf("error getting model from %q: %s\n%s",
							file, res.Status, string(buf)))
			}
		}
	} else {
		buf, err = os.ReadFile(file)
	}
	if err != nil {
		return NewXRError("parsing_data", file,
			"error_detail="+
				fmt.Sprintf("processing %q: %s", file, err))
	}

	buf, xErr = ProcessIncludes(file, buf, true)
	if xErr != nil {
		return xErr
	}

	model, xErr := ParseModel(buf, reg)
	if xErr != nil {
		return NewXRError("parsing_data", file,
			"error_detail="+
				fmt.Sprintf("processing %q: %s", file, xErr))
	}

	model.Registry = reg
	if xErr = model.Verify(); xErr != nil {
		return xErr
	}

	if xErr = reg.SetModel(model, string(buf), true); xErr != nil {
		return xErr
	}

	// reg.Model = model
	// reg.Model.VerifyAndSave()
	return nil
}

func (reg *Registry) Update(obj Object, addType AddType) *XRError {
	defer log.Trace("tx: %s", reg.Tx.UUID)()

	if xErr := CheckAttrs(obj, reg.XID); xErr != nil {
		return xErr
	}

	reg.RemoveReadOnlyImmutable(obj)

	// Normally we should never call Lock() directly, however Registry is
	// kind of special because we rarely know if we want to "Find" the Registry
	// for writing until later in the process. So instead of forcing the
	// code to re-Find with FOR_WRITE, we'll just make it easy and these
	// variants of 'update'  will just lock it automatically
	if _, xErr := reg.Lock(); xErr != nil {
		return xErr
	}

	reg.SetNewObject(obj)

	// Ignore any incoming "model" attribute
	delete(reg.NewObject, "model")

	info := reg.GetRequestInfo()

	if info.HasIgnore("capabilities") && !IsNil(reg.NewObject) {
		delete(reg.NewObject, "capabilities")
	}

	if info.HasIgnore("modelsource") && !IsNil(reg.NewObject) {
		delete(reg.NewObject, "modelsource")
	}

	// Need to do it here instead of under the checkFn because doing it
	// in checkfn causes a circular reference that golang doesn't like
	val, ok := reg.NewObject["modelsource"]
	if ok {
		// Notice that "null" means erase it, not "keep it as is"
		var rawJson []byte

		if IsNil(val) {
			rawJson = []byte("{}")
		} else {
			var err error
			rawJson = val.(json.RawMessage)
			rawJson, err = RemoveSchema(rawJson)
			if err != nil {
				return NewXRError("bad_request", "/",
					"error_detail="+err.Error())
			}
		}

		xErr := reg.SetModel(nil, string(rawJson), false)
		if xErr != nil {
			return xErr
		}

		delete(reg.NewObject, "modelsource")
	}

	// Make sure we always have an ID
	if IsNil(reg.NewObject["registryid"]) {
		reg.EnsureNewObject()
		reg.NewObject["registryid"] = reg.UID
	}

	// Remove/save all Registry level collections from NewObject
	collsMaps := map[string]map[string]any{}
	for _, coll := range reg.GetCollections() {
		plural := coll[0]
		singular := coll[1]

		collVal := obj[plural]
		if IsNil(collVal) {
			continue
		}
		collMap, ok := collVal.(map[string]any)
		if !ok {
			return NewXRError("invalid_attribute", "/",
				"name="+plural,
				"error_detail="+fmt.Sprintf("doesn't appear to be of a "+
					"map of %q", plural))
		}
		for key, val := range collMap {
			_, ok := val.(map[string]any)
			if !ok {
				return NewXRError("invalid_attribute", "/",
					"name="+plural,
					"error_detail="+
						fmt.Sprintf("key %q doesn't appear to be of type %q",
							key, singular))
			}
		}

		// Remove the Groups Collections attributes from the incoming obj
		collsMaps[plural] = collMap
		delete(reg.NewObject, plural)
		delete(reg.NewObject, plural+"count")
		delete(reg.NewObject, plural+"url")
	}

	// For each collection, upsert each entity
	for plural, collMap := range collsMaps {
		for key, val := range collMap {
			valObj, _ := val.(map[string]any)
			_, _, xErr := reg.UpsertGroupWithObject(plural, key, valObj,
				addType)
			if xErr != nil {
				return xErr
			}
		}
	}

	reg.EnsureNewObject()
	if addType == ADD_PATCH {
		// Copy existing props over if the incoming obj doesn't set them
		for k, val := range reg.Object {
			if _, ok := reg.NewObject[k]; !ok {
				reg.NewObject[k] = val
			}
		}
	}

	// Save the registry attributes first (this applies reg.NewObject)
	if xErr := reg.ValidateAndSave(false); xErr != nil {
		return xErr
	}

	// If the model changed (tracked via SetStuff in ProcessCapabilitiesModelSource),
	// verify all data. We can't use Model.VerifyAndSave() because the model was
	// already saved in ProcessCapabilitiesModelSource and the changed flag is false.
	if val, ok := reg.GetStuff("modelchanged"); ok && val == true {
		if xErr := reg.VerifyData(); xErr != nil {
			return xErr
		}
	} else {
		// Normal path: save model if it changed
		if xErr := reg.Model.VerifyAndSave(true); xErr != nil {
			return xErr
		}
	}

	return nil
}

// FindGroup, unlike Resource's FindMeta/FindVersion, keeps its own
// explicit accessMode arg rather than deriving it from a parent's lock
// state - Group/Registry are NOT part of a lockEntityFamily() "family"
// the way Resource+Meta+Version are, and don't need to be: a Group's own
// row (epoch/modifiedat) is only ever mutated via an explicit Touch()/
// Lock() call made right at the point of change (e.g. UpsertResource()
// and Resource.Delete() both call g.Touch() when a child Resource is
// added/removed, since that bumps the owning Group's epoch/modifiedat),
// not by relying on whatever accessMode an earlier FindGroup() call used.
// So a caller that fetches a Group FOR_READ and later needs to mutate it
// is always safe as long as the actual mutation path itself locks first
// (which every Group-mutating call site already does). Contrast this
// with the bug class Resource/Meta/Version's family-locking exists to
// prevent, where a decision (e.g. "does this Resource already exist?")
// computed from an unlocked read could go stale before the write.
func (reg *Registry) FindGroup(gType string, id string, anyCase bool, accessMode int) (*Group, *XRError) {
	defer log.Trace("tx: %s %s,%s,%v", reg.Tx.UUID, gType, id, anyCase)()

	if g := reg.Tx.GetGroup(reg, gType, id); g != nil {
		log.FuncPrintf("tx: %s FindGroup %s,%s from cache",
			reg.Tx.UUID, gType, id)
		if accessMode == FOR_WRITE && g.AccessMode != FOR_WRITE {
			if _, xErr := g.Lock(); xErr != nil {
				return nil, xErr
			}
		}
		return g, nil
	}

	entity, xErr := reg.Tx.Backend.GetEntityFromXID(reg.Tx, reg.SID,
		"/"+gType+"/"+id, false, anyCase, accessMode)
	if xErr != nil {
		return nil, xErr
	}

	if entity == nil {
		log.FuncPrintf("tx: %s None found", reg.Tx.UUID)
		return nil, nil
	}

	g := &Group{Entity: *entity, Registry: reg}
	g.Self = g
	g.Tx.AddGroup(g)
	return g, nil
}

func (reg *Registry) AddGroup(gType string, id string) (*Group, *XRError) {
	g, _, xErr := reg.UpsertGroupWithObject(gType, id, nil, ADD_ADD)
	return g, xErr
}

func (reg *Registry) AddGroupWithObject(gType string, id string, obj Object) (*Group, *XRError) {
	g, _, xErr := reg.UpsertGroupWithObject(gType, id, obj, ADD_ADD)
	return g, xErr
}

// *Group, isNew, error
func (reg *Registry) UpsertGroup(gType string, id string) (*Group, bool, *XRError) {
	return reg.UpsertGroupWithObject(gType, id, nil, ADD_UPSERT)
}

func (reg *Registry) UpsertGroupWithObject(gType string, id string, obj Object, addType AddType) (*Group, bool, *XRError) {
	defer log.Trace("tx: %s %s,%s", reg.Tx.UUID, gType, id)()

	// Move to below, after the "findGroup". Is SaveModel needs it then
	// it'll lock it.
	// _, xErr := reg.Lock()

	if xErr := reg.SaveModel(false); xErr != nil {
		return nil, false, xErr
	}

	if xErr := CheckAttrs(obj, "/"+gType+"/"+id); xErr != nil {
		return nil, false, xErr
	}

	gm := reg.Model.Groups[gType]
	if gm == nil {
		return nil, false, NewXRError("unknown_group_type", "/"+gType,
			"name="+gType)
	}

	if id == "" {
		id = NewUUID()
	}

	isNew := false

	g, xErr := reg.FindGroup(gType, id, true, FOR_WRITE)
	if xErr != nil {
		return nil, false, xErr
	}

	if g == nil {
		// If the group doesn't exist yet then, if the Registry isn't already
		// locked, lock it before we try to create the group.
		// We don't lock it before this because if the group already
		// exists then there should be no reason to touch the Registry
		if _, xErr := reg.Lock(); xErr != nil {
			return nil, false, xErr
		}

		g, xErr = reg.FindGroup(gType, id, true, FOR_WRITE)
		if xErr != nil {
			return nil, false, xErr
		}
	}

	log.FuncPrintf("tx: %s upsertGroup FindGroup(%s,%s,true) => %v",
		reg.Tx.UUID, gType, id, g != nil)

	if g != nil && g.UID != id {
		return nil, false,
			NewXRError("bad_request", "/"+g.UID,
				"error_detail="+
					fmt.Sprintf("Attempting to create a Group "+
						"with a \"%sid\" of %q, when one already exists as %q",
						gm.Singular, id, g.UID))
	}

	if addType == ADD_ADD && g != nil {
		return nil, false,
			NewXRError("bad_request", "/"+id,
				"error_detail="+
					fmt.Sprintf("Group %q of type %q already exists",
						id, gType))
	}

	for g == nil {
		// Not found, so create a new one
		g = &Group{
			Entity: Entity{
				Tx:         reg.Tx,
				AccessMode: FOR_WRITE,
				Registry:   reg,
				SID:        NewUUID(),
				ParentSID:  reg.SID,
				Plural:     gType,
				Singular:   gm.Singular,
				UID:        id,

				Type:     ENTITY_GROUP,
				XID:      "/" + gType + "/" + id,
				Abstract: gType,

				GroupModel: gm,
			},
			Registry: reg,
		}
		g.Self = g

		if xErr = g.Tx.Backend.RegisterEntity(&g.Entity); xErr != nil {
			return nil, false, xErr
		}

		// Use the ID passed as an arg, not from the metadata, as the true
		// ID. If the one in the metadata differs we'll flag it down below
		if xErr = g.JustSet(g.Singular+"id", g.UID); xErr != nil {
			return nil, false, xErr
		}
		isNew = true
		if _, xErr := g.Registry.Touch(); xErr != nil {
			return nil, false, xErr
		}
		g.Tx.AddGroup(g)
	}

	g.RemoveReadOnlyImmutable(obj)

	// Remove all Resource collections from obj before we process it
	objColls := map[string]map[string]any{}
	for _, coll := range g.GetCollections() {
		plural := coll[0]
		singular := coll[1]

		collVal := obj[plural]
		if IsNil(collVal) {
			continue
		}

		collMap, ok := collVal.(map[string]any)
		if !ok {
			return nil, false,
				NewXRError("invalid_attribute", "/"+gType+"/"+id,
					"name="+plural,
					"error_detail="+
						fmt.Sprintf("doesn't appear to be of a "+
							"map of %q", plural))
		}
		for key, val := range collMap {
			_, ok := val.(map[string]any)
			if !ok {
				return nil, false,
					NewXRError("invalid_attribute", g.XID,
						"name="+plural,
						"error_detail="+
							fmt.Sprintf("Key %q doesn't appear to be of "+
								"type %q", key, singular))
			}
		}

		objColls[plural] = collMap
		delete(obj, plural)
		delete(obj, plural+"count")
		delete(obj, plural+"url")
	}

	if isNew || obj != nil {
		if obj != nil {
			g.SetNewObject(obj)
		}

		if addType == ADD_PATCH {
			// Copy existing props over if the incoming obj doesn't set them
			for k, v := range g.Object {
				if _, ok := g.NewObject[k]; !ok {
					g.NewObject[k] = v
				}
			}
		}

		// Make sure we always have an ID
		if IsNil(g.NewObject[g.Singular+"id"]) {
			g.NewObject[g.Singular+"id"] = g.UID
		}

		if xErr = g.ValidateAndSave(false); xErr != nil {
			return nil, false, xErr
		}
	}

	// Now for each inlined Resource collection, upsert each Resource
	for plural, daMap := range objColls {
		for key, val := range daMap {
			valObj, _ := val.(map[string]any)
			_, _, xErr := g.UpsertResource(&ResourceUpsert{
				RType:            plural,
				Id:               key,
				VID:              "",
				Obj:              valObj,
				AddType:          addType,
				ObjIsVer:         false,
				DefaultVersionID: "",
			})

			if xErr != nil {
				return nil, false, xErr
			}
		}
	}

	if xErr = reg.ValidateAndSave(false); xErr != nil {
		return nil, false, xErr
	}

	return g, isNew, nil
}

// Returns a map of groupType->*Group
func (reg *Registry) UpsertJustGroups(rootObj Object, addType AddType) (map[string][]*Group, *XRError) {
	defer log.Trace("tx: %s", reg.Tx.UUID)()

	groups := map[string][]*Group{}

	// Just for nicer error msg than what IncomingObj2Map would produce
	for attrName, _ := range rootObj {
		if reg.Model.FindGroupModel(attrName) == nil {
			return nil, NewXRError("groups_only", "/", "name="+attrName)
		}
	}

	rootMap, xErr := IncomingObj2Map(rootObj, "/", "map of Group types")
	if xErr != nil {
		return nil, xErr
	}

	// rootMap is map[groupType]group
	for gType, gAny := range rootMap {
		/*
			if reg.Model.FindGroupModel(gType) == nil {
				return nil, NewXRError("groups_only", "/", "name="+gType)
			}
		*/

		gMap, xErr := IncomingObj2Map(gAny, "/",
			"\""+reg.Model.Groups[gType].Singular+"\"")
		if xErr != nil {
			return nil, xErr
		}

		// Make sure we include empty groupTypes in the result
		groups[gType] = []*Group{}

		// gMap is maps[gID]group
		for id, gObj := range gMap {
			g, _, xErr := reg.UpsertGroupWithObject(gType, id, gObj, addType)
			if xErr != nil {
				return nil, xErr
			}
			groups[gType] = append(groups[gType], g)
		}
	}

	return groups, nil
}

func (r *Registry) XID2Entity(xidStr string, path string) (*Entity, *XRError) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+err.Error())
	}

	g, xErr := r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
	if xErr != nil {
		return nil, xErr
	}
	if g == nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+fmt.Sprintf("cant find Group %q", xid.Group))
	}
	if xid.Type == ENTITY_GROUP {
		return &g.Entity, nil
	}

	if xid.IsEntity == false || xid.Type == ENTITY_META {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+fmt.Sprintf("%q isn't an xid", xidStr))
	}

	res, xErr := g.FindResource(xid.Resource, xid.ResourceID, false, FOR_READ)
	if xErr != nil {
		return nil, xErr
	}

	if res == nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+fmt.Sprintf("unknown Resource %q", xid.Resource))
	}
	if xid.Type == ENTITY_RESOURCE {
		return &res.Entity, nil
	}

	v, xErr := res.FindVersion(xid.VersionID, false)
	if xErr != nil {
		return nil, xErr
	}
	if v == nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+fmt.Sprintf("unknown Version %q", xid.VersionID))
	}
	if xid.Type == ENTITY_VERSION {
		return &v.Entity, nil
	}

	return nil, NewXRError("malformed_xid", path,
		"xid="+xidStr,
		"error_detail=not a valid XID")
}

func (r *Registry) FindXIDGroup(xidStr string, path string) (*Group, *XRError) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+err.Error())
	}
	if xid.GroupID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"groupid\"")
	}

	return r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
}

func (r *Registry) FindResourceByXID(xidStr string, path string, accessMode int) (*Resource, *XRError) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+err.Error())
	}
	if xid.GroupID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"groupid\"")
	}
	if xid.ResourceID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"resourceid\"")
	}
	g, xErr := r.FindGroup(xid.Group, xid.GroupID, false, accessMode)
	if xErr != nil || g == nil {
		return nil, xErr
	}
	return g.FindResource(xid.Resource, xid.ResourceID, false, accessMode)
}

func (r *Registry) FindXIDVersion(xidStr string, path string) (*Version, *XRError) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+err.Error())
	}
	if xid.GroupID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"groupid\"")
	}
	if xid.ResourceID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"resourceid\"")
	}
	if xid.VersionID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"versionid\"")
	}
	if xid.Version != "versions" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=not a \"versions\" XID")
	}
	g, xErr := r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
	if xErr != nil || g == nil {
		return nil, xErr
	}
	resource, xErr := g.FindResource(xid.Resource, xid.ResourceID, false,
		FOR_READ)
	if xErr != nil || resource == nil {
		return nil, xErr
	}
	return resource.FindVersion(xid.VersionID, false)
}

func (r *Registry) FindXIDMeta(xidStr string, path string) (*Meta, *XRError) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+err.Error())
	}
	if xid.GroupID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"groupid\"")
	}
	if xid.ResourceID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"resourceid\"")
	}
	if xid.Version != "meta" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=not a \"meta\" XID")
	}
	g, xErr := r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
	if !IsNil(err) || g == nil {
		return nil, xErr
	}
	resource, xErr := g.FindResource(xid.Resource, xid.ResourceID, false,
		FOR_READ)
	if xErr != nil || resource == nil {
		return nil, xErr
	}
	return resource.FindMeta(false)
}

/*
func LoadRemoteRegistry(host string) (*Registry, *XRError) {
	reg := &Registry{}

	rootObj := map[string]json.RawMessage(nil)

	// Download model
	data, err := DownloadURL(host + "/model")
	if err != nil {
		data, err2 := DownloadURL(host + "?inline=model,capabilities")
		if err2 != nil {
			return nil, NewXRError("bad_request",
				host+"?inline=model,capabilities",
				"error_detail="+
					fmt.Sprintf("Error getting (%s/model: %s", host, err))
		}

		err = json.Unmarshal(data, &rootObj)
		if err != nil {
			return nil, NewXRError("parsing_response", host,
				"error_detail="+
					fmt.Sprintf("Error parsing Registry root(%s): %s", host,
						err))
		}
		tmp, ok := rootObj["model"]
		if !ok {
			return nil, NewXRError("bad_request", host,
				"error_detail="+
					fmt.Sprintf("Unable to find the model for %q from either "+
						"/model or the \"model\" attribute", host))
		}
		data = []byte(tmp)
	}

	var xErr *XRError
	reg.Model, xErr = ParseModel(data, reg)
	if xErr != nil {
		return nil, xErr.SetDetail("Model:\n" + string(data) + ".")
	}

	// Download capabilities
	data, err = DownloadURL(host + "/capabilities")
	if err != nil {
		if rootObj == nil {
			data, err2 := DownloadURL(host + "?inline=capabilities")
			if err2 != nil {
				return nil, NewXRError("bad_request",
					host+"?inline=capabilities",
					"error_detail="+
						fmt.Sprintf("Error getting capabilities "+
							"(%s/capabilities): %s", host, err))
			}

			err = json.Unmarshal(data, &rootObj)
			if err != nil {
				return nil, NewXRError("parsing_response", host,
					"error_detail="+
						fmt.Sprintf("Error parsing Registry root(%s): %s", host,
							err))
			}
		}
		tmp, ok := rootObj["capabilities"]
		if !ok {
			return nil, NewXRError("bad_request", host,
				"error_detail="+
					fmt.Sprintf("Unable to find the capabilities for %q "+
						"from either /capabilities or the \"capabilities\" "+
						"attribute", host))
		}
		data = []byte(tmp)
	}

	reg.Capabilities, xErr = ParseCapabilities(data)
	if xErr != nil {
		return nil, xErr
	}

	reg.oldCapabilities = reg.Capabilities.Clone()

	return reg, nil
}
*/

func (r *Registry) VerifyData() *XRError {
	// Start with the Registry itself
	if xErr := r.ValidateAndSave(true); xErr != nil {
		return xErr
	}

	// Now do all Groups
	groups, xErr := r.GetGroups(FOR_WRITE)
	if xErr != nil {
		return xErr
	}

	for _, group := range groups {
		if xErr = group.ValidateAndSave(true); xErr != nil {
			return xErr
		}

		// Now do all Resource in this Group and implicitly it's owning Meta
		resources, xErr := group.GetResources(FOR_WRITE)
		if xErr != nil {
			return xErr
		}

		for _, resource := range resources {
			// onlyMetaChanged, forceVerify
			//
			// Deferred (not run synchronously here) so it goes through
			// the same batching/merging logic as every other
			// AddResourceToValidate() call site - r.Validate(nil) below
			// drains it (along with GroupsToValidate) once all Resources
			// and Versions in this loop have been marked/processed.
			r.Tx.AddResourceToValidate(resource, false, true)

			// Skip xref'd resources, the real owner will do it.
			// Note that we're assuming we're just skipping data validation.
			// If we ever addd other checks (like version-level constraints)
			// then we may need to do the stuff below. However, as of now
			// any group-level or resource-level constraints should be done
			// above.
			// But we do need to check the non-xref'd versions on a model
			// change to ensure they're still compliant
			if resource.IsXref() {
				continue
			}

			// Now do Versions. Filter to just ENTITY_VERSION - a
			// Resource's Meta shares the same ParentSID
			// (resource.SID), so without this filter we'd also pick
			// up the Meta row here and mis-process it as a Version.
			versions, xErr := resource.GetVersions(FOR_WRITE)
			if xErr != nil {
				return xErr
			}

			for _, version := range versions {
				if xErr = version.ValidateAndSave(true); xErr != nil {
					return xErr
				}
			}
		}
	}

	// Drain any deferred Resource validation (marked just above via
	// AddResourceToValidate()) and Group constraint checks
	// (GroupsToValidate) - this is the original gap this session set
	// out to fix: VerifyData() never checked group-level equals/enum
	// cross-attribute constraints on model changes, and (as of this
	// change) no longer runs ValidateResource() synchronously either,
	// so both are drained together via the same Registry.Validate()
	// used by every HTTP request (Tx.Validate()). Must run last, after
	// all per-Version attribute validation above, so a more specific
	// per-attribute error isn't masked by a more generic one for the
	// same violation.
	if xErr := r.Validate(); xErr != nil {
		return xErr
	}

	return nil
}

func (r *Registry) GenerateResultEntities(what string,
	XIDs []string) (EntityIterator, *XRError) {

	return r.Tx.Backend.GenerateResultEntities(r, what, XIDs)
}

func (r *Registry) GetEntitiesFromXID(xidStr string, allAttrs bool,
	children bool, anyCase bool, accessMode int) (EntityIterator, *XRError) {

	return r.Tx.Backend.GetEntitiesFromXID(r.Tx, r.SID, xidStr, allAttrs,
		children, anyCase, accessMode)
}

func (r *Registry) GetEntityFromXID(xidStr string, allAttrs bool, anyCase bool,
	accessMode int) (*Entity, *XRError) {

	return r.Tx.Backend.GetEntityFromXID(r.Tx, r.SID, xidStr, allAttrs,
		anyCase, accessMode)
}

func (r *Registry) GetGroupIDsByType(gType string,
	accessMode int) ([]string, *XRError) {

	return r.Tx.Backend.GetGroupIDsByType(r, gType, accessMode)
}

func MapShortSelf(tx *Tx, ss string) (string, string, *XRError) {
	return tx.Backend.MapShortSelf(tx, ss)
}
