import { test, expect } from '@playwright/test';

test('docs editor page loads', async ({ page }) => {
  await page.route('**/api/v1/docs/*', async (route) => {
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
              inlines: [{ text: 'Smoke doc', bold: false, italic: false }],
            },
          ],
        }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/docs/doc-smoke-1');
  await expect(page.locator('.era-brand-mark')).toHaveText('ERA Office');
  await expect(page.locator('.era-brand-mod')).toHaveText('Documents');
  await expect(page.locator('.doc-block')).toContainText('Smoke doc');
});
