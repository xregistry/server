package tests

// err missing ancestor

import (
	"fmt"
	"testing"

	"github.com/xregistry/server/registry"
)

func TestAncestorBasic(t *testing.T) {
	reg := NewRegistry("TestAncestorBasic")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true, false)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f2", `{}`, 201, `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2/versions/1",
  "xid": "/dirs/d1/files/f2/versions/1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-04-11T20:40:37.146317496Z",
  "modifiedat": "2025-04-11T20:40:37.146317496Z",
  "ancestor": "1"
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f3/versions", `{
  "v1": {}
}`, 200, `{
  "v1": {
    "fileid": "f3",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/v1",
    "xid": "/dirs/d1/files/f3/versions/v1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v1"
  }
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f4/versions/v1", `{}`, 201, `{
  "fileid": "f4",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f4/versions/v1",
  "xid": "/dirs/d1/files/f4/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "DELETE", "/dirs/d1/files", ``, 204, ``)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{"ancestor": null}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f2", `{"ancestor": ""}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/dirs/d1/files/f2",
  "title": "The attribute \"ancestor\" is not valid: value \"\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f2", `{"ancestor": "vx"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f2",
  "title": "The request cannot be processed as provided: can't find \"ancestor\" Verison(s): vx"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f2", `{"ancestor": "1"}`, 201, `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2",
  "xid": "/dirs/d1/files/f2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f2", `{"ancestor": "1"}`, 200, `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2",
  "xid": "/dirs/d1/files/f2",
  "epoch": 2,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f2", `{"ancestor": "2"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f2",
  "title": "The request cannot be processed as provided: can't find \"ancestor\" Verison(s): 2"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f2", `{"ancestor": "2"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f2",
  "title": "The request cannot be processed as provided: can't find \"ancestor\" Verison(s): 2"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f3", `{"ancestor": "1"}`, 201, `{
  "fileid": "f3",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f3",
  "xid": "/dirs/d1/files/f3",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f3/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f3/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f3", `{"ancestor": "2"}`, 201, `{
  "fileid": "f3",
  "versionid": "2",
  "self": "http://localhost:8181/dirs/d1/files/f3/versions/2",
  "xid": "/dirs/d1/files/f3/versions/2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "2"
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f3/versions", `{
  "5": {}, "3":{}, "4":{}
}`, 200, `{
  "3": {
    "fileid": "f3",
    "versionid": "3",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/3",
    "xid": "/dirs/d1/files/f3/versions/3",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "2"
  },
  "4": {
    "fileid": "f3",
    "versionid": "4",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/4",
    "xid": "/dirs/d1/files/f3/versions/4",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "3"
  },
  "5": {
    "fileid": "f3",
    "versionid": "5",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/5",
    "xid": "/dirs/d1/files/f3/versions/5",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "4"
  }
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f3/versions", `{
  "5":{"createdat": "2023-01-01T12:00:00Z","ancestor":null},
  "3":{"ancestor":null},
  "4":{"ancestor":null}
}`, 200, `{
  "3": {
    "fileid": "f3",
    "versionid": "3",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/3",
    "xid": "/dirs/d1/files/f3/versions/3",
    "epoch": 2,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestor": "5"
  },
  "4": {
    "fileid": "f3",
    "versionid": "4",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/4",
    "xid": "/dirs/d1/files/f3/versions/4",
    "epoch": 2,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestor": "3"
  },
  "5": {
    "fileid": "f3",
    "versionid": "5",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/5",
    "xid": "/dirs/d1/files/f3/versions/5",
    "epoch": 2,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:03Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestor": "2"
  }
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f3/versions", `{
  "5": {"createdat":null, "ancestor":null},
  "4":{ "ancestor": "1"},
  "3":{"ancestor": null}
}`, 200, `{
  "3": {
    "fileid": "f3",
    "versionid": "3",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/3",
    "xid": "/dirs/d1/files/f3/versions/3",
    "epoch": 3,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestor": "4"
  },
  "4": {
    "fileid": "f3",
    "versionid": "4",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/4",
    "xid": "/dirs/d1/files/f3/versions/4",
    "epoch": 3,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestor": "1"
  },
  "5": {
    "fileid": "f3",
    "versionid": "5",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/5",
    "xid": "/dirs/d1/files/f3/versions/5",
    "epoch": 3,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestor": "3"
  }
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f4/versions", `{
  "1": {"ancestor":"3"}, "2":{}, "3":{}
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f4",
  "title": "The request cannot be processed as provided: circular \"ancestor\" references detected for Versions: 1, 2, 3"
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f4/versions", `{
  "1": {"ancestor":"2"}, "2":{}, "3":{}
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f4",
  "title": "The request cannot be processed as provided: circular \"ancestor\" references detected for Versions: 1, 2"
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f4/versions", `{
  "1":{"ancestor":"2"}, "2":{"ancestor":"1"},
  "3":{"ancestor":"4"}, "4":{"ancestor":"3"}
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f4",
  "title": "The request cannot be processed as provided: circular \"ancestor\" references detected for Versions: 1, 2, 3, 4"
}
`)

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f3/versions/1", `{}`, 204, ``)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f3/versions", ``, 200, `{
  "2": {
    "fileid": "f3",
    "versionid": "2",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/2",
    "xid": "/dirs/d1/files/f3/versions/2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "2"
  },
  "3": {
    "fileid": "f3",
    "versionid": "3",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/3",
    "xid": "/dirs/d1/files/f3/versions/3",
    "epoch": 3,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:03Z",
    "ancestor": "4"
  },
  "4": {
    "fileid": "f3",
    "versionid": "4",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/4",
    "xid": "/dirs/d1/files/f3/versions/4",
    "epoch": 4,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:04Z",
    "ancestor": "4"
  },
  "5": {
    "fileid": "f3",
    "versionid": "5",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/5",
    "xid": "/dirs/d1/files/f3/versions/5",
    "epoch": 3,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:03Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:03Z",
    "ancestor": "3"
  }
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f5/versions", `{"1":{}, "2":{}}`,
		200, `{
  "1": {
    "fileid": "f5",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f5/versions/1",
    "xid": "/dirs/d1/files/f5/versions/1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "1"
  },
  "2": {
    "fileid": "f5",
    "versionid": "2",
    "self": "http://localhost:8181/dirs/d1/files/f5/versions/2",
    "xid": "/dirs/d1/files/f5/versions/2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "1"
  }
}
`)

	// Make sure ancestor doesn't get erased
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f5/versions/1", `{}`, 200, `{
  "fileid": "f5",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f5/versions/1",
  "xid": "/dirs/d1/files/f5/versions/1",
  "epoch": 2,
  "isdefault": false,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestor": "1"
}
`)

	// all epochs should be 1 after this
	xHTTP(t, reg, "POST", "/dirs/d1/files/f6/versions",
		`{"v1":{}, "v2":{}, "v3":{}}`, 200, `{
  "v1": {
    "fileid": "f6",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f6/versions/v1",
    "xid": "/dirs/d1/files/f6/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v1"
  },
  "v2": {
    "fileid": "f6",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f6/versions/v2",
    "xid": "/dirs/d1/files/f6/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v1"
  },
  "v3": {
    "fileid": "f6",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/f6/versions/v3",
    "xid": "/dirs/d1/files/f6/versions/v3",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v2"
  }
}
`)

	// now delete v1 which causes v2's ancestor, epoch and modifiedat to
	// change. Note if we ever optimize things such that when we delete all
	// versions we just delete the resource instead, we'll need to find a
	// different way to make this side-effect aspect is tested some other way:w
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f6/versions",
		`{"v1":{"epoch":1}, "v2":{"epoch": 1}, "v3":{"epoch":1}}`, 204, ``)

}

func TestAncestorWithSicky(t *testing.T) {
	reg := NewRegistry("TestAncestorWithSticky")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	rm, err := gm.AddResourceModel("files", "file", 0, true, true, false)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=meta", `{
      "meta":{"defaultversionsticky": true,"defaultversionid": "v1"},
      "versions":{"v1":{},"v2":{},"v3":{}}
    }`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	f1, err := reg.FindResourceByXID("/dirs/d1/files/f1", "/")
	xNoErr(t, err)
	vas, err := f1.GetOrderedVersionIDs() // []*VersionAncestor
	xNoErr(t, err)

	xCheckEqual(t, "", VAS2String(vas),
		"(v1->v1,0)(v2->v1,1)(v3->v2,2)")

	rm.SetMaxVersions(2)
	xNoErr(t, reg.Model.VerifyAndSave())

	f1, err = reg.FindResourceByXID("/dirs/d1/files/f1", "/")
	xNoErr(t, err)
	vas, err = f1.GetOrderedVersionIDs() // []*VersionAncestor
	xNoErr(t, err)

	xCheckEqual(t, "", VAS2String(vas),
		"(v1->v1,0)(v3->v3,0)")
}

func TestAncestorOrdering(t *testing.T) {
	reg := NewRegistry("TestAncestorOrdering")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true, false)

	// Timestamps should be the determining factor
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "versions": {
    "v1": { "createdat": "2025-01-01T12:00:00" },
    "v2": { "createdat": "2024-01-01T12:00:00" },
    "V3": { "createdat": "2023-01-01T12:00:00" },
    "v4": { "createdat": "2022-01-01T12:00:00" }
  }
}`, 201, `*`)

	f1, err := reg.FindResourceByXID("/dirs/d1/files/f1", "/")
	xNoErr(t, err)
	vas, err := f1.GetOrderedVersionIDs() // []*VersionAncestor
	xNoErr(t, err)

	xCheckEqual(t, "", VAS2String(vas),
		"(v4->v4,0)(V3->v4,1)(v2->V3,1)(v1->v2,2)")

	// Reverse the order of the timestamps, and clear ancestor
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1", `{
  "versions": {
    "v1": { "createdat": "2022-01-01T12:00:00", "ancestor": null },
    "v2": { "createdat": "2023-01-01T12:00:00", "ancestor": null },
    "V3": { "createdat": "2024-01-01T12:00:00", "ancestor": null },
    "v4": { "createdat": "2025-01-01T12:00:00", "ancestor": null }
  }
}`, 200, `*`)

	vas, _ = f1.GetOrderedVersionIDs() // []*VersionAncestor
	xCheckEqual(t, "", VAS2String(vas),
		"(v1->v1,0)(v2->v1,1)(V3->v2,1)(v4->V3,2)")

	// Make it into a tree 1<-2,3,4 diff timestamps
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1", `{
  "versions": {
    "v1": { "createdat": "2025-01-01T12:00:00", "ancestor": "v1" },
    "v2": { "createdat": "2023-01-01T12:00:00", "ancestor": "v1" },
    "V3": { "createdat": "2024-01-01T12:00:00", "ancestor": "v1" },
    "v4": { "createdat": "2022-01-01T12:00:00", "ancestor": "v1" }
  }
}`, 200, `*`)

	vas, _ = f1.GetOrderedVersionIDs() // []*VersionAncestor
	xCheckEqual(t, "", VAS2String(vas),
		"(v1->v1,0)(v4->v1,2)(v2->v1,2)(V3->v1,2)")

	// Same, but use same TS, so it'll alphabetize things (case insense)
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1", `{
  "versions": {
    "v1": { "createdat": "2025-01-01T12:00:00"},
    "v2": { "createdat": "2023-01-01T12:00:00"},
    "V3": { "createdat": "2023-01-01T12:00:00"},
    "v4": { "createdat": "2023-01-01T12:00:00"}
  }
}`, 200, `*`)

	vas, _ = f1.GetOrderedVersionIDs() // []*VersionAncestor
	xCheckEqual(t, "", VAS2String(vas),
		"(v1->v1,0)(v2->v1,2)(V3->v1,2)(v4->v1,2)")

	// Deep tree and add a new more
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1", `{
  "versions": {
    "v1": { "ancestor": "v1" },
    "v2": { "ancestor": "v1" },
    "V3": { "ancestor": "v2" },
    "v4": { "ancestor": "V3" },

    "v1.1.0": { "ancestor": "v1" },
    "v1.1.1": { "ancestor": "v1.1.0" },

    "v2.1.0": { "ancestor": "v2" }
  }
}`, 200, `*`)

	// v4 is older than v1.1.. and v2, and then v1.1 < v2 alphabetically
	vas, _ = f1.GetOrderedVersionIDs() // []*VersionAncestor
	xCheckEqual(t, "", VAS2String(vas),
		"(v1->v1,0)(v2->v1,1)(V3->v2,1)(v1.1.0->v1,1)(v4->V3,2)(v1.1.1->v1.1.0,2)(v2.1.0->v2,2)")

}

func TestAncestorRoots(t *testing.T) {
	reg := NewRegistry("TestAncestorOrdering")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	rm, err := gm.AddResourceModel("files", "file", 0, true, true, false)

	// Start with singlversionroot=default (which should be 'false')

	// Timestamps should be the determining factor
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "versions": {
    "v1": { "createdat": "2025-01-01T12:00:00", "ancestor":"v1" },
    "v2": { "createdat": "2024-01-01T12:00:00", "ancestor":"v2" }
  }
}`, 201, `*`)

	f1, err := reg.FindResourceByXID("/dirs/d1/files/f1", "/")
	xNoErr(t, err)
	vas, err := f1.GetOrderedVersionIDs() // []*VersionAncestor
	xNoErr(t, err)

	xCheckEqual(t, "", VAS2String(vas), "(v2->v2,0)(v1->v1,0)")

	rm.SetSingleVersionRoot(false)
	xNoErr(t, reg.Model.VerifyAndSave())

	// Trying to turn singleversionroot=true should generate an error
	rm.SetSingleVersionRoot(true)
	err = reg.Model.VerifyAndSave()
	xCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "/dirs/d1/files/f1",
  "title": "The request cannot be processed as provided: \"dirs/d1/files/f1\" has too many (2) root versions"
}`)
	reg.LoadModel()   // reset
	rm = rm.Refresh() // reload

	// convert a root into a leaf and try again
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions/v2",
		`{"ancestor":"v1"}`, 200, `*`)

	f1, err = reg.FindResourceByXID("/dirs/d1/files/f1", "/")
	xNoErr(t, err)
	vas, err = f1.GetOrderedVersionIDs() // []*VersionAncestor
	xNoErr(t, err)

	xCheckEqual(t, "", VAS2String(vas), "(v1->v1,0)(v2->v1,2)")

	rm.SetSingleVersionRoot(true)
	xNoErr(t, reg.Model.VerifyAndSave())

	// make sure an add of a root fails
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v3",
		`{"ancestor":"v3"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The request cannot be processed as provided: \"dirs/d1/files/f1\" has too many (2) root versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions/v3",
		`{"ancestor":"v3"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The request cannot be processed as provided: \"dirs/d1/files/f1\" has too many (2) root versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions",
		`{"v3":{"ancestor":"v3"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The request cannot be processed as provided: \"dirs/d1/files/f1\" has too many (2) root versions"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1",
		`{"versions":{"v3":{"ancestor":"v3"}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The request cannot be processed as provided: \"dirs/d1/files/f1\" has too many (2) root versions"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions":{"v3":{"ancestor":"v3"}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The request cannot be processed as provided: \"dirs/d1/files/f1\" has too many (2) root versions"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f2",
		`{"versions":{"v1":{"ancestor":"v1"},"v3":{"ancestor":"v3"}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f2",
  "title": "The request cannot be processed as provided: \"dirs/d1/files/f2\" has too many (2) root versions"
}
`)

}

func TestAncestorCircles(t *testing.T) {
	reg := NewRegistry("TestAncestorCircles")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true, false)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions":{"v1":{"ancestor":"v1"},"v2":{"ancestor":"v1"}}}`,
		201, `*`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1",
		`{"versions":{"v1":{"ancestor":"v2"}}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The request cannot be processed as provided: circular \"ancestor\" references detected for Versions: v1, v2"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f2",
		`{"versions":{"v1":{"ancestor":"v2"},"v2":{"ancestor":"v1"}}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f2",
  "title": "The request cannot be processed as provided: circular \"ancestor\" references detected for Versions: v1, v2"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f2",
		`{"versions":{"v1":{"ancestor":"v2"},"v2":{"ancestor":"v1"},
		              "v3":{"ancestor":"v4"},"v4":{"ancestor":"v3"}}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f2",
  "title": "The request cannot be processed as provided: circular \"ancestor\" references detected for Versions: v1, v2, v3, v4"
}
`)

}

func TestAncestorMaxVersions(t *testing.T) {
	reg := NewRegistry("TestAncestorCircles")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	rm, err := gm.AddResourceModel("files", "file", 0, true, true, false)

	rm.SetMaxVersions(1)
	xNoErr(t, reg.Model.VerifyAndSave())

	// the circular ref shouldn't be an issue because we'll delete the
	// oldest one due to maxversions
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions":{"v1":{"ancestor":"v2"},"v2":{"ancestor":"v1"}}}`,
		201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	//  v2->v1->v3->v3
	// Should delete v3
	rm.SetMaxVersions(2)
	xNoErr(t, reg.Model.VerifyAndSave())

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions":{"v1":{"ancestor":"v1"},"v2":{"ancestor":"v1"}}}`,
		200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions":{"v1":{"ancestor":"v3"},"v3":{"ancestor":"v3"}}}`,
		200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	f1, err := reg.FindResourceByXID("/dirs/d1/files/f1", "/")
	xNoErr(t, err)
	vas, err := f1.GetOrderedVersionIDs() // []*VersionAncestor
	xNoErr(t, err)

	xCheckEqual(t, "", VAS2String(vas),
		"(v1->v1,0)(v2->v1,2)")

	// v3->v2->v1 + default=v1/sticky
	// should delete v2, v3 becomes root
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"meta":{"defaultversionid":"v1","defaultversionsticky":true},
          "versions":{"v1":{"ancestor":"v1"},"v3":{"ancestor":"v2"}}}`,
		200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	f1, err = reg.FindResourceByXID("/dirs/d1/files/f1", "/")
	xNoErr(t, err)
	vas, err = f1.GetOrderedVersionIDs() // []*VersionAncestor
	xNoErr(t, err)

	xCheckEqual(t, "", VAS2String(vas),
		"(v1->v1,0)(v3->v3,0)")
}

func TestAncestorErrors(t *testing.T) {
	reg := NewRegistry("TestAncestorErrors")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, true, false)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions": {"v1":{"ancestor":"v2"}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The request cannot be processed as provided: can't find \"ancestor\" Verison(s): v2"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions": {"v1":{"ancestor":"v1"}}}`, 201, `*`)
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions": {"v1":{"ancestor":"v2"}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The request cannot be processed as provided: can't find \"ancestor\" Verison(s): v2"
}
`)

}

func VAS2String(vas []*registry.VersionAncestor) string {
	res := ""
	for _, va := range vas {
		// Pos, 0=root, 1=middle, 2=leaf
		res += fmt.Sprintf("(%s->%s,%s)", va.VID, va.Ancestor, va.Pos[:1])
	}
	return res
}
