package xrlib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/xregistry/server/registry"
)

// var VerboseFlag = EnvBool("XR_VERBOSE", false)
var DebugFlag = EnvBool("XR_DEBUG", false)
var Server = EnvString("XR_SERVER", "")

func Debug(args ...any) {
	if !DebugFlag || len(args) == 0 || registry.IsNil(args[0]) {
		return
	}
	// VerboseFlag = true
	// Verbose(args)
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
	fmt.Fprint(os.Stderr, str)
	if str[len(str)-1] != '\n' && str[len(str)-1] != '\r' {
		fmt.Fprint(os.Stderr, "\n")
	}
}

/*
func Verbose(args ...any) {
	if !VerboseFlag || len(args) == 0 || registry.IsNil(args[0]) {
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

func EnvBool(name string, def bool) bool {
	val := os.Getenv(name)
	if val != "" {
		def = strings.EqualFold(val, "true")
	}
	return def
}

func EnvString(name string, def string) string {
	val := os.Getenv(name)
	if val != "" {
		def = val
	}
	return def
}

type HttpResponse struct {
	Code int
	Body []byte
}

// statusCode, body
// Add headers (in and out) later
func HttpDo(verb string, url string, body []byte) (*HttpResponse, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	bodyReader := bytes.NewReader(body)

	req, err := http.NewRequest(verb, url, bodyReader)
	if err != nil {
		return nil, err
	}

	Debug("Request: %s %s", verb, url)
	if len(body) != 0 {
		Debug("Request Body:\n%s", string(body))
		Debug("^--")
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err = io.ReadAll(res.Body)
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
		Code: res.StatusCode,
		Body: body,
	}

	Debug("Response: %d", httpRes.Code)
	if len(body) != 0 {
		Debug("Response Body:\n%s", string(body))
		Debug("^--")
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
	if err := registry.Unmarshal(buf, &tmp); err != nil {
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

func ToJSON(val any) string {
	buf, _ := json.MarshalIndent(val, "", "  ")
	return string(buf)
}

func ArrayContains(strs []string, needle string) bool {
	for _, s := range strs {
		if needle == s {
			return true
		}
	}
	return false
}

type XID struct {
	str        string
	Type       int
	IsEntity   bool
	Group      string
	GroupID    string
	Resource   string
	ResourceID string
	Version    string // always "versions" if "/versions" was present
	VersionID  string
}

const (
	ENTITY_REGISTRY = iota
	ENTITY_GROUP
	ENTITY_RESOURCE
	ENTITY_META
	ENTITY_VERSION
	ENTITY_MODEL

	ENTITY_GROUPTYPE
	ENTITY_RESOURCETYPE
	ENTITY_VERSIONTYPE
)

func ParseXID(xidStr string) *XID {
	xidStr = strings.TrimLeft(xidStr, "/")
	parts := strings.SplitN(xidStr, "/", 6)

	xid := &XID{
		str:      xidStr,
		Type:     ENTITY_REGISTRY,
		IsEntity: true,
	}

	if len(parts) > 0 {
		xid.Group = parts[0]
		if xid.Group != "" {
			xid.Type = ENTITY_GROUPTYPE
			xid.IsEntity = false
		}
		if len(parts) > 1 {
			xid.GroupID = parts[1]
			if xid.GroupID != "" {
				xid.Type = ENTITY_GROUP
				xid.IsEntity = true
			}
			if len(parts) > 2 {
				xid.Resource = parts[2]
				if xid.Resource != "" {
					xid.Type = ENTITY_RESOURCETYPE
					xid.IsEntity = false
				}
				if len(parts) > 3 {
					xid.ResourceID = parts[3]
					if xid.ResourceID != "" {
						xid.Type = ENTITY_RESOURCE
						xid.IsEntity = true
					}
					if len(parts) > 4 {
						xid.Version = parts[4]
						if xid.Version != "" {
							xid.Type = ENTITY_VERSIONTYPE
							xid.IsEntity = false
						}
						if len(parts) > 5 {
							xid.VersionID = parts[5]
							if xid.VersionID != "" {
								xid.Type = ENTITY_VERSION
								xid.IsEntity = true
							}
						}
					}
				}
			}
		}
	}
	return xid
}

func (xid *XID) GetResourceModelFrom(reg *Registry) (*ResourceModel, error) {
	if xid.Resource == "" {
		return nil, nil
	}

	gm := reg.Model.Groups[xid.Group]
	if gm == nil {
		return nil, fmt.Errorf("Unknown group type: %s", xid.Group)
	}

	rm := gm.Resources[xid.Resource]
	if rm == nil {
		return nil, fmt.Errorf("Uknown resource type: %s", xid.Resource)
	}
	return rm, nil
}

func (xid *XID) String() string {
	return xid.str
}

func PrettyPrint(object any, prefix string, indent string) string {
	return registry.PrettyPrint(object, prefix, indent)
}

func Humanize(xid string, object any) string {
	str := ""
	xidParts := ParseXID(xid)

	switch xidParts.Type {
	case ENTITY_REGISTRY:
		str = HumanizeRegistry(object)
	case ENTITY_GROUPTYPE:
	case ENTITY_GROUP:
	case ENTITY_RESOURCETYPE:
	case ENTITY_RESOURCE:
	case ENTITY_VERSIONTYPE:
	case ENTITY_VERSION:
	default:
		panic(fmt.Sprintf("Unknown xid type: %v", xidParts.Type))
	}

	return str
}

func HumanizeRegistry(regObj any) string {
	return "Registry:"
}
