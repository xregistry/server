package tests

import (
	"testing"

	. "github.com/xregistry/server/common"
)

func TestFiltersBasic(t *testing.T) {
	reg := NewRegistry("TestFiltersBasic")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true)
	XNoErr(t, err)
	XNoErr(t, reg.SaveModel(true))

	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	d, _ = reg.AddGroup("dirs", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")

	reg.SetSave("labels.reg1", "1ger")
	f.SetSaveDefault("labels.file1", "1elif")

	// /dirs/d1/f1/v1
	//            /v2
	//      /d2/f2/v1
	//             v1.1

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "No Filter",
			URL:  "?",
			Exp: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestFiltersBasic",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "labels": {
    "reg1": "1ger"
  },
  "createdat": "2024-12-01T12:00:01Z",
  "modifiedat": "2024-12-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 2
}
`,
		},
		{
			Name: "Inline - No Filter",
			URL:  "?inline&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "2 leaves match",
			URL:  "?inline&oneline&filter=dirs.files.versions.versionid=v1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{}}}}}}}`,
		},
		{
			Name: "Just one leaf - v2",
			URL:  "?inline&oneline&filter=dirs.files.versions.versionid=v2",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v2":{}}}}}}}`,
		},
		{
			Name: "filter at file level",
			URL:  "?inline&oneline&filter=dirs.files.fileid=f2",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "get groups, filter at resource level",
			URL:  "dirs?inline&oneline&filter=files.fileid=f2",
			Exp:  `{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}`,
		},
		{ // Test some filtering at the root of the GET
			Name: "Get/filter root - match ",
			URL:  "?inline&oneline&filter=registryid=TestFiltersBasic",
			// Return entire tree
			Exp: `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "Get/filter root, no match",
			URL:  "?inline&filter=registryid=xxx",
			// Nothing matched so 404
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "Get root, filter group coll - match",
			URL:  "?inline&oneline&filter=dirs.dirid=d1",
			// Just root + dirs/d1
			Exp: `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}}}}`,
		},
		{
			Name: "Get root, filter group coll - no match",
			URL:  "?inline&filter=dirs.dirid=xxx",
			// Nothing, matched, so 404
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "Get/filter group coll - match",
			URL:  "dirs?inline&oneline&filter=dirid=d1",
			// dirs coll with just d1
			Exp: `{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}}}`,
		},
		{
			Name: "Get/filter group coll - no match",
			URL:  "dirs?inline&oneline&filter=dirid=xxx",
			Exp:  "{}",
		},
		{
			Name: "Get/filter group entity - match",
			URL:  "dirs/d1?inline&oneline&filter=dirid=d1",
			// entire d1 group
			Exp: `{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}}`,
		},
		{
			Name: "Get/filter group entity - no match",
			URL:  "dirs/d1?inline&filter=dirid=xxx",
			// Nothing, matched, so 404
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1) cannot be found.",
  "subject": "/dirs/d1",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "Get group entity, filter resource - match",
			URL:  "dirs/d1?inline&oneline&filter=files.fileid=f1",
			// entire d1
			Exp: `{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}}`,
		},
		{
			Name: "Get group entity, filter resource - no match",
			URL:  "dirs/d1?inline&filter=files.fileid=xxx",
			// Nothing, matched, so 404
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1) cannot be found.",
  "subject": "/dirs/d1",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "Get/filter version coll - match",
			URL:  "dirs/d1/files/f1/versions?inline&oneline&filter=versionid=v1",
			Exp:  `{"v1":{}}`,
		},
		{
			Name: "Get/filter version coll - no match",
			URL:  "dirs/d1/files/f1/versions?inline&oneline&filter=versionid=xxx",
			Exp:  "{}",
		},
		{
			Name: "Get/filter version - match",
			URL:  "dirs/d1/files/f1/versions/v1$details?inline&filter=versionid=v1",
			Exp: `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": false,
  "createdat": "2024-12-01T12:00:00Z",
  "modifiedat": "2024-12-01T12:00:00Z",
  "ancestorid": "v1",
  "filebase64": ""
}
`,
		},
		{
			Name: "Get/filter version - no match",
			URL:  "dirs/d1/files/f1/versions/v1$details?inline&filter=versionid=xxx",
			// Nothing, matched, so 404
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/versions/v1$details) cannot be found.",
  "subject": "/dirs/d1/files/f1/versions/v1$details",
  "source": ":registry:httpStuff:1730"
}
`,
		},

		// Some tag filters
		{
			Name: "Get/filter reg.labels - no match",
			URL:  "?filter=labels.reg1=xxx",
			// Nothing, matched, so 404
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "Get/filter reg.labels - match",
			URL:  "?filter=labels.reg1=1ger",
			Exp: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestFiltersBasic",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "labels": {
    "reg1": "1ger"
  },
  "createdat": "2024-12-01T12:00:01Z",
  "modifiedat": "2024-12-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 2
}
`,
		},
		{
			Name: "Get/filter labels",
			URL:  "?filter=dirs.files.labels.file1=1elif",
			Exp: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestFiltersBasic",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "labels": {
    "reg1": "1ger"
  },
  "createdat": "2024-12-01T12:00:01Z",
  "modifiedat": "2024-12-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs?filter=files.labels.file1=1elif",
  "dirscount": 1
}
`,
		},
		{
			Name: "Get/filter dir file.labels - match 1elif",
			URL:  "?inline&filter=dirs.files.labels.file1=1elif",
			Exp: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestFiltersBasic",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "labels": {
    "reg1": "1ger"
  },
  "createdat": "2024-12-01T12:00:01Z",
  "modifiedat": "2024-12-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs?filter=files.labels.file1=1elif",
  "dirs": {
    "d2": {
      "dirid": "d2",
      "self": "http://localhost:8181/dirs/d2",
      "xid": "/dirs/d2",
      "epoch": 1,
      "createdat": "2024-12-01T12:00:02Z",
      "modifiedat": "2024-12-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d2/files?filter=labels.file1=1elif",
      "files": {
        "f2": {
          "fileid": "f2",
          "versionid": "v1.1",
          "self": "http://localhost:8181/dirs/d2/files/f2$details",
          "xid": "/dirs/d2/files/f2",
          "epoch": 1,
          "isdefault": true,
          "labels": {
            "file1": "1elif"
          },
          "createdat": "2024-12-01T12:00:02Z",
          "modifiedat": "2024-12-01T12:00:02Z",
          "ancestorid": "v1",
          "filebase64": "",

          "metaurl": "http://localhost:8181/dirs/d2/files/f2/meta",
          "meta": {
            "fileid": "f2",
            "self": "http://localhost:8181/dirs/d2/files/f2/meta",
            "xid": "/dirs/d2/files/f2/meta",
            "epoch": 1,
            "createdat": "2024-12-01T12:00:02Z",
            "modifiedat": "2024-12-01T12:00:02Z",
            "readonly": false,

            "defaultversionid": "v1.1",
            "defaultversionurl": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d2/files/f2/versions",
          "versions": {
            "v1": {
              "fileid": "f2",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1$details",
              "xid": "/dirs/d2/files/f2/versions/v1",
              "epoch": 1,
              "isdefault": false,
              "createdat": "2024-12-01T12:00:02Z",
              "modifiedat": "2024-12-01T12:00:02Z",
              "ancestorid": "v1",
              "filebase64": ""
            },
            "v1.1": {
              "fileid": "f2",
              "versionid": "v1.1",
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1$details",
              "xid": "/dirs/d2/files/f2/versions/v1.1",
              "epoch": 1,
              "isdefault": true,
              "labels": {
                "file1": "1elif"
              },
              "createdat": "2024-12-01T12:00:02Z",
              "modifiedat": "2024-12-01T12:00:02Z",
              "ancestorid": "v1",
              "filebase64": ""
            }
          },
          "versionscount": 2
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`,
		},
		{
			Name: "Get/filter dir file.labels - no match empty string",
			URL:  "?inline&filter=dirs.files.labels.file1=",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "Get/filter dir file.labels.xxx - no match empty string",
			URL:  "?inline&filter=dirs.files.labels.xxx=",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "Get/filter dir file.labels.xxx - no match non-empty string",
			URL:  "?inline&filter=dirs.files.labels.xxx",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "Get/filter dir file.labels - match non-empty string",
			URL:  "?inline&filter=dirs.files.labels.file1",
			Exp: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestFiltersBasic",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "labels": {
    "reg1": "1ger"
  },
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs?filter=files.labels.file1",
  "dirs": {
    "d2": {
      "dirid": "d2",
      "self": "http://localhost:8181/dirs/d2",
      "xid": "/dirs/d2",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d2/files?filter=labels.file1",
      "files": {
        "f2": {
          "fileid": "f2",
          "versionid": "v1.1",
          "self": "http://localhost:8181/dirs/d2/files/f2$details",
          "xid": "/dirs/d2/files/f2",
          "epoch": 1,
          "isdefault": true,
          "labels": {
            "file1": "1elif"
          },
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestorid": "v1",
          "filebase64": "",

          "metaurl": "http://localhost:8181/dirs/d2/files/f2/meta",
          "meta": {
            "fileid": "f2",
            "self": "http://localhost:8181/dirs/d2/files/f2/meta",
            "xid": "/dirs/d2/files/f2/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,

            "defaultversionid": "v1.1",
            "defaultversionurl": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d2/files/f2/versions",
          "versions": {
            "v1": {
              "fileid": "f2",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1$details",
              "xid": "/dirs/d2/files/f2/versions/v1",
              "epoch": 1,
              "isdefault": false,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestorid": "v1",
              "filebase64": ""
            },
            "v1.1": {
              "fileid": "f2",
              "versionid": "v1.1",
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1$details",
              "xid": "/dirs/d2/files/f2/versions/v1.1",
              "epoch": 1,
              "isdefault": true,
              "labels": {
                "file1": "1elif"
              },
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestorid": "v1",
              "filebase64": ""
            }
          },
          "versionscount": 2
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`,
		},
	}

	for _, test := range tests {
		t.Logf("Test name: %s", test.Name)
		XCheckGet(t, reg, test.URL, test.Exp)
	}
}

func TestFiltersANDOR(t *testing.T) {
	reg := NewRegistry("TestFiltersANDOR")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true)
	XNoErr(t, err)
	XNoErr(t, reg.SaveModel(true))

	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	f.SetSaveDefault("name", "f1")
	d, _ = reg.AddGroup("dirs", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")
	f.SetSaveDefault("name", "f2")

	gm, err = reg.Model.AddGroupModel("schemagroups", "schemagroup")
	XNoErr(t, err)
	_, err = gm.AddResourceModel("schemas", "schema", 0, true, true)
	XNoErr(t, err)
	XNoErr(t, reg.SaveModel(true))

	sg, err := reg.AddGroup("schemagroups", "sg1")
	XNoErr(t, err)
	s, err := sg.AddResource("schemas", "s1", "v1.0")
	XNoErr(t, err)
	s.AddVersion("v2.0")

	reg.SetSave("labels.reg1", "1ger")
	f.SetSaveDefault("labels.file1", "1elif")

	// /dirs/d1/f1/v1     f1.name=f1
	//            /v2
	//      /d2/f2/v1     f2.name=f2
	//             v1.1
	// /s/sg1/schemas/s1/v1.0
	//                             /v2.0

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "AND same obj/level - match",
			URL:  "?oneline&inline&filter=dirs.files.fileid=f1,dirs.files.name=f1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}}},"schemagroups":{}}`,
		},
		{
			Name: "AND same obj/level - no match",
			URL:  "?inline&filter=dirs.files.fileid=f1,dirs.files.name=f2",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "OR same obj/level - match",
			URL:  "?oneline&inline&filter=dirs.files.fileid=f1&filter=dirs.files.name=f1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}}},"schemagroups":{}}`,
		},
		{
			Name: "multi result 2 levels down - match",
			URL:  "?oneline&inline&filter=dirs.files.versions.versionid=v1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{}}}}}},"schemagroups":{}}`,
		},
		{
			Name: "path + multi result 2 levels down - match",
			URL:  "dirs?oneline&inline&filter=files.versions.versionid=v1",
			Exp:  `{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{}}}}}}`,
		},
		{
			Name: "path + multi result 2 levels down - match",
			URL:  "dirs?oneline&inline&filter=files.versions.versionid=v1*",
			Exp:  `{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}`,
		},
		{
			Name: "path + multi result 2 levels down - no match",
			URL:  "dirs?oneline&inline&filter=files.versions.versionid=xxx",
			Exp:  `{}`,
		},

		// Span group types
		{
			Name: "dirs and s - match both",
			URL:  "?oneline&inline&filter=dirs.dirid=d1&filter=schemagroups.schemagroupid=sg1",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}}},"schemagroups":{"sg1":{"schemas":{"s1":{"meta":{},"versions":{"v1.0":{},"v2.0":{}}}}}}}`,
		},
		{
			Name: "dirs and s - match first",
			URL:  "?oneline&inline&filter=dirs.dirid=d1&filter=schemagroups.schemagroupid=xxx",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}}},"schemagroups":{}}`,
		},
		{
			Name: "dirs and s - match second",
			URL:  "?oneline&inline&filter=dirs.dirid=xxx&filter=schemagroups.schemagroupid=sg1",
			Exp:  `{"dirs":{},"schemagroups":{"sg1":{"schemas":{"s1":{"meta":{},"versions":{"v1.0":{},"v2.0":{}}}}}}}`,
		},
		{
			Name: "dirsOR and sOR - match first",
			URL:  "?oneline&inline&filter=dirs.files.fileid=f1,dirs.files.versions.versionid=v2&filter=schemagroups.schemas.versions.versionid=v1.0,schemagroups.schemas.versions.versionid=v2.0",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v2":{}}}}}},"schemagroups":{}}`,
		},
		{
			Name: "dirsOR and sOR - match second",
			URL:  "?oneline&inline&filter=dirs.files.fileid=f1,dirs.files.versions.versionid=xxx&filter=schemagroups.schemas.versions.versionid=v2.0,schemagroups.schemas.meta.defaultversionid=v2.0",
			Exp:  `{"dirs":{},"schemagroups":{"sg1":{"schemas":{"s1":{"meta":{},"versions":{"v2.0":{}}}}}}}`,
		},
		{
			Name: "dirsOR and sOR - both match",
			URL:  "?oneline&inline&filter=dirs.files.fileid=f1,dirs.files.versions.versionid=v2&filter=schemagroups.schemas.versions.versionid=v2.0,schemagroups.schemas.meta.defaultversionid=v2.0",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v2":{}}}}}},"schemagroups":{"sg1":{"schemas":{"s1":{"meta":{},"versions":{"v2.0":{}}}}}}}`,
		},
	}

	for _, test := range tests {
		t.Logf("Test name: %s", test.Name)
		XCheckGet(t, reg, test.URL, test.Exp)
	}
}

func TestFiltersWildcards(t *testing.T) {
	reg := NewRegistry("TestFiltersWildcards")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true)
	XNoErr(t, err)
	XNoErr(t, reg.SaveModel(true))

	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	f.SetSaveDefault("name", "f1")
	d, _ = reg.AddGroup("dirs", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")
	f.SetSaveDefault("name", "f123")
	f, _ = d.AddResource("files", "f3", "v1")
	f.AddVersion("v1.1")
	f.SetSaveDefault("name", "g%d")
	f, _ = d.AddResource("files", "f4", "v1") // No name at all

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "wildcard at start",
			URL:  "?oneline&inline&filter=dirs.files.name=*3",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "wildcard at end - 1",
			URL:  "?oneline&inline&filter=dirs.files.name=f*",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "wildcard at end - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=f12*",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "wildcard at both ends - 1",
			URL:  "?oneline&inline&filter=dirs.files.name=*f12*",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "wildcard at both ends - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=*12*",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "wildcard at both ends - 3",
			URL:  "?oneline&inline&filter=dirs.files.name=*3*",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "double wildcard - 1",
			URL:  "?oneline&inline&filter=dirs.files.name=**3",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "double wildcard - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=**2**",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "double wildcard - 3",
			URL:  "?oneline&inline&filter=dirs.files.name=f**3",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "multi-wildcard - 1",
			URL:  "?oneline&inline&filter=dirs.files.name=f*1*2*3",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "multi-wildcard - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=*f*1*2*3",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "multi-wildcard - 3",
			URL:  "?oneline&inline&filter=dirs.files.name=f*1*2*3*",
			Exp:  `{"dirs":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "escape - 1",
			URL:  "?oneline&inline&filter=dirs.files.name=g%25d",
			Exp:  `{"dirs":{"d2":{"files":{"f3":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "escape - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=g*d",
			Exp:  `{"dirs":{"d2":{"files":{"f3":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "escape - 3",
			URL:  "?inline&filter=dirs.files.name=g\\*d",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "all - 1",
			URL:  "?oneline&inline&filter=dirs.files.name", // name must be set
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}},"f3":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "all - 2",
			URL:  "?oneline&inline&filter=dirs.files.name=*", // name must be set
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}},"f3":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "all - 3",
			URL:  "?oneline&inline", // verify same as name=* + f4
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}},"f3":{"meta":{},"versions":{"v1":{},"v1.1":{}}},"f4":{"meta":{},"versions":{"v1":{}}}}}}}`,
		},
		{
			Name: "fail - 1",
			URL:  "?inline&filter=dirs.files.name=f*x",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "fail - 2",
			URL:  "?inline&filter=dirs.files.name=*f",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "fail - 3",
			URL:  "?inline&filter=dirs.files.name=z*",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "fail - 4",
			URL:  "?inline&filter=dirs.files.name=*z*",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "fail - 5",
			URL:  "?inline&filter=dirs.files.name=**z**",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "fail - 6",
			URL:  "?inline&filter=dirs.files.description=*",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
	}

	for _, test := range tests {
		t.Logf("Test name: %s", test.Name)
		XCheckGet(t, reg, test.URL, test.Exp)
	}
}

func TestFiltersOps(t *testing.T) {
	reg := NewRegistry("TestFiltersOps")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	rm, err := gm.AddResourceModel("files", "file", 0, true, false) // nodoc
	XNoErr(t, err)
	rm.AddAttr("count", INTEGER)
	XNoErr(t, reg.SaveModel(true))

	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.SetSaveDefault("name", "bob")
	f.SetSaveDefault("count", 3)

	f, _ = d.AddResource("files", "f2", "v1")
	f.SetSaveDefault("name", "")
	f.SetSaveDefault("count", 7)

	d.AddResource("files", "f3", "v1") // no "name", no "count"

	PRE := "?oneline&inline=dirs.files&filter="
	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "name=bob",
			URL:  PRE + "dirs.files.name=bob",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{}}}}}`,
		},
		{
			Name: "name!=bob",
			URL:  PRE + "dirs.files.name!=bob",
			Exp:  `{"dirs":{"d1":{"files":{"f2":{},"f3":{}}}}}`,
		},
		{
			Name: "name=null",
			URL:  PRE + "dirs.files.name=null",
			Exp:  `{"dirs":{"d1":{"files":{"f3":{}}}}}`,
		},
		{
			Name: "name!=null",
			URL:  PRE + "dirs.files.name!=null",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{},"f2":{}}}}}`,
		},
		{
			Name: "name (present)",
			URL:  PRE + "dirs.files.name",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{},"f2":{}}}}}`,
		},
		{
			Name: "name!=bob && name (present)",
			URL:  PRE + "dirs.files.name!=bob,dirs.files.name",
			Exp:  `{"dirs":{"d1":{"files":{"f2":{}}}}}`,
		},
		{
			Name: "name!=bob || name (present)",
			URL:  PRE + "dirs.files.name!=bob&filter=dirs.files.name",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{},"f2":{},"f3":{}}}}}`,
		},

		// Non-root
		{
			Name: "non-root name=bob",
			URL:  "/dirs/d1/files?oneline&filter=name=bob",
			Exp:  `{"f1":{}}`,
		},
		{
			Name: "non-root name!=bob",
			URL:  "/dirs/d1/files?oneline&filter=name!=bob",
			Exp:  `{"f2":{},"f3":{}}`,
		},
		{
			Name: "non-root name=null",
			URL:  "/dirs/d1/files?oneline&filter=name=null",
			Exp:  `{"f3":{}}`,
		},
		{
			Name: "non-root name!=null",
			URL:  "/dirs/d1/files?oneline&filter=name!=null",
			Exp:  `{"f1":{},"f2":{}}`,
		},
		{
			Name: "non-root name (present)",
			URL:  "/dirs/d1/files?oneline&filter=name",
			Exp:  `{"f1":{},"f2":{}}`,
		},
		{
			Name: "non-root name!=bob && name (present)",
			URL:  "/dirs/d1/files?oneline&filter=name!=bob,name",
			Exp:  `{"f2":{}}`,
		},
		{
			Name: "non-root name!=bob || name (present)",
			URL:  "/dirs/d1/files?oneline&filter=name!=bob&filter=name",
			Exp:  `{"f1":{},"f2":{},"f3":{}}`,
		},

		// Case-insensitive string comparison (spec: strings MUST be compared case-insensitively)
		{
			Name: "name=BOB (case-insensitive =)",
			URL:  PRE + "dirs.files.name=BOB",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{}}}}}`,
		},
		{
			Name: "name=Bob (case-insensitive =)",
			URL:  PRE + "dirs.files.name=Bob",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{}}}}}`,
		},
		{
			Name: "name!=BOB (case-insensitive !=)",
			URL:  PRE + "dirs.files.name!=BOB",
			Exp:  `{"dirs":{"d1":{"files":{"f2":{},"f3":{}}}}}`,
		},
		{
			Name: "name<>BOB (case-insensitive <>)",
			URL:  PRE + "dirs.files.name<>BOB",
			Exp:  `{"dirs":{"d1":{"files":{"f2":{},"f3":{}}}}}`,
		},
		{
			Name: "name<BOB (case-insensitive <)",
			URL:  PRE + "dirs.files.name<BOB",
			Exp:  `{"dirs":{"d1":{"files":{"f2":{}}}}}`,
		},
		{
			Name: "name<=BOB (case-insensitive <=)",
			URL:  PRE + "dirs.files.name<=BOB",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{},"f2":{}}}}}`,
		},
		{
			Name: "name>bob upper (case-insensitive >)",
			URL:  "/dirs/d1/files?oneline&filter=name>BOB",
			Exp:  `{}`,
		},
		{
			Name: "name>=BOB (case-insensitive >=)",
			URL:  PRE + "dirs.files.name>=BOB",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{}}}}}`,
		},

		// Comparison operators - string field
		{
			Name: "name<bob",
			URL:  PRE + "dirs.files.name<bob",
			Exp:  `{"dirs":{"d1":{"files":{"f2":{}}}}}`,
		},
		{
			Name: "name<=bob",
			URL:  PRE + "dirs.files.name<=bob",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{},"f2":{}}}}}`,
		},
		{
			Name: "name>\"\"",
			URL:  PRE + "dirs.files.name>",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{}}}}}`,
		},
		{
			Name: "name>=\"\"",
			URL:  PRE + "dirs.files.name>=",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{},"f2":{}}}}}`,
		},
		{
			Name: "name<>bob",
			URL:  PRE + "dirs.files.name<>bob",
			Exp:  `{"dirs":{"d1":{"files":{"f2":{},"f3":{}}}}}`,
		},

		// Comparison operators - non-root string field
		{
			Name: "non-root name<bob",
			URL:  "/dirs/d1/files?oneline&filter=name<bob",
			Exp:  `{"f2":{}}`,
		},
		{
			Name: "non-root name<=bob",
			URL:  "/dirs/d1/files?oneline&filter=name<=bob",
			Exp:  `{"f1":{},"f2":{}}`,
		},
		{
			Name: "non-root name>\"\"",
			URL:  "/dirs/d1/files?oneline&filter=name>",
			Exp:  `{"f1":{}}`,
		},
		{
			Name: "non-root name>=\"\"",
			URL:  "/dirs/d1/files?oneline&filter=name>=",
			Exp:  `{"f1":{},"f2":{}}`,
		},
		{
			Name: "non-root name<>bob",
			URL:  "/dirs/d1/files?oneline&filter=name<>bob",
			Exp:  `{"f2":{},"f3":{}}`,
		},

		// Comparison operators - integer field
		{
			Name: "count>3",
			URL:  PRE + "dirs.files.count>3",
			Exp:  `{"dirs":{"d1":{"files":{"f2":{}}}}}`,
		},
		{
			Name: "count>=3",
			URL:  PRE + "dirs.files.count>=3",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{},"f2":{}}}}}`,
		},
		{
			Name: "count<7",
			URL:  PRE + "dirs.files.count<7",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{}}}}}`,
		},
		{
			Name: "count<=7",
			URL:  PRE + "dirs.files.count<=7",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{},"f2":{}}}}}`,
		},
		{
			Name: "count<>3",
			URL:  PRE + "dirs.files.count<>3",
			Exp:  `{"dirs":{"d1":{"files":{"f2":{},"f3":{}}}}}`,
		},

		// Comparison operators - non-root integer field
		{
			Name: "non-root count>3",
			URL:  "/dirs/d1/files?oneline&filter=count>3",
			Exp:  `{"f2":{}}`,
		},
		{
			Name: "non-root count>=3",
			URL:  "/dirs/d1/files?oneline&filter=count>=3",
			Exp:  `{"f1":{},"f2":{}}`,
		},
		{
			Name: "non-root count<7",
			URL:  "/dirs/d1/files?oneline&filter=count<7",
			Exp:  `{"f1":{}}`,
		},
		{
			Name: "non-root count<=7",
			URL:  "/dirs/d1/files?oneline&filter=count<=7",
			Exp:  `{"f1":{},"f2":{}}`,
		},
		{
			Name: "non-root count<>3",
			URL:  "/dirs/d1/files?oneline&filter=count<>3",
			Exp:  `{"f2":{},"f3":{}}`,
		},

		// Spec compliance: <> null must be FILTER_PRESENT (same as !=null)
		{
			Name: "name<>null is FILTER_PRESENT (spec)",
			URL:  PRE + "dirs.files.name<>null",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{},"f2":{}}}}}`,
		},
		{
			Name: "non-root name<>null is FILTER_PRESENT (spec)",
			URL:  "/dirs/d1/files?oneline&filter=name<>null",
			Exp:  `{"f1":{},"f2":{}}`,
		},

		// Spec compliance: null not allowed with <, <=, >, >=
		{
			Name: "count<null is bad_filter (spec)",
			URL:  "/dirs/d1/files?oneline&filter=count<null",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_filter",
  "title": "An error was found in \"filter\" value (count<null): null is not allowed with <, <=, >, >= operators.",
  "subject": "/dirs/d1/files",
  "args": {
    "error_detail": "null is not allowed with <, <=, >, >= operators",
    "value": "count<null"
  },
  "source": "xxx"
}
`,
		},
		{
			Name: "count<=null is bad_filter (spec)",
			URL:  "/dirs/d1/files?oneline&filter=count<=null",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_filter",
  "title": "An error was found in \"filter\" value (count<=null): null is not allowed with <, <=, >, >= operators.",
  "subject": "/dirs/d1/files",
  "args": {
    "error_detail": "null is not allowed with <, <=, >, >= operators",
    "value": "count<=null"
  },
  "source": "xxx"
}
`,
		},
		{
			Name: "count>null is bad_filter (spec)",
			URL:  "/dirs/d1/files?oneline&filter=count>null",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_filter",
  "title": "An error was found in \"filter\" value (count>null): null is not allowed with <, <=, >, >= operators.",
  "subject": "/dirs/d1/files",
  "args": {
    "error_detail": "null is not allowed with <, <=, >, >= operators",
    "value": "count>null"
  },
  "source": "xxx"
}
`,
		},
		{
			Name: "count>=null is bad_filter (spec)",
			URL:  "/dirs/d1/files?oneline&filter=count>=null",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_filter",
  "title": "An error was found in \"filter\" value (count>=null): null is not allowed with <, <=, >, >= operators.",
  "subject": "/dirs/d1/files",
  "args": {
    "error_detail": "null is not allowed with <, <=, >, >= operators",
    "value": "count>=null"
  },
  "source": "xxx"
}
`,
		},

		// Spec compliance: wildcards not allowed with <, <=, >, >=
		{
			Name: "count<3* is bad_filter (spec)",
			URL:  "/dirs/d1/files?oneline&filter=count<3*",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_filter",
  "title": "An error was found in \"filter\" value (count<3*): wildcards are not allowed with <, <=, >, >= operators.",
  "subject": "/dirs/d1/files",
  "args": {
    "error_detail": "wildcards are not allowed with <, <=, >, >= operators",
    "value": "count<3*"
  },
  "source": "xxx"
}
`,
		},
		{
			Name: "count>=2* is bad_filter (spec)",
			URL:  "/dirs/d1/files?oneline&filter=count>=2*",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_filter",
  "title": "An error was found in \"filter\" value (count>=2*): wildcards are not allowed with <, <=, >, >= operators.",
  "subject": "/dirs/d1/files",
  "args": {
    "error_detail": "wildcards are not allowed with <, <=, >, >= operators",
    "value": "count>=2*"
  },
  "source": "xxx"
}
`,
		},
	}

	for _, test := range tests {
		t.Logf("Test name: %s", test.Name)
		XCheckGet(t, reg, test.URL, test.Exp)
	}
}

func TestFiltersObjs(t *testing.T) {
	reg := NewRegistry("TestFiltersObjs")
	defer PassDeleteReg(t, reg)

	attr, _ := reg.Model.AddAttrObj("regobj1")

	attr, _ = reg.Model.AddAttrObj("regobj2")
	attr.AddAttr("bool", BOOLEAN)

	attr, _ = reg.Model.AddAttrObj("regobj3")
	attr.AddAttr("bool", BOOLEAN)
	XNoErr(t, reg.SaveModel(true))

	XNoErr(t, reg.SetSave("regobj2", map[string]any{}))
	XNoErr(t, reg.SetSave("regobj3", map[string]any{"bool": true}))

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "regobj1 present - not found",
			URL:  "?inline&filter=regobj1",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "regobj1 present - not found",
			URL:  "?inline&filter=regobj1!=null",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "regobj1 not present",
			URL:  "?oneline&inline&filter=regobj1=null",
			Exp:  `{"regobj2":{},"regobj3":{}}`,
		},

		{
			Name: "regobj2 present - found",
			URL:  "?oneline&inline&filter=regobj2",
			Exp:  `{"regobj2":{},"regobj3":{}}`,
		},
		{
			Name: "regobj2 present - found",
			URL:  "?oneline&inline&filter=regobj2!=null",
			Exp:  `{"regobj2":{},"regobj3":{}}`,
		},
		{
			Name: "regobj2 not present",
			URL:  "?inline&filter=regobj2=null",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "regobj2.bool not present",
			URL:  "?oneline&inline&filter=regobj2.bool=null",
			Exp:  `{"regobj2":{},"regobj3":{}}`,
		},
		{
			Name: "regobj2.bool present",
			URL:  "?inline&filter=regobj2.bool",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "regobj2.bool!=true",
			URL:  "?oneline&inline&filter=regobj2.bool!=true",
			Exp:  `{"regobj2":{},"regobj3":{}}`,
		},
		{
			Name: "regobj2.bool != null present",
			URL:  "?inline&filter=regobj2.bool!=null",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "regobj1.bool != null present",
			URL:  "?inline&filter=regobj1.bool!=null",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "regobj1.bool == null true",
			URL:  "?oneline&inline&filter=regobj1.bool=null",
			Exp:  `{"regobj2":{},"regobj3":{}}`,
		},

		{
			Name: "regobj3.bool == null false",
			URL:  "?inline&filter=regobj3.bool=null",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`,
		},
		{
			Name: "regobj3.bool != null true",
			URL:  "?oneline&inline&filter=regobj3.bool!=null",
			Exp:  `{"regobj2":{},"regobj3":{}}`,
		},
		{
			Name: "regobj3.bool present true",
			URL:  "?oneline&inline&filter=regobj3.bool",
			Exp:  `{"regobj2":{},"regobj3":{}}`,
		},
		{
			Name: "regobj3.bool=true",
			URL:  "?oneline&inline&filter=regobj3.bool=true",
			Exp:  `{"regobj2":{},"regobj3":{}}`,
		},
		{
			Name: "regobj3.bool!=false",
			URL:  "?oneline&inline&filter=regobj3.bool!=false",
			Exp:  `{"regobj2":{},"regobj3":{}}`,
		},
		{
			Name: "regobj3.bool!=null",
			URL:  "?oneline&inline&filter=regobj3.bool!=null",
			Exp:  `{"regobj2":{},"regobj3":{}}`,
		},
	}

	for _, test := range tests {
		t.Logf("Test name: %s", test.Name)
		XCheckGet(t, reg, test.URL, test.Exp)
	}
}

func TestFiltersURLs(t *testing.T) {
	reg := NewRegistry("TestFiltersURLs")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true)
	XNoErr(t, err)
	_, err = gm.AddResourceModel("datas", "data", 0, true, true)
	XNoErr(t, err)
	XNoErr(t, reg.SaveModel(true))

	// establish baseline
	XHTTP(t, reg, "PUT", "/?inline=dirs", `{
        "dirs": {
            "d1": {
                "files": {
                    "f1": {},
                    "f2": {}
                },
                "datas": {
                "d1": {},
                "d2": {}
                }
            },
            "d2": {
                "files": {
                    "f1": {},
                    "f2": {}
                },
                "datas": {
                    "d1": {},
                    "d2": {}
                }
            }
        }
    }`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestFiltersURLs",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:02Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

      "datasurl": "http://localhost:8181/dirs/d1/datas",
      "datascount": 2,
      "filesurl": "http://localhost:8181/dirs/d1/files",
      "filescount": 2
    },
    "d2": {
      "dirid": "d2",
      "self": "http://localhost:8181/dirs/d2",
      "xid": "/dirs/d2",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:02Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

      "datasurl": "http://localhost:8181/dirs/d2/datas",
      "datascount": 2,
      "filesurl": "http://localhost:8181/dirs/d2/files",
      "filescount": 2
    }
  },
  "dirscount": 2
}
`)

	// Now test the URLs have the appropriate subsetted filter expressions
	// Start with AND testing
	XHTTP(t, reg, "GET",
		"/?inline=dirs&filter=dirs.dirid=d2,dirs.datas.dataid=d2",
		``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestFiltersURLs",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2026-05-24T15:56:21.489831698Z",
  "modifiedat": "2026-05-24T15:56:21.510904221Z",

  "dirsurl": "http://localhost:8181/dirs?filter=dirid=d2,datas.dataid=d2",
  "dirs": {
    "d2": {
      "dirid": "d2",
      "self": "http://localhost:8181/dirs/d2",
      "xid": "/dirs/d2",
      "epoch": 1,
      "createdat": "2026-05-24T15:56:21.510904221Z",
      "modifiedat": "2026-05-24T15:56:21.510904221Z",

      "datasurl": "http://localhost:8181/dirs/d2/datas?filter=dataid=d2",
      "datascount": 1,
      "filesurl": "http://localhost:8181/dirs/d2/files",
      "filescount": 0
    }
  },
  "dirscount": 1
}
`)

	XHTTP(t, reg, "GET",
		"/dirs?filter=dirid=d2,datas.dataid=d2",
		``, 200, `{
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "createdat": "2026-05-24T16:00:02.376527682Z",
    "modifiedat": "2026-05-24T16:00:02.376527682Z",

    "datasurl": "http://localhost:8181/dirs/d2/datas?filter=dataid=d2",
    "datascount": 1,
    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  }
}
`)

	XHTTP(t, reg, "GET",
		"/dirs/d2?filter=datas.dataid=d2",
		``, 200, `{
  "dirid": "d2",
  "self": "http://localhost:8181/dirs/d2",
  "xid": "/dirs/d2",
  "epoch": 1,
  "createdat": "2026-05-24T16:01:01.865592732Z",
  "modifiedat": "2026-05-24T16:01:01.865592732Z",

  "datasurl": "http://localhost:8181/dirs/d2/datas?filter=dataid=d2",
  "datascount": 1,
  "filesurl": "http://localhost:8181/dirs/d2/files",
  "filescount": 0
}
`)

	XHTTP(t, reg, "GET",
		"/dirs/d2/datas?filter=dataid=d2",
		``, 200, `{
  "d2": {
    "dataid": "d2",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d2/datas/d2$details",
    "xid": "/dirs/d2/datas/d2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1",

    "metaurl": "http://localhost:8181/dirs/d2/datas/d2/meta",
    "versionsurl": "http://localhost:8181/dirs/d2/datas/d2/versions",
    "versionscount": 1
  }
}
`)

	// Now make sure ORs work
	XHTTP(t, reg, "GET",
		"/?inline=dirs&filter=dirs.files.fileid=f1&filter=dirs.datas.dataid=d2",
		``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestFiltersURLs",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2026-05-24T16:04:17.48194014Z",
  "modifiedat": "2026-05-24T16:04:17.502254683Z",

  "dirsurl": "http://localhost:8181/dirs?filter=files.fileid=f1&filter=datas.dataid=d2",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2026-05-24T16:04:17.502254683Z",
      "modifiedat": "2026-05-24T16:04:17.502254683Z",

      "datasurl": "http://localhost:8181/dirs/d1/datas?filter=dataid=d2",
      "datascount": 1,
      "filesurl": "http://localhost:8181/dirs/d1/files?filter=fileid=f1",
      "filescount": 1
    },
    "d2": {
      "dirid": "d2",
      "self": "http://localhost:8181/dirs/d2",
      "xid": "/dirs/d2",
      "epoch": 1,
      "createdat": "2026-05-24T16:04:17.502254683Z",
      "modifiedat": "2026-05-24T16:04:17.502254683Z",

      "datasurl": "http://localhost:8181/dirs/d2/datas?filter=dataid=d2",
      "datascount": 1,
      "filesurl": "http://localhost:8181/dirs/d2/files?filter=fileid=f1",
      "filescount": 1
    }
  },
  "dirscount": 2
}
`)

	XHTTP(t, reg, "GET",
		"/dirs?filter=files.fileid=f1&filter=datas.dataid=d2",
		``, 200, `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2026-05-24T16:05:18.263946789Z",
    "modifiedat": "2026-05-24T16:05:18.263946789Z",

    "datasurl": "http://localhost:8181/dirs/d1/datas?filter=dataid=d2",
    "datascount": 1,
    "filesurl": "http://localhost:8181/dirs/d1/files?filter=fileid=f1",
    "filescount": 1
  },
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "createdat": "2026-05-24T16:05:18.263946789Z",
    "modifiedat": "2026-05-24T16:05:18.263946789Z",

    "datasurl": "http://localhost:8181/dirs/d2/datas?filter=dataid=d2",
    "datascount": 1,
    "filesurl": "http://localhost:8181/dirs/d2/files?filter=fileid=f1",
    "filescount": 1
  }
}
`)

	// Now test some other variants with filters

	// Notice non-in-doc/local/relative URLs see the filters filters
	XHTTP(t, reg, "GET", "/?doc&filter=dirs.files.fileid=f1",
		``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestFiltersURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2026-05-24T16:16:01.48254591Z",
  "modifiedat": "2026-05-24T16:16:01.502446729Z",

  "dirsurl": "http://localhost:8181/dirs?filter=files.fileid=f1",
  "dirscount": 2
}
`)

	// But local (in doc) URLs (e.g. dirsurl) don't see filters
	XHTTP(t, reg, "GET", "/?doc&inline=dirs&filter=dirs.files.fileid=f1",
		``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestFiltersURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2026-05-24T16:17:02.012652627Z",
  "modifiedat": "2026-05-24T16:17:02.032361913Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2026-05-24T16:17:02.032361913Z",
      "modifiedat": "2026-05-24T16:17:02.032361913Z",

      "datasurl": "http://localhost:8181/dirs/d1/datas",
      "datascount": 0,
      "filesurl": "http://localhost:8181/dirs/d1/files?filter=fileid=f1",
      "filescount": 1
    },
    "d2": {
      "dirid": "d2",
      "self": "#/dirs/d2",
      "xid": "/dirs/d2",
      "epoch": 1,
      "createdat": "2026-05-24T16:17:02.032361913Z",
      "modifiedat": "2026-05-24T16:17:02.032361913Z",

      "datasurl": "http://localhost:8181/dirs/d2/datas",
      "datascount": 0,
      "filesurl": "http://localhost:8181/dirs/d2/files?filter=fileid=f1",
      "filescount": 1
    }
  },
  "dirscount": 2
}
`)

}

func TestFiltersMisc(t *testing.T) {
	reg := NewRegistry("TestFiltersMisc")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true)
	XNoErr(t, err)

	XHTTP(t, reg, "PUT", "/?filter=dirs..dirid=d2", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_filter",
  "title": "An error was found in \"filter\" value (dirs..dirid): Unexpected \".\" in \"dirs..dirid\" at pos 6.",
  "subject": "/",
  "args": {
    "error_detail": "Unexpected \".\" in \"dirs..dirid\" at pos 6",
    "value": "dirs..dirid"
  },
  "source": "b51cea166ad9:registry:info:417"
}
`)
}

func TestFiltersWildcardsInName(t *testing.T) {
	reg := NewRegistry("TestFiltersWildcardsInName")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "PUT", "/", `{
      "ext": {
        "arr": [
          {
            "map": {
              "k1": "v1",
              "k2": "v2"
            }
          }, {
            "obj": {
              "a1": "b1"
            }
          }
        ]
      },
      "modelsource": {
        "attributes": {
          "*": {
            "type": "any"
          }
        },
        "groups": {
          "dirs": {
            "singular": "dir",
            "resources": {
              "files": {
                "singular": "file",
                "hasdocument": false,
                "attributes": {
                  "obj": {
                    "type": "object",
                    "attributes": {
                      "str": { "type": "string" },
                      "*": { "type": "string" }
                    }
                  },
                  "map": {
                    "type": "map",
                    "item": { "type": "string" }
                  },
                  "arr": {
                    "type": "array",
                    "item": { "type": "integer" }
                  },
                  "*": {
                    "type": "any"
                  }
                }
              }
            }
          }
        }
      },
      "dirs": {
        "d1": {
          "files": {
            "f1": {
              "obj": {
                "str": "astr",
                "ext": "anext"
              },
              "map": {
                "key1": "111",
                "key2": "222"
              },
              "arr": [ 1, 2, 3 ],
              "ext": {
                "arr": [
                  {
                    "map": {
                      "k1": "v1",
                      "k2": "v2"
                    }
                  }, {
                    "obj": {
                      "a1": "b1"
                    }
                  }
                ]
              }
            },
            "f2": {}
          }
        }
      }
    }`, 200, "*")

	// First make sure a basic test works
	XHTTP(t, reg, "GET", "/dirs/d1/files?filter=obj.str", "", 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f1",
    "xid": "/dirs/d1/files/f1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2026-07-10T14:46:49.221824428Z",
    "modifiedat": "2026-07-10T14:46:49.221824428Z",
    "ancestorid": "1",
    "arr": [
      1,
      2,
      3
    ],
    "ext": {
      "arr": [
        {
          "map": {
            "k1": "v1",
            "k2": "v2"
          }
        },
        {
          "obj": {
            "a1": "b1"
          }
        }
      ]
    },
    "map": {
      "key1": "111",
      "key2": "222"
    },
    "obj": {
      "ext": "anext",
      "str": "astr"
    },

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 1
  }
}
`)

	// Make sure "*" must stand alone in the path
	for _, errQ := range []string{"*x", "x*", "x*x", "*x*"} {
		XHTTP(t, reg, "GET", "/dirs/d1/files?filter=obj."+errQ+"=astr", "",
			400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_filter",
  "title": "An error was found in \"filter\" value (obj.`+errQ+`): Unexpected \"*\" in \"`+errQ+`\".",
  "subject": "/dirs/d1/files",
  "args": {
    "error_detail": "Unexpected \"*\" in \"`+errQ+`\"",
    "value": "obj.`+errQ+`"
  },
  "source": "6401a2345caa:registry:info:468"
}
`)
	}

	// --------------- OBJECT

	// Test for wildcard as obj attr name
	// Attr exists - Matches more than one attr
	XHTTP(t, reg, "GET", "/dirs/d1?inline&filter=files.obj.*", "", 200,
		`^(?s)^.*"f1".*filescount": 1`)

	// Attr doesn't exists - Matches more than one attr
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.obj.*=null", "", 200,
		`^(?s)^.*"f2".*filescount": 1`)

	// Attr does OR doesn't exist
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.obj.*&filter=files.obj.*=null", "",
		200, `^(?s)^.*"f1".*"f2".*filescount": 2`)

	// Attr does AND doesn't exist - no match, no result
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.obj.*,files.obj.*=null", "",
		404, `*`)

	// Make sure multiple matches in the same entity doesn't mess it up
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.obj.*,files.obj.str=astr", "", 200,
		`^(?s)^.*"f1".*filescount": 1`)

	// --------------- MAPS

	// test Map key - exists
	XHTTP(t, reg, "GET", "/dirs/d1?inline&filter=files.map.*", "", 200,
		`^(?s)^.*"f1".*filescount": 1`)

	// key doesn't exist (empty map)
	XHTTP(t, reg, "GET", "/dirs/d1?inline&filter=files.map.*=null", "", 200,
		`^(?s)^.*"f2".*filescount": 1`)

	// --------------- ARRAYS

	// test Array index - exists
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.arr[*]", "", 200,
		`^(?s)^.*"f1".*filescount": 1`)

	// Bad array reference - maybe we should allow it?? Not sure yet
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.arr.*", "", 404, `*`)

	// index doesn't exist (empty array)
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.arr[*]=null", "", 200,
		`^(?s)^.*"f2".*filescount": 1`)

	// exact index missing
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.arr[0]=null", "", 200,
		`^(?s)^.*"f2".*filescount": 1`)

	// exact index match
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.arr[0]=1", "", 200,
		`^(?s)^.*"f1".*filescount": 1`)

	// exact index mismatch - no file returned
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.arr[0]=2", "", 404, `*`)

	// any index match
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.arr[*]=2", "", 200,
		`^(?s)^.*"f1".*filescount": 1`)

	// any index mismatch
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.arr[*]=4", "", 404, `*`)

	// any index >
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.arr[*]>2", "", 200,
		`^(?s)^.*"f1".*filescount": 1`)

	// any index > mismatch
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.arr[*]>4", "", 404, `*`)

	// last index
	/*
		XHTTP(t, reg, "GET",
			"/dirs/d1?inline&filter=files.arr[-1]", "", 200, ``)
	*/

	// --------------- MISC
	/*
			   "ext": {
			     "arr": [
		          {
			       "map": {
			         "k1": "v1",
			         "k2": "v2"
			       }
		          },
		          {
			       "obj": {
			         "a1": "b1"
			       }
		          }
			     ]
			   }
	*/

	// Go deep!!!
	XHTTP(t, reg, "GET",
		//                            e a m  k
		"/dirs/d1?inline&filter=files.*.*[*].*.*=v1", "", 200,
		`^(?s)^*"f1".*"filescount": 1`)

	// Bad value
	XHTTP(t, reg, "GET",
		"/dirs/d1?inline&filter=files.*.*[*].*.*=v3", "", 404, `*`)

	// Too few .*'s
	XHTTP(t, reg, "GET",
		`/dirs/d1?inline&filter=files.*.*["*x"].*=v2`, "", 404, `*`)

	// Just a couple at root
	XHTTP(t, reg, "GET", "/?filter=*.*[*].*.*=v1", "", 200, `*`)
	XHTTP(t, reg, "GET", "/?filter=*.*[*].*.*=v3", "", 404, `*`)
	XHTTP(t, reg, "GET", "/?filter=*.arr[*].*.*=v2", "", 200, `*`)
	XHTTP(t, reg, "GET", "/?filter=ext.arr[*].*.*=v2", "", 200, `*`)
	XHTTP(t, reg, "GET", "/?filter=*.arr[*].obj.*=b1", "", 200, `*`)
	XHTTP(t, reg, "GET", "/?filter=*.arr[*].obj.a1=b1", "", 200, `*`)
	XHTTP(t, reg, "GET", "/?filter=*.arr[1].obj.a1=b1", "", 200, `*`)
	XHTTP(t, reg, "GET", "/?filter=*.arr[0].obj.a1=b1", "", 404, `*`)
	XHTTP(t, reg, "GET", "/?filter=ext.arr[1].obj.a1=b1", "", 200, `*`)
	XHTTP(t, reg, "GET", "/?filter=ext.arr[2]", "", 404, `*`)
	XHTTP(t, reg, "GET", "/?filter=ext", "", 200, `*`)
	XHTTP(t, reg, "GET", "/?filter=ext.arr[1]", "", 200, `*`)

	XHTTP(t, reg, "GET", "/?filter=ext=null", "", 404, `*`)
	XHTTP(t, reg, "GET", "/?filter=ext.arr[1]=null", "", 404, `*`)

	XHTTP(t, reg, "GET", "/?filter=epoch", "", 200, `*`)
	XHTTP(t, reg, "GET", "/?filter=epoch=null", "", 404, `*`)

	XHTTP(t, reg, "GET", "/?filter=foo", "", 404, `*`)
	XHTTP(t, reg, "GET", "/?filter=foo=null", "", 200, `*`)
}
