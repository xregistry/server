package registry

import (
	"fmt"
	"reflect"
	"testing"
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
		}, `GroupModel "Gs1" can't be empty`},
		{"reg 1 group -3 ", Model{
			Groups: map[string]*GroupModel{"Gs1": {}},
		}, `Invalid model type name "Gs1", must match: ^[a-z_][a-z_0-9]{0,57}$`},
		{"reg 1 group -4 ", Model{
			Groups: map[string]*GroupModel{"@": {}},
		}, `Invalid model type name "@", must match: ^[a-z_][a-z_0-9]{0,57}$`},
		{"reg 1 group -5 ", Model{
			Groups: map[string]*GroupModel{"_a": {Plural: "_a", Singular: "a"}},
		}, ``},
		{"reg 1 group -6 ", Model{
			Groups: map[string]*GroupModel{"a234567890123456789012345678901234567890123456789012345678": {Plural: "a234567890123456789012345678901234567890123456789012345678", Singular: "a"}},
		}, ``},
		{"reg 1 group -7 ", Model{
			Groups: map[string]*GroupModel{"a2345678901234567890123456789012345678901234567890123456789": {}},
		}, `Invalid model type name "a2345678901234567890123456789012345678901234567890123456789", must match: ^[a-z_][a-z_0-9]{0,57}$`},
	}

	for _, test := range tests {
		err := test.model.Verify()
		if test.err == "" && err != nil {
			t.Fatalf("ModelVerify: %s - should have worked, got: %s",
				test.name, err)
		}
		if test.err != "" && err == nil {
			t.Fatalf("ModelVerify: %s - should have failed with: %s",
				test.name, test.err)
		}
		if err != nil && test.err != err.Error() {
			t.Fatalf("ModifyVerify: %s\nexp: %s\ngot: %s", test.name,
				test.err, err.Error())
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
			}, `"model.myint" must have a "name" set to "myint"`},
		{"err - missing type", Model{
			Attributes: Attributes{"myint": {Name: "myint"}},
		}, `"model.myint" is missing a "type"`},
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
			`"model.x" must not have a "target" value since "type" is not "xid"`},
		{"err - type - xid - leading chars", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "xx/"}}},
			`"model.x" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`},
		{"err - type - xid - extra / at end", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/xx/"}}},
			`"model.x" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`},
		{"err - type - xid - spaces", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/  xx"}}},
			`"model.x" has an unknown Group type: "  xx"`},
		{"err - type - xid - bad group", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/badg"}},
			Groups: groups},
			`"model.x" has an unknown Group type: "badg"`,
		},
		{"err - type - xid - bad resource", Model{
			Attributes: Attributes{"x": {Name: "x", Type: XID,
				Target: "/dirs/badr"}},
			Groups: groups},
			`"model.x" has an unknown Resource type: "badr"`,
		},
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
			`"model.x" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`,
		},

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
			`"model.x" has an invalid "namecharset" value: foo`},

		{"type - attr - err1", Model{
			Attributes: Attributes{".foo": {Name: ".foo", Type: ANY}}},
			`Error processing "model": Invalid attribute name ".foo", must match: ^[a-z_][a-z_0-9]{0,62}$`},
		{"type - attr - err2", Model{
			Attributes: Attributes{"foo.bar": {}}},
			`Error processing "model": Invalid attribute name "foo.bar", must match: ^[a-z_][a-z_0-9]{0,62}$`},
		{"type - attr - err3", Model{
			Attributes: Attributes{"foo": nil}},
			`Error processing "model": attribute "foo" can't be empty`},
		{"type - attr - err4", Model{
			Attributes: Attributes{"FOO": {}}},
			`Error processing "model": Invalid attribute name "FOO", must match: ^[a-z_][a-z_0-9]{0,62}$`},
		{"type - attr - err5", Model{
			Attributes: Attributes{"9foo": {}}},
			`Error processing "model": Invalid attribute name "9foo", must match: ^[a-z_][a-z_0-9]{0,62}$`},
		{"type - attr - err6", Model{
			Attributes: Attributes{"": {}}},
			`Error processing "model": it has an empty attribute key`},
		{"type - attr - ok1", Model{
			Attributes: Attributes{"a23456789012345678901234567890123456789012345678901234567890123": {Name: "a23456789012345678901234567890123456789012345678901234567890123", Type: STRING}}},
			``},
		{"type - attr - err7", Model{
			Attributes: Attributes{"a234567890123456789012345678901234567890123456789012345678901234": {Name: "a234567890123456789012345678901234567890123456789012345678901234", Type: STRING}}},
			`Error processing "model": Invalid attribute name "a234567890123456789012345678901234567890123456789012345678901234", must match: ^[a-z_][a-z_0-9]{0,62}$`},

		{"type - array - missing item", Model{
			Attributes: Attributes{"x": {Name: "x", Type: ARRAY}}},
			`"model.x" must have an "item" section`},
		{"type - map - missing item", Model{
			Attributes: Attributes{"x": {Name: "x", Type: MAP}}},
			`"model.x" must have an "item" section`},
		{"type - object - missing item", Model{ // odd but allowable
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT}}}, ""},

		{"type - bad urlx", Model{
			Attributes: Attributes{"x": {Name: "x", Type: "urlx"}}},
			`"model.x" has an invalid type: urlx`},

		// Now some Item stuff
		{"Item - missing", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT}}}, ""},
		{"Item - empty - ", Model{
			Attributes: Attributes{"x": {Name: "x", Type: OBJECT,
				Item: &Item{}}}},
			`"model.x" must not have an "item" section`},

		// Nested stuff
		{"Nested - map - obj", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: OBJECT}}}},
			``},
		{"Nested - map - obj - bad namecharset", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: OBJECT, NameCharSet: "foo"}}}},
			`Invalid "namecharset" value: foo`},
		{"Nested - map - obj - missing item - valid", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: OBJECT, Attributes: Attributes{}}}}},
			``},
		{"Nested - map - map - misplaced attrs", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: MAP, Attributes: Attributes{}}}}},
			`"model.m.item" must not have "attributes"`},
		{"Nested - map - array - misplaced attrs", Model{
			Attributes: Attributes{"m": {Name: "m", Type: MAP,
				Item: &Item{Type: ARRAY, Attributes: Attributes{}}}}},
			`"model.m.item" must not have "attributes"`},

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
		err := test.model.Verify()
		if test.err == "" && err != nil {
			t.Fatalf("ModelVerify: %s - should have worked, got: %s",
				test.name, err)
		}
		if test.err != "" && err == nil {
			t.Fatalf("ModelVerify: %s - should have failed with: %s",
				test.name, test.err)
		}
		if err != nil && test.err != err.Error() {
			t.Fatalf("ModifyVerify: %s:\nExp: %s\nGot: %s", test.name,
				test.err, err.Error())
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
			`"model.x" is not a scalar, so "enum" is not allowed`},
		{"empty enum - array", Model{Attributes: Attributes{
			"x": {Name: "x", Type: ARRAY, Enum: []any{1}}}},
			`"model.x" is not a scalar, so "enum" is not allowed`},
		{"empty enum - map", Model{Attributes: Attributes{
			"x": {Name: "x", Type: MAP, Enum: []any{1}}}},
			`"model.x" is not a scalar, so "enum" is not allowed`},
		{"empty enum - any", Model{Attributes: Attributes{
			"x": {Name: "x", Type: ANY, Enum: []any{}}}},
			`"model.x" specifies an "enum" but it is empty`},

		{"enum - bool - true ", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true}}}}, ""},
		{"enum - bool 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true, false}}}}, ""},
		{"enum - bool string", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{true, ""}}}},
			`"model.x" enum value "" must be of type "boolean"`},
		{"enum - bool float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{5.5}}}},
			`"model.x" enum value "5.5" must be of type "boolean"`},
		{"enum - bool map", Model{Attributes: Attributes{
			"x": {Name: "x", Type: BOOLEAN, Enum: []any{map[string]string{}}}}},
			`"model.x" enum value "map[]" must be of type "boolean"`},

		{"enum - decimal 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{5.5}}}}, ""},
		{"enum - decimal 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{5.5, 2}}}}, ""},
		{"enum - decimal bool", Model{Attributes: Attributes{
			"x": {Name: "x", Type: DECIMAL, Enum: []any{true, 5}}}},
			`"model.x" enum value "true" must be of type "decimal"`},

		{"enum - integer 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{1}}}}, ""},
		{"enum - integer 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{-1, 1}}}}, ""},
		{"enum - integer float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{-1, 1, 3.1}}}},
			`"model.x" enum value "3.1" must be of type "integer"`},
		{"enum - integer float", Model{Attributes: Attributes{
			"x": {Name: "x", Type: INTEGER, Enum: []any{[]int{}}}}},
			`"model.x" enum value "[]" must be of type "integer"`},

		{"enum - string 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a"}}}}, ""},
		{"enum - string 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", ""}}}}, ""},
		{"enum - string int", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", 0}}}},
			`"model.x" enum value "0" must be of type "string"`},
		{"enum - string struct", Model{Attributes: Attributes{
			"x": {Name: "x", Type: STRING, Enum: []any{"a", struct{}{}}}}},
			`"model.x" enum value "{}" must be of type "string"`},

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
			`"model.x" enum value "bad" must be of type "timestamp"`},
		{"enum - timestamp type", Model{Attributes: Attributes{
			"x": {Name: "x", Type: TIMESTAMP,
				Enum: []any{"2024-01-02T12:01:02Z", 5.5}}}},
			`"model.x" enum value "5.5" must be of type "timestamp"`},

		{"enum - uint 1", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER, Enum: []any{1}}}}, ""},
		{"enum - uint 2", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER, Enum: []any{1, 2}}}},
			""},
		{"enum - uint bad", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER, Enum: []any{2, -1}}}},
			`"model.x" enum value "-1" must be of type "uinteger"`},
		{"enum - uint type", Model{Attributes: Attributes{
			"x": {Name: "x", Type: UINTEGER,
				Enum: []any{5.5}}}},
			`"model.x" enum value "5.5" must be of type "uinteger"`},

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
		err := test.model.Verify()
		if test.err == "" && err != nil {
			t.Fatalf("ModelVerify: %s - should have worked, got: %s",
				test.name, err)
		}
		if test.err != "" && err == nil {
			t.Fatalf("ModelVerify: %s - should have failed with: %s",
				test.name, test.err)
		}
		if err != nil && test.err != err.Error() {
			t.Fatalf("ModifyVerify: %s\nExp: %s\nGot: %s", test.name,
				test.err, err.Error())
		}
	}
}

func TestGetModelSerializer(t *testing.T) {
	type Match struct {
		format string
		result ModelSerializer
	}
	type Format struct {
		format string
		fn     ModelSerializer
	}

	fn1 := func(m *Model, format string) ([]byte, error) { return nil, nil }
	fn2 := func(m *Model, format string) ([]byte, error) { return nil, nil }
	fn3 := func(m *Model, format string) ([]byte, error) { return nil, nil }
	fn4 := func(m *Model, format string) ([]byte, error) { return nil, nil }

	type Test struct {
		formats []Format
		matches []Match
	}

	tests := []Test{
		{
			formats: []Format{},
			matches: []Match{
				{format: "foo", result: nil},
			},
		},
		{
			formats: []Format{
				{format: "f1", fn: fn1},
			},
			matches: []Match{
				{format: "f1", result: fn1},
				{format: "f1/v1", result: nil},
				{format: "f2", result: nil},
				{format: "f2/v1", result: nil},
			},
		},
		{
			formats: []Format{
				{format: "f1", fn: fn1},
				{format: "f1/v1", fn: fn2},
				{format: "f1/v2", fn: fn3},
			},
			matches: []Match{
				{format: "f1", result: fn1},
				{format: "f1/v1", result: fn2},
				{format: "f1/v2", result: fn3},
				{format: "f1/vx", result: nil},
			},
		},
		{
			formats: []Format{
				{format: "f1/v1", fn: fn2},
				{format: "f1/v2", fn: fn3},
			},
			matches: []Match{
				{format: "f1", result: fn3},
				{format: "f1/v1", result: fn2},
				{format: "f1/v2", result: fn3},
				{format: "f1/vx", result: nil},
			},
		},
		{
			formats: []Format{
				{format: "f1/v1", fn: fn1},
				{format: "f1/v2", fn: fn2},
				{format: "f2/va", fn: fn3},
				{format: "f2/vb", fn: fn4},
			},
			matches: []Match{
				{format: "f1", result: fn2},
				{format: "f1/v1", result: fn1},
				{format: "f1/v2", result: fn2},
				{format: "f1/vx", result: nil},
				{format: "f1vx", result: nil},
				{format: "f2", result: fn3},
				{format: "f2/va", result: fn3},
				{format: "f2/vb", result: fn4},
				{format: "f2/vx", result: nil},
				{format: "f2vx", result: nil},
			},
		},
	}

	for _, test := range tests {
		// Start out clean
		ModelSerializers = map[string]ModelSerializer{}
		for _, f := range test.formats {
			RegisterModelSerializer(f.format, f.fn)
		}
		for _, m := range test.matches {
			fn := GetModelSerializer(m.format)
			if reflect.ValueOf(fn).Pointer() == reflect.ValueOf(m.result).Pointer() {
				continue
			}
			if fn == nil {
				t.Fatalf("Fn is nil for match: %q, Test:\n%s",
					m.format, ToJSON(test.formats))
			}
			if fn != nil && m.result == nil {
				t.Fatalf("Fn should have been nil for match: %q, Test:\n%s",
					m.format, ToJSON(test.formats))
			}
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
	a58 := a50 + "12345678"
	a59 := a50 + "123456789"
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
		{"", `Invalid model type name "", must match: ` + match},
		{"A", `Invalid model type name "A", must match: ` + match},
		{"*", `Invalid model type name "*", must match: ` + match},
		{"@", `Invalid model type name "@", must match: ` + match},
		{"0", `Invalid model type name "0", must match: ` + match},
		{"0a", `Invalid model type name "0a", must match: ` + match},
		{"aZ", `Invalid model type name "aZ", must match: ` + match},
		{a59, `Invalid model type name "` + a59 + `", must match: ` + match},
		{"a", ``},
		{"_", ``},
		{"_a", ``},
		{"_8", ``},
		{"a_", ``},
		{"a_8", ``},
		{"aa", ``},
		{"a9", ``},
		{a58, ``},
	} {
		err := IsValidModelName(test.input)
		got := ""
		if err != nil {
			got = err.Error()
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
		{"", `Invalid attribute name "", must match: ` + match},
		{"A", `Invalid attribute name "A", must match: ` + match},
		{"*", `Invalid attribute name "*", must match: ` + match},
		{"@", `Invalid attribute name "@", must match: ` + match},
		{"0", `Invalid attribute name "0", must match: ` + match},
		{"0a", `Invalid attribute name "0a", must match: ` + match},
		{"aZ", `Invalid attribute name "aZ", must match: ` + match},
		{a64, `Invalid attribute name "` + a64 + `", must match: ` + match},
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
		err := IsValidAttributeName(test.input)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if got != test.result {
			t.Fatalf("Test: %s\nExp: %s\nGot: %s", test.input, test.result, got)
		}
	}

	// Test IDs
	match = RegexpID.String()
	for _, test := range []struct {
		input  string
		result string
	}{
		{"", `Invalid ID "", must match: ` + match},
		{"*", `Invalid ID "*", must match: ` + match},
		{"!", `Invalid ID "!", must match: ` + match},
		{"+", `Invalid ID "+", must match: ` + match},
		{"A*", `Invalid ID "A*", must match: ` + match},
		{"*a", `Invalid ID "*a", must match: ` + match},
		{a129, `Invalid ID "` + a129 + `", must match: ` + match},
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
		{" a", `Invalid ID " a", must match: ` + match},
		{".", `Invalid ID ".", must match: ` + match},
		{"-", `Invalid ID "-", must match: ` + match},
		{"~", `Invalid ID "~", must match: ` + match},
		{"@", `Invalid ID "@", must match: ` + match},
		{"Z.-~_0Nb", ``},
		{a128, ``},
	} {
		err := IsValidID(test.input)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if got != test.result {
			t.Fatalf("Test: %s\nExp: %s\nGot: %s", test.input, test.result, got)
		}
	}

	// Test map keys
	match = RegexpMapKey.String()
	for _, test := range []struct {
		input  string
		result string
	}{
		{"", `Invalid map key name "", must match: ` + match},
		{"_", `Invalid map key name "_", must match: ` + match},
		{".", `Invalid map key name ".", must match: ` + match},
		{"-", `Invalid map key name "-", must match: ` + match},
		{"*", `Invalid map key name "*", must match: ` + match},
		{"!", `Invalid map key name "!", must match: ` + match},
		{"~", `Invalid map key name "~", must match: ` + match},
		{"A", `Invalid map key name "A", must match: ` + match},
		{"aA", `Invalid map key name "aA", must match: ` + match},
		{"Aa", `Invalid map key name "Aa", must match: ` + match},
		{"_a", `Invalid map key name "_a", must match: ` + match},
		{"9A", `Invalid map key name "9A", must match: ` + match},
		{"a*", `Invalid map key name "a*", must match: ` + match},
		{"a!", `Invalid map key name "a!", must match: ` + match},
		{"a~", `Invalid map key name "a~", must match: ` + match},
		{":a", `Invalid map key name ":a", must match: ` + match},
		{a64, `Invalid map key name "` + a64 + `", must match: ` + match},

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
		err := IsValidMapKey(test.input)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if got != test.result {
			t.Fatalf("Test: %s\nExp: %s\nGot: %s", test.input, test.result, got)
		}
	}
}
