package tests

import (
	"strings"
	"testing"

	"github.com/xregistry/server/registry"
)

// checkNoTypeMap makes sure the model (as returned via GET /model) has no
// "typemap" property anywhere in it.
func checkNoTypeMap(t *testing.T, reg *registry.Registry) {
	t.Helper()
	res := XHTTP(t, reg, "GET", "/model", "", 200, "*")
	if strings.Contains(res.body, "typemap") {
		t.Fatalf("Model should not contain a typemap.\nGot:\n%s", res.body)
	}
}

func TestTypeMap(t *testing.T) {
	reg := NewRegistry("TestTypeMap")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "PUT", "/modelsource", MODEL_DIRS, 200, `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file"
        }
      }
    }
  }
}
`)

	// Should be empty initially
	checkNoTypeMap(t, reg)

	XHTTP(t, reg, "PUT", "/modelsource", `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "typemap": { "foo/bar": "json" }
        }
      }
    }
  }
}`, 200, "*")
	XHTTP(t, reg, "GET", "/model", ``, 200,
		"^"+`"typemap":\s*\{\s*"foo/bar":\s*"json"\s*\}`)

	XHTTP(t, reg, "PUT", "/modelsource", MODEL_DIRS, 200, `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file"
        }
      }
    }
  }
}
`)
	checkNoTypeMap(t, reg)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{"contenttype":"bad/bad", "file": "foo"}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "1",
  "contenttype": "bad/bad",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "1",
  "contenttype": "bad/bad",
  "filebase64": "Zm9v",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "typemap": { "bad/bad": "json" }
        }
      }
    }
  }
}`, 200, "*")
	XHTTP(t, reg, "GET", "/model", ``, 200,
		"^"+`"typemap":\s*\{\s*"bad/bad":\s*"json"\s*\}`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "1",
  "contenttype": "bad/bad",
  "file": "foo",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "typemap": { "bad/*": "json" }
        }
      }
    }
  }
}`, 200, "*")
	XHTTP(t, reg, "GET", "/model", ``, 200,
		"^"+`"typemap":\s*\{\s*"bad/\*":\s*"json"\s*\}`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "1",
  "contenttype": "bad/bad",
  "file": "foo",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "typemap": { "bad/*": "json", "bad/b*": "json" }
        }
      }
    }
  }
}`, 200, "*")
	XHTTP(t, reg, "GET", "/model", ``, 200,
		"^"+`"typemap":\s*\{\s*"bad/\*":\s*"json",\s*"bad/b\*":\s*"json"\s*\}`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "1",
  "contenttype": "bad/bad",
  "file": "foo",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "typemap": { "bad/*": "json", "bad/b*": "json", "*/b*": "string" }
        }
      }
    }
  }
}`, 200, "*")
	XHTTP(t, reg, "GET", "/model", ``, 200,
		"^"+`"typemap":\s*\{\s*"\*/b\*":\s*"string",\s*"bad/\*":\s*"json",\s*"bad/b\*":\s*"json"\s*\}`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "1",
  "contenttype": "bad/bad",
  "filebase64": "Zm9v",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "typemap": { "*/b*": "string" }
        }
      }
    }
  }
}`, 200, "*")
	XHTTP(t, reg, "GET", "/model", ``, 200,
		"^"+`"typemap":\s*\{\s*"\*/b\*":\s*"string"\s*\}`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "1",
  "contenttype": "bad/bad",
  "file": "foo",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details",
		`{"file": "{\"foo\":\"bar\"}"}`,
		200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "1",
  "contenttype": "bad/bad",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "1",
  "contenttype": "bad/bad",
  "file": "{\"foo\":\"bar\"}",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "typemap": { "*/b*": "json" }
        }
      }
    }
  }
}`, 200, "*")
	XHTTP(t, reg, "GET", "/model", ``, 200,
		"^"+`"typemap":\s*\{\s*"\*/b\*":\s*"json"\s*\}`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "1",
  "contenttype": "bad/bad",
  "file": {
    "foo": "bar"
  },

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "typemap": { "*/b*": "binary" }
        }
      }
    }
  }
}`, 200, "*")
	XHTTP(t, reg, "GET", "/model", ``, 200,
		"^"+`"typemap":\s*\{\s*"\*/b\*":\s*"binary"\s*\}`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "1",
  "contenttype": "bad/bad",
  "filebase64": "eyJmb28iOiJiYXIifQ==",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details",
		`{"contenttype": null, "file": "foo\"bar"}`,
		200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "1",
  "filebase64": "Zm9vImJhcg==",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Force app/json to binary
	XHTTP(t, reg, "PUT", "/modelsource", `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "typemap": { "*/b*": "binary", "application/json": "binary" }
        }
      }
    }
  }
}`, 200, "*")
	XHTTP(t, reg, "GET", "/model", ``, 200,
		"^"+`"typemap":\s*\{\s*"\*/b\*":\s*"binary",\s*"application/json":\s*"binary"\s*\}`)
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details",
		`{"file": "foo\"bar"}`,
		200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "filebase64": "Zm9vImJhcg==",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "typemap": { "*/b*": "binary" }
        }
      }
    }
  }
}`, 200, "*")
	XHTTP(t, reg, "GET", "/model", ``, 200,
		"^"+`"typemap":\s*\{\s*"\*/b\*":\s*"binary"\s*\}`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "file": "foo\"bar",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

}

func TestTypeMapErrors(t *testing.T) {
	reg := NewRegistry("TestTypeMapErrors")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "PUT", "/modelsource", `{"groups":{"dirs":{"singular":"dir",
  "resources":{"files":{"singular":"file",
  "typemap":{"text/foo":"xml"}}}}}}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Resource \"files\" has an invalid 'typemap' value (xml). Must be one of 'string', 'json' or 'binary'.",
  "subject": "/model",
  "args": {
    "error_detail": "Resource \"files\" has an invalid 'typemap' value (xml). Must be one of 'string', 'json' or 'binary'"
  },
  "instance": "xxx",
  "source": "xxx"
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{"groups":{"dirs":{"singular":"dir",
  "resources":{"files":{"singular":"file",
  "typemap":{"":"json"}}}}}}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Resource \"files\" has an empty 'typemap' key.",
  "subject": "/model",
  "args": {
    "error_detail": "Resource \"files\" has an empty 'typemap' key"
  },
  "instance": "xxx",
  "source": "xxx"
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{"groups":{"dirs":{"singular":"dir",
  "resources":{"files":{"singular":"file",
  "typemap":{"text/foo":""}}}}}}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Resource \"files\" has an invalid 'typemap' value (). Must be one of 'string', 'json' or 'binary'.",
  "subject": "/model",
  "args": {
    "error_detail": "Resource \"files\" has an invalid 'typemap' value (). Must be one of 'string', 'json' or 'binary'"
  },
  "instance": "xxx",
  "source": "xxx"
}
`)

	checkNoTypeMap(t, reg)
}

func TestTypeMapMixedCase(t *testing.T) {
	reg := NewRegistry("TestTypeMapMixedCase")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "PUT", "/modelsource", `{"groups":{"dirs":{
  "singular":"dir","resources":{"files":{"singular":"file",
  "typemap":{"Text/MyJSON":"JSON","TEXT/mystr":"String","Text/MyBin":"BINARY"}}}}}}`,
		200, `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "typemap": {
            "Text/MyJSON": "JSON",
            "TEXT/mystr": "String",
            "Text/MyBin": "BINARY"
          }
        }
      }
    }
  }
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{"contenttype":"tEXt/myjson; charset=utf-8","file":"{\"a\": 1}"}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "1",
  "contenttype": "tEXt/myjson; charset=utf-8",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "1",
  "contenttype": "tEXt/myjson; charset=utf-8",
  "file": {
    "a": 1
  },

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{"contenttype":"text/mystr; charset=utf-8","file":"{\"a\": 1}"}`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "1",
  "contenttype": "text/mystr; charset=utf-8",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "1",
  "contenttype": "text/mystr; charset=utf-8",
  "file": "{\"a\": 1}",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{"contenttype":"TEXT/MYBIN","file":"{\"a\": 1}"}`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:03Z",
  "ancestorid": "1",
  "contenttype": "TEXT/MYBIN",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:03Z",
  "ancestorid": "1",
  "contenttype": "TEXT/MYBIN",
  "filebase64": "eyJhIjogMX0=",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{"contenttype":"application/json; charset=utf-8","file":"{\"a\": 1}"}`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:04Z",
  "ancestorid": "1",
  "contenttype": "application/json; charset=utf-8",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:04Z",
  "ancestorid": "1",
  "contenttype": "application/json; charset=utf-8",
  "file": {
    "a": 1
  },

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{"contenttype":"text/plain","file":"{\"a\": 1}"}`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 5,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:05Z",
  "ancestorid": "1",
  "contenttype": "text/plain",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 5,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:05Z",
  "ancestorid": "1",
  "contenttype": "text/plain",
  "file": "{\"a\": 1}",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{"contenttype":"foo/unknown","file":"{\"a\": 1}"}`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 6,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:06Z",
  "ancestorid": "1",
  "contenttype": "foo/unknown",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 6,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:06Z",
  "ancestorid": "1",
  "contenttype": "foo/unknown",
  "filebase64": "eyJhIjogMX0=",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)
}
