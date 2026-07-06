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

- [ ] **Filter builder UI for JSON view** (in design, not yet
  implemented). Replace/augment the raw multi-line filter textarea with
  a guided builder. Sketch:
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
  - Open question: keep a small "Advanced (raw text)" toggle as an
    escape hatch for filter shapes the builder doesn't cover, or fully
    remove the raw textarea? Leaning toward keeping it as a fallback
    given the spec's dot-notation edge cases (arbitrary nested map
    keys, escaped wildcards) — to confirm with user before
    implementing.
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

- [ ] **Remove dead legacy CSS: `#left`/`#right` id selectors** in
  `registry/ui/style.css` (inside the `@media (max-width: 768px)` block
  around the model-editor rules). These ids are leftover from the old
  standalone model editor page's layout and no longer match anything in
  `index.html` or `app.js` — the ported editor now lives inside the
  SPA's unified `#left-panel`/`#main-view` layout. The surrounding
  `.editorLeftNav` / `.editorActionBar` / `.navToggleBtn` rules in that
  same block ARE still used and should stay; only the `#left`/`#right`
  selectors themselves are dead and safe to delete.

## Completed (for history / context)

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
