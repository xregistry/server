package tests

import (
	"fmt"
	"reflect"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestBasicTypes(t *testing.T) {
	reg := NewRegistry("TestBasicTypes")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttr("regbool1", BOOLEAN)
	reg.Model.AddAttr("regbool2", BOOLEAN)
	reg.Model.AddAttr("regdec1", DECIMAL)
	reg.Model.AddAttr("regdec2", DECIMAL)
	reg.Model.AddAttr("regdec3", DECIMAL)
	reg.Model.AddAttr("regdec4", DECIMAL)
	reg.Model.AddAttr("regint1", INTEGER)
	reg.Model.AddAttr("regint2", INTEGER)
	reg.Model.AddAttr("regint3", INTEGER)
	reg.Model.AddAttr("regstring1", STRING)
	reg.Model.AddAttr("regstring2", STRING)
	reg.Model.AddAttr("reguint1", UINTEGER)
	reg.Model.AddAttr("reguint2", UINTEGER)
	reg.Model.AddAttr("regtime1", TIMESTAMP)

	reg.Model.AddAttr("reganyarrayint", ANY)
	reg.Model.AddAttr("reganyarrayobj", ANY)
	reg.Model.AddAttr("reganyint", ANY)
	reg.Model.AddAttr("reganystr", ANY)
	reg.Model.AddAttr("reganyobj", ANY)

	reg.Model.AddAttrArray("regarrayarrayint",
		registry.NewItemArray(registry.NewItemType(INTEGER)))

	reg.Model.AddAttrArray("regarrayint", registry.NewItemType(INTEGER))
	reg.Model.AddAttrMap("regmapint", registry.NewItemType(INTEGER))
	reg.Model.AddAttrMap("regmapstring", registry.NewItemType(STRING))

	attr, err := reg.Model.AddAttrObj("regobj")
	xNoErr(t, err)
	attr.AddAttr("objbool", BOOLEAN)
	attr.AddAttr("objint", INTEGER)
	attr2, _ := attr.AddAttrObj("objobj")
	attr2.AddAttr("ooint", INTEGER)
	attr.AddAttr("objstr", STRING)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddAttr("dirbool1", BOOLEAN)
	gm.AddAttr("dirbool2", BOOLEAN)
	gm.AddAttr("dirdec1", DECIMAL)
	gm.AddAttr("dirdec2", DECIMAL)
	gm.AddAttr("dirdec3", DECIMAL)
	gm.AddAttr("dirdec4", DECIMAL)
	gm.AddAttr("dirint1", INTEGER)
	gm.AddAttr("dirint2", INTEGER)
	gm.AddAttr("dirint3", INTEGER)
	gm.AddAttr("dirstring1", STRING)
	gm.AddAttr("dirstring2", STRING)

	gm.AddAttr("diranyarray", ANY)
	gm.AddAttr("diranymap", ANY)
	gm.AddAttr("diranyobj", ANY)
	gm.AddAttrArray("dirarrayint", registry.NewItemType(INTEGER))
	gm.AddAttrMap("dirmapint", registry.NewItemType(INTEGER))
	attr, _ = gm.AddAttrObj("dirobj")
	attr.AddAttr("*", ANY)

	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	rm.AddAttr("filebool1", BOOLEAN)
	rm.AddAttr("filebool2", BOOLEAN)
	rm.AddAttr("filedec1", DECIMAL)
	rm.AddAttr("filedec2", DECIMAL)
	rm.AddAttr("filedec3", DECIMAL)
	rm.AddAttr("filedec4", DECIMAL)
	rm.AddAttr("fileint1", INTEGER)
	rm.AddAttr("fileint2", INTEGER)
	rm.AddAttr("fileint3", INTEGER)
	rm.AddAttr("filestring1", STRING)
	rm.AddAttr("filestring2", STRING)

	rm.AddAttr("xid1", XID)
	rm.AddAttr("xid2", XID)
	rm.AddAttr("xid3", XID)
	rm.AddAttr("xidtype1", XIDTYPE)
	rm.AddAttr("xidtype2", XIDTYPE)

	xNoErr(t, reg.SaveModel())

	/* no longer required
	_, err = reg.Model.AddAttrXID("regptr_group", "")
	xCheckErr(t, err, `"model.regptr_group" must have a "target" value since "type" is "xid"`)
	*/

	_, err = reg.Model.AddAttrXID("regptr_group", "qwe")
	xCheckErr(t, err, `"model.regptr_group" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`)

	_, err = reg.Model.AddAttrXID("regptr_group", "qwe/")
	xCheckErr(t, err, `"model.regptr_group" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`)

	_, err = reg.Model.AddAttrXID("regptr_group", " /")
	xCheckErr(t, err, `"model.regptr_group" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`)

	_, err = reg.Model.AddAttrXID("regptr_reg", "/")
	xCheckErr(t, err, `"model.regptr_reg" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`)

	_, err = reg.Model.AddAttrXID("regptr_group", "/xxxs")
	xCheckErr(t, err, `"model.regptr_group" has an unknown Group type: "xxxs"`)

	_, err = reg.Model.AddAttrXID("regptr_group", "/xxxs/")
	xCheckErr(t, err, `"model.regptr_group" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`)

	_, err = reg.Model.AddAttrXID("regptr_group", "/dirs")
	xCheckErr(t, err, ``)

	_, err = reg.Model.AddAttrXID("regptr_res", "/dirs/?")
	xCheckErr(t, err, `"model.regptr_res" has an unknown Resource type: "?"`)

	_, err = reg.Model.AddAttrXID("regptr_res", "/dirs/file")
	xCheckErr(t, err, `"model.regptr_res" has an unknown Resource type: "file"`)

	_, err = reg.Model.AddAttrXID("regptr_res", "/dirs/files")
	xNoErr(t, err)
	xNoErr(t, reg.SaveModel())

	_, err = reg.Model.AddAttrXID("regptr_ver", "/dirs/files/")
	xCheckErr(t, err, `"model.regptr_ver" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`)

	_, err = reg.Model.AddAttrXID("regptr_ver", "/dirs/files/asd")
	xCheckErr(t, err, `"model.regptr_ver" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`)

	_, err = reg.Model.AddAttrXID("regptr_ver", "/dirs/files/asd?")
	xCheckErr(t, err, `"model.regptr_ver" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`)

	_, err = reg.Model.AddAttrXID("regptr_ver", "/dirs/files/versions")
	xNoErr(t, err)
	err = reg.SaveModel()
	xCheckErr(t, err, ``)

	_, err = reg.Model.AddAttrXID("regptr_res_ver", "/dirs/files/versions?asd")
	xCheckErr(t, err, `"model.regptr_res_ver" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`)

	_, err = reg.Model.AddAttrXID("regptr_res_ver", "/dirs/files/versions?/")
	xCheckErr(t, err, `"model.regptr_res_ver" "target" must be of the form: /GROUPS[/RESOURCES[/versions | \[/versions\] ]]`)

	_, err = reg.Model.AddAttrXID("regptr_res_ver", "/dirs/files[/versions]")
	xNoErr(t, err)

	_, err = reg.Model.AddAttrXID("regptr_res_ver2", "/dirs/files[/versions]")
	xNoErr(t, err)

	// Model is fully defined, so save it
	xNoErr(t, reg.SaveModel())

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1", false, registry.FOR_WRITE)

	dir2, _ := reg.AddGroup("dirs", "dir2")

	reg.SaveAllAndCommit()

	// /dirs/d1/f1/v1

	type Prop struct {
		Name     string
		Value    any
		ExpValue any
		ErrMsg   string
	}

	type Test struct {
		Entity registry.EntitySetter // any //  *registry.Entity
		Props  []Prop
	}

	tests := []Test{
		Test{reg, []Prop{
			{"registryid", 66, nil, `Attribute "registryid" must be a string`},
			{"registryid", "*", nil, `Invalid ID "*", must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$`},

			{"regarrayarrayint[1][1]", 66, nil, "Attribute \"regarrayarrayint[1][0]\" must be an integer"},
			{"regarrayint[0]", 1, nil, ""},
			{"regarrayint[2]", 3, nil, "Attribute \"regarrayint[1]\" must be an integer"},
			{"regarrayint[1]", 2, nil, ""},
			{"regarrayint[2]", 3, nil, ""},

			{"regbool1", true, nil, ""},
			{"regbool2", false, nil, ""},
			{"regdec1", 123.5, nil, ""},
			{"regdec2", -123.5, nil, ""},
			{"regdec3", 124.0, nil, ""},
			{"regdec4", 0.0, nil, ""},
			{"regint1", 123, nil, ""},
			{"regint2", -123, nil, ""},
			{"regint3", 0, nil, ""},
			{"regmapint.k1", 123, nil, ""},
			{"regmapint.k2", 234, nil, ""},
			{"regmapstring.k1", "v1", nil, ""},
			{"regmapstring.k2", "v2", nil, ""},
			{"regstring1", "str1", nil, ""},
			{"regstring2", "", nil, ""},
			{"regtime1", "2006-01-02T15:04:05Z", nil, ""},
			{"reguint1", 0, nil, ""},
			{"reguint2", 333, nil, ""},

			{"reganyarrayint[2]", 5, nil, ""},
			{"reganyarrayobj[2].int1", 55, nil, ""},
			{"reganyarrayobj[2].myobj.int2", 555, nil, ""},
			{"reganyarrayobj[0]", -5, nil, ""},
			{"reganyint", 123, nil, ""},
			{"reganyobj.int", 345, nil, ""},
			{"reganyobj.str", "substr", nil, ""},
			{"reganystr", "mystr", nil, ""},
			{"regobj.objbool", true, nil, ""},
			{"regobj.objint", 345, nil, ""},
			{"regobj.objobj.ooint", 999, nil, ""},
			{"regobj.objstr", "in1", nil, ""},

			{"reganyobj.nestobj.int", 123, nil, ""},

			// Syntax checking
			// {"MiXeD", 123,nil, ""},
			{"regarrayint[~abc]", 123, nil,
				`Unexpected ~ in "regarrayint[~abc]" at pos 13`},
			{"regarrayint['~abc']", 123, nil,
				`Unexpected ~ in "regarrayint['~abc']" at pos 14`},
			{"regmapstring.~abc", 123, nil,
				`Unexpected ~ in "regmapstring.~abc" at pos 14`},
			{"regmapstring[~abc]", 123, nil,
				`Unexpected ~ in "regmapstring[~abc]" at pos 14`},
			{"regmapstring['~abc']", 123, nil,
				`Unexpected ~ in "regmapstring['~abc']" at pos 15`},

			// Type checking
			{"epoch", -123, nil,
				`Attribute "epoch" must be a uinteger`},
			{"regobj[1]", "", nil,
				`Attribute "regobj[1]" isn't an array`}, // Not an array
			{"regobj", []any{}, nil,
				`Attribute "regobj" must be a map[string] or object`}, // Not an array
			{"reganyobj2.str", "substr", nil,
				`Invalid extension(s): reganyobj2`}, // unknown attr
			{"regarrayarrayint[0][0]", "abc", nil,
				`Attribute "regarrayarrayint[0][0]" must be an integer`}, // bad type
			{"regarrayint[2]", "abc", nil,
				`Attribute "regarrayint[2]" must be an integer`}, // bad type
			{"regbool1", "123", nil,
				`Attribute "regbool1" must be a boolean`}, // bad type
			{"regdec1", "123", nil,
				`Attribute "regdec1" must be a decimal`}, // bad type
			{"regint1", "123", nil,
				`Attribute "regint1" must be an integer`}, // bad type
			{"regmapint", "123", nil,
				`Attribute "regmapint" must be a map`}, // must be empty
			{"regmapint.k1", "123", nil,
				`Attribute "regmapint.k1" must be an integer`}, // bad type
			{"regmapstring.k1", 123, nil,
				`Attribute "regmapstring.k1" must be a string`}, // bad type
			{"regstring1", 123, nil,
				`Attribute "regstring1" must be a string`}, // bad type
			{"regtime1", "not a time", nil,
				`Attribute "regtime1" is a malformed timestamp`}, // bad format
			{"reguint1", -1, nil,
				`Attribute "reguint1" must be a uinteger`}, // bad uint
			{"unknown_int", 123, nil,
				`Invalid extension(s): unknown_int`}, // unknown attr
			{"unknown_str", "error", nil,
				`Invalid extension(s): unknown_str`}, // unknown attr

			{"regptr_group", "", nil, `Attribute "regptr_group" must be an xid, not empty`},
			{"regptr_group", "/", nil, `Attribute "regptr_group" must match "/dirs" target`},
			{"regptr_group", "/xxx", nil, `Attribute "regptr_group" must match "/dirs" target`},
			{"regptr_group", "/dirs", nil, `Attribute "regptr_group" must match "/dirs" target, missing "dirid"`},
			{"regptr_group", "/dirs2", nil, `Attribute "regptr_group" must match "/dirs" target`},
			{"regptr_group", "/dirs", nil, `Attribute "regptr_group" must match "/dirs" target, missing "dirid"`},
			{"regptr_group", "/dirs/*", nil, `Attribute "regptr_group" must match "/dirs" target: Invalid ID "*", must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$`},
			{"regptr_group", "/dirs/id/", nil, `Attribute "regptr_group" must match "/dirs" target, extra stuff after "id"`},
			{"regptr_group", "/dirs/id/extra", nil, `Attribute "regptr_group" must match "/dirs" target, extra stuff after "id"`},
			{"regptr_group", "/dirs/id/extra/", nil, `Attribute "regptr_group" must match "/dirs" target, extra stuff after "id"`},
			{"regptr_group", "/dirs/d1", nil, ``},

			{"regptr_res", "/dirs/d1", nil, `Attribute "regptr_res" must match "/dirs/files" target, missing "files"`},
			{"regptr_res", "/dirs/d1/", nil, `Attribute "regptr_res" must match "/dirs/files" target, missing "files"`},
			{"regptr_res", "/dirs/d1/fff", nil, `Attribute "regptr_res" must match "/dirs/files" target, missing "files"`},
			{"regptr_res", "/dirs/d1/fff/", nil, `Attribute "regptr_res" must match "/dirs/files" target, missing "files"`},
			{"regptr_res", "/dirs/d1/fff/f2", nil, `Attribute "regptr_res" must match "/dirs/files" target, missing "files"`},
			{"regptr_res", "/dirs/*/files/f2", nil, `Attribute "regptr_res" must match "/dirs/files" target: Invalid ID "*", must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$`},
			{"regptr_res", "/dirs/d1/files", nil, `Attribute "regptr_res" must match "/dirs/files" target, missing "fileid"`},
			{"regptr_res", "/dirs/d1/files/", nil, `Attribute "regptr_res" must match "/dirs/files" target, missing "fileid"`},
			{"regptr_res", "/dirs/d1/files/*", nil, `Attribute "regptr_res" must match "/dirs/files" target: Invalid ID "*", must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$`},
			{"regptr_res", "/dirs/d1/files/f2/versions", nil, `Attribute "regptr_res" must match "/dirs/files" target, extra stuff after "f2"`},
			{"regptr_res", "/dirs/d1/files/f2/versions/v1", nil, `Attribute "regptr_res" must match "/dirs/files" target, extra stuff after "f2"`},
			{"regptr_res", "/dirs/d1/files/f2", nil, ``},

			{"regptr_ver", "/", nil, `Attribute "regptr_ver" must match "/dirs/files/versions" target`},
			{"regptr_ver", "/dirs/d1/files/f2", nil, `Attribute "regptr_ver" must match "/dirs/files/versions" target, missing "versions"`},
			{"regptr_ver", "/dirs/d1/files/f2/vvv", nil, `Attribute "regptr_ver" must match "/dirs/files/versions" target, missing "versions"`},
			{"regptr_ver", "/dirs/d1/files/f2/versions", nil, `Attribute "regptr_ver" must match "/dirs/files/versions" target, missing a "versionid"`},
			{"regptr_ver", "/dirs/d1/files/f2/versions/", nil, `Attribute "regptr_ver" must match "/dirs/files/versions" target, missing a "versionid"`},
			{"regptr_ver", "/dirs/d1/files/f2/versions/v2/", nil, `Attribute "regptr_ver" must match "/dirs/files/versions" target, too long`},
			{"regptr_ver", "/dirs/d1/files/f2/versions/v2/xx", nil, `Attribute "regptr_ver" must match "/dirs/files/versions" target, too long`},
			{"regptr_ver", "/dirs/d1/files/f2/versions/v2?", nil, `Attribute "regptr_ver" must match "/dirs/files/versions" target: Invalid ID "v2?", must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$`},
			{"regptr_ver", "/dirs/d1/files/f2/versions/v2", nil, ``},

			{"regptr_res_ver", "/dirs/d1/files/", nil, `Attribute "regptr_res_ver" must match "/dirs/files[/versions]" target, missing "fileid"`},
			{"regptr_res_ver", "/dirs/d1/files//", nil, `Attribute "regptr_res_ver" must match "/dirs/files[/versions]" target, missing "fileid"`},
			{"regptr_res_ver", "/dirs/d1/files/f2/", nil, `Attribute "regptr_res_ver" must match "/dirs/files[/versions]" target, missing "versions"`},
			{"regptr_res_ver", "/dirs/d1/files/f2/vers", nil, `Attribute "regptr_res_ver" must match "/dirs/files[/versions]" target, missing "versions"`},
			{"regptr_res_ver", "/dirs/d1/files/f2/vers/v1", nil, `Attribute "regptr_res_ver" must match "/dirs/files[/versions]" target, missing "versions"`},
			{"regptr_res_ver", "/dirs/d1/files/f*/vers/v1", nil, `Attribute "regptr_res_ver" must match "/dirs/files[/versions]" target: Invalid ID "f*", must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$`},
			{"regptr_res_ver", "/dirs/d1/files/f2", nil, ``},

			{"regptr_res_ver2", "/dirs/d1/files/f2/versions", nil, `Attribute "regptr_res_ver2" must match "/dirs/files[/versions]" target, missing a "versionid"`},
			{"regptr_res_ver2", "/dirs/d1/files/f2/versions/", nil, `Attribute "regptr_res_ver2" must match "/dirs/files[/versions]" target, missing a "versionid"`},
			{"regptr_res_ver2", "/dirs/d1/files/f2/versions//v2", nil, `Attribute "regptr_res_ver2" must match "/dirs/files[/versions]" target, missing a "versionid"`},
			{"regptr_res_ver2", "/dirs/d1/files/f2/versions/v2/", nil, `Attribute "regptr_res_ver2" must match "/dirs/files[/versions]" target, too long`},
			{"regptr_res_ver2", "/dirs/d1/files/f2/versions/v*", nil, `Attribute "regptr_res_ver2" must match "/dirs/files[/versions]" target: Invalid ID "v*", must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$`},
			{"regptr_res_ver2", "/dirs/d1/files/f2/versions/v2", nil, ``},
		}},
		Test{dir, []Prop{
			{"dirid", 66, nil, `Attribute "dirid" must be a string`},
			{"dirid", "*", nil, `Invalid ID "*", must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$`},

			{"dirstring1", "str2", nil, ""},
			{"dirstring2", "", nil, ""},
			{"dirint1", 234, nil, ""},
			{"dirint2", -234, nil, ""},
			{"dirint3", 0, nil, ""},
			{"dirbool1", true, nil, ""},
			{"dirbool2", false, nil, ""},
			{"dirdec1", 234.5, nil, ""},
			{"dirdec2", -234.5, nil, ""},
			{"dirdec3", 235.0, nil, ""},
			{"dirdec4", 0.0, nil, ""},
		}},
		Test{dir2, []Prop{
			{"diranyarray", []any{}, nil, ""},
			{"diranymap", map[string]any{}, nil, ""},
			{"diranyobj", struct{}{}, map[string]any{}, ""},
			{"dirarrayint", []int{}, []any{}, ""},
			{"dirmapint", map[string]any{}, nil, ""},
			{"dirobj", struct{}{}, map[string]any{}, ""},
		}},
		Test{file, []Prop{
			{"fileid", 66, nil, `Attribute "fileid" must be a string`},
			{"fileid", "*", nil, `Invalid ID "*", must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$`},
			{"versionid", 66, nil, `Attribute "versionid" must be a string`},
			{"versionid", "*", nil, `Invalid ID "*", must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$`},

			{"filestring1", "str3", nil, ""},
			{"filestring2", "", nil, ""},
			{"fileint1", 345, nil, ""},
			{"fileint2", -345, nil, ""},
			{"fileint3", 0, nil, ""},
			{"filebool1", true, nil, ""},
			{"filebool2", false, nil, ""},
			{"filedec1", 345.5, nil, ""},
			{"filedec2", -345.5, nil, ""},
			{"filedec3", 346.0, nil, ""},
			{"filedec4", 0.0, nil, ""},

			{"xid1", "", nil, `Attribute "xid1" () isn't a valid xid, can't be an empty string`},
			{"xid1", "//", nil, `Attribute "xid1" (//) isn't a valid xid, "//" has an empty part at position 1`},
			{"xid1", "/dirs/d1/files/f1/versions/v1/xxx", nil, `Attribute "xid1" (/dirs/d1/files/f1/versions/v1/xxx) isn't a valid xid, XID is too long`},
			{"xid1", "/DIRS", nil, `Attribute "xid1" (/DIRS) references an unknown GroupModel "DIRS"`},
			{"xid1", "/DIRS/d1", nil, `Attribute "xid1" (/DIRS/d1) references an unknown GroupModel "DIRS"`},
			{"xid1", "/dirs/d1/FILES", nil, `Attribute "xid1" (/dirs/d1/FILES) references an unknown ResourceModel "FILES"`},
			{"xid1", "/dirs/d1/FILES/f1", nil, `Attribute "xid1" (/dirs/d1/FILES/f1) references an unknown ResourceModel "FILES"`},
			{"xid1", "/dirs/d1/files/f1/VERSIONS", nil, `Attribute "xid1" (/dirs/d1/files/f1/VERSIONS) isn't a valid xid, references an unknown entity "VERSIONS"`},
			{"xid1", "/dirs/d1/files/f1/VERSIONS/v1", nil, `Attribute "xid1" (/dirs/d1/files/f1/VERSIONS/v1) isn't a valid xid, references an unknown entity "VERSIONS"`},
			{"xid1", "/dirs/d1/files/f1/VERSIONS/v1/xxx", nil, `Attribute "xid1" (/dirs/d1/files/f1/VERSIONS/v1/xxx) isn't a valid xid, references an unknown entity "VERSIONS"`},
			{"xid1", "/dirs/d1/files/f1/META", nil, `Attribute "xid1" (/dirs/d1/files/f1/META) isn't a valid xid, references an unknown entity "META"`},
			{"xid1", "/dirs/d1/files/f1/META/xxx", nil, `Attribute "xid1" (/dirs/d1/files/f1/META/xxx) isn't a valid xid, references an unknown entity "META"`},
			{"xid1", "/dirs/d1/files/f1/meta/xxx", nil, `Attribute "xid1" (/dirs/d1/files/f1/meta/xxx) isn't a valid xid, XID is too long`},
			{"xid1", "/dirs/d1/files/f1/meta", nil, ""},
			{"xid2", "/dirs/d1/files/f1/versions", nil, ""},
			{"xid3", "/dirs/d1/files/f1/versions/v1", nil, ""},

			{"xidtype1", "", nil, `Attribute "xidtype1" () isn't a valid xidtype, can't be an empty string`},
			{"xidtype1", "//", nil, `Attribute "xidtype1" (//) isn't a valid xidtype, "//" has an empty part at position 1`},
			{"xidtype1", "/dirs/files/versions/xxx", nil, `Attribute "xidtype1" (/dirs/files/versions/xxx) isn't a valid xidtype, XIDType is too long`},
			{"xidtype1", "/DIRS", nil, `Attribute "xidtype1" (/DIRS) references an unknown GroupModel "DIRS"`},
			{"xidtype1", "/DIRS/FILES", nil, `Attribute "xidtype1" (/DIRS/FILES) references an unknown GroupModel "DIRS"`},
			{"xidtype1", "/dirs/FILES", nil, `Attribute "xidtype1" (/dirs/FILES) references an unknown ResourceModel "FILES"`},
			{"xidtype1", "/dirs/files/xxx", nil, `Attribute "xidtype1" (/dirs/files/xxx) isn't a valid xidtype, references an unknown entity "xxx"`},
			{"xidtype1", "/dirs/files/meta/xxx", nil, `Attribute "xidtype1" (/dirs/files/meta/xxx) isn't a valid xidtype, XIDType is too long`},
			{"xidtype1", "/dirs/files/META", nil, `Attribute "xidtype1" (/dirs/files/META) isn't a valid xidtype, references an unknown entity "META"`},
			{"xidtype1", "/dirs/files/VERSIONS", nil, `Attribute "xidtype1" (/dirs/files/VERSIONS) isn't a valid xidtype, references an unknown entity "VERSIONS"`},
			{"xidtype1", "/dirs/files/meta", nil, ``},
			{"xidtype2", "/dirs/files/versions", nil, ``},
		}},
		Test{ver, []Prop{
			{"versionid", 66, nil, `Attribute "versionid" must be a string`},
			{"versionid", "*", nil, `Invalid ID "*", must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$`},

			{"filestring1", "str4", nil, ""},
			{"filestring2", "", nil, ""},
			{"fileint1", 456, nil, ""},
			{"fileint2", -456, nil, ""},
			{"fileint3", 0, nil, ""},
			{"filebool1", true, nil, ""},
			{"filebool2", false, nil, ""},
			{"filedec1", 456.5, nil, ""},
			{"filedec2", -456.5, nil, ""},
			{"filedec3", 457.0, nil, ""},
			{"filedec4", 0.0, nil, ""},
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
			// Note that for Resources this will set them on the default Version
			err := setter.SetSave(prop.Name, prop.Value)
			if err != nil && err.Error() != prop.ErrMsg {
				t.Logf("Test: %s  val:%v", prop.Name, prop.Value)
				t.Errorf("Error calling set (%q=%v):\nExp: %s\nGot: %s",
					prop.Name, prop.Value, prop.ErrMsg, err)
				return // stop fast
			}
			if err == nil && prop.ErrMsg != "" {
				t.Logf("Test: %s  val:%v", prop.Name, prop.Value)
				t.Errorf("Setting (%q=%v) was supposed to fail:\nExp: %s",
					prop.Name, prop.Value, prop.ErrMsg)
				return // stop fast
			}
			if err != nil {
				entity.Refresh(registry.FOR_WRITE)
			}
		}

		entity.Refresh(registry.FOR_WRITE) // and then re-get props from DB

		for _, prop := range test.Props {
			if prop.ErrMsg != "" {
				continue
			}
			got := setter.Get(prop.Name) // test.Entity.Get(prop.Name)
			expected := prop.ExpValue
			if expected == nil {
				expected = prop.Value
			}
			if !reflect.DeepEqual(got, expected) {
				// if got != expected {
				t.Logf("%s  val: %v", prop.Name, prop.Value)
				t.Errorf("%T) %s: got %v(%T), expected %v(%T)\n",
					test.Entity, prop.Name, got, got, prop.Value, prop.Value)
				return // stop fast
			}
		}
	}

	xCheckGet(t, reg, "?inline", `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestBasicTypes",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 8,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
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
  "regptr_group": "/dirs/d1",
  "regptr_res": "/dirs/d1/files/f2",
  "regptr_res_ver": "/dirs/d1/files/f2",
  "regptr_res_ver2": "/dirs/d1/files/f2/versions/v2",
  "regptr_ver": "/dirs/d1/files/f2/versions/v2",
  "regstring1": "str1",
  "regstring2": "",
  "regtime1": "2006-01-02T15:04:05Z",
  "reguint1": 0,
  "reguint2": 333,

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2024-01-01T12:00:04Z",
      "modifiedat": "2024-01-01T12:00:02Z",
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

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 5,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:04Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "v1",
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
          "xid1": "/dirs/d1/files/f1/meta",
          "xid2": "/dirs/d1/files/f1/versions",
          "xid3": "/dirs/d1/files/f1/versions/v1",
          "xidtype1": "/dirs/files/meta",
          "xidtype2": "/dirs/files/versions",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:04Z",
            "modifiedat": "2024-01-01T12:00:04Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 5,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:04Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "v1",
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
              "xid1": "/dirs/d1/files/f1/meta",
              "xid2": "/dirs/d1/files/f1/versions",
              "xid3": "/dirs/d1/files/f1/versions/v1",
              "xidtype1": "/dirs/files/meta",
              "xidtype2": "/dirs/files/versions"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1
    },
    "dir2": {
      "dirid": "dir2",
      "self": "http://localhost:8181/dirs/dir2",
      "xid": "/dirs/dir2",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:04Z",
      "modifiedat": "2024-01-01T12:00:04Z",
      "diranyarray": [],
      "diranymap": {},
      "diranyobj": {},
      "dirarrayint": [],
      "dirmapint": {},
      "dirobj": {},

      "filesurl": "http://localhost:8181/dirs/dir2/files",
      "files": {},
      "filescount": 0
    }
  },
  "dirscount": 2
}
`)
}

func TestWildcardBoolTypes(t *testing.T) {
	reg := NewRegistry("TestWildcardBoolTypes")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttr("*", BOOLEAN)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true, true)
	xNoErr(t, err)
	xNoErr(t, reg.Model.Save())

	dir, err := reg.AddGroup("dirs", "d1")
	xNoErr(t, err)
	_, err = dir.AddResource("files", "f1", "v1")
	xNoErr(t, err)

	err = reg.SetSave("bogus", "foo")
	xCheck(t, err.Error() == `Attribute "bogus" must be a boolean`,
		fmt.Sprintf("bogus=foo: %s", err))

	err = reg.SetSave("ext1", true)
	xCheck(t, err == nil, fmt.Sprintf("set ext1: %s", err))
	reg.Refresh(registry.FOR_WRITE)
	val := reg.Get("ext1")
	xCheck(t, val == true, fmt.Sprintf("get ext1: %v", val))

	err = reg.SetSave("ext1", false)
	xCheck(t, err == nil, fmt.Sprintf("set ext1-2: %s", err))
	reg.Refresh(registry.FOR_WRITE)
	xCheck(t, reg.Get("ext1") == false, fmt.Sprintf("get ext1-2: %v", val))
}

func TestWildcardAnyTypes(t *testing.T) {
	reg := NewRegistry("TestWildcardAnyTypes")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttr("*", ANY)
	xNoErr(t, reg.Model.Save())

	// Make sure we can set the same attr to two different types
	err := reg.SetSave("ext1", 5.5)
	xCheck(t, err == nil, fmt.Sprintf("set ext1: %s", err))
	reg.Refresh(registry.FOR_WRITE)
	val := reg.Get("ext1")
	xCheck(t, val == 5.5, fmt.Sprintf("get ext1: %v", val))

	err = reg.SetSave("ext1", "foo")
	xCheck(t, err == nil, fmt.Sprintf("set ext2: %s", err))
	reg.Refresh(registry.FOR_WRITE)
	val = reg.Get("ext1")
	xCheck(t, val == "foo", fmt.Sprintf("get ext2: %v", val))

	// Make sure we add one of a different type
	err = reg.SetSave("ext2", true)
	xCheck(t, err == nil, fmt.Sprintf("set ext3 %s", err))
	reg.Refresh(registry.FOR_WRITE)
	val = reg.Get("ext2")
	xCheck(t, val == true, fmt.Sprintf("get ext3: %v", val))
}

func TestWildcard2LayersTypes(t *testing.T) {
	reg := NewRegistry("TestWildcardAnyTypes")
	defer PassDeleteReg(t, reg)

	_, err := reg.Model.AddAttribute(&registry.Attribute{
		Name: "obj",
		Type: OBJECT,
		Attributes: map[string]*registry.Attribute{
			"map": {
				Name: "map",
				Type: MAP,
				Item: &registry.Item{Type: INTEGER},
			},
			"*": {
				Name: "*",
				Type: ANY,
			},
		},
	})
	xCheck(t, err == nil, "")
	xNoErr(t, reg.Model.Save())

	err = reg.SetSave("obj.map.k1", 5)
	xCheck(t, err == nil, fmt.Sprintf("set foo.k1: %s", err))
	reg.Refresh(registry.FOR_WRITE)
	val := reg.Get("obj.map.k1")
	xCheck(t, val == 5, fmt.Sprintf("get foo.k1: %v", val))

	err = reg.SetSave("obj.map.foo.k1.k2", 5)
	xCheck(t, err.Error() == `Attribute "obj.map.foo" must be an integer`,
		fmt.Sprintf("set obj.map.foo.k1.k2: %s", err))
	// reg.Refresh(registry.FOR_WRITE) // clear bad data

	err = reg.SetSave("obj.myany.foo.k1.k2", 5)
	xCheck(t, err == nil, fmt.Sprintf("set obj.myany.foo.k1.k2: %s", err))
	reg.Refresh(registry.FOR_WRITE)
	val = reg.Get("obj.myany.foo.k1.k2")
	xCheck(t, val == 5, fmt.Sprintf("set obj.myany.foo.k1.k2: %v", val))
	val = reg.Get("obj.myany.bogus.k1.k2")
	xCheck(t, val == nil, fmt.Sprintf("set obj.myany.bogus.k1.k2: %v", val))

}

func TestNameCharSet(t *testing.T) {
	reg := NewRegistry("TestNameCharSet")
	defer PassDeleteReg(t, reg)

	_, err := reg.Model.AddAttribute(&registry.Attribute{
		Name: "obj1",
		Type: OBJECT,
		Attributes: map[string]*registry.Attribute{
			"attr1-": {
				Name: "attr1-",
				Type: STRING,
			},
		},
	})
	xCheckErr(t, err, `Error processing "model.obj1": Invalid attribute name "attr1-", must match: ^[a-z_][a-z_0-9]{0,62}$`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:        "obj1",
		Type:        OBJECT,
		NameCharSet: "strict",
		Attributes: map[string]*registry.Attribute{
			"attr1-": {
				Name: "attr1-",
				Type: STRING,
			},
		},
	})
	xCheckErr(t, err, `Error processing "model.obj1": Invalid attribute name "attr1-", must match: ^[a-z_][a-z_0-9]{0,62}$`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "obj1",
		Type: OBJECT,
		Attributes: map[string]*registry.Attribute{
			"attr1": {
				Name: "attr1",
				Type: STRING,
				IfValues: registry.IfValues{
					"a1": &registry.IfValue{
						SiblingAttributes: registry.Attributes{
							"another-": &registry.Attribute{
								Name: "another-",
								Type: STRING,
							},
						},
					},
				},
			},
		},
	})
	xCheckErr(t, err, `Error processing "model.obj1.attr1.ifvalues.a1": Invalid attribute name "another-", must match: ^[a-z_][a-z_0-9]{0,62}$`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:        "obj1",
		Type:        OBJECT,
		NameCharSet: "extended",
		Attributes: map[string]*registry.Attribute{
			"attr space": {
				Name: "attr space",
				Type: STRING,
			},
		},
	})
	xCheckErr(t, err, `Error processing "model.obj1": Invalid map key name "attr space", must match: ^[a-z0-9][a-z0-9_.:\-]{0,62}$`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "astring",
		Type: STRING,
		IfValues: registry.IfValues{
			"a1": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"bad-": &registry.Attribute{
						Type: STRING,
					},
				},
			},
		},
	})
	xCheckErr(t, err, `Error processing "model.astring.ifvalues.a1": Invalid attribute name "bad-", must match: ^[a-z_][a-z_0-9]{0,62}$`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "astring",
		Type: OBJECT,
		Attributes: map[string]*registry.Attribute{
			"attr1": {
				Type: STRING,
				IfValues: registry.IfValues{
					"a1": &registry.IfValue{
						SiblingAttributes: registry.Attributes{
							"bad-": &registry.Attribute{
								Type: STRING,
							},
						},
					},
				},
			},
		},
	})
	xCheckErr(t, err, `Error processing "model.astring.attr1.ifvalues.a1": Invalid attribute name "bad-", must match: ^[a-z_][a-z_0-9]{0,62}$`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:        "astring",
		Type:        OBJECT,
		NameCharSet: "extended",
		Attributes: map[string]*registry.Attribute{
			"attr1": {
				Type: STRING,
				IfValues: registry.IfValues{
					"a1-": &registry.IfValue{
						SiblingAttributes: registry.Attributes{
							"good-": &registry.Attribute{
								Type: STRING,
							},
						},
					},
				},
			},
		},
	})
	xNoErr(t, err)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:        "obj1",
		Type:        OBJECT,
		NameCharSet: "extended",
		Attributes: map[string]*registry.Attribute{
			"attr1-": {
				Name: "attr1-",
				Type: STRING,
				IfValues: registry.IfValues{
					"a1": &registry.IfValue{
						SiblingAttributes: registry.Attributes{
							"another-": &registry.Attribute{
								Name: "another-",
								Type: STRING,
							},
						},
					},
				},
			},
			"attr1-id": {
				Name: "attr1-id",
				Type: STRING,
			},
			"*": {
				Name: "*",
				Type: INTEGER,
			},
		},
	})
	xNoErr(t, err)
	xNoErr(t, reg.SaveModel())

	err = reg.SetSave("obj1.attr1-", "a1")
	xCheck(t, err == nil, fmt.Sprintf("set foo.attr1-: %s", err))
	err = reg.SetSave("obj1.attr1-id", "a1-id")
	xCheck(t, err == nil, fmt.Sprintf("set foo.attr1-id: %s", err))
	err = reg.SetSave("obj1.foo-bar", 5)
	xCheck(t, err == nil, fmt.Sprintf("set foo.foo-bar: %s", err))

	reg.Refresh(registry.FOR_WRITE)

	val := reg.Get("obj1.attr1-")
	xCheck(t, val == "a1", fmt.Sprintf("set obj1.attr1-: %v", val))
	val = reg.Get("obj1.attr1-id")
	xCheck(t, val == "a1-id", fmt.Sprintf("set obj1.attr1-id: %v", val))
	val = reg.Get("obj1.foo-bar")
	xCheck(t, val == 5, fmt.Sprintf("set obj1.foo-bar: %v", val))

	xHTTP(t, reg, "GET", "/model", ``, 200, `{
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "required": true
    },
    "registryid": {
      "name": "registryid",
      "type": "string",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "xid": {
      "name": "xid",
      "type": "xid",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "readonly": true,
      "required": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "description": {
      "name": "description",
      "type": "string"
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "icon": {
      "name": "icon",
      "type": "url"
    },
    "labels": {
      "name": "labels",
      "type": "map",
      "item": {
        "type": "string"
      }
    },
    "createdat": {
      "name": "createdat",
      "type": "timestamp",
      "required": true
    },
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp",
      "required": true
    },
    "astring": {
      "name": "astring",
      "type": "object",
      "namecharset": "extended",
      "attributes": {
        "attr1": {
          "name": "attr1",
          "type": "string",
          "ifValues": {
            "a1-": {
              "siblingattributes": {
                "good-": {
                  "name": "good-",
                  "type": "string"
                }
              }
            }
          }
        }
      }
    },
    "obj1": {
      "name": "obj1",
      "type": "object",
      "namecharset": "extended",
      "attributes": {
        "attr1-": {
          "name": "attr1-",
          "type": "string",
          "ifValues": {
            "a1": {
              "siblingattributes": {
                "another-": {
                  "name": "another-",
                  "type": "string"
                }
              }
            }
          }
        },
        "attr1-id": {
          "name": "attr1-id",
          "type": "string"
        },
        "*": {
          "name": "*",
          "type": "integer"
        }
      }
    },
    "capabilities": {
      "name": "capabilities",
      "type": "object",
      "attributes": {
        "*": {
          "name": "*",
          "type": "any"
        }
      }
    },
    "model": {
      "name": "model",
      "type": "object",
      "readonly": true,
      "attributes": {
        "*": {
          "name": "*",
          "type": "any"
        }
      }
    },
    "modelsource": {
      "name": "modelsource",
      "type": "object",
      "attributes": {
        "*": {
          "name": "*",
          "type": "any"
        }
      }
    }
  }
}
`)

}
