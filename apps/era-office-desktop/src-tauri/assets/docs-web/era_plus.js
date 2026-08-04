/* Wave ERA+ / O-STUB Lite — Docs: ODT, section break, footnote, manage styles. */
(function () {
  const handlers = window.docsMenuHandlers;
  if (!handlers) return;

  function newId() {
    return crypto.randomUUID ? crypto.randomUUID() : 'x-' + Date.now();
  }

  function insertAfterSelected(block) {
    if (typeof insertBlockAfter === 'function') {
      insertBlockAfter(block);
      return;
    }
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

  function downloadOdt() {
    if (!docId) return setAuthStatus('Open a document first', true);
    setAuthStatus('Exporting ODT…', false);
    fetch('/api/v1/docs/' + encodeURIComponent(docId) + '/export/odt', {
      method: 'POST',
      headers: authHeaders(),
    })
      .then((res) => {
        if (!res.ok) throw new Error(String(res.status));
        return res.blob();
      })
      .then((blob) => {
        const a = document.createElement('a');
        a.href = URL.createObjectURL(blob);
        a.download = (docId || 'export') + '.odt';
        a.click();
        URL.revokeObjectURL(a.href);
        setAuthStatus('ODT export ready', false);
      })
      .catch(() => setAuthStatus('ODT export failed', true));
  }

  function insertSectionBreak() {
    insertAfterSelected({
      id: newId(),
      block_type: 'section_break',
      heading_level: 0,
      inlines: [],
    });
    setAuthStatus('Section break inserted', false);
  }

  function renumberFootnotes() {
    let n = 0;
    (docState.blocks || []).forEach((b) => {
      if (b.block_type === 'footnote') {
        n += 1;
        b.bookmark_name = 'fn' + n;
      }
    });
  }

  async function insertFootnote() {
    const note = await EraOfficeShell.promptText({
      title: 'Insert footnote',
      label: 'Footnote text',
      value: 'Footnote',
      multiline: true,
    });
    if (note == null || !note.trim()) return;
    renumberFootnotes();
    const mark =
      'fn' + ((docState.blocks || []).filter((b) => b.block_type === 'footnote').length + 1);
    const sel = selectedBlock();
    if (sel && sel.block.block_type !== 'footnote' && sel.block.block_type !== 'section_break') {
      pushUndo();
      const offs = selectionOffsetsInBlock(sel.el) || {
        start: blockText(sel.block).length,
        end: blockText(sel.block).length,
      };
      const insertAt = offs.end;
      const marker = '[' + mark + ']';
      sel.block.inlines = insertTextPreserving(
        sel.block.inlines,
        insertAt,
        marker,
        blockMarks(sel.block)
      );
      sel.el.innerHTML = renderInlineHtml(sel.block);
      sendOp({
        type: 'insert_text',
        block_id: sel.block.id,
        offset: insertAt,
        text: marker,
      });
      if (typeof selectRangeInBlock === 'function') {
        selectRangeInBlock(sel.block.id, insertAt, insertAt + marker.length);
      }
    }
    insertAfterSelected({
      id: newId(),
      block_type: 'footnote',
      heading_level: 0,
      bookmark_name: mark,
      inlines: [{ text: note.trim(), bold: false, italic: false, underline: false }],
    });
    setAuthStatus('Footnote ' + mark + ' inserted — click marker or note to jump', false);
  }

  function ensureStyles() {
    if (!Array.isArray(docState.styles)) docState.styles = [];
    return docState.styles;
  }

  function syncStylesIntoGallery() {
    const sel = document.getElementById('styleSelect');
    if (!sel) return;
    // Remove previous custom options
    Array.from(sel.querySelectorAll('option[data-custom-style]')).forEach((o) => o.remove());
    ensureStyles().forEach((s) => {
      const opt = document.createElement('option');
      opt.value = 'custom:' + s.name;
      opt.textContent = s.name + ' (custom)';
      opt.setAttribute('data-custom-style', '1');
      sel.appendChild(opt);
    });
  }

  function renderStylesList() {
    const ul = document.getElementById('stylesList');
    if (!ul) return;
    const styles = ensureStyles();
    ul.innerHTML = '';
    if (!styles.length) {
      ul.innerHTML = '<li class="era-hint">No custom styles yet</li>';
      syncStylesIntoGallery();
      return;
    }
    styles.forEach((s) => {
      const li = document.createElement('li');
      li.style.padding = '.25rem 0';
      li.style.display = 'flex';
      li.style.flexWrap = 'wrap';
      li.style.gap = '.35rem';
      li.style.alignItems = 'center';
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'era-btn';
      btn.textContent = s.name;
      btn.addEventListener('click', () => {
        document.getElementById('styleName').value = s.name;
        document.getElementById('styleFont').value = s.font_family || 'Arial';
        document.getElementById('styleSize').value = String(s.font_size_pt || 12);
        document.getElementById('styleBold').checked = !!s.bold;
        const ital = document.getElementById('styleItalic');
        if (ital) ital.checked = !!s.italic;
      });
      const hint = document.createElement('span');
      hint.className = 'era-hint';
      hint.textContent =
        (s.font_family || '') +
        ' ' +
        (s.font_size_pt || '') +
        'pt' +
        (s.bold ? ' bold' : '') +
        (s.italic ? ' italic' : '');
      const del = document.createElement('button');
      del.type = 'button';
      del.className = 'era-btn';
      del.textContent = 'Delete';
      del.addEventListener('click', () => {
        docState.styles = ensureStyles().filter((x) => x.name !== s.name);
        scheduleAutosave();
        renderStylesList();
        setAuthStatus('Style deleted: ' + s.name, false);
      });
      li.appendChild(btn);
      li.appendChild(hint);
      li.appendChild(del);
      ul.appendChild(li);
    });
    syncStylesIntoGallery();
  }

  function openStyles() {
    renderStylesList();
    const dlg = document.getElementById('stylesDlg');
    if (dlg && dlg.showModal) dlg.showModal();
  }

  function addOrUpdateStyle() {
    const name = (document.getElementById('styleName').value || '').trim();
    if (!name) return setAuthStatus('Style name required', true);
    const styles = ensureStyles();
    const next = {
      name,
      font_family: (document.getElementById('styleFont').value || '').trim() || undefined,
      font_size_pt: parseInt(document.getElementById('styleSize').value, 10) || 12,
      bold: !!document.getElementById('styleBold').checked,
      italic: !!(document.getElementById('styleItalic') && document.getElementById('styleItalic').checked),
    };
    const i = styles.findIndex((s) => s.name.toLowerCase() === name.toLowerCase());
    if (i >= 0) styles[i] = next;
    else styles.push(next);
    scheduleAutosave();
    renderStylesList();
    setAuthStatus('Style saved: ' + name, false);
  }

  function applyNamedStyle() {
    const name = (document.getElementById('styleName').value || '').trim();
    const styles = ensureStyles();
    const s = styles.find((x) => x.name.toLowerCase() === name.toLowerCase());
    if (!s) return setAuthStatus('Select or add a style first', true);
    const sel = selectedBlock();
    if (!sel) return setAuthStatus('Select a paragraph first', true);
    pushUndo();
    sel.block.style_name = s.name;
    const end = blockText(sel.block).length;
    const range =
      (typeof selectionOrTypingRange === 'function' && selectionOrTypingRange(sel)) || {
        start: 0,
        end,
      };
    sel.block.inlines = applyMarksRangeLocal(sel.block.inlines, range.start, range.end, {
      font_family: s.font_family,
      font_size_pt: s.font_size_pt,
      bold: !!s.bold,
      italic: !!s.italic,
    });
    sel.el.innerHTML = renderInlineHtml(sel.block);
    sendOp({
      type: 'set_block_format',
      block_id: sel.block.id,
      style_name: s.name,
    });
    sendOp({
      type: 'set_marks_range',
      block_id: sel.block.id,
      start: range.start,
      end: range.end,
      font_family: s.font_family || '',
      font_size_pt: s.font_size_pt || 0,
      bold: !!s.bold,
      italic: !!s.italic,
    });
    scheduleAutosave();
    setAuthStatus('Applied style ' + s.name, false);
  }

  const addBtn = document.getElementById('styleAddBtn');
  if (addBtn) addBtn.addEventListener('click', addOrUpdateStyle);
  const applyBtn = document.getElementById('styleApplyBtn');
  if (applyBtn) applyBtn.addEventListener('click', applyNamedStyle);

  // Expose gallery sync for app.js applyStyle custom: paths
  window.eraSyncCustomStylesGallery = syncStylesIntoGallery;
  window.eraApplyCustomStyleByName = function (name) {
    document.getElementById('styleName').value = name;
    applyNamedStyle();
  };

  // After doc load, refresh gallery
  const _origRender = typeof renderBlocks === 'function' ? null : null;
  setTimeout(syncStylesIntoGallery, 0);
  document.addEventListener('era-doc-loaded', syncStylesIntoGallery);

  Object.assign(handlers, {
    'file.odt': downloadOdt,
    'insert.section': insertSectionBreak,
    'insert.footnote': insertFootnote,
    'format.styles': openStyles,
  });
})();
