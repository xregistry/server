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
	"text/tabwriter"

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
	Body   []byte
	Header http.Header
}

// statusCode, body
// Add headers (in and out) later
func HttpDo(verb string, url string, body []byte) (*HttpResponse, error) {
	client := &http.Client{}
	// CheckRedirect: func(req *http.Request, via []*http.Request) error {
	// return http.ErrUseLastResponse
	// }}

	bodyReader := bytes.NewReader(body)

	req, err := http.NewRequest(verb, url, bodyReader)
	if err != nil {
		return nil, err
	}

	Debug("Request: %s %s", verb, url)
	if len(body) != 0 {
		Debug("Request Body:\n%s", string(body))
		Debug("--------------------")
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err = io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	if res.StatusCode/100 != 2 {
		tmp := res.Status
		if len(body) != 0 {
			tmp = string(body)
		}
		err = fmt.Errorf(tmp)
	}

	httpRes := &HttpResponse{
		Code:   res.StatusCode,
		Body:   body,
		Header: res.Header,
	}

	Debug("Response: %d", httpRes.Code)

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

	return httpRes, err
}

// Support "http" and "-" (stdin)
func ReadFile(fileName string) ([]byte, error) {
	buf := []byte(nil)
	var err error

	if fileName == "" || fileName == "-" {
		buf, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("Error reading from stdin: %s", err)
		}
	} else if strings.HasPrefix(fileName, "http") {
		res, err := http.Get(fileName)
		if err != nil {
			return nil, err
		}

		buf, err = io.ReadAll(res.Body)
		res.Body.Close()

		if err != nil {
			return nil, err
		}

		if res.StatusCode/100 != 2 {
			return nil, fmt.Errorf("Error downloading %q: %s\n%s",
				fileName, res.Status, string(buf))
		}
	} else {
		buf, err = os.ReadFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("Error reading file %q: %s", fileName, err)
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

func ValidateTypes(xid *Xid, reg *Registry, allowSingular bool) error {
	if xid.Group == "" {
		return nil
	}

	gm := (*GroupModel)(nil)
	gList, err := reg.ListGroupModels()
	if err != nil {
		return err
	}
	sort.Strings(gList)
	for _, plural := range gList {
		m, err := reg.FindGroupModel(plural)
		if err != nil {
			return err
		}
		if m.Plural == xid.Group || (allowSingular && m.Singular == xid.Group) {
			gm = m
			break
		}
	}
	if gm == nil {
		return fmt.Errorf("Unknown Group type: %s", xid.Group)
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
		return fmt.Errorf("Unknown Resource type: %s", xid.Resource)
	}

	if xid.Version != "" {
		if xid.Version != "versions" && (!allowSingular || xid.Version != "version") {
			return fmt.Errorf("Expected %q not %q", "versions", xid.Version)
		}
	}
	return nil
}

func GetResourceModelFrom(xid *Xid, reg *Registry) (*ResourceModel, error) {
	if xid.Resource == "" {
		return nil, nil
	}

	gm, err := reg.FindGroupModel(xid.Group)
	if err != nil {
		return nil, err
	}
	if gm == nil {
		return nil, fmt.Errorf("Unknown group type: %s", xid.Group)
	}

	rm := gm.FindResourceModel(xid.Resource)
	if rm == nil {
		return nil, fmt.Errorf("Unknown resource type: %s", xid.Resource)
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
	padding int, padchar byte, flags uint) *tabwriter.Writer {

	mybool := true
	itw := indentTabWriter{
		RealWriter: output,
		Indent:     indent,
		First:      &mybool,
	}
	w := tabwriter.NewWriter(&itw, minwidth, tabwidth, padding, padchar, flags)

	return w
}

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

func DownloadObject(urlPath string) (map[string]any, error) {
	res, err := HttpDo("GET", urlPath, nil)
	if err != nil {
		return nil, err
	}
	if res.Code != 200 {
		return nil, fmt.Errorf("Error downloading %q: %s %s", urlPath,
			res.Code, res.Body)
	}

	object := map[string]any(nil)
	err = json.Unmarshal(res.Body, &object)
	return object, err
}
