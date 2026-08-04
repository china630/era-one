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

function sheetPayload(cells: Record<string, { value?: string; formula?: string }>) {
  return {
    name: 'Untitled.erat',
    rows: 1024,
    cols: 256,
    cells,
  };
}

test('tables new sheet opens grid editor', async ({ page }) => {
  await page.route('**/api/v1/tables', async (route) => {
    if (route.request().method() === 'POST' && !route.request().url().includes('/import')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ drive_object_id: 'sheet-new-1' }),
      });
      return;
    }
    await route.continue();
  });
  await page.route('**/api/v1/tables/sheet-new-1**', async (route) => {
    if (route.request().url().includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(sheetPayload({})),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/tables');
  await expect(page.locator('.era-brand-mod')).toHaveText('Tables');
  await page.getByRole('button', { name: 'New sheet' }).click();
  await expect(page).toHaveURL(/\/tables\/sheet-new-1/);
  await expect(page.locator('table.sheet')).toBeVisible();
  await expect(page.locator('td[data-addr="A1"]')).toBeVisible();
});

test('tables toolbar shows COUNT and sheet tabs', async ({ page }) => {
  await page.route('**/api/v1/tables/sheet-ui-1**', async (route) => {
    if (route.request().url().includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...sheetPayload({}),
          sheets: [{ name: 'Sheet1' }, { name: 'Sheet2' }],
          active_sheet: 0,
        }),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/tables/sheet-ui-1');
  await expect(page.getByRole('button', { name: 'COUNT', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'COUNTIF', exact: true })).toBeVisible();
  await expect(page.locator('#sheetTabs .tab')).toHaveCount(2);
  await expect(page.locator('#sheetTabs .tab.active')).toHaveText('Sheet1');
});

test('tables cell edit keyboard and SUM formula', async ({ page }) => {
  let cells: Record<string, { value?: string; formula?: string }> = {
    A1: { value: '10' },
    A2: { value: '20' },
  };

  await page.route('**/api/v1/tables/sheet-sum-1**', async (route) => {
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
  await page.goto('/tables/sheet-sum-1');
  const a1 = page.locator('td[data-addr="A1"]');
  const a2 = page.locator('td[data-addr="A2"]');
  const a3 = page.locator('td[data-addr="A3"]');
  await expect(a1).toContainText('10');
  await expect(a2).toContainText('20');

  // GET refresh after commit will return recalculated SUM.
  cells = {
    A1: { value: '10' },
    A2: { value: '20' },
    A3: { value: '30', formula: '=SUM(A1:A2)' },
  };

  await a3.click();
  await page.keyboard.type('=SUM(A1:A2)');
  await a3.press('Enter');

  await expect(a3).toContainText('30', { timeout: 5_000 });
  await expect(a3).toHaveClass(/formula/);

  await a1.click();
  await page.keyboard.press('Alt+ArrowDown');
  await expect(a2).toHaveClass(/selected/);
});

test('tables row-aware sort moves sibling columns', async ({ page }) => {
  let getCount = 0;
  await page.route('**/api/v1/tables/sheet-sort-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      getCount += 1;
      const cells =
        getCount === 1
          ? {
              A1: { value: 'b' },
              B1: { value: 'row-b' },
              A2: { value: 'a' },
              B2: { value: 'row-a' },
            }
          : {
              A1: { value: 'a' },
              B1: { value: 'row-a' },
              A2: { value: 'b' },
              B2: { value: 'row-b' },
            };
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
  await page.goto('/tables/sheet-sort-1');
  await expect(page.locator('td[data-addr="A1"]')).toContainText('b');
  await expect(page.locator('td[data-addr="B1"]')).toContainText('row-b');
  await page.locator('td[data-addr="A1"]').click();
  await page.getByRole('button', { name: 'Data', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="data.sort"]').click();
  await expect(page.locator('#authStatus')).toContainText(/Sorting rows A→Z/i);
  await expect(page.locator('td[data-addr="A1"]')).toContainText('a', { timeout: 5_000 });
  await expect(page.locator('td[data-addr="B1"]')).toContainText('row-a');
  await expect(page.locator('td[data-addr="B2"]')).toContainText('row-b');
});

test('tables import xlsx redirects to grid', async ({ page }) => {
  await page.route('**/api/v1/tables/import', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ drive_object_id: 'sheet-import-1' }),
    });
  });
  await page.route('**/api/v1/tables/sheet-import-1**', async (route) => {
    if (route.request().url().includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          sheetPayload({
            A1: { value: 'Imported' },
          })
        ),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/tables');
  await expect(page.getByRole('button', { name: 'Import xlsx' })).toBeVisible();
  const tmp = path.join(os.tmpdir(), `era-tables-e2e-${Date.now()}.xlsx`);
  fs.writeFileSync(tmp, Buffer.from('PK\x03\x04fake-xlsx'));
  await page.setInputFiles('#file', tmp);
  await expect(page).toHaveURL(/\/tables\/sheet-import-1/, { timeout: 10_000 });
  await expect(page.locator('td[data-addr="A1"]')).toContainText('Imported');
});

test('tables viewport grows beyond 12x6', async ({ page }) => {
  await page.route('**/api/v1/tables/sheet-grow-1**', async (route) => {
    if (route.request().url().includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          sheetPayload({
            G20: { value: 'far' },
            P40: { value: 'edge' },
          })
        ),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/tables/sheet-grow-1');
  await expect(page.locator('td[data-addr="G20"]')).toBeVisible();
  await expect(page.locator('td[data-addr="G20"]')).toContainText('far');
  await expect(page.locator('td[data-addr="P40"]')).toBeVisible();
  await page.getByRole('button', { name: '+4 cols' }).click();
  await expect(page.locator('th').filter({ hasText: 'Q' })).toBeVisible();
});

test('drive new sheet opens tables editor', async ({ page }) => {
  await page.route('**/api/v1/drive/folders/**/children', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ folders: [], objects: [] }),
    });
  });
  await page.route('**/api/v1/tables', async (route) => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ drive_object_id: 'sheet-from-drive' }),
      });
      return;
    }
    await route.continue();
  });
  await page.route('**/api/v1/tables/sheet-from-drive**', async (route) => {
    if (route.request().url().includes('/sync')) {
      await route.continue();
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(sheetPayload({})),
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/drive/');
  await page.getByRole('button', { name: 'New sheet' }).click();
  await expect(page).toHaveURL(/\/tables\/sheet-from-drive/);
  await expect(page.locator('.era-brand-mod')).toHaveText('Tables');
  await expect(page.locator('table.sheet')).toBeVisible();
});
