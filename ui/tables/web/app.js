if (window.EraOfficeShell) {
  if (EraOfficeShell.markActiveNav) EraOfficeShell.markActiveNav('tables');
  if (EraOfficeShell.mountNav) EraOfficeShell.mountNav(document);
  else if (EraOfficeShell.mountIcons) EraOfficeShell.mountIcons(document);
  if (EraOfficeShell.syncUserChip) EraOfficeShell.syncUserChip();
  if (EraOfficeShell.wireTeDisclaimer) {
    EraOfficeShell.wireTeDisclaimer(document.getElementById('teBanner'), 'era_te_dismiss_tables');
  }
}

const grid = document.getElementById('grid');
const authStatus = document.getElementById('authStatus');
const sheetMeta = document.getElementById('sheetMeta');
const formulaInput = document.getElementById('formulaInput');
const activeAddrEl = document.getElementById('activeAddr');
const presenceYou = document.getElementById('presenceYou');
const presencePeers = document.getElementById('presencePeers');
const sheetTabsEl = document.getElementById('sheetTabs');
const filterInput = document.getElementById('filterInput');

/* Virtual window: only ~visible cells in DOM; capacity is logical A–WW × 10K. */
const DEFAULT_VIEW_ROWS = 40;
const DEFAULT_VIEW_COLS = 26; // A–Z first paint
const MAX_VIEW_ROWS = 10000;
const MAX_VIEW_COLS = 621; // A–WW inclusive
const CAPACITY_ROWS = 10000;
const CAPACITY_COLS = 621;
/** Nominal cell size for capacity scrollbar (matches CSS min-width ~5.5rem). */
const SCROLL_COL_W = 88;
const SCROLL_ROW_H = 28;
const SCROLL_CORNER_W = 36;
const WIN_ROW_PAD = 8;
const WIN_COL_PAD = 4;

/** @deprecated alias — window size (not materialized extent). */
let viewRows = DEFAULT_VIEW_ROWS;
let viewCols = DEFAULT_VIEW_COLS;
/** Virtual window origin (row 1-based, col 0-based). */
let startRow = 1;
let startCol = 0;
let winRows = DEFAULT_VIEW_ROWS;
let winCols = DEFAULT_VIEW_COLS;
let scrollSyncing = false;
let virtScrollRaf = 0;

let sheetId = null;
/** @type {{ cells: Record<string, { value?: string, formula?: string }>, rows?: number, cols?: number, sheets?: { name: string }[], active_sheet?: number }} */
let sheetState = { cells: {} };
let sheetTabNames = ['Sheet1'];
let activeSheetIndex = 0;
let ws = null;
let activeAddr = 'A1';
let editing = false;
let freezeOn = false;
/** ERA+ freeze panes: first N data rows / columns (persisted per SheetTab). */
let freezeRows = 0;
let freezeCols = 0;
let filterText = '';
/** @type {{ col: number, mode: string, value: string } | null} */
let filterOpts = null;
let sheetProtected = false;
/** @type {{ c1: number, r1: number, c2: number, r2: number, key: string }[]} */
let cellMerges = [];
/** @type {string[]} A1:B2 strings */
let protectedRanges = [];
let colWidths = [];
let presenceHeartbeatTimer = null;
/** @type {Record<string, { addr: string, color: string }>} */
let peerCells = {};
/** Selection anchor (click / shift+click / drag). End = activeAddr. */
let selAnchorAddr = null;
let isSelectingRange = false;

function identity() {
  const token = localStorage.getItem('era_token') || '';
  try {
    const p = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
    return { tenantId: p.tenant_id || 't-demo', userId: p.sub || 'u-alice' };
  } catch (_) {
    return { tenantId: 't-demo', userId: 'u-alice' };
  }
}

function authHeaders(extra) {
  const h = Object.assign({}, extra || {});
  const token = localStorage.getItem('era_token') || '';
  if (token) h.Authorization = 'Bearer ' + token;
  const { userId, tenantId } = identity();
  h['X-ERA-User'] = userId;
  h['X-ERA-Tenant'] = tenantId;
  return h;
}

function officeFetch(url, opts) {
  opts = opts || {};
  const next = Object.assign({}, opts, { headers: authHeaders(opts.headers) });
  if (window.EraOfficeShell && EraOfficeShell.authFetch) {
    return EraOfficeShell.authFetch(url, next);
  }
  return fetch(url, next).then((res) => {
    if (window.EraOfficeShell && EraOfficeShell.handleUnauthorized) {
      EraOfficeShell.handleUnauthorized(res);
    }
    return res;
  });
}

function setAuthStatus(msg, isErr) {
  if (window.EraOfficeShell && EraOfficeShell.toastStatus) {
    EraOfficeShell.toastStatus(authStatus, msg, !!isErr);
    return;
  }
  if (!authStatus) return;
  authStatus.textContent = msg || '';
  authStatus.className = 'era-status ' + (isErr ? 'err' : 'ok');
}

function refreshPresenceYou() {
  const { userId } = identity();
  if (presenceYou) presenceYou.textContent = userId || 'You';
  if (presencePeers) presencePeers.textContent = 'Peers: —';
  // Never set userChip.textContent — it wipes the green status lamp.
  if (window.EraOfficeShell && EraOfficeShell.syncUserChip) EraOfficeShell.syncUserChip();
}

function peerColor(uid) {
  const colors = ['#c45', '#2a7', '#36c', '#a6a', '#e80'];
  let h = 0;
  const s = String(uid || '');
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0;
  return colors[Math.abs(h) % colors.length];
}

function updatePresencePeers(peers) {
  const { userId } = identity();
  const others = (peers || []).filter((p) => p && p !== userId);
  if (!presencePeers) return;
  if (!others.length) {
    presencePeers.textContent = 'Peers: —';
    return;
  }
  presencePeers.innerHTML = 'Peers: ';
  others.forEach((uid, i) => {
    if (i) presencePeers.appendChild(document.createTextNode(', '));
    const chip = document.createElement('span');
    chip.style.cssText =
      'display:inline-block;padding:0 5px;border-radius:3px;font-size:0.85em;background:' +
      peerColor(uid) +
      ';color:#fff';
    chip.textContent = uid.length > 10 ? uid.slice(0, 8) + '…' : uid;
    presencePeers.appendChild(chip);
  });
}

function broadcastPresenceCell(addr) {
  if (!ws || ws.readyState !== WebSocket.OPEN || !addr) return;
  const { userId } = identity();
  ws.send(
    JSON.stringify({
      type: 'presence_cell',
      from: userId,
      user_id: userId,
      addr,
      color: peerColor(userId),
    })
  );
}

function paintPeerCells() {
  grid.querySelectorAll('td.peer-cell').forEach((td) => {
    td.classList.remove('peer-cell');
    td.style.outline = '';
  });
  Object.keys(peerCells).forEach((uid) => {
    const info = peerCells[uid];
    if (!info || !info.addr) return;
    const td = grid.querySelector('td[data-addr="' + info.addr + '"]');
    if (!td) return;
    td.classList.add('peer-cell');
    td.style.outline = '2px solid ' + (info.color || '#c45');
  });
}

function startPresenceHeartbeat() {
  clearInterval(presenceHeartbeatTimer);
  presenceHeartbeatTimer = setInterval(() => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'presence_heartbeat' }));
    }
  }, 20000);
}

function stopPresenceHeartbeat() {
  clearInterval(presenceHeartbeatTimer);
  presenceHeartbeatTimer = null;
}

function pathSheetId() {
  const parts = location.pathname.replace(/\/$/, '').split('/');
  const id = parts[parts.length - 1];
  return id && id !== 'tables' ? id : null;
}

function colLetter(c) {
  let n = c;
  let s = '';
  do {
    s = String.fromCharCode(65 + (n % 26)) + s;
    n = Math.floor(n / 26) - 1;
  } while (n >= 0);
  return s;
}

function colIndex(letters) {
  let n = 0;
  const s = String(letters || '').toUpperCase();
  for (let i = 0; i < s.length; i++) {
    const code = s.charCodeAt(i);
    if (code < 65 || code > 90) return -1;
    n = n * 26 + (code - 64);
  }
  return n - 1;
}

function parseAddr(addr) {
  const m = /^([A-Z]+)(\d+)$/i.exec(addr || '');
  if (!m) return null;
  const col = colIndex(m[1]);
  if (col < 0) return null;
  return { col, row: parseInt(m[2], 10) };
}

function makeAddr(col, row) {
  return colLetter(col) + row;
}

function colWidthAt(c) {
  return colWidths[c] || SCROLL_COL_W;
}

function rowHeaderWidthPx() {
  const maxShown = Math.max(99, startRow + winRows - 1, CAPACITY_ROWS);
  const digits = String(maxShown).length;
  return Math.max(36, 12 + digits * 9);
}

function gridOffsetLeft() {
  let left = 0;
  for (let c = 0; c < startCol; c++) left += colWidthAt(c);
  return left;
}

function gridOffsetTop() {
  return (startRow - 1) * SCROLL_ROW_H;
}

function totalGridWidth() {
  let w = rowHeaderWidthPx();
  for (let c = 0; c < CAPACITY_COLS; c++) w += colWidthAt(c);
  return w;
}

function totalGridHeight() {
  return SCROLL_ROW_H + CAPACITY_ROWS * SCROLL_ROW_H;
}

function syncGridPhantomSize() {
  const phantom = document.getElementById('gridPhantom');
  if (!phantom) return;
  const w = totalGridWidth() + 'px';
  const h = totalGridHeight() + 'px';
  phantom.style.width = w;
  phantom.style.height = h;
  phantom.style.minWidth = w;
  phantom.style.minHeight = h;
  const head = document.getElementById('colHeadSticky');
  if (head) {
    head.style.width = w;
    head.style.minWidth = w;
  }
}

function updateWindowFromScroll() {
  const wrap = document.getElementById('gridWrap');
  if (!wrap) return;
  const st = Math.max(0, wrap.scrollTop);
  const sl = Math.max(0, wrap.scrollLeft);
  startRow = Math.min(CAPACITY_ROWS, Math.max(1, Math.floor(st / SCROLL_ROW_H) + 1));
  let acc = 0;
  startCol = 0;
  for (let c = 0; c < CAPACITY_COLS; c++) {
    const w = colWidthAt(c);
    if (acc + w > sl) {
      startCol = c;
      break;
    }
    acc += w;
    startCol = c;
  }
  winRows = Math.min(
    CAPACITY_ROWS - startRow + 1,
    Math.max(DEFAULT_VIEW_ROWS, Math.ceil(wrap.clientHeight / SCROLL_ROW_H) + WIN_ROW_PAD)
  );
  winCols = Math.min(
    CAPACITY_COLS - startCol,
    Math.max(DEFAULT_VIEW_COLS, Math.ceil(wrap.clientWidth / SCROLL_COL_W) + WIN_COL_PAD)
  );
  viewRows = winRows;
  viewCols = winCols;
}

function growViewportFromSheet(_sheet) {
  // Virtual grid: DOM size is independent of used range.
  updateWindowFromScroll();
}

function ensureAddrVisible(addr) {
  const p = parseAddr(addr);
  if (!p) return;
  const wrap = document.getElementById('gridWrap');
  if (!wrap) return;
  const rh = SCROLL_ROW_H;
  const targetTop = (p.row - 1) * rh;
  let targetLeft = 0;
  for (let c = 0; c < p.col; c++) targetLeft += colWidthAt(c);
  const viewH = Math.max(rh * 2, wrap.clientHeight - rh);
  const viewW = Math.max(SCROLL_COL_W * 2, wrap.clientWidth - rowHeaderWidthPx());
  let nextTop = wrap.scrollTop;
  let nextLeft = wrap.scrollLeft;
  if (targetTop < wrap.scrollTop) nextTop = targetTop;
  else if (targetTop + rh > wrap.scrollTop + viewH) nextTop = Math.max(0, targetTop - viewH + rh * 2);
  if (targetLeft < wrap.scrollLeft) nextLeft = targetLeft;
  else if (targetLeft + colWidthAt(p.col) > wrap.scrollLeft + viewW) {
    nextLeft = Math.max(0, targetLeft - viewW + colWidthAt(p.col) * 2);
  }
  scrollSyncing = true;
  wrap.scrollTop = nextTop;
  wrap.scrollLeft = nextLeft;
  updateWindowFromScroll();
  renderGrid();
  scrollSyncing = false;
}

function parseRangeStr(s) {
  const parts = String(s || '')
    .toUpperCase()
    .trim()
    .split(':');
  const a = parseAddr(parts[0]);
  const b = parseAddr(parts[1] || parts[0]);
  if (!a || !b) return null;
  return {
    c1: Math.min(a.col, b.col),
    r1: Math.min(a.row, b.row),
    c2: Math.max(a.col, b.col),
    r2: Math.max(a.row, b.row),
    key:
      makeAddr(Math.min(a.col, b.col), Math.min(a.row, b.row)) +
      ':' +
      makeAddr(Math.max(a.col, b.col), Math.max(a.row, b.row)),
  };
}

function loadMergesFromList(list) {
  cellMerges = (list || [])
    .map((s) => parseRangeStr(s))
    .filter(Boolean);
}

function loadProtectedRangesFromList(list) {
  protectedRanges = (list || [])
    .map((s) => {
      const m = parseRangeStr(s);
      return m ? m.key : '';
    })
    .filter(Boolean);
}

function rangesOverlap(a, b) {
  return !(a.c2 < b.c1 || a.c1 > b.c2 || a.r2 < b.r1 || a.r1 > b.r2);
}

function isCoveredByMerge(c, r) {
  for (const m of cellMerges) {
    if (c === m.c1 && r === m.r1) continue;
    if (c >= m.c1 && c <= m.c2 && r >= m.r1 && r <= m.r2) return true;
  }
  return false;
}

function mergeAt(c, r) {
  return cellMerges.find((m) => m.c1 === c && m.r1 === r) || null;
}

function isAddrRangeProtected(addr) {
  const p = parseAddr(addr);
  if (!p) return false;
  return protectedRanges.some((key) => {
    const m = parseRangeStr(key);
    if (!m) return false;
    return p.col >= m.c1 && p.col <= m.c2 && p.row >= m.r1 && p.row <= m.r2;
  });
}

function isAddrProtected(addr) {
  return sheetProtected || isAddrRangeProtected(addr);
}

function syncSheetTabsFromState(sheet) {
  if (sheet.sheets && sheet.sheets.length) {
    sheetTabNames = sheet.sheets.map((t) => t.name || 'Sheet');
    const tab = sheet.sheets[sheet.active_sheet != null ? sheet.active_sheet : activeSheetIndex];
    sheetProtected = !!(tab && tab.protected);
    if (tab) {
      freezeRows = tab.freeze_rows || 0;
      freezeCols = tab.freeze_cols || 0;
      loadMergesFromList(tab.merges);
      loadProtectedRangesFromList(tab.protected_ranges);
      filterOpts = tab.filter_criteria || null;
      if (Array.isArray(tab.charts) && tab.charts.length) restoreCharts(tab.charts);
      loadScenariosFromTab(tab);
    }
  } else if (sheet.sheet_names && sheet.sheet_names.length) {
    sheetTabNames = sheet.sheet_names.slice();
  }
  if (typeof sheet.active_sheet === 'number') {
    activeSheetIndex = sheet.active_sheet;
  }
  if (typeof sheet.protected === 'boolean') sheetProtected = sheet.protected;
  if (typeof sheet.freeze_rows === 'number') freezeRows = sheet.freeze_rows;
  if (typeof sheet.freeze_cols === 'number') freezeCols = sheet.freeze_cols;
  if (Array.isArray(sheet.merges)) loadMergesFromList(sheet.merges);
  if (Array.isArray(sheet.protected_ranges)) loadProtectedRangesFromList(sheet.protected_ranges);
  updateProtectUi();
}

function updateProtectUi() {
  const input = formulaInput;
  if (input) input.readOnly = sheetProtected;
  grid.classList.toggle('sheet-protected', sheetProtected);
  const meta = document.getElementById('sheetMeta');
  if (meta && sheetId) {
    const base = meta.textContent.replace(/\s*·\s*Protected\s*$/, '');
    meta.textContent = sheetProtected ? base + ' · Protected' : base;
  }
}

function renderSheetTabs() {
  if (!sheetTabsEl) return;
  sheetTabsEl.innerHTML = '';
  sheetTabNames.forEach((name, i) => {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'tab' + (i === activeSheetIndex ? ' active' : '');
    btn.textContent = name;
    btn.title = 'Switch · double-click to rename';
    btn.draggable = true;
    btn.addEventListener('click', () => {
      if (i === activeSheetIndex) return;
      sendSheetOp({ type: 'switch_sheet', index: i });
    });
    btn.addEventListener('dblclick', (ev) => {
      ev.preventDefault();
      if (i !== activeSheetIndex) sendSheetOp({ type: 'switch_sheet', index: i });
      setTimeout(() => renameActiveSheetTab(), 50);
    });
    btn.addEventListener('dragstart', (ev) => {
      ev.dataTransfer.setData('text/plain', String(i));
    });
    btn.addEventListener('dragover', (ev) => ev.preventDefault());
    btn.addEventListener('drop', (ev) => {
      ev.preventDefault();
      const from = parseInt(ev.dataTransfer.getData('text/plain'), 10);
      if (isNaN(from) || from === i) return;
      sendSheetOp({ type: 'reorder_sheet', from, to: i });
      const [tab] = sheetTabNames.splice(from, 1);
      sheetTabNames.splice(i, 0, tab);
      if (activeSheetIndex === from) activeSheetIndex = i;
      renderSheetTabs();
      scheduleRefresh();
    });
    sheetTabsEl.appendChild(btn);
  });
}

function cellOf(addr) {
  const cells = sheetState.cells || {};
  return cells[addr] || cells[addr.toUpperCase()] || {};
}

function displayFor(addr) {
  const c = cellOf(addr);
  const raw = c.value != null ? String(c.value) : '';
  const fmt = c.format || '';
  if (!fmt || !raw) return raw;
  const n = parseFloat(raw);
  if (fmt === '0.00' && !isNaN(n)) return n.toFixed(2);
  if (fmt === '%' && !isNaN(n)) return Math.round(n * 100) + '%';
  if (fmt === 'date') return raw;
  return raw;
}

function formulaFor(addr) {
  const c = cellOf(addr);
  return c.formula || '';
}

function applyCells(cells) {
  if (!cells) return;
  sheetState.cells = cells;
  paintGridValues();
  applyRowFilter();
}

function applySyncEnvelope(parsed) {
  if (parsed.cells) applyCells(parsed.cells);
  if (typeof parsed.active_sheet === 'number') activeSheetIndex = parsed.active_sheet;
  if (parsed.sheet_names && parsed.sheet_names.length) sheetTabNames = parsed.sheet_names.slice();
  if (typeof parsed.protected === 'boolean') sheetProtected = parsed.protected;
  if (parsed.op && parsed.op.type === 'protect_sheet') sheetProtected = !!parsed.op.protected;
  if (typeof parsed.freeze_rows === 'number') freezeRows = parsed.freeze_rows;
  if (typeof parsed.freeze_cols === 'number') freezeCols = parsed.freeze_cols;
  if (parsed.op && parsed.op.type === 'freeze_panes') {
    freezeRows = parsed.op.rows || 0;
    freezeCols = parsed.op.cols || 0;
  }
  let needGrid = false;
  if (parsed.op && parsed.op.type === 'set_merges') {
    loadMergesFromList(parsed.op.merges);
    needGrid = true;
  }
  if (parsed.op && parsed.op.type === 'set_protected_ranges') {
    loadProtectedRangesFromList(parsed.op.ranges);
    needGrid = true;
  }
  if (parsed.op && parsed.op.type === 'sort_range') {
    needGrid = false;
    scheduleRefresh();
  }
  if (parsed.op && parsed.op.type === 'set_filter_criteria') {
    filterOpts = parsed.op.criteria || null;
    applyRowFilter();
  }
  if (parsed.op && parsed.op.type === 'set_charts') {
    restoreCharts(parsed.op.charts || []);
  }
  if (parsed.op && parsed.op.type === 'set_scenarios') {
    loadScenariosFromTab({ scenarios: parsed.op.scenarios });
    if (sheetState.sheets && sheetState.sheets[activeSheetIndex]) {
      sheetState.sheets[activeSheetIndex].scenarios = parsed.op.scenarios;
    }
  }
  updateProtectUi();
  renderSheetTabs();
  applyFreezePanesStyles();
  if (needGrid) renderGrid();
  else paintGridValues();
}

function applyCellChrome(td, addr) {
  const c = cellOf(addr);
  td.style.fontWeight = c.bold ? '700' : '';
  td.style.fontFamily = c.font || '';
  td.style.fontSize = c.font_size ? c.font_size + 'pt' : '';
  td.style.textAlign = c.align || '';
  td.style.whiteSpace = c.wrap ? 'pre-wrap' : '';
  const sides = (c.border_sides || (c.border ? 'tblr' : '')).toLowerCase();
  if (sides) {
    td.style.borderTop = sides.includes('t') ? '2px solid var(--era-ink)' : '';
    td.style.borderRight = sides.includes('r') ? '2px solid var(--era-ink)' : '';
    td.style.borderBottom = sides.includes('b') ? '2px solid var(--era-ink)' : '';
    td.style.borderLeft = sides.includes('l') ? '2px solid var(--era-ink)' : '';
  } else {
    td.style.borderTop = td.style.borderRight = td.style.borderBottom = td.style.borderLeft = '';
  }
}

function paintCellTd(td, addr) {
  if (!td) return;
  if (editing && addr === activeAddr && document.activeElement === td) return;
  const f = formulaFor(addr);
  const c = cellOf(addr);
  td.textContent = displayFor(addr);
  td.classList.toggle('formula', !!f);
  td.classList.toggle('range-protected', isAddrRangeProtected(addr));
  const tip = c.note
    ? 'Note: ' + c.note
    : f || (isAddrRangeProtected(addr) ? 'Protected range' : '');
  td.title = tip;
  applyCellChrome(td, addr);
}

function renderCellNotesRail() {
  const ul = document.getElementById('commentsList');
  if (!ul) return;
  ul.innerHTML = '';
  const cells = sheetState.cells || {};
  const notes = Object.keys(cells)
    .filter((a) => cells[a] && cells[a].note)
    .sort();
  if (!notes.length) {
    ul.innerHTML = '<li class="era-hint">No cell notes yet</li>';
    return;
  }
  for (const addr of notes) {
    const li = document.createElement('li');
    li.style.borderBottom = '1px solid var(--era-line)';
    li.style.padding = '0.35rem 0';
    li.innerHTML =
      '<div><strong>' +
      addr +
      '</strong></div><div>' +
      String(cells[addr].note || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;') +
      '</div>';
    li.addEventListener('click', () => {
      activeAddr = addr;
      ensureAddrVisible(addr);
      highlightActive();
      syncFormulaBar();
      const td = grid.querySelector('td[data-addr="' + addr + '"]');
      if (td) td.focus();
    });
    ul.appendChild(li);
  }
}

function paintGridValues() {
  grid.querySelectorAll('td[data-addr]').forEach((td) => {
    paintCellTd(td, td.dataset.addr);
  });
  syncFormulaBar();
  highlightActive();
  renderCellNotesRail();
}

function setCellStylePatch(patch) {
  if (!sheetId || !activeAddr) {
    setAuthStatus('Select a cell first', true);
    return;
  }
  if (!sheetState.cells) sheetState.cells = {};
  const key = activeAddr.toUpperCase();
  const c = sheetState.cells[key] || sheetState.cells[activeAddr] || {};
  sheetState.cells[key] = c;
  if (patch.bold != null) c.bold = patch.bold;
  if (patch.font != null) c.font = patch.font;
  if (patch.font_size != null) c.font_size = patch.font_size;
  if (patch.align != null) c.align = patch.align;
  if (patch.wrap != null) c.wrap = patch.wrap;
  if (patch.border != null) c.border = patch.border;
  if (patch.border_sides != null) c.border_sides = patch.border_sides;
  if (patch.note != null) c.note = patch.note;
  sendSheetOp({
    type: 'set_cell_style',
    addr: activeAddr,
    bold: patch.bold,
    align: patch.align,
    wrap: patch.wrap,
    border: patch.border,
    border_sides: patch.border_sides,
    note: patch.note,
  });
  paintGridValues();
  renderCellNotesRail();
  setAuthStatus(patch.note != null ? 'Note saved on ' + activeAddr : 'Cell style updated', false);
}

function pasteValuesActive() {
  if (!sheetId || !activeAddr) {
    setAuthStatus('Select a cell first', true);
    return;
  }
  const c = cellOf(activeAddr);
  const val = c.value != null ? String(c.value) : '';
  c.formula = '';
  c.value = val;
  sendCell(activeAddr, val, '');
  paintGridValues();
  setAuthStatus('Pasted values (cleared formula on active cell)', false);
}

function currentSelectionRange() {
  const a = parseAddr(selAnchorAddr || activeAddr);
  const b = parseAddr(activeAddr);
  if (!a || !b) return null;
  return {
    c1: Math.min(a.col, b.col),
    r1: Math.min(a.row, b.row),
    c2: Math.max(a.col, b.col),
    r2: Math.max(a.row, b.row),
    key:
      makeAddr(Math.min(a.col, b.col), Math.min(a.row, b.row)) +
      ':' +
      makeAddr(Math.max(a.col, b.col), Math.max(a.row, b.row)),
  };
}

function selectionLabel() {
  const r = currentSelectionRange();
  if (!r) return activeAddr || '';
  if (r.c1 === r.c2 && r.r1 === r.r2) return makeAddr(r.c1, r.r1);
  return r.key;
}

function syncFormulaBar() {
  if (activeAddrEl) {
    const label = selectionLabel();
    if (activeAddrEl.tagName === 'INPUT') {
      if (document.activeElement !== activeAddrEl) activeAddrEl.value = label || '';
    } else {
      activeAddrEl.textContent = label || '—';
    }
  }
  formulaInput.disabled = !sheetId;
  if (editing && document.activeElement === formulaInput) return;
  const f = formulaFor(activeAddr);
  formulaInput.value = f || displayFor(activeAddr);
}

function highlightActive() {
  paintSelection();
}

function paintSelection() {
  const range = currentSelectionRange();
  grid.querySelectorAll('td[data-addr]').forEach((td) => {
    const p = parseAddr(td.dataset.addr);
    let inSel = false;
    if (range && p) {
      inSel = p.col >= range.c1 && p.col <= range.c2 && p.row >= range.r1 && p.row <= range.r2;
    }
    td.classList.toggle('in-sel', inSel && !(td.dataset.addr === activeAddr));
    td.classList.toggle('selected', td.dataset.addr === activeAddr);
  });
  broadcastPresenceCell(activeAddr);
  paintPeerCells();
}

function matchFilterClause(rowNum, clause) {
  if (!clause) return true;
  const addr = makeAddr(clause.col || 0, rowNum);
  const text = (displayFor(addr) || '').trim();
  const mode = clause.mode || 'contains';
  const val = (clause.value || '').trim();
  if (mode === 'empty') return text === '';
  if (mode === 'notempty') return text !== '';
  if (mode === 'equals') return text.toLowerCase() === val.toLowerCase();
  return text.toLowerCase().includes(val.toLowerCase());
}

function rowMatchesFilter(rowNum) {
  if (filterOpts) {
    if (!matchFilterClause(rowNum, filterOpts)) return false;
    if (filterOpts.and) return matchFilterClause(rowNum, filterOpts.and);
    return true;
  }
  const q = filterText.trim().toLowerCase();
  if (!q) return true;
  const cells = sheetState.cells || {};
  for (const addr of Object.keys(cells)) {
    const p = parseAddr(addr);
    if (!p || p.row !== rowNum) continue;
    const text = (displayFor(addr) + ' ' + formulaFor(addr)).toLowerCase();
    if (text.includes(q)) return true;
  }
  return false;
}

function applyRowFilter() {
  const table = grid.querySelector('table.sheet');
  if (!table) return;
  table.querySelectorAll('tbody tr').forEach((tr) => {
    const rowNum = parseInt(tr.dataset.row || '0', 10);
    tr.classList.toggle('filter-hidden', !rowMatchesFilter(rowNum));
  });
}

function setupColumnResize(th, absCol) {
  if (!th || th.querySelector('.col-resize')) return;
  const handle = document.createElement('span');
  handle.className = 'col-resize';
  handle.title = 'Drag to resize column';
  handle.addEventListener('mousedown', (ev) => {
    ev.preventDefault();
    ev.stopPropagation();
    const startX = ev.clientX;
    const table = grid.querySelector('table.sheet');
    const visIdx = absCol - startCol;
    const colEl =
      table && table.querySelector('colgroup col:nth-child(' + (visIdx + 2) + ')');
    const startW = colWidthAt(absCol);
    function onMove(e) {
      const w = Math.max(48, startW + (e.clientX - startX));
      colWidths[absCol] = w;
      th.style.width = w + 'px';
      th.style.minWidth = w + 'px';
      th.style.maxWidth = w + 'px';
      if (colEl) colEl.style.width = w + 'px';
      const bodyTh = table && table.querySelector('th.col-hd[data-col="' + absCol + '"]');
      if (bodyTh) {
        bodyTh.style.width = w + 'px';
        bodyTh.style.minWidth = w + 'px';
        bodyTh.style.maxWidth = w + 'px';
      }
    }
    function onUp() {
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
      syncGridPhantomSize();
      renderColHeader();
    }
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  });
  th.appendChild(handle);
}

function beginCellEdit(td, seedText) {
  if (!td) return;
  const addr = td.dataset.addr;
  activeAddr = addr;
  editing = true;
  td.contentEditable = 'true';
  highlightActive();
  if (seedText != null) {
    td.textContent = seedText;
  } else {
    const f = formulaFor(addr);
    td.textContent = f || displayFor(addr);
  }
  td.focus({ preventScroll: true });
  try {
    const range = document.createRange();
    range.selectNodeContents(td);
    range.collapse(false);
    const sel = window.getSelection();
    if (sel) {
      sel.removeAllRanges();
      sel.addRange(range);
    }
  } catch (_) {}
  syncFormulaBar();
}

function endCellEditIfNeeded() {
  if (!editing) return;
  const prev =
    document.activeElement &&
    document.activeElement.closest &&
    document.activeElement.closest('td[data-addr]');
  if (prev && grid.contains(prev)) {
    editing = false;
    commitCell(prev.dataset.addr, prev.textContent || '');
    prev.contentEditable = 'false';
    paintCellTd(prev, prev.dataset.addr);
  } else {
    editing = false;
  }
}

function selectCellOnly(td, opts) {
  if (!td) return;
  const extend = !!(opts && opts.extend);
  const addr = td.dataset.addr;
  endCellEditIfNeeded();
  if (!extend) selAnchorAddr = addr;
  activeAddr = addr;
  editing = false;
  td.contentEditable = 'false';
  td.tabIndex = 0;
  paintSelection();
  syncFormulaBar();
  if (document.activeElement !== td) td.focus({ preventScroll: true });
}

function tdFromPoint(x, y) {
  const el = document.elementFromPoint(x, y);
  if (!el) return null;
  const td = el.closest && el.closest('td[data-addr]');
  if (!td || !grid.contains(td)) return null;
  return td;
}

function extendSelectionTo(td) {
  if (!td) return;
  endCellEditIfNeeded();
  if (!selAnchorAddr) selAnchorAddr = activeAddr || td.dataset.addr;
  activeAddr = td.dataset.addr;
  editing = false;
  td.contentEditable = 'false';
  paintSelection();
  syncFormulaBar();
}

let gridDelegated = false;
function ensureGridDelegation() {
  if (gridDelegated) return;
  gridDelegated = true;
  grid.addEventListener('focusin', (ev) => {
    const td = ev.target.closest('td[data-addr]');
    if (!td || !grid.contains(td)) return;
    if (!editing) {
      activeAddr = td.dataset.addr;
      td.contentEditable = 'false';
      paintSelection();
      syncFormulaBar();
    }
  });
  grid.addEventListener('focusout', (ev) => {
    const td = ev.target.closest('td[data-addr]');
    if (!td || !grid.contains(td)) return;
    const next = ev.relatedTarget;
    if (next && td.contains(next)) return;
    if (!editing) {
      td.contentEditable = 'false';
      return;
    }
    editing = false;
    const addr = td.dataset.addr;
    commitCell(addr, td.textContent || '');
    td.contentEditable = 'false';
    paintCellTd(td, addr);
  });
  grid.addEventListener('keydown', (ev) => {
    const td = ev.target.closest('td[data-addr]');
    if (!td || !grid.contains(td)) return;
    onCellKey(ev, td.dataset.addr, td);
  });
  grid.addEventListener('mousedown', (ev) => {
    if (ev.button !== 0) return;
    const td = ev.target.closest('td[data-addr]');
    if (!td || !grid.contains(td)) return;
    if (ev.detail > 1) return; // dblclick handled separately
    if (ev.shiftKey && (selAnchorAddr || activeAddr)) {
      if (!selAnchorAddr) selAnchorAddr = activeAddr;
      extendSelectionTo(td);
      ev.preventDefault();
      return;
    }
    selAnchorAddr = td.dataset.addr;
    selectCellOnly(td);
    isSelectingRange = true;
    ev.preventDefault();
  });
  grid.addEventListener('dblclick', (ev) => {
    const td = ev.target.closest('td[data-addr]');
    if (!td || !grid.contains(td)) return;
    selAnchorAddr = td.dataset.addr;
    beginCellEdit(td);
  });
  document.addEventListener('mousemove', (ev) => {
    if (!isSelectingRange) return;
    const td = tdFromPoint(ev.clientX, ev.clientY);
    if (!td) return;
    if (td.dataset.addr === activeAddr) return;
    extendSelectionTo(td);
  });
  document.addEventListener('mouseup', () => {
    isSelectingRange = false;
  });
}

function renderColHeader() {
  const host = document.getElementById('colHeadSticky');
  if (!host) return;
  if (!winRows || !winCols) updateWindowFromScroll();
  const rhW = rowHeaderWidthPx();
  const endCol = Math.min(CAPACITY_COLS - 1, startCol + winCols - 1);
  host.style.width = totalGridWidth() + 'px';
  host.style.minWidth = totalGridWidth() + 'px';
  const parts = [
    '<div class="col-head-inner" style="left:' +
      gridOffsetLeft() +
      'px"><div class="col-head-corner" style="width:' +
      rhW +
      'px;min-width:' +
      rhW +
      'px"></div>',
  ];
  for (let c = startCol; c <= endCol; c++) {
    const w = colWidthAt(c);
    parts.push(
      '<div class="col-head-cell" data-col="' +
        c +
        '" style="width:' +
        w +
        'px;min-width:' +
        w +
        'px;max-width:' +
        w +
        'px">' +
        colLetter(c) +
        '</div>'
    );
  }
  parts.push('</div>');
  host.innerHTML = parts.join('');
  host.querySelectorAll('.col-head-cell').forEach((th) => {
    const c = parseInt(th.getAttribute('data-col') || '0', 10);
    setupColumnResize(th, c);
  });
}

function renderGrid() {
  ensureGridDelegation();
  if (!winRows || !winCols) updateWindowFromScroll();
  if (!selAnchorAddr) selAnchorAddr = activeAddr;
  const rhW = rowHeaderWidthPx();
  const endRow = Math.min(CAPACITY_ROWS, startRow + winRows - 1);
  const endCol = Math.min(CAPACITY_COLS - 1, startCol + winCols - 1);
  renderColHeader();
  const parts = [];
  parts.push(
    '<table class="sheet' +
      (freezeOn ? ' freeze-on' : '') +
      '" role="grid"><colgroup><col style="width:' +
      rhW +
      'px"/>'
  );
  for (let c = startCol; c <= endCol; c++) {
    parts.push('<col style="width:' + colWidthAt(c) + 'px"/>');
  }
  parts.push('</colgroup><tbody>');
  for (let r = startRow; r <= endRow; r++) {
    parts.push(
      '<tr data-row="' +
        r +
        '"><th class="rowhd" style="width:' +
        rhW +
        'px;min-width:' +
        rhW +
        'px">' +
        r +
        '</th>'
    );
    for (let c = startCol; c <= endCol; c++) {
      if (isCoveredByMerge(c, r)) continue;
      const addr = makeAddr(c, r);
      const m = mergeAt(c, r);
      let attrs =
        ' data-addr="' +
        addr +
        '" role="gridcell" tabindex="' +
        (addr === activeAddr ? '0' : '-1') +
        '"';
      let cls = '';
      if (m && m.c1 === c && m.r1 === r) {
        const mc2 = Math.min(m.c2, endCol);
        const mr2 = Math.min(m.r2, endRow);
        attrs +=
          ' colspan="' +
          (mc2 - c + 1) +
          '" rowspan="' +
          (mr2 - r + 1) +
          '"';
        cls = ' class="merged"';
      } else if (m && (m.c1 !== c || m.r1 !== r)) {
        continue;
      }
      parts.push('<td' + cls + attrs + '></td>');
    }
    parts.push('</tr>');
  }
  parts.push('</tbody></table>');
  grid.innerHTML = parts.join('');
  grid.style.position = 'absolute';
  // Body sits under the sticky column-letter strip (SCROLL_ROW_H).
  grid.style.top = SCROLL_ROW_H + gridOffsetTop() + 'px';
  grid.style.left = gridOffsetLeft() + 'px';
  grid.style.width = 'max-content';
  paintGridValues();
  applyRowFilter();
  applyFreezePanesStyles();
  syncGridPhantomSize();
  updateSheetStats();
  wireVirtualScroll();
}

function onCellKey(ev, addr, td) {
  const nav = { ArrowUp: [0, -1], ArrowDown: [0, 1], ArrowLeft: [-1, 0], ArrowRight: [1, 0] };
  if (!editing) {
    if (ev.key === 'F2') {
      ev.preventDefault();
      beginCellEdit(td);
      return;
    }
    if (ev.key === 'Enter' && !ev.shiftKey) {
      ev.preventDefault();
      beginCellEdit(td);
      return;
    }
    if (ev.key === 'Delete' || ev.key === 'Backspace') {
      ev.preventDefault();
      clearActiveCell();
      return;
    }
    if (nav[ev.key] && !ev.ctrlKey && !ev.metaKey && !ev.altKey) {
      ev.preventDefault();
      moveActive(nav[ev.key][0], nav[ev.key][1], { extend: ev.shiftKey });
      return;
    }
    if (ev.key === 'Tab') {
      ev.preventDefault();
      moveActive(ev.shiftKey ? -1 : 1, 0);
      return;
    }
    if (ev.key.length === 1 && !ev.ctrlKey && !ev.metaKey && !ev.altKey) {
      ev.preventDefault();
      beginCellEdit(td, ev.key);
      return;
    }
    return;
  }
  if (ev.key === 'Enter' && !ev.shiftKey) {
    ev.preventDefault();
    td.blur();
    moveActive(0, 1);
    return;
  }
  if (ev.key === 'Escape') {
    ev.preventDefault();
    editing = false;
    td.contentEditable = 'false';
    paintCellTd(td, addr);
    paintSelection();
    syncFormulaBar();
    return;
  }
  if (ev.key === 'Tab') {
    ev.preventDefault();
    td.blur();
    moveActive(ev.shiftKey ? -1 : 1, 0);
  }
}

function moveActive(dc, dr, opts) {
  const extend = !!(opts && opts.extend);
  const p = parseAddr(activeAddr) || { col: 0, row: 1 };
  const col = Math.max(0, Math.min(CAPACITY_COLS - 1, p.col + dc));
  const row = Math.max(1, Math.min(CAPACITY_ROWS, p.row + dr));
  if (!extend) selAnchorAddr = makeAddr(col, row);
  else if (!selAnchorAddr) selAnchorAddr = activeAddr;
  activeAddr = makeAddr(col, row);
  ensureAddrVisible(activeAddr);
  const td = grid.querySelector('td[data-addr="' + activeAddr + '"]');
  if (td) {
    td.contentEditable = 'false';
    td.focus({ preventScroll: true });
  }
  paintSelection();
  syncFormulaBar();
}

function goToAddr(raw) {
  const rawStr = String(raw || '').trim().toUpperCase();
  if (rawStr.includes(':')) {
    const parsed = parseRangeStr(rawStr);
    if (!parsed) {
      setAuthStatus('Invalid address', true);
      return false;
    }
    selAnchorAddr = makeAddr(parsed.c1, parsed.r1);
    activeAddr = makeAddr(parsed.c2, parsed.r2);
    ensureAddrVisible(activeAddr);
    paintSelection();
    syncFormulaBar();
    setAuthStatus('Select ' + parsed.key, false);
    return true;
  }
  const p = parseAddr(rawStr);
  if (!p || p.row < 1 || p.row > CAPACITY_ROWS || p.col < 0 || p.col >= CAPACITY_COLS) {
    setAuthStatus('Invalid address', true);
    return false;
  }
  activeAddr = makeAddr(p.col, p.row);
  selAnchorAddr = activeAddr;
  ensureAddrVisible(activeAddr);
  const td = grid.querySelector('td[data-addr="' + activeAddr + '"]');
  if (td) {
    td.contentEditable = 'false';
    td.focus({ preventScroll: true });
  }
  paintSelection();
  syncFormulaBar();
  setAuthStatus('Go to ' + activeAddr, false);
  return true;
}

function adjustFormulaRows(formula, rowDelta) {
  if (!formula || !rowDelta) return formula;
  return formula.replace(/([A-Z]+)(\d+)/gi, (m, col, row) => {
    const newRow = parseInt(row, 10) + rowDelta;
    if (newRow < 1) return m;
    return col.toUpperCase() + newRow;
  });
}

function commitCell(addr, raw) {
  if (isAddrProtected(addr)) {
    setAuthStatus(
      sheetProtected
        ? 'Sheet is protected — unprotect to edit'
        : 'Cell is in a protected range',
      true
    );
    paintGridValues();
    return;
  }
  const text = String(raw).replace(/\u00a0/g, ' ').trim();
  let value = text;
  let formula = '';
  if (text.startsWith('=')) {
    formula = text;
    value = '';
  }
  const prev = cellOf(addr);
  if ((prev.value || '') === value && (prev.formula || '') === formula) {
    paintGridValues();
    return;
  }
  if (!sheetState.cells) sheetState.cells = {};
  sheetState.cells[addr] = { value: value || prev.value || '', formula };
  sendCell(addr, value, formula);
  scheduleRefresh();
}

function toggleProtectSheet() {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  sheetProtected = !sheetProtected;
  sendSheetOp({ type: 'protect_sheet', protected: sheetProtected });
  updateProtectUi();
  setAuthStatus(sheetProtected ? 'Sheet protected' : 'Sheet unprotected', false);
}

function csvEscape(v) {
  const s = String(v == null ? '' : v);
  if (/[",\n\r]/.test(s)) return '"' + s.replace(/"/g, '""') + '"';
  return s;
}

function exportCsv() {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  const cells = sheetState.cells || {};
  let maxR = 1;
  let maxC = 0;
  Object.keys(cells).forEach((addr) => {
    const p = parseAddr(addr);
    if (!p) return;
    if (p.row > maxR) maxR = p.row;
    if (p.col > maxC) maxC = p.col;
  });
  maxR = Math.min(maxR, CAPACITY_ROWS);
  maxC = Math.min(maxC, CAPACITY_COLS - 1);
  const lines = [];
  for (let r = 1; r <= maxR; r++) {
    const row = [];
    for (let c = 0; c <= maxC; c++) {
      row.push(csvEscape(displayFor(makeAddr(c, r))));
    }
    lines.push(row.join(','));
  }
  const blob = new Blob([lines.join('\r\n') + '\r\n'], { type: 'text/csv;charset=utf-8' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = (sheetId || 'export') + '.csv';
  a.click();
  URL.revokeObjectURL(a.href);
  setAuthStatus('CSV downloaded', false);
}

function persistMerges() {
  const list = cellMerges.map((m) => m.key);
  if (sheetState.sheets && sheetState.sheets[activeSheetIndex]) {
    sheetState.sheets[activeSheetIndex].merges = list.slice();
  }
  sendSheetOp({ type: 'set_merges', merges: list });
}

function persistProtectedRanges() {
  if (sheetState.sheets && sheetState.sheets[activeSheetIndex]) {
    sheetState.sheets[activeSheetIndex].protected_ranges = protectedRanges.slice();
  }
  sendSheetOp({ type: 'set_protected_ranges', ranges: protectedRanges.slice() });
}

function selectionMergeRange() {
  const r = currentSelectionRange();
  if (!r || (r.c1 === r.c2 && r.r1 === r.r2)) return null;
  return r;
}

async function mergeCellsLite() {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  if (sheetProtected) {
    setAuthStatus('Sheet is protected — unprotect to merge', true);
    return;
  }
  let parsed = selectionMergeRange();
  if (!parsed) {
    const p = parseAddr(activeAddr) || { col: 0, row: 1 };
    const def = activeAddr + ':' + makeAddr(Math.min(CAPACITY_COLS - 1, p.col + 1), p.row);
    const input = await EraOfficeShell.promptText({
      title: 'Merge cells',
      label: 'Range (A1:B1)',
      value: def,
      message: 'Select a range with drag or Shift+arrows, or type a range.',
    });
    if (input == null) return;
    parsed = parseRangeStr(input);
  }
  if (!parsed || (parsed.c1 === parsed.c2 && parsed.r1 === parsed.r2)) {
    setAuthStatus('Need a multi-cell range to merge', true);
    return;
  }
  cellMerges = cellMerges.filter((m) => !rangesOverlap(m, parsed));
  cellMerges.push(parsed);
  persistMerges();
  renderGrid();
  setAuthStatus('Merged ' + parsed.key, false);
}

function unmergeCellsLite() {
  if (!sheetId) return;
  const p = parseAddr(activeAddr);
  if (!p) return;
  const before = cellMerges.length;
  cellMerges = cellMerges.filter(
    (m) => !(p.col >= m.c1 && p.col <= m.c2 && p.row >= m.r1 && p.row <= m.r2)
  );
  if (cellMerges.length === before) {
    setAuthStatus('No merge at ' + activeAddr, true);
    return;
  }
  persistMerges();
  renderGrid();
  setAuthStatus('Unmerged at ' + activeAddr, false);
}

async function protectRangesLite() {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  const list = protectedRanges.length ? protectedRanges.join(', ') : '(none)';
  const sel = selectionLabel() || activeAddr || 'A1';
  const action = await EraOfficeShell.chooseOption({
    title: 'Protect ranges',
    message: 'Currently protected: ' + list,
    options: [
      { value: 'add_sel', label: 'Protect selection', hint: sel },
      { value: 'add_custom', label: 'Protect custom range…' },
      { value: 'clear', label: 'Clear all protected ranges' },
    ],
    value: 'add_sel',
  });
  if (!action) return;
  if (action === 'clear') {
    protectedRanges = [];
    persistProtectedRanges();
    paintGridValues();
    setAuthStatus('Protected ranges cleared', false);
    return;
  }
  let parsed = null;
  if (action === 'add_sel') {
    parsed = currentSelectionRange() || parseRangeStr(sel);
  } else {
    const input = await EraOfficeShell.promptText({
      title: 'Protect custom range',
      label: 'Range (A1:B2)',
      value: protectedRanges[0] || 'A1:B2',
    });
    if (input == null) return;
    parsed = parseRangeStr(input);
  }
  if (!parsed) {
    setAuthStatus('Invalid range', true);
    return;
  }
  if (!protectedRanges.includes(parsed.key)) protectedRanges.push(parsed.key);
  persistProtectedRanges();
  paintGridValues();
  setAuthStatus('Protected range ' + parsed.key, false);
}

function fillFilterColSelect(sel) {
  if (!sel) return;
  sel.innerHTML = '';
  let maxC = DEFAULT_VIEW_COLS - 1;
  Object.keys(sheetState.cells || {}).forEach((addr) => {
    const p = parseAddr(addr);
    if (p && p.col > maxC) maxC = p.col;
  });
  maxC = Math.min(CAPACITY_COLS - 1, Math.max(maxC, DEFAULT_VIEW_COLS - 1));
  for (let c = 0; c <= maxC; c++) {
    const opt = document.createElement('option');
    opt.value = String(c);
    opt.textContent = colLetter(c);
    sel.appendChild(opt);
  }
}

function openFilterOptions() {
  const dlg = document.getElementById('filterOptsDlg');
  const colSel = document.getElementById('filterOptsCol');
  const andCol = document.getElementById('filterOptsAndCol');
  if (!dlg || !colSel) return;
  fillFilterColSelect(colSel);
  fillFilterColSelect(andCol);
  if (filterOpts) {
    colSel.value = String(filterOpts.col);
    document.getElementById('filterOptsMode').value = filterOpts.mode || 'contains';
    document.getElementById('filterOptsVal').value = filterOpts.value || '';
    if (filterOpts.and) {
      if (andCol) andCol.value = String(filterOpts.and.col || 0);
      const am = document.getElementById('filterOptsAndMode');
      const av = document.getElementById('filterOptsAndVal');
      if (am) am.value = filterOpts.and.mode || 'contains';
      if (av) av.value = filterOpts.and.value || '';
    } else {
      const av = document.getElementById('filterOptsAndVal');
      if (av) av.value = '';
    }
  } else {
    const p = parseAddr(activeAddr);
    if (p) colSel.value = String(p.col);
  }
  if (typeof dlg.showModal === 'function') dlg.showModal();
}

function sendCell(addr, value, formula) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  ws.send(JSON.stringify({ type: 'set_cell', addr, value: value || '', formula: formula || '' }));
}

function sendSheetOp(op) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  ws.send(JSON.stringify(op));
}

function insertFormula(template) {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  formulaInput.value = template;
  formulaInput.focus();
  setAuthStatus('Edit formula then press Enter', false);
}

function fillDownActive() {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  const p = parseAddr(activeAddr);
  if (!p) return;
  const src = cellOf(activeAddr);
  const srcFormula = src.formula || '';
  const srcValue = srcFormula ? '' : src.value || '';
  const srcDisplay = srcFormula || srcValue;
  if (!srcDisplay) {
    setAuthStatus('Active cell is empty', true);
    return;
  }
  for (let i = 1; i <= 5; i++) {
    const targetRow = p.row + i;
    if (targetRow > CAPACITY_ROWS) break;
    const addr = makeAddr(p.col, targetRow);
    if (srcFormula) {
      const adjusted = adjustFormulaRows(srcFormula, i);
      commitCell(addr, adjusted);
    } else {
      commitCell(addr, srcValue);
    }
  }
  setAuthStatus('Filled down 5 rows', false);
}

function fillRightActive() {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  const p = parseAddr(activeAddr);
  if (!p) return;
  const src = cellOf(activeAddr);
  const srcFormula = src.formula || '';
  const srcValue = srcFormula ? '' : src.value || '';
  if (!(srcFormula || srcValue)) {
    setAuthStatus('Active cell is empty', true);
    return;
  }
  for (let i = 1; i <= 5; i++) {
    const targetCol = p.col + i;
    if (targetCol >= CAPACITY_COLS) break;
    const addr = makeAddr(targetCol, p.row);
    commitCell(addr, srcFormula || srcValue);
  }
  setAuthStatus('Filled right 5 cols', false);
}

function setNumberFormat(fmt) {
  if (!sheetId) return;
  if (isAddrProtected(activeAddr)) {
    setAuthStatus(
      sheetProtected
        ? 'Sheet is protected — unprotect to edit'
        : 'Cell is in a protected range',
      true
    );
    return;
  }
  sendSheetOp({ type: 'set_format', addr: activeAddr, format: fmt || '' });
  if (!sheetState.cells) sheetState.cells = {};
  const c = sheetState.cells[activeAddr] || { value: '', formula: '' };
  c.format = fmt || '';
  sheetState.cells[activeAddr] = c;
  paintGridValues();
  setAuthStatus('Format → ' + (fmt || 'automatic'), false);
}

function insertRowAtActive() {
  const p = parseAddr(activeAddr);
  if (!p || !sheetId) return;
  sendSheetOp({ type: 'insert_row', row: p.row });
  scheduleRefresh();
  setAuthStatus('Inserting row ' + p.row, false);
}

function insertColAtActive() {
  const p = parseAddr(activeAddr);
  if (!p || !sheetId) return;
  sendSheetOp({ type: 'insert_col', col: p.col + 1 });
  scheduleRefresh();
  setAuthStatus('Inserting column', false);
}

function deleteRowAtActive() {
  const p = parseAddr(activeAddr);
  if (!p || !sheetId) return;
  sendSheetOp({ type: 'delete_row', row: p.row });
  scheduleRefresh();
}

function deleteColAtActive() {
  const p = parseAddr(activeAddr);
  if (!p || !sheetId) return;
  sendSheetOp({ type: 'delete_col', col: p.col + 1 });
  scheduleRefresh();
}

function clearActiveCell() {
  if (!sheetId) return;
  commitCell(activeAddr, '');
}

let findCellMatches = [];
let findCellCursor = -1;
let lastFindCellQ = '';

function findNextCell() {
  const input = document.getElementById('findCellInput');
  const q = (input && input.value ? input.value : '').trim().toLowerCase();
  if (!q) return;
  if (q !== lastFindCellQ) {
    findCellMatches = [];
    lastFindCellQ = q;
    const cells = sheetState.cells || {};
    const addrs = Object.keys(cells).sort((a, b) => {
      const pa = parseAddr(a);
      const pb = parseAddr(b);
      if (!pa || !pb) return a.localeCompare(b);
      return pa.row - pb.row || pa.col - pb.col;
    });
    for (let i = 0; i < addrs.length; i++) {
      const addr = addrs[i];
      const text = (displayFor(addr) + ' ' + formulaFor(addr)).toLowerCase();
      if (text.includes(q)) findCellMatches.push(addr);
    }
    findCellCursor = -1;
  }
  if (!findCellMatches.length) {
    setAuthStatus('No matches', true);
    return;
  }
  findCellCursor = (findCellCursor + 1) % findCellMatches.length;
  activeAddr = findCellMatches[findCellCursor];
  ensureAddrVisible(activeAddr);
  highlightActive();
  syncFormulaBar();
  const td = grid.querySelector('td[data-addr="' + activeAddr + '"]');
  if (td) td.focus();
  setAuthStatus('Match ' + (findCellCursor + 1) + ' of ' + findCellMatches.length, false);
}

async function renameActiveSheetTab() {
  if (!sheetId) return;
  const cur = sheetTabNames[activeSheetIndex] || 'Sheet';
  const name = await EraOfficeShell.promptText({
    title: 'Rename sheet',
    label: 'Name',
    value: cur,
  });
  if (name == null || !name.trim()) return;
  sendSheetOp({ type: 'rename_sheet', index: activeSheetIndex, name: name.trim() });
  sheetTabNames[activeSheetIndex] = name.trim();
  renderSheetTabs();
}

function drawBarChart(host, range, vals, chartType) {
  const type = String(chartType || 'bar').toLowerCase() === 'line' ? 'line' : 'bar';
  if (!vals || !vals.length) {
    host.innerHTML =
      '<div class="era-chart-card"><div class="era-chart-card-head"><strong>Chart</strong>' +
      '<button type="button" class="era-btn era-chart-clear">Clear</button></div>' +
      '<p class="era-hint">No numeric values in range ' +
      escapeHtmlSafe(range) +
      '.</p></div>';
    const c = host.querySelector('.era-chart-clear');
    if (c) c.onclick = () => clearSheetCharts();
    return;
  }
  const padL = 36;
  const padB = 22;
  const padT = 12;
  const padR = 12;
  const w = 360;
  const h = 160;
  const plotW = w - padL - padR;
  const plotH = h - padT - padB;
  const max = Math.max(1, ...vals);
  const min = Math.min(0, ...vals);
  const span = max - min || 1;
  const n = vals.length;
  let series = '';
  if (type === 'line') {
    const pts = vals
      .map((v, i) => {
        const x = padL + (n <= 1 ? plotW / 2 : (i * plotW) / (n - 1));
        const y = padT + plotH - ((v - min) / span) * plotH;
        return x.toFixed(1) + ',' + y.toFixed(1);
      })
      .join(' ');
    series =
      '<polyline fill="none" stroke="#3a6ea5" stroke-width="2" points="' + pts + '"/>';
    vals.forEach((v, i) => {
      const x = padL + (n <= 1 ? plotW / 2 : (i * plotW) / (n - 1));
      const y = padT + plotH - ((v - min) / span) * plotH;
      series +=
        '<circle cx="' + x.toFixed(1) + '" cy="' + y.toFixed(1) + '" r="2.5" fill="#3a6ea5"/>';
    });
  } else {
    const barW = Math.max(4, Math.floor(plotW / Math.max(1, n)) - 2);
    vals.forEach((v, i) => {
      const bh = Math.round(((v - min) / span) * plotH);
      const x = padL + i * (barW + 2) + 1;
      const y = padT + plotH - bh;
      series +=
        '<rect x="' +
        x +
        '" y="' +
        y +
        '" width="' +
        barW +
        '" height="' +
        Math.max(1, bh) +
        '" fill="#3a6ea5"/>';
    });
  }
  const axis =
    '<line x1="' +
    padL +
    '" y1="' +
    (padT + plotH) +
    '" x2="' +
    (w - padR) +
    '" y2="' +
    (padT + plotH) +
    '" stroke="#c5cad3" stroke-width="1"/>' +
    '<line x1="' +
    padL +
    '" y1="' +
    padT +
    '" x2="' +
    padL +
    '" y2="' +
    (padT + plotH) +
    '" stroke="#c5cad3" stroke-width="1"/>' +
    '<text x="4" y="' +
    (padT + 10) +
    '" font-size="10" fill="#5c6770">' +
    (Math.round(max * 100) / 100) +
    '</text>' +
    '<text x="4" y="' +
    (padT + plotH) +
    '" font-size="10" fill="#5c6770">' +
    (Math.round(min * 100) / 100) +
    '</text>' +
    '<text x="' +
    padL +
    '" y="' +
    (h - 4) +
    '" font-size="10" fill="#5c6770">' +
    n +
    ' pts · ' +
    type +
    '</text>';
  host.innerHTML =
    '<div class="era-chart-card">' +
    '<div class="era-chart-card-head">' +
    '<strong>Chart · ' +
    escapeHtmlSafe(range) +
    '</strong>' +
    '<span class="era-chart-card-actions">' +
    '<button type="button" class="era-btn era-chart-type" data-type="bar">Bar</button>' +
    '<button type="button" class="era-btn era-chart-type" data-type="line">Line</button>' +
    '<button type="button" class="era-btn era-chart-clear" data-tip="Remove chart">Clear</button>' +
    '</span></div>' +
    '<svg width="' +
    w +
    '" height="' +
    h +
    '" class="era-chart-svg" aria-label="' +
    type +
    ' chart">' +
    axis +
    series +
    '</svg></div>';
  const clearBtn = host.querySelector('.era-chart-clear');
  if (clearBtn) clearBtn.onclick = () => clearSheetCharts();
  host.querySelectorAll('.era-chart-type').forEach((btn) => {
    btn.classList.toggle('era-btn-primary', btn.getAttribute('data-type') === type);
    btn.onclick = () => {
      const t = btn.getAttribute('data-type') || 'bar';
      drawBarChart(host, range, vals, t);
      const charts = [{ chart_type: t, range: String(range).toUpperCase(), title: '' }];
      if (sheetState.sheets && sheetState.sheets[activeSheetIndex]) {
        sheetState.sheets[activeSheetIndex].charts = charts;
      }
      sendSheetOp({ type: 'set_charts', charts });
    };
  });
}

function drawSparklineSvg(host, label, vals) {
  const w = 160;
  const h = 36;
  const min = Math.min(...vals);
  const max = Math.max(...vals);
  const span = max - min || 1;
  const pts = vals
    .map((v, i) => {
      const x = vals.length <= 1 ? w / 2 : 2 + (i * (w - 4)) / (vals.length - 1);
      const y = h - 3 - ((v - min) / span) * (h - 6);
      return x.toFixed(1) + ',' + y.toFixed(1);
    })
    .join(' ');
  host.innerHTML =
    '<div class="era-chart-card era-chart-card-spark">' +
    '<div class="era-chart-card-head">' +
    '<strong>Sparkline · ' +
    escapeHtmlSafe(label) +
    '</strong>' +
    '<button type="button" class="era-btn era-chart-clear">Clear</button>' +
    '</div>' +
    '<svg width="' +
    w +
    '" height="' +
    h +
    '" class="era-chart-svg" aria-label="Sparkline">' +
    '<polyline fill="none" stroke="#3a6ea5" stroke-width="1.5" points="' +
    pts +
    '"/>' +
    '</svg></div>';
  const clearBtn = host.querySelector('.era-chart-clear');
  if (clearBtn) clearBtn.onclick = () => clearSheetCharts();
}

function clearSheetCharts() {
  const host = document.getElementById('chartHost');
  if (host) host.innerHTML = '';
  if (sheetState.sheets && sheetState.sheets[activeSheetIndex]) {
    sheetState.sheets[activeSheetIndex].charts = [];
  }
  sendSheetOp({ type: 'set_charts', charts: [] });
  setAuthStatus('Chart cleared', false);
}

function restoreCharts(charts) {
  const host = document.getElementById('chartHost');
  if (!host) return;
  if (!charts || !charts.length) {
    host.innerHTML = '';
    return;
  }
  const ch = charts[0];
  const collected = collectRangeValues(ch.range);
  if (!collected) return;
  if (String(ch.chart_type || '').toLowerCase() === 'sparkline') {
    drawSparklineSvg(host, ch.range || collected.label, collected.vals);
    return;
  }
  drawBarChart(host, ch.range || collected.label, collected.vals, ch.chart_type || 'bar');
}

async function renderLiteChart() {
  const host = document.getElementById('chartHost');
  if (!host) return;
  const kind = await EraOfficeShell.chooseOption({
    title: 'Insert chart',
    message: 'Choose chart type, then a numeric range.',
    options: [
      { value: 'bar', label: 'Bar chart', hint: 'Compare values side by side' },
      { value: 'line', label: 'Line chart', hint: 'Trends across points' },
    ],
    value: 'bar',
  });
  if (!kind) return;
  const def = selectionLabel() || 'A1:A5';
  const range = await EraOfficeShell.promptText({
    title: 'Chart data range',
    label: 'Range',
    value: def.includes(':') ? def : 'A1:A5',
  });
  if (range == null || !String(range).trim()) return;
  const collected = collectRangeValues(range);
  if (!collected) {
    setAuthStatus('Bad range', true);
    return;
  }
  if (!collected.vals.length) {
    setAuthStatus('No numeric values in range', true);
    return;
  }
  drawBarChart(host, range.toUpperCase(), collected.vals, kind);
  const charts = [{ chart_type: kind, range: range.toUpperCase(), title: '' }];
  if (sheetState.sheets && sheetState.sheets[activeSheetIndex]) {
    sheetState.sheets[activeSheetIndex].charts = charts;
  }
  sendSheetOp({ type: 'set_charts', charts });
  setAuthStatus('Chart saved on sheet tab', false);
}

function collectRangeValues(range) {
  const parts = String(range || '')
    .toUpperCase()
    .split(':');
  const start = parseAddr(parts[0]);
  const end = parseAddr(parts[1] || parts[0]);
  if (!start || !end) return null;
  const vals = [];
  for (let r = Math.min(start.row, end.row); r <= Math.max(start.row, end.row); r++) {
    for (let c = Math.min(start.col, end.col); c <= Math.max(start.col, end.col); c++) {
      const n = parseFloat(displayFor(makeAddr(c, r)) || cellOf(makeAddr(c, r)).value);
      vals.push(isNaN(n) ? 0 : n);
    }
  }
  return { start, end, vals, label: parts.join(':') };
}

async function renderLiteSparkline() {
  const host = document.getElementById('chartHost');
  if (!host) return;
  const def = selectionLabel() || 'A1:A5';
  const range = await EraOfficeShell.promptText({
    title: 'Insert sparkline',
    label: 'Data range',
    value: def.includes(':') ? def : 'A1:A5',
  });
  if (range == null || !String(range).trim()) return;
  const collected = collectRangeValues(range);
  if (!collected) {
    setAuthStatus('Bad range', true);
    return;
  }
  drawSparklineSvg(host, collected.label, collected.vals);
  const charts = [{ chart_type: 'sparkline', range: range.toUpperCase(), title: 'sparkline' }];
  if (sheetState.sheets && sheetState.sheets[activeSheetIndex]) {
    sheetState.sheets[activeSheetIndex].charts = charts;
  }
  sendSheetOp({ type: 'set_charts', charts });
  setAuthStatus('Sparkline saved · ' + collected.label, false);
}

/** @type {Record<string, Record<string, string>>} */
let scenariosStore = Object.create(null);

function loadScenariosFromTab(tab) {
  scenariosStore = Object.create(null);
  const raw = tab && tab.scenarios;
  if (!raw || typeof raw !== 'object') return;
  Object.keys(raw).forEach((name) => {
    if (raw[name] && typeof raw[name] === 'object') scenariosStore[name] = Object.assign({}, raw[name]);
  });
}

function persistScenarios() {
  const payload = {};
  Object.keys(scenariosStore).forEach((k) => {
    payload[k] = scenariosStore[k];
  });
  if (sheetState.sheets && sheetState.sheets[activeSheetIndex]) {
    sheetState.sheets[activeSheetIndex].scenarios = payload;
  }
  sendSheetOp({ type: 'set_scenarios', scenarios: payload });
}

function evalSimpleFormula(formula) {
  let f = String(formula || '').trim();
  if (!f.startsWith('=')) return null;
  f = f.slice(1).trim();
  const expr = f.replace(/([A-Z]+\d+)/gi, (m) => {
    const c = cellOf(m.toUpperCase());
    const n = parseFloat(c.value != null ? c.value : '');
    return isNaN(n) ? '0' : String(n);
  });
  if (!/^[\d+\-*/().\s]+$/.test(expr)) return null;
  try {
    const v = Function('"use strict"; return (' + expr + ')')();
    return typeof v === 'number' && isFinite(v) ? v : null;
  } catch (_) {
    return null;
  }
}

function localNumericFor(addr) {
  const c = cellOf(addr);
  if (c.formula) {
    const v = evalSimpleFormula(c.formula);
    if (v != null) return v;
  }
  const n = parseFloat(displayFor(addr) || (c.value != null ? c.value : ''));
  return isNaN(n) ? NaN : n;
}

function openWhatIfDialog() {
  const dlg = document.getElementById('whatIfDlg');
  if (!dlg) return;
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  const formulaEl = document.getElementById('whatIfFormula');
  const changeEl = document.getElementById('whatIfChange');
  const preview = document.getElementById('whatIfPreview');
  if (formulaEl && !formulaEl.value) formulaEl.value = activeAddr || 'B1';
  if (changeEl && !changeEl.value) changeEl.value = 'A1';
  if (preview) preview.textContent = '';
  if (typeof dlg.showModal === 'function') dlg.showModal();
}

function computeWhatIfSeek(formulaAddr, target, changeAddr) {
  const fAddr = String(formulaAddr || '')
    .trim()
    .toUpperCase();
  const cAddr = String(changeAddr || '')
    .trim()
    .toUpperCase();
  const targetN = parseFloat(target);
  if (!parseAddr(fAddr) || !parseAddr(cAddr) || isNaN(targetN)) {
    return { error: 'What-if: bad inputs' };
  }
  if (!sheetState.cells) sheetState.cells = {};
  const prevChange = Object.assign({}, cellOf(cAddr));
  const prevFormula = Object.assign({}, cellOf(fAddr));
  // Probe slope to pick search direction, then binary search.
  sheetState.cells[cAddr] = { value: '0', formula: '' };
  const y0 = localNumericFor(fAddr);
  sheetState.cells[cAddr] = { value: '1', formula: '' };
  const y1 = localNumericFor(fAddr);
  let lo = -1e9;
  let hi = 1e9;
  const rising = isNaN(y0) || isNaN(y1) || y1 >= y0;
  if (!isNaN(y0) && !isNaN(y1) && Math.abs(y1 - y0) > 1e-12) {
    const est = (targetN - y0) / (y1 - y0);
    const pad = Math.max(1e3, Math.abs(est) * 2);
    lo = est - pad;
    hi = est + pad;
  }
  let best = 0;
  let bestErr = Infinity;
  let bestGot = NaN;
  for (let i = 0; i < 64; i++) {
    const mid = (lo + hi) / 2;
    sheetState.cells[cAddr] = { value: String(mid), formula: '' };
    const got = localNumericFor(fAddr);
    if (isNaN(got)) {
      lo = mid;
      continue;
    }
    const err = Math.abs(got - targetN);
    if (err < bestErr) {
      bestErr = err;
      best = mid;
      bestGot = got;
    }
    if (err < 1e-8) {
      best = mid;
      bestGot = got;
      break;
    }
    if (rising) {
      if (got < targetN) lo = mid;
      else hi = mid;
    } else if (got < targetN) {
      hi = mid;
    } else {
      lo = mid;
    }
  }
  const out =
    Math.abs(best) >= 1e4 || (Math.abs(best) > 0 && Math.abs(best) < 1e-3)
      ? String(best)
      : String(Math.round(best * 1e8) / 1e8);
  if (prevChange.value != null || prevChange.formula) sheetState.cells[cAddr] = prevChange;
  else delete sheetState.cells[cAddr];
  if (prevFormula.formula || prevFormula.value != null) sheetState.cells[fAddr] = prevFormula;
  return {
    fAddr,
    cAddr,
    targetN,
    out,
    bestErr,
    bestGot,
    prevFormula,
  };
}

function runWhatIfSeek(formulaAddr, target, changeAddr, apply) {
  const result = computeWhatIfSeek(formulaAddr, target, changeAddr);
  const preview = document.getElementById('whatIfPreview');
  if (result.error) {
    if (preview) preview.textContent = result.error;
    setAuthStatus(result.error, true);
    return;
  }
  const msg =
    result.cAddr +
    '=' +
    result.out +
    ' → ' +
    result.fAddr +
    ' ≈ ' +
    (isNaN(result.bestGot) ? '?' : String(Math.round(result.bestGot * 1e6) / 1e6)) +
    ' (target ' +
    result.targetN +
    ', err ' +
    result.bestErr.toPrecision(3) +
    ')';
  if (preview) preview.textContent = (apply ? 'Applied: ' : 'Preview: ') + msg;
  if (!apply) {
    setAuthStatus('What-if preview — ' + msg, false);
    return;
  }
  commitCell(result.cAddr, result.out);
  if (result.prevFormula.formula) {
    sheetState.cells[result.cAddr] = { value: result.out, formula: '' };
    const approx = evalSimpleFormula(result.prevFormula.formula);
    sheetState.cells[result.fAddr] = {
      value: approx != null ? String(approx) : result.prevFormula.value || '',
      formula: result.prevFormula.formula,
      format: result.prevFormula.format || '',
    };
    paintGridValues();
  }
  setAuthStatus('What-if: ' + msg, false);
}

function refreshScenariosList() {
  const sel = document.getElementById('scenariosList');
  if (!sel) return;
  const names = Object.keys(scenariosStore).sort();
  sel.innerHTML = '';
  names.forEach((name) => {
    const opt = document.createElement('option');
    opt.value = name;
    opt.textContent = name + ' (' + Object.keys(scenariosStore[name]).length + ' cells)';
    sel.appendChild(opt);
  });
}

function openScenariosDialog() {
  const dlg = document.getElementById('scenariosDlg');
  if (!dlg) return;
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  refreshScenariosList();
  if (typeof dlg.showModal === 'function') dlg.showModal();
}

function saveActiveColumnScenario(name) {
  const n = String(name || '').trim();
  if (!n) {
    setAuthStatus('Scenario name required', true);
    return false;
  }
  const p = parseAddr(activeAddr);
  if (!p) return false;
  /** @type {Record<string, string>} */
  const snap = {};
  Object.keys(sheetState.cells || {}).forEach((addr) => {
    const pa = parseAddr(addr);
    if (!pa || pa.col !== p.col) return;
    const c = cellOf(addr);
    const raw = c.formula || (c.value != null ? String(c.value) : '');
    if (raw !== '') snap[addr] = raw;
  });
  scenariosStore[n] = snap;
  persistScenarios();
  refreshScenariosList();
  setAuthStatus('Scenario saved: ' + n + ' (' + Object.keys(snap).length + ' cells)', false);
  return true;
}

function applyScenario(name) {
  const n = String(name || '').trim();
  const snap = scenariosStore[n];
  if (!snap) {
    setAuthStatus('Scenario not found', true);
    return false;
  }
  Object.keys(snap).forEach((addr) => commitCell(addr, snap[addr]));
  paintGridValues();
  setAuthStatus('Scenario applied: ' + n, false);
  return true;
}

function openConsolidateDialog() {
  const dlg = document.getElementById('consolidateDlg');
  if (!dlg) return;
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  if (typeof dlg.showModal === 'function') dlg.showModal();
}

function cellsMapForSheetRef(sheetRef) {
  const ref = String(sheetRef || 'active').trim();
  if (!ref || /^active$/i.test(ref)) return sheetState.cells || {};
  const byName = (sheetTabNames || []).findIndex(
    (n) => String(n || '').toLowerCase() === ref.toLowerCase()
  );
  let idx = byName;
  if (idx < 0) {
    const n = parseInt(ref, 10);
    if (!isNaN(n)) idx = n;
  }
  if (idx < 0 || !sheetState.sheets || !sheetState.sheets[idx]) return null;
  return sheetState.sheets[idx].cells || {};
}

function collectRangeValuesFromMap(range, cellsMap) {
  const parts = String(range || '')
    .toUpperCase()
    .split(':');
  const start = parseAddr(parts[0]);
  const end = parseAddr(parts[1] || parts[0]);
  if (!start || !end || !cellsMap) return null;
  const vals = [];
  for (let r = Math.min(start.row, end.row); r <= Math.max(start.row, end.row); r++) {
    for (let c = Math.min(start.col, end.col); c <= Math.max(start.col, end.col); c++) {
      const addr = makeAddr(c, r);
      const cell = cellsMap[addr] || cellsMap[addr.toUpperCase()] || {};
      const raw = cell.value != null ? String(cell.value) : '';
      const n = parseFloat(raw);
      vals.push(isNaN(n) ? 0 : n);
    }
  }
  return { vals, label: parts.join(':') };
}

function runConsolidate(sheetRef, range, targetAddr) {
  const tAddr = String(targetAddr || '')
    .trim()
    .toUpperCase();
  if (!parseAddr(tAddr)) {
    setAuthStatus('Consolidate: bad target', true);
    return;
  }
  const map = cellsMapForSheetRef(sheetRef);
  if (!map) {
    setAuthStatus('Consolidate: unknown sheet (use active, name, or index)', true);
    return;
  }
  const collected =
    map === (sheetState.cells || {})
      ? collectRangeValues(range)
      : collectRangeValuesFromMap(range, map);
  if (!collected) {
    setAuthStatus('Consolidate: bad range', true);
    return;
  }
  const sum = collected.vals.reduce((a, b) => a + b, 0);
  commitCell(tAddr, String(sum));
  paintGridValues();
  setAuthStatus('Consolidated sum ' + sum + ' → ' + tAddr, false);
}

function escapeHtmlSafe(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

function toggleFreeze() {
  freezeOn = !freezeOn;
  const table = grid.querySelector('table.sheet');
  if (table) table.classList.toggle('freeze-on', freezeOn);
  document.getElementById('freezeBtn').classList.toggle('era-btn-primary', freezeOn);
  setAuthStatus(freezeOn ? 'Freeze on (header + col A)' : 'Freeze off', false);
}

function defaultColWidth() {
  return 88;
}

function applyFreezePanesStyles() {
  const table = grid.querySelector('table.sheet');
  if (!table) return;
  const panesOn = freezeRows > 0 || freezeCols > 0;
  table.classList.toggle('freeze-panes', panesOn);
  const headerH = SCROLL_ROW_H;
  const sample = table.querySelector('tbody tr');
  const rowH = (sample && sample.offsetHeight) || SCROLL_ROW_H;
  const rowhdW = rowHeaderWidthPx();
  table.querySelectorAll('tbody tr').forEach((tr) => {
    const r = parseInt(tr.dataset.row, 10);
    const rowFrozen = panesOn && r <= freezeRows;
    tr.classList.toggle('frozen-row', rowFrozen);
    tr.querySelectorAll('th, td').forEach((el) => {
      if (rowFrozen) el.style.top = headerH + (r - 1) * rowH + 'px';
      else el.style.top = '';
    });
    const rh = tr.querySelector('th.rowhd');
    if (rh) {
      rh.classList.toggle('frozen-col', freezeCols > 0);
      rh.style.left = freezeCols > 0 ? '0' : '';
    }
    tr.querySelectorAll('td[data-addr]').forEach((td) => {
      const p = parseAddr(td.dataset.addr);
      const absCol = p ? p.col : -1;
      const frozen = panesOn && absCol >= 0 && absCol < freezeCols;
      td.classList.toggle('frozen-col', frozen);
      if (frozen) {
        let left = rowhdW;
        for (let c = 0; c < absCol; c++) left += colWidthAt(c);
        td.style.left = left + 'px';
      } else {
        td.style.left = '';
      }
    });
  });
}

/** Freeze at active cell (rows/cols above-left). */
async function openFreezePanes() {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  const p = parseAddr(activeAddr);
  const rows = p ? Math.max(0, p.row - 1) : 0;
  const cols = p ? p.col : 0;
  const mode = await EraOfficeShell.chooseOption({
    title: 'Freeze panes',
    message:
      'Freeze above/left of ' +
      (activeAddr || 'A1') +
      ' → ' +
      rows +
      ' row(s), ' +
      cols +
      ' col(s).',
    options: [
      { value: 'apply', label: 'Freeze here', hint: rows + ' × ' + cols },
      { value: 'clear', label: 'Unfreeze all' },
    ],
    value: 'apply',
  });
  if (!mode) return;
  if (mode === 'clear') {
    freezeRows = 0;
    freezeCols = 0;
  } else {
    freezeRows = rows;
    freezeCols = cols;
  }
  sendSheetOp({ type: 'freeze_panes', rows: freezeRows, cols: freezeCols });
  applyFreezePanesStyles();
  setAuthStatus(
    freezeRows || freezeCols
      ? 'Freeze panes at selection: ' + freezeRows + ' row(s), ' + freezeCols + ' col(s)'
      : 'Freeze panes cleared',
    false
  );
}

/**
 * Subtotal: group-by left adjacent column when categories change;
 * writes summary rows below the data block (no mid-table row shift).
 */
function insertSubtotalLite() {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  if (sheetProtected) {
    setAuthStatus('Sheet is protected — unprotect to edit', true);
    return;
  }
  const p = parseAddr(activeAddr);
  if (!p) return;
  const valueCol = p.col;
  const groupCol = valueCol > 0 ? valueCol - 1 : null;
  const letter = colLetter(valueCol);

  let startR = null;
  let endR = null;
  let scanMax = 1;
  Object.keys(sheetState.cells || {}).forEach((addr) => {
    const pa = parseAddr(addr);
    if (pa && pa.row > scanMax) scanMax = pa.row;
  });
  scanMax = Math.min(CAPACITY_ROWS, Math.max(scanMax, 1));
  for (let r = 1; r <= scanMax; r++) {
    const vAddr = makeAddr(valueCol, r);
    const gAddr = groupCol != null ? makeAddr(groupCol, r) : null;
    const raw = displayFor(vAddr);
    const gRaw = gAddr ? displayFor(gAddr) : '';
    const f = formulaFor(vAddr);
    const n = parseFloat(raw);
    const has = (raw !== '' && !isNaN(n)) || !!f || gRaw !== '';
    if (has) {
      if (startR === null) startR = r;
      endR = r;
    } else if (startR !== null) {
      break;
    }
  }
  if (startR === null || endR === null) {
    setAuthStatus('Subtotal: no contiguous data in active column', true);
    return;
  }

  const groups = [];
  if (groupCol != null) {
    let gStart = startR;
    let gLabel = displayFor(makeAddr(groupCol, startR)) || '(blank)';
    for (let r = startR + 1; r <= endR + 1; r++) {
      const lab = r <= endR ? displayFor(makeAddr(groupCol, r)) || '(blank)' : null;
      if (lab === null || lab !== gLabel) {
        groups.push({ label: gLabel, from: gStart, to: r - 1 });
        if (lab === null) break;
        gStart = r;
        gLabel = lab;
      }
    }
  } else {
    groups.push({ label: 'Subtotal', from: startR, to: endR });
  }

  let outRow = endR + 1;
  groups.forEach((g) => {
    const formula = '=SUM(' + letter + g.from + ':' + letter + g.to + ')';
    const sumAddr = makeAddr(valueCol, outRow);
    const labelAddr =
      groupCol != null
        ? makeAddr(groupCol, outRow)
        : makeAddr(valueCol === 0 ? 1 : 0, outRow);
    commitCell(labelAddr, String(g.label) + ' total');
    commitCell(sumAddr, formula);
    outRow += 1;
  });
  paintGridValues();
  setAuthStatus('Subtotal: ' + groups.length + ' group(s) below data · ' + letter, false);
}

async function exportOds() {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  setAuthStatus('Exporting ODS…', false);
  const res = await officeFetch('/api/v1/tables/' + encodeURIComponent(sheetId) + '/export/ods', {
    method: 'POST',
    headers: authHeaders(),
  });
  if (!res.ok) {
    setAuthStatus('ODS export failed: ' + res.status, true);
    return;
  }
  const blob = await res.blob();
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = (sheetId || 'export') + '.ods';
  a.click();
  URL.revokeObjectURL(a.href);
  setAuthStatus('ODS export ready', false);
}

function sortActiveColumn(ascending) {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  if (sheetProtected) {
    setAuthStatus('Sheet is protected — unprotect to sort', true);
    return;
  }
  const p = parseAddr(activeAddr);
  if (!p) return;
  // Engine col is 1-based (A=1); UI parseAddr uses 0-based col.
  const col = p.col + 1;
  const asc = ascending !== false;
  sendSheetOp({ type: 'sort_range', col, ascending: asc });
  setAuthStatus(
    (asc ? 'Sorting rows A→Z by column ' : 'Sorting rows Z→A by column ') +
      colLetter(p.col) +
      '…',
    false
  );
  scheduleRefresh();
}

function patchBorderAt(addr, sides) {
  if (!sheetId || !addr) return;
  if (!sheetState.cells) sheetState.cells = {};
  const key = String(addr).toUpperCase();
  const c = sheetState.cells[key] || sheetState.cells[addr] || {};
  sheetState.cells[key] = c;
  const cleaned = String(sides || '')
    .toLowerCase()
    .replace(/[^trbl]/g, '');
  c.border = cleaned.length > 0;
  c.border_sides = cleaned;
  sendSheetOp({
    type: 'set_cell_style',
    addr: key,
    border: c.border,
    border_sides: cleaned,
  });
}

function applyBorderPreset(preset) {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  const range = currentSelectionRange() || parseRangeStr(activeAddr);
  if (!range) return;
  if (preset === 'tblr-outer') {
    for (let r = range.r1; r <= range.r2; r++) {
      for (let c = range.c1; c <= range.c2; c++) {
        let sides = '';
        if (r === range.r1) sides += 't';
        if (r === range.r2) sides += 'b';
        if (c === range.c1) sides += 'l';
        if (c === range.c2) sides += 'r';
        patchBorderAt(makeAddr(c, r), sides);
      }
    }
    setAuthStatus('Outer borders applied', false);
  } else {
    const sides = String(preset || '')
      .toLowerCase()
      .replace(/[^trbl]/g, '');
    for (let r = range.r1; r <= range.r2; r++) {
      for (let c = range.c1; c <= range.c2; c++) {
        patchBorderAt(makeAddr(c, r), sides);
      }
    }
    setAuthStatus(sides ? 'Borders: ' + sides : 'Borders cleared', false);
  }
  paintGridValues();
}

function openBorderSides() {
  const menu = document.getElementById('borderMenu');
  if (menu) {
    menu.classList.add('open');
    return;
  }
  applyBorderPreset('tblr');
}

async function importOdsFile(file) {
  if (!file) return;
  if (!localStorage.getItem('era_token')) {
    setAuthStatus('Sign in via Drive first (era_token).', true);
    return;
  }
  const { tenantId, userId } = identity();
  setAuthStatus('Importing ODS…', false);
  const buf = await file.arrayBuffer();
  const b64 = arrayBufferToBase64(buf);
  const res = await officeFetch('/api/v1/tables/import-ods', {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({
      tenant_id: tenantId,
      user_id: userId,
      name: file.name.replace(/\.ods$/i, '.erat'),
      ods_base64: b64,
    }),
  });
  if (!res.ok) {
    setAuthStatus('ODS import failed: ' + res.status, true);
    return;
  }
  const out = await res.json();
  const id = out.drive_object_id || out.id;
  if (!id) {
    setAuthStatus('ODS import: no sheet id', true);
    return;
  }
  location.href = '/tables/' + encodeURIComponent(id);
}

let refreshTimer = null;
function scheduleRefresh() {
  clearTimeout(refreshTimer);
  refreshTimer = setTimeout(() => {
    if (sheetId) loadSheet(sheetId, true).catch(() => {});
  }, 250);
}

function connectSync(id) {
  stopPresenceHeartbeat();
  if (ws) {
    try {
      ws.close();
    } catch (_) {}
  }
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const token = localStorage.getItem('era_token') || '';
  const q = token ? '?access_token=' + encodeURIComponent(token) : '';
  ws = new WebSocket(`${proto}//${location.host}/api/v1/tables/${id}/sync${q}`);
  ws.onopen = () => {
    console.debug('tables sync connected');
    startPresenceHeartbeat();
  };
  ws.onerror = (e) => console.warn('tables sync error', e);
  ws.onclose = () => stopPresenceHeartbeat();
  ws.onmessage = (ev) => {
    if (ev.data === 'ack') {
      scheduleRefresh();
      return;
    }
    try {
      const parsed = JSON.parse(ev.data);
      if (parsed.type === 'presence') {
        updatePresencePeers(parsed.peers);
        return;
      }
      if (parsed.type === 'presence_cell') {
        const { userId } = identity();
        const uid = parsed.from || parsed.user_id;
        if (uid && uid !== userId && parsed.addr) {
          peerCells[uid] = { addr: parsed.addr, color: parsed.color || peerColor(uid) };
          paintPeerCells();
        }
        return;
      }
      if (parsed.type === 'error' && parsed.code === 'protected') {
        sheetProtected = true;
        updateProtectUi();
        setAuthStatus('Sheet is protected — unprotect to edit', true);
        return;
      }
      if (parsed.cells) {
        applySyncEnvelope(parsed);
        return;
      }
      const op = parsed.op || parsed;
      if (op.type === 'set_cell') {
        if (!sheetState.cells) sheetState.cells = {};
        const prev = sheetState.cells[op.addr] || {};
        sheetState.cells[op.addr] = {
          value: op.value || '',
          formula: op.formula || '',
          format: prev.format || '',
          bold: !!prev.bold,
          align: prev.align || '',
          wrap: !!prev.wrap,
          border: !!prev.border,
        };
        paintGridValues();
        scheduleRefresh();
      } else if (op.type === 'set_cell_style') {
        if (!sheetState.cells) sheetState.cells = {};
        const c = sheetState.cells[op.addr] || { value: '', formula: '' };
        if (op.bold != null) c.bold = !!op.bold;
        if (op.align != null) c.align = op.align || '';
        if (op.wrap != null) c.wrap = !!op.wrap;
        if (op.border != null) c.border = !!op.border;
        sheetState.cells[op.addr] = c;
        paintGridValues();
      }
    } catch (_) {}
  };
}

async function loadSheet(id, quiet) {
  const res = await officeFetch('/api/v1/tables/' + encodeURIComponent(id), {
    headers: authHeaders(),
  });
  if (!res.ok) {
    if (window.EraOfficeShell && EraOfficeShell.handleUnauthorized && EraOfficeShell.handleUnauthorized(res)) {
      return;
    }
    if (!quiet) setAuthStatus('Failed to load sheet: ' + res.status, true);
    return;
  }
  const sheet = await res.json();
  sheetState = sheet;
  sheetId = id;
  syncSheetTabsFromState(sheet);
  renderSheetTabs();
  const prevR = viewRows;
  const prevC = viewCols;
  growViewportFromSheet(sheet);
  const titleEl = document.getElementById('sheetTitle');
  if (titleEl) titleEl.value = sheet.name || 'Untitled sheet';
  const pill = document.getElementById('savePill');
  if (pill && window.EraOfficeShell && EraOfficeShell.setSavePill) {
    EraOfficeShell.setSavePill(pill, 'ok', 'Edits sync live');
  }
  sheetMeta.textContent =
    'Sheet: ' +
    (sheet.name || id) +
    ' · tab ' +
    (sheetTabNames[activeSheetIndex] || activeSheetIndex + 1) +
    ' · virt ' +
    winRows +
    '×' +
    winCols +
    ' (engine ' +
    (sheet.rows || '?') +
    '×' +
    (sheet.cols || '?') +
    ')';
  if (!quiet) setAuthStatus('Sheet open', false);
  updateWindowFromScroll();
  if (!grid.querySelector('table') || prevR !== winRows || prevC !== winCols) {
    renderGrid();
  } else {
    paintGridValues();
    applyFreezePanesStyles();
  }
  applyRowFilter();
  const tab =
    sheet.sheets && sheet.sheets[sheet.active_sheet != null ? sheet.active_sheet : activeSheetIndex];
  if (tab && Array.isArray(tab.charts)) restoreCharts(tab.charts);
  updateSheetStats();
}

function updateSheetStats() {
  const el = document.getElementById('sheetStats');
  if (!el) return;
  const r2 = Math.min(CAPACITY_ROWS, startRow + winRows - 1);
  const c2 = colLetter(Math.min(CAPACITY_COLS - 1, startCol + winCols - 1));
  el.textContent =
    'Window ' +
    startRow +
    '–' +
    r2 +
    ' × ' +
    colLetter(startCol) +
    '–' +
    c2 +
    ' · capacity ' +
    CAPACITY_ROWS +
    ' × WW';
}

function wireVirtualScroll() {
  const wrap = document.getElementById('gridWrap');
  if (!wrap || wrap._eraVirtWired) return;
  wrap._eraVirtWired = true;
  wrap.addEventListener('scroll', () => {
    if (scrollSyncing) return;
    if (virtScrollRaf) cancelAnimationFrame(virtScrollRaf);
    virtScrollRaf = requestAnimationFrame(() => {
      virtScrollRaf = 0;
      if (scrollSyncing) return;
      const prev =
        startRow + ':' + startCol + ':' + winRows + ':' + winCols;
      updateWindowFromScroll();
      const next =
        startRow + ':' + startCol + ':' + winRows + ':' + winCols;
      if (prev !== next) {
        const sl = wrap.scrollLeft;
        const st = wrap.scrollTop;
        renderGrid();
        scrollSyncing = true;
        wrap.scrollLeft = sl;
        wrap.scrollTop = st;
        scrollSyncing = false;
      }
    });
  });
  // Shift/Ctrl + wheel → horizontal (native vertical scroll otherwise).
  wrap.addEventListener(
    'wheel',
    (e) => {
      if (!(e.shiftKey || e.ctrlKey || e.metaKey)) return;
      if (wrap.scrollWidth <= wrap.clientWidth + 1) return;
      wrap.scrollLeft += e.deltaY || e.deltaX;
      e.preventDefault();
    },
    { passive: false }
  );
  wrap.addEventListener('mouseenter', () => {
    try {
      wrap.focus({ preventScroll: true });
    } catch (_) {}
  });
  window.addEventListener('resize', () => {
    if (!sheetId) return;
    updateWindowFromScroll();
    renderGrid();
  });
}

function addRows(n) {
  const wrap = document.getElementById('gridWrap');
  if (wrap) {
    wrap.scrollTop = Math.min(
      wrap.scrollHeight,
      wrap.scrollTop + (n || 100) * SCROLL_ROW_H
    );
  }
  updateSheetStats();
  setAuthStatus('Scrolled +' + (n || 100) + ' rows', false);
}

function addCols(n) {
  const wrap = document.getElementById('gridWrap');
  const step = n || 26;
  if (wrap) {
    let delta = 0;
    for (let i = 0; i < step; i++) delta += SCROLL_COL_W;
    wrap.scrollLeft = Math.min(wrap.scrollWidth, wrap.scrollLeft + delta);
  }
  updateSheetStats();
  setAuthStatus('Scrolled +' + step + ' cols', false);
}

async function createSheet() {
  if (!localStorage.getItem('era_token')) {
    setAuthStatus('Sign in via Drive first → /drive/ (alice@mail.gov.az / 1234)', true);
    return;
  }
  const { tenantId, userId } = identity();
  const res = await officeFetch('/api/v1/tables', {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({
      tenant_id: tenantId,
      user_id: userId,
      name: 'Untitled-' + Date.now() + '.erat',
    }),
  });
  if (!res.ok) {
    if (window.EraOfficeShell && EraOfficeShell.handleUnauthorized && EraOfficeShell.handleUnauthorized(res)) {
      return;
    }
    setAuthStatus(
      res.status === 403 ? 'Create failed: access denied' : 'Create failed: ' + res.status,
      true
    );
    return;
  }
  const data = await res.json();
  if (!data.drive_object_id) {
    setAuthStatus('Create failed: no drive_object_id', true);
    return;
  }
  location.href = '/tables/' + data.drive_object_id;
}

function arrayBufferToBase64(buf) {
  const bytes = new Uint8Array(buf);
  let binary = '';
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode.apply(null, bytes.subarray(i, i + chunk));
  }
  return btoa(binary);
}

document.getElementById('newSheetBtn').onclick = () => {
  createSheet().catch(() => {});
};
const saveSheetBtn = document.getElementById('saveSheetBtn');
if (saveSheetBtn) {
  saveSheetBtn.onclick = () => {
    setAuthStatus(sheetId ? 'Edits sync live' : 'Open or create a sheet first', !sheetId);
  };
}

const printBtn = document.getElementById('printBtn');
if (printBtn) printBtn.onclick = () => window.print();
const pctBtn = document.getElementById('pctBtn');
if (pctBtn) pctBtn.onclick = () => setNumberFormat('%');
const decBtn = document.getElementById('decBtn');
if (decBtn) decBtn.onclick = () => setNumberFormat('0.00');
const mergeBtn = document.getElementById('mergeBtn');
if (mergeBtn) mergeBtn.onclick = () => mergeCellsLite();
const chartBtn = document.getElementById('chartBtn');
if (chartBtn) chartBtn.onclick = () => renderLiteChart();
const filterTbBtn = document.getElementById('filterTbBtn');
if (filterTbBtn) {
  filterTbBtn.onclick = () => {
    const el = document.getElementById('filterInput');
    if (el) el.focus();
  };
}
const zoomSelect = document.getElementById('zoomSelect');
if (zoomSelect) {
  zoomSelect.onchange = () => {
    const wrap = document.getElementById('gridWrap');
    const z = parseInt(zoomSelect.value, 10) || 100;
    if (wrap) wrap.style.zoom = String(z / 100);
  };
}
let paintedStyle = null;
const formatPainterBtn = document.getElementById('formatPainterBtn');
if (formatPainterBtn) {
  formatPainterBtn.onclick = () => {
    if (!activeAddr) return;
    const c = cellOf(activeAddr);
    paintedStyle = { bold: !!c.bold, align: c.align || '', wrap: !!c.wrap, border: !!c.border };
    setAuthStatus('Format painter: click a target cell, then apply via Bold/Align', false);
    formatPainterBtn.classList.add('active');
  };
}

document.getElementById('sumBtn').onclick = () => insertFormula('=SUM(A1:A2)');
const avgBtn = document.getElementById('avgBtn');
if (avgBtn) avgBtn.onclick = () => insertFormula('=AVERAGE(A1:A2)');
const minBtn = document.getElementById('minBtn');
if (minBtn) minBtn.onclick = () => insertFormula('=MIN(A1:A2)');
const maxBtn = document.getElementById('maxBtn');
if (maxBtn) maxBtn.onclick = () => insertFormula('=MAX(A1:A2)');
const roundBtn = document.getElementById('roundBtn');
if (roundBtn) roundBtn.onclick = () => insertFormula('=ROUND(A1,2)');
document.getElementById('countBtn').onclick = () => insertFormula('=COUNT(A1:A3)');
document.getElementById('countifBtn').onclick = () => insertFormula('=COUNTIF(A1:A3,10)');
document.getElementById('ifBtn').onclick = () => insertFormula('=IF(A1>0,"yes","no")');
const cellBoldBtn = document.getElementById('cellBoldBtn');
if (cellBoldBtn) {
  cellBoldBtn.onclick = () => {
    if (paintedStyle) {
      setCellStylePatch({
        bold: paintedStyle.bold,
        align: paintedStyle.align || undefined,
        wrap: paintedStyle.wrap,
        border: paintedStyle.border,
      });
      paintedStyle = null;
      formatPainterBtn && formatPainterBtn.classList.remove('active');
      return;
    }
    setCellStylePatch({ bold: !cellOf(activeAddr).bold });
  };
}
const cellAlignLeftBtn = document.getElementById('cellAlignLeftBtn');
if (cellAlignLeftBtn) cellAlignLeftBtn.onclick = () => setCellStylePatch({ align: 'left' });
const cellAlignCenterBtn = document.getElementById('cellAlignCenterBtn');
if (cellAlignCenterBtn) cellAlignCenterBtn.onclick = () => setCellStylePatch({ align: 'center' });
const cellAlignRightBtn = document.getElementById('cellAlignRightBtn');
if (cellAlignRightBtn) cellAlignRightBtn.onclick = () => setCellStylePatch({ align: 'right' });
const cellWrapBtn = document.getElementById('cellWrapBtn');
if (cellWrapBtn) cellWrapBtn.onclick = () => setCellStylePatch({ wrap: !cellOf(activeAddr).wrap });
const borderMenu = document.getElementById('borderMenu');
if (borderMenu) {
  borderMenu.querySelectorAll('[data-border]').forEach((btn) => {
    btn.addEventListener('click', () => {
      applyBorderPreset(btn.getAttribute('data-border') || '');
      borderMenu.classList.remove('open');
    });
  });
}
const pasteValuesBtn = document.getElementById('pasteValuesBtn');
if (pasteValuesBtn) pasteValuesBtn.onclick = () => pasteValuesActive();
const linkCellBtn = document.getElementById('linkCellBtn');
if (linkCellBtn) {
  linkCellBtn.onclick = () => {
    void (async () => {
      if (!activeAddr) return;
      const url = await EraOfficeShell.promptText({
        title: 'Insert link',
        label: 'URL',
        value: displayFor(activeAddr) || 'https://',
        placeholder: 'https://…',
      });
      if (url == null || !String(url).trim()) return;
      commitCell(activeAddr, String(url).trim());
    })();
  };
}
const commentCellBtn = document.getElementById('commentCellBtn');
if (commentCellBtn) {
  commentCellBtn.onclick = () => {
    void (async () => {
      if (!activeAddr) {
        setAuthStatus('Select a cell first', true);
        return;
      }
      const cur = cellOf(activeAddr).note || '';
      const note = await EraOfficeShell.promptText({
        title: 'Cell note',
        label: 'Note (empty to clear)',
        value: cur,
        multiline: true,
      });
      if (note == null) return;
      setCellStylePatch({ note: note.trim() });
      if (window.EraOfficeShell && EraOfficeShell.setCommentsOpen) {
        EraOfficeShell.setCommentsOpen(true);
      }
    })();
  };
}
if (window.EraOfficeShell) {
  if (EraOfficeShell.wireSessionWatch) EraOfficeShell.wireSessionWatch();
  if (EraOfficeShell.wireCommentsToggle) EraOfficeShell.wireCommentsToggle(false);
}
function bumpFontSizeTb(delta) {
  const sel = document.getElementById('sizeSelect');
  const sizes = [10, 11, 12, 14, 16, 18, 24];
  let cur = parseInt((sel && sel.value) || '10', 10) || 10;
  let idx = sizes.indexOf(cur);
  if (idx < 0) idx = 0;
  idx = Math.max(0, Math.min(sizes.length - 1, idx + delta));
  if (sel) sel.value = String(sizes[idx]);
  setCellStylePatch({ font_size: sizes[idx] });
}
const fontDecTbBtn = document.getElementById('fontDecTbBtn');
if (fontDecTbBtn) fontDecTbBtn.onclick = () => bumpFontSizeTb(-1);
const fontIncTbBtn = document.getElementById('fontIncTbBtn');
if (fontIncTbBtn) fontIncTbBtn.onclick = () => bumpFontSizeTb(1);
if (window.EraOfficeToolbar) EraOfficeToolbar.init(document);
if (window.EraOfficeShell && EraOfficeShell.mountIcons) EraOfficeShell.mountIcons(document);
document.getElementById('fillDownBtn').onclick = () => fillDownActive();
document.getElementById('fillRightBtn').onclick = () => fillRightActive();
document.getElementById('insertRowBtn').onclick = () => insertRowAtActive();
document.getElementById('freezeBtn').onclick = () => toggleFreeze();
document.getElementById('sortColBtn').onclick = () => sortActiveColumn(true);
document.getElementById('findCellBtn').onclick = () => findNextCell();
document.getElementById('numberFormat').onchange = (e) => setNumberFormat(e.target.value);
const fontSelect = document.getElementById('fontSelect');
if (fontSelect) {
  fontSelect.onchange = () => {
    if (!fontSelect.value) return;
    setCellStylePatch({ font: fontSelect.value });
  };
}
const sizeSelect = document.getElementById('sizeSelect');
if (sizeSelect) {
  sizeSelect.onchange = () => {
    const n = parseInt(sizeSelect.value, 10);
    if (!n) return;
    setCellStylePatch({ font_size: n });
  };
}

if (window.EraOfficeMenubar) {
  function openShareSheet() {
    if (!sheetId) {
      setAuthStatus('Open a sheet first', true);
      return;
    }
    const url = location.origin + '/tables/' + sheetId;
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(url).then(
        () => setAuthStatus('Share link copied', false),
        () => EraOfficeShell.promptCopy({ title: 'Share sheet', value: url })
      );
    } else {
      void EraOfficeShell.promptCopy({ title: 'Share sheet', value: url });
    }
  }
  const shareBtn = document.getElementById('shareBtn');
  if (shareBtn) shareBtn.onclick = () => openShareSheet();

  EraOfficeMenubar.init('#menubar', {
    'file.new': () => createSheet().catch(() => {}),
    'file.import': () => document.getElementById('importBtn').click(),
    'file.export': () => document.getElementById('exportBtn').click(),
    'file.share': () => openShareSheet(),
    'edit.find': () => {
      const el = document.getElementById('findCellInput');
      if (el) el.focus();
    },
    'edit.fillDown': fillDownActive,
    'edit.fillRight': fillRightActive,
    'edit.clear': clearActiveCell,
    'edit.pasteValues': pasteValuesActive,
    'edit.delRow': deleteRowAtActive,
    'edit.delCol': deleteColAtActive,
    'file.ods': () => exportOds().catch(() => {}),
    'file.importOds': () => {
      const inp = document.getElementById('odsFile');
      if (inp) inp.click();
    },
    'file.csv': exportCsv,
    'view.freeze': toggleFreeze,
    'view.freezePanes': openFreezePanes,
    'view.formula': () => formulaInput && formulaInput.focus(),
    'insert.sum': () => insertFormula('=SUM(A1:A2)'),
    'insert.average': () => insertFormula('=AVERAGE(A1:A2)'),
    'insert.min': () => insertFormula('=MIN(A1:A2)'),
    'insert.max': () => insertFormula('=MAX(A1:A2)'),
    'insert.round': () => insertFormula('=ROUND(A1,2)'),
    'insert.count': () => insertFormula('=COUNT(A1:A3)'),
    'insert.countif': () => insertFormula('=COUNTIF(A1:A3,10)'),
    'insert.if': () => insertFormula('=IF(A1>0,"yes","no")'),
    'insert.row': insertRowAtActive,
    'insert.col': insertColAtActive,
    'insert.chart': renderLiteChart,
    'insert.sparkline': renderLiteSparkline,
    'format.number': () => setNumberFormat('0.00'),
    'format.percent': () => setNumberFormat('%'),
    'format.plain': () => setNumberFormat(''),
    'format.merge': mergeCellsLite,
    'format.unmerge': unmergeCellsLite,
    'format.bold': () => setCellStylePatch({ bold: !cellOf(activeAddr).bold }),
    'format.alignLeft': () => setCellStylePatch({ align: 'left' }),
    'format.alignCenter': () => setCellStylePatch({ align: 'center' }),
    'format.alignRight': () => setCellStylePatch({ align: 'right' }),
    'format.wrap': () => setCellStylePatch({ wrap: !cellOf(activeAddr).wrap }),
    'format.border': openBorderSides,
    'data.sort': () => sortActiveColumn(true),
    'data.sortDesc': () => sortActiveColumn(false),
    'data.filter': () => filterInput && filterInput.focus(),
    'data.filterOpts': openFilterOptions,
    'data.subtotal': insertSubtotalLite,
    'data.protect': toggleProtectSheet,
    'data.protectRanges': protectRangesLite,
    'data.whatif': openWhatIfDialog,
    'data.scenarios': openScenariosDialog,
    'data.consolidate': openConsolidateDialog,
    'help.about': () => {
      void EraOfficeShell.confirmAction({
        title: 'About ERA Tables',
        message:
          'ERA Tables — collaborative spreadsheets in your contour (not Excel).\n\n' +
          '• Grid is virtualized (capacity A–WW × 10 000); ↓+100 rows / →+cols scroll the view\n' +
          '• Charts are client SVG (bar/line/sparkline) stored on the sheet tab\n' +
          '• Subtotal writes SUM rows below the data block (not mid-table Excel SUBTOTAL)\n' +
          '• Relative formula rewrite after sort is not full Excel parity',
        okLabel: 'OK',
        cancelLabel: 'Close',
      });
    },
  });
}

if (filterInput) {
  filterInput.addEventListener('input', () => {
    filterText = filterInput.value || '';
    filterOpts = null;
    applyRowFilter();
  });
}

const filterOptsDlg = document.getElementById('filterOptsDlg');
if (filterOptsDlg) {
  filterOptsDlg.addEventListener('close', () => {
    const v = filterOptsDlg.returnValue;
    if (v === 'clear') {
      filterOpts = null;
      if (filterInput) filterInput.value = '';
      filterText = '';
      applyRowFilter();
      sendSheetOp({ type: 'set_filter_criteria', criteria: null });
      setAuthStatus('Filter cleared', false);
      return;
    }
    if (v !== 'apply') return;
    const col = parseInt(document.getElementById('filterOptsCol').value || '0', 10);
    const mode = document.getElementById('filterOptsMode').value || 'contains';
    const value = document.getElementById('filterOptsVal').value || '';
    filterOpts = { col, mode, value };
    const andColEl = document.getElementById('filterOptsAndCol');
    const andModeEl = document.getElementById('filterOptsAndMode');
    const andValEl = document.getElementById('filterOptsAndVal');
    if (andColEl && andModeEl && String(andValEl && andValEl.value || '').trim() !== '') {
      filterOpts.and = {
        col: parseInt(andColEl.value || '0', 10),
        mode: andModeEl.value || 'contains',
        value: andValEl.value || '',
      };
    }
    filterText = '';
    if (filterInput) filterInput.value = '';
    applyRowFilter();
    sendSheetOp({ type: 'set_filter_criteria', criteria: filterOpts });
    setAuthStatus('Filter options applied (col ' + colLetter(col) + ')', false);
  });
}

const whatIfDlg = document.getElementById('whatIfDlg');
if (whatIfDlg) {
  const previewBtn = document.getElementById('whatIfPreviewBtn');
  if (previewBtn) {
    previewBtn.addEventListener('click', (ev) => {
      ev.preventDefault();
      runWhatIfSeek(
        document.getElementById('whatIfFormula').value,
        document.getElementById('whatIfTarget').value,
        document.getElementById('whatIfChange').value,
        false
      );
    });
  }
  whatIfDlg.addEventListener('close', () => {
    if (whatIfDlg.returnValue !== 'run') return;
    runWhatIfSeek(
      document.getElementById('whatIfFormula').value,
      document.getElementById('whatIfTarget').value,
      document.getElementById('whatIfChange').value,
      true
    );
  });
}

const scenariosDlg = document.getElementById('scenariosDlg');
if (scenariosDlg) {
  scenariosDlg.addEventListener('close', () => {
    const v = scenariosDlg.returnValue;
    if (v === 'save') {
      const name =
        document.getElementById('scenariosName').value ||
        document.getElementById('scenariosList').value;
      saveActiveColumnScenario(name);
      return;
    }
    if (v === 'apply') {
      const name =
        document.getElementById('scenariosList').value ||
        document.getElementById('scenariosName').value;
      applyScenario(name);
    }
  });
}

const consolidateDlg = document.getElementById('consolidateDlg');
if (consolidateDlg) {
  consolidateDlg.addEventListener('close', () => {
    if (consolidateDlg.returnValue !== 'run') return;
    runConsolidate(
      document.getElementById('consolidateSheet').value,
      document.getElementById('consolidateRange').value,
      document.getElementById('consolidateTarget').value
    );
  });
}

document.getElementById('addSheetTabBtn').onclick = () => {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  const name = 'Sheet' + (sheetTabNames.length + 1);
  sendSheetOp({ type: 'add_sheet', name });
  setAuthStatus('Adding sheet ' + name + '…', false);
};

document.getElementById('addRowsBtn').onclick = () => addRows(100);
document.getElementById('addColsBtn').onclick = () => addCols(26);

formulaInput.addEventListener('keydown', (ev) => {
  if (ev.key === 'Enter') {
    ev.preventDefault();
    editing = false;
    commitCell(activeAddr, formulaInput.value || '');
    formulaInput.blur();
  }
});
if (activeAddrEl && activeAddrEl.tagName === 'INPUT') {
  activeAddrEl.addEventListener('keydown', (ev) => {
    if (ev.key === 'Enter') {
      ev.preventDefault();
      goToAddr(activeAddrEl.value);
      activeAddrEl.blur();
    }
  });
  activeAddrEl.addEventListener('focus', () => {
    activeAddrEl.select();
  });
}

document.getElementById('importBtn').onclick = () => document.getElementById('file').click();
const odsFileInput = document.getElementById('odsFile');
if (odsFileInput) {
  odsFileInput.onchange = async (e) => {
    const f = e.target.files && e.target.files[0];
    e.target.value = '';
    if (f) await importOdsFile(f).catch(() => {});
  };
}
document.getElementById('file').onchange = async (e) => {
  const f = e.target.files[0];
  if (!f) return;
  if (!localStorage.getItem('era_token')) {
    setAuthStatus('Sign in via Drive first (era_token).', true);
    return;
  }
  const { tenantId, userId } = identity();
  setAuthStatus('Importing xlsx…', false);
  const buf = await f.arrayBuffer();
  const b64 = arrayBufferToBase64(buf);
  const res = await officeFetch('/api/v1/tables/import', {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({
      tenant_id: tenantId,
      user_id: userId,
      name: f.name.replace(/\.xlsx$/i, '.erat'),
      xlsx_base64: b64,
    }),
  });
  if (!res.ok) {
    setAuthStatus('Import failed: ' + res.status, true);
    e.target.value = '';
    return;
  }
  const data = await res.json();
  if (!data.drive_object_id) {
    setAuthStatus('Import failed: no drive_object_id', true);
    return;
  }
  location.href = '/tables/' + data.drive_object_id;
};

document.getElementById('exportBtn').onclick = async () => {
  if (!sheetId) {
    setAuthStatus('Open a sheet first', true);
    return;
  }
  setAuthStatus('Exporting xlsx…', false);
  const res = await officeFetch('/api/v1/tables/' + encodeURIComponent(sheetId) + '/export/xlsx', {
    method: 'POST',
    headers: authHeaders(),
  });
  if (!res.ok) {
    setAuthStatus('Export failed: ' + res.status, true);
    return;
  }
  const blob = await res.blob();
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = (sheetId || 'export') + '.xlsx';
  a.click();
  URL.revokeObjectURL(a.href);
  setAuthStatus('Export ready', false);
};

document.addEventListener('keydown', (ev) => {
  if (!sheetId || editing) return;
  if (ev.target === formulaInput || ev.target === activeAddrEl) return;
  if (ev.target && ev.target.closest && ev.target.closest('dialog, input, textarea, select')) return;
  const nav = { ArrowUp: [0, -1], ArrowDown: [0, 1], ArrowLeft: [-1, 0], ArrowRight: [1, 0] };
  if (nav[ev.key] && !ev.ctrlKey && !ev.metaKey && !ev.altKey) {
    // When focus is already on a cell, grid keydown (onCellKey) handles it.
    if (ev.target && ev.target.closest && ev.target.closest('#grid td[data-addr]')) return;
    ev.preventDefault();
    moveActive(nav[ev.key][0], nav[ev.key][1], { extend: !!ev.shiftKey });
  }
});

refreshPresenceYou();
if (window.EraOfficeShell && EraOfficeShell.requireAuthOrRedirect) {
  if (!EraOfficeShell.requireAuthOrRedirect()) {
    /* redirecting */
  }
} else if (!localStorage.getItem('era_token')) {
  location.href = '/login?next=' + encodeURIComponent(location.pathname);
}

sheetId = pathSheetId();
const emptyHint = document.getElementById('emptyHint');
if (sheetId) {
  if (emptyHint) emptyHint.hidden = true;
  sheetMeta.textContent = 'Sheet: ' + sheetId;
  loadSheet(sheetId)
    .then(() => connectSync(sheetId))
    .catch(() => {});
} else if (localStorage.getItem('era_token')) {
  if (emptyHint) emptyHint.hidden = false;
  sheetMeta.textContent = 'Creating Untitled sheet…';
  updateSheetStats();
  createSheet().catch(() => {
    if (emptyHint) emptyHint.hidden = false;
    sheetMeta.textContent = 'No sheet open';
  });
} else {
  updateSheetStats();
}
