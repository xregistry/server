package main

import (
	"github.com/xregistry/server/cmds/xr/xrlib"
)

func TestSniffTest(td *TD) {
	reg := td.GetRegistry()
	td.Log("Server URL: %s", reg.GetServerURL())

	res, _ := reg.HttpDo(VerboseCount > 2, "GET", "", nil)
	td.Log("Body: %s", string(res.Body))

	td.HTTPStatusMustEqual(res, 200, "'GET /' MUST return 200")
	td.Must(len(res.Body) > 0, "'GET /' MUST return a non-empty body")
	td.Log("GET response:\n%s", string(res.Body))
	td.Must(res.JSON != nil, "'GET /' MUST return a JSON body")
}

func TestLoadModel(td *TD) {
	td.DependsOn(TestSniffTest)
	reg := td.GetRegistry()

	res, xErr := reg.HttpDo(VerboseCount > 2, "GET", "/model", nil)
	td.Log("Model: %s", string(res.Body))

	td.MustEqual(res.Code, 200, "'GET /model' MUST return 200")
	td.MustNotEqual(res.Body, nil, "The model MUST NOT be empty")
	td.Must(res.JSON != nil, "'GET /model' MUST return a JSON body")

	_, xErr = xrlib.ParseModel(res.Body)
	td.NoError(xErr, "Parsing model MUST work")
}
