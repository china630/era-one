if (window.EraOfficeShell) {
  if (EraOfficeShell.markActiveNav) EraOfficeShell.markActiveNav('projects');
  if (EraOfficeShell.mountNav) EraOfficeShell.mountNav(document);
  else if (EraOfficeShell.mountIcons) EraOfficeShell.mountIcons(document);
  if (EraOfficeShell.syncUserChip) EraOfficeShell.syncUserChip();
}

const authStatus = document.getElementById('authStatus');
const boardEl = document.getElementById('board');
const boardTitleEl = document.getElementById('boardTitle');
const filterInput = document.getElementById('filterInput');

const COLUMNS = [
  { id: 'backlog', label: 'Backlog' },
  { id: 'todo', label: 'Todo' },
  { id: 'doing', label: 'Doing' },
  { id: 'done', label: 'Done' },
];

/**
 * @typedef {{ id: string, text: string, done: boolean }} ChecklistItem
 * @typedef {{
 *   id: string, title: string, board: string,
 *   drive_object_id?: string, assignee?: string, due_date?: string,
 *   labels: string[], checklist: ChecklistItem[],
 *   priority?: string, sort_key?: number
 * }} Task
 */

/** @type {Task[]} */
let tasks = [];
let boardName = 'Board';
let filterText = '';
/** @type {'board' | 'swimlanes' | 'gantt'} */
let viewMode = 'board';
/** Facet filters */
let filterAssignee = '';
let filterPriority = '';
let filterLabel = '';
let filterOverdue = false;
let pickerSelectedId = '';
let pickerSelectedName = '';
/** Drive picker folder stack: ids, last is current; '' = root */
let pickerFolderStack = [''];
let dragTaskId = null;
let selectedTaskId = '';
/** @type {ChecklistItem[]} */
let editChecklistDraft = [];

function viewModeStorageKey() {
  return 'era_projects_viewMode_' + (projectId || 'legacy');
}

function loadViewModePref() {
  try {
    const v = localStorage.getItem(viewModeStorageKey());
    if (v === 'swimlanes' || v === 'board' || v === 'gantt') viewMode = v;
  } catch (_) {}
}

function saveViewModePref() {
  try {
    localStorage.setItem(viewModeStorageKey(), viewMode);
  } catch (_) {}
}

/** Drive object id when opened as `/projects/{id}` (.eraj); empty = legacy tenant board. */
function pathProjectId() {
  const parts = location.pathname.replace(/\/+$/, '').split('/').filter(Boolean);
  const i = parts.indexOf('projects');
  if (i < 0 || i >= parts.length - 1) return '';
  try {
    return decodeURIComponent(parts[i + 1] || '');
  } catch (_) {
    return parts[i + 1] || '';
  }
}

const projectId = pathProjectId();

function tasksAPI() {
  return projectId
    ? '/api/v1/projects/' + encodeURIComponent(projectId) + '/tasks'
    : '/api/v1/projects/tasks';
}

function taskByIdAPI(id) {
  return projectId
    ? '/api/v1/projects/' + encodeURIComponent(projectId) + '/tasks/' + encodeURIComponent(id)
    : '/api/v1/projects/tasks/' + encodeURIComponent(id);
}

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

function openShareProject() {
  const dlg = document.getElementById('shareDlg');
  const input = document.getElementById('shareLinkInput');
  const driveLink = document.getElementById('shareDriveLink');
  const hint = document.getElementById('shareAclHint');
  const url = location.href;
  if (input) input.value = url;
  if (hint) {
    hint.textContent = projectId
      ? 'Copy link · ACL is managed in Drive on this .eraj object. This dialog does not change permissions.'
      : 'Copy link · Legacy tenant board has no Drive object — create New project (.eraj) for Drive ACL.';
  }
  if (driveLink) {
    if (projectId) {
      driveLink.href = '/drive/?share=' + encodeURIComponent(projectId);
      driveLink.textContent = 'Manage ACL in Drive';
      driveLink.hidden = false;
    } else {
      driveLink.href = '/drive/';
      driveLink.textContent = 'Open Drive';
      driveLink.hidden = false;
    }
  }
  if (dlg && typeof dlg.showModal === 'function') {
    dlg.showModal();
    return;
  }
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(url).then(
      () => setAuthStatus('Share link copied', false),
      () => EraOfficeShell.promptCopy({ title: 'Share project', value: url })
    );
  } else {
    void EraOfficeShell.promptCopy({ title: 'Share project', value: url });
  }
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

function normalizeBoard(b) {
  const id = String(b || 'backlog').toLowerCase();
  if (COLUMNS.some((c) => c.id === id)) return id;
  return 'backlog';
}

function nextBoard(current) {
  const i = COLUMNS.findIndex((c) => c.id === current);
  if (i < 0 || i >= COLUMNS.length - 1) return null;
  return COLUMNS[i + 1].id;
}

function prevBoard(current) {
  const i = COLUMNS.findIndex((c) => c.id === current);
  if (i <= 0) return null;
  return COLUMNS[i - 1].id;
}

function newChecklistId() {
  return 'c' + Date.now().toString(36) + Math.random().toString(36).slice(2, 7);
}

/** @returns {string[]} */
function parseLabelsInput(raw) {
  return String(raw || '')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean);
}

function normalizePriority(p) {
  const v = String(p || '')
    .trim()
    .toLowerCase();
  if (v === 'p0' || v === 'p1' || v === 'p2') return v;
  return '';
}

/** @returns {Task} */
function normalizeTask(raw) {
  const t = raw || {};
  const labels = Array.isArray(t.labels)
    ? t.labels.map((x) => String(x).trim()).filter(Boolean)
    : [];
  const checklist = Array.isArray(t.checklist)
    ? t.checklist.map((item) => ({
        id: String((item && item.id) || newChecklistId()),
        text: String((item && item.text) || ''),
        done: !!(item && item.done),
      }))
    : [];
  const sk = Number(t.sort_key);
  return {
    id: t.id,
    title: t.title || '',
    board: normalizeBoard(t.board),
    drive_object_id: t.drive_object_id || '',
    assignee: t.assignee || '',
    due_date: t.due_date || '',
    labels,
    checklist,
    priority: normalizePriority(t.priority),
    sort_key: isNaN(sk) ? 0 : sk,
  };
}

function checklistProgress(items) {
  const list = Array.isArray(items) ? items : [];
  const done = list.filter((x) => x.done).length;
  return { done, total: list.length };
}

function isOverdue(due) {
  if (!due) return false;
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const d = new Date(due + 'T00:00:00');
  if (isNaN(d.getTime())) return false;
  return d < today;
}

function matchesFilter(t) {
  if (filterAssignee) {
    const a = String(t.assignee || '').trim();
    if (filterAssignee === '__none__') {
      if (a) return false;
    } else if (a.toLowerCase() !== filterAssignee.toLowerCase()) {
      return false;
    }
  }
  if (filterPriority) {
    if (normalizePriority(t.priority) !== filterPriority) return false;
  }
  if (filterLabel) {
    const labels = (t.labels || []).map((x) => String(x).toLowerCase());
    if (!labels.includes(filterLabel.toLowerCase())) return false;
  }
  if (filterOverdue && !isOverdue(t.due_date)) return false;
  const q = filterText.trim().toLowerCase();
  if (!q) return true;
  const labelStr = (t.labels || []).join(' ');
  const hay = [t.title, t.assignee, t.drive_object_id, t.id, labelStr, t.priority]
    .join(' ')
    .toLowerCase();
  return hay.includes(q);
}

function sortTasksInColumn(list) {
  return list.slice().sort((a, b) => {
    const ka = a.sort_key != null ? a.sort_key : 0;
    const kb = b.sort_key != null ? b.sort_key : 0;
    if (ka !== kb) return ka - kb;
    return String(a.id || '').localeCompare(String(b.id || ''));
  });
}

function nextSortKey(board, assignee) {
  const lane = String(assignee || '').trim();
  let max = 0;
  for (const t of tasks) {
    if (normalizeBoard(t.board) !== board) continue;
    if (String(t.assignee || '').trim() !== lane) continue;
    const k = t.sort_key != null ? t.sort_key : 0;
    if (k > max) max = k;
  }
  return max + 1;
}

function knownAssignees() {
  const set = new Set();
  const { userId } = identity();
  if (userId) set.add(userId);
  for (const t of tasks) {
    const a = String(t.assignee || '').trim();
    if (a) set.add(a);
  }
  return Array.from(set).sort((a, b) => a.localeCompare(b));
}

function fillAssigneeDatalist() {
  const dl = document.getElementById('assigneeList');
  if (!dl) return;
  dl.innerHTML = '';
  knownAssignees().forEach((a) => {
    const opt = document.createElement('option');
    opt.value = a;
    dl.appendChild(opt);
  });
}

function refreshFacetOptions() {
  const asg = document.getElementById('filterAssignee');
  if (asg) {
    const cur = asg.value;
    asg.innerHTML = '<option value="">Assignee: all</option><option value="__none__">Unassigned</option>';
    knownAssignees().forEach((a) => {
      const opt = document.createElement('option');
      opt.value = a;
      opt.textContent = a;
      asg.appendChild(opt);
    });
    asg.value = cur;
  }
  const lab = document.getElementById('filterLabel');
  if (lab) {
    const cur = lab.value;
    const labels = new Set();
    tasks.forEach((t) => (t.labels || []).forEach((l) => labels.add(l)));
    lab.innerHTML = '<option value="">Label: all</option>';
    Array.from(labels)
      .sort()
      .forEach((l) => {
        const opt = document.createElement('option');
        opt.value = l;
        opt.textContent = l;
        lab.appendChild(opt);
      });
    lab.value = cur;
  }
  fillAssigneeDatalist();
}

function dueChipClass(due) {
  if (!due) return '';
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const d = new Date(due + 'T00:00:00');
  if (isNaN(d.getTime())) return '';
  const diff = (d - today) / 86400000;
  if (diff < 0) return 'overdue';
  if (diff <= 3) return 'due-soon';
  return '';
}

function updateBoardTitle() {
  if (boardTitleEl) boardTitleEl.textContent = boardName || 'Board';
}

function updateDriveLabel() {
  const id = document.getElementById('driveObjectId').value;
  const el = document.getElementById('driveLinkLabel');
  if (el) el.textContent = id ? 'Linked: ' + id : '';
}

function selectedTask() {
  return tasks.find((x) => x.id === selectedTaskId) || null;
}

function selectTask(id) {
  selectedTaskId = id || '';
  boardEl.querySelectorAll('.card.selected').forEach((el) => el.classList.remove('selected'));
  if (!selectedTaskId) return;
  const card = boardEl.querySelector('.card[data-task-id="' + selectedTaskId + '"]');
  if (card) card.classList.add('selected');
}

/** @param {{ id: string, label: string }} col @param {Task[]} colTasks */
function createColumnEl(col, colTasks) {
  const colEl = document.createElement('div');
  colEl.className = 'column';
  colEl.dataset.board = col.id;
  colEl.innerHTML =
    '<h2>' +
    col.label +
    ' <span class="count">(' +
    colTasks.length +
    ')</span></h2>';

  colEl.addEventListener('dragover', (ev) => {
    ev.preventDefault();
    colEl.classList.add('drag-over');
  });
  colEl.addEventListener('dragleave', () => colEl.classList.remove('drag-over'));
  colEl.addEventListener('drop', (ev) => {
    ev.preventDefault();
    ev.stopPropagation();
    colEl.classList.remove('drag-over');
    const id = ev.dataTransfer.getData('text/plain') || dragTaskId;
    const t = tasks.find((x) => x.id === id);
    if (!t) return;
    const laneEl = colEl.closest('.swimlane');
    let assignee = t.assignee || '';
    if (viewMode === 'swimlanes' && laneEl) {
      const lane = laneEl.dataset.assignee || 'Unassigned';
      assignee = lane === 'Unassigned' ? '' : lane;
    }
    const board = col.id;
    const sameBoard = normalizeBoard(t.board) === board;
    const sameAssignee = String(t.assignee || '').trim() === String(assignee || '').trim();
    const sort_key = nextSortKey(board, assignee);
    if (sameBoard && sameAssignee) {
      if ((t.sort_key != null ? t.sort_key : 0) >= sort_key - 0.001 && sort_key > 1) {
        return;
      }
    }
    moveTaskFull(t, { board, assignee, sort_key }).catch(() => {});
  });

  for (const t of sortTasksInColumn(colTasks)) {
    colEl.appendChild(renderCard(t));
  }
  return colEl;
}

function assigneeLaneKey(t) {
  const a = String((t && t.assignee) || '').trim();
  return a || 'Unassigned';
}

/** @returns {string[]} */
function swimlaneKeys(visible) {
  const keys = new Set();
  for (const t of visible) keys.add(assigneeLaneKey(t));
  const named = Array.from(keys)
    .filter((k) => k !== 'Unassigned')
    .sort((a, b) => a.localeCompare(b));
  if (keys.has('Unassigned') || named.length === 0) named.push('Unassigned');
  return named;
}

function setViewMode(mode) {
  if (mode === 'swimlanes') viewMode = 'swimlanes';
  else if (mode === 'gantt') viewMode = 'gantt';
  else viewMode = 'board';
  saveViewModePref();
  renderBoard();
  setAuthStatus(
    viewMode === 'swimlanes'
      ? 'Swimlanes view'
      : viewMode === 'gantt'
        ? 'Gantt view'
        : 'Board view',
    false
  );
}

function updateBoardStats() {
  const el = document.getElementById('boardStats');
  if (!el) return;
  const n = tasks.length;
  const done = tasks.filter((t) => t.done || normalizeBoard(t.board) === 'done').length;
  el.textContent = n + (n === 1 ? ' task' : ' tasks') + (n ? ' · ' + done + ' done' : '');
}

function renderBoard() {
  boardEl.innerHTML = '';
  updateBoardTitle();
  updateBoardStats();
  fillAssigneeDatalist();
  const visible = tasks.filter((t) => matchesFilter(t));

  if (viewMode === 'gantt') {
    renderGantt(visible);
    if (selectedTaskId && !tasks.some((t) => t.id === selectedTaskId)) {
      selectedTaskId = '';
    }
    return;
  }

  if (viewMode === 'swimlanes') {
    boardEl.className = 'lanes';
    boardEl.setAttribute('aria-label', 'Project swimlanes');
    for (const lane of swimlaneKeys(visible)) {
      const laneTasks = visible.filter((t) => assigneeLaneKey(t) === lane);
      const section = document.createElement('section');
      section.className = 'swimlane';
      section.dataset.assignee = lane;
      const title = document.createElement('h3');
      title.className = 'swimlane-title';
      title.textContent = lane + ' (' + laneTasks.length + ')';
      section.appendChild(title);
      const cols = document.createElement('div');
      cols.className = 'board';
      for (const col of COLUMNS) {
        const colTasks = laneTasks.filter((t) => normalizeBoard(t.board) === col.id);
        cols.appendChild(createColumnEl(col, colTasks));
      }
      section.addEventListener('dragover', (ev) => {
        ev.preventDefault();
        section.classList.add('drag-over');
      });
      section.addEventListener('dragleave', () => section.classList.remove('drag-over'));
      section.addEventListener('drop', (ev) => {
        // Lane-level drop (reassign). Column handlers stopPropagation when they handle.
        ev.preventDefault();
        section.classList.remove('drag-over');
        const id = ev.dataTransfer.getData('text/plain') || dragTaskId;
        const t = tasks.find((x) => x.id === id);
        if (!t) return;
        const lane = section.dataset.assignee || 'Unassigned';
        const assignee = lane === 'Unassigned' ? '' : lane;
        if (String(t.assignee || '').trim() === String(assignee || '').trim()) return;
        const sort_key = nextSortKey(normalizeBoard(t.board), assignee);
        moveTaskFull(t, { board: normalizeBoard(t.board), assignee, sort_key }).catch(() => {});
      });
      section.appendChild(cols);
      boardEl.appendChild(section);
    }
  } else {
    boardEl.className = 'board';
    boardEl.setAttribute('aria-label', 'Project board');
    for (const col of COLUMNS) {
      const colTasks = visible.filter((t) => normalizeBoard(t.board) === col.id);
      boardEl.appendChild(createColumnEl(col, colTasks));
    }
  }

  if (selectedTaskId && !tasks.some((t) => t.id === selectedTaskId)) {
    selectedTaskId = '';
  }
}

function renderCard(t) {
  const card = document.createElement('div');
  card.className = 'card' + (t.id === selectedTaskId ? ' selected' : '');
  card.dataset.taskId = t.id;
  card.draggable = true;
  card.addEventListener('click', (ev) => {
    if (ev.target.closest('button, a, input, label')) return;
    selectTask(t.id);
  });
  card.addEventListener('dragstart', (ev) => {
    dragTaskId = t.id;
    selectTask(t.id);
    card.classList.add('dragging');
    ev.dataTransfer.setData('text/plain', t.id);
    ev.dataTransfer.effectAllowed = 'move';
  });
  card.addEventListener('dragend', () => {
    card.classList.remove('dragging');
    dragTaskId = null;
  });

  const title = document.createElement('div');
  title.className = 'title';
  title.textContent = t.title || '(untitled)';
  card.appendChild(title);

  const chips = document.createElement('div');
  chips.className = 'chips';
  if (t.priority) {
    const c = document.createElement('span');
    c.className = 'chip priority priority-' + t.priority;
    c.textContent = t.priority.toUpperCase();
    c.title = 'Priority ' + t.priority.toUpperCase();
    chips.appendChild(c);
  }
  if (t.assignee) {
    const c = document.createElement('span');
    c.className = 'chip';
    c.textContent = '@' + t.assignee;
    chips.appendChild(c);
  }
  if (t.due_date) {
    const c = document.createElement('span');
    c.className = 'chip ' + dueChipClass(t.due_date);
    c.textContent = t.due_date;
    chips.appendChild(c);
  }
  (t.labels || []).forEach((label) => {
    const c = document.createElement('span');
    c.className = 'chip label';
    c.textContent = label;
    chips.appendChild(c);
  });
  const prog = checklistProgress(t.checklist);
  if (prog.total > 0) {
    const c = document.createElement('span');
    c.className = 'chip checklist-prog';
    c.textContent = prog.done + '/' + prog.total;
    c.title = 'Checklist progress';
    chips.appendChild(c);
  }
  if (chips.childNodes.length) card.appendChild(chips);

  if ((t.checklist || []).length) {
    const ul = document.createElement('ul');
    ul.className = 'checklist';
    t.checklist.forEach((item) => {
      const li = document.createElement('li');
      if (item.done) li.classList.add('done');
      const cb = document.createElement('input');
      cb.type = 'checkbox';
      cb.checked = !!item.done;
      cb.title = 'Toggle checklist item';
      cb.addEventListener('click', (ev) => ev.stopPropagation());
      cb.addEventListener('change', () => {
        toggleChecklistItem(t.id, item.id, cb.checked).catch(() => {});
      });
      const span = document.createElement('span');
      span.textContent = item.text || '(item)';
      li.appendChild(cb);
      li.appendChild(span);
      ul.appendChild(li);
    });
    card.appendChild(ul);
  }

  if (t.drive_object_id) {
    const link = document.createElement('a');
    link.href = '/docs/' + encodeURIComponent(t.drive_object_id);
    link.textContent = 'Open in Docs';
    link.className = 'drive-link';
    card.appendChild(link);
  }

  const actions = document.createElement('div');
  actions.className = 'actions';

  const board = normalizeBoard(t.board);
  const back = prevBoard(board);
  const fwd = nextBoard(board);

  if (back) {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'era-btn';
    btn.textContent = '← ' + back;
    btn.addEventListener('click', () => {
      selectTask(t.id);
      moveTask(t, back);
    });
    actions.appendChild(btn);
  }
  if (fwd) {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'era-btn';
    btn.textContent = fwd + ' →';
    btn.addEventListener('click', () => {
      selectTask(t.id);
      moveTask(t, fwd);
    });
    actions.appendChild(btn);
  }

  const edit = document.createElement('button');
  edit.type = 'button';
  edit.className = 'era-btn';
  edit.textContent = 'Edit';
  edit.addEventListener('click', () => {
    selectTask(t.id);
    openEditTask(t);
  });
  actions.appendChild(edit);

  const del = document.createElement('button');
  del.type = 'button';
  del.className = 'era-btn';
  del.textContent = 'Delete';
  del.addEventListener('click', () => deleteTask(t.id));
  actions.appendChild(del);

  card.appendChild(actions);
  return card;
}

async function api(path, opts) {
  const next = Object.assign({}, opts || {}, {
    headers: authHeaders((opts && opts.headers) || {}),
  });
  const res =
    window.EraOfficeShell && EraOfficeShell.authFetch
      ? await EraOfficeShell.authFetch(path, next)
      : await fetch(path, next);
  if (window.EraOfficeShell && EraOfficeShell.handleUnauthorized) {
    if (EraOfficeShell.handleUnauthorized(res)) return res;
  }
  return res;
}

function commentsStorageKey() {
  return 'era_projects_comments_' + (projectId || 'legacy');
}

function loadBoardComments() {
  try {
    const raw = localStorage.getItem(commentsStorageKey());
    const arr = raw ? JSON.parse(raw) : [];
    return Array.isArray(arr) ? arr : [];
  } catch (_) {
    return [];
  }
}

function saveBoardComments(list) {
  try {
    localStorage.setItem(commentsStorageKey(), JSON.stringify(list || []));
  } catch (_) {}
}

function renderBoardComments() {
  const ul = document.getElementById('commentsList');
  if (!ul) return;
  ul.innerHTML = '';
  const list = loadBoardComments();
  if (!list.length) {
    ul.innerHTML = '<li class="era-hint">No comments yet</li>';
    return;
  }
  list.forEach((c) => {
    const li = document.createElement('li');
    li.style.borderBottom = '1px solid var(--era-line)';
    li.style.padding = '0.35rem 0';
    li.innerHTML =
      '<div><strong>' +
      String(c.author || 'you').replace(/</g, '&lt;') +
      '</strong></div><div>' +
      String(c.text || '').replace(/</g, '&lt;') +
      '</div>';
    ul.appendChild(li);
  });
}

function addBoardComment(text) {
  const t = String(text || '').trim();
  if (!t) return;
  const { userId } = identity();
  const list = loadBoardComments();
  list.unshift({ id: crypto.randomUUID(), text: t, author: userId, at: Date.now() });
  saveBoardComments(list);
  renderBoardComments();
}

function parseDueMs(due) {
  if (!due) return null;
  const d = new Date(due + 'T00:00:00');
  return Number.isNaN(d.getTime()) ? null : d.getTime();
}

function renderGantt(visible) {
  boardEl.className = 'era-gantt';
  boardEl.setAttribute('aria-label', 'Project Gantt');
  boardEl.innerHTML = '';
  const withDates = visible.filter((t) => parseDueMs(t.due_date) != null);
  if (!withDates.length) {
    const empty = document.createElement('div');
    empty.className = 'era-gantt-empty';
    empty.textContent =
      'No due dates yet — set Due on tasks to see bars. Timeline uses due date as bar end (3-day default span).';
    boardEl.appendChild(empty);
    return;
  }
  const dayMs = 86400000;
  let min = Infinity;
  let max = -Infinity;
  withDates.forEach((t) => {
    const end = parseDueMs(t.due_date);
    const start = end - 2 * dayMs;
    if (start < min) min = start;
    if (end > max) max = end;
  });
  min -= dayMs;
  max += 2 * dayMs;
  const days = Math.max(7, Math.round((max - min) / dayMs) + 1);
  const labels = document.createElement('div');
  labels.className = 'era-gantt-labels';
  const headSpacer = document.createElement('div');
  headSpacer.className = 'era-gantt-label';
  headSpacer.style.fontWeight = '650';
  headSpacer.style.color = 'var(--era-muted)';
  headSpacer.textContent = 'Task';
  labels.appendChild(headSpacer);

  const timeline = document.createElement('div');
  const head = document.createElement('div');
  head.className = 'era-gantt-head';
  for (let i = 0; i < days; i++) {
    const d = new Date(min + i * dayMs);
    const cell = document.createElement('div');
    cell.className = 'era-gantt-day';
    cell.textContent = d.getDate();
    cell.title = d.toISOString().slice(0, 10);
    head.appendChild(cell);
  }
  const rows = document.createElement('div');
  rows.className = 'era-gantt-rows';
  rows.appendChild(head);

  withDates
    .slice()
    .sort((a, b) => parseDueMs(a.due_date) - parseDueMs(b.due_date))
    .forEach((t) => {
      const end = parseDueMs(t.due_date);
      const start = end - 2 * dayMs;
      const label = document.createElement('div');
      label.className = 'era-gantt-label' + (t.id === selectedTaskId ? ' selected' : '');
      label.textContent = t.title || '(untitled)';
      label.title = (t.title || '') + (t.due_date ? ' · due ' + t.due_date : '');
      label.addEventListener('click', () => selectTask(t.id));
      labels.appendChild(label);

      const row = document.createElement('div');
      row.className = 'era-gantt-row';
      row.style.width = days * 28 + 'px';
      const bar = document.createElement('div');
      const left = Math.max(0, Math.round((start - min) / dayMs));
      const width = Math.max(1, Math.round((end - start) / dayMs) + 1);
      bar.className =
        'era-gantt-bar' +
        (normalizeBoard(t.board) === 'done' || t.done ? ' done' : '') +
        (isOverdue(t.due_date) && normalizeBoard(t.board) !== 'done' ? ' overdue' : '');
      bar.style.left = left * 28 + 'px';
      bar.style.width = width * 28 - 4 + 'px';
      bar.textContent = t.due_date || '';
      bar.title = (t.title || '') + ' · due ' + (t.due_date || '');
      bar.addEventListener('click', () => selectTask(t.id));
      row.appendChild(bar);
      rows.appendChild(row);
    });

  timeline.appendChild(rows);
  boardEl.appendChild(labels);
  boardEl.appendChild(timeline);
}

async function loadBoardMeta() {
  if (projectId) {
    const res = await api('/api/v1/projects/' + encodeURIComponent(projectId));
    if (!res.ok) return;
    const data = await res.json();
    boardName = (data && data.name) || 'Board.eraj';
    updateBoardTitle();
    return;
  }
  const res = await api('/api/v1/projects/board');
  if (!res.ok) return;
  const data = await res.json();
  boardName = (data && data.name) || 'Board';
  updateBoardTitle();
}

async function renameBoard() {
  const next = await EraOfficeShell.promptText({
    title: 'Rename board',
    label: 'Board name',
    value: boardName || 'Board',
  });
  if (next == null || !next.trim()) return;
  const path = projectId
    ? '/api/v1/projects/' + encodeURIComponent(projectId)
    : '/api/v1/projects/board';
  const res = await api(path, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: next.trim() }),
  });
  if (!res.ok) {
    setAuthStatus('Rename failed: ' + res.status, true);
    return;
  }
  const data = await res.json();
  boardName = data.name || next.trim();
  updateBoardTitle();
  setAuthStatus('Board renamed', false);
}

async function createErajProject() {
  if (!localStorage.getItem('era_token')) {
    setAuthStatus('Sign in via Drive first (era_token).', true);
    return;
  }
  const res = await api('/api/v1/projects', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: 'Untitled-' + Date.now() + '.eraj' }),
  });
  if (!res.ok) {
    setAuthStatus('New project failed: ' + res.status, true);
    return;
  }
  const data = await res.json();
  if (!data.drive_object_id) {
    setAuthStatus('New project failed: no drive_object_id', true);
    return;
  }
  location.href = '/projects/' + encodeURIComponent(data.drive_object_id);
}

async function loadTasks() {
  if (!localStorage.getItem('era_token')) {
    setAuthStatus('Sign in via Drive first (era_token).', true);
    tasks = [];
    renderBoard();
    return;
  }
  await loadBoardMeta().catch(() => {});
  const res = await api(tasksAPI());
  if (!res.ok) {
    setAuthStatus('Load failed: ' + res.status, true);
    return;
  }
  const data = await res.json();
  tasks = (Array.isArray(data) ? data : []).map(normalizeTask);
  refreshFacetOptions();
  const mode = projectId ? '.eraj ' : '';
  setAuthStatus(mode + 'Board loaded — ' + tasks.length + ' task(s)', false);
  renderBoard();
}

function taskPayload(t, overrides) {
  const o = normalizeTask(Object.assign({}, t, overrides || {}));
  return {
    id: o.id,
    title: o.title,
    board: o.board,
    drive_object_id: o.drive_object_id || '',
    assignee: o.assignee || '',
    due_date: o.due_date || '',
    labels: o.labels || [],
    checklist: o.checklist || [],
    priority: o.priority || '',
    sort_key: o.sort_key != null ? o.sort_key : 0,
  };
}

async function addTask() {
  if (!localStorage.getItem('era_token')) {
    setAuthStatus('Sign in via Drive first (era_token).', true);
    return;
  }
  const title = document.getElementById('taskTitle').value.trim();
  if (!title) {
    setAuthStatus('Title required', true);
    return;
  }
  const assignee = document.getElementById('taskAssignee').value.trim();
  const priorityEl = document.getElementById('taskPriority');
  const priority = normalizePriority(priorityEl ? priorityEl.value : '');
  const body = {
    title,
    board: 'backlog',
    assignee,
    due_date: document.getElementById('taskDue').value || '',
    labels: [],
    checklist: [],
    priority,
    sort_key: nextSortKey('backlog', assignee),
  };
  const driveObjectId = document.getElementById('driveObjectId').value.trim();
  if (driveObjectId) body.drive_object_id = driveObjectId;

  const res = await api(tasksAPI(), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    setAuthStatus('Create failed: ' + res.status, true);
    return;
  }
  document.getElementById('taskTitle').value = '';
  document.getElementById('taskAssignee').value = '';
  document.getElementById('taskDue').value = '';
  if (priorityEl) priorityEl.value = '';
  document.getElementById('driveObjectId').value = '';
  updateDriveLabel();
  setAuthStatus('Task created', false);
  await loadTasks();
}

async function saveTask(t) {
  const res = await api(tasksAPI(), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(taskPayload(t)),
  });
  if (!res.ok) {
    setAuthStatus('Save failed: ' + res.status, true);
    return false;
  }
  return true;
}

async function moveTask(t, board) {
  const sort_key = nextSortKey(board, t.assignee || '');
  await moveTaskFull(t, { board, sort_key });
}

async function moveTaskFull(t, patch) {
  const next = Object.assign({}, t, patch || {});
  const ok = await saveTask(next);
  if (!ok) return;
  const bits = [];
  if (patch.board) bits.push(patch.board);
  if (patch.assignee !== undefined) {
    bits.push(patch.assignee ? '@' + patch.assignee : 'Unassigned');
  }
  setAuthStatus('Moved' + (bits.length ? ' → ' + bits.join(' · ') : ''), false);
  await loadTasks();
}

async function deleteTask(id) {
  const res = await api(taskByIdAPI(id), {
    method: 'DELETE',
  });
  if (!res.ok && res.status !== 204) {
    setAuthStatus('Delete failed: ' + res.status, true);
    return;
  }
  if (selectedTaskId === id) selectedTaskId = '';
  setAuthStatus('Task deleted', false);
  await loadTasks();
}

async function toggleChecklistItem(taskId, itemId, done) {
  const t = tasks.find((x) => x.id === taskId);
  if (!t) return;
  const next = (t.checklist || []).map((item) =>
    item.id === itemId ? Object.assign({}, item, { done: !!done }) : item
  );
  const ok = await saveTask(Object.assign({}, t, { checklist: next }));
  if (!ok) return;
  await loadTasks();
}

function renderEditChecklistList() {
  const ul = document.getElementById('editChecklistList');
  if (!ul) return;
  ul.innerHTML = '';
  editChecklistDraft.forEach((item, idx) => {
    const li = document.createElement('li');
    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.checked = !!item.done;
    cb.addEventListener('change', () => {
      editChecklistDraft[idx].done = cb.checked;
    });
    const text = document.createElement('input');
    text.type = 'text';
    text.className = 'era-input';
    text.value = item.text || '';
    text.addEventListener('input', () => {
      editChecklistDraft[idx].text = text.value;
    });
    const rm = document.createElement('button');
    rm.type = 'button';
    rm.className = 'era-btn';
    rm.textContent = '×';
    rm.title = 'Remove item';
    rm.addEventListener('click', () => {
      editChecklistDraft.splice(idx, 1);
      renderEditChecklistList();
    });
    li.appendChild(cb);
    li.appendChild(text);
    li.appendChild(rm);
    ul.appendChild(li);
  });
}

function addEditChecklistItem(text) {
  const trimmed = String(text || '').trim();
  if (!trimmed) return;
  editChecklistDraft.push({ id: newChecklistId(), text: trimmed, done: false });
  renderEditChecklistList();
}

function openEditTask(t) {
  fillAssigneeDatalist();
  document.getElementById('editTaskId').value = t.id;
  document.getElementById('editTitle').value = t.title || '';
  document.getElementById('editAssignee').value = t.assignee || '';
  document.getElementById('editDue').value = t.due_date || '';
  const pri = document.getElementById('editPriority');
  if (pri) pri.value = t.priority || '';
  const labelsEl = document.getElementById('editLabels');
  if (labelsEl) labelsEl.value = (t.labels || []).join(', ');
  editChecklistDraft = (t.checklist || []).map((item) => ({
    id: item.id || newChecklistId(),
    text: item.text || '',
    done: !!item.done,
  }));
  renderEditChecklistList();
  const addInput = document.getElementById('editChecklistAdd');
  if (addInput) addInput.value = '';
  const dlg = document.getElementById('taskEditDlg');
  if (dlg && dlg.showModal) dlg.showModal();
}

async function commitEditTask() {
  const id = document.getElementById('editTaskId').value;
  const t = tasks.find((x) => x.id === id);
  if (!t) return;
  t.title = document.getElementById('editTitle').value.trim() || t.title;
  t.assignee = document.getElementById('editAssignee').value.trim();
  t.due_date = document.getElementById('editDue').value || '';
  const pri = document.getElementById('editPriority');
  t.priority = normalizePriority(pri ? pri.value : '');
  const labelsEl = document.getElementById('editLabels');
  t.labels = parseLabelsInput(labelsEl ? labelsEl.value : '');
  t.checklist = editChecklistDraft
    .map((item) => ({
      id: item.id || newChecklistId(),
      text: String(item.text || '').trim(),
      done: !!item.done,
    }))
    .filter((item) => item.text);
  const ok = await saveTask(t);
  if (ok) {
    setAuthStatus('Task updated', false);
    await loadTasks();
  }
}

function requireSelectedTask() {
  const t = selectedTask();
  if (t) return t;
  setAuthStatus('Select a task first (click a card)', true);
  return null;
}

function editLabelsMenu() {
  const t = requireSelectedTask();
  if (!t) return;
  const dlg = document.getElementById('taskEditDlg');
  if (dlg && typeof dlg.showModal === 'function') {
    openEditTask(t);
    const labelsEl = document.getElementById('editLabels');
    if (labelsEl && typeof labelsEl.focus === 'function') {
      try {
        labelsEl.focus();
      } catch (_) {}
    }
    return;
  }
  void (async () => {
    const current = (t.labels || []).join(', ');
    const next = await EraOfficeShell.promptText({
      title: 'Edit labels',
      label: 'Labels (comma-separated)',
      value: current,
    });
    if (next == null) return;
    const labels = parseLabelsInput(next);
    const ok = await saveTask(Object.assign({}, t, { labels }));
    if (!ok) return;
    setAuthStatus('Labels updated', false);
    await loadTasks();
  })();
}

function editChecklistMenu() {
  const t = requireSelectedTask();
  if (!t) return;
  const dlg = document.getElementById('taskEditDlg');
  if (dlg && typeof dlg.showModal === 'function') {
    openEditTask(t);
    const addInput = document.getElementById('editChecklistAdd');
    if (addInput && typeof addInput.focus === 'function') {
      try {
        addInput.focus();
      } catch (_) {}
    }
    return;
  }
  void (async () => {
    const text = await EraOfficeShell.promptText({
      title: 'Add checklist item',
      label: 'Item text',
      value: '',
    });
    if (text == null || !String(text).trim()) return;
    const checklist = (t.checklist || []).concat([
      { id: newChecklistId(), text: String(text).trim(), done: false },
    ]);
    const ok = await saveTask(Object.assign({}, t, { checklist }));
    if (!ok) return;
    setAuthStatus('Checklist item added', false);
    await loadTasks();
  })();
}

function pickerFolderId() {
  return pickerFolderStack[pickerFolderStack.length - 1] || '';
}

async function loadDrivePickerList() {
  const list = document.getElementById('drivePickerList');
  const crumb = document.getElementById('drivePickerCrumb');
  if (!list) return;
  list.innerHTML = '<li class="era-hint">Loading…</li>';
  pickerSelectedId = '';
  pickerSelectedName = '';
  const folderId = pickerFolderId();
  if (crumb) {
    crumb.textContent = folderId
      ? 'Folder: ' + folderId + (pickerFolderStack.length > 1 ? ' · Up available' : '')
      : 'Root folder';
  }
  try {
    const path = folderId
      ? '/api/v1/drive/folders/' + encodeURIComponent(folderId) + '/children'
      : '/api/v1/drive/folders/_root/children';
    const res = await api(path);
    if (!res.ok) {
      list.innerHTML = '<li class="era-hint">Drive unavailable (' + res.status + ')</li>';
      return;
    }
    const data = await res.json();
    const folders = data.folders || [];
    const objects = data.objects || [];
    list.innerHTML = '';
    if (pickerFolderStack.length > 1) {
      const up = document.createElement('li');
      up.className = 'folder-up';
      up.textContent = '↑ Parent folder';
      up.addEventListener('click', () => {
        pickerFolderStack.pop();
        loadDrivePickerList().catch(() => {});
      });
      list.appendChild(up);
    }
    folders.forEach((f) => {
      const li = document.createElement('li');
      const id = f.id || f.folder_id || '';
      const name = f.name || id;
      li.className = 'folder-item';
      li.textContent = '📁 ' + name;
      li.addEventListener('click', () => {
        if (!id) return;
        pickerFolderStack.push(id);
        loadDrivePickerList().catch(() => {});
      });
      list.appendChild(li);
    });
    objects.forEach((obj) => {
      const li = document.createElement('li');
      const id = obj.id || obj.object_id || '';
      const name = obj.name || id;
      li.textContent = name + (obj.content_type ? ' · ' + obj.content_type : '');
      li.dataset.id = id;
      li.addEventListener('click', () => {
        list.querySelectorAll('li').forEach((n) => n.classList.remove('selected'));
        li.classList.add('selected');
        pickerSelectedId = id;
        pickerSelectedName = name;
      });
      list.appendChild(li);
    });
    if (!folders.length && !objects.length) {
      const empty = document.createElement('li');
      empty.className = 'era-hint';
      empty.textContent = 'Empty folder. Upload in Drive or go up.';
      list.appendChild(empty);
    }
  } catch (_) {
    list.innerHTML = '<li class="era-hint">Drive request failed</li>';
  }
}

async function openDrivePicker() {
  const dlg = document.getElementById('drivePickerDlg');
  pickerFolderStack = [''];
  if (dlg && dlg.showModal) dlg.showModal();
  await loadDrivePickerList();
}

document.getElementById('drivePickerDlg').addEventListener('close', () => {
  const v = document.getElementById('drivePickerDlg').returnValue;
  if (v === 'ok' && pickerSelectedId) {
    document.getElementById('driveObjectId').value = pickerSelectedId;
    updateDriveLabel();
    setAuthStatus('Linked ' + (pickerSelectedName || pickerSelectedId), false);
  } else if (v === 'clear') {
    document.getElementById('driveObjectId').value = '';
    updateDriveLabel();
  }
});

document.getElementById('taskEditDlg').addEventListener('close', () => {
  if (document.getElementById('taskEditDlg').returnValue === 'ok') {
    commitEditTask().catch(() => {});
  }
});

const editChecklistAddBtn = document.getElementById('editChecklistAddBtn');
if (editChecklistAddBtn) {
  editChecklistAddBtn.addEventListener('click', (ev) => {
    ev.preventDefault();
    const input = document.getElementById('editChecklistAdd');
    addEditChecklistItem(input ? input.value : '');
    if (input) input.value = '';
    if (input) input.focus();
  });
}
const editChecklistAdd = document.getElementById('editChecklistAdd');
if (editChecklistAdd) {
  editChecklistAdd.addEventListener('keydown', (ev) => {
    if (ev.key === 'Enter') {
      ev.preventDefault();
      addEditChecklistItem(editChecklistAdd.value);
      editChecklistAdd.value = '';
    }
  });
}

document.getElementById('addBtn').onclick = () => {
  addTask().catch(() => {});
};
document.getElementById('refreshBtn').onclick = () => {
  loadTasks().catch(() => {});
};
document.getElementById('pickDriveBtn').onclick = () => {
  openDrivePicker().catch(() => {});
};
document.getElementById('taskTitle').addEventListener('keydown', (ev) => {
  if (ev.key === 'Enter') addTask().catch(() => {});
});
if (filterInput) {
  filterInput.addEventListener('input', () => {
    filterText = filterInput.value || '';
    renderBoard();
  });
}

const shareBtn = document.getElementById('shareBtn');
if (shareBtn) shareBtn.onclick = () => openShareProject();
const shareCopyBtn = document.getElementById('shareCopyBtn');
if (shareCopyBtn) {
  shareCopyBtn.onclick = () => {
    const input = document.getElementById('shareLinkInput');
    const url = (input && input.value) || location.href;
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(url).then(
        () => setAuthStatus('Share link copied', false),
        () => EraOfficeShell.promptCopy({ title: 'Share project', value: url })
      );
    } else {
      void EraOfficeShell.promptCopy({ title: 'Share project', value: url });
    }
  };
}

function wireFacetFilters() {
  const asg = document.getElementById('filterAssignee');
  const pri = document.getElementById('filterPriority');
  const lab = document.getElementById('filterLabel');
  const od = document.getElementById('filterOverdue');
  if (asg) {
    asg.addEventListener('change', () => {
      filterAssignee = asg.value || '';
      renderBoard();
    });
  }
  if (pri) {
    pri.addEventListener('change', () => {
      filterPriority = normalizePriority(pri.value);
      renderBoard();
    });
  }
  if (lab) {
    lab.addEventListener('change', () => {
      filterLabel = lab.value || '';
      renderBoard();
    });
  }
  if (od) {
    od.addEventListener('change', () => {
      filterOverdue = !!od.checked;
      renderBoard();
    });
  }
}
wireFacetFilters();
loadViewModePref();

if (window.EraOfficeMenubar) {
  EraOfficeMenubar.init('#menubar', {
    'file.new': () => createErajProject().catch(() => {}),
    'file.refresh': () => loadTasks().catch(() => {}),
    'file.rename': () => renameBoard().catch(() => {}),
    'file.share': () => openShareProject(),
    'file.openDrive': () => {
      location.href = '/drive/';
    },
    'edit.add': () => {
      document.getElementById('taskTitle').focus();
    },
    'edit.filter': () => filterInput && filterInput.focus(),
    'edit.labels': () => editLabelsMenu(),
    'edit.checklist': () => editChecklistMenu(),
    'view.board': () => {
      setViewMode('board');
      if (boardEl) boardEl.scrollIntoView();
    },
    'view.swimlanes': () => {
      setViewMode('swimlanes');
      if (boardEl) boardEl.scrollIntoView();
    },
    'view.gantt': () => {
      setViewMode('gantt');
      if (boardEl) boardEl.scrollIntoView();
    },
    'help.about': () => {
      void EraOfficeShell.confirmAction({
        title: 'About ERA Projects',
        message:
          'ERA Projects — kanban board in your contour (not Jira / MS Project).\n\n' +
          '• Board, swimlanes by assignee, priority chips\n' +
          '• Timeline (View → Gantt) bars by due date (3-day default span) — not full Gantt\n' +
          '• Board comments are local-only in this browser (not peer-synced)',
        okLabel: 'OK',
        cancelLabel: 'Close',
      });
    },
  });
}

if (window.EraOfficeShell) {
  if (EraOfficeShell.wireSessionWatch) EraOfficeShell.wireSessionWatch();
  if (EraOfficeShell.wireCommentsToggle) EraOfficeShell.wireCommentsToggle(false);
}
renderBoardComments();
const prjCommentForm = document.getElementById('prjCommentForm');
if (prjCommentForm) {
  prjCommentForm.addEventListener('submit', (ev) => {
    ev.preventDefault();
    const input = document.getElementById('prjCommentInput');
    addBoardComment(input && input.value);
    if (input) input.value = '';
    if (window.EraOfficeShell && EraOfficeShell.setCommentsOpen) {
      EraOfficeShell.setCommentsOpen(true);
    }
  });
}

if (window.EraOfficeShell && EraOfficeShell.requireAuthOrRedirect) {
  if (!EraOfficeShell.requireAuthOrRedirect()) {
    /* redirecting to login */
  } else {
    loadTasks().catch(() => {});
  }
} else if (localStorage.getItem('era_token')) {
  loadTasks().catch(() => {});
} else {
  location.href = '/login?next=' + encodeURIComponent(location.pathname);
}
