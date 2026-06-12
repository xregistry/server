package tests

import (
	"testing"

	. "github.com/xregistry/server/common"
)

func TestHTTPOptions(t *testing.T) {
	reg := NewRegistry("TestHTTPOptions")
	defer PassDeleteReg(t, reg)

	// Test OPTIONS on root
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /",
		URL:    "/",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:GET, OPTIONS, PATCH, POST, PUT",
			"Access-Control-Allow-Methods:GET, OPTIONS, PATCH, POST, PUT",
		},
		ResBody: "",
	})

	// Test OPTIONS on /capabilities
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /capabilities",
		URL:    "/capabilities",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:GET, OPTIONS, PATCH, PUT",
			"Access-Control-Allow-Methods:GET, OPTIONS, PATCH, PUT",
		},
		ResBody: "",
	})

	// Test OPTIONS on /capabilitiesoffered (read-only)
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /capabilitiesoffered",
		URL:    "/capabilitiesoffered",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:GET, OPTIONS",
			"Access-Control-Allow-Methods:GET, OPTIONS",
		},
		ResBody: "",
	})

	// Test OPTIONS on /model (read-only)
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /model",
		URL:    "/model",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:GET, OPTIONS",
			"Access-Control-Allow-Methods:GET, OPTIONS",
		},
		ResBody: "",
	})

	// Test OPTIONS on /modelsource
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /modelsource",
		URL:    "/modelsource",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:GET, OPTIONS, PUT",
			"Access-Control-Allow-Methods:GET, OPTIONS, PUT",
		},
		ResBody: "",
	})

	// Test OPTIONS on /export (read-only)
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /export",
		URL:    "/export",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:GET, OPTIONS",
			"Access-Control-Allow-Methods:GET, OPTIONS",
		},
		ResBody: "",
	})
}

func TestHTTPOptionsWithGroups(t *testing.T) {
	reg := NewRegistry("TestHTTPOptionsWithGroups")
	defer PassDeleteReg(t, reg)

	// Add group and resource models
	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, false)
	XNoErr(t, err)

	// Create a group
	XNoErr(t, reg.SaveModel(true))
	_, err = reg.AddGroup("dirs", "d1")
	XNoErr(t, err)
	XNoErr(t, reg.SaveAllAndCommit())

	// Test OPTIONS on /dirs (collection)
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /dirs",
		URL:    "/dirs",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:DELETE, GET, OPTIONS, PATCH, POST",
			"Access-Control-Allow-Methods:DELETE, GET, OPTIONS, PATCH, POST",
		},
		ResBody: "",
	})

	// Test OPTIONS on /dirs/d1 (group entity)
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /dirs/d1",
		URL:    "/dirs/d1",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:DELETE, GET, OPTIONS, PATCH, POST, PUT",
			"Access-Control-Allow-Methods:DELETE, GET, OPTIONS, PATCH, POST, PUT",
		},
		ResBody: "",
	})

	// Test OPTIONS on /dirs/d1/files (resource collection)
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /dirs/d1/files",
		URL:    "/dirs/d1/files",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:DELETE, GET, OPTIONS, PATCH, POST",
			"Access-Control-Allow-Methods:DELETE, GET, OPTIONS, PATCH, POST",
		},
		ResBody: "",
	})
}

func TestHTTPOptionsWithResources(t *testing.T) {
	reg := NewRegistry("TestHTTPOptionsWithResources")
	defer PassDeleteReg(t, reg)

	// Add group and resource models
	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, false)
	XNoErr(t, err)

	// Create a group and resource
	XNoErr(t, reg.SaveModel(true))
	g, err := reg.AddGroup("dirs", "d1")
	XNoErr(t, err)
	_, err = g.AddResource("files", "f1", "v1")
	XNoErr(t, err)
	XNoErr(t, reg.SaveAllAndCommit())

	// Test OPTIONS on /dirs/d1/files/f1 (resource entity)
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /dirs/d1/files/f1",
		URL:    "/dirs/d1/files/f1",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:DELETE, GET, OPTIONS, PATCH, POST, PUT",
			"Access-Control-Allow-Methods:DELETE, GET, OPTIONS, PATCH, POST, PUT",
		},
		ResBody: "",
	})

	// Test OPTIONS on /dirs/d1/files/f1/meta
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /dirs/d1/files/f1/meta",
		URL:    "/dirs/d1/files/f1/meta",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:DELETE, GET, OPTIONS, PATCH, PUT",
			"Access-Control-Allow-Methods:DELETE, GET, OPTIONS, PATCH, PUT",
		},
		ResBody: "",
	})

	// Test OPTIONS on /dirs/d1/files/f1/versions (versions collection)
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /dirs/d1/files/f1/versions",
		URL:    "/dirs/d1/files/f1/versions",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:DELETE, GET, OPTIONS, PATCH, POST",
			"Access-Control-Allow-Methods:DELETE, GET, OPTIONS, PATCH, POST",
		},
		ResBody: "",
	})

	// Use "v1" as the version ID since that's what we created the resource with
	vID := "v1"

	// Test OPTIONS on /dirs/d1/files/f1/versions/v1 (version entity)
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /dirs/d1/files/f1/versions/" + vID,
		URL:    "/dirs/d1/files/f1/versions/" + vID,
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:DELETE, GET, OPTIONS, PATCH, PUT",
			"Access-Control-Allow-Methods:DELETE, GET, OPTIONS, PATCH, PUT",
		},
		ResBody: "",
	})
}

func TestHTTPOptionsWithCapabilities(t *testing.T) {
	reg := NewRegistry("TestHTTPOptionsWithCapabilities")
	defer PassDeleteReg(t, reg)

	// Add group and resource models
	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, err)
	_, err = gm.AddResourceModel("files", "file", 0, true, false)
	XNoErr(t, err)

	// Disable entities.mutable capability
	XNoErr(t, reg.SaveModel(true))
	XNoErr(t, reg.SetSave("#capabilities", `{
		"available": {
			"entities": {
				"mutable": false
			},
			"modelsource": {
				"mutable": true
			},
			"capabilities": {
				"mutable": true
			}
		}
	}`))
	XNoErr(t, reg.SaveAllAndCommit())

	// Test OPTIONS on root - should only have GET with entities immutable
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS / with immutable entities",
		URL:    "/",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:GET, OPTIONS",
			"Access-Control-Allow-Methods:GET, OPTIONS",
		},
		ResBody: "",
	})

	// Test OPTIONS on /capabilities - should still be mutable
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /capabilities with immutable entities",
		URL:    "/capabilities",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:GET, OPTIONS, PATCH, PUT",
			"Access-Control-Allow-Methods:GET, OPTIONS, PATCH, PUT",
		},
		ResBody: "",
	})

	// Make capabilities immutable too
	XNoErr(t, reg.SetSave("#capabilities", `{
		"available": {
			"capabilities": {
				"mutable": false
			},
			"entities": {
				"mutable": false
			},
			"modelsource": {
				"mutable": true
			}
		}
	}`))
	XNoErr(t, reg.SaveAllAndCommit())

	// Test OPTIONS on /capabilities - should now be read-only
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /capabilities with immutable capabilities",
		URL:    "/capabilities",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:GET, OPTIONS",
			"Access-Control-Allow-Methods:GET, OPTIONS",
		},
		ResBody: "",
	})

	// /modelsource should still be available since we didn't disable it
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "OPTIONS /modelsource (still available)",
		URL:    "/modelsource",
		Method: "OPTIONS",

		Code: 200,
		ResHeaders: []string{
			"Allow:GET, OPTIONS, PUT",
			"Access-Control-Allow-Methods:GET, OPTIONS, PUT",
		},
		ResBody: "",
	})
}
