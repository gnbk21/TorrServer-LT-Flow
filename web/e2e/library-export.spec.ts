import { test, expect } from "@playwright/test";
import { readFile } from "node:fs/promises";

for (const legacy of [false, true]) {
  test(`library downloads preserve formats and exclude private data (${legacy ? "legacy" : "modern"} server)`, async ({
    page,
  }) => {
    const row = {
      hash: "ab".repeat(20),
      title: "Portable episode",
      category: "series",
      stat: 5,
      data: '{"password":"export-secret"}',
      poster: "https://example.test/?token=export-secret",
      torrs_hash: "OpaquePrivateSecret",
    };
    let legacyReads = 0;
    await page.route("**/*", async (route) => {
      const path = new URL(route.request().url()).pathname;
      if (path === "/settings/metadata/v1")
        return route.fulfill({
          status: 404,
          json: { error: "older fixture server" },
        });
      if (path === "/echo")
        return route.fulfill({ body: "MatriX.145.Flow-test" });
      if (path === "/flow/library")
        return route.fulfill({
          json: {
            items: [row],
            total: 1,
            library_total: 1,
            page: 1,
            limit: 50,
            categories: ["series"],
            sampled_at: new Date().toISOString(),
          },
        });
      if (path === "/flow/library/export")
        return legacy
          ? route.fulfill({ status: 404, json: { error: "missing" } })
          : route.fulfill({
              json: [
                {
                  hash: row.hash,
                  title: row.title,
                  category: row.category,
                  torrs_hash: "SafeToken",
                  data: "export-secret",
                },
              ],
            });
      if (path === "/torrents") {
        legacyReads++;
        return route.fulfill({ json: [row] });
      }
      if (path === "/flow/active")
        return route.fulfill({
          json: { items: [], sampled_at: new Date().toISOString() },
        });
      if (path === "/flow/network")
        return route.fulfill({
          json: {
            state: "ADDRESS_READY",
            connectivity: "ONLINE",
            addresses: [],
            checked_at: new Date().toISOString(),
            changed_at: new Date().toISOString(),
            next_check_seconds: 15,
            reannounce_count: 0,
          },
        });
      if (path === "/flow/tray")
        return route.fulfill({
          json: {
            server_state: "RUNNING",
            download_rate: 0,
            buffer_seconds: 0,
            peer_count: 0,
          },
        });
      return route.continue();
    });
    await page.goto("/#/torrents");
    await expect(
      page.getByRole("heading", { name: row.title, exact: true }),
    ).toBeVisible();
    await page.getByText("Export library", { exact: true }).click();
    await expect(
      page.getByText(/omit credentials, private trackers/),
    ).toBeVisible();
    for (const format of ["json", "magnets", "torrs"] as const) {
      const downloadPending = page.waitForEvent("download");
      await page.getByRole("button", { name: format, exact: true }).click();
      const download = await downloadPending;
      expect(download.suggestedFilename()).toBe(
        `flow-library.${format === "json" ? "json" : "txt"}`,
      );
      const path = await download.path();
      const text = await readFile(path!, "utf8");
      expect(text).not.toContain("export-secret");
      expect(text).not.toContain("OpaquePrivateSecret");
      if (format === "json")
        expect(JSON.parse(text)[0]).toMatchObject({
          hash: row.hash,
          title: row.title,
          category: row.category,
        });
      else
        expect(text).toContain(
          format === "torrs" && !legacy
            ? "torrs://SafeToken"
            : `urn:btih:${row.hash}`,
        );
    }
    expect(legacyReads).toBe(legacy ? 3 : 0);
  });
}
