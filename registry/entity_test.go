package registry

import (
	"testing"

	. "github.com/xregistry/server/common"
)

func TestSetProp(t *testing.T) {

	type Obj map[string]any

	type Test struct {
		Name   string
		Start  Obj
		Prop   string
		Value  any
		Result Obj
		Error  string
	}

	tests := []Test{
		{
			Name:  "top - int",
			Start: Obj{},
			Prop:  "myint",
			Value: 5,
			Result: Obj{
				"myint": 5,
			},
			Error: "",
		},
		{
			Name:   "top - int - null",
			Start:  nil, // continue from prev
			Prop:   "myint",
			Value:  nil,
			Result: Obj{},
			Error:  "",
		},
		{
			Name:   "top - string",
			Prop:   "mystr",
			Value:  "hello",
			Result: Obj{"mystr": "hello"},
			Error:  "",
		},
		{
			Name:   "top - string - null",
			Prop:   "mystr",
			Value:  nil,
			Result: Obj{},
			Error:  "",
		},
		{
			Name:   "top - string - null again",
			Prop:   "mystr",
			Value:  nil,
			Result: Obj{},
			Error:  "",
		},

		// array
		{
			Name:   "top - int array",
			Prop:   "myarray[0]",
			Value:  5,
			Result: Obj{"myarray": []any{5}},
			Error:  "",
		},
		{
			Name:   "top - int array - 2",
			Prop:   "myarray[1]",
			Value:  55,
			Result: Obj{"myarray": []any{5, 55}},
			Error:  "",
		},
		{
			Name:   "top - int array - 3",
			Prop:   "myarray[3]",
			Value:  555,
			Result: Obj{"myarray": []any{5, 55, nil, 555}},
			Error:  "",
		},
		{
			Name:   "top - int array - 4",
			Prop:   "myarray[0]",
			Value:  nil,
			Result: Obj{"myarray": []any{nil, 55, nil, 555}},
			Error:  "",
		},
		{
			Name:   "top - int array - empty",
			Prop:   "myarray",
			Value:  []any{},
			Result: Obj{"myarray": []any{}},
			Error:  "",
		},
		{
			Name:   "top - int array - nil",
			Prop:   "myarray",
			Value:  nil,
			Result: Obj{},
			Error:  "",
		},

		// map
		{
			Name:   "top - int map",
			Start:  Obj{},
			Prop:   "mymap.myint",
			Value:  5,
			Result: Obj{"mymap": Obj{"myint": 5}},
			Error:  "",
		},
		{
			Name:   "top - int map 2",
			Prop:   "mymap.myint2",
			Value:  55,
			Result: Obj{"mymap": Obj{"myint": 5, "myint2": 55}},
			Error:  "",
		},
		{
			Name:   "top - int map - 3",
			Prop:   "mymap.myint",
			Value:  10,
			Result: Obj{"mymap": Obj{"myint": 10, "myint2": 55}},
			Error:  "",
		},
		{
			Name:   "top - int map - 4",
			Prop:   "mymap.myint",
			Value:  nil,
			Result: Obj{"mymap": Obj{"myint2": 55}},
			Error:  "",
		},
		{
			Name:   "top - int map - empty",
			Prop:   "mymap",
			Value:  Obj{},
			Result: Obj{"mymap": Obj{}},
			Error:  "",
		},
		{
			Name:   "top - int map - nil",
			Prop:   "mymap",
			Value:  nil,
			Result: Obj{},
			Error:  "",
		},

		// object
		{
			Name:   "top - object",
			Start:  Obj{},
			Prop:   "myobject.myint",
			Value:  5,
			Result: Obj{"myobject": Obj{"myint": 5}},
			Error:  "",
		},
		{
			Name:   "top - object 2",
			Prop:   "myobject.myint2",
			Value:  55,
			Result: Obj{"myobject": Obj{"myint": 5, "myint2": 55}},
			Error:  "",
		},
		{
			Name:   "top - object - 3",
			Prop:   "myobject.mystr",
			Value:  "hello",
			Result: Obj{"myobject": Obj{"myint": 5, "myint2": 55, "mystr": "hello"}},
			Error:  "",
		},
		{
			Name:   "top - object - 4",
			Prop:   "myobject.myint",
			Value:  nil,
			Result: Obj{"myobject": Obj{"myint2": 55, "mystr": "hello"}},
			Error:  "",
		},
		{
			Name:   "top - object - empty",
			Prop:   "myobject",
			Value:  Obj{},
			Result: Obj{"myobject": Obj{}},
			Error:  "",
		},
		{
			Name:   "top - object - nil",
			Prop:   "myobject",
			Value:  nil,
			Result: Obj{},
			Error:  "",
		},

		// nested
		{
			Name:   "top - nested - int",
			Prop:   "myint",
			Value:  5,
			Result: Obj{"myint": 5},
			Error:  "",
		},
		{
			Name:   "top - nested - obj",
			Prop:   "myobj.nest.nestarray[1].deepint",
			Value:  666,
			Result: Obj{"myint": 5, "myobj": Obj{"nest": Obj{"nestarray": []any{nil, Obj{"deepint": 666}}}}},
			Error:  "",
		},
		{
			Name:   "top - nested - obj - add",
			Prop:   "myobj.nest.nestarray[1].deepstr",
			Value:  "hi",
			Result: Obj{"myint": 5, "myobj": Obj{"nest": Obj{"nestarray": []any{nil, Obj{"deepint": 666, "deepstr": "hi"}}}}},
			Error:  "",
		},
		{
			Name:   "top - nested - obj - nil",
			Prop:   "myobj.nest.nestarray[1].deepstr",
			Value:  nil,
			Result: Obj{"myint": 5, "myobj": Obj{"nest": Obj{"nestarray": []any{nil, Obj{"deepint": 666}}}}},
			Error:  "",
		},
		{
			Name:   "top - nested - obj - nil",
			Prop:   "myobj.nest.nestarray[1]",
			Value:  nil,
			Result: Obj{"myint": 5, "myobj": Obj{"nest": Obj{"nestarray": []any{}}}},
			Error:  "",
		},
		{
			Name:   "top - nested - obj - erase",
			Prop:   "myobj.nest.nestarray",
			Value:  nil,
			Result: Obj{"myint": 5, "myobj": Obj{"nest": Obj{}}},
			Error:  "",
		},
		{
			Name:   "top - nested - obj - erase - 2",
			Prop:   "myobj",
			Value:  nil,
			Result: Obj{"myint": 5},
			Error:  "",
		},
		{
			Name:   "top - nested - obj - erase - 3",
			Prop:   "",
			Value:  nil,
			Result: Obj{},
			Error:  "",
		},
	}

	start := map[string]any{}

	for _, test := range tests {
		pp, err := PropPathFromUI(test.Prop)
		if err != nil {
			t.Errorf("Error in test prep %q: %s(%s)", test.Name, test.Prop, err)
			t.FailNow()
		}

		obj := test.Start
		if IsNil(obj) {
			obj = start
		}

		t.Logf("Test: %s", test.Name)
		t.Logf("  Prop: %s", test.Prop)
		t.Logf("  Obj: %s", ToJSON(obj))

		err = ObjectSetProp(obj, pp, test.Value)

		if (err == nil && test.Error != "") ||
			(err != nil && test.Error != err.Error()) {

			t.Errorf("Test: %s - should fail with: %s", test.Name, test.Error)
			if err != nil {
				t.Logf("Got: %s", err)
			}
			t.FailNow()
		}

		if err != nil && test.Error == "" {
			t.Errorf("Test: %s - failed with: %s", test.Name, err)
			t.FailNow()
		}

		exp := ToJSON(test.Result)
		got := ToJSON(obj)
		if got != exp {
			t.Errorf("Test: %s:\nExp: %s\nGot: %s\n", test.Name, exp, got)
			t.FailNow()
		}

		start = obj
	}
}

func TestGetProp(t *testing.T) {
	type Obj map[string]any

	type Test struct {
		Name   string
		Start  Obj
		Prop   string
		Result any
		Error  string
	}

	tests := []Test{
		{
			Name:   "{} - empty",
			Start:  Obj{},
			Prop:   "",
			Result: Obj{},
			Error:  "",
		},
		{
			Name:   "{} - prop",
			Start:  Obj{},
			Prop:   "prop",
			Result: nil,
			Error:  "",
		},
		{
			Name: "full - missing prop",
			Start: Obj{
				"int":      5,
				"string":   "hello",
				"decimal":  4.4,
				"bool":     true,
				"emptyobj": Obj{},
				"obj": Obj{
					"int":       123,
					"array_int": []int{1, 2, 3},
				},
				"emptyarrayobj": []Obj{},
				"arrayobj": []Obj{
					Obj{},
					Obj{"int": 321},
				},
			},
			Prop:   "prop",
			Result: nil,
			Error:  "",
		},
		{Name: "int", Prop: "int", Result: 5},
		{Name: "string", Prop: "string", Result: "hello"},
		{Name: "bool", Prop: "bool", Result: true},
		{Name: "emptyobj", Prop: "emptyobj", Result: Obj{}},

		{Name: "obj", Prop: "obj", Result: Obj{
			"int":       123,
			"array_int": []int{1, 2, 3},
		}},
		{Name: "obj.nada", Prop: "obj.nada", Result: nil},
		{Name: "obj.nada[2]", Prop: "obj.nada[2]",
			Error: "Can't traverse into nothing: obj.nada"},
		{Name: "obj.nada.xxx", Prop: "obj.nada.xxx",
			Error: "Can't traverse into nothing: obj.nada"},
		{Name: "obj.int", Prop: "obj.int", Result: 123},
		{Name: "obj.array", Prop: "obj.array_int", Result: []int{1, 2, 3}},
		{Name: "obj.array_int", Prop: "obj.array_int[1]", Result: 2},

		{Name: "obj.array_int.0", Prop: "obj.array_int[0]", Result: 1},
		{Name: "obj.array_int.3", Prop: "obj.array_int[3]",
			Error: "Array reference \"obj.array_int[3]\" out of bounds: (max:3-1)"},
		{Name: "emptyao", Prop: "emptyarrayobj", Result: []any{}},
		{Name: "arrayobj", Prop: "arrayobj", Result: []Obj{
			Obj{},
			Obj{"int": 321},
		}},
		{Name: "arrayobj[0]", Prop: "arrayobj[0]", Result: Obj{}},
		{Name: "arrayobj[1].int", Prop: "arrayobj[1].int", Result: 321},
		{Name: "arrayobj[1].xxx", Prop: "arrayobj[1].xxx", Result: nil},
		{Name: "arrayobj[3].xxx", Prop: "arrayobj[3].xxx", Error: "Array reference \"arrayobj[3]\" out of bounds: (max:2-1)"},
	}

	start := Obj{}

	for _, test := range tests {
		t.Logf("Test: %s   Prop: %s", test.Name, test.Prop)
		pp, err := PropPathFromUI(test.Prop)
		Must(err)

		obj := test.Start
		if IsNil(obj) {
			obj = start
		}

		val, _, err := ObjectGetProp(obj, pp)

		if (err == nil && test.Error != "") ||
			(err != nil && test.Error != err.Error()) {

			t.Errorf("Should fail with: %s", test.Error)
			if err != nil {
				t.Logf("Got: %s", err)
			}
			t.FailNow()
		}
		if err != nil && test.Error == "" {
			t.Errorf("Failed with: %s", err)
			t.FailNow()
		}

		exp := ToJSON(test.Result)
		got := ToJSON(val)
		if got != exp {
			t.Errorf("\nExp: %s\nGot: %s\n", exp, got)
			t.FailNow()
		}

		start = obj
	}
}

func TestGetPropUpdate(t *testing.T) {
	obj := Object{
		"attr1": "foo",
		"attr2": nil,
	}

	val, ok, err := ObjectGetProp(obj, NewPPP("attr1"))
	if val != "foo" || ok != true || err != nil {
		t.Logf("val: %v  ok: %v  err: %v", val, ok, err)
		t.Fatalf("attr1 failed")
	}

	val, ok, err = ObjectGetProp(obj, NewPPP("attr2"))
	if val != nil || ok != true || err != nil {
		t.Logf("val: %v  ok: %v  err: %v", val, ok, err)
		t.Fatalf("attr2 failed")
	}

	val, ok, err = ObjectGetProp(obj, NewPPP("attr3"))
	if val != nil || ok == true || err != nil {
		t.Logf("val: %v  ok: %v  err: %v", val, ok, err)
		t.Fatalf("attr3 failed")
	}

	val, ok, err = ObjectGetProp(obj, NewPPP("attr4").P("foo"))
	if val != nil || ok == true || err == nil ||
		err.Error() != "Can't traverse into nothing: attr4" {

		t.Logf("val: %v  ok: %v  err: %v", val, ok, err)
		t.Fatalf("attr4.foo failed")
	}

}
