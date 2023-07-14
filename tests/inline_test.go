package tests

import (
	"testing"

	"github.com/duglin/xreg-github/registry"
)

func TestBasicInline(t *testing.T) {
	reg, _ := registry.NewRegistry("TestBasicInline")
	defer reg.Delete()

	gm, _ := reg.AddGroupModel("dirs", "dir", "")
	gm.AddResourceModel("files", "file", 0, true, true)

	d := reg.FindOrAddGroup("dirs", "d1")
	f := d.AddResource("files", "f1", "v1")
	f.FindOrAddVersion("v2")
	d = reg.FindOrAddGroup("dirs", "d2")
	f = d.AddResource("files", "f2", "v1")
	f.FindOrAddVersion("v1.1")

	gm2, _ := reg.AddGroupModel("dirs2", "dir2", "")
	gm2.AddResourceModel("files", "file", 0, true, true)
	d2 := reg.FindOrAddGroup("dirs2", "d2")
	d2.AddResource("files", "f2", "v1")

	// /dirs/d1/files/f1/v1
	//                  /v2
	//      /d2/files/f2/v1
	//                  /v1.1
	// /dirs2/d2/files/f2/v1

	tests := []struct {
		Name string
		URL  string
		Exp  string
	}{
		{
			Name: "No Inline",
			URL:  "?",
			Exp: `{
  "id": "TestBasicInline",
  "self": "http:///",

  "dirsCount": 2,
  "dirsUrl": "http:///dirs",
  "dirs2Count": 1,
  "dirs2Url": "http:///dirs2"
}
`,
		},
		{
			Name: "Inline - No Filter - full",
			URL:  "?inline",
			Exp: `{
  "id": "TestBasicInline",
  "self": "http:///",

  "dirs": {
    "d1": {
      "id": "d1",
      "self": "http:///dirs/d1",

      "files": {
        "f1": {
          "id": "f1",
          "self": "http:///dirs/d1/files/f1",
          "latestId": "v1",
          "latestUrl": "http:///dirs/d1/files/f1/versions/v1",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http:///dirs/d1/files/f1/versions/v1"
            },
            "v2": {
              "id": "v2",
              "self": "http:///dirs/d1/files/f1/versions/v2"
            }
          },
          "versionsCount": 2,
          "versionsUrl": "http:///dirs/d1/files/f1/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http:///dirs/d1/files"
    },
    "d2": {
      "id": "d2",
      "self": "http:///dirs/d2",

      "files": {
        "f2": {
          "id": "f2",
          "self": "http:///dirs/d2/files/f2",
          "latestId": "v1",
          "latestUrl": "http:///dirs/d2/files/f2/versions/v1",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http:///dirs/d2/files/f2/versions/v1"
            },
            "v1.1": {
              "id": "v1.1",
              "self": "http:///dirs/d2/files/f2/versions/v1.1"
            }
          },
          "versionsCount": 2,
          "versionsUrl": "http:///dirs/d2/files/f2/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http:///dirs/d2/files"
    }
  },
  "dirsCount": 2,
  "dirsUrl": "http:///dirs",
  "dirs2": {
    "d2": {
      "id": "d2",
      "self": "http:///dirs2/d2",

      "files": {
        "f2": {
          "id": "f2",
          "self": "http:///dirs2/d2/files/f2",
          "latestId": "v1",
          "latestUrl": "http:///dirs2/d2/files/f2/versions/v1",

          "versions": {
            "v1": {
              "id": "v1",
              "self": "http:///dirs2/d2/files/f2/versions/v1"
            }
          },
          "versionsCount": 1,
          "versionsUrl": "http:///dirs2/d2/files/f2/versions"
        }
      },
      "filesCount": 1,
      "filesUrl": "http:///dirs2/d2/files"
    }
  },
  "dirs2Count": 1,
  "dirs2Url": "http:///dirs2"
}
`,
		},
		{
			Name: "Inline - No Filter",
			URL:  "?inline&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}}}`,
		},
		{
			Name: "Inline * - No Filter",
			URL:  "?inline=*&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}},"dirs2":{"d2":{"files":{"f2":{"versions":{"v1":{}}}}}}}`,
		},
		{
			Name: "inline one level",
			URL:  "?inline=dirs&oneline",
			Exp:  `{"dirs":{"d1":{},"d2":{}}}`,
		},
		{
			Name: "inline one level - invalid",
			URL:  "?inline=xxx&oneline",
			Exp:  `Invalid 'inline' value: "xxx"`,
		},
		{
			Name: "inline one level - invalid - bad case",
			URL:  "?inline=Dirs&oneline",
			Exp:  `Invalid 'inline' value: "Dirs"`,
		},
		{
			Name: "inline two levels - invalid first",
			URL:  "?inline=xxx.files&oneline",
			Exp:  `Invalid 'inline' value: "xxx.files"`,
		},
		{
			Name: "inline two levels - invalid second",
			URL:  "?inline=dirs.xxx&oneline",
			Exp:  `Invalid 'inline' value: "dirs.xxx"`,
		},
		{
			Name: "inline two levels",
			URL:  "?inline=dirs.files&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{}}},"d2":{"files":{"f2":{}}}}}`,
		},
		{
			Name: "inline three levels",
			URL:  "?inline=dirs.files.versions&oneline",
			Exp:  `{"dirs":{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}}`,
		},
		{
			Name: "get one level, inline one level - invalid",
			URL:  "dirs?inline=dirs&oneline",
			Exp:  `Invalid 'inline' value: "dirs"`,
		},
		{
			Name: "get one level, inline one level",
			URL:  "dirs?inline=files&oneline",
			Exp:  `{"d1":{"files":{"f1":{}}},"d2":{"files":{"f2":{}}}}`,
		},
		{
			Name: "get one level, inline two levels",
			URL:  "dirs?inline=files.versions&oneline",
			Exp:  `{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}`,
		},
		{
			Name: "get one level, inline three levels",
			URL:  "dirs?inline=files.versions.xxx&oneline",
			Exp:  `Invalid 'inline' value: "files.versions.xxx"`,
		},
		{
			Name: "get one level, inline one level",
			URL:  "dirs/d1?inline=files&oneline",
			Exp:  `{"files":{"f1":{}}}`,
		},
		{
			Name: "get one level, inline two levels",
			URL:  "dirs/d1?inline=files.versions&oneline",
			Exp:  `{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}`,
		},
		{
			Name: "get one level, inline all",
			URL:  "dirs/d1?inline=&oneline",
			Exp:  `{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}}`,
		},

		{
			Name: "inline 2 top levels",
			URL:  "?inline=dirs,dirs2&oneline",
			Exp:  `{"dirs":{"d1":{},"d2":{}},"dirs2":{"d2":{}}}`,
		},
		{
			Name: "inline 2 top, 1 and 2 levels",
			URL:  "?inline=dirs,dirs2.files&oneline",
			Exp:  `{"dirs":{"d1":{},"d2":{}},"dirs2":{"d2":{"files":{"f2":{}}}}}`,
		},
		{
			Name: "inline 2 top, 1 and 2 levels - one err",
			URL:  "?inline=dirs,dirs2.files.xxx&oneline",
			Exp:  `Invalid 'inline' value: "dirs2.files.xxx"`,
		},
		{
			Name: "get one level, inline 2, 1 and 2 levels same top",
			URL:  "dirs?inline=files,files.versions&oneline",
			Exp:  `{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}`,
		},

		{
			Name: "get one level, inline all",
			URL:  "dirs?inline&oneline",
			Exp:  `{"d1":{"files":{"f1":{"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}}`,
		},
		{
			Name: "get one level/res, inline all",
			URL:  "dirs/d2?inline&oneline",
			Exp:  `{"files":{"f2":{"versions":{"v1":{},"v1.1":{}}}}}`,
		},
	}

	for _, test := range tests {
		xCheckGet(t, reg, test.URL, test.Exp)
	}
}
