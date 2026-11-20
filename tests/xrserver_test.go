package tests

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestXRServerBasic(t *testing.T) {
	defer SetClearEnvVar("HOME", "nodir")() // make sure we don't find one

	cmd := exec.Command("../xrserver", "-?")
	out, err := cmd.CombinedOutput()
	XNoErr(t, err)
	lines, _, _ := strings.Cut(string(out), "Available Commands:")

	// Just look for the first 3 lines
	XEqual(t, "", lines,
		`xRegistry server

Usage:
  xrserver [flags]
  xrserver [command]

`)

	cmd = exec.Command("../xrserver", "--verify")
	out, err = cmd.CombinedOutput()
	XNoErr(t, err)
	lines, _, _ = strings.Cut(string(out), "Available Commands:")

	// Just look for the first 3 lines
	XEqual(t, "", lines, "")

	xrsConfig := registry.NewXRServerConfig("")
	registry.SetXRServerConfigFromEnvVars(xrsConfig)
	defaultRegSegment := xrsConfig.GetAsString("path.defaultreg")
	regCollectionSegment := xrsConfig.GetAsString("path.regcollection")

	connStr := ""
	beName := registry.BackendName(xrsConfig)
	creating := ""

	switch beName {
	case "memory":
		creating = `
2025/05/21 19:01:39 Creating: ` + regCollectionSegment + `/xRegistry`

	case "mysql":
		connStr = fmt.Sprintf("(%s@%s:%s)",
			registry.BackendOpt(xrsConfig, "dbname"),
			registry.BackendOpt(xrsConfig, "host"),
			registry.BackendOpt(xrsConfig, "port"))

	default:
		t.Fatalf("Unknown backend: %s", beName)
	}

	cmd = exec.Command("../xrserver", "--rootapp=xreg", "-v", "--verify")
	out, err = cmd.CombinedOutput()
	t.Logf("out: %s", string(out))
	XNoErr(t, err)
	lines, _, _ = strings.Cut(string(out), "Available Commands:")
	exp := `2025/05/21 19:01:39 GitCommit: 8061f34abf
2025/05/21 19:01:39 Backend: ` + beName + connStr + creating + `
2025/05/21 19:01:39 Path: /ui -> UI
2025/05/21 19:01:39 Path: /` + defaultRegSegment + ` -> ` + regCollectionSegment + `/xRegistry
2025/05/21 19:01:39 Path: / -> ` + regCollectionSegment + `/xRegistry
2025/05/21 19:01:39 Done verifying, exiting
`
	re := regexp.MustCompile(`(^|\n)\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	lines = re.ReplaceAllString(lines, "\nDATE ")
	exp = re.ReplaceAllString(exp, "\nDATE ")

	re = regexp.MustCompile(`GitCommit: [0-9a-f]*\n`)
	lines = re.ReplaceAllString(lines, "GitCommit: <n/a>\n")
	exp = re.ReplaceAllString(exp, "GitCommit: <n/a>\n")

	re = regexp.MustCompile(`Backend: (.*)@(.+)(:[0-9]+)`)
	lines = re.ReplaceAllString(lines, "Backend: $1@xxx$3")
	exp = re.ReplaceAllString(exp, "Backend: $1@xxx$3")

	// Just look for the first 3 lines
	XEqual(t, "", lines, exp)

}

func TestXRServerRecreates(t *testing.T) {
	defer SetClearEnvVar("HOME", "nodir")() // make sure we don't find one

	// Granted we're just checking log messages... maybe one day we'll
	// check the DB itself to make sure the logs aren't lying

	cmd := exec.Command("../xrserver", "--recreatedb", "--rootapp=xreg", "-vv",
		"--verify")
	buf, err := cmd.CombinedOutput()
	XNoErr(t, err)
	out := string(buf)

	xrsConfig := registry.NewXRServerConfig("")
	registry.SetXRServerConfigFromEnvVars(xrsConfig)
	defaultRegSegment := xrsConfig.GetAsString("path.defaultreg")
	regCollectionSegment := xrsConfig.GetAsString("path.regcollection")

	connStr := ""
	beName := registry.BackendName(xrsConfig)
	creating := ""

	switch beName {
	case "memory":

	case "mysql":
		connStr = fmt.Sprintf("(%s@%s:%s)",
			registry.BackendOpt(xrsConfig, "dbname"),
			registry.BackendOpt(xrsConfig, "host"),
			registry.BackendOpt(xrsConfig, "port"))
		creating = `
2025/10/14 12:20:01 Deleting DB: registry`

	default:
		t.Fatalf("Unknown backend: %s", beName)
	}

	creatingDB := ""
	if beName == "mysql" {
		creatingDB = "2025/10/14 12:20:01 Creating DB: registry\n"
	}
	exp := `
2025/10/14 12:20:01 GitCommit: f680917749
2025/10/14 12:20:01 Backend: ` + beName + connStr + creating + `
` + creatingDB + `2025/10/14 12:20:02 Creating: ` + regCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Path: /ui -> UI
2025/10/14 12:20:02 Path: /` + defaultRegSegment + ` -> ` + regCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Path: / -> ` + regCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Done verifying, exiting
`

	re := regexp.MustCompile(`(^|\n)\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	out = re.ReplaceAllString(out, "\nDATE ")
	exp = re.ReplaceAllString(exp, "\nDATE ")

	re = regexp.MustCompile(`GitCommit: [0-9a-f]*\n`)
	out = re.ReplaceAllString(out, "GitCommit: <n/a>\n")
	exp = re.ReplaceAllString(exp, "GitCommit: <n/a>\n")

	re = regexp.MustCompile(`Backend: (.*)@(.+)(:[0-9]+)`)
	out = re.ReplaceAllString(out, "Backend: $1@xxx$3")
	exp = re.ReplaceAllString(exp, "Backend: $1@xxx$3")

	XEqual(t, "", out, exp)

	// --

	cmd = exec.Command("../xrserver", "--recreatereg", "--rootapp=xreg",
		"-vv", "--verify")
	buf, err = cmd.CombinedOutput()
	XNoErr(t, err)
	out = string(buf)

	switch beName {
	case "memory":
		connStr = ""
		creating = ""

	case "mysql":
		connStr = fmt.Sprintf("(%s@%s:%s)",
			registry.BackendOpt(xrsConfig, "dbname"),
			registry.BackendOpt(xrsConfig, "host"),
			registry.BackendOpt(xrsConfig, "port"))
		creating = `
2025/10/14 12:20:01 Deleting xReg: xRegistry`

	default:
		t.Fatalf("Unknown backend: %s", beName)
	}

	exp = `
2025/10/14 12:20:01 GitCommit: f680917749
2025/10/14 12:20:01 Backend: ` + beName + connStr + creating + `
2025/10/14 12:20:02 Creating: ` + regCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Path: /ui -> UI
2025/10/14 12:20:02 Path: /` + defaultRegSegment + ` -> ` + regCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Path: / -> ` + regCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Done verifying, exiting
`

	re = regexp.MustCompile(`(^|\n)\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	out = re.ReplaceAllString(out, "\nDATE ")
	exp = re.ReplaceAllString(exp, "\nDATE ")

	re = regexp.MustCompile(`GitCommit: [0-9a-f]*\n`)
	out = re.ReplaceAllString(out, "GitCommit: <n/a>\n")
	exp = re.ReplaceAllString(exp, "GitCommit: <n/a>\n")

	re = regexp.MustCompile(`Backend: (.*)@(.+)(:[0-9]+)`)
	out = re.ReplaceAllString(out, "Backend: $1@xxx$3")
	exp = re.ReplaceAllString(exp, "Backend: $1@xxx$3")

	XEqual(t, "", out, exp)

	// --

	cmd = exec.Command("../xrserver", "--recreatereg", "--rootapp=xreg",
		"--recreatedb", "-vv",
		"--verify")
	buf, err = cmd.CombinedOutput()
	XNoErr(t, err)
	out = string(buf)

	switch beName {
	case "memory":
		creating = ``

	case "mysql":
		connStr = fmt.Sprintf("(%s@%s:%s)",
			registry.BackendOpt(xrsConfig, "dbname"),
			registry.BackendOpt(xrsConfig, "host"),
			registry.BackendOpt(xrsConfig, "port"))
		creating = `
2025/10/14 12:20:01 Deleting DB: registry`

	default:
		t.Fatalf("Unknown backend: %s", beName)
	}

	exp = `
2025/10/14 12:20:01 GitCommit: f680917749
2025/10/14 12:20:01 Backend: ` + beName + connStr + creating + `
` + creatingDB + `2025/10/14 12:20:02 Creating: ` + regCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Path: /ui -> UI
2025/10/14 12:20:02 Path: /` + defaultRegSegment + ` -> ` + regCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Path: / -> ` + regCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Done verifying, exiting
`

	re = regexp.MustCompile(`(^|\n)\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	out = re.ReplaceAllString(out, "\nDATE ")
	exp = re.ReplaceAllString(exp, "\nDATE ")

	re = regexp.MustCompile(`GitCommit: [0-9a-f]*\n`)
	out = re.ReplaceAllString(out, "GitCommit: <n/a>\n")
	exp = re.ReplaceAllString(exp, "GitCommit: <n/a>\n")

	re = regexp.MustCompile(`Backend: (.*)@(.+)(:[0-9]+)`)
	out = re.ReplaceAllString(out, "Backend: $1@xxx$3")
	exp = re.ReplaceAllString(exp, "Backend: $1@xxx$3")

	XEqual(t, "", out, exp)
}

type xrsExpect struct {
	Code   int
	Expout string
	Experr string
}

func TestXRServerCmds(t *testing.T) {
	defer SetClearEnvVar("HOME", "nodir")() // make sure we don't find one

	xrsConfig := registry.NewXRServerConfig("")
	registry.SetXRServerConfigFromEnvVars(xrsConfig)
	defaultRegSegment := xrsConfig.GetAsString("path.defaultreg")
	regCollectionSegment := xrsConfig.GetAsString("path.regcollection")
	uiSegment := xrsConfig.GetAsString("path.ui")
	beName := registry.BackendName(xrsConfig)

	// Each xrserver call is a separate process so the memory backend never
	// has any data from a previous call. It also has no notion of DBs.
	noDBs := "YYYY/MM/DD HH:MM:SS Backend \"memory\" doesn't support DBs\n"
	noReg := func(name string) string {
		return "YYYY/MM/DD HH:MM:SS Registry \"" + name +
			"\" does not exist\n"
	}

	tests := []struct {
		Args     string
		Stdin    string
		Expected map[string]xrsExpect // key is the backend name
	}{
		{
			Args:  "-v db delete " + TestDBName,
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql":  {Code: -1, Experr: "*"},
				"memory": {Code: 1, Experr: noDBs},
			},
		},
		{
			Args:  "--dontcreate",
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql": {Code: 1,
					Experr: "2025/10/15 00:55:54 DB \"registry\" does not exist\n"},
				"memory": {Code: 1, Experr: noReg("xRegistry")},
			},
		},
		{
			Args:  "-v --dontcreate",
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql": {Code: 1,
					Experr: "YYYY/MM/DD HH:MM:SS GitCommit: sha\n" +
						"YYYY/MM/DD HH:MM:SS Backend: mysql(registry@host:port)\n" +
						"YYYY/MM/DD HH:MM:SS DB \"registry\" does not exist\n"},
				"memory": {Code: 1,
					Experr: "YYYY/MM/DD HH:MM:SS GitCommit: sha\n" +
						"YYYY/MM/DD HH:MM:SS Backend: memory\n" +
						noReg("xRegistry")},
			},
		},
		{
			Args:  "-v db create " + TestDBName,
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql": {Code: 0,
					Experr: "YYYY/MM/DD HH:MM:SS Creating DB: registry\n"},
				"memory": {Code: 1, Experr: noDBs},
			},
		},
		{
			Args:  "-v db get " + TestDBName,
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql":  {Code: 0, Expout: "DB \"registry\" exists\n"},
				"memory": {Code: 1, Experr: noDBs},
			},
		},
		{
			Args:  "-v db list",
			Stdin: "",
			Expected: map[string]xrsExpect{
				// Should look for 'testreg' and title
				"mysql":  {Code: 0, Expout: "*"},
				"memory": {Code: 1, Experr: noDBs},
			},
		},
		{
			Args:  "-v --dontcreate",
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql": {Code: 1,
					Experr: "2025/10/15 19:46:51 GitCommit: 687dd7425c\n" +
						"2025/10/15 19:46:51 Backend: mysql(registry@localhost:3306)\n" +
						"2025/10/15 19:46:51 Registry \"xRegistry\" does not exist\n"},
				"memory": {Code: 1,
					Experr: "2025/10/15 19:46:51 GitCommit: 687dd7425c\n" +
						"2025/10/15 19:46:51 Backend: memory\n" +
						"2025/10/15 19:46:51 Registry \"xRegistry\" does not exist\n"},
			},
		},
		{
			Args:  "-v registry list",
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql":  {Code: 0, Expout: "ID   NAME   CREATED   MODIFIED\n"},
				"memory": {Code: 0, Expout: "ID   NAME   CREATED   MODIFIED\n"},
			},
		},
		{
			Args:  "-v registry create " + TestRegName,
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql": {Code: 0,
					Experr: "YYYY/MM/DD HH:MM:SS Creating: testreg\n"},
				"memory": {Code: 0,
					Experr: "YYYY/MM/DD HH:MM:SS Creating: testreg\n"},
			},
		},
		{
			Args:  "-v registry list",
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql": {Code: 0,
					Expout: `ID        NAME   CREATED               MODIFIED
testreg          2025/10/16 12:23:14   2025/10/16 12:23:14
`},
				"memory": {Code: 0, Expout: "ID   NAME   CREATED   MODIFIED\n"},
			},
		},
		{
			Args:  "-v registry delete " + TestRegName,
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql": {Code: 0,
					Experr: "YYYY/MM/DD HH:MM:SS Deleting: testreg\n"},
				"memory": {Code: 1,
					Experr: "YYYY/MM/DD HH:MM:SS Registry \"testreg\" doesn't exist\n"},
			},
		},
		{
			Args:  "-v registry list",
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql":  {Code: 0, Expout: "ID   NAME   CREATED   MODIFIED\n"},
				"memory": {Code: 0, Expout: "ID   NAME   CREATED   MODIFIED\n"},
			},
		},
		{
			Args:  "-v registry create " + TestRegName,
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql": {Code: 0,
					Experr: "YYYY/MM/DD HH:MM:SS Creating: testreg\n"},
				"memory": {Code: 0,
					Experr: "YYYY/MM/DD HH:MM:SS Creating: testreg\n"},
			},
		},
		{
			Args:  "-v registry get " + TestRegName,
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql": {Code: 0,
					Expout: `ID         : testreg
Created    : YYYY-MM-DDTHH:MM:01Z
Modified   : YYYY-MM-DDTHH:MM:01Z
`},
				"memory": {Code: 1, Experr: noReg("testreg")},
			},
		},
		{
			Args:  "-v --rootapp=xreg --verify --dontcreate -r " + TestRegName,
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql": {Code: 0,
					Experr: "YYYY/MM/DD HH:MM:SS GitCommit: 687dd7425c\n" +
						"YYYY/MM/DD HH:MM:SS Backend: mysql(registry@localhost:3306)\n" +
						"YYYY/MM/DD HH:MM:SS Path: /ui -> UI\n" +
						"YYYY/MM/DD HH:MM:SS Path: /" + defaultRegSegment + " -> " + regCollectionSegment + "/testreg\n" +
						"YYYY/MM/DD HH:MM:SS Path: / -> " + regCollectionSegment + "/testreg\n" +
						"YYYY/MM/DD HH:MM:SS Done verifying, exiting\n"},
				"memory": {Code: 1,
					Experr: "YYYY/MM/DD HH:MM:SS GitCommit: 687dd7425c\n" +
						"YYYY/MM/DD HH:MM:SS Backend: memory\n" +
						noReg("testreg")},
			},
		},
		{
			Args:  "-v run --rootapp=xreg --verify -r " + TestRegName,
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql": {Code: 0,
					Experr: "YYYY/MM/DD HH:MM:SS GitCommit: 687dd7425c\n" +
						"YYYY/MM/DD HH:MM:SS Backend: mysql(registry@localhost:3306)\n" +
						"YYYY/MM/DD HH:MM:SS Path: /ui -> UI\n" +
						"YYYY/MM/DD HH:MM:SS Path: /" + defaultRegSegment + " -> " + regCollectionSegment + "/testreg\n" +
						"YYYY/MM/DD HH:MM:SS Path: / -> " + regCollectionSegment + "/testreg\n" +
						"YYYY/MM/DD HH:MM:SS Done verifying, exiting\n"},
				"memory": {Code: 0,
					Experr: "YYYY/MM/DD HH:MM:SS GitCommit: 687dd7425c\n" +
						"YYYY/MM/DD HH:MM:SS Backend: memory\n" +
						"YYYY/MM/DD HH:MM:SS Creating: " + regCollectionSegment + "/testreg\n" +
						"YYYY/MM/DD HH:MM:SS Path: /ui -> UI\n" +
						"YYYY/MM/DD HH:MM:SS Path: /" + defaultRegSegment + " -> " + regCollectionSegment + "/testreg\n" +
						"YYYY/MM/DD HH:MM:SS Path: / -> " + regCollectionSegment + "/testreg\n" +
						"YYYY/MM/DD HH:MM:SS Done verifying, exiting\n"},
			},
		},
		{
			Args:  "-v --rootapp=ui --verify -r " + TestRegName,
			Stdin: "",
			Expected: map[string]xrsExpect{
				"mysql": {Code: 0,
					Experr: "YYYY/MM/DD HH:MM:SS GitCommit: 687dd7425c\n" +
						"YYYY/MM/DD HH:MM:SS Backend: mysql(registry@localhost:3306)\n" +
						"YYYY/MM/DD HH:MM:SS Path: /ui -> UI\n" +
						"YYYY/MM/DD HH:MM:SS Path: /" + defaultRegSegment + " -> " + regCollectionSegment + "/testreg\n" +
						"YYYY/MM/DD HH:MM:SS Path: / -> " + uiSegment + "\n" +
						"YYYY/MM/DD HH:MM:SS Done verifying, exiting\n"},
				"memory": {Code: 0,
					Experr: "YYYY/MM/DD HH:MM:SS GitCommit: 687dd7425c\n" +
						"YYYY/MM/DD HH:MM:SS Backend: memory\n" +
						"YYYY/MM/DD HH:MM:SS Creating: " + regCollectionSegment + "/testreg\n" +
						"YYYY/MM/DD HH:MM:SS Path: /ui -> UI\n" +
						"YYYY/MM/DD HH:MM:SS Path: /" + defaultRegSegment + " -> " + regCollectionSegment + "/testreg\n" +
						"YYYY/MM/DD HH:MM:SS Path: / -> " + uiSegment + "\n" +
						"YYYY/MM/DD HH:MM:SS Done verifying, exiting\n"},
			},
		},
	}

	for _, test := range tests {
		t.Logf("CMD: xrserver %s", test.Args)
		exp, ok := test.Expected[beName]
		if !ok {
			t.Fatalf("No expected results for backend %q: xrserver %s",
				beName, test.Args)
		}
		XServer(t, test.Args, test.Stdin, exp.Expout, exp.Experr, exp.Code)
	}
}

// Tests the non-DB settings, which apply to all backends
func TestXRServerSettings(t *testing.T) {
	defer SetClearEnvVar("HOME", "nodir")() // make sure we don't find one

	tmphome := t.TempDir()

	// Settings for the active backend need to be keyed by its name
	cfg := registry.NewXRServerConfig("")
	_, xErr := registry.SetXRServerConfigFromEnvVars(cfg)
	XNoErr(t, xErr)
	beName := registry.BackendProfile(cfg)

	beOpt := ""
	if beName == "mysql" { // memory has no options
		beOpt = "backend." + beName + ".dbname: XREG\n"
	}

	configStr := `# a config file
rootapp: xreg
ui.dir: ` + tmphome + `/uidir
ui.xrui.json: ` + tmphome + `/myxrui
verbose: 2
# another comment
http.port: 8686
` + beOpt + `path.ui: iuiu
path.defaultreg: def
path.regcollection: more
`

	XNoErr(t, os.Setenv("HOME", tmphome))

	for _, name := range []string{"XR_ADDR", "XR_PORT"} {
		defer SetClearEnvVar(name, "")()
	}

	XNoErr(t, os.WriteFile(tmphome+"/.xrserver", []byte(configStr), 0600))
	os.Mkdir(tmphome+"/uidir", 0755)
	XNoErr(t, os.WriteFile(tmphome+"/uidir/index.html", []byte("hello"), 0600))
	XNoErr(t, os.WriteFile(tmphome+"/myxrui",
		[]byte(`{"title":"testing"}`), 0600))

	backendRE := ""
	switch beName {
	case "memory":
		backendRE = "memory\n"
	case "mysql":
		backendRE = "mysql\\(XREG@.*:[0-9]+\\)"
	default:
		t.Fatalf("Unknown backend: %s", beName)
	}

	runRes := Run("../xrserver")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()

	for i := 0; ; i++ {
		res, xErr := HttpDo(&HttpRequest{
			Verb: "GET",
			URL:  "http://localhost:8686",
		})
		if res.Code == 200 {
			break
		}
		if !IsNil(xErr) &&
			!strings.Contains(xErr.String(), "connection refused") {
			XNoErr(t, xErr)
		}

		time.Sleep(5 * time.Millisecond)
		if i == 200 {
			t.Logf("out: %s", string(runRes.Out))
			t.Logf("err: %s", string(runRes.Err))
			t.Errorf("Timed-out waiting")
			t.FailNow()
		}
	}

	res, xErr := HttpDo(&HttpRequest{
		Verb: "GET",
		URL:  "http://localhost:8686/iuiu",
	})
	XNoErr(t, xErr)
	res2, xErr := HttpDo(&HttpRequest{
		Verb: "GET",
		URL:  "http://localhost:8686/iuiu/xrui.json",
	})
	XNoErr(t, xErr)

	runRes.Kill()
	runRes.Wait()

	t.Logf("out: %s", string(runRes.Out))
	t.Logf("err: %s", string(runRes.Err))

	// Check we got dbname too
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Backend: "+backendRE)

	// Check: rootapp & indirectly path.regcollection
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Path: / -> more/xRegistry")

	// Check: http.port, path.ui and path.defaultreg
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Listening on :8686")
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Path: /iuiu -> UI")
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Path: /def -> more/xRegistry")

	// Check: ui.dir, ui.xrui.json
	XEqual(t, "", string(runRes.Err), "^(?m)^.*UI Dir: "+tmphome+"/uidir")
	XEqual(t, "", string(runRes.Err), "^(?m)^.*UI xrui.json: "+
		tmphome+"/myxrui")

	// Check: UI's index.html
	XEqual(t, "", string(res.Body), "hello")

	// Check: xrui.json
	XEqual(t, "", string(res2.Body), `{"title":"testing"}`)

	// Make sure --cset overrides the config file
	runRes = Run("../xrserver", "--cset", "path.ui:other", "--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait()

	XEqual(t, "", string(runRes.Err), "^(?m)^.*Path: /other -> UI")
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Done verifying, exiting")

	// Make sure flags override the config file
	runRes = Run("../xrserver", "--rootapp", "ui", "--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait()

	XEqual(t, "", string(runRes.Err), "^(?m)^.*Path: / -> iuiu")
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Done verifying, exiting")
}

// Tests the mysql-specific connection settings (host, port, user, ...)
func TestXRServerConfig(t *testing.T) {
	SkipIfNotMySQL(t) // these settings only apply to mysql

	defer SetClearEnvVar("HOME", "nodir")() // make sure we don't find one

	tmphome := t.TempDir()

	configStr := `# a config file
verbose: 2
# another comment
backend: mysql
backend.mysql.dbname: XREG
backend.mysql.host: 0.0.0.0
backend.mysql.port: 6033
backend.mysql.user: fido
backend.mysql.password: noway
`

	// Set HOME to tmp dir but no config file YET
	XNoErr(t, os.Setenv("HOME", tmphome))

	for _, name := range []string{"XR_BACKEND", "XR_BACKEND_SECRET", "XR_ADDR", "XR_PORT"} {
		defer SetClearEnvVar(name, "")()
	}

	XNoErr(t, os.WriteFile(tmphome+"/.xrserver", []byte(configStr), 0600))

	// Uses config file
	runRes := Run("../xrserver", "--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*0.0.0.0:6033.*connection refused")

	// env beats config file
	XNoErr(t, os.Setenv("XR_BACKEND", "host=127.0.0.1,port=2222"))
	runRes = Run("../xrserver", "--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*127.0.0.1:2222.*connection refused")

	// Make sure flags win
	runRes = Run("../xrserver", "--verify",
		"--backend", "host=localhost,port=4321")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Backend: mysql\\(XREG@localhost:4321\\)(\n|.)*connection refused")

	// Make host/port good - test other stuff
	XNoErr(t, os.Setenv("XR_BACKEND", "mysql,host=127.0.0.1,port=3306"))

	// Test user - config
	runRes = Run("../xrserver", "--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Access denied.*fido")

	// Test user - env var
	XNoErr(t, os.Setenv("XR_BACKEND",
		"mysql,host=127.0.0.1,port=3306,user=duke"))
	runRes = Run("../xrserver", "--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Access denied.*duke")

	// Test user - flag
	runRes = Run("../xrserver", "--verify", "--backend", "user=lola")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Access denied.*lola")

	// Test password - config
	runRes = Run("../xrserver", "--verify", "--backend", "user=root")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Access denied.*root")

	// Test password - env var
	XNoErr(t, os.Setenv("XR_BACKEND",
		"host=127.0.0.1,port=3306,user=root,password=password"))
	runRes = Run("../xrserver", "--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait()

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Done verifying, exiting")

	// Test password - flag, and a comma in the value via repeated flag
	XNoErr(t, os.Setenv("XR_BACKEND",
		"host=127.0.0.1,port=3306,password=bogus"))
	runRes = Run("../xrserver", "--backend", "user=root",
		"--backend", "password=password", "--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait()

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Done verifying, exiting")

	// Check we got dbname too
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Backend: mysql\\(XREG@.*:3306\\)")

	// Make sure --cset overrides config file
	XNoErr(t, os.Unsetenv("XR_BACKEND"))
	runRes = Run("../xrserver",
		"--cset", "backend.mysql.user:root",
		"--cset", "backend.mysql.password:password",
		"--cset", "backend.mysql.host:127.0.0.1",
		"--cset", "backend.mysql.port:3306",
		"--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait()

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Done verifying, exiting")

	// Unknown options, and options for another backend, are rejected
	runRes = Run("../xrserver", "--backend", "bogus=1", "--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait()

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Unknown option\\(s\\) for backend \"mysql\": bogus")
}

// Tests "backend list" and the --backend / XR_BACKEND syntax, all backends
func TestXRServerBackendOptions(t *testing.T) {
	defer SetClearEnvVar("HOME", "nodir")() // make sure we don't find one
	defer SetClearEnvVar("XR_BACKEND", "")()
	defer SetClearEnvVar("XR_BACKEND_SECRET", "")()

	XServer(t, "backend list", "",
		"memory\n"+
			"  Options: none\n\n"+
			"mysql (active) (default)\n"+
			"  Options:\n"+
			"  - dbname: Name of the DB to use (default: registry)\n"+
			"  - host: MySQL server host address (default: 127.0.0.1)\n"+
			"  - port: MySQL server port (default: 3306)\n"+
			"  - user: MySQL user (default: root)\n"+
			"  - password: MySQL password\n", "", 0)

	// Backend name + options, and the active one is reflected in the list
	XServer(t, "--backend memory backend list", "",
		"memory (active)\n"+
			"  Options: none\n\n"+
			"mysql (default)\n"+
			"  Options:\n"+
			"  - dbname: Name of the DB to use (default: registry)\n"+
			"  - host: MySQL server host address (default: 127.0.0.1)\n"+
			"  - port: MySQL server port (default: 3306)\n"+
			"  - user: MySQL user (default: root)\n"+
			"  - password: MySQL password\n", "", 0)

	XServer(t, "--backend memory backend list -o json", "",
		`[
  {
    "name": "memory",
    "active": true,
    "default": false,
    "options": []
  },
  {
    "name": "mysql",
    "active": false,
    "default": true,
    "options": [
      {
        "name": "dbname",
        "description": "Name of the DB to use",
        "default": "registry"
      },
      {
        "name": "host",
        "description": "MySQL server host address",
        "default": "127.0.0.1"
      },
      {
        "name": "port",
        "description": "MySQL server port",
        "default": "3306"
      },
      {
        "name": "user",
        "description": "MySQL user",
        "default": "root"
      },
      {
        "name": "password",
        "description": "MySQL password"
      }
    ]
  }
]
`, "", 0)

	// XR_BACKEND_SECRET is processed after XR_BACKEND
	restoreSecret := SetClearEnvVar("XR_BACKEND_SECRET", "dbname=sec")
	restoreBE := SetClearEnvVar("XR_BACKEND", "mysql,dbname=pub")
	runRes := Run("../xrserver", "-v", "--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait()
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Creating DB: sec$")
	XServer(t, "db delete sec", "", "", "*", -1)
	restoreBE()
	restoreSecret()

	// Unknown option
	XServer(t, "--backend memory,host=x db list", "", "",
		"YYYY/MM/DD HH:MM:SS Backend config error: Unknown option(s) for backend \"memory\": "+
			"host (valid: none).\n", 1)

	// Option w/o a name and an unknown backend
	XServer(t, "--backend memory,bogus db list", "", "",
		"YYYY/MM/DD HH:MM:SS Error processing --backend: Invalid backend option \"bogus\", "+
			"expected NAME=VALUE.\n", 1)
	XServer(t, "--backend nope db list", "", "",
		"YYYY/MM/DD HH:MM:SS Backend config error: Unknown backend \"nope\" "+
			"(available: memory, mysql).\n", 1)

	// Same, via the env var
	defer SetClearEnvVar("XR_BACKEND", "memory,host=x")()
	XServer(t, "db list", "", "",
		"YYYY/MM/DD HH:MM:SS Backend config error: Unknown option(s) for backend \"memory\": "+
			"host (valid: none).\n", 1)
}

func TestXRServerBackend(t *testing.T) {
	defer SetClearEnvVar("HOME", "nodir")() // make sure we don't find one
	defer SetClearEnvVar("XR_BACKEND", "")()

	xrsConfig := registry.NewXRServerConfig("")
	registry.SetXRServerConfigFromEnvVars(xrsConfig)
	defaultRegSegment := xrsConfig.GetAsString("path.defaultreg")
	regCollectionSegment := xrsConfig.GetAsString("path.regcollection")
	mysqlConn := fmt.Sprintf("(%s@%s:%s)",
		registry.BackendOpt(xrsConfig, "dbname"),
		registry.BackendOpt(xrsConfig, "host"),
		registry.BackendOpt(xrsConfig, "port"))

	// Make sure we start clean. The default backend is mysql
	XServer(t, "db delete registry", "", "", "*", -1)

	// Defaults
	XServer(t, "-vv --verify", "", "",
		"YYYY/MM/DD HH:MM:SS GitCommit: sha\n"+
			"YYYY/MM/DD HH:MM:SS Backend: mysql"+mysqlConn+"\n"+
			"YYYY/MM/DD HH:MM:SS Creating DB: registry\n"+
			"YYYY/MM/DD HH:MM:SS Creating: "+regCollectionSegment+
			"/xRegistry\n"+
			"YYYY/MM/DD HH:MM:SS Path: /ui -> UI\n"+
			"YYYY/MM/DD HH:MM:SS Path: /"+defaultRegSegment+" -> "+
			regCollectionSegment+"/xRegistry\n"+
			"YYYY/MM/DD HH:MM:SS Path: / -> ui\n"+
			"YYYY/MM/DD HH:MM:SS Done verifying, exiting\n", 0)

	// Each way of selecting a backend, for each backend. mysql already has
	// its DB and registry from above, memory always starts empty.
	for _, be := range []string{"mysql", "memory"} {
		for _, how := range []string{"env", "flag", "config"} {
			t.Logf("Backend: %s via %s", be, how)

			args := "-vv --verify"
			cfgLine := ""
			home := "nodir"

			switch how {
			case "env":
				defer SetClearEnvVar("XR_BACKEND", be)()
			case "flag":
				defer SetClearEnvVar("XR_BACKEND", "")()
				args = "--backend " + be + " " + args
			case "config":
				defer SetClearEnvVar("XR_BACKEND", "")()
				home = t.TempDir()
				XNoErr(t, os.WriteFile(home+"/.xrserver",
					[]byte("backend: "+be+"\n"), 0600))
				cfgLine = "YYYY/MM/DD HH:MM:SS Config: " + home +
					"/.xrserver\n"
			}
			restoreHome := SetClearEnvVar("HOME", home)

			exp := cfgLine + "YYYY/MM/DD HH:MM:SS GitCommit: sha\n"
			if be == "mysql" {
				exp += "YYYY/MM/DD HH:MM:SS Backend: mysql" + mysqlConn + "\n"
			} else {
				exp += "YYYY/MM/DD HH:MM:SS Backend: memory\n" +
					"YYYY/MM/DD HH:MM:SS Creating: " + regCollectionSegment +
					"/xRegistry\n"
			}
			exp += "YYYY/MM/DD HH:MM:SS Path: /ui -> UI\n" +
				"YYYY/MM/DD HH:MM:SS Path: /" + defaultRegSegment + " -> " +
				regCollectionSegment + "/xRegistry\n" +
				"YYYY/MM/DD HH:MM:SS Path: / -> ui\n" +
				"YYYY/MM/DD HH:MM:SS Done verifying, exiting\n"

			XServer(t, args, "", "", exp, 0)
			restoreHome()
		}
	}
}
