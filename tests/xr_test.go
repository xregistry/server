package tests

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	. "github.com/xregistry/server/common"
)

var RepoBase = "https://raw.githubusercontent.com/xregistry/spec/main"

func TestXRBasic(t *testing.T) {
	reg := NewRegistry("TestXRBasic")
	defer PassDeleteReg(t, reg)

	os.Setenv("XR_SERVER", "localhost:8181")

	cmd := exec.Command("../xr")
	out, err := cmd.CombinedOutput()
	xNoErr(t, err)
	lines, _, _ := strings.Cut(string(out), ":")

	// Just look for the first 3 lines of 'xr' look right
	xCheckEqual(t, "", lines, "xRegistry CLI\n\nUsage")

	// Make sure we can validate the various spec owned model files
	files := []string{
		"sample-model.json",
		"endpoint/model.json",
		"message/model.json",
		"schema/model.json",
	}
	paths := os.Getenv("XR_MODEL_PATH")
	os.Setenv("XR_MODEL_PATH", paths+":files:"+RepoBase)

	for _, file := range files {
		fn := file
		if !strings.HasPrefix(fn, "http") {
			fn, _ = FindModelFile(file)
		}
		if fn == "" {
			t.Errorf("Can't find %q in %q", file, paths)
			t.FailNow()
		}

		cmd = exec.Command("../xr", "model", "verify", fn)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("File: %s\nOut: %s\nErr: %s", file, string(out), err)
		}
		xCheckEqual(t, "", string(out), "")
	}

	// Test for no server specified
	os.Setenv("XR_SERVER", "localhost:8181")
	cmd = exec.Command("../xr", "get")
	out, err = cmd.CombinedOutput()
	xCheckEqual(t, "", string(out), `{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "epoch": 1,
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "registryid": "TestXRBasic",
  "self": "http://localhost:8181/",
  "specversion": "1.0-rc1",
  "xid": "/"
}
`)
	xNoErr(t, err)

	cmd = exec.Command("../xr", "model", "group", "create", "-v", "dirs:dir")
	out, err = cmd.CombinedOutput()
	xCheckEqual(t, "", string(out), "Created Group type: dirs:dir\n")
	xNoErr(t, err)

	cmd = exec.Command("../xr", "model", "resource", "create", "-v",
		"-g", "dirs", "files:file")
	out, err = cmd.CombinedOutput()
	xCheckEqual(t, "", string(out), "Created Resource type: files:file\n")
	xNoErr(t, err)

	cmd = exec.Command("../xr", "model", "get")
	out, err = cmd.CombinedOutput()
	xCheckEqual(t, "", string(out), `xRegistry Model:

ATTRIBUTES: TYPE        REQ RO MUT DEFAULT
dirs        map(object) -   -  y   
dirscount   uinteger    y   y  y   
dirsurl     url         y   y  -   

GROUP: dirs / dir

  ATTRIBUTES: TYPE        REQ RO MUT DEFAULT
  files       map(object) -   -  y   
  filescount  uinteger    y   y  y   
  filesurl    url         y   y  -   

  RESOURCE: files/ file
    Max versions      : 0
    Set version id    : true
    Set version sticky: true
    Has document      : true

    ATTRIBUTES:  TYPE REQ RO MUT DEFAULT
    file         any  -   -  y   
    fileproxyurl url  -   -  y   
    fileurl      url  -   -  y   
`)
	xNoErr(t, err)

	cmd = exec.Command("../xr", "model", "get", "-a")
	out, err = cmd.CombinedOutput()
	xCheckEqual(t, "", string(out), `xRegistry Model:

ATTRIBUTES:   TYPE        REQ RO MUT DEFAULT
capabilities  object      -   -  y   
createdat     timestamp   y   -  y   
description   string      -   -  y   
dirs          map(object) -   -  y   
dirscount     uinteger    y   y  y   
dirsurl       url         y   y  -   
documentation url         -   -  y   
epoch         uinteger    y   y  y   
labels        map(string) -   -  y   
model         object      -   y  y   
modelsource   object      -   -  y   
modifiedat    timestamp   y   -  y   
name          string      -   -  y   
registryid    string      y   y  -   
self          url         y   y  -   
specversion   string      y   y  y   
xid           xid         y   y  -   

GROUP: dirs / dir

  ATTRIBUTES:   TYPE        REQ RO MUT DEFAULT
  createdat     timestamp   y   -  y   
  description   string      -   -  y   
  dirid         string      y   -  -   
  documentation url         -   -  y   
  epoch         uinteger    y   y  y   
  files         map(object) -   -  y   
  filescount    uinteger    y   y  y   
  filesurl      url         y   y  -   
  labels        map(string) -   -  y   
  modifiedat    timestamp   y   -  y   
  name          string      -   -  y   
  self          url         y   y  -   
  xid           xid         y   y  -   

  RESOURCE: files/ file
    Max versions      : 0
    Set version id    : true
    Set version sticky: true
    Has document      : true

    ATTRIBUTES:   TYPE        REQ RO MUT DEFAULT
    ancestor      string      y   -  y   
    contenttype   string      -   -  y   
    createdat     timestamp   y   -  y   
    description   string      -   -  y   
    documentation url         -   -  y   
    epoch         uinteger    y   y  y   
    file          any         -   -  y   
    fileid        string      y   -  -   
    fileproxyurl  url         -   -  y   
    fileurl       url         -   -  y   
    isdefault     boolean     y   y  y   false
    labels        map(string) -   -  y   
    modifiedat    timestamp   y   -  y   
    name          string      -   -  y   
    self          url         y   y  -   
    versionid     string      y   -  -   
    xid           xid         y   y  -   

    RESOURCE ATTRIBUTES: TYPE        REQ RO MUT DEFAULT
    fileid               string      y   -  -   
    meta                 object      -   -  y   
    metaurl              url         y   y  -   
    self                 url         y   y  -   
    versions             map(object) -   -  y   
    versionscount        uinteger    y   y  y   
    versionsurl          url         y   y  -   
    xid                  xid         y   y  -   

    META ATTRIBUTES:       TYPE      REQ RO MUT DEFAULT
    compatibility          string    y   -  y   "none"
    compatibilityauthority string    -   -  y   
    createdat              timestamp y   -  y   
    defaultversionid       string    y   -  y   
    defaultversionsticky   boolean   y   -  y   false
    defaultversionurl      url       y   y  y   
    deprecated             object    -   -  y   
    epoch                  uinteger  y   y  y   
    fileid                 string    y   -  -   
    modifiedat             timestamp y   -  y   
    readonly               boolean   y   y  y   false
    self                   url       y   y  -   
    xid                    xid       y   y  -   
    xref                   url       -   -  y   
`)
	xNoErr(t, err)

	cmd = exec.Command("../xr", "create", "/dirs/d1/files/f1/versions/v1",
		"-vd", "hello world")
	out, err = cmd.CombinedOutput()
	xCheckEqual(t, "", string(out), "Created: /dirs/d1/files/f1/versions/v1\n")

	cmd = exec.Command("../xr", "get", "/dirs/d1/files/f1")
	out, err = cmd.CombinedOutput()
	xCheckEqual(t, "", string(out), "hello world\n")

	cmd = exec.Command("../xr", "get", "/dirs/d1/files/f1$details")
	out, err = cmd.CombinedOutput()
	xCheckEqual(t, "", string(out), `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)
}

func TestXRModel(t *testing.T) {
	reg := NewRegistry("TestXRModel")
	defer PassDeleteReg(t, reg)

	os.Setenv("XR_SERVER", "localhost:8181")

	tests := []struct {
		Args   []string
		Stdin  string
		Output string
	}{
		{
			Args: []string{"model", "update", "-d",
				"@files/dir/model-dirs-inc-docs.json"},
			Stdin:  "",
			Output: "",
		},
	}

	for _, test := range tests {
		t.Logf("Args: %v", test.Args)
		cmd := exec.Command("../xr", test.Args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Output: %s\nErr: %s", string(out), err)
		}
		xCheckEqual(t, "", string(out), test.Output)
	}
}

func TestXRServerBasic(t *testing.T) {
	cmd := exec.Command("../xrserver", "-?")
	out, err := cmd.CombinedOutput()
	xNoErr(t, err)
	lines, _, _ := strings.Cut(string(out), "Available Commands:")

	// Just look for the first 3 lines
	xCheckEqual(t, "", lines,
		`xRegistry server

Usage:
  xrserver [flags]
  xrserver [command]

`)

	cmd = exec.Command("../xrserver", "--verify")
	out, err = cmd.CombinedOutput()
	xNoErr(t, err)
	lines, _, _ = strings.Cut(string(out), "Available Commands:")

	// Just look for the first 3 lines
	xCheckEqual(t, "", lines, "")

	cmd = exec.Command("../xrserver", "-v", "--verify")
	out, err = cmd.CombinedOutput()
	xNoErr(t, err)
	lines, _, _ = strings.Cut(string(out), "Available Commands:")
	exp := `2025/05/21 19:01:39 GitCommit: 8061f34abf
2025/05/21 19:01:39 DB server: localhost:3306
2025/05/21 19:01:39 Default(/): reg-xRegistry
2025/05/21 19:01:39 Done verifying, exiting
`
	re := regexp.MustCompile(`(^|\n)\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	lines = re.ReplaceAllString(lines, "\nDATE ")
	exp = re.ReplaceAllString(exp, "\nDATE ")

	re = regexp.MustCompile(`GitCommit: [0-9a-f]*\n`)
	lines = re.ReplaceAllString(lines, "GitCommit: <n/a>\n")
	exp = re.ReplaceAllString(exp, "GitCommit: <n/a>\n")

	re = regexp.MustCompile(`DB server: .*:3306`)
	lines = re.ReplaceAllString(lines, "DB server: xxx:3306")
	exp = re.ReplaceAllString(exp, "DB server: xxx:3306")

	// Just look for the first 3 lines
	xCheckEqual(t, "", lines, exp)

}
