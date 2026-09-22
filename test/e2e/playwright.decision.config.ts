import { defineConfig } from '@playwright/test';

// Standalone decision tests launch isolated real daemons; no shared review fixtures.
export default defineConfig({
  testDir: './tests', testMatch: /\.decidemode\.spec\.ts$/,
  workers: 1, timeout: 30_000, expect: { timeout: 10_000 },
  reporter: [['list']],
  use: { browserName: 'chromium', screenshot: 'only-on-failure' },
});
