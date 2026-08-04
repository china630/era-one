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

function sheetPayload(cells: Record<string, { value?: string; formula?: string; format?: string }>) {
  return {
    name: 'Untitled.erat',
    rows: 1024,
    cols: 256,
    cells,
    sheets: [{ name: 'Sheet1' }, { name: 'Sheet2' }],
    active_sheet: 0,
  };
}

test('tables menubar number format insert row COUNTIF rename', async ({ page }) => {
  let cells: Record<string, { value?: string; formula?: string; format?: string }> = {
    A1: { value: '10' },
    A2: { value: '20' },
  };

  await page.route('**/api/v1/tables/sheet-wave-c**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(sheetPayload(cells)),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/tables/sheet-wave-c');
  await expect(page.locator('#menubar')).toBeVisible();

  await page.locator('td[data-addr="A1"]').click();
  await page.getByRole('button', { name: 'Format', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="format.number"]').click();

  await page.getByRole('button', { name: 'Insert', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Sheet' }).hover();
  await page.locator('.era-menu-item[data-cmd="insert.row"]').click();

  await expect(page.getByRole('button', { name: 'COUNTIF' })).toBeVisible();
  await page.getByRole('button', { name: 'COUNTIF' }).click();
  await expect(page.locator('#formulaInput')).toHaveValue(/=COUNTIF/);

  page.once('dialog', (d) => d.accept('Renamed'));
  await page.locator('#sheetTabs .tab.active').dblclick();
  await expect(page.locator('#sheetTabs .tab.active')).toHaveText('Renamed', { timeout: 5_000 });
});

test('tables W2 protect and filter options', async ({ page }) => {
  await page.route('**/api/v1/tables/sheet-w2-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          sheetPayload({
            A1: { value: 'alpha' },
            A2: { value: 'beta' },
            B1: { value: 'keep' },
          })
        ),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/tables/sheet-w2-1');
  await expect(page.locator('#menubar')).toBeVisible();

  await page.getByRole('button', { name: 'Data', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="data.filterOpts"]').click();
  await expect(page.locator('#filterOptsDlg')).toBeVisible();
  await page.locator('#filterOptsMode').selectOption('contains');
  await page.locator('#filterOptsVal').fill('alpha');
  await page.locator('#filterOptsDlg button[value="apply"]').click();
  await expect(page.locator('tr[data-row="1"]')).not.toHaveClass(/filter-hidden/);
  await expect(page.locator('tr[data-row="2"]')).toHaveClass(/filter-hidden/);

  await page.getByRole('button', { name: 'Data', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Protect' }).hover();
  await page.locator('.era-menu-item[data-cmd="data.protect"]').click();
  await expect(page.locator('#authStatus')).toContainText(/protected/i);
  await expect(page.locator('#grid')).toHaveClass(/sheet-protected/);
});

test('tables ERA+ ODS freeze panes subtotal menus enabled', async ({ page }) => {
  await page.route('**/api/v1/tables/sheet-era-plus**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          sheetPayload({
            A1: { value: '10' },
            A2: { value: '20' },
          })
        ),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/tables/sheet-era-plus');
  await expect(page.locator('#menubar')).toBeVisible();

  await page.getByRole('button', { name: 'File', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Download' }).hover();
  await expect(page.locator('.era-menu-item[data-cmd="file.ods"]')).toBeEnabled();
  await page.keyboard.press('Escape');

  await page.getByRole('button', { name: 'View', exact: true }).click();
  await expect(page.locator('.era-menu-item[data-cmd="view.freezePanes"]')).toBeEnabled();
  await page.keyboard.press('Escape');

  await page.locator('td[data-addr="A2"]').click();
  await page.getByRole('button', { name: 'Data', exact: true }).click();
  const subtotal = page.locator('.era-menu-item[data-cmd="data.subtotal"]');
  await expect(subtotal).toBeEnabled();
  await subtotal.click();
  await expect(page.locator('#authStatus')).toContainText(/Subtotal lite/i);
});

test('tables CSV merge protect ranges lite', async ({ page }) => {
  await page.route('**/api/v1/tables/sheet-planned-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          sheetPayload({
            A1: { value: 'hello' },
            B1: { value: 'world' },
            A2: { value: 'x' },
          })
        ),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/tables/sheet-planned-1');
  await expect(page.locator('#menubar')).toBeVisible();

  await page.getByRole('button', { name: 'File', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Download' }).hover();
  const csv = page.locator('.era-menu-item[data-cmd="file.csv"]');
  await expect(csv).toBeEnabled();
  const downloadPromise = page.waitForEvent('download');
  await csv.click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/\.csv$/i);
  await expect(page.locator('#authStatus')).toContainText(/CSV/i);

  await page.locator('td[data-addr="A1"]').click();
  page.once('dialog', (d) => d.accept('A1:B1'));
  await page.getByRole('button', { name: 'Format', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="format.merge"]').click();
  await expect(page.locator('#authStatus')).toContainText(/Merged/i);
  await expect(page.locator('td[data-addr="A1"]')).toHaveAttribute('colspan', '2');

  page.once('dialog', (d) => d.accept('A2:A2'));
  await page.getByRole('button', { name: 'Data', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Protect' }).hover();
  await page.locator('.era-menu-item[data-cmd="data.protectRanges"]').click();
  await expect(page.locator('#authStatus')).toContainText(/Protected range/i);
  await expect(page.locator('td[data-addr="A2"]')).toHaveClass(/range-protected/);
});

test('tables LATER sparkline enabled and what-if dialog', async ({ page }) => {
  await page.route('**/api/v1/tables/sheet-later-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          sheetPayload({
            A1: { value: '1' },
            A2: { value: '3' },
            A3: { value: '2' },
            B1: { value: '11', formula: '=A1+10' },
          })
        ),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/tables/sheet-later-1');
  await expect(page.locator('#menubar')).toBeVisible();

  const sparkline = page.locator('.era-menu-item[data-cmd="insert.sparkline"]');
  await page.getByRole('button', { name: 'Insert', exact: true }).click();
  await expect(sparkline).toBeEnabled();
  await page.keyboard.press('Escape');

  await page.getByRole('button', { name: 'Data', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="data.whatif"]').click();
  await expect(page.locator('#whatIfDlg')).toBeVisible();
  await expect(page.locator('#whatIfFormula')).toBeVisible();
  await page.locator('#whatIfDlg button[value="cancel"]').click();
  await expect(page.locator('#whatIfDlg')).toBeHidden();
});
