package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	// "net/http"
	"os"
	"path/filepath"
	"strings"

	// log "github.com/duglin/dlog"
	"github.com/duglin/goldmark"
	"github.com/duglin/goldmark/extension"
	"github.com/duglin/goldmark/parser"
	ghtml "github.com/duglin/goldmark/renderer/html"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
	// "go.abhg.dev/goldmark/anchor"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.AnchorExtender{
			Texter: extension.Text("🔗"),
			// Texter:   extension.Text("☍"),
			Position: extension.After, // or extension.Before
		},
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		// html.WithHardWraps(),
		ghtml.WithUnsafe(),
	),
)

func addDownloadCmd(parent *cobra.Command) {
	downloadCmd := &cobra.Command{
		Use:     "download DIR [XID...] ",
		Short:   "Download entities from registry as individual files",
		Run:     downloadFunc,
		GroupID: "Entities",
	}
	downloadCmd.Flags().StringP("url", "u", "",
		"Host/path to Update xRegistry paths")
	downloadCmd.Flags().BoolP("import", "", false,
		"Create '/import.json' based on /export")
	downloadCmd.Flags().StringP("index", "i", "index.html",
		"Directory index file name (index.html*)")
	downloadCmd.Flag("index").DefValue = "" // hide default text
	downloadCmd.Flags().BoolP("md2html-no-style", "", false,
		"Do not add default styling to html files")
	downloadCmd.Flags().BoolP("md2html", "m", false,
		"Generate HTML files for MD files")
	downloadCmd.Flags().StringP("md2html-css-link", "", "",
		"CSS stylesheet 'link' to add in md2html files")
	downloadCmd.Flags().StringP("md2html-header", "", "",
		"HTML to add in <head> (data,@FILE,@URL,@-)")
	downloadCmd.Flags().StringP("md2html-html", "", "",
		"HTML to add after <head> (data,@FILE,@URL,@-)")
	downloadCmd.Flags().BoolP("capabilities", "c", false,
		"Modify capabilities for static site")
	downloadCmd.Flags().IntP("parallel", "p", 10,
		"Number of items to download in parallel (10*)")
	downloadCmd.Flag("parallel").DefValue = "0" // hide default text
	downloadCmd.Flags().StringSliceP("nodiff", "", nil,
		"No-diff attrs: *,epoch,createdat,modifiedat")

	parent.AddCommand(downloadCmd)
}

func downloadFunc(cmd *cobra.Command, args []string) {
	if GetServer() == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	if len(args) == 0 {
		Error("Missing the DIR argument")
	}

	reg, xErr := xrlib.GetRegistry(GetServer())
	Error(xErr)

	_, xErr = reg.GetModel()
	Error(xErr)

	dir := args[0]
	stat, err := os.Stat(dir)
	if os.IsNotExist(err) || !stat.IsDir() {
		Error(NewXRError("client_error", dir,
			"error_detail="+dir+" must be an existing directory"))
	}
	args = args[1:]

	if len(args) == 0 {
		args = []string{"/"}
	}

	md2html, _ := cmd.Flags().GetBool("md2html")
	md2htmlNoStyle, _ := cmd.Flags().GetBool("md2html-no-style")
	md2htmlLink, _ := cmd.Flags().GetString("md2html-css-link")
	md2htmlHeader, _ := cmd.Flags().GetString("md2html-header")
	md2htmlHTML, _ := cmd.Flags().GetString("md2html-html")

	if md2htmlHeader != "" {
		if md2htmlHeader[0] == '@' {
			buf, xErr := xrlib.ReadFile(md2htmlHeader[1:])
			Error(xErr)
			md2htmlHeader = string(buf)
		}
	}

	if md2htmlHTML != "" {
		if md2htmlHTML[0] == '@' {
			buf, xErr := xrlib.ReadFile(md2htmlHTML[1:])
			Error(xErr)
			md2htmlHTML = string(buf)
		}
	}

	indexFile, _ := cmd.Flags().GetString("index")
	host, _ := cmd.Flags().GetString("url")
	modCap, _ := cmd.Flags().GetBool("capabilities")
	noDiff, _ := cmd.Flags().GetStringSlice("nodiff")
	importFile, _ := cmd.Flags().GetBool("import")

	if host != "" {
		if host[len(host)-1] != '/' {
			host += "/"
		}
	}

	parallel, _ := cmd.Flags().GetInt("parallel")
	if parallel < 1 {
		Error("--parallel must be greater than zero")
	}

	// Our download work queue
	listCH := make(chan *Xid, parallel+1) // 1 for main loop below
	wg := sync.WaitGroup{}
	wg.Add(1)

	noDiffObj := func(obj map[string]any) {}
	noDiffObj = func(obj map[string]any) {
		if !cmd.Flags().Changed("nodiff") || len(obj) == 0 {
			return
		}

		if xidAny, ok := obj["xid"]; ok {
			xidStr, ok := xidAny.(string)
			if !ok {
				return
			}

			// if there's an XID then it must be an object and not a collection
			xid, err := ParseXid(xidStr)
			if err != nil {
				Error(err)
			}

			delete(obj, "shortself") // needs live server

			if host != "" {
				self := host + xid.String()[1:]
				selfAny := obj["self"]
				if _, ok := selfAny.(string); ok && selfAny.(string)[0] != '#' {
					obj["self"] = self
				}

				// Process nested collection URLs
				for k, v := range obj {
					// Only tweak *url fields that are: string, not relative
					vStr, ok := v.(string)
					if !ok || !strings.HasSuffix(k, "url") || vStr[0] == '#' {
						continue
					}

					base := k[:len(k)-3]
					if _, ok := obj[base+"count"]; ok {
						tmp := host + xid.String()[1:]
						if tmp[len(tmp)-1] != '/' {
							tmp += "/"
						}
						obj[k] = tmp + base
					} else if base == "meta" {
						obj[k] = host + xid.String()[1:] + "/" + base
					} else if base == "defaultversion" {
						verID := obj["defaultversionid"].(string)
						// Remove "/meta" from self
						obj[k] = self[:len(self)-5] + "/versions/" + verID
					}
				}
			}

			// Proces the --nodiff flag
			all := ArrayContains(noDiff, "*")

			if all || ArrayContains(noDiff, "epoch") {
				if _, ok := obj["epoch"]; ok {
					obj["epoch"] = 1
				}
			}
			if all || ArrayContains(noDiff, "createdat") {
				if _, ok := obj["createdat"]; ok {
					obj["createdat"] = `2000-01-01T12:00:00.00Z`
				}
			}
			// Must come after "createdat" processing
			if all || ArrayContains(noDiff, "modifiedat") {
				if _, ok := obj["modifiedat"]; ok {
					obj["modifiedat"] = obj["createdat"]
				}
			}
		}

		// Recurse for nested collections
		for _, v := range obj {
			if v1, ok := v.(map[string]any); ok {
				noDiffObj(v1)
			}
		}
	}

	noDiffHeaders := func(headers map[string]string) {
		if !cmd.Flags().Changed("nodiff") {
			return
		}

		delete(headers, "xregistry-shortself")
		for _, k := range SortedKeys(headers) {
			all := ArrayContains(noDiff, "*")

			if all || ArrayContains(noDiff, "epoch") {
				if k == "xregistry-epoch" {
					headers[k] = "1"
				}
			}
			if all || ArrayContains(noDiff, "createdat") {
				if k == "xregistry-createdat" {
					headers[k] = "2000-01-01T12:00:00.00Z"
				}
			}
			if all || ArrayContains(noDiff, "modifiedat") {
				if k == "xregistry-modifiedat" {
					headers[k] = headers["xregistry-createdat"]
				}
			}
		}
	}

	makeImportObj := func(objAny any) {}
	makeImportObj = func(objAny any) {
		obj, ok := objAny.(map[string]any)
		if !ok || len(obj) == 0 {
			return
		}

		xidAny, ok := obj["xid"]
		if !ok {
			return
		}

		xidStr, ok := xidAny.(string)
		if !ok {
			return
		}

		// if there's an XID then it must be an object and not a collection
		xid, err := ParseXid(xidStr)
		if err != nil {
			Error(err)
		}

		// For all entities
		delete(obj, "self")
		delete(obj, "shortself")
		delete(obj, "xid")
		delete(obj, "epoch")

		switch xid.Type {
		case ENTITY_REGISTRY:
			delete(obj, "registryid")

			for _, gm := range reg.Model.Groups {
				delete(obj, gm.Plural+"url")
				delete(obj, gm.Plural+"count")

				if nestedObj, ok := obj[gm.Plural].(map[string]any); ok {
					for _, gObj := range nestedObj {
						makeImportObj(gObj)
					}
				}
			}

		case ENTITY_GROUP:
			gm := reg.Model.Groups[xid.Group]
			for _, rm := range gm.Resources {
				delete(obj, rm.Plural+"url")
				delete(obj, rm.Plural+"count")

				if nestedObj, ok := obj[rm.Plural].(map[string]any); ok {
					for _, rObj := range nestedObj {
						makeImportObj(rObj)
					}
				}
			}

		case ENTITY_RESOURCE:
			gm := reg.Model.Groups[xid.Group]
			rm := gm.Resources[xid.Resource]

			delete(obj, rm.Singular+"id")
			delete(obj, "versionid")
			delete(obj, "isdefault")
			delete(obj, "metaurl")
			delete(obj, "versionsurl")
			delete(obj, "versionscount")

			makeImportObj(obj["meta"])
			if nestedObj, ok := obj["versions"].(map[string]any); ok {
				for _, vObj := range nestedObj {
					makeImportObj(vObj)
				}
			}

		case ENTITY_META:
			gm := reg.Model.Groups[xid.Group]
			rm := gm.Resources[xid.Resource]

			delete(obj, rm.Singular+"id")
			delete(obj, "defaultversionurl")

		case ENTITY_VERSION:
			gm := reg.Model.Groups[xid.Group]
			rm := gm.Resources[xid.Resource]

			delete(obj, rm.Singular+"id")
			delete(obj, "isdefault")
			delete(obj, "versionid")
			delete(obj, "vesionsurl")
			delete(obj, "versionscount")
		}
	}

	downloadXidFn := func(xid *Xid, wait bool) ([]byte, *XRError) {
		if !wait && parallel > 1 {
			listCH <- xid
			return nil, nil
		}

		file := dir
		obj := map[string]any{}
		fname := xid.String()
		if xid.Type == ENTITY_RESOURCE || xid.Type == ENTITY_VERSION {
			fname += "$details"
		}

		data, _ := Download(reg, fname)
		if err := json.Unmarshal(data, &obj); err != nil {
			// fmt.Printf("JSON(%s): %s", fname, string(data))
			Error(NewXRError("parsing_response", reg.GetServerURL(),
				"error_detail="+err.Error()))
		}

		noDiffObj(obj)
		data, err = json.MarshalIndent(obj, "", "  ")
		Error(err)

		switch xid.Type {
		case ENTITY_REGISTRY:
			fallthrough
		case ENTITY_GROUP_TYPE:
			fallthrough
		case ENTITY_RESOURCE_TYPE:
			fallthrough
		case ENTITY_VERSION_TYPE:
			fallthrough
		case ENTITY_GROUP:
			fn := file + strings.TrimRight(xid.String(), "/") + "/" + indexFile
			Write(fn, data)
			Write(fn+".hdr", []byte("content-type: application/json"))

		case ENTITY_RESOURCE:
			fn := file + xid.String() + "$details"
			Write(fn, data)
			Write(fn+".hdr", []byte("content-type: application/json"))

			rm, xErr := reg.FindResourceModel(xid.Group, xid.Resource)
			Error(xErr)

			if rm.HasDocument != nil && *(rm.HasDocument) {
				fn = file + xid.String() + "/" + indexFile
				data, hdr := Download(reg, xid.String())
				Write(fn, data)

				if hdr != nil {
					self := host + xid.String()[1:]
					hdr["xregistry-self"] = self
					hdr["xregistry-versionsurl"] = self + "/versions"
					hdr["xregistry-metaurl"] = self + "/meta"
					if hdr["content-location"] != "" {
						cl := self + "/versions/" + hdr["xregistry-versionid"]
						hdr["content-location"] = cl
					}
					noDiffHeaders(hdr)

					fn = file + xid.String() + ".hdr"
					str := ""
					for _, k := range SortedKeys(hdr) {
						// Assume just one value per header
						str += fmt.Sprintf("%s:%s\n", k, hdr[k])
					}
					Write(fn, []byte(str))
				}

				fn = file + xid.String()
				if md2html && strings.HasSuffix(fn, ".md") {
					fn = fn[:len(fn)-2] + "html"
					html := bytes.Buffer{}

					html.Write([]byte("<html>\n"))

					// Header, if needed
					header := ""

					if !md2htmlNoStyle {
						header += "<style>\n" +
							"  .anchor {\n" +
							"    font-size: 12px ;\n" +
							"    vertical-align: middle ;\n" +
							"    text-decoration: none ;\n" +
							"  }\n" +
							"  body {\n" +
							"    font-family: sans-serif ;\n" +
							"    font-size: 16px ;\n" +
							"    line-height: 1.5 ;\n" +
							"    padding: 5% 10% 5% 10% ;\n" +
							"  }\n" +
							"  pre {\n" +
							"    font-size: 80% ;\n" +
							"    background-color: #f2f2f2 ;\n" +
							"    padding: 12px ;\n" +
							"  }\n" +
							"  code {\n" +
							"    font-size: 85% ;\n" +
							"    background-color: #f2f2f2 ;\n" +
							"    padding: .2em .4em ;\n" +
							"  }\n" +
							"  pre code {\n" +
							"    font-size: inherit ;\n" +
							"    background-color: inherit ;\n" +
							"    padding: 0px ;\n" +
							"  }\n" +
							"  table {\n" +
							"    border: 1px solid lightgray ;\n" +
							"    border-collapse: collapse ;\n" +
							"    border-spacing: 0px ;\n" +
							"    line-height: 24px ;\n" +
							"  }\n" +
							"  tr:nth-child(even) {\n" +
							"    background-color: #f2f2f2 ;\n" +
							"  }\n" +
							"  td,th {\n" +
							"    border: 1px solid lightgray ;\n" +
							"    padding: 5px ;\n" +
							"  }\n" +
							"  td code, th code {\n" +
							"    font-size: inherit ;\n" +
							"  }\n" +
							"</style>\n"
					}
					if md2htmlLink != "" {
						header += `<link rel="stylesheet" href="` +
							md2htmlLink + `">` + "\n"
					}
					if md2htmlHeader != "" {
						header += md2htmlHeader + "\n"
					}
					if header != "" {
						html.Write([]byte("<head>\n" + header + "</head>\n"))
					}

					// Custom HTML after <head>
					if md2htmlHTML != "" {
						html.Write([]byte(md2htmlHTML))
						if md2htmlHTML[len(md2htmlHTML)-1] != '\n' {
							html.Write([]byte("\n"))
						}
					}

					// Do the actual conversion from md->html
					md.Convert(data, &html)

					html.Write([]byte("\n</html>\n"))

					Error(os.WriteFile(fn, html.Bytes(), 0644))
				}
			} else {
				fn := file + xid.String() + "/" + indexFile
				Write(fn, data)
				Write(fn+".hdr", []byte("content-type: application/json"))
			}

		case ENTITY_META:
			fn := file + xid.String()
			Write(fn, data)
			Write(fn+".hdr", []byte("content-type: application/json"))

		case ENTITY_VERSION:
			fn := file + xid.String() + "$details"
			Write(fn, data)
			Write(fn+".hdr", []byte("content-type: application/json"))

			rm, xErr := reg.FindResourceModel(xid.Group, xid.Resource)
			Error(xErr)

			if rm.HasDocument != nil && *(rm.HasDocument) {
				fn = file + xid.String() + "/" + indexFile
				data, hdr := Download(reg, xid.String())
				Write(fn, data)

				if hdr != nil {
					self := host + xid.String()[1:]
					hdr["xregistry-self"] = self
					if hdr["content-location"] != "" {
						hdr["content-location"] = self
					}
					noDiffHeaders(hdr)

					fn = file + xid.String() + ".hdr"
					str := ""
					for _, k := range SortedKeys(hdr) {
						// Assume just one value per header
						str += fmt.Sprintf("%s:%s\n", k, hdr[k])
					}
					Write(fn, []byte(str))
				}

				fn = file + xid.String()
				if md2html && strings.HasSuffix(fn, ".md") {
					fn = fn[:len(fn)-2] + "html"
					html := bytes.Buffer{}
					md.Convert(data, &html)
					Error(os.WriteFile(fn, html.Bytes(), 0644))
				}
			} else {
				fn := file + xid.String() + "/" + indexFile
				Write(fn, data)
				Write(fn+".hdr", []byte("content-type: application/json"))
			}

		}

		return data, nil
	}

	// Process the listCH work-queue in parallel, signal(wg) when all done
	go func() {
		for {
			xid, ok := <-listCH
			if xid == nil && !ok {
				break
			}
			go func() {
				_, xErr := downloadXidFn(xid, true)
				Error(xErr)
			}()
		}
		wg.Done()
	}()

	for _, xidStr := range args {
		if len(xidStr) > 0 && xidStr[0] != '/' {
			xidStr = "/" + xidStr
		}
		xid, err := ParseXid(xidStr)
		Error(err)
		Error(traverseFromXid(reg, xid, dir, downloadXidFn))
	}
	close(listCH) // close work-queue

	data, _ := Download(reg, "/export")
	if len(data) > 0 {
		// If the user wants the "capabilities" to be modified for a static
		// web site then we need to update them in the /export output too
		// obj := map[string]json.RawMessage(nil)
		obj := map[string]any{}
		if err := json.Unmarshal(data, &obj); err != nil {
			Error(NewXRError("parsing_response",
				reg.GetServerURL()+"/export",
				"error_detail="+err.Error()))
		}

		if modCap {
			caps, xErr := ParseCapabilities([]byte(ToJSON(obj["capabilities"])))
			Error(xErr)

			caps.Available = map[string]*AvailableObject{
				"capabilities":        &AvailableObject{Mutable: false},
				"capabilitiesoffered": &AvailableObject{Mutable: false},
				"entities":            &AvailableObject{Mutable: false},
				"export":              &AvailableObject{Mutable: false},
				"model":               &AvailableObject{Mutable: false},
				"modelsource":         &AvailableObject{Mutable: false},
			}
			caps.Flags = nil
			caps.Pagination = false
			caps.ShortSelf = false
			obj["capabilities"] = caps
		}

		noDiffObj(obj)
		data, _ = json.MarshalIndent(obj, "", "  ")

		Write(dir+"/export", data)
		Write(dir+"/export.hdr", []byte("content-type: application/json"))
	}

	if importFile {
		obj := map[string]any{}
		if err := json.Unmarshal(data, &obj); err != nil {
			Error(NewXRError("parsing_response",
				reg.GetServerURL()+"/export",
				"error_detail="+err.Error()))
		}
		makeImportObj(obj)
		data, _ = json.MarshalIndent(obj, "", "  ")

		Write(dir+"/import.json", data)
		Write(dir+"/import.json.hdr", []byte("content-type: application/json"))
	}

	data, _ = Download(reg, "/model")
	if len(data) > 0 {
		Write(dir+"/model", data)
		Write(dir+"/model.hdr", []byte("content-type: application/json"))
	}

	data, _ = Download(reg, "/modelsource")
	if len(data) > 0 {
		Write(dir+"/modelsource", data)
		Write(dir+"/modelsource.hdr", []byte("content-type: application/json"))
	}

	data, _ = Download(reg, "/capabilities")
	if len(data) > 0 {
		if modCap {
			caps, xErr := ParseCapabilities(data)
			Error(xErr)

			caps.Available = map[string]*AvailableObject{
				"capabilities":        &AvailableObject{Mutable: false},
				"capabilitiesoffered": &AvailableObject{Mutable: false},
				"entities":            &AvailableObject{Mutable: false},
				"export":              &AvailableObject{Mutable: false},
				"model":               &AvailableObject{Mutable: false},
				"modelsource":         &AvailableObject{Mutable: false},
			}
			caps.Flags = nil
			caps.Pagination = false
			data, _ = json.MarshalIndent(caps, "", "  ")
		}

		Write(dir+"/capabilities", data)
		Write(dir+"/capabilities.hdr", []byte("content-type: application/json"))
	}

	data, _ = Download(reg, "/capabilitiesoffered")
	if len(data) > 0 {
		Write(dir+"/capabilitiesoffered", data)
		Write(dir+"/capabilitiesoffered.hdr",
			[]byte("content-type: application/json"))
	}

	// Just incase the queue is still processing
	wg.Wait()
}

// Body, Headers
func Download(reg *xrlib.Registry, path string) ([]byte, map[string]string) {
	res, xErr := reg.HttpDo(VerboseCount > 1, "GET", path, nil)
	Error(xErr)

	headers := (map[string]string)(nil)
	// Only save if we have xRegistry headers, but also save special headers
	if res.Header.Get("xregistry-self") != "" {
		headers = map[string]string{}
		saveHeaders := map[string]bool{
			"content-type":        true,
			"content-disposition": true,
			"content-length":      true,
			"content-location":    true,
		}
		for k, _ := range res.Header {
			k = strings.ToLower(k)
			if strings.HasPrefix(k, "xregistry-") || saveHeaders[k] {
				// Assume just one value per header
				headers[k] = res.Header.Get(k)
			}
		}
	}

	return res.Body, headers
}

func Write(file string, data []byte) {
	Verbose("Writing: %s", file)
	Error(os.MkdirAll(filepath.Dir(file), 0774))
	Error(os.WriteFile(file, data, 0644))
}

type traverseFunc func(xid *Xid, wait bool) ([]byte, *XRError)

func traverseFromXid(reg *xrlib.Registry, xid *Xid, root string, fn traverseFunc) *XRError {
	switch xid.Type {
	case ENTITY_REGISTRY:
		fn(xid, false)

		gList, xErr := reg.ListGroupModels()
		Error(xErr)
		sort.Strings(gList)
		for _, gName := range gList {
			nextXid, err := xid.AddPath(gName)
			Error(err)
			traverseFromXid(reg, nextXid, root, fn)
		}

	case ENTITY_GROUP_TYPE:
		fallthrough
	case ENTITY_RESOURCE_TYPE:
		fallthrough
	case ENTITY_VERSION_TYPE:
		data, xErr := fn(xid, true)
		Error(xErr)

		tmp := map[string]any{}
		if err := json.Unmarshal([]byte(data), &tmp); err != nil {
			Error(NewXRError("parsing_response",
				reg.GetServerURL()+xid.String(),
				"error_detail="+err.Error()))
		}

		vList := SortedKeys(tmp)
		for _, vName := range vList {
			nextXid, err := xid.AddPath(vName)
			Error(err)
			traverseFromXid(reg, nextXid, root, fn)
		}

	case ENTITY_GROUP:
		fn(xid, false)

		gm, xErr := reg.FindGroupModel(xid.Group)
		Error(xErr)
		for _, rName := range SortedKeys(gm.Resources) {
			nextXid, err := xid.AddPath(rName)
			Error(err)
			traverseFromXid(reg, nextXid, root, fn)
		}

	case ENTITY_RESOURCE:
		fn(xid, false)

		nextXid, err := xid.AddPath("meta")
		Error(err)
		traverseFromXid(reg, nextXid, root, fn)

		nextXid, err = xid.AddPath("versions")
		Error(err)
		traverseFromXid(reg, nextXid, root, fn)

	case ENTITY_META:
		fn(xid, false)

	case ENTITY_VERSION:
		fn(xid, false)

	}

	return nil
}
