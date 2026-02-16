package tests

import (
	"strings"
	"testing"

	. "github.com/xregistry/server/common"
)

func TestExportBasic(t *testing.T) {
	reg := NewRegistry("TestExportBasic")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details",
		`{"file": { "hello": "world" }}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2$details",
		`{"file": { "hello": "world" }}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx$details",
		`{"meta": { "xref": "/dirs/d1/files/f1" }}`, 201, `*`)

	// Full export - 2 different ways
	code, fullBody := XGET(t, "export")
	XEqual(t, "", code, 200)

	code, manualBody := XGET(t, "?doc&inline=*,capabilities,modelsource")
	XEqual(t, "", code, 200)
	XEqual(t, "", fullBody, manualBody)

	XEqual(t, "", fullBody, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestExportBasic",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

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
      "ignore",
      "inline",
      "setdefaultversionid",
      "sort",
      "specversion"
    ],
    "ignores": [
      "capabilities",
      "defaultversionid",
      "defaultversionsticky",
      "epoch",
      "modelsource",
      "readonly"
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
    "stickyversions": true,
    "versionmodes": [
      "createdat",
      "manual"
    ]
  },
  "modelsource": {},

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
              "ancestor": "v1",
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
              "ancestor": "v1",
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
	code, fullBody = XGET(t, "export?inline=*")
	XEqual(t, "", code, 200)
	code, manualBody = XGET(t, "?doc&inline=*")
	XEqual(t, "", code, 200)
	XEqual(t, "", fullBody, manualBody)

	XEqual(t, "", fullBody, `{
  "specversion": "`+SPECVERSION+`",
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
              "ancestor": "v1",
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
              "ancestor": "v1",
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
	code, fullBody = XGET(t, "export?inline=capabilities")
	XEqual(t, "", code, 200)
	code, manualBody = XGET(t, "?doc&inline=capabilities")
	XEqual(t, "", code, 200)
	XEqual(t, "", fullBody, manualBody)

	XEqual(t, "", fullBody, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestExportBasic",
  "self": "#/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",

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
      "ignore",
      "inline",
      "setdefaultversionid",
      "sort",
      "specversion"
    ],
    "ignores": [
      "capabilities",
      "defaultversionid",
      "defaultversionsticky",
      "epoch",
      "modelsource",
      "readonly"
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
    "stickyversions": true,
    "versionmodes": [
      "createdat",
      "manual"
    ]
  },

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	// Play with ?doc inline just model
	code, fullBody = XGET(t, "export?inline=model")
	XEqual(t, "", code, 200)
	code, manualBody = XGET(t, "?doc&inline=model")
	XEqual(t, "", code, 200)
	XEqual(t, "", fullBody, manualBody)

	XEqual(t, "", fullBody, `{
  "specversion": "`+SPECVERSION+`",
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
        "required": true
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
            "setdefaultversionsticky": true,
            "hasdocument": true,
            "singleversionroot": false,
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
              "ancestor": {
                "name": "ancestor",
                "type": "string",
                "required": true
              },
              "contenttype": {
                "name": "contenttype",
                "type": "string"
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
              "compatibilityauthority": {
                "name": "compatibilityauthority",
                "type": "string",
                "enum": [
                  "external",
                  "server"
                ],
                "strict": false
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
  },

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	// Play with ?doc not at root
	XHTTP(t, reg, "GET", "/dirs?doc&inline=*", ``, 200, `{
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
            "ancestor": "v1",
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
            "ancestor": "v1",
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

	XHTTP(t, reg, "GET", "/dirs/d1?doc&inline=*", ``, 200, `{
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
          "ancestor": "v1",
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
          "ancestor": "v1",
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

	XHTTP(t, reg, "GET", "/dirs/d1/files?doc&inline=*", ``, 200, `{
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
        "ancestor": "v1",
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
        "ancestor": "v1",
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

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?doc&inline=*", ``, 200, `{
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
      "ancestor": "v1",
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
      "ancestor": "v1",
      "contenttype": "application/json",
      "file": {
        "hello": "world"
      }
    }
  },
  "versionscount": 2
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta?doc&inline=*", ``, 200, `{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:04Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx?doc&inline=*", ``, 200, `{
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

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta?doc&inline=*", ``, 200, `{
  "fileid": "fx",
  "self": "#/",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?doc&inline=*", ``, 200, `{
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "#/v1",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2025-01-01T12:00:02Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "ancestor": "v1",
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
    "ancestor": "v1",
    "contenttype": "application/json",
    "file": {
      "hello": "world"
    }
  }
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1?doc&inline=*", ``, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": false,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",
  "ancestor": "v1",
  "contenttype": "application/json",
  "file": {
    "hello": "world"
  }
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/versions?doc", ``, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#cannot_doc_xref",
  "title": "Retrieving the document view of a Version for \"/dirs/d1/files/fx/versions\" is not allowed because it uses \"xref\".",
  "subject": "/dirs/d1/files/fx/versions",
  "source": "e4e59b8a76c4:registry:httpStuff:1756"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/versions/v1?doc", ``, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#cannot_doc_xref",
  "title": "Retrieving the document view of a Version for \"/dirs/d1/files/fx/versions/v1\" is not allowed because it uses \"xref\".",
  "subject": "/dirs/d1/files/fx/versions/v1",
  "source": "e4e59b8a76c4:registry:httpStuff:1723"
}
`)

	// Just some filtering too for fun

	// Make sure that meta.defaultversionurl changes between absolute and
	// relative based on whether the defaultversion appears in "versions"

	// Notice "meta" now appears after "versions" and
	// defaultversionurl is absolute
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?doc&inline&"+
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
      "ancestor": "v1",
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
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "defaultversionsticky": false
  }
}
`)

	// defaultversionurl is relative this time
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?doc&inline&"+
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
      "ancestor": "v1",
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
	code, fullBody = XGET(t, "export?filter=dirs.files.versions.versionid=v2&inline=*")
	XEqual(t, "", code, 200)
	code, manualBody = XGET(t, "?doc&inline=*&filter=dirs.files.versions.versionid=v2")
	XEqual(t, "", code, 200)
	XEqual(t, "", fullBody, manualBody)

	// Notice that "meta" moved down to after the Versions collection

	XEqual(t, "", fullBody, `{
  "specversion": "`+SPECVERSION+`",
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
              "ancestor": "v1",
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

	code, fullBody = XGET(t, "export?filter=dirs.files.versions.versionid=vx&inline=*")
	XEqual(t, "", code, 404)
	code, manualBody = XGET(t, "?doc&inline=*&filter=dirs.files.versions.versionid=vx")
	XEqual(t, "", code, 404)
	fullBody = strings.ReplaceAll(fullBody, "/export", "/")
	XEqual(t, "", fullBody, manualBody)
	XEqual(t, "", fullBody, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/) cannot be found.",
  "subject": "/",
  "source": ":registry:httpStuff:1730"
}
`)

	code, fullBody = XGET(t, "export?filter=dirs.files.versions.versionid=v2,dirs.files.fileid=fx&inline=*")
	XEqual(t, "", code, 200)
	code, manualBody = XGET(t, "?doc&inline=*&filter=dirs.files.versions.versionid=v2,dirs.files.fileid=fx")
	XEqual(t, "", code, 200)
	XEqual(t, "", fullBody, manualBody)

	XEqual(t, "", fullBody, `{
  "specversion": "`+SPECVERSION+`",
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
	XHTTP(t, reg, "GET",
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
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
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

	XHTTP(t, reg, "GET", "?doc", ``, 200,
		`{
  "specversion": "`+SPECVERSION+`",
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

	XHTTP(t, reg, "GET", "/dirs?doc", ``, 200,
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

	XHTTP(t, reg, "GET", "/dirs/d1?doc", ``, 200,
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

	XHTTP(t, reg, "GET", "/dirs/d1/files?doc", ``, 200,
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

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?doc", ``, 200,
		`{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta?doc", ``, 200,
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
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?doc", ``, 200,
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
    "ancestor": "v1",
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
    "ancestor": "v1",
    "contenttype": "application/json"
  }
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1?doc", ``, 200,
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": false,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "v1",
  "contenttype": "application/json"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx?doc", ``, 200,
		`{
  "fileid": "fx",
  "self": "#/",
  "xid": "/dirs/d1/files/fx",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta?doc", ``, 200,
		`{
  "fileid": "fx",
  "self": "#/",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1"
}
`)

	// Test some error cases. Make sure ?doc doesn't change our
	// error checking logic

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1/foo?doc", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/versions/v1/foo) cannot be found.",
  "subject": "/dirs/d1/files/f1/versions/v1/foo",
  "source": ":registry:info:678"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/versions/v1/foo?doc", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/fx/versions/v1/foo) cannot be found.",
  "subject": "/dirs/d1/files/fx/versions/v1/foo",
  "source": ":registry:info:678"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/vx?doc", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/versions/vx) cannot be found.",
  "subject": "/dirs/d1/files/f1/versions/vx",
  "source": ":registry:httpStuff:1730"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fz/versions?doc", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/fz) cannot be found.",
  "subject": "/dirs/d1/files/fz",
  "source": ":registry:httpStuff:1746"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fz?doc", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/fz) cannot be found.",
  "subject": "/dirs/d1/files/fz",
  "source": ":registry:httpStuff:1730"
}
`)

	XHTTP(t, reg, "GET", "/dirs/dx/files?doc", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/dx) cannot be found.",
  "subject": "/dirs/dx",
  "source": ":registry:httpStuff:1746"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/filesx?doc", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/filesx) cannot be found.",
  "detail": "Unknown Resource type: filesx.",
  "subject": "/dirs/d1/filesx",
  "source": ":registry:info:595"
}
`)

	XHTTP(t, reg, "GET", "/dirs/dx?doc", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/dx) cannot be found.",
  "subject": "/dirs/dx",
  "source": ":registry:httpStuff:1730"
}
`)

	XHTTP(t, reg, "GET", "/dirsx?doc", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirsx) cannot be found.",
  "detail": "Unknown Group type: dirsx.",
  "subject": "/dirsx",
  "source": ":registry:info:562"
}
`)

	XHTTP(t, reg, "GET", "/dirs/dx/files/fz/versions/vx?doc", ``, 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/dx/files/fz/versions/vx) cannot be found.",
  "subject": "/dirs/dx/files/fz/versions/vx",
  "source": ":registry:httpStuff:1730"
}
`)

}

func TestExportURLs(t *testing.T) {
	reg := NewRegistry("TestExportURLs")
	defer PassDeleteReg(t, reg)

	gm, _, err := reg.Model.CreateModels("dirs", "dir", "files", "file")
	XNoErr(t, err)
	_, err = gm.AddResourceModelSimple("schemas", "schema")
	XNoErr(t, err)

	XHTTP(t, reg, "GET", "/?doc", "", 200, `{
  "specversion": "`+SPECVERSION+`",
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

	XHTTP(t, reg, "GET", "/?doc&inline=dirs", "", 200, `{
  "specversion": "`+SPECVERSION+`",
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

	XHTTP(t, reg, "PUT", "/dirs/d1", "{}", 201, `*`)

	XHTTP(t, reg, "GET", "/?doc&inline=dirs", "", 200, `{
  "specversion": "`+SPECVERSION+`",
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

	XHTTP(t, reg, "GET", "/dirs?doc&inline=files", "", 200, `{
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

	XHTTP(t, reg, "GET", "/?doc&inline=dirs.files", "", 200, `{
  "specversion": "`+SPECVERSION+`",
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

	XHTTP(t, reg, "GET", "/dirs?doc", "", 200, `{
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

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", "", 201, ``)

	XHTTP(t, reg, "GET", "/?doc&inline=dirs.files", "", 200, `{
  "specversion": "`+SPECVERSION+`",
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

	XHTTP(t, reg, "GET", "/?doc&inline=dirs.files.meta", "", 200, `{
  "specversion": "`+SPECVERSION+`",
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
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
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

	XHTTP(t, reg, "GET", "/?doc&inline=dirs.files.versions", "", 200, `{
  "specversion": "`+SPECVERSION+`",
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
              "modifiedat": "2025-01-01T12:00:03Z",
              "ancestor": "1"
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

	XHTTP(t, reg, "GET", "/?doc&inline=dirs.files.versions,dirs.files.meta", "", 200, `{
  "specversion": "`+SPECVERSION+`",
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
              "modifiedat": "2025-01-01T12:00:03Z",
              "ancestor": "1"
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

	XHTTP(t, reg, "GET", "/dirs/d1/files?doc&inline=versions,meta", "", 200, `{
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
        "modifiedat": "2025-01-01T12:00:03Z",
        "ancestor": "1"
      }
    },
    "versionscount": 1
  }
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files?doc&inline=meta", "", 200, `{
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
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 1
  }
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files?doc&inline=versions", "", 200, `{
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
        "modifiedat": "2025-01-01T12:00:03Z",
        "ancestor": "1"
      }
    },
    "versionscount": 1
  }
}
`)

	XHTTP(t, reg, "GET", "/?doc&inline=dirs.files.meta", "", 200, `{
  "specversion": "`+SPECVERSION+`",
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
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
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

	XHTTP(t, reg, "GET", "/?doc&inline=dirs.files.versions", "", 200, `{
  "specversion": "`+SPECVERSION+`",
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
              "modifiedat": "2025-01-01T12:00:03Z",
              "ancestor": "1"
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

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta?doc", "", 200, `{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:03Z",
  "modifiedat": "2025-01-01T12:00:03Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?doc", "", 200, `{
  "1": {
    "fileid": "f1",
    "versionid": "1",
    "self": "#/1",
    "xid": "/dirs/d1/files/f1/versions/1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2025-01-01T12:00:03Z",
    "modifiedat": "2025-01-01T12:00:03Z",
    "ancestor": "1"
  }
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/1?doc", "", 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/versions/1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:03Z",
  "modifiedat": "2025-01-01T12:00:03Z",
  "ancestor": "1"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx?doc&inline=*", "", 200, `{
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

	XHTTP(t, reg, "GET", "/?doc&inline=*", "", 200, `{
  "specversion": "`+SPECVERSION+`",
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
              "modifiedat": "2025-01-01T12:00:04Z",
              "ancestor": "1",
              "filebase64": ""
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

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1?doc", "{}", 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	// Make sure there's no $details
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?doc&inline=meta", "{}", 200, `{
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
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?doc&inline", "{}", 200, `{
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
      "modifiedat": "2025-01-23T23:08:08.390182537Z",
      "ancestor": "v1"
    }
  },
  "versionscount": 1
}
`)

}
