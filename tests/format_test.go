package tests

import (
	// log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
	"testing"
)

// Tests "format" and "compatibility" and meta.* as http headers

func TestFormatSimple(t *testing.T) {
	reg := NewRegistry("TestFormatSimple")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rm, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)

	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "), 200, `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true
        }
      }
    }
  }
}
`)

	// make sure that if validatecompat=true then validateformat must be true
	rm.SetValidateCompatibility(true)
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Resource \"files\" must have \"validateformat\" set to \"true\" when \"validatecompatibility\" is \"true\".",
  "subject": "/model",
  "args": {
    "error_detail": "Resource \"files\" must have \"validateformat\" set to \"true\" when \"validatecompatibility\" is \"true\""
  },
  "source": "c30ebf8b495a:registry:shared_model:2335"
}
`)

	rm.ClearValidateCompatibility() // clear it to test just format

	rm.SetValidateFormat(true) // And enable validateformat
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "validateformat": true
        }
      }
    }
  }
}
`)

	// Happy path
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "format": "numbers",
  "file": "1\n2\n3"
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-03-06T00:19:13.099947785Z",
  "modifiedat": "2026-03-06T00:19:13.099947785Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "numbers",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Clear validateformat and make sure all is still ok
	rm.ClearValidateFormat()
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true
        }
      }
    }
  }
}
`)

	// Make the resource invalid per the 'format'. Should not error
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2", `not a number`, 201,
		`not a number`)

	// Now try to turn on format validation, should fail on f2
	rm.SetValidateFormat(true)
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_missing",
  "title": "Version \"/dirs/d1/files/f2/versions/1\" needs to have a \"format\" value due to its owning Resource model's \"validateformat\" being set.",
  "subject": "/dirs/d1/files/f2/versions/1",
  "source": "c30ebf8b495a:registry:resource:1708"
}
`)

	// give it a format, but a bad one. No checks so should be ok
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f2",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-format: bad-format",
		},
		ReqBody: "not a number",
		Code:    200,
		ResHeaders: []string{
			"access-control-allow-origin: *",
			"access-control-allow-methods: GET, PATCH, POST, PUT, DELETE",
			"content-disposition: f2",
			"content-location: http://localhost:8181/dirs/d1/files/f2/versions/1",
			"content-type: text/plain; charset=utf-8",
			"xRegistry-fileid: f2",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f2",
			"xRegistry-xid: /dirs/d1/files/f2",
			"xRegistry-epoch: 2",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.236399049Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.291099909Z",
			"xRegistry-ancestor: 1",
			"xRegistry-format: bad-format",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f2/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f2/versions",
			"xRegistry-versionscount: 1",
		},
		ResBody: `not a number`})

	// Try to turn on validateformat again, should still fail due to bad format
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Unknown \"format\" value for /dirs/d1/files/f2: bad-format.",
  "subject": "/dirs/d1/files/f2",
  "args": {
    "error_detail": "Unknown \"format\" value for /dirs/d1/files/f2: bad-format"
  },
  "source": "c30ebf8b495a:registry:resource:1713"
}
`)

	// Now, update good format, but bad data for that format
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f2",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-format: numbers",
		},
		ReqBody: "not a number",
		Code:    200,
		ResHeaders: []string{
			"access-control-allow-origin: *",
			"access-control-allow-methods: GET, PATCH, POST, PUT, DELETE",
			"content-disposition: f2",
			"content-location: http://localhost:8181/dirs/d1/files/f2/versions/1",
			"content-type: text/plain; charset=utf-8",
			"xRegistry-fileid: f2",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f2",
			"xRegistry-xid: /dirs/d1/files/f2",
			"xRegistry-epoch: 3",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.236399049Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.291099909Z",
			"xRegistry-ancestor: 1",
			"xRegistry-format: numbers",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f2/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f2/versions",
			"xRegistry-versionscount: 1",
		},
		ResBody: `not a number`})

	// Try to turn on validateformat again, should still fail due to bad data
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f2/versions/1\" to be non-compliant with its \"format\" (numbers).",
  "detail": "Line 1 isn't an integer: not a number.",
  "subject": "/dirs/d1/files/f2/versions/1",
  "args": {
    "format": "numbers"
  },
  "source": "c30ebf8b495a:registry:format_numbers:36"
}
`)

	// now give it good data
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f2",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-format: numbers",
		},
		ReqBody: "1\n2\n3",
		Code:    200,
		ResHeaders: []string{
			"access-control-allow-origin: *",
			"access-control-allow-methods: GET, PATCH, POST, PUT, DELETE",
			"content-disposition: f2",
			"content-location: http://localhost:8181/dirs/d1/files/f2/versions/1",
			"content-type: text/plain; charset=utf-8",
			"xRegistry-fileid: f2",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f2",
			"xRegistry-xid: /dirs/d1/files/f2",
			"xRegistry-epoch: 4",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.236399049Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.291099909Z",
			"xRegistry-ancestor: 1",
			"xRegistry-format: numbers",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f2/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f2/versions",
			"xRegistry-versionscount: 1",
		},
		ResBody: "1\n2\n3"})

	// Try to turn on validateformat again, should work this time
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "validateformat": true
        }
      }
    }
  }
}
`)

	// Creating a resource w/o a format should fail when validateformat=true
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f3", "1",
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_missing",
  "title": "Version \"/dirs/d1/files/f3/versions/1\" needs to have a \"format\" value due to its owning Resource model's \"validateformat\" being set.",
  "subject": "/dirs/d1/files/f3/versions/1",
  "source": "c30ebf8b495a:registry:resource:1710"
}
`)

	// This one should work since it has a 'format'
	// Case insensitive 'format'
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-format: NuMbErS",
		},
		ReqBody: "3",
		Code:    201,
		ResHeaders: []string{
			"access-control-allow-origin: *",
			"access-control-allow-methods: GET, PATCH, POST, PUT, DELETE",
			"content-disposition: f3",
			"content-location: http://localhost:8181/dirs/d1/files/f3/versions/1",
			"content-type: text/plain; charset=utf-8",
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f3",
			"xRegistry-xid: /dirs/d1/files/f3",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.0Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.0Z",
			"xRegistry-ancestor: 1",
			"xRegistry-format: NuMbErS",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f3/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f3/versions",
			"xRegistry-versionscount: 1",
		},
		ResBody: `3`})
}

func TestFormatCompatSimple(t *testing.T) {
	reg := NewRegistry("TestFormatCompatSimple")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rm, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)

	rm.SetValidateCompatibility(true)

	// Should fail since validateformat isn't set
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Resource \"files\" must have \"validateformat\" set to \"true\" when \"validatecompatibility\" is \"true\".",
  "subject": "/model",
  "args": {
    "error_detail": "Resource \"files\" must have \"validateformat\" set to \"true\" when \"validatecompatibility\" is \"true\""
  },
  "source": "c30ebf8b495a:registry:shared_model:2342"
}
`)

	rm.SetValidateFormat(true)

	// Should work this time
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "validatecompatibility": true,
          "validateformat": true
        }
      }
    }
  }
}
`)

	// Try to turn off validateformat w/o turning off validatecompat will fail
	rm.SetValidateFormat(false)

	// Should work this time
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Resource \"files\" must have \"validateformat\" set to \"true\" when \"validatecompatibility\" is \"true\".",
  "subject": "/model",
  "args": {
    "error_detail": "Resource \"files\" must have \"validateformat\" set to \"true\" when \"validatecompatibility\" is \"true\""
  },
  "source": "c30ebf8b495a:registry:shared_model:2342"
}
`)

	// But turning off both should be ok tho
	rm.ClearValidateCompatibility()
	rm.SetValidateFormat(false)

	// Should work this time
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "validateformat": false
        }
      }
    }
  }
}
`)

	rm.SetValidateCompatibility(true)
	rm.SetValidateFormat(true)

	// Now turn both back on so we can test compat
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "validatecompatibility": true,
          "validateformat": true
        }
      }
    }
  }
}
`)

	// Now let's create some Resources/files

	// Create file w/o format - should fail
	XCheckHTTP(t, reg, &HTTPTest{
		URL:        "/dirs/d1/files/f1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "not a number",
		Code:       400,
		ResHeaders: []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_missing",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" needs to have a \"format\" value due to its owning Resource model's \"validateformat\" being set.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "source": "c30ebf8b495a:registry:resource:1711"
}
`})

	// Now with 'format' - weird casing
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-format: nUmBeRs",
		},
		ReqBody: "1",
		Code:    201,
		ResHeaders: []string{
			"access-control-allow-origin: *",
			"access-control-allow-methods: GET, PATCH, POST, PUT, DELETE",
			"content-disposition: f1",
			"content-location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"content-type: text/plain; charset=utf-8",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.0Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.0Z",
			"xRegistry-ancestor: 1",
			"xRegistry-format: nUmBeRs",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount: 1",
		},
		ResBody: `1`,
	})

	// Turn on compat with bad value (empty string)
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-meta.compatibility: ",
		},
		ReqBody:    "2",
		Code:       400,
		ResHeaders: []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"compatibility\" for \"/dirs/d1/files/f1/meta\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "compatibility"
  },
  "source": "c30ebf8b495a:registry:resource:1616"
}
`,
	})

	// Turn on compat with bad value (unknown)
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-meta.compatibility: unknown",
		},
		ReqBody:    "2",
		Code:       400,
		ResHeaders: []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Unknown \"compatibility\" value for /dirs/d1/files/f1/meta: unknown.",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "error_detail": "Unknown \"compatibility\" value for /dirs/d1/files/f1/meta: unknown"
  },
  "source": "c30ebf8b495a:registry:resource:1842"
}
`,
	})

	// Turn on compat with good value, weird casing
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-meta.compatibility: BaCkWaRd",
		},
		ReqBody: "2",
		Code:    200,
		ResHeaders: []string{
			"access-control-allow-origin: *",
			"access-control-allow-methods: GET, PATCH, POST, PUT, DELETE",
			"content-disposition: f1",
			"content-location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"content-type: text/plain; charset=utf-8",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch: 2",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.0Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.1Z",
			"xRegistry-ancestor: 1",
			"xRegistry-format: nUmBeRs",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount: 1",
		},
		ResBody: `2`,
	})

	// Add a new version w/o format
	XCheckHTTP(t, reg, &HTTPTest{
		URL:        "/dirs/d1/files/f1",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    "2",
		Code:       400,
		ResHeaders: []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_missing",
  "title": "Version \"/dirs/d1/files/f1/versions/2\" needs to have a \"format\" value due to its owning Resource model's \"validateformat\" being set.",
  "subject": "/dirs/d1/files/f1/versions/2",
  "source": "c30ebf8b495a:registry:resource:1711"
}
`,
	})

	// Add a new version w/ bad format
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-format: unknown",
		},
		ReqBody:    "2",
		Code:       400,
		ResHeaders: []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Unknown \"format\" value for /dirs/d1/files/f1: unknown.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "error_detail": "Unknown \"format\" value for /dirs/d1/files/f1: unknown"
  },
  "source": "c30ebf8b495a:registry:resource:1716"
}
`,
	})

	// Try again with good "format" this time
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-format: NUMBers",
		},
		ReqBody: "2",
		Code:    201,
		ResHeaders: []string{
			"access-control-allow-origin: *",
			"access-control-allow-methods: GET, PATCH, POST, PUT, DELETE",
			"content-disposition: f1",
			"content-location: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"content-type: text/plain; charset=utf-8",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 2",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"xRegistry-xid: /dirs/d1/files/f1/versions/2",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.0Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.0Z",
			"xRegistry-ancestor: 1",
			"xRegistry-format: NUMBers",
		},
		ResBody: `2`,
	})

	// update that version with bad format
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-format: unknown",
		},
		ReqBody: "2",
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Unknown \"format\" value for /dirs/d1/files/f1: unknown.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "error_detail": "Unknown \"format\" value for /dirs/d1/files/f1: unknown"
  },
  "source": "c30ebf8b495a:registry:resource:1716"
}
`,
	})

	// update that version with bad doc
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "PUT",
		ReqBody: "text",
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/2\" to be non-compliant with its \"format\" (numbers).",
  "detail": "Line 1 isn't an integer: text.",
  "subject": "/dirs/d1/files/f1/versions/2",
  "args": {
    "format": "numbers"
  },
  "source": "c30ebf8b495a:registry:format_numbers:36"
}
`,
	})

	// update that version with bad doc - not backward compat
	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "PUT",
		ReqBody: "0", // needs to be >= 2
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#compatibility_violation",
  "title": "The request would cause one or more Versions of \"/dirs/d1/files/f1\" to violate its compatibility rule (BaCkWaRd).",
  "detail": "Version \"/dirs/d1/files/f1/versions/2\" (sum: 0) isn't \"BaCkWaRd\" compatible with \"/dirs/d1/files/f1/versions/1\" (sum: 2).",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "value": "BaCkWaRd"
  },
  "source": "c30ebf8b495a:registry:format_numbers:82"
}
`,
	})
}

func TestFormatCompatVariants(t *testing.T) {
	reg := NewRegistry("TestFormatCompatVariants")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rm, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)

	rm.SetValidateCompatibility(true)
	rm.SetValidateFormat(true)

	// Should fail since validateformat isn't set
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `*`)

	type Test struct {
		Name   string
		Method string
		Path   string
		Body   string
		Err    string
	}

	// Missing Format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", "123", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_missing",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" needs to have a \"format\" value due to its owning Resource model's \"validateformat\" being set.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "source": "c30ebf8b495a:registry:resource:1711"
}
`)

	// Bad Format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
    "format": "Unknown",
    "file":  "123"}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Unknown \"format\" value for /dirs/d1/files/f1: Unknown.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "error_detail": "Unknown \"format\" value for /dirs/d1/files/f1: Unknown"
  },
  "source": "c30ebf8b495a:registry:resource:1716"
}
`)

	// Weird but legal format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", `{
    "format": "NuMBers",
    "file":  "1"}`, 201, `*`)

	// Create valid v2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2/versions/v2$details", `{
    "meta": { "compatibility": "backward"},
    "format":"NuMBers",
    "file":  "2"}`, 201, `*`)

	// V3 isn't compatible
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2/versions/v3$details", `{
    "format":"numbers",
    "file":  "0"}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#compatibility_violation",
  "title": "The request would cause one or more Versions of \"/dirs/d1/files/f2\" to violate its compatibility rule (backward).",
  "detail": "Version \"/dirs/d1/files/f2/versions/v3\" (sum: 0) isn't \"backward\" compatible with \"/dirs/d1/files/f2/versions/v2\" (sum: 2).",
  "subject": "/dirs/d1/files/f2",
  "args": {
    "value": "backward"
  },
  "source": "c30ebf8b495a:registry:format_numbers:82"
}
`)

	// Now V3 is compatible
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2/versions/v3$details", `{
    "format":"numbers",
    "file":  "3"}`, 201, `*`)

	// Change v2 to break compat with bad file
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2/versions/v2", `4`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#compatibility_violation",
  "title": "The request would cause one or more Versions of \"/dirs/d1/files/f2\" to violate its compatibility rule (backward).",
  "detail": "Version \"/dirs/d1/files/f2/versions/v3\" (sum: 3) isn't \"backward\" compatible with \"/dirs/d1/files/f2/versions/v2\" (sum: 4).",
  "subject": "/dirs/d1/files/f2",
  "args": {
    "value": "backward"
  },
  "source": "c30ebf8b495a:registry:format_numbers:82"
}
`)

	// Change v2 to break compat with missing format
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2/versions/v2$details", `{
        "format": null
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_missing",
  "title": "Version \"/dirs/d1/files/f2/versions/v2\" needs to have a \"format\" value due to its owning Resource model's \"validateformat\" being set.",
  "subject": "/dirs/d1/files/f2/versions/v2",
  "source": "c30ebf8b495a:registry:resource:1712"
}
`)

	// Change v2 to break compat with empty format
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2/versions/v2$details", `{
        "format": ""
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_missing",
  "title": "Version \"/dirs/d1/files/f2/versions/v2\" needs to have a \"format\" value due to its owning Resource model's \"validateformat\" being set.",
  "subject": "/dirs/d1/files/f2/versions/v2",
  "source": "c30ebf8b495a:registry:resource:1712"
}
`)

	// Change v2 to break compat with bad format
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2/versions/v2$details", `{
        "format": "UnKnown"
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Unknown \"format\" value for /dirs/d1/files/f2: UnKnown.",
  "subject": "/dirs/d1/files/f2",
  "args": {
    "error_detail": "Unknown \"format\" value for /dirs/d1/files/f2: UnKnown"
  },
  "source": "c30ebf8b495a:registry:resource:1717"
}
`)

	// Change v2 to break compat with bad format
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2/versions/v2$details", `{
        "format": "protobuf"
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "/dirs/d1/files/f2/versions/v2is not a valid protobuf file:schema.proto:1:1: syntax error: unexpected int literal.",
  "subject": "/dirs/d1/files/f2/versions/v2",
  "args": {
    "error_detail": "/dirs/d1/files/f2/versions/v2is not a valid protobuf file:schema.proto:1:1: syntax error: unexpected int literal"
  },
  "source": "c30ebf8b495a:registry:format_proto:42"
}
`)

}
