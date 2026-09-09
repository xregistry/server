package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/xregistry/server/cmds/xr/xrlib"
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
	TDClear()
	defer TDClear()

	td := NewTD(nil, "writer")
	td.Pass("entry")

	out := bytes.Buffer{}
	stdout := captureTestStdout(t, func() {
		td.Print(&out, "", false, 1)
	})

	expected := `PASS: writer
└─ PASS: entry
Pass: 2   Fail: 0   Warn: 0   Skip: 0
`
	if out.String() != expected {
		t.Fatalf("Wrong writer output:\nExpected:\n%s\nGot:\n%s",
			expected, out.String())
	}
	if stdout != "" {
		t.Fatalf("TD.Print wrote outside the supplied writer: %q", stdout)
	}
}

func TestConformRunsAreByteIdenticalInProcess(t *testing.T) {
	server, _ := newTestConformServer(t)
	options := testConformOptions()

	first, rc := testConformOutput([]string{server.URL}, options)
	if rc != 0 {
		t.Fatalf("First conform run failed (%d):\n%s", rc, first)
	}

	second, rc := testConformOutput([]string{server.URL}, options)
	if rc != 0 {
		t.Fatalf("Second conform run failed (%d):\n%s", rc, second)
	}

	if first != second {
		t.Fatalf("Conform output changed between runs: %s", Diff(first, second))
	}
}

func TestConformSubsetDoesNotPolluteLaterRun(t *testing.T) {
	server, _ := newTestConformServer(t)
	options := testConformOptions()

	first, rc := testConformOutput([]string{server.URL}, options)
	if rc != 0 {
		t.Fatalf("First full conform run failed (%d):\n%s", rc, first)
	}

	subsetOptions := options
	subsetOptions.debug = true
	subsetOptions.depth = 0
	subsetOptions.failFast = true
	subsetOptions.runFunc = "TestTDMixture"
	subsetOptions.wrapAt = 0
	if subset, rc := testConformOutput(
		[]string{server.URL}, subsetOptions); rc == 0 {

		t.Fatalf("Subset conform run unexpectedly passed:\n%s", subset)
	}

	second, rc := testConformOutput([]string{server.URL}, options)
	if rc != 0 {
		t.Fatalf("Second full conform run failed (%d):\n%s", rc, second)
	}

	if first != second {
		t.Fatalf("Subset run polluted later output: %s", Diff(first, second))
	}
	if len(TestsRun) != 0 || nextStatus != 0 {
		t.Fatalf("Conform TD state was not cleared: %#v, next=%d",
			TestsRun, nextStatus)
	}
}

func TestConformRepeatedTargetsUseFreshRegistries(t *testing.T) {
	server, requestCount := newTestConformServer(t)

	out, rc := testConformOutput(
		[]string{server.URL, server.URL}, testConformOptions())
	if rc != 0 {
		t.Fatalf("Repeated-target conform run failed (%d):\n%s", rc, out)
	}

	parts := strings.Split(out, "\n\n")
	if len(parts) != 2 || parts[0]+"\n" != parts[1] {
		t.Fatalf("Repeated targets changed output or separators:\n%s", out)
	}

	if got := requestCount("/model"); got != 4 {
		t.Fatalf("Repeated targets made %d /model requests, expected 4", got)
	}
	if got := requestCount("/capabilities"); got != 4 {
		t.Fatalf("Repeated targets made %d /capabilities requests, expected 4",
			got)
	}
}

func TestConformanceErrorsRenderCompleteResults(t *testing.T) {
	t.Run("capabilities", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/":
					writeTestConformRoot(w, r)
				case "/capabilities":
					io.WriteString(w, "{")
				default:
					http.NotFound(w, r)
				}
			}))
		t.Cleanup(server.Close)

		TDClear()
		defer TDClear()
		TestsRun[TestFn(TestModel).Name()] = NewTD(nil, "TestModel")
		TestsRun[TestFn(TestCapabilities).Name()] =
			NewTD(nil, "TestCapabilities")

		td := NewTD(nil, server.URL)
		td.SetRegistry(xrlib.DefineRegistry(server.URL))
		td.Run(TestRegistryRoot)

		out := bytes.Buffer{}
		td.Print(&out, "", false, 99)
		got := out.String()
		assertCompleteTargetResult(t, got, server.URL,
			"Retrieving capabilities MUST work")
		if !strings.Contains(got,
			`There was an error parsing "/capabilities"`) {

			t.Fatalf("Missing original capabilities diagnostic:\n%s", got)
		}
	})

	t.Run("resource state", func(t *testing.T) {
		TDClear()
		defer TDClear()
		TestsRun[TestFn(TestGroups).Name()] = NewTD(nil, "TestGroups")

		const target = "http://example.com"
		reg := xrlib.DefineRegistry(target)
		reg.SetStuff("gm", "not a GroupModel")

		td := NewTD(nil, target)
		td.SetRegistry(reg)
		td.Run(TestResources)

		out := bytes.Buffer{}
		td.Print(&out, "", false, 99)
		assertCompleteTargetResult(t, out.String(), target,
			"reg.stuff.gm != *GroupModel")
	})
}

func testConformOptions() conformOptions {
	return conformOptions{
		depth:  2,
		wrapAt: 79,
	}
}

func testConformOutput(servers []string, options conformOptions) (string, int) {
	out := bytes.Buffer{}
	rc := runConform(servers, &out, options)
	return out.String(), rc
}

func newTestConformServer(
	t *testing.T,
) (*httptest.Server, func(string) int) {
	t.Helper()

	requests := map[string]int{}
	requestLock := sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			requestLock.Lock()
			requests[r.URL.Path]++
			requestLock.Unlock()

			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/":
				writeTestConformRoot(w, r)
			case "/model":
				io.WriteString(w, "{}")
			case "/capabilities":
				io.WriteString(w, `{
  "available": {
    "capabilities": {"mutable": true},
    "entities": {"mutable": true},
    "model": {"mutable": false}
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": ["1.0-rc4"],
  "versionmodes": []
}`)
			default:
				http.NotFound(w, r)
			}
		}))
	t.Cleanup(server.Close)

	return server, func(path string) int {
		requestLock.Lock()
		defer requestLock.Unlock()
		return requests[path]
	}
}

func writeTestConformRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{
  "specversion": "1.0-rc4",
  "registryid": "test",
  "self": "http://%s/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2026-01-01T00:00:00Z",
  "modifiedat": "2026-01-01T00:00:00Z"
}`, r.Host)
}

func assertCompleteTargetResult(
	t *testing.T,
	got string,
	target string,
	diagnostic string,
) {
	t.Helper()

	if !strings.HasPrefix(got, "FAIL: "+target+"\n") {
		t.Fatalf("Missing failed target header:\n%s", got)
	}
	if !strings.Contains(got, diagnostic) {
		t.Fatalf("Missing diagnostic %q:\n%s", diagnostic, got)
	}
	summary := strings.LastIndex(got, "\nPass: ")
	if summary < 0 || !strings.HasSuffix(got, "\n") {
		t.Fatalf("Missing complete target summary:\n%s", got)
	}
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
