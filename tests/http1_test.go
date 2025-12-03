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
	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "Create empty model",
		URL:        "/model",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,

		Code:       405,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#action_not_supported",
  "title": "The specified action (PUT) is not supported for: /model.",
  "detail": "Use \"/modelsource\" instead of \"/model\".",
  "subject": "/model",
  "args": {
    "action": "PUT"
  },
  "source": ":registry:httpStuff:1841"
}
`,
	})

	// Create model tests
	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
}
`,
	})

	XHTTP(t, reg, "PUT", "/", `{"description": "testing"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"description\" for \"/\" is not valid: value (testing) must be one of the enum values: one, two.",
  "subject": "/",
  "args": {
    "error_detail": "value (testing) must be one of the enum values: one, two",
    "name": "description"
  },
  "source": ":registry:entity:2581"
}
`)

	XHTTP(t, reg, "PUT", "/", `{}`, 200, `{
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

	XHTTP(t, reg, "PUT", "/", `{"description": "two"}`, 200, `{
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
	XCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("mybool", BOOLEAN)
	XCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("mydec", DECIMAL)
	XCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myint", INTEGER)
	XCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("mystr", STRING)
	XCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("mytime", TIMESTAMP)
	XCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myuint", UINTEGER)
	XCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myuri", URI)
	XCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myuriref", URI_REFERENCE)
	XCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myuritemplate", URI_TEMPLATE)
	XCheckErr(t, err, "")
	_, err = reg.Model.AddAttr("myurl", URL)
	XCheckErr(t, err, "")

	attr, err := reg.Model.AddAttrObj("myobj1")
	XCheckErr(t, err, "")
	_, err = attr.AddAttr("mystr1", STRING)
	XCheckErr(t, err, "")
	_, err = attr.AddAttr("myint1", INTEGER)
	XCheckErr(t, err, "")
	_, err = attr.AddAttr("*", ANY)
	XCheckErr(t, err, "")

	attr, _ = reg.Model.AddAttrObj("myobj2")
	attr.AddAttr("mystr2", STRING)
	obj2, err := attr.AddAttrObj("myobj2_1")
	XCheckErr(t, err, "")
	_, err = obj2.AddAttr("*", INTEGER)
	XCheckErr(t, err, "")

	item := registry.NewItemType(ANY)
	attr, err = reg.Model.AddAttrArray("myarrayany", item)
	XCheckErr(t, err, "")
	attr, err = reg.Model.AddAttrMap("mymapany", item)
	XCheckErr(t, err, "")

	item = registry.NewItemType(UINTEGER)
	attr, err = reg.Model.AddAttrArray("myarrayuint", item)
	XCheckErr(t, err, "")
	attr, err = reg.Model.AddAttrMap("mymapuint", item)
	XCheckErr(t, err, "")

	item = registry.NewItemObject()
	attr, err = reg.Model.AddAttrArray("myarrayemptyobj", item)
	XCheckErr(t, err, "")

	item = registry.NewItemObject()
	item.AddAttr("mapobj_int", INTEGER)
	attr, err = reg.Model.AddAttrMap("mymapobj", item)
	XCheckErr(t, err, "")

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - empty string id",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "{ \"registryid\": \"\" }",
		Code:       400,
		ResHeaders: []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"registryid\" for \"/\" is not valid: can't be an empty string.",
  "subject": "/",
  "args": {
    "error_detail": "can't be an empty string",
    "name": "registryid"
  },
  "source": ":registry:entity:819"
}
`})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad epoch",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "epoch":33
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (33) for \"/\" does not match its current value (4).",
  "subject": "/",
  "args": {
    "bad_epoch": "33",
    "epoch": "4"
  },
  "source": ":registry:entity:1003"
}
`})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mymapobj.mapobj_int\" for \"/\" is not valid: must be a map[string] or object.",
  "subject": "/",
  "args": {
    "error_detail": "must be a map[string] or object",
    "name": "mymapobj.mapobj_int"
  },
  "source": ":registry:entity:1979"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (123) for \"/\" does not match its current value (6).",
  "subject": "/",
  "args": {
    "bad_epoch": "123",
    "epoch": "6"
  },
  "source": ":registry:entity:1003"
}
`},
		{request: `{"epoch":-123}`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"epoch\" for \"/\" is not valid: must be a uinteger.",
  "subject": "/",
  "args": {
    "error_detail": "must be a uinteger",
    "name": "epoch"
  },
  "source": ":registry:entity:2405"
}
`},
		{request: `{"epoch":"asd"}`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"epoch\" for \"/\" is not valid: must be a uinteger.",
  "subject": "/",
  "args": {
    "error_detail": "must be a uinteger",
    "name": "epoch"
  },
  "source": ":registry:entity:2393"
}
`},
		{request: `{"mybool":123}`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mybool\" for \"/\" is not valid: must be a boolean.",
  "subject": "/",
  "args": {
    "error_detail": "must be a boolean",
    "name": "mybool"
  },
  "source": ":registry:entity:2359"
}
`},
		{request: `{"mybool":"False"}`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mybool\" for \"/\" is not valid: must be a boolean.",
  "subject": "/",
  "args": {
    "error_detail": "must be a boolean",
    "name": "mybool"
  },
  "source": ":registry:entity:2359"
}
`},
		{request: `{"mydec":[ 1 ]}`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mydec\" for \"/\" is not valid: must be a decimal.",
  "subject": "/",
  "args": {
    "error_detail": "must be a decimal",
    "name": "mydec"
  },
  "source": ":registry:entity:2365"
}
`},
		{request: `{"mydec": "asd" }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mydec\" for \"/\" is not valid: must be a decimal.",
  "subject": "/",
  "args": {
    "error_detail": "must be a decimal",
    "name": "mydec"
  },
  "source": ":registry:entity:2365"
}
`},
		{request: `{"myint": 1.01 }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myint\" for \"/\" is not valid: must be an integer.",
  "subject": "/",
  "args": {
    "error_detail": "must be an integer",
    "name": "myint"
  },
  "source": ":registry:entity:2373"
}
`},
		{request: `{"myint": {} }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myint\" for \"/\" is not valid: must be an integer.",
  "subject": "/",
  "args": {
    "error_detail": "must be an integer",
    "name": "myint"
  },
  "source": ":registry:entity:2378"
}
`},
		{request: `{"mystr": {} }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mystr\" for \"/\" is not valid: must be a string.",
  "subject": "/",
  "args": {
    "error_detail": "must be a string",
    "name": "mystr"
  },
  "source": ":registry:entity:2513"
}
`},
		{request: `{"mystr": 123 }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mystr\" for \"/\" is not valid: must be a string.",
  "subject": "/",
  "args": {
    "error_detail": "must be a string",
    "name": "mystr"
  },
  "source": ":registry:entity:2513"
}
`},
		{request: `{"mystr": true }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mystr\" for \"/\" is not valid: must be a string.",
  "subject": "/",
  "args": {
    "error_detail": "must be a string",
    "name": "mystr"
  },
  "source": ":registry:entity:2513"
}
`},
		{request: `{"mytime": true }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mytime\" for \"/\" is not valid: must be a timestamp.",
  "subject": "/",
  "args": {
    "error_detail": "must be a timestamp",
    "name": "mytime"
  },
  "source": ":registry:entity:2543"
}
`},
		{request: `{"mytime": "12-12-12" }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mytime\" for \"/\" is not valid: is a malformed timestamp.",
  "subject": "/",
  "args": {
    "error_detail": "is a malformed timestamp",
    "name": "mytime"
  },
  "source": ":registry:entity:2552"
}
`},
		{request: `{"myuint": "str" }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myuint\" for \"/\" is not valid: must be a uinteger.",
  "subject": "/",
  "args": {
    "error_detail": "must be a uinteger",
    "name": "myuint"
  },
  "source": ":registry:entity:2393"
}
`},
		{request: `{"myuint": "123" }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myuint\" for \"/\" is not valid: must be a uinteger.",
  "subject": "/",
  "args": {
    "error_detail": "must be a uinteger",
    "name": "myuint"
  },
  "source": ":registry:entity:2393"
}
`},
		{request: `{"myuint": -123 }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myuint\" for \"/\" is not valid: must be a uinteger.",
  "subject": "/",
  "args": {
    "error_detail": "must be a uinteger",
    "name": "myuint"
  },
  "source": ":registry:entity:2405"
}
`},
		{request: `{"myuri": 123 }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myuri\" for \"/\" is not valid: must be a uri.",
  "subject": "/",
  "args": {
    "error_detail": "must be a uri",
    "name": "myuri"
  },
  "source": ":registry:entity:2519"
}
`},
		{request: `{"myuriref": 123 }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myuriref\" for \"/\" is not valid: must be a uri-reference.",
  "subject": "/",
  "args": {
    "error_detail": "must be a uri-reference",
    "name": "myuriref"
  },
  "source": ":registry:entity:2525"
}
`},
		{request: `{"myuritemplate": 123 }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myuritemplate\" for \"/\" is not valid: must be a uri-template.",
  "subject": "/",
  "args": {
    "error_detail": "must be a uri-template",
    "name": "myuritemplate"
  },
  "source": ":registry:entity:2531"
}
`},
		{request: `{"myurl": 123 }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \" myurl\" for \"/\" is not valid: must be a url.",
  "subject": "/",
  "args": {
    "error_detail": "must be a url",
    "name": " myurl"
  },
  "source": ":registry:entity:2537"
}
`},
		{request: `{"myobj1": 123 }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myobj1\" for \"/\" is not valid: must be a map[string] or object.",
  "subject": "/",
  "args": {
    "error_detail": "must be a map[string] or object",
    "name": "myobj1"
  },
  "source": ":registry:entity:1979"
}
`},
		{request: `{"myobj1": [ 123 ] }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myobj1\" for \"/\" is not valid: must be a map[string] or object.",
  "subject": "/",
  "args": {
    "error_detail": "must be a map[string] or object",
    "name": "myobj1"
  },
  "source": ":registry:entity:1979"
}
`},
		{request: `{"myobj1": { "mystr1": 123 } }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myobj1.mystr1\" for \"/\" is not valid: must be a string.",
  "subject": "/",
  "args": {
    "error_detail": "must be a string",
    "name": "myobj1.mystr1"
  },
  "source": ":registry:entity:2513"
}
`},
		{request: `{"myobj2": { "ext": 123 } }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (myobj2.ext) was specified for \"/\".",
  "subject": "/",
  "args": {
    "name": "myobj2.ext"
  },
  "source": ":registry:entity:2202"
}
`},
		{request: `{"myobj2": { "myobj2_1": { "ext": "str" } } }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myobj2.myobj2_1.ext\" for \"/\" is not valid: must be an integer.",
  "subject": "/",
  "args": {
    "error_detail": "must be an integer",
    "name": "myobj2.myobj2_1.ext"
  },
  "source": ":registry:entity:2378"
}
`},
		{request: `{"myarrayuint": [ 123, -123 ] }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myarrayuint[1]\" for \"/\" is not valid: must be a uinteger.",
  "subject": "/",
  "args": {
    "error_detail": "must be a uinteger",
    "name": "myarrayuint[1]"
  },
  "source": ":registry:entity:2405"
}
`},
		{request: `{"myarrayuint": [ "asd" ] }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myarrayuint[0]\" for \"/\" is not valid: must be a uinteger.",
  "subject": "/",
  "args": {
    "error_detail": "must be a uinteger",
    "name": "myarrayuint[0]"
  },
  "source": ":registry:entity:2393"
}
`},
		{request: `{"myarrayuint": [ 123, null] }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myarrayuint[1]\" for \"/\" is not valid: must be a uinteger.",
  "subject": "/",
  "args": {
    "error_detail": "must be a uinteger",
    "name": "myarrayuint[1]"
  },
  "source": ":registry:entity:2393"
}
`},
		{request: `{"myarrayuint": 123 }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myarrayuint\" for \"/\" is not valid: must be an array.",
  "subject": "/",
  "args": {
    "error_detail": "must be an array",
    "name": "myarrayuint"
  },
  "source": ":registry:entity:2315"
}
`},
		{request: `{"mymapuint": 123 }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mymapuint\" for \"/\" is not valid: must be a map.",
  "subject": "/",
  "args": {
    "error_detail": "must be a map",
    "name": "mymapuint"
  },
  "source": ":registry:entity:2253"
}
`},
		{request: `{"mymapuint": { "asd" : -123 }}`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mymapuint.asd\" for \"/\" is not valid: must be a uinteger.",
  "subject": "/",
  "args": {
    "error_detail": "must be a uinteger",
    "name": "mymapuint.asd"
  },
  "source": ":registry:entity:2405"
}
`},
		// {request: `{"mymapuint": { "asd" : null }}`,
		// response: `attribute "mymapuint.asd" must be a uinteger`},
		{request: `{"myarrayemptyobj": [ { "asd": true } ] }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (myarrayemptyobj[0].asd) was specified for \"/\".",
  "subject": "/",
  "args": {
    "name": "myarrayemptyobj[0].asd"
  },
  "source": ":registry:entity:2202"
}
`},
		{request: `{"myarrayemptyobj": [ [ true ] ] }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myarrayemptyobj[0]\" for \"/\" is not valid: must be a map[string] or object.",
  "subject": "/",
  "args": {
    "error_detail": "must be a map[string] or object",
    "name": "myarrayemptyobj[0]"
  },
  "source": ":registry:entity:1979"
}
`},
		{request: `{"mymapobj": { "asd" : { "mapobj_int" : true } } }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mymapobj.asd.mapobj_int\" for \"/\" is not valid: must be an integer.",
  "subject": "/",
  "args": {
    "error_detail": "must be an integer",
    "name": "mymapobj.asd.mapobj_int"
  },
  "source": ":registry:entity:2386"
}
`},
		{request: `{"mymapobj": { "asd" : { "qwe" : true } } }`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (mymapobj.asd.qwe) was specified for \"/\".",
  "subject": "/",
  "args": {
    "name": "mymapobj.asd.qwe"
  },
  "source": ":registry:entity:2202"
}
`},
		{request: `{"mymapobj": [ true ]}`,
			response: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mymapobj\" for \"/\" is not valid: must be a map.",
  "subject": "/",
  "args": {
    "error_detail": "must be a map",
    "name": "mymapobj"
  },
  "source": ":registry:entity:2261"
}
`},
	}

	for _, test := range typeTests {
		// t.Logf("Test.request: %s", test.request)
		exp := test.response
		XCheckHTTP(t, reg, &HTTPTest{
			Name:       "PUT reg - bad type - request: " + test.request,
			URL:        "/",
			Method:     "PUT",
			ReqHeaders: []string{},
			ReqBody:    test.request,
			Code:       400,
			ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
			ResBody:    exp,
		})
	}

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad self - ignored",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "self": 123
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \" self\" for \"/\" is not valid: must be a url.",
  "subject": "/",
  "args": {
    "error_detail": "must be a url",
    "name": " self"
  },
  "source": ":registry:entity:2545"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad xid - ignored",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "xid": 123
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"xid\" for \"/\" is not valid: must be an xid.",
  "subject": "/",
  "args": {
    "error_detail": "must be an xid",
    "name": "xid"
  },
  "source": ":registry:entity:2419"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad id",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "registryid": 123
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"registryid\" for \"/\" is not valid: must be a string.",
  "subject": "/",
  "args": {
    "error_detail": "must be a string",
    "name": "registryid"
  },
  "source": ":registry:entity:2521"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - bad id",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "registryid": "foo"
}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"registryid\" value (foo) for \"/\" needs to be \"TestHTTPRegistry\".",
  "subject": "/",
  "args": {
    "expected_id": "TestHTTPRegistry",
    "invalid_id": "foo",
    "singular": "registry"
  },
  "source": ":registry:entity:830"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT groups - fail",
		URL:        "/dirs",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       405,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#action_not_supported",
  "title": "The specified action (PUT) is not supported for: /dirs.",
  "detail": "PUT not allowed on collections.",
  "subject": "/dirs",
  "args": {
    "action": "PUT"
  },
  "source": ":registry:httpStuff:1868"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XHTTP(t, reg, "GET", "/dirs", "", 200, `{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"dirid\" value (dir44) for \"/dirs/dir4\" needs to be \"dir4\".",
  "subject": "/dirs/dir4",
  "args": {
    "expected_id": "dir4",
    "invalid_id": "dir44",
    "singular": "dir"
  },
  "source": ":registry:entity:830"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (10) for \"/dirs/dir1\" does not match its current value (4).",
  "subject": "/dirs/dir1",
  "args": {
    "bad_epoch": "10",
    "epoch": "4"
  },
  "source": ":registry:entity:1003"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT group - update - err id",
		URL:        "/dirs/dir1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{ "dirid":"dir2" }`,
		Code:       400,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"dirid\" value (dir2) for \"/dirs/dir1\" needs to be \"dir1\".",
  "subject": "/dirs/dir1",
  "args": {
    "expected_id": "dir1",
    "invalid_id": "dir2",
    "singular": "dir"
  },
  "source": ":registry:entity:830"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"dirid\" value (dir3) for \"/dirs/dir2\" needs to be \"dir2\".",
  "subject": "/dirs/dir2",
  "args": {
    "expected_id": "dir2",
    "invalid_id": "dir3",
    "singular": "dir"
  },
  "source": ":registry:entity:830"
}
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH / - update name",
		URL:        "/",
		Method:     "PATCH",
		ReqBody:    `{"name":"hello"}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - empty",
		URL:        "/",
		Method:     "POST",
		ReqBody:    "",
		Code:       400,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#missing_body",
  "title": "The request is missing an HTTP body - try '{}'.",
  "subject": "/",
  "source": ":registry:httpStuff:3119"
}
`})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - empty",
		URL:        "/",
		Method:     "POST",
		ReqBody:    "{}",
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody:    `{}` + "\n",
	})

	// Check epoch didn't change
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
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

	XCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs",
		Method:  "DELETE",
		Code:    204,
		ResBody: "*",
	})

	reg.Refresh(registry.FOR_READ)
	XCheck(t, regEpoch != reg.Get("epoch"), "regEpoch should be 1")
	XCheck(t, regTime != reg.Get("createdat"), "regEpoch should be 1")

	// Epoch bumped
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - err",
		URL:        "/",
		Method:     "POST",
		ReqBody:    `{"dirs":{"d1":{}}, "foos":{"f1":{},"f2":{"foo":"bar"}}}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (foo) was specified for \"/foos/f2\".",
  "subject": "/foos/f2",
  "args": {
    "name": "foo"
  },
  "source": ":registry:entity:2202"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - err - bad type",
		URL:        "/",
		Method:     "POST",
		ReqBody:    `{"name": "foo"}`,
		Code:       400,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#groups_only",
  "title": "Attribute \"name\" is invalid. Only Group types are allowed to be specified on this request: /.",
  "subject": "/",
  "args": {
    "name": "name"
  },
  "source": ":registry:httpStuff:1927"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST / - err - bad group",
		URL:        "/",
		Method:     "POST",
		ReqBody:    `{"name": { "foo":{} } }`,
		Code:       400,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#groups_only",
  "title": "Attribute \"name\" is invalid. Only Group types are allowed to be specified on this request: /.",
  "subject": "/",
  "args": {
    "name": "name"
  },
  "source": ":registry:httpStuff:1927"
}
`,
	})

	// Make sure nothing changed at the registry level
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources - fail",
		URL:        "/dirs/dir1/files",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       405,
		ResHeaders: []string{"Content-Type:application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#action_not_supported",
  "title": "The specified action (PUT) is not supported for: /dirs/dir1/files.",
  "detail": "PUT not allowed on collections.",
  "subject": "/dirs/dir1/files",
  "args": {
    "action": "PUT"
  },
  "source": ":registry:httpStuff:1868"
}
`,
	})

	XHTTP(t, reg, "POST", "/dirs/dir1/files", "{}", 200, "{}\n")

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - w/bad header - file",
		URL:    "/dirs/dir1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-file: hello",
		},
		ReqBody:    "My cool doc",
		Code:       400,
		ResHeaders: []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xregistry_header",
  "title": "xRegistry HTTP header \"file\" is not allowed on this request: 'xRegistry-file' isn't allowed as an HTTP header.",
  "subject": "/dirs/dir1/files/f1",
  "args": {
    "error_detail": "'xRegistry-file' isn't allowed as an HTTP header",
    "name": "file"
  },
  "source": ":registry:httpStuff:3207"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources - w/bad header - filebase64",
		URL:    "/dirs/dir1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-filebase64: aGVsbG8=",
		},
		ReqBody:    "My cool doc",
		Code:       400,
		ResHeaders: []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xregistry_header",
  "title": "xRegistry HTTP header \"filebase64\" is not allowed on this request: 'xRegistry-filebase64' isn't allowed as an HTTP header.",
  "subject": "/dirs/dir1/files/f1",
  "args": {
    "error_detail": "'xRegistry-filebase64' isn't allowed as an HTTP header",
    "name": "filebase64"
  },
  "source": ":registry:httpStuff:3207"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources - w/doc - bad id",
		URL:        "/dirs/dir1/files/f1",
		Method:     "PUT",
		ReqHeaders: []string{"xRegistry-fileid:f2"},
		ReqBody:    "My cool doc",
		Code:       400,
		ResHeaders: []string{
			"Content-Type: application/json; charset=utf-8",
		},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"fileid\" value (f2) for \"/dirs/dir1/files/f1\" needs to be \"f1\".",
  "subject": "/dirs/dir1/files/f1",
  "args": {
    "expected_id": "f1",
    "invalid_id": "f2",
    "singular": "file"
  },
  "source": ":registry:httpStuff:2166"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xregistry_header",
  "title": "xRegistry HTTP header \"xRegistry-fileurl\" is not allowed on this request: header isn't allowed if there's a body.",
  "subject": "/dirs/dir1/files/f3",
  "args": {
    "error_detail": "header isn't allowed if there's a body",
    "name": "xRegistry-fileurl"
  },
  "source": ":registry:httpStuff:3219"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XNoErr(t, err)
	body, err := io.ReadAll(res.Body)
	XNoErr(t, err)

	XCheckHTTP(t, reg, &HTTPTest{
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
	XNoErr(t, err)
	body, err = io.ReadAll(res.Body)
	XNoErr(t, err)

	resBody := strings.Replace(string(body), `"epoch": 11`, `"epoch": 12`, 1)
	XCheckHTTP(t, reg, &HTTPTest{
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
	XNoErr(t, err)
	body, err = io.ReadAll(res.Body)
	XNoErr(t, err)

	// Change the modifiedat field since it'll change
	re := regexp.MustCompile(`"modifiedat": "[^"]*"`)
	body = re.ReplaceAll(body, []byte(`"modifiedat": "2024-01-01T12:12:12Z"`))

	resBody = strings.Replace(string(body), `"epoch": 1`, `"epoch": 2`, 1)

	XCheckHTTP(t, reg, &HTTPTest{
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

	// Some errors
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "Bad type",
		URL:    "/dirs/dir1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-name-key: foo",
		},
		ReqBody:     "",
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"name\" for \"/dirs/dir1/files/f1/versions/1\" is not valid: must be a string.",
  "subject": "/dirs/dir1/files/f1/versions/1",
  "args": {
    "error_detail": "must be a string",
    "name": "name"
  },
  "source": ":registry:entity:2523"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "unknwon attr",
		URL:    "/dirs/dir1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-foo: foo",
		},
		ReqBody:     "",
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (foo) was specified for \"/dirs/dir1/files/f1/versions/1\".",
  "subject": "/dirs/dir1/files/f1/versions/1",
  "args": {
    "name": "foo"
  },
  "source": ":registry:entity:2193"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "Bad type",
		URL:    "/dirs/dir1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-meta: foo",
		},
		ReqBody:     "",
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "\"meta\" must be an object.",
  "subject": "/dirs/dir1/files/f1",
  "args": {
    "error_detail": "\"meta\" must be an object"
  },
  "source": ":registry:group:178"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "Bad type - label",
		URL:    "/dirs/dir1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-labels: foo",
		},
		ReqBody:     ``,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"labels\" for \"/dirs/dir1/files/f1/versions/1\" is not valid: must be a map.",
  "subject": "/dirs/dir1/files/f1/versions/1",
  "args": {
    "error_detail": "must be a map",
    "name": "labels"
  },
  "source": ":registry:entity:2264"
}
`,
	})

	// This one used to randomlay fail based on the order in which the
	// headers were processed. It would sometimes create a map, erase it with
	// a string and then see another map entry - and that would cause
	// an error because it can't add a map entry to a string.
	// However, that should be fixed and will panic if the code is messed-up.
	// Leaving this test here in case it breaks again - it'll only randomly
	// fail though. Very hard to for the order of processing in this case.
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "Bad type",
		URL:    "/dirs/dir1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-foo: foo",
			"xRegistry-foo: foo",
			"xRegistry-foo: foo",
			"xRegistry-foo-bar-car: foo",
			"xRegistry-foo-bar-dar: foo",
			"xRegistry-foo-bar-far: foo",
			"xRegistry-foo-bar-gar: foo",
		},
		ReqBody:     ``,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (foo) was specified for \"/dirs/dir1/files/f1/versions/1\".",
  "subject": "/dirs/dir1/files/f1/versions/1",
  "args": {
    "name": "foo"
  },
  "source": ":registry:entity:2193"
}
`,
	})
}

func TestHTTPCases(t *testing.T) {
	reg := NewRegistry("TestHTTPCases")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	d, _ := reg.AddGroup("dirs", "d1")
	d.AddResource("files", "f1", "v1")

	XHTTP(t, reg, "GET", "/Dirs", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/D1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/dirs/D1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/D1) cannot be found.",
  "subject": "/dirs/D1",
  "source": ":registry:httpStuff:1730"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/Files", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/Files) cannot be found.",
  "detail": "Unknown Resource type: Files.",
  "subject": "/dirs/d1/Files",
  "source": ":registry:info:595"
}
`)
	XHTTP(t, reg, "GET", "/dirs/D1/Files", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/D1/Files) cannot be found.",
  "detail": "Unknown Resource type: Files.",
  "subject": "/dirs/D1/Files",
  "source": ":registry:info:595"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/D1/Files", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/d1/Files", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/d1/files", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/F1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/F1) cannot be found.",
  "subject": "/dirs/d1/files/F1",
  "source": ":registry:httpStuff:1395"
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/Files/F1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/Files) cannot be found.",
  "detail": "Unknown Resource type: Files.",
  "subject": "/dirs/d1/Files",
  "source": ":registry:info:595"
}
`)
	XHTTP(t, reg, "GET", "/dirs/D1/Files/F1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/D1/Files) cannot be found.",
  "detail": "Unknown Resource type: Files.",
  "subject": "/dirs/D1/Files",
  "source": ":registry:info:595"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/D1/Files/F1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/D1/Files/f1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/D1/files/f1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/d1/Files/f1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/dirs/D1/files/F1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/D1/files/F1) cannot be found.",
  "subject": "/dirs/D1/files/F1",
  "source": ":registry:httpStuff:1395"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/Versions", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/Versions) cannot be found.",
  "detail": "Expected \"versions\" or \"meta\", got: Versions",
  "subject": "/dirs/d1/files/f1/Versions",
  "source": ":registry:info:631"
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/Files/f1/Versions", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/Files) cannot be found.",
  "detail": "Unknown Resource type: Files.",
  "subject": "/dirs/d1/Files",
  "source": ":registry:info:595"
}
`)
	XHTTP(t, reg, "GET", "/dirs/D1/Files/f1/Versions", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/D1/Files) cannot be found.",
  "detail": "Unknown Resource type: Files.",
  "subject": "/dirs/D1/Files",
  "source": ":registry:info:595"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/D1/Files/f1/Versions", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/D1/Files/f1/versions", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/D1/files/f1/Versions", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/d1/Files/f1/Versions", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/dirs/D1/Files/f1/Versions", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/D1/Files) cannot be found.",
  "detail": "Unknown Resource type: Files.",
  "subject": "/dirs/D1/Files",
  "source": ":registry:info:595"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/V1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/versions/V1) cannot be found.",
  "subject": "/dirs/d1/files/f1/versions/V1",
  "source": ":registry:httpStuff:1395"
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/Versions/V1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/Versions) cannot be found.",
  "detail": "Expected \"versions\" or \"meta\", got: Versions",
  "subject": "/dirs/d1/files/f1/Versions",
  "source": ":registry:info:631"
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/Files/f1/Versions/V1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/Files) cannot be found.",
  "detail": "Unknown Resource type: Files.",
  "subject": "/dirs/d1/Files",
  "source": ":registry:info:595"
}
`)
	XHTTP(t, reg, "GET", "/dirs/D1/Files/f1/Versions/V1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/D1/Files) cannot be found.",
  "detail": "Unknown Resource type: Files.",
  "subject": "/dirs/D1/Files",
  "source": ":registry:info:595"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/D1/Files/f1/Versions/V1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/D1/Files/f1/Versions/v1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/D1/Files/f1/versions/V1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/Versions/v1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/Versions) cannot be found.",
  "detail": "Expected \"versions\" or \"meta\", got: Versions",
  "subject": "/dirs/d1/files/f1/Versions",
  "source": ":registry:info:631"
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/Files/f1/versions/v1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/Files) cannot be found.",
  "detail": "Unknown Resource type: Files.",
  "subject": "/dirs/d1/Files",
  "source": ":registry:info:595"
}
`)
	XHTTP(t, reg, "GET", "/dirs/D1/Files/f1/versions/v1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/D1/Files) cannot be found.",
  "detail": "Unknown Resource type: Files.",
  "subject": "/dirs/D1/Files",
  "source": ":registry:info:595"
}
`)
	XHTTP(t, reg, "GET", "/Dirs/d1/files/f1/versions/v1", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/Dirs) cannot be found.",
  "detail": "Unknown Group type: Dirs.",
  "subject": "/Dirs",
  "source": ":registry:info:562"
}
`)

	// Just to make sure we didn't have a typo above
	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XHTTP(t, reg, "PUT", "/dirs/d1", `{ "dirid": "d1" }`, 200, `{
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

	XHTTP(t, reg, "PUT", "/dirs/D1", `{ "dirid": "D1" }`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Attempting to create a Group with a \"dirid\" of \"D1\", when one already exists as \"d1\".",
  "subject": "/d1",
  "args": {
    "error_detail": "Attempting to create a Group with a \"dirid\" of \"D1\", when one already exists as \"d1\""
  },
  "source": ":registry:registry:627"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1", `{ "dirid": "D1" }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"dirid\" value (D1) for \"/dirs/d1\" needs to be \"d1\".",
  "subject": "/dirs/d1",
  "args": {
    "expected_id": "d1",
    "invalid_id": "D1",
    "singular": "dir"
  },
  "source": ":registry:entity:830"
}
`)
	XHTTP(t, reg, "PATCH", "/dirs/d1", `{ "dirid": "D1" }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"dirid\" value (D1) for \"/dirs/d1\" needs to be \"d1\".",
  "subject": "/dirs/d1",
  "args": {
    "expected_id": "d1",
    "invalid_id": "D1",
    "singular": "dir"
  },
  "source": ":registry:entity:830"
}
`)

	// Resource
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{ "fileid": "f1" }`, 200, `{
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

	XHTTP(t, reg, "PUT", "/dirs/d1/files/F1$details", `{ "fileid": "F1" }`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "attempting to create a Resource with a \"fileid\" of \"F1\", when one already exists as \"f1\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "error_detail": "attempting to create a Resource with a \"fileid\" of \"F1\", when one already exists as \"f1\""
  },
  "source": ":registry:group:132"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/D1/files/f1$details", `{ "fileid": "f1" }`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Attempting to create a Group with a \"dirid\" of \"D1\", when one already exists as \"d1\".",
  "subject": "/d1",
  "args": {
    "error_detail": "Attempting to create a Group with a \"dirid\" of \"D1\", when one already exists as \"d1\""
  },
  "source": ":registry:registry:627"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{ "fileid": "F1" }`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"fileid\" value (F1) for \"/dirs/d1/files/f1$details\" needs to be \"f1\".",
  "subject": "/dirs/d1/files/f1$details",
  "args": {
    "expected_id": "f1",
    "invalid_id": "F1",
    "singular": "file"
  },
  "source": ":registry:httpStuff:2166"
}
`)
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/F1$details", `{ "fileid": "F1" }`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "attempting to create a Resource with a \"fileid\" of \"F1\", when one already exists as \"f1\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "error_detail": "attempting to create a Resource with a \"fileid\" of \"F1\", when one already exists as \"f1\""
  },
  "source": ":registry:group:132"
}
`)
	XHTTP(t, reg, "PATCH", "/dirs/D1/files/f1$details", `{ "fileid": "f1" }`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Attempting to create a Group with a \"dirid\" of \"D1\", when one already exists as \"d1\".",
  "subject": "/d1",
  "args": {
    "error_detail": "Attempting to create a Group with a \"dirid\" of \"D1\", when one already exists as \"d1\""
  },
  "source": ":registry:registry:627"
}
`)
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", `{ "fileid": "F1" }`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"fileid\" value (F1) for \"/dirs/d1/files/f1$details\" needs to be \"f1\".",
  "subject": "/dirs/d1/files/f1$details",
  "args": {
    "expected_id": "f1",
    "invalid_id": "F1",
    "singular": "file"
  },
  "source": ":registry:httpStuff:2166"
}
`)

	// Version
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", `{ "versionid": "v1" }`, 200, `{
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

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/V1$details",
		`{ "versionid": "V1" }`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Attempting to create a Version with a \"versionid\" of \"V1\", when one already exists as \"v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Attempting to create a Version with a \"versionid\" of \"V1\", when one already exists as \"v1\""
  },
  "source": ":registry:resource:966"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/F1/versions/v1$details",
		`{ "versionid": "V1" }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "attempting to create a Resource with a \"fileid\" of \"F1\", when one already exists as \"f1\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "error_detail": "attempting to create a Resource with a \"fileid\" of \"F1\", when one already exists as \"f1\""
  },
  "source": ":registry:group:132"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/D1/files/f1/versions/v1$details",
		`{ "versionid": "V1" }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Attempting to create a Group with a \"dirid\" of \"D1\", when one already exists as \"d1\".",
  "subject": "/d1",
  "args": {
    "error_detail": "Attempting to create a Group with a \"dirid\" of \"D1\", when one already exists as \"d1\""
  },
  "source": ":registry:registry:627"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details",
		`{ "versionid": "V1" }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"versionid\" value (V1) for \"/dirs/d1/files/f1/versions/v1\" needs to be \"v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "expected_id": "v1",
    "invalid_id": "V1",
    "singular": "version"
  },
  "source": ":registry:httpStuff:2435"
}
`)
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions/v1$details",
		`{ "versionid": "V1" }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"versionid\" value (V1) for \"/dirs/d1/files/f1/versions/v1\" needs to be \"v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "expected_id": "v1",
    "invalid_id": "V1",
    "singular": "version"
  },
  "source": ":registry:httpStuff:2435"
}
`)

	// Test the ID in the body too (POST)

	// Group
	XHTTP(t, reg, "POST", "/dirs", `{"D1":{"dirid":"D1"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Attempting to create a Group with a \"dirid\" of \"D1\", when one already exists as \"d1\".",
  "subject": "/d1",
  "args": {
    "error_detail": "Attempting to create a Group with a \"dirid\" of \"D1\", when one already exists as \"d1\""
  },
  "source": ":registry:registry:627"
}
`)
	XHTTP(t, reg, "POST", "/dirs", `{"d1":{"dirid":"D1"}}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"dirid\" value (D1) for \"/dirs/d1\" needs to be \"d1\".",
  "subject": "/dirs/d1",
  "args": {
    "expected_id": "d1",
    "invalid_id": "D1",
    "singular": "dir"
  },
  "source": ":registry:entity:830"
}
`)
	XHTTP(t, reg, "POST", "/dirs", `{"D1":{"dirid":"d1"}}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Attempting to create a Group with a \"dirid\" of \"D1\", when one already exists as \"d1\".",
  "subject": "/d1",
  "args": {
    "error_detail": "Attempting to create a Group with a \"dirid\" of \"D1\", when one already exists as \"d1\""
  },
  "source": ":registry:registry:627"
}
`)

	// Resource
	XHTTP(t, reg, "POST", "/dirs/d1/files", `{"F1":{"fileid":"F1"}}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "attempting to create a Resource with a \"fileid\" of \"F1\", when one already exists as \"f1\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "error_detail": "attempting to create a Resource with a \"fileid\" of \"F1\", when one already exists as \"f1\""
  },
  "source": ":registry:group:132"
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files", `{"f1":{"fileid":"F1"}}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"fileid\" value (F1) for \"/dirs/d1/files/f1\" needs to be \"f1\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "expected_id": "f1",
    "invalid_id": "F1",
    "singular": "file"
  },
  "source": ":registry:group:140"
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files", `{"F1":{"fileid":"f1"}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "attempting to create a Resource with a \"fileid\" of \"F1\", when one already exists as \"f1\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "error_detail": "attempting to create a Resource with a \"fileid\" of \"F1\", when one already exists as \"f1\""
  },
  "source": ":registry:group:132"
}
`)

	// Version
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions",
		`{"vv":{"versionid":"vv}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#parsing_data",
  "title": "There was an error parsing the data: path '.vv.versionid': unterminated string.",
  "subject": "/dirs/d1/files/f1/versions",
  "args": {
    "error_detail": "path '.vv.versionid': unterminated string"
  },
  "source": ":registry:httpStuff:3154"
}
`) // just a typo first
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions$details",
		`{"vv":{"versionid":"vv"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_details",
  "title": "Use of \"$details\" in this context is not allowed: /dirs/d1/files/f1/versions$details.",
  "subject": "/dirs/d1/files/f1/versions$details",
  "source": ":registry:info:627"
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions",
		`{"V1":{"versionid":"V1"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Attempting to create a Version with a \"versionid\" of \"V1\", when one already exists as \"v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Attempting to create a Version with a \"versionid\" of \"V1\", when one already exists as \"v1\""
  },
  "source": ":registry:resource:966"
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions",
		`{"v1":{"versionid":"V1"}}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"versionid\" value (V1) for \"/dirs/d1/files/f1/versions/v1\" needs to be \"v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "expected_id": "v1",
    "invalid_id": "V1",
    "singular": "version"
  },
  "source": ":registry:entity:879"
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions",
		`{"V1":{"versionid":"v1"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Attempting to create a Version with a \"versionid\" of \"V1\", when one already exists as \"v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Attempting to create a Version with a \"versionid\" of \"V1\", when one already exists as \"v1\""
  },
  "source": ":registry:resource:966"
}
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{ "x": {"fileid":"x"}}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (x) was specified for \"/dirs/d1/files/f1/versions/1\".",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "name": "x"
  },
  "source": ":registry:entity:2203"
}
`)

	// ProxyURL
	// f, _ := d.AddResource("files", "f1-proxy", "v1")
	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
		Name:        "POST file f1-proxy - create 2 - no meta",
		URL:         "/dirs/d1/files/f1-proxy/versions",
		Method:      "POST",
		ReqHeaders:  []string{},
		ReqBody:     `this is v3`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#parsing_data",
  "title": "There was an error parsing the data: path '': invalid boolean.",
  "subject": "/dirs/d1/files/f1-proxy/versions",
  "args": {
    "error_detail": "path '': invalid boolean"
  },
  "source": ":registry:httpStuff:3154"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
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
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#one_resource",
  "title": "Only one attribute from \"file,fileurl,filebase64,fileproxyurl\" can be present at a time for: /dirs/d1/files/f1-proxy.",
  "subject": "/dirs/d1/files/f1-proxy",
  "args": {
    "list": "file,fileurl,filebase64,fileproxyurl"
  },
  "source": ":registry:shared_model:2517"
}
`,
	})

	// Update default - set 'filebase64' and 'fileurl' - error
	XCheckHTTP(t, reg, &HTTPTest{
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
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#one_resource",
  "title": "Only one attribute from \"file,fileurl,filebase64,fileproxyurl\" can be present at a time for: /dirs/d1/files/f1-proxy.",
  "subject": "/dirs/d1/files/f1-proxy",
  "args": {
    "list": "file,fileurl,filebase64,fileproxyurl"
  },
  "source": ":registry:shared_model:2517"
}
`,
	})

	// Update default - with 'filebase64'
	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XNoErr(t, reg.SaveAllAndCommit())

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)
	regEpoch := reg.GetAsInt("epoch")
	regCreated := reg.GetAsString("createdat")
	regModified := reg.GetAsString("modifiedat")

	XCheck(t, regEpoch == 1, "regEpoch should be 1")
	XCheck(t, !IsNil(regCreated), "regCreated should not be nil")
	XCheck(t, regModified == regCreated, "reg created != modified")
	XCheck(t, regModified != "", "reg modified is ''")
	XCheck(t, regCreated != "", "reg created is ''")

	d1, _ := reg.AddGroup("dirs", "d1")
	XNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)
	d1.Refresh(registry.FOR_WRITE)

	d1Epoch := d1.GetAsInt(NewPP().P("epoch").UI())
	d1Created := d1.GetAsString(NewPP().P("createdat").UI())
	d1Modified := d1.GetAsString(NewPP().P("modifiedat").UI())

	XEqual(t, "", reg.GetAsInt("epoch"), 2)
	XEqual(t, "", reg.GetAsString("createdat"), regCreated, NOMASK_TS)
	XCheckGreater(t, "", reg.GetAsString("modifiedat"), regModified)

	XEqual(t, "", d1Epoch, 1)
	XEqual(t, "", reg.GetAsString("modifiedat"), d1Created, NOMASK_TS)
	XEqual(t, "", reg.GetAsString("modifiedat"), d1Modified, NOMASK_TS)

	regEpoch = reg.GetAsInt("epoch")
	regModified = reg.GetAsString("modifiedat")

	f1, _ := d1.AddResource("files", "f1", "v1")
	f2, _ := d1.AddResource("files", "f2", "v1")
	XNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)
	d1.Refresh(registry.FOR_WRITE)
	f1.Refresh(registry.FOR_WRITE)
	v1, _ := f1.FindVersion("v1", false, registry.FOR_WRITE)
	m1, _ := f1.FindMeta(false, registry.FOR_WRITE)

	XEqual(t, "", reg.GetAsInt("epoch"), 2)
	XEqual(t, "", reg.GetAsString("createdat"), regCreated, NOMASK_TS)
	XEqual(t, "", reg.GetAsString("modifiedat"), regModified, NOMASK_TS)

	XEqual(t, "", d1.GetAsInt("epoch"), 2)
	XEqual(t, "", d1.GetAsString("createdat"), d1Created, NOMASK_TS)
	XCheckGreater(t, "", d1.GetAsString("modifiedat"), d1Modified)

	d1Epoch = d1.GetAsInt("epoch")
	d1Modified = d1.GetAsString("modifiedat")

	XEqual(t, "", m1.GetAsInt("epoch"), 1)
	XEqual(t, "", m1.GetAsString("createdat"), d1Modified, NOMASK_TS)
	XEqual(t, "", m1.GetAsString("modifiedat"), d1Modified, NOMASK_TS)

	m1Created := m1.GetAsString("createdat")
	m1Modified := m1.GetAsString("modifiedat")

	XEqual(t, "", v1.GetAsInt("epoch"), 1)
	XEqual(t, "", v1.GetAsString("createdat"), d1Modified, NOMASK_TS)
	XEqual(t, "", v1.GetAsString("modifiedat"), d1Modified, NOMASK_TS)

	v1Created := v1.GetAsString("createdat")
	v1Modified := v1.GetAsString("modifiedat")

	v2, _ := f1.AddVersion("v2")
	XNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)
	d1.Refresh(registry.FOR_WRITE)
	f1.Refresh(registry.FOR_WRITE)
	m1.Refresh(registry.FOR_WRITE)
	v1.Refresh(registry.FOR_WRITE)

	XEqual(t, "", reg.GetAsInt("epoch"), 2)
	XEqual(t, "", reg.GetAsString("createdat"), regCreated, NOMASK_TS)
	XEqual(t, "", reg.GetAsString("modifiedat"), regModified, NOMASK_TS)

	XEqual(t, "", d1.GetAsInt("epoch"), 2)
	XEqual(t, "", d1.GetAsString("createdat"), d1Created, NOMASK_TS)
	XEqual(t, "", d1.GetAsString("modifiedat"), d1Modified, NOMASK_TS)

	XEqual(t, "", m1.GetAsInt("epoch"), 2)
	XEqual(t, "", m1.GetAsString("createdat"), m1Created, NOMASK_TS)
	XCheckGreater(t, "", m1.GetAsString("modifiedat"), m1Modified)

	m1Modified = m1.GetAsString("modifiedat")

	XEqual(t, "", v1.GetAsInt("epoch"), 1)
	XEqual(t, "", v1.GetAsString("createdat"), v1Created, NOMASK_TS)
	XEqual(t, "", v1.GetAsString("modifiedat"), v1Modified, NOMASK_TS)

	XEqual(t, "", v2.GetAsInt("epoch"), 1)
	XEqual(t, "", v2.GetAsString("createdat"), m1.GetAsString("modifiedat"), NOMASK_TS)
	XEqual(t, "", v2.GetAsString("modifiedat"), m1.GetAsString("modifiedat"), NOMASK_TS)

	XHTTP(t, reg, "GET", "/?inline", ``, 200, `{
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
          "filebase64": "",

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
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
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
              "ancestor": "v1",
              "filebase64": ""
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
              "ancestor": "v1",
              "filebase64": ""
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
          "filebase64": "",

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
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/v1$details",
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
              "ancestor": "v1",
              "filebase64": ""
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
	XNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)
	d1.Refresh(registry.FOR_WRITE)
	f1.Refresh(registry.FOR_WRITE)
	m1.Refresh(registry.FOR_WRITE)
	v1.Refresh(registry.FOR_WRITE)

	XEqual(t, "", reg.GetAsInt("epoch"), 2)
	XEqual(t, "", reg.GetAsString("createdat"), regCreated, NOMASK_TS)
	XEqual(t, "", reg.GetAsString("modifiedat"), regModified, NOMASK_TS)

	XEqual(t, "", d1.GetAsInt("epoch"), 2)
	XEqual(t, "", d1.GetAsString("createdat"), d1Created, NOMASK_TS)
	XEqual(t, "", d1.GetAsString("modifiedat"), d1Modified, NOMASK_TS)

	XEqual(t, "", m1.GetAsInt("epoch"), 3)
	XEqual(t, "", m1.GetAsString("createdat"), m1Created, NOMASK_TS)
	XCheckGreater(t, "", m1.GetAsString("modifiedat"), m1Modified)

	m1Modified = m1.GetAsString("modifiedat")

	XEqual(t, "", v1.GetAsInt("epoch"), 1)
	XEqual(t, "", v1.GetAsString("createdat"), v1Created, NOMASK_TS)
	XEqual(t, "", v1.GetAsString("modifiedat"), v1Modified, NOMASK_TS)

	v1.DeleteSetNextVersion("")
	XNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)
	d1.Refresh(registry.FOR_WRITE)

	XEqual(t, "", reg.GetAsInt("epoch"), 2)
	XEqual(t, "", reg.GetAsString("createdat"), regCreated, NOMASK_TS)
	XEqual(t, "", reg.GetAsString("modifiedat"), regModified, NOMASK_TS)

	XEqual(t, "", d1.GetAsInt("epoch"), 3)
	XEqual(t, "", d1.GetAsString("createdat"), d1Created, NOMASK_TS)
	XCheckGreater(t, "", d1.GetAsString("modifiedat"), d1Modified)

	d1Modified = d1.GetAsString("modifiedat")

	f2.Delete()
	reg.Refresh(registry.FOR_WRITE)
	d1.Refresh(registry.FOR_WRITE)

	XEqual(t, "", reg.GetAsInt("epoch"), 2)
	XEqual(t, "", reg.GetAsString("createdat"), regCreated, NOMASK_TS)
	XEqual(t, "", reg.GetAsString("modifiedat"), regModified, NOMASK_TS)

	XEqual(t, "", d1.GetAsInt("epoch"), 4)
	XEqual(t, "", d1.GetAsString("createdat"), d1Created, NOMASK_TS)
	XCheckGreater(t, "", d1.GetAsString("modifiedat"), d1Modified)

	d1.Delete()
	XNoErr(t, reg.SaveAllAndCommit())
	reg.Refresh(registry.FOR_WRITE)

	XEqual(t, "", reg.GetAsInt("epoch"), 3)
	XEqual(t, "", reg.GetAsString("createdat"), regCreated, NOMASK_TS)
	XCheckGreater(t, "", reg.GetAsString("modifiedat"), regModified)

	XHTTP(t, reg, "GET", "/?inline", ``, 200, `{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XHTTP(t, reg, "GET", "/?inline", ``, 200, `{
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
          "filebase64": "",

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
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
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
              "ancestor": "v1",
              "filebase64": ""
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
              "ancestor": "v1",
              "filebase64": ""
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - int invalid",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
  "myint": 4
}`,
		Code: 400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myint\" for \"/\" is not valid: value (4) must be one of the enum values: 1, 2, 3.",
  "subject": "/",
  "args": {
    "error_detail": "value (4) must be one of the enum values: 1, 2, 3",
    "name": "myint"
  },
  "source": ":registry:entity:2589"
}
`,
	})

	attr = reg.Model.Attributes["myint"]
	attr.SetStrict(false)
	reg.Model.VerifyAndSave()

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XNoErr(t, err)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{"compatibility":"none"}`,
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
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/meta", `{"compatibility":null}`,
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
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/meta",
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
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/meta",
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
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
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
	XCheckErr(t, err, "")
	XNoErr(t, reg.SaveModel())

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "myobj",
		Type: OBJECT,
	})
	// Test empty obj and name conflict with IfValue above
	XCheckErr(t, err,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: duplicate attribute name (myobj) at: myint.ifvalues.10.",
  "subject": "/model",
  "args": {
    "error_detail": "duplicate attribute name (myobj) at: myint.ifvalues.10"
  },
  "source": ":registry:shared_model:2798"
}`)
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
	XCheckErr(t, err, "")
	XNoErr(t, reg.SaveModel())

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "badone",
		Type: INTEGER,
		IfValues: registry.IfValues{
			"": &registry.IfValue{},
		},
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"badone\" has an empty ifvalues key.",
  "subject": "/model",
  "args": {
    "error_detail": "\"badone\" has an empty ifvalues key"
  },
  "source": ":registry:shared_model:3019"
}`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "badone",
		Type: INTEGER,
		IfValues: registry.IfValues{
			"^6": &registry.IfValue{},
		},
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"badone\" has an ifvalues key that starts with \"^\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"badone\" has an ifvalues key that starts with \"^\""
  },
  "source": ":registry:shared_model:3026"
}`)

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - verify ext isn't allowed",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10,
	     "myext": 5.5
	   }`,
		Code: 400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (myext) was specified for \"/\".",
  "subject": "/",
  "args": {
    "name": "myext"
  },
  "source": ":registry:entity:2203"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - required mystr",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 20
	   }`,
		Code: 400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/\" are missing: mystr.",
  "subject": "/",
  "args": {
    "list": "mystr"
  },
  "source": ":registry:entity:2149"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - ifvalue - myext isn't allow any more",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint": 10,
	     "mystr": "hello",
	     "myext": 5.5
	   }`,
		Code: 400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (myext) was specified for \"/\".",
  "subject": "/",
  "args": {
    "name": "myext"
  },
  "source": ":registry:entity:2203"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XCheckErr(t, err, "")

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - nested ifValues - 3",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint5": 1,
	     "myint6": 2
	   }`,
		Code: 400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/\" are missing: myint7.",
  "subject": "/",
  "args": {
    "list": "myint7"
  },
  "source": ":registry:entity:2149"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - nested ifValues - 4",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "myint5": 1,
	     "myint6": 1,
		 "myint7": 5
	   }`,
		Code: 400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (myint7) was specified for \"/\".",
  "subject": "/",
  "args": {
    "name": "myint7"
  },
  "source": ":registry:entity:2203"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	// Test case insensitive IfValues
	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name: "mystr7",
		Type: STRING,
		IfValues: registry.IfValues{
			"AbC": &registry.IfValue{
				SiblingAttributes: registry.Attributes{
					"mystr8": &registry.Attribute{
						Name: "mystr8",
						Type: STRING,
					},
				},
			},
		},
	})
	XCheckErr(t, err, "")

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT reg - case insensitive",
		URL:    "",
		Method: "PUT",
		ReqBody: `{
	     "mystr7": "aBc",
         "mystr8": "hello"
	   }`,
		Code: 200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPIfValues",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 10,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "mystr7": "aBc",
  "mystr8": "hello"
}
`,
	})
}

func TestHTTPResources(t *testing.T) {
	reg := NewRegistry("TestHTTPResources")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, err)
	rm, err := gm.AddResourceModelSimple("files", "file")
	XNoErr(t, err)

	XNoErr(t, reg.SaveModel())

	/*
		_, err := rm.AddAttribute(&registry.Attribute{
			Name: "files",
			Type: INTEGER,
		})
		XCheckErr(t, err, "Attribute name is reserved: files")
	*/

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "file",
		Type: INTEGER,
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: attribute name is reserved: file.",
  "subject": "/model",
  "args": {
    "error_detail": "attribute name is reserved: file"
  },
  "source": ":registry:shared_model:1456"
}`)

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "filebase64",
		Type: INTEGER,
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: attribute name is reserved: filebase64.",
  "subject": "/model",
  "args": {
    "error_detail": "attribute name is reserved: filebase64"
  },
  "source": ":registry:shared_model:1456"
}`)

	_, err = rm.AddAttribute(&registry.Attribute{
		Name: "fileproxyurl",
		Type: INTEGER,
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: attribute name is reserved: fileproxyurl.",
  "subject": "/model",
  "args": {
    "error_detail": "attribute name is reserved: fileproxyurl"
  },
  "source": ":registry:shared_model:1456"
}`)

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
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: duplicate attribute name (file) at: resources.files.mystring.ifvalues.foo.",
  "subject": "/model",
  "args": {
    "error_detail": "duplicate attribute name (file) at: resources.files.mystring.ifvalues.foo"
  },
  "source": ":registry:shared_model:2796"
}`)
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
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: duplicate attribute name (xxx) at: resources.files.mystring.ifvalues.foo.xxx.ifvalues.5.",
  "subject": "/model",
  "args": {
    "error_detail": "duplicate attribute name (xxx) at: resources.files.mystring.ifvalues.foo.xxx.ifvalues.5"
  },
  "source": ":registry:shared_model:2796"
}`)

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
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: duplicate attribute name (file) at: resources.files.mystring.ifvalues.foo.xxx.ifvalues.5.",
  "subject": "/model",
  "args": {
    "error_detail": "duplicate attribute name (file) at: resources.files.mystring.ifvalues.foo.xxx.ifvalues.5"
  },
  "source": ":registry:shared_model:2796"
}`)
	reg.LoadModel()

	// "file" is ok this time because HasDocument=false
	rm = rm.Refresh()
	rm.SetHasDocument(false)
	XNoErr(t, reg.Model.VerifyAndSave())
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
	XNoErr(t, err)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/vx", `{"versionid":"x"}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"versionid\" value (x) for \"/dirs/d1/files/f1/versions/vx\" needs to be \"vx\".",
  "subject": "/dirs/d1/files/f1/versions/vx",
  "args": {
    "expected_id": "vx",
    "invalid_id": "x",
    "singular": "version"
  },
  "source": ":registry:entity:879"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", "{}", 201, `{
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

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
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

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "file": "fff",
  "mystring": "hello",
  "object": {}
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (file) was specified for \"/dirs/d1/files/f1/versions/v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "name": "file"
  },
  "source": ":registry:entity:2203"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "filebase64": "fff",
  "mystring": "hello",
  "object": {}
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (filebase64) was specified for \"/dirs/d1/files/f1/versions/v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "name": "filebase64"
  },
  "source": ":registry:entity:2203"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "fileurl": "fff",
  "mystring": "hello",
  "object": {}
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (fileurl) was specified for \"/dirs/d1/files/f1/versions/v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "name": "fileurl"
  },
  "source": ":registry:entity:2203"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "fileproxyurl": "fff",
  "mystring": "hello",
  "object": {}
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (fileproxyurl) was specified for \"/dirs/d1/files/f1/versions/v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "name": "fileproxyurl"
  },
  "source": ":registry:entity:2203"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "file": "fff",
  "fileurl": "fff",
  "filebase64": "fff",
  "fileproxyurl": "fff",
  "mystring": "hello",
  "object": {}
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (file) was specified for \"/dirs/d1/files/f1/versions/v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "name": "file"
  },
  "source": ":registry:entity:2203"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
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

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
  "file": "fff",
  "mystring": "foo",
  "object": {
    "objstr": "ooo",
    "objint": 5
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (object.objint) was specified for \"/dirs/d1/files/f1/versions/v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "name": "object.objint"
  },
  "source": ":registry:entity:2203"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
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

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
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

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-mystr-foo: hello",
		},
		ReqBody: `hello`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mystr\" for \"/dirs/d1/files/f1/versions/1\" is not valid: must be a string.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "error_detail": "must be a string",
    "name": "mystr"
  },
  "source": ":registry:entity:2522"
}
`,
		ResHeaders: []string{},
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file f1",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-mystr-foo-bar: hello",
		},
		ReqBody: `hello`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"mystr\" for \"/dirs/d1/files/f1/versions/1\" is not valid: must be a string.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "error_detail": "must be a string",
    "name": "mystr"
  },
  "source": ":registry:entity:2522"
}
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/2 - setdefault=2 - diff server",
		URL:         "/dirs/d1/files/f1/versions/2?setdefaultversionid=2",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#setdefaultversionid_not_allowed",
  "title": "Processing \"/dirs/d1/files/f1\", the \"setdefaultversionid\" flag is not allowed to be specified for entities of type \"file\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "singular": "file"
  },
  "source": ":registry:httpStuff:2597"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:        "PUT file f1/1 - setdefault=1 - match server",
		URL:         "/dirs/d1/files/f1/versions/1?setdefaultversionid=1",
		Method:      "PUT",
		ReqHeaders:  []string{},
		ReqBody:     `hello`,
		Code:        400,
		HeaderMasks: []string{},
		ResHeaders:  []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#setdefaultversionid_not_allowed",
  "title": "Processing \"/dirs/d1/files/f1\", the \"setdefaultversionid\" flag is not allowed to be specified for entities of type \"file\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "singular": "file"
  },
  "source": ":registry:httpStuff:2597"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1?setdefault= not allowed",
		URL:     "/dirs/d1/files/f1?setdefaultversionid",
		Method:  "POST",
		ReqBody: "{}",
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#setdefaultversionid_not_allowed",
  "title": "Processing \"/dirs/d1/files/f1\", the \"setdefaultversionid\" flag is not allowed to be specified for entities of type \"file\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "singular": "file"
  },
  "source": ":registry:httpStuff:2597"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1$details?setdefault= not allowed",
		URL:     "/dirs/d1/files/f1$details?setdefaultversionid",
		Method:  "POST",
		ReqBody: "{}",
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#setdefaultversionid_not_allowed",
  "title": "Processing \"/dirs/d1/files/f1\", the \"setdefaultversionid\" flag is not allowed to be specified for entities of type \"file\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "singular": "file"
  },
  "source": ":registry:httpStuff:2597"
}
`,
	})

	// Enable client-side setting
	rm.SetSetDefaultSticky(true)

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file f1?setdefault - empty",
		URL:    "/dirs/d1/files/f1?setdefaultversionid",
		Method: "POST",
		Code:   400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_defaultversionid",
  "title": "An error was found in the \"defaultversionid\" value specified (\"\"): value must not be empty.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "error_detail": "value must not be empty",
    "value": "\"\""
  },
  "source": ":registry:httpStuff:2604"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file f1?setdefault= - empty",
		URL:    "/dirs/d1/files/f1?setdefaultversionid=",
		Method: "POST",
		Code:   400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_defaultversionid",
  "title": "An error was found in the \"defaultversionid\" value specified (\"\"): value must not be empty.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "error_detail": "value must not be empty",
    "value": "\"\""
  },
  "source": ":registry:httpStuff:2604"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1$details?setdefault - empty",
		URL:     "/dirs/d1/files/f1$details?setdefaultversionid",
		Method:  "POST",
		ReqBody: "{}",
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_defaultversionid",
  "title": "An error was found in the \"defaultversionid\" value specified (\"\"): value must not be empty.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "error_detail": "value must not be empty",
    "value": "\"\""
  },
  "source": ":registry:httpStuff:2604"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "POST file f1$details?setdefault= - empty",
		URL:     "/dirs/d1/files/f1$details?setdefaultversionid=",
		Method:  "POST",
		ReqBody: "{}",
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_defaultversionid",
  "title": "An error was found in the \"defaultversionid\" value specified (\"\"): value must not be empty.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "error_detail": "value must not be empty",
    "value": "\"\""
  },
  "source": ":registry:httpStuff:2604"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 6,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/2$details",
  "defaultversionsticky": true
}
`)

	// errors
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad group type",
		URL:        "/badgroup/d1/files/f1$details?setdefaultversionid=3",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-versionid: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/badgroup) cannot be found.",
  "detail": "Unknown Group type: badgroup.",
  "subject": "/badgroup",
  "source": ":registry:info:562"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad header",
		URL:        "/dirs/d1/files/f1$details?setdefaultversionid=3",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-versionid: bogus`},
		ReqBody:    "{}",
		Code:       400,
		ResHeaders: []string{"Content-Type: application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xregistry_header",
  "title": "xRegistry HTTP header \"xregistry-versionid\" is not allowed on this request: including \"xRegistry\" HTTP headers when \"$details\" is used is not allowed.",
  "subject": "/dirs/d1/files/f1$details",
  "args": {
    "error_detail": "including \"xRegistry\" HTTP headers when \"$details\" is used is not allowed",
    "name": "xregistry-versionid"
  },
  "source": ":registry:httpStuff:3145"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad group",
		URL:        "/dirs/dx/files/f11?setdefaultversionid=6",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-versionid: bogus`},
		Code:       400,
		ResHeaders: []string{"Content-Type: application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/dx/files/f11\", the \"version\" with a \"versionid\" value of \"6\" cannot be found.",
  "subject": "/dirs/dx/files/f11",
  "args": {
    "id": "6",
    "singular": "version"
  },
  "source": ":registry:httpStuff:2628"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad resource type",
		URL:        "/dirs/d1/badfiles/f1$details?setdefaultversionid=3",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-versionid: bogus`},
		Code:       404,
		ResHeaders: []string{"Content-Type: application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/badfiles) cannot be found.",
  "detail": "Unknown Resource type: badfiles.",
  "subject": "/dirs/d1/badfiles",
  "source": ":registry:info:595"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad header",
		URL:        "/dirs/d1/files/f1$details?setdefaultversionid=3",
		Method:     "POST",
		ReqHeaders: []string{`xRegistry-versionid: bogus`},
		ReqBody:    "{}",
		Code:       400,
		ResHeaders: []string{"Content-Type: application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xregistry_header",
  "title": "xRegistry HTTP header \"xregistry-versionid\" is not allowed on this request: including \"xRegistry\" HTTP headers when \"$details\" is used is not allowed.",
  "subject": "/dirs/d1/files/f1$details",
  "args": {
    "error_detail": "including \"xRegistry\" HTTP headers when \"$details\" is used is not allowed",
    "name": "xregistry-versionid"
  },
  "source": ":registry:httpStuff:3145"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST setdefault bad version",
		URL:        "/dirs/d1/files/f1$details?setdefaultversionid=6",
		Method:     "POST",
		ReqBody:    "{}",
		Code:       400,
		ResHeaders: []string{"Content-Type: application/json; charset=utf-8"},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f1\", the \"version\" with a \"versionid\" value of \"6\" cannot be found.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "id": "6",
    "singular": "version"
  },
  "source": ":registry:httpStuff:2628"
}
`,
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
	XHTTP(t, reg, "DELETE", "/", "", 405, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#action_not_supported",
  "title": "The specified action (DELETE) is not supported for: /.",
  "subject": "/",
  "args": {
    "action": "DELETE"
  },
  "source": ":registry:httpStuff:2640"
}
`)

	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d2",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d2":{}}`,
		Code:    204,
		ResBody: ``,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d2",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{}`, // should be a no-op, not delete everything
		Code:    204,
		ResBody: ``,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XHTTP(t, reg, "DELETE", "/dirs/d3?epoch=2x", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"epoch\" for \"/dirs/d3\" is not valid: value (2x) must be a uinteger.",
  "subject": "/dirs/d3",
  "args": {
    "error_detail": "value (2x) must be a uinteger",
    "name": "epoch"
  },
  "source": ":registry:httpStuff:2650"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d3?epoch=2", "", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (2) for \"/dirs/d3\" does not match its current value (1).",
  "subject": "/dirs/d3",
  "args": {
    "bad_epoch": "2",
    "epoch": "1"
  },
  "source": ":registry:httpStuff:2686"
}
`)

	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d3 err",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d3": {"epoch":2}}`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (2) for \"/dirs/d3\" does not match its current value (2).",
  "subject": "/dirs/d3",
  "args": {
    "bad_epoch": "2",
    "epoch": "2"
  },
  "source": ":registry:httpStuff:2871"
}
`,
	})

	// TODO add a delete of 2 with bad epoch in 2nd one and verify that
	// the first one isn't deleted due to the transaction rollback

	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d3",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d3":{"dirid": "xx", "epoch":1}}`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"dirid\" value (xx) for \"/dirs/d3\" needs to be \"d3\".",
  "subject": "/dirs/d3",
  "args": {
    "expected_id": "d3",
    "invalid_id": "xx",
    "singular": "dir"
  },
  "source": ":registry:httpStuff:2879"
}
`,
	})

	// Make sure we ignore random attributes too
	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d3",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d3":{"dirid": "d3", "epoch":1, "foo": "bar"}}`,
		Code:    204,
		ResBody: ``,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d3 - already gone",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d3":{"epoch":1}}`,
		Code:    204,
		ResBody: ``,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d4",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d4":{"epoch":"1x"}}`,
		Code:    400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"epoch\" for \"/dirs/d4\" is not valid: must be a uinteger.",
  "subject": "/dirs/d4",
  "args": {
    "error_detail": "must be a uinteger",
    "name": "epoch"
  },
  "source": ":registry:httpStuff:2866"
}
`,
	})
	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - dx",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"dx":{"epoch":1}}`,
		Code:    204,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:    "DELETE /dirs - d4",
		URL:     "/dirs",
		Method:  "DELETE",
		ReqBody: `{"d4":{"epoch":1}}`,
		Code:    204,
		ResBody: ``,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XHTTP(t, reg, "DELETE", "/dirs/d5?epoch=1", "", 204, "")
	XCheckHTTP(t, reg, &HTTPTest{
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

	XHTTP(t, reg, "DELETE", "/dirs", "", 204, "")
	XHTTP(t, reg, "DELETE", "/dirs", "", 204, "")
	XHTTP(t, reg, "DELETE", "/dirsx", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirsx) cannot be found.",
  "detail": "Unknown Group type: dirsx.",
  "subject": "/dirsx",
  "source": ":registry:info:562"
}
`)
	XHTTP(t, reg, "GET", "/dirs", "", 200, "{}\n")

	// Reset
	reg.AddGroup("dirs", "d1")
	reg.AddGroup("dirs", "d2")
	reg.AddGroup("dirs", "d3")
	reg.AddGroup("dirs", "d4")

	// DELETE /GROUPs/GID
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	XCheckHTTP(t, reg, &HTTPTest{
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

	XHTTP(t, reg, "DELETE", "/dirs/d3", "", 204, ``)
	XHTTP(t, reg, "DELETE", "/dirs/dx", "", 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/dx) cannot be found.",
  "subject": "/dirs/dx",
  "source": ":registry:httpStuff:2679"
}
`)

	XCheckHTTP(t, reg, &HTTPTest{
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

	XHTTP(t, reg, "DELETE", "/dirs", "", 204, "")
	XHTTP(t, reg, "GET", "/dirs", "", 200, "{}\n")

	// Reset
	d1, _ := reg.AddGroup("dirs", "d1")
	d1.AddResource("files", "f1", "v1.1")
	d1.AddResource("files", "f2", "v2.1")
	d1.AddResource("files", "f3", "v3.1")
	d1.AddResource("files", "f4", "v4.1")
	d1.AddResource("files", "f5", "v5.1")
	d1.AddResource("files", "f6", "v6.1")

	// DELETE Resources
	XCheckHTTP(t, reg, &HTTPTest{
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
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1", "", 204, "")
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/fx", "", 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/fx) cannot be found.",
  "subject": "/dirs/d1/files/fx",
  "source": ":registry:httpStuff:2723"
}
`)

	// DELETE /dirs/d1/files/f1?epoch=...
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f3?epoch=2x", "", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"epoch\" for \"/dirs/d1/files/f3\" is not valid: value (2x) must be a uinteger.",
  "subject": "/dirs/d1/files/f3",
  "args": {
    "error_detail": "value (2x) must be a uinteger",
    "name": "epoch"
  },
  "source": ":registry:httpStuff:2650"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f3?epoch=2", "", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (2) for \"/dirs/d1/files/f3/meta\" does not match its current value (1).",
  "subject": "/dirs/d1/files/f3/meta",
  "args": {
    "bad_epoch": "2",
    "epoch": "1"
  },
  "source": ":registry:httpStuff:2735"
}
`)

	// Bump epoch of f3
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f3/meta", "{}", 200, `{
  "fileid": "f3",
  "self": "http://localhost:8181/dirs/d1/files/f3/meta",
  "xid": "/dirs/d1/files/f3/meta",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v3.1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f3/versions/v3.1$details",
  "defaultversionsticky": false
}
`)

	// Bump epoch of f2
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2/meta", "{}", 200, `{
  "fileid": "f2",
  "self": "http://localhost:8181/dirs/d1/files/f2/meta",
  "xid": "/dirs/d1/files/f2/meta",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v2.1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/v2.1$details",
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

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f3?epoch=2", "", 204, "")

	// DELETE - testing ids in body
	_, err := d1.AddResource("files", "f3", "v1")
	XNoErr(t, err)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files", `{"f3":{"fileid":"fx"}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"fileid\" value (fx) for \"/dirs/d1/files/f3\" needs to be \"f3\".",
  "subject": "/dirs/d1/files/f3",
  "args": {
    "expected_id": "f3",
    "invalid_id": "fx",
    "singular": "file"
  },
  "source": ":registry:httpStuff:2981"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files", `{"f3":{"meta":{"fileid":"fx"}}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"fileid\" value (fx) for \"/dirs/d1/files/f3\" needs to be \"f3\".",
  "subject": "/dirs/d1/files/f3",
  "args": {
    "expected_id": "f3",
    "invalid_id": "fx",
    "singular": "file"
  },
  "source": ":registry:httpStuff:2955"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files", `{"f3":{"epoch":"2"}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#misplaced_epoch",
  "title": "The specified \"epoch\" value for \"/dirs/d1/files/f3\" needs to be within a \"meta\" entity.",
  "subject": "/dirs/d1/files/f3",
  "source": ":registry:httpStuff:2976"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files", `{"f3":{"fileid":"f3"}}`,
		204, ``)

	// DELETE /dirs/d1/files/f3 - bad epoch in body
	XHTTP(t, reg, "DELETE", "/dirs/d1/files",
		`{"f2":{"meta":{"epoch":"1x"}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"epoch\" for \"/dirs/d1/files/f2/meta\" is not valid: must be a uinteger.",
  "subject": "/dirs/d1/files/f2/meta",
  "args": {
    "error_detail": "must be a uinteger",
    "name": "epoch"
  },
  "source": ":registry:httpStuff:2964"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files",
		`{"f2":{"meta":{"epoch":4}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (4) for \"/dirs/d1/files/f2/meta\" does not match its current value (2).",
  "subject": "/dirs/d1/files/f2/meta",
  "args": {
    "bad_epoch": "4",
    "epoch": "2"
  },
  "source": ":registry:httpStuff:2969"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files",
		`{"f2":{"epoch":99,"meta":{"epoch":2}}}`, 204, "") // ignore top 'epoch'
	XHTTP(t, reg, "DELETE", "/dirs/d1/files",
		`{"fx":{"meta":{"epoch":1}}}`, 204, "")

	XHTTP(t, reg, "DELETE", "/dirs/d1/files",
		`{"f2":{},"f4":{"meta":{"epoch":3}}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (3) for \"/dirs/d1/files/f4/meta\" does not match its current value (1).",
  "subject": "/dirs/d1/files/f4/meta",
  "args": {
    "bad_epoch": "3",
    "epoch": "1"
  },
  "source": ":registry:httpStuff:2969"
}
`)
	// Make sure we ignore random attributes too
	XHTTP(t, reg, "DELETE", "/dirs/d1/files",
		`{"f4":{},"f5":{"meta":{"epoch":1,"foo":"bar"}, "foo":"bar"}}`,
		204, "")

	XHTTP(t, reg, "DELETE", "/dirs/d1/files", `{}`, 204, "") // no-op

	XCheckHTTP(t, reg, &HTTPTest{
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

	XHTTP(t, reg, "DELETE", "/dirs/d1/files", ``, 204, "")

	XCheckHTTP(t, reg, &HTTPTest{
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
	XNoErr(t, err)
	f1.AddVersion("v2")
	f1.AddVersion("v3")
	// v4, _ := f1.AddVersion("v4")
	f1.AddVersion("v4")
	v5, _ := f1.AddVersion("v5")
	XNoErr(t, f1.SetDefault(v5))
	f1.AddVersion("v6")
	f1.AddVersion("v7")
	f1.AddVersion("v8")
	f1.AddVersion("v9")
	f1.AddVersion("v10")

	// t.Logf("v4.old: %s", ToJSON(v4.Object))
	// t.Logf("v4.new: %s", ToJSON(v4.NewObject))

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200,
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
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/vx", "", 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/versions/vx) cannot be found.",
  "subject": "/dirs/d1/files/f1/versions/vx",
  "source": ":registry:httpStuff:2775"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v1", "", 204, "")
	// v2's epoch/modifiedat should change due to changing its ancestor

	// DELETE /dirs/d1/files/f1?epoch=...
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2?epoch=2x", "", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"epoch\" for \"/dirs/d1/files/f1/versions/v2\" is not valid: value (2x) must be a uinteger.",
  "subject": "/dirs/d1/files/f1/versions/v2",
  "args": {
    "error_detail": "value (2x) must be a uinteger",
    "name": "epoch"
  },
  "source": ":registry:httpStuff:2650"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2?epoch=3", "", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (3) for \"/dirs/d1/files/f1/versions/v2\" does not match its current value (2).",
  "subject": "/dirs/d1/files/f1/versions/v2",
  "args": {
    "bad_epoch": "3",
    "epoch": "2"
  },
  "source": ":registry:httpStuff:2782"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2?epoch=2", "", 204, "")

	// DELETE /dirs/d1/files/f1/versions/v4 - bad epoch in body
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v4":{"epoch":"1x"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"epoch\" for \"/dirs/d1/files/f1/versions/v4\" is not valid: value must be a uinteger.",
  "subject": "/dirs/d1/files/f1/versions/v4",
  "args": {
    "error_detail": "value must be a uinteger",
    "name": "epoch"
  },
  "source": ":registry:httpStuff:3050"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v4":{"epoch":2}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (2) for \"/dirs/d1/files/f1/versions/v4\" does not match its current value (2).",
  "subject": "/dirs/d1/files/f1/versions/v4",
  "args": {
    "bad_epoch": "2",
    "epoch": "2"
  },
  "source": ":registry:httpStuff:3055"
}
`)

	// DELETE - bad IDs
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v4":{"fileid":2}}`, 204, "") // ignore fileid
	f1.AddVersion("v4")
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v4":{"versionid":2}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"versionid\" value (2) for \"/dirs/d1/files/f1/versions/v4\" needs to be \"v4\".",
  "subject": "/dirs/d1/files/f1/versions/v4",
  "args": {
    "expected_id": "v4",
    "invalid_id": "2",
    "singular": "version"
  },
  "source": ":registry:httpStuff:3065"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v4":{"fileid":"fx","versionid":"v4"}}`, 204, "") // ignore fileid
	f1.AddVersion("v4")

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `{"v4":{"epoch":1}}`, 204, "")
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `{"v4":{"epoch":1}}`, 204, "")
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `{"vx":{"epoch":1}}`, 204, "")

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v6":{},"v7":{"epoch":3}}`, // v6 will still be around
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (3) for \"/dirs/d1/files/f1/versions/v7\" does not match its current value (3).",
  "subject": "/dirs/d1/files/f1/versions/v7",
  "args": {
    "bad_epoch": "3",
    "epoch": "3"
  },
  "source": ":registry:httpStuff:3055"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions",
		`{"v7":{},"v8":{"epoch":1}}`,
		204, "")

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `{}`, 204, "") // No-op

	// Make sure we have some left, and default is still v5
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", "", 200, `{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v5",
  "filebase64": "",

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
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5$details",
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
      "ancestor": "v9",
      "filebase64": ""
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
      "ancestor": "v3",
      "filebase64": ""
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
      "ancestor": "v5",
      "filebase64": ""
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
      "ancestor": "v5",
      "filebase64": ""
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
      "ancestor": "v9",
      "filebase64": ""
    }
  },
  "versionscount": 5
}
`)

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v5?setdefaultversionid=v3",
		``, 204, "")

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setdefaultversionid=v9",
		``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Can't set \"defaultversionid\" to a Version that is being deleted.",
  "subject": "/dirs/d1/files/f1/versions/v9",
  "args": {
    "error_detail": "Can't set \"defaultversionid\" to a Version that is being deleted"
  },
  "source": ":registry:version:63"
}
`)

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setdefaultversionid=vx",
		``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f1\", the \"version\" with a \"versionid\" value of \"vx\" cannot be found.",
  "detail": "Can't find next default Version \"vx\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "id": "vx",
    "singular": "version"
  },
  "source": ":registry:version:117"
}
`)

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setdefaultversionid=v3",
		``, 204, "")
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v9?setdefaultversionid=vx",
		``, 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/versions/v9) cannot be found.",
  "subject": "/dirs/d1/files/f1/versions/v9",
  "source": ":registry:httpStuff:2775"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", "", 200, `{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v3",
  "filebase64": "",

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
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3$details",
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
      "ancestor": "v10",
      "filebase64": ""
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
      "ancestor": "v3",
      "filebase64": ""
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
      "ancestor": "v6",
      "filebase64": ""
    }
  },
  "versionscount": 3
}
`)

	f1.AddVersion("v1")
	// bad next
	XHTTP(t, reg, "DELETE",
		"/dirs/d1/files/f1/versions?setdefaultversionid=vx", `{"v6":{}}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f1\", the \"version\" with a \"versionid\" value of \"vx\" cannot be found.",
  "detail": "Can't find next default Version \"vx\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "id": "vx",
    "singular": "version"
  },
  "source": ":registry:version:117"
}
`)
	// next = being deleted
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setdefaultversionid=v6",
		`{"v6":{}}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Can't set \"defaultversionid\" to a Version that is being deleted.",
  "subject": "/dirs/d1/files/f1/versions/v6",
  "args": {
    "error_detail": "Can't set \"defaultversionid\" to a Version that is being deleted"
  },
  "source": ":registry:version:63"
}
`)

	// delete non-default, change default
	XHTTP(t, reg, "DELETE",
		"/dirs/d1/files/f1/versions?setdefaultversionid=v10", `{"v6":{}}`, 204, "")
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", "", 200, `{
  "fileid": "f1",
  "versionid": "v10",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v10",
  "filebase64": "",

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
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v10$details",
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
      "ancestor": "v1",
      "filebase64": ""
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
      "ancestor": "v10",
      "filebase64": ""
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
      "ancestor": "v3",
      "filebase64": ""
    }
  },
  "versionscount": 3
}
`)

	// delete non-default, default not move
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", `{"v3":{}}`, 204, "")
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", "", 200, `{
  "fileid": "f1",
  "versionid": "v10",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v10",
  "filebase64": "",

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
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v10$details",
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
      "ancestor": "v1",
      "filebase64": ""
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
      "ancestor": "v10",
      "filebase64": ""
    }
  },
  "versionscount": 2
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions?setdefaultversionid=v1", `{}`, 204, "")
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", "", 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1",
  "filebase64": "",

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
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
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
      "ancestor": "v1",
      "filebase64": ""
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
      "ancestor": "v10",
      "filebase64": ""
    }
  },
  "versionscount": 2
}
`)

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions", ``, 204, "")
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions", "", 404,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1) cannot be found.",
  "subject": "/dirs/d1/files/f1",
  "source": ":registry:httpStuff:1746"
}
`)

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
	XNoErr(t, err)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	_, err = gm.AddAttribute(&registry.Attribute{
		Name:     "req2",
		Type:     STRING,
		Required: true,
	})
	XNoErr(t, err)

	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	_, err = rm.AddAttribute(&registry.Attribute{
		Name:     "req3",
		Type:     STRING,
		Required: true,
	})
	XNoErr(t, err)

	// Must commit before we call Set below otherwise the transaction will
	// be rolled back
	reg.SaveAllAndCommit()

	// Registry itself
	err = reg.SetSave("description", "testing")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/\" are missing: req1.",
  "subject": "/",
  "args": {
    "list": "req1"
  },
  "source": ":registry:entity:2149"
}`)

	XNoErr(t, reg.JustSet("req1", "testing1"))
	XNoErr(t, reg.SetSave("description", "testing"))

	XHTTP(t, reg, "GET", "/", "", 200, `{
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
	XHTTP(t, reg, "PUT", "/dirs/d1", `{"description": "testing"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/dirs/d1\" are missing: req2.",
  "subject": "/dirs/d1",
  "args": {
    "list": "req2"
  },
  "source": ":registry:entity:2149"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1", `{
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
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{"description": "testing"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/dirs/d1/files/f1/versions/1\" are missing: req3.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "list": "req3"
  },
  "source": ":registry:entity:2149"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-description: testingdesc",
		},

		Code:       400,
		ResHeaders: []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/dirs/d1/files/f2/versions/1\" are missing: req3.",
  "subject": "/dirs/d1/files/f2/versions/1",
  "args": {
    "list": "req3"
  },
  "source": ":registry:entity:2149"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2$details",
		Method: "PUT",
		ReqBody: `{
  "description": "testingdesc2"
}`,

		Code:       400,
		ResHeaders: []string{},
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/dirs/d1/files/f2/versions/1\" are missing: req3.",
  "subject": "/dirs/d1/files/f2/versions/1",
  "args": {
    "list": "req3"
  },
  "source": ":registry:entity:2149"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
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

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2/versions/1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-versionid: 1",
			"xRegistry-description: desctesting",
			"xRegistry-req3: null",
		},

		Code: 400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/dirs/d1/files/f2/versions/1\" are missing: req3.",
  "subject": "/dirs/d1/files/f2/versions/1",
  "args": {
    "list": "req3"
  },
  "source": ":registry:entity:2149"
}
`,
	})

	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/f2/versions/1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-versionid: 1",
			"xRegistry-description: desctesting",
			"xRegistry-req3: null",
		},

		Code: 400,
		ResBody: `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/dirs/d1/files/f2/versions/1\" are missing: req3.",
  "subject": "/dirs/d1/files/f2/versions/1",
  "args": {
    "list": "req3"
  },
  "source": ":registry:entity:2149"
}
`,
	})
}
