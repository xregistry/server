package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

const conformMutationEpochSentinel = "xr-conform-create-only"

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
	td.NoErrorStop(xErr, "Retrieving capabilities MUST work")
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
	td.NoErrorStop(xErr, "Retrieving the model MUST work")

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
	preflightRes, xErr := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     reg.WithPath(path),
		Headers: xrlib.HTTPHeaders,
		Debug:   VerboseCount > 2,
	})
	preflightTD.LogHTTP(preflightRes)
	if preflightRes == nil {
		preflightTD.NoErrorStop(xErr, "Preflight GET MUST work")
	}
	preflightTD.Must(preflightRes != nil,
		"Preflight GET MUST return an HTTP response")
	preflightTD.StopOnFail()
	preflightTD.HTTPStatusMustEqual(preflightRes, 404,
		"Preflight GET of mutation destination")
	preflightTD.StopOnFail()

	createTD := NewTD(td, "Create run-owned Group")
	confirmedCreate := (*HttpResponse)(nil)
	defer func() {
		if confirmedCreate == nil {
			td.Msg("Run-owned Group may remain: %s", path)
			return
		}
		cleanupConformMutationGroup(td, groupModel, confirmedCreate,
			capabilities.FlagEnabled("epoch"))
	}()

	createRes, xErr := HttpDo(&HttpRequest{
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
	createTD.NoErrorStop(xErr,
		"Creating the run-owned Group MUST work")
	createTD.HTTPStatusMustEqual(createRes, 201,
		"Creating the run-owned Group")
	createTD.StopOnFail()
	createTD.HTTPBodyMustJSON(createRes, "Creating the run-owned Group")
	createTD.StopOnFail()
	conformMutationGroupState(createTD, createRes, groupModel, path, id)
	createTD.StopOnFail()
	confirmedCreate = createRes
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
	readTD.ObjReqMustEq(readRes.JSON, "epoch", createRes.JSON["epoch"])
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

		// missing lists attributes that prevent using this Group.
		missing := []string{}
		attributes := groupModel.GetBaseAttributes()
		for _, name := range SortedKeys(attributes) {
			attr := attributes[name]
			if attr == nil || !attr.Required || !IsNil(attr.Default) {
				continue
			}
			if attr.ReadOnly || name == groupModel.Singular+"id" ||
				(attr.Internals != nil &&
					(name == "createdat" || name == "modifiedat")) {

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
	groupModel *registry.GroupModel, path, id string) {

	td.ObjReqMustEq(res.JSON, groupModel.Singular+"id", id)
	td.ObjReqMustEq(res.JSON, "xid", path)
	td.ObjReqMustGe(res.JSON, "epoch", 0)
	td.ObjReqMustLt(res.JSON, "epoch", float64(1<<53))

	for _, resourceType := range SortedKeys(groupModel.Resources) {
		countName := resourceType + "count"
		td.ObjReqMustEq(res.JSON, countName, 0)
		td.ObjPropMustNotExist(res.JSON, resourceType)
	}
}

func cleanupConformMutationGroup(td *TD, groupModel *registry.GroupModel,
	confirmedCreate *HttpResponse, useEpochFlag bool) {

	cleanupTD := NewTD(td, "Clean up run-owned Group")
	idName := groupModel.Singular + "id"
	id := confirmedCreate.JSON[idName].(string)
	path := confirmedCreate.JSON["xid"].(string)
	expectedEpoch := confirmedCreate.JSON["epoch"]
	cleaned := false
	defer func() {
		if !cleaned {
			cleanupTD.Msg("Run-owned Group may remain: %s", path)
		}
	}()

	reg := td.GetRegistry()
	res, xErr := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     reg.WithPath(path),
		Headers: xrlib.HTTPHeaders,
		Debug:   VerboseCount > 2,
	})
	cleanupTD.LogHTTP(res)
	if res == nil {
		cleanupTD.NoErrorStop(xErr, "Cleanup GET MUST work")
		cleanupTD.Fail("Cleanup GET MUST return an HTTP response")
		return
	}
	if res.Code == 404 {
		cleanupTD.Fail("Run-owned Group disappeared before cleanup")
		return
	}
	if res.Code != 200 {
		cleanupTD.Fail("Cleanup GET MUST return 200 or 404")
		return
	}
	if res.JSON == nil {
		cleanupTD.Fail("Cleanup GET MUST return a JSON body")
		return
	}

	conformMutationGroupState(cleanupTD, res, groupModel, path, id)
	cleanupTD.StopOnFail()
	cleanupTD.ObjReqMustEq(res.JSON, "epoch", expectedEpoch)
	cleanupTD.StopOnFail()
	cleanupTD.Pass("Cleanup state proof MUST succeed")

	epochString := strconv.FormatFloat(expectedEpoch.(float64), 'f', -1, 64)
	deletePath := path + "?epoch=" + url.QueryEscape(epochString)
	var deleteBody []byte
	if !useEpochFlag {
		deletePath = "/" + groupModel.Plural
		deleteBody = []byte(fmt.Sprintf(`{%s:{"epoch":%s}}`,
			strconv.Quote(id), epochString))
	}
	deleteRes, xErr := HttpDo(&HttpRequest{
		Verb:    "DELETE",
		URL:     reg.WithPath(deletePath),
		Headers: xrlib.HTTPHeaders,
		Body:    deleteBody,
		Debug:   VerboseCount > 2,
	})
	cleanupTD.LogHTTP(deleteRes)
	if deleteRes == nil {
		cleanupTD.Fail("DELETE MUST return an HTTP response")
		return
	}
	cleanupTD.NoErrorStop(xErr,
		"Deleting the run-owned Group MUST work")
	if deleteRes.Code != 200 && deleteRes.Code != 204 {
		cleanupTD.Fail("DELETE MUST return 200 or 204")
		return
	}
	cleanupTD.Pass("DELETE returned an allowed success status")

	verifyRes, xErr := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     reg.WithPath(path),
		Headers: xrlib.HTTPHeaders,
		Debug:   VerboseCount > 2,
	})
	cleanupTD.LogHTTP(verifyRes)
	if verifyRes == nil {
		cleanupTD.NoErrorStop(xErr,
			"Cleanup verification GET MUST work")
		cleanupTD.Fail("Cleanup verification GET MUST return an HTTP response")
		return
	}
	if verifyRes.Code != 404 {
		cleanupTD.Fail("Cleanup verification GET MUST return 404")
		return
	}
	cleaned = true
	cleanupTD.Pass("DELETE removed the exact run-owned Group")
}
