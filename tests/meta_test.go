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
  "subject": "http://localhost:8181/dirs/d1/files/f11/meta",
  "title": "The specified action (POST) is not supported",
  "detail": "POST not allowed on a 'meta'"
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
  "subject": "http://localhost:8181/dirs/d1/files/f11/meta/xxx",
  "title": "The specified entity cannot be found: /dirs/d1/files/f11/meta/xxx"
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
  "subject": "http://localhost:8181/dirs/d1/files/f11/meta/xxx",
  "title": "The specified entity cannot be found: /dirs/d1/files/f11/meta/xxx"
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
  "subject": "http://localhost:8181/dirs/d1/files/f11/meta/xxx",
  "title": "The specified entity cannot be found: /dirs/d1/files/f11/meta/xxx"
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
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The request cannot be processed as provided: invalid extension(s): fff"
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
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f21/meta",
  "title": "The request cannot be processed as provided: invalid extension(s): fff"
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
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The specified action (DELETE) is not supported"
}
`,
	})

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
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The request cannot be processed as provided: when \"versionid\"(v2.0) is present it must match the \"defaultversionid\"(v1.0)"
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
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The attribute \"fileid\" is not valid: must be set to \"f1\", not \"foo\""
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
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The specified \"file\" ID value (foo) needs to be \"f1\""
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
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1/versions/v2.0",
  "title": "The request cannot be processed as provided: invalid extension(s): defaultversionid,defaultversionsticky"
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

	// Update/PUT - update it via 'defverid' - should err since not sticky
	// except bump epoch
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/meta",
		Method:  "PUT",
		ReqBody: `{"defaultversionid":"v1.0"}`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The request cannot be processed as provided: attribute \"defaultversionid\" must be \"1\" since \"defaultversionsticky\" is \"false\""
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

	// Update/PUT - change defverid/unstick - error
	// Include extension
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1/meta",
		Method: "PUT",
		ReqBody: `{
		  "defaultversionid":"v2.0",
		  "defaultversionsticky":null,
		  "foo":"bar"}`,
		Code: 400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The request cannot be processed as provided: attribute \"defaultversionid\" must be \"1\" since \"defaultversionsticky\" is \"false\""
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
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The attribute \"epoch\" is not valid: value (1) doesn't match existing value (12)"
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
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The attribute \"epoch\" is not valid: value (1) doesn't match existing value (12)"
}
`,
	})

}
