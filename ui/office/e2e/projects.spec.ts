import { test, expect } from '@playwright/test';
import { createHmac } from 'crypto';

function signTestJwt(secret: string, payload: Record<string, unknown>): string {
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
  const body = Buffer.from(JSON.stringify(payload)).toString('base64url');
  const data = `${header}.${body}`;
  const sig = createHmac('sha256', secret).update(data).digest('base64url');
  return `${data}.${sig}`;
}

async function injectToken(page: import('@playwright/test').Page) {
  const secret = 'dev-only-change-in-prod';
  const token = signTestJwt(secret, {
    sub: 'staging-user',
    tenant_id: 't-demo',
    exp: Math.floor(Date.now() / 1000) + 3600,
  });
  await page.addInitScript(
    ({ tokenKey, token }) => {
      localStorage.setItem(tokenKey, token);
    },
    { tokenKey: 'era_token', token }
  );
}

function mockProjectsApi(page: import('@playwright/test').Page, storeRef: { store: any[]; boardName: string }) {
  return Promise.all([
    page.route('**/api/v1/projects/board', async (route) => {
      const method = route.request().method();
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ name: storeRef.boardName }),
        });
        return;
      }
      if (method === 'PUT' || method === 'POST') {
        const body = route.request().postDataJSON() as any;
        storeRef.boardName = body.name || 'Board';
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ name: storeRef.boardName }),
        });
        return;
      }
      await route.continue();
    }),
    page.route('**/api/v1/projects/tasks', async (route) => {
      const method = route.request().method();
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(storeRef.store),
        });
        return;
      }
      if (method === 'POST') {
        const body = route.request().postDataJSON() as any;
        const task = {
          id: body.id || 'task-e2e-1',
          title: body.title,
          board: body.board || 'backlog',
          drive_object_id: body.drive_object_id || '',
          assignee: body.assignee || '',
          due_date: body.due_date || '',
          labels: Array.isArray(body.labels) ? body.labels : [],
          checklist: Array.isArray(body.checklist) ? body.checklist : [],
          priority: body.priority || '',
          sort_key: typeof body.sort_key === 'number' ? body.sort_key : 0,
          tenant_id: 't-demo',
        };
        const idx = storeRef.store.findIndex((t) => t.id === task.id);
        if (idx >= 0) storeRef.store[idx] = task;
        else storeRef.store.push(task);
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(task),
        });
        return;
      }
      await route.continue();
    }),
  ]);
}

test('projects board loads and creates task', async ({ page }) => {
  const storeRef = { store: [] as any[], boardName: 'Board' };
  await mockProjectsApi(page, storeRef);

  await injectToken(page);
  await page.goto('/projects');
  await expect(page.locator('.era-brand-mod')).toHaveText('Projects');
  await expect(page.locator('#menubar')).toBeVisible();
  await expect(page.locator('.column[data-board="backlog"]')).toBeVisible();

  await page.fill('#taskTitle', 'Write brief');
  await page.fill('#taskAssignee', 'alice');
  await page.locator('#driveObjectId').evaluate((el: HTMLInputElement) => {
    el.value = 'drv-obj-1';
  });
  await page.locator('#addBtn').click();

  await expect(page.locator('.card .title')).toContainText('Write brief');
  await expect(page.locator('.card .chip')).toContainText('@alice');
  await expect(page.locator('.card a.drive-link')).toHaveAttribute('href', '/docs/drv-obj-1');
  await expect(page.locator('.column[data-board="backlog"] .card')).toHaveCount(1);
});

test('projects move task across columns', async ({ page }) => {
  const storeRef = {
    store: [
      {
        id: 'task-move-1',
        title: 'Move me',
        board: 'todo',
        drive_object_id: '',
        tenant_id: 't-demo',
      },
    ],
    boardName: 'Board',
  };
  await mockProjectsApi(page, storeRef);

  await injectToken(page);
  await page.goto('/projects/');
  await expect(page.locator('.column[data-board="todo"] .card .title')).toContainText('Move me');
  await page.getByRole('button', { name: 'doing →' }).click();
  await expect(page.locator('.column[data-board="doing"] .card .title')).toContainText('Move me');
  await expect(page.locator('.column[data-board="todo"] .card')).toHaveCount(0);
});

test('projects delete task', async ({ page }) => {
  const storeRef = {
    store: [
      {
        id: 'task-del-1',
        title: 'Remove me',
        board: 'done',
        tenant_id: 't-demo',
      },
    ],
    boardName: 'Board',
  };
  await mockProjectsApi(page, storeRef);
  await page.route('**/api/v1/projects/tasks/**', async (route) => {
    if (route.request().method() === 'DELETE') {
      storeRef.store = [];
      await route.fulfill({ status: 204, body: '' });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/projects/');
  await expect(page.locator('.card .title')).toContainText('Remove me');
  await page.getByRole('button', { name: 'Delete' }).click();
  await expect(page.locator('.card')).toHaveCount(0);
});

test('projects drag card and filter and rename', async ({ page }) => {
  const storeRef = {
    store: [
      {
        id: 'task-a',
        title: 'Alpha brief',
        board: 'backlog',
        assignee: 'bob',
        tenant_id: 't-demo',
      },
      {
        id: 'task-b',
        title: 'Other',
        board: 'todo',
        assignee: 'carol',
        tenant_id: 't-demo',
      },
    ],
    boardName: 'Board',
  };
  await mockProjectsApi(page, storeRef);

  await injectToken(page);
  await page.goto('/projects/');
  await expect(page.locator('#boardTitle')).toHaveText('Board');

  page.once('dialog', (d) => d.accept('Sprint Wave E'));
  await page.getByRole('button', { name: 'File', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="file.rename"]').click();
  await expect(page.locator('#boardTitle')).toHaveText('Sprint Wave E');

  await page.fill('#filterInput', 'alpha');
  await expect(page.locator('.card .title')).toHaveCount(1);
  await expect(page.locator('.card .title')).toContainText('Alpha brief');

  await page.fill('#filterInput', '');
  const card = page.locator('.column[data-board="backlog"] .card').first();
  const doing = page.locator('.column[data-board="doing"]');
  await card.dragTo(doing);
  await expect(page.locator('.column[data-board="doing"] .card .title')).toContainText('Alpha brief');
});

test('projects shows label chips and checklist progress', async ({ page }) => {
  const storeRef = {
    store: [
      {
        id: 'task-labels-1',
        title: 'Labeled task',
        board: 'backlog',
        labels: ['design', 'p0'],
        checklist: [
          { id: 'c1', text: 'Draft', done: true },
          { id: 'c2', text: 'Review', done: false },
          { id: 'c3', text: 'Ship', done: false },
        ],
        tenant_id: 't-demo',
      },
    ],
    boardName: 'Board',
  };
  await mockProjectsApi(page, storeRef);

  await injectToken(page);
  await page.goto('/projects/');
  await expect(page.locator('.card .chip.label')).toHaveText(['design', 'p0']);
  await expect(page.locator('.card .chip.checklist-prog')).toHaveText('1/3');
  await expect(page.locator('.card .checklist li')).toHaveCount(3);

  await page.locator('.card .checklist li').nth(1).locator('input[type="checkbox"]').check();
  await expect(page.locator('.card .chip.checklist-prog')).toHaveText('2/3');
});

test('projects swimlanes groups by assignee', async ({ page }) => {
  const storeRef = {
    store: [
      {
        id: 'task-unassigned',
        title: 'No owner',
        board: 'backlog',
        tenant_id: 't-demo',
      },
      {
        id: 'task-alice',
        title: 'Alice owns',
        board: 'todo',
        assignee: 'alice',
        tenant_id: 't-demo',
      },
    ],
    boardName: 'Board',
  };
  await mockProjectsApi(page, storeRef);

  await injectToken(page);
  await page.goto('/projects/');
  await expect(page.locator('#board.board')).toBeVisible();

  await page.getByRole('button', { name: 'View', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="view.swimlanes"]').click();

  await expect(page.locator('#board.lanes')).toBeVisible();
  await expect(page.locator('.swimlane[data-assignee="alice"] .swimlane-title')).toContainText(
    'alice'
  );
  await expect(
    page.locator('.swimlane[data-assignee="Unassigned"] .swimlane-title')
  ).toContainText('Unassigned');
  await expect(
    page.locator('.swimlane[data-assignee="alice"] .column[data-board="todo"] .card .title')
  ).toContainText('Alice owns');
  await expect(
    page.locator('.swimlane[data-assignee="Unassigned"] .column[data-board="backlog"] .card .title')
  ).toContainText('No owner');

  await page.getByRole('button', { name: 'View', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="view.board"]').click();
  await expect(page.locator('#board.board')).toBeVisible();
  await expect(page.locator('.swimlane')).toHaveCount(0);
});

test('projects drive picker lists objects', async ({ page }) => {
  const storeRef = { store: [] as any[], boardName: 'Board' };
  await mockProjectsApi(page, storeRef);
  await page.route('**/api/v1/drive/folders/**/children', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        folders: [{ id: 'fold-1', name: 'Docs' }],
        objects: [{ id: 'doc-1', name: 'Memo.erad', content_type: 'application/erad' }],
      }),
    });
  });

  await injectToken(page);
  await page.goto('/projects/');
  await page.locator('#pickDriveBtn').click();
  await expect(page.locator('#drivePickerDlg')).toBeVisible();
  await expect(page.locator('#drivePickerList li.folder-item')).toContainText('Docs');
  await page.locator('#drivePickerList li').filter({ hasText: 'Memo.erad' }).click();
  await page.locator('#drivePickerOk').click();
  await expect(page.locator('#driveLinkLabel')).toContainText('doc-1');
});

test('projects PRJ-LITE share priority swimlanes reassign', async ({ page }) => {
  const storeRef = {
    store: [
      {
        id: 'task-a',
        title: 'Lane move',
        board: 'todo',
        assignee: 'alice',
        priority: 'p1',
        sort_key: 1,
        tenant_id: 't-demo',
      },
      {
        id: 'task-b',
        title: 'Unowned',
        board: 'backlog',
        assignee: '',
        priority: 'p0',
        sort_key: 1,
        tenant_id: 't-demo',
      },
    ],
    boardName: 'Board',
  };
  await mockProjectsApi(page, storeRef);

  await page.addInitScript(() => {
    localStorage.setItem('era_projects_viewMode_legacy', 'swimlanes');
  });
  await injectToken(page);
  await page.goto('/projects/');

  await page.getByRole('button', { name: 'Share', exact: true }).click();
  await expect(page.locator('#shareDlg')).toBeVisible();
  await expect(page.locator('#shareLinkInput')).toHaveValue(/\/projects/);
  await page.locator('#shareDlg button[value="ok"]').click();

  await expect(page.locator('.card .chip.priority-p0')).toContainText('P0');
  await expect(page.locator('.card .chip.priority-p1')).toContainText('P1');

  await expect(page.locator('#board.lanes')).toBeVisible();
  // Explicit swimlanes toggle (pref may race with first paint).
  await page.getByRole('button', { name: 'View', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="view.swimlanes"]').click();
  await expect(page.locator('#board.lanes')).toBeVisible();

  const card = page.locator('.card[data-task-id="task-a"]');
  const unassignedLane = page.locator('.swimlane[data-assignee="Unassigned"]');
  await card.dragTo(unassignedLane);
  await expect
    .poll(() => storeRef.store.find((t) => t.id === 'task-a')?.assignee ?? 'missing')
    .toBe('');
  await expect(
    page.locator('.swimlane[data-assignee="Unassigned"] .card[data-task-id="task-a"] .title')
  ).toContainText('Lane move');

  await page.locator('#filterPriority').selectOption('p0');
  await expect(page.locator('.card .title')).toHaveCount(1);
  await expect(page.locator('.card .title')).toContainText('Unowned');
});
