package registry

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	log "github.com/duglin/dlog"
	"github.com/go-sql-driver/mysql"
	. "github.com/xregistry/server/common"
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
	// MySQL error numbers we treat as safe/expected to retry the whole HTTP
	// request for (see isRetryableDBErr()/ServeHTTP's retry loop) rather than
	// as a hard failure - both only ever happen because two Txs' row locks
	// (see entity.go's FOR_WRITE "FOR UPDATE" fetches) genuinely collided,
	// not because of a coding bug.
	const (
		mysqlErrLockDeadlock    = 1213 // ER_LOCK_DEADLOCK
		mysqlErrLockWaitTimeout = 1205 // ER_LOCK_WAIT_TIMEOUT
	)

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

func prepare(tx *Tx, query string) (*sql.Stmt, *XRError) {
	// If the current Tx is closed, create a new one
	xErr := tx.EnsureTx()
	if xErr != nil {
		return nil, xErr
	}

	ps, err := tx.tx.(*sql.Tx).Prepare(query)
	if err != nil {
		return nil, NewXRError("server_error", "/").SetDetail(err.Error() + ".")
	}

	return ps, nil
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

	if log.HasVerbose("DEBUG_PullNextRow") {
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
}

func (r *Result) retrieveAllRowsFromDB() {
	for {
		if r.retrieveNextRowFromDB() == false {
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

func (r *Result) retrieveNextRowFromDB() bool {
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

	if log.HasVerbose("RetrieveNextRowFromDB") {
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
	startTime := time.Time{}
	pTime := time.Time{}
	qTime := time.Time{}
	gTime := time.Time{}

	if doTime {
		startTime = time.Now()
	}

	if log.IsFuncVerbose() {
		log.Printf("tx: %s Query: %s", tx.uuid, SubQuery(cmd, args))
	}

	ps, xErr := prepare(tx, cmd)
	if doTime {
		pTime = time.Now()
	}
	PanicIf(xErr != nil, "tx: %s Error Prepping query (%s): %s\n",
		tx.uuid, cmd, ToJSON(xErr))
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
	result.retrieveAllRowsFromDB()

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
	log.FuncPrintf("tx: %s doCount: %q args: %v", tx.uuid, cmd, args)

	if tx.IsLocked() {
		ShowStack("Attempting a write when TX is locked - tx: %p", tx)
		panic("Tx is locked!!")
	}

	ps, xErr := prepare(tx, cmd)
	PanicIf(xErr != nil, "tx:%s CMD: %q args: %v  err: %s",
		tx.uuid, cmd, args, ToJSON(xErr))
	defer ps.Close()

	result, err := ps.Exec(args...)
	if err != nil {
		Panicf("tx: %s doCount: Error DB(%s)->%s\n", tx.uuid,
			SubQuery(cmd, args), err)
	}

	count, _ := result.RowsAffected()
	log.FuncPrintf("tx: %s doCount: %d rows", tx.uuid, count)
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
	log.FuncPrintf("tx: %s DoCount: %s", tx.uuid, cmd)
	count := doCount(tx, cmd, args...)

	PanicIf(count != num,
		"tx: %s DoOne: Error DB(%s) didn't change exactly %d row(%d)",
		tx.uuid, SubQuery(cmd, args), num, count)
}

//go:embed init.sql
var initDB string
var firstTime = true

// dbHolder owns the live *sql.DB handle for a server. It's stored as an
// opaque value under the "dbHolder" key in a *Config (seeded once, at
// startup, by NewXRServerConfig) rather than storing the *sql.DB itself
// directly in Config.Data. That matters because Config.Data is one shared
// map read (without any locking) from many places throughout request
// handling for unrelated keys (e.g. "path.ui", "rootapp") - if the live DB
// handle lived directly in that map, any Open/Close-triggered write to it
// would race (in the Go data-race sense, whole-map, not per-key) against
// those other concurrent, lock-free reads and could crash the process
// ("fatal error: concurrent map writes"). Routing all DB
// open/close/reconnect activity through dbHolder.mu instead means
// Config.Data itself goes back to being write-once-at-startup, and the
// live handle's own mutations never touch that map again.
type dbHolder struct {
	mu sync.Mutex
	db *sql.DB
}

// OpenDB returns the already-open DB connection for xrsConfig, opening one
// first if needed. It's always safe/cheap to call - if a connection is
// already open it's returned as-is; only the first caller (or the first
// caller after a CloseDB) actually pays for a new sql.Open().
func OpenDB(xrsConfig *Config, name string) (*sql.DB, *XRError) {
	defer log.Trace(name)()

	holder, _ := xrsConfig.Get("dbHolder").(*dbHolder)
	PanicIf(holder == nil, "xrsConfig is missing its dbHolder - was it created via NewXRServerConfig?")

	holder.mu.Lock()
	defer holder.mu.Unlock()

	if holder.db != nil {
		return holder.db, nil
	}

	if firstTime {
		log.FuncPrintf("Open DB: %s:%s:%s",
			xrsConfig.GetAsString("db.host"),
			xrsConfig.GetAsString("db.port"), name)
		firstTime = false
	}

	DB, err := sql.Open("mysql",
		xrsConfig.GetAsString("db.user")+":"+
			xrsConfig.GetAsString("db.password")+"@tcp("+
			xrsConfig.GetAsString("db.host")+":"+
			xrsConfig.GetAsString("db.port")+")/"+name)

	if err != nil {
		return nil, NewXRError("server_error", "/",
			fmt.Sprintf("Error talking to SQL: %s", err))
	}

	DB.SetMaxOpenConns(5)
	DB.SetMaxIdleConns(5)

	holder.db = DB
	if xrsConfig.GetAsString("db.name") != name {
		xrsConfig.Set("db.name", name)
	}

	return DB, nil
}

// CloseDB closes (if open) and forgets xrsConfig's cached DB connection so
// the next OpenDB() call re-opens a fresh one. Safe to call concurrently
// with OpenDB()/CloseDB() - both serialize on the same dbHolder.mu.
func CloseDB(xrsConfig *Config) {
	defer log.Trace()()

	holder, _ := xrsConfig.Get("dbHolder").(*dbHolder)
	PanicIf(holder == nil, "xrsConfig is missing its dbHolder - "+
		"was it created via NewXRServerConfig?")

	holder.mu.Lock()
	defer holder.mu.Unlock()

	if holder.db != nil {
		holder.db.Close()
		holder.db = nil
	}
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
