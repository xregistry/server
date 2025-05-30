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
  "specversion": "1.0-rc1",
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
  "specversion": "1.0-rc1",
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
                  "docs": {
                    "name": "docs",
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
  "specversion": "1.0-rc1",
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
  "specversion": "1.0-rc1",
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
  "specversion": "1.0-rc1",
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
