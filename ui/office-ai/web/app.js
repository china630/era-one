if (window.EraOfficeShell) {
  if (EraOfficeShell.markActiveNav) EraOfficeShell.markActiveNav('ai');
  if (EraOfficeShell.mountNav) EraOfficeShell.mountNav(document);
  else if (EraOfficeShell.mountIcons) EraOfficeShell.mountIcons(document);
  if (EraOfficeShell.syncUserChip) EraOfficeShell.syncUserChip();
}

const authStatus = document.getElementById('authStatus');
const sourceText = document.getElementById('sourceText');
const instructionInput = document.getElementById('instructionInput');
const resultEl = document.getElementById('result');
const modeBadge = document.getElementById('modeBadge');
const summarizeBtn = document.getElementById('summarizeBtn');
const rewriteBtn = document.getElementById('rewriteBtn');

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

function setMode(mode) {
  modeBadge.className = '';
  modeBadge.textContent = '';
  if (!mode) return;
  modeBadge.textContent = 'mode=' + mode;
  modeBadge.classList.add(mode === 'ollama' ? 'ollama' : 'stub');
}

function uiMode() {
  const params = new URLSearchParams(location.search);
  let mode = params.get('mode') || '';
  try {
    if (!mode) mode = sessionStorage.getItem('era_office_ai_mode') || '';
    if (mode) sessionStorage.removeItem('era_office_ai_mode');
  } catch (_) {}
  return mode === 'rewrite' ? 'rewrite' : 'summarize';
}

function applyUiMode(mode) {
  document.body.classList.toggle('mode-rewrite', mode === 'rewrite');
  if (summarizeBtn) summarizeBtn.classList.toggle('era-btn-primary', mode !== 'rewrite');
  if (rewriteBtn) rewriteBtn.classList.toggle('era-btn-primary', mode === 'rewrite');
}

function clearAll() {
  sourceText.value = '';
  if (instructionInput) instructionInput.value = '';
  resultEl.textContent = '';
  setMode('');
  setAuthStatus(
    localStorage.getItem('era_token') ? 'Token present' : 'Sign in via Drive first (era_token).',
    !localStorage.getItem('era_token')
  );
}

async function summarize() {
  if (!localStorage.getItem('era_token')) {
    setAuthStatus('Sign in via Drive first (era_token).', true);
    return;
  }
  const text = (sourceText.value || '').trim();
  if (!text) {
    setAuthStatus('Paste source text first', true);
    return;
  }
  summarizeBtn.disabled = true;
  setAuthStatus('Summarizing…', false);
  setMode('');
  try {
    const res = await officeFetch('/api/v1/docs-ai/summarize', {
      method: 'POST',
      headers: authHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ text }),
    });
    if (!res.ok) {
      if (window.EraOfficeShell && EraOfficeShell.handleUnauthorized && EraOfficeShell.handleUnauthorized(res)) {
        return;
      }
      setAuthStatus('Summarize failed: ' + res.status, true);
      resultEl.textContent = '';
      return;
    }
    const data = await res.json();
    resultEl.textContent = data.summary || JSON.stringify(data);
    setMode(data.mode || 'stub');
    setAuthStatus('Summary ready (' + (data.mode || 'stub') + ')', false);
  } finally {
    summarizeBtn.disabled = false;
  }
}

async function rewrite() {
  if (!localStorage.getItem('era_token')) {
    setAuthStatus('Sign in via Drive first (era_token).', true);
    return;
  }
  const text = (sourceText.value || '').trim();
  if (!text) {
    setAuthStatus('Paste source text first', true);
    return;
  }
  if (rewriteBtn) rewriteBtn.disabled = true;
  setAuthStatus('Rewriting…', false);
  setMode('');
  try {
    const instruction = instructionInput && instructionInput.value ? instructionInput.value.trim() : '';
    const res = await officeFetch('/api/v1/docs-ai/rewrite', {
      method: 'POST',
      headers: authHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ text, instruction }),
    });
    if (!res.ok) {
      if (window.EraOfficeShell && EraOfficeShell.handleUnauthorized && EraOfficeShell.handleUnauthorized(res)) {
        return;
      }
      setAuthStatus('Rewrite failed: ' + res.status, true);
      resultEl.textContent = '';
      return;
    }
    const data = await res.json();
    resultEl.textContent = data.rewrite || JSON.stringify(data);
    setMode(data.mode || 'stub');
    setAuthStatus('Rewrite ready (' + (data.mode || 'stub') + ')', false);
  } finally {
    if (rewriteBtn) rewriteBtn.disabled = false;
  }
}

document.getElementById('summarizeBtn').onclick = () => {
  summarize().catch(() => {});
};
if (rewriteBtn) {
  rewriteBtn.onclick = () => {
    rewrite().catch(() => {});
  };
}
document.getElementById('clearBtn').onclick = () => clearAll();

if (window.EraOfficeMenubar) {
  EraOfficeMenubar.init('#menubar', {
    'file.openDrive': () => {
      location.href = '/drive/';
    },
    'file.clear': () => clearAll(),
    'tools.summarize': () => {
      applyUiMode('summarize');
      summarize().catch(() => {});
    },
    'tools.rewrite': () => {
      applyUiMode('rewrite');
      rewrite().catch(() => {});
    },
    'help.about': () =>
      setAuthStatus('ERA Office AI — air-gap assist (stub / in-contour Ollama)', false),
  });
}

const activeMode = uiMode();
applyUiMode(activeMode);

const params = new URLSearchParams(location.search);
let prefill = params.get('text') || params.get('q') || '';
try {
  if (!prefill) {
    prefill = sessionStorage.getItem('era_office_ai_text') || '';
    if (prefill) sessionStorage.removeItem('era_office_ai_text');
  }
} catch (_) {}
if (prefill) sourceText.value = prefill;

if (window.EraOfficeShell) {
  if (EraOfficeShell.wireSessionWatch) EraOfficeShell.wireSessionWatch();
  if (EraOfficeShell.requireAuthOrRedirect && !EraOfficeShell.requireAuthOrRedirect()) {
    /* redirecting */
  }
} else if (!localStorage.getItem('era_token')) {
  location.href = '/login?next=' + encodeURIComponent(location.pathname + location.search);
}
