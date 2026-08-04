if (window.EraOfficeShell) {
  if (EraOfficeShell.markActiveNav) EraOfficeShell.markActiveNav('pres');
  if (EraOfficeShell.mountNav) EraOfficeShell.mountNav(document);
  else if (EraOfficeShell.mountIcons) EraOfficeShell.mountIcons(document);
  if (EraOfficeShell.syncUserChip) EraOfficeShell.syncUserChip();
  if (EraOfficeShell.wireTeDisclaimer) {
    EraOfficeShell.wireTeDisclaimer(document.getElementById('teBanner'), 'era_te_dismiss_pres');
  }
}

const authStatus = document.getElementById('authStatus');
const deckMeta = document.getElementById('deckMeta');
const filmstrip = document.getElementById('filmstrip');
const workspace = document.getElementById('workspace');
const emptyHint = document.getElementById('emptyHint');
const slideTitle = document.getElementById('slideTitle');
const slideBody = document.getElementById('slideBody');
const slideBody2 = document.getElementById('slideBody2');
const slidePos = document.getElementById('slidePos');
const slideCanvas = document.querySelector('.slide-canvas');
const layoutSelect = document.getElementById('layoutSelect');
const presentOverlay = document.getElementById('presentOverlay');
const speakerNotes = document.getElementById('speakerNotes');

let deckId = null;
/** @type {{ name?: string, version?: number, theme_background?: string, default_layout?: string, master_title_placeholder?: string, master_body_placeholder?: string, slides: object[] }} */
let deck = { slides: [], version: 0 };
let slideIndex = 0;
let saveTimer = null;
let undoStack = [];
let redoStack = [];
let presenting = false;
let undoDebounceTimer = null;
let typingMarks = null;
let activeFrameKey = 'title';
let activeBlockId = null;
const RT = () => window.EraRichText;

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

function openShareDeck() {
  const dlg = document.getElementById('shareDlg');
  const input = document.getElementById('shareLinkInput');
  const driveLink = document.getElementById('shareDriveLink');
  const hint = document.getElementById('shareAclHint');
  if (!dlg || !deckId) {
    if (!deckId) setAuthStatus('Open a deck first', true);
    return;
  }
  const url = location.origin + '/presentations/' + encodeURIComponent(deckId);
  if (input) input.value = url;
  if (hint) {
    hint.textContent =
      'Copy link · ACL is managed in Drive (Share on the Drive object). This dialog does not change permissions.';
  }
  if (driveLink) {
    driveLink.href = '/drive/?share=' + encodeURIComponent(deckId);
    driveLink.textContent = 'Manage ACL in Drive';
  }
  if (typeof dlg.showModal === 'function') dlg.showModal();
}

function pathDeckId() {
  const parts = location.pathname.replace(/\/$/, '').split('/');
  const id = parts[parts.length - 1];
  return id && id !== 'presentations' ? id : null;
}

function currentSlide() {
  return deck.slides[slideIndex] || null;
}

function snapshotDeck() {
  return JSON.stringify({
    slides: deck.slides,
    slideIndex,
    theme_background: deck.theme_background,
    default_layout: deck.default_layout,
    master_title_placeholder: deck.master_title_placeholder,
    master_body_placeholder: deck.master_body_placeholder,
  });
}

function applySnapshot(snap) {
  deck.slides = snap.slides;
  deck.theme_background = snap.theme_background;
  deck.default_layout = snap.default_layout;
  deck.master_title_placeholder = snap.master_title_placeholder;
  deck.master_body_placeholder = snap.master_body_placeholder;
  slideIndex = Math.min(snap.slideIndex, Math.max(0, deck.slides.length - 1));
}

function pushUndo() {
  try {
    undoStack.push(snapshotDeck());
    if (undoStack.length > 30) undoStack.shift();
    redoStack = [];
  } catch (_) {}
}

/** Debounced checkpoint for typing (avoids stack noise). */
function pushUndoDebounced() {
  if (undoDebounceTimer) return;
  undoDebounceTimer = setTimeout(() => {
    undoDebounceTimer = null;
    pushUndo();
  }, 400);
}

function undoEdit() {
  if (!undoStack.length) {
    setAuthStatus('Nothing to undo', true);
    return;
  }
  try {
    redoStack.push(snapshotDeck());
    if (redoStack.length > 30) redoStack.shift();
    applySnapshot(JSON.parse(undoStack.pop()));
    renderAll();
    scheduleSave();
    setAuthStatus('Undo', false);
  } catch (_) {}
}

function redoEdit() {
  if (!redoStack.length) {
    setAuthStatus('Nothing to redo', true);
    return;
  }
  try {
    undoStack.push(snapshotDeck());
    if (undoStack.length > 30) undoStack.shift();
    applySnapshot(JSON.parse(redoStack.pop()));
    renderAll();
    scheduleSave();
    setAuthStatus('Redo', false);
  } catch (_) {}
}

function ensureSlideFrames(s) {
  if (!s || !RT()) return s;
  const mk = (plain) => RT().emptyFrame(plain || '');
  if (!s.title_frame || !s.title_frame.blocks) {
    s.title_frame = mk(typeof s.title === 'string' ? s.title : 'Title');
  }
  if (!s.body_frame || !s.body_frame.blocks) {
    s.body_frame = mk(typeof s.body === 'string' ? s.body : '');
  }
  if (!s.body2_frame || !s.body2_frame.blocks) {
    s.body2_frame = mk(typeof s.body2 === 'string' ? s.body2 : '');
  }
  delete s.title_bold;
  delete s.body_bold;
  delete s.title_font;
  delete s.body_font;
  delete s.title_align;
  delete s.body_align;
  delete s.title_font_pt;
  delete s.body_font_pt;
  return s;
}

function frameOf(slide, key) {
  ensureSlideFrames(slide);
  if (key === 'body2') return slide.body2_frame;
  if (key === 'body') return slide.body_frame;
  return slide.title_frame;
}

function slidePlainTitle(s) {
  if (!s) return '';
  ensureSlideFrames(s);
  return RT() ? RT().framePlain(s.title_frame) : s.title || '';
}

function flushEditorToModel() {
  const s = currentSlide();
  if (!s) return;
  ensureSlideFrames(s);
  if (speakerNotes) s.notes = speakerNotes.value || '';
  if (layoutSelect) s.layout = layoutSelect.value || 'title_body';
}

async function sendFrameOp(frameKey, op) {
  if (!deckId) return;
  const s = currentSlide();
  if (!s) return;
  try {
    const res = await officeFetch(
      '/api/v1/presentations/' + encodeURIComponent(deckId) + '/frame-op',
      {
        method: 'POST',
        headers: authHeaders({ 'Content-Type': 'application/json' }),
        body: JSON.stringify({
          slide_id: s.id,
          frame: frameKey,
          op,
          base_version: deck.version || 0,
        }),
      }
    );
    if (res.status === 409) {
      await loadDeck(deckId);
      setAuthStatus('Deck reloaded (conflict)', true);
      return;
    }
    if (!res.ok) {
      scheduleSave();
      return;
    }
    const data = await res.json();
    if (data.deck) {
      deck.version = data.version || data.deck.version || 0;
      // Keep local slides; version only — avoid clobbering caret mid-type.
    } else if (data.version != null) {
      deck.version = data.version;
    }
  } catch (_) {
    scheduleSave();
  }
}

function applyLayoutClass(layout) {
  if (!slideCanvas) return;
  slideCanvas.classList.remove(
    'layout-title_body',
    'layout-title_only',
    'layout-section',
    'layout-two_column'
  );
  const l = layout || 'title_body';
  slideCanvas.classList.add('layout-' + l);
}

function applyBackground() {
  if (!slideCanvas) return;
  const s = currentSlide();
  const bg = (s && s.background) || deck.theme_background || '#ffffff';
  slideCanvas.style.background = bg;
  slideCanvas.style.color = '#202124';}

function renderFilmstrip() {
  filmstrip.innerHTML = '';
  deck.slides.forEach((s, i) => {
    const li = document.createElement('li');
    li.dataset.index = String(i);
    li.draggable = true;
    if (i === slideIndex) li.className = 'active';
    li.innerHTML =
      '<span class="n">' +
      (i + 1) +
      '.</span>' +
      (slidePlainTitle(s) || '(untitled)');
    li.addEventListener('click', () => {
      flushEditorToModel();
      slideIndex = i;
      renderAll();
    });
    li.addEventListener('dragstart', (ev) => {
      ev.dataTransfer.setData('text/plain', String(i));
    });
    li.addEventListener('dragover', (ev) => ev.preventDefault());
    li.addEventListener('drop', (ev) => {
      ev.preventDefault();
      const from = parseInt(ev.dataTransfer.getData('text/plain'), 10);
      const to = i;
      if (isNaN(from) || from === to) return;
      pushUndo();
      flushEditorToModel();
      const [item] = deck.slides.splice(from, 1);
      deck.slides.splice(to, 0, item);
      slideIndex = to;
      renderAll();
      scheduleSave();
      setAuthStatus('Slide reordered', false);
    });
    filmstrip.appendChild(li);
  });
}

function renderStage() {
  const s = currentSlide();
  if (!s) {
    slideTitle.textContent = '';
    slideBody.textContent = '';
    if (slideBody2) slideBody2.textContent = '';
    if (speakerNotes) speakerNotes.value = '';
    slidePos.textContent = 'No slides';
    return;
  }
  ensureSlideFrames(s);
  renderFrameEl(slideTitle, s, 'title', 'Title');
  renderFrameEl(slideBody, s, 'body', 'Body');
  if (slideBody2) renderFrameEl(slideBody2, s, 'body2', 'Column 2');
  if (speakerNotes) speakerNotes.value = s.notes || '';
  slidePos.textContent = 'Slide ' + (slideIndex + 1) + ' / ' + deck.slides.length;
  const layout = s.layout || 'title_body';
  applyLayoutClass(layout);
  applyBackground();
  syncSlideImage();
  if (layoutSelect) layoutSelect.value = layout;
}

function renderFrameEl(el, slide, key, placeholder) {
  if (!el || !RT()) return;
  const frame = frameOf(slide, key);
  el.innerHTML = '';
  el.dataset.frame = key;
  (frame.blocks || []).forEach((block, idx) => {
    const div = document.createElement('div');
    div.className = 'pres-block';
    div.contentEditable = 'true';
    div.dataset.blockId = block.id;
    if (idx === 0) div.dataset.placeholder = placeholder;
    if (block.align) {
      div.style.textAlign = block.align === 'justify' ? 'justify' : block.align;
    }
    if (block.list_type === 'bullet') div.style.listStyle = 'disc';
    div.innerHTML = RT().renderInlineHtml(block) || '';
    div.addEventListener('focus', () => {
      activeFrameKey = key;
      activeBlockId = block.id;
    });
    div.addEventListener('input', () => onPresBlockInput(key, block.id, div));
    div.addEventListener('keydown', (e) => onPresBlockKeydown(e, key, block.id, div));
    el.appendChild(div);
  });
}

function onPresBlockInput(frameKey, blockId, el) {
  const s = currentSlide();
  if (!s || !RT()) return;
  const frame = frameOf(s, frameKey);
  const block = (frame.blocks || []).find((b) => b.id === blockId);
  if (!block) return;
  const prev = RT().blockText(block);
  let next = el.innerText || '';
  if (next.endsWith('\n') && !prev.endsWith('\n')) next = next.slice(0, -1);
  if (prev === next) return;
  pushUndo();
  const d = RT().diffText(prev, next);
  if (d.deletedEnd > d.start) {
    block.inlines = RT().deleteRangePreserving(block.inlines, d.start, d.deletedEnd);
    sendFrameOp(frameKey, {
      type: 'delete_range',
      block_id: blockId,
      start: d.start,
      end: d.deletedEnd,
    });
  }
  if (d.inserted) {
    block.inlines = RT().insertTextPreserving(block.inlines, d.start, d.inserted, typingMarks);
    const op = {
      type: 'insert_text',
      block_id: blockId,
      offset: d.start,
      text: d.inserted,
    };
    if (typingMarks) op.marks = RT().spanFromMarks('', typingMarks);
    sendFrameOp(frameKey, op);
  }
  scheduleSave();
}

function onPresBlockKeydown(e, frameKey, blockId, el) {
  const s = currentSlide();
  if (!s || !RT()) return;
  const frame = frameOf(s, frameKey);
  const block = (frame.blocks || []).find((b) => b.id === blockId);
  if (!block) return;

  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    const offs = RT().selectionOffsetsInBlock(el) || {
      start: RT().blockText(block).length,
      end: RT().blockText(block).length,
    };
    pushUndo();
    const newId = RT().newId('b');
    const parts = RT().splitInlinesAt(block.inlines, offs.start);
    block.inlines = parts.left.length ? parts.left : [RT().spanFromMarks('', {})];
    const neu = RT().copyBlockShell(block, newId, parts.right);
    const idx = frame.blocks.findIndex((b) => b.id === blockId);
    frame.blocks.splice(idx + 1, 0, neu);
    sendFrameOp(frameKey, {
      type: 'split_block',
      block_id: blockId,
      offset: offs.start,
      new_block_id: newId,
    });
    renderStage();
    focusPresBlock(frameKey, newId, 0);
    scheduleSave();
    return;
  }

  if (e.key === 'Enter' && e.shiftKey) {
    e.preventDefault();
    const offs = RT().selectionOffsetsInBlock(el) || {
      start: RT().blockText(block).length,
      end: RT().blockText(block).length,
    };
    pushUndo();
    if (offs.end > offs.start) {
      block.inlines = RT().deleteRangePreserving(block.inlines, offs.start, offs.end);
      sendFrameOp(frameKey, {
        type: 'delete_range',
        block_id: blockId,
        start: offs.start,
        end: offs.end,
      });
    }
    block.inlines = RT().insertTextPreserving(block.inlines, offs.start, '\n', typingMarks);
    sendFrameOp(frameKey, {
      type: 'insert_text',
      block_id: blockId,
      offset: offs.start,
      text: '\n',
    });
    el.innerHTML = RT().renderInlineHtml(block);
    scheduleSave();
    return;
  }

  if (e.key === 'Backspace') {
    const offs = RT().selectionOffsetsInBlock(el);
    if (!offs || offs.start !== offs.end || offs.start !== 0) return;
    const idx = frame.blocks.findIndex((b) => b.id === blockId);
    if (idx <= 0) return;
    e.preventDefault();
    pushUndo();
    const prevB = frame.blocks[idx - 1];
    const mergeAt = RT().blockText(prevB).length;
    prevB.inlines = RT().coalesceInlines((prevB.inlines || []).concat(block.inlines || []));
    frame.blocks.splice(idx, 1);
    sendFrameOp(frameKey, { type: 'merge_blocks', block_id: blockId, with_previous: true });
    renderStage();
    focusPresBlock(frameKey, prevB.id, mergeAt);
    scheduleSave();
  }
}

function focusPresBlock(frameKey, blockId, offset) {
  const root =
    frameKey === 'body2' ? slideBody2 : frameKey === 'body' ? slideBody : slideTitle;
  if (!root) return;
  const el = root.querySelector('.pres-block[data-block-id="' + blockId + '"]');
  if (!el) return;
  el.focus();
  const sel = window.getSelection();
  if (!sel) return;
  const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT, null);
  let remaining = Math.max(0, offset || 0);
  let node = walker.nextNode();
  let last = null;
  while (node) {
    last = node;
    if (remaining <= node.textContent.length) {
      const range = document.createRange();
      range.setStart(node, remaining);
      range.collapse(true);
      sel.removeAllRanges();
      sel.addRange(range);
      return;
    }
    remaining -= node.textContent.length;
    node = walker.nextNode();
  }
  if (last) {
    const range = document.createRange();
    range.setStart(last, last.textContent.length);
    range.collapse(true);
    sel.removeAllRanges();
    sel.addRange(range);
  }
}

function renderAll() {
  const open = !!deckId && deck.slides.length > 0;
  workspace.hidden = !open;
  if (emptyHint) emptyHint.hidden = open;
  renderFilmstrip();
  renderStage();
}

function scheduleSave() {
  const pill = document.getElementById('savePill');
  if (pill && window.EraOfficeShell && EraOfficeShell.setSavePill) {
    EraOfficeShell.setSavePill(pill, 'dirty');
  }
  clearTimeout(saveTimer);
  saveTimer = setTimeout(() => {
    saveDeck(true).catch(() => {});
  }, 600);
}

function requireToken() {
  if (localStorage.getItem('era_token')) return true;
  const login =
    (window.EraOfficeShell && EraOfficeShell.loginUrl && EraOfficeShell.loginUrl()) ||
    '/login?next=' + encodeURIComponent(location.pathname);
  location.href = login;
  return false;
}

async function saveDeck(quiet) {
  if (!deckId) return false;
  if (!requireToken()) {
    const pill = document.getElementById('savePill');
    if (pill && window.EraOfficeShell && EraOfficeShell.setSavePill) {
      EraOfficeShell.setSavePill(pill, 'err', 'Sign in');
    }
    return false;
  }
  flushEditorToModel();
  const titleEl = document.getElementById('deckTitle');
  if (titleEl && titleEl.value.trim()) {
    deck.name = titleEl.value.trim();
  }
  const pill = document.getElementById('savePill');
  if (pill && window.EraOfficeShell && EraOfficeShell.setSavePill) {
    EraOfficeShell.setSavePill(pill, 'saving');
  }
  const res = await officeFetch('/api/v1/presentations/' + encodeURIComponent(deckId), {
    method: 'PUT',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify(deck),
  });
  if (!res.ok) {
    if (window.EraOfficeShell && EraOfficeShell.handleUnauthorized && EraOfficeShell.handleUnauthorized(res)) {
      return false;
    }
    const msg =
      res.status === 403
        ? 'Save failed: access denied'
        : 'Save failed: ' + res.status;
    setAuthStatus(msg, true);
    if (pill && window.EraOfficeShell && EraOfficeShell.setSavePill) {
      EraOfficeShell.setSavePill(pill, 'err', 'Save failed');
    }
    return false;
  }
  if (!quiet) setAuthStatus('Saved', false);
  if (pill && window.EraOfficeShell && EraOfficeShell.setSavePill) {
    EraOfficeShell.setSavePill(pill, 'ok', 'Saved');
  }
  return true;
}

async function loadDeck(id) {
  if (!requireToken()) {
    deckId = null;
    renderAll();
    return;
  }
  const res = await officeFetch('/api/v1/presentations/' + encodeURIComponent(id), {
    headers: authHeaders(),
  });
  if (!res.ok) {
    if (window.EraOfficeShell && EraOfficeShell.handleUnauthorized && EraOfficeShell.handleUnauthorized(res)) {
      return;
    }
    deckId = null;
    const msg =
      res.status === 403
        ? 'Access denied for this deck'
        : 'Failed to load deck: ' + res.status;
    setAuthStatus(msg, true);
    renderAll();
    return;
  }
  deck = await res.json();
  if (!deck.slides || !deck.slides.length) {
    deck.slides = [
      {
        id: crypto.randomUUID(),
        layout: 'title_body',
        title_frame: RT() ? RT().emptyFrame('Title') : undefined,
        body_frame: RT() ? RT().emptyFrame('') : undefined,
        body2_frame: RT() ? RT().emptyFrame('') : undefined,
      },
    ];
  }
  deck.slides.forEach((s) => {
    if (!s.layout) s.layout = 'title_body';
    ensureSlideFrames(s);
  });
  if (deck.version == null) deck.version = 0;
  deckId = id;
  slideIndex = 0;
  deckMeta.textContent = 'Deck: ' + (deck.name || id);
  const titleEl = document.getElementById('deckTitle');
  if (titleEl) titleEl.value = deck.name || 'Untitled deck';
  const pill = document.getElementById('savePill');
  if (pill && window.EraOfficeShell && EraOfficeShell.setSavePill) {
    EraOfficeShell.setSavePill(pill, 'ok', 'Saved');
  }
  if (window.EraOfficeShell && EraOfficeShell.syncUserChip) EraOfficeShell.syncUserChip();
  setAuthStatus('Deck open', false);
  renderAll();
}

async function createDeck() {
  if (!requireToken()) return;
  const { tenantId, userId } = identity();
  const res = await officeFetch('/api/v1/presentations', {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({
      tenant_id: tenantId,
      user_id: userId,
      // Drive forbids duplicate names in a folder.
      name: 'Untitled-' + Date.now() + '.erap',
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
  location.href = '/presentations/' + data.drive_object_id;
}

function addSlide() {
  if (!deckId) {
    setAuthStatus('Open a deck first', true);
    return;
  }
  pushUndo();
  flushEditorToModel();
  const titlePh = (deck.master_title_placeholder || '').trim();
  const bodyPh = (deck.master_body_placeholder || '').trim();
  deck.slides.push({
    id: crypto.randomUUID(),
    notes: '',
    layout: deck.default_layout || 'title_body',
    title_frame: RT().emptyFrame(titlePh || 'New slide'),
    body_frame: RT().emptyFrame(bodyPh || ''),
    body2_frame: RT().emptyFrame(''),
  });
  slideIndex = deck.slides.length - 1;
  renderAll();
  scheduleSave();
  setAuthStatus('Slide added', false);
}

function duplicateSlide() {
  if (!deckId) {
    setAuthStatus('Open a deck first', true);
    return;
  }
  const s = currentSlide();
  if (!s) return;
  pushUndo();
  flushEditorToModel();
  ensureSlideFrames(s);
  const copy = JSON.parse(JSON.stringify(s));
  copy.id = crypto.randomUUID();
  const t = slidePlainTitle(copy) || 'Slide';
  copy.title_frame = RT().emptyFrame(t + ' (copy)');
  deck.slides.splice(slideIndex + 1, 0, copy);
  slideIndex += 1;
  renderAll();
  scheduleSave();
  setAuthStatus('Slide duplicated', false);
}

function activeBlockContext() {
  const s = currentSlide();
  if (!s || !RT()) return null;
  const frameKey = activeFrameKey || 'title';
  const frame = frameOf(s, frameKey);
  let block = (frame.blocks || []).find((b) => b.id === activeBlockId);
  if (!block && frame.blocks && frame.blocks.length) block = frame.blocks[0];
  if (!block) return null;
  const root =
    frameKey === 'body2' ? slideBody2 : frameKey === 'body' ? slideBody : slideTitle;
  const el = root && root.querySelector('.pres-block[data-block-id="' + block.id + '"]');
  return { s, frameKey, frame, block, el };
}

function selectionOrTypingRange(ctx) {
  if (!ctx || !ctx.el) return null;
  const offs = RT().selectionOffsetsInBlock(ctx.el);
  if (offs && offs.end > offs.start) return offs;
  return null;
}

function toggleBoldFormat() {
  const ctx = activeBlockContext();
  if (!ctx) {
    setAuthStatus('Select text in a slide', true);
    return;
  }
  const range = selectionOrTypingRange(ctx);
  let allOn = true;
  if (range) {
    let acc = 0;
    for (const sp of ctx.block.inlines || []) {
      const a = acc;
      const b = acc + (sp.text || '').length;
      acc = b;
      if (b <= range.start || a >= range.end) continue;
      if (!sp.bold) allOn = false;
    }
  } else {
    allOn = !!(typingMarks && typingMarks.bold);
  }
  const nextVal = !allOn;
  if (!range) {
    typingMarks = Object.assign({}, typingMarks || {}, { bold: nextVal });
    setAuthStatus('Bold ' + (nextVal ? 'on' : 'off') + ' (typing)', false);
    return;
  }
  pushUndo();
  ctx.block.inlines = RT().applyMarksRangeLocal(ctx.block.inlines, range.start, range.end, {
    bold: nextVal,
  });
  if (ctx.el) ctx.el.innerHTML = RT().renderInlineHtml(ctx.block);
  sendFrameOp(ctx.frameKey, {
    type: 'set_marks_range',
    block_id: ctx.block.id,
    start: range.start,
    end: range.end,
    bold: nextVal,
  });
  scheduleSave();
  setAuthStatus('Bold ' + (nextVal ? 'on' : 'off'), false);
}

function setAlignFormat(align) {
  const ctx = activeBlockContext();
  if (!ctx) return;
  pushUndo();
  ctx.block.align = align;
  if (ctx.el) ctx.el.style.textAlign = align === 'justify' ? 'justify' : align;
  sendFrameOp(ctx.frameKey, {
    type: 'set_block_format',
    block_id: ctx.block.id,
    align,
  });
  scheduleSave();
}

function stepFont(delta) {
  const ctx = activeBlockContext();
  if (!ctx) return;
  const range = selectionOrTypingRange(ctx);
  const base = ctx.frameKey === 'title' ? 28 : 14;
  const cur =
    (range &&
      (ctx.block.inlines || []).reduce((acc, sp) => sp.font_size_pt || acc, 0)) ||
    (typingMarks && typingMarks.font_size_pt) ||
    base;
  const next = Math.max(10, Math.min(72, cur + delta));
  if (!range) {
    typingMarks = Object.assign({}, typingMarks || {}, { font_size_pt: next });
    setAuthStatus('Font ' + next + 'pt (typing)', false);
    return;
  }
  pushUndo();
  ctx.block.inlines = RT().applyMarksRangeLocal(ctx.block.inlines, range.start, range.end, {
    font_size_pt: next,
  });
  if (ctx.el) ctx.el.innerHTML = RT().renderInlineHtml(ctx.block);
  sendFrameOp(ctx.frameKey, {
    type: 'set_marks_range',
    block_id: ctx.block.id,
    start: range.start,
    end: range.end,
    font_size_pt: next,
  });
  scheduleSave();
  setAuthStatus('Font ' + next + 'pt', false);
}

function setSlideFontFamily(fam) {
  const ctx = activeBlockContext();
  if (!ctx || !fam) return;
  const range = selectionOrTypingRange(ctx);
  if (!range) {
    typingMarks = Object.assign({}, typingMarks || {}, { font_family: fam });
    setAuthStatus('Font ' + fam + ' (typing)', false);
    return;
  }
  pushUndo();
  ctx.block.inlines = RT().applyMarksRangeLocal(ctx.block.inlines, range.start, range.end, {
    font_family: fam,
  });
  if (ctx.el) ctx.el.innerHTML = RT().renderInlineHtml(ctx.block);
  sendFrameOp(ctx.frameKey, {
    type: 'set_marks_range',
    block_id: ctx.block.id,
    start: range.start,
    end: range.end,
    font_family: fam,
  });
  scheduleSave();
  setAuthStatus('Font ' + fam, false);
}

function syncSlideImage() {
  const s = currentSlide();
  const img = document.getElementById('slideImage');
  if (!img || !s) return;
  if (s.image_url) {
    img.src = s.image_url;
    img.hidden = false;
  } else {
    img.removeAttribute('src');
    img.hidden = true;
  }
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
    };
  } catch (_) {
    return null;
  }
}

async function insertSlideImage() {
  const s = currentSlide();
  if (!s) {
    setAuthStatus('Open a deck first', true);
    return;
  }
  const mode = await EraOfficeShell.chooseOption({
    title: 'Insert image',
    message: s.image_url ? 'Current image will be replaced unless cleared.' : 'Choose image source',
    options: [
      { value: 'drive', label: 'From Drive', hint: 'Pick an image in Drive root' },
      { value: 'url', label: 'From URL', hint: 'https:// or data:' },
      { value: 'clear', label: 'Clear image', hint: 'Remove current slide image' },
    ],
    value: s.image_url ? 'url' : 'drive',
  });
  if (mode == null) return;
  if (mode === 'clear') {
    pushUndo();
    s.image_url = null;
    syncSlideImage();
    scheduleSave();
    setAuthStatus('Image cleared', false);
    return;
  }
  let url = '';
  if (mode === 'drive') {
    const picked = await pickDriveImageObject();
    if (!picked) return;
    if (!picked.url) {
      const typed = await EraOfficeShell.promptText({
        title: 'Image URL',
        label: 'URL (https:// or data:)',
        value: '',
        placeholder: 'https://',
      });
      if (typed == null || !String(typed).trim()) return;
      url = String(typed).trim();
    } else {
      url = picked.url;
    }
  } else {
    const typed = await EraOfficeShell.promptText({
      title: 'Image URL',
      label: 'URL (https:// or data:)',
      value: s.image_url || '',
      placeholder: 'https://',
    });
    if (typed == null) return;
    url = String(typed).trim();
    if (!url) {
      pushUndo();
      s.image_url = null;
      syncSlideImage();
      scheduleSave();
      setAuthStatus('Image cleared', false);
      return;
    }
  }
  pushUndo();
  s.image_url = url || null;
  syncSlideImage();
  scheduleSave();
  setAuthStatus(s.image_url ? 'Image inserted' : 'Image cleared', false);
}

function openMasterDlg() {
  if (!deckId) {
    setAuthStatus('Open a deck first', true);
    return;
  }
  const dlg = document.getElementById('masterDlg');
  const bg = document.getElementById('masterThemeBg');
  const layout = document.getElementById('masterDefaultLayout');
  const titlePh = document.getElementById('masterTitlePlaceholder');
  const bodyPh = document.getElementById('masterBodyPlaceholder');
  if (bg) bg.value = deck.theme_background || '';
  if (layout) layout.value = deck.default_layout || 'title_body';
  if (titlePh) titlePh.value = deck.master_title_placeholder || '';
  if (bodyPh) bodyPh.value = deck.master_body_placeholder || '';
  if (dlg && typeof dlg.showModal === 'function') dlg.showModal();
}

function applyMasterDlg() {
  pushUndo();
  flushEditorToModel();
  const bg = document.getElementById('masterThemeBg');
  const layout = document.getElementById('masterDefaultLayout');
  const titlePh = document.getElementById('masterTitlePlaceholder');
  const bodyPh = document.getElementById('masterBodyPlaceholder');
  const theme = ((bg && bg.value) || '').trim();
  deck.theme_background = theme || null;
  deck.default_layout = (layout && layout.value) || 'title_body';
  deck.master_title_placeholder = ((titlePh && titlePh.value) || '').trim() || null;
  deck.master_body_placeholder = ((bodyPh && bodyPh.value) || '').trim() || null;
  applyBackground();
  renderAll();
  scheduleSave();
  setAuthStatus('Master applied', false);
}

function deleteSlide() {
  if (!deckId || deck.slides.length < 2) {
    setAuthStatus('Need at least one slide', true);
    return;
  }
  pushUndo();
  deck.slides.splice(slideIndex, 1);
  if (slideIndex >= deck.slides.length) slideIndex = deck.slides.length - 1;
  renderAll();
  scheduleSave();
}

function goSlide(delta) {
  flushEditorToModel();
  const next = slideIndex + delta;
  if (next < 0 || next >= deck.slides.length) return;
  slideIndex = next;
  renderAll();
  if (presenting) renderPresentSlide();
}

function moveSlide(delta) {
  if (!deckId || deck.slides.length < 2) return;
  pushUndo();
  flushEditorToModel();
  const target = slideIndex + delta;
  if (target < 0 || target >= deck.slides.length) return;
  const tmp = deck.slides[slideIndex];
  deck.slides[slideIndex] = deck.slides[target];
  deck.slides[target] = tmp;
  slideIndex = target;
  renderAll();
  scheduleSave();
  setAuthStatus(delta < 0 ? 'Slide moved up' : 'Slide moved down', false);
}

const THEME_PRESETS = [
  { label: 'Simple Light', value: '#ffffff' },
  { label: 'Soft gray', value: '#f4f7fb' },
  { label: 'Simple Dark', value: '#1e2936' },
  { label: 'Blue wash', value: 'linear-gradient(135deg,#e8f0fe 0%,#ffffff 60%)' },
  { label: 'Warm wash', value: 'linear-gradient(160deg,#fff8f0 0%,#ffffff 70%)' },
  { label: 'Streamline', value: 'linear-gradient(180deg,#0b5fff 0%,#1e2936 55%)' },
];

const LAYOUT_PRESETS = [
  { id: 'title_body', label: 'Title and body' },
  { id: 'title_only', label: 'Title only' },
  { id: 'section', label: 'Section header' },
  { id: 'two_column', label: 'Two columns' },
];

function closePresSidePanels() {
  ['themePanel', 'transitionPanel'].forEach((id) => {
    const el = document.getElementById(id);
    if (el) el.hidden = true;
  });
  document.querySelectorAll('.era-pres-chrome-btn').forEach((b) => b.classList.remove('active'));
}

function openBackgroundDialog() {
  const dlg = document.getElementById('backgroundDlg');
  const cur = (currentSlide() && currentSlide().background) || deck.theme_background || '#ffffff';
  const css = document.getElementById('bgCssInput');
  const color = document.getElementById('bgColorInput');
  const deckChk = document.getElementById('bgDeckChk');
  if (css) css.value = cur;
  if (color && /^#[0-9a-fA-F]{6}$/.test(cur)) color.value = cur;
  if (deckChk) deckChk.checked = false;
  if (dlg && typeof dlg.showModal === 'function') dlg.showModal();
  else setBackground();
}

function applyBackgroundFromDialog() {
  const css = (document.getElementById('bgCssInput') && document.getElementById('bgCssInput').value) || '';
  const color = document.getElementById('bgColorInput');
  const deckChk = document.getElementById('bgDeckChk');
  let v = String(css).trim();
  if (!v && color) v = color.value;
  pushUndo();
  if (deckChk && deckChk.checked) {
    deck.theme_background = v || null;
  } else {
    const s = currentSlide();
    if (s) s.background = v || null;
  }
  applyBackground();
  scheduleSave();
  setAuthStatus('Background applied', false);
}

function setBackground() {
  openBackgroundDialog();
}

function openLayoutDialog() {
  const host = document.getElementById('layoutPresets');
  const dlg = document.getElementById('layoutDlg');
  if (!host || !dlg) {
    if (layoutSelect) layoutSelect.focus();
    return;
  }
  const cur = (currentSlide() && currentSlide().layout) || 'title_body';
  host.innerHTML = '';
  LAYOUT_PRESETS.forEach((p) => {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'era-theme-card' + (p.id === cur ? ' active' : '');
    btn.innerHTML =
      '<div class="era-theme-swatch" style="background:#f1f3f4;display:flex;flex-direction:column;justify-content:center;padding:.35rem;font-size:.65rem;color:#5c6770">' +
      (p.id === 'two_column' ? 'Title<br/>Col · Col' : p.id === 'title_only' ? 'Title' : 'Title<br/>Body') +
      '</div><div>' +
      p.label +
      '</div>';
    btn.addEventListener('click', () => {
      const s = currentSlide();
      if (!s) return;
      pushUndo();
      s.layout = p.id;
      if (layoutSelect) layoutSelect.value = p.id;
      applyLayoutClass(s.layout);
      scheduleSave();
      dlg.close();
      setAuthStatus('Layout: ' + p.label, false);
    });
    host.appendChild(btn);
  });
  if (typeof dlg.showModal === 'function') dlg.showModal();
}

function openThemePanel() {
  closePresSidePanels();
  const panel = document.getElementById('themePanel');
  const cards = document.getElementById('themeCards');
  const btn = document.getElementById('themeChromeBtn');
  if (!panel || !cards) return;
  cards.innerHTML = '';
  const cur = deck.theme_background || '#ffffff';
  THEME_PRESETS.forEach((p) => {
    const card = document.createElement('button');
    card.type = 'button';
    card.className = 'era-theme-card' + (p.value === cur ? ' active' : '');
    card.innerHTML =
      '<div class="era-theme-swatch" style="background:' +
      p.value.replace(/"/g, '') +
      '"></div><div>' +
      p.label +
      '</div>';
    card.addEventListener('click', () => {
      pushUndo();
      deck.theme_background = p.value;
      applyBackground();
      scheduleSave();
      openThemePanel();
      setAuthStatus('Theme: ' + p.label, false);
    });
    cards.appendChild(card);
  });
  panel.hidden = false;
  if (btn) btn.classList.add('active');
}

function openTransitionPanel() {
  closePresSidePanels();
  const panel = document.getElementById('transitionPanel');
  const btn = document.getElementById('transitionChromeBtn');
  const trSel = document.getElementById('transitionSelect');
  const anSel = document.getElementById('animationSelect');
  const s = currentSlide();
  if (trSel) trSel.value = (s && s.transition) || 'none';
  if (anSel) anSel.value = (s && s.animation) || 'none';
  if (panel) panel.hidden = false;
  if (btn) {
    btn.disabled = false;
    btn.classList.add('active');
  }
}

function applyMotionToSlide(all) {
  const trSel = document.getElementById('transitionSelect');
  const anSel = document.getElementById('animationSelect');
  const tr = (trSel && trSel.value) || 'none';
  const an = (anSel && anSel.value) || 'none';
  if (!deck || !deck.slides || !deck.slides.length) {
    setAuthStatus('Open a deck first', true);
    return;
  }
  pushUndo();
  flushEditorToModel();
  const targets = all ? deck.slides : [currentSlide()].filter(Boolean);
  targets.forEach((s) => {
    s.transition = tr === 'none' ? '' : tr;
    s.animation = an === 'none' ? '' : an;
  });
  scheduleSave();
  setAuthStatus(
    all ? 'Motion applied to all slides' : 'Motion applied to this slide',
    false
  );
}

function applyPresentMotionClasses() {
  const ps = document.getElementById('presentSlide');
  if (!ps) return;
  ps.classList.remove(
    'era-tr-fade',
    'era-tr-push',
    'era-tr-wipe',
    'era-tr-morph',
    'era-anim-appear'
  );
  const s = currentSlide();
  if (!s) return;
  const tr = s.transition || '';
  const an = s.animation || '';
  void ps.offsetWidth;
  if (tr === 'fade') ps.classList.add('era-tr-fade');
  else if (tr === 'push') ps.classList.add('era-tr-push');
  else if (tr === 'wipe') ps.classList.add('era-tr-wipe');
  else if (tr === 'morph') ps.classList.add('era-tr-morph');
  if (an === 'appear') ps.classList.add('era-anim-appear');
}

function renderSlideNotesRail() {
  const ul = document.getElementById('commentsList');
  const panel = document.getElementById('commentsPanel');
  if (!ul) return;
  ul.innerHTML = '';
  const slides = (deck && deck.slides) || [];
  let any = false;
  slides.forEach((s, i) => {
    if (!s.notes || !String(s.notes).trim()) return;
    any = true;
    const li = document.createElement('li');
    li.style.borderBottom = '1px solid var(--era-line)';
    li.style.padding = '0.35rem 0';
    li.innerHTML =
      '<div><strong>Slide ' +
      (i + 1) +
      '</strong></div><div style="white-space:pre-wrap">' +
      String(s.notes)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;') +
      '</div>';
    li.addEventListener('click', () => {
      if (typeof flushEditorToModel === 'function') flushEditorToModel();
      slideIndex = i;
      if (typeof renderAll === 'function') renderAll();
    });
    ul.appendChild(li);
  });
  if (!any) ul.innerHTML = '<li class="era-hint">No notes yet — use Comment</li>';
}

function renderPresentSlide() {
  const s = currentSlide();
  if (!s || !presentOverlay) return;
  const ps = document.getElementById('presentSlide');
  ensureSlideFrames(s);
  document.getElementById('presentTitle').textContent = slidePlainTitle(s);
  document.getElementById('presentBody').textContent = RT()
    ? RT().framePlain(s.body_frame)
    : '';
  const b2 = document.getElementById('presentBody2');
  const two = (s.layout || '') === 'two_column';
  ps.classList.toggle('two_col', two);
  b2.textContent = two && RT() ? RT().framePlain(s.body2_frame) : '';
  b2.style.display = two ? 'block' : 'none';
  const notesEl = document.getElementById('presentNotes');
  if (notesEl) notesEl.textContent = s.notes || '';
  const bg = s.background || deck.theme_background || '#ffffff';
  ps.style.background = bg;
  ps.style.color = '#202124';
  const img = document.getElementById('presentImage');
  if (img) {
    if (s.image_url) {
      img.src = s.image_url;
      img.hidden = false;
    } else {
      img.removeAttribute('src');
      img.hidden = true;
    }
  }
  applyPresentMotionClasses();
}

function startPresent() {
  if (!deckId || !deck.slides.length) {
    setAuthStatus('Open a deck first', true);
    return;
  }
  flushEditorToModel();
  presenting = true;
  presentOverlay.classList.add('active');
  renderPresentSlide();
  try {
    presentOverlay.requestFullscreen && presentOverlay.requestFullscreen();
  } catch (_) {}
}

function stopPresent() {
  presenting = false;
  presentOverlay.classList.remove('active');
  if (document.fullscreenElement) {
    document.exitFullscreen().catch(() => {});
  }
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

document.getElementById('newDeckBtn').onclick = () => {
  createDeck().catch(() => {});
};
document.getElementById('addSlideBtn').onclick = () => addSlide();
const dupSlideBtn = document.getElementById('dupSlideBtn');
if (dupSlideBtn) dupSlideBtn.onclick = () => duplicateSlide();
const boldTextBtn = document.getElementById('boldTextBtn');
if (boldTextBtn) boldTextBtn.onclick = () => toggleBoldFormat();
const fontIncBtn = document.getElementById('fontIncBtn');
if (fontIncBtn) fontIncBtn.onclick = () => stepFont(2);
const fontDecBtn = document.getElementById('fontDecBtn');
if (fontDecBtn) fontDecBtn.onclick = () => stepFont(-2);
const sizeSelect = document.getElementById('sizeSelect');
if (sizeSelect) {
  sizeSelect.onchange = () => {
    const n = parseInt(sizeSelect.value, 10);
    if (!n) return;
    const ctx = typeof activeBlockContext === 'function' ? activeBlockContext() : null;
    if (!ctx) return;
    const range =
      typeof selectionOrTypingRange === 'function' ? selectionOrTypingRange(ctx) : null;
    if (!range) {
      typingMarks = Object.assign({}, typingMarks || {}, { font_size_pt: n });
      setAuthStatus('Font ' + n + 'pt (typing)', false);
      return;
    }
    pushUndo();
    ctx.block.inlines = RT().applyMarksRangeLocal(ctx.block.inlines, range.start, range.end, {
      font_size_pt: n,
    });
    if (ctx.el) ctx.el.innerHTML = RT().renderInlineHtml(ctx.block);
    sendFrameOp(ctx.frameKey, {
      type: 'set_marks_range',
      block_id: ctx.block.id,
      start: range.start,
      end: range.end,
      font_size_pt: n,
    });
    scheduleSave();
    setAuthStatus('Font ' + n + 'pt', false);
  };
}
const fontSelect = document.getElementById('fontSelect');
if (fontSelect) {
  fontSelect.onchange = () => setSlideFontFamily(fontSelect.value);
}
const insertImageBtn = document.getElementById('insertImageBtn');
if (insertImageBtn) insertImageBtn.onclick = () => insertSlideImage().catch(() => {});
document.getElementById('moveUpBtn').onclick = () => moveSlide(-1);
document.getElementById('moveDownBtn').onclick = () => moveSlide(1);
document.getElementById('presentBtn').onclick = () => startPresent();
document.getElementById('themeBgBtn').onclick = () => setBackground();
const backgroundChromeBtn = document.getElementById('backgroundChromeBtn');
if (backgroundChromeBtn) backgroundChromeBtn.onclick = () => openBackgroundDialog();
const layoutChromeBtn = document.getElementById('layoutChromeBtn');
if (layoutChromeBtn) layoutChromeBtn.onclick = () => openLayoutDialog();
const themeChromeBtn = document.getElementById('themeChromeBtn');
if (themeChromeBtn) themeChromeBtn.onclick = () => openThemePanel();
const transitionChromeBtn = document.getElementById('transitionChromeBtn');
if (transitionChromeBtn) {
  transitionChromeBtn.disabled = false;
  transitionChromeBtn.onclick = () => openTransitionPanel();
}
const transitionApplyBtn = document.getElementById('transitionApplyBtn');
if (transitionApplyBtn) transitionApplyBtn.onclick = () => applyMotionToSlide(false);
const transitionApplyAllBtn = document.getElementById('transitionApplyAllBtn');
if (transitionApplyAllBtn) transitionApplyAllBtn.onclick = () => applyMotionToSlide(true);
const themePanelClose = document.getElementById('themePanelClose');
if (themePanelClose) themePanelClose.onclick = () => closePresSidePanels();
const transitionPanelClose = document.getElementById('transitionPanelClose');
if (transitionPanelClose) transitionPanelClose.onclick = () => closePresSidePanels();
if (window.EraOfficeShell) {
  if (EraOfficeShell.wireSessionWatch) EraOfficeShell.wireSessionWatch();
  if (EraOfficeShell.wireCommentsToggle) EraOfficeShell.wireCommentsToggle(false);
}
const bgDlg = document.getElementById('backgroundDlg');
if (bgDlg) {
  bgDlg.addEventListener('close', () => {
    if (bgDlg.returnValue === 'ok') applyBackgroundFromDialog();
  });
}
if (window.EraOfficeToolbar) EraOfficeToolbar.init(document);
if (window.EraOfficeShell && EraOfficeShell.mountIcons) EraOfficeShell.mountIcons(document);
document.getElementById('undoBtn').onclick = () => undoEdit();
const redoBtn = document.getElementById('redoBtn');
if (redoBtn) redoBtn.onclick = () => redoEdit();
const printBtn = document.getElementById('printBtn');
if (printBtn) printBtn.onclick = () => printSetup();
const shareCopyBtn = document.getElementById('shareCopyBtn');
if (shareCopyBtn) {
  shareCopyBtn.onclick = () => {
    const input = document.getElementById('shareLinkInput');
    const url = (input && input.value) || '';
    if (!url) return;
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(url).then(
        () => setAuthStatus('Share link copied', false),
        () => EraOfficeShell.promptCopy({ title: 'Share deck', value: url })
      );
    } else {
      void EraOfficeShell.promptCopy({ title: 'Share deck', value: url });
    }
  };
}
const shareBtn = document.getElementById('shareBtn');
if (shareBtn) shareBtn.onclick = () => openShareDeck();
const zoomSelect = document.getElementById('zoomSelect');
if (zoomSelect) {
  zoomSelect.onchange = () => {
    const canvas = document.querySelector('.slide-canvas');
    if (!canvas) return;
    if (zoomSelect.value === 'fit') {
      canvas.style.zoom = '';
      return;
    }
    const z = parseInt(zoomSelect.value, 10) || 100;
    canvas.style.zoom = String(z / 100);
  };
}
const formatPainterBtn = document.getElementById('formatPainterBtn');
if (formatPainterBtn) {
  formatPainterBtn.onclick = () => {
    toggleBoldFormat();
    setAuthStatus('Bold toggled on selection', false);
  };
}
const commentSlideBtn = document.getElementById('commentSlideBtn');
if (commentSlideBtn) {
  commentSlideBtn.onclick = () => {
    void (async () => {
      const n = await EraOfficeShell.promptText({
        title: 'Slide comment',
        label: 'Comment (appends to speaker notes)',
        value: '',
        multiline: true,
      });
      if (n == null || !String(n).trim()) return;
      const s = typeof currentSlide === 'function' ? currentSlide() : null;
      if (!s) return;
      pushUndo();
      const line = '💬 ' + String(n).trim();
      s.notes = ((s.notes || '') + (s.notes ? '\n' : '') + line).trim();
      const notesEl = document.getElementById('speakerNotes');
      if (notesEl) notesEl.value = s.notes;
      if (typeof scheduleSave === 'function') scheduleSave();
      renderSlideNotesRail();
      if (window.EraOfficeShell && EraOfficeShell.setCommentsOpen) {
        EraOfficeShell.setCommentsOpen(true);
      }
      setAuthStatus('Comment added to speaker notes', false);
    })();
  };
}
document.getElementById('saveBtn').onclick = () => {
  saveDeck(false).catch(() => {});
};
document.getElementById('prevBtn').onclick = () => goSlide(-1);
document.getElementById('nextBtn').onclick = () => goSlide(1);

if (layoutSelect) {
  layoutSelect.addEventListener('change', () => {
    const s = currentSlide();
    if (s) {
      pushUndo();
      s.layout = layoutSelect.value;
      applyLayoutClass(s.layout);
      scheduleSave();
    }
  });
}

if (speakerNotes) {
  speakerNotes.addEventListener('input', () => {
    const s = currentSlide();
    if (!s) return;
    pushUndoDebounced();
    s.notes = speakerNotes.value || '';
    scheduleSave();
  });
}

const deckTitleEl = document.getElementById('deckTitle');
if (deckTitleEl) {
  deckTitleEl.addEventListener('change', () => {
    if (!deckId) return;
    deck.name = (deckTitleEl.value || '').trim() || 'Untitled deck';
    deckMeta.textContent = 'Deck: ' + deck.name;
    scheduleSave();
  });
}

slideTitle.addEventListener('input', () => {
  flushEditorToModel();
  renderFilmstrip();
  scheduleSave();
});
slideBody.addEventListener('input', () => {
  flushEditorToModel();
  scheduleSave();
});
if (slideBody2) {
  slideBody2.addEventListener('input', () => {
    flushEditorToModel();
    scheduleSave();
  });
}

let findSlideCursor = -1;

function findNextSlide() {
  const input = document.getElementById('findSlideInput');
  const q = ((input && input.value) || '').trim().toLowerCase();
  if (!q) {
    if (input) input.focus();
    return;
  }
  if (!deck.slides || !deck.slides.length) {
    setAuthStatus('No slides to search', true);
    return;
  }
  flushEditorToModel();
  const n = deck.slides.length;
  for (let step = 1; step <= n; step++) {
    const i = (findSlideCursor + step) % n;
    const s = deck.slides[i];
    ensureSlideFrames(s);
    const hay = (
      slidePlainTitle(s) +
      ' ' +
      (RT() ? RT().framePlain(s.body_frame) : '') +
      ' ' +
      (RT() ? RT().framePlain(s.body2_frame) : '') +
      ' ' +
      (s.notes || '')
    ).toLowerCase();
    if (hay.includes(q)) {
      findSlideCursor = i;
      slideIndex = i;
      renderAll();
      setAuthStatus('Found on slide ' + (i + 1), false);
      return;
    }
  }
  setAuthStatus('No matches', true);
}

function buildPrintRoot() {
  let root = document.getElementById('printRoot');
  if (!root) {
    root = document.createElement('div');
    root.id = 'printRoot';
    root.setAttribute('aria-hidden', 'true');
    document.body.appendChild(root);
  }
  flushEditorToModel();
  root.innerHTML = '';
  deck.slides.forEach((s, i) => {
    ensureSlideFrames(s);
    const page = document.createElement('section');
    page.className = 'print-slide';
    page.style.background = s.background || deck.theme_background || '#ffffff';
    const h = document.createElement('h1');
    h.textContent = slidePlainTitle(s) || 'Slide ' + (i + 1);
    page.appendChild(h);
    const body = document.createElement('div');
    body.className = 'print-body';
    body.textContent = RT() ? RT().framePlain(s.body_frame) : '';
    page.appendChild(body);
    if ((s.layout || '') === 'two_column' && RT()) {
      const b2 = document.createElement('div');
      b2.className = 'print-body';
      b2.textContent = RT().framePlain(s.body2_frame);
      page.appendChild(b2);
    }
    if (s.image_url) {
      const img = document.createElement('img');
      img.src = s.image_url;
      img.alt = '';
      img.className = 'print-image';
      page.appendChild(img);
    }
    if ((s.notes || '').trim()) {
      const notes = document.createElement('p');
      notes.className = 'print-notes';
      notes.textContent = 'Notes: ' + s.notes;
      page.appendChild(notes);
    }
    root.appendChild(page);
  });
  return root;
}

function printSetup() {
  if (!deckId || !deck.slides.length) {
    setAuthStatus('Open a deck first', true);
    return;
  }
  buildPrintRoot();
  document.body.classList.add('era-printing');
  setAuthStatus('Print: ' + deck.slides.length + ' slide(s), one per page', false);
  const cleanup = () => {
    document.body.classList.remove('era-printing');
    window.removeEventListener('afterprint', cleanup);
  };
  window.addEventListener('afterprint', cleanup);
  setTimeout(() => {
    try {
      window.print();
    } finally {
      // Headless / stubbed print may never fire afterprint.
      setTimeout(cleanup, 300);
    }
  }, 50);
}

if (window.EraOfficeMenubar) {
  EraOfficeMenubar.init('#menubar', {
    'file.new': () => createDeck().catch(() => {}),
    'file.import': () => document.getElementById('importBtn').click(),
    'file.export': () => document.getElementById('exportBtn').click(),
    'file.share': () => openShareDeck(),
    'file.odp': () => exportOdp().catch(() => {}),
    'file.save': () => saveDeck(false).catch(() => {}),
    'file.print': printSetup,
    'edit.undo': undoEdit,
    'edit.redo': redoEdit,
    'edit.find': () => {
      const input = document.getElementById('findSlideInput');
      if (input) input.focus();
      else findNextSlide();
    },
    'slide.new': addSlide,
    'slide.duplicate': duplicateSlide,
    'slide.up': () => moveSlide(-1),
    'slide.down': () => moveSlide(1),
    'slide.delete': deleteSlide,
    'slide.bg': setBackground,
    'slide.master': openMasterDlg,
    'insert.image': () => insertSlideImage().catch(() => {}),
    'format.bold': toggleBoldFormat,
    'format.alignLeft': () => setAlignFormat('left'),
    'format.alignCenter': () => setAlignFormat('center'),
    'format.alignRight': () => setAlignFormat('right'),
    'format.fontInc': () => stepFont(2),
    'format.fontDec': () => stepFont(-2),
    'view.present': startPresent,
    'view.filmstrip': () => filmstrip && filmstrip.scrollIntoView(),
    'help.about': () => {
      void EraOfficeShell.confirmAction({
        title: 'About ERA Presentations',
        message:
          'ERA Presentations — deck editing in your contour (not PowerPoint / Google Slides).\n\n' +
          '• Transitions (including Morph) are soft client effects in Slideshow\n' +
          '• Master / theme defaults apply deck-wide backgrounds\n' +
          '• Complex pptx fidelity is improving; prefer ERA-native decks for co-edit',
        okLabel: 'OK',
        cancelLabel: 'Close',
      });
    },
  });
}

const masterDlg = document.getElementById('masterDlg');
if (masterDlg) {
  masterDlg.addEventListener('close', () => {
    if (masterDlg.returnValue === 'ok') applyMasterDlg();
  });
}

const findSlideBtn = document.getElementById('findSlideBtn');
if (findSlideBtn) findSlideBtn.addEventListener('click', findNextSlide);
const findSlideInput = document.getElementById('findSlideInput');
if (findSlideInput) {
  findSlideInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      findNextSlide();
    }
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
  setAuthStatus('Importing pptx…', false);
  const buf = await f.arrayBuffer();
  const b64 = arrayBufferToBase64(buf);
  const res = await officeFetch('/api/v1/presentations/import', {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({
      tenant_id: tenantId,
      user_id: userId,
      name: f.name.replace(/\.pptx$/i, '.erap'),
      pptx_base64: b64,
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
  location.href = '/presentations/' + data.drive_object_id;
};

async function exportBinary(pathSuffix, ext, label) {
  if (!deckId) {
    setAuthStatus('Open a deck first', true);
    return;
  }
  await saveDeck(true);
  setAuthStatus('Exporting ' + label + '…', false);
  const res = await officeFetch(
    '/api/v1/presentations/' + encodeURIComponent(deckId) + '/export/' + pathSuffix,
    { method: 'POST', headers: authHeaders() }
  );
  if (!res.ok) {
    setAuthStatus('Export failed: ' + res.status, true);
    return;
  }
  const blob = await res.blob();
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = (deckId || 'export') + '.' + ext;
  a.click();
  URL.revokeObjectURL(a.href);
  setAuthStatus('Export ready', false);
}

async function exportOdp() {
  await exportBinary('odp', 'odp', 'odp');
}

document.getElementById('exportBtn').onclick = async () => {
  await exportBinary('pptx', 'pptx', 'pptx');
};

document.addEventListener('keydown', (ev) => {
  if (presenting) {
    if (ev.key === 'Escape') {
      ev.preventDefault();
      stopPresent();
      return;
    }
    if (ev.key === 'ArrowRight' || ev.key === 'ArrowDown' || ev.key === ' ' || ev.key === 'PageDown') {
      ev.preventDefault();
      goSlide(1);
      return;
    }
    if (ev.key === 'ArrowLeft' || ev.key === 'ArrowUp' || ev.key === 'PageUp') {
      ev.preventDefault();
      goSlide(-1);
      return;
    }
  }
  if (!deckId) return;
  if ((ev.ctrlKey || ev.metaKey) && ev.key.toLowerCase() === 'z' && !ev.shiftKey) {
    ev.preventDefault();
    undoEdit();
    return;
  }
  if (
    (ev.ctrlKey || ev.metaKey) &&
    (ev.key.toLowerCase() === 'y' || (ev.key.toLowerCase() === 'z' && ev.shiftKey))
  ) {
    ev.preventDefault();
    redoEdit();
    return;
  }
  const tag = (ev.target && ev.target.tagName) || '';
  if (tag === 'INPUT' || (ev.target && ev.target.isContentEditable && !ev.altKey)) {
    if (!ev.altKey) return;
  }
  if (ev.altKey && ev.key === 'ArrowRight') {
    ev.preventDefault();
    goSlide(1);
  }
  if (ev.altKey && ev.key === 'ArrowLeft') {
    ev.preventDefault();
    goSlide(-1);
  }
});

if (window.EraOfficeShell && EraOfficeShell.requireAuthOrRedirect) {
  if (!EraOfficeShell.requireAuthOrRedirect()) {
    /* redirecting to /login */
  }
} else if (!localStorage.getItem('era_token')) {
  location.href = '/login?next=' + encodeURIComponent(location.pathname);
}

deckId = pathDeckId();
if (deckId) {
  loadDeck(deckId).catch(() => {});
} else if (localStorage.getItem('era_token')) {
  if (emptyHint) emptyHint.hidden = false;
  workspace.hidden = true;
  deckMeta.textContent = 'Creating Untitled presentation…';
  createDeck().catch(() => {
    if (emptyHint) emptyHint.hidden = false;
    deckMeta.textContent = 'No deck open';
  });
} else {
  if (emptyHint) emptyHint.hidden = false;
  workspace.hidden = true;
}
