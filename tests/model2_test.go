package tests

import (
	"fmt"
	"os"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

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
	reg.Refresh(registry.FOR_WRITE)

	xCheckGet(t, reg, "/model", `{
  "labels": {
    "reg-label": "reg-value"
  },
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
      "attributes": {
        "*": {
          "name": "*",
          "type": "any"
        }
      }
    },
    "gms1url": {
      "name": "gms1url",
      "type": "url",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "gms1count": {
      "name": "gms1count",
      "type": "uinteger",
      "readonly": true,
      "required": true
    },
    "gms1": {
      "name": "gms1",
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
        "rmsurl": {
          "name": "rmsurl",
          "type": "url",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "rmscount": {
          "name": "rmscount",
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "rms": {
          "name": "rms",
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
        "rms": {
          "plural": "rms",
          "singular": "rm",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "singleversionroot": false,
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
            "rmurl": {
              "name": "rmurl",
              "type": "url"
            },
            "rmproxyurl": {
              "name": "rmproxyurl",
              "type": "url"
            },
            "rm": {
              "name": "rm",
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

	xNoErr(t, reg.Refresh(registry.FOR_WRITE))
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
      "attributes": {
        "*": {
          "name": "*",
          "type": "any"
        }
      }
    },
    "gms1url": {
      "name": "gms1url",
      "type": "url",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "gms1count": {
      "name": "gms1count",
      "type": "uinteger",
      "readonly": true,
      "required": true
    },
    "gms1": {
      "name": "gms1",
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
        "rmsurl": {
          "name": "rmsurl",
          "type": "url",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "rmscount": {
          "name": "rmscount",
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "rms": {
          "name": "rms",
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
        "rms": {
          "plural": "rms",
          "singular": "rm",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "singleversionroot": false,
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
            "rmurl": {
              "name": "rmurl",
              "type": "url"
            },
            "rmproxyurl": {
              "name": "rmproxyurl",
              "type": "url"
            },
            "rm": {
              "name": "rm",
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

	xHTTP(t, reg, "PUT", "/model", `{
  "labels": {
    "reg-label": "reg-value"
  },
  "groups": {
    "dirs": {
      "singular": "dir",
      "labels": {
        "g-label": "g-value"
      },
      "resources": {
        "files": {
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

	for _, prop := range registry.OrderedSpecProps {
		if prop.Name[0] == '$' {
			continue
		}

		v := any(len(prop.Name))
		typ := INTEGER
		if prop.Type == INTEGER || prop.Type == UINTEGER {
			typ = STRING
			v = fmt.Sprintf("%d-%s", v, prop.Name)
		}
		_, err := obj.AddAttr(prop.Name, typ)
		xNoErr(t, err)
		vals["obj."+prop.Name] = v

		if prop.Name == "id" {
			_, err := obj.AddAttr("registryid", typ)
			xNoErr(t, err)
			vals["obj.registryid"] = 10
		}
	}

	for k, v := range vals {
		xNoErr(t, reg.SetSave(k, v))
	}

	// Group level
	obj, err = gm.AddAttrObj("obj")
	xNoErr(t, err)
	vals = map[string]any{}

	d1, err := reg.AddGroup("dirs", "d1")
	xNoErr(t, err)
	for _, prop := range registry.OrderedSpecProps {
		if prop.Name[0] == '$' {
			continue
		}

		v := any(len(prop.Name))
		typ := INTEGER
		if prop.Type == INTEGER || prop.Type == UINTEGER {
			typ = STRING
			v = fmt.Sprintf("%d-%s", v, prop.Name)
		}
		_, err := obj.AddAttr(prop.Name, typ)
		xNoErr(t, err)
		xNoErr(t, d1.SetSave("obj."+prop.Name, v))

		if prop.Name == "id" {
			_, err = obj.AddAttr("registryid", typ)
			_, err = obj.AddAttr("dirid", typ)
			_, err = obj.AddAttr("fileid", typ)
			xNoErr(t, err)
			xNoErr(t, d1.SetSave("obj.registryid", 10))
			xNoErr(t, d1.SetSave("obj.dirid", 5))
			xNoErr(t, d1.SetSave("obj.fileid", 6))
		}
	}

	// Resource level
	obj, err = rm.AddAttrObj("obj")
	xNoErr(t, err)

	objMeta, err := rm.AddMetaAttrObj("obj")
	xNoErr(t, err)

	vals = map[string]any{}

	for _, prop := range registry.OrderedSpecProps {
		if prop.Name[0] == '$' {
			continue
		}

		v := any(len(prop.Name))
		typ := INTEGER
		if prop.Type == INTEGER || prop.Type == UINTEGER {
			typ = STRING
			v = fmt.Sprintf("%d-%s", v, prop.Name)
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
			vals["obj.registryid"] = 10
			vals["obj.dirid"] = 5
			vals["obj.fileid"] = 6
			vals["obj.file"] = 4
			vals["obj.filebase64"] = 10
			vals["obj.fileurl"] = 7
		}
	}

	r1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)

	v1, err := r1.FindVersion("v1", false, registry.FOR_WRITE)
	xNoErr(t, err)

	meta, err := r1.FindMeta(false, registry.FOR_WRITE)
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
    "ancestor": 8,
    "capabilities": 12,
    "compatibility": 13,
    "compatibilityauthority": 22,
    "contenttype": 11,
    "createdat": 9,
    "defaultversionid": 16,
    "defaultversionsticky": 20,
    "defaultversionurl": 17,
    "deprecated": 10,
    "description": 11,
    "documentation": 13,
    "epoch": "5-epoch",
    "id": 2,
    "isdefault": 9,
    "labels": 6,
    "meta": 4,
    "metaurl": 7,
    "model": 5,
    "modifiedat": 10,
    "name": 4,
    "readonly": 8,
    "registryid": 10,
    "self": 4,
    "specversion": 11,
    "versionid": 9,
    "xid": 3,
    "xref": 4
  },

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
      "obj": {
        "name": "obj",
        "type": "object",
        "attributes": {
          "ancestor": {
            "name": "ancestor",
            "type": "integer"
          },
          "capabilities": {
            "name": "capabilities",
            "type": "integer"
          },
          "compatibility": {
            "name": "compatibility",
            "type": "integer"
          },
          "compatibilityauthority": {
            "name": "compatibilityauthority",
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
          "deprecated": {
            "name": "deprecated",
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
          "meta": {
            "name": "meta",
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
          "obj": {
            "name": "obj",
            "type": "object",
            "attributes": {
              "ancestor": {
                "name": "ancestor",
                "type": "integer"
              },
              "capabilities": {
                "name": "capabilities",
                "type": "integer"
              },
              "compatibility": {
                "name": "compatibility",
                "type": "integer"
              },
              "compatibilityauthority": {
                "name": "compatibilityauthority",
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
              "deprecated": {
                "name": "deprecated",
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
              "meta": {
                "name": "meta",
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
              },
              "obj": {
                "name": "obj",
                "type": "object",
                "attributes": {
                  "ancestor": {
                    "name": "ancestor",
                    "type": "integer"
                  },
                  "capabilities": {
                    "name": "capabilities",
                    "type": "integer"
                  },
                  "compatibility": {
                    "name": "compatibility",
                    "type": "integer"
                  },
                  "compatibilityauthority": {
                    "name": "compatibilityauthority",
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
                  "deprecated": {
                    "name": "deprecated",
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
                  "meta": {
                    "name": "meta",
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
              "obj": {
                "name": "obj",
                "type": "object",
                "attributes": {
                  "ancestor": {
                    "name": "ancestor",
                    "type": "integer"
                  },
                  "capabilities": {
                    "name": "capabilities",
                    "type": "integer"
                  },
                  "compatibility": {
                    "name": "compatibility",
                    "type": "integer"
                  },
                  "compatibilityauthority": {
                    "name": "compatibilityauthority",
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
                  "deprecated": {
                    "name": "deprecated",
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
                  "meta": {
                    "name": "meta",
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
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:02Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
      "obj": {
        "ancestor": 8,
        "capabilities": 12,
        "compatibility": 13,
        "compatibilityauthority": 22,
        "contenttype": 11,
        "createdat": 9,
        "defaultversionid": 16,
        "defaultversionsticky": 20,
        "defaultversionurl": 17,
        "deprecated": 10,
        "description": 11,
        "dirid": 5,
        "documentation": 13,
        "epoch": "5-epoch",
        "fileid": 6,
        "id": 2,
        "isdefault": 9,
        "labels": 6,
        "meta": 4,
        "metaurl": 7,
        "model": 5,
        "modifiedat": 10,
        "name": 4,
        "readonly": 8,
        "registryid": 10,
        "self": 4,
        "specversion": 11,
        "versionid": 9,
        "xid": 3,
        "xref": 4
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
          "ancestor": "v1",
          "obj": {
            "ancestor": 8,
            "capabilities": 12,
            "compatibility": 13,
            "compatibilityauthority": 22,
            "contenttype": 11,
            "createdat": 9,
            "defaultversionid": 16,
            "defaultversionsticky": 20,
            "defaultversionurl": 17,
            "deprecated": 10,
            "description": 11,
            "dirid": 5,
            "documentation": 13,
            "epoch": "5-epoch",
            "file": 4,
            "filebase64": 10,
            "fileid": 6,
            "fileurl": 7,
            "id": 2,
            "isdefault": 9,
            "labels": 6,
            "meta": 4,
            "metaurl": 7,
            "model": 5,
            "modifiedat": 10,
            "name": 4,
            "readonly": 8,
            "registryid": 10,
            "self": 4,
            "specversion": 11,
            "versionid": 9,
            "xid": 3,
            "xref": 4
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
              "ancestor": 8,
              "capabilities": 12,
              "compatibility": 13,
              "compatibilityauthority": 22,
              "contenttype": 11,
              "createdat": 9,
              "defaultversionid": 16,
              "defaultversionsticky": 20,
              "defaultversionurl": 17,
              "deprecated": 10,
              "description": 11,
              "dirid": 5,
              "documentation": 13,
              "epoch": "5-epoch",
              "file": 4,
              "filebase64": 10,
              "fileid": 6,
              "fileurl": 7,
              "id": 2,
              "isdefault": 9,
              "labels": 6,
              "meta": 4,
              "metaurl": 7,
              "model": 5,
              "modifiedat": 10,
              "name": 4,
              "readonly": 8,
              "registryid": 10,
              "self": 4,
              "specversion": 11,
              "versionid": 9,
              "xid": 3,
              "xref": 4
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
              "ancestor": "v1",
              "obj": {
                "ancestor": 8,
                "capabilities": 12,
                "compatibility": 13,
                "compatibilityauthority": 22,
                "contenttype": 11,
                "createdat": 9,
                "defaultversionid": 16,
                "defaultversionsticky": 20,
                "defaultversionurl": 17,
                "deprecated": 10,
                "description": 11,
                "dirid": 5,
                "documentation": 13,
                "epoch": "5-epoch",
                "file": 4,
                "filebase64": 10,
                "fileid": 6,
                "fileurl": 7,
                "id": 2,
                "isdefault": 9,
                "labels": 6,
                "meta": 4,
                "metaurl": 7,
                "model": 5,
                "modifiedat": 10,
                "name": 4,
                "readonly": 8,
                "registryid": 10,
                "self": 4,
                "specversion": 11,
                "versionid": 9,
                "xid": 3,
                "xref": 4
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
        "datasurl": {
          "name": "datasurl",
          "type": "url",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "datascount": {
          "name": "datascount",
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "datas": {
          "name": "datas",
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
        },
        "foosurl": {
          "name": "foosurl",
          "type": "url",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "fooscount": {
          "name": "fooscount",
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "foos": {
          "name": "foos",
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
        "datas": {
          "plural": "datas",
          "singular": "data",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "singleversionroot": false,
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
            "dataurl": {
              "name": "dataurl",
              "type": "url"
            },
            "dataproxyurl": {
              "name": "dataproxyurl",
              "type": "url"
            },
            "data": {
              "name": "data",
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
        },
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "singleversionroot": false,
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
        },
        "foos": {
          "plural": "foos",
          "singular": "foo",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "singleversionroot": false,
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
            "foourl": {
              "name": "foourl",
              "type": "url"
            },
            "fooproxyurl": {
              "name": "fooproxyurl",
              "type": "url"
            },
            "foo": {
              "name": "foo",
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
      "required": true,
      "default": "1.0-rc1"
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
              "required": true,
              "default": "none"
            },
            "compatibilityauthority": {
              "name": "compatibilityauthority",
              "type": "string",
              "enum": [
                "external",
                "server"
              ]
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

	// Now a simple $include that works
	buf, err = os.ReadFile("files/dir/model-dirs-inc-docs.json")
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/model", string(buf), 200, `{
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
    },
    "docsurl": {
      "name": "docsurl",
      "type": "url",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "docscount": {
      "name": "docscount",
      "type": "uinteger",
      "readonly": true,
      "required": true
    },
    "docs": {
      "name": "docs",
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
        "formatsurl": {
          "name": "formatsurl",
          "type": "url",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "formatscount": {
          "name": "formatscount",
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "formats": {
          "name": "formats",
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
        "formats": {
          "plural": "formats",
          "singular": "format",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "singleversionroot": false,
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
            "formaturl": {
              "name": "formaturl",
              "type": "url"
            },
            "formatproxyurl": {
              "name": "formatproxyurl",
              "type": "url"
            },
            "format": {
              "name": "format",
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
    },
    "docsurl": {
      "name": "docsurl",
      "type": "url",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "docscount": {
      "name": "docscount",
      "type": "uinteger",
      "readonly": true,
      "required": true
    },
    "docs": {
      "name": "docs",
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
        "formatsurl": {
          "name": "formatsurl",
          "type": "url",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "formatscount": {
          "name": "formatscount",
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "formats": {
          "name": "formats",
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
        "formats": {
          "plural": "formats",
          "singular": "format",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "singleversionroot": false,
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
            "formaturl": {
              "name": "formaturl",
              "type": "url"
            },
            "formatproxyurl": {
              "name": "formatproxyurl",
              "type": "url"
            },
            "format": {
              "name": "format",
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
    },
    "docsurl": {
      "name": "docsurl",
      "type": "url",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "docscount": {
      "name": "docscount",
      "type": "uinteger",
      "readonly": true,
      "required": true
    },
    "docs": {
      "name": "docs",
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
        "formatsurl": {
          "name": "formatsurl",
          "type": "url",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "formatscount": {
          "name": "formatscount",
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "formats": {
          "name": "formats",
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
        "formats": {
          "plural": "formats",
          "singular": "format",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "singleversionroot": false,
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
            "formaturl": {
              "name": "formaturl",
              "type": "url"
            },
            "formatproxyurl": {
              "name": "formatproxyurl",
              "type": "url"
            },
            "format": {
              "name": "format",
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
    },
    "docsurl": {
      "name": "docsurl",
      "type": "url",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "docscount": {
      "name": "docscount",
      "type": "uinteger",
      "readonly": true,
      "required": true
    },
    "docs": {
      "name": "docs",
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
        "formatsurl": {
          "name": "formatsurl",
          "type": "url",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "formatscount": {
          "name": "formatscount",
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "formats": {
          "name": "formats",
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
        "formats": {
          "plural": "formats",
          "singular": "format",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "singleversionroot": false,
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
            "formaturl": {
              "name": "formaturl",
              "type": "url"
            },
            "formatproxyurl": {
              "name": "formatproxyurl",
              "type": "url"
            },
            "format": {
              "name": "format",
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

// Verify that the server will add the missing fields, like name,plural
func TestModelMissingFields(t *testing.T) {
	reg := NewRegistry("TestModelMissingFields")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "PUT", "/model", `{
  "attributes": {
    "specversion": {
	  "type": "string",
	  "readonly": true,
	  "required": true
    },
    "regext": {
      "type": "string"
    }
  },
  "groups": {
    "dirs": {
      "singular": "dir",
      "attributes": {
        "dirid": {
		  "type": "string",
		  "immutable": true,
		  "required": true
        },
        "gext": {
          "type": "string"
        }
      },
      "resources": {
        "files": {
          "singular": "file",
          "attributes": {
            "fileid": {
			  "type": "string",
		      "immutable": true,
		      "required": true
            },
            "rext": {
              "type": "integer"
            }
          },
          "metaattributes": {
            "fileid": {
			  "type": "string",
		      "immutable": true,
		      "required": true
            },
            "mext": {
              "type": "boolean"
            }
          }
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
    "regext": {
      "name": "regext",
      "type": "string"
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
        "gext": {
          "name": "gext",
          "type": "string"
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
            "rext": {
              "name": "rext",
              "type": "integer"
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
            "mext": {
              "name": "mext",
              "type": "boolean"
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
