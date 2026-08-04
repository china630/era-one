import { test, expect } from '@playwright/test';

test('two contexts see merged doc text after reload', async ({ browser }) => {
  const sharedDoc = {
    blocks: [
      {
        id: 'b1',
        block_type: 'paragraph',
        heading_level: 0,
        inlines: [{ text: 'Hi', bold: false, italic: false }],
      },
    ],
  };

  const handler = async (route: import('@playwright/test').Route) => {
    const url = route.request().url();
    if (url.includes('/sync')) {
      await route.fulfill({ status: 101, body: '' }).catch(() => route.continue());
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(sharedDoc),
      });
      return;
    }
    await route.continue();
  };

  const ctxA = await browser.newContext();
  const ctxB = await browser.newContext();
  const pageA = await ctxA.newPage();
  const pageB = await ctxB.newPage();
  await pageA.route('**/api/v1/docs/**', handler);
  await pageB.route('**/api/v1/docs/**', handler);

  await pageA.goto('/docs/coedit-1');
  await pageB.goto('/docs/coedit-1');
  await expect(pageA.locator('.doc-block')).toContainText('Hi');
  sharedDoc.blocks[0].inlines[0].text = 'Hi all';
  await pageB.reload();
  await expect(pageB.locator('.doc-block')).toContainText('Hi all');
  await ctxA.close();
  await ctxB.close();
});
