package main

import (
	// "fmt"
	"regexp"

	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

func TestRegistry(td *TD) {
	td.DependsOn(TestSniff)
	td.Run(TestModel)
	td.Run(TestCapabilities)
	td.Run(TestRegistryRoot)
	td.Run(TestGroups)
	td.Run(TestResources)
}

func TestModel(td *TD) {
	td.DependsOn(TestSniff)
	reg := td.GetRegistry()

	res, _ := reg.HttpDo(VerboseCount > 2, "GET", "/model", nil)
	// td.Log("Model: %s", string(res.Body))
	td.HTTPStatusMustEqual(res, 200, "GET /model")
	td.HTTPBodyMustJSON(res, "GET /model")

	nTD := NewTD(td, "Parsing model MUST work")
	_, xErr := xrlib.ParseModel(res.Body, reg)
	nTD.NoError(xErr)
}

func TestCapabilities(td *TD) {
	// td.DependsOn(TestModel)
	reg := td.GetRegistry()

	nTD := NewTD(td, "Retrieving capabilities")
	_, xErr := reg.GetCapabilities()
	nTD.NoError(xErr)

	if reg.Capabilities == nil {
		td.Skip("No capabilities found - leaving")
		return
	}

	td.Must(reg.Capabilities.IsAvailable("capabilities"),
		"capabilities.available MUST include \"capabilities\"")
	td.Must(reg.Capabilities.IsAvailable("entities"),
		"capabilities.available MUST include \"entities\"")
	td.Must(reg.Capabilities.IsAvailable("model"),
		"capabilities.available MUST include \"model\"")
	td.Must(!reg.Capabilities.IsAvailableMutable("model"),
		"capabilities.available.entities MUST NOT be \"mutable\"")

	// If capabilities is 'available' then it must be available via both
	// the /capabilities API and /?inline=capabilities API, if ?inline is
	// supported

	// Load from /capabilities first
	res1, _ := reg.HttpDo(VerboseCount > 2, "GET", "/capabilities", nil)
	td.HTTPStatusMustEqual(res1, 200, "GET /capabilities")
	td.HTTPBodyMustJSON(res1, "GET /capabilities")

	nTD = NewTD(td, "Parsing capabilities MUST work")
	_, xErr = ParseCapabilities(res1.Body)
	nTD.NoError(xErr)

	// Load from / and make sure 'capabilities' isn't there w/o ?inline
	res2, _ := reg.HttpDo(VerboseCount > 2, "GET", "/", nil)
	td.HTTPStatusMustEqual(res2, 200, "GET /")
	td.HTTPBodyMustJSON(res2, "GET /")
	_, ok := td.GetObjProp(res2.JSON, "capabilities")
	td.MustEqual(ok, false, "'GET /' MUST NOT include 'capabilities' attribute")

	// If ?inline is supported then make sure it's the same capabilities
	// td.Msg("Testing ?inline=capabilities")
	nTD = NewTD(td, "Testing ?inline=capabilities")
	if reg.Capabilities.FlagEnabled("inline") {
		// Load from / and look for "capabilities" attribute
		res2, _ = reg.HttpDo(VerboseCount > 2, "GET", "/?inline=capabilities", nil)
		// td.Log("Capabilities: %s", string(res2.Body))
		nTD.HTTPStatusMustEqual(res2, 200, "GET /?inline=capabilities")
		nTD.HTTPBodyMustJSON(res2, "GET /?inline=capabilities")

		nTD.ObjMustExist(res2.JSON, "capabilities")

		// Both capabilities MUST be the same JSON
		// We may need to do a sorted-json-diff instead at some point
		// res1.JSON["foo"] = "foo"
		nTD.ObjReqMustEq(res2.JSON, "capabilities", res1.JSON,
			"/capabilities MUST = ?inline=capabilities")
	} else {
		nTD.Skip("?inline not supported")
	}

	nTD = NewTD(td, "Parsing Capabilities MUST work")
	_, xErr = ParseCapabilities(res1.Body) // caps, xErr := ...
	nTD.NoError(xErr)

	// Can't validate when on the client
	// td.NoError(caps.Validate(), "Capabilities MUST validate")
}

func TestRegistryRoot(td *TD) {
	td.DependsOn(TestModel)
	td.DependsOn(TestCapabilities)
	reg := td.GetRegistry()

	// Get the root so we can check its attributes
	res, _ := reg.HttpDo(VerboseCount > 2, "GET", "/", nil)

	td.HTTPStatusMustEqual(res, 200, "GET /")
	td.HTTPBodyMustJSON(res, "GET /")

	td.Log("Root: %s", string(res.Body))
	td.ObjReqMustGt(res.JSON, "specversion", "1.0")
	td.ObjReqMustNe(res.JSON, "registryid", "")
	td.ObjReqMustNe(res.JSON, "self", "")
	reg.SetStuff("self", MustString(res.JSON["self"]))

	_, xErr := reg.GetCapabilities()
	Error(xErr)

	if reg.Capabilities == nil {
		td.Skip("\"shortself\" - Capabilities not available")
	} else {
		if reg.Capabilities.ShortSelf {
			td.ObjReqMustNe(res.JSON, "shortself", "")
		} else {
			td.ObjMustNotExist(res.JSON, "shortself")
		}
	}

	td.ObjReqMustEq(res.JSON, "xid", "/")
	td.ObjReqMustGe(res.JSON, "epoch", 0)
	td.ObjMayExist(res.JSON, "name", "")
	td.ObjMayExist(res.JSON, "description", "")
	td.ObjMayExist(res.JSON, "documentation", "")
	td.ObjOptMustNe(res.JSON, "icon", "")
	td.ObjMayExist(res.JSON, "labels")
	td.ObjReqMustEq(res.JSON, "createdat", "ts")
	td.ObjReqMustEq(res.JSON, "modifiedat", "ts")

	td.ObjMustNotExist(res.JSON, "capabilities")
	td.ObjMustNotExist(res.JSON, "model")
	td.ObjMustNotExist(res.JSON, "modelsource")

	self := res.JSON["self"].(string)

	_, xErr = reg.GetModel()
	td.NoErrorStop(xErr, "Retrieving the model MUST work")

	for _, gm := range reg.Model.Groups {
		td.ObjMustExist(res.JSON, gm.Plural+"count", 0)
		td.ObjReqMustEq(res.JSON, gm.Plural+"url", MakeURL(self, gm.Plural))
	}
}

var reSingularID = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9-\._~:@]{0,127}$`)

func TestGroups(td *TD) {
	td.DependsOn(TestModel)
	td.DependsOn(TestCapabilities)
	reg := td.GetRegistry()

	if len(reg.Model.Groups) == 0 {
		td.Skip("No Group Types defined - leaving")
		return
	}

	for _, k := range SortedKeys(reg.Model.Groups) {
		gm := reg.Model.Groups[k]
		gmPath := "/" + gm.Plural
		gmRes, _ := reg.HttpDo(VerboseCount > 2, "GET", gmPath, nil)
		td.HTTPStatusMustEqual(gmRes, 200, "GET "+gmPath)
		td.HTTPBodyMustJSON(gmRes, "GET "+gmPath)

		td.Log(gmPath + ":" + ToJSON(gmRes.JSON))

		if len(gmRes.JSON) == 0 {
			td.Skip("No groups defined for group type %q", k)
		}

		for _, k := range SortedKeys(gmRes.JSON) {
			v := gmRes.JSON[k]
			td.Must(reSingularID.MatchString(k), "%q MUST match %q", k,
				reSingularID)
			td.Must(!IsNil(v), "Value of %q MUST NOT be nil", k)

			gTD := NewTD(td, "Group: %s", k)
			gPath := gmPath + "/" + k

			gRes, _ := reg.HttpDo(VerboseCount > 2, "GET", gPath, nil)
			gTD.HTTPStatusMustEqual(gRes, 200, "GET "+gPath)
			gTD.HTTPBodyMustJSON(gRes, "GET "+gPath)

			gTD.ObjReqMustEq(gRes.JSON, gm.Singular+"id", k)
			gTD.ObjReqMustEq(gRes.JSON, "self",
				MakeURL(reg.GetStuffAsString("self"),
					gm.Plural, MustString(gRes.JSON[gm.Singular+"id"])))
			gTD.ObjReqMustEq(gRes.JSON, "xid",
				MakeURL("/", gm.Plural, MustString(gRes.JSON[gm.Singular+"id"])))
			gTD.ObjReqMustGe(gRes.JSON, "epoch", 0)
			gTD.ObjMayExist(gRes.JSON, "name", "")
			gTD.ObjMayExist(gRes.JSON, "description", "")
			gTD.ObjMayExist(gRes.JSON, "documentation", "")
			gTD.ObjOptMustNe(gRes.JSON, "icon", "")
			gTD.ObjMayExist(gRes.JSON, "labels")
			gTD.ObjReqMustEq(gRes.JSON, "createdat", "ts")
			gTD.ObjReqMustEq(gRes.JSON, "modifiedat", "ts")

			reg.SetStuff("gm", gm)
			reg.SetStuff("groupPath", gPath)
			reg.SetStuff("gxid", MustString(gRes.JSON["xid"]))
			break
		}
	}
}

func TestResources(td *TD) {
	td.DependsOn(TestGroups)
	reg := td.GetRegistry()

	gmAny, ok := reg.GetStuff("gm")
	if !ok {
		// No gm must mean there are no groups
		td.Skip("No Group Types defined  - leaving")
		return
	}
	gm, ok := gmAny.(*xrlib.GroupModel)
	if !ok {
		Error("reg.stuff.gm != *GroupModel")
	}
	gxid := reg.GetStuffAsString("gxid")

	groupPath := reg.GetStuffAsString("groupPath")

	if len(gm.Resources) == 0 {
		td.Skip("No Resource Types defined for %q Group Type - leaving",
			gm.Plural)
		return
	}

	for _, k := range SortedKeys(gm.Resources) {
		rm := gm.Resources[k]

		rPath := groupPath + "/" + rm.Plural
		rmRes, _ := reg.HttpDo(VerboseCount > 2, "GET", rPath, nil)
		td.HTTPStatusMustEqual(rmRes, 200, "GET "+rPath)
		td.HTTPBodyMustJSON(rmRes, "GET "+rPath)

		td.Log(rPath + ":" + ToJSON(rmRes.JSON))

		if len(rmRes.JSON) == 0 {
			td.Skip("No resources defined for resource type  %q - leaving",
				rm.Plural)
		}

		for _, k := range SortedKeys(rmRes.JSON) {
			v := rmRes.JSON[k]
			td.Must(reSingularID.MatchString(k), "%q MUST match %q", k,
				reSingularID)
			td.Must(!IsNil(v), "Value of %q MUST NOT be nil", k)

			rTD := NewTD(td, "Resource: %s", k)
			// TODO conditional $details - check xReg http headers
			rPath := rPath + "/" + k + "$details"

			rRes, _ := reg.HttpDo(VerboseCount > 2, "GET", rPath, nil)
			rTD.HTTPStatusMustEqual(rRes, 200, "GET "+rPath)
			rTD.HTTPBodyMustJSON(rRes, "GET "+rPath)

			rTD.ObjReqMustEq(rRes.JSON, rm.Singular+"id", k)
			// Conditional $details
			rTD.ObjReqMustGe(rRes.JSON, "self",
				MakeURL(reg.GetStuffAsString("self"),
					gxid, rm.Plural, MustString(rRes.JSON[rm.Singular+"id"])))
			rTD.ObjReqMustEq(rRes.JSON, "xid",
				MakeURL(gxid, rm.Plural, MustString(rRes.JSON[rm.Singular+"id"])))
			rTD.ObjReqMustGe(rRes.JSON, "epoch", 0)
			rTD.ObjMayExist(rRes.JSON, "name", "")
			rTD.ObjMayExist(rRes.JSON, "description", "")
			rTD.ObjMayExist(rRes.JSON, "documentation", "")
			rTD.ObjOptMustNe(rRes.JSON, "icon", "")
			rTD.ObjMayExist(rRes.JSON, "labels")
			rTD.ObjReqMustEq(rRes.JSON, "createdat", "ts")
			rTD.ObjReqMustEq(rRes.JSON, "modifiedat", "ts")

			reg.SetStuff("rm", rm)
			reg.SetStuff("resourcePath", rPath)
			break
		}
	}
}
