(function () {
  if (window.EraOfficeShell) {
    if (EraOfficeShell.markActiveNav) EraOfficeShell.markActiveNav('drive');
    if (EraOfficeShell.mountNav) EraOfficeShell.mountNav(document);
    else if (EraOfficeShell.mountIcons) EraOfficeShell.mountIcons(document);
    if (EraOfficeShell.syncUserChip) EraOfficeShell.syncUserChip();
  }

  /** @type {{ id: string, name: string }[]} */
  let pathStack = [{ id: '', name: 'Root' }];
  /** @type {{ id: string, name: string, entries: { principal: string, role: number }[] } | null} */
  let shareState = null;
  /** @type {{ id: string, name: string, kind: 'file' | 'folder' } | null} */
  let moveState = null;
  /** @type {string | null} object URL for image preview (revoked on close) */
  let previewObjectUrl = null;
  /** @type {Map<string, { id: string, name: string, parentId: string }>} */
  const folderIndex = new Map();
  /** @type {Map<string, { id: string, name: string, parentId: string }[]>} children by parent key ('' = root) */
  const treeChildren = new Map();
  /** @type {Set<string>} expanded parent keys ('' = root always shown) */
  const treeExpanded = new Set(['']);
  /** @type {Set<string>} selected item keys: "f:id" | "o:id" */
  const selectedKeys = new Set();
  /** last clicked key for Shift-range select */
  let lastClickedKey = '';
  /** @type {'name'|'date'|'type'} */
  let driveSort = 'name';
  try {
    const s = localStorage.getItem('era_drive_sort');
    if (s === 'name' || s === 'date' || s === 'type') driveSort = s;
  } catch (_) {}
  const TRASH_ID = '__trash__';
  let trashMode = false;
  /** @type {{ folders: any[], objects: any[] }} */
  let lastListing = { folders: [], objects: [] };

  function itemKey(kind, id) {
    return (kind === 'folder' ? 'f:' : 'o:') + id;
  }

  function inTrash() {
    return trashMode || currentFolderId() === TRASH_ID;
  }

  function updateSelectionBar() {
    const bar = document.getElementById('selectionBar');
    const count = document.getElementById('selectionCount');
    const n = selectedKeys.size;
    if (bar) bar.hidden = n === 0 || trashMode;
    if (count) count.textContent = n + (n === 1 ? ' selected' : ' selected');
  }

  function clearSelection() {
    selectedKeys.clear();
    lastClickedKey = '';
    document.querySelectorAll('#files li.selected').forEach((li) => li.classList.remove('selected'));
    document.querySelectorAll('#files .era-row-check').forEach((c) => {
      c.checked = false;
    });
    updateSelectionBar();
  }

  function applySelectionToLi(li, key, on) {
    if (!li) return;
    li.classList.toggle('selected', on);
    const chk = li.querySelector('.era-row-check');
    if (chk) chk.checked = on;
    if (on) selectedKeys.add(key);
    else selectedKeys.delete(key);
  }

  function handleRowSelect(ev, kind, id, orderedKeys) {
    const key = itemKey(kind, id);
    if (ev.shiftKey && lastClickedKey && orderedKeys && orderedKeys.length) {
      const a = orderedKeys.indexOf(lastClickedKey);
      const b = orderedKeys.indexOf(key);
      if (a >= 0 && b >= 0) {
        const lo = Math.min(a, b);
        const hi = Math.max(a, b);
        for (let i = lo; i <= hi; i++) selectedKeys.add(orderedKeys[i]);
        document.querySelectorAll('#files li[data-sel-key]').forEach((li) => {
          applySelectionToLi(li, li.dataset.selKey, selectedKeys.has(li.dataset.selKey));
        });
        updateSelectionBar();
        return;
      }
    }
    if (ev.ctrlKey || ev.metaKey) {
      if (selectedKeys.has(key)) selectedKeys.delete(key);
      else selectedKeys.add(key);
    } else {
      selectedKeys.clear();
      selectedKeys.add(key);
    }
    lastClickedKey = key;
    document.querySelectorAll('#files li[data-sel-key]').forEach((li) => {
      applySelectionToLi(li, li.dataset.selKey, selectedKeys.has(li.dataset.selKey));
    });
    updateSelectionBar();
  }

  function sortListing(folders, objects) {
    const byName = (a, b) =>
      String(field(a, 'name', 'Name') || '').localeCompare(String(field(b, 'name', 'Name') || ''), undefined, {
        sensitivity: 'base',
      });
    const byDate = (a, b) => {
      const ta = Date.parse(field(a, 'updated_at', 'UpdatedAt') || field(a, 'created_at', 'CreatedAt') || 0) || 0;
      const tb = Date.parse(field(b, 'updated_at', 'UpdatedAt') || field(b, 'created_at', 'CreatedAt') || 0) || 0;
      return tb - ta;
    };
    const byType = (a, b) => {
      const na = String(field(a, 'name', 'Name') || '');
      const nb = String(field(b, 'name', 'Name') || '');
      const ea = na.includes('.') ? na.split('.').pop() : '';
      const eb = nb.includes('.') ? nb.split('.').pop() : '';
      return ea.localeCompare(eb) || byName(a, b);
    };
    const cmp = driveSort === 'date' ? byDate : driveSort === 'type' ? byType : byName;
    folders = (folders || []).slice().sort(byName);
    objects = (objects || []).slice().sort(cmp);
    return { folders, objects };
  }

  function currentFolderId() {
    return pathStack[pathStack.length - 1].id;
  }

  function parentKey(parentId) {
    return parentId || '';
  }

  function rememberFolder(f, fallbackParent) {
    const id = field(f, 'id', 'ID');
    if (!id) return null;
    const name = field(f, 'name', 'Name') || '(folder)';
    const parentId = field(f, 'parent_id', 'ParentID') || fallbackParent || '';
    const node = { id, name, parentId };
    folderIndex.set(id, node);
    return node;
  }

  function pathStackForFolder(folderId) {
    if (!folderId) return [{ id: '', name: 'Root' }];
    const chain = [];
    let cur = folderId;
    const seen = new Set();
    while (cur && !seen.has(cur)) {
      seen.add(cur);
      const node = folderIndex.get(cur);
      if (!node) break;
      chain.unshift({ id: node.id, name: node.name });
      cur = node.parentId || '';
    }
    return [{ id: '', name: 'Root' }].concat(chain);
  }

  async function ensureTreeChildren(parentId) {
    const key = parentKey(parentId);
    if (treeChildren.has(key)) return treeChildren.get(key);
    const folderKey = key || '_root';
    const res = await api('/folders/' + encodeURIComponent(folderKey) + '/children');
    if (!res.ok) {
      treeChildren.set(key, []);
      return [];
    }
    const data = await res.json();
    const folders = (data.folders || [])
      .map((f) => rememberFolder(f, key))
      .filter(Boolean)
      .sort((a, b) => a.name.localeCompare(b.name));
    treeChildren.set(key, folders);
    return folders;
  }

  function renderFolderTree() {
    const host = document.getElementById('folderTreeList');
    if (!host) return;
    host.innerHTML = '';

    function renderLevel(parentId, ul) {
      const kids = treeChildren.get(parentKey(parentId)) || [];
      kids.forEach((node) => {
        const li = document.createElement('li');
        const row = document.createElement('div');
        row.style.display = 'flex';
        row.style.alignItems = 'center';
        row.style.gap = '0.05rem';

        const twist = document.createElement('button');
        twist.type = 'button';
        twist.className = 'era-drive-tree-twist';
        const expanded = treeExpanded.has(node.id);
        const hasKidsCache = treeChildren.has(node.id);
        const childCount = hasKidsCache ? (treeChildren.get(node.id) || []).length : -1;
        twist.textContent = expanded ? '▾' : '▸';
        twist.title = expanded ? 'Collapse' : 'Expand';
        twist.addEventListener('click', async (ev) => {
          ev.stopPropagation();
          if (treeExpanded.has(node.id)) {
            treeExpanded.delete(node.id);
            renderFolderTree();
            return;
          }
          treeExpanded.add(node.id);
          await ensureTreeChildren(node.id);
          renderFolderTree();
        });
        if (childCount === 0) {
          twist.disabled = true;
          twist.textContent = '·';
        }

        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className =
          'era-drive-tree-item' +
          (!inTrash() && currentFolderId() === node.id ? ' active' : '');
        btn.setAttribute('data-icon', 'folder');
        btn.dataset.dropFolder = node.id;
        const label = document.createElement('span');
        label.textContent = node.name;
        btn.appendChild(label);
        btn.addEventListener('click', () => navigateToFolder(node.id));
        wireTreeDropTarget(btn, node.id);

        row.appendChild(twist);
        row.appendChild(btn);
        li.appendChild(row);

        if (expanded) {
          const nested = document.createElement('ul');
          renderLevel(node.id, nested);
          li.appendChild(nested);
        }
        ul.appendChild(li);
      });
    }

    const rootLi = document.createElement('li');
    const rootBtn = document.createElement('button');
    rootBtn.type = 'button';
    rootBtn.className =
      'era-drive-tree-item' + (!inTrash() && !currentFolderId() ? ' active' : '');
    rootBtn.setAttribute('data-icon', 'folder');
    rootBtn.dataset.dropFolder = '';
    const rootLabel = document.createElement('span');
    rootLabel.textContent = 'My Drive';
    rootBtn.appendChild(rootLabel);
    rootBtn.addEventListener('click', () => navigateToFolder(''));
    wireTreeDropTarget(rootBtn, '');
    rootLi.appendChild(rootBtn);
    host.appendChild(rootLi);

    const rootKids = document.createElement('ul');
    renderLevel('', rootKids);
    host.appendChild(rootKids);

    // Trash = virtual folder in the tree (enter/exit like any folder)
    const trashLi = document.createElement('li');
    trashLi.style.marginTop = '0.45rem';
    const trashBtn = document.createElement('button');
    trashBtn.type = 'button';
    trashBtn.className = 'era-drive-tree-item' + (inTrash() ? ' active' : '');
    trashBtn.setAttribute('data-icon', 'trash');
    trashBtn.dataset.dropTrash = '1';
    const trashLabel = document.createElement('span');
    trashLabel.textContent = 'Trash';
    trashBtn.appendChild(trashLabel);
    trashBtn.addEventListener('click', () => navigateToFolder(TRASH_ID));
    wireTreeDropTarget(trashBtn, TRASH_ID);
    trashLi.appendChild(trashBtn);
    host.appendChild(trashLi);

    if (window.EraOfficeIcons && EraOfficeIcons.mount) {
      EraOfficeIcons.mount(host);
    }
  }

  async function refreshFolderTree() {
    if (!localStorage.getItem('era_token')) {
      const host = document.getElementById('folderTreeList');
      if (host) host.innerHTML = '';
      return;
    }
    // Invalidate caches along the open path so renames/creates show up.
    treeChildren.delete('');
    pathStack.forEach((seg) => {
      if (seg.id) treeChildren.delete(seg.id);
    });
    await ensureTreeChildren('');
    // Prefetch expanded / path ancestors
    for (const seg of pathStack) {
      if (!seg.id) continue;
      treeExpanded.add(seg.id);
      await ensureTreeChildren(seg.id);
      // also expand parents for visibility
      let p = folderIndex.get(seg.id);
      while (p && p.parentId) {
        treeExpanded.add(p.parentId);
        await ensureTreeChildren(p.parentId);
        p = folderIndex.get(p.parentId);
      }
    }
    for (const id of Array.from(treeExpanded)) {
      if (id) await ensureTreeChildren(id);
    }
    renderFolderTree();
  }

  async function navigateToFolder(folderId) {
    if (folderId === TRASH_ID) {
      trashMode = true;
      pathStack = [{ id: TRASH_ID, name: 'Trash' }];
      clearSelection();
      await refreshFiles();
      return;
    }
    trashMode = false;
    pathStack = pathStackForFolder(folderId || '');
    // If we navigated to a known folder but parents missing, still open it as single segment.
    if (folderId && pathStack.length === 1) {
      const node = folderIndex.get(folderId);
      if (node) pathStack = [{ id: '', name: 'Root' }, { id: node.id, name: node.name }];
    }
    if (folderId) treeExpanded.add(folderId);
    clearSelection();
    await refreshFiles();
  }

  const ROLE_LABEL = { 1: 'Owner', 2: 'Read', 3: 'Write' };

  function field(obj, ...keys) {
    for (const k of keys) {
      if (obj && obj[k] != null && obj[k] !== '') return obj[k];
    }
    return '';
  }

  function currentUserId() {
    const token = localStorage.getItem('era_token') || '';
    try {
      const part = token.split('.')[1];
      const p = JSON.parse(atob(part.replace(/-/g, '+').replace(/_/g, '/')));
      return p.sub || '';
    } catch (_) {
      return '';
    }
  }

  function objectLockedBy(o) {
    return String(field(o, 'locked_by', 'LockedBy') || '');
  }

  function isLockedByOther(o) {
    const by = objectLockedBy(o);
    if (!by) return false;
    const me = currentUserId();
    return !me || by !== me;
  }

  function objectOwnerId(o) {
    return String(field(o, 'owner_user_id', 'OwnerUserID') || '');
  }

  /** Unlock allowed for locker or object owner (matches drive.CanUnlock). */
  function canUnlockObject(o) {
    const me = currentUserId();
    if (!me || !objectLockedBy(o)) return false;
    return objectLockedBy(o) === me || objectOwnerId(o) === me;
  }

  const PREVIEW_MAX_BYTES = 8 * 1024 * 1024;

  function isImagePreviewable(contentType, name) {
    const ct = String(contentType || '').toLowerCase().split(';')[0].trim();
    if (ct.startsWith('image/')) {
      return (
        ct === 'image/png' ||
        ct === 'image/jpeg' ||
        ct === 'image/jpg' ||
        ct === 'image/gif' ||
        ct === 'image/webp' ||
        ct.startsWith('image/')
      );
    }
    const lower = String(name || '').toLowerCase();
    return (
      lower.endsWith('.png') ||
      lower.endsWith('.jpg') ||
      lower.endsWith('.jpeg') ||
      lower.endsWith('.gif') ||
      lower.endsWith('.webp')
    );
  }

  async function confirmOpenWhileLocked(o, openKind) {
    if (!isLockedByOther(o)) return true;
    const by = objectLockedBy(o);
    return EraOfficeShell.confirmAction({
      title: 'File is locked',
      message:
        'This file is locked by ' +
        by +
        '. Open ' +
        (openKind || 'editor') +
        ' in read-only / caution mode? Writes may fail until unlocked.',
      okLabel: 'Open anyway',
    });
  }

  /** @returns {'docs'|'tables'|'pres'|'projects'|null} */
  function detectOpenKind(lowerName, contentType) {
    const ct = String(contentType || '').toLowerCase();
    if (ct.includes('wordprocessingml') || lowerName.endsWith('.erad') || lowerName.endsWith('.docx')) {
      return 'docs';
    }
    if (ct.includes('spreadsheetml') || lowerName.endsWith('.erat') || lowerName.endsWith('.xlsx')) {
      return 'tables';
    }
    if (ct.includes('presentationml') || lowerName.endsWith('.erap') || lowerName.endsWith('.pptx')) {
      return 'pres';
    }
    if (ct.includes('vnd.era.eraj') || lowerName.endsWith('.eraj')) {
      return 'projects';
    }
    return null;
  }

  async function api(path, opts = {}) {
    const url = '/api/v1/drive' + path;
    const res =
      window.EraOfficeShell && EraOfficeShell.authFetch
        ? await EraOfficeShell.authFetch(url, opts)
        : await (async () => {
            const token = localStorage.getItem('era_token') || '';
            const headers = Object.assign({}, opts.headers || {});
            if (token) headers.Authorization = 'Bearer ' + token;
            return fetch(url, Object.assign({}, opts, { headers }));
          })();
    if (res.status === 401) throw new Error('unauthorized');
    return res;
  }

  let driveViewMode = 'list';
  try {
    const v = localStorage.getItem('era_drive_view');
    if (v === 'grid' || v === 'list') driveViewMode = v;
  } catch (_) {}

  function applyDriveView() {
    const list = document.getElementById('files');
    const listBtn = document.getElementById('viewListBtn');
    const gridBtn = document.getElementById('viewGridBtn');
    if (list) {
      list.classList.toggle('era-drive-view-list', driveViewMode === 'list');
      list.classList.toggle('era-drive-view-grid', driveViewMode === 'grid');
    }
    if (listBtn) {
      listBtn.classList.toggle('active', driveViewMode === 'list');
      listBtn.setAttribute('aria-pressed', driveViewMode === 'list' ? 'true' : 'false');
    }
    if (gridBtn) {
      gridBtn.classList.toggle('active', driveViewMode === 'grid');
      gridBtn.setAttribute('aria-pressed', driveViewMode === 'grid' ? 'true' : 'false');
    }
    try { localStorage.setItem('era_drive_view', driveViewMode); } catch (_) {}
  }

  function setDriveStats(text) {
    const el = document.getElementById('driveStats');
    if (el) el.textContent = text || '—';
  }

  function closeNewMenu() {
    const menu = document.getElementById('newMenu');
    const btn = document.getElementById('newMenuBtn');
    if (menu) {
      menu.classList.remove('open');
      menu.hidden = true;
    }
    if (btn) btn.setAttribute('aria-expanded', 'false');
  }

  function toggleNewMenu() {
    const menu = document.getElementById('newMenu');
    const btn = document.getElementById('newMenuBtn');
    if (!menu || !btn) return;
    const open = menu.hidden;
    menu.hidden = !open;
    menu.classList.toggle('open', open);
    btn.setAttribute('aria-expanded', open ? 'true' : 'false');
  }

  async function createFolderFromUi(promptName) {
    let name = (document.getElementById('newFolderName').value || '').trim();
    if (promptName || !name) {
      const typed = await EraOfficeShell.promptText({
        title: 'New folder',
        label: 'Folder name',
        value: name || 'Untitled folder',
      });
      if (typed == null) return;
      name = String(typed).trim();
    }
    if (!name) {
      setAuthStatus('Folder name required', true);
      return;
    }
    if (inTrash()) {
      setAuthStatus('Leave Trash to create a folder', true);
      return;
    }
    const body = { name, parent_id: currentFolderId() };
    const res = await api('/folders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      setAuthStatus('Create folder failed: ' + res.status, true);
      return;
    }
    const input = document.getElementById('newFolderName');
    if (input) input.value = '';
    await refreshFiles();
    await refreshFolderTree();
  }

  function setAuthStatus(msg, isErr) {
    const el = document.getElementById('authStatus');
    if (window.EraOfficeShell && EraOfficeShell.toastStatus) {
      EraOfficeShell.toastStatus(el, msg, !!isErr);
      return;
    }
    if (!el) return;
    el.textContent = msg || '';
    el.className = 'era-status ' + (isErr ? 'err' : 'ok');
  }

  function setSearchClearVisible(visible) {
    const clearBtn = document.getElementById('driveSearchClear');
    if (clearBtn) clearBtn.hidden = !visible;
  }

  function renderBreadcrumb() {
    if (inTrash()) {
      const nav = document.getElementById('breadcrumb');
      if (!nav) return;
      nav.innerHTML = '';
      const root = document.createElement('button');
      root.type = 'button';
      root.className = 'link';
      root.textContent = 'My Drive';
      root.addEventListener('click', () => navigateToFolder(''));
      nav.appendChild(root);
      const sep = document.createElement('span');
      sep.className = 'sep';
      sep.textContent = '/';
      nav.appendChild(sep);
      const here = document.createElement('span');
      here.className = 'here';
      here.textContent = 'Trash';
      nav.appendChild(here);
      return;
    }
    const nav = document.getElementById('breadcrumb');
    nav.innerHTML = '';
    pathStack.forEach((seg, i) => {
      if (i > 0) {
        const sep = document.createElement('span');
        sep.className = 'sep';
        sep.textContent = '/';
        nav.appendChild(sep);
      }
      if (i === pathStack.length - 1) {
        const here = document.createElement('span');
        here.className = 'here';
        here.textContent = seg.name;
        nav.appendChild(here);
      } else {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'link';
        btn.textContent = seg.name;
        btn.addEventListener('click', () => {
          navigateToFolder(seg.id).catch(() => {});
        });
        nav.appendChild(btn);
      }
    });
  }

  function orderedKeysFromListing() {
    const keys = [];
    (lastListing.folders || []).forEach((f) => keys.push(itemKey('folder', field(f, 'id', 'ID'))));
    (lastListing.objects || []).forEach((o) => keys.push(itemKey('file', field(o, 'id', 'ID'))));
    return keys;
  }

  function renderFolder(list, f) {
    const id = field(f, 'id', 'ID');
    const name = field(f, 'name', 'Name') || '(folder)';
    const key = itemKey('folder', id);
    const li = document.createElement('li');
    li.className = 'folder' + (selectedKeys.has(key) ? ' selected' : '');
    li.dataset.folderId = id;
    li.dataset.selKey = key;
    const chk = document.createElement('input');
    chk.type = 'checkbox';
    chk.className = 'era-row-check';
    chk.checked = selectedKeys.has(key);
    chk.addEventListener('click', (ev) => {
      ev.stopPropagation();
      handleRowSelect(ev, 'folder', id, orderedKeysFromListing());
    });
    li.appendChild(chk);
    const nameEl = document.createElement('span');
    nameEl.className = 'name';
    nameEl.setAttribute('data-icon', 'folder');
    const openBtn = document.createElement('button');
    openBtn.type = 'button';
    openBtn.className = 'link';
    openBtn.textContent = name;
    openBtn.addEventListener('click', (ev) => {
      if (ev.shiftKey || ev.ctrlKey || ev.metaKey) {
        ev.preventDefault();
        handleRowSelect(ev, 'folder', id, orderedKeysFromListing());
        return;
      }
      const parent = currentFolderId();
      rememberFolder(f, parent);
      if (parent) treeExpanded.add(parent);
      pathStack.push({ id, name });
      treeExpanded.add(id);
      clearSelection();
      refreshFiles();
    });
    nameEl.appendChild(openBtn);
    li.appendChild(nameEl);
    li.addEventListener('click', (ev) => {
      if (ev.target.closest('button, a, input, .era-icon-btn')) return;
      handleRowSelect(ev, 'folder', id, orderedKeysFromListing());
    });
    const meta = document.createElement('span');
    meta.className = 'meta';
    meta.textContent = 'Folder';
    li.appendChild(meta);
    const actions = ensureActions(li);
    appendIconAction(actions, 'Move', 'move', () => openMoveDialog(id, name, 'folder'));
    appendIconAction(actions, 'Delete', 'trash', () => trashOne('folder', id, name));
    appendMoreMenu(actions, [
      { label: 'Rename', icon: 'rename', fn: () => renameFolder(id, name) },
    ]);
    wireRowDrag(li, 'folder', id);
    wireTreeDropTarget(li, id);
    list.appendChild(li);
  }

  function renderObject(list, o) {
    const id = field(o, 'id', 'ID');
    const name = field(o, 'name', 'Name') || '(file)';
    const size = field(o, 'size_bytes', 'SizeBytes') || 0;
    const ver = field(o, 'version', 'Version') || 1;
    const lockedBy = objectLockedBy(o);
    const lockedByOther = isLockedByOther(o);
    const key = itemKey('file', id);
    const li = document.createElement('li');
    li.className = 'file' + (lockedBy ? ' locked' : '') + (selectedKeys.has(key) ? ' selected' : '');
    li.dataset.objectId = id;
    li.dataset.selKey = key;
    if (lockedBy) li.dataset.lockedBy = lockedBy;

    const chk = document.createElement('input');
    chk.type = 'checkbox';
    chk.className = 'era-row-check';
    chk.checked = selectedKeys.has(key);
    chk.addEventListener('click', (ev) => {
      ev.stopPropagation();
      handleRowSelect(ev, 'file', id, orderedKeysFromListing());
    });
    li.appendChild(chk);

    const nameEl = document.createElement('span');
    nameEl.className = 'name';
    const lower = String(name).toLowerCase();
    const ct = String(field(o, 'content_type', 'ContentType') || '').toLowerCase();
    const openKind = detectOpenKind(lower, ct);
    const OPEN_PATH = { docs: '/docs/', tables: '/tables/', pres: '/presentations/', projects: '/projects/' };
    const KIND_ICON = { docs: 'navDocs', tables: 'navTables', pres: 'navPres', projects: 'navProjects' };
    nameEl.setAttribute('data-icon', KIND_ICON[openKind] || 'file');

    if (openKind) {
      const openLink = document.createElement('button');
      openLink.type = 'button';
      openLink.className = 'link';
      openLink.textContent = name;
      openLink.addEventListener('click', (ev) => {
        if (ev.shiftKey || ev.ctrlKey || ev.metaKey) {
          ev.preventDefault();
          handleRowSelect(ev, 'file', id, orderedKeysFromListing());
          return;
        }
        void (async () => {
          if (!(await confirmOpenWhileLocked(o, 'Open'))) return;
          location.href = OPEN_PATH[openKind] + encodeURIComponent(id);
        })();
      });
      nameEl.appendChild(openLink);
    } else {
      const span = document.createElement('span');
      span.textContent = name;
      nameEl.appendChild(span);
    }
    li.addEventListener('click', (ev) => {
      if (ev.target.closest('button, a, input, .era-icon-btn, select')) return;
      handleRowSelect(ev, 'file', id, orderedKeysFromListing());
    });
    if (lockedBy) {
      const badge = document.createElement('span');
      badge.className = 'lock-badge';
      badge.setAttribute('data-icon', 'lock');
      badge.title = 'Locked by ' + lockedBy;
      badge.setAttribute('aria-label', 'Locked by ' + lockedBy);
      badge.textContent = 'Locked';
      nameEl.appendChild(badge);
    }
    li.appendChild(nameEl);

    const meta = document.createElement('span');
    meta.className = 'meta';
    meta.textContent = size + ' B · v' + ver + (lockedBy ? ' · locked by ' + lockedBy : '');
    li.appendChild(meta);

    const actions = ensureActions(li);
    if (openKind) {
      appendIconAction(actions, 'Open', 'open', () => {
        void (async () => {
          if (!(await confirmOpenWhileLocked(o, 'Open'))) return;
          location.href = OPEN_PATH[openKind] + encodeURIComponent(id);
        })();
      });
    } else {
      appendIconAction(actions, 'Open with…', 'open', () => {
        void (async () => {
          const pick = await EraOfficeShell.chooseOption({
            title: 'Open with…',
            message: 'Choose an app',
            options: [
              { value: 'docs', label: 'Documents', hint: 'ERA Docs' },
              { value: 'tables', label: 'Tables', hint: 'ERA Tables' },
              { value: 'presentations', label: 'Presentations', hint: 'ERA Presentations' },
              { value: 'projects', label: 'Projects', hint: 'ERA Projects' },
            ],
            value: 'docs',
          });
          if (!pick) return;
          const map = {
            docs: 'docs',
            tables: 'tables',
            presentations: 'pres',
            projects: 'projects',
          };
          const kind = map[pick];
          if (!kind) {
            setAuthStatus('Unknown app', true);
            return;
          }
          if (!(await confirmOpenWhileLocked(o, 'Open'))) return;
          location.href = OPEN_PATH[kind] + encodeURIComponent(id);
        })();
      });
    }

    if (lockedByOther) {
      appendIconAction(
        actions,
        'Move disabled — locked by ' + lockedBy,
        'move',
        () => setAuthStatus('File locked by ' + lockedBy + ' — move disabled', true),
        true
      );
      appendIconAction(
        actions,
        'Delete disabled — locked by ' + lockedBy,
        'trash',
        () => setAuthStatus('File locked by ' + lockedBy + ' — delete disabled', true),
        true
      );
    } else {
      appendIconAction(actions, 'Move', 'move', () => openMoveDialog(id, name, 'file'));
      appendIconAction(actions, 'Delete', 'trash', () => trashOne('file', id, name));
    }

    const moreItems = [
      { label: 'Preview', icon: 'image', fn: () => showPreview(id, name) },
      { label: 'Download', icon: 'download', fn: () => downloadObject(id, name) },
      { label: 'Versions', icon: 'history', fn: () => showVersions(id, name) },
      { label: 'Share', icon: 'share', fn: () => openShareDialog(id, name) },
    ];
    if (lockedBy) {
      if (canUnlockObject(o)) {
        moreItems.push({ label: 'Unlock', icon: 'unlock', fn: () => toggleLock(id, name, false) });
      } else {
        moreItems.push({
          label: 'Unlock (locked by ' + lockedBy + ')',
          icon: 'unlock',
          disabled: true,
          fn: () => setAuthStatus('Only locker or owner can unlock (locked by ' + lockedBy + ')', true),
        });
      }
    } else {
      moreItems.push({ label: 'Lock', icon: 'lock', fn: () => toggleLock(id, name, true) });
    }
    if (lockedByOther) {
      moreItems.push({
        label: 'Rename (locked)',
        icon: 'rename',
        disabled: true,
        fn: () => setAuthStatus('File locked by ' + lockedBy + ' — rename disabled', true),
      });
    } else {
      moreItems.push({ label: 'Rename', icon: 'rename', fn: () => renameObject(id, name) });
    }
    appendMoreMenu(actions, moreItems);
    if (!lockedByOther) wireRowDrag(li, 'file', id);

    list.appendChild(li);
  }

  function renderListing(folders, objects, opts) {
    const sorted = sortListing(folders, objects);
    lastListing = sorted;
    const list = document.getElementById('files');
    list.innerHTML = '';
    const empty = !sorted.folders.length && !sorted.objects.length;
    if (empty) {
      const li = document.createElement('li');
      li.className = 'era-drive-empty';
      const isSearch = !!(opts && opts.search);
      const isTrash = inTrash();
      let title = 'This folder is empty';
      let hint = 'Upload a file or create a folder to get started.';
      if (isSearch) {
        title = 'No results';
        hint = 'Try another search term, or clear search to browse folders.';
      } else if (isTrash) {
        title = 'Trash is empty';
        hint = 'Items you move to Trash will appear here.';
      }
      li.innerHTML =
        '<div class="era-drive-empty-card">' +
        '<strong>' +
        title +
        '</strong>' +
        '<p class="era-hint">' +
        hint +
        '</p>' +
        (isSearch || isTrash
          ? ''
          : '<div class="era-drive-empty-actions">' +
            '<button type="button" class="era-btn era-btn-primary" id="emptyUploadBtn">Upload</button>' +
            '<button type="button" class="era-btn" id="emptyNewBtn">New</button>' +
            '</div>') +
        '</div>';
      list.appendChild(li);
      const up = document.getElementById('emptyUploadBtn');
      const neu = document.getElementById('emptyNewBtn');
      if (up) {
        up.onclick = () => {
          const pick = document.getElementById('pickFileBtn');
          if (pick) pick.click();
        };
      }
      if (neu) {
        neu.onclick = () => {
          const btn = document.getElementById('newMenuBtn');
          if (btn) btn.click();
        };
      }
    } else {
      for (const f of sorted.folders) renderFolder(list, f);
      for (const o of sorted.objects) renderObject(list, o);
    }
    if (window.EraOfficeIcons && EraOfficeIcons.mount) {
      EraOfficeIcons.mount(list);
    }
    updateSelectionBar();
  }

  async function trashOne(kind, id, name, skipConfirm) {
    if (!skipConfirm) {
      const ok = await EraOfficeShell.confirmAction({
        title: 'Move to Trash',
        message: 'Move "' + name + '" to Trash?',
        okLabel: 'Move to Trash',
        danger: true,
      });
      if (!ok) return;
    }
    const path =
      kind === 'folder'
        ? '/folders/' + encodeURIComponent(id) + '/trash'
        : '/objects/' + encodeURIComponent(id) + '/trash';
    const res = await api(path, { method: 'POST' });
    if (!res.ok) {
      setAuthStatus('Trash failed: ' + res.status, true);
      return;
    }
    selectedKeys.delete(itemKey(kind === 'folder' ? 'folder' : 'file', id));
    setAuthStatus('Moved to Trash', false);
    await refreshFiles();
    await refreshFolderTree();
  }

  async function trashSelected(skipConfirm) {
    const keys = Array.from(selectedKeys);
    if (!keys.length) return;
    if (!skipConfirm) {
      const ok = await EraOfficeShell.confirmAction({
        title: 'Move to Trash',
        message: 'Move ' + keys.length + ' item(s) to Trash?',
        okLabel: 'Move to Trash',
        danger: true,
      });
      if (!ok) return;
    }
    for (const key of keys) {
      const id = key.slice(2);
      const path = key.startsWith('f:')
        ? '/folders/' + encodeURIComponent(id) + '/trash'
        : '/objects/' + encodeURIComponent(id) + '/trash';
      const res = await api(path, { method: 'POST' });
      if (!res.ok && window.EraOfficeShell && EraOfficeShell.handleUnauthorized(res)) return;
    }
    clearSelection();
    setAuthStatus('Moved to Trash', false);
    await refreshFiles();
    await refreshFolderTree();
  }

  async function moveSelected() {
    const keys = Array.from(selectedKeys);
    if (!keys.length) return;
    const first = keys[0];
    const kind = first.startsWith('f:') ? 'folder' : 'file';
    const id = first.slice(2);
    const name = keys.length === 1 ? id : keys.length + ' items';
    await openMoveDialog(id, name, kind, keys);
  }

  function renderTrashListing(folders, objects) {
    lastListing = { folders: folders || [], objects: objects || [] };
    const list = document.getElementById('files');
    list.innerHTML = '';
    if (!lastListing.folders.length && !lastListing.objects.length) {
      renderListing([], [], { search: false });
      // Force trash empty copy (inTrash() is true here).
      return;
    }
    for (const f of lastListing.folders) {
      const id = field(f, 'id', 'ID');
      const name = field(f, 'name', 'Name') || '(folder)';
      const li = document.createElement('li');
      li.className = 'folder';
      li.innerHTML =
        '<span class="name" data-icon="folder"><span>' +
        name +
        '</span></span><span class="meta">Folder</span>';
      const actions = ensureActions(li);
      appendIconAction(actions, 'Restore', 'undo', () => restoreOne('folder', id));
      list.appendChild(li);
    }
    for (const o of lastListing.objects) {
      const id = field(o, 'id', 'ID');
      const name = field(o, 'name', 'Name') || '(file)';
      const li = document.createElement('li');
      li.className = 'file';
      li.innerHTML =
        '<span class="name" data-icon="file"><span>' +
        name +
        '</span></span><span class="meta">File</span>';
      const actions = ensureActions(li);
      appendIconAction(actions, 'Restore', 'undo', () => restoreOne('file', id));
      list.appendChild(li);
    }
    if (window.EraOfficeIcons && EraOfficeIcons.mount) EraOfficeIcons.mount(list);
    updateSelectionBar();
  }

  async function restoreOne(kind, id) {
    const path =
      kind === 'folder'
        ? '/folders/' + encodeURIComponent(id) + '/restore'
        : '/objects/' + encodeURIComponent(id) + '/restore';
    const res = await api(path, { method: 'POST' });
    if (!res.ok) {
      setAuthStatus('Restore failed: ' + res.status, true);
      return;
    }
    setAuthStatus('Restored', false);
    await refreshFiles();
    await refreshFolderTree();
  }

  async function refreshFiles() {
    setSearchClearVisible(false);
    renderBreadcrumb();
    if (!localStorage.getItem('era_token')) {
      document.getElementById('files').innerHTML = '';
      const tree = document.getElementById('folderTreeList');
      if (tree) tree.innerHTML = '';
      setAuthStatus('Not signed in', true);
      return;
    }
    const hint = document.getElementById('driveHeaderHint');
    if (inTrash()) {
      trashMode = true;
      if (hint) hint.textContent = 'Trash';
      const res = await api('/trash');
      if (!res.ok) {
        setAuthStatus('Trash list failed: ' + res.status, true);
        return;
      }
      const data = await res.json();
      const folders = data.folders || [];
      const objects = data.objects || [];
      const stats = folders.length + ' folder(s) · ' + objects.length + ' file(s) in Trash';
      setAuthStatus('Trash — ' + stats, false);
      setDriveStats(stats);
      renderTrashListing(folders, objects);
      renderFolderTree();
      return;
    }
    trashMode = false;
    if (hint) hint.textContent = 'Folders · files · Office objects';
    const folderKey = currentFolderId() || '_root';
    const res = await api('/folders/' + encodeURIComponent(folderKey) + '/children');
    if (!res.ok) {
      setAuthStatus('Drive list failed: ' + res.status, true);
      return;
    }
    const data = await res.json();
    const folders = data.folders || [];
    const objects = data.objects || [];
    const parent = currentFolderId() || '';
    const remembered = folders
      .map((f) => rememberFolder(f, parent))
      .filter(Boolean)
      .sort((a, b) => a.name.localeCompare(b.name));
    treeChildren.set(parent, remembered);
    const stats =
      folders.length + ' folder(s) · ' + objects.length + ' file(s)';
    setAuthStatus('Signed in — ' + stats, false);
    setDriveStats(stats);
    renderListing(folders, objects);
    applyDriveView();
    await refreshFolderTree();
  }

  async function runSearch() {
    const input = document.getElementById('driveSearchInput');
    const q = (input && input.value ? input.value : '').trim();
    if (!q) {
      await refreshFiles();
      return;
    }
    if (!localStorage.getItem('era_token')) {
      setAuthStatus('Sign in first', true);
      return;
    }
    const res = await api('/search?q=' + encodeURIComponent(q));
    if (!res.ok) {
      setAuthStatus('Search failed: ' + res.status, true);
      return;
    }
    const data = await res.json();
    const folders = data.folders || [];
    const objects = data.objects || [];
    setSearchClearVisible(true);
    const n = folders.length + objects.length;
    setAuthStatus('Search: ' + n + ' results', false);
    renderListing(folders, objects, { search: true });
  }

  async function clearSearch() {
    const input = document.getElementById('driveSearchInput');
    if (input) input.value = '';
    await refreshFiles();
  }

  function appendIconAction(host, label, icon, fn, disabled) {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'era-icon-btn' + (disabled ? ' disabled' : '');
    btn.setAttribute('data-icon', icon);
    btn.setAttribute('title', label);
    btn.setAttribute('aria-label', label);
    if (disabled) {
      btn.setAttribute('aria-disabled', 'true');
    }
    btn.addEventListener('click', () => fn());
    host.appendChild(btn);
  }

  function appendTextAction(host, label, icon, fn, primary, danger, disabled) {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className =
      'era-btn' +
      (primary ? ' era-btn-primary' : '') +
      (danger ? ' era-btn-danger' : '') +
      (disabled ? ' disabled' : '');
    btn.setAttribute('data-icon', icon);
    btn.setAttribute('title', label);
    btn.setAttribute('aria-label', label);
    btn.textContent = label;
    if (disabled) btn.setAttribute('aria-disabled', 'true');
    btn.addEventListener('click', (ev) => {
      ev.stopPropagation();
      if (disabled) return;
      fn();
    });
    host.appendChild(btn);
  }

  function closeAllMoreMenus(except) {
    document.querySelectorAll('.era-more-menu.open').forEach((m) => {
      if (m !== except) m.classList.remove('open');
    });
  }

  function appendMoreMenu(host, items) {
    if (!items || !items.length) return;
    const wrap = document.createElement('div');
    wrap.className = 'era-more-wrap';
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'era-icon-btn';
    btn.setAttribute('data-icon', 'more');
    btn.setAttribute('title', 'More actions');
    btn.setAttribute('aria-label', 'More actions');
    btn.setAttribute('aria-haspopup', 'true');
    const menu = document.createElement('div');
    menu.className = 'era-more-menu';
    menu.setAttribute('role', 'menu');
    for (const item of items) {
      const mi = document.createElement('button');
      mi.type = 'button';
      mi.setAttribute('role', 'menuitem');
      mi.setAttribute('data-icon', item.icon || 'more');
      mi.textContent = item.label;
      if (item.disabled) mi.classList.add('disabled');
      mi.addEventListener('click', (ev) => {
        ev.stopPropagation();
        menu.classList.remove('open');
        if (item.disabled) {
          if (item.fn) item.fn();
          return;
        }
        item.fn();
      });
      menu.appendChild(mi);
    }
    btn.addEventListener('click', (ev) => {
      ev.stopPropagation();
      const open = !menu.classList.contains('open');
      closeAllMoreMenus(menu);
      menu.classList.toggle('open', open);
    });
    wrap.appendChild(btn);
    wrap.appendChild(menu);
    host.appendChild(wrap);
  }

  async function toggleLock(id, name, locked) {
    const res = await api('/objects/' + encodeURIComponent(id), {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ locked }),
    });
    if (!res.ok) {
      setAuthStatus((locked ? 'Lock' : 'Unlock') + ' failed: ' + res.status, true);
      return;
    }
    await refreshFiles();
    setAuthStatus((locked ? 'Locked ' : 'Unlocked ') + name, false);
  }

  function appendAction(li, label, fn) {
    appendIconAction(li, label, 'more', fn);
  }

  function ensureActions(li) {
    let host = li.querySelector('.era-row-actions');
    if (!host) {
      host = document.createElement('div');
      host.className = 'era-row-actions';
      li.appendChild(host);
    }
    return host;
  }

  async function renameObject(id, currentName) {
    const next = await EraOfficeShell.promptText({
      title: 'Rename file',
      label: 'Name',
      value: currentName,
    });
    if (next == null) return;
    const name = next.trim();
    if (!name || name === currentName) return;
    const res = await api('/objects/' + encodeURIComponent(id), {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    });
    if (!res.ok) {
      const msg = res.status === 409 ? 'Rename failed: file is locked' : 'Rename failed: ' + res.status;
      setAuthStatus(msg, true);
      return;
    }
    setAuthStatus('Renamed to ' + name, false);
    await refreshFiles();
  }

  async function renameFolder(id, currentName) {
    const next = await EraOfficeShell.promptText({
      title: 'Rename folder',
      label: 'Name',
      value: currentName,
    });
    if (next == null) return;
    const name = next.trim();
    if (!name || name === currentName) return;
    const res = await api('/folders/' + encodeURIComponent(id), {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    });
    if (!res.ok) {
      setAuthStatus('Folder rename failed: ' + res.status, true);
      return;
    }
    setAuthStatus('Folder renamed to ' + name, false);
    await refreshFiles();
  }

  async function fillMoveFolderSelect(excludeFolderId) {
    const sel = document.getElementById('moveFolderSelect');
    if (!sel) return;
    sel.innerHTML = '';
    const rootOpt = document.createElement('option');
    rootOpt.value = '';
    rootOpt.textContent = 'My Drive (root)';
    sel.appendChild(rootOpt);

    async function walk(parentId, depth) {
      await ensureTreeChildren(parentId);
      const kids = treeChildren.get(parentKey(parentId)) || [];
      for (const node of kids) {
        if (excludeFolderId && node.id === excludeFolderId) continue;
        const opt = document.createElement('option');
        opt.value = node.id;
        opt.textContent = (depth ? '— '.repeat(depth) : '') + node.name;
        sel.appendChild(opt);
        await walk(node.id, depth + 1);
      }
    }
    await walk('', 0);
    const cur = inTrash() ? '' : currentFolderId();
    if ([...sel.options].some((o) => o.value === cur)) sel.value = cur;
  }

  async function openMoveDialog(id, name, kind, multiKeys) {
    moveState = { id, name, kind, multi: multiKeys || null };
    const title = document.getElementById('moveTitle');
    if (title) title.textContent = name;
    await fillMoveFolderSelect(kind === 'folder' ? id : '');
    const panel = document.getElementById('movePanel');
    if (panel) panel.classList.add('open');
    const share = document.getElementById('sharePanel');
    if (share) share.classList.remove('open');
  }

  async function moveKeysToFolder(keys, destFolderId) {
    for (const key of keys) {
      const id = key.slice(2);
      const isFolder = key.startsWith('f:');
      if (isFolder && id === destFolderId) continue;
      const res = isFolder
        ? await api('/folders/' + encodeURIComponent(id), {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ parent_id: destFolderId || '' }),
          })
        : await api('/objects/' + encodeURIComponent(id), {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ folder_id: destFolderId || '' }),
          });
      if (!res.ok) {
        if (window.EraOfficeShell && EraOfficeShell.handleUnauthorized(res)) return false;
        const msg = res.status === 409 ? 'Move failed: file is locked' : 'Move failed: ' + res.status;
        setAuthStatus(msg, true);
        return false;
      }
    }
    return true;
  }

  async function confirmMove() {
    if (!moveState) return;
    const dest = document.getElementById('moveFolderSelect').value;
    const keys = moveState.multi || [itemKey(moveState.kind, moveState.id)];
    const ok = await moveKeysToFolder(keys, dest);
    if (!ok) return;
    setAuthStatus('Moved ' + moveState.name, false);
    moveState = null;
    clearSelection();
    document.getElementById('movePanel').classList.remove('open');
    await refreshFiles();
    await refreshFolderTree();
  }

  function dragPayloadFromKeys(keys) {
    return JSON.stringify({ keys: keys });
  }

  function parseDragPayload(dt) {
    try {
      const raw = dt.getData('application/x-era-drive') || dt.getData('text/plain');
      const data = JSON.parse(raw);
      if (data && Array.isArray(data.keys)) return data.keys;
    } catch (_) {}
    return null;
  }

  function wireRowDrag(li, kind, id) {
    li.draggable = true;
    li.addEventListener('dragstart', (ev) => {
      const key = itemKey(kind, id);
      let keys = Array.from(selectedKeys);
      if (!keys.includes(key)) keys = [key];
      ev.dataTransfer.setData('application/x-era-drive', dragPayloadFromKeys(keys));
      ev.dataTransfer.setData('text/plain', dragPayloadFromKeys(keys));
      ev.dataTransfer.effectAllowed = 'move';
      li.classList.add('dragging');
    });
    li.addEventListener('dragend', () => li.classList.remove('dragging'));
  }

  function wireTreeDropTarget(el, destId) {
    el.addEventListener('dragover', (ev) => {
      if (![...ev.dataTransfer.types].some((t) => t === 'application/x-era-drive' || t === 'text/plain')) {
        return;
      }
      ev.preventDefault();
      ev.dataTransfer.dropEffect = 'move';
      el.classList.add('era-drive-drop-target');
    });
    el.addEventListener('dragleave', () => el.classList.remove('era-drive-drop-target'));
    el.addEventListener('drop', async (ev) => {
      ev.preventDefault();
      el.classList.remove('era-drive-drop-target');
      const keys = parseDragPayload(ev.dataTransfer);
      if (!keys || !keys.length) return;
      if (destId === TRASH_ID) {
        for (const key of keys) selectedKeys.add(key);
        await trashSelected(true);
        return;
      }
      const ok = await moveKeysToFolder(keys, destId || '');
      if (!ok) return;
      clearSelection();
      setAuthStatus('Moved', false);
      await refreshFiles();
      await refreshFolderTree();
    });
  }

  function renderAclList() {
    const ul = document.getElementById('aclList');
    ul.innerHTML = '';
    if (!shareState) return;
    for (let i = 0; i < shareState.entries.length; i++) {
      const e = shareState.entries[i];
      const li = document.createElement('li');
      const role = Number(e.role != null ? e.role : e.Role);
      const principal = e.principal || e.Principal || '';
      li.textContent = principal + ' — ' + (ROLE_LABEL[role] || role);
      const rm = document.createElement('button');
      rm.type = 'button';
      rm.textContent = 'Remove';
      rm.addEventListener('click', () => {
        shareState.entries.splice(i, 1);
        renderAclList();
      });
      li.appendChild(rm);
      ul.appendChild(li);
    }
  }

  async function openShareDialog(id, name) {
    const res = await api('/objects/' + encodeURIComponent(id) + '/meta');
    if (!res.ok) {
      setAuthStatus('Load ACL failed: ' + res.status, true);
      return;
    }
    const data = await res.json();
    const acl = data.acl || [];
    shareState = {
      id,
      name,
      entries: acl.map((e) => ({
        principal: e.principal || e.Principal || '',
        role: Number(e.role != null ? e.role : e.Role) || 2,
      })),
    };
    document.getElementById('shareTitle').textContent = name;
    renderAclList();
    document.getElementById('sharePanel').classList.add('open');
    document.getElementById('movePanel').classList.remove('open');
  }

  async function saveAcl() {
    if (!shareState) return;
    const entries = shareState.entries.map((e) => ({
      principal: e.principal,
      role: Number(e.role),
    }));
    const res = await api('/objects/' + encodeURIComponent(shareState.id) + '/acl', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ entries }),
    });
    if (!res.ok) {
      setAuthStatus('Save ACL failed: ' + res.status, true);
      return;
    }
    setAuthStatus('ACL saved for ' + shareState.name, false);
  }

  async function downloadObject(id, name) {
    const res = await api('/objects/' + encodeURIComponent(id));
    if (!res.ok) {
      setAuthStatus('Download failed: ' + res.status, true);
      return;
    }
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = name || id;
    a.click();
    URL.revokeObjectURL(url);
  }

  function closePreview() {
    if (previewObjectUrl) {
      URL.revokeObjectURL(previewObjectUrl);
      previewObjectUrl = null;
    }
    const body = document.getElementById('previewBody');
    if (body) body.innerHTML = '';
    const hint = document.getElementById('previewSizeHint');
    if (hint) {
      hint.hidden = true;
      hint.textContent = '';
    }
    const panel = document.getElementById('previewPanel');
    if (panel) panel.classList.remove('open');
    const main = document.getElementById('driveMain');
    if (main) main.classList.remove('preview-open');
  }

  function isTextPreviewable(contentType, name) {
    const ct = String(contentType || '').toLowerCase().split(';')[0].trim();
    if (ct.startsWith('text/')) return true;
    const lower = String(name || '').toLowerCase();
    return (
      lower.endsWith('.txt') ||
      lower.endsWith('.md') ||
      lower.endsWith('.csv') ||
      lower.endsWith('.json') ||
      lower.endsWith('.html') ||
      lower.endsWith('.htm')
    );
  }

  function showPreviewFallback(body, id, name, message) {
    const msg = document.createElement('p');
    msg.className = 'era-hint';
    msg.textContent = message || 'No inline preview; use Download';
    body.appendChild(msg);
    const dlBtn = document.createElement('button');
    dlBtn.type = 'button';
    dlBtn.className = 'era-btn';
    dlBtn.setAttribute('data-icon', 'download');
    dlBtn.textContent = 'Download';
    dlBtn.addEventListener('click', () => {
      downloadObject(id, name).catch(() => {});
    });
    body.appendChild(dlBtn);
    if (window.EraOfficeIcons && EraOfficeIcons.mount) {
      EraOfficeIcons.mount(body);
    }
  }

  async function showPreview(id, name) {
    closePreview();
    const panel = document.getElementById('previewPanel');
    const main = document.getElementById('driveMain');
    const title = document.getElementById('previewTitle');
    const body = document.getElementById('previewBody');
    const hint = document.getElementById('previewSizeHint');
    title.textContent = name;
    body.innerHTML = '';
    const res = await api('/objects/' + encodeURIComponent(id));
    if (!res.ok) {
      setAuthStatus('Preview failed: ' + res.status, true);
      return;
    }
    const contentType = res.headers.get('Content-Type') || '';
    const lenHdr = res.headers.get('Content-Length');
    const declared = lenHdr ? parseInt(lenHdr, 10) : NaN;
    if (Number.isFinite(declared) && declared > PREVIEW_MAX_BYTES) {
      if (hint) {
        hint.hidden = false;
        hint.textContent = 'File too large for inline preview (' + Math.round(declared / 1048576) + ' MB > 8 MB).';
      }
      showPreviewFallback(body, id, name, 'Preview skipped — file exceeds 8 MB size cap.');
      if (main) main.classList.add('preview-open');
      panel.classList.add('open');
      return;
    }
    const blob = await res.blob();
    if (blob.size > PREVIEW_MAX_BYTES) {
      if (hint) {
        hint.hidden = false;
        hint.textContent = 'File too large for inline preview (' + Math.round(blob.size / 1048576) + ' MB > 8 MB).';
      }
      showPreviewFallback(body, id, name, 'Preview skipped — file exceeds 8 MB size cap.');
      if (main) main.classList.add('preview-open');
      panel.classList.add('open');
      return;
    }
    const ct = (contentType || blob.type || '').toLowerCase().split(';')[0].trim();

    if (isImagePreviewable(ct, name)) {
      previewObjectUrl = URL.createObjectURL(blob);
      const img = document.createElement('img');
      img.src = previewObjectUrl;
      img.alt = name || 'Preview';
      body.appendChild(img);
    } else if (isTextPreviewable(ct, name)) {
      const text = await blob.text();
      const pre = document.createElement('pre');
      pre.textContent = text;
      body.appendChild(pre);
    } else {
      showPreviewFallback(body, id, name, 'No inline preview for this type; use Download or Open with…');
    }
    if (main) main.classList.add('preview-open');
    panel.classList.add('open');
  }

  async function showVersions(id, name) {
    const panel = document.getElementById('versionsPanel');
    const title = document.getElementById('versionsTitle');
    const ol = document.getElementById('versionsList');
    title.textContent = name;
    ol.innerHTML = '';
    const res = await api('/objects/' + encodeURIComponent(id) + '/versions');
    if (!res.ok) {
      setAuthStatus('Versions failed: ' + res.status, true);
      return;
    }
    const data = await res.json();
    const versions = data.versions || [];
    if (!versions.length) {
      const li = document.createElement('li');
      li.textContent = 'No version history (current object only).';
      ol.appendChild(li);
    }
    for (const v of versions) {
      const li = document.createElement('li');
      const n = field(v, 'version', 'Version');
      const sz = field(v, 'size_bytes', 'SizeBytes');
      const at = field(v, 'created_at', 'CreatedAt');
      li.textContent = 'v' + n + ' — ' + sz + ' B' + (at ? ' — ' + at : '');
      ol.appendChild(li);
    }
    panel.classList.add('open');
  }

  document.getElementById('closeVersionsBtn').addEventListener('click', () => {
    document.getElementById('versionsPanel').classList.remove('open');
  });
  document.getElementById('closePreviewBtn').addEventListener('click', () => {
    closePreview();
  });
  document.getElementById('closeShareBtn').addEventListener('click', () => {
    document.getElementById('sharePanel').classList.remove('open');
    shareState = null;
  });
  document.getElementById('closeMoveBtn').addEventListener('click', () => {
    document.getElementById('movePanel').classList.remove('open');
    moveState = null;
  });
  document.getElementById('moveConfirmBtn').addEventListener('click', () => {
    confirmMove().catch(() => {});
  });
  const movePanelEl = document.getElementById('movePanel');
  if (movePanelEl) {
    movePanelEl.addEventListener('click', (ev) => {
      if (ev.target === movePanelEl) {
        movePanelEl.classList.remove('open');
        moveState = null;
      }
    });
  }
  document.addEventListener('keydown', (ev) => {
    if (ev.key === 'Escape') {
      const panel = document.getElementById('movePanel');
      if (panel && panel.classList.contains('open')) {
        panel.classList.remove('open');
        moveState = null;
        return;
      }
    }
    if (inTrash()) return;
    const tag = (ev.target && ev.target.tagName) || '';
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || (ev.target && ev.target.isContentEditable)) {
      return;
    }
    if (ev.key === 'Delete') {
      if (!selectedKeys.size) return;
      ev.preventDefault();
      trashSelected(false).catch(() => {});
    }
  });
  document.getElementById('aclAddBtn').addEventListener('click', () => {
    if (!shareState) return;
    const principal = document.getElementById('aclPrincipal').value.trim();
    const role = Number(document.getElementById('aclRole').value) || 2;
    if (!principal) {
      setAuthStatus('Principal required', true);
      return;
    }
    shareState.entries.push({ principal, role });
    document.getElementById('aclPrincipal').value = '';
    renderAclList();
  });
  document.getElementById('aclSaveBtn').addEventListener('click', () => {
    saveAcl().catch(() => {});
  });

  const driveSearchBtn = document.getElementById('driveSearchBtn');
  if (driveSearchBtn) {
    driveSearchBtn.addEventListener('click', () => {
      runSearch().catch(() => {});
    });
  }
  const driveSearchClear = document.getElementById('driveSearchClear');
  if (driveSearchClear) {
    driveSearchClear.addEventListener('click', () => {
      clearSearch().catch(() => {});
    });
  }
  const driveSearchInput = document.getElementById('driveSearchInput');
  if (driveSearchInput) {
    driveSearchInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        runSearch().catch(() => {});
      }
    });
  }

  async function createOfficeObject(apiPath, name, hrefPrefix, label) {
    if (!localStorage.getItem('era_token')) {
      setAuthStatus('Sign in first', true);
      return;
    }
    const token = localStorage.getItem('era_token') || '';
    let tenantId = 't-demo';
    let userId = 'u-alice';
    try {
      const part = token.split('.')[1];
      const p = JSON.parse(atob(part.replace(/-/g, '+').replace(/_/g, '/')));
      if (p.tenant_id) tenantId = p.tenant_id;
      if (p.sub) userId = p.sub;
    } catch (_) {}
    const createOpts = {
      method: 'POST',
      headers: {
        Authorization: 'Bearer ' + token,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        tenant_id: tenantId,
        user_id: userId,
        name,
      }),
    };
    const res =
      window.EraOfficeShell && EraOfficeShell.authFetch
        ? await EraOfficeShell.authFetch(apiPath, createOpts)
        : await fetch(apiPath, createOpts);
    if (!res.ok) {
      if (window.EraOfficeShell && EraOfficeShell.handleUnauthorized && EraOfficeShell.handleUnauthorized(res)) {
        return;
      }
      setAuthStatus(label + ' failed: ' + res.status, true);
      return;
    }
    const data = await res.json();
    if (!data.drive_object_id) {
      setAuthStatus(label + ' failed: no drive_object_id', true);
      return;
    }
    location.href = hrefPrefix + data.drive_object_id;
  }

  const newMenuBtn = document.getElementById('newMenuBtn');
  if (newMenuBtn) {
    newMenuBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      toggleNewMenu();
    });
  }
  document.addEventListener('click', () => closeNewMenu());
  const newMenu = document.getElementById('newMenu');
  if (newMenu) {
    newMenu.addEventListener('click', (e) => e.stopPropagation());
  }

  document.getElementById('newDocBtn').addEventListener('click', () => {
    closeNewMenu();
    createOfficeObject(
      '/api/v1/docs',
      'Untitled-' + Date.now() + '.erad',
      '/docs/',
      'New document'
    ).catch(() => {});
  });

  document.getElementById('newSheetBtn').addEventListener('click', () => {
    closeNewMenu();
    createOfficeObject(
      '/api/v1/tables',
      'Untitled-' + Date.now() + '.erat',
      '/tables/',
      'New sheet'
    ).catch(() => {});
  });

  document.getElementById('newDeckBtn').addEventListener('click', () => {
    closeNewMenu();
    createOfficeObject(
      '/api/v1/presentations',
      'Untitled-' + Date.now() + '.erap',
      '/presentations/',
      'New presentation'
    ).catch(() => {});
  });

  document.getElementById('newProjectBtn').addEventListener('click', () => {
    closeNewMenu();
    createOfficeObject(
      '/api/v1/projects',
      'Untitled-' + Date.now() + '.eraj',
      '/projects/',
      'New project'
    ).catch(() => {});
  });

  const newFolderMenuBtn = document.getElementById('newFolderMenuBtn');
  if (newFolderMenuBtn) {
    newFolderMenuBtn.addEventListener('click', () => {
      closeNewMenu();
      createFolderFromUi(true).catch(() => {});
    });
  }

  document.getElementById('createFolderBtn').addEventListener('click', () => {
    createFolderFromUi(false).catch(() => {});
  });
  const treeNewFolderBtn = document.getElementById('treeNewFolderBtn');
  if (treeNewFolderBtn) {
    treeNewFolderBtn.addEventListener('click', () => {
      createFolderFromUi(true).catch(() => {});
    });
  }

  const viewListBtn = document.getElementById('viewListBtn');
  const viewGridBtn = document.getElementById('viewGridBtn');
  if (viewListBtn) {
    viewListBtn.addEventListener('click', () => {
      driveViewMode = 'list';
      applyDriveView();
    });
  }
  if (viewGridBtn) {
    viewGridBtn.addEventListener('click', () => {
      driveViewMode = 'grid';
      applyDriveView();
    });
  }
  const driveSortSelect = document.getElementById('driveSortSelect');
  if (driveSortSelect) {
    driveSortSelect.value = driveSort;
    driveSortSelect.addEventListener('change', () => {
      driveSort = driveSortSelect.value || 'name';
      try { localStorage.setItem('era_drive_sort', driveSort); } catch (_) {}
      renderListing(lastListing.folders, lastListing.objects);
    });
  }
  const selTrashBtn = document.getElementById('selTrashBtn');
  if (selTrashBtn) selTrashBtn.addEventListener('click', () => trashSelected().catch(() => {}));
  const selMoveBtn = document.getElementById('selMoveBtn');
  if (selMoveBtn) selMoveBtn.addEventListener('click', () => moveSelected().catch(() => {}));
  const selClearBtn = document.getElementById('selClearBtn');
  if (selClearBtn) selClearBtn.addEventListener('click', () => clearSelection());
  document.addEventListener('click', (ev) => {
    if (!ev.target.closest('.era-more-wrap')) closeAllMoreMenus();
  });

  const pickFileBtn = document.getElementById('pickFileBtn');
  if (pickFileBtn) {
    pickFileBtn.addEventListener('click', () => {
      document.getElementById('file').click();
    });
  }
  const fileInput = document.getElementById('file');
  if (fileInput) {
    fileInput.addEventListener('change', () => {
      if (pickFileBtn && fileInput.files.length) {
        pickFileBtn.textContent = fileInput.files[0].name;
      }
    });
  }

  document.getElementById('uploadBtn').addEventListener('click', async () => {
    const input = document.getElementById('file');
    if (!input.files.length) {
      input.click();
      return;
    }
    const uploadName = input.files[0].name;
    const folderKey = currentFolderId() || '_root';
    try {
      const listRes = await api('/folders/' + encodeURIComponent(folderKey) + '/children');
      if (listRes.ok) {
        const data = await listRes.json();
        const conflict = (data.objects || []).find((o) => {
          const n = field(o, 'name', 'Name');
          return n === uploadName && isLockedByOther(o);
        });
        if (conflict) {
          setAuthStatus(
            'Upload blocked: "' + uploadName + '" is locked by ' + objectLockedBy(conflict),
            true
          );
          return;
        }
      }
    } catch (_) {
      /* soft check — proceed to server */
    }
    const fd = new FormData();
    fd.append('file', input.files[0]);
    fd.append('name', uploadName);
    const folderId = currentFolderId();
    if (folderId) fd.append('folder_id', folderId);
    const res = await api('/objects', { method: 'POST', body: fd });
    if (!res.ok) {
      setAuthStatus('Upload failed: ' + res.status, true);
      return;
    }
    input.value = '';
    await refreshFiles();
  });

  renderBreadcrumb();
  if (window.EraOfficeShell && EraOfficeShell.requireAuthOrRedirect) {
    if (!EraOfficeShell.requireAuthOrRedirect()) return;
  } else if (!localStorage.getItem('era_token')) {
    location.href = '/login?next=%2Fdrive%2F';
    return;
  }
  if (window.EraOfficeShell) {
    if (EraOfficeShell.markActiveNav) EraOfficeShell.markActiveNav('drive');
    if (EraOfficeShell.mountNav) EraOfficeShell.mountNav(document);
    else if (EraOfficeShell.mountIcons) EraOfficeShell.mountIcons(document);
    if (EraOfficeShell.syncUserChip) EraOfficeShell.syncUserChip();
    if (EraOfficeShell.initChrome) EraOfficeShell.initChrome({ moduleId: 'drive', commentsToggle: null, menubar: false });
  }
  applyDriveView();
  refreshFiles().catch(() => {});
})();
