package registry

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	log "github.com/duglin/dlog"
)

type Capabilities struct {
	EnforceCompatibility bool     `json:"enforcecompatibility"`
	Flags                []string `json:"flags"`
	MaxMaxVersions       int      `json:"maxmaxversions"`
	Mutable              []string `json:"mutable"`
	Pagination           bool     `json:"pagination"`
	Schemas              []string `json:"schemas"`
	ShortSelf            bool     `json:"shortself"`
	SpecVersions         []string `json:"specversions"`
	Sticky               *bool    `json:"sticky"`
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
	EnforceCompatibility OfferedCapability `json:"enforcecompatibility,omitempty"`
	Flags                OfferedCapability `json:"flags,omitempty"`
	MaxMaxVersions       OfferedCapability `json:"maxmaxversions,omitempty"`
	Mutable              OfferedCapability `json:"mutable,omitempty"`
	Pagination           OfferedCapability `json:"pagination,omitempty"`
	Schemas              OfferedCapability `json:"schemas,omitempty"`
	ShortSelf            OfferedCapability `json:"shortself,omitempty"`
	SpecVersions         OfferedCapability `json:"specversions,omitempty"`
	Sticky               OfferedCapability `json:"sticky,omitempty"`
}

var AllowableFlags = ArrayToLower([]string{
	"doc", "epoch", "filter", "inline",
	"nodefaultversionid", "nodefaultversionsticky",
	"noepoch", "noreadonly", "offered",
	"schema", "setdefaultversionid", "specversion"})

var AllowableMutable = ArrayToLower([]string{
	"capabilities", "entities", "model"})

var AllowableSchemas = ArrayToLower([]string{XREGSCHEMA + "/" + SPECVERSION})

var AllowableSpecVersions = ArrayToLower([]string{"0.5"})

var DefaultCapabilities = &Capabilities{
	EnforceCompatibility: false,
	Flags:                AllowableFlags,
	MaxMaxVersions:       0,
	Mutable:              AllowableMutable,
	Pagination:           false,
	Schemas:              AllowableSchemas,
	ShortSelf:            false,
	SpecVersions:         AllowableSpecVersions,
	Sticky:               PtrBool(true),
}

func init() {
	sort.Strings(AllowableFlags)
	sort.Strings(AllowableMutable)
	sort.Strings(AllowableSchemas)
	sort.Strings(AllowableSpecVersions)

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
		EnforceCompatibility: OfferedCapability{
			Type: "boolean",
			Enum: []any{false},
		},
		Flags: OfferedCapability{
			Type: "array",
			Item: &OfferedItem{
				Type: "string",
			},
			Enum: String2AnySlice(AllowableFlags),
		},
		MaxMaxVersions: OfferedCapability{
			Type: "uinteger",
		},
		Mutable: OfferedCapability{
			Type: "string",
			Enum: String2AnySlice(AllowableMutable),
		},
		Pagination: OfferedCapability{
			Type: "boolean",
			Enum: []any{false},
		},
		Schemas: OfferedCapability{
			Type: "string",
			Enum: String2AnySlice(AllowableSchemas),
		},
		ShortSelf: OfferedCapability{
			Type: "boolean",
			Enum: []any{false},
		},
		SpecVersions: OfferedCapability{
			Type: "string",
			Enum: String2AnySlice(AllowableSpecVersions),
		},
		Sticky: OfferedCapability{
			Type: "boolean",
			Enum: []any{false, true},
		},
	}

	return offered
}

func ArrayToLower(arr []string) []string {
	for i, s := range arr {
		arr[i] = strings.ToLower(s)
	}
	return arr
}

func CleanArray(arr []string, full []string, text string) ([]string, error) {
	// Make a copy so we can tweak it
	arr = slices.Clone(arr)

	// Lowercase evrything and look for "*"
	for i, s := range arr {
		s = strings.ToLower(s)

		// Special case these
		if text == "schemas" && s == strings.ToLower(XREGSCHEMA) {
			// Allow just "xregistry-json", we'll add the spec version #
			s = s + "/" + SPECVERSION
		}
		// End-of-special

		arr[i] = s
		if s == "*" {
			if len(arr) != 1 {
				return nil, fmt.Errorf(`"*" must be the only value `+
					`specified for %q`, text)
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
			return nil, fmt.Errorf(`Unknown %q value: %q`, text, as)
		}
		fi--
	}
	if ai < 0 {
		return arr, nil
	}
	return nil, fmt.Errorf(`Unknown %q value: %q`, text, arr[ai])
}

func (c *Capabilities) Validate() error {
	var err error

	if c.Schemas == nil {
		c.Schemas = []string{XREGSCHEMA + "/" + SPECVERSION}
	}
	if c.SpecVersions == nil {
		c.SpecVersions = []string{SPECVERSION}
	}

	if c.EnforceCompatibility != false {
		return fmt.Errorf(`"enforcecapabilities" must be "false"`)
	}

	c.Flags, err = CleanArray(c.Flags, AllowableFlags, "flags")
	if err != nil {
		return err
	}

	if c.MaxMaxVersions < 0 {
		return fmt.Errorf(`"maxmaxversions" must be an unsigned integer`)
	}

	c.Mutable, err = CleanArray(c.Mutable, AllowableMutable, "mutable")
	if err != nil {
		return err
	}

	if c.Pagination != false {
		return fmt.Errorf(`"pagination" must be "false"`)
	}

	c.Schemas, err = CleanArray(c.Schemas, AllowableSchemas, "schemas")
	if err != nil {
		return err
	}

	c.SpecVersions, err = CleanArray(c.SpecVersions, AllowableSpecVersions,
		"specversions")
	if err != nil {
		return err
	}

	if !ArrayContainsAnyCase(c.Schemas, XREGSCHEMA+"/"+SPECVERSION) {
		return fmt.Errorf(`"schemas" must contain %q`, XREGSCHEMA+"/"+SPECVERSION)
	}

	if c.ShortSelf != false {
		return fmt.Errorf(`"shortself" must be "false"`)
	}

	if !ArrayContainsAnyCase(c.SpecVersions, SPECVERSION) {
		return fmt.Errorf(`"specversions" must contain %q`, SPECVERSION)
	}

	if c.Sticky == nil {
		c.Sticky = DefaultCapabilities.Sticky
	}

	return nil
}

func ParseCapabilitiesJSON(buf []byte) (*Capabilities, error) {
	log.VPrintf(4, "Enter: ParseCapabilitiesJSON")
	cap := Capabilities{}

	err := Unmarshal(buf, &cap)
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown field ") {
			err = fmt.Errorf("Unknown capability: %s", err.Error()[14:])
		}
		return nil, err
	}
	return &cap, nil
}

func (c *Capabilities) EnforceCompatibilityEnabled() bool {
	return c.EnforceCompatibility
}

func (c *Capabilities) FlagEnabled(str string) bool {
	return ArrayContainsAnyCase(c.Flags, str)
}

func (c *Capabilities) MaxMaxVersionsEnabled(ver int) bool {
	return ver >= 0 && (c.MaxMaxVersions == 0 || (ver > 0 && ver <= c.MaxMaxVersions))
}

func (c *Capabilities) MutableEnabled(str string) bool {
	return ArrayContainsAnyCase(c.Mutable, str)
}

func (c *Capabilities) PaginationEnabled() bool {
	return c.Pagination
}

func (c *Capabilities) SchemaEnabled(str string) bool {
	return ArrayContainsAnyCase(c.Schemas, str)
}

func (c *Capabilities) ShortSelfEnabled(str string) bool {
	return c.ShortSelf
}

func (c *Capabilities) SpecVersionEnabled(str string) bool {
	return ArrayContainsAnyCase(c.SpecVersions, str)
}

func (c *Capabilities) StickyEnabled() bool {
	return c.Sticky != nil && (*c.Sticky) == true
}
