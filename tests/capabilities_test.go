package tests

import (
	"encoding/json"
	"testing"

	"github.com/xregistry/server/registry"
)

func TestCapabilitySimple(t *testing.T) {
	reg := NewRegistry("TestCapabilitySimple")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "GET", "/capabilities/foo", ``, 404,
		"\"capabilities/foo\" not found\n")

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [
    "collections",
    "doc",
    "epoch",
    "filter",
    "inline",
    "nodefaultversionid",
    "nodefaultversionsticky",
    "noepoch",
    "noreadonly",
    "offered",
    "schema",
    "setdefaultversionid",
    "specversion"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	xHTTP(t, reg, "GET", "?inline=capabilities", ``, 200, `{
  "specversion": "1.0-rc1",
  "registryid": "TestCapabilitySimple",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

  "capabilities": {
    "flags": [
      "collections",
      "doc",
      "epoch",
      "filter",
      "inline",
      "nodefaultversionid",
      "nodefaultversionsticky",
      "noepoch",
      "noreadonly",
      "offered",
      "schema",
      "setdefaultversionid",
      "specversion"
    ],
    "mutable": [
      "capabilities",
      "entities",
      "model"
    ],
    "pagination": false,
    "schemas": [
      "xregistry-json/1.0-rc1"
    ],
    "shortself": false,
    "specversions": [
      "`+registry.SPECVERSION+`"
    ],
    "sticky": true
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
  "pagination": false,
  "schemas": [
    "xregistry-json/1.0-rc1"
  ],
  "shortself": false,
  "specversions": [
    "` + registry.SPECVERSION + `"
  ],
  "sticky": true
}`,
		},
		{
			Name: "full mutable",
			Cap:  `{"mutable":["entities","model","capabilities"]}`,
			Exp: `{
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json/` + registry.SPECVERSION + `"
  ],
  "shortself": false,
  "specversions": [
    "` + registry.SPECVERSION + `"
  ],
  "sticky": true
}`,
		},
		{
			Name: "dup mutable",
			Cap:  `{"mutable":["entities","model","entities","capabilities"]}`,
			Exp: `{
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json/` + registry.SPECVERSION + `"
  ],
  "shortself": false,
  "specversions": [
    "` + registry.SPECVERSION + `"
  ],
  "sticky": true
}`,
		},
		{
			Name: "star mutable",
			Cap:  `{"mutable":["*"]}`,
			Exp: `{
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json/` + registry.SPECVERSION + `"
  ],
  "shortself": false,
  "specversions": [
    "` + registry.SPECVERSION + `"
  ],
  "sticky": true
}`,
		},
		{
			Name: "mutable empty",
			Cap:  `{"mutable":[]}`,
			Exp: `{
  "pagination": false,
  "schemas": [
    "xregistry-json/` + registry.SPECVERSION + `"
  ],
  "shortself": false,
  "specversions": [
    "` + registry.SPECVERSION + `"
  ],
  "sticky": true
}`,
		},
		{
			Name: "star mutable-bad",
			Cap:  `{"mutable":["model","*"]}`,
			Exp:  `"*" must be the only value specified for "mutable"`,
		},
		{
			Name: "bad mutable-1",
			Cap:  `{"mutable":["xx"]}`,
			Exp:  `Unknown "mutable" value: "xx"`,
		},
		{
			Name: "bad mutable-2",
			Cap:  `{"mutable":["model", "xx"]}`,
			Exp:  `Unknown "mutable" value: "xx"`,
		},
		{
			Name: "bad mutable-3",
			Cap:  `{"mutable":["aa", "model"]}`,
			Exp:  `Unknown "mutable" value: "aa"`,
		},
		{
			Name: "bad mutable-4",
			Cap:  `{"mutable":["entities", "ff", "model"]}`,
			Exp:  `Unknown "mutable" value: "ff"`,
		},

		{
			Name: "missing schema",
			Cap:  `{"schemas":[]}`,
			Exp:  `"schemas" must contain "xRegistry-json/` + registry.SPECVERSION + `"`,
		},
		{
			Name: "missing specversion",
			Cap:  `{"specversions":[]}`,
			Exp:  `"specversions" must contain "` + registry.SPECVERSION + `"`,
		},

		{
			Name: "extra key",
			Cap:  `{"pagination": true, "bad": true}`,
			Exp:  `Unknown capability: "bad"`,
		},
	}

	for _, test := range tests {
		c, err := registry.ParseCapabilitiesJSON([]byte(test.Cap))
		if err == nil {
			err = c.Validate()
		}
		res := ""
		if err != nil {
			res = err.Error()
		} else {
			buf, _ := json.MarshalIndent(c, "", "  ")
			res = string(buf)
		}
		xCheckEqual(t, test.Name, res, test.Exp)
	}
}

func TestCapabilityPath(t *testing.T) {
	reg := NewRegistry("TestCapabilityPath")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [
    "collections",
    "doc",
    "epoch",
    "filter",
    "inline",
    "nodefaultversionid",
    "nodefaultversionsticky",
    "noepoch",
    "noreadonly",
    "offered",
    "schema",
    "setdefaultversionid",
    "specversion"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	// Verify current epoch value
	xHTTP(t, reg, "GET", "/", ``, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestCapabilityPath",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z"
}
`)

	// Try to clear it all - some can't be totally erased
	xHTTP(t, reg, "PUT", "/capabilities", `{}`, 200,
		`{
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	// Make sure the Registry epoch changed
	xHTTP(t, reg, "GET", "/", ``, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestCapabilityPath",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	// Setting to nulls
	xHTTP(t, reg, "PUT", "/capabilities", `{
  "flags": null,
  "mutable": null,
  "pagination": null,
  "schemas": null,
  "shortself": null,
  "specversions": null
}`, 200,
		`{
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	// Testing setting everything to the default
	xHTTP(t, reg, "PUT", "/capabilities", `{
  "flags": [
    "collections", "doc", "epoch", "filter", "inline", "nodefaultversionid",
    "nodefaultversionsticky", "noepoch", "noreadonly", "offered", "schema",
	"setdefaultversionid", "specversion"
  ],
  "mutable": [ "capabilities", "entities", "model" ],
  "pagination": false,
  "schemas": [ "xregistry-json/`+registry.SPECVERSION+`" ],
  "shortself": false,
  "specversions": [ "`+registry.SPECVERSION+`" ],
  "sticky": true
}`, 200,
		`{
  "flags": [
    "collections",
    "doc",
    "epoch",
    "filter",
    "inline",
    "nodefaultversionid",
    "nodefaultversionsticky",
    "noepoch",
    "noreadonly",
    "offered",
    "schema",
    "setdefaultversionid",
    "specversion"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [
    "collections",
    "doc",
    "epoch",
    "filter",
    "inline",
    "nodefaultversionid",
    "nodefaultversionsticky",
    "noepoch",
    "noreadonly",
    "offered",
    "schema",
    "setdefaultversionid",
    "specversion"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	// Setting to minimal
	xHTTP(t, reg, "PUT", "/capabilities", `{
}`, 200,
		`{
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	// Test schemas
	xHTTP(t, reg, "PUT", "/capabilities", `{
	"schemas": ["xregistry-json"]
}`, 200, `{
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	// Test some bools
	xHTTP(t, reg, "PUT", "/capabilities", `{
	"pagination": false,
	"shortself": false,
    "sticky": false
}`, 200, `{
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": false
}
`)

	xHTTP(t, reg, "PUT", "/capabilities", `{"pagination":true}`, 400,
		`"pagination" must be "false"`+"\n")

	xHTTP(t, reg, "PUT", "/capabilities", `{"shortself":true}`, 400,
		`"shortself" must be "false"`+"\n")

	// Setting some arrays to [] are an error because we can't do what they
	// asked - which is different from "null"/absent - which means "default"
	xHTTP(t, reg, "PUT", "/capabilities", `{ "schemas": [] }`,
		400, `"schemas" must contain "xRegistry-json/`+registry.SPECVERSION+`"`+"\n")

	// Setting some arrays to [] are an error because we can't do what they
	// asked - which is different from "null"/absent - which means "default"
	xHTTP(t, reg, "PUT", "/capabilities", `{ "specversions": [] }`,
		400, "\"specversions\" must contain \""+registry.SPECVERSION+"\"\n")

	// Unknown key
	xHTTP(t, reg, "PUT", "/capabilities", `{ "foo": [] }`,
		400, "Unknown capability: \"foo\"\n")
}

func TestCapabilityAttr(t *testing.T) {
	reg := NewRegistry("TestCapabilityAttr")
	defer PassDeleteReg(t, reg)

	// Verify epoch value
	xHTTP(t, reg, "GET", "/", ``, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
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
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {} }`, 200,
		`{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "pagination": false,
    "schemas": [
      "xregistry-json/`+registry.SPECVERSION+`"
    ],
    "shortself": false,
    "specversions": [
      "`+registry.SPECVERSION+`"
    ],
    "sticky": true
  }
}
`)

	// Setting to nulls
	// notice ?inline is still disabled!
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {
  "flags": null,
  "mutable": null,
  "pagination": null,
  "schemas": null,
  "shortself": null,
  "specversions": null,
  "sticky": null
}}`, 200,
		`{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	// Testing setting everything to the default
	// inline still disabled
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {
  "flags": [
    "collections", "doc", "epoch", "filter", "inline", "nodefaultversionid",
    "nodefaultversionsticky", "noepoch", "noreadonly", "offered", "schema",
	"setdefaultversionid", "specversion"
  ],
  "mutable": [ "capabilities", "entities", "model" ],
  "pagination": false,
  "schemas": [ "xregistry-json/`+registry.SPECVERSION+`" ],
  "shortself": false,
  "specversions": [ "`+registry.SPECVERSION+`" ],
  "sticky": false
}}`, 200,
		`{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "flags": [
    "collections",
    "doc",
    "epoch",
    "filter",
    "inline",
    "nodefaultversionid",
    "nodefaultversionsticky",
    "noepoch",
    "noreadonly",
    "offered",
    "schema",
    "setdefaultversionid",
    "specversion"
  ],
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": false
}
`)

	// Setting to minimal
	// inline still enabled
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities": {
  "pagination": false,
  "schemas": ["xregistry-json"],
  "shortself": false,
  "specversions": ["`+registry.SPECVERSION+`"],
  "sticky": true
}}`, 200,
		`{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestCapabilityAttr",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "capabilities": {
    "pagination": false,
    "schemas": [
      "xregistry-json/`+registry.SPECVERSION+`"
    ],
    "shortself": false,
    "specversions": [
      "`+registry.SPECVERSION+`"
    ],
    "sticky": true
  }
}
`)

	xHTTP(t, reg, "GET", "/capabilities", ``, 200, `{
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	// Setting some arrays to [] are an error because we can't do what they
	// asked - which is different from "null"/absent - which means "default"
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities":
	    {"schemas": [] }}`,
		400, "\"schemas\" must contain \"xRegistry-json/"+registry.SPECVERSION+"\"\n")

	// Setting some arrays to [] are an error because we can't do what they
	// asked - which is different from "null"/absent - which means "default"
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities":
	    {"specversions": [] }}`,
		400, "\"specversions\" must contain \""+registry.SPECVERSION+"\"\n")

	// Unknown key
	xHTTP(t, reg, "PUT", "/?inline=capabilities", `{ "capabilities":
	    {"foo": [] }}`,
		400, `Unknown capability: "foo"
`)

}

// "collections", "doc", "epoch", "filter", "inline",
// "nodefaultversionid", "nodefaultversionsticky",
// "noepoch", "noreadonly", "offered", "schema", "setdefaultversionid",
// "specversion"})

func TestCapabilityFlagsOff(t *testing.T) {
	reg := NewRegistry("TestCapabilityFlags")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)

	xHTTP(t, reg, "PUT", "/capabilities", `{"mutable":["*"]}`, 200, `{
  "mutable": [
    "capabilities",
    "entities",
    "model"
  ],
  "pagination": false,
  "schemas": [
    "xregistry-json/`+registry.SPECVERSION+`"
  ],
  "shortself": false,
  "specversions": [
    "`+registry.SPECVERSION+`"
  ],
  "sticky": true
}
`)

	// Create a test file
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{}`, 201, `{
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
	xHTTP(t, reg, "GET", "/dirs/d1/files?doc", `{}`, 200, `{
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
	xHTTP(t, reg, "GET", "/dirs/d1/files?filter=foo&inline=bar", `{}`, 200, `{
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
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1?epoch=99", `{}`, 204, ``)

	// Test ?setdefaultversionid
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1?setdefaultversionid=x", `{
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

	// Test ?schema
	xHTTP(t, reg, "GET", "/model?schema=foo", ``, 200, `*`)

	// Test ?specversion
	xHTTP(t, reg, "GET", "/model?specversion=foo", ``, 200, `*`)

	// TODO nodefaultversionid, nodefaultversionsticky, noepoch, noreadonly
}

func TestCapabilityOffered(t *testing.T) {
	reg := NewRegistry("TestCapabilityOffered")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "GET", "/capabilities?offered", ``, 200, `{
  "flags": {
    "type": "array",
    "item": {
      "type": "string"
    },
    "enum": [
      "collections",
      "doc",
      "epoch",
      "filter",
      "inline",
      "nodefaultversionid",
      "nodefaultversionsticky",
      "noepoch",
      "noreadonly",
      "offered",
      "schema",
      "setdefaultversionid",
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
  "schemas": {
    "type": "string",
    "enum": [
      "xregistry-json/`+registry.SPECVERSION+`"
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
      "`+registry.SPECVERSION+`"
    ]
  },
  "sticky": {
    "type": "boolean",
    "enum": [
      false,
      true
    ]
  }
}
`)
}
