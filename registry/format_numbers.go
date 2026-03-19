package registry

// Just a test file "format"
// It'll add up all of the integers on each line of the file.
// Each line number either be blank or an int, anything else fails
// files are compatible if newfile's sum >= old file sum

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

func init() {
	RegisterFormat("numbers", FormatNumbers{})
}

type FormatNumbers struct{}

func GetVersionSum(version *Version) (int, *XRError) {
	sum := 0
	bufAny := version.Get(version.Resource.Singular)
	if !IsNil(bufAny) {
		buf := bufAny.([]byte)
		if len(buf) != 0 {
			for num, line := range strings.Split(string(buf), "\n") {
				line = strings.TrimSpace(line)
				if len(line) == 0 {
					continue
				}
				i, err := strconv.Atoi(line)
				if err != nil {
					return 0, NewXRError("format_violation", version.XID,
						"format=numbers").
						SetDetail(fmt.Sprintf("Line %d isn't an integer: %s.",
							num+1, line))
				}
				sum += i
			}
			// all ok - so just exit
		}
	}
	return sum, nil
}

func (ft FormatNumbers) IsValid(version *Version) *XRError {
	log.VPrintf(3, ">Enter: FormatNumbers.IsValid(%s)", version.UID)
	defer log.VPrintf(3, "<Exit: FormatNumbers.IsValid")

	_, xErr := GetVersionSum(version)
	return xErr
}

func (ft FormatNumbers) IsCompatible(oldVersion, newVersion *Version) *XRError {
	log.VPrintf(3, ">Enter: IsCompliant.IsCompliant(old:%s,new:%s)",
		oldVersion.UID, newVersion.UID)
	defer log.VPrintf(3, "<Exit: FormatNumbers.IsCompliant")

	/*
		log.Printf("Checking compat of old:%s/new:%s for format 'numbers'",
			oldVersion.UID, newVersion.UID)
	*/

	oldSum, xErr := GetVersionSum(oldVersion)
	if xErr != nil {
		return xErr
	}
	newSum, xErr := GetVersionSum(newVersion)
	if xErr != nil {
		return xErr
	}

	// log.Printf("Comparing %v < %v", newSum, oldSum)
	if newSum < oldSum {
		compat := newVersion.
			Resource.
			MustFindMeta(false, FOR_READ).
			GetAsString("compatibility")
		return NewXRError("compatibility_violation", newVersion.Resource.XID,
			"value="+compat).
			SetDetail(fmt.Sprintf("Version %q (sum: %d) isn't %q compatible "+
				"with %q (sum: %d).",
				newVersion.XID, newSum, compat, oldVersion.XID, oldSum))
	}

	return nil
}
