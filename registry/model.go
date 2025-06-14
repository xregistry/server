package registry

import (
	"encoding/json"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

// VerifyAndSave() should be called by automatically but there may be
// cases where someone would need to call it manually (e.g. setting an
// attribute's property - we should technically find a way to catch those
// cases so code above this shouldn't need to think about it
func (m *Model) VerifyAndSave() error {
	if m.GetChanged() == false {
		return nil
	}

	if err := m.Verify(); err != nil {
		// Kind of extreme, but if there's an error revert the entire
		// model to the last known good state. So, all of the changes
		// people made will be lost and any variables are bogus
		// NOTE any local variable pointing to a model entity will need to
		// be refresh/refound, the existing pointer will be bad

		// No longer needed but left around just in case
		// *m = *LoadModel(m.Registry)

		return err
	}

	return m.Save()
}

func (m *Model) Save() error {
	// log.Printf("In model.Save - changed: %v", m.GetChanged())
	if m.GetChanged() == false {
		return nil
	}

	if log.GetVerbose() > 4 {
		buf, _ := json.MarshalIndent(m, "", "  ")
		log.Printf("Saving model:\n%s", string(buf))
	}

	// Create a temporary type so that we don't use the MarshalJSON func
	// in model.go. That one will exclude "model" from the serialization and
	// we don't want to do that when we're saving it in the DB. We only want
	// to do that when we're serializing the model for the end user.

	buf, _ := json.Marshal(m)
	modelStr := string(buf)

	// log.Printf("Saving model itself")
	err := DoZeroTwo(m.Registry.tx, `
        INSERT INTO Models(RegistrySID, Model)
        VALUES(?,?)
        ON DUPLICATE KEY UPDATE Model=?`,

		m.Registry.DbSID, modelStr,
		modelStr)
	if err != nil {
		log.Printf("Error updating model: %s", err)
		return err
	}

	existingModelEntities := map[string]string{} // Abstract->SID
	results, err := Query(m.Registry.tx,
		`SELECT SID,Abstract FROM ModelEntities WHERE RegistrySID=?`,
		m.Registry.DbSID)
	defer results.Close()
	if err != nil {
		log.Printf("Error loading model entities(%s): %s", m.Registry.UID, err)
		return nil
	}
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		sid := NotNilString(row[0])
		abs := NotNilString(row[1])
		existingModelEntities[abs] = sid
	}

	// Remove from existingModelEntities, all MEs that are going to be kept
	// around. Then we'll delete everything else before we re-add the keepers
	// to ensure there isn't any conflicts.
	// We can't just delete the entire set and re-add them because the DB
	// will erase all instances of those types automatically when the types
	// are deleted.

	inUseAbs := map[string]bool{}
	for _, gm := range m.Groups {
		for _, rName := range gm.XImportResources {
			parts := strings.Split(rName, "/")
			rAbs := "/" + gm.Plural + "/" + parts[2]
			if _, ok := existingModelEntities[rAbs]; ok {
				inUseAbs[rAbs] = true
			}
		}
		gAbs := "/" + gm.Plural
		if _, ok := existingModelEntities[gAbs]; ok {
			inUseAbs[gAbs] = true
		}
		for _, rm := range gm.Resources {
			rmAbs := gAbs + "/" + rm.Plural
			if _, ok := existingModelEntities[rmAbs]; ok {
				inUseAbs[rmAbs] = true
			}
		}
	}

	// Delete any model entities not found in the new model
	for meAbs, _ := range existingModelEntities {
		if inUseAbs[meAbs] != true {
			// TODO if we ever think this list is long, make this faster
			err = DoOne(m.Registry.tx, `
                      DELETE FROM ModelEntities
                      WHERE RegistrySID=? AND Abstract=?`,
				m.Registry.DbSID, meAbs)
			if err != nil {
				log.Printf("Error deleting modelEntity(%s): %s", meAbs, err)
				return err
			}
		}
	}

	// Now just add the new ones
	for _, gm := range m.Groups {
		gmAbs := "/" + gm.Plural
		for _, rName := range gm.XImportResources {
			// See if the ximported resource is alread there, if so skip it
			parts := strings.Split(rName, "/")
			targetRM := gm.Model.FindResourceModel(parts[1], parts[2])
			rSID := gm.SID + "-" + targetRM.SID // parts[2]
			rAbs := gmAbs + "/" + parts[2]
			if _, ok := existingModelEntities[rAbs]; !ok {
				// add the new ximported resource
				err := Do(m.Registry.tx,
					`INSERT INTO ModelEntities(
                         SID, RegistrySID, ParentSID,
                         Abstract, Plural, Singular)
                     VALUES(?,?,?,?,?,?)`,
					rSID, m.Registry.DbSID, gm.SID,
					rAbs, targetRM.Plural, targetRM.Singular)
				if err != nil {
					log.Printf("Error inserting modelEntity(%s): %s", rSID, err)
					return err
				}
			}
		}

		// If GroupModel is already in DB then skip it
		if _, ok := existingModelEntities[gmAbs]; !ok {
			// Add new GroupModel
			err := Do(m.Registry.tx,
				`INSERT INTO ModelEntities(
                     SID, RegistrySID, ParentSID,
                     Abstract, Plural, Singular)
                 VALUES(?,?,?,?,?,?)`,
				gm.SID, m.Registry.DbSID, nil,
				gmAbs, gm.Plural, gm.Singular)
			if err != nil {
				log.Printf("Error inserting modelEntity(%s): %s", gm.SID, err)
				return err
			}
		}

		for _, rm := range gm.Resources {
			rmAbs := gmAbs + "/" + rm.Plural
			// If ResourceModel is already in DB then skip it
			if _, ok := existingModelEntities[rmAbs]; !ok {
				// Add new ResourceModel
				err := Do(m.Registry.tx,
					`INSERT INTO ModelEntities(
                             SID, RegistrySID, ParentSID,
                             Abstract, Plural, Singular)
                         VALUES(?,?,?,?,?,?)`,
					rm.SID, m.Registry.DbSID, gm.SID,
					gmAbs+"/"+rm.Plural, rm.Plural, rm.Singular)
				if err != nil {
					log.Printf("Error inserting modelEntity(%s): %s",
						ToJSON(rm), err)
					return err
				}
			}
		}
	}

	m.SetChanged(false)

	return nil
}

func LoadModel(reg *Registry) *Model {
	log.VPrintf(3, ">Enter: LoadModel")
	defer log.VPrintf(3, "<Exit: LoadModel")

	PanicIf(reg == nil, "nil")

	// Load Registry model
	results, err := Query(reg.tx,
		`SELECT Model FROM Models WHERE RegistrySID=?`,
		reg.DbSID)
	defer results.Close()
	if err != nil {
		log.Printf("Error loading registries(%s): %s", reg.UID, err)
		return nil
	}
	row := results.NextRow()
	if row == nil {
		log.Printf("Can't find registry: %s", reg.UID)
		return nil
	}

	modelBuf := []byte(nil)
	if row[0] != nil {
		modelBuf = []byte(NotNilString(row[0]))
	}
	results.Close()

	model, err := ParseModel(modelBuf)
	if err != nil {
		return nil
	}
	model.Registry = reg

	reg.Model = model
	return model
}

func (m *Model) ApplyNewModel(newM *Model) error {
	newM.Registry = m.Registry
	// log.Printf("ApplyNewModel:\n%s\n", ToJSON(newM))

	// Copy existing SIDs into the new Model so we don't create new ones
	for _, gm := range newM.Groups {
		if oldGM := m.FindGroupModel(gm.Plural); oldGM != nil {
			gm.SID = oldGM.SID

			for _, rm := range gm.Resources {
				if oldRM := gm.FindResourceModel(rm.Plural); oldRM != nil {
					rm.SID = oldRM.SID
				}
			}
		}
	}

	m.Registry.Model = newM
	m = newM
	m.SetChanged(true)

	if err := m.VerifyAndSave(); err != nil {
		// Too much to undo. The Verify() at the top should have caught
		// anything wrong
		return err
	}

	return nil
}

func (m *Model) ApplyNewModelFromJSON(buf []byte) error {
	modelSource := string(buf)
	if modelSource == "" {
		modelSource = "{}"
	}

	// Don't allow local files to be included (e.g. ../foo)
	buf, err := ProcessIncludes("", buf, false)
	if err != nil {
		return err
	}

	model, err := ParseModel(buf)
	if err != nil {
		return err
	}
	model.Source = modelSource

	return m.ApplyNewModel(model)
}

func (rm *ResourceModel) VerifyData() error {
	reg := rm.GroupModel.Model.Registry

	// Query to find all Groups/Resources of the proper type.
	// The resulting list MUST be Group followed by it's Resources, repeat...
	gAbs := NewPPP(rm.GroupModel.Plural).Abstract()
	rAbs := NewPPP(rm.GroupModel.Plural).P(rm.Plural).Abstract()
	entities, err := RawEntitiesFromQuery(reg.tx, reg.DbSID, FOR_WRITE,
		`Abstract=? OR Abstract=?`, gAbs, rAbs)
	if err != nil {
		return err
	}

	// First, let's make sure each Resource doesn't have too many Versions
	// or has too many root versions

	group := (*Group)(nil)
	resource := (*Resource)(nil)
	for _, e := range entities {
		if e.Type == ENTITY_GROUP {
			group = &Group{Entity: *e, Registry: reg}
			group.Self = group
		} else {
			PanicIf(group == nil, "Group can't be nil")
			resource = &Resource{Entity: *e, Group: group}
			resource.Self = resource

			if err = resource.EnsureSingleVersionRoot(); err != nil {
				return err
			}

			if err = resource.EnsureMaxVersions(); err != nil {
				return err
			}

			resource.tx.AddResource(resource)
		}
	}

	return nil
}

// The model serializer we use for the "xRegistry" schema format
func Model2xRegistryJson(m *Model, format string) ([]byte, error) {
	return json.MarshalIndent((*UserModel)(m), "", "  ")
}

func GetModelSerializer(format string) ModelSerializer {
	format = strings.ToLower(format)
	searchParts := strings.SplitN(format, "/", 2)
	if searchParts[0] == "" {
		return nil
	}
	if len(searchParts) == 1 {
		searchParts = append(searchParts, "")
	}

	result := ModelSerializer(nil)
	resultVersion := ""

	for format, sm := range ModelSerializers {
		format = strings.ToLower(format)
		parts := strings.SplitN(format, "/", 2)
		if searchParts[0] != parts[0] {
			continue
		}
		if len(parts) == 1 {
			parts = append(parts, "")
		}

		if searchParts[1] != "" {
			if searchParts[1] == parts[1] {
				// Exact match - stop immediately
				result = sm
				break
			}
			// Looking for an exact match - not it so skip it
			continue
		}

		if resultVersion == "" || strings.Compare(parts[1], resultVersion) > 0 {
			result = sm
			resultVersion = parts[1]
		}
	}

	return result
}

func RegisterModelSerializer(name string, sm ModelSerializer) {
	ModelSerializers[name] = sm
}

func init() {
	RegisterModelSerializer(XREGSCHEMA+"/"+SPECVERSION, Model2xRegistryJson)
}
