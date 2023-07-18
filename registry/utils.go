package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/google/uuid"
)

func NewUUID() string {
	return uuid.NewString()[:8]
}

func IsNil(a any) bool {
	val := reflect.ValueOf(a)
	if !val.IsValid() {
		return true
	}
	switch val.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map,
		reflect.Func, reflect.Interface:

		return val.IsNil()
	}
	return false
}

func NotNilString(val *any) string {
	if val == nil || *val == nil {
		return ""
	}

	b := (*val).([]byte)
	return string(b)
}

func NotNilInt(val *any) int {
	if val == nil || *val == nil {
		return 0
	}

	b := (*val).(int64)
	return int(b)
}

func NotNilBool(val *any) bool {
	if val == nil || *val == nil {
		return false
	}

	return ((*val).(int64)) == 1
}

func JSONEscape(obj interface{}) string {
	buf, _ := json.Marshal(obj)
	return string(buf[1 : len(buf)-1])
}

func ToJSON(obj interface{}) string {
	buf, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Fatalf("Error Marshaling: %s", err)
	}
	return string(buf)
}

func URLBuild(base string, paths ...string) string {
	isFrag := strings.Index(base, "#") >= 0
	url := base
	url = strings.TrimRight(url, "/")

	for _, path := range paths {
		if isFrag {
			url += "/" + path
		} else {
			url += "/" + strings.ToLower(path)
		}
	}
	return url
}

func SortedKeys(m interface{}) []string {
	mk := reflect.ValueOf(m).MapKeys()

	keys := make([]string, 0, len(mk))
	for _, k := range mk {
		keys = append(keys, k.String())
	}
	sort.Strings(keys)
	return keys
}

func SetField(res any, name string, value *string, propType string) {
	log.VPrintf(3, ">Enter: SetField(%T, %s=%s(%s))",
		res, name, *value, propType)
	defer log.VPrintf(3, "<Exit: SetField")

	var val any
	var err error

	field := reflect.ValueOf(res).Elem().FieldByName("Extensions")
	if !field.IsValid() {
		panic(fmt.Sprintf("Can't find Extensions: %#v", res))
	}
	if field.IsNil() {
		// Since we're deleting the key anyway we can just return
		if value == nil {
			return
		}
		field.Set(reflect.ValueOf(map[string]any{}))
	}

	if value == nil {
		// delete any existing key from map
		field.SetMapIndex(reflect.ValueOf(name), reflect.Value{})
		return
	}

	if propType == "s" {
		val = *value
	} else if propType == "b" {
		val = (*value == "true")
	} else if propType == "i" {
		val, err = strconv.Atoi(*value)
		if err != nil {
			panic(fmt.Sprintf("error parsing int: %s", val))
		}
	} else if propType == "f" {
		val, err = strconv.ParseFloat(*value, 64)
		if err != nil {
			panic(fmt.Sprintf("error parsing float: %s", val))
		}
	} else {
		panic(fmt.Sprintf("bad type: %v", propType))
	}

	field.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(val))
}

type JSONData struct {
	Prefix   string
	Indent   string
	Registry *Registry
}

func ShowStack() {
	log.VPrintf(2, "-----")
	for i := 1; i < 20; i++ {
		pc, file, line, _ := runtime.Caller(i)
		log.VPrintf(2, "Caller: %s:%d", path.Base(runtime.FuncForPC(pc).Name()), line)
		if strings.Contains(file, "main") {
			break
		}
	}
}

func OneLine(buf []byte) []byte {
	buf = RemoveProps(buf)

	re := regexp.MustCompile(`[\r\n]*`)
	buf = re.ReplaceAll(buf, []byte(""))
	re = regexp.MustCompile(`([^a-zA-Z])\s+([^a-zA-Z])`)
	buf = re.ReplaceAll(buf, []byte(`$1$2`))
	re = regexp.MustCompile(`([^a-zA-Z])\s+([^a-zA-Z])`)
	buf = re.ReplaceAll(buf, []byte(`$1$2`))

	return buf
}

func RemoveProps(buf []byte) []byte {
	re := regexp.MustCompile(`\n[^{}]*\n`)
	buf = re.ReplaceAll(buf, []byte("\n"))

	re = regexp.MustCompile(`\s"tags": {\s*},*`)
	buf = re.ReplaceAll(buf, []byte(""))

	re = regexp.MustCompile(`\n *\n`)
	buf = re.ReplaceAll(buf, []byte("\n"))

	re = regexp.MustCompile(`\n *}\n`)
	buf = re.ReplaceAll(buf, []byte("}\n"))

	re = regexp.MustCompile(`}[\s,]+}`)
	buf = re.ReplaceAll(buf, []byte("}}"))
	buf = re.ReplaceAll(buf, []byte("}}"))

	return buf
}

func HTMLify(r *http.Request, buf []byte) []byte {
	str := fmt.Sprintf(`"(https?://%s[^"\n]*?)"`, r.Host)
	re := regexp.MustCompile(str)
	repl := fmt.Sprintf(`"<a href="$1?%s">$1?%s</a>"`,
		r.URL.RawQuery, r.URL.RawQuery)

	return re.ReplaceAll(buf, []byte(repl))
}
