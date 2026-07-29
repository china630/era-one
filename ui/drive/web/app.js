async function api(path, opts = {}) {
  const token = localStorage.getItem('era_token') || '';
  const headers = Object.assign({ 'Authorization': 'Bearer ' + token }, opts.headers || {});
  const res = await fetch('/api/v1/drive' + path, Object.assign({}, opts, { headers }));
  if (res.status === 401) {
    alert('Login via Mail first (OIDC token in localStorage era_token).');
    throw new Error('unauthorized');
  }
  return res;
}

async function refreshFiles() {
  const list = document.getElementById('files');
  list.innerHTML = '';
  const res = await api('/folders/_root/children');
  if (!res.ok) return;
  const data = await res.json();
  for (const o of (data.objects || [])) {
    const li = document.createElement('li');
    const a = document.createElement('a');
    a.href = '/api/v1/drive/objects/' + o.id;
    a.textContent = o.name + ' (' + o.size_bytes + ' B)';
    li.appendChild(a);
    list.appendChild(li);
  }
}

document.getElementById('uploadBtn').addEventListener('click', async () => {
  const input = document.getElementById('file');
  if (!input.files.length) return;
  const fd = new FormData();
  fd.append('file', input.files[0]);
  fd.append('name', input.files[0].name);
  const res = await api('/objects', { method: 'POST', body: fd });
  if (res.ok) await refreshFiles();
});

refreshFiles().catch(() => {});
