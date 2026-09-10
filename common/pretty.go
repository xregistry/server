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

func ParseJSONToObject(data []byte) (interface{}, error) {
	// Parse JSON while preserving key order

	var parseJSON func(decoder *json.Decoder) (interface{}, error)
	parseJSON = func(decoder *json.Decoder) (interface{}, error) {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case json.Delim:
			if t == '{' {
				ordered := &OrderedMap{Values: make(map[string]interface{})}
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
				arr := []interface{}{}
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
	Values map[string]interface{}
}

// UnmarshalJSON implements custom JSON unmarshaling to preserve key order
func (o *OrderedMap) UnmarshalJSON(data []byte) error {
	o.Values = make(map[string]interface{})
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
		var value interface{}
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
func CanonicalReorderJSON(data []byte, order CanonicalLevelOrder) ([]byte, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return trimmed, nil
	}
	obj, err := ParseJSONToObject(data)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	if err := writeCanonicalValue(buf, obj, "", order); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

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

func writeCanonicalValue(buf *bytes.Buffer, val interface{}, indent string, order CanonicalLevelOrder) error {
	switch v := val.(type) {
	case *OrderedMap:
		if xid, ok := canonicalEntityXid(v); ok {
			return writeCanonicalEntity(buf, v, indent, order(xid.Type), xid, order)
		}
		return writePlainMap(buf, v, indent, order)
	case []interface{}:
		return writeCanonicalArray(buf, v, indent, order)
	default:
		return writeScalar(buf, val)
	}
}

func writeScalar(buf *bytes.Buffer, val interface{}) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	buf.Write(b)
	return nil
}

func writeCanonicalArray(buf *bytes.Buffer, arr []interface{}, indent string, order CanonicalLevelOrder) error {
	if len(arr) == 0 {
		buf.WriteString("[]")
		return nil
	}
	childIndent := indent + "  "
	buf.WriteString("[\n")
	for i, item := range arr {
		buf.WriteString(childIndent)
		if err := writeCanonicalValue(buf, item, childIndent, order); err != nil {
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

// writePlainMap renders om with its OWN original key order preserved
// (this engine has no canonical ordering rule for a non-entity object —
// e.g. "labels", "capabilities", "model", or a collection map keyed by
// entity ID) — but each value is still recursed into, so nested entities
// (e.g. each Group instance inside a Groups collection map) still get
// their own attributes reordered.
func writePlainMap(buf *bytes.Buffer, om *OrderedMap, indent string, order CanonicalLevelOrder) error {
	if len(om.Keys) == 0 {
		buf.WriteString("{}")
		return nil
	}
	childIndent := indent + "  "
	buf.WriteString("{\n")
	for i, key := range om.Keys {
		buf.WriteString(childIndent)
		if err := writeScalar(buf, key); err != nil {
			return err
		}
		buf.WriteString(": ")
		if err := writeCanonicalValue(buf, om.Values[key], childIndent, order); err != nil {
			return err
		}
		if i < len(om.Keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString(indent + "}")
	return nil
}

// writeCanonicalEntity renders om (an object with its own valid "xid") in
// canonical order: spec attrs from tokens (in order, skipped if absent
// from om), a "$space" token emits a blank output line, an "$extensions"
// token expands to every one of om's OWN keys not otherwise consumed by
// a spec-attr token or by one of the heuristics below, alphabetized.
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
func writeCanonicalEntity(buf *bytes.Buffer, om *OrderedMap, indent string, tokens []string, xid *Xid, order CanonicalLevelOrder) error {
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

	// Build the final emission plan: a sequence of either a real
	// "key" to emit, or a blank-line marker — collapsing consecutive
	// blank markers and dropping any leading/trailing one (tokens is
	// already pre-collapsed by the caller, but $extensions can still
	// expand to zero keys, or a spec attr can be legitimately absent
	// from om, so re-collapse here against the REAL emitted sequence).
	type planItem struct {
		blank bool
		key   string
	}
	plan := make([]planItem, 0, len(tokens)+len(extNames))
	extensionsEmitted := false
	for _, t := range tokens {
		switch t {
		case "$space":
			plan = append(plan, planItem{blank: true})
			continue
		case "$extensions":
			// Only expand at the first occurrence — a level's token
			// list should never have more than one "$extensions" slot,
			// but guard against emitting extensions twice if it did.
			if !extensionsEmitted {
				extensionsEmitted = true
				for _, name := range extNames {
					plan = append(plan, planItem{key: name})
				}
			}
			continue
		case "$COLLECTIONS":
			for _, k := range collectionsKeys {
				plan = append(plan, planItem{key: k})
			}
			continue
		case "$RESOURCEurl":
			for _, k := range resourceFamilyKeys {
				plan = append(plan, planItem{key: k})
			}
			continue
		case "$RESOURCEproxyurl", "$RESOURCE", "$RESOURCEbase64":
			continue // already emitted via the "$RESOURCEurl" slot above
		case "id":
			if resolvedIDKey != "" {
				plan = append(plan, planItem{key: resolvedIDKey})
			}
			continue
		}
		if strings.HasPrefix(t, "$") {
			continue // unresolvable placeholder — no-op (see doc comment)
		}
		if _, present := om.Values[t]; present {
			plan = append(plan, planItem{key: t})
		}
	}
	// Collapse consecutive blanks + trim leading/trailing blanks.
	collapsed := make([]planItem, 0, len(plan))
	for _, p := range plan {
		if p.blank {
			if len(collapsed) == 0 || collapsed[len(collapsed)-1].blank {
				continue
			}
		}
		collapsed = append(collapsed, p)
	}
	for len(collapsed) > 0 && collapsed[len(collapsed)-1].blank {
		collapsed = collapsed[:len(collapsed)-1]
	}

	if len(collapsed) == 0 {
		buf.WriteString("{}")
		return nil
	}

	childIndent := indent + "  "
	buf.WriteString("{\n")
	// lastRealIdx tracks the index (within collapsed) of the last
	// non-blank entry, so we know when NOT to print a trailing comma.
	lastRealIdx := -1
	for i := len(collapsed) - 1; i >= 0; i-- {
		if !collapsed[i].blank {
			lastRealIdx = i
			break
		}
	}
	for i, p := range collapsed {
		if p.blank {
			buf.WriteString("\n")
			continue
		}
		buf.WriteString(childIndent)
		if err := writeScalar(buf, p.key); err != nil {
			return err
		}
		buf.WriteString(": ")
		if err := writeCanonicalValue(buf, om.Values[p.key], childIndent, order); err != nil {
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
