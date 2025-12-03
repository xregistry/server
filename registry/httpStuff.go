package registry

import (
	"bytes"
	// "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	// "os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

type Server struct {
	Port       int
	HTTPServer *http.Server
}

func NewServer(port int) *Server {
	server := &Server{
		Port: port,
		HTTPServer: &http.Server{
			Addr: fmt.Sprintf(":%d", port),
		},
	}
	server.HTTPServer.Handler = server
	return server
}

func (s *Server) Close() {
	s.HTTPServer.Close()
}

func (s *Server) Start() *Server {
	go s.Serve()
	/*
		for {
			_, err := http.Get(fmt.Sprintf("http://localhost:%d", s.Port))
			if err == nil || !strings.Contains(err.Error(), "refused") {
				break
			}
		}
	*/
	return s
}

func (s *Server) Serve() {
	log.VPrintf(1, "Listening on %d", s.Port)
	err := s.HTTPServer.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Printf("Serve: %s", err)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var info *RequestInfo
	var tx *Tx

	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("Panic: %s", rec)
			// ShowStack() // Down via NewXRError below now

			// If info isn't defined yet
			if info == nil {
				info = NewRequestInfo(w, r)
			}

			xErr := NewXRError("server_error", "/"+info.OriginalPath).
				SetDetail("An internal error occurred, contact the admin.")
			HTTPWriteError(info, xErr)
		}

		// As of now we should never have more than one active Tx during
		// testing
		/*
			if (os.Getenv("TESTING") != "") && tx != nil {
				l := len(TXs)
				if (tx.tx == nil && l > 0) || (tx.tx != nil && l > 1) {
					log.Printf(">End of HTTP Request")
					log.Printf("len(TXs): %d", l)
					log.Printf("tx.tx: %p", tx.tx)
					DumpTXs()

					log.Printf("Info: %s", ToJSON(info))
					log.Printf("<Exit http req")

					panic("nested Txs")
				}
			}
		*/

		// Explicit Commit() is required, else we'll always rollback
		tx.Rollback()
	}()

	saveVerbose := log.GetVerbose()
	if tmp := r.URL.Query().Get("verbose"); tmp != "" {
		if v, err := strconv.Atoi(tmp); err == nil {
			log.SetVerbose(v)
		}
		defer log.SetVerbose(saveVerbose)
	}

	log.VPrintf(2, "%s %s", r.Method, r.URL)

	if r.URL.Path == "/proxy" {
		HTTPProxy(w, r)
		return
	}

	tx, xErr := NewTx()
	if xErr != nil {
		log.Printf("Error talking to the DB creating new Tx: %s",
			xErr.GetTitle())

		// Special one off - info isn't defined yet
		if info == nil {
			info = NewRequestInfo(w, r)
		}

		HTTPWriteError(info, xErr)

		return
	}

	info, xErr = ParseRequest(tx, w, r)
	tx.RequestInfo = info

	if xErr != nil {
		HTTPWriteError(info, xErr)
		return
	}

	defer func() {
		// If we haven't written anything, this will force the HTTP status code
		// to be written and not default to 200
		info.HTTPWriter.Done()
	}()

	if r.URL.Query().Has("ui") { // Wrap in html page
		info.HTTPWriter = NewPageWriter(info)
	}

	if r.URL.Query().Has("html") || r.URL.Query().Has("noprops") { //HTMLify it
		info.HTTPWriter = NewBufferedWriter(info)
	}

	if sv := info.GetFlag("specversion"); sv != "" {
		if !info.Registry.Capabilities.SpecVersionEnabled(sv) {
			xErr = NewXRError("unsupported_specversion",
				"/"+info.OriginalPath,
				"specversion="+sv,
				"list="+
					strings.Join(info.Registry.Capabilities.SpecVersions, ","))
		}
	}

	if xErr == nil {
		// These should only return an error if they didn't already
		// send a response back to the client.
		switch r.Method {
		case "GET":
			xErr = HTTPGet(info)
		case "PUT":
			xErr = HTTPPutPost(info)
		case "POST":
			xErr = HTTPPutPost(info)
		case "PATCH":
			xErr = HTTPPutPost(info)
		case "DELETE":
			xErr = HTTPDelete(info)
		default:
			xErr = NewXRError("action_not_supported", "/"+info.OriginalPath,
				"action="+r.Method)
		}
	}

	Must(tx.Conditional(xErr))

	if xErr != nil {
		HTTPWriteError(info, xErr)
	}
}

type HTTPWriter interface {
	Write([]byte) (int, error)
	AddHeader(string, string)
	Done()
}

var _ HTTPWriter = &DefaultWriter{}
var _ HTTPWriter = &BufferedWriter{}
var _ HTTPWriter = &DiscardWriter{}
var _ HTTPWriter = &PageWriter{}

func DefaultHTTPWriter(info *RequestInfo) HTTPWriter {
	return &DefaultWriter{
		Info: info,
	}
}

type DefaultWriter struct {
	Info *RequestInfo
}

func (dw *DefaultWriter) Write(b []byte) (int, error) {
	if !dw.Info.SentStatus {
		dw.Info.SentStatus = true
		if dw.Info.StatusCode == 0 {
			dw.Info.StatusCode = http.StatusOK
		}
		dw.Info.OriginalResponse.WriteHeader(dw.Info.StatusCode)
	}
	return dw.Info.OriginalResponse.Write(b)
}

func (dw *DefaultWriter) AddHeader(name, value string) {
	dw.Info.OriginalResponse.Header()[name] = []string{value}
}

func (dw *DefaultWriter) Done() {
	dw.Write(nil)
}

type BufferedWriter struct {
	Info      *RequestInfo
	OldWriter HTTPWriter
	Headers   *map[string]string
	Buffer    *bytes.Buffer
}

func NewBufferedWriter(info *RequestInfo) *BufferedWriter {
	return &BufferedWriter{
		Info:      info,
		OldWriter: info.HTTPWriter,
		Headers:   &map[string]string{},
		Buffer:    &bytes.Buffer{},
	}
}

func (bw *BufferedWriter) Write(b []byte) (int, error) {
	return bw.Buffer.Write(b)
}

func (bw *BufferedWriter) AddHeader(name, value string) {
	(*bw.Headers)[name] = value
}

func (bw *BufferedWriter) Done() {
	req := bw.Info.OriginalRequest
	if req.URL.Query().Has("html") {
		// Override content-type
		bw.AddHeader("Content-Type", "text/html")
	}

	for k, v := range *bw.Headers {
		bw.OldWriter.AddHeader(k, v)
	}

	buf := bw.Buffer.Bytes()
	if req.URL.Query().Has("html") {
		bw.OldWriter.Write([]byte("<pre>\n"))
		buf = HTMLify(req, buf)
	}
	bw.OldWriter.Write(buf)
}

type DiscardWriter struct{}

func (dw *DiscardWriter) Write(b []byte) (int, error)  { return len(b), nil }
func (dw *DiscardWriter) AddHeader(name, value string) {}
func (dw *DiscardWriter) Done()                        {}

var DefaultDiscardWriter = &DiscardWriter{}

type PageWriter struct {
	Info      *RequestInfo
	OldWriter HTTPWriter
	Headers   *map[string]string
	Buffer    *bytes.Buffer
}

func NewPageWriter(info *RequestInfo) *PageWriter {
	return &PageWriter{
		Info:      info,
		OldWriter: info.HTTPWriter,
		Headers:   &map[string]string{},
		Buffer:    &bytes.Buffer{},
	}
}

func (pw *PageWriter) Write(b []byte) (int, error) {
	return pw.Buffer.Write(b)
}

func (pw *PageWriter) AddHeader(name, value string) {
	(*pw.Headers)[name] = value
}

func (pw *PageWriter) Done() {
	pw.AddHeader("Content-Type", "text/html")

	for k, v := range *pw.Headers {
		pw.OldWriter.AddHeader(k, v)
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

		if r.u != "" && !info.APIEnabled("/"+r.u) {
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
		filters = "<b>Filters:</b>\n    <textarea id=filters>" +
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

func HTTPGETCapabilities(info *RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	buf := []byte(nil)
	var err error

	cap := info.Registry.Capabilities
	capStr := info.Registry.GetAsString("#capabilities")
	if capStr != "" {
		var xErr *XRError
		cap, xErr = ParseCapabilitiesJSON([]byte(capStr))
		Must(xErr)
	}

	buf, err = json.MarshalIndent(cap, "", "  ")
	if err != nil {
		return NewXRError("server_error", "/").
			SetDetailf("Error parsing capabilities: %s", err.Error())
	}

	info.AddHeader("Content-Type", "application/json")
	info.Write(buf)
	info.Write([]byte("\n"))
	return nil
}

func HTTPGETCapabilitiesOffered(info *RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	buf := []byte(nil)
	var err error

	offered := GetOffered()
	buf, err = json.MarshalIndent(offered, "", "  ")
	if err != nil {
		return NewXRError("server_error", "/capabilitiesoffered").
			SetDetailf("Error parsing capabilitiesoffered: %s", err.Error())
	}

	info.AddHeader("Content-Type", "application/json")
	info.Write(buf)
	info.Write([]byte("\n"))
	return nil
}

func HTTPGETModel(info *RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	format := info.GetFlag("schema")
	if format == "" {
		format = "xRegistry-json"
	}

	model := info.Registry.Model
	if model == nil {
		model = &Model{}
	}

	buf, xErr := model.SerializeForUser()
	if xErr != nil {
		return NewXRError("server_error", "/"+info.OriginalPath).
			SetDetail(xErr.GetTitle())
	}

	info.AddHeader("Content-Type", "application/json")
	info.Write(buf)
	info.Write([]byte("\n"))
	return nil
}

func HTTPGETModelSource(info *RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	model := info.Registry.Model
	if model == nil {
		model = &Model{}
	}
	modelSrc := model.Source
	if modelSrc == "" {
		modelSrc = "{}"
	}

	buf, err := PrettyPrintJSON([]byte(modelSrc), "", "  ")
	if err != nil {
		return NewXRError("server_error", "/modelsource").
			SetDetailf("Error parsing modelsource: %s", err.Error())
	}

	info.AddHeader("Content-Type", "application/json")
	info.Write(buf)
	info.Write([]byte("\n"))
	return nil
}

func HTTPGETContent(info *RequestInfo) *XRError {
	log.VPrintf(3, ">Enter: HTTPGetContent")
	defer log.VPrintf(3, "<Exit: HTTPGetContent")

	query := `
SELECT
  RegSID,Type,Plural,Singular,eSID,UID,PropName,PropValue,PropType,Path,Abstract
FROM FullTree WHERE RegSID=? AND `
	args := []any{info.Registry.DbSID}

	path := strings.Join(info.Parts, "/")

	// TODO consider excluding the META object from the query instead of
	// dropping it via the if-statement below in the versioncount logic
	if info.VersionUID == "" {
		query += `(Path=? OR Path LIKE ?)`
		args = append(args, path, path+"/%")
	} else {
		query += `Path=?`
		args = append(args, path)
	}
	query += " ORDER BY Path"

	log.VPrintf(3, "Query:\n%s", SubQuery(query, args))

	results := Query(info.tx, query, args...)
	defer results.Close()

	entity, xErr := readNextEntity(info.tx, results, FOR_READ)
	log.VPrintf(3, "Entity: %#v", entity)
	if entity == nil {
		if xErr != nil {
			log.Printf("Error loading entity: %s", xErr)
			return NewXRError("server_error", "/"+path).SetDetailf(
				"error loading entity: %s", xErr.GetTitle())
		} else {
			return NewXRError("not_found", "/"+path)
		}
	}

	var version *Entity
	versionsCount := 0
	if info.VersionUID == "" {
		// We're on a Resource, so go find the default Version and count
		// how many versions there are for the VersionsCount attribute
		group, err := info.Registry.FindGroup(info.GroupType, info.GroupUID, false, FOR_READ)
		if err != nil {
			return NewXRError("server_error",
				info.GetParts(2)).SetDetailf("Error finding Group: %s", err)
		}
		if group == nil {
			return NewXRError("not_found", info.GetParts(2))
		}

		resource, err := group.FindResource(info.ResourceType,
			info.ResourceUID, false, FOR_READ)
		if err != nil {
			return NewXRError("server_error", info.GetParts(4)).
				SetDetailf("Error finding Resource: %s", err)
		}
		if resource == nil {
			return NewXRError("not_found", info.GetParts(4))
		}
		meta, err := resource.FindMeta(false, FOR_READ)
		PanicIf(err != nil, "%s", err)

		vID := meta.GetAsString("defaultversionid")
		for {
			v, err := readNextEntity(info.tx, results, FOR_READ)

			if v != nil && v.Type == ENTITY_META {
				// Skip the "meta" subobject, but first grab the
				// "defaultversionid if we don't already have it,
				// which should only be true in the xref case
				if vID == "" {
					vID = v.GetAsString("defaultversionid")
				}
				continue
			}

			if v == nil && version == nil {
				return NewXRError("server_error", resource.XID+"/versions/"+vID).
					SetDetailf("Error finding Version: %s", err)
			}
			if v == nil {
				break
			}
			versionsCount++
			if v.UID == vID {
				version = v
			}
		}
	} else {
		version = entity
	}

	log.VPrintf(3, "Version: %#v", version)

	headerIt := func(e *Entity, info *RequestInfo, key string, val any, attr *Attribute) *XRError {
		if key[0] == '#' {
			return nil
		}

		if attr.internals != nil && attr.internals.neverSerialize {
			return nil
		}

		if attr.Type == MAP && IsScalar(attr.Item.Type) {
			for name, value := range val.(map[string]any) {
				info.AddHeader("xRegistry-"+key+"-"+name,
					fmt.Sprintf("%v", value))
			}
			return nil
		}

		if !IsScalar(attr.Type) {
			return nil
		}

		var headerName string
		if attr.internals != nil && attr.internals.httpHeader != "" {
			headerName = attr.internals.httpHeader
		} else {
			headerName = "xRegistry-" + key
		}

		str := fmt.Sprintf("%v", val)
		info.AddHeader(headerName, str)

		return nil
	}

	xErr = entity.SerializeProps(info, headerIt)
	if xErr != nil {
		panic(xErr)
	}

	if info.VersionUID == "" {
		info.AddHeader("xRegistry-versionscount",
			fmt.Sprintf("%d", versionsCount))
		info.AddHeader("xRegistry-versionsurl",
			info.BaseURL+"/"+entity.Path+"/versions")
	}
	info.AddHeader("Content-Location", info.BaseURL+"/"+version.Path)
	info.AddHeader("Content-Disposition", info.ResourceUID)

	url := ""
	singular := info.ResourceModel.Singular
	if url = entity.GetAsString(singular + "url"); url != "" {
		info.AddHeader("xRegistry-"+singular+"url", url)

		if info.StatusCode == 0 {
			// If we set it during a PUT/POST, don't override the 201
			info.StatusCode = http.StatusSeeOther
			info.AddHeader("Location", url)
		}
		/*
			http.Redirect(info.OriginalResponse, "/"+info.OriginalPath, url,
				http.StatusSeeOther)
		*/
		return nil
	}

	url = entity.GetAsString(singular + "proxyurl")

	log.VPrintf(3, singular+"proxyurl: %s", url)
	if url != "" {
		// Just act as a proxy and copy the remote resource as our response
		resp, err := http.Get(url)
		if err != nil {
			return NewXRError("parsing_response", "/"+info.OriginalPath,
				"error_detail="+err.Error())
		}
		if resp.StatusCode/100 != 2 {
			info.StatusCode = resp.StatusCode
			// return fmt.Error f("Remote error")
			// Let the body of the response be our body, below
		}

		// Copy all HTTP headers
		for header, value := range resp.Header {
			info.AddHeader(header, strings.Join(value, ","))
		}

		// Now copy the body
		_, err = io.Copy(info, resp.Body)
		if err != nil {
			return NewXRError("parsing_response", "/"+info.OriginalPath,
				"error_detail="+err.Error())
		}
		return nil
	}

	buf := version.Get(singular)
	if buf == nil {
		// No data so just return
		/*
			if info.StatusCode == 0 {
				info.StatusCode = http.StatusNoContent
			}
		*/
		return nil
	}

	info.Write(buf.([]byte))

	return nil
}

func HTTPGet(info *RequestInfo) *XRError {
	log.VPrintf(3, ">Enter: HTTPGet(%s)", info.What)
	defer log.VPrintf(3, "<Exit: HTTPGet(%s)", info.What)

	info.Root = strings.Trim(info.Root, "/")

	if info.RootPath == "model" {
		if !info.APIEnabled("/model") {
			return NewXRError("api_not_found", "/model")
		}
		return HTTPGETModel(info)
	}

	if info.RootPath == "modelsource" {
		if !info.APIEnabled("/modelsource") {
			return NewXRError("api_not_found", "/modelsource")
		}
		return HTTPGETModelSource(info)
	}

	if info.RootPath == "capabilities" {
		if !info.APIEnabled("/capabilities") {
			return NewXRError("api_not_found", "/capabilities")
		}
		return HTTPGETCapabilities(info)
	}

	if info.RootPath == "capabilitiesoffered" {
		if !info.APIEnabled("/capabilitiesoffered") {
			return NewXRError("api_not_found", "/capabilitiesoffered")
		}
		return HTTPGETCapabilitiesOffered(info)
	}

	if info.RootPath == "export" {
		if !info.APIEnabled("/export") {
			return NewXRError("api_not_found", "/export")
		}
		return SerializeQuery(info, nil, "Registry", info.Filters)
	}

	// 'metaInBody' tells us whether xReg metadata should be in the http
	// response body or not (meaning, the hasDoc doc)
	metaInBody := (info.ResourceModel == nil) ||
		(info.ResourceModel.GetHasDocument() == false || info.ShowDetails ||
			info.DoDocView() ||
			(len(info.Parts) == 5 && info.Parts[4] == "meta"))

	// Return the Resource's document
	if info.What == "Entity" && info.ResourceUID != "" && !metaInBody {
		return HTTPGETContent(info)
	}

	// Serialize the xReg metadata
	resPaths := map[string][]string{
		"": []string{strings.Join(info.Parts, "/")},
	}
	return SerializeQuery(info, resPaths, info.What, info.Filters)
}

func SerializeQuery(info *RequestInfo, resPaths map[string][]string,
	what string, filters [][]*FilterExpr) *XRError {

	// Make sure everything is ok before we send back the results
	info.tx.Validate(info)

	// resPaths is used to group the items we want to return. In most cases
	// the items will all be part of one group where that group name doesn't
	// need to be returned - e.g. POST /schemagroup where the response will
	// just be a map of gIDs.
	// However, there are cases where we want to return multiple groupings
	// and each group has a different grouping name. For example:
	// POST / +   { "schemagroups": { "sg1"...}, "messagegroups": { "mg1"...}}
	// In this case the "paths" within each group are all processed as a
	// single query but the normal jwWriter stuff won't show the parent
	// group (schemagroups) because to do so would mean to also so the
	// attributes at that level (meaning the Registry attrs in this case).
	// To avoid this we pass in resPaths which is a map of groupings for
	// each "paths" (IDs) we want to serialize.
	// Each key in the map becomes the grouping key/name.
	// When the key is "" then there shouldn't be any other keys and we
	// won't wrapper it at all.
	// So, "" is for things like:  POST .../rID/versions + map[vID]{version}
	// And "xxx" is for things like POST /  + map[GROUPS]map[gID]{group}
	if resPaths == nil {
		resPaths = map[string][]string{"": nil}
	}

	start := time.Now()

	defer func() {
		if log.GetVerbose() > 3 {
			diff := time.Now().Sub(start).Truncate(time.Millisecond)
			log.Printf("  Total Time: %s", diff)
		}
	}()

	if info.RootPath == "export" && len(info.Inlines) == 0 {
		info.AddInline("*")
		info.AddInline("capabilities")
		info.AddInline("modelsource")
	}

	info.AddHeader("Content-Type", "application/json")
	var jw *JsonWriter
	hasData := false
	keys := SortedKeys(resPaths)
	for i, key := range keys {
		paths, ok := resPaths[key]
		PanicIf(!ok, "can't find %q", key)

		var xErr *XRError
		var results *Result

		// "!" is special - it means skip the query and just produce: {}
		if len(paths) != 1 || paths[0] != "!" {
			query, args, err := GenerateQuery(info.Registry, what, paths,
				filters, info.DoDocView(), info.SortKey)
			if err != nil {
				return err
			}
			results = Query(info.tx, query, args...)
			defer results.Close()

			if log.GetVerbose() > 3 {
				log.Printf("SerializeQuery: %s", SubQuery(query, args))
				diff := time.Now().Sub(start).Truncate(time.Millisecond)
				log.Printf("  Query: # results: %d (time: %s)",
					len(results.AllRows), diff)
			}
		}

		jw = NewJsonWriter(info, results)

		jw.NextEntity()

		// Collections will need to print the {}, so don't error for them
		if what != "Coll" {
			if jw.Entity == nil {
				// Special case, if the URL is ../rID/versions/vID?doc then
				// check to see if Resource has xref set, if so then the error
				// is 400, not 404
				if info.VersionUID != "" && info.DoDocView() {
					path := strings.Join(info.Parts[:len(info.Parts)-2], "/")
					path += "/meta"
					entity, err := RawEntityFromPath(info.tx,
						info.Registry.DbSID, path, false, FOR_READ)
					if err != nil {
						return err
					}

					// Assume that if we cant' find the Resource's meta object
					// then the Resource doesn't exist, so a 404 really is the
					// best response in those cases, so skip the 400
					if entity != nil && !IsNil(entity.Object["xref"]) {
						return NewXRError("cannot_doc_xref", "/"+info.OriginalPath)
					}
				}

				return NewXRError("not_found", info.GetParts(0))
			}
		}

		// Special case, if we're doing a collection, let's make sure we didn't
		// get an empty result due to it's parent not even existing - for
		// example the user used the wrong case (or even name) in the parent's
		// Path
		if what == "Coll" && jw.Entity == nil && len(info.Parts) > 2 {
			path := strings.Join(info.Parts[:len(info.Parts)-1], "/")
			entity, xErr := RawEntityFromPath(info.tx, info.Registry.DbSID,
				path, false, FOR_READ)
			if xErr != nil {
				return xErr
			}
			if IsNil(entity) {
				return NewXRError("not_found", "/"+path)
			}
		}

		// GROUPS/GID/RESOURCES/RID/versions
		// Another special case .../rID/versions?doc when rID has xref
		if jw.Entity == nil && info.DoDocView() && len(info.Parts) == 5 &&
			info.Parts[4] == "versions" {

			// Should be our case since "versions" can never be empty except
			// when xref is set. If this is not longer true then we'll need to
			// check this Resource's xref to see if it's set.
			// Can copy the RawEntityFromPath... stuff above
			return NewXRError("cannot_doc_xref", "/"+info.OriginalPath)
		}

		// Only do this if we're adding the extra grouping wrapper
		if key != "" {
			// Cause the jwWriter to indent since we're adding a wrapper
			jw.indent = "  "
			if i == 0 {
				jw.Print("{\n")
			}
			jw.Printf("  %q: ", key)
		}

		if what == "Coll" {
			_, xErr = jw.WriteCollection()
		} else {
			xErr = jw.WriteEntity()
		}

		if xErr != nil {
			return xErr
		}

		if jw.hasData {
			hasData = true
		}

		// Only do this if we're adding the extra grouping wrapper
		if key != "" {
			if i+1 < len(keys) {
				jw.Print(",\n")
			} else {
				jw.Print("\n}")
			}
		}
	}

	if hasData {
		// Add a tailing \n if there's any data, else skip it
		jw.Print("\n")
	}

	return nil
}

var attrHeaders = map[string]*Attribute{}

func init() {
	// Load-up the attributes that have custom http header names
	for _, attr := range OrderedSpecProps {
		if attr.internals != nil && attr.internals.httpHeader != "" {
			attrHeaders[strings.ToLower(attr.internals.httpHeader)] = attr
		}
	}
}

func HTTPPutPost(info *RequestInfo) *XRError {
	method := info.OriginalRequest.Method
	isNew := false
	paths := ([]string)(nil)
	what := "Entity"
	numParts := len(info.Parts)

	metaInBody := (info.ResourceModel == nil) ||
		(info.ResourceModel.GetHasDocument() == false || info.ShowDetails ||
			(numParts == 5 && info.Parts[4] == "meta"))

	log.VPrintf(3, "HTTPPutPost: %s %s", method, info.OriginalPath)

	info.Root = strings.Trim(info.Root, "/")

	// Capabilities has its own special func
	if info.RootPath == "capabilities" {
		if !info.APIEnabled("/capabilities") {
			return NewXRError("api_not_found", "/capabilities")
		}
		return HTTPPUTCapabilities(info)
	}

	if info.RootPath == "model" {
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action="+method).
			SetDetail("Use \"/modelsource\" instead of \"/model\".")
	}

	// The model has its own special func
	if info.RootPath == "modelsource" {
		if !info.APIEnabled("/modelsource") {
			return NewXRError("api_not_found", "/modelsource")
		}
		return HTTPPUTModelSource(info)
	}

	// Load-up the body
	// //////////////////////////////////////////////////////
	body, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		return NewXRError("parsing_data", "/"+info.OriginalPath,
			"error_detail="+err.Error())
	}
	if len(body) == 0 {
		body = nil
	}

	// Check for some obvious high-level bad states up-front
	// //////////////////////////////////////////////////////
	if info.What == "Coll" && method == "PUT" {
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action="+method).SetDetail("PUT not allowed on collections.")
	}

	if numParts >= 5 && info.Parts[4] == "meta" && method == "POST" {
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action="+method).SetDetail("POST not allowed on a 'meta'.")
	}

	if numParts == 6 && method == "POST" {
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action="+method).
			SetDetail("POST not allowed on a version.")
	}

	if (numParts == 4 || numParts == 6) && !metaInBody && method == "PATCH" {
		return NewXRError("details_required", "/"+info.OriginalPath).
			SetDetail("PATCH is not allowed on Resource documents.")
	}

	// Ok, now start to deal with the incoming request
	//////////////////////////////////////////////////

	// Get the incoming Object either from the body or from xRegistry headers
	IncomingObj, xErr := ExtractIncomingObject(info, body)
	if xErr != nil {
		return xErr
	}

	// Walk the PATH and process things
	///////////////////////////////////

	// URL: /
	// ////////////////////////////////////////////////////////////////
	if numParts == 0 {
		// PUT /     + body:Registry
		// PATCH /   + body:Registry
		// POST /    + body:map[GROUPS]map[id]Group

		if method == "PUT" || method == "PATCH" {
			addType := ADD_UPDATE
			if method == "PATCH" {
				addType = ADD_PATCH
			}
			xErr = info.Registry.Update(IncomingObj, addType)
			if xErr != nil {
				return xErr
			}

			// Return HTTP GET of Registry root
			resPaths := map[string][]string{"": []string{""}}
			return SerializeQuery(info, resPaths, "Registry", info.Filters)
		}

		// Must be POST /    + body:map[GROUPS]map[id]Group

		// Error on anything but a group type
		for key, _ := range IncomingObj {
			if info.Registry.Model.FindGroupModel(key) == nil {
				return NewXRError("groups_only", "/"+info.OriginalPath,
					"name="+key)
			}
		}

		objMap, xErr := IncomingObj2Map(IncomingObj)
		if xErr != nil {
			return xErr.SetSubject(info.GetParts(0)).SetTitle(
				"body must be a map of Group types")
		}

		resPaths := map[string][]string{}
		for gType, gAny := range objMap {
			// Should be caught above, but just in case
			if info.Registry.Model.Groups[gType] == nil {
				return NewXRError("not_found", "/"+gType).
					SetDetailf("Unknown Group type: %s.", gType)
			}

			gMap, xErr := IncomingObj2Map(gAny)
			if xErr != nil {
				return xErr.SetSubject(info.GetParts(0))
			}

			for id, obj := range gMap {
				g, _, xErr := info.Registry.UpsertGroupWithObject(gType,
					id, obj, ADD_UPDATE)
				if xErr != nil {
					return xErr
				}
				resPaths[gType] = append(resPaths[gType], g.Path)
			}

			if len(resPaths[gType]) == 0 {
				// Force an empty collection to be returned
				resPaths[gType] = []string{"!"}
			}
		}

		// Special case - if req is {} then make response {}
		if len(objMap) == 0 {
			resPaths = map[string][]string{"": []string{"!"}}
		}

		// Return HTTP GET of Groups created or updated
		return SerializeQuery(info, resPaths, "Coll", info.Filters)
	}

	// URL: /GROUPs[/gID]...
	// ////////////////////////////////////////////////////////////////
	group := (*Group)(nil)
	groupUID := info.GroupUID

	if numParts == 1 {
		// POST /GROUPs + body:map[id]Group

		objMap, xErr := IncomingObj2Map(IncomingObj)
		if xErr != nil {
			return xErr.SetSubject(info.GetParts(0))
		}

		addType := ADD_UPSERT
		if method == "PATCH" {
			addType = ADD_PATCH
		}

		for id, obj := range objMap {
			g, _, xErr := info.Registry.UpsertGroupWithObject(info.GroupType,
				id, obj, addType)
			if xErr != nil {
				return xErr
			}
			paths = append(paths, g.Path)
		}

		if len(paths) == 0 {
			paths = []string{"!"} // Force an empty collection to be returned
		}

		// Return HTTP GET of Groups created or updated
		resPaths := map[string][]string{"": paths}
		return SerializeQuery(info, resPaths, "Coll", info.Filters)
	}

	if numParts == 2 {
		// PUT    /GROUPs/gID  + body: {group}
		// PATCH  /GROUPs/gID  + body: {group}
		// POST   /GROUPs/gID  + body: map[rType]map[rID]{resource}

		if method == "PUT" || method == "PATCH" {
			addType := ADD_UPSERT
			if method == "PATCH" {
				addType = ADD_PATCH
			}

			group, isNew, xErr := info.Registry.UpsertGroupWithObject(info.GroupType,
				info.GroupUID, IncomingObj, addType)
			if xErr != nil {
				return xErr
			}

			if isNew { // 201, else let it default to 200
				info.AddHeader("Location", info.BaseURL+"/"+group.Path)
				info.StatusCode = http.StatusCreated
			}

			// Return HTTP GET of Group
			resPaths := map[string][]string{"": []string{group.Path}}
			return SerializeQuery(info, resPaths, "Entity", info.Filters)
		}

		// Must be POST /GROUPs/gID + body: map[rType]map[rID]{resource}
		objMap, xErr := IncomingObj2Map(IncomingObj)
		if xErr != nil {
			return xErr.SetSubject(info.GetParts(2)).SetTitle(
				fmt.Sprintf("body must be a map of Resource types"))
		}

		group, _, xErr = info.Registry.UpsertGroup(info.GroupType, groupUID)
		if xErr != nil {
			return xErr
		}

		resPaths := map[string][]string{}
		for rType, rAny := range objMap {
			if info.GroupModel.FindResourceModel(rType) == nil {
				return NewXRError("not_found", info.GetParts(2)).
					SetDetailf("Unknown Resource type: %s.", rType)
			}

			rMap, xErr := IncomingObj2Map(rAny)
			if xErr != nil {
				return xErr.SetSubject(info.GetParts(0))
			}

			for id, obj := range rMap {
				r, _, xErr := group.UpsertResourceWithObject(rType,
					id, "", obj, ADD_UPDATE, false)
				if xErr != nil {
					return xErr
				}
				resPaths[rType] = append(resPaths[rType], r.Path)
			}

			if len(resPaths[rType]) == 0 {
				// Force an empty collection to be returned
				resPaths[rType] = []string{"!"}
			}

		}

		// Special case - if req is {} then make response {}
		if len(objMap) == 0 {
			resPaths = map[string][]string{"": []string{"!"}}
		}

		// Return HTTP GET of Resources created or updated
		return SerializeQuery(info, resPaths, "Coll", info.Filters)
	}

	// Must be PUT/POST /GROUPs/gID/...

	// This will either find or create an empty Group as needed
	group, _, xErr = info.Registry.UpsertGroup(info.GroupType, groupUID)
	if xErr != nil {
		return xErr
	}

	// URL: /GROUPs/gID/RESOURCEs...
	// ////////////////////////////////////////////////////////////////

	// Some global vars
	resource := (*Resource)(nil)
	version := (*Version)(nil)
	resourceUID := info.ResourceUID
	versionUID := info.VersionUID

	// Do Resources and Versions at the same time
	// URL: /GROUPs/gID/RESOURCEs
	// URL: /GROUPs/gID/RESOURCEs/rID
	// URL: /GROUPs/gID/RESOURCEs/rID/versions[/vID]
	// ////////////////////////////////////////////////////////////////

	// If there isn't an explicit "return" then this assumes we're left with
	// a version and will return that back to the client

	if numParts == 3 {
		// POST GROUPs/gID/RESOURCEs + body:map[id]Resource

		objMap, xErr := IncomingObj2Map(IncomingObj)
		if xErr != nil {
			return xErr.SetSubject(info.GetParts(0))
		}

		// For each Resource in the map, upsert it and add it's path to result
		addType := ADD_UPSERT
		if method == "PATCH" {
			addType = ADD_PATCH
		}

		for id, obj := range objMap {
			r, _, xErr := group.UpsertResourceWithObject(info.ResourceType,
				id, "", obj, addType, false)
			if xErr != nil {
				return xErr
			}
			paths = append(paths, r.Path)
		}

		if len(paths) == 0 {
			paths = []string{"!"} // Force an empty collection to be returned
		}

		// Return HTTP GET of Resources created or modified
		resPaths := map[string][]string{"": paths}
		return SerializeQuery(info, resPaths, "Coll", info.Filters)
	}

	if numParts > 3 {
		// GROUPs/gID/RESOURCEs/rID...

		resource, xErr = group.FindResource(info.ResourceType, resourceUID,
			false, FOR_READ)
		if xErr != nil {
			return xErr
		}
	}

	if numParts == 4 && (method == "PUT" || method == "PATCH") {
		// PUT GROUPs/gID/RESOURCEs/rID [$details]

		propsID := "" // RESOURCEid
		if v, ok := IncomingObj[info.ResourceModel.Singular+"id"]; ok {
			if reflect.ValueOf(v).Kind() == reflect.String {
				propsID = NotNilString(&v)
			}
		}

		if propsID != "" && propsID != resourceUID {
			return NewXRError("mismatched_id", "/"+info.OriginalPath,
				"singular="+info.ResourceModel.Singular,
				"invalid_id="+propsID,
				"expected_id="+resourceUID)
		}

		if resource != nil {
			// version, xErr = resource.GetDefault(FOR_WRITE)

			// ID needs to be the version's ID, not the Resources
			// IncomingObj["id"] = version.UID

			// Create a new Resource and it's first/only/default Version
			addType := ADD_UPSERT
			if method == "PATCH" || !metaInBody {
				addType = ADD_PATCH
			}
			resource, _, xErr = group.UpsertResourceWithObject(
				info.ResourceType, resourceUID, "" /*versionUID*/, IncomingObj,
				addType, false)
			if xErr != nil {
				return xErr
			}

			version, xErr = resource.GetDefault(FOR_WRITE)
		} else {
			// Upsert resource's default version
			delete(IncomingObj, info.ResourceModel.Singular+"id") // ID is the Resource's delete it
			addType := ADD_UPSERT
			if method == "PATCH" {
				addType = ADD_PATCH
			}
			resource, isNew, xErr = group.UpsertResourceWithObject(
				info.ResourceType, resourceUID, "" /*versionUID*/, IncomingObj,
				addType, false)
			if xErr != nil {
				return xErr
			}

			version, xErr = resource.GetDefault(FOR_WRITE)
		}
		if xErr != nil {
			return xErr
		}
	}

	if method == "POST" && numParts == 4 {
		// POST GROUPs/gID/RESOURCEs/rID[$details], body=obj or doc
		propsID := "" // versionid
		if v, ok := IncomingObj["versionid"]; ok {
			if reflect.ValueOf(v).Kind() == reflect.String {
				propsID = NotNilString(&v)
			}
		}

		if resource == nil {
			// Implicitly create the resource
			resource, isNew, xErr = group.UpsertResourceWithObject(
				info.ResourceType, resourceUID, propsID, IncomingObj,
				ADD_ADD, true)
			if xErr != nil {
				return xErr
			}
			version, xErr = resource.GetDefault(FOR_WRITE)
		} else {
			version, isNew, xErr = resource.UpsertVersionWithObject(propsID,
				IncomingObj, ADD_UPSERT, false)
		}
		if xErr != nil {
			return xErr
		}
		// Default to just returning the version
	}

	// GROUPs/gID/RESOURCEs/rID/meta
	if numParts > 4 && info.Parts[4] == "meta" {
		// PUT /GROUPs/gID/RESOURCEs/rID/meta
		addType := ADD_UPSERT
		if method == "PATCH" {
			addType = ADD_PATCH
		}

		if resource == nil {
			isNew = true
			propsID := "" // RESOURCEid
			if v, ok := IncomingObj[info.ResourceModel.Singular+"id"]; ok {
				if reflect.ValueOf(v).Kind() == reflect.String {
					propsID = NotNilString(&v)
				}
			}

			if propsID != "" && propsID != resourceUID {
				return NewXRError("mismatched_id", info.GetParts(4),
					"singular="+info.ResourceModel.Singular,
					"invalid_id="+propsID,
					"expected_id="+resourceUID)
			}

			// Implicitly create the resource
			resource, _, xErr = group.UpsertResourceWithObject(
				// TODO check to see if "" should be propsID
				info.ResourceType, resourceUID, "", map[string]any{},
				ADD_ADD, false)
			if xErr != nil {
				return xErr
			}
		}

		// Technically, this will always "update" not "insert"
		meta, _, xErr := resource.UpsertMetaWithObject(IncomingObj, addType,
			true, true)
		if xErr != nil {
			return xErr
		}

		// Return HTTP GET of 'meta'
		if isNew { // 201, else let it default to 200
			info.AddHeader("Location", info.BaseURL+"/"+meta.Path)
			info.StatusCode = http.StatusCreated
		}

		resPaths := map[string][]string{"": []string{meta.Path}}
		return SerializeQuery(info, resPaths, "Entity", info.Filters)
	}

	// Just double-check
	if numParts > 4 {
		PanicIf(info.Parts[4] != "versions", "Not 'versions': %s", info.Parts[4])
	}

	// GROUPs/gID/RESOURCEs/rID/versions...

	if info.ShowDetails && numParts == 5 {
		// PATCH|POST GROUPs/gID/RESOURCEs/rID/versions$details - error
		// TODO add a test for this
		panic("should never get here - info should catch this")
		return NewXRError("bad_details", info.GetParts(5))
	}

	if numParts == 5 && (method == "POST" || method == "PATCH") {
		// POST GROUPs/gID/RESOURCEs/rID/versions, body=map[id]->Version
		// PATCH GROUPs/gID/RESOURCEs/rID/versions, body=map[id]->Version

		// Convert IncomingObj to a map of Objects
		objMap, xErr := IncomingObj2Map(IncomingObj)
		if xErr != nil {
			return xErr
		}

		thisVersion := (*Version)(nil)

		if resource == nil {
			// Implicitly create the resource
			if len(objMap) == 0 {
				return NewXRError("missing_versions", "/"+info.OriginalPath)
			}

			vID := info.GetFlag("setdefaultversionid")
			/*
				if vID == "null" {
				return NewXRError("bad_defaultversionid", "/"+info.OriginalPath,
				"value=null",
				"error_detail=\"null\" is not allowed to be used")
				}
			*/

			if vID == "request" {
				if vID == "request" && len(objMap) > 1 {
					return NewXRError("too_many_versions", "/"+info.OriginalPath)
				}
			}

			tmpObj := map[string]any{
				"versions": (map[string]any)(IncomingObj),
			}

			addType := ADD_UPSERT
			if method == "PATCH" {
				addType = ADD_PATCH
			}

			resource, _, xErr = group.UpsertResourceWithObject(info.ResourceType,
				resourceUID, "", tmpObj, addType, false)

			if xErr != nil {
				return xErr
			}

			v, xErr := resource.GetDefault(FOR_WRITE)
			Must(xErr)
			thisVersion = v

			// Remove the newly created default version from objMap so we
			// won't process it again, but add it to the reuslts collection
			for id, _ := range objMap {
				paths = append(paths, strings.Join(
					[]string{info.GroupType, info.GroupUID, info.ResourceType,
						info.ResourceUID, "versions", id},
					"/"))
			}
		} else {
			meta, xErr := resource.FindMeta(false, FOR_WRITE)
			PanicIf(xErr != nil, "No meta %q: %s", resource.UID, xErr)

			if meta.Get("readonly") == true {
				return NewXRError("readonly", resource.XID)
			}

			// Process the versions
			addType := ADD_UPSERT
			if method == "PATCH" {
				addType = ADD_PATCH
			}
			count := 0
			for id, obj := range objMap {
				count++
				v, _, xErr := resource.UpsertVersionWithObject(id, obj, addType,
					count != len(objMap))
				if xErr != nil {
					return xErr
				}

				paths = append(paths, v.Path)
			}
		}

		xErr = ProcessSetDefaultVersionIDFlag(info, resource, thisVersion)
		if xErr != nil {
			return xErr
		}

		if len(paths) == 0 {
			paths = []string{"!"} // Force an empty collection to be returned
		}
		resPaths := map[string][]string{"": paths}
		return SerializeQuery(info, resPaths, "Coll", info.Filters)
	}

	if numParts == 6 {
		// PUT GROUPs/gID/RESOURCEs/rID/versions/vID [$details]
		propsID := "" //versionid
		if v, ok := IncomingObj["versionid"]; ok {
			if reflect.ValueOf(v).Kind() == reflect.String {
				propsID = NotNilString(&v)
			}
		}

		if resource == nil {
			// Implicitly create the resource
			resource, xErr = group.AddResourceWithObject(info.ResourceType,
				resourceUID, versionUID, IncomingObj, true)
			if xErr != nil {
				return xErr
			}

			isNew = true
		}

		version, xErr = resource.FindVersion(versionUID, false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}

		if version == nil {
			// We have a Resource, so add a new Version based on IncomingObj
			version, isNew, xErr = resource.UpsertVersionWithObject(versionUID,
				IncomingObj, ADD_UPSERT, false)
		} else if !isNew {
			if propsID != "" && propsID != version.UID {
				return NewXRError("mismatched_id", version.XID,
					"singular=version",
					"invalid_id="+propsID,
					"expected_id="+version.UID)
			}

			IncomingObj["versionid"] = version.UID
			addType := ADD_UPSERT
			if method == "PATCH" || !metaInBody {
				addType = ADD_PATCH
			}
			version, _, xErr = resource.UpsertVersionWithObject(version.UID,
				IncomingObj, addType, false)
		}
		if xErr != nil {
			return xErr
		}
	}

	PanicIf(xErr != nil, "err should be nil")

	// Process any ?setdefaultversionid query parameter there might be
	xErr = ProcessSetDefaultVersionIDFlag(info, resource, version)
	if xErr != nil {
		return xErr
	}

	// Make sure everything is ok before we send back the results
	info.tx.Validate(info)

	originalLen := numParts

	// Need to setup info stuff in case we call HTTPGetContent
	info.Parts = []string{info.Parts[0], groupUID,
		info.Parts[2], resourceUID}
	info.What = "Entity"
	info.GroupUID = groupUID
	info.ResourceUID = resourceUID // needed for $details in URLs

	location := info.BaseURL + "/" + resource.Path
	if originalLen > 4 || (originalLen == 4 && method == "POST") {
		info.Parts = append(info.Parts, "versions", version.UID)
		info.VersionUID = version.UID
		location += "/versions/" + version.UID
	}

	if info.ShowDetails { // not 100% sure this the right way/spot
		location += "$details"
	}

	if isNew { // 201, else let it default to 200
		info.AddHeader("Location", location)
		info.StatusCode = http.StatusCreated
	}

	// Return the contents of the entity instead of the xReg metadata
	if !metaInBody {
		return HTTPGETContent(info)
	}

	// Return the xReg metadata of the entity processed
	if paths == nil {
		paths = []string{strings.Join(info.Parts, "/")}
	}

	resPaths := map[string][]string{"": paths}
	return SerializeQuery(info, resPaths, what, info.Filters)
}

func HTTPPUTCapabilities(info *RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	reqBody, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		info.StatusCode = http.StatusInternalServerError
		return NewXRError("parsing_data", info.GetParts(0),
			"error_detail="+err.Error())
	}

	reqBody, err = RemoveSchema(reqBody)
	if err != nil {
		return NewXRError("parsing_data", info.GetParts(0),
			"error_detail="+err.Error())
	}

	cap := &Capabilities{}

	method := info.OriginalRequest.Method
	if method == "PUT" {
		// Fall thru
	} else if method == "PATCH" {
		// put current capabilities into a simple map
		tmp := map[string]any{}
		tmpJSON, _ := json.Marshal(info.Registry.Capabilities)
		Must(Unmarshal(tmpJSON, &tmp))

		// Now override wth anything new
		err := Unmarshal(reqBody, &tmp)
		if err != nil {
			return NewXRError("parsing_data", "/capabilities",
				"error_detail="+err.Error())
		}

		reqBody, _ = json.Marshal(tmp)
	} else {
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action="+info.OriginalRequest.Method)
	}

	cap, xErr := ParseCapabilitiesJSON(reqBody)
	if xErr != nil {
		return xErr
	}

	xErr = cap.Validate()
	if xErr != nil {
		return xErr
	}

	if xErr = info.Registry.SetSave("#capabilities", ToJSON(cap)); xErr != nil {
		return xErr
	}

	return HTTPGETCapabilities(info)
}

func HTTPPUTModelSource(info *RequestInfo) *XRError {
	if len(info.Parts) > 1 {
		return NewXRError("api_not_found", info.GetParts(0))
	}

	if info.OriginalRequest.Method != "PUT" {
		return NewXRError("action_not_supported", "/modelsource",
			"action="+info.OriginalRequest.Method)
	}

	reqBody, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		return NewXRError("parsing_data", info.GetParts(0),
			"error_detail="+err.Error())
	}

	xErr := info.Registry.Model.ApplyNewModelFromJSON(reqBody)
	if xErr != nil {
		return xErr
	}

	return HTTPGETModelSource(info)
}

// Process the ?setdefaultversionid query parameter
// "resource" is the resource we're processing
// "version" is the version that was processed
func ProcessSetDefaultVersionIDFlag(info *RequestInfo, resource *Resource, version *Version) *XRError {
	vIDs := info.GetFlagValues("setdefaultversionid")
	if len(vIDs) == 0 {
		return nil
	}

	if info.ResourceModel.GetSetDefaultSticky() == false {
		return NewXRError("setdefaultversionid_not_allowed", resource.XID,
			"singular="+info.ResourceModel.Singular)
	}

	vID := vIDs[0]

	if vID == "" {
		return NewXRError("bad_defaultversionid", resource.XID,
			"value="+`""`,
			"error_detail=value must not be empty")
	}

	// "null" and "request" have special meaning
	if vID == "null" {
		// Unstick the default version and go back to newest=default
		return resource.SetDefault(nil)
	}

	if vID == "request" {
		if version == nil {
			return NewXRError("defaultversionid_request", resource.XID)
		}
		// stick default version to current one we just processed
		return resource.SetDefault(version)
	}

	version, xErr := resource.FindVersion(vID, false, FOR_READ)
	if xErr != nil {
		return xErr
	}
	if version == nil {
		return NewXRError("unknown_id", resource.XID,
			"singular=version",
			"id="+vID)
	}

	return resource.SetDefault(version)
}

func HTTPDelete(info *RequestInfo) *XRError {
	// DELETE /...
	if len(info.Parts) == 0 {
		// DELETE /
		return NewXRError("action_not_supported", "/", "action=DELETE")
	}

	var xErr *XRError
	var err error
	epochStr := info.GetFlag("epoch")
	epochInt := -1
	if epochStr != "" {
		epochInt, err = strconv.Atoi(epochStr)
		if err != nil || epochInt < 0 {
			return NewXRError("invalid_attribute", "/"+info.OriginalPath,
				"name=epoch",
				"error_detail="+
					fmt.Sprintf("value (%s) must be a uinteger", epochStr))
		}
	}

	// DELETE /GROUPs...
	gm := info.Registry.Model.Groups[info.GroupType]
	if gm == nil {
		return NewXRError("not_found", info.GetParts(1))
	}

	if len(info.Parts) == 1 {
		// DELETE /GROUPs
		xErr = HTTPDeleteGroups(info)
		if xErr != nil {
			return xErr
		}
		info.tx.Validate(info)
		return nil
	}

	// DELETE /GROUPs/gID...
	group, xErr := info.Registry.FindGroup(info.GroupType, info.GroupUID, false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}
	if group == nil {
		return NewXRError("not_found", info.GetParts(2))
	}

	if len(info.Parts) == 2 {
		// DELETE /GROUPs/gID
		if epochInt >= 0 {
			if e := group.Get("epoch"); e != epochInt {
				return NewXRError("mismatched_epoch", group.XID,
					"bad_epoch="+epochStr,
					"epoch="+fmt.Sprintf("%d", e))
			}
		}
		if xErr = group.Delete(); xErr != nil {
			return xErr
		}

		info.tx.Validate(info)

		info.StatusCode = http.StatusNoContent
		return nil
	}

	// DELETE /GROUPs/gID/RESOURCEs...
	if rm := gm.FindResourceModel(info.ResourceType); rm == nil {
		return NewXRError("not_found", info.GetParts(3))
	}

	if len(info.Parts) == 3 {
		// DELETE /GROUPs/gID/RESOURCEs
		xErr = HTTPDeleteResources(info)
		if xErr != nil {
			return xErr
		}
		info.tx.Validate(info)
		return nil
	}

	// DELETE /GROUPs/gID/RESOURCEs/rID...
	resource, xErr := group.FindResource(info.ResourceType, info.ResourceUID,
		false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}
	if resource == nil {
		return NewXRError("not_found", info.GetParts(4))
	}

	meta, xErr := resource.FindMeta(false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}

	if len(info.Parts) == 4 {
		// DELETE /GROUPs/gID/RESOURCEs/rID
		if epochInt >= 0 {
			if e := meta.Get("epoch"); e != epochInt {
				return NewXRError("mismatched_epoch", meta.XID,
					"bad_epoch="+epochStr,
					"epoch="+fmt.Sprintf("%d", e))
			}
		}

		xErr = resource.Delete()
		if xErr != nil {
			return xErr
		}

		info.tx.Validate(info)

		info.StatusCode = http.StatusNoContent
		return nil
	}

	if len(info.Parts) == 5 && info.Parts[4] == "meta" {
		// DELETE /GROUPs/gID/RESOURCEs/rID/meta
		return NewXRError("action_not_supported", "/"+info.OriginalPath,
			"action=DELETE")
	}

	if len(info.Parts) == 5 {
		// DELETE /GROUPs/gID/RESOURCEs/rID/versions
		xErr = HTTPDeleteVersions(info)
		if xErr != nil {
			return xErr
		}

		info.tx.Validate(info)
		return nil
	}

	// DELETE /GROUPs/gID/RESOURCEs/rID/versions/vID...
	version, xErr := resource.FindVersion(info.VersionUID, false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}
	if version == nil {
		return NewXRError("not_found", info.GetParts(0))
	}

	if len(info.Parts) == 6 {
		// DELETE /GROUPs/gID/RESOURCEs/rID/versions/vID
		if epochInt >= 0 {
			if e := version.Get("epoch"); e != epochInt {
				return NewXRError("mismatched_epoch", version.XID,
					"bad_epoch="+epochStr,
					"epoch="+fmt.Sprintf("%d", e))
			}
		}
		nextDefault := info.GetFlag("setdefaultversionid")
		xErr = version.DeleteSetNextVersion(nextDefault)
		if xErr != nil {
			return xErr
		}

		info.tx.Validate(info)

		info.StatusCode = http.StatusNoContent
		return nil
	}

	return NewXRError("not_found", info.GetParts(0))
}

type EpochEntry map[string]any
type EpochEntryMap map[string]EpochEntry

func LoadEpochMap(info *RequestInfo) (EpochEntryMap, *XRError) {
	res := EpochEntryMap{}

	body, err := io.ReadAll(info.OriginalRequest.Body)
	if err != nil {
		return nil, NewXRError("parsing_data", "/"+info.OriginalPath,
			"error_detail="+err.Error())
	}

	bodyStr := strings.TrimSpace(string(body))

	if len(bodyStr) > 0 {
		err = Unmarshal([]byte(bodyStr), &res)
		if err != nil {
			return nil, NewXRError("parsing_data", info.GetParts(0),
				"error_detail="+err.Error())
		}
	} else {
		// EpochEntryMap == nil mean no list at all, not same as empty list
		return nil, nil
	}

	return res, nil
}

func HTTPDeleteGroups(info *RequestInfo) *XRError {
	list, xErr := LoadEpochMap(info)
	if xErr != nil {
		return xErr
	}

	// No list provided so get list of Groups so we can delete them all
	// TODO: optimize this to just delete it all in one shot
	if list == nil {
		list = EpochEntryMap{}
		results := Query(info.tx, `
			SELECT UID
			FROM Entities
			WHERE RegSID=? AND Abstract=?`,
			info.Registry.DbSID, info.GroupType)

		for row := results.NextRow(); row != nil; row = results.NextRow() {
			list[NotNilString(row[0])] = EpochEntry{}
		}
		defer results.Close()
	}

	// Delete each Group, checking epoch first if provided
	for id, entry := range list {
		group, xErr := info.Registry.FindGroup(info.GroupType, id, false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}
		if group == nil {
			// Silently ignore the 404
			continue
		}

		if tmp, ok := entry["epoch"]; ok {
			tmpInt, err := AnyToUInt(tmp)
			if err != nil {
				return NewXRError("invalid_attribute", group.XID,
					"name=epoch",
					"error_detail=must be a uinteger")
			}
			if tmpInt != group.Get("epoch") {
				return NewXRError("mismatched_epoch", group.XID,
					"bad_epoch="+fmt.Sprintf("%v", tmp),
					"epoch="+fmt.Sprintf("%d", tmpInt))
			}
		}

		singular := group.Singular + "id"
		if tmp, ok := entry[singular]; ok && tmp != id {
			return NewXRError("mismatched_id", group.XID,
				"singular="+group.Singular,
				"invalid_id="+fmt.Sprintf("%v", tmp),
				"expected_id="+id)
		}

		xErr = group.Delete()
		if xErr != nil {
			return xErr
		}
	}

	info.StatusCode = http.StatusNoContent
	return nil
}

func HTTPDeleteResources(info *RequestInfo) *XRError {
	list, xErr := LoadEpochMap(info)
	if xErr != nil {
		return xErr
	}

	// No list provided so get list of Resources so we can delete them all
	// TODO: optimize this to just delete it all in one shot
	if list == nil {
		list = EpochEntryMap{}
		results := Query(info.tx, `
			SELECT UID
			FROM Entities
			WHERE RegSID=? AND Abstract=?`,
			info.Registry.DbSID,
			NewPPP(info.GroupType).P(info.ResourceType).Abstract())

		for row := results.NextRow(); row != nil; row = results.NextRow() {
			list[NotNilString(row[0])] = EpochEntry{}
		}
		defer results.Close()
	}

	group, xErr := info.Registry.FindGroup(info.GroupType, info.GroupUID, false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}

	// Delete each Resource, checking epoch first if provided
	for id, entry := range list {
		resource, xErr := group.FindResource(info.ResourceType, id, false,
			FOR_WRITE)
		if xErr != nil {
			return xErr
		}
		if resource == nil {
			// Silently ignore the 404
			continue
		}

		meta, xErr := resource.FindMeta(false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}

		singular := resource.Singular + "id"

		if metaJSON, ok := entry["meta"]; ok {
			metaMap, ok := metaJSON.(map[string]any)
			if !ok {
				if xErr != nil { // makes no sense  TODO
					return NewXRError("invalid_attribute", resource.XID,
						"name=meta",
						"error_detail="+
							fmt.Sprintf("meta needs to be an object, "+
								"not a \"%T\"", metaJSON))
				}
			}

			if tmp, ok := metaMap[singular]; ok && tmp != id {
				return NewXRError("mismatched_id", resource.XID,
					"singular="+resource.Singular,
					"invalid_id="+fmt.Sprintf("%v", tmp),
					"expected_id="+id)
			}

			if tmp, ok := metaMap["epoch"]; ok {
				tmpInt, err := AnyToUInt(tmp)
				if err != nil {
					return NewXRError("invalid_attribute", meta.XID,
						"name=epoch",
						"error_detail=must be a uinteger")
				}
				if tmpInt != meta.Get("epoch") {
					return NewXRError("mismatched_epoch", meta.XID,
						"bad_epoch="+fmt.Sprintf("%v", tmp),
						"epoch="+fmt.Sprintf("%d", meta.Get("epoch")))
				}
			}
		} else {
			if _, ok := entry["epoch"]; ok {
				return NewXRError("misplaced_epoch", resource.XID)
			}
		}

		if tmp, ok := entry[singular]; ok && tmp != id {
			return NewXRError("mismatched_id", resource.XID,
				"singular="+resource.Singular,
				"invalid_id="+fmt.Sprintf("%v", tmp),
				"expected_id="+id)
		}

		xErr = resource.Delete()
		if xErr != nil {
			return xErr
		}
	}

	info.StatusCode = http.StatusNoContent
	return nil
}

func HTTPDeleteVersions(info *RequestInfo) *XRError {
	nextDefault := info.GetFlag("setdefaultversionid")

	list, xErr := LoadEpochMap(info)
	if xErr != nil {
		return xErr
	}

	group, xErr := info.Registry.FindGroup(info.GroupType, info.GroupUID, false, FOR_READ)
	if xErr != nil {
		return xErr
	}

	resource, xErr := group.FindResource(info.ResourceType, info.ResourceUID,
		false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}

	// No list provided so get list of Versions so we can delete them all
	// TODO: optimize this to just delete it all in one shot
	if list == nil {
		list = EpochEntryMap{}
		vIDs, xErr := resource.GetVersionIDs()
		if xErr != nil {
			return xErr
		}
		for _, vID := range vIDs {
			list[vID] = EpochEntry{}
		}
	}

	// Before we delete each one, make sure the epoch value is ok.
	// We can't check and delete at the same time before deleting one
	// might update a future one's ancestor value which will also change
	// its epoch value - which will make the epoch check fail.
	// However, we can save the Versions for the next loop.
	vers := []*Version{}
	for id, entry := range list {
		version, xErr := resource.FindVersion(id, false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}
		if version == nil {
			// Silently ignore the 404
			continue
		}
		vers = append(vers, version)

		// check epoch
		if tmp, ok := entry["epoch"]; ok {
			tmpInt, err := AnyToUInt(tmp)
			if err != nil {
				return NewXRError("invalid_attribute", version.XID,
					"name=epoch",
					"error_detail=value must be a uinteger")
			}
			if tmpInt != version.Get("epoch") {
				return NewXRError("mismatched_epoch", version.XID,
					"bad_epoch="+fmt.Sprintf("%v", tmp),
					"epoch="+fmt.Sprintf("%d", tmpInt))
			}
		}

		// For safety make sure the Resource's ID on the version (if there)
		// matches
		singular := version.Singular + "id"
		if tmp, ok := entry[singular]; ok && tmp != version.Get(singular) {
			return NewXRError("mismatched_id", version.XID,
				"singular=version",
				"invalid_id="+fmt.Sprintf("%v", tmp),
				"expected_id="+fmt.Sprintf("%v", version.Get(singular)))
		}
	}

	// Now we can actually delete each one
	for _, version := range vers {
		xErr = version.DeleteSetNextVersion(nextDefault)
		if xErr != nil {
			return xErr
		}
	}

	if nextDefault != "" {
		version, xErr := resource.FindVersion(nextDefault, false, FOR_READ)
		if xErr != nil {
			return xErr
		}
		xErr = resource.SetDefault(version)
		if xErr != nil {
			return xErr
		}
	}

	info.StatusCode = http.StatusNoContent
	return nil
}

func ExtractIncomingObject(info *RequestInfo, body []byte) (Object, *XRError) {
	IncomingObj := map[string]any{}

	if len(body) == 0 {
		body = nil
	}

	resSingular := ""
	hasDoc := false
	if info.ResourceModel != nil {
		resSingular = info.ResourceModel.Singular
		hasDoc = info.ResourceModel.GetHasDocument()
	}

	// Start with the assumption that we need a body, until proven otherwise
	requireBody := true
	if hasDoc && !info.ShowDetails {
		// .../rID || .../vID
		if len(info.Parts) == 4 || len(info.Parts) == 6 {
			requireBody = false
		}
	}

	if requireBody && len(body) == 0 {
		return nil, NewXRError("missing_body", "/"+info.OriginalPath)
	}

	// len=5 is a special case where we know .../versions always has the
	// metadata in the body so $details isn't needed, and in fact an error

	// GROUPS/GID/RESOURCES/RID/meta|versions/vID
	metaInBody := (info.ShowDetails ||
		len(info.Parts) == 3 ||
		len(info.Parts) == 5 ||
		(info.ResourceModel != nil && hasDoc == false))

	if len(info.Parts) < 3 || metaInBody {
		for k, _ := range info.OriginalRequest.Header {
			k := strings.ToLower(k)
			if strings.HasPrefix(k, "xregistry-") {
				if hasDoc == false {
					return nil, NewXRError("extra_xregistry_header",
						"/"+info.OriginalPath,
						"name="+k,
						"error_detail="+
							fmt.Sprintf("including \"xRegistry\" headers "+
								"for a Resource that has the model "+
								"\"hasdocument\" value of \"false\" is invalid"))
				}
				return nil, NewXRError("extra_xregistry_header",
					"/"+info.OriginalPath,
					"name="+k,
					"error_detail="+
						fmt.Sprintf("including \"xRegistry\" HTTP headers "+
							"when \"$details\" is used is not allowed"))
			}
		}

		err := Unmarshal(body, &IncomingObj)
		if err != nil {
			return nil, NewXRError("parsing_data", info.GetParts(0),
				"error_detail="+err.Error())
		}

		// "modelsource" is sooo special! Don't parse it into a golang type
		// keep it as []byte so that we preserve the order of the map keys
		if len(info.Parts) == 0 && !IsNil(IncomingObj["modelsource"]) {
			tmpReg := struct {
				ModelSource json.RawMessage
			}{}
			if err := json.Unmarshal(body, &tmpReg); err != nil {
				return nil, NewXRError("parsing_data", info.GetParts(0),
					"error_detail="+err.Error())
			}
			IncomingObj["modelsource"] = tmpReg.ModelSource
		}

		// Delete any json schema tag in there
		delete(IncomingObj, "$schema")
	}

	// xReg metadata are in headers, so move them into IncomingObj. We'll
	// copy over the existing properties later once we know what entity
	// we're dealing with
	if len(info.Parts) > 2 && !metaInBody {
		IncomingObj[resSingular] = body // save new body

		seenMaps := map[string]bool{}

		for name, attr := range attrHeaders {
			// TODO we may need some kind of "delete if missing" flag on
			// each HttpHeader attribute since some may want to have an
			// explicit 'null' to be erased instead of just missing (eg patch)
			vals, ok := info.OriginalRequest.Header[http.CanonicalHeaderKey(name)]
			if ok {
				val := vals[0]
				if val == "null" {
					IncomingObj[attr.Name] = nil
				} else {
					IncomingObj[attr.Name] = val
				}
			}
		}

		for key, value := range info.OriginalRequest.Header {
			key := strings.ToLower(key)

			if !strings.HasPrefix(key, "xregistry-") {
				continue
			}

			key = strings.TrimSpace(key[10:]) // remove xRegistry-
			if key == "" {
				return nil, NewXRError("header_error", "/"+info.OriginalPath,
					"name=xRegistry-",
					`error_detail=missing an attribute name after the "-"`)
			}

			if key == resSingular || key == resSingular+"base64" {
				return nil, NewXRError("extra_xregistry_header",
					"/"+info.OriginalPath,
					"name=xRegistry-"+key,
					"error_detail="+
						fmt.Sprintf("'xRegistry-%s' isn't allowed as "+
							"an HTTP header", key))
			}

			if key == resSingular+"url" || key == resSingular+"proxyurl" {
				if len(body) != 0 {
					return nil, NewXRError("extra_xregistry_header",
						"/"+info.OriginalPath,
						"name=xRegistry-"+key,
						"error_detail=header isn't allowed if there's a body")
				}
			}

			val := any(value[0])
			if val == "null" {
				val = nil
			}

			// If there are -'s then it's a non-scalar, convert it.
			// Note that any "-" after the 1st is part of the key name
			// labels-keyName && labels-"key-name"
			parts := strings.SplitN(key, "-", 2)
			if len(parts) > 1 {
				obj := IncomingObj

				if _, ok := seenMaps[parts[0]]; !ok {
					// First time we've seen this map, delete old stuff
					delete(IncomingObj, parts[0])
					seenMaps[parts[0]] = true
				}

				for i, part := range parts {
					if i+1 == len(parts) {
						// Should we just skip all of this logic if nil?
						// If we try, watch for the case where someone
						// has just xReg-label-foo:null, it should probably
						// create the empty map anyway. And watch for the
						// case mentioned below
						if val != nil {
							obj[part] = val
						}
						continue
					}

					prop, ok := obj[part]
					if !ok {
						if val == nil {
							break
						}
						tmpO := map[string]any{}
						obj[part] = tmpO
						obj = map[string]any(tmpO)
					} else {
						obj, ok = prop.(map[string]any)
						PanicIf(!ok, "Prop isn't map: %#v", prop)
					}
				}
			} else {
				if IsNil(val) {
					if _, ok := seenMaps[key]; ok {
						// Do nothing if we've seen keys for this map already.
						// We don't want to erase any keys we just added.
						// This is an edge/error? case where someone included
						// xReg-label:null AND xreg-label-foo:foo - keep "foo"
					} else {
						// delete(IncomingObj, key)
						IncomingObj[key] = nil
					}
				} else {
					IncomingObj[key] = val
					// However, if there's a real (non-nil) value for 'key'
					// and we've seen this key as a map, delete the fact we've
					// seen this map/key before so it'll be created if we do
					// see a map entry again
					delete(seenMaps, key)
				}
			}
		}
	}

	// Convert all HTTP header values into their proper data types since
	// as of now they're all just strings
	if !metaInBody && info.ResourceModel != nil {
		attrs := info.ResourceModel.GetBaseAttributes()
		attrs.AddIfValuesAttributes(IncomingObj)
		attrs.ConvertStrings(IncomingObj)
	}

	return IncomingObj, nil
}

func HTTPProxy(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host") // http://xregistry.io/xreg
	path := r.URL.Query().Get("path") // /GROUPS?inline

	host = strings.Trim(host, "/")
	path = "/" + strings.Trim(path, "/")
	data := []byte(nil)

	reg, xErr := LoadRemoteRegistry(host)
	if xErr != nil {
		data = []byte(xErr.String())
	}

	log.VPrintf(4, "Download: %s%s", host, path)

	var err error
	if data == nil {
		data, err = DownloadURL(host + path)
		if !IsNil(err) {
			// See if we can be tricky and load the index.html file ourselves
			data, err = DownloadURL(host + path + "/index.html")
			if !IsNil(err) {
				data = []byte(err.Error())
			}
		}
	}

	log.VPrintf(4, "Data:\n%s", string(data))

	r.URL, err = url.Parse(path)
	r.RequestURI = path
	if err != nil {
		data = []byte(err.Error())
	}

	info := &RequestInfo{
		OriginalPath:    r.URL.Path, // path,
		OriginalRequest: r,          // not sure this is the best option
		Registry:        reg,
		BaseURL:         host,
		ProxyHost:       host,
		ProxyPath:       path,
	}

	if reg != nil && reg.Model != nil {
		if xErr = info.ParseRequestURL(); xErr != nil {
			data = []byte(xErr.String())
		}
	}

	w.Header().Add("Content-Type", "text/html")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Methods",
		"GET, PATCH, POST, PUT, DELETE")

	html := GenerateUI(info, data)
	w.Write(html)
}

func HTTPWriteError(info *RequestInfo, errAny any) {
	var xErr *XRError
	var ok bool

	if xErr, ok = errAny.(*XRError); !ok {
		xErr = NewXRError("bad_request", "/"+info.OriginalPath,
			"error_detail="+fmt.Sprintf("%v", errAny))
	}

	info.StatusCode = xErr.Code
	info.AddHeader("Content-Type", "application/json; charset=utf-8")

	for k, v := range xErr.Headers {
		info.AddHeader(k, v)
	}

	info.Write([]byte(xErr.ToJSON(info.BaseURL) + "\n"))
}
