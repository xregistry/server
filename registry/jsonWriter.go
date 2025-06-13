package registry

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

type JsonWriter struct {
	info        *RequestInfo
	indent      string
	collPaths   map[int]string   // [eType] URL path to the root of Colls
	unusedColls map[int][]string // [eType][remaining coll names on this eType]

	results *Result // results of DB query
	Entity  *Entity // Current row in the DB results
	hasData bool

	// Save the defaultVersionID as we serialize the Versions collection
	seenDefaultVid string

	// we sometimes need to force an entity to be next, LIFO order
	cachedEntities [](*Entity)

	didCapModelSpace bool
}

func NewJsonWriter(info *RequestInfo, results *Result) *JsonWriter {
	return &JsonWriter{
		info:        info,
		indent:      "",
		collPaths:   map[int]string{},
		unusedColls: map[int][]string{},
		results:     results,
		hasData:     false,
	}
}

func (jw *JsonWriter) Print(str string) {
	fmt.Fprint(jw.info, str)
	jw.hasData = true
}

func (jw *JsonWriter) Printf(format string, args ...any) {
	fmt.Fprintf(jw.info, format, args...)
	jw.hasData = true
}

func (jw *JsonWriter) OptPrintf(format string, args ...any) {
	if len(args) == 0 || args[len(args)-1].(string) != "" {
		jw.Printf(format, args...)
	}
}

func (jw *JsonWriter) Indent() {
	jw.indent += "  "
}

func (jw *JsonWriter) Outdent() {
	if l := len(jw.indent); l > 1 {
		jw.indent = jw.indent[:l-2]
	} else {
		panic("Outdent!!!")
	}
}

func (jw *JsonWriter) NextEntity() (*Entity, error) {
	// If we have a cached entity, return it instead
	var next *Entity
	var err error

	if next = jw.Pop(); next == nil {
		next, err = readNextEntity(jw.info.tx, jw.results, FOR_READ)
	}
	jw.Entity = next
	return jw.Entity, err
}

func (jw *JsonWriter) Push(e *Entity) {
	jw.cachedEntities = append([](*Entity){e}, jw.cachedEntities...)
}

func (jw *JsonWriter) Pop() *Entity {
	if len(jw.cachedEntities) == 0 {
		return nil
	}
	next := jw.cachedEntities[0]
	jw.cachedEntities = jw.cachedEntities[1:]
	return next
}

// This is called WriteCollectionHeaders because it doesn't just write
// the collection, it also writes the COLLECTIONSurl and COLLECTIONscount
// headers/attributes.
// WriteCollection will do the actual processing of the entities in there.
func (jw *JsonWriter) WriteCollectionHeader(extra string) (string, error) {
	myPlural := jw.Entity.Plural
	baseURL := ""

	inlineCollection := jw.info.ShouldInline(jw.Entity.Abstract)

	if jw.info.DoDocView() && inlineCollection {
		// remove GET's base path
		path := path.Dir(jw.Entity.Path)
		path = path[len(jw.info.Root):]
		if strings.HasPrefix(path, "/") {
			path = path[1:]
		}
		baseURL = "#/" + path
	} else {
		baseURL = jw.info.BaseURL + "/" + path.Dir(jw.Entity.Path)
	}

	jw.Printf("%s\n%s\"%surl\": %q,\n", extra, jw.indent, myPlural, baseURL)
	extra = ""

	count := 0
	var err error

	if !inlineCollection {
		// If we're not inlining this collection then just skip over any
		// Entities in the result that are children, but count them so we
		// still show the PLURALcount attribute for the collection
		myAbstract := "-"
		myPlural := ""

		for jw.Entity != nil {
			if myAbstract == "-" {
				myAbstract = jw.Entity.Abstract
				myPlural = jw.Entity.Plural
			}

			if strings.HasPrefix(jw.Entity.Abstract, myAbstract+string(DB_IN)) {
				// Skip descendants that are not immediate children
				if _, err = jw.NextEntity(); err != nil {
					return "", err
				}
				continue
			}

			if strings.HasPrefix(myAbstract, jw.Entity.Abstract+string(DB_IN)) ||
				jw.Entity.Plural != myPlural {
				// Stop on a new parent or a new sibling collection
				break
			}

			if _, err = jw.NextEntity(); err != nil {
				return "", err
			}

			count++
		}
	} else {
		jw.Printf("%s%q: ", jw.indent, jw.Entity.Plural)
		count, err = jw.WriteCollection()
		if err != nil {
			return "", err
		}
		extra = ",\n"
	}

	jw.Printf("%s%s\"%scount\": %d", extra, jw.indent, myPlural, count)

	return ",", nil
}

func (jw *JsonWriter) WriteCollection() (int, error) {
	jw.Printf("{")
	jw.Indent()

	extra := ""
	myAbstract := "-"
	myPlural := ""
	count := 0

	for jw.Entity != nil {
		if myAbstract == "-" {
			myAbstract = jw.Entity.Abstract
			myPlural = jw.Entity.Plural
		}

		if strings.HasPrefix(jw.Entity.Abstract, myAbstract+string(DB_IN)) {
			// Process a child
			if _, err := jw.NextEntity(); err != nil {
				return count, err
			}
			continue
		}

		if strings.HasPrefix(myAbstract, jw.Entity.Abstract+string(DB_IN)) ||
			jw.Entity.Plural != myPlural {
			// Stop on a new parent or a new sibling collection
			break
		}

		if jw.Entity.Type == ENTITY_VERSION && jw.Entity.Object["isdefault"] == true {
			jw.seenDefaultVid = jw.Entity.UID
		}

		jw.Printf("%s\n%s%q: ", extra, jw.indent, jw.Entity.UID)
		if err := jw.WriteEntity(); err != nil {
			return count, err
		}

		count++
		extra = ","
	}

	jw.Outdent()
	if extra != "" {
		jw.Printf("\n%s", jw.indent)
	}
	jw.Print("}")

	return count, nil
}

func (jw *JsonWriter) WriteEntity() error {
	log.VPrintf(3, ">Enter: WriteEntity (%v)", jw.Entity)
	defer log.VPrintf(3, "<Exit: WriteEntity")

	if jw.Entity == nil {
		jw.Printf("{}")
		return nil
	}

	// Is this entity a Resource and does it have a meta.xref value?
	hasXref := false

	extra := "" // stuff to go at end of line during next print - eg: ,
	myType := jw.Entity.Type
	myAbstract := jw.Entity.Abstract
	addSpace := false // Add space before next attribute?

	if log.GetVerbose() > 3 {
		log.VPrintf(0, "eType: %d", myType)
		log.VPrintf(0, "JW:\n%s\n", ToJSON(jw))
		log.VPrintf(0, "JW.Obj:\n%s\n", ToJSON(jw.Entity.Object))
		log.VPrintf(0, "JW.NObj:\n%s\n", ToJSON(jw.Entity.NewObject))
	}

	jw.Printf("{")
	jw.Indent()

	jsonIt := func(e *Entity, info *RequestInfo, key string, val any, attr *Attribute) error {
		log.VPrintf(4, "jsonIt: %q", key)
		if key == "$space" {
			addSpace = true
			return nil
		}

		// Can't figure out how to do this via the props table
		if key == "capabilities" || key == "model" || key == "modelsource" {
			if !jw.didCapModelSpace {
				addSpace = true
				jw.didCapModelSpace = true
			}
		}

		// TODO see if we should skip all $ props
		if key == "$COLLECTIONS" {
			return nil
		}

		if attr != nil && attr.internals != nil && attr.internals.neverSerialize {
			return nil
		}

		if key[0] == '#' || key[0] == '$' {
			// Skip all other internal attributes
			return nil
		}

		// "RESOURCE" has a special serialization func
		if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION {
			rm := e.GetResourceModel()
			if rm.GetHasDocument() && key == rm.Singular {
				return SerializeResourceContents(jw, jw.Entity, jw.info, &extra)
			}
		}

		if addSpace {
			jw.Printf("%s\n", extra)
			extra = ""
			addSpace = false
		}

		buf := []byte(nil)

		// not thrilled but gotta do it when you're picky
		if e.Type == ENTITY_META && key == "deprecated" {
			daMap, ok := val.(map[string]any)
			if ok {
				b := bytes.Buffer{}
				b.WriteString("{")

				keys := []string{"effective", "removal",
					"alternative", "documentation"}

				gotOne := false
				extra := "\n  " + jw.indent
				for _, k := range keys {
					v, ok := daMap[k]
					if ok {
						b.WriteString(fmt.Sprintf("%s%q: %q", extra, k, v))
						extra = ",\n  " + jw.indent
						gotOne = true
					}
				}

				sortedKeys := SortedKeys(daMap)
				for _, k := range sortedKeys {
					if ArrayContains(keys, k) {
						continue
					}
					v := daMap[k]
					if ok {
						b.WriteString(fmt.Sprintf("%s%q: %q", extra, k, v))
						extra = ",\n  " + jw.indent
						gotOne = true
					}
				}
				if gotOne {
					b.WriteString("\n" + jw.indent)
				}

				b.WriteString("}")
				buf = b.Bytes()
			}
		}

		if buf == nil {
			buf, _ = json.MarshalIndent(val, jw.indent, "  ")
		}
		jw.Printf("%s\n%s%q: %s", extra, jw.indent, key, string(buf))
		extra = ","
		return nil
	}

	var err error

	// Skip serializing the root entity's attributes if ?collections is set
	// AND we're on the root entity of the response
	if !jw.info.HasFlag("collections") || jw.info.Root != jw.Entity.Path {
		err := jw.Entity.SerializeProps(jw.info, jsonIt)
		if err != nil {
			panic(err)
		}
	}

	// Now show all of the nested collections
	if extra != "" && myType != ENTITY_RESOURCE {
		// Resources already added the \n before "metaurl"
		extra += "\n" // just because it looks nicer with a blank line
	}

	if jw.Entity.Type == ENTITY_RESOURCE {
		jw.seenDefaultVid = "" // just to be safe, clear it for each Resource
	}

	jw.LoadCollections(myType) // load the list of current collections
	if _, err := jw.NextEntity(); err != nil {
		return err
	}

	// If we need to delay the serialization of "meta" for later
	var cachedMeta *Entity

	// If next entity is 'meta' then skip it if we're not inlining it
	// Note, we're getting lucky that "meta" comes before "versions".
	// We really should fix this.
	if jw.Entity != nil && jw.Entity.Type == ENTITY_META {
		hasXref = !IsNil(jw.Entity.Get("xref"))

		p, _ := PropPathFromPath(jw.Entity.Abstract)
		if jw.info.ShouldInline(p.DB()) {
			verAbs := jw.Entity.Abstract[:len(jw.Entity.Abstract)-4] + "versions"
			versProp, _ := PropPathFromPath(verAbs)

			// If in doc view & there are filters & "versions" is inlined,
			// then we'll serialize "meta" after "versions" so we know if the
			// default Version was included or not. If not then the
			// defaultversionurl needs to be absolute, not relative
			if jw.info.DoDocView() && len(jw.info.Filters) > 0 && jw.info.ShouldInline(versProp.DB()) {
				cachedMeta = jw.Entity
				if _, err = jw.NextEntity(); err != nil {
					return err
				}
			} else {
				jw.Printf("%s\n%s%q: ", extra, jw.indent, "meta")
				if err := jw.WriteEntity(); err != nil {
					return err
				}
				extra = ","
				// We don't need to call "jw.NextEntity()" because the
				// WriteEntity() call above would have already done it for us
			}
		} else {
			// Skip "meta" entity since we're not serialize/inlining it
			if _, err = jw.NextEntity(); err != nil {
				return err
			}
		}
	}

	// Loop thru all of this entity's children.
	// If we have more Entities to process, and the next one is a child
	// (based on its Abstract being a superset of this Entity's Abstract)
	// then it might be a child so Write it.
	// However, before we do, we need to Write all lower (alphabetically)
	// empty collections (WritePreCollections).
	// When done with all children, WritePostCollections will serialize all
	// empty collections that alphabetically come after the last child
	for jw.Entity != nil &&
		(myAbstract == "" ||
			strings.HasPrefix(jw.Entity.Abstract, myAbstract+string(DB_IN))) {

		extra = jw.WritePreCollections(hasXref, extra, jw.Entity.Plural, myType)

		if extra, err = jw.WriteCollectionHeader(extra); err != nil {
			return err
		}
	}
	extra = jw.WritePostCollections(hasXref, extra, myType)

	// After all of the collections, which should include "versions",
	// check to see if we need to serialize a cached "meta" sub-object.
	// If so, make sure to pass along the defaultVersionID (via info.extras)
	// so that the getFn for defaultversionurl has access to it.
	if cachedMeta != nil {
		jw.Push(jw.Entity)
		jw.Entity = cachedMeta
		jw.info.extras["seenDefaultVid"] = jw.seenDefaultVid

		jw.Printf("%s\n%s%q: ", extra, jw.indent, "meta")
		if err := jw.WriteEntity(); err != nil {
			return err
		}
		extra = ","

		delete(jw.info.extras, "seenDefaultVid")
		jw.seenDefaultVid = ""
	}

	// And finally done with this Entity
	jw.Outdent()
	jw.Printf("\n%s}", jw.indent)

	return nil
}

func SerializeResourceContents(jw *JsonWriter, e *Entity, info *RequestInfo, extra *string) error {
	PanicIf(e.Type != ENTITY_RESOURCE && e.Type != ENTITY_VERSION, "Bad eType: %d", e.Type)
	// Add the "resource*" props
	_, rm := jw.Entity.GetModels()
	singular := rm.Singular

	// If the RESOURCE* props aren't there then just exit.
	// This will happen when "export/docView" is enabled because the
	// props won't show up in the Resorce but will on the default version
	// TODO really should do this check in entity.SerializeProps
	if IsNil(jw.Entity.Object["#contentid"]) &&
		IsNil(jw.Entity.Object[singular+"proxyurl"]) {
		return nil
	}

	p2, _ := PropPathFromDB(jw.Entity.Abstract)
	p := p2.P(singular).DB()
	if jw.info.ShouldInline(p) {
		data := []byte{}
		if val := jw.Entity.Get(singular); val != nil {
			var ok bool
			data, ok = val.([]byte)
			PanicIf(!ok, "Can't convert to []byte: %s", val)
		}

		if url := jw.Entity.GetAsString(singular + "proxyurl"); url != "" {
			resp, err := http.Get(url)
			if err != nil {
				data = []byte("GET error:" + err.Error())
			} else if resp.StatusCode/100 != 2 {
				data = []byte("GET error:" + resp.Status)
			} else {
				data, err = io.ReadAll(resp.Body)
				if err != nil {
					data = []byte("GET error:" + err.Error())
				}
			}
		}

		if data == nil {
			return nil
		}

		ct := jw.Entity.GetAsString("contenttype")
		ct = rm.MapContentType(ct)

		// Try to write the body in either JSON (the current
		// raw bytes stored in the DB), or if not valid JSON then
		// base64 encode it.
		if ct == "json" {
			if json.Valid(data) {
				// Only write the data as raw JSON (with indents)
				// if it doesn't start with quotes. For that case
				// since we need to escape the quotes we're going to
				// need to escape things, and in those cases
				// we just base64 encode it (the 'else' clause)
				pretty := bytes.Buffer{}
				err := json.Indent(&pretty, data, jw.indent, "  ")
				PanicIf(err != nil, "Bad JSON: %s", string(data))
				jw.Printf("%s\n%s%q: %s", *extra, jw.indent,
					singular, pretty.String())
			} else {
				// Write as escaped string
				ct = "string"
			}
		}

		if ct == "string" {
			// Write as escaped string
			buf, err := json.Marshal(string(data))
			PanicIf(err != nil, "Can't serialize: %s", string(data))
			jw.Printf("%s\n%s%q: %s", *extra, jw.indent,
				singular, string(buf))
		} else if ct == "binary" {
			str := base64.StdEncoding.EncodeToString(data)
			jw.Printf("%s\n%s\"%sbase64\": %q",
				*extra, jw.indent, singular, str)
		}
		*extra = ","
	}

	return nil
}

func (jw *JsonWriter) LoadCollections(eType int) {
	names := []string{}
	if eType == ENTITY_REGISTRY {
		if jw.info.Registry.Model != nil && jw.info.Registry.Model.Groups != nil {

			names = SortedKeys(jw.info.Registry.Model.Groups)
		}
	} else if eType == ENTITY_GROUP {
		gName, _ := strings.CutSuffix(jw.Entity.Abstract, IN_STR)
		names = SortedKeys(jw.info.Registry.Model.Groups[gName].Resources)
	} else if eType == ENTITY_RESOURCE {
		names = []string{"versions"}
	} else if eType == ENTITY_META {
		names = []string{}
	} else if eType == ENTITY_VERSION {
		names = []string{} // no children of versions
	} else {
		panic(fmt.Sprintf("Unknown eType: %d", eType))
	}
	jw.unusedColls[eType] = names

	p := jw.Entity.Path + "/"
	if p == "/" {
		p = ""
	}
	jw.collPaths[eType] = p
}

func (jw *JsonWriter) WritePreCollections(hasXref bool, extra string, plural string, eType int) string {
	for i, collName := range jw.unusedColls[eType] {
		if collName == plural {
			jw.unusedColls[eType] = jw.unusedColls[eType][i+1:]
			break
		}
		extra = jw.WriteEmptyCollection(hasXref, extra, eType, collName)
	}
	return extra
}

func (jw *JsonWriter) WritePostCollections(hasXref bool, extra string, eType int) string {
	for _, collName := range jw.unusedColls[eType] {
		extra = jw.WriteEmptyCollection(hasXref, extra, eType, collName)
	}

	delete(jw.collPaths, eType)
	delete(jw.unusedColls, eType)
	return extra
}

func (jw *JsonWriter) WriteEmptyCollection(hasXref bool, extra string, eType int, collName string) string {
	// If we're doing a Resource that has a meta.xref, skip "versions"
	if hasXref && collName == "versions" {
		return extra
	}

	p := Path2Abstract(jw.collPaths[eType] + collName)

	inlineCollection := jw.info.ShouldInline(p)
	baseURL := ""
	path := jw.collPaths[eType]

	if !jw.info.DoDocView() || !inlineCollection {
		baseURL = jw.info.BaseURL
	} else {
		baseURL = DOCVIEW_BASE

		// remove GET's base path
		path = path[len(jw.info.Root):]
		if strings.HasPrefix(path, "/") {
			path = path[1:]
		}
	}

	jw.Printf("%s\n%s\"%surl\": \"%s/%s%s\",\n", extra, jw.indent,
		collName, baseURL, path, collName)

	if inlineCollection {
		jw.Printf("%s\"%s\": {},\n", jw.indent, collName)
	}

	jw.Printf("%s\"%scount\": 0", jw.indent, collName)
	extra = ","

	return extra
}

func Path2Abstract(path string) string {
	parts := strings.Split(path, "/")
	addSlash := strings.HasSuffix(path, "/")
	res := ""
	for i, part := range parts {
		if i%2 == 0 {
			if res != "" {
				res += string(DB_IN)
			}
			res += part
		}
	}
	if addSlash {
		res += string(DB_IN)
	}
	return res
}
