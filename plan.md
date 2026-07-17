# xRegistry SPA UI — Cross-Session Plan / Outstanding Work

This file tracks design points and follow-up work for the `registry/ui/` SPA
that should persist across agent sessions (the per-session `todos` SQL table
does NOT survive between sessions — this file is the durable record). Update
it as items are completed or newly identified.

See also `newui.md` (original design draft from the "merge old UI into new
design" session, 2026-07-01) for full context on the overall UI redesign
goals.

## Outstanding

- [x] ~~Registry name override on Config page.~~ **Done** (see Completed
  section below) — implemented 2026-07-09.

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

- [x] ~~Fix `navigateJsonUrl`/`syntaxHighlight`'s link reconstruction gap.~~
  **Done** (see Completed section below) — implemented 2026-07-09.

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

- [x] ~~Support `ifvalues` in `getAttr()` / `getAttrType()` /
  `getExplicitAttrType()` / `getExplicitAttrTypeAtPath()`~~ **DONE**
  (checklist item stale — implementation already completed in a later
  session; see the full "ifvalues" writeup further below, ~line 4000, for
  the `resolveIfValuesAttrs()`/`data`-threading design and
  `reconcileIfValuesOnChange()` live-reactivity follow-up). Verified by
  re-reading `registry/ui/app.js` ~line 5020-5065: `getAttr()` takes a
  `data` param and calls `resolveIfValuesAttrs(attrs, curData)` at every
  traversal depth; `getAttrType()`/`getExplicitAttrType()`/
  `getExplicitAttrTypeAtPath()` all thread the same `data` param through.

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

- [x] ~~New SPA lacks the old `?ui` HTML view's server-side proxy for
  remote registries~~ **DONE** (2026-07-12). Old UI's `/proxy?host=
  <remote>&path=<path>` (`registry/httpStuff.go:122`, `registry/
  ui.go:82-100,2919-2930`) is tightly coupled to the old HTML-templated
  `GenerateUI()` renderer and GET-only — unusable as-is for the new
  JSON-consuming SPA. Built a separate, new byte-level JSON reverse
  proxy instead:
  - **Design (confirmed with @duglin before implementing)**: proxying is
    always **explicit opt-in per server** (a "Use proxy" checkbox at
    add/edit time on the Config page) — no auto-detection of CORS vs.
    network failures, since both throw an identical generic `fetch()`
    error and can't be reliably distinguished.
  - **New file `registry/xrproxy.go`** — route `/xrproxy/
    <base64url(remoteOrigin)>/<rest-of-path>`. The remote origin is
    base64url-encoded (no padding) as a single path segment (not a query
    param) so any real xRegistry path/query can follow it exactly like a
    normal registry root, with zero collision risk and zero special-
    casing needed elsewhere in the SPA's URL-building code.
    `HTTPXRProxy()` forwards the request (method/body/headers minus hop
    headers) to the remote, reads the full response, `bytes.ReplaceAll`s
    the remote's own origin throughout the body to point back through
    the proxy, rewrites `Location`/`Content-Location`/`Link` headers the
    same way, and sets CORS headers. Wired into `registry/httpStuff.go`'s
    dispatcher alongside the existing `/proxy` route.
  - **JS side (`registry/ui/app.js`)**: `LS_PROXY` localStorage map
    (mirrors the existing `LS_NAMES` name-override pattern) +
    `isProxied()`/`setProxied()`; `serverFetchBase(url)` is the new
    central proxy-aware fetch-URL-base helper — `serverBase()` now
    delegates to it. Audited every one of the ~25 existing call sites
    using the raw `(_state.serverURL || window.location.origin)`
    pattern, fixing the real fetch-URL-builder gaps (`probeRegistry()`,
    `ensureModelCached()`, `ensureCapCached()`, `ensureOfferedCached()`,
    the post-save `/model` refetch in `saveModel()`) while deliberately
    leaving cache-key/identity usages (e.g. `_modelCache`/`_capCache`
    lookups) keyed by the **raw** server URL, not the proxy URL — getting
    this distinction right was the crux of the audit. `buildAPIURL()`/
    `buildBaseURL()` were already proxy-safe by construction. Links
    embedded in already-fetched JSON (metaurl, versionsurl, etc.) need no
    separate client-side rewriting since the Go proxy already rewrites
    all embedded absolute URLs server-side before the SPA ever sees them.
  - **Config page UI**: new "Proxy" column/checkbox per server row, "Use
    proxy" checkbox on the add-server form, `cfgSetProxied()`, and
    `cfgSave()`/`cfgAddNew()` updated to carry/set the flag.
  - **Bugs found and fixed via live testing** (against the real public
    `https://xregistry.soaphub.org` test registry):
    1. The proxy originally copied the remote's `Content-Length` header
       verbatim while writing a body that had just been rewritten to a
       different length — broke HTTP response framing (`Content-Length`
       mismatch, truncated/empty body despite `200 OK`). Fixed by
       excluding `Content-Length`/`Content-Encoding`/`Accept-Encoding`
       from blindly-copied headers and explicitly recomputing
       `Content-Length` from the final rewritten body.
    2. Under concurrent load (matching the SPA's real fetch pattern:
       several near-simultaneous requests per page), the remote
       intermittently reported `self`/`Link` with `http://` instead of
       `https://`. **Root-caused via a standalone Go reproduction**: the
       remote runs this same server's code, which falls back to sniffing
       an incoming `Referer` header's scheme (`https:`-prefixed vs. not)
       to guess its own scheme when `r.TLS` is nil (i.e. behind a
       TLS-terminating reverse proxy) — the exact same heuristic as this
       proxy's own `ourOrigin()`. Our proxy was forwarding the browser's
       own `Referer` (pointing at our SPA's `http://localhost:...` page)
       straight through, tricking the remote into reporting itself as
       `http`. Deterministic repro: `curl -H "Referer: http://x" https://
       xregistry.soaphub.org/`. Fixed two ways: (a) stopped forwarding
       `Referer` to the remote at all (added to the hop-header exclusion
       list — verified via a local header-echoing test server that the
       outgoing request now carries no `Referer`); (b) defensively made
       the origin-rewrite itself scheme-agnostic (matches both `http://`
       and `https://` variants of the remote host), so the symptom is
       masked even for well-behaved remotes with their own unrelated
       scheme quirks.
  - **Verified end-to-end via headless-Chromium/CDP automation**: added a
    proxied server via the Config page UI, confirmed `localStorage`
    state, then browsed into it through the SPA's normal navigation
    (List view root doc, drilling into a `schemagroups` collection with
    real nested data/counts/timestamps, JSON view) — all correctly
    routed through `/xrproxy/...` with no console errors and `self`
    links correctly rewritten to the local proxy prefix. All test
    artifacts (temp Chromium profile, scripts, screenshots, the
    temporarily-added test server entry) cleaned up afterward.

- [x] **"Server: <url>" info line under each page's title.** Every content
  page (Data collection/entity, Model, Model Source, Capabilities,
  Capabilities Offered, JSON view) now shows a small gray
  `Server: <url>` line so users always know which server they're
  looking at — always the real/raw server URL, never a proxy's
  base64-encoded `/xrproxy/...` path (relies on `_state.serverURL`
  always holding the raw URL, a deliberate design decision from the
  xrproxy work above).
  - `serverURLLineHtml()` (`registry/ui/app.js`) is the shared HTML-string
    generator; `.eg-server-url-line` (`registry/ui/style.css`) is the
    shared style (small `#888` text).
  - **Data pages** (`renderTableView()`): line inserted directly below the
    `.eg-page-title` div, for both the collection-page title and the
    single-entity-page title (e.g. below "REGISTRY: CloudEvents").
  - **Model/Model Source editor** (`renderEditor()`) and **Capabilities /
    Capabilities Offered list views** (`renderCapEditor()`,
    `renderCapabilitiesOfferedViewer()`): these pages have no page title,
    only a breadcrumb bar, so the line goes directly below the header's
    top breadcrumb bar, as the first child of the editor container —
    i.e. *before* the sub-breadcrumb bar (the "Registry..."/entity-type
    one), for consistency with the other page types. Has a
    `border-bottom` to visually separate it from the sub-breadcrumb/
    section boxes below (`#modelEditor > .eg-server-url-line`,
    `#capEditor > .eg-server-url-line` in `style.css`).
  - **JSON view** (`renderJSONView()`): rather than a separate line, the
    server URL is shown on the left edge of the existing sticky
    "&#9656; all" toolbar row (`.json-exp-wrap` changed to
    `display:flex; justify-content:space-between`), so it doesn't take
    extra vertical space.
  - Hidden on Home and Config pages (neither calls any of the above
    render functions).
  - Verified via headless-Chromium/CDP screenshots across Model,
    Capabilities, and Capabilities Offered list views, and JSON view.

- [x] **Disable empty (count-0) drill-down items in Model/Model Source
  left nav.** Items like "Attributes (0)", "Group Types (0)",
  "Resources (0)", "Version/Resource/Meta Attrs (0)" used to be
  clickable but just led to an empty list page — pointless. `navItem()`
  (`registry/ui/app.js`) now takes an optional `disabled` flag; when the
  underlying count is 0 the click handler isn't wired up and
  `.navItemDisabled` (`style.css`) grays out the text/arrow and disables
  hover/cursor. Actual group/resource entries (not count-based
  container links) are unaffected even if their own sub-counts are 0,
  since they still have a Details view worth visiting.

## Completed (for history / context)

- Redesigned the Home "Registries" **List view** as a card-list rather than
  a plain `<table>` (see chat 2026-07-09 discussion of List vs. Grid). New
  `.reg-list`/`.reg-row-*` CSS classes and a rewritten `renderHomeTable()`
  (`registry/ui/app.js`): each registry is now a rounded, shadowed row-card
  (favicon icon, bold name + description subtitle, Group Type pills,
  right-aligned URL), capped at `max-width:900px` so it doesn't look
  abandoned on wide viewports, with hover elevation. Error state ("failed
  to fetch") shows an inline red "Connection failed" pill (instead of a
  bare `!` dot) that still opens the existing error-detail popup on click.
  Only the Registries List view was touched — the Home "Group Types" flat
  list (`renderHomeFlatList()`) still uses the original `.home-table`/
  `.ht-*` classes untouched, and the Grid view (`renderHomeGrid()`/
  `serverCard()`) is unaffected. Fixed two bugs found while verifying via
  CDP screenshots: (1) `subEl.style.display = ''` doesn't override a CSS
  class's `display:none` (no inline style to clear) — changed to
  `'block'`; (2) `.reg-row-error { opacity: 0.85 }` on the row bled into
  the absolutely-positioned error popup child, washing it out — removed
  entirely since the red pill badge already communicates the error state.

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

## Registries List view: group-types above URL, wrapping description,
## URL tooltip

Follow-up polish to the earlier List-view card-list redesign
(`.reg-list`/`.reg-row` in `registry/ui/style.css`, `renderHomeTable()`
in `registry/ui/app.js`).

**Changes**:
- `renderHomeTable()`: wrapped `.reg-row-groups` (group-type pills) and
  `.reg-row-url` (server URL) in a new `.reg-row-side` column container
  so the pills now stack ABOVE the URL instead of being squeezed
  side-by-side. Added a `title="<full url>"` attribute on the URL
  element so the full address is always available on hover even if
  ellipsis-truncated.
- `style.css`: new `.reg-row-side` (flex column, right-aligned, ~280px
  max-width — kept compact per the user's preference rather than
  widening it, so pills wrap across multiple lines when needed instead
  of taking more horizontal space). `.reg-row-groups`/`.reg-row-url`
  dropped their own independent `max-width` values (now inherit the
  parent's width). `.reg-row-sub` (description subtitle) changed from
  single-line `nowrap` + ellipsis to a 2-line `-webkit-line-clamp`
  clamp. Updated the `@media (max-width: 768px)` mobile override to
  target `.reg-row-side` as a whole (stacking pills/URL left-aligned)
  instead of the old two-sibling selector.
- `app.js`'s probe callback: `subEl.style.display` changed from
  `'block'` to `'-webkit-box'` to match the new line-clamp's required
  `display` value (setting plain `'block'` would have silently
  defeated the clamp).

**Verified** via headless-Chromium CDP against a live xrserver (root
registry given a long description, a 4-group-type model, plus a
deliberately-unreachable second registry for the error-state row):
- Pills render stacked above the URL, right-aligned, within a compact
  column; description clamps to exactly 2 lines with a trailing
  ellipsis.
- Hovering the URL shows the full address via the new `title` tooltip.
- Mobile viewport (375px): row stacks correctly, pills/URL become
  left-aligned as a block.
- Regression check: Grid view (`renderHomeGrid()`/`serverCard()`) and
  the Home "Group Types" flat list (`renderHomeFlatList()`) are
  unaffected.
- `node --check app.js` passes. Test data (root description, group-type
  model, fake unreachable registry entry) reverted/cleaned up; chromium
  test processes and temp profile/log files removed.

**Status**: Complete.

## Group-type pill font size increased for readability

Bumped the shared `.group-type-item` class (`registry/ui/style.css`)
from `font-size: 10px` to `12px` (matching the `.reg-row-sub`
description text size), with padding nudged from `1px 5px` to
`1px 6px` to match. This class is shared across three places, all of
which benefit from the change: the Registries List-view row pills
(`.reg-row-groups`), the Registries Grid-view card pills
(`serverCard()`), and the Home "Group Types" flat-list table's
Resource Types column.

**Verified** via CDP screenshots of both List and Grid views with a
3-group-type test model — pills are noticeably easier to read and
nothing overflows, wraps awkwardly, or crowds the layout. Test data
reverted, chromium test processes cleaned up.

**Status**: Complete.

## Group-type pills now link to their respective collections

Made the `.group-type-item` pills clickable — they now navigate straight
to that collection (e.g. clicking "dirs (3)" browses to `/dirs` on that
server) instead of being plain inert text.

**Changes** (`registry/ui/app.js`):
- New shared helper `groupTypePillHTML(serverUrl, c)` renders a pill as
  an `<a class="group-type-item">` with a real `href` (via `buildURL()`,
  so right-click/open-in-new-tab and hover-preview work) and an
  `onclick` wired to the existing `browseGroupCollection()` navigation
  helper (same one already used by the Group Types page's name links) —
  guarded via `guardedOnclick()` for modifier-key/middle-click support.
- `renderHomeTable()`'s probe callback (`.reg-row-groups` pills, List
  view) and `probeAllCards()`'s probe callback (`.server-card-groups`
  pills, Grid view) both switched from inline `<span>` markup to calling
  the new shared helper.
- The Home "Group Types" flat list (`renderHomeFlatList()`) was left
  alone — its own group-type name is already a link (via the same
  `browseGroupCollection()`), and its pills column shows *resource
  types* within a group definition (not live, directly-browsable
  collections at that scope), so linking those wouldn't be equivalent.

**Changes** (`registry/ui/style.css`):
- `.group-type-item` (now an `<a>`, not a `<span>`): added
  `text-decoration: none`, `cursor: pointer`, `display: inline-block` to
  neutralize default anchor styling, plus a `:hover` state (darker
  background + underline) so it reads as clickable without changing the
  pill's resting appearance.

**Verified** via CDP: pill `href`/`onclick` both point at the correct
collection URL; clicking a pill (List view) navigates to that
collection's table view (`/dirs`, shows "No items found" for an empty
test collection as expected); same pill link works identically in Grid
view; resting appearance of pills unchanged in both views. Test model
data reverted, chromium processes cleaned up.

**Status**: Complete.

## Fixed vertical misalignment of breadcrumb pill / copy-URL icon

### Bug
The Registries/Group Types toggle pill (Home page) and the copy-API-URL
icon button (data pages) both sat noticeably lower than the logo/"/"
separator and the header's other icon buttons (grid/list/json/edit/
gear) — their bottoms didn't line up.

### Root cause
`#breadcrumbs` had a `padding-top: 11px` rule with no accompanying
`padding-bottom`. Since `#breadcrumbs` is itself a flex container
(`align-items: center`) sized to its tallest child (the pill or the
copy-button, both ~22-24px), that top padding became part of the
container's own box height, and got added *below* the vertically-
centered children rather than symmetrically around them — so taller
elements (pill, copy button) ended up flush with the container's
bottom edge, well below where the shorter logo/icon-button siblings
(centered within the shared header height) landed.

### Fix
`registry/ui/style.css`: removed the stray `padding-top: 11px` from
`#breadcrumbs`, then (per follow-up feedback — the user preferred all
breadcrumb content bottom-aligned, not centered) changed `#breadcrumbs`
from `align-items: center` to `align-items: flex-end`. With both
changes, the pill, copy button, "/" separator, and plain breadcrumb
text all share the exact same bounding-box bottom edge — within ~1-2px
of the logo's bottom (previously off by 6.5-7.5px). The remaining 1-2px
is normal font/icon rendering noise (smaller than the header's own
gear/edit icon offset from the logo, ~4-5px) — not a leftover
alignment bug. Any perceived difference between the "/" character and
lettered text (e.g. "xRegistry") beyond that is due to the slash glyph
having no descender (unlike letters such as "g"/"y"), so its ink sits
slightly higher within an identical box — a typographic characteristic
rather than a CSS issue.

### Verified via CDP
Measured/screenshotted bounding rects before and after on: the Home
page's Registries/Group Types pill (bottom offset from logo went from
~6.5px to ~1px), a data page's copy-URL button (from ~7.5px to ~2px),
plain breadcrumb text, a nested-path breadcrumb, and a narrow (375px)
mobile viewport — all align cleanly with no wrapping regressions.
Confirmed via rect measurements that "/", breadcrumb text, and the
copy button now share an identical bottom coordinate under
`align-items: flex-end`. `node --check app.js` passes (CSS-only
change). Chromium test processes cleaned up.

**Status**: Complete.

## Group-type pill font size + spacing + clickable links + breadcrumb
## vertical alignment (small polish batch)

- `.group-type-item` pill font-size bumped from 10px to 12px (padding
  1px 5px -> 1px 6px) to match the description text size, since group
  types can be important info users shouldn't have to squint at. Shared
  by List rows, Grid cards, and the Home "Group Types" flat list.
- `.reg-row-side`'s gap (between the group-type pills and the server
  URL, Registries List view) increased from 4px to 8px so they read as
  clearly separate, not on the verge of overlapping.
- Group-type pills are now clickable links to their respective
  collections, in both List and Grid views of the Registries home page
  (new shared helper `groupTypePillHTML(serverUrl, c)` in `app.js`,
  reused by `renderHomeTable()` and `probeAllCards()`). The Home "Group
  Types" flat list's own resource-type pills were deliberately left as
  plain (non-clickable) `<span>`s since they aren't live/browsable
  collections at that scope.
- Fixed vertical misalignment of the breadcrumb "Group Types" pill and
  copy-URL icon: `#breadcrumbs` had a stray `padding-top: 11px` with no
  matching bottom padding, pushing its tallest children (the pill /
  copy button) down relative to shorter siblings in other flex
  containers (e.g. the logo). Removed the padding. Per follow-up user
  preference, `#breadcrumbs`'s `align-items` was then changed from
  `center` to `flex-end` so all breadcrumb children (separator, text,
  pill, copy button) share an identical bounding-box bottom. A residual
  visual offset between the "/" separator and lettered breadcrumb text
  is a font-rendering artifact (the "/" glyph has no descender) — not a
  layout bug; confirmed via CDP rect measurements that all bottoms are
  mathematically identical.

**Status**: Complete.

## Home "Group Types" page: card-list redesign (List view)

**Context**: this page lists group types across all known registries.
Per user clarification, rows here are **never merged/deduplicated**
across registries — each row is one specific group type as it exists
in one specific registry. Two registries with an identically-named
group type (e.g. both have a "dirs" group type) still get two separate
rows, since like-named group types could have entirely different model
definitions, and since models can change at any time a merge could
become stale (and un-merging later would confuse users more than never
merging in the first place). This is also why each row shows exactly
ONE server URL/registry, never a list of several. This design mirrors
the Registries List redesign (each row = one distinct registry, no
merging there either) rather than being a classic report/index table
with comparable cross-row data.

**Change**: `renderHomeFlatList()` (`app.js`) rewritten from a plain
`<table class="home-table">` to a `.gt-list`/`.gt-row` card-list,
matching the Registries List's `.reg-list`/`.reg-row` visual language:
- `.gt-row-icon` — the owning registry's favicon/icon.
- `.gt-row-main` — group-type name (clickable link to its collection,
  same target as clicking it elsewhere) + item count, and an optional
  description subtitle (2-line clamp, only rendered when non-empty).
- `.gt-row-side` — resource-type pills (`.group-type-item`, kept as
  plain non-clickable `<span>`s — not live collections at this scope)
  stacked above a small owning-registry link (icon + name, navigates to
  that registry's home via the same `buildURL()`/`doBrowse()` pattern
  already used by the Registries List's own row-name link).

Removed now-orphaned `.home-table`/`.ht-name` (bare)/`.ht-desc`/
`.ht-url`/`.ht-groups`/`.ht-groups-inner` CSS rules from `style.css`
(confirmed via grep no JS still references them). Kept `.ht-name-link`/
`.ht-name-text`/`.ht-loading`/`.ht-action` since they're still used
elsewhere (`.ht-name-text`/`.ht-name-link` are shared with the
Registries List's own row-name class list; `.ht-loading` is still used
by this same page's async-loading placeholder).

**Verified** via headless-Chromium CDP against a live xrserver (2 test
registries, one with a "dirs" group type overlapping the other's "dirs"
group type but a different description/item-count, confirming the
no-merge design renders as 2 separate rows): group-type name link
navigates to the correct collection; registry-owner link navigates to
that registry's home page; resource-type pills render as plain
non-clickable spans; mobile viewport (375px) stacks sensibly (pills/
registry link left-aligned below the main content); Registries List
and Grid views (unrelated pages sharing some CSS classes) are
unaffected — regression-checked via screenshot. `node --check app.js`
passes. Test data reset, chromium processes and temp files cleaned up.

**Status**: Complete.

## Group-type / resource-type pill hover help (descriptions)

- Registries List/Grid views: each `.group-type-item` pill (e.g.
  "dirs (3)") now has a `title` attribute showing that Group Type's
  model description, if any, via the shared `groupTypePillHTML()`
  helper (`app.js`). No visual change when a Group Type has no
  description (no empty `title=""` emitted).
- Home "Group Types" page: each resource-type pill in a row's
  `.gt-row-resources` now also has a `title` attribute showing that
  Resource Type's model description. This required widening
  `probeRegistry()`'s per-collection `c.resources` from a plain sorted
  array of plural-name strings to an array of `{plural, description}`
  objects (only consumed by `renderHomeFlatList()`'s pill rendering —
  the separate `c.resources` usage in `renderSingleEntity()`'s own
  in-registry Group Types table, which just comma-joins plural names,
  is a different call site and was left untouched).

**Verified** via headless-Chromium CDP against a live xrserver with a
model that gave both a group type ("dirs") and its resource type
("files") explicit descriptions: confirmed `title` attributes render
correctly on the Registries List pill, the Registries Grid pill, and
the Group Types page's resource-type pill. `node --check app.js`
passes. Test data reset, chromium processes and temp files cleaned up.

**Status**: Complete.

## Group Types page: small polish (mouse cursor + duplicate icon)

- Resource-type pills (`.gt-row-resources .group-type-item`, plain
  non-clickable `<span>`s) were incorrectly showing a pointer cursor
  and a link-style hover (background/underline), because the shared
  `.group-type-item` class had `cursor: pointer` unconditionally, even
  though only the Group Type pill itself (an `<a>`) is actually
  clickable. Fixed by moving `cursor: pointer` and the hover styling to
  a scoped `a.group-type-item` selector in `style.css`, so the plain
  `<span>` resource pills no longer look like links.
- Removed the redundant registry icon (`.gt-row-reg-icon`) from each
  row's right-side owning-registry link — the row already shows that
  registry's icon on the left (`.gt-row-icon`), so showing it a second
  time on the right was unnecessary. The link now shows just the
  registry name. Removed the now-unused `.gt-row-reg-icon` CSS rule.

**Verified** via headless-Chromium CDP: `getComputedStyle(...).cursor`
confirmed `auto` on resource pills vs `pointer` on Group Type link
pills; confirmed `.gt-row-reg-icon` count is 0 after the change and a
screenshot shows a clean single-icon-per-row layout. `node --check
app.js` passes. Chromium processes and temp files cleaned up.

**Status**: Complete.

## Breadcrumb copy-URL button: two-line hover tooltip

- The copy-to-clipboard button after the breadcrumbs (`.bc-copy-btn`)
  now shows a 2-line native tooltip: "Copy API URL for this data" on
  the first line, and a live preview of the exact URL that will be
  copied on the second line (via a literal `\n` inside the `title`
  attribute, which browsers render as a tooltip line break). The
  preview is computed with the same `buildTabAwareAPIURL()` helper the
  click handler already uses, so what's shown always matches what gets
  copied (including Resource/Version pages' tab/version-aware URL).

**Verified** via headless-Chromium CDP: `title` attribute on `.bc-copy-
btn` reads `"Copy API URL for this data\nhttp://localhost:8080..."` on
both a collection page and the registry root page. `node --check
app.js` passes. Chromium processes and temp files cleaned up.

**Status**: Complete.

## Copy-URL tooltip: fixed staleness on tab switch + wrong $details guess

Two follow-up bugs found in the tooltip preview added above (the actual
copy behavior — `copyCurrentAPIURL()` — was always correct; only the
displayed preview text was wrong):

- **Stale on tab/version switch**: switching the Document/Details tab
  bar (`switchDocTab()`) or the Resource page's version-selector
  dropdown (`onVersionSelectChange()`) changed what would actually get
  copied, but neither call re-rendered the breadcrumb bar, so the
  tooltip text kept showing whatever was true at page-load time. Fixed
  by adding `refreshCopyLinkBtnTooltip()` (updates just the button's
  `title` in place) and calling it from both functions.
- **Wrong guess for Document-tab-first resources**: `renderBreadcrumbs()`
  (and thus the tooltip's first computation) runs before the entity's
  tab bar exists in the DOM and before the model fetch resolves, so
  `buildTabAwareAPIURL()`'s fallback (when no `.eg-doc-tab.active`
  element exists yet) had to guess which tab would end up default-
  active. It always guessed "Version Details"/"Version" (`$details`-
  suffixed URL), which is wrong for any resource type with
  `hasdocument: true` — those show the Document tab first (see
  `tabDefs.push()` ordering), so the correct default preview is the
  plain, un-suffixed document URL. Fixed the fallback to check
  `resourceHasDocument(model, path)` (using `_modelCache`) and guess
  `'doc'` when true. Additionally added a `refreshCopyLinkBtnTooltip()`
  call right after `renderSingleEntity()` writes the real tab bar into
  the DOM (depth 4/6+ only), so the tooltip is always corrected to
  match the real default-active tab once it's known, regardless of
  whether the fallback guess was right.

**Verified** via headless-Chromium CDP with a `hasdocument: true`
resource with 2 versions: initial tooltip on both the Resource page and
a Version page now shows the plain document URL (no `$details`) since
Document is the true default tab; manually cycling through Details/
Metadata/Document tabs and the version-selector (regression) still
produces the correct URL for each combination (`$details` for Version
Details, `/meta` for Metadata, plain for Document, `/versions/N` for a
selected version). `node --check app.js` passes. Test data reset,
chromium processes and temp files cleaned up.

**Status**: Complete.

## Removed dead registry-switcher dropdown (`#reg-select`/`#section-select`)

**Found by user report**: a green, empty `<select>` dropdown briefly
flashed to the right of the "xR" logo on every page refresh, replaced
almost instantly by the real breadcrumbs. Root cause: `index.html` had
a `<select id="reg-select">` element with no default `display:none` —
it painted with its full green pill styling (`#reg-select` CSS rule)
for one frame before `renderHeader()` (in `app.js`) ran and force-hid
it via `el('reg-select').style.display = 'none'` on every render.

Investigating further confirmed this dropdown (and a sibling
`#section-select`) were **entirely dead code** — a leftover
registry/section switcher from an old header design, fully superseded
by the current breadcrumbs + Home-page navigation:
- `#reg-select` was unconditionally hidden on every `renderHeader()`
  call and never shown again anywhere.
- Its only populate function, `buildServerDropdown()` (and its helper
  `addOption()`), was never called from anywhere else in `app.js`.
- Its `onchange` handler, `onRegistryChange()`, was therefore also
  unreachable dead code.
- `#section-select` had no HTML element at all anymore (already
  removed at some earlier point) — only dead CSS rules and a dead
  `onSectionChange()` handler remained for it.
- A related one-line dead reference, `el('view-toggle') && (...)` in
  `renderHeader()` (no `#view-toggle` element exists either), was
  cleaned up in the same pass since it's the same category of stale
  header-redesign leftover.

**Removed**: the `<select id="reg-select">` element (`index.html`);
`buildServerDropdown()`, `addOption()`, `onRegistryChange()`,
`onSectionChange()`, and the dead `el('reg-select')`/`el('view-toggle')`
lines in `renderHeader()` (`app.js`); the `#reg-select`/`#section-select`
CSS rules, including their `:hover`/`option` variants and the mobile
media-query override (`style.css`). `serverLabel()` (also defined near
`buildServerDropdown()`) was kept — it's still actively used elsewhere
(e.g. Registries List row names).

**Verified** via headless-Chromium CDP: `document.getElementById(
'reg-select')` is now `null`; screenshot of the header on page load
shows no flash and no layout regression. `node --check app.js` passes;
grepped `app.js`/`index.html`/`style.css` for all removed identifiers to
confirm zero remaining references. Chromium processes and temp files
cleaned up.

**Status**: Complete.

## Spec-defined `icon` attribute: image preview in Property table

**User request**: when the spec-defined `icon` attribute (not any
extension attribute that happens to share the name "icon") has a value,
show it as an actual image thumbnail in the Properties table's value
cell, not just a plain clickable URL link.

**Implementation** (`registry/ui/app.js`, `buildPropsRowsHtml()`): added
an `else if` branch (before the final generic `else`) that checks `k ===
'icon' && specLevel && specLevel.icon && typeof val === 'string' &&
val.trim()` — reusing the existing `specAttrLevel(path)` per-depth map
(already used by `isSpecAttr()`) to distinguish a true spec attribute
from a coincidentally-named extension attribute at levels where `icon`
isn't spec-defined (e.g. Meta/depth 5 — confirmed via `specattrs.js`
that `SPEC_ATTRS.meta` has no `icon` key). When true, renders a small
`<img class="eg-icon-preview" onerror="this.style.display='none'">`
thumbnail inside a `<span class="eg-icon-preview-wrap">`, followed by
the existing link/text rendering (`renderScalarValue()`) unchanged —
thumbnail is additive, not a replacement.

`registry/ui/style.css`: added `.eg-icon-preview-wrap` (inline-flex, 8px
gap) and `.eg-icon-preview` (20x20px, `object-fit:contain`, rounded
corners, vertical-align middle) near `.eg-value`.

**Verified** via CDP against a live xrserver: valid icon URL shows a
correctly-sized thumbnail next to the link on the Registry root page;
broken/unreachable icon URL correctly hides the `<img>` via `onerror`
(no broken-image glyph, no layout gap) while the link text remains;
confirmed (via `specattrs.js` inspection) that Meta-level (depth 5) has
no spec-defined `icon`, so an extension attribute named "icon" there
would correctly fall through to plain link rendering, per the user's
explicit requirement. Entity Grid view was confirmed already fully
removed in an earlier session (see "Grid view removed..." entries
above), so `buildPropsRowsHtml()` is the only properties-rendering path
— no other code path needed the same treatment. `node --check app.js`
passes. Test data (`icon` field) cleared, chromium test process and
temp files cleaned up.

**Status**: Complete.

## Icon propagation from model + entity data (Model/Resource/Group Type
## icons everywhere)

**User request** (2026-07-09): when a Group Type or Resource Type
definition in the model has an `icon` URL (a model-schema field, e.g.
`groups.<plural>.icon`/`groups.<plural>.resources.<plural>.icon` — per
core/spec.md model schema, distinct from an instance's own spec-defined
`icon` attribute), show it in the Model/ModelSource viewer next to the
Group/Resource type entry, and propagate icons everywhere a Group/
Resource list or header is shown: Group Types list (Registry page),
Groups list page (header + rows), Resources list within a Group
Instance page (header + rows), Resource list page (header + rows),
Resource instance page header, and Versions (reusing the owning
Resource's resolved icon — no separate Version Type icon concept).
Precedence everywhere: an instance's own `icon` attribute wins; else
fall back to the model Type's `icon`.

**Implementation** (`registry/ui/app.js`):
- New shared helpers (near `getSingularName()`/`specAttrLevel()`):
  `modelGroupIcon(model, groupPlural)`, `modelResourceIcon(model,
  groupPlural, resPlural)` (read the model's Type-level icon),
  `resolveGroupIcon(model, groupPlural, groupData)`,
  `resolveResourceIcon(model, groupPlural, resPlural, resourceData)`
  (apply the instance-icon-wins-else-model-icon precedence), and
  `iconThumbHtml(url, cssClass)` (shared `<img onerror=...>` builder
  used by every HTML-string rendering spot below).
- **Model/ModelSource viewer** (`buildLeftNav()`): new `withIcon(label,
  iconUrl)` local helper wraps a nav-item label with a leading icon
  thumbnail (DOM-element label, since `navItem()` already supports
  non-string labels) — applied to the Group Types list and the
  Resources list inside a Group Type. Also added a live-updating icon
  thumbnail preview next to the "Icon URL" field in both the Group Type
  and Resource "Details" forms (new `efIconPreview(row)` helper wrapping
  `ef()`, reusing the `.eg-icon-preview` class from the Properties-table
  feature) — updates on `input`, hides via `onerror` when empty/broken.
- **Registry page Group Types list** (`probeRegistry()` +
  `renderHomeFlatList()`): `probeRegistry()` now also attaches `c.icon`
  (Group Type's model icon) and per-resource `icon` (Resource Type's
  model icon) alongside the existing description fields;
  `renderHomeFlatList()` renders both as `row-icon-thumb` thumbnails
  next to the Group Type name and each resource-type pill.
- **Groups/Resources/Versions collection pages** (`renderTableView()`,
  depths 1/3/5): page-title icon (`eg-page-title-icon` class, reusing
  the pre-existing but previously-unwired CSS rule of that name) uses
  `modelGroupIcon()` at depth 1, `modelResourceIcon()` at depth 3 and 5
  (Versions has no separate Type-level icon — reuses the owning
  Resource Type's). Each row's `cell-id` gets a `row-icon-thumb` via
  `resolveGroupIcon()` (depth 1) / `resolveResourceIcon()` (depth 3) /
  `modelResourceIcon()` fallback-only (depth 5, per user's explicit
  "use whatever Resource icon is defined" direction — no per-Version
  instance-icon check).
- **Registry root / Group Instance single-entity pages**
  (`renderSingleEntity()`, depths 0/2/4/6+): page-title icon added via
  `resolveGroupIcon()` (depth 2), `resolveResourceIcon()` (depth 4), or
  `modelResourceIcon()` fallback-only (depth 6+, Version pages — same
  "no per-Version icon check" rule as above). The Group Types /
  Resources collection-refs table (shown at depth 0 and depth 2) also
  gets a `row-icon-thumb` per row via `modelGroupIcon()` (depth 0 rows
  are Group Types) / `modelResourceIcon()` (depth 2 rows are Resource
  Types) — these rows are Type listings, not instances, so no instance-
  level fallback check applies there.
- `registry/ui/style.css`: added `.row-icon-thumb` (16x16, inline,
  margin-right) for table rows/pills/nav labels; reused the existing
  (previously dead/unwired) `.eg-page-title-icon` rule for page titles.

**Verified** via CDP against a live xrserver with a test model (Group
Type "endpoints" + icon, Resource Type "messages" + icon) and instances
covering every precedence combination (a Group/Resource with no own
icon vs. one with its own override icon): confirmed correct icon +
correct precedence (own-icon-wins-else-model-fallback) at every listed
spot — Registry root Group Types table, Groups list page (header +
both rows), Group Instance page (header + Resources row) for both the
fallback and override Group instances, Resources list page (header +
both rows), Resource instance page header for both the fallback and
override Resource instances, Versions list page (header + row), Version
instance page header, and the Model/ModelSource viewer's Group Types/
Resources nav-item icons plus the live Icon-URL field preview. `node
--check app.js` passes. Test model/instance data reverted (`PUT
/modelsource` `{}`), chromium test process and temp files cleaned up.

**Status**: Complete.

## Follow-up: Home "Group Types" page uses the Group-type icon, not the
## owning Registry's icon

**User request** (2026-07-09, same session as the icon-propagation
feature above): on the Home "Group Types" page, the main row icon
(`.gt-row-icon`, the larger icon at the left of each row) should prefer
the model-declared Group-type icon over the owning Registry's icon.

**Implementation** (`registry/ui/app.js`, `renderHomeFlatList()`): the
row's main `<img class="gt-row-icon">` src is now `r.icon || r.regIcon
|| 'favicon.svg'` (was `r.regIcon || 'favicon.svg'`) — `r.icon` (the
Group-type's model icon) already flows in from `probeRegistry()` (added
in the icon-propagation feature above). Also added an inline
`onerror="this.onerror=null;this.src='favicon.svg'"` fallback so a
broken/unreachable Group-type icon URL falls back to the local favicon
rather than showing a broken-image glyph (`r.regIcon` itself was never
onerror-guarded before, so this also improves that pre-existing case).
Removed the now-redundant small `row-icon-thumb` that had been added
next to the Group Type name in the same feature (duplicate icon, same
pattern as the earlier "Group Types page: small polish… duplicate
icon" fix) — the resource-type pills' own thumbnails are unaffected.

**Verified** via CDP: a Group Type with its own model icon shows that
icon in `.gt-row-icon` (not the Registry's); a Group Type with no icon
falls back to the Registry's icon logic (here, the local favicon, since
the test registry had no icon set either); a broken Group-type icon URL
correctly falls back to `favicon.svg` via `onerror` rather than showing
a broken-image glyph. `node --check app.js` passes. Test model/instance
data reverted, chromium test process and temp files cleaned up.

**Status**: Complete.

## Registry root page header shows the Registry's own `icon`

**User request**: "if the registry has an icon defined, let's add it to
the header on the Registry page" — followed by a clarifying "after the
name" (initially interpreted literally as icon-after-text), then
corrected: "I was wrong, it should be on the left, like all other header
icons... with the title text" (i.e. same leading-icon position as
Group/Resource/Version headers), followed by "can you ensure they're
bottom (base) aligned?".

**Implementation** (`registry/ui/app.js`, `renderSingleEntity()`'s shared
title-building block): added a `depthH === 0` branch to `titleIconUrlH`
reading `data.icon` directly (Registry root has no model-level Type icon
to fall back to, unlike Group/Resource/Version). The icon renders via the
same `iconThumbHtml(titleIconUrlH, 'eg-page-title-icon')` call already
used at every other depth, in the same leading position (before the
ID-prefix/"TYPE:"/name) — no special-casing needed once the "after the
name" idea was reverted.

**Alignment fix** (`registry/ui/style.css`): `.eg-page-title` was
`align-items: center`, which vertically centered the 24px icon against
the title text's line-box, looking slightly off since the icon and text
don't share a true baseline. Changed to `align-items: flex-end` so the
icon and text bottoms are flush — applies to all depths sharing this
title bar (Registry/Group/Resource/Version), not just the Registry root.

**Verified** via CDP against a live xrserver: setting the registry's
`icon` (via `PATCH /`) shows it at the front of the "REGISTRY: xRegistry"
title, bottom-aligned with the text; clearing `icon` back to `''`
correctly shows no image (empty-string guard already in place). `node
--check app.js` passes. Test data reverted, chromium processes and temp
files cleaned up.

**Status**: Complete.

## Follow-up: `.eg-page-title` header icon/text alignment — settled on
## uppercase-everywhere + center-aligned (not baseline)

The `align-items: flex-end`/`baseline` experiments above were revisited
after further feedback. Pixel measurement (crop + numpy analysis of a
CDP screenshot) confirmed the CSS baseline math was in fact working
correctly (icon and text bottom edges landed on the exact same pixel
row), but the specific xRegistry logo SVG used for testing has an
asymmetric shape (the "R"'s leg / "X"'s tail dip unevenly), so it never
looks quite flush at the bottom regardless of the CSS rule — a red
herring caused by that particular icon's artwork, not a CSS bug.

Given that, the user asked to instead: 1) make `.eg-page-title` itself
`text-transform: uppercase` (removing the now-redundant declaration from
the nested `.eg-page-title-type` rule) so *all* header text — type label
prefix, ID prefix (e.g. resource id before "VERSION"), and the
name/id value — is consistently uppercased, since previously only the
"TYPE:" label was uppercase while the name/id stayed mixed-case, an
inconsistency across different header layouts (single-entity headers
show "TYPE: Name", collection-list headers show only the plural type
label); 2) remove `align-items: baseline` from `.eg-page-title` (reverted
to `align-items: center`, which was the original rule before this whole
investigation) — visually "some are still off a bit but overall it looks
better" than baseline once everything is uppercase.

**Implementation** (`registry/ui/style.css`): `.eg-page-title` gained
`text-transform: uppercase` and `align-items: center` (was `baseline`);
`.eg-page-title-type` had its own `text-transform: uppercase` removed
(now redundant/inherited from the parent). No `app.js` changes needed —
this is a pure CSS/typography change, the icon/name/id HTML structure is
unchanged.

**Verified** via CDP across Registry root (depth 0), Group instance
(depth 2), Resource instance (depth 4), Groups list (depth 1), and
Resources list (depth 3) pages, each with a model/instance/registry
`icon` set: all headers now show fully uppercase text ("REGISTRY:
XREGISTRY", "ENDPOINT: ENDPOINT ONE", "ENDPOINT EP1 MESSAGES", "MESSAGE:
M1") with the icon vertically centered against the text. `node --check
app.js` passes (no JS changes, but re-verified after the CSS edit per
routine). Test model/instance/registry-icon data reverted, chromium
processes and temp files cleaned up.

**Status**: Complete.

## Registry name override on Config page

**User request**: "add the ability for a user to set the name of a
registry on the config page, so the UI will use that name instead of
the one returned by the server" (originally added to the Outstanding
list 2026-07-09, implemented same day).

**Implementation** (`registry/ui/app.js`):
- New localStorage key `LS_NAMES` ('xreg-name-overrides') holding a
  `{normalizedURL: customName}` map, alongside the existing `LS_SERVERS`/
  `LS_OPTIONS` keys. Helpers: `loadNameOverrides()`, `saveNameOverrides()`,
  `getNameOverride(url)`, `setNameOverride(url, name)` (empty name
  deletes the override, reverting to the server-provided name).
- `serverLabel(url)` — the single shared function already used by
  breadcrumbs, the Registries Grid/List views' sort order and initial
  link text — now checks `getNameOverride()` first, before the probed
  `_labelCache` name, before the raw URL fallback. Since nearly every
  "registry name" display already routed through this one function, most
  of the propagation was free.
- Two probe-callback spots that later overwrite the name text after a
  live fetch (`probeAllCards()` for Grid view, `renderHomeTable()`'s row
  probe) were guarded with `!getNameOverride(...)` so they don't clobber
  the override once the real server response arrives.
- Registry root page header (`renderSingleEntity()`'s depth-0
  `titleDisplayH`): now prefers the override over `data.name`/
  `data.registryid`. The existing "strip a trailing ' Registry' word"
  heuristic (e.g. "CloudEvents Registry" → "CloudEvents", avoiding a
  redundant "REGISTRY: CloudEvents Registry") is now only applied to the
  server-provided fallback name, never to a user's explicit override —
  the user already controls exactly what text to type, so an override
  literally containing the word "Registry" is shown verbatim, not
  silently trimmed.
- Config page (`renderConfig()`): the "Name" column is a
  display-span/hidden-input pair (`cfgNameCellHTML(url)`, mirroring the
  pre-existing URL display/input pair), toggled together with the URL
  pair by one shared Edit/Save/Cancel button set per row — **not** a
  save-on-blur input. This was revised from an initial always-live/
  save-on-blur design after user feedback that it felt risky ("neat,
  but scary — someone could change it by accident"). The local "this
  server" row now also gets its own Edit/Save/Cancel (no Delete),
  covering Name only since its URL is architecturally fixed
  (`window.location.origin`). The "Add server" row gained an optional
  `Name (optional)` input alongside the URL field, so a custom name can
  be set at add-time (`cfgAddNew()` calls `setNameOverride()` only if a
  name was entered). `cfgEdit()`/`cfgSave()`/`cfgCancel()` were
  generalized to toggle/save/cancel whichever of {name, url} pairs
  exist in a row — local rows only have a name pair, other rows have
  both, and the Enter/Escape keydown handlers on each input still work
  the same way they did for URL-only editing before. The probe callback
  still never overwrites a saved override or an in-progress edit —
  updated to keep the display span's text in sync with the probed name
  only when there's no override and the row isn't currently mid-edit
  (in addition to always updating the input's `placeholder`).
- `cfgResetAll()` ("Clear All") now also clears `LS_NAMES`.
  `cfgResetExceptServers()` ("Clear All Except Registry Locations")
  deliberately leaves `LS_NAMES` alone, same as `LS_SERVERS` — a name
  override is metadata about a saved registry location, not a general
  "option", so it's preserved together with the server list it's keyed
  against.
- CSS (`registry/ui/style.css`): `.cfg-name-input` now looks like the
  existing `.cfg-url-input` (bordered text box) since it's only ever
  visible during an active edit, not "always shown, styled to look
  passive" as in the original save-on-blur design. `.cfg-name-display`
  needs no styling of its own (plain text, same as `.cfg-url-display`).

**Verified** via CDP against a live xrserver: local row's Edit → type
name → Save correctly sets the override (confirmed via
`getNameOverride()`) and updates the display span in place; Edit → type
→ Cancel correctly discards the change (display reverts to the prior
saved value, nothing persisted). Added a new server with a name via the
Add row's optional Name field — row rendered with that name
immediately, no separate edit step needed. Edited an existing
user-added row's Name and URL together in one Edit/Save cycle — both
new values took effect (row's `data-cfg-url` and displayed name updated
correctly, `removeServer`/`addServer` + `setNameOverride` all called
with the new URL as key). Delete still works via the unaffected
`cfgDelete()` path. `node --check app.js` passes throughout.

**Status**: Complete.

## `navigateJsonUrl`/`syntaxHighlight`/`renderUrlLinkValue` link
## reconstruction gap — done

**Problem** (deferred from 2026-07-07, fixed 2026-07-09): when a user
clicked a same-server URL string value shown either in JSON view's raw
text (`syntaxHighlight()`) or as a clickable Property-table value
(`renderUrlLinkValue()`, e.g. `self`/`metaurl`/`versionsurl`/any
URI-typed attribute), the click handler (`navigateJsonUrl()`) only
extracted `path` + `filter=` from the clicked link via `filtersFromUrl()`
and let `buildAPIURL()` reconstruct the rest of the fetch URL from
generic `_state` fields — silently dropping any OTHER query param the
server-provided link carried (e.g. a hypothetical `sort=`/`inline=` baked
into a nested collection's `xxxurl` that didn't match the current page's
own active state). The two hover-href builders (`syntaxHighlight()`'s
`fakeSt`, `renderUrlLinkValue()`'s `fakeSt`) had the same issue one level
earlier: neither ever set `apiURL` on their synthetic state object, so
`buildURL()`'s `apiurl=` permalink param was never populated at all for
these hrefs — only a `filter=` param, derived from `st.filters`, ever
showed up.

**Root-cause discussion** (talked through with user before implementing):
confirmed the URL being parsed (`raw`/`inner`/`rawText`) is always a
literal string pulled directly out of the server's own response (JSON
text or an entity's data field) — i.e. it already **is** the correct,
fully-computed link to the next page, including whatever filter the
server subsetted for that specific target. There was never a real reason
to drop it and let generic state reconstruction re-derive an equivalent
URL: `filtersFromUrl()`'s only genuine job is populating the Filter/Sort
panel's displayed state, not building the fetch URL. Since
`pushStateReal()` already calls `syncFiltersFromApiURL()` unconditionally
right after merging in any patch — deriving `_state.filters` from
whatever real `apiURL` was just set — passing the clicked link through
verbatim as `apiURL` (exactly like every other real-link navigation call
site in the file, e.g. lines ~1076/1782/6715) makes the separate
`filters: filtersFromUrl(raw)` in `navigateJsonUrl()`'s own `pushState()`
patch redundant; it was removed.

**Implementation** (`registry/ui/app.js`):
- `navigateJsonUrl()`: now passes `apiURL: raw` in its `pushState()`
  patch instead of `filters: filtersFromUrl(raw)`; `_state.filters` gets
  synced automatically afterward by `pushStateReal()`'s existing
  `syncFiltersFromApiURL()` call.
- `syntaxHighlight()` / `renderUrlLinkValue()`: their `fakeSt` objects
  (used only to compute the displayed hover/right-click href via
  `buildURL()`) now also set `apiURL: inner`/`apiURL: rawText`
  (verbatim), in addition to the pre-existing `filters:
  filtersFromUrl(...)`. The `filters` field is still needed here
  specifically — unlike `navigateJsonUrl()`, `fakeSt` is a one-off object
  that never goes through `pushStateReal()`, and `buildURL()` has a
  dedup check (`filterAlreadyInApiURL`) that compares `st.filters`
  against what's embedded in `st.apiURL` to avoid appending a redundant,
  stale top-level `filter=` — so `filters` must already match what
  `apiURL` carries for that check to correctly suppress the duplicate.
  Setting `apiURL` is what actually fixes the bug (every query param now
  flows into `buildURL()`'s `apiurl=` permalink param, not just filter).
- Updated/corrected surrounding comments (`syncFiltersFromApiURL()`'s doc
  comment referenced navigateJsonUrl's old "no apiURL of its own"
  behavior, now stale) to reflect the fixed behavior.

**Verified** via CDP: called `navigateJsonUrl()` directly with a
same-server URL carrying `filter=`, `sort=`, and an extra non-standard
`foo=bar` param — confirmed `_state.apiURL` retained the exact original
URL string (all three params intact), `_state.filters` correctly
auto-derived to just the filter group, and the address bar's `apiurl=`
param round-tripped the same full URL with no duplicate top-level
`filter=`. Also called `renderUrlLinkValue()` and `syntaxHighlight()`
directly with the same test URL and confirmed both produced an `<a>`
href whose `apiurl=` carries every original query param verbatim, with
matching `onclick="navigateJsonUrl(...)"` targets. `node --check app.js`
passes. Chromium test processes and temp files cleaned up.

**Status**: Complete.

---

## Fixed: registry-root breadcrumb loses/mis-shows a filter applied at depth 0

**Bug reported**: starting at the registry root, applying a filter directly
there (e.g. `schemagroups.schemas.versions.epoch`), then drilling all the
way down to a deep child page, then clicking the ROOT breadcrumb (only the
root — every other breadcrumb level worked fine) did not restore the
filter. Separately, while deep in the hierarchy, hovering the root
breadcrumb showed the wrong (deeper page's already-relativized) filter
value instead of the root's own.

**Root cause**: `crumbURLs` (the per-depth cache of each visited ancestor's
real/filtered URL, used to restore breadcrumb clicks correctly) is only
ever indexed for depths > 0 — both of `pushStateReal()`'s caching blocks,
and the `defaultApiURL` computation that feeds a breadcrumb click's patch,
were explicitly gated with `newDepth > 0` / `_state.path.length > 0`. This
was a deliberate simplification based on an assumption (documented right
next to `buildURL()`'s `apiurl=` gate) that "the registry root's own URL is
always trivially `serverBase()`" — true only for an UNFILTERED root.
Once a filter is applied directly at the root, that assumption breaks:
the root's own filtered URL is never cached anywhere, so:
- `buildBreadcrumbSegments()`'s root segment hardcoded `pageHref([], '',
  ...)` for its hover-href — passing a falsy `apiURL` meant `pageHref()`'s
  `st.filters = filtersFromUrl(st.apiURL)` reassignment never fired,
  silently falling back to whatever `_state.filters` currently held (the
  CURRENT/deepest page's already-relativized filter) — explaining the
  "wrong filter shown on hover" symptom.
- Clicking the root breadcrumb's onclick patch (`{path:[], section:'data',
  ...}`, no `filters`/`apiURL` keys) let `pushStateReal()`'s default-patch
  merge fill in `filters: []`/`apiURL: ''` (since `defaultApiURL` was
  always `''` for depth 0) — wiping the filter entirely.
- `buildURL()`'s `apiurl=` param is also gated the same way, so the root's
  filtered state is only ever round-tripped to the address bar via the
  plain top-level `filter=` param — which is why a manual page **refresh**
  at the root correctly restored the filter (`loadStateFromURL()` reads
  `filter=` directly) even though the breadcrumb-click path had nothing to
  fall back on.

**Fix** (`registry/ui/app.js`): added a new `_state.rootApiURL` field —
crumbURLs' depth-0 counterpart, since crumbURLs itself is only indexed for
depths > 0:
- Initialized to `''` in the main `_state` object declaration.
- `loadStateFromURL()`: synthesizes it via `buildAPIURL()` whenever landing
  directly on the (data-section) root, so a same-session round-trip works
  even without ever clicking Apply again in that session (e.g. a
  bookmarked filtered-root URL).
- `pushStateReal()`: `defaultApiURL`'s computation now falls back to
  `_state.rootApiURL` for `newDepth === 0` (mirroring the `crumbURLs[newDepth
  - 1]` lookup used for depths > 0); both existing apiURL-caching blocks
  (the "path is changing" block and the "Apply changed apiURL without a
  path change" block) now also populate `_state.rootApiURL` at depth 0,
  alongside their existing depth > 0 `crumbURLs` writes; `rootApiURL` is
  reset to `''` alongside `crumbURLs = []` wherever that cache is fully
  invalidated (server/section change, entering data fresh).
- `buildBreadcrumbSegments()`'s root segment now passes `_state.rootApiURL
  || ''` (instead of a hardcoded `''`) to `pageHref()` for the hover-href.
  The onclick patch itself needed NO change — it already omits `apiURL`
  the same way every other depth's breadcrumb onclick does, relying
  entirely on `pushStateReal()`'s `defaultApiURL` fallback to supply the
  correct cached value; this keeps the root breadcrumb consistent with how
  every other depth already works.

**Verified** via headless-Chromium CDP against a live xrserver (nested
`schemagroups`→`schemas`→`versions` test model, filter
`schemagroups.schemas.versions.epoch>0` applied at the root):
- Fresh page load directly on the filtered root correctly synthesizes
  `_state.rootApiURL`.
- Real DOM clicks drilling all the way down to the `versions` collection,
  then a real click on the actual root breadcrumb `<a>` (both in List view
  and in JSON view, starting the drill-down via real JSON-embedded link
  clicks), correctly restored `_state.path = []`, `_state.filters =
  ["schemagroups.schemas.versions.epoch>0"]` (the root-scoped value, not a
  deeper relativized one), and `_state.apiURL` matching the real filtered
  root URL.
- The root breadcrumb's hover-href (`buildBreadcrumbSegments()`'s output)
  was confirmed to show the correct root-scoped filter while deep in the
  hierarchy, not the deeper page's relativized value.
- `node --check app.js` passes. Test model/data reset to `{}`; chromium
  test processes and temp profile/log files cleaned up.

**Status**: Complete.

---

## Fixed: malformed version-link URLs corrupting filter after a breadcrumb click

**Bug reported**: after applying a root-level filter (e.g.
`schemagroups.schemas.versions.epoch`) and drilling all the way down to a
specific version, clicking a GROUP INSTANCE breadcrumb (e.g. `Contoso.ERP`)
lost the filter entirely; its hover-href showed a truncated/wrong filter
(`filter=epoch` instead of the correct `filter=schemas.versions.epoch`).
Every other breadcrumb level worked correctly.

**Root cause**: `versionURLById()` and `defaultVersionURL()` (used to build
the version-picker/"Default Version" links shown on a Resource's detail
page, and by `navigateToVersionById()`/`navigateToVersion()`) constructed a
version's URL by blindly appending `'/' + encodeURIComponent(vid)` onto a
COLLECTION url (`d.versionsurl`, `versionsColl.url`, or the cached
`crumbURLs[4]`) that may already carry a `?filter=...` query string (e.g.
`.../versions?filter=epoch>0`). Appending after the query string produces
a badly malformed URL like `.../versions?filter=epoch>0/v1` — the `/v1`
segment lands INSIDE the query string's value instead of extending the
path. This corrupted URL then became `_state.apiURL`/`crumbURLs[5]` for
the version page, and once `syncFiltersFromApiURL()` (or downstream
breadcrumb filter derivation) parsed it, the resulting filter value was
itself corrupted (`"epoch>0/v1"` as a single bogus filter clause) —
explaining both the "vanishes" and "wrong value shown" symptoms once that
bad state propagated into further filter-relativization logic.

**Fix** (`registry/ui/app.js`): added a shared `appendURLPathSegment(url,
seg)` helper that splits any URL's query string off BEFORE appending the
new path segment, then re-attaches the query string unchanged — instead of
each of the three affected call sites hand-rolling
`url.replace(/\/$/, '') + '/' + encodeURIComponent(seg)` (which silently
assumed the URL never carried a query string). Updated `defaultVersionURL()`
and both fallback branches in `versionURLById()` to use it.

**Verified** via headless-Chromium CDP against a live xrserver (same
`schemagroups`/`Contoso.ERP`/`schemas`/`s1`/`versions`/`v1`/`v2` test model
as the root-breadcrumb fix above, filter
`schemagroups.schemas.versions.epoch>0` applied at the root):
- Before the fix: the resource detail page's `v1`/`v2` quick-links had
  hrefs like `.../versions?filter=epoch>0/v1` (query string BEFORE the id) —
  reproduced exactly. After the fix: correctly
  `.../versions/v1?filter=epoch>0` (id in the path, filter as the query).
- After the fix, drilling all the way down to `v1` via a real click on that
  corrected link, then a real click on the `Contoso.ERP` breadcrumb,
  correctly restored `_state.path = ["schemagroups","Contoso.ERP"]` and
  `_state.filters = ["schemas.versions.epoch>0"]` (previously lost/
  corrupted) — confirmed with `node --check app.js` passing.
- Separately noted (not fixed, flagged to @duglin): drilling down via
  **JSON view's own embedded `self` links** (rather than List view's
  filter-augmented row links — see `entityHrefWithFilter()`) naturally
  loses the filter at entity-instance depths, since a `self` link never
  carries a filter per spec (confirmed: `GET .../Contoso.ERP` never
  returns a `filter=` anywhere in its own response). This is a distinct,
  pre-existing JSON-view-only gap, not the bug just fixed.
- Test model/data reset to `{}`; chromium test processes and temp profile/
  log files cleaned up.

**Status**: Complete (version-link fix). JSON-view self-link filter gap
noted separately, not yet addressed — pending @duglin's input on whether
it's worth fixing.

## Fixed: wrong breadcrumb filter at entity-instance depths skipped during
## JSON-view collection-URL-only navigation

**Bug reported** (@duglin): starting at a registry root with filter
`schemagroups.schemas.versions.epoch` applied, then drilling down in JSON
view by clicking ONLY `<plural>url` collection links (never `self`) —
e.g. `schemagroupsurl` → the nested `schemasurl` embedded inside the
`Contoso.ERP` entity's own data (without ever separately visiting/clicking
`Contoso.ERP`'s own page) — the `Contoso.ERP` breadcrumb's hover URL showed
`filter=epoch>0` instead of the correct `filter=schemas.versions.epoch>0`
(the same value as its parent `schemagroups` breadcrumb). The same wrong-
value pattern also affected the `s1` resource-instance breadcrumb one level
further down (`filter=epoch>0` instead of the correct `versions.epoch>0`).
@duglin clarified this navigation pattern (collection-URL-only, never
`self`) is the way he actually browses — invalidating an earlier, incorrect
diagnosis of this same symptom that assumed `self`-link clicks were
involved.

**Root cause**: `crumbURLs[]` (the per-depth cache of each visited depth's
real server URL, used both for breadcrumb hover hrefs via `pageHref()` and
for restoring `_state.filters`/`apiURL` on an actual breadcrumb click via
`pushStateReal()`) only gets a cache entry for a depth when that depth's
own page is actually visited/rendered. Jumping straight from one collection
link to a NESTED collection link (`navigateJsonUrl()`'s multi-segment jump)
skips over the intermediate entity-instance depth entirely — its
`crumbURLs` slot is left empty. Both `pageHref()` (hover href) and
`pushStateReal()`'s `defaultApiURL` computation (actual click/restore) then
had no real URL to derive that depth's filter from, and silently fell back
to whatever `_state.filters` happened to hold for the CURRENT/deepest page
being viewed — an unrelated, generally-wrong value.

**Key insight used for the fix**: an entity-instance depth's own filter
context is always IDENTICAL to its immediate parent COLLECTION's filter,
unchanged/un-relativized — crossing from a collection into one of its
member entities never trims a filter clause; relativization
(`FiltersRelativeToAbstract()` server-side) only happens when stepping INTO
a nested collection. Verified against real HTTP fetches and against the
already-correct List-view `entityHrefWithFilter()`-computed value for the
same entity.

**Fix** (`registry/ui/app.js`):
- `pageHref()`: when the destination is an entity-instance depth (`path.
  length` even, > 0) with no `apiURL` of its own, derive `st.filters` from
  the immediate parent collection's cached `_state.crumbURLs[path.length -
  2]` (if known) instead of leaving it to inherit `_state.filters` from the
  current page. Fixes the breadcrumb's hover/ctrl-click/copy-link href.
- `pushStateReal()`: added the identical fallback when computing
  `defaultApiURL`/the patch defaults — when `defaultApiURL` is unknown for
  an even `newDepth`, default `filters` (not `apiURL`) to
  `filtersFromUrl(_state.crumbURLs[newDepth - 2])`. `apiURL` itself is
  intentionally left `''` in this case (not backfilled to the parent's real
  URL, which would point at the WRONG endpoint) — `buildAPIURL()`'s
  existing no-real-link fallback already re-appends `_state.filters` onto
  a plain `serverBase() + path.join('/')` construction, and `refresh()`'s
  existing `needsDetails()`/`fetchWithDetailsFallback()` machinery already
  handles appending `$details` for resource/version depths independently
  at fetch time — confirmed via curl that a plain (non-`$details`) GET on
  a resource entity returns its document content-negotiated view, not JSON
  metadata, so this existing fallback machinery being reused here (rather
  than reinvented) is important for correctness.

**Verified** via headless-Chromium CDP against a live xrserver
(`schemagroups`/`Contoso.ERP`/`schemas`/`s1`/`versions` v1,v2 test data,
filter `schemagroups.schemas.versions.epoch>0` applied at root, JSON view,
navigating via collection-URL clicks only): all 5 breadcrumb depths now
show the correct filter (root: none shown in this direct-pushState test
setup — unaffected/pre-existing; `schemagroups`: `schemas.versions.
epoch>0`; `Contoso.ERP`: `schemas.versions.epoch>0`, was wrongly `epoch>0`
before the fix; `schemas`: `versions.epoch>0`; `s1`: `versions.epoch>0`,
was wrongly `epoch>0` before the fix). Confirmed actually CLICKING (not
just hovering) both the `Contoso.ERP` and `s1` breadcrumbs correctly loads
their pages with `_state.filters` restored to the right value, no error,
and each entity's own nested `<plural>url` link in the resulting JSON
correctly shows the properly-relativized filter one level down. `node
--check app.js` passes. Test model reset to `{}`, chromium test processes
and temp profile/log files cleaned up.

**Status**: Complete.

## Added: auto-expand Filters section on navigation when filters are set

JSON view's left-panel Filters section (twisty-collapsible, always defaults
to collapsed on first ever page load) now also auto-expands whenever a
FRESH navigation (different server/section/path — see `fbKey()`) lands on
a page that already has one or more filters active (e.g. followed a
filtered link, typed/pasted a `filter=`-bearing URL, or restored via a
breadcrumb click) — no point landing on an already-filtered page with that
fact hidden behind a collapsed twisty. Still defaults to collapsed when a
fresh navigation has no filters, and does not fight the user's own manual
collapse/expand toggle, or auto-re-expand, for edits made while staying on
the same page (applying a new filter via the wizard without navigating
away doesn't rebuild the draft, so doesn't touch the collapse state either)
— matches the existing "reset only on a fresh navigation" pattern already
used for `_fbDraft`/`_sortDraft`.

`registry/ui/app.js`, `ensureFbDraft()`: added
`_filtersCollapsed = _state.filters.length === 0;` inside the existing
draft-rebuild block (only runs once per fresh navigation, when `fbKey()`
changes).

Verified via headless-Chromium CDP: fresh nav to a path with a filter
already set → twisty shows expanded (▼); fresh nav to a different path
with no filter → collapsed (▶); manually collapsing a filtered page then
adding another filter chip in the same-page draft (no navigation) does not
force it back open. `node --check app.js` passes.

**Status**: Complete.

## Fixed: Version Details/Metadata property tables silently truncated
## long values instead of scrolling horizontally

**Bug reported** (@duglin): on the Resource page's List view, both the
"Version Details" and "Metadata" tabs' tables chopped/truncated wide text
instead of letting the panel scroll horizontally. Same problem on the
Version page.

**Root cause**: `buildEntityPropsTableHtml()`/`renderMetaTable()`'s Property
tables share the generic `.xr-table` class, whose default `td` rule
(`max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space:
nowrap;`) is appropriate for the main collection List/Grid tables (fixed
column widths, deliberately truncated) but was also silently truncating
this Property table's "Value" column — the one column most likely to hold
long text (URLs like `self`/`versionsurl`, long custom string attributes,
etc.) — with no way to see the full value or scroll to it.

**Fix** (`registry/ui/style.css`, `registry/ui/app.js`):
- Added `.xr-table-props td:last-child` override (mirrors the existing
  `.cell-tree` override used for object/array values) removing the
  max-width/ellipsis truncation for the Value column, so the cell/table can
  grow as wide as its content needs.
- Added `overflow-x: auto` to `.eg-doc-tab-panel` so the resulting
  horizontal scrollbar is scoped to just the active tab's content, not the
  whole page (`#main-view` already had its own `overflow: auto`, but
  scrolling at that level would carry the page title/tab bar along
  sideways too).
- The Version page's depth≥6 single-tab special case (no tab bar, content
  rendered directly — see `renderSingleEntity()`) wasn't wrapped in
  `.eg-doc-tab-panel` at all; wrapped it in that same class purely to pick
  up the `overflow-x: auto` scoping.

**Verified** via headless-Chromium CDP at a narrow viewport (500px) with a
long custom string attribute value: both the Resource page's "Version
Details" and "Metadata" tabs, and the Version page's "Version Details" tab,
now report `panel.scrollWidth > panel.clientWidth` with `overflow-x: auto`
and the value cell computed as `white-space: nowrap; text-overflow: clip`
(no longer `ellipsis`) — confirming the long value is fully rendered and
horizontally scrollable rather than chopped. `node --check app.js` passes.
Test model/data reset to `{}`, chromium test processes and temp profile/log
files cleaned up.

**Status**: Complete.

## Added: auto-expand Filters section on navigation when filters are set

`ensureFbDraft()` now sets `_filtersCollapsed = _state.filters.length === 0`
inside its once-per-navigation rebuild block (keyed by `fbKey()` = server +
section + path, deliberately excluding filters so same-page filter edits
never trigger this). Effect: landing on a fresh page with an active filter
auto-expands the JSON-view Filters section; landing with none keeps it
collapsed; manually toggling/editing filters on the same page is left
alone. Verified via CDP with real model/data. Status: Complete.

## Added: commit (⏎) button on Filter Builder's custom-attribute-name input

**Bug reported** (@duglin): after picking "(other / custom attribute)" in
the Filter Builder and typing a name, there was no visible way to confirm
you were done — only worked by blurring the field or pressing Enter
(`onchange`), with zero visual affordance.

**Fix** (`app.js` `fbGenericSegRow()`, `style.css` `.fb-seg-custom-commit`):
added a small `⏎` button right after the custom-name `<input>`, disabled
while the field is empty, enabling live as you type (`fbSegCustomInput()`
toggles it via `oninput`, no full rerender/focus-steal). Clicking it
(`fbCommitSegCustom()`) reads the input's current value and calls the same
`fbSetSegCustom()` the existing `onchange` already used. Covers all
"(other/custom attribute)" entry points (root/group/resource attribute
pickers, map keys, array indices) since they all share this one renderer.
Verified end-to-end via CDP: disabled→enables on typing→click commits and
advances to the operator picker. Status: Complete.

## Fixed: ctrl/cmd/shift/middle-click didn't open a new tab for several
## link types

**Bug reported** (@duglin): testing ctrl-click consistency across the app,
found `self`/`defaultversionurl` (Property-table URL values) didn't open a
new window/tab.

**Root cause found in 3 places**, all missing the `navShouldDefault()` /
`guardedOnclick()` guard used everywhere else (breadcrumbs, rows, tiles):
1. `renderUrlLinkValue()` — every same-server URL-typed scalar shown in a
   Property table (`self`, `defaultversionurl`, `schemasurl`-as-scalar,
   etc.) had `onclick="return navigateJsonUrl(...)"` with no modifier-key
   check, so it always intercepted the click regardless of ctrl/cmd/shift.
2. `syntaxHighlight()` — the JSON-view raw-JSON linkifier had the identical
   unguarded `onclick="return navigateJsonUrl(...)"` for every same-server
   URL string value rendered inline (e.g. `self` shown literally in the
   JSON body).
3. The left-panel "Viewing:" nav section (Data/Export/Model/Model
   Source/Capabilities/Capabilities Offered) rendered as plain
   `<span onclick="pushState(...)">` with no `href` at all — never had
   ctrl-click support to begin with (not a regression, just never built).

**Fix**: (1) and (2) now build their `onclick` via
`guardedOnclick('navigateJsonUrl(...)')` instead of a bare
`return navigateJsonUrl(...)` call. (3)'s "Viewing:" nav items
(`buildRegEndpointPillsHtml()`'s sibling code in the left panel plus
`lpNavPairRow()`) were converted from `<span onclick>` to real
`<a href=... onclick=guardedOnclick(...)>`, using `pageHref()`/`buildURL()`
for the href — same treatment breadcrumbs/rows/tiles/the "Config:" pills
already had. Discussed and confirmed with @duglin that this is the right
call specifically *because* switching "Viewing:" section is a genuinely
different resource/endpoint (Data vs Model vs Capabilities — different API
endpoint entirely), unlike the Resource/Version page's Version
Details/Metadata/Document tabs, which deliberately stay plain tabs (no
ctrl-click) since they show different facets of the *same* already-fetched
entity and don't even touch the breadcrumbs/URL identity the way a real
section switch does.

Audited the rest of the app for the same gap (breadcrumbs, home-page
registry cards/rows, group-type pills on the Group Types page, and the
root page's "Config:" endpoint pills) — all already correctly guarded, no
further gaps found.

**Verified** via CDP: confirmed real `guardedOnclick`/href on all of the
above, and simulated an actual `ctrlKey: true` MouseEvent dispatch on both
the JSON-view `self` link and the "Viewing: Model" nav link — confirmed
`_state`/`view` stayed unchanged (native browser new-tab behavior wins),
where before the fix the same click would have intercepted and navigated
in-app. `node --check app.js` passes. Chromium test processes/profiles
cleaned up, test modelsource reset to `{}`.

**Status**: Complete.

## Open discussion: collection `*url`/`*count` mismatch (still undecided);
## `self`-filter-context and `ancestorurl` questions now resolved

Extended `[[PLAN]]`-mode discussion with @duglin (no code changes) about
three related xRegistry spec questions raised while testing filter/
breadcrumb behavior and the `ancestor`→`ancestorid` rename:

1. **Collection `*url`/`*count` mismatch — still undecided.** When a
   root-level filter's dotted `<PATH>` doesn't reference a sibling group
   type (e.g. `schemagroups.schemas.versions.epoch` applied at root), that
   sibling's `*count` correctly shows `0` (confirmed this matches a strict
   reading of the spec's Filter Flag section — not a bug), but its `*url`
   carries no filter at all, so following it returns the FULL unfiltered
   collection — a misleading `count:0`/`url-returns-everything` mismatch.
   Explored synthesizing an always-empty-but-valid filter clause (e.g.
   `epoch<0`, using `epoch`'s spec-mandated `UINTEGER` type) for such
   `*url`s instead of inventing new reserved syntax; a bare `filter=null`
   idea was floated but flagged as risky since `null` already means
   "attribute absent" in filter grammar, so a distinct token like
   `filter=none`/`filter=false` was suggested instead. **Still deciding —
   no code/spec changes yet.**
2. **`self` carrying filter/query context — decided: NO.** Explored
   whether `self` should be allowed to carry filter/query context the same
   way `<plural>url` attributes do (immutable base path + optional query
   suffix). Worked through the mechanics (relativization would only ever
   add `<PATH>`-prefixed descendant-scoping fragments, never a bare
   no-`<PATH>` fragment, since that's a spec-mandated existence gate —
   404 if it doesn't match, per spec.md ~line 4054), and floated candidate
   alternate-attribute names (`contexturl`/`sourceurl`) for carrying that
   context separately from `self`. **@duglin has since decided against
   this — `self` will NOT carry any filter info, staying strictly
   xid-based and immutable per the current spec wording.** No further
   action needed; this matches the existing implementation
   (`registry/jsonWriter.go`'s `self` getter has no filter logic today).
3. **`ancestorurl` spec gap — decided: NOT adding it.** The version
   `ancestor`/`ancestorid` attribute has no matching `ancestorurl` link
   attribute, unlike other id/url attribute pairs — @duglin had noted this
   as a gap to raise with the other spec authors. **Decided: not adding
   `ancestorurl`.** No further action needed.

Item 1 remains open; items 2 and 3 are now closed/decided — no code or
spec changes required for either.



- `newui.md` originally sketched a *nested* dropdown structure
  (Data→Export, Model→Model Source, Capabilities→Offered). This was
  deliberately replaced during the 2026-07-01 session with a flat
  "Registry Endpoints" tile/list (Model, Model Source, Capabilities,
  Capabilities Offered as siblings) after an explicit Option A vs.
  Option B discussion. Not a dropped requirement.

## Fixed: Config-page breadcrumb text sat higher than other pages'
## breadcrumb text (confirmed against the xR logo too)

**Bug**: the "Config" breadcrumb (top of the Config page) rendered its
text a few pixels higher than the equivalent last-segment text on every
other page, and higher relative to the xR logo icon too (confirmed via
side-by-side screenshots).

**Root cause**: `#breadcrumbs { display:flex; align-items:flex-end; }`
anchors all breadcrumb children to the row's bottom edge. On normal
pages, the row's rendered height is 24px, set by its tallest child —
the copy-API-URL `<button class="bc-copy-btn">` that
`renderBreadcrumbs()` appends after the last segment. The Config page
has no copy button at all (`showCopyLinkBtn()` explicitly excludes the
`'config'` view), so its breadcrumb row was only 15px tall (bare text
only). Measured `.bc-current` rects: normal page `top:17, bottom:32`
(bottom-anchored within the 24px row); Config page (before fix)
`top:12.5, bottom:27.5` (bottom-anchored within its own shorter 15px
row) — same text height, different row height under the same
`flex-end` rule, hence the ~4.5px offset.

**Fix**: added `min-height: 24px` to `#breadcrumbs`
(`registry/ui/style.css`), keeping `align-items: flex-end` unchanged.
This makes the Config page's row reserve the same 24px box even
without a button present, so its text bottom-aligns to the identical
position as every other page — Config moved down to match the normal
pages/logo, not the other way around.

**Verified** via headless-Chromium CDP: `.bc-current`
`getBoundingClientRect()` now reports identical `top:17, bottom:32` on
both a normal data page and the Config page (previously Config was
`top:12.5, bottom:27.5`). Side-by-side header screenshots confirm
"Config" now sits at the same visual baseline as "xRegistry" and the
xR logo. `node --check app.js` passes (CSS-only fix). Test chromium
process/profile/log files cleaned up.

**Status**: Complete.

## Fixed: "Connecting…" stuck forever on unresponsive registries (no
## error shown) — missing fetch timeout in probeRegistry()

**Bug**: a registry whose host is unresponsive (DNS resolves but the
server never answers, e.g. firewalled/down — reported for the "Bad
URL" test registry `http://xregistry.soaphub.org/model`) left its
Home-page card/row stuck on "Connecting…" forever, with no red error
badge/text.

**Root cause (confirmed with real `curl -v` against the actual host,
plus a headless-Chromium simulation)**: `curl -v` showed DNS resolving
fine (`208.97.140.135`) but the TCP connect itself timing out (SYN
never ACKed) — a genuine network-level hang, not a fast DNS/CORS
rejection and not the already-working "response is valid JSON but not
a Registry entity" error path (`Not a valid xRegistry (missing
specversion or registryid)`, still fires correctly and fast when a
server actually responds with the wrong document type). `probeRegistry()`
(`app.js` ~2026) had no timeout anywhere in its fetch pipeline — it
relied purely on the browser's own fetch promise settling. Since
`capP.then(...)` gates the entire rest of the function (including the
`Promise.all(...).catch(cb-with-error)` that produces the red
badge/text), a hung `/capabilities` fetch meant that error path never
ran; the card stayed on its initial "Connecting…" HTML indefinitely
(confirmed via a headless-Chromium test overriding `fetch` to return a
never-settling `Promise`).

**Fix**: added a `fetchWithTimeout(url, ms)` helper (`app.js`, just
above `probeRegistry()`) that wraps `fetch()` with an `AbortController`
+ `setTimeout`, rejecting with a distinguishable `Error('Connection
timed out')` if the request doesn't settle within
`PROBE_FETCH_TIMEOUT_MS` (8000ms, per @duglin). Used it for all three
of `probeRegistry()`'s fetches — `/capabilities`, `/model`, and the
main `/` request (inlined the equivalent of `fetchJSON()`'s
ok-check/error-throw logic since `fetchJSON()` itself is left
unchanged/untouched — this fix is scoped to Home-page registry
probing only, not the app's general-purpose fetch path, so it can't
affect normal data-fetching elsewhere).

**Verified** via headless-Chromium CDP:
- A simulated abort-respecting hung fetch now surfaces "Connection
  timed out" in the card's error text after ~16-18s (two sequential
  8s timeout stages: `/capabilities` first, then the main `/` fetch,
  since the main fetch only starts once `/capabilities` settles) —
  a huge improvement over hanging forever.
- A normal, healthy registry (`http://localhost:8080`) still probes
  and renders correctly (name/label resolved, no error) — no
  regression for the working case.
- `node --check app.js` passes.

Test chromium processes/profiles/log files cleaned up after each run.

**Status**: Complete.

## Added: `ifvalues` support in model-attribute-lookup functions

**Problem**: `getAttr()`/`getAttrType()`/`getExplicitAttrType()`/
`getExplicitAttrTypeAtPath()` only walked the *static* model declaration
of attributes, so any attribute that only exists conditionally via a
model `ifvalues` rule (e.g. an Endpoint's `envelope: "CloudEvents/1.0"`
triggering a sibling `envelopeoptions` attribute, or `protocol:
"AMQP/1.0"` triggering `protocoloptions` — see
`spec/endpoint/model.json`) was invisible to these functions. This meant
such attributes never got correct type-driven rendering (e.g. monospace
for non-string types) in List view's Property tables, meta table, or the
JSON-tree renderer — they fell back to being treated as untyped
extension attributes. Same underlying mechanism covers `messagegroups`/
`messages` (per @duglin) and any other model using `ifvalues`.

**Root cause / design**: mirrors the server's own dynamic-attribute
algorithm, `Attributes.AddIfValuesAttributes()`
(`registry/shared_model.go` ~1560): for each attribute with a non-empty
`ifvalues` map, look up the actual value in the entity's real JSON data;
if it matches (case-insensitively) one of the `ifvalues` keys, merge
that key's `siblingattributes` into the effective attrs map — and check
newly-added siblings' own `ifvalues` too (recursive). The client-side
model walk had no access to the entity's actual data at the point these
lookup functions ran, so it could never evaluate these conditional
rules.

**Fix** (`registry/ui/app.js`):
- New `resolveIfValuesAttrs(attrs, data)` — non-destructive (shallow-
  copies, never mutates the cached model), JS port of the server
  algorithm above.
- `getAttr(model, entityPath, attrKeyPath, data)` — added a `data`
  param; at each traversal depth, resolves ifvalues via
  `resolveIfValuesAttrs()` (using the actual data at that nesting level)
  before doing the key lookup, and drills `data` alongside the attr-map
  traversal for nested paths.
- `getAttrType()`, `getExplicitAttrType()`, `getExplicitAttrTypeAtPath()`
  — all threaded the same `data` param through to `getAttr()`/the attrs-
  map resolution.
- `renderValueTree()` — added a `rootData` param (the entity's top-level
  JSON, held constant through the whole recursion, since
  `getExplicitAttrTypeAtPath()` already does its own full drill-down
  from the top using the complete key path at every call).
- Updated all call sites to pass the actual entity data through:
  `buildPropsRowsHtml()` (has `entityData`), `renderMetaTable()`/
  `renderMetaContent()` (have `d`). While touching `renderMetaContent()`,
  also fixed a pre-existing (unrelated to ifvalues, but on the exact
  line being edited) bug: its `renderValueTree()` call used `_state.path`
  instead of the meta-level path (`_state.path.concat(['meta'])`) — wrong
  depth for meta-level nested-object type resolution. Introduced a
  `metaPath` var and fixed both that call and its neighboring
  `getExplicitAttrType()` call to use it consistently.

**Verified**: `node --check app.js` passes. Live-tested against the
`Endpoints` sample registry (PATCHed `e1`/`e2` test entities with
`envelope: "CloudEvents/1.0"` + `envelopeoptions: {mode, format}` and
`protocol: "AMQP/1.0"` + `protocoloptions: {deployed, durable,
endpoints: [...]}`), calling `getExplicitAttrTypeAtPath()` directly via
CDP against the live model:
- With matching data: `envelopeoptions` → `object`, `envelopeoptions.
  mode` → `string`, `protocoloptions` → `object`, `protocoloptions.
  deployed` → `boolean`, `protocoloptions.endpoints` → `array` — all
  correctly resolved only because of the ifvalues match.
- Without matching data (`{}`/`null`): all of the above correctly
  return `null` — confirming these attributes are genuinely conditional
  and invisible when their trigger isn't set, matching server behavior.
- Test entities/data deleted afterward; xrserver restarted with
  `--recreatedb --samples` to restore clean sample data; chromium test
  processes/profiles/logs cleaned up.

**Not done / follow-up flagged, not yet actioned**: `isSpecAttr()`/
categorization logic (spec vs. extension attribute grouping in Property
tables) is not ifvalues-aware — an ifvalues-triggered attribute like
`envelopeoptions` currently still renders in the "extension attributes"
bucket rather than being recognized as a legitimately-declared attribute
for that entity's current state. Flagging for a future discussion/
decision rather than silently expanding scope here, since the original
`TODO(ifvalues)` was specifically about type/monospace resolution.

**Status**: Complete (core `ifvalues`-aware type resolution). Category-
grouping ifvalues-awareness is a separate, unstarted follow-up.

## Show real server URL (not proxy URL) in display-only text

**Problem**: `xrproxy.go` rewrites the remote server's real origin to the
local `/xrproxy/<base64>` prefix directly in the raw HTTP response body
(and `Location`/`Content-Location`/`Link` headers) before the browser ever
sees it — so `self`/`shortself`/`xid`-adjacent `*url` fields embedded in
JSON responses are already proxy-encoded by the time `app.js` renders
them. This made every displayed URL for a proxied server show the ugly
local proxy form instead of the real remote address, even though only
the *display text* needs to be real — actual `fetch()`/navigation must
keep using the reachable proxy URL.

**Fix**: added `toRealURL(url)` helper (`app.js`, right after
`serverFetchBase()`) — the inverse of `serverFetchBase()`: computes the
current server's proxy prefix and, if `url` starts with it, replaces
that prefix with the real `_state.serverURL`; otherwise no-op (safe when
not proxied too, since `serverFetchBase()` just returns the normalized
real URL in that case, making the prefix-check a harmless identity
substitution).

Applied at 3 display-only sites, always leaving `href`/`onclick`/`fetch`
targets untouched (still built from the raw/proxy value) so navigation
keeps working:
1. `copyBtn('Self', ...)` / `copyBtn('ShortSelf', ...)` in the entity
   detail "Tech row" — tooltip + copy-to-clipboard payload only.
2. `renderUrlLinkValue()` — the shared List-view Property-table link
   renderer used for every same-server URL-shaped value across all
   entity types (`self`, `metaurl`, `versionsurl`, `resourcesurl`,
   `defaultversionurl`, any URI/URL-typed spec or extension attribute,
   …) — added a `displayText` variable, set to `toRealURL(rawText)` only
   on the same-server (in-app-navigable) branch; the external-URL branch
   is untouched (external URLs are never proxy-encoded).
3. `syntaxHighlight()`'s JSON-view same-server linkify branch — added a
   `displayM` variable holding a re-escaped, re-quoted `toRealURL(inner)`
   string whenever it differs from the raw matched token `m`; `href`/
   `onclick` (built from `inner` via `buildURL()`/`navigateJsonUrl()`)
   are unchanged.
4. Breadcrumb bar's copy-URL button (`copyLinkBtnHTML()`'s tooltip
   preview, `refreshCopyLinkBtnTooltip()`, and `copyCurrentAPIURL()`'s
   actual clipboard payload) — all three wrap `buildTabAwareAPIURL()` in
   `toRealURL()` before displaying/copying, since this button's whole
   purpose (per its own doc comment) is giving the user a plain,
   curl-able URL for what's on screen; a real remote URL is strictly
   more useful there than the local proxy form, which only works while
   this SPA/proxy instance is running.

**Explicitly out of scope**: `d.defaultversionurl`'s "URL ↗" button
(`target="_blank"` external navigation) — a hybrid display/navigation
case, deliberately left untouched to avoid unintended behavior changes.

**Verified** via a self-proxied test (flagged the app's own
`http://localhost:8080` as "use proxy" via
`localStorage['xreg-proxy-servers']`, so requests round-trip through
`/xrproxy/<base64-of-itself>/...` back to the same server — a
network-independent way to exercise proxy code paths):
- List view: Registry root's Property table now shows `self` as
  `http://localhost:8080/` (real URL) instead of the proxy-encoded form,
  while the link's `href` still points at the in-app SPA route that
  fetches through the proxy correctly.
- JSON view: same page's `self` field shows the real URL as its
  clickable display text; clicking still navigates correctly via the
  proxy.
- Breadcrumb copy-URL button: tooltip preview and the actual value
  passed to `egCopy()` both confirmed (via direct `Runtime.evaluate` of
  `toRealURL(buildTabAwareAPIURL())`) to be the real `http://localhost:8080`
  URL rather than the proxy-encoded form.
- `node --check app.js` passes throughout.
- Chromium test processes and temp profile/log files cleaned up after
  verification.

**Status**: Complete.

## Known-registries auto-discovery + hide/scan controls + `.xregistry`
## capability gating

**Goal**: let the SPA automatically discover sibling registries served by
hosts the user already knows about, without blind probing of arbitrary
hosts, while giving the user full control (hide from Home without
deleting from Config; toggle which servers are used as discovery
sources; see where each auto-added entry came from).

**Server-side**: added a new `GET <registryBaseURL>/.xregistry` discovery
endpoint (`registry/httpStuff.go`'s `HTTPGETXRegistryDiscovery()`),
returning `{"registries": [url, ...]}` for every registry the host
serves (wraps the existing `GetRegistryNames()` DB query). Deliberately
only a plain array of URLs, not a name -> URL map — the name is
server-owned info the client can always get by fetching the URL itself,
so baking it into this doc would create a second, staleness-prone copy.
Deliberately
NOT named `.well-known` (that has strict RFC 8615 host-root-only
semantics that would be misleading here) — `.xregistry` was chosen as a
short, dot-prefixed, collision-avoiding name that can carry more
cross-registry metadata later. Gated behind the capability system like
`/model`/`/modelsource`/`/capabilities`: added `".xregistry"` to
`SupportedAvailable` (`common/capabilities.go`, `Mutable: false`) and to
`GetOffered()`'s attribute map; `HTTPGet()` checks
`info.IsAvailable(".xregistry")` before serving it, and
`GetAllowedMethods()` reports `GET` accordingly for `OPTIONS`. Confirmed
`.xregistry` appears ONLY nested inside `capabilities.available` (never
as a top-level Registry entity attribute) both via direct server testing
and by auditing all test fixtures. Because JSON map keys are sorted
lexicographically and `.` sorts before letters, `.xregistry` serializes
as the *first* key in `available`, not last.

**SPA storage model** (`registry/ui/app.js`): three independent
per-server-URL localStorage maps, keyed by normalized URL:
- `LS_DISCOVERED` (`xreg-discovered-from`) — the URL of the server whose
  `.xregistry` response caused this entry to be added, or unset for a
  manually-added entry. Set once at `addServer()` time and never
  mutated afterward (first-seen wins if multiple sources report the same
  URL). Purely informational/provenance.
- `LS_SCAN` (`xreg-scan-enabled`) — whether this entry itself is used as
  a discovery source (i.e. whether the SPA calls *its* `.xregistry`
  endpoint during a scan pass). Defaults `true` for manually-added
  entries and for the app's own origin server; defaults `false` for
  freshly auto-discovered entries, so discovery is non-transitive by
  default. Toggleable on any entry via the Config page without touching
  `discoveredFrom` (preserves provenance even after promoting an
  auto-discovered entry into a scan source).
- `LS_HIDDEN` (`xreg-hidden-servers`) — excludes an entry from the Home
  page listing without deleting it from Config; survives repeated
  discovery/scan merges (a rediscovery of an already-hidden URL does not
  silently un-hide it).

**Scan logic** (`scanForRegistries()`): on Home page load, a
non-blocking background pass calls `.xregistry` for every
`scanEnabled=true` entry (including the app's own origin) and merges any
newly-seen `{name, url}` pairs into `LS_SERVERS` (new entries get
`discoveredFrom` = the scanning server's URL, `scanEnabled=false`,
`hidden=false` unless already present/hidden). Self-URL matches are
skipped. An explicit "Refresh known registries" button on the Config
page runs the same pass on demand. `cfgSave()`'s URL-rename flow carries
over `scanEnabled`/`hidden`/`discoveredFrom` to the renamed URL (mirrors
the pre-existing `proxy`-flag carry-over).

**Config page UI**: added Scan/Hide checkbox columns per server row plus
a "Discovered via ..." badge (`.cfg-discovered-badge`, tooltip shows the
source URL, visible text is just "auto") when `discoveredFrom` is set.
The "Refresh known registries" button lives in a `.config-section-header`
flex row (space-between) alongside the "Registry Servers" `<h3>` title,
rather than in the Add-row. The Add-row itself has its own "Scan for
more" checkbox (checked by default) — when adding a new server with it
checked, `cfgAddNew()` immediately runs `scanForRegistries()` for that
new entry instead of waiting for the next background/manual scan.

**Verified** via a live 3-server test (real `xrserver` instances on
ports 8080/8081/8082, backed by mysql, driven through headless Chromium
+ CDP):
- Manually adding a server defaults it to `scanEnabled=true`.
- Scanning a lone server with no siblings finds nothing new.
- Creating a new registry on a second server and re-scanning discovers
  it with the correct `discoveredFrom` = the source server's URL and
  `scanEnabled=false` (non-transitive: the newly discovered registry is
  not itself used as a further scan source until explicitly promoted).
- Hiding an entry removes it from the Home page render while keeping it
  in storage; hidden state survives a subsequent re-scan (does not get
  silently un-hidden).
- The Config page's header row correctly places the title and "Refresh
  known registries" button together (`display:flex;
  justify-content:space-between`).
- Using the Add-row's "Scan for more" checkbox (checked by default) to
  add a third server immediately discovered that server's sibling
  registry without a separate manual refresh click.
- "Discovered via ..." badges render correctly on all auto-added rows
  with accurate source-URL tooltips.
- `.xregistry` capability gating: confirmed via curl that it's
  correctly nested only inside `capabilities.available`.
- Full `make qtest` (all packages) passes.
- Test databases (`registry1`/`registry2`/`registry3`), chromium
  process/profile, and other temp files cleaned up after verification.

**Status**: Complete.

## Fixed: several Full Data Edit Mode bugs (ifvalues-on-dropdown-pick,
## duplicate Resource action bars on non-Default version, false-dirty
## xid/self on version switch, modelsource nav position reset, nested-
## field dirty highlighting)

Six distinct edit-mode bugs reported and fixed in one session (all
verified live against a running `xrserver` + headless Chromium/CDP):

1. **ifvalues not reconciled when a same-named sibling's *shape* changes
   across `ifvalues` branches** (e.g. CloudEvents' `protocol`/
   `protocoloptions` — both KAFKA and NATS branches define a
   `protocoloptions` sibling, but with different sub-schemas).
   `reconcileIfValuesOnChange()` previously only diffed attribute
   *names* before/after a value change, so a same-name/different-
   definition case was invisible to it. Fixed by also comparing
   attribute-definition object identity (`beforeAttrs[n] !==
   afterAttrs[n]`) — cheap and reliable because
   `resolveIfValuesAttrs()` always starts from the same base-attrs
   object reference for unrelated/always-present attributes.
2. **Duplicate Save/Undo/Delete action bars on the Resource page** when
   a non-Default version was selected — the page-level action bar (bound
   to the Default version) was never hidden, so it showed alongside the
   version-selector's own embedded action bar. Fixed by giving the
   page-level bar `id="dataEditorActionBar"` and toggling its visibility
   in `onVersionSelectChangeReal()` based on the selected version.
3. **False "changed" (dirty) highlighting on `xid`/`self` etc. when
   viewing a non-Default version** — `buildPropsRowsHtml()`'s dirty-row
   comparison was hardcoded against `_dataEditSrc` (the Default
   version's snapshot) regardless of which entity was actually being
   rendered. Fixed to pick `_verEditSrc` when
   `handlerFn === 'verEditFieldChange'`.
4. **Modelsource/Model editor remembered drill-down position across
   section re-entry** — the cache-key check in `renderModelEditor()`
   only reset nav state on a genuinely new server/section combo, not on
   a round trip through some other section and back to the same one.
   Fixed in `pushStateReal()`: explicitly reset `_navTab`/`_navPath`/
   `_navSelected`/`_attrNestStack` when entering `model`/`modelsource` via
   a real server/section change (but not a mere view toggle like
   List↔JSON).
5. **Nested-field edits (e.g. `deprecated.alternative`) never gave a
   live dirty-row visual highlight** — only top-level scalar edits got
   the `xr-row-dirty` class via `toggleRowDirty()`. Added an
   `activeEditSrc()` helper (parallel to `activeEditRoot()`) and wired
   `dataEditNestedFieldChange()` to also call `toggleRowDirty()` on the
   containing row. (Save/Undo button enabling itself was already working
   correctly in every tested scenario for this report.)
6. Confirmed the user's manual `ancestor` → `ancestorid` model/server
   rename needed **no UI code changes** — `app.js`/`specattrs.js` already
   used `ancestorid` consistently everywhere (link detection, display,
   priority/suppression key lists, spec-attrs metadata); verified
   end-to-end live against the real server/model/UI afterward.

**Status**: Complete.

## Changed: `.xregistry` discovery doc's `registries` from a
## name→URL map to a plain array of URLs

**Why**: a `name→URL` map bakes a second, easily-stale copy of each
registry's name into the discovery document. The name is
server-owned info (the target registry's own `registryid`/`name`
attribute) — a client that wants it should fetch the registry's own URL,
not trust a possibly-outdated copy embedded in the discovery doc.

**Changes**:
- `registry/httpStuff.go` (`HTTPGETXRegistryDiscovery()`): now returns
  `{"registries": [url, ...]}` instead of `{"registries": {name: url}}`.
- `registry/registry.go` (`GetRegistryNames()`): added `ORDER BY UID` so
  the array has a deterministic order.
- `tests/http3_test.go` (`TestHTTPXRegistryDiscovery`): updated expected
  JSON shape to the array form.
- `registry/ui/app.js`: `scanForRegistries()` now iterates
  `d.registries` as an array of URLs (no more `Object.keys(regs)`); the
  `.xregistry` JSON viewer's `renderXRegRegistriesTable()` now renders a
  single "URL" column (no "Name" column — the server no longer sends
  one).

**Verified**: full `make qtest` passes; live-checked against a running
server that `/model` and `/<resource>$details` already reflect the
user's separate `ancestor` → `ancestorid` rename has no bearing on this
(unrelated), and that `/.xregistry` returns the new array shape.

**Status**: Complete.

## Added: sortable column headers on the Config page's Registry
## Servers table

Clicking a column header (Name, Location, Proxy, Scan, Hide) sorts the
table by that column; clicking the same header again reverses direction.
A `▲`/`▼` arrow (`.cfg-sort-arrow`) shows the active column/direction.
Sort state (`_cfgSortCol`/`_cfgSortDir`) is in-memory only (resets to
Name/ascending on reload) — see `cfgSortBy()`/`cfgSortedServerUrls()`/
`cfgSortHeaderHTML()` in `app.js`. The local ("this server") row always
stays pinned first regardless of sort column/direction, since it isn't a
regular addable/removable entry.

**Status**: Complete.

## Fixed: a server's last-known display name could revert to its
## bare hostname after a page reload if the server was temporarily
## unreachable

**Bug**: `_labelCache` (normalizedURL → probed registry name) was
purely in-memory. A full page reload while a previously-known server
happened to be offline/unreachable meant the cache came back empty, so
`serverLabel()` fell back to displaying the bare hostname instead of the
name that had been successfully learned earlier — even though nothing
about the server's actual name had changed.

**Fix**: `_labelCache` is now persisted to `localStorage` under a new
key, `LS_LABELS` (`xreg-label-cache`), loaded at startup and saved via
`saveLabelCache()` every time `probeRegistry()`'s underlying fetch chain
successfully resolves a non-empty `registryid`. The failure path
(network error, non-2xx, malformed response) was already careful to
never write an empty label into the cache (`cb({label: '', ...})`, and
callers already guard with `if (info.label && ...)` before touching the
DOM) — so no code needed to change there; only the *storage* needed to
survive a reload. `LS_LABELS` was also added to `cfgResetAll()`'s
list of localStorage keys to clear.

**Verified live** with an isolated second `xrserver` instance (separate
`--db` name and port, to avoid touching the real database) added as a
known server: after a successful probe populated and persisted its
label, CDP's `Network.setBlockedURLs` was used to simulate the server
going fully offline, then the page was hard-navigated (simulating F5).
Both the top-level server card and its auto-discovered sibling
registries continued to display their correct last-known names (not
bare hostnames), confirming the fix.

**Caution for future sessions**: do NOT start a second local `xrserver`
instance for testing without an explicit `--db <different-name>` (and
ideally a non-default `-p <port>`) — the default DB name (`registry`)
and port (8080) match whatever instance the user has running, and
`--recreatedb` will destroy their live data. This happened once this
session (an unintended `--recreatedb` against the shared DB); confirmed
recoverable only because the wiped data was just stock `--samples` data,
but it must not be repeated.

**Status**: Complete.

## Added: raw JSON edit ("postman-style") for JSON view

**Feature**: JSON view (`renderJSONView()` in `registry/ui/app.js`) was
always read-only. Hitting "Edit" while in JSON view now swaps the
read-only twisty tree for a fully-editable `<textarea>` containing the
current entity/collection/document's pretty-printed JSON. The user can
freely retype/restructure it (including adding inlined nested xreg
entities) and hit Save (PUT or PATCH, sent verbatim — no auto-diffing,
unlike List view's PATCH) — a lightweight, built-in "Postman" for the
current page. Applies to data entities, data collections, the
Capabilities document, and modelsource; Delete is entity-only (mirrors
List view's existing Delete scope). `model`, `capabilitiesoffered`, and
`xregistry` stay pure read-only, unchanged.

**Changes**:
- `registry/ui/app.js`: added `computeEnableEdit()` (shared helper
  factored out of `renderHeader()`'s inline `enableEdit` logic, now used
  by both `renderHeader()` and the new JSON-edit-mode check so they
  can't drift out of sync); `renderJSONView(data)` now branches to a new
  `renderJSONEditView(data)` when `_state.editMode &&
  computeEnableEdit()`; new state (`_jsonEditDirty`,
  `_jsonEditOrigText`, `_jsonEditDraftText`, `_jsonEditKind`,
  `_jsonEditURL`) and functions `resetJsonEditBuffer()`,
  `jsonEditTarget()`, `renderJSONEditView()`, `jsonEditInputChanged()`
  (live JSON.parse validation, dirty tracking, button enable/disable),
  `jsonEditUndo()`, `jsonEditSave(verb, cb)` (PUT/PATCH, updates the
  section's own source-of-truth cache on success), `jsonEditDelete()`
  (entity-only, delegates to existing `deleteDataEntity()`). Added
  matching `_jsonEditDirty` guards (with "clean leave" buffer reset) to
  `pushState()` and `setDataView()`, following the existing per-buffer
  dirty-guard pattern (`_modelDirty`/`_capDirty`/`_dataDirty`/etc.). The
  buffer is an independent edit buffer that always starts fresh from the
  latest server data each time Edit is turned on in JSON view (does not
  persist across toggle-off/toggle-on, unlike List view's buffers).
- `registry/ui/style.css`: added `.json-edit-textarea` (modeled on the
  existing `.eg-doc-textarea` style).

**Verified live** via headless Chromium/CDP against an isolated test
`xrserver` instance (own `--db`/`-p`, dropped afterward): entity Save
(PUT) and Save (PATCH) with a genuinely partial body both round-tripped
correctly (PATCH confirmed non-auto-diffed — only included keys were
touched server-side); invalid JSON correctly disabled both Save buttons
and showed an inline error; Undo correctly reverted the textarea and
cleared dirty state; the entity Delete button correctly deleted and
navigated to the parent (collection); collection-page JSON edit
correctly rejected PUT (405, server-side collections don't support
full-replace) but accepted PATCH, with no Delete button shown;
Capabilities-document and modelsource JSON edit both saved correctly
via raw JSON, neither showing a Delete button; the new
`_jsonEditDirty` guard correctly triggered the Save/Discard/Cancel
dialog when switching away from an unsaved JSON edit (via List view
switch), and the buffer correctly reset to fresh/empty after Discard;
the pre-existing reverse guard (unsaved List-view edit → switch to
JSON view) was confirmed unaffected and still fires correctly. `make
qtest` is a no-op for this change (UI-only, no Go files touched).

**Status**: Complete.

## Fixed: CORS preflight missing `Access-Control-Allow-Headers`

**Bug**: cross-origin PUT/PATCH/POST/DELETE requests from the SPA (e.g.
editing modelsource on a registry served from a different origin) were
silently failing with a generic CORS error, even though the *actual*
request would have succeeded. Root cause: `DefaultWriter.Write()`
(`registry/httpStuff.go`) set `Access-Control-Allow-Origin` and
`Access-Control-Allow-Methods` on every response, but never
`Access-Control-Allow-Headers`. Since the SPA always sends
`Content-Type: application/json` on mutating requests, the browser
issues a preflight `OPTIONS` request first, and without
`Access-Control-Allow-Headers` echoing back `Content-Type` (or whatever
headers were requested), the browser blocks the real request before it
is ever sent — indistinguishable client-side from a network failure.
This was invisible for same-origin (local or already-proxied) requests,
which is why it went unnoticed until testing against a genuinely
different-origin remote server.

**Fix**: `DefaultWriter.Write()` now reflects back the incoming
preflight's `Access-Control-Request-Headers` value as
`Access-Control-Allow-Headers` (falling back to `Content-Type` if that
request header is absent).

**Verified** via `curl` (both an isolated test server and the full
`make xrserver` rebuild): a preflight `OPTIONS` request now returns
`Access-Control-Allow-Headers`, and GET/PUT continue to work normally.
`make qtest` showed one `TestXRBasic` failure, confirmed pre-existing/
environmental (caused by a live server occupying the default port 8080
during the test run), not caused by this change — `git diff --stat`
showed only `httpStuff.go` touched, plus an unrelated pre-existing
`common/shared_model` diff not made this session. Requires a server
rebuild + restart to take effect (Go-side change, no hot reload).

**Status**: Complete.

## Fixed: modelsource edit-mode nav disabled on empty registry

**Bug**: on an empty registry (0 groups, 0 attributes, etc.), the
Attributes/Group Types nav rows (and their nested Attributes/Resources/
Version Attrs/Resource Attrs/Meta Attrs equivalents) were disabled
whenever their count was 0 — even while in **edit** mode — making it
impossible to drill in and add the very first attribute or group type.
The disable-when-empty behavior is only meant to apply in read-only
(view) mode, where there's nothing to show.

**Fix**: all 7 `navItem(..., disabled)` call sites in the modelsource
nav-building code (`registry/ui/app.js`) changed the disabled condition
from `xCount === 0` to `_modelReadOnly && xCount === 0`.

**Verified** via code inspection (all 7 call sites confirmed via
`grep`); not live-verified in-browser due to a transient headless
Chromium launch failure at the time (see the Chromium/CDP note below —
since resolved).

**Status**: Complete.

## Fixed: stale nested `item`/`attributes` JSON after attribute type switch

**Bug**: in the modelsource editor, switching an attribute's type from
a complex type (`map`/`array`, which carries an `item` sub-object, or
`object`, which carries an `attributes` sub-object) back to a scalar
type (e.g. `string`) left the stale nested `item`/`attributes` data in
the saved JSON, since `saveAttrFrom()` unconditionally copied both
fields over from the existing attribute regardless of the new type.

**Fix**: `saveAttrFrom()` (`registry/ui/app.js`) now only preserves
`attr.attributes` from the existing attribute when the new type is
`object`, and only preserves `attr.item` when the new type is `map` or
`array`.

**Verified** live via headless Chromium/CDP against an isolated test
server: confirmed `map` → `string` correctly strips `item`, and
`object` → `string` correctly strips `attributes`.

**Status**: Complete.

## Fixed: xrproxy corrupting sibling-registry identities in `.xregistry`

**Bug**: many sibling registries served by the same remote host as a
flagged "use proxy" server were being silently, permanently routed
through the local `/xrproxy/...` path even though only the one origin
was ever flagged proxied on the Config page — with no visible
indication why. Observed as a large volume of `/xrproxy/<b64>/...`
requests in the server log for registries that appeared un-proxied in
Config.

**Root cause**: `HTTPXRProxy()` (`registry/xrproxy.go`) does a blind
`bytes.ReplaceAll` of the remote origin string across every proxied
response body, to rewrite embedded `self`/`xid`/`*url` links so they
route back through the local proxy when browsing. This is correct for
entity/collection JSON, but is semantically wrong for the `.xregistry`
discovery document's `registries` array, whose entries are meant to be
the REAL addresses of other, independent sibling registries — not
self-referential navigation links. `scanForRegistries()`
(`registry/ui/app.js`) then naively stored these already-rewritten
proxy URLs as each sibling's permanent identity in `LS_SERVERS`,
hard-wiring them through one specific local proxy path forever (with
`isProxied()` on that literal URL always returning false, since the
flag map is keyed by the corrupted URL itself, not the real origin).

**Fix**: added `decodeAnyXRProxyURL(url)` in `registry/ui/app.js`,
which reverses the local `/xrproxy/<b64url-origin>/...` rewrite back to
the real remote URL (base64url-decoding the origin segment and
reattaching the suffix path). `scanForRegistries()`'s `regs.forEach()`
loop now calls this on every discovered URL before `normalizeURL()` and
the existing "skip local origin" check, restoring the invariant that
identity URLs stored client-side are always the real remote address —
proxy translation happens only at actual fetch time, via
`serverFetchBase()`.

**Verified** live via headless Chromium/CDP against an isolated test
server: added a real remote proxied server, ran `scanForRegistries()`
directly, and confirmed `loadServers()` now contains the siblings'
real remote URLs (not local proxy paths), while `proxyFlags` correctly
contains only the one explicitly-flagged origin — siblings are NOT
auto-flagged proxied, matching the existing per-server opt-in design
(this was a deliberate open question, decided as "no" — see below).

**Open design question, decided**: should auto-discovered siblings of
a proxied server automatically inherit the proxy flag, since in
practice they're almost always served by the identical physical host?
Decided **no** for now — siblings default to non-proxied and the user
must flag each one individually via Config if the host truly requires
proxying for all of them.

**Status**: Complete.

## Fixed: deleting a server left orphaned per-URL state behind

**Bug**: `removeServer()` (`registry/ui/app.js`) only removed the URL
from `LS_SERVERS`, but never cleared the other per-URL localStorage
maps keyed to it (`LS_NAMES`, `LS_PROXY`, `LS_SCAN`, `LS_HIDDEN`,
`LS_DISCOVERED`). If the same URL was later re-added or
re-auto-discovered (e.g. via `scanForRegistries()`), it silently
inherited whatever flags (hidden, proxied, scan-enabled, name override,
discoveredFrom) it had before being deleted — observed as newly
auto-discovered registries mysteriously showing up already hidden.

**Fix**: `removeServer()` now clears all of `LS_NAMES`, `LS_PROXY`,
`LS_SCAN`, `LS_HIDDEN`, and `LS_DISCOVERED` for the removed URL, so a
delete is a full teardown and a later re-add/re-discovery starts from
clean defaults. `cfgSave()`'s URL-rename flow (which calls
`removeServer(oldUrl)` internally as part of the rename) was updated to
capture the old URL's flags into local variables *before* calling
`removeServer()`, since it now wipes them, then apply the captured
values to the new URL afterward — preserving the existing carry-over
behavior.

**Verified** via a standalone Node.js test (stubbed `localStorage`,
functions extracted directly from `app.js`): confirmed hide → delete →
re-add now correctly starts unhidden (previously stayed hidden);
confirmed the rename flow still correctly carries over
proxied/scanEnabled/hidden/discoveredFrom to the new URL and clears
them from the old one.

**Status**: Complete.

## Config page: removed auto-scan, added explicit "Scan for registries"
## review workflow, blocked duplicate-URL adds

**Problem**: the Config page previously auto-scanned `.xregistry` on
every registry visited (via `renderHome()`) and silently auto-added any
newly discovered registries, causing the "known servers" list to grow
unexpectedly large without user awareness or consent. There was also a
per-server "Scan for more" flag/checkbox controlling this background
behavior. Separately, attempting to add the same server URL twice (e.g.
once direct, once via proxy) silently no-op'd with no explanation, since
duplicate handling was based on the URL alone.

**Decision** (discussed at length with the user): remove automatic
background scanning entirely. Add an explicit, user-initiated "Scan for
registries" bulk action on the Config page: select one or more known
server rows (any row can act as a scan source, including the local/
origin row, which is now selectable), click "Scan for registries", and
review the results in a modal before deciding what to add — discovered
URLs are grouped into "Not yet added" (checked by default) vs. "Already
added" (informational only). Clicking "Add Selected" adds only the
checked entries.

True per-entry duplicate-URL support (e.g. one entry proxied, one direct,
both pointing at the same URL) was considered and rejected: proxy state,
the model/label/capabilities caches, and `_state.serverURL`/the
`?server=` query param are all resolved by URL string alone everywhere
in the app — supporting genuine duplicates would require threading a new
per-entry identity through every fetch/cache call site, which the user
judged not worth the complexity. Instead, adding (or renaming to) a URL
that's already configured is now explicitly **blocked**, with an inline
error message telling the user to delete the existing entry first.

**Changes** (`registry/ui/app.js`):
- Removed: `LS_SCAN`, `isAutoScanDisabled()`, `loadScanFlags()`,
  `saveScanFlags()`, `isScanEnabled()`, `setScanEnabled()`,
  `cfgSetAutoScan()`, `cfgScanNow()`, the automatic background scan call
  in `renderHome()`, the Options page's "Auto-scan" toggle row, the
  Config page's "Scan" column, the Add-row's "Scan for more" checkbox,
  and the old "Refresh known registries" button.
- `addServer(url, discoveredFrom)` now returns `true`/`false` (added vs.
  blocked-duplicate) instead of silently no-op'ing; all callers
  (`cfgAddNew()`, the new `cfgConfirmScanResults()`) check this.
- `removeServer(url)` no longer touches the (now-removed) scan-flag map.
- Replaced `scanForRegistries()` with `discoverRegistriesFrom(sourceUrls,
  cb)` — a pure, side-effect-free fetch/categorize helper returning
  `{url, discoveredFrom, alreadyKnown}[]`; it never calls `addServer()`
  itself, so discovery always requires explicit user confirmation via
  the new review modal.
- Added: `cfgScanSelected()` (runs `discoverRegistriesFrom()` on checked
  rows), `cfgShowScanResults(results)` (renders the grouped review
  modal), `cfgConfirmScanResults()` (adds checked new entries, closes
  the modal, re-renders Config).
- The local/origin server row on the Config page is now selectable (so
  it can be used as a scan source or bulk-hidden), but `cfgDeleteSelected()`
  still excludes it from deletion.
- `cfgAddNew()` shows an inline error (`#cfg-new-error`) instead of
  silently doing nothing when the URL is already configured.
- `cfgSave()`'s rename flow now blocks renaming to a URL that's already
  configured elsewhere (`alert(...)`), consistent with the new
  duplicate-blocking policy; removed its now-dead scan-flag references.

**Changes** (`registry/ui/style.css`): removed `.cfg-scan-cell`; added
`.cfg-inline-error`, `.cfg-modal-overlay`, `.cfg-modal`,
`.cfg-modal-title`, `.cfg-scan-group-title`, `.cfg-scan-list`,
`.cfg-scan-item`, `.cfg-scan-item-known`, `.cfg-scan-empty`,
`.cfg-modal-btns`.

**Verified** via headless-Chromium CDP against an isolated test
`xrserver` (own DB/port) serving 3 sibling registries plus the origin
server:
- No auto-growth of the server list across repeated page reloads
  (confirms the background scan is fully gone).
- Selecting all rows and clicking "Scan for registries" correctly listed
  all 3 sibling registries under "Not yet added" (pre-checked); "Add
  Selected" added exactly those 3, bringing the total to 4 rows.
- Re-adding an already-configured URL via the Add row was blocked with
  the correct inline error message and no change to the stored list.
- Re-running the scan afterward correctly reclassified all 3 as "Already
  added" (no "Add Selected" button shown, since nothing was new).
- Deleting a server still fully clears its hidden/proxy state (existing
  fix from earlier this session, re-verified against the new code path).
- `node --check app.js` passes; `make qtest` shows no new failures (only
  the known pre-existing `TestXRBasic` port-8080 flake, unrelated).
- Test DB/server/Chromium instance and temp files all cleaned up
  afterward.

**Status**: Complete.

## Note: headless Chromium/CDP launch — use the snap wrapper, not the raw binary

Launching Chromium via its raw snap binary path
(`/snap/chromium/<rev>/usr/lib/chromium-browser/chrome ...`) can
intermittently fail with `error while loading shared libraries:
libnspr4.so: cannot open shared object file` in this environment. Use
the `chromium` snap wrapper command instead (`/snap/bin/chromium ...`,
or just `chromium ...` if on `PATH`) — `snap run` sets up the correct
library/environment context, and this has proven reliable across
multiple sessions.

## Config page: "Select All" spacing fix

**Changes** (`registry/ui/style.css`): `.cfg-scan-select-all` padding
adjusted (`padding-bottom: 4px` → `padding-bottom: 8px; margin-bottom:
6px;`) so the "Select all" checkbox row in the Scan Results modal isn't
crowded against the first result row below it.

**Verified** via CDP: gap between `.cfg-scan-select-all`'s bottom edge
and the first `.cfg-scan-item`'s top edge measured at 6px (matches the
new `margin-bottom`).

**Status**: Complete.

## JSON edit view UX overhaul

Problem: the raw JSON edit view (Server Endpoints / entity JSON "Edit"
mode) validated JSON on every keystroke, which meant it constantly
flashed an "Invalid JSON" error the instant someone started typing
(e.g. right after an opening `{`) — bad UX. Separately, when a real
server error or the invalid-JSON banner appeared, the action buttons
(which lived below the textarea) could scroll off the bottom of the
screen along with it.

**Changes** (`registry/ui/app.js`):
- `renderJSONEditView()`: moved the action bar (now includes a new
  **Format** button, plus the existing Save PUT/Save PATCH/Undo/Delete)
  into the same sticky top header row as "Server: ..." (reusing the
  read-only JSON view's `.json-exp-wrap` sticky-positioning pattern),
  instead of rendering it below the textarea. Error banners
  (`#jsonEditError`/`#jsonEditInvalid`) now render directly under that
  header too, so they're always visible without scrolling.
- `jsonEditInputChanged()`: no longer calls `JSON.parse()` on every
  keystroke. Now only compares the draft text against the pristine
  original to track dirtiness; Save/Undo buttons are enabled/disabled
  based purely on dirtiness, not JSON validity.
- Added `jsonEditFormat()`: the only place JSON is now actually
  validated before Save. Parses the textarea content on click; if
  valid, pretty-prints it (`JSON.stringify(parsed, null, 2)`) in place;
  if invalid, shows the error in `#jsonEditInvalid` without touching
  the textarea content.
- `jsonEditSave()`'s existing validate-before-send behavior (via
  `#jsonEditError`) is unchanged.

**Changes** (`registry/ui/style.css`): added `.json-edit-header`
(opaque background + border, since this sticky header now hosts a full
button row rather than the read-only view's single small button) and
`.json-edit-header .editorActionBar { pointer-events: auto; }` (since
the parent `.json-exp-wrap` has `pointer-events: none` by default).

**Verified** via CDP against the live dev server: header renders with
all 5 buttons + "Server: ..."; typing invalid JSON shows no live error
and enables Save (dirty-based); clicking Format on invalid JSON shows
the error banner without altering the textarea; clicking Format on
valid-but-unformatted JSON pretty-prints it in place; Undo reverts and
disables Save/Undo; header confirmed `position: sticky` via computed
style.

**Status**: Complete.

## Resource creation: don't pad an empty `{}` as the document body

Bug: creating a new Resource entity (e.g. `dirs/d1/files/<id>`) via the
collection page's "+ Add" form always PUT the form's metadata object
(`{}` if no extra fields were filled in) to the plain, non-`$details`
resource URL. For any resource type with `hasdocument: true`, the
server treats a plain PUT/POST body as the resource's literal document
content (see `registry/httpStuff.go` `metaInBody` logic) — so this
silently created every new hasdocument resource with its document body
set to the two-byte string `{}`, instead of leaving it empty.

**Fix** (`registry/ui/app.js`, `saveNewEntity()`): when creating an
entity at a Resource collection path (`_state.path.length === 3`) whose
resource type has `hasdocument: true` (checked via
`resourceHasDocument()`), append `$details` to the create URL. This
makes the server parse the PUT body as metadata only, leaving the
actual document unset (empty) until the user explicitly sets real
document content afterward.

**Verified**: intercepted the real `fetch()` call from the Add form
(mocked `window.fetch`) and confirmed the URL now ends in `$details`
for a `hasdocument: true` resource type; then did a real create via
curl using that exact `$details` URL/body and confirmed `GET` on the
plain resource URL (non-`$details`) returns an empty body — the
document is no longer padded with `{}`. Test resource deleted
afterward.

**Status**: Complete.

## JSON view Details/Document toggle

Gap found: the old `ui.go` UI had a `detailsSwitch` button letting users
toggle any Resource/Version entity page between `$details` (metadata)
and the plain URL (raw document) when `hasdocument=true`. The new SPA's
List view already has full parity for this via its Document/Meta/
Version Details tab bar, but **JSON view** had no equivalent — it
always force-fetches `$details` for Resource/Version paths
(`needsDetails()`), so there was no way to see a resource's actual
document content while in JSON view.

**Design decisions** (confirmed with @duglin before implementing):
- Toggle defaults to "Details" (today's behavior), not "Document".
- Read-only for this first pass — no Edit-mode support for the
  Document side yet.
- Toggle is always shown on Resource/Version pages, even when the
  resource type has no document defined (`hasdocument: false`) —
  Document mode then just shows a "No document defined" message
  instead of hiding the toggle entirely, so the control behaves the
  same way regardless of resource type (per @duglin: "It should allow
  you to alternate between doc/details even if there's no doc - it'll
  just be blank in that case").
- Visual style: reuses List view's own `.eg-doc-tab`/`.eg-doc-tabs`
  pill styling verbatim (blue, same as the Document/Details tabs on
  List view's Resource/Version pages), not a separate green
  `.editorBtn`-styled or bespoke-color pill — per @duglin: "the style
  of things in json view are green-ish [while list view has] a blue
  tint... let's move json view to match list view."

**Changes** (`registry/ui/app.js`):
- Added `_jsonDocMode` ('' = details, 'doc' = raw document) and
  `_jsonDocModeKey` (resets the toggle back to 'details' whenever
  navigation moves to a different server/path, so switching entities
  always starts back on Details).
- `jsonDocToggleApplies()`: true for any Resource (depth 4) or Version
  (depth >= 6, `path[4] === 'versions'`) entity page — delegates to
  `needsDetails(_state.path)`'s existing depth check, with no
  `hasdocument` gate (the toggle is shown regardless; Document mode
  itself handles the no-document case).
- `buildJsonDocToggleHtml(active)` / `jsonSetDocMode(mode)`: render the
  Details/Document pill (reusing `.eg-doc-tab`/`.eg-doc-tabs`) and
  handle clicks (just re-renders from `_lastData` — no re-fetch needed
  when switching to Details, since that data is already the
  `$details` JSON already in hand).
- `buildJsonExpandAllBtnHtml(disabled)`: factored out the "expand all"
  button so both Details and Document mode headers render the exact
  same button (same id/position) whether usable or not — Document mode
  starts it disabled (grayed out, inert) since raw text/binary content
  isn't a collapsible JSON tree, then enables it if the document
  content does turn out to be JSON. Keeping the button always present
  (rather than only in Details mode) avoids the other header buttons
  shifting position when switching modes, per @duglin: "rather than
  deleting the 'all' button when it's not applicable, let's just
  disable it because otherwise the other buttons move and that looks
  funky".
- `renderJSONView()`: shows the toggle (grouped with "expand all" in a
  `.json-exp-btn-group` wrapper so `.json-exp-wrap`'s `space-between`
  still only splits two groups) when `jsonDocToggleApplies()`;
  delegates to the new `renderJSONViewDocumentMode()` when in 'doc'
  mode.
- `renderJSONViewDocumentMode(entityData)`: when the resource type has
  no document defined (`!resourceHasDocument()`), renders a "No
  document defined for this resource type." message directly, skipping
  the fetch entirely (a plain GET on such a resource just returns the
  same `$details` metadata again, which would be confusing to show
  under a "Document" label). Otherwise fetches and renders the actual
  document content — parsed and shown via the same twisty JSON tree
  when the content happens to be valid JSON (which also re-enables the
  "expand all" button), otherwise as a plain read-only textarea
  (reusing `.eg-doc-textarea`), or a "binary, not previewable" message
  — mirroring List view's Document tab preview
  (`loadDocumentPreview()`'s `<key>url`/`<key>base64`/inline-`<key>`/
  `self`-fallback resolution and `isBinaryContent()`/
  `decodeUTF8Bytes()` helpers), just as JSON view's whole page instead
  of one List-view tab's panel. Its header wraps the toggle in a
  `.json-exp-btn-group` (pointer-events: auto) exactly like the
  Details-mode header does — an earlier version placed it unwrapped as
  a direct child of `.json-exp-wrap` (which has `pointer-events: none`
  by default), which silently made the "Details" button unclickable
  once in Document mode (bug: "once I switch to doc, I can't seem to
  go back. Clicking on 'details' does nothing").
- `renderJSONEditView()` (raw JSON edit mode): now calls a new
  `sizeJsonEditTextarea()` after rendering (and on window resize,
  alongside the existing `sizeDocTextarea()` call) to stretch the
  textarea down to the bottom of the viewport — same idiom as
  `sizeDocTextarea()` for the Document tab's preview — instead of
  stopping short at a fixed `min-height`.

**Changes** (`registry/ui/style.css`):
- `.json-exp-btn-group` (pointer-events: auto wrapper for the header's
  right-side button cluster).
- `.json-exp-btn`/`.json-exp-btn-disabled` restyled from a neutral gray
  pill to List view's blue `.eg-link-btn` color scheme (`#2060a0` text,
  `#eef3fa` background, `#b8cce4` border) for consistency; disabled
  state just dims via `opacity: 0.4`.
- `.json-doc-toggle`/`.json-doc-toggle-btn`: no longer a bespoke pill —
  the toggle buttons now carry `.eg-doc-tab`/`.eg-doc-tabs` classes
  directly (List view's own Document/Details tab styling, verbatim
  blue color scheme), with small `.json-doc-toggle-btn`-scoped overrides
  (smaller font/padding/radius) placed right after `.eg-doc-tab`'s
  rules so they win the cascade, fitting the compact single-row JSON
  view header instead of List view's roomier tab-bar layout.
- `.json-edit-header` no longer has its own background/border/padding
  override — it now inherits `.json-exp-wrap`'s transparent, compact,
  single-row styling as-is, so Edit mode's header no longer visibly
  "grows" or looks like a different design than the read-only header
  (per @duglin: "we're inconsistent between edit mode and view mode
  w.r.t. how the buttons are shown... let's be consistent... I prefer
  the way view mode looks - it's more compact"). The nested
  `.editorActionBar`'s own box styling (padding/border-bottom/
  background) is stripped to `none`/`transparent` so its buttons sit
  inline in the same row, using smaller `.editorBtn` sizing
  (`padding: 1px 8px; font-size: 11px`) to match.
- `.json-edit-textarea`'s `min-height` lowered from `420px` to `200px`
  to match the JS-computed floor in `sizeJsonEditTextarea()` — the
  previous 420px floor was silently overriding the dynamic height calc
  on short viewports (CSS `min-height` always wins over a smaller
  inline `height`), defeating "full window height" on small windows.

**Verified** via CDP against the live dev server, using real
`.click()` calls (not direct function calls, which had previously
masked the pointer-events bug above):
- Toggle renders (defaulting to "Details" active) on Resource entity
  pages regardless of `hasdocument`; a temporary `hasdocument: false`
  resource type/entity was added to `/modelsource` for this, then
  removed again afterward.
- Clicking "Document" on the no-document resource shows the "No
  document defined..." message, with "expand all" auto-disabled and
  no fetch performed.
- Real-click "Details" from Document mode correctly switches back
  (confirms the pointer-events fix) and re-enables "expand all".
- Toggle buttons render with `background-color: rgb(32, 96, 160)`
  (`#2060a0`) when active — matching List view's `.eg-doc-tab.active`
  exactly.
- Edit mode's header height (~23px) now matches the read-only header's
  (~22px) instead of growing into a separate boxed toolbar.
- JSON edit textarea now stretches to fill the viewport down to a
  16px bottom margin (confirmed via `getBoundingClientRect()`) instead
  of stopping short at the old fixed `min-height: 420px`.

**Status**: Complete (read-only). Editing the Document side directly
from JSON view is a possible future follow-up.

## Three List-view cleanup fixes: meta→JSON routing, persistent edit
## mode, Add-entity form attribute order

Three unrelated cleanup items requested together.

**1. Meta tab → JSON view now shows the `/meta` object, not the parent
entity.** Previously, switching to JSON view while the List-view
"Metadata" tab was active always rendered the parent Resource/Version's
own JSON (`_lastData`), silently ignoring which tab was active. New
`renderJSONViewForCurrentTab(data)` (`app.js`) checks
`_state.docTab === 'meta'` (only meaningful for non-collection `data`
section pages) and, if so, renders the already-cached `_metaData` (or
fetches it via `data.metaurl` first) instead. Wired into both
`setDataView()`'s data-page JSON branch and `renderEntityFromData()`'s
JSON branches (covers both switching view live and a fresh page
load/refresh while already on `tab=meta&dview=json`).

**2. Edit mode no longer auto-disables on navigation.** Previously,
essentially every navigation helper (`navigateTo()`, breadcrumb clicks,
`pageHref()`'s synthetic links, version/tab selectors, the meta-page
redirect, etc. — ~29 call sites) explicitly forced `editMode: false`
into its `pushState()`/`pushStateReal()` patch, silently turning edit
mode off on every click even when the existing unsaved-changes
guard (`showLeaveEditDialog()`) had nothing to warn about. Removed
`editMode: false` from all of these navigation patches — `_state.editMode`
(a single flag shared across Data/Model Source/Capabilities/JSON-view
edit) now persists across any navigation, including switching sections
entirely, until the user explicitly toggles it off via the Edit button
(`toggleEdit()`) or dismisses the leave-edit dialog with an explicit
`editMode: false` action. The dirty-state guards themselves
(`pushState()`'s `leavingXEdit` checks, `setDataView()`'s equivalents)
were unaffected — they already keyed off `patch.path`/`patch.section`/
`patch.serverURL`/`patch.view` changing, not `patch.editMode`, so they
still correctly prompt Save/Discard/Cancel before any navigation while
mid-edit; only the forced turn-off after the guard resolves was removed.
Confirmed via CDP: edit mode stays `true` after navigating between
entities and across a section change (Data → root, etc.); the leave-edit
dialog still fires on a simulated dirty edit and, after clicking
Discard, edit mode remains on for the new page.

**3. Add-entity form now orders (and filters) attributes like View
mode.** `buildAddEntityFormHtml()` previously appended every
non-object/array/map key from the model's `attributes` map, alphabetically
sorted, on top of a hardcoded `name`/`description`/`documentation`/`icon`
block — with no `readonly` filter and no de-duplication. Since
`modelAttrsMapForPath()` returns the *full* per-type attribute map
(including core spec attributes like `self`/`shortself`/`xid`/`epoch`/
`self`/`filesurl`/`filescount`, not just genuine extensions), this
actually rendered several server-computed, non-settable fields as
editable inputs, PLUS duplicate rows for `name`/`description`/
`documentation`/`icon` (once from the hardcoded block, again from the
generic loop) — confirmed live via CDP before the fix (extra rows:
`Description`, `dirid`, `Documentation`, `epoch`, `filescount`,
`filesurl`, `Icon`, `Name`, `self`, `shortself`, `xid`). Fixed by:
excluding the dedicated ID field's own key, the four already-rendered
core fields, and any attribute the model marks `readonly`; and ordering
the remaining keys with a new shared helper, `orderPropKeysFlat()` (added
alongside `groupPropsByCategory()`), which applies the exact same
priority sort + category-bucket flattening `buildEntityPropsTableHtml()`
(View mode) uses — just returned as a flat array with no header rows,
since the Add form doesn't need category headers, only matching order.
The global priority list itself was extracted into a shared
`PROPS_PRIORITY` constant so both call sites can never drift apart.
Verified via CDP against a live xrserver (`dirs` group, `d1` entity): Add
form now shows exactly `Name, Description, Documentation, Icon, Created,
Modified` (no duplicates, no readonly/server-computed fields), matching
View mode's category ordering (General before Identity/Versioning/
Content before Timestamps).

`node --check app.js` passes throughout. Chromium test process/profile
cleaned up.

**Status**: Complete.

## Save button relabeling ("full"/"delta") and dual-save leave-edit
## dialog

User asked for two related changes to the app-wide "unsaved changes"
leave-edit dialog (`showLeaveEditDialog()`) and Save button labels:

**1. Renamed all Save button labels.** `Save (Full/PUT)` and
`Save (Partial/PATCH)` were felt to be too geeky — PUT/PATCH are
accurate but jargon-y even though more descriptive than "full"/
"partial" alone. Renamed to `Save (full)` and `Save (delta)`
everywhere: the Data entity, Version, and Meta action bars
(`dataSavePutBtn`/`dataSavePatchBtn`, `verSavePutBtn`/`verSavePatchBtn`,
`metaSavePutBtn`/`metaSavePatchBtn`) and the JSON view's raw-edit action
bar (`jsonEditPutBtn`/`jsonEditPatchBtn`). The Config page's plain
"Save" button (PUT-only, no PATCH concept for that entity) was
correctly left untouched.

**2. Leave-edit dialog now offers a PUT-vs-PATCH choice where relevant.**
Previously, `showLeaveEditDialog(onSave, onDiscard, onCancel)` always
saved via whatever single verb the caller hardcoded (always PUT) when
the user chose to save before navigating away from a dirty editor —
so leaving via the dialog silently discarded the option to do a partial
PATCH save, even on editors (Data entity, Meta, Version, JSON raw edit)
whose on-page action bar already offers both. Added an optional 4th
parameter, `onSaveDelta`: when provided, the dialog renders two Save
buttons — "Save (full)" and "Save (delta)" (both same blue styling) —
letting the user pick right there instead of being forced back to the
page to choose; when omitted, the dialog falls back to the previous
single plain "Save" button. Model Source and Capabilities
(`saveModel()`/`saveCapabilities()`) are single-verb, whole-document-
replace editors (both always call `fetch(url, {method: 'PUT', ...})`
with no verb parameter) — their two `showLeaveEditDialog()` call sites
(in both `pushState()` and `setDataView()`) were intentionally left
unchanged (2-arg, single Save button). The 6 call sites for editors
that genuinely support both verbs (Data entity and JSON-edit guards in
both `pushState()` and `setDataView()`, plus Meta/Version guards in
`pushState()`, `onVersionSelectChange()`, and `switchDocTab()`) were all
updated to pass a `saveXEntity('PATCH', ...)` callback as the 4th arg.
Verified via CDP (Puppeteer-launched Chromium — see note below) against
a live xrserver: editing `dirs/d1`'s Name field then clicking the
"dirs" breadcrumb correctly shows the dialog with both "Save (full)"
and "Save (delta)" buttons; clicking "Save (delta)" fires a real
`PATCH` request (confirmed via intercepted network requests) and
"Save (full)" fires a `PUT`; Model Source's dialog still shows only a
single "Save" button, unchanged.

**False alarm, investigated and disproved:** initial CDP testing
appeared to show a bug where, after the dialog's Save succeeded, the
app didn't navigate to the destination and the dialog stayed open.
Root-caused (via temporary `console.log` instrumentation in
`pushStateReal()`, the dialog's `mkBtn()` onclick, and
`saveDataEntity()`'s success path, all removed again afterward) to a
flaw in the *test script*, not the app: with the dialog open, the page
behind it still has its own (non-disabled, since data was dirty when
it last rendered) action-bar "Save (delta)"/"Save (full)" buttons in
the DOM, sharing the exact same button text as the dialog's buttons.
The test's `Array.from(document.querySelectorAll('button')).find(...)`
matched the *first* one in DOM order — the underlying action-bar
button (which calls `saveDataEntity('PATCH')` with no callback, so it
just re-renders the same entity in place) — not the dialog's own
button. Re-tested scoping the click strictly to the dialog overlay
element and confirmed the real flow works correctly end-to-end: click
"Save (delta)" in the dialog → `PATCH` fires → `saveDataEntity()`'s
success callback runs → `pushStateReal(patch)` fires → URL/page
correctly updates to the original destination (e.g. the `dirs`
collection list). No app bug exists; nothing further to fix.

**Note on headless Chromium via Puppeteer instead of the snap
wrapper:** after `/tmp` was wiped externally mid-session, the system's
snap-confined Chromium (`/snap/bin/chromium`) began failing to launch
headless with "Failed to create socket directory" / process-singleton
errors — traced (via `strace`) to a stale per-snap mount namespace
(`/run/snapd/ns/chromium.mnt`) whose private view of `/tmp` no longer
matched the host's post-wipe `/tmp`, so `mkdir()` calls inside the
sandbox returned `ENOENT` for paths that visibly existed on the host.
Restarting `snapd` could likely fix this but risks disrupting other
users/services on a shared machine, so instead installed `puppeteer`
(`npm install puppeteer` in a scratch dir) purely for its bundled,
non-snap Chromium binary (`~/.cache/puppeteer/chrome/.../chrome`),
after installing its missing shared-lib runtime deps (`libnspr4`,
`libnss3`, `libatk-bridge2.0-0`, etc. via `apt-get install`). Launched
that binary directly with `--headless=new --no-sandbox
--remote-debugging-port=<port> --remote-allow-origins=*
--user-data-dir=<fresh dir>`, then used Puppeteer's `connect()` (not
`launch()`) to attach a script to it over CDP — this sidesteps the
snap sandbox entirely. All scratch dirs/processes cleaned up after
testing.

**Status**: Complete. No app bug found — the earlier "navigate-after-
save" concern was a test-script artifact, not a real issue.

## Fixed: JSON view breadcrumb missing "meta" segment

**Problem**: viewing a Resource/Version's Metadata tab and switching to
JSON view showed the correct `/meta` JSON content, but the breadcrumb
still stopped at the Resource/Version id, not showing "meta".

**Fix**: `buildBreadcrumbSegments()` (`registry/ui/app.js`) now appends
an extra plain (non-clickable) "meta" breadcrumb segment when at a
Resource (depth 4) or Version (depth 6+) page with `_state.docTab ===
'meta'`, marking the previous (entity-id) segment as no longer
"current". List view's own tab-click doesn't trigger this (that path
intentionally skips a full breadcrumb re-render for performance, and
the List page's own tab bar already shows which sub-view is selected),
but a direct page load with `?tab=meta`, or switching to JSON view from
the Metadata tab, both correctly show `.../meta` in the breadcrumb.

**Verified** via CDP against a live xrserver: breadcrumb text reads
`/xRegistry/dirs/d1/files/file1/meta` after clicking the Metadata tab
then switching to JSON view; JSON content's `self` field confirms it's
the actual `/meta` document.

**Status**: Complete.

## Fixed: unsaved "Add new Group/Resource" entity silently discarded on navigate-away

**Problem**: opening the "+ Add <Type>" inline form on a collection
page, entering data, then navigating away (breadcrumb click, section
switch, view switch) before hitting Save gave no unsaved-changes
warning — the in-progress new entity was just silently discarded, with
no indication anything was lost.

**Root cause**: `_addNewOpen`/`_addNewData` (the Add form's state) were
never checked by any of the existing dirty-guard blocks in `pushState()`
/ `setDataView()` — those only check `_dataDirty`, `_metaDirty`,
`_verDirty`, `_modelDirty`, `_capDirty`, `_jsonEditDirty`.

**Fix** (`registry/ui/app.js`):
- New `isAddNewFormDirty()` helper — true if `_addNewOpen` is set AND
  either `_addNewData` has any keys, or the live `addNewIdInput` DOM
  element (whose value is only read at save-time, with no
  onchange/oninput handler wired to `_addNewData`) has a non-empty
  trimmed value.
- `saveNewEntity()` signature extended to accept an optional `cb`
  callback (calls it on success instead of the default `refresh()`,
  preserving old behavior when called with no args from the plain
  "Create" button).
- Added a matching guard block to both `pushState()` and
  `setDataView()` — triggers when on a collection page
  (`isCollection(_state.path)`) with `isAddNewFormDirty()` true and the
  navigation would actually change something relevant. Shows
  `showLeaveEditDialog()` with a single Save button (creating the
  entity is always a PUT-by-id, never PATCH, so no dual-button variant
  is needed) and Discard (calls `cancelAddEntity()`).

**Verified** via CDP against a live xrserver (`dirs` collection):
opening the Add form, typing an ID, then clicking a breadcrumb link
correctly shows the Save/Discard/Cancel dialog; Save creates the entity
via PUT and navigates to the destination; Discard navigates away
without creating anything (confirmed via a 404 on the would-be entity's
URL); an empty/untouched Add form navigates away silently with no
dialog, as before. Test data cleaned up.

**Status**: Complete.

## Fixed: JSON view meta breadcrumb shown after switching back to List view

**Problem**: viewing a Resource/Version's Metadata tab in List view, then
switching to JSON view, correctly showed "meta" as an extra breadcrumb
segment — but switching back to List view kept showing "meta" too,
even though List view's own Metadata tab bar already makes the
selection clear and was never meant to show the extra segment.

**Fix**: `buildBreadcrumbSegments()` (`registry/ui/app.js`) now only
appends the extra "meta" segment when `_state.dataView === 'json'`, in
addition to the existing Resource/Version-depth and `docTab === 'meta'`
checks. List view (`dataView === 'table'`) never gets the segment,
regardless of how it was reached (fresh tab click, or a full re-render
after switching back from JSON view).

**Verified** via CDP: List view → Metadata tab → JSON view shows
`.../file1/meta`; switching back to List view shows the breadcrumb
stopping at `file1` (no "meta"), while the "File Metadata" tab remains
selected/active, exactly as expected.

**Status**: Complete.

## Added: dismissible "X" close button on error banners

**Problem**: the inline `.error-banner` divs shown after a failed
Save/Create/Delete (Add-new-entity, Data/Meta/Version entity editors,
JSON raw edit's Save and Format validation, Model Source, Capabilities)
had no way to dismiss them except fixing the underlying error and
retrying (which happens to also clear the banner as a side effect) —
there was no direct affordance to just close the message.

**Fix** (`registry/ui/app.js`, `registry/ui/style.css`):
- Added two small helpers, `showErrorBanner(div, message)` and
  `hideErrorBanner(div)`, that every error-banner call site now funnels
  through instead of directly poking `.style.display`/`.textContent`.
  `showErrorBanner()` builds the banner's content as a message `<span>`
  plus a `.error-banner-close` `<button>` (rendered as an "×") whose
  `onclick` calls `hideErrorBanner()` on the same div.
- Mechanically replaced every existing
  `div.style.display = 'block'; div.textContent = msg;` /
  `div.style.display = 'none'; div.textContent = '';` pair (across
  `saveNewEntity()`, `saveDataEntity()`, `deleteDataEntity()`,
  `saveMetaEntity()`/its delete, `saveVersionEntity()`/its delete,
  `jsonEditFormat()`, `jsonEditSave()`, `saveModel()`,
  `saveCapabilities()`/its delete) with calls to the two helpers.
- `.error-banner` CSS gained `position: relative` and right-side
  padding to make room for the new `.error-banner-close` button,
  absolutely positioned top-right.
- Also gave the Model Source section's dynamically-created
  `#editorError` div the `error-banner` class it was missing (a small
  pre-existing inconsistency noticed while doing this sweep), so its
  close button renders styled consistently with the others.

**Verified** via CDP: triggering a Create error (invalid dir ID) shows
the red banner with a working "×" in its top-right corner; clicking it
hides the banner (`display: none`). Also verified the JSON raw-edit
view's "Invalid JSON" banner (triggered via `jsonEditSave('PATCH')`
with malformed textarea content) renders and dismisses the same way.

**Status**: Complete.

## Fixed: JSON-view Save losing inline content (missing `$details` suffix
race)

**Problem**: editing/saving a Resource or Version in JSON view sometimes
lost previously-applied `inline=` content after Save — the post-save
refresh wasn't using the same URL-building logic as a normal page load.

**Root cause (two related bugs)**:
1. `jsonEditSave()`'s post-save refresh used `buildBaseURL()` (no query
   params at all) instead of `buildAPIURL()` — fixed by refreshing via
   `fetchWithDetailsFallback(buildAPIURL(), needsDetails(_state.path))`,
   matching `refresh()`'s own logic exactly (a plain `buildAPIURL()` GET
   doesn't honor `$details` on a document-backed Resource/Version, so it
   fetched the raw document instead of entity JSON).
2. `renderJSONEditView()` computed its target save URL (`_jsonEditURL`,
   via `jsonEditTarget()`, which decides whether to append `$details`)
   only once, the first time edit mode was entered — but that decision
   depends on `_modelCache` already being populated
   (`resourceHasDocument()`). On a fresh/bookmarked direct URL load
   straight into `edit=1&dview=json`, the model isn't cached yet, so
   `$details` was silently omitted and locked in for the rest of the
   session (this is what caused the separate-looking "file: null saves a
   weird value" bug reported by the user — the PUT was going to `f1`
   instead of `f1$details`, so it replaced the whole document body
   instead of updating attributes). Fixed by calling
   `ensureModelCached()` before computing `_jsonEditURL`.

**Verified** via isolated headless-Chromium testing (throwaway `xrserver`
instances, never the user's live server): inline content now survives
Save; setting an inlined attribute to `null` and saving correctly clears
it server-side instead of reverting to old document content.

**Status**: Complete.

## Fixed: breadcrumb JSON-view options (inline/sort/docView/binary/
collections) not remembered per-depth

**Problem**: In JSON view, applying `inline=`/`sort=`/doc-view/binary/
collections at one depth of the hierarchy (e.g. inlining "file" on a
Resource, then again on its Version) leaked those options onto every
ancestor breadcrumb (even the registry root), and there was no way to
restore a given depth's own previously-applied options when navigating
back up to it — unlike `filters`, which already had correct per-depth
memory via `_state.crumbURLs`.

**Root cause**: `_state.crumbURLs`/`rootApiURL` (the only genuine
"memory" the app kept per hierarchy depth) only ever stores the bare
`filter=`-bearing API link — `buildAPIURL()` always appends
`inline=`/`sort=`/`doc`/`binary`/`collections` freshly from the CURRENT
global `_state` at fetch time, so there was nothing in the existing
cache to recover these options from. `pageHref()` (breadcrumb/tile hover
hrefs) blindly copied the deepest page's `_state.inlines` etc. onto
every ancestor's synthetic link, and `pushStateReal()`'s "fresh
navigation defaults" block hardcoded `inlines: [], sort: '', docView:
false, binary: false, collections: false` on every real navigation
(including breadcrumb clicks), with no per-depth restoration at all.

**Fix** (Option B, per user: reuse existing URL-parsing logic rather
than a bespoke snapshot object):
- Added a shared `parseJSONOptionsFromQuery(qs)` parser (extracts
  `inline`/`sort`/`doc`/`binary`/`collections` from a query string),
  reused by `loadStateFromURL()` instead of duplicating field parsing.
- Added `_state.crumbOpts[]`/`_state.rootOpts` — a per-depth cache of
  the query-string portion of `buildURL(_state)` (mirrors `crumbURLs`/
  `rootApiURL`'s existing pattern exactly), populated on every
  `pushStateReal()` call and reset/trimmed alongside `crumbURLs` on
  server/section/path changes.
- `pageHref()` now restores `inlines`/`sort`/`docView`/`binary`/
  `collections` for a breadcrumb/tile's destination depth by parsing
  its cached `crumbOpts[depth-1]`/`rootOpts` snapshot, instead of
  copying the current page's values — fixes the hover-href leak.
- `pushStateReal()`'s "fresh navigation defaults" block now computes
  `defaultOptsQS` from the same `crumbOpts`/`rootOpts` cache (mirroring
  how `defaultApiURL`/`defaultFilters` already restore from
  `crumbURLs`) and merges those as defaults instead of hardcoded
  blanks — this is what makes an actual breadcrumb click (not just the
  hover preview) correctly restore a previously-applied depth's
  options. `sort` is additionally guarded to blank when the destination
  isn't a collection (sort is invalid on non-collection paths, same
  guard already used elsewhere).

**Verified** via CDP (isolated test server, throwaway db): inlined
"file" on `dirs/d1/files/f1`, navigated to its version `v1` (confirmed
`_state.inlines` correctly reset there — no leak), then clicked (not
just hovered) the `f1` breadcrumb from `v1` — `_state.inlines` correctly
restored to `["file"]`, the "file" inline checkbox in the left panel was
checked, and the page actually rendered the inlined file content,
confirming the fix works end-to-end for real navigation, not just the
hover-preview href. `node --check app.js` passes. Test server/chromium/
temp files cleaned up.

**Status**: Complete.

## Fixed: Versions List button label order ("Versions (N) List" ->
"Versions List (N)")

**Problem**: the Resource page's "Versions List" link (a real navigation
to the raw Versions collection page) showed the count BEFORE the word
"List" — "Versions (2) List" — which could be misread as if "2" were
part of a versionid rather than a count.

**Fix**: moved the count to the end — now reads "Versions List (2)".

**Status**: Complete.

## Fixed: JSON-view "collections" option available on ineligible entity
types

**Problem**: the JSON left-nav panel's "collections" checkbox was shown
whenever the server declared the `collections` capability flag,
regardless of what entity type the JSON viewer was currently pointed at.
Per spec (and enforced server-side — see `registry/info.go`'s
`bad_flag`/"?collections is only allow on the Registry or Group instance
level" check), `collections=` is only valid at the Registry root or a
Group instance; sending it for any other entity type (a collection
itself, a Resource, a Version, etc.) causes the server to return an
error.

**Fix** (`registry/ui/app.js`):
- Added `collectionsEligible(path)` — true only when `path.length === 0`
  (Registry root) or `path.length === 2` (Group instance).
- The "collections" checkbox in the left panel now only renders when
  `hasF('collections') && collectionsEligible(_state.path)`.
- As defense-in-depth (in case `_state.collections` lingers `true` from
  an earlier eligible page), also gated the `collections=` query param in
  both `buildAPIURL()` (the actual server-fetch URL) and `buildURL()`
  (the address-bar URL) behind the same `collectionsEligible()` check, so
  it can never actually be sent to the server for an ineligible entity
  type even if the state variable is stale.

**Verified** via CDP (isolated test server): checkbox visible at
Registry root and at a Group instance (`dirs/d1`); hidden at a
Group-type collection (`dirs`) and at a Resource instance (`f1`).

**Status**: Complete.

## Fixed: List view's Metadata tab rendered map/object attributes as raw
JSON instead of a pretty nested tree

**Problem**: viewing a Resource/Version's Metadata tab in List view (not
edit mode), a map-typed attribute like `labels` displayed as a raw
`JSON.stringify()` dump (e.g. `{"none":"","stage":"dev"}`) instead of the
same pretty nested key/value tree every other List-view Property table
uses for map/object/array attributes.

**Root cause**: `renderMetaTable()`'s `buildRow()` had its own
`typeof val === 'object'` branch that just did `esc(JSON.stringify(val))`
— it never called `renderValueTree()`, unlike the entity's own Property
table (`buildPropsRowsHtml()`), which already handled the identical case
by rendering a proper nested tree.

**Fix** (`registry/ui/app.js`): changed `renderMetaTable()`'s object
branch to match `buildPropsRowsHtml()`'s exact pattern — empty
maps/arrays show `<span class="vt-empty">empty</span>`, non-empty ones
render via `renderValueTree(val, 0, model, metaPath, [k], d)`, with the
`cell-tree` CSS class applied to the value cell for correct layout.

**Verified** via CDP: the Metadata tab's "Labels" row now renders as a
`.vt-obj`/`.vt-kv` nested tree (`none:` / `stage: dev`) instead of raw
JSON text.

**Status**: Complete.

## Fixed: JSON edit view Save button wording + Apply-while-dirty guard

**Problem** (2 related requests):
1. JSON edit view's Save buttons said "Save (full)"/"Save (delta)" —
   fine for the Data/Meta/Version Property-table editors (friendlier,
   less technical wording), but misleading for JSON view specifically,
   where technical users editing raw JSON directly want to know the
   literal HTTP verb being used ("delta" doesn't obviously mean PATCH).
2. In JSON edit view, clicking "Apply" in the left nav (to apply a new
   filter/sort/inline/doc/binary/collections combination) silently
   discarded any unsaved edits in the JSON textarea and re-fetched fresh
   data — with no warning, unlike every other real navigation away from
   a dirty JSON edit (breadcrumb click, view switch, etc.), which already
   prompts via `showLeaveEditDialog()`.

**Fix** (`registry/ui/app.js`):
1. JSON edit view's own action-bar buttons now read "Save (PUT)"/"Save
   (PATCH)" instead of "Save (full)"/"Save (delta)". `showLeaveEditDialog()`
   gained a new `usePutPatchLabels` parameter (5th arg) that swaps its
   own two Save-button labels the same way — passed `true` only by JSON
   view's two callers (in `pushState()`'s and `setDataView()`'s
   leaving-JSON-edit guards), so the Data/Meta/Version entity editors'
   leave-dialogs keep their existing "Save (full)"/"Save (delta)"
   wording unchanged.
2. `applyJSONOptions()` was split into a thin guard + `applyJSONOptionsReal()`
   (the original logic, unchanged). The guard checks
   `_state.dataView === 'json' && _state.editMode && _jsonEditDirty` and,
   if true, shows the same leave-edit dialog (Save PUT / Save PATCH /
   Discard / Cancel) before proceeding — approving any option (Save or
   Discard) resets the JSON edit buffer and then runs
   `applyJSONOptionsReal()` (refreshing the data with the new options);
   Cancel leaves the dirty edit and the previous options untouched.

**Verified** via CDP: JSON edit buttons show "Save (PUT)"/"Save (PATCH)";
typing a dirty edit then calling Apply shows the leave dialog; Cancel
correctly leaves the dirty textarea content untouched; Discard correctly
clears it and proceeds with the option change.

**Status**: Complete.

### Follow-up: Apply (not dirty) silently refetched but didn't refresh the JSON edit textarea

**Problem**: In JSON edit view, changing a left-nav option (filter/sort/
inline/doc/binary/collections) while NOT dirty and clicking Apply sent a
new request to the server (confirmed via network trace) but the visible
textarea kept showing the stale pre-Apply content.

**Root cause**: `renderJSONEditView()` only (re)seeds its buffer
(`_jsonEditOrigText`/`_jsonEditDraftText`) from freshly-fetched data when
`_jsonEditOrigText` is still `null` — a guard meant to protect an
in-progress edit from being silently clobbered by a background refresh.
Since Apply (not dirty) skips the leave-dialog guard and goes straight to
`applyJSONOptionsReal()`, the buffer was already initialized from the
previous render, so the new data never made it onscreen even though it was
fetched.

**Fix**: `applyJSONOptions()` now calls `resetJsonEditBuffer()` before
re-fetching whenever in JSON edit mode and not dirty (nothing to lose),
forcing `renderJSONEditView()` to re-seed from the fresh response.

**Verified** via CDP: toggled an inline option while in edit mode (not
dirty), clicked Apply — textarea now correctly shows the newly-fetched
(inlined) data instead of the stale pre-Apply buffer.

**Status**: Complete.

### Follow-up: clear JSON edit error banners on any action-bar button click

**Problem**: JSON edit view's two error banners (`jsonEditError` for
server-side Save/Delete failures, `jsonEditInvalid` for client-side JSON
parse errors from Format/Save) weren't consistently cleared when a
different button was clicked afterward — e.g. a stale Save error would
still show after clicking Format or Undo.

**Fix** (`registry/ui/app.js`): added a `clearJsonEditErrors()` helper
(hides both banners) and call it at the start of `jsonEditFormat()`,
`jsonEditUndo()`, `jsonEditSave()`, and `jsonEditDelete()` — so clicking
any action-bar button clears whatever error was left over from a previous
action.

**Verified** via CDP: bad Format leaves the invalid banner visible; Undo
clears it; bad Save shows the error banner; a subsequent Format call
clears it; a valid Save after a bad Format also clears the invalid banner
(via the full re-render on success).

**Status**: Complete.

## Conventions

- Wrap text/comments in the `common/` directory and in this file
  (`plan.md`) to 80 characters per line.

## Fixed: JSON view's "All" expand/collapse button inert in Document mode when the document is JSON

**Bug**: viewing a resource/version's Document content in JSON view
(the "Document" toggle, `dview=json` + Document mode) showed the "All"
expand/collapse button in the top-right corner, but clicking it did
nothing whenever the document itself happened to be JSON (rendered as a
collapsible twisty tree, same as Details mode).

**Root cause**: `buildJsonExpandAllBtnHtml(true)` renders the button
genuinely HTML-`disabled` up front (Document mode doesn't yet know if the
about-to-be-fetched content is JSON or plain text/binary). Once the fetch
resolves and the content turns out to be JSON,
`renderJSONViewDocumentMode()`'s `showJSONTree()` helper only removed the
button's `json-exp-btn-disabled` CSS class — it never cleared the actual
`disabled` DOM attribute. A disabled `<button>` never fires click events
at all, so the button looked enabled but stayed inert.

**Fix** (`registry/ui/app.js`, `renderJSONViewDocumentMode()`'s
`showJSONTree()`): also set `expBtn.disabled = false` alongside removing
the CSS class.

**Verified** via CDP against an isolated test server (port 9095, db
`copiloti_allbtn1`, cleaned up after): for both a Resource-level and a
Version-level document that happens to be JSON, switching to Document
mode leaves the "All" button with `disabled === false`, and clicking it
correctly toggles `data-open`/expands-collapses the tree (confirmed via
`jb1`'s `display` and the button's glyph/label). `node --check app.js`
passes.

**Status**: Complete.

---

## "Domain Focused" mode — hide xReg plumbing in View mode

**Goal**: a global, persisted Config-page toggle ("Domain Focused") that
hides xRegistry-specific chrome for end-users who just want a plain
domain-data catalog (e.g. a "schema registry"), without any per-domain
custom UI.

**Scope**: View mode only (edit mode and JSON view completely
unaffected). When on:
- Hides the "Identity" and "Versioning & State" Property-table categories
  on Registry root / Group / Resource / Version / the Resource page's
  Metadata tab. "Content" and "Timestamps" stay visible.
- Renames the "Extensions" bucket to "<Singular> Metadata" (e.g. "Schema
  Metadata"; "Registry Metadata" at the root, where there's no singular).
- Hides the Registry root's "Config:" pills row (Model/ModelSource/
  Capabilities/etc links).
- Exception: 3 Meta-level "Versioning & State" attributes stay visible
  even when the rest of that category is hidden — `defaultversionid`,
  `defaultversionsticky`, `readonly` — since they're useful, non-plumbing
  info for end users (which version is active, is it locked). Added per
  follow-up user feedback after the initial implementation.

**Implementation** (`registry/ui/app.js`):
- `_opts.domainFocused` (persisted boolean) + `optDomainFocused()` helper,
  following the exact `jsonColorMode` pattern.
- Config page: new checkbox row (`cfgSetDomainFocused()`), immediate
  effect via `refresh()` (no reload needed).
- `groupPropsByCategory()`: new `editable`/`extLabel` params. When
  `optDomainFocused() && !editable`: drops the Identity bucket, reduces
  Versioning & State down to just `DOMAIN_FOCUSED_KEEP_KEYS` (see above),
  and renames Extensions to the caller-supplied `extLabel`. Bug fixed
  during implementation: the function used to return `null` (meaning
  "render the caller's raw flat/unfiltered key list") whenever filtering
  collapsed the bucket count to ≤1 — which silently un-hid everything.
  Fixed by tracking total key count before/after domain-focus filtering
  and always returning the filtered array whenever any keys were actually
  removed, even if that leaves only one bucket.
- 3 call sites (`buildEntityPropsTableHtml()`, `renderMetaTable()`,
  `orderPropKeysFlat()`) updated to pass `editable` + a computed
  `extLabel` per depth/context.
- `buildRegEndpointPillsHtml()`: returns `''` when
  `optDomainFocused() && !_state.editMode`.

**Verified** via headless Chrome (Chrome-for-Testing standalone binary —
the machine's snap-packaged `chromium-browser` has a broken private-`/tmp`
mount preventing headless launch, unrelated to this change) against an
isolated test server (`schemagroups`/`schemas` model, extension attrs
`org`/`owner`/`team` added at each level):
- Config toggle persists across reload and applies immediately without a
  manual refresh.
- Registry root / Group / Resource (View mode): Identity + Versioning &
  State hidden, Extensions renamed to "Registry Metadata" / "<Group>
  Metadata" / "<Resource> Metadata" respectively.
- Metadata tab: Identity hidden; Versioning & State bucket still renders
  but only shows "Default Version ID" / "Default Version Sticky" / "Read
  Only" (their human-friendly `uiLabel`s) — `epoch`/`defaultversionurl`
  correctly still hidden; Timestamps unaffected.
- Edit mode: completely unaffected (Identity/Versioning & State still
  shown, "Extensions" label unchanged).
- JSON view: completely unaffected.
- Toggling back off restores exact prior behavior everywhere.
- `node --check app.js` passes throughout. Test data/model reset, test
  server killed, temp Chrome binary/profile/test scripts cleaned up. Live
  user server (port 8080, default db) untouched throughout.

**Status**: Complete.

---

## Property table cleanup (both xReg and Domain Focused views)

Follow-up polish after "Domain Focused" mode landed:

1. **Removed the redundant "Value" column header** from every Property/
   Meta table's `<th>` row (it's obviously the value column; keeping it
   made the table feel more spreadsheet-ish than intended). The header
   `<th>` is now empty (`<th></th>`) rather than removing the column
   entirely, so the table structure/widths stay unchanged.
2. **Renamed "<Singular> Property" → "<Singular> Details"** (and "Default
   Version/Version (n) Property" → "...Details") everywhere it appears:
   Registry root/Group/Resource/Version Property tables, the Add-entity
   form's table, and the Meta table — in both xReg and Domain Focused
   views.
3. **Domain Focused mode**: removed the text category sub-headings
   (General/Identity/Versioning & State/Content/Timestamps) — not enough
   content remains per category to justify a heading/separator once
   Identity/Versioning & State are hidden. The one exception is the
   Extensions/"<Singular> Metadata" bucket, which still gets a visual
   break — now a plain dark `<hr>` divider line instead of a text label.
   New shared helper `buildPropsCatRowHtml(group, domainFocused)` in
   `app.js` centralizes this (used by both `buildEntityPropsTableHtml()`
   and `renderMetaTable()`); groups are tagged `ext: true` by
   `groupPropsByCategory()` so the helper knows which bucket gets the
   divider instead of nothing.
4. **Row-banding bugfix**: removing the category headers in Domain
   Focused mode exposed a pre-existing quirk — `buildPropsRowsHtml()`'s
   banding always restarted at row 0 for each category group (harmless
   before, since a header row provided a visual break between groups).
   With headers gone, adjacent groups now sit flush against each other,
   so restarting the band count could produce two same-shaded rows
   touching. Fixed by threading a running band-offset across groups
   (`bandT`/`bandM` in the two call sites) — only when Domain Focused is
   active; the normal xReg view keeps its original per-group reset
   (harmless there, so left unchanged to minimize risk).

Verified via headless Chrome (Chrome-for-Testing standalone binary, same
tooling note as the Domain Focused section above) against an isolated
test server: xReg view header/labels/banding all unchanged from before;
Domain Focused view shows no text category headings except a divider
line before Extensions/"<Singular> Metadata", and row shading now
alternates continuously with no adjacent same-shade rows. `node --check
app.js` passes. Test server killed, temp files cleaned up, live user
server (port 8080, default db) untouched throughout.

**Status**: Complete.

---

## Domain Focused: keep "Is Default" + "Ancestor Version ID" visible

Extended `DOMAIN_FOCUSED_KEEP_KEYS` (app.js) to also keep `isdefault` and
`ancestorid` visible on the Version Details table in Domain Focused mode
(previously only `defaultversionid`/`defaultversionsticky`/`readonly`
were kept, for the Meta tab). Same mechanism as before — these two are
useful, non-plumbing info (is this the default version, which version it
descends from) even though the rest of "Versioning & State" stays hidden.

Verified via headless Chrome against an isolated test server: Version
Details table (Domain Focused ON) now shows "Is Default" and "Ancestor
Version ID" while `epoch` etc. remain hidden. `node --check app.js`
passes. Test server cleaned up; live user server untouched.

**Status**: Complete.

---

## Three small follow-up fixes: self-ancestor hiding, Resource tab rename, new-version $details bug

**1. Domain Focused: hide self-referencing Ancestor ID row**

On a resource's first/root version, `ancestorid` always equals that
version's own `versionid` (self-reference), which is a meaningless row
to show an end-user in Domain Focused mode. Added a check in
`buildEntityPropsTableHtml()`: when Domain Focused is on and
`entityData.ancestorid === entityData.versionid`, `ancestorid` is added
to the row-suppression set (on top of the existing `DOMAIN_FOCUSED_KEEP_
KEYS` mechanism that normally keeps it visible). xreg (non-Domain-
Focused) view is unaffected — still always shows Ancestor ID, including
the self-referencing case.

**2. Resource page tab rename: "<Singular> Metadata" -> "<Singular> Details"**

Confirmed with the user this refers to the Resource page's tab
(`tabDefs.push({ key: 'meta', ... })`, e.g. "Schema Metadata"), not the
Domain-Focused-only Extensions-bucket rename inside the Property table
(a separate, already-existing mechanism). Renamed the tab label from
`capTypeT + ' Metadata'` to `capTypeT + ' Details'` — applies in both
xreg and Domain Focused views, since the tab bar itself doesn't depend
on the Domain Focused setting.

**3. Fixed: creating a new Version via the Versions List "+ Add" form
didn't append `$details` to the PUT URL**

`saveNewEntity()` (the shared Add-entity-form save handler, used for
Groups/Resources/Versions alike) only appended `$details` when
`_state.path.length === 3` (creating a Resource) AND the model's
`resourceHasDocument()` was true — so creating a new Version
(`_state.path.length === 5`) never got `$details` at all. For a
`hasdocument=true` resource, a plain (non-`$details`) PUT body is parsed
as the literal document content, not metadata — so the new version's
metadata-only creation body (even `{}`) was being stored as its document,
corrupting/hiding the real document slot.

Per user: "using $details all the time... will always work so no need
to check hasDoc first" — simplified the condition to unconditionally
append `$details` whenever creating a Resource (`path.length === 3`) or a
Version (`path.length === 5`), dropping the `resourceHasDocument()` check
entirely (harmless no-op when there's no document concept, e.g. Groups
are unaffected since `path.length` there is 1).

**Verified** via headless Chrome-for-Testing against an isolated test
xrserver (port 9095, db `copiloti_test3`, dropped afterward):
- Self-ancestor hiding: root version's Ancestor ID row absent in Domain
  Focused mode, present in xreg view.
- Tab rename: "Schema Details" tab label confirmed present, "Schema
  Metadata" confirmed absent, in both views.
- $details fix: captured the actual PUT network request when creating
  version "2" via the Versions List "+ Add Version" form — confirmed URL
  is `.../versions/2$details`; confirmed via direct API fetch afterward
  that `$details` returned the correct metadata (`versionid`, `ancestorid`,
  etc.) and the raw (non-`$details`) document endpoint returned an empty
  body, not the metadata JSON — proving the document slot was correctly
  left untouched.
- `node --check app.js` passes throughout. Test server killed, test DB
  dropped, Chrome/puppeteer temp artifacts removed. Live user server
  (default port/db) untouched throughout.

**Status**: Complete.

---

## Meta tab: make defaultversionid editable when sticky is true; JSON view header solid background

**1. `defaultversionid` on the Meta tab is now editable, gated by `defaultversionsticky`**

Previously `defaultversionid` was unconditionally hard-coded read-only in
edit mode (a link to that version) — this was an intentional but
temporary scoping decision from a prior round. Per user request, it's now
editable exactly when `defaultversionsticky` is `true` (this is what
tells the server to honor an explicit pinned default instead of always
tracking the newest version). When sticky is `false`, the field stays
the read-only link, now with an inline hint: "To edit, set Sticky to
'true'." Toggling the `defaultversionsticky` checkbox forces a full
Meta-tab re-render (`rerenderMetaTab()`) so `defaultversionid`'s row
immediately flips between its read-only/hint form and its editable-input
form, matching the existing `ifvalues` reactivity pattern used elsewhere
in `metaEditFieldChange()`.

**2. JSON view's sticky "Server: ..." header now has a solid background**

`.json-exp-wrap` (the sticky toolbar row showing "Server: <url>" plus the
Details/Document toggle and Expand-all button) had `background:
transparent`, so scrolled JSON content visually overlapped/collided with
the header text as the user scrolled the page. Changed to `background:
ghostwhite` (matching `#json-output`'s background) so the header now
sits on a solid, visually separated bar. Applies to both the read-only
JSON view and the JSON edit view (shares the same `.json-exp-wrap` CSS
class).

**Verified** via headless Chrome-for-Testing against an isolated test
xrserver (port 9096, db `copiloti_test4`, dropped afterward):
- Confirmed `defaultversionid` row renders as a read-only link + hint
  text when sticky is `false`; toggling the sticky checkbox re-renders
  the row as a real `<input type="text">` bound to
  `metaEditFieldChange('defaultversionid', this)`.
- Confirmed `.json-exp-wrap`'s computed background-color is
  `rgb(248, 248, 255)` (ghostwhite), no longer transparent.
- `node --check app.js` passes. Test server killed, test DB dropped,
  Chrome/puppeteer temp artifacts removed. Live user server (default
  port/db) untouched throughout.

**Status**: Complete.

---

## Meta tab: defaultversionid dropdown + revert-on-sticky-off

Follow-up to the previous "editable defaultversionid" round:

1. **Dropdown instead of free text.** `defaultversionid` must reference an
   actual version, so a free-text input risked typos. Now renders as a
   `<select>` populated from `_resVersionsList` (the same versions
   collection already fetched/cached for the Resource page's "Version:"
   selector) with the current value pre-selected. Falls back to the
   previous plain text input if that list hasn't loaded yet (rare
   race — the fetch is normally already in flight/done by the time the
   Meta tab is opened).
2. **Revert on sticky-off.** If the user edits `defaultversionid` (with
   sticky on) and then flips `defaultversionsticky` back to `false`
   before saving, the in-progress `defaultversionid` edit is now reverted
   back to its original server value in `_metaEditData` (since it can no
   longer be shown/saved as editable once sticky is off) — handled in
   `metaEditFieldChange()`'s existing `defaultversionsticky` special case.

**Verified** via headless Chrome-for-Testing against an isolated test
xrserver (port 9097, db `copiloti_test5`, 3 versions, dropped afterward):
- Confirmed the rendered `<select>` lists all 3 real version IDs (1, 2,
  3) with the current one selected.
- Confirmed selecting a different version (e.g. "2"), then toggling
  sticky off, reverts the field back to showing the original value ("3")
  as a read-only link + hint, not the unsaved "2".
- `node --check app.js` passes. Test server killed, test DB dropped,
  Chrome/puppeteer temp artifacts removed. Live user server untouched
  throughout.

**Status**: Complete.

---

## Fixed: Meta tab / Version Details tab "stale DOM on discard" + "edits silently wiped by a redundant render" bugs

Two related bugs found while chasing a user report that the Meta tab's
Save/Undo buttons sometimes stayed disabled even though edited rows showed
the dirty highlight.

### Bug 1 — tab-switch discard never re-rendered the panel being left

`switchDocTabReal()` only toggles CSS `display` on the Document/Version
Details/Meta tab panels — it never re-renders their content (they're built
once, lazily, and then just shown/hidden). Every "Discard" callback in
`switchDocTab()`'s leave-edit guards only reset state variables (`_metaDirty
= false`, `_dataDirty = false`, etc.) but never re-rendered the panel being
left, so stale (pre-discard) HTML — including old input values and
enabled Save/Undo buttons — stayed sitting in the DOM and reappeared
unchanged the next time that tab became visible again. Reported by the user
as: "I edit a version, leave to go to meta, dismiss my changes, then when I
go back to the version tab I see my edits again."

**Fix** (`switchDocTab()` in `registry/ui/app.js`):
- The "leaving Meta tab, discard" callback now calls `rerenderMetaTab()`
  before switching away.
- The "entering Meta tab" leave-guard condition was extended from
  `_dataDirty` alone to also check `_dataEditActiveKind === 'version' &&
  _verDirty` — it was previously missing the case where the user was
  editing a *non-default* version selected via the "Version:" dropdown
  (only checked the default-version/entity edit buffer).
- The "entering Meta tab, discard/save" callbacks now correctly branch on
  which buffer was active (`renderSingleEntity()`/`saveDataEntity()` for
  entity/default-version vs. `rerenderVersionPanel()`/`saveVersionEntity()`
  for a non-default version) instead of assuming only the entity buffer
  could be dirty.

**Verified**: edit Version Details' Name field → switch to Meta tab
(triggers leave-dialog) → Discard → switch back to Version Details →
field correctly reverted, Save button disabled.

### Bug 2 — a harmless redundant re-render could silently reset in-progress edits

`renderSingleEntity()` calls `loadMetaDetails()` once on its first pass
(model not yet cached) and then, once `ensureModelCached()`'s async
callback resolves for the *same* resource, calls `renderSingleEntity()`
again — a second, otherwise-harmless full re-render meant only to apply
the now-available model. But `renderSingleEntity()` unconditionally reset
`_metaData = null` on *every* call, and `loadMetaDetails()`'s `afterLoad()`
unconditionally re-cloned `_metaEditSrc`/`_metaEditData` from the freshly
(re-)fetched server data and reset `_metaDirty = false` on every call too —
so if a user started editing the Meta tab in the narrow window between the
first and second render, the second render's redundant reload would
silently discard the edit and re-disable Save/Undo, while any not-yet-
re-rendered dirty-row highlight from the first render's DOM could still
briefly linger, exactly matching the reported symptom.

**Fix** (`registry/ui/app.js`):
- New `_metaLoadedFor` global tracks which resource's `self` the current
  `_metaData`/`_metaEditSrc`/`_metaEditData` belong to. `renderSingleEntity()`
  now only resets that Meta-tab state when `_metaLoadedFor !== data.self`
  (i.e. genuinely a different resource), not on every redundant render for
  the same one.
- `loadMetaDetails()`'s `afterLoad()` now only (re)initializes
  `_metaEditSrc`/`_metaEditData`/`_metaDirty` the first time
  (`_metaEditData == null`) — a later, redundant call (whether via the
  early-return "already have `_metaData`" path or an actual second fetch)
  just re-renders using whatever edit state already exists, instead of
  clobbering it.

**Verified** via CDP against the live xrserver (read-only reconnaissance +
in-browser edits only, no Save clicks — never mutated the live registry):
traced `renderSingleEntity()`'s two calls directly, confirmed
`_metaLoadedFor` correctly matches on the second (redundant) call and skips
the reset; ran multiple sequential edit/undo/re-edit cycles (Compatibility
enum, a plain string field, a nested `deprecated.effective` timestamp via
its "Now" button, adding a Labels entry, adding an Extension) — Save/Undo
buttons and dirty-row highlighting stayed correctly in sync with
`_metaDirty` throughout every sequence tried.

`node --check app.js` passes. No changes needed elsewhere — this class of
redundant-render risk is specific to the Meta tab's own lazy-load path.

**Status**: Complete.

---

## Version page removal: dead-code cleanup + final verification

Final follow-up to "Retire the List-view 'dedicated Version page'" above.
The redirect/normalizer behavior (`normalizeVersionDepth()`,
`isVersionEntityPath()`, `setDataView()`'s guards) was already implemented
and verified in an earlier session; this pass removed the now-unreachable
List-view rendering code for depth >= 6 paths and re-verified end-to-end.

**Removed (confirmed dead via call-site tracing, not just line-range
deletion)**:
- `renderSingleEntity()`: the page-title/icon depth>=6 branches, the
  `dataEditDepth >= 6` disjunct in `isEditableEntityPage`, the entire
  `depthD >= 6` tab-building `else` branch (which pushed the only-ever
  `'eg-doc-panel-version'` lone tab), and the dead single-tab-shortcut
  branch that was only reachable at that same depth.
- `saveDataEntity()`: its `depthS >= 6` `$details`-suffix check — dead
  because `_dataEditData` (this function's only precondition) is now only
  ever snapshotted for depth 0/2/4 pages.
- `buildPropsRowsHtml()`/`buildEntityPropsTableHtml()`: the `isParentResLink`
  ("link back to parent Resource" row) and version-page-only
  `resourceSingular` derivation — dead because every caller of these two
  functions now only ever passes a depth <= 4 path (confirmed by tracing
  all 5 call sites: `renderSingleEntity()`, `rerenderVersionPanel()`,
  `onVersionSelectChangeReal()`, `refreshVersionDetailsPanel()`).
- `navigateToParentResource()` — fully dead (its only caller was the
  just-removed `isParentResLink` branch); function deleted entirely.

**Confirmed NOT dead / intentionally left unchanged** (traced individually
per the earlier plan's caution, not bulk-removed): `isVersionEntityPath()`/
`normalizeVersionDepth()`, `setDataView()`'s two depth>=6 guards,
breadcrumb-building's JSON-view "meta" segment logic, `buildTabAwareAPIURL()`'s
`isVersion` branch, `buildAddEntityFormHtml()`'s depth>=6 case (the
"Add Version" form on the Versions *collection* page, a different,
still-valid code path), `jsonEditTarget()`'s `$details` check (JSON view's
own Edit mode, which legitimately still operates at depth >= 6), and the
generic depth-classification helpers (`getSingularName()`,
`specAttrLevel()`, `specAttrLevelName()`, `jsonDocToggleApplies()`) shared
by JSON view.

**Verified** via headless-Chromium CDP against an isolated throwaway
xrserver (port 9095, db `copiloti_verdead1`, dropped after testing) with a
`dirs`/`files` model and a resource with 2 versions (1 default, 1 not):
- Resource page (depth 4, default and non-default `?ver=`) renders its tab
  bar (Document/Version Details/File Details/Versions List) and Properties
  table correctly, no JS errors.
- Registry root (depth 0) and Group instance (depth 2) render correctly.
- An old-style bookmarked depth-6 URL with no `dview` redirects to the
  Resource page + `?ver=`, as before.
- The same depth-6 URL with `dview=json` renders unchanged/unredirected —
  JSON view at depth >= 6 fully intact.
- Versions collection page (depth 5) renders correctly.
- Selecting a non-default version via the dropdown, then switching to JSON
  view, correctly navigates to that version's own real depth-6 JSON URL.
- `node --check app.js` passes. No console/page errors observed in any of
  the above scenarios.

All 4 tracking todos (`ver-page-normalizer`, `ver-page-setdataview-check`,
`ver-page-dead-code`, `ver-page-verify`) now marked done.

**Status**: Complete. The "Retire the dedicated Version page" project (all
phases: normalizer, redirect guards, dead-code removal, verification) is
fully finished.

---

## Fixed: Meta tab permanently stuck on "Loading…" (stale detached DOM
## element across an in-flight fetch)

**Root cause**: `loadMetaDetails()` captured `#eg-meta-box` once in a local
`box` variable, then wrote to that same closured reference inside its
`fetch(metaUrl).then(...)` callback. But `renderSingleEntity()` can
legitimately do a full re-render (replacing `main.innerHTML`, and with it
this exact element) *while that fetch is still in flight* — e.g.
`ensureModelCached()`/`ensureCapCached()` resolving asynchronously shortly
after the user switches to the Metadata tab (both trigger a fresh
`renderSingleEntity()`/`renderEntityFromData()` call once they resolve, a
pattern already used to fill in the model/mutable-state once available).
If the user's click happened before those resolved, `_state.docTab` was
already `'meta'`, so the *next* re-render's own tab-building code also
re-invoked `loadMetaDetails()` against a brand-new (but still
"Loading…") `#eg-meta-box`. Whichever fetch/write ended up targeting a
since-replaced, detached box did nothing visible — and if that happened to
be the last one to resolve, the *currently attached* box was left showing
its "Loading…" placeholder forever, with no further trigger to refresh it.
This explains the "no network traffic when I click the tab" symptom too:
the relevant fetch(es) already happened earlier (racing the model/
capabilities load), just before the user looked, and nothing fires again
on the click itself once `_metaData` reflects a completed (if
misdirected) fetch.

**Fix**: `loadMetaDetails()` no longer writes through the closured `box`
reference from inside its async callbacks. It now re-queries
`document.getElementById('eg-meta-box')` at each DOM-write point (success
and error paths), bailing out harmlessly if the element no longer exists.
This guarantees whichever fetch resolves *last* always paints whatever
`#eg-meta-box` is currently in the document, regardless of how many
overlapping re-renders happened in between.

**Verified**: directly simulated the race in a live page (clicked the
Metadata tab to start `loadMetaDetails()`'s fetch, delayed that fetch by
400ms via a monkey-patched `window.fetch`, then forced a full
`renderSingleEntity()` re-render 100ms later mid-flight to detach the
original box) — confirmed the Metadata panel still ends up correctly
populated instead of stuck on "Loading…".

**General pattern worth remembering**: any function that captures a DOM
element reference before an `await`/`.then()` and writes to that same
reference afterward is at risk of this exact bug if the surrounding page
can legitimately do a full-HTML-replacement re-render while the async work
is in flight (true for any Resource/Version page render, which can be
re-triggered by `ensureModelCached()`/`ensureCapCached()`/
`ensureOfferedCached()` resolving after the fact). Prefer re-querying
`getElementById()` right before each write instead of reusing an
earlier-captured reference.

**Follow-up: a second, more significant compounding root cause** (found
after the user offered a more precise theory: "I wonder if it thinks
something is already cached (but it's not) so it doesn't hit the network
but then it waits forever for it to appear in the cache — but never
does"): the shared global `_metaData` is written by **two independent**
code paths — `loadMetaDetails()` (List view's Metadata tab) and
`renderJSONViewForCurrentTab()` (JSON view's meta-segment) — and both had
an unguarded `if (_metaData) {...}` "already loaded, skip the fetch"
check with no verification that `_metaData` actually belonged to the
CURRENTLY-displayed resource. A late-arriving response from either path
(fired for a previous resource, or racing a resource-navigation) could
overwrite the shared `_metaData` with the WRONG resource's data right
after a legitimate reset had already correctly nulled it — so the very
next cache-check on the new resource wrongly treated it as "already
loaded" and skipped the real fetch entirely (explaining "no new network
traffic on click"), and if the mismatched resource/model combination made
`renderMetaBoxContent()` throw, the box was stuck on its placeholder
forever (uncaught exception, same "permanently stuck on Loading" symptom).

**Fix (more complete)**: both `loadMetaDetails()` and
`renderJSONViewForCurrentTab()` now snapshot which entity a call is
actually for (`requestedFor = _lastData` / the `data` parameter) and gate
every subsequent `_metaData`/`_metaLoadedFor` read AND write against that
snapshot still matching the current entity (`stillCurrent()` helper in
`loadMetaDetails()`; `_lastData !== data` checks in
`renderJSONViewForCurrentTab()`) — mirroring the same `_lastData === data`
idiom already used elsewhere in the file (e.g. `ensureModelCached()`/
`ensureCapCached()` re-render callbacks) to guard against acting on a
stale closure after navigation. A stale/foreign response is now silently
discarded instead of being committed, so it can never contaminate the
resource actually on screen. This is the more precise/complete fix for
the bug (Fix #1 above closed a related but secondary hole).

Also removed `toggleMetaBox()` as dead code while auditing this area — it
was Grid view's old collapsible meta-box toggler, unreachable since Grid
view was removed for the data section (see "Grid view removed..." entries
above); it had the exact same stale-`box`-reference bug as the original
`loadMetaDetails()`, so deleting it removes a latent copy of the bug
rather than leaving it to bite again if ever reintroduced.

**Verified**: re-ran the original stale-DOM-reference race simulation
(still passes, box no longer stuck on Loading) plus two new tests
directly exercising `loadMetaDetails()`'s cross-resource guard — (1)
confirmed `_metaLoadedFor` always matches the current entity's `self`
after a normal load, and (2) simulated resource A's delayed `/meta` fetch
resolving *after* `_lastData` had already been swapped to a different
resource B (with A's own reset already having nulled `_metaData`/
`_metaLoadedFor` for B, as `renderSingleEntity()` does on real
navigation) — confirmed A's late response is discarded and does NOT get
committed as B's `_metaLoadedFor`/`_metaData` (previously, before this
fix, an equivalent scenario without the guard incorrectly left
`_metaLoadedFor` pointing at resource A while `_lastData` was already B).

**Status**: still reproduced live after the two fixes above — see the
third, actual root cause below.

**Third root cause (the real one)**: found via the user's own debugging —
they reported that when switching to the Metadata tab on a resource, the
debugger showed `_metaData` already non-null (with what looked like a
valid meta object) at the point `switchDocTabReal()` checks it, yet the
panel never painted. The bug was in `switchDocTabReal()` itself (the tab
*click* handler — distinct from the initial-render path fixed above): its
own gate was

    if (tabKey === 'meta' && !_metaData) { loadMetaDetails(); }

This duplicates `loadMetaDetails()`'s own (now-correct) cache-validity
check, but does it wrong: `_metaData` is a global shared across
resources, so if the user had already viewed a *different* resource's
Metadata tab earlier in the session, `_metaData` was simply non-null
(leftover from that other resource) — `switchDocTabReal()` treated that
as "already loaded for this resource" and skipped `loadMetaDetails()`
entirely. But this new resource's Metadata *panel* is a fresh DOM element
whose innerHTML is still the initial "Loading…" placeholder (see
`renderSingleEntity()`'s tab-panel scaffold) — it had never actually been
populated. Since `loadMetaDetails()` was never called, nothing ever wrote
to that panel, and no fetch ever fired either (matching the "no network
traffic" symptom exactly, and matching the user's own guess: "it thinks
something is already cached... so it doesn't hit the network but then it
waits forever").

**Fix**: `switchDocTabReal()` now calls `loadMetaDetails()`
unconditionally whenever the Metadata tab is activated, exactly matching
the pattern already used elsewhere (`pendingDocTabActivate === 'meta'`
during the initial render always calls `loadMetaDetails()` too — see
around the "Kick off the lazy fetch for whichever tab ended up
default-selected" comment). `loadMetaDetails()` itself already has the
correct `stillCurrent()`/`_metaLoadedFor` cache-validity guard (from the
second fix above), so calling it unconditionally is cheap when the cache
is genuinely still valid, and correctly fetches fresh data otherwise.
Removing the redundant, buggy gate at the call site — rather than trying
to fix it in place — avoids having two different (and now
inconsistent) ideas of "is `_metaData` valid" living in two places.

**Verified**: reproduced the exact user workflow via CDP — loaded
resource A, clicked its Metadata tab (populating `_metaData` for A),
then client-side-navigated to a *different* resource B (schemas list →
click another schema row, no full page reload) and clicked B's Metadata
tab. Confirmed B's panel correctly populates with B's own meta data
(`_metaLoadedFor` matches B's `self`), not stuck on "Loading…" — this
exact sequence would have hung before this fix, since `_metaData` was
non-null (A's leftover) when B's tab was clicked.

**Status**: Complete (all three fixes verified). Awaiting the user's own
live confirmation, since this reproduced consistently for them but not
reliably in headless Chrome testing until this exact click sequence was
replicated.

## Collection page title parent prefixes + page-title color consistency

**Change**: collection page titles (Groups/Resources/Versions, depth
1/3/5) now show a parent-identity prefix, matching the existing
single-entity page title's "Singular: Value" shape but reversed
("Parent: pluraltype"):
- Groups collection (depth 1): `<RegistryName>: <plural group label>`
  (e.g. "CloudEvents: schemagroups") — RegistryName is `serverLabel()`,
  the same name shown as the first breadcrumb segment.
- Resources collection (depth 3): `<Group instance id>: <plural resource
  label>` (e.g. "Contoso.ERP: schemas") — dropped the owning Group's
  singular *type* name that used to prefix this (e.g. previously
  "schemagroup Contoso.ERP schemas"), since the id alone is enough
  context and matches the "Singular-Parent: Child" shape used elsewhere.
- Versions collection (depth 5): `<Resource instance id>: <versions>`
  (e.g. "Contoso.ERP.CancellationData: versions") — added the colon that
  was missing before.

**Color consistency fix**: `.eg-page-title-type` (used for the generic
type/category word half of every page title — "Registry:"/
"schemagroup:"/"schema:" on single-entity pages, and "schemagroups"/
"schemas"/"versions" on collection pages) previously rendered in a muted
blue-gray (`#557`) while `.eg-page-title-id`/`.eg-page-title-id-prefix`
(the actual identifier/name half) rendered dark (`#222`). The two colors
were semantically consistent (dark = specific id/name, muted = generic
type word) but visually looked inconsistent because which one appears
*before* the colon flips between single-entity pages (type word first)
and collection pages (parent id first) — @duglin found it distracting
("feels like my eyes are bugging out"). Fixed by making
`.eg-page-title-type` also `#222`, so every page title segment is now
the same dark color regardless of depth/ordering.

**Verified**: CDP-checked rendered title text + computed color on all 6
depths (0,1,2,3,4,5) against a live remote registry (read-only) —
correct prefix text and uniform `rgb(34,34,34)` color throughout.
@duglin confirmed live ("yup - thanks").

**Status**: Complete.

---

## Domain Focused option reversed: Domain view is now the unnamed default; opt-in renamed "xRegistry Focused"

**Rationale**: after "Domain Focused" mode shipped (see the "Domain
Focused" mode section earlier in this file, ~line 5512 onward, which
describes the ORIGINAL polarity/behavior and is left as historical
record, not rewritten), @duglin asked to flip which state is the
unnamed default. Domain view (hiding Identity/Versioning & State
categories and the Registry root's "Config:" pills row) is now simply
"how the app looks" with no checkbox needed; a new opt-in checkbox,
renamed **"xRegistry Focused"**, shows the full xReg attribute set when
checked. Unchecked (the new default) = Domain view.

**Changes** (`registry/ui/app.js`):
- `_opts.domainFocused` / `optDomainFocused()` / `cfgSetDomainFocused()`
  renamed to `_opts.xregFocused` / `optXregFocused()` /
  `cfgSetXregFocused()`. Default is now a plain falsy check (unset ==
  Domain view), reversing the old "undefined/true == hide xReg
  plumbing" polarity.
- Config-page checkbox id `cfg-domain-focused` → `cfg-xreg-focused`,
  label "Domain Focused" → "xRegistry Focused", tooltip/description
  text rewritten to describe *showing* xReg attributes (the new opt-in
  action) rather than hiding them. Re-alphabetized in the Options list:
  now "JSON coloring" then "xRegistry Focused" (was "Domain Focused"
  then "JSON coloring" under the old name).
- All 4 rendering call sites (`buildRegEndpointPillsHtml()`,
  `groupPropsByCategory()`, `buildEntityPropsTableHtml()`, the Meta-table
  renderer) now derive their local `domainFocused`/`domainFocusedT`/
  `domainFocusedM` boolean as `!optXregFocused() && !editable` instead of
  `optDomainFocused() && !editable`. These rendering-level variable names
  (and `DOMAIN_FOCUSED_KEEP_KEYS`) were deliberately KEPT as-is — they
  describe "is domain-focused rendering active for this render pass", a
  concept whose meaning hasn't changed, only how it's derived from the
  (renamed/reversed) option.
- Updated surrounding comments throughout (`buildPropsCatRowHtml()`,
  `buildEntityPropsTableHtml()`, the Meta-table renderer, and 2 CSS
  comments in `style.css`) from "Domain Focused mode" phrasing to
  "Domain view" for consistency with the new naming.

**Note on existing users' localStorage**: anyone with a previously-saved
`_opts.domainFocused` key (from explicitly toggling the old checkbox, or
from an earlier in-session change that defaulted it to `true`) will have
that key silently ignored now, since the code no longer reads it under
that name. Their effective behavior resets to the new default
(`xregFocused` unset → Domain view) on next load — which happens to
match the intended new default anyway, so not treated as a real
migration problem.

**Verified** via headless-Chromium CDP against a live remote registry
(read-only), simulating a fresh user (`localStorage.clear()`):
- Fresh/default state: `optXregFocused()` is `false`, Property-table
  category headers (Identity/Versioning & State) are absent, Registry
  root's "Config:" pills row is absent, Meta tab's table also has no
  category headers — confirms Domain view is now the true default with
  no action needed.
- Config page: checkbox `cfg-xreg-focused` unchecked by default, labeled
  "xRegistry Focused", options ordered "JSON coloring" then "xRegistry
  Focused" (alphabetical).
- Checking the box: `optXregFocused()` becomes `true`, Property-table
  category headers reappear (General/Identity/Versioning &
  State/Timestamps).
- `node --check app.js` passes. Grep sweep confirms no remaining
  `optDomainFocused`/`_opts.domainFocused`/`cfgSetDomainFocused`/
  `cfg-domain-focused` references anywhere in `app.js`.

**Status**: Complete.

---

## Header toolbar redesign: single view-toggle button + kebab "more" menu for Edit/Config

**Motivation**: @duglin felt the header's top-right button row (Grid/
List/JSON/Edit/Config) was getting visually busy — great for power-user
testing (quick access to every mode), but likely just noise for a
typical user who won't switch modes often. Two changes, both approved:
1. Collapse Grid/List/JSON into a single button (every page only ever
   has 2 of the 3 enabled at once — Home: Grid vs List; everywhere
   else: List vs JSON — so it's always a boolean choice).
2. Move Edit + Config into a dropdown "more" menu.

**Design chosen for the view-toggle button ("1b")**: the button always
shows the icon/title of the *other* (destination) view — clicking
switches to it, and the icon then updates to show the new "other"
choice. (The alternative considered, "1a" — fixed default icon,
highlight to show which of the two is active — was not chosen.)

**Changes** (`registry/ui/index.html`, `app.js`, `style.css`):
- Replaced the 3 separate `data-dview` view-btn spans with a single
  `#view-toggle-btn`, whose icon/title `renderHeader()` sets dynamically
  from the one enabled "other" view (`otherView`/`_headerOtherView`).
  Hidden entirely when there's no other view to switch to (Config page;
  Home "types" group, which only ever offers List).
- New `toggleDataView()` switches to whatever `_headerOtherView`
  currently is.
- Removed the standalone `gear-btn` (Config) and `edit-btn` (Edit) from
  the always-visible header row.
- Added a pinned `#editing-indicator` (pencil icon) that's visible only
  while `_state.editMode` is true, so exiting edit mode always stays a
  single click — @duglin explicitly wanted this exception (starting an
  edit goes through the menu; leaving mid-edit does not).
- Generalized the existing narrow-screen "compact menu" system
  (previously only active below some header width) into a **permanent**
  kebab ("more") menu: renamed `compact-menu-btn`→`more-menu-btn` (icon
  changed from a ▾ down-caret to a ⋮ three-dot "kebab", @duglin's
  choice, used consistently for both the permanent menu and the narrow-
  screen fallback), `openCompactMenu`→`openMoreMenu`,
  `buildCompactMenuItems`→`buildMoreMenuItems`. This menu now always
  contains "Edit" (when applicable) + separator + "Config", regardless
  of viewport width — not just when compact, as before.
- `setHeaderCompact()` simplified: it's now the single source of truth
  for the view-toggle button's `display`, combining both signals
  (`compact` fold-away OR no "other" view at all) — necessary because
  `renderHeader()` always ends by calling `renderBreadcrumbs()`, which
  re-invokes `setHeaderCompact(false)` and would otherwise clobber a
  "hide, no other view" decision made earlier in the same
  `renderHeader()` call. Narrow-viewport fallback: when truly out of
  header width, the view-toggle's one "switch to X" action folds into
  the same kebab menu (reusing the existing pill/breadcrumb-overflow
  width detection) so nothing is lost on small screens.

**Bug found and fixed during verification**: initial implementation set
the view-toggle button's `display` directly inside `renderHeader()`'s
otherView-computation block — this worked most of the time, but was
silently overwritten back to visible on the Home "types" page (and any
other single-view page) because `renderHeader()` always calls
`renderBreadcrumbs()` at its end, which unconditionally calls
`setHeaderCompact(false)`, resetting the button's display. Fixed by
moving the display decision into `setHeaderCompact()` itself (reading
the already-set `_headerOtherView` global), so it's correct no matter
which code path last touches it.

**Verified** via headless-Chromium CDP against a live remote registry
(read-only):
- Home "registry": toggle correctly offers Grid⇄List, icon/title update
  after each click.
- Home "types": toggle correctly hidden entirely (List-only, no other
  view).
- Resource page: toggle correctly offers List⇄JSON.
- Kebab menu on a data page: contains exactly "Edit" and "Config";
  clicking Edit turns on edit mode and shows the pinned editing-
  indicator; clicking the indicator exits edit mode and hides it again.
- Config page: toggle button hidden, kebab menu empty (no Edit/Config
  entries needed there).
- Narrow viewport (480px): toggle button hides, `_headerCompact` flips
  true, and the kebab menu correctly gains a "JSON view" entry (folded-
  in view-toggle action) alongside Edit + Config.
- `node --check app.js` passes; full grep sweep confirms no leftover
  `data-dview`/`gear-btn`/`compact-menu-btn`/`openCompactMenu`/
  `buildCompactMenuItems` references anywhere.

**Status**: Complete.

---

## Header toolbar icon refinement: hamburger for menu, table-style SVG for List view

**Follow-up to the header toolbar redesign above.** @duglin felt the
kebab (⋮) menu icon and the "≡" List-view glyph weren't distinct/clear
enough — supplied a reference image of a table/spreadsheet-style icon
(rounded rect, shaded header row, column divider, row lines) for List
view.

**Changes**:
- `viewIconHtml('table')` (`app.js`) now returns an inline SVG
  (`.hv-table-icon`) closely matching the reference: rounded-rect
  border, a shaded header-row band across the top, a vertical divider
  splitting a narrow first column, and 3 horizontal row divider lines —
  all drawn with `currentColor` so it recolors correctly with the
  button's hover/active states, matching the existing icon convention
  (`hv-grid-icon`, `filter-funnel-icon`).
- `#more-menu-btn`'s icon (`index.html`) changed from the ⋮ three-dot
  kebab to a proper 3-bar hamburger (`.hv-hamburger-icon`, 3 CSS-drawn
  bars) — unambiguous and distinct from the new table icon.
- `style.css`: removed the now-unused `.hv-table-sym` (old "≡" glyph)
  and `.more-menu-icon` (old kebab font-size) rules; added
  `.hv-table-icon` and `.hv-hamburger-icon`/`span` rules.

**Verified**: `node --check app.js` passes; CDP screenshot of the live
header (both List-view-active and JSON-view-active states, and the
opened hamburger dropdown) confirms both icons render crisply and are
visually distinct — table icon clearly reads as "list/table view",
hamburger clearly reads as "more options".

**Status**: Complete.

---

## Header toolbar polish round 2: sizing, Config-page artifact, Filter
## folded into the menu (same pattern as Edit Mode)

**Follow-up to the two entries above.** Several small polish items from
@duglin plus one bigger pattern change (Filter):

1. **Menu button too small** — `#more-menu-btn` had no explicit height
   (unlike `#editing-indicator`/`#filters-toggle-btn`, which already had
   `height: 28px`). Added `#more-menu-btn { height: 28px; }` to match.
2. **"Edit" → "Edit Mode"** — renamed the kebab menu's Edit entry label
   in `buildMoreMenuItems()` for clarity.
3. **Stray vertical line on Config page (and Home "types")** — root
   cause: `#view-controls` is a *bordered wrapper* div around the single
   `#view-toggle-btn` child. `setHeaderCompact()` was only hiding the
   inner button (`display:none`) when there's no "other view" for the
   page (Config; Home "types"), leaving the wrapper's own border/
   background rendered as an empty sliver — visually a thin vertical
   line. Fixed by having `setHeaderCompact()` hide the whole
   `#view-controls` wrapper itself, not just its child button.
4. **Kebab menu opens to nothing on the Config page** — `#more-menu-btn`
   is now also hidden entirely on the Config page (`renderHeader()`
   sets `display:none` when `isConfig`), since `buildMoreMenuItems()`
   always returns an empty list there (nothing to reach from Config:
   no Edit, and Config→Config would be a no-op).
5. **Filter folded into the kebab menu, same pattern as Edit Mode** —
   `#filters-toggle-btn` (the funnel icon) used to be a standalone
   toolbar button whenever filter/sort was supported. Changed to mirror
   the Edit Mode pattern established earlier: the entry point ("Filter")
   now lives in the kebab menu (`buildMoreMenuItems()`, order is now
   **Filter, Edit Mode, Config**), and the pinned funnel icon in the
   toolbar is shown *only* while the filters panel is actually open —
   the one-click way to close it (title changed to "Close Filters /
   Sort"). New module-level `_filtersMenuAvailable` (set by
   `renderHeader()`, mirroring `_headerOtherView`'s pattern) records
   whether the current page supports filter/sort at all, so
   `buildMoreMenuItems()` knows whether to show the "Filter" entry.
6. **View-toggle button changed width when switching Grid⇄List⇄JSON** —
   `#view-controls` had no fixed width, so its only child button's width
   varied with icon-intrinsic size (grid icon 13×13 vs table SVG 18×14
   vs json `{}` glyph). Fixed: `#view-controls` now has an explicit
   `width: 40px`, and `#view-controls .view-btn` fills it
   (`width: 100%`, centered) — button box no longer resizes between
   view states.
7. **View icons looked smaller than Filter/Edit icons** — bumped icon
   sizes without changing the button box: `.hv-grid-icon` 13×13→17×17,
   `.hv-table-icon` 18×14→22×17, `.hv-json-sym` font-size 14px→18px
   (CSS width/height on the table SVG's class override its own inline
   `width`/`height` attributes, so no `app.js` SVG-markup change was
   needed for that one).

**Verified**: `node --check app.js` passes after each change. CDP
checks confirm: Config page — both `#view-controls` and `#more-menu-btn`
compute to `display:none` (header-right collapses to 0 width, no stray
line); Home "types" and Home "registry"/root pages — both still
`display:flex`, fully functional; view-toggle button width stays fixed
at 40px (inner button 38px) across Grid/List/JSON states; icon sizes
confirmed larger via `getBoundingClientRect()` (table 22×17, grid
17×17) while button box unchanged.

**Status**: Complete.

---

## "Show/Hide xReg Data": session override of the Default View config option

Extends the toggle pattern established for Edit Mode/Filter (see the
header toolbar polish entries above) to the existing "xRegistry
Focused" Config-page setting.

**Config page**:
- Replaced the single `#cfg-xreg-focused` checkbox with a "Default
  View" `cfg-option-group` radio pair (`cfgXregRadio()`, mirrors
  `cfgJsonColorRadio()`): **"Hide xReg data"** (today's default,
  Domain-focused) / **"Show xReg data"** (today's opt-in, full
  attribute set). Underlying storage unchanged — still the single
  `_opts.xregFocused` boolean; only the presentation and wording
  changed (dropped "Domain/Model/xRegistry Focused" jargon). Reordered
  alphabetically ("Default View" before "JSON coloring").

**Per-session override**:
- New `_state.xregOverride` (boolean, default `false`), persisted via
  a new `xrv=1` URL param in `buildURL()`/`loadStateFromURL()` — same
  mechanism as `_state.editMode`'s `edit=1`, so it survives navigation
  and a full page refresh, resetting only when explicitly turned off.
- New `effectiveXregFocused()` helper = `optXregFocused() !==
  _state.xregOverride` (XOR — override always flips away from the
  configured default). Replaces the raw `optXregFocused()` at all 4
  existing call sites: the registry-root "Config:" pills row
  (`buildRegEndpointPillsHtml()`), and the 3 `domainFocused`/
  `domainFocusedT`/`domainFocusedM` checks inside
  `groupPropsByCategory()`/`buildEntityPropsTableHtml()`/
  `renderMetaTable()` (covers Registry root, Group/Resource/Version
  instance pages, and the Meta tab, since they all funnel through the
  same shared builders).
- `toggleXregOverride()` flips `_state.xregOverride` via `pushState()`
  (same pattern as `toggleEdit()`), so the URL and the full page
  content update together in one call.

**Kebab menu entry**: added to `buildMoreMenuItems()`, order is now
**Filter, Show/Hide xReg Data, Edit Mode, Config**. Label reflects the
current *effective* state (not the literal configured default) — same
action-based "1b" convention as the view-toggle button: "Show xReg
Data" when currently hiding, "Hide xReg Data" when currently showing.
Gated by a new `_xregDataMenuAvailable` (set in `renderHeader()`,
mirrors `_filtersMenuAvailable`) — available on any data-section page
outside edit mode. **Deliberately available on collection pages too**
(not just single-entity pages that actually have a Property table) —
@duglin felt having it appear/disappear while traversing between
collection and single-entity pages was more visually jarring than
occasionally being a no-op on a collection page.

**Pinned toolbar icon**: new `#xregview-indicator` (standalone
`.icon-btn`, 28px height, positioned left of `#filters-toggle-btn`),
shown only while `_state.xregOverride` is `true`; click reverts to the
configured default. Icon is the official xRegistry "xR" mark (white
variant, from the CNCF artwork repo's
`projects/xregistry/icon/white/xregistry-icon-white.svg`), inlined
directly as SVG paths since the button is always shown in the "active"
(blue) state.

**Verified** via CDP against the live dev server: Config page radios
render/toggle correctly; kebab menu shows the correct contextual label
on both root and (simulated) collection pages; clicking it flips
`effectiveXregFocused()`, updates the URL (`xrv=1`), and shows/hides
the "Config:" pills row; pinned icon appears only while overridden and
reverts to default on click; loading a URL with `xrv=1` directly
(simulating a refresh) restores the override. `node --check app.js`
passes throughout.

**Status**: Complete.

### Kebab "more" menu icons

Small icon added to the left of each kebab-menu item's label, roughly
half the size of the equivalent toolbar button's icon, reusing existing
assets/techniques rather than new artwork:

- **Filter** → `.popup-icon-funnel`, same `clip-path` shape as the
  toolbar `.filter-funnel-icon`, just scaled down (9x8px vs 15x14px) —
  clip-path uses percentage coordinates so it scales cleanly with no
  separate asset needed.
- **Show/Hide xReg Data** → small colored (black + `#0066FF`) two-path
  "xR" SVG, reusing the same paths as the header `#xreg-logo` (NOT the
  all-white variant used by the pinned `#xregview-indicator`) — new
  `xregIconSmallHtml()` helper in `app.js`.
- **Edit Mode** → small pencil glyph (`&#9998;`, `.popup-icon-pencil`),
  flipped via `transform: scaleX(-1)` to match the pinned
  `#editing-indicator`'s icon.
- **Config** → small gear glyph (`&#9881;`, `.popup-icon-gear`), reused
  from the old pre-menu standalone `#gear-btn`. Sized at 19.5px font
  (50% larger than the initial 13px pass, per follow-up feedback).
- **Narrow-screen "switch to Grid/List/JSON view" fallback entry** →
  small grid/table/json icon, reusing `viewIconHtml()` with a new
  optional `small` boolean parameter that swaps in
  `.popup-icon-grid`/`.popup-icon-table`/`.popup-icon-json` classes
  instead of the toolbar-sized `.hv-*` classes (same SVG/markup
  otherwise, so both stay perfectly in sync for a given view).

All icons use `currentColor`, so they automatically inherit the
existing hover/active blue tint (`a.popup-item:hover`,
`.popup-item-active`) with no extra styling needed for that part.

**Implementation**:
- `openHeaderPopup()` (`app.js`) now renders an optional
  `item.icon` (HTML string) inside a fixed-width `.popup-item-icon`
  slot (16x16px, `flex-shrink: 0`) before the label. This is additive
  and backward-compatible: `openBcEllipsis()`/`openBcFull()`
  (breadcrumb popups, which share the same renderer) never set
  `icon`, so they render with no icon slot at all — unchanged layout.
- `buildMoreMenuItems()` (`app.js`) passes an `icon:` field for each of
  the 5 relevant entries (view-switch fallback, Filter, Show/Hide xReg
  Data, Edit Mode, Config).
- `style.css`: `.popup-item` changed from `display: block` to a flex
  row (`align-items: center; gap: 8px`) to lay out icon + label side by
  side; new `.popup-item-icon` fixed-size slot plus per-icon-type
  smaller-size modifier classes (`.popup-icon-funnel`,
  `.popup-icon-xreg`, `.popup-icon-pencil`, `.popup-icon-gear`,
  `.popup-icon-grid`/`-table`/`-json`).

**Follow-up fix**: the "Show/Hide xReg Data" menu entry and pinned
`#xregview-indicator` toolbar icon are now also hidden while in JSON
view (`_state.dataView === 'json' || _state.view === 'json'`) — the
setting only affects the human-readable table rendering (Property
tables / "Config:" pills row), so it has no visible effect on raw JSON
output and showing it there was misleading. Both share the same
`xregDataAvailable` computation in `renderHeader()`, so menu and
toolbar stay in sync automatically.

**Verified** via CDP against the live dev server: kebab menu on
Home/root (Config-only) and on a simulated data page with all four
toggle entries forced available — icons render correctly sized,
aligned, and visually balanced (gear bumped from 13px→19.5px after
initial review); breadcrumb ellipsis/full popups unaffected (no icon
slot regression, since they never pass `icon`); JSON-view exclusion of
the xR entry/icon confirmed via direct computation check. `node --check
app.js` passes throughout.

**Status**: Complete.

### Reduce redundant Home-page registry probing

@duglin noticed every Home-page visit — even just navigating back to it,
or switching between Grid/Tile and List layout — re-fetched
`/capabilities`, `/model`, and `/` for every listed registry via
`probeRegistry()`, with nothing cached across renders.

**In-memory probe cache**: new `_registryProbeCache` (keyed by
`normalizeURL(url)`) stores the full `info` object `probeRegistry()`
delivers. `probeRegistry(url, cb, force)` now checks this cache first —
on a hit (and no `force`), it calls `cb(cached)` immediately with zero
fetches; otherwise it does the existing 3-fetch sequence and populates
the cache before invoking `cb` (only on success — transient errors, e.g.
a momentarily unreachable host, are intentionally never cached, so
they're retried on the next visit rather than sticking around). Being a
plain JS variable (not `localStorage`), a real browser reload naturally
resets it — giving the "just returning to Home vs. hit refresh"
distinction @duglin asked about for free, no special detection logic
needed: in-app navigation/view-switching reuses the cache, an actual
page reload always re-probes everything, exactly as before.

**Auto-invalidation on mutations** (@duglin opted for this over leaving
it purely reload-driven, for accuracy): new `invalidateRegistryProbe(url)`
helper deletes a server's cache entry, called right after each of these
existing mutation-success paths (using `_state.serverURL ||
window.location.origin`): create-entity PUT success, bulk "Delete
Selected" success, single entity DELETE success, `saveModel()` success,
`saveCapabilities()` success. Version delete is intentionally excluded —
Home only shows group-type-level pills/counts, which versions never
affect.

**Manual global "Refresh" button** (@duglin's request — one button for
the whole page, not per-card/per-row): new pinned `#home-refresh-btn`
icon in `#header-right`, positioned left of `#view-controls`, shown only
while `_state.view === 'home'` (both Grid/Tile and List layouts share
it). Icon is a custom inline SVG (circular arc + arrowhead,
`.home-refresh-icon`, `currentColor` stroke, 17x17px) — replaced an
initial Unicode glyph (`&#8635;`) attempt after @duglin found it
unsatisfying. `doHomeRefresh()` wipes `_registryProbeCache`
entirely (simplest correct approach, since Home always lists every known
server anyway) and calls `renderHome()`, forcing a fresh re-probe of
everything currently shown; briefly adds a `.spinning` class (CSS
`@keyframes` rotation) for click feedback.

**Verified** via CDP against the live dev server (network-request
logging, not just visual): initial Home load does the expected 3
requests; switching Grid⇄List via `setDataView()` does zero further
requests; navigating away (Config) and back to Home does zero further
requests; clicking the refresh button (in both Grid and List layouts)
re-issues all 3 requests; a real `page.reload()` re-issues all 3
requests, matching pre-existing behavior. `node --check app.js` passes
throughout.

**Status**: Complete.
