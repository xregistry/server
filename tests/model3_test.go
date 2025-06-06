package tests

import (
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestModelXImportErrors(t *testing.T) {
	reg := NewRegistry("TestModelXImportErrors")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "g1p": {
          "singular": "g1s",
          "resources": { "r1p": { "singular": "r1s" } }
        },
        "g2p": {
          "singular": "g2s",
          "ximportresources": [ "/g1p" ]
        }
      }
    }`, 400, `Group "g2p" has an invalid "ximportresources" value (/g1p), must be of the form "/Group/Resource"
`)

	xHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "g1p": {
          "singular": "g1s",
          "resources": { "r1p": { "singular": "r1s" } }
        },
        "g2p": {
          "singular": "g2s",
          "ximportresources": [ "/gxx/xxx" ]
        }
      }
    }`, 400, `Group "g2p" references a non-existing Group "gxx"
`)

	xHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "g1p": {
          "singular": "g1s",
          "resources": { "r1p": { "singular": "r1s" } }
        },
        "g2p": {
          "singular": "g2s",
          "ximportresources": [ "/g1p/xxx" ]
        }
      }
    }`, 400, `Group "g2p" references a non-existing Resource "/g1p/xxx"
`)

	xHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "g1p": {
          "singular": "g1s",
          "resources": { "r1p": { "singular": "r1p" } }
        },
        "g2p": {
          "singular": "g2s"
        }
      }
    }`, 400, `Resource "r1p" has same value for "plural" and "singular"
`)

	xHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "g1p": {
          "singular": "g1s",
          "resources": { 
            "r1p": { "singular": "r1s" },
            "r2p": { "singular": "r1s" }
          }
        },
        "g2p": {
          "singular": "g2s"
        }
      }
    }`, 400, `Group "g1p" has a Resource "r2p" that has a duplicate "singular" name "r1s"
`)

	xHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "g1p": {
          "singular": "g1s",
          "resources": { "r1p": { "singular": "r1s" } }
        },
        "g2p": {
          "singular": "g2s",
          "ximportresources": [ "/g1p/r1p", "/g1p/r1p" ]
        }
      }
    }`, 400, `Group "g2p" has a duplicate Resource "plural" name "r1p"
`)

	xHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "g1p": {
          "singular": "g1s",
          "resources": {
            "r1p": { "singular": "r1s" },
            "r2p": { "singular": "R1S" }
          }
        },
        "g2p": {
          "singular": "g2s"
        }
      }
    }`, 400, `Invalid model type name "R1S", must match: ^[a-z_][a-z_0-9]{0,57}$
`)

	xHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "g1p": {
          "singular": "g1s",
          "resources": {
            "r1p": { "singular": "r1s" }
          }
        },
        "g2p": {
          "singular": "g2s",
          "ximportresources": [ "/g1p/r1p" ]
        },
        "g3p": {
          "singular": "g3s",
          "ximportresources": [ "/g2p/r1p" ]
        }
      }
    }`, 400, `Group "g3p" references an imported Resource "/g2p/r1p", try using "/g1p/r1p" instead
`)

	xHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "g1p": {
          "singular": "g1s",
          "ximportresources": [ "/g1p/r1p" ],
          "resources": {
            "r1p": { "singular": "r1s" }
          }
        }
      }
    }`, 400, `Group "g1p" has a bad "ximportresources" value (/g1p/r1p), it can't reference its own Group
`)

}

func TestModelXImport(t *testing.T) {
	reg := NewRegistry("TestModel")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "g1p": {
          "singular": "g1s",
          "resources": {
            "r1p": { "singular": "r1s" },
            "r2p": { "singular": "r2s" }
          }
        },
        "g2p": {
          "singular": "g2s",
          "ximportresources": [ "/g1p/r1p" ]
        }
      }
    }`, 200, "*")

	xHTTP(t, reg, "PUT", "/g1p/g1/r1p/r1", "{}", 201, "*")
	xHTTP(t, reg, "PUT", "/g2p/g1/r1p/r1", "{}", 201, "*")
	xHTTP(t, reg, "PUT", "/g2p/g1/r2p/r2", "{}", 404, "*")

	// Erase everything, including the model itself
	xHTTP(t, reg, "DELETE", "/g1p", "", 204, "*")
	xHTTP(t, reg, "DELETE", "/g2p", "", 204, "*")
	xHTTP(t, reg, "PUT", "/modelsource", `{}`, 200, "*")

	xHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "g1p": {
          "singular": "g1s",
          "ximportresources": [ "/g2p/g2r2p" ],
          "resources": {
            "r1p": { "singular": "r1s" },
            "r2p": { "singular": "r2s" }
          }
        },
        "g2p": {
          "singular": "g2s",
          "ximportresources": [ "/g1p/r1p", "/g1p/r2p" ],
          "resources": {
            "g2r2p": { "singular": "g2r2s" }
          }
        }
      }
    }`, 200, "*")

	xHTTP(t, reg, "PUT", "/g1p/g1/r1p/r1", "{}", 201, "*")
	xHTTP(t, reg, "PUT", "/g1p/g1/r2p/r1", "{}", 201, "*")
	xHTTP(t, reg, "PUT", "/g2p/g1/r1p/r1", "{}", 201, "*")
	xHTTP(t, reg, "PUT", "/g2p/g1/r2p/r1", "{}", 201, "*")
	xHTTP(t, reg, "PUT", "/g2p/g1/g2r2p/r1", "{}", 201, "*")

	xHTTP(t, reg, "PUT", "/g2p/g1/r2p/r2/meta", `{"xref":"/g1p/g1/r1p/r1"}`,
		400, `'xref' "/g1p/g1/r1p/r1" must point to a Resource of type "/g1p/r2p" not "/g1p/r1p"
`)
}

func TestModelResourceAttrs(t *testing.T) {
	reg := NewRegistry("TestModelResourceAttrs")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, false)

	_, err := rm.AddResourceAttr("rstring", STRING)
	xNoErr(t, err)

	_, err = rm.AddResourceAttrMap("rmap", registry.NewItemType(STRING))
	xNoErr(t, err)

	_, err = rm.AddResourceAttrObj("robj")
	xNoErr(t, err)

	_, err = rm.AddResourceAttrArray("rarray", registry.NewItemType(INTEGER))
	xNoErr(t, err)

	xNoErr(t, reg.SaveModel())

	xHTTP(t, reg, "GET", "/model", "{}", 200, `{
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
          "hasdocument": false,
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
            "rarray": {
              "name": "rarray",
              "type": "array",
              "item": {
                "type": "integer"
              }
            },
            "rmap": {
              "name": "rmap",
              "type": "map",
              "item": {
                "type": "string"
              }
            },
            "robj": {
              "name": "robj",
              "type": "object"
            },
            "rstring": {
              "name": "rstring",
              "type": "string"
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
}
`)

	xHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "dirs": {
          "singular": "dir",
          "resources": {
            "files": {
              "singular": "file",
              "resourceattributes": {
                "myattr": {
                  "type": "string"
                }
              }
            }
          }
        }
      }
    }`, 200, `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "resourceattributes": {
            "myattr": {
              "type": "string"
            }
          }
        }
      }
    }
  }
}
`)

	xHTTP(t, reg, "GET", "/model", "{}", 200, `{
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
            "myattr": {
              "name": "myattr",
              "type": "string"
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
}
`)

}
