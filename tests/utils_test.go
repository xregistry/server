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

const TestDBName = "registry"
const TestRegName = "testreg"

func TestMain(m *testing.M) {
	if tmp := os.Getenv("RX_VERBOSE"); tmp != "" {
		if tmpInt, err := strconv.Atoi(tmp); err == nil {
			log.SetVerbose(tmpInt)
		}
	}

	// call flag.Parse() here if TestMain uses flags
	registry.DeleteDB(TestRegName)
	registry.CreateDB(TestRegName)
	registry.OpenDB(TestRegName)

	// DBName := "registry"
	// if !registry.DBExists(DBName) {
	// registry.CreateDB(DBName)
	// }
	// registry.OpenDB(DBName)

	if IsPortInUse(8181) {
		panic("Port 8181 is already in use - kill it")
	}

	// Start xRegistry HTTP server
	server := registry.NewServer(8181).Start()

	// Start testing fileserver
	if IsPortInUse(8282) {
		panic("Port 8282 is already in use - kill it")
	}

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
		// registry.DeleteDB(TestRegName)
	}

	if dump := registry.DumpTimings(); dump != "" {
		now := time.Now()
		os.WriteFile(fmt.Sprintf("timings-%s.txt", now.Format("15-04-05")),
			[]byte(registry.DumpTimings()), 0666)
	}
	os.Exit(rc)
}

// The funcs that use registry.* types can't be in "common/test.go" because
// it causes a circular dependency issue. But I wanted to share the other
// testing utils outside of the "testing" dir, so that's why they're split
// across the 2 dirs this way. Kind of weird I know, but I'll clean it later

func NewRegistry(name string, opts ...registry.RegOpt) *registry.Registry {
	reg, _ := registry.FindRegistry(nil, name, registry.FOR_WRITE)
	if reg != nil {
		reg.Delete()
		reg.SaveAllAndCommit()
	}

	reg, xErr := registry.NewRegistry(nil, name, opts...)
	if xErr != nil {
		fmt.Fprintf(os.Stderr, "Error creating registry %q: %s\n", name, xErr)
		ShowStack()
		os.Exit(1)
	}

	reg.SaveAllAndCommit()

	registry.DefaultRegDbSID = reg.DbSID

	/*
		// Now find it again and start a new Tx
		reg, xErr = registry.FindRegistry(nil, name, registry.FOR_WRITE)
		if xErr != nil {
			panic(xErr.String())
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

		xErr := reg.SaveAllAndCommit() // should this be Rollback() ?
		if xErr != nil {
			panic(xErr.String())
		}

		if os.Getenv("NO_DELETE_REGISTRY") == "" {
			// We do this to make sure that we can support more than
			// one registry in the DB at a time
			if xErr := reg.Delete(); xErr != nil {
				registry.DumpTXs()
				panic(xErr.String())
			}
		}
		registry.DefaultRegDbSID = ""
	}

	/*
		xErr := reg.SaveAllAndCommit() // should this be Rollback() ?
		if xErr != nil {
			panic("SaveAllAndCommit: " + xErr.String())
		}
	*/

	// Close the Tx since we're done with all our work
	if tx != nil {
		tx.Commit()
	}
}

func XCheckGet(t *testing.T, reg *registry.Registry, url string, expected string) {
	t.Helper()
	XNoErr(t, reg.SaveModel())
	XNoErr(t, reg.SaveAllAndCommit())

	if len(url) > 0 {
		url = strings.TrimLeft(url, "/")
	}

	res, err := http.Get("http://localhost:8181/" + url)
	XNoErr(t, err)

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

	XEqual(t, "URL: "+url+"\n", buf.String(), expected)
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

func XHTTP(t *testing.T, reg *registry.Registry, verb, url, reqBody string, code int, resBody string, flags ...string) {
	t.Helper()
	XCheckHTTP(t, reg, &HTTPTest{
		URL:        url,
		Method:     verb,
		ReqBody:    reqBody,
		Code:       code,
		ResBody:    resBody,
		ResHeaders: []string{"*"},
	}, flags...)
}

type HTTPResult struct {
	http.Response
	body string
}

func XDoHTTP(t *testing.T, reg *registry.Registry, method string, path string,
	bodyStr string) *HTTPResult {

	XNoErr(t, reg.SaveModel())
	XNoErr(t, reg.SaveAllAndCommit())

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
	XNoErr(t, err)

	doRes, err := client.Do(req)
	XNoErr(t, err)

	result := &HTTPResult{
		Response: *doRes,
	}

	if doRes != nil {
		tmp, _ := io.ReadAll(doRes.Body)
		result.body = string(tmp)
	}
	return result
}

func XCheckHTTP(t *testing.T, reg *registry.Registry, test *HTTPTest, flags ...string) {
	t.Helper()
	XNoErr(t, reg.SaveModel())
	XNoErr(t, reg.SaveAllAndCommit())

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
	XNoErr(t, err)

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

	XNoErr(t, err)
	if test.Code < 10 {
		XCheck(t, int(res.StatusCode/100) == test.Code,
			"Expected status %dxx, got %d\n%s",
			test.Code, res.StatusCode, string(resBody))
	} else {
		XCheck(t, res.StatusCode == test.Code,
			"Expected status %d, got %d\n%s",
			test.Code, res.StatusCode, string(resBody))
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
	TSre := SavedREs[REG_RFC3339]

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
		if re = SavedREs[search]; re == nil {
			re = regexp.MustCompile(search)
			SavedREs[search] = re
		}
		headerMasks = append(headerMasks, re)
		headerReplace = append(headerReplace, replace)
	}

	list := ToJSON(resHeaders)
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

		// t.Logf("Body: %s", string(resBody))
		XEqual(t, "Headers:\n"+list+"\n\nBad Header:"+name+"\n",
			resValue, value, flags...)
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
	if strings.HasPrefix(test.ResBody, "^") {
		re := regexp.MustCompile(test.ResBody)
		if !re.Match(resBody) {
			t.Fatalf("Test: %s\nExpected:\n%s\nGot:\n%s",
				test.Name, test.ResBody, string(resBody))
			t.FailNow()
		}
	} else if test.ResBody != "*" {
		testBody := test.ResBody

		for _, mask := range test.BodyMasks {
			var re *regexp.Regexp
			search, replace, found := strings.Cut(mask, "||")
			if !found {
				// Must be just a property name
				search = fmt.Sprintf(`("%s": ")(.*)(")`, search)
				replace = `${1}xxx${3}`
			}

			if re = SavedREs[search]; re == nil {
				re = regexp.MustCompile(search)
				SavedREs[search] = re
			}

			resBody = re.ReplaceAll(resBody, []byte(replace))
			testBody = re.ReplaceAllString(testBody, replace)
		}

		XEqual(t, "Test: "+test.Name+"\nBody:\n",
			string(resBody), testBody, flags...)
		if t.Failed() {
			t.FailNow()
		}
	}
}

func XCLIServer(serverURL string) {
	os.Setenv("XR_SERVER", serverURL)
}

func XCLI(t *testing.T, line string, in, Eout, Eerr string, work bool) {
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

	if !IsNil(err) && work {
		t.Fatalf("Should have worked: %s\nStdout: %s\nStderr: %s",
			err, stdout.String(), stderr.String())
	} else if IsNil(err) && !work {
		t.Fatalf("Should have failed:\nStdout: %s\nStderr: %s",
			stdout.String(), stderr.String())
	}

	XEqual(t, "Stderr:", stderr.String(), Eerr)
	XEqual(t, "Stdout:", stdout.String(), Eout)
}

func XServer(t *testing.T, line string, in, Eout, Eerr string, code int) {
	t.Helper()

	args := strings.Split(strings.TrimSpace(line), " ")
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}

	cmd := exec.Command("../xrserver", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if in != "" {
		cmd.Stdin = bytes.NewBuffer([]byte(in))
	}

	err := cmd.Run()

	if !IsNil(err) && code == 0 {
		t.Fatalf("Should have worked: %s\nStdout: %s\nStderr: %s",
			err, stdout.String(), stderr.String())
	} else if IsNil(err) && code == 1 {
		t.Fatalf("Should have failed:\nStdout: %s\nStderr: %s",
			stdout.String(), stderr.String())
	}

	if Eerr != "*" {
		XEqual(t, "Stderr:", stderr.String(), Eerr, MASK_SERVER)
	}
	if Eout != "*" {
		XEqual(t, "Stdout:", stdout.String(), Eout, MASK_SERVER)
	}
}
