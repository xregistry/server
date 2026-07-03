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
	for _, lv := range allLevels {
		byLevel[lv] = map[string]bool{}
	}

	for _, a := range registry.GetOrderedSpecAttrTypes() {
		if a.Types == "" {
			// Applies to all entity levels
			for _, lv := range allLevels {
				byLevel[lv][a.Name] = true
			}
		} else {
			for i := 0; i < len(a.Types); i++ {
				if lv, ok := digitToLevel[a.Types[i]]; ok {
					byLevel[lv][a.Name] = true
				}
			}
		}
	}

	// Sort names per level for stable output
	sorted := map[string][]string{}
	for _, lv := range allLevels {
		names := make([]string, 0, len(byLevel[lv]))
		for name := range byLevel[lv] {
			names = append(names, name)
		}
		sort.Strings(names)
		sorted[lv] = names
	}

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

	// fmt.Fprintf(os.Stderr, "genspecattrs: wrote registry/ui/specattrs.js\n")
}
