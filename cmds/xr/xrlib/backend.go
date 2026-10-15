package xrlib

import (
	"fmt"

	// log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

type CLIBackend struct {
	XRConfig *Config
}

var _ Backend = &CLIBackend{}

func NewCLIBackend(c *Config) Backend {
	return &CLIBackend{
		XRConfig: c,
	}
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

func (CLIBE *CLIBackend) NewTx(tx *Tx) *XRError {
	return nil
}

func (CLIBE *CLIBackend) Commit(tx *Tx) *XRError {
	// update with NewObject/NewSystem.... stuff
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) Rollback(tx *Tx) *XRError {
	// erase all NewObject/NewSystem.... stuff
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) ClearResourceModelSystemProps(rm *ResourceModel, props []string) *XRError {
	// Delete all props all all Versions of all Resource instances of "rm"
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) DeleteGroup(g *Group) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) HasReadOnlyResource(g *Group) (bool, *XRError) {
	panic("not yet")
	return false, nil
}

func (CLIBE *CLIBackend) FindBadEqualsVersions(g *Group, gPP *PropPath, rm *ResourceModel, rPP *PropPath) (string, []string, *XRError) {
	panic("not yet")
	return "", nil, nil
}

func (CLIBE *CLIBackend) FindBadEnumVersions(g *Group, c *Constraint, rm *ResourceModel, rPP *PropPath) (string, []string, *XRError) {
	panic("not yet")
	return "", nil, nil
}

func (CLIBE *CLIBackend) GetResources(g *Group, accessMode int) ([]*Resource, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) RegisterEntity(e *Entity) *XRError {
	panic("not yet")
	return nil
}

// JSON, uuid
func (CLIBE *CLIBackend) LoadModelJSON(r *Registry) (string, string, *XRError) {
	panic("not yet")
	return "", "", nil
}

func (CLIBE *CLIBackend) SaveModel(m *Model, changeUUID string) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) RegisterModelEntity(me any) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) DeleteModelEnityByAbstract(r *Registry, abstract string) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) GetModelEntitiesByAbstract(r *Registry) (map[string]string, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) CountGroupModelUsed(r *Registry, gmSID string) (int, *XRError) {
	panic("not yet")
	return 0, nil
}

func (CLIBE *CLIBackend) CountResourceModelUsed(r *Registry, rmSID string) (int, *XRError) {
	panic("not yet")
	return 0, nil
}

// versionXID, attrName, error
func (CLIBE *CLIBackend) IsHasDocUsed(r *Registry, rm *ResourceModel) (string, string, *XRError) {
	panic("not yet")
	return "", "", nil
}

// versionXID, , error
func (CLIBE *CLIBackend) HasDocContent(r *Resource) (string, *XRError) {
	panic("not yet")
	return "", nil
}

func (CLIBE *CLIBackend) RefreshEntity(e *Entity, accessMode int) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) ClearUserProps(e *Entity) *XRError {
	panic("not yet")
	return nil
}

// args is chunks of 13 any's per row/entity
func (CLIEBE *CLIBackend) BatchUpdateProps(e *Entity, isSystem bool, args []any) *XRError {
	panic("not yet")
	return nil
}

// args is chunks of any's (names) per row/entity
func (CLIEBE *CLIBackend) BatchDeleteProps(e *Entity, args []any) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) ListRegistries(tx *Tx) ([]string, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) DeleteRegistry(r *Registry) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) MapRegistryUID2SID(tx *Tx, uid string) (string, *XRError) {
	panic("not yet")
	return "", nil
}

func (CLIBE *CLIBackend) RegistryGetUsesXref(r *Registry) (bool, *XRError) {
	panic("not yet")
	return false, nil
}

func (CLIBE *CLIBackend) RegistrySetUsesXref(r *Registry, b bool) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) RegistryRecalcUsesXref(r *Registry) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) GetGroups(r *Registry, accessMode int) ([]*Group, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) DeleteResource(r *Resource) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) GetResourceContents(e *Entity) ([]byte, *XRError) {
	if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION {
		panic("not yet")
		return []byte("Placeholder"), nil
	}
	panic(fmt.Sprintf("%s: I'm not a Resource or Version", e.XID))
}

func (CLIBE *CLIBackend) SetResourceContents(e *Entity, val []byte) *XRError {
	panic("not yet")
	return nil
}

// recalc all R's versions
func (CLIBE *CLIBackend) RecalcVersionsIsDefault(r *Resource) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) DeleteResourceDefaultVersionProps(r *Resource) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) CopyResourceDefaultVersionProps(r *Resource) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) FindBadAncestorsCreatedAt(r *Resource, lock bool) ([]*AncestorVersion, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) FindBadAncestorsModifiedAt(r *Resource, lock bool) ([]*AncestorVersion, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) FindBadAncestorsSemVer(r *Resource, lock bool) ([]*AncestorVersion, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) FindNewestVersionID(r *Resource, excludeTBD bool) (string, *XRError) {
	panic("not yet")
	return "", nil
}

func (CLIBE *CLIBackend) GetVersionIDs(r *Resource) ([]*VersionAncestor, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetRootVersionIDs(r *Resource) ([]string, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetProblematicVersions(r *Resource) ([]*VersionAncestor, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetChildVersionIDs(r *Resource,
	parentVID string) ([]string, *XRError) {

	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetNumberOfVersions(r *Resource) (int, *XRError) {
	panic("not yet")
	return 0, nil
}

func (CLIBE *CLIBackend) GetVersions(r *Resource,
	accessMode int) ([]*Version, *XRError) {
	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetXrefSources(r *Resource) ([]string, *XRError) {
	panic("not yet")
	return nil, nil
}

// numValues, numNonEmptyUnique, numEmpty, error
func (CLIBE *CLIBackend) CountNonMatchingVersions(r *Resource,
	vAttrPath *PropPath) (int, int, int, *XRError) {

	panic("not yet")
	return 0, 0, 0, nil
}

func (CLIBE *CLIBackend) DeleteMeta(meta *Meta) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) DeleteVersion(v *Version) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) ClearXrefState(meta *Meta) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) CopyXrefState(meta *Meta) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) CopyXrefDefaultVersionProps(r *Resource) *XRError {
	panic("not yet")
	return nil
}

// copy xref vers
func (CLIBE *CLIBackend) CopyXrefVersions(r *Resource, tgtSID string) *XRError {
	panic("not yet")
	return nil
}

func (CLIBE *CLIBackend) GetEntityFromXID(tx *Tx, regSID string, xidStr string,
	allAttrs bool, anyCase bool, accessMode int) (*Entity, *XRError) {

	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) GetEntitiesFromXID(tx *Tx, regSID string,
	xidStr string, allAttrs bool, children bool, anyCase bool,
	accessMode int) (EntityIterator, *XRError) {

	panic("not yet")
	return nil, nil
}
