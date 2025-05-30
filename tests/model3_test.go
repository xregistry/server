package tests

import (
	"testing"
	// "github.com/xregistry/server/registry"
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
