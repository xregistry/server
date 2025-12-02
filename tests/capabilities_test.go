package tests

import (
	"encoding/json"
	"testing"

	. "github.com/xregistry/server/common"
)

func TestCapabilitySimple(t *testing.T) {
	reg := NewRegistry("TestCapabilitySimple")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "GET", "/capabilities/foo", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#api_not_found",
  "title": "The specified API is not supported: /capabilities/foo.",
  "subject": "/capabilities/foo",
  "source": ":registry:httpStuff:1258"
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "apis": [
    "/capabilities",
    "/capabilitiesoffered",
    "/export",
    "/model",
    "/modelsource"
  ],
  "flags": [
    "binary",
    "collections",
    "doc",
    "epoch",
    "filter",
    "ignoredefaultversionid",
    "ignoredefaultversionsticky",
    "ignoreepoch",
    "ignorereadonly",
    "inline",
    "setdefaultversionid",
    "sort",
    "specversion"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
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
    "apis": [
      "/capabilities",
      "/capabilitiesoffered",
      "/export",
      "/model",
      "/modelsource"
    ],
    "flags": [
      "binary",
      "collections",
      "doc",
      "epoch",
      "filter",
      "ignoredefaultversionid",
      "ignoredefaultversionsticky",
      "ignoreepoch",
      "ignorereadonly",
      "inline",
      "setdefaultversionid",
      "sort",
      "specversion"
    ],
    "mutable": [
      "capabilities",
      "entities",
      "model"
    ],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "`+SPECVERSION+`"
    ],
    "stickyversions": true
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
  "apis": [],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "` + SPECVERSION + `"
  ],
  "stickyversions": true
}`,
		},
		{
			Name: "full mutable",
			Cap:  `{"mutable":["entities","model","capabilities"]}`,
			Exp: `{
  "apis": [],
  "flags": [],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "` + SPECVERSION + `"
  ],
  "stickyversions": true
}`,
		},
		{
			Name: "dup mutable",
			Cap:  `{"mutable":["entities","model","entities","capabilities"]}`,
			Exp: `{
  "apis": [],
  "flags": [],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "` + SPECVERSION + `"
  ],
  "stickyversions": true
}`,
		},
		{
			Name: "star mutable",
			Cap:  `{"mutable":["*"]}`,
			Exp: `{
  "apis": [],
  "flags": [],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "` + SPECVERSION + `"
  ],
  "stickyversions": true
}`,
		},
		{
			Name: "mutable empty",
			Cap:  `{"mutable":[]}`,
			Exp: `{
  "apis": [],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "` + SPECVERSION + `"
  ],
  "stickyversions": true
}`,
		},
		{
			Name: "star mutable-bad",
			Cap:  `{"mutable":["model","*"]}`,
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_wildcard",
  "title": "When \"mutable\" includes a value of \"*\" then no other values are allowed.",
  "subject": "/capabilities",
  "args": {
    "field": "mutable"
  },
  "source": ":common:capabilities:157"
}`,
		},
		{
			Name: "bad mutable-1",
			Cap:  `{"mutable":["xx"]}`,
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_value",
  "title": "Invalid value (xx) specified for capability \"mutable\". Allowable values include: capabilities,entities,model.",
  "subject": "/capabilities",
  "args": {
    "field": "mutable",
    "list": "capabilities,entities,model",
    "value": "xx"
  },
  "source": ":common:capabilities:178"
}`,
		},
		{
			Name: "bad mutable-2",
			Cap:  `{"mutable":["model", "xx"]}`,
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_value",
  "title": "Invalid value (xx) specified for capability \"mutable\". Allowable values include: capabilities,entities,model.",
  "subject": "/capabilities",
  "args": {
    "field": "mutable",
    "list": "capabilities,entities,model",
    "value": "xx"
  },
  "source": ":common:capabilities:178"
}`,
		},
		{
			Name: "bad mutable-3",
			Cap:  `{"mutable":["aa", "model"]}`,
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_error",
  "title": "There was an error in the capabilities provided: unknown \"mutable\" value: \"aa\".",
  "subject": "/capabilities",
  "args": {
    "error_detail": "unknown \"mutable\" value: \"aa\""
  },
  "source": ":common:capabilities:188"
}`,
		},
		{
			Name: "bad mutable-4",
			Cap:  `{"mutable":["entities", "ff", "model"]}`,
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_value",
  "title": "Invalid value (ff) specified for capability \"mutable\". Allowable values include: capabilities,entities,model.",
  "subject": "/capabilities",
  "args": {
    "field": "mutable",
    "list": "capabilities,entities,model",
    "value": "ff"
  },
  "source": ":common:capabilities:178"
}`,
		},

		{
			Name: "missing specversion",
			Cap:  `{"specversions":[]}`,
			Exp: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_missing_specversion",
  "title": "The \"specversions\" capability needs to contain \"1.0-rc2\".",
  "subject": "/capabilities",
  "args": {
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
  "apis": [
    "/capabilities",
    "/capabilitiesoffered",
    "/export",
    "/model",
    "/modelsource"
  ],
  "flags": [
    "binary",
    "collections",
    "doc",
    "epoch",
    "filter",
    "ignoredefaultversionid",
    "ignoredefaultversionsticky",
    "ignoreepoch",
    "ignorereadonly",
    "inline",
    "setdefaultversionid",
    "sort",
    "specversion"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
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
  "apis": [],
  "flags": [
    "inline"
  ],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
}
`)

	// Make sure it's turned off, but turn it on for the rest of the
	// tests
	XHTTP(t, reg, "GET", "/capabilities", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#api_not_found",
  "title": "The specified API is not supported: /capabilities.",
  "subject": "/capabilities",
  "source": ":registry:httpStuff:1589"
}
`)

	XHTTP(t, reg, "PUT", "/?inline=capabilities",
		`{"capabilities":{"apis":["/capabilities"]}}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityPath",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "apis": [
      "/capabilities"
    ],
    "flags": [],
    "mutable": [],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "`+SPECVERSION+`"
    ],
    "stickyversions": true
  }
}
`)

	// Make sure the Registry epoch changed
	XHTTP(t, reg, "GET", "/", ``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityPath",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "apis": [
    "/capabilities"
  ],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
}
`)

	// Setting to nulls
	XHTTP(t, reg, "PUT", "/capabilities", `{
  "apis": ["/capabilities"],
  "flags": null,
  "mutable": null,
  "pagination": null,
  "shortself": null,
  "specversions": null
}`, 200,
		`{
  "apis": [
    "/capabilities"
  ],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "apis": [
    "/capabilities"
  ],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
}
`)

	// Testing setting everything to the default
	XHTTP(t, reg, "PUT", "/capabilities", `{
  "apis": [
    "/capabilities", "/capabilitiesoffered", "/export", "/model", "/modelsource"
  ],
  "flags": [
    "binary", "collections", "doc", "epoch", "filter", "inline",
    "ignoredefaultversionid", "ignoredefaultversionsticky", "ignoreepoch",
    "ignorereadonly", "setdefaultversionid", "sort",
    "specversion"
  ],
  "mutable": [ "capabilities", "entities", "model" ],
  "pagination": false,
  "shortself": false,
  "specversions": [ "`+SPECVERSION+`" ],
  "stickyversions": true
}`, 200,
		`{
  "apis": [
    "/capabilities",
    "/capabilitiesoffered",
    "/export",
    "/model",
    "/modelsource"
  ],
  "flags": [
    "binary",
    "collections",
    "doc",
    "epoch",
    "filter",
    "ignoredefaultversionid",
    "ignoredefaultversionsticky",
    "ignoreepoch",
    "ignorereadonly",
    "inline",
    "setdefaultversionid",
    "sort",
    "specversion"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "apis": [
    "/capabilities",
    "/capabilitiesoffered",
    "/export",
    "/model",
    "/modelsource"
  ],
  "flags": [
    "binary",
    "collections",
    "doc",
    "epoch",
    "filter",
    "ignoredefaultversionid",
    "ignoredefaultversionsticky",
    "ignoreepoch",
    "ignorereadonly",
    "inline",
    "setdefaultversionid",
    "sort",
    "specversion"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
}
`)

	// Setting to minimal
	XHTTP(t, reg, "PUT", "/capabilities", `{"apis":["/capabilities"]}`,
		200, `{
  "apis": [
    "/capabilities"
  ],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "apis": [
    "/capabilities"
  ],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
}
`)

	// Test some bools
	XHTTP(t, reg, "PUT", "/capabilities", `{
    "apis":["/capabilities"],
	"pagination": false,
	"shortself": false,
    "stickyversions": false
}`, 200, `{
  "apis": [
    "/capabilities"
  ],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": false
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
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_missing_specversion",
  "title": "The \"specversions\" capability needs to contain \"1.0-rc2\".",
  "subject": "/capabilities",
  "args": {
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
      "capabilities": {"apis":["/capabilities"]} }`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "apis": [
      "/capabilities"
    ],
    "flags": [],
    "mutable": [],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "`+SPECVERSION+`"
    ],
    "stickyversions": true
  }
}
`)

	// Setting to nulls
	// notice ?inline is still disabled!
	XHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {
  "apis": ["/capabilities"],
  "flags": null,
  "mutable": null,
  "pagination": null,
  "shortself": null,
  "specversions": null,
  "stickyversions": null
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
  "apis": [
    "/capabilities"
  ],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
}
`)

	// Testing setting everything to the default
	// inline still disabled
	XHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {
  "apis": ["/export", "/model", "/modelsource", "/capabilities",
    "/capabilitiesoffered"],
  "flags": [
    "binary", "collections", "doc", "epoch", "filter", "inline",
    "ignoredefaultversionid", "ignoredefaultversionsticky", "ignoreepoch",
    "ignorereadonly", "setdefaultversionid", "sort",
    "specversion"
  ],
  "mutable": [ "capabilities", "entities", "model" ],
  "pagination": false,
  "shortself": false,
  "specversions": [ "`+SPECVERSION+`" ],
  "stickyversions": false
}}`, 200,
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "apis": [
    "/capabilities",
    "/capabilitiesoffered",
    "/export",
    "/model",
    "/modelsource"
  ],
  "flags": [
    "binary",
    "collections",
    "doc",
    "epoch",
    "filter",
    "ignoredefaultversionid",
    "ignoredefaultversionsticky",
    "ignoreepoch",
    "ignorereadonly",
    "inline",
    "setdefaultversionid",
    "sort",
    "specversion"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": false
}
`)

	// Setting to minimal
	// inline still enabled
	XHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {
  "apis":["/capabilities"],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": ["`+SPECVERSION+`"],
  "stickyversions": true
}}`, 200,
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "apis": [
      "/capabilities"
    ],
    "flags": [],
    "mutable": [],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "`+SPECVERSION+`"
    ],
    "stickyversions": true
  }
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "apis": [
    "/capabilities"
  ],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
}
`)

	// Setting some arrays to [] are an error because we can't do what they
	// asked - which is different from "null"/absent - which means "default"
	XHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities":
	    {"specversions": [] }}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_missing_specversion",
  "title": "The \"specversions\" capability needs to contain \"1.0-rc2\".",
  "subject": "/capabilities",
  "args": {
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

// "binary", "collections", "doc", "epoch", "filter", "inline",
// "ignoredefaultversionid", "ignoredefaultversionsticky",
// "ignoreepoch", "ignorereadonly", "setdefaultversionid",
// "sort", "specversion"})

func TestCapabilityFlagsOff(t *testing.T) {
	reg := NewRegistry("TestCapabilityFlags")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)

	XHTTP(t, reg, "PUT", "/capabilities", `{
      "apis":["/capabilities","/model"],"mutable":["*"]}`, 200, `{
  "apis": [
    "/capabilities",
    "/model"
  ],
  "flags": [],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
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

	// TODO ignoredefaultversionid, ignoredefaultversionsticky,
	// ignoreepoch, ignorereadonly
}

func TestCapabilityOffered(t *testing.T) {
	reg := NewRegistry("TestCapabilityOffered")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "GET", "/capabilitiesoffered", ``, 200, `{
  "apis": {
    "type": "array",
    "item": {
      "type": "string"
    },
    "enum": [
      "/capabilities",
      "/capabilitiesoffered",
      "/export",
      "/model",
      "/modelsource"
    ]
  },
  "flags": {
    "type": "array",
    "item": {
      "type": "string"
    },
    "enum": [
      "binary",
      "collections",
      "doc",
      "epoch",
      "filter",
      "ignoredefaultversionid",
      "ignoredefaultversionsticky",
      "ignoreepoch",
      "ignorereadonly",
      "inline",
      "setdefaultversionid",
      "sort",
      "specversion"
    ]
  },
  "mutable": {
    "type": "string",
    "enum": [
      "capabilities",
      "entities",
      "model"
    ]
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
    "type": "string",
    "enum": [
      "`+SPECVERSION+`"
    ]
  },
  "stickyversions": {
    "type": "boolean",
    "enum": [
      false,
      true
    ]
  }
}
`)
}

func TestCapabilityAPIs(t *testing.T) {
	reg := NewRegistry("TestCapabilityAPIs")
	defer PassDeleteReg(t, reg)

	// Try to clear it all
	XHTTP(t, reg, "PUT", "/capabilities", `{}`, 200,
		`{
  "apis": [],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#api_not_found",
  "title": "The specified API is not supported: /capabilities.",
  "subject": "/capabilities",
  "source": ":registry:httpStuff:1589"
}
`)
	XHTTP(t, reg, "GET", "/capabilitiesoffered", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#api_not_found",
  "title": "The specified API is not supported: /capabilitiesoffered.",
  "subject": "/capabilitiesoffered",
  "source": ":registry:httpStuff:1596"
}
`)
	XHTTP(t, reg, "GET", "/export", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#api_not_found",
  "title": "The specified API is not supported: /export.",
  "subject": "/export",
  "source": ":registry:httpStuff:1603"
}
`)
	XHTTP(t, reg, "GET", "/model", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#api_not_found",
  "title": "The specified API is not supported: /model.",
  "subject": "/model",
  "source": ":registry:httpStuff:1575"
}
`)
	XHTTP(t, reg, "GET", "/modelsource", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#api_not_found",
  "title": "The specified API is not supported: /modelsource.",
  "subject": "/modelsource",
  "source": ":registry:httpStuff:1582"
}
`)

	// Open /capabilities back up
	XHTTP(t, reg, "PUT", "/?inline=capabilities",
		`{"capabilities":{"apis":["/capabilities"]}}`, 200, `*`)

	XHTTP(t, reg, "PUT", "/capabilities", `{
      "apis":["/capabilities","/export"]}`, 200, `{
  "apis": [
    "/capabilities",
    "/export"
  ],
  "flags": [],
  "mutable": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": true
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200, "*")
	XHTTP(t, reg, "GET", "/export", ``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCapabilityAPIs",
  "self": "#/",
  "xid": "/",
  "epoch": 4,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "apis": [
      "/capabilities",
      "/export"
    ],
    "flags": [],
    "mutable": [],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "`+SPECVERSION+`"
    ],
    "stickyversions": true
  },
  "modelsource": {}
}
`)
	XHTTP(t, reg, "GET", "/model", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#api_not_found",
  "title": "The specified API is not supported: /model.",
  "subject": "/model",
  "source": ":registry:httpStuff:1575"
}
`)
	XHTTP(t, reg, "GET", "/modelsource", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#api_not_found",
  "title": "The specified API is not supported: /modelsource.",
  "subject": "/modelsource",
  "source": ":registry:httpStuff:1582"
}
`)

	// Some errors
	XHTTP(t, reg, "PUT", "/capabilities", `{"apis":["/foo"]}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_value",
  "title": "Invalid value (/foo) specified for capability \"apis\". Allowable values include: /capabilities,/capabilitiesoffered,/export,/model,/modelsource.",
  "subject": "/capabilities",
  "args": {
    "field": "apis",
    "list": "/capabilities,/capabilitiesoffered,/export,/model,/modelsource",
    "value": "/foo"
  },
  "source": ":common:capabilities:178"
}
`)
	XHTTP(t, reg, "PUT", "/capabilities", `{"apis":["foo"]}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_value",
  "title": "Invalid value (foo) specified for capability \"apis\". Allowable values include: /capabilities,/capabilitiesoffered,/export,/model,/modelsource.",
  "subject": "/capabilities",
  "args": {
    "field": "apis",
    "list": "/capabilities,/capabilitiesoffered,/export,/model,/modelsource",
    "value": "foo"
  },
  "source": ":common:capabilities:178"
}
`)
	XHTTP(t, reg, "PUT", "/capabilities", `{"apis":["export"]}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#capability_value",
  "title": "Invalid value (export) specified for capability \"apis\". Allowable values include: /capabilities,/capabilitiesoffered,/export,/model,/modelsource.",
  "subject": "/capabilities",
  "args": {
    "field": "apis",
    "list": "/capabilities,/capabilitiesoffered,/export,/model,/modelsource",
    "value": "export"
  },
  "source": ":common:capabilities:178"
}
`)

	// Reset to default

	// notice that the ?inline will be ignored because it's a valid
	// flag before the PATCH and won't take effect until AFTER this API
	// is complete
	XHTTP(t, reg, "PATCH", "/?inline=capabilities", `{"capabilities":null}`,
		200, `{
  "specversion": "1.0-rc2",
  "registryid": "TestCapabilityAPIs",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	XHTTP(t, reg, "GET", "/capabilities", ``, 200,
		`{
  "apis": [
    "/capabilities",
    "/capabilitiesoffered",
    "/export",
    "/model",
    "/modelsource"
  ],
  "flags": [
    "binary",
    "collections",
    "doc",
    "epoch",
    "filter",
    "ignoredefaultversionid",
    "ignoredefaultversionsticky",
    "ignoreepoch",
    "ignorereadonly",
    "inline",
    "setdefaultversionid",
    "sort",
    "specversion"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "1.0-rc2"
  ],
  "stickyversions": true
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
  "apis": [
    "/capabilities",
    "/capabilitiesoffered",
    "/export",
    "/model",
    "/modelsource"
  ],
  "flags": [
    "inline"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "`+SPECVERSION+`"
  ],
  "stickyversions": false
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
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "apis": [],
    "flags": [
      "filter",
      "inline"
    ],
    "mutable": [],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "`+SPECVERSION+`"
    ],
    "stickyversions": true
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
