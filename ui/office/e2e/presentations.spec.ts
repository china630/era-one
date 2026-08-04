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

function deckPayload(slides: { id: string; title: string; body: string }[]) {
  return {
    name: 'Untitled.erap',
    format: 'erap',
    slides,
  };
}

test('presentations new deck opens editor', async ({ page }) => {
  await page.route('**/api/v1/presentations', async (route) => {
    if (route.request().method() === 'POST' && !route.request().url().includes('/import')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ drive_object_id: 'deck-new-1' }),
      });
      return;
    }
    await route.continue();
  });
  await page.route('**/api/v1/presentations/deck-new-1**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          deckPayload([{ id: 's1', title: 'Welcome', body: 'First slide' }])
        ),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/presentations');
  await expect(page.locator('.era-brand-mod')).toHaveText('Presentations');
  await page.getByRole('button', { name: 'New deck' }).click();
  await expect(page).toHaveURL(/\/presentations\/deck-new-1/);
  await expect(page.locator('#slideTitle')).toContainText('Welcome');
  await expect(page.locator('#slideBody')).toContainText('First slide');
});

test('presentations add slide and navigate', async ({ page }) => {
  await page.route('**/api/v1/presentations/deck-nav-1**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          deckPayload([{ id: 's1', title: 'One', body: 'A' }])
        ),
      });
      return;
    }
    if (route.request().method() === 'PUT') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: route.request().postData() || '{}',
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/presentations/deck-nav-1');
  await expect(page.locator('#slideTitle')).toContainText('One');
  await page.getByRole('button', { name: 'Add slide' }).click();
  await expect(page.locator('#slidePos')).toContainText('Slide 2 / 2');
  await expect(page.locator('#slideTitle')).toContainText('New slide');
  await page.getByRole('button', { name: 'Previous' }).click();
  await expect(page.locator('#slidePos')).toContainText('Slide 1 / 2');
  await expect(page.locator('#filmstrip li')).toHaveCount(2);
});

test('presentations move slide down reorders filmstrip', async ({ page }) => {
  await page.route('**/api/v1/presentations/deck-reorder-1**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          deckPayload([
            { id: 's1', title: 'First', body: 'A', layout: 'title_body' },
            { id: 's2', title: 'Second', body: 'B', layout: 'title_body' },
          ])
        ),
      });
      return;
    }
    if (route.request().method() === 'PUT') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: route.request().postData() || '{}',
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/presentations/deck-reorder-1');
  await expect(page.locator('#filmstrip li').first()).toContainText('First');
  await page.getByRole('button', { name: 'Move down' }).click();
  // Order swaps; selection follows the moved slide (still "First").
  await expect(page.locator('#filmstrip li').first()).toContainText('Second');
  await expect(page.locator('#filmstrip li').nth(1)).toContainText('First');
  await expect(page.locator('#filmstrip li').nth(1)).toHaveClass(/active/);
  await expect(page.locator('#slideTitle')).toContainText('First');
});

test('presentations import pptx redirects to editor', async ({ page }) => {
  await page.route('**/api/v1/presentations/import', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ drive_object_id: 'deck-import-1' }),
    });
  });
  await page.route('**/api/v1/presentations/deck-import-1**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          deckPayload([{ id: 's1', title: 'Imported', body: 'From pptx' }])
        ),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/presentations');
  await expect(page.getByRole('button', { name: 'Import pptx' })).toBeVisible();
  const tmp = path.join(os.tmpdir(), `era-pres-e2e-${Date.now()}.pptx`);
  fs.writeFileSync(tmp, Buffer.from('PK\x03\x04fake-pptx'));
  await page.setInputFiles('#file', tmp);
  await expect(page).toHaveURL(/\/presentations\/deck-import-1/, { timeout: 10_000 });
  await expect(page.locator('#slideTitle')).toContainText('Imported');
});

test('drive new presentation opens editor', async ({ page }) => {
  await page.route('**/api/v1/drive/folders/**/children', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ folders: [], objects: [] }),
    });
  });
  await page.route('**/api/v1/presentations', async (route) => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ drive_object_id: 'deck-from-drive' }),
      });
      return;
    }
    await route.continue();
  });
  await page.route('**/api/v1/presentations/deck-from-drive**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          deckPayload([{ id: 's1', title: 'Deck', body: '' }])
        ),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/drive/');
  await page.getByRole('button', { name: 'New presentation' }).click();
  await expect(page).toHaveURL(/\/presentations\/deck-from-drive/);
  await expect(page.locator('.era-brand-mod')).toHaveText('Presentations');
  await expect(page.locator('#slideTitle')).toContainText('Deck');
});
