package common

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseUI(t *testing.T) {
	type Test struct {
		In  string
		Exp string
	}

	tests := []Test{
		{"", `{null}`},
		{"prop", `{[{"prop",-1}]}`},
		{"*", `{[{"*",-1}]}`},
		{"#prop", `{[{"#prop",-1}]}`},
		{"_", `{[{"_",-1}]}`},
		{"_1", `{[{"_1",-1}]}`},

		{"['prop']", `{[{"prop",-1}]}`},
		{"['prop1.prop2']", `{[{"prop1.prop2",-1}]}`},
		{"['prop1-prop2']", `{[{"prop1-prop2",-1}]}`},
		{"['1']", `{[{"1",-1}]}`},

		{"a1.*", `{[{"a1",-1},{"*",-1}]}`},
		{"a1.a2", `{[{"a1",-1},{"a2",-1}]}`},
		{"a1.a2.a3", `{[{"a1",-1},{"a2",-1},{"a3",-1}]}`},
		{"a1.2", `{[{"a1",-1},{"2",-1}]}`},

		{"a1['*']", `{[{"a1",-1},{"*",-1}]}`},
		{"a1['a2']", `{[{"a1",-1},{"a2",-1}]}`},
		{"a1['2']", `{[{"a1",-1},{"2",-1}]}`},

		{"a1[2]", `{[{"a1",-1},{"2",2}]}`},
		{"a1['*'].a3", `{[{"a1",-1},{"*",-1},{"a3",-1}]}`},
		{"a1['a2'].a3", `{[{"a1",-1},{"a2",-1},{"a3",-1}]}`},
		{"a1['a2'].3", `{[{"a1",-1},{"a2",-1},{"3",-1}]}`},
		{"a1['a2']['a3']", `{[{"a1",-1},{"a2",-1},{"a3",-1}]}`},
		{"a1[2].a3", `{[{"a1",-1},{"2",2},{"a3",-1}]}`},
		{"a1[2]['a3']", `{[{"a1",-1},{"2",2},{"a3",-1}]}`},
		{"a1[2]['a3']", `{[{"a1",-1},{"2",2},{"a3",-1}]}`},
		{"a1[2][3]", `{[{"a1",-1},{"2",2},{"3",3}]}`},

		// Errors
		{".prop", `Unexpected . in ".prop" at pos 1`},
		{".*", `Unexpected . in ".*" at pos 1`},
		{"1", `Unexpected 1 in "1" at pos 1`},
		{"_#1", `Unexpected # in "_#1" at pos 2`},
		{"a1..a2", `Unexpected . in "a1..a2" at pos 4`},
		{"a1.[a]", `Unexpected [ in "a1.[a]" at pos 4`},
		{"[2]", `Unexpected 2 in "[2]" at pos 2`},
		{"[prop1.prop2]", `Expecting a ' at pos 2 in "[prop1.prop2]"`},

		{"a1.", `Unexpected end of property in "a1."`},
		{"*.", `Unexpected end of property in "*."`},
		{"a1['a", `Unexpected end of property in "a1['a"`},
		{"a1['a'", `Unexpected end of property in "a1['a'"`},
		{"a1[1", `Unexpected end of property in "a1[1"`},

		{"a1[]", `Unexpected ] in "a1[]" at pos 4`},
		{"a1['']", `Unexpected ' in "a1['']" at pos 5`},
		{"a1[']", `Unexpected ] in "a1[']" at pos 5`},
	}

	for _, test := range tests {
		pp, err := PropPathFromUI(test.In)
		res := ""
		if pp != nil {
			tmp, _ := json.Marshal(pp)
			res = string(tmp)
			res = strings.ReplaceAll(res, `"Parts":`, "")
			res = strings.ReplaceAll(res, `"Text":`, "")
			res = strings.ReplaceAll(res, `"Index":`, "")
		} else {
			res = err.Error()
		}
		if res != test.Exp {
			t.Logf("Test: %q:\nExp: %s\nGot: %s\n", test.In, test.Exp, res)
			t.Fail()
		}
	}
}

func TestAbstract(t *testing.T) {
	type Test struct {
		P   *PropPath
		Exp string
	}

	tests := []Test{
		{NewPPP("one"), "one"},
		{NewPPP("one").P("two"), "one,two"},
		{NewPPP("one").I(2), "one#2"},
		{NewPPP("one").I(2).P("two"), "one#2,two"},
	}

	for _, test := range tests {
		res := test.P.Abstract()
		if res != test.Exp {
			t.Fatalf("Test: %s\nExp: %s\nGot: %s", test.P.UI(), test.Exp, res)
		}
	}
}

/*
func TestObjectSetProp(t *testing.T) {
	type Test struct {
		In    map[string]any
		Prop  string
		Value any
		Exp   map[string]any
	}

	tests := []Test{
		{
			In:    map[string]any{},
			Prop:  "foo",
			Value: "bar",
			Exp:   map[string]any{"foo": "bar"},
		},
		{
			In:    map[string]any{},
			Prop:  "foo",
			Value: 5,
			Exp:   map[string]any{"foo": 5},
		},
		{
			In:    map[string]any{},
			Prop:  "foo",
			Value: map[string]any{"bar": "car"},
			Exp:   map[string]any{"foo": map[string]any{"bar": "car"}},
		},
		{
			In:    nil,
			Prop:  "foo.bar",
			Value: "rat",
			Exp:   map[string]any{"foo": map[string]any{"bar": "rat"}},
		},
		{
			In:    nil,
			Prop:  "foo.bar",
			Value: nil,
			Exp:   map[string]any{"foo": map[string]any{}},
		},
		{
			In:    map[string]any{},
			Prop:  "foo[0]",
			Value: "bar",
			Exp:   map[string]any{"foo": []any{"bar"}},
		},
		{
			In:    map[string]any{},
			Prop:  "foo[1]",
			Value: "bar",
			Exp:   map[string]any{"foo": []any{nil, "bar"}},
		},
		{
			In:    map[string]any{"foo": []any{nil, "bar"}},
			Prop:  "foo[1]",
			Value: map[string]any{"bar": "foo"},
			Exp:   map[string]any{"foo": []any{nil, map[string]any{"bar": "foo"}}},
		},
		{
			In:    nil,
			Prop:  "foo[1].bar",
			Value: 5,
			Exp:   map[string]any{"foo": []any{nil, map[string]any{"bar": 5}}},
		},
		{
			In:    map[string]any{},
			Prop:  "foo[1].bar",
			Value: 5,
			Exp:   map[string]any{"foo": []any{nil, map[string]any{"bar": 5}}},
		},
	}

	in := map[string]any{}
	for i, test := range tests {
		if test.In != nil {
			// Don't use result from previous test
			in = test.In
		}
		pp, _ := PropPathFromUI(test.Prop)
		err := ObjectSetProp(in, pp, test.Value)
		if err != nil {
			t.Fatalf("Test(%d): %s - Err: %s", i, test.Prop, err)
		}
		if !reflect.DeepEqual(test.Exp, in) {
			t.Fatalf("Test(%d): %s\nExp: %s\nGot: %s",
				i, test.Prop, ToJSON(test.Exp), ToJSON(in))
		}
	}
}
*/
