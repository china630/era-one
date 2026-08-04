import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  timeout: 60_000,
  retries: process.env.CI ? 2 : 0,
  use: {
    baseURL: process.env.ERA_WORKSPACE_URL || 'http://127.0.0.1:8170',
    headless: true,
  },
  webServer: process.env.ERA_E2E_SKIP_SERVER
    ? undefined
    : {
        command: 'go run ./services/platform/cmd/workspace',
        cwd: '../../..',
        url: 'http://127.0.0.1:8170/healthz',
        reuseExistingServer: true,
        timeout: 120_000,
        env: {
          ERA_DRIVE_API_URL: 'http://127.0.0.1:8175',
          ERA_DOCS_API_URL: 'http://127.0.0.1:8142',
        },
      },
});
