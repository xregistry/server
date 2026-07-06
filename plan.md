# xRegistry SPA UI — Cross-Session Plan / Outstanding Work

This file tracks design points and follow-up work for the `registry/ui/` SPA
that should persist across agent sessions (the per-session `todos` SQL table
does NOT survive between sessions — this file is the durable record). Update
it as items are completed or newly identified.

See also `newui.md` (original design draft from the "merge old UI into new
design" session, 2026-07-01) for full context on the overall UI redesign
goals.

## Outstanding

- [ ] **Add filter support to Grid (Tile) and List (Table) views for
  registry Data.** Today `_state.filters` is only wired into the JSON
  view's left panel (`buildAPIURL()` + the `lp-*` filter textarea).
  `newui.md`'s "Misc thoughts" section explicitly calls for filters to
  also work in Tile and Table views, not just JSON. Needs: a filter UI
  affordance in Grid/List headers (not gated to JSON-only), wired through
  the same `_state.filters` / query-string plumbing so Grid/List re-fetch
  with `?filter=...` applied. (Currently deferred — see "Filter builder
  UI" item below, which is being designed for JSON view first.)

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
