/* Wave LATER — Docs: RTF, line numbers, text box, columns, review, compare, mail merge. */
(function () {
  const handlers = window.docsMenuHandlers;
  if (!handlers) return;

  let lineNumbersOn = false;

  function downloadRtf() {
    if (!docId) return setAuthStatus('Open a document first', true);
    setAuthStatus('Exporting RTF…', false);
    fetch('/api/v1/docs/' + encodeURIComponent(docId) + '/export/rtf', {
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
        a.download = (docId || 'export') + '.rtf';
        a.click();
        URL.revokeObjectURL(a.href);
        setAuthStatus('RTF export ready', false);
      })
      .catch(() => setAuthStatus('RTF export failed', true));
  }

  function toggleLineNumbers() {
    lineNumbersOn = !lineNumbersOn;
    if (blocksEl) {
      blocksEl.classList.toggle('line-numbers', lineNumbersOn);
      // Honest MVP: numbers are block indices, not wrapped visual lines.
      if (lineNumbersOn) {
        Array.from(blocksEl.querySelectorAll('.doc-block')).forEach((el, i) => {
          el.dataset.line = String(i + 1);
        });
        /* Quiet toggle — limits explained in Help → About */
      } else {
        Array.from(blocksEl.querySelectorAll('.doc-block')).forEach((el) => {
          delete el.dataset.line;
        });
      }
    }
  }

  function broadcastRevision(action, revision) {
    if (typeof ws === 'undefined' || !ws || ws.readyState !== WebSocket.OPEN) return;
    const { userId } = identity();
    ws.send(
      JSON.stringify({
        type: 'revision_event',
        action,
        from: userId,
        revision,
      })
    );
  }

  async function insertTextBox() {
    const raw = await EraOfficeShell.promptText({
      title: 'Insert text box',
      label: 'Content (optional: text | width%)',
      value: 'Text box',
      message: 'Tip: append " | 60" for 60% width (40–100).',
      multiline: true,
    });
    if (raw == null) return;
    let text = String(raw).trim() || 'Text box';
    let widthPct = 100;
    const m = text.match(/^(.*?)\s*\|\s*(\d{2,3})\s*%?\s*$/);
    if (m) {
      text = m[1].trim() || 'Text box';
      widthPct = Math.min(100, Math.max(40, parseInt(m[2], 10) || 100));
    }
    const id = crypto.randomUUID ? crypto.randomUUID() : 'tb-' + Date.now();
    const block = {
      id,
      block_type: 'text_box',
      heading_level: 0,
      align: 'left',
      indent_mm: undefined,
      box_width_pct: widthPct,
      inlines: [{ text, bold: false, italic: false, underline: false }],
    };
    if (typeof insertBlockAfter === 'function') insertBlockAfter(block);
    else {
      pushUndo();
      const after = selectedBlock();
      let idx = docState.blocks.length - 1;
      if (after) idx = docState.blocks.findIndex((b) => b.id === after.block.id);
      docState.blocks.splice(idx + 1, 0, block);
      sendOp({ type: 'insert_block', after_id: after ? after.block.id : null, block });
      renderBlocks();
      scheduleAutosave();
    }
    setAuthStatus('Text box inserted (' + widthPct + '% width)', false);
  }

  async function setColumns() {
    const dlg = document.getElementById('columnsDlg');
    const cur = (docState.page && docState.page.columns) || 1;
    const sel = document.getElementById('columnsSelect');
    if (sel) sel.value = String(cur);
    if (dlg && typeof dlg.showModal === 'function') {
      dlg.showModal();
      return;
    }
    const raw = await EraOfficeShell.chooseOption({
      title: 'Columns',
      message: 'Page columns (1–3)',
      options: [
        { value: '1', label: '1 column' },
        { value: '2', label: '2 columns' },
        { value: '3', label: '3 columns' },
      ],
      value: String(cur),
    });
    if (raw == null) return;
    applyColumns(Math.min(3, Math.max(1, parseInt(raw, 10) || 1)));
  }

  function applyColumns(n) {
    pushUndo();
    if (!docState.page) docState.page = { size: 'a4', orientation: 'portrait', margins_mm: 20, columns: 1 };
    docState.page.columns = n;
    applyPageChrome();
    const pageSize = document.getElementById('pageColumns');
    if (pageSize) pageSize.value = String(n);
    scheduleAutosave();
    setAuthStatus('Columns: ' + n, false);
  }

  const columnsDlg = document.getElementById('columnsDlg');
  if (columnsDlg) {
    columnsDlg.addEventListener('close', () => {
      if (columnsDlg.returnValue !== 'ok') return;
      const sel = document.getElementById('columnsSelect');
      applyColumns(Math.min(3, Math.max(1, parseInt(sel && sel.value, 10) || 1)));
    });
  }

  function renderRevisionsList() {
    const list = document.getElementById('revisionsList');
    if (!list) return;
    const revs = docState.revisions || [];
    list.innerHTML = '';
    if (!revs.length) {
      list.innerHTML = '<li class="era-hint">No revisions yet</li>';
      return;
    }
    revs.forEach((r) => {
      const li = document.createElement('li');
      li.style.marginBottom = '.35rem';
      li.innerHTML =
        '<div><code>' +
        escapeHtml(r.kind) +
        '</code> · ' +
        escapeHtml(r.author_id || '') +
        '</div><div class="rev-del">' +
        escapeHtml(r.before || '') +
        '</div><div class="rev-ins">' +
        escapeHtml(r.after || '') +
        '</div>';
      const accept = document.createElement('button');
      accept.type = 'button';
      accept.className = 'era-btn';
      accept.textContent = 'Accept';
      accept.style.marginRight = '.25rem';
      accept.addEventListener('click', () => {
        docState.revisions = (docState.revisions || []).filter((x) => x.id !== r.id);
        broadcastRevision('remove', r);
        renderRevisionsList();
        scheduleAutosave();
      });
      const reject = document.createElement('button');
      reject.type = 'button';
      reject.className = 'era-btn';
      reject.textContent = 'Reject';
      reject.addEventListener('click', () => {
        const block = (docState.blocks || []).find((b) => b.id === r.block_id);
        if (block) {
          pushUndo();
          block.inlines = [spanFromMarks(r.before || '', blockMarks(block))];
        }
        docState.revisions = (docState.revisions || []).filter((x) => x.id !== r.id);
        broadcastRevision('remove', r);
        renderBlocks();
        renderRevisionsList();
        scheduleAutosave();
      });
      li.appendChild(accept);
      li.appendChild(reject);
      list.appendChild(li);
    });
  }

  function openReview() {
    const dlg = document.getElementById('reviewDlg');
    const chk = document.getElementById('trackChangesChk');
    if (chk) chk.checked = !!docState.track_changes;
    renderRevisionsList();
    if (dlg && dlg.showModal) dlg.showModal();
  }

  function openCompare() {
    const dlg = document.getElementById('compareDlg');
    const out = document.getElementById('compareOut');
    if (out) out.textContent = '';
    if (dlg && dlg.showModal) dlg.showModal();
  }

  async function runCompare() {
    const otherId = (document.getElementById('compareDocId').value || '').trim();
    const out = document.getElementById('compareOut');
    if (!otherId || !out) return;
    out.textContent = 'Loading…';
    try {
      const res = await fetch('/api/v1/docs/' + encodeURIComponent(otherId), { headers: authHeaders() });
      if (!res.ok) {
        out.textContent = 'Failed: ' + res.status;
        return;
      }
      const other = await res.json();
      const a = (docState.blocks || []).map(blockText);
      const b = (other.blocks || []).map((bl) =>
        (bl.inlines || []).map((i) => i.text || '').join('')
      );
      const max = Math.max(a.length, b.length);
      const lines = [];
      for (let i = 0; i < max; i++) {
        const left = a[i] != null ? a[i] : '';
        const right = b[i] != null ? b[i] : '';
        if (left === right) lines.push('  ' + left);
        else {
          if (left) lines.push('- ' + left);
          if (right) lines.push('+ ' + right);
        }
      }
      out.textContent = lines.join('\n') || '(identical)';
      setAuthStatus('Compare done', false);
    } catch (_) {
      out.textContent = 'Compare failed';
    }
  }

  function openMerge() {
    const dlg = document.getElementById('mergeDlg');
    if (dlg && dlg.showModal) dlg.showModal();
  }

  function parseCsv(text) {
    const rows = String(text || '')
      .split(/\r?\n/)
      .map((l) => l.trim())
      .filter(Boolean);
    if (rows.length < 2) return [];
    const headers = rows[0].split(',').map((h) => h.trim());
    return rows.slice(1).map((line) => {
      const cols = line.split(',').map((c) => c.trim());
      const obj = {};
      headers.forEach((h, i) => {
        obj[h] = cols[i] || '';
      });
      return obj;
    });
  }

  function runMerge() {
    const records = parseCsv(document.getElementById('mergeCsv').value || '');
    if (!records.length) {
      setAuthStatus('Need CSV header + at least one row', true);
      return;
    }
    const template = (docState.blocks || []).map(blockText).join('\n');
    const parts = records.map((rec, idx) => {
      let body = template;
      Object.keys(rec).forEach((k) => {
        body = body.split('<<' + k + '>>').join(rec[k]);
      });
      return '--- Record ' + (idx + 1) + ' ---\n' + body;
    });
    const blob = new Blob([parts.join('\n\n')], { type: 'text/plain;charset=utf-8' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = (docId || 'merge') + '-merged.txt';
    a.click();
    URL.revokeObjectURL(a.href);
    setAuthStatus('Mail merge: ' + records.length + ' record(s)', false);
  }

  const trackChk = document.getElementById('trackChangesChk');
  if (trackChk) {
    trackChk.addEventListener('change', () => {
      docState.track_changes = !!trackChk.checked;
      document.body.classList.toggle('era-suggesting', !!docState.track_changes);
      scheduleAutosave();
      renderBlocks();
      setAuthStatus(docState.track_changes ? 'Track changes on' : 'Track changes off', false);
    });
  }
  const acceptAll = document.getElementById('reviewAcceptAll');
  if (acceptAll) {
    acceptAll.addEventListener('click', () => {
      docState.revisions = [];
      renderRevisionsList();
      scheduleAutosave();
      setAuthStatus('All revisions accepted', false);
    });
  }
  const rejectAll = document.getElementById('reviewRejectAll');
  if (rejectAll) {
    rejectAll.addEventListener('click', () => {
      const revs = (docState.revisions || []).slice().reverse();
      pushUndo();
      revs.forEach((r) => {
        const block = (docState.blocks || []).find((b) => b.id === r.block_id);
        if (block) block.inlines = [spanFromMarks(r.before || '', blockMarks(block))];
      });
      docState.revisions = [];
      renderBlocks();
      renderRevisionsList();
      scheduleAutosave();
      setAuthStatus('All revisions rejected', false);
    });
  }
  const compareRun = document.getElementById('compareRunBtn');
  if (compareRun) compareRun.addEventListener('click', () => runCompare().catch(() => {}));
  const mergeDlg = document.getElementById('mergeDlg');
  if (mergeDlg) {
    mergeDlg.addEventListener('close', () => {
      if (mergeDlg.returnValue === 'run') runMerge();
    });
  }

  Object.assign(handlers, {
    'file.rtf': downloadRtf,
    'view.lineNumbers': toggleLineNumbers,
    'insert.textbox': insertTextBox,
    'format.columns': setColumns,
    'tools.review': openReview,
    'tools.compare': openCompare,
    'tools.merge': openMerge,
  });
})();
