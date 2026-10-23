package xrlib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	// "text/tabwriter"

	// log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

// var VerboseFlag = EnvBool("XR_VERBOSE", false)
// var DebugFlag = EnvBool("XR_DEBUG", false)

// If/when we remove this global "Server" we can then move HTTPHeaders
// to the same spot and remove that global var too
var Server = EnvString("XR_SERVER", "")
var HTTPHeaders = map[string]string(nil) // headers for all server requests

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

func ValidateTypes(xid *Xid, reg *registry.Registry, allowSingular bool) *XRError {
	if xid.Group == "" {
		return nil
	}

	model, xErr := reg.GetModel()
	if xErr != nil {
		return xErr
	}

	gm := (*registry.GroupModel)(nil)
	for _, plural := range SortedKeys(model.Groups) {
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
			SetDetailf("Unknown Group type: %s.", xid.Group)
	}

	if xid.Resource == "" {
		return nil
	}

	rm := (*registry.ResourceModel)(nil)
	for _, m := range gm.Resources {
		if m.Plural == xid.Resource || (allowSingular && m.Singular == xid.Resource) {
			rm = m
			break
		}
	}
	if rm == nil {
		return NewXRError("not_found", xid.Resource).
			SetDetailf("Unknown Resource type: %s.", xid.Resource)
	}

	if xid.Version != "" {
		if xid.Version != "meta" && xid.Version != "versions" && (!allowSingular || xid.Version != "version") {
			return NewXRError("malformed_xid", xid.String(),
				"xid="+xid.String(),
				"error_detail="+
					fmt.Sprintf("expected %q not %q", "versions", xid.Version))
		}
	}
	return nil
}

func GetResourceModelFrom(xid *Xid, reg *registry.Registry) (*registry.ResourceModel, *XRError) {
	if xid.Resource == "" {
		return nil, nil
	}

	gm, xErr := reg.FindGroupModel(xid.Group)
	if xErr != nil {
		return nil, xErr
	}
	if gm == nil {
		return nil, NewXRError("not_found", "/"+xid.Group).
			SetDetailf("Unknown group type: %s.", xid.Group)
	}

	rm := gm.Resources[xid.Resource]
	if rm == nil {
		return nil, NewXRError("not_found", "/"+xid.Group+"/"+xid.Resource).
			SetDetailf("Unknown resource type: %s.", xid.Resource)
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

func DownloadObject(debug bool, urlPath string) (map[string]any, *XRError) {
	res, xErr := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     urlPath,
		Headers: HTTPHeaders,
		Debug:   debug,
	})
	if xErr != nil {
		return nil, xErr
	}
	if res.JSONError != nil {
		return nil, res.JSONError
	}
	PanicIf(IsNil(res.JSON), "json is empty")
	return res.JSON, nil
}
