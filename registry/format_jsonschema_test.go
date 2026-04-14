package registry

// Unit tests for IsValid (via IsValidJson) and IsCompatible (via
// checkCompat) covering all supported JSON Schema features listed at
// the top of format_jsonschema.go.
//
// Tests are run with: make utest

import (
	"encoding/json"
	"testing"
)

// schemaMap parses a JSON string into a map[string]interface{}.
// Panics on invalid JSON to keep test cases concise.
func schemaMap(s string) map[string]interface{} {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		panic("schemaMap: " + err.Error() + ": " + s)
	}
	return m
}

// ──────────────────────────────────────────────────────────────────
// IsValid – tested via the exported IsValidJson helper so no DB is
// required.  The method FormatJson.IsValid is just a thin wrapper
// that extracts []byte from a Version and calls IsValidJson.
// ──────────────────────────────────────────────────────────────────

func TestIsValidJson(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		wantErr bool
	}{
		// ── Valid schemas ──────────────────────────────────────────
		{
			name:   "empty object schema",
			schema: `{}`,
		},
		{
			name:   "boolean true schema",
			schema: `true`,
		},
		{
			name:   "boolean false schema",
			schema: `false`,
		},
		{
			name:   "type string",
			schema: `{"type":"string"}`,
		},
		{
			name:   "type array form",
			schema: `{"type":["string","null"]}`,
		},
		{
			name:   "enum",
			schema: `{"enum":["a","b","c"]}`,
		},
		{
			name:   "const",
			schema: `{"const":"fixed-value"}`,
		},
		{
			name:   "allOf",
			schema: `{"allOf":[{"type":"string"},{"minLength":1}]}`,
		},
		{
			name:   "anyOf",
			schema: `{"anyOf":[{"type":"string"},{"type":"number"}]}`,
		},
		{
			name:   "oneOf",
			schema: `{"oneOf":[{"type":"string"},{"type":"integer"}]}`,
		},
		{
			name:   "not",
			schema: `{"not":{"type":"string"}}`,
		},
		{
			name: "if/then/else",
			schema: `{"if":{"type":"string"},` +
				`"then":{"minLength":1},"else":{"type":"number"}}`,
		},
		{
			name: "object keywords",
			schema: `{
				"type":"object",
				"properties":{"name":{"type":"string"}},
				"required":["name"],
				"additionalProperties":false,
				"patternProperties":{"^x-":{}},
				"unevaluatedProperties":false,
				"propertyNames":{"minLength":1},
				"dependentRequired":{"a":["b"]},
				"dependentSchemas":{"a":{"required":["b"]}},
				"minProperties":1,
				"maxProperties":10
			}`,
		},
		{
			name: "array keywords – uniform items",
			schema: `{
				"type":"array",
				"items":{"type":"string"},
				"additionalItems":false,
				"unevaluatedItems":false,
				"contains":{"type":"string"},
				"minContains":1,
				"maxContains":5,
				"minItems":0,
				"maxItems":10,
				"uniqueItems":true
			}`,
		},
		{
			name: "array keywords – prefixItems",
			schema: `{
				"type":"array",
				"prefixItems":[{"type":"string"},{"type":"number"}],
				"items":false
			}`,
		},
		{
			name: "string keywords",
			schema: `{
				"type":"string",
				"minLength":1,
				"maxLength":100,
				"pattern":"^[a-z]+$",
				"format":"email",
				"contentEncoding":"base64",
				"contentMediaType":"application/json",
				"contentSchema":{"type":"object"}
			}`,
		},
		{
			name: "number keywords",
			schema: `{
				"type":"number",
				"minimum":0,
				"maximum":100,
				"exclusiveMinimum":0,
				"exclusiveMaximum":100,
				"multipleOf":5
			}`,
		},
		{
			name:   "integer type",
			schema: `{"type":"integer","minimum":0}`,
		},

		// ── Invalid schemas ────────────────────────────────────────
		{
			name:    "invalid JSON syntax",
			schema:  `{"type":}`,
			wantErr: true,
		},
		{
			name:    "empty input",
			schema:  ``,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := IsValidJson([]byte(tc.schema))
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────
// IsCompatible – tested via checkCompat which contains all the
// logic.  FormatJson.IsCompatible is a thin wrapper that extracts
// []byte from two Versions, unmarshals them and calls checkCompat.
//
// Convention used throughout:
//   "backward": old ⊆ new  (new can read old-produced data)
//   "forward":  new ⊆ old  (old can read new-produced data)
//               implemented as checkCompat("forward", old, new)
//                which swaps args and runs the backward check.
// ──────────────────────────────────────────────────────────────────

type compatCase struct {
	name    string
	dir     string // "backward" or "forward"
	old     interface{}
	new     interface{}
	wantErr bool
}

func runCompatCases(t *testing.T, cases []compatCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkCompat(tc.dir, tc.old, tc.new)
			if tc.wantErr && err == nil {
				t.Errorf("expected incompatibility, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected incompatibility: %v", err)
			}
		})
	}
}

// ── Boolean schemas ───────────────────────────────────────────────

func TestCheckCompat_BooleanSchemas(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "true backward true – compat",
			dir:  "backward", old: true, new: true,
		},
		{
			name: "false backward false – compat",
			dir:  "backward", old: false, new: false,
		},
		{
			name: "false backward true – compat (false ⊆ true)",
			dir:  "backward", old: false, new: true,
		},
		{
			name:    "true backward false – incompatible",
			dir:     "backward", old: true, new: false,
			wantErr: true,
		},
		{
			name: "map backward true – compat (anything ⊆ true)",
			dir:  "backward",
			old:  schemaMap(`{"type":"string"}`),
			new:  true,
		},
		{
			name: "false backward map – compat (false ⊆ anything)",
			dir:  "backward", old: false,
			new: schemaMap(`{"type":"string"}`),
		},
		{
			// any non-false schema vs new=false is not compat
			name:    "map backward false – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string"}`),
			new:     false,
			wantErr: true,
		},
	})
}

// ── type keyword ──────────────────────────────────────────────────

func TestCheckCompat_Type(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same type – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"string"}`),
			new:  schemaMap(`{"type":"string"}`),
		},
		{
			name: "integer widens to number – compat (backward)",
			dir:  "backward",
			old:  schemaMap(`{"type":"integer"}`),
			new:  schemaMap(`{"type":"number"}`),
		},
		{
			name: "no old type, new adds type – compat " +
				"(typesSubsumed: empty old → true)",
			dir: "backward",
			old: schemaMap(`{}`),
			new: schemaMap(`{"type":"string"}`),
		},
		{
			// Conservative: typesSubsumed(["string"], []) == false
			// even though {} is semantically more permissive.
			name:    "old type, new removes type – incompatible (conservative)",
			dir:     "backward",
			old:     schemaMap(`{"type":"string"}`),
			new:     schemaMap(`{}`),
			wantErr: true,
		},
		{
			name:    "string vs number – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string"}`),
			new:     schemaMap(`{"type":"number"}`),
			wantErr: true,
		},
		{
			name: "type array – compat when new is superset",
			dir:  "backward",
			old:  schemaMap(`{"type":["string","integer"]}`),
			new:  schemaMap(`{"type":["string","integer","number"]}`),
		},
		{
			name:    "type array – incompatible when new drops a type",
			dir:     "backward",
			old:     schemaMap(`{"type":["string","integer"]}`),
			new:     schemaMap(`{"type":["string"]}`),
			wantErr: true,
		},
		{
			name: "forward: narrowing type is allowed",
			dir:  "forward",
			old:  schemaMap(`{"type":"number"}`),
			new:  schemaMap(`{"type":"integer"}`),
		},
		{
			name:    "forward: widening type is forbidden",
			dir:     "forward",
			old:     schemaMap(`{"type":"integer"}`),
			new:     schemaMap(`{"type":"number"}`),
			wantErr: true,
		},
	})
}

// ── enum ──────────────────────────────────────────────────────────

func TestCheckCompat_Enum(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same enum – compat",
			dir:  "backward",
			old:  schemaMap(`{"enum":["a","b"]}`),
			new:  schemaMap(`{"enum":["a","b"]}`),
		},
		{
			name: "new superset enum – compat",
			dir:  "backward",
			old:  schemaMap(`{"enum":["a","b"]}`),
			new:  schemaMap(`{"enum":["a","b","c"]}`),
		},
		{
			name:    "new missing old enum value – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"enum":["a","b"]}`),
			new:     schemaMap(`{"enum":["a"]}`),
			wantErr: true,
		},
		{
			name:    "new adds enum restriction – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string"}`),
			new:     schemaMap(`{"enum":["a","b"]}`),
			wantErr: true,
		},
		{
			name:    "old has enum new removes it – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"enum":["a","b"]}`),
			new:     schemaMap(`{"type":"string"}`),
			wantErr: true,
		},
	})
}

// ── const ─────────────────────────────────────────────────────────

func TestCheckCompat_Const(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same const – compat",
			dir:  "backward",
			old:  schemaMap(`{"const":"foo"}`),
			new:  schemaMap(`{"const":"foo"}`),
		},
		{
			name: "old has const new removes it – compat " +
				"(new is more permissive)",
			dir: "backward",
			old: schemaMap(`{"const":"foo"}`),
			new: schemaMap(`{}`),
		},
		{
			name:    "new adds const – incompatible",
			dir:     "backward",
			old:     schemaMap(`{}`),
			new:     schemaMap(`{"const":"foo"}`),
			wantErr: true,
		},
		{
			name:    "const value changed – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"const":"foo"}`),
			new:     schemaMap(`{"const":"bar"}`),
			wantErr: true,
		},
	})
}

// ── allOf ─────────────────────────────────────────────────────────

func TestCheckCompat_AllOf(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "old allOf compatible with new – compat",
			dir:  "backward",
			old: schemaMap(
				`{"allOf":[{"type":"string"},{"minLength":1}]}`,
			),
			new: schemaMap(`{"type":"string"}`),
		},
		{
			// Use an untyped old schema so that typesSubsumed([], [])
			// passes; demonstrates conservative allOf sub check.
			name: "new allOf compatible with old – compat",
			dir:  "backward",
			old:  schemaMap(`{}`),
			new:  schemaMap(`{"allOf":[{}]}`),
		},
		{
			name:    "new allOf sub narrows type – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string"}`),
			new:     schemaMap(`{"allOf":[{"type":"number"}]}`),
			wantErr: true,
		},
	})
}

// ── anyOf ─────────────────────────────────────────────────────────

func TestCheckCompat_AnyOf(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "old anyOf all subs compat with new – compat",
			dir:  "backward",
			old: schemaMap(
				`{"anyOf":[{"type":"string"},{"type":"string"}]}`,
			),
			new: schemaMap(`{"type":"string"}`),
		},
		{
			// Use an untyped old schema to avoid typesSubsumed
			// rejecting the missing top-level type on anyOf.
			name: "new anyOf – old matches one sub – compat",
			dir:  "backward",
			old:  schemaMap(`{}`),
			new: schemaMap(
				`{"anyOf":[{"type":"string"},{"type":"number"}]}`,
			),
		},
		{
			name:    "new anyOf – old matches no sub – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string"}`),
			new:     schemaMap(`{"anyOf":[{"type":"number"}]}`),
			wantErr: true,
		},
	})
}

// ── oneOf ─────────────────────────────────────────────────────────

func TestCheckCompat_OneOf(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "old oneOf all subs compat with new – compat",
			dir:  "backward",
			old: schemaMap(
				`{"oneOf":[{"type":"string"},{"type":"string"}]}`,
			),
			new: schemaMap(`{"type":"string"}`),
		},
		{
			// Use an untyped old schema to avoid typesSubsumed
			// rejecting the missing top-level type on oneOf.
			name: "new oneOf – old matches one sub – compat",
			dir:  "backward",
			old:  schemaMap(`{}`),
			new: schemaMap(
				`{"oneOf":[{"type":"string"},{"type":"number"}]}`,
			),
		},
		{
			name:    "new oneOf – old matches no sub – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"integer"}`),
			new:     schemaMap(`{"oneOf":[{"type":"string"}]}`),
			wantErr: true,
		},
	})
}

// ── not ───────────────────────────────────────────────────────────

func TestCheckCompat_Not(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			// old={} avoids typesSubsumed rejecting {not:false}'s
			// empty top-level type. The intersection
			// {allOf:[{}, false]} is a false schema → compat.
			name: "new not false schema – compat",
			dir:  "backward",
			old:  schemaMap(`{}`),
			new:  schemaMap(`{"not":false}`),
		},
		{
			// {not:true} is semantically a false schema, but the
			// conservative intersection check does not detect
			// {allOf:[true, new]} as false → incompatible.
			name:    "old not true schema – incompatible (conservative)",
			dir:     "backward",
			old:     schemaMap(`{"not":true}`),
			new:     schemaMap(`{"type":"string"}`),
			wantErr: true,
		},
	})
}

// ── if / then / else ──────────────────────────────────────────────

func TestCheckCompat_IfThenElse(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "matching if/then/else – compat",
			dir:  "backward",
			old: schemaMap(`{
				"if":{"type":"string"},
				"then":{"minLength":1},
				"else":{"type":"number"}
			}`),
			new: schemaMap(`{
				"if":{"type":"string"},
				"then":{"minLength":1},
				"else":{"type":"number"}
			}`),
		},
		{
			name:    "new adds if/then/else – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string"}`),
			new:     schemaMap(`{"if":{"type":"string"},"then":{}}`),
			wantErr: true,
		},
	})
}

// ── Object: properties ────────────────────────────────────────────

func TestCheckCompat_ObjectProperties(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "add optional property – compat (backward)",
			dir:  "backward",
			old: schemaMap(`{
				"type":"object",
				"properties":{"name":{"type":"string"}}
			}`),
			new: schemaMap(`{
				"type":"object",
				"properties":{
					"name":{"type":"string"},
					"age":{"type":"integer"}
				}
			}`),
		},
		{
			name: "delete optional property – compat (backward)",
			dir:  "backward",
			old: schemaMap(`{
				"type":"object",
				"properties":{
					"name":{"type":"string"},
					"age":{"type":"integer"}
				}
			}`),
			new: schemaMap(`{
				"type":"object",
				"properties":{"name":{"type":"string"}}
			}`),
		},
		{
			name:    "property type narrowed – incompatible (backward)",
			dir:     "backward",
			old:     schemaMap(`{"type":"object","properties":{"x":{"type":"number"}}}`),
			new:     schemaMap(`{"type":"object","properties":{"x":{"type":"integer"}}}`),
			wantErr: true,
		},
	})
}

// ── Object: required ─────────────────────────────────────────────

func TestCheckCompat_ObjectRequired(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same required – compat",
			dir:  "backward",
			old: schemaMap(`{
				"type":"object","required":["name"]
			}`),
			new: schemaMap(`{
				"type":"object","required":["name"]
			}`),
		},
		{
			name: "new removes required field – compat (backward)",
			dir:  "backward",
			old: schemaMap(`{
				"type":"object","required":["name","age"]
			}`),
			new: schemaMap(`{
				"type":"object","required":["name"]
			}`),
		},
		{
			name:    "new adds required field – incompatible (backward)",
			dir:     "backward",
			old:     schemaMap(`{"type":"object","required":["name"]}`),
			new:     schemaMap(`{"type":"object","required":["name","age"]}`),
			wantErr: true,
		},
	})
}

// ── Object: additionalProperties ─────────────────────────────────

func TestCheckCompat_ObjectAdditionalProperties(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "both allow additional – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"object","additionalProperties":true}`),
			new:  schemaMap(`{"type":"object","additionalProperties":true}`),
		},
		{
			name:    "new disallows additional – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"object","additionalProperties":true}`),
			new:     schemaMap(`{"type":"object","additionalProperties":false}`),
			wantErr: true,
		},
		{
			name: "new allows additional – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"object","additionalProperties":false}`),
			new:  schemaMap(`{"type":"object","additionalProperties":true}`),
		},
	})
}

// ── Object: patternProperties ─────────────────────────────────────

func TestCheckCompat_ObjectPatternProperties(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same patternProperty – compat",
			dir:  "backward",
			old: schemaMap(
				`{"type":"object",` +
					`"patternProperties":{"^x-":{"type":"string"}}}`,
			),
			new: schemaMap(
				`{"type":"object",` +
					`"patternProperties":{"^x-":{"type":"string"}}}`,
			),
		},
		{
			name:    "patternProperty type narrowed – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"object","patternProperties":{"^x-":{"type":"number"}}}`),
			new:     schemaMap(`{"type":"object","patternProperties":{"^x-":{"type":"integer"}}}`),
			wantErr: true,
		},
	})
}

// ── Object: unevaluatedProperties ────────────────────────────────

func TestCheckCompat_ObjectUnevaluatedProperties(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "both unevaluatedProperties false – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"object","unevaluatedProperties":false}`),
			new:  schemaMap(`{"type":"object","unevaluatedProperties":false}`),
		},
		{
			name: "old has unevaluatedProperties new removes – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"object","unevaluatedProperties":false}`),
			new:  schemaMap(`{"type":"object"}`),
		},
		{
			name:    "new adds unevaluatedProperties – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"object"}`),
			new:     schemaMap(`{"type":"object","unevaluatedProperties":false}`),
			wantErr: true,
		},
	})
}

// ── Object: propertyNames ─────────────────────────────────────────

func TestCheckCompat_ObjectPropertyNames(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same propertyNames – compat",
			dir:  "backward",
			old: schemaMap(
				`{"type":"object","propertyNames":{"minLength":1}}`,
			),
			new: schemaMap(
				`{"type":"object","propertyNames":{"minLength":1}}`,
			),
		},
		{
			name: "propertyNames widened – compat (backward)",
			dir:  "backward",
			old: schemaMap(
				`{"type":"object","propertyNames":{"minLength":3}}`,
			),
			new: schemaMap(
				`{"type":"object","propertyNames":{"minLength":1}}`,
			),
		},
		{
			name:    "new adds propertyNames – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"object"}`),
			new:     schemaMap(`{"type":"object","propertyNames":{"minLength":1}}`),
			wantErr: true,
		},
	})
}

// ── Object: dependentRequired ─────────────────────────────────────

func TestCheckCompat_ObjectDependentRequired(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same dependentRequired – compat",
			dir:  "backward",
			old: schemaMap(
				`{"type":"object","dependentRequired":{"a":["b"]}}`,
			),
			new: schemaMap(
				`{"type":"object","dependentRequired":{"a":["b"]}}`,
			),
		},
		{
			name: "new removes dep requirement – compat (backward)",
			dir:  "backward",
			old: schemaMap(
				`{"type":"object","dependentRequired":{"a":["b","c"]}}`,
			),
			new: schemaMap(
				`{"type":"object","dependentRequired":{"a":["b"]}}`,
			),
		},
		{
			name:    "new adds dependentRequired key – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"object"}`),
			new:     schemaMap(`{"type":"object","dependentRequired":{"a":["b"]}}`),
			wantErr: true,
		},
		{
			name:    "new adds field to existing dep – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"object","dependentRequired":{"a":["b"]}}`),
			new:     schemaMap(`{"type":"object","dependentRequired":{"a":["b","c"]}}`),
			wantErr: true,
		},
	})
}

// ── Object: dependentSchemas ──────────────────────────────────────

func TestCheckCompat_ObjectDependentSchemas(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same dependentSchemas – compat",
			dir:  "backward",
			old: schemaMap(`{"type":"object","dependentSchemas":` +
				`{"a":{"required":["b"]}}}`),
			new: schemaMap(`{"type":"object","dependentSchemas":` +
				`{"a":{"required":["b"]}}}`),
		},
		{
			name:    "new adds dependentSchemas key – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"object"}`),
			new:     schemaMap(`{"type":"object","dependentSchemas":{"a":{"required":["b"]}}}`),
			wantErr: true,
		},
	})
}

// ── Object: minProperties / maxProperties ────────────────────────

func TestCheckCompat_ObjectMinMaxProperties(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same minProperties – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"object","minProperties":1}`),
			new:  schemaMap(`{"type":"object","minProperties":1}`),
		},
		{
			name: "new lowers minProperties – compat (widening)",
			dir:  "backward",
			old:  schemaMap(`{"type":"object","minProperties":2}`),
			new:  schemaMap(`{"type":"object","minProperties":1}`),
		},
		{
			name:    "new raises minProperties – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"object","minProperties":1}`),
			new:     schemaMap(`{"type":"object","minProperties":2}`),
			wantErr: true,
		},
		{
			name: "same maxProperties – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"object","maxProperties":10}`),
			new:  schemaMap(`{"type":"object","maxProperties":10}`),
		},
		{
			name: "new raises maxProperties – compat (widening)",
			dir:  "backward",
			old:  schemaMap(`{"type":"object","maxProperties":5}`),
			new:  schemaMap(`{"type":"object","maxProperties":10}`),
		},
		{
			name:    "new lowers maxProperties – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"object","maxProperties":10}`),
			new:     schemaMap(`{"type":"object","maxProperties":5}`),
			wantErr: true,
		},
	})
}

// ── Array: items (uniform) ────────────────────────────────────────

func TestCheckCompat_ArrayItems(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same items schema – compat",
			dir:  "backward",
			old: schemaMap(
				`{"type":"array","items":{"type":"string"}}`,
			),
			new: schemaMap(
				`{"type":"array","items":{"type":"string"}}`,
			),
		},
		{
			name: "old has items new removes – compat (permissive)",
			dir:  "backward",
			old:  schemaMap(`{"type":"array","items":{"type":"string"}}`),
			new:  schemaMap(`{"type":"array"}`),
		},
		{
			name:    "new adds items restriction – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"array"}`),
			new:     schemaMap(`{"type":"array","items":{"type":"string"}}`),
			wantErr: true,
		},
		{
			name:    "items type narrowed – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"array","items":{"type":"number"}}`),
			new:     schemaMap(`{"type":"array","items":{"type":"integer"}}`),
			wantErr: true,
		},
	})
}

// ── Array: prefixItems / tuple items ─────────────────────────────

func TestCheckCompat_ArrayPrefixItems(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same prefixItems – compat",
			dir:  "backward",
			old: schemaMap(`{
				"type":"array",
				"prefixItems":[{"type":"string"},{"type":"number"}]
			}`),
			new: schemaMap(`{
				"type":"array",
				"prefixItems":[{"type":"string"},{"type":"number"}]
			}`),
		},
		{
			name:    "prefixItem type narrowed – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"array","prefixItems":[{"type":"number"}]}`),
			new:     schemaMap(`{"type":"array","prefixItems":[{"type":"integer"}]}`),
			wantErr: true,
		},
	})
}

// ── Array: additionalItems ────────────────────────────────────────

func TestCheckCompat_ArrayAdditionalItems(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "both additionalItems true – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"array","additionalItems":true}`),
			new:  schemaMap(`{"type":"array","additionalItems":true}`),
		},
		{
			name:    "new disallows additionalItems – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"array","additionalItems":true}`),
			new:     schemaMap(`{"type":"array","additionalItems":false}`),
			wantErr: true,
		},
	})
}

// ── Array: unevaluatedItems ───────────────────────────────────────

func TestCheckCompat_ArrayUnevaluatedItems(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "both unevaluatedItems false – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"array","unevaluatedItems":false}`),
			new:  schemaMap(`{"type":"array","unevaluatedItems":false}`),
		},
		{
			name:    "new adds unevaluatedItems – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"array"}`),
			new:     schemaMap(`{"type":"array","unevaluatedItems":false}`),
			wantErr: true,
		},
		{
			name: "old has unevaluatedItems new removes – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"array","unevaluatedItems":false}`),
			new:  schemaMap(`{"type":"array"}`),
		},
	})
}

// ── Array: contains ───────────────────────────────────────────────

func TestCheckCompat_ArrayContains(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same contains – compat",
			dir:  "backward",
			old: schemaMap(
				`{"type":"array","contains":{"type":"string"}}`,
			),
			new: schemaMap(
				`{"type":"array","contains":{"type":"string"}}`,
			),
		},
		{
			name:    "new adds contains – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"array"}`),
			new:     schemaMap(`{"type":"array","contains":{"type":"string"}}`),
			wantErr: true,
		},
		{
			name: "old has contains new removes – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"array","contains":{"type":"string"}}`),
			new:  schemaMap(`{"type":"array"}`),
		},
	})
}

// ── Array: minContains / maxContains ─────────────────────────────

func TestCheckCompat_ArrayMinMaxContains(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "new lowers minContains – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"array","minContains":2}`),
			new:  schemaMap(`{"type":"array","minContains":1}`),
		},
		{
			name:    "new raises minContains – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"array","minContains":1}`),
			new:     schemaMap(`{"type":"array","minContains":2}`),
			wantErr: true,
		},
		{
			name: "new raises maxContains – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"array","maxContains":5}`),
			new:  schemaMap(`{"type":"array","maxContains":10}`),
		},
		{
			name:    "new lowers maxContains – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"array","maxContains":10}`),
			new:     schemaMap(`{"type":"array","maxContains":5}`),
			wantErr: true,
		},
	})
}

// ── Array: minItems / maxItems ────────────────────────────────────

func TestCheckCompat_ArrayMinMaxItems(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "new lowers minItems – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"array","minItems":2}`),
			new:  schemaMap(`{"type":"array","minItems":1}`),
		},
		{
			name:    "new raises minItems – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"array","minItems":1}`),
			new:     schemaMap(`{"type":"array","minItems":2}`),
			wantErr: true,
		},
		{
			name: "new raises maxItems – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"array","maxItems":5}`),
			new:  schemaMap(`{"type":"array","maxItems":10}`),
		},
		{
			name:    "new lowers maxItems – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"array","maxItems":10}`),
			new:     schemaMap(`{"type":"array","maxItems":5}`),
			wantErr: true,
		},
	})
}

// ── Array: uniqueItems ────────────────────────────────────────────

func TestCheckCompat_ArrayUniqueItems(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "both uniqueItems true – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"array","uniqueItems":true}`),
			new:  schemaMap(`{"type":"array","uniqueItems":true}`),
		},
		{
			name: "old uniqueItems true new false – compat " +
				"(new is more permissive)",
			dir: "backward",
			old: schemaMap(`{"type":"array","uniqueItems":true}`),
			new: schemaMap(`{"type":"array","uniqueItems":false}`),
		},
		{
			name:    "new adds uniqueItems true – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"array"}`),
			new:     schemaMap(`{"type":"array","uniqueItems":true}`),
			wantErr: true,
		},
	})
}

// ── String: minLength / maxLength ─────────────────────────────────

func TestCheckCompat_StringMinMaxLength(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "new lowers minLength – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"string","minLength":3}`),
			new:  schemaMap(`{"type":"string","minLength":1}`),
		},
		{
			name:    "new raises minLength – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string","minLength":1}`),
			new:     schemaMap(`{"type":"string","minLength":5}`),
			wantErr: true,
		},
		{
			name: "new raises maxLength – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"string","maxLength":10}`),
			new:  schemaMap(`{"type":"string","maxLength":100}`),
		},
		{
			name:    "new lowers maxLength – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string","maxLength":100}`),
			new:     schemaMap(`{"type":"string","maxLength":10}`),
			wantErr: true,
		},
	})
}

// ── String: pattern ───────────────────────────────────────────────

func TestCheckCompat_StringPattern(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same pattern – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"string","pattern":"^[a-z]+$"}`),
			new:  schemaMap(`{"type":"string","pattern":"^[a-z]+$"}`),
		},
		{
			name:    "pattern changed – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string","pattern":"^[a-z]+$"}`),
			new:     schemaMap(`{"type":"string","pattern":"^[A-Z]+$"}`),
			wantErr: true,
		},
		{
			name:    "new adds pattern – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string"}`),
			new:     schemaMap(`{"type":"string","pattern":"^[a-z]+$"}`),
			wantErr: true,
		},
		{
			name: "old has pattern new removes – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"string","pattern":"^[a-z]+$"}`),
			new:  schemaMap(`{"type":"string"}`),
		},
	})
}

// ── String: format ────────────────────────────────────────────────

func TestCheckCompat_StringFormat(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same format – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"string","format":"email"}`),
			new:  schemaMap(`{"type":"string","format":"email"}`),
		},
		{
			name:    "format changed – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string","format":"email"}`),
			new:     schemaMap(`{"type":"string","format":"uri"}`),
			wantErr: true,
		},
		{
			name:    "new adds format – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string"}`),
			new:     schemaMap(`{"type":"string","format":"email"}`),
			wantErr: true,
		},
		{
			name: "old has format new removes – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"string","format":"email"}`),
			new:  schemaMap(`{"type":"string"}`),
		},
	})
}

// ── String: contentEncoding ───────────────────────────────────────

func TestCheckCompat_StringContentEncoding(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same contentEncoding – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"string","contentEncoding":"base64"}`),
			new:  schemaMap(`{"type":"string","contentEncoding":"base64"}`),
		},
		{
			name:    "contentEncoding changed – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string","contentEncoding":"base64"}`),
			new:     schemaMap(`{"type":"string","contentEncoding":"quoted-printable"}`),
			wantErr: true,
		},
		{
			name:    "new adds contentEncoding – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string"}`),
			new:     schemaMap(`{"type":"string","contentEncoding":"base64"}`),
			wantErr: true,
		},
	})
}

// ── String: contentMediaType ──────────────────────────────────────

func TestCheckCompat_StringContentMediaType(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same contentMediaType – compat",
			dir:  "backward",
			old: schemaMap(
				`{"type":"string","contentMediaType":"application/json"}`,
			),
			new: schemaMap(
				`{"type":"string","contentMediaType":"application/json"}`,
			),
		},
		{
			name:    "contentMediaType changed – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string","contentMediaType":"application/json"}`),
			new:     schemaMap(`{"type":"string","contentMediaType":"text/plain"}`),
			wantErr: true,
		},
		{
			name:    "new adds contentMediaType – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string"}`),
			new:     schemaMap(`{"type":"string","contentMediaType":"application/json"}`),
			wantErr: true,
		},
	})
}

// ── String: contentSchema ─────────────────────────────────────────

func TestCheckCompat_StringContentSchema(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same contentSchema – compat",
			dir:  "backward",
			old: schemaMap(
				`{"type":"string","contentSchema":{"type":"object"}}`,
			),
			new: schemaMap(
				`{"type":"string","contentSchema":{"type":"object"}}`,
			),
		},
		{
			name:    "new adds contentSchema – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"string"}`),
			new:     schemaMap(`{"type":"string","contentSchema":{"type":"object"}}`),
			wantErr: true,
		},
		{
			name: "old has contentSchema new removes – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"string","contentSchema":{"type":"object"}}`),
			new:  schemaMap(`{"type":"string"}`),
		},
	})
}

// ── Number: minimum / maximum / exclusive ─────────────────────────

func TestCheckCompat_NumberMinMax(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same minimum – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"number","minimum":0}`),
			new:  schemaMap(`{"type":"number","minimum":0}`),
		},
		{
			name: "new lowers minimum – compat (widening)",
			dir:  "backward",
			old:  schemaMap(`{"type":"number","minimum":5}`),
			new:  schemaMap(`{"type":"number","minimum":0}`),
		},
		{
			name:    "new raises minimum – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"number","minimum":0}`),
			new:     schemaMap(`{"type":"number","minimum":5}`),
			wantErr: true,
		},
		{
			name:    "new adds minimum – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"number"}`),
			new:     schemaMap(`{"type":"number","minimum":0}`),
			wantErr: true,
		},
		{
			name: "same maximum – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"number","maximum":100}`),
			new:  schemaMap(`{"type":"number","maximum":100}`),
		},
		{
			name: "new raises maximum – compat (widening)",
			dir:  "backward",
			old:  schemaMap(`{"type":"number","maximum":50}`),
			new:  schemaMap(`{"type":"number","maximum":100}`),
		},
		{
			name:    "new lowers maximum – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"number","maximum":100}`),
			new:     schemaMap(`{"type":"number","maximum":50}`),
			wantErr: true,
		},
		{
			name:    "new adds maximum – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"number"}`),
			new:     schemaMap(`{"type":"number","maximum":100}`),
			wantErr: true,
		},
	})
}

func TestCheckCompat_NumberExclusiveMinMax(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same exclusiveMinimum – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"number","exclusiveMinimum":0}`),
			new:  schemaMap(`{"type":"number","exclusiveMinimum":0}`),
		},
		{
			name:    "new raises exclusiveMinimum – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"number","exclusiveMinimum":0}`),
			new:     schemaMap(`{"type":"number","exclusiveMinimum":5}`),
			wantErr: true,
		},
		{
			name: "exclusive stricter than inclusive at same value",
			dir:  "backward",
			old:  schemaMap(`{"type":"number","minimum":5}`),
			new: schemaMap(
				`{"type":"number","exclusiveMinimum":5}`,
			),
			// new exclusive min == old inclusive min → new is
			// stricter → backward incompatible
			wantErr: true,
		},
		{
			name: "same exclusiveMaximum – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"number","exclusiveMaximum":100}`),
			new:  schemaMap(`{"type":"number","exclusiveMaximum":100}`),
		},
		{
			name:    "new lowers exclusiveMaximum – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"number","exclusiveMaximum":100}`),
			new:     schemaMap(`{"type":"number","exclusiveMaximum":50}`),
			wantErr: true,
		},
	})
}

// ── Number: multipleOf ────────────────────────────────────────────

func TestCheckCompat_NumberMultipleOf(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "same multipleOf – compat",
			dir:  "backward",
			old:  schemaMap(`{"type":"number","multipleOf":5}`),
			new:  schemaMap(`{"type":"number","multipleOf":5}`),
		},
		{
			name: "new multipleOf divides old – compat (widening)",
			dir:  "backward",
			old:  schemaMap(`{"type":"number","multipleOf":10}`),
			new:  schemaMap(`{"type":"number","multipleOf":5}`),
		},
		{
			name:    "new multipleOf does not divide old – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"number","multipleOf":5}`),
			new:     schemaMap(`{"type":"number","multipleOf":3}`),
			wantErr: true,
		},
		{
			name:    "new adds multipleOf – incompatible",
			dir:     "backward",
			old:     schemaMap(`{"type":"number"}`),
			new:     schemaMap(`{"type":"number","multipleOf":5}`),
			wantErr: true,
		},
	})
}

// ── Forward compatibility direction ──────────────────────────────

func TestCheckCompat_ForwardDirection(t *testing.T) {
	runCompatCases(t, []compatCase{
		{
			name: "forward: add optional field – compat",
			dir:  "forward",
			old: schemaMap(`{
				"type":"object",
				"properties":{"name":{"type":"string"}}
			}`),
			new: schemaMap(`{
				"type":"object",
				"properties":{
					"name":{"type":"string"},
					"age":{"type":"integer"}
				}
			}`),
		},
		{
			name:    "forward: delete required field – incompatible",
			dir:     "forward",
			old:     schemaMap(`{"type":"object","required":["name","age"]}`),
			new:     schemaMap(`{"type":"object","required":["name"]}`),
			wantErr: true,
		},
		{
			name: "forward: add required field – compat " +
				"(old can still validate new with the field)",
			dir: "forward",
			old: schemaMap(
				`{"type":"object","required":["name"]}`,
			),
			new: schemaMap(
				`{"type":"object","required":["name","age"]}`,
			),
		},
		{
			name:    "forward: widen minimum – incompatible",
			dir:     "forward",
			old:     schemaMap(`{"type":"number","minimum":5}`),
			new:     schemaMap(`{"type":"number","minimum":0}`),
			wantErr: true,
		},
		{
			name: "forward: narrow minimum – compat",
			dir:  "forward",
			old:  schemaMap(`{"type":"number","minimum":0}`),
			new:  schemaMap(`{"type":"number","minimum":5}`),
		},
	})
}
