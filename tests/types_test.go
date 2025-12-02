package tests

import (
	"reflect"
	"strings"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestTypesBasic(t *testing.T) {
	reg := NewRegistry("TestTypesBasic")
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
	XNoErr(t, err)
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

	XNoErr(t, reg.SaveModel())

	/* no longer required
	_, err = reg.Model.AddAttrXID("regptr_group", "")
	XCheckErr(t, err, `"model.regptr_group" must have a "target" value since "type" is "xid"`)
	*/

	_, err = reg.Model.AddAttrXID("regptr_group", "qwe")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_group\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_group\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2872"
}`)

	_, err = reg.Model.AddAttrXID("regptr_group", "qwe/")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_group\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_group\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2872"
}`)

	_, err = reg.Model.AddAttrXID("regptr_group", " /")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_group\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_group\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2872"
}`)

	_, err = reg.Model.AddAttrXID("regptr_reg", "/")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_reg\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_reg\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2872"
}`)

	_, err = reg.Model.AddAttrXID("regptr_group", "/xxxs")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_group\" has an unknown Group type: \"xxxs\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_group\" has an unknown Group type: \"xxxs\""
  },
  "source": "e4e59b8a76c4:registry:shared_model:2881"
}`)

	_, err = reg.Model.AddAttrXID("regptr_group", "/xxxs/")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_group\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_group\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2872"
}`)

	_, err = reg.Model.AddAttrXID("regptr_group", "/dirs")
	XCheckErr(t, err, ``)

	_, err = reg.Model.AddAttrXID("regptr_res", "/dirs/?")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_res\" has an unknown Resource type: \"?\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_res\" has an unknown Resource type: \"?\""
  },
  "source": "e4e59b8a76c4:registry:shared_model:2888"
}`)

	_, err = reg.Model.AddAttrXID("regptr_res", "/dirs/file")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_res\" has an unknown Resource type: \"file\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_res\" has an unknown Resource type: \"file\""
  },
  "source": "e4e59b8a76c4:registry:shared_model:2888"
}`)

	_, err = reg.Model.AddAttrXID("regptr_res", "/dirs/files")
	XNoErr(t, err)
	XNoErr(t, reg.SaveModel())

	_, err = reg.Model.AddAttrXID("regptr_ver", "/dirs/files/")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_ver\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_ver\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2872"
}`)

	_, err = reg.Model.AddAttrXID("regptr_ver", "/dirs/files/asd")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_ver\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_ver\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2872"
}`)

	_, err = reg.Model.AddAttrXID("regptr_ver", "/dirs/files/asd?")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_ver\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_ver\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2872"
}`)

	_, err = reg.Model.AddAttrXID("regptr_ver", "/dirs/files/versions")
	XNoErr(t, err)
	err = reg.SaveModel()
	XCheckErr(t, err, ``)

	_, err = reg.Model.AddAttrXID("regptr_res_ver", "/dirs/files/versions?asd")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_res_ver\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_res_ver\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2872"
}`)

	_, err = reg.Model.AddAttrXID("regptr_res_ver", "/dirs/files/versions?/")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"regptr_res_ver\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]].",
  "subject": "/model",
  "args": {
    "error_detail": "\"regptr_res_ver\" \"target\" must be of the form: /GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2872"
}`)

	_, err = reg.Model.AddAttrXID("regptr_res_ver", "/dirs/files[/versions]")
	XNoErr(t, err)

	_, err = reg.Model.AddAttrXID("regptr_res_ver2", "/dirs/files[/versions]")
	XNoErr(t, err)

	// Model is fully defined, so save it
	XNoErr(t, reg.SaveModel())

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
			{"registryid", 66, nil, `The attribute(s) "registryid" for "/" is not valid: must be a string.`},
			{"registryid", "*", nil, `The attribute(s) "registryid" for "/" is not valid: value "*" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$.`},

			{"regarrayarrayint[1][1]", 66, nil, `The attribute(s) "regarrayarrayint[1][0]" for "/" is not valid: must be an integer.`},
			{"regarrayint[0]", 1, nil, ""},
			{"regarrayint[2]", 3, nil, `The attribute(s) "regarrayint[1]" for "/" is not valid: must be an integer.`},
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
				`Bad attribute path in "/": Unexpected "~" in "regarrayint[~abc]" at pos 13.`},
			{"regarrayint['~abc']", 123, nil,
				`Bad attribute path in "/": Unexpected "~" in "regarrayint['~abc']" at pos 14.`},
			{"regmapstring.~abc", 123, nil,
				`Bad attribute path in "/": Unexpected "~" in "regmapstring.~abc" at pos 14.`},
			{"regmapstring[~abc]", 123, nil,
				`Bad attribute path in "/": Unexpected "~" in "regmapstring[~abc]" at pos 14.`},
			{"regmapstring['~abc']", 123, nil,
				`Bad attribute path in "/": Unexpected "~" in "regmapstring['~abc']" at pos 15.`},

			// Type checking
			{"epoch", -123, nil,
				`The attribute(s) "epoch" for "/" is not valid: must be a uinteger.`},
			{"regobj[1]", "", nil,
				`The attribute(s) "regobj[1]" for "/" is not valid: attribute "regobj[1]" isn't an array.`}, // Not an array
			{"regobj", []any{}, nil,
				`The attribute(s) "regobj" for "/" is not valid: must be a map[string] or object.`}, // Not an array
			{"reganyobj2.str", "substr", nil,
				`Unknown extension attribute(s) (reganyobj2) specified for: /.`}, // unknown attr
			{"regarrayarrayint[0][0]", "abc", nil,
				`The attribute(s) "regarrayarrayint[0][0]" for "/" is not valid: must be an integer.`}, // bad type
			{"regarrayint[2]", "abc", nil,
				`The attribute(s) "regarrayint[2]" for "/" is not valid: must be an integer.`}, // bad type
			{"regbool1", "123", nil,
				`The attribute(s) "regbool1" for "/" is not valid: must be a boolean.`}, // bad type
			{"regdec1", "123", nil,
				`The attribute(s) "regdec1" for "/" is not valid: must be a decimal.`}, // bad type
			{"regint1", "123", nil,
				`The attribute(s) "regint1" for "/" is not valid: must be an integer.`}, // bad type
			{"regmapint", "123", nil,
				`The attribute(s) "regmapint" for "/" is not valid: must be a map.`}, // must be empty
			{"regmapint.k1", "123", nil,
				`The attribute(s) "regmapint.k1" for "/" is not valid: must be an integer.`}, // bad type
			{"regmapstring.k1", 123, nil,
				`The attribute(s) "regmapstring.k1" for "/" is not valid: must be a string.`}, // bad type
			{"regstring1", 123, nil,
				`The attribute(s) "regstring1" for "/" is not valid: must be a string.`}, // bad type
			{"regtime1", "not a time", nil,
				`The attribute(s) "regtime1" for "/" is not valid: is a malformed timestamp.`}, // bad format
			{"reguint1", -1, nil,
				`The attribute(s) "reguint1" for "/" is not valid: must be a uinteger.`}, // bad uint
			{"unknown_int", 123, nil,
				`Unknown extension attribute(s) (unknown_int) specified for: /.`}, // unknown attr
			{"unknown_str", "error", nil,
				`Unknown extension attribute(s) (unknown_str) specified for: /.`}, // unknown attr

			{"regptr_group", "", nil, `The attribute(s) "regptr_group" for "/" is not valid: must be an xid, not empty.`},
			{"regptr_group", "/", nil, `The attribute(s) "regptr_group" for "/" is not valid: must match "/dirs" target.`},
			{"regptr_group", "/xxx", nil, `The attribute(s) "regptr_group" for "/" is not valid: must match "/dirs" target.`},
			{"regptr_group", "/dirs", nil, `The attribute(s) "regptr_group" for "/" is not valid: must match "/dirs" target, "/dirs" is missing "dirid".`},
			{"regptr_group", "/dirs2", nil, `The attribute(s) "regptr_group" for "/" is not valid: must match "/dirs" target.`},
			{"regptr_group", "/dirs", nil, `The attribute(s) "regptr_group" for "/" is not valid: must match "/dirs" target, "/dirs" is missing "dirid".`},
			{"regptr_group", "/dirs/*", nil, `The attribute(s) "regptr_group" for "/" is not valid: the "dir" ID is not valid: value "*" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$.`},
			{"regptr_group", "/dirs/id/", nil, `The attribute(s) "regptr_group" for "/" is not valid: must match "/dirs" target, extra stuff after "id".`},
			{"regptr_group", "/dirs/id/extra", nil, `The attribute(s) "regptr_group" for "/" is not valid: must match "/dirs" target, extra stuff after "id".`},
			{"regptr_group", "/dirs/id/extra/", nil, `The attribute(s) "regptr_group" for "/" is not valid: must match "/dirs" target, extra stuff after "id".`},
			{"regptr_group", "/dirs/d1", nil, ``},

			{"regptr_res", "/dirs/d1", nil, `The attribute(s) "regptr_res" for "/" is not valid: must match "/dirs/files" target, "/dirs/d1" is missing "files".`},
			{"regptr_res", "/dirs/d1/", nil, `The attribute(s) "regptr_res" for "/" is not valid: must match "/dirs/files" target, "/dirs/d1/" is missing "files".`},
			{"regptr_res", "/dirs/d1/fff", nil, `The attribute(s) "regptr_res" for "/" is not valid: must match "/dirs/files" target, "/dirs/d1/fff" is missing "files".`},
			{"regptr_res", "/dirs/d1/fff/", nil, `The attribute(s) "regptr_res" for "/" is not valid: must match "/dirs/files" target, "/dirs/d1/fff/" is missing "files".`},
			{"regptr_res", "/dirs/d1/fff/f2", nil, `The attribute(s) "regptr_res" for "/" is not valid: must match "/dirs/files" target, "/dirs/d1/fff/f2" is missing "files".`},
			{"regptr_res", "/dirs/*/files/f2", nil, `The attribute(s) "regptr_res" for "/" is not valid: the "dir" ID is not valid: value "*" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$.`},
			{"regptr_res", "/dirs/d1/files", nil, `The attribute(s) "regptr_res" for "/" is not valid: must match "/dirs/files" target, "/dirs/d1/files" is missing "fileid".`},
			{"regptr_res", "/dirs/d1/files/", nil, `The attribute(s) "regptr_res" for "/" is not valid: must match "/dirs/files" target, "/dirs/d1/files/" is missing "fileid".`},
			{"regptr_res", "/dirs/d1/files/*", nil, `The attribute(s) "regptr_res" for "/" is not valid: the "file" ID is not valid: value "*" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$.`},
			{"regptr_res", "/dirs/d1/files/f2/versions", nil, `The attribute(s) "regptr_res" for "/" is not valid: must match "/dirs/files" target, extra stuff after "f2".`},
			{"regptr_res", "/dirs/d1/files/f2/versions/v1", nil, `The attribute(s) "regptr_res" for "/" is not valid: must match "/dirs/files" target, extra stuff after "f2".`},
			{"regptr_res", "/dirs/d1/files/f2", nil, ``},

			{"regptr_ver", "/", nil, `The attribute(s) "regptr_ver" for "/" is not valid: must match "/dirs/files/versions" target.`},
			{"regptr_ver", "/dirs/d1/files/f2", nil, `The attribute(s) "regptr_ver" for "/" is not valid: must match "/dirs/files/versions" target, "/dirs/d1/files/f2" is missing "versions".`},
			{"regptr_ver", "/dirs/d1/files/f2/vvv", nil, `The attribute(s) "regptr_ver" for "/" is not valid: must match "/dirs/files/versions" target, "/dirs/d1/files/f2/vvv" is missing "versions".`},
			{"regptr_ver", "/dirs/d1/files/f2/versions", nil, `The attribute(s) "regptr_ver" for "/" is not valid: must match "/dirs/files/versions" target, "/dirs/d1/files/f2/versions" is missing a "version" ID.`},
			{"regptr_ver", "/dirs/d1/files/f2/versions/", nil, `The attribute(s) "regptr_ver" for "/" is not valid: must match "/dirs/files/versions" target, "/dirs/d1/files/f2/versions/" is missing a "version" ID.`},
			{"regptr_ver", "/dirs/d1/files/f2/versions/v2/", nil, `The attribute(s) "regptr_ver" for "/" is not valid: must match "/dirs/files/versions" target, too long.`},
			{"regptr_ver", "/dirs/d1/files/f2/versions/v2/xx", nil, `The attribute(s) "regptr_ver" for "/" is not valid: must match "/dirs/files/versions" target, too long.`},
			{"regptr_ver", "/dirs/d1/files/f2/versions/v2?", nil, `The attribute(s) "regptr_ver" for "/" is not valid: the "version" ID is not valid: value "v2?" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$.`},
			{"regptr_ver", "/dirs/d1/files/f2/versions/v2", nil, ``},

			{"regptr_res_ver", "/dirs/d1/files/", nil, `The attribute(s) "regptr_res_ver" for "/" is not valid: must match "/dirs/files[/versions]" target, "/dirs/d1/files/" is missing "fileid".`},
			{"regptr_res_ver", "/dirs/d1/files//", nil, `The attribute(s) "regptr_res_ver" for "/" is not valid: must match "/dirs/files[/versions]" target, "/dirs/d1/files//" is missing "fileid".`},
			{"regptr_res_ver", "/dirs/d1/files/f2/", nil, `The attribute(s) "regptr_res_ver" for "/" is not valid: must match "/dirs/files[/versions]" target, "/dirs/d1/files/f2/" is missing "versions".`},
			{"regptr_res_ver", "/dirs/d1/files/f2/vers", nil, `The attribute(s) "regptr_res_ver" for "/" is not valid: must match "/dirs/files[/versions]" target, "/dirs/d1/files/f2/vers" is missing "versions".`},
			{"regptr_res_ver", "/dirs/d1/files/f2/vers/v1", nil, `The attribute(s) "regptr_res_ver" for "/" is not valid: must match "/dirs/files[/versions]" target, "/dirs/d1/files/f2/vers/v1" is missing "versions".`},
			{"regptr_res_ver", "/dirs/d1/files/f*/vers/v1", nil, `The attribute(s) "regptr_res_ver" for "/" is not valid: the "file" ID is not valid: value "f*" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$.`},
			{"regptr_res_ver", "/dirs/d1/files/f2", nil, ``},

			{"regptr_res_ver2", "/dirs/d1/files/f2/versions", nil, `The attribute(s) "regptr_res_ver2" for "/" is not valid: must match "/dirs/files[/versions]" target, "/dirs/d1/files/f2/versions" is missing a "version" ID.`},
			{"regptr_res_ver2", "/dirs/d1/files/f2/versions/", nil, `The attribute(s) "regptr_res_ver2" for "/" is not valid: must match "/dirs/files[/versions]" target, "/dirs/d1/files/f2/versions/" is missing a "version" ID.`},
			{"regptr_res_ver2", "/dirs/d1/files/f2/versions//v2", nil, `The attribute(s) "regptr_res_ver2" for "/" is not valid: must match "/dirs/files[/versions]" target, "/dirs/d1/files/f2/versions//v2" is missing a "version" ID.`},
			{"regptr_res_ver2", "/dirs/d1/files/f2/versions/v2/", nil, `The attribute(s) "regptr_res_ver2" for "/" is not valid: must match "/dirs/files[/versions]" target, too long.`},
			{"regptr_res_ver2", "/dirs/d1/files/f2/versions/v*", nil, `The attribute(s) "regptr_res_ver2" for "/" is not valid: the "version" ID is not valid: value "v*" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$.`},
			{"regptr_res_ver2", "/dirs/d1/files/f2/versions/v2", nil, ``},
		}},
		Test{dir, []Prop{
			{"dirid", 66, nil, `The attribute(s) "dirid" for "/dirs/d1" is not valid: must be a string.`},
			{"dirid", "*", nil, `The attribute(s) "dirid" for "/dirs/d1" is not valid: value "*" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$.`},

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
			{"fileid", 66, nil, `The attribute(s) "fileid" for "/dirs/d1/files/f1/versions/v1" is not valid: must be a string.`},
			{"fileid", "*", nil, `The attribute(s) "fileid" for "/dirs/d1/files/f1/versions/v1" is not valid: value "*" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$.`},
			{"versionid", 66, nil, `The attribute(s) "versionid" for "/dirs/d1/files/f1/versions/v1" is not valid: must be a string.`},
			{"versionid", "*", nil, `The attribute(s) "versionid" for "/dirs/d1/files/f1/versions/v1" is not valid: value "*" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$.`},

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

			{"xid1", "", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value () isn't a valid xid, can't be an empty string.`},
			{"xid1", "//", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (//) isn't a valid xid, "//" has an empty part at position 1.`},
			{"xid1", "/dirs/d1/files/f1/versions/v1/xxx", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/d1/files/f1/versions/v1/xxx) isn't a valid xid, XID is too long.`},
			{"xid1", "/DIRS", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/DIRS) references an unknown Group type "DIRS".`},
			{"xid1", "/DIRS/d1", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/DIRS/d1) references an unknown Group type "DIRS".`},
			{"xid1", "/dirs/d1/FILES", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/d1/FILES) references an unknown Resource type "FILES".`},
			{"xid1", "/dirs/d1/FILES/f1", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/d1/FILES/f1) references an unknown Resource type "FILES".`},
			{"xid1", "/dirs/d1/files/f1/VERSIONS", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/d1/files/f1/VERSIONS) isn't a valid xid, "/dirs/d1/files/f1/VERSIONS" has "VERSIONS" at position 5, needs to be either "versions" or "meta".`},
			{"xid1", "/dirs/d1/files/f1/VERSIONS/v1", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/d1/files/f1/VERSIONS/v1) isn't a valid xid, "/dirs/d1/files/f1/VERSIONS/v1" has "VERSIONS" at position 5, needs to be either "versions" or "meta".`},
			{"xid1", "/dirs/d1/files/f1/VERSIONS/v1/xxx", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/d1/files/f1/VERSIONS/v1/xxx) isn't a valid xid, "/dirs/d1/files/f1/VERSIONS/v1/xxx" has "VERSIONS" at position 5, needs to be either "versions" or "meta".`},
			{"xid1", "/dirs/d1/files/f1/META", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/d1/files/f1/META) isn't a valid xid, "/dirs/d1/files/f1/META" has "META" at position 5, needs to be either "versions" or "meta".`},
			{"xid1", "/dirs/d1/files/f1/META/xxx", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/d1/files/f1/META/xxx) isn't a valid xid, "/dirs/d1/files/f1/META/xxx" has "META" at position 5, needs to be either "versions" or "meta".`},
			{"xid1", "/dirs/d1/files/f1/meta/xxx", nil,
				`The attribute(s) "xid1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/d1/files/f1/meta/xxx) isn't a valid xid, XID is too long.`},
			{"xid1", "/dirs/d1/files/f1/meta", nil, ""},
			{"xid2", "/dirs/d1/files/f1/versions", nil, ""},
			{"xid3", "/dirs/d1/files/f1/versions/v1", nil, ""},

			{"xidtype1", "", nil,
				`The attribute(s) "xidtype1" for "/dirs/d1/files/f1/versions/v1" is not valid: value () isn't a valid xidtype, can't be an empty string.`},
			{"xidtype1", "//", nil,
				`The attribute(s) "xidtype1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (//) isn't a valid xidtype, "//" has an empty part at position 1.`},
			{"xidtype1", "/dirs/files/versions/xxx", nil,
				`The attribute(s) "xidtype1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/files/versions/xxx) isn't a valid xidtype, XIDType is too long.`},
			{"xidtype1", "/DIRS", nil,
				`The attribute(s) "xidtype1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/DIRS) references an unknown Group type "DIRS".`},
			{"xidtype1", "/DIRS/FILES", nil,
				`The attribute(s) "xidtype1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/DIRS/FILES) references an unknown Group type "DIRS".`},
			{"xidtype1", "/dirs/FILES", nil,
				`The attribute(s) "xidtype1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/FILES) references an unknown Resource type "FILES".`},
			{"xidtype1", "/dirs/files/xxx", nil,
				`The attribute(s) "xidtype1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/files/xxx) isn't a valid xidtype, "/dirs/files/xxx" has "xxx" at position 3, needs to be either "versions" or "meta".`},
			{"xidtype1", "/dirs/files/meta/xxx", nil,
				`The attribute(s) "xidtype1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/files/meta/xxx) isn't a valid xidtype, XIDType is too long.`},
			{"xidtype1", "/dirs/files/META", nil,
				`The attribute(s) "xidtype1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/files/META) isn't a valid xidtype, "/dirs/files/META" has "META" at position 3, needs to be either "versions" or "meta".`},
			{"xidtype1", "/dirs/files/VERSIONS", nil,
				`The attribute(s) "xidtype1" for "/dirs/d1/files/f1/versions/v1" is not valid: value (/dirs/files/VERSIONS) isn't a valid xidtype, "/dirs/files/VERSIONS" has "VERSIONS" at position 3, needs to be either "versions" or "meta".`},
			{"xidtype1", "/dirs/files/meta", nil, ``},
			{"xidtype2", "/dirs/files/versions", nil, ``},
		}},
		Test{ver, []Prop{
			{"versionid", 66, nil,
				`The attribute(s) "versionid" for "/dirs/d1/files/f1/versions/v1" is not valid: must be a string.`},
			{"versionid", "*", nil,
				`The attribute(s) "versionid" for "/dirs/d1/files/f1/versions/v1" is not valid: value "*" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$.`},

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
			t.Logf("Test: %s  val:%v", prop.Name, prop.Value)

			// Note that for Resources this will set them on the default Version
			xErr := setter.SetSave(prop.Name, prop.Value)
			if xErr != nil {
				// t.Logf("xerr: %s", xErr)

				// XEqual(t, "eType", xErr.Type,
				// Type2Error["invalid_attributes"].Type)
				// XEqual(t, "eSubject", xErr.Subject, "")
				if !strings.HasSuffix(xErr.GetTitle(), prop.ErrMsg) {
					t.Errorf("Exp: %s", prop.ErrMsg)
					t.Errorf("Got: %s", xErr.GetTitle())
					t.Errorf("Source: %s", xErr.Source)
					t.FailNow()
				}
			}
			if xErr == nil && prop.ErrMsg != "" {
				t.Logf("Test: %s  val:%v", prop.Name, prop.Value)
				t.Errorf("Setting (%q=%v) was supposed to fail:\nExp: %s",
					prop.Name, prop.Value, prop.ErrMsg)
				t.FailNow()
			}
			if xErr != nil {
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
				t.FailNow()
			}
		}
	}

	XCheckGet(t, reg, "?inline", `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestTypesBasic",
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
          "filebase64": "",

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
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
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
              "xidtype2": "/dirs/files/versions",
              "filebase64": ""
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

func TestTypesWildcardBool(t *testing.T) {
	reg := NewRegistry("TestTypesWildcardBool")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttr("*", BOOLEAN)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, err)
	XNoErr(t, reg.Model.Save())

	dir, err := reg.AddGroup("dirs", "d1")
	XNoErr(t, err)
	_, err = dir.AddResource("files", "f1", "v1")
	XNoErr(t, err)

	err = reg.SetSave("bogus", "foo")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attributes",
  "title": "The attribute(s) \"bogus\" for \"/\" is not valid: must be a boolean.",
  "subject": "/",
  "args": {
    "error_detail": "must be a boolean",
    "list": "bogus"
  },
  "source": "e4e59b8a76c4:registry:entity:2369"
}`)

	err = reg.SetSave("ext1", true)
	XCheck(t, err == nil, "set ext1: %s", err)
	reg.Refresh(registry.FOR_WRITE)
	val := reg.Get("ext1")
	XCheck(t, val == true, "get ext1: %v", val)

	err = reg.SetSave("ext1", false)
	XCheck(t, err == nil, "set ext1-2: %s", err)
	reg.Refresh(registry.FOR_WRITE)
	XCheck(t, reg.Get("ext1") == false, "get ext1-2: %v", val)
}

func TestTypesWildcardAny(t *testing.T) {
	reg := NewRegistry("TestTypesWildcardAny")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttr("*", ANY)
	XNoErr(t, reg.Model.Save())

	// Make sure we can set the same attr to two different types
	err := reg.SetSave("ext1", 5.5)
	XCheck(t, err == nil, "set ext1: %s", err)
	reg.Refresh(registry.FOR_WRITE)
	val := reg.Get("ext1")
	XCheck(t, val == 5.5, "get ext1: %v", val)

	err = reg.SetSave("ext1", "foo")
	XCheck(t, err == nil, "set ext2: %s", err)
	reg.Refresh(registry.FOR_WRITE)
	val = reg.Get("ext1")
	XCheck(t, val == "foo", "get ext2: %v", val)

	// Make sure we add one of a different type
	err = reg.SetSave("ext2", true)
	XCheck(t, err == nil, "set ext3 %s", err)
	reg.Refresh(registry.FOR_WRITE)
	val = reg.Get("ext2")
	XCheck(t, val == true, "get ext3: %v", val)
}

func TestTypesWildcard2Layers(t *testing.T) {
	reg := NewRegistry("TestTypesWildcardAny")
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
	XCheck(t, err == nil, "")
	XNoErr(t, reg.Model.Save())

	err = reg.SetSave("obj.map.k1", 5)
	XCheck(t, err == nil, "set foo.k1: %s", err)
	reg.Refresh(registry.FOR_WRITE)
	val := reg.Get("obj.map.k1")
	XCheck(t, val == 5, "get foo.k1: %v", val)

	err = reg.SetSave("obj.map.foo.k1.k2", 5)
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attributes",
  "title": "The attribute(s) \"obj.map.foo\" for \"/\" is not valid: must be an integer.",
  "subject": "/",
  "args": {
    "error_detail": "must be an integer",
    "list": "obj.map.foo"
  },
  "source": "e4e59b8a76c4:registry:entity:2388"
}`)
	// reg.Refresh(registry.FOR_WRITE) // clear bad data

	err = reg.SetSave("obj.myany.foo.k1.k2", 5)
	XCheck(t, err == nil, "set obj.myany.foo.k1.k2: %s", err)
	reg.Refresh(registry.FOR_WRITE)
	val = reg.Get("obj.myany.foo.k1.k2")
	XCheck(t, val == 5, "set obj.myany.foo.k1.k2: %v", val)
	val = reg.Get("obj.myany.bogus.k1.k2")
	XCheck(t, val == nil, "set obj.myany.bogus.k1.k2: %v", val)

}

func TestTypesNameCharSet(t *testing.T) {
	reg := NewRegistry("TestTypesNameCharSet")
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
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attributes",
  "title": "The attribute(s) \"attr1-\" for \"/obj1\" is not valid: must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/obj1",
  "args": {
    "error_detail": "must match: ^[a-z_][a-z_0-9]{0,62}$",
    "list": "attr1-"
  },
  "source": "e4e59b8a76c4:registry:shared_model:47"
}`)

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
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attributes",
  "title": "The attribute(s) \"attr1-\" for \"/obj1\" is not valid: must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/obj1",
  "args": {
    "error_detail": "must match: ^[a-z_][a-z_0-9]{0,62}$",
    "list": "attr1-"
  },
  "source": "e4e59b8a76c4:registry:shared_model:47"
}`)

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
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attributes",
  "title": "The attribute(s) \"another-\" for \"/obj1.attr1.ifvalues.a1\" is not valid: must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/obj1.attr1.ifvalues.a1",
  "args": {
    "error_detail": "must match: ^[a-z_][a-z_0-9]{0,62}$",
    "list": "another-"
  },
  "source": "e4e59b8a76c4:registry:shared_model:47"
}`)

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
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: map key name \"attr space\" in \"obj1\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "map key name \"attr space\" in \"obj1\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$"
  },
  "source": "e4e59b8a76c4:registry:shared_model:65"
}`)

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
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attributes",
  "title": "The attribute(s) \"bad-\" for \"/astring.ifvalues.a1\" is not valid: must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/astring.ifvalues.a1",
  "args": {
    "error_detail": "must match: ^[a-z_][a-z_0-9]{0,62}$",
    "list": "bad-"
  },
  "source": "e4e59b8a76c4:registry:shared_model:47,registry:shared_model:2825"
}`)

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
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attributes",
  "title": "The attribute(s) \"bad-\" for \"/astring.attr1.ifvalues.a1\" is not valid: must match: ^[a-z_][a-z_0-9]{0,62}$.",
  "subject": "/astring.attr1.ifvalues.a1",
  "args": {
    "error_detail": "must match: ^[a-z_][a-z_0-9]{0,62}$",
    "list": "bad-"
  },
  "source": "e4e59b8a76c4:registry:shared_model:47,registry:shared_model:2825"
}`)

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
	XNoErr(t, err)

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
	XNoErr(t, err)
	XNoErr(t, reg.SaveModel())

	err = reg.SetSave("obj1.attr1-", "a1")
	XCheck(t, err == nil, "set foo.attr1-: %s", err)
	err = reg.SetSave("obj1.attr1-id", "a1-id")
	XCheck(t, err == nil, "set foo.attr1-id: %s", err)
	err = reg.SetSave("obj1.foo-bar", 5)
	XCheck(t, err == nil, "set foo.foo-bar: %s", err)

	reg.Refresh(registry.FOR_WRITE)

	val := reg.Get("obj1.attr1-")
	XCheck(t, val == "a1", "set obj1.attr1-: %v", val)
	val = reg.Get("obj1.attr1-id")
	XCheck(t, val == "a1-id", "set obj1.attr1-id: %v", val)
	val = reg.Get("obj1.foo-bar")
	XCheck(t, val == 5, "set obj1.foo-bar: %v", val)

	XHTTP(t, reg, "GET", "/model", ``, 200, `{
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
          "ifvalues": {
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
          "ifvalues": {
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
