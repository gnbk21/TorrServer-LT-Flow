import { test, expect } from "@playwright/test";
import { writeFile } from "node:fs/promises";

for (const total of [1, 200, 1000, 5000])
  test(`library projection handles ${total} entries with bounded rendering`, async ({
    page,
  }, info) => {
    const library = Array.from({ length: total }, (_, i) => ({
      hash: i.toString(16).padStart(40, "0"),
      title: `Fixture ${String(i).padStart(4, "0")}`,
      stat: 3,
      timestamp: i,
      torrent_size: 12345678,
      category: "Fixtures",
    }));
    let libraryRequests = 0;
    await page.route("**/*", async (route) => {
      const url = new URL(route.request().url());
      const path = url.pathname;
      if (path === "/settings/metadata/v1")
        return route.fulfill({
          status: 404,
          json: { error: "older fixture server" },
        });
      if (path === "/echo")
        return route.fulfill({ body: "MatriX.145.Flow-test" });
      if (path === "/flow/active")
        return route.fulfill({
          json: { items: [], sampled_at: new Date().toISOString() },
        });
      if (path === "/flow/library") {
        libraryRequests++;
        const query = url.searchParams.get("q") || "";
        const sort = url.searchParams.get("sort") || "recent";
        const matched = library
          .filter((row) =>
            row.title.toLowerCase().includes(query.toLowerCase()),
          )
          .sort((a, b) =>
            sort === "title"
              ? a.title.localeCompare(b.title)
              : b.timestamp - a.timestamp,
          );
        const pageIndex = Math.max(
          1,
          Math.min(
            Number(url.searchParams.get("page")) || 1,
            Math.max(1, Math.ceil(matched.length / 50)),
          ),
        );
        return route.fulfill({
          json: {
            items: matched.slice((pageIndex - 1) * 50, pageIndex * 50),
            total: matched.length,
            library_total: total,
            page: pageIndex,
            limit: 50,
            categories: ["Fixtures"],
            sampled_at: new Date().toISOString(),
          },
        });
      }
      if (path === "/torrents")
        throw new Error("Projection UI fetched the full legacy library");
      if (path === "/flow/tray")
        return route.fulfill({
          json: {
            server_state: "RUNNING",
            download_rate: 0,
            buffer_seconds: 0,
            peer_count: 0,
          },
        });
      if (path === "/flow/network")
        return route.fulfill({
          json: {
            state: "ADDRESS_READY",
            connectivity: "ONLINE",
            addresses: [],
            checked_at: "2026-09-30T00:00:00Z",
            changed_at: "2026-09-30T00:00:00Z",
            next_check_seconds: 15,
            reannounce_count: 0,
          },
        });
      return route.continue();
    });
    const started = Date.now();
    await page.goto("/#/torrents");
    await expect(
      page.getByRole("heading", {
        name: `Fixture ${String(total - 1).padStart(4, "0")}`,
      }),
    ).toBeVisible();
    const count = Math.min(total, 50);
    await expect(page.getByRole("article")).toHaveCount(count);
    const measurement = {
      fixture_count: library.length,
      rendered_cards: count,
      ready_ms: Date.now() - started,
      dom_nodes: await page.locator("*").count(),
      browser: info.project.name || "chromium",
      mode: process.env.FLOW_LIBRARY_BASELINE ? "baseline" : "paged",
    };
    const filename = info.outputPath("library-measurement.json");
    await writeFile(filename, JSON.stringify(measurement, null, 2));
    await info.attach("library-measurement", {
      path: filename,
      contentType: "application/json",
    });
    if (total > 50) {
      await page
        .getByRole("button", { name: "Next page", exact: true })
        .click();
      await expect(
        page.getByRole("heading", {
          name: `Fixture ${String(total - 51).padStart(4, "0")}`,
        }),
      ).toBeVisible();
      await expect(page.getByRole("article")).toHaveCount(50);
    }
    if (total === 200 && !process.env.FLOW_LIBRARY_BASELINE) {
      await page
        .getByRole("combobox", { name: "View", exact: true })
        .selectOption("list");
      await page
        .getByRole("combobox", { name: "Category", exact: true })
        .selectOption("category:Fixtures");
      await page
        .getByRole("combobox", { name: "Sort by", exact: true })
        .selectOption("title");
      await expect(
        page.getByRole("heading", { name: "Fixture 0000", exact: true }),
      ).toBeVisible();
      await page
        .getByRole("button", { name: "Next page", exact: true })
        .click();
      await expect(
        page.getByRole("heading", { name: "Fixture 0050", exact: true }),
      ).toBeVisible();
      await page.evaluate(() => window.scrollTo(0, 900));
      await expect
        .poll(() =>
          page.evaluate(() => sessionStorage.getItem("flow.library.scroll")),
        )
        .toBe("900");
      await page.reload();
      await expect(page.getByRole("article")).toHaveCount(50);
      await expect(
        page.getByRole("combobox", { name: "View", exact: true }),
      ).toHaveValue("list");
      await expect(
        page.getByRole("combobox", { name: "Category", exact: true }),
      ).toHaveValue("category:Fixtures");
      await expect(
        page.getByRole("combobox", { name: "Sort by", exact: true }),
      ).toHaveValue("title");
      await expect(
        page.getByRole("heading", { name: "Fixture 0050", exact: true }),
      ).toBeAttached();
      await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(900);
      await page
        .getByRole("textbox", { name: "Search", exact: true })
        .fill("Fixture 0060");
      await expect(page.getByRole("article")).toHaveCount(1);
      await page.reload();
      await expect(
        page.getByRole("textbox", { name: "Search", exact: true }),
      ).toHaveValue("Fixture 0060");
      await expect(
        page.getByRole("heading", { name: "Fixture 0060", exact: true }),
      ).toBeVisible();
      // The restored view makes extra intentional requests; keep the following
      // polling bound relative to the next search rather than those interactions.
      libraryRequests = 0;
    }
    await page
      .getByRole("textbox", { name: "Search", exact: true })
      .fill("Fixture 0000");
    await expect(page.getByRole("article")).toHaveCount(1);
    await expect(
      page.getByRole("heading", { name: "Fixture 0000" }),
    ).toBeVisible();
    await page
      .getByRole("textbox", { name: "Search", exact: true })
      .fill("absent fixture");
    await expect(page.getByRole("article")).toHaveCount(0);
    if (!process.env.FLOW_LIBRARY_BASELINE) {
      await page.waitForTimeout(6000);
      expect(libraryRequests).toBeGreaterThanOrEqual(2);
      expect(libraryRequests).toBeLessThanOrEqual(6);
    }
  });
