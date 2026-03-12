package tests

import (
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
	"testing"
	// log "github.com/duglin/dlog"
)

// Tests "format" and "compatibility"

func TestFormatSimple(t *testing.T) {
	reg := NewRegistry("TestFormat")
	defer PassDeleteReg(t, reg)

	gm, xErr := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, xErr)
	rm, xErr := gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, xErr)
	_, xErr = reg.AddGroup("dirs", "d1")
	XNoErr(t, xErr)

	rm.SetValidateCompatibility(true)

	XCheckErr(t, rm.GroupModel.Model.Verify(), `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Resource \"files\" must have \"validateformat\" set to \"true\" when \"validatecompatibility\" is \"true\".",
  "subject": "/model",
  "args": {
    "error_detail": "Resource \"files\" must have \"validateformat\" set to \"true\" when \"validatecompatibility\" is \"true\""
  },
  "source": "74c45abf9c72:registry:shared_model:2335"
}`)

	rm.SetValidateCompatibility(false) // clear it to test just format

	rm.SetValidateFormat(true)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details?inline=meta", `{
  "format": "numbers",
  "file": "1\n2\n3"
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-03-06T00:19:13.099947785Z",
  "modifiedat": "2026-03-06T00:19:13.099947785Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "numbers",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-03-06T00:19:13.099947785Z",
    "modifiedat": "2026-03-06T00:19:13.099947785Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	rm.SetValidateFormat(false)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details?inline=meta", `{
  "format": "numbers",
  "file": "not a number"
}`, 201, `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2$details",
  "xid": "/dirs/d1/files/f2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-03-06T01:34:32.228160585Z",
  "modifiedat": "2026-03-06T01:34:32.228160585Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "numbers",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "meta": {
    "fileid": "f2",
    "self": "http://localhost:8181/dirs/d1/files/f2/meta",
    "xid": "/dirs/d1/files/f2/meta",
    "epoch": 1,
    "createdat": "2026-03-06T01:34:32.228160585Z",
    "modifiedat": "2026-03-06T01:34:32.228160585Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versionscount": 1
}
`)

	/* OLD
		XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2/meta", `{
	  "formatauthority": "numberverifier.com"
	}`, 200, `{
	  "fileid": "f2",
	  "self": "http://localhost:8181/dirs/d1/files/f2/meta",
	  "xid": "/dirs/d1/files/f2/meta",
	  "epoch": 2,
	  "createdat": "2026-03-06T01:36:16.139376177Z",
	  "modifiedat": "2026-03-06T01:36:16.186716182Z",
	  "readonly": false,
	  "formatauthority": "numberverifier.com",

	  "defaultversionid": "1",
	  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
	  "defaultversionsticky": false
	}
	`)
	*/

	// rm.SetValidateCompatibility(true)
	// XCheckErr(t, rm.GroupModel.Model.Verify(), ``)

	rm.SetValidateFormat(true)
	XCheckErr(t, rm.GroupModel.Model.Verify(), `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#format_violation",
  "title": "The request would cause Version \"/dirs/d1/files/f2/versions/1\" to be non-compliant with its \"format\" (numbers).",
  "detail": "Line 1 isn't an integer: not a number.",
  "subject": "/dirs/d1/files/f2/versions/1",
  "args": {
    "format": "numbers"
  },
  "source": "1efff26f0ad5:registry:format_numbers:36"
}`)

	// undo
	XNoErr(t, rm.SetValidateFormat(false))
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2$details", `{
  "format": "unknown",
  "file": "1\n2"
}`, 200, `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2$details",
  "xid": "/dirs/d1/files/f2",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-03-12T01:38:51.008399742Z",
  "modifiedat": "2026-03-12T01:38:51.092727731Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "format": "unknown",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versionscount": 1
}
`)

	XNoErr(t, rm.SetValidateFormat(true))
	XCheckErr(t, rm.GroupModel.Model.Verify(), `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Unknown format for /dirs/d1/files/f2: unknown.",
  "subject": "/dirs/d1/files/f2",
  "args": {
    "error_detail": "Unknown format for /dirs/d1/files/f2: unknown"
  },
  "source": "74c45abf9c72:registry:resource:1683"
}`)

	/* OLD
	   	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2$details?inline=meta", `{
	     "format": "unknown",
	     "file": "1\n2"
	   }`, 200, `{
	     "fileid": "f2",
	     "versionid": "1",
	     "self": "http://localhost:8181/dirs/d1/files/f2$details",
	     "xid": "/dirs/d1/files/f2",
	     "epoch": 2,
	     "isdefault": true,
	     "createdat": "2026-03-06T01:48:59.303390838Z",
	     "modifiedat": "2026-03-06T01:48:59.393145568Z",
	     "ancestor": "1",
	     "contenttype": "application/json",
	     "format": "unknown",

	     "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
	     "meta": {
	       "fileid": "f2",
	       "self": "http://localhost:8181/dirs/d1/files/f2/meta",
	       "xid": "/dirs/d1/files/f2/meta",
	       "epoch": 3,
	       "createdat": "2026-03-06T01:48:59.303390838Z",
	       "modifiedat": "2026-03-06T01:48:59.393145568Z",
	       "readonly": false,

	       "defaultversionid": "1",
	       "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
	       "defaultversionsticky": false
	     },
	     "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
	     "versionscount": 1
	   }
	   `)

	   	XNoErr(t, rm.SetValidateFormat(true))
	   	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2$details?inline=meta", `{
	     "format": "unknown",
	     "file": "1\n2"
	   }`, 400, `{
	     "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
	     "title": "Unknown format for /dirs/d1/files/f2: unknown.",
	     "subject": "/dirs/d1/files/f2",
	     "args": {
	       "error_detail": "Unknown format for /dirs/d1/files/f2: unknown"
	     },
	     "source": "1efff26f0ad5:registry:resource:1685"
	   }
	   `)
	*/
	// Undo any changes - so we can exit the tx w/o any complaints
	registry.LoadModel(reg)
}
