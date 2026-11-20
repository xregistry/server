package memory

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

// A pure in-memory implementation of registry.Backend. Nothing is persisted,
// all data lives in one process-wide store.

func init() {
	xErr := registry.RegisterBackend("memory", NewMemoryBackend)
	if xErr != nil {
		panic(xErr.String() + xErr.Detail)
	}
}

type MemoryBackend struct {
	XRSConfig *Config
}

var _ registry.Backend = &MemoryBackend{}

// The one (implicit) store of this process. Reset() swaps in a fresh one.
var (
	storeMutex sync.Mutex
	store      = newMemDB("memory")
)

func getStore() *memDB {
	storeMutex.Lock()
	defer storeMutex.Unlock()
	return store
}

// Reset discards all of the data, e.g. between tests
func Reset() {
	storeMutex.Lock()
	defer storeMutex.Unlock()
	store = newMemDB("memory")
}

func NewMemoryBackend(c *Config) (registry.Backend, *XRError) {
	return &MemoryBackend{XRSConfig: c}, nil
}

// DB Actions - this backend has no notion of a DB

func (be *MemoryBackend) IsReady(c *Config) *XRError {
	return nil
}

func (be *MemoryBackend) SupportsDBs() bool { return false }

func notSupported() *XRError {
	return NewXRError("server_error", "/").
		SetDetail("Backend \"memory\" doesn't support DBs.")
}

func (be *MemoryBackend) DBExists(c *Config, name string) (bool, *XRError) {
	return false, nil
}

func (be *MemoryBackend) DBCreate(c *Config, name string) *XRError {
	return notSupported()
}

func (be *MemoryBackend) DBDelete(c *Config, name string) *XRError {
	return notSupported()
}

func (be *MemoryBackend) DBList(c *Config) ([]string, *XRError) {
	return nil, notSupported()
}

func (be *MemoryBackend) DBConnectionString(c *Config) (string, *XRError) {
	return "", nil
}

func (be *MemoryBackend) Close(c *Config) *XRError {
	return nil
}

// TX Actions

// It's ok for this to be called multiple times for the same Tx
func (be *MemoryBackend) NewTx(tx *registry.Tx) *XRError {
	if tx.Tx != nil {
		return nil
	}

	db := getStore()

	if xErr := db.acquire(); xErr != nil {
		return xErr
	}

	tx.Tx = &memTx{db: db}
	log.VPrintf(3, "tx: %s Begin in-memory transaction", tx.UUID)
	return nil
}

func (be *MemoryBackend) Commit(tx *registry.Tx) *XRError {
	mtx, ok := tx.Tx.(*memTx)
	if !ok || mtx == nil {
		return nil
	}
	mtx.undo = nil
	mtx.db.release()
	tx.Tx = nil
	return nil
}

func (be *MemoryBackend) Rollback(tx *registry.Tx) *XRError {
	mtx, ok := tx.Tx.(*memTx)
	if !ok || mtx == nil {
		return nil
	}
	mtx.rollback()
	mtx.db.release()
	tx.Tx = nil
	return nil
}

// mt returns the in-memory Tx, (re)opening it if needed
func mt(tx *registry.Tx) (*memTx, *XRError) {
	if xErr := tx.EnsureTx(); xErr != nil {
		return nil, xErr
	}
	return tx.Tx.(*memTx), nil
}

func mustMT(tx *registry.Tx) *memTx {
	m, xErr := mt(tx)
	if xErr != nil {
		panic(xErr.String() + xErr.Detail)
	}
	return m
}

// Model stuff

func (be *MemoryBackend) ClearResourceModelSystemProps(rm *registry.ResourceModel, props []string) *XRError {
	if len(props) == 0 {
		return nil
	}

	reg := rm.GroupModel.Model.Registry
	tx, xErr := mt(reg.Tx)
	if xErr != nil {
		return xErr
	}

	names := map[string]bool{}
	for _, name := range props {
		names[strings.ToLower(name+string(DB_IN))] = true
	}

	clear := func(sid string) {
		for lname := range names {
			tx.delProp(sid, lname)
		}
	}

	for _, r := range tx.db.resources {
		if r.RegSID != reg.SID || r.ModelSID != rm.SID {
			continue
		}
		clear(r.SID)
		for _, e := range tx.childEnts(r.SID) {
			if tx.db.versions[e.SID] != nil {
				clear(e.SID)
			}
		}
	}
	return nil
}

func (be *MemoryBackend) LoadCapabilities(r *registry.Registry) (*Capabilities, *XRError) {
	capVal, ok := r.Object["#capabilities"]
	if !ok {
		return DefaultCapabilities.Clone(), nil
	}

	capStr, ok := capVal.(string)
	PanicIf(!ok, "not a byte array: %T", capVal)
	cap, xErr := ParseCapabilities([]byte(capStr))
	if xErr != nil {
		return nil, xErr
	}

	return cap, nil
}

func (be *MemoryBackend) SaveCapabilities(r *registry.Registry) *XRError {
	return r.SetSave("#capabilities", ToJSON(r.Capabilities))
}

func (be *MemoryBackend) LoadModel(r *registry.Registry) (*registry.Model, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return nil, xErr
	}

	row := tx.db.models[r.SID]
	if row == nil || !row.HasModel || row.Model == "" {
		return nil, nil
	}

	var model *registry.Model
	if model = registry.GetCachedModel(r.SID, row.UUID); model == nil {
		model, xErr = registry.ParseModel([]byte(row.Model), r)
		if xErr != nil {
			return nil, xErr
		}
		registry.SetCachedModel(r.SID, row.UUID, model)
	}

	model.Registry = r
	return model, nil
}

func (be *MemoryBackend) LoadModelSource(r *registry.Registry) ([]byte, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return nil, xErr
	}

	row := tx.db.models[r.SID]
	if row == nil {
		return nil, nil
	}
	if !row.HasSource {
		return []byte{}, nil
	}
	return append([]byte{}, row.Source...), nil
}

func (tx *memTx) modelRowFor(regSID string) *modelRow {
	if old := tx.db.models[regSID]; old != nil {
		n := *old
		return &n
	}
	return &modelRow{RegSID: regSID}
}

func (be *MemoryBackend) SaveModel(r *registry.Registry) *XRError {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return xErr
	}

	buf, _ := json.Marshal(r.Model)
	uuid := NewUUID()

	row := tx.modelRowFor(r.SID)
	row.Model = string(buf)
	row.HasModel = true
	row.UUID = uuid
	putRow(tx, tx.db.models, r.SID, row)

	r.Model.SetStuff("changedUUID", uuid)
	return nil
}

func (be *MemoryBackend) SaveModelSource(r *registry.Registry) *XRError {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return xErr
	}

	row := tx.modelRowFor(r.SID)
	row.Source = append([]byte(nil), r.ModelSource...)
	row.HasSource = r.ModelSource != nil
	putRow(tx, tx.db.models, r.SID, row)
	return nil
}

func (be *MemoryBackend) RegisterGroupModel(gm *registry.GroupModel) *XRError {
	reg := gm.Model.Registry
	tx, xErr := mt(reg.Tx)
	if xErr != nil {
		return xErr
	}

	return tx.addModelEnt(&modelEntRow{
		SID: gm.SID, RegSID: reg.SID, ParentSID: "",
		Abstract: "/" + gm.Plural, Plural: gm.Plural, Singular: gm.Singular,
	})
}

func (be *MemoryBackend) RegisterResourceModel(rm *registry.ResourceModel) *XRError {
	gm := rm.GroupModel
	reg := gm.Model.Registry
	tx, xErr := mt(reg.Tx)
	if xErr != nil {
		return xErr
	}

	return tx.addModelEnt(&modelEntRow{
		SID: rm.SID, RegSID: reg.SID, ParentSID: gm.SID,
		Abstract: "/" + gm.Plural + "/" + rm.Plural,
		Plural:   rm.Plural, Singular: rm.Singular,
	})
}

func (tx *memTx) addModelEnt(me *modelEntRow) *XRError {
	for _, o := range tx.db.modelEnts {
		if o.SID == me.SID {
			return NewXRError("server_error", "/").
				SetDetailf("Duplicate ModelEntity SID: %s.", me.SID)
		}
		if o.RegSID == me.RegSID && (o.Abstract == me.Abstract ||
			(o.ParentSID == me.ParentSID &&
				(strings.EqualFold(o.Plural, me.Plural) ||
					strings.EqualFold(o.Singular, me.Singular)))) {
			return NewXRError("server_error", "/").
				SetDetailf("Duplicate ModelEntity: %s.", me.Abstract)
		}
	}
	putRow(tx, tx.db.modelEnts, me.SID, me)
	return nil
}

func (be *MemoryBackend) DeleteModelEnityByAbstract(reg *registry.Registry, abstract string) *XRError {
	tx, xErr := mt(reg.Tx)
	if xErr != nil {
		return xErr
	}

	for _, me := range tx.db.modelEnts {
		if me.RegSID == reg.SID && me.Abstract == abstract {
			tx.deleteModelEntity(me.SID)
			return nil
		}
	}
	PanicIf(true, "tx: %s DoOne: nothing deleted for %q", reg.Tx.UUID, abstract)
	return nil
}

func (be *MemoryBackend) GetModelEntitiesByAbstract(r *registry.Registry) (map[string]string, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return nil, xErr
	}

	res := map[string]string{}
	for _, me := range tx.db.modelEnts {
		if me.RegSID == r.SID {
			res[me.Abstract] = me.SID
		}
	}
	return res, nil
}

func (be *MemoryBackend) CountGroupModelUsed(r *registry.Registry, gmSID string) (int, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return 0, xErr
	}

	count := 0
	for _, g := range tx.db.groups {
		if g.ModelSID == gmSID {
			count++
		}
	}
	return count, nil
}

func (be *MemoryBackend) CountResourceModelUsed(r *registry.Registry, rmSID string) (int, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return 0, xErr
	}

	count := 0
	for _, res := range tx.db.resources {
		if res.ModelSID == rmSID {
			count++
		}
	}
	return count, nil
}

// versionXID, attr, err
func (be *MemoryBackend) IsHasDocUsed(r *registry.Registry, rm *registry.ResourceModel) (string, string, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return "", "", xErr
	}

	names := []string{
		rm.Singular + string(DB_IN),
		rm.Singular + "url" + string(DB_IN),
		rm.Singular + "base64" + string(DB_IN),
		rm.Singular + "proxyurl" + string(DB_IN),
	}

	vers := []*verRow{}
	for _, v := range tx.db.versions {
		if res := tx.db.resources[v.ResSID]; res != nil && res.ModelSID == rm.SID {
			vers = append(vers, v)
		}
	}
	sort.Slice(vers, func(i, j int) bool { return vers[i].XID < vers[j].XID })

	for _, v := range vers {
		for _, name := range names {
			if p := tx.db.props[v.SID][strings.ToLower(name)]; p != nil {
				return v.XID, strings.TrimRight(p.Name, string(DB_IN)), nil
			}
		}
	}
	return "", "", nil
}

func (be *MemoryBackend) HasDocContent(r *registry.Resource) (string, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return "", xErr
	}

	for _, v := range tx.versionsOf(r.SID) {
		if tx.db.contents[v.SID] != nil {
			return v.XID, nil
		}
	}
	return "", nil
}

// Registry stuff

func (be *MemoryBackend) ListRegistries(tx *registry.Tx) ([]string, *XRError) {
	mtx, xErr := mt(tx)
	if xErr != nil {
		return nil, xErr
	}

	res := []string{}
	for _, r := range mtx.db.regs {
		res = append(res, r.UID)
	}
	sort.Slice(res, func(i, j int) bool {
		a, b := strings.ToLower(res[i]), strings.ToLower(res[j])
		if a != b {
			return a < b
		}
		return res[i] < res[j]
	})
	return res, nil
}

func (be *MemoryBackend) DeleteRegistry(r *registry.Registry) *XRError {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return xErr
	}
	PanicIf(!tx.deleteRegistry(r.SID), "tx: %s DoOne: Registry not found: %s",
		r.Tx.UUID, r.SID)
	return nil
}

func (be *MemoryBackend) MapRegistryUID2SID(tx *registry.Tx, uid string) (string, *XRError) {
	mtx, xErr := mt(tx)
	if xErr != nil {
		return "", xErr
	}

	for _, r := range mtx.db.regs {
		if strings.EqualFold(r.UID, uid) {
			return r.SID, nil
		}
	}
	log.VPrintf(3, "tx: %s None found", tx.UUID)
	return "", nil
}

func (be *MemoryBackend) RegistryGetUsesXref(r *registry.Registry) (bool, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return false, xErr
	}

	if row := tx.db.regs[r.SID]; row != nil {
		return row.UsesXref, nil
	}
	panic("can't find registry: " + r.SID)
}

func (be *MemoryBackend) RegistrySetUsesXref(r *registry.Registry, b bool) *XRError {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return xErr
	}

	row := tx.db.regs[r.Tx.Registry.SID]
	if row != nil && row.UsesXref != b {
		n := *row
		n.UsesXref = b
		putRow(tx, tx.db.regs, n.SID, &n)
	}
	return nil
}

// The triggers on Props/Resources already do this
func (be *MemoryBackend) RegistryRecalcUsesXref(r *registry.Registry) *XRError {
	return nil
}

func (be *MemoryBackend) MapShortSelf(tx *registry.Tx, ss string) (string, string, *XRError) {
	mtx, xErr := mt(tx)
	if xErr != nil {
		return "", "", xErr
	}

	if e := mtx.db.ents[ss]; e != nil {
		if reg := mtx.db.regs[e.RegSID]; reg != nil {
			return reg.UID, e.XID, nil
		}
	}
	return "", "", nil
}

func (be *MemoryBackend) DeleteGroup(g *registry.Group) *XRError {
	tx, xErr := mt(g.Tx)
	if xErr != nil {
		return xErr
	}
	PanicIf(!tx.deleteGroup(g.SID), "tx: %s DoOne: Group not found: %s",
		g.Tx.UUID, g.SID)
	return nil
}

func (be *MemoryBackend) HasReadOnlyResource(g *registry.Group) (bool, *XRError) {
	tx, xErr := mt(g.Tx)
	if xErr != nil {
		return false, xErr
	}

	prefix := g.XID + "/"
	for _, e := range tx.entsOfReg(g.Registry.SID) {
		if e.Type != ENTITY_META || !strings.HasPrefix(e.XID, prefix) {
			continue
		}
		p := tx.db.props[e.SID]["readonly"+string(DB_IN)]
		if p != nil && p.Value == "true" {
			return true, nil
		}
	}
	return false, nil
}

func (be *MemoryBackend) LockResourceBySID(tx *registry.Tx, sid string) *XRError {
	return nil
}

func (be *MemoryBackend) DeleteResource(r *registry.Resource) *XRError {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return xErr
	}
	PanicIf(!tx.deleteResource(r.SID), "tx: %s DoOne: Resource not found: %s",
		r.Tx.UUID, r.SID)
	return nil
}

func (be *MemoryBackend) DeleteMeta(meta *registry.Meta) *XRError {
	tx, xErr := mt(meta.Tx)
	if xErr != nil {
		return xErr
	}
	PanicIf(!delRow(tx, tx.db.metas, meta.SID),
		"tx: %s DoOne: Meta not found: %s", meta.Tx.UUID, meta.SID)
	return nil
}

func (be *MemoryBackend) DeleteVersion(v *registry.Version) *XRError {
	tx, xErr := mt(v.Tx)
	if xErr != nil {
		return xErr
	}
	tx.deleteVersion(v.SID)
	return nil
}

func (be *MemoryBackend) GetResourceContents(e *registry.Entity) ([]byte, *XRError) {
	if e.Type != ENTITY_RESOURCE && e.Type != ENTITY_VERSION {
		panic(fmt.Sprintf("%s: I'm not a Resource or Version", e.XID))
	}

	tx, xErr := mt(e.Tx)
	if xErr != nil {
		return nil, xErr
	}

	cid := e.Get("#contentid")
	cidStr, ok := cid.(string)
	if !ok {
		if IsNil(cid) {
			return nil, nil
		}
		cidStr = fmt.Sprintf("%v", cid)
	}

	row := tx.db.contents[cidStr]
	if row == nil {
		return nil, nil
	}
	return append([]byte{}, row.Data...), nil
}

func (be *MemoryBackend) SetResourceContents(e *registry.Entity, val []byte) *XRError {
	tx, xErr := mt(e.Tx)
	if xErr != nil {
		return xErr
	}

	if IsNil(val) {
		delRow(tx, tx.db.contents, e.SID)
	} else {
		putRow(tx, tx.db.contents, e.SID,
			&contentRow{Data: append([]byte{}, val...)})
	}
	return nil
}
