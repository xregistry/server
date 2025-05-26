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
	cmd := exec.Command("../xr")
	out, err := cmd.CombinedOutput()
	xNoErr(t, err)
	lines, _, _ := strings.Cut(string(out), ":")

	// Just look for the first 3 lines
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
}

/*
func TestXRModel(t *testing.T) {
	tests := []struct {
		Args   []string
		Stdin  string
		Output string
	}{
		{
			Args:   []string{"update", "/model", "-d", "@files/model-dirs.json"},
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
*/

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
