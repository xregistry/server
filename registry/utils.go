package registry

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/duglin/dlog"
	"github.com/google/uuid"
)

var count = 0 // UUID counter

func NewUUID() string {
	count++ // Help keep it unique w/o using the entire UUID string
	return fmt.Sprintf("%s%d", uuid.NewString()[:8], count)
}

func IsURL(str string) bool {
	return strings.HasPrefix(str, "http:") || strings.HasPrefix(str, "https:")
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func PanicIf(b bool, msg string, args ...any) {
	if b {
		Panicf(msg, args...)
	}
}
func Panicf(msg string, args ...any) {
	panic(fmt.Sprintf(msg, args...))
}

func init() {
	if !IsNil(nil) {
		panic("help me1")
	}
	if !IsNil(any(nil)) {
		panic("help me2")
	}
	if !IsNil((*any)(nil)) {
		panic("help me3")
	}
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

	if reflect.ValueOf(*val).Kind() == reflect.String {
		return (*val).(string)
	}

	if reflect.ValueOf(*val).Kind() != reflect.Slice {
		panic(fmt.Sprintf("Not a slice: %T (%#v)", *val, *val))
	}

	b := (*val).([]byte)
	return string(b)
}

func NotNilIntDef(val *any, def int) int {
	if val == nil || *val == nil {
		return def
	}

	var b int

	if reflect.ValueOf(*val).Kind() == reflect.Int64 {
		tmp, _ := (*val).(int64)
		b = int(tmp)
	} else {
		b, _ = (*val).(int)
	}

	return b
}

func NotNilInt(val *any) int {
	return NotNilIntDef(val, 0)
}

func PtrIntDef(val *any, def int) *int {
	result := NotNilIntDef(val, def)
	return &result
}

func NotNilBoolPtr(val *bool) bool {
	return val != nil && (*val) == true
}

func NotNilBoolDef(val *any, def bool) bool {
	if val == nil || *val == nil {
		return def
	}

	return ((*val).(int64)) == 1
}

func PtrBool(b bool) *bool {
	return &b
}

func PtrBoolDef(val *any, def bool) *bool {
	result := NotNilBoolDef(val, def)
	return &result
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

func ToJSONOneLine(obj interface{}) string {
	buf, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		log.Fatalf("Error Marshaling: %s", err)
	}

	re := regexp.MustCompile(`[\s\r\n]*`)
	buf = re.ReplaceAll(buf, []byte(""))

	return string(buf)
}

func Keys(m interface{}) []string {
	mk := reflect.ValueOf(m).MapKeys()

	keys := make([]string, 0, len(mk))
	for _, k := range mk {
		keys = append(keys, k.String())
	}
	return keys
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

func GetStack() []string {
	stack := []string{}

	for i := 1; i < 20; i++ {
		pc, file, line, _ := runtime.Caller(i)
		if line == 0 {
			break
		}
		stack = append(stack,
			fmt.Sprintf("%s - %s:%d",
				path.Base(runtime.FuncForPC(pc).Name()), path.Base(file), line))
		if strings.Contains(file, "main") || strings.Contains(file, "testing") {
			break
		}
	}
	return stack
}

func ShowStack() {
	stack := GetStack()
	log.VPrintf(0, "----- Stack")
	for _, line := range stack {
		log.VPrintf(0, " %s", line)
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

	// re = regexp.MustCompile(`\s"labels": {\s*},*`)
	// buf = re.ReplaceAll(buf, []byte(""))

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
	str := fmt.Sprintf(`"(https?://[^"\n]*?)"`)
	re := regexp.MustCompile(str)
	repl := fmt.Sprintf(`"<a href="$1?%s">$1?%s</a>"`,
		r.URL.RawQuery, r.URL.RawQuery)

	return re.ReplaceAll(buf, []byte(repl))
}

// HTML, # of expands
func RegHTMLify(buf []byte, proxyHost, targetHost string) ([]byte, int) {
	// Escape < and >
	buf = []byte(strings.ReplaceAll(string(buf), "<", "&lt;"))
	buf = []byte(strings.ReplaceAll(string(buf), ">", "&gt;"))

	str := fmt.Sprintf(`"(https?://[^?"\n]*)(\??)([^"\n]*)"`)
	re := regexp.MustCompile(str)
	repl := fmt.Sprintf(`"<a href='$1?ui&$3'>$1$2$3</a>"`)
	buf = re.ReplaceAll(buf, []byte(repl))

	if targetHost != "" {
		matchHost := targetHost

		// Allow for http or https in the output
		if strings.HasPrefix(matchHost, "http") {
			_, matchHost, _ = strings.Cut(matchHost, "://")
		}
		matchHost = `https?://` + matchHost

		str := fmt.Sprintf(`<a href='%s([^?']*)\?ui`, matchHost)
		re := regexp.MustCompile(str)
		repl := fmt.Sprintf(`<a href='%s/proxy?host=%s&path=$1`,
			proxyHost, targetHost)
		buf = re.ReplaceAll(buf, []byte(repl))
	}

	res := new(bytes.Buffer)

	// Now add the toggle (expand) stuff for the JSON nested entities

	// remove trailing \n so we don't have an extra line for the next stuff
	if len(buf) > 0 && buf[len(buf)-1] == '\n' {
		buf = buf[:len(buf)-1]
	}

	count := 0
	for _, line := range strings.Split(string(buf), "\n") {
		spaces := "" // leading spaces
		numSpaces := 0
		first := rune(0) // first non-space char
		last := rune(0)  // last non-space char
		decDepth := false

		for _, ch := range line {
			if first == 0 { // doing spaces
				if ch == ' ' {
					numSpaces++
					spaces += " "
					continue
				}
			}
			if ch != ' ' {
				if first == 0 {
					first = rune(ch)
				}
				last = rune(ch)
			}
		}
		line = line[numSpaces:] // Remove leading spaces

		decDepth = (first == '}' || first == ']')
		incDepth := (last == '{' || last == '[')

		// btn is the special first column of the output, non-selectable
		btn := "<span class=spc> </span>" // default: non-selectable space

		if incDepth {
			// Build the 'expand' toggle char
			count++
			exp := fmt.Sprintf("<span class=exp id='s%d' "+
				"onclick='toggleExp(this)'>"+HTML_EXP+"</span>", count)

			if numSpaces == 0 {
				// OLD - if we want toggle at the root too
				// Use the special first column for it
				// btn = exp
				// end OLD

				// Don't show toggle at root.
				// To ensure count starts at 1 for the real first toggle
				// just decrement it here
				count--
			} else {
				// Replace the last space with the toggle.
				// Add a nearly-hidden space so when people copy the text it
				// won't be missing a space due to the toggle
				spaces = spaces[:numSpaces-1] + exp +
					"<span class=hide > </span>"
			}
		}

		res.WriteString(btn)    // special first column
		res.WriteString(spaces) // spaces + 'exp' if needed
		if decDepth {
			// End the block before the tailing "}" or "]"
			res.WriteString("</span>") // block
		}
		res.WriteString(line)
		if incDepth {
			// write the "..." and then the <span> for the toggle text
			res.WriteString(fmt.Sprintf("<span style='display:none;cursor:pointer' "+
				"id='s%ddots' onclick='toggleExp(s%d)'>...</span>", count, count))
			res.WriteString(fmt.Sprintf("<span id='s%dblock'>", count))
		}

		res.WriteString("\n")
	}

	return res.Bytes(), count
}

func AnyToUInt(val any) (int, error) {
	var err error

	kind := reflect.ValueOf(val).Kind()
	resInt := 0
	if kind == reflect.Float64 { // JSON ints show up as floats
		resInt = int(val.(float64))
		if float64(resInt) != val.(float64) {
			err = fmt.Errorf("must be a uinteger")
		}
	} else if kind != reflect.Int {
		err = fmt.Errorf("must be a uinteger")
	} else {
		resInt = val.(int)
	}

	if err == nil && resInt < 0 {
		err = fmt.Errorf("must be a uinteger")
	}

	return resInt, err
}

func LineNum(buf []byte, pos int) int {
	return bytes.Count(buf[:pos], []byte("\n")) + 1
}

func Unmarshal(buf []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(buf))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		msg := JsonErrorToString(buf, dec, err)
		return errors.New(msg)
	}

	token, err := dec.Token()
	if err != nil && err != io.EOF { // Non-EOF error
		msg := JsonErrorToString(buf, dec, err)
		return errors.New(msg)
	}

	if !IsNil(token) { // There's more
		offset := int(dec.InputOffset())
		near := ""
		if offset > 0 && offset < len(buf) {
			near = fmt.Sprintf(" possibly near position %d", offset)
		}

		return fmt.Errorf("Error parsing json: extra data%s: %s",
			near, token)
	}

	return nil
}

func JsonErrorToString(buf []byte, dec *json.Decoder, err error) string {
	offset := int(dec.InputOffset())
	near := ""
	if offset > 0 && offset < len(buf) {
		near = fmt.Sprintf("possibly near position %d", offset)
	}

	msg := err.Error()

	if jerr, ok := err.(*json.UnmarshalTypeError); ok {
		msg = fmt.Sprintf("Can't parse %q as a(n) %q at line %d",
			jerr.Value, jerr.Type.String(),
			LineNum(buf, int(jerr.Offset)))
	} else if jerr, ok := err.(*json.SyntaxError); ok {
		msg = fmt.Sprintf("Syntax error at line %d: %s",
			LineNum(buf, int(jerr.Offset)), msg)
	} else {
		if msg == "unexpected EOF" {
			msg = "Error parsing json: " + msg
		}
	}
	msg, _ = strings.CutPrefix(msg, "json: ")
	if near != "" {
		msg += "; " + near
	}

	return msg
}

// var re = regexp.MustCompile(`(?m:([^#]*)#[^"]*$)`)
var removeCommentsRE = regexp.MustCompile(`(gm:^(([^"#]|"[^"]*")*)#.*$)`)

func RemoveComments(buf []byte) []byte {
	return removeCommentsRE.ReplaceAll(buf, []byte("${1}"))
}

type IncludeArgs struct {
	// Cache path/name of "" means stdin
	Cache      map[string]map[string]any // Path#.. -> json
	History    []string                  // Just names, no frag, [0]=latest
	LocalFiles bool                      // ok to access local FS files?
}

func ProcessIncludes(file string, buf []byte, localFiles bool) ([]byte, error) {
	data := map[string]any{}

	buf = RemoveComments(buf)

	if err := Unmarshal(buf, &data); err != nil {
		return nil, fmt.Errorf("Error parsing JSON: %s", err)
	}

	includeArgs := IncludeArgs{
		Cache: map[string]map[string]any{
			file: data,
		},
		History:    []string{file}, // stack of base names
		LocalFiles: localFiles,
	}

	if err := IncludeTraverse(includeArgs, data); err != nil {
		return nil, err
	}

	// Convert back to byte
	// buf, err := json.MarshalIndent(data, "", "  ")
	buf, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Error generating JSON: %s", err)
	}

	return buf, nil
}

// data is the current map to check for $include statements
func IncludeTraverse(includeArgs IncludeArgs, data map[string]any) error {
	var err error
	currFile, _ := SplitFragement(includeArgs.History[0]) // Grab just base name

	// log.Printf("IncludeTraverse:")
	// log.Printf("  Cache: %v", SortedKeys(includeArgs.Cache))
	// log.Printf("  History: %v", includeArgs.History)
	// log.Printf("  Recurse:")
	// log.Printf("    Data keys: %v", SortedKeys(data))

	_, ok1 := data["$include"]
	_, ok2 := data["$includes"]
	if ok1 && ok2 {
		return fmt.Errorf("In %q, both $include and $includes is not allowed",
			currFile)
	}

	dataKeys := Keys(data) // so we can add/delete keys
	for _, key := range dataKeys {
		val := data[key]
		if key == "$include" || key == "$includes" {
			delete(data, key)
			list := []string{}

			valValue := reflect.ValueOf(val)
			if key == "$include" {
				if valValue.Kind() != reflect.String {
					return fmt.Errorf("In %q, $include value isn't a string",
						currFile)
				}
				list = []string{val.(string)}
			} else {
				if valValue.Kind() != reflect.Slice {
					return fmt.Errorf("In %q, $includes value isn't an array",
						currFile)
				}

				for i := 0; i < valValue.Len(); i++ {
					impInt := valValue.Index(i).Interface()
					imp, ok := impInt.(string)
					if !ok {
						return fmt.Errorf("In %q, $includes contains a "+
							"non-string value (%v)", currFile, impInt)
					}
					list = append(list, imp)
				}
			}

			for _, impStr := range list {
				for _, name := range includeArgs.History {
					if name == impStr {
						return fmt.Errorf("Recursive on %q", name)
					}
				}

				if len(impStr) == 0 {
					return fmt.Errorf("In %q, $include can't be an empty "+
						"string", currFile)
				}

				// log.Printf("CurrFile: %s\nImpStr: %s", currFile, impStr)
				nextFile := ResolvePath(currFile, impStr)
				// log.Printf("NextFile: %s", nextFile)
				includeData := includeArgs.Cache[nextFile]
				base, fragment := SplitFragement(nextFile)

				if includeData == nil {
					includeData = includeArgs.Cache[base]
					if includeData == nil {
						/*
							fn, err := FindModelFile(base)
							if err != nil {
								return err
							}
						*/
						fn := base

						data := []byte(nil)
						if IsURL(fn) {
							res, err := http.Get(fn)
							if err != nil {
								return err
							}
							if res.StatusCode != 200 {
								return fmt.Errorf("Error getting %q: %s",
									fn, res.Status)
							}
							data, err = io.ReadAll(res.Body)
							res.Body.Close()
							if err != nil {
								return err
							}
						} else {
							if includeArgs.LocalFiles {
								if data, err = os.ReadFile(fn); err != nil {
									return fmt.Errorf("Error reading file "+
										"%q: %s", fn, err)
								}
							} else {
								return fmt.Errorf("Not allowed to access "+
									"file: %s", fn)
							}
						}
						data = RemoveComments(data)

						if err := Unmarshal(data, &includeData); err != nil {
							return err
						}
						includeArgs.Cache[base] = includeData
					}

					// Now, traverse down to the specific field - if needed
					if fragment != "" {
						nextTop := includeArgs.Cache[base]
						impData, err := GetJSONPointer(nextTop, fragment)
						if err != nil {
							return err
						}

						if reflect.ValueOf(impData).Kind() != reflect.Map {
							return fmt.Errorf("In %q, $include(%s) is not a "+
								"map: %s", currFile, impStr,
								reflect.ValueOf(includeData).Kind())
						}

						includeData = impData.(map[string]any)
						includeArgs.Cache[nextFile] = includeData
					}
				}

				// Go deep! (recurse) before we add it to current map
				includeArgs.History = append([]string{nextFile},
					includeArgs.History...)
				if err = IncludeTraverse(includeArgs, includeData); err != nil {
					return err
				}
				includeArgs.History = includeArgs.History[1:]

				// Only copy if we don't already have one by this name
				for k, v := range includeData {
					if _, ok := data[k]; !ok {
						data[k] = v
					}
				}
			}
		} else {
			if reflect.ValueOf(val).Kind() == reflect.Map {
				nextLevel := val.(map[string]any)
				if err = IncludeTraverse(includeArgs, nextLevel); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func SplitFragement(str string) (string, string) {
	parts := strings.SplitN(str, "#", 2)

	if len(parts) != 2 {
		return parts[0], ""
	} else {
		return parts[0], parts[1]
	}
}

var dotdotRE = regexp.MustCompile(`(^|/)[^/]*/\.\.(/|$)`) // removes /../
var slashesRE = regexp.MustCompile(`([^:])//+`)           // : is for URL's ://
var urlPrefixRE = regexp.MustCompile(`^https?://`)
var justHostRE = regexp.MustCompile(`^https?://[^/]*$`) // no path?
var extractHostRE = regexp.MustCompile(`^(https?://[^/]*/).*`)
var endingDots = regexp.MustCompile(`(/\.\.?)$`) // ends with . or ..

func ResolvePath(baseFile string, next string) string {
	baseFile, _ = SplitFragement(baseFile)
	baseFile = endingDots.ReplaceAllString(baseFile, "$1/")

	if next == "" {
		return baseFile
	}
	if next[0] == '#' {
		return baseFile + next
	}

	// Abs URLs
	if IsURL(next) {
		return next
	}

	// baseFile is a URL
	if urlPrefixRE.MatchString(baseFile) {
		if justHostRE.MatchString(baseFile) {
			baseFile += "/"
		}

		if next != "" && next[0] == '/' {
			baseFile = extractHostRE.ReplaceAllString(baseFile, "$1")
		}

		if baseFile[len(baseFile)-1] == '/' { // ends with /
			next = baseFile + next
		} else {
			i := strings.LastIndex(baseFile, "/") // remove last word
			if i >= 0 {
				baseFile = baseFile[:i+1] // keep last /
			}
			next = baseFile + next
		}
	} else {
		// Look for abs path for files
		if len(next) > 0 && next[0] == '/' {
			return next
		}

		if len(next) > 2 && next[1] == ':' {
			// Windows abs path ?
			ch := next[0]
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				return next
			}
		}

		baseFile = path.Dir(baseFile) // remove file name
		next = path.Join(baseFile, next)
	}

	// log.Printf("Before clean: %q", next)
	next, _ = strings.CutPrefix(next, "./")        // remove leading ./
	next = slashesRE.ReplaceAllString(next, "$1/") // squash //'s
	next = strings.ReplaceAll(next, "/./", "/")    // remove pointless /./
	next = dotdotRE.ReplaceAllString(next, "/")    // remove ../'s
	return next
}

var JPtrEsc0 = regexp.MustCompile(`~0`)
var JPtrEsc1 = regexp.MustCompile(`~1`)

func GetJSONPointer(data any, path string) (any, error) {
	// log.Printf("GPtr: path: %q\nData: %s", path, ToJSON(data))
	path = strings.TrimSpace(path)
	if path == "" {
		return data, nil
	}

	if IsNil(data) {
		return nil, nil
	}

	path, _ = strings.CutPrefix(path, "/")
	parts := strings.Split(path, "/")
	// log.Printf("Parts: %q", strings.Join(parts, "|"))

	for i, part := range parts {
		part = JPtrEsc1.ReplaceAllString(part, `/`)
		part = JPtrEsc0.ReplaceAllString(part, `~`)
		// log.Printf("  Part: %s", part)

		dataVal := reflect.ValueOf(data)
		kind := dataVal.Kind()
		if kind == reflect.Map {
			dataVal = dataVal.MapIndex(reflect.ValueOf(part))
			// log.Printf("dataVal: %#v", dataVal)
			if !dataVal.IsValid() {
				return nil, fmt.Errorf("Attribute %q not found",
					strings.Join(parts[:i+1], "/"))
			}
			data = dataVal.Interface()
			continue
		} else if kind == reflect.Slice {
			j, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("Index %q must be an integer",
					"/"+strings.Join(parts[:i+1], "/"))
			}
			if j < 0 || j >= dataVal.Len() { // len(daSlice) {
				return nil, fmt.Errorf("Index %q is out of bounds(0-%d)",
					"/"+strings.Join(parts[:i+1], "/"), dataVal.Len()-1)
			}
			data = dataVal.Index(j).Interface()
			continue
		} else {
			return nil, fmt.Errorf("Can't step into a type of %q, at: %s",
				kind, "/"+strings.Join(parts[:i+1], "/"))
		}
	}

	return data, nil
}

// Either delete or change the value of a map based on "oldVal" being nil or not
func ResetMap[M ~map[K]V, K comparable, V any](m M, key K, oldVal V) {
	if IsNil(oldVal) {
		delete(m, key)
	} else {
		m[key] = oldVal
	}
}

func IncomingObj2Map(incomingObj Object) (map[string]Object, error) {
	result := map[string]Object{}
	for id, obj := range incomingObj {
		oV := reflect.ValueOf(obj)
		if oV.Kind() != reflect.Map ||
			oV.Type().Key().Kind() != reflect.String {

			return nil, fmt.Errorf("Body must be a map of id->Entity")
		}
		newObj := Object{}
		for _, keyVal := range oV.MapKeys() {
			newObj[keyVal.Interface().(string)] =
				oV.MapIndex(keyVal).Interface()
		}
		result[id] = newObj
	}

	return result, nil
}

func Match(pattern string, str string) bool {
	ip, is := 0, 0                   // index of pattern or string
	lp, ls := len(pattern), len(str) // len of pattern or string

	for {
		// log.Printf("Check: %q  vs  %q", pattern[ip:], str[is:])
		// If pattern is empty then result is "is string empty?"
		if ip == lp {
			return is == ls
		}

		p := pattern[ip]
		if p == '*' {
			// DUG todo, remove the resursiveness of this
			for i := 0; i+is <= ls; i++ {
				if Match(pattern[ip+1:], str[is+i:]) {
					return true
				}
			}
			return false
		}

		// If we have a 'p' but string is empty, then false
		if is == ls {
			return false
		}
		s := str[ip]

		if p != s {
			return false
		}
		ip++
		is++
	}
	return false
}

func FindModelFile(name string) (string, error) {
	if IsURL(name) {
		return name, nil
	}

	if strings.HasPrefix(name, "/") {
		return name, nil
	}

	// Consider adding the github repo as a default value to PATH and
	// allowing the filename to be appended to it
	paths := os.Getenv("XR_MODEL_PATH")

	for _, path := range strings.Split(paths, ":") {
		path = strings.TrimSpace(path)
		if path == "" {
			path = "."
		}
		path = path + "/" + name

		if strings.HasPrefix(path, "//") {
			path = "https:" + path
		}

		if IsURL(path) {
			res, err := http.Get(path)
			if err == nil && res.StatusCode/100 == 2 {
				return path, nil
			}
		} else {

			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("Can't find %q in %q", name, paths)
}

func ConvertStrToTime(str string) (time.Time, error) {
	TSformats := []string{
		time.RFC3339,
		// time.RFC3339Nano,
		"2006-01-02T15:04:05.000000000Z07:00",
		"2006-01-02T15:04:05+07:00",
		"2006-01-02T15:04:05+07",
		"2006-01-02T15:04:05",
	}

	for _, tfs := range TSformats {
		if t, err := time.Parse(tfs, str); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("Invalid RFC3339 timestamp: %s", str)
}

func NormalizeStrTime(str string) (string, error) {
	t, err := ConvertStrToTime(str)
	if err != nil {
		return "", err
	}

	str = t.Format(time.RFC3339Nano)
	return str, nil
}

func ArrayContains(strs []string, needle string) bool {
	for _, s := range strs {
		if needle == s {
			return true
		}
	}
	return false
}

func ArrayContainsAnyCase(strs []string, needle string) bool {
	needle = strings.ToLower(needle)
	for _, s := range strs {
		if needle == strings.ToLower(s) {
			return true
		}
	}
	return false
}

// Convert a string into a unique MD5 string - basically just for cases
// where we want to create a tiny URL
func MD5(str string) string {
	sum := md5.Sum([]byte(str)) // 16 bytes, 128 bits
	return MakeShort(sum[:])
}

// Valid tiny URL chars.
// Number of them needs to be a multiple of 2
var ShortChars = "" +
	"abcdefghijklmnopqrstuvwxyz" + // 0->25
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ" + // 26->51
	"0123456789" + // 52->61
	"-@" // 62-63

// Convert a []byte into a string that uses ShortChars as its encoding.
// Similar to base64 but with a larger char set.
func MakeShort(buf []byte) string {
	// Each output char will take 6 bits, which means 64 possible chars.
	// Which is the size of ShortChars.
	// If we tried 7 bits, then we'd need 128 chars, which we don't have.

	num := int(math.Ceil((float64(len(buf)) * 8) / 6)) // # of bytes in result
	buf = append(buf, []byte{0}...)                    // add 0 if need padding

	str := ""      // result string
	size := 0      // size of str
	startByte := 0 // current byte pos in buf
	startBit := 0  // next bit in current byte to grab
	ch := byte(0)  // index into ShortChars array
	left := 6      // bits remaining for current ch (init to "all")

	for size < num {
		grab := 8 - startBit // try to grab the rest of current byte
		if left < grab {
			grab = left // only need 'grab' # of bites
		}

		ch = ch << grab // shift existing bits in ch to make sure for new bits

		// Move bits we want all the way to the right
		bufCH := buf[startByte] >> (8 - grab - startBit)
		mask := byte(0xFF) >> (8 - grab) // grab/mask just bits of interest
		ch |= (bufCH & mask)             // add buf's bits to ch

		// Move to next bit, and next byte if needed
		startBit += grab
		if startBit >= 8 {
			startByte++
			startBit = 0 // 8 - startBit
		}

		// Are we done with current ch? If so, get ShortChar and reset
		left = left - grab
		if left == 0 {
			str += string(ShortChars[ch])
			size++
			left = 6
			ch = 0
		}
	}

	return str
}

// xid:  /GROUPS/gid/RESOURCES/rid/meta|versions/vid
func ParseXID(xid string) ([]string, error) {
	xid = strings.TrimSpace(xid)

	if len(xid) == 0 {
		return nil, fmt.Errorf("XID can't be an empty string")
	}

	if xid[0] != '/' {
		return nil, fmt.Errorf("XID %q must start with /", xid)
	}

	parts := strings.SplitN(xid[1:], "/", 7) // one extra
	if len(parts) > 6 {
		return nil, fmt.Errorf("XID %q is too long", xid)
	}
	for i, str := range parts {
		if str == "" {
			return nil, fmt.Errorf("XID %q has an empty part at position %d",
				xid, i+1)
		}
	}
	return parts, nil
}

// xid:  /GROUPS/RESOURCES[/versions]
func ParseXIDTemplate(xid string) ([]string, error) {
	if len(xid) == 0 {
		return nil, fmt.Errorf("XID can't be an empty string")
	}

	if xid[0] != '/' {
		return nil, fmt.Errorf("XID %q must start with /", xid)
	}

	parts := strings.SplitN(xid[1:], "/", 4) // one extra
	if len(parts) > 3 {
		return nil, fmt.Errorf("XID %q is too long", xid)
	}
	return parts, nil
}

func XID2Abstract(xid string) (string, error) {
	// GROUPS/gid/RESOURCES/rid/versions/vID
	parts, err := ParseXID(xid)
	if err != nil {
		return "", err
	}
	res := "/"
	if len(parts) > 0 {
		res += parts[0]
	}
	if len(parts) > 2 {
		res += "/" + parts[2]
	}
	if len(parts) > 4 {
		res += "/" + parts[4]
	}
	return res, nil
}

func PrettyPrint(object any, prefix string, indent string) string {
	// objType := reflect.ValueOf(object).Type().String()
	buf, _ := json.MarshalIndent(object, prefix, indent)
	return string(buf)
}

func DownloadURL(urlPath string) ([]byte, error) {
	res, err := http.Get(urlPath)
	if err == nil {
		var data []byte
		data, err = io.ReadAll(res.Body)
		res.Body.Close()

		if err == nil {
			if res.StatusCode/100 != 2 {
				err = fmt.Errorf("%s:\n%s", res.Status, string(data))
			} else {
				return data, nil
			}
		}
	}

	return nil, fmt.Errorf("Error retrieving %q: %s", urlPath, err)
}
