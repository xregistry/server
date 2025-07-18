package common

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
	"unicode"

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

func Any2String(val any) string {
	b := (val).([]byte)
	return string(b)
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

func PtrInt(i int) *int {
	return &i
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

func EnvBool(name string, def bool) bool {
	val := os.Getenv(name)
	if val != "" {
		def = strings.EqualFold(val, "true")
	}
	return def
}

func EnvInt(name string, def int) int {
	val := os.Getenv(name)
	if val != "" {
		// Silently ignore errors for now
		valInt, err := strconv.Atoi(val)
		if err != nil {
			def = valInt
		}
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

func JSONEscape(obj interface{}) string {
	buf, _ := json.Marshal(obj)
	return string(buf[1 : len(buf)-1])
}

func ToJSON(obj interface{}) string {
	if obj != nil && reflect.TypeOf(obj).String() == "*errors.errorString" {
		obj = obj.(error).Error()
	}

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

	re = regexp.MustCompile(`\s"labels": {\s*},*`)
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
	str := fmt.Sprintf(`"(https?://[^"\n]*?)"`)
	re := regexp.MustCompile(str)
	repl := fmt.Sprintf(`"<a href="$1?%s">$1?%s</a>"`,
		r.URL.RawQuery, r.URL.RawQuery)

	return re.ReplaceAll(buf, []byte(repl))
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
		if err2 := ValidateJSONWithPath(buf, v); err2 != nil {
			return err2
		}
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

type Object map[string]any

func IncomingObj2Map(incomingObj Object) (map[string]Object, error) {
	result := map[string]Object{}
	for id, obj := range incomingObj {
		oV := reflect.ValueOf(obj)
		if oV.Kind() != reflect.Map ||
			oV.Type().Key().Kind() != reflect.String {

			return nil, fmt.Errorf("Body must be a map of id->Entity, near %q",
				id)
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

	t = t.UTC()

	// str = t.Format("2006-01-02T15:04:05.000000000Z07:00") // trailing 0's
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

func AddQuery(urlPath string, newQuery string) string {
	if newQuery == "" {
		return urlPath
	}

	base, query, _ := strings.Cut(urlPath, "?")
	if query == "" {
		query = "?"
	} else {
		query = "?" + query + "&"
	}
	return base + query + newQuery
}

// Custom json parser

// ONLY USE THIS TO GET BETTER ERRORS
// Always call the normal Unmarshal func first to see if things work.
// This will not call any custom Unmarshal funcs

// ValidateJSONWithPath validates JSON data against a Go value, reporting detailed errors with paths
// for syntax errors, type mismatches, and unknown fields. It uses a custom parser instead of encoding/json.
// It handles struct fields without json tags by using the field name as the JSON key if exported,
// supports case-insensitive matching for JSON keys, and reports type mismatches with the Go type as expected.
// Error messages include "path " before the path, use a single dot for top-level paths (e.g., ".foo"),
func ValidateJSONWithPath(data []byte, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("v must be a non-nil pointer")
	}
	goType := rv.Elem().Type()
	goValue := rv.Elem()

	// Parse JSON manually
	parser := newParser(data)
	value, err := parser.parse()
	if err != nil {
		return fmt.Errorf("path '%s': %v", parser.currentPath(), err)
	}

	// Validate the parsed value against the Go type
	return validateJSON(value, goType, goValue, "")
}

// parser holds the state for the custom JSON parser
type parser struct {
	data []byte
	pos  int
	path []string // Path stack for error reporting
}

// newParser creates a new parser for the given JSON data
func newParser(data []byte) *parser {
	return &parser{data: data, pos: 0, path: []string{}}
}

// currentPath returns the current path as a string
func (p *parser) currentPath() string {
	// if len(p.path) == 0 {
	// return "."
	// }
	return strings.Join(p.path, "")
}

// pushPath adds a path component (e.g., ".field" or "[0]")
func (p *parser) pushPath(component string) {
	p.path = append(p.path, component)
}

// popPath removes the last path component
func (p *parser) popPath() {
	if len(p.path) > 0 {
		p.path = p.path[:len(p.path)-1]
	}
}

// parse parses the JSON data and returns the parsed value or an error
func (p *parser) parse() (interface{}, error) {
	p.skipWhitespace()
	if p.pos >= len(p.data) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	switch p.data[p.pos] {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		return p.parseString()
	case 't', 'f':
		return p.parseBool()
	case 'n':
		return p.parseNull()
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return p.parseNumber()
	default:
		return nil, fmt.Errorf("unexpected character '%c'", p.data[p.pos])
	}
}

// parseObject parses a JSON object
func (p *parser) parseObject() (map[string]interface{}, error) {
	p.pos++ // Skip '{'
	result := make(map[string]interface{})
	p.skipWhitespace()

	if p.pos < len(p.data) && p.data[p.pos] == '}' {
		p.pos++ // Empty object
		return result, nil
	}

	for {
		// Parse key
		key, err := p.parseString()
		if err != nil {
			return nil, fmt.Errorf("parsing object key: %v", err)
		}
		keyStr, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("object key must be a string, got %T", key)
		}
		p.pushPath("." + keyStr)

		// Expect ':'
		p.skipWhitespace()
		if p.pos >= len(p.data) || p.data[p.pos] != ':' {
			return nil, fmt.Errorf("expected ':' after key")
		}
		p.pos++

		// Parse value
		value, err := p.parse()
		if err != nil {
			return nil, err
		}
		result[keyStr] = value
		p.popPath()

		// Expect ',' or '}'
		p.skipWhitespace()
		if p.pos >= len(p.data) {
			return nil, fmt.Errorf("unexpected end of input in object")
		}
		if p.data[p.pos] == '}' {
			p.pos++
			return result, nil
		}
		if p.data[p.pos] != ',' {
			return nil, fmt.Errorf("expected ',' or '}' after value, got '%c'", p.data[p.pos])
		}
		p.pos++
		p.skipWhitespace()
	}
}

// parseArray parses a JSON array
func (p *parser) parseArray() ([]interface{}, error) {
	p.pos++ // Skip '['
	result := []interface{}{}
	p.skipWhitespace()

	if p.pos < len(p.data) && p.data[p.pos] == ']' {
		p.pos++ // Empty array
		return result, nil
	}

	for i := 0; ; i++ {
		p.pushPath(fmt.Sprintf("[%d]", i))
		value, err := p.parse()
		if err != nil {
			return nil, err
		}
		result = append(result, value)
		p.popPath()

		p.skipWhitespace()
		if p.pos >= len(p.data) {
			return nil, fmt.Errorf("unexpected end of input in array")
		}
		if p.data[p.pos] == ']' {
			p.pos++
			return result, nil
		}
		if p.data[p.pos] != ',' {
			return nil, fmt.Errorf("expected ',' or ']' after array element, got '%c'", p.data[p.pos])
		}
		p.pos++
		p.skipWhitespace()
	}
}

// parseString parses a JSON string
func (p *parser) parseString() (interface{}, error) {
	if p.pos >= len(p.data) || p.data[p.pos] != '"' {
		return nil, fmt.Errorf("expected string starting with '\"', got '%c' instead", p.data[p.pos])
	}
	p.pos++

	var sb strings.Builder
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if c == '"' {
			p.pos++
			return sb.String(), nil
		}
		if c == '\\' {
			p.pos++
			if p.pos >= len(p.data) {
				return nil, fmt.Errorf("unexpected end of input in string escape")
			}
			switch p.data[p.pos] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				sb.WriteByte(p.data[p.pos])
				p.pos++
			case 'u':
				p.pos++
				if p.pos+4 > len(p.data) {
					return nil, fmt.Errorf("incomplete unicode escape")
				}
				runeVal, err := strconv.ParseInt(string(p.data[p.pos:p.pos+4]), 16, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid unicode escape: %v", err)
				}
				sb.WriteRune(rune(runeVal))
				p.pos += 4
			default:
				return nil, fmt.Errorf("invalid escape character '%c'", p.data[p.pos])
			}
		} else {
			sb.WriteByte(c)
			p.pos++
		}
	}
	return nil, fmt.Errorf("unterminated string")
}

// parseBool parses a JSON boolean
func (p *parser) parseBool() (interface{}, error) {
	if p.pos+4 <= len(p.data) && string(p.data[p.pos:p.pos+4]) == "true" {
		p.pos += 4
		return true, nil
	}
	if p.pos+5 <= len(p.data) && string(p.data[p.pos:p.pos+5]) == "false" {
		p.pos += 5
		return false, nil
	}
	return nil, fmt.Errorf("invalid boolean")
}

// parseNull parses a JSON null
func (p *parser) parseNull() (interface{}, error) {
	if p.pos+4 <= len(p.data) && string(p.data[p.pos:p.pos+4]) == "null" {
		p.pos += 4
		return nil, nil
	}
	return nil, fmt.Errorf("invalid null")
}

// parseNumber parses a JSON number
func (p *parser) parseNumber() (interface{}, error) {
	start := p.pos
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if !strings.ContainsAny(string(c), "0123456789.-eE+") {
			break
		}
		p.pos++
	}
	numStr := string(p.data[start:p.pos])
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number '%s': %v", numStr, err)
	}
	return num, nil
}

// skipWhitespace skips whitespace characters
func (p *parser) skipWhitespace() {
	for p.pos < len(p.data) && unicode.IsSpace(rune(p.data[p.pos])) {
		p.pos++
	}
}

// validateJSON recursively validates JSON data against a Go type, tracking the path.
func validateJSON(jsonData interface{}, goType reflect.Type, goValue reflect.Value, path string) error {
	// if path == "" {
	// path = "."
	// }

	// Handle pointers
	for goType.Kind() == reflect.Ptr {
		if goValue.IsValid() && goValue.IsNil() {
			// Allocate a new instance if the pointer is nil
			goValue = reflect.New(goType.Elem())
		}
		goType = goType.Elem()
		if goValue.IsValid() {
			goValue = goValue.Elem()
		}
	}

	// Helper function to get JSON type name for error messages
	getJSONTypeName := func(data interface{}) string {
		switch data.(type) {
		case map[string]interface{}:
			return "object"
		case []interface{}:
			return "array"
		case float64:
			return "number"
		case string:
			return "string"
		case bool:
			return "boolean"
		case nil:
			return "null"
		default:
			return fmt.Sprintf("unknown (%T)", data)
		}
	}

	switch jsonData := jsonData.(type) {
	case map[string]interface{}:
		// Handle structs or maps
		switch goType.Kind() {
		case reflect.Struct:
			return validateStruct(jsonData, goType, goValue, path)
		case reflect.Map:
			return validateMap(jsonData, goType, goValue, path)
		default:
			return fmt.Errorf("path '%s': expected %q, got %q", path, goType.Kind(), getJSONTypeName(jsonData))
		}

	case []interface{}:
		// Handle slices or arrays
		switch goType.Kind() {
		case reflect.Slice, reflect.Array:
			return validateSlice(jsonData, goType, goValue, path)
		default:
			return fmt.Errorf("path '%s': expected %q, got %q", path, goType.Kind(), getJSONTypeName(jsonData))
		}

	case float64:
		// JSON numbers are parsed as float64
		switch goType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			return nil
		default:
			return fmt.Errorf("path '%s': expected %q, got %q", path, goType.Kind(), getJSONTypeName(jsonData))
		}

	case string:
		if goType.Kind() == reflect.String {
			return nil
		}
		return fmt.Errorf("path '%s': expected %q, got %q", path, goType.Kind(), getJSONTypeName(jsonData))

	case bool:
		if goType.Kind() == reflect.Bool {
			return nil
		}
		return fmt.Errorf("path '%s': expected %q, got %q", path, goType.Kind(), getJSONTypeName(jsonData))

	case nil:
		if goType.Kind() == reflect.Ptr || goType.Kind() == reflect.Interface || goType.Kind() == reflect.Slice || goType.Kind() == reflect.Map {
			return nil
		}
		return fmt.Errorf("path '%s': expected %q, got %q", path, goType.Kind(), getJSONTypeName(jsonData))

	default:
		return fmt.Errorf("path '%s': expected %q, got %q", path, goType.Kind(), getJSONTypeName(jsonData))
	}
}

func validateStruct(jsonData map[string]interface{}, goType reflect.Type, goValue reflect.Value, path string) error {
	// Build map of valid field names (case-sensitive) and their types
	validFields := make(map[string]reflect.Type)
	// Also build a case-insensitive map to detect ambiguities
	lowerToFields := make(map[string][]string)
	for i := 0; i < goType.NumField(); i++ {
		field := goType.Field(i)
		// Skip unexported fields
		if field.Name[0] < 'A' || field.Name[0] > 'Z' {
			continue
		}
		jsonTag := field.Tag.Get("json")
		name := field.Name // Default to field name
		if jsonTag != "" && jsonTag != "-" {
			name = strings.Split(jsonTag, ",")[0]
		}
		validFields[name] = field.Type
		// Track for case-insensitive matching
		lowerName := strings.ToLower(name)
		lowerToFields[lowerName] = append(lowerToFields[lowerName], name)
	}

	// Check for ambiguous field names (case-insensitive)
	for lowerName, fields := range lowerToFields {
		if len(fields) > 1 {
			return fmt.Errorf("ambiguous field names %v map to the same lowercase name %q", fields, lowerName)
		}
	}

	// Check for unknown fields (case-insensitive)
	for key := range jsonData {
		lowerKey := strings.ToLower(key)
		found := false
		for fieldName := range validFields {
			if strings.ToLower(fieldName) == lowerKey {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown field %q at path '%s'", key,
				path+"."+key)
		}
	}

	// Recursively validate each field
	for key, value := range jsonData {
		lowerKey := strings.ToLower(key)
		var fieldType reflect.Type
		// Find the matching field (case-insensitive)
		for name, fType := range validFields {
			if strings.ToLower(name) == lowerKey {
				fieldType = fType
				break
			}
		}
		if fieldType == nil {
			continue // Already caught as unknown
		}
		var fieldValue reflect.Value
		if goValue.IsValid() {
			for i := 0; i < goType.NumField(); i++ {
				field := goType.Field(i)
				name := field.Name
				if field.Tag.Get("json") != "" && field.Tag.Get("json") != "-" {
					name = strings.Split(field.Tag.Get("json"), ",")[0]
				}
				if strings.ToLower(name) == lowerKey {
					fieldValue = goValue.Field(i)
					break
				}
			}
		}
		if err := validateJSON(value, fieldType, fieldValue, path+"."+key); err != nil {
			return err
		}
	}
	return nil
}

func validateMap(jsonData map[string]interface{}, goType reflect.Type, goValue reflect.Value, path string) error {
	if goType.Key().Kind() != reflect.String {
		return fmt.Errorf("map key must be string, got %q", goType.Key().Kind())
	}
	valueType := goType.Elem()
	var mapValue reflect.Value
	if goValue.IsValid() && goValue.IsNil() {
		goValue = reflect.MakeMap(goType)
	}
	if goValue.IsValid() {
		mapValue = goValue
	}
	for key, value := range jsonData {
		var elemValue reflect.Value
		if mapValue.IsValid() {
			v := mapValue.MapIndex(reflect.ValueOf(key))
			if v.IsValid() {
				elemValue = v
			}
		}
		if err := validateJSON(value, valueType, elemValue, fmt.Sprintf("%s[%q]", path, key)); err != nil {
			return err
		}
	}
	return nil
}

func validateSlice(jsonData []interface{}, goType reflect.Type, goValue reflect.Value, path string) error {
	elemType := goType.Elem()
	var sliceValue reflect.Value
	if goType.Kind() == reflect.Slice && goValue.IsValid() && goValue.IsNil() {
		goValue = reflect.MakeSlice(goType, 0, len(jsonData))
	}
	if goValue.IsValid() {
		sliceValue = goValue
	}
	for i, value := range jsonData {
		var elemValue reflect.Value
		if sliceValue.IsValid() && i < sliceValue.Len() {
			elemValue = sliceValue.Index(i)
		}
		if err := validateJSON(value, elemType, elemValue, fmt.Sprintf("%s[%d]", path, i)); err != nil {
			return err
		}
	}
	return nil
}

// Assumes it's an object - for now
/*
func PrettyPrintJSON(buf []byte) ([]byte, error) {
	if buf == nil {
		return []byte("{}"), nil
	}
	tmp := map[string]any{}
	err := Unmarshal(buf, &tmp)
	if err != nil {
		return nil, err
	}
	buf, _ = json.MarshalIndent(tmp, "", "  ")
	return buf, nil
}
*/

func RemoveSchema(buf []byte) ([]byte, error) {
	obj, err := ParseJSONToObject(buf)
	if err != nil {
		return nil, err
	}

	ordered, ok := obj.(*OrderedMap)
	if !ok {
		return buf, nil
	}

	if ordered.Values["$schema"] == nil {
		return buf, nil
	}

	delete(ordered.Values, "$schema")
	for i, v := range ordered.Keys {
		if v == "$schema" {
			ordered.Keys = append(ordered.Keys[:i], ordered.Keys[i+1:]...)
			break
		}
	}

	return json.Marshal(ordered)
}
