import { defineConfig } from "@playwright/test";
export default defineConfig({
  outputDir: "lab-results",
  testDir: "./e2e",
  testMatch: "upgrade-performance.lab.ts",
  workers: 1,
  use: {
    baseURL: "http://127.0.0.1:4173",
    viewport: { width: 412, height: 915 },
    trace: "retain-on-failure",
  },
  webServer: {
    command:
      "node node_modules/vite/bin/vite.js preview --host 127.0.0.1 --port 4173",
    url: "http://127.0.0.1:4173",
  },
  reporter: [["list"]],
});
