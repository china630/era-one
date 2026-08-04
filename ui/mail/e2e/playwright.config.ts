import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  timeout: 30_000,
  use: {
    baseURL: process.env.ERA_UI_MAIL_URL || 'http://127.0.0.1:8180',
    headless: true,
  },
  webServer: process.env.ERA_E2E_SKIP_SERVER
    ? undefined
    : {
        command: 'go run ./cmd/ui-mail',
        cwd: '..',
        url: 'http://127.0.0.1:8180/mail/healthz',
        reuseExistingServer: true,
        timeout: 60_000,
      },
});
