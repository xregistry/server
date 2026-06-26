package common

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
)

const NO_INDEX = -1
const WILDCARD_INDEX = -2

type PropPath struct {
	Parts []PropPart

	// Are any parts a wildcard (*) ?
	// Note, this info will never be saved in the DB. This is just
	// FYI data for server processing of requests. This is used to
	// avoid having to loop over all parts just to know if a wildcard
	// is being used.
	HasWild bool
}

func NewPPP(prop string) *PropPath {
	return NewPP().P(prop)
}

func NewPP() *PropPath {
	return &PropPath{}
}

func (pp *PropPath) String() string {
	return pp.UI()
}

func (pp *PropPath) Len() int {
	if pp == nil {
		return 0
	}
	return len(pp.Parts)
}

func (pp *PropPath) Top() string {
	if pp == nil || len(pp.Parts) == 0 {
		return ""
	}
	return pp.Parts[0].Text
}

func (pp *PropPath) Last() *PropPart {
	if len(pp.Parts) == 0 {
		return nil
	}
	return &(pp.Parts[len(pp.Parts)-1])
}

func (pp *PropPath) Bottom() string {
	if len(pp.Parts) == 0 {
		return ""
	}
	last := pp.Parts[len(pp.Parts)-1]
	if last.Index >= 0 {
		return "" // maybe error one day??
	}
	return last.Text
}

func (pp *PropPath) First() *PropPath {
	if pp.Len() == 0 {
		return nil
	}
	return &PropPath{
		Parts: pp.Parts[:1],
	}
}

func (pp *PropPath) Next() *PropPath {
	if pp.Len() == 1 {
		return nil
	}
	return &PropPath{
		Parts: pp.Parts[1:],
	}
}

func (pp *PropPath) FromIndex(i int) *PropPath {
	if i > pp.Len() {
		return nil
	}
	return &PropPath{
		Parts: pp.Parts[i:],
	}
}

func MustPropPathFromPath(str string) *PropPath {
	pp, _ := PropPathFromPath(str)
	return pp
}

func PropPathFromPath(str string) (*PropPath, error) {
	str = strings.Trim(str, "/")
	if str == "" {
		return &PropPath{}, nil
	}
	parts := strings.Split(str, "/")
	res := &PropPath{}
	for _, p := range parts {
		res.Parts = append(res.Parts, PropPart{
			Text:  p,
			Index: NO_INDEX,
		})
	}
	return res, nil
}

func (pp *PropPath) Path() string {
	if pp == nil {
		return ""
	}
	res := strings.Builder{}
	for _, part := range pp.Parts {
		if res.Len() > 0 {
			res.WriteRune('/')
		}
		res.WriteString(part.Text)
	}
	return res.String()
}

func (pp *PropPath) DB() string {
	if pp == nil {
		return ""
	}
	res := strings.Builder{}
	for _, part := range pp.Parts {
		if res.Len() != 0 {
			// res.WriteRune(DB_IN)
		}
		if part.Index >= 0 {
			res.WriteRune(DB_INDEX)
		}
		res.WriteString(part.Text)
		res.WriteRune(DB_IN)
	}
	return res.String()
}

// Same as DB() but w/o the trailing ","
func (pp *PropPath) Abstract() string {
	if pp == nil {
		return ""
	}
	res := strings.Builder{}
	for _, part := range pp.Parts {
		if part.Index >= 0 {
			res.WriteRune(DB_INDEX)
		} else if res.Len() != 0 {
			res.WriteRune(DB_IN)
		}
		res.WriteString(part.Text)
	}
	return res.String()
}

func MustPropPathFromDB(str string) *PropPath {
	pp, err := PropPathFromDB(str)
	PanicIf(err != nil, "Bad pp: %s", str)
	return pp
}

func PropPathFromDB(str string) (*PropPath, error) {
	res := &PropPath{}

	if len(str) == 0 || str[0] == '.' || str[0] == '#' {
		str = strings.TrimRight(str, string(DB_IN))
		res.Parts = append(res.Parts, PropPart{
			Text:  str,
			Index: NO_INDEX,
		})
	} else {
		// Assume what's in the DB is correct, so no error checking
		parts := strings.Split(str, string(DB_IN))
		PanicIf(len(parts) == 1 && parts[0] == "", "Empty str")
		for _, p := range parts {
			if p == "" {
				continue // should only happen on trailing DB_IN
			}
			index := NO_INDEX
			if p[0] == DB_INDEX {
				p = p[1:]
				var err error
				index, err = strconv.Atoi(p)
				PanicIf(err != nil, "%q isnt an int: %s", p, err)
			}
			res.Parts = append(res.Parts, PropPart{
				Text:  p,
				Index: index,
			})
		}
	}

	return res, nil
}

var stateTable = [][]string{
	// TODO: switch to a-z instead of 0-9 for state char if we need more than 10
	// Each entry is made up of: "nextState" + "Actions" (1 or more)
	// NextState of "`" means stop processing
	// Spaces in "Actions" is just to make the table look pretty
	//
	// a-z  0-9    -     _     .     [     ]     '     \0   else  *     "
	// A     B     C     D     E     F     G     H     I     J    K     L
	// 0     1     2     3     4     5     6     7     8     9    10    11
	{"cB", "`U", "`U", "cB", "`U", "j ", "`U", "`U", "`U", "`U", "cB", "`U"}, // a-nothing
	{"cB", "cB", "`U", "cB", "`U", "`U", "`U", "`U", "`U", "`U", "cB", "`U"}, // b-seen attr[
	{"cB", "cB", "cB", "cB", "bS", "dS", "`U", "`U", "`S", "`U", "cB", "`U"}, // c-in attr..
	{"`P", "eB", "`U", "`U", "`U", "`U", "`U", "g ", "`U", "`U", "eB", "k "}, // d-seen attr[
	{"`P", "eB", "`U", "`U", "`U", "`U", "fN", "`U", "`U", "`U", "eB", "`U"}, // e-in [..
	{"`U", "`U", "`U", "`U", "bA", "d ", "`U", "`U", "` ", "`U", "`U", "`U"}, // f-seen ..]
	{"hB", "hB", "`U", "`U", "`U", "`U", "`U", "`U", "`U", "`U", "hB", "`U"}, // g-seen ['
	{"hB", "hB", "hB", "hB", "hB", "`U", "`U", "i ", "`U", "`U", "hB", "hB"}, // h-in ['..
	{"`U", "`U", "`U", "`U", "`U", "`U", "fS", "`U", "`U", "`U", "`U", "`U"}, // i-seen ['..'
	{"`Q", "`U", "`U", "`U", "`U", "`U", "`U", "g ", "`U", "`U", "`Q", "k "}, // j-seen root [
	{"lB", "lB", "`U", "`U", "`U", "`U", "`U", "lB", "`U", "`U", "lB", "`U"}, // k-seen ["
	{"lB", "lB", "lB", "lB", "lB", "`U", "`U", "lB", "`U", "`U", "lB", "m "}, // l-in ["..
	{"`U", "`U", "`U", "`U", "`U", "`U", "fS", "`U", "`U", "`U", "`U", "`U"}, // m-seen [".."
	//
	// B = buffer char
	// S = end of string
	// N = end of array index
	// P, Q, U are all error actions
}
var stateTableSize = byte(len(stateTable))

// Mapping of "char" -> "column" in state table
var OLDch2Col = map[byte]int{}
var ch2Col = [256]int{} // array lookup is faster than map

func init() {
	for i := 0; i < 256; i++ {
		if i >= 'a' && i <= 'z' {
			ch2Col[i] = 0
		} else if i >= 'A' && i <= 'Z' {
			ch2Col[i] = 0
		} else if i >= '0' && i <= '9' {
			ch2Col[i] = 1
		} else if i == '-' {
			ch2Col[i] = 2
		} else if i == '_' {
			ch2Col[i] = 3
		} else if i == '.' {
			ch2Col[i] = 4
		} else if i == '[' {
			ch2Col[i] = 5
		} else if i == ']' {
			ch2Col[i] = 6
		} else if i == '\'' {
			ch2Col[i] = 7
		} else if i == 0 {
			ch2Col[i] = 8
		} else if i == '*' {
			ch2Col[i] = 10
		} else if i == '"' {
			ch2Col[i] = 11
		} else {
			ch2Col[i] = 9
		}
	}

	// Old way

	for ch := 'a'; ch <= 'z'; ch++ {
		OLDch2Col[byte(ch)] = 0
		OLDch2Col[byte('A'+(ch-'a'))] = 0
	}
	for ch := '0'; ch <= '9'; ch++ {
		OLDch2Col[byte(ch)] = 1
	}
	OLDch2Col['-'] = 2
	OLDch2Col['_'] = 3
	OLDch2Col['.'] = 4
	OLDch2Col['['] = 5
	OLDch2Col[']'] = 6
	OLDch2Col['\''] = 7
	OLDch2Col[0] = 8

	OLDch2Col['*'] = 10
	OLDch2Col['"'] = 11
}

func MustPropPathFromUI(str string) *PropPath {
	pp, _ := PropPathFromUI(str)
	return pp
}

func PropPathFromUI(str string) (*PropPath, error) {
	res := &PropPath{}

	if len(str) == 0 {
		return res, nil
	}

	if str[0] == '#' {
		// I believe that we treat props that start with "#" as special.
		// They're just for internal attributes. Just use the rest of the
		// string (unparsed) as the Text
		res.Parts = append(res.Parts, PropPart{
			Text:  str,
			Index: NO_INDEX,
		})
	} else {
		chIndex := 0
		ch := str[chIndex]
		buf := strings.Builder{}
		hasStar := false
		for state := 0; state != 255; { // 255(`) = "exit" in stateTable
			col := ch2Col[ch]
			// Still not sure which is faster - or if it matters
			// col, ok := OLDch2Col[ch]
			// if !ok { // "else" column
			// col = 9
			// }

			actions := stateTable[state][col]
			PanicIf(actions[0] < '`' || actions[0] > '`'+stateTableSize,
				"Bad state: Ch:%x(%c) Col:%v Actions:%q", ch, ch, col, actions)
			state = int(actions[0] - 'a')
			for i := 1; i < len(actions); i++ { // 1=Skip next state char
				switch actions[i] {
				case ' ': // ignore
				case 'B': // buffer it
					buf.WriteRune(rune(ch))
					if ch == '*' {
						hasStar = true
					}
				case 'S': // end of string part
					// if not in [] and buf has more than just *, err
					if state != 5 && buf.Len() > 1 && hasStar {
						return nil, fmt.Errorf("Unexpected \"*\" in %q",
							buf.String())
					}
					res.Parts = append(res.Parts, PropPart{
						Text:  buf.String(),
						Index: NO_INDEX,

						// is wildcard unless we're in []
						IsWild: hasStar && state != 5,
					})

					if hasStar && state != 5 { // bubble up if true
						res.HasWild = true
					}

					buf.Reset()
					hasStar = false
				case 'N': // end of array index part
					strBuf := buf.String()

					index := 0
					var err error

					// [*] is a special case
					if len(strBuf) == 1 && hasStar {
						index = WILDCARD_INDEX
						res.HasWild = true // bubble up if true
					} else if len(strBuf) > 1 && hasStar {
						return nil, fmt.Errorf("Unexpected \"*\" in %q",
							buf.String())
					} else {
						index, err = strconv.Atoi(strBuf)
						if err != nil {
							return nil, fmt.Errorf("%q should be an integer",
								strBuf)
						}
					}
					res.Parts = append(res.Parts, PropPart{
						Text:   strBuf,
						Index:  index,
						IsWild: hasStar,
					})
					buf.Reset()
					hasStar = false
				case 'P': // error case
					return nil,
						fmt.Errorf("Expecting an integer at pos %d in %q",
							chIndex+1, str)
				case 'Q': // error case
					return nil, fmt.Errorf("Expecting a ' at pos %d in %q",
						chIndex+1, str)
				case 'U': // error case
					if ch == 0 {
						return nil,
							fmt.Errorf("Unexpected end of property in %q", str)
					} else {
						qCH := '"'
						if ch == '"' {
							qCH = '\''
						}
						return nil, fmt.Errorf("Unexpected %c%c%c in %q "+
							"at pos %d",
							qCH, ch, qCH, str, chIndex+1)
					}
				}
			}
			// Move to next char
			chIndex++
			if chIndex < len(str) {
				ch = str[chIndex]
			} else {
				ch = 0
			}
		}
	}

	return res, nil
}

func (pp *PropPath) UI() string {
	if pp == nil {
		return ""
	}
	res := strings.Builder{}
	for _, part := range pp.Parts {
		if part.Index >= 0 {
			res.WriteString(fmt.Sprintf("[%d]", part.Index))
		} else {
			if res.Len() > 0 {
				if strings.Contains(part.Text, string(UX_IN)) {
					res.WriteString("['" + part.Text + "']")
				} else {
					res.WriteString(string(UX_IN) + part.Text)
				}
			} else {
				res.WriteString(part.Text)
			}
		}
	}
	return res.String()
}

func (pp *PropPath) I(i int) *PropPath {
	return pp.Index(i)
}

func (pp *PropPath) Index(i int) *PropPath {
	newPP := NewPP()
	newPP.Parts = append(pp.Parts, PropPart{
		Text:  fmt.Sprintf("%d", i),
		Index: i,
	})
	return newPP
}

func (pp *PropPath) P(prop string) *PropPath {
	return pp.Prop(prop)
}

func (pp *PropPath) Prop(prop string) *PropPath {
	newPP := NewPP()
	newPP.Parts = append(pp.Parts, PropPart{
		Text:  prop,
		Index: NO_INDEX,
	})
	return newPP
}

func (pp *PropPath) Clone() *PropPath {
	newPP := NewPP()
	newPP.Parts = append([]PropPart{}, pp.Parts...)
	return newPP
}

func (pp *PropPath) Append(addPP *PropPath) *PropPath {
	newPP := NewPP()
	newPP.Parts = append(pp.Parts, addPP.Parts...)
	return newPP
}

func (pp *PropPath) RemoveLast() *PropPath {
	newPP := NewPP()
	newPP.Parts = append([]PropPart{}, pp.Parts[:len(pp.Parts)-1]...)
	return newPP
}

func (pp *PropPath) Equals(other *PropPath) bool {
	return reflect.DeepEqual(pp, other)
}

func (pp *PropPath) HasPrefix(other *PropPath) bool {
	for i, p := range other.Parts {
		if i >= pp.Len() {
			return false
		}
		if !reflect.DeepEqual(pp.Parts[i], p) {
			return false
		}
	}
	return true
}

type PropPart struct {
	Text   string
	Index  int
	IsWild bool // only true when [*] or .* not ['*']
}

func (pp *PropPart) ToInt() int {
	val, err := strconv.Atoi(pp.Text)
	PanicIf(err != nil, "Error parsing int %q: %s", pp.Text, err)
	return val
}

func (pp *PropPart) IsIndex() bool {
	return pp.Index >= 0
}

// Value, Found, Error
func ObjectGetProp(obj any, pp *PropPath) (any, bool, error) {
	return NestedGetProp(obj, pp, NewPP())
}

// Value, Found, Error
func NestedGetProp(obj any, pp *PropPath, prev *PropPath) (any, bool, error) {
	if log.GetVerbose() > 2 {
		log.VPrintf(0, "ObjectGetProp: %q\nobj:\n%s", pp.UI(), ToJSON(obj))
	}
	if pp == nil || pp.Len() == 0 {
		return obj, true, nil
	}
	if IsNil(obj) {
		return nil, false,
			fmt.Errorf("Can't traverse into nothing: %s", prev.UI())
	}

	objValue := reflect.ValueOf(obj)
	part := pp.Parts[0]
	if index := part.Index; index >= 0 {
		// Is an array
		if objValue.Kind() != reflect.Slice {
			return nil, false,
				fmt.Errorf("Can't index into non-array: %s", prev.UI())
		}
		if index < 0 || index >= objValue.Len() {
			return nil, false,
				fmt.Errorf("Array reference %q out of bounds: "+
					"(max:%d-1)", prev.Append(pp.First()).UI(), objValue.Len())
		}
		objValue = objValue.Index(index)
		if objValue.IsValid() {
			obj = objValue.Interface()
		} else {
			panic("help") // Should never get here
			obj = nil
		}
		return NestedGetProp(obj, pp.Next(), prev.Append(pp.First()))
	}

	// Is map/object
	if objValue.Kind() != reflect.Map {
		return nil, false, fmt.Errorf("Can't reference a non-map/object: %s",
			prev.UI())
	}
	if objValue.Type().Key().Kind() != reflect.String {
		return nil, false, fmt.Errorf("key of %q must be a string, not %s",
			prev.UI(), objValue.Type().Key().Kind())
	}

	objValue = objValue.MapIndex(reflect.ValueOf(pp.Top()))
	if objValue.IsValid() {
		obj = objValue.Interface()
	} else {
		if pp.Next().Len() == 0 {
			return nil, false, nil
		}
		obj = nil
	}
	return NestedGetProp(obj, pp.Next(), prev.Append(pp.First()))
}

// Given a PropPath and a value this will add the necessary golang data
// structures to 'obj' to materialize PropPath and set the appropriate
// fields to 'val'
func ObjectSetProp(obj map[string]any, pp *PropPath, val any) error {
	log.VPrintf(4, "ObjectSetProp(%s=%v)", pp, val)
	if pp.Len() == 0 && IsNil(val) {
		// A bit of a special case, not 100% sure if this is ok.
		// Treat nil val as a request to delete all properties.
		// e.g. obj={}
		for k, _ := range obj {
			delete(obj, k)
		}
		return nil
	}
	PanicIf(pp.Len() == 0, "Can't be zero w/non-nil val")

	_, err := MaterializeProp(obj, pp, val, nil)
	return err
}

func MaterializeProp(current any, pp *PropPath, val any, prev *PropPath) (any, error) {
	log.VPrintf(4, ">Enter: MaterializeProp(%s)", pp)
	log.VPrintf(4, "<Exit: MaterializeProp")

	// current is existing value, used for adding to maps/arrays
	if pp == nil {
		return val, nil
	}

	var ok bool
	var err error

	if prev == nil {
		prev = NewPP()
	}

	part := pp.Parts[0]
	if index := part.Index; index >= 0 {
		// Is an array
		// TODO look for cases where Kind(val) == array too - maybe?
		var daArray []any

		if current != nil {
			daArray, ok = current.([]any)
			if !ok {
				return nil, fmt.Errorf("attribute %q isn't an array",
					prev.Append(pp.First()).UI())
			}
		}

		// Resize if needed
		if diff := (1 + index - len(daArray)); diff > 0 {
			daArray = append(daArray, make([]any, diff)...)
		}

		// Trim the end of the array if there are nil's
		daArray[index], err = MaterializeProp(daArray[index], pp.Next(), val,
			prev.Append(pp.First()))
		for len(daArray) > 0 && daArray[len(daArray)-1] == nil {
			daArray = daArray[:len(daArray)-1]
		}
		return daArray, err
	}

	// Is a map/object
	// TODO look for cases where Kind(val) == obj/map too - maybe?

	daMap := map[string]any{}
	if !IsNil(current) {
		daMap, ok = current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("current isn't a map: %T", current)
		}
	}

	res, err := MaterializeProp(daMap[pp.Top()], pp.Next(), val,
		prev.Append(pp.First()))
	if err != nil {
		return nil, err
	}
	if IsNil(res) {
		delete(daMap, pp.Top())
	} else {
		daMap[pp.Top()] = res
	}

	return daMap, nil
}
