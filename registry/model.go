package registry

import (
	"fmt"

	log "github.com/duglin/dlog"
)

type ModelElement struct {
	Singular string
	Plural   string
	Children map[string]*ModelElement
}

type GroupModel struct {
	ID       string
	Registry *Registry

	Plural   string `json:"plural,omitempty"`
	Singular string `json:"singular,omitempty"`
	Schema   string `json:"schema,omitempty"`
	Versions int    `json:"versions"`

	Resources map[string]*ResourceModel // Plural
}

type ResourceModel struct {
	ID         string
	GroupModel *GroupModel

	Plural   string `json:"plural,omitempty"`
	Singular string `json:"singular,omitempty"`
	Versions int    `json:"versions,omitempty"`
}

type Model struct {
	Registry *Registry
	Groups   map[string]*GroupModel // Plural
}

func (g *GroupModel) AddResourceModel(plural string, singular string, versions int) (*ResourceModel, error) {
	if plural == "" {
		return nil, fmt.Errorf("Can't add a group with an empty plural name")
	}
	if singular == "" {
		return nil, fmt.Errorf("Can't add a group with an empty sigular name")
	}
	if versions < 0 {
		return nil, fmt.Errorf("''versions'(%d) must be >= 0", versions)
	}

	mID := NewUUID()

	err := Do(`
		INSERT INTO ModelEntities(
			ID,
			RegistryID,
			ParentID,
			Plural,
			Singular,
			SchemaURL,
			Versions)
		VALUES(?,?,?,?,?,?,?) `,
		mID, g.Registry.ID, g.ID, plural, singular, nil, versions)
	if err != nil {
		log.Printf("Error inserting resourceModel(%s): %s", plural, err)
		return nil, err
	}
	r := &ResourceModel{
		ID:         mID,
		GroupModel: g,
		Singular:   singular,
		Plural:     plural,
		Versions:   versions,
	}

	g.Resources[plural] = r

	return r, nil
}

func (m *Model) ToObject(ctx *Context) (*Object, error) {
	obj := NewObject()
	if m == nil {
		return obj, nil
	}

	groups := NewObject()
	for _, key := range SortedKeys(m.Groups) {
		group := m.Groups[key]
		groupObj := NewObject()
		groupObj.AddProperty("singular", group.Singular)
		groupObj.AddProperty("plural", group.Plural)
		groupObj.AddProperty("schema", group.Schema)

		resObjs := NewObject()
		for _, key := range SortedKeys(group.Resources) {
			res := group.Resources[key]
			resObj := NewObject()
			resObj.AddProperty("singular", res.Singular)
			resObj.AddProperty("plural", res.Plural)
			resObj.AddProperty("versions", res.Versions)
			resObjs.AddProperty(key, resObj)
		}

		groupObj.AddProperty("resources", resObjs)
		groups.AddProperty(key, groupObj)
	}
	obj.AddProperty("groups", groups)
	return obj, nil
}

func CreateGenericModel(model *Model) *ModelElement {
	newModel := &ModelElement{}

	for gKey, gModel := range model.Groups {
		newGroup := &ModelElement{
			Singular: gModel.Singular,
			Plural:   gModel.Plural,
		}

		for rKey, rModel := range gModel.Resources {
			newResource := &ModelElement{
				Singular: rModel.Singular,
				Plural:   rModel.Plural,
				Children: map[string]*ModelElement{
					"versions": &ModelElement{
						Singular: "version",
						Plural:   "versions",
					},
				},
			}

			if len(newGroup.Children) == 0 {
				newGroup.Children = map[string]*ModelElement{}
			}
			newGroup.Children[rKey] = newResource
		}

		if len(newModel.Children) == 0 {
			newModel.Children = map[string]*ModelElement{}
		}
		newModel.Children[gKey] = newGroup
	}

	return newModel
}
