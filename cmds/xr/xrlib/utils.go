package xrlib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
	// "text/tabwriter"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

// var VerboseFlag = EnvBool("XR_VERBOSE", false)
// var DebugFlag = EnvBool("XR_DEBUG", false)
var Server = EnvString("XR_SERVER", "")

func Debug(args ...any) {
	// if !DebugFlag || len(args) == 0 || IsNil(args[0]) {
	if log.GetVerbose() < 2 || len(args) == 0 || IsNil(args[0]) {
		return
	}

	fmtStr := ""
	ok := false

	if fmtStr, ok = args[0].(string); ok {
		// fmtStr already set
	} else {
		fmtStr = fmt.Sprintf("%v", args[0])
	}

	if len(fmtStr) == 0 {
		return
	}

	str := fmt.Sprintf(fmtStr, args[1:]...)

	log.VPrintf(2, str)
	/*
		fmt.Fprint(os.Stderr, str)
		if str[len(str)-1] != '\n' && str[len(str)-1] != '\r' {
			fmt.Fprint(os.Stderr, "\n")
		}
	*/
}

/*
func Verbose(args ...any) {
	if !VerboseFlag || len(args) == 0 || IsNil(args[0]) {
		return
	}

	fmtStr := ""
	ok := false

	if fmtStr, ok = args[0].(string); ok {
		// fmtStr already set
	} else {
		fmtStr = fmt.Sprintf("%v", args[0])
	}

	fmt.Fprintf(os.Stderr, fmtStr+"\n", args[1:]...)
}
*/

type HttpResponse struct {
	Code   int
	Status string
	Body   []byte
	Header http.Header
}

// statusCode, body
// Add headers (in and out) later
func HttpDo(verb string, url string, body []byte) (*HttpResponse, *XRError) {
	client := &http.Client{}
	// CheckRedirect: func(req *http.Request, via []*http.Request) error {
	// return http.ErrUseLastResponse
	// }}

	bodyReader := bytes.NewReader(body)

	req, err := http.NewRequest(verb, url, bodyReader)
	if err != nil {
		return nil, NewXRError("talking_to_server", url,
			"error_detail="+err.Error())
	}

	Debug("Request: %s %s", verb, url)
	if len(body) != 0 {
		Debug("Request Body:\n%s", string(body))
		Debug("--------------------")
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, NewXRError("talking_to_server", url,
			"error_detail="+err.Error())
	}

	body, err = io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, NewXRError("parsing_response", url,
			"error_detail="+err.Error())
	}

	var xErr *XRError
	if res.StatusCode/100 != 2 {
		tmp := res.Status
		if len(body) != 0 {
			tmp = string(body)
		}

		tmp = strings.TrimSpace(tmp)

		// If response has no body then we need to say something back to
		// the user. A non-zero exit code w/o any text isn't helpful.
		if tmp == "" {
			return nil, NewXRError("talking_to_server", url,
				"error_detail="+res.Status)
		} else {
			// If we 'think' it's an XRError then return it, else just
			// return the raw data
			err := json.Unmarshal([]byte(tmp), &xErr)
			if (err == nil && xErr.Type == "") || err != nil {
				xErr = NewXRError("talking_to_server", url,
					"error_detail="+tmp)
			}
		}
	}

	httpRes := &HttpResponse{
		Code:   res.StatusCode,
		Status: res.Status,
		Body:   body,
		Header: res.Header,
	}

	Debug("Response: %s", res.Status)

	showHeaders := map[string]bool{
		"location":                     true,
		"content-type":                 true,
		"content-disposition":          true,
		"access-control-allow-origin":  true,
		"access-control-allow-methods": true,
	}
	for _, key := range SortedKeys(res.Header) {
		val := res.Header[key][0]
		key = strings.ToLower(key)
		if strings.HasPrefix(key, "xregistry-") {
			Debug("xRegistry-%s: %s", key[10:], val)
		} else if showHeaders[key] {
			Debug("%s: %s", key, val)
		}
	}

	if len(body) != 0 {
		Debug("Response Body:\n%s", string(body))
		Debug("--------------------")
	}

	return httpRes, xErr
}

// Support "http" and "-" (stdin)
func ReadFile(fileName string) ([]byte, *XRError) {
	buf := []byte(nil)
	var err error

	if fileName == "" || fileName == "-" {
		buf, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, NewXRError("client_error", fileName,
				"error_detail="+
					"Error reading from stdin: "+err.Error()+".")

		}
	} else if strings.HasPrefix(fileName, "http") {
		res, err := http.Get(fileName)
		if err != nil {
			return nil, NewXRError("talking_to_server", fileName,
				"error_detail="+err.Error())
		}

		buf, err = io.ReadAll(res.Body)
		res.Body.Close()

		if err != nil {
			return nil, NewXRError("parsing_response", fileName,
				"error_detail="+err.Error()+".")
		}

		if res.StatusCode/100 != 2 {
			return nil, NewXRError("talking_to_server", fileName,
				"error_detail="+fmt.Sprintf("%s : %s", res.Status, string(buf)))
		}
	} else {
		buf, err = os.ReadFile(fileName)
		if err != nil {
			return nil, NewXRError("client_error", fileName,
				"error_detail="+
					fmt.Sprintf("Error reading file %q: %s", fileName, err))
		}
	}

	return buf, nil
}

func IsValidJSON(buf []byte) error {
	tmp := map[string]any{}
	if err := Unmarshal(buf, &tmp); err != nil {
		return err
	}
	return nil
}

func AnyToString(val any) (string, error) {
	valStr, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("%q isn't a string value", val)
	}
	return valStr, nil
}

func ValidateTypes(xid *Xid, reg *Registry, allowSingular bool) *XRError {
	if xid.Group == "" {
		return nil
	}

	gm := (*GroupModel)(nil)
	gList, xErr := reg.ListGroupModels()
	if xErr != nil {
		return xErr
	}
	sort.Strings(gList)
	for _, plural := range gList {
		m, xErr := reg.FindGroupModel(plural)
		if xErr != nil {
			return xErr
		}
		if m.Plural == xid.Group || (allowSingular && m.Singular == xid.Group) {
			gm = m
			break
		}
	}
	if gm == nil {
		return NewXRError("not_found", xid.Group).
			SetDetailf("Unknown Group type: %s", xid.Group)
	}

	if xid.Resource == "" {
		return nil
	}

	rm := (*ResourceModel)(nil)
	rList := gm.GetResourceList()
	for _, rName := range rList {
		m := gm.FindResourceModel(rName)
		if m.Plural == xid.Resource || (allowSingular && m.Singular == xid.Resource) {
			rm = m
			break
		}
	}
	if rm == nil {
		return NewXRError("not_found", xid.Resource).
			SetDetailf("Unknown Resource type: %s", xid.Resource)
	}

	if xid.Version != "" {
		if xid.Version != "versions" && (!allowSingular || xid.Version != "version") {
			return NewXRError("malformed_xid", xid.String(),
				"xid="+xid.String(),
				"error_detail="+
					fmt.Sprintf("expected %q not %q", "versions", xid.Version))
		}
	}
	return nil
}

func GetResourceModelFrom(xid *Xid, reg *Registry) (*ResourceModel, *XRError) {
	if xid.Resource == "" {
		return nil, nil
	}

	gm, xErr := reg.FindGroupModel(xid.Group)
	if xErr != nil {
		return nil, xErr
	}
	if gm == nil {
		return nil, NewXRError("not_found", "/"+xid.Group).
			SetDetailf("Unknown group type: %s", xid.Group)
	}

	rm := gm.FindResourceModel(xid.Resource)
	if rm == nil {
		return nil, NewXRError("not_found", "/"+xid.Group+"/"+xid.Resource).
			SetDetailf("Unknown resource type: %s", xid.Resource)
	}
	return rm, nil
}

func PrettyPrint(object any, prefix string, indent string) string {
	buf, _ := json.MarshalIndent(object, prefix, indent)
	return string(buf)
}

func Tablize(xidStr string, object any) string {
	str := ""
	xid, err := ParseXid(xidStr)
	if err != nil {
		panic(err.Error())
	}

	switch xid.Type {
	case ENTITY_REGISTRY:
		str = TablizeRegistry(object)
	case ENTITY_GROUP_TYPE:
	case ENTITY_GROUP:
	case ENTITY_RESOURCE_TYPE:
	case ENTITY_RESOURCE:
	case ENTITY_VERSION_TYPE:
	case ENTITY_VERSION:
	default:
		panic(fmt.Sprintf("Unknown xid type: %v", xid.Type))
	}

	return str
}

func TablizeRegistry(regObj any) string {
	return "Registry:"
}

func XRIndent(buf []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(buf))
	res := bytes.Buffer{}

	indent := ""
	// extra := ""
	var next any
	var nextErr error
	var nextStr string
	var token any
	var err error

	for {
		if IsNil(next) {
			token, err = dec.Token()
		} else {
			token, err = next, nextErr
		}
		next, nextErr = dec.Token()
		nextType := reflect.ValueOf(next).Type().String()
		nextStr, _ = next.(string)

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		tokenVal := reflect.ValueOf(token)
		tokenType := tokenVal.Type().String()
		tokenStr, _ := token.(string)

		switch tokenType {
		case "json.Delim":
			switch tokenStr {
			case "{":
				res.WriteString(indent + tokenStr)
				if nextStr != "{" {
					res.WriteString("\n")
					indent += "  "
				}
			case "}":
				indent = indent[:len(indent)-2]
				res.WriteString("\n" + indent + tokenStr)
			case "[":
				res.WriteString(indent + tokenStr)
				if nextStr != "]" {
					res.WriteString("\n")
					indent += "  "
				}
			case "]":
				indent = indent[:len(indent)-2]
				res.WriteString("\n" + indent + tokenStr)
			default:
				panic(tokenStr)
			}

		case "string", "float64", "<nil>":
			res.WriteString(tokenStr)
			if nextType != "json.Delim" {
				res.WriteString(",\n" + indent)
			}
		}
	}

	return res.Bytes(), nil
}

/*
type indentTabWriter struct {
	RealWriter io.Writer
	Indent     string
	First      *bool // must be a pointer to persist value across Write calls
}

func (itw indentTabWriter) Write(p []byte) (int, error) {
	str := bytes.Buffer{}
	for _, ch := range p {
		if *itw.First {
			*itw.First = false
			str.Write([]byte(itw.Indent))
		}
		str.WriteByte(ch)
		if ch == '\n' {
			*itw.First = true
		}
	}
	_, err := itw.RealWriter.Write(str.Bytes())
	return len(p), err
}

func NewIndentTabWriter(indent string, output io.Writer, minwidth, tabwidth,
	padding int, padchar byte, flags uint) *TabWriter {

	mybool := true
	itw := indentTabWriter{
		RealWriter: output,
		Indent:     indent,
		First:      &mybool,
	}
	w := NewTabWriter(&itw, indent, minwidth, tabwidth, padding, padchar, flags)

	return w
}
*/

func YesNo(v bool) string {
	return BoolStr(v, "y", "n")
}

func YesDash(v bool) string {
	return BoolStr(v, "y", "-")
}

func BoolStr(v bool, yes string, no string) string {
	if v {
		return yes
	}
	return no
}

func DownloadObject(urlPath string) (map[string]any, *XRError) {
	res, xErr := HttpDo("GET", urlPath, nil)
	if xErr != nil {
		return nil, xErr
	}
	if res.Code != 200 {
		return nil, NewXRError("talking_to_server", urlPath,
			"error_detail="+
				fmt.Sprintf("%s: %s", res.Status, res.Body))
	}

	object := map[string]any(nil)
	err := json.Unmarshal(res.Body, &object)
	if err != nil {
		return object, NewXRError("parsing_response", urlPath,
			"error_detail="+err.Error()).
			SetDetail("Response: " + string(res.Body))
	}
	return object, nil
}
