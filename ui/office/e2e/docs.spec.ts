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

const sampleDoc = {
  blocks: [
    {
      id: 'b1',
      block_type: 'paragraph',
      heading_level: 0,
      inlines: [{ text: 'Hello docs', bold: false, italic: false }],
    },
  ],
};

test('docs new document opens editor', async ({ page }) => {
  await page.route('**/api/v1/docs', async (route) => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ drive_object_id: 'doc-new-1' }),
      });
      return;
    }
    await route.continue();
  });
  await page.route('**/api/v1/docs/doc-new-1**', async (route) => {
    if (route.request().url().includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(sampleDoc),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/docs');
  await expect(page.locator('.era-brand-mark')).toHaveText('ERA Office');
  await expect(page.locator('.era-brand-mod')).toHaveText('Documents');
  await page.getByRole('button', { name: 'New' }).click();
  await expect(page).toHaveURL(/\/docs\/doc-new-1/);
  await expect(page.locator('.doc-block')).toContainText('Hello docs');
});

test('docs toolbar heading list bold', async ({ page }) => {
  await page.route('**/api/v1/docs/doc-toolbar-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync')) {
      await route.continue();
      return;
    }
    if (url.includes('/snapshot') && route.request().method() === 'POST') {
      await route.fulfill({ status: 200, body: '' });
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(sampleDoc),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/docs/doc-toolbar-1');
  const block = page.locator('.doc-block').first();
  await expect(block).toContainText('Hello docs');
  await block.click();

  await page.getByRole('button', { name: 'H1' }).click();
  await expect(block).toHaveAttribute('data-type', 'heading');
  await expect(block).toHaveAttribute('data-level', '1');

  await page.getByRole('button', { name: 'H2' }).click();
  await expect(block).toHaveAttribute('data-level', '2');

  await page.getByRole('button', { name: 'Bulleted list' }).click();
  await expect(block).toHaveAttribute('data-type', 'list_item');

  await page.getByRole('button', { name: 'Bold' }).click();
  await expect(block.locator('b')).toContainText('Hello docs');

  await page.locator('#underlineBtn').click();
  await expect(block.locator('u')).toContainText('Hello docs');

  page.once('dialog', (d) => d.accept('https://era.local/docs'));
  await page.locator('#linkBtn').click();
  await expect(block.locator('a')).toHaveAttribute('href', 'https://era.local/docs');

  page.once('dialog', (d) => d.accept('TE comment'));
  await page.locator('#commentBtn').click();
  await expect(page.locator('#commentsList')).toContainText('TE comment');
});

test('docs find and word count', async ({ page }) => {
  await page.route('**/api/v1/docs/doc-find-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(sampleDoc),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/docs/doc-find-1');
  await expect(page.locator('#wordCount')).toHaveText('2 words');

  await page.locator('#findInput').fill('hello');
  await page.getByRole('button', { name: 'Find next' }).click();
  await expect(page.locator('.doc-block.find-highlight')).toContainText('Hello docs');
});

test('docs import docx redirects to editor', async ({ page }) => {
  await page.route('**/api/v1/docs/import', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ drive_object_id: 'doc-import-1' }),
    });
  });
  await page.route('**/api/v1/docs/doc-import-1**', async (route) => {
    if (route.request().url().includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          blocks: [
            {
              id: 'b1',
              block_type: 'paragraph',
              heading_level: 0,
              inlines: [{ text: 'Imported', bold: false, italic: false }],
            },
          ],
          legacy_features_dropped: true,
        }),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/docs');
  await expect(page.getByRole('button', { name: 'Import docx' })).toBeVisible();
  const tmp = path.join(os.tmpdir(), `era-docs-e2e-${Date.now()}.docx`);
  // Minimal bytes are enough for UI to POST base64; import API is mocked.
  fs.writeFileSync(tmp, Buffer.from('PK\x03\x04fake-docx'));
  await page.setInputFiles('#file', tmp);
  await expect(page).toHaveURL(/\/docs\/doc-import-1/, { timeout: 10_000 });
  await expect(page.locator('.doc-block')).toContainText('Imported');
  await expect(page.locator('#banner')).toBeVisible();
});

test('drive new document opens docs editor', async ({ page }) => {
  await page.route('**/api/v1/drive/folders/**/children', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ folders: [], objects: [] }),
    });
  });
  await page.route('**/api/v1/docs', async (route) => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ drive_object_id: 'doc-from-drive' }),
      });
      return;
    }
    await route.continue();
  });
  await page.route('**/api/v1/docs/doc-from-drive**', async (route) => {
    if (route.request().url().includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(sampleDoc),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/drive/');
  await page.getByRole('button', { name: 'New document' }).click();
  await expect(page).toHaveURL(/\/docs\/doc-from-drive/);
  await expect(page.locator('.era-brand-mod')).toHaveText('Documents');
  await expect(page.locator('.doc-block')).toContainText('Hello docs');
});
