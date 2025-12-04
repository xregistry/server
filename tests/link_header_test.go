package tests

import (
	"testing"

	"github.com/xregistry/server/registry"
)

func TestLinkHeader(t *testing.T) {
	reg := NewRegistry("TestLinkHeader")
	defer PassDeleteReg(t, reg)

	// Test Link header on registry root GET
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "Link header on registry root",
		URL:        "/",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Link:<http://localhost:8181>;rel=xregistry-root"},
		ResBody:    "*",
	})

	// Test Link header on error response
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "Link header on error",
		URL:        "/notfound",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       404,
		ResHeaders: []string{"Link:<http://localhost:8181>;rel=xregistry-root"},
		ResBody:    "*",
	})
}

func TestLinkHeaderMultiRegistry(t *testing.T) {
	reg := NewRegistry("TestLinkHeaderReg1")
	defer PassDeleteReg(t, reg)

	reg2, _ := registry.NewRegistry(nil, "TestLinkHeaderReg2")
	reg2.SaveAllAndCommit()
	defer func() {
		reg2.Delete()
		reg2.SaveAllAndCommit()
	}()

	// Test Link header with multi-registry path (reg- prefix)
	XCheckHTTP(t, reg2, &HTTPTest{
		Name:       "Link header with reg- prefix",
		URL:        "/reg-TestLinkHeaderReg2",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code:       200,
		ResHeaders: []string{"Link:<http://localhost:8181/reg-TestLinkHeaderReg2>;rel=xregistry-root"},
		ResBody:    "*",
	})
}
