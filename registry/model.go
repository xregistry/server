package registry

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

// VerifyAndSave() should be called by automatically but there may be
// cases where someone would need to call it manually (e.g. setting an
// attribute's property - we should technically find a way to catch those
// cases so code above this shouldn't need to think about it
func (m *Model) VerifyAndSave(verifyData bool) *XRError {
	if m.GetChanged() == false {
		return nil
	}

	if xErr := m.Verify(); xErr != nil {
		// Kind of extreme, but if there's an error revert the entire
		// model to the last known good state. So, all of the changes
		// people made will be lost and any variables are bogus
		// NOTE any local variable pointing to a model entity will need to
		// be refresh/refound, the existing pointer will be bad

		// No longer needed but left around just in case
		// *m = *LoadModel(m.Registry)

		return xErr
	}

	// Save before we verifyData because saving will delete stuff from the
	// Registry that should no longer exist (like Resources for non-existing
	// types)
	if xErr := m.Save(); xErr != nil {
		return xErr
	}

	if verifyData {
		if xErr := m.Registry.VerifyData(); xErr != nil {
			return xErr
		}
	}

	return nil
}

func (m *Model) Save() *XRError {
	// log.Printf("In model.Save - changed: %v", m.GetChanged())
	if m.GetChanged() == false {
		return nil
	}

	if log.HasKeyword("ModelSave") || log.GetVerbose() > 4 {
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
	DoZeroTwo(m.Registry.tx, `
        INSERT INTO Models(RegistrySID, Model)
        VALUES(?,?)
        ON DUPLICATE KEY UPDATE Model=?`,

		m.Registry.DbSID, modelStr,
		modelStr)

	existingModelEntities := map[string]string{} // Abstract->SID
	results := Query(m.Registry.tx,
		`SELECT SID,Abstract FROM ModelEntities WHERE RegistrySID=?`,
		m.Registry.DbSID)
	defer results.Close()

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
			DoOne(m.Registry.tx, `
                      DELETE FROM ModelEntities
                      WHERE RegistrySID=? AND Abstract=?`,
				m.Registry.DbSID, meAbs)
		}
	}

	// Now just add the new ones
	for _, gm := range m.Groups {
		gmAbs := "/" + gm.Plural

		// If GroupModel is already in DB then skip it
		if _, ok := existingModelEntities[gmAbs]; !ok {
			// Add new GroupModel
			Do(m.Registry.tx,
				`INSERT INTO ModelEntities(
                     SID, RegistrySID, ParentSID,
                     Abstract, Plural, Singular)
                 VALUES(?,?,?,?,?,?)`,
				gm.SID, m.Registry.DbSID, nil,
				gmAbs, gm.Plural, gm.Singular)
		}

		for _, rm := range gm.Resources {
			rmAbs := gmAbs + "/" + rm.Plural
			// If ResourceModel is already in DB then skip it
			if _, ok := existingModelEntities[rmAbs]; !ok {
				// Add new ResourceModel
				Do(m.Registry.tx,
					`INSERT INTO ModelEntities(
                             SID, RegistrySID, ParentSID,
                             Abstract, Plural, Singular)
                         VALUES(?,?,?,?,?,?)`,
					rm.SID, m.Registry.DbSID, gm.SID,
					gmAbs+"/"+rm.Plural, rm.Plural, rm.Singular)
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
	results := Query(reg.tx,
		`SELECT Model FROM Models WHERE RegistrySID=?`,
		reg.DbSID)
	defer results.Close()

	row := results.NextRow()
	if row == nil {
		ShowStack()
		log.Printf("Can't find registry: %s", reg.UID)
		return nil
	}

	modelBuf := []byte(nil)
	if row[0] != nil {
		modelBuf = []byte(NotNilString(row[0]))
	}
	results.Close()

	model, xErr := ParseModel(modelBuf, reg)
	if xErr != nil {
		return nil
	}
	model.Registry = reg

	reg.Model = model
	return model
}

func (m *Model) ApplyNewModel(newM *Model, src string, verifyData bool) *XRError {
	if newM == nil && len(src) != 0 {
		var xErr *XRError
		newM, xErr = ParseModel([]byte(src), m.Registry)
		if xErr != nil {
			return xErr
		}
	}

	newM.Registry = m.Registry
	// log.Printf("ApplyNewModel:\n%s\n", ToJSON(newM))

	// Copy existing SIDs into the new Model so we don't create new ones
	for gmPlural, gm := range newM.Groups {
		// Note: gm.Plural might be ""
		oldGM := m.FindGroupModel(gmPlural)
		if oldGM != nil {
			if oldGM.Singular != gm.Singular {
				return NewXRError("model_error", "/model",
					"error_detail="+
						fmt.Sprintf("changing the singular name of Group %q "+
							"is not allowed", gmPlural))
			}
			gm.SID = oldGM.SID

			for rmPlural, rm := range gm.Resources {
				// Note: rm.Plural might be ""
				if oldRM := oldGM.Resources[rmPlural]; oldRM != nil {
					if oldRM.Singular != rm.Singular {
						return NewXRError("model_error", "/model",
							"error_detail="+
								fmt.Sprintf("changing the singular name of "+
									"Resource %q is not allowed", rmPlural))
					}
					rm.SID = oldRM.SID
				}
			}
		}
	}

	m.Registry.Model = newM
	m = newM
	m.SetChanged(true)

	if src == "" {
		// This should serialize just the bare minimum, only what the
		// user provided, no default values
		// buf, err := json.MarshalIndent(m, "", "  ")
		buf, xErr := m.SerializeForUser()
		if xErr != nil {
			return xErr
		}
		src = string(buf)
	}
	m.Source = src

	if xErr := m.VerifyAndSave(verifyData); xErr != nil {
		// Too much to undo. The Verify() at the top should have caught
		// anything wrong
		return xErr
	}

	return nil
}

func (m *Model) ApplyNewModelFromJSON(buf []byte, verify bool) *XRError {
	modelSource := string(buf)
	modelSource = strings.TrimSpace(modelSource)

	if modelSource == "" {
		return NewXRError("missing_body", "/")
	}

	// Don't allow local files to be included (e.g. ../foo)
	buf, xErr := ProcessIncludes("", []byte(modelSource), false)
	if xErr != nil {
		return xErr
	}

	buf, err := RemoveSchema(buf)
	if err != nil {
		return NewXRError("bad_request", "/", "error_detail="+err.Error())
	}

	model, xErr := ParseModel(buf, m.Registry)
	if xErr != nil {
		return xErr
	}

	// model.Source = modelSource

	return m.ApplyNewModel(model, modelSource, verify)
}

func (rm *ResourceModel) VerifyData() *XRError {
	reg := rm.GroupModel.Model.Registry

	// Query to find all Groups/Resources of the proper type.
	// The resulting list MUST be Group followed by it's Resources, repeat...
	gAbs := NewPPP(rm.GroupModel.Plural).Abstract()
	rAbs := NewPPP(rm.GroupModel.Plural).P(rm.Plural).Abstract()
	entities, xErr := RawEntitiesFromQuery(reg.tx, reg.DbSID, FOR_WRITE,
		`Abstract=? OR Abstract=?`, gAbs, rAbs)
	if xErr != nil {
		return xErr
	}

	// For each Resource, make this it's compliant with all of the various
	// constraints/rules that are defined

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

			if xErr = resource.EnsureSingleVersionRoot(); xErr != nil {
				return xErr
			}

			if xErr = resource.EnsureMaxVersions(); xErr != nil {
				return xErr
			}

			if xErr = resource.EnsureCompat(true); xErr != nil {
				return xErr
			}

			resource.tx.AddResource(resource)
		}
	}

	return nil
}

func (m *Model) SerializeForUser() ([]byte, *XRError) {
	buf, err := json.MarshalIndent((*UserModel)(m), "", "  ")
	if err != nil {
		return nil, NewXRError("server_error", "/").SetDetail(err.Error() + ".")
	}
	return buf, nil
}
