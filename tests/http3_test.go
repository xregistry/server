package tests

import (
	"testing"

	. "github.com/xregistry/server/common"
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
    },
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
              "ancestor": "666",
              "filebase64": ""
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

func TestHTTPModelSource(t *testing.T) {
	reg := NewRegistry("TestHTTPModelSource")
	defer PassDeleteReg(t, reg)

	// Make sure 'model' is ignored
	// Make sure we process "modelsource" before we process the data
	xHTTP(t, reg, "PUT", "/", `{
  "model": { "ignore": "me" },
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "resources": {
          "files": {
            "singular": "file"
          }
        }
      }
    }
  },
  "dirs": {
    "d1": {
      "files": {
       "f1": {
         "file": "hello world"
       }
      }
    }
  }
}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPModelSource",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "PUT", "/?inline=modelsource,model", `{
  "model": { "ignore": "me" },
  "modelsource": {
    "groups": {
      "dirs1": {
        "singular": "dir",
        "resources": {
          "files": {
            "singular": "file"
          }
        }
      }
    }
  },
  "dirs1": {
    "d1": {
      "files": {
       "f1": {
         "file": "hello world"
       }
      }
    }
  }
}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPModelSource",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

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
      "dirs1url": {
        "name": "dirs1url",
        "type": "url",
        "readonly": true,
        "immutable": true,
        "required": true
      },
      "dirs1count": {
        "name": "dirs1count",
        "type": "uinteger",
        "readonly": true,
        "required": true
      },
      "dirs1": {
        "name": "dirs1",
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
      "dirs1": {
        "plural": "dirs1",
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
  "modelsource": {
    "groups": {
      "dirs1": {
        "singular": "dir",
        "resources": {
          "files": {
            "singular": "file"
          }
        }
      }
    }
  },

  "dirs1url": "http://localhost:8181/dirs1",
  "dirs1count": 1
}
`)

	xHTTP(t, reg, "PUT", "/?inline=model,modelsource", `{
  "model": { "ignore": "me" },
  "modelsource": {}
}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPModelSource",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

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
      }
    }
  },
  "modelsource": {}
}
`)

	xHTTP(t, reg, "PUT", "/", `{
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "resources": {
          "files": {
            "singular": "file"
          }
        }
      }
    }
  }
}`, 200, `*`)

	// Notice "null" means erase model
	xHTTP(t, reg, "PUT", "/?inline=model,modelsource", `{
  "model": { "ignore": "me" },
  "modelsource": null
}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPModelSource",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 6,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

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
      }
    }
  },
  "modelsource": {}
}
`)

	// Some errors
	xHTTP(t, reg, "POST", "/modelsource", `{}`, 405,
		"POST not allowed on '/modelsource'\n")
	xHTTP(t, reg, "PATCH", "/modelsource", `{}`, 405,
		"PATCH not allowed on '/modelsource'\n")

	xHTTP(t, reg, "PATCH", "/?inline=modelsource", `{
  "modelsource": {"groups": { "foos": { "singular": "foo"}}}}
`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPModelSource",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 7,
  "createdat": "2025-05-29T21:12:41.262020774Z",
  "modifiedat": "2025-05-29T21:12:41.399898946Z",

  "modelsource": {
    "groups": {
      "foos": {
        "singular": "foo"
      }
    }
  },

  "foosurl": "http://localhost:8181/foos",
  "fooscount": 0
}
`)

	xHTTP(t, reg, "POST", "/", `{
  "modelsource": {}
}`, 400,
		`POST / only allows Group types to be specified. "modelsource" is invalid
`)
}

func TestHTTPSort(t *testing.T) {
	reg := NewRegistry("TestHTTPSort")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "PUT", "/", `{
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "attributes": {
          "mybool": { "type": "boolean" },
          "myfloat": { "type": "decimal" },
          "myint": { "type": "integer" },
          "mystr": { "type": "string" },
          "myobj": {
            "type": "object",
            "attributes": {
              "*": { "type": "any" }
            }
          },
          "*": { "type": "any" }
        },
        "resources": {
          "files": {
            "singular": "file",
            "attributes": {
              "mybool": { "type": "boolean" },
              "myfloat": { "type": "decimal" },
              "myint": { "type": "integer" },
              "mystr": { "type": "string" },
              "myobj": {
                "type": "object",
                "attributes": {
                  "*": { "type": "any" }
                }
              },
              "*": { "type": "any" }
            }
          }
        }
      }
    }
  },
  "dirs": {
    "d1": {
      "name": "d1",
      "myfloat": 3.1,
      "myany": "a string",
      "myobj": { "foo": "bar" },
      "files": {
        "f1": {
          "name": "d1-f1",
          "versions": {
            "v1": { "name": "d1-f1-v1", "mybool": false },
            "v2": { "mybool": true },
            "v3": { "name": "d1-f1-v3", "description": "i'm d1-f1-v3" }
          }
        },
        "f2": {
          "name": "d1-f2",
          "versions": {
            "v2": { "name": "d1-f2-v2" },
            "v1": { "name": "zzzzzzzz" },
            "v3": { "name": "d1-f2-v3" }
          }
        }
      }
    },
    "d2": {
      "name": "d2",
      "myfloat": 1.3,
      "myany": 123,
      "myobj": { "foo": "zzz" }
    },
    "d3": {
      "name": "D1"
    }
  }
}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPSort",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 3
}
`)

	xHTTP(t, reg, "GET", "/?sort=epoch", ``, 400,
		"Can't sort on a non-collection results\n")
	xHTTP(t, reg, "GET", "/dirs/d1?sort=epoch", ``, 400,
		"Can't sort on a non-collection results\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?sort=epoch", ``, 400,
		"Can't sort on a non-collection results\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?sort=epoch", ``, 400,
		"Can't sort on a non-collection results\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1?sort=epoch",
		``, 400, "Can't sort on a non-collection results\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1$details?sort=epoch",
		``, 400, "Can't sort on a non-collection results\n")

	// Notice that d3(D1) comes before d2(d2) - sort by name case insensitively
	// Notice that d1 comes before d3 - same 'name' so sort by id (insensitive)
	xHTTP(t, reg, "GET", "/dirs?sort=name", ``, 200, `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "name": "d1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": "a string",
    "myfloat": 3.1,
    "myobj": {
      "foo": "bar"
    },

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 2
  },
  "d3": {
    "dirid": "d3",
    "self": "http://localhost:8181/dirs/d3",
    "xid": "/dirs/d3",
    "epoch": 1,
    "name": "D1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "filesurl": "http://localhost:8181/dirs/d3/files",
    "filescount": 0
  },
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "name": "d2",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": 123,
    "myfloat": 1.3,
    "myobj": {
      "foo": "zzz"
    },

    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs?sort=name=desc", ``, 200, `{
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "name": "d2",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": 123,
    "myfloat": 1.3,
    "myobj": {
      "foo": "zzz"
    },

    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  },
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "name": "d1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": "a string",
    "myfloat": 3.1,
    "myobj": {
      "foo": "bar"
    },

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 2
  },
  "d3": {
    "dirid": "d3",
    "self": "http://localhost:8181/dirs/d3",
    "xid": "/dirs/d3",
    "epoch": 1,
    "name": "D1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "filesurl": "http://localhost:8181/dirs/d3/files",
    "filescount": 0
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs?sort=myany", ``, 200, `{
  "d3": {
    "dirid": "d3",
    "self": "http://localhost:8181/dirs/d3",
    "xid": "/dirs/d3",
    "epoch": 1,
    "name": "D1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "filesurl": "http://localhost:8181/dirs/d3/files",
    "filescount": 0
  },
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "name": "d2",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": 123,
    "myfloat": 1.3,
    "myobj": {
      "foo": "zzz"
    },

    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  },
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "name": "d1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": "a string",
    "myfloat": 3.1,
    "myobj": {
      "foo": "bar"
    },

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 2
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs?sort=myany=desc", ``, 200, `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "name": "d1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": "a string",
    "myfloat": 3.1,
    "myobj": {
      "foo": "bar"
    },

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 2
  },
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "name": "d2",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": 123,
    "myfloat": 1.3,
    "myobj": {
      "foo": "zzz"
    },

    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  },
  "d3": {
    "dirid": "d3",
    "self": "http://localhost:8181/dirs/d3",
    "xid": "/dirs/d3",
    "epoch": 1,
    "name": "D1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "filesurl": "http://localhost:8181/dirs/d3/files",
    "filescount": 0
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs?sort=myfloat", ``, 200, `{
  "d3": {
    "dirid": "d3",
    "self": "http://localhost:8181/dirs/d3",
    "xid": "/dirs/d3",
    "epoch": 1,
    "name": "D1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "filesurl": "http://localhost:8181/dirs/d3/files",
    "filescount": 0
  },
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "name": "d2",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": 123,
    "myfloat": 1.3,
    "myobj": {
      "foo": "zzz"
    },

    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  },
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "name": "d1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": "a string",
    "myfloat": 3.1,
    "myobj": {
      "foo": "bar"
    },

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 2
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs?sort=myfloat=asc", ``, 200, `{
  "d3": {
    "dirid": "d3",
    "self": "http://localhost:8181/dirs/d3",
    "xid": "/dirs/d3",
    "epoch": 1,
    "name": "D1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "filesurl": "http://localhost:8181/dirs/d3/files",
    "filescount": 0
  },
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "name": "d2",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": 123,
    "myfloat": 1.3,
    "myobj": {
      "foo": "zzz"
    },

    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  },
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "name": "d1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": "a string",
    "myfloat": 3.1,
    "myobj": {
      "foo": "bar"
    },

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 2
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs?sort=myfloat=desc", ``, 200, `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "name": "d1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": "a string",
    "myfloat": 3.1,
    "myobj": {
      "foo": "bar"
    },

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 2
  },
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "name": "d2",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": 123,
    "myfloat": 1.3,
    "myobj": {
      "foo": "zzz"
    },

    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  },
  "d3": {
    "dirid": "d3",
    "self": "http://localhost:8181/dirs/d3",
    "xid": "/dirs/d3",
    "epoch": 1,
    "name": "D1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "filesurl": "http://localhost:8181/dirs/d3/files",
    "filescount": 0
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs?sort=myobj.foo", ``, 200, `{
  "d3": {
    "dirid": "d3",
    "self": "http://localhost:8181/dirs/d3",
    "xid": "/dirs/d3",
    "epoch": 1,
    "name": "D1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "filesurl": "http://localhost:8181/dirs/d3/files",
    "filescount": 0
  },
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "name": "d1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": "a string",
    "myfloat": 3.1,
    "myobj": {
      "foo": "bar"
    },

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 2
  },
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "name": "d2",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": 123,
    "myfloat": 1.3,
    "myobj": {
      "foo": "zzz"
    },

    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs?sort=myobj.foo=desc", ``, 200, `{
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "name": "d2",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": 123,
    "myfloat": 1.3,
    "myobj": {
      "foo": "zzz"
    },

    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  },
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "name": "d1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "myany": "a string",
    "myfloat": 3.1,
    "myobj": {
      "foo": "bar"
    },

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 2
  },
  "d3": {
    "dirid": "d3",
    "self": "http://localhost:8181/dirs/d3",
    "xid": "/dirs/d3",
    "epoch": 1,
    "name": "D1",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "filesurl": "http://localhost:8181/dirs/d3/files",
    "filescount": 0
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?sort=name", ``, 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",
    "epoch": 1,
    "name": "d1-f1-v3",
    "isdefault": true,
    "description": "i'm d1-f1-v3",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v2",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 3
  },
  "f2": {
    "fileid": "f2",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/f2$details",
    "xid": "/dirs/d1/files/f2",
    "epoch": 1,
    "name": "d1-f2-v3",
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v2",

    "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
    "versionscount": 3
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?sort=name=desc", ``, 200, `{
  "f2": {
    "fileid": "f2",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/f2$details",
    "xid": "/dirs/d1/files/f2",
    "epoch": 1,
    "name": "d1-f2-v3",
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v2",

    "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
    "versionscount": 3
  },
  "f1": {
    "fileid": "f1",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",
    "epoch": 1,
    "name": "d1-f1-v3",
    "isdefault": true,
    "description": "i'm d1-f1-v3",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v2",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 3
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?sort=description=desc", ``, 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",
    "epoch": 1,
    "name": "d1-f1-v3",
    "isdefault": true,
    "description": "i'm d1-f1-v3",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v2",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 3
  },
  "f2": {
    "fileid": "f2",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/f2$details",
    "xid": "/dirs/d1/files/f2",
    "epoch": 1,
    "name": "d1-f2-v3",
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v2",

    "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
    "versionscount": 3
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?sort=mybool", ``, 200, `{
  "v3": {
    "fileid": "f1",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3$details",
    "xid": "/dirs/d1/files/f1/versions/v3",
    "epoch": 1,
    "name": "d1-f1-v3",
    "isdefault": true,
    "description": "i'm d1-f1-v3",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v2"
  },
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "name": "d1-f1-v1",
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v1",
    "mybool": false
  },
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v1",
    "mybool": true
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?sort=mybool=desc", ``, 200, `{
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v1",
    "mybool": true
  },
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "name": "d1-f1-v1",
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v1",
    "mybool": false
  },
  "v3": {
    "fileid": "f1",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3$details",
    "xid": "/dirs/d1/files/f1/versions/v3",
    "epoch": 1,
    "name": "d1-f1-v3",
    "isdefault": true,
    "description": "i'm d1-f1-v3",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v2"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?sort=ancestor=desc", ``, 200, `{
  "v3": {
    "fileid": "f1",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3$details",
    "xid": "/dirs/d1/files/f1/versions/v3",
    "epoch": 1,
    "name": "d1-f1-v3",
    "isdefault": true,
    "description": "i'm d1-f1-v3",
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v2"
  },
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "name": "d1-f1-v1",
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v1",
    "mybool": false
  },
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v1",
    "mybool": true
  }
}
`)

}

func TestHTTPSortArray(t *testing.T) {
	reg := NewRegistry("TestHTTPSortArray")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "PUT", "/", `{
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "attributes": {
          "strs": {
            "type": "array",
            "item": {
              "type": "string"
            }
          }
        }
      }
    }
  },
  "dirs": {
    "d1": { "strs": [ "zzz", "bbb" ] },
    "d2": { "strs": [ "aaa", "bbb" ] }
  }
}
`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPSortArray",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 2
}
`)

	xHTTP(t, reg, "GET", "/dirs?sort=strs[0]", "", 200, `{
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "strs": [
      "aaa",
      "bbb"
    ]
  },
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "strs": [
      "zzz",
      "bbb"
    ]
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs?sort=strs[1]", "", 200, `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "strs": [
      "zzz",
      "bbb"
    ]
  },
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "strs": [
      "aaa",
      "bbb"
    ]
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs?sort=strs[0]=desc", "", 200, `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "strs": [
      "zzz",
      "bbb"
    ]
  },
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "strs": [
      "aaa",
      "bbb"
    ]
  }
}
`)

}

func TestHTTPJsonSchema(t *testing.T) {
	reg := NewRegistry("TestHTTPJsonSchema")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "PUT", "/", `{"$schema": "http://foo.com"}`,
		200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPJsonSchema",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)
	xHTTP(t, reg, "PUT", "/", `{"$schema": "http://foo.com", "name": "foo"}`,
		200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPJsonSchema",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "name": "foo",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	xHTTP(t, reg, "PATCH", "/", `{"$schema": "http://foo.com"}`,
		200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPJsonSchema",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "name": "foo",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)
	xHTTP(t, reg, "PATCH", "/", `{"$schema": "http://foo.com", "name": "zoo"}`,
		200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPJsonSchema",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "name": "zoo",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z"
}
`)

	xHTTP(t, reg, "PUT", "/capabilities", `{"$schema": "http://foo.com","apis":["*"],"mutable":["*"]}`,
		200, `{
  "apis": [
    "/capabilities",
    "/capabilitiesoffered",
    "/export",
    "/model",
    "/modelsource"
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

	xHTTP(t, reg, "PATCH", "/capabilities", `{"$schema": "http://foo.com","apis":["*"],"mutable":["*"]}`,
		200, `{
  "apis": [
    "/capabilities",
    "/capabilitiesoffered",
    "/export",
    "/model",
    "/modelsource"
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

	xHTTP(t, reg, "PUT", "/modelsource", `{"$schema": "http://foo.com"}`,
		200, `{
  "$schema": "http://foo.com"
}
`)
	xHTTP(t, reg, "PUT", "/modelsource",
		`{"$schema": "http://foo.com", "groups": {"dirs":{"singular":"dir","resources":{"files": {"singular": "file"}}}}}`, 200, `{
  "$schema": "http://foo.com",
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file"
        }
      }
    }
  }
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", `{"$schema": "http://foo.com", "name":"v1"}`,
		201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "name": "v1",
  "isdefault": true,
  "createdat": "2025-06-18T16:39:53.559486601Z",
  "modifiedat": "2025-06-18T16:39:53.559486601Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions/v1$details", `{"$schema": "http://foo.com", "name":"v11"}`,
		200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 2,
  "name": "v11",
  "isdefault": true,
  "createdat": "2025-06-18T16:39:53.559486601Z",
  "modifiedat": "2025-06-18T16:39:53.559486602Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{"$schema": "http://foo.com", "v2": {}}`,
		200, `{
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "v1"
  }
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1$details", `{"$schema": "http://foo.com"}`,
		201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "xid": "/dirs/d1/files/f1/versions/1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "v2"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{"$schema": "http://foo.com", "name": "foo"}`,
		200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "name": "foo",
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestor": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", `{"$schema": "http://foo.com", "name": "foo2"}`,
		200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "name": "foo2",
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestor": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files", `{"$schema": "http://foo.com"}`,
		200, `{}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files", `{"$schema": "http://foo.com", "f2":{}}`,
		200, `{
  "f2": {
    "fileid": "f2",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f2$details",
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
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1", `{"$schema": "http://foo.com"}`,
		200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 2
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1", `{"$schema": "http://foo.com", "name":"d1"}`,
		200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 4,
  "name": "d1",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 2
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1", `{"$schema": "http://foo.com", "name":"d11"}`,
		200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 5,
  "name": "d11",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 2
}
`)

	xHTTP(t, reg, "POST", "/dirs", `{"$schema": "http://foo.com"}`,
		200, `{}
`)

	xHTTP(t, reg, "POST", "/dirs", `{"$schema": "http://foo.com", "d2":{}}`,
		200, `{
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  }
}
`)

}

func TestHTTPModelEnum(t *testing.T) {
	reg := NewRegistry("TestHTTPModelEnum")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "PUT", "/", `{
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "attributes": {
          "strs": {
            "type": "object",
            "enum": [ 1 ]
          }
        }
      }
    }
  }
}
`, 400, `"groups.dirs.strs" is not a scalar, or an array of scalars, so "enum" is not allowed
`)

	xHTTP(t, reg, "PUT", "/", `{
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "attributes": {
          "strs": {
            "type": "integer",
            "enum": [ 1 ]
          }
        }
      }
    }
  }
  ,
  "dirs": {
    "d1": { "strs":  "2" }
  }
}
`, 400, `Attribute "strs" must be an integer
`)

	xHTTP(t, reg, "PUT", "/", `{
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "attributes": {
          "strs": {
            "type": "array",
            "item": {
              "type": "integer"
            },
            "enum": [ "1" ]
          }
        }
      }
    }
  }
}
`, 400, `"groups.dirs.strs" enum value "1" must be of type "integer"
`)

	xHTTP(t, reg, "PUT", "/", `{
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "attributes": {
          "strs": {
            "type": "array",
            "item": {
              "type": "integer"
            },
            "enum": [ 1 ]
          }
        }
      }
    }
  },
  "dirs": {
    "d1": { "strs": [ "abc" ] }
  }
}
`, 400, `Attribute "strs[0]" must be an integer
`)

	xHTTP(t, reg, "PUT", "/", `{
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "attributes": {
          "strs": {
            "type": "array",
            "item": {
              "type": "integer"
            },
            "enum": [ 1 ]
          }
        }
      }
    }
  },
  "dirs": {
    "d1": { "strs": 2 }
  }
}
`, 400, `Attribute "strs" must be an array
`)

	xHTTP(t, reg, "PUT", "/", `{
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "attributes": {
          "strs": {
            "type": "array",
            "item": {
              "type": "integer"
            },
            "enum": [ 1 ],
            "strict": false
          }
        }
      }
    }
  },
  "dirs": {
    "d1": { "strs": [ 2 ] }
  }
}
`, 200, `*`)

	xHTTP(t, reg, "PUT", "/", `{
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "attributes": {
          "strs": {
            "type": "array",
            "item": {
              "type": "integer"
            },
            "enum": [ 1 ]
          }
        }
      }
    }
  },
  "dirs": {
    "d1": { "strs": [ 2 ] }
  }
}
`, 400, `Attribute "strs[0]"(2) must be one of the enum values: 1
`)

	xHTTP(t, reg, "PUT", "/", `{
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "attributes": {
          "strs": {
            "type": "array",
            "item": {
              "type": "integer"
            },
            "enum": [ 2, 3 ]
          }
        }
      }
    }
  },
  "dirs": {
    "d1": { "strs": [ 3, 2 ] }
  }
}
`, 200, `*`)

	return

	xHTTP(t, reg, "PUT", "/", `{
  "modelsource": {
    "groups": {
      "dirs": {
        "singular": "dir",
        "attributes": {
          "strs": {
            "type": "array",
            "item": {
              "type": "string"
            },
            "enum": { 1 }
          }
        }
      }
    }
  },
  "dirs": {
    "d1": { "strs": [ "zzz", "bbb" ] },
    "d2": { "strs": [ "aaa", "bbb" ] }
  }
}
`, 200, `{
}
`)

}

func TestHTTPBinaryFlag(t *testing.T) {
	reg := NewRegistry("TestHTTPBinaryFlag")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	_, err = gm.AddResourceModelSimple("files", "file")

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details?inline=file", `{
  "file": { "attr": "value" }
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "file": {
    "attr": "value"
  },

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file&binary", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "filebase64": "ewogICJhdHRyIjogInZhbHVlIgp9",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1?inline=files.file&binary", ``, 200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "files": {
    "f1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1$details",
      "xid": "/dirs/d1/files/f1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
      "ancestor": "1",
      "contenttype": "application/json",
      "filebase64": "ewogICJhdHRyIjogInZhbHVlIgp9",

      "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
      "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
      "versionscount": 1
    }
  },
  "filescount": 1
}
`)
}
