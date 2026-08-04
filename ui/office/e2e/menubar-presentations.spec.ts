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

function deckPayload(slides: Record<string, unknown>[]) {
  return { name: 'Untitled.erap', format: 'erap', slides };
}

test('presentations present mode two_column and menubar', async ({ page }) => {
  await page.route('**/api/v1/presentations/deck-wave-d**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          deckPayload([
            { id: 's1', title: 'One', body: 'A', layout: 'title_body' },
            { id: 's2', title: 'Two', body: 'B', body2: 'C', layout: 'two_column' },
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
  await page.goto('/presentations/deck-wave-d');
  await expect(page.locator('#menubar')).toBeVisible();

  await page.locator('#nextBtn').click();
  await expect(page.locator('#slidePos')).toContainText('Slide 2 / 2');
  await page.locator('#layoutSelect').selectOption('two_column');
  await expect(page.locator('.slide-canvas')).toHaveClass(/layout-two_column/);
  await expect(page.locator('#slideBody2')).toBeVisible();

  await page.locator('#presentBtn').click();
  await expect(page.locator('#presentOverlay')).toHaveClass(/active/);
  await expect(page.locator('#presentTitle')).toContainText('Two');
  await page.keyboard.press('ArrowLeft');
  await expect(page.locator('#presentTitle')).toContainText('One');
  await page.keyboard.press('Escape');
  await expect(page.locator('#presentOverlay')).not.toHaveClass(/active/);

  await page.locator('#findSlideInput').fill('Two');
  await page.locator('#findSlideBtn').click();
  await expect(page.locator('#slidePos')).toContainText('Slide 2 / 2');
  await expect(page.locator('#authStatus')).toContainText(/Found on slide 2/i);

  await page.getByRole('button', { name: 'File', exact: true }).click();
  await expect(page.locator('.era-menu-item[data-cmd="file.print"]')).toBeEnabled();
});

test('presentations speaker notes persist and show in present mode', async ({ page }) => {
  let lastPut: string | null = null;
  await page.route('**/api/v1/presentations/deck-notes-1**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          deckPayload([{ id: 's1', title: 'Talk', body: 'Point', layout: 'title_body', notes: '' }])
        ),
      });
      return;
    }
    if (route.request().method() === 'PUT') {
      lastPut = route.request().postData() || '{}';
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: lastPut,
      });
      return;
    }
    await route.continue();
  });

  await injectToken(page);
  await page.goto('/presentations/deck-notes-1');
  await expect(page.locator('#speakerNotes')).toBeVisible();
  await page.locator('#speakerNotes').fill('Remember to smile');
  await page.waitForTimeout(700);
  expect(lastPut).toBeTruthy();
  expect(JSON.parse(lastPut!).slides[0].notes).toContain('Remember to smile');

  await page.locator('#presentBtn').click();
  await expect(page.locator('#presentOverlay')).toHaveClass(/active/);
  await expect(page.locator('#presentNotes')).toContainText('Remember to smile');
  await page.keyboard.press('Escape');
});

test('presentations edit master sets default layout for new slides', async ({ page }) => {
  await page.route('**/api/v1/presentations/deck-master-1**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          deckPayload([{ id: 's1', title: 'One', body: 'A', layout: 'title_body' }])
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
  await page.goto('/presentations/deck-master-1');
  await expect(page.locator('#slideTitle')).toContainText('One');

  await page.getByRole('button', { name: 'Slide', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="slide.master"]').click();
  await expect(page.locator('#masterDlg')).toBeVisible();
  await page.locator('#masterDefaultLayout').selectOption('section');
  await page.locator('#masterTitlePlaceholder').fill('Section title');
  await page.locator('#masterDlgApply').click();
  await expect(page.locator('#masterDlg')).toBeHidden();
  await expect(page.locator('#authStatus')).toContainText(/Master applied/i);

  await page.locator('#addSlideBtn').click();
  await expect(page.locator('#slidePos')).toContainText('Slide 2 / 2');
  await expect(page.locator('#slideTitle')).toContainText('Section title');
  await expect(page.locator('.slide-canvas')).toHaveClass(/layout-section/);
  await expect(page.locator('#layoutSelect')).toHaveValue('section');
});

test('presentations file.odp enabled and export mocked 200', async ({ page }) => {
  await page.route('**/api/v1/presentations/deck-odp-1**', async (route) => {
    const url = route.request().url();
    if (url.includes('/export/odp') && route.request().method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/vnd.oasis.opendocument.presentation',
        body: 'PK\x03\x04mimetypeapplication/vnd.oasis.opendocument.presentation',
      });
      return;
    }
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          deckPayload([{ id: 's1', title: 'ODP', body: 'Export', layout: 'title_body' }])
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
  await page.goto('/presentations/deck-odp-1');
  await expect(page.locator('#slideTitle')).toContainText('ODP');

  await page.getByRole('button', { name: 'File', exact: true }).click();
  await page.locator('.era-submenu-btn', { hasText: 'Download' }).hover();
  const odpItem = page.locator('.era-menu-item[data-cmd="file.odp"]');
  await expect(odpItem).toBeEnabled();
  await odpItem.click();
  await expect(page.locator('#authStatus')).toContainText(/Export ready/i);
});

test('presentations P-LITE share print present image', async ({ page }) => {
  const tinyPng =
    'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==';
  await page.route('**/api/v1/presentations/deck-plite-1**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          deckPayload([
            {
              id: 's1',
              title: 'Alpha',
              body: 'One',
              layout: 'title_body',
              background: '#e8f0fe',
              image_url: tinyPng,
              notes: 'note-a',
            },
            {
              id: 's2',
              title: 'Beta',
              body: 'Two',
              layout: 'title_body',
              notes: 'note-b',
            },
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
  await page.goto('/presentations/deck-plite-1');
  await expect(page.locator('#slideTitle')).toContainText('Alpha');

  await page.getByRole('button', { name: 'Share', exact: true }).click();
  await expect(page.locator('#shareDlg')).toBeVisible();
  await expect(page.locator('#shareLinkInput')).toHaveValue(/\/presentations\/deck-plite-1/);
  await expect(page.locator('#shareDriveLink')).toHaveAttribute('href', /share=deck-plite-1/);
  await page.locator('#shareDlg button[value="ok"]').click();

  await page.evaluate(() => {
    (window as unknown as { print: () => void }).print = () => {};
  });
  await page.getByRole('button', { name: 'File', exact: true }).click();
  await page.locator('.era-menu-item[data-cmd="file.print"]').click();
  await expect(page.locator('#printRoot .print-slide')).toHaveCount(2);
  await expect(page.locator('#authStatus')).toContainText(/2 slide/i);
  await page.keyboard.press('Escape');

  await page.locator('#presentBtn').click();
  await expect(page.locator('#presentOverlay')).toHaveClass(/active/);
  await expect(page.locator('#presentImage')).toBeVisible();
  await expect(page.locator('#presentSlide')).toHaveCSS('background-color', 'rgb(232, 240, 254)');
  await page.keyboard.press('Escape');

  await page.getByRole('button', { name: 'Edit', exact: true }).click();
  await expect(page.locator('.era-menu-item[data-cmd="edit.redo"]')).toBeEnabled();
});

test('presentations filmstrip drag reorder', async ({ page }) => {
  await page.route('**/api/v1/presentations/deck-drag-1**', async (route) => {
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
  await page.goto('/presentations/deck-drag-1');
  const first = page.locator('#filmstrip li').first();
  const second = page.locator('#filmstrip li').nth(1);
  await first.dragTo(second);
  await expect(page.locator('#filmstrip li').first()).toContainText('Second');
});
