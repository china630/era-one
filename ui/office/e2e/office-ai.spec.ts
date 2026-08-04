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

test('office ai summarize stub mode', async ({ page }) => {
  await page.route('**/api/v1/docs-ai/summarize', async (route) => {
    const body = route.request().postDataJSON() as { text?: string };
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        mode: 'stub',
        summary: 'ERA Office AI (air-gap stub): ' + (body?.text || ''),
      }),
    });
  });

  await injectToken(page);
  await page.goto('/office-ai/');
  await expect(page.locator('.era-brand-mod')).toHaveText('Office AI');
  await expect(page.getByText('Air-gap assist only')).toBeVisible();

  await page.fill('#sourceText', 'agenda notes for the board');
  await page.getByRole('button', { name: 'Summarize' }).click();

  await expect(page.locator('#result')).toContainText('agenda notes for the board');
  await expect(page.locator('#modeBadge')).toContainText('mode=stub');
  await expect(page.getByText('Summary ready (stub)')).toBeVisible();
});

test('office ai requires source text', async ({ page }) => {
  await injectToken(page);
  await page.goto('/office-ai/');
  await page.getByRole('button', { name: 'Summarize' }).click();
  await expect(page.getByText('Paste source text first')).toBeVisible();
});

test('office ai rewrite stub mode', async ({ page }) => {
  await page.route('**/api/v1/docs-ai/rewrite', async (route) => {
    const body = route.request().postDataJSON() as { text?: string; instruction?: string };
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        mode: 'stub',
        rewrite: 'ERA Office AI (air-gap stub rewrite): ' + (body?.text || ''),
      }),
    });
  });

  await injectToken(page);
  await page.goto('/office-ai/?mode=rewrite');
  await expect(page.getByRole('button', { name: 'Rewrite' })).toBeVisible();
  await page.fill('#sourceText', 'informal draft text');
  await page.getByRole('button', { name: 'Rewrite' }).click();
  await expect(page.locator('#result')).toContainText('informal draft text');
  await expect(page.getByText('Rewrite ready (stub)')).toBeVisible();
});

test('docs summarize with ai opens office ai', async ({ page }) => {
  await page.route('**/api/v1/docs/doc-ai-1**', async (route) => {
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
              inlines: [{ text: 'Quarterly report draft', bold: false, italic: false }],
            },
          ],
        }),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/docs/doc-ai-1');
  await expect(page.locator('.doc-block')).toContainText('Quarterly report draft');
  await page.getByRole('button', { name: 'Summarize with AI' }).click();
  await expect(page).toHaveURL(/\/office-ai\/?/);
  await expect(page.locator('#sourceText')).toHaveValue(/Quarterly report draft/);
});
