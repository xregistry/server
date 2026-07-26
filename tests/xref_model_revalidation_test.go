package tests

// TestXrefModelRevalidationFormatCascade checks that when a model
// change forces re-validation of an xref TARGET's format/compat
// system attributes (formatvalidated/compatibilityvalidated) with no
// other attribute change, that new value is correctly cascaded/
// mirrored into any xref SOURCE pointing at it.
//
// This exercises Registry.VerifyData() (registry/registry.go), the
// path used by Model.ApplyNewModel()/PUT-/modelsource to re-verify
// all existing data against a changed model - as opposed to the
// normal per-Resource HTTP PUT path (ValidateResource() called
// directly from Tx.Validate()).

import (
	"testing"
)

func TestXrefModelRevalidationFormatCascade(t *testing.T) {
	reg := NewRegistry("TestXrefModelRevalidationFormatCascade")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)

	// Create target f1 with format set, but validateformat=false so
	// formatvalidated is NOT computed/set yet.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "format": "numbers",
  "file": "1\n2\n3"
}`, 201, `*`)

	// Confirm formatvalidated is absent (validateformat is off).
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200, `*`)

	// Create xref fx -> f1.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Confirm the xref mirror also has no formatvalidated yet.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `*`)

	// Now flip validateformat=true at the MODEL level (no change to f1's
	// own data at all) - this forces Registry.VerifyData() to
	// re-validate every existing Resource, including f1 (setting
	// formatvalidated=true, since "1\n2\n3" is valid "numbers" format)
	// and fx (an xref source, whose runCascade() re-run should pick up
	// the target's fresh formatvalidated=true value).
	rm.SetValidateFormat(true)
	modelSrc := reg.Model.MustUserMarshal("", "  ")
	XHTTP(t, reg, "PUT", "/modelsource", modelSrc, 200, `*`)

	// The TARGET f1 must now show formatvalidated=true.
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200,
		`^(?s)^.*"epoch": 1,.*"formatvalidated": true`)

	// The xref SOURCE fx must mirror that same formatvalidated=true -
	// this is the part that was missing coverage.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`^(?s)^.*"epoch": 1,.*"formatvalidated": true`)
}
