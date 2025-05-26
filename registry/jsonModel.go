package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/duglin/dlog"
	// "reflect"
	. "github.com/xregistry/server/common"
)

type UserModel Model
type UserAttribute Attribute
type UserGroupModel GroupModel
type UserResourceModel ResourceModel

func (m *Model) UserMarshal(prefix string, indent string) ([]byte, error) {
	return json.MarshalIndent((*UserModel)(m), prefix, indent)
}

func (um *UserModel) MarshalJSON() ([]byte, error) {
	extra := ""
	buf := bytes.Buffer{}

	buf.WriteRune('{')
	if um.Labels != nil {
		b, _ := json.Marshal(um.Labels)
		buf.WriteString(`"labels":`)
		buf.Write(b)
		extra = ","
	}

	propsOrdered, propsMap := ((*Model)(um)).GetPropsOrdered()
	data, extra := marshalAttrs(um.Attributes, "attributes", extra,
		propsOrdered, propsMap)
	buf.Write(data)

	if len(um.Groups) > 0 {
		buf.WriteString(extra)
		buf.WriteString(`"groups":{`)
		extra = ""
		for _, k := range SortedKeys(um.Groups) {
			group := um.Groups[k]
			buf.WriteString(extra)
			buf.WriteRune('"')
			buf.WriteString(k)
			buf.WriteString(`":`)
			b, _ := json.Marshal((*UserGroupModel)(group))
			buf.Write(b)
			extra = ","
		}
		buf.WriteString(`}`)
		extra = ","
	}

	buf.WriteRune('}')
	return buf.Bytes(), nil
}

func (ug *UserGroupModel) MarshalJSON() ([]byte, error) {
	extra := ""
	buf := bytes.Buffer{}

	buf.WriteRune('{')
	buf.WriteString(`"plural":"`)
	buf.WriteString(ug.Plural)
	buf.WriteString(`","singular":"`)
	buf.WriteString(ug.Singular)
	buf.WriteRune('"')
	if ug.Description != "" {
		buf.WriteString(`,"description":"`)
		buf.WriteString(ug.Description)
		buf.WriteRune('"')
	}
	if ug.ModelVersion != "" {
		buf.WriteString(`,"modelversion":"`)
		buf.WriteString(ug.ModelVersion)
		buf.WriteRune('"')
	}
	if ug.CompatibleWith != "" {
		buf.WriteString(`,"compatiblewith":"`)
		buf.WriteString(ug.CompatibleWith)
		buf.WriteRune('"')
	}
	extra = ","

	if ug.Labels != nil {
		b, _ := json.Marshal(ug.Labels)
		buf.WriteString(`,"labels":`)
		buf.Write(b)
	}

	if len(ug.XImportResources) > 0 {
		b, _ := json.Marshal(ug.XImportResources)
		buf.WriteString(`,"ximportresources":`)
		buf.Write(b)
	}

	propsOrdered, propsMap := ((*GroupModel)(ug)).GetPropsOrdered()
	data, extra := marshalAttrs(ug.Attributes, "attributes", extra,
		propsOrdered, propsMap)
	buf.Write(data)

	if len(ug.Resources) > 0 {
		buf.WriteString(extra)
		buf.WriteString(`"resources":{`)
		extra = ""
		for _, k := range SortedKeys(ug.Resources) {
			resource, _ := ug.Resources[k]
			buf.WriteString(extra)
			buf.WriteRune('"')
			buf.WriteString(k)
			buf.WriteString(`":`)
			b, _ := json.Marshal((*UserResourceModel)(resource))
			buf.Write(b)
			extra = ","
		}
		buf.WriteString(`}`)
		extra = ","
	}

	buf.WriteRune('}')
	return buf.Bytes(), nil
}

func (ur *UserResourceModel) MarshalJSON() ([]byte, error) {
	log.VPrintf(3, "In MarshalJSON")
	extra := ""
	buf := bytes.Buffer{}

	buf.WriteRune('{')
	buf.WriteString(`"plural":"`)
	buf.WriteString(ur.Plural)
	buf.WriteString(`","singular":"`)
	buf.WriteString(ur.Singular)
	buf.WriteRune('"')
	if ur.Description != "" {
		buf.WriteString(`,"description":"`)
		buf.WriteString(ur.Description)
		buf.WriteRune('"')
	}
	buf.WriteString(fmt.Sprintf(`,"maxversions":%d`, ur.MaxVersions))
	buf.WriteString(fmt.Sprintf(`,"setversionid":%v`,
		NotNilBoolPtr(ur.SetVersionId)))
	buf.WriteString(fmt.Sprintf(`,"setdefaultversionsticky":%v`,
		NotNilBoolPtr(ur.SetDefaultSticky)))
	buf.WriteString(fmt.Sprintf(`,"hasdocument":%v`,
		NotNilBoolPtr(ur.HasDocument)))
	buf.WriteString(fmt.Sprintf(`,"singleversionroot":%v`,
		NotNilBoolPtr(ur.SingleVersionRoot)))
	if len(ur.TypeMap) > 0 {
		buf.WriteString(`,"typemaps":`)
		b, _ := json.Marshal(ur.TypeMap)
		buf.Write(b)
	}
	if ur.ModelVersion != "" {
		buf.WriteString(`,"modelversion":"`)
		buf.WriteString(ur.ModelVersion)
		buf.WriteRune('"')
	}
	if ur.CompatibleWith != "" {
		buf.WriteString(`,"compatiblewith":"`)
		buf.WriteString(ur.CompatibleWith)
		buf.WriteRune('"')
	}

	extra = ","

	if ur.Labels != nil {
		b, _ := json.Marshal(ur.Labels)
		buf.WriteString(`,"labels":`)
		buf.Write(b)
	}

	propsOrdered, propsMap := ((*ResourceModel)(ur)).GetPropsOrdered()
	data, extra := marshalAttrs(ur.Attributes, "attributes", extra,
		propsOrdered, propsMap)
	buf.Write(data)

	propsOrdered, propsMap = ((*ResourceModel)(ur)).GetMetaPropsOrdered()
	data, extra = marshalAttrs(ur.MetaAttributes, "metaattributes", extra,
		propsOrdered, propsMap)
	buf.Write(data)

	buf.WriteRune('}')
	return buf.Bytes(), nil
}

func (ua *UserAttribute) MarshalJSON() ([]byte, error) {
	type tmpAttr Attribute
	return json.Marshal((*tmpAttr)(ua))
}

func marshalAttrs(attrs Attributes, name string, extra string,
	propsOrdered []*Attribute,
	propsMap map[string]*Attribute) ([]byte, string) {

	data := []byte(nil)

	if len(attrs) > 0 {
		buf := bytes.Buffer{}
		buf.WriteString(extra)
		buf.WriteRune('"')
		buf.WriteString(name)
		buf.WriteString(`":{`)
		extra = ""

		for _, prop := range propsOrdered {
			name := prop.Name

			if name[0] == '$' {
				if name == "$extensions" {
					list := SortedKeys(attrs)
					if len(list) > 0 && list[0] == "*" {
						list = append(list[1:], list[0])
					}
					for _, k := range list {
						if k[0] == '$' || propsMap[k] != nil {
							continue
						}

						attr, _ := attrs[k]
						buf.WriteString(extra)
						buf.WriteRune('"')
						buf.WriteString(k)
						buf.WriteString(`":`)
						b, _ := json.Marshal((*UserAttribute)(attr))
						buf.Write(b)
						extra = ","
					}
				}
				continue
			}

			if attr, ok := attrs[name]; ok {
				buf.WriteString(extra)
				buf.WriteRune('"')
				buf.WriteString(name)
				buf.WriteString(`":`)
				b, _ := json.Marshal((*UserAttribute)(attr))
				buf.Write(b)
				extra = ","
			}
		}
		buf.WriteRune('}')
		data = buf.Bytes()
		extra = ","
	}

	// log.Printf("Attrs:\n%s\nex: %q\n", data, extra)

	return data, extra
}
