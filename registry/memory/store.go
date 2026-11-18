package memory

import (
	"sort"
	"strings"
	"time"

	. "github.com/xregistry/server/common"
)

// The tables below mirror the ones in registry/mysql/init.sql. Rows are
// immutable once stored - to change a row, copy it and put the new copy.

type regRow struct {
	SID      string
	UID      string
	UsesXref bool
}

type modelRow struct {
	Model     string
	HasModel  bool
	Source    []byte
	UUID      string
	RegSID    string
	HasSource bool
}

type modelEntRow struct {
	SID       string
	RegSID    string
	ParentSID string
	Abstract  string
	Plural    string
	Singular  string
}

type groupRow struct {
	SID      string
	UID      string
	RegSID   string
	ModelSID string
	XID      string
	Abstract string
	Plural   string
	Singular string
}

type resRow struct {
	SID      string
	UID      string
	RegSID   string
	GroupSID string
	ModelSID string
	XID      string
	Abstract string
	Plural   string
	Singular string
}

type metaRow struct {
	SID      string
	RegSID   string
	ResSID   string
	XID      string
	Abstract string
	Plural   string
	Singular string
	XRef     *string
	DefVID   *string
}

type verRow struct {
	SID        string
	UID        string
	RegSID     string
	ResSID     string
	XID        string
	Abstract   string
	AncestorID string
	CreatedAt  string
	ModifiedAt string
}

type contentRow struct {
	Data []byte
}

type entRow struct {
	RegSID        string
	Type          int
	Plural        string
	Singular      string
	ParentSID     string
	SID           string
	UID           string
	Abstract      string
	XID           string
	LowerXID      string
	IsXrefVerCopy bool
}

type propRow struct {
	RegSID    string
	Type      int
	Plural    string
	Singular  string
	ParentSID string
	SID       string
	UID       string
	XID       string
	Name      string // includes trailing DB_IN
	Value     string
	PropType  string
	Abstract  string
	DocView   bool

	IsDefaultVerCopy bool
	IsXrefPropCopy   bool
	IsXrefVerCopy    bool
	IsSystemProp     bool
	IsCalcStatic     bool
	IsCalcDynamic    bool
}

func (p *propRow) isOwn() bool {
	return !p.IsDefaultVerCopy && !p.IsXrefPropCopy && !p.IsXrefVerCopy
}

func (p *propRow) isPlain() bool {
	return p.isOwn() && !p.IsCalcStatic && !p.IsCalcDynamic
}

type memDB struct {
	name string
	lock chan struct{}

	regs      map[string]*regRow // SID
	models    map[string]*modelRow
	modelEnts map[string]*modelEntRow
	groups    map[string]*groupRow
	resources map[string]*resRow
	metas     map[string]*metaRow
	versions  map[string]*verRow
	contents  map[string]*contentRow
	ents      map[string]*entRow

	entByReg    map[string]map[string]*entRow // RegSID->SID
	entByParent map[string]map[string]*entRow // ParentSID->SID
	entByLXID   map[string]*entRow            // RegSID+\x00+LowerXID

	props map[string]map[string]*propRow // SID->lower(Name)
}

func newMemDB(name string) *memDB {
	return &memDB{
		name:        name,
		lock:        make(chan struct{}, 1),
		regs:        map[string]*regRow{},
		models:      map[string]*modelRow{},
		modelEnts:   map[string]*modelEntRow{},
		groups:      map[string]*groupRow{},
		resources:   map[string]*resRow{},
		metas:       map[string]*metaRow{},
		versions:    map[string]*verRow{},
		contents:    map[string]*contentRow{},
		ents:        map[string]*entRow{},
		entByReg:    map[string]map[string]*entRow{},
		entByParent: map[string]map[string]*entRow{},
		entByLXID:   map[string]*entRow{},
		props:       map[string]map[string]*propRow{},
	}
}

// memTx is the state behind registry.Tx.Tx. It holds the DB's lock for its
// whole life and keeps an undo journal so that Rollback can restore the DB.
type memTx struct {
	db   *memDB
	undo []func()
}

const lockTimeout = 120 * time.Second

func (db *memDB) acquire() *XRError {
	select {
	case db.lock <- struct{}{}:
		return nil
	default:
	}

	t := time.NewTimer(lockTimeout)
	defer t.Stop()
	select {
	case db.lock <- struct{}{}:
		return nil
	case <-t.C:
		return NewXRError("server_error", "/").
			SetDetailf("Timed out waiting for the %q in-memory DB lock "+
				"(possible nested Tx).", db.name)
	}
}

func (db *memDB) release() {
	<-db.lock
}

func (tx *memTx) record(fn func()) {
	tx.undo = append(tx.undo, fn)
}

func (tx *memTx) rollback() {
	for i := len(tx.undo) - 1; i >= 0; i-- {
		tx.undo[i]()
	}
	tx.undo = nil
}

func putRow[T any](tx *memTx, m map[string]*T, key string, val *T) {
	old, had := m[key]
	m[key] = val
	tx.record(func() {
		if had {
			m[key] = old
		} else {
			delete(m, key)
		}
	})
}

func delRow[T any](tx *memTx, m map[string]*T, key string) bool {
	old, had := m[key]
	if !had {
		return false
	}
	delete(m, key)
	tx.record(func() { m[key] = old })
	return true
}

func lxidKey(regSID, lxid string) string {
	return regSID + "\x00" + lxid
}

func (db *memDB) rawPutEnt(e *entRow) {
	if old := db.ents[e.SID]; old != nil {
		db.rawDelEnt(old.SID)
	}
	db.ents[e.SID] = e
	if db.entByReg[e.RegSID] == nil {
		db.entByReg[e.RegSID] = map[string]*entRow{}
	}
	db.entByReg[e.RegSID][e.SID] = e
	if db.entByParent[e.ParentSID] == nil {
		db.entByParent[e.ParentSID] = map[string]*entRow{}
	}
	db.entByParent[e.ParentSID][e.SID] = e
	db.entByLXID[lxidKey(e.RegSID, e.LowerXID)] = e
}

func (db *memDB) rawDelEnt(sid string) {
	e := db.ents[sid]
	if e == nil {
		return
	}
	delete(db.ents, sid)
	delete(db.entByReg[e.RegSID], sid)
	if len(db.entByReg[e.RegSID]) == 0 {
		delete(db.entByReg, e.RegSID)
	}
	delete(db.entByParent[e.ParentSID], sid)
	if len(db.entByParent[e.ParentSID]) == 0 {
		delete(db.entByParent, e.ParentSID)
	}
	k := lxidKey(e.RegSID, e.LowerXID)
	if db.entByLXID[k] == e {
		delete(db.entByLXID, k)
	}
}

func (tx *memTx) putEnt(e *entRow) {
	db := tx.db
	old := db.ents[e.SID]
	db.rawPutEnt(e)
	tx.record(func() {
		if old != nil {
			db.rawPutEnt(old)
		} else {
			db.rawDelEnt(e.SID)
		}
	})
}

func (tx *memTx) delEnt(sid string) {
	db := tx.db
	old := db.ents[sid]
	if old == nil {
		return
	}
	db.rawDelEnt(sid)
	tx.record(func() { db.rawPutEnt(old) })
}

func (db *memDB) rawPutProp(p *propRow) {
	m := db.props[p.SID]
	if m == nil {
		m = map[string]*propRow{}
		db.props[p.SID] = m
	}
	m[strings.ToLower(p.Name)] = p
}

func (db *memDB) rawDelProp(sid, lname string) {
	m := db.props[sid]
	if m == nil {
		return
	}
	delete(m, lname)
	if len(m) == 0 {
		delete(db.props, sid)
	}
}

// delProp removes a Props row and fires the "delete" trigger logic
func (tx *memTx) delProp(sid, lname string) bool {
	db := tx.db
	old := db.props[sid][lname]
	if old == nil {
		return false
	}
	db.rawDelProp(sid, lname)
	tx.record(func() { db.rawPutProp(old) })
	tx.propDeleted(old)
	return true
}

// putProp is REPLACE INTO Props: the delete trigger runs for the old row,
// then the insert trigger for the new one
func (tx *memTx) putProp(p *propRow) {
	db := tx.db
	lname := strings.ToLower(p.Name)
	old := db.props[p.SID][lname]
	if old != nil {
		db.rawDelProp(p.SID, lname)
		tx.record(func() { db.rawPutProp(old) })
		tx.propDeleted(old)
	}
	db.rawPutProp(p)
	tx.record(func() { db.rawDelProp(p.SID, lname) })
	tx.propInserted(p)
}

// FullTreeAncestor trigger
func (tx *memTx) propInserted(p *propRow) {
	if !p.isOwn() {
		return
	}
	db := tx.db
	switch p.Type {
	case ENTITY_VERSION:
		v := db.versions[p.SID]
		if v == nil {
			return
		}
		n := *v
		switch strings.ToLower(p.Name) {
		case "ancestorid" + string(DB_IN):
			n.AncestorID = p.Value
		case "createdat" + string(DB_IN):
			n.CreatedAt = p.Value
		case "modifiedat" + string(DB_IN):
			n.ModifiedAt = p.Value
		default:
			return
		}
		putRow(tx, db.versions, n.SID, &n)

	case ENTITY_META:
		m := db.metas[p.SID]
		if m == nil {
			return
		}
		n := *m
		val := p.Value
		switch strings.ToLower(p.Name) {
		case "xref" + string(DB_IN):
			n.XRef = &val
		case "defaultversionid" + string(DB_IN):
			n.DefVID = &val
		default:
			return
		}
		putRow(tx, db.metas, n.SID, &n)
	}
}

// FullTreeXref trigger
func (tx *memTx) propDeleted(p *propRow) {
	if !p.isOwn() || p.Type != ENTITY_META {
		return
	}
	db := tx.db
	m := db.metas[p.SID]

	switch strings.ToLower(p.Name) {
	case "xref" + string(DB_IN):
		if m != nil {
			n := *m
			n.XRef = nil
			putRow(tx, db.metas, n.SID, &n)
		}
		tx.recalcUsesXref(p.RegSID)

	case "defaultversionid" + string(DB_IN):
		if m != nil {
			n := *m
			n.DefVID = nil
			putRow(tx, db.metas, n.SID, &n)
		}
	}
}

// Only ever turns the flag off, like the SQL triggers do
func (tx *memTx) recalcUsesXref(regSID string) {
	db := tx.db
	r := db.regs[regSID]
	if r == nil || !r.UsesXref {
		return
	}

	for _, m := range db.metas {
		if m.RegSID == regSID && m.XRef != nil {
			return
		}
	}

	n := *r
	n.UsesXref = false
	putRow(tx, db.regs, n.SID, &n)
}

func (tx *memTx) propsOf(sid string) []*propRow {
	m := tx.db.props[sid]
	res := make([]*propRow, 0, len(m))
	for _, p := range m {
		res = append(res, p)
	}
	sort.Slice(res, func(i, j int) bool {
		a, b := strings.ToLower(res[i].Name), strings.ToLower(res[j].Name)
		return a < b || (a == b && res[i].Name < res[j].Name)
	})
	return res
}

func (tx *memTx) delPropsWhere(sid string, fn func(*propRow) bool) int {
	count := 0
	for _, p := range tx.propsOf(sid) {
		if fn(p) && tx.delProp(sid, strings.ToLower(p.Name)) {
			count++
		}
	}
	return count
}

func (tx *memTx) delAllProps(sid string) {
	tx.delPropsWhere(sid, func(*propRow) bool { return true })
}

func (tx *memTx) childEnts(parentSID string) []*entRow {
	m := tx.db.entByParent[parentSID]
	res := make([]*entRow, 0, len(m))
	for _, e := range m {
		res = append(res, e)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].SID < res[j].SID })
	return res
}

func (tx *memTx) entsOfReg(regSID string) []*entRow {
	m := tx.db.entByReg[regSID]
	res := make([]*entRow, 0, len(m))
	for _, e := range m {
		res = append(res, e)
	}
	return res
}

// Delete cascades - these mirror the BEFORE DELETE triggers in init.sql

func (tx *memTx) deleteRegistry(sid string) bool {
	db := tx.db
	if db.regs[sid] == nil {
		return false
	}

	for _, g := range db.groups {
		if g.RegSID == sid {
			tx.deleteGroup(g.SID)
		}
	}

	if delRow(tx, db.models, sid) {
		for _, me := range db.modelEnts {
			if me.RegSID == sid {
				tx.deleteModelEntity(me.SID)
			}
		}
	}

	ents := tx.entsOfReg(sid)
	for _, e := range ents {
		tx.delAllProps(e.SID)
	}
	for _, e := range ents {
		tx.delEnt(e.SID)
	}

	delRow(tx, db.regs, sid)
	return true
}

func (tx *memTx) deleteModelEntity(sid string) bool {
	db := tx.db
	if !delRow(tx, db.modelEnts, sid) {
		return false
	}

	for _, g := range db.groups {
		if g.ModelSID == sid {
			tx.deleteGroup(g.SID)
		}
	}
	for _, r := range db.resources {
		if r.ModelSID == sid {
			tx.deleteResource(r.SID)
		}
	}
	return true
}

func (tx *memTx) deleteGroup(sid string) bool {
	db := tx.db
	if !delRow(tx, db.groups, sid) {
		return false
	}

	for _, e := range tx.childEnts(sid) {
		if db.resources[e.SID] != nil {
			tx.deleteResource(e.SID)
		}
	}
	tx.delAllProps(sid)
	tx.delEnt(sid)
	return true
}

func (tx *memTx) deleteResource(sid string) bool {
	db := tx.db
	res := db.resources[sid]
	if res == nil {
		return false
	}

	// ResourcesTrigger: clear the state mirrored into xref sources
	srcMetas := []*metaRow{}
	for _, m := range db.metas {
		if m.RegSID == res.RegSID && m.XRef != nil && *m.XRef == res.XID {
			srcMetas = append(srcMetas, m)
		}
	}
	sort.Slice(srcMetas, func(i, j int) bool {
		return srcMetas[i].SID < srcMetas[j].SID
	})
	for _, m := range srcMetas {
		tx.delPropsWhere(m.SID, func(p *propRow) bool {
			return p.IsXrefPropCopy
		})
		for _, c := range tx.childEnts(m.ResSID) {
			tx.delPropsWhere(c.SID, func(p *propRow) bool {
				return p.IsXrefVerCopy
			})
			if c.IsXrefVerCopy {
				tx.delEnt(c.SID)
			}
		}
		tx.delPropsWhere(m.ResSID, func(p *propRow) bool {
			return p.IsDefaultVerCopy
		})
	}

	for _, m := range db.metas {
		if m.ResSID == sid {
			delRow(tx, db.metas, m.SID)
		}
	}
	kids := tx.childEnts(sid)
	for _, e := range kids {
		if db.versions[e.SID] != nil {
			tx.deleteVersion(e.SID)
		}
	}

	tx.delAllProps(sid)
	for _, e := range kids {
		tx.delAllProps(e.SID)
	}
	tx.delEnt(sid)
	for _, e := range kids {
		tx.delEnt(e.SID)
	}

	delRow(tx, db.resources, sid)
	tx.recalcUsesXref(res.RegSID)
	return true
}

func (tx *memTx) deleteVersion(sid string) bool {
	db := tx.db
	if !delRow(tx, db.versions, sid) {
		return false
	}
	delRow(tx, db.contents, sid)
	tx.delAllProps(sid)
	tx.delEnt(sid)
	return true
}
