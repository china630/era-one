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

test('docs editor page loads', async ({ page }) => {
  await page.route('**/api/v1/docs/**', async (route) => {
    if (route.request().url().includes('/sync')) {
      await route.abort();
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
              inlines: [{ text: 'Smoke doc', bold: false, italic: false }],
            },
          ],
        }),
      });
      return;
    }
    await route.fulfill({ status: 204 });
  });
  await page.route('**/api/v1/drive/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ id: 'doc-smoke-1', name: 'Smoke.doc' }),
    });
  });

  await injectToken(page);
  await page.goto('/docs/doc-smoke-1');
  await expect(page.locator('.era-brand-mark')).toHaveText('ERA Office');
  await expect(page.locator('.era-brand-mod')).toHaveText('Documents');
  await expect(page.locator('.doc-block')).toContainText('Smoke doc');
});
