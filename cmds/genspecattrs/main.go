// genspecattrs generates registry/ui/specattrs.js from the spec-defined
// attribute list in registry.OrderedSpecProps (sourced from common/shared_entity).
// Run via: make .sharedfiles  (triggered when common/shared_entity changes)
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

// Map StrTypes digit chars → JS level names.
// Digit values match the iota order in common/xid.go:
//
//	ENTITY_REGISTRY=0, ENTITY_GROUP=1, ENTITY_RESOURCE=2, ENTITY_META=3,
//	ENTITY_VERSION=4
var digitToLevel = map[byte]string{
	'0' + byte(common.ENTITY_REGISTRY): "registry",
	'0' + byte(common.ENTITY_GROUP):    "group",
	'0' + byte(common.ENTITY_RESOURCE): "resource",
	'0' + byte(common.ENTITY_META):     "meta",
	'0' + byte(common.ENTITY_VERSION):  "version",
}

var allLevels = []string{"registry", "group", "resource", "meta", "version"}

func main() {
	byLevel := map[string]map[string]bool{}
	monoByLevel := map[string]map[string]bool{}
	labelMap := map[string]string{} // attr name → UI label
	for _, lv := range allLevels {
		byLevel[lv] = map[string]bool{}
		monoByLevel[lv] = map[string]bool{}
	}

	for _, a := range registry.GetOrderedSpecAttrTypes() {
		if a.UILabel != "" {
			labelMap[a.Name] = a.UILabel
		}
		if a.Types == "" {
			for _, lv := range allLevels {
				byLevel[lv][a.Name] = true
				if a.UIMonospace {
					monoByLevel[lv][a.Name] = true
				}
			}
		} else {
			for i := 0; i < len(a.Types); i++ {
				if lv, ok := digitToLevel[a.Types[i]]; ok {
					byLevel[lv][a.Name] = true
					if a.UIMonospace {
						monoByLevel[lv][a.Name] = true
					}
				}
			}
		}
	}

	// Sort names per level for stable output
	sorted := map[string][]string{}
	monoSorted := map[string][]string{}
	for _, lv := range allLevels {
		names := make([]string, 0, len(byLevel[lv]))
		for name := range byLevel[lv] {
			names = append(names, name)
		}
		sort.Strings(names)
		sorted[lv] = names

		monoNames := make([]string, 0, len(monoByLevel[lv]))
		for name := range monoByLevel[lv] {
			monoNames = append(monoNames, name)
		}
		sort.Strings(monoNames)
		monoSorted[lv] = monoNames
	}

	// Sort label attr names for stable output
	labelNames := make([]string, 0, len(labelMap))
	for name := range labelMap {
		labelNames = append(labelNames, name)
	}
	sort.Strings(labelNames)

	out, err := os.Create("registry/ui/specattrs.js")
	if err != nil {
		fmt.Fprintf(os.Stderr, "genspecattrs: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	fmt.Fprintln(out, "// AUTO-GENERATED — do not edit directly.")
	fmt.Fprintln(out, "// Source: common/shared_entity  |  "+
		"Regenerate: make .sharedfiles")
	fmt.Fprintln(out, "// Generator: cmds/genspecattrs/main.go")
	fmt.Fprintln(out, "//")
	fmt.Fprintln(out, "// Spec-defined attributes per entity level.")
	fmt.Fprintln(out, "// Extensions: attrs NOT in this set, NOT "+
		"<singular>id, NOT collection keys.")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "var SPEC_ATTRS = {")
	for i, level := range allLevels {
		pairs := make([]string, len(sorted[level]))
		for j, a := range sorted[level] {
			pairs[j] = a + ":1"
		}
		comma := ","
		if i == len(allLevels)-1 {
			comma = ""
		}
		fmt.Fprintf(out, "  %-9s {%s}%s\n", level+":",
			strings.Join(pairs, ", "), comma)
	}
	fmt.Fprintln(out, "};")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "// String-typed spec attributes that should render in monospace in the UI.")
	fmt.Fprintln(out, "// These are technical identifiers/values, not human-readable prose.")
	fmt.Fprintln(out, "// Non-string spec attrs (boolean, integer, timestamp, url, …) are")
	fmt.Fprintln(out, "// already monospaced via model-type logic and are not listed here.")
	fmt.Fprintln(out, "var MONO_ATTRS = {")
	for i, level := range allLevels {
		pairs := make([]string, len(monoSorted[level]))
		for j, a := range monoSorted[level] {
			pairs[j] = a + ":1"
		}
		comma := ","
		if i == len(allLevels)-1 {
			comma = ""
		}
		fmt.Fprintf(out, "  %-9s {%s}%s\n", level+":",
			strings.Join(pairs, ", "), comma)
	}
	fmt.Fprintln(out, "};")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "// Display label overrides for spec-defined attributes.")
	fmt.Fprintln(out, "// labelFor() uses this only when the attribute is confirmed spec-defined")
	fmt.Fprintln(out, "// at the current entity level; extension attrs with the same name get")
	fmt.Fprintln(out, "// the raw attribute name as their label.")
	fmt.Fprintln(out, "var LABEL_ATTRS = {")
	for i, name := range labelNames {
		comma := ","
		if i == len(labelNames)-1 {
			comma = ""
		}
		fmt.Fprintf(out, "  %s: %q%s\n", name, labelMap[name], comma)
	}
	fmt.Fprintln(out, "};")

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "// Spec-defined attribute names in declaration order, per entity level.")
	fmt.Fprintln(out, "// Use for column and property ordering — spec attrs appear before extensions.")
	fmt.Fprintln(out, "// Structural '$'-prefixed entries are excluded (never appear as UI columns).")
	fmt.Fprintln(out, "var SPEC_ATTRS_ORDER = {")

	// Build ordered arrays per level, preserving GetOrderedSpecAttrTypes() declaration order.
	// Deduplicate and skip structural '$'-prefixed attrs.
	orderedByLevel := map[string][]string{}
	seenByLevel := map[string]map[string]bool{}
	for _, lv := range allLevels {
		orderedByLevel[lv] = []string{}
		seenByLevel[lv] = map[string]bool{}
	}
	for _, a := range registry.GetOrderedSpecAttrTypes() {
		if strings.HasPrefix(a.Name, "$") {
			continue // structural / internal — not UI column candidates
		}
		levels := allLevels
		if a.Types != "" {
			levels = []string{}
			for i := 0; i < len(a.Types); i++ {
				if lv, ok := digitToLevel[a.Types[i]]; ok {
					levels = append(levels, lv)
				}
			}
		}
		for _, lv := range levels {
			if !seenByLevel[lv][a.Name] {
				seenByLevel[lv][a.Name] = true
				orderedByLevel[lv] = append(orderedByLevel[lv], a.Name)
			}
		}
	}
	for i, level := range allLevels {
		quoted := make([]string, len(orderedByLevel[level]))
		for j, name := range orderedByLevel[level] {
			quoted[j] = fmt.Sprintf("%q", name)
		}
		comma := ","
		if i == len(allLevels)-1 {
			comma = ""
		}
		fmt.Fprintf(out, "  %-9s [%s]%s\n", level+":", strings.Join(quoted, ", "), comma)
	}
	fmt.Fprintln(out, "};")

	// fmt.Fprintf(os.Stderr, "genspecattrs: wrote registry/ui/specattrs.js\n")
}
