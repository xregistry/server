package tests

import (
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
	XNoErr(t, os.Setenv("HOME", "/nodir")) // make sure we don't find one

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

	cmd = exec.Command("../xrserver", "--rootapp=xreg", "-v", "--verify")
	out, err = cmd.CombinedOutput()
	t.Logf("out: %s", string(out))
	XNoErr(t, err)
	lines, _, _ = strings.Cut(string(out), "Available Commands:")
	exp := `2025/05/21 19:01:39 GitCommit: 8061f34abf
2025/05/21 19:01:39 DB: registry@localhost:3306
2025/05/21 19:01:39 Path: /ui -> UI
2025/05/21 19:01:39 Path: /` + registry.DefaultRegSegment + ` -> ` + registry.RegCollectionSegment + `/xRegistry
2025/05/21 19:01:39 Path: / -> ` + registry.RegCollectionSegment + `/xRegistry
2025/05/21 19:01:39 Done verifying, exiting
`
	re := regexp.MustCompile(`(^|\n)\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	lines = re.ReplaceAllString(lines, "\nDATE ")
	exp = re.ReplaceAllString(exp, "\nDATE ")

	re = regexp.MustCompile(`GitCommit: [0-9a-f]*\n`)
	lines = re.ReplaceAllString(lines, "GitCommit: <n/a>\n")
	exp = re.ReplaceAllString(exp, "GitCommit: <n/a>\n")

	re = regexp.MustCompile(`DB: registry@.*:3306`)
	lines = re.ReplaceAllString(lines, "DB: registry@xxx:3306")
	exp = re.ReplaceAllString(exp, "DB: registry@xxx:3306")

	// Just look for the first 3 lines
	XEqual(t, "", lines, exp)

}

func TestXRServerRecreates(t *testing.T) {
	// Granted we're just checking log messages... maybe one day we'll
	// check the DB itself to make sure the logs aren't lying

	cmd := exec.Command("../xrserver", "--recreatedb", "--rootapp=xreg", "-vv",
		"--verify")
	buf, err := cmd.CombinedOutput()
	XNoErr(t, err)
	out := string(buf)

	exp := `
2025/10/14 12:20:01 GitCommit: f680917749
2025/10/14 12:20:01 DB: registry@localhost:3306
2025/10/14 12:20:01 Deleting DB: registry
2025/10/14 12:20:01 Creating DB: registry
2025/10/14 12:20:02 Creating: ` + registry.RegCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Path: /ui -> UI
2025/10/14 12:20:02 Path: /` + registry.DefaultRegSegment + ` -> ` + registry.RegCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Path: / -> ` + registry.RegCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Done verifying, exiting
`

	re := regexp.MustCompile(`(^|\n)\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	out = re.ReplaceAllString(out, "\nDATE ")
	exp = re.ReplaceAllString(exp, "\nDATE ")

	re = regexp.MustCompile(`GitCommit: [0-9a-f]*\n`)
	out = re.ReplaceAllString(out, "GitCommit: <n/a>\n")
	exp = re.ReplaceAllString(exp, "GitCommit: <n/a>\n")

	re = regexp.MustCompile(`DB: registry@.*:3306`)
	out = re.ReplaceAllString(out, "DB: registry@xxx:3306")
	exp = re.ReplaceAllString(exp, "DB: registry@xxx:3306")

	XEqual(t, "", out, exp)

	// --

	cmd = exec.Command("../xrserver", "--recreatereg", "--rootapp=xreg",
		"-vv", "--verify")
	buf, err = cmd.CombinedOutput()
	XNoErr(t, err)
	out = string(buf)

	exp = `
2025/10/14 12:20:01 GitCommit: f680917749
2025/10/14 12:20:01 DB: registry@localhost:3306
2025/10/14 12:20:01 Deleting xReg: xRegistry
2025/10/14 12:20:02 Creating: ` + registry.RegCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Path: /ui -> UI
2025/10/14 12:20:02 Path: /` + registry.DefaultRegSegment + ` -> ` + registry.RegCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Path: / -> ` + registry.RegCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Done verifying, exiting
`

	re = regexp.MustCompile(`(^|\n)\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	out = re.ReplaceAllString(out, "\nDATE ")
	exp = re.ReplaceAllString(exp, "\nDATE ")

	re = regexp.MustCompile(`GitCommit: [0-9a-f]*\n`)
	out = re.ReplaceAllString(out, "GitCommit: <n/a>\n")
	exp = re.ReplaceAllString(exp, "GitCommit: <n/a>\n")

	re = regexp.MustCompile(`DB: registry@.*:3306`)
	out = re.ReplaceAllString(out, "DB: registry@xxx:3306")
	exp = re.ReplaceAllString(exp, "DB: registry@xxx:3306")

	XEqual(t, "", out, exp)

	// --

	cmd = exec.Command("../xrserver", "--recreatereg", "--rootapp=xreg",
		"--recreatedb", "-vv",
		"--verify")
	buf, err = cmd.CombinedOutput()
	XNoErr(t, err)
	out = string(buf)

	exp = `
2025/10/14 12:20:01 GitCommit: f680917749
2025/10/14 12:20:01 DB: registry@localhost:3306
2025/10/14 12:20:01 Deleting DB: registry
2025/10/14 12:20:01 Creating DB: registry
2025/10/14 12:20:02 Creating: ` + registry.RegCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Path: /ui -> UI
2025/10/14 12:20:02 Path: /` + registry.DefaultRegSegment + ` -> ` + registry.RegCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Path: / -> ` + registry.RegCollectionSegment + `/xRegistry
2025/10/14 12:20:02 Done verifying, exiting
`

	re = regexp.MustCompile(`(^|\n)\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	out = re.ReplaceAllString(out, "\nDATE ")
	exp = re.ReplaceAllString(exp, "\nDATE ")

	re = regexp.MustCompile(`GitCommit: [0-9a-f]*\n`)
	out = re.ReplaceAllString(out, "GitCommit: <n/a>\n")
	exp = re.ReplaceAllString(exp, "GitCommit: <n/a>\n")

	re = regexp.MustCompile(`DB: registry@.*:3306`)
	out = re.ReplaceAllString(out, "DB: registry@xxx:3306")
	exp = re.ReplaceAllString(exp, "DB: registry@xxx:3306")

	XEqual(t, "", out, exp)
}

func TestXRServerCmds(t *testing.T) {
	tests := []struct {
		Args   string
		Stdin  string
		Code   int
		Expout string
		Experr string
	}{
		{
			Args:   "-v db delete " + TestDBName,
			Stdin:  "",
			Code:   -1,
			Experr: "*",
		},
		{
			Args:   "--dontcreate",
			Stdin:  "",
			Code:   1,
			Experr: "2025/10/15 00:55:54 DB \"registry\" does not exist\n",
		},
		{
			Args:  "-v --dontcreate",
			Stdin: "",
			Code:  1,
			Experr: "YYYY/MM/DD HH:MM:SS GitCommit: sha\n" +
				"YYYY/MM/DD HH:MM:SS DB: registry@host:port\n" +
				"YYYY/MM/DD HH:MM:SS DB \"registry\" does not exist\n",
		},
		{
			Args:   "-v db create " + TestDBName,
			Stdin:  "",
			Code:   0,
			Experr: "YYYY/MM/DD HH:MM:SS Creating DB: registry\n",
		},
		{
			Args:   "-v db get " + TestDBName,
			Stdin:  "",
			Code:   0,
			Expout: "DB \"registry\" exists\n",
		},
		{
			Args:   "-v db list",
			Stdin:  "",
			Code:   0,
			Expout: "*", // Should look for 'testreg' and title
		},
		{
			Args:  "-v --dontcreate",
			Stdin: "",
			Code:  1,
			Experr: "2025/10/15 19:46:51 GitCommit: 687dd7425c\n" +
				"2025/10/15 19:46:51 DB: registry@localhost:3306\n" +
				"2025/10/15 19:46:51 Registry \"xRegistry\" does not exist\n",
		},
		{
			Args:   "-v registry list",
			Stdin:  "",
			Code:   0,
			Expout: "ID   NAME   CREATED   MODIFIED\n",
		},
		{
			Args:   "-v registry create " + TestRegName,
			Stdin:  "",
			Code:   0,
			Experr: "YYYY/MM/DD HH:MM:SS Creating: testreg\n",
		},
		{
			Args:  "-v registry list",
			Stdin: "",
			Code:  0,
			Expout: `ID        NAME   CREATED               MODIFIED
testreg          2025/10/16 12:23:14   2025/10/16 12:23:14
`,
		},
		{
			Args:   "-v registry delete " + TestRegName,
			Stdin:  "",
			Code:   0,
			Experr: "YYYY/MM/DD HH:MM:SS Deleting: testreg\n",
		},
		{
			Args:   "-v registry list",
			Stdin:  "",
			Code:   0,
			Expout: "ID   NAME   CREATED   MODIFIED\n",
		},
		{
			Args:   "-v registry create " + TestRegName,
			Stdin:  "",
			Code:   0,
			Experr: "YYYY/MM/DD HH:MM:SS Creating: testreg\n",
		},
		{
			Args:  "-v registry get " + TestRegName,
			Stdin: "",
			Code:  0,
			Expout: `ID         : testreg
Created    : YYYY-MM-DDTHH:MM:01Z
Modified   : YYYY-MM-DDTHH:MM:01Z
`,
		},
		{
			Args:  "-v --rootapp=xreg --verify --dontcreate -r " + TestRegName,
			Stdin: "",
			Code:  0,
			Experr: "YYYY/MM/DD HH:MM:SS GitCommit: 687dd7425c\n" +
				"YYYY/MM/DD HH:MM:SS DB: registry@localhost:3306\n" +
				"YYYY/MM/DD HH:MM:SS Path: /ui -> UI\n" +
				"YYYY/MM/DD HH:MM:SS Path: /" + registry.DefaultRegSegment + " -> " + registry.RegCollectionSegment + "/testreg\n" +
				"YYYY/MM/DD HH:MM:SS Path: / -> " + registry.RegCollectionSegment + "/testreg\n" +
				"YYYY/MM/DD HH:MM:SS Done verifying, exiting\n",
		},
		{
			Args:  "-v run --rootapp=xreg --verify -r " + TestRegName,
			Stdin: "",
			Code:  0,
			Experr: "YYYY/MM/DD HH:MM:SS GitCommit: 687dd7425c\n" +
				"YYYY/MM/DD HH:MM:SS DB: registry@localhost:3306\n" +
				"YYYY/MM/DD HH:MM:SS Path: /ui -> UI\n" +
				"YYYY/MM/DD HH:MM:SS Path: /" + registry.DefaultRegSegment + " -> " + registry.RegCollectionSegment + "/testreg\n" +
				"YYYY/MM/DD HH:MM:SS Path: / -> " + registry.RegCollectionSegment + "/testreg\n" +
				"YYYY/MM/DD HH:MM:SS Done verifying, exiting\n",
		},
		{
			Args:  "-v --rootapp=ui --verify -r " + TestRegName,
			Stdin: "",
			Code:  0,
			Experr: "YYYY/MM/DD HH:MM:SS GitCommit: 687dd7425c\n" +
				"YYYY/MM/DD HH:MM:SS DB: registry@localhost:3306\n" +
				"YYYY/MM/DD HH:MM:SS Path: /ui -> UI\n" +
				"YYYY/MM/DD HH:MM:SS Path: /" + registry.DefaultRegSegment + " -> " + registry.RegCollectionSegment + "/testreg\n" +
				"YYYY/MM/DD HH:MM:SS Path: / -> " + registry.UISegment + "\n" +
				"YYYY/MM/DD HH:MM:SS Done verifying, exiting\n",
		},
	}

	for _, test := range tests {
		t.Logf("CMD: xrserver %s", test.Args)
		XServer(t, test.Args, test.Stdin, test.Expout, test.Experr, test.Code)
	}

}

func TestXRServerConfig(t *testing.T) {
	// reg := NewRegistry("TestXRServerConfig")
	// defer PassDeleteReg(t, reg)

	tmphome, err := os.MkdirTemp("", "xrtest-home")
	XNoErr(t, err)
	defer os.RemoveAll(tmphome)

	configStr := `# a config file
rootapp: xreg
ui.dir: ` + tmphome + `/uidir
ui.xrui.json: ` + tmphome + `/myxrui
verbose: 2
# another comment
http.port: 8686
db.name: XREG
db.host: 0.0.0.0
db.port: 6033
db.user: fido
db.password: noway
path.ui: iuiu
path.defaultreg: def
path.regcollection: more
`

	// Set HOME to tmp dir but no config file YET
	XNoErr(t, os.Setenv("HOME", tmphome))

	XNoErr(t, os.Unsetenv("DBNAME"))
	XNoErr(t, os.Unsetenv("DBHOST"))
	XNoErr(t, os.Unsetenv("DBPORT"))
	XNoErr(t, os.Unsetenv("DBUSER"))
	XNoErr(t, os.Unsetenv("DBPASSWORD"))

	XNoErr(t, os.WriteFile(tmphome+"/.xrserver", []byte(configStr), 0600))
	os.Mkdir(tmphome+"/uidir", 0755)
	XNoErr(t, os.WriteFile(tmphome+"/uidir/index.html", []byte("hello"), 0600))
	XNoErr(t, os.WriteFile(tmphome+"/myxrui",
		[]byte(`{"title":"testing"}`), 0600))

	// Uses config file
	runRes := Run("../xrserver")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*0.0.0.0:6033.*connection refused")

	// env beats config file
	XNoErr(t, os.Setenv("DBHOST", "127.0.0.1"))
	XNoErr(t, os.Setenv("DBPORT", "2222"))
	runRes = Run("../xrserver")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*127.0.0.1:2222.*connection refused")

	// Make sure flags win
	XNoErr(t, os.Setenv("DBHOST", "127.0.0.1"))
	XNoErr(t, os.Setenv("DBPORT", "2222"))
	runRes = Run("../xrserver", "--dbhost", "localhost", "--dbport", "4321")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*DB: XREG@localhost:4321(\n|.)*connection refused")

	// Make DB host/port good - test other DB stuff
	XNoErr(t, os.Setenv("DBHOST", "127.0.0.1"))
	XNoErr(t, os.Setenv("DBPORT", "3306"))

	// Test db user - config
	runRes = Run("../xrserver")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Access denied.*fido")

	// Test db user - env var
	XNoErr(t, os.Setenv("DBUSER", "duke"))
	runRes = Run("../xrserver")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Access denied.*duke")

	// Test db user - flag
	runRes = Run("../xrserver", "--dbuser", "lola")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Access denied.*lola")

	// Test db password - config
	runRes = Run("../xrserver", "--dbuser", "root")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait() // Should generate an error on startup

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Access denied.*root")

	// Test db password - env var
	XNoErr(t, os.Setenv("DBPASSWORD", "password"))
	runRes = Run("../xrserver", "--dbuser", "root", "--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait()

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Done verifying, exiting")

	// Test db password - flag
	XNoErr(t, os.Setenv("DBPASSWORD", "bogus"))
	runRes = Run("../xrserver", "--dbuser", "root", "--dbpassword", "password")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()

	for i := 0; ; i++ {
		res, xErr := CommonHttpDo("GET", "http://localhost:8686", nil, nil)
		if res.Code == 200 {
			break
		}
		if !IsNil(xErr) &&
			!strings.Contains(xErr.String(), "connection refused") {
			XNoErr(t, err)
		}

		time.Sleep(5 * time.Millisecond)
		if i == 200 {
			t.Logf("out: %s", string(runRes.Out))
			t.Logf("err: %s", string(runRes.Err))
			t.Errorf("Timed-out waiting")
			t.FailNow()
		}
	}

	res, xErr := CommonHttpDo("GET", "http://localhost:8686/iuiu", nil, nil)
	res2, xErr := CommonHttpDo("GET", "http://localhost:8686/iuiu/xrui.json", nil, nil)

	runRes.Kill()
	runRes.Wait()

	t.Logf("out: %s", string(runRes.Out))
	t.Logf("err: %s", string(runRes.Err))

	XNoErr(t, xErr)

	// Check we got db.name too
	XEqual(t, "", string(runRes.Err), "^(?m)^.*DB: XREG@.*:3306")

	// Check: rootapp & indirectly path.regcollection
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Path: / -> more/xRegistry")

	// Check: http.port, path.ui and path.defaultreg
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Listening on 8686")
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Path: /iuiu -> UI")
	XEqual(t, "", string(runRes.Err), "^(?m)^.*Path: /def -> more/xRegistry")

	// Check: ui.dir, ui.xrui.json, http.port.path.*
	XEqual(t, "", string(runRes.Err), "^(?m)^.*UI Dir: "+tmphome+"/uidir")
	XEqual(t, "", string(runRes.Err), "^(?m)^.*UI xrui.json: "+
		tmphome+"/myxrui")

	// Check: UI's index.html
	XEqual(t, "", string(res.Body), "hello")

	// Check: xrui.json
	XEqual(t, "", string(res2.Body), `{"title":"testing"}`)

	// Make sure --set overrides config file
	XNoErr(t, os.Unsetenv("DBUSER"))
	XNoErr(t, os.Unsetenv("DBPASSWORD"))
	runRes = Run("../xrserver",
		"--set", "db.user:root",
		"--set", "db.password:password",
		"--verify")
	XNoErr(t, runRes.Error)
	defer runRes.Kill()
	runRes.Wait()

	XEqual(t, "", string(runRes.Err),
		"^(?m)^.*Done verifying, exiting")

	// TODO check to make sure cmd line flags override the non-DB stuff

}
