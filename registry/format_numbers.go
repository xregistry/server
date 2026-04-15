package registry

// Just a test file "format".
// It'll add up all of the integers on each line of the file.
// Each line number either be blank or an int, anything else fails.
// Files are compatible if newfile's sum >= old file sum.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

const NUMBERS_FORMAT = "numbers"

func init() {
	RegisterFormat(NUMBERS_FORMAT, FormatNumbers{})
}

type FormatNumbers struct{}

func GetVersionSum(ver *Version) (int, string, *XRError) {
	format := ver.GetAsString("format")
	if ok, _ := regexp.MatchString("(?i)"+NUMBERS_FORMAT, format); !ok {
		return 0, "true", NewXRError("bad_request", ver.XID,
			"error_detail="+
				fmt.Sprintf(`Version %q has a "format" value of %q, was `+
					`expecting %q`, ver.XID, format, NUMBERS_FORMAT))
	}

	if ver.Resource.ResourceModel.GetHasDocument() == false {
		return 0, "true", NewXRError("format_violation", ver.XID,
			"format="+format).
			SetDetailf(`The Resource (%s) for Version %q does not have `+
				`"hasdocument" in its resource model set to "true", and an `+
				`empty/missing document is not compliant.`,
				ver.Resource.XID, ver.XID)
	}

	if resURL := ver.Get(ver.Resource.Singular + "url"); !IsNil(resURL) {
		return 0, "false, data stored externally",
			NewXRError("format_external", ver.XID)
	}

	buf := []byte(nil)
	if bufAny := ver.Get(ver.Resource.Singular); !IsNil(bufAny) {
		buf = bufAny.([]byte)
	}

	if len(buf) == 0 {
		return 0, "true", NewXRError("format_violation", ver.XID,
			"format="+ver.GetAsString("format")).
			SetDetailf("Version %q is empty and therefore not a "+
				"valid numbers file.", ver.XID)
	}

	sum := 0
	for num, line := range strings.Split(string(buf), "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		i, err := strconv.Atoi(line)
		if err != nil {
			return 0, "true", NewXRError("format_violation", ver.XID,
				"format=numbers").
				SetDetailf("Line %d isn't an integer: %s.", num+1, line)
		}
		sum += i
	}

	// all ok - so just exit
	return sum, "true", nil
}

func (ft FormatNumbers) IsValid(ver *Version) (string, *XRError) {
	log.VPrintf(3, ">Enter: FormatNumbers.IsValid(%s)", ver.UID)
	defer log.VPrintf(3, "<Exit: FormatNumbers.IsValid")

	_, reason, xErr := GetVersionSum(ver)
	return reason, xErr
}

func (ft FormatNumbers) IsCompatible(direction string, oldVer, newVer *Version) (string, *XRError) {
	log.VPrintf(3, ">Enter: IsCompliant.IsCompliant(old:%s,new:%s)",
		oldVer.UID, newVer.UID)
	defer log.VPrintf(3, "<Exit: FormatNumbers.IsCompliant")

	oldSum, reason, xErr := GetVersionSum(oldVer)
	if xErr != nil {
		return reason, xErr
	}
	newSum, reason, xErr := GetVersionSum(newVer)
	if xErr != nil {
		return reason, xErr
	}

	if newSum < oldSum {
		compat := newVer.Resource.MustFindMeta(false, FOR_READ).
			GetAsString("compatibility")
		return "false", NewXRError("compatibility_violation",
			newVer.Resource.XID, "compat="+compat).
			SetDetail(fmt.Sprintf("Version %q (sum: %d) isn't %q compatible "+
				"with %q (sum: %d).",
				newVer.XID, newSum, compat, oldVer.XID, oldSum))
	}

	return "true", nil
}
