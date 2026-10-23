package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

const conformMutationLabel = "xr-conform-run"
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

	nTD = NewTD(td, "Verifying model MUST work")
	nTD.NoErrorStop(model.Verify())
	reg.Model = model
}

func TestCapabilities(td *TD) {
	// td.DependsOn(TestModel)
	reg := td.GetRegistry()

	nTD := NewTD(td, "Retrieving capabilities")
	_, xErr := reg.GetCapabilities()
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
	}

	nTD = NewTD(td, "Parsing Capabilities MUST work")
	_, xErr = ParseCapabilities(res1.Body) // caps, xErr := ...
	nTD.NoError(xErr)
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

	_, xErr := reg.GetCapabilities()
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

func TestTDGroupCreateReadDelete(td *TD) {
	if !td.Config.AllowMutations {
		td.FailNow("Mutation tests require explicit authorization")
	}

	td.DependsOn(TestModel)
	td.DependsOn(TestCapabilities)

	reg := td.GetRegistry()
	if reg.Capabilities == nil {
		td.Skip("No capabilities found - leaving")
		td.Skip()
		return
	}
	if !reg.Capabilities.IsAvailableMutable("entities") {
		td.Skip("Registry entities are not advertised as mutable - leaving")
		td.Skip()
		return
	}
	groupType, groupModel := conformMutationGroupModel(reg.Model)
	if groupModel == nil {
		td.Skip("No Group Type supports run-owned creation - leaving")
		td.Skip()
		return
	}

	id := "xr-conform-" + NewUUID() + NewUUID()
	owner := "xr-conform-" + NewUUID() + NewUUID()
	path := "/" + groupType + "/" + id
	body := []byte(ToJSON(map[string]any{
		groupModel.Singular + "id": id,
		"epoch":                    conformMutationEpochSentinel,
		"labels": map[string]string{
			conformMutationLabel: owner,
		},
	}))

	preflightTD := NewTD(td, "Confirm target Group is absent")
	preflightRes, preflightErr := HttpDo(&HttpRequest{
		Verb:       "GET",
		URL:        reg.WithPath(path),
		Headers:    xrlib.HTTPHeaders,
		Debug:      VerboseCount > 2,
		NoRedirect: true,
	})
	if preflightRes == nil {
		logConformMutationHTTPFailure(preflightTD, "Preflight GET",
			preflightRes, preflightErr)
		preflightTD.Fail("Preflight GET MUST return an HTTP response")
		td.Stop()
	}
	if preflightRes.Code != 404 {
		logConformMutationHTTPFailure(preflightTD, "Preflight GET",
			preflightRes, preflightErr)
	}
	preflightTD.HTTPStatusMustEqual(preflightRes, 404,
		"Preflight GET of mutation destination")
	if preflightTD.Status == FAIL {
		td.Stop()
	}

	createTD := NewTD(td, "Create run-owned Group")
	ownedEpoch := ""
	defer func() {
		cleanupConformMutationGroup(td, groupModel, path, id, owner,
			ownedEpoch)
	}()

	createRes, createErr := HttpDo(&HttpRequest{
		Verb:       "PUT",
		URL:        reg.WithPath(path),
		Headers:    xrlib.HTTPHeaders,
		Body:       body,
		Debug:      VerboseCount > 2,
		NoRedirect: true,
	})
	if createErr != nil || createRes == nil || createRes.Code != 201 {
		logConformMutationHTTPFailure(createTD, "Create PUT",
			createRes, createErr)
	}
	createTD.NoErrorStop(createErr, "Creating the run-owned Group MUST work")

	createTD.HTTPStatusMustEqual(createRes, 201,
		"Creating the run-owned Group")
	if createTD.Status == FAIL {
		td.Stop()
	}
	createTD.HTTPBodyMustJSON(createRes, "Creating the run-owned Group")
	if createRes.JSON != nil {
		createEpoch, stateErr := conformMutationGroupState(createRes.Body, groupModel,
			path, id, owner)
		createTD.NoError(stateErr,
			"Created Group MUST be run-owned and empty")
		if stateErr == nil {
			ownedEpoch = createEpoch
			createTD.Pass("Created Group is run-owned and empty")
		}
	}
	expectedURL, urlErr := reg.URLWithPath(path)
	createTD.NoErrorStop(urlErr, "Building the run-owned Group URL MUST work")
	createTD.HTTPHeaderMustEqual(createRes, "Location",
		expectedURL.String(), "Create MUST return the exact Location")
	if createTD.Status == FAIL {
		td.Stop()
	}

	readTD := NewTD(td, "Read run-owned Group")
	readRes, readErr := HttpDo(&HttpRequest{
		Verb:       "GET",
		URL:        reg.WithPath(path),
		Headers:    xrlib.HTTPHeaders,
		Debug:      VerboseCount > 2,
		NoRedirect: true,
	})
	if readErr != nil || readRes == nil || readRes.Code != 200 {
		logConformMutationHTTPFailure(readTD, "Read GET", readRes, readErr)
	}
	readTD.NoErrorStop(readErr, "Reading the run-owned Group MUST work")
	readTD.HTTPStatusMustEqual(readRes, 200, "Reading the run-owned Group")
	readTD.HTTPBodyMustJSON(readRes, "Reading the run-owned Group")
	if readRes.JSON != nil {
		readEpoch, stateErr := conformMutationGroupState(readRes.Body, groupModel,
			path, id, owner)
		readTD.NoError(stateErr, "Read Group MUST remain run-owned and empty")
		if stateErr == nil {
			readTD.MustEqual(ownedEpoch, readEpoch,
				"Read Group epoch MUST remain unchanged")
		}
	}
	if readTD.Status == FAIL {
		td.Stop()
	}
}

func conformMutationGroupModel(model *registry.Model) (string, *registry.GroupModel) {
	for _, groupType := range SortedKeys(model.Groups) {
		groupModel := model.Groups[groupType]
		if groupModel == nil {
			continue
		}

		_, specAttributes := groupModel.GetPropsOrdered()
		labels := groupModel.Attributes["labels"]
		if labels == nil {
			labels = specAttributes["labels"]
		}
		if labels == nil || labels.ReadOnly || labels.Immutable ||
			labels.Type != MAP || labels.Item == nil ||
			labels.Item.Type != STRING || len(labels.Item.Enum) != 0 {
			continue
		}

		safe := true
		for name, attr := range groupModel.Attributes {
			if specAttributes[name] != nil {
				continue
			}
			if attr != nil && attr.Required &&
				(!attr.ReadOnly || len(attr.IfValues) != 0) {
				safe = false
				break
			}
		}
		if safe {
			return groupType, groupModel
		}
	}
	return "", nil
}

func conformMutationGroupState(body []byte, groupModel *registry.GroupModel,
	path, id, owner string) (string, error) {

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	obj := map[string]any{}
	if err := decoder.Decode(&obj); err != nil {
		return "", fmt.Errorf("invalid JSON body: %w", err)
	}

	if got, ok := obj[groupModel.Singular+"id"].(string); !ok || got != id {
		return "", fmt.Errorf("unexpected %sid", groupModel.Singular)
	}
	if got, ok := obj["xid"].(string); !ok || got != path {
		return "", fmt.Errorf("unexpected xid")
	}

	labels, ok := obj["labels"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("missing ownership labels")
	}
	if got, ok := labels[conformMutationLabel].(string); !ok || got != owner {
		return "", fmt.Errorf("ownership label changed")
	}

	epochVal, ok := obj["epoch"]
	if !ok || IsNil(epochVal) {
		return "", fmt.Errorf("missing epoch")
	}
	epoch, _, err := conformMutationUInt(epochVal)
	if err != nil {
		return "", fmt.Errorf("invalid epoch: %w", err)
	}

	for _, resourceType := range SortedKeys(groupModel.Resources) {
		countName := resourceType + "count"
		countVal, ok := obj[countName]
		if !ok || IsNil(countVal) {
			return "", fmt.Errorf("missing %s", countName)
		}
		_, count, err := conformMutationUInt(countVal)
		if err != nil {
			return "", fmt.Errorf("invalid %s: %w", countName, err)
		}
		if count != 0 {
			return "", fmt.Errorf("%s is not zero", countName)
		}

		if collection, exists := obj[resourceType]; exists &&
			!IsNil(collection) {
			collectionMap, ok := collection.(map[string]any)
			if !ok || len(collectionMap) != 0 {
				return "", fmt.Errorf("%s is not empty", resourceType)
			}
		}
	}

	return epoch, nil
}

func conformMutationUInt(value any) (string, uint64, error) {
	number, ok := value.(json.Number)
	if !ok {
		return "", 0, fmt.Errorf("must be a uinteger")
	}
	raw := number.String()
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("must be a uinteger")
	}
	return raw, parsed, nil
}

func logConformMutationHTTPFailure(td *TD, action string,
	res *HttpResponse, xErr *XRError) {

	if res == nil || res.Code == 0 {
		if xErr != nil {
			td.Log("%s transport error: %s", action, xErr)
		} else {
			td.Log("%s returned no HTTP status", action)
		}
		return
	}

	status := res.Status
	if status == "" {
		status = strconv.Itoa(res.Code)
	}
	detail := fmt.Sprintf("%s response: %s", action, status)
	if location := res.Header.Get("Location"); location != "" {
		detail += "; Location: " + location
	}
	body := strings.TrimSpace(string(res.Body))
	if body == "" {
		detail += "; body: <empty>"
	} else {
		detail += fmt.Sprintf("; body: %q", body)
	}
	td.Log(detail)
}

func cleanupConformMutationGroup(td *TD, groupModel *registry.GroupModel,
	path, id, owner, ownedEpoch string) {

	failFast := td.Config.FailFast
	td.Config.FailFast = false
	defer func() {
		td.Config.FailFast = failFast
	}()

	cleanupTD := NewTD(td, "Clean up run-owned Group")
	reg := td.GetRegistry()
	res, getErr := HttpDo(&HttpRequest{
		Verb:       "GET",
		URL:        reg.WithPath(path),
		Headers:    xrlib.HTTPHeaders,
		Debug:      VerboseCount > 2,
		NoRedirect: true,
	})
	if res == nil {
		logConformMutationHTTPFailure(cleanupTD, "Cleanup GET", res, getErr)
		cleanupTD.Fail("Cleanup GET MUST return an HTTP response")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	if res.Code == 404 {
		if ownedEpoch != "" {
			logConformMutationHTTPFailure(cleanupTD, "Cleanup GET", res, getErr)
			cleanupTD.Fail("Run-owned Group disappeared before cleanup")
		} else {
			cleanupTD.Pass("No run-owned Group remains")
		}
		return
	}
	if res.Code != 200 {
		logConformMutationHTTPFailure(cleanupTD, "Cleanup GET", res, getErr)
		cleanupTD.Fail("Cleanup GET MUST return 200 or 404")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	if res.JSON == nil {
		logConformMutationHTTPFailure(cleanupTD, "Cleanup GET", res, getErr)
		cleanupTD.Fail("Cleanup GET MUST return a JSON body")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	if ownedEpoch == "" {
		cleanupTD.Fail("Cleanup requires a confirmed run-owned creation")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}

	epoch, stateErr := conformMutationGroupState(res.Body, groupModel,
		path, id, owner)
	if stateErr != nil {
		cleanupTD.Log("Ownership proof failed: %s", stateErr)
		cleanupTD.Fail("Cleanup MUST only delete the exact run-owned Group")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	if epoch != ownedEpoch {
		cleanupTD.Log("Cleanup epoch changed: expected %s; got %s",
			ownedEpoch, epoch)
		cleanupTD.Fail("Run-owned Group changed after ownership verification")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	cleanupTD.Pass("Cleanup ownership proof MUST succeed")

	deletePath := path + "?epoch=" + url.QueryEscape(epoch)
	var deleteBody []byte
	if !reg.Capabilities.FlagEnabled("epoch") {
		deletePath = "/" + groupModel.Plural
		deleteBody = []byte(fmt.Sprintf(`{%s:{"epoch":%s}}`,
			strconv.Quote(id), epoch))
	}
	deleteRes, deleteErr := HttpDo(&HttpRequest{
		Verb:       "DELETE",
		URL:        reg.WithPath(deletePath),
		Headers:    xrlib.HTTPHeaders,
		Body:       deleteBody,
		Debug:      VerboseCount > 2,
		NoRedirect: true,
	})
	if deleteErr != nil || deleteRes == nil ||
		(deleteRes.Code != 200 && deleteRes.Code != 204) {

		logConformMutationHTTPFailure(cleanupTD, "Cleanup DELETE",
			deleteRes, deleteErr)
	}
	cleanupTD.NoError(deleteErr, "Deleting the run-owned Group MUST work")
	if deleteRes == nil {
		cleanupTD.Fail("DELETE MUST return an HTTP response")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	if deleteErr != nil || (deleteRes.Code != 200 && deleteRes.Code != 204) {
		cleanupTD.Fail("DELETE MUST return 200 or 204")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	cleanupTD.Pass("DELETE returned an allowed success status")

	verifyRes, verifyErr := HttpDo(&HttpRequest{
		Verb:       "GET",
		URL:        reg.WithPath(path),
		Headers:    xrlib.HTTPHeaders,
		Debug:      VerboseCount > 2,
		NoRedirect: true,
	})
	if verifyRes == nil {
		logConformMutationHTTPFailure(cleanupTD, "Cleanup verification GET",
			verifyRes, verifyErr)
		cleanupTD.Fail("Cleanup verification GET MUST return an HTTP response")
		cleanupTD.Msg("Run-owned Group may remain: %s", path)
		return
	}
	if verifyRes.Code != 404 {
		logConformMutationHTTPFailure(cleanupTD, "Cleanup verification GET",
			verifyRes, verifyErr)
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
		td.Skip("No Group Types defined  - leaving")
		return
	}
	groupPaths, ok := groupPathsAny.(map[string]string)
	if !ok {
		td.FailNow("reg.stuff.groupPaths != map[string]string")
	}

	if len(groupPaths) == 0 {
		td.Skip("No Group Types defined  - leaving")
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
