package tests

import (
	"testing"
	// "github.com/xregistry/server/registry"
)

func TestHTTPMixedCase(t *testing.T) {
	reg := NewRegistry("TestHTTPMixedCase")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	_, err = gm.AddResourceModelSimple("files", "file")

	xHTTP(t, reg, "POST", "/?inline&doc", `{
  "dirs": {
    "Dir1": {
      "files": {
        "File1": {
          "versions": {
            "_": {
              "contenttype": "application/json",
              "file": { "hello": "world" }
            }
          }
        }
      }
    },
    "DiR2_.-~@DiR": {
      "files": {
        "FiLe2_.-~@FiL": {
          "versionid": "666"
        }
      }
    }
  }
}`, 200, `{
  "dirs": {
    "DiR2_.-~@DiR": {
      "dirid": "DiR2_.-~@DiR",
      "self": "#/dirs/DiR2_.-~@DiR",
      "xid": "/dirs/DiR2_.-~@DiR",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

      "filesurl": "#/dirs/DiR2_.-~@DiR/files",
      "files": {
        "FiLe2_.-~@FiL": {
          "fileid": "FiLe2_.-~@FiL",
          "self": "#/dirs/DiR2_.-~@DiR/files/FiLe2_.-~@FiL",
          "xid": "/dirs/DiR2_.-~@DiR/files/FiLe2_.-~@FiL",

          "metaurl": "#/dirs/DiR2_.-~@DiR/files/FiLe2_.-~@FiL/meta",
          "meta": {
            "fileid": "FiLe2_.-~@FiL",
            "self": "#/dirs/DiR2_.-~@DiR/files/FiLe2_.-~@FiL/meta",
            "xid": "/dirs/DiR2_.-~@DiR/files/FiLe2_.-~@FiL/meta",
            "epoch": 1,
            "createdat": "YYYY-MM-DDTHH:MM:01Z",
            "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "666",
            "defaultversionurl": "#/dirs/DiR2_.-~@DiR/files/FiLe2_.-~@FiL/versions/666",
            "defaultversionsticky": false
          },
          "versionsurl": "#/dirs/DiR2_.-~@DiR/files/FiLe2_.-~@FiL/versions",
          "versions": {
            "666": {
              "fileid": "FiLe2_.-~@FiL",
              "versionid": "666",
              "self": "#/dirs/DiR2_.-~@DiR/files/FiLe2_.-~@FiL/versions/666",
              "xid": "/dirs/DiR2_.-~@DiR/files/FiLe2_.-~@FiL/versions/666",
              "epoch": 1,
              "isdefault": true,
              "createdat": "YYYY-MM-DDTHH:MM:01Z",
              "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
              "ancestor": "666"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1
    },
    "Dir1": {
      "dirid": "Dir1",
      "self": "#/dirs/Dir1",
      "xid": "/dirs/Dir1",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

      "filesurl": "#/dirs/Dir1/files",
      "files": {
        "File1": {
          "fileid": "File1",
          "self": "#/dirs/Dir1/files/File1",
          "xid": "/dirs/Dir1/files/File1",

          "metaurl": "#/dirs/Dir1/files/File1/meta",
          "meta": {
            "fileid": "File1",
            "self": "#/dirs/Dir1/files/File1/meta",
            "xid": "/dirs/Dir1/files/File1/meta",
            "epoch": 1,
            "createdat": "YYYY-MM-DDTHH:MM:01Z",
            "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "_",
            "defaultversionurl": "#/dirs/Dir1/files/File1/versions/_",
            "defaultversionsticky": false
          },
          "versionsurl": "#/dirs/Dir1/files/File1/versions",
          "versions": {
            "_": {
              "fileid": "File1",
              "versionid": "_",
              "self": "#/dirs/Dir1/files/File1/versions/_",
              "xid": "/dirs/Dir1/files/File1/versions/_",
              "epoch": 1,
              "isdefault": true,
              "createdat": "YYYY-MM-DDTHH:MM:01Z",
              "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
              "ancestor": "_",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1
    }
  }
}
`)

}
