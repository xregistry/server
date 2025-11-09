package tests

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

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

func TestXRServerRecreates(t *testing.T) {
	// Granted we're just checking log messages... maybe one day we'll
	// check the DB itself to make sure the logs aren't lying

	cmd := exec.Command("../xrserver", "--recreatedb", "-vv", "--verify")
	buf, err := cmd.CombinedOutput()
	xNoErr(t, err)
	out := string(buf)

	exp := `
2025/10/14 12:20:01 GitCommit: f680917749
2025/10/14 12:20:01 DB server: localhost:3306
2025/10/14 12:20:01 Deleting DB: registry
2025/10/14 12:20:01 Creating DB: registry
2025/10/14 12:20:02 Creating xReg: xRegistry
2025/10/14 12:20:02 Default(/): reg-xRegistry
2025/10/14 12:20:02 Done verifying, exiting
`

	re := regexp.MustCompile(`(^|\n)\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	out = re.ReplaceAllString(out, "\nDATE ")
	exp = re.ReplaceAllString(exp, "\nDATE ")

	re = regexp.MustCompile(`GitCommit: [0-9a-f]*\n`)
	out = re.ReplaceAllString(out, "GitCommit: <n/a>\n")
	exp = re.ReplaceAllString(exp, "GitCommit: <n/a>\n")

	re = regexp.MustCompile(`DB server: .*:3306`)
	out = re.ReplaceAllString(out, "DB server: xxx:3306")
	exp = re.ReplaceAllString(exp, "DB server: xxx:3306")

	xCheckEqual(t, "", out, exp)

	// --

	cmd = exec.Command("../xrserver", "--recreatereg", "-vv", "--verify")
	buf, err = cmd.CombinedOutput()
	xNoErr(t, err)
	out = string(buf)

	exp = `
2025/10/14 12:20:01 GitCommit: f680917749
2025/10/14 12:20:01 DB server: localhost:3306
2025/10/14 12:20:01 Deleting xReg: xRegistry
2025/10/14 12:20:02 Creating xReg: xRegistry
2025/10/14 12:20:02 Default(/): reg-xRegistry
2025/10/14 12:20:02 Done verifying, exiting
`

	re = regexp.MustCompile(`(^|\n)\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	out = re.ReplaceAllString(out, "\nDATE ")
	exp = re.ReplaceAllString(exp, "\nDATE ")

	re = regexp.MustCompile(`GitCommit: [0-9a-f]*\n`)
	out = re.ReplaceAllString(out, "GitCommit: <n/a>\n")
	exp = re.ReplaceAllString(exp, "GitCommit: <n/a>\n")

	re = regexp.MustCompile(`DB server: .*:3306`)
	out = re.ReplaceAllString(out, "DB server: xxx:3306")
	exp = re.ReplaceAllString(exp, "DB server: xxx:3306")

	xCheckEqual(t, "", out, exp)

	// --

	cmd = exec.Command("../xrserver", "--recreatereg", "--recreatedb", "-vv",
		"--verify")
	buf, err = cmd.CombinedOutput()
	xNoErr(t, err)
	out = string(buf)

	exp = `
2025/10/14 12:20:01 GitCommit: f680917749
2025/10/14 12:20:01 DB server: localhost:3306
2025/10/14 12:20:01 Deleting DB: registry
2025/10/14 12:20:01 Creating DB: registry
2025/10/14 12:20:02 Creating xReg: xRegistry
2025/10/14 12:20:02 Default(/): reg-xRegistry
2025/10/14 12:20:02 Done verifying, exiting
`

	re = regexp.MustCompile(`(^|\n)\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} `)
	out = re.ReplaceAllString(out, "\nDATE ")
	exp = re.ReplaceAllString(exp, "\nDATE ")

	re = regexp.MustCompile(`GitCommit: [0-9a-f]*\n`)
	out = re.ReplaceAllString(out, "GitCommit: <n/a>\n")
	exp = re.ReplaceAllString(exp, "GitCommit: <n/a>\n")

	re = regexp.MustCompile(`DB server: .*:3306`)
	out = re.ReplaceAllString(out, "DB server: xxx:3306")
	exp = re.ReplaceAllString(exp, "DB server: xxx:3306")

	xCheckEqual(t, "", out, exp)
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
				"YYYY/MM/DD HH:MM:SS DB server: host:port\n" +
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
				"2025/10/15 19:46:51 DB server: localhost:3306\n" +
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
testreg          2025-10-16 12:23:14   2025-10-16 12:23:14
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
			Args:  "-v --verify --dontcreate -r " + TestRegName,
			Stdin: "",
			Code:  0,
			Experr: "YYYY/MM/DD HH:MM:SS GitCommit: 687dd7425c\n" +
				"YYYY/MM/DD HH:MM:SS DB server: localhost:3306\n" +
				"YYYY/MM/DD HH:MM:SS Default(/): reg-testreg\n" +
				"YYYY/MM/DD HH:MM:SS Done verifying, exiting\n",
		},
		{
			Args:  "-v run --verify -r " + TestRegName,
			Stdin: "",
			Code:  0,
			Experr: "YYYY/MM/DD HH:MM:SS GitCommit: 687dd7425c\n" +
				"YYYY/MM/DD HH:MM:SS DB server: localhost:3306\n" +
				"YYYY/MM/DD HH:MM:SS Default(/): reg-testreg\n" +
				"YYYY/MM/DD HH:MM:SS Done verifying, exiting\n",
		},
	}

	for _, test := range tests {
		t.Logf("CMD: xrserver %s", test.Args)
		xServer(t, test.Args, test.Stdin, test.Expout, test.Experr, test.Code)
	}

}
