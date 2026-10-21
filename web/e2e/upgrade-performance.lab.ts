import { test, expect } from "@playwright/test";
import { gzipSync } from "node:zlib";
import { writeFile } from "node:fs/promises";
import { upgradeFixture } from "./upgrade-fixture";

for (const repeat of [1, 2, 3])
  test(`production dashboard lab repeat ${repeat}`, async ({ page }, info) => {
    await upgradeFixture(page, 5000);
    await page.route("**/flow/status/**", (route) =>
      route.fulfill({
        json: {
          hash: new URL(route.request().url()).pathname.split("/").at(-1),
          sampled_at: new Date().toISOString(),
          sessions: [],
          timeline: { source: "server", dropped: 0, events: [] },
        },
      }),
    );
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
      const measurements = {
        lcp_ms: 0,
        cls: 0,
        shifts: [] as unknown[],
        interactions: [] as {
          duration: number;
          id: number;
          name: string;
          target: string;
        }[],
        long_tasks: [] as number[],
      };
      Object.assign(window, { flowLab: measurements });
      if (PerformanceObserver.supportedEntryTypes.includes("event"))
        new PerformanceObserver((list) => {
          for (const entry of list.getEntries()) {
            const event = entry as PerformanceEntry & {
              interactionId?: number;
              target?: Element;
            };
            if (event.interactionId && measurements.interactions.length < 128)
              measurements.interactions.push({
                duration: event.duration,
                id: event.interactionId,
                name: event.name,
                target: event.target?.tagName || "unknown",
              });
          }
        }).observe({
          type: "event",
          buffered: true,
          durationThreshold: 16,
        } as PerformanceObserverInit);
      if (PerformanceObserver.supportedEntryTypes.includes("longtask"))
        new PerformanceObserver((list) => {
          for (const entry of list.getEntries())
            if (measurements.long_tasks.length < 128)
              measurements.long_tasks.push(entry.duration);
        }).observe({ type: "longtask", buffered: true });
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
    const observed = await page.evaluate(() => {
      const data = (
        window as unknown as { flowLab: { lcp_ms: number; cls: number } }
      ).flowLab;
      return { lcp_ms: data.lcp_ms, cls: data.cls };
    });
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
    // Trusted pointer/keyboard events exercise production UI work. Event Timing
    // is a local observation, not field INP or Android acceptance.
    const diagnosticStart = await page.evaluate(() => performance.now());
    await page
      .getByRole("button", { name: "Diagnostics", exact: true })
      .first()
      .click();
    const dialog = page.getByRole("dialog");
    await expect(
      dialog.getByRole("heading", { name: "Playback overview", exact: true }),
    ).toBeVisible();
    const diagnosticsReadyMs =
      (await page.evaluate(() => performance.now())) - diagnosticStart;
    await dialog.getByText("Player measurements", { exact: true }).click();
    const playerSamples = Array.from({ length: 128 }, (_, i) => ({
      elapsed_ms: (i + 1) * 5000,
      recorded_at_ms: 1792620000000 + i * 5000,
      startup_ms: null,
      rebuffer_events: 0,
      rebuffer_ms: 0,
      seek_events: 0,
      seek_buffer_ms: 0,
      paused_ms: 0,
      play_ms: i * 5000,
      buffered_ms: 10000,
      dropped_frames: 0,
      bandwidth_bps: 1000000,
      state: 3,
      playing: true,
      session_id: null,
    }));
    // Start at the browser's real change event, excluding automation protocol
    // and locator polling latency from the file-import responsiveness budget.
    await dialog.getByLabel("Import player JSON report").evaluate((input) => {
      input.addEventListener(
        "change",
        () => {
          const start = performance.now();
          const observer = new MutationObserver(() => {
            if (
              [...document.querySelectorAll("button")].some(
                (button) => button.textContent === "Clear imported report",
              )
            ) {
              observer.disconnect();
              requestAnimationFrame(() =>
                requestAnimationFrame(() =>
                  Object.assign(window, {
                    flowImportPaintMs: performance.now() - start,
                  }),
                ),
              );
            }
          });
          observer.observe(document.body, { childList: true, subtree: true });
        },
        { once: true, capture: true },
      );
    });
    await dialog
      .getByLabel("Import player JSON report")
      .setInputFiles({
        name: "generated-player-report.json",
        mimeType: "application/json",
        buffer: Buffer.from(
          JSON.stringify({
            schema_version: 1,
            source: "flow-player-media3",
            clock: "player-elapsed-and-wall; server-clock-alignment-unknown",
            samples: playerSamples,
          }),
        ),
      });
    await expect(
      dialog.getByRole("button", {
        name: "Clear imported report",
        exact: true,
      }),
    ).toBeVisible();
    await page.waitForFunction(
      () =>
        typeof (window as unknown as { flowImportPaintMs?: number })
          .flowImportPaintMs === "number",
    );
    const playerImportMs = await page.evaluate(
      () =>
        (window as unknown as { flowImportPaintMs: number }).flowImportPaintMs,
    );
    await dialog.getByRole("button", { name: "Close", exact: true }).click();
    const settingsStart = await page.evaluate(() => performance.now());
    await page.locator('a[href="#/settings"]:visible').first().click();
    await expect(
      page.getByRole("tab", { name: "General", exact: true }),
    ).toBeVisible();
    const settingsReadyMs =
      (await page.evaluate(() => performance.now())) - settingsStart;
    await page.getByRole("tab", { name: "Flow", exact: true }).click();
    await page.getByRole("tab", { name: "Advanced", exact: true }).click();
    await expect(
      page.getByRole("heading", {
        name: "Maintenance and recovery",
        exact: true,
      }),
    ).toBeVisible();
    await page.waitForTimeout(300);
    const interactionEvidence = await page.evaluate(() => {
      const data = (
        window as unknown as {
          flowLab: {
            interactions: { duration: number; id: number }[];
            long_tasks: number[];
          };
        }
      ).flowLab;
      return {
        event_timing: data.interactions,
        long_tasks_ms: data.long_tasks,
      };
    });
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
      diagnostics_ready_ms: diagnosticsReadyMs,
      settings_ready_ms: settingsReadyMs,
      player_import_ms: playerImportMs,
      ...interactionEvidence,
      event_timing_is_field_inp: false,
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
    expect(playerImportMs).toBeLessThanOrEqual(200);
    expect(diagnosticsReadyMs).toBeLessThanOrEqual(2500);
    expect(settingsReadyMs).toBeLessThanOrEqual(2500);
    for (const event of interactionEvidence.event_timing)
      expect(event.duration).toBeLessThanOrEqual(200);
  });
