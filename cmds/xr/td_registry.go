package main

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

const conformMutationEpochSentinel = "xr-conform-create-only"

func TestSniff(td *TD) {
	reg := td.GetRegistry()
	td.Log("Server URL: %s", reg.GetServerURL())

	res, _ := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     td.GetRegistry().WithPath("/"),
		Headers: xrlib.HTTPHeaders,
		Body:    nil,
		Debug:   VerboseCount > 2,
	})
	td.HTTPStatusMustEqual(res, 200, "GET /")
	td.HTTPBodyMustJSON(res, "GET /")
}

func TestRegistry(td *TD) {
	td.DependsOn(TestSniff)
	td.Run(TestModel)
	td.Run(TestCapabilities)
	td.Run(TestRegistryRoot)
	td.Run(TestGroups)
	td.Run(TestFilter)
	td.Run(TestResources)
}

func TestModel(td *TD) {
	td.DependsOn(TestSniff)
	reg := td.GetRegistry()

	res, _ := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     td.GetRegistry().WithPath("/model"),
		Headers: xrlib.HTTPHeaders,
		Body:    nil,
		Debug:   VerboseCount > 2,
	})
	// td.Log("Model: %s", string(res.Body))
	td.HTTPStatusMustEqual(res, 200, "GET /model")
	td.HTTPBodyMustJSON(res, "GET /model")

	nTD := NewTD(td, "Parsing model MUST work")
	model, xErr := registry.ParseModel(res.Body, (*registry.Registry)(reg))
	nTD.NoErrorStop(xErr)

	// if td.Config.Model == nil {
	// td.Config.Model = model
	// }

	// if xErr == nil {
	// td.Config.Model = model
	// }

	nTD = NewTD(td, "Verifying model MUST work")
	nTD.NoError(model.Verify())
}

func TestCapabilities(td *TD) {
	// td.DependsOn(TestModel)
	reg := td.GetRegistry()

	nTD := NewTD(td, "Retrieving capabilities")
	_, xErr := (*registry.Registry)(reg).GetCapabilities()
	nTD.NoError(xErr)

	if reg.Capabilities == nil {
		td.Skip("No capabilities found - leaving")
		td.Skip()
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
	res1, _ := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     td.GetRegistry().WithPath("/capabilities"),
		Headers: xrlib.HTTPHeaders,
		Body:    nil,
		Debug:   VerboseCount > 2,
	})
	td.HTTPStatusMustEqual(res1, 200, "GET /capabilities")
	td.HTTPBodyMustJSON(res1, "GET /capabilities")

	nTD = NewTD(td, "Parsing capabilities MUST work")
	_, xErr = ParseCapabilities(res1.Body)
	nTD.NoError(xErr)

	// Load from / and make sure 'capabilities' isn't there w/o ?inline
	res2, _ := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     td.GetRegistry().WithPath("/"),
		Headers: xrlib.HTTPHeaders,
		Body:    nil,
		Debug:   VerboseCount > 2,
	})
	td.HTTPStatusMustEqual(res2, 200, "GET /")
	td.HTTPBodyMustJSON(res2, "GET /")
	_, ok := td.GetObjProp(res2.JSON, "capabilities")
	td.MustEqual(ok, false, "'GET /' MUST NOT include 'capabilities' attribute")

	// If ?inline is supported then make sure it's the same capabilities
	// td.Msg("Testing ?inline=capabilities")
	nTD = NewTD(td, "Testing ?inline=capabilities")
	if reg.Capabilities.FlagEnabled("inline") {
		// Load from / and look for "capabilities" attribute
		res2, _ = HttpDo(&HttpRequest{
			Verb:    "GET",
			URL:     td.GetRegistry().WithPath("/?inline=capabilities"),
			Headers: xrlib.HTTPHeaders,
			Body:    nil,
			Debug:   VerboseCount > 2,
		})
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
		nTD.Skip()
	}

	nTD = NewTD(td, "Parsing Capabilities MUST work")
	_, xErr = ParseCapabilities(res1.Body) // caps, xErr := ...
	nTD.NoError(xErr)

	// if td.Config.Capabilities == nil {
	// td.Config.Capabilities = caps
	// }

	// if xErr == nil {
	// td.Config.Capabilities = caps
	// }

	// Can't validate when on the client
	// nTD = NewTD(td, "Capabilities MUST validate")
	// nTD.NoError(caps.Validate(), "Capabilities MUST validate")
}

func TestRegistryRoot(td *TD) {
	td.DependsOn(TestModel)
	td.DependsOn(TestCapabilities)
	reg := td.GetRegistry()

	// Get the root so we can check its attributes
	res, _ := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     td.GetRegistry().WithPath("/"),
		Headers: xrlib.HTTPHeaders,
		Body:    nil,
		Debug:   VerboseCount > 2,
	})

	td.HTTPStatusMustEqual(res, 200, "GET /")
	td.HTTPBodyMustJSON(res, "GET /")

	td.Log("Root: %s", string(res.Body))
	td.ObjReqMustGt(res.JSON, "specversion", "1.0")
	td.ObjReqMustNe(res.JSON, "registryid", "")
	td.ObjReqMustNe(res.JSON, "self", "")
	reg.SetStuff("self", MustString(res.JSON["self"]))

	_, xErr := (*registry.Registry)(reg).GetCapabilities()
	td.NoErrorStop(xErr, "Retrieving capabilities MUST work")

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

	_, xErr = (*registry.Registry)(reg).GetModel()
	td.NoErrorStop(xErr, "Retrieving the model MUST work")

	for _, name := range SortedKeys(reg.Model.Groups) {
		gm := reg.Model.Groups[name]
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
		td.Skip()
		return
	}

	groupPaths := map[string]string{}

	for _, groupType := range SortedKeys(reg.Model.Groups) {
		gm := reg.Model.Groups[groupType]
		gmPath := "/" + gm.Plural
		gmRes, _ := HttpDo(&HttpRequest{
			Verb:    "GET",
			URL:     td.GetRegistry().WithPath(gmPath),
			Headers: xrlib.HTTPHeaders,
			Body:    nil,
			Debug:   VerboseCount > 2,
		})
		td.HTTPStatusMustEqual(gmRes, 200, "GET "+gmPath)
		td.HTTPBodyMustJSON(gmRes, "GET "+gmPath)

		td.Log(gmPath + ":" + ToJSON(gmRes.JSON))

		if len(gmRes.JSON) == 0 {
			td.Skip("No groups defined for group type %q", groupType)
			td.Skip()
		}

		for _, id := range SortedKeys(gmRes.JSON) {
			v := gmRes.JSON[id]
			td.Must(reSingularID.MatchString(id), "%q MUST match %q", id,
				reSingularID)
			td.Must(!IsNil(v), "Value of %q MUST NOT be nil", id)

			gTD := NewTD(td, "Group: %s", id)
			gPath := gmPath + "/" + id

			gRes, _ := HttpDo(&HttpRequest{
				Verb:    "GET",
				URL:     td.GetRegistry().WithPath(gPath),
				Headers: xrlib.HTTPHeaders,
				Body:    nil,
				Debug:   VerboseCount > 2,
			})
			gTD.HTTPStatusMustEqual(gRes, 200, "GET "+gPath)
			gTD.HTTPBodyMustJSON(gRes, "GET "+gPath)

			gTD.ObjReqMustEq(gRes.JSON, gm.Singular+"id", id)
			gTD.ObjReqMustEq(gRes.JSON, "self",
				MakeURL(reg.GetStuffAsString("self"),
					gm.Plural, MustString(gRes.JSON[gm.Singular+"id"])))
			gTD.ObjReqMustEq(gRes.JSON, "xid",
				MakeURL("/", gm.Plural,
					MustString(gRes.JSON[gm.Singular+"id"])))
			gTD.ObjReqMustGe(gRes.JSON, "epoch", 0)
			gTD.ObjMayExist(gRes.JSON, "name", "")
			gTD.ObjMayExist(gRes.JSON, "description", "")
			gTD.ObjMayExist(gRes.JSON, "documentation", "")
			gTD.ObjOptMustNe(gRes.JSON, "icon", "")
			gTD.ObjMayExist(gRes.JSON, "labels")
			gTD.ObjReqMustEq(gRes.JSON, "createdat", "ts")
			gTD.ObjReqMustEq(gRes.JSON, "modifiedat", "ts")

			groupPaths[groupType] = gPath
			break
		}
	}

	reg.SetStuff("groupPaths", groupPaths)
}

func TestFilter(td *TD) {
	td.DependsOn(TestCapabilities)
	reg := td.GetRegistry()

	if reg.Capabilities == nil {
		td.Skip("No capabilities found - leaving")
		td.Skip()
		return
	}

	if !reg.Capabilities.FlagEnabled("filter") {
		td.Skip("?filter not supported - leaving")
		td.Skip()
		return
	}

	td.DependsOn(TestGroups)

	groupPathsAny, ok := reg.GetStuff("groupPaths")
	if !ok {
		td.Skip("No observable Group Types defined - leaving")
		td.Skip()
		return
	}
	groupPaths, ok := groupPathsAny.(map[string]string)
	PanicIf(!ok, "reg.stuff.groupPaths != map[string]string")

	if len(groupPaths) == 0 {
		td.Skip("No observable Group Types defined - leaving")
		td.Skip()
		return
	}

	xid, err := ParseXid(groupPaths[SortedKeys(groupPaths)[0]])
	PanicIf(err != nil, "reg.stuff.groupPaths has an invalid XID: %s", err)

	gm := reg.Model.Groups[xid.Group]
	filterAttr := gm.Singular + "id"
	collectionPath := "/" + xid.Group
	matchID := xid.GroupID
	if matchID == "null" {
		matchID = "NULL"
	}

	matchPath := collectionPath + "?filter=" + filterAttr + "=" +
		url.QueryEscape(matchID)
	matchRes := td.HTTPGetJSON(matchPath)
	if matchRes != nil {
		td.MustEqual([]string{xid.GroupID}, SortedKeys(matchRes.JSON),
			"'GET %s' MUST return exactly group %q", matchPath, xid.GroupID)
		td.Must(!IsNil(matchRes.JSON[xid.GroupID]),
			"'GET %s' group %q MUST NOT be nil", matchPath, xid.GroupID)
	}

	absentPath := collectionPath + "?filter=" + filterAttr + "=null"
	absentRes := td.HTTPGetJSON(absentPath)
	if absentRes != nil {
		td.MustEqual(0, len(absentRes.JSON),
			"'GET %s' MUST return an empty collection", absentPath)
	}
}

func TestMutations(td *TD) {
	if !td.Config.AllowMutations {
		td.Msg("Mutation tests require --allow-mutations - leaving")
		td.Skip()
		return
	}

	td.DependsOn(TestModel)
	td.DependsOn(TestCapabilities)

	reg := td.GetRegistry()
	registryReg := (*registry.Registry)(reg)

	capabilities, xErr := registryReg.GetCapabilities()
	td.NoError(xErr, "Retrieving capabilities MUST work")
	td.StopOnFail()
	if capabilities == nil {
		td.Skip("No capabilities found - leaving")
		td.Skip()
		return
	}
	if !capabilities.IsAvailableMutable("entities") {
		td.Skip("Registry entities are not advertised as mutable - leaving")
		td.Skip()
		return
	}

	model, xErr := registryReg.GetModel()
	td.NoError(xErr, "Retrieving the model MUST work")
	td.StopOnFail()

	groupType, groupModel, reasons := conformMutationGroupModel(model)
	if groupModel == nil {
		td.Msg("Mutation tests require a Group Type whose client-required " +
			"attributes all have server defaults.")
		for _, reason := range reasons {
			td.Msg(reason)
		}
		td.Msg("No Group Type supports run-owned creation - leaving")
		td.Skip()
		return
	}

	id := "xr-conform-" + NewUUID() + NewUUID()
	path := "/" + groupType + "/" + id
	body := []byte(ToJSON(map[string]any{
		groupModel.Singular + "id": id,
		"epoch":                    conformMutationEpochSentinel,
	}))

	preflightTD := NewTD(td, "Confirm target Group is absent")
	preflightRes, _ := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     reg.WithPath(path),
		Headers: xrlib.HTTPHeaders,
		Debug:   VerboseCount > 2,
	})
	preflightTD.LogHTTP(preflightRes)
	preflightTD.Must(preflightRes != nil,
		"Preflight GET MUST return an HTTP response")
	preflightTD.StopOnFail()
	preflightTD.HTTPStatusMustEqual(preflightRes, 404,
		"Preflight GET of mutation destination")
	preflightTD.StopOnFail()

	createTD := NewTD(td, "Create run-owned Group")
	ownedEpoch := (*uint64)(nil)
	defer func() {
		cleanupConformMutationGroup(td, groupModel, path, id, ownedEpoch,
			capabilities.FlagEnabled("epoch"))
	}()

	createRes, _ := HttpDo(&HttpRequest{
		Verb:    "PUT",
		URL:     reg.WithPath(path),
		Headers: xrlib.HTTPHeaders,
		Body:    body,
		Debug:   VerboseCount > 2,
	})
	createTD.LogHTTP(createRes)
	createTD.Must(createRes != nil,
		"Creating the run-owned Group MUST return an HTTP response")
	createTD.StopOnFail()
	createTD.NoError(createRes.Error, "Creating the run-owned Group MUST work")
	createTD.StopOnFail()
	createTD.HTTPStatusMustEqual(createRes, 201,
		"Creating the run-owned Group")
	createTD.StopOnFail()
	createTD.httpBodyMustJSON(createRes, true,
		"Creating the run-owned Group")
	createTD.StopOnFail()
	ownedEpoch = conformMutationGroupState(createTD, createRes,
		groupModel, path, id)
	createTD.StopOnFail()
	createTD.Pass("Created Group matches the target and is empty")

	expectedURL := reg.WithPath(path)
	createTD.HTTPHeaderMustEqual(createRes, "Location",
		expectedURL, "Create MUST return the exact Location")
	createTD.StopOnFail()

	readTD := NewTD(td, "Read run-owned Group")
	readRes := readTD.HTTPGetJSON(path)
	readTD.StopOnFail()
	conformMutationGroupState(readTD, readRes, groupModel, path, id)
	readTD.StopOnFail()
	readTD.ObjReqMustEq(readRes.JSON, "epoch", int(*ownedEpoch),
		"Group epoch MUST remain unchanged")
	readTD.StopOnFail()
}

func conformMutationGroupModel(model *registry.Model) (
	string, *registry.GroupModel, []string) {

	reasons := []string{}
	if model == nil || len(model.Groups) == 0 {
		return "", nil, []string{"The model defines no Group Types."}
	}

	for _, groupType := range SortedKeys(model.Groups) {
		groupModel := model.Groups[groupType]
		if groupModel == nil {
			reasons = append(reasons,
				fmt.Sprintf("Group Type %q has no definition.", groupType))
			continue
		}

		missing := []string{}
		attributes := groupModel.GetBaseAttributes()
		for _, name := range SortedKeys(attributes) {
			attr := attributes[name]
			if attr == nil || !attr.Required || !IsNil(attr.Default) {
				continue
			}
			if attr.Internals != nil &&
				(attr.ReadOnly || attr.Immutable ||
					attr.Internals.GetFn != nil ||
					attr.Internals.UpdateFn != nil) {

				continue
			}
			missing = append(missing, name)
		}
		if len(missing) == 0 {
			return groupType, groupModel, nil
		}
		reasons = append(reasons, fmt.Sprintf(
			"Group Type %q requires client attribute(s) with no default: %s.",
			groupType, strings.Join(missing, ", ")))
	}
	return "", nil, reasons
}

func conformMutationGroupState(td *TD, res *HttpResponse,
	groupModel *registry.GroupModel, path, id string) *uint64 {

	td.ObjReqMustEq(res.JSON, groupModel.Singular+"id", id,
		"Group ID MUST match the requested ID")
	td.ObjReqMustEq(res.JSON, "xid", path,
		"Group XID MUST match the requested path")
	td.ObjReqMustGe(res.JSON, "epoch", 0)

	epochAny, _ := td.GetObjProp(res.JSON, "epoch")
	epoch, epochErr := conformMutationUInt(epochAny)
	if epochErr != nil {
		td.Log("Invalid epoch: %s", epochErr)
		td.Fail("\"epoch\" MUST be an exact uinteger less than 2^53")
	}

	for _, resourceType := range SortedKeys(groupModel.Resources) {
		countName := resourceType + "count"
		td.ObjReqMustGe(res.JSON, countName, 0)
		td.ObjReqMustEq(res.JSON, countName, 0,
			"%s count MUST be zero", resourceType)

		if _, exists := td.GetObjProp(res.JSON, resourceType); exists {
			td.ObjReqMustEq(res.JSON, resourceType, map[string]any{})
		} else {
			td.ObjPropMustNotExist(res.JSON, resourceType)
		}
	}

	if td.Status == FAIL || epochErr != nil {
		return nil
	}
	return &epoch
}

func conformMutationUInt(value any) (uint64, error) {
	const maxSafeJSONInteger = float64(1 << 53)

	number, ok := value.(float64)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) ||
		number < 0 || math.Trunc(number) != number ||
		number >= maxSafeJSONInteger {

		return 0, fmt.Errorf("must be an exact uinteger less than 2^53")
	}
	return uint64(number), nil
}

func cleanupConformMutationGroup(td *TD, groupModel *registry.GroupModel,
	path, id string, ownedEpoch *uint64, useEpochFlag bool) {

	failFast := td.Config.FailFast
	td.Config.FailFast = false
	defer func() {
		td.Config.FailFast = failFast
	}()

	cleanupTD := NewTD(td, "Clean up run-owned Group")
	reg := td.GetRegistry()
	res, _ := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     reg.WithPath(path),
		Headers: xrlib.HTTPHeaders,
		Debug:   VerboseCount > 2,
	})
	cleanupTD.LogHTTP(res)
	if res == nil {
		cleanupTD.Fail("Cleanup GET MUST return an HTTP response")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	if res.Code == 404 {
		if ownedEpoch != nil {
			cleanupTD.Fail("Run-owned Group disappeared before cleanup")
		} else {
			cleanupTD.Pass("No run-owned Group remains")
		}
		return
	}
	if res.Code != 200 {
		cleanupTD.Fail("Cleanup GET MUST return 200 or 404")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	if res.JSON == nil {
		cleanupTD.Fail("Cleanup GET MUST return a JSON body")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	if ownedEpoch == nil {
		cleanupTD.Fail("Cleanup requires a confirmed run-owned creation")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}

	epoch := conformMutationGroupState(cleanupTD, res, groupModel, path, id)
	if epoch == nil {
		cleanupTD.Fail("Cleanup MUST only delete the exact confirmed Group")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	cleanupTD.ObjReqMustEq(res.JSON, "epoch", int(*ownedEpoch),
		"Group epoch MUST remain unchanged")
	if cleanupTD.Status == FAIL {
		cleanupTD.Fail("Run-owned Group changed after creation verification")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	cleanupTD.Pass("Cleanup state proof MUST succeed")

	epochString := strconv.FormatUint(*epoch, 10)
	deletePath := path + "?epoch=" + url.QueryEscape(epochString)
	var deleteBody []byte
	if !useEpochFlag {
		deletePath = "/" + groupModel.Plural
		deleteBody = []byte(fmt.Sprintf(`{%s:{"epoch":%s}}`,
			strconv.Quote(id), epochString))
	}
	deleteRes, _ := HttpDo(&HttpRequest{
		Verb:    "DELETE",
		URL:     reg.WithPath(deletePath),
		Headers: xrlib.HTTPHeaders,
		Body:    deleteBody,
		Debug:   VerboseCount > 2,
	})
	cleanupTD.LogHTTP(deleteRes)
	if deleteRes == nil {
		cleanupTD.Fail("DELETE MUST return an HTTP response")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	cleanupTD.NoError(deleteRes.Error, "Deleting the run-owned Group MUST work")
	if deleteRes.Error != nil ||
		(deleteRes.Code != 200 && deleteRes.Code != 204) {

		cleanupTD.Fail("DELETE MUST return 200 or 204")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	cleanupTD.Pass("DELETE returned an allowed success status")

	verifyRes, _ := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     reg.WithPath(path),
		Headers: xrlib.HTTPHeaders,
		Debug:   VerboseCount > 2,
	})
	cleanupTD.LogHTTP(verifyRes)
	if verifyRes == nil {
		cleanupTD.Fail("Cleanup verification GET MUST return an HTTP response")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	if verifyRes.Code != 404 {
		cleanupTD.Fail("Cleanup verification GET MUST return 404")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	cleanupTD.Pass("DELETE removed the exact run-owned Group")
}

func TestResources(td *TD) {
	td.DependsOn(TestGroups)
	reg := td.GetRegistry()

	groupPathsAny, ok := reg.GetStuff("groupPaths")
	if !ok {
		td.Skip("No Group Types defined - leaving")
		td.Skip()
		return
	}
	groupPaths, ok := groupPathsAny.(map[string]string)
	if !ok {
		td.FailNow("reg.stuff.groupPaths != map[string]string")
	}

	if len(groupPaths) == 0 {
		td.Skip("No Group Types defined - leaving")
		td.Skip()
		return
	}

	for _, groupType := range SortedKeys(groupPaths) {
		groupPath := groupPaths[groupType]
		gm := reg.Model.Groups[groupType]

		if len(gm.Resources) == 0 {
			td.Skip("No Resource Types defined for %q Group Type - leaving",
				gm.Plural)
			continue
		}

		for _, k := range SortedKeys(gm.Resources) {
			rm := gm.Resources[k]

			rPath := groupPath + "/" + rm.Plural
			rmRes, _ := HttpDo(&HttpRequest{
				Verb:    "GET",
				URL:     td.GetRegistry().WithPath(rPath),
				Headers: xrlib.HTTPHeaders,
				Body:    nil,
				Debug:   VerboseCount > 2,
			})
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

				rRes, _ := HttpDo(&HttpRequest{
					Verb:    "GET",
					URL:     td.GetRegistry().WithPath(rPath),
					Headers: xrlib.HTTPHeaders,
					Body:    nil,
					Debug:   VerboseCount > 2,
				})
				rTD.HTTPStatusMustEqual(rRes, 200, "GET "+rPath)
				if rRes == nil {
					break
				}
				rTD.HTTPBodyMustJSON(rRes, "GET "+rPath)
				if rRes.JSON == nil {
					break
				}

				rTD.ObjReqMustEq(rRes.JSON, rm.Singular+"id", k)
				// Conditional $details
				rTD.ObjReqMustGe(rRes.JSON, "self",
					MakeURL(reg.GetStuffAsString("self"),
						groupPath, rm.Plural,
						MustString(rRes.JSON[rm.Singular+"id"])))
				rTD.ObjReqMustEq(rRes.JSON, "xid",
					MakeURL(groupPath, rm.Plural,
						MustString(rRes.JSON[rm.Singular+"id"])))
				rTD.ObjReqMustGe(rRes.JSON, "epoch", 0)
				rTD.ObjMayExist(rRes.JSON, "name", "")
				rTD.ObjMayExist(rRes.JSON, "description", "")
				rTD.ObjMayExist(rRes.JSON, "documentation", "")
				rTD.ObjOptMustNe(rRes.JSON, "icon", "")
				rTD.ObjMayExist(rRes.JSON, "labels")
				rTD.ObjReqMustEq(rRes.JSON, "createdat", "ts")
				rTD.ObjReqMustEq(rRes.JSON, "modifiedat", "ts")

				break
			}
		}
	}
}
