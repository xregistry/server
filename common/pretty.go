package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// PrettyPrintJSON takes a JSON []byte, prefix, and indent, and returns
// pretty-printed JSON []byte  with the original order of attributes and
// arrays preserved.
// Note, it will NOT do error/syntax checking - maybe we should at some point
func PrettyPrintJSON(data []byte, prefix, indent string) ([]byte, error) {
	// Handle empty object and empty array cases
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 2 {
		if trimmed[0] == '{' && trimmed[1] == '}' {
			return []byte("{}"), nil
		}
		if trimmed[0] == '[' && trimmed[1] == ']' {
			return []byte("[]"), nil
		}
	}

	obj, err := ParseJSONToObject(data)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(obj, prefix, indent)
}

func ParseJSONToObject(data []byte) (any, error) {
	// Parse JSON while preserving key order

	var parseJSON func(decoder *json.Decoder) (any, error)
	parseJSON = func(decoder *json.Decoder) (any, error) {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case json.Delim:
			if t == '{' {
				ordered := &OrderedMap{Values: make(map[string]any)}
				for decoder.More() {
					key, err := decoder.Token()
					if err != nil {
						return nil, err
					}
					keyStr, ok := key.(string)
					if !ok {
						return nil, fmt.Errorf("expected string key")
					}
					ordered.Keys = append(ordered.Keys, keyStr)
					value, err := parseJSON(decoder)
					if err != nil {
						return nil, err
					}
					ordered.Values[keyStr] = value
				}
				// Consume closing '}'
				if _, err := decoder.Token(); err != nil {
					return nil, err
				}
				return ordered, nil
			} else if t == '[' {
				// In the case of `[]` make sure we return zero-size not null
				arr := []any{}
				for decoder.More() {
					value, err := parseJSON(decoder)
					if err != nil {
						return nil, err
					}
					arr = append(arr, value)
				}
				// Consume closing ']'
				if _, err := decoder.Token(); err != nil {
					return nil, err
				}
				return arr, nil
			}
			return nil, fmt.Errorf("unexpected delimiter: %v", t)
		default:
			return t, nil // Scalars (string, number, bool, null)
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return parseJSON(decoder)
}

// OrderedMap holds key-value pairs with order preservation
type OrderedMap struct {
	Keys   []string
	Values map[string]any
}

// UnmarshalJSON implements custom JSON unmarshaling to preserve key order
func (o *OrderedMap) UnmarshalJSON(data []byte) error {
	o.Values = make(map[string]any)
	o.Keys = nil // Reset Keys to ensure no leftover keys

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if t, err := decoder.Token(); err != nil {
		return err
	} else if t != json.Delim('{') {
		return fmt.Errorf("expected object")
	}

	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return err
		}
		keyStr, ok := key.(string)
		if !ok {
			return fmt.Errorf("expected string key")
		}
		o.Keys = append(o.Keys, keyStr)
		var value any
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		o.Values[keyStr] = value
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling to preserve key order
func (o *OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{")
	for i, key := range o.Keys {
		if i > 0 {
			buf.WriteString(",")
		}
		keyBytes, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteString(":")
		valueBytes, err := json.Marshal(o.Values[key])
		if err != nil {
			return nil, err
		}
		buf.Write(valueBytes)
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}

// Has reports whether key is present in o (order-agnostic lookup).
func (o *OrderedMap) Has(key string) bool {
	if o == nil || o.Values == nil {
		return false
	}
	_, ok := o.Values[key]
	return ok
}

// Get returns o's value for key (nil if absent) — a convenience wrapper
// around o.Values[key] for callers that don't need the "present" bool.
func (o *OrderedMap) Get(key string) any {
	if o == nil || o.Values == nil {
		return nil
	}
	return o.Values[key]
}

// Set assigns value to key, preserving its existing position in o.Keys
// if key is already present, or appending it to the end if it's new.
// Used by callers (e.g. cmds/xr/download.go's noDiffObj()) that need to
// overwrite/rewrite specific attribute values on an already canonically
// reordered tree without disturbing the order of what's already there.
func (o *OrderedMap) Set(key string, value any) {
	if o.Values == nil {
		o.Values = make(map[string]any)
	}
	if _, ok := o.Values[key]; !ok {
		o.Keys = append(o.Keys, key)
	}
	o.Values[key] = value
}

// Delete removes key from o, preserving the relative order of o's
// remaining keys (unlike deleting from a plain map, which has no order
// to preserve). A no-op if key isn't present. Used by callers (e.g.
// cmds/xr/get.go's minimize()) that need to strip attributes from an
// already canonically-reordered tree (see CanonicalReorderTree) without
// disturbing the order/blank-line spacing of what's left.
func (o *OrderedMap) Delete(key string) {
	if o == nil {
		return
	}
	for i, k := range o.Keys {
		if k == key {
			o.Keys = append(o.Keys[:i], o.Keys[i+1:]...)
			break
		}
	}
	if o.Values != nil {
		delete(o.Values, key)
	}
}

// RealKeyCount returns the number of keys in o that represent actual
// data attributes, excluding the reserved blank-line sentinel key (see
// canonicalBlankKey) inserted by the reorder pass. Plain len(o.Keys) is
// NOT a reliable "is this entity empty" test on an already-reordered
// tree: Delete() only ever removes named attributes, never the
// sentinel, so a lingering blank-line separator (from a section that
// used to have content, before deletion) would otherwise make
// len(o.Keys) > 0 even though zero real attributes remain. Callers that
// need to decide "does this (possibly minimized) entity still have any
// real content" (e.g. cmds/xr/download.go's decision to skip writing a
// now-truly-empty file) should use this instead of len(o.Keys).
func (o *OrderedMap) RealKeyCount() int {
	if o == nil {
		return 0
	}
	n := 0
	for _, k := range o.Keys {
		if k != canonicalBlankKey {
			n++
		}
	}
	return n
}

// ---- Canonical-order pretty-printer engine --------------------------------
//
// This is the generic, xRegistry-aware half of the "json-pretty-printer"
// backlog item (see TODO.md): unlike PrettyPrintJSON() above (which just
// re-indents the input's OWN attribute order), this reorders each
// xRegistry entity it finds into the spec's canonical attribute order
// (core/spec.md "Design: JSON Serialization"), inserting blank-line
// separators and alphabetizing extension attributes, using a caller-
// supplied per-entity-level ordering table as the source of truth.
//
// The ordering table itself (derived from registry.OrderedSpecProps) has
// to live in the registry/xrlib packages (only they have access to
// OrderedSpecProps) — see CanonicalPrettyPrintJSON() in
// common/shared_entity, which is the actual public entry point callers
// should use. This file only owns the generic (level-order-agnostic)
// walk + reorder + stringify engine.
//
// The engine is deliberately split into two passes:
//  1. "reorder" (CanonicalReorderTree/reorderValue/reorderEntity/...)
//     rewrites each entity's *OrderedMap.Keys into canonical order IN
//     PLACE, recursing into nested entities/collections, and inserts a
//     reserved sentinel key (canonicalBlankKey) wherever a blank-line
//     separator belongs. The result is still a live *OrderedMap tree,
//     not a string.
//  2. "stringify" (stringifyValue/stringifyOrderedMap/...) walks an
//     already-ordered tree and emits indented JSON bytes, treating the
//     sentinel key as "emit a blank line" — re-collapsing consecutive
//     sentinels and trimming leading/trailing ones defensively, in case
//     something deleted keys from the tree between the two passes.
//
// This split exists so that a caller (see cmds/xr/get.go's minimize())
// can call CanonicalReorderTree(), delete whatever keys it wants from
// the resulting tree (an order-preserving *OrderedMap.Delete() call),
// and then stringify what's left — getting a minimized-but-still-
// canonically-ordered result with correctly collapsed spacing, instead
// of trying to reorder JSON that's already had its classifying
// information (xid, ids, model-driven attr names) stripped out first.
// CanonicalPrettyPrintJSON() (the normal, non-minimized entry point) is
// just reorder-then-stringify with nothing deleted in between.

// CanonicalLevelOrder returns, for a given xRegistry entity type
// (common.ENTITY_REGISTRY/ENTITY_GROUP/ENTITY_RESOURCE/ENTITY_META/
// ENTITY_VERSION), the canonical ordered list of attribute-name tokens for
// that level — spec attrs in declaration order, plus the structural
// "$space" (blank-line separator) and "$extensions" (alphabetized-
// extension insertion point) markers already collapsed/trimmed by the
// caller (see registry.GetOrderedSpecAttrTypes()/genspecattrs's identical
// logic). Any other "$"-prefixed token (e.g. "$RESOURCEurl",
// "$COLLECTIONS") is a placeholder this engine can't resolve without the
// real model — it's simply skipped as a no-op; whatever real data would
// have filled that slot ends up alphabetized as a plain extension
// instead (see decisions in plan.md/TODO.md — this is a deliberate,
// documented simplification, not a bug).
type CanonicalLevelOrder func(entityType int) []string

// CanonicalReorderJSON parses data (a single xRegistry entity, or a full
// nested Registry doc) and returns it reformatted with every nested
// entity's own attributes reordered into canonical spec order. Entities
// are found structurally: any JSON object with its own string "xid" field
// is classified (via ParseXid) and reordered; every other nested
// object/array is otherwise left with its original key order but still
// recursed into (so collections of entities — e.g. a map of Group
// instances, a Resource's "versions" map — get each child reordered even
// though the collection wrapper itself isn't touched).
// CanonicalReorderTree parses data (a single xRegistry entity, or a full
// nested Registry doc) and returns it as a live *OrderedMap (or, for a
// bare array/scalar input, a []any/scalar) tree with every
// nested entity's own attributes ALREADY reordered into canonical spec
// order — including a reserved sentinel key (see canonicalBlankKey)
// wherever a blank-line separator belongs. Entities are found
// structurally: any JSON object with its own string "xid" field is
// classified (via ParseXid) and reordered; every other nested
// object/array otherwise keeps its own original key order but is still
// recursed into (so collections of entities — e.g. a map of Group
// instances, a Resource's "versions" map — get each child reordered
// even though the collection wrapper itself isn't touched).
//
// The result is NOT yet stringified — callers that just want formatted
// bytes should use CanonicalReorderJSON(); this entry point exists for
// callers (see cmds/xr/get.go's minimize()) that need to delete keys
// from the tree (via *OrderedMap.Delete(), which is order-preserving)
// BEFORE stringifying, while everything is still correctly classified
// and ordered.
func CanonicalReorderTree(data []byte, order CanonicalLevelOrder) (any, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}
	obj, err := ParseJSONToObject(data)
	if err != nil {
		return nil, err
	}
	return reorderValue(obj, order), nil
}

// CanonicalReorderJSON is CanonicalReorderTree() followed immediately by
// stringifying the result — the normal (non-minimized) entry point.
func CanonicalReorderJSON(data []byte, order CanonicalLevelOrder) ([]byte, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return trimmed, nil
	}
	tree, err := CanonicalReorderTree(data, order)
	if err != nil {
		return nil, err
	}
	return StringifyCanonicalTree(tree)
}

// StringifyCanonicalTree renders an already-reordered tree (as returned
// by CanonicalReorderTree()/CanonicalPrettyReorderTree()) as indented
// JSON bytes. It's the public form of the internal stringifyValue() —
// exposed separately from CanonicalReorderJSON() so a caller can modify
// the tree (e.g. delete keys via *OrderedMap.Delete()) between
// reordering and stringifying it — see cmds/xr/get.go's minimize(),
// which does exactly that for the "xr get -m" flag.
func StringifyCanonicalTree(tree any) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := stringifyValue(buf, tree, ""); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// canonicalBlankKey is a reserved sentinel inserted into an *OrderedMap's
// Keys (mapped to a nil Value) wherever a canonical "$space" blank-line
// separator belongs. It travels through the tree like any other key —
// in particular, a caller deleting REAL keys (e.g. minimize()) never
// needs to touch it — and is only ever interpreted specially by the
// final stringify pass (see stringifyOrderedMap/collapseBlankKeys),
// which re-collapses consecutive/leading/trailing sentinels so deleting
// every real key in a section still collapses cleanly instead of
// leaving a stray or duplicated blank line.
const canonicalBlankKey = "\x00__canonical_blank__\x00"

// canonicalEntityXid returns the parsed *Xid of om if it carries its own
// valid string "xid" attribute and is classifiable as one of the 5
// known entity levels, else ok=false (it's just a plain object/
// collection, not something this engine knows how to reorder).
func canonicalEntityXid(om *OrderedMap) (xid *Xid, ok bool) {
	xidVal, has := om.Values["xid"]
	if !has {
		return nil, false
	}
	xidStr, isStr := xidVal.(string)
	if !isStr || xidStr == "" {
		return nil, false
	}
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, false
	}
	switch xid.Type {
	case ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_RESOURCE, ENTITY_META, ENTITY_VERSION:
		return xid, true
	default:
		return nil, false
	}
}

// reorderValue recursively reorders val's own tree in place (mutating
// any *OrderedMap it finds) and returns it — scalars pass through
// unchanged.
func reorderValue(val any, order CanonicalLevelOrder) any {
	switch v := val.(type) {
	case *OrderedMap:
		if xid, ok := canonicalEntityXid(v); ok {
			return reorderEntity(v, order(xid.Type), xid, order)
		}
		return reorderPlainMap(v, order)
	case []any:
		return reorderArray(v, order)
	default:
		return val
	}
}

// reorderPlainMap leaves om's OWN key order untouched (this engine has
// no canonical ordering rule for a non-entity object — e.g. "labels",
// "capabilities", "model", or a collection map keyed by entity ID) but
// still recurses into each value, so nested entities (e.g. each Group
// instance inside a Groups collection map) still get their own
// attributes reordered.
func reorderPlainMap(om *OrderedMap, order CanonicalLevelOrder) *OrderedMap {
	for _, k := range om.Keys {
		om.Values[k] = reorderValue(om.Values[k], order)
	}
	return om
}

func reorderArray(arr []any, order CanonicalLevelOrder) []any {
	for i, item := range arr {
		arr[i] = reorderValue(item, order)
	}
	return arr
}

// reorderEntity rewrites om's own Keys into canonical order: spec attrs
// from tokens (in order, skipped if absent from om), a "$space" token
// becomes a canonicalBlankKey sentinel, an "$extensions" token expands
// to every one of om's OWN keys not otherwise consumed by a spec-attr
// token or by one of the heuristics below, alphabetized. Each real key's
// own value is recursively reordered too.
//
// Three token kinds need special handling because their real wire-format
// key name depends on the actual (usually unavailable) model:
//   - the generic "id" token (Group/Resource/Meta/Version levels only —
//     Registry's is already hardcoded to the literal "registryid" token,
//     see canonicalLevelOrderFor): resolved by VALUE, not by guessing a
//     name — the entity's own xid already tells us the expected ID
//     string (e.g. the ResourceID for a Resource/Meta/Version), so we
//     look for whichever of om's own keys holds that exact value and
//     ends in "id". This also yields the model's actual singular name
//     (trim the trailing "id"), reused below for "$RESOURCE*".
//   - "$RESOURCEurl"/"$RESOURCEproxyurl"/"$RESOURCE"/"$RESOURCEbase64"
//     (Version level only): resolved directly from the singular name
//     found above (e.g. "messageurl"/"messageproxyurl"/"message"/
//     "messagebase64") — no guessing needed once the singular is known.
//   - "$COLLECTIONS" (Registry/Group/Resource levels): a Resource's own
//     collection is always the fixed, model-independent "versions"/
//     "versionscount"/"versionsurl" triple, so it's hardcoded. Registry's
//     (group-type) and Group's (resource-type) collections have no such
//     fixed name and there can be any number of them, so those ARE
//     detected heuristically: any unconsumed "<name>count" key with a
//     matching unconsumed "<name>url" sibling is treated as one
//     collection (its own inlined map, "<name>", included too if
//     present).
//
// If the "id"-value match can't be found (e.g. minimal/foreign JSON that
// omits the entity's own id attribute), the "id" and "$RESOURCE*" tokens
// are simply skipped and whatever real keys exist fall through to being
// treated as ordinary (alphabetized) extensions instead — same for any
// "$COLLECTIONS" heuristic that finds no count/url pairs.
func reorderEntity(om *OrderedMap, tokens []string, xid *Xid, order CanonicalLevelOrder) *OrderedMap {
	consumed := map[string]bool{}
	for _, t := range tokens {
		if t == "$space" || t == "$extensions" || strings.HasPrefix(t, "$") {
			continue
		}
		consumed[t] = true
	}

	resolvedIDKey, singular := resolveIDKeyByValue(om, xid, consumed)
	if resolvedIDKey != "" {
		consumed[resolvedIDKey] = true
	}

	hasCollectionsToken, hasResourceToken := false, false
	for _, t := range tokens {
		if t == "$COLLECTIONS" {
			hasCollectionsToken = true
		}
		if t == "$RESOURCEurl" {
			hasResourceToken = true
		}
	}

	var collectionsKeys []string
	if hasCollectionsToken {
		if xid.Type == ENTITY_RESOURCE {
			// Fixed, model-independent — every Resource has exactly one
			// collection, always literally named "versions".
			for _, k := range []string{"versionsurl", "versionscount", "versions"} {
				if _, present := om.Values[k]; present {
					collectionsKeys = append(collectionsKeys, k)
					consumed[k] = true
				}
			}
		} else {
			collectionsKeys = detectHeuristicCollections(om, consumed)
			for _, k := range collectionsKeys {
				consumed[k] = true
			}
		}
	}

	var resourceFamilyKeys []string
	if hasResourceToken && singular != "" {
		for _, suffix := range []string{"url", "proxyurl", "", "base64"} {
			k := singular + suffix
			if _, present := om.Values[k]; present {
				resourceFamilyKeys = append(resourceFamilyKeys, k)
				consumed[k] = true
			}
		}
	}

	extNames := make([]string, 0)
	for _, k := range om.Keys {
		if !consumed[k] {
			extNames = append(extNames, k)
		}
	}
	sort.Strings(extNames)

	// Build the final canonical Keys order, using canonicalBlankKey as
	// the blank-line sentinel — collapsing consecutive sentinels and
	// dropping any leading/trailing one (tokens is already pre-collapsed
	// by the caller, but $extensions can still expand to zero keys, or a
	// spec attr can be legitimately absent from om, so re-collapse here
	// against the REAL resulting sequence).
	newKeys := make([]string, 0, len(tokens)+len(extNames))
	appendKey := func(k string) { newKeys = append(newKeys, k) }
	appendBlank := func() {
		if len(newKeys) == 0 || newKeys[len(newKeys)-1] == canonicalBlankKey {
			return
		}
		newKeys = append(newKeys, canonicalBlankKey)
	}
	extensionsEmitted := false
	for _, t := range tokens {
		switch t {
		case "$space":
			appendBlank()
			continue
		case "$extensions":
			// Only expand at the first occurrence — a level's token
			// list should never have more than one "$extensions" slot,
			// but guard against emitting extensions twice if it did.
			if !extensionsEmitted {
				extensionsEmitted = true
				for _, name := range extNames {
					appendKey(name)
				}
			}
			continue
		case "$COLLECTIONS":
			for _, k := range collectionsKeys {
				appendKey(k)
			}
			continue
		case "$RESOURCEurl":
			for _, k := range resourceFamilyKeys {
				appendKey(k)
			}
			continue
		case "$RESOURCEproxyurl", "$RESOURCE", "$RESOURCEbase64":
			continue // already emitted via the "$RESOURCEurl" slot above
		case "id":
			if resolvedIDKey != "" {
				appendKey(resolvedIDKey)
			}
			continue
		}
		if strings.HasPrefix(t, "$") {
			continue // unresolvable placeholder — no-op (see doc comment)
		}
		if _, present := om.Values[t]; present {
			appendKey(t)
		}
	}
	for len(newKeys) > 0 && newKeys[len(newKeys)-1] == canonicalBlankKey {
		newKeys = newKeys[:len(newKeys)-1]
	}

	// Recursively reorder each real key's own value in place.
	for _, k := range newKeys {
		if k == canonicalBlankKey {
			continue
		}
		om.Values[k] = reorderValue(om.Values[k], order)
	}
	if om.Values == nil {
		om.Values = map[string]any{}
	}
	om.Values[canonicalBlankKey] = nil
	om.Keys = newKeys
	return om
}

// writeScalar writes val as JSON. Most callers reach here with an
// actual JSON scalar (string/number/bool/nil), which is written as-is.
// But val may instead be some other Go value never converted into the
// OrderedMap/[]any tree shape (e.g. a *Capabilities struct plugged in
// directly via OrderedMap.Set(), as cmds/xr/download.go's "/export"
// handling does) - json.Marshal-ing that directly would always produce
// a single compacted line, regardless of the surrounding indent depth.
// To keep pretty-printing generic (so callers don't each have to
// remember to pre-convert every non-tree value they insert), detect
// that case - the marshaled bytes start with '{' or '[' - and re-parse
// it into the same OrderedMap/[]any tree shape (preserving its own
// existing order - see ParseJSONToObject) and recurse through
// stringifyValue so it gets properly indented at the current depth.
func writeScalar(buf *bytes.Buffer, val any, indent string) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		if obj, perr := ParseJSONToObject(trimmed); perr == nil {
			return stringifyValue(buf, obj, indent)
		}
		// Fall through to writing the compact form rather than failing
		// the whole document over a value we can't reparse.
	}
	buf.Write(b)
	return nil
}

// stringifyValue writes val (an already-reordered tree — see
// CanonicalReorderTree/reorderValue) as indented JSON bytes. Unlike
// reorderValue, this never mutates or reorders anything; it just walks
// whatever order/structure the tree is already in.
func stringifyValue(buf *bytes.Buffer, val any, indent string) error {
	switch v := val.(type) {
	case *OrderedMap:
		return stringifyOrderedMap(buf, v, indent)
	case []any:
		return stringifyArray(buf, v, indent)
	default:
		return writeScalar(buf, val, indent)
	}
}

func stringifyArray(buf *bytes.Buffer, arr []any, indent string) error {
	if len(arr) == 0 {
		buf.WriteString("[]")
		return nil
	}
	childIndent := indent + "  "
	buf.WriteString("[\n")
	for i, item := range arr {
		buf.WriteString(childIndent)
		if err := stringifyValue(buf, item, childIndent); err != nil {
			return err
		}
		if i < len(arr)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString(indent + "]")
	return nil
}

// stringifyOrderedMap writes om's Keys/Values as indented JSON, treating
// canonicalBlankKey as "emit a blank line". collapseBlankKeys() is
// applied defensively right before emitting — om.Keys is normally
// already correctly collapsed/trimmed by reorderEntity(), but a caller
// (e.g. minimize()) may have deleted real keys from om AFTER reordering
// and BEFORE stringifying, which can leave now-redundant
// consecutive/leading/trailing blank sentinels behind; this keeps the
// final blank-line spacing correct regardless.
func stringifyOrderedMap(buf *bytes.Buffer, om *OrderedMap, indent string) error {
	keys := collapseBlankKeys(om.Keys)
	if len(keys) == 0 {
		buf.WriteString("{}")
		return nil
	}
	childIndent := indent + "  "
	buf.WriteString("{\n")
	// lastRealIdx tracks the index (within keys) of the last non-blank
	// entry, so we know when NOT to print a trailing comma.
	lastRealIdx := -1
	for i := len(keys) - 1; i >= 0; i-- {
		if keys[i] != canonicalBlankKey {
			lastRealIdx = i
			break
		}
	}
	for i, k := range keys {
		if k == canonicalBlankKey {
			buf.WriteString("\n")
			continue
		}
		buf.WriteString(childIndent)
		if err := writeScalar(buf, k, childIndent); err != nil {
			return err
		}
		buf.WriteString(": ")
		if err := stringifyValue(buf, om.Values[k], childIndent); err != nil {
			return err
		}
		if i != lastRealIdx {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString(indent + "}")
	return nil
}

// collapseBlankKeys returns keys with consecutive canonicalBlankKey
// sentinels collapsed to one, and any leading/trailing sentinel dropped
// entirely — same collapse rule reorderEntity() applies when first
// building the order, re-applied here in case keys were deleted after
// reordering (see stringifyOrderedMap's doc comment).
func collapseBlankKeys(keys []string) []string {
	collapsed := make([]string, 0, len(keys))
	for _, k := range keys {
		if k == canonicalBlankKey {
			if len(collapsed) == 0 || collapsed[len(collapsed)-1] == canonicalBlankKey {
				continue
			}
		}
		collapsed = append(collapsed, k)
	}
	for len(collapsed) > 0 && collapsed[len(collapsed)-1] == canonicalBlankKey {
		collapsed = collapsed[:len(collapsed)-1]
	}
	return collapsed
}

// resolveIDKeyByValue finds the real wire-format key name for om's
// generic "id" spec attribute (e.g. "messageid" for a Resource whose
// singular is "message") by matching VALUE rather than guessing a name:
// xid already tells us the exact ID string this entity's "id" attribute
// must hold (GroupID for a Group, ResourceID for a Resource/Meta/
// Version), so we look for whichever of om's own not-yet-consumed keys
// holds that value and ends in "id". Returns ("", "") for the Registry
// level (its "id" token is already hardcoded to the literal "registryid"
// — see canonicalLevelOrderFor) or if no match is found (e.g. minimal/
// foreign JSON omitting the entity's own id attribute).
//
// The returned singular (the matched key with its trailing "id"
// trimmed) is reused by the caller to directly resolve the Version-only
// "$RESOURCE*" family (e.g. "messageurl"/"message"/"messagebase64")
// without any further guessing.
func resolveIDKeyByValue(om *OrderedMap, xid *Xid, consumed map[string]bool) (key string, singular string) {
	if xid.Type == ENTITY_REGISTRY {
		return "", ""
	}
	var want string
	switch xid.Type {
	case ENTITY_GROUP:
		want = xid.GroupID
	case ENTITY_RESOURCE, ENTITY_META, ENTITY_VERSION:
		want = xid.ResourceID
	}
	if want == "" {
		return "", ""
	}
	for _, k := range om.Keys {
		if consumed[k] || k == "id" || !strings.HasSuffix(k, "id") {
			continue
		}
		if s, isStr := om.Values[k].(string); isStr && s == want {
			return k, strings.TrimSuffix(k, "id")
		}
	}
	return "", ""
}

// detectHeuristicCollections finds Registry-level (group-type) or
// Group-level (resource-type) "$COLLECTIONS" entries — there can be any
// number of them, with arbitrary model-defined plural names, so (unlike
// the Resource level's fixed "versions" collection) there's no fixed
// name to hardcode. Instead, any not-yet-consumed "<name>count" key with
// a matching not-yet-consumed "<name>url" sibling is treated as one
// collection; its own inlined map, "<name>", is included too if present.
// Matches are returned sorted alphabetically by base name, each
// contributing up to 3 keys in spec order: "<name>url", "<name>count",
// "<name>".
func detectHeuristicCollections(om *OrderedMap, consumed map[string]bool) []string {
	seenBase := map[string]bool{}
	var bases []string
	for _, k := range om.Keys {
		if consumed[k] || !strings.HasSuffix(k, "count") {
			continue
		}
		base := strings.TrimSuffix(k, "count")
		if base == "" || seenBase[base] {
			continue
		}
		urlKey := base + "url"
		if consumed[urlKey] {
			continue
		}
		if _, hasURL := om.Values[urlKey]; !hasURL {
			continue
		}
		seenBase[base] = true
		bases = append(bases, base)
	}
	sort.Strings(bases)

	var result []string
	for _, base := range bases {
		for _, suffix := range []string{"url", "count", ""} {
			k := base + suffix
			if consumed[k] {
				continue
			}
			if _, present := om.Values[k]; present {
				result = append(result, k)
			}
		}
	}
	return result
}
