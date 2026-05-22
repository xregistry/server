package main

import (
// . "github.com/xregistry/server/common"
)

func TestSniff(td *TD) {
	reg := td.GetRegistry()
	td.Log("Server URL: %s", reg.GetServerURL())

	res, _ := reg.HttpDo(VerboseCount > 2, "GET", "", nil)
	td.HTTPStatusMustEqual(res, 200, "GET /")
	td.HTTPBodyMustJSON(res, "GET /")
}
