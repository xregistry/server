package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

func TestPrettyPrint(t *testing.T) {
	tests := []struct {
		indent   string
		prefix   string
		line     string
		expected string
	}{
		{line: "", expected: "\n"},
		{line: "\n", expected: "\n\n"},
		{line: " \n", expected: "\n\n"},
		{line: "\n\n", expected: "\n\n\n"},
		{line: "1\n2", expected: "1\n2\n"},
		{line: "123456789012345678901234567890123456789012345678901234567890123456789012345678", expected: "123456789012345678901234567890123456789012345678901234567890123456789012345678\n"},
		{line: "1234567890123456789012345678901234567890123456789012345678901234567890123456789", expected: "1234567890123456789012345678901234567890123456789012345678901234567890123456789\n"},
		{line: "12345678901234567890123456789012345678901234567890123456789012345678901234567890", expected: "1234567890123456789012345678901234567890123456789012345678901234567890123456789\n0\n"},
		{line: "1234567890123456789012345678901234567890123456789012345678901234567890123456789 0", expected: "1234567890123456789012345678901234567890123456789012345678901234567890123456789\n0\n"},
		{line: "123456789012345678901234567890123456789012345678901234567890123456789012345678 90", expected: "123456789012345678901234567890123456789012345678901234567890123456789012345678\n90\n"},
		{line: "123456789012345678901234567890123456789012345678901234567890123456789012345678  90", expected: "123456789012345678901234567890123456789012345678901234567890123456789012345678\n90\n"},
		{line: "123456789012345678901234567890123456789012345678901234567890123456789012345678  9 0", expected: "123456789012345678901234567890123456789012345678901234567890123456789012345678\n9 0\n"},
		{line: "1234567890123456789012345678901234567890123456789012345678901234567890123456789 0", expected: "1234567890123456789012345678901234567890123456789012345678901234567890123456789\n0\n"},
		{line: "123456789012345678901234567890123456789012345678901234567890123456789012345 6789 0", expected: "123456789012345678901234567890123456789012345678901234567890123456789012345\n6789 0\n"},
		{line: " 12345678901234567890123456789012345678901234567890123456789012345678901234567890", expected: "\n1234567890123456789012345678901234567890123456789012345678901234567890123456789\n0\n"},
		{line: "\n12345678901234567890123456789012345678901234567890123456789012345678901234567890", expected: "\n1234567890123456789012345678901234567890123456789012345678901234567890123456789\n0\n"},
		{line: "  \n  12345678901234567890123456789012345678901234567890123456789012345678901234567890", expected: "\n\n1234567890123456789012345678901234567890123456789012345678901234567890123456789\n0\n"},
		{line: "  \n  1 2345678901234567890123456789012345678901234567890123456789012345678901234567890123", expected: "\n  1\n2345678901234567890123456789012345678901234567890123456789012345678901234567890\n123\n"},
		{line: "123456789012345678901234567890123456789012345678901234567890123456789012345678\n90123", expected: "123456789012345678901234567890123456789012345678901234567890123456789012345678\n90123\n"},
		{line: "1234567890123456789012345678901234567890123456789012345678901234567890123456789\n0123", expected: "1234567890123456789012345678901234567890123456789012345678901234567890123456789\n0123\n"},
		{line: "12345678901234567890123456789012345678901234567890123456789012345678901234567890\n123", expected: "1234567890123456789012345678901234567890123456789012345678901234567890123456789\n0\n123\n"},

		{
			indent:   "ab",
			prefix:   "cd",
			line:     "  \n  1 2345678901234567890123456789012345678901234567890123456789012345678901234567890123",
			expected: "abcd\nab    1\nab  234567890123456789012345678901234567890123456789012345678901234567890123456\nab  7890123\n",
		},
		{
			indent:   "ab",
			prefix:   "",
			line:     "  \n  1 2345678901234567890123456789012345678901234567890123456789012345678901234567890123",
			expected: "ab\nab  1\nab23456789012345678901234567890123456789012345678901234567890123456789012345678\nab90123\n",
		},
		{
			indent:   "",
			prefix:   "cd",
			line:     "  \n  1 2345678901234567890123456789012345678901234567890123456789012345678901234567890123",
			expected: "cd\n    1\n  23456789012345678901234567890123456789012345678901234567890123456789012345678\n  90123\n",
		},
		{
			indent:   "  ",
			prefix:   "cd",
			line:     "12345678901234567890123456789012345678901234567890123456789012345678901234567890123",
			expected: "  cd123456789012345678901234567890123456789012345678901234567890123456789012345\n    67890123\n",
		},
		{
			indent:   "  ",
			prefix:   "cd",
			line:     "12345678901234567890123456789012345678901234567890123456789012345678901234 567890123",
			expected: "  cd12345678901234567890123456789012345678901234567890123456789012345678901234\n    567890123\n",
		},
		{
			indent:   "  ",
			prefix:   "cd",
			line:     "123456789012345678901234567890123456789012345678901234567890123456789012345 67890123",
			expected: "  cd123456789012345678901234567890123456789012345678901234567890123456789012345\n    67890123\n",
		},
		{
			indent:   "  ",
			prefix:   "cd",
			line:     "1234567890123456789012345678901234567890123456789012345678901234567890123456 7890123",
			expected: "  cd123456789012345678901234567890123456789012345678901234567890123456789012345\n    6 7890123\n",
		},

		{
			indent:   "LD",
			prefix:   "test:",
			line:     "a2345678901234567890123456789012345678901234567890123456789012345678901234567890123",
			expected: "LDtest:a23456789012345678901234567890123456789012345678901234567890123456789012\n       34567890123\n",
		},
		{
			indent:   "TD",
			prefix:   "test:",
			line:     "b2345678901234567890123456789012345678901234567890123456789012345678901234567890123",
			expected: "TDtest:b23456789012345678901234567890123456789012345678901234567890123456789012\nB      34567890123\n",
		},
		{
			indent:   "B ",
			prefix:   "test:",
			line:     "c2345678901234567890123456789012345678901234567890123456789012345678901234567890123",
			expected: "B test:c23456789012345678901234567890123456789012345678901234567890123456789012\nB      34567890123\n",
		},
		{
			indent:   "B B ",
			prefix:   "test:",
			line:     "d2345678901234567890123456789012345678901234567890123456789012345678901234567890123",
			expected: "B B test:d234567890123456789012345678901234567890123456789012345678901234567890\nB B      1234567890123\n",
		},
		{
			indent:   "B LD",
			prefix:   "test:",
			line:     "e2345678901234567890123456789012345678901234567890123456789012345678901234567890123",
			expected: "B LDtest:e234567890123456789012345678901234567890123456789012345678901234567890\nB        1234567890123\n",
		},
		{
			indent:   "B TD",
			prefix:   "test:",
			line:     "f2345678901234567890123456789012345678901234567890123456789012345678901234567890123",
			expected: "B TDtest:f234567890123456789012345678901234567890123456789012345678901234567890\nB B      1234567890123\n",
		},
	}

	for i, test := range tests {
		res := PrettyPrint(Twiddle(test.indent), test.prefix, test.line)
		if res != Twiddle(test.expected) {
			t.Logf("\nTest #%d:\nExp:\n%s\nGot:\n%s", i+1,
				Twiddle(test.expected), res)
			t.FailNow()
		}
	}
}

func TestTDPrintUsesSuppliedWriter(t *testing.T) {
	td := NewTD(nil, "writer")
	td.Pass("entry")

	out := bytes.Buffer{}
	stdout := captureTestStdout(t, func() {
		td.Print(&out, "", 1)
	})

	expected := `PASS: writer
└─ PASS: entry
Pass: 2   Fail: 0   Warn: 0   Skip: 0
`
	XEqual(t, "Writer Output", out.String(), expected)
	XEqual(t, "Stdout", stdout, "")
}

func TestRunConformIsolatesOutputAndConfigState(t *testing.T) {
	staleRun := NewTD(nil, "stale")
	staleRun.Fail("stale failure")
	staleRegistry := xrlib.DefineRegistry("http://stale.example")
	out := bytes.Buffer{}
	config := &TDConfig{
		Out:          &out,
		Registry:     staleRegistry,
		IgnoreWarn:   true,
		NextStatus:   FAIL,
		ConsoleDepth: 2,
		RunFunc:      "TestTDAllPass",
		TestRuns: map[string]*TD{
			TestFn(TestTDInit).Name(): staleRun,
		},
	}
	targets := []string{"http://one.example", "http://one.example"}
	expected := `PASS: http://one.example
└─ PASS: TestTDAllPass
Pass: 18   Fail: 0   Warn: 0   Skip: 0

PASS: http://one.example
└─ PASS: TestTDAllPass
Pass: 18   Fail: 0   Warn: 0   Skip: 0
`

	stdout := captureTestStdout(t, func() {
		if rc := runConform(targets, config); rc != 0 {
			t.Fatalf("Conform run returned %d:\n%s", rc, out.String())
		}
	})
	if stdout != "" {
		t.Fatalf("Conform wrote outside config.Out: %q", stdout)
	}
	if out.String() != expected {
		t.Fatalf("Wrong conform output:\nExpected:\n%s\nGot:\n%s",
			expected, out.String())
	}
	if config.Registry != staleRegistry ||
		config.NextStatus != FAIL ||
		config.ConsoleDepth != 2 ||
		len(config.TestRuns) != 1 ||
		config.TestRuns[TestFn(TestTDInit).Name()] != staleRun {

		t.Fatalf("Base TDConfig was mutated: %#v", config)
	}
}

func TestFailedDependenciesPropagateConsistently(t *testing.T) {
	oldWidth := terminalWidth
	terminalWidth = 0
	defer func() {
		terminalWidth = oldWidth
	}()

	freshRoot := newTestTD("fresh")
	freshRoot.Config.FailFast = true
	freshDependent := freshRoot.Run(dependencyCaller)
	fresh := renderTD(freshRoot)

	cachedRoot := newTestTD("cached")
	// Seed the cached failure before enabling FailFast for the dependent.
	cachedDependency := NewTD(nil, TestFn(dependencyFailure).Name())
	cachedDependency.Fail("dependency failed")
	cachedRoot.Config.FailFast = true
	cachedRoot.Config.TestRuns[TestFn(dependencyFailure).Name()] =
		cachedDependency
	cachedDependent := cachedRoot.Run(dependencyCaller)
	cached := renderTD(cachedRoot)

	XEqual(t, "Fresh Root Status", freshRoot.Status, FAIL)
	XEqual(t, "Fresh Dependent Status", freshDependent.Status, FAIL)
	XEqual(t, "Fresh Dependency Output", fresh, `FAIL: fresh
└─ FAIL: dependencyCaller
   ├─ FAIL: dependencyFailure
   │  └─ FAIL: dependency failed
   └─ Dependency "com/xregistry/server/cmds/xr.dependencyFailure" failed, leaving
Pass: 0   Fail: 4   Warn: 0   Skip: 0
`)

	XEqual(t, "Cached Root Status", cachedRoot.Status, FAIL)
	XEqual(t, "Cached Dependent Status", cachedDependent.Status, FAIL)
	XEqual(t, "Cached Dependency Output", cached, `FAIL: cached
└─ FAIL: dependencyCaller
   ├─ FAIL: com/xregistry/server/cmds/xr.dependencyFailure (cached)
   └─ Dependency "com/xregistry/server/cmds/xr.dependencyFailure" failed, leaving
Pass: 0   Fail: 3   Warn: 0   Skip: 0
`)
}

func TestConformanceStateErrorRendersCompleteResult(t *testing.T) {
	const target = "http://example.com"

	td := newTestTD(target)
	cachePassedTest(td, TestGroups)
	reg := xrlib.DefineRegistry(target)
	reg.SetStuff("gm", "not a GroupModel")
	td.SetRegistry(reg)
	td.Run(TestResources)

	got := renderTD(td)
	XEqual(t, "State Error Output", got, `FAIL: http://example.com
└─ FAIL: TestResources
   ├─ PASS: com/xregistry/server/cmds/xr.TestGroups (cached)
   └─ FAIL: reg.stuff.gm != *GroupModel
Pass: 1   Fail: 3   Warn: 0   Skip: 0
`)
}

func dependencyFailure(td *TD) {
	td.Fail("dependency failed")
}

func dependencyCaller(td *TD) {
	td.DependsOn(dependencyFailure)
	td.Pass("unreachable")
}

func newTestTD(name string) *TD {
	td := NewTD(nil, name)
	td.Config = &TDConfig{
		IgnoreWarn: true,
		TestRuns:   map[string]*TD{},
	}
	return td
}

func renderTD(td *TD) string {
	out := bytes.Buffer{}
	td.Print(&out, "", 99)
	return out.String()
}

func cachePassedTest(td *TD, fn TestFn) {
	cached := NewTD(nil, fn.Name())
	cached.Config = td.Config
	td.Config.TestRuns[fn.Name()] = cached
}

func captureTestStdout(t *testing.T, fn func()) string {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}

	stdout := os.Stdout
	os.Stdout = file
	defer func() {
		os.Stdout = stdout
	}()

	fn()
	os.Stdout = stdout

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	buf, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(buf)
}

func Twiddle(in string) string {
	in = strings.ReplaceAll(in, "T", "├")
	in = strings.ReplaceAll(in, "L", "└")
	in = strings.ReplaceAll(in, "D", "─")
	in = strings.ReplaceAll(in, "B", "│")
	return in
}
