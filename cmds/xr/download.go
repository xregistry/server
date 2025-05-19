package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	// "net/http"
	"os"
	"path/filepath"
	"strings"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	"github.com/xregistry/server/registry"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	ghtml "github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/anchor"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		&anchor.Extender{
			// Texter: anchor.Text("🔗"),
			Texter:   anchor.Text("☍"),
			Position: anchor.Before, // or anchor.After
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
		Use:     "download DIR [ XID...]",
		Short:   "Download entities from registry as individual files",
		Run:     downloadFunc,
		GroupID: "Entities",
	}
	downloadCmd.Flags().StringP("url", "u", "",
		"Host/path to Update xRegistry paths")
	downloadCmd.Flags().StringP("index", "i", "index.html",
		"Directory index file name")
	downloadCmd.Flags().BoolP("md2html", "m", false,
		"Generate HTML files for MD files")
	downloadCmd.Flags().StringP("md2html-css-link", "", "",
		"CSS stylesheet 'link' to add in md2html files")
	downloadCmd.Flags().StringP("md2html-header", "", "",
		"HTML to add in <head> of md2html files (data,@FILE,@URL,@-)")
	downloadCmd.Flags().StringP("md2html-html", "", "",
		"HTML to add after <head> in md2html files (data,@FILE,@URL,@-)")
	downloadCmd.Flags().BoolP("capabilities", "c", false,
		"Modify capabilities for static site")

	parent.AddCommand(downloadCmd)
}

func downloadFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	if len(args) == 0 {
		Error("Missing the DIR argument")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	dir := args[0]
	stat, err := os.Stat(dir)
	if os.IsNotExist(err) || !stat.IsDir() {
		Error("%q must be an existing directory", dir)
	}
	args = args[1:]

	if len(args) == 0 {
		args = []string{"/"}
	}

	md2html, _ := cmd.Flags().GetBool("md2html")
	md2htmlLink, _ := cmd.Flags().GetString("md2html-css-link")
	md2htmlHeader, _ := cmd.Flags().GetString("md2html-header")
	md2htmlHTML, _ := cmd.Flags().GetString("md2html-html")

	if md2htmlHeader != "" {
		if md2htmlHeader[0] == '@' {
			buf, err := xrlib.ReadFile(md2htmlHeader[1:])
			Error(err)
			md2htmlHeader = string(buf)
		}
	}

	if md2htmlHTML != "" {
		if md2htmlHTML[0] == '@' {
			buf, err := xrlib.ReadFile(md2htmlHTML[1:])
			Error(err)
			md2htmlHTML = string(buf)
		}
	}

	indexFile, _ := cmd.Flags().GetString("index")
	host, _ := cmd.Flags().GetString("url")
	modCap, _ := cmd.Flags().GetBool("capabilities")
	if host != "" {
		if host[len(host)-1] != '/' {
			host += "/"
		}
	}

	downloadXIDFn := func(xid *xrlib.XID) ([]byte, error) {
		obj := map[string]json.RawMessage{}
		plurals := []string{}

		file := dir
		data := []byte(nil)
		switch xid.Type {
		case xrlib.ENTITY_REGISTRY:
			data, _ = Download(reg, xid.String())

			Error(json.Unmarshal(data, &obj),
				"%q doesn't appear to be an xRegistry server",
				reg.GetServerURL())

			if host != "" {
				obj["self"] = []byte(fmt.Sprintf("%q", host))
				list, err := reg.ListGroupModels()
				Error(err)
				for _, gmName := range list {
					gm, err := reg.FindGroupModel(gmName)
					Error(err)
					obj[gm.Plural+"url"] =
						[]byte(fmt.Sprintf("%q", host+gm.Plural))
				}
				data, err = json.MarshalIndent(obj, "", "  ")
				Error(err)
			}
			fn := file + "/" + indexFile
			Write(fn, data)
			Write(fn+".hdr", []byte("content-type: application/json"))

		case xrlib.ENTITY_GROUP_TYPE:
			gm, err := reg.FindGroupModel(xid.Group)
			Error(err)

			rList := gm.GetResourceList()
			for _, rName := range rList {
				plurals = append(plurals, rName) // rm.Plural)
			}
			fallthrough
		case xrlib.ENTITY_RESOURCE_TYPE:
			if xid.Type == xrlib.ENTITY_RESOURCE_TYPE {
				plurals = append(plurals, "versions")
			}
			fallthrough
		case xrlib.ENTITY_VERSION_TYPE:
			data, _ = Download(reg, xid.String())

			if host != "" {
				Error(json.Unmarshal(data, &obj))
				for k, d2 := range obj {
					tmp := map[string]json.RawMessage{}
					Error(json.Unmarshal(d2, &tmp))

					self := host + xid.String()[1:] + "/" + k
					tmp["self"] = []byte(fmt.Sprintf("%q", self))

					if _, ok := tmp["metaurl"]; ok {
						tmp["metaurl"] = []byte(fmt.Sprintf("\"%s/meta\"", self))
					}

					sort.Strings(plurals)
					for _, p := range plurals {
						pURL := fmt.Sprintf("%s/%s", self, p)
						tmp[p+"url"] = []byte(fmt.Sprintf("%q", pURL))
					}

					b, err := json.Marshal(tmp)
					Error(err)
					obj[k] = b
				}
				data, err = json.MarshalIndent(obj, "", "  ")
				Error(err)
			}

			fn := file + xid.String() + "/" + indexFile
			Write(fn, data)
			Write(fn+".hdr", []byte("content-type: application/json"))

		case xrlib.ENTITY_GROUP:
			data, _ = Download(reg, xid.String())
			if host != "" {
				Error(json.Unmarshal(data, &obj))
				self := host + xid.String()[1:]
				obj["self"] = []byte(fmt.Sprintf("%q", self))
				gm, err := reg.FindGroupModel(xid.Group)
				Error(err)
				rList := gm.GetResourceList()
				for _, rName := range rList {
					p := fmt.Sprintf(`"%s/%s"`, self, rName) // rm.Plural)
					// obj[rm.Plural+"url"] = []byte(p)
					obj[rName+"url"] = []byte(p)
				}
				data, err = json.MarshalIndent(obj, "", "  ")
				Error(err)
			}
			fn := file + xid.String() + "/" + indexFile
			Write(fn, data)
			Write(fn+".hdr", []byte("content-type: application/json"))

		case xrlib.ENTITY_RESOURCE:
			data, _ = Download(reg, xid.String()+"$details")
			if host != "" {
				Error(json.Unmarshal(data, &obj))
				self := host + xid.String()[1:]
				obj["self"] = []byte(fmt.Sprintf("%q", self))
				obj["versionsurl"] = []byte(`"` + self + "/versions" + `"`)
				obj["metaurl"] = []byte(`"` + self + "/meta" + `"`)
				data, err = json.MarshalIndent(obj, "", "  ")
				Error(err)
			} else {
				Error(json.Unmarshal(data, &obj))
				data, err = json.MarshalIndent(obj, "", "  ")
			}
			fn := file + xid.String() + "$details"
			Write(fn, data)
			Write(fn+".hdr", []byte("content-type: application/json"))

			rm, err := reg.FindResourceModel(xid.Group, xid.Resource)
			Error(err)
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

					fn = file + xid.String() + ".hdr"
					str := ""
					for _, k := range registry.SortedKeys(hdr) {
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

		case xrlib.ENTITY_META:
			data, _ = Download(reg, xid.String())
			if host != "" {
				Error(json.Unmarshal(data, &obj))
				self := host + xid.String()[1:]
				obj["self"] = []byte(fmt.Sprintf("%q", self))
				verid := ""
				Error(json.Unmarshal(obj["defaultversionid"], &verid))
				ver := fmt.Sprintf(`"%s/versions/%s"`, self[:len(self)-5],
					verid)
				obj["defaultversionurl"] = []byte(ver)
				data, err = json.MarshalIndent(obj, "", "  ")
				Error(err)
			} else {
				Error(json.Unmarshal(data, &obj))
				data, err = json.MarshalIndent(obj, "", "  ")
			}
			fn := file + xid.String()
			Write(fn, data)
			Write(fn+".hdr", []byte("content-type: application/json"))

		case xrlib.ENTITY_VERSION:
			data, _ = Download(reg, xid.String()+"$details")
			if host != "" {
				Error(json.Unmarshal(data, &obj))
				self := host + xid.String()[1:]
				obj["self"] = []byte(fmt.Sprintf("%q", self))
				data, err = json.MarshalIndent(obj, "", "  ")
				Error(err)
			} else {
				Error(json.Unmarshal(data, &obj))
				data, err = json.MarshalIndent(obj, "", "  ")
			}
			fn := file + xid.String() + "$details"
			Write(fn, data)
			Write(fn+".hdr", []byte("content-type: application/json"))

			rm, err := reg.FindResourceModel(xid.Group, xid.Resource)
			Error(err)
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

					fn = file + xid.String() + ".hdr"
					str := ""
					for _, k := range registry.SortedKeys(hdr) {
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

	for _, xidStr := range args {
		xid, err := xrlib.ParseXID(xidStr)
		Error(err)
		Error(traverseFromXID(reg, xid, dir, downloadXIDFn))
	}

	data, _ := Download(reg, "/export")
	if len(data) > 0 {
		// If the user wants the "capabilities" to be modified for a static
		// web site then we need to update them in the /export output too
		if modCap {
			tmpData := map[string]json.RawMessage(nil)
			Error(json.Unmarshal(data, &tmpData))
			caps, err := registry.ParseCapabilitiesJSON(tmpData["capabilities"])
			Error(err)
			caps.Flags = nil
			caps.Mutable = nil
			caps.Pagination = false
			tmpData["capabilities"], _ = json.Marshal(caps)
			data, _ = json.MarshalIndent(tmpData, "", "  ")
		}
		Write(dir+"/export", data)
		Write(dir+"/export.hdr", []byte("content-type: application/json"))
	}

	data, _ = Download(reg, "/model")
	if len(data) > 0 {
		Write(dir+"/model", data)
		Write(dir+"/model.hdr", []byte("content-type: application/json"))
	}

	data, _ = Download(reg, "/capabilities")
	if len(data) > 0 {
		if modCap {
			caps, err := registry.ParseCapabilitiesJSON(data)
			Error(err)
			caps.Flags = nil
			caps.Mutable = nil
			caps.Pagination = false
			data, _ = json.MarshalIndent(caps, "", "  ")
		}

		Write(dir+"/capabilities", data)
		Write(dir+"/capabilities.hdr", []byte("content-type: application/json"))
	}
}

// Body, Headers
func Download(reg *xrlib.Registry, path string) ([]byte, map[string]string) {
	res, err := reg.HttpDo("GET", path, nil)
	Error(errors.Unwrap(err))

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

type traverseFunc func(xid *xrlib.XID) ([]byte, error)

func traverseFromXID(reg *xrlib.Registry, xid *xrlib.XID, root string, fn traverseFunc) error {
	switch xid.Type {
	case xrlib.ENTITY_REGISTRY:
		fn(xid)

		gList, err := reg.ListGroupModels()
		Error(err)
		sort.Strings(gList)
		for _, gName := range gList {
			nextXID, err := xrlib.ParseXID(xid.String() + "/" + gName)
			Error(err)
			traverseFromXID(reg, nextXID, root, fn)
		}

	case xrlib.ENTITY_GROUP_TYPE:
		fallthrough
	case xrlib.ENTITY_RESOURCE_TYPE:
		fallthrough
	case xrlib.ENTITY_VERSION_TYPE:
		data, err := fn(xid)
		Error(err)

		tmp := map[string]any{}
		Error(json.Unmarshal([]byte(data), &tmp))

		vList := registry.SortedKeys(tmp)
		for _, vName := range vList {
			nextXID, err := xrlib.ParseXID(xid.String() + "/" + vName)
			Error(err)
			traverseFromXID(reg, nextXID, root, fn)
		}

	case xrlib.ENTITY_GROUP:
		fn(xid)

		gm, err := reg.FindGroupModel(xid.Group)
		Error(err)
		rList := gm.GetResourceList()
		sort.Strings(rList)
		for _, rName := range rList {
			nextXID, err := xrlib.ParseXID(xid.String() + "/" + rName)
			Error(err)
			traverseFromXID(reg, nextXID, root, fn)
		}

	case xrlib.ENTITY_RESOURCE:
		fn(xid)

		nextXID, err := xrlib.ParseXID(xid.String() + "/meta")
		Error(err)
		traverseFromXID(reg, nextXID, root, fn)

		nextXID, err = xrlib.ParseXID(xid.String() + "/versions")
		Error(err)
		traverseFromXID(reg, nextXID, root, fn)

	case xrlib.ENTITY_META:
		fn(xid)

	case xrlib.ENTITY_VERSION:
		fn(xid)

	}

	return nil
}
