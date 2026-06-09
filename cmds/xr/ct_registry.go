package main

import (
	// "fmt"

	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

func TestRegistry(td *TD) {
	td.DependsOn(TestSniff)
	td.Run(TestCapabilities)
	td.Run(TestModel)
	td.Run(TestRoot)
}

func TestCapabilities(td *TD) {
	td.DependsOn(TestSniff)
	reg := td.GetRegistry()

	if reg.Capabilities == nil {
		td.Skip("No capabilities found")
		return
	}

	if !reg.Capabilities.IsAvailable("capabilities") {
		td.Skip("Capabilities are not 'available'")
		return
	}

	// If capabilities is 'available' then it must be available via both
	// the /capabilities API and /?inline=capabilities API, if ?inline is
	// supported

	// Load from /capabilities first
	res1, _ := reg.HttpDo(VerboseCount > 2, "GET", "/capabilities", nil)
	td.HTTPStatusMustEqual(res1, 200, "GET /capabilities")
	td.HTTPBodyMustJSON(res1, "GET /capabilities")
	_, xErr := ParseCapabilities(res1.Body)
	td.NoError(xErr, "Parsing capabilities MUST work")

	// Load from / and make sure 'capabilities' isn't there w/o ?inline
	res2, _ := reg.HttpDo(VerboseCount > 2, "GET", "/", nil)
	td.HTTPStatusMustEqual(res2, 200, "GET /")
	td.HTTPBodyMustJSON(res2, "GET /")
	_, ok := td.GetObjProp(res2.JSON, "capabilities")
	td.MustEqual(ok, false, "'GET /' MUST NOT include 'capabilities' attribute")

	// If ?inline is supported then make sure it's the same capabilities
	if reg.Capabilities.FlagEnabled("inline") {
		// Load from / and look for "capabilities" attribute
		res2, _ = reg.HttpDo(VerboseCount > 2, "GET", "/?inline=capabilities", nil)
		// td.Log("Capabilities: %s", string(res2.Body))
		td.HTTPStatusMustEqual(res2, 200, "GET /?inline=capabilities")
		td.HTTPBodyMustJSON(res2, "GET /?inline=capabilities")
		val, ok := td.GetObjProp(res2.JSON, "capabilities")
		td.MustEqual(ok, true,
			"'GET /?inline=capabilities' MUST include 'capabilities' attribute")

		// Both capabilities MUST be the same JSON
		// We may need to do a sorted-json-diff instead at some point
		td.MustEqual(res1.JSON, val, "Both 'capabilities' JSON MUST be the same")
	} else {
		td.Skip("?inline=capabilities not supported")
	}

	_, xErr = ParseCapabilities(res1.Body) // caps, xErr := ...
	td.NoError(xErr, "Parsing capabilities MUST work")
	td.Pass("Capabilities successfully parsed")

	// Can't validate when on the client
	// td.NoError(caps.Validate(), "Capabilities MUST validate")
}

func TestModel(td *TD) {
	td.DependsOn(TestSniff)
	td.DependsOn(TestCapabilities)
	reg := td.GetRegistry()

	res, _ := reg.HttpDo(VerboseCount > 2, "GET", "/model", nil)
	// td.Log("Model: %s", string(res.Body))
	td.HTTPStatusMustEqual(res, 200, "GET /model")
	td.HTTPBodyMustJSON(res, "GET /model")

	_, xErr := xrlib.ParseModel(res.Body, reg)
	td.NoError(xErr, "Parsing model MUST work")
}

func TestRoot(td *TD) {
	td.DependsOn(TestSniff)
	td.DependsOn(TestCapabilities)
	td.DependsOn(TestModel)
	reg := td.GetRegistry()

	// Get the root so we can check its attributes
	res, _ := reg.HttpDo(VerboseCount > 2, "GET", "/", nil)
	td.HTTPStatusMustEqual(res, 200, "GET /")
	td.HTTPBodyMustJSON(res, "GET /")

	td.Log("Root: %s", string(res.Body))
	td.HTTPPropMustEqual(res, "specversion", SPECVERSION)
	td.HTTPPropMustNotEqual(res, "registryid", "")
	td.HTTPPropMustNotEqual(res, "self", "")
	td.HTTPPropMustNotEqual(res, "epoch", "")

	epoch, err := AnyToUInt(td.HTTPGetProp(res, "epoch"))
	td.NoError(err, "Attribute %q %s(%v)", "epoch", err, epoch)
	td.Must(epoch >= 0, "\"epoch\" (%v) must be >= 0", epoch)
}
