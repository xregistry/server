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

	// Diff against whatever is STILL persisted in the DB right now (not
	// against any in-memory "old" Model/ResourceModel object - a caller
	// may have already mutated one of those directly, e.g.
	// rm.SetValidateFormat(false) on reg.Model itself, before calling
	// Save()/VerifyAndSave(), which would make an in-memory comparison
	// useless). Reading fresh from the DB here - before we overwrite the
	// Models row below - guarantees we see the true "before" state no
	// matter which style of caller (JSON PUT of a separate Model, or
	// direct Go-API mutation of reg.Model) got us here. This is what lets
	// EnsureCompat() (registry/resource.go) skip its old per-save
	// defensive clear of *validated/*validatedreason props when
	// validation is off: this one-time, model-change-triggered sweep is
	// the sole owner of clearing stale values on a true->false
	// transition. false->true needs no clearing - normal per-save
	// checking just starts happening again.
	if oldM := loadModelFromDB(m.Registry, false); oldM != nil {
		for gmPlural, gm := range m.Groups {
			oldGM := oldM.FindGroupModel(gmPlural)
			if oldGM == nil {
				continue
			}
			for rmPlural, rm := range gm.Resources {
				oldRM := oldGM.Resources[rmPlural]
				if oldRM == nil {
					continue
				}
				if oldRM.GetValidateFormat() && !rm.GetValidateFormat() {
					m.Registry.clearValidationSystemProps(rm.SID,
						"formatvalidated", "formatvalidatedreason")
				}
				if oldRM.GetValidateCompatibility() &&
					!rm.GetValidateCompatibility() {
					m.Registry.clearValidationSystemProps(rm.SID,
						"compatibilityvalidated",
						"compatibilityvalidatedreason")
				}
			}
		}
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

	// Remove from existingModelEntities all MEs that are going to be kept
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

	// Before deleting anything, make sure none of the model entities about
	// to be removed still have live instances. Deleting a ModelEntity
	// cascades (via the ModelTrigger DB trigger) into silently deleting
	// every Group/Resource of that type - and everything under them. We'd
	// rather reject the model update and have the user explicitly delete
	// those entities first than have a model change accidentally wipe out
	// data. Check ALL types first (before deleting any) so a rejection
	// never leaves a partial delete behind.
	for meAbs, sid := range existingModelEntities {
		if inUseAbs[meAbs] == true {
			continue
		}

		// A Group type's Abstract looks like "/plural" (1 path segment),
		// a Resource type's looks like "/gPlural/rPlural" (2 segments)
		parts := strings.Split(strings.Trim(meAbs, "/"), "/")

		var count int
		if len(parts) == 1 {
			results := Query(m.Registry.tx,
				`SELECT COUNT(*) FROM "Groups" WHERE ModelSID=?`, sid)
			count = NotNilInt(results.NextRow()[0])
			results.Close()

			if count > 0 {
				return NewXRError("model_error", "/model",
					"error_detail="+
						fmt.Sprintf(`can't remove Group type %q from the `+
							`model - it still has %d entities. Delete `+
							`them before removing the type`,
							parts[0], count))
			}
		} else {
			results := Query(m.Registry.tx,
				`SELECT COUNT(*) FROM Resources WHERE ModelSID=?`, sid)
			count = NotNilInt(results.NextRow()[0])
			results.Close()

			if count > 0 {
				return NewXRError("model_error", "/model",
					"error_detail="+
						fmt.Sprintf(`can't remove Resource type %q from `+
							`Group type %q - it still has %d entities. `+
							`Delete them before removing the type`,
							parts[1], parts[0], count))
			}
		}
	}

	// Delete any model entities not found in the new model
	// TODO consider batching if this gets too slow, or the list is too long
	for meAbs, _ := range existingModelEntities {
		if inUseAbs[meAbs] != true {
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
			DoOne(m.Registry.tx,
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
				DoOne(m.Registry.tx,
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

	model := loadModelFromDB(reg, true)
	if model != nil {
		reg.Model = model
	}
	return model
}

// loadModelFromDB reads and parses whatever Model is CURRENTLY persisted
// in the DB for reg, with no side-effects on reg.Model. This is safe to
// call from the middle of Model.Save() (e.g. to diff the about-to-be-
// overwritten "old" state against the new in-memory one) since - unlike
// LoadModel() - it never clobbers reg.Model out from under the caller.
// If loud is false, a missing row is treated as "nothing persisted yet"
// (e.g. the very first model save for a brand new Registry) instead of
// being logged as an error.
func loadModelFromDB(reg *Registry, loud bool) *Model {
	PanicIf(reg == nil, "nil")

	// Load Registry model
	results := Query(reg.tx,
		`SELECT Model FROM Models WHERE RegistrySID=?`,
		reg.DbSID)
	defer results.Close()

	row := results.NextRow()
	if row == nil {
		if loud {
			ShowStack()
			log.Printf("Can't find registry: %s", reg.UID)
		}
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

					// If hasdocument is transitioning false->true, any
					// pre-existing data under the reserved names
					// (<singular>/<singular>url/<singular>base64/
					// <singular>proxyurl) must not be silently reinterpreted
					// as document content - reject the transition instead
					// and make the user explicitly clear that data first.
					// NOTE: this is a one-time check at the transition
					// moment only - once hasdocument=true these names are
					// legitimately used going forward to set the actual
					// document reference, so this must NOT become a
					// standing/repeatable invariant.
					if !oldRM.GetHasDocument() && rm.GetHasDocument() {
						if xErr := checkHasDocumentEnableViolation(
							m.Registry, oldRM); xErr != nil {
							return xErr
						}
					}
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

// checkHasDocumentEnableViolation returns non-nil XRError if oldRM (a
// Resource type currently hasdocument=false, transitioning to true) has
// any existing Version with a non-nil value for one of the 4 reserved
// "$RESOURCE*" names (<singular>, <singular>url, <singular>base64,
// <singular>proxyurl). Those names are only reserved once hasdocument is
// true, so while hasdocument=false a user may have legitimately declared
// one of them as their own plain extension attribute with real data. Once
// the transition to hasdocument=true happens, that pre-existing data would
// otherwise be silently reinterpreted as document content - so we reject
// the transition instead and make the caller explicitly clear that data
// first (symmetric with checkHasDocumentViolation()'s true->false block).
//
// IMPORTANT: this must only ever be called once, at the exact moment of
// the false->true transition (see ApplyNewModel() above) - NOT as a
// standing/repeatable invariant. Once hasdocument=true, these same names
// are legitimately used going forward to set the actual document
// reference, so re-running this check on every subsequent write would
// incorrectly reject normal document usage.
func checkHasDocumentEnableViolation(reg *Registry, oldRM *ResourceModel) *XRError {
	names := []string{
		oldRM.Singular + string(DB_IN),
		oldRM.Singular + "url" + string(DB_IN),
		oldRM.Singular + "base64" + string(DB_IN),
		oldRM.Singular + "proxyurl" + string(DB_IN),
	}

	query := `
		SELECT v.Path, p.PropName FROM Versions v
		JOIN Resources r ON v.ResourceSID = r.SID
		JOIN Props p ON p.eSID = v.SID
		WHERE r.ModelSID = ?
		AND p.PropName IN (?, ?, ?, ?)
		AND p.PropValue IS NOT NULL
		LIMIT 1`

	results := Query(reg.tx, query, oldRM.SID,
		names[0], names[1], names[2], names[3])
	defer results.Close()

	row := results.NextRow()
	if row != nil {
		versionPath := "/" + string((*(row[0])).([]byte))
		propName := strings.TrimRight(string((*(row[1])).([]byte)),
			string(DB_IN))
		return NewXRError("hasdocument_enable_violation", versionPath,
			"name="+propName)
	}

	return nil
}

// clearValidationSystemProps bulk-clears the given system prop(s) (e.g.
// "formatvalidated"/"formatvalidatedreason" or "compatibilityvalidated"/
// "compatibilityvalidatedreason") from every Version of every Resource
// instance of the ResourceModel identified by modelSID, in one indexed
// sweep. Called by Model.Save() right after a validateformat/
// validatecompatibility true->false transition is detected, so
// EnsureCompat() (registry/resource.go) no longer needs to defensively
// re-clear these on every single save while validation stays off - this
// one-time, model-change-triggered sweep is the sole owner of clearing
// stale values.
func (reg *Registry) clearValidationSystemProps(modelSID string, names ...string) {
	if len(names) == 0 {
		return
	}

	placeholders := make([]string, len(names))
	args := make([]any, 0, len(names)+4)
	for i, name := range names {
		placeholders[i] = "?"
		args = append(args, name+string(DB_IN))
	}
	args = append(args, reg.DbSID, modelSID, reg.DbSID, modelSID)

	// Clear both the Version's own row AND the Resource-level
	// IsDefaultVerCopy mirror of it (same mirroring mechanism as
	// isdefault/createdat/modifiedat - the Resource-level copy is
	// what HTTP GET on the Resource actually serves).
	Do(reg.tx, `
        DELETE FROM Props
        WHERE PropName IN (`+strings.Join(placeholders, ",")+`)
              AND (
                eSID IN (
                    SELECT SID FROM Versions WHERE ResourceSID IN (
                        SELECT SID FROM Resources
                        WHERE RegistrySID=? AND ModelSID=?))
                OR
                eSID IN (
                    SELECT SID FROM Resources
                    WHERE RegistrySID=? AND ModelSID=?)
              )`, args...)
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
		`e.Abstract=? OR e.Abstract=?`, gAbs, rAbs)
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
			resource.tx.FlushSystemProps()

			resource.tx.AddResource(resource)
		}
	}

	return nil
}
