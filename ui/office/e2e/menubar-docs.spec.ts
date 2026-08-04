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

const sampleDoc = {
  blocks: [
    {
      id: 'b1',
      block_type: 'paragraph',
      heading_level: 0,
      inlines: [{ text: 'Hello docs', bold: false, italic: false }],
    },
  ],
  page: { size: 'a4', orientation: 'portrait', margins_mm: 20 },
  header: { text: '', page_numbers: false },
  footer: { text: '', page_numbers: false },
};

test('docs menubar File Download and Format Bold', async ({ page }) => {
  await page.route('**/api/v1/docs/doc-menu-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync')) {
      await route.continue();
      return;
    }
    if (url.includes('/export/docx') && route.request().method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        body: Buffer.from('PK\x03\x04fake'),
      });
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
  await page.goto('/docs/doc-menu-1');
  await expect(page.locator('#menubar')).toBeVisible();
  await expect(page.getByRole('button', { name: 'File', exact: true })).toBeVisible();

  await page.getByRole('button', { name: 'File', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Download' }).hover();
  await page.locator('.era-menu-item[data-cmd="file.export"]').click();
  await expect(page.locator('#authStatus')).toContainText(/Export/i);

  const block = page.locator('.doc-block').first();
  await block.click();
  await page.getByRole('button', { name: 'Format', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Text' }).hover();
  await page.locator('.era-menu-item[data-cmd="format.bold"]').click();
  await expect(block.locator('b')).toContainText('Hello docs');

  await page.getByRole('button', { name: 'File', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Download' }).hover();
  await expect(page.locator('.era-menu-item[data-cmd="file.pdf"]')).toBeEnabled();
  await expect(page.locator('.era-menu-item[data-cmd="file.odt"]')).toBeDisabled();
});

test('docs W2 replace color insert image', async ({ page }) => {
  await page.route('**/api/v1/docs/doc-w2-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync') || url.includes('/snapshot')) {
      if (url.includes('/snapshot')) {
        await route.fulfill({ status: 200, body: '' });
        return;
      }
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
  await page.goto('/docs/doc-w2-1');
  await page.locator('.doc-block').first().click();

  page.once('dialog', (d) => d.accept('#1565c0'));
  await page.getByRole('button', { name: 'Format', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Text' }).hover();
  await page.locator('.era-menu-item[data-cmd="format.color"]').click();
  await expect(page.locator('.doc-block').first().locator('span[style*="color"]')).toBeVisible();

  await page.getByRole('button', { name: 'Edit', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="edit.replace"]').click();
  await expect(page.locator('#replaceDlg')).toBeVisible();
  await page.locator('#replaceFind').fill('Hello');
  await page.locator('#replaceWith').fill('Hi');
  await page.locator('#replaceDlg button[value="all"]').click();
  await expect(page.locator('.doc-block').first()).toContainText('Hi docs');

  page.once('dialog', (d) => d.accept('https://example.com/a.png'));
  await page.getByRole('button', { name: 'Insert', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="insert.image"]').click();
  await expect(page.locator('.doc-block[data-type="image"] img')).toHaveAttribute(
    'src',
    'https://example.com/a.png'
  );
});

test('docs ERA+ odt section footnote styles enabled', async ({ page }) => {
  await page.route('**/api/v1/docs/doc-era-plus-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync') || url.includes('/snapshot')) {
      if (url.includes('/snapshot')) {
        await route.fulfill({ status: 200, body: '' });
        return;
      }
      await route.continue();
      return;
    }
    if (url.includes('/export/odt') && route.request().method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/vnd.oasis.opendocument.text',
        body: Buffer.from('PK\x03\x04odt'),
      });
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
  await page.goto('/docs/doc-era-plus-1');
  await expect(page.locator('#menubar')).toBeVisible();

  await page.getByRole('button', { name: 'File', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Download' }).hover();
  await expect(page.locator('.era-menu-item[data-cmd="file.odt"]')).toBeEnabled();

  await page.getByRole('button', { name: 'Insert', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Break' }).hover();
  await page.locator('.era-menu-item[data-cmd="insert.section"]').click();
  await expect(page.locator('.doc-block[data-type="section_break"]')).toBeVisible();

  page.once('dialog', (d) => d.accept('Note body'));
  await page.locator('.doc-block').first().click();
  await page.getByRole('button', { name: 'Insert', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="insert.footnote"]').click();
  await expect(page.locator('.doc-block[data-type="footnote"]')).toBeVisible();

  await page.evaluate(() => {
    const h = (window as unknown as { docsMenuHandlers?: Record<string, () => void> }).docsMenuHandlers;
    if (h && h['format.styles']) h['format.styles']();
  });
  await expect(page.locator('#stylesDlg')).toBeVisible();
});

test('docs LATER line numbers text box columns review', async ({ page }) => {
  await page.route('**/api/v1/docs/doc-later-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync') || url.includes('/snapshot')) {
      if (url.includes('/snapshot')) {
        await route.fulfill({ status: 200, body: '' });
        return;
      }
      await route.continue();
      return;
    }
    if (url.includes('/export/rtf') && route.request().method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/rtf',
        body: '{\\rtf1 Hello RTF}',
      });
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
  await page.goto('/docs/doc-later-1');
  await page.locator('.doc-block').first().click();

  await page.getByRole('button', { name: 'View', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="view.lineNumbers"]').click();
  await expect(page.locator('#blocks')).toHaveClass(/line-numbers/);

  await page.getByRole('button', { name: 'Insert', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="insert.textbox"]').click();
  await expect(page.locator('.doc-block[data-type="text_box"]')).toBeVisible();

  page.once('dialog', (d) => d.accept('2'));
  await page.evaluate(() => {
    const h = (window as unknown as { docsMenuHandlers?: Record<string, () => void> }).docsMenuHandlers;
    if (h && h['format.columns']) h['format.columns']();
  });
  await expect(page.locator('#blocks')).toHaveClass(/columns-2/);

  await page.getByRole('button', { name: 'Tools', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="tools.review"]').click();
  await expect(page.locator('#reviewDlg')).toBeVisible();
  await page.locator('#trackChangesChk').check();
  await page.locator('#reviewDlg button[value="ok"]').click();

  await page.getByRole('button', { name: 'File', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Download' }).hover();
  await expect(page.locator('.era-menu-item[data-cmd="file.rtf"]')).toBeEnabled();
});

test('docs Format paragraph styles submenu', async ({ page }) => {
  await page.route('**/api/v1/docs/doc-submenu-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync') || url.includes('/snapshot')) {
      if (url.includes('/snapshot')) {
        await route.fulfill({ status: 200, body: '' });
        return;
      }
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
  await page.goto('/docs/doc-submenu-1');
  await page.locator('.doc-block').first().click();
  await page.getByRole('button', { name: 'Format', exact: true }).click();
  await expect(page.locator('.era-submenu-btn', { hasText: 'Text' })).toBeVisible();
  await expect(page.locator('.era-menu-item[data-cmd="format.bold"]')).toBeHidden();
  await page.locator('.era-submenu-btn', { hasText: 'Paragraph styles' }).hover();
  await expect(page.locator('.era-menu-item[data-cmd="format.h1"]')).toBeVisible();
  await page.locator('.era-menu-item[data-cmd="format.h1"]').click();
  await expect(page.locator('.doc-block').first()).toHaveAttribute('data-level', '1');
});

test('docs suggesting mode and fullscreen menus enabled', async ({ page }) => {
  await page.route('**/api/v1/docs/doc-suggest-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync') || url.includes('/snapshot')) {
      if (url.includes('/snapshot')) {
        await route.fulfill({ status: 200, body: '' });
        return;
      }
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
  await page.goto('/docs/doc-suggest-1');
  await expect(page.locator('#menubar')).toBeVisible();

  await page.getByRole('button', { name: 'View', exact: true }).click();
  await expect(page.locator('.era-menu-item[data-cmd="view.suggest"]')).toBeEnabled();
  await expect(page.locator('.era-menu-item[data-cmd="view.fullscreen"]')).toBeEnabled();
  await page.locator('.era-menu-item[data-cmd="view.suggest"]').click();
  await expect(page.locator('#authStatus')).toContainText(/Suggesting on/i);

  await page.getByRole('button', { name: 'View', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="view.suggest"]').click();
  await expect(page.locator('#authStatus')).toContainText(/Suggesting off/i);
});

test('docs page setup numbered list undo', async ({ page }) => {
  await page.route('**/api/v1/docs/doc-gov-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync') || url.includes('/snapshot')) {
      if (url.includes('/snapshot')) {
        await route.fulfill({ status: 200, body: '' });
        return;
      }
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
  await page.goto('/docs/doc-gov-1');
  await page.locator('.doc-block').first().click();

  await page.getByRole('button', { name: 'File', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="file.pageSetup"]').click();
  await expect(page.locator('#pageSetupDlg')).toBeVisible();
  await page.locator('#pageMargins').fill('25');
  await page.locator('#pageSetupOk').click();

  await page.getByRole('button', { name: 'Numbered list' }).click();
  await expect(page.locator('.doc-block').first()).toHaveAttribute('data-type', 'list_item');
  await expect(page.locator('.doc-block').first()).toHaveAttribute('data-list', 'ordered');

  await page.keyboard.type(' more');
  await page.getByRole('button', { name: 'Edit', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="edit.undo"]').click();
});
