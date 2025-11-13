package tests

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	. "github.com/xregistry/server/common"
)

func TestInlineBasic(t *testing.T) {
	reg := NewRegistry("TestInlineBasic")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	d, _ = reg.AddGroup("dirs", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")

	gm2, _ := reg.Model.AddGroupModel("dirs2", "dir2")
	gm2.AddResourceModel("files", "file", 0, true, true, true)
	d2, _ := reg.AddGroup("dirs2", "d2")
	d2.AddResource("files", "f2", "v1")

	// /dirs/d1/files/f1/v1
	//                  /v2
	//      /d2/files/f2/v1
	//                  /v1.1
	// /dirs2/d2/files/f2/v1

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "No Inline",
			URL:  "?",
			Exp: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestInlineBasic",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 2,
  "dirs2url": "http://localhost:8181/dirs2",
  "dirs2count": 1
}
`,
		},
		{
			Name: "Inline - No Filter - full",
			URL:  "?inline",
			Exp: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestInlineBasic",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v2",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "v1",
          "filebase64": "",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v2",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 1,
              "isdefault": false,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "v1",
              "filebase64": ""
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
              "xid": "/dirs/d1/files/f1/versions/v2",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "v1",
              "filebase64": ""
            }
          },
          "versionscount": 2
        }
      },
      "filescount": 1
    },
    "d2": {
      "dirid": "d2",
      "self": "http://localhost:8181/dirs/d2",
      "xid": "/dirs/d2",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d2/files",
      "files": {
        "f2": {
          "fileid": "f2",
          "versionid": "v1.1",
          "self": "http://localhost:8181/dirs/d2/files/f2$details",
          "xid": "/dirs/d2/files/f2",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "v1",
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
            "compatibility": "none",

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
              "ancestor": "v1",
              "filebase64": ""
            },
            "v1.1": {
              "fileid": "f2",
              "versionid": "v1.1",
              "self": "http://localhost:8181/dirs/d2/files/f2/versions/v1.1$details",
              "xid": "/dirs/d2/files/f2/versions/v1.1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "v1",
              "filebase64": ""
            }
          },
          "versionscount": 2
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 2,
  "dirs2url": "http://localhost:8181/dirs2",
  "dirs2": {
    "d2": {
      "dir2id": "d2",
      "self": "http://localhost:8181/dirs2/d2",
      "xid": "/dirs2/d2",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs2/d2/files",
      "files": {
        "f2": {
          "fileid": "f2",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs2/d2/files/f2$details",
          "xid": "/dirs2/d2/files/f2",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "v1",
          "filebase64": "",

          "metaurl": "http://localhost:8181/dirs2/d2/files/f2/meta",
          "meta": {
            "fileid": "f2",
            "self": "http://localhost:8181/dirs2/d2/files/f2/meta",
            "xid": "/dirs2/d2/files/f2/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v1",
            "defaultversionurl": "http://localhost:8181/dirs2/d2/files/f2/versions/v1$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs2/d2/files/f2/versions",
          "versions": {
            "v1": {
              "fileid": "f2",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs2/d2/files/f2/versions/v1$details",
              "xid": "/dirs2/d2/files/f2/versions/v1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "v1",
              "filebase64": ""
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1
    }
  },
  "dirs2count": 1
}
`,
		},
		{
			Name: "Inline - No Filter",
			URL:  "?inline&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{}}}}}}}`,
		},
		{
			Name: "Inline * - * Filter",
			URL:  "?inline=*&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{}}}}}}}`,
		},
		{
			Name: "Inline * - * Filter - not first",
			URL:  "?inline=dirs2,*&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{}}}}}}}`,
		},
		{
			Name: "inline one level",
			URL:  "?inline=dirs&oneline",
			Exp:  `{"dirs":{"d1":{},"d2":{}}}`,
		},
		{
			Name: "inline one level - invalid",
			URL:  "?inline=xxx",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: xxx"
}
`,
		},
		{
			Name: "inline one level - invalid - bad case",
			URL:  "?inline=Dirs",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: Dirs"
}
`,
		},
		{
			Name: "inline two levels - invalid first",
			URL:  "?inline=xxx.files",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: xxx.files"
}
`,
		},
		{
			Name: "inline two levels - invalid second",
			URL:  "?inline=dirs.xxx",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.xxx"
}
`,
		},
		{
			Name: "inline two levels",
			URL:  "?inline=dirs.files&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{}}},"d2":{"files":{"f2":{}}}}}`,
		},
		{
			Name: "inline three levels",
			URL:  "?inline=dirs.files.versions&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "get one level, inline one level - invalid",
			URL:  "dirs?inline=dirs",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs"
}
`,
		},
		{
			Name: "get one level, inline one level",
			URL:  "dirs?inline=files&oneline",
			Exp:  `{"d1":{"files":{"f1":{}}},"d2":{"files":{"f2":{}}}}`,
		},
		{
			Name: "get one level, inline two levels",
			URL:  "dirs?inline=files.versions&oneline",
			Exp:  `{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}`,
		},
		{
			Name: "get one level, inline three levels",
			URL:  "dirs?inline=files.versions.xxx",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs",
  "title": "The request cannot be processed as provided: invalid 'inline' value: files.versions.xxx"
}
`,
		},
		{
			Name: "get one level, inline one level",
			URL:  "dirs/d1?inline=files&oneline",
			Exp:  `{"files":{"f1":{}}}`,
		},
		{
			Name: "get one level, inline two levels",
			URL:  "dirs/d1?inline=files.versions&oneline",
			Exp:  `{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}`,
		},
		{
			Name: "get one level, inline all",
			URL:  "dirs/d1?inline=&oneline",
			Exp:  `{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}}`,
		},

		{
			Name: "inline 2 top levels",
			URL:  "?inline=dirs,dirs2&oneline",
			Exp:  `{"dirs":{"d1":{},"d2":{}},"dirs2":{"d2":{}}}`,
		},
		{
			Name: "inline 2 top, 1 and 2 levels",
			URL:  "?inline=dirs,dirs2.files&oneline",
			Exp:  `{"dirs":{"d1":{},"d2":{}},"dirs2":{"d2":{"files":{"f2":{}}}}}`,
		},
		{
			Name: "inline 2 top, 1 and 2 levels - one err",
			URL:  "?inline=dirs,dirs2.files.xxx",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs2.files.xxx"
}
`,
		},
		{
			Name: "get one level, inline 2, 1 and 2 levels same top",
			URL:  "dirs?inline=files,files.meta,files.versions&oneline",
			Exp:  `{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}`,
		},

		{
			Name: "get one level, inline all",
			URL:  "dirs?inline&oneline",
			Exp:  `{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}`,
		},
		{
			Name: "get one level/res, inline all",
			URL:  "dirs/d2?inline&oneline",
			Exp:  `{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}`,
		},
	}

	for _, test := range tests {
		t.Logf("Testing: %s", test.Name)
		xCheckGet(t, reg, test.URL, test.Exp)
	}
}

func TestInlineResource(t *testing.T) {
	reg := NewRegistry("TestInlineResource")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	d, _ := reg.AddGroup("dirs", "d1")

	// ProxyURL
	f, _ := d.AddResource("files", "f1-proxy", "v1")
	f.SetSaveDefault(NewPP().P("file").UI(), "Hello world! v1")

	v, _ := f.AddVersion("v2")
	v.SetSave(NewPP().P("fileurl").UI(), "http://localhost:8282/EMPTY-URL")

	v, _ = f.AddVersion("v3")
	v.SetSave(NewPP().P("fileproxyurl").UI(), "http://localhost:8282/EMPTY-Proxy")

	// URL
	f, _ = d.AddResource("files", "f2-url", "v1")
	f.SetSaveDefault(NewPP().P("file").UI(), "Hello world! v1")

	v, _ = f.AddVersion("v2")
	v.SetSave(NewPP().P("fileproxyurl").UI(), "http://localhost:8282/EMPTY-Proxy")

	v, _ = f.AddVersion("v3")
	v.SetSave(NewPP().P("fileurl").UI(), "http://localhost:8282/EMPTY-URL")

	// Resource
	f, _ = d.AddResource("files", "f3-resource", "v1")
	f.SetSaveDefault(NewPP().P("fileproxyurl").UI(), "http://localhost:8282/EMPTY-Proxy")

	v, _ = f.AddVersion("v2")
	v.SetSave(NewPP().P("fileurl").UI(), "http://localhost:8282/EMPTY-URL")

	v, _ = f.AddVersion("v3")
	xNoErr(t, v.SetSave(NewPP().P("file").UI(), "Hello world! v3"))

	// /dirs/d1/files/f1-proxy/v1 - resource
	//                        /v2 - URL
	//                        /v3 - ProxyURL  <- default
	// /dirs/d1/files/f2-url/v1 - resource
	//                      /v2 - ProxyURL
	//                      /v3 - URL  <- default
	// /dirs/d1/files/f3-resource/v1 - ProxyURL
	//                           /v2 - URL
	//                           /v3 - resource  <- default

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "No Inline",
			URL:  "?",
			Exp: `{
  "dirscount": 1
}
`,
		},
		{
			Name: "Inline - No Filter - full",
			URL:  "?inline",
			Exp: `{
  "dirs": {
    "d1": {
      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1-proxy": {
          "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
          "filebase64": "aGVsbG8tUHJveHkK",
          "meta": {
            "defaultversionid": "v3",
          },
          "versions": {
            "v1": {
              "filebase64": "SGVsbG8gd29ybGQhIHYx"
            },
            "v2": {
              "fileurl": "http://localhost:8282/EMPTY-URL"
            },
            "v3": {
              "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
              "filebase64": "aGVsbG8tUHJveHkK"
            }
          },
          "versionscount": 3
        },
        "f2-url": {
          "fileurl": "http://localhost:8282/EMPTY-URL",
          "meta": {
            "defaultversionid": "v3",
          },
          "versions": {
            "v1": {
              "filebase64": "SGVsbG8gd29ybGQhIHYx"
            },
            "v2": {
              "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
              "filebase64": "aGVsbG8tUHJveHkK"
            },
            "v3": {
              "fileurl": "http://localhost:8282/EMPTY-URL"
            }
          },
          "versionscount": 3
        },
        "f3-resource": {
          "filebase64": "SGVsbG8gd29ybGQhIHYz",
          "meta": {
            "defaultversionid": "v3",
          },
          "versions": {
            "v1": {
              "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
              "filebase64": "aGVsbG8tUHJveHkK"
            },
            "v2": {
              "fileurl": "http://localhost:8282/EMPTY-URL"
            },
            "v3": {
              "filebase64": "SGVsbG8gd29ybGQhIHYz"
            }
          },
          "versionscount": 3
        }
      },
      "filescount": 3
    }
  },
  "dirscount": 1
}
`,
		},
		{
			Name: "Inline - filter + inline file,meta",
			URL:  "?filter=dirs.files.fileid=f1-proxy&inline=dirs.files.meta,dirs.files.file",
			Exp: `{
  "dirs": {
    "d1": {
      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1-proxy": {
          "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
          "filebase64": "aGVsbG8tUHJveHkK",
          "meta": {
            "defaultversionid": "v3",
          },
          "versionscount": 3
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
			Name: "Inline - filter + inline vers.file,meta",
			URL:  "?filter=dirs.files.fileid=f1-proxy&inline=dirs.files.meta,dirs.files.versions.file",
			Exp: `{
  "dirs": {
    "d1": {
      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1-proxy": {
          "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
          "meta": {
            "defaultversionid": "v3",
          },
          "versions": {
            "v1": {
              "filebase64": "SGVsbG8gd29ybGQhIHYx"
            },
            "v2": {
              "fileurl": "http://localhost:8282/EMPTY-URL"
            },
            "v3": {
              "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
              "filebase64": "aGVsbG8tUHJveHkK"
            }
          },
          "versionscount": 3
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
			Name: "file-proxy",
			URL:  "/dirs/d1/files/f1-proxy",
			Exp:  "hello-Proxy\n",
		},
		{
			Name: "file-url",
			URL:  "/dirs/d1/files/f2-url",
			Exp:  "hello-URL\n",
		},
		{
			Name: "file-resource",
			URL:  "/dirs/d1/files/f3-resource",
			Exp:  `Hello world! v3`,
		},
		{
			Name: "Inline - at file + inline file,meta",
			URL:  "/dirs/d1/files/f1-proxy$details?inline=file,meta",
			Exp: `{
  "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
  "filebase64": "aGVsbG8tUHJveHkK",
  "meta": {
    "defaultversionid": "v3",
  },
  "versionscount": 3
}
`,
		},
		{
			Name: "Inline - at file + inline versions.file,meta",
			URL:  "/dirs/d1/files/f1-proxy$details?inline=versions.file,meta",
			Exp: `{
  "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
  "meta": {
    "defaultversionid": "v3",
  },
  "versions": {
    "v1": {
      "filebase64": "SGVsbG8gd29ybGQhIHYx"
    },
    "v2": {
      "fileurl": "http://localhost:8282/EMPTY-URL"
    },
    "v3": {
      "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
      "filebase64": "aGVsbG8tUHJveHkK"
    }
  },
  "versionscount": 3
}
`,
		},
		{
			Name: "Bad inline xx",
			URL:  "/dirs/d1/files/f1-proxy$details?inline=XXversions.file",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1-proxy$details",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.XXversions.file"
}
`,
		},
		{
			Name: "Bad inline yy",
			URL:  "/?inline=dirs.files.yy",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.yy"
}
`,
		},
		{
			Name: "Bad inline vers.yy",
			URL:  "/?inline=dirs.files.version.yy",
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.version.yy"
}
`,
		},
	}

	// Save everythign to the DB
	xNoErr(t, reg.SaveAllAndCommit())

	for _, test := range tests {
		t.Logf("Testing: %s", test.Name)

		remove := []string{
			`"specversion"`,
			`"registryid"`,
			`"dirid"`,
			`"fileid"`,
			`"versionid"`,
			`"epoch"`,
			`"self"`,
			`"xid"`,
			`"isdefault"`,
			`"metaurl"`,
			`"readonly"`,
			`"compatibility"`,
			`"defaultversionurl"`,
			`"defaultversionsticky"`,
			`"createdat"`,
			`"modifiedat"`,
			`"ancestor"`,
			`"versionsurl"`,
			`"dirsurl"`,
		}

		res, err := http.Get("http://localhost:8181/" + test.URL)
		xCheck(t, err == nil, "%s", err)

		body, _ := io.ReadAll(res.Body)

		for _, str := range remove {
			str = fmt.Sprintf(`(?m)^ *%s.*$\n`, str)
			re := regexp.MustCompile(str)
			body = re.ReplaceAll(body, []byte{})
		}
		body = regexp.MustCompile(`(?m)^ *$\n`).ReplaceAll(body, []byte{})

		xCheckEqual(t, "Test: "+test.Name+"\n", string(body), test.Exp)
	}
}

func TestInlineWildcards(t *testing.T) {
	reg := NewRegistry("TestInlineWildcards")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details",
		`{"file": { "hello": "world"}}`, 201, `*`)

	xHTTP(t, reg, "GET", "?inline=*", ``,
		200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineWildcards",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2025-01-01T12:00:02Z",
          "modifiedat": "2025-01-01T12:00:02Z",
          "ancestor": "v1",
          "contenttype": "application/json",
          "file": {
            "hello": "world"
          },

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:02Z",
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
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:02Z",
              "modifiedat": "2025-01-01T12:00:02Z",
              "ancestor": "v1",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "?inline=dirs.*", ``,
		200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineWildcards",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2025-01-01T12:00:02Z",
          "modifiedat": "2025-01-01T12:00:02Z",
          "ancestor": "v1",
          "contenttype": "application/json",
          "file": {
            "hello": "world"
          },

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:02Z",
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
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:02Z",
              "modifiedat": "2025-01-01T12:00:02Z",
              "ancestor": "v1",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "?inline=dirs.files.*", ``,
		200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineWildcards",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2025-01-01T12:00:02Z",
          "modifiedat": "2025-01-01T12:00:02Z",
          "ancestor": "v1",
          "contenttype": "application/json",
          "file": {
            "hello": "world"
          },

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:02Z",
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
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:02Z",
              "modifiedat": "2025-01-01T12:00:02Z",
              "ancestor": "v1",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "?inline=dirs.files.versions.*", ``,
		200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineWildcards",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2025-01-01T12:00:02Z",
          "modifiedat": "2025-01-01T12:00:02Z",
          "ancestor": "v1",
          "contenttype": "application/json",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:02Z",
              "modifiedat": "2025-01-01T12:00:02Z",
              "ancestor": "v1",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "dirs/?inline=files.versions.*", ``,
		200, `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:02Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "files": {
      "f1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/f1$details",
        "xid": "/dirs/d1/files/f1",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2025-01-01T12:00:02Z",
        "modifiedat": "2025-01-01T12:00:02Z",
        "ancestor": "v1",
        "contenttype": "application/json",

        "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
        "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
        "versions": {
          "v1": {
            "fileid": "f1",
            "versionid": "v1",
            "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
            "xid": "/dirs/d1/files/f1/versions/v1",
            "epoch": 1,
            "isdefault": true,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:02Z",
            "ancestor": "v1",
            "contenttype": "application/json",
            "file": {
              "hello": "world"
            }
          }
        },
        "versionscount": 1
      }
    },
    "filescount": 1
  }
}
`)

	xHTTP(t, reg, "GET", "dirs/?inline=files.*", ``,
		200, `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:02Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "files": {
      "f1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/f1$details",
        "xid": "/dirs/d1/files/f1",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2025-01-01T12:00:02Z",
        "modifiedat": "2025-01-01T12:00:02Z",
        "ancestor": "v1",
        "contenttype": "application/json",
        "file": {
          "hello": "world"
        },

        "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
        "meta": {
          "fileid": "f1",
          "self": "http://localhost:8181/dirs/d1/files/f1/meta",
          "xid": "/dirs/d1/files/f1/meta",
          "epoch": 1,
          "createdat": "2025-01-01T12:00:02Z",
          "modifiedat": "2025-01-01T12:00:02Z",
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
            "epoch": 1,
            "isdefault": true,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:02Z",
            "ancestor": "v1",
            "contenttype": "application/json",
            "file": {
              "hello": "world"
            }
          }
        },
        "versionscount": 1
      }
    },
    "filescount": 1
  }
}
`)

	xHTTP(t, reg, "GET", "dirs/d1?inline=files.versions.*", ``,
		200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "files": {
    "f1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1$details",
      "xid": "/dirs/d1/files/f1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",
      "ancestor": "v1",
      "contenttype": "application/json",

      "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
      "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
      "versions": {
        "v1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
          "xid": "/dirs/d1/files/f1/versions/v1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2025-01-01T12:00:02Z",
          "modifiedat": "2025-01-01T12:00:02Z",
          "ancestor": "v1",
          "contenttype": "application/json",
          "file": {
            "hello": "world"
          }
        }
      },
      "versionscount": 1
    }
  },
  "filescount": 1
}
`)

	xHTTP(t, reg, "GET", "dirs/d1?inline=files.*", ``,
		200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "files": {
    "f1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1$details",
      "xid": "/dirs/d1/files/f1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",
      "ancestor": "v1",
      "contenttype": "application/json",
      "file": {
        "hello": "world"
      },

      "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
      "meta": {
        "fileid": "f1",
        "self": "http://localhost:8181/dirs/d1/files/f1/meta",
        "xid": "/dirs/d1/files/f1/meta",
        "epoch": 1,
        "createdat": "2025-01-01T12:00:02Z",
        "modifiedat": "2025-01-01T12:00:02Z",
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
          "epoch": 1,
          "isdefault": true,
          "createdat": "2025-01-01T12:00:02Z",
          "modifiedat": "2025-01-01T12:00:02Z",
          "ancestor": "v1",
          "contenttype": "application/json",
          "file": {
            "hello": "world"
          }
        }
      },
      "versionscount": 1
    }
  },
  "filescount": 1
}
`)

	xHTTP(t, reg, "GET", "?inline=.*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: Unexpected . in \".*\" at pos 1"
}
`)
	xHTTP(t, reg, "GET", "?inline=foo.*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: foo.*"
}
`)
	xHTTP(t, reg, "GET", "?inline=foo*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: foo*"
}
`)

	xHTTP(t, reg, "GET", "?inline=dirs.bad*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.bad*"
}
`)
	xHTTP(t, reg, "GET", "?inline=dirs.bad.*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.bad.*"
}
`)

	xHTTP(t, reg, "GET", "?inline=dirs.files.bad*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.bad*"
}
`)
	xHTTP(t, reg, "GET", "?inline=dirs.files.bad.*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.bad.*"
}
`)
	xHTTP(t, reg, "GET", "?inline=dirs.files.file*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.file*"
}
`)
	xHTTP(t, reg, "GET", "?inline=dirs.files.file.*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.file.*"
}
`)

	xHTTP(t, reg, "GET", "?inline=dirs.files.meta*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.meta*"
}
`)
	xHTTP(t, reg, "GET", "?inline=dirs.files.meta.*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.meta.*"
}
`)

	xHTTP(t, reg, "GET", "?inline=dirs.files.versions.bad*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.versions.bad*"
}
`)
	xHTTP(t, reg, "GET", "?inline=dirs.files.versions.file*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.versions.file*"
}
`)
	xHTTP(t, reg, "GET", "?inline=dirs.files.versions.file.*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.versions.file.*"
}
`)
	xHTTP(t, reg, "GET", "?inline=dirs.files.versions.file.bad*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: dirs.files.versions.file.bad*"
}
`)

	xHTTP(t, reg, "GET", "?inline=model.*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: model.*"
}
`)
	xHTTP(t, reg, "GET", "?inline=model.bad*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: model.bad*"
}
`)
	xHTTP(t, reg, "GET", "?inline=capabilities.*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: capabilities.*"
}
`)
	xHTTP(t, reg, "GET", "?inline=capabilities.bad*", ``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/",
  "title": "The request cannot be processed as provided: invalid 'inline' value: capabilities.bad*"
}
`)

}

func TestInlineEmpty(t *testing.T) {
	reg := NewRegistry("TestInlineEmpty")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	xHTTP(t, reg, "GET", "/?inline", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineEmpty",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {},
  "dirscount": 0
}
`)

	xHTTP(t, reg, "GET", "/?inline=dirs", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineEmpty",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {},
  "dirscount": 0
}
`)

	xHTTP(t, reg, "GET", "/?inline=dirs.files", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineEmpty",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {},
  "dirscount": 0
}
`)

	xHTTP(t, reg, "GET", "/?inline=dirs.files.versions", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineEmpty",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {},
  "dirscount": 0
}
`)

	xHTTP(t, reg, "GET", "/?inline=dirs.files.versions,dirs.files.meta", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineEmpty",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {},
  "dirscount": 0
}
`)

	reg.AddGroup("dirs", "d1")

	xHTTP(t, reg, "GET", "/?inline", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineEmpty",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {},
      "filescount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?inline=dirs", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineEmpty",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "filescount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?inline=dirs.files", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineEmpty",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {},
      "filescount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?inline=dirs.files.versions", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineEmpty",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {},
      "filescount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?inline=dirs.files.versions,dirs.files.meta", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestInlineEmpty",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {},
      "filescount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs?inline=files.versions,files.meta", "", 200, `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:02Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "files": {},
    "filescount": 0
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1?inline=files.versions,files.meta", "", 200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "files": {},
  "filescount": 0
}
`)

}
