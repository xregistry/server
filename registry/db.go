package registry

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
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
	_ "github.com/go-sql-driver/mysql"
	. "github.com/xregistry/server/common"
)

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

// Holds info about the current transaction. In a lot of ways this is similar
// to golang's Context in that it holds other info related to the current
// changes that are going on. Maybe one day convert this to a Context where
// Tx is just as apsect of it.
type Tx struct {
	tx            *sql.Tx
	Registry      *Registry
	CreateTime    string // use for entity timestamps too
	User          string
	RequestInfo   *RequestInfo
	ClearFullTree bool

	// Cache of entities this Tx is dealing with. Things can get funky if
	// we have more than one instance of the same entity in memory.
	// TODO DUG expand this to save all types, not just Versions.
	// Also, consider having Commit() just automatically call ValidateAndSave
	// for all entities in the Tx - then people don't need to call save
	// explicitly
	Cache map[string]*Entity // e.Path

	// For debugging
	uuid  string   // just a unique ID for the TXs map key
	stack []string // Stack at time NewTX
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
	return fmt.Sprintf("Tx: sql.tx: %s, Registry: %s", txStr, regStr)
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

	t, err := DB.BeginTx(context.Background(),
		&sql.TxOptions{sql.LevelReadCommitted, false})
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
	TXsMutex.Lock()
	TXs[tx.uuid] = tx
	TXsMutex.Unlock()
	return nil
}

func (tx *Tx) RefreshFullTree() {
	return
	if tx.ClearFullTree {
		inDo = true
		tx.ClearFullTree = false

		Do(tx, `DELETE FROM FullTreeTable`)
		// Do(tx, `ALTER TABLE FullTreeTable DISABLE KEYS`)
		Do(tx, `INSERT INTO FullTreeTable SELECT * FROM FullTree`)
		// Do(tx, `ALTER TABLE FullTreeTable ENABLE KEYS`)
		/*
					Do(tx, `TRUNCATE TABLE FullTreeTable ;
			            ALTER TABLE FullTreeTable DISABLE KEYS ;
			            INSERT INTO FullTreeTable SELECT * FROM FullTree ;
			            ALTER TABLE FullTreeTable ENABLE KEYS `)
		*/

		/*
			log.Printf("IsCacheDirty: %v", tx.IsCacheDirty())

			log.Printf("\n=================")
			res, _ := tx.Registry.Query("SELECT EntitySID,PropName from Props order by EntitySID")
			for _, r := range res {
				s := *(r[0].(*any))
				p := *(r[1].(*any))
				log.Printf("%v %v", string(s.([]byte)), string(p.([]byte)))
			}

			log.Printf("\n")
			res, _ = tx.Registry.Query("SELECT Path,PropName from FullTree order by Path")
			for _, r := range res {
				s := *(r[0].(*any))
				p := *(r[1].(*any))
				log.Printf("%v %v", string(s.([]byte)), string(p.([]byte)))
			}
		*/

		/*
			Do(tx, `DELETE FROM FullTreeTable`)

			// log.Printf("Table after delete")
			res := tx.Registry.Query("SELECT Path,PropName from FullTree")
			log.Printf("len(res): %d", len(res))
			for _, r := range res {
				s := *(r[0].(*any))
				p := *(r[1].(*any))
				log.Printf("%v %v", string(s.([]byte)), string(p.([]byte)))
			}
		*/
		/*
				res = tx.Registry.Query("SELECT count(*) FROM FullTreeTable")
				for _, r := range res {
					s := *(r[0].(*any))
					log.Printf("count: %v", s.(int64))
				}
			Do(tx, `INSERT INTO FullTreeTable SELECT * FROM FullTree`)
		*/
		inDo = false
	}
}

func (tx *Tx) DumpCache() {
	log.Printf("==== CACHE =====")
	for _, path := range tx.Cache {
		log.Printf("- %s", path)
	}
}

func (tx *Tx) EraseCache() {
	tx.Cache = map[string]*Entity{}
}

func (tx *Tx) AddToCache(e *Entity) {
	PanicIf(IsNil(e.Self), "Self is nil, %s/%s", e.Singular, e.UID)
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

func (tx *Tx) Validate(info *RequestInfo) {
	/*
		if info != nil {
			log.Printf("--- %s %s", info.OriginalRequest.Method, info.OriginalPath)
		} else {
			log.Printf("---")
		}
	*/

	// Make sure we've saved everything in the cache before we generate
	// the results. If the stack isn't shown, enable it in entity.SetNewObject
	PanicIf(tx.IsCacheDirty(), "Unwritten stuff in cache")

	if tx.Registry != nil {
		PanicIf(tx.Registry.Model.GetChanged(), "Unwritten model")
	}

	// At one point we almost called a ValidateResources type of func to
	// double check everthing is ok. We shouldn't need to, but something
	// to think about if things get complicated
	/*
		if xErr := ValidateResources(tx); xErr != nil {
			return xErr
		}

		// Check again just to be sure ValidateResources didn't mess up
		PanicIf(tx.IsCacheDirty(), "Unwritten stuff in cache")
	*/
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

func (tx *Tx) WriteCache(force bool) *XRError {
	for _, e := range tx.Cache {
		PanicIf(!force && e.NewObject != nil, "Entity %s/%q not saved",
			e.Singular, e.UID)
		if !force && e.NewObject != nil {
			log.Printf("%s: %s", e.Singular, e.UID)
			ShowStack()
		}
		if xErr := e.ValidateAndSave(); xErr != nil {
			return xErr
		}
	}
	return nil
}

// Only call from tests
func (tx *Tx) SaveCommitRefresh() *XRError {
	// savedCache := maps.Clone(tx.Cache)

	if xErr := tx.WriteCache(true); xErr != nil {
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

func (tx *Tx) SaveAllAndCommit() *XRError {
	if xErr := tx.WriteCache(true); xErr != nil {
		return xErr
	}

	tx.Validate(nil)

	return tx.Commit()
}

func (tx *Tx) Commit() *XRError {
	// ShowStack()
	if tx.tx == nil {
		return nil
	}

	if xErr := tx.WriteCache(true); xErr != nil {
		return xErr
	}

	Must(tx.tx.Commit())

	TXsMutex.Lock()
	delete(TXs, tx.uuid)
	TXsMutex.Unlock()
	tx.tx = nil
	tx.CreateTime = ""
	tx.Cache = nil
	tx.uuid = ""

	return nil
}

func (tx *Tx) Rollback() *XRError {
	if tx == nil || tx.tx == nil {
		return nil
	}
	err := tx.tx.Rollback()
	Must(err)

	TXsMutex.Lock()
	delete(TXs, tx.uuid)
	TXsMutex.Unlock()
	tx.tx = nil
	tx.CreateTime = ""
	tx.Cache = nil
	tx.uuid = ""

	return nil
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

func Query(tx *Tx, cmd string, args ...interface{}) *Result {
	/*
		if strings.Index(cmd, "FullTree") >= 0 {
			tx.RefreshFullTree()
		}
	*/

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
	PanicIf(xErr != nil, "Error Prepping query (%s): %s\n", cmd, xErr)
	defer ps.Close()

	rows, err := ps.Query(args...)
	if doTime {
		qTime = time.Now()
	}
	PanicIf(err != nil, "Error querying DB(%s)(%v)->%s\n", cmd, args, err)

	colTypes, err := rows.ColumnTypes()
	PanicIf(err != nil, "Error querying DB(%s)(%v)->%s\n", cmd, args, err)

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

var inDo = false

func doCount(tx *Tx, cmd string, args ...interface{}) int {
	log.VPrintf(4, "doCount: %q args: %v", cmd, args)

	if !inDo {
		if strings.Index(cmd, "INSERT") >= 0 ||
			strings.Index(cmd, "DELETE") >= 0 ||
			strings.Index(cmd, "REPLACE") >= 0 {
			tx.ClearFullTree = true
		}

		/*
			if tx.ClearFullTree && strings.Index(cmd, "FullTree") >= 0 {
				inDo = true
				tx.RefreshFullTree()
				inDo = false
			}
		*/
	}

	ps, xErr := tx.Prepare(cmd)
	PanicIf(xErr != nil, "CMD: %q args: %v  err: %s", cmd, args, xErr)
	defer ps.Close()

	result, err := ps.Exec(args...)
	PanicIf(err != nil, "doCount:Error DB(%s)->%s\n", SubQuery(cmd, args), err)

	count, _ := result.RowsAffected()
	return int(count)
}

func Do(tx *Tx, cmd string, args ...interface{}) {
	doCount(tx, cmd, args...)
}

func DoOne(tx *Tx, cmd string, args ...interface{}) {
	count := doCount(tx, cmd, args...)

	PanicIf(count != 1, "DoOne:Error DB(%s) didn't change exactly 1 row(%d)",
		SubQuery(cmd, args), count)
}

func DoZeroOne(tx *Tx, cmd string, args ...interface{}) {
	count := doCount(tx, cmd, args...)

	PanicIf(count != 0 && count != 1,
		"DoOne:Error DB(%s) didn't change exactly 0/1 row(%d)",
		SubQuery(cmd, args), count)
}

func DoOneTwo(tx *Tx, cmd string, args ...interface{}) {
	count := doCount(tx, cmd, args...)

	PanicIf(count != 1 && count != 2,
		"DoOne:Error DB(%s) didn't change exactly 1/2 row(%d)",
		SubQuery(cmd, args), count)
}

func DoZeroTwo(tx *Tx, cmd string, args ...interface{}) {
	count := doCount(tx, cmd, args...)
	PanicIf(count != 0 && count != 2,
		"DoOne:Error DB(%s) didn't change exactly 0/2 row(%d)",
		SubQuery(cmd, args), count)
}

func DoCount(tx *Tx, num int, cmd string, args ...interface{}) {
	log.VPrintf(4, "DoCount: %s", cmd)
	count := doCount(tx, cmd, args...)

	PanicIf(count != num,
		"DoOne:Error DB(%s) didn't change exactly %d row(%d)",
		SubQuery(cmd, args), num, count)
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
