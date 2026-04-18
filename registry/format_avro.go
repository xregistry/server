// Package registry - Avro schema format compatibility checker.
//
// IsValid verifies that a version's document is a syntactically valid
// Apache Avro schema (JSON encoding).
//
// IsCompatible checks whether two Avro schema versions are compatible
// in the given direction. The rules follow the closed-world assumption
// standard for schema registries (producers only emit fields defined
// in their schema):
//
//	"backward" — consumers using the NEW schema can read messages
//	             produced with the OLD schema.
//	             Permitted changes to the schema:
//	               • Delete any field (old messages have it; new
//	                 reader ignores extra writer fields).
//	               • Add an optional field that carries a default
//	                 value (old messages lack it; new reader uses the
//	                 default).
//	               • Widen a primitive type via Avro promotion
//	                 (int→long→float→double, string↔bytes).
//	             Forbidden changes:
//	               • Add a field without a default value (old messages
//	                 lack it and new reader has no fallback).
//	               • Change a type incompatibly.
//	               • Remove an enum symbol (old messages may carry it).
//	             Implemented as: old ⊆ new (checkAvroBackward(old, new))
//
//	"forward"  — consumers using the OLD schema can read messages
//	             produced with the NEW schema.
//	             Permitted changes to the schema:
//	               • Add any field (old consumers treat unknown fields
//	                 from the writer as ignored).
//	               • Delete a field that has a default value in the
//	                 OLD schema (new messages lack it; old reader
//	                 uses its default).
//	             Forbidden changes:
//	               • Delete a field that has NO default in the OLD
//	                 schema (new messages lack it; old reader has no
//	                 fallback).
//	               • Change a type incompatibly.
//	               • Remove an enum symbol used by old messages.
//	             Implemented as: new ⊆ old (checkAvroBackward(new, old))
//	             (forward compat = backward compat with args swapped)
//
// Compatibility checks – status per Avro construct:
//
// Schemas
//   - [supported]     primitive types (null, boolean, int, long,
//     float, double, bytes, string)
//   - [supported]     named type references (resolved by name in the
//     parse context)
//   - [supported]     record  (field add/remove/type checked)
//   - [supported]     enum    (symbol add/remove checked; aliases
//     not checked)
//   - [supported]     array   (items type checked recursively)
//   - [supported]     map     (values type checked recursively)
//   - [supported]     union   (each writer branch must resolve to a
//     compatible reader branch)
//   - [supported]     fixed   (size must match)
//
// Avro type promotion (widening — backward-compat direction only)
//   - [supported]     int  → long, float, double
//   - [supported]     long → float, double
//   - [supported]     float → double
//   - [supported]     string ↔ bytes
//
// Known limitations
//   - Logical types (decimal, date, uuid, …) are treated as their
//     underlying primitive for compat purposes.
//   - Field aliases are not used for name matching.
//   - Schema aliases on named types are not used for name matching.
//   - Namespace / full-name resolution is done by simple string
//     comparison of the "name" fields.

package registry

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	. "github.com/xregistry/server/common"
)

const AVRO_FORMAT = "avro*"

func init() {
	RegisterFormat(AVRO_FORMAT, FormatAvro{})
}

// FormatAvro implements the Format interface for Apache Avro schemas.
type FormatAvro struct{}

// IsValid checks that the version document is a valid Avro schema.
// string(1st return value) indicates whether validation was attempted
// true or false, why not...
// xErr is the error to use if we're returning an error to the user. Which
// may happen in both cases based on strictvalidation=true
func (fa FormatAvro) IsValid(ver *Version) (string, *XRError) {
	format := ver.GetAsString("format")
	if ok, _ := regexp.MatchString("(?i)"+AVRO_FORMAT, format); !ok {
		return "true", NewXRError("bad_request", ver.XID,
			"error_detail="+
				fmt.Sprintf(`Version %q has a "format" value of %q, was `+
					`expecting %q`, ver.XID, format, AVRO_FORMAT))
	}

	if ver.Resource.ResourceModel.GetHasDocument() == false {
		return "true", NewXRError("format_violation", ver.XID,
			"format="+format).
			SetDetailf(`The Resource (%s) for Version %q does not have `+
				`"hasdocument" in its resource model set to "true", and an `+
				`empty/missing document is not compliant.`,
				ver.Resource.XID, ver.XID)
	}

	if resURL := ver.Get(ver.Resource.Singular + "url"); !IsNil(resURL) {
		return "false, data stored externally",
			NewXRError("format_external", ver.XID)
	}

	buf := []byte(nil)
	if bufAny := ver.Get(ver.Resource.Singular); !IsNil(bufAny) {
		buf = bufAny.([]byte)
	}

	if len(buf) == 0 {
		return "true", NewXRError("format_violation", ver.XID,
			"format="+ver.GetAsString("format")).
			SetDetailf("Version %q is empty and therefore not a "+
				"valid avro schema file.", ver.XID)
	}

	if err := IsValidAvro(buf); err != nil {
		return "true", NewXRError("format_violation", ver.XID,
			"format="+ver.GetAsString("format")).
			SetDetailf("Version %q is not a valid avro schema file: %s.",
				ver.XID, err)
	}

	return "true", nil
}

// IsCompatible checks whether newVersion is compatible with
// oldVersion in the given direction.
func (fa FormatAvro) IsCompatible(
	direction string,
	oldVersion *Version,
	newVersion *Version,
) (string, *XRError) {
	oldBuf, newBuf := []byte(nil), []byte(nil)

	reason, xErr := fa.IsValid(oldVersion)
	if xErr != nil {
		return reason, xErr
	}

	reason, xErr = fa.IsValid(newVersion)
	if xErr != nil {
		return reason, xErr
	}

	if b := oldVersion.Get(oldVersion.Resource.Singular); !IsNil(b) {
		oldBuf = b.([]byte)
	}

	if b := newVersion.Get(newVersion.Resource.Singular); !IsNil(b) {
		newBuf = b.([]byte)
	}

	var oldSchema, newSchema interface{}
	if err := json.Unmarshal(oldBuf, &oldSchema); err != nil {
		return "false", NewXRError("format_violation", oldVersion.XID,
			"format="+oldVersion.GetAsString("format")).
			SetDetailf("Version %q is not a valid avro schema file: %s.",
				oldVersion.XID, err.Error())
	}
	if err := json.Unmarshal(newBuf, &newSchema); err != nil {
		return "false", NewXRError("format_violation", newVersion.XID,
			"format="+newVersion.GetAsString("format")).
			SetDetailf("Version %q is not a valid avro schema file: %s.",
				newVersion.XID, err.Error())
	}

	if err := checkAvroCompat(direction, oldSchema, newSchema); err != nil {
		compat := newVersion.
			Resource.
			MustFindMeta(false, FOR_READ).
			GetAsString("compatibility")

		return "true", NewXRError("compatibility_violation", newVersion.XID,
			"compat="+compat).
			SetDetailf("Version %q isn't %q compatible with %q: %s",
				newVersion.XID, compat, oldVersion.XID,
				err.Error())
	}

	return "true", nil
}

// ── Public helpers ─────────────────────────────────────────────────

// IsValidAvro returns nil when buf is a syntactically valid Avro
// schema (JSON encoding), or an error describing the problem.
func IsValidAvro(buf []byte) error {
	var raw interface{}
	if err := json.Unmarshal(buf, &raw); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}
	named := map[string]bool{}
	return validateAvroSchema(raw, named)
}

// ── Internal: validation ───────────────────────────────────────────

var avroPrimitives = map[string]bool{
	"null": true, "boolean": true,
	"int": true, "long": true,
	"float": true, "double": true,
	"bytes": true, "string": true,
}

func validateAvroSchema(
	s interface{},
	named map[string]bool,
) error {
	switch v := s.(type) {
	case string:
		if avroPrimitives[v] {
			return nil
		}
		// Named type reference
		if named[v] {
			return nil
		}
		return fmt.Errorf("unknown type or reference %q", v)

	case map[string]interface{}:
		return validateAvroObject(v, named)

	case []interface{}:
		// Union
		if len(v) == 0 {
			return fmt.Errorf("union must have at least one branch")
		}
		seen := map[string]bool{}
		for i, branch := range v {
			if err := validateAvroSchema(branch, named); err != nil {
				return fmt.Errorf("union branch %d: %v", i, err)
			}
			k := avroTypeName(branch)
			if seen[k] {
				return fmt.Errorf(
					"union has duplicate type %q at branch %d", k, i,
				)
			}
			seen[k] = true
		}
		return nil

	default:
		return fmt.Errorf("unexpected schema value type %T", s)
	}
}

func validateAvroObject(
	m map[string]interface{},
	named map[string]bool,
) error {
	t, _ := m["type"].(string)
	switch t {
	case "record":
		return validateAvroRecord(m, named)
	case "enum":
		return validateAvroEnum(m, named)
	case "array":
		items, ok := m["items"]
		if !ok {
			return fmt.Errorf("array schema missing \"items\"")
		}
		return validateAvroSchema(items, named)
	case "map":
		values, ok := m["values"]
		if !ok {
			return fmt.Errorf("map schema missing \"values\"")
		}
		return validateAvroSchema(values, named)
	case "fixed":
		return validateAvroFixed(m, named)
	case "":
		// Might be an inline complex schema using the long-form
		// primitive: {"type":"int"} is handled by the string case
		// when the value IS a string; here the whole thing is an
		// object whose "type" value is itself complex.
		inner, ok := m["type"]
		if !ok {
			return fmt.Errorf("schema object missing \"type\" field")
		}
		return validateAvroSchema(inner, named)
	default:
		// The spec allows {"type":"int"} as an object form of a
		// primitive, and also {"type": unionOrComplexSchema}.
		if avroPrimitives[t] {
			return nil
		}
		// Named type reference object (e.g. {"type":"MyRecord"})
		inner, _ := m["type"]
		if inner != nil {
			return validateAvroSchema(inner, named)
		}
		return fmt.Errorf("unknown schema type %q", t)
	}
}

func validateAvroRecord(
	m map[string]interface{},
	named map[string]bool,
) error {
	name, ok := m["name"].(string)
	if !ok || name == "" {
		return fmt.Errorf("record schema missing \"name\"")
	}
	named[name] = true
	fields, ok := m["fields"].([]interface{})
	if !ok {
		return fmt.Errorf("record %q missing \"fields\" array", name)
	}
	for i, f := range fields {
		fm, ok := f.(map[string]interface{})
		if !ok {
			return fmt.Errorf(
				"record %q field %d is not an object", name, i,
			)
		}
		fn, _ := fm["name"].(string)
		if fn == "" {
			return fmt.Errorf(
				"record %q field %d missing \"name\"", name, i,
			)
		}
		ft, ok := fm["type"]
		if !ok {
			return fmt.Errorf(
				"record %q field %q missing \"type\"", name, fn,
			)
		}
		if err := validateAvroSchema(ft, named); err != nil {
			return fmt.Errorf(
				"record %q field %q: %v", name, fn, err,
			)
		}
	}
	return nil
}

func validateAvroEnum(
	m map[string]interface{},
	named map[string]bool,
) error {
	name, ok := m["name"].(string)
	if !ok || name == "" {
		return fmt.Errorf("enum schema missing \"name\"")
	}
	named[name] = true
	syms, ok := m["symbols"].([]interface{})
	if !ok {
		return fmt.Errorf("enum %q missing \"symbols\" array", name)
	}
	if len(syms) == 0 {
		return fmt.Errorf("enum %q must have at least one symbol", name)
	}
	for i, s := range syms {
		if _, ok := s.(string); !ok {
			return fmt.Errorf(
				"enum %q symbol %d is not a string", name, i,
			)
		}
	}
	return nil
}

func validateAvroFixed(
	m map[string]interface{},
	named map[string]bool,
) error {
	name, ok := m["name"].(string)
	if !ok || name == "" {
		return fmt.Errorf("fixed schema missing \"name\"")
	}
	named[name] = true
	sz, ok := m["size"].(float64)
	if !ok || sz <= 0 {
		return fmt.Errorf("fixed %q missing or invalid \"size\"", name)
	}
	return nil
}

// avroTypeName returns a string key for union duplicate detection.
func avroTypeName(s interface{}) string {
	switch v := s.(type) {
	case string:
		return v
	case map[string]interface{}:
		t, _ := v["type"].(string)
		if name, ok := v["name"].(string); ok && name != "" {
			return t + ":" + name
		}
		return t
	default:
		return fmt.Sprintf("%T", s)
	}
}

// ── Internal: compatibility ────────────────────────────────────────

// checkAvroCompat is the top-level dispatcher.
//
// "backward": old ⊆ new — every old-valid message is also new-valid.
// "forward":  new ⊆ old — implemented by swapping the arguments.
func checkAvroCompat(
	direction string,
	old, new interface{},
) error {
	writer, reader := old, new
	if direction == "forward" {
		writer, reader = new, old
	}
	named := map[string]interface{}{}
	return checkAvroBackward(writer, reader, named)
}

// checkAvroBackward verifies writer ⊆ reader using Avro schema
// resolution rules. named is a registry of named schemas seen so far
// (used to resolve type-name references in both schemas).
func checkAvroBackward(
	writer, reader interface{},
	named map[string]interface{},
) error {
	// Resolve string references to their definitions
	w := resolveAvroRef(writer, named)
	r := resolveAvroRef(reader, named)

	// Register any named types encountered
	registerAvroNamed(w, named)
	registerAvroNamed(r, named)

	wStr, wIsStr := w.(string)
	rStr, rIsStr := r.(string)

	// Both primitives / references
	if wIsStr && rIsStr {
		return checkAvroPrimitiveCompat(wStr, rStr)
	}

	// Writer is union
	if wArr, ok := w.([]interface{}); ok {
		return checkAvroWriterUnion(wArr, r, named)
	}

	// Reader is union
	if rArr, ok := r.([]interface{}); ok {
		return checkAvroReaderUnion(w, rArr, named)
	}

	wMap, wIsMap := w.(map[string]interface{})
	rMap, rIsMap := r.(map[string]interface{})

	if !wIsMap || !rIsMap {
		return fmt.Errorf("schema type mismatch: %T vs %T", w, r)
	}

	wType := avroEffectiveType(wMap)
	rType := avroEffectiveType(rMap)

	if wType != rType {
		// Allow primitive promotion via string representation
		if wType != "" && rType != "" {
			return checkAvroPrimitiveCompat(wType, rType)
		}
		return fmt.Errorf(
			"schema type changed from %q to %q", wType, rType,
		)
	}

	switch wType {
	case "record":
		return checkAvroRecordCompat(wMap, rMap, named)
	case "enum":
		return checkAvroEnumCompat(wMap, rMap)
	case "array":
		return checkAvroBackward(wMap["items"], rMap["items"], named)
	case "map":
		return checkAvroBackward(wMap["values"], rMap["values"], named)
	case "fixed":
		return checkAvroFixedCompat(wMap, rMap)
	default:
		return checkAvroPrimitiveCompat(wType, rType)
	}
}

// avroEffectiveType returns the Avro type string for a schema object.
func avroEffectiveType(m map[string]interface{}) string {
	t, _ := m["type"].(string)
	return t
}

// resolveAvroRef resolves a string reference using the named registry.
func resolveAvroRef(
	s interface{},
	named map[string]interface{},
) interface{} {
	str, ok := s.(string)
	if !ok {
		return s
	}
	if def, found := named[str]; found {
		return def
	}
	return s
}

// registerAvroNamed adds record/enum/fixed definitions to named.
func registerAvroNamed(s interface{}, named map[string]interface{}) {
	m, ok := s.(map[string]interface{})
	if !ok {
		return
	}
	n, _ := m["name"].(string)
	if n == "" {
		return
	}
	t, _ := m["type"].(string)
	switch t {
	case "record", "enum", "fixed":
		if _, exists := named[n]; !exists {
			named[n] = s
		}
	}
}

// checkAvroPrimitiveCompat checks type promotion rules.
func checkAvroPrimitiveCompat(writer, reader string) error {
	if writer == reader {
		return nil
	}
	// Avro numeric promotions
	promotions := map[string][]string{
		"int":    {"long", "float", "double"},
		"long":   {"float", "double"},
		"float":  {"double"},
		"string": {"bytes"},
		"bytes":  {"string"},
	}
	for _, target := range promotions[writer] {
		if reader == target {
			return nil
		}
	}
	return fmt.Errorf(
		"type %q cannot be read as %q", writer, reader,
	)
}

// checkAvroWriterUnion: every writer branch must match a reader branch.
func checkAvroWriterUnion(
	writerBranches []interface{},
	reader interface{},
	named map[string]interface{},
) error {
	// If reader is also a union, match each writer branch to a reader
	// branch.
	readerBranches, readerIsUnion := reader.([]interface{})
	for i, wb := range writerBranches {
		if readerIsUnion {
			matched := false
			for _, rb := range readerBranches {
				if checkAvroBackward(wb, rb, named) == nil {
					matched = true
					break
				}
			}
			if !matched {
				return fmt.Errorf(
					"writer union branch %d has no compatible "+
						"reader branch",
					i,
				)
			}
		} else {
			if err := checkAvroBackward(wb, reader, named); err != nil {
				return fmt.Errorf(
					"writer union branch %d not compatible "+
						"with non-union reader: %v",
					i, err,
				)
			}
		}
	}
	return nil
}

// checkAvroReaderUnion: writer (non-union) must match at least one
// reader branch.
func checkAvroReaderUnion(
	writer interface{},
	readerBranches []interface{},
	named map[string]interface{},
) error {
	for _, rb := range readerBranches {
		if checkAvroBackward(writer, rb, named) == nil {
			return nil
		}
	}
	return fmt.Errorf(
		"writer type has no compatible branch in reader union",
	)
}

// checkAvroRecordCompat checks Avro record schema compatibility.
//
// For backward compat (writer=old, reader=new):
//   - For each field in reader: if absent in writer, reader field
//     must have a "default".
//   - For each field in both: types must be recursively compatible.
//   - Extra writer fields are silently ignored by the reader.
func checkAvroRecordCompat(
	writer, reader map[string]interface{},
	named map[string]interface{},
) error {
	wName, _ := writer["name"].(string)
	rName, _ := reader["name"].(string)
	if wName != rName {
		return fmt.Errorf(
			"record name changed from %q to %q", wName, rName,
		)
	}

	wFields := avroFieldMap(writer)
	rFields := avroFieldMap(reader)

	// For each reader field, verify writer compatibility
	for fname, rf := range rFields {
		wf, inWriter := wFields[fname]
		if !inWriter {
			// Reader field absent in writer: must have a default
			if _, hasDefault := rf["default"]; !hasDefault {
				return fmt.Errorf(
					"record %q: field %q added to reader "+
						"without a default value "+
						"(old writer data lacks this field)",
					rName, fname,
				)
			}
			continue
		}
		// Field in both: types must be compatible
		wType, ok1 := wf["type"]
		rType, ok2 := rf["type"]
		if !ok1 || !ok2 {
			continue
		}
		if err := checkAvroBackward(wType, rType, named); err != nil {
			return fmt.Errorf(
				"record %q field %q: %v", rName, fname, err,
			)
		}
	}
	return nil
}

// avroFieldMap returns a map of field name → field object for a
// record schema.
func avroFieldMap(
	record map[string]interface{},
) map[string]map[string]interface{} {
	out := map[string]map[string]interface{}{}
	fields, _ := record["fields"].([]interface{})
	for _, f := range fields {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := fm["name"].(string)
		if name != "" {
			out[name] = fm
		}
	}
	return out
}

// checkAvroEnumCompat checks Avro enum schema compatibility.
//
// Every symbol in the writer enum must appear in the reader enum
// (reader must be able to handle all values the writer may produce).
// The reader may add new symbols.
func checkAvroEnumCompat(
	writer, reader map[string]interface{},
) error {
	wName, _ := writer["name"].(string)
	rName, _ := reader["name"].(string)
	if wName != rName {
		return fmt.Errorf(
			"enum name changed from %q to %q", wName, rName,
		)
	}

	rSyms := avroSymbolSet(reader)
	wSyms, _ := writer["symbols"].([]interface{})
	for _, s := range wSyms {
		sym, _ := s.(string)
		if sym == "" {
			continue
		}
		if !rSyms[sym] {
			// Reader missing a writer symbol; check for reader default
			if _, hasDefault := reader["default"]; hasDefault {
				continue
			}
			return fmt.Errorf(
				"enum %q: symbol %q removed from reader "+
					"(writer may produce this value)",
				rName, sym,
			)
		}
	}
	return nil
}

// avroSymbolSet returns a set of symbol strings for an enum schema.
func avroSymbolSet(
	enum map[string]interface{},
) map[string]bool {
	out := map[string]bool{}
	syms, _ := enum["symbols"].([]interface{})
	for _, s := range syms {
		if str, ok := s.(string); ok {
			out[str] = true
		}
	}
	return out
}

// checkAvroFixedCompat verifies fixed schemas are identical in size.
func checkAvroFixedCompat(
	writer, reader map[string]interface{},
) error {
	wName, _ := writer["name"].(string)
	rName, _ := reader["name"].(string)
	if wName != rName {
		return fmt.Errorf(
			"fixed name changed from %q to %q", wName, rName,
		)
	}
	wSize, _ := writer["size"].(float64)
	rSize, _ := reader["size"].(float64)
	if wSize != rSize {
		return fmt.Errorf(
			"fixed %q size changed from %g to %g", wName, wSize, rSize,
		)
	}
	return nil
}

// ── Utility ────────────────────────────────────────────────────────

// avroSchema is a convenience builder for inline test schemas.
// It joins the given tokens and returns a compact JSON string.
// (Used only in tests; exported so the test file can see it.)
func avroJoin(parts ...string) string {
	return strings.Join(parts, "")
}
