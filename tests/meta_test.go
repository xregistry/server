package tests

import (
	"testing"

	. "github.com/xregistry/server/common"
)

func TestMetaSimple(t *testing.T) {
	reg := NewRegistry("TestMetaSimple")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, false) // noDoc
	rm.AddMetaAttr("foo", ANY)

	// Simple no body create PUT
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: "{}",
		Code:    201,
		ResHeaders: []string{
			"Location: http://localhost:8181/dirs/d1/files/f1/meta",
		},
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`,
	})

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "ancestor": "1"
    }
  },
  "versionscount": 1
}
`)

	// Simple create no body POST - error
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f11/meta",
		Method:  "POST",
		ReqBody: "{}",
		Code:    405,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#action_not_supported",
  "title": "The specified action (POST) is not supported for: /dirs/d1/files/f11/meta.",
  "detail": "POST not allowed on a 'meta'.",
  "subject": "/dirs/d1/files/f11/meta",
  "args": {
    "action": "POST"
  },
  "source": "e4e59b8a76c4:registry:httpStuff:1873"
}
`,
	})

	// Simple create no body PUT, URL too long
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f11/meta/xxx",
		Method: "PUT",
		Code:   404,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f11/meta/xxx) cannot be found.",
  "subject": "/dirs/d1/files/f11/meta/xxx",
  "source": "e4e59b8a76c4:registry:info:661"
}
`,
	})

	// Simple create no body POST, URL too long
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f11/meta/xxx",
		Method: "POST",
		Code:   404,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f11/meta/xxx) cannot be found.",
  "subject": "/dirs/d1/files/f11/meta/xxx",
  "source": "e4e59b8a76c4:registry:info:661"
}
`,
	})

	// Simple create no body PATCH, URL too long
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f11/meta/xxx",
		Method: "PATCH",
		Code:   404,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f11/meta/xxx) cannot be found.",
  "subject": "/dirs/d1/files/f11/meta/xxx",
  "source": "e4e59b8a76c4:registry:info:661"
}
`,
	})

	// Simple create no body PATCH
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f12/meta",
		Method:  "PUT",
		ReqBody: "{}",
		Code:    201,
		ResHeaders: []string{
			"Location: http://localhost:8181/dirs/d1/files/f12/meta",
		},
		ResBody: `{
  "fileid": "f12",
  "self": "http://localhost:8181/dirs/d1/files/f12/meta",
  "xid": "/dirs/d1/files/f12/meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f12/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Simple body create PUT + ext
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f2/meta",
		Method: "PUT",
		ReqBody: `{
  "foo": "bar"
}
`,
		Code: 201,
		ResHeaders: []string{
			"Location: http://localhost:8181/dirs/d1/files/f2/meta",
		},
		ResBody: `{
  "fileid": "f2",
  "self": "http://localhost:8181/dirs/d1/files/f2/meta",
  "xid": "/dirs/d1/files/f2/meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "readonly": false,
  "compatibility": "none",
  "foo": "bar",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Simple body create PATCH + ext
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f21/meta",
		Method: "PUT",
		ReqBody: `{
  "foo": "bar"
}
`,
		Code: 201,
		ResHeaders: []string{
			"Location: http://localhost:8181/dirs/d1/files/f21/meta",
		},
		ResBody: `{
  "fileid": "f21",
  "self": "http://localhost:8181/dirs/d1/files/f21/meta",
  "xid": "/dirs/d1/files/f21/meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "readonly": false,
  "compatibility": "none",
  "foo": "bar",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f21/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Update PUT no body - erases ext
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f2/meta",
		Method:  "PUT",
		ReqBody: "{}",
		Code:    200,
		ResHeaders: []string{
			"-Location",
		},
		ResBody: `{
  "fileid": "f2",
  "self": "http://localhost:8181/dirs/d1/files/f2/meta",
  "xid": "/dirs/d1/files/f2/meta",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1",
  "defaultversionsticky": false
}
`,
	})

	XHTTP(t, reg, "GET", "/dirs/d1/files/f2?inline", ``, 200, `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2",
  "xid": "/dirs/d1/files/f2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "meta": {
    "fileid": "f2",
    "self": "http://localhost:8181/dirs/d1/files/f2/meta",
    "xid": "/dirs/d1/files/f2/meta",
    "epoch": 2,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versions": {
    "1": {
      "fileid": "f2",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f2/versions/1",
      "xid": "/dirs/d1/files/f2/versions/1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "ancestor": "1"
    }
  },
  "versionscount": 1
}
`)

	// Update PUT empty body
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{}`,
		Code:    200,
		ResHeaders: []string{
			"-Location",
		},
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Update PATCH empty body
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f21/meta",
		Method:  "PATCH",
		ReqBody: `{}`,
		Code:    200,
		ResHeaders: []string{
			"-Location",
		},
		ResBody: `{
  "fileid": "f21",
  "self": "http://localhost:8181/dirs/d1/files/f21/meta",
  "xid": "/dirs/d1/files/f21/meta",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",
  "foo": "bar",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f21/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Update PUT + ext
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{ "foo": "zzz"}`,
		Code:    200,
		ResHeaders: []string{
			"-Location",
		},
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",
  "foo": "zzz",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Update PATCH empty body
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f21/meta",
		Method:  "PATCH",
		ReqBody: `{"foo":"aaa"}`,
		Code:    200,
		ResHeaders: []string{
			"-Location",
		},
		ResBody: `{
  "fileid": "f21",
  "self": "http://localhost:8181/dirs/d1/files/f21/meta",
  "xid": "/dirs/d1/files/f21/meta",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",
  "foo": "aaa",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f21/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Update PUT + bad ext
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{ "fff": "zzz"}`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (fff) was specified for \"/dirs/d1/files/f1/meta\".",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "name": "fff"
  },
  "source": "186f71c5fb29:registry:entity:2192"
}
`,
	})

	// Update PATCH + bad ext
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f21/meta",
		Method:  "PATCH",
		ReqBody: `{"fff":"aaa"}`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (fff) was specified for \"/dirs/d1/files/f21/meta\".",
  "subject": "/dirs/d1/files/f21/meta",
  "args": {
    "name": "fff"
  },
  "source": "186f71c5fb29:registry:entity:2192"
}
`,
	})

	// Update PUT, del ext
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{}`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Update PATCH, del ext
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f21/meta",
		Method:  "PATCH",
		ReqBody: `{"foo":null}`,
		Code:    200,
		ResBody: `{
  "fileid": "f21",
  "self": "http://localhost:8181/dirs/d1/files/f21/meta",
  "xid": "/dirs/d1/files/f21/meta",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f21/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Update PUT, add ext again
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{ "foo": "zz1"}`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 5,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",
  "foo": "zz1",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Update PUT, del ext again
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{"foo":null}`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 6,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Make sure DELETE fails
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1/meta",
		Method: "DELETE",
		Code:   405,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#action_not_supported",
  "title": "The specified action (DELETE) is not supported for: /dirs/d1/files/f1/meta.",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "action": "DELETE"
  },
  "source": "e4e59b8a76c4:registry:httpStuff:2754"
}
`,
	})

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f9/meta", `{
      "meta": {},
      "metaurl": "http://localhost:8181/dirs/d1/files/f9/meta",
      "versions": { "v1": {}},
      "versionsurl": "http://localhost:8181/dirs/d1/files/f0/versions",
      "versionscount": 1
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (meta) was specified for \"/dirs/d1/files/f9/meta\".",
  "detail": "Full list: meta,metaurl,versions,versionscount,versionsurl.",
  "subject": "/dirs/d1/files/f9/meta",
  "args": {
    "name": "meta"
  },
  "source": "65b92b8c0e3b:registry:entity:2201"
}
`)

}

func TestMetaCombos(t *testing.T) {
	reg := NewRegistry("TestMetaCombos")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, false) // noDoc
	rm.AddMetaAttr("foo", ANY)

	// Create Resource and set the versionID
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "PUT",
		ReqBody: `{"versionid":"v1.0"}`,
		Code:    201,
		ResHeaders: []string{
			"Location: http://localhost:8181/dirs/d1/files/f1",
		},
		ResBody: `{
  "fileid": "f1",
  "versionid": "v1.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	// Verify it's all correct
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline", ``, 200, `{
  "fileid": "f1",
  "versionid": "v1.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v1.0",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1.0",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1.0": {
      "fileid": "f1",
      "versionid": "v1.0",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1.0",
      "xid": "/dirs/d1/files/f1/versions/v1.0",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "ancestor": "v1.0"
    }
  },
  "versionscount": 1
}
`)

	// PUT again with wrong versionid should fail
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "PUT",
		ReqBody: `{"versionid":"v2.0"}`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"versionid\" value (v2.0) for \"/dirs/d1/files/f1\" needs to be \"v1.0\".",
  "detail": "Must match the \"defaultversionid\" value.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "expected_id": "v1.0",
    "invalid_id": "v2.0",
    "singular": "version"
  },
  "source": "e4e59b8a76c4:registry:group:466"
}
`,
	})

	// PUT again with wrong fileid should fail
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "PUT",
		ReqBody: `{"fileid":"foo"}`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"fileid\" value (foo) for \"/dirs/d1/files/f1\" needs to be \"f1\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "expected_id": "f1",
    "invalid_id": "foo",
    "singular": "file"
  },
  "source": "e4e59b8a76c4:registry:httpStuff:2166"
}
`,
	})

	// PUT on meta with wrong fileid should fail
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{"fileid":"foo"}`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"fileid\" value (foo) for \"/dirs/d1/files/f1/meta\" needs to be \"f1\".",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "expected_id": "f1",
    "invalid_id": "foo",
    "singular": "file"
  },
  "source": "e4e59b8a76c4:registry:resource:489"
}
`,
	})

	// Create a version, setting vid
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "POST",
		ReqBody: `{"versionid":"v2.0"}`,
		Code:    201,
		ResHeaders: []string{
			"Location: http://localhost:8181/dirs/d1/files/f1/versions/v2.0",
		},
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2.0",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2.0",
  "xid": "/dirs/d1/files/f1/versions/v2.0",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1.0"
}
`,
	})

	// Verify
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline", ``, 200, `{
  "fileid": "f1",
  "versionid": "v2.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:02Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2.0",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2.0",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1.0": {
      "fileid": "f1",
      "versionid": "v1.0",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1.0",
      "xid": "/dirs/d1/files/f1/versions/v1.0",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "ancestor": "v1.0"
    },
    "v2.0": {
      "fileid": "f1",
      "versionid": "v2.0",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2.0",
      "xid": "/dirs/d1/files/f1/versions/v2.0",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",
      "ancestor": "v1.0"
    }
  },
  "versionscount": 2
}
`)

	// Update/PUT w/o body should just bump epoch/modifiedat
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "PUT",
		ReqBody: "{}",
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`,
	})

	// Make sure resource's epoch didn't change
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?inline=meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2.0",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2.0",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`,
	})

	// Update/PUT - valid vid
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "PUT",
		ReqBody: `{"versionid": "v2.0"}`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`,
	})

	// Make sure just version's epoch/timestamp changed
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?inline=meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2.0",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2.0",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`,
	})

	// Update/PUT - make defaultversionid sticky
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "PUT",
		ReqBody: `{"meta":{"defaultversionsticky":true}}`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`,
	})

	// defversticky changed, but so did the default version's epoch/timestamp
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?inline=meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 3,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2.0",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2.0",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`,
	})

	// Make sure just version's epoch/timestamp changed
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?inline=meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 3,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2.0",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2.0",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`,
	})

	// Update/PUT - def attrs at the wrong spot
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "PUT",
		ReqBody: `{"defaultversionid": "v1.0","defaultversionsticky":true}`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (defaultversionid) was specified for \"/dirs/d1/files/f1/versions/v2.0\".",
  "detail": "Full list: defaultversionid,defaultversionsticky.",
  "subject": "/dirs/d1/files/f1/versions/v2.0",
  "args": {
    "name": "defaultversionid"
  },
  "source": "186f71c5fb29:registry:entity:2192"
}
`,
	})

	// Update/PUT - stick it again via meta this time
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{"defaultversionsticky":true}`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v2.0",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2.0",
  "defaultversionsticky": true
}
`,
	})

	// meta's epoch changed but the defver didn't
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?inline=meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 4,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:04Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2.0",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2.0",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`,
	})

	// Create new version, defverid should not change, nor meta epoch/ts.
	// New vid should be generated - ie '1'
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "POST",
		ReqBody: "{}",
		Code:    201,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "xid": "/dirs/d1/files/f1/versions/1",
  "epoch": 1,
  "isdefault": false,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v2.0"
}
`,
	})

	// defver and meta should be unchanged
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?inline=meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 5,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:04Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2.0",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2.0",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`,
	})

	// Update/PUT - unstick it, '1' should be the def now
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{"defaultversionsticky":false}`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 6,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// meta's epoch changed but the defver didn't
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?inline=meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v2.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 6,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`,
	})

	// Update/PUT - stick it via 'defverid' AND sticky
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{"defaultversionid":"v1.0","defaultversionsticky":true}`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 7,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v1.0",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1.0",
  "defaultversionsticky": true
}
`,
	})

	// meta's epoch changed but the defver didn't
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?inline=meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v1.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 7,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v1.0",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1.0",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`,
	})

	// Update/PUT - unstick
	// Include extension
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{ "defaultversionsticky":null, "foo":"bar" }`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 8,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",
  "foo": "bar",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// meta's epoch changed but the defver didn't
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?inline=meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v2.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 8,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",
    "foo": "bar",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`,
	})

	// Update/PATCH - change defverid+sticky.
	// Ext should remain
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PATCH",
		ReqBody: `{"defaultversionid":"v1.0","defaultversionsticky":true}`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 9,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",
  "foo": "bar",

  "defaultversionid": "v1.0",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1.0",
  "defaultversionsticky": true
}
`,
	})

	// meta's epoch changed but the defver didn't
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?inline=meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v1.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 9,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",
    "foo": "bar",

    "defaultversionid": "v1.0",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1.0",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`,
	})

	// Update/PATCH - unstick
	// Ext should remain
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PATCH",
		ReqBody: `{"defaultversionsticky":null}`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 10,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",
  "foo": "bar",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Update/PATCH - stick
	// Ext should remain
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PATCH",
		ReqBody: `{"defaultversionsticky":true}`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 11,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",
  "foo": "bar",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": true
}
`,
	})

	// Update/PUT - empty - should erase ext and unstick it
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: "{}",
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 12,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`,
	})

	// Update/PUT meta - bad epoch
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{"epoch": 1}`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (1) for \"/dirs/d1/files/f1/meta\" does not match its current value (12).",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "bad_epoch": "1",
    "epoch": "12"
  },
  "source": "e4e59b8a76c4:registry:entity:1004"
}
`,
	})

	// Update/PUT resource - bad epoch
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "PUT",
		ReqBody: `{"meta":{"epoch": 1}}`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (1) for \"/dirs/d1/files/f1/meta\" does not match its current value (12).",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "bad_epoch": "1",
    "epoch": "12"
  },
  "source": "e4e59b8a76c4:registry:entity:1004"
}
`,
	})

}
