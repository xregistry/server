package tests

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	. "github.com/xregistry/server/common"
)

func TestPaginationBasic(t *testing.T) {
	reg := NewRegistry("TestPaginationBasic")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false
        }
      }
    }
  }
}
`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model)
	XHTTP(t, reg, "POST", "/", `{
  "dirs": {
    "d1": {
      "files": {
        "f1": {
          "versions": {
            "v1":{},"v2":{},"v3":{},"v4":{},"v5":{},
            "v6":{},"v7":{},"v8":{},"v9":{},"v10":{}
          }
        },
        "f2":{},"f3":{},"f4":{},"f5":{},"f6":{},"f7":{},"f8":{},"f9":{},"f10":{}
      }
    },
    "d2":{},"d3":{},"d4":{},"d5":{},"d6":{},"d7":{},"d8":{},
    "d9":{},"d10":{},"d11":{},"d12":{},"d13":{},"d14":{},
    "d15":{},"d16":{},"d17":{},"d18":{},"d19":{},"d20":{}
  }
}`, 200, `*`)

	// Path: /dirs?limit=1
	res := XHTTP(t, reg, "GET", "/dirs?limit=1", "", 200, "*")
	checkKeys(t, res.body, "d1")
	// path, rel, offset, limit, count
	checkLink(t, res, "/dirs", "first", 0, 1, 20)
	next := checkLink(t, res, "/dirs", "next", 1, 1, 20)
	checkLink(t, res, "/dirs", "last", 19, 1, 20)
	noMoreLinks(t, res)

	// test "next"
	res = XHTTP(t, reg, "GET", next, "", 200, "*")
	checkKeys(t, res.body, "d10") // alpabetical, not numeric
	// path, rel, offset, limit, count
	checkLink(t, res, "/dirs", "first", 0, 1, 20)
	checkLink(t, res, "/dirs", "prev", 0, 1, 20)
	checkLink(t, res, "/dirs", "next", 2, 1, 20)
	last := checkLink(t, res, "/dirs", "last", 19, 1, 20)
	noMoreLinks(t, res)

	// test "last"
	res = XHTTP(t, reg, "GET", last, "", 200, "*")
	checkKeys(t, res.body, "d9") // alphabetical, not numeric
	// path, rel, offset, limit, count
	checkLink(t, res, "/dirs", "first", 0, 1, 20)
	checkLink(t, res, "/dirs", "prev", 18, 1, 20)
	checkLink(t, res, "/dirs", "last", 19, 1, 20)
	noMoreLinks(t, res)

	// test bigger limit
	res = XHTTP(t, reg, "GET", "/dirs?limit=5", "", 200, "*")
	checkKeys(t, res.body, "d1,d10,d11,d12,d13")
	// path, rel, offset, limit, count
	checkLink(t, res, "/dirs", "first", 0, 5, 20)
	next = checkLink(t, res, "/dirs", "next", 5, 5, 20)
	checkLink(t, res, "/dirs", "last", 15, 5, 20)
	noMoreLinks(t, res)

	// test 5's next
	res = XHTTP(t, reg, "GET", next, "", 200, "*")
	checkKeys(t, res.body, "d14,d15,d16,d17,d18")
	// path, rel, offset, limit, count
	checkLink(t, res, "/dirs", "first", 0, 5, 20)
	next = checkLink(t, res, "/dirs", "next", 10, 5, 20)
	next = checkLink(t, res, "/dirs", "prev", 0, 5, 20)
	checkLink(t, res, "/dirs", "last", 15, 5, 20)
	noMoreLinks(t, res)

	// test huge limit - the whole (filtered) collection fits on this one
	// page, so no pagination Link headers (not even "first"/"last") are
	// sent
	res = XHTTP(t, reg, "GET", "/dirs?limit=10000", "", 200, "*")
	checkKeys(t, res.body, "d1,d10,d11,d12,d13,d14,d15,d16,d17,d18,d19,"+
		"d2,d20,d3,d4,d5,d6,d7,d8,d9")
	noMoreLinks(t, res)

	// Test resource
	res = XHTTP(t, reg, "GET", "/dirs/d1/files?offset=3&limit=3", "", 200, "*")
	checkKeys(t, res.body, "f3,f4,f5")
	// path, rel, offset, limit, count
	checkLink(t, res, "/dirs/d1/files", "first", 0, 3, 10)
	checkLink(t, res, "/dirs/d1/files", "prev", 0, 3, 10)
	checkLink(t, res, "/dirs/d1/files", "next", 6, 3, 10)
	checkLink(t, res, "/dirs/d1/files", "last", 9, 3, 10)
	noMoreLinks(t, res)

	// Test versions
	res = XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?offset=3&limit=3",
		"", 200, "*")
	checkKeys(t, res.body, "v3,v4,v5")
	// path, rel, offset, limit, count
	checkLink(t, res, "/dirs/d1/files/f1/versions", "first", 0, 3, 10)
	checkLink(t, res, "/dirs/d1/files/f1/versions", "prev", 0, 3, 10)
	checkLink(t, res, "/dirs/d1/files/f1/versions", "next", 6, 3, 10)
	checkLink(t, res, "/dirs/d1/files/f1/versions", "last", 9, 3, 10)
	noMoreLinks(t, res)

	// Test offset > count - results shrunk before I got to the end
	res = XHTTP(t, reg, "GET", "/dirs?offset=25&limit=2", "", 200, "{}\n")
	// path, rel, offset, limit, count
	checkLink(t, res, "/dirs", "first", 0, 2, -1)
	noMoreLinks(t, res)

	// Test sorting
	res = XHTTP(t, reg, "GET", "/dirs?limit=2&sort=dirid=desc", "", 200, `{
  "d9": {
    "dirid": "d9",
    "self": "http://localhost:8181/dirs/d9",
    "xid": "/dirs/d9",
    "epoch": 1,
    "createdat": "2026-09-26T12:58:59.918723011Z",
    "modifiedat": "2026-09-26T12:58:59.918723011Z",

    "filesurl": "http://localhost:8181/dirs/d9/files",
    "filescount": 0
  },
  "d8": {
    "dirid": "d8",
    "self": "http://localhost:8181/dirs/d8",
    "xid": "/dirs/d8",
    "epoch": 1,
    "createdat": "2026-09-26T12:58:59.918723011Z",
    "modifiedat": "2026-09-26T12:58:59.918723011Z",

    "filesurl": "http://localhost:8181/dirs/d8/files",
    "filescount": 0
  }
}
`)
	// path, rel, offset, limit, count
	checkLink(t, res, "/dirs?limit=2&offset=0&sort=dirid%3Ddesc", "first",
		-1, -1, 20)
	checkLink(t, res, "/dirs?limit=2&offset=2&sort=dirid%3Ddesc", "next",
		-1, -1, 20)
	checkLink(t, res, "/dirs?limit=2&offset=18&sort=dirid%3Ddesc", "last",
		-1, -1, 20)
	noMoreLinks(t, res)

	// Test filtering
	res = XHTTP(t, reg, "GET", "/dirs?limit=2&filter=files.fileid&sort=dirid=asc", "", 200, `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2026-09-26T13:10:06.694807235Z",
    "modifiedat": "2026-09-26T13:10:06.694807235Z",

    "filesurl": "http://localhost:8181/dirs/d1/files?filter=fileid",
    "filescount": 10
  }
}
`)
	noMoreLinks(t, res)
}

func TestPaginationErrors(t *testing.T) {
	reg := NewRegistry("TestPaginationErrors")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false
        }
      }
    }
  }
}
`

	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model)
	XHTTP(t, reg, "POST", "/", `{
  "dirs": {
    "d1": {
      "files": {
        "f1": {
          "versions": {"v1":{}}
        },
        "f2":{},"f3":{},"f4":{},"f5":{},"f6":{},"f7":{},"f8":{},"f9":{},"f10":{}
      }
    },
    "d2":{},"d3":{},"d4":{}, "d5":{},"d6":{},"d7":{},"d8":{},"d9":{},"d10":{}
  }
}`, 200, `*`)

	XHTTP(t, reg, "GET", "/dirs?limit", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "\"limit\" value () must be an unsigned integer > 0.",
  "subject": "/dirs?limit",
  "args": {
    "error_detail": "\"limit\" value () must be an unsigned integer > 0"
  },
  "instance": "c749cd1de0eb4cb6",
  "source": "9f636df17262:registry:info:869"
}
`)
	XHTTP(t, reg, "GET", "/dirs?limit=", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "\"limit\" value () must be an unsigned integer > 0.",
  "subject": "/dirs?limit=",
  "args": {
    "error_detail": "\"limit\" value () must be an unsigned integer > 0"
  },
  "instance": "c749cd1de0eb4cb6",
  "source": "9f636df17262:registry:info:869"
}
`)
	XHTTP(t, reg, "GET", "/dirs?limit=0", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "\"limit\" value (0) must be an unsigned integer > 0.",
  "subject": "/dirs?limit=0",
  "args": {
    "error_detail": "\"limit\" value (0) must be an unsigned integer > 0"
  },
  "instance": "0cf823badd824fd3",
  "source": "9f636df17262:registry:info:869"
}
`)
	XHTTP(t, reg, "GET", "/dirs?limit=abc", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "\"limit\" value (abc) must be an unsigned integer > 0.",
  "subject": "/dirs?limit=abc",
  "args": {
    "error_detail": "\"limit\" value (abc) must be an unsigned integer > 0"
  },
  "instance": "c749cd1de0eb4cb6",
  "source": "9f636df17262:registry:info:869"
}
`)
	XHTTP(t, reg, "GET", "/dirs?limit=-1", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "\"limit\" value (-1) must be an unsigned integer > 0.",
  "subject": "/dirs?limit=-1",
  "args": {
    "error_detail": "\"limit\" value (-1) must be an unsigned integer > 0"
  },
  "instance": "c749cd1de0eb4cb6",
  "source": "9f636df17262:registry:info:869"
}
`)
	XHTTP(t, reg, "GET", "/dirs?limit=99999999999999999999999999", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "\"limit\" value (99999999999999999999999999) must be an unsigned integer > 0.",
  "subject": "/dirs?limit=99999999999999999999999999",
  "args": {
    "error_detail": "\"limit\" value (99999999999999999999999999) must be an unsigned integer > 0"
  },
  "instance": "a6b729bd1ca24721",
  "source": "9f636df17262:registry:info:869"
}
`)
	XHTTP(t, reg, "GET", "/dirs?offset=-1", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "\"offset\" value (-1) must be an unsigned integer.",
  "subject": "/dirs?offset=-1",
  "args": {
    "error_detail": "\"offset\" value (-1) must be an unsigned integer"
  },
  "instance": "eaa40df859144c4f",
  "source": "9f636df17262:registry:info:880"
}
`)
	XHTTP(t, reg, "GET", "/dirs?offset=-1&limit=0", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "\"limit\" value (0) must be an unsigned integer > 0.",
  "subject": "/dirs?offset=-1&limit=0",
  "args": {
    "error_detail": "\"limit\" value (0) must be an unsigned integer > 0"
  },
  "instance": "74b6aae9c48d4ad6",
  "source": "9f636df17262:registry:info:869"
}
`)

	XHTTP(t, reg, "GET", "/?limit=2", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Can't paginate a non-collection result set.",
  "subject": "/?limit=2",
  "args": {
    "error_detail": "Can't paginate a non-collection result set"
  },
  "instance": "3b9a3cbbeb4d40b3",
  "source": "9f636df17262:registry:info:861"
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1?limit=2", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Can't paginate a non-collection result set.",
  "subject": "/dirs/d1?limit=2",
  "args": {
    "error_detail": "Can't paginate a non-collection result set"
  },
  "instance": "85ef212da7004b22",
  "source": "9f636df17262:registry:info:861"
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?limit=2", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Can't paginate a non-collection result set.",
  "subject": "/dirs/d1/files/f1?limit=2",
  "args": {
    "error_detail": "Can't paginate a non-collection result set"
  },
  "instance": "443472164ee1411f",
  "source": "9f636df17262:registry:info:861"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1?limit=2", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Can't paginate a non-collection result set.",
  "subject": "/dirs/d1/files/f1/versions/v1?limit=2",
  "args": {
    "error_detail": "Can't paginate a non-collection result set"
  },
  "instance": "49e200aea5d94a24",
  "source": "9f636df17262:registry:info:861"
}
`)

	XHTTP(t, reg, "GET", "/dirs?offset=2", "", 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "\"offset\" can't be used without \"limit\".",
  "subject": "/dirs?offset=2",
  "args": {
    "error_detail": "\"offset\" can't be used without \"limit\""
  },
  "instance": "99092c9a063a40e2",
  "source": "9f636df17262:registry:info:888"
}
`)
}

func noMoreLinks(t *testing.T, res *HTTPResult) {
	t.Helper()

	links := res.Header.Values("Link")
	extra := ""
	for _, l := range links {
		if !strings.Contains(l, "rel=xregistry-root") {
			extra += "\n" + l
		}
	}
	if len(extra) > 0 {
		t.Fatalf("Extra Link headers:%s", extra)
	}
}

func checkLink(t *testing.T, res *HTTPResult, path, rel string,
	offset, limit, count int) string {

	t.Helper()

	daURL := path
	if limit >= 0 {
		daURL = AddQuery(daURL, fmt.Sprintf("limit=%d", limit))
	}

	if offset >= 0 {
		daURL = AddQuery(daURL, fmt.Sprintf("offset=%d", offset))
	}

	exp := "<http://localhost:8181" + daURL + ">;rel=" + rel
	if count >= 0 {
		exp += fmt.Sprintf(";count=%d", count)
	}

	links := res.Header.Values("Link")
	for _, link := range links {
		if link == exp {
			res.Header["Link"] = slices.DeleteFunc(res.Header["Link"],
				func(s string) bool {
					return s == exp
				})
			return daURL
		}
	}
	t.Logf("Got headers:\n%s", strings.Join(links, "\n"))
	t.Fatalf("Missing Link header: %s", exp)

	return daURL
}

func checkKeys(t *testing.T, body string, keys string) {
	t.Helper()

	daMap := map[string]any{}
	XNoErr(t, Unmarshal([]byte(body), &daMap))
	mapKeys := SortedKeys(daMap)
	XEqual(t, "", strings.Join(mapKeys, ","), keys)
}
