package common

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"
)

// Test ResolvePath
func TestResolvePath(t *testing.T) {
	type ResolvePathTest struct {
		Base   string
		Next   string
		Result string
	}

	tests := []ResolvePathTest{
		{"", "", ""},

		// FILE base
		{"", "file.txt", "file.txt"},
		{"dir1/dir2/file1", "file.txt", "dir1/dir2/file.txt"},
		{"dir1/dir2/file1", "/file.txt", "/file.txt"},
		{"dir1/dir2/file1", "d:/file.txt", "d:/file.txt"},
		{"dir1/dir2/file1", "http://foo.com", "http://foo.com"},
		{"d1/d2/f1", "https://foo.com", "https://foo.com"},
		{"d1/d2/f1", "#abc", "d1/d2/f1#abc"},
		{"d1/d2/f1", "file2#abc", "d1/d2/file2#abc"},
		{"dir1/dir2/file1", "./file2#abc", "dir1/dir2/file2#abc"},
		{"dir1/dir2/file1", "/file2", "/file2"},
		{"dir1/dir2/file1", "/file2#abc", "/file2#abc"},
		{"dir1/dir2/file1/", "file2#abc", "dir1/dir2/file1/file2#abc"},
		{"/", "file2#abc", "/file2#abc"},
		{"", "file2#abc", "file2#abc"},
		{"#foo", "file2#abc", "file2#abc"},
		{"f1#foo", "file2#abc", "file2#abc"},
		{"d1/f1#foo", "file2#abc", "d1/file2#abc"},
		{"d1/#foo", "file2#abc", "d1/file2#abc"},
		{"d1/#foo", "/file2#abc", "/file2#abc"},
		{"d1/#foo", "./file2#abc", "d1/file2#abc"},
		{"/d1/#foo", "/file2#abc", "/file2#abc"},
		{"./d1/#foo", "/file2#abc", "/file2#abc"},
		{"./d1/#foo", "./file2#abc", "d1/file2#abc"},
		{"./d1#foo", "./file2#abc", "file2#abc"},
		{"/d1/d2/../f3", "./file2", "/d1/file2"},
		{"/d1/../f3", "./file2", "/file2"},
		{"d1/../f3", "./file2", "file2"},
		{"d1/d2/../d3/../../f3", "file2", "file2"},
		{"d1/d2/d3/../../f3", "../file2", "file2"},
		{"d1/d2/d3/././f3", "file2", "d1/d2/d3/file2"},
		{"d1/d2/d3/.././f3", "file2", "d1/d2/file2"},
		{"../d1/d2/d3/.././f3", "file2", "../d1/d2/file2"},
		{"../d1//d2///d3////.././f3", "file2", "../d1/d2/file2"},
		{"d1/d2/..", "file2", "d1/file2"},
		{"d1/d2/.", "file2", "d1/d2/file2"},
		{"d1/d2/...", "file2", "d1/d2/file2"},

		// HTTP base
		{"http://s1.com/dir1/file",
			"https://foo.com/dir2/file2",
			"https://foo.com/dir2/file2"},
		{"http://s1.com/dir1/file",
			"file2",
			"http://s1.com/dir1/file2"},
		{"http://s1.com/dir1/file",
			"./file2",
			"http://s1.com/dir1/file2"},
		{"http://s1.com/dir1/",
			"./file2",
			"http://s1.com/dir1/file2"},
		{"http://s1.com/dir1/file",
			"",
			"http://s1.com/dir1/file"},
		{"http://s1.com/dir1/file",
			"/file2",
			"http://s1.com/file2"},

		{"http://s1.com/dir1/file",
			"#abc",
			"http://s1.com/dir1/file#abc"},
		{"http://s1.com/dir1/",
			"#abc",
			"http://s1.com/dir1/#abc"},
		{"http://s1.com",
			"file#abc",
			"http://s1.com/file#abc"},
		{"http://s1.com#def",
			"file#abc",
			"http://s1.com/file#abc"},
		{"http://s1.com/d1#def",
			"file#abc",
			"http://s1.com/file#abc"},
		{"http://s1.com/d1/#def",
			"file#abc",
			"http://s1.com/d1/file#abc"},
		{"http://s1.com/d1/d2/..",
			"file#abc",
			"http://s1.com/d1/file#abc"},
		{"http://s1.com/d1/d2/.",
			"file#abc",
			"http://s1.com/d1/d2/file#abc"},
		{"http://s1.com/d1/d2/...",
			"file#abc",
			"http://s1.com/d1/d2/file#abc"},
	}

	for _, test := range tests {
		got := ResolvePath(test.Base, test.Next)
		if got != test.Result {
			t.Fatalf("\n%q + %q:\nExp: %s\nGot: %s\n", test.Base, test.Next,
				test.Result, got)
		}
	}
}

func TestJSONPointer(t *testing.T) {
	data := map[string]any{
		"":    123,
		"str": "hello",
		"a~b": 234,
		"b/c": 345,
		"arr": []int{1, 2, 3},
		"obj": map[string]any{
			"":      666,
			"nil":   nil,
			"obj_a": 222,
			"obj_arr": []any{
				map[string]any{
					"o1": 333,
				},
				map[string]any{
					"o2": 444,
				},
			},
		},
	}

	type Test struct {
		Json   any
		Path   string
		Result any
	}

	tests := []Test{
		{data, ``, data},
		{data, `/`, data[""]},
		{data, `str`, data["str"]},
		{data, `/str`, data["str"]},
		{data, `/a~0b`, data["a~b"]},
		{data, `/b~1c`, data["b/c"]},
		{data, `/arr`, data["arr"]},
		{data, `/arr/0`, (data["arr"].([]int))[0]},
		{data, `/arr/2`, (data["arr"].([]int))[2]},
		{data, `/obj`, data["obj"]},
		{data, `/obj/`, 666},
		{data, `/obj/nil`, nil},
		{data, `/obj/obj_a`, 222},
		{data, `/obj/obj_arr`, ((data["obj"]).(map[string]any))["obj_arr"]},
		{data, `/obj/obj_arr/0`, (((data["obj"]).(map[string]any))["obj_arr"]).([]any)[0]},

		{data, `/obj/obj_arr/0/o1`, 333},
		{data, `/obj/obj_arr/1/o2`, 444},

		{data, `x`, `Attribute "x" not found`},
		{data, `/x`, `Attribute "x" not found`},
		{data, `/arr/`, `Index "/arr/" must be an integer`},
		{data, `/arr/foo`, `Index "/arr/foo" must be an integer`},
		{data, `/arr/-1`, `Index "/arr/-1" is out of bounds(0-2)`},
		{data, `/arr/3`, `Index "/arr/3" is out of bounds(0-2)`},
		{data, `/obj/obj_arr/`, `Index "/obj/obj_arr/" must be an integer`},
		{data, `/obj/obj_arr/o1/`, `Index "/obj/obj_arr/o1" must be an integer`},
		{data, `/obj/obj_arr/0/ox/`, `Attribute "obj/obj_arr/0/ox" not found`},
		{data, `/obj/obj_arr/1/o2/`, `Can't step into a type of "int", at: /obj/obj_arr/1/o2/`},
		{data, `/obj/obj_arr/2`, `Index "/obj/obj_arr/2" is out of bounds(0-1)`},
	}

	for _, test := range tests {
		res, err := GetJSONPointer(test.Json, test.Path)
		if err != nil {
			if err.Error() != test.Result {
				t.Fatalf("Test: %s\nExp: %s\nErr: %s", test.Path, test.Result,
					err)
			}
		} else if ToJSON(res) != ToJSON(test.Result) {
			t.Fatalf("Test: %s\nExp: %s\nGot: %s", test.Path,
				ToJSON(test.Result), ToJSON(res))
		}
	}
}

type FSHandler struct {
	Files map[string]string
}

func (h *FSHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// path, _ := strings.CutPrefix(req.URL.Path, "/")
	if file, ok := h.Files[req.URL.Path]; !ok {
		res.WriteHeader(404)
	} else {
		res.Write([]byte(file))
	}
}

// Test ProcessIncludes
func TestProcessIncludes(t *testing.T) {
	// Setup HTTP server
	httpPaths := map[string]string{
		"/empty":        "",
		"/notjson":      "hello there",
		"/emptyjson":    "{}",
		"/onelevel":     `{"foo":"bar","foo6":666}`,
		"/twolevel":     `{"foo":"bar3","foo6":{"bar":666}}`,
		"/twoarray":     `{"foo":"bar","foo6":[{"x":"y"},{"bar":667}]}`,
		"/nonfoo":       `{"bar":"zzz","foo":"qqq"}`,
		"/nest1":        `{"foo":"bar1","$include":"onelevel"}`,
		"/nest2":        `{"foo":"bar1","$include":"twoarray#/foo6/1"}`,
		"/nest3":        `{"$include": "twoarray#/foo6/1","f3":"bar"}`,
		"/nest3.1":      `{"$include": "/onelevel"}`,
		"/nest3.1.f":    `{"$include": "./onelevel"}`,
		"/nest3.1.f2":   `{"$include": "onelevel"}`,
		"/nest/nest4":   `{"foo":"bar1","$include":"/onelevel"}`,
		"/nest/nest4.f": `{"foo":"bar1","$include":"../onelevel"}`,
		"/nest/nest5":   `{"foo":"bar2","$include":"/nest/nest4"}`,
		"/nest/nest5.f": `{"foo":"bar2","$include":"../nest/nest4.f"}`,
		"/nest/nest6":   `{"foo":"bar2","$include":"http://localhost:4567/nest/nest4"}`,

		"/err1": `{"$include": "empty"}`,
		"/err2": `{"$include": "notjson"}`,
		"/err3": `{"$include": "/err3"}`,
		"/err4": `{"$include": "twolevel/bar"}`,

		"/nest7":      `{"$includes": []}`,
		"/nest7.err1": `{"$include": []}`,
		"/nest7.err2": `{"$includes": [1,2,3]}`,
		"/nest7.err3": `{"$include": "foo", "$includes": []}`,

		"/nest8":  `{"$includes": [ "onelevel", "twolevel" ]}`,
		"/nest9":  `{"$includes": [ "onelevel", "twolevel" ], "foo":"xxx"}`,
		"/nest10": `{"$includes": [ "nonfoo", "onelevel" ], "foo":"xxx"}`,
	}
	if IsPortInUse(4567) {
		t.Fatal("Port 4567 is in-use - kill it")
	}
	server := &http.Server{Addr: ":4567", Handler: &FSHandler{httpPaths}}
	go server.ListenAndServe()

	// Setup our local dir structure
	/*
		files := map[string]string{
			"empty":     "",
			"emptyjson": "{}",
			"simple": `{"foo":"bar"}`,
		}
	*/
	dir, _ := os.MkdirTemp("", "test_xreg")
	defer func() {
		os.RemoveAll(dir)
	}()
	for file, data := range httpPaths {
		os.MkdirAll(dir+"/"+path.Dir(file), 0777)
		os.WriteFile(dir+"/"+file, []byte(data), 0666)
	}

	// Wait for server
	for {
		if _, err := http.Get("http://localhost:4567/"); err == nil {
			break
		}
	}

	type Test struct {
		Path   string // filename or http url to json file
		Result string // json or error msg
	}

	tests := []Test{
		{"empty", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: error parsing JSON: path '': unexpected end of input.",
  "args": {
    "error_detail": "error parsing JSON: path '': unexpected end of input"
  },
  "source": "xxx"
}`},
		{"emptyjson", "{}"},

		{"onelevel", httpPaths["/onelevel"]},
		{"http:/onelevel", httpPaths["/onelevel"]},

		{"nest1", `{"foo":"bar1","foo6":666}`},
		{"http:/nest1", `{"foo":"bar1","foo6":666}`},

		{"nest2", `{"bar":667,"foo":"bar1"}`},
		{"http:/nest2", `{"bar":667,"foo":"bar1"}`},

		{"nest3", `{"bar":667,"f3":"bar"}`},
		{"http:/nest3", `{"bar":667,"f3":"bar"}`},

		{"nest3.1.f", `{"foo":"bar","foo6":666}`},
		{"http:/nest3.1", `{"foo":"bar","foo6":666}`},
		{"http:/nest3.1.f2", `{"foo":"bar","foo6":666}`},

		{"nest/nest4.f", `{"foo":"bar1","foo6":666}`},
		{"http:/nest/nest4", `{"foo":"bar1","foo6":666}`},
		{"http:/nest/nest4.f", `{"foo":"bar1","foo6":666}`},

		{"nest/nest5.f", `{"foo":"bar2","foo6":666}`},
		{"http:/nest/nest5", `{"foo":"bar2","foo6":666}`},

		{"nest/nest6", `{"foo":"bar2","foo6":666}`},
		{"http:/nest/nest6", `{"foo":"bar2","foo6":666}`},

		{"nest7", `{}`},

		{"nest7.err1", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: error processing JSON: In \"/tmp/nest7.err1\", $include value isn't a string.",
  "args": {
    "error_detail": "error processing JSON: In \"/tmp/nest7.err1\", $include value isn't a string"
  },
  "source": "xxx"
}`},
		{"nest7.err2", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: error processing JSON: In \"/tmp/nest7.err2\", $includes contains a non-string value (1).",
  "args": {
    "error_detail": "error processing JSON: In \"/tmp/nest7.err2\", $includes contains a non-string value (1)"
  },
  "source": "xxx"
}`},
		{"nest7.err3", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: error processing JSON: In \"/tmp/nest7.err3\", both $include and $includes is not allowed.",
  "args": {
    "error_detail": "error processing JSON: In \"/tmp/nest7.err3\", both $include and $includes is not allowed"
  },
  "source": "xxx"
}`},
		{"http:/nest7.err1", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: error processing JSON: In \"http://localhost:4567/nest7.err1\", $include value isn't a string.",
  "args": {
    "error_detail": "error processing JSON: In \"http://localhost:4567/nest7.err1\", $include value isn't a string"
  },
  "source": "xxx"
}`},

		{"nest8", `{"foo":"bar","foo6":666}`},
		{"nest9", `{"foo":"xxx","foo6":666}`},
		{"nest10", `{"bar":"zzz","foo":"xxx","foo6":666}`},
	}

	mask := regexp.MustCompile(`"[^"]*/test_xreg[^/]*`)
	/*
			m1 := `{
		  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
		  "subject": "/",
		  "title": "There was an error in the model definition provided: `
			m2 := `"
		}`
	*/

	for i, test := range tests {
		t.Logf("Test #: %d", i)
		t.Logf("  Path: %s", test.Path)
		var buf []byte
		var err error
		if strings.HasPrefix(test.Path, "http:") {
			test.Path = "http://localhost:4567" + test.Path[5:]
			var res *http.Response
			if res, err = http.Get(test.Path); err == nil {
				if res.StatusCode != 200 {
					err = fmt.Errorf("Err %q: %s", test.Path, res.Status)
				} else {
					buf, err = io.ReadAll(res.Body)
					res.Body.Close()
				}
			}
		} else {
			test.Path = dir + "/" + test.Path
			buf, err = os.ReadFile(test.Path)
		}
		if err != nil {
			t.Fatal(err.Error())
		}
		buf, xErr := ProcessIncludes(test.Path, buf,
			!strings.HasPrefix(test.Path, "http"))
		if xErr != nil {
			buf = []byte(xErr.String())
		}
		exp := test.Result
		/*
			if exp[0] != '{' {
				exp = m1 + JSONEscape(test.Result) + m2
			}
		*/
		exp = string(mask.ReplaceAll([]byte(exp), []byte("\"/tmp")))
		buf = mask.ReplaceAll(buf, []byte("\"/tmp"))

		XEqual(t, test.Path+" Orig: "+string(buf), string(buf), exp)
		/*
			if string(buf) != exp {
				t.Fatalf("\nPath: %s\nExp: %s<<\nGot: %s<<",
					test.Path, exp, string(buf))
			}
		*/
	}

	server.Shutdown(context.Background())
}

// Testing a match for "*" wildcard
func TestMatch(t *testing.T) {
	type Test struct {
		Pattern string
		String  string
		Pass    bool
	}

	tests := []Test{
		// Lots of dups but that's ok, it feels more complex this way :-)
		{"", "", true},
		{"*", "", true},
		{"text/plain", "text/plain", true},
		{"*", "text/plain", true},
		{"**", "text/plain", true},
		{"text/*", "text/plain", true},
		{"text/**", "text/plain", true},
		{"*/plain", "text/plain", true},
		{"**/plain", "text/plain", true},
		{"text/*plain", "text/plain", true},
		{"text/*lain", "text/plain", true},
		{"*/plain", "text/plain", true},
		{"*t/*lain", "text/plain", true},
		{"*t*e*x*t*/*p*l*a*i*n*", "text/plain", true},

		{"", "t", false},
		{"t", "", false},
		{"*x/*lain", "text/plain", false},
		{"text/html", "text/plain", false},
		{"/", "text/plain", false},
		{" ", "", false},
		{"", " ", false},
	}

	for _, test := range tests {
		if Match(test.Pattern, test.String) != test.Pass {
			t.Fatalf("P: %q vs S: %q Got: %v",
				test.Pattern, test.String, !test.Pass)
		}
	}
}

func TestMakeShort(t *testing.T) {
	tests := []struct {
		in  []byte
		exp string
	}{
		{in: []byte{}, exp: ""},

		{in: []byte{0}, exp: "aa"},
		{in: []byte{0, 0}, exp: "aaa"},
		{in: []byte{0, 0, 0}, exp: "aaaa"},

		{in: []byte{0x0F}, exp: "dW"},        // 000011 11.... -> 3, 32+16(48)
		{in: []byte{0x0F, 0xFF}, exp: "d@8"}, // 000011,111111,111100->3,63,60
		{in: []byte{0xF0}, exp: "8a"},        // 111100,00000 -> 60,0
		{in: []byte{0xF0, 0xFF}, exp: "8p8"}, // 111100,001111,1111..:60,15,60

		{in: []byte{0xFF}, exp: "@W"},        // 111111,11.... : 63,48
		{in: []byte{0xFF, 0xFF}, exp: "@@8"}, // 111111 111111 1111..:63,63,60
		{in: []byte{0xFF, 0xFF, 0xFF}, exp: "@@@@"},
		{in: []byte{0xFF, 0xFF, 0xFF, 0xFF}, exp: "@@@@@W"},

		{in: []byte{0xAA, 0xAA, 0xAA, 0xAA}, exp: "QQQQQG"},
		{in: []byte{0x55, 0x55, 0x55, 0x55}, exp: "vvvvvq"},
		{in: []byte{0xA5, 0xA5, 0xA5, 0xA5}, exp: "PAwLPq"},
	}

	for _, test := range tests {
		res := MakeShort(test.in)
		if res != test.exp {
			t.Fatalf("%x should map to %q, got %q", test.in, test.exp, res)
		}
	}
}

func TestJsonParser(t *testing.T) {
	myAny := (any)(nil)

	tests := []struct {
		body string
		exp  string
		data any
	}{
		{
			body: "",
			exp:  `v must be a non-nil pointer`,
			data: ((*any)(nil)),
		},
		{
			body: "",
			exp:  `path '': unexpected end of input`,
			data: &myAny,
		},
		{
			body: "",
			exp:  `path '': unexpected end of input`,
			data: &struct{}{},
		},
		{
			body: "{}",
			exp:  ``,
			data: &struct{}{},
		},
		{
			body: `{"foo":"bar"}`,
			exp:  `unknown field "foo" at path '.foo'`,
			data: &struct{}{},
		},
		{
			body: `{"foo":"bar"}`,
			exp:  ``,
			data: &(struct{ Foo string }{}),
		},
		{
			body: `{"foo":{"bar":5}}`,
			exp:  ``,
			data: &(struct{ Foo struct{ Bar int } }{}),
		},
		{
			body: `{"foo":{"bar":"str"}}`,
			exp:  `path '.foo.bar': expected "int", got "string"`,
			data: &(struct{ Foo struct{ Bar int } }{}),
		},
		{
			body: `{"Foo":{"xoo":5}}`,
			exp:  `unknown field "xoo" at path '.Foo.xoo'`,
			data: &(struct{ Foo struct{ Bar int } }{}),
		},
		{
			body: `{"Foo":{"xoo":}}`,
			exp:  `path '.Foo.xoo': unexpected character '}'`,
			data: &(struct{ Foo struct{ Bar int } }{}),
		},
		{
			body: `[]`,
			exp:  ``,
			data: &([]struct{ Foo string }{}),
		},
		{
			body: `[1,2,3]`,
			exp:  `path '[0]': expected "struct", got "number"`,
			data: &([]struct{ Foo string }{}),
		},
		{
			body: `[{"bar":5}]`,
			exp:  `unknown field "bar" at path '[0].bar'`,
			data: &([]struct{ Foo string }{}),
		},
		{
			body: `[{"fOo":"asd"}]`,
			exp:  ``,
			data: &([]struct{ Foo string }{}),
		},
	}

	for _, test := range tests {
		err := Unmarshal([]byte(test.body), test.data)
		if test.exp == "" {
			if err == nil {
				continue
			}
			t.Fatalf("'%s' should have worked\nGot: %s", test.body, err)
		} else {
			if err == nil {
				t.Fatalf("'%s' should have failed\nExp: %s",
					test.body, test.exp)
			}
			if err.Error() != test.exp {
				t.Fatalf("'%s' should have failed\nExp: %s\nGot: %s",
					test.body, test.exp, err)
			}
		}
	}
}
