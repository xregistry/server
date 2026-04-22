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
          "hasdocument": true,
          "singleversionroot": false,
          "validateformat": false,
          "validatecompatibility": false,
          "strictvalidation": false,
          "consistentformat": false
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
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": false,
          "strictvalidation": false,
          "consistentformat": false
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
  "formatvalidated": true,

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
          "hasdocument": true,
          "singleversionroot": false,
          "validateformat": false,
          "validatecompatibility": false,
          "strictvalidation": false,
          "consistentformat": false
        }
      }
    }
  }
}
`)

	// Make the resource invalid per the 'format'. Should not error
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2", `not a number`, 201,
		`not a number`)

	// Now try to turn on format validation+strict, should skip f2
	rm.SetValidateFormat(true)
	rm.SetStrictValidation(true)
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
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": false,
          "strictvalidation": true,
          "consistentformat": false
        }
      }
    }
  }
}
`)

	// Try again w/o strict, should work this time. Missing is ok
	// Strict=false allows for
	rm.SetStrictValidation(false)
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
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": false,
          "strictvalidation": false,
          "consistentformat": false
        }
      }
    }
  }
}
`)

	// give it a format, but a bad one. strict=false so should be ok
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
			"xRegistry-formatvalidated: false",
			"xRegistry-formatvalidatedreason: Unknown format",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f2/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f2/versions",
			"xRegistry-versionscount: 1",
		},
		ResBody: `not a number`})

	// Try to turn on validateformat again, should still fail due to bad format
	rm.SetStrictValidation(true)
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_unknown",
  "title": "Version \"/dirs/d1/files/f2/versions/1\" has a \"format\" value (bad-format) that it not supported.",
  "subject": "/dirs/d1/files/f2/versions/1",
  "args": {
    "format": "bad-format"
  },
  "source": "79ab0198e6b4:registry:resource:1795"
}
`)

	// Now, no validation, update good format, but bad data for that format
	rm.SetValidateFormat(false)
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
          "singleversionroot": false,
          "validateformat": false,
          "validatecompatibility": false,
          "strictvalidation": true,
          "consistentformat": false
        }
      }
    }
  }
}
`)

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
	rm.SetValidateFormat(true)
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
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": false,
          "strictvalidation": true,
          "consistentformat": false
        }
      }
    }
  }
}
`)

	// Creating a resource w/o a format should work validateformat=true, skips
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f3.1", "1",
		201, `1`)

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
			"xRegistry-formatvalidated: true",
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
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "consistentformat": false
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
          "singleversionroot": false,
          "validateformat": false,
          "validatecompatibility": false,
          "strictvalidation": false,
          "consistentformat": false
        }
      }
    }
  }
}
`)

	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)
	rm.SetStrictValidation(true)

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
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true,
          "consistentformat": false
        }
      }
    }
  }
}
`)

	// Now let's create some Resources/files

	/*
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
	*/

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
			"xRegistry-formatvalidated: true",
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
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#compatibility_unknown",
  "title": "The compatibility value (unknown) on Resource \"/dirs/d1/files/f1/meta\" is not supported for format \"numbers\".",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "compat": "unknown",
    "format": "numbers"
  },
  "source": "a3d56ce41e09:registry:resource:1854"
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
			"xRegistry-formatvalidated: true",
			"xRegistry-compatibilityvalidated: true",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount: 1",
		},
		ResBody: `2`,
	})

	// Add a new version w/o format
	/*
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
	*/

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
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_unknown",
  "title": "Version \"/dirs/d1/files/f1/versions/2\" has a \"format\" value (unknown) that it not supported.",
  "subject": "/dirs/d1/files/f1/versions/2",
  "args": {
    "format": "unknown"
  },
  "source": "79ab0198e6b4:registry:resource:1795"
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
			"xRegistry-formatvalidated: true",
			"xRegistry-compatibilityvalidated: true",
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
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_unknown",
  "title": "Version \"/dirs/d1/files/f1/versions/2\" has a \"format\" value (unknown) that it not supported.",
  "subject": "/dirs/d1/files/f1/versions/2",
  "args": {
    "format": "unknown"
  },
  "source": "79ab0198e6b4:registry:resource:1795"
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
    "compat": "BaCkWaRd"
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

	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)
	rm.SetStrictValidation(true)

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
	/*
	   	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", "123", 400, `{
	     "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_missing",
	     "title": "Version \"/dirs/d1/files/f1/versions/1\" needs to have a \"format\" value due to its owning Resource model's \"validateformat\" being set.",
	     "subject": "/dirs/d1/files/f1/versions/1",
	     "source": "c30ebf8b495a:registry:resource:1711"
	   }
	   `)
	*/

	// Bad Format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
    "format": "Unknown",
    "file":  "123"}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_unknown",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" has a \"format\" value (Unknown) that it not supported.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "Unknown"
  },
  "source": "79ab0198e6b4:registry:resource:1795"
}
`)

	// Weird but legal format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", `{
    "format": "NuMBers",
    "file":  "1"}`, 201, `*`)

	// Create valid v2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
    "versionid": "v2",
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
    "compat": "backward"
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
    "compat": "backward"
  },
  "source": "c30ebf8b495a:registry:format_numbers:82"
}
`)

	// Change v2 to break compat with missing format
	/*
	   	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2/versions/v2$details", `{
	           "format": null
	       }`, 400, `{
	     "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_missing",
	     "title": "Version \"/dirs/d1/files/f2/versions/v2\" needs to have a \"format\" value due to its owning Resource model's \"validateformat\" being set.",
	     "subject": "/dirs/d1/files/f2/versions/v2",
	     "source": "c30ebf8b495a:registry:resource:1712"
	   }
	   `)
	*/

	// Change v2 to break compat with empty format
	/*
	   	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2/versions/v2$details", `{
	           "format": ""
	       }`, 400, `{
	     "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_missing",
	     "title": "Version \"/dirs/d1/files/f2/versions/v2\" needs to have a \"format\" value due to its owning Resource model's \"validateformat\" being set.",
	     "subject": "/dirs/d1/files/f2/versions/v2",
	     "source": "c30ebf8b495a:registry:resource:1712"
	   }
	   `)
	*/

	// Change v2 to break compat with bad format
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2/versions/v2$details", `{
        "format": "UnKnown"
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_unknown",
  "title": "Version \"/dirs/d1/files/f2/versions/v2\" has a \"format\" value (UnKnown) that it not supported.",
  "subject": "/dirs/d1/files/f2/versions/v2",
  "args": {
    "format": "UnKnown"
  },
  "source": "79ab0198e6b4:registry:resource:1795"
}
`)

	// Change v2 to break compat with bad format
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2/versions/v2$details", `{
        "format": "protobuf"
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "/dirs/d1/files/f2/versions/v2 is not a valid protobuf file: schema.proto:1:1: syntax error: unexpected int literal.",
  "subject": "/dirs/d1/files/f2/versions/v2",
  "args": {
    "error_detail": "/dirs/d1/files/f2/versions/v2 is not a valid protobuf file: schema.proto:1:1: syntax error: unexpected int literal"
  },
  "source": "c30ebf8b495a:registry:format_proto:42"
}
`)

}

func TestFormatSimpleJson(t *testing.T) {
	reg := NewRegistry("TestFormatSimpleJson")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rm, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)

	rm.SetValidateFormat(true)   // And enable validateformat
	rm.SetStrictValidation(true) // Don't allow unknown formats

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
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": false,
          "strictvalidation": true,
          "consistentformat": false
        }
      }
    }
  }
}
`)

	// Happy path
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "format": "jsonSchema/draft-07",
  "file": "{}"
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
  "format": "jsonSchema/draft-07",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Happy path - tweak format
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", `{
  "format": "jsonSchema/draft-08"
}`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-03-06T00:19:13.099947785Z",
  "modifiedat": "2026-03-06T00:19:13.199947785Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "jsonSchema/draft-08",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// close but not quite the right format
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", `{
  "format": "jsonSchem"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_unknown",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" has a \"format\" value (jsonSchem) that it not supported.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "jsonSchem"
  },
  "source": "79ab0198e6b4:registry:resource:1795"
}
`)

}

func TestFormatStrictNumbers(t *testing.T) {
	reg := NewRegistry("TestFormatStrictNumbers")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rmFile, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)
	rmNoFile, xErr := gm.AddResourceModel("nofiles", "nofile", 0, true, true, false)
	XNoErr(t, xErr)

	rmFile.SetValidateFormat(true)
	rmFile.SetValidateCompatibility(true)
	rmFile.SetStrictValidation(true)
	rmFile.SetConsistentFormat(true)
	rmNoFile.SetValidateFormat(true)
	rmNoFile.SetValidateCompatibility(true)
	rmNoFile.SetStrictValidation(true)
	rmNoFile.SetConsistentFormat(true)

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
          "hasdocument": true,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true,
          "consistentformat": true
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": false,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true,
          "consistentformat": true
        }
      }
    }
  }
}
`)

	// hasdoc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "meta": {"compatibility": "backward" },
        "format": "numbers",
        "file": "1" }`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:07.554485814Z",
  "modifiedat": "2026-04-15T11:16:07.554485814Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/nofiles/f1", `{
        "format": "numbers"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/nofiles/f1/versions/1\" to be non-compliant with its \"format\" (numbers).",
  "detail": "The Resource (/dirs/d1/nofiles/f1) for Version \"/dirs/d1/nofiles/f1/versions/1\" does not have \"hasdocument\" in its resource model set to \"true\", and an empty/missing document is not compliant.",
  "subject": "/dirs/d1/nofiles/f1/versions/1",
  "args": {
    "format": "numbers"
  },
  "source": "79ab0198e6b4:registry:format_numbers:36"
}
`)

	// no doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "numbers",
        "file": null
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (numbers).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid numbers file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "numbers"
  },
  "source": "79ab0198e6b4:registry:format_numbers:60"
}
`)

	// empty doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "numbers",
        "file": ""
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (numbers).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid numbers file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "numbers"
  },
  "source": "79ab0198e6b4:registry:format_numbers:60"
}
`)

	// missing format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "file": "1"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:34.008113923Z",
  "modifiedat": "2026-04-15T11:16:34.135061948Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// unknown format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "unknown",
        "file": "1"
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_unknown",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" has a \"format\" value (unknown) that it not supported.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "unknown"
  },
  "source": "79ab0198e6b4:registry:resource:1802"
}
`)

	// varying format - 1
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "nuMBers",
      "file": "1"
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"nuMBers\" vs \"jsonSchema\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": null,
      "file": "1"
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"\" vs \"jsonSchema\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 3
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "",
      "file": "1"
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"format\" for \"/dirs/d1/files/f2/versions/v1\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "format"
  },
  "source": "79ab0198e6b4:registry:entity:1446"
}
`)

	// RESOURCEurl
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "numbers",
        "fileurl": "http://example.com"
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_external",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" references a document stored outside of the Registry, therefore no validation was performed.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "source": "79ab0198e6b4:registry:format_numbers:46"
}
`)

}

func TestFormatNotStrictNumbers(t *testing.T) {
	reg := NewRegistry("TestFormatNotStrictNumbers")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rmFile, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)
	rmNoFile, xErr := gm.AddResourceModel("nofiles", "nofile", 0, true, true, false)
	XNoErr(t, xErr)

	rmFile.SetValidateFormat(true)
	rmFile.SetValidateCompatibility(true)
	rmFile.SetStrictValidation(false)
	rmFile.SetConsistentFormat(true)
	rmNoFile.SetValidateFormat(true)
	rmNoFile.SetValidateCompatibility(true)
	rmNoFile.SetStrictValidation(false)
	rmNoFile.SetConsistentFormat(true)

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
          "hasdocument": true,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "consistentformat": true
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": false,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "consistentformat": true
        }
      }
    }
  }
}
`)

	// hasdoc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "meta": {"compatibility": "backward" },
        "format": "numbers",
        "file": "1" }`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:07.554485814Z",
  "modifiedat": "2026-04-15T11:16:07.554485814Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/nofiles/f1", `{
        "format": "numbers"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/nofiles/f1/versions/1\" to be non-compliant with its \"format\" (numbers).",
  "detail": "The Resource (/dirs/d1/nofiles/f1) for Version \"/dirs/d1/nofiles/f1/versions/1\" does not have \"hasdocument\" in its resource model set to \"true\", and an empty/missing document is not compliant.",
  "subject": "/dirs/d1/nofiles/f1/versions/1",
  "args": {
    "format": "numbers"
  },
  "source": "79ab0198e6b4:registry:format_numbers:36"
}
`)

	// no doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "numbers",
        "file": null
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (numbers).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid numbers file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "numbers"
  },
  "source": "79ab0198e6b4:registry:format_numbers:60"
}
`)

	// empty doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "numbers",
        "file": ""
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (numbers).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid numbers file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "numbers"
  },
  "source": "79ab0198e6b4:registry:format_numbers:60"
}
`)

	// missing format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "file": "1"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:34.008113923Z",
  "modifiedat": "2026-04-15T11:16:34.135061948Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// unknown format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "unknown",
        "file": "1"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2026-04-15T17:08:33.325493075Z",
  "modifiedat": "2026-04-15T17:08:33.500548614Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "unknown",
  "formatvalidated": false,
  "formatvalidatedreason": "Unknown format",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Unknown format",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// varying format - 1
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "nuMBers",
      "file": "1"
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"nuMBers\" vs \"jsonSchema\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": null,
      "file": "1"
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"\" vs \"jsonSchema\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 3
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "",
      "file": "1"
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"format\" for \"/dirs/d1/files/f2/versions/v1\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "format"
  },
  "source": "79ab0198e6b4:registry:entity:1446"
}
`)

	// varying format - 4
	rmFile.SetConsistentFormat(false)
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "), 200, "*")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "meta": { "compatibility": "backWARD" },
   "versions": {
    "v1": {
      "format": "nuMBers",
      "file": "1"
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Version \"/dirs/d1/files/f2/versions/v1\" has a \"format\" value of \"nuMBers\", was expecting \"jsonschema*\".",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "Version \"/dirs/d1/files/f2/versions/v1\" has a \"format\" value of \"nuMBers\", was expecting \"jsonschema*\""
  },
  "source": "79ab0198e6b4:registry:format_jsonschema:137"
}
`)

	// RESOURCEurl
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "numbers",
        "fileurl": "http://example.com"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2026-04-15T17:09:02.57684679Z",
  "modifiedat": "2026-04-15T17:09:02.924358354Z",
  "ancestor": "1",
  "format": "numbers",
  "formatvalidated": false,
  "formatvalidatedreason": "Data stored externally",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Data stored externally",

  "fileurl": "http://example.com",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

}

func TestFormatStrictAvro(t *testing.T) {
	reg := NewRegistry("TestFormatStrictAvro")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rmFile, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)
	rmNoFile, xErr := gm.AddResourceModel("nofiles", "nofile", 0, true, true, false)
	XNoErr(t, xErr)

	rmFile.SetValidateFormat(true)
	rmFile.SetValidateCompatibility(true)
	rmFile.SetStrictValidation(true)
	rmFile.SetConsistentFormat(true)
	rmNoFile.SetValidateFormat(true)
	rmNoFile.SetValidateCompatibility(true)
	rmNoFile.SetStrictValidation(true)
	rmNoFile.SetConsistentFormat(true)

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
          "hasdocument": true,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true,
          "consistentformat": true
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": false,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true,
          "consistentformat": true
        }
      }
    }
  }
}
`)

	// hasdoc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "meta": {"compatibility": "backward" },
        "format": "avro",
        "file": "\"null\"" }`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:07.554485814Z",
  "modifiedat": "2026-04-15T11:16:07.554485814Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "avro",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/nofiles/f1", `{
        "format": "avro"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/nofiles/f1/versions/1\" to be non-compliant with its \"format\" (avro).",
  "detail": "The Resource (/dirs/d1/nofiles/f1) for Version \"/dirs/d1/nofiles/f1/versions/1\" does not have \"hasdocument\" in its resource model set to \"true\", and an empty/missing document is not compliant.",
  "subject": "/dirs/d1/nofiles/f1/versions/1",
  "args": {
    "format": "avro"
  },
  "source": "79ab0198e6b4:registry:format_avro:36"
}
`)

	// no doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "avro",
        "file": null
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (avro).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid avro schema file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "avro"
  },
  "source": "79ab0198e6b4:registry:format_avro:60"
}
`)

	// empty doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "avro",
        "file": ""
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (avro).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid avro schema file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "avro"
  },
  "source": "79ab0198e6b4:registry:format_avro:60"
}
`)

	// missing format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "file": "\"null\""
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:34.008113923Z",
  "modifiedat": "2026-04-15T11:16:34.135061948Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// unknown format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "unknown",
        "file": "\"null\""
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_unknown",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" has a \"format\" value (unknown) that it not supported.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "unknown"
  },
  "source": "79ab0198e6b4:registry:resource:1802"
}
`)

	// varying format - 1
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "AvrO",
      "file": "\"null\""
    },
    "v2": {
      "format": "jsonSchema",
      "file": "\"null\""
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"AvrO\" vs \"jsonSchema\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": null,
      "file": "\"null\""
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"\" vs \"jsonSchema\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 3
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "",
      "file": "\"null\""
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"format\" for \"/dirs/d1/files/f2/versions/v1\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "format"
  },
  "source": "79ab0198e6b4:registry:entity:1446"
}
`)

	// RESOURCEurl
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "avro",
        "fileurl": "http://example.com"
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_external",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" references a document stored outside of the Registry, therefore no validation was performed.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "source": "79ab0198e6b4:registry:format_avro:46"
}
`)

}

func TestFormatNotStrictAvro(t *testing.T) {
	reg := NewRegistry("TestFormatNotStrictAvro")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rmFile, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)
	rmNoFile, xErr := gm.AddResourceModel("nofiles", "nofile", 0, true, true, false)
	XNoErr(t, xErr)

	rmFile.SetValidateFormat(true)
	rmFile.SetValidateCompatibility(true)
	rmFile.SetStrictValidation(false)
	rmFile.SetConsistentFormat(true)
	rmNoFile.SetValidateFormat(true)
	rmNoFile.SetValidateCompatibility(true)
	rmNoFile.SetStrictValidation(false)
	rmNoFile.SetConsistentFormat(true)

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
          "hasdocument": true,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "consistentformat": true
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": false,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "consistentformat": true
        }
      }
    }
  }
}
`)

	// hasdoc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "meta": {"compatibility": "backward" },
        "format": "avro",
        "file": "\"null\"" }`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:07.554485814Z",
  "modifiedat": "2026-04-15T11:16:07.554485814Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "avro",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/nofiles/f1", `{
        "format": "avro"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/nofiles/f1/versions/1\" to be non-compliant with its \"format\" (avro).",
  "detail": "The Resource (/dirs/d1/nofiles/f1) for Version \"/dirs/d1/nofiles/f1/versions/1\" does not have \"hasdocument\" in its resource model set to \"true\", and an empty/missing document is not compliant.",
  "subject": "/dirs/d1/nofiles/f1/versions/1",
  "args": {
    "format": "avro"
  },
  "source": "79ab0198e6b4:registry:format_avro:36"
}
`)

	// no doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "avro",
        "file": null
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (avro).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid avro schema file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "avro"
  },
  "source": "79ab0198e6b4:registry:format_avro:60"
}
`)

	// empty doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "avro",
        "file": ""
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (avro).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid avro schema file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "avro"
  },
  "source": "79ab0198e6b4:registry:format_avro:60"
}
`)

	// missing format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "file": "\"null\""
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:34.008113923Z",
  "modifiedat": "2026-04-15T11:16:34.135061948Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// unknown format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "unknown",
        "file": "\"null\""
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2026-04-15T17:08:33.325493075Z",
  "modifiedat": "2026-04-15T17:08:33.500548614Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "unknown",
  "formatvalidated": false,
  "formatvalidatedreason": "Unknown format",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Unknown format",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// varying format - 1
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "aVRo",
      "file": "\"null\""
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"aVRo\" vs \"jsonSchema\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": null,
      "file": "\"null\""
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"\" vs \"jsonSchema\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 3
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "",
      "file": "\"null\""
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"format\" for \"/dirs/d1/files/f2/versions/v1\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "format"
  },
  "source": "79ab0198e6b4:registry:entity:1446"
}
`)

	// varying format - 4
	rmFile.SetConsistentFormat(false)
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "), 200, "*")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "meta": { "compatibility": "backWARD" },
   "versions": {
    "v1": {
      "format": "aVRo",
      "file": "\"null\""
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Version \"/dirs/d1/files/f2/versions/v1\" has a \"format\" value of \"aVRo\", was expecting \"jsonschema*\".",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "Version \"/dirs/d1/files/f2/versions/v1\" has a \"format\" value of \"aVRo\", was expecting \"jsonschema*\""
  },
  "source": "79ab0198e6b4:registry:format_jsonschema:137"
}
`)

	// RESOURCEurl
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "avro",
        "fileurl": "http://example.com"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2026-04-15T17:09:02.57684679Z",
  "modifiedat": "2026-04-15T17:09:02.924358354Z",
  "ancestor": "1",
  "format": "avro",
  "formatvalidated": false,
  "formatvalidatedreason": "Data stored externally",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Data stored externally",

  "fileurl": "http://example.com",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

}

func TestFormatStrictJson(t *testing.T) {
	reg := NewRegistry("TestFormatStrictJson")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rmFile, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)
	rmNoFile, xErr := gm.AddResourceModel("nofiles", "nofile", 0, true, true, false)
	XNoErr(t, xErr)

	rmFile.SetValidateFormat(true)
	rmFile.SetValidateCompatibility(true)
	rmFile.SetStrictValidation(true)
	rmFile.SetConsistentFormat(true)
	rmNoFile.SetValidateFormat(true)
	rmNoFile.SetValidateCompatibility(true)
	rmNoFile.SetStrictValidation(true)
	rmNoFile.SetConsistentFormat(true)

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
          "hasdocument": true,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true,
          "consistentformat": true
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": false,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true,
          "consistentformat": true
        }
      }
    }
  }
}
`)

	// hasdoc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "meta": {"compatibility": "backward" },
        "format": "jsonschema",
        "file": "{}" }`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:07.554485814Z",
  "modifiedat": "2026-04-15T11:16:07.554485814Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "jsonschema",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/nofiles/f1", `{
        "format": "jsonschema"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/nofiles/f1/versions/1\" to be non-compliant with its \"format\" (jsonschema).",
  "detail": "The Resource (/dirs/d1/nofiles/f1) for Version \"/dirs/d1/nofiles/f1/versions/1\" does not have \"hasdocument\" in its resource model set to \"true\", and an empty/missing document is not compliant.",
  "subject": "/dirs/d1/nofiles/f1/versions/1",
  "args": {
    "format": "jsonschema"
  },
  "source": "79ab0198e6b4:registry:format_jsonschema:36"
}
`)

	// no doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "jsonschema",
        "file": null
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (jsonschema).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid json schema file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "jsonschema"
  },
  "source": "79ab0198e6b4:registry:format_jsonschema:60"
}
`)

	// empty doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "jsonschema",
        "file": ""
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (jsonschema).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid json schema file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "jsonschema"
  },
  "source": "79ab0198e6b4:registry:format_jsonschema:60"
}
`)

	// missing format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "file": "{}"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:34.008113923Z",
  "modifiedat": "2026-04-15T11:16:34.135061948Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// unknown format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "unknown",
        "file": "{}"
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_unknown",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" has a \"format\" value (unknown) that it not supported.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "unknown"
  },
  "source": "79ab0198e6b4:registry:resource:1802"
}
`)

	// varying format - 1
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "JSONschema",
      "file": "{}"
    },
    "v2": {
      "format": "numbers",
      "file": "1"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"JSONschema\" vs \"numbers\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": null,
      "file": "{}"
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"\" vs \"jsonSchema\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 3
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "",
      "file": "{}"
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"format\" for \"/dirs/d1/files/f2/versions/v1\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "format"
  },
  "source": "79ab0198e6b4:registry:entity:1446"
}
`)

	// RESOURCEurl
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "jsonschema",
        "fileurl": "http://example.com"
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_external",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" references a document stored outside of the Registry, therefore no validation was performed.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "source": "79ab0198e6b4:registry:format_jsonschema:46"
}
`)

}

func TestFormatNotStrictJson(t *testing.T) {
	reg := NewRegistry("TestFormatNotStrictJson")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rmFile, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)
	rmNoFile, xErr := gm.AddResourceModel("nofiles", "nofile", 0, true, true, false)
	XNoErr(t, xErr)

	rmFile.SetValidateFormat(true)
	rmFile.SetValidateCompatibility(true)
	rmFile.SetStrictValidation(false)
	rmFile.SetConsistentFormat(true)
	rmNoFile.SetValidateFormat(true)
	rmNoFile.SetValidateCompatibility(true)
	rmNoFile.SetStrictValidation(false)
	rmNoFile.SetConsistentFormat(true)

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
          "hasdocument": true,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "consistentformat": true
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": false,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "consistentformat": true
        }
      }
    }
  }
}
`)

	// hasdoc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "meta": {"compatibility": "backward" },
        "format": "jsonschema",
        "file": "{}" }`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:07.554485814Z",
  "modifiedat": "2026-04-15T11:16:07.554485814Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "jsonschema",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/nofiles/f1", `{
        "format": "jsonschema"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/nofiles/f1/versions/1\" to be non-compliant with its \"format\" (jsonschema).",
  "detail": "The Resource (/dirs/d1/nofiles/f1) for Version \"/dirs/d1/nofiles/f1/versions/1\" does not have \"hasdocument\" in its resource model set to \"true\", and an empty/missing document is not compliant.",
  "subject": "/dirs/d1/nofiles/f1/versions/1",
  "args": {
    "format": "jsonschema"
  },
  "source": "79ab0198e6b4:registry:format_jsonschema:36"
}
`)

	// no doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "jsonschema",
        "file": null
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (jsonschema).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid json schema file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "jsonschema"
  },
  "source": "79ab0198e6b4:registry:format_jsonschema:60"
}
`)

	// empty doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "jsonschema",
        "file": ""
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (jsonschema).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid json schema file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "jsonschema"
  },
  "source": "79ab0198e6b4:registry:format_jsonschema:60"
}
`)

	// missing format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "file": "{}"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:34.008113923Z",
  "modifiedat": "2026-04-15T11:16:34.135061948Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// unknown format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "unknown",
        "file": "{}"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2026-04-15T17:08:33.325493075Z",
  "modifiedat": "2026-04-15T17:08:33.500548614Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "unknown",
  "formatvalidated": false,
  "formatvalidatedreason": "Unknown format",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Unknown format",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// varying format - 1
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "jSONschema",
      "file": "{}"
    },
    "v2": {
      "format": "avro",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"jSONschema\" vs \"avro\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": null,
      "file": "{}"
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"\" vs \"jsonSchema\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 3
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "",
      "file": "{}"
    },
    "v2": {
      "format": "jsonSchema",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"format\" for \"/dirs/d1/files/f2/versions/v1\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "format"
  },
  "source": "79ab0198e6b4:registry:entity:1446"
}
`)

	// varying format - 4
	rmFile.SetConsistentFormat(false)
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "), 200, "*")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "meta": { "compatibility": "backWARD" },
   "versions": {
    "v1": {
      "format": "jSONschema",
      "file": "{}"
    },
    "v2": {
      "format": "AvrO",
      "file": "\"null\""
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Version \"/dirs/d1/files/f2/versions/v1\" has a \"format\" value of \"jSONschema\", was expecting \"avro*\".",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "Version \"/dirs/d1/files/f2/versions/v1\" has a \"format\" value of \"jSONschema\", was expecting \"avro*\""
  },
  "source": "79ab0198e6b4:registry:format_jsonschema:137"
}
`)

	// RESOURCEurl
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "jsonschema",
        "fileurl": "http://example.com"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2026-04-15T17:09:02.57684679Z",
  "modifiedat": "2026-04-15T17:09:02.924358354Z",
  "ancestor": "1",
  "format": "jsonschema",
  "formatvalidated": false,
  "formatvalidatedreason": "Data stored externally",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Data stored externally",

  "fileurl": "http://example.com",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

}

func TestFormatStrictProtobuf(t *testing.T) {
	reg := NewRegistry("TestFormatStrictProtobuf")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rmFile, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)
	rmNoFile, xErr := gm.AddResourceModel("nofiles", "nofile", 0, true, true, false)
	XNoErr(t, xErr)

	rmFile.SetValidateFormat(true)
	rmFile.SetValidateCompatibility(true)
	rmFile.SetStrictValidation(true)
	rmFile.SetConsistentFormat(true)
	rmNoFile.SetValidateFormat(true)
	rmNoFile.SetValidateCompatibility(true)
	rmNoFile.SetStrictValidation(true)
	rmNoFile.SetConsistentFormat(true)

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
          "hasdocument": true,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true,
          "consistentformat": true
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": false,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true,
          "consistentformat": true
        }
      }
    }
  }
}
`)

	// hasdoc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "meta": {"compatibility": "backward" },
        "format": "protobuf",
        "file": "syntax = \"proto3\"; message E {}" }`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:07.554485814Z",
  "modifiedat": "2026-04-15T11:16:07.554485814Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "protobuf",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/nofiles/f1", `{
        "format": "protobuf"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/nofiles/f1/versions/1\" to be non-compliant with its \"format\" (protobuf).",
  "detail": "The Resource (/dirs/d1/nofiles/f1) for Version \"/dirs/d1/nofiles/f1/versions/1\" does not have \"hasdocument\" in its resource model set to \"true\", and an empty/missing document is not compliant.",
  "subject": "/dirs/d1/nofiles/f1/versions/1",
  "args": {
    "format": "protobuf"
  },
  "source": "79ab0198e6b4:registry:format_protobuf:36"
}
`)

	// no doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "protobuf",
        "file": null
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (protobuf).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid protobuf file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "protobuf"
  },
  "source": "79ab0198e6b4:registry:format_protobuf:60"
}
`)

	// empty doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "protobuf",
        "file": ""
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (protobuf).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid protobuf file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "protobuf"
  },
  "source": "79ab0198e6b4:registry:format_protobuf:60"
}
`)

	// missing format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "file": "syntax = \"proto3\"; message E {}"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:34.008113923Z",
  "modifiedat": "2026-04-15T11:16:34.135061948Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// unknown format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "unknown",
        "file": "syntax = \"proto3\"; message E {}"
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_unknown",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" has a \"format\" value (unknown) that it not supported.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "unknown"
  },
  "source": "79ab0198e6b4:registry:resource:1802"
}
`)

	// varying format - 1
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "protoBUF",
      "file": "syntax = \"proto3\"; message E {}"
    },
    "v2": {
      "format": "numbers",
      "file": "1"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"protoBUF\" vs \"numbers\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": null,
      "file": "syntax = \"proto3\"; message E {}"
    },
    "v2": {
      "format": "protobuf",
      "file": "syntax = \"proto3\"; message E {}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"\" vs \"protobuf\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 3
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "",
      "file": "syntax = \"proto3\"; message E {}"
    },
    "v2": {
      "format": "protobuf",
      "file": "syntax = \"proto3\"; message E {}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"format\" for \"/dirs/d1/files/f2/versions/v1\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "format"
  },
  "source": "79ab0198e6b4:registry:entity:1446"
}
`)

	// RESOURCEurl
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "protobuf",
        "fileurl": "http://example.com"
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_external",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" references a document stored outside of the Registry, therefore no validation was performed.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "source": "79ab0198e6b4:registry:format_protobuf:46"
}
`)

}

func TestFormatNotStrictProtobuf(t *testing.T) {
	reg := NewRegistry("TestFormatNotStrictProtobuf")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rmFile, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)
	rmNoFile, xErr := gm.AddResourceModel("nofiles", "nofile", 0, true, true, false)
	XNoErr(t, xErr)

	rmFile.SetValidateFormat(true)
	rmFile.SetValidateCompatibility(true)
	rmFile.SetStrictValidation(false)
	rmFile.SetConsistentFormat(true)
	rmNoFile.SetValidateFormat(true)
	rmNoFile.SetValidateCompatibility(true)
	rmNoFile.SetStrictValidation(false)
	rmNoFile.SetConsistentFormat(true)

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
          "hasdocument": true,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "consistentformat": true
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": false,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "consistentformat": true
        }
      }
    }
  }
}
`)

	// hasdoc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "meta": {"compatibility": "backward" },
        "format": "protobuf",
        "file": "syntax = \"proto3\"; message E {}" }`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:07.554485814Z",
  "modifiedat": "2026-04-15T11:16:07.554485814Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "protobuf",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/nofiles/f1", `{
        "format": "protobuf"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/nofiles/f1/versions/1\" to be non-compliant with its \"format\" (protobuf).",
  "detail": "The Resource (/dirs/d1/nofiles/f1) for Version \"/dirs/d1/nofiles/f1/versions/1\" does not have \"hasdocument\" in its resource model set to \"true\", and an empty/missing document is not compliant.",
  "subject": "/dirs/d1/nofiles/f1/versions/1",
  "args": {
    "format": "protobuf"
  },
  "source": "79ab0198e6b4:registry:format_protobuf:36"
}
`)

	// no doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "protobuf",
        "file": null
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (protobuf).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid protobuf file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "protobuf"
  },
  "source": "79ab0198e6b4:registry:format_protobuf:60"
}
`)

	// empty doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "protobuf",
        "file": ""
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (protobuf).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid protobuf file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "protobuf"
  },
  "source": "79ab0198e6b4:registry:format_protobuf:60"
}
`)

	// missing format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "file": "syntax = \"proto3\"; message E {}"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:34.008113923Z",
  "modifiedat": "2026-04-15T11:16:34.135061948Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// unknown format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "unknown",
        "file": "syntax = \"proto3\"; message E {}"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2026-04-15T17:08:33.325493075Z",
  "modifiedat": "2026-04-15T17:08:33.500548614Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "unknown",
  "formatvalidated": false,
  "formatvalidatedreason": "Unknown format",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Unknown format",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// varying format - 1
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "protoBUF",
      "file": "syntax = \"proto3\"; message E {}"
    },
    "v2": {
      "format": "avro",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"protoBUF\" vs \"avro\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": null,
      "file": "syntax = \"proto3\"; message E {}"
    },
    "v2": {
      "format": "protoBUF",
      "file": "syntax = \"proto3\"; message E {}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"\" vs \"protoBUF\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 3
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "",
      "file": "syntax = \"proto3\"; message E {}"
    },
    "v2": {
      "format": "protoBUF",
      "file": "syntax = \"proto3\"; message E {}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"format\" for \"/dirs/d1/files/f2/versions/v1\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "format"
  },
  "source": "79ab0198e6b4:registry:entity:1446"
}
`)

	// varying format - 4
	rmFile.SetConsistentFormat(false)
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "), 200, "*")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "meta": { "compatibility": "backWARD" },
   "versions": {
    "v1": {
      "format": "protoBUF",
      "file": "syntax = \"proto3\"; message E {}"
    },
    "v2": {
      "format": "AvrO",
      "file": "\"null\""
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Version \"/dirs/d1/files/f2/versions/v1\" has a \"format\" value of \"protoBUF\", was expecting \"avro*\".",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "Version \"/dirs/d1/files/f2/versions/v1\" has a \"format\" value of \"protoBUF\", was expecting \"avro*\""
  },
  "source": "79ab0198e6b4:registry:format_protobuf:137"
}
`)

	// RESOURCEurl
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "protobuf",
        "fileurl": "http://example.com"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2026-04-15T17:09:02.57684679Z",
  "modifiedat": "2026-04-15T17:09:02.924358354Z",
  "ancestor": "1",
  "format": "protobuf",
  "formatvalidated": false,
  "formatvalidatedreason": "Data stored externally",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Data stored externally",

  "fileurl": "http://example.com",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

}

func TestFormatStrictXMLSchema(t *testing.T) {
	reg := NewRegistry("TestFormatStrictXMLSchema")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rmFile, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)
	rmNoFile, xErr := gm.AddResourceModel("nofiles", "nofile", 0, true, true, false)
	XNoErr(t, xErr)

	rmFile.SetValidateFormat(true)
	rmFile.SetValidateCompatibility(true)
	rmFile.SetStrictValidation(true)
	rmFile.SetConsistentFormat(true)
	rmNoFile.SetValidateFormat(true)
	rmNoFile.SetValidateCompatibility(true)
	rmNoFile.SetStrictValidation(true)
	rmNoFile.SetConsistentFormat(true)

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
          "hasdocument": true,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true,
          "consistentformat": true
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": false,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true,
          "consistentformat": true
        }
      }
    }
  }
}
`)

	// hasdoc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "meta": {"compatibility": "backward" },
        "format": "xmlschema",
        "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
    }`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:07.554485814Z",
  "modifiedat": "2026-04-15T11:16:07.554485814Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "xmlschema",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/nofiles/f1", `{
        "format": "xmlschema"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/nofiles/f1/versions/1\" to be non-compliant with its \"format\" (xmlschema).",
  "detail": "The Resource (/dirs/d1/nofiles/f1) for Version \"/dirs/d1/nofiles/f1/versions/1\" does not have \"hasdocument\" in its resource model set to \"true\", and an empty/missing document is not compliant.",
  "subject": "/dirs/d1/nofiles/f1/versions/1",
  "args": {
    "format": "xmlschema"
  },
  "source": "79ab0198e6b4:registry:format_xmlschema:36"
}
`)

	// no doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "xmlschema",
        "file": null
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (xmlschema).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid xml schema file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "xmlschema"
  },
  "source": "79ab0198e6b4:registry:format_xmlschema:60"
}
`)

	// empty doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "xmlschema",
        "file": ""
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (xmlschema).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid xml schema file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "xmlschema"
  },
  "source": "79ab0198e6b4:registry:format_xmlschema:60"
}
`)

	// missing format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:34.008113923Z",
  "modifiedat": "2026-04-15T11:16:34.135061948Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// unknown format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "unknown",
        "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_unknown",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" has a \"format\" value (unknown) that it not supported.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "unknown"
  },
  "source": "79ab0198e6b4:registry:resource:1802"
}
`)

	// varying format - 1
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "XMLschema",
      "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
    },
    "v2": {
      "format": "numbers",
      "file": "1"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"XMLschema\" vs \"numbers\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": null,
      "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
    },
    "v2": {
      "format": "xmlschema",
      "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"\" vs \"xmlschema\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 3
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "",
      "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
    },
    "v2": {
      "format": "xmlschema",
      "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"format\" for \"/dirs/d1/files/f2/versions/v1\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "format"
  },
  "source": "79ab0198e6b4:registry:entity:1446"
}
`)

	// RESOURCEurl
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "xmlschema",
        "fileurl": "http://example.com"
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_external",
  "title": "Version \"/dirs/d1/files/f1/versions/1\" references a document stored outside of the Registry, therefore no validation was performed.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "source": "79ab0198e6b4:registry:format_xmlschema:46"
}
`)

}

func TestFormatNotStrictXMLSchema(t *testing.T) {
	reg := NewRegistry("TestFormatNotStrictXMLSchema")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rmFile, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)
	rmNoFile, xErr := gm.AddResourceModel("nofiles", "nofile", 0, true, true, false)
	XNoErr(t, xErr)

	rmFile.SetValidateFormat(true)
	rmFile.SetValidateCompatibility(true)
	rmFile.SetStrictValidation(false)
	rmFile.SetConsistentFormat(true)
	rmNoFile.SetValidateFormat(true)
	rmNoFile.SetValidateCompatibility(true)
	rmNoFile.SetStrictValidation(false)
	rmNoFile.SetConsistentFormat(true)

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
          "hasdocument": true,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "consistentformat": true
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": false,
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "consistentformat": true
        }
      }
    }
  }
}
`)

	// hasdoc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "meta": {"compatibility": "backward" },
        "format": "xmlschema",
        "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
  }`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:07.554485814Z",
  "modifiedat": "2026-04-15T11:16:07.554485814Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "xmlschema",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/nofiles/f1", `{
        "format": "xmlschema"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/nofiles/f1/versions/1\" to be non-compliant with its \"format\" (xmlschema).",
  "detail": "The Resource (/dirs/d1/nofiles/f1) for Version \"/dirs/d1/nofiles/f1/versions/1\" does not have \"hasdocument\" in its resource model set to \"true\", and an empty/missing document is not compliant.",
  "subject": "/dirs/d1/nofiles/f1/versions/1",
  "args": {
    "format": "xmlschema"
  },
  "source": "79ab0198e6b4:registry:format_xmlschema:36"
}
`)

	// no doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "xmlschema",
        "file": null
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (xmlschema).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid xml schema file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "xmlschema"
  },
  "source": "79ab0198e6b4:registry:format_xmlschema:60"
}
`)

	// empty doc
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "xmlschema",
        "file": ""
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f1/versions/1\" to be non-compliant with its \"format\" (xmlschema).",
  "detail": "Version \"/dirs/d1/files/f1/versions/1\" is empty and therefore not a valid xml schema file.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "format": "xmlschema"
  },
  "source": "79ab0198e6b4:registry:format_xmlschema:60"
}
`)

	// missing format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:34.008113923Z",
  "modifiedat": "2026-04-15T11:16:34.135061948Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// unknown format
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "unknown",
        "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2026-04-15T17:08:33.325493075Z",
  "modifiedat": "2026-04-15T17:08:33.500548614Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "unknown",
  "formatvalidated": false,
  "formatvalidatedreason": "Unknown format",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Unknown format",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// varying format - 1
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "xmlSCHEMA",
      "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
    },
    "v2": {
      "format": "avro",
      "file": "{}"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"xmlSCHEMA\" vs \"avro\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": null,
      "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
    },
    "v2": {
      "format": "xmlSCHEMA",
      "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_inconsistent",
  "title": "One or more Versions of Resource \"/dirs/d1/files/f2\" do not have the same \"format\" value as mandated by their owning Resource model's \"consistentformat\" attribute being set.",
  "detail": "Formats: \"\" vs \"xmlSCHEMA\".",
  "subject": "/dirs/d1/files/f2",
  "source": "79ab0198e6b4:registry:resource:1749"
}
`)

	// varying format - 3
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "versions": {
    "v1": {
      "format": "",
      "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
    },
    "v2": {
      "format": "xmlSCHEMA",
      "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"format\" for \"/dirs/d1/files/f2/versions/v1\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "format"
  },
  "source": "79ab0198e6b4:registry:entity:1446"
}
`)

	// varying format - 4
	rmFile.SetConsistentFormat(false)
	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "), 200, "*")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{
   "meta": { "compatibility": "backWARD" },
   "versions": {
    "v1": {
      "format": "xmlSCHEMA",
      "file": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"
    },
    "v2": {
      "format": "AvrO",
      "file": "\"null\""
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Version \"/dirs/d1/files/f2/versions/v1\" has a \"format\" value of \"xmlSCHEMA\", was expecting \"avro*\".",
  "subject": "/dirs/d1/files/f2/versions/v1",
  "args": {
    "error_detail": "Version \"/dirs/d1/files/f2/versions/v1\" has a \"format\" value of \"xmlSCHEMA\", was expecting \"avro*\""
  },
  "source": "79ab0198e6b4:registry:format_xmlschema:137"
}
`)

	// RESOURCEurl
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
        "format": "xmlschema",
        "fileurl": "http://example.com"
        }`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2026-04-15T17:09:02.57684679Z",
  "modifiedat": "2026-04-15T17:09:02.924358354Z",
  "ancestor": "1",
  "format": "xmlschema",
  "formatvalidated": false,
  "formatvalidatedreason": "Data stored externally",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Data stored externally",

  "fileurl": "http://example.com",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

}

func TestFormatCompatModes(t *testing.T) {
	reg := NewRegistry("TestFormatCompatModes")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rm, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)

	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)
	rm.SetStrictValidation(true)
	rm.SetConsistentFormat(true)

	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
      "meta": { "compatibility": "forward" },
      "versions": {
        "v1": {
          "format": "numbers",
          "file": "1"
        },
        "v2": {
          "format": "numbers",
          "file": "2"
        }
      }
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#compatibility_violation",
  "title": "The request would cause one or more Versions of \"/dirs/d1/files/f1\" to violate its compatibility rule (forward).",
  "detail": "Version \"/dirs/d1/files/f1/versions/v1\" (sum: 1) isn't \"forward\" compatible with \"/dirs/d1/files/f1/versions/v2\" (sum: 2).",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "compat": "forward"
  },
  "source": "a3d56ce41e09:registry:format_numbers:109"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
      "meta": { "compatibility": "forward" },
      "versions": {
        "v1": {
          "format": "numbers",
          "file": "1"
        },
        "v2": {
          "format": "numbers",
          "file": "1"
        }
      }
    }`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-16T17:12:38.231940065Z",
  "modifiedat": "2026-04-16T17:12:38.231940065Z",
  "ancestor": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
        "v1": {
          "format": "numbers",
          "file": "3"
        },
        "v2": {
          "format": "numbers",
          "file": "2"
        },
        "v3": {
          "format": "numbers",
          "file": "1"
        }
    }`, 200, `*`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", `{
        "meta": { "compatibility": "full" }
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#compatibility_violation",
  "title": "The request would cause one or more Versions of \"/dirs/d1/files/f1\" to violate its compatibility rule (full).",
  "detail": "Version \"/dirs/d1/files/f1/versions/v2\" (sum: 2) isn't \"full\" compatible with \"/dirs/d1/files/f1/versions/v1\" (sum: 3).",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "compat": "full"
  },
  "source": "a3d56ce41e09:registry:format_numbers:109"
}
`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", `{
      "meta": { "compatibility": "full" },
      "versions": {
        "v1": {
          "format": "numbers",
          "file": "3"
        },
        "v2": {
          "format": "numbers",
          "file": "3"
        },
        "v3": {
          "format": "numbers",
          "file": "3"
        }
      }
    }`, 200, `*`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", `{
      "meta": { "compatibility": "backward_transitive" },
      "versions": {
        "v1": {
          "format": "numbers",
          "file": "3"
        },
        "v2": {
          "format": "numbers",
          "file": "4"
        },
        "v3": {
          "format": "numbers",
          "file": "4"
        }
      }
    }`, 200, `*`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", `{
      "meta": { "compatibility": "backward_transitive" },
      "versions": {
        "v1": {
          "format": "numbers",
          "file": "3"
        },
        "v2": {
          "format": "numbers",
          "file": "4"
        },
        "v3": {
          "format": "numbers",
          "file": "2"
        }
      }
    }`, 400, `*`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", `{
      "meta": { "compatibility": "full_transitive" },
      "versions": {
        "v1": {
          "format": "numbers",
          "file": "3"
        },
        "v2": {
          "format": "numbers",
          "file": "1\n1\n1"
        },
        "v3": {
          "format": "numbers",
          "file": "2\n0\n1"
        }
      }
    }`, 200, `*`)

	// compatvalidated should be removed
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", `{
      "meta": { "compatibility": null }
    }`, 200, `{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 5,
  "isdefault": true,
  "createdat": "2026-04-16T17:27:32.182487206Z",
  "modifiedat": "2026-04-16T17:27:32.523524772Z",
  "ancestor": "v2",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

}
