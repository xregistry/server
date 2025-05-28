package main

import (
	"github.com/xregistry/server/cmds/xr/xrlib"
)

func TestSniffTest(td *TD) {
	reg := td.Props["xreg"].(*xrlib.Registry)
	td.Log("Server URL: %s", reg.GetServerURL())

	res, err := reg.HttpDo("GET", "", nil)
	td.NoErrorStop(err, "'GET /' should have worked: %s", err)

	if res.Code != 200 {
		td.Fail("'GET /' MUST return 200, not %d(%s)",
			res.Code, string(res.Body))
	}

	td.Must(len(res.Body) > 0, "'GET /' MUST return a non-empty body")

	if res.Body == nil {
		tmp := " <empty>"
		if len(res.Body) > 0 {
			tmp = "\n" + string(res.Body)
		}
		td.Fail("'GET /' MUST return a JSON body, not:%s", tmp)
	}

	td.Log("GET / returned 200 + JSON body")
}

func TestLoadModel(td *TD) {
	td.DependsOn(TestSniffTest)
	reg := td.Props["xreg"].(*xrlib.Registry)

	res, err := reg.HttpDo("GET", "/model", nil)
	td.NoErrorStop(err, "'GET /model' should have worked: %s", err)
	td.MustEqual(res.Code, 200, "'GET /model' MUST return 200")
	td.MustNotEqual(res.Body, nil, "The model MUST NOT be empty")

	_, err = xrlib.ParseModel(res.Body)
	td.MustEqual(err, nil, "Parsing model should work")

	// td.Log("Model:\n%s", xrlib.ToJSON(data))
	td.Fail("asd")
}
