package tests

import (
	"testing"

	"github.com/xregistry/server/registry"
)

func TestExportBasic(t *testing.T) {
	reg := NewRegistry("TestExportBasic")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details",
		`{"file": { "hello": "world" }}`, 201, `*`)
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2$details",
		`{"file": { "hello": "world" }}`, 201, `*`)
	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx$details",
		`{"meta": { "xref": "/dirs/d1/files/f1" }}`, 201, `*`)

	// Full export - 2 different ways
	code, fullBody := xGET(t, "export")
	xCheckEqual(t, "", code, 200)

	code, manualBody := xGET(t, "?doc&inline=*,capabilities,model")
	xCheckEqual(t, "", code, 200)
	xCheckEqual(t, "", fullBody, manualBody)

	xCheckEqual(t, "", fullBody, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportBasic",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "capabilities": {
    "flags": [
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
  },
  "model": {
    "attributes": {
      "specversion": {
        "name": "specversion",
        "type": "string",
        "readonly": true,
        "immutable": true,
        "required": true
      },
      "registryid": {
        "name": "registryid",
        "type": "string",
        "immutable": true,
        "required": true
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "required": true
      },
      "xid": {
        "name": "xid",
        "type": "xid",
        "readonly": true,
        "required": true
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
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
            "required": true
          },
          "xid": {
            "name": "xid",
            "type": "xid",
            "readonly": true,
            "required": true
          },
          "epoch": {
            "name": "epoch",
            "type": "uinteger",
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
          }
        },
        "resources": {
          "files": {
            "plural": "files",
            "singular": "file",
            "maxversions": 0,
            "setversionid": true,
            "setdefaultversionsticky": true,
            "hasdocument": true,
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
                "required": true
              },
              "xid": {
                "name": "xid",
                "type": "xid",
                "readonly": true,
                "required": true
              },
              "epoch": {
                "name": "epoch",
                "type": "uinteger",
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
              "contenttype": {
                "name": "contenttype",
                "type": "string"
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
                "required": true
              },
              "xid": {
                "name": "xid",
                "type": "xid",
                "readonly": true,
                "required": true
              },
              "xref": {
                "name": "xref",
                "type": "url"
              },
              "epoch": {
                "name": "epoch",
                "type": "uinteger",
                "required": true
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
                "required": true,
                "default": false
              },
              "compatibility": {
                "name": "compatibility",
                "type": "string",
                "enum": [
                  "none",
                  "backward",
                  "backward_transitive",
                  "forward",
                  "forward_transitive",
                  "full",
                  "full_transitive"
                ],
                "strict": false,
                "required": true,
                "default": "none"
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
                "readonly": true,
                "required": true,
                "default": false
              }
            }
          }
        }
      }
    }
  },

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "#/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "#/dirs/d1/files/f1",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "#/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "#/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 2,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:04Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v2",
            "defaultversionurl": "#/dirs/d1/files/f1/versions/v2",
            "defaultversionsticky": false
          },
          "versionsurl": "#/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "#/dirs/d1/files/f1/versions/v1",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 1,
              "isdefault": false,
              "createdat": "2025-01-01T12:00:02Z",
              "modifiedat": "2025-01-01T12:00:02Z",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "#/dirs/d1/files/f1/versions/v2",
              "xid": "/dirs/d1/files/f1/versions/v2",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:04Z",
              "modifiedat": "2025-01-01T12:00:04Z",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            }
          },
          "versionscount": 2
        },
        "fx": {
          "fileid": "fx",
          "self": "#/dirs/d1/files/fx",
          "xid": "/dirs/d1/files/fx",

          "metaurl": "#/dirs/d1/files/fx/meta",
          "meta": {
            "fileid": "fx",
            "self": "#/dirs/d1/files/fx/meta",
            "xid": "/dirs/d1/files/fx/meta",
            "xref": "/dirs/d1/files/f1"
          }
        }
      },
      "filescount": 2
    }
  },
  "dirscount": 1
}
`)

	// Play with ?export vanilla
	code, fullBody = xGET(t, "export?inline=*")
	xCheckEqual(t, "", code, 200)
	code, manualBody = xGET(t, "?doc&inline=*")
	xCheckEqual(t, "", code, 200)
	xCheckEqual(t, "", fullBody, manualBody)

	xCheckEqual(t, "", fullBody, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportBasic",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "#/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "#/dirs/d1/files/f1",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "#/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "#/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 2,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:04Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v2",
            "defaultversionurl": "#/dirs/d1/files/f1/versions/v2",
            "defaultversionsticky": false
          },
          "versionsurl": "#/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "#/dirs/d1/files/f1/versions/v1",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 1,
              "isdefault": false,
              "createdat": "2025-01-01T12:00:02Z",
              "modifiedat": "2025-01-01T12:00:02Z",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "#/dirs/d1/files/f1/versions/v2",
              "xid": "/dirs/d1/files/f1/versions/v2",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:04Z",
              "modifiedat": "2025-01-01T12:00:04Z",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            }
          },
          "versionscount": 2
        },
        "fx": {
          "fileid": "fx",
          "self": "#/dirs/d1/files/fx",
          "xid": "/dirs/d1/files/fx",

          "metaurl": "#/dirs/d1/files/fx/meta",
          "meta": {
            "fileid": "fx",
            "self": "#/dirs/d1/files/fx/meta",
            "xid": "/dirs/d1/files/fx/meta",
            "xref": "/dirs/d1/files/f1"
          }
        }
      },
      "filescount": 2
    }
  },
  "dirscount": 1
}
`)

	// Play with ?doc inline just capabilities
	code, fullBody = xGET(t, "export?inline=capabilities")
	xCheckEqual(t, "", code, 200)
	code, manualBody = xGET(t, "?doc&inline=capabilities")
	xCheckEqual(t, "", code, 200)
	xCheckEqual(t, "", fullBody, manualBody)

	xCheckEqual(t, "", fullBody, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportBasic",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "capabilities": {
    "flags": [
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
  },

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	// Play with ?doc inline just model
	code, fullBody = xGET(t, "export?inline=model")
	xCheckEqual(t, "", code, 200)
	code, manualBody = xGET(t, "?doc&inline=model")
	xCheckEqual(t, "", code, 200)
	xCheckEqual(t, "", fullBody, manualBody)

	xCheckEqual(t, "", fullBody, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportBasic",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "model": {
    "attributes": {
      "specversion": {
        "name": "specversion",
        "type": "string",
        "readonly": true,
        "immutable": true,
        "required": true
      },
      "registryid": {
        "name": "registryid",
        "type": "string",
        "immutable": true,
        "required": true
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "required": true
      },
      "xid": {
        "name": "xid",
        "type": "xid",
        "readonly": true,
        "required": true
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
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
            "required": true
          },
          "xid": {
            "name": "xid",
            "type": "xid",
            "readonly": true,
            "required": true
          },
          "epoch": {
            "name": "epoch",
            "type": "uinteger",
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
          }
        },
        "resources": {
          "files": {
            "plural": "files",
            "singular": "file",
            "maxversions": 0,
            "setversionid": true,
            "setdefaultversionsticky": true,
            "hasdocument": true,
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
                "required": true
              },
              "xid": {
                "name": "xid",
                "type": "xid",
                "readonly": true,
                "required": true
              },
              "epoch": {
                "name": "epoch",
                "type": "uinteger",
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
              "contenttype": {
                "name": "contenttype",
                "type": "string"
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
                "required": true
              },
              "xid": {
                "name": "xid",
                "type": "xid",
                "readonly": true,
                "required": true
              },
              "xref": {
                "name": "xref",
                "type": "url"
              },
              "epoch": {
                "name": "epoch",
                "type": "uinteger",
                "required": true
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
                "required": true,
                "default": false
              },
              "compatibility": {
                "name": "compatibility",
                "type": "string",
                "enum": [
                  "none",
                  "backward",
                  "backward_transitive",
                  "forward",
                  "forward_transitive",
                  "full",
                  "full_transitive"
                ],
                "strict": false,
                "required": true,
                "default": "none"
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
                "readonly": true,
                "required": true,
                "default": false
              }
            }
          }
        }
      }
    }
  },

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	// Play with ?doc not at root
	xHTTP(t, reg, "GET", "/dirs?doc&inline=*", ``, 200, `{
  "d1": {
    "dirid": "d1",
    "self": "#/d1",
    "xid": "/dirs/d1",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:03Z",

    "filesurl": "#/d1/files",
    "files": {
      "f1": {
        "fileid": "f1",
        "self": "#/d1/files/f1",
        "xid": "/dirs/d1/files/f1",

        "metaurl": "#/d1/files/f1/meta",
        "meta": {
          "fileid": "f1",
          "self": "#/d1/files/f1/meta",
          "xid": "/dirs/d1/files/f1/meta",
          "epoch": 2,
          "createdat": "2025-01-01T12:00:02Z",
          "modifiedat": "2025-01-01T12:00:04Z",
          "readonly": false,
          "compatibility": "none",

          "defaultversionid": "v2",
          "defaultversionurl": "#/d1/files/f1/versions/v2",
          "defaultversionsticky": false
        },
        "versionsurl": "#/d1/files/f1/versions",
        "versions": {
          "v1": {
            "fileid": "f1",
            "versionid": "v1",
            "self": "#/d1/files/f1/versions/v1",
            "xid": "/dirs/d1/files/f1/versions/v1",
            "epoch": 1,
            "isdefault": false,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:02Z",
            "contenttype": "application/json",
            "file": {
              "hello": "world"
            }
          },
          "v2": {
            "fileid": "f1",
            "versionid": "v2",
            "self": "#/d1/files/f1/versions/v2",
            "xid": "/dirs/d1/files/f1/versions/v2",
            "epoch": 1,
            "isdefault": true,
            "createdat": "2025-01-01T12:00:04Z",
            "modifiedat": "2025-01-01T12:00:04Z",
            "contenttype": "application/json",
            "file": {
              "hello": "world"
            }
          }
        },
        "versionscount": 2
      },
      "fx": {
        "fileid": "fx",
        "self": "#/d1/files/fx",
        "xid": "/dirs/d1/files/fx",

        "metaurl": "#/d1/files/fx/meta",
        "meta": {
          "fileid": "fx",
          "self": "#/d1/files/fx/meta",
          "xid": "/dirs/d1/files/fx/meta",
          "xref": "/dirs/d1/files/f1"
        }
      }
    },
    "filescount": 2
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1?doc&inline=*", ``, 200, `{
  "dirid": "d1",
  "self": "#/",
  "xid": "/dirs/d1",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:03Z",

  "filesurl": "#/files",
  "files": {
    "f1": {
      "fileid": "f1",
      "self": "#/files/f1",
      "xid": "/dirs/d1/files/f1",

      "metaurl": "#/files/f1/meta",
      "meta": {
        "fileid": "f1",
        "self": "#/files/f1/meta",
        "xid": "/dirs/d1/files/f1/meta",
        "epoch": 2,
        "createdat": "2025-01-01T12:00:02Z",
        "modifiedat": "2025-01-01T12:00:04Z",
        "readonly": false,
        "compatibility": "none",

        "defaultversionid": "v2",
        "defaultversionurl": "#/files/f1/versions/v2",
        "defaultversionsticky": false
      },
      "versionsurl": "#/files/f1/versions",
      "versions": {
        "v1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "#/files/f1/versions/v1",
          "xid": "/dirs/d1/files/f1/versions/v1",
          "epoch": 1,
          "isdefault": false,
          "createdat": "2025-01-01T12:00:02Z",
          "modifiedat": "2025-01-01T12:00:02Z",
          "contenttype": "application/json",
          "file": {
            "hello": "world"
          }
        },
        "v2": {
          "fileid": "f1",
          "versionid": "v2",
          "self": "#/files/f1/versions/v2",
          "xid": "/dirs/d1/files/f1/versions/v2",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2025-01-01T12:00:04Z",
          "modifiedat": "2025-01-01T12:00:04Z",
          "contenttype": "application/json",
          "file": {
            "hello": "world"
          }
        }
      },
      "versionscount": 2
    },
    "fx": {
      "fileid": "fx",
      "self": "#/files/fx",
      "xid": "/dirs/d1/files/fx",

      "metaurl": "#/files/fx/meta",
      "meta": {
        "fileid": "fx",
        "self": "#/files/fx/meta",
        "xid": "/dirs/d1/files/fx/meta",
        "xref": "/dirs/d1/files/f1"
      }
    }
  },
  "filescount": 2
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?doc&inline=*", ``, 200, `{
  "f1": {
    "fileid": "f1",
    "self": "#/f1",
    "xid": "/dirs/d1/files/f1",

    "metaurl": "#/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "#/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:04Z",
      "readonly": false,
      "compatibility": "none",

      "defaultversionid": "v2",
      "defaultversionurl": "#/f1/versions/v2",
      "defaultversionsticky": false
    },
    "versionsurl": "#/f1/versions",
    "versions": {
      "v1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "#/f1/versions/v1",
        "xid": "/dirs/d1/files/f1/versions/v1",
        "epoch": 1,
        "isdefault": false,
        "createdat": "2025-01-01T12:00:02Z",
        "modifiedat": "2025-01-01T12:00:02Z",
        "contenttype": "application/json",
        "file": {
          "hello": "world"
        }
      },
      "v2": {
        "fileid": "f1",
        "versionid": "v2",
        "self": "#/f1/versions/v2",
        "xid": "/dirs/d1/files/f1/versions/v2",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2025-01-01T12:00:04Z",
        "modifiedat": "2025-01-01T12:00:04Z",
        "contenttype": "application/json",
        "file": {
          "hello": "world"
        }
      }
    },
    "versionscount": 2
  },
  "fx": {
    "fileid": "fx",
    "self": "#/fx",
    "xid": "/dirs/d1/files/fx",

    "metaurl": "#/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "#/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "xref": "/dirs/d1/files/f1"
    }
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?doc&inline=*", ``, 200, `{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1",

  "metaurl": "#/meta",
  "meta": {
    "fileid": "f1",
    "self": "#/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:04Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2",
    "defaultversionurl": "#/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "#/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "#/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",
      "contenttype": "application/json",
      "file": {
        "hello": "world"
      }
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "#/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:04Z",
      "modifiedat": "2025-01-01T12:00:04Z",
      "contenttype": "application/json",
      "file": {
        "hello": "world"
      }
    }
  },
  "versionscount": 2
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta?doc&inline=*", ``, 200, `{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:04Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
  "defaultversionsticky": false
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx?doc&inline=*", ``, 200, `{
  "fileid": "fx",
  "self": "#/",
  "xid": "/dirs/d1/files/fx",

  "metaurl": "#/meta",
  "meta": {
    "fileid": "fx",
    "self": "#/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta?doc&inline=*", ``, 200, `{
  "fileid": "fx",
  "self": "#/",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?doc&inline=*", ``, 200, `{
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "#/v1",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "contenttype": "application/json",
    "file": {
      "hello": "world"
    }
  },
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "#/v2",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2025-01-01T12:00:04Z",
    "modifiedat": "2025-01-01T12:00:04Z",
    "contenttype": "application/json",
    "file": {
      "hello": "world"
    }
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1?doc&inline=*", ``, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": false,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",
  "contenttype": "application/json",
  "file": {
    "hello": "world"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx/versions?doc", ``, 400,
		"'doc' flag not allowed on xref'd Versions\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx/versions/v1?doc", ``, 400,
		"'doc' flag not allowed on xref'd Versions\n")

	// Just some filtering too for fun

	// Make sure that meta.defaultversionurl changes between absolute and
	// relative based on whether the defaultversion appears in "versions"

	// Notice "meta" now appears after "versions" and
	// defaultversionurl is absolute
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?doc&inline&"+
		"filter=versions.versionid=v1", "", 200, `{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1",

  "metaurl": "#/meta",
  "versionsurl": "#/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "#/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2025-01-01T12:00:01Z",
      "modifiedat": "2025-01-01T12:00:01Z",
      "contenttype": "application/json",
      "file": {
        "hello": "world"
      }
    }
  },
  "versionscount": 1,
  "meta": {
    "fileid": "f1",
    "self": "#/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  }
}
`)

	// defaultversionurl is relative this time
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?doc&inline&"+
		"filter=versions.versionid=v2", "", 200, `{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1",

  "metaurl": "#/meta",
  "versionsurl": "#/versions",
  "versions": {
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "#/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",
      "contenttype": "application/json",
      "file": {
        "hello": "world"
      }
    }
  },
  "versionscount": 1,
  "meta": {
    "fileid": "f1",
    "self": "#/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2",
    "defaultversionurl": "#/versions/v2",
    "defaultversionsticky": false
  }
}
`)

	// check full output + filtering
	code, fullBody = xGET(t, "export?filter=dirs.files.versions.versionid=v2&inline=*")
	xCheckEqual(t, "", code, 200)
	code, manualBody = xGET(t, "?doc&inline=*&filter=dirs.files.versions.versionid=v2")
	xCheckEqual(t, "", code, 200)
	xCheckEqual(t, "", fullBody, manualBody)

	// Notice that "meta" moved down to after the Versions collection

	xCheckEqual(t, "", fullBody, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportBasic",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "#/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "#/dirs/d1/files/f1",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "#/dirs/d1/files/f1/meta",
          "versionsurl": "#/dirs/d1/files/f1/versions",
          "versions": {
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "#/dirs/d1/files/f1/versions/v2",
              "xid": "/dirs/d1/files/f1/versions/v2",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:04Z",
              "modifiedat": "2025-01-01T12:00:04Z",
              "contenttype": "application/json",
              "file": {
                "hello": "world"
              }
            }
          },
          "versionscount": 1,
          "meta": {
            "fileid": "f1",
            "self": "#/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 2,
            "createdat": "2025-01-01T12:00:02Z",
            "modifiedat": "2025-01-01T12:00:04Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v2",
            "defaultversionurl": "#/dirs/d1/files/f1/versions/v2",
            "defaultversionsticky": false
          }
        },
        "fx": {
          "fileid": "fx",
          "self": "#/dirs/d1/files/fx",
          "xid": "/dirs/d1/files/fx",

          "metaurl": "#/dirs/d1/files/fx/meta",
          "meta": {
            "fileid": "fx",
            "self": "#/dirs/d1/files/fx/meta",
            "xid": "/dirs/d1/files/fx/meta",
            "xref": "/dirs/d1/files/f1"
          }
        }
      },
      "filescount": 2
    }
  },
  "dirscount": 1
}
`)

	code, fullBody = xGET(t, "export?filter=dirs.files.versions.versionid=vx&inline=*")
	xCheckEqual(t, "", code, 404)
	code, manualBody = xGET(t, "?doc&inline=*&filter=dirs.files.versions.versionid=vx")
	xCheckEqual(t, "", code, 404)
	xCheckEqual(t, "", fullBody, manualBody)
	xCheckEqual(t, "", fullBody, "Not found\n")

	code, fullBody = xGET(t, "export?filter=dirs.files.versions.versionid=v2,dirs.files.fileid=fx&inline=*")
	xCheckEqual(t, "", code, 200)
	code, manualBody = xGET(t, "?doc&inline=*&filter=dirs.files.versions.versionid=v2,dirs.files.fileid=fx")
	xCheckEqual(t, "", code, 200)
	xCheckEqual(t, "", fullBody, manualBody)

	xCheckEqual(t, "", fullBody, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportBasic",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "#/dirs/d1/files",
      "files": {
        "fx": {
          "fileid": "fx",
          "self": "#/dirs/d1/files/fx",
          "xid": "/dirs/d1/files/fx",

          "metaurl": "#/dirs/d1/files/fx/meta",
          "meta": {
            "fileid": "fx",
            "self": "#/dirs/d1/files/fx/meta",
            "xid": "/dirs/d1/files/fx/meta",
            "xref": "/dirs/d1/files/f1"
          }
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)

	// Make sure that we only move "meta" after "versions" if we actually
	// inline "versions", even if there are filters
	xHTTP(t, reg, "GET",
		"/dirs/d1/files?doc&inline=meta&filter=versions.versionid=v2",
		"", 200, `{
  "f1": {
    "fileid": "f1",
    "self": "#/f1",
    "xid": "/dirs/d1/files/f1",

    "metaurl": "#/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "#/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:01Z",
      "modifiedat": "2025-01-01T12:00:02Z",
      "readonly": false,
      "compatibility": "none",

      "defaultversionid": "v2",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "self": "#/fx",
    "xid": "/dirs/d1/files/fx",

    "metaurl": "#/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "#/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "xref": "/dirs/d1/files/f1"
    }
  }
}
`)

	// Make sure that ?doc doesn't turn on ?inline by mistake.
	// At one point ?export (?doc) implied ?inline=*

	xHTTP(t, reg, "GET", "?doc", ``, 200,
		`{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportBasic",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs?doc", ``, 200,
		`{
  "d1": {
    "dirid": "d1",
    "self": "#/d1",
    "xid": "/dirs/d1",
    "epoch": 2,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 2
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1?doc", ``, 200,
		`{
  "dirid": "d1",
  "self": "#/",
  "xid": "/dirs/d1",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 2
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?doc", ``, 200,
		`{
  "f1": {
    "fileid": "f1",
    "self": "#/f1",
    "xid": "/dirs/d1/files/f1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 2
  },
  "fx": {
    "fileid": "fx",
    "self": "#/fx",
    "xid": "/dirs/d1/files/fx",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?doc", ``, 200,
		`{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta?doc", ``, 200,
		`{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
  "defaultversionsticky": false
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?doc", ``, 200,
		`{
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "#/v1",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "contenttype": "application/json"
  },
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "#/v2",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "contenttype": "application/json"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1?doc", ``, 200,
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": false,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "contenttype": "application/json"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx?doc", ``, 200,
		`{
  "fileid": "fx",
  "self": "#/",
  "xid": "/dirs/d1/files/fx",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta"
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta?doc", ``, 200,
		`{
  "fileid": "fx",
  "self": "#/",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1"
}
`)

	// Test some error cases. Make sure ?doc doesn't change our
	// error checking logic

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1/foo?doc", ``, 404,
		"URL is too long\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx/versions/v1/foo?doc", ``, 404,
		"URL is too long\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/vx?doc", ``, 404,
		"Not found\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/fz/versions?doc", ``, 404,
		"\"dirs/d1/files/fz\" not found\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/fz?doc", ``, 404,
		"Not found\n")

	xHTTP(t, reg, "GET", "/dirs/dx/files?doc", ``, 404,
		"\"dirs/dx\" not found\n")

	xHTTP(t, reg, "GET", "/dirs/d1/filesx?doc", ``, 404,
		"Unknown Resource type: filesx\n")

	xHTTP(t, reg, "GET", "/dirs/dx?doc", ``, 404,
		"Not found\n")

	xHTTP(t, reg, "GET", "/dirsx?doc", ``, 404,
		"Unknown Group type: dirsx\n")

	xHTTP(t, reg, "GET", "/dirs/dx/files/fz/versions/vx?doc", ``, 404,
		"Not found\n")

}

func TestExportURLs(t *testing.T) {
	reg := NewRegistry("TestExportURLs")
	defer PassDeleteReg(t, reg)

	gm, _, err := reg.Model.CreateModels("dirs", "dir", "files", "file")
	xNoErr(t, err)
	_, err = gm.AddResourceModelSimple("schemas", "schema")
	xNoErr(t, err)

	xHTTP(t, reg, "GET", "/?doc", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0
}
`)

	xHTTP(t, reg, "GET", "/?doc&inline=dirs", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",

  "dirsurl": "#/dirs",
  "dirs": {},
  "dirscount": 0
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1", "{}", 201, `*`)

	xHTTP(t, reg, "GET", "/?doc&inline=dirs", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "filescount": 0,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs?doc&inline=files", "", 200, `{
  "d1": {
    "dirid": "d1",
    "self": "#/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:02Z",

    "filesurl": "#/d1/files",
    "files": {},
    "filescount": 0,
    "schemasurl": "http://localhost:8181/dirs/d1/schemas",
    "schemascount": 0
  }
}
`)

	xHTTP(t, reg, "GET", "/?doc&inline=dirs.files", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:02Z",

      "filesurl": "#/dirs/d1/files",
      "files": {},
      "filescount": 0,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs?doc", "", 200, `{
  "d1": {
    "dirid": "d1",
    "self": "#/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:02Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 0,
    "schemasurl": "http://localhost:8181/dirs/d1/schemas",
    "schemascount": 0
  }
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", "", 201, ``)

	xHTTP(t, reg, "GET", "/?doc&inline=dirs.files", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "#/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "#/dirs/d1/files/f1",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versionscount": 1
        }
      },
      "filescount": 1,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?doc&inline=dirs.files.meta", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "#/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "#/dirs/d1/files/f1",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "#/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "#/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2025-01-01T12:00:03Z",
            "modifiedat": "2025-01-01T12:00:03Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versionscount": 1
        }
      },
      "filescount": 1,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?doc&inline=dirs.files.versions", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "#/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "#/dirs/d1/files/f1",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "versionsurl": "#/dirs/d1/files/f1/versions",
          "versions": {
            "1": {
              "fileid": "f1",
              "versionid": "1",
              "self": "#/dirs/d1/files/f1/versions/1",
              "xid": "/dirs/d1/files/f1/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:03Z",
              "modifiedat": "2025-01-01T12:00:03Z"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?doc&inline=dirs.files.versions,dirs.files.meta", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "#/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "#/dirs/d1/files/f1",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "#/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "#/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2025-01-01T12:00:03Z",
            "modifiedat": "2025-01-01T12:00:03Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "1",
            "defaultversionurl": "#/dirs/d1/files/f1/versions/1",
            "defaultversionsticky": false
          },
          "versionsurl": "#/dirs/d1/files/f1/versions",
          "versions": {
            "1": {
              "fileid": "f1",
              "versionid": "1",
              "self": "#/dirs/d1/files/f1/versions/1",
              "xid": "/dirs/d1/files/f1/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:03Z",
              "modifiedat": "2025-01-01T12:00:03Z"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?doc&inline=versions,meta", "", 200, `{
  "f1": {
    "fileid": "f1",
    "self": "#/f1",
    "xid": "/dirs/d1/files/f1",

    "metaurl": "#/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "#/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:03Z",
      "modifiedat": "2025-01-01T12:00:03Z",
      "readonly": false,
      "compatibility": "none",

      "defaultversionid": "1",
      "defaultversionurl": "#/f1/versions/1",
      "defaultversionsticky": false
    },
    "versionsurl": "#/f1/versions",
    "versions": {
      "1": {
        "fileid": "f1",
        "versionid": "1",
        "self": "#/f1/versions/1",
        "xid": "/dirs/d1/files/f1/versions/1",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2025-01-01T12:00:03Z",
        "modifiedat": "2025-01-01T12:00:03Z"
      }
    },
    "versionscount": 1
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?doc&inline=meta", "", 200, `{
  "f1": {
    "fileid": "f1",
    "self": "#/f1",
    "xid": "/dirs/d1/files/f1",

    "metaurl": "#/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "#/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2025-01-01T12:00:03Z",
      "modifiedat": "2025-01-01T12:00:03Z",
      "readonly": false,
      "compatibility": "none",

      "defaultversionid": "1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 1
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?doc&inline=versions", "", 200, `{
  "f1": {
    "fileid": "f1",
    "self": "#/f1",
    "xid": "/dirs/d1/files/f1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "#/f1/versions",
    "versions": {
      "1": {
        "fileid": "f1",
        "versionid": "1",
        "self": "#/f1/versions/1",
        "xid": "/dirs/d1/files/f1/versions/1",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2025-01-01T12:00:03Z",
        "modifiedat": "2025-01-01T12:00:03Z"
      }
    },
    "versionscount": 1
  }
}
`)

	xHTTP(t, reg, "GET", "/?doc&inline=dirs.files.meta", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "#/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "#/dirs/d1/files/f1",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "#/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "#/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2025-01-01T12:00:03Z",
            "modifiedat": "2025-01-01T12:00:03Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versionscount": 1
        }
      },
      "filescount": 1,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?doc&inline=dirs.files.versions", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "#/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "#/dirs/d1/files/f1",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "versionsurl": "#/dirs/d1/files/f1/versions",
          "versions": {
            "1": {
              "fileid": "f1",
              "versionid": "1",
              "self": "#/dirs/d1/files/f1/versions/1",
              "xid": "/dirs/d1/files/f1/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:03Z",
              "modifiedat": "2025-01-01T12:00:03Z"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1,
      "schemasurl": "http://localhost:8181/dirs/d1/schemas",
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta?doc", "", 200, `{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:03Z",
  "modifiedat": "2025-01-01T12:00:03Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?doc", "", 200, `{
  "1": {
    "fileid": "f1",
    "versionid": "1",
    "self": "#/1",
    "xid": "/dirs/d1/files/f1/versions/1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2025-01-01T12:00:03Z",
    "modifiedat": "2025-01-01T12:00:03Z"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/1?doc", "", 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/versions/1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:03Z",
  "modifiedat": "2025-01-01T12:00:03Z"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/fx?doc&inline=*", "", 200, `{
  "fileid": "fx",
  "self": "#/",
  "xid": "/dirs/d1/files/fx",

  "metaurl": "#/meta",
  "meta": {
    "fileid": "fx",
    "self": "#/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1"
  }
}
`)

	// One file GET of everything

	xHTTP(t, reg, "GET", "/?doc&inline=*", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestExportURLs",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

  "dirsurl": "#/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "#/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 3,
      "createdat": "2025-01-01T12:00:02Z",
      "modifiedat": "2025-01-01T12:00:03Z",

      "filesurl": "#/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "self": "#/dirs/d1/files/f1",
          "xid": "/dirs/d1/files/f1",

          "metaurl": "#/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "#/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2025-01-01T12:00:04Z",
            "modifiedat": "2025-01-01T12:00:04Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "1",
            "defaultversionurl": "#/dirs/d1/files/f1/versions/1",
            "defaultversionsticky": false
          },
          "versionsurl": "#/dirs/d1/files/f1/versions",
          "versions": {
            "1": {
              "fileid": "f1",
              "versionid": "1",
              "self": "#/dirs/d1/files/f1/versions/1",
              "xid": "/dirs/d1/files/f1/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2025-01-01T12:00:04Z",
              "modifiedat": "2025-01-01T12:00:04Z"
            }
          },
          "versionscount": 1
        },
        "fx": {
          "fileid": "fx",
          "self": "#/dirs/d1/files/fx",
          "xid": "/dirs/d1/files/fx",

          "metaurl": "#/dirs/d1/files/fx/meta",
          "meta": {
            "fileid": "fx",
            "self": "#/dirs/d1/files/fx/meta",
            "xid": "/dirs/d1/files/fx/meta",
            "xref": "/dirs/d1/files/f1"
          }
        }
      },
      "filescount": 2,
      "schemasurl": "#/dirs/d1/schemas",
      "schemas": {},
      "schemascount": 0
    }
  },
  "dirscount": 1
}
`)

}

func TestExportNoDoc(t *testing.T) {
	reg := NewRegistry("TestExportBasic")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1?doc", "{}", 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z"
}
`)

	// Make sure there's no $details
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1?doc&inline=meta", "{}", 200, `{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1",

  "metaurl": "#/meta",
  "meta": {
    "fileid": "f1",
    "self": "#/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-23T23:07:14.527627972Z",
    "modifiedat": "2025-01-23T23:07:14.527627972Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// No $default
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1?doc&inline", "{}", 200, `{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1",

  "metaurl": "#/meta",
  "meta": {
    "fileid": "f1",
    "self": "#/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-23T23:08:08.330305606Z",
    "modifiedat": "2025-01-23T23:08:08.330305606Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v1",
    "defaultversionurl": "#/versions/v1",
    "defaultversionsticky": false
  },
  "versionsurl": "#/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "#/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 3,
      "isdefault": true,
      "createdat": "2025-01-23T23:08:08.330305606Z",
      "modifiedat": "2025-01-23T23:08:08.390182537Z"
    }
  },
  "versionscount": 1
}
`)

}
