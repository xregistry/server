package registry

import (
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

type EntitySetter interface {
	Get(name string) any
	JustSet(name string, val any) *XRError
	SetSave(name string, val any) *XRError
	Delete() *XRError
}

type Entity struct {
	Tx           *Tx       `json:"-"`
	AccessMode   int       // FOR_READ, FOR_WRITE
	Registry     *Registry `json:"-"`
	SID          string    // Entity's SID
	ParentSID    string    // Parent entity's SID ("" for the Registry root)
	Plural       string
	Singular     string
	UID          string         // Entity's UID
	Object       map[string]any `json:"-"`
	NewObject    map[string]any `json:"-"` // updated version, save() will store
	OriginObject map[string]any `json:"-"` // pre-Tx values
	// I'm not sure if OriginObject is the best way to go. I added it so that
	// EnsureCompat could view the original value of "compat" but I also could
	// have just set a flag on Meta when "compat" changed (via the
	// UpdateFn(), and then avoid cloning the entire Object. Just something
	// to think about for the future

	// System/NewSystem mirror Object/NewObject but for "system" props
	// (currently just formatvalidated/formatvalidatedreason/
	// compatibilityvalidated/compatibilityvalidatedreason - the
	// IsSystemProp=true attrs). These are kept separate because setting
	// them must NOT bump epoch/modifiedat (so they can't just live in
	// Object/NewObject). SetSystemDBProperty() buffers changes into
	// NewSystem (like Set() buffers into NewObject); SaveSystemProps()
	// flushes NewSystem to the DB (and re-runs the default-version
	// cascade, if needed) at most once per entity per Tx, at commit
	// time - see tx.WriteCache().
	System       map[string]any `json:"-"`
	NewSystem    map[string]any `json:"-"`
	OriginSystem map[string]any `json:"-"` // pre-Tx values,mirrors OriginObject

	// OriginSystem is captured once, by EnsureNewSystem(), the first
	// time SetSystemDBProperty() genuinely buffers a change this Tx
	// (mirrors OriginObject/EnsureNewObject()). Like OriginObject, it's
	// never touched by Refresh(), so comparing it against System at any
	// later point in the Tx (e.g. Resource.runCascade()'s cascade-skip
	// optimization) reliably answers "did a system prop really change
	// this Tx" regardless of any mid-Tx Refresh() calls - unlike
	// EpochSet, which Refresh() resets as part of reloading.

	// These were added just for convenience and so we can use the same
	// struct for traversing the SQL results
	Type     int    // ENTITY_REGISTRY(0)/GROUP(1)/RESOURCE(2)/VERSION(3)/...
	XID      string // /[GROUPS[/GID[/RESOURCES[/RID[/versions[/vID]]]]]]
	Abstract string // [GROUPS[,RESOURCES[,versions]]]
	EpochSet bool   `json:"-"` // Has epoch changed this tx?
	ModSet   bool   `json:"-"` // Has modifiedat changed this tx?
	Self     any    `json:"-"` // ptr to typed Entity (e.g. *Group)

	// Save these values in memory so we only need to get them once
	GroupModel    *GroupModel    `json:"-"` // gModel if it's not a Registry
	ResourceModel *ResourceModel `json:"-"` // If Res,Ver,Meta save rmModel

	// Debugging
	NewObjectStack []string `json:"-"` // stack when NewObj created via Ensure

	// Extensions to 'common'
	stuff map[string]any
}

func (e *Entity) GetStuff(key string) (any, bool) {
	if e.stuff == nil {
		return nil, false
	}

	val, ok := e.stuff[key]
	return val, ok
}

func (e *Entity) GetStuffAsString(key string) string {
	val, ok := e.GetStuff(key)
	if !ok {
		return ""
	}
	return val.(string)
}

func (e *Entity) SetStuff(key string, val any) {
	if IsNil(val) && e.stuff == nil {
		return
	}

	if e.stuff == nil {
		e.stuff = map[string]any{}
	}

	if IsNil(val) {
		delete(e.stuff, key)
	} else {
		e.stuff[key] = val
	}
}

type Registry struct {
	Entity
	Model        *Model        `json:"-"` // nil=not loaded
	ModelSource  []byte        `json:"-"` // nil=not loaded, {}=empty
	Capabilities *Capabilities `json:"-"` // nil=not loaded

	oldCapabilities *Capabilities // as seen at the start of the tx

	// UsesXref mirrors Registries.UsesXref (a raw DB column, NOT a
	// model/user-visible attribute - never serialized, never bumps
	// epoch/modifiedAt). Loaded once per Tx by FindRegistryBySID().
	// Lets Resource.runCascade() skip the xref-target fan-out check
	// entirely for Registries that never use xref. See init.sql's
	// Registries.UsesXref doc comment for the full design.
	UsesXref bool `json:"-"`
}

type Group struct {
	Entity
	Registry *Registry

	// "constraints" is the full set of constraints for the group.
	// - the ones defined at the instance + groupType/model level
	// nil means we haven't built the list of constraints yet.
	// Zero-len  map means we've done the analysis & there are no constraints.
	// So nil is NOT the same as {} !!!
	constraints map[string]*Constraint

	// there are just the ones defined at the group instance level.
	// basically just parse "constraints" attribute
	groupConstraints map[string]*Constraint
}

type Resource struct {
	Entity
	Group *Group
}

type Meta struct {
	Entity
	Resource *Resource
}

type Version struct {
	Entity
	Resource *Resource
}

// This allows for us to choose the order and define custom logic per prop
var OrderedSpecProps = []*Attribute{}

func InitOrderedSpecProps() {
	OrderedSpecProps = []*Attribute{
		{
			Name:     "specversion",
			Type:     STRING,
			ReadOnly: true,
			Required: true,
			Default:  SPECVERSION,

			Internals: &AttrInternals{
				Types:       StrTypes(ENTITY_REGISTRY),
				DontStore:   true,
				UIMonospace: true,
				UILabel:     "Spec Version",

				GetFn: func(e *Entity) any {
					return SPECVERSION
				},
				CheckFn: func(e *Entity) *XRError {
					tmp := e.NewObject["specversion"]
					if !IsNil(tmp) && tmp != "" {
						sv, _ := tmp.(string)
						norm := NormalizeSpecVersion(sv)
						if norm != NormalizeSpecVersion(SPECVERSION) {
							return NewXRError("invalid_attribute", e.XID,
								"name=specversion",
								"error_detail="+
									fmt.Sprintf("invalid value: %v", tmp))
						}
					}
					return nil
				},
			},
		},
		{
			Name:      "id",
			Type:      STRING,
			Immutable: true,
			Required:  true,

			Internals: &AttrInternals{
				Types:        "", // Yes even ENTITY_RESOURCE
				XrefRequired: true,
				UIMonospace:  true,

				CheckFn: func(e *Entity) *XRError {
					singular := e.Singular
					// PanicIf(singular == "", "singular is '' :  %v", e)
					if e.Type == ENTITY_VERSION || e.Type == ENTITY_META {
						_, rm := e.GetModels()
						singular = rm.Singular
					}
					justSingular := singular
					singular += "id"

					oldID := any(e.UID)
					if e.Type == ENTITY_VERSION || e.Type == ENTITY_META {
						// Grab rID from /GROUPs/gID/RESOURCEs/rID/versions/vID
						parts := strings.Split(e.XID, "/")
						oldID = parts[4]
					}
					newID := any(e.NewObject[singular])

					if IsNil(newID) {
						return nil // Not trying to be updated, so skip it
					}

					if newID == "" {
						return NewXRError("invalid_attribute", e.XID,
							"name="+singular,
							"error_detail=can't be an empty string")
					}

					if xErr := IsValidID(newID.(string), singular); xErr != nil {
						xErr.Subject = e.XID
						return xErr
					}
					if oldID != "" && !IsNil(oldID) && newID != oldID {
						info := e.GetRequestInfo()
						if info.HasIgnore("id") && "/"+info.Root == e.XID {
							// Don't check and reset to old ID (if changed)
							e.NewObject[singular] = oldID
						} else {
							return NewXRError("mismatched_id", e.XID,
								"singular="+justSingular,
								"expected_id="+fmt.Sprintf("%v", oldID),
								"invalid_id="+fmt.Sprintf("%v", newID))
						}
					}
					return nil
				},
				UpdateFn: func(e *Entity) *XRError {
					// Make sure the ID is always set
					singular := e.Singular
					if e.Type == ENTITY_VERSION || e.Type == ENTITY_META {
						singular = e.GetResourceSingular()
					}
					singular += "id"

					if e.Type == ENTITY_VERSION {
						// Versions shouldn't store the RESOURCEid
						delete(e.NewObject, singular)
					} else if IsNil(e.NewObject[singular]) {
						log.Printf("tx: %s (%s) %q is nil - "+
							"e.UID=%q e.Type=%d e.Object=%s e.NewObject=%s "+
							"e.OriginObject=%s",
							e.Tx.UUID, e.XID, singular, e.UID, e.Type,
							ToJSON(e.Object), ToJSON(e.NewObject),
							ToJSON(e.OriginObject))
						panic(fmt.Sprintf(`tx: %s (%s) %q is nil - `+
							`that's bad, fix it!`,
							e.Tx.UUID, e.XID, singular))
					}
					return nil
				},
			},
		},
		{
			Name:      "versionid",
			Type:      STRING,
			Immutable: true,
			Required:  true,

			Internals: &AttrInternals{
				Types:       StrTypes(ENTITY_VERSION),
				UIMonospace: true,
				UILabel:     "Version ID",

				CheckFn: func(e *Entity) *XRError {
					oldID := any(e.UID)
					newID := any(e.NewObject["versionid"])

					if IsNil(newID) {
						return nil // Not trying to be updated, so skip it
					}

					if newID == "" {
						return NewXRError("invalid_attribute", e.XID,
							"name=versionid",
							"error_detail="+"can't be an empty string")
					}

					if xErr := IsValidID(newID.(string), "versionid"); xErr != nil {
						xErr.Subject = e.XID
						return xErr
					}

					if oldID != "" && !IsNil(oldID) && newID != oldID {
						info := e.GetRequestInfo()
						if info.HasIgnore("id") && "/"+info.Root == e.XID {
							// Don't check and reset to old ID (if changed)
							e.NewObject["versionid"] = oldID
						} else {
							return NewXRError("mismatched_id", e.XID,
								"singular=version",
								"invalid_id="+fmt.Sprintf("%v", newID),
								"expected_id="+fmt.Sprintf("%v", oldID))
						}
					}
					return nil
				},
				UpdateFn: func(e *Entity) *XRError {
					// Make sure the ID is always set
					if IsNil(e.NewObject["versionid"]) {
						log.Printf("tx: %s (%s) versionid is nil - "+
							"e.UID=%q e.Object=%s e.NewObject=%s e.OriginObject=%s",
							e.Tx.UUID, e.XID, e.UID, ToJSON(e.Object),
							ToJSON(e.NewObject), ToJSON(e.OriginObject))
						panic(fmt.Sprintf(`"tx: %s (%s) versionid" is nil - " +
                    "fix it!`, e.Tx.UUID, e.XID))
					}
					return nil
				},
			},
		},
		{
			Name:      "self",
			Type:      URL,
			ReadOnly:  true,
			Immutable: true,
			Required:  true,

			Internals: &AttrInternals{
				Types:        "", // Yes even ENTITY_RESOURCE
				DontStore:    true,
				XrefRequired: true,

				GetFn: func(e *Entity) any {
					base := ""
					xid := e.XID
					isAbs := false

					info := e.GetRequestInfo()
					if info != nil {
						if info.DoDocView() {
							// remove GET's base path
							xid = xid[1+len(info.Root):] // 1+ for leading /
							// if len(xid) == 0 || xid[0] != '/' {
							// Root returns "#" not "#/"
							if len(xid) > 0 && xid[0] != '/' {
								xid = "/" + xid
							}
							base = DOCVIEW_BASE
						} else {
							isAbs = true
							base = info.BaseURL
						}
					}

					if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION {
						details := info != nil && (info.ShowDetails ||
							info.ResourceUID == "" || len(info.Parts) == 5)

						if (info != nil && info.DoDocView() && !isAbs) ||
							e.GetResourceModel().GetHasDocument() == false {
							details = false
						}

						if details {
							xid += "$details"
						}
					}
					return base + xid
				},
			},
		},
		{
			Name:      "shortself",
			Type:      URL,
			ReadOnly:  true,
			Immutable: true,

			Internals: &AttrInternals{
				Types:        "",
				DontStore:    true,
				XrefRequired: true,

				GetFn: func(e *Entity) any {
					// Capabilities.ShortSelf isn't enabled, so stop
					if e.Registry.Capabilities.ShortSelfEnabled() == false {
						return nil
					}

					base := ""

					// ?doc view shouldn't include it
					info := e.GetRequestInfo()
					if info != nil {
						if info.DoDocView() {
							return nil
						}
						base = info.OriginalBaseURL
					}

					ss := base + "/r/" + e.SID

					selfAny := SpecProps["self"].Internals.GetFn(e)
					if self, ok := selfAny.(string); ok && len(self) < len(ss) {
						ss = self
					}

					return ss
				},
			},
		},
		{
			Name:      "xid",
			Type:      XID,
			ReadOnly:  true,
			Immutable: true,
			Required:  true,

			Internals: &AttrInternals{
				Types:        "",
				DontStore:    true,
				XrefRequired: true,
			},
		},
		{
			Name: "xref",
			Type: URL,

			Internals: &AttrInternals{
				Types: StrTypes(ENTITY_META),
			},
		},
		{
			Name:     "epoch",
			Type:     UINTEGER,
			ReadOnly: true,
			Required: true,

			Internals: &AttrInternals{
				Types: StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_META, ENTITY_VERSION),

				CheckFn: func(e *Entity) *XRError {
					// If we explicitly setEpoch via internal API then don't check
					if e.EpochSet {
						return nil
					}

					val := e.NewObject["epoch"]
					if IsNil(val) {
						return nil
					}

					tmp := e.Object["epoch"]
					oldEpoch := NotNilInt(&tmp)
					if oldEpoch < 0 {
						panic(fmt.Sprintf("tx: %s WHY????", e.Tx.UUID))
						oldEpoch = 0
					}

					newEpoch, err := AnyToUInt(val)
					if err != nil {
						return NewXRError("invalid_attribute", e.XID,
							"name=epoch",
							"error_detail=must be a uinteger")
					}

					if !e.GetRequestInfo().HasIgnore("epoch") && oldEpoch != 0 && newEpoch != oldEpoch {
						return NewXRError("mismatched_epoch", e.XID,
							"bad_epoch="+fmt.Sprintf("%v", val),
							"epoch="+fmt.Sprintf("%d", oldEpoch))
					}
					return nil
				},
				UpdateFn: func(e *Entity) *XRError {
					// Very special, if we're in meta and xref set then
					// erase 'epoch'. We can't do it earlier because we need
					// the CheckFn to be run to make sure any incoming value
					// was valid
					if e.Type == ENTITY_META && e.GetAsString("xref") != "" {
						e.NewObject["epoch"] = nil
						return nil
					}

					// If we already set Epoch in this Tx, just exit
					if e.EpochSet {
						// If we already set epoch this tx but there's no value
						// then grab it from Object, otherwise we'll be missing a
						// value during Save(). This can happen when we Save()
						// more than once on this Entity during the same Tx and
						// the 2nd Save() didn't have epoch as part of the incoming
						// Object
						if IsNil(e.NewObject["epoch"]) {
							e.NewObject["epoch"] = e.Object["epoch"]
						}
						return nil
					}

					// This assumes that ALL entities must have an Epoch property
					// that we want to set. At one point this wasn't true for
					// Resources but hopefully that's no longer true

					oldEpoch := e.Object["epoch"]
					epoch := NotNilInt(&oldEpoch)

					e.NewObject["epoch"] = epoch + 1
					e.EpochSet = true
					return nil
				},
			},
		},
		{
			Name: "name",
			Type: STRING,

			Internals: &AttrInternals{
				Types:   StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_VERSION),
				UILabel: "Name",
			},
		},
		{
			Name:     "isdefault",
			Type:     BOOLEAN,
			ReadOnly: true,
			Required: true,
			Default:  false,

			Internals: &AttrInternals{
				Types:     StrTypes(ENTITY_VERSION),
				DontStore: true,
				UILabel:   "Is Default",
			},
		},
		{
			Name: "description",
			Type: STRING,

			Internals: &AttrInternals{
				Types:   StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_VERSION),
				UILabel: "Description",
			},
		},
		{
			Name: "documentation",
			Type: URL,

			Internals: &AttrInternals{
				Types:   StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_VERSION),
				UILabel: "Documentation",
			},
		},
		{
			Name: "icon",
			Type: URL,

			Internals: &AttrInternals{
				Types:   StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_VERSION),
				UILabel: "Icon",
			},
		},
		{
			Name: "labels",
			Type: MAP,
			Item: &Item{
				Type: STRING,
			},

			Internals: &AttrInternals{
				Types:   StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_META, ENTITY_VERSION),
				UILabel: "Labels",
			},
		},
		{
			Name:     "createdat",
			Type:     TIMESTAMP,
			Required: true,

			Internals: &AttrInternals{
				Types:   StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_META, ENTITY_VERSION),
				UILabel: "Created",

				UpdateFn: func(e *Entity) *XRError {
					if e.Type == ENTITY_META && e.GetAsString("xref") != "" {
						e.NewObject["createdat"] = nil

						// If for some reason there is no saved createTime
						// assume this is a new meta so save 'now'
						if IsNil(e.NewObject["#createdat"]) {
							e.NewObject["#createdat"] = e.Tx.CreateTime
						}
						return nil
					}

					ca, ok := e.NewObject["createdat"]
					// If not there use the existing value, if present
					if !ok {
						ca = e.Object["createdat"]
						e.NewObject["createdat"] = ca
					}
					// Still no value, so use "now"
					if IsNil(ca) {
						ca = e.Tx.CreateTime
					}

					e.NewObject["createdat"] = ca

					return nil
				},
			},
		},
		{
			Name:     "modifiedat",
			Type:     TIMESTAMP,
			Required: true,

			Internals: &AttrInternals{
				Types:   StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_META, ENTITY_VERSION),
				UILabel: "Modified",

				UpdateFn: func(e *Entity) *XRError {
					if e.Type == ENTITY_META && e.GetAsString("xref") != "" {
						e.NewObject["modifiedat"] = nil
						return nil
					}

					ma := e.NewObject["modifiedat"]

					// If we already set modifiedat in this Tx, just exit
					if e.ModSet && !IsNil(ma) && ma != "" {
						return nil
					}

					// If there's no value, or it's the same as the existing
					// value, set to "now"
					if IsNil(ma) || (ma == e.Object["modifiedat"]) {
						ma = e.Tx.CreateTime
					}

					e.NewObject["modifiedat"] = ma
					e.ModSet = true

					return nil
				},
			},
		},
		{
			Name: "$extensions",
			Internals: &AttrInternals{
				Types: StrTypes(ENTITY_REGISTRY),
			},
		},
		{
			Name: "capabilities",
			Type: OBJECT, // This ensures the client sent a map
			Attributes: Attributes{
				"*": &Attribute{
					Name: "*",
					Type: ANY,
				},
			},

			Internals: &AttrInternals{
				Types:     StrTypes(ENTITY_REGISTRY),
				DontStore: true,

				GetFn: func(e *Entity) any {
					// Need to explicitly ask for "capabilities", ?inline=*
					// won't return it
					info := e.GetRequestInfo()
					if info != nil && info.ShouldInline(NewPPP("capabilities").DB()) {
						// Should have been caught in "info" processing
						if !info.IsAvailable("capabilities") {
							return NewXRError("not_available", "/capabilities")
						}

						return e.Registry.Capabilities
					}
					return nil
				},
				CheckFn: func(e *Entity) *XRError {
					// never part of NewObject, it'll be in reg.Capabilities.
					// Any incoming Object["capabilities"] would have been
					// processed in RequestInfo pre-processing
					return nil
				},
				UpdateFn: func(e *Entity) *XRError {
					return nil
				},
			},
		},
		{
			Name:     "model",
			Type:     OBJECT,
			ReadOnly: true,
			Attributes: Attributes{
				"*": &Attribute{
					Name: "*",
					Type: ANY,
				},
			},

			Internals: &AttrInternals{
				Types:     StrTypes(ENTITY_REGISTRY),
				DontStore: true,

				GetFn: func(e *Entity) any {
					// Need to explicitly ask for "model", ?inline=* won't
					// do it
					info := e.GetRequestInfo()
					if info != nil && info.ShouldInline(NewPPP("model").DB()) {
						// Should have been caught in "info" processing
						if !info.IsAvailable("model") {
							return NewXRError("not_available", "model")
						}
						model := info.Registry.Model
						if model == nil {
							model = &Model{}
						}
						httpModel := model // ModelToHTTPModel(model)
						return (*UserModel)(httpModel)
					}
					return nil
				},
			},
		},
		{
			Name: "modelsource",
			Type: OBJECT,
			Attributes: Attributes{
				"*": &Attribute{
					Name: "*",
					Type: ANY,
				},
			},

			Internals: &AttrInternals{
				Types: StrTypes(ENTITY_REGISTRY),

				GetFn: func(e *Entity) any {
					// Need to explicitly ask for "modelsource", ?inline=* won't
					// do it
					info := e.GetRequestInfo()
					if info != nil && info.ShouldInline(NewPPP("modelsource").DB()) {
						// Should have been caught in "info" processing
						if !info.IsAvailable("modelsource") {
							return NewXRError("not_available", "modelsource")
						}
						src, xErr := info.Registry.GetModelSource()
						if xErr != nil {
							return xErr
						}
						if len(src) == 0 {
							return struct{}{}
						}

						obj, err := ParseJSONToObject(src)
						PanicIf(err != nil, "tx: %s %s", e.Tx.UUID, err)
						return obj
					}
					return nil
				},
			},
		},
		{
			Name:     "readonly",
			Type:     BOOLEAN,
			ReadOnly: true,
			Required: true,
			Default:  false,

			Internals: &AttrInternals{
				Types:   StrTypes(ENTITY_META),
				UILabel: "Read Only",
			},
		},
		{
			Name: "compatibility",
			Type: STRING,
			Enum: []any{"backward", "backward_transitive", "forward",
				"forward_transitive", "full", "full_transitive"},
			Strict:   PtrBool(true),
			Required: false,

			Internals: &AttrInternals{
				Types:       StrTypes(ENTITY_META),
				UIMonospace: true,
				UILabel:     "Compatibility",

				UpdateFn: func(e *Entity) *XRError {
					compat, ok := e.NewObject["compatibility"]
					if ok && compat == "" {
						return NewXRError("invalid_attribute", e.XID,
							"name=compatibility",
							"error_detail=can't be an empty string")
					}
					return nil
				},
			},
		},
		{
			Name: "deprecated",
			Type: OBJECT,
			Attributes: Attributes{
				"effective": &Attribute{
					Type: TIMESTAMP,
				},
				"removal": &Attribute{
					Type: TIMESTAMP,
				},
				"alternative": &Attribute{
					Type: URL,
				},
				"documentation": &Attribute{
					Type: URL,
				},
				"*": &Attribute{
					Type: ANY,
				},
			},

			Internals: &AttrInternals{
				Types:   StrTypes(ENTITY_GROUP, ENTITY_META),
				UILabel: "Deprecated",

				UpdateFn: func(e *Entity) *XRError {
					dep, ok := e.NewObject["deprecated"]
					if !ok || IsNil(dep) {
						return nil
					}
					depMap, ok := dep.(map[string]any)
					if !ok {
						return nil
					}
					effectiveStr, _ := depMap["effective"].(string)
					removalStr, _ := depMap["removal"].(string)
					if effectiveStr != "" && removalStr != "" {
						effectiveTime, err1 := ConvertStrToTime(effectiveStr)
						removalTime, err2 := ConvertStrToTime(removalStr)
						if err1 == nil && err2 == nil &&
							removalTime.Before(effectiveTime) {
							return NewXRError("invalid_attribute", e.XID,
								"name=deprecated.removal",
								"error_detail="+
									"must not be sooner than"+
									" deprecated.effective")
						}
					}
					return nil
				},
			},
		},
		{
			Name: "constraints",
			Type: MAP,
			Item: &Item{
				Type: OBJECT,
				Attributes: Attributes{
					"enum": &Attribute{
						Type: ARRAY,
						Item: &Item{
							Type: ANY,
						},
					},
					"default": &Attribute{
						Type: ANY,
					},
					"equals": &Attribute{
						Type: STRING,
					},
				},
			},

			Internals: &AttrInternals{
				Types: StrTypes(ENTITY_GROUP),

				CheckFn: func(e *Entity) *XRError {
					g := e.Self.(*Group)
					// Note, this will not actually do the constaint checks,
					// this just verifies the constraint defintions are valid.
					// The checks over the data will happen right before we
					// commit the tx - in g.Validate()
					constraints := map[string]*Constraint(nil)
					val := e.Get("constraints")
					if !IsNil(val) {
						err := Unmarshal([]byte(ToJSON(val)), &constraints)
						if err != nil {
							return NewXRError("bad_request", g.XID,
								"error_detail="+err.Error())
						}
						g.groupConstraints = constraints
						return g.GroupModel.ValidateConstraints(g, constraints)
					}
					return nil
				},
			},
		},
		{
			Name:     "ancestorid",
			Type:     STRING,
			Required: true,

			Internals: &AttrInternals{
				Types:       StrTypes(ENTITY_VERSION),
				UIMonospace: true,
				UILabel:     "Ancestor Version ID",

				UpdateFn: func(e *Entity) *XRError {
					_, ok := e.NewObject["ancestorid"]
					PanicIf(!ok, "tx: %s Missing versionid", e.Tx.UUID)
					if !ok {
						_, ok := e.NewObject["versionid"]
						PanicIf(!ok, "tx: %s Missing versionid", e.Tx.UUID)
						// Just assign a placeholder to get past validation.
						// CheckAncestors() should fix this before we commit
						// the tx
						e.NewObject["ancestorid"] = ANCESTORID_TBD
					}
					return nil
				},
			},
		},
		{
			Name: "contenttype",
			Type: STRING,

			Internals: &AttrInternals{
				Types:       StrTypes(ENTITY_VERSION),
				HttpHeader:  "Content-Type",
				UIMonospace: true,
				UILabel:     "Content Type",
			},
		},
		{
			Name: "format",
			Type: STRING,

			Internals: &AttrInternals{
				Types:       StrTypes(ENTITY_VERSION),
				UIMonospace: true,
				UILabel:     "Format",

				UpdateFn: func(e *Entity) *XRError {
					format, ok := e.NewObject["format"]
					if ok && format == "" {
						return NewXRError("invalid_attribute", e.XID,
							"name=format",
							"error_detail=can't be an empty string")
					}
					return nil
				},
			},
		},
		{
			Name:     "formatvalidated",
			Type:     BOOLEAN,
			ReadOnly: true,

			Internals: &AttrInternals{
				Types:     StrTypes(ENTITY_VERSION),
				NoDocView: true,
				UILabel:   "Format Validated",
			},
		},
		{
			Name:     "formatvalidatedreason",
			Type:     STRING,
			ReadOnly: true,

			Internals: &AttrInternals{
				Types:       StrTypes(ENTITY_VERSION),
				NoDocView:   true,
				UIMonospace: true,
				UILabel:     "Format Validated Reason",
			},
		},
		{
			Name:     "compatibilityvalidated",
			Type:     BOOLEAN,
			ReadOnly: true,

			Internals: &AttrInternals{
				Types:     StrTypes(ENTITY_VERSION),
				NoDocView: true,
				UILabel:   "Compatibility Validated",
			},
		},
		{
			Name:     "compatibilityvalidatedreason",
			Type:     STRING,
			ReadOnly: true,

			Internals: &AttrInternals{
				Types:       StrTypes(ENTITY_VERSION),
				NoDocView:   true,
				UIMonospace: true,
				UILabel:     "Compatibility Validated Reason",
			},
		},
		{
			Name: "$extensions",
			Internals: &AttrInternals{
				Types: StrTypes(ENTITY_GROUP, ENTITY_RESOURCE, ENTITY_META, ENTITY_VERSION),
			},
		},
		{
			Name: "$space",
			Internals: &AttrInternals{
				Types: "",
			},
		},
		// For the $RESOURCE ones, make sure to use attr.Clone("newname")
		// when the $RESOURCE is substituded with the Resource's singular
		// name. Otherwise you'll be updating this shared entry.
		{
			Name: "$RESOURCEurl",
			Type: URL,
			Internals: &AttrInternals{
				Types: StrTypes(ENTITY_VERSION),

				CheckFn: RESOURCECheckFn,
				UpdateFn: func(e *Entity) *XRError {
					singular := e.GetResourceSingular()
					v, ok := e.NewObject[singular+"url"]
					if ok && !IsNil(v) {
						e.NewObject[singular] = nil
						e.NewObject[singular+"proxyurl"] = nil
						e.NewObject["#contentid"] = nil
					}
					return nil
				},
			},
		},
		{
			Name: "$RESOURCEproxyurl",
			Type: URL,
			Internals: &AttrInternals{
				Types: StrTypes(ENTITY_VERSION),

				CheckFn: RESOURCECheckFn,
				UpdateFn: func(e *Entity) *XRError {
					singular := e.GetResourceSingular()
					v, ok := e.NewObject[singular+"proxyurl"]
					if ok && !IsNil(v) {
						e.NewObject[singular] = nil
						e.NewObject[singular+"url"] = nil
						e.NewObject["#contentid"] = nil
					}
					return nil
				},
			},
		},
		{
			Name: "$RESOURCE",
			Type: ANY,

			Internals: &AttrInternals{
				Types:           StrTypes(ENTITY_VERSION),
				AlwaysSerialize: true, // Will always be missing, so need this

				CheckFn: RESOURCECheckFn,
				UpdateFn: func(e *Entity) *XRError {
					singular := e.GetResourceSingular()
					v, ok := e.NewObject[singular]
					if ok {
						if !IsNil(v) {
							e.NewObject[singular+"url"] = nil
							e.NewObject[singular+"proxyurl"] = nil
							e.NewObject["#contentid"] = e.SID
						} else {
							e.NewObject["#contentid"] = nil
						}
					}
					return nil
				},
			},
		},
		{
			Name: "$RESOURCEbase64",
			Type: STRING,
			Internals: &AttrInternals{
				Types: StrTypes(ENTITY_VERSION),

				CheckFn: RESOURCECheckFn,
				UpdateFn: func(e *Entity) *XRError {
					singular := e.GetResourceSingular()
					v, ok := e.NewObject[singular]
					if ok {
						if !IsNil(v) {
							e.NewObject[singular+"url"] = nil
							e.NewObject[singular+"proxyurl"] = nil
							e.NewObject["#contentid"] = e.SID
						} else {
							e.NewObject["#contentid"] = nil
						}
					}
					return nil
				},
			},
		},
		{
			Name: "$space",
			Internals: &AttrInternals{
				Types: "",
			},
		},
		{
			Name: "meta",
			Type: OBJECT,
			Attributes: Attributes{
				"*": &Attribute{
					Name: "*",
					Type: ANY,
				},
			},
			Internals: &AttrInternals{
				Types: StrTypes(ENTITY_RESOURCE),
			},
		},
		{
			Name:      "metaurl",
			Type:      URL,
			ReadOnly:  true,
			Immutable: true,
			Required:  true,

			Internals: &AttrInternals{
				Types:     StrTypes(ENTITY_RESOURCE),
				DontStore: true,

				GetFn: func(e *Entity) any {
					base := ""
					xid := e.XID

					info := e.GetRequestInfo()
					if info != nil {
						inlineMeta := info.ShouldInline(e.Abstract +
							string(DB_IN) + "meta")

						if !info.DoDocView() || !inlineMeta {
							base = info.BaseURL
						} else {
							base = DOCVIEW_BASE

							// remove GET's base path
							xid = xid[1+len(info.Root):]
							// Not sure this is ever true
							if len(xid) > 0 && xid[0] != '/' {
								xid = "/" + xid
							}
						}
					}

					return base + xid + "/meta"
				},
			},
		},
		{
			Name: "$space",
			Internals: &AttrInternals{
				Types: "",
			},
		},
		{
			Name:     "defaultversionid",
			Type:     STRING,
			Required: true,

			Internals: &AttrInternals{
				Types:       StrTypes(ENTITY_META),
				UIMonospace: true,
				UILabel:     "Default Version ID",

				UpdateFn: func(e *Entity) *XRError {
					// Make sure it has a value, if not copy from existing
					xRef := e.NewObject["xref"]
					PanicIf(xRef == "", "tx: %s xref is ''", e.Tx.UUID)

					/* Really should check this
					   newVal := e.NewObject["defaultversionid"]
					   PanicIf(IsNil(xRef) && IsNil(newVal), "defverid is nil")
					*/

					/*
					   if IsNil(xRef) && IsNil(newVal) {
					       oldVal := e.Object["defaultversionid"]
					       e.NewObject["defaultversionid"] = oldVal
					   }
					*/

					// non-xRef resources MUST have a valid defaultversionid
					if IsNil(xRef) {
						meta, ok := e.Self.(*Meta)
						PanicIf(!ok, "tx: %s e isn't a meta: %#v", e.Tx.UUID, e)

						val := meta.GetAsString("defaultversionid")
						ver, xErr := meta.Resource.FindVersion(val)
						if xErr != nil {
							return xErr
						}
						if IsNil(ver) {
							// ShowStack()
							// meta.Resource.DumpOrderedVersions()
							return NewXRError("unknown_id", meta.XID,
								"singular=version",
								"id="+val)
						}
					}

					return nil
				},
			},
		},
		{
			Name:     "defaultversionurl",
			Type:     URL,
			ReadOnly: true,
			Required: true,

			Internals: &AttrInternals{
				Types:     StrTypes(ENTITY_META),
				DontStore: true,
				UILabel:   "Default Version URL",

				GetFn: func(e *Entity) any {
					val := e.Object["defaultversionid"]
					if IsNil(val) {
						return nil
					}
					valStr := val.(string)

					// replace "meta" with "versions/VID"
					xid := e.XID[:len(e.XID)-4] + "versions/" + valStr
					result := ""
					isAbsURL := false
					suffix := ""

					info := e.GetRequestInfo()
					if info != nil {
						// s/meta/versions/
						abs := e.Abstract[:len(e.Abstract)-4] + "versions"
						inlineVers := info.ShouldInline(abs)
						seenDefVid := info.Extras["seenDefaultVid"]

						if len(info.Parts) == 5 { // pointing directly to /meta
							isAbsURL = true
						}

						if !info.DoDocView() {
							isAbsURL = true
						}

						if !inlineVers {
							isAbsURL = true
						}

						if len(info.Filters) > 0 && seenDefVid != valStr {
							isAbsURL = true
						}

						if isAbsURL {
							result = info.BaseURL
							if e.GetResourceModel().GetHasDocument() == true {
								suffix = "$details"
							}
						} else {
							if info.DoDocView() {
								result = DOCVIEW_BASE

								// remove GET's base path
								xid = xid[1+len(info.Root):]
								if len(xid) == 0 || xid[0] != '/' {
									xid = "/" + xid
								}
							}
						}
					}

					// remove "/meta" so we can add "/versions/vID"
					result += xid + suffix

					return result
				},
			},
		},
		{
			Name:     "defaultversionsticky",
			Type:     BOOLEAN,
			Required: true,
			Default:  false,

			Internals: &AttrInternals{
				Types:   StrTypes(ENTITY_META),
				UILabel: "Default Version Sticky",

				CheckFn: func(e *Entity) *XRError {
					if e.GetResourceModel().GetMaxVersions() == 1 {
						if e.NewObject["defaultversionsticky"] == true {
							return NewXRError("setdefaultversionsticky_false", e.XID)
						}
					}
					return nil
				},
			},
		},
		{
			Name: "$space",
			Internals: &AttrInternals{
				Types: "",
			},
		},
		{
			Name: "$COLLECTIONS", // Implicitly creates the url and count ones
			Type: MAP,
			Item: &Item{
				Type: OBJECT,
				Attributes: Attributes{
					"*": {
						Type: ANY,
					},
				},
			},
			Internals: &AttrInternals{
				Types:     StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_RESOURCE),
				DontStore: true,
			},
		},
	}
}

var SpecProps = map[string]*Attribute{}
var CollectionsURLAttr *Attribute
var CollectionsCountAttr *Attribute
var CollectionsAttr *Attribute
var SpecialAttrHeaders = map[string]*Attribute{} // attrs using HTTP names

func init() {
	// Can't init the funcs at the same time as we define OrderedSpecProps,
	// due to a circular dependency. So we populate the array after its been
	// defined.
	InitOrderedSpecProps()

	// Load map via lower-case version of prop name
	for _, sp := range OrderedSpecProps {
		SpecProps[sp.Name] = sp
	}

	// Load-up the attributes that have custom http header names
	for _, attr := range OrderedSpecProps {
		if attr.Internals != nil && attr.Internals.HttpHeader != "" {
			SpecialAttrHeaders[strings.ToLower(attr.Internals.HttpHeader)] =
				attr
		}
	}

	CollectionsAttr = SpecProps["$COLLECTIONS"]

	CollectionsURLAttr = &Attribute{
		Name:      "$COLLECTIONSurl",
		Type:      URL,
		ReadOnly:  true,
		Immutable: true,
		Required:  true,
		Internals: &AttrInternals{
			Types:     StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_RESOURCE),
			DontStore: true,
		},
	}

	CollectionsCountAttr = &Attribute{
		Name:     "$COLLECTIONScount",
		Type:     UINTEGER,
		ReadOnly: true,
		Required: true,
		Internals: &AttrInternals{
			Types:     StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_RESOURCE),
			DontStore: true,
		},
	}
}

func StrTypes(types ...int) string {
	res := strings.Builder{}
	for _, eType := range types {
		res.WriteByte('0' + byte(eType))
	}
	return res.String()
}

func (e *Entity) GetRequestInfo() *RequestInfo {
	if e.Tx == nil {
		return nil
	}

	return e.Tx.RequestInfo
}

func (e *Entity) GetModels() (*GroupModel, *ResourceModel) {
	if e.GroupModel != nil {
		return e.GroupModel, e.ResourceModel
	}

	if e.Type == ENTITY_REGISTRY || e.Type == ENTITY_MODEL {
		return nil, nil
	}

	e.GroupModel, e.ResourceModel = AbstractToModels(e.Registry, e.Abstract)
	return e.GroupModel, e.ResourceModel
}

// SpecAttrInfo carries the name and entity-type membership for one spec-defined attribute.
// Types is the StrTypes string: "0"=Registry, "1"=Group, "2"=Resource, "3"=Meta, "4"=Version;
// empty string means the attribute applies to all entity types.
type SpecAttrInfo struct {
	Name        string
	Types       string
	UIMonospace bool
	UILabel     string
}

// GetOrderedSpecAttrTypes returns name+types for every entry in
// OrderedSpecProps, in declaration order. Used by cmds/genspecattrs to
// generate registry/http/ui/specattrs.js.
//
// Note: nested attributes of spec-defined complex (object) attrs — e.g.
// "deprecated"'s effective/removal/alternative/documentation — are NOT
// enumerated here. Their types are already fully described in the runtime
// /model response (which the UI caches per-server), so the UI resolves
// monospace formatting for nested fields by walking that cached model at
// render time (see getExplicitAttrTypeAtPath() in registry/http/ui/app.js)
// rather than via codegen — this keeps nested-field handling generic for
// any current or future spec-defined (or model-defined extension) complex
// attribute, without hardcoding attribute names here.
func GetOrderedSpecAttrTypes() []SpecAttrInfo {
	result := make([]SpecAttrInfo, 0, len(OrderedSpecProps))
	for _, attr := range OrderedSpecProps {
		types := ""
		mono := false
		label := ""
		if attr.Internals != nil {
			types = attr.Internals.Types
			mono = attr.Internals.UIMonospace
			label = attr.Internals.UILabel
		}
		result = append(result, SpecAttrInfo{Name: attr.Name, Types: types, UIMonospace: mono, UILabel: label})
	}
	return result
}

// ---- Canonical-order JSON pretty-printer (TODO.md "json-pretty-printer") --
//
// CanonicalPrettyPrintJSON re-orders + reformats arbitrary xRegistry JSON
// (a single entity, or a full nested Registry doc, possibly captured from
// another server) into the spec's canonical attribute order (core/
// spec.md "Design: JSON Serialization"), using this package's own
// OrderedSpecProps (via GetOrderedSpecAttrTypes(), same source
// cmds/genspecattrs uses for the SPA's mirror table) as the ordering
// source of truth. See common.CanonicalReorderJSON() for the generic
// (level-order-agnostic) walk/reorder/stringify engine this just feeds.
//
// Deliberate simplifications (this is a utility function only — nothing
// calls it yet; see plan.md/TODO.md for the full design discussion):
//   - Every nested entity is found structurally, via its own "xid" field
//     — no server/model round-trip is needed or attempted.
//   - Extension (non-spec) attributes are alphabetized at each level's
//     "$extensions" position.
//   - The Registry's own "id" attribute has a fixed, model-independent
//     wire name ("registryid"), hardcoded directly into the ordering
//     table. Group/Resource/Meta/Version's own "id" attribute wire name
//     (e.g. "messageid") instead depends on the model's singular name —
//     resolved by VALUE (not by guessing): the entity's own xid already
//     tells us the exact ID string it must hold, so whichever of the
//     entity's own keys holds that value (and ends in "id") is treated
//     as the resolved "id" attribute (see common.resolveIDKeyByValue()).
//   - "$RESOURCE"-family attrs on a Version (<singular>url/proxyurl/
//     <singular>/<singular>base64) are resolved directly from that same
//     singular name once found — no further guessing needed.
//   - "$COLLECTIONS": a Resource's own collection is always the fixed,
//     model-independent "versions"/"versionscount"/"versionsurl" triple.
//     Registry's (group-type) and Group's (resource-type) collections
//     have no fixed name and there can be any number of them, so those
//     ARE detected heuristically instead: any "<name>count" key with a
//     matching "<name>url" sibling is treated as one collection (see
//     common.detectHeuristicCollections()).
//   - If none of the above heuristics find a match (e.g. minimal/foreign
//     JSON omitting the relevant keys), the corresponding token is
//     simply skipped and the real data — if present under some other
//     name — falls through to being treated as an ordinary (alphabetized)
//     extension instead.
func CanonicalPrettyPrintJSON(data []byte) ([]byte, error) {
	return CanonicalReorderJSON(data, canonicalLevelOrderFor)
}

// CanonicalPrettyReorderTree is the level-order-aware entry point for
// CanonicalReorderTree() (this file dot-imports 'common', so it's
// referenced unqualified below — can't reuse its exact name here, that
// would be a duplicate declaration): parses data and returns it as a
// live *OrderedMap tree with every nested entity's own attributes
// ALREADY reordered into canonical spec order (including blank-line
// sentinels — see canonicalBlankKey) — but NOT yet stringified.
//
// This exists for callers that need to delete keys from an
// already-canonically-ordered tree BEFORE turning it into bytes — e.g.
// cmds/xr/get.go's minimize() (the "xr get -m" flag), which used to
// delete keys from the raw JSON FIRST and only then try to reorder what
// was left. That approach requires guessing at classifying/ordering
// information (xid, ids, model-driven attribute names) that minimize()
// itself had just deleted. Reordering first — while everything is still
// exactly classifiable via the real xid/model — and minimizing
// afterward on the ordered tree (via *OrderedMap.Delete(), which
// preserves the order of what's left) sidesteps that problem entirely:
// no heuristics are needed, and the blank-line spacing collapses
// correctly around whatever got deleted (see stringifyOrderedMap()'s
// doc comment).
func CanonicalPrettyReorderTree(data []byte) (interface{}, error) {
	return CanonicalReorderTree(data, canonicalLevelOrderFor)
}

// CanonicalPrettyReorderTreeOrRaw is CanonicalPrettyReorderTree(), unless
// rawJSON is true (see the "rawjson" .xr config option — cmds/xr/xr.go's
// GetRawJSON()), in which case it just parses data into the same
// OrderedMap/[]any tree shape (via ParseJSONToObject()) WITHOUT
// reordering into canonical attribute order or inserting any blank-line
// separators — i.e. exactly the attribute order the server itself
// returned. This lets a single caller-side pipeline (parse -> optional
// minimize/noDiff -> stringify, see cmds/xr/get.go/set.go/create.go/
// download.go) serve both the default (canonically reordered) output
// and the "rawjson" (server's own raw attribute order) output, without
// duplicating the whole minimize/noDiff logic per mode. "--min"/
// "--nodiff" (see cmds/xr/download.go) still fully apply either way -
// only the reordering/blank-line step is skipped when rawJSON is true.
func CanonicalPrettyReorderTreeOrRaw(data []byte, rawJSON bool) (interface{}, error) {
	if rawJSON {
		return ParseJSONToObject(data)
	}
	return CanonicalReorderTree(data, canonicalLevelOrderFor)
}

var canonicalLevelOrders map[int][]string

// canonicalLevelOrderFor lazily builds (once) and returns the ordered
// token list (spec attr names, in declaration order, plus surviving
// "$space"/"$extensions" markers) for the given ENTITY_* level — mirrors
// cmds/genspecattrs/main.go's identical per-level filter/collapse logic
// (both consume the exact same GetOrderedSpecAttrTypes() declaration
// order, so the Go and JS canonical printers can never silently diverge).
func canonicalLevelOrderFor(entityType int) []string {
	if canonicalLevelOrders == nil {
		canonicalLevelOrders = buildCanonicalLevelOrders()
	}
	return canonicalLevelOrders[entityType]
}

func buildCanonicalLevelOrders() map[int][]string {
	levels := []int{ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_RESOURCE, ENTITY_META, ENTITY_VERSION}
	attrs := GetOrderedSpecAttrTypes()
	result := make(map[int][]string, len(levels))

	for _, lvl := range levels {
		digit := byte('0' + byte(lvl))
		// A Resource's own HTTP GET response isn't just its own
		// RESOURCE-typed attrs: the server mirrors its default
		// Version's own attrs (isdefault/createdat/modifiedat/
		// ancestorid/contenttype/the "$RESOURCE*" content family/etc.)
		// directly onto the Resource's own row (see
		// registry/resource.go's SaveDefaultVersionCascade()/
		// IsDefaultVerCopy mechanism), so the Resource-level canonical
		// order must ALSO include ENTITY_VERSION-typed attrs, in the
		// same master declaration order — otherwise those mirrored
		// attrs aren't recognized as spec attrs and incorrectly fall
		// through to being alphabetized as plain extensions.
		extraDigit := byte(0)
		if lvl == ENTITY_RESOURCE {
			extraDigit = '0' + byte(ENTITY_VERSION)
		}
		seen := map[string]bool{}
		tokens := make([]string, 0, len(attrs))
		for _, a := range attrs {
			applies := a.Types == "" || strings.IndexByte(a.Types, digit) >= 0 ||
				(extraDigit != 0 && strings.IndexByte(a.Types, extraDigit) >= 0)
			if !applies {
				continue
			}
			if a.Name == "$space" || a.Name == "$extensions" {
				tokens = append(tokens, a.Name)
				continue
			}
			if seen[a.Name] {
				continue
			}
			seen[a.Name] = true
			// The Registry's generic "id" attribute always has the fixed,
			// model-independent wire name "registryid" — safe to
			// hardcode (see CanonicalPrettyPrintJSON's doc comment).
			// Group/Resource/Meta's own "<singular>id" wire name DOES
			// depend on the model and is intentionally NOT substituted
			// here (falls through to being treated as an extension).
			if a.Name == "id" && lvl == ENTITY_REGISTRY {
				tokens = append(tokens, "registryid")
				continue
			}
			tokens = append(tokens, a.Name)
		}

		// Collapse consecutive "$space" runs, drop leading/trailing.
		collapsed := make([]string, 0, len(tokens))
		for _, t := range tokens {
			if t == "$space" && (len(collapsed) == 0 || collapsed[len(collapsed)-1] == "$space") {
				continue
			}
			collapsed = append(collapsed, t)
		}
		for len(collapsed) > 0 && collapsed[len(collapsed)-1] == "$space" {
			collapsed = collapsed[:len(collapsed)-1]
		}
		result[lvl] = collapsed
	}
	return result
}

func (e *Entity) ValidateAndSave(force bool) *XRError {
	defer log.Trace("tx: %s %s", e.Tx.UUID, e.XID)()

	// Force will do a validate even if it doesn't look like anything changed.
	// BUT if after validate() nothing still hasn't changed then it doesn't
	// call save()

	// If nothing changed, then exit
	if !force && e.NewObject == nil {
		return nil
	}

	// Make sure we have a tx since Validate assumes it
	if xErr := e.Tx.EnsureTx(); xErr != nil {
		return xErr
	}

	verb := log.IsFuncVerbose()
	if verb {
		log.Printf("tx: %s "+
			"Pre validate %s/%s\ne.Object:\n%s\n\ne.NewObject:\n%s",
			e.Tx.UUID,
			e.Abstract, e.UID, ToJSON(e.Object), ToJSON(e.NewObject))
	}

	if xErr := e.Validate(); xErr != nil {
		return xErr
	}

	if verb {
		log.Printf("tx: %s Post validate(%s): %s", e.Tx.UUID,
			e.XID, ToJSON(e.NewObject))
	}

	if e.NewObject == nil {
		return nil
	}

	return e.Save()
}

// Doesn't fully validate in the sense that it'll assume read-only fields
// are not worth checking since the server generated them.
// This is mainly used for validating input from a client.
// NOTE!!! This isn't a read-only operation. Normally it would be, but to
// avoid traversing the entity more than once, we will tweak things if needed.
// For example, if a missing attribute has a Default value then we'll add it.
func (e *Entity) Validate() *XRError {
	defer log.Trace("tx: %s %s", e.Tx.UUID, e.XID)()
	// Don't touch what was passed in
	attrs := e.GetAttributes(e.NewObject)
	if log.IsFuncVerbose() {
		log.Printf("tx: %s Attrs:\n%s", e.Tx.UUID, ToJSON(attrs))
	}

	// Skip xref's versions since its owning resource should have done it
	/* should not need this, but save as comment just in case
	   if e.Type == ENTITY_VERSION {
	       v := e.Self.(*Version)
	       r := v.Resource
	       if r.IsXref() {
	           return nil
	       }
	   }
	*/

	if e.Type == ENTITY_RESOURCE {
		// Skip Resources // TODO DUG - would prefer to not do this
		return nil
		// If we ever support extensions in resourceattributes
		/*
		   RemoveVersionAttributes(e.ResourceModel, e.NewObject)

		   // Not really correct yet.
		   // should just use resourceattributes + ifvaluesattrs
		   for _, k := range Keys(attrs) {
		       a := attrs[k]
		       if a.InType(ENTITY_VERSION) && !a.InType(ENTITY_RESOURCE) {
		           delete(attrs, k)
		       }
		   }
		*/
	}

	if log.IsFuncVerbose() {
		log.Printf("tx: %s ========", e.Tx.UUID)
		log.Printf("tx: %s NewObject:\n%s", e.Tx.UUID, ToJSON(e.NewObject))
		log.Printf("tx: %s Attrs: %v", e.Tx.UUID, SortedKeys(attrs))
	}

	// If nothing changed then use the original data for validation
	if e.NewObject == nil {
		return e.ValidateObject(e.Object, "strict", attrs, NewPP())
	} else {
		return e.ValidateObject(e.NewObject, "strict", attrs, NewPP())
	}
}

// This should be called after all type-specific calculated properties have
// been removed - such as collections
func (e *Entity) ValidateObject(val any, namecharset string, origAttrs Attributes, path *PropPath) *XRError {

	defer log.Trace("tx: %s %s", e.Tx.UUID, path)()

	if log.IsFuncVerbose() {
		log.Printf("tx: %s Check Obj:\n%s", e.Tx.UUID, ToJSON(val))
		log.Printf("tx: %s OrigAttrs:\n%s", e.Tx.UUID,
			ToJSON(SortedKeys(origAttrs)))
	}

	newObj, ok := val.(map[string]any)
	if !ok {
		return NewXRError("invalid_attribute", e.XID,
			"name="+path.UI(),
			"error_detail="+"must be a map[string] or object")
	}

	// Convert origAttrs to a slice of *Attribute where "*" is first, if there
	attrs := make([]*Attribute, len(origAttrs))
	allAttrNames := map[string]bool{}
	count := 1
	for _, attr := range origAttrs {
		allAttrNames[attr.Name] = true
		if attr.Name == "*" {
			attrs[0] = attr // "*" must appear first in the slice
		} else if count == len(attrs) {
			attrs[0] = attr // at last one and no "*" so use [0]
		} else {
			attrs[count] = attr
			count++
		}
	}

	// Don't touch what was passed in
	objKeys := maps.Clone(newObj)

	attr := (*Attribute)(nil)
	key := ""
	for len(attrs) > 0 {
		l := len(attrs)
		attr = attrs[l-1] // grab last one & remove it
		attrs = attrs[:l-1]

		// Keys are all of the attribute names in newObj we need to check.
		// Normally there's just one (attr.Name) but if attr.Name is "*"
		// then we'll have a list of all remaining attribute names in newObj to
		// check, hence it's a slice not a single string
		keys := []string{}
		if attr.Name != "*" {
			keys = []string{attr.Name}
		} else {
			keys = SortedKeys(objKeys) // no need to be sorted, just grab keys

			// However, look for extensions in Versions that might overlap
			// with Resource attribute, like "meta"
			if path.Len() == 0 && e.Type == ENTITY_VERSION {
				special := map[string]bool{
					"versions":    true,
					"versionsurl": true,
				}
				for _, key := range keys {
					prop := SpecProps[key]
					if special[key] ||
						(prop != nil && prop.InOnlyType(ENTITY_RESOURCE)) {

						return NewXRError("invalid_attribute", e.XID,
							"name="+path.P(key).UI(),
							"error_detail=Versions can't define an "+
								"extension called: "+key)
					}
				}
			}
		}

		// For each attribute (key) in newObj, check its type
		for _, key = range keys {
			if len(key) > 0 && key[0] == '#' && path.Len() == 0 {
				// Skip system attributes, but only at top level
				continue
			}

			val, keyPresent := newObj[key]

			// A Default value is defined but there's no value, so set it
			// and then let normal processing continue
			daDefault := e.CalcAttrDefault(attr, path)

			if !IsNil(daDefault) && (!keyPresent || IsNil(val)) {
				// When meta.xref is set we skip any attributes with default
				// values. However, if this ever changes where some do need
				// to be set, add a flag to the OrderedSpecProps stuff
				// so we don't need to special case each one
				if e.Type != ENTITY_META || e.GetAsString("xref") == "" {
					val = daDefault
					newObj[key] = val
					keyPresent = true
				}
			}

			/* Not sure what this was for :-)  save for now
			   if path.Len() > 0 {
			       if xErr := IsValidAttributeName(path.Bottom(), e.XID, path.UI()); xErr != nil {
			           return xErr
			       }
			   }
			*/

			// Based on the attribute's type check the incoming 'val'.
			// This will check for adherence to the model (eg type),
			// the next section (CheckFn) will allow for more detailed
			// checking, like for valid values
			if !IsNil(val) {
				xErr, haveReplacement, newValue := e.ValidateAttribute(val,
					attr, path.P(key))
				if xErr != nil {
					return xErr
				}
				if haveReplacement {
					val = newValue
					newObj[key] = val
					keyPresent = true
				}
			}

			// GetAttributes already added IfValues for Registry attributes
			if path.Len() >= 1 && len(attr.IfValues) > 0 {
				valStr := strings.ToLower(fmt.Sprintf("%v", val))
				for ifValStr, ifValueData := range attr.IfValues {
					if valStr != strings.ToLower(ifValStr) {
						continue
					}

					for _, newAttr := range ifValueData.SiblingAttributes {
						if _, ok := allAttrNames[newAttr.Name]; ok {
							return NewXRError("invalid_attribute", e.XID,
								"name="+path.P(key).UI(),
								"error_detail="+
									fmt.Sprintf(`has an "ifvalues"`+
										`(%s) that defines a conflictng `+
										`siblingattribute: %s`,
										valStr, newAttr.Name))
						}
						// add new attr to the list so we can check its ifValues
						if newAttr.Name == "*" {
							attrs = append([]*Attribute{newAttr}, attrs...)
						} else {
							attrs = append(attrs, newAttr)
						}
						allAttrNames[newAttr.Name] = true
					}
				}
			}

			// Call the attr's CheckFn if there to make sure any
			// incoming value is ok
			if attr.Internals != nil && attr.Internals.CheckFn != nil {
				if xErr := attr.Internals.CheckFn(e); xErr != nil {
					return xErr
				}
			}

			// Skip/remove 'DontStore' attrs
			if attr.Internals != nil && attr.Internals.DontStore {
				// TODO find a way to allow an admin to set the
				// meta.ReadOnly flag itself
				delete(objKeys, key) // Remove from to-process list
				delete(newObj, key)
				continue
			}

			// If this attr has a func to update its value, call it
			if attr.Internals != nil && attr.Internals.UpdateFn != nil {
				if e.NewObject != nil {
					if xErr := attr.Internals.UpdateFn(e); xErr != nil {
						return xErr
					}
				}

				// grab value in case it changed
				val, keyPresent = newObj[key]
			}

			// Required but not present - note that nil means will be deleted
			if attr.Required && (!keyPresent || IsNil(val)) {
				flagit := true // Assume we'll err

				// Most "meta" attribute aren't actually required when xref
				// is set, so only flag the ones w/o 'XrefRequired=true'
				if e.Type == ENTITY_META && e.GetAsString("xref") != "" &&
					!attr.Internals.XrefRequired {
					flagit = false
				}

				// Version.RESOURCEid MUST be missing, so don't flag it
				// All other entities need that attribute though
				if path.Len() == 0 && e.Type == ENTITY_VERSION &&
					key == e.GetResourceSingular()+"id" {
					flagit = false
				}

				if flagit {
					return NewXRError("required_attribute_missing", e.XID,
						"list="+path.P(key).UI())
				}
			}

			// And finally check to make sure it's a valid attribute name,
			// but only if it's actually present in the object.
			if keyPresent {
				lowerNCS := strings.ToLower(namecharset)
				if lowerNCS == "extended" {
					if xErr := IsValidMapKey(key, e.XID, path.UI()); xErr != nil {
						return xErr
					}
				} else if lowerNCS == "" || lowerNCS == "strict" {
					if xErr := IsValidAttributeName(key, e.XID, path.UI()); xErr != nil {
						return xErr
					}
				} else {
					return NewXRError("bad_request", e.XID,
						"error_detail="+
							fmt.Sprintf("Unknown \"namecharset\" value: %s",
								namecharset))
				}
			}

			// Everything is good, so remove it from to-process list
			delete(objKeys, key)
		}
	}

	// See if we have any extra keys and if so, generate an error
	del := []string{}
	for k, _ := range objKeys {
		if k[0] == '#' {
			del = append(del, k)
		}
	}
	for _, k := range del {
		delete(objKeys, k)
	}
	if len(objKeys) != 0 {
		where := path.UI()
		if where != "" {
			where += "."
		}

		xErr := NewXRError("unknown_attribute", e.XID,
			"name="+where+SortedKeys(objKeys)[0])
		if len(objKeys) > 1 {
			xErr.SetDetailf("Full list: %s.",
				strings.Join(SortedKeys(objKeys), ","))
		}
		return xErr

		/*
		   list := ""
		   for i, k := range SortedKeys(objKeys) {
		       if i > 0 {
		           list += ","
		       }
		       list += where + k
		   }
		   return NewXRError("unknown_attribute", e.XID,
		       "name="+list)
		*/
	}

	return nil
}

// Return: error, haveReplaceValue, newValue
func (e *Entity) ValidateAttribute(val any, attr *Attribute, path *PropPath) (*XRError, bool, any) {
	if log.IsFuncVerbose() {
		log.Printf("tx: %s val: %v", e.Tx.UUID, ToJSON(val))
		log.Printf("tx: %s attr: %v", e.Tx.UUID, ToJSON(attr))
	}

	if attr.Type == ANY {
		// All good - let it thru
		return nil, false, nil
	} else if IsScalar(attr.Type) {
		return e.ValidateScalar(val, attr, path)
	} else if attr.Type == MAP {
		return e.ValidateMap(attr, val, path), false, nil
	} else if attr.Type == ARRAY {
		return e.ValidateArray(attr, val, path), false, nil
	} else if attr.Type == OBJECT {
		/*
		   attrs := e.GetBaseAttributes()
		   if useNew {
		       attrs.AddIfValuesAttributes(e.NewObject)
		   } else {
		       attrs.AddIfValuesAttributes(e.Object)
		   }
		*/

		return e.ValidateObject(val, attr.NameCharSet, attr.Attributes, path),
			false, nil
	}

	ShowStack()
	panic(fmt.Sprintf("tx: %s Unknown type(%s): %s", path.UI(),
		e.Tx.UUID, attr.Type))
}

func (e *Entity) ValidateMap(mapAttr *Attribute, val any, path *PropPath) *XRError {
	if log.IsFuncVerbose() {
		log.Printf("tx: %s item: %v", e.Tx.UUID, ToJSON(mapAttr.Item))
		log.Printf("tx: %s val: %v", e.Tx.UUID, ToJSON(val))
	}

	if IsNil(val) {
		return nil
	}

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Map {
		return NewXRError("invalid_attribute", e.XID,
			"name="+path.UI(),
			"error_detail=must be a map")
	}

	// All values in the map must be of the same type
	attr := &Attribute{
		Type: mapAttr.Item.Type,

		Enum:        mapAttr.Item.Enum,
		Strict:      mapAttr.Item.Strict,
		Target:      mapAttr.Item.Target,
		NameCharSet: mapAttr.Item.NameCharSet,

		Attributes: mapAttr.Item.Attributes,
		Item:       mapAttr.Item.Item,
	}

	for _, k := range valValue.MapKeys() {
		keyName, ok := k.Interface().(string)
		if !ok {
			return NewXRError("invalid_attribute",
				"name="+path.RemoveLast().UI(),
				"error_detail="+
					fmt.Sprintf("map key (%s) needs to be a string, "+
						"not %s", path.Last().Text, k.Kind().String()))
		}

		if path.Len() > 0 {
			if xErr := IsValidMapKey(keyName, e.XID, path.UI()); xErr != nil {
				return xErr
			}
		}

		v := valValue.MapIndex(k).Interface()
		if IsNil(v) {
			continue
		}
		xErr, haveReplacement, newValue := e.ValidateAttribute(v, attr,
			path.P(keyName))
		if xErr != nil {
			return xErr
		}
		if haveReplacement {
			valValue.SetMapIndex(k, reflect.ValueOf(newValue))
		}
	}

	return nil
}

func (e *Entity) ValidateArray(arrayAttr *Attribute, val any, path *PropPath) *XRError {
	if log.IsFuncVerbose() {
		log.Printf("tx: %s item: %s", e.Tx.UUID, ToJSON(arrayAttr.Item))
		log.Printf("tx: %s val: %s", e.Tx.UUID, ToJSON(val))
	}

	if IsNil(val) {
		return nil
	}

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Slice {
		return NewXRError("invalid_attribute", e.XID,
			"name="+path.UI(),
			"error_detail="+"must be an array")
	}

	// All values in the array must be of the same type
	attr := &Attribute{
		Type: arrayAttr.Item.Type,

		Enum:   arrayAttr.Item.Enum,
		Strict: arrayAttr.Item.Strict,
		Target: arrayAttr.Item.Target,

		NameCharSet: arrayAttr.Item.NameCharSet,
		Attributes:  arrayAttr.Item.Attributes,
		Item:        arrayAttr.Item.Item,
	}

	for i := 0; i < valValue.Len(); i++ {
		v := valValue.Index(i).Interface()
		xErr, haveReplacement, newValue := e.ValidateAttribute(v, attr,
			path.I(i))
		if xErr != nil {
			return xErr
		}
		if haveReplacement {
			valValue.Index(i).Set(reflect.ValueOf(newValue))
		}
	}

	return nil
}

// returns: Error, haveReplacementValue, replacementValue
func (e *Entity) ValidateScalar(val any, attr *Attribute, path *PropPath) (*XRError, bool, any) {
	if log.IsFuncVerbose() {
		log.Printf("tx: %s val: %s", e.Tx.UUID, ToJSON(val))
	}

	replace := false
	newValue := (any)(nil)

	// Precompute the type assertions used below - cheaper than repeated
	// reflect.ValueOf(val).Kind() calls, and avoids re-asserting
	// val.(string) in each string-typed case.
	_, isBool := val.(bool)
	valInt, isInt := val.(int)
	valFloat, isFloat := val.(float64)
	valStr, isStr := val.(string)

	switch attr.Type {
	case BOOLEAN:
		if !isBool {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail=must be a boolean"), false, nil
		}
	case DECIMAL:
		if !isInt && !isFloat {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a decimal"), false, nil
		}
	case INTEGER:
		if isFloat {
			if valFloat != float64(int(valFloat)) {
				return NewXRError("invalid_attribute", e.XID,
					"name="+path.UI(),
					"error_detail="+"must be an integer"), false, nil
			}
		} else if !isInt {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be an integer"), false, nil
		}
	case UINTEGER:
		i := 0
		if isFloat {
			i = int(valFloat)
			if valFloat != float64(i) {
				return NewXRError("invalid_attribute", e.XID,
					"name="+path.UI(),
					"error_detail="+"must be a uinteger"), false, nil
			}
		} else if !isInt {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a uinteger"), false, nil
		} else {
			i = valInt
		}
		if i < 0 {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a uinteger"), false, nil
		}
	case XID:
		if !isStr {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be an xid"), false, nil
		}
		str := valStr

		if attr.Target != "" {
			xErr := e.MatchXID(str, attr.Target, attr.Name)
			if xErr != nil {
				return xErr, false, nil
				/*
				   return NewXRError("invalid_attribute", e.XID,
				   "name=" + path.UI(),
				   "error_detail="+err.Error()), false, nil
				*/
			}
		}

		xid, err := ParseXid(str)
		if err != nil {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+
					fmt.Sprintf("value (%s) isn't a valid xid, %s",
						str, err)), false, nil
		}

		if xid.VersionID != "" && xid.Version == "meta" {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+
					fmt.Sprintf("value (%s) isn't a valid xid, "+
						"it must be in the form of: "+
						"/[GROUPS[/GID[/RESOURCES[/GID[/versions[/vid]]]]]]",
						str)), false, nil
		}

		if xid.Type != ENTITY_REGISTRY {
			gm := e.Registry.Model.FindGroupModel(xid.Group)
			if gm == nil {
				return NewXRError("invalid_attribute", e.XID,
					"name="+path.UI(),
					"error_detail="+
						fmt.Sprintf("value (%s) references an unknown "+
							"Group type %q", str, xid.Group)), false, nil
			}

			if xid.Resource != "" {
				rm := gm.Resources[xid.Resource]
				if rm == nil {
					return NewXRError("invalid_attribute", e.XID,
						"name="+path.UI(),
						"error_detail="+
							fmt.Sprintf("value (%s) references an "+
								"unknown Resource type %q", str,
								xid.Resource)), false, nil
				}
			}
		}

	case XIDTYPE:
		if !isStr {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+
					fmt.Sprintf("value  must be an xidtype")), false, nil
		}
		str := valStr

		xidType, err := ParseXidType(str)
		if err != nil {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+
					fmt.Sprintf("value (%s) isn't a valid xidtype, %s",
						str, err)), false, nil
		}

		if xidType.Group != "" {
			gm := e.Registry.Model.FindGroupModel(xidType.Group)
			if gm == nil {
				return NewXRError("invalid_attribute", e.XID,
					"name="+path.UI(),
					"error_detail="+
						fmt.Sprintf("value (%s) references an unknown "+
							"Group type %q", str, xidType.Group)), false, nil
			}

			if xidType.Resource != "" {
				rm := gm.Resources[xidType.Resource]
				if rm == nil {
					return NewXRError("invalid_attribute", e.XID,
						"name="+path.UI(),
						"error_detail="+
							fmt.Sprintf("value (%s) references an "+
								"unknown Resource type %q", str,
								xidType.Resource)), false, nil
				}
			}
		}
	case STRING:
		if !isStr {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a string"), false, nil
		}
	case URI:
		if !isStr {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a uri"), false, nil
		}
	case URIABSOLUTE:
		if !isStr {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a uriabsolute"), false, nil
		}
	case URIRELATIVE:
		if !isStr {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a urirelative"), false, nil
		}
	case URITEMPLATE:
		if !isStr {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a uritemplate"), false, nil
		}
	case URL:
		if !isStr {
			return NewXRError("invalid_attribute", e.XID,
				"name= "+path.UI(),
				"error_detail="+"must be a url"), false, nil
		}
	case URLABSOLUTE:
		if !isStr {
			return NewXRError("invalid_attribute", e.XID,
				"name= "+path.UI(),
				"error_detail="+"must be a urlabsolute"), false, nil
		}
	case URLRELATIVE:
		if !isStr {
			return NewXRError("invalid_attribute", e.XID,
				"name= "+path.UI(),
				"error_detail="+"must be a urlrelative"), false, nil
		}
	case TIMESTAMP:
		if !isStr {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a timestamp"), false, nil
		}
		str := valStr

		var err error
		newValue, err = NormalizeStrTime(str)
		if err != nil {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"is a malformed timestamp"), false, nil
		}
		replace = (newValue != str)
	default:
		panic(fmt.Sprintf("tx; %s Unknown type: %v", e.Tx.UUID, attr.Type))
	}

	// check against enum values - group-level "enum" constraints are
	// enforced separately (and more completely, incl. xref-mirrored
	// data) via Group.validateEnum(), not here.
	enums, strict := attr.Enum, attr.GetStrict()
	// log.Printf("tx: %s Checking: %q: %q vs %q", e.Tx.UUID,
	// attr.Name, val, EnumAsString(enums))
	if strict && !IsValidEnum(val, enums) {
		return NewXRError("invalid_attribute", e.XID,
			"name="+path.UI(),
			"error_detail="+
				fmt.Sprintf("value (%v) must be one of the enum "+
					"values: %s", val, EnumAsString(enums))), false, nil
	}

	return nil, replace, newValue
}

func (e *Entity) GetAttributes(obj Object) Attributes {
	attrs := e.GetBaseAttributes()
	if obj == nil {
		if e.NewObject != nil {
			obj = e.NewObject
		} else {
			obj = e.Object
		}
	}

	attrs.AddIfValuesAttributes(obj)

	return attrs
}

// Returns the initial set of attributes defined for the entity.
func (e *Entity) GetBaseAttributes() Attributes {
	// Add attributes from the model (core and user-defined)
	gm, rm := e.GetModels()

	if e.Type == ENTITY_REGISTRY {
		return e.Registry.Model.GetBaseAttributes()
	}

	if e.Type == ENTITY_GROUP {
		return gm.GetBaseAttributes()
	}

	if e.Type == ENTITY_RESOURCE {
		return rm.GetBaseAttributes()
	}

	if e.Type == ENTITY_META {
		return rm.GetBaseMetaAttributes()
	}

	if e.Type == ENTITY_VERSION {
		// This seems to work for now.
		// At some point we may want to have it only include version-level
		// attributes and not resource-level ones - like versionscount.
		// At which point we may need to add back in the code that removes
		// those resource-level attributes before we create/update a Version
		// (e.g. POST .../rID)
		return rm.GetBaseVersionAttributes()
	}

	panic(fmt.Sprintf("tx: %s Bad type: %v", e.Tx.UUID, e.Type))
}

func (e *Entity) CalcAttrDefault(attr *Attribute, path *PropPath) any {
	if e.Type != ENTITY_VERSION {
		return attr.Default
	}

	v := e.Self.(*Version)
	g := v.Resource.Group

	if c := g.GetAttrConstraint(v, attr, path.P(attr.Name)); c != nil {
		if !IsNil(c.Default) {
			return c.Default
		}
	}

	return attr.Default
}

func (e *Entity) GetOrigin(path string) any {
	if e.OriginObject == nil {
		return nil
	}

	// Note this will NOT do any special processing for things like
	// Resources (r.Singular) that Get() does - we may need to do that one day
	pp, err := PropPathFromUI(path)
	PanicIf(err != nil, "tx: %s %s", e.Tx.UUID, err)

	val, _, _ := ObjectGetProp(e.OriginObject, pp)
	return val
}

func (e *Entity) Get(path string) any {
	pp, err := PropPathFromUI(path)
	PanicIf(err != nil, "tx: %s %s", e.Tx.UUID, err)
	return e.GetPP(pp)
}

func (e *Entity) GetAsString(path string) string {
	val := e.Get(path)
	if IsNil(val) {
		return ""
	}

	if _, ok := val.(string); !ok {
		panic(fmt.Sprintf("tx: %s Not string, got %T(%v)", e.Tx.UUID, val, val))
	}

	str, _ := val.(string)
	return str
}

func (e *Entity) GetOriginAsString(path string) string {
	val := e.GetOrigin(path)
	if IsNil(val) {
		return ""
	}

	if _, ok := val.(string); !ok {
		panic(fmt.Sprintf("tx: %s Not string, got %T(%v)", e.Tx.UUID, val, val))
	}

	str, _ := val.(string)
	return str
}

func (e *Entity) GetAsInt(path string) int {
	val := e.Get(path)
	if IsNil(val) {
		return -1
	}
	i, ok := val.(int)
	PanicIf(!ok, "tx: %s Val: %v  T: %T", e.Tx.UUID, val, val)
	return i
}

func (e *Entity) GetPP(pp *PropPath) any {
	if (e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION) && pp.Len() == 1 {
		rm := e.GetResourceModel()
		if rm.GetHasDocument() && pp.Top() == rm.Singular {
			buf, xErr := e.Tx.Backend.GetResourceContents(e)
			PanicIf(xErr != nil, "tx: %s bubble up?", e.Tx.UUID)
			return buf
		}
	}

	// We used to just grab from Object, not NewObject
	/*
	   // An error from ObjectGetProp is ignored because if they tried to
	   // go into something incorrect/bad we should just return 'nil'.
	   // This may not be the best choice in the long-run - which in case we
	   // should return the 'error'
	   val, _ , _ := ObjectGetProp(e.Object, pp)
	*/

	// See if we have an updated value in NewObject, if not grab from Object
	var val any
	if e.NewObject != nil {
		var ok bool
		val, ok, _ = ObjectGetProp(e.NewObject, pp)
		if !ok {
			// TODO: DUG - we should not need this
			// val, _, _ = ObjectGetProp(e.Object, pp)
		}
	} else {
		val, _, _ = ObjectGetProp(e.Object, pp)
	}

	log.FuncPrintf("tx: %s %s(%s).Get(%s) -> %v", e.Tx.UUID, e.Plural, e.UID,
		pp.DB(), val)
	return val
}

func (e *Entity) GetResourceSingular() string {
	rm := e.GetResourceModel()
	if rm != nil {
		return rm.Singular
	}
	return ""
}

func (e *Entity) GetResourceModel() *ResourceModel {
	_, rm := e.GetModels()
	return rm
}

func (e *Entity) GetGroupModel() *GroupModel {
	gm, _ := e.GetModels()
	return gm
}

// If no match then return an error saying why
func (e *Entity) MatchXID(str string, xid string, attr string) *XRError {
	// 0=all  1=GROUPS  2=RESOURCES  3=versions|""  4=[/versions]|""
	targetParts := targetRE.FindStringSubmatch(xid)

	if len(str) == 0 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail=must be an xid, not empty")
	}
	if str[0] != '/' {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail=must be an xid, and start with /")
	}
	strParts := strings.Split(str, "/")
	if len(strParts) < 2 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail=must be a valid xid")
	}
	if len(strParts[0]) > 0 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail=must be an xid, and start with /")
	}
	if xid == "/" {
		if str != "/" {
			return NewXRError("invalid_attribute", e.XID,
				"name="+attr,
				"error_detail="+fmt.Sprintf("must match %q target", xid))
		}
		return nil
	}
	if targetParts[1] != strParts[1] { // works for "" too
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+fmt.Sprintf("must match %q target", xid))
	}

	gm := e.Registry.Model.Groups[targetParts[1]]
	if gm == nil {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("uses an unknown group %q", targetParts[1]))
	}
	if len(strParts) < 3 || len(strParts[2]) == 0 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, %q is missing \"%sid\"",
					xid, str, gm.Singular))
	}
	if xErr := IsValidID(strParts[2], attr); xErr != nil {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf(`the %q ID is not valid: %s`,
					gm.Singular, xErr.Args["error_detail"]))
	}

	if targetParts[2] == "" { // /GROUPS
		if len(strParts) == 3 {
			return nil
		}
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, extra stuff after %q",
					xid, strParts[2]))
	}

	// targetParts has RESOURCES
	if len(strParts) < 4 { //    /GROUPS/GID/RESOURCES
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, %q is missing %q",
					xid, str, targetParts[2]))
	}

	if targetParts[2] != strParts[3] {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, %q is missing %q",
					xid, str, targetParts[2]))
	}

	rm := gm.Resources[targetParts[2]]
	if rm == nil {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("uses an unknown resource %q", targetParts[2]))
	}

	if len(strParts) < 5 || len(strParts[4]) == 0 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, %q is missing \"%sid\"",
					xid, str, rm.Singular))
	}
	if xErr := IsValidID(strParts[4], attr); xErr != nil {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf(`the %q ID is not valid: %s`,
					rm.Singular, xErr.Args["error_detail"]))
	}

	if targetParts[3] == "" && targetParts[4] == "" {
		if len(strParts) == 5 {
			return nil
		}
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, extra stuff after %q",
					xid, strParts[4]))

	}

	if targetParts[4] != "" { // has [/versions]
		if len(strParts) == 5 {
			//   /GROUPS/RESOURCES[/version]  vs /GROUPS/GID/RESOURCES/RID
			return nil
		}
	}

	if len(strParts) < 6 || strParts[5] != "versions" {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, %q is missing \"versions\"",
					xid, str))
	}

	if len(strParts) < 7 || len(strParts[6]) == 0 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, %q is missing a \"version\" ID",
					xid, str))
	}
	if xErr := IsValidID(strParts[6], attr); xErr != nil {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf(`the "version" ID is not valid: %s`,
					xErr.Args["error_detail"]))
	}

	if len(strParts) > 7 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, too long", xid))
	}

	return nil
}

func (e *Entity) JustSet(path string, val any) *XRError {
	return e.eJustSetPath(path, val)
}

// Set the prop in the Entity (parsing path) but don't Validate or Save
// to the DB
func (e *Entity) eJustSetPath(path string, val any) *XRError {
	pp, err := PropPathFromUI(path)
	if err != nil {
		return NewXRError("bad_request", e.XID,
			"error_detail="+
				fmt.Sprintf("Bad attribute path in \"%s\": %s",
					e.XID, err))
	}
	return e.eJustSet(pp, val)
}

// Set the prop in the Entity but don't Validate or Save to the DB
func (e *Entity) eJustSet(pp *PropPath, val any) *XRError {
	defer log.Trace("tx: %s %s.%s=%v", e.Tx.UUID, e.XID, pp.UI(), val)()

	PanicIf(e.AccessMode != FOR_WRITE, "tx: %s %q isn't FOR_WRITE",
		e.Tx.UUID, e.XID)

	// Assume no other edits are pending
	// e.Refresh() // trying not to have this here

	// If we don't have a NewObject yet then this is our first update
	// so clone the current values before adding the new prop/val
	e.EnsureNewObject()

	// Cheat a little just to make caller's life easier by converting
	// empty structs and maps need to be of the type we like (meaning 'any's)
	if !IsNil(val) {
		if val == struct{}{} {
			val = map[string]any{}
		}
		valValue := reflect.ValueOf(val)
		if valValue.Kind() == reflect.Slice && valValue.Len() == 0 {
			val = []any{}
		}
		if valValue.Kind() == reflect.Map && valValue.Len() == 0 {
			val = map[string]any{}
		}
	}
	// end of cheat

	if pp.Top() == "epoch" {
		e.EpochSet = true
	}
	if pp.Top() == "modifiedat" {
		e.ModSet = true
	}

	// Since "xref" is also a Property on the Resources table we need to
	// set it manually. We can't do it lower down (closer to the DB funcs)
	// because down there "xref" won't appear in NewObject when it's set to nil
	/*
	   if e.Type == ENTITY_RESOURCE && pp.Top() == "xref" {
	       // Handles both val=nil and non-nil cases
	       xErr := Do_OneTwo(e.Tx,
	          `UPDATE Resources SET xRef=? WHERE SID=?`,
	           val, e.SID)
	       if xErr != nil {
	           return xErr
	       }
	   }
	*/

	if log.IsFuncVerbose() {
		log.Printf("tx: %s Abstract/ID: %s/%s", e.Tx.UUID, e.Abstract, e.UID)
		log.Printf("tx: %s e.Object:\n%s", e.Tx.UUID, ToJSON(e.Object))
		log.Printf("tx: %s e.NewObject:\n%s", e.Tx.UUID, ToJSON(e.NewObject))
	}

	err := ObjectSetProp(e.NewObject, pp, val)
	if err != nil {
		return NewXRError("invalid_attribute", e.XID,
			"name="+pp.UI(),
			"error_detail="+err.Error())
	}
	return nil
}

func (e *Entity) EnsureNewObject() bool {
	// Any buffered change must only ever happen on an entity we've
	// already FOR_WRITE-locked (see Entity.Lock()) - otherwise we'd be
	// buffering an edit against a row we never actually took a DB lock
	// on, which is exactly the kind of bug this whole locking-strategy
	// rework is trying to prevent. Catch it here, as early as possible.
	PanicIf(e.AccessMode != FOR_WRITE, "tx: %s %q isn't FOR_WRITE",
		e.Tx.UUID, e.XID)

	if e.NewObject == nil {
		if e.Object == nil {
			e.SetNewObject(map[string]any{})
		} else {
			e.SetNewObject(maps.Clone(e.Object))
		}
		return true
	}
	return false
}

// We use this just to make sure we can set NewObjectStack when we need to
// debug stuff
func (e *Entity) SetNewObject(newObj map[string]any) {
	PanicIf(e.AccessMode != FOR_WRITE, "tx: %s %q isn't FOR_WRITE",
		e.Tx.UUID, e.XID)
	e.NewObject = newObj

	// Copy all system attributes from old Object so we don't lose them
	for k, _ := range e.Object {
		if k[0] == '#' {
			e.NewObject[k] = e.Object[k]
		}
	}

	// Enable the next line when we need to debug when NewObject was created
	// e.NewObjectStack = GetStack()

	// And then use e.ShowStack() to dump it

	/* Sample code to print the stack for where this NewObject was created:
	   log.Printf("tx: %s Stack for NewObject:", e.Tx.UUID)
	   for _, s := range e.NewObjectStack {
	       log.Printf("tx: %s %s", e.Tx.UUID, s)
	   }
	*/
}

func (e *Entity) WasTouched() bool {
	log.FuncPrintf("tx: %s WasTouch: %s", e.Tx.UUID, e.XID)
	return e.EpochSet
}

func (e *Entity) Touch() (bool, *XRError) {
	log.FuncPrintf("tx: %s Touch: %s", e.Tx.UUID, e.XID)

	// See if it's already been modified (and saved) this Tx, if so exit
	if e.ModSet && e.EpochSet {
		return false, nil
	}

	if _, xErr := e.Lock(); xErr != nil {
		return false, xErr
	}

	e.EnsureNewObject()
	return true, nil
}

func (e *Entity) Lock() (bool, *XRError) { // did we lock it?
	defer log.Trace("tx: %s %s:%s", e.Tx.UUID, e.XID, e.SID)()

	if e.AccessMode == FOR_WRITE {
		// Already locked
		return false, nil
	}

	log.FuncPrintf("tx: %s Requesting lock for %q SID=%s connID=%d",
		e.Tx.UUID, e.XID, e.SID, e.Tx.ConnID)

	if xErr := e.Refresh(FOR_WRITE); xErr != nil {
		return false, xErr
	}

	log.FuncPrintf("tx: %s Got lock for %q SID=%s connID=%d",
		e.Tx.UUID, e.XID, e.SID, e.Tx.ConnID)

	// Resource/Meta/Version are logically one unit - Meta holds the
	// data most callers actually care about (defaultversionid, epoch,
	// etc) and CheckAncestors()/EnsureLatest() need a consistent view
	// across all of a Resource's Versions to compute their answers, but
	// individually FOR_WRITE-locking just "e" only ever locks e's own
	// row. Whenever e is a Resource, Meta, or Version, also lock the
	// whole Resource+Meta+Versions family together here, in one place,
	// so callers never need to remember to do this themselves at each
	// call site.
	if xErr := LockEntityFamily(e.Tx, e); xErr != nil {
		return false, xErr
	}

	return true, nil
}

func (e *Entity) ShowStack() {
	log.Printf("tx: %s Stack for NewObject (%s):", e.Tx.UUID, e.XID)
	for _, s := range e.NewObjectStack {
		log.Printf("tx: %s  %s", e.Tx.UUID, s)
	}
}

// Update the entity's Object - not the other props in Entity. Similar to
// Raw EntityFromXID
func (e *Entity) Refresh(accessMode int) *XRError {
	defer log.Trace("tx: %s %s", e.Tx.UUID, e.XID)()

	// If there's a buffered, not-yet-persisted system-prop change on
	// this entity (see SetSystemDBProperty()'s doc comment), flush it
	// to the DB now, BEFORE reloading fresh state from the DB below -
	// otherwise this Refresh() would silently discard it. Concretely:
	// ClearResourceSystemDBProperty() calls FindVersion(uid,
	// FOR_WRITE) on every Version of a Resource, which upgrades and
	// Lock()s/Refresh()es any Version that's already cached at a lower
	// access mode - including one that EnsureCompat()'s earlier
	// format-validation loop may have JUST buffered a
	// formatvalidated/formatvalidatedreason change on (via
	// SetSystemDBProperty(), itself still unflushed at that point).
	// Without this, that buffered write would be wiped out here before
	// ever reaching the DB, right before EnsureCompat() returns.
	if e.NewSystem != nil {
		if xErr := e.SaveSystemProps(); xErr != nil {
			return xErr
		}
	}

	if xErr := e.Tx.Backend.RefreshEntity(e, accessMode); xErr != nil {
		return xErr
	}

	// Erase "new" props
	e.NewObject = nil
	e.NewSystem = nil

	// TODO see if we can remove this - it scares me.
	// Added when I added Touch() - touching parent on add/remove child
	e.EpochSet = false
	e.ModSet = false

	if accessMode == FOR_WRITE {
		e.AccessMode = FOR_WRITE
	}
	if e.AccessMode == 0 {
		e.AccessMode = FOR_READ
	}

	e.Tx.AddToCache(e)

	return nil
}

func (e *Entity) Save() *XRError {
	defer log.Trace("tx: %s %s", e.Tx.UUID, e.XID)()

	PanicIf(e.AccessMode != FOR_WRITE, "tx: %s %q isn't FOR_WRITE",
		e.Tx.UUID, e.XID)

	if log.IsFuncVerbose() {
		log.Printf("tx: %s NewObject:\n%s", e.Tx.UUID, ToJSON(e.NewObject))
		// ShowStack()
	}

	xErr := e.Tx.Backend.SaveEntity(e)
	if xErr != nil {
		return xErr
	}

	// Set e.Object to e.NewObject, skipping all 'nil' attributes
	e.Object = map[string]any{}
	for k, v := range e.NewObject {
		if !IsNil(v) {
			e.Object[k] = v
		}
	}
	e.NewObject = nil

	// Update tx-wide Validate lists based on e.Type - processed at end of tx
	if e.Type == ENTITY_GROUP {
		e.Tx.AddGroupToValidate(e.Self.(*Group))
	} else if e.Type == ENTITY_VERSION {
		// onlyMetaChanged=false because we need the full validation code
		e.Tx.AddResourceToValidate(e.Self.(*Version).Resource, false, false)
	} else if e.Type == ENTITY_META {
		e.Tx.AddResourceToValidate(e.Self.(*Meta).Resource, true, false)
	}

	return nil
}

// Set, Validate and Save to DB but not Commit
func (e *Entity) eSetSave(path string, val any) *XRError {
	defer log.Trace("tx: %s %s=%v", e.Tx.UUID, path, val)()

	pp, err := PropPathFromUI(path)
	if err != nil {
		return NewXRError("bad_request", e.XID,
			"error_detail="+
				fmt.Sprintf("Bad attribute path in \"%s\": %s", e.XID, err))
	}

	// Set, Validate and Save
	xErr := e.SetPP(pp, val)
	if xErr != nil {
		return xErr.SetSubject(e.XID)
	}
	return nil
}

// SaveSystemProps flushes any system-prop changes buffered by
// SetSystemDBProperty() (into e.NewSystem) since the last flush. It's
// called once per cached entity at Tx-commit time (see
// tx.WriteCache()) and is a no-op if nothing was buffered. It diffs
// NewSystem against System so only props that actually changed get
// written to the DB.
func (e *Entity) SaveSystemProps() *XRError {
	if e.NewSystem == nil {
		return nil
	}

	newSystem := e.NewSystem
	e.NewSystem = nil

	changed := map[string]any{}
	for name, newVal := range newSystem {
		oldVal, existed := e.System[name]
		if IsNil(newVal) {
			if existed && !IsNil(oldVal) {
				changed[name] = nil
			}
			continue
		}
		if !existed || !reflect.DeepEqual(oldVal, newVal) {
			changed[name] = newVal
		}
	}

	e.System = newSystem

	if len(changed) == 0 {
		return nil
	}
	xErr := e.Tx.Backend.SaveChangedSystemProps(e, changed)
	if xErr != nil {
		return xErr
	}

	// If this is a Version, make sure we fully validate its owning Resource
	if e.Type == ENTITY_VERSION {
		if v, ok := e.Self.(*Version); ok {
			e.Tx.AddResourceToValidate(v.Resource, true, false)
		}
	}

	return nil
}

// LockResourceFamily locks (FOR UPDATE) just the Resource's own row in
// the Resources table.
//
// Every write path locks the Resource FIRST - before it ever creates or
// touches any Meta/Version/Entities row for it (Group.UpsertResource()
// locks the parent Group first, then FindResource(FOR_WRITE) locks the
// Resource before any Meta/Version insert; HTTPDeleteVersions() locks
// the Resource before reading/deleting any Version, etc). That means
// this single lock already fully serializes any two Txs that would
// otherwise race on the same Resource's family: a second Tx blocks here
// before it can create/modify/delete anything else in the family.
// Verified empirically: a Tx holding only this Resources-row FOR UPDATE
// lock fully blocks a concurrent Tx's plain "INSERT INTO Versions"
// for the same ResourceSID until the first Tx commits/rolls back (even
// though Resources and Versions are different tables/no FK) - the
// second Tx can't even reach its own INSERT until it re-acquires this
// same Resources row lock first, per the locking order above.
//
// The remaining two needs handled by locking Metas/Versions/Entities
// too are each already covered elsewhere:
//   - RR-snapshot freshness for plain SELECTs against Metas/Versions
//     (newestVersionID(), GetProblematicVersions(), etc) is handled by
//     each function's own "if meta.AccessMode==FOR_WRITE" FOR UPDATE
//     hint, independent of this function.
//   - Entity.Save()'s Props DELETE-then-INSERT race is handled by
//     Refresh(FOR_WRITE)'s own "SELECT ... FROM Entities WHERE SID=?
//     FOR UPDATE" on the specific entity being saved.
//
// The old code below locked the whole family (Metas/Versions/Entities
// rows too) - left here, commented out, as a reminder/fallback in case
// concurrency issues resurface and we need to revisit this narrowing.
func LockResourceFamily(tx *Tx, resourceSID string) *XRError {
	return tx.Backend.LockResourceBySID(tx, resourceSID)

	/* OLD
	   lock := func(query string) {
	       results := Q uery(tx, query, resourceSID)
	       results.Close()
	   }

	   lock(`SELECT SID FROM Resources WHERE SID=? FOR UPDATE`)
	*/

	/* OLD
	   lock(`SELECT SID FROM Metas WHERE ResourceSID=? FOR UPDATE`)
	   lock(`SELECT SID FROM Versions WHERE ResourceSID=? FOR UPDATE`)

	   // Entities holds both this Resource's own row (keyed by SID) and
	   // its children's rows (keyed by ParentSID) - two different indexes
	   // (PRIMARY on SID, a separate index on ParentSID). A single query
	   // with "WHERE SID=? OR ParentSID=?" can't use either index (MySQL
	   // falls back to a full index scan of the WHOLE table), which - since
	   // this is a locking FOR UPDATE read - ends up granting X locks on
	   // every OTHER Resource's rows in the table too (verified via
	   // performance_schema.data_locks), a real over-locking bug, not just
	   // a missed optimization. Splitting into two separate queries lets
	   // each one use its own index and lock only the rows that actually
	   // belong to this Resource.
	   lock(`SELECT SID FROM Entities WHERE SID=? FOR UPDATE`)
	   lock(`SELECT SID FROM Entities WHERE ParentSID=? FOR UPDATE`)
	*/

	return nil
}

// lockEntityFamily locks (FOR UPDATE) the full Resource+Meta+Versions
// family that ent belongs to, if any - see LockResourceFamily()'s and
// Entity.Lock()'s doc comments for why. Safe to call with a nil ent or
// a non-Resource/Meta/Version entity (no-op).
//
// IMPORTANT: LockResourceFamily() only takes real DB-level "FOR UPDATE"
// row locks - it has no idea about this Tx's Go-level entity Cache. If
// some OTHER member of the family (e.g. the Meta, when ent is a
// Version) was already fetched FOR_READ earlier in this same Tx and is
// sitting in tx.Cache, DB-locking its row here does NOT, by itself,
// upgrade that cached Go object's AccessMode or refresh its
// Object/NewObject - it would stay FOR_READ with whatever (possibly
// now-stale, or read mid-another-Tx's-Save()) data it originally
// loaded. Every "if meta.AccessMode == FOR_WRITE { lockExpr = ...}"-
// style guard added throughout resource.go/versionmodes.go implicitly
// assumes that AccessMode faithfully reflects whether the row is
// really DB-locked - so after LockResourceFamily() runs, we must also
// walk this Tx's cache and upgrade (Refresh(FOR_WRITE)) every other
// cached Resource/Meta/Version belonging to this same family, or those
// guards (and any code that just trusts a cached Meta/Version's
// AccessMode) can silently keep using stale FOR_READ data even though
// the DB row underneath was just real-locked. This was the root cause
// of a recurring "<attr> is nil" panic under TestMiscConcurrency.
func LockEntityFamily(tx *Tx, ent *Entity) *XRError {
	if ent == nil {
		return nil
	}
	resourceSID := ""
	switch ent.Type {
	case ENTITY_RESOURCE:
		resourceSID = ent.SID
	case ENTITY_META, ENTITY_VERSION:
		resourceSID = ent.ParentSID
	}
	if resourceSID == "" {
		return nil
	}
	if xErr := LockResourceFamily(tx, resourceSID); xErr != nil {
		return xErr
	}

	// Now upgrade any other cached family member (Resource itself, its
	// Meta, or any of its Versions) that isn't already FOR_WRITE, so the
	// Go-level cache matches the DB-level lock state we just took.
	for _, cached := range tx.Cache {
		if cached == ent || cached.AccessMode == FOR_WRITE {
			continue
		}
		isFamilyMember :=
			(cached.Type == ENTITY_RESOURCE && cached.SID == resourceSID) ||
				((cached.Type == ENTITY_META || cached.Type == ENTITY_VERSION) &&
					cached.ParentSID == resourceSID)
		if !isFamilyMember {
			continue
		}
		if cached.NewObject != nil || cached.NewSystem != nil {
			// This entity has buffered, not-yet-persisted edits already
			// applied to it earlier in this same Tx (e.g. an in-progress
			// SetSave()/Update() sequence). Calling Refresh() here would
			// discard e.NewObject wholesale (it only flushes NewSystem,
			// not NewObject - see Refresh()'s own doc comment), silently
			// losing those edits. Just mark it FOR_WRITE (its row is now
			// genuinely DB-locked via LockResourceFamily() above) without
			// reloading, so we don't clobber the pending edits.
			cached.AccessMode = FOR_WRITE
			continue
		}
		log.FuncPrintf("tx: %s lockEntityFamily: upgrading stale cached %q "+
			"(SID=%s) to FOR_WRITE now that its family's DB rows are locked",
			tx.UUID, cached.XID, cached.SID)
		if xErr := cached.Refresh(FOR_WRITE); xErr != nil {
			return xErr
		}
	}

	return nil
}

// This is really just an internal Setter used for testing.
// It'll set a property and then validate and save the entity in the DB
func (e *Entity) SetPP(pp *PropPath, val any) *XRError {
	defer log.Trace("tx: %s %s: %s=%v", e.Tx.UUID, e.SID, pp.UI(), val)()
	defer func() {
		if log.IsFuncVerbose() {
			log.Printf("tx: %s exit: e.Object:\n%s", e.Tx.UUID, ToJSON(e.Object))
		}
	}()

	if xErr := e.eJustSet(pp, val); xErr != nil {
		return xErr
	}

	xErr := e.ValidateAndSave(false)
	if xErr != nil {
		// If there's an error, and we're making the assumption that we're
		// setting and saving all in one shot (and there are no other edits
		// pending), go ahead and undo the changes since they're wrong.
		// Otherwise the caller would need to call Refresh themselves.

		// Not sure why setting it to nil isn't sufficient (todo)
		// e.NewObject = nil
		if tmpxErr := e.Refresh(FOR_READ); tmpxErr != nil {
			panic(fmt.Sprintf("tx: %s %s", e.Tx.UUID, tmpxErr))
		}
	}

	return xErr
}

func GoToOurType(val any) string {
	// Fast path: type switch avoids the reflect.ValueOf() overhead for
	// the concrete types that make up the overwhelming majority of calls
	// (JSON-decoded values, plus our own int/uint64/struct{}{} usages).
	switch val.(type) {
	case bool:
		return BOOLEAN
	case int:
		return INTEGER
	case uint64:
		return UINTEGER
	case float64:
		return DECIMAL
	case string:
		return STRING
	case []any:
		return ARRAY
	case map[string]any:
		return MAP
	case struct{}:
		return OBJECT
	}

	panic(fmt.Sprintf("Bad type: %s", reflect.ValueOf(val).Type()))

	// Slow path: fall back to reflect.Kind for anything not covered above
	// (e.g. some other named slice/map/struct type), so behavior stays
	// identical to before for any case we didn't anticipate.
	switch reflect.ValueOf(val).Kind() {
	case reflect.Interface:
		return ANY
	case reflect.Slice:
		return ARRAY
	case reflect.Map:
		return MAP
	case reflect.Struct:
		return OBJECT
	}
	panic(fmt.Sprintf("Bad Kind: %v", reflect.ValueOf(val).Kind()))
}

func (e *Entity) ToString() string {
	str := fmt.Sprintf("%s\n  Object: %s\n  NewObject: %s",
		e.XID, ToJSON(e.Object), ToJSON(e.NewObject))
	return str
}

// Clears system prop(s) for all versions of this resource. Accepts
// multiple PropPaths so callers that need to clear several system
// props at once (e.g. EnsureCompat's format/compat validated+reason
// attrs) can do so in one pass, instead of looping over all Versions
// once per prop. Goes through the same buffered SetSystemDBProperty()
// mechanism as everything else (see Entity.System's doc comment) -
// using r.FindVersion() (not a throwaway shell object) so a clear
// issued after some other system-prop write on the same Version within
// this same request correctly composes with (and can override) that
// earlier buffered value, rather than racing an immediate DB DELETE
// against a later flush.
func (e *Entity) ClearResourceSystemDBProperty(pps ...*PropPath) *XRError {
	defer log.Trace("tx: %s %d props", e.Tx.UUID, len(pps))()

	if len(pps) == 0 {
		return nil
	}

	r, ok := e.Self.(*Resource)
	PanicIf(!ok, "tx: %s %s isn't a Resource", e.Tx.UUID, e.XID)

	vers, xErr := r.GetVersions(FOR_WRITE)
	if xErr != nil {
		return xErr
	}

	for _, v := range vers {
		for _, pp := range pps {
			v.SetSystemDBProperty(pp, nil)
		}
	}

	return nil
}

// SetSystemDBProperty buffers a system-prop change into e.NewSystem -
// it does NOT write to the DB immediately. The actual write (and, if
// e is a Version, at most one re-run of the default-version cascade)
// happens later, via SaveSystemProps() - either at Tx-commit time
// (called from tx.WriteCache()), or earlier if a caller explicitly
// calls tx.FlushSystemProps() (e.g. EnsureCompat()'s callers do this
// right away, so the buffered values are visible in the same request's
// HTTP response, before the Tx actually commits). This lets callers
// like EnsureCompat() set several system props on the same entity
// back-to-back without each one independently re-triggering the whole
// cascade - see SaveSystemProps().
func (e *Entity) SetSystemDBProperty(pp *PropPath, val any) {
	defer log.Trace("tx: %s %s=%v", e.Tx.UUID, pp, val)()

	PanicIf(pp.UI() == "", "tx: %s pp is empty", e.Tx.UUID)

	/* DUG FT
			name := pp.DB()

			_, propsMap := e.GetPropsOrdered()
			specProp, ok := propsMap[pp.Top()]
			if ok && specProp.Internals != nil {
				// Any prop with "dontStore"=true we skip
				if specProp.Internals.dontStore {
					return
				}
			}

			PanicIf(len(name) > MAX_PROPNAME,
		                "tx: %s SysProp name is too long: %s", e.Tx.UUID, name)
			PanicIf(e.SID == "", "tx: %s SID should not be empty", e.Tx.UUID)
			PanicIf(e.Registry == nil, "tx: %s Registry should not be nil",
			            e.Tx.UUID)

			if !IsNil(val) {
				switch reflect.ValueOf(val).Kind() {
				case reflect.String:
					PanicIf(reflect.ValueOf(val).Len() > MAX_VARCHAR,
	                "tx %s %s:too long", e.Tx.UUID, name)
				case reflect.Slice:
					PanicIf(reflect.ValueOf(val).Len() > 0, "%s:non-empty", name)
				case reflect.Map:
					PanicIf(reflect.ValueOf(val).Len() > 0, "%s:non-empty", name)
				case reflect.Struct:
					PanicIf(reflect.ValueOf(val).Len() > 0, "%s:non-empty", name)
				}
			}
	*/

	// Key by pp.Top() (the plain attribute name), NOT pp.DB() (which
	// has a trailing DB_IN separator) - this must match the key scheme
	// setFromDBNameInto() uses to populate e.System from the DB (via
	// ObjectSetProp(), keyed by the plain name), so a value loaded from
	// the DB and a value buffered here land under the SAME key and can
	// be diffed/overridden correctly in SaveSystemProps().
	if e.NewSystem != nil {
		if reflect.DeepEqual(e.NewSystem[pp.Top()], val) {
			return
		}
	} else if e.System != nil {
		if reflect.DeepEqual(e.System[pp.Top()], val) {
			return
		}
	}

	e.EnsureNewSystem()
	e.NewSystem[pp.Top()] = val

	e.Tx.AddToCache(e)
}

// EnsureNewSystem lazily initializes e.NewSystem (buffered, uncommitted
// system-prop changes) as a clone of e.System - mirrors
// EnsureNewObject(), but must NEVER touch epoch/modifiedat/EpochSet/
// ModSet/NewObject, since system props are, by design, invisible to
// the entity's own change-tracking (see Entity.System's doc comment).
func (e *Entity) EnsureNewSystem() {
	// Same guard as EnsureNewObject() - system-prop buffering must only
	// ever happen on an already FOR_WRITE-locked entity.
	PanicIf(e.AccessMode != FOR_WRITE, "tx: %s %q isn't FOR_WRITE",
		e.Tx.UUID, e.XID)

	if e.NewSystem != nil {
		return
	}
	e.NewSystem = map[string]any{}
	for k, v := range e.System {
		e.NewSystem[k] = v
	}
}

// Should be called by Backend impls to fill-in e.OriginObject either
// from e.Object or a previously cached e's OriginObject
func (e *Entity) PopulateOriginObject() {
	if e == nil {
		return
	}

	regSID := e.SID // assume 'e' is a registry until we know otherwise
	if e.Registry != nil {
		// ok, not a registry, so grab reg's SID
		regSID = e.Registry.SID
	}

	if tmpE := e.Tx.GetEntity(regSID, e.XID); tmpE != nil {
		e.OriginObject = tmpE.OriginObject
		e.OriginSystem = tmpE.OriginSystem
	} else {
		e.OriginObject = maps.Clone(e.Object)
		e.OriginSystem = maps.Clone(e.System)

		// Just make sure they're not nil as a sign that we tried to
		// load them but this a new entity so there's no data yet
		if e.OriginObject == nil {
			e.OriginObject = map[string]any{}
		}
		if e.OriginSystem == nil {
			e.OriginSystem = map[string]any{}
		}
	}
}

// This is used to take a DB entry and update the current Entity's Object
// SetFromDBName parses one Props row's PropValue/PropType and
// applies it into e.Object (own/calculated/cascaded props) or e.System
// (system props, isSystem=true) - the row-processing loop is otherwise
// identical for both, but system props go into their own bucket (see
// Entity.System's doc comment) since they must never touch Object,
// epoch, or modifiedat.
func (e *Entity) SetFromDBName(name string, val *string, propType string) *XRError {
	return e.setFromDBNameInto(&e.Object, name, val, propType)
}

func (e *Entity) SetSystemFromDBName(name string, val *string, propType string) *XRError {
	return e.setFromDBNameInto(&e.System, name, val, propType)
}

func (e *Entity) setFromDBNameInto(dest *map[string]any, name string, val *string, propType string) *XRError {
	var err error
	pp := MustPropPathFromDB(name)

	if val == nil {
		err := ObjectSetProp(*dest, pp, val)
		if err != nil {
			return NewXRError("bad_request", e.XID,
				"error_detail=Error setting attribute: "+err.Error())
		}
		return nil
	}
	if *dest == nil {
		*dest = map[string]any{}
	}
	obj := *dest

	if IsString(propType) {
		err = ObjectSetProp(obj, pp, *val)
	} else if propType == BOOLEAN {
		// Technically the "1" check shouldn't be needed, but just in case
		err = ObjectSetProp(obj, pp, (*val == "1" || (*val == "true")))
	} else if propType == INTEGER || propType == UINTEGER {
		tmpInt, err := strconv.Atoi(*val)
		if err != nil {
			panic(fmt.Sprintf("tx: %s error parsing int: %s: %s",
				e.Tx.UUID, *val, err))
		}
		err = ObjectSetProp(obj, pp, tmpInt)
	} else if propType == DECIMAL {
		tmpFloat, err := strconv.ParseFloat(*val, 64)
		if err != nil {
			panic(fmt.Sprintf("tx: %s error parsing float: %s: %s",
				e.Tx.UUID, *val, err))
		}
		err = ObjectSetProp(obj, pp, tmpFloat)
	} else if propType == MAP {
		if *val != "" {
			panic(fmt.Sprintf("tx: %s MAP value should be empty string",
				e.Tx.UUID))
		}
		err = ObjectSetProp(obj, pp, map[string]any{})
	} else if propType == ARRAY {
		if *val != "" {
			panic(fmt.Sprintf("tx: %s MAP value should be empty string",
				e.Tx.UUID))
		}
		err = ObjectSetProp(obj, pp, []any{})
	} else if propType == OBJECT {
		if *val != "" {
			panic(fmt.Sprintf("tx: %s MAP value should be empty string",
				e.Tx.UUID))
		}
		err = ObjectSetProp(obj, pp, map[string]any{})
	} else {
		panic(fmt.Sprintf("tx: %s bad type(%s): %v", propType, e.Tx.UUID, name))
	}

	if err != nil {
		return NewXRError("bad_request", e.XID,
			"error_detail=Error setting attribute: "+err.Error())
	}
	return nil
}

func (e *Entity) GetPropsOrdered() ([]*Attribute, map[string]*Attribute) {
	switch e.Type {
	case ENTITY_REGISTRY:
		return e.Registry.Model.GetPropsOrdered()
	case ENTITY_GROUP:
		gm, _ := e.GetModels()
		return gm.GetPropsOrdered()
	case ENTITY_RESOURCE:
		_, rm := e.GetModels()
		return rm.GetPropsOrdered()
	case ENTITY_META:
		_, rm := e.GetModels()
		return rm.GetMetaPropsOrdered()
	case ENTITY_VERSION:
		_, rm := e.GetModels()
		return rm.GetVersionPropsOrdered()
	default:
		panic(fmt.Sprintf("tx: %s type: %v", e.Tx.UUID, e.Type))
	}
}

// This is used to serialize an Entity regardless of the format.
// This will:
//   - Use AddCalcProps() to fill in any missing props (eg Entity's GetFn())
//   - Call the passed-in 'fn' to serialize each prop but in the right order
//     as defined by the entity's GetPropsOrdered()
func (e *Entity) SerializeProps(
	fn func(*Entity, string, any, *Attribute) *XRError) *XRError {
	defer log.Trace("tx: %s %s", e.Tx.UUID, e.XID)()

	info := e.GetRequestInfo()
	daObj := e.AddCalcProps()
	attrs := e.GetAttributes(e.Object)

	if log.IsFuncVerbose() {
		log.Printf("tx: %s SerProps.Entity: %s", e.Tx.UUID, ToJSON(e))
		log.Printf("tx: %s SerProps.Obj: %s", e.Tx.UUID, ToJSON(e.Object))
		log.Printf("tx: %s SerProps daObj: %s", e.Tx.UUID, ToJSON(daObj))
		log.Printf("tx: %s SerProps attrs:\n%s", e.Tx.UUID, ToJSON(attrs))
	}

	resourceSingular := ""
	hasDoc := false
	if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION || e.Type == ENTITY_META {
		_, rm := e.GetModels()
		resourceSingular = rm.Singular
		hasDoc = rm.GetHasDocument()
	}

	propsOrdered, propsMap := e.GetPropsOrdered()

	// Loop over the defined props - extensions are done under the "if...$ext"
	for _, prop := range propsOrdered {
		name := prop.Name

		// If hasDoc && we're in doc view &&  on the Resource (not version)
		// then skip the RESOURCE/RESOURCEbase64 attributes.
		// Other version-level attribute are automatically excluded by the
		// query. RESOURCE* aren't part of the Props table, so they're special
		if hasDoc && e.Type == ENTITY_RESOURCE && info.DoDocView() {
			if name == resourceSingular || name == resourceSingular+"base64" {
				continue
			}
		}

		log.FuncPrintf("tx: %s Ser prop(%s): %q", e.Tx.UUID, e.XID, name)

		attr, ok := attrs[name]
		if !ok {
			log.FuncPrintf("tx: %s  skipping %q, no attr", e.Tx.UUID, name)
			delete(daObj, name)
			continue // not allowed at this eType so skip it
		}

		if prop.Name == "$extensions" {
			if prop.InType(e.Type) {
				for _, objKey := range SortedKeys(daObj) {
					// Skip spec defined properties
					if propsMap[objKey] != nil {
						continue
					}

					val, _ := daObj[objKey]
					attr := attrs[objKey]
					delete(daObj, objKey)
					if attr == nil {
						attr = attrs["*"]
						PanicIf(objKey[0] != '#' && attr == nil,
							"tx: %s Can't find attr for (%s) %q",
							e.Tx.UUID, e.XID, objKey)
					}
					// log.Printf("tx: %s Ser*ext(%s): %q", e.Tx.UUID, e.XID,
					//  objKey)

					if xErr := fn(e, objKey, val, attr); xErr != nil {
						return xErr
					}
				}
			}
			continue
		}

		if name[0] == '$' || (prop.Internals != nil && prop.Internals.AlwaysSerialize) {
			log.FuncPrintf("tx: %s forced serialization of %q", e.Tx.UUID, name)
			if xErr := fn(e, name, nil, attr); xErr != nil {
				return xErr
			}
			continue
		}

		// Should be a no-op for Resources.
		if val, ok := daObj[name]; ok {
			log.FuncPrintf("tx: %s val: %v", e.Tx.UUID, val)
			if !IsNil(val) {
				xErr := fn(e, name, val, attr)
				if xErr != nil {
					return xErr
				}
			}
			delete(daObj, name)
		} else {
			log.FuncPrintf("tx: %s no value for %q", e.Tx.UUID, name)
		}
	}

	// Now do all other props (extensions) alphabetically
	/*
		for _, objKey := range SortedKeys(daObj) {
			attrKey := objKey
			if attrKey == e.Singular+"id" {
				attrKey = "id"
			}
			val, _ := daObj[objKey]
			attr := attrs[attrKey]
			if attr == nil {
				attr = attrs["*"]
				PanicIf(attrKey[0] != '#' && attr == nil,
					"tx: %s Can't find attr for %q", e.Tx.UUID, attrKey)
			}

			if xErr := fn(e, objKey, val, attr); xErr != nil {
				return xErr
			}
		}
	*/

	return nil
}

// This will add in the calculated properties into the entity. This will
// normally be called after a query using FullTree view and before we serialize
// the entity we need to add the non-DB-stored properties (meaning, the
// calculated ones.
// Note that we make a copy and don't touch the entity itself. Serializing
// an entity shouldn't have side-effects.
func (e *Entity) AddCalcProps() map[string]any {
	mat := map[string]any{}
	// System props (formatvalidated, compatibilityvalidated, ...) live in
	// their own bucket (see Entity.System's doc comment) so their writes
	// don't touch epoch/modifiedat - but they're still real, serializable
	// top-level attributes, so merge them in here (Object wins on any
	// overlap, though the two buckets should never share a key).
	for k, v := range e.System {
		mat[k] = v
	}
	for k, v := range e.Object {
		mat[k] = v
	}

	// Regardless of the type of entity, set the generated properties
	propsOrdered, _ := e.GetPropsOrdered()

	for _, prop := range propsOrdered {
		// Only generate props that have a Fn
		if prop.Internals != nil && prop.Internals.GetFn != nil {
			// Only generate/set the value if it's not already set
			if _, ok := mat[prop.Name]; !ok {
				if val := prop.Internals.GetFn(e); !IsNil(val) {
					// Only write it if we have a value
					// log.Printf("tx: %s Added calc prop: %q",
					// e.Tx.UUID, prop.Name)
					mat[prop.Name] = val
				}
			}
		}
	}

	return mat
}

// This will remove all Collection related attributes from the entity.
// While this is an Entity.Func, we allow callers to pass in the Object
// data to use instead of the e.Object/NewObject so that we'll use this
// Entity's Type (which tells us which collections it has), on the 'obj'.
// This is handy for cases where we need to remove the Resource's collections
// from a Version's Object - like on  a PUT to /GROUPs/gID/RESOURECEs/rID
// where we're passing in what looks like a Resource entity, but we're
// really using it to create a Version
func (e *Entity) RemoveCollections(obj Object) {
	if obj == nil {
		obj = e.NewObject
	}

	for _, coll := range e.GetCollections() {
		delete(obj, coll[0])
		delete(obj, coll[0]+"count")
		delete(obj, coll[0]+"url")
	}
}

// Array of plural/singular pairs
func (e *Entity) GetCollections() [][2]string {
	result := [][2]string{}
	switch e.Type {
	case ENTITY_REGISTRY:
		gs := e.Registry.Model.Groups
		for _, k := range Keys(gs) {
			result = append(result, [2]string{gs[k].Plural, gs[k].Singular})
		}
		return result
	case ENTITY_GROUP:
		gm, _ := e.GetModels()
		for _, rm := range gm.Resources {
			result = append(result, [2]string{rm.Plural, rm.Singular})
		}
		return result
	case ENTITY_RESOURCE:
		result = append(result, [2]string{"versions", "version"})
		return result
	case ENTITY_META:
		return nil
	case ENTITY_VERSION:
		return nil
	}
	panic(fmt.Sprintf("tx: %s bad type: %d", e.Tx.UUID, e.Type))
	return nil
}

func (e *Entity) RemoveReadOnlyImmutable(obj Object) {
	// Don't touch what was passed in
	attrs := e.GetAttributes(obj)
	singular := e.Singular

	if e.Type == ENTITY_VERSION {
		singular = e.Self.(*Version).Resource.Singular
	} else if e.Type == ENTITY_META {
		singular = e.Self.(*Meta).Resource.Singular
	}
	singular += "id"

	for _, attr := range attrs {
		if attr.ReadOnly == false && attr.Immutable == false {
			continue
		}
		key := attr.Name

		// epoch and IDs are special, we do need to check their values later
		// so don't delete them
		if key == "versionid" || key == singular || key == "epoch" {
			// However, if this is a new entity(no OriginObject), then go
			// ahead and skip/delete "epoch" so we don't do any do any error
			// checking on it - even it's data type
			if key == "epoch" && len(e.OriginObject) == 0 {
				delete(obj, key)
			}
			continue
		}

		delete(obj, key)
	}
}

func PrepUpdateEntity(e *Entity) *XRError {
	attrs := e.GetAttributes(e.NewObject)

	for key, _ := range attrs {
		attr := attrs[key]

		// Any ReadOnly attribute in Object, but not in NewObject, must
		// be one that we want to keep around. Note that a 'nil' in NewObject
		// will not grab the one in Object - assumes we want to erase the val
		/*
			if attr.ReadOnly {
				oldVal, ok1 := e.Object[attr.Name]
				_, ok2 := e.NewObject[attr.Name]
				if ok1 && !ok2 {
					e.NewObject[attr.Name] = oldVal
				}
			}
		*/

		if e.NewObject != nil {
			if attr.InType(e.Type) && attr.Internals != nil &&
				attr.Internals.UpdateFn != nil {

				if xErr := attr.Internals.UpdateFn(e); xErr != nil {
					return xErr
				}
			}
		}
	}

	return nil
}

// We call this to verify that the top level attribute names are valid.
// We can't really do this during the Validation funcs because at that point
// in the process we may have added #xxx type of attribute names, and "#"
// isn't a valid char. And we need to make sure users don't try to pass in
// attributes that start with "#" to attack us.
// Now, one way around this is to move the system props (#xxx) into a separate
// map (out of Object and NewObject) but then we'd need to duplicate a lot
// of logic - but it might actually make for a cleaner design to keep
// system data out of the user data space, so worth considering in the future.
// I really would prefer to push this down in the stack though.
func CheckAttrs(obj map[string]any, source string) *XRError {
	if obj == nil {
		return nil
	}
	for k, _ := range obj {
		if xErr := IsValidAttributeName(k, source, ""); xErr != nil {
			// log.Printf("tx: %s Key: %q", e.Tx.UUID, k)
			// ShowStack()
			return xErr
		}
	}
	return nil
}
