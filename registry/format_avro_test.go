package registry

// Unit tests for Avro IsValid (via IsValidAvro) and IsCompatible
// (via checkAvroCompat) covering all supported Avro constructs.
//
// Tests are run with: make utest

import (
	"encoding/json"
	"testing"
)

// ── Helpers ────────────────────────────────────────────────────────

type avroCase struct {
	name    string
	dir     string
	old     string // JSON-encoded Avro schema
	new     string // JSON-encoded Avro schema
	wantErr bool
}

func runAvroCases(t *testing.T, cases []avroCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var oldV, newV interface{}
			if err := json.Unmarshal(
				[]byte(tc.old), &oldV,
			); err != nil {
				t.Fatalf("bad old schema JSON: %v", err)
			}
			if err := json.Unmarshal(
				[]byte(tc.new), &newV,
			); err != nil {
				t.Fatalf("bad new schema JSON: %v", err)
			}
			err := checkAvroCompat(tc.dir, oldV, newV)
			if tc.wantErr && err == nil {
				t.Errorf("expected incompatibility, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected incompatibility: %v", err)
			}
		})
	}
}

// ── IsValidAvro ────────────────────────────────────────────────────

func TestIsValidAvro(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		wantErr bool
	}{
		// ── Valid ──────────────────────────────────────────────────
		{
			name:   "primitive null",
			schema: `"null"`,
		},
		{
			name:   "primitive string",
			schema: `"string"`,
		},
		{
			name:   "primitive int",
			schema: `"int"`,
		},
		{
			name:   "primitive long",
			schema: `"long"`,
		},
		{
			name:   "primitive float",
			schema: `"float"`,
		},
		{
			name:   "primitive double",
			schema: `"double"`,
		},
		{
			name:   "primitive bytes",
			schema: `"bytes"`,
		},
		{
			name:   "primitive boolean",
			schema: `"boolean"`,
		},
		{
			name: "record with fields",
			schema: `{
				"type":"record","name":"User",
				"fields":[
					{"name":"id","type":"int"},
					{"name":"name","type":"string"}
				]
			}`,
		},
		{
			name: "record with optional field (union with null)",
			schema: `{
				"type":"record","name":"Event",
				"fields":[
					{"name":"id","type":"int"},
					{"name":"tag",
					 "type":["null","string"],
					 "default":null}
				]
			}`,
		},
		{
			name: "enum",
			schema: `{
				"type":"enum","name":"Status",
				"symbols":["ACTIVE","INACTIVE","UNKNOWN"]
			}`,
		},
		{
			name:   "array of strings",
			schema: `{"type":"array","items":"string"}`,
		},
		{
			name:   "map of ints",
			schema: `{"type":"map","values":"int"}`,
		},
		{
			name:   "union of null and string",
			schema: `["null","string"]`,
		},
		{
			name:   "union of null, int, string",
			schema: `["null","int","string"]`,
		},
		{
			name: "fixed type",
			schema: `{
				"type":"fixed","name":"Uuid","size":16
			}`,
		},
		{
			name: "nested record",
			schema: `{
				"type":"record","name":"Outer",
				"fields":[{
					"name":"inner",
					"type":{
						"type":"record","name":"Inner",
						"fields":[{"name":"v","type":"string"}]
					}
				}]
			}`,
		},

		// ── Invalid ────────────────────────────────────────────────
		{
			name:    "empty input",
			schema:  ``,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			schema:  `{broken`,
			wantErr: true,
		},
		{
			name:    "unknown primitive type",
			schema:  `"uuid"`,
			wantErr: true,
		},
		{
			name:    "record missing name",
			schema:  `{"type":"record","fields":[]}`,
			wantErr: true,
		},
		{
			name:    "record missing fields",
			schema:  `{"type":"record","name":"X"}`,
			wantErr: true,
		},
		{
			name: "record field missing type",
			schema: `{
				"type":"record","name":"X",
				"fields":[{"name":"x"}]
			}`,
			wantErr: true,
		},
		{
			name:    "enum missing symbols",
			schema:  `{"type":"enum","name":"E"}`,
			wantErr: true,
		},
		{
			name:    "enum empty symbols",
			schema:  `{"type":"enum","name":"E","symbols":[]}`,
			wantErr: true,
		},
		{
			name:    "array missing items",
			schema:  `{"type":"array"}`,
			wantErr: true,
		},
		{
			name:    "map missing values",
			schema:  `{"type":"map"}`,
			wantErr: true,
		},
		{
			name:    "fixed missing size",
			schema:  `{"type":"fixed","name":"F"}`,
			wantErr: true,
		},
		{
			name:    "union with duplicate type",
			schema:  `["string","string"]`,
			wantErr: true,
		},
		{
			name:    "empty union",
			schema:  `[]`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := IsValidAvro([]byte(tc.schema))
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ── Primitive type promotion ───────────────────────────────────────

func TestAvroCompat_PrimitivePromotion(t *testing.T) {
	runAvroCases(t, []avroCase{
		{
			name: "same primitive – compat",
			dir:  "backward",
			old:  `"string"`, new: `"string"`,
		},
		{
			name: "int → long – compat (backward)",
			dir:  "backward",
			old:  `"int"`, new: `"long"`,
		},
		{
			name: "int → float – compat (backward)",
			dir:  "backward",
			old:  `"int"`, new: `"float"`,
		},
		{
			name: "int → double – compat (backward)",
			dir:  "backward",
			old:  `"int"`, new: `"double"`,
		},
		{
			name: "long → float – compat (backward)",
			dir:  "backward",
			old:  `"long"`, new: `"float"`,
		},
		{
			name: "long → double – compat (backward)",
			dir:  "backward",
			old:  `"long"`, new: `"double"`,
		},
		{
			name: "float → double – compat (backward)",
			dir:  "backward",
			old:  `"float"`, new: `"double"`,
		},
		{
			name: "string → bytes – compat",
			dir:  "backward",
			old:  `"string"`, new: `"bytes"`,
		},
		{
			name: "bytes → string – compat",
			dir:  "backward",
			old:  `"bytes"`, new: `"string"`,
		},
		{
			name:    "int → string – incompatible",
			dir:     "backward",
			old:     `"int"`, new: `"string"`,
			wantErr: true,
		},
		{
			name:    "double → int – incompatible (no demotion)",
			dir:     "backward",
			old:     `"double"`, new: `"int"`,
			wantErr: true,
		},
		{
			name:    "null → string – incompatible",
			dir:     "backward",
			old:     `"null"`, new: `"string"`,
			wantErr: true,
		},
	})
}

// ── Record: adding / removing fields ──────────────────────────────

func TestAvroCompat_RecordFields(t *testing.T) {
	base := `{
		"type":"record","name":"User",
		"fields":[
			{"name":"id","type":"int"},
			{"name":"name","type":"string"}
		]
	}`

	withDefault := `{
		"type":"record","name":"User",
		"fields":[
			{"name":"id","type":"int"},
			{"name":"name","type":"string"},
			{"name":"email","type":"string","default":""}
		]
	}`

	withoutDefault := `{
		"type":"record","name":"User",
		"fields":[
			{"name":"id","type":"int"},
			{"name":"name","type":"string"},
			{"name":"email","type":"string"}
		]
	}`

	idOnly := `{
		"type":"record","name":"User",
		"fields":[{"name":"id","type":"int"}]
	}`

	runAvroCases(t, []avroCase{
		{
			name: "identical records – compat",
			dir:  "backward", old: base, new: base,
		},
		{
			// backward: new reader adds field with default → compat
			// (old writer data lacks it; new reader uses default)
			name: "add field with default – compat (backward)",
			dir:  "backward", old: base, new: withDefault,
		},
		{
			// backward: new reader adds field without default → NOT
			// compat (old writer data lacks it; no fallback)
			name:    "add field without default – incompatible (backward)",
			dir:     "backward", old: base, new: withoutDefault,
			wantErr: true,
		},
		{
			// backward: new reader removes "name" → compat
			// (old writer data has it; reader ignores extra fields)
			name: "remove field – compat (backward)",
			dir:  "backward", old: base, new: idOnly,
		},
		{
			// forward = checkAvroBackward(new=writer, old=reader)
			// new writer adds "email" with no default; old reader
			// sees it as extra writer field → ignores → compat
			name: "add field – compat (forward)",
			dir:  "forward", old: base, new: withoutDefault,
		},
		{
			// forward: new writer removes "name" which old reader has
			// but without a default → old reader can't fill gap
			name:    "remove field without reader default – incompatible (forward)",
			dir:     "forward", old: base, new: idOnly,
			wantErr: true,
		},
		{
			// forward: new writer removes "email" that old reader
			// carries with a default → compat
			name: "remove field that has default in old – compat (forward)",
			dir:  "forward", old: withDefault, new: base,
		},
	})
}

// ── Record: field type changes ─────────────────────────────────────

func TestAvroCompat_RecordFieldTypes(t *testing.T) {
	runAvroCases(t, []avroCase{
		{
			name: "widen int→long in field – compat (backward)",
			dir:  "backward",
			old: `{"type":"record","name":"R",
				"fields":[{"name":"x","type":"int"}]}`,
			new: `{"type":"record","name":"R",
				"fields":[{"name":"x","type":"long"}]}`,
		},
		{
			name:    "narrow long→int in field – incompatible",
			dir:     "backward",
			old:     `{"type":"record","name":"R","fields":[{"name":"x","type":"long"}]}`,
			new:     `{"type":"record","name":"R","fields":[{"name":"x","type":"int"}]}`,
			wantErr: true,
		},
		{
			name:    "incompatible string→int – incompatible",
			dir:     "backward",
			old:     `{"type":"record","name":"R","fields":[{"name":"x","type":"string"}]}`,
			new:     `{"type":"record","name":"R","fields":[{"name":"x","type":"int"}]}`,
			wantErr: true,
		},
	})
}

// ── Record: name change ────────────────────────────────────────────

func TestAvroCompat_RecordName(t *testing.T) {
	runAvroCases(t, []avroCase{
		{
			name:    "record name changed – incompatible",
			dir:     "backward",
			old:     `{"type":"record","name":"Old","fields":[]}`,
			new:     `{"type":"record","name":"New","fields":[]}`,
			wantErr: true,
		},
	})
}

// ── Enum ───────────────────────────────────────────────────────────

func TestAvroCompat_Enum(t *testing.T) {
	runAvroCases(t, []avroCase{
		{
			name: "same enum – compat",
			dir:  "backward",
			old:  `{"type":"enum","name":"E","symbols":["A","B","C"]}`,
			new:  `{"type":"enum","name":"E","symbols":["A","B","C"]}`,
		},
		{
			// backward: new reader adds a symbol → compat
			// (old writer won't produce it, reader handles it)
			name: "add symbol to reader – compat (backward)",
			dir:  "backward",
			old:  `{"type":"enum","name":"E","symbols":["A","B"]}`,
			new:  `{"type":"enum","name":"E","symbols":["A","B","C"]}`,
		},
		{
			// backward: new reader removes symbol "C" that old writer
			// may produce and reader has no default → NOT compat
			name:    "remove symbol – incompatible (backward)",
			dir:     "backward",
			old:     `{"type":"enum","name":"E","symbols":["A","B","C"]}`,
			new:     `{"type":"enum","name":"E","symbols":["A","B"]}`,
			wantErr: true,
		},
		{
			// backward: reader removes symbol but has enum default
			// → unknown values fall back to default → compat
			name: "remove symbol with enum default – compat (backward)",
			dir:  "backward",
			old:  `{"type":"enum","name":"E","symbols":["A","B","C"]}`,
			new:  `{"type":"enum","name":"E","symbols":["A","B"],"default":"A"}`,
		},
		{
			// forward = checkAvroBackward(new=writer, old=reader)
			// new writer can produce "C"; old reader doesn't know
			// "C" and has no default → NOT compat
			name:    "add symbol – incompatible (forward, no default)",
			dir:     "forward",
			old:     `{"type":"enum","name":"E","symbols":["A","B"]}`,
			new:     `{"type":"enum","name":"E","symbols":["A","B","C"]}`,
			wantErr: true,
		},
		{
			// forward: new writer removes "C"; old reader never
			// sees an unknown value → compat
			name: "remove symbol – compat (forward)",
			dir:  "forward",
			old:  `{"type":"enum","name":"E","symbols":["A","B","C"]}`,
			new:  `{"type":"enum","name":"E","symbols":["A","B"]}`,
		},
		{
			name:    "enum name changed – incompatible",
			dir:     "backward",
			old:     `{"type":"enum","name":"Old","symbols":["A"]}`,
			new:     `{"type":"enum","name":"New","symbols":["A"]}`,
			wantErr: true,
		},
	})
}

// ── Array ──────────────────────────────────────────────────────────

func TestAvroCompat_Array(t *testing.T) {
	runAvroCases(t, []avroCase{
		{
			name: "same array – compat",
			dir:  "backward",
			old:  `{"type":"array","items":"string"}`,
			new:  `{"type":"array","items":"string"}`,
		},
		{
			name: "array items int→long – compat (backward)",
			dir:  "backward",
			old:  `{"type":"array","items":"int"}`,
			new:  `{"type":"array","items":"long"}`,
		},
		{
			name:    "array items string→int – incompatible",
			dir:     "backward",
			old:     `{"type":"array","items":"string"}`,
			new:     `{"type":"array","items":"int"}`,
			wantErr: true,
		},
		{
			// forward: new array writer uses "long", old reader
			// expects "int" → long can't be read as int → NOT compat
			name:    "array items widen – incompatible (forward)",
			dir:     "forward",
			old:     `{"type":"array","items":"int"}`,
			new:     `{"type":"array","items":"long"}`,
			wantErr: true,
		},
	})
}

// ── Map ────────────────────────────────────────────────────────────

func TestAvroCompat_Map(t *testing.T) {
	runAvroCases(t, []avroCase{
		{
			name: "same map – compat",
			dir:  "backward",
			old:  `{"type":"map","values":"string"}`,
			new:  `{"type":"map","values":"string"}`,
		},
		{
			name: "map values int→long – compat (backward)",
			dir:  "backward",
			old:  `{"type":"map","values":"int"}`,
			new:  `{"type":"map","values":"long"}`,
		},
		{
			name:    "map values string→int – incompatible",
			dir:     "backward",
			old:     `{"type":"map","values":"string"}`,
			new:     `{"type":"map","values":"int"}`,
			wantErr: true,
		},
	})
}

// ── Union ──────────────────────────────────────────────────────────

func TestAvroCompat_Union(t *testing.T) {
	runAvroCases(t, []avroCase{
		{
			name: "same union – compat",
			dir:  "backward",
			old:  `["null","string"]`,
			new:  `["null","string"]`,
		},
		{
			// backward: new reader gains an extra branch → compat
			// (old writer never produces "int"; reader handles all
			// old values)
			name: "add branch to reader – compat (backward)",
			dir:  "backward",
			old:  `["null","string"]`,
			new:  `["null","string","int"]`,
		},
		{
			// backward: old writer can produce "int"; new reader
			// has no "int" branch → NOT compat
			name:    "remove branch from reader – incompatible (backward)",
			dir:     "backward",
			old:     `["null","string","int"]`,
			new:     `["null","string"]`,
			wantErr: true,
		},
		{
			// writer is non-union "string"; reader union has "string"
			// branch → compat
			name: "non-union writer matched by reader union – compat",
			dir:  "backward",
			old:  `"string"`,
			new:  `["null","string"]`,
		},
		{
			// writer is non-union "int"; reader union has no "int"
			name:    "non-union writer not in reader union – incompatible",
			dir:     "backward",
			old:     `"int"`,
			new:     `["null","string"]`,
			wantErr: true,
		},
		{
			// forward: new writer can produce "int"; old reader union
			// has no "int" branch → NOT compat
			name:    "add branch to writer – incompatible (forward)",
			dir:     "forward",
			old:     `["null","string"]`,
			new:     `["null","string","int"]`,
			wantErr: true,
		},
		{
			// forward: new writer drops "int"; old reader still has
			// it but never sees unknown values → compat
			name: "remove branch from writer – compat (forward)",
			dir:  "forward",
			old:  `["null","string","int"]`,
			new:  `["null","string"]`,
		},
	})
}

// ── Fixed ──────────────────────────────────────────────────────────

func TestAvroCompat_Fixed(t *testing.T) {
	runAvroCases(t, []avroCase{
		{
			name: "same fixed – compat",
			dir:  "backward",
			old:  `{"type":"fixed","name":"Hash","size":16}`,
			new:  `{"type":"fixed","name":"Hash","size":16}`,
		},
		{
			name:    "fixed size changed – incompatible",
			dir:     "backward",
			old:     `{"type":"fixed","name":"Hash","size":16}`,
			new:     `{"type":"fixed","name":"Hash","size":32}`,
			wantErr: true,
		},
		{
			name:    "fixed name changed – incompatible",
			dir:     "backward",
			old:     `{"type":"fixed","name":"A","size":8}`,
			new:     `{"type":"fixed","name":"B","size":8}`,
			wantErr: true,
		},
	})
}

// ── Nested / complex schemas ───────────────────────────────────────

func TestAvroCompat_Nested(t *testing.T) {
	outerV1 := `{
		"type":"record","name":"Outer",
		"fields":[{
			"name":"inner",
			"type":{
				"type":"record","name":"Inner",
				"fields":[{"name":"val","type":"string"}]
			}
		}]
	}`

	outerAddFieldWithDefault := `{
		"type":"record","name":"Outer",
		"fields":[{
			"name":"inner",
			"type":{
				"type":"record","name":"Inner",
				"fields":[
					{"name":"val","type":"string"},
					{"name":"num","type":"int","default":0}
				]
			}
		}]
	}`

	outerAddFieldNoDefault := `{
		"type":"record","name":"Outer",
		"fields":[{
			"name":"inner",
			"type":{
				"type":"record","name":"Inner",
				"fields":[
					{"name":"val","type":"string"},
					{"name":"num","type":"int"}
				]
			}
		}]
	}`

	runAvroCases(t, []avroCase{
		{
			name: "nested record add field with default – compat",
			dir:  "backward",
			old:  outerV1, new: outerAddFieldWithDefault,
		},
		{
			name:    "nested record add field without default – incompatible",
			dir:     "backward",
			old:     outerV1, new: outerAddFieldNoDefault,
			wantErr: true,
		},
	})
}

// ── Forward direction (summary cases) ─────────────────────────────

func TestAvroCompat_ForwardDirection(t *testing.T) {
	runAvroCases(t, []avroCase{
		{
			// forward = checkAvroBackward(new=writer, old=reader)
			// new writes "long"; old reader expects "int" → NOT compat
			name:    "type widening – incompatible (forward)",
			dir:     "forward",
			old:     `"int"`, new: `"long"`,
			wantErr: true,
		},
		{
			name: "same type – compat (forward)",
			dir:  "forward",
			old:  `"int"`, new: `"int"`,
		},
		{
			// forward: new removes union branch it would write;
			// old reader still handles remaining branches → compat
			name: "remove union branch from writer – compat (forward)",
			dir:  "forward",
			old:  `["null","string","int"]`, new: `["null","string"]`,
		},
	})
}
