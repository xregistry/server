package main

import (
	"bytes"
	"fmt"
	"testing"

	. "github.com/xregistry/server/common"
)

func TestTDJSONPreservesOrderAndEscaping(t *testing.T) {
	root := NewTD(nil, `root "quoted"`)
	root.Config.ShowLogs = true

	child := NewTD(root, "child")
	child.Pass("line\n\"value\"")
	root.Warn("careful")
	root.Log("<raw>")

	out := bytes.Buffer{}
	if err := printTDJSON(&out, []*TD{root}); err != nil {
		t.Fatal(err)
	}

	XEqual(t, "TD JSON", out.String(), `[
  {
    "name": "root \"quoted\"",
    "status": "PASS",
    "pass": 3,
    "fail": 0,
    "warn": 1,
    "skip": 0,
    "entries": [
      {
        "subtest": {
          "name": "child",
          "status": "PASS",
          "pass": 2,
          "fail": 0,
          "warn": 0,
          "skip": 0,
          "entries": [
            {
              "status": "PASS",
              "text": "line\n\"value\""
            }
          ]
        }
      },
      {
        "status": "WARN",
        "text": "careful"
      },
      {
        "status": "LOG",
        "text": "\u003craw\u003e"
      }
    ]
  }
]
`)
}

func TestTDJSONRejectsInvalidStatus(t *testing.T) {
	defer func() {
		XEqual(t, "Panic", fmt.Sprint(recover()), "invalid TD status: 0")
	}()
	tdJSONStatus(0)
}
