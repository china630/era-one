const blocksEl = document.getElementById('blocks');
const banner = document.getElementById('banner');
const authStatus = document.getElementById('authStatus');
const docMeta = document.getElementById('docMeta');
const saveStatus = document.getElementById('saveStatus');
const savePill = document.getElementById('savePill');
const docTitle = document.getElementById('docTitle');
const presenceYou = document.getElementById('presenceYou');
const presencePeers = document.getElementById('presencePeers');
const userChip = document.getElementById('userChip');

let docState = { blocks: [], comments: [] };
let docId = null;
let docName = 'Untitled';
let ws = null;
let syncTimer = null;
let autosaveTimer = null;
let activeBlockId = null;
let dirty = false;
let saving = false;
let presenceHeartbeatTimer = null;
let findMatches = [];
let findCursor = -1;
let lastFindQuery = '';
let undoStack = [];
let redoStack = [];
const UNDO_MAX = 40;
let peerCursors = {};
/** Marks applied to the next typed characters when selection is collapsed. */
let typingMarks = null;

if (window.EraOfficeShell) {
  if (EraOfficeShell.markActiveNav) EraOfficeShell.markActiveNav('docs');
  if (EraOfficeShell.mountNav) EraOfficeShell.mountNav(document);
  else if (EraOfficeShell.mountIcons) EraOfficeShell.mountIcons(document);
  if (EraOfficeShell.syncUserChip) EraOfficeShell.syncUserChip();
  if (EraOfficeShell.wireTeDisclaimer) {
    EraOfficeShell.wireTeDisclaimer(document.getElementById('teBanner'), 'era_te_dismiss_docs');
  }
}

function jwtPayload() {
  const token = localStorage.getItem('era_token') || '';
  if (!token) return null;
  try {
    const part = token.split('.')[1];
    const json = atob(part.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(json);
  } catch (_) {
    return null;
  }
}

function identity() {
  const p = jwtPayload() || {};
  return {
    tenantId: p.tenant_id || 't-demo',
    userId: p.sub || 'u-alice',
  };
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

function setDocMeta(text) {
  if (docMeta) docMeta.textContent = text;
}

function setSaveStatus(text, state) {
  if (saveStatus) saveStatus.textContent = text;
  if (!savePill) return;
  if (window.EraOfficeShell && EraOfficeShell.setSavePill && state) {
    EraOfficeShell.setSavePill(savePill, state, text);
    return;
  }
  savePill.textContent = text || '';
}

function setDocTitleUI(name, enabled) {
  docName = name || 'Untitled';
  if (!docTitle) return;
  docTitle.value = docName;
  docTitle.disabled = !enabled;
}

function refreshPresence() {
  const { userId } = identity();
  if (presenceYou) presenceYou.textContent = userId || 'You';
  if (presencePeers) presencePeers.textContent = 'Peers: —';
  // Never set userChip.textContent — it wipes the green status lamp.
  if (window.EraOfficeShell && EraOfficeShell.syncUserChip) EraOfficeShell.syncUserChip();
}

function peerColor(uid) {
  const colors = ['#c45', '#2a7', '#36c', '#a6a', '#e80', '#088'];
  return colors[Math.abs(hashStr(String(uid || ''))) % colors.length];
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
    chip.className = 'peer-chip';
    chip.style.cssText =
      'display:inline-block;padding:0 5px;border-radius:3px;font-size:0.85em;background:' +
      peerColor(uid) +
      ';color:#fff';
    chip.textContent = uid.length > 12 ? uid.slice(0, 10) + '…' : uid;
    presencePeers.appendChild(chip);
  });
}

function countWords() {
  const text = (docState.blocks || []).map((b) => blockText(b)).join(' ');
  if (!text.trim()) return 0;
  return text.trim().split(/\s+/).filter(Boolean).length;
}

function countChars() {
  return (docState.blocks || []).map((b) => blockText(b)).join('').length;
}

function estimatePages() {
  // ~250 words/page for memo estimate
  return Math.max(1, Math.ceil(countWords() / 250) || 0);
}

function updateWordCount() {
  const el = document.getElementById('wordCount');
  if (el) el.textContent = countWords() + ' words';
}

function openWordCountDialog() {
  const dlg = document.getElementById('wordCountDlg');
  const w = document.getElementById('wcWords');
  const c = document.getElementById('wcChars');
  const p = document.getElementById('wcPages');
  if (w) w.textContent = String(countWords());
  if (c) c.textContent = String(countChars());
  if (p) p.textContent = String(countWords() ? estimatePages() : 0);
  if (dlg && typeof dlg.showModal === 'function') dlg.showModal();
  else setAuthStatus(countWords() + ' words', false);
}

function syncRulerFromSelection() {
  const marker = document.getElementById('rulerIndent');
  const track = document.querySelector('#docRuler .era-doc-ruler-track');
  if (!marker) return;
  const sel = selectedBlock();
  const mm = (sel && sel.block && sel.block.indent_mm) || 0;
  const margins = (docState.page && docState.page.margins_mm) || 20;
  marker.style.left = mm + 'mm';
  marker.setAttribute('aria-valuenow', String(mm));
  if (track) {
    track.style.marginLeft = Math.min(40, margins) * 0.15 + 'rem';
    track.title = 'Margins ' + margins + 'mm · drag indent (first-line ≈ block indent)';
  }
}

function setIndentMm(mm) {
  const sel = selectedBlock();
  if (!sel) return;
  const next = Math.max(0, Math.min(80, Math.round(mm)));
  pushUndo();
  sel.block.indent_mm = next || undefined;
  sel.el.style.paddingLeft = next ? next + 'mm' : '';
  sendOp({ type: 'set_block_format', block_id: sel.block.id, indent_mm: next });
  scheduleAutosave();
  syncRulerFromSelection();
}

function initDocRuler() {
  const track = document.querySelector('#docRuler .era-doc-ruler-track');
  const marker = document.getElementById('rulerIndent');
  if (!track || !marker) return;
  let dragging = false;
  const pxToMm = (clientX) => {
    const rect = track.getBoundingClientRect();
    const x = Math.max(0, Math.min(rect.width, clientX - rect.left));
    // track ≈ 170mm content width on A4-ish page
    return (x / rect.width) * 80;
  };
  marker.addEventListener('pointerdown', (ev) => {
    dragging = true;
    marker.setPointerCapture(ev.pointerId);
    ev.preventDefault();
  });
  marker.addEventListener('pointermove', (ev) => {
    if (!dragging) return;
    setIndentMm(pxToMm(ev.clientX));
  });
  marker.addEventListener('pointerup', () => {
    dragging = false;
  });
  track.addEventListener('pointerdown', (ev) => {
    if (ev.target === marker) return;
    setIndentMm(pxToMm(ev.clientX));
  });
}

function openTableInsertDialog() {
  const dlg = document.getElementById('tableInsertDlg');
  if (dlg && typeof dlg.showModal === 'function') dlg.showModal();
  else insertTableBlock();
}

function insertTableFromDialog() {
  const rowsEl = document.getElementById('tableRows');
  const colsEl = document.getElementById('tableCols');
  const rows = Math.min(12, Math.max(1, parseInt((rowsEl && rowsEl.value) || '3', 10) || 3));
  const cols = Math.min(8, Math.max(1, parseInt((colsEl && colsEl.value) || '3', 10) || 3));
  const cells = [];
  const tsv = [];
  for (let r = 0; r < rows; r++) {
    const line = [];
    for (let c = 0; c < cols; c++) line.push(r === 0 ? 'H' + (c + 1) : '');
    cells.push(line.slice());
    tsv.push(line.join('\t'));
  }
  insertBlockAfter({
    id: newBlockId(),
    block_type: 'table',
    heading_level: 0,
    table_cells: cells,
    inlines: [{ text: tsv.join('\n'), bold: false, italic: false, underline: false }],
  });
  setAuthStatus('Table ' + rows + '×' + cols + ' inserted', false);
}

function tableCellsFromBlock(block) {
  if (block.table_cells && block.table_cells.length) return block.table_cells;
  return blockText(block)
    .split('\n')
    .filter(Boolean)
    .map((line) => line.split('\t'));
}

function syncTableTsv(block) {
  const cells = tableCellsFromBlock(block);
  block.table_cells = cells;
  block.inlines = [{ text: cells.map((r) => r.join('\t')).join('\n'), bold: false, italic: false, underline: false }];
}

/** Push full table TSV replace over WS so peers see edits (not only autosave snapshot). */
function broadcastTableTextReplace(blockId, prevText, nextText) {
  if (prevText === nextText) return;
  if (prevText && prevText.length) {
    sendOp({ type: 'delete_range', block_id: blockId, start: 0, end: prevText.length });
  }
  if (nextText) {
    sendOp({ type: 'insert_text', block_id: blockId, offset: 0, text: nextText });
  }
}

function refreshTableCellsFromInlines(block) {
  if (!block || block.block_type !== 'table') return;
  block.table_cells = blockText(block)
    .split('\n')
    .map((line) => line.split('\t'));
  while (block.table_cells.length && block.table_cells[block.table_cells.length - 1].length === 1 && block.table_cells[block.table_cells.length - 1][0] === '') {
    block.table_cells.pop();
  }
}

function insertTableRow(block) {
  const cells = tableCellsFromBlock(block);
  if (!cells.length) return;
  const cols = cells[0].length || 1;
  cells.push(Array(cols).fill(''));
  block.table_cells = cells;
  syncTableTsv(block);
}

function insertTableCol(block) {
  const cells = tableCellsFromBlock(block);
  cells.forEach((r) => r.push(''));
  block.table_cells = cells;
  syncTableTsv(block);
}

function deleteTableRow(block, rowIdx) {
  const cells = tableCellsFromBlock(block);
  if (cells.length <= 1) return false;
  const r = rowIdx == null ? cells.length - 1 : rowIdx;
  if (r < 0 || r >= cells.length) return false;
  cells.splice(r, 1);
  block.table_cells = cells;
  syncTableTsv(block);
  return true;
}

function deleteTableCol(block, colIdx) {
  const cells = tableCellsFromBlock(block);
  if (!cells.length || (cells[0] && cells[0].length <= 1)) return false;
  const c = colIdx == null ? cells[0].length - 1 : colIdx;
  cells.forEach((row) => {
    if (c >= 0 && c < row.length) row.splice(c, 1);
  });
  block.table_cells = cells;
  syncTableTsv(block);
  return true;
}

function clearFindHighlight() {
  if (!blocksEl) return;
  blocksEl.querySelectorAll('.doc-block.find-highlight').forEach((el) => {
    el.classList.remove('find-highlight');
  });
}

function collectFindMatches(query) {
  if (!query) return [];
  const q = query.toLowerCase();
  const matches = [];
  for (const block of docState.blocks || []) {
    if (blockText(block).toLowerCase().includes(q)) {
      matches.push(block.id);
    }
  }
  return matches;
}

function findNext() {
  const input = document.getElementById('findInput');
  const query = (input && input.value ? input.value : '').trim();
  clearFindHighlight();
  if (!query) {
    findMatches = [];
    findCursor = -1;
    lastFindQuery = '';
    return;
  }
  if (query !== lastFindQuery) {
    findMatches = collectFindMatches(query);
    findCursor = -1;
    lastFindQuery = query;
  }
  if (!findMatches.length) {
    setAuthStatus('No matches for “' + query + '”', true);
    return;
  }
  findCursor = (findCursor + 1) % findMatches.length;
  const blockId = findMatches[findCursor];
  const el = blocksEl.querySelector('.doc-block[data-block-id="' + blockId + '"]');
  if (el) {
    el.classList.add('find-highlight');
    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    setAuthStatus('Match ' + (findCursor + 1) + ' of ' + findMatches.length, false);
  }
}

function scheduleAutosave() {
  if (!docId) return;
  dirty = true;
  setSaveStatus('Unsaved changes', 'dirty');
  clearTimeout(autosaveTimer);
  autosaveTimer = setTimeout(() => {
    autosaveSnapshot().catch(() => {});
  }, 2500);
}

async function autosaveSnapshot() {
  if (!docId || !dirty || saving) return;
  saving = true;
  setSaveStatus('Saving…', 'saving');
  const ok = await snapshotDoc();
  saving = false;
  if (ok) {
    dirty = false;
    setSaveStatus('Saved · ' + new Date().toLocaleTimeString(), 'ok');
  } else {
    setSaveStatus('Autosave failed — try Save now', 'err');
  }
}

async function loadDriveName(id) {
  try {
    const res = await officeFetch('/api/v1/drive/objects/' + encodeURIComponent(id) + '/meta', {
      headers: authHeaders(),
    });
    if (!res.ok) return;
    const data = await res.json();
    const name = data.name || data.Name || 'Untitled';
    setDocTitleUI(name, true);
    setDocMeta('Document: ' + name);
  } catch (_) {}
}

async function renameDriveObject(name) {
  if (!docId || !name || name === docName) return;
  const res = await officeFetch('/api/v1/drive/objects/' + encodeURIComponent(docId), {
    method: 'PATCH',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({ name }),
  });
  if (!res.ok) {
    setAuthStatus('Rename failed: ' + res.status, true);
    setDocTitleUI(docName, true);
    return;
  }
  setDocTitleUI(name, true);
  setDocMeta('Document: ' + name);
  setAuthStatus('Renamed', false);
}

function openShareDialog() {
  const dlg = document.getElementById('shareDlg');
  const input = document.getElementById('shareLinkInput');
  const driveLink = document.getElementById('shareDriveLink');
  const hint = document.getElementById('shareAclHint');
  if (!docId) {
    location.href = '/drive/';
    return;
  }
  const url = location.origin + '/docs/' + encodeURIComponent(docId);
  if (input) input.value = url;
  if (hint) {
    hint.textContent =
      'Copy link · ACL is managed in Drive (Share on the Drive object). This dialog does not change permissions.';
  }
  if (driveLink) {
    driveLink.href = '/drive/?share=' + encodeURIComponent(docId);
    driveLink.textContent = 'Manage ACL in Drive';
  }
  if (dlg && typeof dlg.showModal === 'function') {
    dlg.showModal();
    return;
  }
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(url).then(
      () => setAuthStatus('Share link copied', false),
      () => EraOfficeShell.promptCopy({ title: 'Share document', value: url })
    );
  } else {
    void EraOfficeShell.promptCopy({ title: 'Share document', value: url });
  }
}

function authHeaders(extra) {
  const headers = Object.assign({}, extra || {});
  const token = localStorage.getItem('era_token') || '';
  if (token) headers.Authorization = 'Bearer ' + token;
  const { userId, tenantId } = identity();
  headers['X-ERA-User'] = userId;
  headers['X-ERA-Tenant'] = tenantId;
  return headers;
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

function pathDocId() {
  const parts = location.pathname.replace(/\/$/, '').split('/');
  const id = parts[parts.length - 1];
  return id && id !== 'docs' ? id : null;
}

function blockText(block) {
  return (block.inlines || []).map((i) => i.text || '').join('');
}

function blockMarks(block) {
  const spans = block.inlines || [];
  if (!spans.length) {
    return {
      bold: false,
      italic: false,
      underline: false,
      strike: false,
      superscript: false,
      subscript: false,
      link_url: '',
      font_family: '',
      font_size_pt: null,
      color: '',
      highlight: '',
    };
  }
  return {
    bold: spans.every((s) => !!s.bold),
    italic: spans.every((s) => !!s.italic),
    underline: spans.every((s) => !!s.underline),
    strike: spans.every((s) => !!s.strike),
    superscript: spans.every((s) => !!s.superscript),
    subscript: spans.every((s) => !!s.subscript),
    link_url: (spans[0] && spans[0].link_url) || '',
    font_family: (spans[0] && spans[0].font_family) || '',
    font_size_pt: spans[0] && spans[0].font_size_pt != null ? spans[0].font_size_pt : null,
    color: (spans[0] && spans[0].color) || '',
    highlight: (spans[0] && spans[0].highlight) || '',
  };
}

function spanFromMarks(text, marks, patch) {
  const m = Object.assign({}, marks || {}, patch || {});
  return {
    text: text || '',
    bold: !!m.bold,
    italic: !!m.italic,
    underline: !!m.underline,
    strike: !!m.strike,
    superscript: !!m.superscript,
    subscript: !!m.subscript,
    link_url: m.link_url || undefined,
    font_family: m.font_family || undefined,
    font_size_pt: m.font_size_pt || undefined,
    color: m.color || undefined,
    highlight: m.highlight || undefined,
  };
}

function marksEqual(a, b) {
  if (!a || !b) return false;
  return (
    !!a.bold === !!b.bold &&
    !!a.italic === !!b.italic &&
    !!a.underline === !!b.underline &&
    !!a.strike === !!b.strike &&
    !!a.superscript === !!b.superscript &&
    !!a.subscript === !!b.subscript &&
    (a.link_url || '') === (b.link_url || '') &&
    (a.font_family || '') === (b.font_family || '') &&
    (a.font_size_pt || 0) === (b.font_size_pt || 0) &&
    (a.color || '') === (b.color || '') &&
    (a.highlight || '') === (b.highlight || '')
  );
}

function withText(marks, text) {
  return spanFromMarks(text, marks);
}

function coalesceInlines(inlines) {
  const out = [];
  for (const span of inlines || []) {
    if (!span || !span.text) continue;
    const last = out[out.length - 1];
    if (last && marksEqual(last, span)) last.text += span.text;
    else out.push(Object.assign({}, span));
  }
  if (!out.length) out.push(spanFromMarks('', {}));
  return out;
}

/** Split so offset is on a span boundary; returns index of span starting at offset. */
function splitInlinesAt(inlines, offset) {
  const list = (inlines || []).map((s) => Object.assign({}, s, { text: s.text || '' }));
  const plain = list.map((s) => s.text).join('');
  const off = Math.max(0, Math.min(offset, plain.length));
  if (off === 0) return { left: [], right: list };
  if (off >= plain.length) return { left: list, right: [] };
  let acc = 0;
  for (let i = 0; i < list.length; i++) {
    const len = list[i].text.length;
    if (acc + len === off) {
      return { left: coalesceInlines(list.slice(0, i + 1)), right: coalesceInlines(list.slice(i + 1)) };
    }
    if (acc + len > off) {
      const local = off - acc;
      const leftSpan = withText(list[i], list[i].text.slice(0, local));
      const rightSpan = withText(list[i], list[i].text.slice(local));
      const left = list.slice(0, i).concat(leftSpan.text ? [leftSpan] : []);
      const right = (rightSpan.text ? [rightSpan] : []).concat(list.slice(i + 1));
      return { left: coalesceInlines(left), right: coalesceInlines(right) };
    }
    acc += len;
  }
  return { left: list, right: [] };
}

function insertTextPreserving(inlines, offset, text, typing) {
  if (!text) return coalesceInlines(inlines);
  const { left, right } = splitInlinesAt(inlines, offset);
  const template =
    typing ||
    (left.length ? left[left.length - 1] : null) ||
    (right.length ? right[0] : null) ||
    {};
  const mid = withText(template, text);
  return coalesceInlines(left.concat([mid], right));
}

function deleteRangePreserving(inlines, start, end) {
  if (start >= end) return coalesceInlines(inlines);
  const a = splitInlinesAt(inlines, start);
  const b = splitInlinesAt(a.right, end - start);
  return coalesceInlines(a.left.concat(b.right));
}

function applyMarksRangeLocal(inlines, start, end, patch) {
  if (start >= end) return coalesceInlines(inlines);
  const head = splitInlinesAt(inlines, start);
  const midRight = splitInlinesAt(head.right, end - start);
  const mid = midRight.left.map((s) => spanFromMarks(s.text, s, patch));
  return coalesceInlines(head.left.concat(mid, midRight.right));
}

function diffText(prev, next) {
  let start = 0;
  const minLen = Math.min(prev.length, next.length);
  while (start < minLen && prev.charAt(start) === next.charAt(start)) start++;
  let endPrev = prev.length;
  let endNext = next.length;
  while (endPrev > start && endNext > start && prev.charAt(endPrev - 1) === next.charAt(endNext - 1)) {
    endPrev--;
    endNext--;
  }
  return { start, deletedEnd: endPrev, inserted: next.slice(start, endNext) };
}

/** Caret / selection offsets within a .doc-block (UTF-16 code units ≈ JS string). */
function selectionOffsetsInBlock(blockEl) {
  const sel = window.getSelection();
  if (!sel || !blockEl) return null;
  const root = blockEl;
  function offsetFromPoint(node, nodeOffset) {
    const range = document.createRange();
    range.selectNodeContents(root);
    range.setEnd(node, nodeOffset);
    return range.toString().length;
  }
  try {
    if (!root.contains(sel.anchorNode)) return null;
    const a = offsetFromPoint(sel.anchorNode, sel.anchorOffset);
    const f = root.contains(sel.focusNode)
      ? offsetFromPoint(sel.focusNode, sel.focusOffset)
      : a;
    return { start: Math.min(a, f), end: Math.max(a, f) };
  } catch (_) {
    return null;
  }
}

function pointInBlock(blockEl, offset) {
  const walker = document.createTreeWalker(blockEl, NodeFilter.SHOW_TEXT, null);
  let remaining = Math.max(0, offset || 0);
  let node = walker.nextNode();
  let last = null;
  while (node) {
    last = node;
    if (remaining <= node.textContent.length) {
      return { node, offset: remaining };
    }
    remaining -= node.textContent.length;
    node = walker.nextNode();
  }
  if (last) return { node: last, offset: last.textContent.length };
  return { node: blockEl, offset: 0 };
}

function focusBlockAt(blockId, offset) {
  selectRangeInBlock(blockId, offset, offset);
}

/** Restore a non-collapsed selection after re-render / mark apply. */
function selectRangeInBlock(blockId, start, end) {
  const el = blocksEl.querySelector('.doc-block[data-block-id="' + blockId + '"]');
  if (!el) return;
  el.focus();
  const sel = window.getSelection();
  if (!sel) return;
  const a = Math.max(0, Math.min(start, end));
  const b = Math.max(0, Math.max(start, end));
  const p0 = pointInBlock(el, a);
  const p1 = pointInBlock(el, b);
  try {
    const range = document.createRange();
    range.setStart(p0.node, p0.offset);
    range.setEnd(p1.node, p1.offset);
    sel.removeAllRanges();
    sel.addRange(range);
  } catch (_) {
    focusBlockAt(blockId, b);
  }
}

function closestDocBlock(node) {
  if (!node) return null;
  const el = node.nodeType === 1 ? node : node.parentElement;
  return el && el.closest ? el.closest('.doc-block') : null;
}

/**
 * Selection targets for formatting. Supports multi-paragraph ranges
 * when the editing surface allows cross-block DOM selection.
 */
function collectFormatTargets() {
  const sel = window.getSelection();
  if (!sel || !sel.rangeCount || !blocksEl) {
    const one = selectedBlock();
    if (!one) return [];
    const offs = selectionOrTypingRange(one);
    if (!offs) return [{ block: one.block, el: one.el, start: 0, end: 0, collapsed: true }];
    return [{ block: one.block, el: one.el, start: offs.start, end: offs.end, collapsed: false }];
  }
  const anchorEl = closestDocBlock(sel.anchorNode);
  const focusEl = closestDocBlock(sel.focusNode);
  if (!anchorEl || !focusEl) {
    const one = selectedBlock();
    return one ? [{ block: one.block, el: one.el, start: 0, end: 0, collapsed: true }] : [];
  }
  const blocks = Array.from(blocksEl.querySelectorAll('.doc-block')).filter(
    (el) => el.dataset.type !== 'page_break' && el.dataset.type !== 'section_break'
  );
  const ai = blocks.indexOf(anchorEl);
  const fi = blocks.indexOf(focusEl);
  if (ai < 0 || fi < 0) return [];
  if (ai === fi) {
    const block = docState.blocks.find((b) => b.id === anchorEl.dataset.blockId);
    if (!block) return [];
    const offs = selectionOffsetsInBlock(anchorEl);
    if (offs && offs.end > offs.start) {
      return [{ block, el: anchorEl, start: offs.start, end: offs.end, collapsed: false }];
    }
    return [{ block, el: anchorEl, start: 0, end: 0, collapsed: true }];
  }
  const from = Math.min(ai, fi);
  const to = Math.max(ai, fi);
  const anchorIsStart = ai === from;
  const startOffs = selectionOffsetsInBlock(blocks[from]) || { start: 0, end: 0 };
  const endOffs = selectionOffsetsInBlock(blocks[to]) || { start: 0, end: 0 };
  // In a multi-block selection, offsets in each edge block are relative to that block.
  // Use the smaller offset on the start block and larger on the end block from the live range.
  let edgeStart = startOffs.start;
  let edgeEnd = endOffs.end;
  try {
    const live = sel.getRangeAt(0);
    const r0 = document.createRange();
    r0.selectNodeContents(blocks[from]);
    r0.setEnd(live.startContainer, live.startOffset);
    edgeStart = r0.toString().length;
    const r1 = document.createRange();
    r1.selectNodeContents(blocks[to]);
    r1.setEnd(live.endContainer, live.endOffset);
    edgeEnd = r1.toString().length;
  } catch (_) {
    edgeStart = anchorIsStart ? startOffs.start : startOffs.end;
    edgeEnd = anchorIsStart ? endOffs.end : endOffs.start;
  }
  const out = [];
  for (let i = from; i <= to; i++) {
    const el = blocks[i];
    const block = docState.blocks.find((b) => b.id === el.dataset.blockId);
    if (!block) continue;
    const len = blockText(block).length;
    let start = 0;
    let end = len;
    if (i === from) start = Math.max(0, Math.min(len, edgeStart));
    if (i === to) end = Math.max(0, Math.min(len, edgeEnd));
    if (i === from && i === to && end <= start) {
      out.push({ block, el, start: 0, end: 0, collapsed: true });
    } else {
      out.push({ block, el, start, end, collapsed: end <= start });
    }
  }
  return out;
}

function restoreMultiSelection(targets) {
  if (!targets || !targets.length) return;
  if (targets.length === 1) {
    const t = targets[0];
    if (!t.collapsed && t.end > t.start) selectRangeInBlock(t.block.id, t.start, t.end);
    else focusBlockAt(t.block.id, t.end || t.start || 0);
    return;
  }
  const first = targets[0];
  const last = targets[targets.length - 1];
  const p0 = pointInBlock(first.el, first.start);
  const p1 = pointInBlock(last.el, last.end);
  const sel = window.getSelection();
  if (!sel) return;
  try {
    const range = document.createRange();
    range.setStart(p0.node, p0.offset);
    range.setEnd(p1.node, p1.offset);
    sel.removeAllRanges();
    sel.addRange(range);
  } catch (_) {
    selectRangeInBlock(last.block.id, last.start, last.end);
  }
}

function wireBlocksSurface() {
  if (!blocksEl || blocksEl._eraSurfaceWired) return;
  blocksEl._eraSurfaceWired = true;
  // Single editing host so the browser can select across consecutive paragraphs.
  blocksEl.setAttribute('contenteditable', 'true');
  blocksEl.addEventListener('focusin', (e) => {
    const el = closestDocBlock(e.target);
    if (el && el.dataset.blockId) {
      activeBlockId = el.dataset.blockId;
      broadcastCaret(el.dataset.blockId, 0);
      syncRulerFromSelection();
    }
  });
  // Click [fnN] in body → jump to footnote block
  blocksEl.addEventListener('click', (e) => {
    const blockEl = closestDocBlock(e.target);
    if (!blockEl || blockEl.dataset.type === 'footnote') return;
    const sel = window.getSelection();
    if (!sel || !sel.rangeCount) return;
    try {
      const r = sel.getRangeAt(0);
      const probe = document.createRange();
      probe.selectNodeContents(blockEl);
      const full = blockEl.textContent || '';
      const m = full.match(/\[fn\d+\]/i);
      if (!m) return;
      // Only jump if click landed near a footnote marker (caret context).
      const off = (() => {
        const rr = document.createRange();
        rr.selectNodeContents(blockEl);
        rr.setEnd(r.startContainer, r.startOffset);
        return rr.toString().length;
      })();
      const re = /\[fn\d+\]/gi;
      let hit = null;
      let mm;
      while ((mm = re.exec(full))) {
        if (off >= mm.index && off <= mm.index + mm[0].length) {
          hit = mm[0].slice(1, -1);
          break;
        }
      }
      if (!hit) return;
      const fn = blocksEl.querySelector('.doc-block[data-type="footnote"] .doc-fn-mark[data-fn="' + hit + '"]');
      const host = fn ? fn.closest('.doc-block') : null;
      if (host) {
        host.scrollIntoView({ block: 'center' });
        host.classList.add('find-highlight');
        setTimeout(() => host.classList.remove('find-highlight'), 1200);
      }
    } catch (_) {}
  });
}

function newBlockId() {
  return crypto.randomUUID ? crypto.randomUUID() : 'b-' + Date.now() + '-' + Math.random().toString(16).slice(2);
}

function copyBlockShell(block, id, inlines) {
  return {
    id,
    block_type: block.block_type || 'paragraph',
    heading_level: block.heading_level || 0,
    list_type: block.list_type || undefined,
    align: block.align || 'left',
    line_spacing: block.line_spacing,
    indent_mm: block.indent_mm,
    space_before_pt: block.space_before_pt,
    space_after_pt: block.space_after_pt,
    list_level: block.list_level || 0,
    list_marker: block.list_marker,
    list_restart: false,
    style_name: block.style_name,
    lang: block.lang,
    inlines: coalesceInlines(inlines),
  };
}

function pushUndo() {
  try {
    undoStack.push(JSON.stringify(docState));
    if (undoStack.length > UNDO_MAX) undoStack.shift();
    redoStack = [];
  } catch (_) {}
}

function restoreSnapshot(json) {
  try {
    applyDoc(JSON.parse(json));
    scheduleAutosave();
  } catch (_) {}
}

function undoEdit() {
  if (!undoStack.length) {
    setAuthStatus('Nothing to undo', true);
    return;
  }
  try {
    redoStack.push(JSON.stringify(docState));
  } catch (_) {}
  restoreSnapshot(undoStack.pop());
  setAuthStatus('Undo', false);
}

function redoEdit() {
  if (!redoStack.length) {
    setAuthStatus('Nothing to redo', true);
    return;
  }
  try {
    undoStack.push(JSON.stringify(docState));
  } catch (_) {}
  restoreSnapshot(redoStack.pop());
  setAuthStatus('Redo', false);
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function renderInlineHtml(block) {
  const spans = block.inlines || [];
  if (!spans.length) return '';
  return spans
    .map((span) => {
      let html = escapeHtml(span.text || '').replace(/\n/g, '<br>');
      if (span.bold) html = '<b>' + html + '</b>';
      if (span.italic) html = '<i>' + html + '</i>';
      if (span.underline) html = '<u>' + html + '</u>';
      if (span.strike) html = '<s>' + html + '</s>';
      if (span.superscript) html = '<sup>' + html + '</sup>';
      if (span.subscript) html = '<sub>' + html + '</sub>';
      const style = [];
      if (span.font_family) style.push('font-family:' + span.font_family);
      if (span.font_size_pt) style.push('font-size:' + span.font_size_pt + 'pt');
      if (span.color) style.push('color:' + span.color);
      if (span.highlight) style.push('background:' + span.highlight);
      if (style.length) html = '<span style="' + style.join(';') + '">' + html + '</span>';
      if (span.link_url) {
        html =
          '<a href="' +
          escapeHtml(span.link_url) +
          '" target="_blank" rel="noopener noreferrer">' +
          html +
          '</a>';
      }
      return html;
    })
    .join('');
}

function renderComments() {
  const ul = document.getElementById('commentsList');
  if (!ul) return;
  ul.innerHTML = '';
  const comments = docState.comments || [];
  if (!comments.length) {
    ul.innerHTML = '<li class="era-hint">No comments yet</li>';
    return;
  }
  for (const c of comments) {
    const li = document.createElement('li');
    li.style.borderBottom = '1px solid var(--era-line)';
    li.style.padding = '0.35rem 0';
    const open = !c.resolved;
    const quote = c.quote
      ? '<div class="era-hint" style="font-style:italic">“' + escapeHtml(c.quote) + '”</div>'
      : '';
    li.innerHTML =
      '<div><strong>' +
      escapeHtml(c.author_id || 'user') +
      '</strong>' +
      (open ? '' : ' · resolved') +
      '</div>' +
      quote +
      '<div>' +
      escapeHtml(c.text || '') +
      '</div>';
    if (open) {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'era-btn';
      btn.textContent = 'Resolve';
      btn.style.marginTop = '0.25rem';
      btn.addEventListener('click', () => resolveComment(c.id));
      li.appendChild(btn);
    }
    li.addEventListener('click', (ev) => {
      if (ev.target.tagName === 'BUTTON') return;
      const el = blocksEl && blocksEl.querySelector('.doc-block[data-block-id="' + c.block_id + '"]');
      if (el) {
        el.scrollIntoView({ block: 'center' });
        el.classList.add('find-highlight');
        setTimeout(() => el.classList.remove('find-highlight'), 1200);
      }
    });
    ul.appendChild(li);
  }
}

function applyPageChrome() {
  if (!docState.page) docState.page = { size: 'a4', orientation: 'portrait', margins_mm: 20 };
  const page = docState.page || {};
  if (!page.size) page.size = 'a4';
  const mm = page.margins_mm != null ? page.margins_mm : 20;
  if (blocksEl) {
    blocksEl.style.padding = mm + 'mm';
    blocksEl.classList.toggle('landscape', page.orientation === 'landscape');
    blocksEl.dataset.pageSize = page.size || 'a4';
    // Letter ≈ 816×1056; A4 ≈ 794×1123 @ 96dpi
    const a4 = !page.size || page.size === 'a4';
    const landscape = page.orientation === 'landscape';
    let w = a4 ? 794 : 816;
    let h = a4 ? 1123 : 1056;
    if (landscape) {
      const t = w;
      w = h;
      h = t;
    }
    blocksEl.style.width = w + 'px';
    blocksEl.style.maxWidth = w + 'px';
    blocksEl.style.minHeight = h + 'px';
    const hRuler = document.getElementById('docRuler');
    const vRuler = document.getElementById('docRulerV');
    if (hRuler) hRuler.style.width = w + 'px';
    if (vRuler) vRuler.style.minHeight = h + 'px';
    const cols = page.columns != null ? Number(page.columns) : 1;
    blocksEl.classList.toggle('columns-2', cols === 2);
    blocksEl.classList.toggle('columns-3', cols >= 3);
  }
  const strip = document.getElementById('headerStrip');
  const hi = document.getElementById('headerInput');
  const fi = document.getElementById('footerInput');
  const pn = document.getElementById('pageNumbersChk');
  if (hi && docState.header) hi.value = docState.header.text || '';
  if (fi && docState.footer) fi.value = docState.footer.text || '';
  if (pn) {
    pn.checked = !!(
      (docState.header && docState.header.page_numbers) ||
      (docState.footer && docState.footer.page_numbers)
    );
  }
  if (strip && docState.header && (docState.header.text || (docState.footer && docState.footer.text))) {
    strip.hidden = false;
  }
}

function pendingRevisionFor(blockId) {
  const revs = docState.revisions || [];
  for (let i = revs.length - 1; i >= 0; i--) {
    if (revs[i].block_id === blockId) return revs[i];
  }
  return null;
}

function renderBlocks() {
  wireBlocksSurface();
  blocksEl.innerHTML = '';
  applyPageChrome();
  document.body.classList.toggle('era-suggesting', !!docState.track_changes);
  let blockNum = 0;
  for (const block of docState.blocks) {
    const type = block.block_type || 'paragraph';
    if (type === 'page_break' || type === 'section_break') {
      const br = document.createElement('div');
      br.className = 'doc-block' + (type === 'page_break' ? ' doc-page-break' : '');
      br.dataset.blockId = block.id;
      br.dataset.type = type;
      blockNum += 1;
      br.dataset.line = String(blockNum);
      br.contentEditable = 'false';
      br.setAttribute('contenteditable', 'false');
      br.textContent = type === 'section_break' ? '— Section break —' : '— Page break —';
      if (type === 'page_break') {
        br.style.textAlign = 'center';
        br.style.color = 'var(--era-muted)';
        br.style.borderTop = '1px dashed var(--era-line)';
        br.style.margin = '1rem 0';
        br.style.padding = '0.25rem';
      }
      blocksEl.appendChild(br);
      continue;
    }
    const el = document.createElement('div');
    el.className = 'doc-block';
    el.dataset.blockId = block.id;
    blockNum += 1;
    el.dataset.line = String(blockNum);
    if (block.list_restart) el.dataset.listRestart = '1';
    if (block.list_marker) el.dataset.listMarker = block.list_marker;
    const pending = pendingRevisionFor(block.id);
    if (pending) el.classList.add('doc-block-suggested');
    const special =
      type === 'image' ||
      type === 'table' ||
      type === 'bookmark' ||
      type === 'toc' ||
      type === 'footnote';
    // Text blocks inherit the surface contenteditable; special nodes stay inert.
    if (special) {
      el.contentEditable = 'false';
      el.setAttribute('contenteditable', 'false');
    } else {
      el.removeAttribute('contenteditable');
    }
    el.dataset.type =
      type === 'heading'
        ? 'heading'
        : type === 'list_item'
          ? 'list_item'
          : type === 'text_box'
            ? 'text_box'
            : special
              ? type
              : 'paragraph';
    if (type === 'text_box') {
      el.classList.add('doc-textbox');
      const wp = block.box_width_pct != null ? Number(block.box_width_pct) : 100;
      el.style.width = Math.min(100, Math.max(40, wp)) + '%';
      el.style.maxWidth = '100%';
      el.style.boxSizing = 'border-box';
    }
    if (type === 'heading') {
      el.dataset.level = String(block.heading_level || 1);
    } else {
      delete el.dataset.level;
    }
    if (block.list_type === 'ordered') el.dataset.list = 'ordered';
    if (block.align) el.style.textAlign = block.align === 'justify' ? 'justify' : block.align;
    if (block.indent_mm) el.style.paddingLeft = block.indent_mm + 'mm';
    if (block.style_name === 'title') el.style.fontSize = '1.75rem';
    if (type === 'image' && block.image_url) {
      const alt = escapeHtml(blockText(block) || 'image');
      const img = document.createElement('img');
      img.alt = blockText(block) || 'image';
      img.style.maxWidth = '100%';
      img.style.height = 'auto';
      el.appendChild(img);
      const src = block.image_url;
      if (src.startsWith('/api/v1/drive/objects/')) {
        officeFetch(src, { headers: authHeaders() })
          .then((r) => (r.ok ? r.blob() : Promise.reject(r.status)))
          .then((blob) => {
            img.src = URL.createObjectURL(blob);
          })
          .catch(() => {
            el.appendChild(document.createTextNode('[image load failed] ' + alt));
          });
      } else {
        img.src = src;
      }
    } else if (type === 'table') {
      const rows = tableCellsFromBlock(block);
      let html = '<table class="era-doc-table"><tbody>';
      rows.forEach((r, ri) => {
        html += '<tr>';
        r.forEach((c, ci) => {
          html +=
            '<td contenteditable="true" data-r="' +
            ri +
            '" data-c="' +
            ci +
            '">' +
            escapeHtml(c) +
            '</td>';
        });
        html += '</tr>';
      });
      html += '</tbody></table>';
      html +=
        '<div class="era-table-ops">' +
        '<button type="button" class="era-btn era-tbl-row">+ Row</button> ' +
        '<button type="button" class="era-btn era-tbl-col">+ Col</button> ' +
        '<button type="button" class="era-btn era-tbl-del-row">− Row</button> ' +
        '<button type="button" class="era-btn era-tbl-del-col">− Col</button>' +
        '</div>';
      el.innerHTML = html;
      el.querySelectorAll('td[contenteditable]').forEach((td) => {
        td.addEventListener('blur', () => {
          const prev = blockText(block);
          const cells = tableCellsFromBlock(block).map((row) => row.slice());
          const r = parseInt(td.dataset.r, 10);
          const c = parseInt(td.dataset.c, 10);
          if (!cells[r]) return;
          const nextVal = td.textContent || '';
          if ((cells[r][c] || '') === nextVal) return;
          pushUndo();
          cells[r][c] = nextVal;
          block.table_cells = cells;
          syncTableTsv(block);
          broadcastTableTextReplace(block.id, prev, blockText(block));
          scheduleAutosave();
        });
      });
      const rowBtn = el.querySelector('.era-tbl-row');
      const colBtn = el.querySelector('.era-tbl-col');
      const delRowBtn = el.querySelector('.era-tbl-del-row');
      const delColBtn = el.querySelector('.era-tbl-del-col');
      if (rowBtn)
        rowBtn.addEventListener('click', () => {
          const prev = blockText(block);
          pushUndo();
          insertTableRow(block);
          broadcastTableTextReplace(block.id, prev, blockText(block));
          renderBlocks();
          scheduleAutosave();
        });
      if (colBtn)
        colBtn.addEventListener('click', () => {
          const prev = blockText(block);
          pushUndo();
          insertTableCol(block);
          broadcastTableTextReplace(block.id, prev, blockText(block));
          renderBlocks();
          scheduleAutosave();
        });
      if (delRowBtn)
        delRowBtn.addEventListener('click', () => {
          const prev = blockText(block);
          pushUndo();
          if (!deleteTableRow(block)) {
            setAuthStatus('Table needs at least one row', true);
            return;
          }
          broadcastTableTextReplace(block.id, prev, blockText(block));
          renderBlocks();
          scheduleAutosave();
        });
      if (delColBtn)
        delColBtn.addEventListener('click', () => {
          const prev = blockText(block);
          pushUndo();
          if (!deleteTableCol(block)) {
            setAuthStatus('Table needs at least one column', true);
            return;
          }
          broadcastTableTextReplace(block.id, prev, blockText(block));
          renderBlocks();
          scheduleAutosave();
        });
    } else if (type === 'bookmark') {
      el.innerHTML =
        '<a href="#bm-' +
        encodeURIComponent(block.bookmark_name || block.id) +
        '" class="doc-bookmark" id="bm-' +
        escapeHtml(block.bookmark_name || block.id) +
        '">' +
        escapeHtml(blockText(block) || block.bookmark_name || 'bookmark') +
        '</a>';
    } else if (type === 'toc') {
      const wrap = document.createElement('div');
      wrap.className = 'doc-toc';
      const pre = document.createElement('pre');
      pre.style.cssText = 'margin:0;white-space:pre-wrap;font:inherit';
      pre.textContent = blockText(block);
      wrap.appendChild(pre);
      const refresh = document.createElement('button');
      refresh.type = 'button';
      refresh.className = 'era-btn';
      refresh.textContent = 'Refresh TOC';
      refresh.addEventListener('click', () => refreshTocBlock(block));
      wrap.appendChild(refresh);
      el.appendChild(wrap);
      renderTocLinks(el, block);
    } else if (block.style_name === 'horizontal_line') {
      el.contentEditable = 'false';
      el.setAttribute('contenteditable', 'false');
      el.classList.add('doc-hr');
      el.innerHTML = '<hr/>';
    } else if (type === 'footnote') {
      el.innerHTML =
        '<sup class="doc-fn-mark" data-fn="' +
        escapeHtml(block.bookmark_name || 'fn') +
        '" title="Jump to reference">[' +
        escapeHtml(block.bookmark_name || 'fn') +
        ']</sup> ' +
        escapeHtml(blockText(block));
      el.title = 'Footnote ' + (block.bookmark_name || '');
      el.addEventListener('click', (ev) => {
        if (!(ev.target.closest && ev.target.closest('.doc-fn-mark'))) return;
        const mark = '[' + (block.bookmark_name || '') + ']';
        const host = Array.from(blocksEl.querySelectorAll('.doc-block')).find((b) => {
          if (b.dataset.type === 'footnote') return false;
          return (b.textContent || '').indexOf(mark) >= 0;
        });
        if (host) {
          host.scrollIntoView({ block: 'center' });
          host.classList.add('find-highlight');
          setTimeout(() => host.classList.remove('find-highlight'), 1200);
        }
      });
    } else if (pending && docState.track_changes) {
      el.innerHTML =
        '<span class="rev-del">' +
        escapeHtml(pending.before || '') +
        '</span> <span class="rev-ins">' +
        escapeHtml(pending.after || blockText(block)) +
        '</span>';
    } else {
      el.innerHTML = renderInlineHtml(block);
    }
    if (pending) {
      const badge = document.createElement('span');
      badge.className = 'doc-suggest-badge';
      badge.title = 'Pending suggestion — Tools → Review';
      badge.textContent = 'suggested';
      el.appendChild(badge);
    }
    blocksEl.appendChild(el);
  }
  if (!blocksEl._eraInputWired) {
    blocksEl._eraInputWired = true;
    blocksEl.addEventListener('input', () => {
      const el = closestDocBlock(window.getSelection() && window.getSelection().anchorNode);
      if (!el || el.getAttribute('contenteditable') === 'false') return;
      onBlockInput(el.dataset.blockId, el);
    });
    blocksEl.addEventListener('keydown', (e) => {
      const el = closestDocBlock(window.getSelection() && window.getSelection().anchorNode);
      if (!el || el.getAttribute('contenteditable') === 'false') return;
      onBlockKeydown(e, el.dataset.blockId, el);
    });
  }
  let line = 1;
  blocksEl.querySelectorAll('.doc-block').forEach((el) => {
    if (el.dataset.type === 'page_break' || el.dataset.type === 'section_break') return;
    el.dataset.line = String(line++);
  });
  paintPeerCursors();
  updateWordCount();
}

function paintPeerCursors() {
  blocksEl.querySelectorAll('.peer-caret').forEach((n) => n.remove());
  Object.keys(peerCursors).forEach((uid) => {
    const c = peerCursors[uid];
    if (!c || !c.block_id) return;
    const el = blocksEl.querySelector('.doc-block[data-block-id="' + c.block_id + '"]');
    if (!el) return;
    const tip = document.createElement('span');
    tip.className = 'peer-caret';
    tip.style.cssText =
      'position:absolute;font-size:0.65rem;background:' +
      (c.color || '#c45') +
      ';color:#fff;padding:0 3px;border-radius:2px;margin-left:2px';
    tip.textContent = uid.slice(0, 8);
    el.style.position = 'relative';
    el.appendChild(tip);
  });
}

function broadcastCaret(blockId, offset) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  const { userId } = identity();
  ws.send(
    JSON.stringify({
      type: 'presence_caret',
      from: userId,
      user_id: userId,
      caret: { block_id: blockId, offset: offset || 0, color: peerColor(userId) },
    })
  );
}

function hashStr(s) {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0;
  return h;
}

function wsUrl(id) {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const token = localStorage.getItem('era_token') || '';
  const q = token ? '?access_token=' + encodeURIComponent(token) : '';
  return `${proto}//${location.host}/api/v1/docs/${id}/sync${q}`;
}

function applyRemoteOp(op) {
  if (!op || !op.type || !docState.blocks) return;
  const block = docState.blocks.find((b) => b.id === op.block_id);
  if (op.type === 'insert_text' && block) {
    const offset = op.offset || 0;
    const text = op.text || '';
    block.inlines = insertTextPreserving(block.inlines, offset, text, op.marks || null);
    refreshTableCellsFromInlines(block);
    renderBlocks();
    return;
  }
  if (op.type === 'delete_range' && block) {
    block.inlines = deleteRangePreserving(block.inlines, op.start || 0, op.end || 0);
    refreshTableCellsFromInlines(block);
    renderBlocks();
    return;
  }
  if (op.type === 'split_block' && block) {
    const idx = docState.blocks.findIndex((b) => b.id === op.block_id);
    if (idx < 0 || docState.blocks.some((b) => b.id === op.new_block_id)) return;
    const parts = splitInlinesAt(block.inlines, op.offset || 0);
    block.inlines = parts.left.length ? parts.left : [spanFromMarks('', {})];
    const neu = copyBlockShell(block, op.new_block_id, parts.right);
    docState.blocks.splice(idx + 1, 0, neu);
    renderBlocks();
    return;
  }
  if (op.type === 'merge_blocks' && op.with_previous !== false) {
    const idx = docState.blocks.findIndex((b) => b.id === op.block_id);
    if (idx <= 0) return;
    const prevB = docState.blocks[idx - 1];
    const cur = docState.blocks[idx];
    prevB.inlines = coalesceInlines((prevB.inlines || []).concat(cur.inlines || []));
    docState.blocks.splice(idx, 1);
    renderBlocks();
    return;
  }
  if (op.type === 'set_marks_range' && block) {
    const patch = {};
    ['bold', 'italic', 'underline', 'strike', 'superscript', 'subscript'].forEach((k) => {
      if (op[k] != null) patch[k] = op[k];
    });
    ['link_url', 'font_family', 'color', 'highlight'].forEach((k) => {
      if (Object.prototype.hasOwnProperty.call(op, k)) patch[k] = op[k] || undefined;
    });
    if (op.font_size_pt != null) patch.font_size_pt = op.font_size_pt || undefined;
    block.inlines = applyMarksRangeLocal(block.inlines, op.start || 0, op.end || 0, patch);
    renderBlocks();
    return;
  }
  if (op.type === 'set_block_type' && block) {
    block.block_type = op.block_type || 'paragraph';
    block.heading_level = op.heading_level || 0;
    if (op.block_type === 'list_item') block.list_type = op.list_type || 'bullet';
    renderBlocks();
    return;
  }
  if (op.type === 'insert_block' && op.block) {
    const idx = docState.blocks.findIndex((b) => b.id === op.after_id);
    if (idx >= 0) docState.blocks.splice(idx + 1, 0, op.block);
    else docState.blocks.push(op.block);
    renderBlocks();
    return;
  }
  if (op.type === 'set_block_format' && block) {
    if (op.align) block.align = op.align;
    if (op.clear_list) {
      block.list_type = undefined;
      block.list_marker = undefined;
      block.list_level = 0;
      if (block.block_type === 'list_item') block.block_type = 'paragraph';
    } else if (Object.prototype.hasOwnProperty.call(op, 'list_type')) {
      block.list_type = op.list_type || undefined;
      if (block.list_type) block.block_type = 'list_item';
    }
    if (op.indent_mm != null) block.indent_mm = op.indent_mm;
    if (op.line_spacing != null) block.line_spacing = op.line_spacing;
    if (op.list_level != null) block.list_level = op.list_level;
    if (op.list_marker != null) block.list_marker = op.list_marker;
    if (op.style_name != null) block.style_name = op.style_name || undefined;
    renderBlocks();
    return;
  }
  if (op.type === 'set_inline_marks' && block) {
    const end = blockText(block).length;
    const patch = {
      bold: !!op.bold,
      italic: !!op.italic,
      underline: !!op.underline,
      strike: !!op.strike,
      superscript: !!op.superscript,
      subscript: !!op.subscript,
    };
    if (Object.prototype.hasOwnProperty.call(op, 'link_url')) {
      patch.link_url = op.link_url || undefined;
    }
    block.inlines = applyMarksRangeLocal(block.inlines, 0, end, patch);
    renderBlocks();
    return;
  }
  if (op.type === 'add_comment' && op.id) {
    if (!docState.comments) docState.comments = [];
    if (!docState.comments.some((c) => c.id === op.id)) {
      docState.comments.push({
        id: op.id,
        block_id: op.block_id,
        author_id: op.author_id || '',
        text: op.text || '',
        resolved: false,
      });
    }
    renderComments();
    return;
  }
  if (op.type === 'resolve_comment' && op.id) {
    const c = (docState.comments || []).find((x) => x.id === op.id);
    if (c) c.resolved = true;
    renderComments();
  }
}

function onSyncMessage(ev) {
  const raw = typeof ev.data === 'string' ? ev.data : '';
  if (!raw || raw === 'ack') return;
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch (_) {
    return;
  }
  if (parsed.type === 'presence') {
    updatePresencePeers(parsed.peers || []);
    return;
  }
  if (parsed.type === 'presence_caret' || (parsed.caret && (parsed.from || parsed.user_id))) {
    const { userId } = identity();
    const uid = parsed.from || parsed.user_id;
    if (uid && uid !== userId && parsed.caret) {
      peerCursors[uid] = parsed.caret;
      paintPeerCursors();
    }
    return;
  }
  if (parsed.type === 'revision_event' && parsed.revision) {
    const { userId } = identity();
    if (parsed.from && parsed.from === userId) return;
    if (!docState.revisions) docState.revisions = [];
    if (parsed.action === 'add') {
      if (!docState.revisions.some((r) => r.id === parsed.revision.id)) {
        docState.revisions.push(parsed.revision);
      }
    } else if (parsed.action === 'remove' && parsed.revision.id) {
      docState.revisions = docState.revisions.filter((r) => r.id !== parsed.revision.id);
    }
    renderBlocks();
    return;
  }
  const op = parsed.op && parsed.op.type ? parsed.op : parsed;
  if (op && op.type) applyRemoteOp(op);
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

function connectSync(id) {
  stopPresenceHeartbeat();
  if (ws) {
    try {
      ws.close();
    } catch (_) {}
  }
  ws = new WebSocket(wsUrl(id));
  ws.onopen = () => {
    console.debug('docs sync connected');
    startPresenceHeartbeat();
  };
  ws.onerror = (e) => console.warn('docs sync error', e);
  ws.onclose = () => stopPresenceHeartbeat();
  ws.onmessage = onSyncMessage;
}

function sendOp(op) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  ws.send(JSON.stringify(op));
}

function onBlockInput(blockId, el) {
  const block = docState.blocks.find((b) => b.id === blockId);
  if (!block) return;
  const prev = blockText(block);
  // Preserve soft breaks; strip only a single trailing newline browsers often add.
  let next = el.innerText || '';
  if (next.endsWith('\n') && !prev.endsWith('\n')) next = next.slice(0, -1);
  if (prev === next) return;
  pushUndo();
  if (docState.track_changes) {
    if (!docState.revisions) docState.revisions = [];
    const rev = {
      id: crypto.randomUUID ? crypto.randomUUID() : 'r-' + Date.now(),
      block_id: blockId,
      kind: 'replace',
      before: prev,
      after: next,
      author_id: identity().userId,
    };
    docState.revisions.push(rev);
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(
        JSON.stringify({
          type: 'revision_event',
          action: 'add',
          from: identity().userId,
          revision: rev,
        })
      );
    }
  }
  const d = diffText(prev, next);
  if (d.deletedEnd > d.start) {
    block.inlines = deleteRangePreserving(block.inlines, d.start, d.deletedEnd);
  }
  if (d.inserted) {
    block.inlines = insertTextPreserving(block.inlines, d.start, d.inserted, typingMarks);
  }
  clearTimeout(syncTimer);
  syncTimer = setTimeout(() => {
    if (d.deletedEnd > d.start) {
      sendOp({
        type: 'delete_range',
        block_id: blockId,
        start: d.start,
        end: d.deletedEnd,
      });
    }
    if (d.inserted) {
      const op = {
        type: 'insert_text',
        block_id: blockId,
        offset: d.start,
        text: d.inserted,
      };
      if (typingMarks) op.marks = spanFromMarks('', typingMarks);
      sendOp(op);
    }
    scheduleAutosave();
  }, 200);
  updateWordCount();
}

function onBlockKeydown(e, blockId, el) {
  const block = docState.blocks.find((b) => b.id === blockId);
  if (!block) return;
  const special =
    block.block_type === 'image' ||
    block.block_type === 'table' ||
    block.block_type === 'page_break';
  if (special) return;

  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    const offs = selectionOffsetsInBlock(el) || { start: blockText(block).length, end: blockText(block).length };
    const caret = offs.start;
    // Empty list item → exit list (Word/Docs).
    if (
      (block.list_type || block.block_type === 'list_item') &&
      !blockText(block).trim()
    ) {
      pushUndo();
      block.list_type = undefined;
      block.list_marker = undefined;
      block.list_level = 0;
      block.block_type = 'paragraph';
      sendOp({ type: 'set_block_format', block_id: blockId, clear_list: true });
      renderBlocks();
      focusBlockAt(blockId, 0);
      scheduleAutosave();
      return;
    }
    pushUndo();
    const newId = newBlockId();
    const parts = splitInlinesAt(block.inlines, caret);
    block.inlines = parts.left.length ? parts.left : [spanFromMarks('', {})];
    const neu = copyBlockShell(block, newId, parts.right);
    const idx = docState.blocks.findIndex((b) => b.id === blockId);
    docState.blocks.splice(idx + 1, 0, neu);
    sendOp({
      type: 'split_block',
      block_id: blockId,
      offset: caret,
      new_block_id: newId,
    });
    renderBlocks();
    focusBlockAt(newId, 0);
    scheduleAutosave();
    return;
  }

  if (e.key === 'Enter' && e.shiftKey) {
    e.preventDefault();
    const offs = selectionOffsetsInBlock(el) || { start: blockText(block).length, end: blockText(block).length };
    if (offs.end > offs.start) {
      pushUndo();
      block.inlines = deleteRangePreserving(block.inlines, offs.start, offs.end);
      sendOp({
        type: 'delete_range',
        block_id: blockId,
        start: offs.start,
        end: offs.end,
      });
    } else {
      pushUndo();
    }
    block.inlines = insertTextPreserving(block.inlines, offs.start, '\n', typingMarks);
    sendOp({
      type: 'insert_text',
      block_id: blockId,
      offset: offs.start,
      text: '\n',
      marks: typingMarks ? spanFromMarks('', typingMarks) : undefined,
    });
    el.innerHTML = renderInlineHtml(block);
    focusBlockAt(blockId, offs.start + 1);
    scheduleAutosave();
    return;
  }

  if (e.key === 'Backspace') {
    const offs = selectionOffsetsInBlock(el);
    if (!offs || offs.start !== offs.end || offs.start !== 0) return;
    const idx = docState.blocks.findIndex((b) => b.id === blockId);
    if (idx <= 0) return;
    e.preventDefault();
    pushUndo();
    const prevB = docState.blocks[idx - 1];
    const mergeAt = blockText(prevB).length;
    prevB.inlines = coalesceInlines((prevB.inlines || []).concat(block.inlines || []));
    docState.blocks.splice(idx, 1);
    sendOp({ type: 'merge_blocks', block_id: blockId, with_previous: true });
    renderBlocks();
    focusBlockAt(prevB.id, mergeAt);
    scheduleAutosave();
  }
}

function applyDoc(doc) {
  docState = doc;
  if (!docState.comments) docState.comments = [];
  if (!docState.page) docState.page = { size: 'a4', orientation: 'portrait', margins_mm: 20 };
  if (!docState.header) docState.header = { text: '', page_numbers: false };
  if (!docState.footer) docState.footer = { text: '', page_numbers: false };
  if (!Array.isArray(docState.styles)) docState.styles = [];
  if (doc.legacy_features_dropped) banner.style.display = 'block';
  renderBlocks();
  renderComments();
  updateWordCount();
  clearFindHighlight();
  findMatches = [];
  findCursor = -1;
  lastFindQuery = '';
  try {
    document.dispatchEvent(new CustomEvent('era-doc-loaded'));
  } catch (_) {}
  if (window.eraSyncCustomStylesGallery) window.eraSyncCustomStylesGallery();
}

function selectedBlockEl() {
  const sel = window.getSelection();
  if (sel && sel.anchorNode) {
    const el =
      sel.anchorNode.nodeType === 1
        ? sel.anchorNode.closest('.doc-block')
        : sel.anchorNode.parentElement && sel.anchorNode.parentElement.closest('.doc-block');
    if (el) return el;
  }
  if (activeBlockId) {
    return blocksEl.querySelector('.doc-block[data-block-id="' + activeBlockId + '"]');
  }
  return blocksEl.querySelector('.doc-block');
}

function selectedBlock() {
  const el = selectedBlockEl();
  if (!el) return null;
  const block = docState.blocks.find((b) => b.id === el.dataset.blockId);
  return block ? { block, el } : null;
}

async function snapshotDoc() {
  if (!docId) return;
  const { tenantId, userId } = identity();
  const res = await officeFetch(`/api/v1/docs/${docId}/snapshot`, {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({ tenant_id: tenantId, user_id: userId }),
  });
  if (!res.ok) {
    if (window.EraOfficeShell && EraOfficeShell.handleUnauthorized && EraOfficeShell.handleUnauthorized(res)) {
      return false;
    }
    const msg =
      res.status === 403 ? 'Save failed: access denied' : 'Snapshot failed: ' + res.status;
    setAuthStatus(msg, true);
    setSaveStatus(msg, 'err');
    return false;
  }
  return true;
}

function selectionOrTypingRange(sel) {
  const offs = selectionOffsetsInBlock(sel.el);
  if (offs && offs.end > offs.start) return offs;
  return null;
}

function markAllOnInRange(block, start, end, mark) {
  let allOn = true;
  let acc = 0;
  for (const s of block.inlines || []) {
    const a = acc;
    const b = acc + (s.text || '').length;
    acc = b;
    if (b <= start || a >= end) continue;
    if (!s[mark]) allOn = false;
  }
  return allOn;
}

function toggleMark(mark) {
  const targets = collectFormatTargets();
  if (!targets.length) {
    setAuthStatus('Select a paragraph first', true);
    return;
  }
  const ranged = targets.filter((t) => !t.collapsed && t.end > t.start);
  if (!ranged.length) {
    const t0 = targets[0];
    const marks = blockMarks(t0.block);
    const cur = !!(typingMarks ? typingMarks[mark] : marks[mark]);
    const nextVal = !cur;
    typingMarks = Object.assign({}, typingMarks || marks, { [mark]: nextVal });
    setAuthStatus((mark === 'strike' ? 'Strike' : mark) + (nextVal ? ' on' : ' off') + ' (typing)', false);
    return;
  }
  let allOn = true;
  for (const t of ranged) {
    if (!markAllOnInRange(t.block, t.start, t.end, mark)) allOn = false;
  }
  const nextVal = !allOn;
  pushUndo();
  for (const t of ranged) {
    const patch = { [mark]: nextVal };
    t.block.inlines = applyMarksRangeLocal(t.block.inlines, t.start, t.end, patch);
    t.el.innerHTML = renderInlineHtml(t.block);
    const op = {
      type: 'set_marks_range',
      block_id: t.block.id,
      start: t.start,
      end: t.end,
    };
    op[mark] = nextVal;
    sendOp(op);
  }
  restoreMultiSelection(ranged);
  scheduleAutosave();
  setAuthStatus((mark === 'strike' ? 'Strike' : mark) + (nextVal ? ' on' : ' off'), false);
}

function setFontFamily(fam) {
  const targets = collectFormatTargets();
  if (!targets.length) return;
  const ranged = targets.filter((t) => !t.collapsed && t.end > t.start);
  if (!ranged.length) {
    const t0 = targets[0];
    typingMarks = Object.assign({}, typingMarks || blockMarks(t0.block), {
      font_family: fam || undefined,
    });
    setAuthStatus('Font ' + (fam || 'default') + ' (typing)', false);
    return;
  }
  pushUndo();
  for (const t of ranged) {
    t.block.inlines = applyMarksRangeLocal(t.block.inlines, t.start, t.end, {
      font_family: fam || undefined,
    });
    t.el.innerHTML = renderInlineHtml(t.block);
    sendOp({
      type: 'set_marks_range',
      block_id: t.block.id,
      start: t.start,
      end: t.end,
      font_family: fam || '',
    });
  }
  restoreMultiSelection(ranged);
  scheduleAutosave();
}

function setFontSize(pt) {
  const targets = collectFormatTargets();
  if (!targets.length) return;
  const n = pt ? parseInt(pt, 10) : 0;
  const ranged = targets.filter((t) => !t.collapsed && t.end > t.start);
  if (!ranged.length) {
    const t0 = targets[0];
    typingMarks = Object.assign({}, typingMarks || blockMarks(t0.block), {
      font_size_pt: n || undefined,
    });
    return;
  }
  pushUndo();
  for (const t of ranged) {
    t.block.inlines = applyMarksRangeLocal(t.block.inlines, t.start, t.end, {
      font_size_pt: n || undefined,
    });
    t.el.innerHTML = renderInlineHtml(t.block);
    sendOp({
      type: 'set_marks_range',
      block_id: t.block.id,
      start: t.start,
      end: t.end,
      font_size_pt: n || 0,
    });
  }
  restoreMultiSelection(ranged);
  scheduleAutosave();
}

function setAlign(align) {
  const targets = collectFormatTargets();
  if (!targets.length) return;
  pushUndo();
  const seen = new Set();
  for (const t of targets) {
    if (seen.has(t.block.id)) continue;
    seen.add(t.block.id);
    t.block.align = align;
    t.el.style.textAlign = align === 'justify' ? 'justify' : align;
    sendOp({ type: 'set_block_format', block_id: t.block.id, align });
  }
  restoreMultiSelection(targets.filter((t) => !t.collapsed));
  scheduleAutosave();
  setAuthStatus('Align ' + align, false);
}

function clearFormatting() {
  const sel = selectedBlock();
  if (!sel) return;
  pushUndo();
  const end = blockText(sel.block).length;
  const range = selectionOrTypingRange(sel) || { start: 0, end };
  const patch = {
    bold: false,
    italic: false,
    underline: false,
    strike: false,
    superscript: false,
    subscript: false,
    font_family: undefined,
    font_size_pt: undefined,
    color: undefined,
    highlight: undefined,
    link_url: undefined,
  };
  sel.block.inlines = applyMarksRangeLocal(sel.block.inlines, range.start, range.end, patch);
  if (range.start === 0 && range.end === end) {
    sel.block.align = 'left';
    sel.block.style_name = 'normal';
    sel.block.indent_mm = undefined;
    sel.block.line_spacing = undefined;
    sel.block.space_before_pt = undefined;
    sel.block.space_after_pt = undefined;
    sel.block.list_level = 0;
    sel.block.list_marker = undefined;
    sel.block.list_restart = false;
    sel.el.style.textAlign = 'left';
    sel.el.style.paddingLeft = '';
    sendOp({
      type: 'set_block_format',
      block_id: sel.block.id,
      align: 'left',
      style_name: 'normal',
      indent_mm: 0,
      line_spacing: '1.15',
      space_before_pt: 0,
      space_after_pt: 0,
      clear_list: true,
    });
  }
  sel.el.innerHTML = renderInlineHtml(sel.block);
  sendOp({
    type: 'set_marks_range',
    block_id: sel.block.id,
    start: range.start,
    end: range.end,
    bold: false,
    italic: false,
    underline: false,
    strike: false,
    superscript: false,
    subscript: false,
    font_family: '',
    font_size_pt: 0,
    color: '',
    highlight: '',
    link_url: '',
  });
  typingMarks = null;
  selectRangeInBlock(sel.block.id, range.start, range.end);
  scheduleAutosave();
}

/** @type {object|null} */
let formatPainterClip = null;
let showFormattingMarks = false;

function bumpIndent(delta) {
  const sel = selectedBlock();
  if (!sel) return;
  const cur = sel.block.indent_mm || 0;
  setIndentMm(cur + delta);
}

function setLineSpacing(val) {
  const sel = selectedBlock();
  if (!sel || !val) return;
  pushUndo();
  sel.block.line_spacing = val;
  sel.el.style.lineHeight = val;
  sendOp({ type: 'set_block_format', block_id: sel.block.id, line_spacing: val });
  scheduleAutosave();
}

async function setParaSpacing() {
  const sel = selectedBlock();
  if (!sel) return;
  const before = await EraOfficeShell.promptText({
    title: 'Paragraph spacing',
    label: 'Space before (pt)',
    value: String(sel.block.space_before_pt || 0),
  });
  if (before == null) return;
  const after = await EraOfficeShell.promptText({
    title: 'Paragraph spacing',
    label: 'Space after (pt)',
    value: String(sel.block.space_after_pt || 0),
  });
  if (after == null) return;
  pushUndo();
  sel.block.space_before_pt = parseInt(before, 10) || 0;
  sel.block.space_after_pt = parseInt(after, 10) || 0;
  sel.el.style.marginTop = sel.block.space_before_pt + 'pt';
  sel.el.style.marginBottom = sel.block.space_after_pt + 'pt';
  sendOp({
    type: 'set_block_format',
    block_id: sel.block.id,
    space_before_pt: sel.block.space_before_pt,
    space_after_pt: sel.block.space_after_pt,
  });
  scheduleAutosave();
}

function bumpListLevel(delta) {
  const sel = selectedBlock();
  if (!sel) return;
  if (sel.block.block_type !== 'list_item') setListType('bullet');
  const cur = sel.block.list_level || 0;
  const next = Math.max(0, Math.min(8, cur + delta));
  pushUndo();
  sel.block.list_level = next;
  sel.el.style.marginLeft = next * 12 + 'mm';
  sendOp({ type: 'set_block_format', block_id: sel.block.id, list_level: next });
  scheduleAutosave();
}

async function changeListMarker() {
  const sel = selectedBlock();
  if (!sel) return;
  const m = await EraOfficeShell.chooseOption({
    title: 'List marker',
    message: 'Choose a marker style',
    options: [
      { value: 'disc', label: 'Disc', hint: 'Bullet' },
      { value: 'circle', label: 'Circle', hint: 'Bullet' },
      { value: 'square', label: 'Square', hint: 'Bullet' },
      { value: 'decimal', label: 'Decimal', hint: '1, 2, 3…' },
      { value: 'lower_alpha', label: 'Lower alpha', hint: 'a, b, c…' },
      { value: 'lower_roman', label: 'Lower roman', hint: 'i, ii, iii…' },
    ],
    value: sel.block.list_marker || 'disc',
  });
  if (!m) return;
  const marker = String(m).trim().toLowerCase().replace(/\s+/g, '_');
  pushUndo();
  if (sel.block.block_type !== 'list_item')
    setListType(/decimal|alpha|roman/i.test(marker) ? 'ordered' : 'bullet');
  sel.block.list_marker = marker;
  if (sel.el) sel.el.dataset.listMarker = marker;
  sendOp({ type: 'set_block_format', block_id: sel.block.id, list_marker: sel.block.list_marker });
  setAuthStatus('List marker: ' + marker, false);
  scheduleAutosave();
}

function restartListNumbering() {
  const sel = selectedBlock();
  if (!sel) return;
  pushUndo();
  sel.block.list_restart = true;
  if (sel.el) sel.el.dataset.listRestart = '1';
  sendOp({ type: 'set_block_format', block_id: sel.block.id, list_restart: true });
  setAuthStatus('Numbering restart at this item (visible marker ①)', false);
  scheduleAutosave();
}

function toggleScript(kind) {
  const sel = selectedBlock();
  if (!sel) return;
  pushUndo();
  const marks = blockMarks(sel.block);
  const on = kind === 'super' ? !marks.superscript : !marks.subscript;
  sel.block.inlines = [spanFromMarks(blockText(sel.block), marks, {
    superscript: kind === 'super' ? on : false,
    subscript: kind === 'sub' ? on : false,
  })];
  sel.el.innerHTML = renderInlineHtml(sel.block);
  sendOp({
    type: 'set_inline_marks',
    block_id: sel.block.id,
    bold: !!marks.bold,
    italic: !!marks.italic,
    underline: !!marks.underline,
    strike: !!marks.strike,
    superscript: kind === 'super' ? on : false,
    subscript: kind === 'sub' ? on : false,
  });
  scheduleAutosave();
}

function armFormatPainter() {
  const sel = selectedBlock();
  if (!sel) {
    setAuthStatus('Select a block to copy format', true);
    return;
  }
  formatPainterClip = {
    align: sel.block.align,
    indent_mm: sel.block.indent_mm,
    line_spacing: sel.block.line_spacing,
    space_before_pt: sel.block.space_before_pt,
    space_after_pt: sel.block.space_after_pt,
    style_name: sel.block.style_name,
    list_type: sel.block.list_type,
    list_level: sel.block.list_level,
    list_marker: sel.block.list_marker,
    marks: blockMarks(sel.block),
  };
  setAuthStatus('Format Painter armed — click a block to apply', false);
}

function applyFormatPainterTo(sel) {
  if (!formatPainterClip || !sel) return;
  pushUndo();
  const c = formatPainterClip;
  Object.assign(sel.block, {
    align: c.align,
    indent_mm: c.indent_mm,
    line_spacing: c.line_spacing,
    space_before_pt: c.space_before_pt,
    space_after_pt: c.space_after_pt,
    style_name: c.style_name,
    list_type: c.list_type,
    list_level: c.list_level,
    list_marker: c.list_marker,
  });
  const plain = blockText(sel.block);
  const m = c.marks || {};
  // Preserve multi-span structure by applying marks across the whole block.
  sel.block.inlines = applyMarksRangeLocal(sel.block.inlines || [{ text: plain }], 0, plain.length, {
    bold: !!m.bold,
    italic: !!m.italic,
    underline: !!m.underline,
    strike: !!m.strike,
    superscript: !!m.superscript,
    subscript: !!m.subscript,
    font_family: m.font_family || undefined,
    font_size_pt: m.font_size_pt || undefined,
    color: m.color || undefined,
    highlight: m.highlight || undefined,
  });
  formatPainterClip = null;
  sendOp({
    type: 'set_block_format',
    block_id: sel.block.id,
    align: c.align,
    indent_mm: c.indent_mm,
    line_spacing: c.line_spacing,
    space_before_pt: c.space_before_pt,
    space_after_pt: c.space_after_pt,
    style_name: c.style_name || '',
    list_type: c.list_type,
    list_level: c.list_level,
    list_marker: c.list_marker,
  });
  sendOp({
    type: 'set_marks_range',
    block_id: sel.block.id,
    start: 0,
    end: plain.length,
    bold: !!m.bold,
    italic: !!m.italic,
    underline: !!m.underline,
    strike: !!m.strike,
    font_family: m.font_family || '',
    font_size_pt: m.font_size_pt || 0,
    color: m.color || '',
    highlight: m.highlight || '',
  });
  renderBlocks();
  scheduleAutosave();
  setAuthStatus('Format applied', false);
}

function toggleShowMarks() {
  showFormattingMarks = !showFormattingMarks;
  const page = document.getElementById('blocks');
  if (page) page.classList.toggle('era-show-marks', showFormattingMarks);
  setAuthStatus(showFormattingMarks ? 'Formatting marks on' : 'Formatting marks off', false);
}

async function insertSymbol() {
  const sel = selectedBlock();
  if (!sel) return;
  const ch = await EraOfficeShell.promptText({
    title: 'Insert symbol',
    label: 'Symbol',
    value: '©',
    message: 'Examples: © ® ™ § ± →',
    placeholder: '©',
  });
  if (!ch) return;
  pushUndo();
  const at = blockText(sel.block).length;
  const marks = typingMarks || blockMarks(sel.block);
  sel.block.inlines = insertTextPreserving(sel.block.inlines, at, ch, marks);
  sel.el.innerHTML = renderInlineHtml(sel.block);
  sendOp({
    type: 'insert_text',
    block_id: sel.block.id,
    offset: at,
    text: ch,
  });
  scheduleAutosave();
}

async function pastePlainText(t) {
  const sel = selectedBlock();
  if (!sel || t == null) return;
  const text = String(t).replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  // Multi-paragraph paste: first para into selection, rest as new blocks (MVP).
  const parts = text.split('\n');
  pushUndo();
  const { block, el } = sel;
  let range = selectionOffsetsInBlock(el) || { start: blockText(block).length, end: blockText(block).length };
  if (range.end > range.start) {
    block.inlines = deleteRangePreserving(block.inlines, range.start, range.end);
    sendOp({
      type: 'delete_range',
      block_id: block.id,
      start: range.start,
      end: range.end,
    });
  }
  const first = parts[0] || '';
  if (first) {
    block.inlines = insertTextPreserving(block.inlines, range.start, first, typingMarks || blockMarks(block));
    sendOp({
      type: 'insert_text',
      block_id: block.id,
      offset: range.start,
      text: first,
    });
  }
  el.innerHTML = renderInlineHtml(block);
  let afterId = block.id;
  for (let i = 1; i < parts.length; i++) {
    const nb = {
      id: newBlockId(),
      block_type: 'paragraph',
      heading_level: 0,
      inlines: [{ text: parts[i], bold: false, italic: false, underline: false }],
    };
    const idx = docState.blocks.findIndex((b) => b.id === afterId);
    docState.blocks.splice(idx + 1, 0, nb);
    sendOp({ type: 'insert_block', after_id: afterId, block: nb });
    afterId = nb.id;
  }
  if (parts.length > 1) renderBlocks();
  scheduleAutosave();
  setAuthStatus('Pasted plain text', false);
}

function insertHorizontalLine() {
  if (!docState.blocks) return;
  pushUndo();
  const id =
    typeof crypto !== 'undefined' && crypto.randomUUID
      ? crypto.randomUUID()
      : 'hr-' + Date.now();
  const after = selectedBlock();
  const block = {
    id,
    block_type: 'paragraph',
    style_name: 'horizontal_line',
    heading_level: 0,
    inlines: [{ text: '' }],
  };
  if (after) {
    const idx = docState.blocks.findIndex((b) => b.id === after.block.id);
    docState.blocks.splice(idx + 1, 0, block);
    sendOp({ type: 'insert_block', after_id: after.block.id, block });
  } else {
    docState.blocks.push(block);
    sendOp({ type: 'insert_block', after_id: null, block });
  }
  renderBlocks();
  scheduleAutosave();
}

function insertPageBreak() {
  if (!docState.blocks) return;
  pushUndo();
  const id =
    typeof crypto !== 'undefined' && crypto.randomUUID
      ? crypto.randomUUID()
      : 'pb-' + Date.now();
  const after = selectedBlock();
  const block = {
    id,
    block_type: 'page_break',
    heading_level: 0,
    inlines: [],
  };
  if (after) {
    const idx = docState.blocks.findIndex((b) => b.id === after.block.id);
    docState.blocks.splice(idx + 1, 0, block);
    sendOp({ type: 'insert_block', after_id: after.block.id, block });
  } else {
    docState.blocks.push(block);
  }
  renderBlocks();
  scheduleAutosave();
}

function setListType(kind, marker) {
  const targets = collectFormatTargets();
  if (!targets.length) {
    setAuthStatus('Select a paragraph first', true);
    return;
  }
  pushUndo();
  const seen = new Set();
  for (const t of targets) {
    if (seen.has(t.block.id)) continue;
    seen.add(t.block.id);
    const { block, el } = t;
    if (
      !marker &&
      block.block_type === 'list_item' &&
      block.list_type === kind &&
      targets.length === 1
    ) {
      setBlockType('paragraph', 0);
      return;
    }
    block.block_type = 'list_item';
    block.list_type = kind;
    block.heading_level = 0;
    if (marker) {
      block.list_marker = marker;
      el.dataset.listMarker = marker;
    }
    el.dataset.type = 'list_item';
    el.dataset.list = kind;
    const op = {
      type: 'set_block_type',
      block_id: block.id,
      block_type: 'list_item',
      heading_level: 0,
      list_type: kind,
    };
    if (marker) op.list_marker = marker;
    sendOp(op);
  }
  scheduleAutosave();
}

function applyStyle(style) {
  if (String(style || '').startsWith('custom:')) {
    const name = String(style).slice('custom:'.length);
    if (window.eraApplyCustomStyleByName) window.eraApplyCustomStyleByName(name);
    return;
  }
  if (style === 'p' || style === 'normal') {
    setBlockType('paragraph', 0);
    const sel = selectedBlock();
    if (sel) {
      sel.block.style_name = 'normal';
      sendOp({ type: 'set_block_format', block_id: sel.block.id, style_name: 'normal' });
    }
    return;
  }
  if (style === 'quote' || style === 'caption') {
    setBlockType('paragraph', 0);
    const sel = selectedBlock();
    if (sel) {
      pushUndo();
      sel.block.style_name = style;
      if (style === 'quote') sel.el.style.fontStyle = 'italic';
      sendOp({ type: 'set_block_format', block_id: sel.block.id, style_name: style });
      scheduleAutosave();
    }
    return;
  }
  if (style === 'title') {
    setBlockType('heading', 1);
    const sel = selectedBlock();
    if (sel) {
      sel.block.style_name = 'title';
      sel.el.style.fontSize = '1.75rem';
    }
    return;
  }
  const level = parseInt(String(style).replace('h', ''), 10);
  if (level >= 1 && level <= 6) setBlockType('heading', level);
}

function openPageSetup() {
  const dlg = document.getElementById('pageSetupDlg');
  const page = docState.page || {};
  document.getElementById('pageSize').value = page.size || 'a4';
  document.getElementById('pageOrient').value = page.orientation || 'portrait';
  document.getElementById('pageMargins').value = page.margins_mm != null ? page.margins_mm : 20;
  if (dlg && dlg.showModal) dlg.showModal();
}

function commitPageSetup() {
  pushUndo();
  const prevCols = (docState.page && docState.page.columns) || 1;
  docState.page = {
    size: document.getElementById('pageSize').value,
    orientation: document.getElementById('pageOrient').value,
    margins_mm: parseInt(document.getElementById('pageMargins').value, 10) || 20,
    columns: prevCols,
  };
  applyPageChrome();
  scheduleAutosave();
  setAuthStatus('Page setup updated', false);
}

function showHeaderFooter(which) {
  const strip = document.getElementById('headerStrip');
  if (strip) strip.hidden = false;
  if (!docState.header) docState.header = { text: '', page_numbers: false };
  if (!docState.footer) docState.footer = { text: '', page_numbers: false };
  if (which === 'numbers') {
    docState.footer.page_numbers = true;
    document.getElementById('pageNumbersChk').checked = true;
  }
  setAuthStatus('Edit header/footer below the toolbar', false);
}

function syncHeaderFooterFromInputs() {
  if (!docState.header) docState.header = { text: '', page_numbers: false };
  if (!docState.footer) docState.footer = { text: '', page_numbers: false };
  docState.header.text = document.getElementById('headerInput').value || '';
  docState.footer.text = document.getElementById('footerInput').value || '';
  const pn = document.getElementById('pageNumbersChk').checked;
  docState.header.page_numbers = pn;
  docState.footer.page_numbers = pn;
  scheduleAutosave();
}

async function setLink() {
  const sel = selectedBlock();
  if (!sel) {
    setAuthStatus('Select a paragraph first', true);
    return;
  }
  const { block, el } = sel;
  const marks = blockMarks(block);
  const range = selectionOrTypingRange(sel) || { start: 0, end: blockText(block).length };
  const next = await EraOfficeShell.promptText({
    title: 'Link',
    label: 'URL (empty to remove)',
    value: marks.link_url || 'https://',
    placeholder: 'https://',
  });
  if (next == null) return;
  const url = next.trim();
  pushUndo();
  block.inlines = applyMarksRangeLocal(block.inlines, range.start, range.end, {
    link_url: url || undefined,
  });
  el.innerHTML = renderInlineHtml(block);
  selectRangeInBlock(block.id, range.start, range.end);
  sendOp({
    type: 'set_marks_range',
    block_id: block.id,
    start: range.start,
    end: range.end,
    link_url: url,
  });
  scheduleAutosave();
  setAuthStatus(url ? 'Link set' : 'Link removed', false);
}

async function addComment() {
  const sel = selectedBlock();
  if (!sel) {
    setAuthStatus('Select a paragraph first', true);
    return;
  }
  const text = await EraOfficeShell.promptText({
    title: 'Add comment',
    label: 'Comment text',
    value: '',
    multiline: true,
  });
  if (text == null || !text.trim()) return;
  const { userId } = identity();
  const id =
    typeof crypto !== 'undefined' && crypto.randomUUID
      ? crypto.randomUUID()
      : 'c-' + Date.now();
  const range = selectionOffsetsInBlock(sel.el) || { start: 0, end: blockText(sel.block).length };
  const quote = blockText(sel.block).slice(range.start, range.end) || blockText(sel.block).slice(0, 80);
  const op = {
    type: 'add_comment',
    id,
    block_id: sel.block.id,
    author_id: userId,
    text: text.trim(),
    start: range.start,
    end: range.end,
    quote: quote,
  };
  if (!docState.comments) docState.comments = [];
  docState.comments.push({
    id,
    block_id: sel.block.id,
    author_id: userId,
    text: text.trim(),
    resolved: false,
    start: range.start,
    end: range.end,
    quote,
  });
  sendOp(op);
  renderComments();
  if (window.EraOfficeShell && EraOfficeShell.setCommentsOpen) {
    EraOfficeShell.setCommentsOpen(true);
  }
  scheduleAutosave();
  setAuthStatus('Comment added', false);
}

function resolveComment(id) {
  const c = (docState.comments || []).find((x) => x.id === id);
  if (!c) return;
  c.resolved = true;
  sendOp({ type: 'resolve_comment', id });
  renderComments();
  scheduleAutosave();
}

function setBlockType(blockType, headingLevel) {
  const sel = selectedBlock();
  if (!sel) {
    setAuthStatus('Select a paragraph first', true);
    return;
  }
  pushUndo();
  const { block, el } = sel;
  const op = {
    type: 'set_block_type',
    block_id: block.id,
    block_type: blockType,
    heading_level: headingLevel || 0,
  };
  if (blockType === 'list_item') op.list_type = 'bullet';
  sendOp(op);
  block.block_type = blockType;
  block.heading_level = headingLevel || 0;
  if (blockType === 'list_item') block.list_type = 'bullet';
  el.dataset.type =
    blockType === 'heading' ? 'heading' : blockType === 'list_item' ? 'list_item' : 'paragraph';
  if (blockType === 'heading') el.dataset.level = String(headingLevel || 1);
  else delete el.dataset.level;
  scheduleAutosave();
  setAuthStatus('Block type → ' + blockType + (headingLevel ? ' L' + headingLevel : ''), false);
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

/** AC-O8: when Comms deep-link carries intent_exp/intent_sig, verify HMAC server-side. */
async function verifyCommsIntent(id) {
  const params = new URLSearchParams(location.search);
  const exp = params.get('intent_exp');
  const sig = params.get('intent_sig');
  if (!exp && !sig) return true;
  if (!exp || !sig) {
    setAuthStatus('Invalid Comms edit link (missing intent).', true);
    return false;
  }
  const q = new URLSearchParams({ intent_exp: exp, intent_sig: sig });
  const res = await officeFetch(`/api/v1/docs/${id}/verify-intent?${q}`, {
    headers: authHeaders(),
  });
  if (!res.ok) {
    setAuthStatus('Edit link expired or invalid (' + res.status + ').', true);
    return false;
  }
  return true;
}

async function loadDoc(id) {
  if (!(await verifyCommsIntent(id))) return;
  const params = new URLSearchParams(location.search);
  const exp = params.get('intent_exp');
  const sig = params.get('intent_sig');
  let url = `/api/v1/docs/${id}`;
  if (exp && sig) {
    url += '?' + new URLSearchParams({ intent_exp: exp, intent_sig: sig });
  }
  const res = await officeFetch(url, {
    headers: authHeaders(),
  });
  if (!res.ok) {
    if (window.EraOfficeShell && EraOfficeShell.handleUnauthorized && EraOfficeShell.handleUnauthorized(res)) {
      return;
    }
    const msg =
      res.status === 403
        ? 'Access denied for this document'
        : 'Failed to load document: ' + res.status;
    setAuthStatus(msg, true);
    return;
  }
  applyDoc(await res.json());
  setDocMeta('Document: ' + id);
  setAuthStatus('Document open', false);
  setSaveStatus('All changes saved', 'ok');
  setDocTitleUI('Untitled', true);
  await loadDriveName(id);
  refreshPresence();
  connectSync(id);
}

async function createDocument() {
  if (!localStorage.getItem('era_token')) {
    setAuthStatus('Sign in via Drive first (era_token).', true);
    return;
  }
  const { tenantId, userId } = identity();
  // Drive forbids duplicate names in a folder — use a unique default.
  const name = 'Untitled-' + Date.now() + '.erad';
  const res = await officeFetch('/api/v1/docs', {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({
      tenant_id: tenantId,
      user_id: userId,
      name,
    }),
  });
  if (!res.ok) {
    setAuthStatus(
      res.status === 502 || res.status === 409
        ? 'Create failed: name conflict or Drive bind (' + res.status + ')'
        : 'Create failed: ' + res.status,
      true
    );
    return;
  }
  const data = await res.json();
  if (!data.drive_object_id) {
    setAuthStatus('Create failed: no drive_object_id', true);
    return;
  }
  location.href = '/docs/' + data.drive_object_id;
}

document.getElementById('newDocBtn').onclick = () => {
  createDocument().catch(() => {});
};

document.getElementById('findNextBtn').onclick = () => findNext();
document.getElementById('findInput').addEventListener('keydown', (e) => {
  if (e.key === 'Enter') {
    e.preventDefault();
    findNext();
  }
});

document.getElementById('boldBtn').onclick = () => toggleMark('bold');
document.getElementById('italicBtn').onclick = () => toggleMark('italic');
document.getElementById('underlineBtn').onclick = () => toggleMark('underline');
document.getElementById('strikeBtn').onclick = () => toggleMark('strike');
document.getElementById('linkBtn').onclick = () => setLink();
document.getElementById('commentBtn').onclick = () => addComment();
document.getElementById('alignLeftBtn').onclick = () => setAlign('left');
document.getElementById('alignCenterBtn').onclick = () => setAlign('center');
document.getElementById('alignRightBtn').onclick = () => setAlign('right');
const alignJustifyBtn = document.getElementById('alignJustifyBtn');
if (alignJustifyBtn) alignJustifyBtn.onclick = () => setAlign('justify');
const indentDecBtn = document.getElementById('indentDecBtn');
if (indentDecBtn) indentDecBtn.onclick = () => bumpIndent(-10);
const indentIncBtn = document.getElementById('indentIncBtn');
if (indentIncBtn) indentIncBtn.onclick = () => bumpIndent(10);
const lineSpacingSelect = document.getElementById('lineSpacingSelect');
if (lineSpacingSelect) lineSpacingSelect.onchange = (e) => setLineSpacing(e.target.value);
document.getElementById('numberedListBtn').onclick = () => setListType('ordered');

document.getElementById('listBtn').onclick = () => setListType('bullet');
document.querySelectorAll('.era-menu-preset[data-list]').forEach((btn) => {
  btn.addEventListener('click', () => setListType(btn.getAttribute('data-list'), btn.getAttribute('data-marker') || undefined));
});
if (window.EraOfficeToolbar) EraOfficeToolbar.init(document);
if (window.EraOfficeShell && EraOfficeShell.mountIcons) EraOfficeShell.mountIcons(document);
const listLevelDecBtn = document.getElementById('listLevelDecBtn');
if (listLevelDecBtn) listLevelDecBtn.onclick = () => bumpListLevel(-1);
const listLevelIncBtn = document.getElementById('listLevelIncBtn');
if (listLevelIncBtn) listLevelIncBtn.onclick = () => bumpListLevel(1);
const formatPainterBtn = document.getElementById('formatPainterBtn');
if (formatPainterBtn) formatPainterBtn.onclick = () => armFormatPainter();
const undoBtn = document.getElementById('undoBtn');
if (undoBtn) undoBtn.onclick = () => undoEdit();
const redoBtn = document.getElementById('redoBtn');
if (redoBtn) redoBtn.onclick = () => redoEdit();
const printBtn = document.getElementById('printBtn');
if (printBtn) printBtn.onclick = () => window.print();
const spellBtn = document.getElementById('spellBtn');
if (spellBtn) spellBtn.onclick = () => runSpellingLite();
const imageTbBtn = document.getElementById('imageTbBtn');
if (imageTbBtn) imageTbBtn.onclick = () => insertImageBlock();
const clearFmtBtn = document.getElementById('clearFmtBtn');
if (clearFmtBtn) clearFmtBtn.onclick = () => clearFormatting();
const zoomSelect = document.getElementById('zoomSelect');
if (zoomSelect) {
  zoomSelect.onchange = () => {
    const page = document.getElementById('blocks');
    if (!page) return;
    const z = parseInt(zoomSelect.value, 10) || 100;
    page.style.zoom = String(z / 100);
  };
}
function bumpFontSizeTb(delta) {
  const sel = document.getElementById('sizeSelect');
  const sizes = [10, 11, 12, 14, 16, 18, 24];
  let cur = parseInt((sel && sel.value) || '11', 10) || 11;
  let idx = sizes.indexOf(cur);
  if (idx < 0) idx = 1;
  idx = Math.max(0, Math.min(sizes.length - 1, idx + delta));
  if (sel) sel.value = String(sizes[idx]);
  setFontSize(String(sizes[idx]));
}
const fontDecTbBtn = document.getElementById('fontDecTbBtn');
if (fontDecTbBtn) fontDecTbBtn.onclick = () => bumpFontSizeTb(-1);
const fontIncTbBtn = document.getElementById('fontIncTbBtn');
if (fontIncTbBtn) fontIncTbBtn.onclick = () => bumpFontSizeTb(1);
const superBtn = document.getElementById('superBtn');
if (superBtn) superBtn.onclick = () => toggleScript('super');
const subBtn = document.getElementById('subBtn');
if (subBtn) subBtn.onclick = () => toggleScript('sub');

document.getElementById('styleSelect').onchange = (e) => applyStyle(e.target.value);
document.getElementById('fontSelect').onchange = (e) => setFontFamily(e.target.value);
document.getElementById('sizeSelect').onchange = (e) => setFontSize(e.target.value);

document.getElementById('blocks').addEventListener('click', () => {
  if (!formatPainterClip) return;
  const sel = selectedBlock();
  if (sel) applyFormatPainterTo(sel);
});

['headerInput', 'footerInput', 'pageNumbersChk'].forEach((id) => {
  const el = document.getElementById(id);
  if (el) el.addEventListener('change', () => syncHeaderFooterFromInputs());
});

document.getElementById('pageSetupDlg').addEventListener('close', () => {
  if (document.getElementById('pageSetupDlg').returnValue === 'ok') commitPageSetup();
});

document.getElementById('snapshotBtn').onclick = async () => {
  if (!docId) return;
  clearTimeout(autosaveTimer);
  saving = true;
  setSaveStatus('Saving…', 'saving');
  const ok = await snapshotDoc();
  saving = false;
  if (ok) {
    dirty = false;
    setAuthStatus('Saved to Drive', false);
    setSaveStatus('Saved · ' + new Date().toLocaleTimeString(), 'ok');
  } else {
    setSaveStatus('Save failed', 'err');
  }
};

const shareBtn = document.getElementById('shareBtn');
if (shareBtn) shareBtn.onclick = () => openShareDialog();
const shareCopyBtn = document.getElementById('shareCopyBtn');
if (shareCopyBtn) {
  shareCopyBtn.onclick = () => {
    const input = document.getElementById('shareLinkInput');
    const url = (input && input.value) || '';
    if (!url) return;
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(url).then(
        () => setAuthStatus('Link copied', false),
        () => EraOfficeShell.promptCopy({ title: 'Share document', value: url })
      );
    } else {
      void EraOfficeShell.promptCopy({ title: 'Share document', value: url });
    }
  };
}
if (window.EraOfficeShell && EraOfficeShell.wireDocTitle) {
  EraOfficeShell.wireDocTitle(docTitle, (name) => {
    renameDriveObject(name).catch(() => {});
  });
}

document.getElementById('importBtn').onclick = () => document.getElementById('file').click();
document.getElementById('file').onchange = async (e) => {
  const f = e.target.files[0];
  if (!f) return;
  if (!localStorage.getItem('era_token')) {
    setAuthStatus('Sign in via Drive first (era_token).', true);
    return;
  }
  const { tenantId, userId } = identity();
  setAuthStatus('Importing docx…', false);
  const buf = await f.arrayBuffer();
  const b64 = arrayBufferToBase64(buf);
  const res = await officeFetch('/api/v1/docs/import', {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({ tenant_id: tenantId, user_id: userId, docx_base64: b64, name: f.name }),
  });
  if (!res.ok) {
    const msg =
      res.status === 409
        ? 'Import failed: a file with this name already exists in Drive'
        : res.status === 502
          ? 'Import failed: Docs could not write to Drive (502) — retry or check Drive'
          : res.status === 400
            ? 'Import failed: unsupported or corrupt docx'
            : 'Import failed: ' + res.status;
    setAuthStatus(msg, true);
    e.target.value = '';
    return;
  }
  const data = await res.json();
  if (!data.drive_object_id) {
    setAuthStatus('Import failed: no drive_object_id', true);
    return;
  }
  location.href = '/docs/' + data.drive_object_id;
};

document.getElementById('exportBtn').onclick = async () => {
  if (!docId) {
    setAuthStatus('Open a document first', true);
    return;
  }
  setAuthStatus('Exporting docx…', false);
  const res = await officeFetch(`/api/v1/docs/${docId}/export/docx`, {
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
  a.download = (docId || 'export') + '.docx';
  a.click();
  URL.revokeObjectURL(a.href);
  setAuthStatus('Export ready', false);
};

document.getElementById('summarizeAIBtn').onclick = () => {
  const text = (docState.blocks || []).map((b) => blockText(b)).filter(Boolean).join('\n\n');
  if (!text.trim()) {
    setAuthStatus('Document has no text to summarize', true);
    return;
  }
  try {
    sessionStorage.setItem('era_office_ai_text', text);
    sessionStorage.removeItem('era_office_ai_mode');
  } catch (_) {}
  location.href = '/office-ai/';
};

document.getElementById('rewriteAIBtn').onclick = () => {
  const text = (docState.blocks || []).map((b) => blockText(b)).filter(Boolean).join('\n\n');
  if (!text.trim()) {
    setAuthStatus('Document has no text to rewrite', true);
    return;
  }
  try {
    sessionStorage.setItem('era_office_ai_text', text);
    sessionStorage.setItem('era_office_ai_mode', 'rewrite');
  } catch (_) {}
  location.href = '/office-ai/?mode=rewrite';
};

function doExport() {
  document.getElementById('exportBtn').click();
}
function doImport() {
  document.getElementById('importBtn').click();
}
function doSave() {
  document.getElementById('snapshotBtn').click();
}

function insertBlockAfter(block) {
  pushUndo();
  const after = selectedBlock();
  let idx = docState.blocks.length - 1;
  if (after) idx = docState.blocks.findIndex((b) => b.id === after.block.id);
  docState.blocks.splice(idx + 1, 0, block);
  sendOp({
    type: 'insert_block',
    after_id: after ? after.block.id : null,
    block,
  });
  renderBlocks();
  scheduleAutosave();
}

function setTextColor(color) {
  const sel = selectedBlock();
  if (!sel) return setAuthStatus('Select a paragraph first', true);
  const range = selectionOrTypingRange(sel) || { start: 0, end: blockText(sel.block).length };
  if (range.start === range.end && !selectionOrTypingRange(sel)) {
    typingMarks = Object.assign({}, typingMarks || blockMarks(sel.block), {
      color: color || undefined,
    });
    setAuthStatus('Text color (typing)', false);
    return;
  }
  pushUndo();
  const { block, el } = sel;
  block.inlines = applyMarksRangeLocal(block.inlines, range.start, range.end, {
    color: color || undefined,
  });
  el.innerHTML = renderInlineHtml(block);
  selectRangeInBlock(block.id, range.start, range.end);
  sendOp({
    type: 'set_marks_range',
    block_id: block.id,
    start: range.start,
    end: range.end,
    color: color || '',
  });
  scheduleAutosave();
  setAuthStatus('Text color applied', false);
}

function setHighlightColor(color) {
  const sel = selectedBlock();
  if (!sel) return setAuthStatus('Select a paragraph first', true);
  const range = selectionOrTypingRange(sel) || { start: 0, end: blockText(sel.block).length };
  pushUndo();
  const { block, el } = sel;
  block.inlines = applyMarksRangeLocal(block.inlines, range.start, range.end, {
    highlight: color || undefined,
  });
  el.innerHTML = renderInlineHtml(block);
  selectRangeInBlock(block.id, range.start, range.end);
  sendOp({
    type: 'set_marks_range',
    block_id: block.id,
    start: range.start,
    end: range.end,
    highlight: color || '',
  });
  scheduleAutosave();
  setAuthStatus('Highlight applied', false);
}

async function pickDriveImageObject() {
  try {
    const res = await officeFetch('/api/v1/drive/folders/_root/children', { headers: authHeaders() });
    if (!res.ok) return null;
    const data = await res.json();
    const objs = (data.objects || []).filter((o) => {
      const n = String(o.name || '').toLowerCase();
      const ct = String(o.content_type || '').toLowerCase();
      return ct.startsWith('image/') || /\.(png|jpe?g|gif|webp)$/i.test(n);
    });
    if (!objs.length) {
      setAuthStatus('No images in Drive root — paste URL or upload first', true);
      return null;
    }
    const options = objs
      .map((o, i) => ({
        value: String(i),
        label: o.name || o.id,
        hint: o.content_type || o.id,
      }))
      .concat([{ value: 'url', label: 'Paste URL instead…', hint: 'https:// or data:' }]);
    const pick = await EraOfficeShell.chooseOption({
      title: 'Pick Drive image',
      message: 'Images in Drive root',
      options,
      value: '0',
    });
    if (pick == null) return null;
    if (pick === 'url') return { url: null };
    const idx = parseInt(pick, 10);
    if (Number.isNaN(idx) || idx < 0 || idx >= objs.length) return null;
    const o = objs[idx];
    return {
      url: '/api/v1/drive/objects/' + encodeURIComponent(o.id),
      object_id: o.id,
      alt: o.name || '',
    };
  } catch (_) {
    return null;
  }
}

async function insertImageBlock() {
  const mode = await EraOfficeShell.chooseOption({
    title: 'Insert image',
    message: 'Choose image source',
    options: [
      { value: 'drive', label: 'From Drive', hint: 'Pick an image in Drive root' },
      { value: 'url', label: 'From URL', hint: 'https:// or data:' },
    ],
    value: 'drive',
  });
  if (mode == null) return;
  let imageUrl = '';
  let alt = '';
  let objectId = '';
  if (mode === 'drive') {
    const picked = await pickDriveImageObject();
    if (!picked) return;
    if (!picked.url) {
      const url = await EraOfficeShell.promptText({
        title: 'Image URL',
        label: 'URL (https:// or data:)',
        value: '',
        placeholder: 'https://',
      });
      if (url == null || !url.trim()) return;
      imageUrl = url.trim();
    } else {
      imageUrl = picked.url;
      alt = picked.alt || '';
      objectId = picked.object_id || '';
    }
  } else {
    const url = await EraOfficeShell.promptText({
      title: 'Image URL',
      label: 'URL (https:// or data:)',
      value: '',
      placeholder: 'https://',
    });
    if (url == null || !String(url).trim()) return;
    imageUrl = String(url).trim();
  }
  insertBlockAfter({
    id: newBlockId(),
    block_type: 'image',
    heading_level: 0,
    inlines: [{ text: alt, bold: false, italic: false, underline: false }],
    image_url: imageUrl,
    image_object_id: objectId || undefined,
  });
  setAuthStatus('Image inserted', false);
}

function insertTableBlock() {
  openTableInsertDialog();
}

async function insertBookmarkBlock() {
  const name = await EraOfficeShell.promptText({
    title: 'Insert bookmark',
    label: 'Bookmark name',
    value: 'mark1',
  });
  if (name == null || !name.trim()) return;
  insertBlockAfter({
    id: newBlockId(),
    block_type: 'bookmark',
    heading_level: 0,
    bookmark_name: name.trim(),
    inlines: [{ text: 'Bookmark: ' + name.trim(), bold: false, italic: false, underline: false }],
  });
  setAuthStatus('Bookmark inserted — use Go to bookmark / TOC', false);
}

function collectHeadingBlocks() {
  return (docState.blocks || []).filter(
    (b) =>
      b.block_type === 'heading' ||
      (b.heading_level > 0) ||
      (b.style_name && /^h[1-6]$/i.test(b.style_name))
  );
}

function tocTextFromHeadings(headings) {
  return headings.length
    ? headings.map((b, i) => i + 1 + '. ' + blockText(b) + '\t#' + b.id).join('\n')
    : '(No headings yet — add Heading 1–3 then refresh TOC)';
}

function refreshTocBlock(block) {
  pushUndo();
  const headings = collectHeadingBlocks();
  block.inlines = [
    { text: 'Contents\n' + tocTextFromHeadings(headings), bold: false, italic: false, underline: false },
  ];
  renderBlocks();
  scheduleAutosave();
  setAuthStatus('TOC refreshed', false);
}

function renderTocLinks(el, block) {
  const pre = el.querySelector('pre');
  if (!pre) return;
  const lines = blockText(block).split('\n');
  pre.innerHTML = '';
  lines.forEach((line) => {
    const parts = line.split('\t#');
    const row = document.createElement('div');
    if (parts.length === 2) {
      const a = document.createElement('a');
      a.href = '#';
      a.className = 'doc-toc-link';
      a.textContent = parts[0];
      a.addEventListener('click', (ev) => {
        ev.preventDefault();
        const target = blocksEl.querySelector('.doc-block[data-block-id="' + parts[1] + '"]');
        if (target) {
          target.scrollIntoView({ block: 'center' });
          target.classList.add('find-highlight');
          setTimeout(() => target.classList.remove('find-highlight'), 1200);
        }
      });
      row.appendChild(a);
    } else {
      row.textContent = line;
    }
    pre.appendChild(row);
  });
}

function insertTocBlock() {
  const headings = collectHeadingBlocks();
  insertBlockAfter({
    id: newBlockId(),
    block_type: 'toc',
    heading_level: 0,
    inlines: [
      {
        text: 'Contents\n' + tocTextFromHeadings(headings),
        bold: false,
        italic: false,
        underline: false,
      },
    ],
  });
  setAuthStatus('Table of contents inserted', false);
}

async function goToBookmark() {
  const bms = (docState.blocks || []).filter((b) => b.block_type === 'bookmark');
  if (!bms.length) {
    setAuthStatus('No bookmarks in document', true);
    return;
  }
  const name = await EraOfficeShell.chooseOption({
    title: 'Go to bookmark',
    options: bms.map((b) => ({
      value: b.bookmark_name || b.id,
      label: b.bookmark_name || b.id,
      hint: b.id,
    })),
    value: bms[0].bookmark_name || bms[0].id,
  });
  if (!name || !String(name).trim()) return;
  const bm = bms.find((b) => (b.bookmark_name || b.id) === String(name).trim());
  if (!bm) {
    setAuthStatus('Bookmark not found', true);
    return;
  }
  const el = blocksEl.querySelector('.doc-block[data-block-id="' + bm.id + '"]');
  if (el) {
    el.scrollIntoView({ block: 'center' });
    setAuthStatus('Jumped to bookmark ' + String(name).trim(), false);
  }
}

function openReplaceDialog() {
  const dlg = document.getElementById('replaceDlg');
  if (!dlg) return;
  const findEl = document.getElementById('replaceFind');
  const findInput = document.getElementById('findInput');
  if (findEl && findInput) findEl.value = findInput.value || '';
  if (typeof dlg.showModal === 'function') dlg.showModal();
}

function doFindReplace(mode) {
  const find = (document.getElementById('replaceFind').value || '').trim();
  const withText = document.getElementById('replaceWith').value || '';
  if (!find) return;
  pushUndo();
  let n = 0;
  for (const b of docState.blocks || []) {
    if (!b.inlines || !b.inlines.length) continue;
    const t = blockText(b);
    if (!t.includes(find)) continue;
    if (mode === 'one' && n > 0) break;
    const next = mode === 'all' ? t.split(find).join(withText) : t.replace(find, withText);
    b.inlines = [spanFromMarks(next, blockMarks(b))];
    n += mode === 'all' ? t.split(find).length - 1 : 1;
    if (mode === 'one') break;
  }
  renderBlocks();
  scheduleAutosave();
  setAuthStatus(n ? 'Replaced ' + n + ' occurrence(s)' : 'No matches', !n);
}

async function openVersionsDialog() {
  if (!docId) return setAuthStatus('Open a document first', true);
  const dlg = document.getElementById('versionsDlg');
  const list = document.getElementById('versionsDlgList');
  if (!dlg || !list) return;
  list.innerHTML = '<li class="era-hint">Loading…</li>';
  if (typeof dlg.showModal === 'function') dlg.showModal();
  try {
    const res = await officeFetch('/api/v1/drive/objects/' + encodeURIComponent(docId) + '/versions', {
      headers: authHeaders(),
    });
    if (!res.ok) {
      list.innerHTML = '<li class="era-hint">Failed: ' + res.status + '</li>';
      return;
    }
    const data = await res.json();
    const vs = data.versions || data || [];
    if (!vs.length) {
      list.innerHTML = '<li class="era-hint">No versions yet</li>';
      return;
    }
    list.innerHTML = '';
    vs.forEach((v) => {
      const li = document.createElement('li');
      li.textContent =
        'v' +
        (v.version || v.Version || '?') +
        (v.created_at || v.CreatedAt ? ' · ' + (v.created_at || v.CreatedAt) : '') +
        (v.size_bytes != null ? ' · ' + v.size_bytes + ' B' : '');
      list.appendChild(li);
    });
  } catch (_) {
    list.innerHTML = '<li class="era-hint">Failed to load versions</li>';
  }
}

async function setBlockLanguage() {
  const sel = selectedBlock();
  if (!sel) return setAuthStatus('Select a paragraph first', true);
  const lang = await EraOfficeShell.promptText({
    title: 'Language',
    label: 'Language (BCP-47)',
    value: sel.block.lang || 'en',
    placeholder: 'en',
  });
  if (lang == null) return;
  pushUndo();
  sel.block.lang = lang.trim() || undefined;
  scheduleAutosave();
  setAuthStatus('Language: ' + (sel.block.lang || 'default'), false);
}

const SPELL_IGNORE = new Set();
const SPELL_WORDLISTS = {
  en: {
    the: 1, and: 1, for: 1, that: 1, with: 1, this: 1, from: 1, have: 1,
    document: 1, office: 1, era: 1, drive: 1, table: 1, sheet: 1, presentation: 1,
    project: 1, paragraph: 1, heading: 1, comment: 1, version: 1, export: 1,
  },
  ru: { это: 1, для: 1, документ: 1, офис: 1, таблица: 1, проект: 1 },
  az: { bu: 1, və: 1, sənəd: 1, ofis: 1, cədvəl: 1 },
};

function runSpellingLite() {
  const docLang = ((docState.lang || (docState.page && docState.page.lang) || 'en') + '').slice(0, 2).toLowerCase();
  const common = Object.assign({}, SPELL_WORDLISTS.en, SPELL_WORDLISTS[docLang] || {});
  let flags = 0;
  document.querySelectorAll('.doc-block').forEach((el) => {
    const id = el.dataset.blockId;
    const block = (docState.blocks || []).find((b) => b.id === id);
    if (!block || !block.inlines) return;
    if (block.block_type === 'image' || block.block_type === 'table') return;
    const lang = ((block.lang || docLang) + '').slice(0, 2).toLowerCase();
    const dict = Object.assign({}, common, SPELL_WORDLISTS[lang] || {});
    const words = blockText(block).split(/(\s+)/);
    el.innerHTML = words
      .map((w) => {
        const key = w.toLowerCase();
        if (SPELL_IGNORE.has(key)) return escapeHtml(w);
        if (!/^[A-Za-zÀ-ÿА-яƏəŞşÇçĞğİıÖöÜü]{4,}$/u.test(w) || dict[key]) return escapeHtml(w);
        if (/[aeiou]{3,}|[^aeiouаеёиоуыэюяə]{5,}/i.test(w)) {
          flags++;
          return (
            '<span class="spell-flag" title="Possible typo — dblclick to ignore" data-word="' +
            escapeHtml(key) +
            '">' +
            escapeHtml(w) +
            '</span>'
          );
        }
        return escapeHtml(w);
      })
      .join('');
    el.querySelectorAll('.spell-flag').forEach((span) => {
      span.addEventListener('dblclick', () => {
        SPELL_IGNORE.add(span.dataset.word || '');
        span.classList.remove('spell-flag');
        setAuthStatus('Ignored: ' + (span.dataset.word || ''), false);
      });
    });
  });
  setAuthStatus(
    flags
      ? 'Spelling (' + docLang + '): ' + flags + ' issue(s) — dblclick to ignore'
      : 'Spelling (' + docLang + '): no issues flagged',
    false
  );
}

const colorInput = document.getElementById('colorInput');
if (colorInput) colorInput.addEventListener('change', () => setTextColor(colorInput.value));
const highlightInput = document.getElementById('highlightInput');
if (highlightInput) highlightInput.addEventListener('change', () => setHighlightColor(highlightInput.value));
const replaceDlg = document.getElementById('replaceDlg');
if (replaceDlg) {
  replaceDlg.addEventListener('close', () => {
    const v = replaceDlg.returnValue;
    if (v === 'one' || v === 'all') doFindReplace(v);
  });
}

const docsMenuHandlers = {
  'file.new': () => createDocument().catch(() => {}),
  'file.open': () => {
    location.href = '/drive/';
  },
  'file.share': () => openShareDialog(),
  'file.import': doImport,
  'file.export': doExport,
  'file.pdf': () => {
    setAuthStatus('Print / Save as PDF — use the browser Print dialog (no server PDF)', false);
    window.print();
  },
  'file.versions': () => openVersionsDialog().catch(() => {}),
  'file.print': () => {
    setAuthStatus('Print / Save as PDF — browser dialog', false);
    window.print();
  },
  'file.pageSetup': openPageSetup,
  'file.save': doSave,
  'edit.undo': undoEdit,
  'edit.redo': redoEdit,
  'edit.cut': () => document.execCommand('cut'),
  'edit.copy': () => document.execCommand('copy'),
  'edit.paste': async () => {
    try {
      const t = await navigator.clipboard.readText();
      if (t == null) {
        setAuthStatus('Rich HTML paste from Word is out of MVP — use Paste plain', true);
        return;
      }
      await pastePlainText(t);
    } catch (_) {
      setAuthStatus('Paste failed — try Paste plain (HTML from Word not supported)', true);
    }
  },
  'edit.pastePlain': async () => {
    try {
      const t = await navigator.clipboard.readText();
      await pastePlainText(t);
    } catch (_) {
      setAuthStatus('Paste plain failed (clipboard permission)', true);
    }
  },
  'edit.selectAll': () => {
    const range = document.createRange();
    range.selectNodeContents(blocksEl);
    const s = window.getSelection();
    s.removeAllRanges();
    s.addRange(range);
  },
  'edit.find': () => {
    const input = document.getElementById('findInput');
    if (input) input.focus();
  },
  'edit.replace': openReplaceDialog,
  'view.printLayout': () => {
    blocksEl && blocksEl.classList.add('era-doc-page');
    setAuthStatus('Print layout', false);
  },
  'view.wordCount': openWordCountDialog,
  'view.suggest': () => {
    if (!docId) {
      setAuthStatus('Open a document first', true);
      return;
    }
    docState.track_changes = !docState.track_changes;
    const chk = document.getElementById('trackChangesChk');
    if (chk) chk.checked = !!docState.track_changes;
    document.body.classList.toggle('era-suggesting', !!docState.track_changes);
    scheduleAutosave();
    renderBlocks();
    setAuthStatus(
      docState.track_changes
        ? 'Suggesting on — edits recorded as revisions'
        : 'Suggesting off',
      false
    );
  },
  'view.fullscreen': () => {
    const el =
      document.querySelector('.era-doc-page') ||
      document.querySelector('main.era-main') ||
      document.documentElement;
    if (!document.fullscreenElement) {
      if (el && el.requestFullscreen) {
        el.requestFullscreen().catch(() => setAuthStatus('Fullscreen denied', true));
        setAuthStatus('Fullscreen (Esc to exit)', false);
      } else {
        setAuthStatus('Fullscreen not supported', true);
      }
    } else if (document.exitFullscreen) {
      document.exitFullscreen().catch(() => {});
    }
  },
  'insert.link': () => setLink(),
  'insert.comment': () => addComment(),
  'insert.pageBreak': insertPageBreak,
  'insert.header': () => showHeaderFooter('header'),
  'insert.footer': () => showHeaderFooter('footer'),
  'insert.pageNumbers': () => showHeaderFooter('numbers'),
  'insert.image': insertImageBlock,
  'insert.table': openTableInsertDialog,
  'insert.toc': insertTocBlock,
  'insert.bookmark': insertBookmarkBlock,
  'edit.gotoBookmark': goToBookmark,
  'format.bold': () => toggleMark('bold'),
  'format.italic': () => toggleMark('italic'),
  'format.underline': () => toggleMark('underline'),
  'format.strike': () => toggleMark('strike'),
  'format.font': () => {
    const el = document.getElementById('fontSelect');
    if (el) {
      el.focus();
      try {
        if (typeof el.showPicker === 'function') el.showPicker();
      } catch (_) {}
      return;
    }
    void (async () => {
      const f = await EraOfficeShell.promptText({
        title: 'Font',
        label: 'Font family',
        value: 'Arial',
      });
      if (f != null) setFontFamily(f.trim());
    })();
  },
  'format.size': () => {
    const el = document.getElementById('sizeSelect');
    if (el) {
      el.focus();
      try {
        if (typeof el.showPicker === 'function') el.showPicker();
      } catch (_) {}
      return;
    }
    void (async () => {
      const s = await EraOfficeShell.promptText({
        title: 'Font size',
        label: 'Size (pt)',
        value: '12',
      });
      if (s != null) setFontSize(s.trim());
    })();
  },
  'format.color': () => {
    const el = colorInput || document.getElementById('colorInput');
    if (el) {
      el.focus();
      try {
        if (typeof el.showPicker === 'function') el.showPicker();
        else el.click();
      } catch (_) {
        try {
          el.click();
        } catch (__) {}
      }
      return;
    }
    void (async () => {
      const c = await EraOfficeShell.promptText({
        title: 'Text color',
        label: 'Color (#hex)',
        value: '#c62828',
      });
      if (c != null) setTextColor(c.trim());
    })();
  },
  'format.highlight': () => {
    const el = highlightInput || document.getElementById('highlightInput');
    if (el) {
      el.focus();
      try {
        if (typeof el.showPicker === 'function') el.showPicker();
        else el.click();
      } catch (_) {
        try {
          el.click();
        } catch (__) {}
      }
      return;
    }
    void (async () => {
      const c = await EraOfficeShell.promptText({
        title: 'Highlight',
        label: 'Color (#hex)',
        value: '#fff59d',
      });
      if (c != null) setHighlightColor(c.trim());
    })();
  },
  'format.language': setBlockLanguage,
  'format.title': () => applyStyle('title'),
  'format.h1': () => applyStyle('h1'),
  'format.h2': () => applyStyle('h2'),
  'format.h3': () => applyStyle('h3'),
  'format.h4': () => applyStyle('h4'),
  'format.h5': () => applyStyle('h5'),
  'format.h6': () => applyStyle('h6'),
  'edit.formatPainter': armFormatPainter,
  'view.showMarks': toggleShowMarks,
  'insert.symbol': insertSymbol,
  'insert.hr': insertHorizontalLine,
  'format.super': () => toggleScript('super'),
  'format.sub': () => toggleScript('sub'),
  'format.normal': () => applyStyle('normal'),
  'format.quote': () => applyStyle('quote'),
  'format.caption': () => applyStyle('caption'),
  'format.alignLeft': () => setAlign('left'),
  'format.alignCenter': () => setAlign('center'),
  'format.alignRight': () => setAlign('right'),
  'format.alignJustify': () => setAlign('justify'),
  'format.indentDec': () => bumpIndent(-10),
  'format.indentInc': () => bumpIndent(10),
  'format.lineSpacing': () => {
    const el = document.getElementById('lineSpacingSelect');
    if (el) {
      el.focus();
      try {
        if (typeof el.showPicker === 'function') el.showPicker();
      } catch (_) {}
      return;
    }
    void (async () => {
      const v = await EraOfficeShell.chooseOption({
        title: 'Line spacing',
        options: [
          { value: '1.0', label: 'Single' },
          { value: '1.15', label: '1.15' },
          { value: '1.5', label: '1.5' },
          { value: '2.0', label: 'Double' },
        ],
        value: '1.15',
      });
      if (v) setLineSpacing(String(v).trim());
    })();
  },
  'format.paraSpacing': setParaSpacing,
  'format.bullet': () => setListType('bullet'),
  'format.numbered': () => setListType('ordered'),
  'format.listLevelDec': () => bumpListLevel(-1),
  'format.listLevelInc': () => bumpListLevel(1),
  'format.listMarker': changeListMarker,
  'format.listRestart': restartListNumbering,
  'format.clear': clearFormatting,
  'tools.wordCount': openWordCountDialog,
  'tools.spelling': runSpellingLite,
  'tools.summarize': () => document.getElementById('summarizeAIBtn').click(),
  'tools.rewrite': () => document.getElementById('rewriteAIBtn').click(),
  'help.shortcuts': () =>
    setAuthStatus('Ctrl+Z undo · Ctrl+Y redo · Find in toolbar · Format via menu', false),
  'help.about': () => {
    void EraOfficeShell.confirmAction({
      title: 'About ERA Documents',
      message:
        'ERA Documents — collaborative editing in your contour (not Microsoft Word).\n\n' +
        '• Co-edit via Workspace; snapshots go to Drive\n' +
        '• Section breaks are in-flow markers; columns are page layout (not Word sections)\n' +
        '• Block numbers (View) number blocks, not wrapped visual lines\n' +
        '• Spelling is a light dictionary pass — not full proofing',
      okLabel: 'OK',
      cancelLabel: 'Close',
    });
  },
};

window.docsMenuHandlers = docsMenuHandlers;
if (window.EraOfficeMenubar) {
  EraOfficeMenubar.init('#menubar', docsMenuHandlers);
}

document.addEventListener('keydown', (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'z' && !e.shiftKey) {
    e.preventDefault();
    undoEdit();
  }
  if ((e.ctrlKey || e.metaKey) && (e.key.toLowerCase() === 'y' || (e.key.toLowerCase() === 'z' && e.shiftKey))) {
    e.preventDefault();
    redoEdit();
  }
});

refreshPresence();
if (window.EraOfficeShell && EraOfficeShell.requireAuthOrRedirect) {
  if (!EraOfficeShell.requireAuthOrRedirect()) {
    /* redirecting to /login */
  }
} else if (!localStorage.getItem('era_token')) {
  location.href =
    '/login?next=' + encodeURIComponent(location.pathname + location.search);
}
if (window.EraOfficeShell) {
  if (EraOfficeShell.wireSessionWatch) EraOfficeShell.wireSessionWatch();
  if (EraOfficeShell.wireCommentsToggle) EraOfficeShell.wireCommentsToggle(true);
}

initDocRuler();
const tableInsertDlg = document.getElementById('tableInsertDlg');
if (tableInsertDlg) {
  tableInsertDlg.addEventListener('close', () => {
    if (tableInsertDlg.returnValue === 'ok') insertTableFromDialog();
  });
}

docId = pathDocId();
if (docId) {
  loadDoc(docId).catch(() => {});
} else if (localStorage.getItem('era_token')) {
  setDocMeta('Creating Untitled…');
  setDocTitleUI('Untitled', false);
  createDocument().catch(() => {
    blocksEl.innerHTML =
      '<p class="era-empty">Could not create a document. Use File → New.</p>';
    setDocMeta('No document open');
  });
}
