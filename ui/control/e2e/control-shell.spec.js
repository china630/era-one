const { test, expect } = require('@playwright/test');

/**
 * Control shell smoke — requires control-plane on :8090 with ERA_UI_DIR pointing at repo ui/.
 * Skip if ERA_CONTROL_E2E!=1 (CI optional).
 */
const base = process.env.ERA_CONTROL_URL || 'http://127.0.0.1:8090';
const enabled = process.env.ERA_CONTROL_E2E === '1';

test.describe('Control shell', () => {
  test.skip(!enabled, 'set ERA_CONTROL_E2E=1 to run');

  test('SOC home loads shell chrome', async ({ page }) => {
    await page.goto(base + '/ui/control/');
    await expect(page.locator('.era-brand')).toContainText('ERA Control');
    await expect(page.locator('#era-nav a').first()).toBeVisible();
  });

  test('Manage module shows WHQL honesty banner', async ({ page }) => {
    await page.goto(base + '/ui/control/manage/');
    await expect(page.locator('.era-banner')).toContainText('telemetry_only');
  });

  test('Vuln and Perimeter modules exist', async ({ page }) => {
    await page.goto(base + '/ui/control/vuln/');
    await expect(page.locator('h1')).toContainText('Vuln');
    await page.goto(base + '/ui/control/perimeter/');
    await expect(page.locator('h1')).toContainText('Perimeter');
  });

  test('viewer cannot PUT enforcement via API', async ({ request }) => {
    const res = await request.put(base + '/api/v1/enforcement/policy', {
      headers: { 'X-ERA-Role': 'viewer', 'Content-Type': 'application/json' },
      data: {},
    });
    expect([401, 403]).toContain(res.status());
  });
});
