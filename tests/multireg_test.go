package tests

import (
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestMultiReg(t *testing.T) {
	reg := NewRegistry("TestMultiReg")
	defer PassDeleteReg(t, reg)

	model1 := `{
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
}`
	XHTTP(t, reg, "PUT", "/modelsource", model1, 200, model1+"\n")

	reg2, err := registry.NewRegistry(nil, "reg2")
	defer PassDeleteReg(t, reg2)
	XNoErr(t, err)
	model2 := `{
  "groups": {
    "reg2_dirs": {
      "singular": "reg2_dir",
      "resources": {
        "reg2_files": {
          "singular": "reg2_file"
        }
      }
    }
  }
}`
	XHTTP(t, reg2, "PUT", "/"+registry.RegCollectionSegment+"/reg2/modelsource", model2, 200, model2+"\n")

	// reg
	XHTTP(t, reg, "GET", "/", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestMultiReg",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0
}
`)

	// reg2
	XHTTP(t, reg2, "GET", "/"+registry.RegCollectionSegment+"/reg2", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "reg2",
  "self": "http://localhost:8181/`+registry.RegCollectionSegment+`/reg2/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "reg2_dirsurl": "http://localhost:8181/`+registry.RegCollectionSegment+`/reg2/reg2_dirs",
  "reg2_dirscount": 0
}
`)

	XHTTP(t, reg2, "GET", "/"+registry.RegCollectionSegment+"/reg2/reg2_dirs", "", 200, "{}\n")

	XHTTP(t, reg2, "PUT", "/"+registry.RegCollectionSegment+"/reg2/reg2_dirs/d2", "{}", 201, `{
  "reg2_dirid": "d2",
  "self": "http://localhost:8181/`+registry.RegCollectionSegment+`/reg2/reg2_dirs/d2",
  "xid": "/reg2_dirs/d2",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "reg2_filesurl": "http://localhost:8181/`+registry.RegCollectionSegment+`/reg2/reg2_dirs/d2/reg2_files",
  "reg2_filescount": 0
}
`)

	XHTTP(t, reg2, "PUT", "/"+registry.RegCollectionSegment+"/reg2/reg2_dirs/d2/reg2_files/f2$details", "{}", 201, `{
  "reg2_fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/`+registry.RegCollectionSegment+`/reg2/reg2_dirs/d2/reg2_files/f2$details",
  "xid": "/reg2_dirs/d2/reg2_files/f2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/`+registry.RegCollectionSegment+`/reg2/reg2_dirs/d2/reg2_files/f2/meta",
  "versionsurl": "http://localhost:8181/`+registry.RegCollectionSegment+`/reg2/reg2_dirs/d2/reg2_files/f2/versions",
  "versionscount": 1
}
`)
}
