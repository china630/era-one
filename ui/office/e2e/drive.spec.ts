import { test, expect } from '@playwright/test';
import { createHmac } from 'crypto';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';

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

test('drive upload appears in file list', async ({ page }) => {
  let uploaded = false;
  await page.route('**/api/v1/drive/folders/_root/children', async (route) => {
    const objects = uploaded
      ? [{ id: 'obj-e2e-1', name: 'e2e-upload.txt', size_bytes: 11, version: 1 }]
      : [];
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ folders: [], objects }),
    });
  });

  await page.route('**/api/v1/drive/objects', async (route) => {
    if (route.request().method() === 'POST') {
      uploaded = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: 'obj-e2e-1', name: 'e2e-upload.txt', size_bytes: 11 }),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/drive/');
  await expect(page.locator('.era-brand-mod')).toHaveText('Drive');

  const tmp = path.join(os.tmpdir(), `era-e2e-${Date.now()}.txt`);
  fs.writeFileSync(tmp, 'hello e2e');
  await page.setInputFiles('#file', tmp);
  await page.getByRole('button', { name: 'Upload' }).click();
  await expect(page.getByText('e2e-upload.txt')).toBeVisible({ timeout: 10_000 });
});

test('drive create folder navigate upload breadcrumb', async ({ page }) => {
  /** @type {Record<string, { folders: any[]; objects: any[] }>} */
  const tree: Record<string, { folders: any[]; objects: any[] }> = {
    _root: { folders: [], objects: [] },
  };
  let folderSeq = 0;
  let objSeq = 0;
  let lastListKey = '_root';

  await page.route('**/api/v1/drive/folders/**/children', async (route) => {
    const url = route.request().url();
    const m = url.match(/\/folders\/([^/]+)\/children/);
    const key = m ? decodeURIComponent(m[1]) : '_root';
    lastListKey = key;
    const node = tree[key] || { folders: [], objects: [] };
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(node),
    });
  });

  await page.route('**/api/v1/drive/folders', async (route) => {
    if (route.request().method() !== 'POST') {
      await route.continue();
      return;
    }
    const body = route.request().postDataJSON() as { name: string; parent_id?: string };
    folderSeq += 1;
    const id = `fld-${folderSeq}`;
    const parentKey = body.parent_id || '_root';
    if (!tree[parentKey]) tree[parentKey] = { folders: [], objects: [] };
    const folder = { id, name: body.name, parent_id: body.parent_id || '' };
    tree[parentKey].folders.push(folder);
    tree[id] = { folders: [], objects: [] };
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(folder),
    });
  });

  await page.route('**/api/v1/drive/objects', async (route) => {
    if (route.request().method() !== 'POST') {
      await route.continue();
      return;
    }
    objSeq += 1;
    const id = `obj-${objSeq}`;
    const key = lastListKey || '_root';
    if (!tree[key]) tree[key] = { folders: [], objects: [] };
    const obj = { id, name: 'nested-e2e.txt', size_bytes: 12, version: 1, folder_id: key === '_root' ? '' : key };
    tree[key].objects.push(obj);
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(obj),
    });
  });

  await page.route('**/api/v1/drive/objects/*/versions', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        versions: [{ version: 1, size_bytes: 12, created_at: '2026-07-30T00:00:00Z' }],
      }),
    });
  });

  await injectToken(page);
  await page.goto('/drive/');
  await expect(page.locator('.era-brand-mod')).toHaveText('Drive');
  await expect(page.locator('#breadcrumb .here')).toHaveText('Root');

  await page.fill('#newFolderName', 'Reports');
  await page.getByRole('button', { name: 'Create folder' }).click();
  await expect(page.getByRole('button', { name: 'Reports' })).toBeVisible({ timeout: 10_000 });

  await page.getByRole('button', { name: 'Reports' }).click();
  await expect(page.locator('#breadcrumb .here')).toHaveText('Reports');
  await expect(page.locator('#breadcrumb')).toContainText('Root');

  const tmp = path.join(os.tmpdir(), `era-nested-${Date.now()}.txt`);
  fs.writeFileSync(tmp, 'nested file');
  await page.setInputFiles('#file', tmp);
  await page.getByRole('button', { name: 'Upload' }).click();
  await expect(page.getByText('nested-e2e.txt')).toBeVisible({ timeout: 10_000 });

  await page.getByRole('button', { name: 'Versions' }).click();
  await expect(page.locator('#versionsPanel')).toHaveClass(/open/);
  await expect(page.locator('#versionsList')).toContainText('v1');

  await page.locator('#breadcrumb button.link').filter({ hasText: 'Root' }).click();
  await expect(page.locator('#breadcrumb .here')).toHaveText('Root');
  await expect(page.getByRole('button', { name: 'Reports' })).toBeVisible();
});

test('drive search lists mocked results and clear returns to browse', async ({ page }) => {
  await page.route('**/api/v1/drive/folders/_root/children', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        folders: [{ id: 'fld-root', name: 'Inbox' }],
        objects: [{ id: 'obj-root', name: 'readme.txt', size_bytes: 6, version: 1 }],
      }),
    });
  });

  await page.route('**/api/v1/drive/search**', async (route) => {
    const url = new URL(route.request().url());
    expect(url.searchParams.get('q')).toBe('found');
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        folders: [{ id: 'fld-s1', name: 'FoundFolder' }],
        objects: [{ id: 'obj-s1', name: 'found-doc.txt', size_bytes: 3, version: 1 }],
      }),
    });
  });

  await injectToken(page);
  await page.goto('/drive/');
  await expect(page.getByRole('button', { name: 'Inbox' })).toBeVisible();
  await expect(page.getByText('readme.txt')).toBeVisible();

  await page.fill('#driveSearchInput', 'found');
  await page.locator('#driveSearchBtn').click();
  await expect(page.locator('#authStatus')).toContainText('Search: 2 results');
  await expect(page.getByRole('button', { name: 'FoundFolder' })).toBeVisible();
  await expect(page.getByText('found-doc.txt')).toBeVisible();
  await expect(page.locator('#driveSearchClear')).toBeVisible();

  await page.locator('#driveSearchClear').click();
  await expect(page.locator('#authStatus')).toContainText('folder(s)');
  await expect(page.getByRole('button', { name: 'Inbox' })).toBeVisible();
  await expect(page.getByText('readme.txt')).toBeVisible();
  await expect(page.locator('#driveSearchClear')).toHaveJSProperty('hidden', true);
});

test('drive preview shows text content', async ({ page }) => {
  await page.route('**/api/v1/drive/folders/_root/children', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        folders: [],
        objects: [{ id: 'obj-prev-1', name: 'note.txt', size_bytes: 5, version: 1 }],
      }),
    });
  });
  await page.route('**/api/v1/drive/objects/obj-prev-1', async (route) => {
    if (route.request().method() === 'GET' && !route.request().url().includes('/meta')) {
      await route.fulfill({
        status: 200,
        contentType: 'text/plain',
        body: 'hello',
        headers: { 'Content-Type': 'text/plain' },
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/drive/');
  await expect(page.getByText('note.txt')).toBeVisible();
  await page.getByRole('button', { name: 'Preview' }).click();
  await expect(page.locator('#driveMain')).toHaveClass(/preview-open/);
  await expect(page.locator('#previewPanel')).toBeVisible();
  await expect(page.locator('#previewBody')).toContainText('hello');
});

test('drive preview image by extension and unsupported type', async ({ page }) => {
  await page.route('**/api/v1/drive/folders/_root/children', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        folders: [],
        objects: [
          { id: 'obj-img-1', name: 'photo.png', size_bytes: 68, version: 1 },
          { id: 'obj-bin-1', name: 'blob.bin', size_bytes: 4, version: 1 },
        ],
      }),
    });
  });
  // 1x1 PNG
  const pngB64 =
    'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==';
  await page.route('**/api/v1/drive/objects/obj-img-1', async (route) => {
    if (route.request().method() === 'GET' && !route.request().url().includes('/meta')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/octet-stream',
        body: Buffer.from(pngB64, 'base64'),
      });
      return;
    }
    await route.continue();
  });
  await page.route('**/api/v1/drive/objects/obj-bin-1', async (route) => {
    if (route.request().method() === 'GET' && !route.request().url().includes('/meta')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/octet-stream',
        body: 'xxxx',
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/drive/');
  await page.locator('li.file', { hasText: 'photo.png' }).getByRole('button', { name: 'Preview' }).click();
  await expect(page.locator('#previewBody img')).toBeVisible({ timeout: 10_000 });

  await page.locator('li.file', { hasText: 'blob.bin' }).getByRole('button', { name: 'Preview' }).click();
  await expect(page.locator('#previewBody')).toContainText(/No inline preview/i);
});

test('drive lock toggle shows badge', async ({ page }) => {
  let objects = [
    { id: 'obj-lock-1', name: 'contract.txt', size_bytes: 8, version: 1, locked_by: '' },
  ];

  await page.route('**/api/v1/drive/folders/_root/children', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ folders: [], objects }),
    });
  });
  await page.route('**/api/v1/drive/objects/obj-lock-1', async (route) => {
    if (route.request().method() === 'PATCH') {
      const body = route.request().postDataJSON() as { locked?: boolean };
      if (body.locked === true) {
        objects = [{ ...objects[0], locked_by: 'staging-user' }];
      } else if (body.locked === false) {
        objects = [{ ...objects[0], locked_by: '' }];
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'obj-lock-1',
          name: objects[0].name,
          folder_id: '',
          locked_by: objects[0].locked_by || null,
        }),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/drive/');
  await expect(page.getByText('contract.txt')).toBeVisible();
  await expect(page.locator('.lock-badge')).toHaveCount(0);

  await page.getByRole('button', { name: 'Lock' }).click();
  await expect(page.locator('.lock-badge')).toBeVisible({ timeout: 10_000 });
  await expect(page.locator('.lock-badge')).toContainText('Locked');
  await expect(page.locator('#authStatus')).toContainText('Locked contract.txt');

  await page.getByRole('button', { name: 'Unlock' }).click();
  await expect(page.locator('.lock-badge')).toHaveCount(0, { timeout: 10_000 });
});

test('drive lock by other hides unlock and warns open', async ({ page }) => {
  const objects = [
    {
      id: 'obj-lock-other',
      name: 'sealed.txt',
      size_bytes: 4,
      version: 1,
      locked_by: 'alice',
      owner_user_id: 'alice',
      content_type: 'text/plain',
    },
  ];
  await page.route('**/api/v1/drive/folders/_root/children', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ folders: [], objects }),
    });
  });

  await injectToken(page);
  await page.goto('/drive/');
  await expect(page.locator('.lock-badge')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Unlock', exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: /Unlock disabled/ })).toBeVisible();

  page.once('dialog', async (d) => {
    expect(d.message()).toMatch(/locked by alice/i);
    await d.dismiss();
  });
  // Open with Documents via select (no native openKind for .txt)
  await page.locator('select.open-with-select').selectOption('docs');
});

test('drive rename and share ACL', async ({ page }) => {
  let objects = [{ id: 'obj-share-1', name: 'memo.txt', size_bytes: 4, version: 1 }];
  let acl = [
    { principal: 'user:staging-user', role: 1 },
  ];

  await page.route('**/api/v1/drive/folders/_root/children', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ folders: [], objects }),
    });
  });
  await page.route('**/api/v1/drive/objects/obj-share-1', async (route) => {
    if (route.request().method() === 'PATCH') {
      const body = route.request().postDataJSON() as { name?: string };
      if (body.name) objects = [{ ...objects[0], name: body.name }];
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: 'obj-share-1', name: objects[0].name, folder_id: '' }),
      });
      return;
    }
    await route.continue();
  });
  await page.route('**/api/v1/drive/objects/obj-share-1/meta', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'obj-share-1',
        name: objects[0].name,
        folder_id: '',
        acl,
      }),
    });
  });
  await page.route('**/api/v1/drive/objects/obj-share-1/acl', async (route) => {
    if (route.request().method() === 'PATCH') {
      const body = route.request().postDataJSON() as { entries: typeof acl };
      acl = body.entries;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'ok' }),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/drive/');
  await expect(page.getByText('memo.txt')).toBeVisible();

  page.once('dialog', (d) => d.accept('memo-renamed.txt'));
  await page.getByRole('button', { name: 'Rename' }).click();
  await expect(page.getByText('memo-renamed.txt')).toBeVisible({ timeout: 10_000 });

  await page.getByRole('button', { name: 'Share' }).click();
  await expect(page.locator('#sharePanel')).toHaveClass(/open/);
  await page.fill('#aclPrincipal', 'user:bob');
  await page.selectOption('#aclRole', '2');
  await page.getByRole('button', { name: 'Add' }).click();
  await page.getByRole('button', { name: 'Save ACL' }).click();
  await expect(page.locator('#authStatus')).toContainText('ACL saved');
});
