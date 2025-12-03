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
  "title": "There was an error in the model definition provided: Group definition for \"Gs1\" can't be empty.",
  "subject": "/model",
  "args": {
    "error_detail": "Group definition for \"Gs1\" can't be empty"
  },
  "source": "xxx"
}`},
		{"reg 1 group -3 ", Model{
			Groups: map[string]*GroupModel{"Gs1": {}},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: model type name \"Gs1\" must match: ^[a-z_][a-z_0-9]{0,56}$.",
  "subject": "/model",
  "args": {
    "error_detail": "model type name \"Gs1\" must match: ^[a-z_][a-z_0-9]{0,56}$"
  },
  "source": ":registry:shared_model:33"
}`},
		{"reg 1 group -4 ", Model{
			Groups: map[string]*GroupModel{"@": {}},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: model type name \"@\" must match: ^[a-z_][a-z_0-9]{0,56}$.",
  "subject": "/model",
  "args": {
    "error_detail": "model type name \"@\" must match: ^[a-z_][a-z_0-9]{0,56}$"
  },
  "source": ":registry:shared_model:33"
}`},
		{"reg 1 group -4.5 ", Model{
			Groups: map[string]*GroupModel{"a": {}},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group \"a\" is missing a \"singular\" value.",
  "subject": "/model",
  "args": {
    "error_detail": "Group \"a\" is missing a \"singular\" value"
  },
  "source": ":registry:shared_model:1894"
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
  "title": "There was an error in the model definition provided: Group \"a\" has same value for \"plural\" and \"singular\".",
  "subject": "/model",
  "args": {
    "error_detail": "Group \"a\" has same value for \"plural\" and \"singular\""
  },
  "source": ":registry:shared_model:1900"
}`},
		{"reg 1 group -6 ", Model{
			Groups: map[string]*GroupModel{"a23456789012345678901234567890123456789012345678901234567": {Plural: "a23456789012345678901234567890123456789012345678901234567", Singular: "a"}},
		}, ``},
		{"reg 1 group -7 ", Model{
			Groups: map[string]*GroupModel{"a234567890123456789012345678901234567890123456789012345678": {}},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: model type name \"a234567890123456789012345678901234567890123456789012345678\" must match: ^[a-z_][a-z_0-9]{0,56}$.",
  "subject": "/model",
  "args": {
    "error_detail": "model type name \"a234567890123456789012345678901234567890123456789012345678\" must match: ^[a-z_][a-z_0-9]{0,56}$"
  },
  "source": ":registry:shared_model:33"
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
  "title": "There was an error in the model definition provided: Resource \"rs\" is missing a \"singular\" value.",
  "subject": "/model",
  "args": {
    "error_detail": "Resource \"rs\" is missing a \"singular\" value"
  },
  "source": ":registry:shared_model:2262"
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
  "title": "There was an error in the model definition provided: model type name \"@\" must match: ^[a-z_][a-z_0-9]{0,56}$.",
  "subject": "/model",
  "args": {
    "error_detail": "model type name \"@\" must match: ^[a-z_][a-z_0-9]{0,56}$"
  },
  "source": ":registry:shared_model:33"
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
  "title": "There was an error in the model definition provided: Resource \"rs\" has same value for \"plural\" and \"singular\".",
  "subject": "/model",
  "args": {
    "error_detail": "Resource \"rs\" has same value for \"plural\" and \"singular\""
  },
  "source": ":registry:shared_model:2268"
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
  "title": "There was an error in the model definition provided: Resource \"rs\" has a \"singular\" value (r) that matches another Resource's \"plural\" value.",
  "subject": "/model",
  "args": {
    "error_detail": "Resource \"rs\" has a \"singular\" value (r) that matches another Resource's \"plural\" value"
  },
  "source": ":registry:shared_model:2275"
}`},
		{"reg 1 group 8  ", Model{
			Groups: map[string]*GroupModel{
				"gs": {Singular: "g"},
				"g":  {Singular: "gsx"},
			},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group \"gs\" has a \"singular\" value (g) that matches another Group's \"plural\" value.",
  "subject": "/model",
  "args": {
    "error_detail": "Group \"gs\" has a \"singular\" value (g) that matches another Group's \"plural\" value"
  },
  "source": ":registry:shared_model:1907"
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

	tests := []Test{
		{"empty attrs", Model{Attributes: Attributes{}}, ""},
		{
			"err - wrong name", Model{
				Attributes: Attributes{"myint": {Name: "bad"}},
			}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"myint\" must have a \"name\" set to \"myint\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"myint\" must have a \"name\" set to \"myint\""
  },
  "source": ":registry:shared_model:2830"
}`},
		{"err - missing type", Model{
			Attributes: Attributes{"myint": {Name: "myint"}},
		}, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"myint\" is missing a \"type\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"myint\" is missing a \"type\""
  },
  "source": ":registry:shared_model:2836"
}`},
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
			`"x" must have a "target" value since "type" is "xid"`},
		*/

		{"err - type - xid - extra target", Model{
			Attributes: Attributes{"x": {Name: "x", Type: STRING, Target: "/"}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" must not have a \"target\" value since \"type\" is not \"xid\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" must not have a \"target\" value since \"type\" is not \"xid\""
  },
  "source": ":registry:shared_model:2891"
}`},
		{"err - type - xid - leading chars", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "xx/"}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": ":registry:shared_model:2865"
}`},
		{"err - type - xid - extra / at end", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/xx/"}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": ":registry:shared_model:2865"
}`},
		{"err - type - xid - spaces", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/  xx"}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" has an unknown Group type: \"  xx\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" has an unknown Group type: \"  xx\""
  },
  "source": ":registry:shared_model:2874"
}`},
		{"err - type - xid - bad group", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/badg"}},
			Groups: groups},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" has an unknown Group type: \"badg\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" has an unknown Group type: \"badg\""
  },
  "source": ":registry:shared_model:2874"
}`,
		},
		{"err - type - xid - bad resource", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/dirs/badr"}},
			Groups: groups},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" has an unknown Resource type: \"badr\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" has an unknown Resource type: \"badr\""
  },
  "source": ":registry:shared_model:2881"
}`},
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
			`"x" must have a "target" value since "type" is "xid"`},
		*/
		{"type - xid - reg - /", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/"}}, Groups: groups},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": ":registry:shared_model:2865"
}`},

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
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" has an invalid \"namecharset\" value: foo.",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" has an invalid \"namecharset\" value: foo"
  },
  "source": ":registry:shared_model:2969"
}`},
		{"type - attr - err1", Model{
			Attributes: Attributes{".foo": {Name: ".foo", Type: ANY}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \".foo\" for \"/model\" is not valid: \".foo\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\".foo\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": ".foo"
  },
  "source": "e4e59b8a76c4:registry:shared_model:53,registry:shared_model:2834"
}`},
		{"type - attr - err2", Model{
			Attributes: Attributes{"foo.bar": {}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"foo.bar\" for \"/model\" is not valid: \"foo.bar\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\"foo.bar\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": "foo.bar"
  },
  "source": "e4e59b8a76c4:registry:shared_model:53,registry:shared_model:2834"
}`},
		{"type - attr - err3", Model{
			Attributes: Attributes{"foo": nil}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: attribute \"foo\" can't be empty.",
  "subject": "/model",
  "args": {
    "error_detail": "attribute \"foo\" can't be empty"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2791"
}`},
		{"type - attr - err4", Model{
			Attributes: Attributes{"FOO": {}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"FOO\" for \"/model\" is not valid: \"FOO\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\"FOO\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": "FOO"
  },
  "source": "e4e59b8a76c4:registry:shared_model:53,registry:shared_model:2834"
}`},
		{"type - attr - err5", Model{
			Attributes: Attributes{"9foo": {}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"9foo\" for \"/model\" is not valid: \"9foo\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\"9foo\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": "9foo"
  },
  "source": "e4e59b8a76c4:registry:shared_model:53,registry:shared_model:2834"
}`},
		{"type - attr - err6", Model{
			Attributes: Attributes{"": {}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: while processing \"/\", it has an empty attribute key.",
  "subject": "/model",
  "args": {
    "error_detail": "while processing \"/\", it has an empty attribute key"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2797"
}`},
		{"type - attr - ok1", Model{
			Attributes: Attributes{"a23456789012345678901234567890123456789012345678901234567890123": {Name: "a23456789012345678901234567890123456789012345678901234567890123", Type: STRING}}},
			``},
		{"type - attr - err7", Model{
			Attributes: Attributes{"a234567890123456789012345678901234567890123456789012345678901234": {Name: "a234567890123456789012345678901234567890123456789012345678901234", Type: STRING}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"a234567890123456789012345678901234567890123456789012345678901234\" for \"/model\" is not valid: \"a234567890123456789012345678901234567890123456789012345678901234\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\"a234567890123456789012345678901234567890123456789012345678901234\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": "a234567890123456789012345678901234567890123456789012345678901234"
  },
  "source": "e4e59b8a76c4:registry:shared_model:53,registry:shared_model:2834"
}`},

		{"type - array - missing item", Model{
			Attributes: Attributes{"x": {Name: "x", Type: ARRAY}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" must have an \"item\" section.",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" must have an \"item\" section"
  },
  "source": ":registry:shared_model:2904"
}`},
		{"type - map - missing item", Model{
			Attributes: Attributes{"x": {Name: "x", Type: MAP}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" must have an \"item\" section.",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" must have an \"item\" section"
  },
  "source": ":registry:shared_model:2904"
}`},
		{"type - object - missing item", Model{ // odd but allowable
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT}}}, ""},

		{"type - bad urlx", Model{
			Attributes: Attributes{"x": {Name: "x", Type: "urlx"}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" has an invalid type: urlx.",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" has an invalid type: urlx"
  },
  "source": ":registry:shared_model:2841"
}`},

		// Now some Item stuff
		{"Item - missing", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT}}}, ""},
		{"Item - empty - ", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				Item: &Item{}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" must not have an \"item\" section.",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" must not have an \"item\" section"
  },
  "source": ":registry:shared_model:2976"
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
  "title": "There was an error in the model definition provided: invalid \"namecharset\" value: foo.",
  "subject": "/model",
  "args": {
    "error_detail": "invalid \"namecharset\" value: foo"
  },
  "source": ":registry:shared_model:3093"
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
  "title": "There was an error in the model definition provided: \"m.item\" must not have \"attributes\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"m.item\" must not have \"attributes\""
  },
  "source": ":registry:shared_model:3078"
}`},
		{"Nested - map - array - misplaced attrs", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: ARRAY, Attributes: Attributes{}}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"m.item\" must not have \"attributes\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"m.item\" must not have \"attributes\""
  },
  "source": ":registry:shared_model:3078"
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
		if xErr != nil {
			XEqual(t, test.name, xErr.String(), test.err)
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
  "title": "There was an error in the model definition provided: \"x\" is not a scalar, or an array of scalars, so \"enum\" is not allowed.",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" is not a scalar, or an array of scalars, so \"enum\" is not allowed"
  },
  "source": ":registry:shared_model:2926"
}`},
		{"empty enum - array", Model{Attributes: Attributes{
			"x": {Name: "x", Type: ARRAY, Item: &Item{Type: OBJECT}, Enum: []any{1}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" is not a scalar, or an array of scalars, so \"enum\" is not allowed.",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" is not a scalar, or an array of scalars, so \"enum\" is not allowed"
  },
  "source": ":registry:shared_model:2926"
}`},
		{"empty enum - map", Model{Attributes: Attributes{
			"x": {Name: "x", Type: MAP, Item: &Item{Type: OBJECT}, Enum: []any{1}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" is not a scalar, or an array of scalars, so \"enum\" is not allowed.",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" is not a scalar, or an array of scalars, so \"enum\" is not allowed"
  },
  "source": ":registry:shared_model:2926"
}`},
		{"empty enum - any", Model{Attributes: Attributes{
			"x": {Name: "x", Type: ANY, Enum: []any{}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" specifies an \"enum\" but it is empty.",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" specifies an \"enum\" but it is empty"
  },
  "source": ":registry:shared_model:2913"
}`},

		{"enum - bool - true ", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true}}}}, ""},
		{"enum - bool 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true, false}}}}, ""},
		{"enum - bool string", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true, ""}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" enum value \"\" must be of type \"boolean\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" enum value \"\" must be of type \"boolean\""
  },
  "source": ":registry:shared_model:2934"
}`},
		{"enum - bool float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{5.5}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" enum value \"5.5\" must be of type \"boolean\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" enum value \"5.5\" must be of type \"boolean\""
  },
  "source": ":registry:shared_model:2934"
}`},
		{"enum - bool map", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{map[string]string{}}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" enum value \"map[]\" must be of type \"boolean\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" enum value \"map[]\" must be of type \"boolean\""
  },
  "source": ":registry:shared_model:2934"
}`},

		{"enum - decimal 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{5.5}}}}, ""},
		{"enum - decimal 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{5.5, 2}}}}, ""},
		{"enum - decimal bool", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{true, 5}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" enum value \"true\" must be of type \"decimal\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" enum value \"true\" must be of type \"decimal\""
  },
  "source": ":registry:shared_model:2934"
}`},

		{"enum - integer 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{1}}}}, ""},
		{"enum - integer 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{-1, 1}}}}, ""},
		{"enum - integer float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{-1, 1, 3.1}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" enum value \"3.1\" must be of type \"integer\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" enum value \"3.1\" must be of type \"integer\""
  },
  "source": ":registry:shared_model:2934"
}`},
		{"enum - integer float2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{[]int{}}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" enum value \"[]\" must be of type \"integer\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" enum value \"[]\" must be of type \"integer\""
  },
  "source": ":registry:shared_model:2934"
}`},

		{"enum - string 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a"}}}}, ""},
		{"enum - string 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", ""}}}}, ""},
		{"enum - string int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", 0}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" enum value \"0\" must be of type \"string\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" enum value \"0\" must be of type \"string\""
  },
  "source": ":registry:shared_model:2934"
}`},
		{"enum - string struct", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", struct{}{}}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" enum value \"{}\" must be of type \"string\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" enum value \"{}\" must be of type \"string\""
  },
  "source": ":registry:shared_model:2934"
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
  "title": "There was an error in the model definition provided: \"x\" enum value \"bad\" must be of type \"timestamp\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" enum value \"bad\" must be of type \"timestamp\""
  },
  "source": ":registry:shared_model:2934"
}`},
		{"enum - timestamp type", Model{Attributes: Attributes{
			"x": {Name: "x", Type: TIMESTAMP,
				Enum: []any{"2024-01-02T12:01:02Z", 5.5}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" enum value \"5.5\" must be of type \"timestamp\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" enum value \"5.5\" must be of type \"timestamp\""
  },
  "source": ":registry:shared_model:2934"
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
  "title": "There was an error in the model definition provided: \"x\" enum value \"-1\" must be of type \"uinteger\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" enum value \"-1\" must be of type \"uinteger\""
  },
  "source": ":registry:shared_model:2934"
}`},
		{"enum - uint type", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER,
				Enum: []any{5.5}}}},
			`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"x\" enum value \"5.5\" must be of type \"uinteger\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"x\" enum value \"5.5\" must be of type \"uinteger\""
  },
  "source": ":registry:shared_model:2934"
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
		if xErr != nil {
			XEqual(t, test.name, xErr.String(), test.err)
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
	for _, test := range []struct {
		input  string
		result string
	}{
		{"", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: model type name \"\" must match: ^[a-z_][a-z_0-9]{0,56}$.",
  "subject": "/model",
  "args": {
    "error_detail": "model type name \"\" must match: ^[a-z_][a-z_0-9]{0,56}$"
  },
  "source": ":registry:shared_model:33"
}`},
		{"A", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: model type name \"A\" must match: ^[a-z_][a-z_0-9]{0,56}$.",
  "subject": "/model",
  "args": {
    "error_detail": "model type name \"A\" must match: ^[a-z_][a-z_0-9]{0,56}$"
  },
  "source": ":registry:shared_model:33"
}`},
		{"*", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: model type name \"*\" must match: ^[a-z_][a-z_0-9]{0,56}$.",
  "subject": "/model",
  "args": {
    "error_detail": "model type name \"*\" must match: ^[a-z_][a-z_0-9]{0,56}$"
  },
  "source": ":registry:shared_model:33"
}`},
		{"@", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: model type name \"@\" must match: ^[a-z_][a-z_0-9]{0,56}$.",
  "subject": "/model",
  "args": {
    "error_detail": "model type name \"@\" must match: ^[a-z_][a-z_0-9]{0,56}$"
  },
  "source": ":registry:shared_model:33"
}`},
		{"0", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: model type name \"0\" must match: ^[a-z_][a-z_0-9]{0,56}$.",
  "subject": "/model",
  "args": {
    "error_detail": "model type name \"0\" must match: ^[a-z_][a-z_0-9]{0,56}$"
  },
  "source": ":registry:shared_model:33"
}`},
		{"0a", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: model type name \"0a\" must match: ^[a-z_][a-z_0-9]{0,56}$.",
  "subject": "/model",
  "args": {
    "error_detail": "model type name \"0a\" must match: ^[a-z_][a-z_0-9]{0,56}$"
  },
  "source": ":registry:shared_model:33"
}`},
		{"aZ", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: model type name \"aZ\" must match: ^[a-z_][a-z_0-9]{0,56}$.",
  "subject": "/model",
  "args": {
    "error_detail": "model type name \"aZ\" must match: ^[a-z_][a-z_0-9]{0,56}$"
  },
  "source": ":registry:shared_model:33"
}`},
		{a58, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: model type name \"a234567890a234567890a234567890a234567890a23456789012345678\" must match: ^[a-z_][a-z_0-9]{0,56}$.",
  "subject": "/model",
  "args": {
    "error_detail": "model type name \"a234567890a234567890a234567890a234567890a23456789012345678\" must match: ^[a-z_][a-z_0-9]{0,56}$"
  },
  "source": ":registry:shared_model:33"
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
		XEqual(t, test.input, got, test.result)
	}

	// Test attribute names
	for _, test := range []struct {
		input  string
		result string
	}{
		{"", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"\" for \"/model\" is not valid: \"\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\"\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": ""
  },
  "source": "e4e59b8a76c4:registry:shared_model:53"
}`},
		{"A", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"A\" for \"/model\" is not valid: \"A\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\"A\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": "A"
  },
  "source": "e4e59b8a76c4:registry:shared_model:53"
}`},
		{"*", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"*\" for \"/model\" is not valid: \"*\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\"*\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": "*"
  },
  "source": "e4e59b8a76c4:registry:shared_model:53"
}`},
		{"@", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"@\" for \"/model\" is not valid: \"@\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\"@\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": "@"
  },
  "source": "e4e59b8a76c4:registry:shared_model:53"
}`},
		{"0", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"0\" for \"/model\" is not valid: \"0\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\"0\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": "0"
  },
  "source": "e4e59b8a76c4:registry:shared_model:53"
}`},
		{"0a", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"0a\" for \"/model\" is not valid: \"0a\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\"0a\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": "0a"
  },
  "source": "e4e59b8a76c4:registry:shared_model:53"
}`},
		{"aZ", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"aZ\" for \"/model\" is not valid: \"aZ\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\"aZ\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": "aZ"
  },
  "source": "e4e59b8a76c4:registry:shared_model:53"
}`},
		{a64, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"a234567890a234567890a234567890a234567890a234567890a2345678901234\" for \"/model\" is not valid: \"a234567890a234567890a234567890a234567890a234567890a2345678901234\" must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "\"a234567890a234567890a234567890a234567890a234567890a2345678901234\" must match: ^[a-z_][a-z_0-9]{0,62}$",
    "name": "a234567890a234567890a234567890a234567890a234567890a2345678901234"
  },
  "source": "e4e59b8a76c4:registry:shared_model:53"
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
		xErr := IsValidAttributeName(test.input, "/model", "")
		got := ""
		if xErr != nil {
			got = xErr.String()
		}
		XEqual(t, test.input, got, test.result)
	}

	// Test IDs
	for _, test := range []struct {
		input  string
		result string
	}{
		{"", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_id",
  "title": "The specified ID value () is malformed: ID value \"\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/model",
  "args": {
    "error_detail": "ID value \"\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "id": ""
  },
  "source": ":registry:shared_model:76"
}`},
		{"*", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_id",
  "title": "The specified ID value (*) is malformed: ID value \"*\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/model",
  "args": {
    "error_detail": "ID value \"*\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "id": "*"
  },
  "source": ":registry:shared_model:76"
}`},
		{"!", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_id",
  "title": "The specified ID value (!) is malformed: ID value \"!\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/model",
  "args": {
    "error_detail": "ID value \"!\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "id": "!"
  },
  "source": ":registry:shared_model:76"
}`},
		{"+", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_id",
  "title": "The specified ID value (+) is malformed: ID value \"+\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/model",
  "args": {
    "error_detail": "ID value \"+\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "id": "+"
  },
  "source": ":registry:shared_model:76"
}`},
		{"A*", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_id",
  "title": "The specified ID value (A*) is malformed: ID value \"A*\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/model",
  "args": {
    "error_detail": "ID value \"A*\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "id": "A*"
  },
  "source": ":registry:shared_model:76"
}`},
		{"*a", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_id",
  "title": "The specified ID value (*a) is malformed: ID value \"*a\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/model",
  "args": {
    "error_detail": "ID value \"*a\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "id": "*a"
  },
  "source": ":registry:shared_model:76"
}`},
		{a129, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_id",
  "title": "The specified ID value (a234567890a234567890a234567890a234567890a234567890a2345678901234a234567890a234567890a234567890a234567890a234567890a23456789012349) is malformed: ID value \"a234567890a234567890a234567890a234567890a234567890a2345678901234a234567890a234567890a234567890a234567890a234567890a23456789012349\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/model",
  "args": {
    "error_detail": "ID value \"a234567890a234567890a234567890a234567890a234567890a2345678901234a234567890a234567890a234567890a234567890a234567890a23456789012349\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "id": "a234567890a234567890a234567890a234567890a234567890a2345678901234a234567890a234567890a234567890a234567890a234567890a23456789012349"
  },
  "source": ":registry:shared_model:76"
}`},
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
		{" a", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_id",
  "title": "The specified ID value ( a) is malformed: ID value \" a\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/model",
  "args": {
    "error_detail": "ID value \" a\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "id": " a"
  },
  "source": ":registry:shared_model:76"
}`},
		{".", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_id",
  "title": "The specified ID value (.) is malformed: ID value \".\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/model",
  "args": {
    "error_detail": "ID value \".\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "id": "."
  },
  "source": ":registry:shared_model:76"
}`},
		{"-", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_id",
  "title": "The specified ID value (-) is malformed: ID value \"-\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/model",
  "args": {
    "error_detail": "ID value \"-\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "id": "-"
  },
  "source": ":registry:shared_model:76"
}`},
		{"~", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_id",
  "title": "The specified ID value (~) is malformed: ID value \"~\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/model",
  "args": {
    "error_detail": "ID value \"~\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "id": "~"
  },
  "source": ":registry:shared_model:76"
}`},
		{"@", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_id",
  "title": "The specified ID value (@) is malformed: ID value \"@\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/model",
  "args": {
    "error_detail": "ID value \"@\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "id": "@"
  },
  "source": ":registry:shared_model:76"
}`},
		{"Z.-~:_0Nb", ``},
		{a128, ``},
	} {
		xErr := IsValidID(test.input, "")
		got := ""
		if xErr != nil {
			got = xErr.String()
		}
		XEqual(t, "Input: "+test.input, got, test.result)
	}

	// Test map keys
	fn := func(v string) string {
		return fmt.Sprintf(`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: map key name \"%s\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "map key name \"%s\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$"
  },
  "source": ":registry:shared_model:61"
}`, v, v)
	}
	for _, test := range []struct {
		input  string
		result string
	}{
		{"", fn("")},
		{"_", fn("_")},
		{".", fn(".")},
		{"-", fn("-")},
		{"*", fn("*")},
		{"!", fn("!")},
		{"~", fn("~")},
		{"A", fn("A")},
		{"aA", fn("aA")},
		{"Aa", fn("Aa")},
		{"_a", fn("_a")},
		{"9A", fn("9A")},
		{"a*", fn("a*")},
		{"a!", fn("a!")},
		{"a~", fn("a~")},
		{":a", fn(":a")},
		{a64, fn(a64)},

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
		xErr := IsValidMapKey(test.input, "/model", "")
		got := ""
		if xErr != nil {
			got = xErr.String()
		}
		XEqual(t, test.input, got, test.result)
	}
}
