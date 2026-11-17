package memory

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func (tx *memTx) versionsOf(resSID string) []*verRow {
	res := []*verRow{}
	for _, e := range tx.childEnts(resSID) {
		if v := tx.db.versions[e.SID]; v != nil {
			res = append(res, v)
		}
	}
	return res
}

func (tx *memTx) metaOf(resSID string) *metaRow {
	for _, e := range tx.childEnts(resSID) {
		if m := tx.db.metas[e.SID]; m != nil {
			return m
		}
	}
	return nil
}

// A Props row for e. Callers set the Name/Value/PropType and any flags.
func propFor(e *registry.Entity, name, val, ptype string) *propRow {
	return &propRow{
		RegSID: e.Registry.SID, Type: e.Type, Plural: e.Plural,
		Singular: e.Singular, ParentSID: e.ParentSID, SID: e.SID, UID: e.UID,
		XID: e.XID, Name: name, Value: val, PropType: ptype,
		Abstract: e.Abstract, DocView: true,
	}
}

func (be *MemoryBackend) RegisterEntity(e *registry.Entity) *XRError {
	tx, xErr := mt(e.Tx)
	if xErr != nil {
		return xErr
	}
	db := tx.db

	switch e.Type {
	case ENTITY_REGISTRY:
		putRow(tx, db.regs, e.SID, &regRow{SID: e.SID, UID: e.UID})
		putRow(tx, db.models, e.SID, &modelRow{RegSID: e.SID})

	case ENTITY_GROUP:
		putRow(tx, db.groups, e.SID, &groupRow{
			SID: e.SID, UID: e.UID, RegSID: e.Registry.SID,
			ModelSID: e.GroupModel.SID, XID: e.XID, Abstract: e.Abstract,
			Plural: e.Plural, Singular: e.Singular})

	case ENTITY_RESOURCE:
		putRow(tx, db.resources, e.SID, &resRow{
			SID: e.SID, UID: e.UID, RegSID: e.Registry.SID,
			GroupSID: e.ParentSID, ModelSID: e.ResourceModel.SID,
			XID: e.XID, Abstract: e.Abstract,
			Plural: e.Plural, Singular: e.Singular})

	case ENTITY_META:
		putRow(tx, db.metas, e.SID, &metaRow{
			SID: e.SID, RegSID: e.Registry.SID, ResSID: e.ParentSID,
			XID: e.XID, Abstract: e.Abstract,
			Plural: e.ResourceModel.Plural, Singular: e.ResourceModel.Singular})

	case ENTITY_VERSION:
		putRow(tx, db.versions, e.SID, &verRow{
			SID: e.SID, UID: e.UID, RegSID: e.Registry.SID,
			ResSID: e.ParentSID, XID: e.XID, Abstract: e.Abstract})

	default:
		panic(fmt.Sprintf("Uknown type: %d", e.Type))
	}

	lxid := strings.ToLower(e.XID)
	if old := db.entByLXID[lxidKey(e.Registry.SID, lxid)]; old != nil {
		return NewXRError("server_error", "/").
			SetDetailf("Duplicate entity XID: %s.", e.XID)
	}
	tx.putEnt(&entRow{
		RegSID: e.Registry.SID, Type: e.Type, Plural: e.Plural,
		Singular: e.Singular, ParentSID: e.ParentSID, SID: e.SID, UID: e.UID,
		Abstract: e.Abstract, XID: e.XID, LowerXID: lxid,
	})

	p := propFor(e, "xid"+string(DB_IN), e.XID, STRING)
	p.IsCalcStatic = true
	tx.putProp(p)

	if e.Type == ENTITY_VERSION {
		res := db.resources[e.ParentSID]
		if res != nil {
			p = propFor(e, res.Singular+"id"+string(DB_IN), res.UID, STRING)
			p.IsCalcStatic = true
			tx.putProp(p)
		}

		m := tx.metaOf(e.ParentSID)
		if m == nil {
			return NewXRError("server_error", "/").
				SetDetailf("Can't find the Meta for %q.", e.XID)
		}
		val := "false"
		if m.DefVID != nil && strings.EqualFold(*m.DefVID, e.UID) {
			val = "true"
		}
		p = propFor(e, "isdefault"+string(DB_IN), val, BOOLEAN)
		p.IsCalcDynamic = true
		tx.putProp(p)
	}

	return nil
}

func (tx *memTx) setEntityProps(e *registry.Entity, props []*propRow) *XRError {
	for _, p := range props {
		var xErr *XRError
		val := p.Value
		if p.IsSystemProp {
			xErr = e.SetSystemFromDBName(p.Name, &val, p.PropType)
		} else {
			xErr = e.SetFromDBName(p.Name, &val, p.PropType)
		}
		if xErr != nil {
			return xErr
		}
	}
	return nil
}

func (be *MemoryBackend) RefreshEntity(e *registry.Entity, accessMode int) *XRError {
	tx, xErr := mt(e.Tx)
	if xErr != nil {
		return xErr
	}

	if accessMode == FOR_WRITE && tx.db.ents[e.SID] == nil {
		return NewXRError("not_found", e.XID).
			SetDetailf("%q was deleted concurrently.", e.XID)
	}

	log.VPrintf(3, "tx: %s Refreshing %q", e.Tx.UUID, e.XID)

	e.Object = map[string]any{}
	e.System = map[string]any{}

	props := []*propRow{}
	for _, p := range tx.propsOf(e.SID) {
		if p.isPlain() {
			props = append(props, p)
		}
	}
	return tx.setEntityProps(e, props)
}

func (be *MemoryBackend) ClearUserProps(e *registry.Entity) *XRError {
	tx, xErr := mt(e.Tx)
	if xErr != nil {
		return xErr
	}

	tx.delPropsWhere(e.SID, func(p *propRow) bool {
		return p.isPlain() && !p.IsSystemProp
	})
	return nil
}

func (be *MemoryBackend) ListProps(reg *registry.Registry, all bool) ([][3]string, *XRError) {
	tx, xErr := mt(reg.GetTx())
	if xErr != nil {
		return nil, xErr
	}

	rows := []*propRow{}
	for _, m := range tx.db.props {
		for _, p := range m {
			if p.RegSID == reg.SID && (all || p.isPlain()) {
				rows = append(rows, p)
			}
		}
	}

	// Case-insensitive, like the mysql collation
	sort.Slice(rows, func(i, j int) bool {
		ax, bx := strings.ToLower(rows[i].XID), strings.ToLower(rows[j].XID)
		if ax != bx {
			return ax < bx
		}
		an, bn := strings.ToLower(rows[i].Name), strings.ToLower(rows[j].Name)
		if an != bn {
			return an < bn
		}
		return rows[i].Name < rows[j].Name
	})

	res := make([][3]string, len(rows))
	for i, p := range rows {
		res[i] = [3]string{p.XID, p.Name, p.Value}
	}
	return res, nil
}

// Building Entities from rows

func (be *MemoryBackend) buildEntity(rtx *registry.Tx, e *entRow, props []*propRow, mode int, mask uint64) (*registry.Entity, *XRError) {
	entity := &registry.Entity{
		Tx:         rtx,
		AccessMode: mode,
		Registry:   rtx.Registry,
		SID:        e.SID,
		ParentSID:  e.ParentSID,
		Plural:     e.Plural,
		Singular:   e.Singular,
		UID:        e.UID,

		Type:     e.Type,
		XID:      e.XID,
		Abstract: e.Abstract,
	}

	entity.GroupModel, entity.ResourceModel =
		registry.AbstractToModels(rtx.Registry, entity.Abstract)

	if mask != 0 {
		entity.SetStuff("filterMask", mask)
	}

	if xErr := (&memTx{}).setEntityProps(entity, props); xErr != nil {
		return nil, xErr
	}

	entity.PopulateOriginObject()
	return entity, nil
}

func sortByXID(ents []*entRow) {
	sort.Slice(ents, func(i, j int) bool { return ents[i].XID < ents[j].XID })
}

func (be *MemoryBackend) entitiesWhere(rtx *registry.Tx, regSID string, mode int, pred func(*entRow) bool) ([]*registry.Entity, *XRError) {
	tx, xErr := mt(rtx)
	if xErr != nil {
		return nil, xErr
	}

	ents := []*entRow{}
	for _, e := range tx.entsOfReg(regSID) {
		if pred(e) {
			ents = append(ents, e)
		}
	}
	sortByXID(ents)

	res := make([]*registry.Entity, 0, len(ents))
	for _, e := range ents {
		props := []*propRow{}
		for _, p := range tx.propsOf(e.SID) {
			if p.isPlain() {
				props = append(props, p)
			}
		}
		entity, xErr := be.buildEntity(rtx, e, props, mode, 0)
		if xErr != nil {
			return nil, xErr
		}
		res = append(res, entity)
	}

	if mode == FOR_WRITE {
		for _, entity := range res {
			if xErr := registry.LockEntityFamily(rtx, entity); xErr != nil {
				return nil, xErr
			}
		}
	}
	return res, nil
}

func (be *MemoryBackend) GetGroups(r *registry.Registry, accessMode int) ([]*registry.Group, *XRError) {
	entities, xErr := be.entitiesWhere(r.Tx, r.SID, accessMode,
		func(e *entRow) bool { return e.Type == ENTITY_GROUP })
	if xErr != nil {
		return nil, xErr
	}

	groups := make([]*registry.Group, len(entities))
	for i, e := range entities {
		gType, _, _ := strings.Cut(e.XID, string(DB_IN))
		g := r.Tx.GetGroup(r, gType, e.UID)
		if g == nil {
			g = &registry.Group{
				Entity:   *e,
				Registry: r,
			}
			g.Self = g
			g.Tx.AddGroup(g)
		}
		groups[i] = g
	}
	return groups, nil
}

func (be *MemoryBackend) idsByType(rtx *registry.Tx, parentSID, typ string) ([]string, *XRError) {
	tx, xErr := mt(rtx)
	if xErr != nil {
		return nil, xErr
	}

	list := []string{}
	for _, e := range tx.childEnts(parentSID) {
		if e.Plural == typ {
			list = append(list, e.UID)
		}
	}
	return list, nil
}

func (be *MemoryBackend) GetGroupIDsByType(r *registry.Registry, gType string, accessMode int) ([]string, *XRError) {
	return be.idsByType(r.Tx, r.SID, gType)
}

func (be *MemoryBackend) GetResourceIDsByType(g *registry.Group, rType string, accessMode int) ([]string, *XRError) {
	return be.idsByType(g.Tx, g.SID, rType)
}

func (be *MemoryBackend) GetResources(g *registry.Group, accessMode int) ([]*registry.Resource, *XRError) {
	entities, xErr := be.entitiesWhere(g.Tx, g.Registry.SID, FOR_WRITE,
		func(e *entRow) bool { return e.ParentSID == g.SID })
	if xErr != nil {
		return nil, xErr
	}

	resources := make([]*registry.Resource, len(entities))
	for i, e := range entities {
		parts := strings.Split(e.XID, "/") // /gType/gID/rType/rID
		r := g.Tx.GetResource(g, parts[3], e.UID)
		if r == nil {
			r = &registry.Resource{
				Entity: *e,
				Group:  g,
			}
			r.Self = r
			r.Tx.AddResource(r)
		}
		resources[i] = r
	}
	return resources, nil
}

func (be *MemoryBackend) GetVersions(r *registry.Resource, accessMode int) ([]*registry.Version, *XRError) {
	entities, xErr := be.entitiesWhere(r.Tx, r.Registry.SID, accessMode,
		func(e *entRow) bool {
			return e.ParentSID == r.SID && e.Type == ENTITY_VERSION
		})
	if xErr != nil {
		return nil, xErr
	}

	versions := make([]*registry.Version, len(entities))
	for i, e := range entities {
		v := r.Tx.GetVersion(r, e.UID)
		if v == nil {
			v = &registry.Version{
				Entity:   *e,
				Resource: r,
			}
			v.Self = v
			v.Tx.AddVersion(v)
		}
		versions[i] = v
	}
	return versions, nil
}

func (be *MemoryBackend) EntityRealXID(tx *registry.Tx, xidStr string, accessMode int) (string, *XRError) {
	mtx, xErr := mt(tx)
	if xErr != nil {
		return "", xErr
	}

	e := mtx.db.entByLXID[lxidKey(tx.Registry.SID, strings.ToLower(xidStr))]
	if e == nil {
		return "", nil
	}
	return e.XID, nil
}

func (be *MemoryBackend) GetEntityFromXID(tx *registry.Tx, regSID string, xidStr string, allAttrs bool, accessMode int) (*registry.Entity, *XRError) {
	iter, xErr := be.GetEntitiesFromXID(tx, regSID, xidStr, allAttrs, false,
		accessMode)
	if xErr != nil {
		return nil, xErr
	}

	e, xErr := iter.Next()
	if xErr != nil {
		return nil, xErr
	}

	e2, xErr := iter.Next()
	if xErr != nil {
		return nil, xErr
	}
	PanicIf(e2 != nil, "too many entities: %v", e2)

	return e, nil
}

func (be *MemoryBackend) GetEntitiesFromXID(tx *registry.Tx, regSID string, xidStr string, allAttrs bool, children bool, accessMode int) (registry.EntityIterator, *XRError) {
	mtx, xErr := mt(tx)
	if xErr != nil {
		return nil, xErr
	}

	prefix := xidStr + "/"
	ents := []*entRow{}
	for _, e := range mtx.entsOfReg(regSID) {
		if e.XID == xidStr || (children && strings.HasPrefix(e.XID, prefix)) {
			ents = append(ents, e)
		}
	}
	sortByXID(ents)

	items := []iterItem{}
	for _, e := range ents {
		props := []*propRow{}
		for _, p := range mtx.propsOf(e.SID) {
			if allAttrs || p.isPlain() {
				props = append(props, p)
			}
		}
		// Entities without any matching props don't show up
		if len(props) > 0 {
			items = append(items, iterItem{ent: e, props: props})
		}
	}

	return &memIter{be: be, tx: tx, mode: accessMode, items: items,
		size: uint64(len(items)), total: math.MaxUint64}, nil
}

// SaveEntity & friends

func (be *MemoryBackend) SaveEntity(e *registry.Entity) *XRError {
	newObj := e.NewObject

	tx, xErr := mt(e.Tx)
	if xErr != nil {
		return xErr
	}

	if xErr := be.ClearUserProps(e); xErr != nil {
		return xErr
	}

	resSingular := ""
	resHasDoc := false
	if rm := e.GetResourceModel(); rm != nil {
		resSingular = rm.Singular
		resHasDoc = rm.GetHasDocument()
	}

	rows := []*propRow{}
	setProp := func(pp *PropPath, val any) *XRError {
		row, skip, xErr := prepDBProperty(e, pp, val)
		if xErr != nil {
			return xErr
		}
		if !skip && row != nil {
			rows = append(rows, row)
		}
		return nil
	}

	var traverse func(pp *PropPath, val any) *XRError
	traverse = func(pp *PropPath, val any) *XRError {
		if IsNil(val) {
			return nil
		}

		valValue := reflect.ValueOf(val)

		switch valValue.Kind() {
		case reflect.Map:
			keys := valValue.MapKeys()
			count := 0
			for _, keyValue := range keys {
				if keyValue.Kind() != reflect.String {
					return NewXRError("invalid_attribute", e.XID,
						"name="+pp.RemoveLast().UI(),
						"error_detail="+
							fmt.Sprintf("map key (%s) needs to be a string, "+
								"not %s", pp.Last().Text,
								keyValue.Kind().String()))
				}

				k := keyValue.Interface().(string)
				v := valValue.MapIndex(keyValue).Interface()
				if resHasDoc && pp.Len() == 0 && k == resSingular {
					if xErr := setProp(pp.P(k), v); xErr != nil {
						return xErr
					}
				} else if k[0] == '#' {
					if xErr := setProp(pp.P(k), v); xErr != nil {
						return xErr
					}
				} else {
					if IsNil(v) {
						continue
					}
					if xErr := traverse(pp.P(k), v); xErr != nil {
						return xErr
					}
				}
				count++
			}
			if count == 0 && pp.Len() != 0 {
				return setProp(pp, map[string]any{})
			}

		case reflect.Slice:
			if valValue.Len() == 0 {
				valValue = reflect.MakeSlice(reflect.TypeOf(val), 0, 0)
				return setProp(pp, valValue.Interface())
			}
			for i := 0; i < valValue.Len(); i++ {
				v := valValue.Index(i).Interface()
				if xErr := traverse(pp.I(i), v); xErr != nil {
					return xErr
				}
			}

		case reflect.Struct:
			panic("a struct")

		default:
			return setProp(pp, val)
		}
		return nil
	}

	if xErr := traverse(NewPP(), newObj); xErr != nil {
		return xErr
	}

	for _, row := range rows {
		tx.putProp(row)
	}
	return nil
}

// Validates and converts val into the Props row for pp. skip=true means
// there's nothing to store (e.g. DontStore props, or the RESOURCE content
// which lives in its own table).
func prepDBProperty(e *registry.Entity, pp *PropPath, val any) (row *propRow, skip bool, xErr *XRError) {
	PanicIf(pp.UI() == "", "pp is empty")

	name := pp.DB()
	docView := true

	if len(name) > MAX_PROPNAME {
		return nil, false, NewXRError("invalid_attribute",
			e.XID, "name="+name,
			"error_detail="+
				fmt.Sprintf("attribute names must not exceed %d chars",
					MAX_PROPNAME))
	}

	_, propsMap := e.GetPropsOrdered()
	specProp, ok := propsMap[pp.Top()]
	if ok && specProp.Internals != nil {
		if specProp.Internals.DontStore {
			return nil, true, nil
		}
		if specProp.Internals.NoDocView {
			docView = false
		}
	}

	PanicIf(e.SID == "", "SID should not be empty")
	PanicIf(e.Registry == nil, "Registry should not be nil")

	if (e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION) && pp.Len() == 1 {
		rm := e.GetResourceModel()
		if rm.GetHasDocument() && pp.Top() == rm.Singular {
			if IsNil(val) {
				if xErr := e.Tx.Backend.SetResourceContents(e, nil); xErr != nil {
					return nil, false, xErr
				}
			} else {
				var buf []byte

				if by, ok := val.([]byte); ok {
					buf = by
				} else if str, ok := val.(string); ok {
					buf = []byte(str)
				} else {
					buf = []byte(fmt.Sprintf("%v", val))
				}

				if xErr := e.Tx.Backend.SetResourceContents(e, buf); xErr != nil {
					return nil, false, xErr
				}

				PanicIf(IsNil(e.NewObject["#contentid"]), "Missing cid")

				return nil, true, nil
			}
		}
	}

	if IsNil(val) {
		return nil, true, nil
	}

	ptype := registry.GoToOurType(val)
	dbVal := val
	if ptype == BOOLEAN {
		if val == true {
			dbVal = "true"
		} else {
			dbVal = "false"
		}
	}

	switch ptype {
	case STRING:
		if s := val.(string); len(s) > MAX_VARCHAR {
			return nil, false, NewXRError("invalid_attribute",
				e.XID, "name="+pp.UI(),
				"error_detail="+
					fmt.Sprintf("must be less than %d chars",
						MAX_VARCHAR+1))
		}
	case ARRAY:
		if a := val.([]any); len(a) > 0 {
			return nil, false, NewXRError("invalid_attribute",
				e.XID, "name="+pp.UI(),
				"error_detail=can't set non-empty arrays")
		}
		dbVal = ""
	case MAP:
		if m := val.(map[string]any); len(m) > 0 {
			return nil, false, NewXRError("invalid_attribute",
				e.XID, "name= "+pp.UI(),
				"error_detail=can't set non-empty maps")
		}
		dbVal = ""
	case OBJECT:
		dbVal = ""
	}

	row = propFor(e, name, fmt.Sprintf("%v", dbVal), ptype)
	row.DocView = docView
	return row, false, nil
}

func (be *MemoryBackend) SaveChangedSystemProps(e *registry.Entity, changed map[string]any) *XRError {
	tx, xErr := mt(e.Tx)
	if xErr != nil {
		return xErr
	}

	_, propsMap := e.GetPropsOrdered()

	inserts := []*propRow{}
	deletes := []string{}

	names := make([]string, 0, len(changed))
	for name := range changed {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		val := changed[name]
		docView := true
		if specProp, ok := propsMap[name]; ok && specProp.Internals != nil &&
			specProp.Internals.NoDocView {
			docView = false
		}
		dbName := NewPPP(name).DB()

		if IsNil(val) {
			deletes = append(deletes, strings.ToLower(dbName))
			continue
		}

		propType := registry.GoToOurType(val)
		dbVal := val
		if propType == BOOLEAN {
			if val == true {
				dbVal = "true"
			} else {
				dbVal = "false"
			}
		}
		if propType == ARRAY || propType == MAP || propType == OBJECT {
			dbVal = ""
		}

		row := propFor(e, dbName, fmt.Sprintf("%v", dbVal), propType)
		row.DocView = docView
		row.IsSystemProp = true
		inserts = append(inserts, row)
	}

	for _, lname := range deletes {
		if p := tx.db.props[e.SID][lname]; p != nil && p.isOwn() {
			tx.delProp(e.SID, lname)
		}
	}
	for _, row := range inserts {
		tx.putProp(row)
	}
	return nil
}

// Default Version & Xref stuff

func (be *MemoryBackend) RecalcVersionsIsDefault(r *registry.Resource) *XRError {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return xErr
	}

	m := tx.metaOf(r.SID)
	if m == nil {
		return nil
	}

	name := strings.ToLower("isdefault" + string(DB_IN))
	for _, v := range tx.versionsOf(r.SID) {
		p := tx.db.props[v.SID][name]
		if p == nil {
			continue
		}
		val := "false"
		if m.DefVID != nil && strings.EqualFold(v.UID, *m.DefVID) {
			val = "true"
		}
		n := *p
		n.Value = val
		tx.replaceProp(p, &n)
	}
	return nil
}

// A direct UPDATE of a Props row (no triggers)
func (tx *memTx) replaceProp(old, n *propRow) {
	db := tx.db
	db.rawPutProp(n)
	tx.record(func() { db.rawPutProp(old) })
}

func (be *MemoryBackend) DeleteResourceDefaultVersionProps(r *registry.Resource) *XRError {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return xErr
	}

	tx.delPropsWhere(r.SID, func(p *propRow) bool { return p.IsDefaultVerCopy })
	return nil
}

func (be *MemoryBackend) CopyResourceDefaultVersionProps(r *registry.Resource) *XRError {
	ver, xErr := r.GetDefault()
	if xErr != nil {
		return xErr
	}
	if ver == nil {
		return nil
	}

	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return xErr
	}

	for _, p := range tx.propsOf(ver.SID) {
		if !p.isOwn() || p.IsCalcStatic {
			continue
		}
		n := propFor(&r.Entity, p.Name, p.Value, p.PropType)
		n.DocView = false
		n.IsDefaultVerCopy = true
		tx.putProp(n)
	}
	return nil
}

func (be *MemoryBackend) ClearXrefState(meta *registry.Meta) *XRError {
	tx, xErr := mt(meta.Tx)
	if xErr != nil {
		return xErr
	}

	tx.delPropsWhere(meta.SID, func(p *propRow) bool { return p.IsXrefPropCopy })
	for _, c := range tx.childEnts(meta.ParentSID) {
		tx.delPropsWhere(c.SID, func(p *propRow) bool { return p.IsXrefVerCopy })
	}
	for _, c := range tx.childEnts(meta.ParentSID) {
		if c.IsXrefVerCopy {
			tx.delEnt(c.SID)
		}
	}
	return nil
}

func (tx *memTx) findResourceByXID(regSID, xid string) *resRow {
	e := tx.db.entByLXID[lxidKey(regSID, strings.ToLower(xid))]
	if e == nil || e.XID != xid {
		return nil
	}
	return tx.db.resources[e.SID]
}

func (be *MemoryBackend) CopyXrefState(meta *registry.Meta) *XRError {
	PanicIf(meta.AccessMode != FOR_WRITE, "%q should be locked", meta.XID)
	xRefXID := meta.GetAsString("xref")

	tx, xErr := mt(meta.Tx)
	if xErr != nil {
		return xErr
	}

	target := tx.findResourceByXID(meta.Registry.SID, xRefXID)
	if target == nil {
		return nil
	}
	targetMeta := tx.metaOf(target.SID)
	if targetMeta == nil {
		return nil
	}

	resource := meta.Self.(*registry.Meta).Resource

	skip1 := strings.ToLower("xref" + string(DB_IN))
	skip2 := strings.ToLower(target.Singular + "id" + string(DB_IN))
	for _, p := range tx.propsOf(targetMeta.SID) {
		lname := strings.ToLower(p.Name)
		if !p.isPlain() || lname == skip1 || lname == skip2 || p.Name[0] == '#' {
			continue
		}
		n := propFor(&meta.Entity, p.Name, p.Value, p.PropType)
		n.DocView = false
		n.IsXrefPropCopy = true
		tx.putProp(n)
	}

	if resource != nil {
		if xErr := resource.SaveXrefVersionCopies(target.SID); xErr != nil {
			return xErr
		}
	}
	return nil
}

func (be *MemoryBackend) CopyXrefDefaultVersionProps(r *registry.Resource) *XRError {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return xErr
	}

	srcM := tx.metaOf(r.SID)
	if srcM == nil || srcM.XRef == nil {
		return nil
	}
	target := tx.findResourceByXID(srcM.RegSID, *srcM.XRef)
	if target == nil {
		return nil
	}
	tm := tx.metaOf(target.SID)
	if tm == nil || tm.DefVID == nil {
		return nil
	}

	var defVer *verRow
	for _, v := range tx.versionsOf(target.SID) {
		if strings.EqualFold(v.UID, *tm.DefVID) {
			defVer = v
			break
		}
	}
	if defVer == nil {
		return nil
	}

	synthSID := fmt.Sprintf("-%s-%s", r.SID, defVer.SID)
	for _, p := range tx.propsOf(synthSID) {
		if !p.IsXrefVerCopy || p.IsCalcStatic {
			continue
		}
		n := propFor(&r.Entity, p.Name, p.Value, p.PropType)
		n.DocView = false
		n.IsDefaultVerCopy = true
		tx.putProp(n)
	}
	return nil
}

func (be *MemoryBackend) CopyXrefVersions(r *registry.Resource, targetSID string) *XRError {
	if r == nil {
		return nil
	}

	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return xErr
	}
	db := tx.db

	srcSID := r.SID
	synthAbstract := r.Abstract + string(DB_IN) + "versions"
	tgtVers := tx.versionsOf(targetSID)
	sort.Slice(tgtVers, func(i, j int) bool { return tgtVers[i].SID < tgtVers[j].SID })

	for _, v := range tgtVers {
		sid := fmt.Sprintf("-%s-%s", srcSID, v.SID)
		tx.delAllProps(sid)
		tx.delEnt(sid)
	}

	tm := tx.metaOf(targetSID)
	srcRes := db.resources[srcSID]
	skip := strings.ToLower("xref" + string(DB_IN))

	for _, v := range tgtVers {
		sid := fmt.Sprintf("-%s-%s", srcSID, v.SID)
		xid := r.XID + "/versions/" + v.UID

		tx.putEnt(&entRow{
			RegSID: r.Registry.SID, Type: ENTITY_VERSION, Plural: "versions",
			Singular: "version", ParentSID: srcSID, SID: sid, UID: v.UID,
			Abstract: synthAbstract, XID: xid, LowerXID: strings.ToLower(xid),
			IsXrefVerCopy: true,
		})

		mk := func(name, val, ptype string) *propRow {
			return &propRow{
				RegSID: r.Registry.SID, Type: ENTITY_VERSION,
				Plural: "versions", Singular: "version", ParentSID: srcSID,
				SID: sid, UID: v.UID, XID: xid, Name: name, Value: val,
				PropType: ptype, Abstract: synthAbstract,
				IsXrefVerCopy: true,
			}
		}

		for _, p := range tx.propsOf(v.SID) {
			if !p.isPlain() || strings.ToLower(p.Name) == skip {
				continue
			}
			tx.putProp(mk(p.Name, p.Value, p.PropType))
		}

		p := mk("xid"+string(DB_IN), xid, STRING)
		p.IsCalcStatic = true
		tx.putProp(p)

		if srcRes != nil {
			p = mk(srcRes.Singular+"id"+string(DB_IN), srcRes.UID, STRING)
			p.IsCalcStatic = true
			tx.putProp(p)
		}

		if tm != nil {
			val := "false"
			if tm.DefVID != nil && strings.EqualFold(*tm.DefVID, v.UID) {
				val = "true"
			}
			p = mk("isdefault"+string(DB_IN), val, BOOLEAN)
			p.IsCalcDynamic = true
			tx.putProp(p)
		}
	}
	return nil
}

func (be *MemoryBackend) GetXrefSources(r *registry.Resource) ([]string, *XRError) {
	tx, xErr := mt(r.Tx)
	if xErr != nil {
		return nil, xErr
	}

	XIDs := []string{}
	for _, m := range tx.db.metas {
		if m.RegSID == r.Registry.SID && m.XRef != nil && *m.XRef == r.XID {
			if res := tx.db.resources[m.ResSID]; res != nil {
				XIDs = append(XIDs, res.XID)
			}
		}
	}
	sort.Strings(XIDs)
	return XIDs, nil
}
