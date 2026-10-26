package xrlib

import (
	"encoding/json"
	"fmt"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func init() {
	xErr := registry.RegisterBackend("cli", NewCLIBackend)
	if xErr != nil {
		panic(xErr.String() + xErr.Detail)
	}
}

type CLIBackend struct {
	XRConfig *Config
}

type CLITx struct {
}

var _ registry.Backend = &CLIBackend{}

func NewCLIBackend(c *Config) (registry.Backend, *XRError) {
	return &CLIBackend{
		XRConfig: c,
	}, nil
}

func (CLIBE *CLIBackend) DBExists(c *Config, name string) bool {
	panic("not yet")
	return false
}

func (CLIBE *CLIBackend) DBCreate(c *Config, name string) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) DBDelete(c *Config, name string) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) DBList(c *Config) ([]string, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) Close(c *Config) *XRError {
	return nil
}

func (CLIBE *CLIBackend) NewTx(tx *registry.Tx) *XRError {
	if tx.Tx != nil {
		return nil
	}

	tx.Tx = &CLITx{}

	return nil
}

func (CLIBE *CLIBackend) Commit(tx *registry.Tx) *XRError {
	// update with NewObject/NewSystem.... stuff
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) Rollback(tx *registry.Tx) *XRError {
	// erase all NewObject/NewSystem.... stuff
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) ClearResourceModelSystemProps(rm *registry.ResourceModel, props []string) *XRError {
	// Delete all props all all Versions of all Resource instances of "rm"
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) DeleteGroup(g *registry.Group) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) HasReadOnlyResource(g *registry.Group) (bool, *XRError) {
	panic("not yet")
	return false, nil
}

func (CLIBE *CLIBackend) FindBadEqualsVersions(g *registry.Group, gPP *PropPath, rm *registry.ResourceModel, rPP *PropPath) (string, []string, *XRError) {
	panic("not yet")
	return "", nil, nil
}

func (CLIBE *CLIBackend) FindBadEnumVersions(g *registry.Group, c *registry.Constraint, rm *registry.ResourceModel, rPP *PropPath) (string, []string, *XRError) {
	panic("not yet")
	return "", nil, nil
}

func (CLIBE *CLIBackend) GetResources(g *registry.Group, accessMode int) ([]*registry.Resource, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetResourceIDsByType(g *registry.Group, rType string,
	accessMode int) ([]string, *XRError) {

	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) RegisterEntity(e *registry.Entity) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) LoadCapabilities(r *registry.Registry) (*Capabilities, *XRError) {
	cliReg := (*CLIReg)(r)
	res, xErr := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     cliReg.WithPath("/capabilities"),
		Headers: HTTPHeaders,
		Debug:   log.GetLevel() > 2,
	})
	if xErr != nil {
		return nil, xErr
	}

	tmpCap := (*Capabilities)(nil)
	if err := json.Unmarshal(res.Body, &tmpCap); err != nil {
		return nil, NewXRError("parsing_data", "/capabilities",
			"error_detail="+err.Error()).
			SetDetailf("Response: %s.", string(res.Body))
	}
	r.Capabilities = tmpCap

	return r.Capabilities, nil
}

func (CLIBE *CLIBackend) SaveCapabilities(r *registry.Registry) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) LoadModel(r *registry.Registry) (*registry.Model, *XRError) {
	cliReg := (*CLIReg)(r)
	res, xErr := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     cliReg.WithPath("/model"),
		Headers: HTTPHeaders,
		Debug:   log.GetLevel() > 2,
	})
	if xErr != nil {
		return nil, xErr
	}

	newModel := (*registry.Model)(nil)
	if err := json.Unmarshal(res.Body, &newModel); err != nil {
		return nil, NewXRError("parsing_data", "/model",
			"error_detail="+err.Error()).
			SetDetailf("Response: %s.", string(res.Body))
	}

	newModel.Registry = r
	newModel.ApplyDefaults(r)
	newModel.SetSpecPropsFields()
	newModel.SetPointers()

	return newModel, nil
}

func (CLIBE *CLIBackend) LoadModelSource(r *registry.Registry) ([]byte, *XRError) {
	cliReg := (*CLIReg)(r)

	res, xErr := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     cliReg.WithPath("/modelsource"),
		Headers: HTTPHeaders,
		Debug:   log.GetLevel() > 2,
	})

	if xErr != nil {
		return nil, xErr
	}

	newModelSource := []byte{}

	if res.Code != 404 {
		// We silently ignore 404 for modelsource
		// TODO look for it at {"modelsource"} instead
		if xErr != nil {
			return nil, xErr
		}

		srcModel := registry.Model{}

		if err := json.Unmarshal(res.Body, &srcModel); err != nil {
			return nil, NewXRError("parsing_data", "/modelsource",
				"error_detail="+err.Error()).
				SetDetailf("Response: %s.", string(res.Body))
		}

		newModelSource = res.Body
	}

	return newModelSource, nil
}

func (CLIBE *CLIBackend) SaveModel(r *registry.Registry) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) SaveModelSource(r *registry.Registry) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) RegisterGroupModel(gm *registry.GroupModel) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) RegisterResourceModel(rm *registry.ResourceModel) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) DeleteModelEnityByAbstract(r *registry.Registry, abstract string) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) GetModelEntitiesByAbstract(r *registry.Registry) (map[string]string, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) CountGroupModelUsed(r *registry.Registry, gmSID string) (int, *XRError) {
	panic("not yet")
	return 0, nil
}

func (CLIBE *CLIBackend) CountResourceModelUsed(r *registry.Registry, rmSID string) (int, *XRError) {
	panic("not yet")
	return 0, nil
}

// versionXID, attrName, error
func (CLIBE *CLIBackend) IsHasDocUsed(r *registry.Registry, rm *registry.ResourceModel) (string, string, *XRError) {
	panic("not yet")
	return "", "", nil
}

// versionXID, , error
func (CLIBE *CLIBackend) HasDocContent(r *registry.Resource) (string, *XRError) {
	panic("not yet")
	return "", nil
}

func (CLIBE *CLIBackend) RefreshEntity(e *registry.Entity, accessMode int) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) ClearUserProps(e *registry.Entity) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) ListRegistries(tx *registry.Tx) ([]string, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) DeleteRegistry(r *registry.Registry) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) MapRegistryUID2SID(tx *registry.Tx, uid string) (string, *XRError) {
	panic("not yet")
	return "", nil
}

func (CLIBE *CLIBackend) RegistryGetUsesXref(r *registry.Registry) (bool, *XRError) {
	panic("not yet")
	return false, nil
}

func (CLIBE *CLIBackend) RegistrySetUsesXref(r *registry.Registry, b bool) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) RegistryRecalcUsesXref(r *registry.Registry) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) GetGroups(r *registry.Registry, accessMode int) ([]*registry.Group, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetGroupIDsByType(r *registry.Registry, gType string,
	accessMode int) ([]string, *XRError) {

	panic("not yet")
	return nil, nil
}

// regUID, entityXID, err
func (CLIBE *CLIBackend) MapShortSelf(tx *registry.Tx, ss string) (string, string, *XRError) {
	panic("not yet")
	return "", "", nil
}

func (CLIBE *CLIBackend) GenerateResultEntities(reg *registry.Registry, what string, XIDs []string) (registry.EntityIterator, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) LockResourceBySID(tx *registry.Tx, sid string) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) DeleteResource(r *registry.Resource) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) GetResourceContents(e *registry.Entity) ([]byte, *XRError) {
	if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION {
		panic("not yet")
		return []byte("Placeholder"), nil
	}
	panic(fmt.Sprintf("%s: I'm not a Resource or Version", e.XID))
}

func (CLIBE *CLIBackend) SetResourceContents(e *registry.Entity, val []byte) *XRError {
	panic("not yet")
	return nil
}

// recalc all R's versions
func (CLIBE *CLIBackend) RecalcVersionsIsDefault(r *registry.Resource) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) DeleteResourceDefaultVersionProps(r *registry.Resource) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) CopyResourceDefaultVersionProps(r *registry.Resource) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) FindBadAncestorsCreatedAt(r *registry.Resource, lock bool) ([]*registry.AncestorVersion, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) FindBadAncestorsModifiedAt(r *registry.Resource, lock bool) ([]*registry.AncestorVersion, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) FindBadAncestorsSemVer(r *registry.Resource, lock bool) ([]*registry.AncestorVersion, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) FindNewestVersionID(r *registry.Resource, excludeTBD bool) (string, *XRError) {
	panic("not yet")
	return "", nil
}

func (CLIBE *CLIBackend) GetVersionIDs(r *registry.Resource) ([]*registry.VersionAncestor, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetRootVersionIDs(r *registry.Resource) ([]string, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetProblematicVersions(r *registry.Resource) ([]*registry.VersionAncestor, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetChildVersionIDs(r *registry.Resource,
	parentVID string) ([]string, *XRError) {

	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetNumberOfVersions(r *registry.Resource) (int, *XRError) {
	panic("not yet")
	return 0, nil
}

func (CLIBE *CLIBackend) GetVersions(r *registry.Resource,
	accessMode int) ([]*registry.Version, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetXrefSources(r *registry.Resource) ([]string, *XRError) {
	panic("not yet")
	return nil, nil
}

// numValues, numNonEmptyUnique, numEmpty, error
func (CLIBE *CLIBackend) CountNonMatchingVersions(r *registry.Resource,
	vAttrPath *PropPath) (int, int, int, *XRError) {

	panic("not yet")
	return 0, 0, 0, nil
}

func (CLIBE *CLIBackend) DeleteMeta(meta *registry.Meta) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) DeleteVersion(v *registry.Version) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) ClearXrefState(meta *registry.Meta) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) CopyXrefState(meta *registry.Meta) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) CopyXrefDefaultVersionProps(r *registry.Resource) *XRError {
	panic("not yet")
	return nil
}

// copy xref vers
func (CLIBE *CLIBackend) CopyXrefVersions(r *registry.Resource, tgtSID string) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) GetEntityFromXID(tx *registry.Tx, regSID string, xidStr string,
	allAttrs bool, anyCase bool, accessMode int) (*registry.Entity, *XRError) {

	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetEntitiesFromXID(tx *registry.Tx, regSID string,
	xidStr string, allAttrs bool, children bool, anyCase bool,
	accessMode int) (registry.EntityIterator, *XRError) {

	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) SaveEntity(e *registry.Entity) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) SaveChangedSystemProps(e *registry.Entity, changed map[string]any) *XRError {
	panic("not yet")
	return nil
}
