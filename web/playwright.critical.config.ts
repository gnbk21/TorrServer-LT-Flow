import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./e2e",
  testMatch: "flow13-critical.spec.ts",
  outputDir: "critical-results",
  workers: 2,
  fullyParallel: true,
  use: { baseURL: "http://127.0.0.1:4175", trace: "retain-on-failure" },
  projects: [
    { name: "chromium", use: { browserName: "chromium" } },
    { name: "firefox", use: { browserName: "firefox" } },
    { name: "webkit", use: { browserName: "webkit" } },
  ],
  webServer: {
    command:
      "node node_modules/vite/bin/vite.js preview --host 127.0.0.1 --port 4175",
    url: "http://127.0.0.1:4175",
    reuseExistingServer: false,
  },
  reporter: [
    ["list"],
    ["html", { outputFolder: "critical-report", open: "never" }],
  ],
});
