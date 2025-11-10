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

func ErrFatalf(errAny any, args ...any) {
	if IsNil(errAny) {
		return
	}
	format := "%s"
	if len(args) > 0 {
		format = args[0].(string)
		args = args[1:]
	} else {
		args = []any{errAny}
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

	var xErr *XRError
	if reg == nil {
		reg, xErr = registry.FindRegistry(nil, "APIs-Guru", registry.FOR_WRITE)
		ErrFatalf(xErr)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, xErr = registry.NewRegistry(nil, "APIs-Guru")
		ErrFatalf(xErr, "Error creating new registry: %s", xErr)
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

	newModel := &registry.Model{}
	g, xErr := newModel.AddGroupModel("apiproviders", "apiprovider")
	ErrFatalf(xErr)
	r, xErr := g.AddResourceModel("apis", "api", 2, true, true, true)
	_, xErr = r.AddAttr("format", STRING)
	ErrFatalf(xErr)

	ErrFatalf(reg.Model.ApplyNewModel(newModel, ""))

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

		group, xErr := reg.FindGroup("apiproviders", parts[0], false,
			registry.FOR_WRITE)
		ErrFatalf(xErr)

		if group == nil {
			group, xErr = reg.AddGroup("apiproviders", parts[0])
			ErrFatalf(xErr)
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

		res, xErr := group.AddResource("apis", resName, "v1")
		ErrFatalf(xErr)

		version, xErr := res.FindVersion(parts[verIndex], false,
			registry.FOR_WRITE)
		ErrFatalf(xErr)
		if version != nil {
			log.Fatalf("Have more than one file per version: %s\n", header.Name)
		}

		buf := &bytes.Buffer{}
		io.Copy(buf, reader)
		version, xErr = res.AddVersion(parts[verIndex])
		ErrFatalf(xErr)
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
	var xErr *XRError
	if reg == nil {
		reg, xErr = registry.FindRegistry(nil, "TestRegistry",
			registry.FOR_WRITE)
		ErrFatalf(xErr)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, xErr = registry.NewRegistry(nil, "TestRegistry")
		ErrFatalf(xErr, "Error creating new registry: %s", xErr)
		defer reg.Rollback()

		ErrFatalf(reg.SetSave("#baseURL", "http://soaphub.org:8585/"))
		ErrFatalf(reg.SetSave("name", "Test Registry"))
		ErrFatalf(reg.SetSave("description", "A test reg"))
		ErrFatalf(reg.SetSave("documentation", "https://github.com/xregistry/server"))

		ErrFatalf(reg.SetSave("labels.stage", "prod"))

		newModel := &registry.Model{}

		_, xErr = newModel.AddAttr("bool1", BOOLEAN)
		ErrFatalf(xErr)
		_, xErr = newModel.AddAttr("int1", INTEGER)
		ErrFatalf(xErr)
		_, xErr = newModel.AddAttr("dec1", DECIMAL)
		ErrFatalf(xErr)
		_, xErr = newModel.AddAttr("str1", STRING)
		ErrFatalf(xErr)
		_, xErr = newModel.AddAttrMap("map1", registry.NewItemType(STRING))
		ErrFatalf(xErr)
		_, xErr = newModel.AddAttrArray("arr1", registry.NewItemType(STRING))
		ErrFatalf(xErr)

		_, xErr = newModel.AddAttrMap("emptymap", registry.NewItemType(STRING))
		ErrFatalf(xErr)
		_, xErr = newModel.AddAttrArray("emptyarr", registry.NewItemType(STRING))
		ErrFatalf(xErr)
		_, xErr = newModel.AddAttrObj("emptyobj")
		ErrFatalf(xErr)
		obj, xErr := newModel.AddAttrObj("modelobj")
		ErrFatalf(xErr)
		_, xErr = obj.AddAttr("model", STRING)
		ErrFatalf(xErr)
		_, xErr = obj.AddAttr("model2", STRING)
		ErrFatalf(xErr)

		item := registry.NewItemObject()
		_, xErr = item.AddAttr("inint", INTEGER)
		ErrFatalf(xErr)
		_, xErr = newModel.AddAttrMap("mapobj", item)
		ErrFatalf(xErr)

		_, xErr = newModel.AddAttrArray("arrmapstr",
			registry.NewItemMap(registry.NewItemType(STRING)))
		ErrFatalf(xErr)

		item = registry.NewItemMap(registry.NewItemObject())
		_, xErr = newModel.AddAttrArray("arrmapobj", item)
		ErrFatalf(xErr)
		item = item.Item
		_, xErr = item.AddAttr("aoint", INTEGER)
		ErrFatalf(xErr)
		objAttr, xErr := item.AddAttrObj("objint")
		ErrFatalf(xErr)
		_, xErr = objAttr.AddAttr("anobjint", INTEGER)
		ErrFatalf(xErr)

		item = registry.NewItemObject()
		_, xErr = item.AddAttr("aoint", INTEGER)
		ErrFatalf(xErr)
		_, xErr = newModel.AddAttrArray("arrobj", item)
		ErrFatalf(xErr)

		ErrFatalf(reg.Model.ApplyNewModel(newModel, ""))

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
		ErrFatalf(reg.SetSave("arrmapobj[1].key1",
			map[string]any{}))
	}

	newModel := reg.Model.GetSourceAsModel()
	if newModel == nil {
		ErrFatalf(fmt.Errorf("modelsource is empty"))
	}

	Verbose("Loading: /reg-%s", reg.UID)
	gm, xErr := newModel.AddGroupModel("dirs", "dir")
	ErrFatalf(xErr)
	rm, xErr := gm.AddResourceModel("files", "file", 2, true, true, true)
	_, xErr = rm.AddMetaAttr("rext", STRING)
	ErrFatalf(xErr)
	_, xErr = rm.AddMetaAttr("*", ANY)
	ErrFatalf(xErr)
	_, xErr = rm.AddAttr("vext", STRING)
	ErrFatalf(xErr)
	rm, xErr = gm.AddResourceModel("datas", "data", 2, true, true, false)
	ErrFatalf(xErr)
	_, xErr = rm.AddAttr("*", STRING)
	ErrFatalf(xErr)

	_, xErr = newModel.AddAttrXID("resptr", "/dirs/files[/versions]")
	ErrFatalf(xErr)

	ErrFatalf(reg.Model.ApplyNewModel(newModel, ""))

	g, xErr := reg.AddGroup("dirs", "d1")
	ErrFatalf(xErr)
	ErrFatalf(g.SetSave("labels.private", "true"))
	r, xErr := g.AddResource("files", "f1", "v1")
	ErrFatalf(xErr)
	ErrFatalf(g.SetSave("labels.private", "true"))
	_, xErr = r.AddVersion("v2")
	ErrFatalf(xErr)
	ErrFatalf(r.SetSaveMeta("labels.stage", "dev"))
	ErrFatalf(r.SetSaveMeta("labels.none", ""))
	ErrFatalf(r.SetSaveMeta("rext", "a string"))
	ErrFatalf(r.SetSaveDefault("vext", "a ver string"))
	ErrFatalf(reg.SetSave("resptr", "/dirs/d1/files/f1/versions/v1"))

	ErrFatalf(r.SetSave("file", `{"hello":"world"}`))
	ErrFatalf(r.SetSave("contenttype", `application/json`))

	r, xErr = g.AddResource("files", "fr", "v1")
	ErrFatalf(xErr)
	ErrFatalf(r.SetSaveMeta("readonly", true))

	_, xErr = g.AddResource("datas", "d1", "v1")

	_, xErr = g.AddResourceWithObject("files", "fx", "",
		map[string]any{
			"meta": map[string]any{"xref": "/dirs/d1/files/f1"},
		}, false)
	ErrFatalf(xErr)

	reg.Commit()
	return reg
}

func LoadEndpointsSample(reg *registry.Registry) *registry.Registry {
	var xErr *XRError
	if reg == nil {
		reg, xErr = registry.FindRegistry(nil, "Endpoints", registry.FOR_WRITE)
		ErrFatalf(xErr)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, xErr = registry.NewRegistry(nil, "Endpoints")
		ErrFatalf(xErr, "Error creating new registry: %s", xErr)
		defer reg.Rollback()

		ErrFatalf(reg.SetSave("#baseURL", "http://soaphub.org:8585/"))
		ErrFatalf(reg.SetSave("name", "Endpoints Registry"))
		ErrFatalf(reg.SetSave("description", "An impl of the endpoints spec"))
		ErrFatalf(reg.SetSave("documentation", "https://github.com/xregistry/server"))
	}

	Verbose("Loading: /reg-%s", reg.UID)
	fn, err := common.FindModelFile("endpoint/model.json")
	ErrFatalf(err)
	xErr = reg.LoadModelFromFile(fn)
	ErrFatalf(xErr)

	// End of model

	g, xErr := reg.AddGroupWithObject("endpoints", "e1", common.Object{
		"usage": []string{"producer"},
	})
	ErrFatalf(xErr)
	ErrFatalf(g.SetSave("name", "end1"))
	ErrFatalf(g.SetSave("epoch", 1))
	ErrFatalf(g.SetSave("labels.stage", "dev"))
	ErrFatalf(g.SetSave("labels.stale", "true"))

	r, xErr := g.AddResource("messages", "created", "v1")
	ErrFatalf(xErr)
	v, xErr := r.FindVersion("v1", false, registry.FOR_WRITE)
	ErrFatalf(xErr)
	ErrFatalf(v.SetSave("name", "blobCreated"))
	ErrFatalf(v.SetSave("epoch", 2))

	v, xErr = r.AddVersion("v2")
	ErrFatalf(xErr)
	ErrFatalf(v.SetSave("name", "blobCreated"))
	ErrFatalf(v.SetSave("epoch", 4))
	ErrFatalf(r.SetDefault(v))

	r, xErr = g.AddResource("messages", "deleted", "v1.0")
	ErrFatalf(xErr)
	v, xErr = r.FindVersion("v1.0", false, registry.FOR_WRITE)
	ErrFatalf(xErr)
	ErrFatalf(v.SetSave("name", "blobDeleted"))
	ErrFatalf(v.SetSave("epoch", 3))

	g, xErr = reg.AddGroupWithObject("endpoints", "e2", common.Object{
		"usage": []string{"consumer"},
	})
	ErrFatalf(xErr)
	ErrFatalf(g.SetSave("name", "end1"))
	ErrFatalf(g.SetSave("epoch", 1))

	ErrFatalf(reg.Model.Verify())
	reg.Commit()
	return reg
}

func LoadMessagesSample(reg *registry.Registry) *registry.Registry {
	var xErr *XRError
	if reg == nil {
		reg, xErr = registry.FindRegistry(nil, "Messages", registry.FOR_WRITE)
		ErrFatalf(xErr)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, xErr = registry.NewRegistry(nil, "Messages")
		ErrFatalf(xErr, "Error creating new registry: %s", xErr)
		defer reg.Rollback()

		reg.SetSave("#baseURL", "http://soaphub.org:8585/")
		reg.SetSave("name", "Messages Registry")
		reg.SetSave("description", "An impl of the sages spec")
		reg.SetSave("documentation", "https://github.com/xregistry/server")
	}

	Verbose("Loading: /reg-%s", reg.UID)
	fn, err := common.FindModelFile("message/model.json")
	ErrFatalf(err)
	xErr = reg.LoadModelFromFile(fn)
	ErrFatalf(xErr)

	// End of model

	ErrFatalf(reg.Model.Verify())
	reg.Commit()
	return reg
}

func LoadSchemasSample(reg *registry.Registry) *registry.Registry {
	var xErr *XRError
	if reg == nil {
		reg, xErr = registry.FindRegistry(nil, "Schemas", registry.FOR_WRITE)
		ErrFatalf(xErr)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, xErr = registry.NewRegistry(nil, "Schemas")
		ErrFatalf(xErr, "Error creating new registry: %s", xErr)
		defer reg.Rollback()

		reg.SetSave("#baseURL", "http://soaphub.org:8585/")
		reg.SetSave("name", "Schemas Registry")
		reg.SetSave("description", "An impl of the schemas spec")
		reg.SetSave("documentation", "https://github.com/xregistry/server")
	}

	Verbose("Loading: /reg-%s", reg.UID)
	fn, err := common.FindModelFile("schema/model.json")
	ErrFatalf(err)
	xErr = reg.LoadModelFromFile(fn)
	ErrFatalf(xErr)

	// End of model

	ErrFatalf(reg.Model.Verify())
	reg.Commit()
	return reg
}

func LoadLargeSample(reg *registry.Registry) *registry.Registry {
	var xErr *XRError
	start := time.Now()
	if reg == nil {
		reg, xErr = registry.FindRegistry(nil, "Large", registry.FOR_WRITE)
		ErrFatalf(xErr)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, xErr = registry.NewRegistry(nil, "Large")
		ErrFatalf(xErr, "Error creating new registry: %s", xErr)
		defer reg.Rollback()

		reg.SetSave("#baseURL", "http://soaphub.org:8585/")
		reg.SetSave("name", "Large Registry")
		reg.SetSave("description", "A large Registry")
		reg.SetSave("documentation", "https://github.com/xregistry/server")
	}

	Verbose("Loading: /reg-%s", reg.UID)

	newModel := &registry.Model{}

	gm, _ := newModel.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	ErrFatalf(reg.Model.ApplyNewModel(newModel, ""))

	maxD, maxF, maxV := 10, 150, 5
	dirs, files, vers := 0, 0, 0
	for dcount := 0; dcount < maxD; dcount++ {
		dName := fmt.Sprintf("dir%d", dcount)
		d, xErr := reg.AddGroup("dirs", dName)
		ErrFatalf(xErr)
		dirs++
		for fcount := 0; fcount < maxF; fcount++ {
			fName := fmt.Sprintf("file%d", fcount)
			f, xErr := d.AddResource("files", fName, "v0")
			ErrFatalf(xErr)
			files++
			vers++
			for vcount := 1; vcount < maxV; vcount++ {
				_, xErr = f.AddVersion(fmt.Sprintf("v%d", vcount))
				vers++
				ErrFatalf(xErr)
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
	var xErr *XRError
	if reg == nil {
		reg, xErr = registry.FindRegistry(nil, "DocStore", registry.FOR_WRITE)
		ErrFatalf(xErr)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, xErr = registry.NewRegistry(nil, "DocStore")
		ErrFatalf(xErr, "Error creating new registry: %s", xErr)
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
	var xErr *XRError

	if reg == nil {
		reg, xErr = registry.FindRegistry(nil, "CloudEvents", registry.FOR_WRITE)
		ErrFatalf(xErr)
		if reg != nil {
			reg.Rollback()
			return reg
		}

		reg, xErr = registry.NewRegistry(nil, "CloudEvents")
		ErrFatalf(xErr, "Error creating new registry: %s", xErr)
		defer reg.Rollback()

		reg.SetSave("#baseURL", "http://soaphub.org:8585/")
		reg.SetSave("name", "CloudEvents Registry")
		reg.SetSave("description", "An impl of the CloudEvents xReg spec")
		reg.SetSave("documentation", "https://github.com/xregistry/server")
	}

	Verbose("Loading: /reg-%s", reg.UID)
	fn, err := common.FindModelFile("cloudevents/model.json")
	ErrFatalf(err)
	xErr = reg.LoadModelFromFile(fn)
	ErrFatalf(xErr)

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
		g, xErr := reg.AddGroupWithObject("endpoints", "e1", common.Object{
			"usage": []string{"producer"},
		})
		ErrFatalf(xErr)

		r, xErr := g.AddResource("messages", "blobCreated", "v1")
		ErrFatalf(xErr)

		r, xErr = g.AddResource("messages", "blobDeleted", "v1.0")
		ErrFatalf(xErr)

		g, xErr = reg.AddGroupWithObject("endpoints", "e2", common.Object{
			"usage": []string{"consumer"},
		})
		ErrFatalf(xErr)
		r, xErr = g.AddResource("messages", "popped", "v1.0")
		ErrFatalf(xErr)

		// Schemas
		g, xErr = reg.AddGroupWithObject("schemagroups", "sg1", common.Object{
			"format": "text",
		})
		ErrFatalf(xErr)
		r, xErr = g.AddResourceWithObject("schemas", "popped", "v1.0",
			common.Object{"format": "text"}, false)
		ErrFatalf(xErr)
		_, xErr = r.AddVersionWithObject("v2.0", common.Object{
			"format": "text",
		})
		ErrFatalf(xErr)
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
			IncomingObj, xErr := registry.ExtractIncomingObject(info, body)
			ErrFatalf(xErr)
			for key, _ := range IncomingObj {
				if reg.Model.FindGroupModel(key) == nil {
					ErrFatalf(fmt.Errorf("  - POST / only allows Group "+
						"types to be specified. %q is invalid", key))
				}
			}

			objMap, xErr := IncomingObj2Map(IncomingObj)
			ErrFatalf(xErr)

			for gType, gAny := range objMap {
				gMap, xErr := IncomingObj2Map(gAny)
				ErrFatalf(xErr)

				for id, obj := range gMap {
					_, _, xErr := info.Registry.UpsertGroupWithObject(gType,
						id, obj, registry.ADD_UPDATE)
					if xErr != nil {
						log.Printf("From: %s", file.DownloadURL)
						log.Printf("Input:\n%s", ToJSON(obj))
					}
					ErrFatalf(xErr, "  - %s", xErr)
				}
			}
		}
	}

	reg.Commit()
	return reg
}
