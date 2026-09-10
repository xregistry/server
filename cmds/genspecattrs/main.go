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
	// The UI's own build-time commit sha — NOT the sha of whatever
	// registry server the SPA happens to be pointed at (that's runtime,
	// per-server data unrelated to this file). Read from the GIT_COMMIT
	// env var the Makefile already exports for BUILDFLAGS/STATIC (falls
	// back to common.GitCommit, e.g. when this generator is invoked with
	// -ldflags directly instead of via `make`). Surfaced by the Config
	// page's "About" section (see renderConfig() in app.js). Since this
	// only regenerates when its Makefile prerequisites change (see
	// registry/ui/specattrs.js's rule), it reflects the commit as of the
	// last time this file was regenerated, not necessarily HEAD.
	uiCommit := os.Getenv("GIT_COMMIT")
	if uiCommit == "" {
		uiCommit = common.GitCommit
	}
	fmt.Fprintf(out, "var XREG_UI_COMMIT = %q;\n", uiCommit)
	fmt.Fprintln(out, "")
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

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "// Full canonical attribute order per entity level, INCLUDING the")
	fmt.Fprintln(out, "// structural '$space' (blank-line separator) and '$extensions'")
	fmt.Fprintln(out, "// (alphabetized-extension insertion point) markers from")
	fmt.Fprintln(out, "// registry.OrderedSpecProps, in declaration order. Unlike")
	fmt.Fprintln(out, "// SPEC_ATTRS_ORDER above (which drops '$'-prefixed entries — it's")
	fmt.Fprintln(out, "// only used for UI column ordering), this preserves them so a")
	fmt.Fprintln(out, "// canonical-order JSON pretty-printer can reproduce the spec's")
	fmt.Fprintln(out, "// pseudo-JSON layout (see core/spec.md \"Design: JSON Serialization\").")
	fmt.Fprintln(out, "// '$RESOURCE*'/'$COLLECTIONS' placeholder tokens are kept verbatim —")
	fmt.Fprintln(out, "// a consumer without the real model can't resolve them to real")
	fmt.Fprintln(out, "// attribute names, so it should just skip over them as no-ops.")
	fmt.Fprintln(out, "// Consecutive '$space' entries (which can end up adjacent after")
	fmt.Fprintln(out, "// per-level filtering removes everything between two of them) are")
	fmt.Fprintln(out, "// already collapsed to one here, and no leading/trailing '$space'")
	fmt.Fprintln(out, "// survives — so a consumer can treat every remaining '$space' as")
	fmt.Fprintln(out, "// exactly one blank line to emit.")
	fmt.Fprintln(out, "var SPEC_ATTRS_CANONICAL_ORDER = {")
	canonicalByLevel := map[string][]string{}
	for _, lv := range allLevels {
		seen := map[string]bool{}
		tokens := []string{}
		for _, a := range registry.GetOrderedSpecAttrTypes() {
			levels := allLevels
			if a.Types != "" {
				levels = []string{}
				for i := 0; i < len(a.Types); i++ {
					if l, ok := digitToLevel[a.Types[i]]; ok {
						levels = append(levels, l)
					}
				}
			}
			applies := false
			for _, l := range levels {
				if l == lv {
					applies = true
					break
				}
				// A Resource's own HTTP GET response isn't just its own
				// "resource"-typed attrs: the server mirrors its default
				// Version's own attrs (isdefault/createdat/modifiedat/
				// ancestorid/contenttype/the "$RESOURCE*" content
				// family/etc.) directly onto the Resource's own row
				// (see registry/resource.go's
				// SaveDefaultVersionCascade()/IsDefaultVerCopy
				// mechanism, and the identical comment in
				// common/shared_entity's buildCanonicalLevelOrders()),
				// so the "resource" canonical order must ALSO include
				// "version"-typed attrs, in the same master declaration
				// order — otherwise those mirrored attrs aren't
				// recognized as spec attrs and incorrectly fall through
				// to being alphabetized as plain extensions.
				if lv == "resource" && l == "version" {
					applies = true
					break
				}
			}
			if !applies {
				continue
			}
			// $space/$extensions are structural — always kept, never
			// deduped against a "seen name" (multiple $space markers are
			// expected/legit; collapsing of consecutive ones happens
			// below, after this per-attribute pass).
			if a.Name == "$space" || a.Name == "$extensions" {
				tokens = append(tokens, a.Name)
				continue
			}
			if seen[a.Name] {
				continue
			}
			seen[a.Name] = true
			// The Registry's generic "id" attribute always has the
			// fixed, model-independent wire name "registryid" — safe to
			// hardcode here (mirrors the same substitution in
			// common/shared_entity's canonicalLevelOrderFor()).
			// Group/Resource/Meta's own "<singular>id" wire name DOES
			// depend on the model and is intentionally left
			// unsubstituted (falls through to being treated as an
			// extension attribute by the canonical printer).
			if a.Name == "id" && lv == "registry" {
				tokens = append(tokens, "registryid")
				continue
			}
			tokens = append(tokens, a.Name)
		}

		// Collapse consecutive "$space" runs into a single one, and drop
		// any leading/trailing "$space".
		collapsed := make([]string, 0, len(tokens))
		for _, t := range tokens {
			if t == "$space" {
				if len(collapsed) == 0 || collapsed[len(collapsed)-1] == "$space" {
					continue
				}
			}
			collapsed = append(collapsed, t)
		}
		for len(collapsed) > 0 && collapsed[len(collapsed)-1] == "$space" {
			collapsed = collapsed[:len(collapsed)-1]
		}
		canonicalByLevel[lv] = collapsed
	}
	for i, level := range allLevels {
		quoted := make([]string, len(canonicalByLevel[level]))
		for j, name := range canonicalByLevel[level] {
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
