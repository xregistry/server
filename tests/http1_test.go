package tests

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestHTTPhtml(t *testing.T) {
	reg := NewRegistry("TestHTTPhtml")
	defer PassDeleteReg(t, reg)

	// Check as part of Reg request
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "?html",
		URL:        "?html",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Content-Type:text/html"},
		ResBody: `<pre>
{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPhtml",
  "self": "<a href="http://localhost:8181/?html">http://localhost:8181/?html</a>",
  "xid": "/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`,
	})
}

func TestHTTPModel(t *testing.T) {
	reg := NewRegistry("TestHTTPModel")
	defer PassDeleteReg(t, reg)

	// Check as part of Reg request
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "?inline=model",
		URL:        "?inline=model",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPModel",
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
  }
}
`,
	})

	// Just model, no reg content
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "/model",
		URL:        "/model",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
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
}
`,
	})

	// Error creating - wrong endpoint
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Create empty model",
		URL:        "/model",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,

		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody: `Use "/modelsource" instead of "/model"
`,
	})

	// Create model tests
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Create empty model",
		URL:        "/modelsource",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "check Create empty model",
		URL:        "/model",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    `{}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
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
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Create model - defaults",
		URL:        "/modelsource",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
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
    }

  }
}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
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
    }
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Create model - defaults",
		URL:        "/model",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
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
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Create model - full",
		URL:        "/modelsource",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": false,
          "singleversionroot": false
        }
      }
    }
  }
}`,

		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "setdefaultversionsticky": true,
          "hasdocument": false,
          "singleversionroot": false
        }
      }
    }
  }
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Create model - full",
		URL:        "/model",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
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
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Modify description",
		URL:        "/modelsource",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "attributes": {
    "description": {
      "name": "description",
      "type": "string",
      "enum": [ "one", "two" ]
    }
  },
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
    }
  }
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "attributes": {
    "description": {
      "name": "description",
      "type": "string",
      "enum": [
        "one",
        "two"
      ]
    }
  },
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
    }
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Modify description",
		URL:        "/model",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
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
      "type": "string",
      "enum": [
        "one",
        "two"
      ]
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
`,
	})

	xHTTP(t, reg, "PUT", "/", `{"description": "testing"}`, 400,
		`Attribute "description"(testing) must be one of the enum values: one, two`+"\n")

	xHTTP(t, reg, "PUT", "/", `{}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPModel",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0
}
`)

	xHTTP(t, reg, "PUT", "/", `{"description": "two"}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPModel",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "description": "two",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0
}
`)
}

func TestHTTPRegistry(t *testing.T) {
	reg := NewRegistry("TestHTTPRegistry")
	defer PassDeleteReg(t, reg)

	_, err := reg.Model.AddAttr("myany", ANY)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("mybool", BOOLEAN)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("mydec", DECIMAL)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myint", INTEGER)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("mystr", STRING)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("mytime", TIMESTAMP)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myuint", UINTEGER)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myuri", URI)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myuriref", URI_REFERENCE)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myuritemplate", URI_TEMPLATE)
	xCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myurl", URL)
	xCheckErr(t, err, "")

	attr, err := reg.Model.AddAttrObj("myobj1")
	xCheckErr(t, err, "")
	_, err = attr.AddAttr("mystr1", STRING)
	xCheckErr(t, err, "")
	_, err = attr.AddAttr("myint1", INTEGER)
	xCheckErr(t, err, "")
	_, err = attr.AddAttr("*", ANY)
	xCheckErr(t, err, "")

	attr, _ = reg.Model.AddAttrObj("myobj2")
	attr.AddAttr("mystr2", STRING)
	obj2, err := attr.AddAttrObj("myobj2_1")
	xCheckErr(t, err, "")
	_, err = obj2.AddAttr("*", INTEGER)
	xCheckErr(t, err, "")

	item := registry.NewItemType(ANY)
	attr, err = reg.Model.AddAttrArray("myarrayany", item)
	xCheckErr(t, err, "")
	attr, err = reg.Model.AddAttrMap("mymapany", item)
	xCheckErr(t, err, "")

	item = registry.NewItemType(UINTEGER)
	attr, err = reg.Model.AddAttrArray("myarrayuint", item)
	xCheckErr(t, err, "")
	attr, err = reg.Model.AddAttrMap("mymapuint", item)
	xCheckErr(t, err, "")

	item = registry.NewItemObject()
	attr, err = reg.Model.AddAttrArray("myarrayemptyobj", item)
	xCheckErr(t, err, "")

	item = registry.NewItemObject()
	item.AddAttr("mapobj_int", INTEGER)
	attr, err = reg.Model.AddAttrMap("mymapobj", item)
	xCheckErr(t, err, "")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - empty string id",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "{ \"registryid\": \"\" }",
		Code:       400,
		ResHeaders: []string{},
		ResBody:    "\"registryid\" can't be an empty string\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - empty",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "{}",
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - empty json",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "{}",
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - good epoch",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "epoch": 3
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad epoch",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "epoch":33
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Attribute \"epoch\"(33) doesn't match existing value (4)\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - full good",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPRegistry",
  "epoch": 4,

  "myany": 5.5,
  "mybool": true,
  "mydec": 2.4,
  "myint": -666,
  "mystr": "hello",
  "mytime": "2024-01-01T12:01:02Z",
  "myuint": 123,
  "myuri": "http://foo.com",
  "myuriref": "/foo",
  "myuritemplate": "...",
  "myurl": "http://someurl.com",
  "myobj1": {
    "mystr1": "str1",
    "myint1": 345,
    "myobj1_ext": 9.2
  },
  "myobj2": {
    "mystr2": "str2",
    "myobj2_1": {
      "myobj2_1_ext": 444
    }
  },
  "myarrayany": [
    { "any1": -333},
    "any2-str"
  ],
  "mymapany": {
    "key1": 1,
    "key2": "2"
  },
  "myarrayuint": [ 2, 999 ],
  "myarrayemptyobj": [],
  "mymapobj": {
    "mymapobj_k1": { "mapobj_int": 333 }
  }
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myany": 5.5,
  "myarrayany": [
    {
      "any1": -333
    },
    "any2-str"
  ],
  "myarrayemptyobj": [],
  "myarrayuint": [
    2,
    999
  ],
  "mybool": true,
  "mydec": 2.4,
  "myint": -666,
  "mymapany": {
    "key1": 1,
    "key2": "2"
  },
  "mymapobj": {
    "mymapobj_k1": {
      "mapobj_int": 333
    }
  },
  "myobj1": {
    "myint1": 345,
    "myobj1_ext": 9.2,
    "mystr1": "str1"
  },
  "myobj2": {
    "myobj2_1": {
      "myobj2_1_ext": 444
    },
    "mystr2": "str2"
  },
  "mystr": "hello",
  "mytime": "2024-01-01T12:01:02Z",
  "myuint": 123,
  "myuri": "http://foo.com",
  "myuriref": "/foo",
  "myuritemplate": "...",
  "myurl": "http://someurl.com"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad object",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "mymapobj": {
    "mapobj_int": 333
  }
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody: `Attribute "mymapobj.mapobj_int" must be a map[string] or object
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - full empties",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPRegistry",
  "epoch": 5,

  "myany": 5.5,
  "myint": 4.0,
  "mybool": null,
  "myuri": null,
  "myobj1": {},
  "myobj2": null,
  "myarrayany": [],
  "mymapany": {},
  "myarrayuint": null,
  "myarrayemptyobj": [],
  "mymapobj": {
    "mymapobj_key1": {}
  },
  "mymapuint": {
    "asd": null
  }
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 6,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myany": 5.5,
  "myarrayany": [],
  "myarrayemptyobj": [],
  "myint": 4,
  "mymapany": {},
  "mymapobj": {
    "mymapobj_key1": {}
  },
  "mymapuint": {},
  "myobj1": {}
}
`,
	})

	type typeTest struct {
		request  string
		response string
	}

	typeTests := []typeTest{
		{request: `{"epoch":123}`,
			response: `Attribute "epoch"(123) doesn't match existing value (6)`},
		{request: `{"epoch":-123}`,
			response: `Attribute "epoch" must be a uinteger`},
		{request: `{"epoch":"asd"}`,
			response: `Attribute "epoch" must be a uinteger`},
		{request: `{"mybool":123}`,
			response: `Attribute "mybool" must be a boolean`},
		{request: `{"mybool":"False"}`,
			response: `Attribute "mybool" must be a boolean`},
		{request: `{"mydec":[ 1 ]}`,
			response: `Attribute "mydec" must be a decimal`},
		{request: `{"mydec": "asd" }`,
			response: `Attribute "mydec" must be a decimal`},
		{request: `{"myint": 1.01 }`,
			response: `Attribute "myint" must be an integer`},
		{request: `{"myint": {} }`,
			response: `Attribute "myint" must be an integer`},
		{request: `{"mystr": {} }`,
			response: `Attribute "mystr" must be a string`},
		{request: `{"mystr": 123 }`,
			response: `Attribute "mystr" must be a string`},
		{request: `{"mystr": true }`,
			response: `Attribute "mystr" must be a string`},
		{request: `{"mytime": true }`,
			response: `Attribute "mytime" must be a timestamp`},
		{request: `{"mytime": "12-12-12" }`,
			response: `Attribute "mytime" is a malformed timestamp`},
		{request: `{"myuint": "str" }`,
			response: `Attribute "myuint" must be a uinteger`},
		{request: `{"myuint": "123" }`,
			response: `Attribute "myuint" must be a uinteger`},
		{request: `{"myuint": -123 }`,
			response: `Attribute "myuint" must be a uinteger`},
		{request: `{"myuri": 123 }`,
			response: `Attribute "myuri" must be a uri`},
		{request: `{"myuriref": 123 }`,
			response: `Attribute "myuriref" must be a uri-reference`},
		{request: `{"myuritemplate": 123 }`,
			response: `Attribute "myuritemplate" must be a uri-template`},
		{request: `{"myurl": 123 }`,
			response: `Attribute "myurl" must be a url`},
		{request: `{"myobj1": 123 }`,
			response: `Attribute "myobj1" must be a map[string] or object`},
		{request: `{"myobj1": [ 123 ] }`,
			response: `Attribute "myobj1" must be a map[string] or object`},
		{request: `{"myobj1": { "mystr1": 123 } }`,
			response: `Attribute "myobj1.mystr1" must be a string`},
		{request: `{"myobj2": { "ext": 123 } }`,
			response: `Invalid extension(s) in "myobj2": ext`},
		{request: `{"myobj2": { "myobj2_1": { "ext": "str" } } }`,
			response: `Attribute "myobj2.myobj2_1.ext" must be an integer`},
		{request: `{"myarrayuint": [ 123, -123 ] }`,
			response: `Attribute "myarrayuint[1]" must be a uinteger`},
		{request: `{"myarrayuint": [ "asd" ] }`,
			response: `Attribute "myarrayuint[0]" must be a uinteger`},
		{request: `{"myarrayuint": [ 123, null] }`,
			response: `Attribute "myarrayuint[1]" must be a uinteger`},
		{request: `{"myarrayuint": 123 }`,
			response: `Attribute "myarrayuint" must be an array`},
		{request: `{"mymapuint": 123 }`,
			response: `Attribute "mymapuint" must be a map`},
		{request: `{"mymapuint": { "asd" : -123 }}`,
			response: `Attribute "mymapuint.asd" must be a uinteger`},
		// {request: `{"mymapuint": { "asd" : null }}`,
		// response: `Attribute "mymapuint.asd" must be a uinteger`},
		{request: `{"myarrayemptyobj": [ { "asd": true } ] }`,
			response: `Invalid extension(s) in "myarrayemptyobj[0]": asd`},
		{request: `{"myarrayemptyobj": [ [ true ] ] }`,
			response: `Attribute "myarrayemptyobj[0]" must be a map[string] or object`},
		{request: `{"mymapobj": { "asd" : { "mapobj_int" : true } } }`,
			response: `Attribute "mymapobj.asd.mapobj_int" must be an integer`},
		{request: `{"mymapobj": { "asd" : { "qwe" : true } } }`,
			response: `Invalid extension(s) in "mymapobj.asd": qwe`},
		{request: `{"mymapobj": [ true ]}`,
			response: `Attribute "mymapobj" must be a map`},
	}

	for _, test := range typeTests {
		xCheckHTTP(t, reg, &HTTPTest{
			Name:       "PUT reg - bad type - request: " + test.request,
			URL:        "/",
			Method:     "PUT",
			ReqHeaders: []string{},
			ReqBody:    test.request,
			Code:       400,
			ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
			ResBody:    test.response + "\n",
		})
	}

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad self - ignored",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "self": 123
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody: `Attribute "self" must be a url
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad xid - ignored",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "xid": 123
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody: `Attribute "xid" must be an xid
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad id",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "registryid": 123
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Attribute \"registryid\" must be a string\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad id",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "registryid": "foo"
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody: `The "registryid" attribute must be set to "TestHTTPRegistry", not "foo"
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - options",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "documentation": "docs"
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 7,
  "documentation": "docs",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - options - del",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "registryid": null,
  "self": null,
  "xid": null
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 8,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - swap any - 1",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "myany": 5.5,
  "mymapany": {
    "any1": {
	  "foo": "bar"
	}
  }
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 9,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myany": 5.5,
  "mymapany": {
    "any1": {
      "foo": "bar"
    }
  }
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - swap any - 2",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "myany": "foo",
  "mymapany": {
    "any1": 2.3
  }
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 10,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myany": "foo",
  "mymapany": {
    "any1": 2.3
  }
}
`})

}

func TestHTTPGroups(t *testing.T) {
	reg := NewRegistry("TestHTTPGroups")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddAttr("format", STRING)
	gm.AddResourceModel("files", "file", 0, true, true, true)

	attr, _ := gm.AddAttrObj("myobj")
	attr.AddAttr("foo", STRING)
	attr.AddAttr("*", ANY)

	item := registry.NewItemType(ANY)
	attr, _ = gm.AddAttrArray("myarray", item)
	attr, _ = gm.AddAttrMap("mymap", item)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT groups - fail",
		URL:        "/dirs",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       405,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "PUT not allowed on collections\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "Create group - {}",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    "{}",
		Code:       200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST group - full, single",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
  "dir1": {
    "dirid":"dir1",
    "name":"my group",
    "description":"desc",
    "documentation":"docs-url",
    "labels": {
      "label1": "value1",
      "label2": "5",
      "label3": "123.456",
      "label4": "",
      "label5": null
    },
    "format":"my group",
    "myarray": [ "hello", 5 ],
    "mymap": { "item1": 5.5 },
    "myobj": { "item2": [ "hi" ] }
  }
}`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "dir1": {
    "dirid": "dir1",
    "self": "http://localhost:8181/dirs/dir1",
    "xid": "/dirs/dir1",
    "epoch": 1,
    "name": "my group",
    "description": "desc",
    "documentation": "docs-url",
    "labels": {
      "label1": "value1",
      "label2": "5",
      "label3": "123.456",
      "label4": ""
    },
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "format": "my group",
    "myarray": [
      "hello",
      5
    ],
    "mymap": {
      "item1": 5.5
    },
    "myobj": {
      "item2": [
        "hi"
      ]
    },

    "filesurl": "http://localhost:8181/dirs/dir1/files",
    "filescount": 0
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST group - full, multiple",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
  "dir2": {
    "dirid":"dir2",
    "name":"my group",
    "description":"desc",
    "documentation":"docs-url",
    "labels": {
      "label1": "value1",
      "label2": "5",
      "label3": "123.456",
      "label4": "",
      "label5": null
    },
    "format":"my group",
    "myarray": [ "hello", 5 ],
    "mymap": { "item1": 5.5 },
    "myobj": { "item2": [ "hi" ] }
  },
  "dir3": {}
}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "dir2": {
    "dirid": "dir2",
    "self": "http://localhost:8181/dirs/dir2",
    "xid": "/dirs/dir2",
    "epoch": 1,
    "name": "my group",
    "description": "desc",
    "documentation": "docs-url",
    "labels": {
      "label1": "value1",
      "label2": "5",
      "label3": "123.456",
      "label4": ""
    },
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "format": "my group",
    "myarray": [
      "hello",
      5
    ],
    "mymap": {
      "item1": 5.5
    },
    "myobj": {
      "item2": [
        "hi"
      ]
    },

    "filesurl": "http://localhost:8181/dirs/dir2/files",
    "filescount": 0
  },
  "dir3": {
    "dirid": "dir3",
    "self": "http://localhost:8181/dirs/dir3",
    "xid": "/dirs/dir3",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/dir3/files",
    "filescount": 0
  }
}
`,
	})

	xHTTP(t, reg, "GET", "/dirs", "", 200, `{
  "dir1": {
    "dirid": "dir1",
    "self": "http://localhost:8181/dirs/dir1",
    "xid": "/dirs/dir1",
    "epoch": 1,
    "name": "my group",
    "description": "desc",
    "documentation": "docs-url",
    "labels": {
      "label1": "value1",
      "label2": "5",
      "label3": "123.456",
      "label4": ""
    },
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "format": "my group",
    "myarray": [
      "hello",
      5
    ],
    "mymap": {
      "item1": 5.5
    },
    "myobj": {
      "item2": [
        "hi"
      ]
    },

    "filesurl": "http://localhost:8181/dirs/dir1/files",
    "filescount": 0
  },
  "dir2": {
    "dirid": "dir2",
    "self": "http://localhost:8181/dirs/dir2",
    "xid": "/dirs/dir2",
    "epoch": 1,
    "name": "my group",
    "description": "desc",
    "documentation": "docs-url",
    "labels": {
      "label1": "value1",
      "label2": "5",
      "label3": "123.456",
      "label4": ""
    },
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "format": "my group",
    "myarray": [
      "hello",
      5
    ],
    "mymap": {
      "item1": 5.5
    },
    "myobj": {
      "item2": [
        "hi"
      ]
    },

    "filesurl": "http://localhost:8181/dirs/dir2/files",
    "filescount": 0
  },
  "dir3": {
    "dirid": "dir3",
    "self": "http://localhost:8181/dirs/dir3",
    "xid": "/dirs/dir3",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "filesurl": "http://localhost:8181/dirs/dir3/files",
    "filescount": 0
  }
}
`)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST group - full, multiple - clear",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
  "dir1": {},
  "dir2": {},
  "dir3": {
    "description": "hello"
  }
}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "dir1": {
    "dirid": "dir1",
    "self": "http://localhost:8181/dirs/dir1",
    "xid": "/dirs/dir1",
    "epoch": 2,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "filesurl": "http://localhost:8181/dirs/dir1/files",
    "filescount": 0
  },
  "dir2": {
    "dirid": "dir2",
    "self": "http://localhost:8181/dirs/dir2",
    "xid": "/dirs/dir2",
    "epoch": 2,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "filesurl": "http://localhost:8181/dirs/dir2/files",
    "filescount": 0
  },
  "dir3": {
    "dirid": "dir3",
    "self": "http://localhost:8181/dirs/dir3",
    "xid": "/dirs/dir3",
    "epoch": 2,
    "description": "hello",
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",

    "filesurl": "http://localhost:8181/dirs/dir3/files",
    "filescount": 0
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST group - full, multiple - err",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
  "dir2": {
    "dirid":"dir2",
    "name":"my group",
    "description":"desc",
    "documentation":"docs-url",
    "labels": {
      "label1": "value1",
      "label2": "5",
      "label3": "123.456",
      "label4": "",
      "label5": null
    },
    "format":"my group",
    "myarray": [ "hello", 5 ],
    "mymap": { "item1": 5.5 },
    "myobj": { "item2": [ "hi" ] }
  },
  "dir3": {},
  "dir4": {
    "dirid": "dir44"
  }
}`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `The "dirid" attribute must be set to "dir4", not "dir44"
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT group - update",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "dirid":"dir1",
  "epoch": 2,
  "name":"my group new",
  "description":"desc new",
  "documentation":"docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "format": "myformat/1",
  "myarray": [],
  "mymap": {},
  "myobj": {}
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 3,
  "name": "my group new",
  "description": "desc new",
  "documentation": "docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "format": "myformat/1",
  "myarray": [],
  "mymap": {},
  "myobj": {},

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 0
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT group - update - null",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "dirid":"dir1",
  "epoch": 3,
  "name":"my group new",
  "description":"desc new",
  "documentation":"docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "format": "myformat/1",
  "myarray": null,
  "mymap": null,
  "myobj": null
}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 4,
  "name": "my group new",
  "description": "desc new",
  "documentation": "docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "format": "myformat/1",

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 0
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT group - update - err epoch",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "dirid":"dir1",
  "epoch": 10,
  "name":"my group new",
  "description":"desc new",
  "documentation":"docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "format":"myformat/1"
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Attribute \"epoch\"(10) doesn't match existing value (4)\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT group - update - err id",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{ "dirid":"dir2" }`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody: `The "dirid" attribute must be set to "dir1", not "dir2"
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT group - update - clear",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 5,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 0
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT group - create - error",
		URL:        "/dirs/dir2",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "dirid":"dir3",
  "name":"my group new",
  "description":"desc new",
  "documentation":"docs-url-new",
  "labels": {
    "label.new": "new"
  },
  "format": "myformat/1"
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody: `The "dirid" attribute must be set to "dir2", not "dir3"
`,
	})

}

func TestHTTPRegGroups(t *testing.T) {
	reg := NewRegistry("TestHTTPRegGroups")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	gm, _ = reg.Model.AddGroupModel("foos", "foo")
	gm.AddResourceModel("bars", "bat", 0, true, true, true)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH / - update name",
		URL:        "/",
		Method:     "PATCH",
		ReqBody:    `{"name":"hello"}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "1.0-rc1",
  "registryid": "TestHTTPRegGroups",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "name": "hello",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0,
  "foosurl": "http://localhost:8181/foos",
  "fooscount": 0
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - empty",
		URL:        "/",
		Method:     "POST",
		ReqBody:    "",
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "An HTTP body must be specified\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - empty",
		URL:        "/",
		Method:     "POST",
		ReqBody:    "{}",
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody:    `{}` + "\n",
	})

	// Check epoch didn't change
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "1.0-rc1",
  "registryid": "TestHTTPRegGroups",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "name": "hello",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0,
  "foosurl": "http://localhost:8181/foos",
  "fooscount": 0
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - one grouptype/no groups",
		URL:        "/",
		Method:     "POST",
		ReqBody:    `{"dirs":{}}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "dirs": {}
}
`,
	})

	// Check epoch didn't change
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "1.0-rc1",
  "registryid": "TestHTTPRegGroups",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "name": "hello",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0,
  "foosurl": "http://localhost:8181/foos",
  "fooscount": 0
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - one grouptype/one group",
		URL:        "/",
		Method:     "POST",
		ReqBody:    `{"dirs":{"d1":{}}}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "filescount": 0
    }
  }
}
`,
	})

	// Epoch bumped
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "1.0-rc1",
  "registryid": "TestHTTPRegGroups",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "name": "hello",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1,
  "foosurl": "http://localhost:8181/foos",
  "fooscount": 0
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - one grouptype/two groups",
		URL:        "/",
		Method:     "POST",
		ReqBody:    `{"dirs":{"d1":{},"d2":{}}}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "filescount": 0
    },
    "d2": {
      "dirid": "d2",
      "self": "http://localhost:8181/dirs/d2",
      "xid": "/dirs/d2",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:02Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

      "filesurl": "http://localhost:8181/dirs/d2/files",
      "filescount": 0
    }
  }
}
`,
	})

	// Epoch bumped
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "1.0-rc1",
  "registryid": "TestHTTPRegGroups",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "name": "hello",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 2,
  "foosurl": "http://localhost:8181/foos",
  "fooscount": 0
}
`})

	reg.Refresh(registry.FOR_READ)
	regEpoch := reg.Get("epoch")
	regTime := reg.Get("modifiedat")

	xCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs",
		Method:  "DELETE",
		Code:    204,
		ResBody: "*",
	})

	reg.Refresh(registry.FOR_READ)
	xCheck(t, regEpoch != reg.Get("epoch"), "regEpoch should be 1")
	xCheck(t, regTime != reg.Get("createdat"), "regEpoch should be 1")

	// Epoch bumped
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "1.0-rc1",
  "registryid": "TestHTTPRegGroups",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "name": "hello",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0,
  "foosurl": "http://localhost:8181/foos",
  "fooscount": 0
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - two grouptypes/no groups",
		URL:        "/",
		Method:     "POST",
		ReqBody:    `{"dirs":{}, "foos":{}}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "dirs": {},
  "foos": {}
}
`,
	})

	// Epoch unchanged
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "1.0-rc1",
  "registryid": "TestHTTPRegGroups",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "name": "hello",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0,
  "foosurl": "http://localhost:8181/foos",
  "fooscount": 0
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - two grouptypes/groups",
		URL:        "/",
		Method:     "POST",
		ReqBody:    `{"dirs":{"d1":{}}, "foos":{"f1":{},"f2":{}}}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "filescount": 0
    }
  },
  "foos": {
    "f1": {
      "fooid": "f1",
      "self": "http://localhost:8181/foos/f1",
      "xid": "/foos/f1",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

      "barsurl": "http://localhost:8181/foos/f1/bars",
      "barscount": 0
    },
    "f2": {
      "fooid": "f2",
      "self": "http://localhost:8181/foos/f2",
      "xid": "/foos/f2",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

      "barsurl": "http://localhost:8181/foos/f2/bars",
      "barscount": 0
    }
  }
}
`,
	})

	// Epoch unchanged
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "1.0-rc1",
  "registryid": "TestHTTPRegGroups",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 6,
  "name": "hello",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1,
  "foosurl": "http://localhost:8181/foos",
  "fooscount": 2
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - err",
		URL:        "/",
		Method:     "POST",
		ReqBody:    `{"dirs":{"d1":{}}, "foos":{"f1":{},"f2":{"foo":"bar"}}}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "Invalid extension(s): foo\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - err - bad type",
		URL:        "/",
		Method:     "POST",
		ReqBody:    `{"name": "foo"}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "POST / only allows Group types to be specified. \"name\" is invalid\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - err - bad group",
		URL:        "/",
		Method:     "POST",
		ReqBody:    `{"name": { "foo":{} } }`,
		Code:       400,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "POST / only allows Group types to be specified. \"name\" is invalid\n",
	})

	// Make sure nothing changed at the registry level
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "1.0-rc1",
  "registryid": "TestHTTPRegGroups",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 6,
  "name": "hello",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1,
  "foosurl": "http://localhost:8181/foos",
  "fooscount": 2
}
`})
}

func TestHTTPResourcesHeaders(t *testing.T) {
	reg := NewRegistry("TestHTTPResourcesHeaders")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	reg.AddGroup("dirs", "dir1")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources - fail",
		URL:        "/dirs/dir1/files",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       405,
		ResHeaders: []string{"Content-Type:text/plain; charset=utf-8"},
		ResBody:    "PUT not allowed on collections\n",
	})

	xHTTP(t, reg, "POST", "/dirs/dir1/files", "{}", 200, "{}\n")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources - w/doc",
		URL:        "/dirs/dir1/files/f1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "My cool doc",
		Code:       201,
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-xid: /dirs/dir1/files/f1",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f1/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f1/versions",
			"xRegistry-versionscount: 1",
			"Location: http://localhost:8181/dirs/dir1/files/f1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"Content-Disposition: f1",
			"Content-Length: 11",
		},
		ResBody: `My cool doc`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - w/doc - new content-type",
		URL:    "/dirs/dir1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"Content-Type: my/format",
		},
		ReqBody: "My cool doc - new",
		Code:    200,
		ResHeaders: []string{
			"Content-Type: my/format",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-xid: /dirs/dir1/files/f1",
			"xRegistry-epoch: 2",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f1/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f1/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"Content-Disposition: f1",
			"Content-Length: 17",
		},
		ResBody: `My cool doc - new`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources - w/doc - no content-type",
		URL:        "/dirs/dir1/files/f1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "My cool doc - new one",
		Code:       200,
		ResHeaders: []string{
			"Content-Type: my/format",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-xid: /dirs/dir1/files/f1",
			"xRegistry-epoch: 3",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f1/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f1/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"Content-Disposition: f1",
			"Content-Length: 21",
		},
		ResBody: `My cool doc - new one`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - w/doc - revert content-type and body",
		URL:    "/dirs/dir1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"Content-Type: null",
		},
		ReqBody: "My cool doc - new x2",
		Code:    200,
		ResHeaders: []string{
			// "Content-Type: text/plain; charset=utf-8",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f1",
			"xRegistry-xid: /dirs/dir1/files/f1",
			"xRegistry-epoch: 4",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f1/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f1/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f1/versions/1",
			"Content-Disposition: f1",
			"Content-Length: 20",
		},
		ResBody: `My cool doc - new x2`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources - w/doc - bad id",
		URL:        "/dirs/dir1/files/f1",
		Method:     "PUT",
		ReqHeaders: []string{"xRegistry-fileid:f2"},
		ReqBody:    "My cool doc",
		Code:       400,
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
		},
		ResBody: `The "fileid" attribute must be set to "f1", not "f2"
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources/res - w/doc + data",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileid: f3",
			"xRegistry-name: my doc",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-labels-l3: null",
		},
		ReqBody:     "My cool doc",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-xid: /dirs/dir1/files/f3",
			"xRegistry-epoch: 1",
			"xRegistry-name: my doc",
			"xRegistry-isdefault: true",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f3/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"xRegistry-versionscount: 1",
			"Location: http://localhost:8181/dirs/dir1/files/f3",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Disposition: f3",
			"Content-Length: 11",
		},
		ResBody: `My cool doc`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT resources - update default - content",
		URL:         "/dirs/dir1/files/f3",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     "My cool doc - v2",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-xid: /dirs/dir1/files/f3",
			"xRegistry-epoch: 2",
			"xRegistry-name: my doc",
			"xRegistry-isdefault: true",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f3/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Disposition: f3",
			"Content-Length: 16",
		},
		ResBody: `My cool doc - v2`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - create - URL",
		URL:    "/dirs/dir1/files/f4",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-name: my doc",
			"xRegistry-fileurl: http://example.com",
		},
		ReqBody:     "",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f4",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f4",
			"xRegistry-xid: /dirs/dir1/files/f4",
			"xRegistry-epoch: 1",
			"xRegistry-name: my doc",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-fileurl: http://example.com",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f4/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f4/versions",
			"xRegistry-versionscount: 1",
			"Location: http://localhost:8181/dirs/dir1/files/f4",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f4/versions/1",
			"Content-Disposition: f4",
		},
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - URL",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileurl: http://example.com",
		},
		ReqBody:     "",
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-xid: /dirs/dir1/files/f3",
			"xRegistry-epoch: 3",
			"xRegistry-name: my doc",
			"xRegistry-isdefault: true",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-fileurl: http://example.com",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f3/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"xRegistry-versionscount: 1",
			"Location: http://example.com",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Disposition: f3",
		},
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - URL + body - error",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileurl: example.com",
		},
		ReqBody:     "My cool doc - v2",
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody:     "'xRegistry-fileurl' isn't allowed if there's a body\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - URL - null",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileurl: null",
		},
		ReqBody:     "",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-xid: /dirs/dir1/files/f3",
			"xRegistry-epoch: 4",
			"xRegistry-name: my doc",
			"xRegistry-isdefault: true",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f3/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Disposition: f3",
		},
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT resources - update default - w/body",
		URL:         "/dirs/dir1/files/f3",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-xid: /dirs/dir1/files/f3",
			"xRegistry-epoch: 5",
			"xRegistry-name: my doc",
			"xRegistry-isdefault: true",
			"xRegistry-description: very cool",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f3/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Disposition: f3",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - w/body - clear 1 prop",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-description: null",
		},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-xid: /dirs/dir1/files/f3",
			"xRegistry-epoch: 6",
			"xRegistry-name: my doc",
			"xRegistry-isdefault: true",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: v1",
			"xRegistry-labels-l2: 5",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f3/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Disposition: f3",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - w/body - edit 2 label",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-labels-l1: l1l1",
			"xRegistry-labels-l4: 4444",
		},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-xid: /dirs/dir1/files/f3",
			"xRegistry-epoch: 7",
			"xRegistry-name: my doc",
			"xRegistry-isdefault: true",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l1: l1l1",
			"xRegistry-labels-l4: 4444",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f3/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Disposition: f3",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - w/body - edit 1 label",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-labels-l3: 3333",
		},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-xid: /dirs/dir1/files/f3",
			"xRegistry-epoch: 8",
			"xRegistry-name: my doc",
			"xRegistry-isdefault: true",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-l3: 3333",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f3/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Disposition: f3",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - w/body - delete labels",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-labels: null",
		},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-xid: /dirs/dir1/files/f3",
			"xRegistry-epoch: 9",
			"xRegistry-name: my doc",
			"xRegistry-isdefault: true",
			"xRegistry-documentation: my doc url",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f3/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Disposition: f3",
		},
		ResBody: "another body",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - update default - w/body - delete+add labels",
		URL:    "/dirs/dir1/files/f3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-labels: null",
			"xRegistry-labels-foo: foo",
			"xRegistry-labels-foo-bar: l-foo-bar",
			"xRegistry-labels-foo_bar: l-foo_bar",
			"xRegistry-labels-foo.bar: l-foo.bar",
		},
		ReqBody:     "another body",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-xid: /dirs/dir1/files/f3",
			"xRegistry-epoch: 10",
			"xRegistry-name: my doc",
			"xRegistry-isdefault: true",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-foo: foo",
			"xRegistry-labels-foo-bar: l-foo-bar",
			"xRegistry-labels-foo_bar: l-foo_bar",
			"xRegistry-labels-foo.bar: l-foo.bar",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f3/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Disposition: f3",
		},
		ResBody: "another body",
	})

	// Checking GET+PUTs

	// 1
	res, err := http.Get("http://localhost:8181/dirs/dir1/files/f3")
	xNoErr(t, err)
	body, err := io.ReadAll(res.Body)
	xNoErr(t, err)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT resources - echo'ing resource GET",
		URL:         "/dirs/dir1/files/f3",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     string(body),
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f3",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/dir1/files/f3",
			"xRegistry-xid: /dirs/dir1/files/f3",
			"xRegistry-epoch: 11",
			"xRegistry-name: my doc",
			"xRegistry-isdefault: true",
			"xRegistry-documentation: my doc url",
			"xRegistry-labels-foo: foo",
			"xRegistry-labels-foo-bar: l-foo-bar",
			"xRegistry-labels-foo_bar: l-foo_bar",
			"xRegistry-labels-foo.bar: l-foo.bar",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/dir1/files/f3/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/dir1/files/f3/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/dir1/files/f3/versions/1",
			"Content-Disposition: f3",
		},
		ResBody: string(body),
	})

	// 2
	res, err = http.Get("http://localhost:8181/dirs/dir1/files/f3$details")
	xNoErr(t, err)
	body, err = io.ReadAll(res.Body)
	xNoErr(t, err)

	resBody := strings.Replace(string(body), `"epoch": 11`, `"epoch": 12`, 1)
	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT resources - echo'ing resource GET$details",
		URL:         "/dirs/dir1/files/f3$details",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     string(body),
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody:     resBody,
	})

	// 3
	res, err = http.Get("http://localhost:8181/")
	xNoErr(t, err)
	body, err = io.ReadAll(res.Body)
	xNoErr(t, err)

	// Change the modifiedat field since it'll change
	re := regexp.MustCompile(`"modifiedat": "[^"]*"`)
	body = re.ReplaceAll(body, []byte(`"modifiedat": "2024-01-01T12:12:12Z"`))

	resBody = strings.Replace(string(body), `"epoch": 1`, `"epoch": 2`, 1)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT resources - echo'ing registry GET",
		URL:         "/",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     string(body),
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody:     resBody,
	})
}

func TestHTTPCases(t *testing.T) {
	reg := NewRegistry("TestHTTPCases")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	d, _ := reg.AddGroup("dirs", "d1")
	d.AddResource("files", "f1", "v1")

	xHTTP(t, reg, "GET", "/Dirs", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/Dirs/D1", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/dirs/D1", "", 404, "Not found\n")

	xHTTP(t, reg, "GET", "/dirs/d1/Files", "", 404, "Unknown Resource type: Files\n")
	xHTTP(t, reg, "GET", "/dirs/D1/Files", "", 404, "Unknown Resource type: Files\n")
	xHTTP(t, reg, "GET", "/Dirs/D1/Files", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/Dirs/d1/Files", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/Dirs/d1/files", "", 404, "Unknown Group type: Dirs\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/F1", "", 404, "\"dirs/d1/files/F1\" not found\n")
	xHTTP(t, reg, "GET", "/dirs/d1/Files/F1", "", 404, "Unknown Resource type: Files\n")
	xHTTP(t, reg, "GET", "/dirs/D1/Files/F1", "", 404, "Unknown Resource type: Files\n")
	xHTTP(t, reg, "GET", "/Dirs/D1/Files/F1", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/Dirs/D1/Files/f1", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/Dirs/D1/files/f1", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/Dirs/d1/Files/f1", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/dirs/D1/files/F1", "", 404, "\"dirs/D1/files/F1\" not found\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/Versions", "", 404, "Expected \"versions\" or \"meta\", got: Versions\n")
	xHTTP(t, reg, "GET", "/dirs/d1/Files/f1/Versions", "", 404, "Unknown Resource type: Files\n")
	xHTTP(t, reg, "GET", "/dirs/D1/Files/f1/Versions", "", 404, "Unknown Resource type: Files\n")
	xHTTP(t, reg, "GET", "/Dirs/D1/Files/f1/Versions", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/Dirs/D1/Files/f1/versions", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/Dirs/D1/files/f1/Versions", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/Dirs/d1/Files/f1/Versions", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/dirs/D1/Files/f1/Versions", "", 404, "Unknown Resource type: Files\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/V1", "", 404, "\"dirs/d1/files/f1/versions/V1\" not found\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/Versions/V1", "", 404, "Expected \"versions\" or \"meta\", got: Versions\n")
	xHTTP(t, reg, "GET", "/dirs/d1/Files/f1/Versions/V1", "", 404, "Unknown Resource type: Files\n")
	xHTTP(t, reg, "GET", "/dirs/D1/Files/f1/Versions/V1", "", 404, "Unknown Resource type: Files\n")
	xHTTP(t, reg, "GET", "/Dirs/D1/Files/f1/Versions/V1", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/Dirs/D1/Files/f1/Versions/v1", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/Dirs/D1/Files/f1/versions/V1", "", 404, "Unknown Group type: Dirs\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/Versions/v1", "", 404, "Expected \"versions\" or \"meta\", got: Versions\n")
	xHTTP(t, reg, "GET", "/dirs/d1/Files/f1/versions/v1", "", 404, "Unknown Resource type: Files\n")
	xHTTP(t, reg, "GET", "/dirs/D1/Files/f1/versions/v1", "", 404, "Unknown Resource type: Files\n")
	xHTTP(t, reg, "GET", "/Dirs/d1/files/f1/versions/v1", "", 404, "Unknown Group type: Dirs\n")

	// Just to make sure we didn't have a typo above
	xCheckHTTP(t, reg, &HTTPTest{
		URL:         "/dirs/d1/files/f1/versions/v1",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f1",
			"xRegistry-versionid: v1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/v1",
			"xRegistry-xid: /dirs/d1/files/f1/versions/v1",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v1",
		},
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:         "/dirs/d1/files/f1",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f1",
			"xRegistry-versionid: v1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v1",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount: 1",
		},
		ResBody: "",
	})

	// Test the ID in the body too (PUT and PATCH)

	// Group
	xHTTP(t, reg, "PUT", "/dirs/d1", `{ "dirid": "d1" }`, 200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 1
}
`)

	xHTTP(t, reg, "PUT", "/dirs/D1", `{ "dirid": "D1" }`, 400, `Attempting to create a Group with a "dirid" of "D1", when one already exists as "d1"
`)
	xHTTP(t, reg, "PUT", "/dirs/d1", `{ "dirid": "D1" }`, 400, `The "dirid" attribute must be set to "d1", not "D1"
`)
	xHTTP(t, reg, "PATCH", "/dirs/d1", `{ "dirid": "D1" }`, 400, `The "dirid" attribute must be set to "d1", not "D1"
`)

	// Resource
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{ "fileid": "f1" }`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/F1$details", `{ "fileid": "F1" }`, 400, `Attempting to create a Resource with a "fileid" of "F1", when one already exists as "f1"
`)
	xHTTP(t, reg, "PUT", "/dirs/D1/files/f1$details", `{ "fileid": "f1" }`, 400, `Attempting to create a Group with a "dirid" of "D1", when one already exists as "d1"
`)
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{ "fileid": "F1" }`, 400, `The "fileid" attribute must be set to "f1", not "F1"
`)
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/F1$details", `{ "fileid": "F1" }`, 400, `Attempting to create a Resource with a "fileid" of "F1", when one already exists as "f1"
`)
	xHTTP(t, reg, "PATCH", "/dirs/D1/files/f1$details", `{ "fileid": "f1" }`, 400, `Attempting to create a Group with a "dirid" of "D1", when one already exists as "d1"
`)
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", `{ "fileid": "F1" }`, 400, `The "fileid" attribute must be set to "f1", not "F1"
`)

	// Version
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", `{ "versionid": "v1" }`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/V1$details", `{ "versionid": "V1" }`, 400, `Attempting to create a Version with a "versionid" of "V1", when one already exists as "v1"
`)
	xHTTP(t, reg, "PUT", "/dirs/d1/files/F1/versions/v1$details", `{ "versionid": "V1" }`, 400, `Attempting to create a Resource with a "fileid" of "F1", when one already exists as "f1"
`)
	xHTTP(t, reg, "PUT", "/dirs/D1/files/f1/versions/v1$details", `{ "versionid": "V1" }`, 400, `Attempting to create a Group with a "dirid" of "D1", when one already exists as "d1"
`)
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", `{ "versionid": "V1" }`, 400, `The "versionid" attribute must be set to "v1", not "V1"
`)
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions/v1$details", `{ "versionid": "V1" }`, 400, `The "versionid" attribute must be set to "v1", not "V1"
`)

	// Test the ID in the body too (POST)

	// Group
	xHTTP(t, reg, "POST", "/dirs", `{"D1":{"dirid":"D1"}}`, 400, `Attempting to create a Group with a "dirid" of "D1", when one already exists as "d1"
`)
	xHTTP(t, reg, "POST", "/dirs", `{"d1":{"dirid":"D1"}}`, 400, `The "dirid" attribute must be set to "d1", not "D1"
`)
	xHTTP(t, reg, "POST", "/dirs", `{"D1":{"dirid":"d1"}}`, 400, `Attempting to create a Group with a "dirid" of "D1", when one already exists as "d1"
`)

	// Resource
	xHTTP(t, reg, "POST", "/dirs/d1/files", `{"F1":{"fileid":"F1"}}`, 400, `Attempting to create a Resource with a "fileid" of "F1", when one already exists as "f1"
`)
	xHTTP(t, reg, "POST", "/dirs/d1/files", `{"f1":{"fileid":"F1"}}`, 400, `The "fileid" attribute must be set to "f1", not "F1"
`)
	xHTTP(t, reg, "POST", "/dirs/d1/files", `{"F1":{"fileid":"f1"}}`, 400, `Attempting to create a Resource with a "fileid" of "F1", when one already exists as "f1"
`)

	// Version
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{"vv":{"versionid":"vv}}`, 400, `path '.vv.versionid': unterminated string
`) // just a typo first
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions$details",
		`{"vv":{"versionid":"vv"}}`, 400,
		`$details isn't allowed on "/dirs/d1/files/f1/versions$details"
`)
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{"V1":{"versionid":"V1"}}`, 400, `Attempting to create a Version with a "versionid" of "V1", when one already exists as "v1"
`)
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{"v1":{"versionid":"V1"}}`, 400, `The "versionid" attribute must be set to "v1", not "V1"
`)
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{"V1":{"versionid":"v1"}}`, 400, `Attempting to create a Version with a "versionid" of "V1", when one already exists as "v1"
`)

}

func TestHTTPResourcesContentHeaders(t *testing.T) {
	reg := NewRegistry("TestHTTPResourcesContentHeaders")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	d, _ := reg.AddGroup("dirs", "d1")

	// ProxyURL
	f, _ := d.AddResource("files", "f1-proxy", "v1")
	f.SetSaveDefault(NewPP().P("file").UI(), "Hello world! v1")

	v, _ := f.AddVersion("v2")
	v.SetSave(NewPP().P("fileurl").UI(), "http://localhost:8282/EMPTY-URL")

	v, _ = f.AddVersion("v3")
	v.SetSave(NewPP().P("fileproxyurl").UI(), "http://localhost:8282/EMPTY-Proxy")

	// URL
	f, _ = d.AddResource("files", "f2-url", "v1")
	f.SetSaveDefault(NewPP().P("file").UI(), "Hello world! v1")

	v, _ = f.AddVersion("v2")
	v.SetSave(NewPP().P("fileproxyurl").UI(), "http://localhost:8282/EMPTY-Proxy")

	v, _ = f.AddVersion("v3")
	v.SetSave(NewPP().P("fileurl").UI(), "http://localhost:8282/EMPTY-URL")

	// Resource
	f, _ = d.AddResource("files", "f3-resource", "v1")
	f.SetSaveDefault(NewPP().P("fileproxyurl").UI(), "http://localhost:8282/EMPTY-Proxy")

	v, _ = f.AddVersion("v2")
	v.SetSave(NewPP().P("fileurl").UI(), "http://localhost:8282/EMPTY-URL")

	v, _ = f.AddVersion("v3")
	v.SetSave(NewPP().P("file").UI(), "Hello world! v3")

	// /dirs/d1/files/f1-proxy/v1 - resource
	//                        /v2 - URL
	//                        /v3 - ProxyURL  <- default
	// /dirs/d1/files/f2-url/v1 - resource
	//                      /v2 - ProxyURL
	//                      /v3 - URL  <- default
	// /dirs/d1/files/f3-resource/v1 - ProxyURL
	//                           /v2 - URL
	//                           /v3 - resource  <- default

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - f1",
		URL:         "/dirs/d1/files/f1-proxy",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f1-proxy",
			"xRegistry-versionid: v3",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy",
			"xRegistry-xid: /dirs/d1/files/f1-proxy",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v2",
			"xRegistry-fileproxyurl: http://localhost:8282/EMPTY-Proxy",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f1-proxy/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1-proxy/versions",
			"xRegistry-versionscount: 3",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1-proxy/versions/v3",
			"Content-Disposition: f1-proxy",
		},
		ResBody: "hello-Proxy\n",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1-proxy",
		Body:    "hello-Proxy\n",
		Headers: nil,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - f1/v3",
		URL:         "/dirs/d1/files/f1-proxy/versions/v3",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f1-proxy",
			"xRegistry-versionid: v3",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy/versions/v3",
			"xRegistry-xid: /dirs/d1/files/f1-proxy/versions/v3",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v2",
			"xRegistry-fileproxyurl: http://localhost:8282/EMPTY-Proxy",
		},
		ResBody: "hello-Proxy\n",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1-proxy/versions/v3",
		Body:    "hello-Proxy\n",
		Headers: nil,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - f1/v2",
		URL:         "/dirs/d1/files/f1-proxy/versions/v2",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f1-proxy",
			"xRegistry-versionid: v2",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy/versions/v2",
			"xRegistry-xid: /dirs/d1/files/f1-proxy/versions/v2",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: false",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v1",
			"xRegistry-fileurl: http://localhost:8282/EMPTY-URL",
			"Location: http://localhost:8282/EMPTY-URL",
		},
		ResBody: "",
	})
	CompareContentMeta(t, reg, &Test{
		Code: 303,
		URL:  "dirs/d1/files/f1-proxy/versions/v2",
		Headers: []string{
			"Location: http://localhost:8282/EMPTY-URL",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - f2",
		URL:         "/dirs/d1/files/f2-url",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f2-url",
			"xRegistry-versionid: v3",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f2-url",
			"xRegistry-xid: /dirs/d1/files/f2-url",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v2",
			"xRegistry-fileurl: http://localhost:8282/EMPTY-URL",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f2-url/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f2-url/versions",
			"xRegistry-versionscount: 3",
			"Location: http://localhost:8282/EMPTY-URL",
		},
		ResBody: "",
	})
	CompareContentMeta(t, reg, &Test{
		Code: 303,
		URL:  "dirs/d1/files/f2-url",
		Headers: []string{
			"Location: http://localhost:8282/EMPTY-URL",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - f3",
		URL:         "/dirs/d1/files/f3-resource",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f3-resource",
			"xRegistry-versionid: v3",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f3-resource",
			"xRegistry-xid: /dirs/d1/files/f3-resource",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v2",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f3-resource/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f3-resource/versions",
			"xRegistry-versionscount: 3",
		},
		ResBody: "Hello world! v3",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f3-resource/versions/v3",
		Headers: []string{},
		Body:    "Hello world! v3",
	})
}

func TestHTTPVersions(t *testing.T) {
	reg := NewRegistry("TestHTTPVersions")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	reg.AddGroup("dirs", "d1")

	// Quick test to make sure body is a Resource and not a collection
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{ "x": {"fileid":"x"}}`,
		400, `Invalid extension(s): x
`)

	// ProxyURL
	// f, _ := d.AddResource("files", "f1-proxy", "v1")
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT file f1-proxy",
		URL:        "/dirs/d1/files/f1-proxy$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "file": "Hello world! v1"
}`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f1-proxy$details",
		},
		ResBody: `{
  "fileid": "f1-proxy",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy$details",
  "xid": "/dirs/d1/files/f1-proxy",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1-proxy/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions",
  "versionscount": 1
}
`,
	})

	// Now inline "file"
	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET file f1-proxy + inline",
		URL:         "/dirs/d1/files/f1-proxy$details?inline=file",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     ``,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "fileid": "f1-proxy",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy$details",
  "xid": "/dirs/d1/files/f1-proxy",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "file": "Hello world! v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1-proxy/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET file f1-proxy/v/1",
		URL:         "/dirs/d1/files/f1-proxy/versions/1",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     ``,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f1-proxy",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy/versions/1",
			"xRegistry-xid: /dirs/d1/files/f1-proxy/versions/1",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1-proxy/versions/1",
			"Content-Disposition: f1-proxy",
			"Content-Length: 15",
			"Content-Type: application/json",
		},
		ResBody: "Hello world! v1",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET file f1-proxy/v/1+inline",
		URL:         "/dirs/d1/files/f1-proxy$details?inline=file",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     "",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "fileid": "f1-proxy",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy$details",
  "xid": "/dirs/d1/files/f1-proxy",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "file": "Hello world! v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1-proxy/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions",
  "versionscount": 1
}
`,
	})

	// add new version via POST to "versions" collection
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST file f1-proxy - create v2",
		URL:        "/dirs/d1/files/f1-proxy/versions",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "v2": {
		    "fileid": "f1-proxy",
		    "versionid": "v2",
            "file": "Hello world! v2"
		  }
		}`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "v2": {
    "fileid": "f1-proxy",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/v2$details",
    "xid": "/dirs/d1/files/f1-proxy/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "1",
    "contenttype": "application/json"
  }
}
`,
	})

	// Error on non-metadata body
	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "POST file f1-proxy - create 2 - no meta",
		URL:         "/dirs/d1/files/f1-proxy/versions",
		Method:      "POST",
		ReqHeaders:  []string{},
		ReqBody:     `this is v3`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `path '': invalid boolean
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "POST file f1-proxy - create 2 - empty",
		URL:         "/dirs/d1/files/f1-proxy/versions",
		Method:      "POST",
		ReqHeaders:  []string{},
		ReqBody:     `{"2":{}}`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "2": {
    "fileid": "f1-proxy",
    "versionid": "2",
    "self": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/2$details",
    "xid": "/dirs/d1/files/f1-proxy/versions/2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:00Z",
    "modifiedat": "2024-01-01T12:00:00Z",
    "ancestor": "v2"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET file f1-proxy - v2 + inline",
		URL:         "/dirs/d1/files/f1-proxy/versions/v2$details?inline=file",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     ``,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "fileid": "f1-proxy",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy/versions/v2$details",
  "xid": "/dirs/d1/files/f1-proxy/versions/v2",
  "epoch": 1,
  "isdefault": false,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "file": "Hello world! v2"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1-proxy - update contents",
		URL:         "/dirs/d1/files/f1-proxy",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `more data`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid:f1-proxy",
			"xRegistry-versionid: 2",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1-proxy",
			"xRegistry-xid: /dirs/d1/files/f1-proxy",
			"xRegistry-epoch:2",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: v2",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f1-proxy/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1-proxy/versions",
			"xRegistry-versionscount:3",
		},
		ResBody: `more data`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET file f1-proxy - check update",
		URL:         "/dirs/d1/files/f1-proxy",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     ``,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid:f1-proxy",
			"xRegistry-versionid: 2",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1-proxy",
			"xRegistry-xid: /dirs/d1/files/f1-proxy",
			"xRegistry-epoch:2",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: v2",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f1-proxy/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1-proxy/versions",
			"xRegistry-versionscount:3",
		},
		ResBody: `more data`,
	})

	// Update default with fileURL
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT file f1-proxy - use fileurl",
		URL:        "/dirs/d1/files/f1-proxy$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "fileid": "f1-proxy",
		  "fileurl": "http://localhost:8282/EMPTY-URL"
		}`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "fileid": "f1-proxy",
  "versionid": "2",
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy$details",
  "xid": "/dirs/d1/files/f1-proxy",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v2",

  "fileurl": "http://localhost:8282/EMPTY-URL",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1-proxy/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions",
  "versionscount": 3
}
`,
	})

	// Update default - delete fileurl, notice no "id" either
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT file f1-proxy - del fileurl",
		URL:        "/dirs/d1/files/f1-proxy$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		}`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "fileid": "f1-proxy",
  "versionid": "2",
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy$details",
  "xid": "/dirs/d1/files/f1-proxy",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1-proxy/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions",
  "versionscount": 3
}
`,
	})

	// Update default - set 'file' and 'fileurl' - error
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT file f1-proxy - dup files",
		URL:        "/dirs/d1/files/f1-proxy$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "file": "hello world",
		  "fileurl": "http://example.com"
		}`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `Only one of file,fileurl,filebase64,fileproxyurl can be present at a time
`,
	})

	// Update default - set 'filebase64' and 'fileurl' - error
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT file f1-proxy - dup files base64",
		URL:        "/dirs/d1/files/f1-proxy$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "filebase64": "aGVsbG8K",
		  "fileurl": "http://example.com"
		}`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `Only one of file,fileurl,filebase64,fileproxyurl can be present at a time
`,
	})

	// Update default - with 'filebase64'
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT file f1-proxy - use base64",
		URL:        "/dirs/d1/files/f1-proxy$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "filebase64": "aGVsbG8K"
		}`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "fileid": "f1-proxy",
  "versionid": "2",
  "self": "http://localhost:8181/dirs/d1/files/f1-proxy$details",
  "xid": "/dirs/d1/files/f1-proxy",
  "epoch": 5,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1-proxy/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1-proxy/versions",
  "versionscount": 3
}
`,
	})

	// Get default
	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET file f1-proxy - use base64",
		URL:         "/dirs/d1/files/f1-proxy",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     "",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f1-proxy",
			"xRegistry-versionid: 2",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1-proxy",
			"xRegistry-xid: /dirs/d1/files/f1-proxy",
			"xRegistry-epoch: 5",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: v2",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f1-proxy/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1-proxy/versions",
			"xRegistry-versionscount: 3",
		},
		ResBody: `hello
`,
	})

	// test the variants of how to store a resource

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT files/f2/versions/v1 - resource",
		URL:         "/dirs/d1/files/f2/versions/v1",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     "Hello world - v1",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f2/versions/v1",
			"Content-Location:http://localhost:8181/dirs/d1/files/f2/versions/v1",
			"Content-Disposition: f2",
			"xRegistry-fileid:f2",
			"xRegistry-versionid:v1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f2/versions/v1",
			"xRegistry-xid: /dirs/d1/files/f2/versions/v1",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v1",
		},
		ResBody: "Hello world - v1",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT files/f2/versions/v2 - fileproxyurl",
		URL:    "/dirs/d1/files/f2/versions/v2",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileproxyurl:http://localhost:8282/EMPTY-Proxy",
		},
		ReqBody:     "",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f2/versions/v2",
			"Content-Location:http://localhost:8181/dirs/d1/files/f2/versions/v2",
			"Content-Disposition: f2",
			"xRegistry-fileid:f2",
			"xRegistry-versionid:v2",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f2/versions/v2",
			"xRegistry-xid: /dirs/d1/files/f2/versions/v2",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v1",
			"xRegistry-fileproxyurl: http://localhost:8282/EMPTY-Proxy",
		},
		ResBody: "hello-Proxy\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT files/f2/versions/v3 - resourceURL",
		URL:    "/dirs/d1/files/f2/versions/v3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-fileurl:http://localhost:8282/EMPTY-URL",
		},
		ReqBody:     "",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f2/versions/v3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f2/versions/v3",
			"Content-Disposition: f2",
			"xRegistry-fileid:f2",
			"xRegistry-versionid:v3",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f2/versions/v3",
			"xRegistry-xid: /dirs/d1/files/f2/versions/v3",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v2",
			"xRegistry-fileurl:http://localhost:8282/EMPTY-URL",
		},
		ResBody: "",
	})

	// testing of "isdefault" processing

	// Set up the following:
	// /dirs/d1/files/ff1-proxy/v1 - resource
	//                        /v2 - URL
	//                        /v3 - ProxyURL  <- default
	// /dirs/d1/files/ff2-url/v1 - resource
	//                      /v2 - ProxyURL
	//                      /v3 - URL  <- default
	// /dirs/d1/files/ff3-resource/v1 - ProxyURL
	//                           /v2 - URL
	//                           /v3 - resource  <- default

	// Now create the ff1-proxy variants
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff1-proxy-v1 Resource",
		URL:    "/dirs/d1/files/ff1-proxy$details",
		Method: "POST",
		ReqBody: `{
		  "versionid": "v1",
		  "file": "In resource ff1-proxy"
		}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "ff1-proxy",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v1$details",
  "xid": "/dirs/d1/files/ff1-proxy/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",
  "contenttype": "application/json"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff1-proxy-v2 URL",
		URL:    "/dirs/d1/files/ff1-proxy$details",
		Method: "POST",
		ReqBody: `{
	      "versionid": "v2",
	      "fileurl": "http://localhost:8282/EMPTY-URL"
		}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "ff1-proxy",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v2$details",
  "xid": "/dirs/d1/files/ff1-proxy/versions/v2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",

  "fileurl": "http://localhost:8282/EMPTY-URL"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff1-proxy-v3 ProxyURL",
		URL:    "/dirs/d1/files/ff1-proxy$details",
		Method: "POST",
		ReqBody: `{
		  "versionid": "v3",
		  "fileproxyurl": "http://localhost:8282/EMPTY-Proxy"
		}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "ff1-proxy",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v3$details",
  "xid": "/dirs/d1/files/ff1-proxy/versions/v3",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v2",

  "fileproxyurl": "http://localhost:8282/EMPTY-Proxy"
}
`,
	})

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff1-proxy",
		Headers: []string{},
		Body:    "hello-Proxy\n",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff1-proxy/versions/v1",
		Headers: []string{},
		Body:    "In resource ff1-proxy",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    303,
		URL:     "dirs/d1/files/ff1-proxy/versions/v2",
		Headers: []string{},
		Body:    "",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff1-proxy/versions/v3",
		Headers: []string{},
		Body:    "hello-Proxy\n",
	})

	// Now create the ff2-url variants
	// ///////////////////////////////
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff2-url-v1 resource",
		URL:    "/dirs/d1/files/ff2-url$details",
		Method: "POST",
		ReqBody: `{
		  "versionid": "v1",
		  "file": "In resource ff2-url"
		}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "ff2-url",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/ff2-url/versions/v1$details",
  "xid": "/dirs/d1/files/ff2-url/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",
  "contenttype": "application/json"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff2-url-v2 ProxyURL",
		URL:    "/dirs/d1/files/ff2-url$details",
		Method: "POST",
		ReqBody: `{
		  "versionid": "v2",
		  "fileproxyurl": "http://localhost:8282/EMPTY-Proxy"
		}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "ff2-url",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/ff2-url/versions/v2$details",
  "xid": "/dirs/d1/files/ff2-url/versions/v2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",

  "fileproxyurl": "http://localhost:8282/EMPTY-Proxy"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff2-url-v2 URL",
		URL:    "/dirs/d1/files/ff2-url$details",
		Method: "POST",
		ReqBody: `{
		  "versionid": "v3",
		  "fileurl": "http://localhost:8282/EMPTY-URL"
		}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "ff2-url",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/ff2-url/versions/v3$details",
  "xid": "/dirs/d1/files/ff2-url/versions/v3",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v2",

  "fileurl": "http://localhost:8282/EMPTY-URL"
}
`,
	})

	CompareContentMeta(t, reg, &Test{
		Code:    303,
		URL:     "dirs/d1/files/ff2-url",
		Headers: []string{},
		Body:    "",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff2-url/versions/v1",
		Headers: []string{},
		Body:    "In resource ff2-url",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff2-url/versions/v2",
		Headers: []string{},
		Body:    "hello-Proxy\n",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    303,
		URL:     "dirs/d1/files/ff2-url/versions/v3",
		Headers: []string{},
		Body:    "",
	})

	// Now create the ff3-resource variants
	// ///////////////////////////////
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff3-resource-v1 ProxyURL",
		URL:    "/dirs/d1/files/ff3-resource$details",
		Method: "POST",
		ReqBody: `{
		  "versionid": "v1",
		  "fileproxyurl": "http://localhost:8282/EMPTY-Proxy"
		}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "ff3-resource",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/ff3-resource/versions/v1$details",
  "xid": "/dirs/d1/files/ff3-resource/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",

  "fileproxyurl": "http://localhost:8282/EMPTY-Proxy"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff3-resource-v2 URL",
		URL:    "/dirs/d1/files/ff3-resource$details",
		Method: "POST",
		ReqBody: `{
		  "versionid": "v2",
		  "fileurl": "http://localhost:8282/EMPTY-URL"
		}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "ff3-resource",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/ff3-resource/versions/v2$details",
  "xid": "/dirs/d1/files/ff3-resource/versions/v2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",

  "fileurl": "http://localhost:8282/EMPTY-URL"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file ff3-resource-v3 resource",
		URL:    "/dirs/d1/files/ff3-resource$details",
		Method: "POST",
		ReqBody: `{
		  "versionid": "v3",
		  "file": "In resource ff3-resource"
		}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "ff3-resource",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/ff3-resource/versions/v3$details",
  "xid": "/dirs/d1/files/ff3-resource/versions/v3",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v2",
  "contenttype": "application/json"
}
`,
	})

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff3-resource",
		Headers: []string{},
		Body:    "In resource ff3-resource",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff3-resource/versions/v1",
		Headers: []string{},
		Body:    "hello-Proxy\n",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    303,
		URL:     "dirs/d1/files/ff3-resource/versions/v2",
		Headers: []string{},
		Body:    "",
	})
	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/ff3-resource/versions/v3",
		Headers: []string{},
		Body:    "In resource ff3-resource",
	})

	// Now do some testing

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - ff1",
		URL:         "/dirs/d1/files/ff1-proxy",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: ff1-proxy",
			"xRegistry-versionid: v3",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff1-proxy",
			"xRegistry-xid: /dirs/d1/files/ff1-proxy",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v2",
			"xRegistry-fileproxyurl: http://localhost:8282/EMPTY-Proxy",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/ff1-proxy/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/ff1-proxy/versions",
			"xRegistry-versionscount: 3",
			"Content-Location: http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v3",
			"Content-Disposition: ff1-proxy",
		},
		ResBody: "hello-Proxy\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - ff1/v3",
		URL:         "/dirs/d1/files/ff1-proxy/versions/v3",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: ff1-proxy",
			"xRegistry-versionid: v3",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v3",
			"xRegistry-xid: /dirs/d1/files/ff1-proxy/versions/v3",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v2",
			"xRegistry-fileproxyurl: http://localhost:8282/EMPTY-Proxy",
		},
		ResBody: "hello-Proxy\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - ff1/v2",
		URL:         "/dirs/d1/files/ff1-proxy/versions/v2",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: ff1-proxy",
			"xRegistry-versionid: v2",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff1-proxy/versions/v2",
			"xRegistry-xid: /dirs/d1/files/ff1-proxy/versions/v2",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: false",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v1",
			"xRegistry-fileurl: http://localhost:8282/EMPTY-URL",
			"Location: http://localhost:8282/EMPTY-URL",
		},
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - ff2",
		URL:         "/dirs/d1/files/ff2-url",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        303,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: ff2-url",
			"xRegistry-versionid: v3",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff2-url",
			"xRegistry-xid: /dirs/d1/files/ff2-url",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v2",
			"xRegistry-fileurl: http://localhost:8282/EMPTY-URL",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/ff2-url/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/ff2-url/versions",
			"xRegistry-versionscount: 3",
			"Location: http://localhost:8282/EMPTY-URL",
		},
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET resource - default - ff3",
		URL:         "/dirs/d1/files/ff3-resource",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: ff3-resource",
			"xRegistry-versionid: v3",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/ff3-resource",
			"xRegistry-xid: /dirs/d1/files/ff3-resource",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v2",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/ff3-resource/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/ff3-resource/versions",
			"xRegistry-versionscount: 3",
		},
		ResBody: "In resource ff3-resource",
	})

	// Test content-type
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT files/f5/versions/v1 - content-type",
		URL:    "/dirs/d1/files/f5/versions/v1",
		Method: "PUT",
		ReqHeaders: []string{
			"Content-Type: my/format",
		},
		ReqBody:     "Hello world - v1",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f5/versions/v1",
			"Content-Length:16",
			"Content-Type:my/format",
			"Content-Location:http://localhost:8181/dirs/d1/files/f5/versions/v1",
			"Content-Disposition:f5",
			"xRegistry-fileid:f5",
			"xRegistry-versionid:v1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f5/versions/v1",
			"xRegistry-xid: /dirs/d1/files/f5/versions/v1",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v1",
		},
		ResBody: "Hello world - v1",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST files/f5 - add version - content-type",
		URL:    "/dirs/d1/files/f5",
		Method: "POST",
		ReqHeaders: []string{
			// Notice no "ID" - so this is also testing "POST Resource w/o id"
			"Content-Type: my/format2",
		},
		ReqBody:     "Hello world - v2",
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Location:http://localhost:8181/dirs/d1/files/f5/versions/1",
			"Content-Length:16",
			"Content-Type:my/format2",
			"Content-Location:http://localhost:8181/dirs/d1/files/f5/versions/1",
			"Content-Disposition:f5",
			"xRegistry-fileid:f5",
			"xRegistry-versionid:1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f5/versions/1",
			"xRegistry-xid: /dirs/d1/files/f5/versions/1",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v1",
		},
		ResBody: "Hello world - v2",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET files/f5$details - content-type",
		URL:         "/dirs/d1/files/f5$details",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     "",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "fileid": "f5",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f5$details",
  "xid": "/dirs/d1/files/f5",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",
  "contenttype": "my/format2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f5/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f5/versions",
  "versionscount": 2
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET files/f5 - content-type",
		URL:         "/dirs/d1/files/f5",
		Method:      "GET",
		ReqHeaders:  []string{},
		ReqBody:     "",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Length:16",
			"Content-Type:my/format2",
			"Content-Location:http://localhost:8181/dirs/d1/files/f5/versions/1",
			"Content-Disposition:f5",
			"xRegistry-fileid:f5",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f5",
			"xRegistry-xid: /dirs/d1/files/f5",
			"xRegistry-epoch:1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f5/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f5/versions",
			"xRegistry-versionscount:2",
		},
		ResBody: "Hello world - v2",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT files/f5/v1$details - revert content-type",
		URL:        "/dirs/d1/files/f5/versions/v1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f5/versions/xxx$details",
  "xid": "/dirs/d1/files/f5/versions/xxx",
  "epoch": 1
}`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "fileid": "f5",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f5/versions/v1$details",
  "xid": "/dirs/d1/files/f5/versions/v1",
  "epoch": 2,
  "isdefault": false,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET files/f5$details - content-type - again",
		URL:         "/dirs/d1/files/f5$details",
		Method:      "GET",
		ReqHeaders:  []string{},
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "fileid": "f5",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f5$details",
  "xid": "/dirs/d1/files/f5",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",
  "contenttype": "my/format2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f5/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f5/versions",
  "versionscount": 2
}
`,
	})

}

func TestHTTPEpochTimesAddRemove(t *testing.T) {
	reg := NewRegistry("TestHTTPEpochTimesAddRemove")
	defer PassDeleteReg(t, reg)
	xNoErr(t, reg.SaveAllAndCommit())

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	xNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)
	regEpoch := reg.GetAsInt("epoch")
	regCreated := reg.GetAsString("createdat")
	regModified := reg.GetAsString("modifiedat")

	xCheck(t, regEpoch == 1, "regEpoch should be 1")
	xCheck(t, !IsNil(regCreated), "regCreated should not be nil")
	xCheck(t, regModified == regCreated, "reg created != modified")
	xCheck(t, regModified != "", "reg modified is ''")
	xCheck(t, regCreated != "", "reg created is ''")

	d1, _ := reg.AddGroup("dirs", "d1")
	xNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)
	d1.Refresh(registry.FOR_WRITE)

	d1Epoch := d1.GetAsInt(NewPP().P("epoch").UI())
	d1Created := d1.GetAsString(NewPP().P("createdat").UI())
	d1Modified := d1.GetAsString(NewPP().P("modifiedat").UI())

	xCheckEqual(t, "", reg.GetAsInt("epoch"), 2)
	xCheckEqual(t, "", reg.GetAsString("createdat"), "--"+regCreated)
	xCheckGreater(t, "", reg.GetAsString("modifiedat"), regModified)

	xCheckEqual(t, "", d1Epoch, 1)
	xCheckEqual(t, "", reg.GetAsString("modifiedat"), "--"+d1Created)
	xCheckEqual(t, "", reg.GetAsString("modifiedat"), "--"+d1Modified)

	regEpoch = reg.GetAsInt("epoch")
	regModified = reg.GetAsString("modifiedat")

	f1, _ := d1.AddResource("files", "f1", "v1")
	f2, _ := d1.AddResource("files", "f2", "v1")
	xNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)
	d1.Refresh(registry.FOR_WRITE)
	f1.Refresh(registry.FOR_WRITE)
	v1, _ := f1.FindVersion("v1", false, registry.FOR_WRITE)
	m1, _ := f1.FindMeta(false, registry.FOR_WRITE)

	xCheckEqual(t, "", reg.GetAsInt("epoch"), 2)
	xCheckEqual(t, "", reg.GetAsString("createdat"), "--"+regCreated)
	xCheckEqual(t, "", reg.GetAsString("modifiedat"), "--"+regModified)

	xCheckEqual(t, "", d1.GetAsInt("epoch"), 2)
	xCheckEqual(t, "", d1.GetAsString("createdat"), "--"+d1Created)
	xCheckGreater(t, "", d1.GetAsString("modifiedat"), d1Modified)

	d1Epoch = d1.GetAsInt("epoch")
	d1Modified = d1.GetAsString("modifiedat")

	xCheckEqual(t, "", m1.GetAsInt("epoch"), 1)
	xCheckEqual(t, "", m1.GetAsString("createdat"), "--"+d1Modified)
	xCheckEqual(t, "", m1.GetAsString("modifiedat"), "--"+d1Modified)

	m1Created := m1.GetAsString("createdat")
	m1Modified := m1.GetAsString("modifiedat")

	xCheckEqual(t, "", v1.GetAsInt("epoch"), 1)
	xCheckEqual(t, "", v1.GetAsString("createdat"), "--"+d1Modified)
	xCheckEqual(t, "", v1.GetAsString("modifiedat"), "--"+d1Modified)

	v1Created := v1.GetAsString("createdat")
	v1Modified := v1.GetAsString("modifiedat")

	v2, _ := f1.AddVersion("v2")
	xNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)
	d1.Refresh(registry.FOR_WRITE)
	f1.Refresh(registry.FOR_WRITE)
	m1.Refresh(registry.FOR_WRITE)
	v1.Refresh(registry.FOR_WRITE)

	xCheckEqual(t, "", reg.GetAsInt("epoch"), 2)
	xCheckEqual(t, "", reg.GetAsString("createdat"), "--"+regCreated)
	xCheckEqual(t, "", reg.GetAsString("modifiedat"), "--"+regModified)

	xCheckEqual(t, "", d1.GetAsInt("epoch"), 2)
	xCheckEqual(t, "", d1.GetAsString("createdat"), "--"+d1Created)
	xCheckEqual(t, "", d1.GetAsString("modifiedat"), "--"+d1Modified)

	xCheckEqual(t, "", m1.GetAsInt("epoch"), 2)
	xCheckEqual(t, "", m1.GetAsString("createdat"), "--"+m1Created)
	xCheckGreater(t, "", m1.GetAsString("modifiedat"), m1Modified)

	m1Modified = m1.GetAsString("modifiedat")

	xCheckEqual(t, "", v1.GetAsInt("epoch"), 1)
	xCheckEqual(t, "", v1.GetAsString("createdat"), "--"+v1Created)
	xCheckEqual(t, "", v1.GetAsString("modifiedat"), "--"+v1Modified)

	xCheckEqual(t, "", v2.GetAsInt("epoch"), 1)
	xCheckEqual(t, "", v2.GetAsString("createdat"), "--"+m1.GetAsString("modifiedat"))
	xCheckEqual(t, "", v2.GetAsString("modifiedat"), "--"+m1.GetAsString("modifiedat"))

	xHTTP(t, reg, "GET", "/?inline", ``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPEpochTimesAddRemove",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "createdat": "YYYY-MM-DDTHH:MM:02Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:03Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v2",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "YYYY-MM-DDTHH:MM:04Z",
          "modifiedat": "YYYY-MM-DDTHH:MM:04Z",
          "ancestor": "v1",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 2,
            "createdat": "YYYY-MM-DDTHH:MM:03Z",
            "modifiedat": "YYYY-MM-DDTHH:MM:04Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v2",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 1,
              "isdefault": false,
              "createdat": "YYYY-MM-DDTHH:MM:03Z",
              "modifiedat": "YYYY-MM-DDTHH:MM:03Z",
              "ancestor": "v1"
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
              "xid": "/dirs/d1/files/f1/versions/v2",
              "epoch": 1,
              "isdefault": true,
              "createdat": "YYYY-MM-DDTHH:MM:04Z",
              "modifiedat": "YYYY-MM-DDTHH:MM:04Z",
              "ancestor": "v1"
            }
          },
          "versionscount": 2
        },
        "f2": {
          "fileid": "f2",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/d1/files/f2$details",
          "xid": "/dirs/d1/files/f2",
          "epoch": 1,
          "isdefault": true,
          "createdat": "YYYY-MM-DDTHH:MM:03Z",
          "modifiedat": "YYYY-MM-DDTHH:MM:03Z",
          "ancestor": "v1",

          "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
          "meta": {
            "fileid": "f2",
            "self": "http://localhost:8181/dirs/d1/files/f2/meta",
            "xid": "/dirs/d1/files/f2/meta",
            "epoch": 1,
            "createdat": "YYYY-MM-DDTHH:MM:03Z",
            "modifiedat": "YYYY-MM-DDTHH:MM:03Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/v1",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
          "versions": {
            "v1": {
              "fileid": "f2",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f2/versions/v1$details",
              "xid": "/dirs/d1/files/f2/versions/v1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "YYYY-MM-DDTHH:MM:03Z",
              "modifiedat": "YYYY-MM-DDTHH:MM:03Z",
              "ancestor": "v1"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 2
    }
  },
  "dirscount": 1
}
`)

	// Now do DELETE up the tree

	v2.DeleteSetNextVersion("")
	xNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)
	d1.Refresh(registry.FOR_WRITE)
	f1.Refresh(registry.FOR_WRITE)
	m1.Refresh(registry.FOR_WRITE)
	v1.Refresh(registry.FOR_WRITE)

	xCheckEqual(t, "", reg.GetAsInt("epoch"), 2)
	xCheckEqual(t, "", reg.GetAsString("createdat"), "--"+regCreated)
	xCheckEqual(t, "", reg.GetAsString("modifiedat"), "--"+regModified)

	xCheckEqual(t, "", d1.GetAsInt("epoch"), 2)
	xCheckEqual(t, "", d1.GetAsString("createdat"), "--"+d1Created)
	xCheckEqual(t, "", d1.GetAsString("modifiedat"), "--"+d1Modified)

	xCheckEqual(t, "", m1.GetAsInt("epoch"), 3)
	xCheckEqual(t, "", m1.GetAsString("createdat"), "--"+m1Created)
	xCheckGreater(t, "", m1.GetAsString("modifiedat"), m1Modified)

	m1Modified = m1.GetAsString("modifiedat")

	xCheckEqual(t, "", v1.GetAsInt("epoch"), 1)
	xCheckEqual(t, "", v1.GetAsString("createdat"), "--"+v1Created)
	xCheckEqual(t, "", v1.GetAsString("modifiedat"), "--"+v1Modified)

	v1.DeleteSetNextVersion("")
	xNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)
	d1.Refresh(registry.FOR_WRITE)

	xCheckEqual(t, "", reg.GetAsInt("epoch"), 2)
	xCheckEqual(t, "", reg.GetAsString("createdat"), "--"+regCreated)
	xCheckEqual(t, "", reg.GetAsString("modifiedat"), "--"+regModified)

	xCheckEqual(t, "", d1.GetAsInt("epoch"), 3)
	xCheckEqual(t, "", d1.GetAsString("createdat"), "--"+d1Created)
	xCheckGreater(t, "", d1.GetAsString("modifiedat"), d1Modified)

	d1Modified = d1.GetAsString("modifiedat")

	f2.Delete()
	reg.Refresh(registry.FOR_WRITE)
	d1.Refresh(registry.FOR_WRITE)

	xCheckEqual(t, "", reg.GetAsInt("epoch"), 2)
	xCheckEqual(t, "", reg.GetAsString("createdat"), "--"+regCreated)
	xCheckEqual(t, "", reg.GetAsString("modifiedat"), "--"+regModified)

	xCheckEqual(t, "", d1.GetAsInt("epoch"), 4)
	xCheckEqual(t, "", d1.GetAsString("createdat"), "--"+d1Created)
	xCheckGreater(t, "", d1.GetAsString("modifiedat"), d1Modified)

	d1.Delete()
	xNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)

	xCheckEqual(t, "", reg.GetAsInt("epoch"), 3)
	xCheckEqual(t, "", reg.GetAsString("createdat"), "--"+regCreated)
	xCheckGreater(t, "", reg.GetAsString("modifiedat"), regModified)

	xHTTP(t, reg, "GET", "/?inline", ``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPEpochTimesAddRemove",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {},
  "dirscount": 0
}
`)

	// Now add everything at once, epoch=1 and times are same

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/",
		Method: "PUT",
		ReqBody: `{
          "dirs": {
            "d1": {
              "files": {
	            "f1": {
                  "versions": {
                    "v1": {},
                    "v2": {}
                  }
                }
              }
            }
          }
        }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPEpochTimesAddRemove",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`,
	})

	xHTTP(t, reg, "GET", "/?inline", ``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPEpochTimesAddRemove",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:02Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v2",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "YYYY-MM-DDTHH:MM:02Z",
          "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
          "ancestor": "v1",

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

            "defaultversionid": "v2",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 1,
              "isdefault": false,
              "createdat": "YYYY-MM-DDTHH:MM:02Z",
              "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
              "ancestor": "v1"
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
              "xid": "/dirs/d1/files/f1/versions/v2",
              "epoch": 1,
              "isdefault": true,
              "createdat": "YYYY-MM-DDTHH:MM:02Z",
              "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
              "ancestor": "v1"
            }
          },
          "versionscount": 2
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)
}

func TestHTTPEnum(t *testing.T) {
	reg := NewRegistry("TestHTTPEnum")
	defer PassDeleteReg(t, reg)

	attr, _ := reg.Model.AddAttribute(&registry.Attribute{
		Name:   "myint",
		Type:   INTEGER,
		Enum:   []any{1, 2, 3},
		Strict: PtrBool(true),
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - baseline",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
}`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPEnum",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - int valid",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
  "myint": 2
}`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPEnum",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myint": 2
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - int invalid",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
  "myint": 4
}`,
		Code: 400,
		ResBody: "Attribute \"myint\"(4) must be " +
			"one of the enum values: 1, 2, 3\n",
	})

	attr = reg.Model.Attributes["myint"]
	attr.SetStrict(false)
	reg.Model.VerifyAndSave()

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - int valid - no-strict",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
  "myint": 4
}`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPEnum",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myint": 4
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - int valid - no-strict - valid enum",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
  "myint": 1
}`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPEnum",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myint": 1
}
`,
	})

	// TODO test other enum types and test in Groups and Resources
}

func TestHTTPCompatility(t *testing.T) {
	reg := NewRegistry("TestHTTPCompatibility")
	defer PassDeleteReg(t, reg)

	_, _, err := reg.Model.CreateModels("dirs", "dir", "files", "file")
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{"compatibility":"none"}`,
		201, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/meta", `{"compatibility":null}`,
		200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/meta",
		`{"compatibility":"backward"}`, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 3,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "backward",
  "compatibilityauthority": "external",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/meta",
		`{"compatibility":"mine"}`, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 4,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "mine",
  "compatibilityauthority": "external",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
  "defaultversionsticky": false
}
`)

}

func TestHTTPIfValue(t *testing.T) {
	reg := NewRegistry("TestHTTPIfValues")
	defer PassDeleteReg(t, reg)

	_, err := reg.Model.AddAttribute(&registry.Attribute{
		Name: "myint",
		Type: INTEGER,
		IfValues: registry.IfValues{
			"10": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"mystr": &registry.Attribute{
						Name: "mystr",
						Type: STRING,
					},
					"myobj": &registry.Attribute{
						Name: "myobj",
						Type: OBJECT,
						Attributes: registry.Attributes{
							"subint": &registry.Attribute{
								Name: "subint",
								Type: INTEGER,
							},
							"subobj": &registry.Attribute{
								Name: "subobj",
								Type: OBJECT,
								Attributes: registry.Attributes{
									"subsubint": &registry.Attribute{
										Name: "subsubint",
										Type: INTEGER,
									},
									"*": &registry.Attribute{
										Name: "*",
										Type: ANY,
									},
								},
							},
						},
					},
				},
			},
			"20": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"mystr": &registry.Attribute{
						Name:     "mystr",
						Type:     STRING,
						Required: true,
					},
					"*": &registry.Attribute{
						Name: "*",
						Type: ANY,
					},
				},
			},
		},
	})
	xCheckErr(t, err, "")
	xNoErr(t, reg.SaveModel())

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "myobj",
		Type: OBJECT,
	})
	// Test empty obj and name conflict with IfValue above
	xCheckErr(t, err,
		`Duplicate attribute name (myobj) at: model.myint.ifvalues.10`)
	reg.LoadModel()

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "myobj2",
		Type: OBJECT,
		Attributes: registry.Attributes{
			"subint1": &registry.Attribute{
				Name: "subint1",
				Type: INTEGER,
				IfValues: registry.IfValues{
					"666": &registry.IfValue{
						SiblingAttributes: registry.Attributes{
							"reqint": &registry.Attribute{
								Name: "reqint",
								Type: INTEGER,
							},
						},
					},
				},
			},
		},
	})
	xCheckErr(t, err, "")
	xNoErr(t, reg.SaveModel())

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "badone",
		Type: INTEGER,
		IfValues: registry.IfValues{
			"": &registry.IfValue{},
		},
	})
	xCheckErr(t, err, "\"model\" has an empty ifvalues key")

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "badone",
		Type: INTEGER,
		IfValues: registry.IfValues{
			"^6": &registry.IfValue{},
		},
	})
	xCheckErr(t, err, "\"model\" has an ifvalues key that starts with \"^\"")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - 1",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPIfValues",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myint": 10
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - verify ext isn't allowed",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10,
	     "myext": 5.5
	   }`,
		Code:    400,
		ResBody: "Invalid extension(s): myext\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - required mystr",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 20
	   }`,
		Code:    400,
		ResBody: "Required property \"mystr\" is missing\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - required mystr, allow ext",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 20,
	     "mystr": "hello",
	     "myext": 5.5
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPIfValues",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myext": 5.5,
  "myint": 20,
  "mystr": "hello"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - myext isn't allow any more",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10,
	     "mystr": "hello",
	     "myext": 5.5
	   }`,
		Code:    400,
		ResBody: "Invalid extension(s): myext\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - 3 levels - valid",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10,
	     "mystr": "hello",
		 "myobj": {
		   "subint": 123,
		   "subobj": {
		     "subsubint": 432
		   }
		 }
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPIfValues",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myint": 10,
  "myobj": {
    "subint": 123,
    "subobj": {
      "subsubint": 432
    }
  },
  "mystr": "hello"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - 3 levels - valid, unknown 3 level",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10,
	     "mystr": "hello",
		 "myobj": {
		   "subint": 123,
		   "subobj": {
		     "subsubint": 432,
			 "okext": "hello"
		   }
		 }
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPIfValues",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myint": 10,
  "myobj": {
    "subint": 123,
    "subobj": {
      "okext": "hello",
      "subsubint": 432
    }
  },
  "mystr": "hello"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - down a level - valid",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10,
	     "mystr": "hello",
		 "myobj2": {
		   "subint1": 666,
		   "reqint": 777
		 }
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPIfValues",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 6,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myint": 10,
  "myobj2": {
    "reqint": 777,
    "subint1": 666
  },
  "mystr": "hello"
}
`,
	})

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "myint5",
		Type: INTEGER,
		IfValues: registry.IfValues{
			"1": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"myint6": &registry.Attribute{
						Name: "myint6",
						Type: INTEGER,
						IfValues: registry.IfValues{
							"2": &registry.IfValue{
								SiblingAttributes: registry.Attributes{
									"myint7": {
										Name:     "myint7",
										Type:     INTEGER,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	xCheckErr(t, err, "")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - nested ifValues - 1",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint5": 1
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPIfValues",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 7,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myint5": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - nested ifValues - 2",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint5": 1,
	     "myint6": 1
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPIfValues",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 8,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myint5": 1,
  "myint6": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - nested ifValues - 3",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint5": 1,
	     "myint6": 2
	   }`,
		Code: 400,
		ResBody: `Required property "myint7" is missing
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - nested ifValues - 4",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint5": 1,
	     "myint6": 1,
		 "myint7": 5
	   }`,
		Code: 400,
		ResBody: `Invalid extension(s): myint7
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - nested ifValues - 5",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint5": 1,
	     "myint6": 2,
		 "myint7": 5
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPIfValues",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 9,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "myint5": 1,
  "myint6": 2,
  "myint7": 5
}
`,
	})

}

func TestHTTPResources(t *testing.T) {
	reg := NewRegistry("TestHTTPResources")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)
	rm, err := gm.AddResourceModelSimple("files", "file")
	xNoErr(t, err)

	xNoErr(t, reg.SaveModel())

	/*
		_, err := rm.AddAttribute(&registry.Attribute{
			Name: "files",
			Type: INTEGER,
		})
		xCheckErr(t, err, "Attribute name is reserved: files")
	*/

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "file",
		Type: INTEGER,
	})
	xCheckErr(t, err, "Attribute name is reserved: file")

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "filebase64",
		Type: INTEGER,
	})
	xCheckErr(t, err, "Attribute name is reserved: filebase64")

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "fileproxyurl",
		Type: INTEGER,
	})
	xCheckErr(t, err, "Attribute name is reserved: fileproxyurl")

	reg.LoadModel() // shouldn't be need but just in case

	rm = rm.Refresh()
	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "mystring",
		Type: STRING,
		IfValues: registry.IfValues{
			"foo": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"file": &registry.Attribute{
						Name:     "file",
						Type:     INTEGER,
						IfValues: registry.IfValues{},
					},
				},
			},
		},
	})
	xCheckErr(t, err, "Duplicate attribute name (file) at: resources.files.mystring.ifvalues.foo")
	reg.LoadModel()

	rm = rm.Refresh()
	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "mystring",
		Type: STRING,
		IfValues: registry.IfValues{
			"foo": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"xxx": &registry.Attribute{
						Name: "xxx",
						Type: INTEGER,
						IfValues: registry.IfValues{
							"5": &registry.IfValue{
								SiblingAttributes: registry.Attributes{
									"xxx": &registry.Attribute{
										Name: "xxx",
										Type: STRING,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	xCheckErr(t, err, "Duplicate attribute name (xxx) at: resources.files.mystring.ifvalues.foo.xxx.ifvalues.5")

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "mystring",
		Type: STRING,
		IfValues: registry.IfValues{
			"foo": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"xxx": &registry.Attribute{
						Name: "xxx",
						Type: INTEGER,
						IfValues: registry.IfValues{
							"5": &registry.IfValue{
								SiblingAttributes: registry.Attributes{
									"file": &registry.Attribute{
										Name: "file",
										Type: STRING,
									},
								},
							},
						},
					},
				},
			},
		},
	})
	xCheckErr(t, err, "Duplicate attribute name (file) at: resources.files.mystring.ifvalues.foo.xxx.ifvalues.5")
	reg.LoadModel()

	// "file" is ok this time because HasDocument=false
	rm = rm.Refresh()
	rm.SetHasDocument(false)
	xNoErr(t, reg.Model.VerifyAndSave())
	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "mystring",
		Type: STRING,
		IfValues: registry.IfValues{
			"foo": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"file": &registry.Attribute{
						Name: "file",
						Type: STRING,
					},
					"object": &registry.Attribute{
						Name: "object",
						Type: OBJECT,
						Attributes: registry.Attributes{
							"objstr": &registry.Attribute{
								Name: "objstr",
								Type: STRING,
								IfValues: registry.IfValues{
									"objval": {
										SiblingAttributes: registry.Attributes{
											"objint": &registry.Attribute{
												Name: "objint",
												Type: INTEGER,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/vx", `{"versionid":"x"}`,
		400, `The "versionid" attribute must be set to "vx", not "x"
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", "{}", 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "mystring": "hello"
}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1",
  "mystring": "hello"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "file": "fff",
  "mystring": "hello",
  "object": {}
}`, 400, `Invalid extension(s): file,object
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "filebase64": "fff",
  "mystring": "hello",
  "object": {}
}`, 400, `Invalid extension(s): filebase64,object
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "fileurl": "fff",
  "mystring": "hello",
  "object": {}
}`, 400, `Invalid extension(s): fileurl,object
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "fileproxyurl": "fff",
  "mystring": "hello",
  "object": {}
}`, 400, `Invalid extension(s): fileproxyurl,object
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "file": "fff",
  "fileurl": "fff",
  "filebase64": "fff",
  "fileproxyurl": "fff",
  "mystring": "hello",
  "object": {}
}`, 400, `Invalid extension(s): file,filebase64,fileproxyurl,fileurl,object
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "file": "fff",
  "mystring": "foo",
  "object": {}
}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1",
  "file": "fff",
  "mystring": "foo",
  "object": {}
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "file": "fff",
  "mystring": "foo",
  "object": {
    "objstr": "ooo",
    "objint": 5
  }
}`, 400, `Invalid extension(s) in "object": objint
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "file": "fff",
  "mystring": "foo",
  "object": {
    "objstr": "objval",
    "objint": 5
  }
}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1",
  "file": "fff",
  "mystring": "foo",
  "object": {
    "objint": 5,
    "objstr": "objval"
  }
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "mystring": null
}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 5,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "mystring": null
}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 6,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1"
}
`)

}

func TestHTTPNonStrings(t *testing.T) {
	reg := NewRegistry("TestHTTPNonStrings")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true /* L */, true, true)

	// rm.AddAttr("myint", INTEGER)
	attr, _ := rm.AddAttr("myint", INTEGER)
	attr.IfValues = registry.IfValues{
		"-5": &registry.IfValue{
			SiblingAttributes: registry.Attributes{
				"ifext": {
					Name: "ifext",
					Type: INTEGER,
				},
			},
		},
	}

	rm.AddAttr("mydec", DECIMAL)
	rm.AddAttr("mybool", BOOLEAN)
	rm.AddAttr("myuint", UINTEGER)
	rm.AddAttr("mystr", STRING)
	rm.AddAttr("*", BOOLEAN)
	rm.AddAttrMap("mymapint", registry.NewItemType(INTEGER))
	rm.AddAttrMap("mymapdec", registry.NewItemType(DECIMAL))
	rm.AddAttrMap("mymapbool", registry.NewItemType(BOOLEAN))
	rm.AddAttrMap("mymapuint", registry.NewItemType(UINTEGER))

	reg.AddGroup("dirs", "d1")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-myint: -5",
			"xRegistry-mydec: 5.4",
			"xRegistry-mybool: true",
			"xRegistry-myuint: 5",
			"xRegistry-mymapint-k1: -6",
			"xRegistry-mymapdec-k2: -6.5",
			"xRegistry-mymapbool-k3: false",
			"xRegistry-mymapuint-k4: 6",
			"xRegistry-ext: true",
			"xRegistry-ifext: 666",
		},
		ReqBody: `hello`,
		Code:    201,
		ResBody: `hello`,
		ResHeaders: []string{
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch: 1",
			"xRegistry-myint: -5",
			"xRegistry-mydec: 5.4",
			"xRegistry-mybool: true",
			"xRegistry-myuint: 5",
			"xRegistry-mymapint-k1: -6",
			"xRegistry-mymapdec-k2: -6.5",
			"xRegistry-mymapbool-k3: false",
			"xRegistry-mymapuint-k4: 6",
			"xRegistry-ext: true",
			"xRegistry-ifext: 666",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount: 1",
			"Content-Type:text/plain; charset=utf-8",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Disposition:f1",
			"Content-Length:5",
			"Location:http://localhost:8181/dirs/d1/files/f1",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "GET file f1",
		URL:         "/dirs/d1/files/f1$details",
		Method:      "GET",
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "ext": true,
  "ifext": 666,
  "mybool": true,
  "mydec": 5.4,
  "myint": -5,
  "mymapbool": {
    "k3": false
  },
  "mymapdec": {
    "k2": -6.5
  },
  "mymapint": {
    "k1": -6
  },
  "mymapuint": {
    "k4": 6
  },
  "myuint": 5,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-mystr-foo: hello",
		},
		ReqBody: `hello`,
		Code:    400,
		ResBody: `Attribute "mystr" must be a string
`,
		ResHeaders: []string{},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-mystr-foo-bar: hello",
		},
		ReqBody: `hello`,
		Code:    400,
		ResBody: `Attribute "mystr" must be a string
`,
		ResHeaders: []string{},
	})
}

func TestHTTPDefault(t *testing.T) {
	reg := NewRegistry("TestHTTPDefault")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true /* L */, true, true)

	reg.AddGroup("dirs", "d1")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1 - isdefault = true",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-isdefault: true",
		},
		ReqBody:     `hello`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Disposition:f1",
			"Location: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount: 1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "POST file f1 - no isdefault",
		URL:         "/dirs/d1/files/f1",
		Method:      "POST",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"Content-Disposition:f1",
			"Location: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 2",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"xRegistry-xid: /dirs/d1/files/f1/versions/2",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/1 - setdefaultversionid = 1",
		URL:         "/dirs/d1/files/f1/versions/1?setdefaultversionid=1",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Disposition:f1",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-xid: /dirs/d1/files/f1/versions/1",
			"xRegistry-epoch: 2",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/1 - setdefaultversionid = null, switches default",
		URL:         "/dirs/d1/files/f1/versions/1?setdefaultversionid=null",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Disposition:f1",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-xid: /dirs/d1/files/f1/versions/1",
			"xRegistry-epoch: 3",
			"xRegistry-isdefault: false",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
		},
		ResBody: `hello`,
	})

	rm.SetSetDefaultSticky(false)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/2 - setdefault=2 - diff server",
		URL:         "/dirs/d1/files/f1/versions/2?setdefaultversionid=2",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `Resource "files" doesn't allow setting of "defaultversionid"
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/1 - setdefault=1 - match server",
		URL:         "/dirs/d1/files/f1/versions/1?setdefaultversionid=1",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `Resource "files" doesn't allow setting of "defaultversionid"
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/1 - no setdefault",
		URL:         "/dirs/d1/files/f1/versions/1",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Disposition:f1",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-xid: /dirs/d1/files/f1/versions/1",
			"xRegistry-epoch: 4",
			"xRegistry-isdefault: false",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/2 - no setdefault",
		URL:         "/dirs/d1/files/f1/versions/2",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"Content-Disposition:f1",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 2",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"xRegistry-xid: /dirs/d1/files/f1/versions/2",
			"xRegistry-epoch: 2",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
		},
		ResBody: `hello`,
	})

	// Now test ?setdefaultversionid stuff
	///////////////////////////////

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?setdefault= not allowed",
		URL:     "/dirs/d1/files/f1?setdefaultversionid",
		Method:  "POST",
		ReqBody: "{}",
		Code:    400,
		ResBody: `Resource "files" doesn't allow setting of "defaultversionid"` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1$details?setdefault= not allowed",
		URL:     "/dirs/d1/files/f1$details?setdefaultversionid",
		Method:  "POST",
		ReqBody: "{}",
		Code:    400,
		ResBody: `Resource "files" doesn't allow setting of "defaultversionid"` + "\n",
	})

	// Enable client-side setting
	rm.SetSetDefaultSticky(true)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?setdefault - empty",
		URL:     "/dirs/d1/files/f1?setdefaultversionid",
		Method:  "POST",
		Code:    400,
		ResBody: `"setdefaultversionid" must not be empty` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?setdefault= - empty",
		URL:     "/dirs/d1/files/f1?setdefaultversionid=",
		Method:  "POST",
		Code:    400,
		ResBody: `"setdefaultversionid" must not be empty` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1$details?setdefault - empty",
		URL:     "/dirs/d1/files/f1$details?setdefaultversionid",
		Method:  "POST",
		ReqBody: "{}",
		Code:    400,
		ResBody: `"setdefaultversionid" must not be empty` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1$details?setdefault= - empty",
		URL:     "/dirs/d1/files/f1$details?setdefaultversionid=",
		Method:  "POST",
		ReqBody: "{}",
		Code:    400,
		ResBody: `"setdefaultversionid" must not be empty` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file f1?setdefault=1 - no change",
		URL:    "/dirs/d1/files/f1?setdefaultversionid=1",
		Method: "POST",
		ReqHeaders: []string{
			`xRegistry-versionid: newone`,
		},
		ReqBody:     `pick me`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: newone",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/newone",
			"xRegistry-xid: /dirs/d1/files/f1/versions/newone",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: false",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 2",
			"Content-Length: 7",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/newone",
			"Content-Disposition:f1",
			"Location: http://localhost:8181/dirs/d1/files/f1/versions/newone",
		},
		ResBody: `pick me`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file f1?setdefault=2",
		URL:    "/dirs/d1/files/f1?setdefaultversionid=2",
		Method: "POST",
		ReqHeaders: []string{
			`xRegistry-versionid: bogus`,
		},
		ReqBody:     `some text`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: text/plain; charset=utf-8",
			"xRegistry-fileid: f1",
			"xRegistry-versionid: bogus",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/bogus",
			"xRegistry-xid: /dirs/d1/files/f1/versions/bogus",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: false",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: newone",
			"Content-Length: 9",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/bogus",
			"Content-Disposition:f1",
		},
		ResBody: `some text`,
	})

	// Make sure defaultversionid was processed
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 6,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/2",
  "defaultversionsticky": true
}
`)

	// errors
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad group type",
		URL:        "/badgroup/d1/files/f1$details?setdefaultversionid=3",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-versionid: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    "Unknown Group type: badgroup\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad header",
		URL:        "/dirs/d1/files/f1$details?setdefaultversionid=3",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-versionid: bogus`},
		ReqBody:    "{}",
		Code:       400,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Including "xRegistry" headers when "$details" is used is invalid` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad group",
		URL:        "/dirs/dx/files/f11?setdefaultversionid=6",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-versionid: bogus`},
		Code:       400,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Version "6" not found` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad resource type",
		URL:        "/dirs/d1/badfiles/f1$details?setdefaultversionid=3",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-versionid: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Unknown Resource type: badfiles` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad header",
		URL:        "/dirs/d1/files/f1$details?setdefaultversionid=3",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-versionid: bogus`},
		ReqBody:    "{}",
		Code:       400,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Including "xRegistry" headers when "$details" is used is invalid` + "\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad version",
		URL:        "/dirs/d1/files/f1$details?setdefaultversionid=6",
		Method:     "POST",
		ReqBody:    "{}",
		Code:       400,
		ResHeaders: []string{"Content-Type: text/plain; charset=utf-8"},
		ResBody:    `Version "6" not found` + "\n",
	})

}

func TestHTTPDelete(t *testing.T) {
	reg := NewRegistry("TestHTTPDelete")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	reg.AddGroup("dirs", "d1")
	reg.AddGroup("dirs", "d2")
	reg.AddGroup("dirs", "d3")
	reg.AddGroup("dirs", "d4")
	reg.AddGroup("dirs", "d5")

	// DELETE /GROUPs
	xHTTP(t, reg, "DELETE", "/", "", 405, "Can't delete an entire registry\n")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d2",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d2":{}}`,
		Code:    204,
		ResBody: ``,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d2",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{}`, // should be a no-op, not delete everything
		Code:    204,
		ResBody: ``,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs - 1",
		URL:    "/dirs",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 0
  },
  "d3": {
    "dirid": "d3",
    "self": "http://localhost:8181/dirs/d3",
    "xid": "/dirs/d3",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d3/files",
    "filescount": 0
  },
  "d4": {
    "dirid": "d4",
    "self": "http://localhost:8181/dirs/d4",
    "xid": "/dirs/d4",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d4/files",
    "filescount": 0
  },
  "d5": {
    "dirid": "d5",
    "self": "http://localhost:8181/dirs/d5",
    "xid": "/dirs/d5",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d5/files",
    "filescount": 0
  }
}
`,
	})

	xHTTP(t, reg, "DELETE", "/dirs/d3?epoch=2x", "", 400,
		"Epoch value \"2x\" must be an UINTEGER\n")
	xHTTP(t, reg, "DELETE", "/dirs/d3?epoch=2", "", 400,
		"Epoch value for \"d3\" must be 1 not 2\n")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d3 err",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d3": {"epoch":2}}`,
		Code:    400,
		ResBody: `Epoch value for "d3" must be 1 not 2
`,
	})

	// TODO add a delete of 2 with bad epoch in 2nd one and verify that
	// the first one isn't deleted due to the transaction rollback

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d3",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d3":{"dirid": "xx", "epoch":1}}`,
		Code:    400,
		ResBody: `"dirid" value for "d3" must be "d3" not "xx"
`,
	})

	// Make sure we ignore random attributes too
	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d3",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d3":{"dirid": "d3", "epoch":1, "foo": "bar"}}`,
		Code:    204,
		ResBody: ``,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d3 - already gone",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d3":{"epoch":1}}`,
		Code:    204,
		ResBody: ``,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d4",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d4":{"epoch":"1x"}}`,
		Code:    400,
		ResBody: `Epoch value for "d4" must be a uinteger
`,
	})
	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - dx",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"dx":{"epoch":1}}`,
		Code:    204,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d4",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d4":{"epoch":1}}`,
		Code:    204,
		ResBody: ``,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs - 2",
		URL:    "/dirs",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 0
  },
  "d5": {
    "dirid": "d5",
    "self": "http://localhost:8181/dirs/d5",
    "xid": "/dirs/d5",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d5/files",
    "filescount": 0
  }
}
`,
	})

	xHTTP(t, reg, "DELETE", "/dirs/d5?epoch=1", "", 204, "")
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs - 2",
		URL:    "/dirs",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 0
  }
}
`,
	})

	xHTTP(t, reg, "DELETE", "/dirs", "", 204, "")
	xHTTP(t, reg, "DELETE", "/dirs", "", 204, "")
	xHTTP(t, reg, "DELETE", "/dirsx", "", 404, "Unknown Group type: dirsx\n")
	xHTTP(t, reg, "GET", "/dirs", "", 200, "{}\n")

	// Reset
	reg.AddGroup("dirs", "d1")
	reg.AddGroup("dirs", "d2")
	reg.AddGroup("dirs", "d3")
	reg.AddGroup("dirs", "d4")

	// DELETE /GROUPs/GID
	xHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs - 4",
		URL:    "/dirs",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  },
  "d3": {
    "dirid": "d3",
    "self": "http://localhost:8181/dirs/d3",
    "xid": "/dirs/d3",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d3/files",
    "filescount": 0
  },
  "d4": {
    "dirid": "d4",
    "self": "http://localhost:8181/dirs/d4",
    "xid": "/dirs/d4",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d4/files",
    "filescount": 0
  }
}
`,
	})

	xHTTP(t, reg, "DELETE", "/dirs/d3", "", 204, ``)
	xHTTP(t, reg, "DELETE", "/dirs/dx", "", 404, `Group "dx" not found`+"\n")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs - 5",
		URL:    "/dirs",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "d2": {
    "dirid": "d2",
    "self": "http://localhost:8181/dirs/d2",
    "xid": "/dirs/d2",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d2/files",
    "filescount": 0
  },
  "d4": {
    "dirid": "d4",
    "self": "http://localhost:8181/dirs/d4",
    "xid": "/dirs/d4",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d4/files",
    "filescount": 0
  }
}
`,
	})

	xHTTP(t, reg, "DELETE", "/dirs", "", 204, "")
	xHTTP(t, reg, "GET", "/dirs", "", 200, "{}\n")

	// Reset
	d1, _ := reg.AddGroup("dirs", "d1")
	d1.AddResource("files", "f1", "v1.1")
	d1.AddResource("files", "f2", "v2.1")
	d1.AddResource("files", "f3", "v3.1")
	d1.AddResource("files", "f4", "v4.1")
	d1.AddResource("files", "f5", "v5.1")
	d1.AddResource("files", "f6", "v6.1")

	// DELETE Resources
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs/d1 - 7",
		URL:    "/dirs/d1",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 6
}
`,
	})

	// DELETE /dirs/d1/files/f1
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1", "", 204, "")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/fx", "", 404,
		"Resource \"fx\" not found\n")

	// DELETE /dirs/d1/files/f1?epoch=...
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f3?epoch=2x", "", 400,
		"Epoch value \"2x\" must be an UINTEGER\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f3?epoch=2", "", 400,
		"Epoch value for \"f3\" must be 1 not 2\n")

	// Bump epoch of f3
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f3/meta", "{}", 200, `{
  "fileid": "f3",
  "self": "http://localhost:8181/dirs/d1/files/f3/meta",
  "xid": "/dirs/d1/files/f3/meta",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v3.1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f3/versions/v3.1",
  "defaultversionsticky": false
}
`)

	// Bump epoch of f2
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f2/meta", "{}", 200, `{
  "fileid": "f2",
  "self": "http://localhost:8181/dirs/d1/files/f2/meta",
  "xid": "/dirs/d1/files/f2/meta",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v2.1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/v2.1",
  "defaultversionsticky": false
}
`)

	/*
		"f2": { "meta": { "epoch": 2}, "versions": { "v2.1": { "epoch": 1,
		"f3": { "meta": { "epoch": 2}, "versions": { "v3.1": { "epoch": 1,
		"f4": { "meta": { "epoch": 1}, "versions": { "v4.1": { "epoch": 1,
		"f5": { "meta": { "epoch": 1}, "versions": { "v5.1": { "epoch": 1,
		"f6": { "meta": { "epoch": 1}, "versions": { "v6.1": { "epoch": 1,
	*/

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f3?epoch=2", "", 204, "")

	// DELETE - testing ids in body
	_, err := d1.AddResource("files", "f3", "v1")
	xNoErr(t, err)
	xHTTP(t, reg, "DELETE", "/dirs/d1/files", `{"f3":{"fileid":"fx"}}`,
		400, `"fileid" value for "f3" must be "f3" not "fx"
`)
	xHTTP(t, reg, "DELETE", "/dirs/d1/files", `{"f3":{"meta":{"fileid":"fx"}}}`,
		400, `"fileid" value for "f3" must be "f3" not "fx"
`)
	xHTTP(t, reg, "DELETE", "/dirs/d1/files", `{"f3":{"epoch":"2"}}`,
		400, `"epoch" should be under a "meta" map
`)
	xHTTP(t, reg, "DELETE", "/dirs/d1/files", `{"f3":{"fileid":"f3"}}`,
		204, ``)

	// DELETE /dirs/d1/files/f3 - bad epoch in body
	xHTTP(t, reg, "DELETE", "/dirs/d1/files",
		`{"f2":{"meta":{"epoch":"1x"}}}`, 400,
		"Epoch value for \"f2\" must be a uinteger\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files",
		`{"f2":{"meta":{"epoch":4}}}`, 400,
		"Epoch value for \"f2\" must be 2 not 4\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files",
		`{"f2":{"epoch":99,"meta":{"epoch":2}}}`, 204, "") // ignore top 'epoch'
	xHTTP(t, reg, "DELETE", "/dirs/d1/files",
		`{"fx":{"meta":{"epoch":1}}}`, 204, "")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files",
		`{"f2":{},"f4":{"meta":{"epoch":3}}}`,
		400, "Epoch value for \"f4\" must be 1 not 3\n")
	// Make sure we ignore random attributes too
	xHTTP(t, reg, "DELETE", "/dirs/d1/files",
		`{"f4":{},"f5":{"meta":{"epoch":1,"foo":"bar"}, "foo":"bar"}}`,
		204, "")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files", `{}`, 204, "") // no-op

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "GET /dirs/d1 - 7",
		URL:    "/dirs/d1/files",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "f6": {
    "fileid": "f6",
    "versionid": "v6.1",
    "self": "http://localhost:8181/dirs/d1/files/f6$details",
    "xid": "/dirs/d1/files/f6",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v6.1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f6/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f6/versions",
    "versionscount": 1
  }
}
`,
	})

	xHTTP(t, reg, "DELETE", "/dirs/d1/files", ``, 204, "")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:    "GET /dirs/d1 - 7",
		URL:     "/dirs/d1/files",
		Method:  "GET",
		Code:    200,
		ResBody: "{}\n",
	})

	// TODO
	// DEL /dirs/d1/files [ f2,f4 ] - bad epoch on 2nd,verify f2 is still there

	// DELETE Versions
	f1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)
	f1.AddVersion("v2")
	f1.AddVersion("v3")
	v4, _ := f1.AddVersion("v4")
	v5, _ := f1.AddVersion("v5")
	xNoErr(t, f1.SetDefault(v5))
	f1.AddVersion("v6")
	f1.AddVersion("v7")
	f1.AddVersion("v8")
	f1.AddVersion("v9")
	f1.AddVersion("v10")

	t.Logf("v4.old: %s", ToJSON(v4.Object))
	t.Logf("v4.new: %s", ToJSON(v4.NewObject))

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200,
		`{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v4",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 10
}
`)
	// DELETE v1
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/vx", "", 404,
		"Version \"vx\" not found\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v1", "", 204, "")
	// v2's epoch/modifiedat should change due to changing its ancestor

	// DELETE /dirs/d1/files/f1?epoch=...
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2?epoch=2x", "", 400,
		"Epoch value \"2x\" must be an UINTEGER\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2?epoch=3", "", 400,
		"Epoch value for \"v2\" must be 2 not 3\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2?epoch=2", "", 204, "")

	// DELETE /dirs/d1/files/f1/versions/v4 - bad epoch in body
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v4":{"epoch":"1x"}}`, 400,
		"Epoch value for \"v4\" must be a uinteger\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v4":{"epoch":2}}`, 400,
		"Epoch value for \"v4\" must be 1 not 2\n")

	// DELETE - bad IDs
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v4":{"fileid":2}}`, 204, "") // ignore fileid
	f1.AddVersion("v4")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v4":{"versionid":2}}`, 400,
		`"versionid" value for "v4" must be "v4" not "2"
`)
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v4":{"fileid":"fx","versionid":"v4"}}`, 204, "") // ignore fileid
	f1.AddVersion("v4")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `{"v4":{"epoch":1}}`, 204, "")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `{"v4":{"epoch":1}}`, 204, "")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `{"vx":{"epoch":1}}`, 204, "")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v6":{},"v7":{"epoch":3}}`, // v6 will still be around
		400, "Epoch value for \"v7\" must be 1 not 3\n")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v7":{},"v8":{"epoch":1}}`,
		204, "")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `{}`, 204, "") // No-op

	// Make sure we have some left, and default is still v5
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", "", 200, `{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v5",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 9,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:03Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v5",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v10": {
      "fileid": "f1",
      "versionid": "v10",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10$details",
      "xid": "/dirs/d1/files/f1/versions/v10",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "ancestor": "v9"
    },
    "v3": {
      "fileid": "f1",
      "versionid": "v3",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3$details",
      "xid": "/dirs/d1/files/f1/versions/v3",
      "epoch": 2,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:04Z",
      "ancestor": "v3"
    },
    "v5": {
      "fileid": "f1",
      "versionid": "v5",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v5$details",
      "xid": "/dirs/d1/files/f1/versions/v5",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:02Z",
      "ancestor": "v5"
    },
    "v6": {
      "fileid": "f1",
      "versionid": "v6",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v6$details",
      "xid": "/dirs/d1/files/f1/versions/v6",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "ancestor": "v5"
    },
    "v9": {
      "fileid": "f1",
      "versionid": "v9",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v9$details",
      "xid": "/dirs/d1/files/f1/versions/v9",
      "epoch": 2,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:03Z",
      "ancestor": "v9"
    }
  },
  "versionscount": 5
}
`)

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v5?setdefaultversionid=v3",
		``, 204, "")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setdefaultversionid=v9",
		``, 400, "Can't set defaultversionid to Version being deleted\n")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setdefaultversionid=vx",
		``, 400, "Can't find next default Version \"vx\"\n")

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setdefaultversionid=v3",
		``, 204, "")
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setdefaultversionid=vx",
		``, 404, "Version \"v9\" not found\n")

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", "", 200, `{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v3",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 11,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:03Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v10": {
      "fileid": "f1",
      "versionid": "v10",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10$details",
      "xid": "/dirs/d1/files/f1/versions/v10",
      "epoch": 2,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:03Z",
      "ancestor": "v10"
    },
    "v3": {
      "fileid": "f1",
      "versionid": "v3",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3$details",
      "xid": "/dirs/d1/files/f1/versions/v3",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:02Z",
      "ancestor": "v3"
    },
    "v6": {
      "fileid": "f1",
      "versionid": "v6",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v6$details",
      "xid": "/dirs/d1/files/f1/versions/v6",
      "epoch": 2,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:04Z",
      "ancestor": "v6"
    }
  },
  "versionscount": 3
}
`)

	f1.AddVersion("v1")
	// bad next
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setdefaultversionid=vx", `{"v6":{}}`, 400, "Can't find next default Version \"vx\"\n")
	// next = being deleted
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setdefaultversionid=v6", `{"v6":{}}`, 400, "Can't set defaultversionid to Version being deleted\n")

	// delete non-default, change default
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setdefaultversionid=v10", `{"v6":{}}`, 204, "")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", "", 200, `{
  "fileid": "f1",
  "versionid": "v10",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v10",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 13,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:03Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v10",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v10",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 2,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:04Z",
      "modifiedat": "2024-01-01T12:00:03Z",
      "ancestor": "v1"
    },
    "v10": {
      "fileid": "f1",
      "versionid": "v10",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10$details",
      "xid": "/dirs/d1/files/f1/versions/v10",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:02Z",
      "ancestor": "v10"
    },
    "v3": {
      "fileid": "f1",
      "versionid": "v3",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3$details",
      "xid": "/dirs/d1/files/f1/versions/v3",
      "epoch": 2,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:05Z",
      "ancestor": "v3"
    }
  },
  "versionscount": 3
}
`)

	// delete non-default, default not move
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `{"v3":{}}`, 204, "")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", "", 200, `{
  "fileid": "f1",
  "versionid": "v10",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v10",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 14,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:03Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v10",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v10",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 2,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:04Z",
      "modifiedat": "2024-01-01T12:00:05Z",
      "ancestor": "v1"
    },
    "v10": {
      "fileid": "f1",
      "versionid": "v10",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10$details",
      "xid": "/dirs/d1/files/f1/versions/v10",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:02Z",
      "ancestor": "v10"
    }
  },
  "versionscount": 2
}
`)
	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setdefaultversionid=v1", `{}`, 204, "")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", "", 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 15,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:04Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:02Z",
      "ancestor": "v1"
    },
    "v10": {
      "fileid": "f1",
      "versionid": "v10",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v10$details",
      "xid": "/dirs/d1/files/f1/versions/v10",
      "epoch": 2,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:05Z",
      "ancestor": "v10"
    }
  },
  "versionscount": 2
}
`)

	xHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", ``, 204, "")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions", "", 404,
		"\"dirs/d1/files/f1\" not found\n")

	// TODO
	// DEL /..versions/ [ v2,v4 ] - bad epoch on 2nd,verify v2 is still there
}

func TestHTTPRequiredFields(t *testing.T) {
	reg := NewRegistry("TestHTTPRequiredFields")
	defer PassDeleteReg(t, reg)

	_, err := reg.Model.AddAttribute(&registry.Attribute{
		Name:     "req1",
		Type:     STRING,
		Required: true,
	})
	xNoErr(t, err)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	_, err = gm.AddAttribute(&registry.Attribute{
		Name:     "req2",
		Type:     STRING,
		Required: true,
	})
	xNoErr(t, err)

	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	_, err = rm.AddAttribute(&registry.Attribute{
		Name:     "req3",
		Type:     STRING,
		Required: true,
	})
	xNoErr(t, err)

	// Must commit before we call Set below otherwise the transaction will
	// be rolled back
	reg.SaveAllAndCommit()

	// Registry itself
	err = reg.SetSave("description", "testing")
	xCheckErr(t, err, "Required property \"req1\" is missing")

	xNoErr(t, reg.JustSet("req1", "testing1"))
	xNoErr(t, reg.SetSave("description", "testing"))

	xHTTP(t, reg, "GET", "/", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRequiredFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "description": "testing",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "req1": "testing1",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0
}
`)

	// Groups
	xHTTP(t, reg, "PUT", "/dirs/d1", `{"description": "testing"}`, 400,
		`Required property "req2" is missing`+"\n")

	xHTTP(t, reg, "PUT", "/dirs/d1", `{
  "description": "testing",
  "req2": "testing2"
}`, 201, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "description": "testing",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "req2": "testing2",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 0
}
`)

	// Resources
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{"description": "testing"}`, 400,
		`Required property "req3" is missing`+"\n")

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "description": "testingdesc3",
  "req3": "testing3"
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "description": "testingdesc3",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "req3": "testing3",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-description: testingdesc",
		},

		Code:       400,
		ResHeaders: []string{},
		ResBody: `Required property "req3" is missing
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2$details",
		Method: "PUT",
		ReqBody: `{
  "description": "testingdesc2"
}`,

		Code:       400,
		ResHeaders: []string{},
		ResBody: `Required property "req3" is missing
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-description: desctesting",
			"xRegistry-req3: testing3",
		},

		Code: 201,
		ResHeaders: []string{
			"xRegistry-fileid: f2",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f2",
			"xRegistry-xid: /dirs/d1/files/f2",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-description: desctesting",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-req3: testing3",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f2/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f2/versions",
			"xRegistry-versionscount: 1",

			"Content-Length: 0",
			"Content-Location: http://localhost:8181/dirs/d1/files/f2/versions/1",
			"Content-Disposition:f2",
			"Location: http://localhost:8181/dirs/d1/files/f2",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2$details",
		Method: "PUT",
		ReqBody: `{
  "description": "desctesting3",
  "req3": "testing4"
}`,

		Code: 200,
		ResBody: `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2$details",
  "xid": "/dirs/d1/files/f2",
  "epoch": 2,
  "isdefault": true,
  "description": "desctesting3",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "req3": "testing4",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2/versions/1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-versionid: 1",
			"xRegistry-description: desctesting",
			"xRegistry-req3: null",
		},

		Code:    400,
		ResBody: "Required property \"req3\" is missing\n",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2/versions/1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-versionid: 1",
			"xRegistry-description: desctesting",
			"xRegistry-req3: null",
		},

		Code:    400,
		ResBody: "Required property \"req3\" is missing\n",
	})
}
