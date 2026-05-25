package registry

import (
	"bytes"
	"fmt"
	"net/http"
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

	if !pw.Info.SentStatus {
		pw.Info.SentStatus = true
		if pw.Info.StatusCode == 0 {
			pw.Info.StatusCode = http.StatusOK
		}
		pw.Info.OriginalResponse.WriteHeader(pw.Info.StatusCode)
	}

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

	html := `<html>
<style>
  a:visited {
    color: black ;
  }
  form {
    display: inline ;
  }
  body {
    display: flex ;
    flex-direction: row ;
    flex-wrap: nowrap ;
    justify-content: flex-start ;
    height: 100% ;
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
    overflow: auto ;
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
</style>
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
    event.preventDefault(); // don't bubble event

    // make ctl-a only select the output, not the entire page
    var range = document.createRange()
    range.selectNodeContents(document.getElementById("text"))
    window.getSelection().empty()
    window.getSelection().addRange(range)
  }
}

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
    ><div class=expandAll>
      <span id=expAll class=expandBtn title="Collapse/Expand all" onclick=toggleExp(null,false)>` + HTML_MIN + `</span>
    </div
    ><div id='text'
>` + string(output) + `
    </div> <!-- text -->
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
