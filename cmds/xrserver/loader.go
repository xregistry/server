package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	// "errors"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	log "github.com/duglin/dlog"
	"github.com/xregistry/server/common"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

var Token string
var Secret string

func ErrFatalf(err error, args ...any) {
	if err == nil {
		return
	}
	format := "%s"
	if len(args) > 0 {
		format = args[0].(string)
		args = args[1:]
	} else {
		args = []any{err}
	}
	log.Printf(format, args...)
	common.ShowStack()
	os.Exit(1)
}

func init() {
	if tmp := os.Getenv("githubToken"); tmp != "" {
		Token = tmp
	} else {
		if buf, _ := os.ReadFile(".github"); len(buf) > 0 {
			Token = string(buf)
		}
	}
}

func LoadAPIGuru(reg *registry.Registry, orgName string, repoName string) *registry.Registry {
	var err error
	Token = strings.TrimSpace(Token)

	/*
		gh := github.NewGitHubClient("api.github.com", Token, Secret)
		repo, err := gh.GetRepository(orgName, repoName)
		if err != nil {
			log.Fatalf("Error finding repo %s/%s: %s", orgName, repoName, err)
		}

		tarStream, err := repo.GetTar()
		if err != nil {
			log.Fatalf("Error getting tar from repo %s/%s: %s",
				orgName, repoName, err)
		}
		defer tarStream.Close()
	*/

	buf, err := ioutil.ReadFile("misc/repo.tar")
	if err != nil {
		log.Fatalf("Can't load 'misc/repo.tar': %s", err)
	}
	tarStream := bytes.NewReader(buf)

	gzf, _ := gzip.NewReader(tarStream)
	reader := tar.NewReader(gzf)

	if reg == nil {
		reg, err = registry.FindRegistry(nil, "APIs-Guru", registry.FOR_WRITE)
		ErrFatalf(err)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, err = registry.NewRegistry(nil, "APIs-Guru")
		ErrFatalf(err, "Error creating new registry: %s", err)
		// Verbose( "New registry:\n%#v", reg)
		defer reg.Rollback()

		ErrFatalf(reg.SetSave("#baseURL", "http://soaphub.org:8585/"))
		ErrFatalf(reg.SetSave("name", "APIs-guru Registry"))
		ErrFatalf(reg.SetSave("description", "xRegistry view of github.com/APIs-guru/openapi-directory"))
		ErrFatalf(reg.SetSave("documentation", "https://github.com/xregistry/server"))
		ErrFatalf(reg.Refresh(registry.FOR_READ))
		// Verbose( "New registry:\n%#v", reg)

		// TODO Support "model" being part of the Registry struct above
	}

	Verbose("Loading: /reg-%s", reg.UID)
	g, err := reg.Model.AddGroupModel("apiproviders", "apiprovider")
	ErrFatalf(err)
	r, err := g.AddResourceModel("apis", "api", 2, true, true, true)
	_, err = r.AddAttr("format", STRING)
	ErrFatalf(err)

	ErrFatalf(reg.Model.VerifyAndSave())

	iter := 0

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error getting next tar entry: %s", err)
		}

		// Skip non-regular files (and dirs)
		if header.Typeflag > '9' || header.Typeflag == tar.TypeDir {
			continue
		}

		i := 0
		// Skip files not under the APIs dir
		if i = strings.Index(header.Name, "/APIs/"); i < 0 {
			continue
		}

		// Just a subset for now
		if strings.Index(header.Name, "/docker.com/") < 0 &&
			strings.Index(header.Name, "/adobe.com/") < 0 &&
			strings.Index(header.Name, "/fec.gov/") < 0 &&
			strings.Index(header.Name, "/apiz.ebay.com/") < 0 {
			continue
		}

		parts := strings.Split(strings.Trim(header.Name[i+6:], "/"), "/")
		// org/service/version/file
		// org/version/file

		group, err := reg.FindGroup("apiproviders", parts[0], false,
			registry.FOR_WRITE)
		ErrFatalf(err)

		if group == nil {
			group, err = reg.AddGroup("apiproviders", parts[0])
			ErrFatalf(err)
		}

		ErrFatalf(group.SetSave("name", group.UID))
		ErrFatalf(group.SetSave("modifiedat", time.Now().Format(time.RFC3339)))
		ErrFatalf(group.SetSave("epoch", 5))

		// group2 := reg.FindGroup("apiproviders", parts[0], registry.FOR_WRITE)
		// log.Printf("Find Group:\n%s", registry.ToJSON(group2))

		resName := "core"
		verIndex := 1
		if len(parts) == 4 {
			resName = parts[1]
			verIndex++
		}

		res, err := group.AddResource("apis", resName, "v1")
		ErrFatalf(err)

		version, err := res.FindVersion(parts[verIndex], false,
			registry.FOR_WRITE)
		ErrFatalf(err)
		if version != nil {
			log.Fatalf("Have more than one file per version: %s\n", header.Name)
		}

		buf := &bytes.Buffer{}
		io.Copy(buf, reader)
		version, err = res.AddVersion(parts[verIndex])
		ErrFatalf(err)
		ErrFatalf(version.SetSave("name", parts[verIndex+1]))
		ErrFatalf(version.SetSave("format", "openapi/3.0.6"))

		// Don't upload the file contents into the registry. Instead just
		// give the registry a URL to it and ask it to server it via proxy.
		// We could have also just set the resourceURI to the file but
		// I wanted the URL to the file to be the registry and not github

		base := "https://raw.githubusercontent.com/APIs-guru/" +
			"openapi-directory/main/APIs/"
		switch iter % 3 {
		case 0:
			ErrFatalf(version.SetSave("api", buf.Bytes()))
		case 1:
			ErrFatalf(version.SetSave("apiurl", base+header.Name[i+6:]))
		case 2:
			ErrFatalf(version.SetSave("apiproxyurl", base+header.Name[i+6:]))
		}
		iter++
	}

	ErrFatalf(reg.Model.Verify())
	reg.Commit()
	return reg
}

func LoadDirsSample(reg *registry.Registry) *registry.Registry {
	var err error
	if reg == nil {
		reg, err = registry.FindRegistry(nil, "TestRegistry",
			registry.FOR_WRITE)
		ErrFatalf(err)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, err = registry.NewRegistry(nil, "TestRegistry")
		ErrFatalf(err, "Error creating new registry: %s", err)
		defer reg.Rollback()

		ErrFatalf(reg.SetSave("#baseURL", "http://soaphub.org:8585/"))
		ErrFatalf(reg.SetSave("name", "Test Registry"))
		ErrFatalf(reg.SetSave("description", "A test reg"))
		ErrFatalf(reg.SetSave("documentation", "https://github.com/xregistry/server"))

		ErrFatalf(reg.SetSave("labels.stage", "prod"))

		_, err = reg.Model.AddAttr("bool1", BOOLEAN)
		ErrFatalf(err)
		_, err = reg.Model.AddAttr("int1", INTEGER)
		ErrFatalf(err)
		_, err = reg.Model.AddAttr("dec1", DECIMAL)
		ErrFatalf(err)
		_, err = reg.Model.AddAttr("str1", STRING)
		ErrFatalf(err)
		_, err = reg.Model.AddAttrMap("map1", registry.NewItemType(STRING))
		ErrFatalf(err)
		_, err = reg.Model.AddAttrArray("arr1", registry.NewItemType(STRING))
		ErrFatalf(err)

		_, err = reg.Model.AddAttrMap("emptymap", registry.NewItemType(STRING))
		ErrFatalf(err)
		_, err = reg.Model.AddAttrArray("emptyarr", registry.NewItemType(STRING))
		ErrFatalf(err)
		_, err = reg.Model.AddAttrObj("emptyobj")
		ErrFatalf(err)
		obj, err := reg.Model.AddAttrObj("modelobj")
		ErrFatalf(err)
		_, err = obj.AddAttr("model", STRING)
		ErrFatalf(err)
		_, err = obj.AddAttr("model2", STRING)
		ErrFatalf(err)

		item := registry.NewItemObject()
		_, err = item.AddAttr("inint", INTEGER)
		ErrFatalf(err)
		_, err = reg.Model.AddAttrMap("mapobj", item)
		ErrFatalf(err)

		_, err = reg.Model.AddAttrArray("arrmap",
			registry.NewItemMap(registry.NewItemType(STRING)))
		ErrFatalf(err)
		ErrFatalf(reg.Model.VerifyAndSave())

		ErrFatalf(reg.SetSave("bool1", true))
		ErrFatalf(reg.SetSave("int1", 1))
		ErrFatalf(reg.SetSave("dec1", 1.1))
		ErrFatalf(reg.SetSave("str1", "hi"))
		ErrFatalf(reg.SetSave("map1.k1", "v1"))

		ErrFatalf(reg.SetSave("emptymap", map[string]int{}))
		ErrFatalf(reg.SetSave("emptyarr", []int{}))
		ErrFatalf(reg.SetSave("emptyobj", map[string]any{})) // struct{}{}))

		ErrFatalf(reg.SetSave("arr1[0]", "arr1-value"))
		ErrFatalf(reg.SetSave("mapobj.mapkey.inint", 5))
		ErrFatalf(reg.SetSave("mapobj['cool_key'].inint", 666))
		ErrFatalf(reg.SetSave("arrmap[1].key1", "arrmapk1-value"))
	}

	Verbose("Loading: /reg-%s", reg.UID)
	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	ErrFatalf(err)
	rm, err := gm.AddResourceModel("files", "file", 2, true, true, true)
	_, err = rm.AddMetaAttr("rext", STRING)
	ErrFatalf(err)
	_, err = rm.AddMetaAttr("*", ANY)
	ErrFatalf(err)
	_, err = rm.AddAttr("vext", STRING)
	ErrFatalf(err)
	rm, err = gm.AddResourceModel("datas", "data", 2, true, true, false)
	ErrFatalf(err)
	_, err = rm.AddAttr("*", STRING)
	ErrFatalf(err)

	_, err = reg.Model.AddAttrXID("resptr", "/dirs/files[/versions]")
	ErrFatalf(err)

	ErrFatalf(reg.Model.VerifyAndSave())

	g, err := reg.AddGroup("dirs", "d1")
	ErrFatalf(err)
	ErrFatalf(g.SetSave("labels.private", "true"))
	r, err := g.AddResource("files", "f1", "v1")
	ErrFatalf(err)
	ErrFatalf(g.SetSave("labels.private", "true"))
	_, err = r.AddVersion("v2")
	ErrFatalf(err)
	ErrFatalf(r.SetSaveMeta("labels.stage", "dev"))
	ErrFatalf(r.SetSaveMeta("labels.none", ""))
	ErrFatalf(r.SetSaveMeta("rext", "a string"))
	ErrFatalf(r.SetSaveDefault("vext", "a ver string"))
	ErrFatalf(reg.SetSave("resptr", "/dirs/d1/files/f1/versions/v1"))

	ErrFatalf(r.SetSave("file", `{"hello":"world"}`))
	ErrFatalf(r.SetSave("contenttype", `application/json`))

	r, err = g.AddResource("files", "fr", "v1")
	ErrFatalf(err)
	ErrFatalf(r.SetSaveMeta("readonly", true))

	_, err = g.AddResource("datas", "d1", "v1")

	_, err = g.AddResourceWithObject("files", "fx", "",
		map[string]any{
			"meta": map[string]any{"xref": "/dirs/d1/files/f1"},
		}, false)
	ErrFatalf(err)

	reg.Commit()
	return reg
}

func LoadEndpointsSample(reg *registry.Registry) *registry.Registry {
	var err error
	if reg == nil {
		reg, err = registry.FindRegistry(nil, "Endpoints", registry.FOR_WRITE)
		ErrFatalf(err)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, err = registry.NewRegistry(nil, "Endpoints")
		ErrFatalf(err, "Error creating new registry: %s", err)
		defer reg.Rollback()

		ErrFatalf(reg.SetSave("#baseURL", "http://soaphub.org:8585/"))
		ErrFatalf(reg.SetSave("name", "Endpoints Registry"))
		ErrFatalf(reg.SetSave("description", "An impl of the endpoints spec"))
		ErrFatalf(reg.SetSave("documentation", "https://github.com/xregistry/server"))
	}

	Verbose("Loading: /reg-%s", reg.UID)
	fn, err := common.FindModelFile("endpoint/model.json")
	ErrFatalf(err)
	err = reg.LoadModelFromFile(fn)
	ErrFatalf(err)

	// End of model

	g, err := reg.AddGroupWithObject("endpoints", "e1", common.Object{
		"usage": []string{"producer"},
	})
	ErrFatalf(err)
	ErrFatalf(g.SetSave("name", "end1"))
	ErrFatalf(g.SetSave("epoch", 1))
	ErrFatalf(g.SetSave("labels.stage", "dev"))
	ErrFatalf(g.SetSave("labels.stale", "true"))

	r, err := g.AddResource("messages", "created", "v1")
	ErrFatalf(err)
	v, err := r.FindVersion("v1", false, registry.FOR_WRITE)
	ErrFatalf(err)
	ErrFatalf(v.SetSave("name", "blobCreated"))
	ErrFatalf(v.SetSave("epoch", 2))

	v, err = r.AddVersion("v2")
	ErrFatalf(err)
	ErrFatalf(v.SetSave("name", "blobCreated"))
	ErrFatalf(v.SetSave("epoch", 4))
	ErrFatalf(r.SetDefault(v))

	r, err = g.AddResource("messages", "deleted", "v1.0")
	ErrFatalf(err)
	v, err = r.FindVersion("v1.0", false, registry.FOR_WRITE)
	ErrFatalf(err)
	ErrFatalf(v.SetSave("name", "blobDeleted"))
	ErrFatalf(v.SetSave("epoch", 3))

	g, err = reg.AddGroupWithObject("endpoints", "e2", common.Object{
		"usage": []string{"consumer"},
	})
	ErrFatalf(err)
	ErrFatalf(g.SetSave("name", "end1"))
	ErrFatalf(g.SetSave("epoch", 1))

	ErrFatalf(reg.Model.Verify())
	reg.Commit()
	return reg
}

func LoadMessagesSample(reg *registry.Registry) *registry.Registry {
	var err error
	if reg == nil {
		reg, err = registry.FindRegistry(nil, "Messages", registry.FOR_WRITE)
		ErrFatalf(err)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, err = registry.NewRegistry(nil, "Messages")
		ErrFatalf(err, "Error creating new registry: %s", err)
		defer reg.Rollback()

		reg.SetSave("#baseURL", "http://soaphub.org:8585/")
		reg.SetSave("name", "Messages Registry")
		reg.SetSave("description", "An impl of the sages spec")
		reg.SetSave("documentation", "https://github.com/xregistry/server")
	}

	Verbose("Loading: /reg-%s", reg.UID)
	fn, err := common.FindModelFile("message/model.json")
	ErrFatalf(err)
	err = reg.LoadModelFromFile(fn)
	ErrFatalf(err)

	// End of model

	ErrFatalf(reg.Model.Verify())
	reg.Commit()
	return reg
}

func LoadSchemasSample(reg *registry.Registry) *registry.Registry {
	var err error
	if reg == nil {
		reg, err = registry.FindRegistry(nil, "Schemas", registry.FOR_WRITE)
		ErrFatalf(err)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, err = registry.NewRegistry(nil, "Schemas")
		ErrFatalf(err, "Error creating new registry: %s", err)
		defer reg.Rollback()

		reg.SetSave("#baseURL", "http://soaphub.org:8585/")
		reg.SetSave("name", "Schemas Registry")
		reg.SetSave("description", "An impl of the schemas spec")
		reg.SetSave("documentation", "https://github.com/xregistry/server")
	}

	Verbose("Loading: /reg-%s", reg.UID)
	fn, err := common.FindModelFile("schema/model.json")
	ErrFatalf(err)
	err = reg.LoadModelFromFile(fn)
	ErrFatalf(err)

	// End of model

	ErrFatalf(reg.Model.Verify())
	reg.Commit()
	return reg
}

func LoadLargeSample(reg *registry.Registry) *registry.Registry {
	var err error
	start := time.Now()
	if reg == nil {
		reg, err = registry.FindRegistry(nil, "Large", registry.FOR_WRITE)
		ErrFatalf(err)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, err = registry.NewRegistry(nil, "Large")
		ErrFatalf(err, "Error creating new registry: %s", err)
		defer reg.Rollback()

		reg.SetSave("#baseURL", "http://soaphub.org:8585/")
		reg.SetSave("name", "Large Registry")
		reg.SetSave("description", "A large Registry")
		reg.SetSave("documentation", "https://github.com/xregistry/server")
	}

	Verbose("Loading: /reg-%s", reg.UID)
	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	ErrFatalf(reg.Model.VerifyAndSave())

	maxD, maxF, maxV := 10, 150, 5
	dirs, files, vers := 0, 0, 0
	for dcount := 0; dcount < maxD; dcount++ {
		dName := fmt.Sprintf("dir%d", dcount)
		d, err := reg.AddGroup("dirs", dName)
		ErrFatalf(err)
		dirs++
		for fcount := 0; fcount < maxF; fcount++ {
			fName := fmt.Sprintf("file%d", fcount)
			f, err := d.AddResource("files", fName, "v0")
			ErrFatalf(err)
			files++
			vers++
			for vcount := 1; vcount < maxV; vcount++ {
				_, err = f.AddVersion(fmt.Sprintf("v%d", vcount))
				vers++
				ErrFatalf(err)
				ErrFatalf(reg.Commit())
			}
		}
	}

	// End of model

	ErrFatalf(reg.Model.Verify())
	reg.Commit()
	dur := time.Now().Sub(start).Round(time.Second)
	Verbose("Done loading registry: %s (time: %s)", reg.UID, dur)
	Verbose("Dirs: %d  Files: %d  Versions: %d", dirs, files, vers)
	return reg
}

func LoadDocStore(reg *registry.Registry) *registry.Registry {
	var err error
	if reg == nil {
		reg, err = registry.FindRegistry(nil, "DocStore", registry.FOR_WRITE)
		ErrFatalf(err)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, err = registry.NewRegistry(nil, "DocStore")
		ErrFatalf(err, "Error creating new registry: %s", err)
		defer reg.Rollback()

		reg.SetSave("#baseURL", "http://soaphub.org:8585/")
		reg.SetSave("name", "DocStore Registry")
		reg.SetSave("description", "A doc store Registry")
		reg.SetSave("documentation", "https://github.com/xregistry/server")
	}

	Verbose("Loading: /reg-%s", reg.UID)
	// Use JSON for this model so that "modelsource" has something in it
	ErrFatalf(reg.Model.ApplyNewModelFromJSON([]byte(`{
      "groups": {
        "documents": {
          "singular": "document",
          "resources": {
            "formats": {
              "singular": "format"
            }
          }
        }
      }
    }
    `)))

	// ErrFatalf(reg.Model.VerifyAndSave())

	g, _ := reg.AddGroup("documents", "mydoc1")
	g.SetSave("labels.group", "g1")

	r, _ := g.AddResource("formats", "json", "v1")
	r.SetSaveDefault("contenttype", "application/json")
	r.SetSaveDefault("format", `{"prop": "A document 1"}`)

	r, _ = g.AddResource("formats", "xml", "v1")
	r.SetSaveDefault("contenttype", "application/xml")
	r.SetSaveDefault("format", `<elem title="A document 1"/>`)

	g, _ = reg.AddGroup("documents", "mydoc2")

	r, _ = g.AddResource("formats", "json", "v1")
	r.SetSaveDefault("contenttype", "application/json")
	r.SetSaveDefault("format", `{"prop": "A document 2"}`)

	r, _ = g.AddResource("formats", "xml", "v1")
	r.SetSaveDefault("contenttype", "application/xml")
	r.SetSaveDefault("format", `<elem title="A document 2"/>`)

	// End of model

	ErrFatalf(reg.Model.Verify())
	reg.Commit()
	return reg
}

func LoadCESample(reg *registry.Registry) *registry.Registry {
	var err error

	if reg == nil {
		reg, err = registry.FindRegistry(nil, "CloudEvents", registry.FOR_WRITE)
		ErrFatalf(err)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, err = registry.NewRegistry(nil, "CloudEvents")
		ErrFatalf(err, "Error creating new registry: %s", err)
		defer reg.Rollback()

		reg.SetSave("#baseURL", "http://soaphub.org:8585/")
		reg.SetSave("name", "CloudEvents Registry")
		reg.SetSave("description", "An impl of the CloudEvents xReg spec")
		reg.SetSave("documentation", "https://github.com/xregistry/server")
	}

	Verbose("Loading: /reg-%s", reg.UID)
	fn, err := common.FindModelFile("cloudevents/model.json")
	ErrFatalf(err)
	err = reg.LoadModelFromFile(fn)
	ErrFatalf(err)

	// End of model

	repoURL := "https://api.github.com/repos/xregistry/spec"
	samplesDirURL := repoURL + "/contents/cloudevents/samples/scenarios"
	res, err := http.Get(samplesDirURL)

	body := []byte{}
	if res != nil {
		body, _ = io.ReadAll(res.Body)
		res.Body.Close()
	}

	if err != nil {
		Verbose("  - Error loading samples dir: %s", err)
	} else if res.StatusCode != 200 {
		Verbose("  - Error loading samples dir: %s\n%s", res.Status, string(body))
	}

	if err != nil || res.StatusCode != 200 {
		Verbose("  - Loading fake data instead")

		// Endpoints
		g, err := reg.AddGroupWithObject("endpoints", "e1", common.Object{
			"usage": []string{"producer"},
		})
		ErrFatalf(err)

		r, err := g.AddResource("messages", "blobCreated", "v1")
		ErrFatalf(err)

		r, err = g.AddResource("messages", "blobDeleted", "v1.0")
		ErrFatalf(err)

		g, err = reg.AddGroupWithObject("endpoints", "e2", common.Object{
			"usage": []string{"consumer"},
		})
		ErrFatalf(err)
		r, err = g.AddResource("messages", "popped", "v1.0")
		ErrFatalf(err)

		// Schemas
		g, err = reg.AddGroupWithObject("schemagroups", "sg1", common.Object{
			"format": "text",
		})
		ErrFatalf(err)
		r, err = g.AddResourceWithObject("schemas", "popped", "v1.0",
			common.Object{"format": "text"}, false)
		ErrFatalf(err)
		_, err = r.AddVersionWithObject("v2.0", common.Object{
			"format": "text",
		})
		ErrFatalf(err)
	} else {
		files := []struct {
			Name        string `json:"name"`
			DownloadURL string `json:"download_url"`
			Type        string `json:"type"`
		}{}

		err = json.Unmarshal(body, &files)
		ErrFatalf(err)

		for _, file := range files {
			if !strings.HasSuffix(file.Name, "xreg.json") {
				continue
			}
			// Verbose("  - %s", file.Name)
			res, err := http.Get(file.DownloadURL)
			ErrFatalf(err)
			if res.StatusCode != 200 {
				ErrFatalf(fmt.Errorf(""), "Error downloading sample %q: %s",
					file.Name, res.Status)
			}

			// TODO create an import() func so we can just call it instead of
			// doing an HTTP call
			body, _ = io.ReadAll(res.Body)
			res.Body.Close()

			r := &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme:  "http",
					Host:    "localhost:8181",
					Path:    "",
					RawPath: "",
				},
			}
			info := &registry.RequestInfo{
				OriginalPath:    r.URL.Path, // path,
				OriginalRequest: r,          // not sure this is the best option
				Registry:        reg,
				BaseURL:         r.URL.String(),
			}

			if reg != nil && reg.Model != nil {
				ErrFatalf(info.ParseRequestURL())
			}

			// Error on anything but a group type
			IncomingObj, err := registry.ExtractIncomingObject(info, body)
			ErrFatalf(err)
			for key, _ := range IncomingObj {
				if reg.Model.FindGroupModel(key) == nil {
					ErrFatalf(fmt.Errorf("  - POST / only allows Group "+
						"types to be specified. %q is invalid", key))
				}
			}

			objMap, err := IncomingObj2Map(IncomingObj)
			ErrFatalf(err)

			for gType, gAny := range objMap {
				gMap, err := IncomingObj2Map(gAny)
				ErrFatalf(err)

				for id, obj := range gMap {
					_, _, err := info.Registry.UpsertGroupWithObject(gType,
						id, obj, registry.ADD_UPDATE)
					if err != nil {
						log.Printf("From: %s", file.DownloadURL)
						log.Printf("Input:\n%s", ToJSON(obj))
					}
					ErrFatalf(err, "  - %s", err)
				}
			}
		}
	}

	reg.Commit()
	return reg
}
