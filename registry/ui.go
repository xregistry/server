package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

type PageWriter struct {
	Info      *RequestInfo
	OldWriter HTTPWriter
	Headers   *map[string][]string
	Buffer    *bytes.Buffer
}

func NewPageWriter(info *RequestInfo) *PageWriter {
	return &PageWriter{
		Info:      info,
		OldWriter: info.HTTPWriter,
		Headers:   &map[string][]string{},
		Buffer:    &bytes.Buffer{},
	}
}

func (pw *PageWriter) Write(b []byte) (int, error) {
	return pw.Buffer.Write(b)
}

func (pw *PageWriter) SetHeader(name, value string) {
	(*pw.Headers)[name] = []string{value}
}

func (pw *PageWriter) AddHeader(name, value string) {
	(*pw.Headers)[name] = append((*pw.Headers)[name], value)
}

func (pw *PageWriter) GetHeader(name string) string {
	vals := (*pw.Headers)[name]
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

func (pw *PageWriter) GetHeaderValues(name string) []string {
	return (*pw.Headers)[name]
}

func (pw *PageWriter) Done() {
	pw.SetHeader("Content-Type", "text/html")

	for k, values := range *pw.Headers {
		for _, val := range values {
			pw.OldWriter.AddHeader(k, val)
		}
	}

	// Let DefaultWriter.Write() handle setting SentStatus and writing headers
	// if !pw.Info.SentStatus {
	// 	pw.Info.SentStatus = true
	// 	if pw.Info.StatusCode == 0 {
	// 		pw.Info.StatusCode = http.StatusOK
	// 	}
	// 	pw.Info.OriginalResponse.WriteHeader(pw.Info.StatusCode)
	// }

	data := pw.Buffer.Bytes()

	html := GenerateUI(pw.Info, data)

	pw.OldWriter.Write(html)
	pw.OldWriter.Done()
}

func BuildURL(info *RequestInfo, path string) string {
	if info.ProxyHost == "" {
		if path != "" && path[0] != '/' {
			path = "/" + path
		}
		return info.BaseURL + path + "?ui"
	}
	return fmt.Sprintf("/proxy?host=%s&path=%s",
		info.ProxyHost, url.QueryEscape(path))
}

func BuildURLNoUI(info *RequestInfo, path string) string {
	if info.ProxyHost == "" {
		if path != "" && path[0] != '/' {
			path = "/" + path
		}
		return info.BaseURL + path
	}
	return fmt.Sprintf("/proxy?host=%s&path=%s",
		info.ProxyHost, path)
}

func GenerateUI(info *RequestInfo, data []byte) []byte {
	log.VPrintf(3, ">Enter: GenerateUI")
	defer log.VPrintf(3, "<Exit: GenerateUI")

	list := ""
	regs, err := GetRegistryNames()
	Must(err)

	sort.Strings(regs)
	regs = append([]string{"Default"}, regs...)
	if info.ProxyHost != "" {
		_, tmp, _ := strings.Cut(info.ProxyHost, "://")
		if tmp == "" {
			tmp = info.ProxyHost
		}
		if len(tmp) > 17 {
			tmp = tmp[:17]
		}
		regs = append(regs, tmp)
	}
	regs = append(regs, "xRegistry.io")
	regs = append(regs, "Proxy ...")

	selectedRegistry := ""
	for _, name := range regs {
		checked := ""
		if info.ProxyHost != "" && strings.Contains(info.ProxyHost, name) {
			selectedRegistry = name
			checked = " selected"
		} else if name == "Default" && !strings.Contains(info.BaseURL, "/reg-") && info.ProxyHost == "" {
			selectedRegistry = name
			checked = " selected"
		} else if strings.Contains(info.BaseURL, "/reg-"+name) {
			selectedRegistry = name
			checked = " selected"
		}
		list += fmt.Sprintf("\n      <option%s>%s</option>", checked, name)
	}
	list += "\n"

	roots := ""
	options := ""
	filters := ""
	sortKey := ""
	inlines := ""
	apply := ""

	rootList := []struct {
		u    string
		name string
	}{
		{"", "Registry Root"},
		{"capabilities", "Capabilities"},
		{"model", "Model"},
		{"export", "Export"},
	}

	for _, r := range rootList {
		name := r.name

		if r.u != "" && !info.IsAvailable(r.u) {
			continue
		}

		if info.RootPath == r.u {
			name = "<b>" + name + "</b>"
		}

		roots += fmt.Sprintf("  <li class=myli><a href=\"%s\">%s</a>",
			BuildURL(info, r.u), name)

		if r.u == "capabilities" {
			roots += "&nbsp;&nbsp;("
			if info.RootPath == "capabilitiesoffered" {
				roots += "<b>"
			}
			roots += fmt.Sprintf("<a href=\"%s\">offered</a>",
				BuildURL(info, r.u+"offered"))
			if info.RootPath == "capabilitiesoffered" {
				roots += "</b>"
			}
			roots += ")"
		}

		if r.u == "model" {
			roots += "&nbsp;&nbsp;("
			if info.RootPath == "modelsource" {
				roots += "<b>"
			}
			roots += fmt.Sprintf("<a href=\"%s\">source</a>",
				BuildURL(info, r.u+"source"))
			if info.RootPath == "modelsource" {
				roots += "</b>"
			}
			roots += ")"
		}

		roots += "</li>\n"
	}

	if info.RootPath == "" {
		if info.FlagEnabled("binary") {
			checked := ""
			if info.HasFlag("binary") {
				checked = " checked"
			}
			options +=
				"    <div>\n" +
					"      <input id=binary type='checkbox'" + checked + "/>binary\n" +
					"    </div>\n"

		}

		if info.FlagEnabled("collections") {
			checked := ""
			if info.HasFlag("collections") {
				checked = " checked"
			}
			options +=
				"    <div>\n" +
					"      <input id=collections type='checkbox'" + checked + "/>collections\n" +
					"    </div>\n"

		}

		if info.FlagEnabled("doc") {
			checked := ""
			if info.DoDocView() {
				checked = " checked"
			}
			options +=
				"    <div>\n" +
					"      <input id=docview type='checkbox'" + checked + "/>doc view\n" +
					"    </div>\n"
		}

		if options != "" { // Wrapper if any
			options = "<b>Options:</b>\n<div class=options>\n" +
				options +
				"</div>\n    <hr style=\"width: 95%%\">\n"
		}
	}

	if info.FlagEnabled("sort") && (info.What == "Coll") {
		checked := ""
		val := info.GetFlag("sort")
		val, desc, _ := strings.Cut(val, "=")
		if desc == "desc" {
			checked = " checked"
		}
		sortKey = "<div class=sortsection>\n" +
			"  <b>Sort:</b>\n" +
			"  <input type=text id=sortkey value='" + val + "'>\n" +
			"  <div class=sortsectioncheckbox>\n" +
			"    <input id=sortdesc type='checkbox'" + checked + "/>desc\n" +
			"  </div>\n" +
			"</div>\n"
	}

	if info.FlagEnabled("filter") && (info.RootPath == "" || info.RootPath == "export") {
		prefix := MustPropPathFromPath(info.Abstract).UI()
		if prefix != "" {
			prefix += string(UX_IN)
		}
		for _, arrayF := range info.Filters {
			if filters != "" {
				filters += "\n"
			}
			subF := ""
			for _, FE := range arrayF {
				if subF != "" {
					subF += ","
				}
				next := MustPropPathFromDB(FE.Path).UI()
				next, _ = strings.CutPrefix(next, prefix)
				subF += next
				if FE.Operator == FILTER_EQUAL {
					subF += "=" + FE.Value
				}
			}
			filters += subF
		}
		filters = "<b>Filters:</b> <div class=filterHelp>(each line=filterExpr)</div>\n" +
			"    <textarea id=filters>" +
			filters + "</textarea>\n"
	}

	// Process inlines

	// check to see if the currently selected inlines are the default
	// ones based on presence of '/export'
	defaultInlines := false
	if info.RootPath == "export" {
		defaultInlines = (len(info.Inlines) == 3 &&
			info.IsInlineSet(NewPPP("capabilities").DB()) &&
			info.IsInlineSet(NewPPP("modelsource").DB()) &&
			info.IsInlineSet(NewPPP("*").DB()))
	}

	if info.FlagEnabled("inline") && (info.RootPath == "" || info.RootPath == "export") {
		inlineCount := 0
		checked := ""

		// ----  Add model,capabilitie only if we're at the root

		if info.GroupType == "" {
			if !defaultInlines &&
				info.IsInlineSet(NewPPP("capabilities").DB()) {

				checked = " checked"
			}

			inlines += fmt.Sprintf(`
    <div class=inlines>
      <input id=inline%d type='checkbox' value='capabilities'`+
				checked+`/>capabilities
    </div>`, inlineCount)
			inlineCount++
			checked = ""

			// ----

			if !defaultInlines && info.IsInlineSet(NewPPP("model").DB()) {
				checked = " checked"
			}
			inlines += fmt.Sprintf(`
    <div class=inlines>
      <input id=inline%d type='checkbox' value='model'`+checked+`/>model
    </div>`, inlineCount)
			inlineCount++
			checked = ""

			// ----

			if !defaultInlines && info.IsInlineSet(NewPPP("modelsource").DB()) {
				checked = " checked"
			}
			inlines += fmt.Sprintf(`
    <div class=inlines>
      <input id=inline%d type='checkbox' value='modelsource'`+checked+`/>modelsource
    </div>`, inlineCount)
			inlineCount++
			checked = ""
		}

		if inlines != "" {
			inlines += "\n    <div class=line></div>"
		}

		// ----  * (all)
		if len(info.Parts) != 5 || info.Parts[4] != "meta" {
			if !defaultInlines && info.IsInlineSet(NewPPP("*").DB()) {
				checked = " checked"
			}
			inlines += fmt.Sprintf(`
    <div class=inlines>
      <input id=inline%d type='checkbox' value='*'`+checked+`/>* (all)
    </div>`, inlineCount)
			inlineCount++
			checked = ""
		}

		// ---- Now add based on the model and depth

		inlineOptions := []string{}
		if info.RootPath == "" || info.RootPath == "export" {
			if info.GroupType == "" {
				inlineOptions = GetRegistryModelInlines(info.Registry.Model)
			} else if len(info.Parts) <= 2 {
				inlineOptions = GetGroupModelInlines(info.GroupModel)
			} else if len(info.Parts) <= 4 {
				inlineOptions = GetResourceModelInlines(info.ResourceModel)
			} else {
				if info.Parts[4] != "meta" {
					inlineOptions = GetVersionModelInlines(info.ResourceModel)
				}
			}
		}

		pp, _ := PropPathFromPath(info.Abstract)
		for i, inline := range inlineOptions {
			hilight := ""
			if i%2 == 0 {
				hilight = " inlinehilight"
			}
			checked = ""
			pInline := MustPropPathFromUI(inline)
			fullPP := pp.Append(pInline)
			if info.IsInlineSet(fullPP.DB()) {
				checked = " checked"
			}

			mini := ""
			if (fullPP.Len() != 3 || fullPP.Parts[2].Text == "versions") &&
				fullPP.Len() != 4 {
				minichecked := ""
				if info.IsInlineSet(fullPP.Append(NewPPP("*")).DB()) {
					minichecked = " checked"
				}
				mini = fmt.Sprintf(`
      <div class=minicheckdiv>
        <input id=inline%d type='checkbox' class=minicheck value='%s'%s/>.*
      </div>`, inlineCount, inline+".*", minichecked)
				inlineCount++
			}

			inlines += fmt.Sprintf(`
    <div class='inlines%s'>
      <input id=inline%d type='checkbox' value='%s'%s/>%s%s
    </div>`, hilight, inlineCount, inline, checked, inline, mini)

			inlineCount++

		}

		// If we have any, wrapper it
		if inlines != "" {
			inlines = "<b>Inlines:</b>\n" + inlines
		}
	}

	tmp := BuildURL(info, "")
	urlPath := fmt.Sprintf(`<a href="%s">%s</a>`, tmp, info.BaseURL)
	for i, p := range info.Parts {
		tmp := BuildURL(info, strings.Join(info.Parts[:i+1], "/"))
		urlPath += fmt.Sprintf(`/<a href="%s">%s</a>`, tmp, p)
	}

	detailsSwitch := "false"
	detailsButton := ""
	if info.RootPath == "" || info.RootPath == "export" {
		detailsText := ""
		if info.ShowDetails {
			detailsSwitch = "true"
			detailsText = "Show document"

			tmp := BuildURL(info, strings.Join(info.Parts, "/")+"$details")
			urlPath += fmt.Sprintf(`<a href="%s">$details</a>`, tmp)
		} else {
			detailsSwitch = "false"
			detailsText = "Show details"
		}

		if info.ResourceUID != "" && info.What == "Entity" &&
			(len(info.Parts) != 5 || info.Parts[4] != "meta") {
			if info.ResourceModel.GetHasDocument() {
				detailsButton = fmt.Sprintf(`<center>
      <button id=details onclick='detailsSwitch=!detailsSwitch ; apply()'>%s</button>
    </center>
`, detailsText)
			} else {
				detailsButton = fmt.Sprintf("<br><center>No documents defined<br>for %q</center>", info.ResourceType)
			}
		}
	}

	applyBtn := ""
	if options != "" || filters != "" || sortKey != "" || inlines != "" {
		applyBtn = `<fieldset>
    <legend align=center>
      <button id=applyBtn onclick='apply()'>Apply</button>
    </legend>
    ` + options + `
    ` + sortKey + `
    ` + filters + `
    ` + inlines + `
    ` + apply + `
    </fieldset>`
	}

	addUI := "ui"
	AMP := "&"
	if info.ProxyHost != "" {
		addUI = ""
		AMP = url.QueryEscape(AMP)
	}

	output, expands := RegHTMLify(data, ".", info.ProxyHost)

	autoExpand := ""
	if expands > 3 {
		autoExpand = `
<script>
toggleExp(null, false);
</script>
`
	}

	// Model editor/viewer: shown on modelsource and model pages
	modelEditorEnabled := (info.RootPath == "modelsource" || info.RootPath == "model") &&
		info.IsAvailable(info.RootPath)
	modelMutable := info.IsAvailableMutable("modelsource")
	modelPutURL := BuildURLNoUI(info, "modelsource")
	rawModelJSON := "null"
	if modelEditorEnabled {
		if b, e := json.Marshal(string(data)); e == nil {
			rawModelJSON = string(b)
		}
	}

	editorCSS := ""
	editorJS := ""
	editBtn := ""
	editorDiv := ""
	if modelEditorEnabled {
		editorCSS = `
  /* ---- Model Editor/Viewer ---- */
  .viewToggle {
    display: inline-flex ;
    border: 1px solid #aaa ;
    border-radius: 6px ;
    overflow: hidden ;
    margin-right: 6px ;
    vertical-align: middle ;
  }
  .viewToggleBtn {
    cursor: pointer ;
    padding: 1px 9px ;
    font-size: 14px ;
    font-family: sans-serif ;
    font-weight: bold ;
    background: #e0e0e0 ;
    color: #555 ;
    user-select: none ;
  }
  .viewToggleBtn:hover { background: #d0d0d0 ; }
  .viewToggleBtnActive { background: #407d16 ; color: white ; }
  .viewToggleBtnActive:hover { background: #407d16 ; color: white ; }
  .viewTogglePencil {
    cursor: pointer ;
    padding: 1px 8px ;
    font-size: 15px ;
    font-family: sans-serif ;
    border: 1px solid #aaa ;
    border-radius: 6px ;
    background: #e0e0e0 ;
    color: #555 ;
    user-select: none ;
    vertical-align: middle ;
    margin-right: 6px ;
    display: inline-block ;
    transform: scaleX(-1) ;
  }
  .viewTogglePencil:hover { background: #d0d0d0 ; }
  .viewTogglePencilActive { background: #407d16 ; color: white ; border-color: #407d16 ; }
  .viewTogglePencilActive:hover { background: #407d16 ; color: white ; }
  #modelEditor {
    display: none ;
    flex-direction: column ;
    background: ghostwhite ;
    font-family: sans-serif ;
    font-size: 13px ;
    overflow: hidden ;
    flex: 1 ;
    height: 100% ;
  }
  .editorActionBar {
    display: flex ;
    align-items: center ;
    gap: 8px ;
    padding: 4px 10px ;
    border-bottom: 2px solid #ccc ;
    background: white ;
    flex-shrink: 0 ;
  }
  .editorTabBar {
    display: flex ;
    background: #f0f0f0 ;
    border-bottom: 1px solid #ccc ;
    flex-shrink: 0 ;
  }
  .editorTab {
    padding: 6px 20px ;
    cursor: pointer ;
    font-weight: bold ;
    font-size: 13px ;
    border-right: 1px solid #ccc ;
    color: #555 ;
    user-select: none ;
  }
  .editorTab:hover { background: #e0e0e0 ; }
  .editorTabActive { background: white ; color: #2060a0 ; border-bottom: 2px solid #2060a0 ; margin-bottom: -1px ; }
  .editorBreadcrumb {
    padding: 5px 10px ;
    font-size: 13px ;
    color: #555 ;
    background: #f8f8f8 ;
    border-bottom: 2px solid #bbb ;
    flex-shrink: 0 ;
    display: flex ;
    align-items: center ;
    flex-wrap: wrap ;
    gap: 4px ;
  }
  .navToggleBtn {
    display: none ;
    background: none ;
    border: none ;
    font-size: 20px ;
    cursor: pointer ;
    padding: 0 4px ;
    color: #333 ;
    flex-shrink: 0 ;
    align-self: center ;
    line-height: 1 ;
  }
  .bcLink { cursor: pointer ; color: #2060a0 ; }
  .bcLink:hover { text-decoration: underline ; }
  .bcSep { color: #555 ; margin: 0 6px ; font-size: 18px ; line-height: 1 ; vertical-align: middle ; }
  .bcCurrent { color: #333 ; font-weight: bold ; }
  .editorBody {
    display: flex ;
    flex: 1 ;
    overflow: hidden ;
    position: relative ;
  }
  .editorLeftNav {
    width: 200px ;
    min-width: 160px ;
    overflow-y: auto ;
    border-right: 1px solid #ccc ;
    background: #fafafa ;
    display: flex ;
    flex-direction: column ;
    flex-shrink: 0 ;
  }
  .navItem {
    padding: 7px 12px ;
    cursor: pointer ;
    font-size: 13px ;
    color: #333 ;
    border-bottom: 1px solid #f0f0f0 ;
    user-select: none ;
    display: flex ;
    align-items: center ;
  }
  .navItemSelected { background: #dde8f8 ; font-weight: bold ; color: #1a3a6a ; }
  .navItemArrow { margin-left: auto ; color: #555 ; font-size: 18px ; font-weight: bold ; }
  .navItemDel { margin-left: 6px ; color: #a00 ; font-size: 11px ; cursor: pointer ; padding: 0 4px ; border-radius: 3px ; flex-shrink: 0 ; }
  .navItemAdd {
    padding: 7px 12px ;
    color: #407d16 ;
    cursor: pointer ;
    font-size: 12px ;
    font-weight: bold ;
    border-bottom: 1px solid #e8e8e8 ;
    user-select: none ;
  }
  @media (hover: hover) {
    .navItem:hover { background: #e8eef8 ; }
    .navItemSelected:hover { background: #dde8f8 ; }
    .navItemDel:hover { background: #fee ; }
    .navItemAdd:hover { background: #e8f8e0 ; }
  }
  .editorRightPanel {
    flex: 1 ;
    overflow-y: auto ;
    overflow-x: hidden ;
    padding: 16px 20px ;
  }
  .editorHint {
    color: #bbb ;
    font-style: italic ;
    text-align: center ;
    margin-top: 40px ;
    font-size: 15px ;
  }
  .editorFormTitle {
    font-size: 14px ;
    font-weight: bold ;
    color: #333 ;
    margin-bottom: 14px ;
    padding-bottom: 6px ;
    border-bottom: 1px solid #ddd ;
  }
  .editorField {
    display: flex ;
    align-items: center ;
    flex-wrap: wrap ;
    gap: 10px ;
    margin-bottom: 8px ;
  }
  .editorField label {
    width: 160px ;
    font-size: 12px ;
    font-weight: bold ;
    color: #444 ;
    text-align: left ;
    flex-shrink: 0 ;
  }
  .editorInput {
    flex: 1 ;
    min-width: 120px ;
    border: 1px solid #ccc ;
    border-radius: 4px ;
    padding: 3px 6px ;
    font-size: 12px ;
    box-sizing: border-box ;
    height: 26px ;
  }
  .editorInput[type="number"] { height: 26px ; }
  select.editorInput {
    height: 26px ;
    padding: 3px 22px 3px 6px ;
    margin: 0 ;
    background-color: white ;
    color: #333 ;
    font-weight: normal ;
    -webkit-appearance: none ;
    -moz-appearance: none ;
    appearance: none ;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6'%3E%3Cpath d='M0 0l5 6 5-6z' fill='%23222'/%3E%3C/svg%3E") ;
    background-repeat: no-repeat ;
    background-position: right 6px center ;
    background-size: 10px 6px ;
    cursor: pointer ;
    width: 100% ;
  }
  .editorSelectWrap { flex: 1 ; min-width: 0 ; }
  textarea.editorInput { height: auto ; }
  .editorInput:focus { outline: 2px solid #7ab0e0 ; border-color: #7ab0e0 ; }
  .editorCheckRow {
    display: flex ;
    align-items: center ;
    gap: 6px ;
    margin-bottom: 6px ;
    margin-left: 170px ;
  }
  .editorCheckRow label { font-size: 12px ; color: #555 ; }
  .editorSectionLabel {
    font-size: 12px ;
    font-weight: bold ;
    color: #444 ;
    margin: 14px 0 6px ;
    padding-bottom: 4px ;
    border-bottom: 1px solid #e0e0e0 ;
  }
  .boolGrid {
    display: grid ;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)) ;
    width: 100% ;
    gap: 6px 10px ;
    margin-bottom: 8px ;
  }
  .boolCell {
    display: flex ;
    flex-direction: row ;
    align-items: center ;
    gap: 6px ;
  }
  .boolCell label {
    font-size: 11px ;
    font-weight: bold ;
    color: #444 ;
    width: 65px ;
    flex-shrink: 0 ;
    white-space: normal ;
    line-height: 1.3 ;
  }
  .boolSeg {
    display: inline-flex ;
    border: 1px solid #ccc ;
    border-radius: 8px ;
    overflow: hidden ;
    height: 28px ;
    flex-shrink: 0 ;
  }
  .boolSegBtn {
    border: none ;
    padding: 0 4px ;
    min-width: 34px ;
    font-size: 12px ;
    cursor: pointer ;
    background: #f5f5f5 ;
    color: #000 ;
    line-height: 28px ;
    text-align: center ;
  }
  .boolSegBtn:not(:last-child) { border-right: 1px solid #ccc ; }
  .boolSegBtn.boolSegActive { background: #4a7c24 ; color: #fff ; }
  .boolSegBtn:hover:not(.boolSegActive) { background: #e8e8e8 ; }
  .boolSegReadOnly .boolSegBtn { cursor: default ; pointer-events: none ; }
  .boolSegReadOnly .boolSegBtn:not(.boolSegActive) { color: #bbb ; }
  .editorBtn {
    border-radius: 8px ;
    border: 1px solid #407d16 ;
    background: #407d16 ;
    padding: 3px 12px ;
    color: white ;
    cursor: pointer ;
    font-size: 12px ;
    font-weight: bold ;
  }
  .editorBtn:hover { background: #c4c4c4 ; color: black ; }
  .editorBtn:disabled { opacity: 0.4 ; cursor: not-allowed ; }
  .editorBtn:disabled:hover { background: inherit ; color: white ; }
  #saveBtn { background: #2060a0 ; border-color: #2060a0 ; }
  #saveBtn:hover { background: #c4c4c4 ; color: black ; }
  #undoBtn { background: #a06020 ; border-color: #a06020 ; }
  #undoBtn:hover { background: #c4c4c4 ; color: black ; }
  .rmBtn {
    border-radius: 5px ;
    border: 1px solid #a00 ;
    background: #fff ;
    color: #a00 ;
    cursor: pointer ;
    padding: 2px 8px ;
    font-size: 11px ;
  }
  .rmBtn:hover { background: #fee ; }
  .readOnlyBanner {
    background: #fff3cd ;
    border: 1px solid #ffc107 ;
    border-radius: 6px ;
    padding: 2px 10px ;
    font-size: 12px ;
    font-weight: bold ;
    color: #856404 ;
  }
  #editorError {
    background: #fee ;
    border: 1px solid #c00 ;
    border-radius: 4px ;
    padding: 8px 12px ;
    font-size: 12px ;
    color: #900 ;
    white-space: pre-wrap ;
    margin: 0 10px 6px ;
  }
  .labelsWrap { display: flex ; align-items: flex-start ; gap: 8px ; }
  .labelsRows { display: flex ; flex-direction: column ; gap: 4px ; flex: 1 ; }
  .labelRow { display: flex ; align-items: center ; gap: 6px ; }
  .labelKey { width: 120px ; }
  .labelVal { flex: 1 ; }
  .constraintBlock { border:1px solid #ddd; border-radius:4px; padding:8px; margin-bottom:8px; background:#fafafa; }
  .constraintBlockHdr { display:flex; align-items:center; justify-content:space-between; margin-bottom:6px; }
  .constraintBlockTitle { font-size:12px; font-weight:600; color:#555; }
  .cstrPathRow { display:flex; align-items:center; gap:4px; margin-bottom:4px; }
  .cstrPathRow label { width:160px; flex-shrink:0; font-size:13px; }
  .cstrPathDot { font-weight:bold; padding:0 2px; color:#666; }
  .cstrResSel, .cstrAttrSel { flex:1; }
  .cstrEnumSection { margin-top:6px; }
  .savingOverlay { position:fixed; inset:0; background:rgba(0,0,0,0.45); z-index:9999;
    display:flex; align-items:center; justify-content:center; }
  .savingBox { background:#fff; border-radius:8px; padding:28px 40px; box-shadow:0 4px 24px rgba(0,0,0,0.3);
    display:flex; flex-direction:column; align-items:center; gap:16px; font-size:15px; color:#333; }
  .savingSpinner { width:36px; height:36px; border:4px solid #ddd; border-top-color:#555;
    border-radius:50%; animation:spin 0.8s linear infinite; }
  @keyframes spin { to { transform:rotate(360deg); } }
  @media (max-width: 768px) {
    body { flex-direction: column ; height: 100dvh ; }
    #left { border-right: none ; border-bottom: 2px solid lightgray ; min-width: unset ; width: 100% ; }
    #right { flex: 1 ; min-height: 0 ; overflow: hidden ; }
    .navToggleBtn { display: inline-flex ; align-items: center ; }
    .editorLeftNav {
      display: none ;
      position: fixed ;
      top: 0 ; left: 0 ;
      width: 80% ; max-width: 280px ;
      overflow-y: auto ;
      padding-bottom: env(safe-area-inset-bottom, 0px) ;
      z-index: 100 ;
      box-shadow: 4px 0 16px rgba(0,0,0,0.25) ;
      border-right: 1px solid #ccc ;
    }
    .editorLeftNav.navOpen { display: flex ; }
    .editorActionBar { position: sticky ; top: 0 ; z-index: 10 ; }
    .editorField { flex-direction: column ; align-items: stretch ; gap: 4px ; }
    .editorField label { width: auto ; }
    .editorInput { font-size: 16px ; }
    .editorBtn { min-height: 36px ; }
    .boolSeg { height: 18px ; }
    .boolSegBtn { line-height: 18px ; }
  }
`

		editorJS = `
// ---- Model Editor ----

var _modelPutURL = ` + "`" + modelPutURL + "`" + ` ;
var _modelMutable = ` + fmt.Sprintf("%v", modelMutable) + ` ;
var _modelReadOnly = true ; // runtime: true in json/view modes, false in edit mode
var _modelSrc = null ;
var _modelData = null ;
var _modelDirty = false ;
var _navTab = 'registry' ;
var _navPath = [] ;
var _navSelected = null ;
var _attrNestStack = [] ; // [{key,isItem}] — nested attr drilldown beyond _navPath
var _cstrCounter = 0 ; // unique ID counter for constraint enum containers

function initModelEditor() {
  _modelSrc = JSON.parse(` + rawModelJSON + `) ;
  _modelData = deepClone(_modelSrc) ;
}
initModelEditor() ;

function deepClone(o) { return JSON.parse(JSON.stringify(o)) ; }
function markDirty() {
  if (!_modelDirty) {
    _modelDirty = true ;
    var sb = document.getElementById('saveBtn') ; if (sb) sb.disabled = false ;
    var ub = document.getElementById('undoBtn') ; if (ub) ub.disabled = false ;
  }
}

window.addEventListener('beforeunload', function(e) {
  if (_modelDirty) { e.preventDefault() ; e.returnValue = '' ; }
}) ;

function showLeaveEditDialog(onSave, onDiscard) {
  var overlay = document.createElement('div') ;
  overlay.style.cssText = 'position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.35);z-index:9999;display:flex;align-items:center;justify-content:center;' ;
  var box = document.createElement('div') ;
  box.style.cssText = 'background:white;border-radius:8px;padding:24px;box-shadow:0 4px 24px rgba(0,0,0,0.25);max-width:340px;width:90%;font-family:sans-serif;' ;
  var msg = document.createElement('p') ; msg.textContent = 'You have unsaved changes.' ;
  msg.style.cssText = 'margin:0 0 20px;font-size:14px;color:#333;' ;
  box.appendChild(msg) ;
  var btns = document.createElement('div') ; btns.style.cssText = 'display:flex;gap:8px;justify-content:flex-end;' ;
  function mkBtn(label, fn, css) {
    var b = document.createElement('button') ; b.textContent = label ;
    b.style.cssText = 'padding:6px 16px;border-radius:5px;cursor:pointer;font-size:13px;font-weight:bold;' + css ;
    b.onclick = function() { document.body.removeChild(overlay) ; fn() ; } ;
    btns.appendChild(b) ;
  }
  mkBtn('Cancel',  function(){},  'background:#f0f0f0;color:#333;border:1px solid #ccc;') ;
  mkBtn('Discard', onDiscard,     'background:#f8d7da;color:#721c24;border:1px solid #f5c6cb;') ;
  mkBtn('Save',    onSave,        'background:#2060a0;color:white;border:1px solid #2060a0;') ;
  box.appendChild(btns) ; overlay.appendChild(box) ; document.body.appendChild(overlay) ;
}

function doSwitchView(mode) {
  var textDiv = document.getElementById('text') ;
  var edDiv   = document.getElementById('modelEditor') ;
  var expAll  = document.getElementById('expAll') ;
  var leftNav = document.getElementById('left') ;
  var myOut   = document.getElementById('myOutput') ;
  var jBtn = document.getElementById('viewToggleJson') ;
  var fBtn = document.getElementById('viewToggleForm') ;
  var eBtn = document.getElementById('viewToggleEdit') ;

  if (mode === 'json') {
    _modelReadOnly = true ;
    if (textDiv) textDiv.style.display = '' ;
    if (edDiv)   edDiv.style.display   = 'none' ;
    if (expAll)  expAll.style.display  = '' ;
    if (leftNav) leftNav.style.display = '' ;
    if (myOut) { myOut.style.display = '' ; myOut.style.flexDirection = '' ;
                 myOut.style.padding = '' ; myOut.style.overflow = '' ; }
    // Move expandAll div back to its fixed position in myOutput
    var exAll = document.getElementById('expandAll') ;
    if (exAll && myOut) { exAll.style.position = '' ; exAll.style.marginLeft = '' ; myOut.insertBefore(exAll, myOut.firstChild) ; }
    if (jBtn) jBtn.className = 'viewToggleBtn viewToggleBtnActive' ;
    if (fBtn) fBtn.className = 'viewToggleBtn' ;
    if (eBtn) eBtn.style.display = 'none' ;
  } else {
    var wasInEditor = edDiv && edDiv.style.display !== 'none' ;
    _modelReadOnly = (mode === 'view') ;
    if (!wasInEditor) {
      _navTab = 'registry' ; _navPath = [] ; _navSelected = null ; _attrNestStack = [] ;
    }
    if (textDiv) textDiv.style.display = 'none' ;
    if (edDiv)   edDiv.style.display   = 'flex' ;
    if (expAll)  expAll.style.display  = 'none' ;
    if (leftNav) leftNav.style.display = 'none' ;
    if (myOut) { myOut.style.display = 'flex' ; myOut.style.flexDirection = 'column' ;
                 myOut.style.padding = '0' ; myOut.style.overflow = 'hidden' ; }
    if (jBtn) jBtn.className = 'viewToggleBtn' ;
    if (fBtn) fBtn.className = 'viewToggleBtn viewToggleBtnActive' ;
    if (eBtn && _modelMutable) {
      eBtn.style.display = '' ;
      eBtn.className = 'viewTogglePencil' + (mode === 'edit' ? ' viewTogglePencilActive' : '') ;
    }
    renderEditor() ;
  }
}

function switchView(mode) {
  if (mode === 'toggle-edit') {
    mode = _modelReadOnly ? 'edit' : 'view' ;
  }
  // Leaving edit mode with unsaved changes — offer Save / Discard / Cancel
  if (!_modelReadOnly && _modelDirty && mode !== 'edit') {
    showLeaveEditDialog(
      function() { saveModel(function() { doSwitchView(mode) ; }) ; },
      function() { _modelDirty = false ; _modelData = deepClone(_modelSrc) ; doSwitchView(mode) ; }
    ) ;
    return ;
  }
  doSwitchView(mode) ;
}

// ---- Navigation primitives ----

function drillDown(path) {
  var beforePath = _navPath.slice() ;
  collectCurrentEditor() ;
  _attrNestStack = [] ;
  // Fix up stale path segments in case collectCurrentEditor renamed a group/resource key
  _navPath = path.map(function(seg, i) {
    if (i < beforePath.length && beforePath[i] === seg && _navPath[i] && _navPath[i] !== seg)
      return _navPath[i] ;
    return seg ;
  }) ;
  _navSelected = null ;
  renderEditor() ;
}

function selectItem(key) {
  collectCurrentEditor() ;
  _navSelected = key ;
  renderEditor() ;
}

function changeTab(tab) {
  collectCurrentEditor() ;
  _attrNestStack = [] ;
  _navTab = tab ; _navPath = [] ; _navSelected = null ;
  renderEditor() ;
}

// ---- Attr nesting helpers ----

// Returns the base attributes map (or item parent) at the current _navPath level.
function getBaseAttrsObj() {
  var m = _modelData || {} ;
  if (_navTab === 'registry') {
    if (!m.attributes) m.attributes = {} ;
    return m.attributes ;
  }
  var gk = _navPath[0] ; if (!gk) return {} ;
  if (!m.groups) m.groups = {} ;
  var grp = m.groups[gk] ; if (!grp) return {} ;
  if (_navPath.length === 2) {
    var sec = _navPath[1] ;
    if (sec === 'attributes') { if (!grp.attributes) grp.attributes = {} ; return grp.attributes ; }
  }
  if (_navPath.length === 4) {
    var rk = _navPath[2], attrSec = _navPath[3] ;
    var dataKey = attrSec === 'versionattributes' ? 'attributes' : attrSec ;
    if (!grp.resources) grp.resources = {} ;
    var res = grp.resources[rk] ; if (!res) return {} ;
    if (!res[dataKey]) res[dataKey] = {} ;
    return res[dataKey] ;
  }
  return {} ;
}

// Traverses _attrNestStack from the base attrs object.
// Returns {attrsObj, parentAttr, isItem, ifvMap} where:
//   isItem:false, ifvMap:null → attrsObj is the attrs map to show/edit
//   isItem:true              → parentAttr is the item object (map/array)
//   ifvMap:non-null          → currently viewing ifvalues key list
// If createMissing=true, creates intermediate structures as needed.
function resolveAttrNesting(createMissing) {
  var cur = getBaseAttrsObj() ;
  var curParent = null ; // last resolved attrObj from isItem:true, for __item__:isItem:true chaining
  var ifvMap = null ;
  for (var i = 0; i < _attrNestStack.length; i++) {
    var entry = _attrNestStack[i] ;
    if (entry.key === '__item__' && !entry.isItem) { continue ; } // sentinel: inside item.attributes
    if (entry.key === '__item__' && entry.isItem) {
      // Descend into curParent.item.item (map/array item chain)
      if (!curParent) return {attrsObj:{}, parentAttr:null, isItem:true, ifvMap:null} ;
      var prevItem = curParent.item ;
      if (!prevItem) {
        if (createMissing) { curParent.item = {} ; prevItem = curParent.item ; }
        else return {attrsObj:{}, parentAttr:null, isItem:true, ifvMap:null} ;
      }
      curParent = prevItem ; // curParent.item is now the next item to render
      if (i === _attrNestStack.length - 1) return {attrsObj:{}, parentAttr:curParent, isItem:true, ifvMap:null} ;
      continue ;
    }
    if (entry.isIfValues) {
      var attrObj = cur[entry.key] ;
      if (!attrObj) {
        if (createMissing) { cur[entry.key] = {} ; attrObj = cur[entry.key] ; }
        else return {attrsObj:{}, parentAttr:null, isItem:false, ifvMap:null} ;
      }
      if (!attrObj.ifvalues) {
        if (createMissing) attrObj.ifvalues = {} ; else attrObj.ifvalues = {} ;
      }
      ifvMap = attrObj.ifvalues ;
      if (i === _attrNestStack.length - 1) return {attrsObj:{}, parentAttr:null, isItem:false, ifvMap:ifvMap} ;
    } else if (entry.isSiblings) {
      if (!ifvMap) return {attrsObj:{}, parentAttr:null, isItem:false, ifvMap:null} ;
      var ifval = ifvMap[entry.key] ;
      if (!ifval) {
        if (createMissing) { ifvMap[entry.key] = {} ; ifval = ifvMap[entry.key] ; }
        else return {attrsObj:{}, parentAttr:null, isItem:false, ifvMap:null} ;
      }
      if (!ifval.siblingattributes) {
        if (createMissing) ifval.siblingattributes = {} ;
        else return {attrsObj:{}, parentAttr:null, isItem:false, ifvMap:null} ;
      }
      cur = ifval.siblingattributes ; ifvMap = null ;
    } else if (entry.isItem) {
      var attrObj = cur[entry.key] ;
      if (!attrObj) {
        if (createMissing) { cur[entry.key] = {} ; attrObj = cur[entry.key] ; }
        else return {attrsObj:{}, parentAttr:{}, isItem:false, ifvMap:null} ;
      }
      if (!attrObj.item) {
        if (createMissing) attrObj.item = {} ;
        else return {attrsObj:{}, parentAttr:{}, isItem:true, ifvMap:null} ;
      }
      curParent = attrObj ; // save for potential __item__:isItem:true chaining
      if (i === _attrNestStack.length - 1) return {attrsObj:{}, parentAttr:attrObj, isItem:true, ifvMap:null} ;
      // Look ahead: if next is __item__:isItem:false (object sub-attrs sentinel), advance cur
      var nextEntry = _attrNestStack[i+1] ;
      if (nextEntry && nextEntry.key === '__item__' && !nextEntry.isItem) {
        var itm = attrObj.item ;
        if (!itm.attributes) { if (createMissing) itm.attributes = {} ; else return {attrsObj:{},parentAttr:{},isItem:false,ifvMap:null} ; }
        cur = itm.attributes ;
      }
      // If next is __item__:isItem:true, curParent is set and will be handled above
    } else {
      var attrObj = cur[entry.key] ;
      if (!attrObj) {
        if (createMissing) { cur[entry.key] = {} ; attrObj = cur[entry.key] ; }
        else return {attrsObj:{}, parentAttr:{}, isItem:false, ifvMap:null} ;
      }
      if (!attrObj.attributes) {
        if (createMissing) attrObj.attributes = {} ;
        else return {attrsObj:{}, parentAttr:{}, isItem:false, ifvMap:null} ;
      }
      cur = attrObj.attributes ;
    }
  }
  return {attrsObj:cur, parentAttr:null, isItem:false, ifvMap:ifvMap} ;
}

// Drills into a nested attribute level.
function drillIntoAttr(attrKey, isItem) {
  collectCurrentEditor() ;
  // _navSelected may have been updated by collectCurrentEditor (rename) — use it
  var resolvedKey = _navSelected || attrKey ;
  var attrType = null ;
  if (isItem) {
    var ctx0 = resolveAttrNesting(false) ;
    attrType = ctx0.attrsObj && ctx0.attrsObj[resolvedKey] ? (ctx0.attrsObj[resolvedKey].type || 'map') : 'map' ;
  }
  _attrNestStack.push({key:resolvedKey, isItem:isItem, attrType:attrType}) ;
  _navSelected = isItem ? '__item__' : null ;
  renderEditor() ;
}

// Pops _attrNestStack back to depth d (0 = fully exit nesting).
function popAttrNestTo(d) {
  collectCurrentEditor() ;
  _attrNestStack = _attrNestStack.slice(0, d) ;
  _navSelected = null ;
  renderEditor() ;
}

// ---- If Values helpers ----

function addNewIfValue() {
  collectCurrentEditor() ;
  var ctx = resolveAttrNesting(true) ;
  var ifv = ctx.ifvMap ; if (!ifv) return ;
  var k = uniqueKey(ifv, 'value') ;
  ifv[k] = {siblingattributes:{}} ;
  markDirty() ; _navSelected = k ; renderEditor() ;
}

function deleteIfValue(k) {
  var ctx = resolveAttrNesting(false) ;
  if (ctx.ifvMap) delete ctx.ifvMap[k] ;
  markDirty() ; if (_navSelected === k) _navSelected = null ; renderEditor() ;
}

function drillIntoIfValueSiblings() {
  collectCurrentEditor() ;
  var resolvedKey = _navSelected ;
  _attrNestStack.push({key:resolvedKey, isSiblings:true}) ;
  _navSelected = null ;
  renderEditor() ;
}

function renderIfValueForm(div, valueKey, ifvMap) {
  var titleEl = document.createElement('div') ; titleEl.className = 'editorFormTitle' ;
  titleEl.textContent = 'If Value: ' + valueKey ; div.appendChild(titleEl) ;
  var origInp = document.createElement('input') ; origInp.type = 'hidden' ;
  origInp.id = 'ef_ifvalue_orig' ; origInp.value = valueKey ; div.appendChild(origInp) ;
  var keyRow = ef('ef_ifvalue_key', 'Value', valueKey, true) ;
  var keyInp = keyRow.querySelector('input') ;
  keyInp.oninput = function() {
    var v = keyInp.value.trim() || '\u2026' ;
    titleEl.textContent = 'If Value: ' + v ;
    var navEl = document.querySelector('.navItemSelected') ;
    if (navEl) { var sp = navEl.firstChild ; if (sp) sp.textContent = v ; }
  } ;
  div.appendChild(keyRow) ;
  var sibCount = Object.keys(((ifvMap[valueKey]||{}).siblingattributes)||{}).length ;
  var drilledBtnRow = document.createElement('div') ; drilledBtnRow.className = 'editorField' ;
  drilledBtnRow.style.marginTop = '8px' ;
  var spacer = document.createElement('label') ; spacer.style.visibility = 'hidden' ;
  drilledBtnRow.appendChild(spacer) ;
  var drilledBtn = document.createElement('button') ; drilledBtn.className = 'editorBtn navDrillBtn' ;
  drilledBtn.style.cssText = 'font-size:11px;padding:3px 8px;' ;
  drilledBtn.textContent = '\u25b6 Edit Sibling Attributes' + (sibCount ? ' ('+sibCount+')' : '') ;
  drilledBtn.onclick = drillIntoIfValueSiblings ;
  drilledBtnRow.appendChild(drilledBtn) ; div.appendChild(drilledBtnRow) ;
}

function saveIfValueFrom(ifvMap, origKey) {
  if (!ifvMap) return ;
  var keyEl = document.getElementById('ef_ifvalue_key') ; if (!keyEl) return ;
  var newKey = keyEl.value.trim() || origKey ;
  var existing = ifvMap[origKey] || {siblingattributes:{}} ;
  if (newKey !== origKey) delete ifvMap[origKey] ;
  ifvMap[newKey] = existing ;
  if (_navSelected === origKey) _navSelected = newKey ;
}

// ---- Main render ----

function renderEditor() {
  var div = document.getElementById('modelEditor') ;
  // Rescue #expandAll from the old breadcrumb before wiping innerHTML
  var exAll = document.getElementById('expandAll') ;
  var myOut = document.getElementById('myOutput') ;
  if (exAll && div.contains(exAll) && myOut) {
    exAll.style.position = '' ; exAll.style.marginLeft = '' ;
    myOut.insertBefore(exAll, myOut.firstChild) ;
  }
  div.innerHTML = '' ;

  // Action bar
  var bar = document.createElement('div') ;
  bar.className = 'editorActionBar' ;
  if (!_modelReadOnly) {
    var sb = document.createElement('button') ;
    sb.className = 'editorBtn' ; sb.id = 'saveBtn' ;
    sb.textContent = 'Save' ; sb.onclick = function() { saveModel() ; } ; sb.disabled = !_modelDirty ;
    bar.appendChild(sb) ;
    var ub = document.createElement('button') ;
    ub.className = 'editorBtn' ; ub.id = 'undoBtn' ;
    ub.textContent = 'Undo' ; ub.onclick = undoModel ; ub.disabled = !_modelDirty ;
    bar.appendChild(ub) ;
  } else {
    // No buttons — collapse the bar completely
    bar.style.cssText = 'padding:0;border:none;margin:0;height:0;' ;
  }
  div.appendChild(bar) ;

  if (!_modelReadOnly) {
    var errDiv = document.createElement('div') ;
    errDiv.id = 'editorError' ; errDiv.style.display = 'none' ;
    div.appendChild(errDiv) ;
  }

  // Auto-select 'fields' when entering registry root or group/resource level with nothing selected
  if (_navSelected === null) {
    if (_navTab === 'registry' && _navPath.length === 0) _navSelected = 'fields' ;
    else if (_navTab === 'groups' && (_navPath.length === 1 || _navPath.length === 3)) _navSelected = 'fields' ;
  }

  // Breadcrumb (replaces tab bar)
  var bc = buildBreadcrumb() ;
  // Mobile nav toggle button — insert before breadcrumb content
  var toggleBtn = document.createElement('button') ;
  toggleBtn.className = 'navToggleBtn' ; toggleBtn.type = 'button' ;
  toggleBtn.textContent = '\u2630' ; toggleBtn.title = 'Show navigation' ;
  bc.insertBefore(toggleBtn, bc.firstChild) ;
  // Move the view-toggle buttons into the breadcrumb (right-aligned)
  var exAll = document.getElementById('expandAll') ;
  if (exAll) { exAll.style.position = 'static' ; exAll.style.marginLeft = 'auto' ; bc.appendChild(exAll) ; }
  div.appendChild(bc) ;

  // Body: left nav + right panel
  var body = document.createElement('div') ; body.className = 'editorBody' ;
  var lnav = document.createElement('div') ; lnav.className = 'editorLeftNav' ;
  buildLeftNav(lnav) ;
  var rpanel = document.createElement('div') ; rpanel.className = 'editorRightPanel' ;
  buildRightPanel(rpanel) ;
  // Backdrop for nav overlay (mobile only)
  var backdrop = document.createElement('div') ;
  backdrop.style.cssText = 'display:none;position:fixed;inset:0;background:rgba(0,0,0,0.3);z-index:99;' ;
  function openNav() {
    var bc = document.querySelector('.editorBreadcrumb') ;
    var topPx = bc ? (bc.offsetTop + bc.offsetHeight) : 0 ;
    lnav.style.top = topPx + 'px' ;
    lnav.style.maxHeight = 'calc(100dvh - ' + topPx + 'px - env(safe-area-inset-bottom, 0px))' ;
    backdrop.style.top = lnav.style.top ;
    lnav.classList.add('navOpen') ; backdrop.style.display = 'block' ; toggleBtn.textContent = '\u2715' ;
  }
  window._editorOpenNav = openNav ;
  function closeNav() {
    lnav.classList.remove('navOpen') ; backdrop.style.display = 'none' ; toggleBtn.textContent = '\u2630' ;
  }
  toggleBtn.onclick = function() { lnav.classList.contains('navOpen') ? closeNav() : openNav() ; } ;
  backdrop.onclick = closeNav ;
  body.appendChild(backdrop) ; body.appendChild(lnav) ; body.appendChild(rpanel) ;
  div.appendChild(body) ;

  if (_modelReadOnly) applyReadOnly(div) ;
  if (!_modelReadOnly) {
    div.addEventListener('input', markDirty) ;
    div.addEventListener('change', markDirty) ;
  }
}

// ---- Breadcrumb ----

function buildBreadcrumb() {
  var labelMap = {
    'fields':'Details', 'attributes':'Attributes', 'resources':'Resources',
    'versionattributes':'Version Attrs', 'resourceattributes':'Resource Attrs',
    'metaattributes':'Meta Attrs'
  } ;
  var segs = [] ;
  segs.push({label: 'Registry', tab: 'registry', path: []}) ;
  if (_navTab === 'groups') {
    segs.push({label: 'Groups', tab: 'groups', path: []}) ;
    _navPath.forEach(function(seg, i) {
      var label = labelMap[seg] || seg ;
      var id = (!labelMap[seg] && i === 0) ? 'bcGroupKey' : (!labelMap[seg] && i === 2) ? 'bcResourceKey' : null ;
      segs.push({label: label, tab: 'groups', path: _navPath.slice(0, i+1), id: id}) ;
    }) ;
  } else if (_navPath.length > 0) {
    _navPath.forEach(function(seg, i) {
      segs.push({label: labelMap[seg] || seg, tab: 'registry', path: _navPath.slice(0, i+1)}) ;
    }) ;
  }
  var bc = document.createElement('div') ; bc.className = 'editorBreadcrumb' ;
  var allSegs = segs.slice() ; // structural segments

  // Append _attrNestStack segments — each entry generates 2 segments with full nav info
  _attrNestStack.forEach(function(entry, i) {
    if (entry.isIfValues) {
      allSegs.push({label: entry.key, nestDepth: i, backKey: entry.key}) ;
      allSegs.push({label: 'If-Values', nestDepth: i+1, backKey: null}) ;
    } else if (entry.isSiblings) {
      allSegs.push({label: entry.key, nestDepth: i, backKey: entry.key}) ;
      allSegs.push({label: 'Siblings', nestDepth: i+1, backKey: null}) ;
    } else if (entry.isItem && entry.key === '__item__') {
      // Item-chain sentinel: just one segment for the inner item level
      var typeLabel2 = (entry.attrType || 'map') + ' details' ;
      allSegs.push({label: typeLabel2, nestDepth: i+1, backKey: '__item__'}) ;
    } else if (entry.isItem) {
      allSegs.push({label: entry.key, nestDepth: i, backKey: entry.key}) ;
      var typeLabel = (entry.attrType || 'map') + ' details' ;
      allSegs.push({label: typeLabel, nestDepth: i+1, backKey: '__item__'}) ;
    } else if (entry.key === '__item__') {
      allSegs.push({label: 'Item', nestDepth: i, backKey: '__item__'}) ;
      allSegs.push({label: 'Attributes', nestDepth: i+1, backKey: null}) ;
    } else {
      allSegs.push({label: entry.key, nestDepth: i, backKey: entry.key}) ;
      allSegs.push({label: 'Attributes', nestDepth: i+1, backKey: null}) ;
    }
  }) ;

  allSegs.forEach(function(s, i) {
    if (i > 0) { var sep = document.createElement('span') ; sep.className = 'bcSep' ; sep.textContent = '\u203a' ; bc.appendChild(sep) ; }
    if (i === allSegs.length - 1) {
      var cur = document.createElement('span') ; cur.className = 'bcCurrent' ; cur.textContent = s.label ;
      if (s.id) cur.id = s.id ;
      bc.appendChild(cur) ;
    } else {
      var lnk = document.createElement('span') ; lnk.className = 'bcLink' ; lnk.textContent = s.label ;
      if (s.id) lnk.id = s.id ;
      if (s.nestDepth !== undefined) {
        // Nest-stack segment — pop to nestDepth and optionally re-select
        var nd = s.nestDepth, bk = s.backKey ;
        lnk.onclick = function() {
          collectCurrentEditor() ;
          _attrNestStack = _attrNestStack.slice(0, nd) ;
          _navSelected = bk || null ;
          renderEditor() ;
        } ;
      } else {
        var st = s.tab, sp = s.path ;
        lnk.onclick = function() { collectCurrentEditor() ; _attrNestStack = [] ; _navTab = st ; _navPath = sp ; _navSelected = null ; renderEditor() ; } ;
      }
      bc.appendChild(lnk) ;
    }
  }) ;
  return bc ;
}

// ---- Left Nav ----

function buildLeftNav(div) {
  var model = _modelData || {} ;

  function navItem(label, isContainer, isSelected, clickFn, deleteFn) {
    var el = document.createElement('div') ;
    el.className = 'navItem' + (isSelected ? ' navItemSelected' : '') ;
    var lbl = document.createElement('span') ; lbl.style.flex = '1' ;
    if (typeof label === 'string') { lbl.textContent = label ; } else { lbl.appendChild(label) ; }
    el.appendChild(lbl) ;
    if (deleteFn && !_modelReadOnly) {
      var del = document.createElement('span') ; del.className = 'navItemDel' ;
      del.textContent = '\u2715' ; del.title = 'Remove' ;
      del.onclick = function(e) { e.stopPropagation() ; confirmDel('"' + (typeof label === 'string' ? label : el.textContent.trim()) + '"', deleteFn) ; } ;
      el.appendChild(del) ;
    }
    if (isContainer) {
      var arr = document.createElement('span') ; arr.className = 'navItemArrow' ; arr.textContent = '\u203a' ;
      el.appendChild(arr) ;
    }
    el.onclick = clickFn ; return el ;
  }

  function navAdd(label, fn) {
    var el = document.createElement('div') ; el.className = 'navItemAdd' ;
    el.textContent = label ; el.onclick = fn ; return el ;
  }

  function attrLabel(k) {
    if (k !== '*') return k ;
    var el = document.createElement('span') ;
    var star = document.createElement('span') ; star.textContent = '*' ;
    star.style.cssText = 'font-size:16px;font-weight:bold;vertical-align:middle;line-height:1;' ;
    var desc = document.createElement('span') ; desc.textContent = ' (wildcard extension)' ;
    desc.style.cssText = 'color:#888;font-style:italic;font-size:11px;' ;
    el.appendChild(star) ; el.appendChild(desc) ; return el ;
  }

  function attrSort(keys) {
    return keys.sort(function(a, b) {
      if (a === '*') return 1 ; if (b === '*') return -1 ; return a.localeCompare(b) ;
    }) ;
  }

  function withCount(label, n) { return label + ' (' + n + ')' ; }

  if (_attrNestStack.length > 0) {
    var top = _attrNestStack[_attrNestStack.length - 1] ;
    if (top.isItem) {
      div.appendChild(navItem('Item', false, _navSelected === '__item__', function() { selectItem('__item__') ; })) ;
    } else if (top.isIfValues) {
      var ctx = resolveAttrNesting(false) ;
      var ifv = ctx.ifvMap || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Value', addNewIfValue)) ;
      Object.keys(ifv).sort().forEach(function(k) {
        div.appendChild(navItem(k, false, _navSelected === k,
          (function(key){ return function(){ selectItem(key) ; } ; })(k),
          (function(key){ return function(){ deleteIfValue(key) ; } ; })(k))) ;
      }) ;
    } else {
      // Regular nested attrs or siblings context
      var ctx = resolveAttrNesting(false) ;
      var nestedAttrs = ctx.attrsObj || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Attribute', addNewAttr)) ;
      attrSort(Object.keys(nestedAttrs)).forEach(function(k) {
        div.appendChild(navItem(attrLabel(k), false, _navSelected === k,
          (function(key){ return function(){ selectItem(key) ; } ; })(k),
          (function(key){ return function(){ deleteAttr(key) ; } ; })(k))) ;
      }) ;
    }
    return ;
  }

  if (_navTab === 'registry') {
    if (_navPath.length === 0) {
      div.appendChild(navItem('Details', false, _navSelected === 'fields', function() { selectItem('fields') ; })) ;
      div.appendChild(navItem(withCount('Attributes', Object.keys(model.attributes||{}).length), true, false, function() { drillDown(['attributes']) ; })) ;
      div.appendChild(navItem(withCount('Groups', Object.keys(model.groups||{}).length), true, false, function() { changeTab('groups') ; })) ;
    } else {
      var attrs = model.attributes || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Attribute', addNewAttr)) ;
      attrSort(Object.keys(attrs)).forEach(function(k) {
        div.appendChild(navItem(attrLabel(k), false, _navSelected === k,
          (function(key){ return function(){ selectItem(key) ; } ; })(k),
          (function(key){ return function(){ deleteAttr(key) ; } ; })(k))) ;
      }) ;
    }
  } else {
    if (_navPath.length === 0) {
      var groups = model.groups || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Group', addNewGroup)) ;
      Object.keys(groups).sort().forEach(function(k) {
        var rCount = Object.keys((groups[k]||{}).resources || {}).length ;
        div.appendChild(navItem(withCount(k, rCount), true, false,
          (function(key){ return function(){ drillDown([key]) ; } ; })(k),
          (function(key){ return function(){ deleteGroup(key) ; } ; })(k))) ;
      }) ;
    } else if (_navPath.length === 1) {
      var gk = _navPath[0] ;
      var grpData = model.groups[gk] || {} ;
      div.appendChild(navItem('Details', false, _navSelected === 'fields', function() { selectItem('fields') ; })) ;
      div.appendChild(navItem(withCount('Attributes', Object.keys(grpData.attributes||{}).length), true, false, function() { drillDown([gk, 'attributes']) ; })) ;
      div.appendChild(navItem(withCount('Resources', Object.keys(grpData.resources||{}).length), true, false, function() { drillDown([gk, 'resources']) ; })) ;
    } else if (_navPath.length === 2 && _navPath[1] === 'attributes') {
      var gk = _navPath[0] ;
      var attrs = (model.groups[gk] || {}).attributes || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Attribute', addNewAttr)) ;
      attrSort(Object.keys(attrs)).forEach(function(k) {
        div.appendChild(navItem(attrLabel(k), false, _navSelected === k,
          (function(key){ return function(){ selectItem(key) ; } ; })(k),
          (function(key){ return function(){ deleteAttr(key) ; } ; })(k))) ;
      }) ;
    } else if (_navPath.length === 2 && _navPath[1] === 'resources') {
      var gk = _navPath[0] ;
      var resources = (model.groups[gk] || {}).resources || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Resource', function(){ addNewResource(gk) ; })) ;
      Object.keys(resources).sort().forEach(function(k) {
        div.appendChild(navItem(k, true, false,
          (function(key){ return function(){ drillDown([gk, 'resources', key]) ; } ; })(k),
          (function(key){ return function(){ deleteResource(gk, key) ; } ; })(k))) ;
      }) ;
    } else if (_navPath.length === 3) {
      var gk = _navPath[0], rk = _navPath[2] ;
      var resData = ((model.groups[gk]||{}).resources||{})[rk] || {} ;
      div.appendChild(navItem('Details', false, _navSelected === 'fields', function() { selectItem('fields') ; })) ;
      div.appendChild(navItem(withCount('Version Attrs', Object.keys(resData.attributes||{}).length), true, false, function(){ drillDown([gk,'resources',rk,'versionattributes']) ; })) ;
      div.appendChild(navItem(withCount('Resource Attrs', Object.keys(resData.resourceattributes||{}).length), true, false, function(){ drillDown([gk,'resources',rk,'resourceattributes']) ; })) ;
      div.appendChild(navItem(withCount('Meta Attrs', Object.keys(resData.metaattributes||{}).length), true, false, function(){ drillDown([gk,'resources',rk,'metaattributes']) ; })) ;
    } else if (_navPath.length === 4) {
      var gk = _navPath[0], rk = _navPath[2], attrSec = _navPath[3] ;
      var dataKey = attrSec === 'versionattributes' ? 'attributes' : attrSec ;
      var res = ((model.groups[gk] || {}).resources || {})[rk] || {} ;
      var attrs = res[dataKey] || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Attribute', addNewAttr)) ;
      attrSort(Object.keys(attrs)).forEach(function(k) {
        div.appendChild(navItem(attrLabel(k), false, _navSelected === k,
          (function(key){ return function(){ selectItem(key) ; } ; })(k),
          (function(key){ return function(){ deleteAttr(key) ; } ; })(k))) ;
      }) ;
    }
  }
}

// ---- Right Panel ----

function buildRightPanel(div) {
  if (!_navSelected) {
    var hint = document.createElement('div') ; hint.className = 'editorHint' ;
    hint.textContent = '\u2190 Select an item from the left' ; div.appendChild(hint) ;
    // On mobile the nav is hidden in a dropdown — auto-open it so user isn't stranded
    var toggleBtn = document.querySelector('.navToggleBtn') ;
    if (toggleBtn && getComputedStyle(toggleBtn).display !== 'none') {
      setTimeout(function() { var o = window._editorOpenNav ; if (o) o() ; }, 50) ;
    }
    return ;
  }

  // Nested attribute context
  if (_attrNestStack.length > 0) {
    var top = _attrNestStack[_attrNestStack.length - 1] ;
    if (top.isItem && _navSelected === '__item__') {
      var ctx = resolveAttrNesting(false) ;
      renderItemForm(div, ctx.parentAttr ? (ctx.parentAttr.item || {}) : {}) ;
      return ;
    }
    if (top.isIfValues && _navSelected) {
      var ctx = resolveAttrNesting(false) ;
      renderIfValueForm(div, _navSelected, ctx.ifvMap || {}) ;
      return ;
    }
    if (!top.isItem && !top.isIfValues) {
      // Regular nested attrs or siblings
      var ctx2 = resolveAttrNesting(false) ;
      var nestedAttr = (ctx2.attrsObj || {})[_navSelected] || {} ;
      renderAttrForm(div, nestedAttr) ;
      return ;
    }
  }

  var model = _modelData || {} ;
  if (_navTab === 'registry') {
    if (_navSelected === 'fields') { renderRegistryFields(div) ; }
    else { renderAttrForm(div, (model.attributes || {})[_navSelected] || {}) ; }
  } else {
    var gk = _navPath.length > 0 ? _navPath[0] : null ;
    if (_navPath.length === 1 && _navSelected === 'fields') {
      renderGroupFields(div, gk) ;
    } else if (_navPath.length === 2 && _navPath[1] === 'attributes') {
      renderAttrForm(div, ((model.groups[gk] || {}).attributes || {})[_navSelected] || {}) ;
    } else if (_navPath.length === 3 && _navSelected === 'fields') {
      renderResourceFields(div, gk, _navPath[2]) ;
    } else if (_navPath.length === 4) {
      var attrSec = _navPath[3] ;
      var dataKey = attrSec === 'versionattributes' ? 'attributes' : attrSec ;
      var res = (((model.groups[gk] || {}).resources || {})[_navPath[2]] || {}) ;
      renderAttrForm(div, (res[dataKey] || {})[_navSelected] || {}) ;
    }
  }
}

// ---- Collect current editor into _modelData ----

function collectCurrentEditor() {
  if (!_navSelected) return ;

  // Nested attribute context — handle first
  if (_attrNestStack.length > 0) {
    var top = _attrNestStack[_attrNestStack.length - 1] ;
    if (top.isItem && _navSelected === '__item__') {
      var ctx = resolveAttrNesting(true) ;
      if (ctx.parentAttr) saveItemForm(ctx.parentAttr) ;
      return ;
    }
    if (top.isIfValues && _navSelected) {
      var ctx = resolveAttrNesting(true) ;
      saveIfValueFrom(ctx.ifvMap, _navSelected) ;
      return ;
    }
    if (!top.isItem && !top.isIfValues) {
      var ctx2 = resolveAttrNesting(true) ;
      if (ctx2.attrsObj) saveAttrFrom(ctx2.attrsObj, _navSelected) ;
      return ;
    }
    return ;
  }

  var model = _modelData || {} ;
  if (_navTab === 'registry') {
    if (_navSelected === 'fields') {
      var d = fv('ef_description') ; if (d) model.description = d ; else delete model.description ;
      var dc = fv('ef_documentation') ; if (dc) model.documentation = dc ; else delete model.documentation ;
      var lbls = collectLabels('ef_labels') ;
      if (Object.keys(lbls).length) model.labels = lbls ; else delete model.labels ;
    } else {
      if (!model.attributes) model.attributes = {} ;
      saveAttrFrom(model.attributes, _navSelected) ;
    }
  } else {
    var gk = _navPath.length > 0 ? _navPath[0] : null ; if (!gk) return ;
    if (!model.groups) model.groups = {} ;
    if (!model.groups[gk]) model.groups[gk] = {} ;
    var grp = model.groups[gk] ;
    if (_navPath.length === 1 && _navSelected === 'fields') {
      saveGroupFields(gk) ;
    } else if (_navPath.length === 2 && _navPath[1] === 'attributes') {
      if (!grp.attributes) grp.attributes = {} ;
      saveAttrFrom(grp.attributes, _navSelected) ;
    } else if (_navPath.length === 3 && _navSelected === 'fields') {
      var rk = _navPath[2] ;
      if (!grp.resources) grp.resources = {} ;
      if (!grp.resources[rk]) grp.resources[rk] = {} ;
      saveResourceFields(gk, rk) ;
    } else if (_navPath.length === 4) {
      var rk = _navPath[2], attrSec = _navPath[3] ;
      var dataKey = attrSec === 'versionattributes' ? 'attributes' : attrSec ;
      if (!grp.resources) grp.resources = {} ;
      if (!grp.resources[rk]) grp.resources[rk] = {} ;
      var res = grp.resources[rk] ;
      if (!res[dataKey]) res[dataKey] = {} ;
      saveAttrFrom(res[dataKey], _navSelected) ;
    }
  }
}

function saveAttrFrom(attrsObj, origKey) {
  var nameEl = document.getElementById('ef_name') ; if (!nameEl) return ;
  var newName = nameEl.value.trim() || origKey ;
  // Read existing entry first so we can preserve nested structures (attributes/item)
  // that are edited via drill-down and not touched by this form
  var existing = attrsObj[origKey] || {} ;
  var attr = { name: newName } ;
  var t = fv('ef_type') ; if (t) attr.type = t ;
  var d = fv('ef_description') ; if (d) attr.description = d ;
  var def = fv('ef_default') ; if (def !== '') attr.default = def ;
  var tgt = fv('ef_target') ;
  var targetEl = document.getElementById('ef_target') ;
  if (tgt && targetEl && !targetEl.disabled) attr.target = tgt ;
  var ncs = fv('ef_namecharset') ;
  var ncsEl = document.getElementById('ef_namecharset') ;
  if (ncs && ncsEl && !ncsEl.disabled) attr.namecharset = ncs ;
  var enm = collectEnum('ef_enum') ;
  if (enm.length) attr.enum = enm ;
  ['required','readonly','immutable','matchcase','matchversions','strict'].forEach(function(f) {
    var v = fvBool('ef_'+f) ;
    if (v === true) attr[f] = true ;
    else if (v === false) attr[f] = false ;
    else delete attr[f] ;
  }) ;
  // Preserve nested structures edited via drill-down (not part of this form)
  if (existing.attributes) attr.attributes = existing.attributes ;
  if (existing.item) attr.item = existing.item ;
  if (existing.ifvalues) attr.ifvalues = existing.ifvalues ;
  if (newName !== origKey && attrsObj[origKey] !== undefined) delete attrsObj[origKey] ;
  attrsObj[newName] = attr ;
  if (_navSelected === origKey) _navSelected = newName ;
}

function saveGroupFields(gk) {
  var model = _modelData || {} ;
  if (!model.groups) model.groups = {} ;
  var grp = model.groups[gk] || {} ;
  var plural = fv('ef_plural') ;
  setOrDel(grp, 'plural', plural) ; setOrDel(grp, 'singular', fv('ef_singular')) ;
  setOrDel(grp, 'description', fv('ef_description')) ; setOrDel(grp, 'documentation', fv('ef_documentation')) ;
  setOrDel(grp, 'icon', fv('ef_icon')) ; setOrDel(grp, 'modelversion', fv('ef_modelversion')) ;
  setOrDel(grp, 'modelcompatiblewith', fv('ef_modelcompatiblewith')) ;
  var lbls = collectLabels('ef_labels') ;
  if (Object.keys(lbls).length) grp.labels = lbls ; else delete grp.labels ;
  var cstrs = collectConstraints('ef_constraints') ;
  if (Object.keys(cstrs).length) grp.constraints = cstrs ; else delete grp.constraints ;
  var newKey = plural || gk ;
  if (newKey !== gk) { delete model.groups[gk] ; model.groups[newKey] = grp ; _navPath[0] = newKey ; }
  else model.groups[gk] = grp ;
}

function saveResourceFields(gk, rk) {
  var model = _modelData || {} ;
  var grp = (model.groups || {})[gk] || {} ;
  var res = (grp.resources || {})[rk] || {} ;
  var plural = fv('ef_plural') ;
  setOrDel(res, 'plural', plural) ; setOrDel(res, 'singular', fv('ef_singular')) ;
  setOrDel(res, 'description', fv('ef_description')) ; setOrDel(res, 'documentation', fv('ef_documentation')) ;
  setOrDel(res, 'icon', fv('ef_icon')) ; setOrDel(res, 'modelversion', fv('ef_modelversion')) ;
  setOrDel(res, 'modelcompatiblewith', fv('ef_modelcompatiblewith')) ;
  var maxv = fv('ef_maxversions') ;
  if (maxv !== '') res.maxversions = parseInt(maxv, 10) || 0 ; else delete res.maxversions ;
  setOrDel(res, 'versionmode', fv('ef_versionmode')) ;
  ['setversionid','hasdocument','singleversionroot','validateformat','validatecompatibility','strictvalidation'].forEach(function(f) {
    var v = fvBool('ef_'+f) ;
    if (v === true) res[f] = true ;
    else if (v === false) res[f] = false ;
    else delete res[f] ;
  }) ;
  var lbls = collectLabels('ef_labels') ;
  if (Object.keys(lbls).length) res.labels = lbls ; else delete res.labels ;
  var newKey = plural || rk ;
  if (newKey !== rk) { delete grp.resources[rk] ; grp.resources[newKey] = res ; _navPath[2] = newKey ; }
  else grp.resources[rk] = res ;
}

function setOrDel(obj, key, val) { if (val) obj[key] = val ; else delete obj[key] ; }

// ---- Form renderers ----

function addFormTitle(div, title) {
  var h = document.createElement('div') ; h.className = 'editorFormTitle' ;
  h.textContent = title ; div.appendChild(h) ;
}

function renderRegistryFields(div) {
  var m = _modelData || {} ;
  addFormTitle(div, 'Registry Details') ;
  div.appendChild(ef('ef_description', 'Description', m.description||'')) ;
  div.appendChild(ef('ef_documentation', 'Documentation', m.documentation||'')) ;
  div.appendChild(makeLabelsEditor('ef_labels', m.labels||{})) ;
}

function renderGroupFields(div, gk) {
  var grp = ((_modelData||{}).groups||{})[gk] || {} ;
  var titleEl = document.createElement('div') ; titleEl.className = 'editorFormTitle' ;
  titleEl.textContent = 'Group: ' + (grp.plural || gk) ; div.appendChild(titleEl) ;
  var pluralRow = ef('ef_plural', 'Plural', grp.plural||gk, true) ; div.appendChild(pluralRow) ;
  var pluralInp = pluralRow.querySelector('input') ;
  pluralInp.oninput = function() {
    var v = pluralInp.value.trim() || gk ;
    titleEl.textContent = 'Group: ' + v ;
    var bc = document.getElementById('bcGroupKey') ; if (bc) bc.textContent = v ;
  } ;
  div.appendChild(ef('ef_singular', 'Singular', grp.singular||'', true)) ;
  div.appendChild(ef('ef_description', 'Description', grp.description||'')) ;
  div.appendChild(ef('ef_documentation', 'Documentation', grp.documentation||'')) ;
  div.appendChild(ef('ef_icon', 'Icon URL', grp.icon||'')) ;
  div.appendChild(ef('ef_modelversion', 'Model Version', grp.modelversion||'')) ;
  div.appendChild(ef('ef_modelcompatiblewith', 'ModelCompatibleWith', grp.modelcompatiblewith||'')) ;
  div.appendChild(makeLabelsEditor('ef_labels', grp.labels||{})) ;
  div.appendChild(makeConstraintsEditor('ef_constraints', grp.constraints||{}, gk)) ;
}

function renderResourceFields(div, gk, rk) {
  var res = (((_modelData||{}).groups||{})[gk]||{}).resources||{} ;
  var r = res[rk] || {} ;
  var titleEl = document.createElement('div') ; titleEl.className = 'editorFormTitle' ;
  titleEl.textContent = 'Resource: ' + (r.plural || rk) ; div.appendChild(titleEl) ;
  var pluralRow = ef('ef_plural', 'Plural', r.plural||rk, true) ; div.appendChild(pluralRow) ;
  var pluralInp = pluralRow.querySelector('input') ;
  pluralInp.oninput = function() {
    var v = pluralInp.value.trim() || rk ;
    titleEl.textContent = 'Resource: ' + v ;
    var bc = document.getElementById('bcResourceKey') ; if (bc) bc.textContent = v ;
  } ;
  div.appendChild(ef('ef_singular', 'Singular', r.singular||'', true)) ;
  div.appendChild(ef('ef_description', 'Description', r.description||'')) ;
  div.appendChild(ef('ef_documentation', 'Documentation', r.documentation||'')) ;
  div.appendChild(ef('ef_icon', 'Icon URL', r.icon||'')) ;
  div.appendChild(ef('ef_modelversion', 'Model Version', r.modelversion||'')) ;
  div.appendChild(ef('ef_modelcompatiblewith', 'ModelCompatibleWith', r.modelcompatiblewith||'')) ;
  div.appendChild(efNum('ef_maxversions', 'Max Versions', r.maxversions)) ;
  div.appendChild(ef('ef_versionmode', 'Version Mode', r.versionmode||'')) ;
  var optSec = document.createElement('div') ; optSec.className = 'editorSectionLabel' ; optSec.textContent = 'Options' ;
  div.appendChild(optSec) ;
  var optList = [
    ['hasdocument',          'Has Document',          r.hasdocument],
    ['setversionid',         'Set Version ID',         r.setversionid],
    ['singleversionroot',    'Single Version Root',    r.singleversionroot],
    ['strictvalidation',     'Strict Validation',      r.strictvalidation],
    ['validatecompatibility','Validate Compatibility', r.validatecompatibility],
    ['validateformat',       'Validate Format',        r.validateformat]
  ] ;
  var boolGrid = document.createElement('div') ; boolGrid.className = 'boolGrid' ;
  optList.forEach(function(t) { boolGrid.appendChild(efBool('ef_'+t[0], t[1], t[2])) ; }) ;
  div.appendChild(boolGrid) ;
  div.appendChild(makeLabelsEditor('ef_labels', r.labels||{})) ;
}

function renderAttrForm(div, attr) {
  // Determine if this is the versionattributes context (matchversions only shown here)
  var isVersionAttrs = (_navPath.length === 4 && _navPath[3] === 'versionattributes') ;

  // Title with live update as name is typed
  var titleEl = document.createElement('div') ; titleEl.className = 'editorFormTitle' ;
  titleEl.textContent = 'Attribute: ' + (attr.name || _navSelected || '') ;
  div.appendChild(titleEl) ;

  var origInp = document.createElement('input') ; origInp.type = 'hidden' ;
  origInp.id = 'ef_origname' ; origInp.value = attr.name || _navSelected || '' ;
  div.appendChild(origInp) ;

  var nameRow = ef('ef_name', 'Name', attr.name || _navSelected || '', true) ;
  var nameInp = nameRow.querySelector('input') ;
  nameInp.maxLength = 63 ;
  nameInp.title = 'Lowercase letters, digits, underscore only; max 63 chars; cannot start with a digit. Use * for wildcard extension.' ;
  nameInp.oninput = function() {
    var raw = nameInp.value ;
    if (raw.indexOf('*') !== -1) {
      // Any input with * collapses to just '*'
      nameInp.value = '*' ;
    } else {
      var cleaned = raw.toLowerCase().replace(/[^a-z0-9_]/g, '') ;
      if (cleaned !== raw) {
        var pos = nameInp.selectionStart - (raw.length - cleaned.length) ;
        nameInp.value = cleaned ; nameInp.selectionStart = nameInp.selectionEnd = Math.max(0, pos) ;
      }
    }
    var v = nameInp.value.trim() || '\u2026' ;
    titleEl.textContent = 'Attribute: ' + v ;
    var navEl = document.querySelector('.navItemSelected') ;
    if (navEl) { var sp = navEl.firstChild ; if (sp) sp.textContent = v ; }
  } ;
  div.appendChild(nameRow) ;

  // Type dropdown
  var typeRow = document.createElement('div') ; typeRow.className = 'editorField' ;
  var typeLbl = document.createElement('label') ; typeLbl.textContent = 'Type:' ;
  var typeReq = document.createElement('span') ; typeReq.textContent = ' *' ; typeReq.style.cssText = 'color:#c00;font-weight:bold;' ;
  typeLbl.appendChild(typeReq) ;
  var typeSel = document.createElement('select') ; typeSel.id = 'ef_type' ; typeSel.className = 'editorInput' ;
  ['boolean','decimal','integer','string','timestamp',
   'uinteger','uri','uriabsolute','urirelative','uritemplate','url','urlabsolute','urlrelative','xid','xidtype',
   'any','array','map','object'
  ].forEach(function(opt) {
    var o = document.createElement('option') ; o.value = opt ; o.textContent = opt ;
    if ((attr.type||'string') === opt) o.selected = true ;
    typeSel.appendChild(o) ;
  }) ;
  typeRow.appendChild(typeLbl) ;
  var typeWrap = document.createElement('div') ; typeWrap.className = 'editorSelectWrap' ;
  typeWrap.appendChild(typeSel) ; typeRow.appendChild(typeWrap) ; div.appendChild(typeRow) ;

  // Nested-type drill-down button — right below Type, aligned with the dropdown
  var nestBtnRow = document.createElement('div') ; nestBtnRow.className = 'editorField' ;
  nestBtnRow.style.marginBottom = '6px' ;
  var nestLblSpacer = document.createElement('label') ; nestLblSpacer.style.visibility = 'hidden' ;
  nestBtnRow.appendChild(nestLblSpacer) ;
  var nestBtn = document.createElement('button') ;
  nestBtn.className = 'editorBtn navDrillBtn' ;
  nestBtn.style.cssText = 'font-size:11px;padding:3px 8px;' ;
  var currentAttrKey = _navSelected ;
  function updateNestBtn() {
    var t = typeSel.value ;
    if (t === 'object') {
      var cnt = Object.keys(attr.attributes || {}).length ;
      nestBtn.textContent = '\u25b6 Edit Nested Attributes' + (cnt ? ' ('+cnt+')' : '') ;
      nestBtn.style.display = '' ;
      nestBtn.onclick = function() { drillIntoAttr(currentAttrKey, false) ; } ;
    } else if (t === 'map' || t === 'array') {
      nestBtn.textContent = '\u25b6 Edit ' + t + ' details' ;
      nestBtn.style.display = '' ;
      nestBtn.onclick = function() { drillIntoAttr(currentAttrKey, true) ; } ;
    } else {
      nestBtn.style.display = 'none' ;
    }
  }
  nestBtnRow.appendChild(nestBtn) ; div.appendChild(nestBtnRow) ;
  updateNestBtn() ;
  typeSel.addEventListener('change', updateNestBtn) ;

  div.appendChild(ef('ef_description', 'Description', attr.description||'')) ;
  div.appendChild(ef('ef_default', 'Default', attr.default !== undefined ? String(attr.default) : '')) ;

  // Target — text field, only relevant for url/xid
  var targetRow = ef('ef_target', 'Target', attr.target||'') ; div.appendChild(targetRow) ;
  var targetInp = targetRow.querySelector('input') ;
  targetInp.placeholder = 'e.g. /groups/resources' ;

  // Name Charset — dropdown, only relevant for type=object
  var ncsRow = document.createElement('div') ; ncsRow.className = 'editorField' ;
  var ncsLbl = document.createElement('label') ; ncsLbl.textContent = 'Name Charset:' ;
  var ncsSel = document.createElement('select') ; ncsSel.id = 'ef_namecharset' ; ncsSel.className = 'editorInput' ;
  var ncsWrap = document.createElement('div') ; ncsWrap.className = 'editorSelectWrap' ;
  [['','(default / strict)'],['strict','strict'],['extended','extended']].forEach(function(p) {
    var o = document.createElement('option') ; o.value = p[0] ; o.textContent = p[1] ;
    if ((attr.namecharset||'') === p[0]) o.selected = true ;
    ncsSel.appendChild(o) ;
  }) ;
  ncsWrap.appendChild(ncsSel) ; ncsRow.appendChild(ncsLbl) ; ncsRow.appendChild(ncsWrap) ; div.appendChild(ncsRow) ;

  // Enable/disable target and namecharset based on current type
  function syncTypeFields() {
    var t = typeSel.value ;
    var targetTypes = {url:1,urlabsolute:1,urlrelative:1,uri:1,uriabsolute:1,urirelative:1,uritemplate:1,xid:1,xidtype:1} ;
    targetInp.disabled = !targetTypes[t] ;
    targetInp.style.opacity = targetInp.disabled ? '0.4' : '1' ;
    ncsSel.disabled = (t !== 'object') ;
    ncsSel.style.opacity = ncsSel.disabled ? '0.4' : '1' ;
  }
  syncTypeFields() ;
  typeSel.addEventListener('change', syncTypeFields) ;

  div.appendChild(makeEnumEditor('ef_enum', Array.isArray(attr.enum) ? attr.enum : [])) ;
  var optSec = document.createElement('div') ; optSec.className = 'editorSectionLabel' ; optSec.textContent = 'Options' ;
  div.appendChild(optSec) ;
  var optList = [
    ['immutable',  'Immutable',  attr.immutable],
    ['matchcase',  'Match Case', attr.matchcase],
    ['readonly',   'Read Only',  attr.readonly],
    ['required',   'Required',   attr.required],
    ['strict',     'Strict',     attr.strict]
  ] ;
  if (isVersionAttrs) optList.push(['matchversions','Match Versions', attr.matchversions]) ;
  // Sort alphabetically by label
  optList.sort(function(a,b){ return a[1].localeCompare(b[1]) ; }) ;
  var boolGrid = document.createElement('div') ; boolGrid.className = 'boolGrid' ;
  optList.forEach(function(t) { boolGrid.appendChild(efBool('ef_'+t[0], t[1], t[2])) ; }) ;
  div.appendChild(boolGrid) ;

  // If-Values drill-down button — left-aligned under section header
  var ifvSec = document.createElement('div') ; ifvSec.className = 'editorSectionLabel' ; ifvSec.textContent = 'If-Values' ;
  div.appendChild(ifvSec) ;
  var ifvCount = Object.keys(attr.ifvalues || {}).length ;
  if (_modelReadOnly && !ifvCount) {
    var ifvNone = document.createElement('span') ; ifvNone.textContent = '\u2014 none \u2014' ;
    ifvNone.style.cssText = 'color:#aaa;font-size:12px;font-style:italic;margin-left:4px;' ;
    div.appendChild(ifvNone) ;
  } else {
    var ifvBtn = document.createElement('button') ; ifvBtn.className = 'editorBtn navDrillBtn' ;
    ifvBtn.style.cssText = 'font-size:11px;padding:3px 8px;margin-bottom:6px;' ;
    ifvBtn.textContent = '\u25b6 If-Values' + (ifvCount ? ' ('+ifvCount+')' : '') ;
    ifvBtn.onclick = function() {
      collectCurrentEditor() ;
      var resolvedKey = _navSelected || currentAttrKey ;
      _attrNestStack.push({key:resolvedKey, isIfValues:true}) ;
      _navSelected = null ; renderEditor() ;
    } ;
    div.appendChild(ifvBtn) ;
  }
}

function renderItemForm(div, item) {
  item = item || {} ;
  // Determine parent type from stack (map/array) for title
  var parentType = 'map' ;
  for (var si = _attrNestStack.length-1; si >= 0; si--) {
    if (_attrNestStack[si].isItem) { parentType = _attrNestStack[si].attrType || 'map' ; break ; }
  }
  var titleEl = document.createElement('div') ; titleEl.className = 'editorFormTitle' ;
  titleEl.textContent = parentType.charAt(0).toUpperCase() + parentType.slice(1) + ' Details' ;
  div.appendChild(titleEl) ;

  // Type dropdown
  var typeRow = document.createElement('div') ; typeRow.className = 'editorField' ;
  var typeLbl = document.createElement('label') ; typeLbl.textContent = 'Type:' ;
  var typeSel = document.createElement('select') ; typeSel.id = 'ef_item_type' ; typeSel.className = 'editorInput' ;
  ['boolean','decimal','integer','string','timestamp',
   'uinteger','uri','uriabsolute','urirelative','uritemplate','url','urlabsolute','urlrelative','xid','xidtype',
   'any','array','map','object'
  ].forEach(function(opt) {
    var o = document.createElement('option') ; o.value = opt ; o.textContent = opt ;
    if ((item.type||'string') === opt) o.selected = true ;
    typeSel.appendChild(o) ;
  }) ;
  typeRow.appendChild(typeLbl) ;
  var typeWrap = document.createElement('div') ; typeWrap.className = 'editorSelectWrap' ;
  typeWrap.appendChild(typeSel) ; typeRow.appendChild(typeWrap) ; div.appendChild(typeRow) ;

  // Nested-type drill-down button — right below Type, aligned with the dropdown
  var nestBtnRow = document.createElement('div') ; nestBtnRow.className = 'editorField' ;
  nestBtnRow.style.marginBottom = '6px' ;
  var nestLblSpacer2 = document.createElement('label') ; nestLblSpacer2.style.visibility = 'hidden' ;
  nestBtnRow.appendChild(nestLblSpacer2) ;
  var nestBtn = document.createElement('button') ; nestBtn.className = 'editorBtn navDrillBtn' ;
  nestBtn.style.cssText = 'font-size:11px;padding:3px 8px;' ;
  function updateItemNestBtn() {
    var t = typeSel.value ;
    if (t === 'object') {
      var cnt = Object.keys(item.attributes || {}).length ;
      nestBtn.textContent = '\u25b6 Edit Nested Attributes' + (cnt ? ' ('+cnt+')' : '') ;
      nestBtn.style.display = '' ;
      nestBtn.onclick = function() {
        var top = _attrNestStack[_attrNestStack.length - 1] ;
        var parentKey = top ? top.key : null ; if (!parentKey) return ;
        var ctx = resolveAttrNesting(true) ;
        if (ctx.parentAttr) saveItemForm(ctx.parentAttr) ;
        _attrNestStack.push({key:'__item__', isItem:false}) ;
        _navSelected = null ; renderEditor() ;
      } ;
    } else if (t === 'map' || t === 'array') {
      nestBtn.textContent = '\u25b6 Edit ' + t + ' details' ;
      nestBtn.style.display = '' ;
      nestBtn.onclick = function() {
        var top = _attrNestStack[_attrNestStack.length - 1] ;
        var parentKey = top ? top.key : null ; if (!parentKey) return ;
        var ctx = resolveAttrNesting(true) ;
        if (ctx.parentAttr) saveItemForm(ctx.parentAttr) ;
        _attrNestStack.push({key:'__item__', isItem:true, attrType:t}) ;
        _navSelected = '__item__' ; renderEditor() ;
      } ;
    } else {
      nestBtn.style.display = 'none' ;
    }
  }
  nestBtnRow.appendChild(nestBtn) ; div.appendChild(nestBtnRow) ;

  var targetRow = ef('ef_item_target', 'Target', item.target||'') ; div.appendChild(targetRow) ;
  var targetInp = targetRow.querySelector('input') ;
  targetInp.placeholder = 'e.g. /groups/resources' ;

  var ncsRow = document.createElement('div') ; ncsRow.className = 'editorField' ;
  var ncsLbl = document.createElement('label') ; ncsLbl.textContent = 'Name Charset:' ;
  var ncsSel = document.createElement('select') ; ncsSel.id = 'ef_item_namecharset' ; ncsSel.className = 'editorInput' ;
  var ncsWrap = document.createElement('div') ; ncsWrap.className = 'editorSelectWrap' ;
  [['','(default / strict)'],['strict','strict'],['extended','extended']].forEach(function(p) {
    var o = document.createElement('option') ; o.value = p[0] ; o.textContent = p[1] ;
    if ((item.namecharset||'') === p[0]) o.selected = true ;
    ncsSel.appendChild(o) ;
  }) ;
  ncsWrap.appendChild(ncsSel) ; ncsRow.appendChild(ncsLbl) ; ncsRow.appendChild(ncsWrap) ; div.appendChild(ncsRow) ;

  // These fields are only meaningful for complex (object/map/array) item types
  var complexSec = document.createElement('div') ;
  complexSec.appendChild(ef('ef_item_description', 'Description', item.description||'')) ;
  complexSec.appendChild(ef('ef_item_default', 'Default', item.default !== undefined ? String(item.default) : '')) ;
  complexSec.appendChild(makeEnumEditor('ef_item_enum', Array.isArray(item.enum) ? item.enum : [])) ;
  var optSec = document.createElement('div') ; optSec.className = 'editorSectionLabel' ; optSec.textContent = 'Options' ;
  complexSec.appendChild(optSec) ;
  var optList = [
    ['item_readonly', 'Read Only', item.readonly],
    ['item_strict',   'Strict',    item.strict]
  ] ;
  var boolGrid = document.createElement('div') ; boolGrid.className = 'boolGrid' ;
  optList.forEach(function(t) { boolGrid.appendChild(efBool('ef_'+t[0], t[1], t[2])) ; }) ;
  complexSec.appendChild(boolGrid) ;
  div.appendChild(complexSec) ;

  function syncItemTypeFields() {
    var t = typeSel.value ;
    var targetTypes = {url:1,urlabsolute:1,urlrelative:1,uri:1,uriabsolute:1,urirelative:1,uritemplate:1,xid:1,xidtype:1} ;
    targetInp.disabled = !targetTypes[t] ;
    targetInp.style.opacity = targetInp.disabled ? '0.4' : '1' ;
    ncsSel.disabled = (t !== 'object') ;
    ncsSel.style.opacity = ncsSel.disabled ? '0.4' : '1' ;
    updateItemNestBtn() ;
    // description/default/enum/options are only relevant for complex types
    complexSec.style.display = {object:1,map:1,array:1}[t] ? '' : 'none' ;
  }
  updateItemNestBtn() ;
  syncItemTypeFields() ;
  typeSel.addEventListener('change', syncItemTypeFields) ;
}

function saveItemForm(parentAttr) {
  if (!parentAttr) return ;
  if (!parentAttr.item) parentAttr.item = {} ;
  var itm = parentAttr.item ;
  var t = fv('ef_item_type') ; if (t) itm.type = t ; else delete itm.type ;
  var d = fv('ef_item_description') ; if (d) itm.description = d ; else delete itm.description ;
  var def = fv('ef_item_default') ; if (def !== '') itm.default = def ; else delete itm.default ;
  var targetEl = document.getElementById('ef_item_target') ;
  if (targetEl && !targetEl.disabled) { var tgt = targetEl.value.trim() ; if (tgt) itm.target = tgt ; else delete itm.target ; }
  var ncsEl = document.getElementById('ef_item_namecharset') ;
  if (ncsEl && !ncsEl.disabled) { var ncs = ncsEl.value ; if (ncs) itm.namecharset = ncs ; else delete itm.namecharset ; }
  var enm = collectEnum('ef_item_enum') ;
  if (enm.length) itm.enum = enm ; else delete itm.enum ;
  var rov = fvBool('ef_item_readonly') ; if (rov === true) itm.readonly = true ; else if (rov === false) itm.readonly = false ; else delete itm.readonly ;
  var stv = fvBool('ef_item_strict') ; if (stv === true) itm.strict = true ; else if (stv === false) itm.strict = false ; else delete itm.strict ;
}

function uniqueKey(obj, base) {
  if (!obj || !obj[base]) return base ;
  var i = 2 ; while (obj[base+i]) i++ ; return base+i ;
}

function addNewGroup() {
  collectCurrentEditor() ;
  var m = _modelData || {} ; if (!m.groups) m.groups = {} ;
  var key = uniqueKey(m.groups, 'new') ;
  m.groups[key] = {plural:'',singular:''} ;
  markDirty() ; _navTab = 'groups' ; _navPath = [key] ; _navSelected = 'fields' ; renderEditor() ;
}

function addNewResource(gk) {
  collectCurrentEditor() ;
  var m = _modelData || {} ; var grp = (m.groups||{})[gk] ; if (!grp) return ;
  if (!grp.resources) grp.resources = {} ;
  var key = uniqueKey(grp.resources, 'new') ;
  grp.resources[key] = {plural:'',singular:''} ;
  markDirty() ; _navPath = [gk,'resources',key] ; _navSelected = 'fields' ; renderEditor() ;
}

function addNewAttr() {
  collectCurrentEditor() ;
  // Nested context: use resolved attrs container
  if (_attrNestStack.length > 0) {
    var top = _attrNestStack[_attrNestStack.length - 1] ;
    if (!top.isItem && !top.isIfValues) {
      var ctx = resolveAttrNesting(true) ;
      var nestedAttrs = ctx.attrsObj ;
      if (!nestedAttrs) return ;
      var key = uniqueKey(nestedAttrs, 'new') ;
      nestedAttrs[key] = {name:key, type:'string'} ;
      markDirty() ; _navSelected = key ; renderEditor() ;
    }
    return ;
  }
  var m = _modelData || {} ; var attrsObj ;
  if (_navTab === 'registry') {
    if (!m.attributes) m.attributes = {} ; attrsObj = m.attributes ;
  } else {
    var gk = _navPath[0] ; var grp = (m.groups||{})[gk] ; if (!grp) return ;
    if (_navPath.length === 2 && _navPath[1] === 'attributes') {
      if (!grp.attributes) grp.attributes = {} ; attrsObj = grp.attributes ;
    } else if (_navPath.length === 4) {
      var rk = _navPath[2], attrSec = _navPath[3] ;
      var dataKey = attrSec === 'versionattributes' ? 'attributes' : attrSec ;
      var res = (grp.resources||{})[rk] ; if (!res) return ;
      if (!res[dataKey]) res[dataKey] = {} ; attrsObj = res[dataKey] ;
    }
  }
  if (!attrsObj) return ;
  var key = uniqueKey(attrsObj, 'new') ;
  attrsObj[key] = {name:key, type:'string'} ;
  markDirty() ; _navSelected = key ; renderEditor() ;
}

function confirmDel(label, fn) {
  if (confirm('Delete ' + label + '?')) fn() ;
}

function deleteGroup(gk) {
  var m = _modelData || {} ; if (m.groups) delete m.groups[gk] ;
  markDirty() ; _navPath = [] ; _navSelected = null ; renderEditor() ;
}

function deleteResource(gk, rk) {
  var m = _modelData || {} ; var grp = (m.groups||{})[gk] ;
  if (grp && grp.resources) delete grp.resources[rk] ;
  markDirty() ; _navPath = [gk,'resources'] ; _navSelected = null ; renderEditor() ;
}

function deleteAttr(key) {
  // Nested context: delete from resolved attrs container
  if (_attrNestStack.length > 0) {
    var top = _attrNestStack[_attrNestStack.length - 1] ;
    if (!top.isItem && !top.isIfValues) {
      var ctx = resolveAttrNesting(false) ;
      if (ctx.attrsObj) delete ctx.attrsObj[key] ;
      markDirty() ; if (_navSelected === key) _navSelected = null ; renderEditor() ;
    }
    return ;
  }
  var m = _modelData || {} ; var attrsObj ;
  if (_navTab === 'registry') { attrsObj = m.attributes ; }
  else {
    var gk = _navPath[0] ; var grp = (m.groups||{})[gk] ;
    if (grp) {
      if (_navPath.length === 2 && _navPath[1] === 'attributes') attrsObj = grp.attributes ;
      else if (_navPath.length === 4) {
        var dataKey = _navPath[3] === 'versionattributes' ? 'attributes' : _navPath[3] ;
        var res = (grp.resources||{})[_navPath[2]] ; if (res) attrsObj = res[dataKey] ;
      }
    }
  }
  if (attrsObj) delete attrsObj[key] ;
  markDirty() ; if (_navSelected === key) _navSelected = null ; renderEditor() ;
}

// ---- UI helpers ----

function ef(id, label, value, required) {
  var row = document.createElement('div') ; row.className = 'editorField' ;
  var lbl = document.createElement('label') ; lbl.textContent = label + ':' ;
  if (required) {
    var req = document.createElement('span') ; req.textContent = ' *' ;
    req.style.cssText = 'color:#c00;font-weight:bold;' ;
    lbl.appendChild(req) ;
  }
  var inp = document.createElement('input') ;
  inp.type = 'text' ; inp.id = id ; inp.value = value ; inp.className = 'editorInput' ;
  row.appendChild(lbl) ; row.appendChild(inp) ; return row ;
}

function efNum(id, label, value) {
  var row = document.createElement('div') ; row.className = 'editorField' ;
  var lbl = document.createElement('label') ; lbl.textContent = label + ':' ;
  var inp = document.createElement('input') ;
  inp.type = 'number' ; inp.id = id ; inp.min = '0' ; inp.className = 'editorInput' ;
  inp.value = (value !== undefined && value !== null) ? value : '' ;
  row.appendChild(lbl) ; row.appendChild(inp) ; return row ;
}

function ecb(id, label, checked) {
  var row = document.createElement('div') ; row.className = 'editorCheckRow' ;
  var cb = document.createElement('input') ; cb.type = 'checkbox' ; cb.id = id ; cb.checked = checked ;
  var lbl = document.createElement('label') ; lbl.textContent = label ; lbl.htmlFor = id ;
  row.appendChild(cb) ; row.appendChild(lbl) ; return row ;
}

function efBool(id, label, value) {
  var cell = document.createElement('div') ; cell.className = 'boolCell' ;
  var lbl = document.createElement('label') ; lbl.textContent = label + ':' ;
  var seg = document.createElement('div') ;
  var cur = (value === true) ? 'true' : (value === false) ? 'false' : '' ;
  seg.className = 'boolSeg' + (_modelReadOnly ? ' boolSegReadOnly' : '') ;
  seg.id = id ; seg.dataset.val = cur ;
  var btns = [['true','true'],['false','false'],['\u2014','']] ;
  btns.forEach(function(b) {
    var btn = document.createElement('button') ; btn.type = 'button' ;
    btn.textContent = b[0] ; btn.className = 'boolSegBtn' + (cur === b[1] ? ' boolSegActive' : '') ;
    if (b[1] === '') btn.title = 'Unspecified' ;
    btn.onclick = function() {
      seg.dataset.val = b[1] ;
      seg.querySelectorAll('.boolSegBtn').forEach(function(x){ x.classList.remove('boolSegActive') ; }) ;
      btn.classList.add('boolSegActive') ;
      markDirty() ;
    } ;
    seg.appendChild(btn) ;
  }) ;
  cell.appendChild(lbl) ; cell.appendChild(seg) ; return cell ;
}

function fvBool(id) {
  var el = document.getElementById(id) ;
  if (!el) return null ;
  if (el.dataset.val === 'true') return true ;
  if (el.dataset.val === 'false') return false ;
  return null ;
}

function makeLabelsEditor(containerId, labels) {
  var sec = document.createElement('div') ;
  var hdr = document.createElement('div') ; hdr.className = 'editorSectionLabel' ;
  hdr.textContent = 'Labels' ; sec.appendChild(hdr) ;
  if (_modelReadOnly && Object.keys(labels).length === 0) {
    var none = document.createElement('span') ; none.textContent = '\u2014 none \u2014' ;
    none.style.cssText = 'color:#aaa;font-size:12px;font-style:italic;margin-left:4px;' ;
    sec.appendChild(none) ; return sec ;
  }
  var wrap = document.createElement('div') ; wrap.className = 'labelsWrap' ;
  if (!_modelReadOnly) {
    var addBtn = document.createElement('button') ; addBtn.className = 'editorBtn' ;
    addBtn.textContent = '+ Add' ; addBtn.title = 'Add Label' ;
    addBtn.style.flexShrink = '0' ; addBtn.style.alignSelf = 'flex-start' ;
    addBtn.onclick = function() { rowsDiv.appendChild(makeLabelRow('','')) ; } ;
    wrap.appendChild(addBtn) ;
  }
  var rowsDiv = document.createElement('div') ; rowsDiv.className = 'labelsRows' ;
  rowsDiv.id = containerId ;
  Object.keys(labels).forEach(function(k) { rowsDiv.appendChild(makeLabelRow(k, labels[k])) ; }) ;
  wrap.appendChild(rowsDiv) ;
  sec.appendChild(wrap) ;
  return sec ;
}

function makeLabelRow(k, v) {
  var row = document.createElement('div') ; row.className = 'labelRow' ;
  var ki = document.createElement('input') ; ki.type = 'text' ; ki.placeholder = 'key' ;
  ki.value = k ; ki.className = 'editorInput labelKey' ;
  var vi = document.createElement('input') ; vi.type = 'text' ; vi.placeholder = 'value' ;
  vi.value = v ; vi.className = 'editorInput labelVal' ;
  var rb = document.createElement('button') ; rb.className = 'rmBtn' ; rb.textContent = 'Remove' ;
  rb.onclick = function() { confirmDel('this label', function() { row.remove() ; }) ; } ;
  row.appendChild(ki) ; row.appendChild(vi) ; row.appendChild(rb) ; return row ;
}

function collectLabels(containerId) {
  var container = document.getElementById(containerId) ; var labels = {} ;
  if (!container) return labels ;
  container.querySelectorAll('.labelRow').forEach(function(row) {
    var inputs = row.querySelectorAll('input') ;
    var k = inputs[0] ? inputs[0].value.trim() : '' ;
    var v = inputs[1] ? inputs[1].value.trim() : '' ;
    if (k) labels[k] = v ;
  }) ;
  return labels ;
}

function makeEnumEditor(containerId, values) {
  var sec = document.createElement('div') ;
  var hdr = document.createElement('div') ; hdr.className = 'editorSectionLabel' ;
  hdr.textContent = 'Enum' ; sec.appendChild(hdr) ;
  if (_modelReadOnly && values.length === 0) {
    var none = document.createElement('span') ; none.textContent = '\u2014 none \u2014' ;
    none.style.cssText = 'color:#aaa;font-size:12px;font-style:italic;margin-left:4px;' ;
    sec.appendChild(none) ; return sec ;
  }
  var wrap = document.createElement('div') ; wrap.className = 'labelsWrap' ;
  if (!_modelReadOnly) {
    var addBtn = document.createElement('button') ; addBtn.className = 'editorBtn' ;
    addBtn.textContent = '+ Add' ; addBtn.title = 'Add enum value' ;
    addBtn.style.flexShrink = '0' ; addBtn.style.alignSelf = 'flex-start' ;
    addBtn.onclick = function() { rowsDiv.appendChild(makeEnumRow('')) ; } ;
    wrap.appendChild(addBtn) ;
  }
  var rowsDiv = document.createElement('div') ; rowsDiv.id = containerId ;
  rowsDiv.style.cssText = 'flex:1;display:flex;flex-direction:column;gap:4px;' ;
  values.forEach(function(v) { rowsDiv.appendChild(makeEnumRow(String(v))) ; }) ;
  wrap.appendChild(rowsDiv) ;
  sec.appendChild(wrap) ;
  return sec ;
}

function makeEnumRow(val) {
  var row = document.createElement('div') ; row.className = 'labelRow' ;
  var inp = document.createElement('input') ; inp.type = 'text' ; inp.placeholder = 'value' ;
  inp.value = val ; inp.className = 'editorInput' ; inp.style.flex = '1' ;
  var rb = document.createElement('button') ; rb.className = 'rmBtn' ; rb.textContent = 'Remove' ;
  rb.onclick = function() { confirmDel('this enum value', function() { row.remove() ; }) ; } ;
  row.appendChild(inp) ; row.appendChild(rb) ; return row ;
}

function collectEnum(containerId) {
  var container = document.getElementById(containerId) ; var vals = [] ;
  if (!container) return vals ;
  container.querySelectorAll('.labelRow').forEach(function(row) {
    var inp = row.querySelector('input') ;
    var v = inp ? inp.value.trim() : '' ;
    if (v !== '') vals.push(v) ;
  }) ;
  return vals ;
}

function getScalarAttrNames(attrsObj) {
  var scalars = ['boolean','decimal','integer','string','timestamp','uinteger',
    'uri','uriabsolute','urirelative','uritemplate','url','urlabsolute','urlrelative','xid','xidtype'] ;
  var names = [] ;
  Object.keys(attrsObj||{}).forEach(function(k) {
    if (k === '*') return ;
    var a = attrsObj[k] ; if (!a) return ;
    if (scalars.indexOf(a.type||'string') !== -1) names.push(k) ;
  }) ;
  return names.sort() ;
}

function populateCstrAttrSel(attrSel, gk, resPlural, selectedVal) {
  while (attrSel.firstChild) attrSel.removeChild(attrSel.firstChild) ;
  var blank = document.createElement('option') ; blank.value = '' ; blank.textContent = '-- attribute --' ;
  attrSel.appendChild(blank) ;
  if (!resPlural) return ;
  var res = (((_modelData||{}).groups||{})[gk]||{}).resources||{} ;
  var rObj = res[resPlural] ;
  if (!rObj) return ;
  getScalarAttrNames(rObj.attributes).forEach(function(n) {
    var o = document.createElement('option') ; o.value = n ; o.textContent = n ;
    if (n === selectedVal) o.selected = true ;
    attrSel.appendChild(o) ;
  }) ;
}

function makeConstraintsEditor(containerId, constraints, gk) {
  var sec = document.createElement('div') ;
  var hdr = document.createElement('div') ; hdr.className = 'editorSectionLabel' ;
  hdr.textContent = 'Constraints' ; sec.appendChild(hdr) ;
  if (_modelReadOnly && Object.keys(constraints||{}).length === 0) {
    var none = document.createElement('span') ; none.textContent = '\u2014 none \u2014' ;
    none.style.cssText = 'color:#aaa;font-size:12px;font-style:italic;margin-left:4px;' ;
    sec.appendChild(none) ; return sec ;
  }
  var blocksDiv = document.createElement('div') ; blocksDiv.id = containerId ;
  blocksDiv.style.cssText = 'display:flex;flex-direction:column;' ;
  Object.keys(constraints||{}).forEach(function(k) {
    blocksDiv.appendChild(makeConstraintRow(k, constraints[k]||{}, gk)) ;
  }) ;
  sec.appendChild(blocksDiv) ;
  if (!_modelReadOnly) {
    var addBtn = document.createElement('button') ; addBtn.className = 'editorBtn' ;
    addBtn.textContent = '+ Add' ; addBtn.title = 'Add Constraint' ;
    addBtn.style.cssText = 'margin-top:4px;align-self:flex-start;' ;
    addBtn.onclick = function() { blocksDiv.appendChild(makeConstraintRow('', {}, gk)) ; } ;
    sec.appendChild(addBtn) ;
  }
  return sec ;
}

function makeConstraintRow(key, constraint, gk) {
  var idx = _cstrCounter++ ;
  var block = document.createElement('div') ; block.className = 'constraintBlock' ;
  block.dataset.cstrIdx = String(idx) ;
  block.dataset.origKey = key ; // preserve original key as fallback if selects can't resolve

  // Header row with title and Remove button
  var blockHdr = document.createElement('div') ; blockHdr.className = 'constraintBlockHdr' ;
  var titleSpan = document.createElement('span') ; titleSpan.className = 'constraintBlockTitle' ;
  titleSpan.textContent = key || 'New Constraint' ; blockHdr.appendChild(titleSpan) ;
  if (!_modelReadOnly) {
    var rb = document.createElement('button') ; rb.className = 'rmBtn' ; rb.textContent = 'Remove' ;
    rb.onclick = function() { confirmDel('"' + (titleSpan.textContent || 'this constraint') + '"', function() { block.remove() ; }) ; } ;
    blockHdr.appendChild(rb) ;
  }
  block.appendChild(blockHdr) ;

  // Parse key into resPlural + attrName
  var dotIdx = key.indexOf('.') ;
  var initRes = dotIdx !== -1 ? key.substring(0, dotIdx) : key ;
  var initAttr = dotIdx !== -1 ? key.substring(dotIdx+1) : '' ;

  if (_modelReadOnly) {
    // Read-only: show fields as text
    function roRow(lbl, val) {
      var row = document.createElement('div') ; row.className = 'editorField' ;
      var l = document.createElement('label') ; l.textContent = lbl ;
      var s = document.createElement('span') ; s.style.cssText = 'font-size:13px;color:#333;' ;
      s.textContent = val || '\u2014' ; row.appendChild(l) ; row.appendChild(s) ; return row ;
    }
    var pathStr = key || '\u2014' ;
    block.appendChild(roRow('Path:', pathStr)) ;
    var defStr = constraint.default !== undefined ? JSON.stringify(constraint.default) : '' ;
    if (defStr) block.appendChild(roRow('Default:', defStr)) ;
    if (constraint.equals) block.appendChild(roRow('Equals:', constraint.equals)) ;
    var enumArr = Array.isArray(constraint.enum) ? constraint.enum : [] ;
    if (enumArr.length) block.appendChild(roRow('Enum:', enumArr.join(', '))) ;
    return block ;
  }

  // Edit mode: path row with two selects
  var pathRow = document.createElement('div') ; pathRow.className = 'cstrPathRow' ;
  var pathLbl = document.createElement('label') ;
  pathLbl.appendChild(document.createTextNode('Path:')) ;
  var pathReq = document.createElement('span') ; pathReq.textContent = ' *' ; pathReq.style.cssText = 'color:#c00;font-weight:bold;' ;
  pathLbl.appendChild(pathReq) ;
  pathRow.appendChild(pathLbl) ;

  var resSel = document.createElement('select') ; resSel.className = 'cstrResSel editorSelectWrap editorInput' ;
  var resBlank = document.createElement('option') ; resBlank.value = '' ; resBlank.textContent = '-- resource --' ;
  resSel.appendChild(resBlank) ;
  var res = (((_modelData||{}).groups||{})[gk]||{}).resources||{} ;
  Object.keys(res).sort().forEach(function(rk) {
    var o = document.createElement('option') ; o.value = rk ; o.textContent = rk ;
    if (rk === initRes) o.selected = true ;
    resSel.appendChild(o) ;
  }) ;

  var dot = document.createElement('span') ; dot.className = 'cstrPathDot' ; dot.textContent = '.' ;

  var attrSel = document.createElement('select') ; attrSel.className = 'cstrAttrSel editorSelectWrap editorInput' ;
  populateCstrAttrSel(attrSel, gk, initRes, initAttr) ;

  resSel.onchange = function() {
    var newRes = resSel.value ;
    titleSpan.textContent = newRes ? (newRes + '.' + (attrSel.value||'?')) : 'New Constraint' ;
    populateCstrAttrSel(attrSel, gk, newRes, '') ;
  } ;
  attrSel.onchange = function() {
    var r = resSel.value ; var a = attrSel.value ;
    titleSpan.textContent = (r && a) ? (r + '.' + a) : (r ? r + '.?' : 'New Constraint') ;
  } ;

  pathRow.appendChild(resSel) ; pathRow.appendChild(dot) ; pathRow.appendChild(attrSel) ;
  block.appendChild(pathRow) ;

  // Default field
  var defRow = document.createElement('div') ; defRow.className = 'editorField' ;
  var defLbl = document.createElement('label') ; defLbl.textContent = 'Default:' ;
  var defInp = document.createElement('input') ; defInp.type = 'text' ; defInp.className = 'cstrDef editorInput' ;
  defInp.placeholder = 'default value' ;
  defInp.value = constraint.default !== undefined ? JSON.stringify(constraint.default) : '' ;
  defRow.appendChild(defLbl) ; defRow.appendChild(defInp) ;
  block.appendChild(defRow) ;

  // Equals field — dropdown of group scalar attrs
  var eqRow = document.createElement('div') ; eqRow.className = 'editorField' ;
  var eqLbl = document.createElement('label') ; eqLbl.textContent = 'Equals:' ;
  var eqSel = document.createElement('select') ; eqSel.className = 'cstrEqSel editorSelectWrap editorInput' ;
  var eqBlank = document.createElement('option') ; eqBlank.value = '' ; eqBlank.textContent = '-- none --' ;
  eqSel.appendChild(eqBlank) ;
  var grpAttrs = getScalarAttrNames((((_modelData||{}).groups||{})[gk]||{}).attributes||{}) ;
  grpAttrs.forEach(function(n) {
    var o = document.createElement('option') ; o.value = n ; o.textContent = n ;
    if (n === (constraint.equals||'')) o.selected = true ;
    eqSel.appendChild(o) ;
  }) ;
  eqRow.appendChild(eqLbl) ; eqRow.appendChild(eqSel) ;
  block.appendChild(eqRow) ;

  // Enum editor
  var enumSec = document.createElement('div') ; enumSec.className = 'cstrEnumSection' ;
  var enumHdr = document.createElement('div') ; enumHdr.className = 'editorSectionLabel' ;
  enumHdr.textContent = 'Enum' ; enumSec.appendChild(enumHdr) ;
  var enumWrap = document.createElement('div') ; enumWrap.className = 'labelsWrap' ;
  var enumAddBtn = document.createElement('button') ; enumAddBtn.className = 'editorBtn' ;
  enumAddBtn.textContent = '+ Add' ; enumAddBtn.title = 'Add enum value' ;
  enumAddBtn.style.flexShrink = '0' ; enumAddBtn.style.alignSelf = 'flex-start' ;
  var enumRowsDiv = document.createElement('div') ; enumRowsDiv.id = 'cstr_enum_' + idx ;
  enumRowsDiv.style.cssText = 'flex:1;display:flex;flex-direction:column;gap:4px;' ;
  enumAddBtn.onclick = function() { enumRowsDiv.appendChild(makeEnumRow('')) ; } ;
  var enumArr2 = Array.isArray(constraint.enum) ? constraint.enum : [] ;
  enumArr2.forEach(function(v) { enumRowsDiv.appendChild(makeEnumRow(String(v))) ; }) ;
  enumWrap.appendChild(enumAddBtn) ; enumWrap.appendChild(enumRowsDiv) ;
  enumSec.appendChild(enumWrap) ; block.appendChild(enumSec) ;

  return block ;
}

function collectConstraints(containerId) {
  var container = document.getElementById(containerId) ; var constraints = {} ;
  if (!container) return constraints ;
  container.querySelectorAll('.constraintBlock').forEach(function(block) {
    var resSel = block.querySelector('.cstrResSel') ;
    var attrSel = block.querySelector('.cstrAttrSel') ;
    var resVal = resSel ? resSel.value.trim() : '' ;
    var attrVal = attrSel ? attrSel.value.trim() : '' ;
    var key = (resVal && attrVal) ? (resVal + '.' + attrVal) : (block.dataset.origKey || '') ;
    if (!key) return ; // truly new/empty constraint with no path — skip
    var c = {} ;
    var defInp = block.querySelector('.cstrDef') ;
    var defVal = defInp ? defInp.value.trim() : '' ;
    if (defVal !== '') { try { c.default = JSON.parse(defVal) ; } catch(e) { c.default = defVal ; } }
    var eqSel = block.querySelector('.cstrEqSel') ;
    var eq = eqSel ? eqSel.value.trim() : '' ;
    if (eq) c.equals = eq ;
    var enumDiv = document.getElementById('cstr_enum_' + block.dataset.cstrIdx) ;
    if (enumDiv) {
      var vals = [] ;
      enumDiv.querySelectorAll('.labelRow input').forEach(function(inp) {
        var v = inp.value.trim() ; if (v) vals.push(v) ;
      }) ;
      if (vals.length) c.enum = vals ;
    }
    constraints[key] = c ;
  }) ;
  return constraints ;
}

function fv(id) {
  var el = document.getElementById(id) ; if (!el) return '' ;
  return el.value !== undefined ? el.value.trim() : '' ;
}

// ---- Save / Undo / ReadOnly ----

function undoModel() {
  _modelDirty = false ;
  _modelData = deepClone(_modelSrc) ;
  _navPath = [] ; _navSelected = null ; renderEditor() ;
}

function applyReadOnly(container) {
  container.querySelectorAll('input, select, textarea').forEach(function(el) { el.disabled = true ; }) ;
  container.querySelectorAll('.editorBtn:not(.navDrillBtn), .rmBtn, .navItemAdd, .navItemDel').forEach(function(el) { el.style.display = 'none' ; }) ;
}

function saveModel(onSuccess) {
  collectCurrentEditor() ;
  var model = _modelData || {} ;
  var errDiv = document.getElementById('editorError') ;
  if (errDiv) { errDiv.style.display = 'none' ; errDiv.textContent = '' ; }

  // Show blocking overlay while PUT is in flight
  var overlay = document.createElement('div') ; overlay.className = 'savingOverlay' ;
  var box = document.createElement('div') ; box.className = 'savingBox' ;
  var spinner = document.createElement('div') ; spinner.className = 'savingSpinner' ;
  var msg = document.createElement('div') ; msg.textContent = 'Saving\u2026 validating registry' ;
  box.appendChild(spinner) ; box.appendChild(msg) ; overlay.appendChild(box) ;
  document.body.appendChild(overlay) ;
  function removeOverlay() { if (overlay.parentNode) overlay.parentNode.removeChild(overlay) ; }

  fetch(_modelPutURL, {
    method: 'PUT',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(model, null, 2)
  }).then(function(resp) {
    return resp.text().then(function(text) {
      removeOverlay() ;
      if (resp.ok) {
        _modelDirty = false ;
        _modelSrc = deepClone(_modelData) ;
        if (onSuccess) { onSuccess() ; } else { window.location.reload() ; }
      } else { if (errDiv) { errDiv.style.display = 'block' ; errDiv.textContent = 'Error (' + resp.status + '):\n' + text ; } }
    }) ;
  }).catch(function(err) {
    removeOverlay() ;
    if (errDiv) { errDiv.style.display = 'block' ; errDiv.textContent = 'Network error: ' + err.message ; }
  }) ;
}
`
		editBtn = `<div class="viewToggle"><span id="viewToggleJson" class="viewToggleBtn viewToggleBtnActive" onclick="switchView('json')" title="View as JSON">{&thinsp;}</span><span id="viewToggleForm" class="viewToggleBtn" onclick="switchView('view')" title="View as model">&#9776;</span></div><span id="viewToggleEdit" class="viewTogglePencil" onclick="switchView('toggle-edit')" title="Edit model" style="display:none">&#9998;</span>`
		editorDiv = `<div id=modelEditor></div>`
	}

	html := `<html>
<head><meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover"></head>
<style>
  a:visited {
    color: black ;
  }
  form {
    display: inline ;
  }
  html, body { overflow: hidden ; }
  body {
    display: flex ;
    flex-direction: row ;
    flex-wrap: nowrap ;
    justify-content: flex-start ;
    height: 100vh ;
    margin: 0 ;
  }
  #left {
    padding: 0 0 25 0 ;
    background-color: snow ; // #e3e3e3 ; // gainsboro ; // lightsteelblue;
    white-space: nowrap ;
    overflow-y: auto ;
    min-width: fit-content ;
    display: table ;
    border-right: 2px solid lightgray ;
  }
  #right {
    display: flex ;
    flex-direction: column ;
    flex-wrap: nowrap ;
    justify-content: flex-start ;
    width: 100% ;
    overflow: hidden ;
  }
  #url {
    background-color: lightgray;
    border: 0px ;
    display: flex ;
    flex-direction: row ;
    align-items: center ;
    padding: 5px ;
    margin: 0px ;
  }
  #myURL {
    width: 40em ;
  }

  #registry {
    display: flex ;
    align-items: baseline ;
    padding: 5 10 5 5 ;
    background: lightgray ; // white
  }

  #options {
    display: flex ;
    flex-direction: column ;
    padding: 5 0 5 5 ;
  }

  #xRegLogo {
    cursor: pointer ;
    height: 20px ;
    width: 20px ;
    // margin-left: auto ;
    padding-left: 5px ;
  }

  #githubLogo {
    position: fixed ;
    left: 0 ;
    bottom: 0 ;
    cursor: pointer ;
    font-family: courier ;
    font-size: 10 ;
    display: inline-flex ;
    align-items: baseline ;
    background-color: snow ;
  }

  button {
    // margin-left: 5px ;
  }
  #buttonList {
    // margin-top: 10px ;
    // padding-top: 5px ;
  }
  #buttonBar {
    background-color: #e3e3e3 ; // lightsteelblue;
    display: flex ;
    flex-direction: column ;
    align-items: start ;
    padding: 2px ;
  }

  .colorButton, #applyBtn, #details {
    border-radius: 13px ;
    border: 1px solid #407d16 ;
    background: #407d16 ;
    padding: 5 20 6 20 ;
    color: white ;
  }

  #details {
    font-weight: bold ;
    display: inline ;
    margin-top: 10px ;
    margin-bottom: 10px ;
    padding: 1 10 2 10 ;
  }
  #details:hover { background: #c4c4c4 ; color : black ; }
  #details:active { background: #c4c4c4 ; color : black ; }
  #details:focus { background: darkgray ; color : black ; }

  legend {
    margin-bottom: 3 ;
  }

  fieldset {
    border-width: 1 0 1 1 ;
    border-color: black ;
    padding: 0 4 4 4 ;
    background-color: #f2eeee ;
    margin: 0 0 0 2 ;
  }

  #applyBtn {
    font-weight: bold ;
    margin: 5 0 0 0 ;
  }

  #applyBtn:hover { background: #c4c4c4 ; color : black ; }
  #applyBtn:active { background: #c4c4c4 ; color : black ; }
  #applyBtn:focus { background: darkgray ; color : black ; }

  textarea {
    margin-bottom: 10px ;
    min-width: 99% ;
    max-width: 95% ;
  }
  #filters {
    display: block ;
    min-height: 8em ;
    font-size: 12px ;
    font-family: courier ;
    width: 100%
  }
  .filterHelp {
    display: inline ;
    font-size: 12px ;
    font-family: courier ;
  }
  select {
    background: #407d16 ;
    border: 1px solid #407d16 ;
    border-radius: 13px ;
    color: white ;
    font-weight: bold ;
    margin-left: 10px ;
    padding: 2 10 3 10 ;
    align-self: center
  }

  select:hover { background: #c4c4c4 ; color : black ; }
  select:active { background: #c4c4c4 ; color : black ; }
  select:focus { background: darkgray ; color : black ; }

  .myli {
    margin-left: 3px ;
  }

  .options {
    margin-top: 5px ;
    font-size: 13px ;
    font-family: courier ;
  }
  .sortsection {
    display: block ;
    margin: 5px 0px 5px 0px ;
    white-space: nowrap:
  }
  .sortsectioncheckbox {
    font-size: 13px ;
    font-family: courier ;
    display: inline ;
  }
  .inlines {
    display: flex ;
    font-size: 13px ;
    font-family: courier ;
    align-items: center ;
  }

  .inlinehilight {
    background-color : #e1e1e1 ;
  }

  .minicheckdiv {
    margin-left: auto ;
    padding: 0 2 0 5 ;
  }

  .minicheck {
    height: 10px ;
    width: 10px ;
    margin: 0px ;
  }

  .line {
    width: 90% ;
    border-bottom: 1px solid black ;
    margin: 3 0 3 20 ;
  }
  #urlPath {
    background-color: ghostwhite; // lightgray ;
    padding: 3px ;
    font-size: 16px ;
    font-family: courier ;
    border-bottom: 4px solid #e3e3e3 ; // lightsteelblue ;
  }
  #myOutput {
    background-color: ghostwhite;
    border: 0px ;
    padding: 5px ;
    flex: 1 ;
    overflow: auto ;
    white-space: pre ;
    font-family: courier ;
    font-size: 14px ;
    line-height: 16px ; // easier to read this way
  }

  .expandAll {
    cursor: pointer ;
    display: inline-flex ;
    position: fixed ;
    top: 30 ;
    right: 16 ;
  }

  .expandBtn {
    cursor: pointer ;
    display: inline-block ;
    width: 2ch ;
    text-align: center ;
    border: 1px solid darkgrey ;
    border-radius: 5px ;
    background-color: lightgrey ;
    margin-right: 2px ;
    font-family: menlo ;
    font-size: 16px ;
  }

  .spc {
    cursor: default ;
    width: 14px ;
    display: inline ;
    user-select: none ;
    -webkit-user-select: none; /* Safari */
    -ms-user-select: none; /* IE 10 and IE 11 */
    text-align: center ;
  }

  .exp {
    cursor: pointer ;
    width: 1ch ;
    display: inline-block ;
    user-select: none ;
    -webkit-user-select: none; /* Safari */
    -ms-user-select: none; /* IE 10 and IE 11 */
    text-align: center ;

    // background-color: lightsteelblue ; // #e7eff7 ;
    color: black ;
    // font-weight: bold ;
    font-family: menlo ;
    font-size: medium ;
    overflow: hidden ;
    line-height: 10px ;
    max-width: 8px ;
  }

  .hide {
    width:0px ;
    display:inline-block ;
  }

  pre {
    margin: 0px ;
  }
  li {
    white-space: nowrap ;
    list-style-type: disc ;
  }
` + editorCSS + `</style>
<div id=left>
  <div id=registry>
    <a href="javascript:opensrc()" title="https://xRegistry.io" style="text-decoration:none;color:black">
      <svg id=xRegLogo viewBox="0 0 663 800" fill="none">
        <title>https://xRegistry.io</title>
        <g clip-path="url(#clip0_1_142)">
        <path d="M490.6 800H662.2L494.2 461.6C562.2 438.2 648.6 380.5 648.6 238.8C648.6 5.3 413.9 0 413.9 0H3.40002L80.39 155.3H391.8C391.8 155.3 492.3 153.8 492.3 238.9C492.3 323.9 391.8 322.5 391.8 322.5H390.6L316.2 449L490.6 800Z" fill="black"/>
        <path d="M392.7 322.4H281L266.7 346.6L196.4 466.2L111.7 322.4H0L140.5 561.2L0 800H111.7L196.4 656.2L281 800H392.7L252.2 561.2L317.9 449.6L392.7 322.4Z" fill="#0066FF"/>
        </g>
        <defs>
          <clipPath id="clip0_1_142">
            <rect width="662.2" height="800" fill="white"/>
          </clipPath>
        </defs>
      </svg><b>egistry:</b>
    </a>
    <select id=regList onchange="changeRegistry(value)">` + list + `    </select>
  </div>
  <div id=options>
` + roots + `
  <div id=buttonList>
    ` + applyBtn + `
    ` + detailsButton + `
  </div>
  </div> <!-- options -->
</div>  <!-- left -->

<script>

var detailsSwitch = ` + detailsSwitch + `;

function changeRegistry(name) {
  var loc = ""

  if (name == "Default") loc = "/?ui"
  else if (name == "xRegistry.io") {
    loc = "/proxy?host=http://xregistry.io/xreg" ;
  } else if (name == "Proxy ...") {
    proxy = prompt("xRegistry host URL")
    if (proxy != null) {
      if ( !proxy.startsWith("http") ) proxy = "http://" + proxy ;
      loc = "/proxy?host=" + proxy ;
    } else {
      document.getElementById('regList').value = "` + selectedRegistry + `";
      return false ;
    }
  } else loc = "/reg-" + name + "?ui"


  window.location = loc
}

function opensrc(loc) {
  if (loc == null) loc = "https://xregistry.io"
  else if (loc == "commit") {
    loc = "https://github.com/xregistry/server/tree/` + GitCommit + `"
  }
  window.open( loc )
}

function apply() {
  var loc = "` + BuildURLNoUI(info, "") + `/` + strings.Join(info.Parts, "/") + `"

  if (detailsSwitch) loc += "$details"
  loc += "?` + addUI + `"

  ex = document.getElementById("binary")
  if (ex != null && ex.checked) loc += "` + AMP + `binary"

  ex = document.getElementById("collections")
  if (ex != null && ex.checked) loc += "` + AMP + `collections"

  ex = document.getElementById("docview")
  if (ex != null && ex.checked) loc += "` + AMP + `doc"

  var elem = document.getElementById("sortkey")
  if (elem != null && elem.value != "") {
    loc += "` + AMP + `sort=" + elem.value
    elem = document.getElementById("sortdesc")
    if (elem != null && elem.checked) {
      loc += "=desc"
    }
  }

  var elem = document.getElementById("filters")
  if (elem != null) {
    var filters = elem.value
    var lines = filters.split("\n")
    for (var i = 0 ; i < lines.length ; i++ ) {
      if (lines[i] != "") {
        loc += "` + AMP + `filter=" + lines[i]
      }
    }
  }

  for (var i = 0 ; ; i++ ) {
    var box = document.getElementById("inline"+i)
    if (box == null) { break }
    if (box.checked) {
      loc += "` + AMP + `inline=" + box.value
    }
  }

  window.location = loc
}

function toggleExp(elem, exp) {
  if ( !elem ) {
    elem = document.getElementById("expAll")
    exp = (elem.show == true)
    elem.show = !exp
    elem.innerHTML = (exp ? '` + HTML_MIN + `' : '` + HTML_EXP + `')
    for ( var i = 1 ;; i++ ) {
      elem = document.getElementById("s"+i)
      if ( !elem ) return false
      toggleExp(elem, exp)
    }
  }

  var id = elem.id
  var block = document.getElementById(id+'block')
  if ( exp === undefined ) exp = (block.style.display == "none")

  elem.innerHTML = (exp ? "` + HTML_EXP + `" : "` + HTML_MIN + `")

  block.style.display = (exp?"inline":"none")
  document.getElementById(id+'dots').style.display = (exp?"none":"inline")
}

function dokeydown(event) {
  if (event.key == "a" && (event.ctrlKey || event.metaKey)) {
    var tag = event.target ? event.target.tagName : '' ;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return ; // let browser handle it
    event.preventDefault(); // don't bubble event

    // make ctl-a only select the output, not the entire page
    var range = document.createRange()
    range.selectNodeContents(document.getElementById("text"))
    window.getSelection().empty()
    window.getSelection().addRange(range)
  }
}

` + editorJS + `
</script>

<div id=right>
    <!--
    <form id=url onsubmit="go();return false;">
      <div style="margin:0 5 0 10">URL:</div>
      <input id=myURL type=text>
      <button type=submit> Go! </button>
    </form>
    -->
  <div id=urlPath>
    <b>Path:</b> ` + urlPath + `
  </div>
  <div id=myOutput tabindex=0 onkeydown=dokeydown(event)
    ><div id=expandAll class=expandAll>
      ` + editBtn + `
      <span id=expAll class=expandBtn title="Collapse/Expand all" onclick=toggleExp(null,false)>` + HTML_MIN + `</span>
    </div
    ><div id='text'
>` + string(output) + `
    </div> <!-- text -->
    ` + editorDiv + `
  </div> <!-- myOutput -->
</div> <!-- right -->

<div id="githubLogo">
<svg height="20" aria-hidden="true" viewBox="0 0 24 24" width="20" onclick="opensrc('commit')">
  <title>Open commit: ` + GitCommit + `</title>
  <path d="M12.5.75C6.146.75 1 5.896 1 12.25c0 5.089 3.292 9.387 7.863 10.91.575.101.79-.244.79-.546 0-.273-.014-1.178-.014-2.142-2.889.532-3.636-.704-3.866-1.35-.13-.331-.69-1.352-1.18-1.625-.402-.216-.977-.748-.014-.762.906-.014 1.553.834 1.769 1.179 1.035 1.74 2.688 1.25 3.349.948.1-.747.402-1.25.733-1.538-2.559-.287-5.232-1.279-5.232-5.678 0-1.25.445-2.285 1.178-3.09-.115-.288-.517-1.467.115-3.048 0 0 .963-.302 3.163 1.179.92-.259 1.897-.388 2.875-.388.977 0 1.955.13 2.875.388 2.2-1.495 3.162-1.179 3.162-1.179.633 1.581.23 2.76.115 3.048.733.805 1.179 1.825 1.179 3.09 0 4.413-2.688 5.39-5.247 5.678.417.36.776 1.05.776 2.128 0 1.538-.014 2.774-.014 3.162 0 .302.216.662.79.547C20.709 21.637 24 17.324 24 12.25 24 5.896 18.854.75 12.5.75Z"></path>
</svg>` + GitCommit[:min(len(GitCommit), 12)] + `
</div>
` + autoExpand + `
</html>
`

	return []byte(html)
}

func GetRegistryModelInlines(m *Model) []string {
	res := []string{}

	for _, key := range SortedKeys(m.Groups) {
		gm := m.Groups[key]
		res = append(res, gm.Plural)
		for _, inline := range GetGroupModelInlines(gm) {
			res = append(res, gm.Plural+"."+inline)
		}
	}

	sort.Strings(res)

	return res
}

func GetGroupModelInlines(gm *GroupModel) []string {
	res := []string{}

	list := gm.GetResourceList()
	sort.Strings(list)

	for _, key := range list {
		rm := gm.FindResourceModel(key)
		res = append(res, rm.Plural)
		for _, inline := range GetResourceModelInlines(rm) {
			res = append(res, rm.Plural+"."+inline)
		}
	}
	return res
}

func GetResourceModelInlines(rm *ResourceModel) []string {
	res := []string{}

	if rm.GetHasDocument() {
		res = append(res, rm.Singular)
	}

	res = append(res, "meta")

	res = append(res, "versions")
	for _, inline := range GetVersionModelInlines(rm) {
		res = append(res, "versions."+inline)
	}

	return res
}

func GetVersionModelInlines(rm *ResourceModel) []string {
	return []string{rm.Singular}
}
