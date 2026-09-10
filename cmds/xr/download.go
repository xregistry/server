package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"sync"
	// "net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
		Use:     "download DIR [XID...]",
		Short:   `Download entities from registry as individual files`,
		Run:     downloadFunc,
		GroupID: "Entities",

		Annotations: map[string]string{
			"usage": `
Notes:
  - Primary use case: download the files for use in a static file/web server.
  - XID may also include the following:
    /capabilities /capabilitiesoffered /export /model /modelsource (or --all)
  - Use --min to minimize the files created by removing static/duplicate info.
    Primary use case: storing files in a repository for manual edits. Default
    --index changed from "index.html" to "document".`,
		},
	}

	downloadCmd.Flags().BoolP("all", "a", false,
		"Download all data (e.g. export, model)")
	downloadCmd.Flags().StringP("url", "u", "",
		"Host/path to update xRegistry paths")
	downloadCmd.Flags().BoolP("import", "", false,
		"Create '/import.json' based on /export")
	downloadCmd.Flags().StringP("index", "i", "",
		"Directory index file name (index.html*)")
	downloadCmd.Flag("index").DefValue = "" // hide default text
	downloadCmd.Flags().BoolP("md2html-no-style", "", false,
		"Do not add default styling to html files")
	downloadCmd.Flags().BoolP("md2html", "", false,
		"Generate HTML files for MD files")
	downloadCmd.Flags().StringP("md2html-css-link", "", "",
		"CSS stylesheet 'link' to add in md2html files")
	downloadCmd.Flags().StringP("md2html-header", "", "",
		"HTML to add in <head> (data,@FILE,@URL,@-)")
	downloadCmd.Flags().StringP("md2html-html", "", "",
		"HTML to add after <head> (data,@FILE,@URL,@-)")
	downloadCmd.Flags().BoolP("min", "", false,
		"Minimize the data (e.g. no collection json)")
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

	if dir == "/" {
		Error(NewXRError("client_error", dir,
			"error_detail=Can't use '/' as the dir for safety. Use '/./' to force it"))
	}

	stat, err := os.Stat(dir)
	if os.IsNotExist(err) || !stat.IsDir() {
		Error(NewXRError("client_error", dir,
			"error_detail="+dir+" must be an existing directory"))
	}
	args = args[1:]

	all, _ := cmd.Flags().GetBool("all")
	md2html, _ := cmd.Flags().GetBool("md2html")
	md2htmlNoStyle, _ := cmd.Flags().GetBool("md2html-no-style")
	md2htmlLink, _ := cmd.Flags().GetString("md2html-css-link")
	md2htmlHeader, _ := cmd.Flags().GetString("md2html-header")
	md2htmlHTML, _ := cmd.Flags().GetString("md2html-html")
	minimal, _ := cmd.Flags().GetBool("min")

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
	if indexFile == "" {
		if minimal {
			indexFile = "document"
		} else {
			indexFile = "index.html"
		}
	}

	host, _ := cmd.Flags().GetString("url")
	modCap, _ := cmd.Flags().GetBool("capabilities")
	noDiff, _ := cmd.Flags().GetStringSlice("nodiff")
	importFile, _ := cmd.Flags().GetBool("import")

	if host != "" {
		if host[len(host)-1] != '/' {
			host += "/"
		}
	}

	maxThreads, _ := cmd.Flags().GetInt("parallel")
	if maxThreads < 1 {
		Error("--parallel must be greater than zero")
	}
	traverseQueue = make(chan bool, maxThreads)

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
			allNoDiff := ArrayContains(noDiff, "*")

			if allNoDiff || ArrayContains(noDiff, "epoch") {
				if _, ok := obj["epoch"]; ok {
					obj["epoch"] = 1
				}
			}
			if allNoDiff || ArrayContains(noDiff, "createdat") {
				if _, ok := obj["createdat"]; ok {
					obj["createdat"] = `2000-01-01T12:00:00.00Z`
				}
			}
			// Must come after "createdat" processing
			if allNoDiff || ArrayContains(noDiff, "modifiedat") {
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
			allNoDiff := ArrayContains(noDiff, "*")

			if allNoDiff || ArrayContains(noDiff, "epoch") {
				if k == "xregistry-epoch" {
					headers[k] = "1"
				}
			}
			if allNoDiff || ArrayContains(noDiff, "createdat") {
				if k == "xregistry-createdat" {
					headers[k] = "2000-01-01T12:00:00.00Z"
				}
			}
			if allNoDiff || ArrayContains(noDiff, "modifiedat") {
				if k == "xregistry-modifiedat" {
					headers[k] = headers["xregistry-createdat"]
				}
			}
		}
	}

	makeImportObj := func(objAny any, minimal bool) {}
	makeImportObj = func(objAny any, minimal bool) {
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

		if minimal {
			delete(obj, "createdat")
			delete(obj, "modifiedat")
		}

		switch xid.Type {
		case ENTITY_REGISTRY:
			delete(obj, "registryid")

			if minimal {
				delete(obj, "specversion")
			}

			for _, gm := range reg.Model.Groups {
				delete(obj, gm.Plural+"url")
				delete(obj, gm.Plural+"count")
			}

		case ENTITY_GROUP:
			gm := reg.Model.Groups[xid.Group]

			delete(obj, gm.Singular+"id")

			for _, rm := range gm.Resources {
				delete(obj, rm.Plural+"url")
				delete(obj, rm.Plural+"count")
			}

		case ENTITY_RESOURCE:
			gm := reg.Model.Groups[xid.Group]
			rm := gm.Resources[xid.Resource]

			if rm.GetMaxVersions() == 1 {
				delete(obj, rm.Singular+"id")
				delete(obj, "versionid")
				delete(obj, "ancestorid")
				delete(obj, "isdefault")
				delete(obj, "metaurl")
				delete(obj, "versionscount")
				delete(obj, "versionsurl")
			} else {
				for attr, _ := range obj {
					delete(obj, attr)
				}
			}

		case ENTITY_META:
			gm := reg.Model.Groups[xid.Group]
			rm := gm.Resources[xid.Resource]

			delete(obj, rm.Singular+"id")
			delete(obj, "defaultversionurl")

			if minimal {
				if obj["readonly"] == false {
					delete(obj, "readonly")
				}
				if obj["defaultversionsticky"] == false {
					delete(obj, "defaultversionsticky")
				}
			}

		case ENTITY_VERSION:
			gm := reg.Model.Groups[xid.Group]
			rm := gm.Resources[xid.Resource]

			delete(obj, rm.Singular+"id")
			delete(obj, "isdefault")
			delete(obj, "versionid")
		}
	}

	traverseFromXid = func(reg *xrlib.Registry, xid *Xid, root string) *XRError {
		// fmt.Printf("In %q\n", xid)
		// defer fmt.Printf("Out %q\n", xid)

		obj := map[string]any{}
		fname := xid.String()
		if xid.Type == ENTITY_RESOURCE || xid.Type == ENTITY_VERSION {
			fname += "$details"
		}

		data, _ := Download(reg, fname)
		if err := json.Unmarshal(data, &obj); err != nil {
			// fmt.Printf("JSON(%s): %s", fname, string(data))
			Error(NewXRError("client_error", "",
				"error_detail="+
					fmt.Sprintf("%q doesn't appear to be an xRegistry entity",
						xid)))

			Error(NewXRError("parsing_response", reg.GetServerURL(),
				"error_detail="+err.Error()))
		}

		noDiffObj(obj)

		if minimal {
			makeImportObj(obj, true)
		}

		data, err := json.MarshalIndent(obj, "", "  ")
		Error(err)

		// just to look nice
		if len(data) > 0 && data[len(data)-1] != '\n' {
			data = append(data, '\n')
		}

		switch xid.Type {
		case ENTITY_REGISTRY:
			fn := root + strings.TrimRight(xid.String(), "/")

			if !minimal || len(obj) > 0 {
				fn := fn + "/" + indexFile
				Write(fn, data)
				if !minimal {
					Write(fn+".hdr", []byte("content-type: application/json"))
				}
			}

			gList, xErr := reg.ListGroupModels()
			Error(xErr)
			sort.Strings(gList)
			for _, gName := range gList {
				nextXid, err := xid.AddPath(gName)
				Error(err)

				Enqueue(reg, nextXid, root)
			}

		case ENTITY_GROUP_TYPE:
			fallthrough
		case ENTITY_RESOURCE_TYPE:
			fallthrough
		case ENTITY_VERSION_TYPE:
			fn := root + strings.TrimRight(xid.String(), "/")

			if !minimal || !xid.IsCollectionType() {
				if !minimal || len(obj) > 0 {
					fn := fn + "/" + indexFile
					Write(fn, data)
					Write(fn+".hdr", []byte("content-type: application/json"))
				}
			}

			tmp := map[string]any{}
			if err := json.Unmarshal([]byte(data), &tmp); err != nil {
				Error(NewXRError("parsing_response",
					reg.GetServerURL()+xid.String(),
					"error_detail="+err.Error()))
			}

			// if minimal && xid.Type == ENTITY_VERSION_TYPE && len(tmp) == 1 {
			// break
			// }

			vList := SortedKeys(tmp)
			for _, vName := range vList {
				nextXid, err := xid.AddPath(vName)
				Error(err)
				Enqueue(reg, nextXid, root)
			}

		case ENTITY_GROUP:
			fn := root + strings.TrimRight(xid.String(), "/")
			// Always create the group dir
			Error(os.MkdirAll(fn, 0774))

			if !minimal || len(obj) > 0 {
				fn := fn + "/" + indexFile
				Write(fn, data)
				if !minimal {
					Write(fn+".hdr", []byte("content-type: application/json"))
				}
			}

			gm, xErr := reg.FindGroupModel(xid.Group)
			Error(xErr)
			if gm != nil {
				for _, rName := range SortedKeys(gm.Resources) {
					nextXid, err := xid.AddPath(rName)
					Error(err)
					Enqueue(reg, nextXid, root)
				}
			}

		case ENTITY_RESOURCE:
			rm, xErr := reg.FindResourceModel(xid.Group, xid.Resource)
			Error(xErr)

			// If !hasDoc then at least show {} for the resource
			if !minimal && len(obj) > 0 { // || !rm.GetHasDocument() {
				fn := root + xid.String() + "$details"
				Write(fn, data)
				if !minimal {
					Write(fn+".hdr", []byte("content-type: application/json"))
				}
			}

			if !minimal && rm.GetHasDocument() {
				fn := root + xid.String() + "/" + indexFile
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

					fn = root + xid.String() + ".hdr"
					str := ""
					for _, k := range SortedKeys(hdr) {
						// Assume just one value per header
						str += fmt.Sprintf("%s:%s\n", k, hdr[k])
					}
					Write(fn, []byte(str))
				}

				// Now create the html file if needed
				fn = root + xid.String()
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
						html.Write([]byte("<head>\n" + header +
							"\n</head>\n"))
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
				} // EO-html file generation
			} else {
				if !minimal || len(obj) > 0 {
					fn := root + xid.String() + "/" + indexFile
					Write(fn, data)
					if !minimal {
						Write(fn+".hdr", []byte("content-type: application/json"))
					}
				}
			}

			nextXid, err := xid.AddPath("meta")
			Error(err)
			Enqueue(reg, nextXid, root)

			nextXid, err = xid.AddPath("versions")
			Error(err)
			Enqueue(reg, nextXid, root)

		case ENTITY_META:
			if !minimal || len(obj) > 0 {
				fn := root + xid.String()
				Write(fn, data)
				if !minimal {
					Write(fn+".hdr", []byte("content-type: application/json"))
				}
			}

		case ENTITY_VERSION:
			rm, xErr := reg.FindResourceModel(xid.Group, xid.Resource)
			Error(xErr)

			// If !hasDoc then at least show {} for the version
			if len(obj) > 0 || !minimal || !rm.GetHasDocument() {
				fn := root + xid.String() + "$details"
				Write(fn, data)
				if !minimal {
					Write(fn+".hdr", []byte("content-type: application/json"))
				}
			}

			if rm.GetHasDocument() {
				fn := root + xid.String() + "/" + indexFile
				data, hdr := Download(reg, xid.String())
				Write(fn, data)

				if !minimal {
					if hdr != nil {
						self := host + xid.String()[1:]
						hdr["xregistry-self"] = self
						if hdr["content-location"] != "" {
							hdr["content-location"] = self
						}
						noDiffHeaders(hdr)

						if !minimal {
							fn = root + xid.String() + ".hdr"
							str := ""
							for _, k := range SortedKeys(hdr) {
								// Assume just one value per header
								str += fmt.Sprintf("%s:%s\n", k, hdr[k])
							}
							Write(fn, []byte(str))
						}
					}
				}

				// For now don't generate the version's html file because
				// the relative links only work when it's at the Resource level
				/*
					if md2html && strings.HasSuffix(xid.ResourceID, ".md") {
						// Use versionid as base filename, not resourceid
						fn = root + xid.String() + ".html"
						html := bytes.Buffer{}

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
							html.Write([]byte("<head>\n" + header +
								"\n</head>\n"))
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
				*/
			} else {
				if !minimal && len(obj) > 0 {
					fn := root + xid.String() + "/" + indexFile
					Write(fn, data)
					if !minimal {
						Write(fn+".hdr",
							[]byte("content-type: application/json"))
					}
				}
			}
		}

		return nil
	}

	if len(args) == 0 {
		args = []string{"/"}
	} else {
		for i, arg := range args {
			if len(arg) > 0 && arg[0] != '/' {
				args[i] = "/" + arg
			}
		}
	}

	if all {
		for _, xid := range []string{"/capabilities", "/capabilitiesoffered",
			"/export", "/model", "/modelsource"} {

			if !ArrayContains(args, xid) {
				args = append(args, xid)
			}
		}
	}

	// Sort and remove dups
	slices.Sort(args)
	args = slices.Compact(args)

	exportData := []byte{}

	go ProcessWorkQueue()
	for _, xidStr := range args {
		switch xidStr {
		case "/capabilities":
			data, _ := Download(reg, "/capabilities")
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
			if !minimal {
				Write(dir+"/capabilities.hdr",
					[]byte("content-type: application/json"))
			}

		case "/capabilitiesoffered":
			data, _ := Download(reg, "/capabilitiesoffered")
			Write(dir+"/capabilitiesoffered", data)
			if !minimal {
				Write(dir+"/capabilitiesoffered.hdr",
					[]byte("content-type: application/json"))
			}

		case "/export":
			exportData, _ := Download(reg, "/export")
			if len(exportData) > 0 {
				// If the user wants the "capabilities" to be modified for a
				// static web site then we need to update them in the /export
				// output too
				obj := map[string]any{}
				if err := json.Unmarshal(exportData, &obj); err != nil {
					Error(NewXRError("parsing_response",
						reg.GetServerURL()+"/export",
						"error_detail="+err.Error()))
				}

				if modCap {
					caps, xErr := ParseCapabilities([]byte(
						ToJSON(obj["capabilities"])))
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
				exportData, _ = json.MarshalIndent(obj, "", "  ")

				Write(dir+"/export", exportData)
				if !minimal {
					Write(dir+"/export.hdr",
						[]byte("content-type: application/json"))
				}
			}

		case "/model":
			data, _ := Download(reg, "/model")
			Write(dir+"/model", data)
			if !minimal {
				Write(dir+"/model.hdr",
					[]byte("content-type: application/json"))
			}

		case "/modelsource":
			modelSrc, _ := Download(reg, "/modelsource")
			Write(dir+"/modelsource", modelSrc)
			if !minimal {
				Write(dir+"/modelsource.hdr",
					[]byte("content-type: application/json"))
			}

		default:
			// Must be normal entity in the registry
			xid, err := ParseXid(xidStr)
			Error(err)
			Enqueue(reg, xid, dir)
		}
	}

	traverseWG.Wait()
	close(traverseQueue)

	if importFile {
		// See if we already download "export", use it if so
		data := []byte{}
		if len(exportData) > 0 {
			data = exportData
		} else {
			data, _ = Download(reg, "/export")
		}

		obj := map[string]any{}
		if err := json.Unmarshal(data, &obj); err != nil {
			Error(NewXRError("parsing_response",
				reg.GetServerURL()+"/export",
				"error_detail="+err.Error()))
		}

		makeImportObj(obj, minimal)
		data, _ = json.MarshalIndent(obj, "", "  ")

		Write(dir+"/import.json", data)
		if !minimal {
			Write(dir+"/import.json.hdr",
				[]byte("content-type: application/json"))
		}
	}
}

// Body, Headers
func Download(reg *xrlib.Registry, path string) ([]byte, map[string]string) {
	res, xErr := reg.HttpDo(VerboseCount > 1, "GET", path, nil)
	// if xErr != nil {
	// return nil, nil
	// }
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

var traverseQueue = make(chan bool, 10)
var traverseWG = sync.WaitGroup{}

type QueueItem struct {
	reg  *xrlib.Registry
	xid  *Xid
	root string
}

var WorkQueue = []*QueueItem{}
var WorkLock = sync.RWMutex{}

func Enqueue(reg *xrlib.Registry, xid *Xid, root string) {
	// fmt.Printf("Enqueuing: %q\n", xid)
	WorkLock.Lock()
	defer WorkLock.Unlock()

	WorkQueue = append(WorkQueue, &QueueItem{
		reg:  reg,
		xid:  xid,
		root: root,
	})
	traverseWG.Add(1)
}

func Dequeue() *QueueItem {
	WorkLock.Lock()
	defer WorkLock.Unlock()

	if len(WorkQueue) == 0 {
		return nil
	}
	next := WorkQueue[0]
	WorkQueue = WorkQueue[1:]
	return next
}

func QueueSize() int {
	WorkLock.Lock()
	defer WorkLock.Unlock()
	return len(WorkQueue)
}

type traverseFunc func(xid *Xid) ([]byte, *XRError)

var traverseFromXid = (func(reg *xrlib.Registry, xid *Xid, root string) *XRError)(nil)

func ProcessWorkQueue() {
	stop := false
	for stop == false {
		if QueueSize() == 0 {
			time.Sleep(5 * time.Millisecond)
			continue
		}

		// At least one item is waiting. Wait for a free worker slot to open
		traverseQueue <- true
		qi := Dequeue()

		go func() {
			defer func() {
				_, ok := <-traverseQueue
				traverseWG.Done()
				if !ok {
					stop = true
				}
			}()
			traverseFromXid(qi.reg, qi.xid, qi.root)
		}()
	}
	fmt.Printf("Exiting processworkqueue\n")
}
