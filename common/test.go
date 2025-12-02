package common

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// Flags
var NOMASK_TS = "NoMaskTS"
var MASK_SERVER = "MaskServer"
var NOMASK_INSTANCE = "NoMaskInstance"

var REG_RFC3339 = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[-+]\d{2}:\d{2})`
var REG_TSSLASH = `\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`
var REG_COMMIT = `GitCommit: [0-9a-f]*\n`
var REG_DBHOST = `DB server: .*`
var REG_INSTANCE = `"source": "[^"]*"`

var SavedREs = map[string]*regexp.Regexp{
	REG_RFC3339:  regexp.MustCompile(REG_RFC3339),
	REG_TSSLASH:  regexp.MustCompile(REG_TSSLASH),
	REG_COMMIT:   regexp.MustCompile(REG_COMMIT),
	REG_DBHOST:   regexp.MustCompile(REG_DBHOST),
	REG_INSTANCE: regexp.MustCompile(REG_INSTANCE),
}

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

	re := SavedREs[REG_RFC3339]
	return re.ReplaceAllStringFunc(input, replaceFunc)
}

func XEqual(t *testing.T, extra string, gotAny any, expAny any, flags ...string) {
	t.Helper()
	pos := 0

	exp := fmt.Sprintf("%v", expAny)
	got := fmt.Sprintf("%v", gotAny)

	orig := "\nOrigGot: " + got + "\n"

	if exp == "*" {
		return
	}

	flagsMap := map[string]bool{}
	for _, f := range flags {
		flagsMap[f] = true
	}

	// See if they asked us to NOT mask timestamps
	if !flagsMap[NOMASK_TS] {
		got = MaskTimestamps(got)
		exp = MaskTimestamps(exp)
	}

	if flagsMap[MASK_SERVER] {
		got = SavedREs[REG_TSSLASH].ReplaceAllString(got, "YYYY/MM/DD HH:MM:SS")
		exp = SavedREs[REG_TSSLASH].ReplaceAllString(exp, "YYYY/MM/DD HH:MM:SS")

		got = SavedREs[REG_COMMIT].ReplaceAllString(got, "GitCommit: sha\n")
		exp = SavedREs[REG_COMMIT].ReplaceAllString(exp, "GitCommit: sha\n")

		got = SavedREs[REG_DBHOST].ReplaceAllString(got, "DB server: host:port")
		exp = SavedREs[REG_DBHOST].ReplaceAllString(exp, "DB server: host:port")
	}

	if !flagsMap[NOMASK_INSTANCE] {
		got = SavedREs[REG_INSTANCE].ReplaceAllString(got, `"source": "xxx"`)
		exp = SavedREs[REG_INSTANCE].ReplaceAllString(exp, `"source": "xxx"`)
	}

	for pos < len(got) && pos < len(exp) && got[pos] == exp[pos] {
		pos++
	}
	if pos == len(got) && pos == len(exp) {
		return
	}

	if pos == len(got) {
		t.Fatalf(extra+orig+
			"\nExpected:\n"+exp+
			"\nGot:\n"+got+
			"\nGot ended early at(%d)[%02X]:\n%q",
			pos, exp[pos], got[pos:])
	}

	if pos == len(exp) {
		t.Fatalf(extra+orig+
			"\nExpected:\n"+exp+
			"\nGot:\n"+got+
			"\nExp ended early at(%d)[%02X]:\n"+got[pos:],
			pos, got[pos])
	}

	expMax := pos + 90
	if expMax > len(exp) {
		expMax = len(exp)
	}

	t.Fatalf(extra+orig+
		"\nExpected:\n"+exp+
		"\nGot:\n"+got+
		"\nDiff at(%d)[x%0x/x%0x]:"+
		"\nExp subset:\n"+exp[pos:expMax]+
		"\nGot:\n"+got[pos:],
		pos, exp[pos], got[pos])
}

// got, any
func XCheckErr(t *testing.T, errAny any, errStr string) {
	t.Helper()

	if IsNil(errAny) {
		if errStr == "" {
			return
		}
		t.Fatalf("\nGot:<no err>\nExp: %s", errStr)
	}

	if errStr == "" {
		t.Fatalf("Test failed: %s", errAny)
	}

	XEqual(t, "", errAny, errStr)
}

func XCheck(t *testing.T, b bool, errStr string, args ...any) {
	t.Helper()
	if !b {
		t.Fatalf(errStr, args...)
	}
}

func Fail(t *testing.T, str string, args ...any) {
	t.Helper()
	text := strings.TrimSpace(fmt.Sprintf(str, args...))
	t.Fatalf("%s\n\n", text)
}

func XNoErr(t *testing.T, errAny any) {
	t.Helper()
	if !IsNil(errAny) {
		t.Fatalf("Unexpected error: %s", errAny)
	}
}

func XCheckNotEqual(t *testing.T, extra string, gotAny any, expAny any) {
	t.Helper()

	exp := fmt.Sprintf("%v", expAny)
	got := fmt.Sprintf("%v", gotAny)

	if exp != got {
		return
	}

	t.Fatalf("Should differ, but they're both:\n%s", exp)
}

func XCheckGreater(t *testing.T, extra string, newAny any, oldAny any) {
	t.Helper()

	New := fmt.Sprintf("%v", newAny)
	Old := fmt.Sprintf("%v", oldAny)

	if New > Old {
		return
	}

	t.Fatalf("New not > Old:\nOld:\n%s\n\nNew:\n%s", Old, New)
}

// http code, body
func XGET(t *testing.T, url string) (int, string) {
	t.Helper()
	url = "http://localhost:8181/" + url
	res, err := http.Get(url)
	if !IsNil(err) {
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

func XJSONCheck(t *testing.T, gotObj any, expObj any) {
	t.Helper()
	got := ToJSON(gotObj)
	exp := ToJSON(expObj)
	XEqual(t, "", got, exp)
}

/*
func XErrDiff(t *testing.T, gotErrAny any, expErrAny any) {
	var gotErr *XRError
	var expErr *XRError

	if str, ok := gotErrAny.(string); ok {
		if err := json.Unmarshal([]byte(str), &gotErr); err != nil {
			panic("Should be a json err: " + str + ":" + err.Error())
		}
	} else if gotErr, ok = gotErrAny.(*XRError); !ok {
		panic("Unknown gotErrAny:" + fmt.Sprintf("%v", gotErrAny))
	}

	if str, ok := expErrAny.(string); ok {
		if err := json.Unmarshal([]byte(str), &expErr); err != nil {
			panic("Should be a json err: " + str + ":" + err.Error())
		}
	} else if gotErr, ok = gotErrAny.(*XRError); !ok {
		panic("Unknown expErrAny:" + fmt.Sprintf("%v", expErrAny))
	}

	XEqual(t, "Err.Type", gotErr.Type, expErr.Type)
	XEqual(t, "Err.Title", gotErr.Title, expErr.Title)
    ...
}
*/
