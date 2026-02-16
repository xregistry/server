package common

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	log "github.com/duglin/dlog"
)

type Capabilities struct {
	APIs           []string `json:"apis"`           // must not have omitempty
	Flags          []string `json:"flags"`          // must not have omitempty
	Ignores        []string `json:"ignores"`        // must not have omitempty
	Mutable        []string `json:"mutable"`        // must not have omitempty
	Pagination     bool     `json:"pagination"`     // must not have omitempty
	ShortSelf      bool     `json:"shortself"`      // must not have omitempty
	SpecVersions   []string `json:"specversions"`   // must not have omitempty
	StickyVersions *bool    `json:"stickyversions"` // must not have omitempty
	VersionModes   []string `json:"versionmodes"`   // must not have omitempty
}

type OfferedCapability struct {
	Type          string       `json:"type,omitempty"`
	Item          *OfferedItem `json:"item,omitempty"`
	Enum          []any        `json:"enum,omitempty"`
	Min           any          `json:"min,omitempty"`
	Max           any          `json:"max,omitempty"`
	Documentation string       `json:"documentation,omitempty"`
}

type OfferedItem struct {
	Type string `json:"type,omitempty"`
}

type Offered struct {
	APIs           OfferedCapability `json:"apis,omitempty"`
	Flags          OfferedCapability `json:"flags,omitempty"`
	Ignores        OfferedCapability `json:"ignores,omitempty"`
	Mutable        OfferedCapability `json:"mutable,omitempty"`
	Pagination     OfferedCapability `json:"pagination,omitempty"`
	ShortSelf      OfferedCapability `json:"shortself,omitempty"`
	SpecVersions   OfferedCapability `json:"specversions,omitempty"`
	StickyVersions OfferedCapability `json:"stickyversions,omitempty"`
	VersionModes   OfferedCapability `json:"versionmodes,omitempty"`
}

var AllowableAPIs = ArrayToLower([]string{
	"/capabilities", "/capabilitiesoffered", "/export",
	"/model", "/modelsource"})

var AllowableFlags = ArrayToLower([]string{
	"binary", "collections", "doc", "epoch", "filter", "ignore", "inline",
	"setdefaultversionid", "sort", "specversion"})

var AllowableIgnores = ArrayToLower([]string{
	"capabilities", "defaultversionid", "defaultversionsticky", "epoch",
	"modelsource", "readonly"})

var AllowableMutable = ArrayToLower([]string{
	"capabilities", "entities", "model"})

var AllowableSpecVersions = ArrayToLower([]string{"1.0-rc2", SPECVERSION})

var AllowableVersionModes = ArrayToLower([]string{"manual", "createdat"})

var SupportedFlags = ArrayToLower([]string{
	"binary", "collections", "doc", "epoch", "filter", "ignore", "inline",
	"setdefaultversionid", "sort", "specversion"})

var SupportedIgnores = ArrayToLower([]string{
	"capabilities", "defaultversionid", "defaultversionsticky", "epoch",
	"modelsource", "readonly"})

var DefaultCapabilities = &Capabilities{
	APIs:           AllowableAPIs,
	Flags:          SupportedFlags,
	Ignores:        SupportedIgnores,
	Mutable:        AllowableMutable,
	Pagination:     false,
	ShortSelf:      false,
	SpecVersions:   AllowableSpecVersions,
	StickyVersions: PtrBool(true),
	VersionModes:   AllowableVersionModes,
}

func init() {
	sort.Strings(AllowableAPIs)
	sort.Strings(AllowableFlags)
	sort.Strings(AllowableIgnores)
	sort.Strings(AllowableMutable)
	sort.Strings(AllowableSpecVersions)
	sort.Strings(AllowableVersionModes)

	sort.Strings(SupportedFlags)
	sort.Strings(SupportedIgnores)

	Must(DefaultCapabilities.Validate())
}

func String2AnySlice(strs []string) []any {
	res := make([]any, len(strs))

	for i, v := range strs {
		res[i] = v
	}

	return res
}

func GetOffered() *Offered {
	offered := &Offered{
		APIs: OfferedCapability{
			Type: "array",
			Item: &OfferedItem{
				Type: "string",
			},
			Enum: String2AnySlice(AllowableAPIs),
		},
		Flags: OfferedCapability{
			Type: "array",
			Item: &OfferedItem{
				Type: "string",
			},
			Enum: String2AnySlice(SupportedFlags),
		},
		Ignores: OfferedCapability{
			Type: "array",
			Item: &OfferedItem{
				Type: "string",
			},
			Enum: String2AnySlice(SupportedIgnores),
		},
		Mutable: OfferedCapability{
			Type: "string",
			Enum: String2AnySlice(AllowableMutable),
		},
		Pagination: OfferedCapability{
			Type: "boolean",
			Enum: []any{false},
		},
		ShortSelf: OfferedCapability{
			Type: "boolean",
			Enum: []any{false},
		},
		SpecVersions: OfferedCapability{
			Type: "array",
			Item: &OfferedItem{
				Type: "string",
			},
			Enum: String2AnySlice(AllowableSpecVersions),
		},
		StickyVersions: OfferedCapability{
			Type: "boolean",
			Enum: []any{false, true},
		},
		VersionModes: OfferedCapability{
			Type: "array",
			Item: &OfferedItem{
				Type: "string",
			},
			Enum: String2AnySlice(AllowableVersionModes),
		},
	}

	return offered
}

func ArrayToLower(arr []string) []string {
	for i, s := range arr {
		arr[i] = strings.ToLower(s)
	}
	arr = slices.Compact(arr) // remove dups
	return arr
}

func CleanArray(arr []string, full []string, text string) ([]string, *XRError) {
	// Make a copy so we can tweak it
	arr = slices.Clone(arr)

	// Lowercase evrything and look for "*"
	for i, s := range arr {
		s = strings.ToLower(s)

		arr[i] = s
		if s == "*" {
			if len(arr) != 1 {
				return nil, NewXRError("capability_wildcard", "/capabilities",
					"field="+text)
			}
			return full, nil
		}

	}

	sort.Strings(arr)         // sort 'em
	arr = slices.Compact(arr) // remove dups
	if len(arr) == 0 {
		arr = []string{}
	}

	// Now look for valid values
	ai, fi := len(arr)-1, len(full)-1
	for ai >= 0 && fi >= 0 {
		as, fs := arr[ai], full[fi]
		if as == fs {
			ai--
		} else if as > fs {
			return nil, NewXRError("capability_value", "/capabilities",
				"value="+as,
				"field="+text,
				"list="+strings.Join(full, ","))
		}
		fi--
	}
	if ai < 0 {
		return arr, nil
	}
	return nil, NewXRError("capability_error", "/capabilities",
		"error_detail="+
			fmt.Sprintf(`unknown %q value: %q`, text, arr[ai]))
}

func (c *Capabilities) Validate() *XRError {
	var xErr *XRError

	c.APIs, xErr = CleanArray(c.APIs, AllowableAPIs, "apis")
	if xErr != nil {
		return xErr
	}

	c.Flags, xErr = CleanArray(c.Flags, AllowableFlags, "flags")
	if xErr != nil {
		return xErr
	}

	c.Ignores, xErr = CleanArray(c.Ignores, AllowableIgnores, "ignores")
	if xErr != nil {
		return xErr
	}

	c.Mutable, xErr = CleanArray(c.Mutable, AllowableMutable, "mutable")
	if xErr != nil {
		return xErr
	}

	if c.Pagination != false {
		return NewXRError("capability_value", "/capabilities",
			"value=true",
			"field=pagination",
			"list=false")
	}

	if c.ShortSelf != false {
		return NewXRError("capability_value", "/capabilities",
			"value=true",
			"field=shortself",
			"list=false")
	}

	if c.SpecVersions == nil {
		c.SpecVersions = []string{SPECVERSION}
	}

	c.SpecVersions, xErr = CleanArray(c.SpecVersions, AllowableSpecVersions,
		"specversions")
	if xErr != nil {
		return xErr
	}

	if !ArrayContainsAnyCase(c.SpecVersions, SPECVERSION) {
		return NewXRError("capability_missing_value", "/capabilities",
			"name="+"specversions",
			"value="+SPECVERSION)
	}

	if c.StickyVersions == nil {
		c.StickyVersions = DefaultCapabilities.StickyVersions
	}

	if c.VersionModes == nil {
		c.VersionModes = []string{VERSIONMODE}
	}

	c.VersionModes, xErr = CleanArray(c.VersionModes, AllowableVersionModes,
		"versionmodes")
	if xErr != nil {
		return xErr
	}

	if !ArrayContainsAnyCase(c.VersionModes, VERSIONMODE) {
		return NewXRError("capability_missing_value", "/capabilities",
			"name=versionmodes",
			"value="+VERSIONMODE)
	}

	return nil
}

func ParseCapabilitiesJSON(buf []byte) (*Capabilities, *XRError) {
	log.VPrintf(4, "Enter: ParseCapabilitiesJSON")
	cap := Capabilities{}

	err := Unmarshal(buf, &cap)
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown field ") {
			field, _, _ := strings.Cut(err.Error()[15:], "\"")
			return nil, NewXRError("capability_unknown", "/capabilities",
				"field="+field)
		}
		return nil, NewXRError("capability_error", "/capabilities",
			"error_detail="+fmt.Sprintf("error parsing data: %s", err))
	}
	return &cap, nil
}

func (c *Capabilities) APIEnabled(str string) bool {
	return ArrayContains(c.APIs, str)
}

func (c *Capabilities) FlagEnabled(str string) bool {
	return ArrayContains(c.Flags, str)
}

func (c *Capabilities) IgnoresEnabled(str string) bool {
	return ArrayContains(c.Ignores, str)
}

func (c *Capabilities) MutableEnabled(str string) bool {
	return ArrayContains(c.Mutable, str)
}

func (c *Capabilities) PaginationEnabled() bool {
	return c.Pagination
}

func (c *Capabilities) ShortSelfEnabled(str string) bool {
	return c.ShortSelf
}

func (c *Capabilities) SpecVersionEnabled(str string) bool {
	return ArrayContains(c.SpecVersions, str)
}

func (c *Capabilities) StickyVersionsEnabled() bool {
	return c.StickyVersions != nil && (*c.StickyVersions) == true
}

func (c *Capabilities) VersionModeEnabled(str string) bool {
	return ArrayContains(c.VersionModes, str)
}
