import { test, expect } from "@playwright/test";
import { gzipSync } from "node:zlib";
import { writeFile } from "node:fs/promises";
import { upgradeFixture } from "./upgrade-fixture";

for (const repeat of [1, 2, 3])
  test(`production dashboard lab repeat ${repeat}`, async ({ page }, info) => {
    await upgradeFixture(page, 5000);
    const session = await page.context().newCDPSession(page);
    await session.send("Network.enable");
    await session.send("Network.setCacheDisabled", { cacheDisabled: true });
    await session.send("Emulation.setCPUThrottlingRate", { rate: 4 });
    await session.send("Network.emulateNetworkConditions", {
      offline: false,
      latency: 40,
      downloadThroughput: 1_250_000,
      uploadThroughput: 625_000,
    });
    await page.addInitScript(() => {
      localStorage.setItem("i18nextLng", "en");
      const measurements = { lcp_ms: 0, cls: 0, shifts: [] as unknown[] };
      Object.assign(window, { flowLab: measurements });
      new PerformanceObserver((list) => {
        for (const entry of list.getEntries())
          measurements.lcp_ms = entry.startTime;
      }).observe({ type: "largest-contentful-paint", buffered: true });
      new PerformanceObserver((list) => {
        for (const entry of list.getEntries())
          if (
            !(entry as PerformanceEntry & { hadRecentInput: boolean })
              .hadRecentInput
          ) {
            measurements.cls += (
              entry as PerformanceEntry & { value: number }
            ).value;
            measurements.shifts.push({
              time: entry.startTime,
              value: (entry as PerformanceEntry & { value: number }).value,
              sources: (
                entry as PerformanceEntry & {
                  sources: {
                    node: Element;
                    previousRect: DOMRectReadOnly;
                    currentRect: DOMRectReadOnly;
                  }[];
                }
              ).sources.map((source) => ({
                tag: source.node?.tagName,
                class: source.node?.className,
                previous: source.previousRect.toJSON(),
                current: source.currentRect.toJSON(),
              })),
            });
          }
      }).observe({ type: "layout-shift", buffered: true });
    });
    const scripts = new Map<string, Promise<number>>();
    page.on("response", (response) => {
      if (new URL(response.url()).pathname.endsWith(".js"))
        scripts.set(
          response.url(),
          response.body().then((bytes) => gzipSync(bytes).length),
        );
    });
    await page.goto("/");
    await expect(
      page.getByRole("heading", { name: "Ready when you are" }),
    ).toBeVisible();
    await page.waitForTimeout(800);
    const initialJsGzipBytes = (await Promise.all(scripts.values())).reduce(
      (sum, size) => sum + size,
      0,
    );
    const observed = await page.evaluate(
      () =>
        (window as unknown as { flowLab: { lcp_ms: number; cls: number } })
          .flowLab,
    );
    // Local input feedback is measured to its next painted frame, not API search
    // completion or real-phone INP. Preserve the observation and lab conditions.
    await page.locator('a[href="#/torrents"]:visible').first().click();
    await expect(page.getByRole("article")).toHaveCount(50);
    const interactionMs = await page
      .getByRole("textbox", { name: "Search", exact: true })
      .evaluate(
        (input) =>
          new Promise<number>((resolve) => {
            const started = performance.now();
            const field = input as HTMLInputElement;
            const setter = Object.getOwnPropertyDescriptor(
              HTMLInputElement.prototype,
              "value",
            )!.set!;
            setter.call(field, "Fixture");
            field.dispatchEvent(new Event("input", { bubbles: true }));
            requestAnimationFrame(() =>
              requestAnimationFrame(() => resolve(performance.now() - started)),
            );
          }),
      );
    const report = {
      repeat,
      lab_only: true,
      cpu_slowdown: 4,
      latency_ms: 40,
      download_bytes_per_second: 1_250_000,
      viewport: [412, 915],
      library_entries: 5000,
      initial_js_gzip_bytes: initialJsGzipBytes,
      ...observed,
      input_paint_ms: interactionMs,
    };
    await writeFile(
      info.outputPath("measurement.json"),
      JSON.stringify(report, null, 2),
    );
    expect(initialJsGzipBytes).toBeLessThanOrEqual(200 * 1024);
    expect(observed.lcp_ms).toBeGreaterThan(0);
    expect(observed.lcp_ms).toBeLessThanOrEqual(2500);
    expect(observed.cls).toBeLessThanOrEqual(0.1);
    expect(interactionMs).toBeLessThanOrEqual(200);
  });
