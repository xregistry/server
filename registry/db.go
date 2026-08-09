package registry

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"os"
	"reflect"
	"regexp"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	log "github.com/duglin/dlog"
	"github.com/go-sql-driver/mysql"
	. "github.com/xregistry/server/common"
)

// MySQL error numbers we treat as safe/expected to retry the whole HTTP
// request for (see isRetryableDBErr()/ServeHTTP's retry loop) rather than
// as a hard failure - both only ever happen because two Txs' row locks
// (see entity.go's FOR_WRITE "FOR UPDATE" fetches) genuinely collided,
// not because of a coding bug.
const (
	mysqlErrLockDeadlock    = 1213 // ER_LOCK_DEADLOCK
	mysqlErrLockWaitTimeout = 1205 // ER_LOCK_WAIT_TIMEOUT
)

// isRetryableDBErr inspects a recovered panic value (or a plain error) and
// reports whether it's a MySQL deadlock/lock-wait-timeout - the only
// conditions ServeHTTP's per-request retry loop should transparently
// retry on a fresh Tx. Everything else (syntax errors, bugs, connection
// loss, etc.) is NOT retryable and should keep surfacing exactly as it
// does today (500 via the outer recover()).
//
// This codebase's Query()/doCount()/etc. always panic via
// Must()/PanicIf()/Panicf() (see common/utils.go), which panic with a
// formatted STRING (fmt.Sprintf(msg, args...)), not the original *error*
// value - so the underlying *mysql.MySQLError is normally unwrappable
// from the recovered panic value. Try errors.As() first (in case a
// caller ever panics with the raw error directly), then fall back to
// matching the well-known MySQL error text embedded in that string,
// which is the case that matters in practice here.
func isRetryableDBErr(v any) bool {
	if err, ok := v.(error); ok {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			return mysqlErr.Number == mysqlErrLockDeadlock ||
				mysqlErr.Number == mysqlErrLockWaitTimeout
		}
	}

	msg := fmt.Sprint(v)
	return strings.Contains(msg, "Error 1213") ||
		strings.Contains(msg, "Error 1205")
}

var DB *sql.DB
var DB_Name = ""
var DB_InitFunc func()

var DBUSER = "root"
var DBHOST = "localhost"
var DBPORT = "3306"
var DBPASSWORD = "password"

// TODO load these from a config file
func init() {
	if tmp := os.Getenv("DBUSER"); tmp != "" {
		DBUSER = tmp
	}
	if tmp := os.Getenv("DBPASSWORD"); tmp != "" {
		DBPASSWORD = tmp
	}
	if tmp := os.Getenv("DBHOST"); tmp != "" {
		DBHOST = tmp
	}
	if tmp := os.Getenv("DBPORT"); tmp != "" {
		DBPORT = tmp
	}
}

// Active transaction - mainly for debugging and testing
var TXs = map[string]*Tx{}
var TXsMutex = sync.RWMutex{}

func DumpTXs() {
	// Only show info if there are active Txs
	if len(TXs) == 0 {
		return
	}

	count := 1
	TXsMutex.RLock()
	for _, t := range TXs {
		log.Printf("NewTx Stack %d:", count)
		for _, s := range t.stack {
			log.Printf("  %s", s)
		}
		count++
	}
	TXsMutex.RUnlock()

	// Show threads/processes
	pprof.Lookup("goroutine").WriteTo(PProfFilter, 1)

	log.Printf("==========================")
	log.Printf("")
	PProfFilter.count = 0
	PProfFilter.inSection = false
	PProfFilter.buffer.Reset()
}

var PProfFilter = &FilterPProf{}

type FilterPProf struct {
	buffer    bytes.Buffer
	count     int
	inSection bool
}

// Extract func name and files info
var fpRE = regexp.MustCompile(`^#\s+[^\s]+\s+[^.]*.[^.]*\.([^+]*)\+[^/]*.*/(.*)$`)

// When dumping all processes, filter out the ones that aren't running our
// code and only show lines of interest to keep it small
func (fp *FilterPProf) Write(p []byte) (n int, err error) {
	for _, b := range p {
		if b == '\n' {
			line := fp.buffer.String()
			fp.buffer.Reset()

			if strings.Contains(line, "xreg-github") &&
				!strings.Contains(line, "(*Server).Serve+") &&
				!strings.Contains(line, "(*Server).Serve.") &&
				!strings.Contains(line, "TestMain") {

				if !fp.inSection {
					fp.inSection = true
					fp.count++
					log.Printf("Thread %d:", fp.count)
				}

				line = fpRE.ReplaceAllString(line, `  $1   $2`)
				log.Printf(line)
			} else {
				fp.inSection = false
			}
		} else {
			fp.buffer.WriteByte(b)
		}
	}
	return len(p), nil
}

// resourceValidation is one entry in Tx.ResourcesToValidate - it carries
// the flags Resource.ValidateResource() needs, merged across however
// many times AddResourceToValidate() got called for the same Resource
// within one Tx (see AddResourceToValidate()'s doc comment for the
// merge policy).
type resourceValidation struct {
	r               *Resource
	onlyMetaChanged bool
	force           bool
}

// Holds info about the current transaction. In a lot of ways this is similar
// to golang's Context in that it holds other info related to the current
// changes that are going on. Maybe one day convert this to a Context where
// Tx is just as apsect of it.
type Tx struct {
	tx          *sql.Tx
	Registry    *Registry
	CreateTime  string // use for entity timestamps too
	User        string
	RequestInfo *RequestInfo
	Locked      bool // no more writes allowed!
	Validated   bool // just to make sure it's not called more than once

	// Cache of entities this Tx is dealing with. Things can get funky if
	// we have more than one instance of the same entity in memory.
	// TODO DUG expand this to save all types, not just Versions.
	// Also, consider having Commit() just automatically call ValidateAndSave
	// for all entities in the Tx - then people don't need to call save
	// explicitly
	Cache map[string]*Entity // e.Path

	// List of Group XIDs of all the Groups we need to run verfication on
	// w.r.t. constraints. This isn't always the same as "groups that changed"
	// since it's possible only a Resource change could break a constraint.
	GroupsToValidate map[string]*Group

	// Resources (keyed by DbSID) that need Resource.ValidateResource()
	// (re-)run before this Tx's results are visible - including their
	// default-version-copy cascade and xref fan-out.
	ResourcesToValidate map[string]*resourceValidation

	// Snapshot of the batch Registry.Validate()'s drain loop is
	// CURRENTLY iterating over (keyed by DbSID), exposed so
	// Resource.runCascade() can tell whether an xref TARGET it's about
	// to fan out to is itself still pending in the very same batch
	// (i.e. hasn't started its own ValidateResource() yet) - see
	// runCascade()'s "skip our own insert if the target is still
	// pending" optimization. Deliberately separate from
	// ResourcesToValidate above: Registry.Validate() swaps that field
	// to a fresh empty map before draining a batch (so any NEW marks
	// added mid-batch, e.g. via EnsureLatest(), safely land in the next
	// iteration instead of racing the in-progress range over the old
	// map) - so ResourcesToValidate itself is never a reliable way to
	// ask "is X still pending in the batch currently being drained".
	// Set for the duration of one drain-loop batch only; nil otherwise.
	ResourcesValidatingBatch map[string]bool

	// For debugging
	uuid   string   // just a unique ID for the TXs map key
	stack  []string // Stack at time NewTX
	connID int64    // MySQL CONNECTION_ID() this Tx is bound to
}

func (tx *Tx) IsOpen() bool {
	return tx.tx != nil
}

func (tx *Tx) String() string {
	regStr := "<none>"
	if tx.Registry != nil {
		regStr = tx.Registry.DbSID
	}

	txStr := "<none>"
	if tx.tx != nil {
		txStr = "<set>"
	}
	return fmt.Sprintf("tx: sql.tx: %s, Registry: %s", txStr, regStr)
}

func NewTx() (*Tx, *XRError) {
	log.VPrintf(4, ">Enter: NewTx")
	defer log.VPrintf(4, "<exit: NewTx")

	tx := &Tx{}
	xErr := tx.NewTx()
	if xErr != nil {
		return nil, xErr
	}
	return tx, nil
}

// It's ok for this to be called multiple times for the same Tx just to
// make sure we have an active transaction - it's a no-op at that point
func (tx *Tx) NewTx() *XRError {
	log.VPrintf(4, ">Enter: tx.NewTx")
	defer log.VPrintf(4, "<Exit: tx.NewTx")

	if DB == nil {
		if DB_Name == "" {
			return NewXRError("server_error", "/").SetDetail("No DB_Name set.")
		}
		xErr := OpenDB(DB_Name)
		if xErr != nil {
			return xErr
		}
	}

	if tx.tx != nil {
		return nil
	}

	// REPEATABLE READ (InnoDB's default) rather than READ COMMITTED: it
	// gives every plain (non-locking) SELECT in this Tx one consistent
	// snapshot/read-view established at the transaction's first read -
	// effectively "snapshot at tx start" for our purposes - while
	// SELECT ... FOR UPDATE reads (see entity.go's FOR_WRITE fetches)
	// still always see latest-committed data and take real row locks.
	// This combination is what makes the FOR_READ/FOR_WRITE distinction
	// in Entity.AccessMode actually mean something at the DB level.
	t, err := DB.BeginTx(context.Background(),
		&sql.TxOptions{sql.LevelRepeatableRead, false})
	if err != nil {
		DB = nil
		return NewXRError("server_error", "/").SetDetail(err.Error() + ".")
		// panic("Error talking to the DB: %s", err)
	}

	tx.tx = t
	tx.CreateTime = time.Now().UTC().Format(time.RFC3339Nano)
	tx.Cache = map[string]*Entity{}
	tx.uuid = NewUUID()
	tx.stack = GetStack()

	log.VPrintf(3, "tx: %s Begin transaction", tx.uuid)

	if log.GetVerbose() > 2 {
		var connID int64
		t.QueryRow("SELECT CONNECTION_ID()").Scan(&connID)
		tx.connID = connID

		var autocommit int
		var isoLevel string
		err := t.QueryRow("SELECT @@autocommit, "+
			"@@session.transaction_isolation").Scan(&autocommit, &isoLevel)
		if err != nil {
			log.Printf("tx: %s connID=%d error checking "+
				"autocommit/isolation: %s", tx.uuid, connID, err)
		} else {
			log.Printf("tx: %s connID=%d autocommit=%d isolation=%s",
				tx.uuid, connID, autocommit, isoLevel)
		}

		log.Printf("tx: %s bound to MySQL CONNECTION_ID=%d", tx.uuid, connID)
	}

	TXsMutex.Lock()
	TXs[tx.uuid] = tx
	TXsMutex.Unlock()

	return nil
}

func (tx *Tx) DumpCache() {
	log.Printf("==== CACHE =====")
	for path, _ := range tx.Cache {
		log.Printf("- %s", path)
	}
}

func (tx *Tx) EraseCache() {
	tx.Cache = map[string]*Entity{}
}

func (tx *Tx) AddToCache(e *Entity) {
	PanicIf(IsNil(e.Self), "tx: %s Self is nil, %s/%s",
		tx.uuid, e.Singular, e.UID)
	tx.Cache[e.Registry.UID+"/"+e.Path] = e
}

func (tx *Tx) RemoveFromCache(e *Entity) {
	// If NewObject is missing or its the same a Ob then we're ok.
	// "same" is ok because it means it was just touched, not really changed

	// TODO turn this off when in prod (the maps.Equals probably isn't too
	// expensive, but it's not free
	if e.NewObject != nil && !maps.Equal(e.Object, e.NewObject) {
		log.Printf("OldObject:\n%s", ToJSON(e.Object))
		log.Printf("NewObject:\n%s", ToJSON(e.NewObject))
		e.ShowStack()
		panic(e.Path + " is dirty")
	}
	delete(tx.Cache, e.Registry.UID+"/"+e.Path)
}

func (tx *Tx) Lock() {
	tx.Locked = true
}

func (tx *Tx) IsLocked() bool {
	return tx.Locked
}

// Validate does this Tx's own bookkeeping/sanity-checks, then delegates
// the actual (registry-wide) Resource/Group validation to
// Registry.Validate() - that logic belongs at the Registry level, not
// here, even though the pending-work lists it drains (ResourcesToValidate/
// GroupsToValidate, above) are themselves Tx-scoped.
func (tx *Tx) Validate(info *RequestInfo) *XRError {
	// DUG see if we can add this back in
	// PanicIf(tx.Validated, "Already validated. tx: %p", tx)

	tx.Validated = true

	/*
		if info != nil {
			log.Printf("--- %s %s", info.OriginalRequest.Method, info.OriginalPath)
		} else {
			log.Printf("---")
		}
	*/

	if tx.Registry != nil {
		PanicIf(tx.Registry.Model.GetChanged(), "tx: %s Unwritten model",
			tx.uuid)

		// Drains any deferred Resource/Group validation - this can
		// still write (e.g. meta.ValidateAndSave()/r.ValidateAndSave()
		// for Resources that were only marked via
		// AddResourceToValidate() rather than validated immediately),
		// so it must run before the cache-dirty assertion below.
		if xErr := tx.Registry.Validate(info); xErr != nil {
			return xErr
		}
	}

	// Make sure we've saved everything in the cache before we generate
	// the results. If the stack isn't shown, enable it in entity.SetNewObject
	PanicIf(tx.IsCacheDirty(), "tx: %s Unwritten stuff in cache", tx.uuid)

	return nil
}

func (tx *Tx) IsCacheDirty() bool {
	dirty := false
	for _, e := range tx.Cache {
		if len(e.NewObject) != 0 {
			log.Printf("Dirty: %q", e.Path)
			log.Printf("NewObj:\n%s", ToJSON(e.NewObject))
			log.Printf("Stack for NewObj:")
			for _, s := range e.NewObjectStack {
				log.Printf("  %s", s)
			}
			if len(e.NewObjectStack) == 0 {
				log.Printf("  Enable this via entity.SetNewObject")
			}
			dirty = true
		}
	}
	return dirty
}

func (tx *Tx) DumpDirtyCache() {
	log.Printf("==== DIRTY CACHE =====")
	for path, e := range tx.Cache {
		if len(e.NewObject) != 0 {
			log.Printf("- %s", path)
		}
	}
}

func (tx *Tx) WriteCache(force bool) *XRError {
	for _, e := range tx.Cache {
		if true { // !force {
			if e.NewObject != nil {
				log.Printf("%s: %s", e.Singular, e.UID)
				log.Printf("%s", ToJSON(e.NewObject))
				ShowStack()
			}
			PanicIf(e.NewObject != nil, "tx: %s Entity %s/%q not saved",
				tx.uuid, e.Singular, e.UID)
		}
		if xErr := e.ValidateAndSave(false); xErr != nil {
			return xErr
		}

		// Flush any buffered system-prop changes (SetSystemDBProperty())
		// now, at commit time - a no-op if nothing was buffered. This is
		// what lets several system props on the same entity (e.g.
		// EnsureCompat()'s format/compat validated+reason attrs) get
		// written - and, if relevant, the default-version cascade
		// re-run - at most ONCE per entity per Tx, instead of once per
		// individual SetSystemDBProperty() call.
		e.SaveSystemProps()
	}
	return nil
}

// FlushSystemProps flushes any system-prop changes buffered on any
// entity currently in this Tx's cache (see Entity.SetSystemDBProperty()/
// SaveSystemProps()). Unlike WriteCache()'s own flush (which only
// happens at Commit() time), this can be called earlier - e.g. right
// after EnsureCompat() finishes setting several system props across
// possibly-multiple Versions - so the buffered values are visible to
// the response that gets serialized later in the same request, well
// before the Tx actually commits.
// DUG TODO THIS NEEDS TO BE DONE ON A PER RESOURCE BASIS NOT GLOBAL
func (tx *Tx) FlushSystemProps() {
	for _, e := range tx.Cache {
		e.SaveSystemProps()
	}
}

func (tx *Tx) AddGroupToValidate(g *Group) {
	if tx.GroupsToValidate == nil {
		tx.GroupsToValidate = map[string]*Group{}
	}
	tx.GroupsToValidate[g.XID] = g
}

// AddResourceToValidate records that r needs Resource.ValidateResource()
// (re-)run before this Tx's results are used, without running it
// immediately. It's safe/cheap to call this many times for the same
// Resource within one Tx (e.g. once per Version save, once per
// buffered-system-prop flush) - Registry.Validate() collapses
// all of them into a single ValidateResource() run using the final
// state, mirroring AddGroupToValidate() above. If r is already marked,
// the onlyMetaChanged/force flags are merged with whatever's already
// there rather than overwritten: onlyMetaChanged stays true only if
// every mark agreed (AND), while force becomes true if any mark asked
// for it (OR) - so the eventual single run is at least as thorough as
// every individual request made of it. A no-op if r is nil.
func (tx *Tx) AddResourceToValidate(r *Resource, onlyMetaChanged bool, force bool) {
	if r == nil {
		return
	}
	if tx.ResourcesToValidate == nil {
		tx.ResourcesToValidate = map[string]*resourceValidation{}
	}
	if existing, ok := tx.ResourcesToValidate[r.DbSID]; ok {
		existing.onlyMetaChanged = existing.onlyMetaChanged && onlyMetaChanged
		existing.force = existing.force || force
		return
	}
	tx.ResourcesToValidate[r.DbSID] = &resourceValidation{
		r:               r,
		onlyMetaChanged: onlyMetaChanged,
		force:           force,
	}
}

// Only call from tests
func (tx *Tx) SaveCommitRefresh() *XRError {
	// savedCache := maps.Clone(tx.Cache)

	if xErr := tx.SaveAll(); xErr != nil {
		return xErr
	}
	tx.Validate(nil)

	if xErr := tx.Commit(); xErr != nil {
		return xErr
	}

	/*
		// Reload all cached entities so the tests don't need to do it themselves
		log.Printf("cache size: %d", len(tx.Cache))
		for _, e := range tx.Cache {
			log.Printf("  Refresh: %s/%s", e.Singular, e.UID)
			e.Refresh()
		}
	*/

	return nil
}

func (tx *Tx) SaveAll() *XRError {
	// Drain any deferred Resource/Group validation (see
	// Tx.AddResourceToValidate()) BEFORE flushing the cache below -
	// otherwise a Resource/Meta/Version this drain still needs to
	// update (e.g. resolving "defaultversionid" via EnsureLatest())
	// would still show up dirty and trip WriteCache()'s "not saved"
	// sanity check. tx.Validate() (called later, by callers like
	// SaveAllAndCommit()) re-runs this drain too, but it's a cheap
	// no-op the second time since everything's already drained.
	if tx.Registry != nil {
		if xErr := tx.Registry.Validate(nil); xErr != nil {
			return xErr
		}
	}

	if xErr := tx.WriteCache(true); xErr != nil {
		return xErr
	}
	return nil
}

func (tx *Tx) SaveAllAndCommit() *XRError {
	if xErr := tx.SaveAll(); xErr != nil {
		return xErr
	}

	if xErr := tx.Validate(nil); xErr != nil {
		return xErr
	}

	return tx.Commit()
}

func (tx *Tx) Commit() *XRError {
	// ShowStack()

	if !tx.IsLocked() {
		// DUG see if we can add this back in
		// panic("Tx isn't locked!!")
	}

	if tx.tx == nil {
		return nil
	}

	tx.Locked = false

	if xErr := tx.WriteCache(true); xErr != nil {
		tx.Rollback()
		return xErr
	}

	Must(tx.tx.Commit())
	log.VPrintf(3, "tx: %s Committed", tx.uuid)

	tx.Clear()
	return nil
}

func (tx *Tx) Rollback() *XRError {
	if tx == nil || tx.tx == nil {
		return nil
	}

	err := tx.tx.Rollback()
	Must(err)
	log.VPrintf(3, "tx: %s Rolled Back", tx.uuid)

	tx.Clear()
	return nil
}

// Clears all variables - the tx MUST be Committed or Rolledback prior to this.
// This is just a bookkeeping func
func (tx *Tx) Clear() {
	TXsMutex.Lock()
	delete(TXs, tx.uuid)
	TXsMutex.Unlock()
	tx.tx = nil
	// tx.Registry = nil // new
	tx.CreateTime = ""
	// tx.RequestInfo = nil // new
	// tx.Locked = false    // new
	// tx.Validated = false // new
	tx.EraseCache()
	tx.GroupsToValidate = nil
	tx.ResourcesToValidate = nil
	tx.ResourcesValidatingBatch = nil
	tx.uuid = ""
	tx.stack = nil
}

func (tx *Tx) Conditional(xErr *XRError) *XRError {
	if xErr == nil {
		return tx.Commit()
	}
	return tx.Rollback()
}

func (tx *Tx) Prepare(query string) (*sql.Stmt, *XRError) {
	// If the current Tx is closed, create a new one
	if tx.tx == nil {
		xErr := tx.NewTx()
		if xErr != nil {
			return nil, xErr
		}
	}
	ps, err := tx.tx.Prepare(query)
	if err != nil {
		return nil, NewXRError("server_error", "/").SetDetail(err.Error() + ".")
	}

	return ps, nil
}

func (tx *Tx) AddRegistry(r *Registry) { tx.AddToCache(&r.Entity) }
func (tx *Tx) GetRegistry(rID string) *Registry {
	entry, ok := tx.Cache[rID]
	if !ok {
		return nil
	}
	return (entry.Self).(*Registry)
}

func (tx *Tx) AddGroup(g *Group) { tx.AddToCache(&g.Entity) }
func (tx *Tx) GetGroup(r *Registry, plural string, gID string) *Group {
	entry, ok := tx.Cache[r.Registry.UID+"/"+plural+"/"+gID]
	if !ok {
		return nil
	}
	return (entry.Self).(*Group)
}

func (tx *Tx) AddResource(r *Resource) { tx.AddToCache(&r.Entity) }
func (tx *Tx) GetResource(g *Group, plural string, rID string) *Resource {
	entry, ok := tx.Cache[g.Registry.UID+"/"+g.Path+"/"+plural+"/"+rID]
	if !ok {
		return nil
	}
	return (entry.Self).(*Resource)
}

func (tx *Tx) AddMeta(m *Meta) { tx.AddToCache(&m.Entity) }
func (tx *Tx) GetMeta(r *Resource) *Meta {
	entry, ok := tx.Cache[r.Registry.UID+"/"+r.Path+"/meta"]
	if !ok {
		return nil
	}
	return (entry.Self).(*Meta)
}

func (tx *Tx) AddVersion(v *Version) { tx.AddToCache(&v.Entity) }
func (tx *Tx) GetVersion(r *Resource, vID string) *Version {
	entry, ok := tx.Cache[r.Registry.UID+"/"+r.Path+"/versions/"+vID]
	if !ok {
		return nil
	}
	return (entry.Self).(*Version)
}

type Result struct {
	tx       *Tx
	sqlRows  *sql.Rows
	colTypes []reflect.Type
	Data     []*any // One row
	TempData []any
	Reuse    bool

	AllRows [][]*any
}

func (r *Result) Close() {
	if r == nil {
		return
	}

	if r.Data == nil {
		// Already done
		return
	}

	if r.tx != nil {
		r.tx = nil
	}

	if r.sqlRows != nil {
		r.sqlRows.Close()
		r.sqlRows = nil
	}

	r.Data = nil
	r.TempData = nil
	r.AllRows = nil
}

func (r *Result) Push() {
	if r.Reuse {
		panic("Already pushed")
	}
	r.Reuse = true
}

func (r *Result) NextRow() []*any {
	if r == nil || r.Data == nil {
		return nil
	}

	if r.Reuse {
		r.Reuse = false
	} else {
		// check for error from PullNextRow
		r.PullNextRow()
	}

	return r.Data
}

func (r *Result) PullNextRow() {
	if r.AllRows == nil || len(r.AllRows) == 0 {
		r.Close()
		return
	}

	r.Data = r.AllRows[0]
	r.AllRows = r.AllRows[1:]

	if log.GetVerbose() > 3 {
		dd := []string{}
		for _, d := range r.Data {
			dVal := reflect.ValueOf(*d)
			if !IsNil(*d) && dVal.Type().String() == "[]uint8" {
				// if reflect.ValueOf(*d).Type().String() == "[]uint8"
				dd = append(dd, string((*d).([]byte)))
			} else {
				dd = append(dd, fmt.Sprintf("%v", *d))
			}
		}
		log.VPrintf(4, "row: %v", dd)
	}
}

func (r *Result) RetrieveAllRowsFromDB() {
	for {
		if r.RetrieveNextRowFromDB() == false {
			break
		}
		r.AllRows = append(r.AllRows, r.Data)
	}
	// When done, technically r.Data contains the last item from the query
	// but it'll be overwritten on the first call to PullNextRow

	// rows.Next() returning false means EITHER "no more rows" (the
	// normal/expected case) OR that iteration stopped early because of a
	// real error (e.g. the connection's transaction was killed as a
	// deadlock victim, or a lock-wait-timeout occurred, mid-scan) -
	// database/sql only surfaces that error via rows.Err(), never from
	// Next() itself. Without this check, a genuine mid-query MySQL error
	// (including ones that silently end this transaction, e.g. via an
	// implicit rollback + a fresh connection-level transaction on the
	// very next statement) would be indistinguishable from a normal,
	// successful, empty/short result set - letting execution proceed as
	// though a "FOR UPDATE" lock had been granted when in fact it never
	// was. Panic so isRetryableDBErr()'s existing deadlock/lock-timeout
	// retry logic (see httpStuff.go) can catch and retry it like any
	// other DB error.
	if r.sqlRows != nil {
		if err := r.sqlRows.Err(); err != nil {
			r.sqlRows.Close()
			r.sqlRows = nil
			Panicf("Error retrieving rows from DB: %s", err)
		}
	}

	// Close the MYSQL query and prepare stmt
	if r.sqlRows != nil {
		r.sqlRows.Close()
		r.sqlRows = nil
	}
}

func (r *Result) RetrieveNextRowFromDB() bool {
	if r.sqlRows == nil {
		panic("sqlRows is nil")
	}
	if r.sqlRows.Next() == false {
		// r.Close()
		return false
	}

	r.TempData = make([]any, len(r.TempData))
	r.Data = make([]*any, len(r.Data))
	for i, _ := range r.TempData {
		r.TempData[i] = new(any)
		r.Data[i] = r.TempData[i].(*any)
	}

	err := r.sqlRows.Scan(r.TempData...) // Can't pass r.Data directly
	if err != nil {
		panic(fmt.Sprintf("Error scanning DB row: %s", err))
		// should return err.  r.Data = nil ; return err..
	}

	// Move data from TempData to Data

	if log.GetVerbose() >= 4 {
		dd := []string{}
		for _, d := range r.Data {
			dVal := reflect.ValueOf(*d)
			if !IsNil(*d) && dVal.Type().String() == "[]uint8" {
				// if reflect.ValueOf(*d).Type().String() == "[]uint8"
				dd = append(dd, string((*d).([]byte)))
			} else {
				dd = append(dd, fmt.Sprintf("%v", *d))
			}
		}
		log.Printf("row: %v", dd)
	}
	return true
}

type queryTime struct {
	count    int
	prepDur  time.Duration
	queryDur time.Duration
	getDur   time.Duration
	totalDur time.Duration
}

var queryTimes = map[string]*queryTime{}
var doTime = os.Getenv("XR_TIMING") != ""

func DumpTimings() string {
	if !doTime {
		return ""
	}

	str := ""
	str += fmt.Sprintf("Count|Prep|Prep Avg|Query|Query Avg|Get|Get Avg|Total|Total Avg|CMD\n")
	for cmd, qt := range queryTimes {
		cmd = strings.ReplaceAll(cmd, "\n", " ")
		cmd = strings.ReplaceAll(cmd, "|", "@")

		str += fmt.Sprintf("%v|%d|%d|%d|%d|%d|%d|%d|%d|%s\n",
			qt.count,
			qt.prepDur, qt.prepDur/time.Duration(qt.count),
			qt.queryDur, qt.queryDur/time.Duration(qt.count),
			qt.getDur, qt.getDur/time.Duration(qt.count),
			qt.totalDur, qt.totalDur/time.Duration(qt.count),
			cmd)
	}

	return str
}

// VerifyLockHeldOrPanic is a hard invariant check against MySQL's own
// lock state (not Go-level bookkeeping, which can't see locks taken by
// the "Family" lock helpers' multi-row FOR UPDATE queries unless every
// call site remembers to record them): it queries
// performance_schema.data_locks (via this Tx's own connection) to
// confirm this Tx's connection actually holds a GRANTED exclusive (X)
// record lock on the Entities row for eSID, right now - not just that
// our own in-Go AccessMode bookkeeping says FOR_WRITE. If, after a few
// short retries (to ride out any performance_schema visibility lag),
// we still see no such lock, we are NOT actually holding the row lock
// we need before Save()'s destructive DELETE+INSERT of Props - this
// must never be allowed to silently proceed, since doing so is exactly
// what let a losing Tx corrupt another Tx's (or its own already
// stale) entity by racing its Props rewrite.
func VerifyLockHeldOrPanic(tx *Tx, eSID string, xid string) {
	if log.GetVerbose() < 3 {
		return
	}
	// need tx.ConnID to be set from NewTx

	const maxAttempts = 5
	var rowsSeen []string
	found := false

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		results := Query(tx, `
			SELECT dl.LOCK_MODE, dl.LOCK_STATUS, dl.LOCK_DATA, th.PROCESSLIST_ID
			FROM performance_schema.data_locks dl
			JOIN performance_schema.threads th
			  ON th.THREAD_ID = dl.THREAD_ID
			WHERE dl.OBJECT_NAME = 'Entities'
			  AND th.PROCESSLIST_ID = ?
			  AND dl.LOCK_TYPE = 'RECORD'
			  AND dl.LOCK_DATA LIKE CONCAT('''', ?, '''%')`,
			tx.connID, eSID)

		rowsSeen = rowsSeen[:0]
		for {
			row := results.NextRow()
			if row == nil {
				break
			}
			lockMode := NotNilString(row[0])
			lockStatus := NotNilString(row[1])
			lockData := NotNilString(row[2])
			mysqlThreadID := *row[3]
			rowsSeen = append(rowsSeen, fmt.Sprintf("mode=%s status=%s data=%s connID=%v",
				lockMode, lockStatus, lockData, mysqlThreadID))
			if lockStatus == "GRANTED" &&
				(lockMode == "X" || lockMode == "X,REC_NOT_GAP" || lockMode == "X,GAP") {
				found = true
			}
		}
		results.Close()

		if found {
			return
		}

		if attempt < maxAttempts {
			time.Sleep(10 * time.Millisecond)
		}
	}

	ShowStack()
	panic(fmt.Sprintf(
		"tx: %s VerifyLockHeldOrPanic(%s eSID=%s): NO GRANTED X row lock "+
			"found for this tx's connID=%d in performance_schema.data_locks "+
			"after %d attempts right before Save()'s destructive DELETE+"+
			"INSERT - rows seen: %v", tx.uuid, xid, eSID, tx.connID,
		maxAttempts, rowsSeen))
}

func Query(tx *Tx, cmd string, args ...interface{}) *Result {
	startTime := time.Time{}
	pTime := time.Time{}
	qTime := time.Time{}
	gTime := time.Time{}

	if doTime {
		startTime = time.Now()
	}

	if log.GetVerbose() >= 4 {
		log.Printf("Query: %s", SubQuery(cmd, args))
	}

	ps, xErr := tx.Prepare(cmd)
	if doTime {
		pTime = time.Now()
	}
	PanicIf(xErr != nil, "tx: %s Error Prepping query (%s): %s\n",
		tx.uuid, cmd, xErr)
	defer ps.Close()

	rows, err := ps.Query(args...)
	if doTime {
		qTime = time.Now()
	}
	PanicIf(err != nil, "tx: %s Error querying DB(%s)(%v)->%s\n",
		tx.uuid, cmd, args, err)

	colTypes, err := rows.ColumnTypes()
	PanicIf(err != nil, "tx: %s Error querying DB(%s)(%v)->%s\n",
		tx.uuid, cmd, args, err)

	result := &Result{
		tx:       tx,
		sqlRows:  rows,
		colTypes: []reflect.Type{},
	}

	for _, col := range colTypes {
		result.colTypes = append(result.colTypes, col.ScanType())
		result.Data = append(result.Data, new(any))
		result.TempData = append(result.TempData, new(any))
	}

	// Download all data. We used to pull from DB on each PullNextRow
	// but mysql doesn't support multiple queries being active in the same Tx
	result.RetrieveAllRowsFromDB()

	if doTime {
		gTime = time.Now()

		qt, ok := queryTimes[cmd]
		if !ok {
			qt = &queryTime{}
			queryTimes[cmd] = qt
		}
		pDiff := pTime.Sub(startTime)
		qDiff := qTime.Sub(pTime)
		gDiff := gTime.Sub(qTime)
		tDiff := gTime.Sub(startTime)

		qt.prepDur += pDiff
		qt.queryDur += qDiff
		qt.getDur += gDiff
		qt.totalDur += tDiff
		qt.count++
	}

	return result
}

func doCount(tx *Tx, cmd string, args ...interface{}) int {
	log.VPrintf(4, "doCount: %q args: %v", cmd, args)

	if tx.IsLocked() {
		ShowStack("Attempting a write when TX is locked - tx: %p", tx)
		panic("Tx is locked!!")
	}

	ps, xErr := tx.Prepare(cmd)
	PanicIf(xErr != nil, "tx:%s CMD: %q args: %v  err: %s",
		tx.uuid, cmd, args, xErr)
	defer ps.Close()

	result, err := ps.Exec(args...)
	if err != nil {
		Panicf("tx: %s doCount: Error DB(%s)->%s\n", tx.uuid,
			SubQuery(cmd, args), err)
	}

	count, _ := result.RowsAffected()
	log.VPrintf(4, "doCount: %d rows", count)
	return int(count)
}

func Do(tx *Tx, cmd string, args ...interface{}) {
	doCount(tx, cmd, args...)
}

func DoOne(tx *Tx, cmd string, args ...interface{}) {
	count := doCount(tx, cmd, args...)

	PanicIf(count != 1, "tx: %s DoOne: Error DB(%s) didn't change "+
		"exactly 1 row(%d)", tx.uuid, SubQuery(cmd, args), count)
}

func DoZeroOne(tx *Tx, cmd string, args ...interface{}) {
	count := doCount(tx, cmd, args...)

	PanicIf(count != 0 && count != 1,
		"tx: %s DoOne: Error DB(%s) didn't change exactly 0/1 row(%d)",
		tx.uuid, SubQuery(cmd, args), count)
}

func DoOneTwo(tx *Tx, cmd string, args ...interface{}) {
	count := doCount(tx, cmd, args...)

	PanicIf(count != 1 && count != 2,
		"tx: %s DoOne: Error DB(%s) didn't change exactly 1/2 row(%d)",
		tx.uuid, SubQuery(cmd, args), count)
}

func DoZeroTwo(tx *Tx, cmd string, args ...interface{}) {
	count := doCount(tx, cmd, args...)
	PanicIf(count != 0 && count != 2,
		"tx: %s DoOne: Error DB(%s) didn't change exactly 0/2 row(%d)",
		tx.uuid, SubQuery(cmd, args), count)
}

func DoCount(tx *Tx, num int, cmd string, args ...interface{}) {
	log.VPrintf(4, "DoCount: %s", cmd)
	count := doCount(tx, cmd, args...)

	PanicIf(count != num,
		"tx: %s DoOne: Error DB(%s) didn't change exactly %d row(%d)",
		tx.uuid, SubQuery(cmd, args), num, count)
}

func DBExists(name string) bool {
	log.VPrintf(3, ">Enter: DBExists %q", name)
	defer log.VPrintf(3, "<Exit: DBExists")
	db, err := sql.Open("mysql",
		DBUSER+":"+DBPASSWORD+"@tcp("+DBHOST+":"+DBPORT+")/")
	PanicIf(err != nil, "Error opening DB: %s", err)
	defer db.Close()

	rows, err := db.Query(`
		SELECT SCHEMA_NAME
		FROM INFORMATION_SCHEMA.SCHEMATA
		WHERE SCHEMA_NAME=?`, name)
	PanicIf(err != nil, "Error querying DB: %s", err)
	defer rows.Close()

	found := rows.Next()
	log.VPrintf(3, "<Exit: found: %v", found)
	return found
}

//go:embed init.sql
var initDB string
var firstTime = true

func OpenDB(name string) *XRError {
	if firstTime {
		log.VPrintf(3, "Open DB: %s:%s", DBHOST, DBPORT)
		firstTime = false
	}

	log.VPrintf(3, ">Enter: OpenDB %q", name)
	defer log.VPrintf(3, "<Exit: OpenDB")

	// DB, err := sql.Open("mysql",
	// DBUSER + ":"+DBPASSWORD+"@tcp(localhost:3306)/")
	var err error

	DB, err = sql.Open("mysql",
		DBUSER+":"+DBPASSWORD+"@tcp("+DBHOST+":"+DBPORT+")/"+name)

	if err != nil {
		DB = nil
		return NewXRError("server_error", "/",
			fmt.Sprintf("Error talking to SQL: %s", err))
	}

	DB_Name = name
	DB.SetMaxOpenConns(5)
	DB.SetMaxIdleConns(5)

	if DB_InitFunc != nil {
		DB_InitFunc()
	}

	return nil
}

func ListDBs() ([]string, *XRError) {
	log.VPrintf(3, ">Enter: ListDBs")
	defer log.VPrintf(3, "<Exit: ListDBs")

	db, err := sql.Open("mysql",
		DBUSER+":"+DBPASSWORD+"@tcp("+DBHOST+":"+DBPORT+")/")
	if err != nil {
		return nil, NewXRError("server_error", "/").SetDetail(err.Error() + ".")
	}
	defer db.Close()

	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return nil, NewXRError("server_error", "/").SetDetail(err.Error() + ".")
	}
	defer rows.Close()

	sysNames := []string{"information_schema", "mysql",
		"performance_schema", "sys"}

	names := []string{}
	for rows.Next() {
		name := ""
		if err := rows.Scan(&name); err != nil {
			return nil, NewXRError("server_error", "/").SetDetail(err.Error() + ".")
		}
		if !ArrayContains(sysNames, name) {
			names = append(names, name)
		}
	}

	return names, nil
}

func CreateDB(name string) error {
	log.VPrintf(3, ">Enter: CreateDB %q", name)
	defer log.VPrintf(3, "<Exit: CreateDB")

	db, err := sql.Open("mysql",
		DBUSER+":"+DBPASSWORD+"@tcp("+DBHOST+":"+DBPORT+")/")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if _, err = db.Exec("CREATE DATABASE " + name); err != nil {
		panic(err)
	}

	if _, err = db.Exec("USE " + name); err != nil {
		panic(err)
	}

	log.VPrintf(3, "Creating DB")

	for _, cmd := range strings.Split(initDB, ";") {
		cmd = strings.TrimSpace(cmd)
		cmd = ReplaceVariables(cmd)
		if cmd == "" {
			continue
		}

		log.VPrintf(4, "CMD: %s", cmd)
		if _, err := db.Exec(cmd); err != nil {
			panic(fmt.Sprintf("Error on: %s\n%s", cmd, err))
		}
	}

	return nil
}

func ReplaceVariables(str string) string {
	if str == "" {
		return str
	}

	vars := [][2]string{
		{"$$", ";"}, // can't use ; in file
		{"$ENTITY_REGISTRY", StrTypes(ENTITY_REGISTRY)},
		{"$ENTITY_GROUP", StrTypes(ENTITY_GROUP)},
		{"$ENTITY_RESOURCE", StrTypes(ENTITY_RESOURCE)},
		{"$ENTITY_META", StrTypes(ENTITY_META)},
		{"$ENTITY_VERSION", StrTypes(ENTITY_VERSION)},
		{"$DB_IN", string(DB_IN)},
		{"$MAX_VARCHAR", fmt.Sprintf("%d", MAX_VARCHAR)},
		{"$MAX_PROPNAME", fmt.Sprintf("%d", MAX_PROPNAME)},
	}

	for _, vs := range vars {
		str = strings.Replace(str, vs[0], vs[1], -1)
	}
	return str
}

func DeleteDB(name string) error {
	log.VPrintf(3, "Deleting DB %q", name)

	db, err := sql.Open("mysql",
		DBUSER+":"+DBPASSWORD+"@tcp("+DBHOST+":"+DBPORT+")/")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec("DROP DATABASE IF EXISTS " + name)
	if err != nil {
		panic(err)
	}
	return nil
}

func SubQuery(query string, args []interface{}) string {
	argNum := 0

	for pos := 0; pos < len(query); pos++ {
		if ch := query[pos]; ch != '?' {
			continue
		}
		if argNum >= len(args) {
			panic(fmt.Sprintf("Extra ? in query at %q", query[pos:]))
		}

		val := fmt.Sprintf("%v", args[argNum])
		query = fmt.Sprintf("%s'%s'%s", query[:pos], val, query[pos+1:])
		pos += len(val) + 1 // one more will be added due to pos++
		argNum++
	}
	if argNum != len(args) {
		panic(fmt.Sprintf("Too many args passed into %q", query))
	}
	return query
}

/*
select * from FullTree
where
  eID in (
    select gID from FullTree
	where PropName='Name' and PropValue='docker.com' and Path='apiProviders/7fbc05b2'
    union select rID from FullTree
	where PropName='Name' and PropValue='docker.com' and Path='apiProviders/7fbc05b2'
	union select vID from FullTree
	where PropName='Name' and PropValue='docker.com' and Path='apiProviders/7fbc05b2'
  )
  order by Path ;


Children:
select ft.* from FullTree as ft where ft.Path like concat((select Path from FullTree where PropValue=4 and LevelNum=2),'/%') order by ft.Path ;

Node+Children:
select ft.* from FullTree as ft where ft.Path like concat((select Path from FullTree where PropValue=4 and LevelNum=2),'%') order by ft.Path ;

Parents:
select ft.* from FullTree as ft where (select Path from FullTree where PropValue=4 and LevelNum=2) like concat(ft.Path, '/%') order by ft.Path;

Node+Parents:
select ft.* from FullTree as ft where (select Path from FullTree where PropValue=4 and LevelNum=2) like concat(ft.Path, '%') order by ft.Path;



NODES + Children:
select ft2.* from FullTree as ft right JOIN FullTree as ft2 on(ft2.Path like concat(ft.Path, '%')) where (ft.PropValue=3 and ft.LevelNum=2) or (ft.PropValue=4 and ft.LevelNum=3) group by ft2.eID,ft2.PropName Order by ft2.Path;

PARENTS (not NODES):
select ft2.* from FullTree as ft right JOIN FullTree as ft2 on(ft.Path like concat(ft2.Path,'/%')) where (ft.PropValue=3 and ft.LevelNum=2) or (ft.PropValue=4 and ft.LevelNum=3) group by ft2.eID,ft2.PropName Order by ft2.Path;

( ( exp1 AND expr2 ...) or ( expr3 AND expr4 ) )
Find IDs that match expr1 OR expr2
SELECT eID FROM FullTree WHERE ( (expr1) OR (expr2) );
SELECT eID FROM FullTree WHERE (Level=2 AND PropName='epoch' && PropValue='4');

Given an ID find all Parents (include original ID)
WITH RECURSIVE cte(eID,ParentID,Path) AS (
  SELECT eID,ParentID,Path FROM Entities
  WHERE eID in (
    -- below find IDs of interes
	SELECT eID FROM FullTree
	  WHERE (PropName='labels.int' AND PropValue=3 AND Level=2)
    -- end of ID selection
  )
  UNION ALL SELECT e.eID,e.ParentID,e.Path FROM Entities AS e
  INNER JOIN cte ON e.eID=cte.ParentID)
SELECT * FROM cte ;

Given an ID find all Leaves (with recursion)
WITH RECURSIVE cte(eID,ParentID,Path) AS (
  SELECT eID,ParentID,Path FROM Entities
  WHERE eID='f91a4ec9'
  UNION ALL SELECT e.eID,e.ParentID,e.Path FROM Entities AS e
    INNER JOIN cte ON e.ParentID=cte.eID)
SELECT eID,ParentID,Path FROM cte
WHERE eID IN (SELECT * FROM Leaves);

Given an ID find all Leaves (w/o recursion)
  Should use IDs instead of Path to pick-up the Registry itself
SELECT e2.eID,e2.ParentID,e2.Path FROM Entities AS e1
RIGHT JOIN Entities AS e2 ON (e2.Path=e1.Path OR e2.Path LIKE
CONCAT(e1.Path,'%')) WHERE e1.eID in (
  -- below finds IDs of interest
  SELECT eID FROM FullTree
  WHERE (PropName='labels.int' AND PropValue=3 AND Level=2)
  -- end of ID selection
  )
AND e2.eID IN (SELECT * from Leaves);

Given an ID, find all leaves, and then find all Parents
-- Finding all parents
WITH RECURSIVE cte(eID,ParentID,Path) AS (
  SELECT eID,ParentID,Path FROM Entities
  WHERE eID in (
    -- below find IDs of interest (finding all leaves)
	SELECT e2.eID FROM Entities AS e1
	RIGHT JOIN Entities AS e2 ON (
	  e2.RegID=e1.RegID AND
	  (e2.Path=e1.Path OR e2.Path LIKE CONCAT(e1.Path,'%'))
	)
	WHERE e1.eID in (
	  -- below finds SeachNodes/IDs of interest
	  -- Add regID into the search
	    SELECT eID FROM FullTree
		WHERE (PropName='labels.int' AND PropValue=3 AND Level=2)
	  -- end of ID selection
	)
	AND e2.eID IN (SELECT * from Leaves)
    -- end of Leaves/ID selection
  )
  UNION ALL SELECT e.eID,e.ParentID,e.Path FROM Entities AS e
  INNER JOIN cte ON e.eID=cte.ParentID)
SELECT * FROM cte ;

(expr1 AND expr2)
WITH RECURSIVE cte(eID,ParentID,Path) AS (
  SELECT eID,ParentID,Path FROM Entities
  WHERE eID in (
    -- below find IDs of interest (finding all leaves)
	-- start of (expr1 and expr2 and expr3)
	SELECT list.eID FROM (
	  SELECT count(*) as cnt,e2.eID,e2.Path FROM Entities AS e1
	  RIGHT JOIN (
	    -- below finds SeachNodes/IDs of interest
	    -- Add regID into the search
	      SELECT eID,Path FROM FullTree
		  WHERE (CONCAT(Abstract,'.',PropName)='myGroups/ress.labels.int')
		  UNION ALL
	      SELECT eID,Path FROM FullTree
		  WHERE (PropName='labels.int' AND PropValue=3 AND Level=2)
		  UNION ALL
		  SELECT eID,Path from FullTree
		  WHERE (PropName='id' AND PropValue='g1' AND Level=1)
	    -- end of ID selection
	  ) as res ON ( res.eID=e1.eID )
	  JOIN Entities AS e2 ON (
	    (e2.Path=res.Path OR e2.Path LIKE CONCAT(res.Path,'%'))
	    AND e2.eID IN (SELECT * from Leaves)
	  ) GROUP BY e2.eID
      -- end of Leaves/ID selection
    ) as list
    WHERE list.cnt=3
	-- end of (expr1 and expr2 and expr3)

	-- ADD the next OR expr here
	UNION
	-- start of expr4
    SELECT list.eID FROM (
      SELECT count(*) as cnt,e2.eID,e2.Path FROM Entities AS e1
      RIGHT JOIN (
        -- below finds SeachNodes/IDs of interest
        -- Add regID into the search
          SELECT eID,Path FROM FullTree
          WHERE (PropName='defaultVersionId' AND PropValue='v1.0' AND Level=2)
        -- end of ID selection
      ) as res ON ( res.eID=e1.eID )
      JOIN Entities AS e2 ON (
        (e2.Path=res.Path OR e2.Path LIKE CONCAT(res.Path,'%'))
        AND e2.eID IN (SELECT * from Leaves)
      ) GROUP BY e2.eID
      -- end of Leaves/ID selection
    ) as list
    WHERE list.cnt=1
  )
  UNION ALL SELECT e.eID,e.ParentID,e.Path FROM Entities AS e
  INNER JOIN cte ON e.eID=cte.ParentID)
SELECT * FROM cte ;
*/
