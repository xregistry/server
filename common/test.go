package common

import (
	"fmt"
	"regexp"
	"testing"
)

var NOMASK_TS = "NoMaskTS"
var TSREGEXP = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[-+]\d{2}:\d{2})`
var TSMASK = TSREGEXP + `||YYYY-MM-DDTHH:MM:SSZ`

var SavedREs = map[string]*regexp.Regexp{
	TSREGEXP: regexp.MustCompile(TSREGEXP),
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

	re := SavedREs[TSREGEXP]
	return re.ReplaceAllStringFunc(input, replaceFunc)
}

func XEqual(t *testing.T, extra string, gotAny any, expAny any, flags ...string) {
	t.Helper()
	pos := 0

	exp := fmt.Sprintf("%v", expAny)
	got := fmt.Sprintf("%v", gotAny)

	if exp == "*" {
		return
	}

	flagsMap := map[string]bool{}
	for _, f := range flags {
		flagsMap[f] = true
	}

	// expected output starting with "--TS--" means "skip timestamp masking"
	if !flagsMap[NOMASK_TS] {
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
