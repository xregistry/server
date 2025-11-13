package registry

import (
	"fmt"
	"testing"

	. "github.com/xregistry/server/common"
)

func TestModelVerifySimple(t *testing.T) {
	type Test struct {
		name  string
		model Model
		err   string
	}

	tests := []Test{
		{"empty model", Model{}, ""},
		{"empty model - 2", Model{
			Attributes: map[string]*Attribute{},
			Groups:     map[string]*GroupModel{},
		}, ""},

		{"reg 1 attr - full", Model{
			Attributes: Attributes{
				"myint": &Attribute{
					Name:        "myint",
					Type:        "integer",
					Description: "cool int",
					Enum:        []any{1},
					Strict:      PtrBool(true),
					ReadOnly:    true,
					Required:    true,

					IfValues: IfValues{},
				},
			},
		}, ""},
		{"reg 1 group -1 ", Model{
			Groups: map[string]*GroupModel{
				"gs1": &GroupModel{
					Plural:   "gs1",
					Singular: "g1",
				},
			},
		}, ""},
		{"reg 1 group -2 ", Model{
			Groups: map[string]*GroupModel{"Gs1": nil},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: Group definition for \"Gs1\" can't be empty"
}`},
		{"reg 1 group -3 ", Model{
			Groups: map[string]*GroupModel{"Gs1": {}},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: model type name \"Gs1\" must match: ^[a-z_][a-z_0-9]{0,56}$"
}`},
		{"reg 1 group -4 ", Model{
			Groups: map[string]*GroupModel{"@": {}},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: model type name \"@\" must match: ^[a-z_][a-z_0-9]{0,56}$"
}`},
		{"reg 1 group -4.5 ", Model{
			Groups: map[string]*GroupModel{"a": {}},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: Group \"a\" is missing a \"singular\" value"
}`},
		{"reg 1 group -5 ", Model{
			Groups: map[string]*GroupModel{"_a": {Plural: "_a", Singular: "a"}},
		}, ``},
		{"reg 1 group -5.5 ", Model{
			Groups: map[string]*GroupModel{"_a": {Singular: "a"}},
		}, ``},
		{"reg 1 group -5.6 ", Model{
			Groups: map[string]*GroupModel{"a": {Singular: "a"}},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: Group \"a\" has same value for \"plural\" and \"singular\""
}`},
		{"reg 1 group -6 ", Model{
			Groups: map[string]*GroupModel{"a23456789012345678901234567890123456789012345678901234567": {Plural: "a23456789012345678901234567890123456789012345678901234567", Singular: "a"}},
		}, ``},
		{"reg 1 group -7 ", Model{
			Groups: map[string]*GroupModel{"a234567890123456789012345678901234567890123456789012345678": {}},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: model type name \"a234567890123456789012345678901234567890123456789012345678\" must match: ^[a-z_][a-z_0-9]{0,56}$"
}`},

		{"reg 1 res 1  ", Model{
			Groups: map[string]*GroupModel{
				"gs": {
					Singular: "g",
					Resources: map[string]*ResourceModel{
						"rs": {},
					},
				},
			},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: Resource \"rs\" is missing a \"singular\" value"
}`},
		{"reg 1 res 2  ", Model{
			Groups: map[string]*GroupModel{
				"gs": {
					Singular: "g",
					Resources: map[string]*ResourceModel{
						"@": {},
					},
				},
			},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: model type name \"@\" must match: ^[a-z_][a-z_0-9]{0,56}$"
}`},
		{"reg 1 res 3  ", Model{
			Groups: map[string]*GroupModel{
				"gs": {
					Singular: "g",
					Resources: map[string]*ResourceModel{
						"rs": {Singular: "rs"},
					},
				},
			},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: Resource \"rs\" has same value for \"plural\" and \"singular\""
}`},

		{"reg 1 res 4  ", Model{
			Groups: map[string]*GroupModel{
				"gs": {
					Singular: "g",
					Resources: map[string]*ResourceModel{
						"rs": {Singular: "r"},
						"r":  {Singular: "rsx"},
					},
				},
			},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: Resource \"rs\" has a \"singular\" value (r) that matches another Resource's \"plural\" value"
}`},
		{"reg 1 group 8  ", Model{
			Groups: map[string]*GroupModel{
				"gs": {Singular: "g"},
				"g":  {Singular: "gsx"},
			},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: Group \"gs\" has a \"singular\" value (g) that matches another Group's \"plural\" value"
}`},
	}

	for _, test := range tests {
		t.Logf("test: %s", test.name)
		xErr := test.model.Verify()
		if test.err == "" && xErr != nil {
			t.Fatalf("ModelVerify: %s - should have worked, got: %s",
				test.name, xErr)
		}
		if test.err != "" && xErr == nil {
			t.Fatalf("ModelVerify: %s - should have failed with: %s",
				test.name, test.err)
		}
		if xErr != nil {
			XEqual(t, test.name, xErr.String(), test.err)
		}
	}
}

func TestModelVerifyRegAttr(t *testing.T) {
	type Test struct {
		name  string
		model Model
		err   string
	}

	groups := map[string]*GroupModel{
		"dirs": &GroupModel{
			Plural:   "dirs",
			Singular: "dir",
			Resources: map[string]*ResourceModel{
				"files": &ResourceModel{
					Plural:   "files",
					Singular: "file",
				},
			},
		},
	}

	m1 := `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"`
	m2 := `"
}`
	a1 := `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "`
	a2 := `"
}`
	tests := []Test{
		{"empty attrs", Model{Attributes: Attributes{}}, ""},
		{
			"err - wrong name", Model{
				Attributes: Attributes{"myint": {Name: "bad"}},
			}, m1 + `model.myint\" must have a \"name\" set to \"myint\"` + m2},
		{"err - missing type", Model{
			Attributes: Attributes{"myint": {Name: "myint"}},
		}, m1 + `model.myint\" is missing a \"type\"` + m2},
		// Test all valid types
		{"type - boolean", Model{
			Attributes: Attributes{"x": {Name: "x", Type: BOOLEAN}}}, ``},
		{"type - decimal", Model{
			Attributes: Attributes{"x": {Name: "x", Type: DECIMAL}}}, ``},
		{"type - integer", Model{
			Attributes: Attributes{"x": {Name: "x", Type: INTEGER}}}, ``},

		/* no longer required
		{"err - type - xid - missing target", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID}}},
			`"model.x" must have a "target" value since "type" is "xid"`},
		*/

		{"err - type - xid - extra target", Model{
			Attributes: Attributes{"x": {Name: "x", Type: STRING, Target: "/"}}},
			m1 + `model.x\" must not have a \"target\" value since \"type\" is not \"xid\"` + m2},
		{"err - type - xid - leading chars", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "xx/"}}},
			m1 + `model.x\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]` + m2},
		{"err - type - xid - extra / at end", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/xx/"}}},
			m1 + `model.x\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]` + m2},
		{"err - type - xid - spaces", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/  xx"}}},
			m1 + `model.x\" has an unknown Group type: \"  xx\"` + m2},
		{"err - type - xid - bad group", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/badg"}},
			Groups: groups},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" has an unknown Group type: \"badg\""
}`,
		},
		{"err - type - xid - bad resource", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/dirs/badr"}},
			Groups: groups},
			m1 + `model.x\" has an unknown Resource type: \"badr\"` + m2},
		{"type - xid - group", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/dirs"}}, Groups: groups}, ``,
		},
		{"type - xid - res", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/dirs/files"}}, Groups: groups}, ``,
		},
		{"type - xid - versions", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/dirs/files/versions"}}, Groups: groups}, ``,
		},
		{"type - xid - both", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/dirs/files[/versions]"}}, Groups: groups}, ``,
		},

		/* no longer required
		{"type - xid - reg - ''", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: ""}}, Groups: groups},
			`"model.x" must have a "target" value since "type" is "xid"`},
		*/
		{"type - xid - reg - /", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/"}}, Groups: groups},
			m1 + `model.x\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]` + m2},

		{"type - string", Model{
			Attributes: Attributes{"x": {Name: "x", Type: STRING}}}, ``},
		{"type - timestamp", Model{
			Attributes: Attributes{"x": {Name: "x", Type: TIMESTAMP}}}, ``},
		{"type - uinteger", Model{
			Attributes: Attributes{"x": {Name: "x", Type: UINTEGER}}}, ``},
		{"type - uri", Model{
			Attributes: Attributes{"x": {Name: "x", Type: URI}}}, ``},
		{"type - urireference", Model{
			Attributes: Attributes{"x": {Name: "x", Type: URI_REFERENCE}}}, ``},
		{"type - uritempalte", Model{
			Attributes: Attributes{"x": {Name: "x", Type: URI_TEMPLATE}}}, ``},
		{"type - url", Model{
			Attributes: Attributes{"x": {Name: "x", Type: URL}}}, ``},
		{"type - any", Model{
			Attributes: Attributes{"x": {Name: "x", Type: ANY}}}, ``},
		{"type - any", Model{
			Attributes: Attributes{"*": {Name: "*", Type: ANY}}}, ``},

		{"type - array", Model{
			Attributes: Attributes{"x": {Name: "x", Type: ARRAY,
				Item: &Item{Type: INTEGER}}}}, ``},
		{"type - map", Model{
			Attributes: Attributes{"x": {Name: "x", Type: MAP,
				Item: &Item{Type: STRING}}}}, ``},
		{"type - object - 1", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT}}}, ``},
		{"type - object - 2", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				Attributes: Attributes{}}}}, ``},
		{"type - object - strict '' - ", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				NameCharSet: ""}}}, ``},
		{"type - object - strict - ", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				NameCharSet: "strict"}}}, ``},
		{"type - object - extended '' - ", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				NameCharSet: "extended"}}}, ``},
		{"type - object - bad name charset - ", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				NameCharSet: "foo"}}},
			m1 + `model.x\" has an invalid \"namecharset\" value: foo` + m2},
		{"type - attr - err1", Model{
			Attributes: Attributes{".foo": {Name: ".foo", Type: ANY}}},
			a1 + `The attribute \".foo\" is not valid: while processing \"model\", attribute name \".foo\" must match: ^[a-z_][a-z_0-9]{0,62}$` + a2},
		{"type - attr - err2", Model{
			Attributes: Attributes{"foo.bar": {}}},
			a1 + `The attribute \"foo.bar\" is not valid: while processing \"model\", attribute name \"foo.bar\" must match: ^[a-z_][a-z_0-9]{0,62}$` + a2},
		{"type - attr - err3", Model{
			Attributes: Attributes{"foo": nil}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: while processing \"model\", attribute \"foo\" can't be empty"
}`},
		{"type - attr - err4", Model{
			Attributes: Attributes{"FOO": {}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"FOO\" is not valid: while processing \"model\", attribute name \"FOO\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"type - attr - err5", Model{
			Attributes: Attributes{"9foo": {}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"9foo\" is not valid: while processing \"model\", attribute name \"9foo\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"type - attr - err6", Model{
			Attributes: Attributes{"": {}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: while processing \"model\", it has an empty attribute key"
}`},
		{"type - attr - ok1", Model{
			Attributes: Attributes{"a23456789012345678901234567890123456789012345678901234567890123": {Name: "a23456789012345678901234567890123456789012345678901234567890123", Type: STRING}}},
			``},
		{"type - attr - err7", Model{
			Attributes: Attributes{"a234567890123456789012345678901234567890123456789012345678901234": {Name: "a234567890123456789012345678901234567890123456789012345678901234", Type: STRING}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"a234567890123456789012345678901234567890123456789012345678901234\" is not valid: while processing \"model\", attribute name \"a234567890123456789012345678901234567890123456789012345678901234\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},

		{"type - array - missing item", Model{
			Attributes: Attributes{"x": {Name: "x", Type: ARRAY}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" must have an \"item\" section"
}`},
		{"type - map - missing item", Model{
			Attributes: Attributes{"x": {Name: "x", Type: MAP}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" must have an \"item\" section"
}`},
		{"type - object - missing item", Model{ // odd but allowable
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT}}}, ""},

		{"type - bad urlx", Model{
			Attributes: Attributes{"x": {Name: "x", Type: "urlx"}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" has an invalid type: urlx"
}`},

		// Now some Item stuff
		{"Item - missing", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT}}}, ""},
		{"Item - empty - ", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				Item: &Item{}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" must not have an \"item\" section"
}`},

		// Nested stuff
		{"Nested - map - obj", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: OBJECT}}}},
			``},
		{"Nested - map - obj - bad namecharset", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: OBJECT, NameCharSet: "foo"}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: invalid \"namecharset\" value: foo"
}`},
		{"Nested - map - obj - missing item - valid", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: OBJECT, Attributes: Attributes{}}}}},
			``},
		{"Nested - map - map - misplaced attrs", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: MAP, Attributes: Attributes{}}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.m.item\" must not have \"attributes\""
}`},
		{"Nested - map - array - misplaced attrs", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: ARRAY, Attributes: Attributes{}}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.m.item\" must not have \"attributes\""
}`},

		{"Nested - map - obj + attr", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: OBJECT, Attributes: Attributes{
					"i": {Name: "i", Type: INTEGER}}}}}},
			``},
		{"Nested - map - obj + obj + attr", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: OBJECT, Attributes: Attributes{
					"i": {Name: "i", Type: OBJECT,
						Attributes: Attributes{"s": {Name: "s",
							Type: STRING}}}}}}}},
			``},
	}

	for _, test := range tests {
		xErr := test.model.Verify()
		if test.err == "" && xErr != nil {
			t.Fatalf("ModelVerify: %s - should have worked, got: %s",
				test.name, xErr)
		}
		if test.err != "" && xErr == nil {
			t.Fatalf("ModelVerify: %s - should have failed with: %s",
				test.name, test.err)
		}
		if xErr != nil && test.err != xErr.String() {
			t.Fatalf("ModifyVerify: %s:\nExp: %s\nGot: %s", test.name,
				test.err, xErr.String())
		}
	}
}

func TestModelVerifyEnum(t *testing.T) {
	type Test struct {
		name  string
		model Model
		err   string
	}

	tests := []Test{
		{"empty enum - int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{1}}}}, ""},
		{"empty enum - obj", Model{Attributes: Attributes{
			"x": {Name: "x", Type: OBJECT, Enum: []any{1}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" is not a scalar, or an array of scalars, so \"enum\" is not allowed"
}`},
		{"empty enum - array", Model{Attributes: Attributes{
			"x": {Name: "x", Type: ARRAY, Item: &Item{Type: OBJECT}, Enum: []any{1}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" is not a scalar, or an array of scalars, so \"enum\" is not allowed"
}`},
		{"empty enum - map", Model{Attributes: Attributes{
			"x": {Name: "x", Type: MAP, Item: &Item{Type: OBJECT}, Enum: []any{1}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" is not a scalar, or an array of scalars, so \"enum\" is not allowed"
}`},
		{"empty enum - any", Model{Attributes: Attributes{
			"x": {Name: "x", Type: ANY, Enum: []any{}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" specifies an \"enum\" but it is empty"
}`},

		{"enum - bool - true ", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true}}}}, ""},
		{"enum - bool 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true, false}}}}, ""},
		{"enum - bool string", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true, ""}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" enum value \"\" must be of type \"boolean\""
}`},
		{"enum - bool float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{5.5}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" enum value \"5.5\" must be of type \"boolean\""
}`},
		{"enum - bool map", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{map[string]string{}}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" enum value \"map[]\" must be of type \"boolean\""
}`},

		{"enum - decimal 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{5.5}}}}, ""},
		{"enum - decimal 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{5.5, 2}}}}, ""},
		{"enum - decimal bool", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{true, 5}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" enum value \"true\" must be of type \"decimal\""
}`},

		{"enum - integer 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{1}}}}, ""},
		{"enum - integer 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{-1, 1}}}}, ""},
		{"enum - integer float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{-1, 1, 3.1}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" enum value \"3.1\" must be of type \"integer\""
}`},
		{"enum - integer float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{[]int{}}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" enum value \"[]\" must be of type \"integer\""
}`},

		{"enum - string 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a"}}}}, ""},
		{"enum - string 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", ""}}}}, ""},
		{"enum - string int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", 0}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" enum value \"0\" must be of type \"string\""
}`},
		{"enum - string struct", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", struct{}{}}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" enum value \"{}\" must be of type \"string\""
}`},

		{"enum - timestamp 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: TIMESTAMP,
				Enum: []any{"2024-01-02T12:01:02Z"}}}}, ""},
		{"enum - timestamp 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: TIMESTAMP,
				Enum: []any{"2024-01-02T12:01:02Z", "2000-12-31T01:02:03Z"}}}},
			""},
		{"enum - timestamp bad", Model{Attributes: Attributes{
			"x": {Name: "x", Type: TIMESTAMP,
				Enum: []any{"2024-01-02T12:01:02Z", "bad"}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" enum value \"bad\" must be of type \"timestamp\""
}`},
		{"enum - timestamp type", Model{Attributes: Attributes{
			"x": {Name: "x", Type: TIMESTAMP,
				Enum: []any{"2024-01-02T12:01:02Z", 5.5}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" enum value \"5.5\" must be of type \"timestamp\""
}`},

		{"enum - uint 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER, Enum: []any{1}}}}, ""},
		{"enum - uint 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER, Enum: []any{1, 2}}}},
			""},
		{"enum - uint bad", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER, Enum: []any{2, -1}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" enum value \"-1\" must be of type \"uinteger\""
}`},
		{"enum - uint type", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER,
				Enum: []any{5.5}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.x\" enum value \"5.5\" must be of type \"uinteger\""
}`},

		{"empty enum - int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: URI, Enum: []any{"..."}}}}, ""},
		{"empty enum - int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: URI_REFERENCE, Enum: []any{"..."}}}}, ""},
		{"empty enum - int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: URI_TEMPLATE, Enum: []any{"..."}}}}, ""},
		{"empty enum - int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: URL, Enum: []any{"..."}}}}, ""},
	}

	for _, test := range tests {
		xErr := test.model.Verify()
		if test.err == "" && xErr != nil {
			t.Fatalf("ModelVerify: %s - should have worked, got: %s",
				test.name, xErr)
		}
		if test.err != "" && xErr == nil {
			t.Fatalf("ModelVerify: %s - should have failed with: %s",
				test.name, test.err)
		}
		if xErr != nil && test.err != xErr.String() {
			t.Fatalf("ModifyVerify: %s\nExp: %s\nGot: %s", test.name,
				test.err, xErr.String())
		}
	}
}

func TestTargetRegExp(t *testing.T) {
	// targetRE
	for _, test := range []struct {
		input  string
		result []string
	}{
		{"", nil},
		{"/", nil},
		{"//versions", nil},
		{"///versions", nil},
		{"/g", []string{"g", "", "", ""}},
		{"/g/", nil},
		{"/g//versions", nil},
		{"/g/[/versions]", nil},
		{"/g/r", []string{"g", "r", "", ""}},
		{"/g/r/", nil},
		{"/g/r//", nil},
		{"/g/r//versions", nil},
		{"/g/r/[/versions]", nil},
		{"/g/r/versions", []string{"g", "r", "versions", ""}},
		{"/g/r//versions/", nil},
		{"/g/r//versions[/versions]", nil},
		{"/g/r[/versions]", []string{"g", "r", "", "[/versions]"}},
		{"/g/r/[/versions]", nil},
		{"/g/r[/versions]/", nil},
		{"/g/r[/versions]/versions", nil},
	} {
		parts := targetRE.FindStringSubmatch(test.input)
		tmpParts := []string{}
		if len(parts) > 1 {
			tmpParts = parts[1:]
		}

		exp := fmt.Sprintf("%#v", test.result)
		got := fmt.Sprintf("%#v", tmpParts)

		if (len(parts) == 0 || parts[0] == "") && test.result == nil {
			continue
		}

		if len(tmpParts) != len(test.result) || exp != got {
			t.Fatalf("\nIn: %s\nExp: %s\nGot: %s", test.input, exp, got)
		}
	}
}

func TestValidChars(t *testing.T) {
	a10 := "a234567890"
	a50 := a10 + a10 + a10 + a10 + a10
	a57 := a50 + "1234567"
	a58 := a50 + "12345678"
	a60 := a50 + a10
	a63 := a60 + "123"
	a64 := a63 + "4"
	a128 := a64 + a64
	a129 := a128 + "9"

	// Test Group and Resource model type names
	match := RegexpModelName.String()
	for _, test := range []struct {
		input  string
		result string
	}{
		{"", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: model type name \"\" must match: ^[a-z_][a-z_0-9]{0,56}$"
}`},
		{"A", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: model type name \"A\" must match: ^[a-z_][a-z_0-9]{0,56}$"
}`},
		{"*", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: model type name \"*\" must match: ^[a-z_][a-z_0-9]{0,56}$"
}`},
		{"@", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: model type name \"@\" must match: ^[a-z_][a-z_0-9]{0,56}$"
}`},
		{"0", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: model type name \"0\" must match: ^[a-z_][a-z_0-9]{0,56}$"
}`},
		{"0a", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: model type name \"0a\" must match: ^[a-z_][a-z_0-9]{0,56}$"
}`},
		{"aZ", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: model type name \"aZ\" must match: ^[a-z_][a-z_0-9]{0,56}$"
}`},
		{a58, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: model type name \"a234567890a234567890a234567890a234567890a23456789012345678\" must match: ^[a-z_][a-z_0-9]{0,56}$"
}`},
		{"a", ``},
		{"_", ``},
		{"_a", ``},
		{"_8", ``},
		{"a_", ``},
		{"a_8", ``},
		{"aa", ``},
		{"a9", ``},
		{a57, ``},
	} {
		xErr := IsValidModelName(test.input)
		got := ""
		if xErr != nil {
			got = xErr.String()
		}
		if got != test.result {
			t.Fatalf("Test: %s\nExp: %s\nGot: %s", test.input, test.result, got)
		}
	}

	// Test attribute names
	match = RegexpPropName.String()
	for _, test := range []struct {
		input  string
		result string
	}{
		{"", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"\" is not valid: attribute name \"\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"A", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"A\" is not valid: attribute name \"A\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"*", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"*\" is not valid: attribute name \"*\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"@", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"@\" is not valid: attribute name \"@\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"0", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"0\" is not valid: attribute name \"0\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"0a", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"0a\" is not valid: attribute name \"0a\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"aZ", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"aZ\" is not valid: attribute name \"aZ\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{a64, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"` + a64 + `\" is not valid: attribute name \"` + a64 + `\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"a", ``},
		{"_", ``},
		{"_a", ``},
		{"_8", ``},
		{"a_", ``},
		{"a_8", ``},
		{"aa", ``},
		{"a9", ``},
		{a63, ``},
	} {
		xErr := IsValidAttributeName(test.input, "")
		got := ""
		if xErr != nil {
			got = xErr.String()
		}
		if got != test.result {
			t.Fatalf("Test: %s\nExp: %s\nGot: %s", test.input, test.result, got)
		}
	}

	// Test IDs
	match = RegexpID.String()
	m1 := `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "/",
  "title": "The request cannot be processed as provided: ID value \"`
	m2 := `\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$"
}`
	for _, test := range []struct {
		input  string
		result string
	}{
		{"", m1 + "" + m2},
		{"*", m1 + "*" + m2},
		{"!", m1 + "!" + m2},
		{"+", m1 + "+" + m2},
		{"A*", m1 + "A*" + m2},
		{"*a", m1 + "*a" + m2},
		{a129, m1 + a129 + m2},
		{"a", ``},
		{"A", ``},
		{"_", ``},
		{"0", ``},
		{"9", ``},
		{"aa", ``},
		{"aA", ``},
		{"a_", ``},
		{"a.", ``},
		{"a-", ``},
		{"a~", ``},
		{"a@", ``},
		{"a9", ``},
		{"9a", ``},
		{"9A", ``},
		{"9_", ``},
		{"9.", ``},
		{"9-", ``},
		{"9~", ``},
		{"9@", ``},
		{"90", ``},
		{"_Z", ``},
		{"_Z_", ``},
		{" a", m1 + " a" + m2},
		{".", m1 + "." + m2},
		{"-", m1 + "-" + m2},
		{"~", m1 + "~" + m2},
		{"@", m1 + "@" + m2},
		{"Z.-~:_0Nb", ``},
		{a128, ``},
	} {
		xErr := IsValidID(test.input, "")
		got := ""
		if xErr != nil {
			got = xErr.String()
		}
		if got != test.result {
			t.Fatalf("Test: %s\nExp: %s\nGot: %s", test.input, test.result, got)
		}
	}

	// Test map keys
	match = JSONEscape(RegexpMapKey.String())
	m1 = `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: map key name \"`
	m2 = `\" must match: ` + match + `"
}`
	for _, test := range []struct {
		input  string
		result string
	}{
		{"", m1 + "" + m2},
		{"_", m1 + "_" + m2},
		{".", m1 + "." + m2},
		{"-", m1 + "-" + m2},
		{"*", m1 + "*" + m2},
		{"!", m1 + "!" + m2},
		{"~", m1 + "~" + m2},
		{"A", m1 + "A" + m2},
		{"aA", m1 + "aA" + m2},
		{"Aa", m1 + "Aa" + m2},
		{"_a", m1 + "_a" + m2},
		{"9A", m1 + "9A" + m2},
		{"a*", m1 + "a*" + m2},
		{"a!", m1 + "a!" + m2},
		{"a~", m1 + "a~" + m2},
		{":a", m1 + ":a" + m2},
		{a64, m1 + a64 + m2},

		{"a", ``},
		{"0", ``},
		{"a0", ``},
		{"0a", ``},
		{"zb", ``},
		{"m_.-", ``},
		{"m-", ``},
		{"m_", ``},
		{"m-z", ``},
		{"m.9", ``},
		{"m_9", ``},
		{"m:9", ``},
		{a63, ``},
	} {
		xErr := IsValidMapKey(test.input, "")
		got := ""
		if xErr != nil {
			got = xErr.String()
		}
		if got != test.result {
			t.Fatalf("Test: %s\nExp: %s\nGot: %s", test.input, test.result, got)
		}
	}
}
