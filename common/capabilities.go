package common

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	log "github.com/duglin/dlog"
)

type Capabilities struct {
	// THESE MUST NOT HAVE "omitempty" on them
	Available       map[string]*AvailableObject `json:"available"`
	Compatibilities map[string][]string         `json:"compatibilities"`
	Flags           []string                    `json:"flags"`
	Formats         []string                    `json:"formats"`
	Ignores         []string                    `json:"ignores"`
	Pagination      bool                        `json:"pagination"`
	ShortSelf       bool                        `json:"shortself"`
	SpecVersions    []string                    `json:"specversions"`
	VersionModes    []string                    `json:"versionmodes"`
}

type AvailableObject struct {
	Mutable bool `json:"mutable"`
}

type OfferedCapability struct {
	Type          string                        `json:"type,omitempty"`
	Enum          []any                         `json:"enum,omitempty"`
	Min           any                           `json:"min,omitempty"`
	Max           any                           `json:"max,omitempty"`
	Documentation string                        `json:"documentation,omitempty"`
	Attributes    map[string]*OfferedCapability `json:"attributes,omitempty"`
	Item          *OfferedItem                  `json:"item,omitempty"`
}

type OfferedItem struct {
	Type       string                        `json:"type,omitempty"`
	Attributes map[string]*OfferedCapability `json:"attributes,omitempty"`
	Item       *OfferedItem                  `json:"item,omitempty"`
}

type Offered struct {
	Available       OfferedCapability `json:"available,omitempty"`
	Compatibilities OfferedCapability `json:"compatibilities,omitempty"`
	Flags           OfferedCapability `json:"flags,omitempty"`
	Formats         OfferedCapability `json:"formats,omitempty"`
	Ignores         OfferedCapability `json:"ignores,omitempty"`
	Pagination      OfferedCapability `json:"pagination,omitempty"`
	ShortSelf       OfferedCapability `json:"shortself,omitempty"`
	SpecVersions    OfferedCapability `json:"specversions,omitempty"`
	VersionModes    OfferedCapability `json:"versionmodes,omitempty"`
}

var SupportedAvailable = map[string]*AvailableObject{
	"capabilities": &AvailableObject{
		Mutable: true,
	},
	"capabilitiesoffered": &AvailableObject{
		Mutable: false,
	},
	"entities": &AvailableObject{
		Mutable: true,
	},
	"export": &AvailableObject{
		Mutable: false,
	},
	"model": &AvailableObject{
		Mutable: false,
	},
	"modelsource": &AvailableObject{
		Mutable: true,
	},
}

var SupportedCompatibilities = map[string][]string{}

var SupportedFlags = ArrayToLower([]string{
	"binary", "collections", "doc", "epoch", "filter", "ignore", "inline",
	"setdefaultversionid", "sort", "specversion"})

var SupportedFormats = []string{}

var SupportedIgnores = ArrayToLower([]string{
	"capabilities", "defaultversionid", "defaultversionsticky", "epoch", "id",
	"modelsource", "readonly"})

var SupportedSpecVersions = ArrayToLower([]string{"1.0-rc3", SPECVERSION})

var SupportedVersionModes = ArrayToLower([]string{"manual", "createdat"})

var DefaultCapabilities = &Capabilities{
	Available:       SupportedAvailable,
	Compatibilities: SupportedCompatibilities,
	Flags:           SupportedFlags,
	Formats:         SupportedFormats,
	Ignores:         SupportedIgnores,
	Pagination:      false,
	ShortSelf:       false,
	SpecVersions:    SupportedSpecVersions,
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
	sort.Strings(SupportedFlags)
	sort.Strings(SupportedFormats)
	sort.Strings(SupportedIgnores)
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
		Available: OfferedCapability{
			Type: "object",
			Attributes: map[string]*OfferedCapability{
				"capabilities": &OfferedCapability{
					Type: "object",
					Attributes: map[string]*OfferedCapability{
						"mutable": {Type: "boolean"}},
				},
				"capabilitiesoffered": &OfferedCapability{
					Type: "object",
					Attributes: map[string]*OfferedCapability{
						"mutable": {Type: "boolean"}},
				},
				"entities": &OfferedCapability{
					Type: "object",
					Attributes: map[string]*OfferedCapability{
						"mutable": {Type: "boolean"}},
				},
				"export": &OfferedCapability{
					Type: "object",
					Attributes: map[string]*OfferedCapability{
						"mutable": {Type: "boolean", Enum: []any{false}}},
				},
				"model": &OfferedCapability{
					Type: "object",
					Attributes: map[string]*OfferedCapability{
						"mutable": {Type: "boolean", Enum: []any{false}}},
				},
				"modelsource": &OfferedCapability{
					Type: "object",
					Attributes: map[string]*OfferedCapability{
						"mutable": {Type: "boolean"}},
				},
			},
		},
		Compatibilities: OfferedCapability{}, // Do it below
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
		VersionModes: OfferedCapability{
			Type: "array",
			Item: &OfferedItem{
				Type: "string",
			},
			Enum: String2AnySlice(SupportedVersionModes),
		},
	}

	// Dynamically create the Compatibility offering
	offered.Compatibilities.Type = "object"
	for format, list := range SupportedCompatibilities {
		if offered.Compatibilities.Attributes == nil {
			offered.Compatibilities.Attributes = map[string]*OfferedCapability{}
		}
		offered.Compatibilities.Attributes[format] = &OfferedCapability{
			Type: "array",
			Enum: String2AnySlice(list),
			Item: &OfferedItem{
				Type: "string",
			},
		}
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

	for avail, availObj := range c.Available {
		supportedAvailObj, ok := SupportedAvailable[avail]
		if !ok {
			extra := ""
			if _, ok = SupportedAvailable[strings.ToLower(avail)]; ok {
				extra = ". Wrong case, try: " + strings.ToLower(avail)
			}
			return NewXRError("capability_error", "/capabilities",
				fmt.Sprintf("error_detail=Unknown \"available\" value: %s%s",
					avail, extra))
		}
		if supportedAvailObj.Mutable == false && availObj.Mutable == true {
			return NewXRError("capability_error", "/capabilities",
				fmt.Sprintf("error_detail=\"available\" value %q is not "+
					"allowed to be mutable", avail))
		}
	}

	if c.Available == nil {
		c.Available = map[string]*AvailableObject{}
	}
	if _, ok := c.Available["entities"]; !ok {
		c.Available["entities"] = &AvailableObject{
			Mutable: true,
		}
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

	// Validate specversions using normalized comparison
	// (patch level is ignored), but preserve the original
	// (lowercased) values for storage/display.
	hasCurrentSpecVersion := false
	for i, sv := range c.SpecVersions {
		c.SpecVersions[i] = strings.ToLower(sv)
		norm := NormalizeSpecVersion(sv)
		if !ArrayContains(SupportedSpecVersions, norm) {
			return NewXRError("capability_value", "/capabilities",
				"value="+strings.ToLower(sv),
				"field=specversions",
				"list="+strings.Join(SupportedSpecVersions, ","))
		}
		if norm == NormalizeSpecVersion(SPECVERSION) {
			hasCurrentSpecVersion = true
		}
	}
	sort.Strings(c.SpecVersions)
	c.SpecVersions = slices.Compact(c.SpecVersions)
	if !hasCurrentSpecVersion {
		return NewXRError("capability_missing_value", "/capabilities",
			"name="+"specversions",
			"value="+SPECVERSION)
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

func ParseCapabilities(buf []byte) (*Capabilities, *XRError) {
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

func (c *Capabilities) SetAvailable(data string, mutable bool) {
	if c.Available == nil {
		c.Available = map[string]*AvailableObject{}
	}
	c.Available[data] = &AvailableObject{
		Mutable: mutable,
	}
}

// Is the data GET-able
func (c *Capabilities) IsAvailable(str string) bool {
	if c.Available == nil {
		return false
	}

	_, ok := c.Available[str]
	return ok
}

// Is the data GET-able and mutable
func (c *Capabilities) IsAvailableMutable(str string) bool {
	if c.Available == nil {
		return false
	}

	avail, ok := c.Available[str]
	return ok && avail.Mutable
}

func (c *Capabilities) FlagEnabled(str string) bool {
	return ArrayContains(c.Flags, str)
}

func (c *Capabilities) IgnoresEnabled(str string) bool {
	return ArrayContains(c.Ignores, str)
}

func (c *Capabilities) PaginationEnabled() bool {
	return c.Pagination
}

func (c *Capabilities) ShortSelfEnabled(str string) bool {
	return c.ShortSelf
}

// NormalizeSpecVersion strips any patch-level version component for comparison
// per spec: only major.minor is compared; any suffix after the first "-" is
// preserved as-is. Input is lowercased. Examples:
//
//	"1.0.1-rc2" -> "1.0-rc2"
//	"1.0.5"     -> "1.0"
//	"1.0-rc2"   -> "1.0-rc2"
//	"1.0-RC2"   -> "1.0-rc2"
func NormalizeSpecVersion(sv string) string {
	sv = strings.ToLower(sv)
	base, suffix, hasSuffix := strings.Cut(sv, "-")
	parts := strings.SplitN(base, ".", 3)
	if len(parts) >= 2 {
		base = parts[0] + "." + parts[1]
	}
	if hasSuffix {
		return base + "-" + suffix
	}
	return base
}

func (c *Capabilities) SpecVersionEnabled(str string) bool {
	norm := NormalizeSpecVersion(str)
	for _, sv := range c.SpecVersions {
		if NormalizeSpecVersion(sv) == norm {
			return true
		}
	}
	return false
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

func (c *Capabilities) Clone() *Capabilities {
	buf, _ := json.Marshal(c)
	newCaps := (*Capabilities)(nil)
	json.Unmarshal(buf, &newCaps)
	return newCaps
}
