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
	rm, xErr := gm.AddResourceModel("files", "file", 0, true, true)
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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": false,
          "validatecompatibility": false,
          "strictvalidation": false
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
  "instance": "xxx",
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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": false,
          "strictvalidation": false
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
  "ancestorid": "1",
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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": false,
          "validatecompatibility": false,
          "strictvalidation": false
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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": false,
          "strictvalidation": true
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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": false,
          "strictvalidation": false
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
			"access-control-allow-methods: DELETE, GET, OPTIONS, PATCH, POST, PUT",
			"content-disposition: f2",
			"content-location: http://localhost:8181/dirs/d1/files/f2/versions/1",
			"xRegistry-fileid: f2",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f2",
			"xRegistry-xid: /dirs/d1/files/f2",
			"xRegistry-epoch: 2",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.236399049Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.291099909Z",
			"xRegistry-ancestorid: 1",
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
  "instance": "xxx",
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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": false,
          "validatecompatibility": false,
          "strictvalidation": true
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
			"access-control-allow-methods: DELETE, GET, OPTIONS, PATCH, POST, PUT",
			"content-disposition: f2",
			"content-location: http://localhost:8181/dirs/d1/files/f2/versions/1",
			"xRegistry-fileid: f2",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f2",
			"xRegistry-xid: /dirs/d1/files/f2",
			"xRegistry-epoch: 3",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.236399049Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.291099909Z",
			"xRegistry-ancestorid: 1",
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
  "instance": "xxx",
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
			"access-control-allow-methods: DELETE, GET, OPTIONS, PATCH, POST, PUT",
			"content-disposition: f2",
			"content-location: http://localhost:8181/dirs/d1/files/f2/versions/1",
			"xRegistry-fileid: f2",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f2",
			"xRegistry-xid: /dirs/d1/files/f2",
			"xRegistry-epoch: 4",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.236399049Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.291099909Z",
			"xRegistry-ancestorid: 1",
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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": false,
          "strictvalidation": true
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
			"access-control-allow-methods: DELETE, GET, OPTIONS, PATCH, POST, PUT",
			"content-disposition: f3",
			"content-location: http://localhost:8181/dirs/d1/files/f3/versions/1",
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f3",
			"xRegistry-xid: /dirs/d1/files/f3",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.0Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.0Z",
			"xRegistry-ancestorid: 1",
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
	rm, xErr := gm.AddResourceModel("files", "file", 0, true, true)
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
  "instance": "xxx",
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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false
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
  "instance": "xxx",
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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": false,
          "validatecompatibility": false,
          "strictvalidation": false
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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true
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
			     "instance": "xxx",
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
			"access-control-allow-methods: DELETE, GET, OPTIONS, PATCH, POST, PUT",
			"content-disposition: f1",
			"content-location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.0Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.0Z",
			"xRegistry-ancestorid: 1",
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
  "instance": "xxx",
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
  "instance": "xxx",
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
			"access-control-allow-methods: DELETE, GET, OPTIONS, PATCH, POST, PUT",
			"content-disposition: f1",
			"content-location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch: 2",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.0Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.1Z",
			"xRegistry-ancestorid: 1",
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
			     "instance": "xxx",
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
  "instance": "xxx",
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
			"access-control-allow-methods: DELETE, GET, OPTIONS, PATCH, POST, PUT",
			"content-disposition: f1",
			"content-location: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 2",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"xRegistry-xid: /dirs/d1/files/f1/versions/2",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2026-03-13T20:24:48.0Z",
			"xRegistry-modifiedat: 2026-03-13T20:24:48.0Z",
			"xRegistry-ancestorid: 1",
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
  "instance": "xxx",
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
  "instance": "xxx",
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
  "instance": "xxx",
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
	rm, xErr := gm.AddResourceModel("files", "file", 0, true, true)
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
			     "instance": "xxx",
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
  "instance": "xxx",
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
  "instance": "xxx",
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
  "instance": "xxx",
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
			     "instance": "xxx",
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
			     "instance": "xxx",
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
  "instance": "xxx",
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
  "instance": "xxx",
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
	rm, xErr := gm.AddResourceModel("files", "file", 0, true, true)
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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": false,
          "strictvalidation": true
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
  "ancestorid": "1",
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
  "ancestorid": "1",
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
  "instance": "xxx",
  "source": "79ab0198e6b4:registry:resource:1795"
}
`)

}

func TestFormatStrict(t *testing.T) {
	reg := NewRegistry("TestFormatStrict")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rmFile, xErr := gm.AddResourceModel("files", "file", 0, true, true)
	XNoErr(t, xErr)
	rmNoFile, xErr := gm.AddResourceModel("nofiles", "nofile", 0, true, false)
	XNoErr(t, xErr)

	rmFile.SetValidateFormat(true)
	rmFile.SetValidateCompatibility(true)
	rmFile.SetStrictValidation(true)
	attr, _ := rmFile.AddAttr("format", STRING)
	attr.SetMatchVersions(true)
	rmNoFile.SetValidateFormat(true)
	rmNoFile.SetValidateCompatibility(true)
	rmNoFile.SetStrictValidation(true)

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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true,
          "attributes": {
            "format": {
              "name": "format",
              "type": "string",
              "matchversions": true
            }
          }
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": false,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": true
        }
      }
    }
  }
}
`)

	// format->sample file
	type aFormat struct {
		Name          string
		MixedName     string
		GoodFile      string
		BadFile       string
		AltFormat     string
		AltFormatFile string
	}

	formats := []aFormat{
		{
			Name:          "numbers",
			MixedName:     "nUmBers",
			GoodFile:      `1`,
			BadFile:       "bad one",
			AltFormat:     "jsonSchema",
			AltFormatFile: "{}",
		},
		{
			Name:          "jsonSchema",
			MixedName:     "JSonSChema",
			GoodFile:      `{}`,
			BadFile:       "bad one",
			AltFormat:     "nUmbers",
			AltFormatFile: "5",
		},
		{
			Name:          "avro",
			MixedName:     "AvRo",
			GoodFile:      `\"null\"`,
			BadFile:       "bad one",
			AltFormat:     "nUmbers",
			AltFormatFile: "5",
		},
		{
			Name:          "protobuf",
			MixedName:     "PrOTObUf",
			GoodFile:      `syntax = \"proto3\"; message E {}`,
			BadFile:       "bad one",
			AltFormat:     "nUmbers",
			AltFormatFile: "5",
		},
		{
			Name:          "xmlschema",
			MixedName:     "XmLScHema",
			GoodFile:      `<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>`,
			BadFile:       "bad one",
			AltFormat:     "nUmbers",
			AltFormatFile: "5",
		},
	}

	for _, af := range formats {

		// hasdoc
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f."+af.Name+"$details", `{
        "meta": {"compatibility": "backward" },
        "format": "`+af.Name+`",
        "file": "`+af.GoodFile+`" }`, 201, `{
  "fileid": "f.`+af.Name+`",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`$details",
  "xid": "/dirs/d1/files/f.`+af.Name+`",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:07.554485814Z",
  "modifiedat": "2026-04-15T11:16:07.554485814Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "`+af.Name+`",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`/versions",
  "versionscount": 1
}
`)

		XHTTP(t, reg, "PUT", "/dirs/d1/nofiles/f."+af.Name, `{
        "format": "`+af.Name+`"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/nofiles/f.`+af.Name+`/versions/1\" to be non-compliant with its \"format\" (`+af.Name+`).",
  "detail": "The Resource (/dirs/d1/nofiles/f.`+af.Name+`) for Version \"/dirs/d1/nofiles/f.`+af.Name+`/versions/1\" does not have \"hasdocument\" in its resource model set to \"true\", and an empty/missing document is not compliant.",
  "subject": "/dirs/d1/nofiles/f.`+af.Name+`/versions/1",
  "args": {
    "format": "`+af.Name+`"
  },
  "instance": "xxx",
  "source": "79ab0198e6b4:registry:format_numbers:36"
}
`)

		// no doc
		// For regex: escape " ( and source
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f."+af.Name+"$details", `{
        "format": "`+af.Name+`",
        "file": null
        }`, 400, `^{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \\"/dirs/d1/files/f.`+af.Name+`/versions/1\\" to be non-compliant with its \\"format\\" \(`+af.Name+`\).",
  "detail": "Version \\"/dirs/d1/files/f.`+af.Name+`/versions/1\\" is empty and therefore not a valid .* file.",
  "subject": "/dirs/d1/files/f.`+af.Name+`/versions/1",
  "args": {
    "format": "`+af.Name+`"
  },
  "instance": ".*",
  "source": ".*"
}
`)

		// empty doc
		// For regex: escape " ( and source
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f."+af.Name+"$details", `{
        "format": "`+af.Name+`",
        "file": ""
        }`, 400, `^{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \\"/dirs/d1/files/f.`+af.Name+`/versions/1\\" to be non-compliant with its \\"format\\" \(`+af.Name+`\).",
  "detail": "Version \\"/dirs/d1/files/f.`+af.Name+`/versions/1\\" is empty and therefore not a valid .* file.",
  "subject": "/dirs/d1/files/f.`+af.Name+`/versions/1",
  "args": {
    "format": "`+af.Name+`"
  },
  "instance": ".*",
  "source": ".*"
}
`)

		// missing format
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f."+af.Name+"$details", `{
        "file": "1"
        }`, 200, `{
  "fileid": "f.`+af.Name+`",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`$details",
  "xid": "/dirs/d1/files/f.`+af.Name+`",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:34.008113923Z",
  "modifiedat": "2026-04-15T11:16:34.135061948Z",
  "ancestorid": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`/versions",
  "versionscount": 1
}
`)

		// unknown format
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f."+af.Name+"$details", `{
        "format": "unknown",
        "file": "1"
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_unknown",
  "title": "Version \"/dirs/d1/files/f.`+af.Name+`/versions/1\" has a \"format\" value (unknown) that it not supported.",
  "subject": "/dirs/d1/files/f.`+af.Name+`/versions/1",
  "args": {
    "format": "unknown"
  },
  "instance": "xxx",
  "source": "79ab0198e6b4:registry:resource:1802"
}
`)

		// varying format - 1
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f2."+af.Name+"$details", `{
		   "versions": {
		    "v1": {
		      "format": "`+af.MixedName+`",
		      "file": "`+af.GoodFile+`"
		    },
		    "v2": {
		      "format": "`+af.AltFormat+`",
		      "file": "`+af.AltFormatFile+`"
		    }
		  }
		}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"format\" attribute across the Versions of \"/dirs/d1/files/f2.`+af.Name+`\" to be different.",
  "detail": "Unique values: 2. Versions w/o values: 0.",
  "subject": "/dirs/d1/files/f2.`+af.Name+`",
  "args": {
    "name": "format"
  },
  "instance": "xxx",
  "source": "3225fb09cd3a:registry:resource:2081"
}
`)

		// varying format - 2
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f2."+af.Name+"$details", `{
		   "versions": {
		    "v1": {
		      "format": null,
		      "file": "`+af.GoodFile+`"
		    },
		    "v2": {
		      "format": "`+af.AltFormat+`",
		      "file": "`+af.AltFormatFile+`"
		    }
		  }
		}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"format\" attribute across the Versions of \"/dirs/d1/files/f2.`+af.Name+`\" to be different.",
  "detail": "Unique values: 1. Versions w/o values: 1.",
  "subject": "/dirs/d1/files/f2.`+af.Name+`",
  "args": {
    "name": "format"
  },
  "instance": "xxx",
  "source": "3225fb09cd3a:registry:resource:2081"
}
`)

		// varying format - 3
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f2."+af.Name+"$details", `{
   "versions": {
    "v1": {
      "format": "",
      "file": "1"
    },
    "v2": {
      "format": "`+af.AltFormat+`",
      "file": "`+af.AltFormatFile+`"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"format\" for \"/dirs/d1/files/f2.`+af.Name+`/versions/v1\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f2.`+af.Name+`/versions/v1",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "format"
  },
  "instance": "xxx",
  "source": "79ab0198e6b4:registry:entity:1446"
}
`)

		// RESOURCEurl
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f1."+af.Name+"$details", `{
        "format": "`+af.Name+`",
        "fileurl": "http://example.com"
        }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_external",
  "title": "Version \"/dirs/d1/files/f1.`+af.Name+`/versions/1\" references a document stored outside of the Registry, therefore no validation was performed.",
  "subject": "/dirs/d1/files/f1.`+af.Name+`/versions/1",
  "instance": "xxx",
  "source": "79ab0198e6b4:registry:format_numbers:46"
}
`)

	}
}

func TestFormatNotStrict(t *testing.T) {
	reg := NewRegistry("TestFormatStrict")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rmFile, xErr := gm.AddResourceModel("files", "file", 0, true, true)
	XNoErr(t, xErr)
	rmNoFile, xErr := gm.AddResourceModel("nofiles", "nofile", 0, true, false)
	XNoErr(t, xErr)

	rmFile.SetValidateFormat(true)
	rmFile.SetValidateCompatibility(true)
	rmFile.SetStrictValidation(false)
	attr, _ := rmFile.AddAttr("format", STRING)
	attr.SetMatchVersions(true)
	rmNoFile.SetValidateFormat(true)
	rmNoFile.SetValidateCompatibility(true)
	rmNoFile.SetStrictValidation(false)
	attr, _ = rmNoFile.AddAttr("format", STRING)
	attr.SetMatchVersions(true)

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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "attributes": {
            "format": {
              "name": "format",
              "type": "string",
              "matchversions": true
            }
          }
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": false,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "attributes": {
            "format": {
              "name": "format",
              "type": "string",
              "matchversions": true
            }
          }
        }
      }
    }
  }
}
`)

	// format->sample file
	type aFormat struct {
		Name          string
		MixedName     string
		GoodFile      string
		BadFile       string
		AltFormat     string
		AltFormatFile string
	}

	formats := []aFormat{
		{
			Name:          "numbers",
			MixedName:     "nUmBers",
			GoodFile:      `1`,
			BadFile:       "bad one",
			AltFormat:     "jsonSchema",
			AltFormatFile: "{}",
		},
		{
			Name:          "jsonSchema",
			MixedName:     "JSonSChema",
			GoodFile:      `{}`,
			BadFile:       "bad one",
			AltFormat:     "nUmbers",
			AltFormatFile: "5",
		},
		{
			Name:          "avro",
			MixedName:     "AvRo",
			GoodFile:      `\"null\"`,
			BadFile:       "bad one",
			AltFormat:     "nUmbers",
			AltFormatFile: "5",
		},
		{
			Name:          "protobuf",
			MixedName:     "PrOTObUf",
			GoodFile:      `syntax = \"proto3\"; message E {}`,
			BadFile:       "bad one",
			AltFormat:     "nUmbers",
			AltFormatFile: "5",
		},
		{
			Name:          "xmlschema",
			MixedName:     "XmLScHema",
			GoodFile:      `<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>`,
			BadFile:       "bad one",
			AltFormat:     "nUmbers",
			AltFormatFile: "5",
		},
	}

	for _, af := range formats {

		// hasdoc
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f."+af.Name+"$details", `{
        "meta": {"compatibility": "backward" },
        "format": "`+af.Name+`",
        "file": "`+af.GoodFile+`" }`, 201, `{
  "fileid": "f.`+af.Name+`",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`$details",
  "xid": "/dirs/d1/files/f.`+af.Name+`",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:07.554485814Z",
  "modifiedat": "2026-04-15T11:16:07.554485814Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "`+af.Name+`",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`/versions",
  "versionscount": 1
}
`)

		XHTTP(t, reg, "PUT", "/dirs/d1/nofiles/f."+af.Name, `{
        "format": "`+af.Name+`"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/nofiles/f.`+af.Name+`/versions/1\" to be non-compliant with its \"format\" (`+af.Name+`).",
  "detail": "The Resource (/dirs/d1/nofiles/f.`+af.Name+`) for Version \"/dirs/d1/nofiles/f.`+af.Name+`/versions/1\" does not have \"hasdocument\" in its resource model set to \"true\", and an empty/missing document is not compliant.",
  "subject": "/dirs/d1/nofiles/f.`+af.Name+`/versions/1",
  "args": {
    "format": "`+af.Name+`"
  },
  "instance": "xxx",
  "source": "79ab0198e6b4:registry:format_numbers:36"
}
`)

		// no doc
		// For regex: escape " ( and source
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f."+af.Name+"$details", `{
        "format": "`+af.Name+`",
        "file": null
        }`, 400, `^{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \\"/dirs/d1/files/f.`+af.Name+`/versions/1\\" to be non-compliant with its \\"format\\" \(`+af.Name+`\).",
  "detail": "Version \\"/dirs/d1/files/f.`+af.Name+`/versions/1\\" is empty and therefore not a valid .* file.",
  "subject": "/dirs/d1/files/f.`+af.Name+`/versions/1",
  "args": {
    "format": "`+af.Name+`"
  },
  "instance": ".*",
  "source": ".*"
}
`)

		// empty doc
		// For regex: escape " ( and source
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f."+af.Name+"$details", `{
        "format": "`+af.Name+`",
        "file": ""
        }`, 400, `^{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \\"/dirs/d1/files/f.`+af.Name+`/versions/1\\" to be non-compliant with its \\"format\\" \(`+af.Name+`\).",
  "detail": "Version \\"/dirs/d1/files/f.`+af.Name+`/versions/1\\" is empty and therefore not a valid .* file.",
  "subject": "/dirs/d1/files/f.`+af.Name+`/versions/1",
  "args": {
    "format": "`+af.Name+`"
  },
  "instance": ".*",
  "source": ".*"
}
`)

		// missing format
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f."+af.Name+"$details", `{
        "file": "1"
        }`, 200, `{
  "fileid": "f.`+af.Name+`",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`$details",
  "xid": "/dirs/d1/files/f.`+af.Name+`",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-04-15T11:16:34.008113923Z",
  "modifiedat": "2026-04-15T11:16:34.135061948Z",
  "ancestorid": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`/versions",
  "versionscount": 1
}
`)

		// unknown format
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f."+af.Name+"$details", `{
        "format": "unknown",
        "file": "1"
        }`, 200, `{
  "fileid": "f.`+af.Name+`",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`$details",
  "xid": "/dirs/d1/files/f.`+af.Name+`",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2026-04-15T17:08:33.325493075Z",
  "modifiedat": "2026-04-15T17:08:33.500548614Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "unknown",
  "formatvalidated": false,
  "formatvalidatedreason": "Unknown format",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Unknown format",

  "metaurl": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`/versions",
  "versionscount": 1
}
`)

		// varying format - 1
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f2."+af.Name+"$details", `{
		   "versions": {
		    "v1": {
		      "format": "`+af.MixedName+`",
		      "file": "`+af.GoodFile+`"
		    },
		    "v2": {
		      "format": "`+af.AltFormat+`",
		      "file": "`+af.AltFormatFile+`"
		    }
		  }
		}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"format\" attribute across the Versions of \"/dirs/d1/files/f2.`+af.Name+`\" to be different.",
  "detail": "Unique values: 2. Versions w/o values: 0.",
  "subject": "/dirs/d1/files/f2.`+af.Name+`",
  "args": {
    "name": "format"
  },
  "instance": "xxx",
  "source": "3225fb09cd3a:registry:resource:2081"
}
`)

		// varying format - 2
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f2."+af.Name+"$details", `{
		   "versions": {
		    "v1": {
		      "format": null,
		      "file": "`+af.GoodFile+`"
		    },
		    "v2": {
		      "format": "`+af.AltFormat+`",
		      "file": "`+af.AltFormatFile+`"
		    }
		  }
		}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"format\" attribute across the Versions of \"/dirs/d1/files/f2.`+af.Name+`\" to be different.",
  "detail": "Unique values: 1. Versions w/o values: 1.",
  "subject": "/dirs/d1/files/f2.`+af.Name+`",
  "args": {
    "name": "format"
  },
  "instance": "xxx",
  "source": "3225fb09cd3a:registry:resource:2081"
}
`)

		// varying format - 3
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f2."+af.Name+"$details", `{
   "versions": {
    "v1": {
      "format": "",
      "file": "1"
    },
    "v2": {
      "format": "`+af.AltFormat+`",
      "file": "`+af.AltFormatFile+`"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"format\" for \"/dirs/d1/files/f2.`+af.Name+`/versions/v1\" is not valid: can't be an empty string.",
  "subject": "/dirs/d1/files/f2.`+af.Name+`/versions/v1",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "format"
  },
  "instance": "xxx",
  "source": "79ab0198e6b4:registry:entity:1446"
}
`)

		// varying format - 4
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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "attributes": {
            "format": {
              "name": "format",
              "type": "string",
              "matchversions": true
            }
          }
        },
        "nofiles": {
          "plural": "nofiles",
          "singular": "nofile",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": false,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "attributes": {
            "format": {
              "name": "format",
              "type": "string",
              "matchversions": true
            }
          }
        }
      }
    }
  }
}
`)

		XHTTP(t, reg, "PUT", "/dirs/d1/files/f2."+af.Name+"$details", `{
   "meta": { "compatibility": "backWARD" },
   "versions": {
    "v1": {
      "format": "`+af.MixedName+`",
      "file": "`+af.GoodFile+`"
    },
    "v2": {
      "format": "`+af.AltFormat+`",
      "file": "`+af.AltFormatFile+`"
    }
  }
}`, 400, `^{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Version \\"/dirs/d1/files/f2.`+af.Name+`/versions/v1\\" has a \\"format\\" value of \\"`+af.MixedName+`\\", was expecting \\".*\\".",
  "subject": "/dirs/d1/files/f2.`+af.Name+`/versions/v1",
  "args": {
    "error_detail": "Version \\"/dirs/d1/files/f2.`+af.Name+`/versions/v1\\" has a \\"format\\" value of \\"`+af.MixedName+`\\", was expecting \\".*\\""
  },
  "instance": ".*",
  "source": ".*"
}
`)

		// RESOURCEurl
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f."+af.Name+"$details", `{
        "format": "`+af.Name+`",
        "fileurl": "http://example.com"
        }`, 200, `{
  "fileid": "f.`+af.Name+`",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`$details",
  "xid": "/dirs/d1/files/f.`+af.Name+`",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2026-04-15T17:09:02.57684679Z",
  "modifiedat": "2026-04-15T17:09:02.924358354Z",
  "ancestorid": "1",
  "format": "`+af.Name+`",
  "formatvalidated": false,
  "formatvalidatedreason": "Data stored externally",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Data stored externally",

  "fileurl": "http://example.com",

  "metaurl": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f.`+af.Name+`/versions",
  "versionscount": 1
}
`)
	}
}

func TestFormatCompatModes(t *testing.T) {
	reg := NewRegistry("TestFormatCompatModes")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, xErr := model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rm, xErr := gm.AddResourceModel("files", "file", 0, true, true)
	XNoErr(t, xErr)

	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)
	rm.SetStrictValidation(true)

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
  "instance": "xxx",
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
  "ancestorid": "v1",
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
  "instance": "xxx",
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
  "ancestorid": "v2",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

}

// TestFormatXrefCascadeOnDirectSave (gaps item 1, remaining half) checks
// that formatvalidated correctly cascades into an xref source through the
// NORMAL per-Resource save path (runCascade() called directly from
// ValidateResource(), not via Registry.VerifyData()'s model-driven
// revalidation pass, which is what tests/xref_model_revalidation_test.go
// covers instead). format_test.go has zero xref mentions, so this was
// completely unverified before.
func TestFormatXrefCascadeOnDirectSave(t *testing.T) {
	reg := NewRegistry("TestFormatXrefCascadeOnDirectSave")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, _ := model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)

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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": false,
          "strictvalidation": false
        }
      }
    }
  }
}
`)

	// Create the target with a valid "numbers" document straight away.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "format": "numbers",
  "file": "1"
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:24:29.606428854Z",
  "modifiedat": "2026-07-27T00:24:29.606428854Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Create the xref source pointing at it.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T00:24:31.661658328Z",
  "modifiedat": "2026-07-27T00:24:31.661658328Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
  "defaultversionsticky": false
}
`)

	// The mirror must already show formatvalidated=true.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:24:33.708753687Z",
  "modifiedat": "2026-07-27T00:24:33.708753687Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Now directly PUT a new (still valid) document onto the TARGET - a
	// perfectly normal save, no model change involved at all.
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
  "v2": {
    "format": "numbers",
    "file": "2"
  }
}`, 200, `{
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2026-07-27T00:24:35.911759653Z",
    "modifiedat": "2026-07-27T00:24:35.911759653Z",
    "ancestorid": "1",
    "contenttype": "application/json",
    "format": "numbers",
    "formatvalidated": true
  }
}
`)

	// The xref mirror must reflect both the new default Version and the
	// (still true) formatvalidated flag for it.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:24:37.975662554Z",
  "modifiedat": "2026-07-27T00:24:37.975662554Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)
}

// TestFormatXrefCompatModeChangeCascade (gaps item 5) combines a Version-content-
// unrelated compatibility MODE change (meta.compatibility, an
// instance-level attribute) with an active xref, exercising the normal
// runCascade() fan-out path (as opposed to the model-driven
// Registry.VerifyData() path already covered by
// tests/xref_model_revalidation_test.go).
func TestFormatXrefCompatModeChangeCascade(t *testing.T) {
	reg := NewRegistry("TestFormatXrefCompatModeChangeCascade")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, _ := model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)

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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false
        }
      }
    }
  }
}
`)

	// v1 sum=2, v2 sum=2 - equal, so compatible under any mode, including
	// bidirectionally (needed since "full" checks both directions).
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "versions": {
    "v1": { "format": "numbers", "file": "2" },
    "v2": { "format": "numbers", "file": "2" }
  }
}`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:31.864526553Z",
  "modifiedat": "2026-07-27T00:25:31.864526553Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T00:25:33.901048058Z",
  "modifiedat": "2026-07-27T00:25:33.901048058Z",
  "readonly": false,
  "compatibility": "backward",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v2$details",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta", ``, 200,
		`{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T00:25:35.886918015Z",
  "modifiedat": "2026-07-27T00:25:35.886918015Z",
  "readonly": false,
  "compatibility": "backward",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v2$details",
  "defaultversionsticky": false
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:37.948014926Z",
  "modifiedat": "2026-07-27T00:25:37.948014926Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)

	// Change ONLY the compatibility MODE on the target - no version
	// content change at all - equal sums remain compatible under "full"
	// (bidirectional) too.
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details",
		`{"meta":{"compatibility":"full"}}`, 200,
		`{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:39.976131708Z",
  "modifiedat": "2026-07-27T00:25:40.114064757Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	// The xref mirror must reflect BOTH the new mode and the refreshed
	// compatibilityvalidated result.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta", ``, 200,
		`{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 2,
  "createdat": "2026-07-27T00:25:42.126789781Z",
  "modifiedat": "2026-07-27T00:25:42.272503534Z",
  "readonly": false,
  "compatibility": "full",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v2$details",
  "defaultversionsticky": false
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:44.2221894Z",
  "modifiedat": "2026-07-27T00:25:44.356102502Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)
}

// TestFormatXrefValidationFailureCascade verifies that a FAILED format
// validation state (formatvalidated=false/compatibilityvalidated=false,
// via an "unknown format" document - the only kind of format failure
// that doesn't outright reject the save, see EnsureCompat()) on the xref
// TARGET is correctly mirrored into the xref SOURCE too, not just the
// "everything is valid" happy path already covered by
// TestFormatXrefCascadeOnDirectSave. This is created BEFORE the xref
// exists, so the xref creation itself must pick up the already-failed
// state.
func TestFormatXrefValidationFailureCascade(t *testing.T) {
	reg := NewRegistry("TestFormatXrefValidationFailureCascade")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, _ := model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)

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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false
        }
      }
    }
  }
}
`)

	// Create the target with an "unknown" format - EnsureCompat() doesn't
	// reject this outright (only "strict" mode would), it just flags
	// formatvalidated=false/compatibilityvalidated=false with a
	// "Unknown format" reason.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "format": "unknown",
  "file": "1"
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:18:59.386435643Z",
  "modifiedat": "2026-07-27T01:18:59.386435643Z",
  "ancestorid": "1",
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

	// Create the xref source pointing at the already-failed target.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T01:19:01.83746876Z",
  "modifiedat": "2026-07-27T01:19:01.83746876Z",
  "readonly": false,
  "compatibility": "backward",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
  "defaultversionsticky": false
}
`)

	// The mirror must show the SAME failed state, not silently omit it
	// or default it back to "true"/absent.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:19:04.344048762Z",
  "modifiedat": "2026-07-27T01:19:04.344048762Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "unknown",
  "formatvalidated": false,
  "formatvalidatedreason": "Unknown format",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Unknown format",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Now fix the target's format - the failure state must clear on
	// both the target AND (more importantly) the mirror.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "format": "numbers",
  "file": "1"
}`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T01:19:06.908004939Z",
  "modifiedat": "2026-07-27T01:19:07.055228045Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T01:19:09.446274109Z",
  "modifiedat": "2026-07-27T01:19:09.587498469Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
}

// TestFormatXrefValidationFailureCascadeAfterUpdate is the "after the
// fact" variant of TestFormatXrefValidationFailureCascade: the xref is
// created while the target is VALID (so the mirror initially shows
// formatvalidated=true), and only THEN does the target transition into
// the failed ("unknown format") state via a normal direct update - the
// mirror must pick up that failure transition too, not just the initial
// state at xref-creation time.
func TestFormatXrefValidationFailureCascadeAfterUpdate(t *testing.T) {
	reg := NewRegistry("TestFormatXrefValidationFailureCascadeAfterUpdate")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, _ := model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)

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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false
        }
      }
    }
  }
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "format": "numbers",
  "file": "1"
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:19:54.525985019Z",
  "modifiedat": "2026-07-27T01:19:54.525985019Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T01:19:57.177137209Z",
  "modifiedat": "2026-07-27T01:19:57.177137209Z",
  "readonly": false,
  "compatibility": "backward",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
  "defaultversionsticky": false
}
`)

	// Confirm the mirror starts out fully valid.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:19:59.93952847Z",
  "modifiedat": "2026-07-27T01:19:59.93952847Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Flip the TARGET (already xref'd) to an unknown format via a
	// normal direct update - no model change involved.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "format": "unknown",
  "file": "1"
}`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T01:20:02.787321133Z",
  "modifiedat": "2026-07-27T01:20:02.984856352Z",
  "ancestorid": "1",
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

	// The mirror must now reflect the failure too.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T01:20:05.655873226Z",
  "modifiedat": "2026-07-27T01:20:05.845462478Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "unknown",
  "formatvalidated": false,
  "formatvalidatedreason": "Unknown format",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Unknown format",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
}

// TestFormatXrefTargetUpdateRejectionDoesNotCorruptMirror verifies that when a
// direct update to the xref TARGET is REJECTED outright (a real
// "compatibility_violation" 400, not just a formatvalidated/
// compatibilityvalidated=false flag - see FormatNumbers.IsCompatible())
// the target's data is left untouched AND the xref SOURCE's mirror
// remains correctly in-sync with the target's last-good state - i.e. a
// failed write to the target must not leave the mirror stale, corrupted,
// or partially updated.
func TestFormatXrefTargetUpdateRejectionDoesNotCorruptMirror(t *testing.T) {
	reg := NewRegistry("TestFormatXrefTargetUpdateRejectionDoesNotCorruptMirror")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, _ := model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)

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
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false
        }
      }
    }
  }
}
`)

	// Target starts with a single Version, sum=5.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "format": "numbers",
  "file": "5"
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:20:20.723551092Z",
  "modifiedat": "2026-07-27T01:20:20.723551092Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T01:20:22.72379931Z",
  "modifiedat": "2026-07-27T01:20:22.72379931Z",
  "readonly": false,
  "compatibility": "backward",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
  "defaultversionsticky": false
}
`)

	// Confirm the mirror's initial good state.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:20:24.854766301Z",
  "modifiedat": "2026-07-27T01:20:24.854766301Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Attempt to add a new default Version with sum=2 (< 5) - "backward"
	// compatibility requires the new Version's sum to be >= the old
	// one, so this must be REJECTED outright (not just flagged false).
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
  "v2": {
    "format": "numbers",
    "file": "2"
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#compatibility_violation",
  "title": "The request would cause one or more Versions of \"/dirs/d1/files/f1\" to violate its compatibility rule (backward).",
  "detail": "Version \"/dirs/d1/files/f1/versions/v2\" (sum: 2) isn't \"backward\" compatible with \"/dirs/d1/files/f1/versions/1\" (sum: 5).",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "compat": "backward"
  },
  "instance": "xxx",
  "source": ":registry:format_numbers:109"
}
`)

	// The target itself must be completely unaffected by the rejected
	// request - still just the original Version.
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:20:29.063641818Z",
  "modifiedat": "2026-07-27T01:20:29.063641818Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// And the mirror must ALSO still be showing that same untouched,
	// last-good state - not corrupted, not partially applied, and not
	// pointing at a Version that was never actually created.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:20:31.1941946Z",
  "modifiedat": "2026-07-27T01:20:31.1941946Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
}

// ---- Moved from xref_model_revalidation_test.go ----

func TestFormatXrefModelRevalidationFormatCascade(t *testing.T) {
	reg := NewRegistry("TestFormatXrefModelRevalidationFormatCascade")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)

	// Create target f1 with format set, but validateformat=false so
	// formatvalidated is NOT computed/set yet.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "format": "numbers",
  "file": "1\n2\n3"
}`, 201, `*`)

	// Confirm formatvalidated is absent (validateformat is off).
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200, `*`)

	// Create xref fx -> f1.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Confirm the xref mirror also has no formatvalidated yet.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `*`)

	// Now flip validateformat=true at the MODEL level (no change to f1's
	// own data at all) - this forces Registry.VerifyData() to
	// re-validate every existing Resource, including f1 (setting
	// formatvalidated=true, since "1\n2\n3" is valid "numbers" format)
	// and fx (an xref source, whose runCascade() re-run should pick up
	// the target's fresh formatvalidated=true value).
	rm.SetValidateFormat(true)
	modelSrc := reg.Model.MustUserMarshal("", "  ")
	XHTTP(t, reg, "PUT", "/modelsource", modelSrc, 200, `*`)

	// The TARGET f1 must now show formatvalidated=true.
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200,
		`^(?s)^.*"epoch": 1,.*"formatvalidated": true`)

	// The xref SOURCE fx must mirror that same formatvalidated=true -
	// this is the part that was missing coverage.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`^(?s)^.*"epoch": 1,.*"formatvalidated": true`)
}

// TestFormatXrefModelRevalidationCompatCascade is the compatibilityvalidated
// analog of TestFormatXrefModelRevalidationFormatCascade above (gaps item 1's
// remaining "compatibilityvalidated variant" sub-case) - it checks that
// when a model change forces Registry.VerifyData() to re-validate an
// xref TARGET's compatibilityvalidated system attribute (with no other
// attribute change), that fresh value is correctly cascaded into any
// xref SOURCE pointing at it, exactly as formatvalidated is above.
func TestFormatXrefModelRevalidationCompatCascade(t *testing.T) {
	reg := NewRegistry("TestFormatXrefModelRevalidationCompatCascade")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)

	// Create target f1 with two versions (equal sums, so compatible
	// under any mode), but validatecompatibility=false so
	// compatibilityvalidated is NOT computed/set yet.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "versions": {
    "v1": { "format": "numbers", "file": "2" },
    "v2": { "format": "numbers", "file": "2" }
  }
}`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:28:13.175487333Z",
  "modifiedat": "2026-07-27T00:28:13.175487333Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	// Confirm compatibilityvalidated is absent (validatecompatibility is
	// off).
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:28:15.548850155Z",
  "modifiedat": "2026-07-27T00:28:15.548850155Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	// Create xref fx -> f1.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T00:28:17.538190459Z",
  "modifiedat": "2026-07-27T00:28:17.538190459Z",
  "readonly": false,
  "compatibility": "backward",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v2$details",
  "defaultversionsticky": false
}
`)

	// Confirm the xref mirror also has no compatibilityvalidated yet.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:28:19.619210932Z",
  "modifiedat": "2026-07-27T00:28:19.619210932Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)

	// Now flip validatecompatibility=true at the MODEL level (no change
	// to f1's own data at all) - this forces Registry.VerifyData() to
	// re-validate every existing Resource, including f1 (setting
	// compatibilityvalidated=true, since v1/v2 are compatible under
	// "backward") and fx (an xref source, whose runCascade() re-run
	// should pick up the target's fresh compatibilityvalidated=true
	// value).
	rm.SetValidateCompatibility(true)
	modelSrc := reg.Model.MustUserMarshal("", "  ")
	XHTTP(t, reg, "PUT", "/modelsource", modelSrc, 200, `{
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "required": true,
      "default": "`+SPECVERSION+`"
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
    "shortself": {
      "name": "shortself",
      "type": "url",
      "readonly": true,
      "immutable": true
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
    },
    "dirsurl": {
      "name": "dirsurl",
      "type": "url",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "dirscount": {
      "name": "dirscount",
      "type": "uinteger",
      "readonly": true,
      "required": true
    },
    "dirs": {
      "name": "dirs",
      "type": "map",
      "item": {
        "type": "object",
        "attributes": {
          "*": {
            "name": "*",
            "type": "any"
          }
        }
      }
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "dirid": {
          "name": "dirid",
          "type": "string",
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
        "shortself": {
          "name": "shortself",
          "type": "url",
          "readonly": true,
          "immutable": true
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
        "deprecated": {
          "name": "deprecated",
          "type": "object",
          "attributes": {
            "alternative": {
              "name": "alternative",
              "type": "url"
            },
            "documentation": {
              "name": "documentation",
              "type": "url"
            },
            "effective": {
              "name": "effective",
              "type": "timestamp"
            },
            "removal": {
              "name": "removal",
              "type": "timestamp"
            },
            "*": {
              "name": "*",
              "type": "any"
            }
          }
        },
        "constraints": {
          "name": "constraints",
          "type": "map",
          "item": {
            "type": "object",
            "attributes": {
              "default": {
                "name": "default",
                "type": "any"
              },
              "enum": {
                "name": "enum",
                "type": "array",
                "item": {
                  "type": "any"
                }
              },
              "equals": {
                "name": "equals",
                "type": "string"
              }
            }
          }
        },
        "filesurl": {
          "name": "filesurl",
          "type": "url",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "filescount": {
          "name": "filescount",
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "files": {
          "name": "files",
          "type": "map",
          "item": {
            "type": "object",
            "attributes": {
              "*": {
                "name": "*",
                "type": "any"
              }
            }
          }
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
          "strictvalidation": false,
          "attributes": {
            "fileid": {
              "name": "fileid",
              "type": "string",
              "immutable": true,
              "required": true
            },
            "versionid": {
              "name": "versionid",
              "type": "string",
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
            "shortself": {
              "name": "shortself",
              "type": "url",
              "readonly": true,
              "immutable": true
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
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true,
              "required": true,
              "default": false
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
            "ancestorid": {
              "name": "ancestorid",
              "type": "string",
              "required": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
            },
            "format": {
              "name": "format",
              "type": "string"
            },
            "formatvalidated": {
              "name": "formatvalidated",
              "type": "boolean",
              "readonly": true
            },
            "formatvalidatedreason": {
              "name": "formatvalidatedreason",
              "type": "string",
              "readonly": true
            },
            "compatibilityvalidated": {
              "name": "compatibilityvalidated",
              "type": "boolean",
              "readonly": true
            },
            "compatibilityvalidatedreason": {
              "name": "compatibilityvalidatedreason",
              "type": "string",
              "readonly": true
            },
            "fileurl": {
              "name": "fileurl",
              "type": "url"
            },
            "fileproxyurl": {
              "name": "fileproxyurl",
              "type": "url"
            },
            "file": {
              "name": "file",
              "type": "any"
            },
            "filebase64": {
              "name": "filebase64",
              "type": "string"
            }
          },
          "resourceattributes": {
            "fileid": {
              "name": "fileid",
              "type": "string",
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
            "shortself": {
              "name": "shortself",
              "type": "url",
              "readonly": true,
              "immutable": true
            },
            "xid": {
              "name": "xid",
              "type": "xid",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "metaurl": {
              "name": "metaurl",
              "type": "url",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "meta": {
              "name": "meta",
              "type": "object",
              "attributes": {
                "*": {
                  "name": "*",
                  "type": "any"
                }
              }
            },
            "versionsurl": {
              "name": "versionsurl",
              "type": "url",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "versionscount": {
              "name": "versionscount",
              "type": "uinteger",
              "readonly": true,
              "required": true
            },
            "versions": {
              "name": "versions",
              "type": "map",
              "item": {
                "type": "object",
                "attributes": {
                  "*": {
                    "name": "*",
                    "type": "any"
                  }
                }
              }
            }
          },
          "metaattributes": {
            "fileid": {
              "name": "fileid",
              "type": "string",
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
            "shortself": {
              "name": "shortself",
              "type": "url",
              "readonly": true,
              "immutable": true
            },
            "xid": {
              "name": "xid",
              "type": "xid",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "xref": {
              "name": "xref",
              "type": "url"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "readonly": true,
              "required": true
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
            "readonly": {
              "name": "readonly",
              "type": "boolean",
              "readonly": true,
              "required": true,
              "default": false
            },
            "compatibility": {
              "name": "compatibility",
              "type": "string",
              "enum": [
                "backward",
                "backward_transitive",
                "forward",
                "forward_transitive",
                "full",
                "full_transitive"
              ],
              "strict": true
            },
            "deprecated": {
              "name": "deprecated",
              "type": "object",
              "attributes": {
                "alternative": {
                  "name": "alternative",
                  "type": "url"
                },
                "documentation": {
                  "name": "documentation",
                  "type": "url"
                },
                "effective": {
                  "name": "effective",
                  "type": "timestamp"
                },
                "removal": {
                  "name": "removal",
                  "type": "timestamp"
                },
                "*": {
                  "name": "*",
                  "type": "any"
                }
              }
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "required": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
              "readonly": true,
              "required": true
            },
            "defaultversionsticky": {
              "name": "defaultversionsticky",
              "type": "boolean",
              "required": true,
              "default": false
            }
          }
        }
      }
    }
  }
}
`)

	// The TARGET f1 must now show compatibilityvalidated=true.
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200,
		`{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:28:23.801192485Z",
  "modifiedat": "2026-07-27T00:28:23.801192485Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	// The xref SOURCE fx must mirror that same
	// compatibilityvalidated=true - this is the part that was missing
	// coverage.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:28:25.925416292Z",
  "modifiedat": "2026-07-27T00:28:25.925416292Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)
}

// TestFormatModelOffClearsStaleFormatValidated verifies that turning
// validateformat off at the model level (ApplyNewModel()'s
// clearValidationSystemProps() bulk sweep) correctly clears a stale
// formatvalidated/formatvalidatedreason value on a Resource that is
// NOT touched again after the model change - EnsureCompat() no longer
// re-clears defensively on every save while validation is off, so this
// one-time model-transition sweep is the only thing that can catch it.
func TestFormatModelOffClearsStaleFormatValidated(t *testing.T) {
	reg := NewRegistry("TestFormatModelOffClearsStaleFormatValidated")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)

	// Create f1 with a valid "numbers" format -> formatvalidated=true.
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
  "createdat": "2026-07-28T13:00:00.0Z",
  "modifiedat": "2026-07-28T13:00:00.0Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Turn validateformat OFF at the model level - f1 is NOT touched
	// again.
	rm.SetValidateFormat(false)
	XHTTP(t, reg, "PUT", "/modelsource", reg.Model.MustUserMarshal("", "  "),
		200, `*`)

	// f1's formatvalidated must now be gone, purely from the
	// model-transition bulk sweep - not from any per-save clear.
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-28T13:00:00.0Z",
  "modifiedat": "2026-07-28T13:00:00.0Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)
}

// TestFormatModelOffClearsStaleCompatValidated is the
// compatibilityvalidated analog of
// TestFormatModelOffClearsStaleFormatValidated above - turning
// validatecompatibility off at the model level must clear a stale
// compatibilityvalidated/compatibilityvalidatedreason value on an
// untouched Resource, while leaving formatvalidated (still on) alone.
func TestFormatModelOffClearsStaleCompatValidated(t *testing.T) {
	reg := NewRegistry("TestFormatModelOffClearsStaleCompatValidated")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)

	// Create f1 with two compatible versions (equal sums) and
	// "compatibility":"backward" set -> compatibilityvalidated=true.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "versions": {
    "v1": { "format": "numbers", "file": "2" },
    "v2": { "format": "numbers", "file": "2" }
  }
}`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-28T13:00:00.0Z",
  "modifiedat": "2026-07-28T13:00:00.0Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	// Turn validatecompatibility OFF at the model level (validateformat
	// stays ON) - f1 is NOT touched again.
	rm.SetValidateCompatibility(false)
	XHTTP(t, reg, "PUT", "/modelsource", reg.Model.MustUserMarshal("", "  "),
		200, `*`)

	// f1's compatibilityvalidated must now be gone, but formatvalidated
	// (still on) must remain - purely from the model-transition bulk
	// sweep, since f1 was never re-saved.
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-28T13:00:00.0Z",
  "modifiedat": "2026-07-28T13:00:00.0Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)
}
