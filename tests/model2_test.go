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

	reg.Model.AddLabel("reg-label", "reg-value")

	gm, err := reg.Model.AddGroupModel("gms1", "gm1")
	XCheck(t, gm != nil && err == nil, "gm should have worked")
	gm.AddLabel("g-label", "g-value")

	rm, err := gm.AddResourceModel("rms", "rm", 0, true, true, true)
	XCheck(t, rm != nil && err == nil, "rm should have worked: %s", err)
	rm.AddLabel("r-label", "r-value")

	reg.SaveAllAndCommit()
	reg.Refresh(registry.FOR_WRITE)

	XCheckGet(t, reg, "/model", `{
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
          "labels": {
            "r-label": "r-value"
          },
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
            "rmbase64": {
              "name": "rmbase64",
              "type": "string"
            }
          },
          "resourceattributes": {
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
}
`)

	XNoErr(t, reg.Refresh(registry.FOR_WRITE))
	reg.LoadModel()

	gm = reg.Model.FindGroupModel(gm.Plural)
	rm = gm.Resources[rm.Plural]

	reg.Model.RemoveLabel("reg-label")
	gm.RemoveLabel("g-label")
	rm.RemoveLabel("r-label")

	XCheckGet(t, reg, "/model", `{
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
            "rmbase64": {
              "name": "rmbase64",
              "type": "string"
            }
          },
          "resourceattributes": {
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
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
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
}
`)

	XHTTP(t, reg, "GET", "/model", ``, 200, `{
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
          "labels": {
            "r-label": "r-value"
          },
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
}
`)

	// Now some errors
	XHTTP(t, reg, "PUT", "/modelsource", `{
      "labels":{"bad one": "1"}
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: map key name \"bad one\" in \"labels\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "map key name \"bad one\" in \"labels\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$"
  },
  "source": "e4e59b8a76c4:registry:shared_model:65"
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
      "groups":{
        "dirs": {
          "singular":"dir",
          "labels":{"bad one": "1"}}}
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: map key name \"bad one\" in \"groups.dirs.labels\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "map key name \"bad one\" in \"groups.dirs.labels\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$"
  },
  "source": "e4e59b8a76c4:registry:shared_model:65,registry:shared_model:2097"
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
      "groups":{
        "dirs": {
          "singular":"dir",
          "resources":{
            "files":{
              "singular":"file",
              "labels":{"bad one": "1"}}}}}
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: map key name \"bad one\" in \"groups.dirs.resources.files.labels\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$.",
  "subject": "/model",
  "args": {
    "error_detail": "map key name \"bad one\" in \"groups.dirs.resources.files.labels\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$"
  },
  "source": "e4e59b8a76c4:registry:shared_model:65,registry:shared_model:2402"
}
`)

}

// Make sure that we can use spec defined attribute names in a nested
// Object w/o the code mucking with it. There are some spots in the code
// where we'll do thinkg like skip certain attributes but we should only
// do that at the top level of the entire, not within a nested object.
func TestModelUseSpecAttrs(t *testing.T) {
	reg := NewRegistry("TestModelUseSpecAttrs")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, false)

	// Registry level
	obj, err := reg.Model.AddAttrObj("obj")
	XNoErr(t, err)
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
		XNoErr(t, err)
		vals["obj."+prop.Name] = v

		if prop.Name == "id" {
			_, err := obj.AddAttr("registryid", typ)
			XNoErr(t, err)
			vals["obj.registryid"] = 10
		}
	}

	for k, v := range vals {
		XNoErr(t, reg.SetSave(k, v))
	}

	// Group level
	obj, err = gm.AddAttrObj("obj")
	XNoErr(t, err)
	vals = map[string]any{}

	d1, err := reg.AddGroup("dirs", "d1")
	XNoErr(t, err)
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
		XNoErr(t, err)
		XNoErr(t, d1.SetSave("obj."+prop.Name, v))

		if prop.Name == "id" {
			_, err = obj.AddAttr("registryid", typ)
			_, err = obj.AddAttr("dirid", typ)
			_, err = obj.AddAttr("fileid", typ)
			XNoErr(t, err)
			XNoErr(t, d1.SetSave("obj.registryid", 10))
			XNoErr(t, d1.SetSave("obj.dirid", 5))
			XNoErr(t, d1.SetSave("obj.fileid", 6))
		}
	}

	// Resource level
	obj, err = rm.AddAttrObj("obj")
	XNoErr(t, err)

	objMeta, err := rm.AddMetaAttrObj("obj")
	XNoErr(t, err)

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
		XNoErr(t, err)

		_, err = objMeta.AddAttr(prop.Name, typ)
		XNoErr(t, err)

		vals["obj."+prop.Name] = v

		if prop.Name == "id" {
			_, err = obj.AddAttr("registryid", typ)
			_, err = obj.AddAttr("dirid", typ)
			_, err = obj.AddAttr("fileid", typ)
			_, err = obj.AddAttr("file", typ)
			_, err = obj.AddAttr("filebase64", typ)
			_, err = obj.AddAttr("fileurl", typ)
			XNoErr(t, err)
			_, err = objMeta.AddAttr("registryid", typ)
			_, err = objMeta.AddAttr("dirid", typ)
			_, err = objMeta.AddAttr("fileid", typ)
			_, err = objMeta.AddAttr("file", typ)
			_, err = objMeta.AddAttr("filebase64", typ)
			_, err = objMeta.AddAttr("fileurl", typ)
			XNoErr(t, err)
			vals["obj.registryid"] = 10
			vals["obj.dirid"] = 5
			vals["obj.fileid"] = 6
			vals["obj.file"] = 4
			vals["obj.filebase64"] = 10
			vals["obj.fileurl"] = 7
		}
	}

	r1, err := d1.AddResource("files", "f1", "v1")
	XNoErr(t, err)

	v1, err := r1.FindVersion("v1", false, registry.FOR_WRITE)
	XNoErr(t, err)

	meta, err := r1.FindMeta(false, registry.FOR_WRITE)
	XNoErr(t, err)

	for k, v := range vals {
		XNoErr(t, v1.SetSave(k, v))
		XNoErr(t, meta.SetSave(k, v))
	}

	XHTTP(t, reg, "GET", "?inline=*,model", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestModelUseSpecAttrs",
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
    "icon": 4,
    "id": 2,
    "isdefault": 9,
    "labels": 6,
    "meta": 4,
    "metaurl": 7,
    "model": 5,
    "modelsource": 11,
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
          "icon": {
            "name": "icon",
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
          "modelsource": {
            "name": "modelsource",
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
              "icon": {
                "name": "icon",
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
              "modelsource": {
                "name": "modelsource",
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
                  "icon": {
                    "name": "icon",
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
                  "modelsource": {
                    "name": "modelsource",
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
                  "icon": {
                    "name": "icon",
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
                  "modelsource": {
                    "name": "modelsource",
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
        "icon": 4,
        "id": 2,
        "isdefault": 9,
        "labels": 6,
        "meta": 4,
        "metaurl": 7,
        "model": 5,
        "modelsource": 11,
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
            "icon": 4,
            "id": 2,
            "isdefault": 9,
            "labels": 6,
            "meta": 4,
            "metaurl": 7,
            "model": 5,
            "modelsource": 11,
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
              "icon": 4,
              "id": 2,
              "isdefault": 9,
              "labels": 6,
              "meta": 4,
              "metaurl": 7,
              "model": 5,
              "modelsource": 11,
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
                "icon": 4,
                "id": 2,
                "isdefault": 9,
                "labels": 6,
                "meta": 4,
                "metaurl": 7,
                "model": 5,
                "modelsource": 11,
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

	XHTTP(t, reg, "PUT", "/modelsource", `{
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
}
`)

	XHTTP(t, reg, "GET", "/model", ``, 200, `{
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
          "modelversion": "3.2",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "singleversionroot": false,
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
            "database64": {
              "name": "database64",
              "type": "string"
            }
          },
          "resourceattributes": {
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
        },
        "files": {
          "plural": "files",
          "singular": "file",
          "modelversion": "3.1",
          "compatiblewith": "some-url2",
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
        },
        "foos": {
          "plural": "foos",
          "singular": "foo",
          "compatiblewith": "some-url3",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": true,
          "singleversionroot": false,
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
            "foobase64": {
              "name": "foobase64",
              "type": "string"
            }
          },
          "resourceattributes": {
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
}
`)
}

func TestModelIncludes(t *testing.T) {
	reg := NewRegistry("TestModelCompatibleWith")
	defer PassDeleteReg(t, reg)

	// First try w/o any includes
	buf, err := os.ReadFile("files/sample-model.json")
	XNoErr(t, err)

	XHTTP(t, reg, "PUT", "/modelsource", string(buf), 200, `{
  "attributes": {
    "specversion": {
      "type": "string",
      "readonly": true,
      "required": true,
      "default": "`+SPECVERSION+`"
    },
    "registryid": {
      "type": "string",
      "immutable": true,
      "readonly": true,
      "required": true
    },
    "self": {
      "type": "url",
      "immutable": true,
      "readonly": true,
      "required": true
    },
    "xid": {
      "type": "xid",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "epoch": {
      "type": "uinteger",
      "readonly": true,
      "required": true
    },
    "name": {
      "type": "string"
    },
    "description": {
      "type": "string"
    },
    "documentation": {
      "type": "url"
    },
    "icon": {
      "type": "url"
    },
    "labels": {
      "type": "map",
      "item": {
        "type": "string"
      }
    },
    "createdat": {
      "type": "timestamp",
      "required": true
    },
    "modifiedat": {
      "type": "timestamp",
      "required": true
    },
    "capabilities": {
      "type": "object",
      "attributes": {
        "*": {
          "type": "any"
        }
      }
    },
    "model": {
      "type": "object",
      "attributes": {
        "*": {
          "type": "any"
        }
      }
    },
    "dirsurl": {
      "type": "url",
      "immutable": true,
      "readonly": true,
      "required": true
    },
    "dirscount": {
      "type": "uinteger",
      "readonly": true,
      "required": true
    },
    "dirs": {
      "type": "map",
      "item": {
        "type": "object",
        "attributes": {
          "*": {
            "type": "any"
          }
        }
      }
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
        "self": {
          "type": "url",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "xid": {
          "type": "xid",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "epoch": {
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "name": {
          "type": "string"
        },
        "description": {
          "type": "string"
        },
        "documentation": {
          "type": "url"
        },
        "icon": {
          "type": "url"
        },
        "labels": {
          "type": "map",
          "item": {
            "type": "string"
          }
        },
        "createdat": {
          "type": "timestamp",
          "required": true
        },
        "modifiedat": {
          "type": "timestamp",
          "required": true
        },
        "filesurl": {
          "type": "url",
          "immutable": true,
          "readonly": true,
          "required": true
        },
        "filescount": {
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "files": {
          "type": "map",
          "item": {
            "type": "object",
            "attributes": {
              "*": {
                "type": "any"
              }
            }
          }
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
            "versionid": {
              "type": "string",
              "immutable": true,
              "required": true
            },
            "self": {
              "type": "url",
              "immutable": true,
              "readonly": true,
              "required": true
            },
            "xid": {
              "type": "xid",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "epoch": {
              "type": "uinteger",
              "readonly": true,
              "required": true
            },
            "name": {
              "type": "string"
            },
            "isdefault": {
              "type": "boolean",
              "readonly": true,
              "required": true,
              "default": false
            },
            "description": {
              "type": "string"
            },
            "documentation": {
              "type": "url"
            },
            "icon": {
              "type": "url"
            },
            "labels": {
              "type": "map",
              "item": {
                "type": "string"
              }
            },
            "createdat": {
              "type": "timestamp",
              "required": true
            },
            "modifiedat": {
              "type": "timestamp",
              "required": true
            },
            "contenttype": {
              "type": "string"
            },
            "versionsurl": {
              "type": "url",
              "immutable": true,
              "readonly": true,
              "required": true
            },
            "versionscount": {
              "type": "uinteger",
              "readonly": true,
              "required": true
            },
            "versions": {
              "type": "map",
              "item": {
                "type": "object",
                "attributes": {
                  "*": {
                    "type": "any"
                  }
                }
              }
            }
          },
          "metaattributes": {
            "fileid": {
              "type": "string",
              "immutable": true,
              "required": true
            },
            "self": {
              "type": "url",
              "immutable": true,
              "readonly": true,
              "required": true
            },
            "xid": {
              "type": "xid",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "xref": {
              "type": "url"
            },
            "epoch": {
              "type": "uinteger",
              "readonly": true,
              "required": true
            },
            "readonly": {
              "type": "boolean",
              "readonly": true,
              "required": true,
              "default": false
            },
            "compatibility": {
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
              "type": "string",
              "enum": [
                "external",
                "server"
              ]
            },
            "deprecated": {
              "type": "object",
              "attributes": {
                "effective": {
                  "type": "timestamp"
                },
                "removal": {
                  "type": "timestamp"
                },
                "alternative": {
                  "type": "url"
                },
                "documentation": {
                  "type": "url"
                },
                "*": {
                  "type": "any"
                }
              }
            },
            "defaultversionid": {
              "type": "string",
              "required": true
            },
            "defaultversionurl": {
              "type": "url",
              "readonly": true,
              "required": true
            },
            "defaultversionsticky": {
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

	XHTTP(t, reg, "GET", "/model", "", 200, `{
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "required": true,
      "default": "`+SPECVERSION+`"
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
}
`)

	// Now a simple $include that works
	buf, err = os.ReadFile("files/dir/model-dirs-inc-docs.json")
	XNoErr(t, err)

	XHTTP(t, reg, "PUT", "/modelsource", string(buf), 200, `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file"
        }
      }
    },
    "$include": "http://localhost:8282/dir/model-docs.json#groups"
  }
}
`)

	XHTTP(t, reg, "GET", "/model", string(buf), 200, `{
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
            "formatbase64": {
              "name": "formatbase64",
              "type": "string"
            }
          },
          "resourceattributes": {
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

	XHTTP(t, reg, "PUT", "/modelsource", str, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: error processing JSON: Not allowed to access file: ../foo.json.",
  "args": {
    "error_detail": "error processing JSON: Not allowed to access file: ../foo.json"
  },
  "source": "e4e59b8a76c4:common:utils:427"
}
`)

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
	XHTTP(t, reg, "PUT", "/modelsource", str, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: error processing JSON: Not allowed to access file: /foo.json.",
  "args": {
    "error_detail": "error processing JSON: Not allowed to access file: /foo.json"
  },
  "source": "e4e59b8a76c4:common:utils:427"
}
`)

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
	XHTTP(t, reg, "PUT", "/modelsource", str, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: error processing JSON: Get \"http://bogus.bogus.bogus.bogus.com/bogus.json\": dial tcp: lookup bogus.bogus.bogus.bogus.com on 127.0.0.53:53: no such host.",
  "args": {
    "error_detail": "error processing JSON: Get \"http://bogus.bogus.bogus.bogus.com/bogus.json\": dial tcp: lookup bogus.bogus.bogus.bogus.com on 127.0.0.53:53: no such host"
  },
  "source": "e4e59b8a76c4:common:utils:427"
}
`)

	// nested include with http ref
	str = `
{
  "$include": "http://localhost:8282/dir/model-dirs-inc-docs.json"
}
`

	XHTTP(t, reg, "PUT", "/modelsource", str, 200, `{
  "$include": "http://localhost:8282/dir/model-dirs-inc-docs.json"
}
`)

	XHTTP(t, reg, "GET", "/model", str, 200, `{
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
            "formatbase64": {
              "name": "formatbase64",
              "type": "string"
            }
          },
          "resourceattributes": {
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
}
`)

	// nested include with local ref
	str = `
{
  "$include": "http://localhost:8282/dir/model-dirs-inc-docs-indirect.json"
}
`

	XHTTP(t, reg, "PUT", "/modelsource", str, 200, `{
  "$include": "http://localhost:8282/dir/model-dirs-inc-docs-indirect.json"
}
`)

	XHTTP(t, reg, "GET", "/model", "", 200, `{
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
            "formatbase64": {
              "name": "formatbase64",
              "type": "string"
            }
          },
          "resourceattributes": {
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
}
`)

	// nested include with local ref using ..
	str = `
{
  "$include": "http://localhost:8282/dir/model-dirs-inc-docs-indirect2.json"
}
`

	XHTTP(t, reg, "PUT", "/modelsource", str, 200, `{
  "$include": "http://localhost:8282/dir/model-dirs-inc-docs-indirect2.json"
}
`)

	XHTTP(t, reg, "GET", "/model", "", 200, `{
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
            "formatbase64": {
              "name": "formatbase64",
              "type": "string"
            }
          },
          "resourceattributes": {
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
}
`)

}

// Verify that the server will add the missing fields, like name,plural
func TestModelMissingFields(t *testing.T) {
	reg := NewRegistry("TestModelMissingFields")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "PUT", "/modelsource", `{
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
}
`)

	XHTTP(t, reg, "GET", "/model", ``, 200, `{
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
