import { defineConfig, devices } from '@playwright/test';

// End-to-end tests drive the real binary against a migrated, seeded Postgres
// (see `task test:e2e`). They run on a dedicated port so a dev server on :3000
// is never disturbed.
const PORT = 3100;
const BASE_URL = `http://127.0.0.1:${PORT}`;

export default defineConfig({
  testDir: './e2e',
  // Tests share one seeded database and mutate it, so run them serially.
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? 'github' : [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: 'go run ./cmd/app serve',
    url: `${BASE_URL}/readyz`,
    reuseExistingServer: false,
    timeout: 180_000,
    env: {
      APP_ENV: 'dev',
      HTTP_ADDR: `127.0.0.1:${PORT}`,
      SITE_URL: BASE_URL,
      PRODUCT_NAME: 'Grumble',
      DATABASE_URL:
        process.env.DATABASE_URL ??
        'postgres://grumble:grumble@localhost:5433/grumble?sslmode=disable',
      ANON_ID_SECRET: 'dev-anon-secret-change-me-0000',
      IP_HASH_SALT: 'dev-ip-hash-salt-change-me0000',
    },
  },
});
