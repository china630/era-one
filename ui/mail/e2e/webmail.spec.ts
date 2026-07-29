import { test, expect } from '@playwright/test';
import { createHmac } from 'crypto';

function signTestJwt(secret: string, payload: Record<string, unknown>): string {
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
  const body = Buffer.from(JSON.stringify(payload)).toString('base64url');
  const data = `${header}.${body}`;
  const sig = createHmac('sha256', secret).update(data).digest('base64url');
  return `${data}.${sig}`;
}

test('webmail smoke with mocked token and APIs', async ({ page }) => {
  const secret = 'dev-only-change-in-prod';
  const token = signTestJwt(secret, {
    sub: 'u-alice',
    email: 'alice@mail.gov.az',
    tenant_id: 't-demo',
    exp: Math.floor(Date.now() / 1000) + 3600,
  });

  await page.route('**/mail/api/policy**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        quota_mb_per_user: 512,
        max_attachment_size_mb: 25,
        retention_days: 365,
        max_attachments_per_message: 50,
      }),
    });
  });

  await page.route('**/mail/api/mailbox**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        email: 'alice@mail.gov.az',
        quota_bytes: 1024,
        used_bytes: 1024,
      }),
    });
  });

  await page.route('**/mail/api/message?**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ uid: 1, subject: 'Hello', body: 'Test body' }),
    });
  });

  await page.route('**/mail/api/messages**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        messages: [{ uid: 1, subject: 'Hello', body: 'Test body' }],
      }),
    });
  });

  await page.addInitScript(
    ({ tokenKey, token, emailKey, email, tenantKey, tenant }) => {
      localStorage.setItem(tokenKey, token);
      localStorage.setItem(emailKey, email);
      localStorage.setItem(tenantKey, tenant);
    },
    {
      tokenKey: 'era_mail_token',
      token,
      emailKey: 'era_mail_email',
      email: 'alice@mail.gov.az',
      tenantKey: 'era_mail_tenant',
      tenant: 't-demo',
    }
  );

  await page.goto('/mail');
  await expect(page.getByRole('heading', { name: 'ERA Webmail' })).toBeVisible();
  await expect(page.locator('#user')).toHaveText('alice@mail.gov.az');
  await expect(page.locator('#policy')).toContainText('512 MB');

  await page.locator('#inbox li').first().click();
  await expect(page.locator('#msgBody')).toHaveText('Test body');

  await page.locator('#composeBtn').click();
  const sendBtn = page.locator('#sendBtn');
  await expect(sendBtn).toBeDisabled();
});
