// ============================================================
// xRegistry SPA — app.js
// ============================================================
//
// xRegistry path structure (relative to registry root):
//   depth 0                                    Registry entity  (single)
//   depth 1  [G]                               Group collection
//   depth 2  [G, gId]                          Group entity     (single)
//   depth 3  [G, gId, R]                       Resource collection
//   depth 4  [G, gId, R, rId]                  Resource entity  (single) *
//   depth 5  [G, gId, R, rId, "versions"]      Versions collection
//   depth 5  [G, gId, R, rId, "meta"]          Meta entity      (single)
//   depth 6  [G, gId, R, rId, "versions", vId] Version entity   (single) *
//
//   * Resource and Version entities: append "$details" to get JSON metadata
//     (needed when hasdocument=true so the document body is not returned
//     instead of metadata).  $details is NOT valid on Groups or Registry root.
//     A fallback to plain GET is used in case the server returns 400.
//
// State is encoded in URL query params for bookmarkability.
// Servers (registry root URLs) are persisted in localStorage.
// ============================================================

// ---- State ----------------------------------------------------------------

var _state = {
  view:        'home',  // 'home' | 'tile' | 'table' | 'json'
  homeGroup:   'registry',  // 'registry' | 'types' — overridden from localStorage in init()
  homeLayouts: {registry: 'grid', types: 'grid'}, // per-group layout, overridden from localStorage
  dataView:    'grid',  // 'grid' | 'table' | 'json'
  serverURL:   '',      // full URL to registry root, e.g. 'http://localhost:8080'
                        // '' = same origin as the SPA
  section:     'data',  // 'data' | 'model' | 'modelsource' | 'capabilities' | 'capabilitiesoffered' | 'xregistry'
  path:        [],      // path segments relative to registry root (data section only)
  editMode:    false,
  xregOverride: false,  // session-level "Show/Hide xReg Data" override; see effectiveXregFocused()
  mutable:     false,

  // JSON-view query options
  inlines:     [],
  filters:     [],
  sort:        '',
  docView:     false,
  binary:      false,
  collections: false,
  useExport:   false,   // use /export endpoint instead of registry root (depth 0 only)

  // Link-driven navigation (data section only) — see "Link-driven navigation"
  // notes near buildAPIURL()/pushStateReal(). apiURL is the real, server-provided
  // absolute URL (self / <plural>url / versionsurl / metaurl / etc.) used to fetch
  // the CURRENT page's data — never constructed from `path` when a real link was
  // available. crumbURLs caches the real URL used at each visited depth this
  // session (parallel array to `path`), so breadcrumb clicks back to an
  // already-visited ancestor reuse it instead of reconstructing.
  apiURL:      '',
  crumbURLs:   [],
  // Cached real/filtered URL for the registry ROOT (depth 0) specifically.
  // crumbURLs only tracks depths > 0 (it's indexed by path segment), but the
  // root itself can also carry a real, filtered apiURL (e.g. a filter
  // applied directly at the root) — this mirrors crumbURLs' role for that
  // one special depth. See pushStateReal()/buildBreadcrumbSegments().
  rootApiURL:  '',
  // Per-depth snapshot of the JSON-view option flags (inline/sort/docView/
  // binary/collections) actually last applied AT that depth — the query-
  // string portion of the exact address-bar URL buildURL() would produce
  // for that page, captured whenever pushStateReal() lands on/updates it.
  // Parallel arrays to crumbURLs/rootApiURL (same depth-0-is-special split),
  // used by pageHref() to restore an ancestor's own real option state
  // instead of leaking the CURRENT (possibly much deeper) page's values
  // onto every breadcrumb/tile/row href — see parseJSONOptionsFromQuery().
  crumbOpts:   [],
  rootOpts:    '',

  // Resource/Version page Document/Details tab bar + version-selector
  // dropdown — remembered across a manual browser Refresh (but not across
  // navigating to a genuinely different resource, see pushStateReal()).
  // '' = default (first tab / "Default" version).
  docTab:      '',
  resVersion:  '',
};

// ---- Server/registry management (localStorage) ---------------------------

var LS_SERVERS     = 'xreg-servers';
var LS_OPTIONS     = 'xreg-options';
var LS_NAMES       = 'xreg-name-overrides';
var LS_PROXY       = 'xreg-proxy-servers';
var LS_DISCOVERED  = 'xreg-discovered-from';
var LS_HIDDEN      = 'xreg-hidden-servers';
var LS_LABELS      = 'xreg-label-cache';
// normalizedURL → last-known-good probed registry name. Persisted (not
// just in-memory) so that a server which happens to be offline/unreachable
// at page-load time doesn't lose its previously-learned display name —
// probeServer()'s failure path deliberately never writes an empty label
// here; only an actual successful GET is allowed to update (or clear, if
// the registry itself really reports registryid: "") this cache.
var _labelCache = (function() {
  try { return JSON.parse(localStorage.getItem(LS_LABELS) || '{}'); } catch(e) { return {}; }
})();
function saveLabelCache() {
  try { localStorage.setItem(LS_LABELS, JSON.stringify(_labelCache)); } catch(e) {}
}
var _modelCache    = {};  // normalizedURL → model JSON
var _capCache      = {};  // normalizedURL → capabilities JSON
var _offeredCache  = {};  // normalizedURL → capabilitiesoffered JSON
var _headerCompact = false;
var _fbDraft        = null;  // filter-builder working draft, see fbXxx() funcs
var _fbDraftKey     = null;  // server|section|path this draft belongs to
// Filters section collapse state — always starts collapsed on a fresh
// page load (not persisted to localStorage); stays as toggled while
// navigating during the current session.
var _filtersCollapsed = true;
// Grid/List "Filters" panel visibility toggle — separate from the JSON
// view's own left panel (always shown there). Default hidden so Grid/List
// stay full-width unless the user opens it. See plan.md "Filter support
// in Grid/List views".
var _filtersPanelOpen = false;
var _sortDraft        = null;  // sort-picker working draft, see sortXxx() funcs
var _sortDraftKey      = null;  // server|section|path this draft belongs to

// ---- Options (persisted) --------------------------------------------------
var _opts = (function() {
  try { return JSON.parse(localStorage.getItem(LS_OPTIONS) || '{}'); } catch(e) { return {}; }
})();

function saveOpts() {
  try { localStorage.setItem(LS_OPTIONS, JSON.stringify(_opts)); } catch(e) {}
}

function optJsonColorMode() { return _opts.jsonColorMode || 'full'; }

// "Domain view" — hiding xRegistry-plumbing Property-table categories
// (Identity, Versioning & State) in View mode, and renaming the
// "Extensions" bucket to "<Singular> Metadata" — is now the DEFAULT
// behavior (see plan.md "Reversing the Domain/xReg focus default").
// "xRegistry Focused" is the opt-IN Config-page checkbox that switches
// to the full xRegistry attribute set instead; it's off by default, so
// a plain truthy check on the (possibly unset) option correctly means
// "not xReg-focused, i.e. Domain view is active". View mode only — edit
// mode and JSON view always show every attribute regardless of this
// setting.
function optXregFocused() { return !!_opts.xregFocused; }

// Per-session override of optXregFocused(), toggled via the kebab menu's
// "Show/Hide xReg Data" entry (see buildMoreMenuItems()/toggleXregOverride())
// and the pinned #xregview-indicator toolbar icon. Persisted via the `xrv=1`
// URL param (see buildURL()/loadStateFromURL()) so it survives navigation
// and refresh, same as _state.editMode/_filtersPanelOpen — resets only when
// the user explicitly turns it off (menu entry again, or the pinned icon).
// Always flips AWAY from the configured default (see effectiveXregFocused()),
// never a second independent setting of its own.
function effectiveXregFocused() { return optXregFocused() !== !!_state.xregOverride; }

// Reflects the current JSON color-mode option onto <body> so the CSS
// rules in style.css (scoped via body[data-json-color=...]) can
// override the default per-token colors used by syntaxHighlight().
// 'full' (default) — today's behavior, all tokens colored.
// 'minimal' — everything black except linkified URL values.
// 'none' — everything black, including linkified URL values.
function applyJsonColorMode() {
  document.body.setAttribute('data-json-color', optJsonColorMode());
}
function optHomeGroup()   {
  // migrate legacy homeView key
  if (_opts.homeView !== undefined) {
    var g = _opts.homeView === 'flat' ? 'types' : 'registry';
    var l = _opts.homeView === 'table' ? 'list' : 'grid';
    _opts.homeGroup  = g;
    _opts.homeLayout = l;
    delete _opts.homeView;
    saveOpts();
  }
  return _opts.homeGroup || 'registry';
}
function optHomeLayouts() {
  // Migrate legacy single homeLayout → per-group object
  if (_opts.homeLayout !== undefined && !_opts.homeLayouts) {
    _opts.homeLayouts = {registry: _opts.homeLayout, types: 'grid'};
    delete _opts.homeLayout;
    saveOpts();
  }
  return _opts.homeLayouts || {registry: 'grid', types: 'grid'};
}
function currentHomeLayout() {
  // Home 'types' (cross-registry Group Types) page: Grid view has been
  // removed — always List, regardless of any previously-saved preference.
  // See plan.md "Grid view removed".
  if (_state.homeGroup === 'types') return 'table';
  return (_state.homeLayouts || {})[_state.homeGroup] || 'grid';
}

function loadServers() {
  try {
    return JSON.parse(localStorage.getItem(LS_SERVERS) || '[]').map(normalizeURL);
  }
  catch(e) { return []; }
}

function saveServers(list) {
  localStorage.setItem(LS_SERVERS, JSON.stringify(list));
}

function normalizeURL(url) {
  url = url.trim().replace(/\/$/, '');
  if (url && !/^https?:\/\//i.test(url)) url = 'http://' + url;
  return url;
}

// addServer(url) - manual add (from the Config page "Add" flow).
// addServer(url, discoveredFrom) - auto-add from a reviewed discovery
// scan (see discoverRegistriesFrom()/the Config page's "Scan for
// registries" action): discoveredFrom records which server's
// `.xregistry` response reported this URL, purely for traceability
// (never mutated afterward - see setDiscoveredFrom()).
// Blocks (returns false, no-ops entirely) if the URL is already
// configured — a URL is always a server's unique identity, so adding an
// already-known URL again does nothing; the caller must show an error
// telling the user to delete the existing entry first if they want to
// re-add it with different settings (e.g. a different proxy flag).
// Returns true if the server was actually added.
function addServer(url, discoveredFrom) {
  url = normalizeURL(url);
  if (!url) return false;
  var list = loadServers();
  if (list.includes(url)) return false;
  list.push(url);
  saveServers(list);
  if (discoveredFrom) setDiscoveredFrom(url, discoveredFrom);
  return true;
}

// Removes a server from LS_SERVERS AND clears every other per-URL flag
// map keyed to it (name override, proxy, hidden, discoveredFrom) —
// deleting a server is a full teardown, not just a list removal, so a
// later re-add/re-discovery of the same URL starts from clean defaults
// instead of silently inheriting stale state (e.g. a "hidden" flag from
// a server that was hidden, then deleted, then re-discovered much later
// under the same URL).
function removeServer(url) {
  var norm = normalizeURL(url);
  saveServers(loadServers().filter(function(u) { return u !== url; }));
  setNameOverride(norm, '');
  setProxied(norm, false);
  setHidden(norm, false);
  var discMap = loadDiscoveredFrom();
  delete discMap[norm];
  saveDiscoveredFrom(discMap);
}

// ---- Registry name overrides (persisted) -----------------------------------
//
// Lets a user give a registry a custom display name from the Config page,
// used everywhere the UI would otherwise show the server-reported name
// (registryid, or a spec `name` attribute if present) — the Registries
// list/grid, breadcrumbs, and the Registry root page's own header title.
// Purely client-side/cosmetic: never sent to the server, never affects the
// actual registry data. Keyed by normalizeURL() so it survives http(s)/
// trailing-slash variations the same way LS_SERVERS does.
function loadNameOverrides() {
  try { return JSON.parse(localStorage.getItem(LS_NAMES) || '{}'); }
  catch(e) { return {}; }
}
function saveNameOverrides(map) {
  try { localStorage.setItem(LS_NAMES, JSON.stringify(map)); } catch(e) {}
}
function getNameOverride(url) {
  var map = loadNameOverrides();
  return map[normalizeURL(url)] || '';
}
function setNameOverride(url, name) {
  var map = loadNameOverrides();
  var norm = normalizeURL(url);
  name = (name || '').trim();
  if (name) map[norm] = name; else delete map[norm];
  saveNameOverrides(map);
}

// ---- Proxy-server flags (persisted) ----------------------------------------
//
// Some remote registries don't set permissive CORS headers, so direct
// browser fetch() calls to them fail. Rather than try to auto-detect this
// (a CORS failure and a real network failure both throw the same generic
// fetch error, so it can't be told apart reliably), the user explicitly
// flags a server as "use proxy" when adding/editing it on the Config page.
// Keyed by normalizeURL(), same pattern as the name-overrides map above.
function loadProxyFlags() {
  try { return JSON.parse(localStorage.getItem(LS_PROXY) || '{}'); }
  catch(e) { return {}; }
}
function saveProxyFlags(map) {
  try { localStorage.setItem(LS_PROXY, JSON.stringify(map)); } catch(e) {}
}
function isProxied(url) {
  return !!loadProxyFlags()[normalizeURL(url)];
}
function setProxied(url, on) {
  var map  = loadProxyFlags();
  var norm = normalizeURL(url);
  if (on) map[norm] = true; else delete map[norm];
  saveProxyFlags(map);
}

// ---- Known-registries auto-discovery (persisted) --------------------------
//
// See plan.md "known-registries-autoadd-hide". Three independent per-server
// flags, each its own map (same keyed-by-normalizeURL() shape as the proxy
// flags above), rather than folding them into LS_SERVERS itself (which
// stays a plain array of URL strings) - this matches the existing
// convention of one small parallel map per piece of per-server metadata.

// discoveredFrom: which server's .xregistry response caused this URL to
// be added (or unset for a manually-added server). Purely informational/
// provenance - set once at creation time (see addServer()) and never
// mutated afterward, even if the same URL is later re-discovered from a
// different source (first-seen wins).
function loadDiscoveredFrom() {
  try { return JSON.parse(localStorage.getItem(LS_DISCOVERED) || '{}'); }
  catch(e) { return {}; }
}
function saveDiscoveredFrom(map) {
  try { localStorage.setItem(LS_DISCOVERED, JSON.stringify(map)); } catch(e) {}
}
function getDiscoveredFrom(url) {
  return loadDiscoveredFrom()[normalizeURL(url)] || '';
}
function setDiscoveredFrom(url, fromUrl) {
  var map  = loadDiscoveredFrom();
  var norm = normalizeURL(url);
  if (map[norm]) return; // first-seen wins - never overwrite
  if (fromUrl) map[norm] = fromUrl; else delete map[norm];
  saveDiscoveredFrom(map);
}

// hidden: excludes a server from the Home page listing without deleting
// it from config - so re-running discovery does not keep re-adding
// something the user chose to declutter, and its other settings (proxy,
// discoveredFrom) are preserved if unhidden later.
function loadHiddenFlags() {
  try { return JSON.parse(localStorage.getItem(LS_HIDDEN) || '{}'); }
  catch(e) { return {}; }
}
function saveHiddenFlags(map) {
  try { localStorage.setItem(LS_HIDDEN, JSON.stringify(map)); } catch(e) {}
}
function isHidden(url) {
  return !!loadHiddenFlags()[normalizeURL(url)];
}
function setHidden(url, on) {
  var map  = loadHiddenFlags();
  var norm = normalizeURL(url);
  if (on) map[norm] = true; else delete map[norm];
  saveHiddenFlags(map);
}

// Reverses the local xrproxy rewrite (see registry/xrproxy.go
// HTTPXRProxy()'s response-body ReplaceAll of the remote origin) for a
// single URL string, IF it's one of our own /xrproxy/<b64-origin>/...
// URLs. The proxy rewrites the remote origin wherever it appears in ANY
// response body it forwards (so embedded self/xid/*url links route back
// through it) — but this is semantically wrong for a `.xregistry`
// discovery document's `registries` list, whose entries are meant to be
// the REAL addresses of other, independent registries, not self-
// referential navigation links. Without this reversal,
// discoverRegistriesFrom() would report the proxy-rewritten URL as a
// sibling registry's "identity", silently hard-wiring it through this
// one local server's proxy forever - even though nothing
// flags it as proxied on the Config page (see plan.md's xrproxy writeup:
// identity URLs are always supposed to be the real remote address; only
// serverFetchBase() ever translates to a proxy URL, and only at
// fetch-time). Returns `url` unchanged if it isn't one of our own
// /xrproxy/ URLs (or doesn't decode to a plausible origin).
function decodeAnyXRProxyURL(url) {
  var prefix = normalizeURL(window.location.origin) + '/xrproxy/';
  if (!url || url.indexOf(prefix) !== 0) return url;
  var rest = url.slice(prefix.length);
  var slashIdx = rest.indexOf('/');
  var seg = slashIdx === -1 ? rest : rest.slice(0, slashIdx);
  var suffix = slashIdx === -1 ? '' : rest.slice(slashIdx);
  try {
    var pad = seg.replace(/-/g, '+').replace(/_/g, '/');
    pad += '='.repeat((4 - pad.length % 4) % 4);
    var decodedOrigin = atob(pad);
    if (!/^https?:\/\//i.test(decodedOrigin)) return url;
    return decodedOrigin.replace(/\/$/, '') + suffix;
  } catch (e) { return url; }
}

// Queries `.xregistry` (see registry/httpStuff.go
// HTTPGETXRegistryDiscovery()) on each given source URL for sibling
// registries it knows about, and reports back a categorized list —
// does NOT mutate storage itself (see the Config page's "Scan for
// registries" bulk action / cfgScanSelected(), which shows the results
// in a review dialog and only calls addServer() for entries the user
// explicitly confirms). A single unreachable/non-implementing source
// (404, network error, etc) is silently skipped — one bad host must not
// block discovery via the others. Results are deduped across sources
// (first-seen source wins for discoveredFrom, matching the existing
// "first seen wins" convention in setDiscoveredFrom()). Excludes the
// local origin itself (always shown on Home separately). Calls
// cb(results) once every source has settled (fetch failures included),
// never throws — results is an array of
// {url, discoveredFrom, alreadyKnown}.
function discoverRegistriesFrom(sourceUrls, cb) {
  var origin = normalizeURL(window.location.origin);
  var known  = loadServers();
  var seen   = {};   // normalizedURL -> true, for cross-source dedup
  var results = [];
  var remaining = sourceUrls.length;
  if (!remaining) { if (cb) cb([]); return; }

  function done() {
    remaining--;
    if (remaining <= 0 && cb) cb(results);
  }

  sourceUrls.forEach(function(sourceUrl) {
    fetch(serverFetchBase(sourceUrl) + '/.xregistry')
      .then(function(r) { if (!r.ok) throw new Error('HTTP ' + r.status); return r.json(); })
      .then(function(d) {
        var regs = (d && d.registries) || [];
        regs.forEach(function(url) {
          url = normalizeURL(decodeAnyXRProxyURL(url || ''));
          if (!url || url === origin || seen[url]) return;
          seen[url] = true;
          results.push({
            url: url,
            discoveredFrom: sourceUrl,
            alreadyKnown: known.includes(url)
          });
        });
        done();
      })
      .catch(function() { done(); }); // server doesn't implement .xregistry, or unreachable — skip
  });
}


// base64url-encode (no padding) a string for use as a single /xrproxy/ path
// segment - mirrors the encoding the Go server (EncodeRProxyOrigin in
// registry/xrproxy.go) expects/produces.
function b64urlEncode(str) {
  return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

// Returns the URL to actually use when building fetch() requests against
// `url` - the raw url itself, unless the user flagged it as "use proxy" on
// the Config page, in which case this returns the local /xrproxy/ URL that
// transparently forwards to it (see registry/xrproxy.go). Everywhere else
// in the app (labels, breadcrumbs, Config page, LS_SERVERS, cache keys)
// keeps using the raw server URL as its identity - only the actual network
// request needs to go through the proxy.
function serverFetchBase(url) {
  var norm = normalizeURL(url || window.location.origin);
  if (isProxied(norm)) {
    var origin = window.location.origin.replace(/\/$/, '');
    return origin + '/xrproxy/' + b64urlEncode(norm);
  }
  return norm;
}

// Inverse of serverFetchBase(): given a URL that may be prefixed with our
// local /xrproxy/<base64> proxy path (for the CURRENT server, _state.
// serverURL), returns the real remote URL instead — for DISPLAY PURPOSES
// ONLY (tooltips, copy-to-clipboard, JSON view text). Never use this for
// anything that actually fetches or navigates — only the proxy URL works
// from the browser (that's the entire reason the proxy exists: the real
// remote origin is typically blocked by CORS). Returns `url` unchanged if
// it isn't proxied (not proxied at all, already a real absolute URL, or a
// relative path).
function toRealURL(url) {
  if (!url) return url;
  var proxyBase = serverFetchBase(_state.serverURL).replace(/\/$/, '');
  if (url.indexOf(proxyBase) !== 0) return url;
  var real = normalizeURL(_state.serverURL || window.location.origin).replace(/\/$/, '');
  return real + url.slice(proxyBase.length);
}

// ---- Init -----------------------------------------------------------------

window.addEventListener('DOMContentLoaded', init);
window.addEventListener('popstate', function() { loadStateFromURL(); renderHeader(); refresh(); });
window.addEventListener('resize', function() { renderHeader(); sizeDocTextarea(); sizeJsonEditTextarea(); });

// Scope Ctrl/Cmd+A to just the JSON output when focus is inside it, instead
// of selecting the entire page (mirrors the old ui.go `dokeydown()` trick).
document.addEventListener('keydown', function(e) {
  if (e.key !== 'a' && e.key !== 'A') return;
  if (!e.ctrlKey && !e.metaKey) return;
  var tag = e.target && e.target.tagName;
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;
  var out = el('json-output');
  if (!out || !out.contains(e.target)) return;
  e.preventDefault();
  var range = document.createRange();
  range.selectNodeContents(out);
  var sel = window.getSelection();
  sel.removeAllRanges();
  sel.addRange(range);
});

function init() {
  _state.homeGroup   = optHomeGroup();
  _state.homeLayouts = optHomeLayouts();
  applyJsonColorMode();
  loadStateFromURL();

  // view-toggle-btn / editing-indicator now use inline onclick handlers
  // directly (single elements, not delegated data-dview buttons anymore).

  // Left-panel drag-resize
  var resizer = el('left-panel-resizer');
  var lpanel  = el('left-panel');
  if (resizer && lpanel) {
    if (_opts.leftPanelWidth) lpanel.style.width = _opts.leftPanelWidth + 'px';
    resizer.addEventListener('mousedown', function(e) {
      e.preventDefault();
      var startX = e.clientX, startW = lpanel.offsetWidth;
      resizer.classList.add('dragging');
      function onMove(e) {
        var w = Math.max(140, Math.min(700, startW + e.clientX - startX));
        lpanel.style.width = w + 'px';
      }
      function onUp() {
        resizer.classList.remove('dragging');
        _opts.leftPanelWidth = lpanel.offsetWidth;
        saveOpts();
        document.removeEventListener('mousemove', onMove);
        document.removeEventListener('mouseup',   onUp);
      }
      document.addEventListener('mousemove', onMove);
      document.addEventListener('mouseup',   onUp);
    });
  }

  renderHeader();
  refresh();
}

// Show or hide the left panel and its resizer handle together.
function setLeftPanelVisible(show) {
  var d = show ? '' : 'none';
  var lp = el('left-panel'),  lr = el('left-panel-resizer');
  if (lp) lp.style.display = d;
  if (lr) lr.style.display = d;
}

// ---- URL state -----------------------------------------------------------

function loadStateFromURL() {
  var p = new URLSearchParams(window.location.search);
  _state.view        = p.get('view')    || 'home';
  _state.serverURL   = p.get('server')  || '';
  _state.section     = p.get('section') || 'data';
  _state.path        = decodePath(p.get('path') || '');
  // Use explicit dview param if present; otherwise restore saved preference for this
  // section/depth (data pages restore per-depth, model/modelsource restore per-section)
  _state.dataView    = p.get('dview') || defaultDataView(_state.section, _state.path.length, _state.path);
  _state.editMode    = p.get('edit') === '1';
  // Session-level "Show/Hide xReg Data" override (see effectiveXregFocused()) —
  // persists across navigation and refresh, same as editMode, until the user
  // explicitly turns it off via the kebab menu or the pinned toolbar icon.
  _state.xregOverride = p.get('xrv') === '1';
  var jsonOpts       = parseJSONOptionsFromQuery(p.toString());
  _state.inlines     = jsonOpts.inlines;
  _state.filters     = (p.get('filter') || '').split('\n').filter(Boolean);
  // Sort is only valid on collection-shaped paths (see pageHref()'s note) —
  // the server 400s if asked to sort a non-collection endpoint. Guard here
  // too so a stale/hand-edited/bookmarked URL degrades gracefully (silently
  // drops the invalid param) instead of producing a server error on load.
  _state.sort        = isCollection(_state.path) ? jsonOpts.sort : '';
  _state.docView     = jsonOpts.docView;
  _state.binary      = jsonOpts.binary;
  _state.collections = jsonOpts.collections;
  _state.useExport   = p.get('export')      === '1';
  // apiurl= is the real, server-provided absolute URL that produced the current
  // page (see "Link-driven navigation" notes) — reused verbatim on refresh/
  // bookmark instead of reconstructing. crumbURLs (per-depth cache of real
  // ancestor URLs) can't survive a reload — nothing in a fetched response hands
  // us its own ancestors' URLs — so it always starts empty; breadcrumb clicks to
  // an un-cached ancestor fall back to plain construction (accepted trade-off).
  _state.apiURL      = p.get('apiurl') || '';
  _state.crumbURLs   = [];
  // Same trade-off as crumbURLs (can't survive a reload) — but if we're
  // landing directly on the registry root with a filter/sort/etc already
  // applied (e.g. a bookmark), synthesize the equivalent real root URL so a
  // same-session breadcrumb round-trip (drill down, then click back to the
  // root) restores it correctly. See buildBreadcrumbSegments()/
  // pushStateReal() for how this is kept in sync afterward.
  _state.rootApiURL  = (_state.section === 'data' && _state.path.length === 0)
    ? buildAPIURL() : '';
  // crumbOpts/rootOpts — same trade-off/pattern as crumbURLs/rootApiURL just
  // above: ancestor snapshots can't survive a reload, so crumbOpts always
  // starts empty; but the CURRENT (root) page's own just-loaded query string
  // already IS the snapshot rootOpts would hold, so seed it directly rather
  // than leaving it blank until the next navigation.
  _state.crumbOpts   = [];
  _state.rootOpts    = (_state.section === 'data' && _state.path.length === 0)
    ? p.toString() : '';
  _state.docTab      = p.get('tab') || '';
  _state.resVersion  = p.get('ver') || '';
  // List view's "Filters" toggle panel (isGridFiltersOnlyMode()) — persist
  // its open/closed state across a refresh, same as the Filters/Sort
  // values it displays. Restored here as a plain global (not part of
  // _state) since that's how the rest of the app already tracks it (see
  // toggleFiltersPanel()); every data-page render already re-checks
  // isGridFiltersOnlyMode() to decide whether to show the panel, so simply
  // restoring this flag is enough to bring the panel back on load.
  _filtersPanelOpen  = p.get('panel') === '1';
  // Retire the dedicated Version page for anything but JSON view — see
  // normalizeVersionDepth()'s doc comment. Covers direct/bookmarked loads
  // of an old `.../versions/<vid>` URL with no (or a non-json) `dview=`.
  normalizeVersionDepth(_state);
  syncFiltersFromApiURL();
}

function buildURL(st) {
  var p = new URLSearchParams();
  if (st.view && st.view !== 'home')       p.set('view',    st.view);
  // 'server' is meaningless on the home page (it lists all registries, not
  // one) — guard against a stale leftover _state.serverURL leaking into the
  // address bar even if some call site forgets to clear it explicitly.
  if (st.serverURL && st.view !== 'home')  p.set('server',  st.serverURL);
  if (st.section && st.section !== 'data') p.set('section', st.section);
  if (st.editMode)                         p.set('edit', '1');
  if (st.xregOverride)                     p.set('xrv', '1');
  if (st.path   && st.path.length)         p.set('path',    encodePath(st.path));
  // Real server-provided URL for the current page (data section, non-root only —
  // the registry root's own URL is always trivially serverBase() itself).
  if (st.section === 'data' && st.apiURL && st.path && st.path.length) {
    p.set('apiurl', st.apiURL);
  }
  var defaultView = defaultDataView(st.section, (st.path || []).length, st.path);
  if (st.dataView && st.dataView !== defaultView) p.set('dview', st.dataView);

  // Resource/Version page tab + version-selector persistence — only
  // meaningful on those pages (depth 4 = Resource, depth 6+ = Version),
  // in the data section, and only when non-default (keeps URLs clean
  // everywhere else). See plan.md "Remember selected version + active tab".
  var pathLenU = (st.path || []).length;
  if (st.section === 'data' && (pathLenU === 4 || pathLenU >= 6)) {
    if (st.docTab)                       p.set('tab', st.docTab);
    if (pathLenU === 4 && st.resVersion) p.set('ver', st.resVersion);
  }


  // JSON-view-only params: only add when actually in JSON view. Model/modelsource's
  // "list" (editor) dataView is not a JSON view, so it's excluded here.
  // New path-specific query params should be added here; they will naturally
  // be absent from the URL in all non-JSON-view contexts.
  var isJsonOnlySection = st.section === 'capabilities' || st.section === 'capabilitiesoffered'
    || st.section === 'xregistry';
  var inJsonView = st.dataView === 'json' || st.view === 'json' || isJsonOnlySection;
  // Filters and Sort are supported by List view too now (unlike inline/doc/
  // binary/collections, which remain JSON-view-only for now — see plan.md
  // "Filter support in Grid/List views" and "List view Sort picker") — so
  // persist them regardless of view, otherwise applying either in List
  // silently vanishes on refresh (loadStateFromURL() has nothing to read
  // it back from).
  //
  // But skip the top-level `filter=` param when `apiurl=` (just above) is
  // itself already carrying the exact same filters embedded in its own
  // query string — that's the case whenever `st.apiURL` came from
  // applyFilters() (current collection page, post-Apply) or
  // entityHrefWithFilter() (an entity row's own link, rescoped to match
  // the collection's active filter): writing both would show `filter=`
  // twice in the address bar/hover URL for no benefit, since
  // syncFiltersFromApiURL() already re-derives `_state.filters` straight
  // from `apiurl=` on load. Only fall back to the explicit top-level param
  // when `apiurl=` isn't set or doesn't already match (e.g. links to pages
  // with no cached real URL yet) — it remains the sole source there.
  var apiURLFilters = (st.section === 'data' && st.apiURL && st.path && st.path.length)
    ? filtersFromUrl(st.apiURL) : null;
  var filterAlreadyInApiURL = apiURLFilters && sameStringSet(apiURLFilters, st.filters || []);
  if (st.filters && st.filters.length && !filterAlreadyInApiURL) {
    p.set('filter', st.filters.join('\n'));
  }
  if (st.sort)                         p.set('sort',   st.sort);
  // Persist List view's Filters/Sort panel open/closed state too (same
  // reasoning as filter/sort above — otherwise the panel a user left open
  // silently closes on refresh, even though navigating the hierarchy
  // normally keeps it open). Only meaningful outside JSON view, which
  // always shows its own full left panel regardless of this flag.
  if (_filtersPanelOpen && !inJsonView) p.set('panel', '1');
  if (inJsonView) {
    if (st.inlines && st.inlines.length)   p.set('inline',      st.inlines.join(','));
    if (st.docView)                        p.set('doc',         '1');
    if (st.binary)                         p.set('binary',      '1');
    if (st.collections && collectionsEligible(st.path))  p.set('collections', '1');
    if (st.useExport && st.path && st.path.length === 0) p.set('export', '1');
  }

  var qs = p.toString();
  return window.location.pathname + (qs ? '?' + qs : '');
}

function pushState(patch) {
  // Guard: leaving modelsource edit mode (or navigating away from it entirely)
  // with unsaved changes — offer Save / Discard / Cancel before proceeding.
  if (_state.section === 'modelsource' && _state.editMode && _modelDirty) {
    var leavingEdit = patch.editMode === false || patch.section !== undefined
      || patch.path !== undefined || patch.serverURL !== undefined || patch.view !== undefined;
    if (leavingEdit) {
      showLeaveEditDialog(
        function() { saveModel(function() { _modelDirty = false; pushStateReal(patch); }); },
        function() { _modelDirty = false; _modelData = deepClone(_modelSrc); pushStateReal(patch); }
      );
      return;
    }
  }
  // Guard: leaving capabilities edit mode with unsaved changes — same pattern.
  if (_state.section === 'capabilities' && _state.editMode && _capDirty) {
    var leavingCapEdit = patch.editMode === false || patch.section !== undefined
      || patch.path !== undefined || patch.serverURL !== undefined || patch.view !== undefined;
    if (leavingCapEdit) {
      showLeaveEditDialog(
        function() { saveCapabilities(function() { _capDirty = false; pushStateReal(patch); }); },
        function() { _capDirty = false; _capData = deepClone(_capSrc); pushStateReal(patch); }
      );
      return;
    }
  }
  // Guard: leaving a collection page with an open, non-empty inline
  // "Add new <Type>" form (Group/Resource/Version create) — same
  // unsaved-changes pattern as the entity/meta/version/modelsource/
  // capabilities guards above, but for the Add form's own separate
  // _addNewOpen/_addNewData state (see isAddNewFormDirty()). Only a
  // single "Create" save action applies here (always a PUT-by-id, no
  // PUT-vs-PATCH choice), so no 4th onSaveDelta argument is passed.
  if (_state.section === 'data' && isCollection(_state.path) && isAddNewFormDirty()) {
    var leavingAddNew = patch.section !== undefined || patch.path !== undefined
      || patch.serverURL !== undefined || patch.view !== undefined
      || (patch.dataView !== undefined && patch.dataView !== _state.dataView);
    if (leavingAddNew) {
      showLeaveEditDialog(
        function() { saveNewEntity(function() { pushStateReal(patch); }); },
        function() { cancelAddEntity(); pushStateReal(patch); }
      );
      return;
    }
  }
  // Guard: leaving an in-progress entity data edit (List view) with unsaved
  // changes — same pattern as modelsource/capabilities above.
  if (_state.section === 'data' && _state.editMode && _dataDirty) {
    var leavingDataEdit = patch.editMode === false || patch.section !== undefined
      || patch.path !== undefined || patch.serverURL !== undefined || patch.view !== undefined;
    if (leavingDataEdit) {
      showLeaveEditDialog(
        function() { saveDataEntity('PUT', function() { pushStateReal(patch); }); },
        function() { _dataDirty = false; _dataEditData = deepClone(_dataEditSrc); pushStateReal(patch); },
        null,
        function() { saveDataEntity('PATCH', function() { pushStateReal(patch); }); }
      );
      return;
    }
  }
  // Guard: leaving an in-progress Meta-tab edit with unsaved changes — same
  // pattern as the entity-data guard above (the Meta tab has its own
  // separate working-copy pair, see _metaEditSrc/_metaEditData).
  if (_state.section === 'data' && _state.editMode && _metaDirty) {
    var leavingMetaEdit = patch.editMode === false || patch.section !== undefined
      || patch.path !== undefined || patch.serverURL !== undefined || patch.view !== undefined;
    if (leavingMetaEdit) {
      showLeaveEditDialog(
        function() { saveMetaEntity('PUT', function() { pushStateReal(patch); }); },
        function() { _metaDirty = false; _metaEditData = deepClone(_metaEditSrc); pushStateReal(patch); },
        null,
        function() { saveMetaEntity('PATCH', function() { pushStateReal(patch); }); }
      );
      return;
    }
  }
  // Guard: leaving an in-progress edit of a non-default version (picked via
  // the Resource page's "Version:" dropdown) with unsaved changes — same
  // pattern as the entity-data/Meta guards above (see _verEditSrc/_verEditData).
  if (_state.section === 'data' && _state.editMode && _verDirty) {
    var leavingVerEdit = patch.editMode === false || patch.section !== undefined
      || patch.path !== undefined || patch.serverURL !== undefined || patch.view !== undefined;
    if (leavingVerEdit) {
      showLeaveEditDialog(
        function() { saveVersionEntity('PUT', function() { pushStateReal(patch); }); },
        function() { _verDirty = false; _verEditData = deepClone(_verEditSrc); pushStateReal(patch); },
        null,
        function() { saveVersionEntity('PATCH', function() { pushStateReal(patch); }); }
      );
      return;
    }
  }
  // Guard: leaving JSON view's raw-textarea edit ("postman-style", see
  // renderJSONEditView()) with unsaved changes, or turning edit off while
  // staying on the same page — same pattern as the guards above. This one
  // applies regardless of `_state.section`, since the raw JSON editor works
  // the same way for data/capabilities/modelsource alike.
  if (_state.dataView === 'json' && _state.editMode && _jsonEditDirty) {
    var leavingJsonEdit = patch.editMode === false || patch.section !== undefined
      || patch.path !== undefined || patch.serverURL !== undefined || patch.view !== undefined;
    if (leavingJsonEdit) {
      showLeaveEditDialog(
        function() { jsonEditSave('PUT', function() { resetJsonEditBuffer(); pushStateReal(patch); }); },
        function() { resetJsonEditBuffer(); pushStateReal(patch); },
        null,
        function() { jsonEditSave('PATCH', function() { resetJsonEditBuffer(); pushStateReal(patch); }); },
        true
      );
      return;
    }
  }
  // Whenever we're leaving JSON-edit mode entirely (editMode turned off, or
  // navigating away) with NO unsaved changes, still reset the buffer so the
  // next time Edit is turned on in JSON view it starts fresh from the
  // server (see resetJsonEditBuffer()'s doc comment / the "Independent
  // buffer" design decision in plan.md).
  if (_state.dataView === 'json' && _state.editMode && _jsonEditOrigText !== null) {
    var leavingJsonClean = patch.editMode === false || patch.section !== undefined
      || patch.path !== undefined || patch.serverURL !== undefined || patch.view !== undefined;
    if (leavingJsonClean) resetJsonEditBuffer();
  }
  pushStateReal(patch);
}

function pushStateReal(patch) {
  // Restore depth/section-specific view preference whenever we enter (or re-enter) a
  // data/model/modelsource page: on server change, path change, section change, or
  // transitioning from home/config to data.
  var changingServer  = patch.serverURL !== undefined && patch.serverURL !== _state.serverURL;
  var changingPath    = patch.path !== undefined &&
                       JSON.stringify(patch.path) !== JSON.stringify(_state.path);
  var changingSection = patch.section !== undefined && patch.section !== _state.section;
  var enteringData    = patch.view !== undefined &&
                       patch.view !== 'home' && patch.view !== 'config' &&
                       (_state.view === 'home' || _state.view === 'config');
  if (changingServer || changingPath || changingSection || enteringData) {
    var newPath    = patch.path !== undefined ? patch.path : _state.path || [];
    var newDepth   = newPath.length;
    var newSection = patch.section !== undefined ? patch.section : _state.section;
    var savedView  = defaultDataView(newSection, newDepth, newPath);
    // JSON view is "sticky" across navigation — moving up/down within a section's
    // pages, or switching between Registry Data / Model / Model Source / Capabilities
    // / Capabilities Offered (e.g. via the Registry Endpoints panel or the
    // "← Registry Data" link) — all keep JSON view, just like clicking a URL inside
    // the JSON content itself always stays in JSON. Only breaks when changing
    // servers or freshly entering the data section from Home/Config, where the
    // section/depth default should apply instead.
    if (!changingServer && !enteringData && (changingPath || changingSection)
        && _state.dataView === 'json' && patch.dataView === undefined) {
      savedView = 'json';
    }

    // Invalidate the Filters/Sort picker drafts on every fresh navigation
    // (as opposed to in-page actions like switching List<->JSON view or
    // opening/closing the panel, which should NOT lose an in-progress,
    // not-yet-applied edit). Their lazy "rebuild only if the key changed"
    // checks (ensureFbDraft()/ensureSortDraft()) aren't enough on their
    // own: those functions only run while their section actually renders,
    // and Sort only renders on collection pages — so a round trip through
    // a non-collection page (e.g. Collection A -> registry root -> back to
    // Collection A) never invalidates the key, leaving a stale draft with
    // the old attribute still selected even though _state.sort was
    // correctly reset to '' below. Filters happens to self-heal today only
    // because Filters also renders at the root, but that's incidental —
    // reset both explicitly here so it doesn't depend on which sections
    // happen to render along the way. See plan.md "Stale Sort picker
    // after breadcrumb round-trip".
    _sortDraft = null; _sortDraftKey = null;
    _fbDraft = null;   _fbDraftKey = null;

    // Always land back at the top of the Model/Model Source editor's own
    // drill-down tree when (re-)entering that section from anywhere else
    // (or switching servers) — otherwise renderModelEditor()'s
    // _modelLoadedFor cache-key check (serverURL|section, unchanged by a
    // mere round trip through some other section and back) would silently
    // keep the drilled-down _navPath/_navTab/_navSelected from a previous
    // visit. A plain path/view toggle *within* an already-active
    // model/modelsource page (e.g. switching List<->JSON view) must NOT
    // reset this — only a genuine section/server change should.
    if ((changingServer || changingSection) && (newSection === 'model' || newSection === 'modelsource')) {
      _navTab = 'registry'; _navPath = []; _navSelected = null; _attrNestStack = [];
    }

    // Link-driven navigation bookkeeping: crumbURLs caches the real URL used at
    // each visited depth this session. Server/section changes (or freshly
    // entering data) invalidate the whole cache; a plain path change just trims
    // any now-stale deeper entries (shallower ancestors already visited stay
    // valid). If the caller didn't hand us a real link (patch.apiURL), default
    // to whatever this depth's cached ancestor URL is (covers e.g. redirecting
    // up to an already-visited entity) — falling back to '' (→ buildBaseURL()
    // construction) only when truly nothing is known for this depth.
    if (changingServer || changingSection || enteringData) {
      _state.crumbURLs = [];
      _state.rootApiURL = '';
      _state.crumbOpts = [];
      _state.rootOpts = '';
    } else if (changingPath) {
      _state.crumbURLs = (_state.crumbURLs || []).slice(0, newDepth);
      _state.crumbOpts = (_state.crumbOpts || []).slice(0, newDepth);
    }
    // rootApiURL is crumbURLs' depth-0 counterpart (crumbURLs is only
    // indexed for depths > 0 — see its own declaration comment).
    var defaultApiURL = (newSection !== 'data') ? ''
      : (newDepth > 0 ? (_state.crumbURLs[newDepth - 1] || '')
                      : (_state.rootApiURL || ''));

    // Entity-instance depths (Group/Resource/Version instances, i.e. even
    // newDepth) never get their own crumbURLs entry when reached purely by
    // following nested <plural>url collection links (the normal drill-down
    // pattern) — that navigation jumps straight from one collection to the
    // next, never landing on the entity's own page in between. When that's
    // the case (defaultApiURL is unknown here), fall back to the immediate
    // parent COLLECTION's cached filter, UNCHANGED/un-relativized: crossing
    // from a collection into one of its member entities never trims a
    // filter clause (only entering a NESTED collection does, per the
    // server's FiltersRelativeToAbstract()) — so an entity's own filter
    // context is always identical to its parent collection's. Leaving
    // apiURL itself at '' is fine/intentional: buildAPIURL()'s existing
    // no-real-link fallback already re-appends _state.filters onto the
    // plain path-constructed URL, and refresh()'s needsDetails() handles
    // any required "$details" suffix independently at fetch time.
    var defaultFilters = [];
    if (newSection === 'data' && !defaultApiURL && newDepth > 0 && newDepth % 2 === 0) {
      var parentCollURL = _state.crumbURLs[newDepth - 2];
      if (parentCollURL) defaultFilters = filtersFromUrl(parentCollURL);
    }

    // Likewise restore whatever JSON-view option flags (inline/sort/
    // docView/binary/collections) were actually last applied AT this
    // destination depth this session, from the same per-depth snapshot
    // pageHref() already reads (crumbOpts/rootOpts — see their declaration
    // and pushStateReal()'s own snapshot logic further below). Without
    // this, clicking an ancestor breadcrumb whose HOVER preview correctly
    // showed e.g. "?inline=file" would still silently land on a page with
    // inline cleared — the click patch here never mentions inlines at all,
    // so only this "fresh navigation defaults" merge below actually
    // decides what the destination page ends up with. A depth with no
    // cached snapshot yet (a genuinely fresh drill-down, never visited
    // this session) naturally gets all-blank/false defaults from
    // parseJSONOptionsFromQuery(''), identical to the previous hardcoded
    // literals — so this one path now correctly covers both "returning to
    // an already-visited page" (restore) and "drilling into a brand new
    // one" (blank).
    var defaultOptsQS = (newSection !== 'data') ? ''
      : (newDepth > 0 ? ((_state.crumbOpts && _state.crumbOpts[newDepth - 1]) || '')
                      : (_state.rootOpts || ''));
    var defaultJsonOpts = parseJSONOptionsFromQuery(defaultOptsQS);
    if (!isCollection(newPath)) defaultJsonOpts.sort = '';

    // Prepend defaults so explicit values in patch still win
    patch = Object.assign({
      inlines: defaultJsonOpts.inlines, filters: defaultFilters, sort: defaultJsonOpts.sort,
      docView: defaultJsonOpts.docView, binary: defaultJsonOpts.binary,
      collections: defaultJsonOpts.collections,
      useExport: false, section: 'data', dataView: savedView, apiURL: defaultApiURL,
      // A real navigation (not a tab-click/version-select, which sync the URL
      // directly via history.replaceState and never reach pushStateReal) means
      // a different resource/page — don't carry over the previous page's tab/
      // version preference. See plan.md "Remember selected version + active tab".
      docTab: '', resVersion: ''
    }, patch);

    if (patch.apiURL && newSection === 'data' && newDepth > 0) {
      _state.crumbURLs[newDepth - 1] = patch.apiURL;
    } else if (patch.apiURL && newSection === 'data' && newDepth === 0) {
      _state.rootApiURL = patch.apiURL;
    }
  }
  Object.assign(_state, patch);

  // Retire the dedicated Version page for anything but JSON view — see
  // normalizeVersionDepth()'s doc comment. Runs AFTER dataView is fully
  // resolved (including the sticky-JSON-view logic above), so a version
  // link clicked from inside JSON view (dataView already 'json' by the
  // time we get here) is correctly left alone, while a plain List-view row
  // click into `.../versions/<vid>` gets rewritten to the Resource page +
  // resVersion before anything renders.
  normalizeVersionDepth(_state);

  syncFiltersFromApiURL();

  // applyFilters()/similar callers may set a new apiURL for the CURRENT
  // depth without a path change (so the block above never ran) — keep
  // crumbURLs (or rootApiURL, at depth 0) in sync here too, so a later
  // breadcrumb click back to this depth doesn't regress to the pre-Apply
  // link.
  if (patch.apiURL !== undefined && _state.section === 'data' && _state.path.length > 0) {
    _state.crumbURLs = _state.crumbURLs || [];
    _state.crumbURLs[_state.path.length - 1] = patch.apiURL;
  } else if (patch.apiURL !== undefined && _state.section === 'data' && _state.path.length === 0) {
    _state.rootApiURL = patch.apiURL;
  }

  // Snapshot this depth's current JSON-view option flags (inline/sort/
  // docView/binary/collections) as the exact query string buildURL() would
  // produce for this page right now — i.e. the same string a browser
  // refresh/bookmark would already need in order to restore this page
  // correctly. Runs unconditionally on every navigation/in-page option
  // change (not just when apiURL itself changed), so pageHref() can later
  // restore an ancestor's own real last-applied options (see crumbOpts/
  // rootOpts's declaration and parseJSONOptionsFromQuery()) instead of
  // leaking the CURRENT (possibly much deeper) page's values onto every
  // other breadcrumb/tile/row href.
  if (_state.section === 'data') {
    var optsQS = buildURL(_state).split('?')[1] || '';
    if (_state.path.length > 0) {
      _state.crumbOpts = _state.crumbOpts || [];
      _state.crumbOpts[_state.path.length - 1] = optsQS;
    } else {
      _state.rootOpts = optsQS;
    }
  }

  history.pushState(null, '', buildURL(_state));
  renderHeader();
  refresh();
}

// Default dataView for a given section/path-depth, honoring saved preferences.
//   data                — per-depth preference (_opts.depthViews), default 'grid'
//   model / modelsource / capabilities / capabilitiesoffered — per-section
//                         preference (_opts.sectionViews), default 'table' (list)
function defaultDataView(section, pathLen, path) {
  if (section === 'model' || section === 'modelsource' || section === 'capabilities'
      || section === 'capabilitiesoffered' || section === 'xregistry') {
    return (_opts.sectionViews || {})[section] || 'table';
  }
  // Grid view has been removed entirely for the data section — Registry
  // root, Group/Resource/Version entities, and the Groups/Resources/
  // Versions collections are all List/JSON only now (List is strictly
  // more capable everywhere Grid used to appear — sorting, a Document
  // column on Versions, etc. — see plan.md "Grid view removed"). Always
  // default to List; a previously-saved per-depth Grid preference (from
  // before this removal) is intentionally ignored.
  if (section === 'data') return 'table';
  return (_opts.depthViews || {})[pathLen] || 'grid';
}

// True when `path` points at an individual Version entity (depth >= 6,
// path[4] === 'versions'). The dedicated List-view Version page has been
// retired (see plan.md "Retire the List-view dedicated Version page") — a
// path like this is only ever meant to be rendered by JSON view now;
// anywhere else it must be normalized to the owning Resource page (depth 4)
// + resVersion, via normalizeVersionDepth() below.
function isVersionEntityPath(path) {
  return !!path && path.length >= 6 && path[4] === 'versions';
}

// Central invariant enforcement for the retired dedicated Version page: a
// depth >= 6 ("data" section) path is only ever valid when dataView is
// 'json' — JSON view is the only view left that still addresses a version
// by its own literal URL (needed for link-driven navigation inside JSON
// content, and because it has no version-selector dropdown of its own).
// Any other dataView (List view, the only other data-section view) gets
// silently rewritten here to the Resource page (depth 4) + resVersion,
// landing on that page's normal default tab. Called from the two places
// state is fully resolved just before rendering: loadStateFromURL() (direct/
// bookmarked loads) and the tail of pushStateReal() (all real navigation,
// including the Versions List row click and the "Default Version" pill,
// which intentionally still navigate straight to the depth-6 path — this
// function is what actually retires the page, not the callers). Returns
// true if `state` was rewritten.
function normalizeVersionDepth(state) {
  if (state.section !== 'data' || state.dataView === 'json'
      || !isVersionEntityPath(state.path)) {
    return false;
  }
  state.resVersion = state.path[5];
  state.path = state.path.slice(0, 4);
  // Any apiURL carried in (a real link) pointed at the VERSION's own
  // endpoint — leaving it as-is would make buildAPIURL() fetch the wrong
  // entity now that path points at the Resource instead. Derive the
  // Resource's own real URL by stripping the trailing "/versions/<vid>"
  // segment pair (mirrors resolveResourceMetaUrl()'s same technique);
  // falls back to '' (buildBaseURL() reconstructs it from the now-
  // truncated path) if there was no apiURL to begin with.
  if (state.apiURL) {
    var qIdx = state.apiURL.indexOf('?');
    var base = qIdx >= 0 ? state.apiURL.slice(0, qIdx) : state.apiURL;
    var qs   = qIdx >= 0 ? state.apiURL.slice(qIdx) : '';
    state.apiURL = base.replace(/\/versions\/[^/?]+\/?$/, '') + qs;
  }
  return true;
}

function encodePath(parts) { return parts.map(encodeURIComponent).join('/'); }
function decodePath(str)   { return str ? str.split('/').map(decodeURIComponent).filter(Boolean) : []; }
function csvList(s)        { return s ? s.split(',').filter(Boolean) : []; }

// Parses the JSON-view option flags (inline/sort/docView/binary/collections)
// out of a query string. Shared by loadStateFromURL() (the browser address
// bar on a real page load/refresh) and pageHref() (restoring an ancestor's
// own last-applied options from its cached per-depth snapshot — see
// crumbOpts/rootOpts in pushStateReal()) so both read these fields
// identically instead of keeping two separate lists of fields in sync.
// Does NOT itself apply the "sort is only valid on a collection path" guard
// — every caller already applies that separately against its own
// destination path (see loadStateFromURL()/pageHref()).
function parseJSONOptionsFromQuery(qs) {
  var p = new URLSearchParams(qs || '');
  return {
    inlines:     csvList(p.get('inline')),
    sort:        p.get('sort') || '',
    docView:     p.get('doc')         === '1',
    binary:      p.get('binary')      === '1',
    collections: p.get('collections') === '1'
  };
}

// ---- Entity type from path -----------------------------------------------
//
// Returns true when the URL at this path points to a *collection* of entities.
// Everything else is a single entity.

function isCollection(path) {
  if (!path || path.length === 0) return false;   // registry root = single entity
  var last = path[path.length - 1];
  if (last === 'versions') return true;           // versions collection
  if (last === 'meta')     return false;          // meta sub-entity = single
  return (path.length % 2 === 1);                 // odd depth = collection
}

// Per spec, the "collections" flag (inline nested Group-type/Resource-type
// collections in the response) is only meaningful on the Registry root
// (path.length === 0) or on a Group instance (path.length === 2) — every
// other entity type (a collection itself, a Resource, a Version, etc.) has
// no nested `<plural>` collections of its own for it to affect.
function collectionsEligible(path) {
  var len = (path || []).length;
  return len === 0 || len === 2;
}

// Should we append $details to force JSON metadata view?
// True for resource (depth 4) and version (depth 6) entities.
// Safe to always do: spec says $details on non-doc resources = same as absent.
function needsDetails(path) {
  if (!path || _state.section !== 'data') return false;
  var len = path.length;
  if (len === 4) return true;  // resource entity
  if (len === 6 && path[4] === 'versions') return true;  // version entity
  return false;
}

// ---- API URL builder -------------------------------------------------------
//
// Link-driven navigation: the base URL for the current data-section page is
// the real, server-provided URL that was followed to get here (_state.apiURL,
// set by navigateTo()/navigateToNestedColl()/version navigators/breadcrumb
// clicks — see pushStateReal()), NOT reconstructed from `path` + serverBase().
// buildBaseURL() is only the fallback used when no real link is known yet for
// this depth (first-ever load of a server/registry with no prior navigation,
// or a legacy bookmark without an apiurl= param). Client-added query params
// (inline/filter/sort/etc., JSON view only) are always fine to append — that's
// the client expressing its own intent, not re-deriving something the server
// already computed.
//
// /model, /modelsource, /capabilities, /capabilitiesoffered, /export are
// intentionally NOT linked from the registry root doc (avoids cluttering the
// main data response for typical consumers; the fixed suffix is a well-known
// convention, akin to `.well-known`) — those stay constructed on purpose.

function serverBase() {
  return serverFetchBase(_state.serverURL);
}

// Fallback-only construction: serverBase() + path segments, no query. Used
// when no real link (_state.apiURL) is known for the current depth.
function buildBaseURL() {
  var path = _state.path;
  return serverBase() + (path.length ? '/' + path.join('/') : '');
}

function buildAPIURL() {
  var base = serverBase();
  if (_state.section === 'model')                return base + '/model';
  if (_state.section === 'modelsource')          return base + '/modelsource';
  if (_state.section === 'capabilities')         return base + '/capabilities';
  if (_state.section === 'capabilitiesoffered')  return base + '/capabilitiesoffered';
  if (_state.section === 'xregistry')            return base + '/.xregistry';
  if (_state.useExport)                          return base + '/export';

  // Link-driven navigation: when a real apiURL is already known, trust it
  // completely as-is — it already carries whatever filter belongs at this
  // position (either baked in by the server when following a link from an
  // already-filtered response, or baked in by our own applyFilters() when
  // the user explicitly applied a new filter — see plan.md "Filter support
  // in Grid/List views"). _state.filters is only re-appended here as a
  // fallback, when no real link is known yet (first-ever load, or a
  // bookmarked URL with filter= but no apiurl=). Re-appending _state.filters
  // unconditionally on top of a real apiURL that may already have its own
  // filter= baked in would silently double up the query param.
  var hasRealLink = !!_state.apiURL;
  var url = _state.apiURL || buildBaseURL();

  var q = [];
  _state.inlines.forEach(function(i) { q.push('inline=' + encodeURIComponent(i)); });
  if (!hasRealLink) {
    _state.filters.forEach(function(f) { q.push('filter=' + encodeURIComponent(f)); });
  }
  if (_state.sort)        q.push('sort=' + encodeURIComponent(_state.sort));
  if (_state.docView)     q.push('doc');
  if (_state.binary)      q.push('binary');
  if (_state.collections && collectionsEligible(_state.path)) q.push('collections');

  if (!q.length) return url;
  return url + (url.indexOf('?') >= 0 ? '&' : '?') + q.join('&');
}

// Strips only filter=... tokens from a URL's query string, via a plain
// token-level split (NOT URLSearchParams re-serialization, which would
// re-encode every other param and could mangle intentionally-unescaped
// characters, e.g. colons in ids — see plan.md's apiurl/colon-id example).
// Everything else in the URL (path, other query params) is preserved
// verbatim. Used by applyFilters() to rebuild a URL with a fresh filter
// while keeping any/all other query state intact.
function stripFilterParams(url) {
  var qIdx = url.indexOf('?');
  if (qIdx < 0) return url;
  var base = url.slice(0, qIdx);
  var qs   = url.slice(qIdx + 1);
  var kept = qs.split('&').filter(function(pair) {
    return pair.split('=')[0] !== 'filter';
  });
  return kept.length ? base + '?' + kept.join('&') : base;
}

// Computes a fresh {filters, apiURL} patch reflecting the filter builder's
// current draft — the one deliberate spot where a filter expression is
// actually constructed client-side (see plan.md "Filter support in
// Grid/List views"). Shared by JSON view's Apply button and the new
// Grid/List Apply button. Everything downstream (further real links
// returned by the server) carries this filter forward automatically —
// no other code needs to reconstruct or re-propagate it.
function applyFilters() {
  var newFilters = fbCollectFilters();
  var base = stripFilterParams(_state.apiURL || buildBaseURL());
  var filterQS = newFilters.map(function(f) { return 'filter=' + encodeURIComponent(f); }).join('&');
  var newApiURL = filterQS ? base + (base.indexOf('?') >= 0 ? '&' : '?') + filterQS : base;
  return {filters: newFilters, apiURL: newApiURL};
}

// ---- Header --------------------------------------------------------------

// Returns an HTML string for a small "Server: <url>" line, meant to be
// inserted directly below a page's title (e.g. "REGISTRY: CloudEvents").
// Always shows the real server URL (_state.serverURL), never a proxy's
// base64-encoded /xrproxy/ URL — that distinction is exactly why
// _state.serverURL is kept as the raw identity everywhere and only
// serverFetchBase()'s actual fetch calls get proxy-translated (see
// plan.md's xrproxy writeup), so no special-casing is needed here.
function serverURLLineHtml() {
  var url = _state.serverURL || window.location.origin;
  return '<div class="eg-server-url-line" title="' + esc(url) + '">Server: '
    + esc(url) + '</div>';
}

// Whether the current page's "Edit" button is enabled, independent of
// which dataView (list/json) is currently showing. Shared by renderHeader()
// (to enable/disable the Edit button itself) and renderJSONView() (to
// decide whether to show the raw-JSON-textarea editor). Keeping this in one
// place avoids the two drifting out of sync — see plan.md "Raw JSON edit".
function computeEnableEdit() {
  var isHome   = (_state.view === 'home');
  var isConfig = (_state.view === 'config');
  var isData   = !isHome && !isConfig;
  if (isConfig || isHome) return false;
  var section      = _state.section;
  var isCapSection  = isData && (section === 'capabilities');
  var isModelSourceSection = isData && (section === 'modelsource');
  var isModelSection    = isData && (section === 'model' || section === 'modelsource');
  var isCapOfferedSection = isData && (section === 'capabilitiesoffered');
  var isXRegistrySection  = isData && (section === 'xregistry');
  var isListOnlySection = isModelSection || isCapSection || isCapOfferedSection
    || isXRegistrySection;
  if (isListOnlySection) {
    return isCapSection ? _state.mutable : isModelSourceSection && _state.mutable;
  }
  return _state.mutable;
}

// Icon markup + label for each Grid/List/JSON view — shared by the
// single view-toggle button (renderHeader()) and the kebab "more" menu's
// narrow-screen fallback entries (buildMoreMenuItems()), so both always
// render identical icons for a given view.
var VIEW_LABELS = {grid: 'Grid view', table: 'List view', json: 'JSON view'};
var _headerOtherView = null; // the single "other" view the toggle button currently
                              // offers to switch to; set by renderHeader(), read by
                              // toggleDataView().
var _filtersMenuAvailable = false; // whether the current page supports filter/sort
                                    // at all (set by renderHeader(), read by
                                    // buildMoreMenuItems() to decide whether to show
                                    // the "Filter" menu entry).
var _xregDataMenuAvailable = false; // whether the current page supports the
                                     // Show/Hide xReg Data override at all (set by
                                     // renderHeader(), read by buildMoreMenuItems()).
function viewIconHtml(v, small) {
  if (v === 'grid')  return '<span class="' + (small ? 'popup-icon-grid' : 'hv-grid-icon') + '"><span></span><span></span><span></span><span></span></span>';
  if (v === 'table') return '<svg class="' + (small ? 'popup-icon-table' : 'hv-table-icon') + '" width="18" height="14" viewBox="0 0 18 14" fill="none" xmlns="http://www.w3.org/2000/svg">'
    + '<rect x="0.75" y="0.75" width="16.5" height="12.5" rx="2" stroke="currentColor" stroke-width="1.3"/>'
    + '<rect x="0.75" y="0.75" width="16.5" height="3.6" rx="1.4" fill="currentColor" fill-opacity="0.3"/>'
    + '<path d="M0.75 4.35H17.25" stroke="currentColor" stroke-width="1.1"/>'
    + '<path d="M5.6 0.75V13.25" stroke="currentColor" stroke-width="1.1"/>'
    + '<path d="M0.75 7.85H17.25" stroke="currentColor" stroke-width="1"/>'
    + '<path d="M0.75 11.05H17.25" stroke="currentColor" stroke-width="1"/>'
    + '</svg>';
  if (v === 'json')  return '<span class="' + (small ? 'popup-icon-json' : 'hv-json-sym') + '">{}</span>';
  return '';
}

// Small colored (black + xRegistry-blue) "xR" icon for the "Show/Hide xReg
// Data" menu entry — reuses the same two SVG paths as the header #xreg-logo
// (NOT the all-white variant used by the pinned #xregview-indicator), just
// at menu-icon size.
function xregIconSmallHtml() {
  return '<svg class="popup-icon-xreg" viewBox="0 0 663 800" xmlns="http://www.w3.org/2000/svg">'
    + '<path d="M490.6 800H662.2L494.2 461.6C562.2 438.2 648.6 380.5 648.6 238.8C648.6 5.3 413.9 0 413.9 0H3.40002L80.39 155.3H391.8C391.8 155.3 492.3 153.8 492.3 238.9C492.3 323.9 391.8 322.5 391.8 322.5H390.6L316.2 449L490.6 800Z" fill="black"/>'
    + '<path d="M392.7 322.4H281L266.7 346.6L196.4 466.2L111.7 322.4H0L140.5 561.2L0 800H111.7L196.4 656.2L281 800H392.7L252.2 561.2L317.9 449.6L392.7 322.4Z" fill="#0066FF"/>'
    + '</svg>';
}

function renderHeader() {
  var isHome   = (_state.view === 'home');
  var isConfig = (_state.view === 'config');
  var isData   = !isHome && !isConfig;

  // The "xR" logo always targets the home page (no server/section/path), so
  // its href never depends on the current page beyond the pathname itself.
  // Set it here (rather than as static HTML) so it stays a real, accurate
  // link — same "always show the true destination" rule as breadcrumbs/tiles.
  var logo = el('logo-link');
  if (logo) logo.setAttribute('href', buildURL({view: 'home', path: [], serverURL: ''}));

  el('breadcrumbs').style.display    = '';
  setHeaderCompact(false);

  // On home, show buttons reflecting current group's layout without corrupting _state.dataView
  var effectiveView = isHome ? currentHomeLayout() : _state.dataView;

  // Section-specific view rules:
  //   data                        — Grid view has been removed entirely
  //                                 for the data section (Registry root,
  //                                 Group/Resource/Version entities, and
  //                                 the Groups/Resources/Versions
  //                                 collections) — List/JSON only now.
  //                                 See plan.md "Grid view removed".
  //   model / modelsource         — no grid (list-style editor only), list+json available;
  //                                 edit only ever available on modelsource (never model)
  //   capabilities                — no grid (list-style editor only), list+json available;
  //                                 edit available when the doc itself is mutable
  //   capabilitiesoffered         — no grid (list-style viewer only), list+json available;
  //                                 always read-only (server-declared schema document)
  //   xregistry                   — no grid (list-style viewer only), list+json available;
  //                                 always read-only (server-declared discovery document)
  //   home 'types' (cross-registry Group Types list) — Grid removed, List
  //                                 only; home 'registry' (list of known
  //                                 registries) is unaffected.
  var section          = _state.section;
  var isModelSection    = isData && (section === 'model' || section === 'modelsource');
  var isCapSection      = isData && (section === 'capabilities');
  var isCapOfferedSection = isData && (section === 'capabilitiesoffered');
  var isXRegistrySection  = isData && (section === 'xregistry');
  var isListOnlySection = isModelSection || isCapSection || isCapOfferedSection
    || isXRegistrySection;

  var enableGrid, enableList, enableJson;
  if (isConfig) {
    enableGrid = enableList = enableJson = false;
  } else if (isHome) {
    var isHomeTypes = _state.homeGroup === 'types';
    enableGrid = !isHomeTypes; enableList = true; enableJson = false;
  } else if (isListOnlySection) {
    enableGrid = false; enableList = true; enableJson = true;
  } else {
    enableGrid = false; enableList = enableJson = true;
  }

  // Compute the single "other" (non-current) enabled view — Grid/List/
  // JSON always collapse to at most a boolean choice on every page (see
  // enableGrid/enableList/enableJson above), so there's ever only zero or
  // one "other" view to offer. The button always shows the DESTINATION
  // view's icon/title (design "1b" — click switches to whatever's shown),
  // not the currently-active one, and hides entirely when there's no
  // other view to switch to (e.g. Config page; Home "types" group, which
  // only ever offers List).
  var enabledViews = [];
  if (!isConfig) {
    if (enableGrid) enabledViews.push('grid');
    if (enableList) enabledViews.push('table');
    if (enableJson) enabledViews.push('json');
  }
  var otherView = enabledViews.filter(function(v) { return v !== effectiveView; })[0] || null;
  // `_headerOtherView` is set before renderHeader() calls renderBreadcrumbs()
  // below, and renderBreadcrumbs() re-invokes setHeaderCompact(false) —
  // whose own display logic (not this block) makes the final display=none/''
  // decision using this value, so the two stay in sync regardless of call
  // order. See setHeaderCompact().
  _headerOtherView = otherView; // consumed by toggleDataView() + setHeaderCompact()
  var viewToggleBtn = el('view-toggle-btn');
  if (viewToggleBtn && otherView) {
    viewToggleBtn.innerHTML = viewIconHtml(otherView);
    viewToggleBtn.title = 'Switch to ' + VIEW_LABELS[otherView];
  }

  // Global "Refresh" button — Home page only (both Grid/Tile and List
  // layouts), lets the user force a fresh re-probe of every listed
  // registry on demand, bypassing the in-memory probe cache (see
  // _registryProbeCache/probeRegistry()) without needing a full page
  // reload. Positioned as its own pinned icon, left of the view-toggle
  // button, same slot pattern as Edit Mode/Filter/Show-Hide-xReg-Data.
  var homeRefreshBtn = el('home-refresh-btn');
  if (homeRefreshBtn) homeRefreshBtn.style.display = isHome ? '' : 'none';

  // Pinned "editing" indicator — the only Edit-related control left
  // directly in the header (outside the kebab menu); visible only while
  // actually editing, so leaving edit mode always stays a single click.
  // Starting an edit, by contrast, goes through the kebab menu's "Edit"
  // entry (see buildMoreMenuItems()) since it's used less often.
  var editingIndicator = el('editing-indicator');
  if (editingIndicator) {
    editingIndicator.style.display = (_state.editMode && !isConfig) ? '' : 'none';
    editingIndicator.classList.add('active'); // always the "on" blue styling while shown
  }

  // Kebab "more" menu button — buildMoreMenuItems() always returns an
  // empty list on the Config page itself (nothing to reach from there:
  // no Edit, and Config navigating to Config would be a no-op), so hide
  // the button entirely rather than leave a clickable control that opens
  // nothing.
  var moreMenuBtn = el('more-menu-btn');
  if (moreMenuBtn) moreMenuBtn.style.display = isConfig ? 'none' : '';

  // Filters/Sort toggle button — now follows the same pattern as the
  // pinned editing-indicator: the entry point ("Filter") lives in the
  // kebab menu (see buildMoreMenuItems()), and this pinned toolbar icon
  // is shown only while the panel is actually open, as the one-click way
  // to close it. Only relevant for the plain 'data' section (grid/list/
  // json all already have their own filter+sort UI otherwise; model/
  // capabilities/etc support neither filter= nor sort=) and outside JSON
  // view (which always shows the full left panel already). Available
  // whenever either filter or sort is supported — Sort's picker lives in
  // this same panel too (see renderJSONLeftPanel()), so a server that
  // offers sort but not filter still needs a way to reach it from List
  // view.
  var filtersBtn = el('filters-toggle-btn');
  if (filtersBtn) {
    var svURL2 = normalizeURL(_state.serverURL || window.location.origin);
    var capLoaded2 = _capCache.hasOwnProperty(svURL2);
    if (isData && section === 'data' && !capLoaded2) {
      ensureCapCached(_state.serverURL || window.location.origin, function() { renderHeader(); });
    }
    var capData2 = _capCache[svURL2];
    var flags2 = (capData2 && capData2.flags) || [];
    var filterSupported2 = flags2.indexOf('filter') !== -1;
    var sortSupported2 = flags2.indexOf('sort') !== -1 && isCollection(_state.path);
    var panelSupported2 = filterSupported2 || sortSupported2;
    var showFiltersBtn = isData && section === 'data' && effectiveView !== 'json'
      && panelSupported2;
    _filtersMenuAvailable = showFiltersBtn;
    filtersBtn.style.display = (showFiltersBtn && _filtersPanelOpen) ? '' : 'none';
    // If we've confirmed (capabilities loaded) that this registry/section
    // genuinely doesn't support filter= or sort=, but the Grid/List
    // filters-only panel was left open from elsewhere (e.g. switching from
    // a registry that does support one of them), force it closed —
    // otherwise it's stuck open with "No options" and no button left to
    // close it (the button itself is hidden in this case). Only fires on
    // confirmed non-support, not merely because we're currently in JSON
    // view or another section, so the open/closed state still persists
    // normally across those.
    if (isData && section === 'data' && capLoaded2 && !panelSupported2
        && _filtersPanelOpen) {
      _filtersPanelOpen = false;
      history.replaceState(null, '', buildURL(_state));
      setLeftPanelVisible(_state.dataView === 'json' || _state.view === 'json');
    }
    filtersBtn.classList.toggle('active', _filtersPanelOpen);
    var fCount = (_state.filters || []).length;
    var countEl = el('filters-toggle-count');
    if (countEl) countEl.textContent = fCount ? (' (' + fCount + ')') : '';
    // Small ▲/▼ indicator showing the table's current sort direction —
    // the only visual cue for it while the panel is closed, since (per
    // plan.md "List view Sort picker") we deliberately don't use a count
    // (sort isn't a quantity) or a separate pill. Reflects whichever sort
    // mechanism is currently in effect: the Sort picker's server-side sort
    // (_state.sort) or a column-header click (_sortCol/_sortAsc) — the two
    // are mutually exclusive (see sortBy()/applyGridFilters()), so at most
    // one is ever active. Shows nothing until the user deliberately picks
    // a sort or clicks a column (the implicit default ID-ascending order
    // doesn't count as "active").
    var sortArrowEl = el('filters-toggle-sort-arrow');
    if (sortArrowEl) {
      var sortDesc = _state.sort ? (trimSplit(_state.sort, '=')[1] === 'desc')
        : (_sortCol ? !_sortAsc : null);
      sortArrowEl.textContent = sortDesc === null ? ''
        : (sortDesc ? '\u25bc' : '\u25b2');
    }
  }

  // "Show/Hide xReg Data" pinned toolbar icon — same pattern as the
  // filters-toggle-btn above: the entry point lives in the kebab menu
  // (see buildMoreMenuItems()), and this pinned icon is shown only while
  // the override is actually active, as the one-click way to revert to
  // the configured default (see effectiveXregFocused()). Available on any
  // data-section page, not just single-entity ones — even though
  // collections have no Property table to actually toggle, keeping the
  // control available (rather than appearing/disappearing as you
  // traverse collection vs. single-entity pages) is less visually
  // jarring, and it's still a valid choice to leave on/off while passing
  // through a collection page. Edit mode already always shows everything
  // regardless of this setting, so it's excluded there.
  var xregViewBtn = el('xregview-indicator');
  if (xregViewBtn) {
    // Excluded in JSON view: the setting only affects the human-readable
    // table rendering (Property tables/Config: pills), not the raw JSON
    // returned by the API — so it has no visible effect there at all.
    var inJsonViewForXreg = _state.dataView === 'json' || _state.view === 'json';
    var xregDataAvailable = isData && section === 'data' && !_state.editMode && !inJsonViewForXreg;
    _xregDataMenuAvailable = xregDataAvailable;
    xregViewBtn.style.display = (xregDataAvailable && _state.xregOverride) ? '' : 'none';
    xregViewBtn.classList.add('active'); // always the "on" blue styling while shown
  }

  // For data pages, skip breadcrumb render if label not cached yet —
  // the probe in refresh() will call renderBreadcrumbs() once the name arrives.
  if (isData) {
    var svURL = normalizeURL(_state.serverURL || window.location.origin);
    if (!_labelCache[svURL]) return;
  }
  renderBreadcrumbs();
}

function goToConfig() {
  pushState({view: 'config'});
}

function setHomeGroup(v) {
  _state.homeGroup = v;
  _opts.homeGroup  = v;
  saveOpts();
  renderHeader();    // updates active layout button for the new group
  renderHome();
}

function toggleFiltersPanel() {
  // Defensive no-op: the button is only ever shown outside JSON view (see
  // renderHeader()'s showFiltersBtn condition), but guard here too so a
  // stale click can't blow away JSON view's own always-on left panel.
  if (_state.dataView === 'json' || _state.view === 'json') return;
  _filtersPanelOpen = !_filtersPanelOpen;
  // Keep the URL in sync (buildURL() reflects _filtersPanelOpen via
  // 'panel=1') so a refresh restores the panel's open/closed state —
  // otherwise navigating the hierarchy preserves it but a reload silently
  // drops it back to closed.
  history.replaceState(null, '', buildURL(_state));
  renderHeader();
  var showGridFilters = isGridFiltersOnlyMode();
  setLeftPanelVisible(showGridFilters);
  if (showGridFilters) renderJSONLeftPanel(true);
}

function toggleHomeLayout() {
  setDataView(currentHomeLayout() === 'table' ? 'grid' : 'table');
}

// Single view-toggle button's click handler — switches to whichever view
// the button is currently displaying (always the "other"/destination view;
// see the otherView computation in renderHeader(), cached here so this
// doesn't need to re-derive enableGrid/enableList/enableJson itself).
function toggleDataView() {
  if (!_headerOtherView) return;
  setDataView(_headerOtherView);
}

function setDataView(v) {
  // Guard: leaving a collection's List view (where the inline "Add new
  // <Type>" form lives) with an open, non-empty Add form — e.g. switching
  // to JSON view — same pattern as pushState()'s matching guard.
  if (_state.section === 'data' && isCollection(_state.path) && isAddNewFormDirty()
      && v !== _state.dataView) {
    showLeaveEditDialog(
      function() { saveNewEntity(function() { setDataView(v); }); },
      function() { cancelAddEntity(); setDataView(v); }
    );
    return;
  }
  // Guard: leaving the modelsource list/editor view while mid-edit with unsaved
  // changes (e.g. switching to JSON view) — offer Save / Discard / Cancel first.
  if (_state.section === 'modelsource' && _state.editMode && _modelDirty
      && v !== _state.dataView) {
    showLeaveEditDialog(
      function() { saveModel(function() { _modelDirty = false; setDataView(v); }); },
      function() { _modelDirty = false; _modelData = deepClone(_modelSrc); setDataView(v); }
    );
    return;
  }
  // Guard: leaving the capabilities list/editor view while mid-edit with unsaved
  // changes — same pattern.
  if (_state.section === 'capabilities' && _state.editMode && _capDirty
      && v !== _state.dataView) {
    showLeaveEditDialog(
      function() { saveCapabilities(function() { _capDirty = false; setDataView(v); }); },
      function() { _capDirty = false; _capData = deepClone(_capSrc); setDataView(v); }
    );
    return;
  }
  // Guard: leaving List view mid-entity-data-edit with unsaved changes —
  // same pattern.
  if (_state.section === 'data' && _state.editMode && _dataDirty
      && v !== _state.dataView) {
    showLeaveEditDialog(
      function() { saveDataEntity('PUT', function() { setDataView(v); }); },
      function() { _dataDirty = false; _dataEditData = deepClone(_dataEditSrc); setDataView(v); },
      null,
      function() { saveDataEntity('PATCH', function() { setDataView(v); }); }
    );
    return;
  }
  // Guard: leaving JSON view's raw-textarea edit with unsaved changes — same
  // pattern (applies regardless of _state.section, see pushState()'s
  // matching guard for the same reasoning).
  if (_state.dataView === 'json' && _state.editMode && _jsonEditDirty
      && v !== 'json') {
    showLeaveEditDialog(
      function() { jsonEditSave('PUT', function() { resetJsonEditBuffer(); setDataView(v); }); },
      function() { resetJsonEditBuffer(); setDataView(v); },
      null,
      function() { jsonEditSave('PATCH', function() { resetJsonEditBuffer(); setDataView(v); }); },
      true
    );
    return;
  }
  // No unsaved JSON-edit changes, but still leaving JSON view while it was
  // in edit mode — reset the buffer so re-entering JSON view later starts
  // fresh (see resetJsonEditBuffer()'s doc comment).
  if (_state.dataView === 'json' && _state.editMode && v !== 'json'
      && _jsonEditOrigText !== null) {
    resetJsonEditBuffer();
  }

  // Retiring the dedicated Version page (see normalizeVersionDepth()):
  // this view-toggle mutates _state directly and never routes through
  // pushStateReal(), so it needs its own explicit check. Switching a
  // depth->=6 Version's JSON view to List view must redirect to the
  // Resource page + resVersion (the retired page's replacement) rather
  // than reviving List-view rendering at depth 6 in place.
  if (_state.section === 'data' && v !== 'json' && isVersionEntityPath(_state.path)) {
    var verId = _state.path[5];
    pushState({path: _state.path.slice(0, 4), resVersion: verId, dataView: v});
    return;
  }
  // The reverse: switching a Resource page's List view TO JSON view while
  // the "Version:" dropdown has a non-default version selected takes the
  // user to that VERSION's own JSON view (depth 6, its real URL) — JSON
  // view has no version-selector control of its own, so this is the only
  // way to reach a specific version's raw JSON. Selecting "Default"
  // (resVersion === '') keeps JSON view scoped to the Resource itself, as
  // today.
  if (_state.section === 'data' && v === 'json' && _state.path.length === 4
      && _state.resVersion) {
    var selVid = _state.resVersion;
    var verEntry = (_resVersionsList || []).filter(function(rv) {
      return itemNavKey(rv) === selVid;
    })[0];
    var verSelf = (verEntry && verEntry.self) || _verSelfUrl || '';
    pushState({
      path: _state.path.concat(['versions', selVid]),
      resVersion: '', dataView: 'json', apiURL: verSelf
    });
    return;
  }

  _state.dataView = v;

  // On home page, persist per-group layout preference (independent of data
  // pages) BEFORE renderHeader() below — renderHeader() recomputes each
  // button's active state from currentHomeLayout() (which reads
  // _state.homeLayouts), so this must happen first or it'll immediately
  // overwrite the manual toggle just below with the stale (pre-click) value,
  // leaving the header showing the previous button as active until the next
  // click.
  if (_state.view === 'home') {
    _state.homeLayouts[_state.homeGroup] = v;
    if (!_opts.homeLayouts) _opts.homeLayouts = {registry: 'grid', types: 'grid'};
    _opts.homeLayouts[_state.homeGroup] = v;
    saveOpts();
  }

  // Refresh header — this recomputes the view-toggle button's icon/title
  // (now showing the new "other" view) and Filters button visibility/
  // active-state/count, all of which depend on effectiveView, which just
  // changed.
  renderHeader();

  if (_state.view === 'home') {
    renderHome();
    return;
  }

  // Model/Model Source: persist per-section preference (list vs json); no per-depth
  // concept applies here, and "grid" is never a valid choice for these sections.
  if (_state.section === 'model' || _state.section === 'modelsource') {
    if (!_opts.sectionViews) _opts.sectionViews = {};
    _opts.sectionViews[_state.section] = v;
    saveOpts();
    history.replaceState(null, '', buildURL(_state));
    if (_lastData) {
      setLeftPanelVisible(v === 'json');
      v === 'json' ? renderJSONView(_lastData) : renderModelEditor(_lastData);
    }
    return;
  }

  // Capabilities: same per-section persisted preference as model/modelsource.
  if (_state.section === 'capabilities') {
    if (!_opts.sectionViews) _opts.sectionViews = {};
    _opts.sectionViews[_state.section] = v;
    saveOpts();
    history.replaceState(null, '', buildURL(_state));
    if (_lastData) {
      setLeftPanelVisible(v === 'json');
      if (v === 'json') {
        renderJSONView(_lastData);
      } else {
        var svBaseC2 = (_state.serverURL || window.location.origin).replace(/\/$/, '');
        ensureOfferedCached(svBaseC2, function(offered) {
          renderCapabilitiesEditor(_lastData, offered);
        });
      }
    }
    return;
  }

  // Capabilities Offered: same per-section persisted preference; always
  // read-only (see refresh()'s isCapOfferedSection branch).
  if (_state.section === 'capabilitiesoffered') {
    if (!_opts.sectionViews) _opts.sectionViews = {};
    _opts.sectionViews[_state.section] = v;
    saveOpts();
    history.replaceState(null, '', buildURL(_state));
    if (_lastData) {
      setLeftPanelVisible(v === 'json');
      v === 'json' ? renderJSONView(_lastData) : renderCapabilitiesOfferedViewer(_lastData);
    }
    return;
  }

  // .xregistry: same per-section persisted preference; always read-only
  // (see refresh()'s isXRegistrySection branch).
  if (_state.section === 'xregistry') {
    if (!_opts.sectionViews) _opts.sectionViews = {};
    _opts.sectionViews[_state.section] = v;
    saveOpts();
    history.replaceState(null, '', buildURL(_state));
    if (_lastData) {
      setLeftPanelVisible(v === 'json');
      v === 'json' ? renderJSONView(_lastData) : renderXRegistryViewer(_lastData);
    }
    return;
  }

  // Data page: persist grid/table preference per path depth.
  // JSON is not saved here — it's sticky within a session (see pushState) but
  // returns to the saved grid/table view when entering a registry fresh.
  if (!_opts.depthViews) _opts.depthViews = {};
  if (v === 'grid' || v === 'table') {
    _opts.depthViews[_state.path.length] = v;
    saveOpts();
  }

  // Keep URL in sync so refresh restores the same view
  history.replaceState(null, '', buildURL(_state));

  if (_lastData) {
    setLeftPanelVisible(v === 'json' || _filtersPanelOpen);
    if (v === 'json') { renderJSONViewForCurrentTab(_lastData); return; }
    if (_filtersPanelOpen) renderJSONLeftPanel(true);
    var coll = isCollection(_state.path);
    if (coll) {
      renderTableView(_lastData);
    } else {
      renderSingleEntity(_lastData);
    }
  }
}

// Build the registry dropdown: Home + known registries + Add
function serverLabel(url) {
  var norm = normalizeURL(url || window.location.origin);
  var override = getNameOverride(norm);
  if (override) return override;
  if (_labelCache[norm]) return _labelCache[norm];
  return url.replace(/^https?:\/\//, '').replace(/\/$/, '') || url;
}

// When JSON view is entered while the List-view "Metadata" tab is active
// (_state.docTab === 'meta'), show the /meta object's own JSON instead of
// silently falling back to the parent Resource/Version's JSON — the user
// should see whatever they were just looking at. Only applies to single
// entities (collections never have a Metadata tab).
function renderJSONViewForCurrentTab(data) {
  if (_state.section === 'data' && _state.docTab === 'meta' && !isCollection(_state.path)) {
    // Trust the shared _metaData cache only when _metaLoadedFor confirms it
    // was actually populated for THIS entity (same guard renderSingleEntity()
    // uses) — _metaData is shared with loadMetaDetails() (List view's own
    // Metadata tab), so without this check, switching to JSON view could
    // show a stale, different resource's meta object left over from
    // whichever entity was last viewed there. See loadMetaDetails()'s
    // matching "stillCurrent()" comment for the fuller race explanation.
    if (_metaData && _metaLoadedFor === (data && data.self)) { renderJSONView(_metaData); return; }
    var metaUrl = data && data.metaurl;
    if (metaUrl) {
      fetch(metaUrl)
        .then(function(r) { if (!r.ok) throw new Error('HTTP ' + r.status); return r.json(); })
        .then(function(d) {
          if (_lastData !== data) return; // navigated away before this resolved — discard
          _metaData = d;
          _metaLoadedFor = data.self;
          renderJSONView(d);
        })
        .catch(function() { if (_lastData === data) renderJSONView(data); }); // fall back to entity JSON on error
      return;
    }
  }
  renderJSONView(data);
}

function setView(view) {
  pushState({view: view});
}

function toggleEdit() {
  pushState({editMode: !_state.editMode});
}

// Flips the per-session "Show/Hide xReg Data" override (see
// effectiveXregFocused()) — same pushState() pattern as toggleEdit(), so
// the URL (xrv=1) and the full page content (Identity/Versioning & State
// sections, Config: pills) both update together.
function toggleXregOverride() {
  pushState({xregOverride: !_state.xregOverride});
}

// ---- Breadcrumbs ---------------------------------------------------------

var _bcSep  = '<span class="bc-space"></span><span class="bc-sep">/</span><span class="bc-space"></span>';
var _bcSegs = []; // current segments, shared with popup openers

// Returns [{label, onclick|null, href|null, isCurrent}] for the current state
function buildBreadcrumbSegments() {
  if (_state.view === 'home')   return null; // handled specially in renderBreadcrumbs
  if (_state.view === 'config') return [{label:'Config',     onclick:null, isCurrent:true}];

  var segs = [];
  var regLabel = serverLabel(_state.serverURL || window.location.origin);
  var regClick = guardedOnclick('pushState({path:[],section:\'data\',useExport:false})');
  // Use the cached real/filtered root URL (if any) so the hover-href
  // reflects the root's OWN active filter, not whatever the current
  // (deeper) page's already-relativized filter happens to be. See
  // rootApiURL's declaration comment / pushStateReal().
  var regHref  = pageHref([], _state.rootApiURL || '', {useExport: false});
  var isSection = _state.section !== 'data';
  segs.push({label: regLabel, onclick: isSection || _state.path.length > 0 ? regClick : null, href: regHref, isCurrent: !isSection && _state.path.length === 0});

  // If in a section view, add the section name as the last breadcrumb
  if (isSection) {
    var sectionLabels = {model:'Model', modelsource:'Model Source', capabilities:'Capabilities', capabilitiesoffered:'Capabilities Offered', xregistry:'.xregistry'};
    segs.push({label: sectionLabels[_state.section] || _state.section, onclick: null, href: null, isCurrent: true});
    return segs;
  }

  _state.path.forEach(function(seg, i) {
    var newPath = _state.path.slice(0, i + 1);
    var isLast  = (i === _state.path.length - 1);
    var click   = isLast ? null
      : guardedOnclick('pushState({path:' + esc(JSON.stringify(newPath))
        + ',section:\'data\'})');
    // Real bookmarkable URL for this breadcrumb level — a cached real link
    // (_state.crumbURLs[i]) if this ancestor was visited this session,
    // otherwise the same trim fallback pushStateReal() would use.
    var href = isLast ? null : pageHref(newPath, _state.crumbURLs[i] || '');
    segs.push({label: seg, onclick: click, href: href, isCurrent: isLast});
  });

  // When the Resource/Version page's Metadata tab is active, the visible
  // content (List view's Metadata tab, or JSON view's routed /meta doc —
  // see renderJSONViewForCurrentTab()) is really a distinct sub-entity, not
  // just another view of the Resource/Version itself. List view still has
  // its own tab bar as a visual cue, but JSON view has none — so without an
  // extra breadcrumb segment here, JSON view gives no indication you're
  // looking at /meta rather than the Resource/Version's own document. Only
  // applies at Resource (depth 4) or Version (depth >= 6) pages — the only
  // depths that ever have a Metadata tab (see tabDefs.push() in
  // renderSingleEntity()).
  var metaDepth = _state.path.length;
  var isResourceOrVersion = metaDepth === 4 || metaDepth >= 6;
  // Only JSON view gets the extra "meta" segment: List view's own tab bar
  // already makes the Metadata-tab selection visually unambiguous (see
  // switchDocTab()'s perf-skip comment), so adding a breadcrumb segment
  // there too would be redundant — and worse, it would outlive the tab
  // switch that goes back to List (a fresh, full render, not the tab-click
  // skip-optimization) since docTab stays 'meta' the whole time.
  if (isResourceOrVersion && _state.docTab === 'meta' && _state.dataView === 'json') {
    var lastSeg = segs[segs.length - 1];
    if (lastSeg) {
      lastSeg.isCurrent = false;
      // This segment was originally built as the last (current, non-link)
      // segment, so it got onclick:null/href:null above. Now that "meta" is
      // the true last segment, restore a real link so clicking the
      // Resource/Version id navigates back to its own (non-meta) tab.
      var lastPath = _state.path.slice(0, metaDepth);
      lastSeg.onclick = guardedOnclick('pushState({path:' + esc(JSON.stringify(lastPath))
        + ',section:\'data\',docTab:\'\'})');
      lastSeg.href = pageHref(lastPath, _state.crumbURLs[metaDepth - 1] || '');
    }
    segs.push({label: 'meta', onclick: null, href: null, isCurrent: true});
  }
  return segs;
}

// Every nav <a>'s onclick handler must call this FIRST and bail out (return
// true, i.e. let the browser perform its native default action) whenever the
// click carries a "open in new tab/window" gesture — ctrl/cmd/shift-click or
// a middle-click. Without this check, unconditionally returning false (to
// suppress the default action for our fast pushState() SPA navigation) would
// also suppress the new-tab gesture, since browsers honor onclick's return
// value even when a modifier key is held. Real middle-clicks normally never
// invoke a "click" onclick handler at all (browsers fire it as a separate
// "auxclick" instead) but the e.button === 1 check is kept as a defensive
// no-op for browsers/environments where that isn't true.
function navShouldDefault(e) {
  return !!(e && (e.ctrlKey || e.metaKey || e.shiftKey || e.button === 1));
}

// Wraps a pushState/navigate expression with the navShouldDefault() guard so
// every nav <a>'s onclick attribute gets the same "let the browser handle
// ctrl/cmd/shift/middle-click natively" behavior with one call, instead of
// each call site needing to remember the "if(...)return true;" boilerplate.
function guardedOnclick(expr) {
  return 'if(navShouldDefault(event))return true;' + expr + ';return false';
}

// The real, bookmarkable URL for a page at `path` with the (possibly cached,
// possibly empty/fallback) `apiURL` — used for <a href> hover-preview and
// native ctrl/middle-click/"open in new tab" targets across breadcrumbs,
// tiles, rows, and nav pills. Actual (fast, no-reload) navigation always goes
// through the accompanying onclick's pushState() call, not this href.
//
// Sort is only valid on collection-shaped paths — the server explicitly
// rejects `sort=` on a non-collection endpoint (spec `sort_noncollection`).
// A real click already resets `_state.sort` correctly via pushStateReal()'s
// "fresh navigation" defaults, but this synthetic href doesn't go through
// that reset — so if the CURRENT page has an active sort and `path` points
// to a non-collection destination (e.g. a collection row's entity link),
// drop sort here too. Otherwise ctrl/middle-click ("open in new tab") or a
// copied/bookmarked link would 400.
//
// Filter needs similar care, but for a different reason: when `apiURL` is a
// real server-provided link (e.g. a Group Type tile's `<plural>url`, or an
// entity row's own `self`), it already carries whatever filter applies to
// THAT destination — server-side, relativized to its own abstract path (see
// FiltersRelativeToAbstract()) — which may look nothing like the CURRENT
// page's `_state.filters` string (e.g. root filter `dirs.files.epoch>0`
// becomes `files.epoch>0` once you're inside `dirs`). A real click already
// re-derives `_state.filters` correctly from the new `apiURL` via
// syncFiltersFromApiURL() in pushStateReal() — mirror that exact logic here
// too, so this synthetic href shows the same (single, correctly-scoped)
// filter a real navigation would end up with, instead of carrying the
// CURRENT page's differently-scoped filter forward and showing both.
function pageHref(path, apiURL, extra) {
  var st = Object.assign({}, _state, {path: path, apiURL: apiURL || '', section: 'data'}, extra || {});
  // JSON-view option flags (inline/sort/docView/binary/collections) are
  // local to whichever depth they were actually applied at — unlike
  // filters, they aren't designed to propagate across the hierarchy, so
  // Object.assign() above would otherwise leak the CURRENT (possibly much
  // deeper) page's own values onto every other page's synthetic hover-
  // preview href, even the registry root's. Restore whatever was really
  // last applied AT this destination depth instead, from the per-depth
  // query-string snapshot pushStateReal() caches in crumbOpts/rootOpts —
  // reusing the exact same parser loadStateFromURL() uses for a real
  // browser refresh/bookmark (parseJSONOptionsFromQuery()), rather than
  // keeping a second bespoke list of these fields in sync. A depth never
  // actually visited this session (no cached snapshot yet) simply gets
  // none of these applied, same as a brand new page would.
  var cachedOptsQS = (path.length > 0)
    ? (_state.crumbOpts || [])[path.length - 1]
    : (_state.rootOpts || '');
  var jsonOpts = parseJSONOptionsFromQuery(cachedOptsQS || '');
  st.inlines     = jsonOpts.inlines;
  st.sort        = jsonOpts.sort;
  st.docView     = jsonOpts.docView;
  st.binary      = jsonOpts.binary;
  st.collections = jsonOpts.collections;
  // Sort is only valid on collection-shaped paths — the server explicitly
  // rejects `sort=` on a non-collection endpoint (spec `sort_noncollection`).
  // The cached snapshot above might have been taken at a collection-shaped
  // ancestor whose sort doesn't apply to this specific `path` (or there may
  // be no cached snapshot at all for a non-collection destination), so this
  // guard still applies on top, same as before.
  if (!isCollection(st.path)) st.sort = '';
  if (st.section === 'data' && st.apiURL) {
    st.filters = filtersFromUrl(st.apiURL);
  } else if (st.section === 'data' && path.length > 0 && path.length % 2 === 0) {
    // Entity-instance depth (a Group/Resource/Version instance, e.g.
    // Contoso.ERP or s1) with no cached real URL of its own. crumbURLs
    // only gets populated for depths actually landed on directly — but
    // browsing purely via nested <plural>url links (the normal/expected
    // way to drill down, per @duglin) jumps straight from one COLLECTION
    // to the next, never landing on the entity's own page in between, so
    // its crumbURLs slot stays empty. An entity's own filter context is
    // always IDENTICAL to its immediate parent COLLECTION's (crossing
    // from a collection into one of its member entities never trims/
    // relativizes a filter clause — only entering a NESTED collection
    // does, per the server's FiltersRelativeToAbstract()) — so borrow the
    // parent collection's cached URL here instead of falling through to
    // _state.filters below, which is the CURRENT (possibly far deeper,
    // unrelated) page's own value.
    var parentURL = _state.crumbURLs && _state.crumbURLs[path.length - 2];
    if (parentURL) st.filters = filtersFromUrl(parentURL);
  }
  return buildURL(st);
}

// An entity's own `self` link never carries filter context by default —
// it's the bare canonical URL. But when the CURRENT collection view is
// itself filtered, the server has already confirmed (via `filter=` on any
// URL, not just collections) that appending the same currently-active
// filter onto an item's `self` link correctly rescopes that item's own
// nested-collection links too (e.g. entity `self`+`?filter=schemas.
// versions.schemaid=X` rescopes that entity's own `schemasurl`, same as it
// would on the collection URL that listed it). So: pass entity navigation
// through here (instead of raw `item.self`) so the real, hoverable/
// ctrl-clickable link — and the plain GET that follows a click — carries
// the filter forward, without any client-side seed/caching trickery.
//
// Version entities are the one exception: a version is always a leaf —
// spec-guaranteed to have no nested `<plural>url` collections of its own
// — so there's nothing left to rescope by carrying the filter forward.
// Appending it there would be misleading (implying more scoping is still
// happening when it isn't) with no actual effect, so skip it for those.
//
// But Versions aren't the ONLY case where a filter clause has nothing left
// to rescope: `_state.filters`' clauses are already relative to the
// CURRENT collection's abstract (e.g. viewing `dirs/d1/files` filtered by
// `epoch>0` — a bare, dot-free clause about the FILE's own attribute, used
// to decide which files show up in that collection). Once you're AT a
// specific member of that collection (e.g. file `f1`), a clause only still
// means something if it references one of THAT member's own nested child
// collections (e.g. `versions.epoch>0` would still rescope `f1`'s own
// `versionsurl`) — a bare/terminal clause like `epoch>0` has nothing left
// to rescope on `f1`'s own page (confirmed via the server: `GET
// f1$details?filter=epoch>0` returns 200 with `versionsurl` completely
// unfiltered) and would misleadingly show up as an "active filter" on a
// page where it does nothing. `filtersRelevantForEntity()` drops any such
// now-terminal clauses, keeping only those that still address one of the
// destination's real children (its resource plurals for a Group entity,
// or `versions` for a Resource entity).
function entityHrefWithFilter(self, itemPath) {
  var isVersionEntity = itemPath && itemPath.length === 6 && itemPath[4] === 'versions';
  if (!self || isVersionEntity || !isCollection(_state.path)
      || !_state.filters || !_state.filters.length) return self || '';
  var relevant = filtersRelevantForEntity(_state.filters, itemPath);
  if (!relevant.length) return self;
  var qs = relevant.map(function(f) { return 'filter=' + encodeURIComponent(f); }).join('&');
  return self + (self.indexOf('?') >= 0 ? '&' : '?') + qs;
}

// The nested child-collection plural names an entity at `path` actually
// has — used by filtersRelevantForEntity() to tell whether a filter clause
// still references something below that entity (and thus is still worth
// carrying forward) or is "terminal" (about the entity's own attribute,
// nothing left to rescope). Groups: their declared resource plurals.
// Resources: always just `versions`. Registry root / Version entities:
// no children of the relevant kind.
function childCollectionsFor(path) {
  var depth = (path || []).length;
  if (depth === 2) {
    var model = _modelCache[normalizeURL(serverBase())];
    var gm = model && model.groups && model.groups[path[0]];
    return gm && gm.resources ? Object.keys(gm.resources) : [];
  }
  if (depth === 4) return ['versions'];
  return [];
}

// Filters `filters` (an array of OR-groups, each an AND-joined filter
// clause string, e.g. "a=1,b=2") down to only the clauses that still
// reference one of `path`'s own nested child collections — mirrors the
// server's FiltersRelativeToAbstract(), which does the same per-clause
// keep/drop when computing a `<COLLECTION>url`'s embedded filter. See
// entityHrefWithFilter()'s comment for the full rationale.
function filtersRelevantForEntity(filters, path) {
  var children = childCollectionsFor(path);
  if (!children.length) return [];
  return (filters || []).map(function(group) {
    var kept = trimSplit(group, ',').filter(function(clause) {
      return children.some(function(c) { return clause.indexOf(c + '.') === 0; });
    });
    return kept.join(',');
  }).filter(Boolean);
}

function renderSegment(seg) {
  if (seg.isCurrent || !seg.onclick) {
    return '<span class="bc-current">' + esc(seg.label) + '</span>';
  }
  return '<a class="bc-link" href="' + esc(seg.href || '#') + '" onclick="' + seg.onclick + '">' + esc(seg.label) + '</a>';
}

function breadcrumbsFromSegments(segs) {
  return segs.map(function(s) { return _bcSep + renderSegment(s); }).join('');
}

// Copy-to-clipboard link, appended after the last breadcrumb segment, so
// there's always a plain, curl-able URL for exactly the data currently
// being displayed — no UI-only params (view=, panel=, etc.), just the real
// API request buildAPIURL() would make (respecting any active
// filter/sort/inline/section). Not shown on the Home or Config pages,
// since neither has a single "data" URL to copy.
function showCopyLinkBtn() {
  return _state.view !== 'home' && _state.view !== 'config';
}

// Standard "two overlapping documents" copy glyph (same generic design
// used by Material Icons' content_copy / most icon sets) rendered as an
// inline SVG rather than the clipboard emoji, for a crisper, more
// consistent look across platforms/fonts.
var _copyIconSVG = '<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor" aria-hidden="true">'
  + '<path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>'
  + '</svg>';

function copyLinkBtnHTML() {
  // Second tooltip line previews the exact URL that will be copied, so
  // users don't have to click first just to see what they're about to get.
  // toRealURL() here is display-only (this is a preview, not a fetch/nav
  // target) — when proxied, show/copy the real remote URL rather than our
  // local /xrproxy/<base64> form, same as the other display-only sites.
  var urlPreview = toRealURL(buildTabAwareAPIURL());
  return '<button class="icon-btn bc-copy-btn" onclick="copyCurrentAPIURL(event)" '
       + 'title="Copy API URL for this data\n' + esc(urlPreview) + '">' + _copyIconSVG + '</button>';
}

// Refreshes the copy-URL button's tooltip in place, without re-rendering the
// whole breadcrumb bar — needed because switching the Document/Details tab
// or the version-selector dropdown changes what buildTabAwareAPIURL() would
// return, but neither does a full renderBreadcrumbs() (see switchDocTab()/
// onVersionSelectChange()). Without this the tooltip text goes stale even
// though the actual copy (which always calls buildTabAwareAPIURL() live at
// click-time) stays correct.
function refreshCopyLinkBtnTooltip() {
  var btn = document.querySelector('.bc-copy-btn');
  if (btn) btn.title = 'Copy API URL for this data\n' + toRealURL(buildTabAwareAPIURL());
}

function copyCurrentAPIURL(event) {
  if (event) event.stopPropagation();
  egCopy(toRealURL(buildTabAwareAPIURL()), 'API URL');
}

// buildAPIURL() returns the URL used to FETCH the current Resource/Version
// entity page (always the entity's own default-version data, with
// $details appended as needed for the fetch to work) — it has no notion
// of which tab (Version Details / Document / Metadata) or which version
// (via the Resource page's version-selector dropdown) is CURRENTLY being
// displayed on screen. The copy button's whole point is "give me the URL
// for what I'm looking at right now", so this wraps buildAPIURL() with
// that extra tab/version awareness for Resource and Version entity pages
// only; every other page (collections, Groups, model/modelsource/etc.)
// falls through to plain buildAPIURL() unchanged.
function buildTabAwareAPIURL() {
  var path = _state.path;
  var depth = (path || []).length;
  var isResource = _state.section === 'data' && depth === 4;
  var isVersion  = _state.section === 'data' && depth >= 6 && path[4] === 'versions';
  if (!isResource && !isVersion) return buildAPIURL();

  // The entity JSON currently backing the visible tabs — reflects the
  // version-selector dropdown's pick on a Resource page (onVersionSelectChange()
  // keeps this in sync), or is just the Version's own data on a Version page.
  var activeData = _docActiveVersionData || _resDefaultData || _lastData;
  var selfUrl = activeData && activeData.self;
  if (!selfUrl) return buildAPIURL();

  // _state.docTab is '' whenever the user hasn't switched away from
  // whichever tab happens to render first (see switchDocTab()) — and tab
  // ORDER isn't fixed: it's "Document" first when the resource type has
  // one (resourceHasDocument()), otherwise "Version Details"/"Version" is
  // first (see tabDefs.push() order in the entity render). So an empty
  // docTab does NOT reliably mean "defver/version" — read the truly active
  // tab straight from the DOM (already authoritative, always in sync)
  // rather than re-deriving/guessing that same ordering here.
  //
  // The DOM tab bar doesn't exist yet on the very first breadcrumb render
  // (renderBreadcrumbs() runs before renderSingleEntity() builds the tab
  // bar), so the fallback below has to guess which tab will end up active
  // — it must match the same hasDocument-first ordering tabDefs.push()
  // uses, otherwise a Document-tab resource's tooltip would wrongly show
  // the $details-suffixed "Version Details" URL instead of the plain
  // document URL until the user manually switches tabs.
  var activeTabEl = document.querySelector('.eg-doc-tab.active[data-tab]');
  var tab;
  if (activeTabEl) {
    tab = activeTabEl.getAttribute('data-tab');
  } else {
    var modelFallback = _modelCache[normalizeURL(_state.serverURL || window.location.origin)] || null;
    tab = resourceHasDocument(modelFallback, path) ? 'doc' : (isResource ? 'defver' : 'version');
  }

  if (tab === 'doc') {
    // Document tab: the plain (no $details) URL — GETting it returns the
    // document's actual content itself (computed by the server whether the
    // document is a real hosted <key>url, base64, or an inline JSON value),
    // exactly matching what the Document tab's preview shows.
    return selfUrl.replace(/\$details$/, '');
  }
  if (tab === 'meta' && isResource) {
    return resolveResourceMetaUrl(activeData) || selfUrl;
  }
  // Default "Version Details" tab (defver/version key): the entity's own
  // metadata view — same $details-forcing fetchWithDetailsFallback() would
  // do, just computed directly since we already know this is a Resource/
  // Version entity.
  return /\$details$/.test(selfUrl) ? selfUrl : selfUrl.replace(/(\?|$)/, '$details$1');
}

function renderBreadcrumbs() {
  var nav = el('breadcrumbs');
  if (!nav) return;

  closeHeaderPopup();
  setHeaderCompact(false);

  // Home page: render a 2-button Registries / Group Types pill in the breadcrumb
  if (_state.view === 'home') {
    var hg = _state.homeGroup;
    nav.innerHTML = _bcSep
      + '<span class="bc-home-pill">'
      +   '<span class="bc-home-opt' + (hg === 'registry' ? ' active' : '') + '" onclick="setHomeGroup(\'registry\')">Registries</span>'
      +   '<span class="bc-home-opt' + (hg === 'types'    ? ' active' : '') + '" onclick="setHomeGroup(\'types\')">Group Types</span>'
      + '</span>';
    nav.style.overflow = '';
    // Home has no breadcrumb segments to collapse (just the pill above), but
    // on narrow screens the flex layout can still squeeze #breadcrumbs down
    // below the pill's natural content width. The pill has its own
    // overflow:hidden (to round its corners), which traps that clipping —
    // so nav.scrollWidth never reports it. Check the pill itself instead:
    // if its content (scrollWidth) is wider than the box it was actually
    // given (clientWidth), it's being clipped, and folding the right-side
    // buttons into the compact dropdown frees up enough room to fit it.
    requestAnimationFrame(function() {
      var pill = nav.querySelector('.bc-home-pill');
      if (pill && pill.scrollWidth > pill.clientWidth) setHeaderCompact(true);
    });
    return;
  }

  _bcSegs = buildBreadcrumbSegments();
  nav.innerHTML = breadcrumbsFromSegments(_bcSegs) + (showCopyLinkBtn() ? copyLinkBtnHTML() : '');
  nav.style.overflow = '';

  requestAnimationFrame(function() {
    var headerEl    = document.getElementById('header');
    var headerLeft  = document.getElementById('header-left');
    var headerRight = document.getElementById('header-right');
    if (!headerEl || !headerLeft || !headerRight) return;
    var available = headerEl.offsetWidth - headerLeft.offsetWidth - headerRight.offsetWidth - 16;

    if (nav.scrollWidth <= available) return; // full fits

    // Level 1: collapse middle segments into …
    collapseBreadcrumbs(nav, _bcSegs);

    requestAnimationFrame(function() {
      var available2 = headerEl.offsetWidth - headerLeft.offsetWidth - headerRight.offsetWidth - 16;
      if (nav.scrollWidth <= available2) return; // level 1 fits

      // Level 2: single label + compact right buttons
      collapseLevel2(nav, _bcSegs);
    });
  });
}

function collapseBreadcrumbs(nav, segs) {
  if (segs.length <= 2) { nav.style.overflow = 'hidden'; return; }
  nav.style.overflow = 'visible';
  var first = segs[0];
  var last  = segs[segs.length - 1];
  nav.innerHTML = _bcSep + renderSegment(first)
    + _bcSep + '<button class="bc-ellipsis" onclick="openBcEllipsis(event)" title="Show path">'
    + '&hellip;<span class="bc-ellipsis-arrow">&#9660;</span></button>'
    + _bcSep + renderSegment(last)
    + (showCopyLinkBtn() ? copyLinkBtnHTML() : '');
}

function collapseLevel2(nav, segs) {
  nav.style.overflow = 'visible';
  var last = segs[segs.length - 1];
  nav.innerHTML = _bcSep
    + '<button class="bc-full-menu" onclick="openBcFull(event)" title="Navigate">'
    + esc(last.label) + '<span class="bc-ellipsis-arrow"> &#9660;</span></button>';
  setHeaderCompact(true);
}

// ---- Shared header popup -------------------------------------------------

function openHeaderPopup(anchorEl, items, rightAlign) {
  var popup = el('header-popup');
  if (!popup) return;
  popup.innerHTML = items.map(function(item) {
    if (item.sep) return '<hr class="popup-sep">';
    var cls = 'popup-item' + (item.active ? ' popup-item-active' : '');
    // item.icon is optional — only the kebab "more" menu passes it; the
    // breadcrumb ellipsis/full popups (openBcEllipsis()/openBcFull()) don't
    // set it, so they render with no icon slot at all (unchanged layout).
    var iconHtml = item.icon ? '<span class="popup-item-icon">' + item.icon + '</span>' : '';
    if (item.onclick) {
      return '<a class="' + cls + '" href="#" onclick="closeHeaderPopup();' + item.onclick + '">'
           + iconHtml + esc(item.label) + '</a>';
    }
    return '<span class="' + cls + ' popup-item-cur">' + iconHtml + esc(item.label) + '</span>';
  }).join('');
  var rect = anchorEl.getBoundingClientRect();
  popup.style.top = (rect.bottom + 4) + 'px';
  if (rightAlign) {
    popup.style.left  = 'auto';
    popup.style.right = (window.innerWidth - rect.right) + 'px';
  } else {
    popup.style.left  = rect.left + 'px';
    popup.style.right = 'auto';
  }
  popup.classList.add('popup-open');
}

function closeHeaderPopup() {
  var popup = el('header-popup');
  if (popup) popup.classList.remove('popup-open');
}

function toggleHeaderPopup(anchorEl, items, rightAlign) {
  var popup = el('header-popup');
  if (popup && popup.classList.contains('popup-open')) { closeHeaderPopup(); return; }
  openHeaderPopup(anchorEl, items, rightAlign);
}

function openBcEllipsis(e) {
  e.stopPropagation();
  var middle = _bcSegs.slice(1, -1);
  toggleHeaderPopup(e.currentTarget, middle.map(function(s) {
    return {label: s.label, onclick: s.onclick};
  }));
}

function openBcFull(e) {
  e.stopPropagation();
  toggleHeaderPopup(e.currentTarget, _bcSegs.map(function(s) {
    return {label: s.label, onclick: s.onclick, active: s.isCurrent};
  }));
}

function openMoreMenu(e) {
  e.stopPropagation();
  toggleHeaderPopup(e.currentTarget, buildMoreMenuItems(), true); // right-align
}

function buildMoreMenuItems() {
  var items = [];
  var isHome   = (_state.view === 'home');
  var isConfig = (_state.view === 'config');
  // Note: the "Registries / Group Types" home pill is NOT duplicated here —
  // it lives in the breadcrumb area, unaffected by any of this.

  // Narrow-screen fallback: the view-toggle button itself is hidden
  // directly in the header while _headerCompact is true (see
  // setHeaderCompact()), so fold its one "switch to X" action in here too
  // — otherwise it'd be unreachable on very narrow viewports. Reuses the
  // same _headerOtherView renderHeader() already computed, so this can't
  // drift out of sync with which view is actually available.
  if (_headerCompact && !isConfig && _headerOtherView) {
    items.push({label: VIEW_LABELS[_headerOtherView], onclick: "setDataView('" + _headerOtherView + "')", icon: viewIconHtml(_headerOtherView, true)});
  }
  // Filter — always available here (not just narrow screens) now that the
  // standalone filters-toggle-btn only pins itself while the panel is
  // open; the pinned button (see renderHeader()) remains the one-click
  // way to *close* the panel once it's open.
  if (!isConfig && _filtersMenuAvailable) {
    items.push({label: 'Filter', onclick: 'toggleFiltersPanel()', active: _filtersPanelOpen, icon: '<span class="popup-icon-funnel"></span>'});
  }
  // Show/Hide xReg Data — session override of the Config page's Default
  // View option (see effectiveXregFocused()). Label always describes what
  // clicking will do *right now*, based on the current effective state
  // (config default XOR override), not the literal configured default —
  // same "1b" convention as the view-toggle button. The pinned
  // #xregview-indicator (see renderHeader()) remains the one-click way to
  // revert to the default once overridden.
  if (!isConfig && _xregDataMenuAvailable) {
    items.push({
      label: effectiveXregFocused() ? 'Hide xReg Data' : 'Show xReg Data',
      onclick: 'toggleXregOverride()',
      active: _state.xregOverride,
      icon: xregIconSmallHtml()
    });
  }
  // Edit — always available here (not just narrow screens) now that the
  // standalone edit-btn has moved into this menu; the pinned
  // editing-indicator (see renderHeader()) remains the one-click way to
  // *exit* edit mode once it's on.
  if (!isConfig && computeEnableEdit()) {
    items.push({label: 'Edit Mode', onclick: 'toggleEdit()', active: _state.editMode, icon: '<span class="popup-icon-pencil">&#9998;</span>'});
  }
  // Config — always available here (not just narrow screens) now that the
  // standalone gear-btn has moved into this menu.
  if (!isConfig) {
    if (items.length) items.push({sep: true});
    items.push({label: 'Config', onclick: 'goToConfig()', icon: '<span class="popup-icon-gear">&#9881;</span>'});
  }
  return items;
}

function setHeaderCompact(compact) {
  _headerCompact = compact;
  // Only the view-toggle button folds away in compact mode — the kebab
  // "more" menu button is permanently visible at every width (it's always
  // the home for Edit/Config now, not just a narrow-screen fallback), and
  // the pinned editing-indicator stays visible too so exiting edit mode
  // never requires opening a menu. This is the single source of truth for
  // the toggle button's display: it's hidden when compact (folded into the
  // kebab menu) OR when there's no "other" view at all for this page (see
  // renderHeader()'s _headerOtherView) — combining both here, rather than
  // in renderHeader() itself, since renderHeader() always ends by calling
  // renderBreadcrumbs(), which re-invokes setHeaderCompact(false) and would
  // otherwise clobber a renderHeader()-set "no other view" hide.
  // Hides the whole #view-controls wrapper (not just the button inside)
  // — the wrapper has its own border/background, so hiding only the
  // button left a stray empty bordered box (a thin vertical line) behind
  // on pages with no "other" view (Config; Home "types").
  var viewControls = el('view-controls');
  if (viewControls) viewControls.style.display = (compact || !_headerOtherView) ? 'none' : '';
}


// Close header popup and any open error popups on outside click
document.addEventListener('click', function() {
  closeHeaderPopup();
  qsa('.server-card-err-popup').forEach(function(p) { p.style.display = 'none'; });
});

function crumb(label, clickExpr) {
  if (!clickExpr) return '<span class="bc-current">' + esc(label) + '</span>';
  return '<a class="bc-link" href="#" onclick="' + clickExpr + ';return false">' + esc(label) + '</a>';
}

// ---- Refresh (main render loop) ------------------------------------------

var _lastData = null;
var _metaData = null;          // cached meta response for resource page meta box
// Which resource's `self` the above _metaData/_metaEditSrc/_metaEditData
// currently belong to — lets renderSingleEntity()'s redundant re-render
// (fired again once ensureModelCached()'s async callback resolves for the
// SAME resource — see its `_metaData = null` reset below) tell "same
// resource, just re-rendering" apart from "genuinely navigated to a
// different resource", so an in-progress Meta-tab edit isn't silently
// wiped out by that harmless second render. See loadMetaDetails().
var _metaLoadedFor = null;
var _docPillsMetaCompat = null; // cached meta.compatibility value for the Document tab's
                                 // Compatibility pill; null = not yet fetched, '' = fetched
                                 // but not set/unavailable. Reset whenever a new resource
                                 // page renders (compatibility is resource-wide, so it does
                                 // NOT need to be re-fetched when the version-selector
                                 // dropdown picks a different version of the same resource).
var _metaResourceIdField = ''; // resource's own ID field name, set when resource page renders
var _metaEntityType = '';      // resource's singular type name, used by List view's meta table header
var _docSingular = '';         // resource's singular type name, used by List view's Document tab
var _docPreviewLoaded = false; // whether the Document tab's inline preview has been fetched yet
var _docActiveVersionData = null; // entity JSON currently backing the Document tab (Default or a picked version)
// Snapshot of what's needed to redraw the "Version Details"/"<Type> Property"
// panel once ensureDocPillsCompat()'s async Meta fetch resolves — the panel
// may already have rendered (without the "(compat)" prefix on Compatibility
// Validated) before that fetch completes. See refreshVersionDetailsPanel().
var _docPropsPanelInfo = null;
// Set right before redirecting a standalone "meta" page (depth 5) up to its
// parent Resource page (depth 4) — see renderSingleEntity()/renderEntityGrid()
// "Meta page redirect" and setDataView()'s json-view-of-meta special case.
// Consumed once, immediately after the Resource page finishes rendering, to
// auto-select/expand the Metadata tab (List view) or box (Grid view) so the
// user lands on the same content they were viewing instead of the generic
// default (Document/Version Details tab, or a collapsed Metadata box).
var _pendingMetaTabOnLoad = false;
// Resource page (depth 4) version-selector dropdown state — stashed globally

// since onVersionSelectChange() runs from a later, independent DOM event,
// outside the renderSingleEntity() closure that built the dropdown.
var _resModel = null;          // model snapshot, for building props tables dynamically
var _resPath = null;           // _state.path snapshot (depth 4) at render time
var _resCapType = '';          // capitalized singular type, e.g. "Schema"
var _resDefaultData = null;    // the resource's own JSON — the "Default" option's data
var _resCollKeys = null;       // collKeys snapshot, to suppress <plural>url/<plural>count fields
var _resVersionsUrl = '';      // this resource's versions collection URL
var _resVersionsList = null;   // fetched versions collection items, cached
var _resSelectedVersionId = 'default'; // currently selected dropdown value
// The page-level "Entity Data Editor" action bar's own HTML (Save/Undo/
// Delete for the Default version), stashed so onVersionSelectChangeReal()/
// rerenderVersionPanel() can restore it into #dataEditorActionBarSlot when
// switching back to "Default", without needing to recompute it (its
// disabled/enabled state reflects _dataDirty at the time renderSingleEntity()
// last ran, same as before this slot existed).
var _dataEditorActionBarHtml = '';

function refresh() {
  var main = el('main-view');

  if (_state.view === 'home') {
    setLeftPanelVisible(false);
    renderHome();
    return;
  }

  if (_state.view === 'config') {
    setLeftPanelVisible(false);
    renderConfig();
    return;
  }

  // Probe registry label if not yet cached, then refresh breadcrumbs
  var svURL = normalizeURL(_state.serverURL || window.location.origin);
  if (!_labelCache[svURL]) {
    probeRegistry(svURL, function(info) {
      if (info.label) renderBreadcrumbs();
    });
  }

  var isModelSection        = (_state.section === 'model' || _state.section === 'modelsource');
  var isCapabilitiesSection = (_state.section === 'capabilities');
  var isCapOfferedSection   = (_state.section === 'capabilitiesoffered');
  var isXRegistrySection    = (_state.section === 'xregistry');
  // Grid/List's own "Filters" toggle (separate from JSON view, which always
  // shows the full left panel) — see plan.md "Filter support in Grid/List
  // views".
  var showGridFilters = isGridFiltersOnlyMode();

  setLeftPanelVisible(_state.view === 'json' || _state.dataView === 'json' || showGridFilters);
  main.innerHTML = spinner();

  // Capabilities Offered — list (read-only schema viewer) or JSON view, per
  // _state.dataView. Always read-only (server-declared schema document, not
  // user-edited) — see plan.md "Capabilities/CapabilitiesOffered List view".
  if (isCapOfferedSection) {
    var offeredURL = buildAPIURL();
    fetchJSON(offeredURL)
      .then(function(data) {
        _lastData = data;
        _state.mutable = false;
        renderHeader();
        if (_state.dataView === 'json') {
          renderJSONView(data);
        } else {
          renderCapabilitiesOfferedViewer(data);
        }
      })
      .catch(function(err) {
        main.innerHTML = '<div class="error-banner">Error loading:\n'
          + esc(offeredURL) + '\n\n' + esc(String(err)) + '</div>';
      });
    return;
  }

  // .xregistry — the small cross-registry discovery document (see
  // plan.md "known-registries-autoadd-hide") — list (simple read-only
  // viewer) or JSON view, per _state.dataView. Always read-only, like
  // Capabilities Offered — it's a server-generated document, never
  // user-edited.
  if (isXRegistrySection) {
    var xregURL = buildAPIURL();
    fetchJSON(xregURL)
      .then(function(data) {
        _lastData = data;
        _state.mutable = false;
        renderHeader();
        if (_state.dataView === 'json') {
          renderJSONView(data);
        } else {
          renderXRegistryViewer(data);
        }
      })
      .catch(function(err) {
        main.innerHTML = '<div class="error-banner">Error loading:\n'
          + esc(xregURL) + '\n\n' + esc(String(err)) + '</div>';
      });
    return;
  }

  // Capabilities — list (editor) or JSON view, per _state.dataView. Editable
  // when the doc itself reports available.capabilities.mutable === true.
  if (isCapabilitiesSection) {
    var svBaseC = (_state.serverURL || window.location.origin).replace(/\/$/, '');
    ensureCapCached(svBaseC, function(cap) {
      if (!cap) {
        main.innerHTML = '<div class="error-banner">Error loading:\n'
          + esc(svBaseC + '/capabilities') + '</div>';
        return;
      }
      var avail = cap.available;
      _state.mutable = !!(avail && avail.capabilities && avail.capabilities.mutable);
      _lastData = cap;
      renderHeader();
      if (_state.dataView === 'json') {
        renderJSONView(cap);
      } else {
        ensureOfferedCached(svBaseC, function(offered) {
          renderCapabilitiesEditor(cap, offered);
        });
      }
    });
    return;
  }

  // Model / Model Source — list (editor) or JSON view, per _state.dataView.
  // Editing is only ever enabled while on modelsource (see renderHeader()).
  if (isModelSection) {
    var modelURL = buildAPIURL();
    var svBaseM  = (_state.serverURL || window.location.origin).replace(/\/$/, '');
    ensureCapCached(svBaseM, function(cap) {
      var avail = cap && cap.available;
      // /model and /modelsource are optional, capability-gated endpoints
      // (per spec) — buildRegEndpointPillsHtml() already hides their pills
      // when unavailable, but a stale bookmark/pushState (or switching to a
      // server that doesn't expose one) can still land here directly. Once
      // capabilities are known, check availability up front instead of
      // firing the fetch and rendering a raw "Error loading" banner on 404.
      if (avail && !avail[_state.section]) {
        _lastData = null;
        renderHeader();
        main.innerHTML = '<div class="state-msg">This server does not expose a '
          + esc(_state.section === 'modelsource' ? 'model source' : 'model')
          + ' endpoint.</div>';
        return;
      }
      _state.mutable = !!(avail && avail[_state.section] && avail[_state.section].mutable);
      fetchJSON(modelURL)
        .then(function(data) {
          _lastData = data;
          renderHeader();
          if (_state.dataView === 'json') {
            renderJSONView(data);
          } else {
            renderModelEditor(data);
          }
        })
        .catch(function(err) {
          main.innerHTML = '<div class="error-banner">Error loading:\n'
            + esc(modelURL) + '\n\n' + esc(String(err)) + '</div>';
        });
    });
    return;
  }

  var apiURL = buildAPIURL();
  var coll   = isCollection(_state.path);

  // For resource/version entities we try $details first so that document-backed
  // resources return their JSON metadata rather than their document body.
  // If the server returns 400 (resource has no document), fall back to plain GET.
  var needsDet = needsDetails(_state.path);

  fetchWithDetailsFallback(apiURL, needsDet)
    .then(function(data) {
      renderEntityFromData(data, coll);
    })
    .catch(function(err) {
      main.innerHTML = '<div class="error-banner">Error loading:\n'
        + esc(apiURL) + '\n\n' + esc(String(err)) + '</div>';
    });
}

// Shared tail of the fetch-based branch in refresh().
function renderEntityFromData(data, coll) {
  _lastData = data;
  var svBaseE = (_state.serverURL || window.location.origin).replace(/\/$/, '');
  var capKeyE = normalizeURL(svBaseE);
  if (_capCache.hasOwnProperty(capKeyE)) {
    _state.mutable = entitiesMutableFromCap(_capCache[capKeyE]);
  } else {
    // Not cached yet — render this pass as non-mutable (Edit button/Add-
    // Delete controls hidden), then re-render once /capabilities resolves
    // and gets cached, same async-then-rerender pattern renderHeader()
    // already uses for the Filters/Sort toggle button (see capLoaded2).
    _state.mutable = false;
    ensureCapCached(svBaseE, function() { if (_lastData === data) renderEntityFromData(data, coll); });
  }
  // Grid view has been removed entirely for the data section — normalize
  // any stale dview=grid (old bookmark/back-forward history, from before
  // this removal) to table so the header doesn't show Grid as "active"
  // (even though its button is disabled) and the URL stays consistent.
  if (_state.section === 'data' && _state.dataView === 'grid') {
    _state.dataView = 'table';
    history.replaceState(null, '', buildURL(_state));
  }
  renderHeader();
  switch (_state.view) {
    case 'json': renderJSONViewForCurrentTab(data); break;
    default:
      if (_state.dataView === 'json') {
        renderJSONViewForCurrentTab(data);
      } else if (coll) {
        renderTableView(data);
      } else {
        renderSingleEntity(data);
      }
  }
  // Grid/List's own Filters-only left panel (independent of JSON view's
  // always-on panel) — render its content when toggled open.
  if (isGridFiltersOnlyMode()) renderJSONLeftPanel(true);
}

// Whether the "entities" capability (i.e. Registry/Group/Resource/Version
// data — as opposed to model/modelsource/capabilities/etc, each gated by
// their own matching capability key) is declared mutable by the server's
// /capabilities document. Drives both the Edit button's enabled state and
// _state.mutable for the plain 'data' section (single-entity edit mode,
// collection Add/Delete) — see plan.md "Full Data Edit Mode".
function entitiesMutableFromCap(cap) {
  var avail = cap && cap.available;
  return !!(avail && avail.entities && avail.entities.mutable);
}

// ---- Home view -----------------------------------------------------------

function renderHome() {
  var main = el('main-view');
  var origin = window.location.origin;
  var servers = loadServers().filter(function(u) { return !isHidden(u); });
  var allServers = [origin].concat(servers.filter(function(u) { return u !== origin; }));

  var g = _state.homeGroup;
  var l = currentHomeLayout(); // per-group persisted layout, independent of data pages
  if (g === 'types') {
    renderHomeFlatList(main, allServers); // Grid removed for this page — always List
  } else {
    l === 'table' ? renderHomeTable(main, allServers) : renderHomeGrid(main, allServers);
  }
}

// Manual global "Refresh" button (see #home-refresh-btn/renderHeader()) —
// forces a fresh re-probe of every registry currently listed on Home,
// bypassing _registryProbeCache entirely. Simplest correct approach: wipe
// the whole cache (Home already lists every known server, so a per-server
// selective clear wouldn't save anything meaningful here) and re-render.
// Briefly spins the icon via a CSS class for visual feedback that the
// click actually did something, even though the re-probe itself usually
// completes fast enough that the spin is mostly decorative.
function doHomeRefresh() {
  _registryProbeCache = {};
  var btn = el('home-refresh-btn');
  if (btn) {
    btn.classList.add('spinning');
    setTimeout(function() { btn.classList.remove('spinning'); }, 600);
  }
  renderHome();
}

function renderHomeGrid(main, servers) {
  var sorted = servers.slice().sort(function(a, b) {
    return serverLabel(a).toLowerCase().localeCompare(serverLabel(b).toLowerCase());
  });
  var html = '<div class="home-page"><div class="home-grid">';
  sorted.forEach(function(url) { html += serverCard(url); });
  html += '</div></div>';
  main.innerHTML = html;
  probeAllCards(main);
}

function renderHomeTable(main, servers) {
  var sorted = servers.slice().sort(function(a, b) {
    return serverLabel(a).toLowerCase().localeCompare(serverLabel(b).toLowerCase());
  });
  // Card-list design (see plan.md "List view visual redesign for Registries
  // home page") — a stack of rounded row-cards rather than a plain <table>,
  // so List reads as a denser sibling of Grid rather than a cold fallback.
  // Uses its own .reg-* classes (not the generic .home-table/.ht-* ones,
  // which stay as-is for the Home "types" flat list — see
  // renderHomeFlatList()).
  var html = '<div class="home-page"><div class="reg-list">';
  sorted.forEach(function(url) {
    var sv = (url === window.location.origin) ? '' : url;
    var href = buildURL(Object.assign({}, _state, {view: 'table', serverURL: sv, section: 'data', path: []}));
    html += '<div class="reg-row" data-server-url="' + esc(url) + '">'
      + '<img src="favicon.svg" class="reg-row-icon" alt="" width="20" height="20">'
      + '<div class="reg-row-main">'
      +   '<div class="reg-row-title">'
      +     '<a class="reg-row-name ht-name-text ht-name-link" href="' + esc(href) + '" onclick="' + esc(guardedOnclick('doBrowse(' + JSON.stringify(url) + ')')) + '">' + esc(serverLabel(url)) + '</a>'
      +     '<span class="reg-row-err-badge" style="display:none" title="Click for error details">Connection failed</span>'
      +   '</div>'
      +   '<div class="reg-row-sub"></div>'
      + '</div>'
      + '<div class="reg-row-side">'
      +   '<div class="reg-row-groups"><span class="ht-loading">…</span></div>'
      +   '<div class="reg-row-url" title="' + esc(url) + '">' + esc(url) + '</div>'
      + '</div>'
      + '<div class="server-card-err-popup" style="display:none">'
      +   '<div class="server-card-err-popup-title">Connection Error</div>'
      +   '<div class="server-card-err-popup-msg"></div>'
      +   '<button class="home-btn home-btn-secondary" style="font-size:11px;padding:2px 8px" '
      +     'onclick="this.closest(\'.server-card-err-popup\').style.display=\'none\'">Close</button>'
      + '</div>'
      + '</div>';
  });
  html += '</div></div>';
  main.innerHTML = html;

  main.querySelectorAll('[data-server-url]').forEach(function(row) {
    probeRegistry(row.dataset.serverUrl, function(info) {
      var nameEl   = row.querySelector('.ht-name-text');
      var subEl    = row.querySelector('.reg-row-sub');
      var groupsEl = row.querySelector('.reg-row-groups');
      if (info.error) {
        // disable the name link, mark the row as errored, and wire up the
        // "Connection failed" badge to toggle the existing error popup.
        if (nameEl) { nameEl.classList.remove('ht-name-link'); nameEl.removeAttribute('onclick'); }
        row.classList.add('reg-row-error');
        var badge = row.querySelector('.reg-row-err-badge');
        if (badge) {
          badge.style.display = '';
          badge.addEventListener('click', function(e) {
            e.stopPropagation();
            var popup = row.querySelector('.server-card-err-popup');
            if (!popup) return;
            var showing = popup.style.display !== 'none';
            // close all open error popups first
            qsa('.server-card-err-popup').forEach(function(p) { p.style.display = 'none'; });
            if (!showing) {
              popup.style.display = '';
              popup.querySelector('.server-card-err-popup-msg').textContent = info.error;
            }
          });
        }
        if (groupsEl) groupsEl.textContent = '';
      } else {
        if (nameEl && info.label && !getNameOverride(row.dataset.serverUrl)) nameEl.textContent = info.label;
        if (info.icon) {
          var iconEl = row.querySelector('.reg-row-icon');
          if (iconEl) iconEl.src = info.icon;
        }
        if (subEl && info.description) {
          subEl.textContent = info.description;
          subEl.style.display = '-webkit-box';
        }
        if (groupsEl) {
          groupsEl.innerHTML = info.colls.length
            ? info.colls.map(function(c) {
                return groupTypePillHTML(row.dataset.serverUrl, c);
              }).join('')
            : '<span class="group-type-none">none</span>';
        }
      }
      sortServerElements(row.closest('.reg-list'));
    });
  });
}

// renderHomeFlatGrid() (Grid view for the Home 'types' cross-registry Group
// Types page) has been removed — renderHomeFlatList() already shows the
// same fields (name/link, description, item count, resource types,
// registry) in a more scannable table. See plan.md "Grid view removed".

function browseGroupCollection(serverUrl, collName, url) {
  var sv = (serverUrl === window.location.origin) ? '' : serverUrl;
  pushState({view: 'table', serverURL: sv, section: 'data', path: [collName], apiURL: url || ''});
}

// Renders a single "group-type-item" pill as a clickable link to that
// collection (e.g. clicking "dirs (3)" browses straight to /dirs on that
// server) — shared by the List-view rows, Grid-view cards, and the Home
// "Group Types" flat list's own name-link (see plan.md "Group-type pills
// link to their collections").
function groupTypePillHTML(serverUrl, c) {
  var onclick = guardedOnclick('browseGroupCollection(' + JSON.stringify(serverUrl) + ',' + JSON.stringify(c.plural) + ',' + JSON.stringify(c.url) + ')');
  var sv = (serverUrl === window.location.origin) ? '' : serverUrl;
  var href = buildURL(Object.assign({}, _state, {view: 'table', serverURL: sv, section: 'data', path: [c.plural], apiURL: c.url || ''}));
  // Hover help shows the Group Type's model description, if any, so users
  // don't have to leave the page to learn what a pill (e.g. "dirs") means.
  var titleAttr = c.description ? ' title="' + esc(c.description) + '"' : '';
  return '<a class="group-type-item" href="' + esc(href) + '" onclick="' + esc(onclick) + '"' + titleAttr + '>' + esc(c.plural) + ' (' + c.count + ')</a>';
}

function renderHomeFlatList(main, servers) {
  // Card-list design mirroring the Registries List redesign (see plan.md
  // "List view visual redesign for Registries home page"). Each row here
  // is NOT a merged/deduplicated "group type" — it's one specific
  // group-type-as-it-exists-in-one-registry (no cross-registry merging
  // is done, since like-named group types on different registries could
  // have entirely different model definitions). So each row genuinely
  // has its own owning-registry identity, same as a Registries List row
  // has its own server identity — hence the registry's icon/name is
  // shown as the row's "owner", replacing the old plain URL column.
  main.innerHTML = '<div class="home-page"><div class="gt-list" id="flat-list-body">'
    + '<div class="gt-row-loading" style="color:#aaa;font-size:13px">Loading…</div>'
    + '</div></div>';

  var pending = servers.length;
  var allRows = [];

  function finish() {
    allRows.sort(function(a, b) {
      var n = a.plural.localeCompare(b.plural);
      return n !== 0 ? n : a.regLabel.localeCompare(b.regLabel);
    });
    var container = el('flat-list-body');
    if (!container) return;
    if (allRows.length === 0) {
      container.innerHTML = '<div class="gt-row-loading" style="font-style:italic">No group types found</div>';
      return;
    }
    container.innerHTML = allRows.map(function(r) {
      var onclick = guardedOnclick('browseGroupCollection(' + JSON.stringify(r.serverUrl) + ',' + JSON.stringify(r.plural) + ',' + JSON.stringify(r.url) + ')');
      var sv = (r.serverUrl === window.location.origin) ? '' : r.serverUrl;
      var href = buildURL(Object.assign({}, _state, {view: 'table', serverURL: sv, section: 'data', path: [r.plural], apiURL: r.url || ''}));
      var regHref = buildURL(Object.assign({}, _state, {view: 'table', serverURL: sv, section: 'data', path: []}));
      var regOnclick = guardedOnclick('doBrowse(' + JSON.stringify(r.serverUrl) + ')');
      return '<div class="gt-row">'
        + '<img src="' + esc(r.icon || r.regIcon || 'favicon.svg') + '" class="gt-row-icon" alt="" width="20" height="20" onerror="this.onerror=null;this.src=\'favicon.svg\'">'
        + '<div class="gt-row-main">'
        +   '<div class="gt-row-title">'
        +     '<a class="gt-row-name" href="' + esc(href) + '" onclick="' + esc(onclick) + '">' + esc(r.plural) + '</a>'
        +     '<span class="gt-row-count">' + r.count + (r.count === 1 ? ' item' : ' items') + '</span>'
        +   '</div>'
        +   (r.description
              ? '<div class="gt-row-sub">' + esc(r.description) + '</div>'
              : '')
        + '</div>'
        + '<div class="gt-row-side">'
        +   '<div class="gt-row-resources">'
        +     (r.resources.length
                  ? r.resources.map(function(res) {
                      var titleAttr = res.description ? ' title="' + esc(res.description) + '"' : '';
                      return '<span class="group-type-item"' + titleAttr + '>' + iconThumbHtml(res.icon, 'row-icon-thumb') + esc(res.plural) + '</span>';
                    }).join('')
                  : '<span class="group-type-none">none</span>')
        +   '</div>'
        +   '<a class="gt-row-registry" href="' + esc(regHref) + '" onclick="' + esc(regOnclick) + '" title="' + esc(r.serverUrl) + '">'
        +     '<span class="gt-row-reg-name">' + esc(r.regLabel) + '</span>'
        +   '</a>'
        + '</div>'
        + '</div>';
    }).join('');
  }

  if (pending === 0) { finish(); return; }

  servers.forEach(function(url) {
    probeRegistry(url, function(info) {
      if (!info.error) {
        var label = info.label || serverLabel(url);
        info.colls.forEach(function(c) {
          allRows.push({plural: c.plural, count: c.count, resources: c.resources || [],
                        description: c.description || '', serverUrl: url, regLabel: label,
                        regIcon: info.icon || '', icon: c.icon || '', url: c.url});
        });
      }
      pending--;
      if (pending === 0) finish();
    });
  });
}


function sortServerElements(container) {
  if (!container) return;
  var els = Array.prototype.slice.call(container.querySelectorAll('[data-server-url]'));
  els.sort(function(a, b) {
    var la = (a.querySelector('.server-card-name, .ht-name-text') || a).textContent.trim().toLowerCase();
    var lb = (b.querySelector('.server-card-name, .ht-name-text') || b).textContent.trim().toLowerCase();
    return la.localeCompare(lb);
  });
  els.forEach(function(el) { container.appendChild(el); });
}

function probeAllCards(main) {
  var container = main.querySelector('.home-grid, tbody');
  main.querySelectorAll('[data-server-url]').forEach(function(card) {
    probeRegistry(card.dataset.serverUrl, function(info) {
      var nameEl   = card.querySelector('.server-card-name');
      var groupsEl = card.querySelector('.server-card-groups');
      if (info.error) {
        var badge = document.createElement('span');
        badge.className = 'server-card-err-badge';
        badge.textContent = '!';
        var titleEl = card.querySelector('.server-card-title');
        if (titleEl) titleEl.appendChild(badge);
        card.style.cursor = 'not-allowed';
        card.style.opacity = '0.75';
        var bodyEl = card.querySelector('.server-card-body');
        if (bodyEl) {
          bodyEl.querySelector('.server-card-groups-label').style.display = 'none';
          bodyEl.querySelector('.server-card-groups').style.display = 'none';
          var errText = bodyEl.querySelector('.server-card-err-text');
          if (errText) { errText.textContent = info.error; errText.style.display = ''; }
        }
      } else {
        if (nameEl && info.label && !getNameOverride(card.dataset.serverUrl)) nameEl.textContent = info.label;
        if (info.icon) {
          var iconEl = card.querySelector('.server-card-icon');
          if (iconEl) iconEl.src = info.icon;
        }
        if (info.description) {
          var descEl = card.querySelector('.server-card-desc');
          if (descEl) {
            var descText = info.description.length > 150 ? info.description.slice(0, 150) + '…' : info.description;
            descEl.textContent = descText;
            descEl.style.display = '';
          }
        }
        if (groupsEl) {
          groupsEl.innerHTML = info.colls.length
            ? info.colls.map(function(c) {
                return groupTypePillHTML(card.dataset.serverUrl, c);
              }).join('')
            : '<span class="group-type-none">none</span>';
        }
      }
      sortServerElements(container);
    });
  });
}

function serverCard(url) {
  var sv = (url === window.location.origin) ? '' : url;
  var href = buildURL(Object.assign({}, _state, {view: 'table', serverURL: sv, section: 'data', path: []}));
  return '<div class="server-card" data-server-url="' + esc(url) + '">'
    + '<div class="server-card-title">'
    +   '<img src="favicon.svg" class="server-card-icon" alt="" width="16" height="16">'
    +   '<a class="server-card-name" href="' + esc(href) + '" onclick="return serverCardClick(event,this.closest(\'.server-card\'),' + esc(JSON.stringify(url)) + ')">' + esc(serverLabel(url)) + '</a>'
    + '</div>'
    + '<div class="server-card-desc" style="display:none"></div>'
    + '<hr class="server-card-divider">'
    + '<div class="server-card-body">'
    +   '<div class="server-card-groups-label">Group Types</div>'
    +   '<div class="server-card-groups">Connecting…</div>'
    +   '<div class="server-card-err-text" style="display:none"></div>'
    + '</div>'
    + '<div class="server-card-url">' + esc(url) + '</div>'
    + '</div>';
}

// Home-page registry-probing timeout, in ms. Without this, an unresponsive
// host (e.g. a TCP connect that never gets ACKed) leaves probeRegistry()'s
// fetch() promises pending forever, so the Home-page card/row stays stuck
// on "Connecting…" indefinitely instead of showing an error. See plan.md
// "Fix: probeRegistry() hangs forever on an unresponsive host".
var PROBE_FETCH_TIMEOUT_MS = 8000;

// fetch() wrapper that aborts (via AbortController) and rejects with a
// distinguishable "Connection timed out" error if the request doesn't
// settle within `ms` — used only for Home-page registry probing so a slow
// data fetch elsewhere in the app isn't affected.
function fetchWithTimeout(url, ms) {
  var controller = new AbortController();
  var timer = setTimeout(function() { controller.abort(); }, ms);
  return fetch(url, {signal: controller.signal})
    .catch(function(err) {
      if (err && err.name === 'AbortError') throw new Error('Connection timed out');
      throw err;
    })
    .finally(function() { clearTimeout(timer); });
}

// In-memory cache of probeRegistry() results, keyed by normalizeURL(url) —
// avoids re-fetching /capabilities + /model + / on every Home-page render
// (e.g. just navigating back to Home, or switching Grid<->List layout),
// which previously re-probed every listed registry from scratch every
// time. Being a plain JS variable (not localStorage), it's naturally
// cleared by a real browser reload — so "just returning to Home" reuses
// cached data, while an actual page refresh still re-probes everything,
// with no special detection logic needed. Invalidated explicitly after
// in-app mutations (see invalidateRegistryProbe()) and by the manual
// Home-page refresh button (see doHomeRefresh()).
var _registryProbeCache = {};

// Deletes the cached probe result (if any) for a server, forcing the next
// probeRegistry() call for it to re-fetch. Called after any in-app
// mutation that could change what Home shows for that registry (group/
// resource create/delete, model save, capabilities save) — see call
// sites in createEntity's PUT success handler, collDeleteSelected(),
// deleteDataEntity(), saveModel(), and saveCapabilities().
function invalidateRegistryProbe(url) {
  delete _registryProbeCache[normalizeURL(url)];
}

function probeRegistry(url, cb, force) {
  var normUrl = normalizeURL(url);
  if (!force && _registryProbeCache[normUrl]) {
    cb(_registryProbeCache[normUrl]);
    return;
  }
  var fetchUrl = serverFetchBase(url);
  // Fetch capabilities first; use it to decide what else to fetch.
  // Per spec: if /capabilities is 404, default available = {entities:{mutable:true}}
  var capP = fetchWithTimeout(fetchUrl + '/capabilities', PROBE_FETCH_TIMEOUT_MS)
    .then(function(r) { return r.ok ? r.json() : null; })
    .catch(function() { return null; });

  capP.then(function(cap) {
    var available = (cap && cap.available) || {entities: {mutable: true}};
    // Populate capabilities cache so JSON view left panel gets it for free
    _capCache[normUrl] = cap || {available: available, flags: []};

    // Only fetch model if capabilities says it's available
    var modelP = available.model
      ? fetchWithTimeout(fetchUrl + '/model', PROBE_FETCH_TIMEOUT_MS).then(function(r) { return r.json(); }).catch(function() { return null; })
      : Promise.resolve(null);

    Promise.all([
      fetchWithTimeout(fetchUrl + '/', PROBE_FETCH_TIMEOUT_MS).then(function(r) {
        if (!r.ok) return r.text().then(function(t) { throw new Error('HTTP ' + r.status + ' — ' + t.slice(0, 300)); });
        return r.json();
      }),
      modelP
    ])
      .then(function(results) {
        var data  = results[0];
        var model = results[1];
        if (!data.specversion || !data.registryid) {
          cb({label: '', colls: [], icon: '', available: available, error: 'Not a valid xRegistry (missing specversion or registryid)'});
          return;
        }
        var label = data.registryid || '';
        if (label) { _labelCache[normUrl] = label; saveLabelCache(); }
        var colls = findCollectionRefs(model, [], data);
        colls.forEach(function(c) {
          var grpDef = model && model.groups && model.groups[c.plural];
          // Each resource type carries its own model description too, so
          // the Home "Group Types" page can show it as hover help on the
          // resource-type pill (see plan.md "Group Types page: resource
          // pill hover help"). Also carries its model "icon" (if any) — see
          // plan.md "Icon propagation from model + entity data" — shown as
          // a thumbnail next to the Group Type name/resource-type pill.
          c.resources = grpDef && grpDef.resources
            ? Object.keys(grpDef.resources).sort().map(function(rp) {
                return {plural: rp, description: (grpDef.resources[rp] && grpDef.resources[rp].description) || '',
                        icon: (grpDef.resources[rp] && grpDef.resources[rp].icon) || ''};
              })
            : [];
          c.description = (grpDef && grpDef.description) || '';
          c.icon        = (grpDef && grpDef.icon) || '';
        });
        var info = {label: label, colls: colls, icon: data.icon || '', description: data.description || '', available: available, error: null};
        _registryProbeCache[normUrl] = info; // only cache successes — transient errors (e.g. a momentarily unreachable host) should retry on next visit, not stick around
        cb(info);
      })
      .catch(function(err) { cb({label: '', colls: [], icon: '', available: available, error: (err && err.message) ? err.message : String(err)}); });
  });
}

function doRemoveServer(url) {
  removeServer(url);
  refresh();
}

function doBrowse(url) {
  var sv = (url === window.location.origin) ? '' : url;
  pushState({view: 'table', serverURL: sv, section: 'data', path: []});
}

// Returns true (let the browser perform its native <a> action) for any
// ctrl/cmd/shift/middle-click gesture, even on an error-state card — by
// design, modifier-click always honors the real href and skips app-level
// gates like the error check below, matching how plain links behave.
// Otherwise: false when the card is in an error state (suppresses default
// navigation for a normal left-click), true after routing a normal click
// through the fast SPA doBrowse() navigation.
function serverCardClick(e, card, url) {
  if (navShouldDefault(e)) return true;
  if (card.querySelector('.server-card-err-badge')) return false;
  doBrowse(url);
  return false;
}

// ---- Config page ----------------------------------------------------------

// Builds one radio-button <label> for the "JSON coloring" tri-state
// option group on the Config page.
function cfgJsonColorRadio(mode, label, desc) {
  return '<label class="cfg-radio-row" title="' + esc(desc) + '">'
    + '<input type="radio" name="opt-json-color" value="' + mode + '"'
    + (optJsonColorMode() === mode ? ' checked' : '')
    + ' onchange="cfgSetJsonColor(\'' + mode + '\')">'
    + '<span class="cfg-radio-label">' + esc(label) + '</span>'
    + '</label>';
}

// Radio option for the "Default View" setting — mirrors cfgJsonColorRadio()
// above. mode is 'hide' (optXregFocused() === false, today's default) or
// 'show' (optXregFocused() === true).
function cfgXregRadio(mode, label, desc) {
  var checked = (mode === 'show') === optXregFocused();
  return '<label class="cfg-radio-row" title="' + esc(desc) + '">'
    + '<input type="radio" name="opt-xreg-focused" value="' + mode + '"'
    + (checked ? ' checked' : '')
    + ' onchange="cfgSetXregFocused(\'' + mode + '\' === \'show\')">'
    + '<span class="cfg-radio-label">' + esc(label) + '</span>'
    + '</label>';
}

// Sort state for the Config page's Registry Servers table — persisted only
// in-memory (resets to the default Name/ascending on reload), toggled by
// clicking a sortable column header (see cfgSortBy()/cfgSortHeaderHTML()).
// The local server's row is always pinned first regardless of sort column/
// direction (it's "this server", not a regular addable/removable entry).
var _cfgSortCol = 'name';
var _cfgSortDir = 'asc';

// Builds one sortable <th>, wiring up the click handler and rendering the
// ▲/▼ arrow when this is the currently active sort column.
function cfgSortHeaderHTML(col, label, extraAttrs) {
  var arrow = '';
  if (_cfgSortCol === col) {
    arrow = '<span class="cfg-sort-arrow">' + (_cfgSortDir === 'asc' ? '\u25B2' : '\u25BC') + '</span>';
  }
  return '<th class="cfg-sortable"' + (extraAttrs || '') + ' onclick="cfgSortBy(\'' + col + '\')">'
    + esc(label) + arrow + '</th>';
}

// Handles a click on a Registry Servers column header — toggles direction
// if the same column is clicked again, otherwise switches to the new
// column defaulting to ascending.
function cfgSortBy(col) {
  if (_cfgSortCol === col) {
    _cfgSortDir = (_cfgSortDir === 'asc') ? 'desc' : 'asc';
  } else {
    _cfgSortCol = col;
    _cfgSortDir = 'asc';
  }
  renderConfig();
}

// Returns the sort-key value for one server URL/column, used by
// cfgSortedServerUrls() below.
function cfgSortKeyFor(url, col) {
  switch (col) {
    case 'location': return url.toLowerCase();
    case 'proxy':     return isProxied(url) ? 1 : 0;
    case 'hide':      return isHidden(url) ? 1 : 0;
    case 'name':
    default:          return serverLabel(url).toLowerCase();
  }
}

// Sorts the (non-local) server URL list per the current
// _cfgSortCol/_cfgSortDir, with a stable Name tie-breaker so equal-value
// rows (e.g. two servers both un-proxied) don't visibly reshuffle between
// renders.
function cfgSortedServerUrls(urls) {
  var col = _cfgSortCol, dir = _cfgSortDir;
  return urls.slice().sort(function(a, b) {
    var ka = cfgSortKeyFor(a, col), kb = cfgSortKeyFor(b, col);
    var cmp;
    if (typeof ka === 'number' && typeof kb === 'number') cmp = ka - kb;
    else cmp = String(ka).localeCompare(String(kb));
    if (cmp === 0) {
      cmp = serverLabel(a).toLowerCase().localeCompare(serverLabel(b).toLowerCase());
    }
    return dir === 'desc' ? -cmp : cmp;
  });
}

function renderConfig() {
  var main   = el('main-view');
  var origin = window.location.origin;
  var servers = loadServers();

  var html = '<div class="config-page"><div class="config-section">'
    + '<div class="config-section-header">'
    +   '<h3 class="config-section-title">Registry Servers</h3>'
    +   '<div class="config-section-header-btns">'
    +     '<button class="cfg-btn cfg-btn-danger" id="cfg-delete-selected-btn" disabled '
    +       'onclick="cfgDeleteSelected()" title="Delete all checked servers'
    +       ' below">Delete Selected</button>'
    +     '<button class="cfg-btn" id="cfg-scan-selected-btn" disabled '
    +       'onclick="cfgScanSelected()" title="Check all checked servers'
    +       ' below (and/or this server) for other registries they know'
    +       ' about, and review the results before adding any">Scan for'
    +       ' registries</button>'
    +   '</div>'
    + '</div>'
    + '<table class="config-table"><thead><tr>'
    + '<th class="cfg-select-cell"><input type="checkbox" id="cfg-select-all" '
    +   'title="Select/deselect all servers" '
    +   'onchange="cfgToggleSelectAll(this)"></th>'
    + cfgSortHeaderHTML('name', 'Name')
    + cfgSortHeaderHTML('location', 'Location')
    + cfgSortHeaderHTML('proxy', 'Proxy')
    + cfgSortHeaderHTML('hide', 'Hide', ' title="Hide from the Home page (without deleting)"')
    + '<th></th></tr></thead><tbody>';

  // Local server — its URL is fixed (can't ever be a different server),
  // but its display Name can still be overridden and edited like any
  // other registry, via its own Edit/Save/Cancel (no Delete/URL editing).
  // Proxying only ever makes sense for a *remote* server, so this row has
  // no checkbox at all (blank cell) rather than a disabled/checked one.
  // Its select checkbox is still real (usable as a scan source, or to
  // hide it), even though it can't be deleted.
  html += '<tr data-cfg-url="' + esc(normalizeURL(origin)) + '" '
    + 'data-cfg-name="' + esc(getNameOverride(normalizeURL(origin))) + '">'
    + '<td class="cfg-select-cell">'
    +   '<input type="checkbox" class="cfg-select-input" '
    +     'onchange="cfgUpdateSelection()">'
    + '</td>'
    + '<td class="cfg-name">' + cfgNameCellHTML(normalizeURL(origin)) + '</td>'
    + '<td><span class="cfg-url-display">' + esc(origin)
    + ' <span class="config-local-badge">this server</span></span></td>'
    + '<td></td>'
    + '<td class="cfg-hide-cell">'
    +   '<input type="checkbox" class="cfg-hide-input"'
    +     (isHidden(normalizeURL(origin)) ? ' checked' : '') + ' '
    +     'onchange="cfgSetHidden(\'' + esc(normalizeURL(origin)) + '\', this.checked)">'
    + '</td>'
    + '<td class="cfg-actions">'
    +   '<button class="cfg-btn cfg-edit" onclick="cfgEdit(this)">Edit</button>'
    +   '<button class="cfg-btn cfg-btn-primary cfg-save" style="display:none" onclick="cfgSave(this)">Save</button>'
    +   '<button class="cfg-btn cfg-btn-cancel cfg-cancel" style="display:none" onclick="cfgCancel(this)">Cancel</button>'
    + '</td></tr>';

  // User-added servers — sorted per the current column/direction chosen
  // via the sortable column headers (see cfgSortBy()/cfgSortedServerUrls()),
  // defaulting to Name/ascending (same serverLabel()-based ordering used
  // on the Home page's Registries list) for a deterministic initial order.
  cfgSortedServerUrls(servers.filter(function(u) { return u !== origin; })).forEach(function(url) {
    var discFrom = getDiscoveredFrom(url);
    html += '<tr data-cfg-url="' + esc(url) + '" '
      + 'data-cfg-name="' + esc(getNameOverride(normalizeURL(url))) + '">'
      + '<td class="cfg-select-cell">'
      +   '<input type="checkbox" class="cfg-select-input" '
      +     'onchange="cfgUpdateSelection()">'
      + '</td>'
      + '<td class="cfg-name">' + cfgNameCellHTML(url) + '</td>'
      + '<td>'
      +   '<span class="cfg-url-display"' + (discFrom ? ' title="Discovered via '
            + esc(discFrom) + '"' : '') + '>' + esc(url) + '</span>'
      +   '<input class="cfg-url-input" style="display:none" value="' + esc(url) + '" '
      +     'onkeydown="if(event.key===\'Enter\')cfgSave(this);'
      +               'else if(event.key===\'Escape\')cfgCancel(this)">'
      + '</td>'
      + '<td class="cfg-proxy-cell" title="Route requests to this server'
      +   ' through this server\u2019s own proxy - use this if the remote'
      +   ' registry doesn\u2019t allow direct cross-origin access (CORS)">'
      +   '<input type="checkbox" class="cfg-proxy-input"'
      +     (isProxied(url) ? ' checked' : '') + ' '
      +     'onchange="cfgSetProxied(\'' + esc(url) + '\', this.checked)">'
      + '</td>'
      + '<td class="cfg-hide-cell" title="Hide from the Home page (without'
      +   ' deleting)">'
      +   '<input type="checkbox" class="cfg-hide-input"'
      +     (isHidden(url) ? ' checked' : '') + ' '
      +     'onchange="cfgSetHidden(\'' + esc(url) + '\', this.checked)">'
      + '</td>'
      + '<td class="cfg-actions">'
      +   '<button class="cfg-btn cfg-edit" onclick="cfgEdit(this)">Edit</button>'
      +   '<button class="cfg-btn cfg-btn-primary cfg-save" style="display:none" onclick="cfgSave(this)">Save</button>'
      +   '<button class="cfg-btn cfg-btn-cancel cfg-cancel" style="display:none" onclick="cfgCancel(this)">Cancel</button>'
      + '</td></tr>';
  });

  html += '</tbody><tfoot><tr class="cfg-add-tr">'
    + '<td class="cfg-select-cell"></td>'
    + '<td>'
    +   '<input type="text" id="cfg-new-name" class="cfg-name-input" '
    +     'placeholder="Name (optional)" '
    +     'onkeydown="if(event.key===\'Enter\')cfgAddNew()">'
    + '</td>'
    + '<td>'
    +   '<input type="text" id="cfg-new-url" class="cfg-url-input" '
    +     'placeholder="http://example.com" '
    +     'onkeydown="if(event.key===\'Enter\')cfgAddNew()">'
    +   '<span id="cfg-new-error" class="cfg-inline-error" style="display:none"></span>'
    + '</td>'
    + '<td class="cfg-proxy-cell" title="Route requests to this server'
    +   ' through this server\u2019s own proxy - use this if the remote'
    +   ' registry doesn\u2019t allow direct cross-origin access (CORS)">'
    +   '<input type="checkbox" id="cfg-new-proxy">'
    + '</td>'
    + '<td class="cfg-hide-cell"></td>'
    + '<td class="cfg-actions">'
    +   '<button class="cfg-btn cfg-btn-primary" onclick="cfgAddNew()">Add</button>'
    + '</td>'
    + '</tr></tfoot></table>'
    + '</div>'

    // ---- Options section ----
    + '<div class="config-section">'
    + '<h3 class="config-section-title">Options</h3>'
    // Options below are alphabetized by label (Default View, then
    // JSON coloring) — keep new options inserted in alphabetical
    // order too.
    + '<div class="cfg-option-row cfg-option-group"'
    +   ' title="Controls whether xRegistry-specific Property-table sections'
    +   ' (Identity, Versioning &amp; State) and the Config: links are shown'
    +   ' by default, in addition to your own data. View mode only \u2014'
    +   ' edit mode and JSON view always show everything. Can be overridden'
    +   ' temporarily for the current page via the menu\u2019s Show/Hide xReg'
    +   ' Data entry.">'
    +   '<span class="cfg-option-label">Default View</span>'
    +   '<div class="cfg-radio-set">'
    +     cfgXregRadio('hide', 'Hide xReg data',
          'Show only your own data \u2014 today\u2019s default')
    +     cfgXregRadio('show', 'Show xReg data',
          'Also show xRegistry-specific details (Identity, Versioning'
          + ' &amp; State, Config links) \u2014 the full xRegistry attribute set')
    +   '</div>'
    +   '<span class="cfg-option-desc">Show xRegistry-specific details'
    +   ' (Identity, Versioning &amp; State, Config links) in view mode,'
    +   ' in addition to your own data</span>'
    + '</div>'

    + '<div class="cfg-option-row cfg-option-group">'
    +   '<span class="cfg-option-label">JSON coloring</span>'
    +   '<div class="cfg-radio-set">'
    +     cfgJsonColorRadio('full', 'Full color',
          'Keys, strings, numbers, booleans and links are all colored'
          + ' (today\u2019s default)')
    +     cfgJsonColorRadio('minimal', 'Minimal color',
          'Everything is black except links')
    +     cfgJsonColorRadio('none', 'No color',
          'Everything is black, including links')
    +   '</div>'
    +   '<span class="cfg-option-desc">Choose how much syntax coloring'
    +   ' the JSON view uses</span>'
    + '</div>'

    + '</div>'

    // ---- Reset section ----
    + '<div class="config-section">'
    + '<h3 class="config-section-title">Reset</h3>'
    + '<p class="config-section-desc">If something looks wrong, you can'
    +   ' clear the browser-side data this app keeps (saved registry'
    +   ' locations and/or your option preferences above) and start fresh.'
    +   ' This does not change anything on any registry server.</p>'
    + '<div class="cfg-reset-row">'
    +   '<button class="cfg-btn cfg-btn-danger" onclick="cfgResetAll()">Clear All</button>'
    +   '<button class="cfg-btn" onclick="cfgResetExceptServers()">Clear All Except Registry Locations</button>'
    + '</div>'
    + '</div>';
  main.innerHTML = html;

  // Probe all servers; mark any that error with the same ! badge + popup as the home page
  var allUrls = [origin].concat(servers.filter(function(u) { return u !== origin; }));
  allUrls.forEach(function(url) {
    var norm = normalizeURL(url);
    probeRegistry(url, function(info) {
      var tr = main.querySelector('tr[data-cfg-url="' + norm + '"]');
      if (tr && info.label) {
        var nameInput = tr.querySelector('.cfg-name-input');
        // Only fill in the probed name as a placeholder — never overwrite
        // an override the user has set, or a name currently being edited.
        if (nameInput) nameInput.placeholder = info.label;
        var nameDisplay = tr.querySelector('.cfg-name-display');
        if (nameDisplay && !getNameOverride(norm)
            && (!nameInput || nameInput.style.display === 'none')) {
          nameDisplay.textContent = info.label;
        }
      }
      if (!info.error) return;
      var disp = tr && tr.querySelector('.cfg-url-display');
      if (!disp || disp.querySelector('.server-card-err-badge')) return;

      // Badge
      var badge = document.createElement('span');
      badge.className = 'server-card-err-badge';
      badge.textContent = '!';
      badge.title = 'Click for error details';
      disp.appendChild(badge);

      // Popup (same structure as home page cards)
      var popup = document.createElement('div');
      popup.className = 'server-card-err-popup';
      popup.style.display = 'none';
      popup.innerHTML = '<div class="server-card-err-popup-title">Connection Error</div>'
        + '<div class="server-card-err-popup-msg">' + esc(info.error) + '</div>'
        + '<button class="cfg-btn" style="align-self:flex-end"'
        +   ' onclick="this.closest(\'.server-card-err-popup\').style.display=\'none\'">Close</button>';
      disp.style.position = 'relative';
      disp.appendChild(popup);

      badge.addEventListener('click', function(e) {
        e.stopPropagation();
        var showing = popup.style.display !== 'none';
        qsa('.server-card-err-popup').forEach(function(p) { p.style.display = 'none'; });
        if (!showing) popup.style.display = '';
      });
    });
  });
}

// Builds the Name cell's display+input pair for a Config-page server row,
// mirroring the existing display/input pattern used for the URL column.
// Shows the current override (if any), else the probed server name — kept
// in sync with the always-visible display span. The input itself only
// becomes visible (and editable) once the row's Edit button is clicked; see
// cfgEdit()/cfgSave()/cfgCancel().
function cfgNameCellHTML(url) {
  var norm     = normalizeURL(url);
  var override = getNameOverride(norm);
  var probed   = _labelCache[norm] || '';
  var shown    = override || probed || '\u2014';
  return '<span class="cfg-name-display">' + esc(shown) + '</span>'
    + '<input class="cfg-name-input" style="display:none" value="' + esc(override) + '" '
    +   'placeholder="' + esc(probed) + '" '
    +   'onkeydown="if(event.key===\'Enter\')cfgSave(this);'
    +             'else if(event.key===\'Escape\')cfgCancel(this)">';
}

// Reveals whichever of {Name, URL} editable inputs exist in this row (the
// local "this server" row only has a Name input; other rows have both) and
// swaps the Edit button for Save/Cancel.
function cfgEdit(btn) {
  var tr = btn.closest('tr');
  tr.querySelectorAll('.cfg-name-display, .cfg-url-display').forEach(function(e) { e.style.display = 'none'; });
  tr.querySelectorAll('.cfg-name-input, .cfg-url-input').forEach(function(e) { e.style.display = ''; });
  tr.querySelector('.cfg-edit').style.display   = 'none';
  tr.querySelector('.cfg-save').style.display   = '';
  tr.querySelector('.cfg-cancel').style.display = '';
  var inp = tr.querySelector('.cfg-name-input') || tr.querySelector('.cfg-url-input');
  if (inp) { inp.focus(); inp.select(); }
}

function cfgCancel(el) {
  var tr      = el.closest('tr');
  var nameInp = tr.querySelector('.cfg-name-input');
  var urlInp  = tr.querySelector('.cfg-url-input');
  if (nameInp) nameInp.value = tr.dataset.cfgName || '';
  if (urlInp)  urlInp.value  = tr.dataset.cfgUrl;
  tr.querySelectorAll('.cfg-name-display, .cfg-url-display').forEach(function(e) { e.style.display = ''; });
  tr.querySelectorAll('.cfg-name-input, .cfg-url-input').forEach(function(e) { e.style.display = 'none'; });
  tr.querySelector('.cfg-edit').style.display   = '';
  tr.querySelector('.cfg-save').style.display   = 'none';
  tr.querySelector('.cfg-cancel').style.display = 'none';
}

function cfgSave(el) {
  var tr      = el.closest('tr');
  var oldUrl  = tr.dataset.cfgUrl;
  var urlInp  = tr.querySelector('.cfg-url-input');
  var nameInp = tr.querySelector('.cfg-name-input');
  var newUrl  = urlInp ? normalizeURL(urlInp.value) : oldUrl;
  if (urlInp && !newUrl) return;

  if (urlInp && newUrl !== oldUrl) {
    // Renaming to a URL that's already configured elsewhere is blocked,
    // same as adding a duplicate via the Add row — a URL is a server's
    // unique identity (see plan.md's duplicate-URL discussion).
    if (loadServers().includes(newUrl)) {
      alert('Already configured — delete the existing entry for that URL'
        + ' first if you want to merge settings into it.');
      return;
    }
    // Capture the old URL's flags BEFORE removeServer() wipes them (it
    // now clears all per-URL state on delete — see removeServer()) so
    // they can be carried over to the renamed URL below instead of lost.
    var oldProxied  = isProxied(oldUrl);
    var oldHidden   = isHidden(oldUrl);
    var oldDiscFrom = getDiscoveredFrom(oldUrl);
    removeServer(oldUrl);
    addServer(newUrl);
    // Carry every per-URL flag over to the renamed URL — each is keyed by
    // URL (like the name override above), so a rename would otherwise
    // silently lose it. discoveredFrom is provenance (set once, "first
    // seen wins" — see setDiscoveredFrom()), so it's only copied when the
    // renamed URL doesn't already have one of its own.
    setProxied(newUrl, oldProxied);
    setHidden(newUrl, oldHidden);
    if (oldDiscFrom) setDiscoveredFrom(newUrl, oldDiscFrom);
  }
  if (nameInp) setNameOverride(newUrl, nameInp.value.trim());
  renderConfig();
}

// Live-toggled straight from the Config page's per-server checkbox — takes
// effect immediately, no Edit/Save/Cancel step needed (see cfgSave() above
// for how the flag survives a URL rename via the Edit flow instead).
function cfgSetProxied(url, on) {
  setProxied(url, on);
}

function cfgSetHidden(url, on) {
  setHidden(url, on);
}

// Multi-select bulk actions (Delete / Scan for registries) ----------------

// Header "select all" checkbox: check/uncheck every row's checkbox
// (including the local "this server" row, which is now selectable too —
// it can be used as a scan source or hidden, even though it can't be
// deleted), then refresh the bulk-action buttons' enabled/count state.
function cfgToggleSelectAll(cb) {
  document.querySelectorAll('.cfg-select-input').forEach(function(inp) {
    inp.checked = cb.checked;
  });
  cfgUpdateSelection();
}

// Called on every row checkbox change: updates the Delete Selected/Scan
// for registries buttons' label/enabled state and keeps the header
// "select all" checkbox in sync (checked/indeterminate/unchecked) with
// the rows.
function cfgUpdateSelection() {
  var boxes  = document.querySelectorAll('.cfg-select-input');
  var checked = document.querySelectorAll('.cfg-select-input:checked');
  var origin = normalizeURL(window.location.origin);
  // The local server row can be selected for scanning/hiding, but can
  // never be deleted, so it's excluded from the Delete Selected count.
  var deletableChecked = Array.prototype.filter.call(checked, function(inp) {
    return inp.closest('tr').dataset.cfgUrl !== origin;
  });
  var delBtn = el('cfg-delete-selected-btn');
  if (delBtn) {
    delBtn.disabled = deletableChecked.length === 0;
    delBtn.textContent = deletableChecked.length > 0
      ? 'Delete Selected (' + deletableChecked.length + ')' : 'Delete Selected';
  }
  var scanBtn = el('cfg-scan-selected-btn');
  if (scanBtn) {
    scanBtn.disabled = checked.length === 0;
    scanBtn.textContent = checked.length > 0
      ? 'Scan for registries (' + checked.length + ')' : 'Scan for registries';
  }
  var all = el('cfg-select-all');
  if (all) {
    all.checked = boxes.length > 0 && checked.length === boxes.length;
    all.indeterminate = checked.length > 0 && checked.length < boxes.length;
  }
}

// Deletes every checked, deletable server row in one pass (the local
// server row, if checked, is silently skipped — it can't be deleted),
// then re-renders the Config page (which naturally resets the selection
// since the table is rebuilt from scratch).
function cfgDeleteSelected() {
  var origin = normalizeURL(window.location.origin);
  var urls = [];
  document.querySelectorAll('.cfg-select-input:checked').forEach(function(inp) {
    var url = inp.closest('tr').dataset.cfgUrl;
    if (url !== origin) urls.push(url);
  });
  if (urls.length === 0) return;
  urls.forEach(function(url) { removeServer(url); });
  renderConfig();
}

// "Scan for registries" bulk action: queries `.xregistry` on every
// checked row (see discoverRegistriesFrom()) and opens a review dialog
// grouping results into "Already added" / "Not yet added" — no server is
// actually added until the user confirms via the dialog's "Add Selected"
// button (see cfgConfirmScanResults()). Replaces the old automatic
// background scan entirely (see plan.md).
function cfgScanSelected() {
  var sources = [];
  document.querySelectorAll('.cfg-select-input:checked').forEach(function(inp) {
    sources.push(inp.closest('tr').dataset.cfgUrl);
  });
  if (sources.length === 0) return;
  var btn = el('cfg-scan-selected-btn');
  if (btn) { btn.disabled = true; btn.textContent = 'Scanning\u2026'; }
  discoverRegistriesFrom(sources, function(results) {
    if (btn) { btn.disabled = false; cfgUpdateSelection(); }
    cfgShowScanResults(results);
  });
}

// Renders the post-scan review dialog: a simple centered overlay with
// two grouped sections (Already added / Not yet added), each row for a
// "Not yet added" entry getting its own pre-checked checkbox so the user
// can deselect any they don't want before confirming. A "Select all"
// checkbox atop the "Not yet added" group toggles them all at once (see
// cfgToggleScanResultsAll()).
function cfgShowScanResults(results) {
  var existing = document.getElementById('cfg-scan-results-overlay');
  if (existing) existing.remove();

  var known   = results.filter(function(r) { return r.alreadyKnown; });
  var unknown = results.filter(function(r) { return !r.alreadyKnown; });

  var body;
  if (results.length === 0) {
    body = '<p class="cfg-scan-empty">No sibling registries were found on'
      + ' the selected server(s).</p>';
  } else {
    body = '';
    if (unknown.length) {
      body += '<h4 class="cfg-scan-group-title">Not yet added</h4>'
        + '<label class="cfg-scan-select-all">'
        +   '<input type="checkbox" id="cfg-scan-select-all-input" checked '
        +     'onchange="cfgToggleScanResultsAll(this.checked)"> Select all'
        + '</label>'
        + '<ul class="cfg-scan-list">'
        + unknown.map(function(r, i) {
            return '<li class="cfg-scan-item">'
              + '<label><input type="checkbox" class="cfg-scan-result-input" '
              +   'data-url="' + esc(r.url) + '" data-from="' + esc(r.discoveredFrom) + '" '
              +   'checked onchange="cfgUpdateScanSelectAll()"> '
              + esc(r.url) + '</label>'
              + '</li>';
          }).join('')
        + '</ul>';
    }
    if (known.length) {
      body += '<h4 class="cfg-scan-group-title">Already added</h4><ul class="cfg-scan-list cfg-scan-list-known">'
        + known.map(function(r) {
            return '<li class="cfg-scan-item cfg-scan-item-known">' + esc(r.url) + '</li>';
          }).join('')
        + '</ul>';
    }
  }

  var overlay = document.createElement('div');
  overlay.id = 'cfg-scan-results-overlay';
  overlay.className = 'cfg-modal-overlay';
  overlay.innerHTML = '<div class="cfg-modal">'
    + '<h3 class="cfg-modal-title">Scan Results</h3>'
    + body
    + '<div class="cfg-modal-btns">'
    +   (unknown.length ? '<button class="cfg-btn" onclick="cfgConfirmScanResults()">Add Selected</button>' : '')
    +   '<button class="cfg-btn" onclick="document.getElementById(\'cfg-scan-results-overlay\').remove()">Close</button>'
    + '</div>'
    + '</div>';
  document.body.appendChild(overlay);
}

// "Select all" checkbox atop the "Not yet added" list — checks/unchecks
// every per-row checkbox in that group to match.
function cfgToggleScanResultsAll(checked) {
  document.querySelectorAll('.cfg-scan-result-input').forEach(function(inp) {
    inp.checked = checked;
  });
}

// Keeps the "Select all" checkbox in sync (checked only when every row is
// checked) when a user toggles individual rows one at a time.
function cfgUpdateScanSelectAll() {
  var all  = el('cfg-scan-select-all-input');
  if (!all) return;
  var rows = document.querySelectorAll('.cfg-scan-result-input');
  var checkedCount = document.querySelectorAll('.cfg-scan-result-input:checked').length;
  all.checked = rows.length > 0 && checkedCount === rows.length;
}

// "Add Selected" in the scan-results dialog: adds every checked "Not yet
// added" entry (addServer() itself still guards against a URL that
// somehow became known in the meantime), then closes the dialog and
// re-renders Config so the new rows appear immediately.
function cfgConfirmScanResults() {
  document.querySelectorAll('.cfg-scan-result-input:checked').forEach(function(inp) {
    addServer(inp.dataset.url, inp.dataset.from);
  });
  var overlay = document.getElementById('cfg-scan-results-overlay');
  if (overlay) overlay.remove();
  renderConfig();
}

function cfgSetOpt(key, val) {
  _opts[key] = val;
  saveOpts();
}

function cfgSetJsonColor(mode) {
  _opts.jsonColorMode = mode;
  saveOpts();
  applyJsonColorMode();
}

// Flips the "xRegistry Focused" option (see optXregFocused()) and
// re-renders the current page immediately so the effect is visible
// without a manual reload, matching cfgSetJsonColor()'s behavior.
function cfgSetXregFocused(checked) {
  _opts.xregFocused = !!checked;
  saveOpts();
  refresh();
}

// Adds a new server from the Config page's Add row. Blocks (shows an
// inline error, touches nothing) if the URL is already configured — a
// URL is a server's unique identity, so re-adding a known URL (e.g. to
// try a different proxy setting) is not supported; the user must delete
// the existing entry first (see plan.md's duplicate-URL discussion).
function cfgAddNew() {
  var inp      = el('cfg-new-url');
  var nameInp  = el('cfg-new-name');
  var proxyInp = el('cfg-new-proxy');
  var errEl    = el('cfg-new-error');
  if (!inp || !inp.value.trim()) return;
  var url = inp.value.trim();
  if (errEl) { hideErrorBanner(errEl); }
  if (!addServer(url)) {
    if (errEl) {
      errEl.textContent = 'Already configured — delete the existing entry'
        + ' first if you want to re-add it with different settings.';
      errEl.style.display = '';
    }
    return;
  }
  if (nameInp && nameInp.value.trim()) setNameOverride(url, nameInp.value.trim());
  if (proxyInp && proxyInp.checked) setProxied(url, true);
  renderConfig();
  var newInp = el('cfg-new-url');
  if (newInp) newInp.focus();
}

// ---- Reset (clear browser-side state) -------------------------------------
//
// All browser-side state this app keeps lives in a handful of localStorage
// keys (LS_SERVERS, LS_OPTIONS, LS_NAMES, LS_PROXY, LS_DISCOVERED,
// LS_HIDDEN, LS_LABELS) plus a handful of in-memory caches (_modelCache/
// _capCache/_offeredCache etc.) that are rebuilt automatically on next use —
// a full page reload after clearing localStorage is therefore sufficient to
// reset everything, with no need to individually track/clear each in-memory
// cache here.

function cfgResetAll() {
  if (!window.confirm('Clear ALL saved registry locations and options, and reload? This cannot be undone.')) return;
  localStorage.removeItem(LS_SERVERS);
  localStorage.removeItem(LS_OPTIONS);
  localStorage.removeItem(LS_NAMES);
  localStorage.removeItem(LS_PROXY);
  localStorage.removeItem(LS_DISCOVERED);
  localStorage.removeItem(LS_HIDDEN);
  localStorage.removeItem(LS_LABELS);
  window.location.reload();
}

function cfgResetExceptServers() {
  if (!window.confirm('Clear all options (but keep your saved registry locations), and reload?')) return;
  localStorage.removeItem(LS_OPTIONS);
  window.location.reload();
}

// Grid (tile) view for the Groups/Resources/Versions collections was
// removed — List view (renderTableView) has all the same information plus
// sorting and (for Versions) a Document column, so Grid added no unique
// value there. See plan.md "Grid view removed for collection pages".

// ---- Table view ----------------------------------------------------------

var _sortCol = null;
var _sortAsc = true;
var _tableViewItems = []; // current renderTableView() items, indexed for per-row doc buttons

// ---- Collection Add/Delete (List view) -------------------------------------
//
// "+ Add <Singular>" button on Group/Resource/Version collection List pages
// (depth 1/3/5) opens an inline blank entity form above the table, reusing
// renderEditableScalarInput() (the same per-attribute input rendering the
// single-entity editor uses) so new/existing entities look and behave
// consistently. Only the entity's own ID (required) + simple universal
// spec fields (name/description/documentation/icon, offered only when
// specAttrLevel() confirms they're defined at this depth) + model-declared
// top-level SCALAR attributes are offered — nested object/array/map
// attributes and Labels/Constraints/Deprecated are out of scope for v1,
// same limitation as the single-entity editor (see plan.md "Full Data Edit
// Mode").
//
// Bulk delete reuses the Config page's checkbox/select-all/"Delete
// Selected (N)" pattern (cfgToggleSelectAll/cfgUpdateSelection/
// cfgDeleteSelected) — see collToggleSelectAll()/collUpdateSelection()/
// collDeleteSelected() below.
var _addNewOpen  = false;
var _addNewData  = {};
var _addNewKey   = null; // "serverURL|collPath" — resets the form when navigating to a different collection

// Returns the model's raw top-level attributes map (before ifvalues
// resolution) for the entity that would live at `path` — depth 2 (group
// instance), depth 4/6 (resource/version, which share one attributes map;
// GET /resource returns the flattened default version) are the only depths
// collection Add ever needs (collection paths are depth 1/3/5, so the
// synthetic new-entity path is always +1 of that).
function modelAttrsMapForPath(model, path) {
  var depth = path ? path.length : 0;
  if (depth === 2) {
    var gm = model && model.groups && model.groups[path[0]];
    return (gm && gm.attributes) || {};
  }
  if (depth >= 4) {
    var gm2 = model && model.groups && model.groups[path[0]];
    var rm  = gm2 && gm2.resources && gm2.resources[path[2]];
    return (rm && rm.attributes) || {};
  }
  return {};
}

function toggleAddEntityForm() {
  _addNewOpen = !_addNewOpen;
  if (!_addNewOpen) _addNewData = {};
  if (_lastData) renderTableView(_lastData);
}

function cancelAddEntity() {
  _addNewOpen = false;
  _addNewData = {};
  if (_lastData) renderTableView(_lastData);
}

// True whenever the inline "Add new <Type>" form is open AND has some
// actual content typed in — used by pushState()/setDataView()'s
// leave-guard so navigating away doesn't silently discard an in-progress
// new Group/Resource the way it used to (see plan.md "Add-new-entity
// leave-without-saving guard"). _addNewData only tracks the model/
// extension attribute inputs (see addNewFieldChange()) — the required ID
// input has no onchange handler and is only read live from the DOM at
// save time (see saveNewEntity()), so it must be checked separately here.
function isAddNewFormDirty() {
  if (!_addNewOpen) return false;
  if (Object.keys(_addNewData).length > 0) return true;
  var idInput = document.getElementById('addNewIdInput');
  return !!(idInput && idInput.value && idInput.value.trim());
}

// onchange handler for every input in the Add form — mirrors
// dataEditFieldChange() but writes into _addNewData instead of
// _dataEditData (the Add form has no "pristine snapshot" — everything
// typed is new).
function addNewFieldChange(k, inputEl) {
  var type = inputEl.dataset.ftype || 'string';
  var val;
  if (type === 'boolean') {
    val = inputEl.checked;
  } else if (type === 'integer' || type === 'uinteger') {
    val = inputEl.value === '' ? null : parseInt(inputEl.value, 10);
  } else if (type === 'decimal') {
    val = inputEl.value === '' ? null : parseFloat(inputEl.value);
  } else {
    val = inputEl.value;
  }
  if (val === '' || val == null) { delete _addNewData[k]; }
  else { _addNewData[k] = val; }
}

function addNewEntityAddExtension() {
  var nameEl = document.getElementById('addNewExtName');
  var valEl  = document.getElementById('addNewExtVal');
  if (!nameEl || !valEl) return;
  var name = (nameEl.value || '').trim();
  if (!name) return;
  if (Object.prototype.hasOwnProperty.call(_addNewData, name)) {
    alert('Attribute "' + name + '" already exists.');
    return;
  }
  _addNewData[name] = readExtTypeValue('addNewExtType', 'addNewExtVal');
  if (_lastData) renderTableView(_lastData);
}

// Builds the blank "Add new <Singular>" panel — shown above the collection
// table when _addNewOpen is true.
// Toolbar row above the collection table: "+ Add <Singular>" (toggles the
// blank entity form) and "Delete Selected" (bulk-delete, disabled until at
// least one row is checked) — same placement/styling convention as the
// Config page's per-section header buttons.
function buildAddEntityToolbarHtml(model) {
  var entityPath = _state.path.concat(['__new__']);
  var singular = capitalize(getSingularName(model, entityPath) || 'Item');
  // When the "Add" form is open, show Create/Cancel here (up top) instead of
  // the bottom of the form — avoids the earlier duplicate Create/Cancel
  // pair (one up top toggling the form, one at the bottom of the form).
  var addBtnsHtml = _addNewOpen
    ? '<button class="cfg-btn cfg-btn-primary" id="coll-add-new-create-btn" onclick="saveNewEntity()">Create</button>'
      + ' <button class="cfg-btn cfg-btn-cancel" id="coll-add-new-btn" onclick="cancelAddEntity()">Cancel</button>'
    : '<button class="cfg-btn cfg-btn-primary" id="coll-add-new-btn" onclick="toggleAddEntityForm()">+ Add ' + esc(singular) + '</button>';
  return '<div class="config-section-header" style="margin-bottom:8px">'
    + '<div class="config-section-header-btns">'
    + addBtnsHtml
    + ' <button class="cfg-btn cfg-btn-danger" id="coll-delete-selected-btn" disabled '
    + 'onclick="collDeleteSelected()" title="Delete all checked rows below">Delete Selected</button>'
    + '</div></div>';
}

function buildAddEntityFormHtml(model, collPath) {
  var entityPath = collPath.concat(['__new__']);
  var singular   = (getSingularName(model, entityPath) || 'entity').toLowerCase();
  var idKey      = singular + 'id';
  var specLevel  = specAttrLevel(entityPath);
  var attrsMap   = modelAttrsMapForPath(model, entityPath);
  var depth = entityPath.length;
  var resourceSingular = (depth === 4) ? singular
    : (depth >= 6 && model) ? (getSingularName(model, entityPath.slice(0, 4)) || '').toLowerCase()
    : null;

  function propRow(label, inputHtml) {
    return '<tr><td style="font-weight:bold;color:#444;width:200px">' + esc(label) + '</td><td>' + inputHtml + '</td></tr>';
  }

  var rowsHtml = '';
  ['name', 'description', 'documentation', 'icon'].forEach(function(k) {
    if (!specLevel || !specLevel[k]) return;
    var attrType = (k === 'documentation' || k === 'icon') ? 'url' : 'string';
    var val = _addNewData[k] != null ? _addNewData[k] : '';
    rowsHtml += propRow(labelFor(k, specLevel, singular),
      renderEditableScalarInput(k, val, {type: attrType}, 'addNewFieldChange'));
  });
  // Remaining model-declared attributes (extensions, plus any timestamp
  // spec attrs like createdat/modifiedat that are legitimately settable at
  // creation time) — exclude the dedicated ID field, name/description/
  // documentation/icon (already rendered above, would otherwise duplicate),
  // complex types (object/array/map, not editable here), and anything the
  // model marks readonly (server-computed: self/shortself/xid/epoch/etc, or
  // a <plural>url/<plural>count sub-collection pair) since those can never
  // be set by the client. Ordered the same way View mode's Property table
  // orders its rows (see orderPropKeysFlat()), instead of a flat
  // alphabetical sort, so this form reads in the same familiar order.
  var alreadyShown = {name: true, description: true, documentation: true, icon: true};
  var scalarKeys = Object.keys(attrsMap).filter(function(k) {
    if (k === '*' || k === idKey || alreadyShown[k]) return false;
    var attr = attrsMap[k];
    if (!attr || attr.type === 'object' || attr.type === 'array' || attr.type === 'map') return false;
    if (attr.readonly) return false;
    return true;
  });
  orderPropKeysFlat(scalarKeys, specLevel, singular, resourceSingular).forEach(function(k) {
    var attr = attrsMap[k];
    var val = _addNewData[k] != null ? _addNewData[k] : (attr.type === 'boolean' ? false : '');
    rowsHtml += propRow(labelFor(k, specLevel, singular),
      renderEditableScalarInput(k, val, attr, 'addNewFieldChange'));
  });

  // Error banner sits at the top of the panel (right below the Create/
  // Cancel buttons in buildAddEntityToolbarHtml(), which is rendered just
  // before this) rather than below the property table, so a Create-time
  // error is immediately visible without scrolling past the whole form.
  var html = '<div class="xr-add-entity-panel" style="margin-bottom:16px">'
    + '<div id="addNewEntityError" class="error-banner" style="display:none"></div>'
    + '<table class="xr-table xr-table-props"><thead><tr><th>' + esc(capitalize(singular) + ' Details') + '</th><th></th></tr></thead><tbody>'
    + propRow(capitalize(singular) + ' ID *', '<input type="text" id="addNewIdInput" class="xr-edit-input" required>')
    + rowsHtml;
  if (attrsMap['*']) {
    html += propRow('+ Add Extension',
      '<input type="text" id="addNewExtName" class="xr-edit-input" placeholder="name" style="width:140px">'
      + ' ' + renderExtTypeValueWidgetHtml('addNewExtType', 'addNewExtVal')
      + ' <button class="editorBtn editorBtnSmall" onclick="addNewEntityAddExtension()">Add</button>');
  }
  html += '</tbody></table>'
    + '</div>';
  return html;
}

// Creates the new entity via PUT to <collection URL>/<id>, then refreshes
// the collection page (a plain refresh() re-fetch — simplest way to pick
// up the new row with correctly server-computed fields like self/xid/
// createdat, same approach used elsewhere after a mutation).
function saveNewEntity(cb) {
  var errDiv = document.getElementById('addNewEntityError');
  if (errDiv) { hideErrorBanner(errDiv); }
  var idInput = document.getElementById('addNewIdInput');
  var id = idInput ? idInput.value.trim() : '';
  if (!id) {
    if (errDiv) { showErrorBanner(errDiv, 'ID is required.'); }
    return;
  }
  var url = buildBaseURL() + '/' + encodeURIComponent(id);
  var body = Object.assign({}, _addNewData);

  // Resources/Versions treat a plain (non-$details) PUT body as the
  // literal document content when the entity has a document (or as soon
  // as one is added later, for Versions of a hasdocument=true Resource)
  // — so sending our metadata-only object (even an empty {}) here would
  // create the entity with THAT as its document instead of leaving it
  // empty. Use $details so this body is always parsed as metadata only,
  // leaving the actual document unset for now (the user can add real
  // document content afterward). Always applied for these two collection
  // types (Resources: path length 3, Versions: path length 5) regardless
  // of the model's hasdocument setting — $details is a harmless no-op
  // when there's no document concept, so there's no need to check first.
  if (_state.path.length === 3 || _state.path.length === 5) {
    url += '$details';
  }

  var overlay = document.createElement('div'); overlay.className = 'savingOverlay';
  var box = document.createElement('div'); box.className = 'savingBox';
  var spinnerEl = document.createElement('div'); spinnerEl.className = 'savingSpinner';
  var msgEl = document.createElement('div'); msgEl.textContent = 'Creating\u2026';
  box.appendChild(spinnerEl); box.appendChild(msgEl); overlay.appendChild(box);
  document.body.appendChild(overlay);
  function removeOverlay() { if (overlay.parentNode) overlay.parentNode.removeChild(overlay); }

  fetch(url, {
    method: 'PUT',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(body, null, 2)
  }).then(function(resp) {
    return resp.text().then(function(text) {
      removeOverlay();
      if (resp.ok) {
        _addNewOpen = false; _addNewData = {};
        invalidateRegistryProbe(_state.serverURL || window.location.origin);
        if (cb) { cb(); } else { refresh(); }
      } else {
        if (errDiv) { showErrorBanner(errDiv, 'Error (' + resp.status + '):\n' + text); }
      }
    });
  }).catch(function(err) {
    removeOverlay();
    if (errDiv) { showErrorBanner(errDiv, 'Network error: ' + err.message); }
  });
}

// ---- Collection bulk delete -------------------------------------------------

var _collSelectedIds = {}; // id string -> true, current collection page only
var _collSelectedKey = null; // "serverURL|collPath" — resets selection when navigating

function collToggleSelectAll(cb) {
  document.querySelectorAll('.coll-select-input').forEach(function(inp) {
    inp.checked = cb.checked;
    if (cb.checked) _collSelectedIds[inp.value] = true; else delete _collSelectedIds[inp.value];
  });
  collUpdateSelection();
}

function collUpdateSelection() {
  var boxes = document.querySelectorAll('.coll-select-input');
  _collSelectedIds = {};
  var checked = 0;
  boxes.forEach(function(inp) {
    if (inp.checked) { _collSelectedIds[inp.value] = true; checked++; }
  });
  var btn = el('coll-delete-selected-btn');
  if (btn) {
    btn.disabled = checked === 0;
    btn.textContent = checked > 0 ? 'Delete Selected (' + checked + ')' : 'Delete Selected';
  }
  var all = el('coll-select-all');
  if (all) {
    all.checked = boxes.length > 0 && checked === boxes.length;
    all.indeterminate = checked > 0 && checked < boxes.length;
  }
}

// Deletes every checked row's entity, one DELETE request each, then
// refreshes the collection page. Reports the first failure but keeps
// going so one bad row doesn't block deleting the rest.
function collDeleteSelected() {
  var ids = Object.keys(_collSelectedIds);
  if (!ids.length) return;
  if (!confirm('Delete ' + ids.length + ' item' + (ids.length > 1 ? 's' : '') + '? This cannot be undone.')) return;

  var overlay = document.createElement('div'); overlay.className = 'savingOverlay';
  var box = document.createElement('div'); box.className = 'savingBox';
  var spinnerEl = document.createElement('div'); spinnerEl.className = 'savingSpinner';
  var msgEl = document.createElement('div'); msgEl.textContent = 'Deleting\u2026';
  box.appendChild(spinnerEl); box.appendChild(msgEl); overlay.appendChild(box);
  document.body.appendChild(overlay);
  function removeOverlay() { if (overlay.parentNode) overlay.parentNode.removeChild(overlay); }

  var base = buildBaseURL();
  var errors = [];
  Promise.all(ids.map(function(id) {
    return fetch(base + '/' + encodeURIComponent(id), {method: 'DELETE'}).then(function(resp) {
      if (!resp.ok && resp.status !== 204) {
        return resp.text().then(function(text) { errors.push(id + ': ' + resp.status + ' ' + text); });
      }
    }).catch(function(err) { errors.push(id + ': ' + err.message); });
  })).then(function() {
    removeOverlay();
    _collSelectedIds = {};
    invalidateRegistryProbe(_state.serverURL || window.location.origin);
    if (errors.length) alert('Some deletes failed:\n' + errors.join('\n'));
    refresh();
  });
}

function renderTableView(data) {
  var main = el('main-view');
  // If a server-side sort= is active (applied via the Sort picker, now
  // available in this panel too — see renderJSONLeftPanel()) and the user
  // hasn't clicked a column header yet (_sortCol null), honor the server's
  // returned order instead of the default ID sort. Clicking any header (see
  // sortBy()) clears _state.sort/the picker so only one sort mechanism is
  // ever "active" at a time — see plan.md "List view Sort picker".
  var preserveOrder = !_sortCol && !!_state.sort;
  var items = collectionItems(data, preserveOrder);

  var svBase = (_state.serverURL || window.location.origin).replace(/\/$/, '');
  var modelKey = normalizeURL(svBase);
  if (!_modelCache.hasOwnProperty(modelKey)) {
    ensureModelCached(svBase, function() {
      if (_lastData === data) renderTableView(data);
    });
  }
  var model  = _modelCache[modelKey] || null;
  var depth  = _state.path.length; // 1=groups, 3=resources, 5=versions

  // Collection Add/Delete only ever apply to actual Group/Resource/Version
  // collections (depth 1/3/5) — never the Registry root's own "Group
  // Types"/"Resources" summary tables (those are rendered by
  // renderSingleEntity(), a different function entirely). Also gated on
  // _state.editMode — like single-entity edit mode, these mutation
  // controls (Add button, selection checkboxes, Delete Selected) should
  // stay hidden until the user explicitly turns edit mode on via the
  // pencil button, not merely because the underlying collection happens
  // to be mutable.
  var canAddDelete = _state.mutable && _state.editMode
    && (depth === 1 || depth === 3 || depth === 5);
  var collKeyNow = svBase + '|' + _state.path.join('/');
  if (_addNewKey !== collKeyNow) { _addNewOpen = false; _addNewData = {}; _addNewKey = collKeyNow; }
  if (_collSelectedKey !== collKeyNow) { _collSelectedIds = {}; _collSelectedKey = collKeyNow; }

  if (items.length === 0) {
    main.innerHTML = '<div id="table-container">'
      + (canAddDelete ? buildAddEntityToolbarHtml(model) : '')
      + (canAddDelete && _addNewOpen ? buildAddEntityFormHtml(model, _state.path) : '')
      + '<div class="state-msg">No items found</div></div>';
    return;
  }

  // Determine which optional columns to show based on data presence
  var hasName = items.some(function(it) { return it.name != null && it.name !== ''; });
  var hasDesc = items.some(function(it) { return it.description != null && it.description !== ''; });
  // Show children column for group (depth 1) and resource (depth 3) collections
  var showChildren = (depth === 1 || depth === 3);
  // Show a document column for the versions collection (depth 5) when the
  // resource type has a document — mirrors the Grid view's document tile.
  var showDoc = (depth === 5) && resourceHasDocument(model, _state.path);
  var docSingular = showDoc ? getSingularName(model, _state.path.slice(0, 4)) : null;

  // Sort support — '__id' is a virtual column keyed by itemNavKey
  if (_sortCol) {
    items = items.slice().sort(function(a, b) {
      var av = _sortCol === '__id' ? itemNavKey(a) : String(a[_sortCol] == null ? '' : a[_sortCol]);
      var bv = _sortCol === '__id' ? itemNavKey(b) : String(b[_sortCol] == null ? '' : b[_sortCol]);
      return _sortAsc ? av.localeCompare(bv) : bv.localeCompare(av);
    });
  }
  _tableViewItems = items;

  function thSort(col, label, extraCls) {
    var cls = col === _sortCol ? (_sortAsc ? ' sorted-asc' : ' sorted-desc') : '';
    if (extraCls) cls += ' ' + extraCls;
    return '<th class="' + cls + '" onclick="sortBy(\'' + esc(col) + '\')">' + esc(label) + '</th>';
  }

  var idColLabel = capitalize(getSingularName(model, _state.path.concat(['__x__'])));
  var showVersionId = (depth === 3); // resource collection: show its default version id
  var html = '<div id="table-container">';

  // Page title — collection pages have no single-entity name to show (Grid
  // view's tile layout doesn't show one either), but for visual consistency
  // with the single-entity table views' new page title, show the plural
  // collection name (e.g. "schemagroups") using the same title styling,
  // prefixed with whichever parent identity applies — consistently shaped
  // as "Singular-Parent: Child" at every depth (see plan.md "Collection
  // page title parent prefixes"):
  //   - Groups collection (depth 1): the Registry's own display name (same
  //     name shown as the first breadcrumb segment — see serverLabel()).
  //   - Resources collection (depth 3): the owning Group instance's id
  //     (its singular Group-type name is dropped — the id alone is enough
  //     context here, unlike depth 5 below where the Resource id doesn't
  //     otherwise appear on screen).
  //   - Versions collection (depth 5): the owning Resource instance's id.
  var pluralLabel = _state.path[_state.path.length - 1];
  var titleIdPrefix = '';
  if (depth === 1) {
    var regLabelT = serverLabel(_state.serverURL || window.location.origin);
    titleIdPrefix = '<span class="eg-page-title-id-prefix">' + esc(regLabelT) + ':</span> ';
  } else if (depth === 3) {
    titleIdPrefix = '<span class="eg-page-title-id-prefix">' + esc(_state.path[1]) + ':</span> ';
  } else if (depth === 5) {
    titleIdPrefix = '<span class="eg-page-title-id-prefix">' + esc(_state.path[3]) + ':</span> ';
  }
  // Header icon: Groups list (depth 1) shows the model's Group-type icon;
  // Resources list (depth 3) and Versions list (depth 5, no Version-level
  // Type concept — reuses the owning Resource Type's icon) show the model's
  // Resource-type icon. See plan.md "Icon propagation from model + entity
  // data".
  var titleIconUrl = '';
  if (depth === 1) titleIconUrl = modelGroupIcon(model, _state.path[0]);
  else if (depth === 3 || depth === 5) titleIconUrl = modelResourceIcon(model, _state.path[0], _state.path[2]);
  html += '<div class="eg-page-title">' + iconThumbHtml(titleIconUrl, 'eg-page-title-icon') + titleIdPrefix + '<span class="eg-page-title-type">' + esc(pluralLabel) + '</span></div>';
  html += serverURLLineHtml();

  if (canAddDelete) html += buildAddEntityToolbarHtml(model);
  if (canAddDelete && _addNewOpen) html += buildAddEntityFormHtml(model, _state.path);

  html += '<table class="xr-table"><thead><tr>';
  if (canAddDelete) {
    html += '<th class="cfg-select-cell"><input type="checkbox" id="coll-select-all" '
      + 'title="Select/deselect all" onchange="collToggleSelectAll(this)"></th>';
  }
  html += thSort('__id', idColLabel);
  if (hasName) html += thSort('name', 'Name');
  if (hasDesc) html += thSort('description', 'Description');
  if (showVersionId) html += '<th class="col-center cell-version-hdr">Default<br>Version</th>';
  if (showChildren) html += '<th' + (depth === 3 ? ' class="col-center"' : '') + '>' + (depth === 1 ? 'Resources' : 'Versions') + '</th>';
  if (showDoc) html += '<th>Document</th>';
  html += thSort('createdat', 'Created', 'col-center');
  html += thSort('modifiedat', 'Modified', 'col-center');
  html += '</tr></thead><tbody>';

  items.forEach(function(item, idx) {
    var id      = itemNavKey(item);
    var itemPath = _state.path.concat([id]);
    var colls   = showChildren ? findCollectionRefs(model, itemPath, item) : [];

    var childrenHtml = '';
    if (showChildren) {
      if (colls.length) {
        if (depth === 3) {
          // Single "versions" collection per resource — just the count, still
          // clickable to navigate straight into it (see navigateToNestedColl()).
          childrenHtml = colls.map(function(c) {
            var clickExpr = 'event.stopPropagation();navigateToNestedColl(' + JSON.stringify(id) + ',' + JSON.stringify(c.plural) + ',' + JSON.stringify(c.url) + ')';
            var pillHref = pageHref(itemPath.concat([c.plural]), c.url);
            return '<a class="cell-version-count" href="' + esc(pillHref) + '" onclick="' + esc(guardedOnclick(clickExpr)) + '">' + c.count + '</a>';
          }).join(' ');
        } else {
          childrenHtml = colls.map(function(c) {
            var clickExpr = 'event.stopPropagation();navigateToNestedColl(' + JSON.stringify(id) + ',' + JSON.stringify(c.plural) + ',' + JSON.stringify(c.url) + ')';
            var pillHref = pageHref(itemPath.concat([c.plural]), c.url);
            return '<a class="coll-tile-res-pill coll-tile-res-pill-clickable" href="' + esc(pillHref) + '" onclick="' + esc(guardedOnclick(clickExpr)) + '">' + esc(c.plural) + ' (' + c.count + ')</a>';
          }).join(' ');
        }
      } else {
        childrenHtml = '<span class="coll-tile-res-none">—</span>';
      }
    }

    // The row itself is no longer clickable (so its text can be selected/
    // copied) — only the id cell's text is a real <a>, consistent with the
    // Grid view's tile-id-link.
    var rowSelf = entityHrefWithFilter(item.self || '', itemPath);
    var rowClickExpr = 'navigateTo(' + JSON.stringify(id) + ',' + JSON.stringify(rowSelf) + ')';
    var rowHref = pageHref(itemPath, rowSelf);
    // Row icon: own instance `icon` attribute wins, else model Group-type
    // (depth 1) or Resource-type (depth 3) icon fallback. Versions (depth
    // 5) reuse the owning Resource's resolved icon (no separate Version
    // Type icon concept) — see plan.md "Icon propagation from model +
    // entity data".
    var rowIconUrl = '';
    if (depth === 1) rowIconUrl = resolveGroupIcon(model, _state.path[0], item);
    else if (depth === 3) rowIconUrl = resolveResourceIcon(model, _state.path[0], _state.path[2], item);
    else if (depth === 5) rowIconUrl = modelResourceIcon(model, _state.path[0], _state.path[2]);
    html += '<tr>';
    if (canAddDelete) {
      var checkedNow = _collSelectedIds[id] ? ' checked' : '';
      html += '<td class="cfg-select-cell"><input type="checkbox" class="coll-select-input" value="'
        + esc(id) + '"' + checkedNow + ' onchange="collUpdateSelection()"></td>';
    }
    html += '<td class="cell-id">' + iconThumbHtml(rowIconUrl, 'row-icon-thumb') + '<a href="' + esc(rowHref) + '" onclick="' + esc(guardedOnclick(rowClickExpr)) + '">' + esc(id) + '</a></td>';
    if (hasName) html += '<td>' + esc(item.name != null ? String(item.name) : '') + '</td>';
    if (hasDesc) html += '<td class="cell-desc"><div class="cell-desc-text">' + esc(item.description != null ? String(item.description) : '') + '</div></td>';
    if (showVersionId) {
      if (item.versionid != null) {
        var vUrl2 = defaultVersionURL(item, itemPath, colls);
        var vPath2 = itemPath.concat(['versions', String(item.versionid)]);
        var vClickExpr2 = 'event.stopPropagation();navigateToDefaultVersion(' + JSON.stringify(id) + ',' + JSON.stringify(item.versionid) + ',' + JSON.stringify(vUrl2) + ')';
        var vHref2 = pageHref(vPath2, vUrl2);
        html += '<td><a class="cell-version-count" href="' + esc(vHref2) + '" onclick="' + esc(guardedOnclick(vClickExpr2)) + '">' + esc(String(item.versionid)) + '</a></td>';
      } else {
        html += '<td></td>';
      }
    }
    if (showChildren) html += '<td class="cell-children' + (depth === 3 ? ' col-center' : '') + '">' + childrenHtml + '</td>';
    if (showDoc) {
      var docClickExpr = 'event.stopPropagation();openDocument(' + JSON.stringify(docSingular) + ', _tableViewItems[' + idx + '])';
      html += '<td class="cell-children">'
            + '<button class="cfg-btn" style="font-size:11px;padding:2px 8px" onclick="' + esc(docClickExpr) + '">View</button>'
            + '</td>';
    }
    html += '<td class="cell-timestamp col-center">' + esc(formatTimestamp(item.createdat)) + '</td>';
    html += '<td class="cell-timestamp col-center">' + esc(formatTimestamp(item.modifiedat)) + '</td>';
    html += '</tr>';
  });

  html += '</tbody></table></div>';
  main.innerHTML = html;
  if (canAddDelete) collUpdateSelection();
}

function sortBy(col) {
  if (_sortCol !== col) { _sortCol = col; _sortAsc = true; }
  else { _sortAsc = !_sortAsc; }
  // A column-header click is a deliberate client-side override — clear any
  // active server-side sort (and its picker draft) so the two mechanisms
  // are never both "active" at once (see plan.md "List view Sort picker").
  // No re-fetch needed: the already-fetched data just gets re-sorted in
  // place by renderTableView() below; only _state.sort/the URL/the picker
  // UI need to reflect that the server sort is no longer in effect.
  if (_state.sort) {
    _state.sort = '';
    history.replaceState(null, '', buildURL(_state));
    if (_sortDraft && _sortDraftKey === sortKey()) {
      _sortDraft = {mode: '', attr: '', mapKey: '', custom: '', desc: false};
    }
    var sortHost = el('lp-sort-section');
    if (sortHost) {
      var svBaseS = (_state.serverURL || window.location.origin).replace(/\/$/, '');
      sortHost.innerHTML = buildSortSectionInner(_modelCache[normalizeURL(svBaseS)] || null);
    }
  }
  if (_lastData) renderTableView(_lastData);
  // Update the Filters/Sort toggle button's sort-direction arrow to match
  // this column-click sort (or the clearing of a prior server sort) —
  // renderTableView() only redraws the table body, not the header button.
  renderHeader();
}

// ---- Single entity view --------------------------------------------------
//
// For the Registry root and Group/Resource entities.
// Scalar props shown in a property table; collection references (pairs of
// <name>url + <name>count) rendered as clickable rows.

// Registry endpoint pills (depth 0 only, both List and Grid views) — a
// compact presentation of which optional server endpoints (Model, Model
// Source, Capabilities, Capabilities Offered, .xregistry) this registry
// exposes, replacing the older separate "Registry Endpoints" table/tile-
// section. Mutable endpoints get a trailing pencil icon. Returns '' when
// nothing to show (i.e. no capabilities data cached yet, or no optional
// endpoints available).
//
// Section identifiers (used for _state.section/routing) never contain a
// dot, but the .xregistry capability's key in the server's `available` map
// literally does (see common/capabilities.go) — sectionCapKey() bridges
// that one mismatch everywhere a section id is used to look into `available`.
function sectionCapKey(s) { return s === 'xregistry' ? '.xregistry' : s; }

function buildRegEndpointPillsHtml() {
  if (_state.path.length !== 0) return '';
  // Domain view (the default; View mode only, see effectiveXregFocused()):
  // hide the whole "Config:" pills row — these are xRegistry-specific
  // endpoints (Model/ModelSource/Capabilities/etc), not domain data.
  if (!effectiveXregFocused() && !_state.editMode) return '';
  var svBaseP = (_state.serverURL || window.location.origin).replace(/\/$/, '');
  var capDataP = _capCache[normalizeURL(svBaseP)];
  var availP   = capDataP && capDataP.available;
  var sectionTilesP = ['model','modelsource','capabilities','capabilitiesoffered','xregistry'];
  var availSectionsP = sectionTilesP.filter(function(s) { return availP && availP[sectionCapKey(s)]; });
  if (!availSectionsP.length) return '';
  var sectionNamesP = {model:'Model', modelsource:'Model Source', capabilities:'Capabilities', capabilitiesoffered:'Capabilities Offered', xregistry:'.xregistry'};
  var html = '<div class="reg-endpoint-pills">';
  html += '<span class="reg-endpoint-pills-title">Config:</span>';
  availSectionsP.forEach(function(s) {
    var mutP = availP[sectionCapKey(s)] && availP[sectionCapKey(s)].mutable;
    var pushExprP = 'pushState({section:\'' + s + '\',useExport:false})';
    var sHrefP = buildURL(Object.assign({}, _state, {section: s, useExport: false}));
    html += '<a class="reg-endpoint-pill" href="' + esc(sHrefP) + '" onclick="' + esc(guardedOnclick(pushExprP)) + '">'
      + esc(sectionNamesP[s])
      + (mutP ? ' <span class="reg-endpoint-pill-edit" title="Mutable">&#9998;</span>' : '')
      + '</a>';
  });
  html += '</div>';
  return html;
}

// Builds the "Entity Data Editor" Save (PUT/PATCH)/Undo/Delete action bar —
// shared by renderSingleEntity() (initial render) and
// onVersionSelectChangeReal() (redraw when the "Version:" dropdown is
// switched back to "Default") so both stay in sync. Always targets the
// Default version's data (_dataEditData); deleteDisabled is only true at
// depth 0 (Registry root), where deleting the whole registry has no
// sensible "back to parent" destination in this UI.
function buildDataEditorActionBarHtml(deleteDisabled) {
  var deleteDisabledAttr = deleteDisabled ? ' disabled title="Deleting the registry itself is not supported here"' : '';
  return '<div id="dataEditorError" class="error-banner" style="display:none"></div>'
    + '<div class="editorActionBar" id="dataEditorActionBar" style="margin-bottom:8px">'
    + '<button class="editorBtn" id="dataSavePutBtn" onclick="saveDataEntity(\'PUT\')"' + (_dataDirty ? '' : ' disabled') + '>Save (full)</button>'
    + ' <button class="editorBtn" id="dataSavePatchBtn" onclick="saveDataEntity(\'PATCH\')"' + (_dataDirty ? '' : ' disabled') + '>Save (delta)</button>'
    + ' <button class="editorBtn" id="dataUndoBtn" onclick="undoDataEdit()"' + (_dataDirty ? '' : ' disabled') + '>Undo</button>'
    + ' <button class="editorBtn editorBtnDanger" onclick="deleteDataEntity()"' + deleteDisabledAttr + '>Delete</button>'
    + '</div>';
}

function renderSingleEntity(data) {
  var main = el('main-view');
  if (!data || typeof data !== 'object') {
    main.innerHTML = '<div class="state-msg">' + esc(String(data)) + '</div>';
    return;
  }

  // Meta page (depth 5) is replaced by the inline Metadata tab on the
  // Resource page — redirect up (mirrors renderEntityGrid()'s same redirect
  // for Grid view). Pass dataView explicitly — otherwise pushStateReal()
  // would recompute it from the per-depth default/JSON-sticky rules,
  // clobbering the view (table) the user just clicked to get here.
  if (_state.path.length === 5 && _state.path[4] === 'meta') {
    _pendingMetaTabOnLoad = true;
    pushState({path: _state.path.slice(0, 4), dataView: _state.dataView});
    return;
  }

  var svBase = (_state.serverURL || window.location.origin).replace(/\/$/, '');
  var modelKey = normalizeURL(svBase);
  if (!_modelCache.hasOwnProperty(modelKey)) {
    ensureModelCached(svBase, function() {
      if (_lastData === data) renderSingleEntity(data);
    });
  }
  var model  = _modelCache[modelKey] || null;

  // Entity Data Editor (see "Entity Data Editor" section below) — only
  // Registry root/Group/Resource pages (single-entity Property tables) are
  // editable; collection pages never get an inline edit form. (Version
  // pages, depth >= 6, no longer render through this List-view path at
  // all — the dedicated Version page was retired; see
  // normalizeVersionDepth(). Version entities are still editable, just via
  // the Resource page's "Version:" dropdown — see _verEditData/_verDirty.)
  // _dataEditData is snapshotted fresh whenever the server+path changes, so
  // navigating to a different entity always starts from that entity's own
  // data — but re-rendering the SAME entity (e.g. after a save, or toggling
  // edit mode on/off) reuses the existing working copy rather than
  // clobbering an in-progress edit.
  var dataEditDepth = _state.path.length;
  var isEditableEntityPage = (dataEditDepth === 0 || dataEditDepth === 2
    || dataEditDepth === 4);
  if (isEditableEntityPage) {
    var dataEditKey = svBase + '|' + _state.path.join('/');
    if (_dataLoadedFor !== dataEditKey) {
      _dataEditSrc  = deepClone(data);
      _dataEditData = deepClone(_dataEditSrc);
      _dataDirty    = false;
      _dataLoadedFor = dataEditKey;
      _itemTypeHints = {};
      _shapeHints = {};
    }
  }
  var dataEditingNow = isEditableEntityPage && _state.editMode && _state.mutable;
  var propsEntityData = dataEditingNow ? _dataEditData : data;
  // Default to 'entity' — set to 'meta' below only if the Metadata tab ends
  // up being the initially-active tab (see pendingDocTabActivate handling).
  _dataEditActiveKind = 'entity';

  var colls  = findCollectionRefs(model, _state.path, data);
  var collKeys = {};
  colls.forEach(function(c) {
    collKeys[c.plural + 'url'] = true;
    collKeys[c.plural + 'count'] = true;
    // The bare plural name itself (e.g. "definitions") is also a model-
    // declared attribute — the server's auto-generated map representation
    // of the sub-collection — but it's structural navigation (like
    // meta/metaurl), not real entity content, and editing a whole nested
    // Resource/Version collection inline here makes no sense. Only matters
    // in edit mode (see buildEntityPropsTableHtml()'s "show all model
    // attributes" merge, which is what would otherwise surface it), but
    // suppressing it unconditionally is harmless and simpler.
    collKeys[c.plural] = true;
  });

  // Attach model info to the Group Types collections row (depth 0), same as
  // renderEntityGrid's Grid-view tiles, so the table view can show matching
  // "Resource Types" and "Description" columns.
  if (model && model.groups && _state.path.length === 0) {
    colls.forEach(function(c) {
      var grpDef = model.groups[c.plural];
      c.resources   = grpDef && grpDef.resources ? Object.keys(grpDef.resources).sort() : [];
      c.description = (grpDef && grpDef.description) || '';
    });
  }

  // Attach model description to the Resources collections row (depth 2, a
  // Group instance page), same as Grid view's resource-type tiles, so the
  // table view's "Resources" table can show a matching Description column.
  if (model && model.groups && _state.path.length === 2) {
    var grpDefR = model.groups[_state.path[0]];
    colls.forEach(function(c) {
      var resDefR = grpDefR && grpDefR.resources && grpDefR.resources[c.plural];
      c.description = (resDefR && resDefR.description) || '';
    });
  }

  // Priority ordering for props — hand-tuned for UX, not spec declaration order.
  // specAttrOrder() gives spec-canonical order but it doesn't match what's most useful
  // to show first in the UI. Includes both scalar and complex (object/array) attrs —
  // complex values render as a nested key/value tree in the same 2nd column, same
  // approach as Grid view's unknown-extension rows (see renderValueTree()).
  var priority = ['registryid','xid','name','description','specversion',
    'epoch','createdat','modifiedat','versionid','isdefault','ancestorid'];
  var attrKeys = Object.keys(data).filter(function(k) {
    return !collKeys[k];
  }).sort(function(a, b) {
    var ai = priority.indexOf(a), bi = priority.indexOf(b);
    if (ai >= 0 && bi >= 0) return ai - bi;
    if (ai >= 0) return -1; if (bi >= 0) return 1;
    return a.localeCompare(b);
  });

  var html = '<div id="table-container">';

  // Page title — mirrors Grid view's "TYPE: name" header, so List/Table
  // view gets the same top-of-page identification at every depth, not just
  // the registry root. Uses the same ID-vs-name logic as Grid view's title
  // (getSingularName() for the type label, data.name/idVal for the display
  // text). At depth 0 the name often already ends with the word "Registry"
  // (e.g. "CloudEvents Registry"), which would be redundant right after the
  // "REGISTRY:" label, so that trailing word is stripped in that case.
  {
    var depthH = _state.path.length;
    var entityTypeH = getSingularName(model, _state.path) || 'Registry';
    var idFieldNameH = entityTypeH.toLowerCase() + 'id';
    var idValH = data[idFieldNameH] !== undefined ? data[idFieldNameH]
      : depthH > 0 ? _state.path[depthH - 1] : data.registryid;
    var titleDisplayH = data.name || (idValH != null ? String(idValH) : '');
    if (depthH === 0) {
      // Registry root header — a user-set name override (Config page)
      // takes priority over the server-reported name/registryid. See
      // plan.md "Registry name override on Config page". Only the
      // server-provided fallback gets the trailing-"Registry"-word
      // stripped (e.g. "CloudEvents Registry" → "CloudEvents", to avoid
      // "REGISTRY: CloudEvents Registry") — an explicit user override is
      // shown verbatim since the user already controls exactly what to
      // type there.
      var regNameOverrideH = getNameOverride(_state.serverURL || window.location.origin);
      titleDisplayH = regNameOverrideH
        || titleDisplayH.replace(/\s+Registry$/i, '');
    }
    // Header icon: Registry root page (depth 0) uses its own `icon` if
    // set; Group instance page (depth 2) uses its own `icon` if set, else
    // the model's Group-type icon; Resource instance page (depth 4) uses
    // its own `icon` else the model's Resource-type icon. See plan.md
    // "Icon propagation from model + entity data". (Version pages, depth
    // >= 6, no longer render through this List-view path at all — the
    // dedicated Version page was retired; see normalizeVersionDepth().)
    var titleIconUrlH = '';
    if (depthH === 0) titleIconUrlH = (data && typeof data.icon === 'string' && data.icon.trim()) ? data.icon : '';
    else if (depthH === 2) titleIconUrlH = resolveGroupIcon(model, _state.path[0], data);
    else if (depthH === 4) titleIconUrlH = resolveResourceIcon(model, _state.path[0], _state.path[2], data);
    html += '<div class="eg-page-title">' + iconThumbHtml(titleIconUrlH, 'eg-page-title-icon')
      + '<span class="eg-page-title-type">' + esc(entityTypeH) + ':</span>'
      + (titleDisplayH ? ' <span class="eg-page-title-id">' + esc(titleDisplayH) + '</span>' : '')
      + '</div>';
    html += serverURLLineHtml();
  }

  // Registry endpoint pills (depth 0 only) — see buildRegEndpointPillsHtml().
  html += buildRegEndpointPillsHtml();

  // Collections section — id cell is a real link; row itself is plain text
  // (not clickable) so its content can be selected/copied.
  // Suppressed entirely at depth 4 (Resource page): the only collection
  // there is "versions", and that's now covered by the "Versions List"
  // navigation link in the Document/Details tab bar below — see plan.md
  // "Versions List navigation link".
  if (colls.length && _state.path.length !== 4) {
    var depthT = _state.path.length;
    // Match the Grid view's section-header wording (GROUP TYPES / RESOURCES).
    var collsHeaderT = depthT === 0 ? 'Group Types' : depthT === 2 ? 'Resources' : 'Collection';
    var showResTypes = depthT === 0; // matches Grid view's group-type tiles
    var showCollDesc = colls.some(function(c) { return c.description; });
    html += '<table class="xr-table" style="margin-bottom:16px">';
    html += '<thead><tr><th>' + esc(collsHeaderT) + '</th><th>Count</th>'
          + (showResTypes ? '<th>Resource Types</th>' : '')
          + (showCollDesc ? '<th>Description</th>' : '')
          + '</tr></thead>';
    html += '<tbody>';
    colls.forEach(function(c) {
      var collClickExpr = 'navigateTo(' + JSON.stringify(c.plural) + ',' + JSON.stringify(c.url) + ')';
      var collHref = pageHref(_state.path.concat([c.plural]), c.url);
      var resTypesHtml = showResTypes
        ? (c.resources && c.resources.length ? esc(c.resources.join(', ')) : '')
        : '';
      // Collection row icon: depth 0 rows are Group Types, depth 2 rows are
      // Resource Types — both are Type listings (not instances), so only
      // the model-declared Type icon applies (no instance-level fallback).
      var collIconUrl = depthT === 0 ? modelGroupIcon(model, c.plural)
        : depthT === 2 ? modelResourceIcon(model, _state.path[0], c.plural) : '';
      html += '<tr>'
        + '<td class="cell-id">' + iconThumbHtml(collIconUrl, 'row-icon-thumb') + '<a href="' + esc(collHref) + '" onclick="' + esc(guardedOnclick(collClickExpr)) + '">' + esc(c.plural) + '</a></td>'
        + '<td>' + c.count + '</td>'
        + (showResTypes ? '<td>' + resTypesHtml + '</td>' : '')
        + (showCollDesc ? '<td class="cell-desc"><div class="cell-desc-text">' + esc(c.description || '') + '</div></td>' : '')
        + '</tr>';
    });
    html += '</tbody></table>';
  }

  // Entity Data Editor action bar — Save (PUT/PATCH) / Undo / Delete.
  // Only rendered when editing is actually active; disabled at depth 0
  // (Registry root) since deleting a whole registry has no sensible
  // "back to parent" destination in this UI and is out of scope for v1.
  // On Registry root/Group instance pages (depth 0/2, no tab bar — see the
  // plain Properties table branch below) it's placed directly above that
  // table. On Resource/Version pages (depth 4/6+, which have a Document/
  // Details tab bar) it's rendered as a sibling of the tab panels, right
  // below the tab bar buttons — see further down — so it stays visible no
  // matter which tab is active instead of being tucked inside one panel's
  // content (and hidden along with it via display:none when inactive).
  var dataEditorActionBarHtml = dataEditingNow ? buildDataEditorActionBarHtml(dataEditDepth === 0) : '';

  // Document / Details tab bar (depth 4 = Resource entity, depth 6+ =
  // Version entity) — replaces the old always-stacked meta-box +
  // Document-table + Properties-table layout with a small pill tab bar;
  // only one panel is visible at a time, and the first available tab is
  // always the default selection on every render (no persistence across
  // navigation). See plan.md "tabbed Document/Details component".
  var depthD = _state.path.length;
  var pendingDocTabActivate = null;
  _resVersionsUrl = ''; // reset each render; only set (truthy) for Resource pages (depth 4)
  if (depthD === 4) {
    var entityTypeT = getSingularName(model, _state.path);
    var capTypeT = capitalize(entityTypeT);
    var hasDocD = resourceHasDocument(model, _state.path);
    var docSingularD = entityTypeT;

    // Properties table content — built via the shared buildEntityPropsTableHtml()
    // helper (also used later to redraw this panel when the version-selector
    // dropdown, below, picks a different version).
    var propHeaderT = versionPropHeaderLabel(true, data && data.versionid);
    var propsTableHtml = buildEntityPropsTableHtml(propsEntityData, propHeaderT, model, _state.path, collKeys, dataEditingNow);

    var tabDefs = [];
    if (hasDocD) {
      // Compatibility's actual value (as opposed to whether it *validated*)
      // lives on the resource's Meta object, not on the flattened
      // Resource/Version data — fetched separately/asynchronously below via
      // ensureDocPillsCompat(), same pattern as the Document preview itself.
      _docPillsMetaCompat = null;
      // Snapshot for refreshVersionDetailsPanel() to redraw the props panel
      // (with the "(compat)" prefix on Compatibility Validated) once that
      // async fetch resolves — see _docPropsPanelInfo above.
      _docPropsPanelInfo = { panelId: 'eg-doc-panel-defver',
        headerLabel: propHeaderT, model: model, path: _state.path, collKeys: collKeys };
      tabDefs.push({ key: 'doc', label: 'Document',
        content: '<div id="eg-doc-pills">' + buildDocInfoPillsHtml(data) + '</div>'
               + '<div id="eg-doc-preview-box"><div class="eg-loading">Loading document\u2026</div></div>' });
      ensureDocPillsCompat(data);
    }
    var versionsUrlD = '';
    var versionsListLinkHtml = '';
    // Version property/details panel — a plain tab, same as before this
    // feature existed. Which version's data it shows is now controlled by
    // a *separate* standalone "Version:" dropdown rendered above the tab
    // bar (see plan.md "version-selector dropdown"), not by the tab bar
    // itself. Defaults to "Default" — the resource's own
    // flattened-default-version data, already rendered above.
    var versionsCollD = colls.filter(function(c) { return c.plural === 'versions'; })[0];
    versionsUrlD = versionsCollD ? versionsCollD.url : '';
    tabDefs.push({ key: 'defver', label: 'Version Details', content: propsTableHtml });
    // Only reset the Meta tab's cached/edit state when this render is for
    // a genuinely different resource than whatever it currently belongs
    // to — renderSingleEntity() also re-runs harmlessly for the SAME
    // resource once ensureModelCached()'s async callback resolves (see
    // above), and unconditionally nulling this out here previously wiped
    // an in-progress Meta-tab edit if the user started editing before
    // that second, model-now-available render fired (loadMetaDetails()
    // would then re-fetch and stomp _metaEditData/_metaDirty back to
    // pristine — see loadMetaDetails() for the matching half of this fix).
    if (_metaLoadedFor !== data.self) {
      _metaData = null;
      _metaEditSrc = null;
      _metaEditData = null;
      _metaDirty = false;
      _metaLoadedFor = data.self;
    }
    _metaResourceIdField = entityTypeT.toLowerCase() + 'id';
    _metaEntityType = entityTypeT;
    tabDefs.push({ key: 'meta', label: capTypeT + ' Details',
      content: '<div id="eg-meta-box"><div class="eg-loading">Loading\u2026</div></div>' });
    // "Versions List" — a real navigation link (not a tab switch) to the
    // raw Versions collection page, styled to match the pill tabs. Lets
    // users get to the full List/Grid/filterable view of all versions
    // (useful when there are many versions — a flat dropdown doesn't
    // scale, but the collection page's existing filter support does).
    // See plan.md "version-selector dropdown" for the design rationale.
    if (versionsCollD) {
      var versionsListHref = pageHref(_state.path.concat(['versions']), versionsCollD.url);
      var versionsListClick = guardedOnclick('navigateTo(\'versions\',' + JSON.stringify(versionsCollD.url) + ')');
      versionsListLinkHtml = '<a class="eg-doc-tab eg-doc-tab-link" href="' + esc(versionsListHref)
        + '" onclick="' + esc(versionsListClick) + '">Versions List (' + esc(String(versionsCollD.count)) + ')</a>';
    }

    _docSingular = docSingularD;
    _docPreviewLoaded = false;
    // Version-selector state (depth 4 only) — stashed globally since
    // onVersionSelectChange() runs from a later, independent DOM event, not
    // from within this render closure. See plan.md.
    _resModel = model;
    _resPath = _state.path.slice();
    _resCapType = capTypeT;
    _resDefaultData = data;
    _resCollKeys = collKeys;
    _resVersionsUrl = versionsUrlD;
    _resVersionsList = null;
    _resSelectedVersionId = 'default';
    _docActiveVersionData = data;
    _verEditSrc = null; _verEditData = null; _verDirty = false; _verEditVid = null;
    _verSelfUrl = ''; _verPanelId = null; _verHeaderLabel = null;

    // Standalone "Version:" dropdown (depth 4 only, when the resource has a
    // versions collection) — a separate control from the tab buttons, laid
    // out inline in the same row (see plan.md "version-selector dropdown").
    // Picking a version and picking which tab/panel to view are independent
    // choices; this control never switches tabs itself. Options are
    // populated asynchronously once the versions collection is fetched
    // (loadVersionsForSelect()); until then only "Default" is selectable.
    // Metadata (metaurl) is a per-Resource concept, not per-version, so the
    // version-selector has no effect while the Metadata tab is active —
    // start it disabled (and showing "N/A") if that's the tab being
    // restored on load (kept in sync afterwards by switchDocTab()). See
    // plan.md "Metadata tab disables version selector". The real "default"
    // option is always kept underneath the "N/A" placeholder (just not
    // selected) so switching away from Metadata later has something valid
    // to fall back to — see syncVersionSelectorForTab().
    var verSelDisabledD = (_state.docTab === 'meta');
    var versionSelectorHtml = versionsUrlD
      ? '<span class="eg-version-selector"><label for="eg-doc-version-select">Version:</label>'
        + '<select id="eg-doc-version-select"' + (verSelDisabledD ? ' disabled title="Metadata is the same for all versions"' : '')
        + ' onchange="onVersionSelectChange(this.value)">'
        + (verSelDisabledD ? '<option value="__na__" selected>N/A</option>' : '')
        + '<option value="default"' + (verSelDisabledD ? '' : ' selected') + '>' + esc(defaultOptionLabel(data)) + '</option>'
        + '</select></span>'
      : '';

    if (tabDefs.length || versionSelectorHtml) {
      // Restore the previously-active tab (from a Refresh) if it matches
      // one of this render's tabs; otherwise default to the first tab, as
      // before. See plan.md "Remember selected version + active tab".
      var initActiveIdx = 0;
      if (_state.docTab) {
        var restoredIdx = tabDefs.reduce(function(found, t, i) { return t.key === _state.docTab ? i : found; }, -1);
        if (restoredIdx >= 0) initActiveIdx = restoredIdx;
      }
      // Build the row's inner pieces in order: version selector first (if
      // present), then the tab buttons in their normal order (Document,
      // Version Details, Schema/Message/etc. Details), then the "Versions
      // List" navigation link last.
      var rowParts = [];
      if (versionSelectorHtml) rowParts.push(versionSelectorHtml);
      tabDefs.forEach(function(t, i) {
        rowParts.push('<button class="eg-doc-tab' + (i === initActiveIdx ? ' active' : '') + '" data-tab="' + esc(t.key)
          + '" onclick="switchDocTab(\'' + esc(t.key) + '\')">' + esc(t.label) + '</button>');
      });
      if (versionsListLinkHtml) rowParts.push(versionsListLinkHtml);
      html += '<div class="eg-doc-tabs">' + rowParts.join('') + '</div>';
      // Entity Data Editor action bar (Save/Undo/Delete) — rendered once
      // here, as a sibling of the tab panels rather than nested inside any
      // single one of them, so it stays visible no matter which tab
      // (Document/Version Details/Metadata) is currently active instead of
      // being hidden along with an inactive panel's display:none. Wrapped
      // in a stable-id slot so the version-selector dropdown can swap its
      // content in place (Default version's own bar vs. a differently-
      // selected version's own scoped bar) without moving the action bar
      // down to the bottom of the panel content — see
      // onVersionSelectChangeReal()/rerenderVersionPanel(). The Metadata
      // tab gets the same slot-swap treatment (see buildMetaActionBarHtml()
      // / switchDocTabReal()) — if it's the initially-active tab, show its
      // own Save/Undo bar here from the start instead of the (irrelevant,
      // permanently-disabled-until-a-Document-tab-edit) page-level bar.
      _dataEditorActionBarHtml = dataEditorActionBarHtml;
      var initialActionBarHtml = (tabDefs[initActiveIdx] && tabDefs[initActiveIdx].key === 'meta' && dataEditingNow)
        ? buildMetaActionBarHtml() : dataEditorActionBarHtml;
      html += '<div id="dataEditorActionBarSlot">' + initialActionBarHtml + '</div>';
      tabDefs.forEach(function(t, i) {
        html += '<div class="eg-doc-tab-panel" id="eg-doc-panel-' + esc(t.key) + '" data-tab="' + esc(t.key)
          + '"' + (i === initActiveIdx ? '' : ' style="display:none"') + '>' + t.content + '</div>';
      });
      pendingDocTabActivate = tabDefs[initActiveIdx].key;
    }
  } else if (attrKeys.length || dataEditingNow) {
    // Registry root (depth 0) / Group instance (depth 2) — no tab bar here
    // (only Resource/Version entities get the Document/Details tabs), just
    // the plain Properties table, same as before this feature existed.
    // Reuses the shared buildEntityPropsTableHtml() builder (identical
    // key-filter/sort/priority logic to attrKeys above) so this table gets
    // the same banding/badges/category-grouping treatment as every other
    // Property table, instead of duplicating the per-row rendering logic.
    var entityTypeP  = getSingularName(model, _state.path);
    var capTypeP = capitalize(entityTypeP);
    var propHeaderP = capTypeP + ' Details';
    html += dataEditorActionBarHtml
      + buildEntityPropsTableHtml(propsEntityData, propHeaderP, model, _state.path, collKeys, dataEditingNow);
  }

  html += '</div>';
  main.innerHTML = html;

  // The copy-URL button's tooltip may have been set (in renderBreadcrumbs(),
  // which runs before this tab bar exists in the DOM / before the model
  // fetch resolves) using a guessed default tab — refresh it now that the
  // real tab bar/active tab is in place, so e.g. a hasDocument resource
  // whose Document tab is the true default doesn't show a stale
  // $details-suffixed URL until the user manually clicks a tab.
  if (depthD === 4) refreshCopyLinkBtnTooltip();

  // Kick off the lazy fetch for whichever tab ended up default-selected
  // (Document or Details — the Default/Version Details panels already have
  // their content inline and need no fetch).
  if (pendingDocTabActivate === 'doc') { _docPreviewLoaded = true; loadDocumentPreview(); }
  else if (pendingDocTabActivate === 'meta') { _dataEditActiveKind = 'meta'; loadMetaDetails(); }
  // Populate the version-selector dropdown's options (Resource page only).
  if (_resVersionsUrl) loadVersionsForSelect();
  // Redirected here from a standalone "meta" page (depth 5) — land directly
  // on the Metadata tab instead of the generic default (Document/Version
  // Details), so the user sees the same content they were viewing.
  if (_pendingMetaTabOnLoad) {
    _pendingMetaTabOnLoad = false;
    if (document.querySelector('.eg-doc-tab[data-tab="meta"]')) switchDocTab('meta');
  }
}

// ---- Grid view for single entity (Registry / Group / Resource / Version) -

// Fetch and cache the model for a registry base URL (non-blocking)
function ensureModelCached(baseURL, cb) {
  var key = normalizeURL(baseURL);
  if (_modelCache[key]) { if (cb) cb(_modelCache[key]); return; }
  fetch(serverFetchBase(baseURL) + '/model')
    .then(function(r) { return r.json(); })
    .then(function(m) { _modelCache[key] = m; if (cb) cb(m); })
    .catch(function()  { _modelCache[key] = null; if (cb) cb(null); });
}

function ensureCapCached(baseURL, cb) {
  var key = normalizeURL(baseURL);
  if (_capCache.hasOwnProperty(key)) { if (cb) cb(_capCache[key]); return; }
  _capCache[key] = undefined; // mark in-flight
  fetch(serverFetchBase(baseURL) + '/capabilities')
    .then(function(r) { return r.json(); })
    .then(function(c) { _capCache[key] = c; if (cb) cb(c); })
    .catch(function()  { _capCache[key] = null; if (cb) cb(null); });
}

// Whether the current registry's cached /capabilities declares support for
// flag f (e.g. 'filter', 'sort', 'inline') — a standalone equivalent of
// renderJSONLeftPanel()'s local hasF() closure, usable from top-level
// functions (checkbox onchange handlers, sortRerender(), fbRerender()) that
// aren't nested inside that render call. See computeApplyDirty().
function capHasFlag(f) {
  var svBase = (_state.serverURL || window.location.origin).replace(/\/$/, '');
  var cap    = _capCache[normalizeURL(svBase)];
  var flags  = (cap && cap.flags) || [];
  return flags.indexOf(f) !== -1;
}

// Fetch and cache /capabilitiesoffered for a registry base URL (non-blocking).
// Used to know the declared type/enum/attributes of extension capabilities
// so the Capabilities editor can render/edit them generically.
function ensureOfferedCached(baseURL, cb) {
  var key = normalizeURL(baseURL);
  if (_offeredCache.hasOwnProperty(key)) { if (cb) cb(_offeredCache[key]); return; }
  _offeredCache[key] = undefined; // mark in-flight
  fetch(serverFetchBase(baseURL) + '/capabilitiesoffered')
    .then(function(r) { return r.json(); })
    .then(function(o) { _offeredCache[key] = o; if (cb) cb(o); })
    .catch(function()  { _offeredCache[key] = null; if (cb) cb(null); });
}

// Return singular entity type name using path depth + model lookup
// path: [] = Registry, [G,gId] = group, [G,gId,R,rId] = resource, [...,"versions",vId] = version
function getSingularName(model, path) {
  var len = path.length;
  if (len === 0) return 'Registry';
  if (len >= 6)  return 'version';
  var grpDef = model && model.groups && model.groups[path[0]];
  if (len === 2) return grpDef ? grpDef.singular : path[0].replace(/s$/, '');
  if (len >= 4) {
    var resDef = grpDef && grpDef.resources && grpDef.resources[path[2]];
    return resDef ? resDef.singular : path[2].replace(/s$/, '');
  }
  return 'entity';
}

// Returns true when the resource type at this path (a resource, its
// versions collection, or a specific version — depth >= 4) has
// hasdocument === true in the model. Document field names (e.g.
// "<singular>url"/"<singular>base64") are always keyed off the *resource's*
// singular name (see getSingularName(model, path.slice(0,4))), regardless
// of whether you're looking at the resource entity itself or one of its
// versions.
function resourceHasDocument(model, path) {
  if (!model || !model.groups || !path || path.length < 3) return false;
  var grpDef = model.groups[path[0]];
  var resDef = grpDef && grpDef.resources && grpDef.resources[path[2]];
  return !!(resDef && resDef.hasdocument);
}

// Attributes that are part of xRegistry structure — not shown as extensions
// specAttrLevel returns the SPEC_ATTRS sub-object for the given path depth.
// Resource entities (depth 4) blend resource + version attrs since GET /resource
// returns the default version flattened.
function specAttrLevel(path) {
  if (typeof SPEC_ATTRS === 'undefined') return {}; // specattrs.js not yet loaded
  var depth = path.length;
  if (depth === 0) return SPEC_ATTRS.registry;
  if (depth === 2) return SPEC_ATTRS.group;
  if (depth === 4) {
    // resource entity shows flattened default version — merge both sets
    var merged = {};
    var r = SPEC_ATTRS.resource, v = SPEC_ATTRS.version;
    for (var k in r) merged[k] = 1;
    for (var k in v) merged[k] = 1;
    return merged;
  }
  if (depth === 5) return SPEC_ATTRS.meta; // [G,gId,R,rId,"meta"]
  if (depth >= 6) return SPEC_ATTRS.version;
  return {}; // unrecognized depth — treat all attrs as extensions
}

// specAttrLevelName returns the SPEC_ATTRS_ORDER level name string for the path.
// For depth 4 (resource showing flattened version), returns 'version' since most
// user-visible attrs come from that level.
function specAttrLevelName(path) {
  var depth = path ? path.length : 0;
  if (depth === 0) return 'registry';
  if (depth === 2) return 'group';
  if (depth === 4) return 'version'; // resource shows flattened default version
  if (depth === 5) return 'meta';
  if (depth >= 6) return 'version';
  return null;
}

// ---- Icon resolution (model Group/Resource Type icon + instance override) --
//
// Per xRegistry spec (core/spec.md model schema), a Group Type or Resource
// Type definition MAY declare an "icon" URL — a static icon representing
// that *type*, independent of any icon an individual Group/Resource
// *instance* may set via its own spec-defined "icon" attribute. Per user
// request (2026-07-09): show model-declared Type icons in the Model/
// ModelSource viewer, and everywhere a Group/Resource list or header is
// shown, an instance's own icon (if set) wins over its Type's model icon.
// Versions have no separate Type-level icon concept — the owning
// Resource's icon (resolved the same way) is reused there.

// Model definition's icon for a Group Type (path[0] = group plural), or ''.
function modelGroupIcon(model, groupPlural) {
  var grpDef = model && model.groups && model.groups[groupPlural];
  return (grpDef && typeof grpDef.icon === 'string' && grpDef.icon.trim()) ? grpDef.icon : '';
}

// Model definition's icon for a Resource Type (path[0]=group plural,
// path[2]=resource plural), or ''.
function modelResourceIcon(model, groupPlural, resPlural) {
  var grpDef = model && model.groups && model.groups[groupPlural];
  var resDef = grpDef && grpDef.resources && grpDef.resources[resPlural];
  return (resDef && typeof resDef.icon === 'string' && resDef.icon.trim()) ? resDef.icon : '';
}

// Resolves the icon to display for a Group instance: its own `icon`
// attribute wins; otherwise falls back to the model's Group-type icon.
function resolveGroupIcon(model, groupPlural, groupData) {
  if (groupData && typeof groupData.icon === 'string' && groupData.icon.trim()) return groupData.icon;
  return modelGroupIcon(model, groupPlural);
}

// Resolves the icon to display for a Resource instance: its own `icon`
// attribute wins; otherwise falls back to the model's Resource-type icon.
function resolveResourceIcon(model, groupPlural, resPlural, resourceData) {
  if (resourceData && typeof resourceData.icon === 'string' && resourceData.icon.trim()) return resourceData.icon;
  return modelResourceIcon(model, groupPlural, resPlural);
}

// Builds a small <img> icon-thumbnail tag (or '' if no url) — shared by
// every icon-display spot added for this feature. onerror hides broken
// images rather than showing a broken-image glyph.
function iconThumbHtml(url, cssClass) {
  if (!url) return '';
  return '<img src="' + esc(url) + '" class="' + (cssClass || 'row-icon-thumb')
    + '" alt="" onerror="this.style.display=\'none\'">';
}

// specAttrOrder returns the SPEC_ATTRS_ORDER array for the given path, or [].
function specAttrOrder(path) {
  if (typeof SPEC_ATTRS_ORDER === 'undefined') return [];
  var name = specAttrLevelName(path);
  return (name && SPEC_ATTRS_ORDER[name]) || [];
}

// monoAttrLevel returns the MONO_ATTRS sub-object for the given path depth,
// mirroring specAttrLevel()'s per-depth mapping (including the depth-4
// resource+version merge, since GET /resource returns the flattened default
// version and can surface MONO_ATTRS entries from either set).
function monoAttrLevel(path) {
  if (typeof MONO_ATTRS === 'undefined') return {};
  var depth = path ? path.length : 0;
  if (depth === 0) return MONO_ATTRS.registry;
  if (depth === 2) return MONO_ATTRS.group;
  if (depth === 4) {
    var merged = {};
    var r = MONO_ATTRS.resource, v = MONO_ATTRS.version;
    for (var k in r) merged[k] = 1;
    for (var k in v) merged[k] = 1;
    return merged;
  }
  if (depth === 5) return MONO_ATTRS.meta;
  if (depth >= 6) return MONO_ATTRS.version;
  return {};
}

// isMonoSpecAttr returns true if key k should be rendered monospaced because
// it is both a spec-defined attribute at the current entity level AND is in
// MONO_ATTRS (string-typed spec attrs that are technical, not human prose).
// The dynamic "id" entry in MONO_ATTRS matches any <singular>id field.
// When path is supplied, the MONO_ATTRS set is resolved via monoAttrLevel(path)
// (depth-based, correctly handling the depth-4 resource+version merge). When
// path is omitted, falls back to matching specLevel by reference against
// SPEC_ATTRS's per-level objects (only valid for non-merged levels — depth 4's
// specLevel is always a freshly-built merged object, so it can never match by
// reference; callers at depth 4 MUST pass path).
function isMonoSpecAttr(k, specLevel, singular, path) {
  if (!isSpecAttr(k, specLevel, singular, null)) return false;
  var monoSet = null;
  if (typeof MONO_ATTRS !== 'undefined' && typeof SPEC_ATTRS !== 'undefined') {
    if (path) {
      monoSet = monoAttrLevel(path);
    } else {
      var levelNames = ['registry','group','resource','meta','version'];
      for (var i = 0; i < levelNames.length; i++) {
        if (SPEC_ATTRS[levelNames[i]] === specLevel) {
          monoSet = MONO_ATTRS[levelNames[i]] || {};
          break;
        }
      }
    }
  }
  if (!monoSet) return false;
  if (monoSet[k]) return true;
  // dynamic id pattern: MONO_ATTRS.*.id covers <singular>id fields
  if (monoSet.id && singular && k === singular + 'id') return true;
  return false;
}

// Given an attrs map and the actual data object at that same nesting level,
// returns a (possibly new) attrs map with any conditionally-added ifvalues
// sibling attributes merged in, based on the real attribute values present
// in `data`. Mirrors the server's Attributes.AddIfValuesAttributes()
// (registry/shared_model.go): for each attribute with a non-empty
// "ifvalues" map, if `data` has a matching value (case-insensitive) for
// that attribute's name, the corresponding siblingattributes are added to
// the returned map — and, since a newly-added sibling could itself declare
// ifvalues, those are checked too (recursively, matching the server's
// growing attrNames list). Non-destructive: never mutates the cached
// model's attrs map. Returns `attrs` unchanged (same reference) when there
// is nothing to resolve, or when `data` isn't available.
function resolveIfValuesAttrs(attrs, data) {
  if (!attrs || !data || typeof data !== 'object') return attrs;
  var names = Object.keys(attrs);
  var hasIfValues = names.some(function(n) { return attrs[n] && attrs[n].ifvalues; });
  if (!hasIfValues) return attrs;

  var result = Object.assign({}, attrs); // shallow copy — don't mutate the model
  for (var i = 0; i < names.length; i++) { // names grows as siblings are added
    var attr = result[names[i]];
    if (!attr || !attr.ifvalues || attr.name === '*') continue;
    var val = data[attr.name];
    if (val === undefined || val === null) continue;
    var valStr = String(val).toLowerCase();
    Object.keys(attr.ifvalues).forEach(function(ifValStr) {
      if (ifValStr.toLowerCase() !== valStr) return;
      var ifValueData = attr.ifvalues[ifValStr];
      var sibs = (ifValueData && ifValueData.siblingattributes) || {};
      Object.keys(sibs).forEach(function(sName) {
        if (!result[sName]) {
          result[sName] = sibs[sName];
          names.push(sName);
        }
      });
    });
  }
  return result;
}

// Returns the resolved (ifvalues-applied) top-level attributes map declared
// by the model for the entity at `path`'s depth — Registry root (depth 0),
// Group (depth 2), Resource/Version (depth >= 4, shared attributes map), or
// the Meta sub-entity (depth 5). Used by buildEntityPropsTableHtml() to
// surface every model-declared attribute in edit mode, even ones the
// current entity has no value for yet (see "Full Data Edit Mode" plan.md
// notes — "show all model attributes").
function topLevelAttrsMapFor(model, path, data) {
  if (!model) return {};
  var depth = path ? path.length : 0;
  var attrs;
  if (depth === 0) {
    attrs = model.attributes;
  } else if (depth === 2) {
    var gm = model.groups && model.groups[path[0]];
    attrs = gm && gm.attributes;
  } else if (depth === 5) {
    var gmM = model.groups && model.groups[path[0]];
    var rmM = gmM && gmM.resources && gmM.resources[path[2]];
    attrs = rmM && rmM.metaattributes;
  } else if (depth >= 4) {
    var gm2 = model.groups && model.groups[path[0]];
    var rm  = gm2 && gm2.resources && gm2.resources[path[2]];
    attrs = rm && rm.attributes;
  }
  if (!attrs) return {};
  return resolveIfValuesAttrs(attrs, data) || {};
}

// Live ifvalues reactivity for the top-level scalar edit handlers
// (dataEditFieldChange()/metaEditFieldChange()/verEditFieldChange()):
// called right after a top-level attribute's value is changed, with
// `beforeAttrs` being the resolved attrs map (topLevelAttrsMapFor()) taken
// *before* the value was written into `data`. Compares it against the
// freshly-resolved attrs map (using the new value already in `data`) and:
//   - deletes any data value for an attribute that is no longer resolved
//     (i.e. it only existed because the *old* value matched one of some
//     attribute's "ifvalues" and added it as a sibling — now stale, so its
//     data shouldn't linger and risk being rejected by the server on Save
//     as an undeclared attribute), and
//   - returns true if the set of resolved attribute names changed at all
//     (sibling attributes appeared and/or disappeared), so the caller
//     knows a full re-render is needed to show/hide the corresponding rows
//     immediately, instead of waiting for some unrelated re-render.
// Returns false (no data touched) when nothing changed, so ordinary edits
// keep their existing no-rerender/toggleRowDirty-only behavior.
function reconcileIfValuesOnChange(model, path, beforeAttrs, data) {
  if (!model || !data) return false;
  var afterAttrs = topLevelAttrsMapFor(model, path, data);
  var beforeNames = {};
  var changed = false;
  Object.keys(beforeAttrs || {}).forEach(function(n) {
    beforeNames[n] = true;
    if (n === '*') return;
    if (!afterAttrs.hasOwnProperty(n)) {
      // Only present before because of the old value's ifvalues match —
      // no longer valid, so drop its data along with its row.
      if (data.hasOwnProperty(n)) delete data[n];
      changed = true;
      return;
    }
    // Same attribute name present both before and after, but two different
    // ifvalues branches (e.g. two different values' siblingattributes) can
    // redeclare the same sibling name with a completely different
    // definition/shape (see e.g. CloudEvents' "protocol" attribute, whose
    // "protocoloptions" sibling has a distinct shape per protocol value).
    // A same-name check alone would miss this — compare object identity too
    // (topLevelAttrsMapFor()/resolveIfValuesAttrs() only shallow-copies, so
    // unrelated/always-present attrs keep the same reference across calls;
    // only re-resolved ifvalues siblings get a new object each time their
    // matching branch changes). Stale data no longer matching the new
    // definition's shape must be dropped, same as a disappearing attribute.
    if (beforeAttrs[n] !== afterAttrs[n]) {
      if (data.hasOwnProperty(n)) delete data[n];
      changed = true;
    }
  });
  if (!changed) {
    changed = Object.keys(afterAttrs).some(function(n) { return n !== '*' && !beforeNames[n]; });
  }
  return changed;
}

// getAttr returns the full Attribute definition object from the model for
// the given attribute key path (array) within an entity at entityPath depth.
// attrKeyPath is an array for nested traversal, e.g. ['myattr'] or ['obj','child'].
// Falls back to the '*' wildcard entry for undeclared extension attributes.
// Returns null only on model compliance violation (should not happen in practice).
//
// `data` (optional) is the actual entity JSON at entityPath's depth. When
// provided, ifvalues conditional sibling-attribute rules are evaluated (see
// resolveIfValuesAttrs()) at every level of the traversal, so attributes
// that only exist because a sibling's value matched one of its "ifvalues"
// keys are found too — not just the static model declaration.
function getAttr(model, entityPath, attrKeyPath, data) {
  if (!model || !attrKeyPath || attrKeyPath.length === 0) return null;
  var depth = entityPath ? entityPath.length : 0;

  // Find the top-level attributes map for this entity depth
  var attrs;
  if (depth === 0) {
    attrs = model.attributes;
  } else if (depth === 2) {
    var gm = model.groups && model.groups[entityPath[0]];
    attrs = gm && gm.attributes;
  } else if (depth === 5) {
    // Meta entity ([G,gId,R,rId,"meta"]) — the model exposes a dedicated
    // metaattributes map (distinct from the resource/version-flattened
    // "attributes" map), so this must be read separately.
    var gmM = model.groups && model.groups[entityPath[0]];
    var rmM = gmM && gmM.resources && gmM.resources[entityPath[2]];
    attrs = rmM && rmM.metaattributes;
  } else if (depth >= 4) {
    var gm2 = model.groups && model.groups[entityPath[0]];
    var rm  = gm2 && gm2.resources && gm2.resources[entityPath[2]];
    attrs = rm && rm.attributes;
  }
  if (!attrs) return null;

  // Traverse attrKeyPath, following .attributes for nested objects
  var attr = null;
  var curData = data;
  for (var i = 0; i < attrKeyPath.length; i++) {
    var key = attrKeyPath[i];
    var resolvedAttrs = resolveIfValuesAttrs(attrs, curData);
    attr = resolvedAttrs[key] || resolvedAttrs['*'] || null;
    if (!attr) return null;
    if (i < attrKeyPath.length - 1) {
      attrs = attr.attributes;
      if (!attrs) return null;
      curData = (curData && typeof curData === 'object') ? curData[key] : null;
    }
  }
  return attr;
}

// Convenience wrapper — returns just the type string (or null).
function getAttrType(model, entityPath, attrKeyPath, data) {
  var attr = getAttr(model, entityPath, attrKeyPath, data);
  return attr ? (attr.type || null) : null;
}

// getAttr()'s wildcard ('*') fallback for undeclared extension attributes
// is always just {type:'any'} — the model has no way to know in advance
// what shape a given extension's value will actually take. Without this,
// an "any"-typed extension whose value happens to be a map/array/object
// (e.g. one added via the "+ Add Extension" type picker's map/array/object
// choice — see renderExtTypeValueWidgetHtml()) would fall through to the
// plain scalar input/display, which just stringifies it — showing the
// useless "[object Object]"/"[object Array]" instead of an editable
// tree. This synthesizes an equivalent map/array attr (item type "any",
// so its own children get the same any-type treatment recursively) purely
// from the actual runtime value shape, whenever the declared attr doesn't
// already commit to a complex type. Used by every complex-value edit
// render site (buildPropsRowsHtml, renderMetaTable, and
// renderEditableComplexValue's map/array/object branches for their own
// nested values) so this applies uniformly no matter how deep the
// wildcard-typed value is nested.
// Like effectiveEditAttr(), but for one specific nested location
// (keyPath) — checks _shapeHints first (an explicit map/object choice
// recorded at add-time by dataEditMapAddEntry()/dataEditArrayAddItem()/
// dataEditObjAddKey()) before falling back to inferring purely from the
// declared attr + runtime value. Needed because an empty {} can't
// otherwise be told apart as "map" vs "object" once the value has no
// other content yet.
function effectiveEditAttrAtPath(keyPath, attr, val) {
  var hint = _shapeHints[JSON.stringify(keyPath)];
  if (hint) return hint;
  return effectiveEditAttr(attr, val);
}

// True for a top-level (or Meta tab) attribute whose type is still fully
// ambiguous ("any"/undeclared, with no _shapeHints choice recorded yet —
// see effectiveEditAttrAtPath()) AND which has no value at all yet. Used
// by buildPropsRowsHtml()/renderMetaTable() to show the "Set Value" type
// picker (see renderComplexAddButtonHtml()/dataEditSetAnyType()) instead
// of a plain (always-string) text input for a never-set "any"-typed
// attribute — otherwise there'd be no way to make it e.g. a boolean,
// number, or complex value in the first place. Once a value/hint exists,
// effectiveEditAttr()'s own runtime-type inference takes over as usual.
function isAnyTypeUnset(attr, val) {
  return (!attr || !attr.type || attr.type === 'any') && (val === null || val === undefined);
}

function effectiveEditAttr(attr, val) {
  if (attr && (attr.type === 'map' || attr.type === 'object' || attr.type === 'array')) return attr;
  if (val !== null && typeof val === 'object') {
    // Array runtime shape is unambiguous. A plain object, though, could be
    // either a homogeneous "map" or a free-form "object" — JSON alone
    // can't tell us which the user meant. Default to "object" (a
    // wildcarded, freely-extensible bag of any-typed keys) since that's
    // the more general shape and matches how "any"-typed extensibility
    // already works everywhere else (e.g. the top-level "+ Add
    // Extension" row) — this also keeps the type pill accurate ("Object"
    // instead of "Map, Any"). A genuinely-declared map<T> never reaches
    // this branch at all (it's caught by the first line above).
    return Array.isArray(val)
      ? {type: 'array', item: {type: 'any'}}
      : {type: 'object', attributes: {'*': {type: 'any'}}};
  }
  // Scalar runtime-type inference for "any"/undeclared attrs — e.g. an
  // any-typed map/array/object entry whose actual value was set via the
  // type-picker add-button (renderComplexAddButtonHtml()) with a concrete
  // scalar type (boolean/integer/etc). The item's *declared* type stays
  // "any" forever (that's the model's own wildcard/"any" declaration), so
  // without this both the type pill and the scalar input widget (checkbox
  // vs number vs text, see renderEditableScalarInput()/
  // renderEditableScalarInputAtPath()) would keep showing "Any"/a plain
  // text box even after a concrete type has clearly been chosen. Can't
  // tell integer vs decimal apart at runtime (both are just JS numbers) —
  // "decimal" is used as the generic numeric fallback, matching
  // renderEditableScalarInput()'s own no-attr fallback.
  if ((!attr || !attr.type || attr.type === 'any') && val !== null && val !== undefined) {
    if (typeof val === 'boolean') return Object.assign({}, attr, {type: 'boolean'});
    if (typeof val === 'number') return Object.assign({}, attr, {type: 'decimal'});
    if (typeof val === 'string') return Object.assign({}, attr, {type: 'string'});
  }
  return attr;
}

// Display name for one attribute type, e.g. "uinteger" -> "Uinteger" — used
// by typeLabelFor()/typePillHtml() below.
function typeDisplayName(t) {
  return t ? (t.charAt(0).toUpperCase() + t.slice(1)) : 'Any';
}

// One-line type description for a value's "type pill" (see typePillHtml()
// below) — just the type name for a scalar or object (object is keyed/
// heterogeneous, so there's no single "item type" to add), or "Type,
// ItemType" for a map/array, showing one level of nesting so the user
// knows what to expect/enter without fully expanding it (e.g. "Array,
// Map" for an array of maps — the map's own contents get their own pill,
// one level deeper, once expanded/added).
function typeLabelFor(attr) {
  var t = (attr && attr.type) || 'any';
  var label = typeDisplayName(t);
  if (t === 'map' || t === 'array') {
    label += ', ' + typeDisplayName(attr.item && attr.item.type);
  }
  return label;
}

// Small faded "type" pill rendered directly above a value's editor (see
// buildPropsRowsHtml()/renderMetaTable() for top-level attributes, and
// renderEditableComplexValue()/its scalar-entry call sites for nested map/
// array/object values) — the Property/Meta tables have no separate "Type"
// column, so without this there's no way to tell e.g. an "any"-typed
// extension's actual runtime type, or what an empty map/array's items are
// supposed to be, before typing a value.
function typePillHtml(attr) {
  return '<div class="xr-type-pill">' + esc(typeLabelFor(attr)) + '</div>';
}

// Resolves the "effective" type for one attribute name within an already-resolved
// attrs map: prefers an explicit (non-wildcard) declaration; otherwise, if the '*'
// wildcard itself declares a concrete type (i.e. not "any"/absent), that type still
// applies — a model author who writes `"*": {type: "url"}` is making a real, deliberate
// schema statement ("every undeclared attribute here is a URL"), so it should still
// drive monospace formatting. Only a fully generic/untyped wildcard (type "any" or
// missing) is treated as "unknown" and returns null.
function typeFromAttrsMap(attrs, key) {
  if (!attrs) return null;
  if (attrs[key]) return attrs[key].type || null;
  var wc = attrs['*'];
  if (wc && wc.type && wc.type !== 'any') return wc.type;
  return null;
}

// Like getAttrType but, for undeclared (wildcard-only) attributes, only returns a
// type when the '*' wildcard itself declares something more specific than "any"
// (see typeFromAttrsMap). Used for monospace decisions so that fully generic
// extension attributes (wildcard type "any"/absent) aren't incorrectly monospaced,
// while extensions under a concretely-typed wildcard (e.g. "*": {type: "url"}) are.
//
// `data` (optional) is the actual entity JSON at entityPath's depth, used to
// resolve ifvalues conditional sibling attributes (see resolveIfValuesAttrs())
// before looking up `key` — e.g. an Endpoint's "envelopeoptions" attribute
// only exists in the effective attrs map when that entity's "envelope"
// attribute is set to "CloudEvents/1.0" (see endpoint/model.json).
function getExplicitAttrType(model, entityPath, key, data) {
  if (!model || !key) return null;
  var depth = entityPath ? entityPath.length : 0;
  var attrs;
  if (depth === 0) {
    attrs = model.attributes;
  } else if (depth === 2) {
    var gm = model.groups && model.groups[entityPath[0]];
    attrs = gm && gm.attributes;
  } else if (depth === 5) {
    // Meta entity ([G,gId,R,rId,"meta"]) — the model exposes a dedicated
    // metaattributes map (distinct from the resource/version-flattened
    // "attributes" map), so this must be read separately.
    var gmM = model.groups && model.groups[entityPath[0]];
    var rmM = gmM && gmM.resources && gmM.resources[entityPath[2]];
    attrs = rmM && rmM.metaattributes;
  } else if (depth >= 4) {
    var gm2 = model.groups && model.groups[entityPath[0]];
    var rm  = gm2 && gm2.resources && gm2.resources[entityPath[2]];
    attrs = rm && rm.attributes;
  }
  attrs = resolveIfValuesAttrs(attrs, data);
  return typeFromAttrsMap(attrs, key);
}
// Like getExplicitAttrType but supports a multi-segment path for nested attrs (e.g.
// ["deprecated", "effective"]), by walking .attributes at each intermediate level.
// Intermediate segments may fall back to the '*' wildcard purely for structural
// traversal (needed to reach the nested attrs map at all). The FINAL segment's type
// is resolved via typeFromAttrsMap, so a nested field explicitly named in the model
// gets its own type, and a nested extension field only gets a type when the nested
// '*' wildcard itself declares something concrete (not "any").
// This is what makes nested monospace formatting (e.g. for "deprecated"'s children)
// fully model-driven: it reads the already-cached runtime /model, so it automatically
// works for any spec-defined or model-defined complex attribute, without hardcoding
// attribute names anywhere in the UI or in generated code.
//
// `data` (optional) is the actual entity JSON at entityPath's depth — the same
// root object attrKeyPath is addressing into. At every level of the
// traversal (including the final segment) ifvalues are resolved against the
// corresponding nested data object (see resolveIfValuesAttrs()), so e.g.
// an Endpoint's "protocoloptions" (only present when "protocol" ==
// "AMQP/1.0") gets its correct type, and any *nested* ifvalues within it
// would be resolved too.
function getExplicitAttrTypeAtPath(model, entityPath, attrKeyPath, data) {
  if (!model || !attrKeyPath || attrKeyPath.length === 0) return null;
  var depth = entityPath ? entityPath.length : 0;
  var attrs;
  if (depth === 0) {
    attrs = model.attributes;
  } else if (depth === 2) {
    var gm = model.groups && model.groups[entityPath[0]];
    attrs = gm && gm.attributes;
  } else if (depth === 5) {
    // Meta entity ([G,gId,R,rId,"meta"]) — the model exposes a dedicated
    // metaattributes map (distinct from the resource/version-flattened
    // "attributes" map), so this must be read separately.
    var gmM = model.groups && model.groups[entityPath[0]];
    var rmM = gmM && gmM.resources && gmM.resources[entityPath[2]];
    attrs = rmM && rmM.metaattributes;
  } else if (depth >= 4) {
    var gm2 = model.groups && model.groups[entityPath[0]];
    var rm  = gm2 && gm2.resources && gm2.resources[entityPath[2]];
    attrs = rm && rm.attributes;
  }
  if (!attrs) return null;
  var attr = null;
  var curData = data;
  for (var i = 0; i < attrKeyPath.length; i++) {
    var key = attrKeyPath[i];
    var isLast = (i === attrKeyPath.length - 1);
    var resolvedAttrs = resolveIfValuesAttrs(attrs, curData);
    if (isLast) {
      return typeFromAttrsMap(resolvedAttrs, key);
    }
    attr = resolvedAttrs[key] || resolvedAttrs['*'] || null;
    if (!attr) return null;
    attrs = attr.attributes;
    if (!attrs) return null;
    curData = (curData && typeof curData === 'object') ? curData[key] : null;
  }
  return null;
}
// Handles the two dynamic name patterns from OrderedSpecProps:
//   "id"          → matches <singular>id  (e.g. "messageid", "registryid")
//   "$RESOURCE*"  → matches <resourceSingular>, <resourceSingular>url,
//                   <resourceSingular>base64, <resourceSingular>proxyurl
function isSpecAttr(k, specLevel, singular, resourceSingular) {
  if (specLevel[k]) return true;
  if (specLevel['id'] && singular && k === singular + 'id') return true;
  if (resourceSingular) {
    if (specLevel['$RESOURCE']         && k === resourceSingular)             return true;
    if (specLevel['$RESOURCEurl']      && k === resourceSingular + 'url')     return true;
    if (specLevel['$RESOURCEbase64']   && k === resourceSingular + 'base64')  return true;
    if (specLevel['$RESOURCEproxyurl'] && k === resourceSingular + 'proxyurl') return true;
    // Version entities also carry the owning Resource's own <resourceSingular>id
    // (e.g. "fileid" on a Version of a "file" Resource) — a spec-defined
    // identity field (see registry/entity.go GetResourceSingular()+"id"),
    // distinct from the Version's own <singular>id ("versionid").
    if (resourceSingular !== singular && k === resourceSingular + 'id') return true;
  }
  return false;
}

var INFO_ICON = '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" '
  + 'viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" '
  + 'stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">'
  + '<circle cx="12" cy="12" r="10"/>'
  + '<line x1="12" y1="8" x2="12" y2="8"/>'
  + '<line x1="12" y1="12" x2="12" y2="16"/>'
  + '</svg>';

// collectionTile()/groupTileHTML() were the shared tile-rendering helpers
// for Grid view (Registry root/Group entity pages via renderEntityGrid(),
// and the Home 'types' cross-registry Group Types page via
// renderHomeFlatGrid()) — both removed, see plan.md "Grid view removed".

function openDocument(singular, itemData) {
  var data = itemData || _lastData;
  if (!data) return;
  var key = singular.toLowerCase();

  // 1. URL variant — open directly in new tab
  if (data[key + 'url']) {
    window.open(data[key + 'url'], '_blank');
    return;
  }

  // 2. Base64 variant — decode and open as blob
  if (data[key + 'base64']) {
    try {
      var ct = data.contenttype || 'application/octet-stream';
      var binary = atob(data[key + 'base64']);
      var bytes = new Uint8Array(binary.length);
      for (var i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
      var blob = new Blob([bytes], {type: ct});
      window.open(URL.createObjectURL(blob), '_blank');
    } catch(e) { showToast('Error decoding document'); }
    return;
  }

  // 3. Inline JSON variant
  if (data[key] !== undefined && data[key] !== null) {
    var json = JSON.stringify(data[key], null, 2);
    var blob = new Blob([json], {type: 'application/json'});
    window.open(URL.createObjectURL(blob), '_blank');
    return;
  }

  // 4. Fallback — strip $details from self
  if (data.self) {
    window.open(data.self.replace(/\$details$/, ''), '_blank');
    return;
  }

  showToast('Document not available');
}

// ATTR_LABELS replaced by generated LABEL_ATTRS in specattrs.js (see AttrInternals.uiLabel).
// labelFor returns the display label for attribute key k.
// When specLevel+singular are provided, LABEL_ATTRS is only applied for genuine
// spec-defined attrs at that entity level — extension attrs with coincidentally
// matching names fall back to the raw key, avoiding misleading labels.
function labelFor(k, specLevel, singular) {
  if (typeof LABEL_ATTRS !== 'undefined' && LABEL_ATTRS[k]) {
    if (!specLevel || isSpecAttr(k, specLevel, singular || '', null)) {
      return LABEL_ATTRS[k];
    }
  }
  return k;
}

var _toastTimer = null;
function showToast(msg) {
  var t = el('eg-toast');
  if (!t) {
    t = document.createElement('div');
    t.id = 'eg-toast';
    document.body.appendChild(t);
  }
  t.textContent = msg;
  t.className = 'eg-toast eg-toast-show';
  if (_toastTimer) clearTimeout(_toastTimer);
  _toastTimer = setTimeout(function() {
    t.className = 'eg-toast';
    _toastTimer = null;
  }, 2000);
}

function copyBtn(label, value) {
  var safeLabel = label.replace(/'/g, "\\'");
  return '<button class="eg-link-btn eg-copy-btn" title="' + esc(value) + '" '
       + 'data-copy="' + esc(value) + '" '
       + 'onclick="egCopy(this.dataset.copy,\'' + safeLabel + '\');">'
       + esc(label) + '</button>';
}

function egCopy(text, label) {
  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard.writeText(text)
      .then(function() { showToast(label + ' copied'); })
      .catch(function() { egCopyFallback(text, label); });
  } else {
    egCopyFallback(text, label);
  }
}

function egCopyFallback(text, label) {
  var ta = document.createElement('textarea');
  ta.value = text;
  ta.style.cssText = 'position:fixed;top:-9999px;left:-9999px';
  document.body.appendChild(ta);
  ta.select();
  try {
    document.execCommand('copy');
    showToast(label + ' copied');
  } catch(e) {
    showToast('Copy failed');
  }
  document.body.removeChild(ta);
}

// Recursively render any JSON value as styled HTML with copyable leaves.
// model/entityPath/keyPath (all optional) let this look up the actual attribute
// type from the runtime-cached model schema so nested scalar fields with a
// non-string type (e.g. "deprecated.effective" being a timestamp, or
// "deprecated.alternative" being a url) render in monospace, generically for
// ANY spec-defined or model-defined complex attribute — no hardcoded attribute
// names required. keyPath tracks the attribute path from the top of the value
// being rendered (e.g. ["deprecated", "effective"]); it is intentionally NOT
// extended into array items (array item types aren't addressable this way),
// so scalar array items keep their previous unstyled rendering.
//
// rootData (optional) is the top-level entity JSON that keyPath is relative
// to — i.e. the same object passed as `data` into buildPropsRowsHtml()/
// renderMetaTable() — kept constant through the recursion (unlike `val`,
// which changes at each nesting level) so getExplicitAttrTypeAtPath() can
// resolve ifvalues conditional sibling attributes at every level by
// drilling into rootData itself via keyPath.
function renderValueTree(val, depth, model, entityPath, keyPath, rootData) {
  var attrType = keyPath ? getExplicitAttrTypeAtPath(model, entityPath, keyPath, rootData) : null;
  var forceMono = attrType !== null && attrType !== 'string';
  function leaf(raw, display) {
    var cls = forceMono ? ' class="eg-mono"' : '';
    return '<span' + cls + '>' + display + '</span>';
  }
  if (val === null)              return leaf('null', '<span class="vt-null">null</span>');
  if (val === undefined)         return '<span class="vt-null">undefined</span>';
  if (typeof val === 'boolean')  return leaf(val, String(val));
  if (typeof val === 'number')   return leaf(val, String(val));
  if (typeof val === 'string')   return leaf(val, esc(val));

  if (Array.isArray(val)) {
    if (val.length === 0) return '<span class="vt-empty">empty</span>';
    var items = val.map(function(item, idx) {
      var isComplex = item !== null && typeof item === 'object';
      var sep    = (isComplex && idx > 0) ? '<div class="vt-arr-sep"></div>' : '';
      var badge  = '<span class="vt-arr-idx">[' + idx + ']</span>';
      return sep + '<div class="vt-arr-item">'
           + badge + renderValueTree(item, depth) + '</div>';
    });
    return '<div class="vt-arr">' + items.join('') + '</div>';
  }

  // object / map — rendered as its own compact two-column grid (key |
  // value) scoped to just this object's keys, so its column width is
  // independent of any sibling/parent grid. Any nested complex value
  // (isComplex) recurses into another such grid inside its own value
  // cell, which is why nesting no longer needs a manual depth-based
  // indent or a connecting border-left — the column offset itself shows
  // the nesting.
  var keys = Object.keys(val).sort();
  if (keys.length === 0) return '<span class="vt-empty">empty</span>';
  var rows = keys.map(function(k) {
    var child = val[k];
    var isComplex = child !== null && typeof child === 'object';
    var childKeyPath = keyPath ? keyPath.concat([k]) : null;
    var childHtml = renderValueTree(child, depth + 1, model, entityPath, childKeyPath, rootData);
    return '<div class="vt-kv' + (isComplex ? ' vt-kv-block' : '') + '">'
         + '<span class="vt-key">' + esc(k) + ':</span>'
         + '<span class="vt-kv-value">' + childHtml + '</span>'
         + '</div>';
  });
  return '<div class="vt-obj">' + rows.join('') + '</div>';
}

function copyable(text) {
  return '<span class="eg-value">' + esc(text) + '</span>';
}

function copyableMonospace(text) {
  return '<span class="eg-value eg-mono">' + esc(text) + '</span>';
}

// Renders a real, clickable <a> for a URL-shaped Property-table scalar
// value — same-server URLs (self, metaurl, versionsurl, defaultversionurl,
// any URI/URL-typed spec or extension attr, ...) navigate within the SPA
// (mirrors syntaxHighlight()'s JSON-view linkification, setting apiURL to
// the value's own raw URL verbatim — all query params, not just
// filter= — so the displayed href and the actual navigation target both
// match exactly what the server returned); other (external) URLs open in
// a new tab. Detection is purely content-based (any string starting with
// http(s)://), the same approach syntaxHighlight() uses for JSON view —
// this way it applies uniformly to every URI/URL-typed field without
// needing model-driven type resolution everywhere (which isn't available
// at the Meta level — see renderMetaTable()).
function renderUrlLinkValue(rawText, isMono) {
  var svBase = serverBase();
  var urlPath = rawText.split('?')[0].split('#')[0].replace(/\/?\$details$/, '');
  var href, target = '', onclick = '', displayText = rawText;
  if (urlPath.indexOf(svBase) === 0) {
    var rel      = urlPath.slice(svBase.length).replace(/^\//, '');
    var segments = rel ? rel.split('/') : [];
    // filters is set here (in addition to apiURL) purely so buildURL()'s
    // apiurl-vs-filter dedup logic sees them already matching and skips
    // appending a redundant top-level filter= — see the longer comment
    // in syntaxHighlight()'s equivalent fakeSt construction.
    var fakeSt   = Object.assign({}, _state, {
      view: 'table', section: 'data', path: segments,
      apiURL: rawText, filters: filtersFromUrl(rawText)
    });
    href    = buildURL(fakeSt);
    // guardedOnclick() lets ctrl/cmd/shift/middle-click fall through to the
    // browser's native "open in new tab" on the real href above, instead of
    // always intercepting the click for in-app navigation.
    var navExpr = 'navigateJsonUrl(\'' + rawText.replace(/\\/g,'\\\\').replace(/'/g,"\\'") + '\')';
    onclick = ' onclick="' + esc(guardedOnclick(navExpr)) + '"';
    // When browsing through the proxy, rawText is our local
    // /xrproxy/<base64> URL — show the real remote URL instead (display
    // text only; href/onclick above must keep using the proxy URL so
    // clicking still navigates correctly).
    displayText = toRealURL(rawText);
  } else {
    href   = rawText;
    target = ' target="_blank" rel="noopener"';
  }
  var cls = 'eg-value' + (isMono ? ' eg-mono' : '');
  return '<a class="' + cls + '" href="' + esc(href) + '"' + target + onclick + '>' + esc(displayText) + '</a>';
}

// Renders a Property-table scalar value: a clickable link (via
// renderUrlLinkValue()) if it looks like a URL, otherwise plain
// copyable(Monospace) text.
function renderScalarValue(val, isMono) {
  var text = String(val);
  if (/^https?:\/\//.test(text)) return renderUrlLinkValue(text, isMono);
  return isMono ? copyableMonospace(text) : copyable(text);
}

// Renders a compact pill badge for a boolean Property-table value, instead
// of plain "true"/"false" text. False isn't a "bad" state for most spec
// booleans (e.g. isdefault:false is perfectly normal), so both use a
// neutral/positive palette rather than green/red.
// falseIcon lets callers swap the default "✕ false" for a custom icon-only
// badge (no "false" text) — used for formatvalidated/compatibilityvalidated,
// where false doesn't mean "failed", just "not validated"/"unknown"; adding
// "false" next to the icon reads as a negative result, which is misleading.
function renderBoolBadge(val, falseIcon) {
  return '<span class="eg-bool-badge ' + (val ? 'eg-bool-true' : 'eg-bool-false') + '">'
       + (val ? '\u2713 true' : (falseIcon ? falseIcon : '\u2715 false')) + '</span>';
}

// DOM-element twin of renderBoolBadge(), for callers building elements via
// document.createElement() rather than innerHTML strings (e.g. the
// Capabilities/Model editors' read-only boolean displays). `label` lets
// callers override the badge text (e.g. an em-dash for "unspecified"),
// keeping the same neutral pill styling.
function boolBadgeEl(val, label) {
  var span = document.createElement('span');
  span.className = 'eg-bool-badge ' + (val ? 'eg-bool-true' : 'eg-bool-false');
  span.textContent = label !== undefined ? label : (val ? '\u2713 true' : '\u2715 false');
  return span;
}

// Rough human relative-time string ("3 days ago", "in 2 hours", "just now")
// for a timestamp value's hover tooltip. Deliberately simple/approximate —
// it's a hint, not a precise duration, and keeps the SPA dependency-free.
// Returns null if the string doesn't parse as a date.
function relativeTimeFromNow(iso) {
  var t = Date.parse(iso);
  if (isNaN(t)) return null;
  var diffMs = Date.now() - t;
  var future = diffMs < 0;
  var sec = Math.round(Math.abs(diffMs) / 1000);
  var units = [['year', 365 * 24 * 3600], ['month', 30 * 24 * 3600],
    ['day', 24 * 3600], ['hour', 3600], ['minute', 60]];
  for (var i = 0; i < units.length; i++) {
    var n = Math.floor(sec / units[i][1]);
    if (n >= 1) {
      var label = n + ' ' + units[i][0] + (n === 1 ? '' : 's');
      return future ? ('in ' + label) : (label + ' ago');
    }
  }
  return 'just now';
}

// Renders a Property-table value known to be a "timestamp"-typed attribute
// (see SPEC_ATTRS/model attr type "timestamp") — same color/size as any
// other property value (see .cell-timestamp-prop; the muted-gray/small
// treatment is reserved for collection List view's Created/Modified
// columns via .cell-timestamp), plus a relative-time hover tooltip. isMono
// controls monospace font (spec timestamps are always monospace already
// via MONO_ATTRS/type checks upstream, but kept as a parameter for
// consistency with callers).
function formatTimestampValue(rawText, isMono) {
  var rel = relativeTimeFromNow(rawText);
  var titleAttr = rel ? ' title="' + esc(rel) + '"' : '';
  return '<span class="eg-value cell-timestamp-prop' + (isMono ? ' eg-mono' : '') + '"' + titleAttr + '>'
       + esc(rawText) + '</span>';
}

// Fixed, ordered category buckets for grouping spec-defined attributes in
// List view's Property tables (see plan.md "Property table categories").
// Only used for keys confirmed spec-defined at the current entity level
// (via isSpecAttr()) — extension/custom attrs always land in a separate,
// always-last "Extensions" bucket (see groupPropsByCategory()), never
// subdivided further.
var PROP_CATEGORY_DEFS = [
  { label: 'General',            keys: {name:1, description:1, documentation:1, icon:1, labels:1} },
  { label: 'Identity',           keys: {id:1, versionid:1, xid:1, self:1, shortself:1},
    order: ['resourceid', 'id', 'versionid', 'xid', 'self', 'shortself'] },
  { label: 'Versioning & State',
    keys: {specversion:1, epoch:1, isdefault:1, deprecated:1, readonly:1, ancestorid:1,
           xref:1, defaultversionid:1, defaultversionurl:1, defaultversionsticky:1, constraints:1} },
  { label: 'Content',
    keys: {contenttype:1, format:1, formatvalidated:1, formatvalidatedreason:1,
           compatibility:1, compatibilityvalidated:1, compatibilityvalidatedreason:1,
           meta:1, metaurl:1, model:1, modelsource:1, capabilities:1} },
  { label: 'Timestamps',         keys: {createdat:1, modifiedat:1} }
];

// Buckets an already-filtered/sorted Property-table `keys` array into the
// labeled categories above, for List view. Spec-defined attrs (per
// isSpecAttr(), which also matches dynamic patterns like <singular>id and
// $RESOURCE*) go into their matching bucket, falling back to "Content" for
// the rare spec attr not explicitly listed; non-spec/custom attrs always
// go into a final "Extensions" bucket. Returns null — meaning "render a
// flat, ungrouped list, like before this feature existed" — when there's
// no spec-level info to categorize against, or when everything collapses
// into a single non-empty bucket anyway (a lone category header wouldn't
// help readability there).
//
// Within a bucket, keys are re-sorted per that category's own `order`
// array (when it has one) rather than keeping the caller's pre-sorted
// order — e.g. Identity always wants "id, xid, self, shortself"
// regardless of how buildEntityPropsTableHtml()'s generic priority sort
// happened to place them. The dynamic <singular>id field (e.g. "fileid")
// is treated as "id" for ordering purposes even though its literal key
// differs. Categories without an `order` array keep the incoming order.
// A few "Versioning & State" attributes are useful enough to end-users
// (not just xReg plumbing) that Domain view keeps them visible even
// though the rest of that category is hidden — e.g. the Meta tab's
// defaultversionid/defaultversionsticky/readonly (which version is
// active, whether changes are locked out), and the Version Details
// table's isdefault/ancestorid (whether this is the default version, and
// which version it descends from).
var DOMAIN_FOCUSED_KEEP_KEYS = {defaultversionid:1, defaultversionsticky:1, readonly:1,
  isdefault:1, ancestorid:1};

function groupPropsByCategory(keys, specLevel, singular, resourceSingular, editable, extLabel) {
  if (!specLevel) return null;
  // Domain view (the default; View mode only, see effectiveXregFocused()):
  // drop the Identity and Versioning & State buckets entirely, and use
  // the caller-supplied extLabel ("<Singular> Metadata") in place of the
  // literal "Extensions" label. Edit mode is never affected.
  var domainFocused = !effectiveXregFocused() && !editable;
  var buckets = PROP_CATEGORY_DEFS.map(function(def) { return { label: def.label, keys: [], order: def.order }; });
  var identityBucket = buckets.filter(function(b) { return b.label === 'Identity'; })[0];
  var contentBucket = buckets.filter(function(b) { return b.label === 'Content'; })[0];
  var extBucket = { label: (domainFocused && extLabel) ? extLabel : 'Extensions', keys: [], ext: true };
  keys.forEach(function(k) {
    if (!isSpecAttr(k, specLevel, singular, resourceSingular)) { extBucket.keys.push(k); return; }
    // Dynamic <singular>id pattern (e.g. "fileid") — isSpecAttr() matches it via
    // specLevel['id'], but it won't equal the literal "id" key any bucket lists,
    // so it needs its own check here; it's always an Identity-style field.
    if (specLevel['id'] && singular && k === singular + 'id') { identityBucket.keys.push(k); return; }
    // Dynamic <resourceSingular>id pattern on a Version (e.g. "fileid") — same
    // Identity treatment as the singular>id case above, just referring to the
    // owning Resource's id instead of the Version's own.
    if (resourceSingular && resourceSingular !== singular && k === resourceSingular + 'id') {
      identityBucket.keys.push(k); return;
    }
    for (var i = 0; i < PROP_CATEGORY_DEFS.length; i++) {
      if (PROP_CATEGORY_DEFS[i].keys[k]) { buckets[i].keys.push(k); return; }
    }
    contentBucket.keys.push(k); // spec attr not covered by any named bucket (rare — e.g. $RESOURCE* dynamic fields)
  });
  buckets.forEach(function(b) {
    if (!b.order) return;
    // Map dynamic <singular>id / <resourceSingular>id keys to their virtual
    // "id"/"resourceid" order-array slots — see the identityBucket special
    // cases above for why these two are treated identically to the plain
    // "id" key everywhere else. Keeps ordering identical between the
    // Resource page's "Version Details" tab and the dedicated Version page
    // (both show a "resourceid, id, xid, self, …"-style Identity section).
    function orderKey(k) {
      if (singular && k === singular + 'id') return 'id';
      if (resourceSingular && resourceSingular !== singular && k === resourceSingular + 'id') return 'resourceid';
      return k;
    }
    b.keys.sort(function(a, c) {
      var na = orderKey(a), nc = orderKey(c);
      var ia = b.order.indexOf(na), ic = b.order.indexOf(nc);
      if (ia >= 0 && ic >= 0) return ia - ic;
      if (ia >= 0) return -1; if (ic >= 0) return 1;
      return a.localeCompare(c);
    });
  });
  var all = buckets.concat([extBucket]).filter(function(b) { return b.keys.length > 0; });
  if (domainFocused) {
    var totalKeys = all.reduce(function(n, b) { return n + b.keys.length; }, 0);
    var domainAll = all.map(function(b) {
      if (b.label === 'Identity') return null;
      if (b.label === 'Versioning & State') {
        var kept = b.keys.filter(function(k) { return DOMAIN_FOCUSED_KEEP_KEYS[k]; });
        return kept.length ? { label: b.label, keys: kept, order: b.order } : null;
      }
      return b;
    }).filter(Boolean);
    var domainKeys = domainAll.reduce(function(n, b) { return n + b.keys.length; }, 0);
    // Must not fall back to the caller's raw/unfiltered `keys` list (which
    // still contains Identity/Versioning & State attrs) just because
    // hiding those categories collapsed the remaining set to 0 or 1
    // buckets — always return the (possibly single-bucket) filtered array
    // whenever anything was actually hidden, even at the cost of showing
    // one lone category header instead of a flat list.
    if (domainKeys !== totalKeys) return domainAll;
  }
  return all.length > 1 ? all : null;
}

// Builds the category-header <tr> a Property table shows above each
// group returned by groupPropsByCategory() — shared by
// buildEntityPropsTableHtml() and renderMetaTable() so both apply Domain
// view identically. In Domain view (the default; View mode only), there's
// so little left to group that a text heading per category isn't worth
// it — except the Extensions/"<Singular> Metadata" bucket, which still
// gets a visual break via a plain divider line (no text) so it reads as
// a distinct section without looking like more xReg plumbing.
function buildPropsCatRowHtml(group, domainFocused) {
  if (domainFocused) {
    if (group.ext) return '<tr class="xr-props-cat-divider"><td colspan="2"><hr class="xr-props-divider"></td></tr>';
    return '';
  }
  return '<tr class="xr-props-cat"><td colspan="2">' + esc(group.label) + '</td></tr>';
}

// Global attribute-priority list used to order a Property table's keys
// before any category grouping is applied — shared by
// buildEntityPropsTableHtml() (View mode) and orderPropKeysFlat() below
// (the Add-entity form), so both always agree on ordering.
var PROPS_PRIORITY = ['registryid', 'xid', 'name', 'description', 'specversion',
  'epoch', 'createdat', 'modifiedat', 'versionid', 'isdefault', 'ancestorid'];

// Returns `keys` re-ordered to match exactly what View mode's Property
// table would show (buildEntityPropsTableHtml()'s priority sort, then
// flattened through the same category buckets groupPropsByCategory()
// uses) — but as a plain flat array, with no category header rows. Used
// by callers (e.g. the Add-entity form) that want the same left-to-right
// attribute ordering as View mode without needing the header rows
// themselves.
function orderPropKeysFlat(keys, specLevel, singular, resourceSingular) {
  var sorted = keys.slice().sort(function(a, b) {
    var ai = PROPS_PRIORITY.indexOf(a), bi = PROPS_PRIORITY.indexOf(b);
    if (ai >= 0 && bi >= 0) return ai - bi;
    if (ai >= 0) return -1; if (bi >= 0) return 1;
    return a.localeCompare(b);
  });
  var groups = groupPropsByCategory(sorted, specLevel, singular, resourceSingular, true);
  if (!groups) return sorted;
  var flat = [];
  groups.forEach(function(g) { flat = flat.concat(g.keys); });
  return flat;
}

// ---- Entity Data Editor (List view) ----------------------------------------
//
// Adds an in-place "edit mode" to the single-entity Properties table built
// by buildEntityPropsTableHtml()/buildPropsRowsHtml() (Registry root, Group,
// Resource, and Version pages — see renderSingleEntity()). Unlike the
// ModelSource/Capabilities editors (which replace #main-view with a fully
// separate custom UI), this piggybacks directly on the existing read-only
// Property table renderer: when `editable` is passed as true AND
// _state.editMode is on, each row's value cell swaps its normal read-only
// display for an input matching the attribute's model type. See plan.md
// "Full Data Edit Mode" for the full design/scope notes (v1 is scalar
// attributes + adding extensions only — nested array/object/map editing and
// live ifvalues-reactivity are tracked follow-ups).
//
// _dataEditSrc / _dataEditData mirror the _capSrc/_capData pattern: a
// pristine snapshot (for Undo/PATCH-diffing) and the working copy being
// edited. Keyed by server+path so navigating to a different entity always
// starts from a fresh snapshot, but toggling edit mode on/off (or an
// in-place refresh while already editing) doesn't clobber in-progress edits.
var _dataEditSrc    = null;
var _dataEditData   = null;
var _dataDirty      = false;
var _dataLoadedFor  = null;

// Meta tab (Resource page "<Type> Metadata" tab) editing — same
// pristine-snapshot/working-copy pattern as _dataEditSrc/_dataEditData
// above, but kept as its own separate pair rather than reused, since the
// Meta entity lives at a different URL (metaurl) than the Resource/
// Version entity the "Version Details"/"Version" tab edits, and a user
// can have unsaved changes on *either* tab (not both at once — see
// _dataEditActiveKind below and switchDocTab()'s unsaved-changes guard).
var _metaEditSrc    = null;
var _metaEditData   = null;
var _metaDirty      = false;

// Resource page's "Version:" dropdown editing (a version other than
// "Default") — same pristine-snapshot/working-copy pattern as the Meta
// tab above, but scoped to whichever non-default version id is currently
// selected (_verEditVid). Picking "Default" reuses _dataEditData/_dataDirty
// directly instead (the resource's own data already IS the default
// version's data), so this pair is only ever populated for a genuinely
// non-default selection. _verSelfUrl/_verPanelId/_verHeaderLabel are
// stashed alongside so saveVersionEntity()/rerenderVersionPanel() (run
// from later, independent DOM events) know where to PUT/PATCH and which
// panel/header to redraw. Reset whenever the Resource page itself
// re-renders from scratch (see _resSelectedVersionId reset).
var _verEditSrc     = null;
var _verEditData    = null;
var _verDirty       = false;
var _verEditVid     = null;
var _verSelfUrl     = '';
var _verPanelId     = null;
var _verHeaderLabel = null;

// User-chosen "items are always this one type" hints for "any"/undeclared
// map/array containers (see the "Items are:" selector in
// renderEditableComplexValue()'s map/array branches, and
// dataEditSetItemTypeHint() below) — JSON has no way to record this on the
// data itself (an empty {}/[] looks the same either way), so it's tracked
// here instead, keyed by the container's own keyPath (JSON.stringify'd).
// Shared across both the entity/Meta tab working copies (data and meta
// keyPaths never collide in practice, since they always start with a
// different top-level attribute name) and reset whenever either working
// copy is reloaded from scratch, same as _dataEditSrc/_metaEditSrc.
//
// NOTE: this is a *different* concept from _shapeHints below, even though
// both are path-keyed "can't tell from the JSON alone" hints — this one
// says "what type should THIS container's future children default to",
// while _shapeHints says "what type IS the value already sitting at this
// exact path". Keeping them in separate stores (rather than reusing one
// for both) matters because a map/object's own path is used as the key
// for both concepts at once (its own shape, and its children's item-type
// default) — sharing a single store caused the map's own map-vs-object
// shape hint to be misread back as its *own children's* item-type hint,
// e.g. a key explicitly added as "map" showing a "Map, Map" pill instead
// of "Map, Any" (or whatever its actual item type is/becomes).
var _itemTypeHints  = {};

// User-chosen "this specific value is definitely type X" hints — recorded
// only for the map-vs-object ambiguity (see dataEditMapAddEntry()'s
// comment for why: both start out as an indistinguishable empty {}, and
// nothing else about the value can tell them apart later). Keyed by the
// exact keyPath of the value itself (not a container's children, unlike
// _itemTypeHints above). Reset alongside _itemTypeHints.
var _shapeHints     = {};

// Which working-copy pair (_dataEditData/_dataEditSrc/_dataDirty for the
// entity itself, or _metaEditData/_metaEditSrc/_metaDirty for the Resource
// page's Meta tab) the shared nested-attribute mutation primitives below
// (dataEditNestedFieldChange(), dataEditMapAddEntry(), etc.) currently
// target. Only one of the two tabs is ever visibly "being edited" at a
// time, so a single shared indicator — flipped by switchDocTab() whenever
// the active tab changes, and by renderMetaTable()/renderSingleEntity()
// when (re)rendering their own editable rows — is enough; see
// activeEditRoot()/markActiveEditDirty()/rerenderActiveEdit() just below.
var _dataEditActiveKind = 'entity'; // 'entity' | 'meta' | 'version'

function activeEditRoot() {
  if (_dataEditActiveKind === 'meta') return _metaEditData;
  if (_dataEditActiveKind === 'version') return _verEditData;
  return _dataEditData;
}
// Pristine (pre-edit) snapshot matching activeEditRoot() above — used to
// compute whether a nested edit actually differs from the saved value
// (see dataEditNestedFieldChange()'s dirty-row highlight).
function activeEditSrc() {
  if (_dataEditActiveKind === 'meta') return _metaEditSrc;
  if (_dataEditActiveKind === 'version') return _verEditSrc;
  return _dataEditSrc;
}
function markActiveEditDirty() {
  if (_dataEditActiveKind === 'meta') markMetaDirty();
  else if (_dataEditActiveKind === 'version') markVerDirty();
  else markDataDirty();
}
function rerenderActiveEdit() {
  if (_dataEditActiveKind === 'meta') rerenderMetaTab();
  else if (_dataEditActiveKind === 'version') rerenderVersionPanel();
  else renderSingleEntity(_dataEditData);
}

// Core spec attributes that must always stay read-only in entity edit mode
// regardless of model — server-generated/managed values PUT/PATCH can never
// set (see common/shared_entity's OrderedSpecProps ReadOnly/Immutable
// flags). specattrs.js (auto-generated) doesn't currently carry per-attr
// readonly/immutable flags, so this is a small manually-maintained list
// instead — see plan.md's "full-data-edit-mode" follow-up to fold this into
// codegen later.
//
// NOTE: `createdat`/`modifiedat` are deliberately NOT in this list — per
// spec.md's "createdat"/"modifiedat" Attribute sections, both are
// client-settable in a write request ("If present in a write operation
// request, the value MUST override any existing value..."); the server
// only supplies a default (current time) when the request omits them.
// They're genuinely editable, not read-only, matching what /model actually
// declares (no readonly/immutable flag on either) — see isDataEditReadOnly()
// below, which now checks the model instead of assuming.
var DATA_EDIT_READONLY = {
  specversion: 1, id: 1, versionid: 1, self: 1, shortself: 1, xid: 1,
  epoch: 1, isdefault: 1, readonly: 1,
  formatvalidated: 1, formatvalidatedreason: 1,
  compatibilityvalidated: 1, compatibilityvalidatedreason: 1,
  ancestorid: 1, meta: 1, metaurl: 1
};
// `attr` (the model's own Attribute definition, when available) is checked
// too — any attribute (spec or extension) the model itself marks readonly
// or immutable must stay read-only here regardless of the static list
// above, which only covers spec-defined attributes not otherwise carried
// with per-attr flags (see specattrs.js note above). `singular`/
// `resourceSingular` catch the dynamic <singular>id / <resourceSingular>id
// identity fields (e.g. "endpointid", "registryid") — these are just as
// immutable as a literal "id" key, but aren't literally named "id" so the
// static list can't match them by name alone.
function isDataEditReadOnly(k, attr, singular, resourceSingular) {
  if (DATA_EDIT_READONLY[k]) return true;
  if (attr && (attr.readonly || attr.immutable)) return true;
  if (singular && k === singular + 'id') return true;
  if (resourceSingular && resourceSingular !== singular && k === resourceSingular + 'id') return true;
  return false;
}

// Shared enum-input renderer for both renderEditableScalarInput() (top-
// level scalar attrs) and renderEditableScalarInputAtPath() (nested
// map/object/array leaf values). Strict enums (the model default — see
// shared_model.go Attribute.GetStrict(), which returns true unless the
// model explicitly sets strict:false) render as a plain <select>: the
// value MUST be one of the declared enum entries. A non-strict enum
// (attr.strict === false) offers the enum values as *suggestions* via a
// small independent dropdown (see toggleEnumDropdown()/pickEnumValue()
// below) — a native <input list="..."> + <datalist> combo was tried
// first, but browsers unconditionally filter a datalist's options
// against the input's *current* text (even when opened via
// showPicker()), so a field that already has a value only ever shows the
// one entry matching what's typed — indistinguishable from "no other
// choices exist". There's no way to disable that browser-side filtering,
// so instead this renders its own always-complete, unfiltered list: it
// only ever reads the field's value (to highlight the current match) and
// only ever writes to it when an item is actually clicked — typing in the
// field itself is completely free-form and never touched by the dropdown.
//
// The list itself is rendered into a single shared, body-level "portal"
// element (see ensureEnumDropdownPortal()) instead of inline next to the
// input — an inline dropdown positioned via the input's own ancestor
// (position:relative + position:absolute) can get clipped by any
// scrollable/overflow-hidden ancestor (e.g. the Properties table's own
// container) once the input is near the bottom of the visible area, with
// no way to scroll to see the rest of it. The portal is appended directly
// to <body> and positioned with `position: fixed` from the caret button's
// own on-screen coordinates (see positionEnumDropdownPortal()), so it's
// never clipped and flips to open upward automatically when there isn't
// enough room below.
var _enumInputSeq = 0;
function renderEnumInputHtml(enumVals, val, type, isStrict, onchg) {
  if (isStrict === false) {
    var seq = ++_enumInputSeq;
    var inputId = 'enuminput_' + seq;
    var curStr = val == null ? '' : String(val);
    var itemsJson = JSON.stringify(enumVals.map(function(ev) { return String(ev); }));
    return '<span class="xr-enum-combo">'
      + '<input type="text" id="' + esc(inputId) + '" class="xr-edit-input" autocomplete="off" data-ftype="' + esc(type) + '"'
      + ' value="' + esc(curStr) + '"' + onchg + '>'
      + '<button type="button" class="editorBtn editorBtnSmall xr-enum-caret-btn" '
      + 'title="Show suggested values" data-input-id="' + esc(inputId) + '" data-items="' + esc(itemsJson) + '" '
      + 'onclick="toggleEnumDropdown(this)">\u25BC</button>'
      + '</span>';
  }
  var opts = '<option value=""></option>' + enumVals.map(function(ev) {
    var sel = (val != null && String(ev) === String(val)) ? ' selected' : '';
    return '<option value="' + esc(String(ev)) + '"' + sel + '>' + esc(String(ev)) + '</option>';
  }).join('');
  return '<select class="xr-edit-input" data-ftype="' + esc(type) + '"' + onchg + '>' + opts + '</select>';
}

// Lazily creates (once) the single shared portal element every non-strict
// enum dropdown reuses — appended directly to <body> so `position: fixed`
// coordinates (see positionEnumDropdownPortal()) are never affected by
// any ancestor's scroll position or overflow clipping.
function ensureEnumDropdownPortal() {
  var portal = document.getElementById('xr-enum-dropdown-portal');
  if (!portal) {
    portal = document.createElement('div');
    portal.id = 'xr-enum-dropdown-portal';
    portal.className = 'xr-enum-dropdown xr-enum-dropdown-portal';
    portal.style.display = 'none';
    document.body.appendChild(portal);
  }
  return portal;
}

var _enumDropdownOutsideHandler = null;
var _enumDropdownOpenBtn = null;

// Closes the currently-open enum suggestion dropdown, if any.
function closeAllEnumDropdowns() {
  var portal = document.getElementById('xr-enum-dropdown-portal');
  if (portal) { portal.style.display = 'none'; portal.innerHTML = ''; }
  _enumDropdownOpenBtn = null;
  if (_enumDropdownOutsideHandler) {
    document.removeEventListener('mousedown', _enumDropdownOutsideHandler);
    window.removeEventListener('scroll', closeAllEnumDropdowns, true);
    window.removeEventListener('resize', closeAllEnumDropdowns);
    _enumDropdownOutsideHandler = null;
  }
}

// Positions the shared portal (already populated + display:block, but
// still effectively invisible via visibility:hidden) against `anchorEl`
// (the caret button), flipping above the anchor instead of below it when
// there isn't enough viewport room underneath, and clamping both axes so
// it never renders off-screen. Uses `position: fixed`, so all coordinates
// are viewport-relative (getBoundingClientRect() already returns
// viewport-relative rects too — no scroll-offset math needed).
function positionEnumDropdownPortal(portal, anchorEl) {
  var rect = anchorEl.getBoundingClientRect();
  var vw = document.documentElement.clientWidth;
  var vh = document.documentElement.clientHeight;
  var ddRect = portal.getBoundingClientRect(); // measured while display:block + visibility:hidden
  var spaceBelow = vh - rect.bottom;
  var spaceAbove = rect.top;
  var top = (spaceBelow >= ddRect.height || spaceBelow >= spaceAbove)
    ? rect.bottom + 2
    : rect.top - ddRect.height - 2;
  top = Math.max(4, Math.min(top, vh - ddRect.height - 4));
  var left = Math.max(4, Math.min(rect.left, vw - ddRect.width - 4));
  portal.style.top = top + 'px';
  portal.style.left = left + 'px';
}

// Opens/closes a non-strict enum's suggestion dropdown (see
// renderEnumInputHtml()) — always shows every enum value, completely
// independent of whatever text is currently in the field, since the whole
// point is to work around the browser's own datalist filtering (which
// can't be turned off). Clicking outside the dropdown, scrolling, or
// resizing the window closes it without changing anything.
function toggleEnumDropdown(btn) {
  if (_enumDropdownOpenBtn === btn) { closeAllEnumDropdowns(); return; }
  closeAllEnumDropdowns();
  var inputId = btn.getAttribute('data-input-id');
  var items;
  try { items = JSON.parse(btn.getAttribute('data-items') || '[]'); } catch (e) { items = []; }
  var inp = document.getElementById(inputId);
  var curVal = inp ? inp.value : '';

  var portal = ensureEnumDropdownPortal();
  portal.innerHTML = '';
  items.forEach(function(ev) {
    var item = document.createElement('div');
    item.className = 'xr-enum-dropdown-item' + (ev === curVal ? ' xr-enum-dropdown-item-selected' : '');
    item.textContent = ev;
    item.addEventListener('click', function() { pickEnumValue(inputId, ev); });
    portal.appendChild(item);
  });

  portal.style.visibility = 'hidden';
  portal.style.display = 'block';
  positionEnumDropdownPortal(portal, btn);
  portal.style.visibility = 'visible';
  _enumDropdownOpenBtn = btn;

  _enumDropdownOutsideHandler = function(e) {
    if (!portal.contains(e.target) && e.target !== btn) closeAllEnumDropdowns();
  };
  // Deferred so the very click that opened the dropdown doesn't also
  // immediately trigger the outside-click handler and close it again.
  setTimeout(function() {
    document.addEventListener('mousedown', _enumDropdownOutsideHandler);
    window.addEventListener('scroll', closeAllEnumDropdowns, true);
    window.addEventListener('resize', closeAllEnumDropdowns);
  }, 0);
}

// Selects an enum value from the dropdown into its entry field — the only
// path that ever writes into a non-strict enum input other than the user
// directly typing into it. Fires a real change event so the existing
// onchange handler (dataEditFieldChange()/dataEditNestedFieldChange()/
// metaEditFieldChange()/verEditFieldChange(), whichever this input was
// wired to) picks it up exactly like a normal edit.
function pickEnumValue(inputId, val) {
  var inp = document.getElementById(inputId);
  if (!inp) return;
  inp.value = val;
  inp.dispatchEvent(new Event('change', {bubbles: true}));
  closeAllEnumDropdowns();
}

// Renders an editable input for one scalar attribute value, matching its
// model type (enum takes priority over type — a "select" is always clearer
// than a typed input when the value is constrained to a fixed list).
// `handlerFn` (default 'dataEditFieldChange') lets other callers — e.g. the
// collection Add-entity form's addNewFieldChange() — reuse this same input
// rendering against a different backing object.
function renderEditableScalarInput(k, val, attr, handlerFn) {
  var type = (attr && attr.type)
    || (typeof val === 'boolean' ? 'boolean' : typeof val === 'number' ? 'decimal' : 'string');
  var enumVals = attr && attr.enum;
  var handler = handlerFn || 'dataEditFieldChange';
  var onchg = ' onchange="' + esc(handler) + '(' + esc(JSON.stringify(k)) + ', this)"';
  if (enumVals && enumVals.length) {
    return renderEnumInputHtml(enumVals, val, type, attr && attr.strict, onchg);
  }
  if (type === 'boolean') {
    return '<label class="xr-bool-edit"><input type="checkbox" class="xr-edit-input" data-ftype="boolean"'
      + (val ? ' checked' : '') + onchg + '> true</label>';
  }
  if (type === 'timestamp') {
    return renderTimestampInputHtml(handler + '(' + JSON.stringify(k) + ', this)', val);
  }
  if (type === 'integer' || type === 'uinteger' || type === 'decimal') {
    return '<input type="number" class="xr-edit-input" data-ftype="' + esc(type) + '"'
      + (type === 'decimal' ? ' step="any"' : ' step="1"')
      + ' value="' + esc(val == null ? '' : String(val)) + '"' + onchg + '>';
  }
  return '<input type="text" class="xr-edit-input" data-ftype="string"'
    + ' value="' + esc(val == null ? '' : String(val)) + '"' + onchg + '>';
}

// Renders an editable text input for a timestamp-typed attribute (RFC3339
// string, e.g. createdat/modifiedat or any model-declared timestamp
// extension), plus a convenience "Now" button that fills in the current
// time — mirroring spec.md's own "a value of null MUST use the current
// date/time" behavior for createdat, and modifiedat's similar "absent ⇒
// current time" default, as an explicit one-click option rather than
// requiring the value be hand-typed. `onchgJs` is the raw (unescaped) JS
// call to invoke on change (e.g. `dataEditFieldChange("createdat", this)`),
// matching the calling convention renderEditableScalarInput()/
// renderEditableScalarInputAtPath() already use for their other types.
var _tsInputSeq = 0;
function renderTimestampInputHtml(onchgJs, val) {
  var id = 'tsinput_' + (++_tsInputSeq);
  var nowJs = 'var e=document.getElementById(' + JSON.stringify(id) + ');'
    + "e.value=new Date().toISOString();"
    + "e.dispatchEvent(new Event('change',{bubbles:true}));";
  return '<span class="xr-ts-edit">'
    + '<input type="text" id="' + esc(id) + '" class="xr-edit-input eg-mono" data-ftype="timestamp" '
    + 'placeholder="RFC3339, e.g. 2030-12-19T06:00:00Z" value="' + esc(val == null ? '' : String(val)) + '" '
    + 'onchange="' + esc(onchgJs) + '">'
    + ' <button type="button" class="editorBtn editorBtnSmall" onclick="' + esc(nowJs) + '">Now</button>'
    + '</span>';
}

// ---- Complex (map/object/array) attribute editing -------------------------
//
// Like renderEditableScalarInput() but for one leaf value living at an
// arbitrary nested location (keyPath, e.g. ['labels','env'] or
// ['meta','tags', 2]) inside a complex attribute's editor — see
// renderEditableComplexValue() below. Wired to dataEditNestedFieldChange()
// instead of dataEditFieldChange() since a single top-level key string
// isn't enough to address a nested location.
function renderEditableScalarInputAtPath(keyPath, val, attr) {
  var type = (attr && attr.type)
    || (typeof val === 'boolean' ? 'boolean' : typeof val === 'number' ? 'decimal' : 'string');
  var enumVals = attr && attr.enum;
  var onchg = ' onchange="dataEditNestedFieldChange(' + esc(JSON.stringify(keyPath)) + ', this)"';
  if (enumVals && enumVals.length) {
    return renderEnumInputHtml(enumVals, val, type, attr && attr.strict, onchg);
  }
  if (type === 'boolean') {
    return '<label class="xr-bool-edit"><input type="checkbox" class="xr-edit-input" data-ftype="boolean"'
      + (val ? ' checked' : '') + onchg + '> true</label>';
  }
  if (type === 'timestamp') {
    return renderTimestampInputHtml('dataEditNestedFieldChange(' + JSON.stringify(keyPath) + ', this)', val);
  }
  if (type === 'integer' || type === 'uinteger' || type === 'decimal') {
    return '<input type="number" class="xr-edit-input" data-ftype="' + esc(type) + '"'
      + (type === 'decimal' ? ' step="any"' : ' step="1"')
      + ' value="' + esc(val == null ? '' : String(val)) + '"' + onchg + '>';
  }
  return '<input type="text" class="xr-edit-input" data-ftype="string"'
    + ' value="' + esc(val == null ? '' : String(val)) + '"' + onchg + '>';
}

// Renders an editable UI for a map/object/array-typed attribute value, at
// the given keyPath into _dataEditData (e.g. ['labels'] for a top-level map
// attribute, or ['meta','tags'] for a nested one). Mutations are written
// directly into the value referenced by keyPath (via getInPath/setInPath),
// so arbitrarily nested complex attributes all share the same small set of
// mutation primitives instead of one bespoke handler per top-level
// attribute. v1 scope: map (any item type, incl. nested complex items),
// fixed-schema object (recursing into its declared .attributes), and array
// (any item type). A complex value with no usable model shape info (no
// declared .item/.attributes) falls back to the existing read-only tree
// render, same graceful-degradation approach used elsewhere for unmodeled
// shapes. See plan.md "Full Data Edit Mode" for design notes.
function renderEditableComplexValue(keyPath, val, attr, model, entityPath, rootData) {
  var type = attr && attr.type;
  var kpJson = esc(JSON.stringify(keyPath));
  if (type === 'map') {
    var kpKey = JSON.stringify(keyPath);
    var declaredItemAttr = (attr && attr.item) || {type: 'string'};
    var itemTypeHintable = !declaredItemAttr.type || declaredItemAttr.type === 'any';
    var itemAttr = effectiveItemAttr(keyPath, attr);
    var mapVal = (val && typeof val === 'object' && !Array.isArray(val)) ? val : {};
    var entries = Object.keys(mapVal).sort();
    var rows = entries.map(function(mk) {
      var childKeyPath = keyPath.concat([mk]);
      // effectiveEditAttr() upgrades an "any"-typed item (e.g. a map whose
      // values can be anything, per-entry) to a synthesized map/array attr
      // when THIS entry's actual runtime value is complex — a map<any>
      // can hold a mix of scalar and complex values across its different
      // keys, so this has to be re-checked per entry, not once for the
      // whole map.
      var effItemAttr = effectiveEditAttrAtPath(childKeyPath, itemAttr, mapVal[mk]);
      var itemIsComplexEntry = effItemAttr.type === 'object' || effItemAttr.type === 'map' || effItemAttr.type === 'array';
      var inputHtml = itemIsComplexEntry
        ? renderEditableComplexValue(childKeyPath, mapVal[mk], effItemAttr, model, entityPath, rootData)
        : typePillHtml(effItemAttr) + renderEditableScalarInputAtPath(childKeyPath, mapVal[mk], effItemAttr);
      // Key is itself editable (rename) — an <input>, not static text, so
      // the map's key can be corrected/changed, not just its value. Commits
      // (and does a full re-render, like Add/Remove) only on change/blur,
      // never per-keystroke.
      var keyInputHtml = '<input type="text" class="xr-edit-input xr-map-key-input" value="' + esc(mk) + '" '
        + 'onchange="dataEditMapRenameKey(' + kpJson + ',' + esc(JSON.stringify(mk)) + ', this)">';
      var removeBtnHtml = '<button class="editorBtn editorBtnSmall editorBtnDanger xr-rm-btn-small" '
        + 'onclick="dataEditMapRemoveEntry(' + kpJson + ',' + esc(JSON.stringify(mk)) + ')" title="Remove">\u2715</button>';
      // A complex (map/object/array) value renders on its own line below
      // the key, indented — squeezing a whole nested editor to the right
      // of its key (like a plain scalar value) makes it hard to see what
      // belongs to what. Scalars keep the original compact inline layout.
      // The remove button sits to the *left* of the key (small "X",
      // matching the array item treatment below) in both cases.
      if (itemIsComplexEntry) {
        return '<div class="xr-map-row xr-map-row-complex">'
          + '<div class="xr-map-row-head">' + removeBtnHtml + '<span class="xr-map-key">' + keyInputHtml + '</span></div>'
          + '<div class="xr-map-val xr-map-val-indent">' + inputHtml + '</div>'
          + '</div>';
      }
      return '<div class="xr-map-row">'
        + '<div class="xr-map-row-head">' + removeBtnHtml + '<span class="xr-map-key">' + keyInputHtml + '</span></div>'
        + '<span class="xr-map-val">' + inputHtml + '</span>'
        + '</div>';
    });
    var addId = 'mapadd_' + keyPath.join('_').replace(/[^a-zA-Z0-9]/g, '') + '_' + Math.random().toString(36).slice(2, 7);
    // No key or value is collected up front — like the array's "+ Add
    // Item" button, this just creates a new entry (an auto-generated,
    // immediately-renamable key, and an empty default value for the
    // chosen type) which is then edited normally afterward. See
    // renderComplexAddButtonHtml() for why/how the type is chosen when
    // the item type is "any"/undeclared.
    rows.push('<div class="xr-map-row xr-map-addrow">'
      + renderComplexAddButtonHtml(itemAttr, '+ Add Entry', addId, 'dataEditMapAddEntry', kpJson)
      + (itemTypeHintable ? itemTypeHintSelectorHtml(kpJson, _itemTypeHints[kpKey] && _itemTypeHints[kpKey].type) : '')
      + '</div>');
    return typePillHtml(Object.assign({}, attr, {item: itemAttr})) + '<div class="xr-map-editor">' + rows.join('') + '</div>';
  }
  if (type === 'array') {
    var kpKey2 = JSON.stringify(keyPath);
    var declaredItemAttr2 = (attr && attr.item) || {type: 'string'};
    var itemTypeHintable2 = !declaredItemAttr2.type || declaredItemAttr2.type === 'any';
    var itemAttr2 = effectiveItemAttr(keyPath, attr);
    var items = Array.isArray(val) ? val : [];
    var rows2 = items.map(function(iv, idx) {
      var childKeyPath = keyPath.concat([idx]);
      // Per-index effectiveEditAttr() upgrade — see the map branch's
      // itemIsComplexEntry comment above; an array<any> can likewise mix
      // scalar and complex items across its indices.
      var effItemAttr2 = effectiveEditAttrAtPath(childKeyPath, itemAttr2, iv);
      var itemIsComplexEntry2 = effItemAttr2.type === 'object' || effItemAttr2.type === 'map' || effItemAttr2.type === 'array';
      var inputHtml = itemIsComplexEntry2
        ? renderEditableComplexValue(childKeyPath, iv, effItemAttr2, model, entityPath, rootData)
        : typePillHtml(effItemAttr2) + renderEditableScalarInputAtPath(childKeyPath, iv, effItemAttr2);
      var removeBtnHtml2 = '<button class="editorBtn editorBtnSmall editorBtnDanger xr-rm-btn-small" '
        + 'onclick="dataEditArrayRemoveItem(' + kpJson + ',' + idx + ')" title="Remove">\u2715</button>';
      // Complex items get the same stacked/indented treatment as a map's
      // complex values (see the map branch above) — index + remove button
      // side by side on their own header line, nested editor indented
      // below. Scalar items get the same X-then-[idx]-then-value inline
      // layout, for consistency — the remove button sits to the *left*
      // of the index it belongs to (matching map/object entries' X-then-
      // key layout), regardless of item shape.
      if (itemIsComplexEntry2) {
        return '<div class="xr-arr-row xr-arr-row-complex">'
          + '<div class="xr-arr-row-head">' + removeBtnHtml2 + '<span class="vt-arr-idx">[' + idx + ']</span></div>'
          + '<div class="xr-arr-val xr-map-val-indent">' + inputHtml + '</div>'
          + '</div>';
      }
      return '<div class="xr-arr-row">' + removeBtnHtml2
        + '<span class="vt-arr-idx">[' + idx + ']</span>'
        + '<span class="xr-arr-val">' + inputHtml + '</span>'
        + '</div>';
    });
    var arrAddId = 'arradd_' + keyPath.join('_').replace(/[^a-zA-Z0-9]/g, '') + '_' + Math.random().toString(36).slice(2, 7);
    rows2.push('<div class="xr-arr-row xr-arr-addrow">'
      + renderComplexAddButtonHtml(itemAttr2, '+ Add Item', arrAddId, 'dataEditArrayAddItem', kpJson)
      + (itemTypeHintable2 ? itemTypeHintSelectorHtml(kpJson, _itemTypeHints[kpKey2] && _itemTypeHints[kpKey2].type) : '')
      + '</div>');
    return typePillHtml(Object.assign({}, attr, {item: itemAttr2})) + '<div class="xr-arr-editor">' + rows2.join('') + '</div>';
  }
  if (type === 'object') {
    var subAttrs = (attr && attr.attributes) || {};
    var wildcardAttr = subAttrs['*'];
    var declaredSubKeys = Object.keys(subAttrs).filter(function(sk) { return sk !== '*'; });
    var objVal = (val && typeof val === 'object' && !Array.isArray(val)) ? val : {};
    // Union of model-declared sub-attributes and any extension keys already
    // present on this object (set via the "+ Add" row below in an earlier
    // edit) — without this, an object whose model only declares "*" (no
    // named sub-attributes at all, e.g. a bare extensible object) would
    // have nothing to iterate and fall through to the read-only tree,
    // even though it should be editable via its wildcard.
    var allSubKeys = declaredSubKeys.slice();
    Object.keys(objVal).forEach(function(pk) { if (allSubKeys.indexOf(pk) === -1) allSubKeys.push(pk); });
    if (!allSubKeys.length && !wildcardAttr) return renderValueTree(val, 0, model, entityPath, keyPath, rootData);
    var rows3 = allSubKeys.map(function(sk) {
      var sAttr = subAttrs[sk] || wildcardAttr || {type: 'string'};
      var childKeyPath = keyPath.concat([sk]);
      var effSAttr = effectiveEditAttrAtPath(childKeyPath, sAttr, objVal[sk]);
      var itemIsComplexEntry3 = effSAttr.type === 'object' || effSAttr.type === 'map' || effSAttr.type === 'array';
      var inputHtml = itemIsComplexEntry3
        ? renderEditableComplexValue(childKeyPath, objVal[sk], effSAttr, model, entityPath, rootData)
        : typePillHtml(effSAttr) + renderEditableScalarInputAtPath(childKeyPath, objVal[sk], effSAttr);
      // Only extension keys (not declared by the model) get a Remove button
      // — a model-declared sub-attribute can be cleared/edited but not
      // removed from the object's shape entirely. Extension keys are also
      // renamable (an <input>, like a map key), since they're just
      // user-chosen names, not fixed by the model.
      var isExtKey = declaredSubKeys.indexOf(sk) === -1;
      var keyHtml = isExtKey
        ? '<input type="text" class="xr-edit-input xr-map-key-input" value="' + esc(sk) + '" '
          + 'onchange="dataEditObjRenameKey(' + kpJson + ',' + esc(JSON.stringify(sk)) + ', this)">'
        : esc(labelFor(sk));
      var removeBtn = isExtKey
        ? '<button class="editorBtn editorBtnSmall editorBtnDanger xr-rm-btn-small" '
          + 'onclick="dataEditObjRemoveKey(' + kpJson + ',' + esc(JSON.stringify(sk)) + ')" title="Remove">\u2715</button>'
        : '';
      // Complex sub-values get the same head(X+key)/border + indented
      // value layout as map/array complex entries above, for the same
      // reason — squeezing a nested editor inline with the key makes it
      // hard to tell what belongs to what. The remove button sits to the
      // *left* of the key (same as a map entry's key) in both cases.
      // Scalar sub-values get the *same* head/border wrapper (unlike map/
      // array, which only border-wrap their complex entries) — an
      // object's keys benefit from a consistent divider between "key" and
      // "value" regardless of the value's shape, since object rows mix
      // named/model-declared keys with free-form extension keys.
      if (itemIsComplexEntry3) {
        return '<div class="xr-obj-row xr-obj-row-complex">'
          + '<div class="xr-obj-row-head">' + removeBtn + '<span class="xr-obj-key">' + keyHtml + ':</span></div>'
          + '<div class="xr-obj-val xr-map-val-indent">' + inputHtml + '</div>'
          + '</div>';
      }
      return '<div class="xr-obj-row">'
        + '<div class="xr-obj-row-head">' + removeBtn + '<span class="xr-obj-key">' + keyHtml + ':</span></div>'
        + '<span class="xr-obj-val">' + inputHtml + '</span></div>';
    });
    var html3 = '<div class="xr-obj-editor">' + rows3.join('');
    if (wildcardAttr) {
      var addId3 = 'objadd_' + keyPath.join('_').replace(/[^a-zA-Z0-9]/g, '') + '_' + Math.random().toString(36).slice(2, 7);
      html3 += '<div class="xr-obj-row xr-obj-addrow">'
        + renderComplexAddButtonHtml(wildcardAttr, '+ Add Key', addId3, 'dataEditObjAddKey', kpJson)
        + '</div>';
    }
    html3 += '</div>';
    return typePillHtml(attr) + html3;
  }
  // Unknown/unsupported complex shape (no declared item/attributes info) —
  // fall back to the existing read-only tree render.
  return renderValueTree(val, 0, model, entityPath, keyPath, rootData);
}

// Reads the value at an arbitrary nested location (array of string/number
// segments) inside obj — the read counterpart of setInPath() below.
function getInPath(obj, keyPath) {
  var cur = obj;
  for (var i = 0; i < keyPath.length && cur != null; i++) cur = cur[keyPath[i]];
  return cur;
}

// Writes val at an arbitrary nested location inside obj, creating any
// missing intermediate containers along the way (object vs array chosen by
// peeking at the *next* path segment's type — numeric segments imply an
// array). Used by all the complex-attribute mutation handlers below so
// map/object/array editing shares one small set of primitives regardless
// of nesting depth.
function setInPath(obj, keyPath, val) {
  var cur = obj;
  for (var i = 0; i < keyPath.length - 1; i++) {
    var k = keyPath[i];
    if (cur[k] == null || typeof cur[k] !== 'object') {
      cur[k] = (typeof keyPath[i + 1] === 'number') ? [] : {};
    }
    cur = cur[k];
  }
  cur[keyPath[keyPath.length - 1]] = val;
}

// onchange handler for scalar leaf inputs anywhere inside a complex
// (map/object/array) attribute's editor — mirrors dataEditFieldChange() but
// addresses an arbitrary nested location via keyPath instead of a single
// top-level key. No re-render (like dataEditFieldChange) so typing never
// loses input focus.
function dataEditNestedFieldChange(keyPath, inputEl) {
  var root = activeEditRoot();
  if (!root) return;
  var type = inputEl.dataset.ftype || 'string';
  var val;
  if (type === 'boolean') val = inputEl.checked;
  else if (type === 'integer' || type === 'uinteger') val = inputEl.value === '' ? null : parseInt(inputEl.value, 10);
  else if (type === 'decimal') val = inputEl.value === '' ? null : parseFloat(inputEl.value);
  else val = inputEl.value;
  setInPath(root, keyPath, val);
  markActiveEditDirty();
  // Highlight the containing top-level row as changed too (mirrors
  // dataEditFieldChange()'s toggleRowDirty()) — otherwise a nested edit
  // (e.g. "deprecated.alternative") marks the working copy dirty (Save
  // buttons correctly enable) but gives no visual feedback at all on the
  // row itself, since this handler deliberately never triggers a full
  // re-render. Compares the whole top-level container's value against its
  // pristine snapshot (activeEditSrc()) rather than just the one leaf,
  // since any of its nested fields changing makes the row dirty.
  var topKey = keyPath[0];
  var src = activeEditSrc();
  var newTopVal = getInPath(root, [topKey]);
  var isDirty = !src || JSON.stringify(newTopVal) !== JSON.stringify(src[topKey]);
  toggleRowDirty(inputEl, isDirty);
}

// Adds a new map entry at keyPath — no key or value is collected up front
// (see renderComplexAddButtonHtml()): an auto-generated, unique key gets
// an empty default value for the chosen type, then renders as a normal
// row the user edits in place (rename the key via its own input, fill in
// the value). itemType is the item's declared type when concrete, or null
// when it's "any"/undeclared — in which case it's read from the add
// button's type-select overlay (id addId + '_t') instead. Full re-render
// (like dataEditAddExtensionV2()) since it's only ever triggered by an
// explicit button click, never by typing.
function dataEditMapAddEntry(keyPath, addId, itemType) {
  var root = activeEditRoot();
  if (!root) return;
  var mapObj = getInPath(root, keyPath);
  if (!mapObj || typeof mapObj !== 'object' || Array.isArray(mapObj)) {
    mapObj = {};
    setInPath(root, keyPath, mapObj);
  }
  if (!itemType) {
    var typeEl = document.getElementById(addId + '_t');
    itemType = typeEl ? typeEl.value : 'string';
  }
  var key = genUniqueKey(mapObj, 'key');
  mapObj[key] = defaultValueForType(itemType);
  // Remember the explicitly-chosen type per-entry (not just per-container
  // — see itemTypeHintSelectorHtml() for the container-wide version) —
  // needed not only for the map-vs-object ambiguity (both start out as a
  // plain {}, indistinguishable afterward — see effectiveEditAttr()) but
  // also to keep e.g. "integer" vs "decimal" vs "uinteger", or
  // "timestamp"/"uri"/"url"/"xid" vs plain "string", from collapsing back
  // down to a generic runtime-type guess once a value exists.
  _shapeHints[JSON.stringify(keyPath.concat([key]))] = shapedAttrForType(itemType);
  markActiveEditDirty();
  rerenderActiveEdit();
}

function dataEditMapRemoveEntry(keyPath, mapKey) {
  var root = activeEditRoot();
  if (!root) return;
  var mapObj = getInPath(root, keyPath);
  if (!mapObj || typeof mapObj !== 'object') return;
  delete mapObj[mapKey];
  delete _shapeHints[JSON.stringify(keyPath.concat([mapKey]))];
  markActiveEditDirty();
  rerenderActiveEdit();
}

// Renames a map entry's key in place (preserving its value and the map's
// insertion order) — triggered by the key <input>'s onchange (see
// renderEditableComplexValue()'s map branch). Reverts to the old key
// (via a full re-render, discarding the invalid typed value) if the new
// name is empty, unchanged, or collides with another existing key.
function dataEditMapRenameKey(keyPath, oldKey, inputEl) {
  var root = activeEditRoot();
  if (!root) return;
  var mapObj = getInPath(root, keyPath);
  if (!mapObj || typeof mapObj !== 'object') return;
  var newKey = (inputEl.value || '').trim();
  if (!newKey || newKey === oldKey) { rerenderActiveEdit(); return; }
  if (Object.prototype.hasOwnProperty.call(mapObj, newKey)) {
    alert('Key "' + newKey + '" already exists.');
    rerenderActiveEdit();
    return;
  }
  var rebuilt = {};
  Object.keys(mapObj).forEach(function(k) {
    rebuilt[k === oldKey ? newKey : k] = mapObj[k];
  });
  setInPath(root, keyPath, rebuilt);
  // Carry the entry's map-vs-object hint (if any) over to its new key —
  // otherwise renaming would silently lose an explicit type choice (see
  // dataEditMapAddEntry()'s matching comment).
  var oldHintKey = JSON.stringify(keyPath.concat([oldKey]));
  if (_shapeHints[oldHintKey]) {
    _shapeHints[JSON.stringify(keyPath.concat([newKey]))] = _shapeHints[oldHintKey];
    delete _shapeHints[oldHintKey];
  }
  markActiveEditDirty();
  rerenderActiveEdit();
}

// Adds a new item to an array-typed attribute at keyPath. itemType is the
// item's declared type when concrete, or null when it's "any"/undeclared,
// read instead from the add button's type-select overlay (id addId +
// '_t') — see renderComplexAddButtonHtml(). The new item gets an empty
// default value for the chosen type and is then edited normally in place
// afterward, same as dataEditMapAddEntry() above.
function dataEditArrayAddItem(keyPath, addId, itemType) {
  var root = activeEditRoot();
  if (!root) return;
  var arr = getInPath(root, keyPath);
  if (!Array.isArray(arr)) { arr = []; setInPath(root, keyPath, arr); }
  if (!itemType) {
    var typeEl = document.getElementById(addId + '_t');
    itemType = typeEl ? typeEl.value : 'string';
  }
  var idx = arr.length;
  arr.push(defaultValueForType(itemType));
  // See dataEditMapAddEntry()'s matching comment — remember the explicit
  // type choice per-item.
  _shapeHints[JSON.stringify(keyPath.concat([idx]))] = shapedAttrForType(itemType);
  markActiveEditDirty();
  rerenderActiveEdit();
}

function dataEditArrayRemoveItem(keyPath, idx) {
  var root = activeEditRoot();
  if (!root) return;
  var arr = getInPath(root, keyPath);
  if (!Array.isArray(arr)) return;
  arr.splice(idx, 1);
  delete _shapeHints[JSON.stringify(keyPath.concat([idx]))];
  markActiveEditDirty();
  rerenderActiveEdit();
}

// Adds a new extension key to an object-typed attribute (or nested object)
// whose model declares a wildcard ("*") — see renderEditableComplexValue()'s
// object branch. No key is collected up front — an auto-generated, unique
// key gets an empty default value for the chosen type (declared, or read
// from the add button's type-select overlay when "any"/undeclared — see
// renderComplexAddButtonHtml()), then renders as a normal, immediately-
// renamable row, same as dataEditMapAddEntry()/dataEditArrayAddItem().
function dataEditObjAddKey(keyPath, addId, itemType) {
  var root = activeEditRoot();
  if (!root) return;
  var objVal = getInPath(root, keyPath);
  if (!objVal || typeof objVal !== 'object' || Array.isArray(objVal)) {
    objVal = {};
    setInPath(root, keyPath, objVal);
  }
  if (!itemType) {
    var typeEl = document.getElementById(addId + '_t');
    itemType = typeEl ? typeEl.value : 'string';
  }
  var key = genUniqueKey(objVal, 'key');
  objVal[key] = defaultValueForType(itemType);
  // See dataEditMapAddEntry()'s matching comment — remember the explicit
  // type choice per-key.
  _shapeHints[JSON.stringify(keyPath.concat([key]))] = shapedAttrForType(itemType);
  markActiveEditDirty();
  rerenderActiveEdit();
}

// Sets a never-set, still-fully-ambiguous ("any"/undeclared) top-level (or
// Meta tab) attribute's value for the first time — wired to the "Set
// Value" split button rendered by buildPropsRowsHtml()/renderMetaTable()
// (see isAnyTypeUnset()/renderComplexAddButtonHtml()) instead of the
// plain always-string input those rows would otherwise get. Same
// shape-hint bookkeeping as dataEditMapAddEntry()/dataEditObjAddKey()
// above (keyed by the attribute's own keyPath here, e.g. ['mylabel'])
// so the chosen type — including map/object/array, which then renders
// via renderEditableComplexValue() on the next render — sticks instead
// of collapsing back down to a runtime-type guess.
function dataEditSetAnyType(keyPath, addId, itemType) {
  var root = activeEditRoot();
  if (!root) return;
  if (!itemType) {
    var typeEl = document.getElementById(addId + '_t');
    itemType = typeEl ? typeEl.value : 'string';
  }
  setInPath(root, keyPath, defaultValueForType(itemType));
  _shapeHints[JSON.stringify(keyPath)] = shapedAttrForType(itemType);
  markActiveEditDirty();
  rerenderActiveEdit();
}

// Adds a brand-new top-level extension attribute (name typed up front,
// same as before, since an extension's name is meaningful and can't be
// auto-generated) — but its type/value is now chosen via the same split
// "+ Add"-button-with-overlay-type-picker pattern used everywhere else
// (map/array/object "+Add", the "Set Value" picker above), replacing the
// older pre-collect {type select + value input + separate Add button}
// row (renderExtTypeValueWidgetHtml() — still used, unchanged, by the
// "Add new entity" form's own extension row). Shared by both the entity
// and Meta tabs via activeEditRoot()/markActiveEditDirty()/
// rerenderActiveEdit(), same as every other nested-edit handler above.
function dataEditAddExtensionV2(nameInputId, addId, itemType) {
  var root = activeEditRoot();
  if (!root) return;
  var nameEl = document.getElementById(nameInputId);
  var name = nameEl ? (nameEl.value || '').trim() : '';
  if (!name) { alert('Enter an extension name.'); return; }
  if (Object.prototype.hasOwnProperty.call(root, name)) {
    alert('Attribute "' + name + '" already exists.');
    return;
  }
  if (!itemType) {
    var typeEl = document.getElementById(addId + '_t');
    itemType = typeEl ? typeEl.value : 'string';
  }
  root[name] = defaultValueForType(itemType);
  _shapeHints[JSON.stringify([name])] = shapedAttrForType(itemType);
  markActiveEditDirty();
  rerenderActiveEdit();
}


// Renames an extension key of an object-typed attribute — same rename
// semantics as dataEditMapRenameKey() above, but only ever wired to
// extension keys (isExtKey in renderEditableComplexValue()'s object
// branch), never a model-declared sub-attribute's name.
function dataEditObjRenameKey(keyPath, oldKey, inputEl) {
  var root = activeEditRoot();
  if (!root) return;
  var objVal = getInPath(root, keyPath);
  if (!objVal || typeof objVal !== 'object') return;
  var newKey = (inputEl.value || '').trim();
  if (!newKey || newKey === oldKey) { rerenderActiveEdit(); return; }
  if (Object.prototype.hasOwnProperty.call(objVal, newKey)) {
    alert('Key "' + newKey + '" already exists.');
    rerenderActiveEdit();
    return;
  }
  var rebuilt = {};
  Object.keys(objVal).forEach(function(k) {
    rebuilt[k === oldKey ? newKey : k] = objVal[k];
  });
  setInPath(root, keyPath, rebuilt);
  var oldHintKey2 = JSON.stringify(keyPath.concat([oldKey]));
  if (_shapeHints[oldHintKey2]) {
    _shapeHints[JSON.stringify(keyPath.concat([newKey]))] = _shapeHints[oldHintKey2];
    delete _shapeHints[oldHintKey2];
  }
  markActiveEditDirty();
  rerenderActiveEdit();
}

// Removes an extension key from an object-typed attribute — only ever
// wired to keys not declared by the model (see isExtKey in
// renderEditableComplexValue()'s object branch), so this never removes a
// model-declared sub-attribute's slot.
function dataEditObjRemoveKey(keyPath, key) {
  var root = activeEditRoot();
  if (!root) return;
  var objVal = getInPath(root, keyPath);
  if (!objVal || typeof objVal !== 'object') return;
  delete objVal[key];
  delete _shapeHints[JSON.stringify(keyPath.concat([key]))];
  markActiveEditDirty();
  rerenderActiveEdit();
}

// onchange handler for every editable input rendered above — mutates the
// working copy (_dataEditData) directly and marks the page dirty. No full
// table re-render happens here (unlike dataEditAddExtensionV2()) so typing
// never loses input focus — but the row's dirty (changed) highlight is
// still updated immediately and deterministically via toggleRowDirty(),
// rather than relying on some *other*, unrelated action (e.g. Add Item on
// a different attribute) to eventually trigger a full re-render that
// happens to recompute it. Previously this only happened as a side effect
// of nested map/array/object handlers' rerenderActiveEdit() calls, which
// is why scalar/boolean edits wouldn't show as changed until some other
// edit elsewhere forced a redraw — and even then inconsistently, since it
// depended on exactly when/whether that redraw happened to run.
function dataEditFieldChange(k, inputEl) {
  if (!_dataEditData) return;
  var type = inputEl.dataset.ftype || 'string';
  var val;
  if (type === 'boolean') {
    val = inputEl.checked;
  } else if (type === 'integer' || type === 'uinteger') {
    val = inputEl.value === '' ? null : parseInt(inputEl.value, 10);
  } else if (type === 'decimal') {
    val = inputEl.value === '' ? null : parseFloat(inputEl.value);
  } else {
    val = inputEl.value;
  }
  var model = _modelCache[normalizeURL(_state.serverURL || window.location.origin)] || null;
  var beforeAttrs = topLevelAttrsMapFor(model, _state.path, _dataEditData);
  _dataEditData[k] = val;
  markDataDirty();
  // ifvalues reactivity: if this attribute's new value adds/removes
  // conditional sibling attributes, a full re-render is needed so those
  // rows appear/disappear immediately (see reconcileIfValuesOnChange()).
  if (reconcileIfValuesOnChange(model, _state.path, beforeAttrs, _dataEditData)) {
    renderSingleEntity(_lastData || _dataEditSrc);
    return;
  }
  toggleRowDirty(inputEl, _dataEditSrc && JSON.stringify(val) !== JSON.stringify(_dataEditSrc[k]));
}

// Toggles the "changed" (xr-row-dirty) highlight on the <tr> containing
// the given input, without touching anything else in the table — used by
// dataEditFieldChange()/metaEditFieldChange() so a scalar/boolean edit's
// dirty-row highlight updates immediately and every time, regardless of
// whether anything else on the page happens to trigger a full re-render.
function toggleRowDirty(inputEl, isDirty) {
  var tr = inputEl && inputEl.closest ? inputEl.closest('tr') : null;
  if (tr) tr.classList.toggle('xr-row-dirty', !!isDirty);
}

function markDataDirty() {
  if (!_dataDirty) {
    _dataDirty = true;
    var pb = document.getElementById('dataSavePutBtn');   if (pb) pb.disabled = false;
    var ab = document.getElementById('dataSavePatchBtn'); if (ab) ab.disabled = false;
    var ub = document.getElementById('dataUndoBtn');      if (ub) ub.disabled = false;
  }
}

// Shared <option> list for every "pick a type" dropdown in Full Data Edit
// Mode — the top-level "+ Add Extension" widget below, and the compact
// type-only overlay used by renderComplexAddButtonHtml() for map/array/
// object entries whose item type is "any"/undeclared.
// `selectedType` (optional) marks the matching <option> "selected" — used
// by the "Items are:" hint selector (dataEditSetItemTypeHint()) to reflect
// a container's current item-type hint; other callers omit it, leaving the
// browser's own default (the first option, "string") as before.
function extTypeOptionsHtml(selectedType) {
  var types = ['string', 'boolean', 'integer', 'uinteger', 'decimal',
    'timestamp', 'uri', 'url', 'xid', 'map', 'array', 'object'];
  return types.map(function(t) {
    return '<option value="' + t + '"' + (t === selectedType ? ' selected' : '') + '>' + t + '</option>';
  }).join('');
}

// Default "empty" value for a given type — used when a new map entry/array
// item/object key is added without an inline value field (see
// renderComplexAddButtonHtml() below): the entry is created with a
// sensible empty default and then edited in place afterward, exactly like
// a freshly-added complex value already works.
function defaultValueForType(t) {
  if (t === 'boolean') return false;
  if (t === 'integer' || t === 'uinteger' || t === 'decimal') return 0;
  if (t === 'array') return [];
  if (t === 'map' || t === 'object') return {};
  return '';
}

// Generates a key not already present in obj, e.g. "key1", "key2", ... —
// used for new map entries/object extension keys, whose name isn't asked
// for up front (see renderComplexAddButtonHtml()); the auto-generated key
// then appears as a normal, immediately-renamable row (its rename input
// works exactly like any other map/object key).
function genUniqueKey(obj, base) {
  base = base || 'key';
  var n = 1;
  while (Object.prototype.hasOwnProperty.call(obj, base + n)) n++;
  return base + n;
}

// Renders a map entry/array item/object key's "+ Add" control. Mirrors the
// array's existing "+ Add Item" button in NOT asking for any value up
// front — the new entry is created with an empty default value/key and
// then edited normally afterward (rename the key, fill in the value),
// rather than pre-collecting key/value in the add-row itself (which read
// oddly — data provided before the entry technically exists — and is
// altogether impossible for a nested map/object/array value anyway).
//
// When itemAttr's type is concrete (a real scalar type, or already a
// declared complex type), there's nothing to ask — a single button
// suffices. When it's "any"/undeclared (e.g. a map<any> or the object's
// "*" wildcard), the caller can't know in advance what shape the user
// wants, so this renders a two-zone "split button" (same visual pattern
// as the filter builder's "+ Add (AND) to" target picker): the button
// itself, plus a compact type <select> overlaid on its right edge. The
// button's onclick reads the currently-selected type out of that overlay.
function renderComplexAddButtonHtml(itemAttr, label, addId, onclickFnName, keyPathJson) {
  var isComplex = itemAttr && (itemAttr.type === 'object' || itemAttr.type === 'map' || itemAttr.type === 'array');
  var isAny = !isComplex && (!itemAttr || !itemAttr.type || itemAttr.type === 'any');
  if (!isAny) {
    return '<button class="editorBtn editorBtnSmall" '
      + 'onclick="' + onclickFnName + '(' + keyPathJson + ',' + esc(JSON.stringify(addId)) + ',' + esc(JSON.stringify(itemAttr.type)) + ')">'
      + label + '</button>';
  }
  return '<span class="xr-add-split">'
    + '<button class="editorBtn editorBtnSmall xr-add-split-btn" '
    + 'onclick="' + onclickFnName + '(' + keyPathJson + ',' + esc(JSON.stringify(addId)) + ',null)">' + label + '</button>'
    + '<select id="' + esc(addId) + '_t" class="xr-add-split-type">' + extTypeOptionsHtml() + '</select>'
    + '</span>';
}

// A map/array whose item type is "any"/undeclared can hold a mix of
// scalar/complex values (see effectiveEditAttr()) — but a user who wants
// it to behave as a homogeneous "map/array of one type" (matching a
// genuinely model-declared map<T>/array<T>, which never shows this
// selector at all) needs a way to say so, since an empty {}/[] alone
// can't record that intent in the JSON data itself. This renders that
// small "Items are: <select>" control (wired to
// dataEditSetItemTypeHint() below), shown alongside the container's own
// "+ Add" button whenever its item type isn't already fixed by the
// model. Choosing a concrete type here makes every subsequent Add use
// that type directly (renderComplexAddButtonHtml() no longer needs its
// own per-entry type overlay once the container's item type is no
// longer "any") — "(any)" clears the hint back to the original
// per-entry-typed behavior.
// Builds a properly-shaped attr for a bare type name (as chosen from a
// type <select>, e.g. an "Items are:" hint or an explicit add-time choice)
// — a plain {type:'map'/'object'/'array'} alone isn't enough for either
// shape to actually render/edit correctly (the map/array branches need an
// .item, the object branch needs a wildcarded .attributes to have
// anything to "+ Add" into), so any code recording a type choice for a
// complex shape should go through this rather than building the attr by
// hand, to avoid ending up with a shape that looks "empty" with no way to
// add children (see dataEditSetItemTypeHint()/effectiveEditAttrAtPath()).
function shapedAttrForType(t) {
  if (t === 'object') return {type: 'object', attributes: {'*': {type: 'any'}}};
  if (t === 'map') return {type: 'map', item: {type: 'any'}};
  if (t === 'array') return {type: 'array', item: {type: 'any'}};
  return {type: t};
}

function itemTypeHintSelectorHtml(keyPathJson, currentType) {
  return ' <span class="xr-item-hint">Items are '
    + '<select class="xr-item-hint-select" onchange="dataEditSetItemTypeHint(' + keyPathJson + ', this.value)">'
    + '<option value=""' + (!currentType || currentType === 'any' ? ' selected' : '') + '>(any)</option>'
    + extTypeOptionsHtml(currentType)
    + '</select></span>';
}

// Sets (or, if chosenType is empty, clears) the "items are always this
// type" hint for the map/array container at keyPath — see
// itemTypeHintSelectorHtml() above. Purely a rendering hint (see
// _itemTypeHints/effectiveItemAttr()); never written into the actual
// JSON data.
function dataEditSetItemTypeHint(keyPath, chosenType) {
  var key = JSON.stringify(keyPath);
  if (!chosenType) delete _itemTypeHints[key];
  else _itemTypeHints[key] = shapedAttrForType(chosenType);
  rerenderActiveEdit();
}

// Resolves a map/array's actual item attr for rendering: the user's own
// "Items are:" hint (see above) takes priority when set, then the
// model-declared item type, then a plain string fallback — same
// resolution order/purpose as effectiveEditAttr(), but for the
// container's *declared* item shape rather than one entry's actual
// runtime value.
function effectiveItemAttr(keyPath, attr) {
  return _itemTypeHints[JSON.stringify(keyPath)] || (attr && attr.item) || {type: 'string'};
}

// Renders a "type" <select> + a matching value input for choosing an
// extension attribute's actual JSON type before typing its value —
// extension attributes are declared "any" by the model's wildcard ("*"),
// so without this every new extension would always end up a plain string
// regardless of what the user actually intended (e.g. a boolean flag or a
// number). Now only used by the "Add new entity" form's own "+ Add
// Extension" row (addNewEntityAddExtension()) — the existing-entity/Meta
// tab "+ Add Extension" rows use the split-button pattern instead (see
// dataEditAddExtensionV2()). `valId`'s DOM node gets swapped out (via
// extTypeWidgetTypeChanged()) each time the type selector changes, since a
// checkbox/number/text input aren't interchangeable via attribute tweaks
// alone.
function renderExtTypeValueWidgetHtml(typeId, valId) {
  return '<select id="' + esc(typeId) + '" class="xr-edit-input" style="width:90px" '
    + 'onchange="extTypeWidgetTypeChanged(' + esc(JSON.stringify(typeId)) + ',' + esc(JSON.stringify(valId)) + ')">'
    + extTypeOptionsHtml()
    + '</select>'
    + ' <input type="text" id="' + esc(valId) + '" class="xr-edit-input" placeholder="value" style="width:160px">';
}

// onchange handler for the type <select> above — swaps the value input's
// DOM node to match (checkbox for boolean, number for integer/uinteger/
// decimal, a disabled placeholder for map/array/object since those start
// out as an empty container to be edited afterward — not typed inline —
// text otherwise), preserving its id so readExtTypeValue() below can
// still find it.
function extTypeWidgetTypeChanged(typeId, valId) {
  var typeEl = document.getElementById(typeId);
  var valEl  = document.getElementById(valId);
  if (!typeEl || !valEl) return;
  var t = typeEl.value;
  var replacement;
  if (t === 'boolean') {
    replacement = '<input type="checkbox" id="' + esc(valId) + '" class="xr-edit-input">';
  } else if (t === 'integer' || t === 'uinteger' || t === 'decimal') {
    replacement = '<input type="number" id="' + esc(valId) + '" class="xr-edit-input" style="width:160px" '
      + (t === 'decimal' ? 'step="any"' : 'step="1"') + '>';
  } else if (t === 'map' || t === 'array' || t === 'object') {
    replacement = '<input type="text" id="' + esc(valId) + '" class="xr-edit-input" style="width:160px" '
      + 'value="(empty ' + (t === 'array' ? '[]' : '{}') + ' \u2014 edit after adding)" disabled>';
  } else {
    replacement = '<input type="text" id="' + esc(valId) + '" class="xr-edit-input" placeholder="value" style="width:160px">';
  }
  valEl.outerHTML = replacement;
}

// Reads the current value out of an extension-widget pair, converted to
// match the chosen type (see renderExtTypeValueWidgetHtml() above). Complex
// types (map/array/object) always start out as an empty container — see
// extTypeWidgetTypeChanged() — which then renders with its own nested
// editor (Add Item/+Add/wildcard "+Add") immediately after being added,
// same as a complex map-entry's value (see dataEditMapAddEntry()).
function readExtTypeValue(typeId, valId) {
  var typeEl = document.getElementById(typeId);
  var valEl  = document.getElementById(valId);
  var t = typeEl ? typeEl.value : 'string';
  if (t === 'boolean') return !!(valEl && valEl.checked);
  if (t === 'integer' || t === 'uinteger') return (!valEl || valEl.value === '') ? 0 : parseInt(valEl.value, 10);
  if (t === 'decimal') return (!valEl || valEl.value === '') ? 0 : parseFloat(valEl.value);
  if (t === 'array') return [];
  if (t === 'map' || t === 'object') return {};
  return valEl ? valEl.value : '';
}

function undoDataEdit() {
  _dataDirty = false;
  _dataEditData = deepClone(_dataEditSrc);
  renderSingleEntity(_lastData || _dataEditSrc);
}

// Deletes the entity currently on-screen, after confirmation, then
// navigates back up to its parent (collection) page.
function deleteDataEntity() {
  var typeLabel = capitalize(getSingularName(_modelCache[normalizeURL(_state.serverURL || window.location.origin)], _state.path) || 'entity');
  if (!confirm('Delete this ' + typeLabel + '? This cannot be undone.')) return;
  var errDiv = document.getElementById('dataEditorError');
  if (errDiv) { hideErrorBanner(errDiv); }
  fetch(buildBaseURL(), {method: 'DELETE'}).then(function(resp) {
    if (resp.ok || resp.status === 204) {
      _dataDirty = false; _dataEditData = null; _dataEditSrc = null; _dataLoadedFor = null;
      invalidateRegistryProbe(_state.serverURL || window.location.origin);
      pushState({path: _state.path.slice(0, -1)});
      return;
    }
    return resp.text().then(function(text) {
      if (errDiv) { showErrorBanner(errDiv, 'Error (' + resp.status + '):\n' + text); }
    });
  }).catch(function(err) {
    if (errDiv) { showErrorBanner(errDiv, 'Network error: ' + err.message); }
  });
}

// Saves the working copy via PUT (full replacement) or PATCH (only the
// attributes that differ from the pristine snapshot — a shallow top-level
// diff, matching the fact that v1 only supports editing top-level scalar
// attributes/extensions). Resource/Version entities with a document body
// must target the $details-suffixed URL — bare PATCH is otherwise rejected
// server-side (registry/httpStuff.go's metaInBody check).
function saveDataEntity(verb, cb) {
  if (!_dataEditData) return;
  var errDiv = document.getElementById('dataEditorError');
  if (errDiv) { hideErrorBanner(errDiv); }

  var url = buildBaseURL();
  // Only Resource pages (depth 4) need the $details suffix here — the
  // dedicated Version page (depth >= 6) was retired from List view (see
  // normalizeVersionDepth()); a Version's own edits now go through
  // saveVersionEntity() instead, which has its own matching check.
  if (_state.path.length === 4) {
    var svBaseS = (_state.serverURL || window.location.origin).replace(/\/$/, '');
    var modelS = _modelCache[normalizeURL(svBaseS)] || null;
    if (resourceHasDocument(modelS, _state.path)) {
      url = url.replace(/(\?|$)/, '$details$1');
    }
  }

  var body;
  if (verb === 'PATCH') {
    body = {};
    Object.keys(_dataEditData).forEach(function(k) {
      if (JSON.stringify(_dataEditData[k]) !== JSON.stringify(_dataEditSrc[k])) body[k] = _dataEditData[k];
    });
  } else {
    body = _dataEditData;
  }

  var overlay = document.createElement('div'); overlay.className = 'savingOverlay';
  var box = document.createElement('div'); box.className = 'savingBox';
  var spinner = document.createElement('div'); spinner.className = 'savingSpinner';
  var msg = document.createElement('div'); msg.textContent = 'Saving\u2026';
  box.appendChild(spinner); box.appendChild(msg); overlay.appendChild(box);
  document.body.appendChild(overlay);
  function removeOverlay() { if (overlay.parentNode) overlay.parentNode.removeChild(overlay); }

  fetch(url, {
    method: verb,
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(body, null, 2)
  }).then(function(resp) {
    return resp.text().then(function(text) {
      removeOverlay();
      if (resp.ok) {
        var updated = null;
        try { updated = JSON.parse(text); } catch (e) { /* fall back below */ }
        _dataDirty = false;
        _dataEditSrc  = updated ? deepClone(updated) : deepClone(_dataEditData);
        _dataEditData = deepClone(_dataEditSrc);
        _lastData = _dataEditSrc;
        if (cb) { cb(); } else { renderSingleEntity(_dataEditSrc); }
      } else {
        if (errDiv) { showErrorBanner(errDiv, 'Error (' + resp.status + '):\n' + text); }
      }
    });
  }).catch(function(err) {
    removeOverlay();
    if (errDiv) { showErrorBanner(errDiv, 'Network error: ' + err.message); }
  });
}

// ---- Meta tab (Resource page "<Type> Metadata") editing -------------------
//
// Mirrors dataEditFieldChange()/markDataDirty()/dataEditAddExtensionV2()/
// undoDataEdit()/saveDataEntity() above almost exactly, but targets
// _metaEditData/_metaEditSrc/_metaDirty and the Meta entity's own URL
// (metaurl) instead of the Resource/Version entity's. Kept as separate
// functions (rather than parameterizing the entity-edit ones) since the
// two save paths differ in a few small but real ways: no $details
// suffixing (Meta is never a document-bearing entity), no PATCH-vs-full
// entity distinction beyond the same shallow top-level diff, and no
// delete/navigate-away action (a Resource's Meta sub-entity isn't
// independently deletable).

// onchange handler for Meta's own top-level scalar inputs — passed as the
// `handlerFn` override to renderEditableScalarInput() (see renderMetaTable()).
function metaEditFieldChange(k, inputEl) {
  if (!_metaEditData) return;
  var type = inputEl.dataset.ftype || 'string';
  var val;
  if (type === 'boolean') {
    val = inputEl.checked;
  } else if (type === 'integer' || type === 'uinteger') {
    val = inputEl.value === '' ? null : parseInt(inputEl.value, 10);
  } else if (type === 'decimal') {
    val = inputEl.value === '' ? null : parseFloat(inputEl.value);
  } else {
    val = inputEl.value;
  }
  var model = _modelCache[normalizeURL(_state.serverURL || window.location.origin)] || null;
  var metaPath = (_state.path || []).concat(['meta']);
  var beforeAttrs = topLevelAttrsMapFor(model, metaPath, _metaEditData);
  _metaEditData[k] = val;
  markMetaDirty();
  // ifvalues reactivity — see reconcileIfValuesOnChange().
  if (reconcileIfValuesOnChange(model, metaPath, beforeAttrs, _metaEditData)) {
    rerenderMetaTab();
    return;
  }
  // defaultversionsticky gates whether defaultversionid is editable —
  // toggling it needs a full re-render so that row flips between its
  // read-only-link/hint form and its editable-input form immediately.
  // Turning sticky back off also reverts any in-progress defaultversionid
  // edit back to its original (server) value, since that edit can no
  // longer be saved/shown as editable once sticky is off.
  if (k === 'defaultversionsticky') {
    if (!val && _metaEditSrc && 'defaultversionid' in _metaEditSrc) {
      _metaEditData.defaultversionid = _metaEditSrc.defaultversionid;
    }
    rerenderMetaTab();
    return;
  }
  toggleRowDirty(inputEl, _metaEditSrc && JSON.stringify(val) !== JSON.stringify(_metaEditSrc[k]));
}

function markMetaDirty() {
  if (!_metaDirty) {
    _metaDirty = true;
    var pb = document.getElementById('metaSavePutBtn');   if (pb) pb.disabled = false;
    var ab = document.getElementById('metaSavePatchBtn'); if (ab) ab.disabled = false;
    var ub = document.getElementById('metaUndoBtn');      if (ub) ub.disabled = false;
  }
}

// Re-renders just the Meta tab's box in place (analogous to
// renderSingleEntity() for the main entity) — used by every nested
// mutation handler above (dataEditMapAddEntry() etc.) when
// _dataEditActiveKind is 'meta', and by dataEditAddExtensionV2()/
// undoMetaEdit() below.
function rerenderMetaTab() {
  var box = document.getElementById('eg-meta-box');
  if (!box || !_metaEditData) return;
  var svURL = normalizeURL(_state.serverURL || window.location.origin);
  box.innerHTML = renderMetaBoxContent(_metaEditData, _modelCache[svURL] || null, true);
  // The Save/Undo action bar now lives in the shared #dataEditorActionBarSlot
  // (see buildMetaActionBarHtml()), not inside this box's own innerHTML —
  // refresh it too so Undo/Save-success (both of which reset _metaDirty and
  // call this function) actually re-disable the buttons instead of leaving
  // the slot's stale (still-enabled) markup in place.
  var slot = document.getElementById('dataEditorActionBarSlot');
  if (slot && _dataEditActiveKind === 'meta') slot.innerHTML = buildMetaActionBarHtml();
}

function undoMetaEdit() {
  _metaDirty = false;
  _metaEditData = deepClone(_metaEditSrc);
  rerenderMetaTab();
}

// Meta tab's own Save (full)/Save (delta)/Undo action bar — rendered into
// the SAME shared #dataEditorActionBarSlot used by the page-level Entity
// Data Editor bar and the version-selector's per-version bar (see
// buildVersionActionBarHtml()), never appended inside the Meta panel's own
// content. Previously this bar was built inline at the bottom of
// renderMetaTable()'s returned HTML — landing far below the (often long)
// Meta properties table, while the ALWAYS-VISIBLE page-level entity bar
// sat right below the tab row showing its own (irrelevant, permanently
// disabled while only editing Meta) Save/Undo state. That made it look
// like "the" Save/Undo buttons never enabled, since the real ones were out
// of view. No Delete button here — a Resource's Meta sub-entity isn't
// independently deletable (see saveMetaEntity()'s header comment).
function buildMetaActionBarHtml() {
  return '<div id="metaEditorError" class="error-banner" style="display:none"></div>'
    + '<div class="editorActionBar">'
    + '<button class="editorBtn" id="metaSavePutBtn" onclick="saveMetaEntity(\'PUT\')"' + (_metaDirty ? '' : ' disabled') + '>Save (full)</button>'
    + ' <button class="editorBtn" id="metaSavePatchBtn" onclick="saveMetaEntity(\'PATCH\')"' + (_metaDirty ? '' : ' disabled') + '>Save (delta)</button>'
    + ' <button class="editorBtn" id="metaUndoBtn" onclick="undoMetaEdit()"' + (_metaDirty ? '' : ' disabled') + '>Undo</button>'
    + '</div>';
}

// Saves the Meta working copy via PUT/PATCH, same shallow top-level diff
// as saveDataEntity(). Targets the Resource's metaurl (captured off
// _lastData when the Meta tab was first loaded — see loadMetaDetails()),
// not buildBaseURL() (which always points at the Resource/Version entity).
function saveMetaEntity(verb, cb) {
  if (!_metaEditData) return;
  var errDiv = document.getElementById('metaEditorError');
  if (errDiv) { hideErrorBanner(errDiv); }

  var url = _metaEditSrc && _metaEditSrc.self;
  if (!url) { url = _lastData && _lastData.metaurl; }
  if (!url) return;

  var body;
  if (verb === 'PATCH') {
    body = {};
    Object.keys(_metaEditData).forEach(function(k) {
      if (JSON.stringify(_metaEditData[k]) !== JSON.stringify(_metaEditSrc[k])) body[k] = _metaEditData[k];
    });
  } else {
    body = _metaEditData;
  }

  var overlay = document.createElement('div'); overlay.className = 'savingOverlay';
  var box = document.createElement('div'); box.className = 'savingBox';
  var spinner = document.createElement('div'); spinner.className = 'savingSpinner';
  var msg = document.createElement('div'); msg.textContent = 'Saving\u2026';
  box.appendChild(spinner); box.appendChild(msg); overlay.appendChild(box);
  document.body.appendChild(overlay);
  function removeOverlay() { if (overlay.parentNode) overlay.parentNode.removeChild(overlay); }

  fetch(url, {
    method: verb,
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(body, null, 2)
  }).then(function(resp) {
    return resp.text().then(function(text) {
      removeOverlay();
      if (resp.ok) {
        var updated = null;
        try { updated = JSON.parse(text); } catch (e) { /* fall back below */ }
        _metaDirty = false;
        _metaEditSrc  = updated ? deepClone(updated) : deepClone(_metaEditData);
        _metaEditData = deepClone(_metaEditSrc);
        _metaData = _metaEditSrc;
        if (cb) { cb(); } else { rerenderMetaTab(); }
      } else {
        if (errDiv) { showErrorBanner(errDiv, 'Error (' + resp.status + '):\n' + text); }
      }
    });
  }).catch(function(err) {
    removeOverlay();
    if (errDiv) { showErrorBanner(errDiv, 'Network error: ' + err.message); }
  });
}

// ---- Resource page "Version:" dropdown editing (non-default version) ------
//
// Mirrors the Meta tab's editing functions immediately above almost
// exactly, but targets _verEditData/_verEditSrc/_verDirty and the selected
// version's own URL (_verSelfUrl) instead of the Resource's metaurl.
// Picking "Default" from the dropdown never uses this pair — it reuses
// _dataEditData/_dataDirty directly (see onVersionSelectChangeReal()),
// since the resource's own data already IS the default version's data.

// onchange handler for the version panel's own top-level scalar inputs —
// passed as the `handlerFn` override to renderEditableScalarInput() (see
// onVersionSelectChangeReal()/rerenderVersionPanel()).
function verEditFieldChange(k, inputEl) {
  if (!_verEditData) return;
  var type = inputEl.dataset.ftype || 'string';
  var val;
  if (type === 'boolean') {
    val = inputEl.checked;
  } else if (type === 'integer' || type === 'uinteger') {
    val = inputEl.value === '' ? null : parseInt(inputEl.value, 10);
  } else if (type === 'decimal') {
    val = inputEl.value === '' ? null : parseFloat(inputEl.value);
  } else {
    val = inputEl.value;
  }
  var beforeAttrs = topLevelAttrsMapFor(_resModel, _resPath, _verEditData);
  _verEditData[k] = val;
  markVerDirty();
  // ifvalues reactivity — see reconcileIfValuesOnChange().
  if (reconcileIfValuesOnChange(_resModel, _resPath, beforeAttrs, _verEditData)) {
    rerenderVersionPanel();
    return;
  }
  toggleRowDirty(inputEl, _verEditSrc && JSON.stringify(val) !== JSON.stringify(_verEditSrc[k]));
}

function markVerDirty() {
  if (!_verDirty) {
    _verDirty = true;
    var pb = document.getElementById('verSavePutBtn');   if (pb) pb.disabled = false;
    var ab = document.getElementById('verSavePatchBtn'); if (ab) ab.disabled = false;
    var ub = document.getElementById('verUndoBtn');      if (ub) ub.disabled = false;
  }
}

// Builds the Save/Undo/Delete action bar embedded directly in the version
// panel's own content (like the Meta tab's own bar) — the page-level
// "Entity Data Editor" action bar only ever targets the Default version,
// so a genuinely different selected version needs its own, scoped to
// _verEditData/_verSelfUrl.
function buildVersionActionBarHtml() {
  return '<div id="verEditorError" class="error-banner" style="display:none"></div>'
    + '<div class="editorActionBar">'
    + '<button class="editorBtn" id="verSavePutBtn" onclick="saveVersionEntity(\'PUT\')"' + (_verDirty ? '' : ' disabled') + '>Save (full)</button>'
    + ' <button class="editorBtn" id="verSavePatchBtn" onclick="saveVersionEntity(\'PATCH\')"' + (_verDirty ? '' : ' disabled') + '>Save (delta)</button>'
    + ' <button class="editorBtn" id="verUndoBtn" onclick="undoVersionEdit()"' + (_verDirty ? '' : ' disabled') + '>Undo</button>'
    + ' <button class="editorBtn editorBtnDanger" onclick="deleteVersionEntity()">Delete</button>'
    + '</div>';
}

// Re-renders just the version panel in place (analogous to
// rerenderMetaTab()) — used by every nested mutation handler above
// (dataEditMapAddEntry() etc.) when _dataEditActiveKind is 'version', and
// by undoVersionEdit()/saveVersionEntity() below.
function rerenderVersionPanel() {
  var panel = document.getElementById(_verPanelId || 'eg-doc-panel-defver');
  if (!panel || !_verEditData) return;
  panel.innerHTML = buildEntityPropsTableHtml(_verEditData, _verHeaderLabel || 'Version Property',
    _resModel, _resPath, null, true, 'verEditFieldChange');
  // Refresh the shared action-bar slot too (its Save/Undo buttons' disabled
  // state depends on _verDirty, which just changed) — see
  // onVersionSelectChangeReal()'s matching comment for why this lives in
  // the slot rather than appended to the panel itself.
  var actionBarSlot = document.getElementById('dataEditorActionBarSlot');
  if (actionBarSlot) actionBarSlot.innerHTML = buildVersionActionBarHtml();
}

function undoVersionEdit() {
  _verDirty = false;
  _verEditData = deepClone(_verEditSrc);
  rerenderVersionPanel();
}

// Deletes the currently-selected (non-default) version, after confirmation,
// then falls back the dropdown/panel back to "Default" and refreshes the
// versions list (the deleted id must disappear from its options).
function deleteVersionEntity() {
  if (!_verSelfUrl) return;
  if (!confirm('Delete this version? This cannot be undone.')) return;
  var errDiv = document.getElementById('verEditorError');
  if (errDiv) { hideErrorBanner(errDiv); }
  fetch(_verSelfUrl, {method: 'DELETE'}).then(function(resp) {
    if (resp.ok || resp.status === 204) {
      _verDirty = false; _verEditData = null; _verEditSrc = null; _verEditVid = null; _verSelfUrl = '';
      _resVersionsList = null;
      var sel = document.getElementById('eg-doc-version-select');
      if (sel) sel.value = 'default';
      onVersionSelectChangeReal('default');
      loadVersionsForSelect();
      return;
    }
    return resp.text().then(function(text) {
      if (errDiv) { showErrorBanner(errDiv, 'Error (' + resp.status + '):\n' + text); }
    });
  }).catch(function(err) {
    if (errDiv) { showErrorBanner(errDiv, 'Network error: ' + err.message); }
  });
}

// Saves the version working copy via PUT/PATCH, same shallow top-level
// diff as saveDataEntity()/saveMetaEntity(). Targets the version's own
// self URL ($details-suffixed when the resource has a document body, same
// rule as saveDataEntity()), not buildBaseURL() (which always points at
// whichever entity/version the main page URL is on, not the dropdown pick).
function saveVersionEntity(verb, cb) {
  if (!_verEditData || !_verSelfUrl) return;
  var errDiv = document.getElementById('verEditorError');
  if (errDiv) { hideErrorBanner(errDiv); }

  // _verSelfUrl comes verbatim from the version's own server-provided
  // "self" link (see onVersionSelectChangeReal()) — unlike saveDataEntity()'s
  // buildBaseURL() (client-constructed from the page path, needing a manual
  // $details suffix), the server already includes $details in "self" for
  // document-bearing resources, so no extra suffixing is needed (or safe —
  // appending again would double it up and corrupt the version id parsing).
  var url = _verSelfUrl;

  var body;
  if (verb === 'PATCH') {
    body = {};
    Object.keys(_verEditData).forEach(function(k) {
      if (JSON.stringify(_verEditData[k]) !== JSON.stringify(_verEditSrc[k])) body[k] = _verEditData[k];
    });
  } else {
    body = _verEditData;
  }

  var overlay = document.createElement('div'); overlay.className = 'savingOverlay';
  var box = document.createElement('div'); box.className = 'savingBox';
  var spinner = document.createElement('div'); spinner.className = 'savingSpinner';
  var msg = document.createElement('div'); msg.textContent = 'Saving\u2026';
  box.appendChild(spinner); box.appendChild(msg); overlay.appendChild(box);
  document.body.appendChild(overlay);
  function removeOverlay() { if (overlay.parentNode) overlay.parentNode.removeChild(overlay); }

  fetch(url, {
    method: verb,
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(body, null, 2)
  }).then(function(resp) {
    return resp.text().then(function(text) {
      removeOverlay();
      if (resp.ok) {
        var updated = null;
        try { updated = JSON.parse(text); } catch (e) { /* fall back below */ }
        _verDirty = false;
        _verEditSrc  = updated ? deepClone(updated) : deepClone(_verEditData);
        _verEditData = deepClone(_verEditSrc);
        if (_resVersionsList) {
          _resVersionsList = _resVersionsList.map(function(v) {
            return (itemNavKey(v) === _verEditVid) ? _verEditSrc : v;
          });
        }
        if (cb) {
          cb();
        } else {
          rerenderVersionPanel();
          var pillsBox = document.getElementById('eg-doc-pills');
          if (pillsBox) pillsBox.innerHTML = buildDocInfoPillsHtml(_verEditSrc);
          if (document.getElementById('eg-doc-preview-box')) {
            _docActiveVersionData = _verEditSrc;
            _docPreviewLoaded = true;
            loadDocumentPreview();
          }
        }
      } else {
        if (errDiv) { showErrorBanner(errDiv, 'Error (' + resp.status + '):\n' + text); }
      }
    });
  }).catch(function(err) {
    removeOverlay();
    if (errDiv) { showErrorBanner(errDiv, 'Network error: ' + err.message); }
  });
}



// row always renders the same (unbanded) shade, so grouping still reads
// as intentional. `startBand` lets a flat (single-run) list keep
// continuous banding when there is no grouping at all.
// `editable` (default false) opts a call site into edit-mode rendering —
// only renderSingleEntity()'s own top-level calls pass true; redraws of a
// *different* version (version-selector dropdown) or the Meta tab stay
// read-only regardless of _state.editMode, since editing there would
// target the wrong URL.
function buildPropsRowsHtml(keys, entityData, model, path, specLevel, singular, startBand, resourceSingular, editable, handlerFn) {
  var html = '';
  var depthB = path ? path.length : 0;
  keys.forEach(function(k, i) {
    var val = entityData[k];
    var display, valueCellClass = '';
    var attrType = getExplicitAttrType(model, path, k, entityData);
    var isAncestorLink = (k === 'ancestorid' || (k === 'versionid' && depthB === 4));
    var editAttr = editable ? getAttr(model, path, [k], entityData) : null;
    var effAttr = editable ? effectiveEditAttrAtPath([k], editAttr, val) : editAttr;
    var isComplexKind = effAttr
      && (effAttr.type === 'map' || effAttr.type === 'object' || effAttr.type === 'array');
    var isEditableRow = editable
      && !isDataEditReadOnly(k, editAttr, singular, resourceSingular)
      && !isAncestorLink
      && (isComplexKind || (val !== null && typeof val !== 'object'));
    if (isEditableRow && isComplexKind) {
      display = renderEditableComplexValue([k], val, effAttr, model, path, entityData);
      valueCellClass = ' class="cell-tree"';
    } else if (isEditableRow) {
      if (isAnyTypeUnset(effAttr, val)) {
        var anyAddId = 'anytype_' + k.replace(/[^a-zA-Z0-9]/g, '') + '_' + Math.random().toString(36).slice(2, 7);
        display = typePillHtml(effAttr)
          + renderComplexAddButtonHtml(effAttr, 'Set Value', anyAddId, 'dataEditSetAnyType', esc(JSON.stringify([k])));
      } else {
        display = typePillHtml(effAttr) + renderEditableScalarInput(k, val, effAttr, handlerFn);
      }
    } else if (val !== null && typeof val === 'object') {
      var isEmpty = Array.isArray(val) ? val.length === 0 : Object.keys(val).length === 0;
      display = isEmpty
        ? '<span class="vt-empty">empty</span>'
        : renderValueTree(val, 0, model, path, [k], entityData);
      valueCellClass = ' class="cell-tree"';
    } else if (val === undefined) {
      // A model-declared attribute merged into `keys` (see
      // buildEntityPropsTableHtml()'s "show all model attributes" edit-mode
      // support) that this entity has no value for at all, and which isn't
      // itself editable here (e.g. read-only/immutable) — distinct from an
      // explicit JSON null.
      display = '<span style="color:#999">not set</span>';
    } else if (val == null) {
      display = '<span style="color:#999">null</span>';
    } else if (typeof val === 'boolean') {
      var isValidatedAttr = (k === 'formatvalidated' || k === 'compatibilityvalidated');
      display = renderBoolBadge(val, isValidatedAttr ? '?' : undefined);
      // Format/compatibility validation reasons aren't shown as their own
      // row (see suppressed formatvalidatedreason/compatibilityvalidatedreason
      // in buildEntityPropsTableHtml) — instead the reason string is appended
      // right after the "Validated" pill on this same line, since it only
      // ever matters in context of that pill (a plain standalone row read
      // oddly separated from the pill it explains).
      if (isValidatedAttr) {
        var reasonVal = entityData[k + 'reason'];
        if (reasonVal) display += ' <span class="eg-value">' + esc(String(reasonVal)) + '</span>';
      }
      // Compatibility Validated's rule string (e.g. "backward") lives on the
      // resource's Meta object, not this flattened Resource/Version data —
      // prefixed here in parens once known (see _docPillsMetaCompat/
      // ensureDocPillsCompat(), shared with the Document tab's Compatibility
      // pill). May not be loaded yet on first render — refreshVersionDetailsPanel()
      // redraws this panel once the async Meta fetch resolves it.
      if (k === 'compatibilityvalidated' && _docPillsMetaCompat) {
        display = '<span class="eg-value eg-mono">(' + esc(_docPillsMetaCompat) + ')</span> ' + display;
      }
    } else if (k === 'ancestorid' || (k === 'versionid' && depthB === 4)) {
      // Link to the Resource page's version-selector dropdown for this
      // version (versionid only shown on the Resource page's "Version
      // Details" tab — the dedicated depth>=6 Version page was retired
      // from List view, see normalizeVersionDepth()).
      var vid = String(val);
      var vHref = pageHref(path.slice(0, 4).concat(['versions', vid]), versionURLById(vid));
      var vClick = 'navigateToVersionById(' + JSON.stringify(vid) + ')';
      display = '<a class="eg-value eg-mono eg-link" href="' + esc(vHref) + '" '
              + 'onclick="' + esc(guardedOnclick(vClick)) + '">' + esc(vid) + '</a>';
    } else if (k === 'icon' && specLevel && specLevel.icon && typeof val === 'string' && val.trim()) {
      // Spec-defined "icon" attribute (Registry/Group/Resource/Version) is a
      // URL to an image — show a small live preview next to the usual
      // clickable link, so users can see at a glance what it looks like
      // without leaving the page. An extension attribute that merely
      // happens to be named "icon" (specLevel.icon falsy at this depth)
      // still falls through to the generic scalar/link rendering below.
      // onerror hides the <img> (rather than showing a broken-image icon)
      // if the URL doesn't actually resolve to a loadable image.
      display = '<span class="eg-icon-preview-wrap">'
              + '<img src="' + esc(val) + '" class="eg-icon-preview" alt="" onerror="this.style.display=\'none\'">'
              + renderScalarValue(val, isMonoSpecAttr(k, specLevel, singular, path))
              + '</span>';
    } else {
      var isMono = isMonoSpecAttr(k, specLevel, singular, path)
        || (attrType !== null && attrType !== 'string');
      display = (attrType === 'timestamp')
        ? formatTimestampValue(String(val), isMono)
        : renderScalarValue(val, isMono);
    }
    var banded = (startBand + i) % 2 === 1;
    // Compare against the pristine snapshot matching *this* rendering's
    // handlerFn/data, not always _dataEditSrc — this same function is
    // reused for the Resource page's own data (dataEditFieldChange /
    // _dataEditSrc) AND the version-selector's non-Default version panel
    // (verEditFieldChange / _verEditSrc); hardcoding _dataEditSrc there
    // compared a version's fields (e.g. xid/self) against the *Default*
    // version's snapshot, falsely marking almost everything dirty.
    var dirtySrc = (handlerFn === 'verEditFieldChange') ? _verEditSrc : _dataEditSrc;
    var isDirty = editable && dirtySrc
      && JSON.stringify(val) !== JSON.stringify(dirtySrc[k]);
    var rowClasses = [];
    if (banded) rowClasses.push('xr-row-band');
    if (isDirty) rowClasses.push('xr-row-dirty');
    html += '<tr' + (rowClasses.length ? ' class="' + rowClasses.join(' ') + '"' : '') + '><td style="font-weight:bold;color:#444;width:200px">' + esc(labelFor(k, specLevel, singular))
          + '</td><td' + valueCellClass + '>' + display + '</td></tr>';
  });
  return html;
}


// Builds a "<Header>" / "Value" properties table for any entity-like JSON
// object (a Resource's own flattened-default-version data, or a specific
// Version item fetched separately) — shared by the initial Resource-page
// render and by the version-selector dropdown's onVersionSelectChange(),
// which redraws this same panel with a different version's data.
// collKeys (optional) suppresses <plural>url/<plural>count sub-collection
// fields already shown elsewhere (e.g. "versionsurl"/"versionscount" on a
// Resource) — versions themselves have no sub-collections, so callers
// rendering a specific version can omit it. "meta"/"metaurl" are always
// suppressed here too — like the collection fields, they're structural
// navigation to a separate sub-entity (accessible via the Meta tab), not
// real content of this entity, and metaurl isn't caught by collKeys since
// there's no matching "metacount" partner for the *url/*count fallback scan.
function buildEntityPropsTableHtml(entityData, headerLabel, model, path, collKeys, editable, handlerFn) {
  // capabilities/model/modelsource are registry-root attributes the model
  // declares (type object), but none are editable inline from this entity
  // page (yet) — they each have their own dedicated viewer/editor UI
  // (Capabilities, Model, ModelSource nav entries) reached via the server
  // endpoints list, so surfacing them here in edit mode would just show a
  // dead-end "Content" section with nothing actually usable. Suppressed
  // like meta/metaurl above.
  var domainFocusedT = !effectiveXregFocused() && !editable;
  var suppressed = Object.assign({}, collKeys || {}, {meta: true, metaurl: true,
    formatvalidatedreason: true, compatibilityvalidatedreason: true,
    capabilities: true, model: true, modelsource: true});
  // Domain view: a version's own "ancestorid" pointing at itself
  // (the root/first version of a Resource) isn't meaningful info for an
  // end-user — hide the row entirely rather than showing an Ancestor
  // Version ID link to itself. xReg-focused view still shows it (accurate/
  // technically correct info there).
  if (domainFocusedT && entityData && entityData.ancestorid !== undefined
      && entityData.versionid !== undefined && entityData.ancestorid === entityData.versionid) {
    suppressed.ancestorid = true;
  }
  var priority = PROPS_PRIORITY;
  function sortKeys(arr) {
    return arr.sort(function(a, b) {
      var ai = priority.indexOf(a), bi = priority.indexOf(b);
      if (ai >= 0 && bi >= 0) return ai - bi;
      if (ai >= 0) return -1; if (bi >= 0) return 1;
      return a.localeCompare(b);
    });
  }
  var keys = sortKeys(Object.keys(entityData).filter(function(k) {
    return k !== '__mapKey' && !suppressed[k];
  }));
  // In edit mode, also surface every model-declared attribute this entity
  // hasn't set a value for at all yet (e.g. an unset "labels" map, or a
  // scalar extension attribute nobody has used) — otherwise there'd be no
  // way to give one a value for the first time. See plan.md "Full Data
  // Edit Mode" — "show all model attributes".
  if (editable && model) {
    var declaredAttrs = topLevelAttrsMapFor(model, path, entityData);
    var addedAny = false;
    Object.keys(declaredAttrs).forEach(function(ak) {
      if (ak === '*' || suppressed[ak] || keys.indexOf(ak) !== -1) return;
      keys.push(ak);
      addedAny = true;
    });
    if (addedAny) keys = sortKeys(keys);
  }
  if (!keys.length && !editable) return '<div class="eg-row"><span class="eg-value" style="color:#aaa">No properties</span></div>';
  var specLevel = specAttrLevel(path);
  var singular  = (getSingularName(model, path) || '').toLowerCase();
  var depth = path ? path.length : 0;
  // Only Resource pages (depth 4) have a distinct resourceSingular != singular
  // (the dedicated depth>=6 Version page was retired from List view, see
  // normalizeVersionDepth()).
  var resourceSingular = (depth === 4) ? singular : null;
  var groups = groupPropsByCategory(keys, specLevel, singular, resourceSingular, editable,
    depth === 0 ? 'Registry Metadata'
      : depth === 2 ? capitalize(singular) + ' Metadata'
      : capitalize(resourceSingular || singular) + ' Metadata');
  var html = '<table class="xr-table xr-table-props"><thead><tr><th>' + esc(headerLabel) + '</th><th></th></tr></thead><tbody>';
  if (groups) {
    // Domain view drops most category headers (see
    // buildPropsCatRowHtml()), so adjacent groups now sit directly next
    // to each other with no visual break — row banding needs to keep
    // counting across that boundary (running `bandT`) instead of
    // resetting to 0 per group, or two same-shaded rows can end up
    // touching. xRegistry-focused view keeps the original per-group
    // reset since its header row already breaks the shading.
    var bandT = 0;
    groups.forEach(function(g) {
      html += buildPropsCatRowHtml(g, domainFocusedT);
      html += buildPropsRowsHtml(g.keys, entityData, model, path, specLevel, singular, domainFocusedT ? bandT : 0, resourceSingular, editable, handlerFn);
      bandT += g.keys.length;
    });
  } else {
    html += buildPropsRowsHtml(keys, entityData, model, path, specLevel, singular, 0, resourceSingular, editable, handlerFn);
  }
  // Editable "+ Add Extension" row — only shown when the model declares a
  // wildcard ("*") attribute at this level, i.e. extensions are allowed.
  // When the model exists but doesn't declare "*" (extensions genuinely
  // disallowed by this model, per spec — the server itself will 400 an
  // unknown_attribute), show an explanatory muted row instead of silently
  // omitting the option — otherwise a user has no way to tell "extensions
  // aren't offered" apart from "this UI can't add extensions".
  if (editable && model) {
    var wildcardAttrEntity = getAttr(model, path, ['*'], entityData);
    if (wildcardAttrEntity) {
      var extAddId = 'dataExtAdd_' + Math.random().toString(36).slice(2, 7);
      html += '<tr><td style="font-weight:bold;color:#444;width:200px">+ Add Extension</td><td>'
        + '<input type="text" id="dataEditExtName" class="xr-edit-input" placeholder="name" style="width:140px">'
        + ' ' + renderComplexAddButtonHtml(wildcardAttrEntity, '+ Add', extAddId, 'dataEditAddExtensionV2', esc(JSON.stringify('dataEditExtName')))
        + '</td></tr>';
    } else {
      html += '<tr><td style="font-weight:bold;color:#444;width:200px">+ Add Extension</td><td>'
        + '<span style="color:#999;font-style:italic">This model does not allow extension attributes here</span>'
        + '</td></tr>';
    }
  }
  html += '</tbody></table>';
  return html;
}

// Whether the Resource page's Meta tab should render as editable right
// now — same _state.mutable ("entities" capability) + _state.editMode gate
// the main entity edit pipeline uses, but additionally scoped to depth 4
// (Resource) since that's the only depth the Meta tab exists at all
// (Version pages have no separate Meta split — see the tabDefs building
// code in the entity render function).
function metaEditableNow() {
  return _state.section === 'data' && _state.editMode && _state.mutable
    && _state.path && _state.path.length === 4;
}

// Fetches and renders the Metadata tab's own /meta sub-entity content —
// used by the Document/Details tab bar's "<Singular> Details" panel, which
// is always visible once selected. When metaEditableNow(), also snapshots
// _metaEditSrc/_metaEditData (the Meta tab's own pristine/working-copy
// pair — see saveMetaEntity()/undoMetaEdit()) so edit mode has something to
// diff/PATCH/undo against.
function loadMetaDetails() {
  var box = document.getElementById('eg-meta-box');
  if (!box) return;
  var editableM = metaEditableNow();
  // Snapshot which entity this call is actually for. `_metaData`/
  // `_metaLoadedFor` are reset in renderSingleEntity() whenever the
  // resource changes, but that reset can itself be raced: if a PREVIOUS
  // resource's meta fetch (or renderJSONViewForCurrentTab()'s own,
  // completely separate meta fetch — both read/write the same global
  // `_metaData`) is still in flight when the user navigates to a new
  // resource, its late-arriving response used to unconditionally stomp
  // `_metaData` with the WRONG resource's data right after the correct
  // reset had already happened. That stale, mismatched object then made
  // the very next `if (_metaData) {...}` check here look like "already
  // loaded", skipping the real fetch entirely (no new network request —
  // matching the reported symptom) — and if that mismatched data/model
  // combination happened to throw inside renderMetaBoxContent(), the box
  // was left on its "Loading…" placeholder forever. Guarding every
  // `_metaData` write/read here against `_lastData === requestedFor`
  // (i.e. "is this response still for the entity currently on screen?")
  // closes that hole regardless of fetch ordering.
  var requestedFor = _lastData;
  function stillCurrent() { return _lastData === requestedFor; }
  // Re-fetches `#eg-meta-box` right before each DOM write below (rather than
  // reusing the `box` reference captured above) because renderSingleEntity()
  // can legitimately re-render the whole page (replacing main.innerHTML,
  // and with it this exact element) while this function's own fetch is
  // still in flight — e.g. ensureModelCached()/ensureCapCached() resolving
  // asynchronously after the user has already switched to the Metadata tab.
  // Writing to the stale, now-detached `box` in that case is invisible and
  // silently leaves the *new* (currently-attached) placeholder stuck on
  // "Loading…" forever, since nothing else would ever refresh it. This was
  // the source of the "Meta tab hangs on Loading" bug reported after
  // navigating through several pages quickly.
  function afterLoad(d) {
    // Only (re)initialize the edit buffer the first time it's populated for
    // this resource — loadMetaDetails() can legitimately run again for the
    // SAME already-loaded resource (e.g. renderSingleEntity()'s harmless
    // re-render once ensureModelCached() resolves, which can race with an
    // in-progress edit — see the _metaLoadedFor guard above). Re-cloning
    // from `d` (the freshly re-fetched pristine server data) every time
    // would silently discard any not-yet-saved edit the user had already
    // made, even though the row still shows it as dirty.
    if (editableM && _metaEditData == null) {
      _metaEditSrc  = deepClone(d);
      _metaEditData = deepClone(_metaEditSrc);
      _metaDirty    = false;
    }
    var liveBox = document.getElementById('eg-meta-box');
    if (!liveBox) return;
    var svURL = normalizeURL(_state.serverURL || window.location.origin);
    liveBox.innerHTML = renderMetaBoxContent(editableM ? _metaEditData : d, _modelCache[svURL] || null, editableM);
  }
  if (_metaData && stillCurrent() && _metaLoadedFor === (_lastData && _lastData.self)) { afterLoad(_metaData); return; }
  box.innerHTML = '<div class="eg-loading">Loading\u2026</div>';
  var metaUrl = _lastData && _lastData.metaurl;
  if (!metaUrl) { box.innerHTML = '<div class="eg-row"><span class="eg-value" style="color:#aaa">No meta URL available</span></div>'; return; }
  fetch(metaUrl)
    .then(function(r) { if (!r.ok) throw new Error('HTTP ' + r.status); return r.json(); })
    .then(function(d) {
      if (!stillCurrent()) return; // navigated away before this resolved — discard
      _metaData = d;
      _metaLoadedFor = _lastData && _lastData.self;
      afterLoad(d);
    })
    .catch(function(e) {
      if (!stillCurrent()) return;
      var liveBox = document.getElementById('eg-meta-box');
      if (!liveBox) return;
      liveBox.innerHTML = '<div class="eg-row"><span class="eg-value" style="color:#c00;font-family:monospace">' + esc((e && e.message) ? e.message : String(e)) + '</span></div>';
    });
}

// Grows the Document tab's textarea to fill the remaining visible viewport
// space below it, so it uses as much room as possible without pushing the
// "Open in new tab" link (which sits right after it) below the fold. Re-run
// whenever the textarea's content/visibility changes or the window resizes.
function sizeDocTextarea() {
  var ta = document.querySelector('.eg-doc-textarea');
  if (!ta || ta.offsetParent === null) return; // not rendered / panel hidden
  var actions = ta.nextElementSibling;
  var reserve = (actions && actions.classList.contains('eg-doc-preview-actions')) ? actions.offsetHeight + 12 : 0;
  var available = window.innerHeight - ta.getBoundingClientRect().top - reserve - 16; // bottom breathing room
  ta.style.height = Math.max(200, Math.floor(available)) + 'px';
}

// Fetches the Resource page's versions collection (once) to populate the
// version-selector dropdown's options: "Default" first (already selected),
// then every version id, in the same order collectionItems() sorts them
// elsewhere (by map key). A fetch failure just leaves "Default" as the
// only option — non-critical, so no error UI is shown for this one.
function loadVersionsForSelect() {
  var sel = document.getElementById('eg-doc-version-select');
  if (!sel || !_resVersionsUrl) return;
  fetch(_resVersionsUrl)
    .then(function(r) { if (!r.ok) throw new Error('HTTP ' + r.status); return r.json(); })
    .then(function(d) {
      _resVersionsList = collectionItems(d);
      // Restore the version selected before a Refresh, if it still exists
      // in the collection (falls back to "default" otherwise). See
      // plan.md "Remember selected version + active tab".
      var restoredVid = (_state.resVersion && _resVersionsList.some(function(v) { return itemNavKey(v) === _state.resVersion; }))
        ? _state.resVersion : null;
      var current = restoredVid || sel.value || 'default';
      var html = '<option value="default"' + (current === 'default' ? ' selected' : '') + '>'
        + esc(defaultOptionLabel(_resDefaultData)) + '</option>';
      _resVersionsList.forEach(function(v) {
        var vid = itemNavKey(v);
        html += '<option value="' + esc(vid) + '"' + (current === vid ? ' selected' : '') + '>' + esc(vid) + '</option>';
      });
      sel.innerHTML = html;
      if (restoredVid) onVersionSelectChange(restoredVid);
      // If the Metadata tab is already active when this async fetch
      // resolves, re-apply the "N/A" disabled state on top of the
      // freshly-populated real options (otherwise they'd silently
      // replace it). See plan.md "Metadata tab disables version selector".
      var activeTabBtn = document.querySelector('.eg-doc-tab.active[data-tab]');
      if (activeTabBtn && activeTabBtn.getAttribute('data-tab') === 'meta') syncVersionSelectorForTab('meta');
    })
    .catch(function() { /* leave "Default" only — non-critical */ });
}

// Handles the standalone "Version:" dropdown's change event (Resource page
// only): redraws the "Version Details" tab panel with the picked version's
// own data, and refreshes the Document tab to match (whether or not it's
// the currently-visible tab), reusing the same buildEntityPropsTableHtml()/
// loadDocumentPreview() logic as the initial render. This dropdown is a
// separate control from the tab bar (see plan.md "version-selector
// dropdown") — it only changes *which version's data* feeds the tabs, it
// does not itself switch tabs.
//
// Guards leaving an in-progress edit of the CURRENTLY-selected non-default
// version (_verDirty) before switching to a different one — same
// Save/Discard/Cancel pattern as switchDocTab()'s Meta-tab guard. Cancel
// reverts the <select>'s already-changed displayed value back (the native
// onchange event fires after the browser already updated it).
function onVersionSelectChange(vid) {
  if (_dataEditActiveKind === 'version' && _verDirty && vid !== _verEditVid) {
    var sel = document.getElementById('eg-doc-version-select');
    var prevVid = _resSelectedVersionId;
    showLeaveEditDialog(
      function() { saveVersionEntity('PUT', function() { onVersionSelectChangeReal(vid); }); },
      function() { _verDirty = false; _verEditData = deepClone(_verEditSrc); onVersionSelectChangeReal(vid); },
      function() { if (sel) sel.value = prevVid; },
      function() { saveVersionEntity('PATCH', function() { onVersionSelectChangeReal(vid); }); }
    );
    return;
  }
  onVersionSelectChangeReal(vid);
}

function onVersionSelectChangeReal(vid) {
  _resSelectedVersionId = vid;
  // Sync the URL in place (no navigation/refetch) so a later manual Refresh
  // restores this version — same idiom as pushStateReal()'s history.pushState,
  // just non-navigational. See plan.md "Remember selected version + active tab".
  _state.resVersion = (vid === 'default') ? '' : vid;
  history.replaceState(null, '', buildURL(_state));
  var panel = document.getElementById('eg-doc-panel-defver');
  var verData, headerLabel, collKeysForVer;
  // Editing a specific version requires the same _state.mutable/editMode
  // gate the Meta tab uses (metaEditableNow() is depth-4-only, same scope
  // the version-selector itself only ever exists at). See plan.md "Fix:
  // can't edit versions from the Resource page's dropdown".
  var editableNow = metaEditableNow();
  var handlerFn = 'dataEditFieldChange';
  if (vid === 'default') {
    // "Default" is the resource's own data — reuse the existing entity
    // working copy (_dataEditData) instead of a separate snapshot, so
    // edits here stay in sync with (and are saved by) the page's own
    // Entity Data Editor action bar, exactly as before this fix.
    verData = editableNow ? _dataEditData : _resDefaultData;
    headerLabel = versionPropHeaderLabel(true, verData && verData.versionid);
    collKeysForVer = _resCollKeys;
    _dataEditActiveKind = 'entity';
    _verEditVid = null;
  } else {
    var rawVerData = (_resVersionsList || []).filter(function(v) { return itemNavKey(v) === vid; })[0];
    if (!rawVerData) return;
    if (editableNow) {
      // Only re-snapshot when switching to a version we don't already
      // have an in-progress working copy for — re-selecting the same one
      // (e.g. switching tabs and back) must not clobber unsaved edits.
      if (_verEditVid !== vid) {
        _verEditSrc  = deepClone(rawVerData);
        _verEditData = deepClone(_verEditSrc);
        _verDirty    = false;
        _verEditVid  = vid;
        _verSelfUrl  = rawVerData.self || '';
      }
      verData = _verEditData;
      _dataEditActiveKind = 'version';
      handlerFn = 'verEditFieldChange';
    } else {
      verData = rawVerData;
    }
    headerLabel = versionPropHeaderLabel(false, vid);
    collKeysForVer = null; // versions have no sub-collections to suppress
  }
  if (!verData) return;
  _verPanelId = 'eg-doc-panel-defver';
  _verHeaderLabel = headerLabel;
  if (panel) {
    panel.innerHTML = buildEntityPropsTableHtml(verData, headerLabel, _resModel, _resPath, collKeysForVer, editableNow, handlerFn);
  }
  // The single shared action-bar slot (right below the "Version:"/tab-bar
  // row — see renderSingleEntity()) shows whichever action bar applies to
  // the CURRENTLY selected version: the page-level Entity Data Editor bar
  // (_dataEditorActionBarHtml, targets the Default version/_dataEditData)
  // when "Default" is selected, or this version's own scoped bar
  // (buildVersionActionBarHtml(), targets _verEditData/_verSelfUrl) for any
  // other selection — never both, and never appended to the bottom of the
  // property table itself (previously buildVersionActionBarHtml() was
  // appended inside the panel's own innerHTML, landing it below whatever
  // else happened to be in that panel instead of in the same fixed spot
  // used for every other version/tab).
  var actionBarSlot = document.getElementById('dataEditorActionBarSlot');
  if (actionBarSlot) {
    if (vid === 'default') {
      actionBarSlot.innerHTML = _dataEditorActionBarHtml;
    } else if (editableNow) {
      actionBarSlot.innerHTML = buildVersionActionBarHtml();
    } else {
      actionBarSlot.innerHTML = '';
    }
  }

  // Keep the Document tab showing the selected version's document too —
  // refresh it eagerly even if it's not the currently-visible tab.
  if (document.getElementById('eg-doc-preview-box')) {
    _docActiveVersionData = verData;
    _docPreviewLoaded = true;
    loadDocumentPreview();
  }
  var pillsBox = document.getElementById('eg-doc-pills');
  if (pillsBox) pillsBox.innerHTML = buildDocInfoPillsHtml(verData);
  refreshCopyLinkBtnTooltip();
}


// Guard wrapper around switchDocTabReal(): leaving the Metadata tab with
// unsaved edits (_metaDirty) offers Save / Discard / Cancel before the tab
// actually switches — same pattern as pushState()'s edit-mode guards, but
// scoped to the tab bar since Meta-tab edits are independent of (and don't
// block) switching to the Document/Version-Details tabs' own content.
//
// Bug fix: every "Discard" callback below must not just reset the dirty
// flag/working-copy variables — it must also re-render the panel being
// left. Tab panels are only ever built once and then shown/hidden via
// switchDocTabReal()'s CSS display toggle (not rebuilt on every switch),
// so without an explicit re-render here the OLD (pre-discard, still
// showing the discarded edits with Save/Undo left enabled) markup simply
// sits untouched in the DOM and reappears exactly as it was the next time
// the user switches back to that tab — even though the underlying data/
// dirty-flag state was correctly reset. See plan.md.
function switchDocTab(tabKey) {
  var curActive = document.querySelector('.eg-doc-tab.active');
  var curKey = curActive ? curActive.getAttribute('data-tab') : null;
  if (curKey === 'meta' && tabKey !== 'meta' && _metaDirty) {
    showLeaveEditDialog(
      function() { saveMetaEntity('PUT', function() { switchDocTabReal(tabKey); }); },
      function() { _metaDirty = false; _metaEditData = deepClone(_metaEditSrc); rerenderMetaTab(); switchDocTabReal(tabKey); },
      null,
      function() { saveMetaEntity('PATCH', function() { switchDocTabReal(tabKey); }); }
    );
    return;
  }
  // Meta is a wholly separate object (its own URL/working-copy) from the
  // entity/version data shown by the other tabs, so unsaved edits there
  // must guard a switch INTO Meta too, not just out of it. This covers
  // BOTH the Default-version panel's dirty state (_dataDirty) and a
  // non-default selected version's own in-progress edit (_verDirty) —
  // exactly one of the two can be active at a time, depending on which
  // version the "Version:" dropdown currently has selected (see
  // onVersionSelectChangeReal()) — previously only _dataDirty was
  // checked, so leaving a dirty non-default version's edit for the Meta
  // tab skipped the guard entirely.
  var enteringVersionDirty = _dataEditActiveKind === 'version' && _verDirty;
  if (curKey !== 'meta' && tabKey === 'meta' && (_dataDirty || enteringVersionDirty)) {
    showLeaveEditDialog(
      function() {
        if (enteringVersionDirty) saveVersionEntity('PUT', function() { switchDocTabReal(tabKey); });
        else saveDataEntity('PUT', function() { switchDocTabReal(tabKey); });
      },
      function() {
        if (enteringVersionDirty) {
          _verDirty = false; _verEditData = deepClone(_verEditSrc);
          rerenderVersionPanel();
        } else {
          _dataDirty = false; _dataEditData = deepClone(_dataEditSrc);
          renderSingleEntity(_lastData || _dataEditSrc);
        }
        switchDocTabReal(tabKey);
      },
      null,
      function() {
        if (enteringVersionDirty) saveVersionEntity('PATCH', function() { switchDocTabReal(tabKey); });
        else saveDataEntity('PATCH', function() { switchDocTabReal(tabKey); });
      }
    );
    return;
  }
  switchDocTabReal(tabKey);
}

// Toggle the active Document/Details tab: swaps the .active button class
// and shows only the matching panel, lazy-loading the Document preview or
// the Details meta content the first time each is shown (cached after that).
function switchDocTabReal(tabKey) {
  // Nested map/array/object edit handlers (dataEditNestedFieldChange() etc.)
  // are shared between the main entity Property table and the Meta tab —
  // see activeEditRoot(). Keep the flag in sync with whichever tab is now
  // active so those handlers always target the right working copy.
  // Nested map/array/object edit handlers (dataEditNestedFieldChange() etc.)
  // are shared between the main entity Property table, the Meta tab, and
  // the version-selector's non-default working copy — see activeEditRoot().
  // Keep the flag in sync with whichever tab is now active so those
  // handlers always target the right working copy. 'defver'/'version' both
  // stay 'version' rather than resetting to 'entity' if a non-default
  // version's working copy is currently in progress (_verEditVid set) —
  // otherwise switching tabs and back would silently retarget nested edits
  // at the wrong object.
  if (tabKey === 'meta') _dataEditActiveKind = 'meta';
  else if ((tabKey === 'defver' || tabKey === 'version') && _verEditVid) _dataEditActiveKind = 'version';
  else _dataEditActiveKind = 'entity';
  var tabs = document.querySelectorAll('.eg-doc-tab');
  tabs.forEach(function(t) { t.classList.toggle('active', t.getAttribute('data-tab') === tabKey); });
  var panels = document.querySelectorAll('.eg-doc-tab-panel');
  panels.forEach(function(p) { p.style.display = (p.getAttribute('data-tab') === tabKey) ? '' : 'none'; });
  // Sync the URL in place (no navigation/refetch) so a later manual Refresh
  // restores this tab — same idiom as onVersionSelectChange() above. Treat
  // the first tab as "default" (empty _state.docTab) so the URL stays clean
  // when the user is on the natural default tab, mirroring how 'default' is
  // handled for the version selector. See plan.md "Remember selected version
  // + active tab".
  var isFirstTab = tabs.length > 0 && tabs[0].getAttribute('data-tab') === tabKey;
  _state.docTab = isFirstTab ? '' : tabKey;
  history.replaceState(null, '', buildURL(_state));
  // Swap the shared #dataEditorActionBarSlot to whichever bar applies to
  // the tab we're switching to/away from — see buildMetaActionBarHtml() for
  // why Metadata needs its own slot content instead of showing the
  // page-level entity bar (which never reflects Meta's own dirty state).
  // Switching away from Metadata restores the page-level bar (Version
  // Details/Document share that one; the version-selector's own swap logic
  // in onVersionSelectChangeReal() takes over from there if a non-default
  // version is also selected).
  var actionBarSlotT = document.getElementById('dataEditorActionBarSlot');
  if (actionBarSlotT) {
    if (tabKey === 'meta' && _state.editMode && _state.mutable) {
      actionBarSlotT.innerHTML = buildMetaActionBarHtml();
    } else if ((tabKey === 'defver' || tabKey === 'version') && _verEditVid) {
      // A non-default version's own working copy is in progress (same
      // condition used above for _dataEditActiveKind) — restore its scoped
      // bar instead of the page-level one, matching
      // onVersionSelectChangeReal()'s own slot-swap logic.
      actionBarSlotT.innerHTML = buildVersionActionBarHtml();
    } else {
      actionBarSlotT.innerHTML = _dataEditorActionBarHtml;
    }
  }
  if (tabKey === 'doc' && !_docPreviewLoaded) { _docPreviewLoaded = true; loadDocumentPreview(); }
  // Always call loadMetaDetails() here (rather than gating on `!_metaData`)
  // and let IT decide whether `_metaData` is actually still valid for the
  // CURRENT resource. `_metaData` is a global shared with
  // renderJSONViewForCurrentTab(), so a stale, non-null leftover from a
  // previously-viewed resource used to make this gate wrongly skip
  // loadMetaDetails() entirely — the Metadata panel's placeholder HTML
  // (never actually populated for THIS resource) was then left showing
  // "Loading…" forever, and no fetch ever fired (matching the "no network
  // traffic" symptom exactly). See loadMetaDetails()'s own
  // `stillCurrent()`/`_metaLoadedFor` guard, which correctly reuses the
  // cache when it's actually still valid, so calling it unconditionally
  // here is cheap and safe either way.
  if (tabKey === 'meta') { loadMetaDetails(); }
  // The panel was just made visible (or already was) — resize the textarea
  // now that layout/geometry is accurate (hidden panels report 0 height).
  if (tabKey === 'doc') sizeDocTextarea();
  syncVersionSelectorForTab(tabKey);
  refreshCopyLinkBtnTooltip();
}

// Metadata (metaurl) is a per-Resource concept, not per-version, so the
// version-selector dropdown has no effect while the Metadata tab is
// active. Swap it to a disabled "N/A" state in that case (both visually
// greyed and unclickable), remembering/restoring the previously-selected
// version so switching back to another tab picks up right where the user
// left off. See plan.md "Metadata tab disables version selector".
function syncVersionSelectorForTab(tabKey) {
  var sel = document.getElementById('eg-doc-version-select');
  if (!sel) return;
  if (tabKey === 'meta') {
    // Never stash "__na__" itself as the value to restore later — that
    // would happen if this runs more than once while already on the
    // Metadata tab (e.g. loadVersionsForSelect()'s async options-refresh
    // re-invoking this) and would permanently lose the real selection.
    if (sel.dataset.prevValue === undefined && sel.value !== '__na__') sel.dataset.prevValue = sel.value;
    var naOpt = sel.querySelector('option[value="__na__"]');
    if (!naOpt) {
      naOpt = document.createElement('option');
      naOpt.value = '__na__';
      naOpt.textContent = 'N/A';
      sel.insertBefore(naOpt, sel.firstChild);
    }
    sel.value = '__na__';
    sel.disabled = true;
    sel.title = 'Metadata is the same for all versions';
  } else {
    sel.disabled = false;
    sel.title = '';
    var naOpt2 = sel.querySelector('option[value="__na__"]');
    if (naOpt2) naOpt2.remove();
    if (sel.dataset.prevValue !== undefined) {
      sel.value = sel.dataset.prevValue;
      delete sel.dataset.prevValue;
    }
  }
}

// Sniffs raw bytes (NOT the declared contenttype) to decide binary vs. text:
// a NUL byte anywhere, or a high ratio of non-printable/non-whitespace
// control characters in a leading sample, is treated as binary content.
function isBinaryContent(buf) {
  var bytes = new Uint8Array(buf);
  var len = Math.min(bytes.length, 8000);
  if (len === 0) return false;
  var suspicious = 0;
  for (var i = 0; i < len; i++) {
    var b = bytes[i];
    if (b === 0) return true;
    if (b < 32 && b !== 9 && b !== 10 && b !== 13) suspicious++;
  }
  return (suspicious / len) > 0.05;
}

function decodeUTF8Bytes(buf) {
  try { return new TextDecoder('utf-8', { fatal: false }).decode(buf); }
  catch (e) {
    var bytes = new Uint8Array(buf), s = '';
    for (var i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i]);
    return s;
  }
}

// Small info-pill row shown in the Document tab, between the tab button
// row and the document preview — surfaces the doc-specific attributes
// (contenttype, format, and the format/compatibility validation results)
// without needing to switch to the Version Details tab. Each pill only
// appears when its underlying attribute is actually set; returns '' when
// none apply (no empty row rendered). Content-Type (no validation result
// to show) reuses the Labels map's two-tone key/value pill idiom
// (.eg-label-key/.eg-label-val); Format/Compatibility (which each have a
// pass/fail result) use the dedicated 3-section pill (docPill3Html —
// label | value | badge) instead, so the value->badge association is
// visually unambiguous — see plan.md "pill design consistency".
//
// Compatibility's actual rule value (e.g. "backward") lives on the Meta
// object (ENTITY_META), not the flattened Resource/Version data, so it
// isn't available synchronously here — the caller kicks off
// ensureDocPillsCompat() separately and this function just reads whatever
// is currently cached in _docPillsMetaCompat (null until fetched), same
// pattern as _docActiveVersionData for the version-selector.
function buildDocInfoPillsHtml(data) {
  if (!data) return '';
  var pills = [];
  if (data.contenttype) {
    pills.push('<span class="eg-doc-pill-item">' + docKeyValPillHtml('Content-Type', data.contenttype) + '</span>');
  }
  if (data.format) {
    pills.push(docPill3Html('Format', data.format, data.formatvalidated, data.formatvalidatedreason));
  }
  if (data.compatibilityvalidated === true || data.compatibilityvalidated === false) {
    pills.push(docPill3Html('Compatibility', _docPillsMetaCompat, data.compatibilityvalidated, data.compatibilityvalidatedreason));
  }
  return pills.length ? '<div class="eg-doc-pills">' + pills.join('') + '</div>' : '';
}

// Two-tone key/value pill — same markup/classes as the Labels map's pills
// (buildPropsRowsHtml()'s labelParts.map(...)) so Format/Content-Type read
// as the same "attribute tag" idiom used elsewhere in the app.
function docKeyValPillHtml(key, val) {
  return '<span class="eg-label-pair"><span class="eg-label-key">' + esc(key) + '</span>'
       + '<span class="eg-label-val eg-mono">' + esc(val) + '</span></span>';
}

// Fetches the resource's Meta object (if not already cached — shared with
// the "<Type> Metadata" tab's _metaData, so visiting that tab first avoids
// a second fetch) purely to read the "compatibility" rule string (e.g.
// "backward") for the Document tab's Compatibility pill. Only the *value*
// is meta-level; whether it validated (compatibilityvalidated) is already
// version-level and available synchronously. A no-op once
// _docPillsMetaCompat is non-null (already fetched, or already determined
// there's nothing to fetch — see buildDocInfoPillsHtml()).
function ensureDocPillsCompat(data) {
  if (_docPillsMetaCompat !== null) return;
  if (!data || (data.compatibilityvalidated !== true && data.compatibilityvalidated !== false)) {
    _docPillsMetaCompat = '';
    return;
  }
  if (_metaData && _metaData.compatibility) {
    _docPillsMetaCompat = _metaData.compatibility;
    refreshDocPills(data);
    refreshVersionDetailsPanel();
    return;
  }
  var metaUrl = resolveResourceMetaUrl(data);
  if (!metaUrl) { _docPillsMetaCompat = ''; return; }
  fetch(metaUrl)
    .then(function(r) { if (!r.ok) throw new Error('HTTP ' + r.status); return r.json(); })
    .then(function(d) {
      _docPillsMetaCompat = (d && d.compatibility) || '';
      refreshDocPills(_docActiveVersionData || data);
      refreshVersionDetailsPanel();
    })
    .catch(function() { _docPillsMetaCompat = ''; });
}

// Re-renders the #eg-doc-pills row in place once the async Meta fetch
// above (or _metaData, if already loaded) resolves the Compatibility
// pill's value.
function refreshDocPills(data) {
  var pillsBox = document.getElementById('eg-doc-pills');
  if (pillsBox) pillsBox.innerHTML = buildDocInfoPillsHtml(data);
}

// Re-renders the "Version Details"/"<Type> Property" panel in place once
// ensureDocPillsCompat()'s async Meta fetch resolves _docPillsMetaCompat —
// so the Compatibility Validated row picks up its "(compat)" value prefix
// (see buildPropsRowsHtml()) without requiring a manual refresh. No-op if
// that panel isn't currently in the DOM (e.g. user navigated away already)
// or nothing was snapshotted for it (see _docPropsPanelInfo).
function refreshVersionDetailsPanel() {
  if (!_docPropsPanelInfo) return;
  var panel = document.getElementById(_docPropsPanelInfo.panelId);
  if (!panel) return;
  var data = _docActiveVersionData || _resDefaultData;
  if (!data) return;
  panel.innerHTML = buildEntityPropsTableHtml(data, _docPropsPanelInfo.headerLabel,
    _docPropsPanelInfo.model, _docPropsPanelInfo.path, _docPropsPanelInfo.collKeys);
}


// Resource-level metaurl is only present on the Resource page's own
// flattened data (depth 4, ENTITY_RESOURCE-level attribute) — not on
// individual Version pages/entries (depth 6+, or versions-collection
// items), since it's a per-resource, not per-version, concept. On a
// Version page/entry we derive it from that version's own "self" URL by
// stripping the trailing "/versions/<id>" segment and appending "/meta",
// following the same convention the server itself uses to build metaurl
// from a resource's self.
function resolveResourceMetaUrl(data) {
  if (!data) return '';
  if (data.metaurl) return data.metaurl;
  if (data.self) {
    var m = data.self.replace(/\/versions\/[^\/]+\/?$/, '');
    if (m !== data.self) return m.replace(/\/$/, '') + '/meta';
  }
  return '';
}

// Renders a "3-section" pill — label | value | validation-status badge —
// for a doc attribute that has both a value and a validation result
// (Format, Compatibility). A single bordered container with the
// label/value/badge as contiguous flush segments (rather than two
// separately-boxed pills side by side) makes the label->value->badge
// association unambiguous at a glance, and gives Format/Compatibility a
// matching, more prominent shape than the plain Content-Type pill (which
// has no badge to attach). See plan.md "pill design consistency" for the
// discussion behind this.
//   - label:     e.g. "Format", "Compatibility" — always shown.
//   - value:     e.g. "avro/1.9.0" — omitted (no value segment) when
//                falsy, e.g. Compatibility's value hasn't finished its
//                async meta fetch yet, or genuinely isn't set.
//   - validated: true/false/undefined — the matching "<x>validated"
//                boolean; omitted (no badge segment) when neither true
//                nor false (attribute not applicable). false does NOT
//                mean something is wrong — it just means the server
//                hasn't checked — so it gets the same neutral gray
//                treatment as an ordinary boolean "false", not a red
//                "failure" color (see .eg-doc-pill3-fail in style.css).
//   - reason:    the matching "<x>validatedreason" attribute's value, if
//                any — shown in a popup when the gray X badge is clicked.
//                Only made clickable when a reason is actually present;
//                otherwise there's nothing useful to show.
function docPill3Html(label, value, validated, reason) {
  var html = '<span class="eg-doc-pill3">';
  html += '<span class="eg-doc-pill3-label">' + esc(label) + '</span>';
  if (value) {
    html += '<span class="eg-doc-pill3-value eg-mono">' + esc(value) + '</span>';
  }
  if (validated === true) {
    html += '<span class="eg-doc-pill3-badge eg-doc-pill3-ok" title="' + esc(label) + ' validated">\u2713</span>';
  } else if (validated === false) {
    if (reason) {
      var onclickExpr = 'showValidationReasonPopup(' + JSON.stringify(label) + ', ' + JSON.stringify(reason) + ')';
      html += '<span class="eg-doc-pill3-badge eg-doc-pill3-fail eg-doc-pill3-clickable" title="Click for details" onclick="'
        + esc(onclickExpr) + '">?</span>';
    } else {
      html += '<span class="eg-doc-pill3-badge eg-doc-pill3-fail" title="' + esc(label) + ' not validated">?</span>';
    }
  }
  html += '</span>';
  return html;
}

// Small modal popup showing why a format/compatibility validation didn't
// pass — opened by clicking the X badge in the Document tab's info pills
// (see buildDocInfoPillsHtml()/docPill3Html()), only shown when a
// "<x>validatedreason" value is actually available. Reuses the same plain
// overlay+box pattern as showLeaveEditDialog() (no existing generic modal
// helper to share).
function showValidationReasonPopup(label, reason) {
  var overlay = document.createElement('div');
  overlay.style.cssText = 'position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.35);z-index:9999;display:flex;align-items:center;justify-content:center;';
  // Clicking the dimmed backdrop (not the box itself) closes the popup,
  // same as clicking "Close" — e.target check ensures clicks inside the
  // box (which don't stop propagation) don't also trigger this.
  overlay.onclick = function(e) { if (e.target === overlay) document.body.removeChild(overlay); };
  var box = document.createElement('div');
  box.style.cssText = 'background:white;border-radius:8px;padding:20px 24px;box-shadow:0 4px 24px rgba(0,0,0,0.25);max-width:420px;width:90%;font-family:sans-serif;';
  var title = document.createElement('div');
  title.textContent = label + ' Not Validated';
  title.style.cssText = 'font-weight:bold;font-size:14px;margin-bottom:10px;color:#333;';
  box.appendChild(title);
  var msg = document.createElement('p');
  msg.textContent = reason || 'No reason provided.';
  msg.style.cssText = 'margin:0 0 18px;font-size:13px;color:#333;white-space:pre-wrap;';
  box.appendChild(msg);
  var btns = document.createElement('div');
  btns.style.cssText = 'display:flex;justify-content:flex-end;';
  var closeBtn = document.createElement('button');
  closeBtn.textContent = 'Close';
  closeBtn.style.cssText = 'padding:6px 16px;border-radius:5px;cursor:pointer;font-size:13px;font-weight:bold;background:#f0f0f0;color:#333;border:1px solid #ccc;';
  closeBtn.onclick = function() { document.body.removeChild(overlay); };
  btns.appendChild(closeBtn);
  box.appendChild(btns);
  overlay.appendChild(box);
  document.body.appendChild(overlay);
}

// Inline preview for the Document tab. Reuses openDocument()'s source-
// resolution priority (<key>url -> <key>base64 -> inline JSON value ->
// self with $details stripped) but fetches/decodes and renders inline
// (read-only textarea for text, "Binary file" message for binary) instead
// of always opening a new tab. A small "Open in new tab" link/button is
// always shown alongside the result (and as the sole fallback on error).
function loadDocumentPreview() {
  var box = document.getElementById('eg-doc-preview-box');
  if (!box) return;
  var singular = _docSingular;
  var data = _docActiveVersionData || _lastData;
  if (!singular || !data) { box.innerHTML = '<div class="eg-doc-binary-msg">Document not available.</div>'; return; }
  var key = singular.toLowerCase();

  function openTabBtn(url) {
    return url ? '<div class="eg-doc-preview-actions"><a href="' + esc(url) + '" target="_blank" rel="noopener" class="eg-link-btn">Open in new tab</a></div>' : '';
  }
  function showText(text, url) {
    box.innerHTML = '<textarea class="eg-doc-textarea" readonly>' + esc(text) + '</textarea>' + openTabBtn(url);
    sizeDocTextarea();
  }
  function showBinary(url) {
    box.innerHTML = '<div class="eg-doc-binary-msg">Binary file \u2014 preview not available.</div>' + openTabBtn(url);
  }
  function showError(msg, url) {
    box.innerHTML = '<div class="eg-doc-error-msg">' + esc(msg) + '</div>' + openTabBtn(url);
  }
  function fetchAndRender(url) {
    fetch(url)
      .then(function(r) { if (!r.ok) throw new Error('HTTP ' + r.status); return r.arrayBuffer(); })
      .then(function(buf) { isBinaryContent(buf) ? showBinary(url) : showText(decodeUTF8Bytes(buf), url); })
      .catch(function(e) { showError('Could not load document: ' + ((e && e.message) || String(e)), url); });
  }

  if (data[key + 'url']) { fetchAndRender(data[key + 'url']); return; }

  if (data[key + 'base64']) {
    try {
      var binary = atob(data[key + 'base64']);
      var bytes = new Uint8Array(binary.length);
      for (var i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
      var blobUrl = URL.createObjectURL(new Blob([bytes], { type: data.contenttype || 'application/octet-stream' }));
      if (isBinaryContent(bytes.buffer)) showBinary(blobUrl); else showText(decodeUTF8Bytes(bytes.buffer), blobUrl);
    } catch (e) { showError('Error decoding document: ' + ((e && e.message) || String(e))); }
    return;
  }

  if (data[key] !== undefined && data[key] !== null) {
    var json = JSON.stringify(data[key], null, 2);
    var blobUrl2 = URL.createObjectURL(new Blob([json], { type: 'application/json' }));
    showText(json, blobUrl2);
    return;
  }

  if (data.self) { fetchAndRender(data.self.replace(/\$details$/, '')); return; }

  box.innerHTML = '<div class="eg-doc-binary-msg">Document not available.</div>';
}

// Dispatch the meta box body to the format matching the current data view:
// List view gets a plain key/value table (like the entity's own Property
// table); Grid view keeps the richer label/row rendering (renderMetaContent).
// `editable` (default false) is only ever true for List view — Grid view's
// renderMetaContent() stays read-only regardless (matching how the rest of
// Full Data Edit Mode is List-view-only for now).
function renderMetaBoxContent(d, model, editable) {
  return _state.dataView === 'table' ? renderMetaTable(d, model, editable) : renderMetaContent(d, model);
}

// Plain table rendering of the meta/details data, for List view. Mirrors the
// "<Type> Property" / "Value" table used for the entity's own scalar props,
// so the meta box looks consistent with the rest of List view instead of
// Grid view's more human-readable label/row layout. Now fully model-driven
// (getAttr()/getExplicitAttrType() support the Meta level via the model's
// dedicated metaattributes map — see getAttr()) — timestamp detection and
// monospace formatting work exactly like buildEntityPropsTableHtml(), no
// hardcoded meta-only special-casing needed.
//
// `editable` opts into the same in-place edit-mode rendering
// buildEntityPropsTableHtml()/buildPropsRowsHtml() give the entity's own
// Property table — scalar inputs, map/object/array editors, dirty-row
// highlighting, and a "+ Add Extension" row when the model allows it —
// reusing renderEditableScalarInput()/renderEditableComplexValue() directly
// (with a custom handlerFn so top-level scalar edits land in
// _metaEditData, not _dataEditData — see metaEditFieldChange()). Complex
// (map/object/array) edits go through the same shared nested-mutation
// primitives the entity table uses (dataEditNestedFieldChange() etc.),
// gated by _dataEditActiveKind === 'meta' (set here before rendering).
function renderMetaTable(d, model, editable) {
  if (editable) _dataEditActiveKind = 'meta';
  var suppressed = { metaurl: 1 };
  if (_metaResourceIdField) suppressed[_metaResourceIdField] = 1;
  var metaPath = (_state.path || []).concat(['meta']);
  var keys = Object.keys(d).filter(function(k) { return !suppressed[k]; });
  // In edit mode, also surface every model-declared meta attribute with no
  // value set yet — same "show all model attributes" treatment
  // buildEntityPropsTableHtml() gives the entity's own Property table.
  if (editable && model) {
    var declaredMeta = topLevelAttrsMapFor(model, metaPath, d);
    Object.keys(declaredMeta).forEach(function(ak) {
      if (ak === '*' || suppressed[ak] || keys.indexOf(ak) !== -1) return;
      keys.push(ak);
    });
  }
  keys.sort();
  if (!keys.length && !editable) return '<div class="eg-row"><span class="eg-value" style="color:#aaa">No details available</span></div>';
  var capType = capitalize(_metaEntityType);
  var specLevel = (typeof SPEC_ATTRS !== 'undefined') ? SPEC_ATTRS.meta : null;
  var singular = (_metaEntityType || '').toLowerCase();
  var groups = groupPropsByCategory(keys, specLevel, singular, null, editable, capType + ' Metadata');

  function buildRow(k, banded) {
    var val = d[k];
    var display, valueCellClass = '';
    var attrType = getExplicitAttrType(model, metaPath, k, d);
    var editAttr = editable ? getAttr(model, metaPath, [k], d) : null;
    var effAttr = editable ? effectiveEditAttrAtPath([k], editAttr, val) : editAttr;
    var isComplexKind = effAttr
      && (effAttr.type === 'map' || effAttr.type === 'object' || effAttr.type === 'array');
    // defaultversionid can only be edited while defaultversionsticky is
    // true (that's what tells the server to honor an explicit default
    // instead of always tracking the newest version) — otherwise it stays
    // a read-only link, with a hint on how to unlock it.
    var isDefaultVerId = k === 'defaultversionid';
    var stickyOn = !!d.defaultversionsticky;
    var isEditableRow = editable && (!isDefaultVerId || stickyOn)
      && !isDataEditReadOnly(k, editAttr, singular, null)
      && (isComplexKind || (val !== null && typeof val !== 'object'));
    if (isEditableRow) {
      if (isComplexKind) {
        display = renderEditableComplexValue([k], val, effAttr, model, metaPath, d);
        valueCellClass = ' class="cell-tree"';
      } else if (isAnyTypeUnset(effAttr, val)) {
        var metaAnyAddId = 'metaanytype_' + k.replace(/[^a-zA-Z0-9]/g, '') + '_' + Math.random().toString(36).slice(2, 7);
        display = typePillHtml(effAttr)
          + renderComplexAddButtonHtml(effAttr, 'Set Value', metaAnyAddId, 'dataEditSetAnyType', esc(JSON.stringify([k])));
      } else if (isDefaultVerId) {
        // Dropdown of actual version IDs rather than a free-text input —
        // defaultversionid must reference a real version, so let the user
        // pick from the same versions list backing the Resource page's
        // "Version:" selector (_resVersionsList) instead of risking a typo.
        // Falls back to a plain text input if that list hasn't loaded yet.
        if (_resVersionsList && _resVersionsList.length) {
          var dvOpts = _resVersionsList.map(function(v) {
            var vid = itemNavKey(v);
            return '<option value="' + esc(vid) + '"' + (String(val) === vid ? ' selected' : '') + '>' + esc(vid) + '</option>';
          }).join('');
          display = typePillHtml(effAttr)
            + '<select class="xr-edit-input" data-ftype="string" onchange="metaEditFieldChange(\'' + k + '\', this)">'
            + dvOpts + '</select>';
        } else {
          display = typePillHtml(effAttr) + renderEditableScalarInput(k, val, effAttr || {type: attrType || 'string'}, 'metaEditFieldChange');
        }
      } else {
        display = typePillHtml(effAttr) + renderEditableScalarInput(k, val, effAttr || {type: attrType || 'string'}, 'metaEditFieldChange');
      }
    } else if (editable && isDefaultVerId && val != null) {
      // Sticky is off — keep the read-only link, plus a hint on how to
      // make this field editable.
      var dvid2 = String(val);
      var dvHref2 = pageHref(_state.path.slice(0, 4).concat(['versions', dvid2]), versionURLById(dvid2));
      var dvClick2 = 'navigateToVersionById(' + JSON.stringify(dvid2) + ')';
      display = '<a class="eg-value eg-mono eg-link" href="' + esc(dvHref2) + '" '
              + 'onclick="' + esc(guardedOnclick(dvClick2)) + '">' + esc(dvid2) + '</a>'
              + ' <span class="xr-hint">To edit, set Sticky to \u201ctrue\u201d.</span>';
    } else if (val == null) {
      display = '<span style="color:#999">null</span>';
    } else if (typeof val === 'boolean') {
      display = renderBoolBadge(val);
    } else if (typeof val === 'object') {
      // Pretty nested tree, matching buildPropsRowsHtml()'s view-mode
      // rendering of map/object/array attributes on the entity's own
      // Property table — was previously a raw JSON.stringify() dump here,
      // inconsistent with every other List-view table.
      var isEmptyM = Array.isArray(val) ? val.length === 0 : Object.keys(val).length === 0;
      display = isEmptyM
        ? '<span class="vt-empty">empty</span>'
        : renderValueTree(val, 0, model, metaPath, [k], d);
      valueCellClass = ' class="cell-tree"';
    } else if (k === 'defaultversionid') {
      // Link to the dedicated Version page for this version, matching Grid
      // view's "→ Visit" link for the same field (renderMetaContent()).
      var dvid = String(val);
      var dvHref = pageHref(_state.path.slice(0, 4).concat(['versions', dvid]), versionURLById(dvid));
      var dvClick = 'navigateToVersionById(' + JSON.stringify(dvid) + ')';
      display = '<a class="eg-value eg-mono eg-link" href="' + esc(dvHref) + '" '
              + 'onclick="' + esc(guardedOnclick(dvClick)) + '">' + esc(dvid) + '</a>';
    } else {
      var isMono = isMonoSpecAttr(k, specLevel, singular, metaPath)
        || (attrType !== null && attrType !== 'string');
      display = (attrType === 'timestamp')
        ? formatTimestampValue(String(val), isMono)
        : renderScalarValue(val, isMono);
    }
    var isDirty = editable && _metaEditSrc
      && JSON.stringify(val) !== JSON.stringify(_metaEditSrc[k]);
    var rowClasses = [];
    if (banded) rowClasses.push('xr-row-band');
    if (isDirty) rowClasses.push('xr-row-dirty');
    return '<tr' + (rowClasses.length ? ' class="' + rowClasses.join(' ') + '"' : '') + '><td style="font-weight:bold;color:#444;width:200px">' + esc(labelFor(k, specLevel, singular))
         + '</td><td' + valueCellClass + '>' + display + '</td></tr>';
  }

  var html = '<table class="xr-table xr-table-props"><thead><tr><th>' + esc(capType) + ' Details</th><th></th></tr></thead><tbody>';
  if (groups) {
    var domainFocusedM = !effectiveXregFocused() && !editable;
    // Same running-band fix as buildEntityPropsTableHtml() — Domain
    // view drops most category headers, so banding must keep counting
    // across group boundaries instead of resetting per group.
    var bandM = 0;
    groups.forEach(function(g) {
      html += buildPropsCatRowHtml(g, domainFocusedM);
      g.keys.forEach(function(k, i) { html += buildRow(k, (domainFocusedM ? bandM + i : i) % 2 === 1); });
      bandM += g.keys.length;
    });
  } else {
    keys.forEach(function(k, i) { html += buildRow(k, i % 2 === 1); });
  }
  // Editable "+ Add Extension" row — same wildcard-gated treatment as
  // buildEntityPropsTableHtml().
  if (editable && model) {
    var wildcardAttrMeta = getAttr(model, metaPath, ['*'], d);
    if (wildcardAttrMeta) {
      var metaExtAddId = 'metaExtAdd_' + Math.random().toString(36).slice(2, 7);
      html += '<tr><td style="font-weight:bold;color:#444;width:200px">+ Add Extension</td><td>'
        + '<input type="text" id="metaEditExtName" class="xr-edit-input" placeholder="name" style="width:140px">'
        + ' ' + renderComplexAddButtonHtml(wildcardAttrMeta, '+ Add', metaExtAddId, 'dataEditAddExtensionV2', esc(JSON.stringify('metaEditExtName')))
        + '</td></tr>';
    } else {
      html += '<tr><td style="font-weight:bold;color:#444;width:200px">+ Add Extension</td><td>'
        + '<span style="color:#999;font-style:italic">This model does not allow extension attributes here</span>'
        + '</td></tr>';
    }
  }
  html += '</tbody></table>';
  // Save/Undo action bar moved to the shared #dataEditorActionBarSlot — see
  // buildMetaActionBarHtml() and its slot-swap wiring in
  // renderSingleEntity()/switchDocTabReal() — so it's not duplicated far
  // below the page-level entity action bar and easy to miss.
  return html;
}


function renderMetaContent(d, model) {
  var html = '';
  var metaRendered = {};
  var metaPath = (_state.path || []).concat(['meta']);

  // Suppress the resource's own ID field — it's already in the page title context
  if (_metaResourceIdField) metaRendered[_metaResourceIdField] = 1;
  // Suppress internal/nav fields
  metaRendered.metaurl     = 1;
  // Mark defaultversionid/url as handled (rendered below after tech row)
  metaRendered.defaultversionid  = 1;
  metaRendered.defaultversionurl = 1;

  // Spec-defined attribute rows are laid out as their own two-column CSS
  // Grid (see .eg-attr-grid in style.css). Unknown extension attrs (below
  // the <hr class="eg-ext-sep">, added further down) get a separate grid
  // of their own further down, so the two sections' label columns can
  // size independently of each other.
  html += '<div class="eg-spec-rows eg-attr-grid">';

  // 1. Temporal
  if (d.createdat)  html += '<div class="eg-row eg-temporal"><span class="eg-label">Created:</span>'  + copyableMonospace(d.createdat)  + '</div>';
  if (d.modifiedat) html += '<div class="eg-row eg-temporal"><span class="eg-label">Modified:</span>' + copyableMonospace(d.modifiedat) + '</div>';
  metaRendered.createdat  = 1;
  metaRendered.modifiedat = 1;

  // 2. Tech row: epoch + self/shortself/xid as copy buttons. Always exactly
  // two top-level children (label + one value-wrapper span) so this row
  // works correctly as a 2-cell participant in the .eg-spec-rows CSS Grid
  // — extra top-level children would otherwise shift every following
  // row's grid-column placement.
  var techVal = '';
  if (d.epoch !== undefined) techVal += '<span class="eg-epoch">' + copyableMonospace(String(d.epoch)) + '</span>';
  if (d.self)      techVal += copyBtn('Self', toRealURL(d.self));
  if (d.shortself) techVal += copyBtn('ShortSelf', toRealURL(d.shortself));
  if (d.xid)       techVal += copyBtn('XID', d.xid);
  if (techVal) {
    html += '<div class="eg-row eg-technical">'
          + '<span class="eg-label">' + (d.epoch !== undefined ? 'Epoch:' : '') + '</span>'
          + '<span class="eg-value eg-tech-value">' + techVal + '</span></div>';
  }
  metaRendered.epoch     = 1;
  metaRendered.self      = 1;
  metaRendered.shortself = 1;
  metaRendered.xid       = 1;

  // 3. Default version ID with → View + URL ↗ buttons (after epoch)
  if (d.defaultversionid !== undefined) {
    var dvid = String(d.defaultversionid);
    var dvRow = copyableMonospace(dvid);
    dvRow += ' <a class="eg-link-btn eg-link-btn-nav" data-vid="' + esc(dvid) + '" '
           + 'href="' + esc(pageHref(_state.path.slice(0,4).concat(['versions', dvid]), versionURLById(dvid))) + '" '
           + 'onclick="if(navShouldDefault(event))return true;navigateToVersionById(this.dataset.vid);return false">→ Visit</a>';
    if (d.defaultversionurl) {
      dvRow += ' <a href="' + esc(d.defaultversionurl) + '" target="_blank" '
             + 'class="eg-link-btn" title="' + esc(d.defaultversionurl) + '">URL ↗</a>';
    }
    html += '<div class="eg-row"><span class="eg-label">Default Version ID:</span>'
          + '<span class="eg-value">' + dvRow + '</span></div>';
  }

  // 4. Labels
  if (d.labels && typeof d.labels === 'object') {
    var labelKeys = Object.keys(d.labels).sort();
    if (labelKeys.length) {
      var labelParts = labelKeys.map(function(k) {
        var kSpan = '<span class="eg-label-key">' + esc(k) + '</span>';
        var vSpan = '<span class="eg-label-val">' + esc(String(d.labels[k])) + '</span>';
        return '<span class="eg-label-pair">' + kSpan + vSpan + '</span>';
      });
      html += '<div class="eg-row eg-labels"><span class="eg-label">Labels:</span>'
            + '<span class="eg-label-list">' + labelParts.join('') + '</span></div>';
    }
  }
  metaRendered.labels = 1;

  // 5. defaultversionsticky, readonly
  if (d.defaultversionsticky !== undefined)
    html += row('Default Version Sticky', copyableMonospace(String(d.defaultversionsticky)));
  metaRendered.defaultversionsticky = 1;
  if (d.readonly !== undefined)
    html += row('Read Only', copyableMonospace(String(d.readonly)));
  metaRendered.readonly = 1;

  // 6. Remaining: spec attrs above <hr>, user extensions below
  var metaSpecLevel = (typeof SPEC_ATTRS !== 'undefined') ? SPEC_ATTRS.meta : {};
  var _metaSing = _metaResourceIdField ? _metaResourceIdField.replace(/id$/, '') : '';
  var remaining = Object.keys(d).filter(function(k) { return !metaRendered[k]; }).sort();
  var specKeys  = remaining.filter(function(k) { return  isSpecAttr(k, metaSpecLevel, _metaSing, null); });
  var extKeys   = remaining.filter(function(k) { return !isSpecAttr(k, metaSpecLevel, _metaSing, null); });

  function metaAttrRow(k) {
    var v = d[k];
    if (v !== null && typeof v === 'object') {
      var isEmpty = Array.isArray(v) ? v.length === 0 : Object.keys(v).length === 0;
      if (isEmpty) {
        html += row(labelFor(k, metaSpecLevel, _metaSing), '<span class="vt-empty">empty</span>');
      } else {
        html += '<div class="eg-ext-complex">'
              + '<div class="eg-ext-complex-key">' + esc(labelFor(k, metaSpecLevel, _metaSing)) + ':</div>'
              + '<div class="eg-ext-complex-body">' + renderValueTree(v, 0, model, metaPath, [k], d) + '</div>'
              + '</div>';
      }
    } else {
      // meta entity: use same logic as renderAttrRow with explicit-type-only monospace check
      var attrTypeMeta = getExplicitAttrType(model, metaPath, k, d);
      var isMono = isMonoSpecAttr(k, metaSpecLevel, _metaSing)
        || (attrTypeMeta !== null && attrTypeMeta !== 'string');
      html += row(labelFor(k, metaSpecLevel, _metaSing), renderScalarValue(v, isMono));
    }
  }
  specKeys.forEach(metaAttrRow);
  html += '</div>'; // close .eg-spec-rows
  if (extKeys.length) {
    html += '<hr class="eg-ext-sep">';
    // Own grid, independent of .eg-spec-rows' column width — extension
    // attr names can be arbitrarily long/short and unrelated to spec ones.
    html += '<div class="eg-ext-rows eg-attr-grid">';
    extKeys.forEach(metaAttrRow);
    html += '</div>'; // close .eg-ext-rows
  }
  return html;
}

function row(label, value, cls) {
  if (value === undefined || value === null || value === '') return '';
  return '<div class="eg-row' + (cls ? ' ' + cls : '') + '">'
    + (label ? '<span class="eg-label">' + esc(label) + ':</span>' : '')
    + '<span class="eg-value">' + value + '</span>'
    + '</div>';
}

// renderEntityGrid() (Grid view for the Registry root and Group entity
// pages) has been removed entirely — renderSingleEntity() (List view)
// already has full feature parity (same title, Group Types/Resources
// table, spec/extension attribute rows) plus more polish, so Grid added
// no unique value. See plan.md "Grid view removed".


// ---- JSON view -----------------------------------------------------------

// Details/Document toggle state — lets JSON view switch a Resource/Version
// entity page between the normal $details metadata JSON (default) and the
// entity's actual document content, mirroring List view's Document/Version
// Details tabs (see plan.md "JSON view Details/Document toggle"). Read-only
// for now — no edit-mode support for the Document side yet. Reset whenever
// navigation moves to a different server/path so switching entities always
// starts back on Details.
var _jsonDocMode    = '';   // '' = details (default), 'doc' = raw document
var _jsonDocModeKey = null; // "serverURL|path" — resets _jsonDocMode when navigating elsewhere

// Whether the current page is a Resource (depth 4) or Version (depth >= 6,
// path[4] === 'versions') entity — the only structural shape where a plain
// (non-$details) URL means something different from $details. The toggle
// is shown even when this resource type has no document defined
// (hasdocument=false) — Document mode then just renders an explanatory
// empty state instead of hiding the toggle entirely, so switching back and
// forth always works the same way regardless of resource type.
function jsonDocToggleApplies() {
  return needsDetails(_state.path);
}

// "Details | Document" toggle for the JSON view header — styled to match
// Edit mode's action-bar buttons (.editorBtn) rather than a bespoke pill,
// so JSON view doesn't look like a different app from Edit mode. The
// currently-active side is rendered disabled (same convention already used
// for Edit mode's Save/Undo buttons when there's nothing to do) instead of
// a separate "active" color scheme.
function buildJsonDocToggleHtml(active) {
  return '<span class="json-doc-toggle eg-doc-tabs">' +
    '<button class="eg-doc-tab json-doc-toggle-btn' + (active === 'details' ? ' active' : '')
      + '" onclick="jsonSetDocMode(\'details\')">Details</button>' +
    '<button class="eg-doc-tab json-doc-toggle-btn' + (active === 'doc' ? ' active' : '')
      + '" onclick="jsonSetDocMode(\'doc\')">Document</button>' +
  '</span>';
}

function jsonSetDocMode(mode) {
  if (_jsonDocMode === mode) return;
  _jsonDocMode = mode;
  if (_lastData) renderJSONView(_lastData);
}

// "Expand all" button markup — kept identical (same id, same position in
// the header) whether it's currently usable or not, so switching between
// Details/Document mode never shifts the other header buttons around.
// `disabled` is true whenever the content on screen isn't a collapsible
// JSON tree (e.g. Document mode showing plain text/binary, or still
// loading) — the button stays visible but inert in that case.
function buildJsonExpandAllBtnHtml(disabled) {
  return '<button type="button" class="eg-doc-tab json-exp-btn' + (disabled ? ' json-exp-btn-disabled' : '') + '"'
    + ' id="json-exp-btn" data-open="false"'
    + (disabled ? ' disabled' : '')
    + ' onclick="jsonToggleAll()" title="Expand all">&#9656; all</button>';
}

function renderJSONView(data) {
  renderJSONLeftPanel();
  if (_state.editMode && computeEnableEdit()) {
    renderJSONEditView(data);
    return;
  }

  var keyNow = normalizeURL(_state.serverURL || window.location.origin) + '|' + _state.path.join('/');
  if (_jsonDocModeKey !== keyNow) { _jsonDocMode = ''; _jsonDocModeKey = keyNow; }

  var showDocToggle = jsonDocToggleApplies();
  if (showDocToggle && _jsonDocMode === 'doc') {
    renderJSONViewDocumentMode(data);
    return;
  }

  var jsonHtml = addTwisties(syntaxHighlight(JSON.stringify(data, null, 2)));
  var serverURL = _state.serverURL || window.location.origin;
  el('main-view').innerHTML =
    '<div class="json-exp-wrap">' +
      '<span class="json-exp-server-url" title="' + esc(serverURL) + '">Server: '
        + esc(serverURL) + '</span>' +
      '<span class="json-exp-btn-group">' +
        (showDocToggle ? buildJsonDocToggleHtml('details') : '') +
        buildJsonExpandAllBtnHtml(false) +
      '</span>' +
    '</div>' +
    '<div id="json-output" tabindex="0">' + jsonHtml + '</div>';
}

// JSON view's "Document" mode — instead of the $details metadata JSON,
// shows the resource/version's actual document content: parsed and shown
// via the same twisty tree when it happens to be JSON, otherwise as plain
// read-only text, or a "binary, not previewable" message — mirroring List
// view's Document tab preview (loadDocumentPreview()), just rendered as
// the whole JSON view page instead of one tab's panel. Read-only: no Edit
// support for the Document side yet (see plan.md). When the resource type
// has no document defined at all (hasdocument=false), this renders an
// empty/explanatory state instead of attempting a fetch — a plain GET on
// such a resource just returns the same metadata as $details, which would
// be confusing to show under a "Document" label.
function renderJSONViewDocumentMode(entityData) {
  var serverURL = _state.serverURL || window.location.origin;
  el('main-view').innerHTML =
    '<div class="json-exp-wrap">' +
      '<span class="json-exp-server-url" title="' + esc(serverURL) + '">Server: '
        + esc(serverURL) + '</span>' +
      '<span class="json-exp-btn-group">' +
        buildJsonDocToggleHtml('doc') +
        buildJsonExpandAllBtnHtml(true) +
      '</span>' +
    '</div>' +
    '<div id="json-output" tabindex="0"><div class="eg-loading">Loading document\u2026</div></div>';

  var model = _modelCache[normalizeURL(_state.serverURL || window.location.origin)] || null;
  var depthJ = _state.path.length;
  var singular = (depthJ === 4) ? getSingularName(model, _state.path)
                                 : getSingularName(model, _state.path.slice(0, 4));
  var key = (singular || '').toLowerCase();
  var box = el('json-output');
  if (!box) return;

  if (!resourceHasDocument(model, _state.path)) {
    box.innerHTML = '<div class="eg-doc-binary-msg">No document defined for this resource type.</div>';
    return;
  }

  function showJSONTree(obj) {
    box.innerHTML = addTwisties(syntaxHighlight(JSON.stringify(obj, null, 2)));
    var expBtn = document.getElementById('json-exp-btn');
    // The button starts genuinely HTML-disabled (see buildJsonExpandAllBtnHtml())
    // since Document mode doesn't know yet whether the fetched content will
    // turn out to be JSON (twisty tree, expandable) or plain text/binary
    // (nothing to expand). Once it turns out to be JSON, clear the actual
    // `disabled` attribute too, not just its CSS class — a disabled button
    // never fires click events at all, so leaving the attribute set left
    // "All" visually enabled but permanently inert (see plan.md).
    if (expBtn) {
      expBtn.classList.remove('json-exp-btn-disabled');
      expBtn.disabled = false;
      expBtn.dataset.open = 'false';
    }
  }
  function actionsHtml(url) {
    return url ? '<div class="eg-doc-preview-actions"><a href="' + esc(url)
      + '" target="_blank" rel="noopener" class="eg-link-btn">Open in new tab</a></div>' : '';
  }
  function showText(text, url) {
    box.innerHTML = '<textarea class="eg-doc-textarea" readonly>' + esc(text) + '</textarea>' + actionsHtml(url);
  }
  function showBinary(url) {
    box.innerHTML = '<div class="eg-doc-binary-msg">Binary file \u2014 preview not available.</div>' + actionsHtml(url);
  }
  function showError(msg, url) {
    box.innerHTML = '<div class="eg-doc-error-msg">' + esc(msg) + '</div>' + actionsHtml(url);
  }
  function renderTextOrJSON(text, url) {
    try { showJSONTree(JSON.parse(text)); }
    catch (e) { showText(text, url); }
  }
  function fetchAndRender(url) {
    fetch(url)
      .then(function(r) { if (!r.ok) throw new Error('HTTP ' + r.status); return r.arrayBuffer(); })
      .then(function(buf) { isBinaryContent(buf) ? showBinary(url) : renderTextOrJSON(decodeUTF8Bytes(buf), url); })
      .catch(function(e) { showError('Could not load document: ' + ((e && e.message) || String(e)), url); });
  }

  if (!key || !entityData) { box.innerHTML = '<div class="eg-doc-binary-msg">Document not available.</div>'; return; }

  if (entityData[key + 'url']) { fetchAndRender(entityData[key + 'url']); return; }

  if (entityData[key + 'base64']) {
    try {
      var binary = atob(entityData[key + 'base64']);
      var bytes = new Uint8Array(binary.length);
      for (var i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
      var blobUrl = URL.createObjectURL(new Blob([bytes], { type: entityData.contenttype || 'application/octet-stream' }));
      if (isBinaryContent(bytes.buffer)) showBinary(blobUrl); else renderTextOrJSON(decodeUTF8Bytes(bytes.buffer), blobUrl);
    } catch (e) { showError('Error decoding document: ' + ((e && e.message) || String(e))); }
    return;
  }

  if (entityData[key] !== undefined && entityData[key] !== null) { showJSONTree(entityData[key]); return; }

  if (entityData.self) { fetchAndRender(entityData.self.replace(/\$details$/, '')); return; }

  box.innerHTML = '<div class="eg-doc-binary-msg">Document not available.</div>';
}

// ---- JSON view raw editing ("postman-style") ------------------------------
//
// When Edit is turned on while in JSON view, the read-only twisty tree above
// is replaced by a plain, fully-editable textarea containing the entity/
// collection/document's pretty-printed JSON. Save (PUT) and Save (PATCH)
// both send whatever is currently in the textarea verbatim — no auto-diff
// against the original, unlike List view's PATCH — so the user has full
// manual control (true "postman"-style editing). See plan.md "Raw JSON
// edit ('postman-style') for JSON view" for the full design discussion.
//
// This is an INDEPENDENT edit buffer from List view's own per-section
// working copies (_dataEditData/_modelData/_capData) — it always starts
// fresh from the latest server-fetched data every time Edit is turned on
// in JSON view (see resetJsonEditBuffer()/the null-sentinel pattern below),
// and switching List<->JSON view mid-edit warns about losing whichever
// view's changes are unsaved (see the _jsonEditDirty guards added to
// pushState()/setDataView()).
var _jsonEditOrigText  = null;  // null = needs (re)initializing from `data`
var _jsonEditDraftText = null;  // current (possibly unsaved) textarea text
var _jsonEditDirty     = false;
var _jsonEditKind      = '';    // 'entity' | 'collection' | 'capabilities' | 'modelsource'
var _jsonEditURL       = '';    // target endpoint for Save/Delete

function resetJsonEditBuffer() {
  _jsonEditOrigText  = null;
  _jsonEditDraftText = null;
  _jsonEditDirty     = false;
  _jsonEditKind      = '';
  _jsonEditURL       = '';
}

// Resolves {kind, url} for the current page — mirrors the URL-construction
// logic already used by saveDataEntity() (entity $details suffixing) and
// buildAPIURL() (modelsource/capabilities fixed endpoints).
function jsonEditTarget() {
  if (_state.section === 'modelsource') return {kind: 'modelsource', url: serverBase() + '/modelsource'};
  if (_state.section === 'capabilities') return {kind: 'capabilities', url: serverBase() + '/capabilities'};
  var coll = isCollection(_state.path);
  if (coll) return {kind: 'collection', url: buildBaseURL()};
  var url = buildBaseURL();
  var depthS = _state.path.length;
  if (depthS === 4 || depthS >= 6) {
    var svBaseS = (_state.serverURL || window.location.origin).replace(/\/$/, '');
    var modelS = _modelCache[normalizeURL(svBaseS)] || null;
    if (resourceHasDocument(modelS, _state.path)) url = url.replace(/(\?|$)/, '$details$1');
  }
  return {kind: 'entity', url: url};
}

function renderJSONEditView(data) {
  if (_jsonEditOrigText === null) {
    // Fresh entry into JSON edit mode for this page — snapshot the current
    // server data as both the pristine original and the initial draft.
    // jsonEditTarget()'s $details-suffix decision (for Resource/Version
    // entities) depends on the model already being cached (resourceHasDocument()).
    // On a normal in-app navigation the model is already warm by the time
    // edit mode is reachable, but a fresh/bookmarked direct URL load straight
    // into edit=1&dview=json can race ahead of the model fetch — silently
    // omitting $details and causing every subsequent Save to PUT/PATCH the
    // bare (non-$details) endpoint, which replaces the resource's raw
    // document content wholesale instead of updating its attributes. Route
    // through ensureModelCached() (a same-tick callback if already cached,
    // otherwise a real fetch-then-callback) to close that race before
    // computing/locking in _jsonEditURL.
    var svBaseJ = (_state.serverURL || window.location.origin).replace(/\/$/, '');
    ensureModelCached(svBaseJ, function() {
      if (_jsonEditOrigText !== null) return; // already initialized meanwhile
      var target = jsonEditTarget();
      _jsonEditKind = target.kind;
      _jsonEditURL  = target.url;
      _jsonEditOrigText  = JSON.stringify(data, null, 2);
      _jsonEditDraftText = _jsonEditOrigText;
      _jsonEditDirty     = false;
      renderJSONEditView(data);
    });
    return;
  }
  var serverURL = _state.serverURL || window.location.origin;
  var isEntity = (_jsonEditKind === 'entity');
  var deleteDisabled = (isEntity && _state.path.length === 0)
    ? ' disabled title="Deleting the registry itself is not supported here"' : '';
  // Action bar lives in the same sticky header row as "Server: ..." (see
  // .json-exp-wrap, also used by the read-only JSON view) rather than
  // below the textarea — with a server error banner shown, the old
  // bottom-of-page action bar could scroll off screen entirely. Sticky-
  // positioning it at the top (like the read-only view's "expand all"
  // button) keeps Save/Undo/Format/Delete always reachable.
  var html = '<div class="json-exp-wrap json-edit-header">' +
      '<span class="json-exp-server-url" title="' + esc(serverURL) + '">Server: '
        + esc(serverURL) + '</span>' +
      '<div class="editorActionBar" id="jsonEditActionBar">' +
        '<button class="editorBtn" id="jsonEditFormatBtn" onclick="jsonEditFormat()"'
          + '>Format</button>' +
        ' <button class="editorBtn" id="jsonEditPutBtn" onclick="jsonEditSave(\'PUT\')"'
          + (_jsonEditDirty ? '' : ' disabled') + '>Save (PUT)</button>' +
        ' <button class="editorBtn" id="jsonEditPatchBtn" onclick="jsonEditSave(\'PATCH\')"'
          + (_jsonEditDirty ? '' : ' disabled') + '>Save (PATCH)</button>' +
        ' <button class="editorBtn" id="jsonEditUndoBtn" onclick="jsonEditUndo()"'
          + (_jsonEditDirty ? '' : ' disabled') + '>Undo</button>' +
        (isEntity ? ' <button class="editorBtn editorBtnDanger" onclick="jsonEditDelete()"' + deleteDisabled + '>Delete</button>' : '') +
      '</div>' +
    '</div>' +
    '<div id="jsonEditError" class="error-banner" style="display:none"></div>' +
    '<div id="jsonEditInvalid" class="error-banner" style="display:none"></div>' +
    '<textarea id="jsonEditTextarea" class="json-edit-textarea" spellcheck="false"' +
      ' oninput="jsonEditInputChanged()">' + esc(_jsonEditDraftText) + '</textarea>';
  el('main-view').innerHTML = html;
  sizeJsonEditTextarea();
}

// Stretches the JSON-edit textarea down to the bottom of the viewport
// (minus a little breathing room), same idiom as sizeDocTextarea() for
// the Document tab's read-only preview — otherwise the textarea's CSS
// min-height leaves it stopping well short of the window on tall screens.
function sizeJsonEditTextarea() {
  var ta = el('jsonEditTextarea');
  if (!ta || ta.offsetParent === null) return; // not rendered / panel hidden
  var available = window.innerHeight - ta.getBoundingClientRect().top - 16; // bottom breathing room
  ta.style.height = Math.max(200, Math.floor(available)) + 'px';
}

// Hides both of JSON edit view's error banners (the parse-error one from
// Format/Save, and the server-error one from a failed Save/Delete) —
// called at the start of every action-bar button handler so clicking any
// of them (Format/Save/Undo/Delete) clears whatever error was left over
// from a previous action, instead of leaving a stale message on screen
// while a new action is attempted.
function clearJsonEditErrors() {
  hideErrorBanner(el('jsonEditError'));
  hideErrorBanner(el('jsonEditInvalid'));
}

// oninput handler for the raw JSON textarea. Deliberately does NOT
// validate JSON on every keystroke — flagging "invalid JSON" errors while
// someone is mid-edit (e.g. right after typing an opening brace) is bad
// UX. Instead this only tracks whether the buffer is dirty (differs from
// the pristine original) to toggle Save/Undo; syntax is only checked when
// the user explicitly clicks Format (see jsonEditFormat()) or Save (see
// jsonEditSave(), which still guards against submitting invalid JSON).
function jsonEditInputChanged() {
  var ta = el('jsonEditTextarea');
  if (!ta) return;
  _jsonEditDraftText = ta.value;
  _jsonEditDirty = _jsonEditDraftText !== _jsonEditOrigText;
  var putBtn = el('jsonEditPutBtn'), patchBtn = el('jsonEditPatchBtn'), undoBtn = el('jsonEditUndoBtn');
  if (putBtn)   putBtn.disabled   = !_jsonEditDirty;
  if (patchBtn) patchBtn.disabled = !_jsonEditDirty;
  if (undoBtn)  undoBtn.disabled  = !_jsonEditDirty;
}

// "Format" button — the only place JSON syntax is actually validated
// before Save. Parses the current textarea content and, if valid,
// replaces it with a pretty-printed (2-space indent) version; if invalid,
// shows the parse error in the same banner Save's own validation uses,
// without touching the textarea content.
function jsonEditFormat() {
  var ta = el('jsonEditTextarea');
  if (!ta) return;
  clearJsonEditErrors();
  var invalidDiv = el('jsonEditInvalid');
  try {
    var parsed = JSON.parse(ta.value);
    var pretty = JSON.stringify(parsed, null, 2);
    ta.value = pretty;
    _jsonEditDraftText = pretty;
    _jsonEditDirty = pretty !== _jsonEditOrigText;
  } catch (e) {
    if (invalidDiv) { showErrorBanner(invalidDiv, 'Invalid JSON: ' + e.message); }
    return;
  }
  var putBtn = el('jsonEditPutBtn'), patchBtn = el('jsonEditPatchBtn'), undoBtn = el('jsonEditUndoBtn');
  if (putBtn)   putBtn.disabled   = !_jsonEditDirty;
  if (patchBtn) patchBtn.disabled = !_jsonEditDirty;
  if (undoBtn)  undoBtn.disabled  = !_jsonEditDirty;
}

function jsonEditUndo() {
  clearJsonEditErrors();
  _jsonEditDraftText = _jsonEditOrigText;
  _jsonEditDirty = false;
  var ta = el('jsonEditTextarea');
  if (ta) ta.value = _jsonEditDraftText;
  var putBtn = el('jsonEditPutBtn'), patchBtn = el('jsonEditPatchBtn'), undoBtn = el('jsonEditUndoBtn');
  if (putBtn)   putBtn.disabled   = true;
  if (patchBtn) patchBtn.disabled = true;
  if (undoBtn)  undoBtn.disabled  = true;
}

// Saves the raw textarea content verbatim (no auto-diff, per the "fully
// manual/postman-style" design decision) via PUT or PATCH to the target
// endpoint resolved when edit mode was entered. Mirrors saveDataEntity()'s
// overlay/error-banner UX.
function jsonEditSave(verb, cb) {
  clearJsonEditErrors();
  var errDiv = el('jsonEditError');
  var body;
  try { body = JSON.parse(_jsonEditDraftText); } catch (e) {
    if (errDiv) { showErrorBanner(errDiv, 'Invalid JSON: ' + e.message); }
    return;
  }

  var overlay = document.createElement('div'); overlay.className = 'savingOverlay';
  var box = document.createElement('div'); box.className = 'savingBox';
  var spinner = document.createElement('div'); spinner.className = 'savingSpinner';
  var msg = document.createElement('div'); msg.textContent = 'Saving\u2026';
  box.appendChild(spinner); box.appendChild(msg); overlay.appendChild(box);
  document.body.appendChild(overlay);
  function removeOverlay() { if (overlay.parentNode) overlay.parentNode.removeChild(overlay); }

  fetch(_jsonEditURL, {
    method: verb,
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(body, null, 2)
  }).then(function(resp) {
    return resp.text().then(function(text) {
      removeOverlay();
      if (resp.ok) {
        // The PUT/PATCH response body only ever reflects the bare entity —
        // it never honors this page's inline=/filter=/etc query params the
        // way a GET via buildAPIURL() does (same URL a plain browser
        // refresh would use). Re-fetch through that same URL (with the same
        // $details fallback refresh() itself uses for document-backed
        // Resource/Version entities — a plain GET on those returns the raw
        // document body, not the entity JSON, which would silently drop
        // fields like filebase64) so any inlined sub-collections the user
        // had selected (e.g. "file", "meta", "versions") are still present
        // after Save, instead of disappearing. Falls back to the raw
        // response body (or the just-submitted request body, worst case) if
        // this re-fetch itself fails for some reason — better to show
        // *something* than nothing after a successful save.
        var fallback = null;
        try { fallback = JSON.parse(text); } catch (e) { /* fall back further below */ }
        fetchWithDetailsFallback(buildAPIURL(), needsDetails(_state.path)).catch(function() {
          return fallback !== null ? fallback : body;
        }).then(function(updatedData) {
          finishJsonEditSave(updatedData, cb);
        });
      } else {
        if (errDiv) { showErrorBanner(errDiv, 'Error (' + resp.status + '):\n' + text); }
      }
    });
  }).catch(function(err) {
    removeOverlay();
    if (errDiv) { showErrorBanner(errDiv, 'Network error: ' + err.message); }
  });
}

// Shared tail of jsonEditSave()'s success path — updates every buffer/cache
// that needs to reflect the freshly (re-)fetched post-save data, then
// either hands off to the caller's continuation (cb, e.g. navigating away)
// or re-renders in place.
function finishJsonEditSave(updatedData, cb) {
  var newText = JSON.stringify(updatedData, null, 2);
  _jsonEditOrigText  = newText;
  _jsonEditDraftText = newText;
  _jsonEditDirty     = false;
  // Keep the underlying section's own "source of truth" cache in sync
  // so switching back to List view afterward reflects the saved
  // state rather than stale pre-save data.
  if (_jsonEditKind === 'modelsource') {
    _modelSrc = deepClone(updatedData); _modelData = deepClone(_modelSrc); _modelDirty = false;
  } else if (_jsonEditKind === 'capabilities') {
    _capSrc = deepClone(updatedData); _capData = deepClone(_capSrc); _capDirty = false;
  } else if (_jsonEditKind === 'entity') {
    _dataEditSrc = deepClone(updatedData); _dataEditData = deepClone(_dataEditSrc); _dataDirty = false;
  }
  _lastData = updatedData;
  if (cb) { cb(); } else { renderJSONEditView(updatedData); }
}

// Delete — single entities only (not collections/capabilities/modelsource),
// same scope as List view. Delegates directly to the existing
// deleteDataEntity() confirm+DELETE+navigate-to-parent flow. Clears the
// dirty flag (not the whole buffer — deleteDataEntity()'s own confirm()
// dialog is synchronous, so we don't yet know if the user will actually go
// through with it) just so a successful delete's own pushState() call
// below doesn't ALSO trigger the "unsaved JSON edit" leave-guard on top of
// the delete confirmation the user already answered.
function jsonEditDelete() {
  clearJsonEditErrors();
  _jsonEditDirty = false;
  deleteDataEntity();
}

// Process syntaxHighlighted JSON HTML to add twisty expand/collapse spans.
// Mirrors the old Go RegHTMLify logic: every line gets a fixed-width gutter
// column (.jt-spc for non-openers, .jt toggle for openers).  The newline at
// the end of each opener line is placed INSIDE the block span so when collapsed
// the closing bracket appears on the same line as the opener.
// All blocks start collapsed.
function addTwisties(html) {
  var SPC  = '<span class="jt-spc"> </span>';
  var lines = html.split('\n');
  var count = 0;
  var depth = 0;
  var out   = [];

  for (var i = 0; i < lines.length; i++) {
    var line    = lines[i];
    var text    = line.replace(/<[^>]+>/g, ''); // strip HTML for structural analysis
    var ns      = 0;
    while (ns < text.length && text[ns] === ' ') ns++;
    var trimmed = text.substring(ns);
    if (!trimmed) { out.push(SPC + line); continue; }

    var first    = trimmed[0];
    var last     = trimmed[trimmed.length - 1];
    var isOpener = (last === '{' || last === '[');
    var isCloser = (first === '}' || first === ']');

    if (isCloser) {
      if (depth > 0) {
        // spc + ns spaces are inside the block (hidden when collapsed).
        // </span> closes the block.
        // trimmed (e.g. "},") is outside the block — appears on same line
        // as the opener when collapsed since the opener's \n is inside the block.
        out.push(SPC + ' '.repeat(ns) + '</span>' + trimmed);
        depth--;
      } else {
        out.push(SPC + line);
      }
    } else if (isOpener && ns > 0) {
      count++;
      depth++;
      var n      = count;
      var indent = ns > 1 ? ' '.repeat(ns - 1) : '';
      // .jt's own box stays exactly 1ch at the CONTAINER's font-size (13px)
      // so it precisely replaces the one native indent space it sits in
      // place of — alignment with non-opener siblings depends on this.
      // The glyph itself is wrapped in .jt-glyph (larger font, small right
      // margin for breathing room) which can overflow .jt's box (see
      // `overflow: visible` in CSS) purely visually, without affecting the
      // outer box's contribution to the line's layout width.
      // .jt is user-select:none (mirrors the old ui.go .exp class) so the
      // glyph itself is never included in a copy/paste. .jt-copysp is a
      // zero-width but still-selectable span holding a real space
      // character, so copied/pasted text still has the exact same
      // indentation as plain JSON.stringify output (mirrors the old
      // ui.go RegHTMLify ".hide" trick).
      var tw = '<span class="jt" id="jt' + n + '" onclick="jsonToggle(' +
        n + ')"><span class="jt-glyph">&#9656;</span></span>' +
        '<span class="jt-copysp"> </span>';
      var dots   = '<span class="jd" id="jd' + n + '" onclick="jsonToggle(' + n + ')">&#8230;</span>';
      // jt-spc gutter + (ns-1) indent spaces + toggle + content + dots.
      // <jb> is appended at END of line so the \n from join is INSIDE the block.
      out.push(SPC + indent + tw + line.substring(ns) + dots + '<span class="jb" id="jb' + n + '" style="display:none">');
    } else {
      out.push(SPC + line);
    }
  }
  return out.join('\n');
}

// Toggle one expandable block (n = numeric id).
function jsonToggle(n) {
  var tw    = document.getElementById('jt' + n);
  var glyph = tw ? tw.querySelector('.jt-glyph') : null;
  var blk   = document.getElementById('jb' + n);
  var dots  = document.getElementById('jd' + n);
  if (!blk) return;
  var open = blk.style.display === 'none';
  blk.style.display        = open ? '' : 'none';
  if (dots)  dots.style.display = open ? 'none' : '';
  if (glyph) glyph.innerHTML    = open ? '&#9662;' : '&#9656;';
}

// Toggle all blocks expand/collapse.
function jsonToggleAll() {
  var btn    = document.getElementById('json-exp-btn');
  if (btn && btn.classList.contains('json-exp-btn-disabled')) return;
  var expand = btn ? btn.dataset.open !== 'true' : true;
  for (var i = 1; ; i++) {
    var blk  = document.getElementById('jb' + i);
    if (!blk) break;
    var tw    = document.getElementById('jt' + i);
    var glyph = tw ? tw.querySelector('.jt-glyph') : null;
    var dots  = document.getElementById('jd' + i);
    blk.style.display        = expand ? '' : 'none';
    if (dots)  dots.style.display = expand ? 'none' : '';
    if (glyph) glyph.innerHTML    = expand ? '&#9662;' : '&#9656;';
  }
  if (btn) {
    btn.dataset.open = expand ? 'true' : 'false';
    btn.innerHTML    = (expand ? '&#9662;' : '&#9656;') + ' all';
    btn.title        = expand ? 'Collapse all' : 'Expand all';
  }
}

function renderJSONLeftPanel(filtersOnly) {
  var inner = el('left-panel-inner');
  if (!inner) return;
  var html = '';

  var svBase2  = (_state.serverURL || window.location.origin).replace(/\/$/, '');
  var normUrl2 = normalizeURL(svBase2);
  var model2   = _modelCache[normUrl2] || null;
  var cap2     = _capCache.hasOwnProperty(normUrl2) ? _capCache[normUrl2] : undefined;

  // Trigger model fetch if not yet cached
  if (!_modelCache.hasOwnProperty(normUrl2)) {
    ensureModelCached(svBase2, function() {
      if (_state.dataView === 'json' || _state.view === 'json' || _filtersPanelOpen) renderJSONLeftPanel(filtersOnly);
    });
  }

  // Trigger capability fetch if not yet cached; re-render when ready
  if (!_capCache.hasOwnProperty(normUrl2)) {
    ensureCapCached(svBase2, function() {
      if (_state.dataView === 'json' || _state.view === 'json' || _filtersPanelOpen) renderJSONLeftPanel(filtersOnly);
    });
    // Don't render anything until we know what the server supports
    inner.innerHTML = '';
    return;
  }

  var flags = (cap2 && cap2.flags) || [];
  var hasF  = function(f) { return flags.indexOf(f) !== -1; };
  var avail2 = cap2 && cap2.available;

  // Registry Endpoints navigation — at root depth, show links to model/capabilities/etc.
  // Show in both section views and normal data view so you can switch sections easily.
  // Filters (below) get their own leading divider-with-Apply-button combo
  // instead of a plain divider, so skip the plain one here to avoid a
  // doubled-up line. Skipped entirely in filtersOnly mode (Grid/List's
  // slimmed-down panel — see plan.md "Filter support in Grid/List views").
  var filterHasApplyDivider = _state.section === 'data' && hasF('filter');
  if (!filtersOnly && _state.path.length === 0) {
    var hasModel   = avail2 && avail2.model;
    var hasMSource  = avail2 && avail2.modelsource;
    var hasCap     = avail2 && avail2.capabilities;
    var hasCapOff  = avail2 && avail2.capabilitiesoffered;
    var hasXReg    = avail2 && avail2[sectionCapKey('xregistry')];
    var hasExport  = avail2 && avail2['export'];
    if (hasModel || hasMSource || hasCap || hasCapOff || hasXReg || hasExport) {
      html += '<div class="lp-section"><div class="lp-title">Viewing:</div>';
      // "Registry Data (Export)" is listed first so it's always one click
      // away to get back from a Model/Capabilities section view, and so
      // the active Export state is highlighted the same way Model/Source
      // etc are, instead of the old checkmark-only indicator.
      var dataActive   = _state.section === 'data' && !_state.useExport;
      var exportActive = _state.section === 'data' && _state.useExport;
      var dataHref   = pageHref([], _state.rootApiURL || '', {section: 'data', useExport: false});
      var exportHref = pageHref([], _state.rootApiURL || '', {section: 'data', useExport: true});
      html += '<div class="lp-nav-row">'
        + '<a class="lp-nav-item lp-nav-inline'
        + (dataActive ? ' lp-nav-active' : '') + '" href="' + esc(dataHref)
        + '" onclick="' + esc(guardedOnclick('pushState('
        + '{section:\'data\',path:[],useExport:false})'))
        + '">Data</a>';
      if (hasExport) {
        html += ' <span class="lp-nav-sub">(<a class="lp-nav-item '
          + 'lp-nav-inline' + (exportActive ? ' lp-nav-active' : '')
          + '" href="' + esc(exportHref) + '" onclick="'
          + esc(guardedOnclick('pushState({section:\'data\',path:[],'
          + 'useExport:true})')) + '">Export</a>)</span>';
      }
      html += '</div>';
      // "Model (Source)" and "Capabilities (Offered)" share one line each
      // (matches the old ui.go layout) instead of 4 stacked rows, to save
      // vertical space — the sub-link only appears when that endpoint is
      // actually available. .xregistry has no sub-link of its own, so it
      // gets its own single-item row via the same helper (hasSub: false).
      html += lpNavPairRow('Model', 'model', hasModel,
        'Source', 'modelsource', hasMSource);
      html += lpNavPairRow('Capabilities', 'capabilities', hasCap,
        'Offered', 'capabilitiesoffered', hasCapOff);
      html += lpNavPairRow('.xregistry', 'xregistry', hasXReg,
        '', '', false);
      html += '</div>';
      if (!filterHasApplyDivider) html += '<hr class="lp-divider">';
    }
  }

  var hasOpts = false; // true when there's at least one filter/option/inline to apply
  // In filtersOnly mode (Grid/List's slimmed-down panel), the Apply button
  // must only touch filters — applyJSONOptions() also collects sort/inline/
  // doc/binary/collections from DOM elements that don't exist in this mode,
  // which would silently reset any JSON-view-only state. applyGridFilters()
  // calls the shared applyFilters() helper directly, nothing else.
  var applyFn = filtersOnly ? 'applyGridFilters' : 'applyJSONOptions';

  if (_state.section === 'data') {
    // Computed up front (rather than where the Sort section itself is
    // built, further below) since the Filters section right below needs to
    // know whether Sort will follow it in filtersOnly mode, to decide
    // whether a divider belongs between them.
    var sortAvailable = hasF('sort') && isCollection(_state.path);

    // Filters — only if server supports 'filter'. A divider-line-with-
    // Apply-button combo is shown right above the Filters heading (in
    // addition to the full-width Apply button at the very bottom of the
    // panel) so it's reachable without scrolling past a long filter list.
    // The section body itself is collapsible (twisty + a "(N)" count of
    // currently-defined filter expressions, visible even while
    // collapsed) to save vertical space — collapse state is not tied to
    // the Apply-button divider above, which always shows regardless.
    if (hasF('filter')) {
      hasOpts = true;
      ensureFbDraft();
      html += '<div class="lp-divider-apply">'
        + '<span class="lp-divider-line"></span>'
        + '<button class="lp-apply lp-apply-top" onclick="' + applyFn + '()">'
        + 'Apply</button>'
        + '<span class="lp-divider-line"></span></div>'
        + '<div class="lp-section" id="lp-filter-section">'
        + fbFiltersTitleHTML(fbFilterCount(_fbDraft.groups), filtersOnly)
        + ((_filtersCollapsed && !filtersOnly) ? '' : '<div class="lp-filter-indent">'
            + buildFilterSectionInner(model2) + '</div>')
        + '</div>'
        + ((filtersOnly && !sortAvailable) ? '' : '<hr class="lp-divider">');
    }

    // Sort — only if server supports 'sort' and the current path points at
    // a collection (Groups/Resources/Versions); sorting a single entity
    // isn't meaningful, per the spec's sort_noncollection error. Now shown
    // in both the full JSON panel and Grid/List's filtersOnly panel (List
    // view honors it — see renderTableView()'s preserveOrder — until a
    // column header is clicked, which clears it; see sortBy()). Kept to
    // one line ("Sort:" + attribute dropdown) until an attribute is
    // actually chosen, at which point the map-key/order/clear rows appear
    // below — no twisty needed since there's only ever one control (unlike
    // Filters, which can grow to many expressions).
    if (sortAvailable) {
      hasOpts = true;
      ensureSortDraft();
      html += '<div class="lp-section" id="lp-sort-section">'
        + buildSortSectionInner(model2)
        + '</div><hr class="lp-divider">';
    }

    // Options/Inlines stay JSON-view-only for now — skipped entirely in
    // filtersOnly mode (Grid/List). See plan.md "Filter support in
    // Grid/List views" — not yet decided whether these make sense there.
    if (!filtersOnly) {
    // Options — only show individual options whose flag is enabled
    var optHtml = '';
    if (hasF('doc'))         optHtml += lpCheck('lp-doc', 'doc view',    _state.docView);
    if (hasF('binary'))      optHtml += lpCheck('lp-bin', 'binary',      _state.binary);
    if (hasF('collections') && collectionsEligible(_state.path)) optHtml += lpCheck('lp-col', 'collections', _state.collections);
    if (optHtml) {
      hasOpts = true;
      html += '<div class="lp-section"><div class="lp-title">Options</div>'
        + optHtml + '</div><hr class="lp-divider">';
    }

    // Inlines — only if server supports 'inline'
    if (hasF('inline')) {
      var inlineOpts = buildInlineOptions(model2, _state.path);
      var hasReal = inlineOpts.some(function(o) { return !o.sep; });
      if (hasReal) {
        hasOpts = true;
        html += '<div class="lp-section"><div class="lp-title">Inlines</div>';
        var rowIdx = 0;
        inlineOpts.forEach(function(opt) {
          if (opt.sep) { html += '<div class="lp-sep-line"></div>'; return; }
          var checked   = _state.inlines.includes(opt.value)          ? ' checked' : '';
          var dschecked = opt.dotStar && _state.inlines.includes(opt.value + '.*') ? ' checked' : '';
          var rowCls = 'lp-item' + (rowIdx % 2 === 0 ? ' lp-even' : '');
          var dotStarHtml = opt.dotStar
            ? '<span class="lp-dotstar">'
                + '<input type="checkbox" class="lp-inline-cb" value="' + esc(opt.value + '.*') + '"' + dschecked + ' onchange="updateApplyButtonState()">'
                + '<span class="lp-dotstar-label">.*</span>'
                + '</span>'
            : '<span class="lp-dotstar"></span>';
          html += '<div class="' + rowCls + '">'
            + '<input type="checkbox" class="lp-inline-cb" value="' + esc(opt.value) + '"' + checked + ' onchange="updateApplyButtonState()">'
            + '<span class="lp-inline-label">' + esc(opt.label) + '</span>'
            + dotStarHtml
            + '</div>';
          rowIdx++;
        });
        html += '</div><hr class="lp-divider">';
      }
    }
    } // !filtersOnly
  }

  if (!html)    html = '<div class="lp-no-opts">No options</div>';
  // In filtersOnly mode (Grid/List) with only Filters shown, its own
  // leading divider-with-Apply combo (right above the "Filters" heading)
  // is already sufficient — skip this trailing one to avoid showing two
  // identical Apply buttons for a single section. Once Sort is also shown
  // (sortAvailable), there are now two sections, so a trailing Apply
  // (reachable without scrolling back up to Filters) makes sense again —
  // same as the full JSON panel already does unconditionally.
  if (hasOpts && (!filtersOnly || sortAvailable)) {
    // Reuse the divider-apply combo (line + centered Apply + line) instead
    // of a separate full-width button; strip the trailing plain divider
    // left by the last section so we don't get doubled-up lines.
    var trailingHr = '<hr class="lp-divider">';
    if (html.slice(-trailingHr.length) === trailingHr) {
      html = html.slice(0, -trailingHr.length);
    }
    html += '<div class="lp-divider-apply">'
      + '<span class="lp-divider-line"></span>'
      + '<button class="lp-apply lp-apply-top" onclick="' + applyFn + '()">'
      + 'Apply</button>'
      + '<span class="lp-divider-line"></span></div>';
  }
  inner.innerHTML = html;
  updateApplyButtonState();
}

// Shared by JSON view's Apply button (applyJSONOptions) and Grid/List's
// filtersOnly panel Apply button — the one deliberate spot a filter
// expression is actually constructed client-side. See applyFilters()/
// plan.md "Filter support in Grid/List views". Also collects the Sort
// picker's value (now shown in this panel too — see renderJSONLeftPanel())
// so List view's Apply button applies both together, same as JSON view's.
function applyGridFilters() {
  var patch = applyFilters();
  // Sync the draft to match what was actually just applied — otherwise
  // (since fbKey() only depends on server/section/path, not filters) the
  // draft survives untouched across this navigation and the panel
  // re-renders showing its *pre-Apply* state: if Advanced mode was active,
  // the raw textarea reappears with its old (possibly empty) content
  // instead of the just-applied filter, looking like Apply "cleared" it.
  // Switching back to the graphical builder here (rather than leaving
  // Advanced mode on) also matches the graphical builder actually
  // reflecting reality: it should show chips for whatever's now applied.
  if (_fbDraft) {
    _fbDraft.groups   = patch.filters.slice();
    _fbDraft.advanced = false;
    _fbDraft.editing  = null;
  }
  // Guard against a stale _sortDraft left over from a different page (e.g.
  // one that supports sort, if the current page doesn't — ensureSortDraft()
  // is only called when the Sort section actually renders for this exact
  // server/section/path) by checking the draft's key matches here too.
  var newSort = (_sortDraft && _sortDraftKey === sortKey()) ? sortCollectValue() : _state.sort;
  if (newSort !== _state.sort) {
    // A fresh server sort is being deliberately applied here — any
    // client-side column-click override (see sortBy()) should no longer
    // silently mask it. Left untouched when only filters changed, so
    // filtering and column-click sorting stay independent of each other.
    _sortCol = null; _sortAsc = true;
  }
  patch.sort = newSort;
  pushState(patch);
}

function lpCheck(id, label, checked) {
  return '<div class="lp-item"><input type="checkbox" id="' + id + '"'
    + (checked ? ' checked':'') + ' onchange="updateApplyButtonState()"> ' + label + '</div>';
}

// Renders one "Registry Endpoints" row combining a main nav item with an
// optional parenthetical sub nav item on the same line, e.g.
// "Model (Source)" / "Capabilities (Offered)" — matches the old ui.go
// layout instead of stacking each endpoint on its own line. Returns ''
// if neither the main nor the sub endpoint is available.
function lpNavPairRow(mainLabel, mainSection, hasMain,
                       subLabel, subSection, hasSub) {
  if (!hasMain && !hasSub) return '';
  var row = '<div class="lp-nav-row">';
  if (hasMain) {
    var mainActive = _state.section === mainSection;
    var mainCls = 'lp-nav-item lp-nav-inline' +
      (mainActive ? ' lp-nav-active' : '');
    var mainHref = pageHref([], '', {section: mainSection, useExport: false});
    row += '<a class="' + mainCls + '" href="' + esc(mainHref) + '" onclick="'
      + esc(guardedOnclick('pushState('
      + '{section:\'' + mainSection + '\','
      + 'useExport:false})')) + '">' + esc(mainLabel) + '</a>';
  } else {
    row += esc(mainLabel);
  }
  if (hasSub) {
    var subActive = _state.section === subSection;
    var subCls = 'lp-nav-item lp-nav-inline' +
      (subActive ? ' lp-nav-active' : '');
    var subHref = pageHref([], '', {section: subSection, useExport: false});
    row += ' <span class="lp-nav-sub">(<a class="' + subCls
      + '" href="' + esc(subHref) + '" onclick="'
      + esc(guardedOnclick('pushState({section:\'' + subSection + '\','
      + 'useExport:false})')) + '">' + esc(subLabel)
      + '</a>)</span>';
  }
  row += '</div>';
  return row;
}

// ---- Filter builder (JSON left panel) -------------------------------------
//
// Builds xRegistry `?filter=` expressions by walking the runtime /model
// (same spirit as buildInlineOptions()) instead of requiring users to hand
// type dot-notation filter syntax. See the spec's "Filter Flag" and
// "Dot-Notation in Filters" sections for the exact grammar being modeled
// here (registry/spec/core/spec.md).
//
// _fbDraft.groups: array of strings, one per `filter=` OR group; each
//   string may itself contain comma-separated AND'd expressions — this is
//   exactly the wire format already used by _state.filters, so nothing
//   about pushState()/buildAPIURL() needed to change.
// _fbDraft.wiz: the in-progress "add a filter" wizard state:
//   {gPlural, rPlural, level, segs:[{text,kind,join}], op, value}

function fbKey() {
  return serverBase() + '|' + _state.section + '|' + _state.path.join('/');
}

function emptyWizard() {
  return {
    gPlural: null, rPlural: null, level: null,
    segs: [], op: '', value: '', valueTouched: false
  };
}

// Splits a comma-list into trimmed, non-empty parts (used both for the
// `filter=` OR-group wire format and the raw-textarea line format).
function trimSplit(str, sep) {
  return str.split(sep).map(function(s) { return s.trim(); }).filter(Boolean);
}

function ensureFbDraft() {
  var key = fbKey();
  if (!_fbDraft || _fbDraftKey !== key) {
    _fbDraft = {
      groups: _state.filters.slice(), advanced: false, wiz: emptyWizard(),
      editing: null,   // {gi, ei} of the chip currently loaded for editing
      addTarget: null  // explicit OR-group index to AND into, or null for
                        // "the last group" (see fbAddTargetIndex())
    };
    _fbDraftKey = key;
    // Auto-expand the Filters section on a fresh navigation when it
    // arrives with at least one filter already defined (e.g. followed a
    // filtered link, or a bookmark/history entry with filter= set) — no
    // point landing on a page with active filters but hiding that fact
    // behind a collapsed twisty. Still defaults to collapsed when empty,
    // and doesn't fight the user's own manual toggle for the rest of this
    // page view (this only runs when the draft is rebuilt, i.e. once per
    // fresh navigation — see fbKey()).
    _filtersCollapsed = _state.filters.length === 0;
  }
}

function fbRerender() {
  var host = el('lp-filter-section');
  if (!host) return;
  var svBase = serverBase();
  var model  = _modelCache[normalizeURL(svBase)] || null;
  var count  = fbFilterCount(_fbDraft ? _fbDraft.groups : []);
  var filtersOnly = isGridFiltersOnlyMode();
  var inner = (_filtersCollapsed && !filtersOnly) ? '' : buildFilterSectionInner(model);
  // Keep in sync with the initial render (renderJSONLeftPanel()) — the
  // body gets a 6px indent so it visually aligns with the Options/Inlines
  // checkbox rows below it (JSON view) or the Sort section below it (List
  // view). Without this wrapper, any fbRerender() (e.g. picking a wizard
  // dropdown, toggling Advanced mode) would strip that indent back out.
  if (inner) inner = '<div class="lp-filter-indent">' + inner + '</div>';
  host.innerHTML = fbFiltersTitleHTML(count, filtersOnly) + inner;
  updateApplyButtonState();
}

// Renders the collapsible "Filters" section title: label + "(N)" count
// of currently-defined filter expressions (shown even while collapsed,
// so you can tell at a glance whether any are set) + twisty (▶/▼) on
// the right.
function fbFiltersTitleHTML(count, noTwisty) {
  var countHTML = count
    ? ' <span class="lp-title-count">(' + count + ')</span>' : '';
  // "Clear" erases the draft only — it doesn't requery until Apply is
  // clicked, matching every other filter-builder edit (remove chip, remove
  // group, etc.). Only shown when there's actually something to clear.
  // Shown in both JSON view's full panel and Grid/List's filtersOnly panel.
  var clearHTML = count
    ? ' <span class="lp-title-clear" title="Clear all filters (Apply to'
      + ' requery)" onclick="event.stopPropagation(); fbClearAllFilters()">'
      + 'Clear</span>'
    : '';
  if (noTwisty) {
    // Grid/List's filtersOnly panel: Filters is the only section, so a
    // collapse twisty just adds a pointless extra click — always expanded,
    // plain non-interactive title.
    return '<div class="lp-title"><span>Filters' + countHTML + clearHTML
      + '</span></div>';
  }
  var twisty = _filtersCollapsed ? '▶' : '▼';
  return '<div class="lp-title lp-title-collapsible" '
    + 'onclick="fbToggleCollapsed()">'
    + '<span>Filters' + countHTML + clearHTML + '</span>'
    + '<span class="lp-title-twisty lp-title-twisty-right">' + twisty
    + '</span></div>';
}

// Erases the entire filter draft (all OR-groups, and the Advanced-mode raw
// textarea) without requerying — the change only takes effect once Apply is
// clicked, same as every other filter-builder edit.
function fbClearAllFilters() {
  if (!_fbDraft) return;
  _fbDraft.groups  = [];
  _fbDraft.editing = null;
  _fbDraft.addTarget = null;
  fbRerender();
}

// Number of filter expressions (OR-groups) currently defined — matches the
// count shown by the header's Filters toggle button (_state.filters.length,
// see renderHeader()). Each group may itself be a comma-joined AND-list of
// sub-expressions, but those sub-expressions together still form a single
// filter expression, not multiple — counting them separately previously
// made this "(N)" disagree with the header's "(N)" for compound filters.
function fbFilterCount(groups) {
  return (groups && groups.length) || 0;
}

// Toggles the Filters section's collapsed/expanded state. Always starts
// collapsed on a fresh page load (see _filtersCollapsed init); this just
// flips it for the current session/navigation.
function fbToggleCollapsed() {
  _filtersCollapsed = !_filtersCollapsed;
  renderJSONLeftPanel(isGridFiltersOnlyMode());
}

// Whether the left panel is currently showing Grid/List's slimmed-down
// Filters-only view (as opposed to JSON view's full panel). Centralized
// here so every internal filter-builder handler that needs to re-render
// the whole panel (not just the #lp-filter-section host, like fbRerender())
// preserves the correct mode instead of always falling back to the full
// JSON panel.
function isGridFiltersOnlyMode() {
  return _state.section === 'data' && _state.dataView !== 'json'
    && _state.view !== 'json' && _filtersPanelOpen;
}

// Order-independent string-array equality — used by computeApplyDirty() to
// compare "what's currently in the draft/DOM controls" against "what's
// actually applied in _state", regardless of any incidental reordering
// (e.g. checkbox render order vs. _state.inlines's stored order).
function sameStringSet(a, b) {
  a = (a || []).slice().sort();
  b = (b || []).slice().sort();
  if (a.length !== b.length) return false;
  for (var i = 0; i < a.length; i++) { if (a[i] !== b[i]) return false; }
  return true;
}

// Whether the currently-open Filters/Sort/Options/Inlines panel has any
// pending change relative to what's actually applied in _state — drives
// enabling/disabling the Apply button(s) live (see plan.md "Filter Builder
// Apply button — only enable when there are pending changes"). Scoped to
// exactly what filtersOnly mode (Grid/List) renders (filters + sort only);
// the full JSON panel additionally covers doc/binary/collections + inlines,
// none of which filtersOnly mode's DOM even contains.
function computeApplyDirty(filtersOnly) {
  if (_state.section !== 'data') return false;
  if (capHasFlag('filter') && _fbDraft
      && !sameStringSet(fbCollectFilters(), _state.filters)) {
    return true;
  }
  if (capHasFlag('sort') && isCollection(_state.path) && _sortDraft
      && sortCollectValue() !== (_state.sort || '')) {
    return true;
  }
  if (!filtersOnly) {
    var docEl = el('lp-doc'), binEl = el('lp-bin'), colEl = el('lp-col');
    if (docEl && !!docEl.checked !== !!_state.docView)     return true;
    if (binEl && !!binEl.checked !== !!_state.binary)      return true;
    if (colEl && !!colEl.checked !== !!_state.collections) return true;
    if (capHasFlag('inline')) {
      var checkedInlines = qsa('.lp-inline-cb')
        .filter(function(cb) { return cb.checked; })
        .map(function(cb) { return cb.value; });
      if (!sameStringSet(checkedInlines, _state.inlines)) return true;
    }
  }
  return false;
}

// Enables/disables every currently-rendered Apply button (there can be two:
// the leading divider-Apply above Filters, and the trailing one at the
// panel's bottom) based on computeApplyDirty() — called after any control
// that affects the draft/DOM state changes. Deliberately does NOT re-render
// the panel itself (that would blow away focus/scroll/cursor position in
// controls like the advanced filter textarea) — just flips .disabled.
function updateApplyButtonState() {
  var buttons = qsa('.lp-apply');
  if (!buttons.length) return;
  var dirty = computeApplyDirty(isGridFiltersOnlyMode());
  buttons.forEach(function(b) { b.disabled = !dirty; });
}

// Returns the filters array to send: either the builder's groups, or the
// raw textarea lines when Advanced (raw text) mode is active.
function fbCollectFilters() {
  if (!_fbDraft) return _state.filters;
  if (_fbDraft.advanced) {
    var ta = el('lp-filters-raw');
    return ta ? trimSplit(ta.value, '\n') : _fbDraft.groups.slice();
  }
  return _fbDraft.groups.slice();
}

function buildFilterSectionInner(model) {
  var d    = _fbDraft;
  var html = '<label class="fb-adv-toggle"><input type="checkbox"'
    + (d.advanced ? ' checked' : '')
    + ' onchange="fbToggleAdvanced(this.checked)"> Advanced (raw text)</label>';

  if (d.advanced) {
    html += '<textarea class="lp-filter-area" id="lp-filters-raw" '
      + 'oninput="updateApplyButtonState()">'
      + esc(d.groups.join('\n')) + '</textarea>';
    return html;
  }

  html += '<div class="fb-groups">' + fbGroupsHTML(d.groups) + '</div>';
  if (d.editing) {
    html += '<div class="fb-editing-hint">Editing the highlighted filter'
      + ' below</div>';
  }
  html += '<div class="fb-wizard-label">'
    + '<span class="lp-divider-line"></span>'
    + '<span class="fb-wizard-label-text">Filter Builder</span>'
    + '<span class="lp-divider-line"></span></div>';
  html += '<div class="fb-wizard">' + buildWizardHTML(model) + '</div>';
  return html;
}

function fbToggleAdvanced(checked) {
  if (checked) {
    _fbDraft.advanced = true;
  } else {
    var ta = el('lp-filters-raw');
    if (ta) _fbDraft.groups = trimSplit(ta.value, '\n');
    _fbDraft.advanced = false;
  }
  _fbDraft.editing = null;
  fbRerender();
}

function fbGroupsHTML(groups) {
  if (!groups.length) return '<div class="fb-empty">No filters yet</div>';
  var editing = _fbDraft.editing;
  return groups.map(function(g, gi) {
    var exprs = trimSplit(g, ',');
    var chips = exprs.map(function(e, ei) {
      var isEditing = !!editing && editing.gi === gi && editing.ei === ei;
      var cls = 'fb-chip' + (isEditing ? ' fb-chip-editing' : '');
      return '<span class="' + cls + '">'
        + '<span class="fb-chip-text" title="Click to edit"'
        + ' onclick="fbEditExpr(' + gi + ',' + ei + ')">' + esc(e) + '</span>'
        + '<span class="fb-chip-x" title="Remove"'
        + ' onclick="fbRemoveExpr(' + gi + ',' + ei + ')">'
        + '&times;</span></span>';
    }).join('<span class="fb-and">AND</span>');
    var badge = groups.length > 1
      ? '<span class="fb-group-label" title="' + esc(fbGroupPreview(gi))
        + '">' + esc(fbGroupShortLabel(gi)) + '</span>'
      : '';
    return '<div class="fb-group-row">' + badge + chips
      + '<span class="fb-group-x" title="Remove this OR group"'
      + ' onclick="fbRemoveGroup(' + gi + ')">&times;</span></div>';
  }).join('<div class="fb-or">OR</div>');
}

// Keeps _fbDraft.editing's {gi, ei} pointer consistent after a chip/group
// removal shifts indices around it — clearing it if the edited expr (or
// its whole group) was the one removed.
function fbAdjustEditingAfterRemove(gi, ei, groupRemoved) {
  var editing = _fbDraft.editing;
  if (!editing) return;
  if (groupRemoved) {
    if (editing.gi === gi) { _fbDraft.editing = null; return; }
    if (editing.gi > gi)   editing.gi--;
    return;
  }
  if (editing.gi !== gi) return;
  if (editing.ei === ei)  { _fbDraft.editing = null; return; }
  if (editing.ei > ei)    editing.ei--;
}

function fbRemoveExpr(gi, ei) {
  var exprs = trimSplit(_fbDraft.groups[gi], ',');
  exprs.splice(ei, 1);
  var groupRemoved = (exprs.length === 0);
  if (groupRemoved) _fbDraft.groups.splice(gi, 1);
  else              _fbDraft.groups[gi] = exprs.join(',');
  fbAdjustEditingAfterRemove(gi, ei, groupRemoved);
  _fbDraft.addTarget = null;
  fbRerender();
}

function fbRemoveGroup(gi) {
  _fbDraft.groups.splice(gi, 1);
  fbAdjustEditingAfterRemove(gi, -1, true);
  _fbDraft.addTarget = null;
  fbRerender();
}

// Loads an existing filter expression back into the wizard for editing,
// without removing it from its OR group — it stays in place (and is
// visually highlighted, see fbGroupsHTML) until "Update" or "Cancel" is
// clicked. If the expression can't be cleanly parsed against the current
// model (e.g. it was hand-typed via Advanced mode, or uses syntax the
// wizard doesn't model like a literal '*' key), it's left alone — only
// editable via Advanced mode.
function fbEditExpr(gi, ei) {
  var svBase = serverBase();
  var model  = _modelCache[normalizeURL(svBase)] || null;
  var exprs  = trimSplit(_fbDraft.groups[gi], ',');
  var expr   = exprs[ei];
  if (expr === undefined) return;

  var wiz = fbParseExpr(model, expr);
  if (!wiz) return;

  _fbDraft.editing = {gi: gi, ei: ei};
  _fbDraft.wiz      = wiz;
  fbRerender();
}

// Replaces the expression currently being edited, in its original spot
// (same OR group, same AND position) — does not reorder anything.
function fbUpdateExpr() {
  var editing = _fbDraft.editing;
  if (!editing) return;
  var svBase = serverBase();
  var model  = _modelCache[normalizeURL(svBase)] || null;
  var wiz    = _fbDraft.wiz;
  var ctx    = fbRootContext(model, wiz);
  var path   = fbAssemblePath(ctx, wiz.segs);
  if (!path || fbValidate(wiz)) return;

  var exprs = trimSplit(_fbDraft.groups[editing.gi], ',');
  exprs[editing.ei] = path + fbOpToExprSuffix(wiz);
  _fbDraft.groups[editing.gi] = exprs.join(',');

  _fbDraft.editing = null;
  _fbDraft.wiz      = emptyWizard();
  fbRerender();
}

// Discards the in-progress edit, leaving the original expression as-is.
function fbCancelEdit() {
  _fbDraft.editing = null;
  _fbDraft.wiz      = emptyWizard();
  fbRerender();
}

// ---- Wizard: merged attribute+collection pickers, attribute path ---------

// Collection-shadow attribute names: every child collection also shows
// up as `{plural}` (the map itself), `{plural}count`, and `{plural}url`
// attributes on its parent — these are redundant with the "step into"
// choice for that same collection, so they're excluded from the
// attribute portion of a merged picker to avoid confusing duplicates.
function fbCollectionShadowNames(keys) {
  var set = {};
  keys.forEach(function(k) {
    set[k] = true; set[k + 'count'] = true; set[k + 'url'] = true;
  });
  return set;
}

// Merges a resource's version-level attributes with its resource-entity
// attributes (a Resource's JSON representation implicitly inlines its
// default version's attributes), excluding the meta/versions shadow
// attrs (those are offered as "step into" choices instead).
function fbMergeResourceAttrs(rm) {
  var shadow = fbCollectionShadowNames(['meta', 'versions']);
  var merged = {};
  Object.keys(rm.attributes || {}).forEach(function(k) {
    if (!shadow[k]) merged[k] = rm.attributes[k];
  });
  // Resource-specific attrs (self, xid, messageid, etc.) take priority
  // over any same-named version attr.
  Object.keys(rm.resourceattributes || {}).forEach(function(k) {
    if (!shadow[k]) merged[k] = rm.resourceattributes[k];
  });
  return merged;
}

// Builds the "Attributes" option group for a merged picker: real
// attribute names (minus '*' and any shadow names) plus a trailing
// "(other / custom attribute)" escape hatch. Values are prefixed
// "attr:" so the dispatcher can tell them apart from "step into" picks.
function fbMergedAttrOptions(attrsMap, shadowNames) {
  var opts = [];
  Object.keys(attrsMap || {}).sort().forEach(function(k) {
    if (k === '*') return;
    if (shadowNames && shadowNames[k]) return;
    opts.push({value: 'attr:' + k, label: k + '  (' + attrsMap[k].type + ')'});
  });
  opts.push({value: 'attr:__custom__', label: '(other / custom attribute)'});
  return opts;
}

// Renders one merged <select> offering both "filter on an attribute at
// this level" and "step into a child collection" choices, grouped under
// native <optgroup> labels (no custom popover needed). `childOpts` is
// the array of {value, label} step-into choices (already prefixed by
// the caller, e.g. "grp:"/"res:"/"step:"). `singular` is this level's
// singular entity name (e.g. "Registry"/"dir"/"file"), used to label
// the attributes optgroup so it's clear which entity's attributes are
// being offered.
function fbMergedSelectRow(attrOpts, childOpts, onchangeAttr, singular) {
  var html = '<select class="fb-seg-select" onchange="'
    + onchangeAttr + '"><option value="">-- choose --</option>';
  if (childOpts.length) {
    html += '<optgroup label="Step into">'
      + fbOptionsHTML(childOpts, '') + '</optgroup>';
  }
  if (attrOpts.length) {
    html += '<optgroup label="' + esc(singular) + ' Attributes">'
      + fbOptionsHTML(attrOpts, '') + '</optgroup>';
  }
  html += '</select>';
  return '<div class="fb-seg-row">' + html + '</div>';
}

// Applies picking an attribute directly at a merged picker (root, group,
// or resource level) — begins the attribute-path walk at segs[0].
// "__custom__" seeds an empty custom segment so the existing
// fbGenericSegRow() custom-text-input flow (used for deeper segments)
// kicks in unchanged.
function fbApplyAttrChoice(name) {
  if (name === '__custom__') {
    _fbDraft.wiz.segs = [{text: '', kind: 'custom', join: 'attr'}];
  } else {
    _fbDraft.wiz.segs = [{text: name, kind: 'attr', join: 'attr'}];
  }
  _fbDraft.wiz.op    = '';
  _fbDraft.wiz.value = '';
  fbRerender();
}

function fbSetRootChoice(v) {
  if (!v) return;
  if (v.slice(0, 4) === 'grp:')  { fbSetRoot(v.slice(4)); return; }
  if (v.slice(0, 5) === 'attr:') { fbApplyAttrChoice(v.slice(5)); }
}

function fbSetGroupChoice(v) {
  if (!v) return;
  if (v.slice(0, 4) === 'res:')  { fbSetResource(v.slice(4)); return; }
  if (v.slice(0, 5) === 'attr:') { fbApplyAttrChoice(v.slice(5)); }
}

function fbSetResourceChoice(v) {
  if (!v) return;
  if (v === 'step:meta')     { fbSetLevel('meta'); return; }
  if (v === 'step:versions') { fbSetLevel('versions'); return; }
  if (v.slice(0, 5) === 'attr:') { fbApplyAttrChoice(v.slice(5)); }
}

// Determines which parts of the Group type / Resource type / Level chain
// are already implied by the current navigation path (_state.path), so
// the wizard only offers picks for what's still ambiguous relative to
// the JSON currently being viewed — e.g. browsing inside a group type
// collection (depth 1/2) should not re-offer sibling group types, and
// browsing inside a resource collection (depth 3+) should not re-offer
// group/resource type at all.
//
// Path depth convention (see isCollection()/needsDetails() above):
//   0: registry root            1: group type coll.   2: group entity
//   3: resource type coll.      4: resource entity
//   5: "meta" entity or "versions" coll.   6: version entity
function fbPathAnchor(model) {
  var path  = _state.path;
  var depth = path.length;
  if (depth === 0) {
    return {
      showGroup: true, showResource: true, showLevel: true,
      gPlural: null, rPlural: null, level: null
    };
  }
  var gPlural = path[0];
  if (depth <= 2) {
    return {
      showGroup: false, showResource: true, showLevel: true,
      gPlural: gPlural, rPlural: null, level: null
    };
  }
  var rPlural = path[2];
  if (depth <= 4) {
    return {
      showGroup: false, showResource: false, showLevel: true,
      gPlural: gPlural, rPlural: rPlural, level: null
    };
  }
  // depth >= 5: already inside the "meta" or "versions" sub-scope.
  var level = (path[4] === 'meta') ? 'meta' : 'versions';
  return {
    showGroup: false, showResource: false, showLevel: false,
    gPlural: gPlural, rPlural: rPlural, level: level
  };
}

// {prefix, attrsMap} for the wizard's chosen root, before any attribute
// path segments are applied. Merges the path anchor (the parts of
// Group/Resource/Level already implied by where you're currently
// browsing — see fbPathAnchor()) with any remaining picks made in the
// wizard itself. Resource-level attrs live in 3 separate maps depending
// wizard itself. Resource-level attrs live in 3 separate maps depending
// on which "level" is picked (resourceattributes / metaattributes /
// attributes==version-attrs) — see model shape in registry.
function fbRootContext(model, wiz) {
  var anchor  = fbPathAnchor(model);
  var gPlural = anchor.gPlural || wiz.gPlural;
  var prefix  = '';

  if (!gPlural) {
    return {prefix: '', attrsMap: (model && model.attributes) || {}};
  }
  var gm = model && model.groups && model.groups[gPlural];
  // Only emit a path segment for parts the *wizard* chose — anything
  // already implied by the current browsing path is left off, since the
  // filter is relative to whatever JSON is currently being viewed.
  if (!anchor.gPlural) prefix += gPlural + '.';
  if (!gm) return {prefix: prefix, attrsMap: {}};

  var rPlural = anchor.rPlural || wiz.rPlural;
  if (!rPlural) return {prefix: prefix, attrsMap: gm.attributes || {}};

  var rm = gm.resources && gm.resources[rPlural];
  if (!anchor.rPlural) prefix += rPlural + '.';
  if (!rm) return {prefix: prefix, attrsMap: {}};

  var level = anchor.level || wiz.level || 'resource';
  var attrsMap;
  if (level === 'meta') {
    if (!anchor.level) prefix += 'meta.';
    attrsMap = rm.metaattributes || {};
  } else if (level === 'versions') {
    if (!anchor.level) prefix += 'versions.';
    attrsMap = rm.attributes || {};
  } else {
    // "resource" level: a Resource's JSON implicitly inlines its default
    // version's attributes, so merge both maps (see
    // fbMergeResourceAttrs()) rather than resourceattributes alone.
    attrsMap = fbMergeResourceAttrs(rm);
  }
  return {prefix: prefix, attrsMap: attrsMap};
}

// object/map/array/leaf — drives whether we need another seg picker.
function fbAttrKind(attr) {
  if (!attr) return 'leaf';
  if (attr.type === 'object' && attr.attributes) return 'object';
  if (attr.type === 'map')   return 'map';
  if (attr.type === 'array') return 'array';
  return 'leaf';
}

function fbAttrOptions(attrsMap) {
  var opts = [];
  Object.keys(attrsMap).sort().forEach(function(k) {
    if (k === '*') return;
    opts.push({value: k, label: k + '  (' + attrsMap[k].type + ')'});
  });
  opts.push({value: '__custom__', label: '(other / custom attribute)'});
  return opts;
}

function fbMapOptions() {
  return [
    {value: '*',          label: '.* (any value)'},
    {value: '__custom__', label: '(specific key)'}
  ];
}

function fbArrayOptions() {
  return [
    {value: '[*]',  label: '[*] (any item)'},
    {value: '[-1]', label: '[-1] (last item)'}
  ];
}

function fbOptionsHTML(options, chosen) {
  return options.map(function(o) {
    var sel = (chosen === o.value) ? ' selected' : '';
    return '<option value="' + esc(o.value) + '"' + sel + '>'
      + esc(o.label) + '</option>';
  }).join('');
}

// ---- Sort picker (JSON left panel) ----------------------------------------
//
// Builds a single `?sort=<ATTRIBUTE>[=asc|desc]` value (see the spec's
// "Sort Flag" section). Much simpler than the Filter Builder: only one
// attribute may be chosen (no AND/OR groups, no comparison operator), and
// per spec the attribute MUST be a scalar (or a map value, e.g.
// `labels.stage`) directly on the collection's entities — no drilling into
// a nested child collection. So this reuses the Filter Builder's
// model-driven attribute enumeration (fbRootContext/fbPathAnchor) and its
// <select>-based picker styling (.fb-seg-row/.fb-seg-select/.fb-seg-custom),
// but with the "step into a child collection" choices left out entirely.
//
// _sortDraft: {mode, attr, mapKey, custom, desc}
//   mode: '' (none chosen) | 'attr' (plain scalar) | 'map' (needs a key)
//         | 'custom' (freeform dot-path)
//   attr: chosen attribute name when mode is 'attr' or 'map'
//   mapKey: the key typed in for a chosen map attribute (mode 'map')
//   custom: freeform dot-path text (mode 'custom')
//   desc: boolean, true = descending order

function sortKey() {
  return serverBase() + '|' + _state.section + '|' + _state.path.join('/');
}

function ensureSortDraft() {
  var key = sortKey();
  if (!_sortDraft || _sortDraftKey !== key) {
    var parts = trimSplit(_state.sort || '', '=');
    var attrPath = parts[0] || '';
    var desc = parts[1] === 'desc';
    _sortDraft = sortDraftFromPath(attrPath, desc);
    _sortDraftKey = key;
  }
}

// Reconstructs a draft {mode, attr, mapKey, custom, desc} from a wire-format
// attribute path (e.g. '', 'name', 'labels.stage') — used both when a
// draft is first created from _state.sort, and there's no model context
// needed here since we're just splitting text, not validating it against
// the model (validation only affects which choices the <select> offers).
function sortDraftFromPath(attrPath, desc) {
  if (!attrPath) {
    return {mode: '', attr: '', mapKey: '', custom: '', desc: desc};
  }
  var dot = attrPath.indexOf('.');
  if (dot === -1) {
    return {mode: 'attr', attr: attrPath, mapKey: '', custom: '', desc: desc};
  }
  return {
    mode: 'map', attr: attrPath.slice(0, dot),
    mapKey: attrPath.slice(dot + 1), custom: '', desc: desc
  };
}

function sortRerender() {
  var host = el('lp-sort-section');
  if (!host) return;
  var svBase = serverBase();
  var model  = _modelCache[normalizeURL(svBase)] || null;
  host.innerHTML = buildSortSectionInner(model);
  updateApplyButtonState();
}

// Collection-shadow attribute names to exclude from the Sort picker at
// the Group-collection level (depth 1): a Group entity's JSON inlines
// `{plural}`/`{plural}count`/`{plural}url` for each of its child Resource
// types, which are redundant/invalid as sort targets (sort must not
// traverse into a nested collection). Resource- and Version-level
// contexts don't need this (fbRootContext already excludes the
// meta/versions shadow attrs for Resources, and Versions have no
// children at all).
function sortShadowNames(model) {
  var anchor = fbPathAnchor(model);
  if (anchor.gPlural && !anchor.rPlural) {
    var gm = model && model.groups && model.groups[anchor.gPlural];
    return fbCollectionShadowNames(Object.keys((gm && gm.resources) || {}));
  }
  return null;
}

// Builds the Sort attribute <select> options: scalar ("leaf") attributes
// directly, map attributes (need a follow-up key input), plus a trailing
// "(other / custom attribute)" freeform escape hatch. Object/array
// attributes are excluded entirely — not valid sort targets.
function sortAttrOptions(attrsMap, shadowNames) {
  var opts = [];
  Object.keys(attrsMap || {}).sort().forEach(function(k) {
    if (k === '*') return;
    if (shadowNames && shadowNames[k]) return;
    var kind = fbAttrKind(attrsMap[k]);
    if (kind !== 'leaf' && kind !== 'map') return;
    var suffix = kind === 'map'
      ? '  (map — pick a key)' : '  (' + attrsMap[k].type + ')';
    opts.push({value: k, label: k + suffix});
  });
  opts.push({value: '__custom__', label: '(other / custom attribute)'});
  return opts;
}

// The wire-format attribute path implied by the current draft, or '' if
// nothing usable has been chosen yet (e.g. a map attribute with no key
// typed in, or an empty custom field).
function sortDraftPath() {
  if (!_sortDraft) return '';
  if (_sortDraft.mode === 'attr')   return _sortDraft.attr;
  if (_sortDraft.mode === 'map') {
    return _sortDraft.mapKey ? _sortDraft.attr + '.' + _sortDraft.mapKey : '';
  }
  if (_sortDraft.mode === 'custom') return _sortDraft.custom;
  return '';
}

// Returns the wire-format `sort=` value to send (e.g. '', 'name',
// 'name=desc') — read by applyJSONOptions().
function sortCollectValue() {
  var path = sortDraftPath();
  if (!path) return '';
  return _sortDraft.desc ? path + '=desc' : path;
}

function sortSetAttr(value, model) {
  if (value === '__custom__') {
    _sortDraft.mode = 'custom'; _sortDraft.attr = ''; _sortDraft.mapKey = '';
  } else if (value === '') {
    _sortDraft.mode = ''; _sortDraft.attr = '';
    _sortDraft.mapKey = ''; _sortDraft.custom = '';
  } else {
    var ctx  = fbRootContext(model, {});
    var kind = fbAttrKind(ctx.attrsMap[value]);
    _sortDraft.mode = (kind === 'map') ? 'map' : 'attr';
    _sortDraft.attr = value;
    _sortDraft.mapKey = '';
  }
  sortRerender();
}

function sortSetMapKey(value) {
  _sortDraft.mapKey = value;
  sortRerender();
}

function sortSetCustom(value) {
  _sortDraft.custom = value;
  sortRerender();
}

function sortSetOrder(desc) {
  _sortDraft.desc = desc;
  sortRerender();
}

function sortClear() {
  _sortDraft = {mode: '', attr: '', mapKey: '', custom: '', desc: false};
  sortRerender();
}

function buildSortSectionInner(model) {
  var ctx        = fbRootContext(model, {});
  var shadow     = sortShadowNames(model);
  var options    = sortAttrOptions(ctx.attrsMap, shadow);
  var chosenVal  = _sortDraft.mode === 'custom'
    ? '__custom__' : _sortDraft.attr;
  var html = '<div class="fb-seg-row"><span class="sort-label">'
    + 'Sort:</span><select class="fb-seg-select" '
    + 'onchange="sortSetAttr(this.value, '
    + '_modelCache[normalizeURL(serverBase())])">'
    + '<option value="">-- none --</option>'
    + fbOptionsHTML(options, chosenVal)
    + '</select></div>';

  // Everything below the initial "Sort:" row (map-key/custom-path input,
  // Order asc/desc + Clear) gets the same 6px indent used under the
  // Filters title — visually ties them to this section and keeps them
  // aligned with the Options checkbox rows below (see .lp-filter-indent).
  var below = '';

  if (_sortDraft.mode === 'map') {
    below += '<div class="fb-seg-row">'
      + '<input type="text" class="fb-seg-custom" '
      + 'placeholder="key name (e.g. stage)" '
      + 'value="' + esc(_sortDraft.mapKey) + '" '
      + 'onchange="sortSetMapKey(this.value)"></div>';
  } else if (_sortDraft.mode === 'custom') {
    below += '<div class="fb-seg-row">'
      + '<input type="text" class="fb-seg-custom" '
      + 'placeholder="dot-path e.g. labels.stage" '
      + 'value="' + esc(_sortDraft.custom) + '" '
      + 'onchange="sortSetCustom(this.value)"></div>';
  }

  if (sortDraftPath()) {
    below += '<div class="fb-seg-row sort-order-row">'
      + '<span class="sort-order-label">Order:</span>'
      + '<div class="boolSeg sort-order-seg">'
      + '<button type="button" class="boolSegBtn'
      + (!_sortDraft.desc ? ' boolSegActive' : '')
      + '" onclick="sortSetOrder(false)">asc</button>'
      + '<button type="button" class="boolSegBtn'
      + (_sortDraft.desc ? ' boolSegActive' : '')
      + '" onclick="sortSetOrder(true)">desc</button>'
      + '</div>'
      + '<span class="sort-clear-btn" onclick="sortClear()">Clear sort</span>'
      + '</div>';
  }

  if (below) html += '<div class="lp-filter-indent">' + below + '</div>';

  return html;
}

function fbSelectRow(label, options, chosen, onchangeAttr) {
  return '<div class="fb-seg-row"><span class="fb-seg-label">'
    + esc(label) + ':</span>'
    + '<select class="fb-seg-select" onchange="' + onchangeAttr + '">'
    + '<option value="">-- choose --</option>'
    + fbOptionsHTML(options, chosen)
    + '</select></div>';
}

// Renders one attribute/map-key/array-index picker row for wiz.segs[i].
function fbGenericSegRow(i, options, seg, joinKind) {
  var chosenVal = !seg ? '' : (seg.kind === 'custom' ? '__custom__' : seg.text);
  var onchg = 'fbSetSeg(' + i + ', this.value, \'' + joinKind + '\')';
  var html  = '<select class="fb-seg-select" onchange="' + onchg + '">'
    + '<option value="">-- choose --</option>'
    + fbOptionsHTML(options, chosenVal)
    + '</select>';
  if (chosenVal === '__custom__') {
    var custOnchg =
      'fbSetSegCustom(' + i + ', this.value, \'' + joinKind + '\')';
    var curVal = seg && seg.kind === 'custom' ? seg.text : '';
    html += '<input type="text" class="fb-seg-custom" placeholder="name"'
      + ' id="fb-seg-custom-' + i + '"'
      + ' value="' + esc(curVal) + '"'
      + ' oninput="fbSegCustomInput(' + i + ')"'
      + ' onchange="' + custOnchg + '">'
      + '<button type="button" class="fb-seg-custom-commit"'
      + (curVal.trim() ? '' : ' disabled')
      + ' title="Confirm name" '
      + 'onclick="fbCommitSegCustom(' + i + ', \'' + joinKind + '\')">'
      + '\u23ce</button>';
  }
  return '<div class="fb-seg-row">' + html + '</div>';
}

// Live-typing handler for the custom-attribute-name input: just toggles the
// adjacent commit (\u23ce) button's enabled state as the user types, without
// a full fbRerender() (which would rebuild the input and steal focus).
// The actual value only gets committed via fbSetSegCustom(), triggered by
// the input's onchange (blur/Enter) or by clicking the commit button.
function fbSegCustomInput(i) {
  var inp = el('fb-seg-custom-' + i);
  var btn = inp && inp.nextElementSibling;
  if (btn && btn.classList.contains('fb-seg-custom-commit')) {
    btn.disabled = !inp.value.trim();
  }
}

function fbCommitSegCustom(i, joinKind) {
  var inp = el('fb-seg-custom-' + i);
  if (inp) fbSetSegCustom(i, inp.value, joinKind);
}

function fbSetRoot(v) {
  _fbDraft.wiz = emptyWizard();
  _fbDraft.wiz.gPlural = v || null;
  fbRerender();
}

function fbSetResource(v) {
  _fbDraft.wiz.rPlural = v || null;
  _fbDraft.wiz.level   = null;
  _fbDraft.wiz.segs    = [];
  _fbDraft.wiz.op      = '';
  _fbDraft.wiz.value   = '';
  fbRerender();
}

function fbSetLevel(v) {
  _fbDraft.wiz.level = v || null;
  _fbDraft.wiz.segs  = [];
  _fbDraft.wiz.op    = '';
  _fbDraft.wiz.value = '';
  fbRerender();
}

function fbSetSeg(i, value, joinKind) {
  var segs = _fbDraft.wiz.segs.slice(0, i);
  if (value === '__custom__') {
    segs.push({text: '', kind: 'custom', join: joinKind});
  } else if (value !== '') {
    segs.push({text: value, kind: 'attr', join: joinKind});
  }
  _fbDraft.wiz.segs  = segs;
  _fbDraft.wiz.op    = '';
  _fbDraft.wiz.value = '';
  fbRerender();
}

function fbSetSegCustom(i, text, joinKind) {
  var segs = _fbDraft.wiz.segs.slice(0, i + 1);
  segs[i] = {text: text, kind: 'custom', join: joinKind};
  _fbDraft.wiz.segs  = segs;
  _fbDraft.wiz.op    = '';
  _fbDraft.wiz.value = '';
  fbRerender();
}

function fbSetOp(v) {
  _fbDraft.wiz.op = v;
  _fbDraft.wiz.valueTouched = false;  // don't show a stale error immediately
  fbRerender();
  var input = document.querySelector('.fb-val-input');
  if (input) input.focus();
}

// Live-typing handler: update the value + patch just the error/Add-button
// state in place, without a full fbRerender() (which would rebuild the
// input and steal focus mid-keystroke). Typing always hides the error
// immediately; it only reappears once the field is blurred while invalid.
function fbOnValueInput(inputEl) {
  _fbDraft.wiz.value = inputEl.value;
  _fbDraft.wiz.valueTouched = false;
  fbPatchOpValueState();
}

function fbOnValueBlur() {
  _fbDraft.wiz.valueTouched = true;
  fbPatchOpValueState();
}

function fbPatchOpValueState() {
  var err     = fbValidate(_fbDraft.wiz);
  var showErr = _fbDraft.wiz.valueTouched && !!err;
  var errEl   = el('fb-error');
  if (errEl) {
    errEl.style.display = showErr ? '' : 'none';
    errEl.textContent   = showErr ? err : '';
  }
  qsa('.fb-add-btn').forEach(function(b) { b.disabled = !!err; });
}

function fbAssemblePath(ctx, segs) {
  var out = ctx.prefix;
  segs.forEach(function(s, i) {
    if (!s.text) return;
    if (s.join === 'arr') out += s.text;
    else                  out += (i === 0 ? '' : '.') + s.text;
  });
  return out;
}

function fbOpToExprSuffix(wiz) {
  var op = wiz.op || 'present';
  switch (op) {
    case 'present': return '';
    case 'absent':  return '=null';
    case 'eq':      return '=' + wiz.value;
    case 'ne':      return '!=' + wiz.value;
    case 'lt':      return '<' + wiz.value;
    case 'le':      return '<=' + wiz.value;
    case 'gt':      return '>' + wiz.value;
    case 'ge':      return '>=' + wiz.value;
  }
  return '';
}

function fbValidate(wiz) {
  var op = wiz.op || 'present';
  if (op === 'present' || op === 'absent') return null;
  var v = wiz.value || '';
  if (v === '') return 'A value is required for this operator.';
  var isCompareOp = (op === 'lt' || op === 'le' || op === 'gt' || op === 'ge');
  if (isCompareOp && v.indexOf('*') !== -1) {
    return 'Wildcards are not allowed with <, <=, >, >= operators.';
  }
  return null;
}

// Update/Cancel buttons shown while editing an existing filter
// expression (_fbDraft.editing). `dis` is the ' disabled' attribute
// string (or '') to apply to Update — disabled when the wizard isn't
// currently in a valid, complete state.
function fbEditingBarButtons(dis) {
  return '<button class="fb-add-btn"' + dis + ' onclick="fbUpdateExpr()">'
    + 'Update</button>'
    + '<button class="fb-add-btn fb-cancel-btn" onclick="fbCancelEdit()">'
    + 'Cancel</button>';
}

// Persistent Update/Cancel row shown whenever an expression is being
// edited, even if the wizard has been navigated away from a complete
// leaf/op/value state (e.g. the user deleted a breadcrumb to redefine
// the attribute path mid-edit). Without this, Update/Cancel would only
// ever appear via fbOpValueRow()'s add-row — which isn't rendered at
// all once the wizard falls out of its "complete" state — making it
// look like editing was silently abandoned. Update is disabled until
// the wizard reaches a valid, complete state again (mirroring
// fbUpdateExpr()'s own "!path || fbValidate(wiz)" guard, so the button
// is never enabled for a click that would otherwise be a no-op); Cancel
// always works so the user can back out.
function fbEditingBar() {
  var svBase = serverBase();
  var model  = _modelCache[normalizeURL(svBase)] || null;
  var wiz    = _fbDraft.wiz;
  var ctx    = fbRootContext(model, wiz);
  var path   = fbAssemblePath(ctx, wiz.segs);
  var err    = !path ? 'Choose an attribute to filter on.' : fbValidate(wiz);
  var dis    = err ? ' disabled' : '';
  return '<div class="fb-add-row">' + fbEditingBarButtons(dis) + '</div>';
}

// `presenceOnly` restricts the operator choices to "is present"/"is
// absent" — used when stopping the wizard at a complex (object/map/
// array) attribute rather than drilling down to one of its scalar
// children, since comparison operators don't make sense against a
// complex value.
function fbOpValueRow(presenceOnly) {
  var wiz = _fbDraft.wiz;
  var op  = wiz.op || 'present';
  var ops = [
    {value: 'present', label: 'is present'},
    {value: 'absent',  label: 'is absent'}
  ];
  if (!presenceOnly) {
    ops = ops.concat([
      {value: 'eq', label: '= (equals)'},
      {value: 'ne', label: '!= (not equals)'},
      {value: 'lt', label: '< (less than)'},
      {value: 'le', label: '<= (less or equal)'},
      {value: 'gt', label: '> (greater than)'},
      {value: 'ge', label: '>= (greater or equal)'}
    ]);
  }
  var html = '<div class="fb-op-row">'
    + '<select class="fb-op-select" onchange="fbSetOp(this.value)">'
    + fbOptionsHTML(ops, op)
    + '</select>';
  if (op !== 'present' && op !== 'absent') {
    html += '<input type="text" class="fb-val-input" placeholder="value"'
      + ' value="' + esc(wiz.value || '') + '"'
      + ' oninput="fbOnValueInput(this)" onblur="fbOnValueBlur()">';
  }
  html += '</div>';

  // Error div is always emitted (display toggled) so fbPatchOpValueState()
  // can update it in place while typing, without a full fbRerender().
  var err     = fbValidate(wiz);
  var showErr = wiz.valueTouched && !!err;
  html += '<div class="fb-error" id="fb-error"'
    + (showErr ? '' : ' style="display:none"') + '>'
    + esc(showErr ? err : '') + '</div>';

  var dis = err ? ' disabled' : '';
  html += '<div class="fb-add-row">';
  if (_fbDraft.editing) {
    html += fbEditingBarButtons(dis);
  } else if (_fbDraft.groups.length === 0) {
    // Nothing to AND/OR with yet — a single unambiguous action.
    html += '<button class="fb-add-btn"' + dis + ' onclick="fbAdd(\'or\')">'
      + '+ Add filter</button>';
  } else if (_fbDraft.groups.length > 1) {
    // With more than one existing OR-group, "AND" is ambiguous — the
    // AND button becomes a two-zone "split button": the button itself
    // (left-aligned text) plus a compact overlay <select> on its right
    // edge showing only "F1"/"F2"/... so the target-picker visually
    // reads as part of the AND button only, not the OR button.
    var targetIdx = fbAddTargetIndex();
    html += '<span class="fb-and-split">'
      + '<button class="fb-add-btn fb-and-split-btn"' + dis
      + ' onclick="fbAdd(\'and\')">+ Add (AND) to</button>'
      + '<select class="fb-and-split-target"'
      + ' onchange="fbSetAddTarget(this.value)">'
      + _fbDraft.groups.map(function(g, gi) {
          var sel = (gi === targetIdx) ? ' selected' : '';
          return '<option value="' + gi + '"' + sel
            + ' title="' + esc(fbGroupPreview(gi)) + '">'
            + esc(fbGroupShortLabel(gi)) + '</option>';
        }).join('') + '</select></span>'
      + '<button class="fb-add-btn"' + dis + ' onclick="fbAdd(\'or\')">'
      + '+ Add (OR)</button>';
  } else {
    html += '<button class="fb-add-btn"' + dis + ' onclick="fbAdd(\'and\')">'
      + '+ Add (AND)</button>'
      + '<button class="fb-add-btn"' + dis + ' onclick="fbAdd(\'or\')">'
      + '+ Add (OR)</button>';
  }
  html += '</div>';
  return html;
}

// Which OR-group index "+ Add (AND)" should append to. Explicit picks
// (via the "into: ..." dropdown, shown once there's more than one group)
// are stored in _fbDraft.addTarget; null falls back to the last group,
// i.e. the common case of building up the most recently added group.
function fbAddTargetIndex() {
  var n = _fbDraft.groups.length;
  if (!n) return -1;
  var t = _fbDraft.addTarget;
  if (t === null || t === undefined || t < 0 || t >= n) return n - 1;
  return t;
}

function fbSetAddTarget(v) {
  _fbDraft.addTarget = (v === '' ? null : parseInt(v, 10));
  fbRerender();
}

// Compact "F1"/"F2"/... label used for the group corner-badge and the
// AND split-button's overlay dropdown; full context is exposed via the
// title="" tooltip (see fbGroupPreview) on both.
function fbGroupShortLabel(gi) {
  return 'F' + (gi + 1);
}

// Full preview for a group, e.g. "Filter 2: name=foo AND ep…" —
// truncated so long filters don't blow out a tooltip. Used as the
// title="" attribute on the group badge and split-button options.
function fbGroupPreview(gi) {
  var text = _fbDraft.groups[gi].replace(/,/g, ' AND ');
  if (text.length > 28) text = text.slice(0, 27) + '\u2026';
  return 'Filter ' + (gi + 1) + ': ' + text;
}

function fbAdd(mode) {
  var svBase = serverBase();
  var model  = _modelCache[normalizeURL(svBase)] || null;
  var wiz    = _fbDraft.wiz;
  var ctx    = fbRootContext(model, wiz);
  var path   = fbAssemblePath(ctx, wiz.segs);
  if (!path || fbValidate(wiz)) return;
  var expr = path + fbOpToExprSuffix(wiz);

  if (mode === 'and' && _fbDraft.groups.length > 0) {
    var target = fbAddTargetIndex();
    _fbDraft.groups[target] = _fbDraft.groups[target] + ',' + expr;
  } else {
    _fbDraft.groups.push(expr);
    _fbDraft.addTarget = null;
  }
  // Fully reset the wizard (including group/resource/level) after each
  // add, so the breadcrumb clears and the next filter starts fresh from
  // the top-level picker rather than assuming the same entity scope.
  _fbDraft.wiz = emptyWizard();
  fbRerender();
}

// Given the current traversal kind ('object'/'map'/'array') and the item
// type it just consumed, returns the next {kind, attrsMap, item} to walk
// into. Shared by the object/map/array branches of buildWizardHTML() below.
function fbNextFrontier(nextAttr) {
  var k = fbAttrKind(nextAttr);
  if (k === 'object') {
    return {kind: 'object', attrsMap: nextAttr.attributes, item: null};
  }
  if (k === 'map') {
    return {kind: 'map', attrsMap: null, item: nextAttr.item};
  }
  if (k === 'array') {
    return {kind: 'array', attrsMap: null, item: nextAttr.item};
  }
  return {kind: 'leaf', attrsMap: null, item: null};
}

// Splits a filter expression's operator+value suffix off its path,
// reversing fbOpToExprSuffix(). Longest operators are tried first so
// e.g. "<=" isn't mistaken for "<".
function fbSplitExprOp(expr) {
  var m = expr.match(/(!=|<>|<=|>=|=|<|>)/);
  if (!m) return {path: expr, op: 'present', value: ''};
  var path = expr.slice(0, m.index);
  var val  = expr.slice(m.index + m[1].length);
  if (m[1] === '=' && val === 'null') {
    return {path: path, op: 'absent', value: ''};
  }
  if ((m[1] === '!=' || m[1] === '<>') && val === 'null') {
    // Spec: "!=null"/"<>null" means the same as "present" — best effort.
    return {path: path, op: 'present', value: ''};
  }
  var opMap = {
    '=': 'eq', '!=': 'ne', '<>': 'ne',
    '<': 'lt', '<=': 'le', '>': 'gt', '>=': 'ge'
  };
  return {path: path, op: opMap[m[1]], value: val};
}

// Tokenizes a dot/bracket path string into ordered {text, join} segments,
// mirroring the join conventions used by fbAssemblePath(): 'attr' segments
// were joined with a leading dot (except the very first); 'arr' segments
// are bracket literals ("[*]"/"[-1]") with no separator.
function fbTokenizePath(str) {
  var toks = [];
  var re   = /(\[[^\]]*\])|([^.\[]+)/g;
  var m;
  while ((m = re.exec(str))) {
    if (m[1]) toks.push({text: m[1], join: 'arr'});
    else      toks.push({text: m[2], join: 'attr'});
  }
  return toks;
}

// Reverses fbAssemblePath()/fbOpToExprSuffix() to reconstruct a wizard
// state from an existing filter expression string, so clicking a chip
// can load it back in for editing. Returns null if the expression
// doesn't cleanly match the current model/path anchor (e.g. a
// hand-typed Advanced-mode expression) — callers should leave those
// alone rather than show a bogus/incomplete wizard.
function fbParseExpr(model, expr) {
  var split  = fbSplitExprOp(expr);
  var anchor = fbPathAnchor(model);
  var tokens = fbTokenizePath(split.path);
  var idx    = 0;

  var wiz = emptyWizard();
  wiz.gPlural = anchor.gPlural;
  wiz.rPlural = anchor.rPlural;
  wiz.level   = anchor.level;

  if (anchor.showGroup && tokens[idx] && model && model.groups
      && model.groups[tokens[idx].text]) {
    wiz.gPlural = tokens[idx].text;
    idx++;
  }
  var gm = wiz.gPlural && model && model.groups && model.groups[wiz.gPlural];

  if (wiz.gPlural && anchor.showResource) {
    if (tokens[idx] && gm && gm.resources && gm.resources[tokens[idx].text]) {
      wiz.rPlural = tokens[idx].text;
      idx++;
    }
    // No resource-plural token found — this expr targets the group's
    // own attributes directly (wiz.rPlural stays null/unset).
  }

  if (wiz.gPlural && wiz.rPlural && anchor.showLevel) {
    if (tokens[idx] && tokens[idx].text === 'meta') {
      wiz.level = 'meta';
      idx++;
    } else if (tokens[idx] && tokens[idx].text === 'versions') {
      wiz.level = 'versions';
      idx++;
    }
    // Otherwise this expr targets the resource entity's own (merged)
    // attributes directly — wiz.level stays null (implicit "resource").
  }

  var ctx      = fbRootContext(model, wiz);
  var runKind  = 'object';
  var runAttrs = ctx.attrsMap;
  var runItem  = null;
  var segs     = [];

  for (; idx < tokens.length; idx++) {
    var tok = tokens[idx];
    var next;
    if (runKind === 'object') {
      if (runAttrs && runAttrs.hasOwnProperty(tok.text)) {
        segs.push({text: tok.text, kind: 'attr', join: tok.join});
        next = fbNextFrontier(runAttrs[tok.text]);
      } else {
        segs.push({text: tok.text, kind: 'custom', join: tok.join});
        next = {kind: 'leaf', attrsMap: null, item: null};
      }
    } else if (runKind === 'map') {
      if (tok.text === '*') {
        segs.push({text: '*', kind: 'attr', join: tok.join});
        next = fbNextFrontier(runItem);
      } else {
        segs.push({text: tok.text, kind: 'custom', join: tok.join});
        next = {kind: 'leaf', attrsMap: null, item: null};
      }
    } else if (runKind === 'array') {
      segs.push({text: tok.text, kind: 'attr', join: tok.join});
      next = fbNextFrontier(runItem);
    } else {
      return null;  // trailing tokens past a leaf — not one we built
    }
    runKind = next.kind; runAttrs = next.attrsMap; runItem = next.item;
  }
  if (!segs.length) return null;

  wiz.segs         = segs;
  wiz.op           = split.op;
  wiz.value        = split.value;
  wiz.valueTouched = false;
  return wiz;
}

// Walks the model (mirroring fbRootContext's traversal) to render one
// picker row per already-chosen path segment, plus one more row for the
// next choice — recursing through object/map/array attribute types until
// a leaf (scalar/any/custom) is reached, at which point the operator+value
// row is shown.
//
// Rendering strategy (breadcrumb hybrid, see plan.md "Breadcrumb-style
// wizard rendering"): already-decided levels collapse into small
// plain-text crumbs (clickable to reopen via fbJumpTo()); exactly one
// native <select>/row is shown at a time for the next undecided choice.
// This keeps the picker itself a plain OS-native <select> (good mobile
// UX, no custom popover) while still giving a compact "path so far"
// view instead of stacking every already-answered dropdown.
// Builds one crumb descriptor; rendered later by fbBreadcrumbHTML() once
// the full list is known, so the LAST crumb (the one right before the
// active picker) can be styled differently from earlier ones. `navTo`
// is invoked by clicking the crumb's text (keeps this level's own
// choice, clears only what comes after it — "step back in to re-decide
// what's next"); `delTo` is invoked by the crumb's own "x" (deletes
// this level's choice AND everything after it).
function fbCrumb(label, navTo, delTo) {
  return {label: label, navTo: navTo, delTo: delTo};
}

function fbCrumbSep() {
  return '<span class="fb-crumb-sep">\u203a</span>';
}

// Each crumb renders as one pill (like a filter-expression chip): text
// on the left, a small red "x" on the right. The LAST crumb (closest to
// the active picker) has non-clickable text, since there's no "next"
// step to reopen for it — it's already driving the picker that's
// showing; only its "x" (delete this level + everything after) applies.
// Earlier crumbs have clickable text ("step back in": keep this level's
// own choice, clear only its descendants) as well as their own "x"
// (delete this level's choice and everything after it).
function fbBreadcrumbHTML(crumbs) {
  if (!crumbs.length) return '';
  var parts = crumbs.map(function(c, i) {
    var isLast = (i === crumbs.length - 1);
    var text = isLast
      ? '<span class="fb-crumb-text fb-crumb-text-static">'
        + esc(c.label) + '</span>'
      : '<span class="fb-crumb-text" onclick="' + c.navTo + '">'
        + esc(c.label) + '</span>';
    var x = '<span class="fb-crumb-x" onclick="' + c.delTo
      + '" title="Remove">\u00d7</span>';
    return '<span class="fb-crumb">' + text + x + '</span>';
  });
  return '<div class="fb-breadcrumb">' + parts.join(fbCrumbSep())
    + '</div>';
}

// Clears the attribute-path segments (and any pending op/value) while
// keeping the current Group/Resource/Level scope — used when the user
// clicks the LEVEL crumb's text ("keep this scope, let me re-pick the
// attribute path under it").
function fbClearSegsKeepLevel() {
  _fbDraft.wiz.segs  = [];
  _fbDraft.wiz.op    = '';
  _fbDraft.wiz.value = '';
  fbRerender();
}

// Keeps segs[0..idx] (i.e. the segment at `idx` itself), clearing any
// deeper segments and the pending op/value — used when the user clicks
// a SEG crumb's text ("keep this attribute step, let me re-pick what's
// under it").
function fbTruncateSegsKeepIdx(idx) {
  _fbDraft.wiz.segs  = _fbDraft.wiz.segs.slice(0, idx + 1);
  _fbDraft.wiz.op    = '';
  _fbDraft.wiz.value = '';
  fbRerender();
}

// Clicking a crumb's text ("step back in"): keeps that level's own
// choice, clears only what comes after it. Group/Resource reuse the
// next level's setter (called with '') since "clear everything after
// Group" is exactly what fbSetResource('') already does (and likewise
// for Resource/fbSetLevel('')) — those setters never touch the level
// above them.
function fbJumpToStay(which, idx) {
  if (which === 'group')    { fbSetResource('');          return; }
  if (which === 'resource') { fbSetLevel('');              return; }
  if (which === 'level')    { fbClearSegsKeepLevel();      return; }
  if (which === 'seg')      { fbTruncateSegsKeepIdx(idx);  return; }
}

// Clicking a crumb's "x" ("delete this level"): clears this level's own
// choice AND everything after it (since descendants depend on it).
// `which` is one of 'group'/'resource'/'level'/'seg'; `idx` is only used
// for 'seg' (the segment index to delete from).
function fbJumpTo(which, idx) {
  if (which === 'group')    { fbSetRoot('');     return; }
  if (which === 'resource') { fbSetResource(''); return; }
  if (which === 'level')    { fbSetLevel('');    return; }
  if (which === 'seg') {
    _fbDraft.wiz.segs  = _fbDraft.wiz.segs.slice(0, idx);
    _fbDraft.wiz.op    = '';
    _fbDraft.wiz.value = '';
    fbRerender();
  }
}

// Breadcrumb row helper — used at every point buildWizardHTML() can emit
// HTML (including its early returns for the root/group/resource attr-
// picker rows). The persistent editing banner (Update/Cancel) is
// appended separately, after the attribute/collection picker row, so
// it stays at the bottom of the Filter Builder widget block rather
// than sitting oddly between the breadcrumbs and the picker.
function fbCrumbsWithBanner(crumbs) {
  return fbBreadcrumbHTML(crumbs);
}

// Editing banner (Update/Cancel), rendered only when mid-edit. Kept as
// a separate helper (rather than folded into fbCrumbsWithBanner) so
// callers can place it after the attribute/collection picker row.
function fbTrailingEditingBar() {
  return _fbDraft.editing ? fbEditingBar() : '';
}

function buildWizardHTML(model) {
  var wiz    = _fbDraft.wiz;
  var anchor = fbPathAnchor(model);
  var crumbs = [];

  // ---- Registry root: attrs of the root, or step into a Group type ----
  if (anchor.showGroup) {
    if (!wiz.gPlural && wiz.segs.length === 0) {
      var rootAttrs  = (model && model.attributes) || {};
      var rootShadow = fbCollectionShadowNames(
        Object.keys((model && model.groups) || {}));
      var rootAttrOpts = fbMergedAttrOptions(rootAttrs, rootShadow);
      var rootChildOpts = Object.keys((model && model.groups) || {})
        .sort().map(function(g) { return {value: 'grp:' + g, label: g}; });
      return fbCrumbsWithBanner(crumbs) + fbMergedSelectRow(
        rootAttrOpts, rootChildOpts, 'fbSetRootChoice(this.value)', 'Registry')
        + fbTrailingEditingBar();
    }
    if (wiz.gPlural) crumbs.push(fbCrumb(wiz.gPlural,
      "fbJumpToStay('group')", "fbJumpTo('group')"));
  }

  var gPlural = anchor.gPlural || wiz.gPlural;

  // ---- Group entity: attrs of the group, or step into a Resource type -
  if (gPlural && anchor.showResource) {
    var gm = model && model.groups && model.groups[gPlural];
    if (!wiz.rPlural && wiz.segs.length === 0) {
      var groupAttrs  = (gm && gm.attributes) || {};
      var groupShadow = fbCollectionShadowNames(
        Object.keys((gm && gm.resources) || {}));
      var groupAttrOpts = fbMergedAttrOptions(groupAttrs, groupShadow);
      var groupChildOpts = Object.keys((gm && gm.resources) || {})
        .sort().map(function(r) { return {value: 'res:' + r, label: r}; });
      var groupSingular = (gm && gm.singular) || gPlural.replace(/s$/, '');
      return fbCrumbsWithBanner(crumbs) + fbMergedSelectRow(
        groupAttrOpts, groupChildOpts, 'fbSetGroupChoice(this.value)',
        groupSingular) + fbTrailingEditingBar();
    }
    if (wiz.rPlural) {
      crumbs.push(fbCrumb(wiz.rPlural,
        "fbJumpToStay('resource')", "fbJumpTo('resource')"));
    }
  }

  var rPlural = anchor.rPlural || wiz.rPlural;

  // ---- Resource entity: attrs (incl. inlined version attrs), or step
  // into Meta / Versions ----
  if (gPlural && rPlural && anchor.showLevel) {
    var gm2 = model && model.groups && model.groups[gPlural];
    var rm  = gm2 && gm2.resources && gm2.resources[rPlural];
    if (!wiz.level && wiz.segs.length === 0) {
      var resAttrs = rm ? fbMergeResourceAttrs(rm) : {};
      var resAttrOpts  = fbMergedAttrOptions(resAttrs, null);
      var resChildOpts = [
        {value: 'step:meta',     label: 'meta'},
        {value: 'step:versions', label: 'versions'}
      ];
      var resSingular = (rm && rm.singular) || rPlural.replace(/s$/, '');
      return fbCrumbsWithBanner(crumbs) + fbMergedSelectRow(
        resAttrOpts, resChildOpts, 'fbSetResourceChoice(this.value)',
        resSingular) + fbTrailingEditingBar();
    }
    if (wiz.level) crumbs.push(fbCrumb(wiz.level,
      "fbJumpToStay('level')", "fbJumpTo('level')"));
  }

  var ctx      = fbRootContext(model, wiz);
  var segs     = wiz.segs;
  var runKind  = 'object';
  var runAttrs = ctx.attrsMap;
  var runItem  = null;
  var active   = '';
  // True once we've broken out of the loop at a frontier reached AFTER
  // picking at least one segment (i > 0) whose type is object/map/array
  // (not a scalar leaf) — lets the user stop drilling and filter on
  // presence/absence of that complex attribute itself (e.g.
  // `schemagroups.schema.deprecated` — "deprecated" is an object, but
  // "is present"/"is absent" is still a perfectly valid filter on it).
  var canStopHere = false;

  for (var i = 0; i <= segs.length; i++) {
    var seg = segs[i];
    if (runKind === 'object') {
      if (!runAttrs) break;
      if (!seg || !seg.text) {
        active = fbGenericSegRow(i, fbAttrOptions(runAttrs), seg, 'attr');
        canStopHere = (i > 0);
        break;
      }
      crumbs.push(fbCrumb(seg.text,
        "fbJumpToStay('seg', " + i + ")", "fbJumpTo('seg', " + i + ")"));
      var next = fbNextFrontier(runAttrs[seg.text] || null);
      runKind = next.kind; runAttrs = next.attrsMap; runItem = next.item;
    } else if (runKind === 'map') {
      if (!seg || !seg.text) {
        active = fbGenericSegRow(i, fbMapOptions(), seg, 'attr');
        canStopHere = (i > 0);
        break;
      }
      crumbs.push(fbCrumb(seg.text,
        "fbJumpToStay('seg', " + i + ")", "fbJumpTo('seg', " + i + ")"));
      var next2 = fbNextFrontier(runItem);
      runKind = next2.kind; runAttrs = next2.attrsMap; runItem = next2.item;
    } else if (runKind === 'array') {
      if (!seg || !seg.text) {
        active = fbGenericSegRow(i, fbArrayOptions(), seg, 'arr');
        canStopHere = (i > 0);
        break;
      }
      crumbs.push(fbCrumb(seg.text,
        "fbJumpToStay('seg', " + i + ")", "fbJumpTo('seg', " + i + ")"));
      var next3 = fbNextFrontier(runItem);
      runKind = next3.kind; runAttrs = next3.attrsMap; runItem = next3.item;
    } else {
      break;
    }
  }

  var html = fbBreadcrumbHTML(crumbs) + active;

  var lastSeg = segs[segs.length - 1];
  if (runKind === 'leaf' && segs.length > 0 && lastSeg && lastSeg.text) {
    html += fbOpValueRow(false);
  } else if (canStopHere) {
    html += fbOpValueRow(true);
  } else if (_fbDraft.editing) {
    // Mid-edit but navigated away from a complete leaf/op/value state
    // (e.g. deleted a breadcrumb to redefine the attribute path) —
    // fbOpValueRow() won't render at all here, so keep Update/Cancel
    // visible via the same persistent banner used in the early-return
    // cases above; Update stays disabled until a valid leaf is chosen.
    html += fbEditingBar();
  }

  return html;
}

// Applying a new filter/sort/inline/doc/binary/collections combination from
// the left nav re-fetches and re-renders the page — effectively a
// navigation away from whatever's currently in the JSON edit textarea, same
// as clicking a breadcrumb or switching view. So it needs the same
// unsaved-changes guard pushState()'s own leaving-JSON-edit check gives
// real navigations (that guard doesn't fire here since this doesn't change
// section/path/serverURL/view — it's a same-page options change — so
// without this it would silently discard an in-progress edit). See
// showLeaveEditDialog()'s usePutPatchLabels doc comment for why JSON view
// uses "Save (PUT)"/"Save (PATCH)" wording here.
function applyJSONOptions() {
  if (_state.dataView === 'json' && _state.editMode && _jsonEditDirty) {
    showLeaveEditDialog(
      function() { jsonEditSave('PUT', function() { resetJsonEditBuffer(); applyJSONOptionsReal(); }); },
      function() { resetJsonEditBuffer(); applyJSONOptionsReal(); },
      null,
      function() { jsonEditSave('PATCH', function() { resetJsonEditBuffer(); applyJSONOptionsReal(); }); },
      true
    );
    return;
  }
  // Not dirty — nothing to lose, but renderJSONEditView() only (re)seeds its
  // buffer from freshly-fetched data when _jsonEditOrigText is still null
  // (see its own doc comment). Without resetting it here first, the refetch
  // below would silently keep showing the OLD (pre-Apply) buffer contents
  // even though a new request for the new options was actually sent — the
  // same "already initialized meanwhile" guard that (correctly) prevents a
  // stale buffer from clobbering an in-progress edit would otherwise also
  // (incorrectly) prevent a clean buffer from ever picking up the new data.
  if (_state.dataView === 'json' && _state.editMode) resetJsonEditBuffer();
  applyJSONOptionsReal();
}

function applyJSONOptionsReal() {
  var doc = el('lp-doc'), bin = el('lp-bin'), col = el('lp-col');
  var cbs = qsa('.lp-inline-cb');
  var inlines = [];
  cbs.forEach(function(cb) { if (cb.checked) inlines.push(cb.value); });
  // Filters are handled by the shared applyFilters() helper (bakes the new
  // filter into a fresh apiURL) — see plan.md "Filter support in Grid/List
  // views". Sort/inline/docView/binary/collections stay JSON-view-only,
  // always freshly appended by buildAPIURL(), unchanged from before.
  var filterPatch = applyFilters();
  // Same draft-resync as applyGridFilters() — see its comment: without
  // this, Advanced (raw text) mode's textarea re-renders with its stale
  // pre-Apply content instead of what was actually just applied.
  if (_fbDraft) {
    _fbDraft.groups   = filterPatch.filters.slice();
    _fbDraft.advanced = false;
    _fbDraft.editing  = null;
  }
  // Same stale-draft guard as applyGridFilters() — see its comment.
  var newSortJ = (_sortDraft && _sortDraftKey === sortKey()) ? sortCollectValue() : _state.sort;
  if (newSortJ !== _state.sort) {
    // Keep List view's column-click override (see sortBy()) from later
    // masking a sort that was just deliberately (re)applied here.
    _sortCol = null; _sortAsc = true;
  }
  pushState(Object.assign({
    sort:        newSortJ,
    docView:     doc ? doc.checked : false,
    binary:      bin ? bin.checked : false,
    collections: col ? col.checked : false,
    inlines:     inlines
  }, filterPatch));
}

// Build model-driven inline options for JSON left panel.
// Returns array of {value, label, dotStar?} objects plus {sep:true} separators.
// Structure mirrors the old ui.go inline logic (capabilities/model at root,
// * always, then model-driven hierarchy with .* for containers).
function buildInlineOptions(model, path) {
  var opts  = [];
  var depth = path.length;
  var last  = path[path.length - 1];

  if (last === 'meta') return opts;  // no inlines on meta page

  // Registry root: offer server-level options
  if (depth === 0) {
    opts.push({value: 'capabilities', label: 'capabilities'});
    opts.push({value: 'model',        label: 'model'});
    opts.push({value: 'modelsource',  label: 'modelsource'});
    // Separator between the config-level (server) options above and the
    // user/data-level options below, matching the old UI's layout.
    opts.push({sep: true});
  }

  // * (all) — always available, in normal list flow
  opts.push({value: '*', label: '* (all)'});

  if (!model) return opts;

  function getRM(gPlural, rPlural) {
    var gm = model.groups && model.groups[gPlural];
    return gm && gm.resources && gm.resources[rPlural];
  }

  // Add resource-level inlines (meta, versions, optional doc) with a path prefix
  function addResInlines(gPlural, rPlural, prefix) {
    var rm = getRM(gPlural, rPlural);
    if (!rm) return;
    var rSing = rm.singular || rPlural.replace(/s$/, '');
    if (rm.hasdocument) opts.push({value: prefix + rSing,      label: prefix + rSing});
    opts.push(           {value: prefix + 'meta',              label: prefix + 'meta'});
    opts.push(           {value: prefix + 'versions',          label: prefix + 'versions',         dotStar: true});
    opts.push(           {value: prefix + 'versions.' + rSing, label: prefix + 'versions.' + rSing});
  }

  if (depth === 0) {
    // Registry root: all groups and their resources
    Object.keys(model.groups || {}).sort().forEach(function(gPlural) {
      var gm = model.groups[gPlural];
      opts.push({value: gPlural, label: gPlural, dotStar: true});
      Object.keys(gm.resources || {}).sort().forEach(function(rPlural) {
        opts.push({value: gPlural + '.' + rPlural, label: gPlural + '.' + rPlural, dotStar: true});
        addResInlines(gPlural, rPlural, gPlural + '.' + rPlural + '.');
      });
    });
  } else if (depth <= 2) {
    // Group collection (depth 1) or group entity (depth 2)
    var gPlural = path[0];
    var gm = model.groups && model.groups[gPlural];
    if (gm) {
      Object.keys(gm.resources || {}).sort().forEach(function(rPlural) {
        opts.push({value: rPlural, label: rPlural, dotStar: true});
        addResInlines(gPlural, rPlural, rPlural + '.');
      });
    }
  } else if (depth <= 4) {
    // Resource collection (depth 3) or resource entity (depth 4)
    addResInlines(path[0], path[2], '');
  } else if (last === 'versions' || depth === 6) {
    // Version collection or version entity
    var rm = getRM(path[0], path[2]);
    if (rm) opts.push({value: rm.singular, label: rm.singular});
  }

  return opts;
}

// Extracts the `filter=` OR-groups from a raw URL string's query
// string, in the app's own internal representation (array of one
// string per OR-group). The server emits one repeated `filter=` query
// param per OR-group (see FiltersRelativeToAbstract in info.go) — a
// DIFFERENT wire convention than the app's own permalink format (single
// `filter=` param, OR-groups newline-joined — see buildURL/
// loadStateFromURL). Used when linkifying a same-server JSON URL value
// (e.g. a nested collection's `xxxurl`): the server has already
// computed the correctly-subsetted filter expression for that specific
// URL, so it must be reused verbatim rather than re-derived from the
// UI's own currently-active filter state (which reflects the CURRENT
// level, not the target one). Returns [] if the URL has no filter
// param at all — i.e. the target URL is unfiltered.
// Keep _state.filters in sync with whatever the current real apiURL
// actually carries, in this one central place — called from both
// pushStateReal() (ordinary navigation, breadcrumb click, applyFilters())
// and loadStateFromURL() (fresh load / bookmark / popstate), so a bookmarked
// deep link with a filter baked into `apiurl=` shows the same filter builder
// state as arriving there by clicking through the UI. No-op when apiURL is
// falsy (leaves filters exactly as already set) — every same-server
// real-link navigation (navigateJsonUrl(), entity row/link clicks, etc.)
// always sets apiURL, so this covers them all uniformly with no separate
// per-call-site filter extraction needed. See plan.md "Filter support in
// Grid/List views" for the full rationale. (Entity `self` links get the
// currently-active filter appended by entityHrefWithFilter() before we
// ever navigate to them, so this plain apiURL-derived sync is enough
// everywhere — no seed/reconstruction needed.)
function syncFiltersFromApiURL() {
  if (_state.section !== 'data' || !_state.apiURL) return;
  _state.filters = filtersFromUrl(_state.apiURL);
}

function filtersFromUrl(rawUrl) {
  var qIdx = rawUrl.indexOf('?');
  if (qIdx < 0) return [];
  var groups = [];
  new URLSearchParams(rawUrl.slice(qIdx + 1)).forEach(function(v, k) {
    if (k === 'filter') groups.push(v);
  });
  return groups;
}

function navigateJsonUrl(encodedUrl) {
  // encodedUrl is HTML-attribute-encoded; decode it back to a real URL
  var raw = encodedUrl.replace(/&amp;/g,'&').replace(/&lt;/g,'<').replace(/&gt;/g,'>').replace(/&quot;/g,'"');
  var svBase = serverBase();
  var urlPath = raw.split('?')[0].split('#')[0];   // strip query + fragment
  urlPath = urlPath.replace(/\/?\$details$/, '');   // strip $details suffix
  if (urlPath.indexOf(svBase) === 0) {
    var rel      = urlPath.slice(svBase.length).replace(/^\//, '');
    var segments = rel ? rel.split('/') : [];
    // Pass the clicked link through verbatim as apiURL (like every other
    // real-link navigation call site in this file) instead of only
    // extracting its filter= and letting buildAPIURL() reconstruct the
    // rest from generic state — that silently dropped any other query
    // param the server-provided link carried. pushStateReal() already
    // calls syncFiltersFromApiURL() right after this, which re-derives
    // _state.filters from this same apiURL for the Filter/Sort panel, so
    // no separate filtersFromUrl() call is needed here anymore.
    pushState({
      view: 'table', section: 'data', path: segments, apiURL: raw
    });
  } else {
    window.open(raw, '_blank', 'noopener');
  }
  return false;
}

function syntaxHighlight(str) {
  var svBase = serverBase();
  return str
    .replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
    .replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
      function(m) {
        var c = /^"/.test(m) ? (/:$/.test(m) ? 'json-key' : 'json-str')
              : /true|false/.test(m) ? 'json-bool' : /null/.test(m) ? 'json-null' : 'json-num';

        // Linkify URL string values (not keys)
        if (c === 'json-str') {
          // Strip outer quotes and unescape HTML entities to get raw content
          var inner = m.slice(1, -1).replace(/&amp;/g,'&').replace(/&lt;/g,'<').replace(/&gt;/g,'>');
          if (/^https?:\/\//.test(inner)) {
            var urlPath = inner.split('?')[0].split('#')[0].replace(/\/?\$details$/, '');
            var href, target = '', onclick, displayM = m;
            if (urlPath.indexOf(svBase) === 0) {
              // Same-server: build SPA href for right-click "open in new
              // tab" support. Set apiURL to the link's own raw value
              // (verbatim, all query params included — not just filter=)
              // so buildURL() encodes the exact target URL as `apiurl=`,
              // matching what navigateJsonUrl() will actually navigate to
              // on click. filters is still derived here (rather than left
              // to sync automatically, as navigateJsonUrl()'s real
              // pushStateReal() call does) because fakeSt is a synthetic,
              // one-off object used only to compute this display href —
              // it never goes through pushStateReal()/
              // syncFiltersFromApiURL() — and buildURL() needs filters to
              // already match apiURL's embedded filter to avoid appending
              // a redundant, stale top-level filter= on top of it.
              var rel      = urlPath.slice(svBase.length).replace(/^\//, '');
              var segments = rel ? rel.split('/') : [];
              var fakeSt   = Object.assign({}, _state, {
                view: 'table', section: 'data', path: segments,
                apiURL: inner, filters: filtersFromUrl(inner)
              });
              href    = buildURL(fakeSt);
              // guardedOnclick() lets ctrl/cmd/shift/middle-click fall
              // through to the browser's native "open in new tab" on the
              // real href above, instead of always intercepting the click.
              var navExpr = 'navigateJsonUrl(\'' + inner.replace(/\\/g,'\\\\').replace(/'/g,"\\'") + '\')';
              onclick = guardedOnclick(navExpr);
              // When browsing through the proxy, the underlying data (and
              // thus `inner`) contains our local /xrproxy/<base64> URL —
              // show the real remote URL instead (display text only; href/
              // onclick above must keep using the proxy URL so clicking
              // still navigates correctly).
              var realInner = toRealURL(inner);
              if (realInner !== inner) {
                displayM = '"' + realInner.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;') + '"';
              }
            } else {
              href    = inner;
              target  = ' target="_blank" rel="noopener"';
              onclick = '';
            }
            var attrOnclick = onclick ? ' onclick="' + esc(onclick) + '"' : '';
            return '<span class="' + c + '"><a class="json-url" href="' + esc(href) + '"' + target + attrOnclick + '>' + displayM + '</a></span>';
          }
        }

        return '<span class="' + c + '">' + m + '</span>';
      });
}

// ---- Collection helpers --------------------------------------------------

// Extract entity items from a collection response.
// A collection response is a flat JSON object keyed by <singular>id.
// Pagination metadata lives in the Link response header (not in the body).
// preserveOrder skips the default ID-alphabetical sort, keeping items in
// the order the server/data object returned them — used by List view when
// an active server-side `sort=` should be visibly honored (see
// activeServerSortLabel()/renderTableView()). All other callers (e.g. the
// Resource page's version dropdown) omit it and get the usual ID sort.
function collectionItems(data, preserveOrder) {
  if (!data || typeof data !== 'object') return [];
  var items = [];
  Object.keys(data).forEach(function(k) {
    var v = data[k];
    if (typeof v === 'object' && v !== null && !Array.isArray(v)) {
      // Attach the map key in case xid is absent (shouldn't happen per spec but be safe)
      items.push(Object.assign({__mapKey: k}, v));
    }
  });
  if (!preserveOrder) {
    items.sort(function(a, b) { return itemNavKey(a).localeCompare(itemNavKey(b)); });
  }
  return items;
}

// The navigation key for an item — the map key it's stored under in the
// collection response, which per spec IS the entity's own <singular>id (no
// parsing needed at all — see collectionItems()'s __mapKey). Falls back to
// splitting xid's last segment only defensively, for the rare case an item
// wasn't obtained via collectionItems() and so has no __mapKey attached.
function itemNavKey(item) {
  if (item.__mapKey) return item.__mapKey;
  if (item.xid) {
    var segs = item.xid.replace(/^\//, '').split('/');
    return segs[segs.length - 1] || '';
  }
  return '';
}

// Find navigable sub-collections using the model definition.
// model: the registry model object (may be null — falls back to scanning data)
// path: current navigation path array ([] = registry root, [G,gId] = group instance, etc.)
// data: the entity JSON object
// Returns [{plural, count, url}]
function findCollectionRefs(model, path, data) {
  if (!data || typeof data !== 'object') return [];
  var plurals = [];
  var depth = path ? path.length : 0;

  if (model && model.groups) {
    if (depth === 0) {
      // Registry root — collections are group types
      plurals = Object.keys(model.groups);
    } else if (depth === 2) {
      // Group instance — collections are resource types
      var grpDef = model.groups[path[0]];
      if (grpDef && grpDef.resources) plurals = Object.keys(grpDef.resources);
    }
    // depth 4+ (resource instance) has no sub-collections in xRegistry
  }

  // Fallback: scan data for *url/*count pairs (model unavailable)
  if (plurals.length === 0) {
    Object.keys(data).forEach(function(k) {
      if (k.endsWith('url') && data[k.slice(0, -3) + 'count'] !== undefined)
        plurals.push(k.slice(0, -3));
    });
  }

  var result = [];
  plurals.forEach(function(p) {
    var urlVal   = data[p + 'url'];
    var countVal = data[p + 'count'];
    if (urlVal !== undefined || countVal !== undefined) {
      result.push({plural: p, count: countVal !== undefined ? countVal : 0, url: urlVal || ''});
    }
  });
  result.sort(function(a, b) { return a.plural.localeCompare(b.plural); });
  return result;
}

function deriveColumns(items, collKeySet) {
  // Prefer xid first (shows navigable id), then common fields
  var priority = ['xid','name','description','epoch','createdat','modifiedat',
    'versionid','isdefault','ancestorid','contenttype'];
  var seen = {}, cols = [];
  priority.forEach(function(c) {
    if (items.some(function(it) { return it[c] !== undefined; })) {
      seen[c] = true; cols.push(c);
    }
  });
  var skip = collKeySet || {};
  items.forEach(function(item) {
    Object.keys(item).forEach(function(k) {
      if (!seen[k] && !k.startsWith('__') && !skip[k]) {
        var v = item[k];
        if (typeof v !== 'object' || v === null) { seen[k] = true; cols.push(k); }
      }
    });
  });
  return cols.slice(0, 8);
}

// ---- Navigate ------------------------------------------------------------
//
// Link-driven: every call site below passes the REAL server-provided URL for
// the destination (self / <plural>url / versionsurl / etc.) — see the
// "Link-driven navigation" notes near buildAPIURL(). `url` is optional only
// for backward callers; when omitted, buildAPIURL()'s buildBaseURL()
// fallback silently reconstructs from `path` (accepted only where no real
// link exists at all — see versionURLById()).
//
// Note: an entity's own `self` link never carries filter context on its
// own — the server only rescopes a `filter=` param onto nested-collection
// links it's actually asked to compute (confirmed: `GET self?filter=...`
// DOES work and rescopes correctly, same as any collection URL). So when
// navigating INTO an entity from a filtered collection, callers append the
// collection's currently-active filter onto its `self` link themselves
// (see `entityHrefWithFilter()`) rather than us trying to avoid the
// refetch — this keeps hover/ctrl-click/normal-click all showing and using
// the exact same real URL, and refresh() always does a plain GET.

function navigateTo(id, url) {
  // If navigating INTO a collection from the registry root or single entity,
  // the id IS the collection name (e.g., "endpoints") and we just append it.
  pushState({path: _state.path.concat([id]), apiURL: url || ''});
}

// Navigate directly into a nested collection shown as a resource-pill on a
// collection view's tile/row (e.g. clicking "files (2)" on group "d1" while
// viewing the "dirs" collection takes you straight to dirs/d1/files, instead
// of first landing on the d1 entity page). itemId is the clicked row/tile's
// own id; plural is the nested collection's name; url is its real <plural>url.
function navigateToNestedColl(itemId, plural, url) {
  pushState({path: _state.path.concat([itemId, plural]), apiURL: url || ''});
}

// Navigate straight to a resource's default version (the "Default Version: X"
// pill shown on a resource tile/row in the resources collection view).
// itemId is the resource's own id; versionId is its default version's id;
// url is that version's real self link (see defaultVersionURL()).
function navigateToDefaultVersion(itemId, versionId, url) {
  pushState({path: _state.path.concat([itemId, 'versions', String(versionId)]), apiURL: url || ''});
}

// Appends a path segment (e.g. a version id) to a collection URL that may
// already carry a query string (most commonly `?filter=...`, baked in by
// the server or by our own applyFilters()) — a plain `url + '/' + seg`
// would land the new segment AFTER the query string (e.g.
// `.../versions?filter=epoch>0/v1`, badly malformed) instead of extending
// the actual path. Splits off any query string first, appends the segment
// to the path portion only, then re-attaches the query string unchanged.
function appendURLPathSegment(url, seg) {
  var qIdx = url.indexOf('?');
  var path = qIdx >= 0 ? url.slice(0, qIdx) : url;
  var query = qIdx >= 0 ? url.slice(qIdx) : '';
  return path.replace(/\/$/, '') + '/' + encodeURIComponent(seg) + query;
}

// Resolve the real URL for a resource item's default version, shown as the
// "Default Version: X" pill on a resource tile/row in the resources
// collection view. item is one entry from collectionItems(data); colls is
// that item's own findCollectionRefs() result (used to get its versionsurl).
// Prefers an actual link over any construction, same priority as
// versionURLById(): item.defaultversionurl (if present and matching), then
// item's own versionsurl + versionid, then plain path construction.
function defaultVersionURL(item, itemPath, colls) {
  if (item.defaultversionurl && (item.defaultversionid === undefined || String(item.defaultversionid) === String(item.versionid))) {
    return item.defaultversionurl;
  }
  var versionsColl = (colls || []).filter(function(c) { return c.plural === 'versions'; })[0];
  if (versionsColl && versionsColl.url) {
    return appendURLPathSegment(versionsColl.url, item.versionid);
  }
  return serverBase() + '/' + itemPath.concat(['versions', String(item.versionid)]).join('/');
}

// Resolve the real URL for a sibling version ("→ Visit" buttons next to
// defaultversionid/versionid/ancestorid), preferring an actual link over any
// construction, in this order:
//   1. An exact link the server already gave us for THIS id (defaultversionurl,
//      when it happens to match) — zero construction.
//   2. The resource's own real versionsurl + the known versionid — appending
//      a spec-guaranteed <singular>id to its own collection's link (a much
//      narrower exception than reconstructing the whole path).
//   3. A same-session cached versions-collection URL (crumbURLs), if we
//      reached here by drilling through the versions list already.
//   4. Last resort: plain path construction — needed because version
//      entities don't carry a versionsurl/parent link of their own (a
//      discovered spec gap — see plan.md, "Link-driven navigation").
function versionURLById(vid) {
  var d = _lastData;
  if (d && d.defaultversionid !== undefined && String(d.defaultversionid) === String(vid) && d.defaultversionurl) {
    return d.defaultversionurl;
  }
  if (d && d.versionsurl) {
    return appendURLPathSegment(d.versionsurl, vid);
  }
  var vColl = _state.crumbURLs && _state.crumbURLs[4]; // depth 5 = ".../versions" collection
  if (vColl) return appendURLPathSegment(vColl, vid);
  var basePath = _state.path.slice(0, 4);
  return serverBase() + '/' + basePath.concat(['versions', vid]).join('/');
}


// Navigate to a specific version from the meta page (path: [..., resource, rId, "meta"])
function navigateToVersion(vId) {
  var basePath = _state.path.slice(0, -1); // strip "meta"
  pushState({path: basePath.concat(['versions', vId]), apiURL: versionURLById(vId)});
}

// Navigate to a version by ID from the current resource or version context
function navigateToVersionById(vId) {
  var basePath = _state.path.slice(0, 4); // [G, gId, R, rId]
  pushState({path: basePath.concat(['versions', vId]), apiURL: versionURLById(vId)});
}

// ---- Model Editor (ported from registry/ui.go's ?html model editor) ------
//
// Renders a full browse+edit UI for a registry's model/modelsource data,
// reusing the same drill-down/forms engine as the legacy server-rendered
// editor. Entry point is renderModelEditor(data), called from refresh() when
// _state.section is 'model' or 'modelsource' and _state.dataView !== 'json'.
//
// Read-only vs editable is driven by _state.editMode (the shared header
// pencil button); editing is only ever enabled while on 'modelsource'
// (see renderHeader()) — saves always PUT to /modelsource.

var _modelPutURL    = '';
var _modelMutable   = false;
var _modelReadOnly  = true;  // runtime: true unless _state.editMode && section === modelsource
var _modelSrc       = null;  // pristine copy of last-loaded model (for undo)
var _modelData      = null;  // working copy being edited/viewed
var _modelDirty     = false;
var _modelLoadedFor = null;  // "serverURL|section" key used to detect a fresh load
var _navTab         = 'registry';
var _navPath        = [];
var _navSelected    = null;
var _attrNestStack  = []; // [{key,isItem}] — nested attr drilldown beyond _navPath
var _cstrCounter    = 0;  // unique ID counter for constraint enum containers

// Entry point called from refresh().
function renderModelEditor(data) {
  var main = el('main-view');
  var key = normalizeURL(_state.serverURL || window.location.origin) + '|' + _state.section;
  if (_modelLoadedFor !== key) {
    _modelSrc  = deepClone(data);
    _modelData = deepClone(_modelSrc);
    _modelDirty = false;
    _navTab = 'registry'; _navPath = []; _navSelected = null; _attrNestStack = [];
    _modelLoadedFor = key;
  }
  _modelMutable  = _state.mutable;
  _modelPutURL   = buildAPIURL();
  _modelReadOnly = !(_state.editMode && _state.section === 'modelsource');
  main.innerHTML = '<div id="modelEditor"></div>';
  renderEditor();
}


function deepClone(o) { return JSON.parse(JSON.stringify(o)) ; }
function markDirty() {
  if (!_modelDirty) {
    _modelDirty = true ;
    var sb = document.getElementById('saveBtn') ; if (sb) sb.disabled = false ;
    var ub = document.getElementById('undoBtn') ; if (ub) ub.disabled = false ;
  }
}

window.addEventListener('beforeunload', function(e) {
  if (_modelDirty || _capDirty || _dataDirty) { e.preventDefault() ; e.returnValue = '' ; }
}) ;

// Shows the "unsaved changes" dialog with Cancel / Discard / Save button(s).
// `onSaveFull` is the sole Save action for editors that only ever do a
// full-document PUT (Model Source, Capabilities, and any other single-verb
// editor) — rendered as a plain "Save" button. Callers whose editor also
// supports a partial PATCH (Data entity, Meta, Version, JSON view's raw
// edit) additionally pass `onSaveDelta`, which renders two buttons instead
// — "Save (full)" (onSaveFull) and "Save (delta)" (onSaveDelta) — so the
// user can pick PUT vs PATCH right here instead of being forced back to
// the page to choose. Labels mirror the entity action bar's own Save
// buttons (see buildDataEditorActionBarHtml() etc.) for consistency.
// `usePutPatchLabels` swaps those two labels for "Save (PUT)"/"Save
// (PATCH)" instead — used only by JSON view's raw-edit callers, whose own
// action-bar buttons already say PUT/PATCH (technical users editing raw
// JSON directly benefit from the literal HTTP verb; the Data/Meta/Version
// Property-table editors keep the friendlier full/delta wording).
function showLeaveEditDialog(onSaveFull, onDiscard, onCancel, onSaveDelta, usePutPatchLabels) {
  var overlay = document.createElement('div') ;
  overlay.style.cssText = 'position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.35);z-index:9999;display:flex;align-items:center;justify-content:center;' ;
  var box = document.createElement('div') ;
  box.style.cssText = 'background:white;border-radius:8px;padding:24px;box-shadow:0 4px 24px rgba(0,0,0,0.25);max-width:340px;width:90%;font-family:sans-serif;' ;
  var msg = document.createElement('p') ; msg.textContent = 'You have unsaved changes.' ;
  msg.style.cssText = 'margin:0 0 20px;font-size:14px;color:#333;' ;
  box.appendChild(msg) ;
  var btns = document.createElement('div') ; btns.style.cssText = 'display:flex;gap:8px;justify-content:flex-end;flex-wrap:wrap;' ;
  function mkBtn(label, fn, css) {
    var b = document.createElement('button') ; b.textContent = label ;
    b.style.cssText = 'padding:6px 16px;border-radius:5px;cursor:pointer;font-size:13px;font-weight:bold;' + css ;
    b.onclick = function() { document.body.removeChild(overlay) ; fn() ; } ;
    btns.appendChild(b) ;
  }
  mkBtn('Cancel',  function(){ if (onCancel) onCancel(); },  'background:#f0f0f0;color:#333;border:1px solid #ccc;') ;
  mkBtn('Discard', onDiscard,     'background:#f8d7da;color:#721c24;border:1px solid #f5c6cb;') ;
  if (onSaveDelta) {
    var fullLabel  = usePutPatchLabels ? 'Save (PUT)'   : 'Save (full)';
    var deltaLabel = usePutPatchLabels ? 'Save (PATCH)' : 'Save (delta)';
    mkBtn(fullLabel,  onSaveFull,  'background:#2060a0;color:white;border:1px solid #2060a0;') ;
    mkBtn(deltaLabel, onSaveDelta, 'background:#2060a0;color:white;border:1px solid #2060a0;') ;
  } else {
    mkBtn('Save', onSaveFull, 'background:#2060a0;color:white;border:1px solid #2060a0;') ;
  }
  box.appendChild(btns) ; overlay.appendChild(box) ; document.body.appendChild(overlay) ;
}



// ---- Navigation primitives ----

function drillDown(path) {
  var beforePath = _navPath.slice() ;
  collectCurrentEditor() ;
  _attrNestStack = [] ;
  // Fix up stale path segments in case collectCurrentEditor renamed a group/resource key
  _navPath = path.map(function(seg, i) {
    if (i < beforePath.length && beforePath[i] === seg && _navPath[i] && _navPath[i] !== seg)
      return _navPath[i] ;
    return seg ;
  }) ;
  _navSelected = null ;
  renderEditor() ;
}

function selectItem(key) {
  collectCurrentEditor() ;
  _navSelected = key ;
  renderEditor() ;
}

function changeTab(tab) {
  collectCurrentEditor() ;
  _attrNestStack = [] ;
  _navTab = tab ; _navPath = [] ; _navSelected = null ;
  renderEditor() ;
}

// ---- Attr nesting helpers ----

// Returns the base attributes map (or item parent) at the current _navPath level.
function getBaseAttrsObj() {
  var m = _modelData || {} ;
  if (_navTab === 'registry') {
    if (!m.attributes) m.attributes = {} ;
    return m.attributes ;
  }
  var gk = _navPath[0] ; if (!gk) return {} ;
  if (!m.groups) m.groups = {} ;
  var grp = m.groups[gk] ; if (!grp) return {} ;
  if (_navPath.length === 2) {
    var sec = _navPath[1] ;
    if (sec === 'attributes') { if (!grp.attributes) grp.attributes = {} ; return grp.attributes ; }
  }
  if (_navPath.length === 4) {
    var rk = _navPath[2], attrSec = _navPath[3] ;
    var dataKey = attrSec === 'versionattributes' ? 'attributes' : attrSec ;
    if (!grp.resources) grp.resources = {} ;
    var res = grp.resources[rk] ; if (!res) return {} ;
    if (!res[dataKey]) res[dataKey] = {} ;
    return res[dataKey] ;
  }
  return {} ;
}

// Traverses _attrNestStack from the base attrs object.
// Returns {attrsObj, parentAttr, isItem, ifvMap} where:
//   isItem:false, ifvMap:null → attrsObj is the attrs map to show/edit
//   isItem:true              → parentAttr is the item object (map/array)
//   ifvMap:non-null          → currently viewing ifvalues key list
// If createMissing=true, creates intermediate structures as needed.
function resolveAttrNesting(createMissing) {
  var cur = getBaseAttrsObj() ;
  var curParent = null ; // last resolved attrObj from isItem:true, for __item__:isItem:true chaining
  var ifvMap = null ;
  for (var i = 0; i < _attrNestStack.length; i++) {
    var entry = _attrNestStack[i] ;
    if (entry.key === '__item__' && !entry.isItem) { continue ; } // sentinel: inside item.attributes
    if (entry.key === '__item__' && entry.isItem) {
      // Descend into curParent.item.item (map/array item chain)
      if (!curParent) return {attrsObj:{}, parentAttr:null, isItem:true, ifvMap:null} ;
      var prevItem = curParent.item ;
      if (!prevItem) {
        if (createMissing) { curParent.item = {} ; prevItem = curParent.item ; }
        else return {attrsObj:{}, parentAttr:null, isItem:true, ifvMap:null} ;
      }
      curParent = prevItem ; // curParent.item is now the next item to render
      if (i === _attrNestStack.length - 1) return {attrsObj:{}, parentAttr:curParent, isItem:true, ifvMap:null} ;
      continue ;
    }
    if (entry.isIfValues) {
      var attrObj = cur[entry.key] ;
      if (!attrObj) {
        if (createMissing) { cur[entry.key] = {} ; attrObj = cur[entry.key] ; }
        else return {attrsObj:{}, parentAttr:null, isItem:false, ifvMap:null} ;
      }
      if (!attrObj.ifvalues) {
        if (createMissing) attrObj.ifvalues = {} ; else attrObj.ifvalues = {} ;
      }
      ifvMap = attrObj.ifvalues ;
      if (i === _attrNestStack.length - 1) return {attrsObj:{}, parentAttr:null, isItem:false, ifvMap:ifvMap} ;
    } else if (entry.isSiblings) {
      if (!ifvMap) return {attrsObj:{}, parentAttr:null, isItem:false, ifvMap:null} ;
      var ifval = ifvMap[entry.key] ;
      if (!ifval) {
        if (createMissing) { ifvMap[entry.key] = {} ; ifval = ifvMap[entry.key] ; }
        else return {attrsObj:{}, parentAttr:null, isItem:false, ifvMap:null} ;
      }
      if (!ifval.siblingattributes) {
        if (createMissing) ifval.siblingattributes = {} ;
        else return {attrsObj:{}, parentAttr:null, isItem:false, ifvMap:null} ;
      }
      cur = ifval.siblingattributes ; ifvMap = null ;
    } else if (entry.isItem) {
      var attrObj = cur[entry.key] ;
      if (!attrObj) {
        if (createMissing) { cur[entry.key] = {} ; attrObj = cur[entry.key] ; }
        else return {attrsObj:{}, parentAttr:{}, isItem:false, ifvMap:null} ;
      }
      if (!attrObj.item) {
        if (createMissing) attrObj.item = {} ;
        else return {attrsObj:{}, parentAttr:{}, isItem:true, ifvMap:null} ;
      }
      curParent = attrObj ; // save for potential __item__:isItem:true chaining
      if (i === _attrNestStack.length - 1) return {attrsObj:{}, parentAttr:attrObj, isItem:true, ifvMap:null} ;
      // Look ahead: if next is __item__:isItem:false (object sub-attrs sentinel), advance cur
      var nextEntry = _attrNestStack[i+1] ;
      if (nextEntry && nextEntry.key === '__item__' && !nextEntry.isItem) {
        var itm = attrObj.item ;
        if (!itm.attributes) { if (createMissing) itm.attributes = {} ; else return {attrsObj:{},parentAttr:{},isItem:false,ifvMap:null} ; }
        cur = itm.attributes ;
      }
      // If next is __item__:isItem:true, curParent is set and will be handled above
    } else {
      var attrObj = cur[entry.key] ;
      if (!attrObj) {
        if (createMissing) { cur[entry.key] = {} ; attrObj = cur[entry.key] ; }
        else return {attrsObj:{}, parentAttr:{}, isItem:false, ifvMap:null} ;
      }
      if (!attrObj.attributes) {
        if (createMissing) attrObj.attributes = {} ;
        else return {attrsObj:{}, parentAttr:{}, isItem:false, ifvMap:null} ;
      }
      cur = attrObj.attributes ;
    }
  }
  return {attrsObj:cur, parentAttr:null, isItem:false, ifvMap:ifvMap} ;
}

// Drills into a nested attribute level.
function drillIntoAttr(attrKey, isItem) {
  collectCurrentEditor() ;
  // _navSelected may have been updated by collectCurrentEditor (rename) — use it
  var resolvedKey = _navSelected || attrKey ;
  var attrType = null ;
  if (isItem) {
    var ctx0 = resolveAttrNesting(false) ;
    attrType = ctx0.attrsObj && ctx0.attrsObj[resolvedKey] ? (ctx0.attrsObj[resolvedKey].type || 'map') : 'map' ;
  }
  _attrNestStack.push({key:resolvedKey, isItem:isItem, attrType:attrType}) ;
  _navSelected = isItem ? '__item__' : null ;
  renderEditor() ;
}

// Pops _attrNestStack back to depth d (0 = fully exit nesting).
function popAttrNestTo(d) {
  collectCurrentEditor() ;
  _attrNestStack = _attrNestStack.slice(0, d) ;
  _navSelected = null ;
  renderEditor() ;
}

// ---- If Values helpers ----

function addNewIfValue() {
  collectCurrentEditor() ;
  var ctx = resolveAttrNesting(true) ;
  var ifv = ctx.ifvMap ; if (!ifv) return ;
  var k = uniqueKey(ifv, 'value') ;
  ifv[k] = {siblingattributes:{}} ;
  markDirty() ; _navSelected = k ; renderEditor() ;
}

function deleteIfValue(k) {
  var ctx = resolveAttrNesting(false) ;
  if (ctx.ifvMap) delete ctx.ifvMap[k] ;
  markDirty() ; if (_navSelected === k) _navSelected = null ; renderEditor() ;
}

function drillIntoIfValueSiblings() {
  collectCurrentEditor() ;
  var resolvedKey = _navSelected ;
  _attrNestStack.push({key:resolvedKey, isSiblings:true}) ;
  _navSelected = null ;
  renderEditor() ;
}

function renderIfValueForm(div, valueKey, ifvMap) {
  var titleEl = document.createElement('div') ; titleEl.className = 'editorFormTitle' ;
  titleEl.textContent = 'If Value: ' + valueKey ; div.appendChild(titleEl) ;
  var origInp = document.createElement('input') ; origInp.type = 'hidden' ;
  origInp.id = 'ef_ifvalue_orig' ; origInp.value = valueKey ; div.appendChild(origInp) ;
  var keyRow = ef('ef_ifvalue_key', 'Value', valueKey, true) ;
  var keyInp = keyRow.querySelector('input') ;
  keyInp.oninput = function() {
    var v = keyInp.value.trim() || '\u2026' ;
    titleEl.textContent = 'If Value: ' + v ;
    var navEl = document.querySelector('.navItemSelected') ;
    if (navEl) { var sp = navEl.firstChild ; if (sp) sp.textContent = v ; }
  } ;
  div.appendChild(keyRow) ;
  var sibCount = Object.keys(((ifvMap[valueKey]||{}).siblingattributes)||{}).length ;
  var drilledBtnRow = document.createElement('div') ; drilledBtnRow.className = 'editorField' ;
  drilledBtnRow.style.marginTop = '8px' ;
  var spacer = document.createElement('label') ; spacer.style.visibility = 'hidden' ;
  drilledBtnRow.appendChild(spacer) ;
  var drilledBtn = document.createElement('button') ; drilledBtn.className = 'editorBtn navDrillBtn' ;
  drilledBtn.style.cssText = 'font-size:11px;padding:3px 8px;' ;
  drilledBtn.textContent = '\u25b6 Edit Sibling Attributes' + (sibCount ? ' ('+sibCount+')' : '') ;
  drilledBtn.onclick = drillIntoIfValueSiblings ;
  drilledBtnRow.appendChild(drilledBtn) ; div.appendChild(drilledBtnRow) ;
}

function saveIfValueFrom(ifvMap, origKey) {
  if (!ifvMap) return ;
  var keyEl = document.getElementById('ef_ifvalue_key') ; if (!keyEl) return ;
  var newKey = keyEl.value.trim() || origKey ;
  var existing = ifvMap[origKey] || {siblingattributes:{}} ;
  if (newKey !== origKey) delete ifvMap[origKey] ;
  ifvMap[newKey] = existing ;
  if (_navSelected === origKey) _navSelected = newKey ;
}

// ---- Main render ----

function renderEditor() {
  var div = document.getElementById('modelEditor') ;
  // Rescue #expandAll from the old breadcrumb before wiping innerHTML
  var exAll = document.getElementById('expandAll') ;
  var myOut = document.getElementById('myOutput') ;
  if (exAll && div.contains(exAll) && myOut) {
    exAll.style.position = '' ; exAll.style.marginLeft = '' ;
    myOut.insertBefore(exAll, myOut.firstChild) ;
  }
  div.innerHTML = '' ;

  // Action bar
  var bar = document.createElement('div') ;
  bar.className = 'editorActionBar' ;
  if (!_modelReadOnly) {
    var sb = document.createElement('button') ;
    sb.className = 'editorBtn' ; sb.id = 'saveBtn' ;
    sb.textContent = 'Save' ; sb.onclick = function() { saveModel(function() { toggleEdit() ; }) ; } ; sb.disabled = !_modelDirty ;
    bar.appendChild(sb) ;
    var ub = document.createElement('button') ;
    ub.className = 'editorBtn' ; ub.id = 'undoBtn' ;
    ub.textContent = 'Undo' ; ub.onclick = undoModel ; ub.disabled = !_modelDirty ;
    bar.appendChild(ub) ;
  } else {
    // No buttons — collapse the bar completely
    bar.style.cssText = 'padding:0;border:none;margin:0;height:0;' ;
  }
  div.appendChild(bar) ;

  if (!_modelReadOnly) {
    var errDiv = document.createElement('div') ;
    errDiv.id = 'editorError' ; errDiv.className = 'error-banner' ; errDiv.style.display = 'none' ;
    div.appendChild(errDiv) ;
  }

  // Auto-select 'fields' when entering registry root or group/resource level with nothing selected
  if (_navSelected === null) {
    if (_navTab === 'registry' && _navPath.length === 0) _navSelected = 'fields' ;
    else if (_navTab === 'groups' && (_navPath.length === 1 || _navPath.length === 3)) _navSelected = 'fields' ;
  }

  // Server URL line (no page title on this view, so show it just below
  // the header's breadcrumb bar, before the sub-breadcrumb ("Registry...")
  // bar below — consistent with the other pages).
  div.insertAdjacentHTML('beforeend', serverURLLineHtml()) ;

  // Breadcrumb (replaces tab bar)
  var bc = buildBreadcrumb() ;
  // Mobile nav toggle button — insert before breadcrumb content
  var toggleBtn = document.createElement('button') ;
  toggleBtn.className = 'navToggleBtn' ; toggleBtn.type = 'button' ;
  toggleBtn.textContent = '\u2630' ; toggleBtn.title = 'Show navigation' ;
  bc.insertBefore(toggleBtn, bc.firstChild) ;
  // Move the view-toggle buttons into the breadcrumb (right-aligned)
  var exAll = document.getElementById('expandAll') ;
  if (exAll) { exAll.style.position = 'static' ; exAll.style.marginLeft = 'auto' ; bc.appendChild(exAll) ; }
  div.appendChild(bc) ;

  // Body: left nav + right panel
  var body = document.createElement('div') ; body.className = 'editorBody' ;
  var lnav = document.createElement('div') ; lnav.className = 'editorLeftNav' ;
  buildLeftNav(lnav) ;
  var rpanel = document.createElement('div') ; rpanel.className = 'editorRightPanel' ;
  buildRightPanel(rpanel) ;
  // Backdrop for nav overlay (mobile only)
  var backdrop = document.createElement('div') ;
  backdrop.style.cssText = 'display:none;position:fixed;inset:0;background:rgba(0,0,0,0.3);z-index:99;' ;
  function openNav() {
    var bc = document.querySelector('.editorBreadcrumb') ;
    var topPx = bc ? (bc.offsetTop + bc.offsetHeight) : 0 ;
    lnav.style.top = topPx + 'px' ;
    lnav.style.maxHeight = 'calc(100dvh - ' + topPx + 'px - env(safe-area-inset-bottom, 0px))' ;
    backdrop.style.top = lnav.style.top ;
    lnav.classList.add('navOpen') ; backdrop.style.display = 'block' ; toggleBtn.textContent = '\u2715' ;
  }
  window._editorOpenNav = openNav ;
  function closeNav() {
    lnav.classList.remove('navOpen') ; backdrop.style.display = 'none' ; toggleBtn.textContent = '\u2630' ;
  }
  toggleBtn.onclick = function() { lnav.classList.contains('navOpen') ? closeNav() : openNav() ; } ;
  backdrop.onclick = closeNav ;
  body.appendChild(backdrop) ; body.appendChild(lnav) ; body.appendChild(rpanel) ;
  div.appendChild(body) ;

  if (_modelReadOnly) applyReadOnly(div) ;
  if (!_modelReadOnly) {
    div.addEventListener('input', markDirty) ;
    div.addEventListener('change', markDirty) ;
  }
}

// ---- Breadcrumb ----

function buildBreadcrumb() {
  var labelMap = {
    'fields':'Details', 'attributes':'Attributes', 'resources':'Resources',
    'versionattributes':'Version Attrs', 'resourceattributes':'Resource Attrs',
    'metaattributes':'Meta Attrs'
  } ;
  var segs = [] ;
  segs.push({label: 'Registry', tab: 'registry', path: []}) ;
  if (_navTab === 'groups') {
    segs.push({label: 'Group Types', tab: 'groups', path: []}) ;
    _navPath.forEach(function(seg, i) {
      var label = labelMap[seg] || seg ;
      var id = (!labelMap[seg] && i === 0) ? 'bcGroupKey' : (!labelMap[seg] && i === 2) ? 'bcResourceKey' : null ;
      segs.push({label: label, tab: 'groups', path: _navPath.slice(0, i+1), id: id}) ;
    }) ;
  } else if (_navPath.length > 0) {
    _navPath.forEach(function(seg, i) {
      segs.push({label: labelMap[seg] || seg, tab: 'registry', path: _navPath.slice(0, i+1)}) ;
    }) ;
  }
  var bc = document.createElement('div') ; bc.className = 'editorBreadcrumb' ;
  var allSegs = segs.slice() ; // structural segments

  // Append _attrNestStack segments — each entry generates 2 segments with full nav info
  _attrNestStack.forEach(function(entry, i) {
    if (entry.isIfValues) {
      allSegs.push({label: entry.key, nestDepth: i, backKey: entry.key}) ;
      allSegs.push({label: 'If-Values', nestDepth: i+1, backKey: null}) ;
    } else if (entry.isSiblings) {
      allSegs.push({label: entry.key, nestDepth: i, backKey: entry.key}) ;
      allSegs.push({label: 'Siblings', nestDepth: i+1, backKey: null}) ;
    } else if (entry.isItem && entry.key === '__item__') {
      // Item-chain sentinel: just one segment for the inner item level
      var typeLabel2 = (entry.attrType || 'map') + ' details' ;
      allSegs.push({label: typeLabel2, nestDepth: i+1, backKey: '__item__'}) ;
    } else if (entry.isItem) {
      allSegs.push({label: entry.key, nestDepth: i, backKey: entry.key}) ;
      var typeLabel = (entry.attrType || 'map') + ' details' ;
      allSegs.push({label: typeLabel, nestDepth: i+1, backKey: '__item__'}) ;
    } else if (entry.key === '__item__') {
      allSegs.push({label: 'Item', nestDepth: i, backKey: '__item__'}) ;
      allSegs.push({label: 'Attributes', nestDepth: i+1, backKey: null}) ;
    } else {
      allSegs.push({label: entry.key, nestDepth: i, backKey: entry.key}) ;
      allSegs.push({label: 'Attributes', nestDepth: i+1, backKey: null}) ;
    }
  }) ;

  allSegs.forEach(function(s, i) {
    if (i > 0) { var sep = document.createElement('span') ; sep.className = 'bcSep' ; sep.textContent = '\u203a' ; bc.appendChild(sep) ; }
    if (i === allSegs.length - 1) {
      var cur = document.createElement('span') ; cur.className = 'bcCurrent' ; cur.textContent = s.label ;
      if (s.id) cur.id = s.id ;
      bc.appendChild(cur) ;
    } else {
      var lnk = document.createElement('span') ; lnk.className = 'bcLink' ; lnk.textContent = s.label ;
      if (s.id) lnk.id = s.id ;
      if (s.nestDepth !== undefined) {
        // Nest-stack segment — pop to nestDepth and optionally re-select
        var nd = s.nestDepth, bk = s.backKey ;
        lnk.onclick = function() {
          collectCurrentEditor() ;
          _attrNestStack = _attrNestStack.slice(0, nd) ;
          _navSelected = bk || null ;
          renderEditor() ;
        } ;
      } else {
        var st = s.tab, sp = s.path ;
        lnk.onclick = function() { collectCurrentEditor() ; _attrNestStack = [] ; _navTab = st ; _navPath = sp ; _navSelected = null ; renderEditor() ; } ;
      }
      bc.appendChild(lnk) ;
    }
  }) ;
  return bc ;
}

// ---- Left Nav ----

function buildLeftNav(div) {
  var model = _modelData || {} ;

  function navItem(label, isContainer, isSelected, clickFn, deleteFn, disabled) {
    var el = document.createElement('div') ;
    el.className = 'navItem' + (isSelected ? ' navItemSelected' : '')
      + (disabled ? ' navItemDisabled' : '') ;
    var lbl = document.createElement('span') ; lbl.style.flex = '1' ;
    if (typeof label === 'string') { lbl.textContent = label ; } else { lbl.appendChild(label) ; }
    el.appendChild(lbl) ;
    if (deleteFn && !_modelReadOnly) {
      var del = document.createElement('span') ; del.className = 'navItemDel' ;
      del.textContent = '\u2715' ; del.title = 'Remove' ;
      del.onclick = function(e) { e.stopPropagation() ; confirmDel('"' + (typeof label === 'string' ? label : el.textContent.trim()) + '"', deleteFn) ; } ;
      el.appendChild(del) ;
    }
    if (isContainer) {
      var arr = document.createElement('span') ; arr.className = 'navItemArrow' ; arr.textContent = '\u203a' ;
      el.appendChild(arr) ;
    }
    // Disabled container items (count of 0) have nothing to drill into —
    // don't wire up the click handler so it's a visual no-op.
    if (!disabled) el.onclick = clickFn ;
    return el ;
  }

  function navAdd(label, fn) {
    var el = document.createElement('div') ; el.className = 'navItemAdd' ;
    el.textContent = label ; el.onclick = fn ; return el ;
  }

  function attrLabel(k) {
    if (k !== '*') return k ;
    var el = document.createElement('span') ;
    var star = document.createElement('span') ; star.textContent = '*' ;
    star.style.cssText = 'font-size:16px;font-weight:bold;vertical-align:middle;line-height:1;' ;
    var desc = document.createElement('span') ; desc.textContent = ' (wildcard extension)' ;
    desc.style.cssText = 'color:#888;font-style:italic;font-size:11px;' ;
    el.appendChild(star) ; el.appendChild(desc) ; return el ;
  }

  function attrSort(keys) {
    return keys.sort(function(a, b) {
      if (a === '*') return 1 ; if (b === '*') return -1 ; return a.localeCompare(b) ;
    }) ;
  }

  function withCount(label, n) { return label + ' (' + n + ')' ; }

  // Wraps a text label with a leading icon thumbnail when iconUrl is set —
  // used for Group Type / Resource Type nav items so their model-declared
  // "icon" (see resolveGroupIcon()/resolveResourceIcon()) shows in the
  // left-nav list. Returns the plain string label when there's no icon, so
  // navItem()'s string-vs-element branch still hits the plain string path.
  function withIcon(label, iconUrl) {
    if (!iconUrl) return label ;
    var wrap = document.createElement('span') ;
    wrap.style.cssText = 'display:inline-flex;align-items:center;' ;
    var img = document.createElement('img') ; img.src = iconUrl ; img.alt = '' ;
    img.className = 'row-icon-thumb' ;
    img.onerror = function() { img.style.display = 'none' ; } ;
    wrap.appendChild(img) ;
    var span = document.createElement('span') ; span.textContent = label ;
    wrap.appendChild(span) ;
    return wrap ;
  }

  if (_attrNestStack.length > 0) {
    var top = _attrNestStack[_attrNestStack.length - 1] ;
    if (top.isItem) {
      div.appendChild(navItem('Item', false, _navSelected === '__item__', function() { selectItem('__item__') ; })) ;
    } else if (top.isIfValues) {
      var ctx = resolveAttrNesting(false) ;
      var ifv = ctx.ifvMap || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Value', addNewIfValue)) ;
      Object.keys(ifv).sort().forEach(function(k) {
        div.appendChild(navItem(k, false, _navSelected === k,
          (function(key){ return function(){ selectItem(key) ; } ; })(k),
          (function(key){ return function(){ deleteIfValue(key) ; } ; })(k))) ;
      }) ;
    } else {
      // Regular nested attrs or siblings context
      var ctx = resolveAttrNesting(false) ;
      var nestedAttrs = ctx.attrsObj || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Attribute', addNewAttr)) ;
      attrSort(Object.keys(nestedAttrs)).forEach(function(k) {
        div.appendChild(navItem(attrLabel(k), false, _navSelected === k,
          (function(key){ return function(){ selectItem(key) ; } ; })(k),
          (function(key){ return function(){ deleteAttr(key) ; } ; })(k))) ;
      }) ;
    }
    return ;
  }

  if (_navTab === 'registry') {
    if (_navPath.length === 0) {
      div.appendChild(navItem('Details', false, _navSelected === 'fields', function() { selectItem('fields') ; })) ;
      var attrCount0 = Object.keys(model.attributes||{}).length ;
      div.appendChild(navItem(withCount('Attributes', attrCount0), true, false, function() { drillDown(['attributes']) ; }, null, _modelReadOnly && attrCount0 === 0)) ;
      var groupCount0 = Object.keys(model.groups||{}).length ;
      div.appendChild(navItem(withCount('Group Types', groupCount0), true, false, function() { changeTab('groups') ; }, null, _modelReadOnly && groupCount0 === 0)) ;
    } else {
      var attrs = model.attributes || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Attribute', addNewAttr)) ;
      attrSort(Object.keys(attrs)).forEach(function(k) {
        div.appendChild(navItem(attrLabel(k), false, _navSelected === k,
          (function(key){ return function(){ selectItem(key) ; } ; })(k),
          (function(key){ return function(){ deleteAttr(key) ; } ; })(k))) ;
      }) ;
    }
  } else {
    if (_navPath.length === 0) {
      var groups = model.groups || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Group', addNewGroup)) ;
      Object.keys(groups).sort().forEach(function(k) {
        var rCount = Object.keys((groups[k]||{}).resources || {}).length ;
        div.appendChild(navItem(withIcon(withCount(k, rCount), groups[k] && groups[k].icon), true, false,
          (function(key){ return function(){ drillDown([key]) ; } ; })(k),
          (function(key){ return function(){ deleteGroup(key) ; } ; })(k))) ;
      }) ;
    } else if (_navPath.length === 1) {
      var gk = _navPath[0] ;
      var grpData = model.groups[gk] || {} ;
      div.appendChild(navItem('Details', false, _navSelected === 'fields', function() { selectItem('fields') ; })) ;
      var grpAttrCount0 = Object.keys(grpData.attributes||{}).length ;
      div.appendChild(navItem(withCount('Attributes', grpAttrCount0), true, false, function() { drillDown([gk, 'attributes']) ; }, null, _modelReadOnly && grpAttrCount0 === 0)) ;
      var grpResCount0 = Object.keys(grpData.resources||{}).length ;
      div.appendChild(navItem(withCount('Resources', grpResCount0), true, false, function() { drillDown([gk, 'resources']) ; }, null, _modelReadOnly && grpResCount0 === 0)) ;
    } else if (_navPath.length === 2 && _navPath[1] === 'attributes') {
      var gk = _navPath[0] ;
      var attrs = (model.groups[gk] || {}).attributes || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Attribute', addNewAttr)) ;
      attrSort(Object.keys(attrs)).forEach(function(k) {
        div.appendChild(navItem(attrLabel(k), false, _navSelected === k,
          (function(key){ return function(){ selectItem(key) ; } ; })(k),
          (function(key){ return function(){ deleteAttr(key) ; } ; })(k))) ;
      }) ;
    } else if (_navPath.length === 2 && _navPath[1] === 'resources') {
      var gk = _navPath[0] ;
      var resources = (model.groups[gk] || {}).resources || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Resource', function(){ addNewResource(gk) ; })) ;
      Object.keys(resources).sort().forEach(function(k) {
        div.appendChild(navItem(withIcon(k, resources[k] && resources[k].icon), true, false,
          (function(key){ return function(){ drillDown([gk, 'resources', key]) ; } ; })(k),
          (function(key){ return function(){ deleteResource(gk, key) ; } ; })(k))) ;
      }) ;
    } else if (_navPath.length === 3) {
      var gk = _navPath[0], rk = _navPath[2] ;
      var resData = ((model.groups[gk]||{}).resources||{})[rk] || {} ;
      div.appendChild(navItem('Details', false, _navSelected === 'fields', function() { selectItem('fields') ; })) ;
      var verAttrCount0 = Object.keys(resData.attributes||{}).length ;
      div.appendChild(navItem(withCount('Version Attrs', verAttrCount0), true, false, function(){ drillDown([gk,'resources',rk,'versionattributes']) ; }, null, _modelReadOnly && verAttrCount0 === 0)) ;
      var resAttrCount0 = Object.keys(resData.resourceattributes||{}).length ;
      div.appendChild(navItem(withCount('Resource Attrs', resAttrCount0), true, false, function(){ drillDown([gk,'resources',rk,'resourceattributes']) ; }, null, _modelReadOnly && resAttrCount0 === 0)) ;
      var metaAttrCount0 = Object.keys(resData.metaattributes||{}).length ;
      div.appendChild(navItem(withCount('Meta Attrs', metaAttrCount0), true, false, function(){ drillDown([gk,'resources',rk,'metaattributes']) ; }, null, _modelReadOnly && metaAttrCount0 === 0)) ;
    } else if (_navPath.length === 4) {
      var gk = _navPath[0], rk = _navPath[2], attrSec = _navPath[3] ;
      var dataKey = attrSec === 'versionattributes' ? 'attributes' : attrSec ;
      var res = ((model.groups[gk] || {}).resources || {})[rk] || {} ;
      var attrs = res[dataKey] || {} ;
      if (!_modelReadOnly) div.appendChild(navAdd('+ Add Attribute', addNewAttr)) ;
      attrSort(Object.keys(attrs)).forEach(function(k) {
        div.appendChild(navItem(attrLabel(k), false, _navSelected === k,
          (function(key){ return function(){ selectItem(key) ; } ; })(k),
          (function(key){ return function(){ deleteAttr(key) ; } ; })(k))) ;
      }) ;
    }
  }
}

// ---- Right Panel ----

function buildRightPanel(div) {
  if (!_navSelected) {
    var hint = document.createElement('div') ; hint.className = 'editorHint' ;
    hint.textContent = '\u2190 Select an item from the left' ; div.appendChild(hint) ;
    // On mobile the nav is hidden in a dropdown — auto-open it so user isn't stranded
    var toggleBtn = document.querySelector('.navToggleBtn') ;
    if (toggleBtn && getComputedStyle(toggleBtn).display !== 'none') {
      setTimeout(function() { var o = window._editorOpenNav ; if (o) o() ; }, 50) ;
    }
    return ;
  }

  // Nested attribute context
  if (_attrNestStack.length > 0) {
    var top = _attrNestStack[_attrNestStack.length - 1] ;
    if (top.isItem && _navSelected === '__item__') {
      var ctx = resolveAttrNesting(false) ;
      renderItemForm(div, ctx.parentAttr ? (ctx.parentAttr.item || {}) : {}) ;
      return ;
    }
    if (top.isIfValues && _navSelected) {
      var ctx = resolveAttrNesting(false) ;
      renderIfValueForm(div, _navSelected, ctx.ifvMap || {}) ;
      return ;
    }
    if (!top.isItem && !top.isIfValues) {
      // Regular nested attrs or siblings
      var ctx2 = resolveAttrNesting(false) ;
      var nestedAttr = (ctx2.attrsObj || {})[_navSelected] || {} ;
      renderAttrForm(div, nestedAttr) ;
      return ;
    }
  }

  var model = _modelData || {} ;
  if (_navTab === 'registry') {
    if (_navSelected === 'fields') { renderRegistryFields(div) ; }
    else { renderAttrForm(div, (model.attributes || {})[_navSelected] || {}) ; }
  } else {
    var gk = _navPath.length > 0 ? _navPath[0] : null ;
    if (_navPath.length === 1 && _navSelected === 'fields') {
      renderGroupFields(div, gk) ;
    } else if (_navPath.length === 2 && _navPath[1] === 'attributes') {
      renderAttrForm(div, ((model.groups[gk] || {}).attributes || {})[_navSelected] || {}) ;
    } else if (_navPath.length === 3 && _navSelected === 'fields') {
      renderResourceFields(div, gk, _navPath[2]) ;
    } else if (_navPath.length === 4) {
      var attrSec = _navPath[3] ;
      var dataKey = attrSec === 'versionattributes' ? 'attributes' : attrSec ;
      var res = (((model.groups[gk] || {}).resources || {})[_navPath[2]] || {}) ;
      renderAttrForm(div, (res[dataKey] || {})[_navSelected] || {}) ;
    }
  }
}

// ---- Collect current editor into _modelData ----

function collectCurrentEditor() {
  if (!_navSelected) return ;
  // Read-only mode never renders editable inputs (just plain text spans), so
  // there's nothing to collect — and doing so anyway would silently
  // overwrite real data (e.g. constraints) with empty placeholders, since
  // the DOM-reading helpers (collectConstraints, collectLabels, etc.)
  // assume the edit-mode DOM structure.
  if (_modelReadOnly) return ;

  // Nested attribute context — handle first
  if (_attrNestStack.length > 0) {
    var top = _attrNestStack[_attrNestStack.length - 1] ;
    if (top.isItem && _navSelected === '__item__') {
      var ctx = resolveAttrNesting(true) ;
      if (ctx.parentAttr) saveItemForm(ctx.parentAttr) ;
      return ;
    }
    if (top.isIfValues && _navSelected) {
      var ctx = resolveAttrNesting(true) ;
      saveIfValueFrom(ctx.ifvMap, _navSelected) ;
      return ;
    }
    if (!top.isItem && !top.isIfValues) {
      var ctx2 = resolveAttrNesting(true) ;
      if (ctx2.attrsObj) saveAttrFrom(ctx2.attrsObj, _navSelected) ;
      return ;
    }
    return ;
  }

  var model = _modelData || {} ;
  if (_navTab === 'registry') {
    if (_navSelected === 'fields') {
      var d = fv('ef_description') ; if (d) model.description = d ; else delete model.description ;
      var dc = fv('ef_documentation') ; if (dc) model.documentation = dc ; else delete model.documentation ;
      var lbls = collectLabels('ef_labels') ;
      if (Object.keys(lbls).length) model.labels = lbls ; else delete model.labels ;
    } else {
      if (!model.attributes) model.attributes = {} ;
      saveAttrFrom(model.attributes, _navSelected) ;
    }
  } else {
    var gk = _navPath.length > 0 ? _navPath[0] : null ; if (!gk) return ;
    if (!model.groups) model.groups = {} ;
    if (!model.groups[gk]) model.groups[gk] = {} ;
    var grp = model.groups[gk] ;
    if (_navPath.length === 1 && _navSelected === 'fields') {
      saveGroupFields(gk) ;
    } else if (_navPath.length === 2 && _navPath[1] === 'attributes') {
      if (!grp.attributes) grp.attributes = {} ;
      saveAttrFrom(grp.attributes, _navSelected) ;
    } else if (_navPath.length === 3 && _navSelected === 'fields') {
      var rk = _navPath[2] ;
      if (!grp.resources) grp.resources = {} ;
      if (!grp.resources[rk]) grp.resources[rk] = {} ;
      saveResourceFields(gk, rk) ;
    } else if (_navPath.length === 4) {
      var rk = _navPath[2], attrSec = _navPath[3] ;
      var dataKey = attrSec === 'versionattributes' ? 'attributes' : attrSec ;
      if (!grp.resources) grp.resources = {} ;
      if (!grp.resources[rk]) grp.resources[rk] = {} ;
      var res = grp.resources[rk] ;
      if (!res[dataKey]) res[dataKey] = {} ;
      saveAttrFrom(res[dataKey], _navSelected) ;
    }
  }
}

function saveAttrFrom(attrsObj, origKey) {
  var nameEl = document.getElementById('ef_name') ; if (!nameEl) return ;
  var newName = nameEl.value.trim() || origKey ;
  // Read existing entry first so we can preserve nested structures (attributes/item)
  // that are edited via drill-down and not touched by this form
  var existing = attrsObj[origKey] || {} ;
  var attr = { name: newName } ;
  var t = fv('ef_type') ; if (t) attr.type = t ;
  var d = fv('ef_description') ; if (d) attr.description = d ;
  var def = fv('ef_default') ; if (def !== '') attr.default = def ;
  var tgt = fv('ef_target') ;
  var targetEl = document.getElementById('ef_target') ;
  if (tgt && targetEl && !targetEl.disabled) attr.target = tgt ;
  var ncs = fv('ef_namecharset') ;
  var ncsEl = document.getElementById('ef_namecharset') ;
  if (ncs && ncsEl && !ncsEl.disabled) attr.namecharset = ncs ;
  var enm = collectEnum('ef_enum') ;
  if (enm.length) attr.enum = enm ;
  ['required','readonly','immutable','matchcase','matchversions','strict'].forEach(function(f) {
    var v = fvBool('ef_'+f) ;
    if (v === true) attr[f] = true ;
    else if (v === false) attr[f] = false ;
    else delete attr[f] ;
  }) ;
  // Preserve nested structures edited via drill-down (not part of this
  // form) — but only when still relevant to the (possibly just-changed)
  // type, so switching e.g. map -> string doesn't leave a stale "item"
  // (or object -> string leaving a stale "attributes") in the saved JSON.
  if (existing.attributes && t === 'object') attr.attributes = existing.attributes ;
  if (existing.item && (t === 'map' || t === 'array')) attr.item = existing.item ;
  if (existing.ifvalues) attr.ifvalues = existing.ifvalues ;
  if (newName !== origKey && attrsObj[origKey] !== undefined) delete attrsObj[origKey] ;
  attrsObj[newName] = attr ;
  if (_navSelected === origKey) _navSelected = newName ;
}

function saveGroupFields(gk) {
  var model = _modelData || {} ;
  if (!model.groups) model.groups = {} ;
  var grp = model.groups[gk] || {} ;
  var plural = fv('ef_plural') ;
  setOrDel(grp, 'plural', plural) ; setOrDel(grp, 'singular', fv('ef_singular')) ;
  setOrDel(grp, 'description', fv('ef_description')) ; setOrDel(grp, 'documentation', fv('ef_documentation')) ;
  setOrDel(grp, 'icon', fv('ef_icon')) ; setOrDel(grp, 'modelversion', fv('ef_modelversion')) ;
  setOrDel(grp, 'modelcompatiblewith', fv('ef_modelcompatiblewith')) ;
  var lbls = collectLabels('ef_labels') ;
  if (Object.keys(lbls).length) grp.labels = lbls ; else delete grp.labels ;
  var cstrs = collectConstraints('ef_constraints') ;
  if (Object.keys(cstrs).length) grp.constraints = cstrs ; else delete grp.constraints ;
  var newKey = plural || gk ;
  if (newKey !== gk) { delete model.groups[gk] ; model.groups[newKey] = grp ; _navPath[0] = newKey ; }
  else model.groups[gk] = grp ;
}

function saveResourceFields(gk, rk) {
  var model = _modelData || {} ;
  var grp = (model.groups || {})[gk] || {} ;
  var res = (grp.resources || {})[rk] || {} ;
  var plural = fv('ef_plural') ;
  setOrDel(res, 'plural', plural) ; setOrDel(res, 'singular', fv('ef_singular')) ;
  setOrDel(res, 'description', fv('ef_description')) ; setOrDel(res, 'documentation', fv('ef_documentation')) ;
  setOrDel(res, 'icon', fv('ef_icon')) ; setOrDel(res, 'modelversion', fv('ef_modelversion')) ;
  setOrDel(res, 'modelcompatiblewith', fv('ef_modelcompatiblewith')) ;
  var maxv = fv('ef_maxversions') ;
  if (maxv !== '') res.maxversions = parseInt(maxv, 10) || 0 ; else delete res.maxversions ;
  setOrDel(res, 'versionmode', fv('ef_versionmode')) ;
  ['setversionid','hasdocument','singleversionroot','validateformat','validatecompatibility','strictvalidation'].forEach(function(f) {
    var v = fvBool('ef_'+f) ;
    if (v === true) res[f] = true ;
    else if (v === false) res[f] = false ;
    else delete res[f] ;
  }) ;
  var lbls = collectLabels('ef_labels') ;
  if (Object.keys(lbls).length) res.labels = lbls ; else delete res.labels ;
  var newKey = plural || rk ;
  if (newKey !== rk) { delete grp.resources[rk] ; grp.resources[newKey] = res ; _navPath[2] = newKey ; }
  else grp.resources[rk] = res ;
}

function setOrDel(obj, key, val) { if (val) obj[key] = val ; else delete obj[key] ; }

// ---- Form renderers ----

function addFormTitle(div, title) {
  var h = document.createElement('div') ; h.className = 'editorFormTitle' ;
  h.textContent = title ; div.appendChild(h) ;
}

function renderRegistryFields(div) {
  var m = _modelData || {} ;
  addFormTitle(div, 'Registry Details') ;
  div.appendChild(ef('ef_description', 'Description', m.description||'')) ;
  div.appendChild(ef('ef_documentation', 'Documentation', m.documentation||'')) ;
  div.appendChild(makeLabelsEditor('ef_labels', m.labels||{})) ;
}

function renderGroupFields(div, gk) {
  var grp = ((_modelData||{}).groups||{})[gk] || {} ;
  var titleEl = document.createElement('div') ; titleEl.className = 'editorFormTitle' ;
  titleEl.textContent = 'Group Type: ' + (grp.plural || gk) ; div.appendChild(titleEl) ;
  var pluralRow = ef('ef_plural', 'Plural', grp.plural||gk, true) ; div.appendChild(pluralRow) ;
  var pluralInp = pluralRow.querySelector('input') ;
  pluralInp.oninput = function() {
    var v = pluralInp.value.trim() || gk ;
    titleEl.textContent = 'Group Type: ' + v ;
    var bc = document.getElementById('bcGroupKey') ; if (bc) bc.textContent = v ;
  } ;
  div.appendChild(ef('ef_singular', 'Singular', grp.singular||'', true)) ;
  div.appendChild(ef('ef_description', 'Description', grp.description||'')) ;
  div.appendChild(ef('ef_documentation', 'Documentation', grp.documentation||'')) ;
  div.appendChild(efIconPreview(ef('ef_icon', 'Icon URL', grp.icon||''))) ;
  div.appendChild(ef('ef_modelversion', 'Model Version', grp.modelversion||'')) ;
  div.appendChild(ef('ef_modelcompatiblewith', 'ModelCompatibleWith', grp.modelcompatiblewith||'')) ;
  div.appendChild(makeLabelsEditor('ef_labels', grp.labels||{})) ;
  div.appendChild(makeConstraintsEditor('ef_constraints', grp.constraints||{}, gk)) ;
}

function renderResourceFields(div, gk, rk) {
  var res = (((_modelData||{}).groups||{})[gk]||{}).resources||{} ;
  var r = res[rk] || {} ;
  var titleEl = document.createElement('div') ; titleEl.className = 'editorFormTitle' ;
  titleEl.textContent = 'Resource: ' + (r.plural || rk) ; div.appendChild(titleEl) ;
  var pluralRow = ef('ef_plural', 'Plural', r.plural||rk, true) ; div.appendChild(pluralRow) ;
  var pluralInp = pluralRow.querySelector('input') ;
  pluralInp.oninput = function() {
    var v = pluralInp.value.trim() || rk ;
    titleEl.textContent = 'Resource: ' + v ;
    var bc = document.getElementById('bcResourceKey') ; if (bc) bc.textContent = v ;
  } ;
  div.appendChild(ef('ef_singular', 'Singular', r.singular||'', true)) ;
  div.appendChild(ef('ef_description', 'Description', r.description||'')) ;
  div.appendChild(ef('ef_documentation', 'Documentation', r.documentation||'')) ;
  div.appendChild(efIconPreview(ef('ef_icon', 'Icon URL', r.icon||''))) ;
  div.appendChild(ef('ef_modelversion', 'Model Version', r.modelversion||'')) ;
  div.appendChild(ef('ef_modelcompatiblewith', 'ModelCompatibleWith', r.modelcompatiblewith||'')) ;
  div.appendChild(efNum('ef_maxversions', 'Max Versions', r.maxversions)) ;
  div.appendChild(ef('ef_versionmode', 'Version Mode', r.versionmode||'')) ;
  var optSec = document.createElement('div') ; optSec.className = 'editorSectionLabel' ; optSec.textContent = 'Options' ;
  div.appendChild(optSec) ;
  var optList = [
    ['hasdocument',          'Has Document',          r.hasdocument],
    ['setversionid',         'Set Version ID',         r.setversionid],
    ['singleversionroot',    'Single Version Root',    r.singleversionroot],
    ['strictvalidation',     'Strict Validation',      r.strictvalidation],
    ['validatecompatibility','Validate Compatibility', r.validatecompatibility],
    ['validateformat',       'Validate Format',        r.validateformat]
  ] ;
  var boolGrid = document.createElement('div') ; boolGrid.className = 'boolGrid' ;
  optList.forEach(function(t) { boolGrid.appendChild(efBool('ef_'+t[0], t[1], t[2])) ; }) ;
  div.appendChild(boolGrid) ;
  div.appendChild(makeLabelsEditor('ef_labels', r.labels||{})) ;
}

function renderAttrForm(div, attr) {
  // Determine if this is the versionattributes context (matchversions only shown here)
  var isVersionAttrs = (_navPath.length === 4 && _navPath[3] === 'versionattributes') ;

  // Title with live update as name is typed
  var titleEl = document.createElement('div') ; titleEl.className = 'editorFormTitle' ;
  titleEl.textContent = 'Attribute: ' + (attr.name || _navSelected || '') ;
  div.appendChild(titleEl) ;

  var origInp = document.createElement('input') ; origInp.type = 'hidden' ;
  origInp.id = 'ef_origname' ; origInp.value = attr.name || _navSelected || '' ;
  div.appendChild(origInp) ;

  var nameRow = ef('ef_name', 'Name', attr.name || _navSelected || '', true) ;
  var nameInp = nameRow.querySelector('input') ;
  nameInp.maxLength = 63 ;
  nameInp.title = 'Lowercase letters, digits, underscore only; max 63 chars; cannot start with a digit. Use * for wildcard extension.' ;
  nameInp.oninput = function() {
    var raw = nameInp.value ;
    if (raw.indexOf('*') !== -1) {
      // Any input with * collapses to just '*'
      nameInp.value = '*' ;
    } else {
      var cleaned = raw.toLowerCase().replace(/[^a-z0-9_]/g, '') ;
      if (cleaned !== raw) {
        var pos = nameInp.selectionStart - (raw.length - cleaned.length) ;
        nameInp.value = cleaned ; nameInp.selectionStart = nameInp.selectionEnd = Math.max(0, pos) ;
      }
    }
    var v = nameInp.value.trim() || '\u2026' ;
    titleEl.textContent = 'Attribute: ' + v ;
    var navEl = document.querySelector('.navItemSelected') ;
    if (navEl) { var sp = navEl.firstChild ; if (sp) sp.textContent = v ; }
  } ;
  div.appendChild(nameRow) ;

  // Type dropdown
  var typeRow = document.createElement('div') ; typeRow.className = 'editorField' ;
  var typeLbl = document.createElement('label') ; typeLbl.textContent = 'Type:' ;
  var typeReq = document.createElement('span') ; typeReq.textContent = ' *' ; typeReq.style.cssText = 'color:#c00;font-weight:bold;' ;
  typeLbl.appendChild(typeReq) ;
  var typeSel = document.createElement('select') ; typeSel.id = 'ef_type' ; typeSel.className = 'editorInput' ;
  ['boolean','decimal','integer','string','timestamp',
   'uinteger','uri','uriabsolute','urirelative','uritemplate','url','urlabsolute','urlrelative','xid','xidtype',
   'any','array','map','object'
  ].forEach(function(opt) {
    var o = document.createElement('option') ; o.value = opt ; o.textContent = opt ;
    if ((attr.type||'string') === opt) o.selected = true ;
    typeSel.appendChild(o) ;
  }) ;
  typeRow.appendChild(typeLbl) ;
  var typeWrap = document.createElement('div') ; typeWrap.className = 'editorSelectWrap' ;
  typeWrap.appendChild(typeSel) ; typeRow.appendChild(typeWrap) ; div.appendChild(typeRow) ;

  // Nested-type drill-down button — right below Type, aligned with the dropdown
  var nestBtnRow = document.createElement('div') ; nestBtnRow.className = 'editorField' ;
  nestBtnRow.style.marginBottom = '6px' ;
  var nestLblSpacer = document.createElement('label') ; nestLblSpacer.style.visibility = 'hidden' ;
  nestBtnRow.appendChild(nestLblSpacer) ;
  var nestBtn = document.createElement('button') ;
  nestBtn.className = 'editorBtn navDrillBtn' ;
  nestBtn.style.cssText = 'font-size:11px;padding:3px 8px;' ;
  var currentAttrKey = _navSelected ;
  function updateNestBtn() {
    var t = typeSel.value ;
    if (t === 'object') {
      var cnt = Object.keys(attr.attributes || {}).length ;
      nestBtn.textContent = '\u25b6 Edit Nested Attributes' + (cnt ? ' ('+cnt+')' : '') ;
      nestBtn.style.display = '' ;
      nestBtn.onclick = function() { drillIntoAttr(currentAttrKey, false) ; } ;
    } else if (t === 'map' || t === 'array') {
      nestBtn.textContent = '\u25b6 Edit ' + t + ' details' ;
      nestBtn.style.display = '' ;
      nestBtn.onclick = function() { drillIntoAttr(currentAttrKey, true) ; } ;
    } else {
      nestBtn.style.display = 'none' ;
    }
  }
  nestBtnRow.appendChild(nestBtn) ; div.appendChild(nestBtnRow) ;
  updateNestBtn() ;
  typeSel.addEventListener('change', updateNestBtn) ;

  div.appendChild(ef('ef_description', 'Description', attr.description||'')) ;
  div.appendChild(ef('ef_default', 'Default', attr.default !== undefined ? String(attr.default) : '')) ;

  // Target — text field, only relevant for url/xid
  var targetRow = ef('ef_target', 'Target', attr.target||'') ; div.appendChild(targetRow) ;
  var targetInp = targetRow.querySelector('input') ;
  targetInp.placeholder = 'e.g. /groups/resources' ;

  // Name Charset — dropdown, only relevant for type=object
  var ncsRow = document.createElement('div') ; ncsRow.className = 'editorField' ;
  var ncsLbl = document.createElement('label') ; ncsLbl.textContent = 'Name Charset:' ;
  var ncsSel = document.createElement('select') ; ncsSel.id = 'ef_namecharset' ; ncsSel.className = 'editorInput' ;
  var ncsWrap = document.createElement('div') ; ncsWrap.className = 'editorSelectWrap' ;
  [['','(default / strict)'],['strict','strict'],['extended','extended']].forEach(function(p) {
    var o = document.createElement('option') ; o.value = p[0] ; o.textContent = p[1] ;
    if ((attr.namecharset||'') === p[0]) o.selected = true ;
    ncsSel.appendChild(o) ;
  }) ;
  ncsWrap.appendChild(ncsSel) ; ncsRow.appendChild(ncsLbl) ; ncsRow.appendChild(ncsWrap) ; div.appendChild(ncsRow) ;

  // Enable/disable target and namecharset based on current type
  function syncTypeFields() {
    var t = typeSel.value ;
    var targetTypes = {url:1,urlabsolute:1,urlrelative:1,uri:1,uriabsolute:1,urirelative:1,uritemplate:1,xid:1,xidtype:1} ;
    targetInp.disabled = !targetTypes[t] ;
    targetInp.style.opacity = targetInp.disabled ? '0.4' : '1' ;
    ncsSel.disabled = (t !== 'object') ;
    ncsSel.style.opacity = ncsSel.disabled ? '0.4' : '1' ;
  }
  syncTypeFields() ;
  typeSel.addEventListener('change', syncTypeFields) ;

  div.appendChild(makeEnumEditor('ef_enum', Array.isArray(attr.enum) ? attr.enum : [])) ;
  var optSec = document.createElement('div') ; optSec.className = 'editorSectionLabel' ; optSec.textContent = 'Options' ;
  div.appendChild(optSec) ;
  var optList = [
    ['immutable',  'Immutable',  attr.immutable],
    ['matchcase',  'Match Case', attr.matchcase],
    ['readonly',   'Read Only',  attr.readonly],
    ['required',   'Required',   attr.required],
    ['strict',     'Strict',     attr.strict]
  ] ;
  if (isVersionAttrs) optList.push(['matchversions','Match Versions', attr.matchversions]) ;
  // Sort alphabetically by label
  optList.sort(function(a,b){ return a[1].localeCompare(b[1]) ; }) ;
  var boolGrid = document.createElement('div') ; boolGrid.className = 'boolGrid' ;
  optList.forEach(function(t) { boolGrid.appendChild(efBool('ef_'+t[0], t[1], t[2])) ; }) ;
  div.appendChild(boolGrid) ;

  // If-Values drill-down button — next to the section header, matching the
  // Labels/Enum/Constraints "+ Add" placement pattern.
  var ifvSec = document.createElement('div') ; ifvSec.className = 'editorSectionLabel' ; ifvSec.textContent = 'If-Values' ;
  div.appendChild(ifvSec) ;
  var ifvCount = Object.keys(attr.ifvalues || {}).length ;
  if (_modelReadOnly && !ifvCount) {
    var ifvNone = document.createElement('span') ; ifvNone.textContent = '\u2014 none \u2014' ;
    ifvNone.style.cssText = 'color:#aaa;font-size:12px;font-style:italic;margin-left:4px;' ;
    div.appendChild(ifvNone) ;
  } else {
    var ifvBtn = document.createElement('button') ; ifvBtn.className = 'editorBtn editorBtnSmall navDrillBtn' ;
    ifvBtn.textContent = '\u25b6 If-Values' + (ifvCount ? ' ('+ifvCount+')' : '') ;
    ifvBtn.onclick = function() {
      collectCurrentEditor() ;
      var resolvedKey = _navSelected || currentAttrKey ;
      _attrNestStack.push({key:resolvedKey, isIfValues:true}) ;
      _navSelected = null ; renderEditor() ;
    } ;
    ifvSec.appendChild(ifvBtn) ;
  }
}

function renderItemForm(div, item) {
  item = item || {} ;
  // Determine parent type from stack (map/array) for title
  var parentType = 'map' ;
  for (var si = _attrNestStack.length-1; si >= 0; si--) {
    if (_attrNestStack[si].isItem) { parentType = _attrNestStack[si].attrType || 'map' ; break ; }
  }
  var titleEl = document.createElement('div') ; titleEl.className = 'editorFormTitle' ;
  titleEl.textContent = parentType.charAt(0).toUpperCase() + parentType.slice(1) + ' Details' ;
  div.appendChild(titleEl) ;

  // Type dropdown
  var typeRow = document.createElement('div') ; typeRow.className = 'editorField' ;
  var typeLbl = document.createElement('label') ; typeLbl.textContent = 'Type:' ;
  var typeSel = document.createElement('select') ; typeSel.id = 'ef_item_type' ; typeSel.className = 'editorInput' ;
  ['boolean','decimal','integer','string','timestamp',
   'uinteger','uri','uriabsolute','urirelative','uritemplate','url','urlabsolute','urlrelative','xid','xidtype',
   'any','array','map','object'
  ].forEach(function(opt) {
    var o = document.createElement('option') ; o.value = opt ; o.textContent = opt ;
    if ((item.type||'string') === opt) o.selected = true ;
    typeSel.appendChild(o) ;
  }) ;
  typeRow.appendChild(typeLbl) ;
  var typeWrap = document.createElement('div') ; typeWrap.className = 'editorSelectWrap' ;
  typeWrap.appendChild(typeSel) ; typeRow.appendChild(typeWrap) ; div.appendChild(typeRow) ;

  // Nested-type drill-down button — right below Type, aligned with the dropdown
  var nestBtnRow = document.createElement('div') ; nestBtnRow.className = 'editorField' ;
  nestBtnRow.style.marginBottom = '6px' ;
  var nestLblSpacer2 = document.createElement('label') ; nestLblSpacer2.style.visibility = 'hidden' ;
  nestBtnRow.appendChild(nestLblSpacer2) ;
  var nestBtn = document.createElement('button') ; nestBtn.className = 'editorBtn navDrillBtn' ;
  nestBtn.style.cssText = 'font-size:11px;padding:3px 8px;' ;
  function updateItemNestBtn() {
    var t = typeSel.value ;
    if (t === 'object') {
      var cnt = Object.keys(item.attributes || {}).length ;
      nestBtn.textContent = '\u25b6 Edit Nested Attributes' + (cnt ? ' ('+cnt+')' : '') ;
      nestBtn.style.display = '' ;
      nestBtn.onclick = function() {
        var top = _attrNestStack[_attrNestStack.length - 1] ;
        var parentKey = top ? top.key : null ; if (!parentKey) return ;
        var ctx = resolveAttrNesting(true) ;
        if (ctx.parentAttr) saveItemForm(ctx.parentAttr) ;
        _attrNestStack.push({key:'__item__', isItem:false}) ;
        _navSelected = null ; renderEditor() ;
      } ;
    } else if (t === 'map' || t === 'array') {
      nestBtn.textContent = '\u25b6 Edit ' + t + ' details' ;
      nestBtn.style.display = '' ;
      nestBtn.onclick = function() {
        var top = _attrNestStack[_attrNestStack.length - 1] ;
        var parentKey = top ? top.key : null ; if (!parentKey) return ;
        var ctx = resolveAttrNesting(true) ;
        if (ctx.parentAttr) saveItemForm(ctx.parentAttr) ;
        _attrNestStack.push({key:'__item__', isItem:true, attrType:t}) ;
        _navSelected = '__item__' ; renderEditor() ;
      } ;
    } else {
      nestBtn.style.display = 'none' ;
    }
  }
  nestBtnRow.appendChild(nestBtn) ; div.appendChild(nestBtnRow) ;

  var targetRow = ef('ef_item_target', 'Target', item.target||'') ; div.appendChild(targetRow) ;
  var targetInp = targetRow.querySelector('input') ;
  targetInp.placeholder = 'e.g. /groups/resources' ;

  var ncsRow = document.createElement('div') ; ncsRow.className = 'editorField' ;
  var ncsLbl = document.createElement('label') ; ncsLbl.textContent = 'Name Charset:' ;
  var ncsSel = document.createElement('select') ; ncsSel.id = 'ef_item_namecharset' ; ncsSel.className = 'editorInput' ;
  var ncsWrap = document.createElement('div') ; ncsWrap.className = 'editorSelectWrap' ;
  [['','(default / strict)'],['strict','strict'],['extended','extended']].forEach(function(p) {
    var o = document.createElement('option') ; o.value = p[0] ; o.textContent = p[1] ;
    if ((item.namecharset||'') === p[0]) o.selected = true ;
    ncsSel.appendChild(o) ;
  }) ;
  ncsWrap.appendChild(ncsSel) ; ncsRow.appendChild(ncsLbl) ; ncsRow.appendChild(ncsWrap) ; div.appendChild(ncsRow) ;

  // These fields are only meaningful for complex (object/map/array) item types
  var complexSec = document.createElement('div') ;
  complexSec.appendChild(ef('ef_item_description', 'Description', item.description||'')) ;
  complexSec.appendChild(ef('ef_item_default', 'Default', item.default !== undefined ? String(item.default) : '')) ;
  complexSec.appendChild(makeEnumEditor('ef_item_enum', Array.isArray(item.enum) ? item.enum : [])) ;
  var optSec = document.createElement('div') ; optSec.className = 'editorSectionLabel' ; optSec.textContent = 'Options' ;
  complexSec.appendChild(optSec) ;
  var optList = [
    ['item_readonly', 'Read Only', item.readonly],
    ['item_strict',   'Strict',    item.strict]
  ] ;
  var boolGrid = document.createElement('div') ; boolGrid.className = 'boolGrid' ;
  optList.forEach(function(t) { boolGrid.appendChild(efBool('ef_'+t[0], t[1], t[2])) ; }) ;
  complexSec.appendChild(boolGrid) ;
  div.appendChild(complexSec) ;

  function syncItemTypeFields() {
    var t = typeSel.value ;
    var targetTypes = {url:1,urlabsolute:1,urlrelative:1,uri:1,uriabsolute:1,urirelative:1,uritemplate:1,xid:1,xidtype:1} ;
    targetInp.disabled = !targetTypes[t] ;
    targetInp.style.opacity = targetInp.disabled ? '0.4' : '1' ;
    ncsSel.disabled = (t !== 'object') ;
    ncsSel.style.opacity = ncsSel.disabled ? '0.4' : '1' ;
    updateItemNestBtn() ;
    // description/default/enum/options are only relevant for complex types
    complexSec.style.display = {object:1,map:1,array:1}[t] ? '' : 'none' ;
  }
  updateItemNestBtn() ;
  syncItemTypeFields() ;
  typeSel.addEventListener('change', syncItemTypeFields) ;
}

function saveItemForm(parentAttr) {
  if (!parentAttr) return ;
  if (!parentAttr.item) parentAttr.item = {} ;
  var itm = parentAttr.item ;
  var t = fv('ef_item_type') ; if (t) itm.type = t ; else delete itm.type ;
  var d = fv('ef_item_description') ; if (d) itm.description = d ; else delete itm.description ;
  var def = fv('ef_item_default') ; if (def !== '') itm.default = def ; else delete itm.default ;
  var targetEl = document.getElementById('ef_item_target') ;
  if (targetEl && !targetEl.disabled) { var tgt = targetEl.value.trim() ; if (tgt) itm.target = tgt ; else delete itm.target ; }
  var ncsEl = document.getElementById('ef_item_namecharset') ;
  if (ncsEl && !ncsEl.disabled) { var ncs = ncsEl.value ; if (ncs) itm.namecharset = ncs ; else delete itm.namecharset ; }
  var enm = collectEnum('ef_item_enum') ;
  if (enm.length) itm.enum = enm ; else delete itm.enum ;
  var rov = fvBool('ef_item_readonly') ; if (rov === true) itm.readonly = true ; else if (rov === false) itm.readonly = false ; else delete itm.readonly ;
  var stv = fvBool('ef_item_strict') ; if (stv === true) itm.strict = true ; else if (stv === false) itm.strict = false ; else delete itm.strict ;
}

function uniqueKey(obj, base) {
  if (!obj || !obj[base]) return base ;
  var i = 2 ; while (obj[base+i]) i++ ; return base+i ;
}

function addNewGroup() {
  collectCurrentEditor() ;
  var m = _modelData || {} ; if (!m.groups) m.groups = {} ;
  var key = uniqueKey(m.groups, 'new') ;
  m.groups[key] = {plural:'',singular:''} ;
  markDirty() ; _navTab = 'groups' ; _navPath = [key] ; _navSelected = 'fields' ; renderEditor() ;
}

function addNewResource(gk) {
  collectCurrentEditor() ;
  var m = _modelData || {} ; var grp = (m.groups||{})[gk] ; if (!grp) return ;
  if (!grp.resources) grp.resources = {} ;
  var key = uniqueKey(grp.resources, 'new') ;
  grp.resources[key] = {plural:'',singular:''} ;
  markDirty() ; _navPath = [gk,'resources',key] ; _navSelected = 'fields' ; renderEditor() ;
}

function addNewAttr() {
  collectCurrentEditor() ;
  // Nested context: use resolved attrs container
  if (_attrNestStack.length > 0) {
    var top = _attrNestStack[_attrNestStack.length - 1] ;
    if (!top.isItem && !top.isIfValues) {
      var ctx = resolveAttrNesting(true) ;
      var nestedAttrs = ctx.attrsObj ;
      if (!nestedAttrs) return ;
      var key = uniqueKey(nestedAttrs, 'new') ;
      nestedAttrs[key] = {name:key, type:'string'} ;
      markDirty() ; _navSelected = key ; renderEditor() ;
    }
    return ;
  }
  var m = _modelData || {} ; var attrsObj ;
  if (_navTab === 'registry') {
    if (!m.attributes) m.attributes = {} ; attrsObj = m.attributes ;
  } else {
    var gk = _navPath[0] ; var grp = (m.groups||{})[gk] ; if (!grp) return ;
    if (_navPath.length === 2 && _navPath[1] === 'attributes') {
      if (!grp.attributes) grp.attributes = {} ; attrsObj = grp.attributes ;
    } else if (_navPath.length === 4) {
      var rk = _navPath[2], attrSec = _navPath[3] ;
      var dataKey = attrSec === 'versionattributes' ? 'attributes' : attrSec ;
      var res = (grp.resources||{})[rk] ; if (!res) return ;
      if (!res[dataKey]) res[dataKey] = {} ; attrsObj = res[dataKey] ;
    }
  }
  if (!attrsObj) return ;
  var key = uniqueKey(attrsObj, 'new') ;
  attrsObj[key] = {name:key, type:'string'} ;
  markDirty() ; _navSelected = key ; renderEditor() ;
}

function confirmDel(label, fn) {
  if (confirm('Delete ' + label + '?')) fn() ;
}

function deleteGroup(gk) {
  var m = _modelData || {} ; if (m.groups) delete m.groups[gk] ;
  markDirty() ; _navPath = [] ; _navSelected = null ; renderEditor() ;
}

function deleteResource(gk, rk) {
  var m = _modelData || {} ; var grp = (m.groups||{})[gk] ;
  if (grp && grp.resources) delete grp.resources[rk] ;
  markDirty() ; _navPath = [gk,'resources'] ; _navSelected = null ; renderEditor() ;
}

function deleteAttr(key) {
  // Nested context: delete from resolved attrs container
  if (_attrNestStack.length > 0) {
    var top = _attrNestStack[_attrNestStack.length - 1] ;
    if (!top.isItem && !top.isIfValues) {
      var ctx = resolveAttrNesting(false) ;
      if (ctx.attrsObj) delete ctx.attrsObj[key] ;
      markDirty() ; if (_navSelected === key) _navSelected = null ; renderEditor() ;
    }
    return ;
  }
  var m = _modelData || {} ; var attrsObj ;
  if (_navTab === 'registry') { attrsObj = m.attributes ; }
  else {
    var gk = _navPath[0] ; var grp = (m.groups||{})[gk] ;
    if (grp) {
      if (_navPath.length === 2 && _navPath[1] === 'attributes') attrsObj = grp.attributes ;
      else if (_navPath.length === 4) {
        var dataKey = _navPath[3] === 'versionattributes' ? 'attributes' : _navPath[3] ;
        var res = (grp.resources||{})[_navPath[2]] ; if (res) attrsObj = res[dataKey] ;
      }
    }
  }
  if (attrsObj) delete attrsObj[key] ;
  markDirty() ; if (_navSelected === key) _navSelected = null ; renderEditor() ;
}

// ---- UI helpers ----

function ef(id, label, value, required) {
  var row = document.createElement('div') ; row.className = 'editorField' ;
  var lbl = document.createElement('label') ; lbl.textContent = label + ':' ;
  if (required) {
    var req = document.createElement('span') ; req.textContent = ' *' ;
    req.style.cssText = 'color:#c00;font-weight:bold;' ;
    lbl.appendChild(req) ;
  }
  var inp = document.createElement('input') ;
  inp.type = 'text' ; inp.id = id ; inp.value = value ; inp.className = 'editorInput' ;
  row.appendChild(lbl) ; row.appendChild(inp) ; return row ;
}

// Attaches a live icon-thumbnail preview next to an "Icon URL" field row
// (see ef()) — updates as the user types, hides automatically (via
// onerror) when the URL is empty or doesn't resolve to a loadable image.
// Used by the Group Type / Resource Type "Details" forms in the Model/
// ModelSource viewer — see plan.md "Icon propagation from model + entity
// data".
function efIconPreview(row) {
  var inp = row.querySelector('input') ;
  var img = document.createElement('img') ;
  img.className = 'eg-icon-preview' ; img.alt = '' ;
  img.style.marginLeft = '10px' ;
  img.onerror = function() { img.style.display = 'none' ; } ;
  function refresh() {
    var v = inp.value.trim() ;
    if (v) { img.style.display = '' ; img.src = v ; } else { img.style.display = 'none' ; }
  }
  refresh() ;
  inp.addEventListener('input', refresh) ;
  row.appendChild(img) ;
  return row ;
}

function efNum(id, label, value) {
  var row = document.createElement('div') ; row.className = 'editorField' ;
  var lbl = document.createElement('label') ; lbl.textContent = label + ':' ;
  var inp = document.createElement('input') ;
  inp.type = 'number' ; inp.id = id ; inp.min = '0' ; inp.className = 'editorInput' ;
  inp.value = (value !== undefined && value !== null) ? value : '' ;
  row.appendChild(lbl) ; row.appendChild(inp) ; return row ;
}

function ecb(id, label, checked) {
  var row = document.createElement('div') ; row.className = 'editorCheckRow' ;
  var cb = document.createElement('input') ; cb.type = 'checkbox' ; cb.id = id ; cb.checked = checked ;
  var lbl = document.createElement('label') ; lbl.textContent = label ; lbl.htmlFor = id ;
  row.appendChild(cb) ; row.appendChild(lbl) ; return row ;
}

function efBool(id, label, value) {
  var cell = document.createElement('div') ; cell.className = 'boolCell' ;
  var lbl = document.createElement('label') ; lbl.textContent = label + ':' ;
  var cur = (value === true) ? 'true' : (value === false) ? 'false' : '' ;

  // The segmented true/false/— pill only makes sense while actually
  // editable; in read-only mode there's nothing to click, so show the same
  // compact check/x badge used elsewhere for boolean values instead (an
  // em-dash badge for "unspecified", matching the pill's own '\u2014').
  if (_modelReadOnly) {
    var badge = (cur === '')
      ? boolBadgeEl(false, '\u2014')
      : boolBadgeEl(cur === 'true') ;
    badge.id = id ; badge.dataset.val = cur ;
    cell.appendChild(lbl) ; cell.appendChild(badge) ; return cell ;
  }

  var seg = document.createElement('div') ;
  seg.className = 'boolSeg' ;
  seg.id = id ; seg.dataset.val = cur ;
  var btns = [['true','true'],['false','false'],['\u2014','']] ;
  btns.forEach(function(b) {
    var btn = document.createElement('button') ; btn.type = 'button' ;
    btn.textContent = b[0] ; btn.className = 'boolSegBtn' + (cur === b[1] ? ' boolSegActive' : '') ;
    if (b[1] === '') btn.title = 'Unspecified' ;
    btn.onclick = function() {
      seg.dataset.val = b[1] ;
      seg.querySelectorAll('.boolSegBtn').forEach(function(x){ x.classList.remove('boolSegActive') ; }) ;
      btn.classList.add('boolSegActive') ;
      markDirty() ;
    } ;
    seg.appendChild(btn) ;
  }) ;
  cell.appendChild(lbl) ; cell.appendChild(seg) ; return cell ;
}

function fvBool(id) {
  var el = document.getElementById(id) ;
  if (!el) return null ;
  if (el.dataset.val === 'true') return true ;
  if (el.dataset.val === 'false') return false ;
  return null ;
}

function makeLabelsEditor(containerId, labels) {
  var sec = document.createElement('div') ;
  var hdr = document.createElement('div') ; hdr.className = 'editorSectionLabel' ;
  hdr.textContent = 'Labels' ; sec.appendChild(hdr) ;
  if (_modelReadOnly && Object.keys(labels).length === 0) {
    var none = document.createElement('span') ; none.textContent = '\u2014 none \u2014' ;
    none.style.cssText = 'color:#aaa;font-size:12px;font-style:italic;margin-left:4px;' ;
    sec.appendChild(none) ; return sec ;
  }
  var rowsDiv = document.createElement('div') ; rowsDiv.className = 'labelsRows' ;
  rowsDiv.id = containerId ;
  if (!_modelReadOnly) {
    var addBtn = document.createElement('button') ; addBtn.className = 'editorBtn editorBtnSmall' ;
    addBtn.textContent = '+ Add' ; addBtn.title = 'Add Label' ;
    addBtn.onclick = function() { rowsDiv.appendChild(makeLabelRow('','')) ; } ;
    hdr.appendChild(addBtn) ;
  }
  Object.keys(labels).forEach(function(k) { rowsDiv.appendChild(makeLabelRow(k, labels[k])) ; }) ;
  sec.appendChild(rowsDiv) ;
  return sec ;
}

function makeLabelRow(k, v) {
  var row = document.createElement('div') ; row.className = 'labelRow' ;
  var ki = document.createElement('input') ; ki.type = 'text' ; ki.placeholder = 'key' ;
  ki.value = k ; ki.className = 'editorInput labelKey' ;
  var vi = document.createElement('input') ; vi.type = 'text' ; vi.placeholder = 'value' ;
  vi.value = v ; vi.className = 'editorInput labelVal' ;
  var rb = document.createElement('button') ; rb.className = 'rmBtn' ; rb.textContent = 'Remove' ;
  rb.onclick = function() { confirmDel('this label', function() { row.remove() ; }) ; } ;
  row.appendChild(ki) ; row.appendChild(vi) ; row.appendChild(rb) ; return row ;
}

function collectLabels(containerId) {
  var container = document.getElementById(containerId) ; var labels = {} ;
  if (!container) return labels ;
  container.querySelectorAll('.labelRow').forEach(function(row) {
    var inputs = row.querySelectorAll('input') ;
    var k = inputs[0] ? inputs[0].value.trim() : '' ;
    var v = inputs[1] ? inputs[1].value.trim() : '' ;
    if (k) labels[k] = v ;
  }) ;
  return labels ;
}

function makeEnumEditor(containerId, values) {
  var sec = document.createElement('div') ;
  var hdr = document.createElement('div') ; hdr.className = 'editorSectionLabel' ;
  hdr.textContent = 'Enum' ; sec.appendChild(hdr) ;
  if (_modelReadOnly && values.length === 0) {
    var none = document.createElement('span') ; none.textContent = '\u2014 none \u2014' ;
    none.style.cssText = 'color:#aaa;font-size:12px;font-style:italic;margin-left:4px;' ;
    sec.appendChild(none) ; return sec ;
  }
  var rowsDiv = document.createElement('div') ; rowsDiv.id = containerId ;
  rowsDiv.style.cssText = 'display:flex;flex-direction:column;gap:4px;' ;
  if (!_modelReadOnly) {
    var addBtn = document.createElement('button') ; addBtn.className = 'editorBtn editorBtnSmall' ;
    addBtn.textContent = '+ Add' ; addBtn.title = 'Add enum value' ;
    addBtn.onclick = function() { rowsDiv.appendChild(makeEnumRow('')) ; } ;
    hdr.appendChild(addBtn) ;
  }
  values.forEach(function(v) { rowsDiv.appendChild(makeEnumRow(String(v))) ; }) ;
  sec.appendChild(rowsDiv) ;
  return sec ;
}

function makeEnumRow(val) {
  var row = document.createElement('div') ; row.className = 'labelRow' ;
  var inp = document.createElement('input') ; inp.type = 'text' ; inp.placeholder = 'value' ;
  inp.value = val ; inp.className = 'editorInput' ; inp.style.flex = '1' ;
  var rb = document.createElement('button') ; rb.className = 'rmBtn' ; rb.textContent = 'Remove' ;
  rb.onclick = function() { confirmDel('this enum value', function() { row.remove() ; }) ; } ;
  row.appendChild(inp) ; row.appendChild(rb) ; return row ;
}

function collectEnum(containerId) {
  var container = document.getElementById(containerId) ; var vals = [] ;
  if (!container) return vals ;
  container.querySelectorAll('.labelRow').forEach(function(row) {
    var inp = row.querySelector('input') ;
    var v = inp ? inp.value.trim() : '' ;
    if (v !== '') vals.push(v) ;
  }) ;
  return vals ;
}

function getScalarAttrNames(attrsObj) {
  var scalars = ['boolean','decimal','integer','string','timestamp','uinteger',
    'uri','uriabsolute','urirelative','uritemplate','url','urlabsolute','urlrelative','xid','xidtype'] ;
  var names = [] ;
  Object.keys(attrsObj||{}).forEach(function(k) {
    if (k === '*') return ;
    var a = attrsObj[k] ; if (!a) return ;
    if (scalars.indexOf(a.type||'string') !== -1) names.push(k) ;
  }) ;
  return names.sort() ;
}

function populateCstrAttrSel(attrSel, gk, resPlural, selectedVal) {
  while (attrSel.firstChild) attrSel.removeChild(attrSel.firstChild) ;
  var blank = document.createElement('option') ; blank.value = '' ; blank.textContent = '-- attribute --' ;
  attrSel.appendChild(blank) ;
  if (!resPlural) return ;
  var res = (((_modelData||{}).groups||{})[gk]||{}).resources||{} ;
  var rObj = res[resPlural] ;
  if (!rObj) return ;
  getScalarAttrNames(rObj.attributes).forEach(function(n) {
    var o = document.createElement('option') ; o.value = n ; o.textContent = n ;
    if (n === selectedVal) o.selected = true ;
    attrSel.appendChild(o) ;
  }) ;
}

function makeConstraintsEditor(containerId, constraints, gk) {
  var sec = document.createElement('div') ;
  var hdr = document.createElement('div') ; hdr.className = 'editorSectionLabel' ;
  hdr.textContent = 'Constraints' ; sec.appendChild(hdr) ;
  var blocksDiv = document.createElement('div') ; blocksDiv.id = containerId ;
  blocksDiv.style.cssText = 'display:flex;flex-direction:column;' ;
  if (!_modelReadOnly) {
    var addBtn = document.createElement('button') ; addBtn.className = 'editorBtn editorBtnSmall' ;
    addBtn.textContent = '+ Add' ; addBtn.title = 'Add Constraint' ;
    addBtn.onclick = function() {
      var row = makeConstraintRow('', {}, gk) ;
      blocksDiv.insertBefore(row, blocksDiv.firstChild) ;
      row.scrollIntoView({block:'nearest'}) ;
    } ;
    hdr.appendChild(addBtn) ;
  }
  if (_modelReadOnly && Object.keys(constraints||{}).length === 0) {
    var none = document.createElement('span') ; none.textContent = '\u2014 none \u2014' ;
    none.style.cssText = 'color:#aaa;font-size:12px;font-style:italic;margin-left:4px;' ;
    sec.appendChild(none) ; return sec ;
  }
  Object.keys(constraints||{}).forEach(function(k) {
    blocksDiv.appendChild(makeConstraintRow(k, constraints[k]||{}, gk)) ;
  }) ;
  sec.appendChild(blocksDiv) ;
  return sec ;
}

function makeConstraintRow(key, constraint, gk) {
  var idx = _cstrCounter++ ;
  var block = document.createElement('div') ; block.className = 'constraintBlock' ;
  block.dataset.cstrIdx = String(idx) ;
  block.dataset.origKey = key ; // preserve original key as fallback if selects can't resolve

  // Header row with title and Remove button
  var blockHdr = document.createElement('div') ; blockHdr.className = 'constraintBlockHdr' ;
  var titleSpan = document.createElement('span') ; titleSpan.className = 'constraintBlockTitle' ;
  titleSpan.textContent = key || 'New Constraint' ; blockHdr.appendChild(titleSpan) ;
  if (!_modelReadOnly) {
    var rb = document.createElement('button') ; rb.className = 'rmBtn' ; rb.textContent = 'Remove' ;
    rb.onclick = function() { confirmDel('"' + (titleSpan.textContent || 'this constraint') + '"', function() { block.remove() ; }) ; } ;
    blockHdr.appendChild(rb) ;
  }
  block.appendChild(blockHdr) ;

  // Parse key into resPlural + attrName
  var dotIdx = key.indexOf('.') ;
  var initRes = dotIdx !== -1 ? key.substring(0, dotIdx) : key ;
  var initAttr = dotIdx !== -1 ? key.substring(dotIdx+1) : '' ;

  if (_modelReadOnly) {
    // Read-only: show fields as text. Note: the "Path" isn't repeated here
    // since it's already shown as this constraint block's title (see
    // titleSpan above) — showing it again in the body would be redundant.
    function roRow(lbl, val) {
      var row = document.createElement('div') ; row.className = 'editorField' ;
      var l = document.createElement('label') ; l.textContent = lbl ;
      var s = document.createElement('span') ; s.style.cssText = 'font-size:13px;color:#333;' ;
      s.textContent = val || '\u2014' ; row.appendChild(l) ; row.appendChild(s) ; return row ;
    }
    var defStr = constraint.default !== undefined ? JSON.stringify(constraint.default) : '' ;
    if (defStr) block.appendChild(roRow('Default:', defStr)) ;
    if (constraint.equals) block.appendChild(roRow('Equals:', gk + '.' + constraint.equals)) ;
    var enumArr = Array.isArray(constraint.enum) ? constraint.enum : [] ;
    if (enumArr.length) block.appendChild(roRow('Enum:', enumArr.join(', '))) ;
    if (!defStr && !constraint.equals && !enumArr.length) {
      var noneSpan = document.createElement('span') ;
      noneSpan.textContent = '\u2014 no constraint data set \u2014' ;
      noneSpan.style.cssText = 'color:#aaa;font-size:12px;font-style:italic;' ;
      block.appendChild(noneSpan) ;
    }
    return block ;
  }

  // Edit mode: path row with two selects
  var pathRow = document.createElement('div') ; pathRow.className = 'cstrPathRow' ;
  var pathLbl = document.createElement('label') ;
  pathLbl.appendChild(document.createTextNode('Path:')) ;
  var pathReq = document.createElement('span') ; pathReq.textContent = ' *' ; pathReq.style.cssText = 'color:#c00;font-weight:bold;' ;
  pathLbl.appendChild(pathReq) ;
  pathRow.appendChild(pathLbl) ;

  var resSel = document.createElement('select') ; resSel.className = 'cstrResSel editorSelectWrap editorInput' ;
  var resBlank = document.createElement('option') ; resBlank.value = '' ; resBlank.textContent = '-- resource --' ;
  resSel.appendChild(resBlank) ;
  var res = (((_modelData||{}).groups||{})[gk]||{}).resources||{} ;
  Object.keys(res).sort().forEach(function(rk) {
    var o = document.createElement('option') ; o.value = rk ; o.textContent = rk ;
    if (rk === initRes) o.selected = true ;
    resSel.appendChild(o) ;
  }) ;

  var dot = document.createElement('span') ; dot.className = 'cstrPathDot' ; dot.textContent = '.' ;

  var attrSel = document.createElement('select') ; attrSel.className = 'cstrAttrSel editorSelectWrap editorInput' ;
  populateCstrAttrSel(attrSel, gk, initRes, initAttr) ;

  resSel.onchange = function() {
    var newRes = resSel.value ;
    titleSpan.textContent = newRes ? (newRes + '.' + (attrSel.value||'?')) : 'New Constraint' ;
    populateCstrAttrSel(attrSel, gk, newRes, '') ;
  } ;
  attrSel.onchange = function() {
    var r = resSel.value ; var a = attrSel.value ;
    titleSpan.textContent = (r && a) ? (r + '.' + a) : (r ? r + '.?' : 'New Constraint') ;
  } ;

  var pathValueWrap = document.createElement('div') ; pathValueWrap.className = 'cstrPathValue' ;
  pathValueWrap.appendChild(resSel) ; pathValueWrap.appendChild(dot) ; pathValueWrap.appendChild(attrSel) ;
  pathRow.appendChild(pathValueWrap) ;
  block.appendChild(pathRow) ;

  // Default field
  var defRow = document.createElement('div') ; defRow.className = 'editorField' ;
  var defLbl = document.createElement('label') ; defLbl.textContent = 'Default:' ;
  var defInp = document.createElement('input') ; defInp.type = 'text' ; defInp.className = 'cstrDef editorInput' ;
  defInp.placeholder = 'default value' ;
  defInp.value = constraint.default !== undefined ? JSON.stringify(constraint.default) : '' ;
  defRow.appendChild(defLbl) ; defRow.appendChild(defInp) ;
  block.appendChild(defRow) ;

  // Equals field — dropdown of group scalar attrs
  var eqRow = document.createElement('div') ; eqRow.className = 'editorField' ;
  var eqLbl = document.createElement('label') ; eqLbl.textContent = 'Equals:' ;
  var eqPrefix = document.createElement('span') ; eqPrefix.className = 'cstrEqPrefix' ;
  eqPrefix.textContent = gk + '.' ;
  var eqSel = document.createElement('select') ; eqSel.className = 'cstrEqSel editorSelectWrap editorInput' ;
  var eqBlank = document.createElement('option') ; eqBlank.value = '' ; eqBlank.textContent = '-- none --' ;
  eqSel.appendChild(eqBlank) ;
  var grpAttrs = getScalarAttrNames((((_modelData||{}).groups||{})[gk]||{}).attributes||{}) ;
  grpAttrs.forEach(function(n) {
    var o = document.createElement('option') ; o.value = n ; o.textContent = n ;
    if (n === (constraint.equals||'')) o.selected = true ;
    eqSel.appendChild(o) ;
  }) ;
  eqRow.appendChild(eqLbl) ; eqRow.appendChild(eqPrefix) ; eqRow.appendChild(eqSel) ;
  block.appendChild(eqRow) ;

  // Enum editor
  var enumSec = document.createElement('div') ; enumSec.className = 'cstrEnumSection' ;
  var enumHdr = document.createElement('div') ; enumHdr.className = 'editorSectionLabel' ;
  enumHdr.textContent = 'Enum' ; enumSec.appendChild(enumHdr) ;
  var enumRowsDiv = document.createElement('div') ; enumRowsDiv.id = 'cstr_enum_' + idx ;
  enumRowsDiv.style.cssText = 'display:flex;flex-direction:column;gap:4px;' ;
  var enumAddBtn = document.createElement('button') ; enumAddBtn.className = 'editorBtn editorBtnSmall' ;
  enumAddBtn.textContent = '+ Add' ; enumAddBtn.title = 'Add enum value' ;
  enumAddBtn.onclick = function() { enumRowsDiv.appendChild(makeEnumRow('')) ; } ;
  enumHdr.appendChild(enumAddBtn) ;
  var enumArr2 = Array.isArray(constraint.enum) ? constraint.enum : [] ;
  enumArr2.forEach(function(v) { enumRowsDiv.appendChild(makeEnumRow(String(v))) ; }) ;
  enumSec.appendChild(enumRowsDiv) ; block.appendChild(enumSec) ;

  return block ;
}

function collectConstraints(containerId) {
  var container = document.getElementById(containerId) ; var constraints = {} ;
  if (!container) return constraints ;
  container.querySelectorAll('.constraintBlock').forEach(function(block) {
    var resSel = block.querySelector('.cstrResSel') ;
    var attrSel = block.querySelector('.cstrAttrSel') ;
    var resVal = resSel ? resSel.value.trim() : '' ;
    var attrVal = attrSel ? attrSel.value.trim() : '' ;
    var key = (resVal && attrVal) ? (resVal + '.' + attrVal) : (block.dataset.origKey || '') ;
    if (!key) return ; // truly new/empty constraint with no path — skip
    var c = {} ;
    var defInp = block.querySelector('.cstrDef') ;
    var defVal = defInp ? defInp.value.trim() : '' ;
    if (defVal !== '') { try { c.default = JSON.parse(defVal) ; } catch(e) { c.default = defVal ; } }
    var eqSel = block.querySelector('.cstrEqSel') ;
    var eq = eqSel ? eqSel.value.trim() : '' ;
    if (eq) c.equals = eq ;
    var enumDiv = document.getElementById('cstr_enum_' + block.dataset.cstrIdx) ;
    if (enumDiv) {
      var vals = [] ;
      enumDiv.querySelectorAll('.labelRow input').forEach(function(inp) {
        var v = inp.value.trim() ; if (v) vals.push(v) ;
      }) ;
      if (vals.length) c.enum = vals ;
    }
    constraints[key] = c ;
  }) ;
  return constraints ;
}

function fv(id) {
  var el = document.getElementById(id) ; if (!el) return '' ;
  return el.value !== undefined ? el.value.trim() : '' ;
}

// ---- Save / Undo / ReadOnly ----

function undoModel() {
  _modelDirty = false ;
  _modelData = deepClone(_modelSrc) ;
  _navPath = [] ; _navSelected = null ; renderEditor() ;
}

function applyReadOnly(container) {
  container.querySelectorAll('input, select, textarea').forEach(function(el) { el.disabled = true ; }) ;
  container.querySelectorAll('.editorBtn:not(.navDrillBtn), .rmBtn, .navItemAdd, .navItemDel').forEach(function(el) { el.style.display = 'none' ; }) ;
}

function saveModel(onSuccess) {
  collectCurrentEditor() ;
  var model = _modelData || {} ;
  var errDiv = document.getElementById('editorError') ;
  if (errDiv) { hideErrorBanner(errDiv) ; }

  // Show blocking overlay while PUT is in flight
  var overlay = document.createElement('div') ; overlay.className = 'savingOverlay' ;
  var box = document.createElement('div') ; box.className = 'savingBox' ;
  var spinner = document.createElement('div') ; spinner.className = 'savingSpinner' ;
  var msg = document.createElement('div') ; msg.textContent = 'Saving\u2026 validating registry' ;
  box.appendChild(spinner) ; box.appendChild(msg) ; overlay.appendChild(box) ;
  document.body.appendChild(overlay) ;
  function removeOverlay() { if (overlay.parentNode) overlay.parentNode.removeChild(overlay) ; }

  fetch(_modelPutURL, {
    method: 'PUT',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(model, null, 2)
  }).then(function(resp) {
    return resp.text().then(function(text) {
      removeOverlay() ;
      if (resp.ok) {
        _modelDirty = false ;
        _modelSrc = deepClone(_modelData) ;
        // Saving modelsource regenerates the derived /model view, but the PUT
        // response here is the modelsource doc, not /model — so _modelCache
        // (used for entity-type resolution, tile descriptions, JSON left
        // panel, etc. while browsing data) is now stale. Refresh it with a
        // fresh GET /model before continuing.
        var svBaseSave = (_state.serverURL || window.location.origin).replace(/\/$/, '') ;
        var mKey = normalizeURL(svBaseSave) ;
        invalidateRegistryProbe(svBaseSave) ;
        fetch(serverFetchBase(svBaseSave) + '/model')
          .then(function(r) { return r.json() ; })
          .then(function(m) { _modelCache[mKey] = m ; })
          .catch(function()  { delete _modelCache[mKey] ; })
          .then(function() {
            if (onSuccess) { onSuccess() ; } else { window.location.reload() ; }
          }) ;
      } else { if (errDiv) { showErrorBanner(errDiv, 'Error (' + resp.status + '):\n' + text) ; } }
    }) ;
  }).catch(function(err) {
    removeOverlay() ;
    if (errDiv) { showErrorBanner(errDiv, 'Network error: ' + err.message) ; }
  }) ;
}

// ---- Capabilities Editor ---------------------------------------------------
//
// Renders a human-readable browse+edit form for a registry's /capabilities
// document. Entry point is renderCapabilitiesEditor(data, offered), called
// from refresh()/setDataView() when _state.section === 'capabilities' and
// _state.dataView !== 'json'. `offered` is the /capabilitiesoffered document
// (may be null if it failed to load — extension capabilities then fall back
// to a raw read-only JSON blob so nothing is silently dropped).
//
// Read-only vs editable is driven by _state.editMode (shared header pencil
// button); editing is enabled when available.capabilities.mutable is true
// (see refresh()). Saves always PUT to /capabilities.

var _capPutURL    = '';
var _capReadOnly  = true;   // runtime: true unless _state.editMode
var _capSrc       = null;   // pristine copy of last-loaded capabilities (for undo)
var _capData      = null;   // working copy being edited/viewed
var _capDirty     = false;
var _capLoadedFor = null;   // "serverURL|section" key used to detect a fresh load
var _capOffered   = null;   // /capabilitiesoffered doc for the currently-loaded server

// Known top-level Capabilities keys (per common/capabilities.go). Anything
// else found in the doc is treated as an extension capability.
var _capKnownKeys = ['available', 'compatibilities', 'flags', 'formats',
  'ignores', 'mutable', 'pagination', 'shortself', 'specversions', 'versionmodes'];

function renderCapabilitiesEditor(data, offered) {
  var main = el('main-view');
  var key = normalizeURL(_state.serverURL || window.location.origin) + '|' + _state.section;
  if (_capLoadedFor !== key) {
    _capSrc  = deepClone(data);
    _capData = deepClone(_capSrc);
    _capDirty = false;
    _capLoadedFor = key;
  }
  _capOffered  = offered || null;
  _capPutURL   = buildAPIURL();
  _capReadOnly = !_state.editMode;
  main.innerHTML = '<div id="capEditor"></div>';
  renderCapEditor();
}

function markCapDirty() {
  if (!_capDirty) {
    _capDirty = true;
    var sb = document.getElementById('capSaveBtn'); if (sb) sb.disabled = false;
    var ub = document.getElementById('capUndoBtn'); if (ub) ub.disabled = false;
  }
}

function undoCapabilities() {
  _capDirty = false;
  _capData = deepClone(_capSrc);
  renderCapEditor();
}

function saveCapabilities(onSuccess) {
  var errDiv = document.getElementById('capEditorError');
  if (errDiv) { hideErrorBanner(errDiv); }

  var overlay = document.createElement('div'); overlay.className = 'savingOverlay';
  var box = document.createElement('div'); box.className = 'savingBox';
  var spinner = document.createElement('div'); spinner.className = 'savingSpinner';
  var msg = document.createElement('div'); msg.textContent = 'Saving\u2026 validating capabilities';
  box.appendChild(spinner); box.appendChild(msg); overlay.appendChild(box);
  document.body.appendChild(overlay);
  function removeOverlay() { if (overlay.parentNode) overlay.parentNode.removeChild(overlay); }

  fetch(_capPutURL, {
    method: 'PUT',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(_capData, null, 2)
  }).then(function(resp) {
    return resp.text().then(function(text) {
      removeOverlay();
      if (resp.ok) {
        // The PUT response body is the server's canonical, re-validated
        // capabilities doc (see HTTPPUTCapabilities -> HTTPGETCapabilities).
        // Use it (falling back to what we sent if parsing fails) as the new
        // pristine copy, and refresh _capCache for this server so anything
        // relying on it (Registry Endpoints list, Model/Model Source
        // edit-enabled checks, etc.) immediately reflects the new
        // mutability/settings instead of stale cached data.
        var updated = null;
        try { updated = JSON.parse(text); } catch (e) { /* fall back below */ }
        _capDirty = false;
        _capSrc  = updated ? deepClone(updated) : deepClone(_capData);
        _capData = deepClone(_capSrc);
        var svBaseSave = (_state.serverURL || window.location.origin).replace(/\/$/, '');
        _capCache[normalizeURL(svBaseSave)] = deepClone(_capSrc);
        invalidateRegistryProbe(svBaseSave);
        if (onSuccess) { onSuccess(); } else { window.location.reload(); }
      } else {
        if (errDiv) { showErrorBanner(errDiv, 'Error (' + resp.status + '):\n' + text); }
      }
    });
  }).catch(function(err) {
    removeOverlay();
    if (errDiv) { showErrorBanner(errDiv, 'Network error: ' + err.message); }
  });
}

// Look up the OfferedCapability node for a top-level key (used to know type/
// enum/attributes for rendering). Returns null if not offered / unknown.
function capOfferedNode(key) {
  return (_capOffered && _capOffered[key]) || null;
}

// True if this offered node's value is locked to a single enum value (e.g.
// {type:'boolean', enum:[false]}) — such fields are always shown read-only,
// even while the page is in edit mode.
function capNodeLocked(node) {
  return !!(node && Array.isArray(node.enum) && node.enum.length === 1);
}

function capSectionEl(title) {
  var sec = document.createElement('div'); sec.className = 'capSection';
  var hdr = document.createElement('div'); hdr.className = 'capSectionTitle';
  hdr.textContent = title; sec.appendChild(hdr);
  var body = document.createElement('div'); body.className = 'capSectionBody';
  sec.appendChild(body);
  return {sec: sec, body: body};
}

// Two-state (true/false) toggle, styled like the Model editor's boolSeg
// control. `locked` forces read-only regardless of _capReadOnly (used when
// the offered-capabilities schema pins this field to a single enum value,
// so it could never actually be switched even while editing).
//
// The segmented true/false pill only makes sense while the value is truly
// editable — i.e. edit mode is on AND it isn't locked to one value. In every
// other case (plain read-only viewing, or a locked field) there's nothing
// to click, so show the same compact check/x badge used for boolean
// Property-table values elsewhere (isdefault, formatvalidated, etc.)
// instead of a pill that looks interactive but isn't.
function capBoolToggle(value, locked, onChange) {
  var readOnly = _capReadOnly || locked;
  if (readOnly) return boolBadgeEl(!!value);

  var seg = document.createElement('div');
  seg.className = 'boolSeg';
  var cur = value === true ? 'true' : 'false';
  seg.dataset.val = cur;
  [['true', 'true'], ['false', 'false']].forEach(function(b) {
    var btn = document.createElement('button'); btn.type = 'button';
    btn.textContent = b[0];
    btn.className = 'boolSegBtn' + (cur === b[1] ? ' boolSegActive' : '');
    btn.onclick = function() {
      seg.dataset.val = b[1];
      seg.querySelectorAll('.boolSegBtn').forEach(function(x) { x.classList.remove('boolSegActive'); });
      btn.classList.add('boolSegActive');
      markCapDirty();
      if (onChange) onChange(b[1] === 'true');
    };
    seg.appendChild(btn);
  });
  return seg;
}

// Editable chip list for a string array. `enumOptions` (optional) constrains
// what can be added via a <select>; otherwise a free-text input is used.
// onChange(newArray) fires whenever the list changes.
function capChipList(values, enumOptions, onChange) {
  var wrap = document.createElement('div'); wrap.className = 'capChipWrap';
  var chipsDiv = document.createElement('div'); chipsDiv.className = 'capChips';

  function currentValues() {
    return Array.prototype.slice.call(chipsDiv.querySelectorAll('.capChip')).map(function(c) {
      return c.dataset.val;
    });
  }
  function fireChange() { markCapDirty(); if (onChange) onChange(currentValues()); }

  function addChip(v) {
    var chip = document.createElement('span'); chip.className = 'capChip'; chip.dataset.val = v;
    var txt = document.createElement('span'); txt.textContent = v; chip.appendChild(txt);
    if (!_capReadOnly) {
      var rm = document.createElement('span'); rm.className = 'capChipRemove'; rm.textContent = '\u00d7';
      rm.title = 'Remove';
      rm.onclick = function() {
        chip.remove(); fireChange();
        var opt = wrap.querySelector('option[value="' + CSS.escape(v) + '"]');
        if (opt) opt.disabled = false;
      };
      chip.appendChild(rm);
    }
    chipsDiv.appendChild(chip);
  }
  (values || []).forEach(addChip);
  wrap.appendChild(chipsDiv);

  if (!_capReadOnly) {
    var addRow = document.createElement('div'); addRow.className = 'capChipAddRow';
    if (enumOptions && enumOptions.length) {
      var sel = document.createElement('select'); sel.className = 'editorInput';
      var placeholder = document.createElement('option');
      placeholder.value = ''; placeholder.textContent = '-- add --';
      sel.appendChild(placeholder);
      var already = currentValues();
      enumOptions.forEach(function(o) {
        var opt = document.createElement('option'); opt.value = o; opt.textContent = o;
        if (already.indexOf(o) !== -1) opt.disabled = true;
        sel.appendChild(opt);
      });
      sel.onchange = function() {
        var v = sel.value;
        if (v && currentValues().indexOf(v) === -1) {
          addChip(v); fireChange();
          var opt = sel.querySelector('option[value="' + CSS.escape(v) + '"]');
          if (opt) opt.disabled = true;
        }
        sel.value = '';
      };
      addRow.appendChild(sel);
    } else {
      var inp = document.createElement('input'); inp.type = 'text'; inp.className = 'editorInput';
      inp.placeholder = 'add value, then Enter';
      inp.onkeydown = function(e) {
        if (e.key === 'Enter') {
          e.preventDefault();
          var v = inp.value.trim();
          if (v && currentValues().indexOf(v) === -1) { addChip(v); fireChange(); }
          inp.value = '';
        }
      };
      addRow.appendChild(inp);
    }
    wrap.appendChild(addRow);
  } else if (!(values || []).length) {
    var none = document.createElement('span'); none.className = 'capNone'; none.textContent = '\u2014 none \u2014';
    wrap.appendChild(none);
  }
  return wrap;
}

function capLabelRow(label, valueEl) {
  var row = document.createElement('div'); row.className = 'capFieldRow';
  var lbl = document.createElement('label'); lbl.textContent = label + ':';
  row.appendChild(lbl); row.appendChild(valueEl);
  return row;
}

// Generic recursive renderer for extension capabilities, driven by the
// matching OfferedCapability node's declared type. Falls back to a raw
// read-only JSON blob when the type is unknown/unavailable so nothing is
// ever silently dropped.
function renderCapValueGeneric(value, node, onChange) {
  var type = node && node.type;
  if (type === 'boolean') {
    return capBoolToggle(!!value, capNodeLocked(node), onChange);
  }
  if (type === 'array') {
    return capChipList((value || []).map(String), node && node.enum, onChange);
  }
  if (type === 'object') {
    var box = document.createElement('div'); box.className = 'capObjectBox';
    var attrs = (node && node.attributes) || {};
    var obj = value && typeof value === 'object' ? value : {};
    Object.keys(attrs).forEach(function(k) {
      var childNode = attrs[k];
      var childEl = renderCapValueGeneric(obj[k], childNode, function(nv) { obj[k] = nv; });
      box.appendChild(capLabelRow(k, childEl));
    });
    if (onChange) onChange(obj);
    return box;
  }
  if (type === 'map') {
    var mbox = document.createElement('div'); mbox.className = 'capMapBox';
    var itemNode = node && node.item;
    var mval = value && typeof value === 'object' ? value : {};
    function renderRows() {
      mbox.innerHTML = '';
      Object.keys(mval).forEach(function(k) {
        var rowDiv = document.createElement('div'); rowDiv.className = 'capMapRow';
        var kLbl = document.createElement('span'); kLbl.className = 'capMapKey'; kLbl.textContent = k;
        rowDiv.appendChild(kLbl);
        var childEl = renderCapValueGeneric(mval[k], itemNode, function(nv) { mval[k] = nv; });
        rowDiv.appendChild(childEl);
        if (!_capReadOnly) {
          var rm = document.createElement('button'); rm.className = 'rmBtn'; rm.textContent = 'Remove';
          rm.onclick = function() { delete mval[k]; markCapDirty(); if (onChange) onChange(mval); renderRows(); };
          rowDiv.appendChild(rm);
        }
        mbox.appendChild(rowDiv);
      });
      if (!_capReadOnly) {
        var addRow = document.createElement('div'); addRow.className = 'capMapAddRow';
        var kInp = document.createElement('input'); kInp.type = 'text'; kInp.className = 'editorInput';
        kInp.placeholder = 'new key';
        var addBtn = document.createElement('button'); addBtn.className = 'editorBtn'; addBtn.textContent = '+ Add';
        addBtn.onclick = function() {
          var k = kInp.value.trim();
          if (!k || mval.hasOwnProperty(k)) return;
          mval[k] = (itemNode && itemNode.type === 'array') ? [] : (itemNode && itemNode.type === 'boolean') ? false : (itemNode && itemNode.type === 'object') ? {} : '';
          markCapDirty(); if (onChange) onChange(mval); renderRows();
        };
        addRow.appendChild(kInp); addRow.appendChild(addBtn);
        mbox.appendChild(addRow);
      }
    }
    renderRows();
    return mbox;
  }
  if (type === 'string' || type === 'integer' || type === 'decimal') {
    if (node && Array.isArray(node.enum) && node.enum.length) {
      var sel2 = document.createElement('select'); sel2.className = 'editorInput';
      node.enum.forEach(function(o) {
        var opt = document.createElement('option'); opt.value = o; opt.textContent = String(o);
        if (String(value) === String(o)) opt.selected = true;
        sel2.appendChild(opt);
      });
      sel2.disabled = _capReadOnly || capNodeLocked(node);
      sel2.onchange = function() { markCapDirty(); if (onChange) onChange(sel2.value); };
      return sel2;
    }
    var inp2 = document.createElement('input'); inp2.type = 'text'; inp2.className = 'editorInput';
    inp2.value = value !== undefined && value !== null ? String(value) : '';
    inp2.disabled = _capReadOnly || capNodeLocked(node);
    inp2.onchange = function() { markCapDirty(); if (onChange) onChange(inp2.value); };
    return inp2;
  }
  // Unknown/unrecognized type, or no offered info available — raw read-only
  // fallback so the value is at least visible and never silently dropped.
  var pre = document.createElement('pre'); pre.className = 'capRawFallback';
  pre.textContent = JSON.stringify(value, null, 2);
  return pre;
}

function renderCapEditor() {
  var host = document.getElementById('capEditor');
  if (!host) return;
  host.innerHTML = '';
  var data = _capData || {};

  // Server URL line (no page title on this view, so show it just below
  // the breadcrumb bar instead, above the section boxes).
  host.insertAdjacentHTML('beforeend', serverURLLineHtml()) ;

  // Action bar (Save/Undo) — only meaningful while editing.
  if (_state.editMode) {
    var bar = document.createElement('div'); bar.className = 'editorActionBar';
    var saveBtn = document.createElement('button'); saveBtn.className = 'editorBtn'; saveBtn.id = 'capSaveBtn';
    saveBtn.textContent = 'Save'; saveBtn.disabled = !_capDirty;
    saveBtn.onclick = function() { saveCapabilities(function() { toggleEdit(); }); };
    var undoBtn = document.createElement('button'); undoBtn.className = 'editorBtn'; undoBtn.id = 'capUndoBtn';
    undoBtn.textContent = 'Undo'; undoBtn.disabled = !_capDirty;
    undoBtn.onclick = function() { undoCapabilities(); };
    bar.appendChild(saveBtn); bar.appendChild(undoBtn);
    host.appendChild(bar);

    var errDiv = document.createElement('div'); errDiv.id = 'capEditorError'; errDiv.className = 'error-banner';
    errDiv.style.display = 'none';
    host.appendChild(errDiv);
  }

  var body = document.createElement('div'); body.className = 'capBody';
  host.appendChild(body);

  // ---- Available Entities ----
  var availSec = capSectionEl('Available Entities');
  var availTable = document.createElement('table'); availTable.className = 'capTable';
  data.available = data.available || {};
  Object.keys(data.available).sort().forEach(function(entName) {
    var entObj = data.available[entName];
    var entOffered = capOfferedNode('available');
    var entNode = entOffered && entOffered.attributes && entOffered.attributes[entName];
    var mutNode = entNode && entNode.attributes && entNode.attributes.mutable;
    var tr = document.createElement('tr');
    var tdName = document.createElement('td'); tdName.textContent = entName;
    var tdMut = document.createElement('td');
    tdMut.appendChild(capBoolToggle(!!entObj.mutable, capNodeLocked(mutNode), function(nv) { entObj.mutable = nv; }));
    tr.appendChild(tdName); tr.appendChild(tdMut);
    availTable.appendChild(tr);
  });
  var availThead = document.createElement('tr');
  ['Entity', 'Mutable'].forEach(function(h) { var th = document.createElement('th'); th.textContent = h; availThead.appendChild(th); });
  availTable.insertBefore(availThead, availTable.firstChild);
  availSec.body.appendChild(availTable);
  body.appendChild(availSec.sec);

  // ---- Compatibility ----
  var compatSec = capSectionEl('Compatibility');
  data.compatibilities = data.compatibilities || {};
  var compatOffered = capOfferedNode('compatibilities');
  var compatTable = document.createElement('table'); compatTable.className = 'capTable';
  var compatThead = document.createElement('tr');
  ['Format', 'Compatibility Modes'].forEach(function(h) { var th = document.createElement('th'); th.textContent = h; compatThead.appendChild(th); });
  compatTable.appendChild(compatThead);
  Object.keys(data.compatibilities).sort().forEach(function(fmt) {
    var fmtNode = compatOffered && compatOffered.attributes && compatOffered.attributes[fmt];
    var tr = document.createElement('tr');
    var tdFmt = document.createElement('td'); tdFmt.textContent = fmt;
    var tdModes = document.createElement('td');
    tdModes.appendChild(capChipList(data.compatibilities[fmt], fmtNode && fmtNode.enum, function(nv) { data.compatibilities[fmt] = nv; }));
    tr.appendChild(tdFmt); tr.appendChild(tdModes);
    compatTable.appendChild(tr);
  });
  compatSec.body.appendChild(compatTable);
  // Allow adding a compatibility row for any currently-enabled format not yet listed
  // (compatibilities keys must be a subset of `formats`, per spec/server validation).
  if (!_capReadOnly) {
    var availFormats = (data.formats || []).filter(function(f) { return !data.compatibilities.hasOwnProperty(f); });
    if (availFormats.length) {
      var addCompatRow = document.createElement('div'); addCompatRow.className = 'capChipAddRow';
      var fmtSel = document.createElement('select'); fmtSel.className = 'editorInput';
      var ph = document.createElement('option'); ph.value = ''; ph.textContent = '-- add format --';
      fmtSel.appendChild(ph);
      availFormats.forEach(function(f) { var opt = document.createElement('option'); opt.value = f; opt.textContent = f; fmtSel.appendChild(opt); });
      fmtSel.onchange = function() {
        var f = fmtSel.value;
        if (f) { data.compatibilities[f] = []; markCapDirty(); renderCapEditor(); }
      };
      addCompatRow.appendChild(fmtSel);
      compatSec.body.appendChild(addCompatRow);
    }
  }
  body.appendChild(compatSec.sec);

  // ---- Flags / Formats / Ignores / Spec Versions / Version Modes ----
  var listsSec = capSectionEl('Capabilities');
  [
    ['flags', 'Flags'], ['formats', 'Formats'], ['ignores', 'Ignores'],
    ['specversions', 'Spec Versions'], ['versionmodes', 'Version Modes']
  ].forEach(function(pair) {
    var k = pair[0], label = pair[1];
    var node = capOfferedNode(k);
    var chipEl = capChipList(data[k] || [], node && node.enum, function(nv) { data[k] = nv; });
    listsSec.body.appendChild(capLabelRow(label, chipEl));
  });
  body.appendChild(listsSec.sec);

  // ---- Settings ----
  var settingsSec = capSectionEl('Settings');
  ['pagination', 'shortself'].forEach(function(k) {
    var node = capOfferedNode(k);
    var toggle = capBoolToggle(!!data[k], capNodeLocked(node), function(nv) { data[k] = nv; });
    settingsSec.body.appendChild(capLabelRow(k === 'pagination' ? 'Pagination' : 'Short Self', toggle));
  });
  body.appendChild(settingsSec.sec);

  // ---- Extensions ----
  var extraKeys = Object.keys(data).filter(function(k) { return _capKnownKeys.indexOf(k) === -1; });
  if (extraKeys.length || !_capReadOnly) {
    var extSec = capSectionEl('Extensions');
    if (!extraKeys.length) {
      var none2 = document.createElement('span'); none2.className = 'capNone'; none2.textContent = '\u2014 none \u2014';
      extSec.body.appendChild(none2);
    }
    extraKeys.sort().forEach(function(k) {
      var node = capOfferedNode(k);
      var valueEl = renderCapValueGeneric(data[k], node, function(nv) { data[k] = nv; });
      extSec.body.appendChild(capLabelRow(k, valueEl));
    });
    body.appendChild(extSec.sec);
  }
}

// Renders a read-only, human-readable List view for a registry's
// /capabilitiesoffered document. Unlike /capabilities, this document is
// itself a schema description (each entry looks like {type, attributes} /
// {type, enum, item} / etc. — see common/capabilities.go's
// OfferedCapability shape) rather than a set of actual values, so it gets
// its own recursive schema-node renderer (renderCapSchemaNode) instead of
// reusing renderCapValueGeneric (which pairs a *value* with its offered
// node). Always read-only — capabilitiesoffered is a server-declared
// document, never user-edited (see plan.md "Capabilities/CapabilitiesOffered
// List view").
function renderCapSchemaNode(node) {
  if (!node || typeof node !== 'object') {
    var pre = document.createElement('pre'); pre.className = 'capRawFallback';
    pre.textContent = JSON.stringify(node, null, 2);
    return pre;
  }
  var type = node.type || '';
  if (type === 'object' && node.attributes) {
    var box = document.createElement('div'); box.className = 'capObjectBox';
    Object.keys(node.attributes).sort().forEach(function(k) {
      box.appendChild(capLabelRow(k, renderCapSchemaNode(node.attributes[k])));
    });
    return box;
  }
  // Array / map with a nested item schema (e.g. compatibilities' array-of-
  // string-enum entries) — show the item type plus allowed values as chips.
  // Note: the enum of allowed values lives on the array/map node itself
  // (node.enum), not on node.item — node.item only carries the item's type.
  if ((type === 'array' || type === 'map') && node.item) {
    var wrap = document.createElement('div'); wrap.className = 'capSchemaWrap';
    var typeLbl = document.createElement('span'); typeLbl.className = 'capSchemaType';
    typeLbl.textContent = type + ' of ' + (node.item.type || '?');
    wrap.appendChild(typeLbl);
    var enumVals = Array.isArray(node.enum) ? node.enum : node.item.enum;
    if (Array.isArray(enumVals) && enumVals.length) {
      wrap.appendChild(capChipList(enumVals, null, null));
    }
    return wrap;
  }
  // Leaf: plain type, optionally constrained to an enum of allowed values.
  var leaf = document.createElement('div'); leaf.className = 'capSchemaWrap';
  var lbl = document.createElement('span'); lbl.className = 'capSchemaType';
  lbl.textContent = type || 'any';
  leaf.appendChild(lbl);
  if (Array.isArray(node.enum) && node.enum.length) {
    leaf.appendChild(capChipList(node.enum, null, null));
  }
  return leaf;
}

// Display labels for the "Other Capabilities" merged table (see
// renderCapabilitiesOfferedViewer) — same names used for these keys
// elsewhere (e.g. the actual /capabilities editor's Capabilities/Settings
// sections). Any key not listed here (a future/unknown addition) just
// falls back to its raw key name, so nothing is ever silently dropped.
var CAP_OFFERED_KEY_LABELS = {
  flags: 'Flags', formats: 'Formats', ignores: 'Ignores',
  specversions: 'Spec Versions', versionmodes: 'Version Modes',
  pagination: 'Pagination', shortself: 'Short Self'
};

// One row's "Type" text for the offered-capabilities schema tables — e.g.
// "boolean", or "array of string" for an array/map node with a nested item
// schema (the item's own type, since the enum of allowed values lives on
// the array/map node itself, not on node.item).
function capSchemaTypeText(node) {
  if (!node || typeof node !== 'object') return 'any';
  if ((node.type === 'array' || node.type === 'map') && node.item) {
    return node.type + ' of ' + (node.item.type || '?');
  }
  return node.type || 'any';
}

// One row's "Values" cell — the enum chip list when the schema constrains
// this field to a fixed set of allowed values, otherwise a muted "any"
// placeholder (matching capChipList's own "— none —" empty-state style).
// Booleans are the one exception: an unconstrained boolean's only two
// possible values are always true/false, so show those explicitly rather
// than a vague "any".
function capSchemaValuesEl(node) {
  if (node && Array.isArray(node.enum) && node.enum.length) {
    return capChipList(node.enum, null, null);
  }
  if (node && node.type === 'boolean') {
    return capChipList(['true', 'false'], null, null);
  }
  var none = document.createElement('span'); none.className = 'capNone'; none.textContent = '\u2014 any \u2014';
  return none;
}

function capOfferedTableHead(table, headers) {
  var tr = document.createElement('tr');
  headers.forEach(function(h) { var th = document.createElement('th'); th.textContent = h; tr.appendChild(th); });
  table.appendChild(tr);
}

// Read-only List view for a registry's /capabilitiesoffered document. This
// is itself a schema description (each entry looks like {type, attributes}
// / {type, enum, item} — see common/capabilities.go's OfferedCapability
// shape) rather than a set of actual values, so instead of reusing
// renderCapValueGeneric (which pairs a *value* with its offered node) each
// top-level key gets a small dedicated table, grouped to match how the
// data is actually shaped:
//  - "Available": one row per API (available.attributes), API/"Can be
//    mutable?" columns — answers the yes/no question directly (locked to
//    enum:[false] means "false", otherwise "true" remains possible) rather
//    than showing a redundant "boolean" type badge on every row.
//  - "Compatibilities": one row per format, Format/Type/Available Values
//    columns.
//  - "Other Capabilities": every remaining top-level key (flags, formats,
//    ignores, specversions, versionmodes, pagination, shortself, and any
//    future addition) merged into one Name/Type/Available Values table
//    instead of
//    a separate one-row section per key, since they all share the exact
//    same {type[, enum][, item]} leaf shape.
// Always read-only — capabilitiesoffered is a server-declared document,
// never user-edited (see plan.md "Capabilities/CapabilitiesOffered List
// view").
function renderCapabilitiesOfferedViewer(data) {
  var main = el('main-view');
  main.innerHTML = '<div id="capEditor"></div>';
  var wrap = el('capEditor');
  wrap.insertAdjacentHTML('beforeend', serverURLLineHtml()) ;
  var body = document.createElement('div'); body.className = 'capBody';
  wrap.appendChild(body);
  data = data || {};
  var readOnlyPrev = _capReadOnly;
  _capReadOnly = true; // capChipList() consults this — force read-only rendering

  if (data.available && data.available.attributes) {
    var availSec = capSectionEl('Available');
    var availTable = document.createElement('table'); availTable.className = 'capTable';
    capOfferedTableHead(availTable, ['API', 'Can be mutable?']);
    Object.keys(data.available.attributes).sort().forEach(function(api) {
      var apiNode = data.available.attributes[api];
      var mutNode = apiNode && apiNode.attributes && apiNode.attributes.mutable;
      var tr = document.createElement('tr');
      var tdApi = document.createElement('td'); tdApi.textContent = api;
      // Every entry here is always type "boolean", so skip the redundant
      // type badge. Answer the column's yes/no question directly: locked
      // to enum:[false] means "no", otherwise true remains a possible
      // value (even if not locked to it), so the answer is "yes"/true.
      var canBeMutable = !(mutNode && Array.isArray(mutNode.enum) && mutNode.enum.length
        && mutNode.enum.indexOf(true) === -1);
      var tdMut = document.createElement('td');
      tdMut.appendChild(capChipList([String(canBeMutable)], null, null));
      tr.appendChild(tdApi); tr.appendChild(tdMut);
      availTable.appendChild(tr);
    });
    availSec.body.appendChild(availTable);
    body.appendChild(availSec.sec);
  }

  if (data.compatibilities && data.compatibilities.attributes) {
    var compatSec = capSectionEl('Compatibilities');
    var compatTable = document.createElement('table'); compatTable.className = 'capTable';
    capOfferedTableHead(compatTable, ['Format', 'Type', 'Available Values']);
    Object.keys(data.compatibilities.attributes).sort().forEach(function(fmt) {
      var fmtNode = data.compatibilities.attributes[fmt];
      var tr = document.createElement('tr');
      var tdFmt = document.createElement('td'); tdFmt.textContent = fmt;
      var tdType = document.createElement('td'); tdType.textContent = capSchemaTypeText(fmtNode);
      var tdVals = document.createElement('td'); tdVals.appendChild(capSchemaValuesEl(fmtNode));
      tr.appendChild(tdFmt); tr.appendChild(tdType); tr.appendChild(tdVals);
      compatTable.appendChild(tr);
    });
    compatSec.body.appendChild(compatTable);
    body.appendChild(compatSec.sec);
  }

  var otherKeys = Object.keys(data).filter(function(k) {
    return k !== 'available' && k !== 'compatibilities';
  }).sort();
  if (otherKeys.length) {
    var otherSec = capSectionEl('Other Capabilities');
    var otherTable = document.createElement('table'); otherTable.className = 'capTable';
    capOfferedTableHead(otherTable, ['Name', 'Type', 'Available Values']);
    otherKeys.forEach(function(k) {
      var node = data[k];
      var tr = document.createElement('tr');
      var tdName = document.createElement('td'); tdName.textContent = CAP_OFFERED_KEY_LABELS[k] || k;
      var tdType = document.createElement('td'); tdType.textContent = capSchemaTypeText(node);
      var tdVals = document.createElement('td'); tdVals.appendChild(capSchemaValuesEl(node));
      tr.appendChild(tdName); tr.appendChild(tdType); tr.appendChild(tdVals);
      otherTable.appendChild(tr);
    });
    otherSec.body.appendChild(otherTable);
    body.appendChild(otherSec.sec);
  }

  _capReadOnly = readOnlyPrev;
}

// .xregistry — the small, extensible cross-registry discovery document
// (see plan.md "known-registries-autoadd-hide" and common/capabilities.go's
// HTTPGETXRegistryDiscovery). Always read-only, list-style viewer — shares
// the same capSectionEl visual structure as Capabilities Offered above, but
// the data here is plain *values* (not a schema), so it's rendered
// directly rather than through renderCapSchemaNode/renderCapValueGeneric.
// Today's only defined key is "registries" (an array of every registry
// URL this server currently knows about); any other/future top-level key
// falls back to a plain indented JSON blob so this stays forward-
// compatible without needing a code change for every new key added later.
function renderXRegistryViewer(data) {
  var main = el('main-view');
  main.innerHTML = '<div id="xregEditor"></div>';
  var wrap = el('xregEditor');
  wrap.insertAdjacentHTML('beforeend', serverURLLineHtml());
  var body = document.createElement('div'); body.className = 'capBody';
  wrap.appendChild(body);
  Object.keys(data || {}).sort().forEach(function(k) {
    var sec = capSectionEl(k);
    if (k === 'registries' && Array.isArray(data[k])) {
      sec.body.appendChild(renderXRegRegistriesTable(data[k]));
    } else {
      var pre = document.createElement('pre'); pre.className = 'capRawFallback';
      pre.textContent = JSON.stringify(data[k], null, 2);
      sec.body.appendChild(pre);
    }
    body.appendChild(sec.sec);
  });
}

// Builds the "registries" table for renderXRegistryViewer(): one row per
// URL, sorted, with the URL as a real clickable link that browses
// straight to that registry (same doBrowse() used by the Home page's
// registry cards/rows) — modifier-clicks still open a real new tab/window
// via the normal <a href> since doBrowse() is only invoked from onclick,
// never in place of the href itself. There's no "name" column here — the
// server intentionally only sends URLs (see HTTPGETXRegistryDiscovery());
// the name lives at the target URL itself so it can never get out of
// sync with this discovery doc.
function renderXRegRegistriesTable(registries) {
  var table = document.createElement('table'); table.className = 'capTable';
  var thead = document.createElement('thead');
  thead.innerHTML = '<tr><th>URL</th></tr>';
  table.appendChild(thead);
  var tbody = document.createElement('tbody');
  (registries || []).slice().sort(function(a, b) {
    return String(a).toLowerCase().localeCompare(String(b).toLowerCase());
  }).forEach(function(url) {
    var tr = document.createElement('tr');
    var urlTd = document.createElement('td');
    var a = document.createElement('a');
    a.href = url; a.textContent = url;
    a.onclick = function(e) { return serverCardClick(e, tr, url); };
    urlTd.appendChild(a);
    tr.appendChild(urlTd);
    tbody.appendChild(tr);
  });
  table.appendChild(tbody);
  return table;
}

// ---- Utilities -----------------------------------------------------------

function fetchJSON(url) {
  return fetch(url, {headers: {'Accept': 'application/json'}})
    .then(function(resp) {
      if (!resp.ok) {
        return resp.text().then(function(t) {
          throw new Error('HTTP ' + resp.status + ' — ' + t.slice(0, 300));
        });
      }
      return resp.json();
    });
}

// Fetch a resource/version entity.  Try with $details appended first (needed
// when hasdocument=true so the document body is not returned in place of
// metadata).  $details is valid on all resources and versions; if the server
// returns 400 for any reason, fall back to a plain GET.
function fetchWithDetailsFallback(url, useDetails) {
  if (!useDetails) return fetchJSON(url);
  // Don't double-append: a real server-provided/bookmarked apiurl= may
  // already have $details baked in (e.g. a Resource/Version deep link) —
  // appending again would produce ".../foo$details$details" and 404.
  if (/\$details$/.test(url.split('?')[0])) return fetchJSON(url);
  var detailsURL = url.replace(/(\?|$)/, '$details$1');
  return fetchJSON(detailsURL).catch(function() {
    return fetchJSON(url);
  });
}

function spinner() {
  return '<div class="spinner-wrap">Loading…</div>';
}

function el(id)    { return document.getElementById(id); }
function qsa(sel)  { return Array.from(document.querySelectorAll(sel)); }

// ---- Error banners (Save/Create/Delete failure messages) -----------------
//
// All the inline ".error-banner" divs shown after a failed Save/Create/
// Delete (Add-new-entity, Data/Meta/Version editors, JSON raw edit, Model
// Source, Capabilities) go through these two helpers so every one of them
// gets the same dismissible "X" close button, instead of each call site
// duplicating the markup. showErrorBanner() rebuilds the div's content
// (a message span + close button) rather than plain textContent, since a
// close button needs to coexist with the message text as a DOM child.
function hideErrorBanner(div) {
  if (!div) return;
  div.style.display = 'none';
  div.textContent = '';
}

function showErrorBanner(div, message) {
  if (!div) return;
  div.textContent = '';
  var closeBtn = document.createElement('button');
  closeBtn.type = 'button';
  closeBtn.className = 'error-banner-close';
  closeBtn.setAttribute('aria-label', 'Dismiss');
  closeBtn.textContent = '\u00d7';
  closeBtn.onclick = function() { hideErrorBanner(div); };
  div.appendChild(closeBtn);
  var msgSpan = document.createElement('span');
  msgSpan.className = 'error-banner-msg';
  msgSpan.textContent = message;
  div.appendChild(msgSpan);
  div.style.display = 'block';
}

function capitalize(s) {
  s = String(s || '');
  return s.charAt(0).toUpperCase() + s.slice(1);
}

// Formats an ISO-8601 timestamp (e.g. createdat/modifiedat) into the
// browser's local time as "MM/DD/YYYY hh:mm:ss AM/PM TZ" (e.g.
// "07/06/2026 07:22:30 PM EDT"), used by collection Grid/List views'
// Created/Modified display. Returns '' for missing/unparseable input.
function formatTimestamp(iso) {
  if (!iso) return '';
  var d = new Date(iso);
  if (isNaN(d.getTime())) return '';
  var parts = {};
  try {
    new Intl.DateTimeFormat('en-US', {
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit', second: '2-digit',
      hour12: true, timeZoneName: 'short'
    }).formatToParts(d).forEach(function(p) { parts[p.type] = p.value; });
  } catch (e) { return d.toString(); }
  if (!parts.month) return d.toString();
  return parts.month + '/' + parts.day + '/' + parts.year + ' '
       + parts.hour + ':' + parts.minute + ':' + parts.second + ' '
       + (parts.dayPeriod || '') + (parts.timeZoneName ? ' ' + parts.timeZoneName : '');
}

// List view's Resource-page Property table column-1 header. Deliberately
// omits the resource's singular type name (e.g. "File"): the table's own
// page title/breadcrumb already establishes the entity type, so repeating
// it in the table header is redundant. isDefault controls the "Default "
// prefix (shown for the resource's own flattened default version; omitted
// when the version-selector dropdown picks a specific non-default version).
function versionPropHeaderLabel(isDefault, vid) {
  return (isDefault ? 'Default Version' : 'Version')
    + (vid !== undefined && vid !== null ? ' (' + esc(String(vid)) + ')' : '')
    + ' Details';
}

// "Default" option label for the Resource page's standalone "Version:"
// dropdown — includes the default version's own versionid, e.g.
// "Default (1)", so it's clear which version "Default" currently resolves
// to without needing to select it first.
function defaultOptionLabel(data) {
  return 'Default' + (data && data.versionid !== undefined ? ' (' + String(data.versionid) + ')' : '');
}

function esc(s) {
  if (s == null) return '';
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
                  .replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

// Close inline error popups when clicking outside them
document.addEventListener('click', function(e) {
  if (!e.target.closest('.server-card-err-badge') && !e.target.closest('.server-card-err-popup')) {
    qsa('.server-card-err-popup').forEach(function(p) { p.style.display = 'none'; });
  }
});
