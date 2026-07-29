const blocksEl = document.getElementById('blocks');
const banner = document.getElementById('banner');

let docState = { blocks: [] };
let docId = null;
let ws = null;
let syncTimer = null;
const tenantId = 't-dev';
const userId = 'u-dev';

function pathDocId() {
  const parts = location.pathname.replace(/\/$/, '').split('/');
  const id = parts[parts.length - 1];
  return id && id !== 'docs' ? id : null;
}

function blockText(block) {
  return (block.inlines || []).map((i) => i.text || '').join('');
}

function renderBlocks() {
  blocksEl.innerHTML = '';
  for (const block of docState.blocks) {
    const el = document.createElement('div');
    el.className = 'doc-block';
    el.contentEditable = 'true';
    el.dataset.blockId = block.id;
    const type = block.block_type || 'paragraph';
    el.dataset.type = type === 'heading' ? 'heading' : type === 'list_item' ? 'list_item' : 'paragraph';
    el.textContent = blockText(block);
    el.addEventListener('input', () => onBlockInput(block.id, el));
    blocksEl.appendChild(el);
  }
}

function wsUrl(id) {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${location.host}/api/v1/docs/${id}/sync`;
}

function connectSync(id) {
  if (ws) {
    try { ws.close(); } catch (_) {}
  }
  ws = new WebSocket(wsUrl(id));
  ws.onopen = () => console.debug('docs sync connected');
  ws.onerror = (e) => console.warn('docs sync error', e);
}

function sendOp(op) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  ws.send(JSON.stringify(op));
}

function onBlockInput(blockId, el) {
  const block = docState.blocks.find((b) => b.id === blockId);
  if (!block) return;
  const prev = blockText(block);
  const next = el.textContent || '';
  if (prev === next) return;
  block.inlines = [{ text: next, bold: false, italic: false }];

  clearTimeout(syncTimer);
  syncTimer = setTimeout(() => {
    if (next.length >= prev.length) {
      const inserted = next.slice(prev.length);
      const offset = prev.length;
      sendOp({ type: 'insert_text', block_id: blockId, offset, text: inserted });
    } else {
      sendOp({
        type: 'delete_range',
        block_id: blockId,
        start: next.length,
        end: prev.length,
      });
    }
  }, 200);
}

function applyDoc(doc) {
  docState = doc;
  if (doc.legacy_features_dropped) banner.style.display = 'block';
  renderBlocks();
}

async function loadDoc(id) {
  const res = await fetch(`/api/v1/docs/${id}`, {
    headers: { 'X-ERA-User': userId },
  });
  if (!res.ok) return;
  applyDoc(await res.json());
  connectSync(id);
}

document.getElementById('boldBtn').onclick = () => document.execCommand('bold');
document.getElementById('italicBtn').onclick = () => document.execCommand('italic');

document.getElementById('h1Btn').onclick = () => {
  const sel = window.getSelection();
  const blockEl = sel && sel.anchorNode
    ? sel.anchorNode.parentElement?.closest?.('.doc-block')
    : null;
  if (!blockEl) return;
  const blockId = blockEl.dataset.blockId;
  sendOp({ type: 'set_block_type', block_id: blockId, block_type: 'heading', heading_level: 1 });
  const block = docState.blocks.find((b) => b.id === blockId);
  if (block) {
    block.block_type = 'heading';
    block.heading_level = 1;
    blockEl.dataset.type = 'heading';
  }
};

document.getElementById('listBtn').onclick = () => {
  const block = docState.blocks[docState.blocks.length - 1];
  if (!block) return;
  const newBlock = {
    id: crypto.randomUUID(),
    block_type: 'list_item',
    heading_level: 0,
    list_type: 'bullet',
    inlines: [{ text: '', bold: false, italic: false }],
  };
  sendOp({
    type: 'insert_block',
    after_id: block.id,
    block: newBlock,
  });
  docState.blocks.push(newBlock);
  renderBlocks();
};

document.getElementById('snapshotBtn').onclick = async () => {
  if (!docId) return;
  await fetch(`/api/v1/docs/${docId}/snapshot`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tenant_id: tenantId, user_id: userId }),
  });
};

document.getElementById('importBtn').onclick = () => document.getElementById('file').click();
document.getElementById('file').onchange = async (e) => {
  const f = e.target.files[0];
  if (!f) return;
  const buf = await f.arrayBuffer();
  const b64 = btoa(String.fromCharCode(...new Uint8Array(buf)));
  const res = await fetch('/api/v1/docs/import', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tenant_id: tenantId, user_id: userId, docx_base64: b64 }),
  });
  if (res.ok) {
    const data = await res.json();
    location.href = '/docs/' + data.drive_object_id;
  }
};

document.getElementById('exportBtn').onclick = async () => {
  if (!docId) return;
  const res = await fetch(`/api/v1/docs/${docId}/export/docx`, { method: 'POST' });
  if (res.ok) {
    const blob = await res.blob();
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'export.docx';
    a.click();
  }
};

docId = pathDocId();
if (docId) {
  loadDoc(docId).catch(() => {});
} else {
  blocksEl.innerHTML = '<p>Open a document from Drive or import docx.</p>';
}
