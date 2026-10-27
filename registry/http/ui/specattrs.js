// AUTO-GENERATED — do not edit directly.
// Source: common/shared_entity  |  Regenerate: make .sharedfiles
// Generator: cmds/genspecattrs/main.go
//
var XREG_UI_COMMIT = "$COMMIT";

// Spec-defined attributes per entity level.
// Extensions: attrs NOT in this set, NOT <singular>id, NOT collection keys.

var SPEC_ATTRS = {
  registry: {$COLLECTIONS:1, $extensions:1, $space:1, capabilities:1, createdat:1, description:1, documentation:1, epoch:1, icon:1, id:1, labels:1, model:1, modelsource:1, modifiedat:1, name:1, self:1, shortself:1, specversion:1, xid:1},
  group:    {$COLLECTIONS:1, $extensions:1, $space:1, constraints:1, createdat:1, deprecated:1, description:1, documentation:1, epoch:1, icon:1, id:1, labels:1, modifiedat:1, name:1, self:1, shortself:1, xid:1},
  resource: {$COLLECTIONS:1, $extensions:1, $space:1, id:1, meta:1, metaurl:1, self:1, shortself:1, xid:1},
  meta:     {$extensions:1, $space:1, compatibility:1, createdat:1, defaultversionid:1, defaultversionsticky:1, defaultversionurl:1, deprecated:1, epoch:1, id:1, labels:1, modifiedat:1, readonly:1, self:1, shortself:1, xid:1, xref:1},
  version:  {$RESOURCE:1, $RESOURCEbase64:1, $RESOURCEproxyurl:1, $RESOURCEurl:1, $extensions:1, $space:1, ancestorid:1, compatibilityvalidated:1, compatibilityvalidatedreason:1, contenttype:1, createdat:1, description:1, documentation:1, epoch:1, format:1, formatvalidated:1, formatvalidatedreason:1, icon:1, id:1, isdefault:1, labels:1, modifiedat:1, name:1, self:1, shortself:1, versionid:1, xid:1}
};

// String-typed spec attributes that should render in monospace in the UI.
// These are technical identifiers/values, not human-readable prose.
// Non-string spec attrs (boolean, integer, timestamp, url, …) are
// already monospaced via model-type logic and are not listed here.
var MONO_ATTRS = {
  registry: {id:1, specversion:1},
  group:    {id:1},
  resource: {id:1},
  meta:     {compatibility:1, defaultversionid:1, id:1},
  version:  {ancestorid:1, compatibilityvalidatedreason:1, contenttype:1, format:1, formatvalidatedreason:1, id:1, versionid:1}
};

// Display label overrides for spec-defined attributes.
// labelFor() uses this only when the attribute is confirmed spec-defined
// at the current entity level; extension attrs with the same name get
// the raw attribute name as their label.
var LABEL_ATTRS = {
  ancestorid: "Ancestor Version ID",
  compatibility: "Compatibility",
  compatibilityvalidated: "Compatibility Validated",
  compatibilityvalidatedreason: "Compatibility Validated Reason",
  contenttype: "Content Type",
  createdat: "Created",
  defaultversionid: "Default Version ID",
  defaultversionsticky: "Default Version Sticky",
  defaultversionurl: "Default Version URL",
  deprecated: "Deprecated",
  description: "Description",
  documentation: "Documentation",
  format: "Format",
  formatvalidated: "Format Validated",
  formatvalidatedreason: "Format Validated Reason",
  icon: "Icon",
  isdefault: "Is Default",
  labels: "Labels",
  modifiedat: "Modified",
  name: "Name",
  readonly: "Read Only",
  specversion: "Spec Version",
  versionid: "Version ID"
};

// Spec-defined attribute names in declaration order, per entity level.
// Use for column and property ordering — spec attrs appear before extensions.
// Structural '$'-prefixed entries are excluded (never appear as UI columns).
var SPEC_ATTRS_ORDER = {
  registry: ["specversion", "id", "self", "shortself", "xid", "epoch", "name", "description", "documentation", "icon", "labels", "createdat", "modifiedat", "capabilities", "model", "modelsource"],
  group:    ["id", "self", "shortself", "xid", "epoch", "name", "description", "documentation", "icon", "labels", "createdat", "modifiedat", "deprecated", "constraints"],
  resource: ["id", "self", "shortself", "xid", "meta", "metaurl"],
  meta:     ["id", "self", "shortself", "xid", "xref", "epoch", "labels", "createdat", "modifiedat", "readonly", "compatibility", "deprecated", "defaultversionid", "defaultversionurl", "defaultversionsticky"],
  version:  ["id", "versionid", "self", "shortself", "xid", "epoch", "name", "isdefault", "description", "documentation", "icon", "labels", "createdat", "modifiedat", "ancestorid", "contenttype", "format", "formatvalidated", "formatvalidatedreason", "compatibilityvalidated", "compatibilityvalidatedreason"]
};

// Full canonical attribute order per entity level, INCLUDING the
// structural '$space' (blank-line separator) and '$extensions'
// (alphabetized-extension insertion point) markers from
// registry.OrderedSpecProps, in declaration order. Unlike
// SPEC_ATTRS_ORDER above (which drops '$'-prefixed entries — it's
// only used for UI column ordering), this preserves them so a
// canonical-order JSON pretty-printer can reproduce the spec's
// pseudo-JSON layout (see core/spec.md "Design: JSON Serialization").
// '$RESOURCE*'/'$COLLECTIONS' placeholder tokens are kept verbatim —
// a consumer without the real model can't resolve them to real
// attribute names, so it should just skip over them as no-ops.
// Consecutive '$space' entries (which can end up adjacent after
// per-level filtering removes everything between two of them) are
// already collapsed to one here, and no leading/trailing '$space'
// survives — so a consumer can treat every remaining '$space' as
// exactly one blank line to emit.
var SPEC_ATTRS_CANONICAL_ORDER = {
  registry: ["specversion", "registryid", "self", "shortself", "xid", "epoch", "name", "description", "documentation", "icon", "labels", "createdat", "modifiedat", "$extensions", "capabilities", "model", "modelsource", "$space", "$COLLECTIONS"],
  group:    ["id", "self", "shortself", "xid", "epoch", "name", "description", "documentation", "icon", "labels", "createdat", "modifiedat", "deprecated", "constraints", "$extensions", "$space", "$COLLECTIONS"],
  resource: ["id", "versionid", "self", "shortself", "xid", "epoch", "name", "isdefault", "description", "documentation", "icon", "labels", "createdat", "modifiedat", "ancestorid", "contenttype", "format", "formatvalidated", "formatvalidatedreason", "compatibilityvalidated", "compatibilityvalidatedreason", "$extensions", "$space", "$RESOURCEurl", "$RESOURCEproxyurl", "$RESOURCE", "$RESOURCEbase64", "$space", "meta", "metaurl", "$space", "$COLLECTIONS"],
  meta:     ["id", "self", "shortself", "xid", "xref", "epoch", "labels", "createdat", "modifiedat", "readonly", "compatibility", "deprecated", "$extensions", "$space", "defaultversionid", "defaultversionurl", "defaultversionsticky"],
  version:  ["id", "versionid", "self", "shortself", "xid", "epoch", "name", "isdefault", "description", "documentation", "icon", "labels", "createdat", "modifiedat", "ancestorid", "contenttype", "format", "formatvalidated", "formatvalidatedreason", "compatibilityvalidated", "compatibilityvalidatedreason", "$extensions", "$space", "$RESOURCEurl", "$RESOURCEproxyurl", "$RESOURCE", "$RESOURCEbase64"]
};
