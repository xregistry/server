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
	Mutable        []string `json:"mutable"`        // must not have omitempty
	Pagination     bool     `json:"pagination"`     // must not have omitempty
	ShortSelf      bool     `json:"shortself"`      // must not have omitempty
	SpecVersions   []string `json:"specversions"`   // must not have omitempty
	StickyVersions *bool    `json:"stickyversions"` // must not have omitempty
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
	Mutable        OfferedCapability `json:"mutable,omitempty"`
	Pagination     OfferedCapability `json:"pagination,omitempty"`
	ShortSelf      OfferedCapability `json:"shortself,omitempty"`
	SpecVersions   OfferedCapability `json:"specversions,omitempty"`
	StickyVersions OfferedCapability `json:"stickyversions,omitempty"`
}

var AllowableAPIs = ArrayToLower([]string{
	"/capabilities", "/capabilitiesoffered", "/export",
	"/model", "/modelsource"})

var AllowableFlags = ArrayToLower([]string{
	"binary", "collections", "doc", "epoch", "filter",
	"ignoredefaultversionid", "ignoredefaultversionsticky",
	"ignoreepoch", "ignorereadonly", "inline",
	"setdefaultversionid", "sort", "specversion"})

var AllowableMutable = ArrayToLower([]string{
	"capabilities", "entities", "model"})

var AllowableSpecVersions = ArrayToLower([]string{"1.0-rc2", SPECVERSION})

var SupportedFlags = ArrayToLower([]string{
	"binary", "collections", "doc", "epoch", "filter",
	"ignoredefaultversionid", "ignoredefaultversionsticky",
	"ignoreepoch", "ignorereadonly", "inline",
	"setdefaultversionid", "sort", "specversion"})

var DefaultCapabilities = &Capabilities{
	APIs:           AllowableAPIs,
	Flags:          SupportedFlags,
	Mutable:        AllowableMutable,
	Pagination:     false,
	ShortSelf:      false,
	SpecVersions:   AllowableSpecVersions,
	StickyVersions: PtrBool(true),
}

func init() {
	sort.Strings(AllowableAPIs)
	sort.Strings(AllowableFlags)
	sort.Strings(AllowableMutable)
	sort.Strings(AllowableSpecVersions)

	sort.Strings(SupportedFlags)

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
			Type: "string",
			Enum: String2AnySlice(AllowableSpecVersions),
		},
		StickyVersions: OfferedCapability{
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
	arr = slices.Compact(arr) // remove dups
	return arr
}

func CleanArray(arr []string, full []string, text string) ([]string, error) {
	// Make a copy so we can tweak it
	arr = slices.Clone(arr)

	// Lowercase evrything and look for "*"
	for i, s := range arr {
		s = strings.ToLower(s)

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

	if c.SpecVersions == nil {
		c.SpecVersions = []string{SPECVERSION}
	}

	c.APIs, err = CleanArray(c.APIs, AllowableAPIs, "apis")
	if err != nil {
		return err
	}

	c.Flags, err = CleanArray(c.Flags, AllowableFlags, "flags")
	if err != nil {
		return err
	}

	c.Mutable, err = CleanArray(c.Mutable, AllowableMutable, "mutable")
	if err != nil {
		return err
	}

	if c.Pagination != false {
		return fmt.Errorf(`"pagination" must be "false"`)
	}

	c.SpecVersions, err = CleanArray(c.SpecVersions, AllowableSpecVersions,
		"specversions")
	if err != nil {
		return err
	}

	if c.ShortSelf != false {
		return fmt.Errorf(`"shortself" must be "false"`)
	}

	if !ArrayContainsAnyCase(c.SpecVersions, SPECVERSION) {
		return fmt.Errorf(`"specversions" must contain %q`, SPECVERSION)
	}

	if c.StickyVersions == nil {
		c.StickyVersions = DefaultCapabilities.StickyVersions
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

func (c *Capabilities) APIEnabled(str string) bool {
	return ArrayContainsAnyCase(c.APIs, str)
}

func (c *Capabilities) FlagEnabled(str string) bool {
	return ArrayContainsAnyCase(c.Flags, str)
}

func (c *Capabilities) MutableEnabled(str string) bool {
	return ArrayContainsAnyCase(c.Mutable, str)
}

func (c *Capabilities) PaginationEnabled() bool {
	return c.Pagination
}

func (c *Capabilities) ShortSelfEnabled(str string) bool {
	return c.ShortSelf
}

func (c *Capabilities) SpecVersionEnabled(str string) bool {
	return ArrayContainsAnyCase(c.SpecVersions, str)
}

func (c *Capabilities) StickyVersionsEnabled() bool {
	return c.StickyVersions != nil && (*c.StickyVersions) == true
}
