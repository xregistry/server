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
	// log.FuncPrintf("tx: %s In model.Save - changed: %v",
	// m.Registry.Tx.uuid, m.GetChanged())
	if m.GetChanged() == false {
		return nil
	}

	if log.HasVerbose("ModelSave") || log.IsFuncVerbose() {
		buf, _ := json.MarshalIndent(m, "", "  ")
		log.Printf("tx: %s Saving model:\n%s", m.Registry.Tx.uuid, string(buf))
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

	oldM, xErr := loadModelFromDB(m.Registry, false)
	if xErr != nil {
		return xErr
	}

	if oldM != nil {
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

				props := []string{}

				if oldRM.GetValidateFormat() && !rm.GetValidateFormat() {
					props = append(props, "formatvalidated",
						"formatvalidatedreason")
				}
				if oldRM.GetValidateCompatibility() &&
					!rm.GetValidateCompatibility() {
					props = append(props, "compatibilityvalidated",
						"compatibilityvalidatedreason")
				}
				if len(props) > 0 {
					xErr := rm.clearValidationSystemProps(props)
					if xErr != nil {
						return xErr
					}
				}
			}
		}
	}

	// OLD?
	// Create a temporary type so that we don't use the MarshalJSON func
	// in model.go. That one will exclude "model" from the serialization and
	// we don't want to do that when we're saving it in the DB. We only want
	// to do that when we're serializing the model for the end user.

	// A fresh opaque value every save - lets other requests/replicas
	// (see loadModelFromDB's cache check) cheaply notice this Model row
	// changed without having to compare/parse the full JSON blob.
	changedID := NewUUID()

	xErr = m.Registry.Tx.Backend.SaveModel(m, changedID)
	if xErr != nil {
		return xErr
	}

	// Load old model's ModelEntities so we can diff them
	// map[string]string ; Abstract->SID
	existingModelEntities, xErr :=
		m.Registry.Tx.Backend.GetModelEntitiesByAbstract(m.Registry)
	if xErr != nil {
		return xErr
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

		if len(parts) == 1 {
			count, xErr := m.Registry.Tx.Backend.CountGroupModelUsed(m.Registry, sid)
			if xErr != nil {
				return xErr
			}

			if count > 0 {
				return NewXRError("model_error", "/model",
					"error_detail="+
						fmt.Sprintf(`can't remove Group type %q from the `+
							`model - it still has %d entities. Delete `+
							`them before removing the type`,
							parts[0], count))
			}
		} else {
			count, xErr := m.Registry.Tx.Backend.CountResourceModelUsed(m.Registry, sid)
			if xErr != nil {
				return xErr
			}

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
			xErr := m.Registry.Tx.Backend.DeleteModelEnityByAbstract(
				m.Registry, meAbs)
			if xErr != nil {
				return xErr
			}
		}
	}

	// Now just add the new ones
	for _, gm := range m.Groups {
		gmAbs := "/" + gm.Plural

		// If GroupModel is already in DB then skip it
		if _, ok := existingModelEntities[gmAbs]; !ok {
			// Add new GroupModel
			xErr := m.Registry.Tx.Backend.RegisterModelEntity(gm)
			if xErr != nil {
				return xErr
			}
		}

		for _, rm := range gm.Resources {
			rmAbs := gmAbs + "/" + rm.Plural
			// If ResourceModel is already in DB then skip it
			if _, ok := existingModelEntities[rmAbs]; !ok {
				// Add new ResourceModel
				xErr := m.Registry.Tx.Backend.RegisterModelEntity(rm)
				if xErr != nil {
					return xErr
				}
			}
		}
	}

	m.SetChanged(false)

	// Seed the cache with what we just wrote (only now that changed==false,
	// so cache hits don't hand out a Model that looks dirty) - lets the
	// very next request skip re-fetching+re-parsing what we already have.
	setCachedModel(m.Registry.DbSID, changedID, m)

	return nil
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
	// log.FuncPrintf("tx: %s ApplyNewModel:\n%s", m.Registry.Tx.uuid,
	// ToJSON(newM))

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
	// If verifyData==true then it'll call reg.ValidateAndSave() to make sure
	// we process the changed Reg( touch, if nothing else). But if verifyData
	// is false then the caller will be responsible for calling Val&Save.
	_, xErr := m.Registry.Touch()
	if xErr != nil {
		return xErr
	}

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

		// Not everything goes thru the http code
		if xErr.Instance == "" && m.Registry != nil {
			xErr.Instance = m.Registry.Tx.uuid
		}

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

	vXID, attrName, xErr := reg.Tx.Backend.IsHasDocUsed(reg, oldRM)
	if xErr != nil {
		return xErr
	}

	if vXID != "" {
		return NewXRError("hasdocument_enable_violation", vXID,
			"name="+attrName)
	}

	return nil
}

func (rm *ResourceModel) clearValidationSystemProps(props []string) *XRError {
	if len(props) == 0 {
		return nil
	}

	reg := rm.GroupModel.Model.Registry
	return reg.Tx.Backend.ClearResourceModelSystemProps(rm, props)
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
