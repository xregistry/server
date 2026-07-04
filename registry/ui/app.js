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
  homeLayout:  'grid',      // 'grid' | 'list'      — overridden from localStorage in init()
  dataView:    'grid',  // 'grid' | 'table' | 'json'
  serverURL:   '',      // full URL to registry root, e.g. 'http://localhost:8080'
                        // '' = same origin as the SPA
  section:     'data',  // 'data' | 'model' | 'capabilities'
  path:        [],      // path segments relative to registry root (data section only)
  editMode:    false,
  mutable:     false,

  // JSON-view query options
  inlines:     [],
  filters:     [],
  docView:     false,
  binary:      false,
  collections: false,
};

// ---- Server/registry management (localStorage) ---------------------------

var LS_SERVERS     = 'xreg-servers';
var LS_OPTIONS     = 'xreg-options';
var _labelCache    = {};  // normalizedURL → probed registry name
var _modelCache    = {};  // normalizedURL → model JSON
var _headerCompact = false;

// ---- Options (persisted) --------------------------------------------------
var _opts = (function() {
  try { return JSON.parse(localStorage.getItem(LS_OPTIONS) || '{}'); } catch(e) { return {}; }
})();

function saveOpts() {
  try { localStorage.setItem(LS_OPTIONS, JSON.stringify(_opts)); } catch(e) {}
}

function optClickToCopy() { return !!_opts.clickToCopy; }
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
function optHomeLayout()  { return _opts.homeLayout || 'grid'; }

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

function init() {
  _state.homeGroup  = optHomeGroup();
  _state.homeLayout = optHomeLayout();
  loadStateFromURL();
  renderHeader();
  refresh();
}

// ---- URL state -----------------------------------------------------------

function loadStateFromURL() {
  var p = new URLSearchParams(window.location.search);
  _state.view        = p.get('view')    || 'home';
  _state.serverURL   = p.get('server')  || '';
  _state.section     = p.get('section') || 'data';
  _state.path        = decodePath(p.get('path') || '');
  _state.editMode    = p.get('edit') === '1';
  _state.inlines     = csvList(p.get('inline'));
  _state.filters     = (p.get('filter') || '').split('\n').filter(Boolean);
  _state.docView     = p.get('doc')         === '1';
  _state.binary      = p.get('binary')      === '1';
  _state.collections = p.get('collections') === '1';
}

function buildURL(st) {
  var p = new URLSearchParams();
  if (st.view && st.view !== 'home')       p.set('view',    st.view);
  if (st.serverURL)                        p.set('server',  st.serverURL);
  if (st.section && st.section !== 'data') p.set('section', st.section);
  if (st.editMode)                         p.set('edit', '1');
  if (st.path   && st.path.length)         p.set('path',    encodePath(st.path));
  if (st.inlines && st.inlines.length)     p.set('inline',  st.inlines.join(','));
  if (st.filters && st.filters.length)     p.set('filter',  st.filters.join('\n'));
  if (st.docView)                          p.set('doc',         '1');
  if (st.binary)                           p.set('binary',      '1');
  if (st.collections)                      p.set('collections', '1');
  var qs = p.toString();
  return window.location.pathname + (qs ? '?' + qs : '');
}

function pushState(patch) {
  Object.assign(_state, patch);
  history.pushState(null, '', buildURL(_state));
  renderHeader();
  refresh();
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
  if (_state.section === 'model')        return serverBase() + '/model';
  if (_state.section === 'capabilities') return serverBase() + '/capabilities';

  var path = _state.path;
  var url = serverBase() + (path.length ? '/' + path.join('/') : '');

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
  el('section-select').style.display = 'none';
  el('breadcrumbs').style.display    = '';
  el('view-toggle') && (el('view-toggle').style.display = 'none');
  setHeaderCompact(false);
  el('edit-btn').style.display       = isData ? '' : 'none';

  var hvt = el('home-group-toggle');
  var hlt = el('home-layout-btn');
  if (hvt) hvt.style.display = isHome ? '' : 'none';
  if (hlt) hlt.style.display = isHome ? '' : 'none';
  var dvt = el('data-view-toggle');
  if (dvt) dvt.style.display = isData ? '' : 'none';
  if (isData) {
    qsa('[data-dview]').forEach(function(b) {
      b.classList.toggle('active', b.dataset.dview === _state.dataView);
    });
  }

  if (isHome) {
    // inject folder icon SVG into the group toggle button (can't put it in HTML easily)
    var fi = el('hg-folder-icon');
    if (fi && !fi.innerHTML) fi.innerHTML = FOLDER_ICON;
    qsa('[data-hgroup]').forEach(function(b) {
      b.classList.toggle('active', b.dataset.hgroup === _state.homeGroup);
    });
    if (hlt) {
      var hltBtn = hlt.querySelector('.view-btn');
      if (hltBtn) hltBtn.classList.toggle('active', _state.homeLayout === 'list');
    }
  }

  var gb = el('gear-btn');
  if (gb) gb.style.display = isHome ? '' : 'none';

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
  qsa('[data-hgroup]').forEach(function(b) {
    b.classList.toggle('active', b.dataset.hgroup === v);
  });
  renderBreadcrumbs();
  renderHome();
}

function toggleHomeLayout() {
  var v = _state.homeLayout === 'list' ? 'grid' : 'list';
  _state.homeLayout = v;
  _opts.homeLayout  = v;
  saveOpts();
  var hlt = el('home-layout-btn');
  if (hlt) {
    var hltBtn = hlt.querySelector('.view-btn');
    if (hltBtn) hltBtn.classList.toggle('active', v === 'list');
  }
  renderHome();
}

function setDataView(v) {
  _state.dataView = v;
  qsa('[data-dview]').forEach(function(b) {
    b.classList.toggle('active', b.dataset.dview === v);
  });
  if (_lastData) {
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
  if (_state.view === 'home')   return [{label: _state.homeGroup === 'types' ? 'Group Types' : 'Registries', onclick:null, isCurrent:true}];
  if (_state.view === 'config') return [{label:'Config',     onclick:null, isCurrent:true}];

  var segs = [];
  var regLabel = serverLabel(_state.serverURL || window.location.origin);
  var regClick = 'pushState({path:[],section:\'data\',editMode:false});return false';
  segs.push({label: regLabel, onclick: regClick, isCurrent: _state.path.length === 0});

  _state.path.forEach(function(seg, i) {
    var newPath = _state.path.slice(0, i + 1);
    var isLast  = (i === _state.path.length - 1);
    var click   = isLast ? null
      : 'pushState({path:' + esc(JSON.stringify(newPath))
        + ',section:' + esc(JSON.stringify(_state.section)) + ',editMode:false});return false';
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
    var hl = _state.homeLayout;
    items.push({label: 'By Registry',    onclick: "setHomeGroup('registry')", active: hg === 'registry'});
    items.push({label: 'By Group Type',  onclick: "setHomeGroup('types')",    active: hg === 'types'});
    items.push({label: 'List view',      onclick: 'toggleHomeLayout()',       active: hl === 'list'});
    items.push({sep: true});
    items.push({label: 'Config', onclick: 'goToConfig()'});
  }
  if (isData) {
    var dv = _state.dataView || 'grid';
    items.push({label: 'Grid view',  onclick: "setDataView('grid')",  active: dv === 'grid'});
    items.push({label: 'Table view', onclick: "setDataView('table')", active: dv === 'table'});
    items.push({label: 'JSON view',  onclick: "setDataView('json')",  active: dv === 'json'});
    if (_state.mutable) items.push({label: 'Edit', onclick: 'toggleEdit()'});
  }
  return items;
}

function setHeaderCompact(compact) {
  _headerCompact = compact;
  var homeGroupToggle = el('home-group-toggle');
  var homeLayoutBtn   = el('home-layout-btn');
  var dataToggle = el('data-view-toggle');
  var editBtn    = el('edit-btn');
  var gearBtn    = el('gear-btn');
  var compactBtn = el('compact-menu-btn');
  if (!compactBtn) return;
  if (compact) {
    if (homeGroupToggle) homeGroupToggle.style.display = 'none';
    if (homeLayoutBtn)   homeLayoutBtn.style.display   = 'none';
    if (dataToggle) dataToggle.style.display = 'none';
    if (editBtn)    editBtn.style.display    = 'none';
    if (gearBtn)    gearBtn.style.display    = 'none';
    compactBtn.style.display = '';
  } else {
    compactBtn.style.display = 'none';
    // renderHeader() restores correct visibility for other buttons
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

function refresh() {
  var main = el('main-view');
  var leftPanel = el('left-panel');

  if (_state.view === 'home') {
    leftPanel.style.display = 'none';
    renderHome();
    return;
  }

  if (_state.view === 'config') {
    leftPanel.style.display = 'none';
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

  leftPanel.style.display = (_state.view === 'json') ? '' : 'none';
  main.innerHTML = spinner();

  var apiURL = buildAPIURL();
  var coll   = isCollection(_state.path);

  // For resource/version entities we try $details first so that document-backed
  // resources return their JSON metadata rather than their document body.
  // If the server returns 400 (resource has no document), fall back to plain GET.
  fetchWithDetailsFallback(apiURL, needsDetails(_state.path))
    .then(function(data) {
      _lastData = data;
      _state.mutable = detectMutable(data);
      var eb = el('edit-btn');
      if (eb) eb.style.display = (_state.mutable && !_headerCompact) ? '' : 'none';

      switch (_state.view) {
        case 'json': renderJSONView(data); break;
        default:
          if (coll) {
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
  var l = _state.homeLayout;
  if (g === 'types') {
    l === 'list' ? renderHomeFlatList(main, allServers) : renderHomeFlatGrid(main, allServers);
  } else {
    l === 'list' ? renderHomeTable(main, allServers) : renderHomeGrid(main, allServers);
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
    +   '<th>Registry</th><th>Group Types</th><th>Location</th>'
    + '</tr></thead><tbody>';
  sorted.forEach(function(url) {
    html += '<tr data-server-url="' + esc(url) + '">'
      + '<td class="ht-name" style="position:relative">'
      +   '<span class="ht-name-text ht-name-link" onclick="doBrowse(\'' + esc(url) + '\')">' + esc(serverLabel(url)) + '</span>'
      +   '<div class="ht-desc" style="display:none"></div>'
      +   '<div class="server-card-err-popup" style="display:none">'
      +     '<div class="server-card-err-popup-title">Connection Error</div>'
      +     '<div class="server-card-err-popup-msg"></div>'
      +     '<button class="home-btn home-btn-secondary" style="font-size:11px;padding:2px 8px" '
      +       'onclick="this.closest(\'.server-card-err-popup\').style.display=\'none\'">Close</button>'
      +   '</div>'
      + '</td>'
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
        var descEl = row.querySelector('.ht-desc');
        if (descEl && info.description) { descEl.textContent = info.description; descEl.style.display = ''; }
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
    +   '<th>Group Type</th><th>Items</th><th>Resource Types</th><th>Registry</th>'
    + '</tr></thead><tbody id="flat-list-body"><tr><td colspan="4" style="color:#aaa;font-size:13px">Loading…</td></tr></tbody></table></div>';

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
      tbody.innerHTML = '<tr><td colspan="4" style="color:#aaa;font-size:13px;font-style:italic">No group types found</td></tr>';
      return;
    }
    tbody.innerHTML = allRows.map(function(r) {
      var onclick = 'browseGroupCollection(\'' + esc(r.serverUrl) + '\',\'' + esc(r.plural) + '\')';
      var descHtml = r.description ? '<div class="ht-desc">' + esc(r.description) + '</div>' : '';
      return '<tr>'
        + '<td class="ht-name ht-name-link" style="font-weight:bold" onclick="' + onclick + '">' + esc(r.plural) + descHtml + '</td>'
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
  var rootP  = fetchJSON(normUrl + '/');
  var modelP = fetch(normUrl + '/model').then(function(r) { return r.json(); }).catch(function() { return null; });
  Promise.all([rootP, modelP])
    .then(function(results) {
      var data  = results[0];
      var model = results[1];
      if (!data.specversion || !data.registryid) {
        cb({label: '', colls: [], icon: '', error: 'Not a valid xRegistry (missing specversion or registryid)'});
        return;
      }
      var label = data.registryid || '';
      if (label) _labelCache[normUrl] = label;
      var colls = findCollectionRefs(model, [], data);
      colls.forEach(function(c) {
        var grpDef = model && model.groups && model.groups[c.plural];
        c.resources    = grpDef && grpDef.resources ? Object.keys(grpDef.resources).sort() : [];
        c.description  = (grpDef && grpDef.description) || '';
      });
      cb({label: label, colls: colls, icon: data.icon || '', description: data.description || '', error: null});
    })
    .catch(function(err) { cb({label: '', colls: [], icon: '', error: (err && err.message) ? err.message : String(err)}); });
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

function renderConfig() {
  var main   = el('main-view');
  var origin = window.location.origin;
  var servers = loadServers();

  var html = '<div class="config-page"><div class="config-section">'
    + '<h3 class="config-section-title">Registry Servers</h3>'
    + '<table class="config-table"><thead><tr><th>Location</th><th></th></tr></thead><tbody>';

  // Local server — not editable or deletable
  html += '<tr><td>' + esc(origin)
    + ' <span class="config-local-badge">this server</span></td><td></td></tr>';

  // User-added servers
  servers.filter(function(u) { return u !== origin; }).forEach(function(url) {
    html += '<tr data-cfg-url="' + esc(url) + '">'
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
    +   '<input type="checkbox" id="opt-click-to-copy"'
    +   (optClickToCopy() ? ' checked' : '')
    +   ' onchange="cfgSetOpt(\'clickToCopy\',this.checked)">'
    +   '<span class="cfg-option-label">Click to copy</span>'
    +   '<span class="cfg-option-desc">Click any value in the details view to copy it to the clipboard</span>'
    + '</label>'
    + '</div>'

    + '</div>';
  main.innerHTML = html;
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

function renderTableView(data) {
  var main = el('main-view');
  var items = collectionItems(data);

  if (items.length === 0) {
    main.innerHTML = '<div class="state-msg">No items found</div>';
    return;
  }

  if (_sortCol) {
    items = items.slice().sort(function(a, b) {
      var av = String(a[_sortCol] == null ? '' : a[_sortCol]);
      var bv = String(b[_sortCol] == null ? '' : b[_sortCol]);
      return _sortAsc ? av.localeCompare(bv) : bv.localeCompare(av);
    });
  }

  // Build collKeySet from model so deriveColumns can exclude collection structural keys
  var svBase = (_state.serverURL || window.location.origin).replace(/\/$/, '');
  var model  = _modelCache[normalizeURL(svBase)] || null;
  // items are one depth deeper than current path (each item is at path + [id])
  var itemPath = _state.path.concat(['__item__']);
  var colls = items.length > 0 ? findCollectionRefs(model, itemPath, items[0]) : [];
  var collKeySet = {};
  colls.forEach(function(c) {
    collKeySet[c.plural] = true;
    collKeySet[c.plural + 'url'] = true;
    collKeySet[c.plural + 'count'] = true;
  });

  var cols = deriveColumns(items, collKeySet);
  var html = '<div id="table-container"><table class="xr-table"><thead><tr>';
  cols.forEach(function(col) {
    var cls = col === _sortCol ? (_sortAsc ? ' sorted-asc' : ' sorted-desc') : '';
    html += '<th class="' + cls + '" onclick="sortBy(\'' + esc(col) + '\')">'
          + esc(col) + '</th>';
  });
  html += '</tr></thead><tbody>';
  items.forEach(function(item) {
    var navKey = itemNavKey(item);
    html += '<tr onclick="navigateTo(\'' + esc(navKey) + '\')">';
    cols.forEach(function(col) {
      var val = item[col];
      var display = (val == null) ? '' : String(val);
      var cls = col === 'xid' ? ' class="cell-id"' : '';
      html += '<td' + cls + ' title="' + esc(display) + '">' + esc(display) + '</td>';
    });
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
  var model  = _modelCache[normalizeURL(svBase)] || null;
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
    html += '<table class="xr-table" style="margin-bottom:16px">'
      + '<thead><tr><th>Collection</th><th>Count</th></tr></thead><tbody>';
    colls.forEach(function(c) {
      html += '<tr onclick="navigateTo(\'' + esc(c.plural) + '\')" style="cursor:pointer">'
        + '<td class="cell-id">' + esc(c.plural) + '</td>'
        + '<td>' + c.count + '</td>'
        + '</tr>';
    });
    html += '</tbody></table>';
  }

  // Scalar properties
  if (scalarKeys.length) {
    html += '<table class="xr-table"><thead><tr><th>Property</th><th>Value</th></tr></thead><tbody>';
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

// Like getAttrType but returns null when the attr is only matched by the '*'
// wildcard catch-all (i.e., not explicitly named in the model). Used for
// monospace decisions so that extension attributes don't inherit the
// wildcard's type (often ANY) and get incorrectly formatted as monospace.
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
  if (!attrs || !attrs[key]) return null;
  return attrs[key].type || null;
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

function openDocument(singular) {
  var data = _lastData;
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
function renderValueTree(val, depth) {
  var c2c = optClickToCopy();
  function leaf(raw, display) {
    if (!c2c) return '<span>' + display + '</span>';
    return '<span class="eg-copyable" data-copy="' + esc(String(raw)) + '" onclick="egCopy(this.dataset.copy,\'\')">' + display + '</span>';
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
    if (isComplex) {
      return '<div class="vt-kv vt-kv-block" ' + indent + '>'
           + '<span class="vt-key">' + esc(k) + ':</span>'
           + renderValueTree(child, depth + 1)
           + '</div>';
    }
    return '<div class="vt-kv" ' + indent + '>'
         + '<span class="vt-key">' + esc(k) + ':</span> '
         + renderValueTree(child, depth + 1)
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
    box.innerHTML = renderMetaContent(_metaData, _modelCache[svURL] || null);
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
      box.innerHTML = renderMetaContent(d, _modelCache[svURL2] || null);
    })
    .catch(function(e) { box.innerHTML = '<div class="eg-row"><span class="eg-value" style="color:#c00;font-family:monospace">' + esc((e && e.message) ? e.message : String(e)) + '</span></div>'; });
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
              + '<div class="eg-ext-complex-body">' + renderValueTree(v, 0) + '</div>'
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
  var hasDocument = false;
  if ((depth === 4 || depth >= 6) && model && model.groups && model.groups[_state.path[0]]) {
    var _rm = model.groups[_state.path[0]].resources && model.groups[_state.path[0]].resources[_state.path[2]];
    if (_rm && _rm.hasdocument) hasDocument = true;
  }

  // ---- Collections ----
  if (colls.length || depth === 4 || depth === 0 || depth === 2 || (depth >= 6 && hasDocument)) {
    // Resources: no section header (only one collection type), but add meta tile
    if (depth !== 4 && depth < 6) {
      var collsLabel = _state.path.length === 0 ? 'GROUPS' : 'RESOURCES';
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
  }

  // ---- Resource Meta box (depth 4 only): collapsed by default, lazy-fetched on expand ----
  if (depth === 4) {
    _metaData = null;
    _metaResourceIdField = idFieldName;  // suppress resource's own ID in meta content
    html += '<div class="eg-section-header eg-details-header">'
          + '<span>' + esc(entityType) + ' Details'
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
    detailsLabel = 'Default ' + capType + ' Version'
      + (data.versionid !== undefined ? ' (' + esc(String(data.versionid)) + ')' : '')
      + ' Details';
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
              + '<div class="eg-ext-complex-body">' + renderValueTree(v, 0) + '</div>'
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
  renderJSONLeftPanel(data);
  el('main-view').innerHTML =
    '<div id="json-output">' + syntaxHighlight(JSON.stringify(data, null, 2)) + '</div>';
}

function renderJSONLeftPanel(data) {
  var inner = el('left-panel-inner');
  if (!inner) return;
  var html = '';

  if (_state.section === 'data') {
    html += '<div class="lp-section"><div class="lp-title">Filters '
      + '<span style="font-weight:normal;font-size:11px;color:#888">(one per line)</span></div>'
      + '<textarea class="lp-filter-area" id="lp-filters">'
      + esc(_state.filters.join('\n')) + '</textarea></div><hr class="lp-divider">';
  }

  html += '<div class="lp-section"><div class="lp-title">Options</div>'
    + lpCheck('lp-doc', 'doc view',    _state.docView)
    + lpCheck('lp-bin', 'binary',      _state.binary)
    + lpCheck('lp-col', 'collections', _state.collections)
    + '</div><hr class="lp-divider">';

  var svBase2 = (_state.serverURL || window.location.origin).replace(/\/$/, '');
  var model2  = _modelCache[normalizeURL(svBase2)] || null;
  var inlineOpts = inlineOptions(model2, _state.path, data);
  if (inlineOpts.length) {
    html += '<div class="lp-section"><div class="lp-title">Inlines</div>';
    inlineOpts.forEach(function(opt, i) {
      html += '<div class="lp-item' + (i%2===0 ? ' lp-even':'') + '">'
        + '<input type="checkbox" class="lp-inline-cb" value="' + esc(opt) + '"'
        + (_state.inlines.includes(opt) ? ' checked' : '') + '> ' + esc(opt) + '</div>';
    });
    html += '</div><hr class="lp-divider">';
  }

  html += '<button class="lp-apply" onclick="applyJSONOptions()">Apply</button>';
  inner.innerHTML = html;
}

function lpCheck(id, label, checked) {
  return '<div class="lp-item"><input type="checkbox" id="' + id + '"'
    + (checked ? ' checked':'') + '> ' + label + '</div>';
}

function applyJSONOptions() {
  var fa  = el('lp-filters'), doc = el('lp-doc'),
      bin = el('lp-bin'),    col = el('lp-col');
  var cbs = qsa('.lp-inline-cb');
  var inlines = [];
  cbs.forEach(function(cb) { if (cb.checked) inlines.push(cb.value); });
  pushState({
    filters:     fa  ? fa.value.split('\n').filter(Boolean) : [],
    docView:     doc ? doc.checked : false,
    binary:      bin ? bin.checked : false,
    collections: col ? col.checked : false,
    inlines:     inlines
  });
}

// Derive inline options from the model and current response.
// Model-defined collection plurals are offered as inline options.
// Excludes metadata scalars and collection structural keys.
function inlineOptions(model, path, data) {
  if (!data || typeof data !== 'object') return [];
  var skip = new Set(['epoch','createdat','modifiedat','labels']);
  var colls = findCollectionRefs(model, path, data);
  var collKeySet = {};
  colls.forEach(function(c) {
    collKeySet[c.plural] = true;
    collKeySet[c.plural + 'url'] = true;
    collKeySet[c.plural + 'count'] = true;
  });
  var opts = [];
  Object.keys(data).forEach(function(k) {
    if (!skip.has(k) && !collKeySet[k] && typeof data[k] === 'object' && data[k] !== null) {
      opts.push(k);
    }
  });
  colls.forEach(function(c) {
    if (!opts.includes(c.plural)) opts.push(c.plural);
  });
  return opts;
}

function syntaxHighlight(str) {
  return str
    .replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
    .replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
      function(m) {
        var c = /^"/.test(m) ? (/:$/.test(m) ? 'json-key' : 'json-str')
              : /true|false/.test(m) ? 'json-bool' : /null/.test(m) ? 'json-null' : 'json-num';
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
