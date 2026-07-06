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
  section:     'data',  // 'data' | 'model' | 'modelsource' | 'capabilities' | 'capabilitiesoffered'
  path:        [],      // path segments relative to registry root (data section only)
  editMode:    false,
  mutable:     false,

  // JSON-view query options
  inlines:     [],
  filters:     [],
  docView:     false,
  binary:      false,
  collections: false,
  useExport:   false,   // use /export endpoint instead of registry root (depth 0 only)
};

// ---- Server/registry management (localStorage) ---------------------------

var LS_SERVERS     = 'xreg-servers';
var LS_OPTIONS     = 'xreg-options';
var _labelCache    = {};  // normalizedURL → probed registry name
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

// ---- Options (persisted) --------------------------------------------------
var _opts = (function() {
  try { return JSON.parse(localStorage.getItem(LS_OPTIONS) || '{}'); } catch(e) { return {}; }
})();

function saveOpts() {
  try { localStorage.setItem(LS_OPTIONS, JSON.stringify(_opts)); } catch(e) {}
}

function optClickToCopy() { return !!_opts.clickToCopy; }
function optJsonColorMode() { return _opts.jsonColorMode || 'full'; }

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

function addServer(url) {
  url = normalizeURL(url);
  if (!url) return;
  var list = loadServers();
  if (!list.includes(url)) { list.push(url); saveServers(list); }
}

function removeServer(url) {
  saveServers(loadServers().filter(function(u) { return u !== url; }));
}

// ---- Init -----------------------------------------------------------------

window.addEventListener('DOMContentLoaded', init);
window.addEventListener('popstate', function() { loadStateFromURL(); renderHeader(); refresh(); });
window.addEventListener('resize', function() { renderHeader(); });

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

  // Wire up unified view-controls buttons (data-dview="grid|table|json" + edit-btn)
  var vc = el('view-controls');
  if (vc) {
    vc.addEventListener('click', function(e) {
      var btn = e.target.closest('[data-dview]');
      if (btn && !btn.classList.contains('view-btn-disabled')) setDataView(btn.dataset.dview);
    });
  }

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
  _state.dataView    = p.get('dview') || defaultDataView(_state.section, _state.path.length);
  _state.editMode    = p.get('edit') === '1';
  _state.inlines     = csvList(p.get('inline'));
  _state.filters     = (p.get('filter') || '').split('\n').filter(Boolean);
  _state.docView     = p.get('doc')         === '1';
  _state.binary      = p.get('binary')      === '1';
  _state.collections = p.get('collections') === '1';
  _state.useExport   = p.get('export')      === '1';
}

function buildURL(st) {
  var p = new URLSearchParams();
  if (st.view && st.view !== 'home')       p.set('view',    st.view);
  if (st.serverURL)                        p.set('server',  st.serverURL);
  if (st.section && st.section !== 'data') p.set('section', st.section);
  if (st.editMode)                         p.set('edit', '1');
  if (st.path   && st.path.length)         p.set('path',    encodePath(st.path));
  var defaultView = defaultDataView(st.section, (st.path || []).length);
  if (st.dataView && st.dataView !== defaultView) p.set('dview', st.dataView);

  // JSON-view-only params: only add when actually in JSON view. Model/modelsource's
  // "list" (editor) dataView is not a JSON view, so it's excluded here.
  // New path-specific query params should be added here; they will naturally
  // be absent from the URL in all non-JSON-view contexts.
  var isJsonOnlySection = st.section === 'capabilities' || st.section === 'capabilitiesoffered';
  var inJsonView = st.dataView === 'json' || st.view === 'json' || isJsonOnlySection;
  if (inJsonView) {
    if (st.inlines && st.inlines.length)   p.set('inline',      st.inlines.join(','));
    if (st.filters && st.filters.length)   p.set('filter',      st.filters.join('\n'));
    if (st.docView)                        p.set('doc',         '1');
    if (st.binary)                         p.set('binary',      '1');
    if (st.collections)                    p.set('collections', '1');
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
    var newDepth   = (patch.path !== undefined ? patch.path : _state.path || []).length;
    var newSection = patch.section !== undefined ? patch.section : _state.section;
    var savedView  = defaultDataView(newSection, newDepth);
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
    // Prepend defaults so explicit values in patch still win
    patch = Object.assign({
      inlines: [], filters: [], docView: false, binary: false, collections: false,
      useExport: false, section: 'data', dataView: savedView
    }, patch);
  }
  Object.assign(_state, patch);
  history.pushState(null, '', buildURL(_state));
  renderHeader();
  refresh();
}

// Default dataView for a given section/path-depth, honoring saved preferences.
//   data                — per-depth preference (_opts.depthViews), default 'grid'
//   model / modelsource — per-section preference (_opts.sectionViews), default 'table' (list)
//   capabilities / capabilitiesoffered — always 'json' (no other view exists yet)
function defaultDataView(section, pathLen) {
  if (section === 'model' || section === 'modelsource' || section === 'capabilities') {
    return (_opts.sectionViews || {})[section] || 'table';
  }
  if (section === 'capabilitiesoffered') {
    return 'json';
  }
  return (_opts.depthViews || {})[pathLen] || 'grid';
}

function encodePath(parts) { return parts.map(encodeURIComponent).join('/'); }
function decodePath(str)   { return str ? str.split('/').map(decodeURIComponent).filter(Boolean) : []; }
function csvList(s)        { return s ? s.split(',').filter(Boolean) : []; }

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

// ---- API URL builder -----------------------------------------------------

function serverBase() {
  return (_state.serverURL || window.location.origin).replace(/\/$/, '');
}

function buildAPIURL() {
  var base = serverBase();
  if (_state.section === 'model')                return base + '/model';
  if (_state.section === 'modelsource')          return base + '/modelsource';
  if (_state.section === 'capabilities')         return base + '/capabilities';
  if (_state.section === 'capabilitiesoffered')  return base + '/capabilitiesoffered';
  if (_state.useExport)                          return base + '/export';

  var path = _state.path;
  var url = base + (path.length ? '/' + path.join('/') : '');

  var q = [];
  _state.inlines.forEach(function(i) { q.push('inline=' + encodeURIComponent(i)); });
  _state.filters.forEach(function(f) { q.push('filter=' + encodeURIComponent(f)); });
  if (_state.docView)     q.push('doc');
  if (_state.binary)      q.push('binary');
  if (_state.collections) q.push('collections');

  return q.length ? url + '?' + q.join('&') : url;
}

// ---- Header --------------------------------------------------------------

function renderHeader() {
  var isHome   = (_state.view === 'home');
  var isConfig = (_state.view === 'config');
  var isData   = !isHome && !isConfig;

  el('reg-select').style.display     = 'none';
  el('breadcrumbs').style.display    = '';
  el('view-toggle') && (el('view-toggle').style.display = 'none');
  setHeaderCompact(false);

  // On home, show buttons reflecting current group's layout without corrupting _state.dataView
  var effectiveView = isHome ? currentHomeLayout() : _state.dataView;

  // Gear: always visible on all pages
  var gb = el('gear-btn');
  if (gb) gb.style.display = '';

  // Section-specific view rules:
  //   data                        — grid/table/json all available, edit when mutable
  //   model / modelsource         — no grid (list-style editor only), list+json available;
  //                                 edit only ever available on modelsource (never model)
  //   capabilities                — no grid (list-style editor only), list+json available;
  //                                 edit available when the doc itself is mutable
  //   capabilitiesoffered         — JSON only, no editing (schema document)
  var section          = _state.section;
  var isModelSection    = isData && (section === 'model' || section === 'modelsource');
  var isCapSection      = isData && (section === 'capabilities');
  var isListOnlySection = isModelSection || isCapSection;
  var isJsonOnlySection = isData && (section === 'capabilitiesoffered');

  var enableGrid, enableList, enableJson, enableEdit;
  if (isConfig) {
    enableGrid = enableList = enableJson = enableEdit = false;
  } else if (isHome) {
    enableGrid = enableList = true;  enableJson = false; enableEdit = false;
  } else if (isJsonOnlySection) {
    enableGrid = enableList = false; enableJson = true;  enableEdit = false;
  } else if (isListOnlySection) {
    enableGrid = false; enableList = true; enableJson = true;
    enableEdit = isCapSection ? _state.mutable : (section === 'modelsource') && _state.mutable;
  } else {
    enableGrid = enableList = enableJson = true;
    enableEdit = _state.mutable;
  }

  qsa('[data-dview]').forEach(function(b) {
    var v = b.dataset.dview;
    var active = isJsonOnlySection ? (v === 'json') : (v === effectiveView);
    b.classList.toggle('active', active);
    var disabled = isConfig
      || (v === 'grid'  && !enableGrid)
      || (v === 'table' && !enableList)
      || (v === 'json'  && !enableJson);
    b.classList.toggle('view-btn-disabled', disabled);
  });
  var editBtn = el('edit-btn');
  if (editBtn) {
    editBtn.classList.toggle('active', _state.editMode);
    editBtn.classList.toggle('view-btn-disabled', isConfig || !enableEdit);
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
  pushState({view: 'config', editMode: false});
}

function setHomeGroup(v) {
  _state.homeGroup = v;
  _opts.homeGroup  = v;
  saveOpts();
  renderHeader();    // updates active layout button for the new group
  renderHome();
}

function toggleHomeLayout() {
  setDataView(currentHomeLayout() === 'table' ? 'grid' : 'table');
}

function setDataView(v) {
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

  _state.dataView = v;
  qsa('[data-dview]').forEach(function(b) {
    b.classList.toggle('active', b.dataset.dview === v);
  });

  // On home page, persist per-group layout preference (independent of data pages)
  if (_state.view === 'home') {
    _state.homeLayouts[_state.homeGroup] = v;
    if (!_opts.homeLayouts) _opts.homeLayouts = {registry: 'grid', types: 'grid'};
    _opts.homeLayouts[_state.homeGroup] = v;
    saveOpts();
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
    setLeftPanelVisible(v === 'json');
    if (v === 'json') { renderJSONView(_lastData); return; }
    var coll = isCollection(_state.path);
    if (coll) {
      v === 'grid' ? renderTileView(_lastData) : renderTableView(_lastData);
    } else {
      v === 'grid' ? renderEntityGrid(_lastData) : renderSingleEntity(_lastData);
    }
  }
}

// Build the registry dropdown: Home + known registries + Add
function buildServerDropdown() {
  var sel = el('reg-select');
  if (!sel) return;
  sel.innerHTML = '';

  addOption(sel, '__home__', 'Home',  _state.view === 'home');

  var servers = loadServers();
  // Always show local server if we're running from xrserver
  var origin = window.location.origin;
  if (!servers.includes(origin)) {
    addOption(sel, origin, serverLabel(origin), _state.serverURL === '');
  }
  servers.forEach(function(url) {
    var isCurrent = _state.serverURL === url ||
                    (_state.serverURL === '' && url === origin);
    addOption(sel, url, serverLabel(url), isCurrent && _state.view !== 'home');
  });

  addOption(sel, '__add__', '+ Add registry…', false);
}

function addOption(sel, val, text, selected) {
  var o = document.createElement('option');
  o.value = val;
  o.textContent = text;
  if (selected) o.selected = true;
  sel.appendChild(o);
}

function serverLabel(url) {
  var norm = normalizeURL(url || window.location.origin);
  if (_labelCache[norm]) return _labelCache[norm];
  return url.replace(/^https?:\/\//, '').replace(/\/$/, '') || url;
}

function onRegistryChange(uid) {
  if (uid === '__home__' || uid === '__add__') {
    pushState({view: 'home'});
    return;
  }
  var sv = (uid === window.location.origin) ? '' : uid;
  pushState({view: 'table', serverURL: sv, section: 'data', path: [], editMode: false});
}

function onSectionChange(section) {
  pushState({section: section, path: [], editMode: false});
}

function setView(view) {
  pushState({view: view, editMode: false});
}

function toggleEdit() {
  pushState({editMode: !_state.editMode});
}

// ---- Breadcrumbs ---------------------------------------------------------

var _bcSep  = '<span class="bc-space"></span><span class="bc-sep">/</span><span class="bc-space"></span>';
var _bcSegs = []; // current segments, shared with popup openers

// Returns [{label, onclick|null, isCurrent}] for the current state
function buildBreadcrumbSegments() {
  if (_state.view === 'home')   return null; // handled specially in renderBreadcrumbs
  if (_state.view === 'config') return [{label:'Config',     onclick:null, isCurrent:true}];

  var segs = [];
  var regLabel = serverLabel(_state.serverURL || window.location.origin);
  var regClick = 'pushState({path:[],section:\'data\',useExport:false,editMode:false});return false';
  var isSection = _state.section !== 'data';
  segs.push({label: regLabel, onclick: isSection || _state.path.length > 0 ? regClick : null, isCurrent: !isSection && _state.path.length === 0});

  // If in a section view, add the section name as the last breadcrumb
  if (isSection) {
    var sectionLabels = {model:'Model', modelsource:'Model Source', capabilities:'Capabilities', capabilitiesoffered:'Capabilities Offered'};
    segs.push({label: sectionLabels[_state.section] || _state.section, onclick: null, isCurrent: true});
    return segs;
  }

  _state.path.forEach(function(seg, i) {
    var newPath = _state.path.slice(0, i + 1);
    var isLast  = (i === _state.path.length - 1);
    var click   = isLast ? null
      : 'pushState({path:' + esc(JSON.stringify(newPath))
        + ',section:\'data\',editMode:false});return false';
    segs.push({label: seg, onclick: click, isCurrent: isLast});
  });
  return segs;
}

function renderSegment(seg) {
  if (seg.isCurrent || !seg.onclick) {
    return '<span class="bc-current">' + esc(seg.label) + '</span>';
  }
  return '<a class="bc-link" href="#" onclick="' + seg.onclick + '">' + esc(seg.label) + '</a>';
}

function breadcrumbsFromSegments(segs) {
  return segs.map(function(s) { return _bcSep + renderSegment(s); }).join('');
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
    return;
  }

  _bcSegs = buildBreadcrumbSegments();
  nav.innerHTML = breadcrumbsFromSegments(_bcSegs);
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
    + _bcSep + renderSegment(last);
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
    if (item.onclick) {
      return '<a class="' + cls + '" href="#" onclick="closeHeaderPopup();' + item.onclick + '">'
           + esc(item.label) + '</a>';
    }
    return '<span class="' + cls + ' popup-item-cur">' + esc(item.label) + '</span>';
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

function openCompactMenu(e) {
  e.stopPropagation();
  toggleHeaderPopup(e.currentTarget, buildCompactMenuItems(), true); // right-align
}

function buildCompactMenuItems() {
  var items = [];
  var isHome = (_state.view === 'home');
  var isData = !isHome && _state.view !== 'config';
  if (isHome) {
    var hg = _state.homeGroup;
    items.push({label: 'By Registry',    onclick: "setHomeGroup('registry')", active: hg === 'registry'});
    items.push({label: 'By Group Type',  onclick: "setHomeGroup('types')",    active: hg === 'types'});
    items.push({sep: true});
  }
  var dv = isHome ? currentHomeLayout() : (_state.dataView || 'grid');
  items.push({label: 'Grid view',  onclick: "setDataView('grid')",  active: dv === 'grid'});
  items.push({label: 'List view',  onclick: "setDataView('table')", active: dv === 'table'});
  if (isData) {
    items.push({label: 'JSON view',  onclick: "setDataView('json')",  active: dv === 'json'});
    if (_state.mutable) items.push({label: 'Edit', onclick: 'toggleEdit()'});
  }
  if (!isData) {
    items.push({sep: true});
    items.push({label: 'Config', onclick: 'goToConfig()'});
  }
  return items;
}

function setHeaderCompact(compact) {
  _headerCompact = compact;
  var viewControls = el('view-controls');
  var gearBtn      = el('gear-btn');
  var compactBtn   = el('compact-menu-btn');
  if (!compactBtn) return;
  if (compact) {
    if (viewControls) viewControls.style.display = 'none';
    if (gearBtn)      gearBtn.style.display      = 'none';
    compactBtn.style.display = '';
  } else {
    if (viewControls) viewControls.style.display = '';
    if (gearBtn)      gearBtn.style.display      = '';
    compactBtn.style.display = 'none';
  }
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
var _metaResourceIdField = ''; // resource's own ID field name, set when resource page renders
var _metaEntityType = '';      // resource's singular type name, used by List view's meta table header

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
  var isJsonOnlySection     = (_state.section === 'capabilitiesoffered');

  setLeftPanelVisible(_state.view === 'json' || _state.dataView === 'json' || isJsonOnlySection);
  main.innerHTML = spinner();

  // Capabilities Offered — always JSON (schema document, not user-edited)
  if (isJsonOnlySection) {
    var sectionURL = buildAPIURL();
    fetchJSON(sectionURL)
      .then(function(data) {
        _lastData = data;
        _state.mutable = false;
        renderHeader();
        renderJSONView(data);
      })
      .catch(function(err) {
        main.innerHTML = '<div class="error-banner">Error loading:\n'
          + esc(sectionURL) + '\n\n' + esc(String(err)) + '</div>';
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
  fetchWithDetailsFallback(apiURL, needsDetails(_state.path))
    .then(function(data) {
      _lastData = data;
      _state.mutable = detectMutable(data);
      // Re-render header to update edit button enabled/active state
      renderHeader();

      switch (_state.view) {
        case 'json': renderJSONView(data); break;
        default:
          if (_state.dataView === 'json') {
            renderJSONView(data);
          } else if (coll) {
            _state.dataView === 'grid' ? renderTileView(data) : renderTableView(data);
          } else {
            _state.dataView === 'grid' ? renderEntityGrid(data) : renderSingleEntity(data);
          }
      }
    })
    .catch(function(err) {
      main.innerHTML = '<div class="error-banner">Error loading:\n'
        + esc(apiURL) + '\n\n' + esc(String(err)) + '</div>';
    });
}

function detectMutable(data) {
  // Real implementation: check capabilities.available.entities.mutable or Allow header.
  // For now, assume mutable when browsing a registry.
  return !!_state.serverURL || _state.view !== 'home';
}

// ---- Home view -----------------------------------------------------------

function renderHome() {
  var main = el('main-view');
  var origin = window.location.origin;
  var servers = loadServers();
  var allServers = [origin].concat(servers.filter(function(u) { return u !== origin; }));

  var g = _state.homeGroup;
  var l = currentHomeLayout(); // per-group persisted layout, independent of data pages
  if (g === 'types') {
    l === 'table' ? renderHomeFlatList(main, allServers) : renderHomeFlatGrid(main, allServers);
  } else {
    l === 'table' ? renderHomeTable(main, allServers) : renderHomeGrid(main, allServers);
  }
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
  var html = '<div class="home-page">'
    + '<table class="home-table"><thead><tr>'
    +   '<th>Registry</th><th>Description</th><th>Group Types</th><th>Location</th>'
    + '</tr></thead><tbody>';
  sorted.forEach(function(url) {
    html += '<tr data-server-url="' + esc(url) + '">'
      + '<td class="ht-name" style="position:relative">'
      +   '<span class="ht-name-text ht-name-link" onclick="doBrowse(\'' + esc(url) + '\')">' + esc(serverLabel(url)) + '</span>'
      +   '<div class="server-card-err-popup" style="display:none">'
      +     '<div class="server-card-err-popup-title">Connection Error</div>'
      +     '<div class="server-card-err-popup-msg"></div>'
      +     '<button class="home-btn home-btn-secondary" style="font-size:11px;padding:2px 8px" '
      +       'onclick="this.closest(\'.server-card-err-popup\').style.display=\'none\'">Close</button>'
      +   '</div>'
      + '</td>'
      + '<td class="ht-desc-col" style="color:#666;font-size:13px"></td>'
      + '<td class="ht-groups"><div class="ht-groups-inner"><span class="ht-loading">…</span></div></td>'
      + '<td class="ht-url">' + esc(url) + '</td>'
      + '</tr>';
  });
  html += '</tbody></table></div>';
  main.innerHTML = html;

  main.querySelectorAll('[data-server-url]').forEach(function(row) {
    probeRegistry(row.dataset.serverUrl, function(info) {
      var nameEl   = row.querySelector('.ht-name-text');
      var groupsEl = row.querySelector('.ht-groups-inner');
      if (info.error) {
        // disable the name link and show error badge with popup
        if (nameEl) { nameEl.classList.remove('ht-name-link'); nameEl.removeAttribute('onclick'); }
        var badge = document.createElement('span');
        badge.className = 'server-card-err-badge';
        badge.textContent = '!';
        badge.title = 'Click for error details';
        badge.style.marginLeft = '6px';
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
        var nameCell = row.querySelector('.ht-name');
        if (nameCell) nameCell.appendChild(badge);
        if (groupsEl) groupsEl.textContent = '';
      } else {
        if (nameEl && info.label) nameEl.textContent = info.label;
        var descEl = row.querySelector('.ht-desc-col');
        if (descEl && info.description) descEl.textContent = info.description;
        if (groupsEl) {
          groupsEl.innerHTML = info.colls.length
            ? info.colls.map(function(c) {
                return '<span class="group-type-item">' + esc(c.plural) + ' (' + c.count + ')</span>';
              }).join('')
            : '<span class="group-type-none">none</span>';
        }
      }
      sortServerElements(row.closest('tbody'));
    });
  });
}

function renderHomeFlatGrid(main, servers) {
  main.innerHTML = '<div class="home-page"><div class="home-grid flat-home-grid" id="flat-grid"><span style="color:#aaa;font-size:13px">Loading…</span></div></div>';

  var pending = servers.length;
  var allTiles = [];

  function finish() {
    allTiles.sort(function(a, b) {
      var n = a.plural.localeCompare(b.plural);
      return n !== 0 ? n : a.regLabel.localeCompare(b.regLabel);
    });
    var grid = el('flat-grid');
    if (!grid) return;
    if (allTiles.length === 0) {
      grid.innerHTML = '<span style="color:#aaa;font-size:13px;font-style:italic">No group types found</span>';
      return;
    }
    grid.innerHTML = allTiles.map(function(t) {
      var onclick = 'browseGroupCollection(\'' + esc(t.serverUrl) + '\',\'' + esc(t.plural) + '\')';
      return groupTileHTML(t, onclick, '', t.regLabel);
    }).join('');
  }

  if (pending === 0) { finish(); return; }

  servers.forEach(function(url) {
    probeRegistry(url, function(info) {
      if (!info.error) {
        var label = info.label || serverLabel(url);
        info.colls.forEach(function(c) {
          allTiles.push({plural: c.plural, count: c.count, serverUrl: url, regLabel: label,
                         description: c.description || '', resources: c.resources || []});
        });
      }
      pending--;
      if (pending === 0) finish();
    });
  });
}

function browseGroupCollection(serverUrl, collName) {
  var sv = (serverUrl === window.location.origin) ? '' : serverUrl;
  pushState({view: 'table', serverURL: sv, section: 'data', path: [collName], editMode: false});
}

function renderHomeFlatList(main, servers) {
  main.innerHTML = '<div class="home-page">'
    + '<table class="home-table"><thead><tr>'
    +   '<th>Group Type</th><th>Description</th><th>Items</th><th>Resource Types</th><th>Registry</th>'
    + '</tr></thead><tbody id="flat-list-body"><tr><td colspan="5" style="color:#aaa;font-size:13px">Loading…</td></tr></tbody></table></div>';

  var pending = servers.length;
  var allRows = [];

  function finish() {
    allRows.sort(function(a, b) {
      var n = a.plural.localeCompare(b.plural);
      return n !== 0 ? n : a.regLabel.localeCompare(b.regLabel);
    });
    var tbody = el('flat-list-body');
    if (!tbody) return;
    if (allRows.length === 0) {
      tbody.innerHTML = '<tr><td colspan="5" style="color:#aaa;font-size:13px;font-style:italic">No group types found</td></tr>';
      return;
    }
    tbody.innerHTML = allRows.map(function(r) {
      var onclick = 'browseGroupCollection(\'' + esc(r.serverUrl) + '\',\'' + esc(r.plural) + '\')';
      return '<tr>'
        + '<td class="ht-name ht-name-link" style="font-weight:bold" onclick="' + onclick + '">' + esc(r.plural) + '</td>'
        + '<td class="ht-desc-col" style="color:#666;font-size:13px">' + esc(r.description) + '</td>'
        + '<td class="ht-groups">' + r.count + '</td>'
        + '<td class="ht-groups"><div class="ht-groups-inner">'
        +   (r.resources.length
              ? r.resources.map(function(res) { return '<span class="group-type-item">' + esc(res) + '</span>'; }).join('')
              : '<span class="group-type-none">none</span>')
        + '</div></td>'
        + '<td class="ht-url">' + esc(r.regLabel) + '<div class="ht-desc">' + esc(r.serverUrl) + '</div></td>'
        + '</tr>';
    }).join('');
  }

  if (pending === 0) { finish(); return; }

  servers.forEach(function(url) {
    probeRegistry(url, function(info) {
      if (!info.error) {
        var label = info.label || serverLabel(url);
        info.colls.forEach(function(c) {
          allRows.push({plural: c.plural, count: c.count, resources: c.resources || [],
                        description: c.description || '', serverUrl: url, regLabel: label});
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
        if (nameEl && info.label) nameEl.textContent = info.label;
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
                return '<span class="group-type-item">' + esc(c.plural) + ' (' + c.count + ')</span>';
              }).join('')
            : '<span class="group-type-none">none</span>';
        }
      }
      sortServerElements(container);
    });
  });
}

function serverCard(url) {
  return '<div class="server-card" onclick="serverCardClick(this,\'' + esc(url) + '\')" data-server-url="' + esc(url) + '">'
    + '<div class="server-card-title">'
    +   '<img src="favicon.svg" class="server-card-icon" alt="" width="16" height="16">'
    +   '<span class="server-card-name">' + esc(serverLabel(url)) + '</span>'
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

function probeRegistry(url, cb) {
  var normUrl = normalizeURL(url);
  // Fetch capabilities first; use it to decide what else to fetch.
  // Per spec: if /capabilities is 404, default available = {entities:{mutable:true}}
  var capP = fetch(normUrl + '/capabilities')
    .then(function(r) { return r.ok ? r.json() : null; })
    .catch(function() { return null; });

  capP.then(function(cap) {
    var available = (cap && cap.available) || {entities: {mutable: true}};
    // Populate capabilities cache so JSON view left panel gets it for free
    _capCache[normUrl] = cap || {available: available, flags: []};

    // Only fetch model if capabilities says it's available
    var modelP = available.model
      ? fetch(normUrl + '/model').then(function(r) { return r.json(); }).catch(function() { return null; })
      : Promise.resolve(null);

    Promise.all([fetchJSON(normUrl + '/'), modelP])
      .then(function(results) {
        var data  = results[0];
        var model = results[1];
        if (!data.specversion || !data.registryid) {
          cb({label: '', colls: [], icon: '', available: available, error: 'Not a valid xRegistry (missing specversion or registryid)'});
          return;
        }
        var label = data.registryid || '';
        if (label) _labelCache[normUrl] = label;
        var colls = findCollectionRefs(model, [], data);
        colls.forEach(function(c) {
          var grpDef = model && model.groups && model.groups[c.plural];
          c.resources   = grpDef && grpDef.resources ? Object.keys(grpDef.resources).sort() : [];
          c.description = (grpDef && grpDef.description) || '';
        });
        cb({label: label, colls: colls, icon: data.icon || '', description: data.description || '', available: available, error: null});
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
  pushState({view: 'table', serverURL: sv, section: 'data', path: [], editMode: false});
}

function serverCardClick(card, url) {
  if (card.querySelector('.server-card-err-badge')) return;
  doBrowse(url);
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

function renderConfig() {
  var main   = el('main-view');
  var origin = window.location.origin;
  var servers = loadServers();

  var html = '<div class="config-page"><div class="config-section">'
    + '<h3 class="config-section-title">Registry Servers</h3>'
    + '<table class="config-table"><thead><tr><th>Name</th><th>Location</th><th></th></tr></thead><tbody>';

  // Local server — not editable or deletable
  html += '<tr data-cfg-url="' + esc(normalizeURL(origin)) + '">'
    + '<td class="cfg-name">' + esc(_labelCache[normalizeURL(origin)] || '') + '</td>'
    + '<td><span class="cfg-url-display">' + esc(origin)
    + ' <span class="config-local-badge">this server</span></span></td><td></td></tr>';

  // User-added servers
  servers.filter(function(u) { return u !== origin; }).forEach(function(url) {
    html += '<tr data-cfg-url="' + esc(url) + '">'
      + '<td class="cfg-name">' + esc(_labelCache[normalizeURL(url)] || '') + '</td>'
      + '<td>'
      +   '<span class="cfg-url-display">' + esc(url) + '</span>'
      +   '<input class="cfg-url-input" style="display:none" value="' + esc(url) + '" '
      +     'onkeydown="if(event.key===\'Enter\')cfgSave(this);'
      +               'else if(event.key===\'Escape\')cfgCancel(this)">'
      + '</td>'
      + '<td class="cfg-actions">'
      +   '<button class="cfg-btn cfg-edit" onclick="cfgEdit(this)">Edit</button>'
      +   '<button class="cfg-btn cfg-save" style="display:none" onclick="cfgSave(this)">Save</button>'
      +   '<button class="cfg-btn cfg-cancel" style="display:none" onclick="cfgCancel(this)">Cancel</button>'
      +   '<button class="cfg-btn cfg-del" onclick="cfgDelete(this)">Delete</button>'
      + '</td></tr>';
  });

  html += '</tbody></table>'
    + '<div class="cfg-add-row">'
    +   '<input type="text" id="cfg-new-url" placeholder="http://example.com" '
    +          'onkeydown="if(event.key===\'Enter\')cfgAddNew()">'
    +   '<button class="cfg-btn" onclick="cfgAddNew()">Add</button>'
    + '</div>'
    + '</div>'

    // ---- Options section ----
    + '<div class="config-section">'
    + '<h3 class="config-section-title">Options</h3>'
    + '<label class="cfg-option-row">'
    +   '<span class="cfg-option-label">Click to copy</span>'
    +   '<input type="checkbox" id="opt-click-to-copy"'
    +   (optClickToCopy() ? ' checked' : '')
    +   ' onchange="cfgSetOpt(\'clickToCopy\',this.checked)">'
    +   '<span class="cfg-option-desc">Click any value in the details'
    +   ' view to copy it to the clipboard</span>'
    + '</label>'
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

    + '</div>';
  main.innerHTML = html;

  // Probe all servers; mark any that error with the same ! badge + popup as the home page
  var allUrls = [origin].concat(servers.filter(function(u) { return u !== origin; }));
  allUrls.forEach(function(url) {
    var norm = normalizeURL(url);
    probeRegistry(url, function(info) {
      var tr = main.querySelector('tr[data-cfg-url="' + norm + '"]');
      if (tr && info.label) {
        var nameCell = tr.querySelector('.cfg-name');
        if (nameCell) nameCell.textContent = info.label;
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

function cfgEdit(btn) {
  var tr = btn.closest('tr');
  tr.querySelector('.cfg-url-display').style.display = 'none';
  tr.querySelector('.cfg-url-input').style.display   = '';
  tr.querySelector('.cfg-edit').style.display        = 'none';
  tr.querySelector('.cfg-save').style.display        = '';
  tr.querySelector('.cfg-cancel').style.display      = '';
  var inp = tr.querySelector('.cfg-url-input');
  inp.focus(); inp.select();
}

function cfgCancel(el) {
  var tr = el.closest('tr');
  var inp = tr.querySelector('.cfg-url-input');
  inp.value = tr.dataset.cfgUrl;
  tr.querySelector('.cfg-url-display').style.display = '';
  inp.style.display                                  = 'none';
  tr.querySelector('.cfg-edit').style.display        = '';
  tr.querySelector('.cfg-save').style.display        = 'none';
  tr.querySelector('.cfg-cancel').style.display      = 'none';
}

function cfgSave(el) {
  var tr  = el.closest('tr');
  var old = tr.dataset.cfgUrl;
  var neu = normalizeURL(tr.querySelector('.cfg-url-input').value);
  if (!neu) return;
  removeServer(old);
  addServer(neu);
  renderConfig();
}

function cfgDelete(btn) {
  removeServer(btn.closest('tr').dataset.cfgUrl);
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

function cfgAddNew() {
  var inp = el('cfg-new-url');
  if (!inp || !inp.value.trim()) return;
  addServer(inp.value.trim());
  renderConfig();
  var newInp = el('cfg-new-url');
  if (newInp) newInp.focus();
}

// ---- Tile view -----------------------------------------------------------

function renderTileView(data) {
  var main = el('main-view');
  var items = collectionItems(data);

  if (items.length === 0) {
    main.innerHTML = '<div class="state-msg">No items found</div>';
    return;
  }

  var html = '<div id="tile-container">';
  // Pick icon based on collection depth: groups=folder, resources+versions=doc
  var tileIcon = '';
  if (_state.path.length === 1)                        tileIcon = FOLDER_ICON;
  else if (_state.path.length === 3 || _state.path.length === 5) tileIcon = DOC_ICON;

  items.forEach(function(item) {
    var id   = itemNavKey(item);
    var desc = item.description || '';
    var svBase   = (_state.serverURL || window.location.origin).replace(/\/$/, '');
    var model    = _modelCache[normalizeURL(svBase)] || null;
    var itemPath = _state.path.concat([id]);
    var colls = findCollectionRefs(model, itemPath, item);

    // Build the full-width sub-collection footer (outside tile-body so it spans full width)
    var footerHtml = '';
    if (_state.path.length === 1) {
      var collItems = colls.length
        ? colls.map(function(c) {
            return '<span class="coll-tile-res-pill">' + esc(c.plural) + ' (' + c.count + ')</span>';
          }).join('')
        : '<span class="coll-tile-res-none">none</span>';
      footerHtml = '<hr class="coll-tile-divider">'
            + '<div class="coll-tile-res-hdr">Resources:</div>'
            + '<div class="coll-tile-res">' + collItems + '</div>';
    } else if (_state.path.length === 3 && colls.length) {
      var verItems = colls.map(function(c) {
        return '<span class="coll-tile-res-pill">' + esc(c.plural) + ': ' + c.count + '</span>';
      }).join('');
      footerHtml = '<hr class="coll-tile-divider">'
            + '<div class="coll-tile-res">' + verItems + '</div>';
    }

    html += '<div class="tile" onclick="navigateTo(\'' + esc(id) + '\')">';
    html += '<div class="tile-top">';
    if (tileIcon) html += '<div class="tile-icon">' + tileIcon + '</div>';
    html += '<div class="tile-body">';
    html += '<div class="tile-id">' + esc(id) + '</div>';
    if (item.name) html += '<div class="tile-name">' + esc(item.name) + '</div>';
    if (desc)      html += '<div class="tile-desc">' + esc(desc) + '</div>';
    html += '</div></div>'; // close tile-body + tile-top
    html += footerHtml;
    html += '</div>'; // close tile
  });
  html += '</div>';
  main.innerHTML = html;
}

// ---- Table view ----------------------------------------------------------

var _sortCol = null;
var _sortAsc = true;
var _tableViewItems = []; // current renderTableView() items, indexed for per-row doc buttons

function renderTableView(data) {
  var main = el('main-view');
  var items = collectionItems(data);

  if (items.length === 0) {
    main.innerHTML = '<div class="state-msg">No items found</div>';
    return;
  }

  var svBase = (_state.serverURL || window.location.origin).replace(/\/$/, '');
  var modelKey = normalizeURL(svBase);
  if (!_modelCache.hasOwnProperty(modelKey)) {
    ensureModelCached(svBase, function() {
      if (_lastData === data) renderTableView(data);
    });
  }
  var model  = _modelCache[modelKey] || null;
  var depth  = _state.path.length; // 1=groups, 3=resources, 5=versions

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

  function thSort(col, label) {
    var cls = col === _sortCol ? (_sortAsc ? ' sorted-asc' : ' sorted-desc') : '';
    return '<th class="' + cls + '" onclick="sortBy(\'' + esc(col) + '\')">' + esc(label) + '</th>';
  }

  var idColLabel = capitalize(getSingularName(model, _state.path.concat(['__x__'])));
  var html = '<div id="table-container"><table class="xr-table"><thead><tr>';
  html += thSort('__id', idColLabel);
  if (hasName) html += thSort('name', 'Name');
  if (hasDesc) html += thSort('description', 'Description');
  if (showChildren) html += '<th>' + (depth === 1 ? 'Resources' : 'Versions') + '</th>';
  if (showDoc) html += '<th>Document</th>';
  html += '</tr></thead><tbody>';

  items.forEach(function(item, idx) {
    var id      = itemNavKey(item);
    var itemPath = _state.path.concat([id]);
    var colls   = showChildren ? findCollectionRefs(model, itemPath, item) : [];

    var childrenHtml = '';
    if (showChildren) {
      if (colls.length) {
        childrenHtml = colls.map(function(c) {
          return '<span class="coll-tile-res-pill">' + esc(c.plural) + ' (' + c.count + ')</span>';
        }).join(' ');
      } else {
        childrenHtml = '<span class="coll-tile-res-none">—</span>';
      }
    }

    html += '<tr onclick="navigateTo(\'' + esc(id) + '\')">';
    html += '<td class="cell-id">' + esc(id) + '</td>';
    if (hasName) html += '<td>' + esc(item.name != null ? String(item.name) : '') + '</td>';
    if (hasDesc) html += '<td class="cell-desc">' + esc(item.description != null ? String(item.description) : '') + '</td>';
    if (showChildren) html += '<td class="cell-children">' + childrenHtml + '</td>';
    if (showDoc) {
      var docClickExpr = 'event.stopPropagation();openDocument(' + JSON.stringify(docSingular) + ', _tableViewItems[' + idx + '])';
      html += '<td class="cell-children">'
            + '<button class="cfg-btn" style="font-size:11px;padding:2px 8px" onclick="' + esc(docClickExpr) + '">View</button>'
            + '</td>';
    }
    html += '</tr>';
  });

  html += '</tbody></table></div>';
  main.innerHTML = html;
}

function sortBy(col) {
  if (_sortCol !== col) { _sortCol = col; _sortAsc = true; }
  else { _sortAsc = !_sortAsc; }
  if (_lastData) renderTableView(_lastData);
}

// ---- Single entity view --------------------------------------------------
//
// For the Registry root and Group/Resource entities.
// Scalar props shown in a property table; collection references (pairs of
// <name>url + <name>count) rendered as clickable rows.

function renderSingleEntity(data) {
  var main = el('main-view');
  if (!data || typeof data !== 'object') {
    main.innerHTML = '<div class="state-msg">' + esc(String(data)) + '</div>';
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
  var colls  = findCollectionRefs(model, _state.path, data);
  var collKeys = {};
  colls.forEach(function(c) { collKeys[c.plural + 'url'] = true; collKeys[c.plural + 'count'] = true; });

  // Priority ordering for scalar props — hand-tuned for UX, not spec declaration order.
  // specAttrOrder() gives spec-canonical order but it doesn't match what's most useful
  // to show first in the UI.
  var priority = ['registryid','xid','name','description','specversion',
    'epoch','createdat','modifiedat','versionid','isdefault','ancestor'];
  var scalarKeys = Object.keys(data).filter(function(k) {
    return !collKeys[k] && typeof data[k] !== 'object';
  }).sort(function(a, b) {
    var ai = priority.indexOf(a), bi = priority.indexOf(b);
    if (ai >= 0 && bi >= 0) return ai - bi;
    if (ai >= 0) return -1; if (bi >= 0) return 1;
    return a.localeCompare(b);
  });

  var html = '<div id="table-container">';

  // Collections section — clickable
  if (colls.length) {
    var depthT = _state.path.length;
    // Match the Grid view's section-header wording (GROUP TYPES / RESOURCES),
    // and likewise suppress the header at depth 4 (a resource's version
    // collection is the only collection type there, so a label is redundant —
    // see the matching comment in renderEntityGrid).
    var collsHeaderT = depthT === 0 ? 'Group Types' : depthT === 2 ? 'Resources' : 'Collection';
    html += '<table class="xr-table" style="margin-bottom:16px">';
    if (depthT !== 4) {
      html += '<thead><tr><th>' + esc(collsHeaderT) + '</th><th>Count</th></tr></thead>';
    }
    html += '<tbody>';
    colls.forEach(function(c) {
      html += '<tr onclick="navigateTo(\'' + esc(c.plural) + '\')" style="cursor:pointer">'
        + '<td class="cell-id">' + esc(c.plural) + '</td>'
        + '<td>' + c.count + '</td>'
        + '</tr>';
    });
    html += '</tbody></table>';
  }

  // Server endpoints (depth 0 only) — same sections shown in grid view
  if (_state.path.length === 0) {
    var svBaseT = (_state.serverURL || window.location.origin).replace(/\/$/, '');
    var capDataT = _capCache[normalizeURL(svBaseT)];
    var availT   = capDataT && capDataT.available;
    var sectionTilesT = ['model','modelsource','capabilities','capabilitiesoffered'];
    var availSectionsT = sectionTilesT.filter(function(s) { return availT && availT[s]; });
    if (availSectionsT.length) {
      var sectionNamesT = {model:'Model', modelsource:'Model Source', capabilities:'Capabilities', capabilitiesoffered:'Capabilities Offered'};
      html += '<table class="xr-table" style="margin-bottom:16px">'
        + '<thead><tr><th>Registry Endpoints</th><th>Mutable</th></tr></thead><tbody>';
      availSectionsT.forEach(function(s) {
        var mut = availT[s] && availT[s].mutable ? '✓' : '';
        var click = 'pushState({section:\'' + s + '\',editMode:false,useExport:false});return false';
        html += '<tr onclick="' + esc(click) + '" style="cursor:pointer">'
          + '<td class="cell-id">' + esc(sectionNamesT[s]) + '</td>'
          + '<td>' + mut + '</td>'
          + '</tr>';
      });
      html += '</tbody></table>';
    }
  }

  // Resource meta box (depth 4 only) — mirrors Grid view's collapsible
  // "<Singular> Details" box (twisty + Copy JSON), lazy-fetched via metaurl.
  // Reuses toggleMetaBox()/copyMetaJSON()/renderMetaContent() as-is, since
  // they're DOM-id-driven and view-agnostic.
  var depthD = _state.path.length;
  if (depthD === 4) {
    var entityTypeT = getSingularName(model, _state.path);
    _metaData = null;
    _metaResourceIdField = entityTypeT.toLowerCase() + 'id';
    _metaEntityType = entityTypeT;
    html += '<div class="eg-section-header eg-details-header">'
          + '<span>' + esc(capitalize(entityTypeT)) + ' Details'
          + ' <button class="eg-twisty" id="eg-meta-twisty" onclick="toggleMetaBox()">▶</button>'
          + '</span>'
          + '<button class="eg-copy-json-btn" onclick="copyMetaJSON()">{ } Copy JSON</button>'
          + '</div>';
    html += '<div class="eg-meta-details-flat" id="eg-meta-box" style="display:none"></div>';
  }

  // Document (depth 4 = resource entity, depth 6+ = version entity) — mirrors
  // the Grid view's document tile so the doc is reachable from List view too.
  if ((depthD === 4 || depthD >= 6) && resourceHasDocument(model, _state.path)) {
    var docSingularD = getSingularName(model, _state.path.slice(0, 4));
    html += '<table class="xr-table" style="margin-bottom:16px">'
      + '<thead><tr><th>Document</th><th>Content Type</th></tr></thead><tbody>'
      + '<tr onclick="openDocument(\'' + esc(docSingularD) + '\')" style="cursor:pointer">'
      +   '<td class="cell-id">' + esc(docSingularD) + ' document</td>'
      +   '<td>' + esc(data.contenttype || '') + '</td>'
      + '</tr></tbody></table>';
  }

  // Scalar properties
  if (scalarKeys.length) {
    var capTypeT = capitalize(getSingularName(model, _state.path));
    var propHeaderT = depthD === 4 ? defaultVersionLabel(capTypeT, data) + ' Property' : capTypeT + ' Property';
    html += '<table class="xr-table"><thead><tr><th>' + esc(propHeaderT) + '</th><th>Value</th></tr></thead><tbody>';
    scalarKeys.forEach(function(k) {
      var val = data[k];
      var display = (val == null) ? '<span style="color:#999">null</span>' : esc(String(val));
      html += '<tr><td style="font-weight:bold;color:#444;width:200px">' + esc(k)
            + '</td><td>' + display + '</td></tr>';
    });
    html += '</tbody></table>';
  }

  html += '</div>';
  main.innerHTML = html;
}

// ---- Grid view for single entity (Registry / Group / Resource / Version) -

// Fetch and cache the model for a registry base URL (non-blocking)
function ensureModelCached(baseURL, cb) {
  var key = normalizeURL(baseURL);
  if (_modelCache[key]) { if (cb) cb(_modelCache[key]); return; }
  fetch(baseURL.replace(/\/$/, '') + '/model')
    .then(function(r) { return r.json(); })
    .then(function(m) { _modelCache[key] = m; if (cb) cb(m); })
    .catch(function()  { _modelCache[key] = null; if (cb) cb(null); });
}

function ensureCapCached(baseURL, cb) {
  var key = normalizeURL(baseURL);
  if (_capCache.hasOwnProperty(key)) { if (cb) cb(_capCache[key]); return; }
  _capCache[key] = undefined; // mark in-flight
  fetch(baseURL.replace(/\/$/, '') + '/capabilities')
    .then(function(r) { return r.json(); })
    .then(function(c) { _capCache[key] = c; if (cb) cb(c); })
    .catch(function()  { _capCache[key] = null; if (cb) cb(null); });
}

// Fetch and cache /capabilitiesoffered for a registry base URL (non-blocking).
// Used to know the declared type/enum/attributes of extension capabilities
// so the Capabilities editor can render/edit them generically.
function ensureOfferedCached(baseURL, cb) {
  var key = normalizeURL(baseURL);
  if (_offeredCache.hasOwnProperty(key)) { if (cb) cb(_offeredCache[key]); return; }
  _offeredCache[key] = undefined; // mark in-flight
  fetch(baseURL.replace(/\/$/, '') + '/capabilitiesoffered')
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

// specAttrOrder returns the SPEC_ATTRS_ORDER array for the given path, or [].
function specAttrOrder(path) {
  if (typeof SPEC_ATTRS_ORDER === 'undefined') return [];
  var name = specAttrLevelName(path);
  return (name && SPEC_ATTRS_ORDER[name]) || [];
}

// isMonoSpecAttr returns true if key k should be rendered monospaced because
// it is both a spec-defined attribute at the current entity level AND is in
// MONO_ATTRS (string-typed spec attrs that are technical, not human prose).
// The dynamic "id" entry in MONO_ATTRS matches any <singular>id field.
function isMonoSpecAttr(k, specLevel, singular) {
  if (!isSpecAttr(k, specLevel, singular, null)) return false;
  // Find the MONO_ATTRS sub-object that corresponds to this specLevel
  var monoSet = null;
  if (typeof MONO_ATTRS !== 'undefined' && typeof SPEC_ATTRS !== 'undefined') {
    var levelNames = ['registry','group','resource','meta','version'];
    for (var i = 0; i < levelNames.length; i++) {
      if (SPEC_ATTRS[levelNames[i]] === specLevel) {
        monoSet = MONO_ATTRS[levelNames[i]] || {};
        break;
      }
    }
  }
  if (!monoSet) return false;
  if (monoSet[k]) return true;
  // dynamic id pattern: MONO_ATTRS.*.id covers <singular>id fields
  if (monoSet.id && singular && k === singular + 'id') return true;
  return false;
}

// getAttr returns the full Attribute definition object from the model for
// the given attribute key path (array) within an entity at entityPath depth.
// attrKeyPath is an array for nested traversal, e.g. ['myattr'] or ['obj','child'].
// Falls back to the '*' wildcard entry for undeclared extension attributes.
// Returns null only on model compliance violation (should not happen in practice).
//
// TODO(ifvalues): when ifvalues support is added, a 'data' parameter (the actual
// entity JSON) will be needed here so conditional sibling-attribute rules can be
// evaluated to find additional attributes introduced by ifvalues matches.
function getAttr(model, entityPath, attrKeyPath) {
  if (!model || !attrKeyPath || attrKeyPath.length === 0) return null;
  var depth = entityPath ? entityPath.length : 0;

  // Find the top-level attributes map for this entity depth
  var attrs;
  if (depth === 0) {
    attrs = model.attributes;
  } else if (depth === 2) {
    var gm = model.groups && model.groups[entityPath[0]];
    attrs = gm && gm.attributes;
  } else if (depth >= 4) {
    var gm2 = model.groups && model.groups[entityPath[0]];
    var rm  = gm2 && gm2.resources && gm2.resources[entityPath[2]];
    attrs = rm && rm.attributes;
  }
  if (!attrs) return null;

  // Traverse attrKeyPath, following .attributes for nested objects
  var attr = null;
  for (var i = 0; i < attrKeyPath.length; i++) {
    var key = attrKeyPath[i];
    attr = attrs[key] || attrs['*'] || null;
    if (!attr) return null;
    if (i < attrKeyPath.length - 1) {
      attrs = attr.attributes;
      if (!attrs) return null;
    }
  }
  return attr;
}

// Convenience wrapper — returns just the type string (or null).
function getAttrType(model, entityPath, attrKeyPath) {
  var attr = getAttr(model, entityPath, attrKeyPath);
  return attr ? (attr.type || null) : null;
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
function getExplicitAttrType(model, entityPath, key) {
  if (!model || !key) return null;
  var depth = entityPath ? entityPath.length : 0;
  var attrs;
  if (depth === 0) {
    attrs = model.attributes;
  } else if (depth === 2) {
    var gm = model.groups && model.groups[entityPath[0]];
    attrs = gm && gm.attributes;
  } else if (depth >= 4) {
    var gm2 = model.groups && model.groups[entityPath[0]];
    var rm  = gm2 && gm2.resources && gm2.resources[entityPath[2]];
    attrs = rm && rm.attributes;
  }
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
function getExplicitAttrTypeAtPath(model, entityPath, attrKeyPath) {
  if (!model || !attrKeyPath || attrKeyPath.length === 0) return null;
  var depth = entityPath ? entityPath.length : 0;
  var attrs;
  if (depth === 0) {
    attrs = model.attributes;
  } else if (depth === 2) {
    var gm = model.groups && model.groups[entityPath[0]];
    attrs = gm && gm.attributes;
  } else if (depth >= 4) {
    var gm2 = model.groups && model.groups[entityPath[0]];
    var rm  = gm2 && gm2.resources && gm2.resources[entityPath[2]];
    attrs = rm && rm.attributes;
  }
  if (!attrs) return null;
  var attr = null;
  for (var i = 0; i < attrKeyPath.length; i++) {
    var key = attrKeyPath[i];
    var isLast = (i === attrKeyPath.length - 1);
    if (isLast) {
      return typeFromAttrsMap(attrs, key);
    }
    attr = attrs[key] || attrs['*'] || null;
    if (!attr) return null;
    attrs = attr.attributes;
    if (!attrs) return null;
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
  }
  return false;
}

var FOLDER_ICON = '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" '
  + 'viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" '
  + 'stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">'
  + '<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>'
  + '</svg>';

var DOC_ICON = '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" '
  + 'viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" '
  + 'stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">'
  + '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>'
  + '<polyline points="14 2 14 8 20 8"/>'
  + '<line x1="16" y1="13" x2="8" y2="13"/>'
  + '<line x1="16" y1="17" x2="8" y2="17"/>'
  + '<polyline points="10 9 9 9 8 9"/>'
  + '</svg>';

var INFO_ICON = '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" '
  + 'viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" '
  + 'stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">'
  + '<circle cx="12" cy="12" r="10"/>'
  + '<line x1="12" y1="8" x2="12" y2="8"/>'
  + '<line x1="12" y1="12" x2="12" y2="16"/>'
  + '</svg>';

function collectionTile(coll) {
  var onclick = coll.count === 0
    ? 'showToast(\'Nothing to show\')'
    : 'navigateTo(\'' + esc(coll.plural) + '\')';
  var emptyCls = coll.count === 0 ? ' coll-tile-empty' : '';
  return groupTileHTML(coll, onclick, emptyCls, '');
}

function groupTileHTML(coll, onclick, extraCls, regLabel) {
  var descHtml = coll.description
    ? '<div class="coll-tile-desc">' + esc(coll.description) + '</div>'
    : '';
  // Only show Resource Types section when the model has provided the list (depth 0 group tiles)
  var resHtml = '';
  if (coll.resources !== undefined) {
    var resItems = coll.resources.length
      ? coll.resources.map(function(r) { return '<span class="coll-tile-res-pill">' + esc(r) + '</span>'; }).join('')
      : '<span class="coll-tile-res-none">none</span>';
    resHtml = '<hr class="coll-tile-divider">'
      + '<div class="coll-tile-res-hdr">Resource Types:</div>'
      + '<div class="coll-tile-res">' + resItems + '</div>';
  }
  var regHtml = regLabel
    ? '<div class="coll-tile-reg">' + esc(regLabel) + '</div>'
    : '';
  return '<div class="coll-tile' + (extraCls || '') + '" onclick="' + onclick + '">'
    + '<div class="coll-tile-top">'
    +   '<div class="coll-tile-icon">' + FOLDER_ICON + '</div>'
    +   '<div class="coll-tile-summary">'
    +     '<div class="coll-tile-name">' + esc(coll.plural) + '</div>'
    +     '<div class="coll-tile-count">' + coll.count + ' item' + (coll.count !== 1 ? 's' : '') + '</div>'
    +   '</div>'
    + '</div>'
    + descHtml
    + resHtml
    + regHtml
    + '</div>';
}

function docTile(singular, contenttype) {
  return '<div class="coll-tile coll-tile-meta" onclick="openDocument(\'' + esc(singular) + '\')">'
    + '<div class="coll-tile-top">'
    +   '<div class="coll-tile-icon">' + DOC_ICON + '</div>'
    +   '<div class="coll-tile-summary">'
    +     '<div class="coll-tile-name">' + esc(singular) + ' document</div>'
    +     '<div class="coll-tile-count">' + esc(contenttype || '') + '</div>'
    +   '</div>'
    + '</div>'
    + '</div>';
}

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
function renderValueTree(val, depth, model, entityPath, keyPath) {
  var c2c = optClickToCopy();
  var attrType = keyPath ? getExplicitAttrTypeAtPath(model, entityPath, keyPath) : null;
  var forceMono = attrType !== null && attrType !== 'string';
  function leaf(raw, display) {
    var cls = forceMono ? ' class="eg-mono"' : '';
    if (!c2c) return '<span' + cls + '>' + display + '</span>';
    return '<span class="eg-copyable' + (forceMono ? ' eg-mono' : '') + '" data-copy="' + esc(String(raw)) + '" onclick="egCopy(this.dataset.copy,\'\')">' + display + '</span>';
  }
  if (val === null)              return leaf('null', '<span class="vt-null">null</span>');
  if (val === undefined)         return '<span class="vt-null">undefined</span>';
  if (typeof val === 'boolean')  return leaf(val, String(val));
  if (typeof val === 'number')   return leaf(val, String(val));
  if (typeof val === 'string')   return leaf(val, esc(val));

  var indent = 'style="margin-left:' + (depth * 14) + 'px"';

  if (Array.isArray(val)) {
    if (val.length === 0) return '<span class="vt-empty">empty</span>';
    var items = val.map(function(item, idx) {
      var isComplex = item !== null && typeof item === 'object';
      var sep    = (isComplex && idx > 0) ? '<div class="vt-arr-sep"></div>' : '';
      var badge  = '<span class="vt-arr-idx">[' + idx + ']</span>';
      return sep + '<div class="vt-arr-item" ' + indent + '>'
           + badge + renderValueTree(item, depth) + '</div>';
    });
    return '<div class="vt-arr">' + items.join('') + '</div>';
  }

  // object / map
  var keys = Object.keys(val).sort();
  if (keys.length === 0) return '<span class="vt-empty">empty</span>';
  var rows = keys.map(function(k) {
    var child = val[k];
    var isComplex = child !== null && typeof child === 'object';
    var childKeyPath = keyPath ? keyPath.concat([k]) : null;
    if (isComplex) {
      return '<div class="vt-kv vt-kv-block" ' + indent + '>'
           + '<span class="vt-key">' + esc(k) + ':</span>'
           + renderValueTree(child, depth + 1, model, entityPath, childKeyPath)
           + '</div>';
    }
    return '<div class="vt-kv" ' + indent + '>'
         + '<span class="vt-key">' + esc(k) + ':</span> '
         + renderValueTree(child, depth + 1, model, entityPath, childKeyPath)
         + '</div>';
  });
  return '<div class="vt-obj">' + rows.join('') + '</div>';
}

function copyable(text) {
  if (!optClickToCopy()) return '<span class="eg-value">' + esc(text) + '</span>';
  return '<span class="eg-copyable" data-copy="' + esc(text) + '" onclick="egCopy(this.dataset.copy,\'\')">' + esc(text) + '</span>';
}

function copyableMonospace(text) {
  if (!optClickToCopy()) return '<span class="eg-value eg-mono">' + esc(text) + '</span>';
  return '<span class="eg-copyable eg-mono" data-copy="' + esc(text) + '" onclick="egCopy(this.dataset.copy,\'\')">' + esc(text) + '</span>';
}

function copyEntityJSON() {
  egCopy(JSON.stringify(_lastData, null, 2), 'JSON');
}

function copyMetaJSON() {
  if (_metaData) egCopy(JSON.stringify(_metaData, null, 2), 'JSON');
}

function toggleMetaBox() {
  var box    = document.getElementById('eg-meta-box');
  var twisty = document.getElementById('eg-meta-twisty');
  if (!box || !twisty) return;
  var opening = box.style.display === 'none';
  box.style.display = opening ? '' : 'none';
  twisty.textContent = opening ? '▼' : '▶';
  if (!opening) return;
  if (_metaData) {
    var svURL = normalizeURL(_state.serverURL || window.location.origin);
    box.innerHTML = renderMetaBoxContent(_metaData, _modelCache[svURL] || null);
    return;
  }
  box.innerHTML = '<div class="eg-loading">Loading\u2026</div>';
  var metaUrl = _lastData && _lastData.metaurl;
  if (!metaUrl) { box.innerHTML = '<div class="eg-row"><span class="eg-value" style="color:#aaa">No meta URL available</span></div>'; return; }
  fetch(metaUrl)
    .then(function(r) { if (!r.ok) throw new Error('HTTP ' + r.status); return r.json(); })
    .then(function(d) {
      _metaData = d;
      var svURL2 = normalizeURL(_state.serverURL || window.location.origin);
      box.innerHTML = renderMetaBoxContent(d, _modelCache[svURL2] || null);
    })
    .catch(function(e) { box.innerHTML = '<div class="eg-row"><span class="eg-value" style="color:#c00;font-family:monospace">' + esc((e && e.message) ? e.message : String(e)) + '</span></div>'; });
}

// Dispatch the meta box body to the format matching the current data view:
// List view gets a plain key/value table (like the entity's own Property
// table); Grid view keeps the richer label/row rendering (renderMetaContent).
function renderMetaBoxContent(d, model) {
  return _state.dataView === 'table' ? renderMetaTable(d) : renderMetaContent(d, model);
}

// Plain table rendering of the meta/details data, for List view. Mirrors the
// "<Type> Property" / "Value" table used for the entity's own scalar props,
// so the meta box looks consistent with the rest of List view instead of
// Grid view's more human-readable label/row layout.
function renderMetaTable(d) {
  var suppressed = { metaurl: 1 };
  if (_metaResourceIdField) suppressed[_metaResourceIdField] = 1;
  var keys = Object.keys(d).filter(function(k) { return !suppressed[k]; }).sort();
  if (!keys.length) return '<div class="eg-row"><span class="eg-value" style="color:#aaa">No details available</span></div>';
  var capType = capitalize(_metaEntityType);
  var html = '<table class="xr-table"><thead><tr><th>' + esc(capType) + ' Property</th><th>Value</th></tr></thead><tbody>';
  keys.forEach(function(k) {
    var val = d[k];
    var display;
    if (val == null) {
      display = '<span style="color:#999">null</span>';
    } else if (typeof val === 'object') {
      display = esc(JSON.stringify(val));
    } else {
      display = esc(String(val));
    }
    html += '<tr><td style="font-weight:bold;color:#444;width:200px">' + esc(k)
          + '</td><td>' + display + '</td></tr>';
  });
  html += '</tbody></table>';
  return html;
}

function renderMetaContent(d, model) {
  var html = '';
  var metaRendered = {};

  // Suppress the resource's own ID field — it's already in the page title context
  if (_metaResourceIdField) metaRendered[_metaResourceIdField] = 1;
  // Suppress internal/nav fields
  metaRendered.metaurl     = 1;
  // Mark defaultversionid/url as handled (rendered below after tech row)
  metaRendered.defaultversionid  = 1;
  metaRendered.defaultversionurl = 1;

  // 1. Temporal
  if (d.createdat)  html += '<div class="eg-row eg-temporal"><span class="eg-label">Created:</span>'  + copyableMonospace(d.createdat)  + '</div>';
  if (d.modifiedat) html += '<div class="eg-row eg-temporal"><span class="eg-label">Modified:</span>' + copyableMonospace(d.modifiedat) + '</div>';
  metaRendered.createdat  = 1;
  metaRendered.modifiedat = 1;

  // 2. Tech row: epoch + self/shortself/xid as copy buttons
  var techRow = '';
  if (d.epoch !== undefined) techRow += '<span class="eg-label">Epoch:</span><span class="eg-value eg-epoch">' + copyableMonospace(String(d.epoch)) + '</span>';
  if (d.self)      techRow += copyBtn('Self', d.self);
  if (d.shortself) techRow += copyBtn('ShortSelf', d.shortself);
  if (d.xid)       techRow += copyBtn('XID', d.xid);
  if (techRow) html += '<div class="eg-row eg-technical">' + techRow + '</div>';
  metaRendered.epoch     = 1;
  metaRendered.self      = 1;
  metaRendered.shortself = 1;
  metaRendered.xid       = 1;

  // 3. Default version ID with → View + URL ↗ buttons (after epoch)
  if (d.defaultversionid !== undefined) {
    var dvid = String(d.defaultversionid);
    var dvRow = copyableMonospace(dvid);
    dvRow += ' <button class="eg-link-btn eg-link-btn-nav" data-vid="' + esc(dvid) + '" '
           + 'onclick="navigateToVersionById(this.dataset.vid)">→ Visit</button>';
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
        var kSpan = optClickToCopy()
          ? '<span class="eg-label-key eg-copyable" data-copy="' + esc(k) + '" onclick="egCopy(this.dataset.copy,\'\')">' + esc(k) + '</span>'
          : '<span class="eg-label-key">' + esc(k) + '</span>';
        var vSpan = optClickToCopy()
          ? '<span class="eg-label-val eg-copyable" data-copy="' + esc(String(d.labels[k])) + '" onclick="egCopy(this.dataset.copy,\'\')">' + esc(String(d.labels[k])) + '</span>'
          : '<span class="eg-label-val">' + esc(String(d.labels[k])) + '</span>';
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
              + '<div class="eg-ext-complex-body">' + renderValueTree(v, 0, model, _state.path, [k]) + '</div>'
              + '</div>';
      }
    } else {
      // meta entity: use same logic as renderAttrRow with explicit-type-only monospace check
      var attrTypeMeta = getExplicitAttrType(model, _state.path, k);
      var isMono = isMonoSpecAttr(k, metaSpecLevel, _metaSing)
        || (attrTypeMeta !== null && attrTypeMeta !== 'string');
      html += row(labelFor(k, metaSpecLevel, _metaSing), isMono ? copyableMonospace(String(v)) : copyable(String(v)));
    }
  }
  specKeys.forEach(metaAttrRow);
  if (extKeys.length) {
    html += '<hr class="eg-ext-sep">';
    extKeys.forEach(metaAttrRow);
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

function renderEntityGrid(data) {
  var main = el('main-view');
  var depth = _state.path.length;

  // Meta page (depth 5) is replaced by the inline meta box on the resource page — redirect up
  if (depth === 5 && _state.path[4] === 'meta') {
    pushState({path: _state.path.slice(0, 4), editMode: false});
    return;
  }

  // ---- Resolve entity type from model (path-based, not field-based) ----
  var svBase   = (_state.serverURL || window.location.origin).replace(/\/$/, '');
  var modelKey = normalizeURL(svBase);

  // If model not yet cached, fetch it and re-render — first pass uses fallback
  if (!_modelCache.hasOwnProperty(modelKey)) {
    ensureModelCached(svBase, function() {
      if (_lastData === data) renderEntityGrid(data);
    });
  }
  var model      = _modelCache[modelKey] || null;
  var entityType = getSingularName(model, _state.path);

  var colls = findCollectionRefs(model, _state.path, data);
  var collKeys = {};
  colls.forEach(function(c) {
    collKeys[c.plural] = true;
    collKeys[c.plural + 'url'] = true;
    collKeys[c.plural + 'count'] = true;
  });

  // Attach model info to collection tiles
  if (model && model.groups) {
    if (depth === 0) {
      // Group-type tiles: attach resource type list + description from model
      colls.forEach(function(c) {
        var grpDef = model.groups[c.plural];
        c.resources   = grpDef && grpDef.resources ? Object.keys(grpDef.resources).sort() : [];
        c.description = (grpDef && grpDef.description) || '';
      });
    } else if (depth === 2) {
      // Resource-type tiles: attach description from model.groups[g].resources[r]
      var grpDef2 = model.groups[_state.path[0]];
      colls.forEach(function(c) {
        var resDef = grpDef2 && grpDef2.resources && grpDef2.resources[c.plural];
        c.description = (resDef && resDef.description) || '';
      });
    }
  }
  // ID field name is <singular>id (e.g. "dir" → "dirid"); last path segment as fallback
  var idFieldName = entityType.toLowerCase() + 'id';
  var idVal = data[idFieldName] !== undefined ? data[idFieldName]
            : _state.path.length > 0 ? _state.path[_state.path.length - 1] : data.registryid;

  var html = '<div class="eg-page">';

  // ---- Page title: SINGULAR: name-or-id ----
  var resSingular = (depth >= 6 && model)
    ? getSingularName(model, _state.path.slice(0, 4))
    : null;
  var titleDisplay = data.name || (idVal != null ? String(idVal) : '');
  var titleType = (depth >= 6 && resSingular)
    ? resSingular + ' VERSION'
    : entityType;
  var pageTitle = '<span class="eg-page-title-type">' + esc(titleType) + ':</span>';
  if (titleDisplay) {
    var titleId = optClickToCopy()
      ? '<span class="eg-page-title-id eg-copyable" data-copy="' + esc(titleDisplay) + '" onclick="egCopy(this.dataset.copy,\'\')">' + esc(titleDisplay) + '</span>'
      : '<span class="eg-page-title-id">' + esc(titleDisplay) + '</span>';
    pageTitle += ' ' + titleId;
  }
  if (data.icon) {
    var iconImg = '<img src="' + esc(data.icon) + '" class="eg-page-title-icon" alt="" '
                + 'onerror="this.style.display=\'none\';this.nextElementSibling.style.display=\'inline-flex\'">';
    if (optClickToCopy()) {
      pageTitle += '<span class="eg-copyable" data-copy="' + esc(data.icon) + '" onclick="egCopy(this.dataset.copy,\'Icon URL\')" title="Click to copy icon URL">'
                + iconImg + '</span>';
    } else {
      pageTitle += iconImg;
    }
    pageTitle += '<span class="eg-icon-err eg-page-title-icon-err" style="display:none" title="Failed to load: ' + esc(data.icon) + '">'
               + '<svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#ccc" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">'
               + '<rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/>'
               + '<polyline points="21 15 16 10 5 21"/>'
               + '<line x1="3" y1="3" x2="21" y2="21" stroke="#e88" stroke-width="1.5"/>'
               + '</svg></span>';
  }
  html += '<div class="eg-page-title">' + pageTitle + '</div>';

  // Check hasdocument from model for resource (depth 4) and version (depth 6+)
  var hasDocument = (depth === 4 || depth >= 6) && resourceHasDocument(model, _state.path);

  // ---- Collections ----
  if (colls.length || depth === 4 || depth === 0 || depth === 2 || (depth >= 6 && hasDocument)) {
    // Resources: no section header (only one collection type), but add meta tile
    if (depth !== 4 && depth < 6) {
      var collsLabel = _state.path.length === 0 ? 'GROUP TYPES' : 'RESOURCES';
      html += '<div class="eg-section-header">' + collsLabel + '</div>';
    }
    html += '<div class="eg-colls">';
    colls.forEach(function(c) { html += collectionTile(c); });
    if (hasDocument && (depth === 4 || depth >= 6)) {
      var docSingular = (depth >= 6 && resSingular) ? resSingular : entityType;
      html += docTile(docSingular, data.contenttype);
    }
    if (depth === 0 && colls.length === 0) {
      html += '<div class="eg-colls-empty">No group types defined</div>';
    }
    if (depth === 2 && colls.length === 0) {
      html += '<div class="eg-colls-empty">No resource types defined</div>';
    }
    html += '</div>';

    // At registry root (depth 0), show available server endpoints as extra tiles
    if (depth === 0) {
      var svBase0 = (_state.serverURL || window.location.origin).replace(/\/$/, '');
      var capData = _capCache[normalizeURL(svBase0)];
      var avail   = capData && capData.available;
      var sectionTiles = ['model','modelsource','capabilities','capabilitiesoffered'];
      var availSections = sectionTiles.filter(function(s) { return avail && avail[s]; });
      if (availSections.length) {
        html += '<div class="eg-section-header">REGISTRY ENDPOINTS</div>';
        html += '<div class="eg-colls">';
        var sectionNames = {model:'Model', modelsource:'Model Source', capabilities:'Capabilities', capabilitiesoffered:'Capabilities Offered'};
        availSections.forEach(function(s) {
          var mut = avail[s] && avail[s].mutable;
          var click = 'pushState({section:\'' + s + '\',editMode:false,useExport:false});return false';
          html += '<div class="eg-coll-tile" onclick="' + esc(click) + '">'
               +   '<div class="eg-coll-name">' + esc(sectionNames[s]) + '</div>'
               +   (mut ? '<div class="eg-coll-mutable-badge">mutable</div>' : '')
               + '</div>';
        });
        html += '</div>';
      }
    }
  }

  // ---- Resource Meta box (depth 4 only): collapsed by default, lazy-fetched on expand ----
  if (depth === 4) {
    _metaData = null;
    _metaResourceIdField = idFieldName;  // suppress resource's own ID in meta content
    html += '<div class="eg-section-header eg-details-header">'
          + '<span>' + esc(capitalize(entityType)) + ' Details'
          + ' <button class="eg-twisty" id="eg-meta-twisty" onclick="toggleMetaBox()">▶</button>'
          + '</span>'
          + '<button class="eg-copy-json-btn" onclick="copyMetaJSON()">{ } Copy JSON</button>'
          + '</div>';
    html += '<div class="eg-details eg-meta-details" id="eg-meta-box" style="display:none"></div>';
  }

  var capType = entityType.charAt(0).toUpperCase() + entityType.slice(1);
  var detailsLabel;
  if (depth === 0) {
    detailsLabel = 'Registry Details';
  } else if (depth === 2) {
    detailsLabel = capType + ' Details';
  } else if (depth === 4) {
    detailsLabel = defaultVersionLabel(capType, data) + ' Details';
  } else if (depth >= 6) {
    detailsLabel = 'Version Details';
  } else {
    detailsLabel = 'Details'; // meta (depth 5) — leave as is
  }
  html += '<div class="eg-section-header eg-details-header">' + detailsLabel
        + '<button class="eg-copy-json-btn" onclick="copyEntityJSON()">{ } Copy JSON</button>'
        + '</div>';
  html += '<div class="eg-details">';

  // Description first — human-readable text before attribute rows
  if (data.description) {
    html += optClickToCopy()
      ? '<div class="eg-description eg-copyable" data-copy="' + esc(data.description) + '" onclick="egCopy(this.dataset.copy,\'\')">' + esc(data.description) + '</div>'
      : '<div class="eg-description">' + esc(data.description) + '</div>';
  }

  // For version pages: compute parent resource info; doc ID appears after description
  var versionParentSingular = '';
  var versionParentIdField = '';
  if (depth >= 6) {
    if (model && model.groups && model.groups[_state.path[0]]) {
      var _resDef = model.groups[_state.path[0]].resources && model.groups[_state.path[0]].resources[_state.path[2]];
      if (_resDef) versionParentSingular = _resDef.singular;
    }
    if (!versionParentSingular) versionParentSingular = _state.path[2].replace(/s$/, '');
    versionParentIdField = versionParentSingular.toLowerCase() + 'id';
    if (data[versionParentIdField] !== undefined) {
      var _docId = String(data[versionParentIdField]);
      var _docRow = copyableMonospace(_docId)
        + ' <button class="eg-link-btn eg-link-btn-nav" onclick="navigateToParentResource()">→ Visit</button>';
      html += '<div class="eg-row"><span class="eg-label">' + esc(versionParentSingular + ' ID') + ':</span>'
            + '<span class="eg-value">' + _docRow + '</span></div>';
    }
  }

  // If name was used in the title, show ID: <id> after description
  if (data.name && idVal != null) {
    html += '<div class="eg-row"><span class="eg-label">' + esc(idFieldName) + ':</span>'
          + copyableMonospace(String(idVal)) + '</div>';
  }

  // Documentation
  if (data.documentation) {
    html += row('Documentation',
      '<a href="' + esc(data.documentation) + '" target="_blank" class="eg-link">'
      + esc(data.documentation) + '</a>'
      + (optClickToCopy() ? '<span class="eg-copyable eg-doc-copy" data-copy="' + esc(data.documentation) + '" onclick="egCopy(this.dataset.copy,\'URL\')" title="Copy URL">⧉</span>' : ''));
  }

  // Row 4: temporal — created on its own line, modified on the next
  if (data.createdat)  html += '<div class="eg-row eg-temporal"><span class="eg-label">Created:</span>' + copyableMonospace(data.createdat) + '</div>';
  if (data.modifiedat) html += '<div class="eg-row eg-temporal"><span class="eg-label">Modified:</span>' + copyableMonospace(data.modifiedat) + '</div>';

  // Row 5: epoch + self/shortself/xid as pill buttons
  var techRow = '';
  if (data.epoch !== undefined) techRow += '<span class="eg-label">Epoch:</span><span class="eg-value eg-epoch">' + copyableMonospace(String(data.epoch)) + '</span>';
  if (data.self)      techRow += copyBtn('Self', data.self);
  if (data.shortself) techRow += copyBtn('ShortSelf', data.shortself);
  if (data.xid)       techRow += copyBtn('XID', data.xid);
  if (techRow) html += '<div class="eg-row eg-technical">' + techRow + '</div>';

  // Row 6: labels
  if (data.labels && typeof data.labels === 'object') {
    var labelKeys = Object.keys(data.labels).sort();
    if (labelKeys.length) {
      var labelParts = labelKeys.map(function(k) {
        var kSpan = optClickToCopy()
          ? '<span class="eg-label-key eg-copyable" data-copy="' + esc(k) + '" onclick="egCopy(this.dataset.copy,\'\')">' + esc(k) + '</span>'
          : '<span class="eg-label-key">' + esc(k) + '</span>';
        var vSpan = optClickToCopy()
          ? '<span class="eg-label-val eg-copyable" data-copy="' + esc(String(data.labels[k])) + '" onclick="egCopy(this.dataset.copy,\'\')">' + esc(String(data.labels[k])) + '</span>'
          : '<span class="eg-label-val">' + esc(String(data.labels[k])) + '</span>';
        return '<span class="eg-label-pair">' + kSpan + vSpan + '</span>';
      });
      html += '<div class="eg-row eg-labels"><span class="eg-label">Labels:</span>'
            + '<span class="eg-label-list">' + labelParts.join('') + '</span></div>';
    }
  }

  // Row 7: resource/version/meta-specific spec fields shown above the separator
  var extraRendered = {};

  if (depth === 4) {
    // Resource: show default Version ID and Ancestor Version ID; suppress isdefault
    if (data.versionid !== undefined) {
      var _vid = String(data.versionid);
      var _vidRow = copyableMonospace(_vid)
        + ' <button class="eg-link-btn eg-link-btn-nav" data-vid="' + esc(_vid) + '" '
        + 'onclick="navigateToVersionById(this.dataset.vid)">→ Visit</button>';
      html += '<div class="eg-row"><span class="eg-label">Version ID:</span>'
            + '<span class="eg-value">' + _vidRow + '</span></div>';
    }
    extraRendered.versionid = 1;
    if (data.ancestor !== undefined && data.ancestor !== null) {
      var _anc = String(data.ancestor);
      var _ancRow = copyableMonospace(_anc)
        + ' <button class="eg-link-btn eg-link-btn-nav" data-vid="' + esc(_anc) + '" '
        + 'onclick="navigateToVersionById(this.dataset.vid)">→ Visit</button>';
      html += '<div class="eg-row"><span class="eg-label">Ancestor Version ID:</span>'
            + '<span class="eg-value">' + _ancRow + '</span></div>';
    }
    extraRendered.ancestor  = 1;
    extraRendered.isdefault = 1;  // hide — always true for the default version
    extraRendered.metaurl   = 1;  // suppress — accessible via the meta tile
  } else if (depth >= 6) {
    // Version: doc ID already rendered at top; show ancestor version id with Visit, and isdefault
    extraRendered[versionParentIdField] = 1;  // already rendered above
    if (data.ancestor !== undefined && data.ancestor !== null) {
      var _vancId = String(data.ancestor);
      var _vancRow = copyableMonospace(_vancId)
        + ' <button class="eg-link-btn eg-link-btn-nav" data-vid="' + esc(_vancId) + '" '
        + 'onclick="navigateToVersionById(this.dataset.vid)">→ Visit</button>';
      html += '<div class="eg-row"><span class="eg-label">Ancestor Version ID:</span>'
            + '<span class="eg-value">' + _vancRow + '</span></div>';
    }
    extraRendered.ancestor = 1;
    if (data.isdefault !== undefined) {
      html += row('Is Default', copyableMonospace(String(data.isdefault)));
    }
    extraRendered.isdefault = 1;
  }

  // Split remaining keys into spec-defined (above <hr>) vs user extensions (below <hr>).
  var renderedAttrs = {
    labels:1, name:1, description:1, documentation:1, icon:1,
    createdat:1, modifiedat:1, epoch:1,
    self:1, shortself:1, xid:1, metaurl:1
  };
  renderedAttrs[idFieldName] = 1;
  Object.keys(extraRendered).forEach(function(k) { renderedAttrs[k] = 1; });

  var specLevel = specAttrLevel(_state.path);
  // singular for "id" expansion; resourceSingular for "$RESOURCE*" expansion
  var _singular = entityType.toLowerCase();
  var _resSing  = (depth === 4) ? _singular
                : (depth >= 6 && resSingular) ? resSingular.toLowerCase()
                : null;
  var remainingKeys = Object.keys(data).filter(function(k) {
    return !renderedAttrs[k] && !collKeys[k];
  }).sort();

  function renderAttrRow(k) {
    var v = data[k];
    if (v !== null && typeof v === 'object') {
      var isEmpty = Array.isArray(v) ? v.length === 0 : Object.keys(v).length === 0;
      if (isEmpty) {
        html += row(labelFor(k, specLevel, _singular), '<span class="vt-empty">empty</span>');
      } else {
        html += '<div class="eg-ext-complex">'
              + '<div class="eg-ext-complex-key">' + esc(labelFor(k, specLevel, _singular)) + ':</div>'
              + '<div class="eg-ext-complex-body">' + renderValueTree(v, 0, model, _state.path, [k]) + '</div>'
              + '</div>';
      }
    } else {
      // Monospace decision:
      // 1. String-typed spec attrs listed in MONO_ATTRS (generated from AttrInternals.uiMonospace)
      //    that are confirmed spec attrs at THIS entity level → always monospace.
      // 2. Explicitly model-named (non-wildcard) attrs with non-string type → monospace.
      //    Extension attrs that only match the '*' wildcard use null type → not monospace.
      var attrType = getExplicitAttrType(model, _state.path, k);
      var isMono = isMonoSpecAttr(k, specLevel, _singular)
        || (attrType !== null && attrType !== 'string');
      var valHtml = isMono ? copyableMonospace(String(v)) : copyable(String(v));
      html += row(labelFor(k, specLevel, _singular), valHtml);
    }
  }

  var specKeys = remainingKeys.filter(function(k) { return  isSpecAttr(k, specLevel, _singular, _resSing); });
  var extKeys  = remainingKeys.filter(function(k) { return !isSpecAttr(k, specLevel, _singular, _resSing); });

  specKeys.forEach(renderAttrRow);

  if (extKeys.length) {
    html += '<hr class="eg-ext-sep">';
    extKeys.forEach(renderAttrRow);
  }

  html += '</div></div>';
  main.innerHTML = html;
}

// ---- JSON view -----------------------------------------------------------

function renderJSONView(data) {
  renderJSONLeftPanel();
  var jsonHtml = addTwisties(syntaxHighlight(JSON.stringify(data, null, 2)));
  el('main-view').innerHTML =
    '<div class="json-exp-wrap">' +
      '<span class="json-exp-btn" id="json-exp-btn" data-open="false"' +
      ' onclick="jsonToggleAll()" title="Expand all">&#9656; all</span>' +
    '</div>' +
    '<div id="json-output" tabindex="0">' + jsonHtml + '</div>';
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

function renderJSONLeftPanel() {
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
      if (_state.dataView === 'json' || _state.view === 'json') renderJSONLeftPanel();
    });
  }

  // Trigger capability fetch if not yet cached; re-render when ready
  if (!_capCache.hasOwnProperty(normUrl2)) {
    ensureCapCached(svBase2, function() {
      if (_state.dataView === 'json' || _state.view === 'json') renderJSONLeftPanel();
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
  // doubled-up line.
  var filterHasApplyDivider = _state.section === 'data' && hasF('filter');
  if (_state.path.length === 0) {
    var hasModel   = avail2 && avail2.model;
    var hasMSource  = avail2 && avail2.modelsource;
    var hasCap     = avail2 && avail2.capabilities;
    var hasCapOff  = avail2 && avail2.capabilitiesoffered;
    var hasExport  = avail2 && avail2['export'];
    if (hasModel || hasMSource || hasCap || hasCapOff || hasExport) {
      html += '<div class="lp-section"><div class="lp-title">Registry Endpoints</div>';
      // "Model (Source)" and "Capabilities (Offered)" share one line each
      // (matches the old ui.go layout) instead of 4 stacked rows, to save
      // vertical space — the sub-link only appears when that endpoint is
      // actually available.
      html += lpNavPairRow('Model', 'model', hasModel,
        'Source', 'modelsource', hasMSource);
      html += lpNavPairRow('Capabilities', 'capabilities', hasCap,
        'Offered', 'capabilitiesoffered', hasCapOff);
      // Export as a nav item — toggles useExport on the registry root data view
      if (hasExport && _state.section === 'data') {
        var exportActive = _state.useExport;
        html += '<div class="lp-nav-item' + (exportActive ? ' lp-nav-active' : '') + '" '
          + 'onclick="pushState({useExport:' + (!exportActive) + ',section:\'data\'})">'
          + 'Export' + (exportActive ? ' ✓' : '') + '</div>';
      }
      // "Data" link to return from a section view
      if (_state.section !== 'data') {
        html += '<div class="lp-nav-item" onclick="pushState({section:\'data\',path:[],editMode:false,useExport:false})">← Registry Data</div>';
      }
      html += '</div>';
      if (!filterHasApplyDivider) html += '<hr class="lp-divider">';
    }
  }

  var hasOpts = false; // true when there's at least one filter/option/inline to apply

  if (_state.section === 'data') {
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
        + '<button class="lp-apply lp-apply-top" onclick="applyJSONOptions()">'
        + 'Apply</button>'
        + '<span class="lp-divider-line"></span></div>'
        + '<div class="lp-section" id="lp-filter-section">'
        + fbFiltersTitleHTML(fbFilterCount(_fbDraft.groups))
        + (_filtersCollapsed ? '' : buildFilterSectionInner(model2))
        + '</div><hr class="lp-divider">';
    }

    // Options — only show individual options whose flag is enabled
    var optHtml = '';
    if (hasF('doc'))         optHtml += lpCheck('lp-doc', 'doc view',    _state.docView);
    if (hasF('binary'))      optHtml += lpCheck('lp-bin', 'binary',      _state.binary);
    if (hasF('collections')) optHtml += lpCheck('lp-col', 'collections', _state.collections);
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
                + '<input type="checkbox" class="lp-inline-cb" value="' + esc(opt.value + '.*') + '"' + dschecked + '>'
                + '<span class="lp-dotstar-label">.*</span>'
                + '</span>'
            : '<span class="lp-dotstar"></span>';
          html += '<div class="' + rowCls + '">'
            + '<input type="checkbox" class="lp-inline-cb" value="' + esc(opt.value) + '"' + checked + '>'
            + '<span class="lp-inline-label">' + esc(opt.label) + '</span>'
            + dotStarHtml
            + '</div>';
          rowIdx++;
        });
        html += '</div><hr class="lp-divider">';
      }
    }
  }

  if (!html)    html = '<div class="lp-no-opts">No options</div>';
  if (hasOpts) {
    // Reuse the divider-apply combo (line + centered Apply + line) instead
    // of a separate full-width button; strip the trailing plain divider
    // left by the last section so we don't get doubled-up lines.
    var trailingHr = '<hr class="lp-divider">';
    if (html.slice(-trailingHr.length) === trailingHr) {
      html = html.slice(0, -trailingHr.length);
    }
    html += '<div class="lp-divider-apply">'
      + '<span class="lp-divider-line"></span>'
      + '<button class="lp-apply lp-apply-top" onclick="applyJSONOptions()">'
      + 'Apply</button>'
      + '<span class="lp-divider-line"></span></div>';
  }
  inner.innerHTML = html;
}

function lpCheck(id, label, checked) {
  return '<div class="lp-item"><input type="checkbox" id="' + id + '"'
    + (checked ? ' checked':'') + '> ' + label + '</div>';
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
    row += '<span class="' + mainCls + '" onclick="pushState('
      + '{section:\'' + mainSection + '\',editMode:false,'
      + 'useExport:false})">' + esc(mainLabel) + '</span>';
  } else {
    row += esc(mainLabel);
  }
  if (hasSub) {
    var subActive = _state.section === subSection;
    var subCls = 'lp-nav-item lp-nav-inline' +
      (subActive ? ' lp-nav-active' : '');
    row += ' <span class="lp-nav-sub">(<span class="' + subCls
      + '" onclick="pushState({section:\'' + subSection + '\','
      + 'editMode:false,useExport:false})">' + esc(subLabel)
      + '</span>)</span>';
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
  }
}

function fbRerender() {
  var host = el('lp-filter-section');
  if (!host) return;
  var svBase = serverBase();
  var model  = _modelCache[normalizeURL(svBase)] || null;
  var count  = fbFilterCount(_fbDraft ? _fbDraft.groups : []);
  host.innerHTML = fbFiltersTitleHTML(count)
    + (_filtersCollapsed ? '' : buildFilterSectionInner(model));
}

// Renders the collapsible "Filters" section title: label + "(N)" count
// of currently-defined filter expressions (shown even while collapsed,
// so you can tell at a glance whether any are set) + twisty (▶/▼) on
// the right.
function fbFiltersTitleHTML(count) {
  var twisty = _filtersCollapsed ? '▶' : '▼';
  var countHTML = count
    ? ' <span class="lp-title-count">(' + count + ')</span>' : '';
  return '<div class="lp-title lp-title-collapsible" '
    + 'onclick="fbToggleCollapsed()">'
    + '<span>Filters' + countHTML + '</span>'
    + '<span class="lp-title-twisty lp-title-twisty-right">' + twisty
    + '</span></div>';
}

// Total number of leaf filter expressions across all OR-groups (each
// group is a comma-joined AND-list) — shown as "(N)" next to the
// Filters title even while the section is collapsed.
function fbFilterCount(groups) {
  if (!groups || !groups.length) return 0;
  var n = 0;
  groups.forEach(function(g) { n += trimSplit(g, ',').length; });
  return n;
}

// Toggles the Filters section's collapsed/expanded state. Always starts
// collapsed on a fresh page load (see _filtersCollapsed init); this just
// flips it for the current session/navigation.
function fbToggleCollapsed() {
  _filtersCollapsed = !_filtersCollapsed;
  renderJSONLeftPanel();
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
    html += '<textarea class="lp-filter-area" id="lp-filters-raw">'
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
    html += '<input type="text" class="fb-seg-custom" placeholder="name"'
      + ' value="' + esc(seg && seg.kind === 'custom' ? seg.text : '') + '"'
      + ' onchange="' + custOnchg + '">';
  }
  return '<div class="fb-seg-row">' + html + '</div>';
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

function applyJSONOptions() {
  var doc = el('lp-doc'), bin = el('lp-bin'), col = el('lp-col');
  var cbs = qsa('.lp-inline-cb');
  var inlines = [];
  cbs.forEach(function(cb) { if (cb.checked) inlines.push(cb.value); });
  pushState({
    filters:     fbCollectFilters(),
    docView:     doc ? doc.checked : false,
    binary:      bin ? bin.checked : false,
    collections: col ? col.checked : false,
    inlines:     inlines
  });
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
    pushState({
      view: 'table', section: 'data', path: segments, editMode: false,
      filters: filtersFromUrl(raw)
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
            var href, target = '', onclick;
            if (urlPath.indexOf(svBase) === 0) {
              // Same-server: build SPA href for right-click "open in new
              // tab" support. Use the filter (if any) actually embedded
              // in THIS url value — not the UI's current filter state —
              // so the hover-preview href and the actual navigation
              // target both match what the server returned, instead of
              // re-blending in a stale/unrelated filter.
              var rel      = urlPath.slice(svBase.length).replace(/^\//, '');
              var segments = rel ? rel.split('/') : [];
              var fakeSt   = Object.assign({}, _state, {
                view: 'table', section: 'data', path: segments, editMode: false,
                filters: filtersFromUrl(inner)
              });
              href    = buildURL(fakeSt);
              onclick = 'return navigateJsonUrl(\'' + inner.replace(/\\/g,'\\\\').replace(/'/g,"\\'") + '\')';
            } else {
              href    = inner;
              target  = ' target="_blank" rel="noopener"';
              onclick = '';
            }
            var attrOnclick = onclick ? ' onclick="' + onclick + '"' : '';
            return '<span class="' + c + '"><a class="json-url" href="' + esc(href) + '"' + target + attrOnclick + '>' + m + '</a></span>';
          }
        }

        return '<span class="' + c + '">' + m + '</span>';
      });
}

// ---- Collection helpers --------------------------------------------------

// Extract entity items from a collection response.
// A collection response is a flat JSON object keyed by <singular>id.
// Pagination metadata lives in the Link response header (not in the body).
function collectionItems(data) {
  if (!data || typeof data !== 'object') return [];
  var items = [];
  Object.keys(data).forEach(function(k) {
    var v = data[k];
    if (typeof v === 'object' && v !== null && !Array.isArray(v)) {
      // Attach the map key in case xid is absent (shouldn't happen per spec but be safe)
      items.push(Object.assign({__mapKey: k}, v));
    }
  });
  items.sort(function(a, b) { return itemNavKey(a).localeCompare(itemNavKey(b)); });
  return items;
}

// The navigation key for an item — last segment of xid, or the map key.
// xid is a relative URL like "/endpoints/ep1"; last segment = "ep1".
function itemNavKey(item) {
  if (item.xid) {
    var segs = item.xid.replace(/^\//, '').split('/');
    return segs[segs.length - 1] || item.__mapKey || '';
  }
  return item.__mapKey || '';
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
    'versionid','isdefault','ancestor','contenttype'];
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

function navigateTo(id) {
  // If navigating INTO a collection from the registry root or single entity,
  // the id IS the collection name (e.g., "endpoints") and we just append it.
  pushState({path: _state.path.concat([id]), editMode: false});
}

// Navigate to a specific version from the meta page (path: [..., resource, rId, "meta"])
function navigateToVersion(vId) {
  var basePath = _state.path.slice(0, -1); // strip "meta"
  pushState({path: basePath.concat(['versions', vId]), editMode: false});
}

// Navigate to a version by ID from the current resource or version context
function navigateToVersionById(vId) {
  var basePath = _state.path.slice(0, 4); // [G, gId, R, rId]
  pushState({path: basePath.concat(['versions', vId]), editMode: false});
}

// Navigate to the parent resource from a version page
function navigateToParentResource() {
  pushState({path: _state.path.slice(0, 4), editMode: false});
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
  if (_modelDirty || _capDirty) { e.preventDefault() ; e.returnValue = '' ; }
}) ;

function showLeaveEditDialog(onSave, onDiscard) {
  var overlay = document.createElement('div') ;
  overlay.style.cssText = 'position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.35);z-index:9999;display:flex;align-items:center;justify-content:center;' ;
  var box = document.createElement('div') ;
  box.style.cssText = 'background:white;border-radius:8px;padding:24px;box-shadow:0 4px 24px rgba(0,0,0,0.25);max-width:340px;width:90%;font-family:sans-serif;' ;
  var msg = document.createElement('p') ; msg.textContent = 'You have unsaved changes.' ;
  msg.style.cssText = 'margin:0 0 20px;font-size:14px;color:#333;' ;
  box.appendChild(msg) ;
  var btns = document.createElement('div') ; btns.style.cssText = 'display:flex;gap:8px;justify-content:flex-end;' ;
  function mkBtn(label, fn, css) {
    var b = document.createElement('button') ; b.textContent = label ;
    b.style.cssText = 'padding:6px 16px;border-radius:5px;cursor:pointer;font-size:13px;font-weight:bold;' + css ;
    b.onclick = function() { document.body.removeChild(overlay) ; fn() ; } ;
    btns.appendChild(b) ;
  }
  mkBtn('Cancel',  function(){},  'background:#f0f0f0;color:#333;border:1px solid #ccc;') ;
  mkBtn('Discard', onDiscard,     'background:#f8d7da;color:#721c24;border:1px solid #f5c6cb;') ;
  mkBtn('Save',    onSave,        'background:#2060a0;color:white;border:1px solid #2060a0;') ;
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
    errDiv.id = 'editorError' ; errDiv.style.display = 'none' ;
    div.appendChild(errDiv) ;
  }

  // Auto-select 'fields' when entering registry root or group/resource level with nothing selected
  if (_navSelected === null) {
    if (_navTab === 'registry' && _navPath.length === 0) _navSelected = 'fields' ;
    else if (_navTab === 'groups' && (_navPath.length === 1 || _navPath.length === 3)) _navSelected = 'fields' ;
  }

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

  function navItem(label, isContainer, isSelected, clickFn, deleteFn) {
    var el = document.createElement('div') ;
    el.className = 'navItem' + (isSelected ? ' navItemSelected' : '') ;
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
    el.onclick = clickFn ; return el ;
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
      div.appendChild(navItem(withCount('Attributes', Object.keys(model.attributes||{}).length), true, false, function() { drillDown(['attributes']) ; })) ;
      div.appendChild(navItem(withCount('Group Types', Object.keys(model.groups||{}).length), true, false, function() { changeTab('groups') ; })) ;
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
        div.appendChild(navItem(withCount(k, rCount), true, false,
          (function(key){ return function(){ drillDown([key]) ; } ; })(k),
          (function(key){ return function(){ deleteGroup(key) ; } ; })(k))) ;
      }) ;
    } else if (_navPath.length === 1) {
      var gk = _navPath[0] ;
      var grpData = model.groups[gk] || {} ;
      div.appendChild(navItem('Details', false, _navSelected === 'fields', function() { selectItem('fields') ; })) ;
      div.appendChild(navItem(withCount('Attributes', Object.keys(grpData.attributes||{}).length), true, false, function() { drillDown([gk, 'attributes']) ; })) ;
      div.appendChild(navItem(withCount('Resources', Object.keys(grpData.resources||{}).length), true, false, function() { drillDown([gk, 'resources']) ; })) ;
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
        div.appendChild(navItem(k, true, false,
          (function(key){ return function(){ drillDown([gk, 'resources', key]) ; } ; })(k),
          (function(key){ return function(){ deleteResource(gk, key) ; } ; })(k))) ;
      }) ;
    } else if (_navPath.length === 3) {
      var gk = _navPath[0], rk = _navPath[2] ;
      var resData = ((model.groups[gk]||{}).resources||{})[rk] || {} ;
      div.appendChild(navItem('Details', false, _navSelected === 'fields', function() { selectItem('fields') ; })) ;
      div.appendChild(navItem(withCount('Version Attrs', Object.keys(resData.attributes||{}).length), true, false, function(){ drillDown([gk,'resources',rk,'versionattributes']) ; })) ;
      div.appendChild(navItem(withCount('Resource Attrs', Object.keys(resData.resourceattributes||{}).length), true, false, function(){ drillDown([gk,'resources',rk,'resourceattributes']) ; })) ;
      div.appendChild(navItem(withCount('Meta Attrs', Object.keys(resData.metaattributes||{}).length), true, false, function(){ drillDown([gk,'resources',rk,'metaattributes']) ; })) ;
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
  // Preserve nested structures edited via drill-down (not part of this form)
  if (existing.attributes) attr.attributes = existing.attributes ;
  if (existing.item) attr.item = existing.item ;
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
  div.appendChild(ef('ef_icon', 'Icon URL', grp.icon||'')) ;
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
  div.appendChild(ef('ef_icon', 'Icon URL', r.icon||'')) ;
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

  // If-Values drill-down button — left-aligned under section header
  var ifvSec = document.createElement('div') ; ifvSec.className = 'editorSectionLabel' ; ifvSec.textContent = 'If-Values' ;
  div.appendChild(ifvSec) ;
  var ifvCount = Object.keys(attr.ifvalues || {}).length ;
  if (_modelReadOnly && !ifvCount) {
    var ifvNone = document.createElement('span') ; ifvNone.textContent = '\u2014 none \u2014' ;
    ifvNone.style.cssText = 'color:#aaa;font-size:12px;font-style:italic;margin-left:4px;' ;
    div.appendChild(ifvNone) ;
  } else {
    var ifvBtn = document.createElement('button') ; ifvBtn.className = 'editorBtn navDrillBtn' ;
    ifvBtn.style.cssText = 'font-size:11px;padding:3px 8px;margin-bottom:6px;' ;
    ifvBtn.textContent = '\u25b6 If-Values' + (ifvCount ? ' ('+ifvCount+')' : '') ;
    ifvBtn.onclick = function() {
      collectCurrentEditor() ;
      var resolvedKey = _navSelected || currentAttrKey ;
      _attrNestStack.push({key:resolvedKey, isIfValues:true}) ;
      _navSelected = null ; renderEditor() ;
    } ;
    div.appendChild(ifvBtn) ;
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
  var seg = document.createElement('div') ;
  var cur = (value === true) ? 'true' : (value === false) ? 'false' : '' ;
  seg.className = 'boolSeg' + (_modelReadOnly ? ' boolSegReadOnly' : '') ;
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
  var wrap = document.createElement('div') ; wrap.className = 'labelsWrap' ;
  if (!_modelReadOnly) {
    var addBtn = document.createElement('button') ; addBtn.className = 'editorBtn' ;
    addBtn.textContent = '+ Add' ; addBtn.title = 'Add Label' ;
    addBtn.style.flexShrink = '0' ; addBtn.style.alignSelf = 'flex-start' ;
    addBtn.onclick = function() { rowsDiv.appendChild(makeLabelRow('','')) ; } ;
    wrap.appendChild(addBtn) ;
  }
  var rowsDiv = document.createElement('div') ; rowsDiv.className = 'labelsRows' ;
  rowsDiv.id = containerId ;
  Object.keys(labels).forEach(function(k) { rowsDiv.appendChild(makeLabelRow(k, labels[k])) ; }) ;
  wrap.appendChild(rowsDiv) ;
  sec.appendChild(wrap) ;
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
  var wrap = document.createElement('div') ; wrap.className = 'labelsWrap' ;
  if (!_modelReadOnly) {
    var addBtn = document.createElement('button') ; addBtn.className = 'editorBtn' ;
    addBtn.textContent = '+ Add' ; addBtn.title = 'Add enum value' ;
    addBtn.style.flexShrink = '0' ; addBtn.style.alignSelf = 'flex-start' ;
    addBtn.onclick = function() { rowsDiv.appendChild(makeEnumRow('')) ; } ;
    wrap.appendChild(addBtn) ;
  }
  var rowsDiv = document.createElement('div') ; rowsDiv.id = containerId ;
  rowsDiv.style.cssText = 'flex:1;display:flex;flex-direction:column;gap:4px;' ;
  values.forEach(function(v) { rowsDiv.appendChild(makeEnumRow(String(v))) ; }) ;
  wrap.appendChild(rowsDiv) ;
  sec.appendChild(wrap) ;
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
  if (_modelReadOnly && Object.keys(constraints||{}).length === 0) {
    var none = document.createElement('span') ; none.textContent = '\u2014 none \u2014' ;
    none.style.cssText = 'color:#aaa;font-size:12px;font-style:italic;margin-left:4px;' ;
    sec.appendChild(none) ; return sec ;
  }
  var blocksDiv = document.createElement('div') ; blocksDiv.id = containerId ;
  blocksDiv.style.cssText = 'display:flex;flex-direction:column;' ;
  Object.keys(constraints||{}).forEach(function(k) {
    blocksDiv.appendChild(makeConstraintRow(k, constraints[k]||{}, gk)) ;
  }) ;
  sec.appendChild(blocksDiv) ;
  if (!_modelReadOnly) {
    var addBtn = document.createElement('button') ; addBtn.className = 'editorBtn' ;
    addBtn.textContent = '+ Add' ; addBtn.title = 'Add Constraint' ;
    addBtn.style.cssText = 'margin-top:4px;align-self:flex-start;' ;
    addBtn.onclick = function() { blocksDiv.appendChild(makeConstraintRow('', {}, gk)) ; } ;
    sec.appendChild(addBtn) ;
  }
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
    // Read-only: show fields as text
    function roRow(lbl, val) {
      var row = document.createElement('div') ; row.className = 'editorField' ;
      var l = document.createElement('label') ; l.textContent = lbl ;
      var s = document.createElement('span') ; s.style.cssText = 'font-size:13px;color:#333;' ;
      s.textContent = val || '\u2014' ; row.appendChild(l) ; row.appendChild(s) ; return row ;
    }
    var pathStr = key || '\u2014' ;
    block.appendChild(roRow('Path:', pathStr)) ;
    var defStr = constraint.default !== undefined ? JSON.stringify(constraint.default) : '' ;
    if (defStr) block.appendChild(roRow('Default:', defStr)) ;
    if (constraint.equals) block.appendChild(roRow('Equals:', constraint.equals)) ;
    var enumArr = Array.isArray(constraint.enum) ? constraint.enum : [] ;
    if (enumArr.length) block.appendChild(roRow('Enum:', enumArr.join(', '))) ;
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

  pathRow.appendChild(resSel) ; pathRow.appendChild(dot) ; pathRow.appendChild(attrSel) ;
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
  var eqSel = document.createElement('select') ; eqSel.className = 'cstrEqSel editorSelectWrap editorInput' ;
  var eqBlank = document.createElement('option') ; eqBlank.value = '' ; eqBlank.textContent = '-- none --' ;
  eqSel.appendChild(eqBlank) ;
  var grpAttrs = getScalarAttrNames((((_modelData||{}).groups||{})[gk]||{}).attributes||{}) ;
  grpAttrs.forEach(function(n) {
    var o = document.createElement('option') ; o.value = n ; o.textContent = n ;
    if (n === (constraint.equals||'')) o.selected = true ;
    eqSel.appendChild(o) ;
  }) ;
  eqRow.appendChild(eqLbl) ; eqRow.appendChild(eqSel) ;
  block.appendChild(eqRow) ;

  // Enum editor
  var enumSec = document.createElement('div') ; enumSec.className = 'cstrEnumSection' ;
  var enumHdr = document.createElement('div') ; enumHdr.className = 'editorSectionLabel' ;
  enumHdr.textContent = 'Enum' ; enumSec.appendChild(enumHdr) ;
  var enumWrap = document.createElement('div') ; enumWrap.className = 'labelsWrap' ;
  var enumAddBtn = document.createElement('button') ; enumAddBtn.className = 'editorBtn' ;
  enumAddBtn.textContent = '+ Add' ; enumAddBtn.title = 'Add enum value' ;
  enumAddBtn.style.flexShrink = '0' ; enumAddBtn.style.alignSelf = 'flex-start' ;
  var enumRowsDiv = document.createElement('div') ; enumRowsDiv.id = 'cstr_enum_' + idx ;
  enumRowsDiv.style.cssText = 'flex:1;display:flex;flex-direction:column;gap:4px;' ;
  enumAddBtn.onclick = function() { enumRowsDiv.appendChild(makeEnumRow('')) ; } ;
  var enumArr2 = Array.isArray(constraint.enum) ? constraint.enum : [] ;
  enumArr2.forEach(function(v) { enumRowsDiv.appendChild(makeEnumRow(String(v))) ; }) ;
  enumWrap.appendChild(enumAddBtn) ; enumWrap.appendChild(enumRowsDiv) ;
  enumSec.appendChild(enumWrap) ; block.appendChild(enumSec) ;

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
  if (errDiv) { errDiv.style.display = 'none' ; errDiv.textContent = '' ; }

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
        fetch(svBaseSave + '/model')
          .then(function(r) { return r.json() ; })
          .then(function(m) { _modelCache[mKey] = m ; })
          .catch(function()  { delete _modelCache[mKey] ; })
          .then(function() {
            if (onSuccess) { onSuccess() ; } else { window.location.reload() ; }
          }) ;
      } else { if (errDiv) { errDiv.style.display = 'block' ; errDiv.textContent = 'Error (' + resp.status + '):\n' + text ; } }
    }) ;
  }).catch(function(err) {
    removeOverlay() ;
    if (errDiv) { errDiv.style.display = 'block' ; errDiv.textContent = 'Network error: ' + err.message ; }
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
  if (errDiv) { errDiv.style.display = 'none'; errDiv.textContent = ''; }

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
        if (onSuccess) { onSuccess(); } else { window.location.reload(); }
      } else {
        if (errDiv) { errDiv.style.display = 'block'; errDiv.textContent = 'Error (' + resp.status + '):\n' + text; }
      }
    });
  }).catch(function(err) {
    removeOverlay();
    if (errDiv) { errDiv.style.display = 'block'; errDiv.textContent = 'Network error: ' + err.message; }
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
// control. `locked` forces read-only regardless of _capReadOnly.
function capBoolToggle(value, locked, onChange) {
  var seg = document.createElement('div');
  var readOnly = _capReadOnly || locked;
  seg.className = 'boolSeg' + (readOnly ? ' boolSegReadOnly' : '');
  var cur = value === true ? 'true' : 'false';
  seg.dataset.val = cur;
  [['true', 'true'], ['false', 'false']].forEach(function(b) {
    var btn = document.createElement('button'); btn.type = 'button';
    btn.textContent = b[0];
    btn.className = 'boolSegBtn' + (cur === b[1] ? ' boolSegActive' : '');
    btn.onclick = function() {
      if (readOnly) return;
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

function capitalize(s) {
  s = String(s || '');
  return s.charAt(0).toUpperCase() + s.slice(1);
}

// Shared "Default <Type> Version (n)" label fragment used by both the Grid
// view's Details header and the List view's Property header at depth 4, so
// the two stay in sync. capType is the capitalized resource singular name.
function defaultVersionLabel(capType, data) {
  return 'Default ' + capType + ' Version'
    + (data && data.versionid !== undefined ? ' (' + esc(String(data.versionid)) + ')' : '');
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
