package registry

// Unit tests for IsValidJsonStructure and checkJSCompat, covering the
// features listed at the top of format_jsonstructure.go.
//
// Tests are run with: make utest

import (
	"testing"
)

// ────────────────────────────────────────────────────────────────
// IsValidJsonStructure
// ────────────────────────────────────────────────────────────────

func TestIsValidJsonStructure(t *testing.T) {
	tests := []struct {
		name    string
		doc     string
		wantErr bool
	}{
		// ── Valid documents ────────────────────────────────────────
		{
			name: "minimal inline object",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/Person",
				"name":"Person",
				"type":"object",
				"properties":{"name":{"type":"string"}}
			}`,
		},
		{
			name: "with $root and definitions",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/Person",
				"name":"Person",
				"$root":"#/definitions/Person",
				"definitions":{
					"Person":{
						"type":"object",
						"properties":{"name":{"type":"string"}}
					}
				}
			}`,
		},
		{
			name: "namespaced definitions",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/Person",
				"name":"Person",
				"$root":"#/definitions/NS/Person",
				"definitions":{
					"NS":{
						"Person":{
							"type":"object",
							"properties":{"name":{"type":"string"}}
						}
					}
				}
			}`,
		},
		{
			name: "primitive types",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/T",
				"name":"T",
				"type":"object",
				"properties":{
					"a":{"type":"string"},
					"b":{"type":"number"},
					"c":{"type":"integer"},
					"d":{"type":"boolean"},
					"e":{"type":"null"},
					"f":{"type":"int8"},
					"g":{"type":"uint64"},
					"h":{"type":"float"},
					"i":{"type":"double"},
					"j":{"type":"decimal","precision":10,"scale":2},
					"k":{"type":"date"},
					"l":{"type":"datetime"},
					"m":{"type":"time"},
					"n":{"type":"duration"},
					"o":{"type":"uuid"},
					"p":{"type":"uri"},
					"q":{"type":"jsonpointer"},
					"r":{"type":"binary","contentEncoding":"base64"}
				}
			}`,
		},
		{
			name: "array of string",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/T",
				"name":"T",
				"type":"array",
				"items":{"type":"string"}
			}`,
		},
		{
			name: "set of string",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/T",
				"name":"T",
				"type":"set",
				"items":{"type":"string"}
			}`,
		},
		{
			name: "map of string",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/T",
				"name":"T",
				"type":"map",
				"values":{"type":"string"}
			}`,
		},
		{
			name: "tuple",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/T",
				"name":"T",
				"type":"tuple",
				"properties":{
					"name":{"type":"string"},
					"age":{"type":"int32"}
				},
				"tuple":["name","age"]
			}`,
		},
		{
			name: "any",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/T",
				"name":"T",
				"type":"any"
			}`,
		},
		{
			name: "tagged choice",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/T",
				"name":"T",
				"type":"choice",
				"choices":{
					"string":{"type":"string"},
					"int32":{"type":"int32"}
				}
			}`,
		},
		{
			name: "inline union choice with $extends+selector",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/T",
				"name":"T",
				"$root":"#/definitions/MyChoice",
				"definitions":{
					"Address":{
						"abstract":true,
						"type":"object",
						"properties":{"city":{"type":"string"}}
					},
					"StreetAddress":{
						"type":"object",
						"$extends":"#/definitions/Address",
						"properties":{"street":{"type":"string"}}
					},
					"MyChoice":{
						"type":"choice",
						"$extends":"#/definitions/Address",
						"selector":"addressType",
						"choices":{
							"StreetAddress":{
								"type":{"$ref":"#/definitions/StreetAddress"}
							}
						}
					}
				}
			}`,
		},
		{
			name: "type union of string and int32",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/T",
				"name":"T",
				"type":"object",
				"properties":{
					"val":{"type":["string","int32"]}
				}
			}`,
		},
		{
			name: "$extends abstract base",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/T",
				"name":"T",
				"$root":"#/definitions/StreetAddress",
				"definitions":{
					"Address":{
						"abstract":true,
						"type":"object",
						"properties":{"city":{"type":"string"}}
					},
					"StreetAddress":{
						"type":"object",
						"$extends":"#/definitions/Address",
						"properties":{"street":{"type":"string"}}
					}
				}
			}`,
		},
		{
			name: "required + additionalProperties false",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/T",
				"name":"T",
				"type":"object",
				"properties":{"a":{"type":"string"}},
				"required":["a"],
				"additionalProperties":false
			}`,
		},
		{
			name: "required alternative sets",
			doc: `{
				"$schema":"https://json-structure.org/meta/core/v0/#",
				"$id":"https://example.com/T",
				"name":"T",
				"type":"object",
				"properties":{
					"name":{"type":"string"},
					"fins":{"type":"int32"},
					"legs":{"type":"int32"}
				},
				"required":[["name","fins"],["name","legs"]]
			}`,
		},

		// ── Invalid documents ──────────────────────────────────────
		{
			name:    "not json",
			doc:     `{not json`,
			wantErr: true,
		},
		{
			name:    "missing $schema",
			doc:     `{"$id":"x","name":"T","type":"any"}`,
			wantErr: true,
		},
		{
			name:    "missing $id",
			doc:     `{"$schema":"x","name":"T","type":"any"}`,
			wantErr: true,
		},
		{
			name:    "missing name",
			doc:     `{"$schema":"x","$id":"y","type":"any"}`,
			wantErr: true,
		},
		{
			name: "type and $root both present",
			doc: `{
				"$schema":"x","$id":"y","name":"T",
				"type":"any",
				"$root":"#/definitions/T"
			}`,
			wantErr: true,
		},
		{
			name:    "neither type nor $root",
			doc:     `{"$schema":"x","$id":"y","name":"T"}`,
			wantErr: true,
		},
		{
			name:    "object missing properties",
			doc:     `{"$schema":"x","$id":"y","name":"T","type":"object"}`,
			wantErr: true,
		},
		{
			name:    "array missing items",
			doc:     `{"$schema":"x","$id":"y","name":"T","type":"array"}`,
			wantErr: true,
		},
		{
			name:    "map missing values",
			doc:     `{"$schema":"x","$id":"y","name":"T","type":"map"}`,
			wantErr: true,
		},
		{
			name: "tuple missing tuple keyword",
			doc: `{
				"$schema":"x","$id":"y","name":"T","type":"tuple",
				"properties":{"a":{"type":"string"}}
			}`,
			wantErr: true,
		},
		{
			name: "tuple length mismatch",
			doc: `{
				"$schema":"x","$id":"y","name":"T","type":"tuple",
				"properties":{"a":{"type":"string"},"b":{"type":"int32"}},
				"tuple":["a"]
			}`,
			wantErr: true,
		},
		{
			name:    "choice missing choices",
			doc:     `{"$schema":"x","$id":"y","name":"T","type":"choice"}`,
			wantErr: true,
		},
		{
			name:    "unknown type",
			doc:     `{"$schema":"x","$id":"y","name":"T","type":"bogus"}`,
			wantErr: true,
		},
		{
			name: "required references unknown property",
			doc: `{
				"$schema":"x","$id":"y","name":"T","type":"object",
				"properties":{"a":{"type":"string"}},
				"required":["b"]
			}`,
			wantErr: true,
		},
		{
			name: "$ref does not resolve",
			doc: `{
				"$schema":"x","$id":"y","name":"T","type":"object",
				"properties":{
					"a":{"type":{"$ref":"#/definitions/Missing"}}
				}
			}`,
			wantErr: true,
		},
		{
			name: "$extends target not abstract",
			doc: `{
				"$schema":"x","$id":"y","name":"T",
				"$root":"#/definitions/Sub",
				"definitions":{
					"Base":{
						"type":"object",
						"properties":{"a":{"type":"string"}}
					},
					"Sub":{
						"type":"object",
						"$extends":"#/definitions/Base",
						"properties":{"b":{"type":"string"}}
					}
				}
			}`,
			wantErr: true,
		},
		{
			name: "union with inline compound type not permitted",
			doc: `{
				"$schema":"x","$id":"y","name":"T","type":"object",
				"properties":{
					"a":{"type":["string",{"type":"object"}]}
				}
			}`,
			wantErr: true,
		},
		{
			name: "invalid property identifier",
			doc: `{
				"$schema":"x","$id":"y","name":"T","type":"object",
				"properties":{"1bad":{"type":"string"}}
			}`,
			wantErr: true,
		},
		{
			name: "maxLength on non-string/binary",
			doc: `{
				"$schema":"x","$id":"y","name":"T","type":"object",
				"properties":{"a":{"type":"int32","maxLength":5}}
			}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := IsValidJsonStructure([]byte(tc.doc))
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ────────────────────────────────────────────────────────────────
// checkJSCompat
// ────────────────────────────────────────────────────────────────

type jsCompatCase struct {
	name    string
	dir     string // "backward" or "forward"
	old     string
	new     string
	wantErr bool
}

func runJSCompatCases(t *testing.T, cases []jsCompatCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oldDoc, err := parseJSDoc([]byte(tc.old))
			if err != nil {
				t.Fatalf("parseJSDoc(old): %v", err)
			}
			newDoc, err := parseJSDoc([]byte(tc.new))
			if err != nil {
				t.Fatalf("parseJSDoc(new): %v", err)
			}
			err = checkJSCompat(tc.dir, oldDoc, newDoc)
			if tc.wantErr && err == nil {
				t.Errorf("expected incompatibility, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected incompatibility: %v", err)
			}
		})
	}
}

func TestJSCompat_Object(t *testing.T) {
	runJSCompatCases(t, []jsCompatCase{
		{
			name: "delete optional field – compat",
			dir:  "backward",
			old: `{"type":"object","properties":{
				"a":{"type":"string"},"b":{"type":"string"}}}`,
			new: `{"type":"object","properties":{"a":{"type":"string"}}}`,
		},
		{
			name: "add optional field – compat",
			dir:  "backward",
			old:  `{"type":"object","properties":{"a":{"type":"string"}}}`,
			new: `{"type":"object","properties":{
				"a":{"type":"string"},"b":{"type":"string"}}}`,
		},
		{
			name: "add required field – incompatible",
			dir:  "backward",
			old:  `{"type":"object","properties":{"a":{"type":"string"}}}`,
			new: `{"type":"object","properties":{
				"a":{"type":"string"},"b":{"type":"string"}},
				"required":["b"]}`,
			wantErr: true,
		},
		{
			name: "delete required field – compat",
			dir:  "backward",
			old: `{"type":"object","properties":{
				"a":{"type":"string"},"b":{"type":"string"}},
				"required":["a","b"]}`,
			new: `{"type":"object","properties":{
				"a":{"type":"string"}},"required":["a"]}`,
		},
		{
			name: "removed property blocked by additionalProperties:false",
			dir:  "backward",
			old: `{"type":"object","properties":{
				"a":{"type":"string"},"b":{"type":"string"}}}`,
			new: `{"type":"object","properties":{"a":{"type":"string"}},
				"additionalProperties":false}`,
			wantErr: true,
		},
		{
			name: "alternative required sets identical – compat",
			dir:  "backward",
			old: `{"type":"object","properties":{
				"name":{"type":"string"},"fins":{"type":"int32"},
				"legs":{"type":"int32"}},
				"required":[["name","fins"],["name","legs"]]}`,
			new: `{"type":"object","properties":{
				"name":{"type":"string"},"fins":{"type":"int32"},
				"legs":{"type":"int32"}},
				"required":[["name","fins"],["name","legs"]]}`,
		},
		{
			name: "alternative required sets changed – incompatible",
			dir:  "backward",
			old: `{"type":"object","properties":{
				"name":{"type":"string"},"fins":{"type":"int32"},
				"legs":{"type":"int32"}},
				"required":[["name","fins"],["name","legs"]]}`,
			new: `{"type":"object","properties":{
				"name":{"type":"string"},"fins":{"type":"int32"},
				"legs":{"type":"int32"}},
				"required":[["name","fins"]]}`,
			wantErr: true,
		},
	})
}

func TestJSCompat_Primitives(t *testing.T) {
	runJSCompatCases(t, []jsCompatCase{
		{
			name: "int16 widens to int32 – compat",
			dir:  "backward",
			old:  `{"type":"int16"}`,
			new:  `{"type":"int32"}`,
		},
		{
			name:    "int32 narrows to int16 – incompatible",
			dir:     "backward",
			old:     `{"type":"int32"}`,
			new:     `{"type":"int16"}`,
			wantErr: true,
		},
		{
			name: "uint8 widens to uint64 – compat",
			dir:  "backward",
			old:  `{"type":"uint8"}`,
			new:  `{"type":"uint64"}`,
		},
		{
			name: "float widens to double – compat",
			dir:  "backward",
			old:  `{"type":"float"}`,
			new:  `{"type":"double"}`,
		},
		{
			name:    "int32 to float – incompatible (no cross-family)",
			dir:     "backward",
			old:     `{"type":"int32"}`,
			new:     `{"type":"float"}`,
			wantErr: true,
		},
		{
			name: "int32 widens to number – compat",
			dir:  "backward",
			old:  `{"type":"int32"}`,
			new:  `{"type":"number"}`,
		},
		{
			name:    "number narrows to int32 – incompatible",
			dir:     "backward",
			old:     `{"type":"number"}`,
			new:     `{"type":"int32"}`,
			wantErr: true,
		},
		{
			name: "integer alias treated as int32",
			dir:  "backward",
			old:  `{"type":"integer"}`,
			new:  `{"type":"int64"}`,
		},
		{
			name: "same exact primitive – compat",
			dir:  "backward",
			old:  `{"type":"uuid"}`,
			new:  `{"type":"uuid"}`,
		},
		{
			name:    "uuid to uri – incompatible",
			dir:     "backward",
			old:     `{"type":"uuid"}`,
			new:     `{"type":"uri"}`,
			wantErr: true,
		},
		{
			name: "string maxLength grows – compat",
			dir:  "backward",
			old:  `{"type":"string","maxLength":10}`,
			new:  `{"type":"string","maxLength":20}`,
		},
		{
			name:    "string maxLength shrinks – incompatible",
			dir:     "backward",
			old:     `{"type":"string","maxLength":20}`,
			new:     `{"type":"string","maxLength":10}`,
			wantErr: true,
		},
		{
			name: "string maxLength removed (unrestricted) – compat",
			dir:  "backward",
			old:  `{"type":"string","maxLength":10}`,
			new:  `{"type":"string"}`,
		},
		{
			name:    "string adds maxLength – incompatible",
			dir:     "backward",
			old:     `{"type":"string"}`,
			new:     `{"type":"string","maxLength":10}`,
			wantErr: true,
		},
		{
			name: "decimal precision/scale grow – compat",
			dir:  "backward",
			old:  `{"type":"decimal","precision":5,"scale":1}`,
			new:  `{"type":"decimal","precision":10,"scale":2}`,
		},
		{
			name:    "decimal precision shrinks – incompatible",
			dir:     "backward",
			old:     `{"type":"decimal","precision":10}`,
			new:     `{"type":"decimal","precision":5}`,
			wantErr: true,
		},
		{
			name: "enum: new adds values – compat",
			dir:  "backward",
			old:  `{"type":"string","enum":["a","b"]}`,
			new:  `{"type":"string","enum":["a","b","c"]}`,
		},
		{
			name:    "enum: new removes value – incompatible",
			dir:     "backward",
			old:     `{"type":"string","enum":["a","b"]}`,
			new:     `{"type":"string","enum":["a"]}`,
			wantErr: true,
		},
		{
			name: "const unchanged – compat",
			dir:  "backward",
			old:  `{"type":"string","const":"fixed"}`,
			new:  `{"type":"string","const":"fixed"}`,
		},
		{
			name:    "const changed – incompatible",
			dir:     "backward",
			old:     `{"type":"string","const":"a"}`,
			new:     `{"type":"string","const":"b"}`,
			wantErr: true,
		},
	})
}

func TestJSCompat_ArraySetMap(t *testing.T) {
	runJSCompatCases(t, []jsCompatCase{
		{
			name: "array items widen – compat",
			dir:  "backward",
			old:  `{"type":"array","items":{"type":"int16"}}`,
			new:  `{"type":"array","items":{"type":"int32"}}`,
		},
		{
			name:    "array items narrow – incompatible",
			dir:     "backward",
			old:     `{"type":"array","items":{"type":"int32"}}`,
			new:     `{"type":"array","items":{"type":"int16"}}`,
			wantErr: true,
		},
		{
			name: "set items widen – compat",
			dir:  "backward",
			old:  `{"type":"set","items":{"type":"int16"}}`,
			new:  `{"type":"set","items":{"type":"int32"}}`,
		},
		{
			name: "map values widen – compat",
			dir:  "backward",
			old:  `{"type":"map","values":{"type":"int16"}}`,
			new:  `{"type":"map","values":{"type":"int32"}}`,
		},
		{
			name:    "map values narrow – incompatible",
			dir:     "backward",
			old:     `{"type":"map","values":{"type":"int32"}}`,
			new:     `{"type":"map","values":{"type":"int16"}}`,
			wantErr: true,
		},
	})
}

func TestJSCompat_Tuple(t *testing.T) {
	runJSCompatCases(t, []jsCompatCase{
		{
			name: "same-length tuple, position types widen – compat",
			dir:  "backward",
			old: `{"type":"tuple","properties":{
				"a":{"type":"string"},"b":{"type":"int16"}},
				"tuple":["a","b"]}`,
			new: `{"type":"tuple","properties":{
				"a":{"type":"string"},"b":{"type":"int32"}},
				"tuple":["a","b"]}`,
		},
		{
			name: "tuple length changed – incompatible",
			dir:  "backward",
			old: `{"type":"tuple","properties":{
				"a":{"type":"string"}},"tuple":["a"]}`,
			new: `{"type":"tuple","properties":{
				"a":{"type":"string"},"b":{"type":"int32"}},
				"tuple":["a","b"]}`,
			wantErr: true,
		},
	})
}

func TestJSCompat_Choice(t *testing.T) {
	runJSCompatCases(t, []jsCompatCase{
		{
			name: "new adds a choice – compat",
			dir:  "backward",
			old: `{"type":"choice","choices":{
				"string":{"type":"string"}}}`,
			new: `{"type":"choice","choices":{
				"string":{"type":"string"},"int32":{"type":"int32"}}}`,
		},
		{
			name: "new removes a choice – incompatible",
			dir:  "backward",
			old: `{"type":"choice","choices":{
				"string":{"type":"string"},"int32":{"type":"int32"}}}`,
			new: `{"type":"choice","choices":{
				"string":{"type":"string"}}}`,
			wantErr: true,
		},
		{
			name: "choice member narrows – incompatible",
			dir:  "backward",
			old: `{"type":"choice","choices":{
				"n":{"type":"int32"}}}`,
			new: `{"type":"choice","choices":{
				"n":{"type":"int16"}}}`,
			wantErr: true,
		},
	})
}

func TestJSCompat_Any(t *testing.T) {
	runJSCompatCases(t, []jsCompatCase{
		{
			name: "new is any – always compat",
			dir:  "backward",
			old:  `{"type":"object","properties":{"a":{"type":"string"}}}`,
			new:  `{"type":"any"}`,
		},
		{
			name:    "old is any, new restricts – incompatible",
			dir:     "backward",
			old:     `{"type":"any"}`,
			new:     `{"type":"object","properties":{"a":{"type":"string"}}}`,
			wantErr: true,
		},
	})
}

func TestJSCompat_Union(t *testing.T) {
	runJSCompatCases(t, []jsCompatCase{
		{
			name: "old union: every branch must be compat with new",
			dir:  "backward",
			old: `{"type":"object","properties":{
				"v":{"type":["int16","int32"]}}}`,
			new: `{"type":"object","properties":{
				"v":{"type":"int32"}}}`,
		},
		{
			name: "old union branch incompatible with new – incompatible",
			dir:  "backward",
			old: `{"type":"object","properties":{
				"v":{"type":["int32","string"]}}}`,
			new: `{"type":"object","properties":{
				"v":{"type":"int32"}}}`,
			wantErr: true,
		},
		{
			name: "new union: old must match at least one branch – compat",
			dir:  "backward",
			old: `{"type":"object","properties":{
				"v":{"type":"int16"}}}`,
			new: `{"type":"object","properties":{
				"v":{"type":["string","int32"]}}}`,
		},
		{
			name: "new union: old matches no branch – incompatible",
			dir:  "backward",
			old: `{"type":"object","properties":{
				"v":{"type":"uuid"}}}`,
			new: `{"type":"object","properties":{
				"v":{"type":["string","int32"]}}}`,
			wantErr: true,
		},
	})
}

func TestJSCompat_Extends(t *testing.T) {
	runJSCompatCases(t, []jsCompatCase{
		{
			name: "$extends flattens base properties – compat when " +
				"subtype adds optional field",
			dir: "backward",
			old: `{
				"$root":"#/definitions/Sub",
				"definitions":{
					"Base":{"abstract":true,"type":"object",
						"properties":{"city":{"type":"string"}}},
					"Sub":{"type":"object",
						"$extends":"#/definitions/Base",
						"properties":{"street":{"type":"string"}}}
				}
			}`,
			new: `{
				"$root":"#/definitions/Sub",
				"definitions":{
					"Base":{"abstract":true,"type":"object",
						"properties":{"city":{"type":"string"},
							"state":{"type":"string"}}},
					"Sub":{"type":"object",
						"$extends":"#/definitions/Base",
						"properties":{"street":{"type":"string"}}}
				}
			}`,
		},
		{
			name: "$extends base loses a property – incompatible " +
				"(base required it)",
			dir: "backward",
			old: `{
				"$root":"#/definitions/Sub",
				"definitions":{
					"Base":{"abstract":true,"type":"object",
						"properties":{"city":{"type":"string"}},
						"required":["city"]},
					"Sub":{"type":"object",
						"$extends":"#/definitions/Base",
						"properties":{"street":{"type":"string"}}}
				}
			}`,
			new: `{
				"$root":"#/definitions/Sub",
				"definitions":{
					"Base":{"abstract":true,"type":"object",
						"properties":{"other":{"type":"string"}},
						"required":["other"]},
					"Sub":{"type":"object",
						"$extends":"#/definitions/Base",
						"properties":{"street":{"type":"string"}}}
				}
			}`,
			wantErr: true,
		},
	})
}

func TestJSCompat_Ref(t *testing.T) {
	runJSCompatCases(t, []jsCompatCase{
		{
			name: "$ref to a compatible definition – compat",
			dir:  "backward",
			old: `{
				"type":"object",
				"properties":{"addr":{"type":{"$ref":"#/definitions/Addr"}}},
				"definitions":{
					"Addr":{"type":"object",
						"properties":{"city":{"type":"string"}}}
				}
			}`,
			new: `{
				"type":"object",
				"properties":{"addr":{"type":{"$ref":"#/definitions/Addr"}}},
				"definitions":{
					"Addr":{"type":"object",
						"properties":{"city":{"type":"string"},
							"zip":{"type":"string"}}}
				}
			}`,
		},
		{
			name: "$ref to an incompatible definition – incompatible",
			dir:  "backward",
			old: `{
				"type":"object",
				"properties":{"addr":{"type":{"$ref":"#/definitions/Addr"}}},
				"definitions":{
					"Addr":{"type":"object",
						"properties":{"city":{"type":"string"}}}
				}
			}`,
			new: `{
				"type":"object",
				"properties":{"addr":{"type":{"$ref":"#/definitions/Addr"}}},
				"definitions":{
					"Addr":{"type":"object",
						"properties":{"city":{"type":"string"}},
						"required":["city"]}
				}
			}`,
			wantErr: true,
		},
	})
}

func TestJSCompat_ForwardDirection(t *testing.T) {
	runJSCompatCases(t, []jsCompatCase{
		{
			name: "forward: add optional field – compat",
			dir:  "forward",
			old:  `{"type":"object","properties":{"a":{"type":"string"}}}`,
			new: `{"type":"object","properties":{
				"a":{"type":"string"},"b":{"type":"string"}}}`,
		},
		{
			name: "forward: delete required field – incompatible",
			dir:  "forward",
			old: `{"type":"object","properties":{
				"a":{"type":"string"},"b":{"type":"string"}},
				"required":["a","b"]}`,
			new: `{"type":"object","properties":{
				"a":{"type":"string"}},"required":["a"]}`,
			wantErr: true,
		},
		{
			name: "forward: narrow primitive – compat",
			dir:  "forward",
			old:  `{"type":"int32"}`,
			new:  `{"type":"int16"}`,
		},
		{
			name:    "forward: widen primitive – incompatible",
			dir:     "forward",
			old:     `{"type":"int16"}`,
			new:     `{"type":"int32"}`,
			wantErr: true,
		},
	})
}
