package registry

import (
	"fmt"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

const DEFAULT_BACKEND = "mysql"

// BackendOption describes a single configuration option of a backend. It's
// set via "backend.<profile>.<Name>" in the config file, or via the
// --backend / XR_BACKEND "name=value" syntax.
type BackendOption struct {
	Name        string
	Default     string
	Description string
	Secret      bool // Don't show the value/default in listings
}

// DBNameOption is understood by all backends
var DBNameOption = BackendOption{
	Name:        "dbname",
	Default:     DefDBName,
	Description: "Name of the DB to use",
}

type BackendInfo struct {
	Name    string
	NewFunc func(c *Config) (Backend, *XRError)
	Options []BackendOption
}

var RegisteredBackends = map[string]*BackendInfo{} // Name->BackendInfo

func RegisterBackend(name string, fn func(c *Config) (Backend, *XRError),
	opts ...BackendOption) *XRError {

	log.VPrintf(3, "Registering backend: %s", name)
	if RegisteredBackends[name] != nil {
		return NewXRError("server_error", "/").
			SetDetailf("Duplicate backend registered: %s.", name)
	}

	RegisteredBackends[name] = &BackendInfo{
		Name:    name,
		NewFunc: fn,
		Options: opts,
	}

	return nil
}

func NewBackend(c *Config) (Backend, *XRError) {
	backendName := BackendName(c)

	if len(RegisteredBackends) == 0 {
		return nil, NewXRError("server_error", "/").
			SetDetailf("No backends registered.")
	}

	be := RegisteredBackends[backendName]
	if be == nil {
		return nil, NewXRError("server_error", "/").
			SetDetailf("Backend (%s) can't be found.", backendName)
	}

	return be.NewFunc(c)
}

// BackendProfile returns the name of the active backend profile, which is
// the config's "backend" value (or the default backend).
func BackendProfile(c *Config) string {
	name := c.GetAsString("backend")
	if name == "" {
		name = DEFAULT_BACKEND

		// If there's only one backend registered, use it
		if len(RegisteredBackends) == 1 {
			name = Keys(RegisteredBackends)[0]
		}
	}
	return name
}

// BackendName returns the backend type of the active profile. A profile's
// type is "backend.<profile>.type", which defaults to the profile's name.
func BackendName(c *Config) string {
	profile := BackendProfile(c)
	if t := c.GetAsString("backend." + profile + ".type"); t != "" {
		return t
	}
	return profile
}

func backendOptionSpec(c *Config, name string) *BackendOption {
	if be := RegisteredBackends[BackendName(c)]; be != nil {
		for i := range be.Options {
			if be.Options[i].Name == name {
				return &be.Options[i]
			}
		}
	}
	return nil
}

// BackendOpt returns the value of the named option of the active backend
// profile, or the option's default if it hasn't been set.
func BackendOpt(c *Config, name string) string {
	key := "backend." + BackendProfile(c) + "." + name
	if v := c.GetAsString(key); v != "" {
		return v
	}
	if spec := backendOptionSpec(c, name); spec != nil {
		return spec.Default
	}
	return ""
}

func SetBackendOpt(c *Config, name, value string) {
	c.Set("backend."+BackendProfile(c)+"."+name, value)
}

// ParseBackendSpec applies a "[NAME,]opt=val,..." string (from --backend or
// XR_BACKEND) to the config. A leading item without a "=" selects the
// backend profile, otherwise the options apply to the active profile. A comma
// in a value can be escaped as "\,".
func ParseBackendSpec(c *Config, spec string) *XRError {
	items := []string{}
	cur := strings.Builder{}
	for i := 0; i < len(spec); i++ {
		if spec[i] == '\\' && i+1 < len(spec) && spec[i+1] == ',' {
			cur.WriteByte(',')
			i++
		} else if spec[i] == ',' {
			items = append(items, cur.String())
			cur.Reset()
		} else {
			cur.WriteByte(spec[i])
		}
	}
	items = append(items, cur.String())

	for i, item := range items {
		name, value, hasVal := strings.Cut(item, "=")
		name = strings.TrimSpace(name)
		if !hasVal {
			if i != 0 || name == "" {
				return NewXRError("server_error", "/").SetDetailf(
					"Invalid backend option %q, expected NAME=VALUE.", item)
			}
			c.Set("backend", name)
			continue
		}
		if name == "" {
			return NewXRError("server_error", "/").SetDetailf(
				"Invalid backend option %q, missing name.", item)
		}
		SetBackendOpt(c, name, value)
	}
	return nil
}

// ValidateBackendConfig makes sure the active backend exists and that all of
// the options set for its profile are ones that it knows about.
func ValidateBackendConfig(c *Config) *XRError {
	profile := BackendProfile(c)
	typ := BackendName(c)
	be := RegisteredBackends[typ]
	if be == nil {
		names := SortedKeys(RegisteredBackends)
		return NewXRError("server_error", "/").SetDetailf(
			"Unknown backend %q (available: %s).", typ,
			strings.Join(names, ", "))
	}

	prefix := "backend." + profile + "."
	bad := []string{}
	for _, key := range SortedKeys(c.Data) {
		opt, ok := strings.CutPrefix(key, prefix)
		if !ok || opt == "type" {
			continue
		}
		if backendOptionSpec(c, opt) == nil {
			bad = append(bad, opt)
		}
	}
	if len(bad) > 0 {
		valid := []string{}
		for _, o := range be.Options {
			valid = append(valid, o.Name)
		}
		return NewXRError("server_error", "/").SetDetailf(
			"Unknown option(s) for backend %q: %s (valid: %s).", profile,
			strings.Join(bad, ", "), strings.Join(valid, ", "))
	}
	return nil
}

// ListBackends returns the registered backends and their options, either as
// a human readable table or as JSON.
func ListBackends(c *Config, output string) string {
	type optInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Default     string `json:"default,omitempty"`
	}
	type beInfo struct {
		Name    string    `json:"name"`
		Active  bool      `json:"active"`
		Default bool      `json:"default"`
		Options []optInfo `json:"options"`
	}

	names := SortedKeys(RegisteredBackends)
	active := BackendName(c)

	list := []beInfo{}
	for _, name := range names {
		bi := beInfo{Name: name, Active: name == active,
			Default: name == DEFAULT_BACKEND, Options: []optInfo{}}
		for _, o := range RegisteredBackends[name].Options {
			oi := optInfo{Name: o.Name, Description: o.Description}
			if !o.Secret {
				oi.Default = o.Default
			}
			bi.Options = append(bi.Options, oi)
		}
		list = append(list, bi)
	}

	if output == "json" {
		return ToJSON(list) + "\n"
	}

	buf := strings.Builder{}
	for i, bi := range list {
		if i > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(bi.Name)
		if bi.Active {
			buf.WriteString(" (active)")
		}
		if bi.Default {
			buf.WriteString(" (default)")
		}
		buf.WriteString("\n  Options:\n")
		for _, o := range bi.Options {
			def := ""
			if o.Default != "" {
				def = fmt.Sprintf(" (default: %s)", o.Default)
			}
			fmt.Fprintf(&buf, "  - %s: %s%s\n", o.Name, o.Description, def)
		}
	}
	return buf.String()
}

func MustNewBackend(c *Config) Backend {
	be, xErr := NewBackend(c)
	if xErr != nil {
		panic(xErr)
	}
	return be
}

type Backend interface {
	// DB related stuff
	/////////////////////////////////////////////////////////////////////
	DBExists(c *Config, name string) bool
	DBCreate(c *Config, name string) *XRError
	DBDelete(c *Config, name string) *XRError
	DBList(c *Config) ([]string, *XRError)

	DBConnectionString(c *Config) (string, *XRError)

	// Close releases any long-lived resources (e.g. connection pools) that
	// the backend shares across all Backend instances for this Config.
	// It's safe to use the backend again afterwards, it'll just reconnect.
	Close(c *Config) *XRError

	// Tx related stuff
	/////////////////////////////////////////////////////////////////////
	NewTx(tx *Tx) *XRError
	Commit(tx *Tx) *XRError
	Rollback(tx *Tx) *XRError

	// Model related stuff
	/////////////////////////////////////////////////////////////////////

	// Clear the given systemProps from all versions/resources of rm type
	ClearResourceModelSystemProps(rm *ResourceModel, props []string) *XRError

	// Make Entity known to the backend/DB
	// Doesn't save any user props yet, just make it known. It will creaet
	// some system managed props though:
	// all: xid
	// version: RESOURCEid, isdefault
	RegisterEntity(e *Entity) *XRError

	// Populate r.Capabilities with latest
	// If nil, the backend can decide what to return: nil vs defaultCaps
	LoadCapabilities(r *Registry) (*Capabilities, *XRError)
	// Store r.Capabilities to DB
	SaveCapabilities(r *Registry) *XRError

	// Load model from DB but do not store it in r.Model yet.
	LoadModel(r *Registry) (*Model, *XRError)
	// Load source from DB but do not store it in r.Source yet.
	LoadModelSource(r *Registry) ([]byte, *XRError)

	// Save r.Model to DB
	SaveModel(r *Registry) *XRError
	SaveModelSource(r *Registry) *XRError

	// Add GM/RM to their Registry model in the DB
	RegisterGroupModel(gm *GroupModel) *XRError
	RegisterResourceModel(rm *ResourceModel) *XRError

	// DeleteModelEntitiesByAbstact must delete all instances of that type
	// May need to create one for GM and one for RM
	DeleteModelEnityByAbstract(reg *Registry, abstract string) *XRError

	// map(Abs->SID)
	// Load all model entities (GM, RM) by abstract, along with their SIDs
	GetModelEntitiesByAbstract(r *Registry) (map[string]string, *XRError)

	// How many Group instances use the GM SID?
	CountGroupModelUsed(r *Registry, gmSID string) (int, *XRError)
	// How many Resources use the RM SID? Across all Group instances
	CountResourceModelUsed(r *Registry, rmSID string) (int, *XRError)

	// Find all Versions of instances of RM that have document, either in
	// any of the RESOURCE* props
	// versionXID, attribugName, error
	IsHasDocUsed(r *Registry, rm *ResourceModel) (string, string, *XRError)

	// versionXID of version that has doc content
	HasDocContent(r *Resource) (string, *XRError)

	// Just update e's Object - everything else will be updated by other code
	RefreshEntity(e *Entity, accessMode int) *XRError

	ClearUserProps(e *Entity) *XRError

	// This might need to be changed later if it ends up this is not possible
	// for all backends (or is too table/sql dependent)
	// Result: [](XID, PropName, PropValue as string)
	// This returns all non-generated/calucated props.
	// Used by TestMiscDBRows
	ListProps(r *Registry, all bool) ([][3]string, *XRError)

	// Registry stuff
	ListRegistries(tx *Tx) ([]string, *XRError) // Order by UID,ASC case-insen.

	DeleteRegistry(r *Registry) *XRError // Recursive delete (data and model)
	MapRegistryUID2SID(tx *Tx, uid string) (string, *XRError)
	RegistryGetUsesXref(r *Registry) (bool, *XRError)
	RegistrySetUsesXref(r *Registry, b bool) *XRError
	RegistryRecalcUsesXref(r *Registry) *XRError
	GetGroups(r *Registry, accessMode int) ([]*Group, *XRError)
	GetGroupIDsByType(r *Registry, gType string,
		accessMode int) ([]string, *XRError)
	// regUID, entityXID, err
	MapShortSelf(tx *Tx, ss string) (string, string, *XRError)
	GenerateResultEntities(reg *Registry, what string, XIDs []string) (EntityIterator, *XRError)

	// Group stuff
	DeleteGroup(g *Group) *XRError // recursive delete
	HasReadOnlyResource(g *Group) (bool, *XRError)
	FindBadEqualsVersions(g *Group, gPP *PropPath, rm *ResourceModel, rPP *PropPath) (string, []string, *XRError) // rID, vIDs, err
	FindBadEnumVersions(g *Group, c *Constraint, rm *ResourceModel, rPP *PropPath) (string, []string, *XRError)   // rID, vIDs, err
	GetResources(g *Group, accessMode int) ([]*Resource, *XRError)
	GetResourceIDsByType(g *Group, rType string,
		accessMode int) ([]string, *XRError)

	// Resource stuff
	LockResourceBySID(tx *Tx, sid string) *XRError
	DeleteResource(r *Resource) *XRError // recursive delete/fixup Reg.UsesXref
	GetResourceContents(e *Entity) ([]byte, *XRError)
	SetResourceContents(e *Entity, val []byte) *XRError

	RecalcVersionsIsDefault(r *Resource) *XRError // recalc all R's versions
	DeleteResourceDefaultVersionProps(r *Resource) *XRError
	CopyResourceDefaultVersionProps(r *Resource) *XRError

	FindBadAncestorsCreatedAt(r *Resource, lock bool) ([]*AncestorVersion, *XRError)
	FindBadAncestorsModifiedAt(r *Resource, lock bool) ([]*AncestorVersion, *XRError)
	FindBadAncestorsSemVer(r *Resource, lock bool) ([]*AncestorVersion, *XRError)

	FindNewestVersionID(r *Resource, excludeTBD bool) (string, *XRError)
	GetVersionIDs(r *Resource) ([]*VersionAncestor, *XRError)
	GetRootVersionIDs(r *Resource) ([]string, *XRError)
	GetProblematicVersions(r *Resource) ([]*VersionAncestor, *XRError)
	GetChildVersionIDs(r *Resource, parentVID string) ([]string, *XRError)
	GetNumberOfVersions(r *Resource) (int, *XRError)
	GetVersions(r *Resource, accessMode int) ([]*Version, *XRError)

	GetXrefSources(r *Resource) ([]string, *XRError)

	// numValues, numNonEmptyUnique, numEmpty, error
	CountNonMatchingVersions(r *Resource, vAttrPath *PropPath) (int, int, int, *XRError)

	// Meta stuff
	DeleteMeta(meta *Meta) *XRError // recursive delete

	// Version stuff
	DeleteVersion(v *Version) *XRError // recursive delete & ResourceContent

	// Xref stuff
	ClearXrefState(meta *Meta) *XRError                   // in prep for new/del xref
	CopyXrefState(meta *Meta) *XRError                    // copy xref meta props & vers
	CopyXrefDefaultVersionProps(r *Resource) *XRError     // copy xref def ver
	CopyXrefVersions(r *Resource, tgtSID string) *XRError // copy xref vers

	// Misc Entity stuff

	// Does entity exist by any cased-XID?
	// real (case sensitive XID), error. ""=does not exist
	EntityRealXID(tx *Tx, xidStr string, accessMode int) (string, *XRError)

	// needs to call PropulateOriginObject
	GetEntityFromXID(tx *Tx, regSID string, xidStr string, allAttrs bool,
		accessMode int) (*Entity, *XRError)
	// needs to call PropulateOriginObject
	GetEntitiesFromXID(tx *Tx, regSID string, xidStr string, allAttrs bool,
		children bool, accessMode int) (EntityIterator, *XRError)

	// SaveEntity needs to also update reg.UsesXref if resource.xref changes
	SaveEntity(e *Entity) *XRError
	SaveChangedSystemProps(e *Entity, changed map[string]any) *XRError
}

// TODO see if we can use VersionAncestor instead
type AncestorVersion struct {
	VersionUID string
	AncestorID string
}

type VersionAncestor struct {
	VID        string
	AncestorID string
	CreatedAt  string
}

type EntityIterator interface {
	Next() (*Entity, *XRError) // nil,nil if at ened, not nil, Error
	Close()
	Offset() uint64 // of next entity (zero before first Next(), -1 if unknown
	Size() uint64   // total # of entities in iteration (-1 if unknown)
	Total() uint64  // Size = this iter, Total = all pages(iters) in DB
	// if not paging, then total=math.MaxUint64
}
