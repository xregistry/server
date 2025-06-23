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
