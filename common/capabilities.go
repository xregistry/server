package common

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	log "github.com/duglin/dlog"
)

type Capabilities struct {
	// THESE MUST NOT HAVE "omitempty" on them
	APIs            []string            `json:"apis"`
	Compatibilities map[string][]string `json:"compatibilities"`
	Flags           []string            `json:"flags"`
	Formats         []string            `json:"formats"`
	Ignores         []string            `json:"ignores"`
	Mutable         []string            `json:"mutable"`
	Pagination      bool                `json:"pagination"`
	ShortSelf       bool                `json:"shortself"`
	SpecVersions    []string            `json:"specversions"`
	StickyVersions  *bool               `json:"stickyversions"`
	VersionModes    []string            `json:"versionmodes"`
}

type OfferedCapability struct {
	Type          string              `json:"type,omitempty"`
	Item          *OfferedItem        `json:"item,omitempty"`
	Enum          []any               `json:"enum,omitempty"`
	Options       map[string][]string `json:"options,omitempty"`
	Min           any                 `json:"min,omitempty"`
	Max           any                 `json:"max,omitempty"`
	Documentation string              `json:"documentation,omitempty"`
}

type OfferedItem struct {
	Type string `json:"type,omitempty"`
}

type Offered struct {
	APIs            OfferedCapability `json:"apis,omitempty"`
	Compatibilities OfferedCapability `json:"compatibilities,omitempty"`
	Flags           OfferedCapability `json:"flags,omitempty"`
	Formats         OfferedCapability `json:"formats,omitempty"`
	Ignores         OfferedCapability `json:"ignores,omitempty"`
	Mutable         OfferedCapability `json:"mutable,omitempty"`
	Pagination      OfferedCapability `json:"pagination,omitempty"`
	ShortSelf       OfferedCapability `json:"shortself,omitempty"`
	SpecVersions    OfferedCapability `json:"specversions,omitempty"`
	StickyVersions  OfferedCapability `json:"stickyversions,omitempty"`
	VersionModes    OfferedCapability `json:"versionmodes,omitempty"`
}

var SupportedAPIs = ArrayToLower([]string{
	"/capabilities", "/capabilitiesoffered", "/export",
	"/model", "/modelsource"})

var SupportedCompatibilities = map[string][]string{}

var SupportedFlags = ArrayToLower([]string{
	"binary", "collections", "doc", "epoch", "filter", "ignore", "inline",
	"setdefaultversionid", "sort", "specversion"})

var SupportedFormats = []string{}

var SupportedIgnores = ArrayToLower([]string{
	"capabilities", "defaultversionid", "defaultversionsticky", "epoch", "id",
	"modelsource", "readonly"})

var SupportedMutable = ArrayToLower([]string{
	"capabilities", "entities", "model"})

var SupportedSpecVersions = ArrayToLower([]string{"1.0-rc2", SPECVERSION})

var SupportedVersionModes = ArrayToLower([]string{"manual", "createdat"})

var DefaultCapabilities = &Capabilities{
	APIs:            SupportedAPIs,
	Compatibilities: SupportedCompatibilities,
	Flags:           SupportedFlags,
	Formats:         SupportedFormats,
	Ignores:         SupportedIgnores,
	Mutable:         SupportedMutable,
	Pagination:      false,
	ShortSelf:       false,
	SpecVersions:    SupportedSpecVersions,
	StickyVersions:  PtrBool(true),
	VersionModes:    SupportedVersionModes,
}

func AddSupportedFormat(format string, compats []string) {
	PanicIf(format == "", "Can't be empty")

	SupportedFormats = append(SupportedFormats, format)
	sort.Strings(SupportedFormats)
	DefaultCapabilities.Formats = SupportedFormats

	if len(compats) != 0 {
		sort.Strings(compats)
		SupportedCompatibilities[format] = compats
	}

	Must(DefaultCapabilities.Validate())
}

func init() {
	sort.Strings(SupportedAPIs)
	sort.Strings(SupportedFlags)
	sort.Strings(SupportedFormats)
	sort.Strings(SupportedIgnores)
	sort.Strings(SupportedMutable)
	sort.Strings(SupportedSpecVersions)
	sort.Strings(SupportedVersionModes)

	for _, compats := range SupportedCompatibilities {
		sort.Strings(compats)
	}

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
			Enum: String2AnySlice(SupportedAPIs),
		},
		Compatibilities: OfferedCapability{
			Type: "map",
			Item: &OfferedItem{
				Type: "string",
			},
			Options: SupportedCompatibilities,
		},
		Flags: OfferedCapability{
			Type: "array",
			Item: &OfferedItem{
				Type: "string",
			},
			Enum: String2AnySlice(SupportedFlags),
		},
		Formats: OfferedCapability{
			Type: "array",
			Item: &OfferedItem{
				Type: "string",
			},
			Enum: String2AnySlice(SupportedFormats),
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
			Enum: String2AnySlice(SupportedMutable),
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
			Enum: String2AnySlice(SupportedSpecVersions),
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
			Enum: String2AnySlice(SupportedVersionModes),
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

	// Lowercase everything and look for "*"
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

	c.APIs, xErr = CleanArray(c.APIs, SupportedAPIs, "apis")
	if xErr != nil {
		return xErr
	}

	if c.Compatibilities == nil {
		c.Compatibilities = map[string][]string{}
	}
	for format, compats := range c.Compatibilities {
		if !ArrayContainsAnyCase(c.Formats, format) {
			return NewXRError("capability_value", "/capabilities",
				"field=compatibilities",
				"value="+format,
				"list="+strings.Join(c.Formats, ","))
		}

		c.Compatibilities[format], xErr =
			CleanArray(compats, SupportedCompatibilities[format],
				fmt.Sprintf("compatibilities[%s]", format))
		if xErr != nil {
			return xErr
		}
	}

	c.Flags, xErr = CleanArray(c.Flags, SupportedFlags, "flags")
	if xErr != nil {
		return xErr
	}

	c.Formats, xErr = CleanArray(c.Formats, SupportedFormats, "formats")
	if xErr != nil {
		return xErr
	}
	for _, format := range c.Formats {
		if !ArrayContainsAnyCase(c.Formats, format) {
			return NewXRError("capability_value", "/capabilities",
				"name=compatibilities",
				"value="+format,
				"list="+strings.Join(c.Formats, ","))
		}
	}

	c.Ignores, xErr = CleanArray(c.Ignores, SupportedIgnores, "ignores")
	if xErr != nil {
		return xErr
	}

	c.Mutable, xErr = CleanArray(c.Mutable, SupportedMutable, "mutable")
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

	c.SpecVersions, xErr = CleanArray(c.SpecVersions, SupportedSpecVersions,
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

	c.VersionModes, xErr = CleanArray(c.VersionModes, SupportedVersionModes,
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
	return ArrayContainsAnyCase(c.VersionModes, strings.ToLower(str))
}

func (c *Capabilities) FormatEnabled(str string) bool {
	return ArrayContainsAnyCase(c.Formats, str)
}

func (c *Capabilities) CompatibilityEnabled(format, compat string) bool {
	compatibilities, ok := c.Compatibilities[format]
	return ok && ArrayContainsAnyCase(compatibilities, compat)
}
