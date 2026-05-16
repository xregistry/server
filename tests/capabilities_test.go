package tests

import (
	"encoding/json"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestCapabilitySimple(t *testing.T) {
	reg := NewRegistry("TestCapabilitySimple")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "GET", "/capabilities/foo", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/http.md#api_not_found",
  "title": "The specified API is not supported: /capabilities/foo.",
  "subject": "/capabilities/foo",
  "source": ":registry:httpStuff:1258"
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "capabilitiesoffered": {
      "mutable": false
    },
    "entities": {
      "mutable": true
    },
    "export": {
      "mutable": false
    },
    "model": {
      "mutable": false
    },
    "modelsource": {
      "mutable": true
    }
  },
  "compatibilities": {
    "avro*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "jsonschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "numbers": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "protobuf*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "xmlschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ]
  },
  "flags": [
    "binary",
    "collections",
    "doc",
    "epoch",
    "filter",
    "ignore",
    "inline",
    "setdefaultversionid",
    "sort",
    "specversion"
  ],
  "formats": [
    "avro*",
    "jsonschema*",
    "numbers",
    "protobuf*",
    "xmlschema*"
  ],
  "ignores": [
    "capabilities",
    "defaultversionid",
    "defaultversionsticky",
    "epoch",
    "id",
    "modelsource",
    "readonly"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "createdat",
    "manual"
  ]
}
`)

	XHTTP(t, reg, "GET", "?inline=capabilities", ``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilitySimple",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

  "capabilities": {
    "available": {
      "capabilities": {
        "mutable": true
      },
      "capabilitiesoffered": {
        "mutable": false
      },
      "entities": {
        "mutable": true
      },
      "export": {
        "mutable": false
      },
      "model": {
        "mutable": false
      },
      "modelsource": {
        "mutable": true
      }
    },
    "compatibilities": {
      "avro*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "jsonschema*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "numbers": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "protobuf*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "xmlschema*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ]
    },
    "flags": [
      "binary",
      "collections",
      "doc",
      "epoch",
      "filter",
      "ignore",
      "inline",
      "setdefaultversionid",
      "sort",
      "specversion"
    ],
    "formats": [
      "avro*",
      "jsonschema*",
      "numbers",
      "protobuf*",
      "xmlschema*"
    ],
    "ignores": [
      "capabilities",
      "defaultversionid",
      "defaultversionsticky",
      "epoch",
      "id",
      "modelsource",
      "readonly"
    ],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "`+SPECVERSION+`"
    ],
    "stickyversions": true,
    "versionmodes": [
      "createdat",
      "manual"
    ]
  }
}
`)

	tests := []struct {
		Name string
		Cap  string
		Exp  string
	}{
		{
			Name: "empty",
			Cap:  `{}`,
			Exp: `{
  "available": {
    "entities": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "` + SPECVERSION + `"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}`,
		},
		{
			Name: "full mutable",
			Cap: `{"available":{
                     "modelsource":{"mutable":true},
                     "entities":{"mutable":true},
                     "capabilities":{"mutable":true}}}`,
			Exp: `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    },
    "modelsource": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "` + SPECVERSION + `"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}`,
		},
		{
			Name: "dup mutable",
			Cap: `{"available":{
                     "modelsource":{"mutable":true},
                     "entities":{"mutable":true},
                     "capabilities":{"mutable":true}}}`,
			Exp: `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    },
    "modelsource": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "` + SPECVERSION + `"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}`,
		},

		{
			Name: "missing specversion",
			Cap:  `{"specversions":[]}`,
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_missing_value",
  "title": "The \"specversions\" capability needs to contain \"1.0-rc2\".",
  "subject": "/capabilities",
  "args": {
    "name": "specversions",
    "value": "1.0-rc2"
  },
  "source": ":common:capabilities:232"
}`,
		},

		{
			Name: "extra key",
			Cap:  `{"pagination": true, "bad": true}`,
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_unknown",
  "title": "Unknown capability specified: bad.",
  "subject": "/capabilities",
  "args": {
    "field": "bad"
  },
  "source": ":common:capabilities:251"
}`,
		},
	}

	for _, test := range tests {
		c, xErr := ParseCapabilitiesJSON([]byte(test.Cap))
		if xErr == nil {
			xErr = c.Validate()
		}
		res := ""
		if xErr != nil {
			res = xErr.String()
		} else {
			buf, _ := json.MarshalIndent(c, "", "  ")
			res = string(buf)
		}
		XEqual(t, test.Name, res, test.Exp)
	}
}

func TestCapabilityPath(t *testing.T) {
	reg := NewRegistry("TestCapabilityPath")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "capabilitiesoffered": {
      "mutable": false
    },
    "entities": {
      "mutable": true
    },
    "export": {
      "mutable": false
    },
    "model": {
      "mutable": false
    },
    "modelsource": {
      "mutable": true
    }
  },
  "compatibilities": {
    "avro*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "jsonschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "numbers": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "protobuf*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "xmlschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ]
  },
  "flags": [
    "binary",
    "collections",
    "doc",
    "epoch",
    "filter",
    "ignore",
    "inline",
    "setdefaultversionid",
    "sort",
    "specversion"
  ],
  "formats": [
    "avro*",
    "jsonschema*",
    "numbers",
    "protobuf*",
    "xmlschema*"
  ],
  "ignores": [
    "capabilities",
    "defaultversionid",
    "defaultversionsticky",
    "epoch",
    "id",
    "modelsource",
    "readonly"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "createdat",
    "manual"
  ]
}
`)

	// Verify current epoch value
	XHTTP(t, reg, "GET", "/", ``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityPath",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z"
}
`)

	// Try to clear it all - some can't be totally erased
	XHTTP(t, reg, "PUT", "/capabilities", `{"flags":["inline"]}`, 200,
		`{
  "available": {
    "entities": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [
    "inline"
  ],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}
`)

	// Make sure it's turned off, but turn it on for the rest of the
	// tests
	XHTTP(t, reg, "GET", "/capabilities", ``, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_available",
  "title": "The requested data (/capabilities) is not available.",
  "subject": "/capabilities",
  "source": "b1fcff68b7f8:registry:httpStuff:655"
}
`)

	XNoErr(t, reg.Refresh(registry.FOR_WRITE))
	reg.Capabilities.SetAvailable("capabilities", true)
	reg.Capabilities.SetAvailable("entities", true)
	XNoErr(t, reg.SaveCapabilities())
	// Epoch should now be 3

	// Notice no flags are enabled, so inline is ignored
	XHTTP(t, reg, "PUT", "/?inline=capabilities",
		`{"capabilities":{"available":{"capabilities":{"mutable":true}}}}`,
		200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityPath",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	// Make sure the Registry epoch changed
	XHTTP(t, reg, "GET", "/", ``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityPath",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}
`)

	// Setting to nulls
	XHTTP(t, reg, "PUT", "/capabilities", `{
  "available": {"capabilities":{"mutable":true}},
  "flags": null,
  "ignores": null,
  "pagination": null,
  "shortself": null,
  "specversions": null
}`, 200,
		`{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}
`)

	// Testing setting everything to the default
	XHTTP(t, reg, "PUT", "/capabilities", `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "capabilitiesoffered": {
      "mutable": false
    },
    "entities": {
      "mutable": true
    },
    "export": {
      "mutable": false
    },
    "model": {
      "mutable": false
    },
    "modelsource": {
      "mutable": true
    }
  },
  "compatibilities": {
    "avro*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "jsonschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "numbers": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "protobuf*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "xmlschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ]
  },
  "flags": [
    "binary", "collections", "doc", "epoch", "filter", "inline", "ignore",
    "setdefaultversionid", "sort", "specversion"
  ],
  "formats": [
    "avro*",
    "jsonschema*",
    "numbers",
    "protobuf*",
    "xmlschema*"
  ],
  "ignores": [ "capabilities", "defaultversionid", "defaultversionsticky",
    "epoch", "id", "modelsource", "readonly" ],
  "pagination": false,
  "shortself": false,
  "specversions": [ "`+SPECVERSION+`" ],
  "stickyversions": true,
  "versionmodes": [ "createdat", "manual" ]
}`, 200,
		`{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "capabilitiesoffered": {
      "mutable": false
    },
    "entities": {
      "mutable": true
    },
    "export": {
      "mutable": false
    },
    "model": {
      "mutable": false
    },
    "modelsource": {
      "mutable": true
    }
  },
  "compatibilities": {
    "avro*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "jsonschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "numbers": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "protobuf*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "xmlschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ]
  },
  "flags": [
    "binary",
    "collections",
    "doc",
    "epoch",
    "filter",
    "ignore",
    "inline",
    "setdefaultversionid",
    "sort",
    "specversion"
  ],
  "formats": [
    "avro*",
    "jsonschema*",
    "numbers",
    "protobuf*",
    "xmlschema*"
  ],
  "ignores": [
    "capabilities",
    "defaultversionid",
    "defaultversionsticky",
    "epoch",
    "id",
    "modelsource",
    "readonly"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "createdat",
    "manual"
  ]
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "capabilitiesoffered": {
      "mutable": false
    },
    "entities": {
      "mutable": true
    },
    "export": {
      "mutable": false
    },
    "model": {
      "mutable": false
    },
    "modelsource": {
      "mutable": true
    }
  },
  "compatibilities": {
    "avro*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "jsonschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "numbers": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "protobuf*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "xmlschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ]
  },
  "flags": [
    "binary",
    "collections",
    "doc",
    "epoch",
    "filter",
    "ignore",
    "inline",
    "setdefaultversionid",
    "sort",
    "specversion"
  ],
  "formats": [
    "avro*",
    "jsonschema*",
    "numbers",
    "protobuf*",
    "xmlschema*"
  ],
  "ignores": [
    "capabilities",
    "defaultversionid",
    "defaultversionsticky",
    "epoch",
    "id",
    "modelsource",
    "readonly"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "createdat",
    "manual"
  ]
}
`)

	// Setting to minimal
	XHTTP(t, reg, "PUT", "/capabilities", `{"available":{"capabilities":{"mutable":true}}}`,
		200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}
`)

	// Test some bools
	XHTTP(t, reg, "PUT", "/capabilities", `{
    "available":{"capabilities":{"mutable":true}},
	"pagination": false,
	"shortself": false,
    "stickyversions": false
}`, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": false,
  "versionmodes": [
    "manual"
  ]
}
`)

	XHTTP(t, reg, "PUT", "/capabilities", `{"pagination":true}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_value",
  "title": "Invalid value (true) specified for capability \"pagination\". Allowable values include: false.",
  "subject": "/capabilities",
  "args": {
    "field": "pagination",
    "list": "false",
    "value": "true"
  },
  "source": ":common:capabilities:216"
}
`)

	XHTTP(t, reg, "PUT", "/capabilities", `{"pagination":"false"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_error",
  "title": "There was an error in the capabilities provided: error parsing data: path '.pagination': expected \"bool\", got \"string\".",
  "subject": "/capabilities",
  "args": {
    "error_detail": "error parsing data: path '.pagination': expected \"bool\", got \"string\""
  },
  "source": ":common:capabilities:256"
}
`)

	XHTTP(t, reg, "PUT", "/capabilities", `{"shortself":true}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_value",
  "title": "Invalid value (true) specified for capability \"shortself\". Allowable values include: false.",
  "subject": "/capabilities",
  "args": {
    "field": "shortself",
    "list": "false",
    "value": "true"
  },
  "source": ":common:capabilities:229"
}
`)

	// Setting some arrays to [] are an error because we can't do what they
	// asked - which is different from "null"/absent - which means "default"
	XHTTP(t, reg, "PUT", "/capabilities", `{ "specversions": [] }`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_missing_value",
  "title": "The \"specversions\" capability needs to contain \"1.0-rc2\".",
  "subject": "/capabilities",
  "args": {
    "name": "specversions",
    "value": "1.0-rc2"
  },
  "source": ":common:capabilities:236"
}
`)

	// Unknown key
	XHTTP(t, reg, "PUT", "/capabilities", `{ "foo": [] }`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_unknown",
  "title": "Unknown capability specified: foo.",
  "subject": "/capabilities",
  "args": {
    "field": "foo"
  },
  "source": ":common:capabilities:255"
}
`)
}

func TestCapabilityAttr(t *testing.T) {
	reg := NewRegistry("TestCapabilityAttr")
	defer PassDeleteReg(t, reg)

	// Verify epoch value
	XHTTP(t, reg, "GET", "/", ``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z"
}
`)

	// Try to clear it all - some can't be totally erased.
	// Notice epoch value changed
	XHTTP(t, reg, "PUT", "/?inline=capabilities", `{
      "capabilities": {
        "available":{
          "capabilities":{"mutable":true},
          "entities":{"mutable":true}
        },
        "flags": ["inline"]
      }
    }`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "available": {
      "capabilities": {
        "mutable": true
      },
      "entities": {
        "mutable": true
      }
    },
    "compatibilities": {},
    "flags": [
      "inline"
    ],
    "formats": [],
    "ignores": [],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "`+SPECVERSION+`"
    ],
    "stickyversions": true,
    "versionmodes": [
      "manual"
    ]
  }
}
`)

	// Setting to nulls
	// notice ?inline is still disabled!
	XHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {
  "available": {"capabilities":{"mutable":true},"entities":{"mutable":true}},
  "flags": null,
  "ignores": null,
  "pagination": null,
  "shortself": null,
  "specversions": null,
  "stickyversions": null,
  "versionmodes": null
}}`, 200,
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}
`)

	// Testing setting everything to the default
	// inline will be enabled due to the update
	XHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {
  "available": {
    "capabilities": { "mutable": true },
    "capabilitiesoffered": { "mutable": false },
    "entities": { "mutable": true },
    "export": { "mutable": false },
    "model": { "mutable": false },
    "modelsource": { "mutable": true }
  },
  "compatibilities": {
    "avro*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "jsonschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "numbers": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "protobuf*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "xmlschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ]
  },
  "flags": [
    "binary", "collections", "doc", "epoch", "filter", "inline", "ignore",
    "setdefaultversionid", "sort", "specversion"
  ],
  "formats": [
    "avro*",
    "jsonschema*",
    "numbers",
    "protobuf*",
    "xmlschema*"
  ],
  "ignores": [
    "capabilities", "defaultversionid", "defaultversionsticky", "epoch",
    "id", "modelsource", "readonly"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [ "`+SPECVERSION+`" ],
  "stickyversions": false,
  "versionmodes": [ "createdat", "manual" ]
}}`, 200,
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "available": {
      "capabilities": {
        "mutable": true
      },
      "capabilitiesoffered": {
        "mutable": false
      },
      "entities": {
        "mutable": true
      },
      "export": {
        "mutable": false
      },
      "model": {
        "mutable": false
      },
      "modelsource": {
        "mutable": true
      }
    },
    "compatibilities": {
      "avro*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "jsonschema*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "numbers": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "protobuf*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "xmlschema*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ]
    },
    "flags": [
      "binary",
      "collections",
      "doc",
      "epoch",
      "filter",
      "ignore",
      "inline",
      "setdefaultversionid",
      "sort",
      "specversion"
    ],
    "formats": [
      "avro*",
      "jsonschema*",
      "numbers",
      "protobuf*",
      "xmlschema*"
    ],
    "ignores": [
      "capabilities",
      "defaultversionid",
      "defaultversionsticky",
      "epoch",
      "id",
      "modelsource",
      "readonly"
    ],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "1.0-rc2"
    ],
    "stickyversions": false,
    "versionmodes": [
      "createdat",
      "manual"
    ]
  }
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "capabilitiesoffered": {
      "mutable": false
    },
    "entities": {
      "mutable": true
    },
    "export": {
      "mutable": false
    },
    "model": {
      "mutable": false
    },
    "modelsource": {
      "mutable": true
    }
  },
  "compatibilities": {
    "avro*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "jsonschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "numbers": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "protobuf*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "xmlschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ]
  },
  "flags": [
    "binary",
    "collections",
    "doc",
    "epoch",
    "filter",
    "ignore",
    "inline",
    "setdefaultversionid",
    "sort",
    "specversion"
  ],
  "formats": [
    "avro*",
    "jsonschema*",
    "numbers",
    "protobuf*",
    "xmlschema*"
  ],
  "ignores": [
    "capabilities",
    "defaultversionid",
    "defaultversionsticky",
    "epoch",
    "id",
    "modelsource",
    "readonly"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": false,
  "versionmodes": [
    "createdat",
    "manual"
  ]
}
`)

	// Setting to minimal
	// inline not enabled
	XHTTP(t, reg, "PUT", "/?inline=capabilities", `{
  "capabilities": {
    "available":{"capabilities":{"mutable":true},"entities":{"mutable":true}},
    "compatibilities": {},
    "flags": [],
    "formats": [],
    "ignores": [],
    "pagination": false,
    "shortself": false,
    "specversions": ["`+SPECVERSION+`"],
    "stickyversions": true,
    "versionmodes": [ "manual" ]
  }
}`, 200,
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}
`)

	// Setting some arrays to [] are an error because we can't do what they
	// asked - which is different from "null"/absent - which means "default"
	XHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities":
	    {"specversions": [] }}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_missing_value",
  "title": "The \"specversions\" capability needs to contain \"1.0-rc2\".",
  "subject": "/capabilities",
  "args": {
    "name": "specversions",
    "value": "1.0-rc2"
  },
  "source": ":common:capabilities:236"
}
`)

	// Unknown key
	XHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities":
	    {"foo": [] }}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_unknown",
  "title": "Unknown capability specified: foo.",
  "subject": "/capabilities",
  "args": {
    "field": "foo"
  },
  "source": ":common:capabilities:255"
}
`)

}

// "binary", "collections", "doc", "epoch", "filter", "inline", "ignore",
// "setdefaultversionid", "sort", "specversion"})

func TestCapabilityFlagsOff(t *testing.T) {
	reg := NewRegistry("TestCapabilityFlags")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)

	XHTTP(t, reg, "PUT", "/capabilities", `{
      "available":{
        "capabilities":{"mutable":true},
        "entities":{"mutable":true},
        "model":{"mutable":false}}}`, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    },
    "model": {
      "mutable": false
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}
`)

	// Create a test file
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Test ?doc
	XHTTP(t, reg, "GET", "/dirs/d1/files?doc", `{}`, 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f1",
    "xid": "/dirs/d1/files/f1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 1
  }
}
`)

	// Test ?filter & ?inline - notice value isn't even analyzed
	XHTTP(t, reg, "GET", "/dirs/d1/files?filter=foo&inline=bar", `{}`, 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f1",
    "xid": "/dirs/d1/files/f1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 1
  }
}
`)

	// Bad epoch should be ignored
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1?epoch=99", `{}`, 204, ``)

	// Test ?setdefaultversionid
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?setdefaultversionid=x", `{
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Test ?specversion
	XHTTP(t, reg, "GET", "/model?specversion=foo", ``, 200, `*`)

	// TODO ignore
}

func TestCapabilityOffered(t *testing.T) {
	reg := NewRegistry("TestCapabilityOffered")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "GET", "/capabilitiesoffered", ``, 200, `{
  "available": {
    "type": "object",
    "attributes": {
      "capabilities": {
        "type": "object",
        "attributes": {
          "mutable": {
            "type": "boolean"
          }
        }
      },
      "capabilitiesoffered": {
        "type": "object",
        "attributes": {
          "mutable": {
            "type": "boolean"
          }
        }
      },
      "entities": {
        "type": "object",
        "attributes": {
          "mutable": {
            "type": "boolean"
          }
        }
      },
      "export": {
        "type": "object",
        "attributes": {
          "mutable": {
            "type": "boolean",
            "enum": [
              false
            ]
          }
        }
      },
      "model": {
        "type": "object",
        "attributes": {
          "mutable": {
            "type": "boolean",
            "enum": [
              false
            ]
          }
        }
      },
      "modelsource": {
        "type": "object",
        "attributes": {
          "mutable": {
            "type": "boolean"
          }
        }
      }
    }
  },
  "compatibilities": {
    "type": "object",
    "attributes": {
      "avro*": {
        "type": "array",
        "enum": [
          "backward",
          "backward_transitive",
          "forward",
          "forward_transitive",
          "full",
          "full_transitive"
        ],
        "item": {
          "type": "string"
        }
      },
      "jsonschema*": {
        "type": "array",
        "enum": [
          "backward",
          "backward_transitive",
          "forward",
          "forward_transitive",
          "full",
          "full_transitive"
        ],
        "item": {
          "type": "string"
        }
      },
      "numbers": {
        "type": "array",
        "enum": [
          "backward",
          "backward_transitive",
          "forward",
          "forward_transitive",
          "full",
          "full_transitive"
        ],
        "item": {
          "type": "string"
        }
      },
      "protobuf*": {
        "type": "array",
        "enum": [
          "backward",
          "backward_transitive",
          "forward",
          "forward_transitive",
          "full",
          "full_transitive"
        ],
        "item": {
          "type": "string"
        }
      },
      "xmlschema*": {
        "type": "array",
        "enum": [
          "backward",
          "backward_transitive",
          "forward",
          "forward_transitive",
          "full",
          "full_transitive"
        ],
        "item": {
          "type": "string"
        }
      }
    }
  },
  "flags": {
    "type": "array",
    "enum": [
      "binary",
      "collections",
      "doc",
      "epoch",
      "filter",
      "ignore",
      "inline",
      "setdefaultversionid",
      "sort",
      "specversion"
    ],
    "item": {
      "type": "string"
    }
  },
  "formats": {
    "type": "array",
    "enum": [
      "avro*",
      "jsonschema*",
      "numbers",
      "protobuf*",
      "xmlschema*"
    ],
    "item": {
      "type": "string"
    }
  },
  "ignores": {
    "type": "array",
    "enum": [
      "capabilities",
      "defaultversionid",
      "defaultversionsticky",
      "epoch",
      "id",
      "modelsource",
      "readonly"
    ],
    "item": {
      "type": "string"
    }
  },
  "pagination": {
    "type": "boolean",
    "enum": [
      false
    ]
  },
  "shortself": {
    "type": "boolean",
    "enum": [
      false
    ]
  },
  "specversions": {
    "type": "array",
    "enum": [
      "`+SPECVERSION+`"
    ],
    "item": {
      "type": "string"
    }
  },
  "stickyversions": {
    "type": "boolean",
    "enum": [
      false,
      true
    ]
  },
  "versionmodes": {
    "type": "array",
    "enum": [
      "createdat",
      "manual"
    ],
    "item": {
      "type": "string"
    }
  }
}
`)
}

func TestCapabilityAvailable(t *testing.T) {
	reg := NewRegistry("TestCapabilityAPIs")
	defer PassDeleteReg(t, reg)

	// Try to clear it all
	XHTTP(t, reg, "PUT", "/capabilities", `{
      "available":{
        "entities":{
          "mutable":false
        }
      }
    }`, 200,
		`{
  "available": {
    "entities": {
      "mutable": false
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_available",
  "title": "The requested data (/capabilities) is not available.",
  "subject": "/capabilities",
  "source": "b1fcff68b7f8:registry:httpStuff:655"
}
`)
	XHTTP(t, reg, "GET", "/capabilitiesoffered", ``, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_available",
  "title": "The requested data (/capabilitiesoffered) is not available.",
  "subject": "/capabilitiesoffered",
  "source": "b1fcff68b7f8:registry:httpStuff:662"
}
`)
	XHTTP(t, reg, "GET", "/export", ``, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_available",
  "title": "The requested data (/export) is not available.",
  "subject": "/export",
  "source": "b1fcff68b7f8:registry:httpStuff:669"
}
`)
	XHTTP(t, reg, "GET", "/model", ``, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_available",
  "title": "The requested data (/model) is not available.",
  "subject": "/model",
  "source": "b1fcff68b7f8:registry:httpStuff:676"
}
`)
	XHTTP(t, reg, "GET", "/modelsource", ``, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_available",
  "title": "The requested data (/modelsource) is not available.",
  "subject": "/modelsource",
  "source": "b1fcff68b7f8:registry:httpStuff:683"
}
`)

	// Now test mutability
	XHTTP(t, reg, "PUT", "/capabilities", `{}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_available",
  "title": "The requested data (/capabilities) is not available.",
  "subject": "/capabilities",
  "source": "b1fcff68b7f8:registry:httpStuff:910"
}
`)

	XHTTP(t, reg, "PUT", "/", `{}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_available",
  "title": "The requested data (/) is not available.",
  "subject": "/",
  "source": "b1fcff68b7f8:registry:httpStuff:930"
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", ``, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_available",
  "title": "The requested data (/modelsource) is not available.",
  "subject": "/modelsource",
  "source": "b1fcff68b7f8:registry:httpStuff:924"
}
`)

	reg.Capabilities.SetAvailable("capabilities", true)
	reg.Capabilities.SetAvailable("entities", true)
	XNoErr(t, reg.SaveCapabilities())
	XNoErr(t, reg.Refresh(registry.FOR_WRITE))

	// Open /capabilities back up
	XHTTP(t, reg, "PUT", "/capabilities",
		`{"available":{
            "capabilities":{"mutable":true},
            "entities":{"mutable":true}
          }}`, 200, `*`)

	XHTTP(t, reg, "PUT", "/?inline=capabilities",
		`{"capabilities":{
            "available":{
              "capabilities":{"mutable":true},
              "entities":{"mutable":true}
            }}}`,
		200, `*`)

	XHTTP(t, reg, "PUT", "/capabilities",
		`{"available":{
            "capabilities":{"mutable":true},
            "export":{"mutable":false}
          }}`,
		200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    },
    "export": {
      "mutable": false
    }
  },
  "compatibilities": {},
  "flags": [],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}
`)

	// XHTTP(t, reg, "GET", "/capabilities", ``, 200, "*")
	XHTTP(t, reg, "GET", "/export", ``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityAPIs",
  "self": "#/",
  "xid": "/",
  "epoch": 4,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "available": {
      "capabilities": {
        "mutable": true
      },
      "entities": {
        "mutable": true
      },
      "export": {
        "mutable": false
      }
    },
    "compatibilities": {},
    "flags": [],
    "formats": [],
    "ignores": [],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "`+SPECVERSION+`"
    ],
    "stickyversions": true,
    "versionmodes": [
      "manual"
    ]
  }
}
`)

	XHTTP(t, reg, "GET", "/model", ``, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_available",
  "title": "The requested data (/model) is not available.",
  "subject": "/model",
  "source": "b1fcff68b7f8:registry:httpStuff:676"
}
`)
	XHTTP(t, reg, "GET", "/modelsource", ``, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_available",
  "title": "The requested data (/modelsource) is not available.",
  "subject": "/modelsource",
  "source": "b1fcff68b7f8:registry:httpStuff:683"
}
`)

	// Some errors
	XHTTP(t, reg, "PUT", "/capabilities", `{"available":{"foo":{}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_error",
  "title": "There was an error in the capabilities provided: Unknown \"available\" value: foo.",
  "subject": "/capabilities",
  "args": {
    "error_detail": "Unknown \"available\" value: foo"
  },
  "source": "b1fcff68b7f8:common:capabilities:319"
}
`)
	XHTTP(t, reg, "PUT", "/capabilities", `{"available":{"/foo":{}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_error",
  "title": "There was an error in the capabilities provided: Unknown \"available\" value: /foo.",
  "subject": "/capabilities",
  "args": {
    "error_detail": "Unknown \"available\" value: /foo"
  },
  "source": "b1fcff68b7f8:common:capabilities:319"
}
`)
	XHTTP(t, reg, "PUT", "/capabilities", `{
      "available":{
        "capabilities":{"mutable":true},
        "export":{"mutable":true}
      }
    }`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_error",
  "title": "There was an error in the capabilities provided: \"available\" value \"export\" is not allowed to be mutable.",
  "subject": "/capabilities",
  "args": {
    "error_detail": "\"available\" value \"export\" is not allowed to be mutable"
  },
  "source": "b1fcff68b7f8:common:capabilities:324"
}
`)

	XNoErr(t, reg.Refresh(registry.FOR_WRITE))
	reg.Capabilities.SetAvailable("capabilities", true)
	reg.Capabilities.SetAvailable("entities", true)
	XNoErr(t, reg.SaveCapabilities())
	XNoErr(t, reg.Refresh(registry.FOR_WRITE))

	// Reset to default

	// despite "flags" being empty prior to this command, since capabilities
	// are being reset to their default values, "flags" should immediately
	// allow "inline" to be enabled for processing of the response
	XHTTP(t, reg, "PATCH", "/?inline=capabilities", `{"capabilities":null}`,
		200, `{
  "specversion": "1.0-rc2",
  "registryid": "TestCapabilityAPIs",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 6,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "available": {
      "capabilities": {
        "mutable": true
      },
      "capabilitiesoffered": {
        "mutable": false
      },
      "entities": {
        "mutable": true
      },
      "export": {
        "mutable": false
      },
      "model": {
        "mutable": false
      },
      "modelsource": {
        "mutable": true
      }
    },
    "compatibilities": {
      "avro*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "jsonschema*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "numbers": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "protobuf*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "xmlschema*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ]
    },
    "flags": [
      "binary",
      "collections",
      "doc",
      "epoch",
      "filter",
      "ignore",
      "inline",
      "setdefaultversionid",
      "sort",
      "specversion"
    ],
    "formats": [
      "avro*",
      "jsonschema*",
      "numbers",
      "protobuf*",
      "xmlschema*"
    ],
    "ignores": [
      "capabilities",
      "defaultversionid",
      "defaultversionsticky",
      "epoch",
      "id",
      "modelsource",
      "readonly"
    ],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "1.0-rc2"
    ],
    "stickyversions": true,
    "versionmodes": [
      "createdat",
      "manual"
    ]
  }
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200,
		`{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "capabilitiesoffered": {
      "mutable": false
    },
    "entities": {
      "mutable": true
    },
    "export": {
      "mutable": false
    },
    "model": {
      "mutable": false
    },
    "modelsource": {
      "mutable": true
    }
  },
  "compatibilities": {
    "avro*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "jsonschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "numbers": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "protobuf*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "xmlschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ]
  },
  "flags": [
    "binary",
    "collections",
    "doc",
    "epoch",
    "filter",
    "ignore",
    "inline",
    "setdefaultversionid",
    "sort",
    "specversion"
  ],
  "formats": [
    "avro*",
    "jsonschema*",
    "numbers",
    "protobuf*",
    "xmlschema*"
  ],
  "ignores": [
    "capabilities",
    "defaultversionid",
    "defaultversionsticky",
    "epoch",
    "id",
    "modelsource",
    "readonly"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "1.0-rc2"
  ],
  "stickyversions": true,
  "versionmodes": [
    "createdat",
    "manual"
  ]
}
`)

}

func TestCapabilityPatch(t *testing.T) {
	reg := NewRegistry("TestCapabilityPatch")
	defer PassDeleteReg(t, reg)

	// Try to clear it all
	XHTTP(t, reg, "PATCH", "/capabilities", `{
      "flags": ["inline"],
      "stickyversions": false
    }`, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "capabilitiesoffered": {
      "mutable": false
    },
    "entities": {
      "mutable": true
    },
    "export": {
      "mutable": false
    },
    "model": {
      "mutable": false
    },
    "modelsource": {
      "mutable": true
    }
  },
  "compatibilities": {
    "avro*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "jsonschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "numbers": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "protobuf*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ],
    "xmlschema*": [
      "backward",
      "backward_transitive",
      "forward",
      "forward_transitive",
      "full",
      "full_transitive"
    ]
  },
  "flags": [
    "inline"
  ],
  "formats": [
    "avro*",
    "jsonschema*",
    "numbers",
    "protobuf*",
    "xmlschema*"
  ],
  "ignores": [
    "capabilities",
    "defaultversionid",
    "defaultversionsticky",
    "epoch",
    "id",
    "modelsource",
    "readonly"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": false,
  "versionmodes": [
    "createdat",
    "manual"
  ]
}
`)

	XHTTP(t, reg, "PATCH", "/", `{
  "description": "test"
}`, 200, `{
  "specversion": "1.0-rc2",
  "registryid": "TestCapabilityPatch",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "description": "test",
  "createdat": "2026-05-16T20:31:36.518315881Z",
  "modifiedat": "2026-05-16T20:31:36.540563891Z"
}
`)

	XHTTP(t, reg, "PATCH", "/?inline=capabilities", `{
  "capabilities": {
    "flags": [ "inline", "filter" ],
    "stickyversions": true
}
}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityPatch",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "description": "test",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "available": {
      "entities": {
        "mutable": true
      }
    },
    "compatibilities": {},
    "flags": [
      "filter",
      "inline"
    ],
    "formats": [],
    "ignores": [],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "`+SPECVERSION+`"
    ],
    "stickyversions": true,
    "versionmodes": [
      "manual"
    ]
  }
}
`)

}

func TestCapabilityPost(t *testing.T) {
	reg := NewRegistry("TestCapabilityPost")
	defer PassDeleteReg(t, reg)

	// Try to clear it all
	XHTTP(t, reg, "POST", "/capabilities", `{}`, 405,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#action_not_supported",
  "title": "The specified action (POST) is not supported for: /capabilities.",
  "subject": "/capabilities",
  "args": {
    "action": "POST"
  },
  "source": ":registry:httpStuff:2540"
}
`)
	XHTTP(t, reg, "POST", "/", `{
  "capabilities": {}
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#groups_only",
  "title": "Attribute \"capabilities\" is invalid. Only Group types are allowed to be specified on this request: /.",
  "subject": "/",
  "args": {
    "name": "capabilities"
  },
  "source": ":registry:httpStuff:1927"
}
`)

}
