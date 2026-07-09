# xRegistry SPA UI — Cross-Session Plan / Outstanding Work

This file tracks design points and follow-up work for the `registry/ui/` SPA
that should persist across agent sessions (the per-session `todos` SQL table
does NOT survive between sessions — this file is the durable record). Update
it as items are completed or newly identified.

See also `newui.md` (original design draft from the "merge old UI into new
design" session, 2026-07-01) for full context on the overall UI redesign
goals.

## Outstanding

- [x] ~~Add filter support to Grid (Tile) and List (Table) views for
  registry Data.~~ **OBSOLETE / done, no action needed** (updated
  2026-07-08 during a later session): Grid view was removed entirely for
  the `data` section in checkpoints 034-035 (`enableGrid = false` for
  every data-section depth — see `registry/ui/app.js` ~line 689/692), so
  there's no Grid view left to add filtering to. The List (Table) half
  was separately completed via the extensive filter-builder/sort/
  Apply-button work — the table's fetch is driven by `_state.apiURL`
  (which carries `filter=`), so List view already reflects server-side
  filtering. The original design write-up is kept below (see "Filter
  support in Grid/List views (design)") for historical context only.

- [ ] **Fix `navigateJsonUrl`/`syntaxHighlight`'s link reconstruction gap.**
  Discovered during the above design discussion (2026-07-07), explicitly
  deferred — not required for Grid/List filter support and has no
  observable real-world impact today, but is a genuine robustness gap:
  when a user clicks a URL string value shown inside JSON view's raw
  text, `navigateJsonUrl()` only extracts `path` + `filter` from the
  clicked link and reconstructs everything else via `buildBaseURL()` —
  silently dropping any OTHER query params the link might carry (e.g. a
  hypothetical required `?xreg=true`, or any other future non-filter
  query convention). Confirmed via direct question/answer during design:
  this would genuinely break in that hypothetical scenario. Fix: once
  `stripFilterParams()` exists (see Grid/List filter design below),
  retrofit `navigateJsonUrl`/`syntaxHighlight` to preserve the clicked
  link verbatim as `apiURL` (stripping only `filter=`, same as
  everywhere else) instead of reconstructing from path alone — making
  all 3 views share one single, fully link-preserving mechanism instead
  of JSON's in-content clicks being the one remaining exception.

- [x] **Filter builder UI for JSON view** — implemented and live-tested
  via CDP (chip-based OR-groups, model-driven cascading wizard,
  path-scoped root/resource options, non-destructive click-to-edit with
  Update/Cancel, focus-safe value validation, "(group itself)" resource
  option, breadcrumb-style wizard rendering — see below). Original
  design sketch kept below for reference:

- [x] **Breadcrumb-style wizard rendering (filter builder UX polish).**
  Implemented and live-tested via CDP (2026-07-06). Replaced the old
  always-show-all-levels stacked `<select>` rendering in
  `buildWizardHTML()` with a hybrid breadcrumb: once a level (Group
  type, Resource type, Level, each attr segment) is picked, it collapses
  to plain clickable breadcrumb text (e.g. `endpoints › messages ›
  format`), and only ONE active `<select>` shows for the next undecided
  level. Clicking an earlier breadcrumb crumb (`fbJumpTo(which, idx)`)
  reopens that level's dropdown and truncates everything chosen after
  it. Decision: chose this hybrid over a full inline breadcrumb with
  per-segment popovers — popovers are risky on mobile (small tap
  targets, clipping/positioning issues on narrow viewports); a native
  `<select>` below a plain-text breadcrumb keeps the OS-native picker
  UX and avoids custom popover code while still giving the compact
  "path so far" look. Level gets special handling (`_fbDraft.levelOpen`
  flag) since it silently defaults to "resource" and shouldn't force an
  extra click. New CSS: `.fb-breadcrumb`/`.fb-crumb`/`.fb-crumb-sep`
  with ~32px mobile tap targets and `flex-wrap`.

- [x] **Merged attribute + "step into" picker (replaces separate
  Group/Resource/Level dropdowns, fixes root-level attribute bug).**
  Implemented and live-tested via CDP (real DOM click-through,
  confirmed filter chips add/edit/round-trip correctly). Fixed a real
  bug: `fbRootOptions()` used `value: ''` for BOTH the "-- choose --"
  placeholder AND the "(registry root)" option, so picking "registry
  root" was indistinguishable from no selection — the wizard could
  never let a user filter on a root-level attribute directly. Root
  cause is the same "`''` as both placeholder and legit value"
  anti-pattern that motivated the earlier `FB_SELF` sentinel; this time
  it's fixed structurally instead of with another sentinel.
  - At each of Root / Group-type / Resource-type, a single `<select>`
    now shows BOTH filterable attributes (`<optgroup label="Attributes">`,
    `attr:`-prefixed) AND child collections to step into
    (`<optgroup label="Step into">`, `grp:`/`res:`/`step:meta`/
    `step:versions`-prefixed) — no more forced default and no more
    "(group itself)"/`FB_SELF` special case, since picking a Group's
    own attribute directly is now just as natural as stepping into one
    of its Resource types.
  - "Collection shadow" attributes (e.g. root's `endpoints`/
    `endpointscount`/`endpointsurl`, group's `messages`/`messagescount`/
    `messagesurl`, resource's `meta`/`metaurl`/`versions`/
    `versionscount`/`versionsurl`) are excluded from the "Attributes"
    optgroup since they duplicate the "Step into" choice and are
    confusing to see as filterable leaf attributes
    (`fbCollectionShadowNames()`).
  - Resource-level attribute filtering now merges Version-level
    attributes (`rm.attributes`) into Resource-entity attributes
    (`rm.resourceattributes`), since a Resource's JSON representation
    implicitly inlines its default version — `resourceattributes` wins
    on name collisions (`fbMergeResourceAttrs()`). Choosing a Resource
    type no longer silently forces "Resource" as the level; Meta,
    Versions, and any attribute are equal, undecided choices in one
    merged menu until the user picks one.
  - `FB_SELF`, `fbRootOptions()`, `fbResourceOptions()`,
    `fbLevelOptions()`, and `_fbDraft.levelOpen` were all removed —
    superseded by `fbMergedAttrOptions()`, `fbMergedSelectRow()`,
    `fbApplyAttrChoice()`, `fbSetRootChoice()`/`fbSetGroupChoice()`/
    `fbSetResourceChoice()`. `fbParseExpr()` and `buildWizardHTML()`
    updated to match: `wiz.level` stays `null` (not `'resource'`) when
    unset, and breadcrumb crumbs only render for levels actually
    explicitly decided.

- [x] **Merged-picker/breadcrumb polish + AND-target picker.**
    Implemented and live-tested via CDP. Three changes:
    - `fbMergedSelectRow()`: "Step into" optgroup now renders BEFORE
      "Attributes" (renamed to "SINGULAR Attributes") so the usually-short
      collection list isn't buried below a long attribute list.
    - Breadcrumb click/delete affordance (`fbCrumb()`/`fbBreadcrumbHTML()`):
      crumbs are now built as `{label, onclick}` descriptors and rendered
      in one pass so the LAST crumb (immediately before the active picker)
      can be styled as plain, non-clickable `.fb-crumb-text` — clicking
      crumb text elsewhere means "jump back and re-decide this level",
      which doesn't apply to the level currently being decided. EVERY
      crumb (including the last) also gets a small red "x"
      (`.fb-crumb-x`) as an explicit, unambiguous delete action —
      `fbJumpTo()` itself is unchanged, this is purely a rendering split.
    - "+ Add (AND)" target picker: previously AND always appended to the
      LAST OR-group, so building an AND expression onto an earlier group
      required deleting/redoing everything after it. Added
      `_fbDraft.addTarget` (explicit group index, or `null` = "last
      group"), a `fbAddTargetIndex()` getter (clamps/falls back safely),
      `fbSetAddTarget()` setter, and `fbGroupPreview()` (truncated
      "Group N: expr AND expr…" label). A `<select class="fb-add-target">`
      appears in the add-row — only when there's more than one OR-group
      (unambiguous otherwise) — listing every group's preview, defaulting
      to the last. `fbAdd('and')` now appends to the selected target
      instead of always the last group; `addTarget` resets to `null`
      (back to "last") whenever a group/expr is added or removed, so it
      never points at a stale/out-of-range index.

- [x] **Present/absent for complex attrs; breadcrumb-in-pill; keep-vs-
  delete crumb split; AND-target rename/reposition; full reset on add.**
  Implemented and live-tested via CDP (real DOM click-through). Five
  changes:
  - **Present/absent for complex (object/map/array) attributes.**
    Previously the op/value row only appeared once the segment walk
    reached a scalar leaf, so there was no way to filter on e.g.
    `schemagroups.schema.deprecated` (an object) being present/absent —
    you were forced to keep drilling into its sub-fields. Now,
    whenever the walk breaks at a frontier reached AFTER picking at
    least one segment (`i > 0`) whose type is object/map/array, a
    `fbOpValueRow(true)` (presence-only: "is present"/"is absent", no
    comparison operators or value input) renders alongside the
    continue-drilling picker — the user can either stop here or keep
    going deeper. `fbOpValueRow()` takes a new `presenceOnly` param;
    `buildWizardHTML()` tracks a `canStopHere` flag set at each of the
    3 generic-seg-row break points.
  - **Breadcrumb crumb is now one visual pill (box)**, matching the
    `.fb-chip` style used for filter-expression chips: `.fb-crumb`
    wraps both `.fb-crumb-text` and `.fb-crumb-x` (previously they were
    two separate floating spans). The last crumb's text gets a
    `.fb-crumb-text-static` modifier (no pointer cursor/hover
    underline) since it isn't clickable.
  - **Clicking a crumb's TEXT vs its "x" now do different things.**
    Text click (`fbJumpToStay()`) keeps that level's own choice and
    clears only its descendants (e.g. clicking the "messagegroups" text
    keeps `gPlural` and only clears `rPlural`/`level`/`segs` — lets you
    re-pick a different resource without losing the group). The "x"
    (`fbJumpTo()`, unchanged) deletes that level's own choice AND
    everything after it (cascades up, since descendants depend on it).
    New helpers: `fbClearSegsKeepLevel()` (Level crumb text — keep
    level, clear segs/op/value) and `fbTruncateSegsKeepIdx(idx)` (Seg
    crumb text — keep segs[0..idx], clear deeper + op/value). Group/
    Resource crumb text reuses the next level's existing "clear
    everything after"-setter (`fbSetResource('')`/`fbSetLevel('')`)
    since those never touch the level above them anyway.
  - **"Add to:" dropdown moved after both Add buttons**, with an
    explicit "Add to:" label (`<label class="fb-add-target-label">`),
    and its option text renamed from "Group N: ..." to "Filter N: ..."
    (`fbGroupPreview()`) to match the "Filters" terminology used
    elsewhere in the panel.
  - **Full wizard reset after every `fbAdd()`.** Previously only
    `segs`/`op`/`value` were cleared after adding a filter, keeping
    `gPlural`/`rPlural`/`level` so several AND's against the same
    entity didn't require re-picking it. Per updated feedback, the
    entire wizard (including group/resource/level) now resets via
    `_fbDraft.wiz = emptyWizard()`, so all breadcrumbs clear and the
    next filter starts fresh from the top-level picker.

- [ ] *(superseded by the item above — kept only for historical
  context of the original flat-dropdown sketch)* Filter builder UI
  original sketch:
  - Filters are stored internally as `_state.filters` = an array of OR
    groups, each an array of AND'd expression objects
    `{path, attr, op, value}` — NOT raw strings — serialized to
    `?filter=...` text only at fetch time (`buildAPIURL()`), and parsed
    back from the URL on load for shareable/bookmarkable links.
  - Existing groups render as chip rows: each OR group is a row of
    small removable chips (one per AND'd expression, e.g.
    `endpoints.name = myendpoint ×`), with a row-level `×` to drop the
    whole OR group, and an "or" divider between rows.
  - "Add a filter" builder row below the chips:
    - Group-type dropdown, model-driven via `model.groups` (walking the
      same structure as `buildInlineOptions()`), plus a "(registry
      root)" option for root-level attributes.
    - Resource-type dropdown (shown only if the chosen group has
      resources).
    - Sub-path dropdown for `meta` / `versions` / doc, shown only when
      relevant (mirrors `addResInlines()`'s existing prefix logic).
    - Attribute dropdown, populated from the model's attributes for
      that entity type (spec-defined + model-defined extensions), with
      an "(other / custom)" option revealing a free-text field for map
      keys, `"*"` extension names, or attributes the model can't
      enumerate.
    - When the chosen attribute is a map or array type, offer
      auto-suffix chips for `.* ` (map wildcard), `[*]` (array
      wildcard), `[-1]` (last array item) per the spec's dot-notation
      rules — user picks one or types a literal key.
    - Operator dropdown: present, `=`, `!=`/`<>`, `<`, `<=`, `>`, `>=`,
      absent (`=null`). Value field is freeform text (no enum support
      exists in the attribute model) and hidden/disabled for
      present/absent. A small hint clarifies `*` = wildcard, `\*` =
      literal, and that wildcards/`null` aren't allowed with
      comparison operators (validated client-side to match
      `ParseFilters()`'s rules in `registry/info.go`).
    - "Add (AND)" appends the expression to the currently-open/last OR
      group; "Add (OR)" starts a brand new OR group.
  - A read-only preview line shows the resulting `?filter=...` query
    string live, so the actual syntax is visible/learnable but never
    needs to be hand-typed.
  - Decision: keep a small "Advanced (raw text)" toggle as an escape
    hatch for filter shapes the builder doesn't cover (arbitrary nested
    map keys, escaped wildcards, etc.), in the hopes it can be removed
    later once the builder proves it covers everything in practice.
  - Once proven out for JSON view, revisit whether to surface it in
    Grid/List too (see item above).

- [ ] **Support `ifvalues` in `getAttr()` / `getAttrType()` /
  `getExplicitAttrType()` / `getExplicitAttrTypeAtPath()`**
  (`registry/ui/app.js`, ~line 1868). These functions currently only
  walk the model's static `.attributes`/`.attributes.*` maps. They do
  not evaluate `ifvalues` conditional sibling-attribute rules, so
  attributes that only become defined when a sibling has a specific
  value are invisible to model-driven UI logic (e.g. monospace-type
  lookups, label overrides). Will need a `data` (actual entity JSON)
  parameter threaded through so conditional matches can be evaluated
  alongside the static model walk. There's already a `TODO(ifvalues)`
  comment marking this in the code.

- [ ] **Generic xRegistry JSON pretty-printer (JS + Go)**. The spec
  (`core/spec.md`, "Design: JSON Serialization" section) shows a
  pseudo-JSON layout for a Registry's full JSON body with a specific
  attribute ordering and blank-line placement. Our server already
  produces output matching that shape when *we're* the one streaming
  it (`registry/jsonWriter.go`, driven by `OrderedSpecProps`'s
  `$space`/`$extensions` marker entries — see `registry/shared_entity.go`
  ~line 114). This todo is a different, standalone tool: given
  *arbitrary* xRegistry JSON (e.g. any single entity or a whole
  Registry doc, possibly from a non-xrserver/less-strict server), use
  the entity's `xid` to identify its type/depth, then recursively
  re-order and re-format that JSON — as deep as it goes — to match the
  spec's pseudo-JSON attribute ordering/blank-line/extension-placement
  conventions, purely by rearranging an already-parsed JSON value (not
  by re-deriving it from our DB model). Needed in two forms:
  - A JS version for the SPA UI's JSON view, so JSON from any
    xRegistry-compliant server displays "prettied up" consistently,
    not just ours.
  - A Go version (reusing the same ordering rules/logic conceptually,
    likely built on `OrderedSpecProps`) for use in the `xr` CLI.
  - Needs a design pass before implementation: how to reuse
    `OrderedSpecProps`'s `$space`/`$extensions` ordering metadata as a
    shared source of truth for both the existing streaming writer and
    this new "reformat already-parsed JSON" tool, without duplicating
    the ordering rules in two places.

- [x] ~~Make Resource Grid view (depth 4) more user-friendly (document in
  its own tile; clickable version list).~~ **OBSOLETE / done, no action
  needed** (added and immediately resolved in a later session): Grid
  view was removed entirely for Resource/Version pages in checkpoint 034
  (2026-07-08), so there's no Grid view left to improve. The Document
  tab and version-selector dropdown in the current List-view-based
  Resource/Version page already cover both underlying goals (document
  shown directly, versions individually selectable/clickable).

- [ ] **Tooling to keep the `registry/ui/xreg` static test fixture in
  sync.** Design/update tooling so per-entity `$details` files and
  parent collection `index.html` listings stay in sync automatically,
  avoiding staleness bugs like the missing `shortself`/`format`/`compat`
  attributes seen in a past session. Needs discussion with the user on
  approach before implementing.

- [x] ~~Extend the Filter Builder Apply-button dirty-check beyond
  filters.~~ **DONE** (found already implemented when re-checked
  2026-07-09 — apparently completed in an earlier session without this
  file being updated). `computeApplyDirty()` (`registry/ui/app.js` ~line
  4753) covers filters (draft groups/advanced textarea vs
  `_state.filters`), sort (`sortCollectValue()` vs `_state.sort`),
  options (doc/binary/collections checkboxes vs `_state.docView`/
  `binary`/`collections`, only in the full JSON-view panel), and inlines
  (checked boxes vs `_state.inlines`). All relevant controls
  (checkboxes, inline boxes, sort control, advanced textarea) already
  call `updateApplyButtonState()` on `onchange`/`oninput`, which flips
  `.disabled` on every `.lp-apply` button. Verified via CDP: both Apply
  buttons start disabled, become enabled when the sort draft changes,
  and go back to disabled when reverted.

- [x] ~~Support Capabilities/Capabilities-Offered in List view, with edit
  for Capabilities.~~ **DONE** (2026-07-09). Investigation found Capabilities
  already had a full List view + editing (`renderCapabilitiesEditor`/
  `renderCapEditor`, `registry/ui/app.js` ~8244-8674) from earlier
  session work — only `capabilitiesoffered` needed a List view. Added
  `renderCapabilitiesOfferedViewer()`/`renderCapSchemaNode()` (app.js
  ~8677-8735): a new recursive **read-only** renderer for the
  offered-capabilities schema shape (`{type,attributes}` /
  `{type,enum,item}`), reusing the existing `capSection`/`capObjectBox`/
  `capChipList` CSS/DOM helpers so it looks consistent with the
  Capabilities editor. Wired into `refresh()` and `setDataView()`
  alongside `capabilities`; `defaultDataView()` and `renderHeader()`'s
  view-button-enable logic updated so `capabilitiesoffered` now defaults
  to `table` (List), with Grid disabled, JSON enabled, and Edit always
  disabled (server-declared schema document, never user-editable).
  Verified via CDP screenshots: List view renders all top-level sections
  (`available`, `compatibilities` w/ enum chips, `flags`, `formats`,
  `ignores`, etc.), JSON toggle works, Edit button stays disabled, and
  the registry root page's "Capabilities Offered" nav pill correctly
  links to it — with no regression to the existing Capabilities
  List/edit functionality.

- [ ] **New SPA lacks the old `?ui` HTML view's server-side proxy for
  remote registries (real functional gap, found via analysis
  2026-07-09).** Old UI has a real backend proxy: `GET
  /proxy?host=<remote>&path=<path>` (`registry/httpStuff.go:122`,
  `registry/ui.go:82-100,2919-2930`) — the local xrserver process
  fetches the remote xRegistry server itself and relays it to the
  browser, avoiding CORS entirely. The new `/ui/` SPA's `addServer()`
  (`registry/ui/app.js:154`) only does direct client-side `fetch()` to
  whatever server URL the user adds — confirmed zero references to
  `/proxy` anywhere in `app.js`. Adding a remote registry that doesn't
  set permissive CORS headers will silently fail in the new SPA where
  the old UI would have worked. Needs discussion: should the SPA route
  arbitrary remote-server fetches through `/proxy` transparently (e.g.
  detect cross-origin servers, or fetch failures, and retry via
  `/proxy`)? Everything else compared between the old `?ui` HTML view
  and the new SPA (registry picker dropdown, "open source/commit" link,
  a `<form id=url>` URL bar) was found to be either already replicated
  differently or dead/commented-out code in the old UI — not a real
  functional gap.

## Completed (for history / context)

- Removed dead legacy CSS: `#left`/`#right` id selectors in
  `registry/ui/style.css`'s `@media (max-width: 768px)` block — leftover
  from the old standalone model editor page, no longer matched anything
  in `index.html`/`app.js`. `.editorLeftNav`/`.editorActionBar`/
  `.navToggleBtn` rules in the same block were kept (still used).

- Model/Model Source viewer merged into the new SPA design (List view
  default, Grid disabled, JSON view enabled) — see session
  `538b1546...` checkpoints.
- Capabilities/Capabilities Offered viewer+editor built to match,
  including save/undo, dirty-state guard, cache refresh after save.
- Grid/List view consistency pass: meta box parity, header wording
  parity ("Registry Property", "Group Types", "Default X Version (n)
  Property", etc.), document-view links added to Resource/Version list
  views.
- JSON view navigation stickiness (Endpoints and "← Registry Data" no
  longer drop out of JSON view unexpectedly).
- Deprecated-attribute monospace bug: fixed to be fully model-driven —
  reads the runtime-cached `/model` recursively
  (`getExplicitAttrTypeAtPath()`) rather than hardcoding attribute names
  via codegen. Also fixed to honor concretely-typed wildcards (e.g.
  `"*": {type: "url"}`), not just explicitly-named attrs.

- Filter builder: group-label badge + split "AND" button. Each OR-group
  row (`.fb-group-row`) now shows a small "F1"/"F2"/... badge, absolutely
  positioned overlapping the row's top-left corner (title="" tooltip
  shows the full "Filter N: expr…" preview). The old standalone "Add to:"
  labeled dropdown was replaced with a two-zone "split button": the
  "+ Add (AND)" button (left-aligned text, `flex:1`) plus a compact
  overlay `<select>` on its right edge (`position:relative` on a
  `.fb-and-split` wrapper, `position:absolute`+`z-index` on
  `.fb-and-split-target`) showing only "F1"/"F2"/... (again with a
  `title=""` tooltip per `<option>` for the full preview). Only rendered
  when there's more than one OR-group — single-group case stays a plain
  unambiguous button. "+ Add (OR)" remains a fully separate plain button,
  visually reinforcing that the AND-target picker doesn't apply to it.
  `min-width` on `.fb-and-split` + the pre-existing `flex-wrap` on
  `.fb-add-row` let "+ Add (OR)" drop to its own line on narrow widths
  (verified at 320px viewport via CDP screenshot — badges and split
  button render cleanly, OR button wraps as expected). New helper
  `fbGroupShortLabel(gi)` returns "F"+(gi+1); `fbGroupPreview(gi)` kept
  for the full-text tooltip. Verified via CDP with real DOM
  select-change + button-click events: selecting "F1" from the overlay
  and clicking "+ Add (AND)" correctly appended the new expression to
  the first group (not the last), and "+ Add (OR)" still always creates
  a new group regardless of the AND-target selection.
- Follow-up tweak: `.fb-and-split-target` now insets `top:1px;
  right:1px; bottom:1px` (rather than flush 0) and is `width:37px`
  (up from 30px, to fit double-digit "F10"/"F11"+ labels) so the
  overlay select sits fully inside the AND button's own border rather
  than overlapping/covering it; right-side `border-radius` tightened
  to 3px (1px less than the button's 4px) to nest cleanly in the
  corner. Button's `padding-right` bumped to 40px to match. Verified
  via CDP with 11 OR-groups (F11) that the select's bounding rect is
  fully contained within the button's bounding rect.
- Split-AND button renamed "+ Add (AND)" → "+ Add (AND) to" (only when
  the split target-picker is shown, i.e. >1 group) to make explicit
  that it appends to whichever "F#" is selected.
- Second "Apply" button added right above the "Filters" heading (in
  addition to the existing full-width one at the bottom of the left
  panel), so it's reachable without scrolling past a long filter list.
  Rendered as a thin `<hr>`-style divider line flanking a small pill
  button on both sides (`.lp-divider-apply`/`.lp-divider-line`/
  `.lp-apply-top`), which also centers the button between the two line
  segments (each `flex:1`).
- Fixed a filter-query-param bug in the JSON viewer's URL linkification
  (`syntaxHighlight()`/`navigateJsonUrl()`): when a JSON value is itself
  a same-server URL that already has its own (correctly subsetted, by
  the server) `filter=` query param — e.g. a nested collection's
  `xxxurl` computed relative to the current path — the UI was
  discarding that embedded filter and re-deriving one from the app's
  own currently-active `_state.filters` instead. This made the
  hover-preview href (and, prior to this fix, actual click-navigation
  too) show/use the WRONG filter — the one active at the current level,
  not the one the server had already computed for the target URL. Fixed
  by adding `filtersFromUrl(rawUrl)`, which parses the URL's own query
  string and extracts its `filter=` param(s) — note the server emits
  one repeated `filter=` key per OR-group (`FiltersRelativeToAbstract()`
  in `registry/info.go`), a different wire convention than the app's own
  permalink format (single `filter=`, OR-groups newline-joined) — so
  `filtersFromUrl()` collects all `filter=` occurrences via
  `URLSearchParams.forEach()`. Both the linkified `href` (hover) and the
  `navigateJsonUrl()` click-handler's `pushState()` now use this
  extracted value instead of blending in unrelated local UI state.
  Verified via CDP: `filtersFromUrl()` correctly parses single, multiple
  repeated, and absent `filter=` cases; a JSON string URL value with
  `?filter=deprecated` produces a hover href containing `filter=
  deprecated` (not the app's separately-set current filter), and
  clicking it navigates with the same correct filter.
- Fixed a bug where the persistent Update/Cancel controls (shown while
  editing an existing filter expression) would disappear entirely if
  the user deleted a breadcrumb mid-edit, because they were only ever
  rendered via `fbOpValueRow()`'s add-row — which itself is only
  reached once the wizard is in a complete leaf/op/value state.
  Extracted `fbEditingBarButtons(dis)` (the Update/Cancel markup) and a
  new `fbEditingBar()` (full row + disabled-state computation) and a
  `fbCrumbsWithBanner(crumbs)` helper (breadcrumbs + banner when mid-
  edit) used at all 4 places `buildWizardHTML()` can emit HTML — the 3
  early-return attr-picker rows (root/group/resource) plus a new
  fallback `else if (_fbDraft.editing)` branch at the end (for the
  generic-seg-row case) — so Update/Cancel now stay visible throughout
  an edit regardless of navigation, only disappearing on explicit
  Cancel/Update. `fbEditingBar()`'s Update-disabled check mirrors
  `fbUpdateExpr()`'s own guard exactly (`!path || fbValidate(wiz)`, not
  just `fbValidate` alone) so the button is never left looking clickable
  when clicking it would actually be a no-op (e.g. after breadcrumbs
  were cleared back to an incomplete path). Verified via CDP: editing a
  chip, then simulating a breadcrumb "x" delete (`fbJumpTo('seg', 0)`),
  keeps both buttons visible with Update now disabled (path empty) and
  Cancel fully functional.

## Follow-up: replace "SINGULAR Attributes" placeholder with real name
The merged attribute/collection picker's optgroup label read literally
"SINGULAR Attributes" at every level. Fixed by adding a 4th `singular`
parameter to `fbMergedSelectRow()`, used to build the label
(`esc(singular) + ' Attributes'`) instead of the hardcoded string.
Call sites now pass: `'Registry'` at the root level (matching the
existing `getSingularName()` convention for path length 0); the
group's `gm.singular` (falling back to `gPlural` with trailing "s"
stripped) at the group level; and the resource's `rm.singular` (same
fallback) at the resource level. Verified via CDP at all three levels:
"Registry Attributes", "dir Attributes", "file Attributes".

## Planned: Sort support in the UI
Add the ability to sort list/grid results in the SPA, mirroring the
xRegistry `?sort=` API query parameter. Not yet designed in detail —
placeholder entry to track the request; needs a follow-up design pass
covering: where the sort control lives (per-collection, alongside the
filter builder?), how multi-field/direction sort is expressed in the
UI, and how the resulting `?sort=` param interacts with the existing
filter/breadcrumb URL-building logic.

## Follow-up: bottom Apply button now centered on a divider line
The full-width green "Apply" button at the very bottom of the JSON
left panel is replaced with the same divider-line + centered-button
combo already used above the Filters heading, for visual consistency.
The trailing plain `<hr class="lp-divider">` left by the last rendered
section is stripped before appending the combo so the two divider
lines don't double up. Verified via CDP: bottom of panel now shows a
centered Apply button flanked by divider lines, matching the one above
Filters.

## Follow-up: separator between config-level and user-level Inlines
`buildInlineOptions()` now inserts a `{sep: true}` entry (rendered as
`.lp-sep-line`) between the root-level `capabilities`/`model`/
`modelsource` options and the always-present `* (all)` option, matching
the old UI's separation of server/config-level inline options from
user/data-level ones. Verified via CDP: `.lp-sep-line` appears right
after the "modelsource" checkbox row, before "* (all)".

## Follow-up: "Filter Builder" label above the attribute/collection wizard
Added a small centered "Filter Builder" label (flanked by divider
lines, reusing `.lp-divider-line`) between the filter groups/breadcrumb
area ("No filters yet" or existing chips) and the merged attribute/
collection picker row, so first-time users understand what the
"-- choose --" dropdown below is for. New `.fb-wizard-label` /
`.fb-wizard-label-text` CSS classes. Verified via CDP: label renders
between "No filters yet" and the picker `<select>`. The label's own
divider lines were made dashed (`.fb-wizard-label .lp-divider-line`)
and `.fb-wizard`'s separate dashed top border was removed, so there's
only one separator line (dashed) instead of two stacked ones.

## Follow-up: darker "* (all)" separator + Update/Cancel moved to bottom
Two small tweaks:
- `.lp-sep-line` (the dashed line before "* (all)" in Inlines) was
  hard to see; darkened from `#ddd` to `#999`.
- The persistent Update/Cancel editing banner used to render right
  after the breadcrumbs but *before* the attribute/collection picker
  row in the root/group/resource early-return cases of
  `buildWizardHTML()` — visually out of place since the picker
  normally sits directly below the breadcrumbs. `fbCrumbsWithBanner()`
  now only renders the breadcrumb row; a new `fbTrailingEditingBar()`
  helper renders the Update/Cancel banner and is appended *after* the
  picker row instead, at the bottom of the Filter Builder widget block.
  The other (non-early-return) path in `buildWizardHTML()` already
  appended the banner after its picker/op-value row, so no change was
  needed there. Verified via CDP: simulating an edit at the root level,
  the Update/Cancel row now renders after the `<select class="fb-seg-
  select">` element instead of before it.

## Follow-up: JSON twisty vertical alignment + spacing
The expand/collapse "twisty" triangle (`.jt`) in the JSON view looked
bottom-aligned relative to the line's text instead of vertically
centered. Nudged it up with `position: relative; top: -2px` (kept
`vertical-align: middle` too). Also added `margin-right: 2px` so
there's a small visible gap between the twisty and the JSON text that
follows it. Both rules live on the single shared `.jt` CSS class used
for every collapsible line, so all attributes at any given level stay
consistently aligned. Verified via CDP screenshot: triangles now look
centered against the text baseline with visible gap before the quote.

**Correction**: the initial fix broke column alignment between opener
lines (`.jt`) and non-opener/gutter lines (`.jt-spc`) because the new
`margin-right` was only added to `.jt`, and a follow-up request to
bump `.jt`'s font-size to 16px (without matching `.jt-spc`) would have
made the two elements' `1ch` widths diverge. Fixed by keeping
`font-size: 16px` and `margin-right: 2px` identical on *both* `.jt`
and `.jt-spc` (this mirrors the old `ui.go`/RegHTMLify `.exp`/`.spc`
gutter-column approach, which also kept the toggle and spacer classes'
sizing in lockstep). Verified via CDP screenshot: all `"..."` quotes
now line up in the same column whether or not the line has a twisty.

**Second correction (the real root cause)**: the previous fix only
addressed the *outer*, always-present `.jt-spc` gutter (the very first
column on every line) — it did not address the actual bug, which is
that `addTwisties()` only ever replaces the *last native indent space*
with the twisty span on opener lines (e.g. `"endpoints": {`); plain
attribute lines (e.g. `"endpointsurl": "..."`) keep their full,
untouched native indentation. Since `.jt`'s box (width + margin, at
16px font) is wider than the single monospace indent-space character
it replaces, opener lines ended up visibly indented further right than
their non-opener siblings at the same JSON nesting depth — exactly
what was reported ("endpoints" vs "endpointsurl" not aligning).
Fixed by introducing a new `.jt-slot` class: an invisible placeholder
with the *exact same* box (`width: 1ch; margin-right: 2px; font-size:
16px`) as `.jt`, now also substituted for the last native indent space
on every plain (non-opener, non-closer) attribute line, exactly the
way `.jt` already did for openers. Both variants now consume identical
box width regardless of glyph content, so every line at a given
nesting depth reduces its indent by the same amount and stays aligned,
independent of the real monospace character metrics (no more reliance
on `.jt`'s box happening to match one real space's rendered width).
`.jt-spc` (the true, always-present *first* gutter column) was
reverted to a simple, unrelated rule (`width: 1ch; font-size: 13px`,
matching the JSON container's own font) since it no longer needs to
mirror `.jt`'s sizing — the alignment fix lives entirely in the new
`.jt`/`.jt-slot` pairing. Verified via CDP: simulated JSON with
`endpointsurl`/`endpointscount`/`endpoints` siblings, plus the nested
`available.{capabilities,capabilitiesoffered,entities,export}` block
in `/capabilities` — all quotes align at every depth, both twisty and
non-twisty rows.

**Third correction (the actual final fix — the `.jt-slot` approach was
itself wrong)**: the previous "correction" was reported by the user as
introducing two *new* regressions, discovered by copy-pasting the JSON
output into `vim` (not just eyeballing it on screen): (1) top-level
plain attributes were now indented only 1 space instead of 2 — because
`.jt-slot` *removed* one real native indent-space character from every
plain line and replaced it with a placeholder that has no text content
at all, so plain lines lost a real character of indentation; (2) the
closing `}`/`]` lines (never touched by any of this logic — they always
kept their full native indent) now looked "indented one char too much"
purely *relative to* the now-under-indented plain lines above them.
Root design flaw: trying to keep opener/plain lines aligned by forcing
*both* to consume an identical, glyph-sized box was the wrong lever —
it changes the actual indentation depth of plain lines, which must stay
exactly as `JSON.stringify` produced it (both for on-screen alignment
*and*, importantly, for copy/paste fidelity into a plain-text editor).
The real fix has two independent parts:
- **Reverted the plain-line `.jt-slot` substitution entirely** — plain
  (non-opener, non-closer) lines are back to keeping their full,
  untouched native indentation, exactly like closer lines. `.jt-slot`
  was removed as dead code.
- **Made `.jt`'s own box exactly 1 native character wide** instead of
  trying to make plain lines match a bigger glyph box. `.jt`'s
  `font-size` was set back to the container's own 13px (so its `1ch`
  width is *identical* to the one real indent-space character it
  replaces — this is the actual constraint for opener/plain-sibling
  alignment, both on screen and when copied). The bigger, more legible
  16px glyph is now rendered by a *nested* `<span class="jt-glyph">`
  inside `.jt` — since `.jt` has `overflow: visible`, the larger glyph
  can visually spill outside `.jt`'s narrow box without changing `.jt`'s
  own contribution to the line's layout width, so alignment is
  unaffected by how big the visible glyph is drawn. The small visual
  gap before the JSON text (previously attempted via `margin-right`,
  which had no effect since there was no sibling *inside* `.jt` to push
  away from) is now created via `left: -4px` on `.jt-glyph` (shifts the
  glyph's rendered position left, inside `.jt`'s box, freeing up ~2px
  of visual space before the following text — verified empirically via
  `getBoundingClientRect()` gap measurement, ~2.17px).
- **Copy/paste fidelity** (new fix, matching the old `ui.go`/
  `RegHTMLify` `.hide`-span trick): `.jt` (and its `.jt-glyph` child) are
  `user-select: none`, so the glyph character itself is *never* included
  in a copy/paste — confirmed via `Selection.toString()` (which, unlike
  `innerText`, actually respects `user-select: none` for programmatic
  selections, e.g. from a Ctrl-A handler). Because removing the glyph
  from the copy stream would leave opener lines one real character
  short (breaking indentation in pasted text), a new `.jt-copysp` span —
  zero-width (`width: 0`), *not* `user-select: none`, and critically
  **not** `overflow: hidden` (an earlier attempt with `overflow: hidden`
  caused Chromium to drop the span's text from `Selection.toString()`
  entirely, re-losing the character) — contributes exactly one real
  space character back into the copy stream, invisibly. Net result:
  copy-pasted JSON has byte-identical indentation to what
  `JSON.stringify(data, null, 2)` produced, verified by selecting the
  entire `#json-output` element's contents via `Range.selectNodeContents`
  + `Selection.toString()` and `JSON.parse()`-validating the result, for
  both object- and array-nested test payloads (`{a, list:[...], b:{c:{d}},
  z}`), across multiple depths — indentation was `[0,2,2,4,4,4,6,6,4,2,2,
  4,6,4,2,2,0]`, exactly matching native `JSON.stringify` output.
- **Ctrl-A scoping** (new, requested alongside the above): a global
  `keydown` listener (added once, at module scope, near the other
  top-level `window.addEventListener` calls) intercepts Ctrl/Cmd+A only
  when focus is inside `#json-output` (which now has `tabindex="0"` so
  it can receive focus when clicked) and not inside an INPUT/TEXTAREA/
  SELECT; it calls `preventDefault()` and manually builds a `Range` via
  `range.selectNodeContents(el)` + `getSelection().addRange()`, mirroring
  the old `ui.go` `dokeydown()` pattern — this scopes "select all" to
  just the JSON text instead of the whole page. Verified via CDP: after
  focusing `#json-output` and dispatching a synthetic Ctrl-A `keydown`,
  the resulting selection was exactly the JSON text (261 chars, parses
  as valid JSON, does not include page-chrome text like "Registries").
- `jsonToggle()`/`jsonToggleAll()` were updated to swap the inner
  `.jt-glyph`'s `innerHTML` (▶/▼) instead of the outer `.jt` element's,
  since `.jt` now always wraps the glyph in a nested span.

## Left-panel space savings (Registry Endpoints + Filters) — done

Two changes to reduce wasted vertical space at the top of the left
panel at the root of the registry, before "Options"/"Inlines":

1. **Merge Model/Model Source and Capabilities/Capabilities Offered
   onto one line each**, matching the old `ui.go` pattern (`Model
   (Source)`, `Capabilities (Offered)`) instead of 4 separate stacked
   `.lp-nav-item` rows in "Registry Endpoints". Each combined row has
   the main label as one clickable nav item and the parenthetical
   sub-label as a second, smaller clickable nav item on the same line;
   only the currently-active one is highlighted (`.lp-nav-active`).
   The sub-label is only shown when that sub-endpoint is actually
   available (matches today's per-item availability check) — if only
   the main endpoint is available, the row degrades to just the main
   label with no parens. `Export` and `← Registry Data` remain their
   own separate lines (unchanged).
2. **Collapsible "Filters" section** with a `(N)` count of currently-
   defined filter expressions (summed across all OR-groups) right
   after the "Filters" title, and a twisty (▶/▼) just after that (not
   pushed to the far right of the row — sits close to the label) even
   while collapsed, so you can see at a glance whether/how many
   filters are set without expanding. Defaults to **collapsed** on
   every page load (not persisted across reloads/sessions — always
   starts collapsed) but stays expanded/collapsed as toggled during
   the current session/navigation. The "Apply" divider-button combo
   above the Filters title is unaffected by collapse state (still
   shown, since it applies whatever filters are currently set,
   collapsed or not). "Options" and "Inlines" section headers are
   *not* getting the same twisty treatment for now — Filters only.

**Implementation**: `lpNavPairRow()` (new helper, `app.js`) builds each
combined endpoint row; `fbFilterCount()` sums leaf expressions across
`_fbDraft.groups`; `fbFiltersTitleHTML()` renders the twisty/count
title (shared by both the full left-panel render and `fbRerender()`'s
partial re-render after chip edits); `fbToggleCollapsed()` flips the
module-level `_filtersCollapsed` (default `true`) and re-renders.
New CSS: `.lp-nav-row`/`.lp-nav-inline`/`.lp-nav-sub` (combined nav
rows), `.lp-title-collapsible`/`.lp-title-twisty`/`.lp-title-count`
(Filters header). Verified via CDP: screenshots confirm "Model
(Source)" / "Capabilities (Offered)" render on one line each with only
the exact active section bolded/highlighted (e.g. navigating to
`modelsource` highlights "Source", not "Model"); Filters starts
collapsed showing `▶Filters` (no count when empty) and `▶Filters (3)`
once filters are set, expands to the full builder on click showing
`▼Filters (3)`, and toggles back to collapsed correctly.

## Config page: JSON coloring tri-state option — done

Added a tri-state option to the Config page's "Options" section,
`_opts.jsonColorMode` (persisted, default `'full'`):
- **Full color** (today's default) — keys/strings/numbers/booleans/
  links each keep their own distinct color.
- **Minimal color** — everything is black except linkified URL
  values (links keep their color/underline).
- **No color** — everything is black, including links (the dotted/
  solid underline still shows so links remain identifiable/clickable).

Implementation: `optJsonColorMode()` reads the option;
`applyJsonColorMode()` reflects it onto `<body data-json-color="...">`
(called on `init()` and whenever the option changes via
`cfgSetJsonColor()`); CSS overrides in `style.css` are scoped under
`body[data-json-color="minimal"]` / `="none"` and simply force the
existing `.json-key`/`.json-str`/`.json-num`/`.json-bool`/`.json-null`
(and, for "none" only, `.json-url`) colors to black — no changes to
`syntaxHighlight()` itself were needed. UI: three radio buttons
(`cfgJsonColorRadio()` helper) under a new "JSON coloring" row in the
Options section, each with a `title` tooltip describing the mode.
Verified via CDP: took screenshots of the JSON view in all 3 modes
confirming correct coloring in each; confirmed the choice persists
across a full page reload (localStorage); confirmed the Config page's
radio buttons correctly reflect the current selection after reload.

### Follow-up: Options section grid layout + boolean tri-state text centering

Two follow-up fixes requested after the above:

1. **Config page Options alignment.** The original `.cfg-option-row`
   was a `flex` row with `align-items: baseline`, so the checkbox row
   ("Click to copy") and the radio-group row ("JSON coloring") didn't
   line up — labels/descriptions started at different x-positions and
   spacing was inconsistent. First tried a 3-column grid (checkbox,
   label, description all on one line), but that still felt awkward.
   Settled on a cleaner 2-row-per-option layout instead: `.cfg-option-row`
   is a 2-column CSS Grid (`grid-template-columns: 150px 1fr`) — the
   label sits in column 1 (fixed width, so every row's label starts at
   the same x), the editable control(s) (checkbox or radio set) sit in
   column 2 on that same first grid row (so all controls line up at the
   same x too, regardless of label length), and the one-line
   description spans both columns on the row below
   (`grid-column: 1 / 3; grid-row: 2`). A thin `border-top` between
   `.cfg-option-row + .cfg-option-row` separates each option block.
2. **Boolean tri-state (`true`/`false`/`—`) segmented-button text
   vertical centering**, seen when editing Model Source attribute
   options (Immutable/Required/etc.). Root cause: `.boolSeg` is 28px
   tall with a 1px border (border-box), so its actual content-box
   height is 26px, but `.boolSegBtn` used `line-height: 28px` to
   center text — a 2px mismatch between the line-box and the real
   button height that (depending on font metrics) can visibly push
   the text off-center. Fixed by dropping the `line-height` trick in
   favor of `display: flex; align-items: center; justify-content:
   center` directly on `.boolSegBtn`, which centers correctly
   regardless of exact pixel height or font ascent/descent — a more
   robust fix than manually adding `align-self: end` per option cell.
   Also removed the now-redundant mobile-breakpoint `line-height: 18px`
   override (the flex centering makes it unnecessary at any size).
   Verified via CDP: measured the button/text bounding boxes before
   and after (both were already sub-pixel centered in the test
   environment's font, but the fix eliminates the underlying 2px
   line-height/content-box mismatch that can cause visible drift with
   other fonts/OSes) and confirmed no visual regression via screenshot.

## Sort Flag support (JSON view)

Added `?sort=<ATTRIBUTE>[=asc|desc]` support to the SPA's JSON view, per
the spec's "Sort Flag" section. Modeled closely on the Filter Builder
(`registry/ui/app.js`), reusing its model-driven attribute-enumeration
helpers rather than duplicating any model-walking logic, but intentionally
much simpler since sort allows only a single attribute + order (no AND/OR
groups, no comparison operator, and — per spec — no drilling into a
nested child collection).

- `_state.sort` — a string holding the wire-format value verbatim (e.g.
  `''`, `'name'`, `'labels.stage=desc'`), threaded through
  `loadStateFromURL()`/`buildURL()`/`buildAPIURL()`/`pushStateReal()`'s
  default-reset object exactly like `_state.filters`.
- Gated in `renderJSONLeftPanel()` by `hasF('sort') &&
  isCollection(_state.path)` — only shown for Group/Resource/Version
  collection pages, matching the spec's `sort_noncollection` restriction.
- `_sortDraft` — working draft `{mode, attr, mapKey, custom, desc}`,
  keyed per server/section/path (`sortKey()`, mirrors `fbKey()`),
  rebuilt from `_state.sort` via `sortDraftFromPath()` whenever the key
  changes (so browser back/forward and page reloads restore the picker
  correctly).
- Attribute picker reuses `fbRootContext(model, {})` — already computes
  exactly the attribute map of whatever collection is currently being
  browsed (Group/Resource/Version, based purely on `_state.path` via
  `fbPathAnchor()`), so no new model-traversal code was needed. Options
  are built by a new `sortAttrOptions()` (adapted from `fbAttrOptions()`)
  that keeps only `leaf` and `map`-kind attributes (via `fbAttrKind()`),
  excluding `object`/`array` entirely, plus a trailing "(other / custom
  attribute)" freeform escape hatch — same UX pattern as the Filter
  Builder's attribute picker.
- New `sortShadowNames()` excludes a Group-collection's child-resource
  shadow attrs (`{plural}`/`{plural}count`/`{plural}url`, e.g.
  `messages`/`messagescount`/`messagesurl`) from the picker at the
  Group-collection level (depth 1) — sort must not target a nested
  collection even though it appears as a `map`-typed attribute on the
  parent. Resource/Version levels don't need this: `fbRootContext()`
  already excludes the meta/versions shadow attrs for Resources, and
  Versions have no children.
- Map attributes (e.g. `labels`) reveal a follow-up "key name" text
  input (`sortSetMapKey()`), producing a `labels.<key>` wire path —
  same sub-flow style as the Filter Builder's map-key input. The
  "(other / custom attribute)" choice reveals a freeform dot-path input
  instead (`sortSetCustom()`), for anything not directly enumerable.
- Kept to one line ("Sort:" label + attribute dropdown, reusing
  `.fb-seg-label`/`.fb-seg-select`) until an attribute is actually chosen
  — no twisty/collapse needed, unlike Filters, since Sort only ever has
  one control (a multi-expression collapsible section didn't make sense
  here). Once chosen, the map-key/order/clear rows appear below.
- Order toggle is a 2-state Asc/Desc control reusing the existing
  `.boolSeg`/`.boolSegBtn` widget/CSS as-is (no new CSS needed) — only
  shown once a usable attribute path has been chosen. A "Clear sort"
  text link (new `.sort-clear-btn` style) below it clears the sort
  entirely — an explicit "✕" next to the asc/desc pill was tried first
  but read as clearing just the order, not the whole sort, so it was
  replaced with a separate labeled link on its own row.
- Draft isn't committed to `_state.sort` until the existing shared
  "Apply" button is clicked — `applyJSONOptions()` gained one field,
  `sort: sortCollectValue()`, alongside the existing filters/inlines/etc.
- Explicitly out of scope for this pass (documented, not overlooked):
  Grid/List (Table/Tile) view sorting (a separate follow-up, analogous
  to the existing `grid-list-filters` todo) and special `bad_sort`
  error UI (falls through to the existing generic JSON-fetch error
  banner).

Verified via a CDP-driven headless-Chromium script against a temporary
test model (a `endpoints` group with a `labels` map attribute and a
`messages` resource type, two sample `endpoints` entities with different
`labels.stage` values):
- Sort section renders only on collection pages (absent at the registry
  root and on a single-entity page); present at both the Group-collection
  (`/endpoints`) and Resource-collection (`/endpoints/e1/messages`)
  levels, with the correct attribute set at each (shadow names like
  `messages`/`messagescount`/`messagesurl` correctly excluded at the
  Group level; `meta`/`versions` shadow attrs correctly excluded at the
  Resource level, matching the Filter Builder's existing behavior).
- Picking a map attribute (`labels`) revealed the key-name input;
  typing `stage` + choosing "desc" + clicking Apply produced
  `_state.sort === 'labels.stage=desc'`, a correctly-encoded API URL
  (`?sort=labels.stage%3Ddesc`), and a bookmarkable page URL with the
  same `sort` param.
- A real GET against `/endpoints?sort=labels.stage=desc` (and the
  percent-encoded equivalent — both parse identically server-side)
  returned entities in the expected descending order.
- Reloading the page with `&sort=labels.stage%3Ddesc` in the URL
  correctly restored the picker's attribute/key/order selections from
  `_state.sort`.
- Clicking the "✕" clear control + Apply correctly emptied `_state.sort`
  and removed the `sort` param from the URL.
- Test data (sample model + entities) was fully removed/restored to the
  original empty state afterward.

## Entity Details box: spec-attribute grid alignment — done

`renderEntityGrid()`'s "<Type> Details" box and the Resource/Version "Meta"
box (`renderMetaContent()`) render spec-defined attributes as label/value
rows. These were originally independent flex rows (`.eg-row`), each sized
to its own content, so any row with a different `gap` override (e.g.
`.eg-technical`'s tighter button spacing, `.eg-labels`' tighter list
spacing) silently shifted that row's value out of alignment with the rest
— reported as "Epoch's value is slightly to the left of the other values"
(and Labels had the same issue).

Fixed by converting `.eg-spec-rows` (the wrapper around all spec-attribute
rows, added in the previous session) into a real two-column CSS Grid
(`grid-template-columns: max-content 1fr`) instead of independent flex
rows with a shared `min-width` hack:
- Each direct child (`.eg-row` / `.eg-ext-complex`) gets `display:
  contents`, so its own children (label + value) become the actual grid
  items placed into column 1 / column 2 — this guarantees alignment
  regardless of any row's internal gap/spacing needs.
- This requires every row to contribute *exactly* two top-level children.
  Rows built via the `row(label, value)` helper already did. The one
  exception was the Epoch/Self/ShortSelf/XID "technical" row, which had
  up to 5 top-level children (label + epoch value + 3 button elements) —
  refactored to always emit exactly two: a (possibly empty, when epoch is
  absent) label span and a single `.eg-tech-value` wrapper span containing
  the epoch value + all pill buttons, with its own internal `gap: 6px
  12px` for compact button spacing without affecting grid-column
  placement.
- Complex/nested spec attributes (objects like `deprecated`, which has
  `effective`/`removal`/`documentation` sub-fields) previously stacked the
  key above an indented, left-bordered body. Per explicit request, these
  now instead sit in the same two-column grid as every other row (key in
  column 1, nested tree in column 2), and the outermost connecting
  vertical line (`.eg-ext-complex-body`'s `border-left`) is dropped since
  the body is no longer visually "attached" below the key — deeper
  nesting levels inside the tree (`.vt-kv-block > .vt-obj/.vt-arr`) keep
  their own border unaffected.
- Unknown extension attributes (below the `.eg-ext-sep` `<hr>`) are
  deliberately excluded from the grid and keep their original stacked
  flex-row / bordered-body layout — scoped via `.eg-spec-rows` ancestor
  selectors so no JS branching was needed for the different treatment.

Verified via CDP/screenshots: Epoch, Labels, and all other spec rows now
align in the same column; a `deprecated` meta attribute renders with its
label in column 1 and its nested key/value tree in column 2 with no
outer border line; an unknown extension attribute below the separator
still renders in its original (unaligned, bordered) style. Test data
cleaned up afterward.

## Entity Details box: independent extension grid + recursive nested value-tree grids — done

Follow-up to the section above. Two remaining asks:
1. Give unknown extension attributes (below the `.eg-ext-sep` `<hr>`) the
   same column-aligned grid treatment as spec attributes, but as a
   *separate* grid instance so its column-1 width (driven by extension
   attribute name lengths) is independent of the spec section's.
2. For complex (object/map) attribute values shown in column 2 — both
   spec-level (e.g. `deprecated`) and extension-level — recursively apply
   the same two-column grid treatment at every nesting level, so a
   multi-level nested object (e.g. `extraconfig.backoff.{initialms,
   maxms, jitter}`) has each level's keys aligned within their own scope,
   with nesting shown purely via column indentation (no connecting
   border lines needed).

Implementation:
- Generalized the grid mechanics from a `.eg-spec-rows`-only rule into a
  shared `.eg-attr-grid` class (`display: grid; grid-template-columns:
  max-content 1fr; ...` + `> .eg-row, > .eg-ext-complex { display:
  contents; }`). The spec wrapper is now `.eg-spec-rows.eg-attr-grid`;
  extension attributes (previously rendered unwrapped) are now wrapped in
  a new `.eg-ext-rows.eg-attr-grid` container in both `renderEntityGrid()`
  and `renderMetaContent()`. Since each is its own grid formatting
  context, `max-content` column widths are computed independently per
  section.
- `.eg-ext-complex-key`/`.eg-ext-complex-body` (used for any complex
  attribute, spec or extension) now unconditionally use `grid-column: 1`
  / `grid-column: 2` with no border-left — no more default
  stacked/bordered fallback, since every call site is now inside some
  `.eg-attr-grid`.
- `renderValueTree()` (`app.js`) rewritten so each object/map level
  renders as its own self-contained grid: `.vt-obj { display: grid;
  grid-template-columns: max-content 1fr; ... }`, with each `.vt-kv`/
  `.vt-kv-block` row set to `display: contents` and exactly two children
  — a `.vt-key` label span and a new `.vt-kv-value` wrapper span holding
  the (possibly recursive) value. A nested object's `.vt-obj` grid lives
  inside its parent's `.vt-kv-value` cell, so it's a fresh, independently
  sized grid — this is what makes multi-level nesting "just work" without
  any manual depth/indent bookkeeping. The old manual `depth`-based
  `margin-left` indent and the `.vt-kv-block > .vt-obj/.vt-arr
  border-left` connecting line were both dropped for objects (arrays are
  unchanged — still flex-based with their own indent, since the request
  was specifically about "complex objects... obj/map attribute").

Verified via CDP/screenshots with a temporary `endpoints` model: an
extension attribute `extraconfig` containing a nested `backoff` object
(itself containing `initialms`/`maxms`/`jitter`) renders with `backoff:`
and its sibling `retrylimit:` aligned in one column, and `initialms`/
`maxms`/`jitter` aligned in their own nested column one level in — with
no connecting border lines at any level. The extension section's column-1
width (short names like `customfield`/`extraconfig`) is visibly narrower
than and independent of the spec section's column-1 width (longer names
like `documentation`/`endpointid`). The spec-level `deprecated` meta
attribute (with `effective`/`removal`/`documentation`) still renders
correctly. Test data cleaned up afterward.

## Value-tree array indent bug + List view missing complex attrs — done

Two related fixes found while reviewing the HardCoded (`registry/ui/xreg/`) test
registry's `extobj.attrObj` (which has sibling `nestedStr`/`nestedArr`/`nestedObj`
attrs):

1. **Array values misaligned vs. sibling object/string values.** After the
   `.vt-obj` grid conversion, objects no longer use a manual depth-based
   indent (`margin-left: depth*14px`) — alignment is purely via the grid
   column. But `.vt-arr-item` still had the old manual indent left over,
   so an array-valued sibling (e.g. `nestedArr`) rendered its `[0]/[1]/[2]`
   items shifted further right than a same-level object-valued sibling
   (e.g. `nestedObj`)'s nested grid, even though both keys' *labels*
   aligned correctly in column 1. Fixed by dropping the indent style from
   `.vt-arr-item` entirely — verified via CDP `getBoundingClientRect()`
   that `nestedArr`'s value, `nestedObj`'s value, and `nestedStr`'s value
   all now start at the same x position (all children of the same
   `attrObj` value-tree grid).
2. **List view (`renderSingleEntity()`) silently dropped every
   object/array-valued attribute** (labels, extension maps/arrays/objects,
   even the spec `deprecated`-style values) — the scalar-property filter
   excluded anything with `typeof === 'object'` and never rendered a
   fallback for it. Fixed by including those keys in the same Property
   table and rendering their value cell with `renderValueTree()` (the
   same nested-grid renderer Grid view's extension rows use), giving a
   `.cell-tree` class to that `<td>` to undo the table's default
   nowrap/ellipsis/max-width truncation (only appropriate for plain
   scalar text). Verified via CDP screenshot on the HardCoded registry's
   List view: `extarray`, `extarrayobj`, `extmap`, `extobj`, and `labels`
   (previously entirely missing from the page) now render correctly with
   the same nested-tree layout as Grid view.
3. **List view's scalar values didn't follow the normal-vs-monospace
   convention** used everywhere else (Grid view's `renderAttrRow()`,
   `renderValueTree()`'s own leaves) — every scalar was plain escaped
   text regardless of type. Fixed by applying the same decision in
   `renderSingleEntity()`'s Property table: monospace (via
   `copyableMonospace()`) if the attr is a spec attr in `MONO_ATTRS` for
   the current entity level, or if it's an explicitly model-defined
   (non-wildcard) attr with a non-string type; otherwise normal prose
   text (via `copyable()`), matching Grid view exactly (including
   click-to-copy). Verified via CDP screenshot: `specversion`, `epoch`,
   `createdat`, `modifiedat`, `documentation`, `icon` render monospace;
   `name`/`description` remain normal text — same as Grid view.

## Collection views: Created/Modified timestamps — done

Added Created/Modified display to both collection views (Grid tile view
`renderTileView()` and Table/List view `renderTableView()`), formatted via
a new shared `formatTimestamp()` helper as `MM/DD/YYYY hh:mm:ss AM/PM TZ`
(e.g. `07/06/2026 07:22:30 PM EDT`) in the browser's local timezone, built
from `Intl.DateTimeFormat(...).formatToParts()` for cross-browser TZ
abbreviation support.
- Tile view: a `.tile-times` block at the bottom of each tile, right
  aligned, small/muted text, with a thin top border separating it from
  the tile body / resource-pill footer above.
- Table view: two new sortable columns ("Created"/"Modified", `.
  cell-timestamp`), added after the existing Document column, using the
  existing generic string-sort (ISO timestamps sort correctly lexically,
  no special-casing needed in `sortBy()`).

Verified via CDP screenshot on the HardCoded registry's `dirs` collection
in both views.

## Collection views: clickable nested-collection pills — done

On collection views (Grid tile view + Table/List view), the resource-pill
footer/column showing a tile/row's own nested collections (e.g. "files
(2)" shown on group "d1" while viewing the "dirs" collection) is now
clickable and navigates straight into that nested collection
(`dirs/d1/files`) instead of requiring a click on the tile/row first (to
land on `dirs/d1`) and then a second click on the resources list there.

Implementation: new `navigateToNestedColl(itemId, plural)` (in `app.js`,
next to `navigateTo()`) pushes `_state.path.concat([itemId, plural])`.
Each pill's `onclick` calls `event.stopPropagation()` first so the
tile/row's own `navigateTo(id)` handler doesn't also fire. A new
`.coll-tile-res-pill-clickable` CSS class (cursor: pointer + hover
highlight) is applied only to these navigable pills — kept separate from
the base `.coll-tile-res-pill` class, which is also used for the
non-clickable "Resource Types" list shown on the Registry root's Group
Type tiles (model schema names, not actual navigable entities).

Verified via CDP: clicking the "files" pill on `d1`'s tile (Grid view)
and row (List view) while viewing `dirs` both navigate directly to
`dirs/d1/files`.

## Resource collection views: show default version id — done

On the Resources collection view (a group's list of resources, e.g.
`dirs/d1/files`), both Grid and Table/List view now surface each
resource's default version id (`item.versionid`, already present on the
flattened resource entity):
- Grid tile view: a new "Version: `<versionid>`" pill shown before the
  existing "versions: N" count pill in the tile's footer.
- Table/List view: a new "Version" column inserted before the existing
  "Versions" column, populated with `item.versionid`. The "Versions"
  column's own display was simplified from a `plural (count)` pill
  (redundant with the new column header) to just the bare count — still
  clickable (`.cell-version-count`, styled as link-like text rather than
  a pill) to navigate into that resource's versions collection via
  `navigateToNestedColl()`. Both the "Versions" header and its count cells
  are centered (`.col-center`) rather than left-aligned like most columns,
  since a single bare number reads better centered under its header.

Note: the Group collection view's "Resources" column/footer (potentially
multiple resource *types* per group, e.g. "files (2)") is unaffected and
still shows the `plural (count)` pill form, since there a plain count
alone wouldn't identify which resource type it refers to — only the
single-resource-type "Versions" column was simplified.

Verified via CDP screenshot on `dirs/d1/files` in both views.

## Config page: Reset (clear browser-side state) — done

Added a "Reset" section to the Config page (`renderConfig()`), below
"Registry Servers" and "Options", so a user can easily recover if
something looks wrong client-side, without affecting any registry
server. Two choices, both behind a `window.confirm()` guard since
they're destructive and irreversible:
- **Clear All** (`cfgResetAll()`, styled as a danger button): removes
  both `xreg-servers` (`LS_SERVERS`) and `xreg-options` (`LS_OPTIONS`)
  from `localStorage`, then does a full `window.location.reload()`.
- **Clear All Except Registry Locations** (`cfgResetExceptServers()`):
  removes only `xreg-options`, keeping saved server URLs, then reloads.

All of this app's browser-side state lives in exactly those two
localStorage keys plus a handful of in-memory JS caches
(`_labelCache`/`_modelCache`/`_capCache`/`_offeredCache`/etc.) that are
lazily rebuilt on next use — a full page reload after clearing
localStorage is sufficient to reset everything, so there was no need to
individually track/clear each in-memory cache.

Verified via CDP: added a test server + toggled an option, then
confirmed "Clear All Except Registry Locations" kept the server but
reset the option, and "Clear All" wiped both localStorage keys entirely.

## Link-driven navigation (stop constructing URLs) — done

Per xRegistry's core design principle — clients should never need to
construct or parse URLs for simple hierarchy traversal — replaced the
UI's client-side URL construction (`buildAPIURL()` assembling
`serverBase() + path.join('/')` from `_state.path`) with a link-driven
model that follows real server-provided hypermedia fields (`self`,
`<plural>url`, `versionsurl`, `metaurl`, `defaultversionurl`) at every
navigation step.

Key changes (`registry/ui/app.js`):
- `_state.apiURL` — the real resolved URL for the current data page;
  persisted in the address bar (`apiurl=` query param) alongside the
  existing `path=` (kept only for breadcrumb labels/model lookups, never
  used to construct fetch URLs anymore). `refresh()`/`buildAPIURL()` use
  `_state.apiURL` directly when present, falling back to plain
  construction (`buildBaseURL()`) only when unknown (old bookmarks, or
  the 4 intentionally-unlinked sections below).
- `_state.crumbURLs` — session-only cache of the real URL used at each
  visited path depth, populated by every forward link-follow. Breadcrumb
  clicks to a previously-visited ancestor reuse the cached URL for free;
  `pushStateReal()`'s default-apiURL-from-crumbURLs logic makes this
  "just work" for any path-shortening `pushState` call (breadcrumbs, the
  meta-page-redirect, etc.) without each call site needing special code.
- Navigation functions (`navigateTo`, `navigateToNestedColl`,
  `navigateToVersion(ById)`, `navigateToParentResource`) now take/derive
  a real URL instead of reconstructing one from `path` + a new segment.
  `versionURLById()` prefers, in order: exact `defaultversionurl` match,
  `versionsurl + '/' + id`, cached crumbURLs, and only then falls back to
  construction.
- `itemNavKey()` now prefers `item.__mapKey` (the collection's own map
  key — literally the entity's `<singular>id`, since collections are
  keyed by it per spec) over splitting `xid`, eliminating unnecessary
  string parsing.
- All click-target call sites (tile/list views, collections table,
  home-page flat listings) updated to pass the real link URL already
  present in the fetched JSON rather than reconstructing one.

Intentional exceptions (still use fixed-suffix construction off the
server base, unchanged): `/model`, `/modelsource`, `/capabilities`,
`/capabilitiesoffered`, `/export` — these are deliberately not linked
from the registry root (avoids cluttering the main data doc; treated
like a well-known-suffix convention). Confirmed with user.

Cold-deep-link edge case (confirmed with user): on a fresh page load
(bookmark/refresh), there's no prior session history, so `crumbURLs` is
empty. Clicking an ancestor breadcrumb that was never visited this
session falls back to trimming the *current* resolved URL (equivalent
to `buildBaseURL()` for the truncated path) — still a form of
construction, accepted as a rare-case trade-off. Known limitation: any
filter context applied higher up the original (pre-reload) traversal
chain is lost for a trimmed ancestor URL in this specific case.

Two spec characteristics worth noting (neither is actually a problem for
the UI — confirmed both are universal properties of the hierarchy, not
version/resource-specific gaps, and our design already handles them
generically):
- No entity anywhere links back to its parent (not just versions →
  resource). Handled uniformly at every depth by `crumbURLs` (reuse the
  real URL if this ancestor was visited this session) plus the accepted
  trim fallback when it wasn't — the same single mechanism regardless of
  which level you're going "up" from.
- No collection (versions, resource types, group types, ...) is inlined
  by default (`?inline=` is an opt-in optimization only). Not an issue
  because the UI never assumed an inlined map anywhere — `findCollectionRefs()`
  only reads `<plural>url`/`<plural>count` to show a tile, and items are
  always fetched lazily via a real GET on click, for every collection
  relationship uniformly. Jumping to one *specific* known sibling version
  by id still needs no full-path construction: just `versionsurl + '/' +
  id`, appending a guaranteed id onto a real collection link.

Verified via a CDP-driven headless-Chromium script against a live
sample registry (`reg-Endpoints`, nested endpoint/message/version
hierarchy): clicked through registry root → group type → group
instance → resource type → resource instance → versions → specific
version, confirming every fetched URL exactly matched a real link field
from the previous response (`endpointsurl`, entity `self`,
`messagesurl`, `versionsurl`, version `self`). Confirmed the address
bar persists `apiurl=`, a fresh reload of a deep bookmarked URL performs
a single GET using that persisted value (no path re-walking), and a
breadcrumb click on a cold-loaded (uncached) ancestor still resolves
correctly via the accepted trim fallback. Also confirmed the "→ Visit"
parent-resource button correctly reuses the session's cached resource
URL.

## Follow-ups after link-driven navigation landed — done

Three small bugs/gaps found while exercising the new link-driven
navigation, all fixed and CDP-verified against `reg-Endpoints`:

- **Stale `server=` on the home page.** Going registry → home (via the
  "xR" breadcrumb logo, or the header dropdown's Home option) left a
  leftover `server=` query param in the address bar, even though the
  home view isn't tied to any one registry. Root cause:
  `pushState({view:'home', ...})` call sites never cleared
  `_state.serverURL`, and `buildURL()` unconditionally emitted `server=`
  whenever it was truthy. Fixed by clearing `serverURL: ''` at both
  call sites (`index.html`'s `#logo-link` onclick, `onRegistryChange()`'s
  `__home__` branch) plus a defensive guard in `buildURL()` that never
  emits `server=` while `st.view === 'home'`.
- **Breadcrumb hover showed the wrong/identical URL.** Every breadcrumb
  `<a>` had a hardcoded `href="#"`, with real navigation done entirely
  via `onclick="...;return false"`. Since `#` resolves relative to the
  *current* page, hovering any ancestor breadcrumb showed "current deep
  page URL + #" for every segment — identical and wrong. Fixed by adding
  `pageHref(path, apiURL, extra)` (a small wrapper around `buildURL()`)
  and giving each breadcrumb segment its own real, accurate `href`
  (`buildBreadcrumbSegments()`/`renderSegment()`). Bonus: this also made
  ctrl/middle-click "open ancestor in new tab" work correctly for free.
- **Make all navigation click-targets real `<a href>` elements**, so
  users can natively ctrl/middle-click or right-click "open in new tab"
  anywhere they can already click to navigate — while keeping the
  existing fast SPA `pushState()` on a plain click. Explicitly OUT of
  scope (per user): non-navigation displayed values like the "self"
  field. Converted, all using `pageHref()`/`buildURL()` to compute a
  real href alongside the existing `onclick`:
  - Grid-view tiles: the tile itself stays a plain `<div class="tile">`
    (so nested pills, which need their own destination, can't be inside
    an outer `<a>`); a full-cover invisible first-child
    `<a class="tile-linkarea" style="position:absolute;inset:0">` catches
    whole-tile clicks (the "stretched-link" pattern, same technique
    Bootstrap uses). Nested resource-pills got `position:relative;
    z-index:2` so they sit above the overlay and keep navigating to
    their own target.
  - Table-view rows: `<tr>` can't legally be an `<a>`, so the row keeps
    its existing `onclick` (whole-row click convenience, no new-tab
    support there), while the id-cell's text and any nested pills are
    now real `<a>` tags (with `event.stopPropagation()` to avoid double
    navigation firing from both the cell's link and the row's onclick).
  - Collection tiles (`collectionTile()`/`groupTileHTML()`), the
    "Registry Endpoints" section tiles, `renderSingleEntity()`'s
    collections table, the home page's flat grid/list tiles/rows, and
    the home page's server cards (grid view) — all converted the same
    way: real `<a>`, computed href, existing `onclick` kept (with
    `return false`) for the fast-path SPA navigation.
  - The "→ Visit" buttons (default version id, version id, ancestor
    version id ×2, parent-resource) were `<button onclick=...>`;
    converted to `<a href=... onclick=...;return false>` — the
    `.eg-link-btn` CSS class already had no button-specific styling
    reliance, so the tag swap is visually a no-op.
  - Accepted trade-off (confirmed reasonable, not fixed): native
    middle-click/ctrl-click bypasses any JS-side "is this actually
    clickable" gate, since the browser never runs the onclick handler
    for those gestures. Concretely, the home page's error-state server
    cards (a `!` badge means "couldn't connect") can now still be
    opened in a new tab via middle-click even though a normal left-click
    is blocked by `serverCardClick()`'s early-return. This matches how
    real links normally behave and was judged an acceptable trade-off
    for correct new-tab support everywhere else.

## Query-parameter reference (why `view`/`dview`/`apiurl` etc. look the
## way they do)

Came up when reviewing hover URLs/address-bar contents; recorded here
since it's non-obvious from the param names alone. See `buildURL()`
(~line 254) for the authoritative logic.

- **`view`** is the *page-level mode*, not a data display choice: only
  `table` (in-registry browsing — the vast majority of pages) or
  `config` ever appear in the URL; `home` is the default and is never
  written to the URL. This name is a historical leftover from before
  the grid/list/JSON toggle was split out into its own `dview` param —
  today it really means "which top-level page are we on", not "which
  view of the data".
- **`dview`** ("data view") is the actual per-page display toggle —
  `grid` / `table` (List) / `json` (or the model editor's `list`/`json`
  for `section=model|modelsource`). It's omitted whenever it equals the
  computed default for that section/depth (`defaultDataView()`), so
  most URLs don't show it at all — you mostly see it when a user has
  picked a *non-default* view for that depth/section, or when a
  persisted per-depth/per-section preference differs from the
  hardcoded default.
- **`apiurl`** is the actual server-provided URL (`self` /
  `<plural>url` / `versionsurl` / etc.) used to fetch the current
  page's data — *not* reconstructed from `server=` + `path=` at fetch
  time (see `buildAPIURL()`, ~line 451: `_state.apiURL || buildBaseURL()`,
  i.e. the real link is used when we have it, falling back to
  path-based construction only when we don't). It's only ever emitted
  when `section=data` (the four fixed-suffix sections — model,
  modelsource, capabilities, capabilitiesoffered — always use a
  hardcoded, trivial URL, so there's nothing worth persisting) and only
  when `path` is non-empty (the registry root's URL is always trivially
  `server` itself, so it'd be pure redundancy there).

  **Concrete, already-verified case where it diverges from `server=`+
  `path=` reconstruction today** (not just a future hypothetical):
  encoding of IDs containing `:` or `@` — both are legal xRegistry id
  characters per the spec's id regex (`shared_model.go`'s `RegexpID`:
  `[a-zA-Z0-9_.\-~:@]`, both unreserved in a URL path segment per RFC
  3986, so the Go server never percent-encodes them when building
  `self`/`Location`/etc. — confirmed by creating a real version with id
  `test:colon`: the server returned
  `.../versions/test:colon`, raw colon, unescaped). Our client-side
  path-reconstruction (`buildBaseURL()`, via `encodeURIComponent` per
  segment) would instead produce `.../versions/test%3Acolon` for that
  same id — a different literal string for the "same" URL. Using the
  real `apiurl` avoids ever needing to reconcile that mismatch. More
  generally (not yet concretely exercised, but the reason the design
  doesn't special-case around the above): the server is never obligated
  to return simple path-concatenated URLs at all — e.g. once a query is
  scoped by an inherited filter (see below), a collection's real
  `<plural>url` could legitimately point somewhere `server`+`path`
  alone could never reconstruct. Persisting the real link (rather than
  only `server`+`path`) is what lets a bookmarked/reloaded deep link
  skip re-deriving it and skip re-walking the hierarchy.
- **`filter`** (only ever added while in JSON view — see the
  `inJsonView` guard in `buildURL()`) is a *separate*, independent query
  param today — it is never appended onto the end of `path=` or
  `apiurl=` in the address bar. Server-side, at actual fetch time,
  `buildAPIURL()` appends `_state.filters`/`_state.inlines`/etc. as
  fresh query params onto whichever URL it's using (`apiURL` or the
  path-based fallback) — but that's assembled at fetch time, not stored
  back into the persisted `apiurl=` value shown in the browser bar.
  Once filtering is extended to grid/list views generally (tracked as
  a pending backlog item — see `grid-list-filters` follow-up), the plan
  discussed earlier in this doc is for the *server's own* returned
  collection links to already carry the inherited filter baked directly
  into the URL string; at that point such a filter would show up
  naturally as part of `apiurl=`'s value itself (not as today's
  separate top-level `filter=` param), and `path=`/`apiurl=` would
  finally diverge more often since `path=` only ever describes segment
  names, never query state.

## Follow-up: ctrl-click "open in new tab" bug + stale logo-link href — done

Two regressions reported after the link-driven-navigation/real-anchor work
landed:

1. **Logo-link (`#logo-link`, the "xR" icon) had a stale/wrong href.**
   Unlike the breadcrumb root segment (fixed previously), this separate
   element still had a hardcoded `href="#"`, which resolves relative to
   the *current* page — so hovering it while on, say, the Model page
   showed a URL with `&section=model` still attached, and opening it in a
   new tab landed back on the current page instead of Home. Fix:
   `renderHeader()` now sets `#logo-link`'s `href` on every render via
   `buildURL({view:'home', path:[], serverURL:''})`, so it always reflects
   the true (query-param-free) home URL.

2. **Ctrl/cmd/shift-click never opened a new tab, anywhere.** Every
   converted nav `<a>`'s `onclick` unconditionally ended in `return
   false` (to suppress default navigation in favor of SPA `pushState`).
   Browsers honor that suppression even when a modifier key is held, so
   despite `href` being correct, the browser's native "open in new tab"
   behavior was silently blocked everywhere. (A true middle-click doesn't
   normally fire `onclick` in most browsers, so it was largely
   unaffected; the break was specifically ctrl/cmd/shift + left-click.)

   Fix: added two small shared helpers in `app.js`:
   - `navShouldDefault(e)` — `true` if `e.ctrlKey || e.metaKey ||
     e.shiftKey || e.button === 1`.
   - `guardedOnclick(expr)` — wraps a raw onclick expression as
     `if(navShouldDefault(event))return true; <expr>; return false`.

   Applied across every real nav anchor: breadcrumb segments, the
   Registry Endpoints tiles (Grid) and table (List), grid-view tile
   pills/link-area, table-view row id-cells and pills, List-view
   collection-table id-cells, `collectionTile()`, the home page's
   flat grid/list tiles/rows, `renderHomeTable()`'s name link,
   `serverCard()` (`serverCardClick` now takes the event first and
   checks `navShouldDefault` *before* the connection-error gate, so a
   modifier-click still opens a broken registry's URL in a new tab —
   intentional, matches how real `<a>` elements behave), and all 5
   "→ Visit" buttons. Left out of scope: the header's compact-mode "..."
   popup menu items (`href="#"`, secondary/corner-case UI, not part of
   the reported regression).

   While auditing, two previously-missed conversions (same root category
   — a clickable spot with no real `href` at all, predating this turn's
   regression) were also found and fixed: `renderSingleEntity()`'s
   List-view "Registry Endpoints" table (the Grid-view equivalent had
   been converted already, this one was missed) and `renderHomeTable()`'s
   name cell (a plain `<span onclick>`; the sibling `renderHomeFlatList()`
   table had been converted already, this one was missed). Converting
   the latter to a real `<a>` surfaced a small CSS bug: `.ht-name-link`'s
   rule assumed it always wrapped a nested `<a>` (`.ht-name-link a {
   color: inherit; text-decoration: none; }`), so when the anchor itself
   carried the `.ht-name-link` class directly it fell back to default
   browser blue/underline styling. Fixed by adding `text-decoration:
   none` directly to the `.ht-name-link` base rule so it applies whether
   `.ht-name-link` is the anchor itself or a wrapper around one.

   Verified via CDP (headless Chromium): for the logo-link, a breadcrumb
   segment, a Grid tile, a List row's id-cell, and a "→ Visit" button —
   simulated ctrl-clicks left `_state` unchanged and did not prevent the
   event's default action (so the browser is free to follow `href` into
   a new tab), while plain clicks on the same elements still correctly
   triggered the fast SPA `pushState()` navigation. Confirmed the
   `renderHomeTable()` styling fix visually (link renders in the same
   blue, non-underlined style as before, underlining only on hover).

## Filter support in Grid/List views (design)

Goal: bring the same OR-group filter builder already built for JSON view
to Grid and List views too, so all 3 views are feature-equivalent for
filtering (Sort/Inline/Options stay JSON-only for now — not yet decided
whether those make sense for Grid/List). A filter set anywhere should
survive switching between Grid/List/JSON and moving up/down the
hierarchy, without the client ever reconstructing a filter expression
itself — matching the existing "follow links, don't build URLs"
architecture.

**UI placement** (confirmed with user):
- Reuse the exact same filter-builder component JSON view already has
  (`buildFilterSectionInner()`/`fbFiltersTitleHTML()`/`ensureFbDraft()`
  etc.) — just the Filters section, not Sort/Inline/Options.
- Add a new "Filters (N)" toggle button in the header, next to the
  Grid/List/JSON view-switch buttons, visible only when
  `_state.section === 'data'` and the server capability reports
  `hasF('filter')`. Default OFF/hidden (Grid/List stay full-width by
  default); clicking toggles a panel open. `N` is the current filter
  count, shown even while collapsed (mirrors the existing `(N)` count on
  the JSON left-panel's own Filters section header).

**Key realization from design discussion — no URL reconstruction
needed at all for navigation itself:**
Real links followed while navigating (`navigateTo()`,
`navigateToNestedColl()`, breadcrumb clicks, tile/row clicks, etc.)
already carry a correctly-scoped `filter=` param when they originate
from an already-filtered response — the *server* computes and bakes
this in (confirmed via `curl`: fetching a registry root with
`?filter=endpoints.name=end1` returns `"endpointsurl":
".../endpoints?filter=name=end1"` — already narrowed to the child
collection's own attribute-naming scope). So none of the existing
navigation functions need to change — they already store the real
followed link verbatim as `_state.apiURL`, and that's sufficient.

The only place a filter expression is ever actually *constructed*
client-side is the deliberate "Apply" action (same as JSON view already
does), which becomes the single point where a new, user-edited filter
gets baked into a fresh `apiURL`. From then on, everything downstream
(further real links returned by the server) carries the filter forward
automatically — no client-side propagation logic needed anywhere else.

**Required code changes:**

1. `buildAPIURL()`: currently *always* appends `_state.filters` onto
   whatever `apiURL` is. Change so `_state.filters` is only appended in
   the fallback branch (`_state.apiURL` empty → `buildBaseURL()` used —
   i.e. first-ever load / a bookmarked URL with `filter=` but no
   `apiurl=`). When a real `_state.apiURL` is known, trust it completely
   as-is — it already contains whatever filter is relevant at this
   position (baked in by the server, or by our own "Apply" step below).
   This is what prevents the double-`filter=` param bug that motivated
   this whole design discussion.

2. New `stripFilterParams(url)` helper: strips only `filter=...` tokens
   from a URL's query string (plain token-level split, NOT
   `URLSearchParams` re-serialization, to avoid mangling any other
   params' original encoding — e.g. unescaped colons in ids).

3. New `applyFilters()` helper (shared by JSON view's existing Apply
   button and the new Grid/List Apply button): computes
   `{filters: fbCollectFilters(), apiURL: stripFilterParams(_state.apiURL
   || buildBaseURL()) + freshly-appended filter params}` and returns it
   as a patch. `applyJSONOptions()` is refactored to merge this patch in
   alongside its existing sort/inline/docView/binary/collections patch
   (those stay exactly as they work today — always freshly appended,
   JSON-only, no baking-into-apiURL needed since they're not exposed
   outside JSON view). The new Grid/List "Apply" button in the Filters
   panel just calls `pushState(applyFilters())` directly.

4. `pushStateReal()`: add a small resync step — whenever `_state.apiURL`
   ends up truthy after applying a patch, refresh `_state.filters =
   filtersFromUrl(_state.apiURL)` so the two always agree, in one
   central place, regardless of which code path changed `apiURL`
   (ordinary navigation, breadcrumb click, or Apply). This is what feeds
   the Filter Builder's displayed/editable draft (`ensureFbDraft()`
   already seeds from `_state.filters`) — the draft is always an
   accurate reflection of "what's actually active right now," never
   independently tracked/stale.

5. `crumbURLs` cache: when Apply changes `_state.apiURL` for the
   *current* depth (no path change), also patch
   `_state.crumbURLs[_state.path.length - 1]` to the new value, so a
   later breadcrumb-click back to this depth doesn't regress to the
   pre-Apply link.

6. Left panel: extend `setLeftPanelVisible()`/`renderJSONLeftPanel()`
   with a mode flag so Grid/List can show a slimmed-down version
   containing only the Filters section (skip the Registry Endpoints nav
   block, Sort, Inline, Options), toggled by the new header button
   rather than being tied to `_state.view === 'json'`.

**Verified via `curl` during design discussion:**
- `GET /reg-Endpoints?filter=endpoints.name=end1` →
  `"endpointsurl": ".../endpoints?filter=name=end1"` — confirms the
  server narrows/rescopes filter expressions per-level in returned
  links, which is the load-bearing fact behind not needing any
  client-side filter propagation during navigation.
- A plain collection GET (e.g. `/endpoints?filter=...`) returns a bare
  `{id: {...}, ...}` map with no top-level `self` of its own (only
  individual entities inside it have `self`) — confirms there's no
  "collection-level self" to resync `apiURL` from after a fetch; the
  `apiURL` used for the request itself remains the right source of
  truth.
- Confirmed a hypothetical gap in JSON view's own in-content link-click
  handling (`navigateJsonUrl`/`syntaxHighlight`): it only extracts
  `path` + `filter` from a clicked link and reconstructs the rest via
  `buildBaseURL()`, silently dropping any other non-filter query params
  a link might carry (e.g. a hypothetical required `?xreg=true`). Noted
  as a known, pre-existing, out-of-scope gap for this feature — not
  fixed as part of this work, but worth revisiting since the new
  `stripFilterParams()`-based approach (preserve everything, touch only
  `filter=`) is strictly more robust and could retrofit that code path
  later if desired.

### Implementation — done
All items above implemented in `registry/ui/app.js` / `index.html`:
- `buildAPIURL()` only appends `_state.filters` in the no-real-`apiURL`
  fallback branch; fixed a latent `?`/`&` separator bug in the same pass.
- `stripFilterParams(url)` / `applyFilters()` — shared helpers; `applyFilters()`
  is the single deliberate spot a new filter expression is baked into
  `apiURL` (used by both JSON view's Apply and the new Grid/List Apply).
- `pushStateReal()` centrally resyncs `_state.filters` from `apiURL` after
  every patch (whenever `apiURL` is present), and keeps `crumbURLs` in sync
  for Apply-without-path-change. This is what makes filters survive Grid ↔
  List ↔ JSON view switches and hierarchy navigation with no per-callsite
  plumbing.
- `applyJSONOptions()` refactored to merge in `applyFilters()`'s patch
  instead of computing filters/apiURL inline.
- New `applyGridFilters()` — `pushState(applyFilters())` — the Grid/List
  Apply button's handler (kept separate from `applyJSONOptions()` so it
  never touches JSON-only state like sort/inline/doc-view that don't exist
  in the slimmed panel).
- `renderJSONLeftPanel(filtersOnly)` — new `filtersOnly` mode renders only
  the Filters section (no Registry Endpoints nav, no Sort/Options/Inlines);
  its Apply button calls `applyGridFilters()` instead of `applyJSONOptions()`.
- New `isGridFiltersOnlyMode()` helper — single source of truth for "is the
  left panel currently in Grid/List's slimmed mode", used by `refresh()`,
  `setDataView()`, `toggleFiltersPanel()`, and `fbToggleCollapsed()` (the one
  internal filter-builder handler that does a full panel re-render rather
  than just refreshing `#lp-filter-section` in place via `fbRerender()`).
- New "Filters" header toggle button (`#filters-toggle-btn` in `index.html`,
  wired in `renderHeader()`) — shown only for the plain `data` section
  (outside JSON view) when the server's capabilities report `filter` in
  `flags`; shows an active/inactive state plus a live `(N)` count of
  currently-applied filter expressions. `renderHeader()` lazily fetches
  capabilities via `ensureCapCached()` if not yet cached, re-rendering the
  header once available (capabilities aren't otherwise pre-fetched for the
  plain data section).
- `toggleFiltersPanel()` flips `_filtersPanelOpen`, re-renders the header,
  and shows/hides + (re)renders the left panel accordingly.
- `refresh()` and `setDataView()` both call `renderJSONLeftPanel(true)` and
  `setLeftPanelVisible(...)` when Grid/List's Filters panel is toggled open,
  independent of the JSON-view-driven left panel logic.

**Verified via headless-Chromium (CDP) smoke test:** navigated to a
collection in List view, confirmed the Filters button is hidden until
capabilities load then appears with correct enabled state; toggled it open —
panel shows only the Filters section (Apply → `applyGridFilters()`, no Sort/
Options/Inlines/Registry-Endpoints); expanding/collapsing the Filters twisty
(`fbToggleCollapsed()`) correctly stays in filters-only mode (this was a real
bug caught during verification — fixed via `isGridFiltersOnlyMode()`);
entered a raw filter expression via Advanced mode and clicked Apply — `_state.
apiURL` and `_state.filters` updated correctly and the address bar reflected
the new `apiurl=`; switched List → JSON → List again — filter state persisted
unchanged across all switches with no duplicate `filter=` params. (Deep
hierarchy-navigation-with-active-filter re-verification was not repeated in
this pass since it depends on the link-driven-navigation work verified
earlier in this same design's development.)

### Follow-up fixes (found via real-world testing against xregistry.soaphub.org)

1. **Filters button state going stale across view switches.** `setDataView()`
   only toggled `[data-dview]` buttons' active class directly — it never
   called `renderHeader()` — so the new Filters button (not a `[data-dview]`
   button) kept showing whatever visibility/active-state it had from the
   last full header render. Symptom: button visible while in JSON view (it
   should only ever show outside JSON view), and clicking it there called
   `setLeftPanelVisible(false)`, hiding JSON view's own always-on left panel;
   switching back to Grid/List afterwards left the button stuck hidden too.
   Fixed by having `setDataView()` call `renderHeader()` on every switch, and
   added a defensive guard in `toggleFiltersPanel()` so a stale click while
   in JSON view is a no-op rather than corrupting JSON view's panel state.

2. **Filter silently lost one hop after drilling into an entity from a
   filtered collection (the big one) — plus its own hover/ctrl-click bug.**
   Repro: at the registry root, apply a dotted filter naming a deeply-nested
   attribute (e.g. `schemagroups.schemas.versions.schemaid=...`) — the
   `schemagroups` tile correctly narrows to 1 and its `apiurl` carries the
   rescoped `?filter=schemas.versions.schemaid=...`. Clicking into that one
   schemagroup entity, its `schemas` tile still correctly shows "1" — **but**
   clicking into `schemas` shows all 16, filter gone. Root cause: an
   entity's own `self` link (the only link a row/tile can navigate through
   to reach it) is the bare canonical URL and **never** carries filter
   context on its own — verified via `curl`: `GET /schemagroups/Contoso.ERP`
   (via bare `self`) returns `schemasurl` with no filter and
   `schemascount: 16`.
   - **First attempt (reverted):** had `navigateTo()` stash the
     already-known entity JSON from the parent collection response as an
     `_entitySeed`, and skip the re-fetch entirely on landing, reusing the
     seed's already-correctly-scoped `<plural>url` links. This worked, but
     introduced real problems: the entity's hover/ctrl-click/bookmark URL
     (still just the bare `self`) never showed the filter even though the
     rendered page was in fact filtered; refresh/reload of that URL lost
     the filter and the correctly-scoped view entirely; and the whole
     mechanism (seed stashing, path+URL matching, single-use consumption,
     attribute-by-attribute merging for `$details` fetches) was a lot of
     extra state and edge cases just to avoid one GET.
   - **Actual fix:** confirmed via `curl` that the server also honors
     `filter=` appended directly onto a single entity's own `self` URL —
     `GET /schemagroups/Contoso.ERP?filter=schemas.versions.schemaid=X`
     correctly returns `schemasurl` rescoped to `?filter=versions.schemaid
     =X` and `schemascount: 1`, exactly like it does for collection URLs.
     So instead of avoiding the re-fetch, tile/row links now simply append
     the CURRENT collection's active filter onto the entity's own `self`
     link before ever using it — see `entityHrefWithFilter()` (in
     `pageHref`'s neighborhood) — so the real `href` used for hover preview,
     ctrl/middle-click "open in new tab", bookmarking, AND the plain click
     itself are all the exact same URL, and `refresh()` always does one
     ordinary GET with no special-casing.
   - This fully replaced and removed the `_entitySeed` mechanism:
     `navigateTo()` is back to its original 2-argument form (no `item`
     seed), `refresh()` always fetches, and `_tileViewItems` (added only to
     support the seed) was removed. `filtersFromUrl(_state.apiURL)` alone
     is now sufficient everywhere (see `syncFiltersFromApiURL()`) since the
     apiURL itself always carries the filter when one is active, at every
     depth — no reconstruction-from-nested-links logic needed either.
   - Verified via CDP against xregistry.soaphub.org, full repro chain: root
     (bookmarked URL, filter embedded in `apiurl`) → filter builder shows it
     on load → click `schemagroups` tile → hovering `Contoso.ERP`'s tile
     shows `?filter=schemas.versions.schemaid=...` appended to its own
     `self` link → click it → fresh GET confirmed (`_lastData.schemasurl`/
     `schemascount: 1` came back correctly rescoped from the server, not
     reused) → **hard-reloading that exact page's URL** reproduces the
     identical filtered result from scratch → click into `schemas` →
     still just the 1 matching item, filter fully intact end-to-end.

3. **Version entities incorrectly showed a `?filter=` on hover too.** Since
   `entityHrefWithFilter()` (fix #2) unconditionally appended the active
   filter to every entity's `self` link while browsing a filtered
   collection, it also did this for version tiles/rows inside a `versions`
   collection — but a version is always a spec-guaranteed leaf (no nested
   `<plural>url` collections of its own), so there's nothing left to
   rescope; showing `?filter=...` there was misleading busywork with no
   actual effect. Fixed by having `entityHrefWithFilter()` take the
   destination `itemPath` and skip appending when it's a version entity
   (`path.length === 6 && path[4] === 'versions'`). Verified via CDP: a
   version tile's real `href` (and the URL it navigates to) is now the
   plain unfiltered `self` link, with no filter param.

4. **Filters button visually mismatched other header icon buttons (esp.
   the gear/config button), in both height and with-vs-without the "(N)"
   count.** It was using the `.view-btn` class (grouped toggle-button
   styling: flat, no border, small font) instead of `.icon-btn` (standalone
   rounded icon button: bordered, `#e8e8e8` background) used by `#gear-btn`
   and friends. Fixed by switching its class to `.icon-btn` in `index.html`,
   re-sizing `.filter-funnel-icon` in `style.css`, and pinning an explicit
   `height: 28px` on `#filters-toggle-btn` so it measures identically to
   `#gear-btn` (28px) whether or not a filter count is showing (previously
   26px with a filter active, since the icon+count's natural content height
   fell short of the fixed-height gear icon's). Verified via CDP
   `getBoundingClientRect()` comparison in both states (with/without an
   active filter) — both now exactly 28px tall, matching `#gear-btn`.

5. **Applying an "Advanced (raw text)" filter looked like it got cleared,
   and stayed in raw-text mode instead of switching to the graphical
   builder.** Root cause: `applyGridFilters()`/`applyJSONOptions()` read
   the raw textarea's value into the outgoing filter patch via
   `fbCollectFilters()`, but never wrote it back into `_fbDraft` itself.
   Since `fbKey()` (used by `ensureFbDraft()` to decide whether to reset the
   draft) only depends on server/section/path — not on filters — the draft
   survived the navigation completely untouched, so the panel re-rendered
   showing its *pre-Apply* state: Advanced mode still checked, textarea
   re-populated from the stale (often empty) `_fbDraft.groups`, looking
   exactly like Apply had wiped it. Fixed by having both Apply handlers
   write the newly-applied filters back into `_fbDraft.groups` and reset
   `_fbDraft.advanced = false` (switching back to the graphical builder,
   which the user preferred) right before `pushState()`. Verified via CDP:
   pasting a raw filter expression, toggling Advanced, and clicking Apply
   now correctly lands on the graphical builder showing that exact filter
   as a chip — not an empty/stuck raw textarea.

6. **Whole-tile click blocked text selection/copy, on every tile type
   (entity Grid tiles, home-page Registry cards, Group Types tiles, doc
   tiles).** The entire tile was an `<a>`/clickable `<div>` with an
   invisible full-cover click target, so users couldn't select/copy any
   text inside a tile (description, id, etc.) without triggering
   navigation. Fixed across all four tile renderers so only the title/id
   is a real clickable, underlined link (`.tile-id-link` for entity
   tiles, `.server-card-name` for Registry cards, `.coll-tile-name` for
   Group Types/doc tiles), while the rest of the tile is a plain
   non-interactive container (no `cursor:pointer`, no hover/active
   transforms, no `user-select:none`). Hover/ctrl-click/middle-click
   ("open in new tab") behavior is preserved since the link keeps the
   same real `href`/`onclick` the old full-cover click target had.
   Verified via CDP that `.server-card`/`.coll-tile`/`.tile` are now
   plain elements with `cursor:auto`, and that only the name/id renders
   as an `<a>`.
7. **Footer alignment consistency.** Bottom-aligned the Created/Modified
   timestamps on entity tiles (`.tile-times`) and the registry-name label
   on Group Types tiles (`.coll-tile-reg`), using the same `margin-top:
   auto` technique already used for `.server-card-url` on Registry cards
   (parent is `display:flex; flex-direction:column` sitting as a direct
   child of a CSS Grid container with default `align-items:stretch`, so
   same-row tiles are equal height and the footer is pushed flush to the
   bottom regardless of variable content above it). Also ensured the
   horizontal divider immediately above `.tile-times` has at least as
   much space above it as below it (6px), by adding `margin-bottom: 6px`
   to `.tile-body` (covers tiles with no resource-pill footer) and
   `.coll-tile-res` (covers tiles that do have one) — previously that gap
   could collapse to 0 whenever `margin-top: auto` had no leftover space
   to distribute.
8. **Table/List view rows were also fully clickable, same problem as
   tiles.** Removed the `<tr onclick="...">` and the `tr:hover` background/
   pointer-cursor styling; only the id cell's text is now a real
   underlined `<a>` (same href/onclick as before). Other already-clickable
   cells (e.g. the Resources/Versions pills) are untouched since they were
   already scoped to their own link, not the whole row. Verified via CDP:
   row has no `onclick` attribute and `cursor: auto`, while the id link
   still navigates correctly.
9. **Resource tile/row's "Version: X" label was static text, and didn't
   make clear it was the *default* version.** Renamed to "Default
   Version: X" and made it a real clickable link (`.coll-tile-res-pill-
   clickable` in Grid view, `.cell-version-count` in Table view) that
   navigates straight to that version in the versions collection —
   matching the existing pattern for the versions-count pill/cell next to
   it. Added `defaultVersionURL(item, itemPath, colls)` (resolves the
   version's real URL: prefers `item.defaultversionurl` when it matches,
   else the item's own `versionsurl` + versionid, else plain path
   construction — same priority as the existing `versionURLById()`) and
   `navigateToDefaultVersion(itemId, versionId, url)`. Verified via CDP:
   the pill's href resolves to `.../versions/1`, and clicking it lands on
   that exact version.
10. **Removed the general "click to copy" feature entirely** (it got in
    the way of highlighting/selecting text). This was an opt-in Config
    page checkbox (`optClickToCopy()`/`_opts.clickToCopy`) that wrapped
    almost every value shown on an entity's details page — description,
    page title id, icon, labels, epoch/timestamps, documentation URL, and
    any `renderValueTree()` leaf — in a clickable `.eg-copyable` span.
    Removed the option/checkbox, `optClickToCopy()`, and all of its
    branches; `copyable()`/`copyableMonospace()`/`renderValueTree()`'s
    `leaf()` now unconditionally render plain (non-interactive) spans.
    Removed the now-dead `.eg-copyable`/`.eg-doc-copy` CSS rules too. Left
    untouched: the explicit **Self / ShortSelf / XID** pill buttons
    (`copyBtn()`, `.eg-copy-btn`) and the **"{ } Copy JSON"** button
    (`copyEntityJSON()`/`copyMetaJSON()`, `.eg-copy-json-btn`) — these are
    dedicated, clearly-a-button UI, not generic click-anywhere-to-copy
    text, so they stay. Verified via CDP: `.eg-copyable` count is 0 on an
    entity details page, while Self/XID/Copy JSON buttons still render
    and work.
11. **Registry root's "Group Types" table (List view) was missing the
    "Resource Types" and "Description" columns the matching Grid-view
    tiles show**, and (along with the "Registry Endpoints" table and the
    document row on the resource/version details page) still had the
    whole-row-clickable problem that the other table/list views had
    already been fixed for. Fixed all three in `renderSingleEntity()`:
    attached `c.resources`/`c.description` from the model onto the
    Group Types collection refs (same as `renderEntityGrid()`'s tiles),
    added `<th>Resource Types</th>`/`<th>Description</th>` columns shown
    only when applicable (resource types only at depth 0; description
    only when at least one row has one — same "only show if present"
    pattern as other optional columns), and removed the `<tr onclick>`/
    `cursor:pointer` from all three tables so only each row's name/link
    cell is clickable. Verified via CDP: all `tbody tr` elements have no
    `onclick` and `cursor:auto`; clicking the id link still navigates
    correctly.
12. **Description column was single-line-truncated (ellipsis) in every
    table/list view; now allows wrapping across up to 2 lines** before
    ellipsis-truncating. Applies uniformly to `.cell-desc` wherever it's
    used (`renderTableView()`'s entity collections, and
    `renderSingleEntity()`'s Group Types table). Needed a small wrinkle:
    `-webkit-line-clamp`/`display:-webkit-box` doesn't reliably clip
    inside a `<td>` itself (computed `display` silently became
    `flow-root` instead, so overflow leaked a partial 3rd line) — fixed
    by wrapping the description text in a nested `<div
    class="cell-desc-text">` inside the `<td class="cell-desc">`, with
    the line-clamp styling on that inner div instead. Verified via CDP
    (inner div's rendered height matches exactly 2 lines) and visually
    via screenshot (clean 2-line wrap + ellipsis, uniform row heights).
13. **Group Instance page's "Resources" table (List/Table view, depth 2)
    now also gets a Description column** (mirroring Grid view's
    resource-type tiles, which already show a description from
    `model.groups[g].resources[r].description`), reusing the same
    generic `showCollDesc`/`.cell-desc`/`.cell-desc-text` machinery
    already in place for the Group Types table. Verified via CDP by
    injecting a long test description into the model cache and
    re-rendering — confirmed the column appears and wraps/truncates at
    2 lines exactly like the other Description columns.


14. **Table view header row background darkened** from `#f0f0f0` to
    `#e2e2e2` (hover `#e4e4e4` → `#d4d4d4`, border `#ddd` → `#ccc`) since
    it was too close to the page's `#f5f5f5` background, making the
    header hard to distinguish. Verified via screenshot.


15. **Resource Collection table view**: "Default Version" header now
    wraps across 2 lines ("Default" / "Version" via `<br>` + new
    `.cell-version-hdr` class with `width:1%; white-space:normal`)
    instead of forcing a wide single-line column. Also centered the
    "Created" and "Modified" date column headers (`thSort()` now takes
    an optional extra-class param) to match the already-centered
    Versions/Resources count columns. Verified via CDP/screenshot.


16. **Registry page (Table/List view, depth 0) — added a row of small
    pills above the Group Types table**, one per available Registry
    Endpoint section (Model, Model Source, Capabilities, Capabilities
    Offered), as an alternative/experimental presentation alongside the
    existing "Registry Endpoints" table (kept as-is per user request).
    Mutable endpoints get a trailing pencil icon (`&#9998;`, matching
    the header's edit-pencil symbol). New `.reg-endpoint-pills`/
    `.reg-endpoint-pill`/`.reg-endpoint-pill-edit` CSS. Verified via CDP
    screenshot and confirmed pill click navigates to the right section.


17. **All table/list views now show a page title mirroring Grid view's
    "TYPE: name" header**: single-entity pages (Registry root, Group
    instance, Resource instance, Version instance — `renderSingleEntity()`)
    show the same title format Grid view uses (e.g. "REGISTRY: CloudEvents",
    "SCHEMA VERSION: 1"), with the registry root stripping a redundant
    trailing " Registry" word from the name (e.g. "CloudEvents Registry" →
    "CloudEvents"). Collection pages (Groups/Resources/Versions listings —
    `renderTableView()`), which have no title in Grid's tile layout, get a
    simple plural-name title (e.g. "SCHEMAGROUPS") for visual consistency.
    The title's bottom divider line (`border-bottom` on `.eg-page-title`)
    is suppressed specifically inside `#table-container` so it doesn't
    duplicate the visual weight already provided by the table headers below
    it — Grid view's title divider is untouched.
18. Renamed the Registry Config pills' label from "Registry Config:" to
    "Config:" and darkened it from gray (#666) to black (#222) for
    better readability; moved onto the same line as the pills.
19. **Removed the "Registry Endpoints" table** (Table/List view, registry
    root) entirely — initially just hidden behind `if (false && ...)`,
    later deleted outright once confirmed the "Config:" pills above the
    Group Types table fully replace it.
    All verified via CDP screenshots across registry root, Group
    collection, Group instance, and Version instance pages.


20. **Versions collection page title (depth 5, Table/List view)** now
    prefixes the owning Resource ID before "VERSIONS" (e.g. "Contoso.ERP.
    CancellationData VERSIONS"), matching the info previously only
    visible via the "..." breadcrumb. New `.eg-page-title-id-prefix` CSS
    class (plain, non-flex, unlike `.eg-page-title-id` which is used
    after the type label elsewhere). Other collection pages (Group Types,
    Resources, etc.) are unaffected. Verified via CDP screenshot.


21. **Version instance page title (depth 6, Table/List view)** now
    prefixes the owning Resource ID before "VERSION" instead of the
    resource's singular type name (e.g. "Contoso.ERP.CancellationData
    VERSION: 1" rather than "SCHEMA VERSION: 1"), reusing the same
    `.eg-page-title-id-prefix` class as the Versions collection page
    title. Resource instance page title (depth 4) is unaffected. Verified
    via CDP screenshot.


22. **Resources collection page title (depth 3, Table/List view)** now
    prefixes the owning Group's singular type + id before the plural
    label (e.g. "schemagroup Contoso.ERP SCHEMAS"), same pattern as the
    Versions collection page's Resource ID prefix. Verified via CDP
    screenshot; other collection page titles (Group Types, Versions)
    unaffected.


## Resource/Version page (Table/List view): tabbed Document/Details
## component — done

Replaced the old always-stacked layout (collapsible "`<Singular>`
Details" meta box + Document table + Properties table) on Resource
(depth 4) and Version (depth 6+) single-entity pages, Table/List view
only, with a small pill tab bar (`.eg-doc-tabs`/`.eg-doc-tab`) plus one
content panel (`.eg-doc-tab-panel`) shown at a time. First available tab
is always the default on every render (no persistence across
navigation). Grid view (`renderEntityGrid()`) is untouched.

- **Resource page tabs** (in order, when available): "`<Singular>`
  Document" (only if `hasdocument`), "Default `<Singular>` Version (id)
  Details" (existing Properties table content, unchanged), "`<Singular>`
  Details" (existing meta-box content, unchanged, twisty removed since
  tabs already gate visibility — Copy JSON button now lives inside the
  panel instead of the old collapsible header).
- **Version page tabs**: "`<Singular>` Document" (only if `hasdocument`),
  "Version Details" (existing Properties table content, unchanged) — no
  separate meta tab, since Version entities have no meta/properties
  split.
- **Document tab inline preview** (`loadDocumentPreview()`): reuses
  `openDocument()`'s existing source-resolution priority (`<key>url` →
  `<key>base64` → inline JSON value → `self` with `$details` stripped)
  but fetches/decodes and renders inline instead of always opening a new
  tab. Binary-vs-text detection is **byte-sniffing only** (NUL byte, or
  >5% non-whitespace control chars in a leading 8KB sample) — the
  declared `contenttype` is intentionally ignored, per explicit user
  choice. Text renders in a read-only `<textarea>`; binary shows "Binary
  file — preview not available."; both (and fetch errors) show a small
  "Open in new tab" link/button alongside/instead.
- **Details tab** (`loadMetaDetails()`): same lazy metaurl-fetch +
  render logic as the old `toggleMetaBox()`, minus the collapse/twisty
  toggling (tab visibility already handles that) — `renderMetaBoxContent()`
  /`renderMetaTable()`/`copyMetaJSON()` reused as-is.
- **Textarea auto-sizing** (`sizeDocTextarea()`): the Document tab's
  textarea grows to fill the remaining visible viewport space below it
  (`window.innerHeight` minus its top offset minus the "Open in new tab"
  link's height/margin), so it uses as much room as possible without
  pushing that link below the fold. Re-run on tab activation, after the
  preview content loads, and on window resize.
- New CSS: `.eg-doc-tabs`, `.eg-doc-tab`/`.eg-doc-tab.active`,
  `.eg-doc-tab-panel`, `.eg-doc-panel-header`, `.eg-doc-textarea`,
  `.eg-doc-preview-actions`, `.eg-doc-binary-msg`, `.eg-doc-error-msg`.
- Verified via CDP against xregistry.soaphub.org: 3-tab Resource page
  (Document defaults, textarea renders schema JSON, other 2 tabs
  unchanged content), 2-tab Version page, Document tab correctly absent
  when `hasdocument:false`, Details tab lazy-fetch + Copy JSON, simulated
  fetch-failure fallback (inline error + "Open in new tab"), simulated
  binary-content fallback ("Binary file..." + "Open in new tab"), and
  textarea auto-sizing (grows to fill viewport, "Open in new tab" stays
  visible, recalculates on `resize`). Grid view confirmed unaffected.

## Resource page: version-selector dropdown (experimental, precursor to
## retiring the Versions collection page) — done, pending user feedback

Added a standalone "Version:" control to the Resource page (depth 4) that
lets the user pick which version's data feeds the existing Document/Details
tabs, independent of which tab is currently active. Two iterations:

1. First attempt embedded the version picker *as* one of the pill tabs (a
   `<select>` styled to look like a button). User feedback: this conflated
   two independent choices (which version vs. which panel to view) into one
   control — split them apart.
2. Current design: the "Version:" dropdown (`#eg-doc-version-select`, wrapped
   in `<span class="eg-version-selector">`) is a **separate, non-tab
   control** rendered inline in the *same row* as the tab buttons — final
   order is `[Version: <dropdown>] [Document] [Version Details] [<Singular>
   Details]` (dropdown always first; the tab buttons follow in their normal
   order, still starting with "Document" as the default-active tab; no
   border on the dropdown — sits flush with the neighboring tab buttons).
   The Document tab's label was also simplified from "`<Singular>`
   Document" to plain "Document" (the entity type is already shown in the
   page's own title/breadcrumb). Selecting a version never switches tabs —
   it only changes which version's data the "Version Details" tab and the
   Document tab's preview show. Lists "Default
   (`<versionid>`)" (always first, selected by default — includes the
   resource's actual default versionid, via new `defaultOptionLabel(data)`
   helper, so it's clear which version "Default" resolves to without
   selecting it) plus one option per version id, fetched from
   the resource's `versionsurl` collection. This is a design experiment
   requested by the user as a precursor to a possible future decision to
   retire the separate Versions collection page — not yet approved for that;
   for now it only changes the Resource page.

- **Data source**: `GET <versionsurl>` (no `$details`) returns a map of
  versionid → full version object (all scalar properties: epoch,
  description, createdat, modifiedat, versionid, isdefault, ancestor,
  contenttype, format, self, etc.) but **no document content fields** — the
  document itself is still fetched via that version's own `self` field
  (with the `$details` suffix stripped), exactly matching the existing
  `openDocument()`/`loadDocumentPreview()` fallback logic. No new
  document-URL-construction code was needed.
- **`buildEntityPropsTableHtml(entityData, headerLabel, model, path,
  collKeys)`**: shared helper factored out of the original inline
  Properties-table-building code; used both for the initial Resource-page
  render and to rebuild the "Version Details" panel after a version is
  picked.
- **Global state for cross-closure handlers** (`_docActiveVersionData`,
  `_resModel`, `_resPath`, `_resCapType`, `_resDefaultData`, `_resCollKeys`,
  `_resVersionsUrl`, `_resVersionsList`, `_resSelectedVersionId`): needed
  because `loadVersionsForSelect()`/`onVersionSelectChange()` run from
  later, independent `<select>` change events, not from within the
  `renderSingleEntity()` closure that built the dropdown — same pattern
  already used for `_metaData`/`_docSingular`/`_docPreviewLoaded`. Reset at
  the top of the relevant code path so state doesn't leak across
  navigations to non-Resource pages.
- **`loadVersionsForSelect()`**: fetches `_resVersionsUrl`, converts the
  response via the existing `collectionItems()` helper, and populates the
  select's options (Default + one per version id via `itemNavKey()`),
  preserving the current selection; fails silently (non-critical, no error
  UI) if the fetch fails.
- **`onVersionSelectChange(vid)`**: looks up the selected version's data
  (from `_resDefaultData` for "default", else from the cached
  `_resVersionsList`), rebuilds the "Version Details" tab panel (key
  `defver`, plain button, label always "Version Details" — no longer
  embeds the version id/"Default" in the tab label since that's now shown
  by the dropdown) via `buildEntityPropsTableHtml()` (passing `collKeys:
  null` for a specific version — sub-collection suppression only applies
  to the resource's own top-level JSON; the table's own header row still
  dynamically reflects the picked version, e.g. "Schema Version (1)
  Property"), and refreshes the Document tab's content via
  `loadDocumentPreview()` by first setting `_docActiveVersionData` +
  `_docPreviewLoaded = true` — done unconditionally (not just when the
  Document tab is currently visible) so switching to it later shows the
  right version's content without an extra click.
- `loadDocumentPreview()` changed to read `_docActiveVersionData ||
  _lastData` instead of always `_lastData`, so it picks up the selected
  version transparently.
- New CSS: `.eg-version-selector` (inline-flex, no border — sits flush in
  the same `.eg-doc-tabs` row as the tab buttons) and its `label`/`select`
  styling (select styled to echo the tab-button palette without literally
  being a tab).
- Version page (depth 6+) is untouched — still uses the plain button tab
  bar (2 tabs, no version selector), since a Version entity has no
  "default vs specific version" distinction to select between.
- **"Details" → "Metadata" rename**: the "`<Singular>` Details" tab (List
  view, `meta` tab key) and its Grid-view equivalent (the collapsible
  "`<Singular>` Details" section, both driven by the same `metaurl`-fetched
  content via `loadMetaDetails()`/`renderMetaBoxContent()`) were renamed to
  "`<Singular>` Metadata" (e.g. "Schema Metadata") — this data genuinely is
  the entity's xRegistry metadata (readonly/compatibility/deprecated flags,
  etc.), so the more specific term reads better than the generic "Details".
  The unrelated Grid-view raw-properties-table headers (`detailsLabel` at
  depth 0/2/4/6, e.g. "Default Schema Version (2) Details") were
  intentionally left as "Details" — they show the entity's own flattened
  properties, not the separate `metaurl` metadata concept, so renaming them
  to "Metadata" would be misleading. The "Version Details" tab (List view
  `defver`/`version` tab keys, properties-table content) was also left
  unchanged for the same reason.
- **"Versions List" navigation link**: a discussion of whether the new
  Resource-page tab bar could fully replace the raw Versions collection
  page (and Version entity page) surfaced two gaps: (1) the "Version:"
  dropdown doesn't scale well to resources with many versions — no
  search/sort/filter, just a flat list — whereas the Versions collection
  page (List/Grid view) already has full filter support (see
  `plan.md`'s filter-support sections); (2) there was no direct link from
  the new tab bar to that collection page anymore (only the pre-existing
  "versions" link/box above the tab bar). Decision: keep both pages for
  now (don't retire them) and add a real navigation link, styled as a pill
  to match the tab buttons but visually distinguished — `.eg-doc-tab-link`
  (an `<a>`, not a `<button>`, no `active` state, with a trailing "→" via
  `::after` content) — placed last in the tab-bar row, after "`<Singular>`
  Metadata", labeled "Versions (`<count>`) List" (includes the versions
  collection's own count, from the same `colls` data already used
  elsewhere, e.g. "Versions (2) List"). Clicking/activating it navigates
  (`navigateTo('versions', <collection url>)`, same helper used by the
  ordinary collections-table links elsewhere) to the raw Versions
  collection page exactly as if the user had clicked the "versions"
  link/box. The direct link to the entity's own `/meta` endpoint was
  explicitly discussed and intentionally skipped for now (lower priority,
  no UI need identified yet).
- **Removed the "versions" link/box from the Resource page** (List/Table
  view): the plain collections-table (id/count rows, used at other depths
  for e.g. "schemas"/"resources") is now suppressed entirely at depth 4 —
  its only row there was ever the single "versions" collection, and that's
  now fully covered by the "Versions (n) List" link described above. Grid
  view's "versions" card/tile is unrelated and untouched (different
  layout, not requested).
- **Version page (depth 6+): tab bar suppressed when there's no Document
  tab.** When a resource type has no document (`hasdocument:false`),
  the Version entity page only ever had one possible tab ("Version
  Details") — no Document tab, and (unlike the Resource page) no
  version-selector/Versions-List controls at that depth either — so the
  pill tab bar had nothing to actually switch between. In that case the
  "Version Details" content (the plain Properties table) is now rendered
  directly, with no tab button/chrome at all, matching how the Registry
  root (depth 0) and Group instance (depth 2) pages already show a plain
  Properties table with no tabs. When the resource type *does* have a
  document, the Version page's 2-tab bar (Document / Version Details) is
  unchanged.

Verified via CDP against a real multi-version resource
(`Fabrikam.Watchkam.MotionDetectedEventData`, 2 versions): dropdown
populated with "Default", "1", "2"; selecting "1" correctly fetched
version 1's document and updated both the Document tab and the "Schema
Version (1) Property" table (`isdefault: false`); switching back to
"Default" correctly reverted both panels to the resource's own flattened
data (`isdefault: true`, matching the highest/default version). Also
verified: a single-version resource's dropdown shows exactly `["default",
"1"]` with no errors; a resource with `hasdocument:false` still shows the
dropdown as the active/default tab (no Document tab present); the Version
entity page's tab bar is unaffected (still 2 plain buttons, no dropdown).

## Removed all "{ } Copy JSON" buttons (List and Grid views) — done

Removed the "{ } Copy JSON" button in all 3 places it appeared:
List view's "`<Singular>` Metadata" tab panel (Resource page, depth 4),
Grid view's collapsible "`<Type>` Metadata" box header (depth 4), and
Grid view's "Details" section header (present at every depth: Registry
Details, `<Type>` Details, Default Version Details, Version Details).
Also removed the now-dead `copyMetaJSON()`/`copyEntityJSON()` functions,
the `.eg-copy-json-btn` CSS rule, and the now-unused
`.eg-doc-panel-header` CSS rule (it existed solely to right-align the
Copy JSON button above the Metadata tab's content). `_lastData`/
`_metaData` (the JSON payloads the buttons used to copy) are still used
elsewhere and were left untouched. Verified via CDP screenshots: List
view's Document/Version Details/Metadata tabs and Grid view's Metadata
box/Details section all render cleanly with no leftover button or
button-shaped gap. Context: part of a broader push (per user feedback)
to make the List view fully capable, as a precursor to possibly
deprecating most of the Grid view — not yet decided.

## List view: standalone "meta" page (depth 5) redirects to Resource page — done

List view (`renderSingleEntity()`) had no redirect for the standalone `meta`
entity page (depth 5, `[G,gId,R,rId,"meta"]") — unlike Grid view
(`renderEntityGrid()`), which already redirected up to the Resource page
(depth 4). This meant: viewing a resource's `/meta` JSON in JSON view, then
clicking the Grid icon correctly landed on the Resource page, but clicking
the List icon dropped you onto the old standalone meta page (no longer
reachable any other way, and inconsistent with Grid view / the removed
"versions" box work). Fixed by adding the same depth-5-redirects-to-depth-4
check to the top of `renderSingleEntity()`, and, for a nicer landing
experience, auto-selecting the "`<Singular>` Metadata" tab (List view) /
auto-expanding the Metadata box (Grid view) via a one-shot
`_pendingMetaTabOnLoad` flag consumed right after the Resource page finishes
rendering — so the user sees the same Metadata content they were just
viewing, instead of the generic default tab.

**Bug found and fixed along the way**: the redirect's `pushState()` call
didn't pass `dataView` explicitly, so `pushStateReal()`'s "restore
depth/section default" logic recomputed it from `_opts.depthViews[4]`
(defaulting to `grid`) — silently overriding the view (`table`/`grid`) the
user had just clicked to trigger the redirect in the first place. Fixed by
passing `dataView: _state.dataView` explicitly in both redirects (List and
Grid), since an explicit `patch.dataView` always wins over the recomputed
default.

Verified via CDP: navigated directly to a resource's `/meta` endpoint in
JSON view (`view=json`, path ending in `/meta`); clicking the List icon
now correctly lands on the Resource page in List view, dataView reads
`table` (previously incorrectly ended up `grid`), and the Metadata tab
auto-activates. Also confirmed Grid view's matching redirect still works
after adding the same explicit-`dataView` fix.

## Home page tables: header row darkened + bold to match List view — done

`.home-table` (Registries tab / Group Types tab on the Home page) and
`.config-table` (Settings/Config page's "Registry Servers" list) both still
had the old light, non-bold header style (`color:#666; font-weight:normal;
border-bottom:#ddd`, no background) predating the `.xr-table` header
darkening (`#f0f0f0` → `#e2e2e2`, see item 14 above). Updated both to match
`.xr-table`'s header exactly (`background:#e2e2e2; font-weight:bold;
color:#333; border-bottom:2px solid #ccc; padding:8px 12px`). Also aligned
`.home-table`'s remaining box model with `.xr-table` for full consistency:
switched `border-collapse` from `separate` (with a plain `1px solid #ddd`
border) to `collapse` + `overflow:hidden` + the same `box-shadow` and
`border-radius:6px` (was 8px); `td` padding 8px→7px and border-bottom color
`#eee`→`#f0f0f0` to match exactly; `vertical-align` middle→top. Verified via
CDP screenshots: Registries list, Group Types list, and the Config page's
Registry Servers table all render with the same darkened/bold header, and
the table's rounded corners still clip cleanly under the new
`border-collapse:collapse` mode (no square-corner artifacts).

## Home page: Grid/List button required 2 clicks to visually update — fixed

Bug: on the Home page only (Registries tab / Group Types tab), clicking the
Grid or List icon changed the rendered content correctly but left the
*previously* active icon highlighted — requiring a second click to catch
the header up, and producing confusingly "swapped" behavior (e.g. click
List while Grid is active → view changes to List but Grid icon stays lit;
then click Grid → header icon jumps to List while content changes back to
Grid).

Root cause, in `setDataView(v)`: it manually toggled each button's `active`
class to match `v` (correct), then immediately called `renderHeader()` —
which recomputes each button's active state from `effectiveView`
(`currentHomeLayout()` on the Home page, reading `_state.homeLayouts[
_state.homeGroup]`). But the per-group `_state.homeLayouts[...] = v`
assignment happened in a block *after* that `renderHeader()` call, so
`renderHeader()` was still reading the stale (pre-click) layout value and
immediately re-clobbering the manual toggle back to the old button.
Data pages never hit this: they only ever read `_state.dataView` (already
set at the top of the function), with no separate persisted-preference
lookup in between, so no equivalent ordering issue exists there.

Fix: reordered `setDataView(v)` so the Home-page `_state.homeLayouts[...]
= v` (+ `_opts`/`saveOpts()`) persistence happens *before* the manual
button-toggle and `renderHeader()` call, so `effectiveView` is already
correct by the time the header re-renders. Verified via CDP: single click
on List (from Grid) now shows only `table` as `active` and renders List
content; single click back to Grid shows only `grid` as `active`.
Confirmed on both the Registries and Group Types tabs.

## List view Property tables: banding, badges, timestamps, category grouping

List view's plain key/value "Property" tables (the entity's own scalar
attributes) were functionally fine but visually plain/spreadsheet-like.
There are 3 render sites building this same table style in `app.js`:
`buildEntityPropsTableHtml()` (main Resource/Version Property panel),
the depth-0/2 (Registry root/Group instance) inline table inside
`renderSingleEntity()`, and `renderMetaTable()` (Metadata tab/box).
The depth-0/2 site turned out to be an exact duplicate of
`buildEntityPropsTableHtml()`'s per-row logic — refactored it to just
call the shared helper instead, so this feature only needed 2 real
implementations (`buildEntityPropsTableHtml()` + `renderMetaTable()`).

Four visual improvements added, all scoped via a new `xr-table-props`
modifier class (alongside `xr-table`) so regular collection List tables
are unaffected:

1. **Zebra banding** — subtle `#fafbfc` alternating rows, applied via an
   explicit per-row CSS class (`xr-row-band`) rather than `nth-child`,
   specifically so the alternation **restarts at each category-group
   boundary** (see #4) — every group's first row always renders
   unbanded, so grouping reads as intentional rather than the pattern
   drifting based on how many rows preceded it. Category header rows are
   never banded. Falls back to continuous whole-table banding when a
   table has no grouping.
2. **Boolean badges** (`renderBoolBadge()`) — compact pill: "✓ true"
   (soft green) / "✕ false" (neutral gray, not red — false isn't a "bad"
   state for most spec booleans, e.g. `isdefault:false`).
3. **Timestamp styling** (`formatTimestampValue()` +
   `relativeTimeFromNow()`) — type-driven via
   `getExplicitAttrType(...) === 'timestamp'` (a real xRegistry model
   type), so it covers `createdat`/`modifiedat` *and* any extension
   attribute a model declares `type: timestamp`. Renders muted-gray
   (reusing the existing `.cell-timestamp` treatment from collection
   List columns) plus a `title` tooltip with a simple relative-time
   string ("3 days ago"), computed client-side. `renderMetaTable()` is
   the one exception — Meta-level extension attrs aren't resolvable
   against the model today (`getAttr()` only supports Registry/Group/
   Resource attribute maps, not Meta), so its timestamp detection is
   limited to the two known meta spec keys (`createdat`/`modifiedat`)
   via a small `META_TIMESTAMP_KEYS` set rather than being fully
   type-driven.
4. **Category grouping** (`groupPropsByCategory()`, `PROP_CATEGORY_DEFS`)
   — spec-defined attrs (confirmed via the existing `isSpecAttr()`,
   including its dynamic `<singular>id`/`$RESOURCE*` pattern matching)
   are bucketed into fixed, ordered categories: **Identity** (id/
   versionid/xid/self/name — including the dynamic `<singular>id` field,
   e.g. `fileid`, handled as a special case since it won't literally
   match the bucket's `id` key), **Description**, **Versioning & State**,
   **Content**, **Timestamps**. Non-spec/custom attrs always land in a
   final **Extensions** bucket, never subdivided. Rendered as a slim,
   unbanded divider row (small-caps, muted gray, thin top border) — not
   a heavy box. Only non-empty categories are shown; if everything
   collapses into a single bucket (or there's no model/specLevel to
   categorize against), `groupPropsByCategory()` returns `null` and the
   caller falls back to today's flat, ungrouped list.

Verified via CDP screenshots against a test `dirs`/`files` model with a
boolean extension attr, a `type: timestamp` extension attr, and a
multi-version resource: Resource page's "Version Details" tab, "File
Metadata" tab, Registry root, and Group instance page all show correct
category headers (including `fileid` properly landing in Identity, not
Content), banded rows restarting per group, green/gray boolean pills, and
muted timestamps with populated relative-time tooltips (spot-checked via
`element.getAttribute('title')`).

## Fixed: meaningless filter clause carried onto Resource entity link (no rescoping effect)

### Problem
On a filtered Resource *collection* page (e.g. `dirs/d1/files` with a bare,
dot-free filter clause like `epoch>0` applied — i.e. one that refers to the
FILE's own attribute, used to decide which files show up in that
collection), hovering over (or clicking into) a specific file entity (e.g.
`f1`) carried that same `epoch>0` clause forward onto `f1`'s own link and
into `_state.filters`/the address bar after a real click-through. But once
you're AT `f1`, that clause has zero rescoping effect — confirmed via curl
that `GET f1$details?filter=epoch>0` returns 200 with `versionsurl`
completely unfiltered/bare — so showing it as an "active filter" on `f1`'s
own page was misleading with no actual effect. (`entityHrefWithFilter()`
already special-cased Version entities the same way, since a Version is
always a leaf with nothing to rescope — but the same "nothing left to
rescope" reasoning wasn't applied to Resource entities, whose only
possible child is `versions`.)

### Root cause
`entityHrefWithFilter()` carried `_state.filters` forward onto ANY
non-Version entity's link unconditionally, without checking whether each
filter clause actually still referenced one of the destination entity's
own child collections (a Group's resource plurals, or `versions` for a
Resource). A clause with no such reference is "terminal" — it was only
ever meaningful for filtering the PARENT collection, and has nothing left
to rescope on the member entity's own page.

### Fix
Added two new helpers (`registry/ui/app.js`):
- `childCollectionsFor(path)` — returns the nested child-collection plural
  names an entity at `path` actually has: a Group entity's declared
  resource plurals (from `_modelCache`), or `['versions']` for a Resource
  entity, or `[]` otherwise (Registry root, Version entities).
- `filtersRelevantForEntity(filters, path)` — filters an OR-group/AND-clause
  filter array down to only the clauses that reference one of `path`'s own
  children (mirrors the server's `FiltersRelativeToAbstract()` keep/drop
  logic).

`entityHrefWithFilter()` now calls `filtersRelevantForEntity()` instead of
using `_state.filters` directly, and returns the bare `self` link
untouched if nothing remains relevant. Also cleaned up a duplicated
comment block above the function left over from an earlier edit.

### Verified via CDP (live xrserver, `dirs`/`dirs/d1`/`dirs/d1/files/f1` test data)
- `files` collection filtered by bare `epoch>0`: hovering over / clicking
  into `f1` no longer carries any filter forward (`_state.filters` empty,
  no `filter=` in the address bar) — matches the confirmed-no-effect curl
  check.
- Same collection filtered by `versions.epoch>0` (references `f1`'s actual
  `versions` child): correctly still carries forward onto `f1`'s link
  unmodified (`filter=versions.epoch%3E0`).
- Root-level Group Type case regression-checked: `dirs.files.epoch>0` at
  the registry root still correctly relativizes to `files.epoch>0` on the
  `dirs` collection, and further correctly carries forward onto member
  entity `d1`'s own link (since `files` is one of `dirs`' real resource
  plurals) — both hover href and full click-through confirmed consistent.
- A bare, non-child-referencing clause (`description=d1`) on the `dirs`
  Group collection is correctly dropped from member entity `d1`'s link
  (no matching child collection named `description`).
- `node --check app.js` passes. Test data deleted, model reset to `{}`,
  chromium test process and temp profile/log files cleaned up.

**Status**: Complete.


## Added: Copy-API-URL button at the end of the breadcrumbs

### Request
Add a copy-to-clipboard icon at the end of the breadcrumbs that copies a
plain, curl-able URL for exactly the data currently being displayed — no
UI-only params (view=, panel=, dview=, etc.).

### Implementation (`registry/ui/app.js`, `registry/ui/style.css`)
- Reused the app's existing clipboard infrastructure (`egCopy()`/
  `showToast()`, already used elsewhere for other copy buttons) rather than
  adding a new one.
- New `copyCurrentAPIURL(event)` — calls `egCopy(buildAPIURL(), 'API URL')`.
  `buildAPIURL()` already exists and computes exactly the real request URL
  for whatever's currently displayed (respects the active section — data/
  model/modelsource/capabilities/capabilitiesoffered/export — plus any
  active filter/sort/inline params), so no new URL-building logic was
  needed.
- New `copyLinkBtnHTML()` / `showCopyLinkBtn()` helpers, and a small
  `<button class="icon-btn bc-copy-btn">📋</button>` appended right after
  the last breadcrumb segment in all the places breadcrumb HTML gets
  written: the normal `renderBreadcrumbs()` path and the `collapseBreadcrumbs()`
  (level-1, "…"-collapsed) path. Deliberately NOT added to `collapseLevel2()`
  (the extreme-narrow-screen fallback that collapses everything to a single
  label + hamburger menu) or to the Home/Config pages (`showCopyLinkBtn()`
  returns false for those) — none of those have a single meaningful "data
  URL" to copy.
- CSS: `.bc-copy-btn` reuses the existing `.icon-btn` look (same as the
  gear/edit buttons) but smaller (`font-size: 14px` icon, tighter padding)
  to fit inline with the 13px breadcrumb row.

### Verified via CDP (live xrserver)
- Button appears at the end of the breadcrumb row on ordinary data pages,
  nested entity pages, and both `model`/`modelsource` sections (correctly
  absent on Home and Config pages).
- Clicking it calls `navigator.clipboard.writeText()` with exactly the
  right URL and shows the "API URL copied" toast — verified for: the bare
  registry root, a Group Type collection with both `sort=` and `filter=`
  applied (full apiURL round-tripped correctly, e.g.
  `.../dirs?sort=filesid&filter=description=d1`), and the `modelsource`
  section (`.../modelsource`).
- Confirmed still correctly positioned immediately after the last
  breadcrumb segment at a 4-deep nested Resource entity page.
- (Test-harness note: an unrelated bug in the CDP test script itself —
  using `/json/new?<url>` with a raw, un-escaped `&`-containing target URL
  — was discovered and fixed mid-session, since Chrome's endpoint decodes
  the whole thing and had been silently truncating/mis-parsing test
  target URLs at embedded `&`s. Switched to opening a blank tab + a
  `Page.navigate` WebSocket command instead, which preserves the exact URL
  string. This was purely a test-tooling issue, not an app bug.)
- Test data (`dirs`/`dirs/d1`/`dirs/d1/files/f1`) deleted, model reset to
  `{}`, `node --check app.js` passes, chromium processes and temp files
  cleaned up.

**Status**: Complete.


## Added: Copy-API-URL button in breadcrumb bar

### Feature
A small clipboard-icon button (📋) is now appended right after the last
breadcrumb segment on every data/model/section page (Registry root, any
Group/Resource/Version entity or collection, and the model/modelsource/
capabilities/capabilitiesoffered sections). Clicking it copies the real,
plain, curl-able API URL for exactly what's currently displayed — the
same URL `buildAPIURL()` would use to actually fetch the page's data,
including any active filter/sort/inline params — to the clipboard, with
a toast confirmation ("API URL copied"). Not shown on the Home or Config
pages, since neither has a single "data" URL to copy.

### Implementation
`registry/ui/app.js`:
- `showCopyLinkBtn()` — true whenever `_state.view` isn't `'home'`/`'config'`.
- `copyLinkBtnHTML()` — renders the button (`.icon-btn.bc-copy-btn`, using
  the existing `icon-btn` header-button style).
- `copyCurrentAPIURL(event)` — calls `buildAPIURL()` and reuses the
  existing `egCopy()` clipboard-copy-with-toast helper (already used
  elsewhere, e.g. in doc-info pill copy buttons).
- Wired into all breadcrumb render paths in `renderBreadcrumbs()`
  (normal, full-width case) and `collapseBreadcrumbs()` (the
  `…`-collapsed medium-width case). Intentionally left off the most
  extreme `collapseLevel2()` narrow-mode fallback (single label + compact
  hamburger menu) to avoid crowding an already-cramped layout.

`registry/ui/style.css`: added `.bc-copy-btn` (small left margin, smaller
icon font size, reuses `.icon-btn` base styling).

### Verified via CDP (live xrserver)
- Registry root (no filter/sort): copies bare `http://<server>` URL.
- Filtered + sorted collection page (`dirs` with `sort=filesid` and
  `filter=description=d1` applied via a real `pushState()`, matching how
  the app itself would reach that state): copies the exact combined URL
  `http://<server>/dirs?sort=filesid&filter=description=d1` — matching
  `_state.apiURL`/what a real fetch would use.
- `modelsource` section (from a data-page `view:'table'` context, matching
  real navigation): copies `http://<server>/modelsource`.
- Home page and Config page: button correctly absent in both cases.
- `node --check app.js` passes.

Note: initial CDP testing hit a false-positive "missing filter" result —
traced to a test-harness artifact (Chrome's `/json/new?url=` DevTools
endpoint double-decodes percent-escaped characters embedded in the target
URL, corrupting a hand-crafted nested `apiurl=` query value before the
page even loads). Not an app bug — confirmed by re-testing via a real
in-page `pushState()` call instead of a hand-encoded URL string, which
produced the correct combined filter+sort URL.

**Status**: Complete.


## Updated: Copy-API-URL icon + tab/version-aware URL on Resource/Version pages

### Icon change
Replaced the clipboard emoji with a proper inline SVG copy icon (the
standard "two overlapping documents" glyph, same design language as
Material Icons' `content_copy` and most icon sets — matches the visual
style the user referenced from thenounproject). `_copyIconSVG` constant
in `app.js`; `.bc-copy-btn svg { display: block; }` in `style.css`.

### Tab/version-aware URL (Resource & Version entity pages)
#### Problem discussed
On a Resource/Version entity page, `buildAPIURL()` (used by the copy
button) returned the bare entity URL with no `$details` — meaning the
copied URL, if curled, would return the raw document body (or error)
instead of the JSON actually shown on screen. Separately, the user asked
whether the copied URL should reflect which tab (Version Details /
Document / Metadata) and which version (via the Resource page's
version-selector dropdown) is currently selected, rather than always
being the default entity URL.

#### Design (confirmed with user)
- **Version Details tab** (default detail view — key `defver` on Resource
  pages, `version` on Version pages): entity/version's own URL + `$details`.
- **Document tab**: the *plain* URL (no `$details`) — GETting it returns
  the actual document content regardless of whether it's a real hosted
  `<key>url`, base64, or an inline JSON value (server computes it either
  way) — confirmed with user this is correct even when there's no
  independently-hosted URL to point to.
- **Metadata tab** (Resource pages only): `resolveResourceMetaUrl()`'s
  `.../meta` URL.
- **Version-selector dropdown** (Resource pages only): swaps in that
  version's own `self` URL as the base before applying the above tab logic.

#### Implementation
`registry/ui/app.js`: added `buildTabAwareAPIURL()`, called by
`copyCurrentAPIURL()` instead of `buildAPIURL()` directly.
- Only kicks in for Resource (`path.length === 4`) or Version
  (`path.length >= 6 && path[4] === 'versions'`) entities in the `data`
  section — everything else (collections, Groups, model/modelsource/etc.)
  falls through to plain `buildAPIURL()` unchanged.
- Reads the currently-active tab straight from the DOM
  (`.eg-doc-tab.active[data-tab]`) rather than trying to re-derive/guess
  tab order from `_state.docTab` — tab order isn't fixed (Document comes
  FIRST when the resource type has one, otherwise Version Details is
  first), so an empty `_state.docTab` doesn't reliably mean any one
  specific tab.
- Uses `_docActiveVersionData || _resDefaultData || _lastData` as the
  entity data source — already the exact same "currently displayed"
  data the version-selector (`onVersionSelectChange()`) and Document tab
  logic use, so it automatically reflects the current version selection
  with no extra bookkeeping.

### Verified via CDP (live xrserver, `dirs/d1/files/f1` resource with 2
versions: `1` and default `v2`)
- Default landing tab (Document, since this resource type has one):
  copies the plain content URL (no `$details`).
- Switched to "Version Details" tab: copies `.../f1$details`.
- Switched to "Metadata" tab: copies `.../f1/meta`.
- Version-selector dropdown → version `1`, on "Version Details" tab:
  copies `.../f1/versions/1$details`.
- Same version selected, switched to "Document" tab: copies
  `.../f1/versions/1` (no `$details`).
- Direct navigation to a Version entity page (`.../versions/1`): tab keys
  are `['doc','version']`; "Document" tab copies plain URL, "Version
  Details" tab (`version` key) copies `.../versions/1$details`.
- Regression check: a plain collection page (`dirs`) still copies via
  unchanged `buildAPIURL()` (no tab-awareness applied, as expected).
- `node --check app.js` passes. Test data cleaned up, chromium/temp files
  removed.

**Status**: Complete.


## Updated: Metadata tab disables version selector

### Problem discussed
On the Resource page, the version-selector dropdown lets the user view
different versions' data in the Document/Version Details tabs. But
`metaurl`/the Metadata tab is a per-Resource concept, not per-version
(`loadMetaDetails()` always reads `_lastData.metaurl`, ignoring which
version is selected) — so the dropdown has no actual effect while
viewing Metadata, which could mislead the user into thinking it does.

### Fix
`registry/ui/app.js`:
- `switchDocTab()`: disables `#eg-doc-version-select` (with a
  `title="Metadata is the same for all versions"` tooltip) whenever
  switching to the `meta` tab; re-enables it when switching away.
- Initial render: added `verSelDisabledD` (`_state.docTab === 'meta'`)
  so the selector starts disabled if the page loads directly onto the
  Metadata tab (e.g. via a restored `tab=meta` URL param), not just when
  switching tabs interactively afterward.

### Verified via CDP (`dirs/d1/files/f1` test resource, 2 versions)
- Selector starts enabled on default tab; switching to Metadata tab
  disables it (with tooltip); switching back to Version Details
  re-enables it.
- Loading the page directly with `tab=meta` in the URL: selector starts
  disabled and Metadata tab is active immediately, no interactive
  switch required.
- `node --check app.js` passes. Test data cleaned up, chromium/temp
  files removed.

**Status**: Complete.


## Updated: Version selector shows "N/A" (not just disabled-looking) on Metadata tab

Follow-up to "Metadata tab disables version selector" — user asked for it
to visually look disabled (not just unclickable) and, even better, show
"N/A" instead of a version value.

### Implementation
`registry/ui/app.js`:
- New shared helper `syncVersionSelectorForTab(tabKey)` (replaces the
  inline disable-only logic previously in `switchDocTab()`): on the
  `meta` tab, stashes the current selection in `sel.dataset.prevValue`,
  injects a temporary `<option value="__na__">N/A</option>`, selects it,
  and disables the control; on any other tab, removes that option and
  restores the previously-selected version.
- `switchDocTab()` now just calls `syncVersionSelectorForTab(tabKey)`.
- `loadVersionsForSelect()`'s async version-list fetch re-applies the N/A
  state afterward if the Metadata tab happens to already be active when
  it resolves (otherwise the freshly-populated real `<option>`s would
  silently replace the N/A placeholder).
- Initial render (landing directly on `tab=meta` via a restored URL)
  builds the `<select>` with the `N/A` option and `disabled` up front.

`registry/ui/style.css`: added `.eg-version-selector select:disabled`
(grey text/background, `cursor: not-allowed`) so the control looks
visually disabled, not just non-interactive.

### Verified via CDP (`dirs/d1/files/f1`, 2 versions)
- Picking version `1`, then switching to Metadata tab: selector shows
  "N/A", is disabled, computed style shows grey colors + not-allowed
  cursor.
- Switching back to "Version Details": selector correctly restores to
  version `1` (not reset to "Default").
- Landing directly on a `tab=meta` URL: selector starts as "N/A"/disabled
  immediately, no interactive switch needed.
- `node --check app.js` passes. Test data cleaned up, chromium/temp
  files removed.

**Status**: Complete.


## Updated: Fixed blank version selector after leaving Metadata tab on page load

### Bug
Loading/refreshing the page directly onto the Metadata tab (`tab=meta`
in the URL), then switching to the Document or Version Details tab,
left the version selector showing blank instead of "Default (1)".
Interactively selecting a version first (or navigating without a
refresh) masked the bug, since it went through a different code path.

### Root cause
1. The initial "start disabled on Metadata tab" render only injected the
   `N/A` option (real `default` option omitted) — already partially
   fixed earlier this session by always including the real `default`
   option underneath `N/A`.
2. Remaining bug: `loadVersionsForSelect()`'s async version-list fetch
   calls `syncVersionSelectorForTab('meta')` again to re-apply the N/A
   placeholder once real options are populated. But at the moment that
   runs, `sel.value` could still read `'__na__'` (the still-selected
   placeholder) — and since `dataset.prevValue` was undefined, the code
   stashed `'__na__'` itself as the "value to restore later", permanently
   losing the real previous selection.

### Fix
`registry/ui/app.js`, `syncVersionSelectorForTab()`: added a guard so
`'__na__'` itself is never captured as `dataset.prevValue` — only stash
`sel.value` when switching state, only if it isn't already `__na__`
`(sel.value !== '__na__')`. Without a real one to restore, switching away
now correctly falls back to whichever option the browser naturally
selects (the real "default" option), instead of nothing.

### Verified via CDP
- Direct page load on `tab=meta`, then switch to Document tab: now shows
  "Default (1)" (previously blank). Switching on to Version Details tab
  also shows "Default (1)" correctly.
- Regression check: interactively selecting version `1`, then Metadata
  tab, then back to Document tab: still correctly restores to version
  `1` (not reset to Default) — unaffected by this fix.
- `node --check app.js` passes. Test data cleaned up, chromium/temp
  files removed.

**Status**: Complete.

## Known non-gaps (design decisions made, not oversights)


- `newui.md` originally sketched a *nested* dropdown structure
  (Data→Export, Model→Model Source, Capabilities→Offered). This was
  deliberately replaced during the 2026-07-01 session with a flat
  "Registry Endpoints" tile/list (Model, Model Source, Capabilities,
  Capabilities Offered as siblings) after an explicit Option A vs.
  Option B discussion. Not a dropped requirement.

## Conventions

- Wrap text/comments in the `common/` directory and in this file
  (`plan.md`) to 80 characters per line.
