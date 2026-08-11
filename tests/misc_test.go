package tests

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	log "github.com/duglin/dlog"

	. "github.com/xregistry/server/common"
)

func TestMiscDBRows(t *testing.T) {
	// Make sure we don't create extra extra stuff in the DB.
	reg := NewRegistry("TestMiscDBRows")
	defer PassDeleteReg(t, reg)

	_, _, err := reg.Model.CreateModels("dirs", "dir", "files", "file")
	XNoErr(t, err)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",
  "ancestorid": "v1"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref": "/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "readonly": false,

  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1$details",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fy/meta",
		`{"xref": "/dirs/d1/files/zz"}`, 201, `{
  "fileid": "fy",
  "self": "http://localhost:8181/dirs/d1/files/fy/meta",
  "xid": "/dirs/d1/files/fy/meta",
  "xref": "/dirs/d1/files/zz"
}
`)

	strFn := func(v any) string {
		vp := v.(*any)
		return NotNilString(vp)
	}

	rows := reg.Query("SELECT Path,PropName,PropValue "+
		"FROM Props WHERE RegSID=? AND IsDefaultVerCopy=false AND "+
		"IsXrefPropCopy=false AND IsXrefVerCopy=false AND "+
		"IsCalcStatic=false AND IsCalcDynamic=false "+
		"ORDER BY Path, PropName ",
		reg.DbSID)

	result := ""
	for _, row := range rows {
		result += fmt.Sprintf("%s: %s -> %s\n",
			strFn(row[0]), strFn(row[1]), strFn(row[2]))
	}
	result = MaskTimestamps(result)

	// Some thing to note about this output, for those new to this stuff
	// - each name ends with , (DB_IN) for each parsing/searching
	// - d1's modifiedat timestamp was changed due to fx being created
	// - props that start with "#" are private and for system use/tracking
	// - fx's #createdat is when it was created, if needed when xref is del'd
	// - fx's #epoch is saved so we can calc the new epoch if xref is del'd
	// - #nextversionid is what vID we should use on next system set vID
	// - All entities need at least one Prop, so fx needs 'fileid'
	XEqual(t, "", result, `: createdat, -> YYYY-MM-DDTHH:MM:01Z
: epoch, -> 2
: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
: registryid, -> TestMiscDBRows
dirs/d1: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1: dirid, -> d1
dirs/d1: epoch, -> 3
dirs/d1: modifiedat, -> YYYY-MM-DDTHH:MM:03Z
dirs/d1/files/f1: fileid, -> f1
dirs/d1/files/f1/meta: #nextversionid, -> 1
dirs/d1/files/f1/meta: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/meta: defaultversionid, -> v1
dirs/d1/files/f1/meta: defaultversionsticky, -> false
dirs/d1/files/f1/meta: epoch, -> 1
dirs/d1/files/f1/meta: fileid, -> f1
dirs/d1/files/f1/meta: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/meta: readonly, -> false
dirs/d1/files/f1/versions/v1: ancestorid, -> v1
dirs/d1/files/f1/versions/v1: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/versions/v1: epoch, -> 1
dirs/d1/files/f1/versions/v1: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/versions/v1: versionid, -> v1
dirs/d1/files/fx: fileid, -> fx
dirs/d1/files/fx/meta: #createdat, -> YYYY-MM-DDTHH:MM:04Z
dirs/d1/files/fx/meta: #epoch, -> 1
dirs/d1/files/fx/meta: #nextversionid, -> 2
dirs/d1/files/fx/meta: fileid, -> fx
dirs/d1/files/fx/meta: xref, -> /dirs/d1/files/f1
dirs/d1/files/fy: fileid, -> fy
dirs/d1/files/fy/meta: #createdat, -> YYYY-MM-DDTHH:MM:03Z
dirs/d1/files/fy/meta: #epoch, -> 1
dirs/d1/files/fy/meta: #nextversionid, -> 2
dirs/d1/files/fy/meta: fileid, -> fy
dirs/d1/files/fy/meta: xref, -> /dirs/d1/files/zz
`)

	// Same query but don't exclude the calculated attributes.
	// We want to check EVERYTHING!
	rows = reg.Query("SELECT Path,PropName,PropValue "+
		"FROM Props WHERE RegSID=? ORDER BY Path, PropName ",
		reg.DbSID)

	result = ""
	for _, row := range rows {
		result += fmt.Sprintf("%s: %s -> %s\n",
			strFn(row[0]), strFn(row[1]), strFn(row[2]))
	}
	result = MaskTimestamps(result)

	XEqual(t, "", result,
		`: createdat, -> YYYY-MM-DDTHH:MM:01Z
: epoch, -> 2
: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
: registryid, -> TestMiscDBRows
: xid, -> /
dirs/d1: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1: dirid, -> d1
dirs/d1: epoch, -> 3
dirs/d1: modifiedat, -> YYYY-MM-DDTHH:MM:03Z
dirs/d1: xid, -> /dirs/d1
dirs/d1/files/f1: ancestorid, -> v1
dirs/d1/files/f1: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1: epoch, -> 1
dirs/d1/files/f1: fileid, -> f1
dirs/d1/files/f1: isdefault, -> true
dirs/d1/files/f1: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1: versionid, -> v1
dirs/d1/files/f1: xid, -> /dirs/d1/files/f1
dirs/d1/files/f1/meta: #nextversionid, -> 1
dirs/d1/files/f1/meta: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/meta: defaultversionid, -> v1
dirs/d1/files/f1/meta: defaultversionsticky, -> false
dirs/d1/files/f1/meta: epoch, -> 1
dirs/d1/files/f1/meta: fileid, -> f1
dirs/d1/files/f1/meta: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/meta: readonly, -> false
dirs/d1/files/f1/meta: xid, -> /dirs/d1/files/f1/meta
dirs/d1/files/f1/versions/v1: ancestorid, -> v1
dirs/d1/files/f1/versions/v1: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/versions/v1: epoch, -> 1
dirs/d1/files/f1/versions/v1: fileid, -> f1
dirs/d1/files/f1/versions/v1: isdefault, -> true
dirs/d1/files/f1/versions/v1: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/versions/v1: versionid, -> v1
dirs/d1/files/f1/versions/v1: xid, -> /dirs/d1/files/f1/versions/v1
dirs/d1/files/fx: ancestorid, -> v1
dirs/d1/files/fx: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/fx: epoch, -> 1
dirs/d1/files/fx: fileid, -> fx
dirs/d1/files/fx: isdefault, -> true
dirs/d1/files/fx: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/fx: versionid, -> v1
dirs/d1/files/fx: xid, -> /dirs/d1/files/fx
dirs/d1/files/fx/meta: #createdat, -> YYYY-MM-DDTHH:MM:04Z
dirs/d1/files/fx/meta: #epoch, -> 1
dirs/d1/files/fx/meta: #nextversionid, -> 2
dirs/d1/files/fx/meta: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/fx/meta: defaultversionid, -> v1
dirs/d1/files/fx/meta: defaultversionsticky, -> false
dirs/d1/files/fx/meta: epoch, -> 1
dirs/d1/files/fx/meta: fileid, -> fx
dirs/d1/files/fx/meta: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/fx/meta: readonly, -> false
dirs/d1/files/fx/meta: xid, -> /dirs/d1/files/fx/meta
dirs/d1/files/fx/meta: xref, -> /dirs/d1/files/f1
dirs/d1/files/fx/versions/v1: ancestorid, -> v1
dirs/d1/files/fx/versions/v1: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/fx/versions/v1: epoch, -> 1
dirs/d1/files/fx/versions/v1: fileid, -> fx
dirs/d1/files/fx/versions/v1: isdefault, -> true
dirs/d1/files/fx/versions/v1: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/fx/versions/v1: versionid, -> v1
dirs/d1/files/fx/versions/v1: xid, -> /dirs/d1/files/fx/versions/v1
dirs/d1/files/fy: fileid, -> fy
dirs/d1/files/fy: xid, -> /dirs/d1/files/fy
dirs/d1/files/fy/meta: #createdat, -> YYYY-MM-DDTHH:MM:03Z
dirs/d1/files/fy/meta: #epoch, -> 1
dirs/d1/files/fy/meta: #nextversionid, -> 2
dirs/d1/files/fy/meta: fileid, -> fy
dirs/d1/files/fy/meta: xid, -> /dirs/d1/files/fy/meta
dirs/d1/files/fy/meta: xref, -> /dirs/d1/files/zz
`)

}

func TestMiscCORS(t *testing.T) {
	reg := NewRegistry("TestMiscCORS")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir"
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")
	// XHTTP(t, reg, "PUT", "/dirs/d1", `{}`, 201, `*`)

	type Test struct {
		method string
		url    string
		body   string
		code   int
	}

	for _, test := range []Test{
		{"GET", "/", "", 200},
		{"GET", "/?ui", "", 200},
		{"GET", "/ui", "", 301},
		{"GET", "/proxy?host=http://xregistry.io/xreg", "", 200},
		{"GET", "/reg-TestMiscCORS", "", 200},
		{"DELETE", "/", "", 405},
		{"PUT", "/dirs/d1", "{}", 201},
		{"PUT", "/dirs/d1", "", 400},
		{"DELETE", "/dirs/d1", "", 204},
		{"DELETE", "/", "", 405},
		{"POST", "/dirs", "{}", 200},
		{"POST", "/dirs", "", 400},
		{"PATCH", "/dirs/d1", "{}", 201},
		{"PATCH", "/dirs/d1", "", 400},
	} {
		t.Logf("Test: %s %s", test.method, test.url)
		res := XHTTP(t, reg, test.method, test.url, test.body, test.code, "*")
		t.Logf("response body: %s", res.body)

		XEqual(t, "cors header",
			res.Header.Get("Access-Control-Allow-Origin"), "*")

		// Different endpoints have different allowed methods
		expectedMethods := "DELETE, GET, OPTIONS, PATCH, POST, PUT"
		testLinkHeader := true

		if test.url == "/" || test.url == "/?ui" ||
			test.url == "/reg-TestMiscCORS" {

			// Root doesn't support DELETE
			expectedMethods = "GET, OPTIONS, PATCH, POST, PUT"
		} else if test.url == "/ui" {
			expectedMethods = ""
			testLinkHeader = false
		} else if test.url == "/proxy?host=http://xregistry.io/xreg" {
			// Proxy has its own methods, skip check
			expectedMethods = res.Header.Get("Access-Control-Allow-Methods")
		} else if test.url == "/dirs" {
			// Collection
			expectedMethods = "DELETE, GET, OPTIONS, PATCH, POST"
		}

		XEqual(t, "cors header",
			res.Header.Get("Access-Control-Allow-Methods"),
			expectedMethods)

		if testLinkHeader {
			linkHeader := res.Header.Get("Link")
			XCheck(t, linkHeader != "",
				"Link header should be present for %s %s",
				test.method, test.url)

			expectedURL := "http://localhost:8181"
			if test.url == "/reg-TestMiscCORS" {
				expectedURL = "http://localhost:8181/reg-TestMiscCORS"
			}
			XEqual(t, "link header",
				linkHeader, fmt.Sprintf("<%s>;rel=xregistry-root", expectedURL))
		}
	}
}

type Job struct {
	t         *testing.T
	name      string
	startFlag *bool
	wg        *sync.WaitGroup
	parallel  int
	total     int
	fn        func(num int)

	active int32
}

func NewJob(test *testing.T, name string, sf *bool, wg *sync.WaitGroup, p int, t int, fn func(num int)) *Job {
	j := &Job{
		t:         test,
		name:      name,
		startFlag: sf,
		wg:        wg,
		parallel:  p,
		total:     t,
		fn:        fn,
	}

	ready := int32(0)
	wg.Add(1)
	go func() {
		j.t.Logf("Defined: %s", j.name)
		defer func() {
			j.wg.Done()
			j.t.Logf("Done: %s (job)", j.name)
		}()

		for i := 0; i < j.total; {
			if atomic.LoadInt32(&j.active) < int32(j.parallel) {
				atomic.AddInt32(&j.active, 1)
				go func(c int) {
					defer func(d int) {
						atomic.AddInt32(&j.active, -1)
						j.t.Logf("Done: %s (%d)", j.name, d)
					}(c)
					first := true
					for *j.startFlag == false {
						if first {
							j.t.Logf("Waiting: %s (%d)", j.name, c)
							first = false
							atomic.AddInt32(&ready, 1)
						}
						time.Sleep(2 * time.Millisecond)
					}
					j.t.Logf("Sending: %s (%d)", j.name, c)
					j.fn(c)
				}(i)
				i++
			} else {
				time.Sleep(10 * time.Millisecond)
			}
		}
		for atomic.LoadInt32(&j.active) > 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}()
	// Wait until we have 'parallel' threads ready to go
	for atomic.LoadInt32(&ready) < int32(j.parallel) {
		time.Sleep(2 * time.Millisecond)
	}
	return j
}

func TestMiscConcurrency(t *testing.T) {
	reg := NewRegistry("TestMiscConcurrency")
	defer PassDeleteReg(t, reg)

	models := []string{
		`{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "versionmode": "manual",
          "hasdocument": false
        }
      }
    }
  }
}`,
		`{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "versionmode": "createdat"
        }
      }
    }
  }
}`,
	}

	redoXHTTP := func(verb, path, in string, rc int, out string) {
		for {
			res := XDoHTTP(t, reg, verb, path, in)
			// if res != nil && (res.StatusCode == 500 || res.StatusCode == 503) &&
			if res != nil && res.StatusCode == 503 &&
				strings.Contains(res.body, "server_busy") {
				time.Sleep(100 * time.Millisecond)
				t.Logf("Got 500+try again, retrying...")
				continue
			}

			// t.Logf("Code: %d", res.StatusCode)
			// t.Logf("Body: %s", res.body)

			if rc < 10 {
				XEqual(t, "Unexpected status code", res.StatusCode/100, rc)
			} else {
				XEqual(t, "Unexpected status code", res.StatusCode, rc)
			}
			break
		}
	}

	runs := 0
	for _, mod := range models {
		t.Logf("============================\nMODEL:\n%s\n", mod)
		XHTTP(t, reg, "PUT", "/modelsource", mod, 200, mod+"\n")

		startFlag := false
		wg := &sync.WaitGroup{}

		NewJob(t, "PATCH /", &startFlag, wg, 5, 10, func(num int) {
			redoXHTTP("PATCH", fmt.Sprintf("/"), "{}", 200, "*")
		})

		NewJob(t, "PUT dx", &startFlag, wg, 5, 10, func(num int) {
			redoXHTTP("PUT", fmt.Sprintf("/dirs/d%d", num), "{}", 2, "*")
		})
		NewJob(t, "PUT d1", &startFlag, wg, 5, 10, func(num int) {
			redoXHTTP("PUT", fmt.Sprintf("/dirs/d1"), "{}", 2, "*")
		})

		NewJob(t, "PUT fx", &startFlag, wg, 5, 10, func(num int) {
			redoXHTTP("PUT", fmt.Sprintf("/dirs/d1/files/f%d", num), "{}", 2, "*")
		})
		NewJob(t, "PUT f1", &startFlag, wg, 5, 10, func(num int) {
			redoXHTTP("PUT", fmt.Sprintf("/dirs/d1/files/f1"), "{}", 2, "*")
		})

		NewJob(t, "PUT vx", &startFlag, wg, 5, 10, func(num int) {
			redoXHTTP("PUT", fmt.Sprintf("/dirs/d1/files/f1/versions/%d", num), "{}", 2, "*")
		})
		NewJob(t, "PUT v1", &startFlag, wg, 5, 10, func(num int) {
			redoXHTTP("PUT", fmt.Sprintf("/dirs/d1/files/f1/versions/1"), "{}", 2, "*")
		})

		// oldVerbose := log.GetVerbose()
		// log.SetVerbose(2) // To see server's activity
		// defer log.SetVerbose(oldVerbose)

		runs = runs + 1

		t.Logf("GO Run #%d!!! -----", runs)
		startFlag = true
		wg.Wait()
		t.Logf("DONE Run #%d", runs)

		res := XHTTP(t, reg, "GET", "/?inline", "", 200, "*")

		type tmp struct {
			Epoch     int
			DirsCount int `json:"DirsCount,omitempty"`
			Dirs      map[string]struct {
				Epoch      int
				FilesCount int `json:"FilesCount,omitempty"`
				Files      map[string]struct {
					Meta struct {
						Epoch int
					}
					Epoch         int
					VersionsCount int `json:"VersionsCount,omitempty"`
					Versions      map[string]struct {
						Epoch int
					}
				}
			} `json:"Dirs,omitempty"`
		}
		data := tmp{}
		Unmarshal([]byte(res.body), &data)

		// t.Logf("Json: %s", ToJSON(data))

		// 1=initial, 1=model, 20=/,/dir/x PUT, 1=if deep PUT creates it
		if data.Epoch < 1+runs*21 || data.Epoch > 1+runs*22 {
			t.Fatalf("data.Epoch(%d) needs to be beween %d and %d",
				data.Epoch, 1+runs*21, 1+runs*22)
		}
		XEqual(t, "", data.DirsCount, 10)

		// can be either depending on the order in which things are created
		if data.Dirs["d1"].Epoch != 20 && data.Dirs["d1"].Epoch != 21 {
			t.Fatalf("data.Dirs[d1].Epoch should be 20 or 21, got: %d",
				data.Dirs["d1"].Epoch)
		}

		XEqual(t, "", data.Dirs["d1"].FilesCount, 10)
		XEqual(t, "", data.Dirs["d1"].Files["f1"].Meta.Epoch, 10)
		XEqual(t, "", data.Dirs["d1"].Files["f1"].VersionsCount, 10)

		// clean-up for next round
		XHTTP(t, reg, "DELETE", "/dirs", "", 204, "")
	}
}

// TestMiscDeadlockRetry validates the ServeHTTP retry loop
// (isRetryableDBErr()/serveOneAttempt() in registry/httpStuff.go) by
// forcing a real MySQL deadlock (error 1213) between two concurrent HTTP
// requests, and confirming all requests eventually succeed with no
// client-visible error - i.e. the server transparently retried at least
// one of them on a fresh Tx.
//
// The deadlock is forced naturally (no test-only server code): a bulk
// "DELETE /dirs" (HTTPDeleteGroups in registry/httpStuff.go) iterates the
// set of existing Group IDs as a Go map, whose iteration order is
// randomized per-goroutine/run, locking (FOR_WRITE) each Group row one at
// a time as it goes, then finally locking the Registry row itself
// (Group.Delete() -> Registry.Touch()) once per Group deleted. Two
// concurrent "DELETE /dirs" requests will each visit the same set of
// Group rows in their own randomized order, so with enough Groups and
// enough concurrent attempts, the two Txs are virtually certain to
// eventually lock a pair of rows in reverse order relative to each
// other - exactly what triggers InnoDB's deadlock detector (1213).
func TestMiscDeadlockRetry(t *testing.T) {
	reg := NewRegistry("TestMiscDeadlockRetry")

	// Capture the server's log output so we can confirm the "Retrying"
	// message (emitted by ServeHTTP right before it loops for another
	// attempt) actually fired at least once.
	buf := &SyncBuffer{}
	saveVerbose := log.GetVerbose()
	saveWriter := log.Writer()
	log.SetOutput(buf)
	log.SetVerbose(1)
	// Registered BEFORE the PassDeleteReg defer below so it runs AFTER
	// it (defers run LIFO) - otherwise log output gets reset to nil
	// while PassDeleteReg's cleanup (which can still log, e.g. if it
	// needs to reopen a closed Tx) is still running. Restore the
	// PREVIOUS writer (not nil) since a nil writer would crash any
	// later test that logs after this one runs.
	defer func() {
		log.SetOutput(saveWriter)
		log.SetVerbose(saveVerbose)
	}()
	defer PassDeleteReg(t, reg)

	_, _, err := reg.Model.CreateModels("dirs", "dir", "files", "file")
	XNoErr(t, err)

	const numDirs = 5 // used to be 30
	const numRounds = 6
	const numRequests = 3 // concurrent "DELETE /dirs" per round

	for round := 0; round < numRounds; round++ {
		for i := 0; i < numDirs; i++ {
			XHTTP(t, reg, "PUT", fmt.Sprintf("/dirs/d%d", i), "{}", 201, "*")
		}

		wg := &sync.WaitGroup{}

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				XHTTP(t, reg, "DELETE", "/dirs", "", 204, "")
			}(i)
		}
		wg.Wait()

		// All dirs should be gone regardless of how many times any
		// individual request was retried.
		res := XHTTP(t, reg, "GET", "/", "", 200, "*")
		type tmp struct {
			DirsCount int `json:"dirscount"`
		}
		data := tmp{}
		Unmarshal([]byte(res.body), &data)
		XEqual(t, "", data.DirsCount, 0)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "Retrying") {
		t.Fatalf("expected at least one retry due to a DB lock conflict, " +
			"but none was logged")
	}
}

// SyncBuffer is a concurrency-safe bytes.Buffer wrapper, needed because
// the dlog package (used by the server, running concurrently across
// goroutines in this test) writes to whatever io.Writer is passed to
// log.SetOutput() without any synchronization of its own.
type SyncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *SyncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *SyncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
