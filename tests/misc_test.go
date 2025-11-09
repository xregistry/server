package tests

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	// log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

func TestDBRows(t *testing.T) {
	// Make sure we don't create extra extra stuff in the DB.
	reg := NewRegistry("TestDBRows")
	defer PassDeleteReg(t, reg)

	_, _, err := reg.Model.CreateModels("dirs", "dir", "files", "file")
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref": "/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1$details",
  "defaultversionsticky": false
}
`)

	strFn := func(v any) string {
		vp := v.(*any)
		return NotNilString(vp)
	}

	rows := reg.Query("SELECT e.Path,p.PropName,p.PropValue "+
		"FROM Props AS p "+
		"JOIN Entities AS e ON (p.EntitySID=e.eSID) WHERE p.RegistrySID=? "+
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
	xCheckEqual(t, "", result,
		`: createdat, -> YYYY-MM-DDTHH:MM:01Z
: epoch, -> 2
: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
: registryid, -> TestDBRows
dirs/d1: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1: dirid, -> d1
dirs/d1: epoch, -> 2
dirs/d1: modifiedat, -> YYYY-MM-DDTHH:MM:03Z
dirs/d1/files/f1: fileid, -> f1
dirs/d1/files/f1/meta: #nextversionid, -> 1
dirs/d1/files/f1/meta: compatibility, -> none
dirs/d1/files/f1/meta: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/meta: defaultversionid, -> v1
dirs/d1/files/f1/meta: defaultversionsticky, -> false
dirs/d1/files/f1/meta: epoch, -> 1
dirs/d1/files/f1/meta: fileid, -> f1
dirs/d1/files/f1/meta: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/meta: readonly, -> false
dirs/d1/files/f1/versions/v1: ancestor, -> v1
dirs/d1/files/f1/versions/v1: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/versions/v1: epoch, -> 1
dirs/d1/files/f1/versions/v1: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/versions/v1: versionid, -> v1
dirs/d1/files/fx: fileid, -> fx
dirs/d1/files/fx/meta: #createdat, -> YYYY-MM-DDTHH:MM:03Z
dirs/d1/files/fx/meta: #epoch, -> 1
dirs/d1/files/fx/meta: #nextversionid, -> 2
dirs/d1/files/fx/meta: fileid, -> fx
dirs/d1/files/fx/meta: xref, -> /dirs/d1/files/f1
`)
}

func TestCORS(t *testing.T) {
	reg := NewRegistry("TestCORS")
	defer PassDeleteReg(t, reg)

	reg.Model.AddGroupModel("dirs", "dir")
	// xHTTP(t, reg, "PUT", "/dirs/d1", `{}`, 201, `*`)

	type Test struct {
		method string
		url    string
		body   string
		code   int
	}

	for _, test := range []Test{
		{"GET", "/", "", 200},
		{"GET", "/?ui", "", 200},
		{"GET", "/proxy?host=xregistry.io/xreg", "", 200},
		{"GET", "/reg-TestCORS", "", 200},
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
		res := xDoHTTP(t, reg, test.method, test.url, test.body)
		t.Logf("response body: %s", res.body)

		xCheckEqual(t, "status code", res.StatusCode, test.code)

		xCheckEqual(t, "cors header",
			res.Header.Get("Access-Control-Allow-Origin"), "*")
		xCheckEqual(t, "cors header",
			res.Header.Get("Access-Control-Allow-Methods"),
			"GET, PATCH, POST, PUT, DELETE")
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
	// Wait until we have 'paralle' threads ready to go
	for atomic.LoadInt32(&ready) < int32(j.parallel) {
		time.Sleep(2 * time.Millisecond)
	}
	return j
}

func TestConcurrency(t *testing.T) {
	reg := NewRegistry("TestConcurrency")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModelSimple("files", "file")
	reg.SaveAllAndCommit()

	startFlag := false
	wg := &sync.WaitGroup{}

	NewJob(t, "PATCH /", &startFlag, wg, 5, 10, func(num int) {
		xHTTP(t, reg, "PATCH", fmt.Sprintf("/"), "{}", 200, "*")
	})

	NewJob(t, "PUT dx", &startFlag, wg, 5, 10, func(num int) {
		xHTTP(t, reg, "PUT", fmt.Sprintf("/dirs/d%d", num), "{}", 2, "*")
	})
	NewJob(t, "PUT d1", &startFlag, wg, 5, 10, func(num int) {
		xHTTP(t, reg, "PUT", fmt.Sprintf("/dirs/d1"), "{}", 2, "*")
	})

	NewJob(t, "PUT fx", &startFlag, wg, 5, 10, func(num int) {
		xHTTP(t, reg, "PUT", fmt.Sprintf("/dirs/d1/files/f%d", num), "{}", 2, "*")
	})
	NewJob(t, "PUT f1", &startFlag, wg, 5, 10, func(num int) {
		xHTTP(t, reg, "PUT", fmt.Sprintf("/dirs/d1/files/f1"), "{}", 2, "*")
	})

	NewJob(t, "PUT vx", &startFlag, wg, 5, 10, func(num int) {
		xHTTP(t, reg, "PUT", fmt.Sprintf("/dirs/d1/files/f1/versions/v%d", num), "{}", 2, "*")
	})
	NewJob(t, "PUT v1", &startFlag, wg, 5, 10, func(num int) {
		xHTTP(t, reg, "PUT", fmt.Sprintf("/dirs/d1/files/f1/versions/v1"), "{}", 2, "*")
	})

	// log.SetVerbose(2) // To see server's activity
	defer func() {
		// log.SetVerbose(0)
	}()

	t.Logf("GO!!! -----")
	startFlag = true
	wg.Wait()
	t.Logf("DONE")
	res := xDoHTTP(t, reg, "GET", "/?inline", "")

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

	t.Logf("Json: %s", ToJSON(data))

	// May need to check for 20 here (see below)
	xCheckEqual(t, "", data.Epoch, 21)
	xCheckEqual(t, "", data.DirsCount, 10)

	// can be either depending on the order in which things are created
	if data.Dirs["d1"].Epoch != 20 && data.Dirs["d1"].Epoch != 21 {
		t.Fatalf("data.Dirs[d1].Epoch should be 20 or 21, got: %d",
			data.Dirs["d1"].Epoch)
	}

	xCheckEqual(t, "", data.Dirs["d1"].FilesCount, 10)

	// version "1" may not exist if a PUT .../vX arrives before PUT .../f1
	xCheckEqual(t, "", data.Dirs["d1"].Files["f1"].Meta.Epoch,
		data.Dirs["d1"].Files["f1"].VersionsCount)
}
