package tests

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestBasicTypes(t *testing.T) {
	reg := NewRegistry("TestBasicTypes")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttr("regbool1", registry.BOOLEAN)
	reg.Model.AddAttr("regbool2", registry.BOOLEAN)
	reg.Model.AddAttr("regdec1", registry.DECIMAL)
	reg.Model.AddAttr("regdec2", registry.DECIMAL)
	reg.Model.AddAttr("regdec3", registry.DECIMAL)
	reg.Model.AddAttr("regdec4", registry.DECIMAL)
	reg.Model.AddAttr("regint1", registry.INTEGER)
	reg.Model.AddAttr("regint2", registry.INTEGER)
	reg.Model.AddAttr("regint3", registry.INTEGER)
	reg.Model.AddAttr("regstring1", registry.STRING)
	reg.Model.AddAttr("regstring2", registry.STRING)
	reg.Model.AddAttr("reguint1", registry.UINTEGER)
	reg.Model.AddAttr("reguint2", registry.UINTEGER)
	reg.Model.AddAttr("regtime1", registry.TIME)

	reg.Model.AddAttr("reganyarrayint", registry.ANY)
	reg.Model.AddAttr("reganyarrayobj", registry.ANY)
	reg.Model.AddAttr("reganyint", registry.ANY)
	reg.Model.AddAttr("reganystr", registry.ANY)
	reg.Model.AddAttr("reganyobj", registry.ANY)

	reg.Model.AddAttribute(&registry.Attribute{
		Name: "regarrayarrayint",
		Type: registry.ARRAY,
		Item: &registry.Item{
			Type: registry.ARRAY,
			Item: &registry.Item{
				Type: registry.INTEGER,
			},
		},
	})

	reg.Model.AddAttribute(&registry.Attribute{
		Name: "regarrayint",
		Type: registry.ARRAY,
		Item: &registry.Item{Type: registry.INTEGER},
	})

	reg.Model.AddAttribute(&registry.Attribute{
		Name: "regmapint",
		Type: registry.MAP,
		Item: &registry.Item{Type: registry.INTEGER},
	})
	reg.Model.AddAttribute(&registry.Attribute{
		Name: "regmapstring",
		Type: registry.MAP,
		Item: &registry.Item{Type: registry.STRING},
	})
	reg.Model.AddAttribute(&registry.Attribute{
		Name: "regobj",
		Type: registry.OBJECT,
		Item: &registry.Item{
			Attributes: map[string]*registry.Attribute{
				"objbool": &registry.Attribute{
					Name: "objbool",
					Type: registry.BOOLEAN,
				},
				"objint": &registry.Attribute{
					Name: "objint",
					Type: registry.INTEGER,
				},
				"objobj": &registry.Attribute{
					Name: "objobj",
					Type: registry.OBJECT,
					Item: &registry.Item{
						Attributes: map[string]*registry.Attribute{
							"ooint": &registry.Attribute{
								Name: "ooint",
								Type: registry.INTEGER,
							},
						},
					},
				},
				"objstr": &registry.Attribute{
					Name: "objstr",
					Type: registry.STRING,
				},
			},
		},
	})

	// TODO - do we need this?
	reg.Model.Save()

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddAttr("dirbool1", registry.BOOLEAN)
	gm.AddAttr("dirbool2", registry.BOOLEAN)
	gm.AddAttr("dirdec1", registry.DECIMAL)
	gm.AddAttr("dirdec2", registry.DECIMAL)
	gm.AddAttr("dirdec3", registry.DECIMAL)
	gm.AddAttr("dirdec4", registry.DECIMAL)
	gm.AddAttr("dirint1", registry.INTEGER)
	gm.AddAttr("dirint2", registry.INTEGER)
	gm.AddAttr("dirint3", registry.INTEGER)
	gm.AddAttr("dirstring1", registry.STRING)
	gm.AddAttr("dirstring2", registry.STRING)

	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	rm.AddAttr("filebool1", registry.BOOLEAN)
	rm.AddAttr("filebool2", registry.BOOLEAN)
	rm.AddAttr("filedec1", registry.DECIMAL)
	rm.AddAttr("filedec2", registry.DECIMAL)
	rm.AddAttr("filedec3", registry.DECIMAL)
	rm.AddAttr("filedec4", registry.DECIMAL)
	rm.AddAttr("fileint1", registry.INTEGER)
	rm.AddAttr("fileint2", registry.INTEGER)
	rm.AddAttr("fileint3", registry.INTEGER)
	rm.AddAttr("filestring1", registry.STRING)
	rm.AddAttr("filestring2", registry.STRING)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1")

	// /dirs/d1/f1/v1

	type Prop struct {
		Name   string
		Value  any
		ErrMsg string
	}

	type Test struct {
		Entity registry.EntitySetter // any //  *registry.Entity
		Props  []Prop
	}

	tests := []Test{
		Test{reg, []Prop{
			{"regarrayarrayint[1][1]", 66, ""},
			{"regarrayint[0]", 1, ""},
			{"regarrayint[2]", 3, ""},
			{"regarrayint[1]", 2, ""},
			{"regbool1", true, ""},
			{"regbool2", false, ""},
			{"regdec1", 123.5, ""},
			{"regdec2", -123.5, ""},
			{"regdec3", 124.0, ""},
			{"regdec4", 0.0, ""},
			{"regint1", 123, ""},
			{"regint2", -123, ""},
			{"regint3", 0, ""},
			{"regmapint.k1", 123, ""},
			{"regmapint.k2", 234, ""},
			{"regmapstring.k1", "v1", ""},
			{"regmapstring.k2", "v2", ""},
			{"regstring1", "str1", ""},
			{"regstring2", "", ""},
			{"regtime1", "2006-01-02T15:04:05Z", ""},
			{"reguint1", 0, ""},
			{"reguint2", 333, ""},

			{"reganyarrayint[2]", 5, ""},
			{"reganyarrayobj[2].int1", 55, ""},
			{"reganyarrayobj[2].myobj.int2", 555, ""},
			{"reganyarrayobj[0]", -5, ""},
			{"reganyint", 123, ""},
			{"reganyobj.int", 345, ""},
			{"reganyobj.str", "substr", ""},
			{"reganystr", "mystr", ""},
			{"regobj.objbool", true, ""},
			{"regobj.objint", 345, ""},
			{"regobj.objobj.ooint", 999, ""},
			{"regobj.objstr", "in1", ""},

			{"reganyobj.nestobj.int", 123, ""},

			// Syntax checking
			// {"MiXeD", 123, ""},
			{"regarrayint[~abc]", 123,
				`Unexpected ~ in "regarrayint[~abc]" at pos 13`},
			{"regarrayint['~abc']", 123,
				`Unexpected ~ in "regarrayint['~abc']" at pos 14`},
			{"regmapstring.~abc", 123,
				`Unexpected ~ in "regmapstring.~abc" at pos 14`},
			{"regmapstring[~abc]", 123,
				`Unexpected ~ in "regmapstring[~abc]" at pos 14`},
			{"regmapstring['~abc']", 123,
				`Unexpected ~ in "regmapstring['~abc']" at pos 15`},

			// Type checking
			{"epoch", -123,
				`"-123" should be an uinteger`}, // bad uint
			{"reganyobj2.str", "substr",
				`Can't find attribute "reganyobj2.str"`}, // unknown attr
			{"regarrayarrayint[0][0]", "abc",
				`"abc" should be an integer`}, // bad type
			{"regarrayint[2]", "abc",
				`"abc" should be an integer`}, // bad type
			{"regbool1", "123",
				`"123" should be a boolean`}, // bad type
			{"regdec1", "123",
				`"123" should be a decimal`}, // bad type
			{"regint1", "123",
				`"123" should be an integer`}, // bad type
			{"regmapint", "123",
				`Unsupported type: map`}, // bad type
			{"regmapint.k1", "123",
				`"123" should be an integer`}, // bad type
			{"regmapstring.k1", 123,
				`"123" should be a string`}, // bad type
			{"regstring1", 123,
				`"123" should be a string`}, // bad type
			{"regtime1", "not a time",
				`Malformed timestamp "not a time": parsing time "not a time" as "2006-01-02T15:04:05Z07:00": cannot parse "not a time" as "2006"`}, // bad date format
			{"reguint1", -1,
				`"-1" should be an uinteger`}, // bad uint
			{"unknown_int", 123,
				`Can't find attribute "unknown_int"`}, // unknown attr
			{"unknown_str", "error",
				`Can't find attribute "unknown_str"`}, // unknown attr
		}},
		Test{dir, []Prop{
			{"dirstring1", "str2", ""},
			{"dirstring2", "", ""},
			{"dirint1", 234, ""},
			{"dirint2", -234, ""},
			{"dirint3", 0, ""},
			{"dirbool1", true, ""},
			{"dirbool2", false, ""},
			{"dirdec1", 234.5, ""},
			{"dirdec2", -234.5, ""},
			{"dirdec3", 235.0, ""},
			{"dirdec4", 0.0, ""},
		}},
		Test{file, []Prop{
			{"filestring1", "str3", ""},
			{"filestring2", "", ""},
			{"fileint1", 345, ""},
			{"fileint2", -345, ""},
			{"fileint3", 0, ""},
			{"filebool1", true, ""},
			{"filebool2", false, ""},
			{"filedec1", 345.5, ""},
			{"filedec2", -345.5, ""},
			{"filedec3", 346.0, ""},
			{"filedec4", 0.0, ""},
		}},
		Test{ver, []Prop{
			{"filestring1", "str4", ""},
			{"filestring2", "", ""},
			{"fileint1", 456, ""},
			{"fileint2", -456, ""},
			{"fileint3", 0, ""},
			{"filebool1", true, ""},
			{"filebool2", false, ""},
			{"filedec1", 456.5, ""},
			{"filedec2", -456.5, ""},
			{"filedec3", 457.0, ""},
			{"filedec4", 0.0, ""},
		}},
	}

	for _, test := range tests {
		var entity *registry.Entity
		eField := reflect.ValueOf(test.Entity).Elem().FieldByName("Entity")
		if !eField.IsValid() {
			panic("help me")
		}
		entity = eField.Addr().Interface().(*registry.Entity)
		setter := test.Entity

		for _, prop := range test.Props {
			// Note that for Resources this will set them on the latest Version
			err := setter.Set(prop.Name, prop.Value)
			if err != nil && err.Error() != prop.ErrMsg {
				t.Errorf("Error calling set (%q=%v): %q expected %q", prop.Name,
					prop.Value, err, prop.ErrMsg)
				return // stop fast
			}
			if err == nil && prop.ErrMsg != "" {
				t.Errorf("Setting (%q=%v) was supposed to fail: %s",
					prop.Name, prop.Value, prop.ErrMsg)
				return // stop fast
			}
		}

		entity.Props = map[string]any{} // force delete everything
		entity.Refresh()                // and then re-get props from DB

		for _, prop := range test.Props {
			if prop.ErrMsg != "" {
				continue
			}
			got := setter.Get(prop.Name) // test.Entity.Get(prop.Name)
			if got != prop.Value {
				t.Errorf("%T) %s: got %v(%T), expected %v(%T)\n",
					test.Entity, prop.Name, got, got, prop.Value, prop.Value)
				return // stop fast
			}
		}
	}

	xCheckGet(t, reg, "?inline", `{
  "specversion": "0.5",
  "id": "TestBasicTypes",
  "epoch": 1,
  "self": "http://localhost:8181/",
  "reganyarrayint": [
    null,
    null,
    5
  ],
  "reganyarrayobj": [
    -5,
    null,
    {
      "int1": 55,
      "myobj": {
        "int2": 555
      }
    }
  ],
  "reganyint": 123,
  "reganyobj": {
    "int": 345,
    "nestobj": {
      "int": 123
    },
    "str": "substr"
  },
  "reganystr": "mystr",
  "regarrayarrayint": [
    null,
    [
      null,
      66
    ]
  ],
  "regarrayint": [
    1,
    2,
    3
  ],
  "regbool1": true,
  "regbool2": false,
  "regdec1": 123.5,
  "regdec2": -123.5,
  "regdec3": 124,
  "regdec4": 0,
  "regint1": 123,
  "regint2": -123,
  "regint3": 0,
  "regmapint": {
    "k1": 123,
    "k2": 234
  },
  "regmapstring": {
    "k1": "v1",
    "k2": "v2"
  },
  "regobj": {
    "objbool": true,
    "objint": 345,
    "objobj": {
      "ooint": 999
    },
    "objstr": "in1"
  },
  "regstring1": "str1",
  "regstring2": "",
  "regtime1": "2006-01-02T15:04:05Z",
  "reguint1": 0,
  "reguint2": 333,

  "dirs": {
    "d1": {
      "id": "d1",
      "epoch": 1,
      "self": "http://localhost:8181/dirs/d1",
      "dirbool1": true,
      "dirbool2": false,
      "dirdec1": 234.5,
      "dirdec2": -234.5,
      "dirdec3": 235,
      "dirdec4": 0,
      "dirint1": 234,
      "dirint2": -234,
      "dirint3": 0,
      "dirstring1": "str2",
      "dirstring2": "",

      "files": {
        "f1": {
          "id": "f1",
          "epoch": 1,
          "self": "http://localhost:8181/dirs/d1/files/f1",
          "latestversionid": "v1",
          "latestversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
          "filebool1": true,
          "filebool2": false,
          "filedec1": 456.5,
          "filedec2": -456.5,
          "filedec3": 457,
          "filedec4": 0,
          "fileint1": 456,
          "fileint2": -456,
          "fileint3": 0,
          "filestring1": "str4",
          "filestring2": "",

          "versions": {
            "v1": {
              "id": "v1",
              "epoch": 1,
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
              "latest": true,
              "filebool1": true,
              "filebool2": false,
              "filedec1": 456.5,
              "filedec2": -456.5,
              "filedec3": 457,
              "filedec4": 0,
              "fileint1": 456,
              "fileint2": -456,
              "fileint3": 0,
              "filestring1": "str4",
              "filestring2": ""
            }
          },
          "versionscount": 1,
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions"
        }
      },
      "filescount": 1,
      "filesurl": "http://localhost:8181/dirs/d1/files"
    }
  },
  "dirscount": 1,
  "dirsurl": "http://localhost:8181/dirs"
}
`)
}

func TestWildcardBoolTypes(t *testing.T) {
	reg := NewRegistry("TestWildcardBoolTypes")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttr("*", registry.BOOLEAN)
	reg.Model.Save()

	err := reg.Set("bogus", "foo")
	xCheck(t, err.Error() == `"foo" should be a boolean`,
		fmt.Sprintf("bogus=foo: %s", err))

	err = reg.Set("ext1", true)
	xCheck(t, err == nil, fmt.Sprintf("set ext1: %s", err))
	reg.Refresh()
	val := reg.Get("ext1")
	xCheck(t, val == true, fmt.Sprintf("get ext1: %v", val))

	err = reg.Set("ext1", false)
	xCheck(t, err == nil, fmt.Sprintf("set ext1-2: %s", err))
	reg.Refresh()
	xCheck(t, reg.Get("ext1") == false, fmt.Sprintf("get ext1-2: %v", val))
}

func TestWildcardAnyTypes(t *testing.T) {
	reg := NewRegistry("TestWildcardAnyTypes")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttr("*", registry.ANY)
	reg.Model.Save()

	// Make sure we can set the same attr to two different types
	err := reg.Set("ext1", 5.5)
	xCheck(t, err == nil, fmt.Sprintf("set ext1: %s", err))
	reg.Refresh()
	val := reg.Get("ext1")
	xCheck(t, val == 5.5, fmt.Sprintf("get ext1: %v", val))

	err = reg.Set("ext1", "foo")
	xCheck(t, err == nil, fmt.Sprintf("set ext2: %s", err))
	reg.Refresh()
	val = reg.Get("ext1")
	xCheck(t, val == "foo", fmt.Sprintf("get ext2: %v", val))

	// Make sure we add one of a different type
	err = reg.Set("ext2", true)
	xCheck(t, err == nil, fmt.Sprintf("set ext3 %s", err))
	reg.Refresh()
	val = reg.Get("ext2")
	xCheck(t, val == true, fmt.Sprintf("get ext3: %v", val))
}

func TestWildcard2LayersTypes(t *testing.T) {
	reg := NewRegistry("TestWildcardAnyTypes")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttribute(&registry.Attribute{
		Name: "obj",
		Type: registry.OBJECT,
		Item: &registry.Item{
			Attributes: map[string]*registry.Attribute{
				"map": {
					Name: "map",
					Type: registry.MAP,
					Item: &registry.Item{Type: registry.INTEGER},
				},
				"*": {
					Name: "*",
					Type: registry.ANY,
				},
			},
		},
	})
	reg.Model.Save()

	err := reg.Set("obj.map.k1", 5)
	xCheck(t, err == nil, fmt.Sprintf("set foo.k1: %s", err))
	reg.Refresh()
	val := reg.Get("obj.map.k1")
	xCheck(t, val == 5, fmt.Sprintf("get foo.k1: %v", val))

	err = reg.Set("obj.map.foo.k1.k2", 5)
	xCheck(t, err.Error() == `Traversing into scalar "foo": obj.map.foo.k1.k2`,
		fmt.Sprintf("set obj.map.foo.k1.k2: %s", err))

	err = reg.Set("obj.myany.foo.k1.k2", 5)
	reg.Refresh()
	val = reg.Get("obj.myany.foo.k1.k2")
	xCheck(t, val == 5, fmt.Sprintf("set obj.myany.foo.k1.k2: %v", val))
	val = reg.Get("obj.myany.bogus.k1.k2")
	xCheck(t, val == nil, fmt.Sprintf("set obj.myany.bogus.k1.k2: %v", val))

}
