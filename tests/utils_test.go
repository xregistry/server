package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	gourl "net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestMain(m *testing.M) {
	if tmp := os.Getenv("RX_VERBOSE"); tmp != "" {
		if tmpInt, err := strconv.Atoi(tmp); err == nil {
			log.SetVerbose(tmpInt)
		}
	}

	// call flag.Parse() here if TestMain uses flags
	registry.DeleteDB("testreg")
	registry.CreateDB("testreg")
	registry.OpenDB("testreg")

	// DBName := "registry"
	// if !registry.DBExists(DBName) {
	// registry.CreateDB(DBName)
	// }
	// registry.OpenDB(DBName)

	// Start xRegistry HTTP server
	server := registry.NewServer(8181).Start()

	// Start testing fileserver
	fsServer := &http.Server{
		Addr:    ":8282",
		Handler: http.FileServer(http.Dir("files")),
	}
	go fsServer.ListenAndServe()

	// Run the tests
	rc := m.Run()

	// Shutdown HTTP servers
	server.Close()
	fsServer.Close()

	if rc == 0 {
		// registry.DeleteDB("testreg")
	}

	if dump := registry.DumpTimings(); dump != "" {
		now := time.Now()
		os.WriteFile(fmt.Sprintf("timings-%s.txt", now.Format("15-04-05")),
			[]byte(registry.DumpTimings()), 0666)
	}
	os.Exit(rc)
}

func NewRegistry(name string, opts ...registry.RegOpt) *registry.Registry {
	var err error

	reg, _ := registry.FindRegistry(nil, name, registry.FOR_WRITE)
	if reg != nil {
		reg.Delete()
		reg.SaveAllAndCommit()
	}

	reg, err = registry.NewRegistry(nil, name, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating registry %q: %s\n", name, err)
		ShowStack()
		os.Exit(1)
	}

	reg.SaveAllAndCommit()

	registry.DefaultRegDbSID = reg.DbSID

	/*
		// Now find it again and start a new Tx
		reg, err = registry.FindRegistry(nil, name, registry.FOR_WRITE)
		if err != nil {
			panic(err.Error())
		}
		if reg == nil {
			panic("nil")
		}
	*/

	return reg
}

func PassDeleteReg(t *testing.T, reg *registry.Registry) {
	tx := reg.GetTx()

	if !t.Failed() {
		if tx != nil && tx.IsOpen() {
			if reg.Model.GetChanged() || tx.IsCacheDirty() {
				log.Printf("Tx still open")
				if reg.Model.GetChanged() {
					log.Printf("AND model is changed")
				}
				if tx.IsCacheDirty() {
					log.Printf("AND cache is dirty")
					tx.DumpCache()
				}
				registry.DumpTXs()
				ShowStack()
				os.Exit(1)
			}
		}

		if reg.Model.GetChanged() {
			// This is a show stopped
			panic("Unsaved model outside of a tx")
		}

		if tx := reg.GetTx(); tx.IsCacheDirty() {
			tx.DumpCache()
			panic("Cache is dirty outside of a tx")
		}

		err := reg.SaveAllAndCommit() // should this be Rollback() ?
		if err != nil {
			panic(err.Error())
		}

		if os.Getenv("NO_DELETE_REGISTRY") == "" {
			// We do this to make sure that we can support more than
			// one registry in the DB at a time
			if err := reg.Delete(); err != nil {
				registry.DumpTXs()
				panic(err.Error())
			}
		}
		registry.DefaultRegDbSID = ""
	}

	/*
		err := reg.SaveAllAndCommit() // should this be Rollback() ?
		if err != nil {
			panic("SaveAllAndCommit: " + err.Error())
		}
	*/

	// Close the Tx since we're done with all our work
	if tx != nil {
		tx.Commit()
	}
}

func Fail(t *testing.T, str string, args ...any) {
	t.Helper()
	text := strings.TrimSpace(fmt.Sprintf(str, args...))
	t.Fatalf("%s\n\n", text)
}

func xCheckErr(t *testing.T, err error, errStr string) {
	t.Helper()
	if err == nil {
		if errStr == "" {
			return
		}
		t.Fatalf("\nGot:<no err>\nExp: %s", errStr)
	}

	if errStr == "" {
		t.Fatalf("Test failed: %s", err)
	}

	if err.Error() != errStr {
		t.Fatalf("\nGot: %s\nExp: %s", err.Error(), errStr)
	}
}

func xCheck(t *testing.T, b bool, errStr string, args ...any) {
	t.Helper()
	if !b {
		t.Fatalf(errStr, args...)
	}
}

func xNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

func xCheckGet(t *testing.T, reg *registry.Registry, url string, expected string) {
	t.Helper()
	xNoErr(t, reg.SaveModel())
	xNoErr(t, reg.SaveAllAndCommit())

	if len(url) > 0 {
		url = strings.TrimLeft(url, "/")
	}

	res, err := http.Get("http://localhost:8181/" + url)
	xNoErr(t, err)

	body, err := io.ReadAll(res.Body)
	buf := bytes.NewBuffer(body)
	daURL, _ := gourl.Parse(url)

	if daURL.Query().Has("noprops") {
		buf = bytes.NewBuffer(RemoveProps(buf.Bytes()))
		// expected = string(RemoveProps([]byte(expected)))
	}
	if daURL.Query().Has("oneline") {
		buf = bytes.NewBuffer(OneLine(buf.Bytes()))
		expected = string(OneLine([]byte(expected)))
	}

	xCheckEqual(t, "URL: "+url+"\n", buf.String(), expected)
}

func xCheckNotEqual(t *testing.T, extra string, gotAny any, expAny any) {
	t.Helper()

	exp := fmt.Sprintf("%v", expAny)
	got := fmt.Sprintf("%v", gotAny)

	if exp != got {
		return
	}

	t.Fatalf("Should differ, but they're both:\n%s", exp)
}

func xCheckGreater(t *testing.T, extra string, newAny any, oldAny any) {
	t.Helper()

	New := fmt.Sprintf("%v", newAny)
	Old := fmt.Sprintf("%v", oldAny)

	if New > Old {
		return
	}

	t.Fatalf("New not > Old:\nOld:\n%s\n\nNew:\n%s", Old, New)
}

var TSREGEXP = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[-+]\d{2}:\d{2})`
var TSMASK = TSREGEXP + `||YYYY-MM-DDTHH:MM:SSZ`

// Mask timestamps, but if (for the same input) the same TS is used, make sure
// the mask result is the same for just those two
func MaskTimestamps(input string) string {
	seenTS := map[string]string{}

	replaceFunc := func(input string) string {
		if val, ok := seenTS[input]; ok {
			return val
		}
		val := fmt.Sprintf("YYYY-MM-DDTHH:MM:%02dZ", len(seenTS)+1)
		seenTS[input] = val
		return val
	}

	re := savedREs[TSREGEXP]
	return re.ReplaceAllStringFunc(input, replaceFunc)
}

func xCheckEqual(t *testing.T, extra string, gotAny any, expAny any) {
	t.Helper()
	pos := 0

	exp := fmt.Sprintf("%v", expAny)
	got := fmt.Sprintf("%v", gotAny)

	if exp == "*" {
		return
	}

	// expected output starting with "--TS--" means "skip timestamp masking"
	if len(exp) > 6 && exp[0:6] == "--TS--" {
		exp = exp[6:]
	} else {
		got = MaskTimestamps(got)
		exp = MaskTimestamps(exp)
	}

	for pos < len(got) && pos < len(exp) && got[pos] == exp[pos] {
		pos++
	}
	if pos == len(got) && pos == len(exp) {
		return
	}

	if pos == len(got) {
		t.Fatalf(extra+
			"\nExpected:\n"+exp+
			"\nGot:\n"+got+
			"\nGot ended early at(%d)[%02X]:\n%q",
			pos, exp[pos], got[pos:])
	}

	if pos == len(exp) {
		t.Fatalf(extra+
			"\nExpected:\n"+exp+
			"\nGot:\n"+got+
			"\nExp ended early at(%d)[%02X]:\n"+got[pos:],
			pos, got[pos])
	}

	expMax := pos + 90
	if expMax > len(exp) {
		expMax = len(exp)
	}

	t.Fatalf(extra+
		"\nExpected:\n"+exp+
		"\nGot:\n"+got+
		"\nDiff at(%d)[x%0x/x%0x]:"+
		"\nExp subset:\n"+exp[pos:expMax]+
		"\nGot:\n"+got[pos:],
		pos, exp[pos], got[pos])
}

type HTTPTest struct {
	Name       string
	URL        string
	Method     string
	ReqHeaders []string // name:value
	ReqBody    string

	Code        int
	HeaderMasks []string
	ResHeaders  []string // name:value
	BodyMasks   []string // "PROPNAME" or "SEARCH||REPLACE"
	ResBody     string
}

// http code, body
func xGET(t *testing.T, url string) (int, string) {
	t.Helper()
	url = "http://localhost:8181/" + url
	res, err := http.Get(url)
	if err != nil {
		t.Fatalf("HTTP GET error: %s", err)
	}

	body, _ := io.ReadAll(res.Body)
	/*
		if res.StatusCode != 200 {
			t.Logf("URL: %s", url)
			t.Logf("Code: %d\n%s", res.StatusCode, string(body))
		}
	*/

	return res.StatusCode, string(body)
}

func xHTTP(t *testing.T, reg *registry.Registry, verb, url, reqBody string, code int, resBody string) {
	t.Helper()
	xCheckHTTP(t, reg, &HTTPTest{
		URL:        url,
		Method:     verb,
		ReqBody:    reqBody,
		Code:       code,
		ResBody:    resBody,
		ResHeaders: []string{"*"},
	})
}

type HTTPResult struct {
	http.Response
	body string
}

func xDoHTTP(t *testing.T, reg *registry.Registry, method string, path string,
	bodyStr string) *HTTPResult {

	xNoErr(t, reg.SaveModel())
	xNoErr(t, reg.SaveAllAndCommit())

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	body := io.Reader(nil)
	if bodyStr != "" {
		body = bytes.NewReader([]byte(bodyStr))
	}

	path = strings.TrimLeft(path, "/")

	req, err := http.NewRequest(method, "http://localhost:8181/"+path, body)
	xNoErr(t, err)

	doRes, err := client.Do(req)
	xNoErr(t, err)

	result := &HTTPResult{
		Response: *doRes,
	}

	if doRes != nil {
		tmp, _ := io.ReadAll(doRes.Body)
		result.body = string(tmp)
	}
	return result
}

func xCheckHTTP(t *testing.T, reg *registry.Registry, test *HTTPTest) {
	t.Helper()
	xNoErr(t, reg.SaveModel())
	xNoErr(t, reg.SaveAllAndCommit())

	// t.Logf("Test: %s", test.Name)
	// t.Logf(">> %s %s  (%s)", test.Method, test.URL, registry.GetStack()[1])

	if test.Name != "" {
		test.Name += ": "
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	body := io.Reader(nil)
	if test.ReqBody != "" {
		body = bytes.NewReader([]byte(test.ReqBody))
	}

	if len(test.URL) > 0 {
		test.URL = strings.TrimLeft(test.URL, "/")
	}

	req, err := http.NewRequest(test.Method,
		"http://localhost:8181/"+test.URL, body)
	xNoErr(t, err)

	// Add all request headers to the outbound message
	for _, header := range test.ReqHeaders {
		name, value, _ := strings.Cut(header, ":")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		req.Header.Add(name, value)
	}

	resBody := []byte{}
	res, err := client.Do(req)
	if res != nil {
		resBody, _ = io.ReadAll(res.Body)
	}

	xNoErr(t, err)
	if test.Code < 10 {
		xCheck(t, int(res.StatusCode/100) == test.Code,
			fmt.Sprintf("Expected status %dxx, got %d\n%s",
				test.Code, res.StatusCode, string(resBody)))
	} else {
		xCheck(t, res.StatusCode == test.Code,
			fmt.Sprintf("Expected status %d, got %d\n%s",
				test.Code, res.StatusCode, string(resBody)))
	}

	// t.Logf("%v\n%s", res.Header, string(resBody))
	testHeaders := map[string]string{}

	// This stuff is for masking timestamps. Need to make sure that we
	// process the expected and result timestamps in the same order, so
	// use 2 different "seenTS" maps
	testSeenTS := map[string]string{}
	resSeenTS := map[string]string{}
	replaceFunc := func(input string, seenTS map[string]string) string {
		if val, ok := seenTS[input]; ok {
			return val
		}
		val := fmt.Sprintf("YYYY-MM-DDTHH:MM:%02dZ", len(seenTS)+1)
		seenTS[input] = val
		return val
	}
	testReplaceFunc := func(input string) string {
		return replaceFunc(input, testSeenTS)
	}
	resReplaceFunc := func(input string) string {
		return replaceFunc(input, resSeenTS)
	}
	TSre := savedREs[TSREGEXP]

	// Parse expected headers - split and lowercase the name
	for _, v := range test.ResHeaders {
		name, value, _ := strings.Cut(v, ":")
		name = strings.ToLower(name)
		testHeaders[name] = strings.TrimSpace(value)
	}

	// Extract the response headers - lowercase the name.
	// Save the complete list for error reporting (gotHeaders)
	resHeaders := map[string]string{}
	gotHeaders := ""

	for name, vals := range res.Header {
		value := ""
		if len(vals) > 0 {
			value = vals[0]
		}

		name = strings.ToLower(name)
		resHeaders[name] = strings.TrimSpace(value)
		gotHeaders += fmt.Sprintf("\n%s: %s", name, value)
	}

	// Parse the headerMasks, if any so we can quickly use them later on
	headerMasks := []*regexp.Regexp{}
	headerReplace := []string{}
	for _, mask := range test.HeaderMasks {
		var re *regexp.Regexp
		search, replace, _ := strings.Cut(mask, "||")
		if re = savedREs[search]; re == nil {
			re = regexp.MustCompile(search)
			savedREs[search] = re
		}
		headerMasks = append(headerMasks, re)
		headerReplace = append(headerReplace, replace)
	}

	for name, value := range testHeaders {
		if name == "*" {
			continue
			// see comment in next section
		}

		// Make sure headers that start with '-' are NOT in the response
		if name[0] == '-' {
			if _, ok := resHeaders[name[1:]]; ok {
				t.Errorf("%sHeader '%s: %s' should not be "+
					"present\n\nGot headers:%s",
					test.Name, name[1:], value, gotHeaders)
				t.FailNow()
			}
			continue
		}

		resValue, ok := resHeaders[name]
		if !ok {
			t.Errorf("%s\nMissing header: %s: %s\n\nGot headers:%s\n\nBody: %s",
				test.Name, name, value, gotHeaders, string(resBody))
			t.FailNow()
		}

		// Mask timestamps
		if strings.HasSuffix(name, "at") {
			value = TSre.ReplaceAllStringFunc(value, testReplaceFunc)
			resValue = TSre.ReplaceAllStringFunc(resValue, resReplaceFunc)
		}

		first := true // only mask the expected value once
		for i, re := range headerMasks {
			if first {
				value = re.ReplaceAllString(value, headerReplace[i])
				first = false
			}
			resValue = re.ReplaceAllString(resValue, headerReplace[i])
		}

		xCheckEqual(t, "Header:"+name+"\n", resValue, value)
		// Delete the response header so we'll know if there are any
		// unexpected xregistry- headers left around
		delete(resHeaders, name)
	}

	// Make sure we don't have any extra xReg headers
	// testHeaders with just "*":"" means skip all header checks
	// didn't use len(testHeaders) == 0 to ensure we don't skip by accident
	if len(testHeaders) != 1 || testHeaders["*"] != "" {
		for name, _ := range resHeaders {
			if !strings.HasPrefix(name, "xregistry-") {
				continue
			}
			t.Fatalf("%s\nExtra header(%s)\nGot:%s", test.Name, name, gotHeaders)
		}
	}

	// Only check body if not "*"
	if test.ResBody != "*" {
		testBody := test.ResBody

		for _, mask := range test.BodyMasks {
			var re *regexp.Regexp
			search, replace, found := strings.Cut(mask, "||")
			if !found {
				// Must be just a property name
				search = fmt.Sprintf(`("%s": ")(.*)(")`, search)
				replace = `${1}xxx${3}`
			}

			if re = savedREs[search]; re == nil {
				re = regexp.MustCompile(search)
				savedREs[search] = re
			}

			resBody = re.ReplaceAll(resBody, []byte(replace))
			testBody = re.ReplaceAllString(testBody, replace)
		}

		xCheckEqual(t, "Test: "+test.Name+"\nBody:\n",
			string(resBody), testBody)
		if t.Failed() {
			t.FailNow()
		}
	}
}

var savedREs = map[string]*regexp.Regexp{
	TSREGEXP: regexp.MustCompile(TSREGEXP),
}

func xJSONCheck(t *testing.T, gotObj any, expObj any) {
	t.Helper()
	got := ToJSON(gotObj)
	exp := ToJSON(expObj)
	xCheckEqual(t, "", got, exp)
}

func xCLIServer(serverURL string) {
	os.Setenv("XR_SERVER", serverURL)
}

func xCLI(t *testing.T, line string, in, Eout, Eerr string, work bool) {
	t.Helper()

	args := strings.Split(line, " ")
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}

	cmd := exec.Command("../xr", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if in != "" {
		cmd.Stdin = bytes.NewBuffer([]byte(in))
	}

	err := cmd.Run()

	if err != nil && work {
		t.Fatalf("Should have worked: %s\nStdout: %s\nStderr: %s",
			err, stdout.String(), stderr.String())
	} else if err == nil && !work {
		t.Fatalf("Should have failed:\nStdout: %s\nStderr: %s",
			stdout.String(), stderr.String())
	}

	xCheckEqual(t, "Stderr:", stderr.String(), Eerr)
	xCheckEqual(t, "Stdout:", stdout.String(), Eout)
}
