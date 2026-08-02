import { defineConfig, devices } from "@playwright/test";

import { e2eEnv } from "./e2e/env";

export default defineConfig({
  testDir: "./e2e",
  outputDir: "./.playwright-results",
  globalSetup: "./e2e/global-setup.ts",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? "github" : "list",
  use: {
    baseURL: e2eEnv.consoleUrl,
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: [
    {
      command: `"${e2eEnv.binaryPath}" serve`,
      cwd: e2eEnv.repoRoot,
      url: `${e2eEnv.apiUrl}/api/health`,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      env: {
        ...process.env,
        DATABASE_URL: e2eEnv.databaseUrl,
      },
    },
    {
      command: `npm run dev -- --host 127.0.0.1 --port ${e2eEnv.consolePort}`,
      cwd: e2eEnv.consoleDir,
      url: e2eEnv.consoleUrl,
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
  ],
});
