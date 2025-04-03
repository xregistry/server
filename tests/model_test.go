package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/xregistry/server/registry"
)

func TestNoModel(t *testing.T) {
	reg := NewRegistry("TestNoModel")
	defer PassDeleteReg(t, reg)

	xCheckGet(t, reg, "/model", `{
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
  }
}
`)

	xCheckGet(t, reg, "?inline=model", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestNoModel",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

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
    }
  }
}
`)

	xHTTP(t, reg, "GET", "/model/foo", "", 404, "\"model/foo\" not found\n")
}

func TestGroupModelCreate(t *testing.T) {
	reg := NewRegistry("TestGroupModelCreate")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)

	xCheckGet(t, reg, "/model", `{
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
      }
    }
  }
}
`)

	xCheckGet(t, reg, "/model", `{
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
      }
    }
  }
}
`)

	xCheckGet(t, reg, "/model", `{
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
      }
    }
  }
}
`)

	// Now error checking
	gm, err = reg.Model.AddGroupModel("dirs1", "") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("", "") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("", "") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("", "dir1") // missing value
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("dirs", "dir") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("dirs1", "dir") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")

	gm, err = reg.Model.AddGroupModel("dirs", "dir1") // dup
	xCheck(t, gm == nil && err != nil, "gm should have failed")
}

func TestResourceModelCreate(t *testing.T) {
	reg := NewRegistry("TestResourceModels")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	xCheck(t, gm != nil, "gm should have worked")

	rm, err := gm.AddResourceModel("files", "file", 5, true, true, true)
	xNoErr(t, err)
	xCheck(t, rm != nil, "rm should have worked")

	rm2, err := gm.AddResourceModel("files", "file", 0, true, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	rm2, err = gm.AddResourceModel("files2", "file", 0, true, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	rm2, err = gm.AddResourceModel("", "file2", 0, true, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	rm2, err = gm.AddResourceModel("files2", "", 0, true, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	rm2, err = gm.AddResourceModel("files2", "file2", -1, true, true, true)
	xCheck(t, rm2 == nil && err != nil, "rm2 should have failed")

	gm2, err := reg.Model.AddGroupModel("dirs2", "dir2")
	xNoErr(t, err)
	xCheck(t, gm != nil, "gm2 should have worked")

	rm2, err = gm2.AddResourceModel("files", "file", 0, true, true, true)
	xCheck(t, rm != nil && err == nil, "gm2/rm2 should have worked")

	xCheckGet(t, reg, "/model", `{
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
          "maxversions": 5,
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
    },
    "dirs2": {
      "plural": "dirs2",
      "singular": "dir2",
      "attributes": {
        "dir2id": {
          "name": "dir2id",
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
}
`)

	rm2.Delete()
	xCheckGet(t, reg, "/model", `{
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
          "maxversions": 5,
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
    },
    "dirs2": {
      "plural": "dirs2",
      "singular": "dir2",
      "attributes": {
        "dir2id": {
          "name": "dir2id",
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
      }
    }
  }
}
`)

	xCheckGet(t, reg, "/model", `{
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
          "maxversions": 5,
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
    },
    "dirs2": {
      "plural": "dirs2",
      "singular": "dir2",
      "attributes": {
        "dir2id": {
          "name": "dir2id",
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
      }
    }
  }
}
`)

	gm2.Delete()
	xCheckGet(t, reg, "/model", `{
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
          "maxversions": 5,
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
}
`)

	xCheckGet(t, reg, "/model", `{
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
          "maxversions": 5,
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
}
`)

	newModel := &registry.Model{
		Attributes: registry.Attributes{
			"mystr": &registry.Attribute{
				Name: "mystr",
				Type: registry.STRING,
			},
		},
		Groups: map[string]*registry.GroupModel{
			"dirs": &registry.GroupModel{
				Plural:   "dirs",
				Singular: "dir",
				Resources: map[string]*registry.ResourceModel{
					"files": &registry.ResourceModel{
						Plural:           "files",
						Singular:         "file",
						MaxVersions:      6,
						SetVersionId:     registry.PtrBool(false),
						SetDefaultSticky: registry.PtrBool(false),
						HasDocument:      registry.PtrBool(false),
					},
				},
			},
		},
	}

	xNoErr(t, reg.Model.ApplyNewModel(newModel))
	xCheckGet(t, reg, "/model", `{
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
    },
    "mystr": {
      "name": "mystr",
      "type": "string"
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
          "maxversions": 6,
          "setversionid": false,
          "setdefaultversionsticky": false,
          "hasdocument": false,
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
}
`)

	// Make sure we allow, but ignore updates to "model"
	newModel = &registry.Model{
		Attributes: registry.Attributes{
			"model": &registry.Attribute{
				Name: "model",
				Type: registry.STRING,
			},
		},
	}
	err = reg.Model.ApplyNewModel(newModel)
	xNoErr(t, err)

	// Rollback since the previous "newModel" erased too much
	xNoErr(t, reg.Rollback())
	reg.Refresh()
	reg.LoadModel()

	g, err := reg.AddGroup("dirs", "dir1")
	xNoErr(t, err)
	g.AddResource("files", "f1", "v1")

	xCheckGet(t, reg, "?inline=model,dirs.files", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestResourceModels",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

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
      },
      "mystr": {
        "name": "mystr",
        "type": "string"
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
            "maxversions": 6,
            "setversionid": false,
            "setdefaultversionsticky": false,
            "hasdocument": false,
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
  "dirs": {
    "dir1": {
      "dirid": "dir1",
      "self": "http://localhost:8181/dirs/dir1",
      "xid": "/dirs/dir1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/dir1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/dir1/files/f1",
          "xid": "/dirs/dir1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",

          "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
          "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
          "versionscount": 1
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)

	newModel = &registry.Model{
		Groups: map[string]*registry.GroupModel{
			"dirs": &registry.GroupModel{
				Plural:   "dirs",
				Singular: "dir",
				Resources: map[string]*registry.ResourceModel{
					"files2": &registry.ResourceModel{
						Plural:           "files2",
						Singular:         "file",
						MaxVersions:      6,
						SetVersionId:     registry.PtrBool(false),
						SetDefaultSticky: registry.PtrBool(false),
						HasDocument:      registry.PtrBool(false),
					},
				},
			},
		},
	}

	reg.Model.ApplyNewModel(newModel)
	xCheckGet(t, reg, "?inline=model&inline=dirs", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestResourceModels",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

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
          "files2": {
            "plural": "files2",
            "singular": "file",
            "maxversions": 6,
            "setversionid": false,
            "setdefaultversionsticky": false,
            "hasdocument": false,
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
  "dirs": {
    "dir1": {
      "dirid": "dir1",
      "self": "http://localhost:8181/dirs/dir1",
      "xid": "/dirs/dir1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "files2url": "http://localhost:8181/dirs/dir1/files2",
      "files2count": 0
    }
  },
  "dirscount": 1
}
`)

	newModel = &registry.Model{
		Groups: map[string]*registry.GroupModel{
			"dirs": &registry.GroupModel{
				Plural:   "dirs",
				Singular: "dir",
			},
		},
	}

	reg.Model.ApplyNewModel(newModel)
	xCheckGet(t, reg, "?inline=model&inline=dirs", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestResourceModels",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

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
        }
      }
    }
  },

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "dir1": {
      "dirid": "dir1",
      "self": "http://localhost:8181/dirs/dir1",
      "xid": "/dirs/dir1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z"
    }
  },
  "dirscount": 1
}
`)

	newModel = &registry.Model{
		Groups: map[string]*registry.GroupModel{
			"dirs2": &registry.GroupModel{
				Plural:   "dirs2",
				Singular: "dir2",
			},
		},
	}
	reg.Model.ApplyNewModel(newModel)
	xCheckGet(t, reg, "?inline=model&inline=", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestResourceModels",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

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
      "dirs2": {
        "plural": "dirs2",
        "singular": "dir2",
        "attributes": {
          "dir2id": {
            "name": "dir2id",
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
        }
      }
    }
  },

  "dirs2url": "http://localhost:8181/dirs2",
  "dirs2": {},
  "dirs2count": 0
}
`)
}

func TestMultModelCreate(t *testing.T) {
	reg := NewRegistry("TestMultModelCreate")
	defer PassDeleteReg(t, reg)

	gm1, err := reg.Model.AddGroupModel("gms1", "gm1")
	xCheck(t, gm1 != nil && err == nil, "gm1 should have worked")

	rm1, err := gm1.AddResourceModel("rms1", "rm1", 0, true, true, true)
	xCheck(t, rm1 != nil && err == nil, "rm1 should have worked: %s", err)

	rm2, err := gm1.AddResourceModel("rms2", "rm2", 1, true, false, true)
	xCheck(t, rm2 != nil && err == nil, "rm2 should have worked: %s", err)

	gm2, err := reg.Model.AddGroupModel("gms2", "gm2")
	xCheck(t, gm1 != nil && err == nil, "gm1 should have worked: %s", err)

	rm21, err := gm2.AddResourceModel("rms1", "rm1", 2, true, true, true)
	xCheck(t, rm21 != nil && err == nil, "rm21 should have worked: %s", err)

	rm22, err := gm2.AddResourceModel("rms2", "rm2", 3, true, true, true)
	xCheck(t, rm22 != nil && err == nil, "rm12 should have worked: %s", err)

	xCheckGet(t, reg, "/model", `{
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
    "gms1": {
      "plural": "gms1",
      "singular": "gm1",
      "attributes": {
        "gm1id": {
          "name": "gm1id",
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
        "rms1": {
          "plural": "rms1",
          "singular": "rm1",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "attributes": {
            "rm1id": {
              "name": "rm1id",
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
            "rm1id": {
              "name": "rm1id",
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
        },
        "rms2": {
          "plural": "rms2",
          "singular": "rm2",
          "maxversions": 1,
          "setversionid": true,
          "setdefaultversionsticky": false,
          "hasdocument": true,
          "attributes": {
            "rm2id": {
              "name": "rm2id",
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
            "rm2id": {
              "name": "rm2id",
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
    },
    "gms2": {
      "plural": "gms2",
      "singular": "gm2",
      "attributes": {
        "gm2id": {
          "name": "gm2id",
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
        "rms1": {
          "plural": "rms1",
          "singular": "rm1",
          "maxversions": 2,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "attributes": {
            "rm1id": {
              "name": "rm1id",
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
            "rm1id": {
              "name": "rm1id",
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
        },
        "rms2": {
          "plural": "rms2",
          "singular": "rm2",
          "maxversions": 3,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "attributes": {
            "rm2id": {
              "name": "rm2id",
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
            "rm2id": {
              "name": "rm2id",
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
}
`)
}

func TestModelAPI(t *testing.T) {
	reg := NewRegistry("TestModelAPI")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs1", "dir1")
	gm.AddResourceModel("files", "file", 2, true, false, true)

	gm2, _ := reg.Model.AddGroupModel("dirs2", "dir2")
	gm2.AddResourceModel("files", "file", 0, false, true, true)

	m := reg.LoadModel()
	xJSONCheck(t, m, reg.Model)
}

func TestMultModel2Create(t *testing.T) {
	reg := NewRegistry("TestMultModel2Create")
	defer PassDeleteReg(t, reg)

	reg.SaveAllAndCommit()
	reg.Refresh()

	gm, _ := reg.Model.AddGroupModel("dirs1", "dir1")
	gm.AddResourceModel("files", "file", 2, true, false, true)

	d, _ := reg.AddGroup("dirs1", "d1")
	f, _ := d.AddResource("files", "f1", "v1")
	f.AddVersion("v2")
	d, _ = reg.AddGroup("dirs1", "d2")
	f, _ = d.AddResource("files", "f2", "v1")
	f.AddVersion("v1.1")

	gm2, _ := reg.Model.AddGroupModel("dirs2", "dir2")
	gm2.AddResourceModel("files", "file", 0, false, true, true)
	d2, _ := reg.AddGroup("dirs2", "d2")
	d2.AddResource("files", "f2", "v1")

	// /dirs1/d1/f1/v1
	//            /v2
	//       /d2/f2/v1
	//             v1.1
	// /dirs2/f2/f2/v1

	xCheckGet(t, reg, "?inline=model&inline", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestMultModel2Create",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

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
      "dirs1": {
        "plural": "dirs1",
        "singular": "dir1",
        "attributes": {
          "dir1id": {
            "name": "dir1id",
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
            "maxversions": 2,
            "setversionid": true,
            "setdefaultversionsticky": false,
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
      },
      "dirs2": {
        "plural": "dirs2",
        "singular": "dir2",
        "attributes": {
          "dir2id": {
            "name": "dir2id",
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
            "setversionid": false,
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

  "dirs1url": "http://localhost:8181/dirs1",
  "dirs1": {
    "d1": {
      "dir1id": "d1",
      "self": "http://localhost:8181/dirs1/d1",
      "xid": "/dirs1/d1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs1/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v2",
          "self": "http://localhost:8181/dirs1/d1/files/f1$details",
          "xid": "/dirs1/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",

          "metaurl": "http://localhost:8181/dirs1/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs1/d1/files/f1/meta",
            "xid": "/dirs1/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v2",
            "defaultversionurl": "http://localhost:8181/dirs1/d1/files/f1/versions/v2",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs1/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs1/d1/files/f1/versions/v1$details",
              "xid": "/dirs1/d1/files/f1/versions/v1",
              "epoch": 1,
              "isdefault": false,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "http://localhost:8181/dirs1/d1/files/f1/versions/v2$details",
              "xid": "/dirs1/d1/files/f1/versions/v2",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            }
          },
          "versionscount": 2
        }
      },
      "filescount": 1
    },
    "d2": {
      "dir1id": "d2",
      "self": "http://localhost:8181/dirs1/d2",
      "xid": "/dirs1/d2",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs1/d2/files",
      "files": {
        "f2": {
          "fileid": "f2",
          "versionid": "v1.1",
          "self": "http://localhost:8181/dirs1/d2/files/f2$details",
          "xid": "/dirs1/d2/files/f2",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",

          "metaurl": "http://localhost:8181/dirs1/d2/files/f2/meta",
          "meta": {
            "fileid": "f2",
            "self": "http://localhost:8181/dirs1/d2/files/f2/meta",
            "xid": "/dirs1/d2/files/f2/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v1.1",
            "defaultversionurl": "http://localhost:8181/dirs1/d2/files/f2/versions/v1.1",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs1/d2/files/f2/versions",
          "versions": {
            "v1": {
              "fileid": "f2",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs1/d2/files/f2/versions/v1$details",
              "xid": "/dirs1/d2/files/f2/versions/v1",
              "epoch": 1,
              "isdefault": false,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            },
            "v1.1": {
              "fileid": "f2",
              "versionid": "v1.1",
              "self": "http://localhost:8181/dirs1/d2/files/f2/versions/v1.1$details",
              "xid": "/dirs1/d2/files/f2/versions/v1.1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            }
          },
          "versionscount": 2
        }
      },
      "filescount": 1
    }
  },
  "dirs1count": 2,
  "dirs2url": "http://localhost:8181/dirs2",
  "dirs2": {
    "d2": {
      "dir2id": "d2",
      "self": "http://localhost:8181/dirs2/d2",
      "xid": "/dirs2/d2",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs2/d2/files",
      "files": {
        "f2": {
          "fileid": "f2",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs2/d2/files/f2$details",
          "xid": "/dirs2/d2/files/f2",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",

          "metaurl": "http://localhost:8181/dirs2/d2/files/f2/meta",
          "meta": {
            "fileid": "f2",
            "self": "http://localhost:8181/dirs2/d2/files/f2/meta",
            "xid": "/dirs2/d2/files/f2/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v1",
            "defaultversionurl": "http://localhost:8181/dirs2/d2/files/f2/versions/v1",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs2/d2/files/f2/versions",
          "versions": {
            "v1": {
              "fileid": "f2",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs2/d2/files/f2/versions/v1$details",
              "xid": "/dirs2/d2/files/f2/versions/v1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1
    }
  },
  "dirs2count": 1
}
`)

	gm, _ = reg.Model.AddGroupModel("dirs0", "dir0")
	gm.AddResourceModel("files", "file", 2, true, false, true)
	gm, _ = reg.Model.AddGroupModel("dirs3", "dir3")
	gm.AddResourceModel("files", "file", 2, true, false, true)

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs1":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{}}}}}},"dirs3":{}}`)

	gm, _ = reg.Model.AddGroupModel("dirs15", "dir15")
	gm.AddResourceModel("files", "file", 2, true, false, true)

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs1":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}},"dirs15":{},"dirs2":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{}}}}}},"dirs3":{}}`)

	gm, _ = reg.Model.AddGroupModel("dirs01", "dir01")
	gm, _ = reg.Model.AddGroupModel("dirs02", "dir02")
	gm, _ = reg.Model.AddGroupModel("dirs14", "dir014")
	gm, _ = reg.Model.AddGroupModel("dirs16", "dir016")
	gm, _ = reg.Model.AddGroupModel("dirs4", "dir4")
	gm, _ = reg.Model.AddGroupModel("dirs5", "dir5")

	xCheckGet(t, reg, "?inline&oneline",
		`{"dirs0":{},"dirs01":{},"dirs02":{},"dirs1":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}},"dirs14":{},"dirs15":{},"dirs16":{},"dirs2":{"d2":{"files":{"f2":{"meta":{},"versions":{"v1":{}}}}}},"dirs3":{},"dirs4":{},"dirs5":{}}`)
}

func TestModelLabels(t *testing.T) {
	reg := NewRegistry("TestModelLabels")
	defer PassDeleteReg(t, reg)

	xNoErr(t, reg.Model.AddLabel("reg-label", "reg-value"))

	gm, err := reg.Model.AddGroupModel("gms1", "gm1")
	xCheck(t, gm != nil && err == nil, "gm should have worked")
	xNoErr(t, gm.AddLabel("g-label", "g-value"))

	rm, err := gm.AddResourceModel("rms", "rm", 0, true, true, true)
	xCheck(t, rm != nil && err == nil, "rm should have worked: %s", err)
	xNoErr(t, rm.AddLabel("r-label", "r-value"))

	reg.SaveAllAndCommit()
	reg.Refresh()

	xCheckGet(t, reg, "/model", `{
  "labels": {
    "reg-label": "reg-value"
  },
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
    "gms1": {
      "plural": "gms1",
      "singular": "gm1",
      "labels": {
        "g-label": "g-value"
      },
      "attributes": {
        "gm1id": {
          "name": "gm1id",
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
        "rms": {
          "plural": "rms",
          "singular": "rm",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "labels": {
            "r-label": "r-value"
          },
          "attributes": {
            "rmid": {
              "name": "rmid",
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
            "rmid": {
              "name": "rmid",
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
}
`)

	xNoErr(t, reg.Refresh())
	reg.LoadModel()

	gm = reg.Model.FindGroupModel(gm.Plural)
	rm = gm.Resources[rm.Plural]

	xNoErr(t, reg.Model.RemoveLabel("reg-label"))
	xNoErr(t, gm.RemoveLabel("g-label"))
	xNoErr(t, rm.RemoveLabel("r-label"))

	xCheckGet(t, reg, "/model", `{
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
    "gms1": {
      "plural": "gms1",
      "singular": "gm1",
      "attributes": {
        "gm1id": {
          "name": "gm1id",
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
        "rms": {
          "plural": "rms",
          "singular": "rm",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "attributes": {
            "rmid": {
              "name": "rmid",
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
            "rmid": {
              "name": "rmid",
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
}
`)

	xHTTP(t, reg, "PUT", "/model", `{
  "labels": {
    "reg-label": "reg-value"
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "labels": {
        "g-label": "g-value"
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "labels": {
            "r-label": "r-value"
          }
        }
      }
    }
  }
}`, 200, `{
  "labels": {
    "reg-label": "reg-value"
  },
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
      "labels": {
        "g-label": "g-value"
      },
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
          "labels": {
            "r-label": "r-value"
          },
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
}
`)

}

// Make sure that we can use spec defined attribute names in a nested
// Object w/o the code mucking with it. There are some spots in the code
// where we'll do thinkg like skip certain attributes but we should only
// do that at the top level of the entire, not within a nested object.
func TestUseSpecAttrs(t *testing.T) {
	reg := NewRegistry("TestUseSpecAttrs")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, false)

	// Registry level
	obj, err := reg.Model.AddAttrObj("obj")
	xNoErr(t, err)
	vals := map[string]any{}

	count := 0
	for _, prop := range registry.OrderedSpecProps {
		if prop.Name[0] == '$' {
			continue
		}

		v := any(count)
		typ := registry.INTEGER
		if prop.Type == registry.INTEGER || prop.Type == registry.UINTEGER {
			typ = registry.STRING
			v = fmt.Sprintf("%d-%s", count, prop.Name)
		}
		_, err := obj.AddAttr(prop.Name, typ)
		xNoErr(t, err)
		vals["obj."+prop.Name] = v

		if prop.Name == "id" {
			_, err := obj.AddAttr("registryid", typ)
			xNoErr(t, err)
			vals["obj.registryid"] = v
		}
		count++
	}

	for k, v := range vals {
		xNoErr(t, reg.SetSave(k, v))
	}

	// Group level
	obj, err = gm.AddAttrObj("obj")
	xNoErr(t, err)
	vals = map[string]any{}

	count = 0
	for _, prop := range registry.OrderedSpecProps {
		if prop.Name[0] == '$' {
			continue
		}

		v := any(count)
		typ := registry.INTEGER
		if prop.Type == registry.INTEGER || prop.Type == registry.UINTEGER {
			typ = registry.STRING
			v = fmt.Sprintf("%d-%s", count, prop.Name)
		}
		_, err := obj.AddAttr(prop.Name, typ)
		xNoErr(t, err)
		vals["obj."+prop.Name] = v

		if prop.Name == "id" {
			_, err = obj.AddAttr("registryid", typ)
			_, err = obj.AddAttr("dirid", typ)
			_, err = obj.AddAttr("fileid", typ)
			xNoErr(t, err)
			vals["obj.registryid"] = v
			vals["obj.dirid"] = v
			vals["obj.fileid"] = v
		}
		count++
	}

	d1, err := reg.AddGroup("dirs", "d1")
	xNoErr(t, err)
	for k, v := range vals {
		xNoErr(t, d1.SetSave(k, v))
	}

	// Resource level
	obj, err = rm.AddAttrObj("obj")
	xNoErr(t, err)

	objMeta, err := rm.AddMetaAttrObj("obj")
	xNoErr(t, err)

	vals = map[string]any{}

	count = 0
	for _, prop := range registry.OrderedSpecProps {
		if prop.Name[0] == '$' {
			continue
		}

		v := any(count)
		typ := registry.INTEGER
		if prop.Type == registry.INTEGER || prop.Type == registry.UINTEGER {
			typ = registry.STRING
			v = fmt.Sprintf("%d-%s", count, prop.Name)
		}
		_, err := obj.AddAttr(prop.Name, typ)
		xNoErr(t, err)

		_, err = objMeta.AddAttr(prop.Name, typ)
		xNoErr(t, err)

		vals["obj."+prop.Name] = v

		if prop.Name == "id" {
			_, err = obj.AddAttr("registryid", typ)
			_, err = obj.AddAttr("dirid", typ)
			_, err = obj.AddAttr("fileid", typ)
			_, err = obj.AddAttr("file", typ)
			_, err = obj.AddAttr("filebase64", typ)
			_, err = obj.AddAttr("fileurl", typ)
			xNoErr(t, err)
			_, err = objMeta.AddAttr("registryid", typ)
			_, err = objMeta.AddAttr("dirid", typ)
			_, err = objMeta.AddAttr("fileid", typ)
			_, err = objMeta.AddAttr("file", typ)
			_, err = objMeta.AddAttr("filebase64", typ)
			_, err = objMeta.AddAttr("fileurl", typ)
			xNoErr(t, err)
			vals["obj.registryid"] = v
			vals["obj.dirid"] = v
			vals["obj.fileid"] = v
			vals["obj.file"] = v
			vals["obj.filebase64"] = v
			vals["obj.fileurl"] = v
		}
		count++
	}

	r1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)

	v1, err := r1.FindVersion("v1", false)
	xNoErr(t, err)

	meta, err := r1.FindMeta(false)
	xNoErr(t, err)

	for k, v := range vals {
		xNoErr(t, v1.SetSave(k, v))
		xNoErr(t, meta.SetSave(k, v))
	}

	xHTTP(t, reg, "GET", "?inline=*,model", "", 200, `{
  "specversion": "1.0-rc1",
  "registryid": "TestUseSpecAttrs",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "obj": {
    "capabilities": 21,
    "compatibility": 15,
    "contenttype": 16,
    "createdat": 12,
    "defaultversionid": 18,
    "defaultversionsticky": 20,
    "defaultversionurl": 19,
    "description": 9,
    "documentation": 10,
    "epoch": "6-epoch",
    "id": 1,
    "isdefault": 8,
    "labels": 11,
    "metaurl": 17,
    "model": 22,
    "modifiedat": 13,
    "name": 7,
    "readonly": 14,
    "registryid": 1,
    "self": 3,
    "specversion": 0,
    "versionid": 2,
    "xid": 4,
    "xref": 5
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
      },
      "obj": {
        "name": "obj",
        "type": "object",
        "attributes": {
          "capabilities": {
            "name": "capabilities",
            "type": "integer"
          },
          "compatibility": {
            "name": "compatibility",
            "type": "integer"
          },
          "contenttype": {
            "name": "contenttype",
            "type": "integer"
          },
          "createdat": {
            "name": "createdat",
            "type": "integer"
          },
          "defaultversionid": {
            "name": "defaultversionid",
            "type": "integer"
          },
          "defaultversionsticky": {
            "name": "defaultversionsticky",
            "type": "integer"
          },
          "defaultversionurl": {
            "name": "defaultversionurl",
            "type": "integer"
          },
          "description": {
            "name": "description",
            "type": "integer"
          },
          "documentation": {
            "name": "documentation",
            "type": "integer"
          },
          "epoch": {
            "name": "epoch",
            "type": "string"
          },
          "id": {
            "name": "id",
            "type": "integer"
          },
          "isdefault": {
            "name": "isdefault",
            "type": "integer"
          },
          "labels": {
            "name": "labels",
            "type": "integer"
          },
          "metaurl": {
            "name": "metaurl",
            "type": "integer"
          },
          "model": {
            "name": "model",
            "type": "integer"
          },
          "modifiedat": {
            "name": "modifiedat",
            "type": "integer"
          },
          "name": {
            "name": "name",
            "type": "integer"
          },
          "readonly": {
            "name": "readonly",
            "type": "integer"
          },
          "registryid": {
            "name": "registryid",
            "type": "integer"
          },
          "self": {
            "name": "self",
            "type": "integer"
          },
          "specversion": {
            "name": "specversion",
            "type": "integer"
          },
          "versionid": {
            "name": "versionid",
            "type": "integer"
          },
          "xid": {
            "name": "xid",
            "type": "integer"
          },
          "xref": {
            "name": "xref",
            "type": "integer"
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
          },
          "obj": {
            "name": "obj",
            "type": "object",
            "attributes": {
              "capabilities": {
                "name": "capabilities",
                "type": "integer"
              },
              "compatibility": {
                "name": "compatibility",
                "type": "integer"
              },
              "contenttype": {
                "name": "contenttype",
                "type": "integer"
              },
              "createdat": {
                "name": "createdat",
                "type": "integer"
              },
              "defaultversionid": {
                "name": "defaultversionid",
                "type": "integer"
              },
              "defaultversionsticky": {
                "name": "defaultversionsticky",
                "type": "integer"
              },
              "defaultversionurl": {
                "name": "defaultversionurl",
                "type": "integer"
              },
              "description": {
                "name": "description",
                "type": "integer"
              },
              "dirid": {
                "name": "dirid",
                "type": "integer"
              },
              "documentation": {
                "name": "documentation",
                "type": "integer"
              },
              "epoch": {
                "name": "epoch",
                "type": "string"
              },
              "fileid": {
                "name": "fileid",
                "type": "integer"
              },
              "id": {
                "name": "id",
                "type": "integer"
              },
              "isdefault": {
                "name": "isdefault",
                "type": "integer"
              },
              "labels": {
                "name": "labels",
                "type": "integer"
              },
              "metaurl": {
                "name": "metaurl",
                "type": "integer"
              },
              "model": {
                "name": "model",
                "type": "integer"
              },
              "modifiedat": {
                "name": "modifiedat",
                "type": "integer"
              },
              "name": {
                "name": "name",
                "type": "integer"
              },
              "readonly": {
                "name": "readonly",
                "type": "integer"
              },
              "registryid": {
                "name": "registryid",
                "type": "integer"
              },
              "self": {
                "name": "self",
                "type": "integer"
              },
              "specversion": {
                "name": "specversion",
                "type": "integer"
              },
              "versionid": {
                "name": "versionid",
                "type": "integer"
              },
              "xid": {
                "name": "xid",
                "type": "integer"
              },
              "xref": {
                "name": "xref",
                "type": "integer"
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
              },
              "obj": {
                "name": "obj",
                "type": "object",
                "attributes": {
                  "capabilities": {
                    "name": "capabilities",
                    "type": "integer"
                  },
                  "compatibility": {
                    "name": "compatibility",
                    "type": "integer"
                  },
                  "contenttype": {
                    "name": "contenttype",
                    "type": "integer"
                  },
                  "createdat": {
                    "name": "createdat",
                    "type": "integer"
                  },
                  "defaultversionid": {
                    "name": "defaultversionid",
                    "type": "integer"
                  },
                  "defaultversionsticky": {
                    "name": "defaultversionsticky",
                    "type": "integer"
                  },
                  "defaultversionurl": {
                    "name": "defaultversionurl",
                    "type": "integer"
                  },
                  "description": {
                    "name": "description",
                    "type": "integer"
                  },
                  "dirid": {
                    "name": "dirid",
                    "type": "integer"
                  },
                  "documentation": {
                    "name": "documentation",
                    "type": "integer"
                  },
                  "epoch": {
                    "name": "epoch",
                    "type": "string"
                  },
                  "file": {
                    "name": "file",
                    "type": "integer"
                  },
                  "filebase64": {
                    "name": "filebase64",
                    "type": "integer"
                  },
                  "fileid": {
                    "name": "fileid",
                    "type": "integer"
                  },
                  "fileurl": {
                    "name": "fileurl",
                    "type": "integer"
                  },
                  "id": {
                    "name": "id",
                    "type": "integer"
                  },
                  "isdefault": {
                    "name": "isdefault",
                    "type": "integer"
                  },
                  "labels": {
                    "name": "labels",
                    "type": "integer"
                  },
                  "metaurl": {
                    "name": "metaurl",
                    "type": "integer"
                  },
                  "model": {
                    "name": "model",
                    "type": "integer"
                  },
                  "modifiedat": {
                    "name": "modifiedat",
                    "type": "integer"
                  },
                  "name": {
                    "name": "name",
                    "type": "integer"
                  },
                  "readonly": {
                    "name": "readonly",
                    "type": "integer"
                  },
                  "registryid": {
                    "name": "registryid",
                    "type": "integer"
                  },
                  "self": {
                    "name": "self",
                    "type": "integer"
                  },
                  "specversion": {
                    "name": "specversion",
                    "type": "integer"
                  },
                  "versionid": {
                    "name": "versionid",
                    "type": "integer"
                  },
                  "xid": {
                    "name": "xid",
                    "type": "integer"
                  },
                  "xref": {
                    "name": "xref",
                    "type": "integer"
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
              },
              "obj": {
                "name": "obj",
                "type": "object",
                "attributes": {
                  "capabilities": {
                    "name": "capabilities",
                    "type": "integer"
                  },
                  "compatibility": {
                    "name": "compatibility",
                    "type": "integer"
                  },
                  "contenttype": {
                    "name": "contenttype",
                    "type": "integer"
                  },
                  "createdat": {
                    "name": "createdat",
                    "type": "integer"
                  },
                  "defaultversionid": {
                    "name": "defaultversionid",
                    "type": "integer"
                  },
                  "defaultversionsticky": {
                    "name": "defaultversionsticky",
                    "type": "integer"
                  },
                  "defaultversionurl": {
                    "name": "defaultversionurl",
                    "type": "integer"
                  },
                  "description": {
                    "name": "description",
                    "type": "integer"
                  },
                  "dirid": {
                    "name": "dirid",
                    "type": "integer"
                  },
                  "documentation": {
                    "name": "documentation",
                    "type": "integer"
                  },
                  "epoch": {
                    "name": "epoch",
                    "type": "string"
                  },
                  "file": {
                    "name": "file",
                    "type": "integer"
                  },
                  "filebase64": {
                    "name": "filebase64",
                    "type": "integer"
                  },
                  "fileid": {
                    "name": "fileid",
                    "type": "integer"
                  },
                  "fileurl": {
                    "name": "fileurl",
                    "type": "integer"
                  },
                  "id": {
                    "name": "id",
                    "type": "integer"
                  },
                  "isdefault": {
                    "name": "isdefault",
                    "type": "integer"
                  },
                  "labels": {
                    "name": "labels",
                    "type": "integer"
                  },
                  "metaurl": {
                    "name": "metaurl",
                    "type": "integer"
                  },
                  "model": {
                    "name": "model",
                    "type": "integer"
                  },
                  "modifiedat": {
                    "name": "modifiedat",
                    "type": "integer"
                  },
                  "name": {
                    "name": "name",
                    "type": "integer"
                  },
                  "readonly": {
                    "name": "readonly",
                    "type": "integer"
                  },
                  "registryid": {
                    "name": "registryid",
                    "type": "integer"
                  },
                  "self": {
                    "name": "self",
                    "type": "integer"
                  },
                  "specversion": {
                    "name": "specversion",
                    "type": "integer"
                  },
                  "versionid": {
                    "name": "versionid",
                    "type": "integer"
                  },
                  "xid": {
                    "name": "xid",
                    "type": "integer"
                  },
                  "xref": {
                    "name": "xref",
                    "type": "integer"
                  }
                }
              }
            }
          }
        }
      }
    }
  },

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:02Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
      "obj": {
        "capabilities": 21,
        "compatibility": 15,
        "contenttype": 16,
        "createdat": 12,
        "defaultversionid": 18,
        "defaultversionsticky": 20,
        "defaultversionurl": 19,
        "description": 9,
        "dirid": 1,
        "documentation": 10,
        "epoch": "6-epoch",
        "fileid": 1,
        "id": 1,
        "isdefault": 8,
        "labels": 11,
        "metaurl": 17,
        "model": 22,
        "modifiedat": 13,
        "name": 7,
        "readonly": 14,
        "registryid": 1,
        "self": 3,
        "specversion": 0,
        "versionid": 2,
        "xid": 4,
        "xref": 5
      },

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/d1/files/f1",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "YYYY-MM-DDTHH:MM:02Z",
          "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
          "obj": {
            "capabilities": 21,
            "compatibility": 15,
            "contenttype": 16,
            "createdat": 12,
            "defaultversionid": 18,
            "defaultversionsticky": 20,
            "defaultversionurl": 19,
            "description": 9,
            "dirid": 1,
            "documentation": 10,
            "epoch": "6-epoch",
            "file": 1,
            "filebase64": 1,
            "fileid": 1,
            "fileurl": 1,
            "id": 1,
            "isdefault": 8,
            "labels": 11,
            "metaurl": 17,
            "model": 22,
            "modifiedat": 13,
            "name": 7,
            "readonly": 14,
            "registryid": 1,
            "self": 3,
            "specversion": 0,
            "versionid": 2,
            "xid": 4,
            "xref": 5
          },

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "YYYY-MM-DDTHH:MM:02Z",
            "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
            "readonly": false,
            "compatibility": "none",
            "obj": {
              "capabilities": 21,
              "compatibility": 15,
              "contenttype": 16,
              "createdat": 12,
              "defaultversionid": 18,
              "defaultversionsticky": 20,
              "defaultversionurl": 19,
              "description": 9,
              "dirid": 1,
              "documentation": 10,
              "epoch": "6-epoch",
              "file": 1,
              "filebase64": 1,
              "fileid": 1,
              "fileurl": 1,
              "id": 1,
              "isdefault": 8,
              "labels": 11,
              "metaurl": 17,
              "model": 22,
              "modifiedat": 13,
              "name": 7,
              "readonly": 14,
              "registryid": 1,
              "self": 3,
              "specversion": 0,
              "versionid": 2,
              "xid": 4,
              "xref": 5
            },

            "defaultversionid": "v1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "YYYY-MM-DDTHH:MM:02Z",
              "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
              "obj": {
                "capabilities": 21,
                "compatibility": 15,
                "contenttype": 16,
                "createdat": 12,
                "defaultversionid": 18,
                "defaultversionsticky": 20,
                "defaultversionurl": 19,
                "description": 9,
                "dirid": 1,
                "documentation": 10,
                "epoch": "6-epoch",
                "file": 1,
                "filebase64": 1,
                "fileid": 1,
                "fileurl": 1,
                "id": 1,
                "isdefault": 8,
                "labels": 11,
                "metaurl": 17,
                "model": 22,
                "modifiedat": 13,
                "name": 7,
                "readonly": 14,
                "registryid": 1,
                "self": 3,
                "specversion": 0,
                "versionid": 2,
                "xid": 4,
                "xref": 5
              }
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)
}

func TestModelCompatibleWith(t *testing.T) {
	reg := NewRegistry("TestModelCompatibleWith")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "PUT", "/model", `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "modelversion": "1.0",
      "compatiblewith": "some-url1",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "modelversion": "3.1",
          "compatiblewith": "some-url2"
        },
        "datas": {
          "plural": "datas",
          "singular": "data",
          "modelversion": "3.2"
        },
        "foos": {
          "plural": "foos",
          "singular": "foo",
          "compatiblewith": "some-url3"
        }
      }
    }
  }
}`, 200, `{
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
      "modelversion": "1.0",
      "compatiblewith": "some-url1",
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
        "datas": {
          "plural": "datas",
          "singular": "data",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "modelversion": "3.2",
          "attributes": {
            "dataid": {
              "name": "dataid",
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
            "dataid": {
              "name": "dataid",
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
        },
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "modelversion": "3.1",
          "compatiblewith": "some-url2",
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
        },
        "foos": {
          "plural": "foos",
          "singular": "foo",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "compatiblewith": "some-url3",
          "attributes": {
            "fooid": {
              "name": "fooid",
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
            "fooid": {
              "name": "fooid",
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
}
`)
}

func TestModelIncludes(t *testing.T) {
	reg := NewRegistry("TestModelCompatibleWith")
	defer PassDeleteReg(t, reg)

	// First try w/o any includes
	buf, err := os.ReadFile("files/sample-model.json")
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/model", string(buf), 200, `{
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
      "readonly": true,
      "required": true
    },
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp",
      "readonly": true,
      "required": true
    },
    "id": {
      "name": "id",
      "type": "string",
      "immutable": true,
      "required": true
    }
  },
  "groups": {
    "sample_groups": {
      "plural": "sample_groups",
      "singular": "sample_group",
      "attributes": {
        "sample_groupid": {
          "name": "sample_groupid",
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
          "readonly": true,
          "required": true
        },
        "modifiedat": {
          "name": "modifiedat",
          "type": "timestamp",
          "readonly": true,
          "required": true
        },
        "id": {
          "name": "id",
          "type": "string",
          "immutable": true,
          "required": true
        },
        "origin": {
          "name": "origin",
          "type": "uri"
        }
      },
      "resources": {
        "sample_resources": {
          "plural": "sample_resources",
          "singular": "sample_resource",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "attributes": {
            "sample_resourceid": {
              "name": "sample_resourceid",
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
              "readonly": true,
              "required": true
            },
            "modifiedat": {
              "name": "modifiedat",
              "type": "timestamp",
              "readonly": true,
              "required": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "readonly": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
              "readonly": true
            },
            "default": {
              "name": "default",
              "type": "boolean"
            },
            "id": {
              "name": "id",
              "type": "string",
              "immutable": true,
              "required": true
            },
            "origin": {
              "name": "origin",
              "type": "uri"
            }
          },
          "metaattributes": {
            "sample_resourceid": {
              "name": "sample_resourceid",
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
}
`)

	// Now a simple $include that works
	buf, err = os.ReadFile("files/dir/model-dirs-inc-docs.json")
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/model", string(buf), 200, `{
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
    },
    "docs": {
      "plural": "docs",
      "singular": "doc",
      "attributes": {
        "docid": {
          "name": "docid",
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
        "formats": {
          "plural": "formats",
          "singular": "format",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "attributes": {
            "formatid": {
              "name": "formatid",
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
            "formatid": {
              "name": "formatid",
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
}
`)

	// Now one with an invalid include - using ..
	str := `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file"
        }
      }
    } ,
    "$include": "../foo.json"
  }
}
`

	xHTTP(t, reg, "PUT", "/model", str, 400,
		"Not allowed to access file: ../foo.json\n")

	// Another bad one - using /foo
	str = `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file"
        }
      }
    } ,
    "$include": "/foo.json"
  }
}
`
	xHTTP(t, reg, "PUT", "/model", str, 400,
		"Not allowed to access file: /foo.json\n")

	// Another bad one - 404 url
	str = `
{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file"
        }
      }
    } ,
    "$include": "http://bogus.bogus.bogus.bogus.com/bogus.json"
  }
}
`
	xHTTP(t, reg, "PUT", "/model", str, 400,
		`Get "http://bogus.bogus.bogus.bogus.com/bogus.json": dial tcp: lookup bogus.bogus.bogus.bogus.com on 127.0.0.53:53: no such host
`)

	// nested include with http ref
	str = `
{
  "$include": "http://localhost:8282/dir/model-dirs-inc-docs.json"
}
`

	xHTTP(t, reg, "PUT", "/model", str, 200, `{
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
    },
    "docs": {
      "plural": "docs",
      "singular": "doc",
      "attributes": {
        "docid": {
          "name": "docid",
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
        "formats": {
          "plural": "formats",
          "singular": "format",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "attributes": {
            "formatid": {
              "name": "formatid",
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
            "formatid": {
              "name": "formatid",
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
}
`)

	// nested include with local ref
	str = `
{
  "$include": "http://localhost:8282/dir/model-dirs-inc-docs-indirect.json"
}
`

	xHTTP(t, reg, "PUT", "/model", str, 200, `{
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
    },
    "docs": {
      "plural": "docs",
      "singular": "doc",
      "attributes": {
        "docid": {
          "name": "docid",
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
        "formats": {
          "plural": "formats",
          "singular": "format",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "attributes": {
            "formatid": {
              "name": "formatid",
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
            "formatid": {
              "name": "formatid",
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
}
`)

	// nested include with local ref using ..
	str = `
{
  "$include": "http://localhost:8282/dir/model-dirs-inc-docs-indirect2.json"
}
`

	xHTTP(t, reg, "PUT", "/model", str, 200, `{
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
    },
    "docs": {
      "plural": "docs",
      "singular": "doc",
      "attributes": {
        "docid": {
          "name": "docid",
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
        "formats": {
          "plural": "formats",
          "singular": "format",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "attributes": {
            "formatid": {
              "name": "formatid",
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
            "formatid": {
              "name": "formatid",
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
}
`)

}
