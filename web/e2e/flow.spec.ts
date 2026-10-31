import { test, expect, type Page } from "@playwright/test";
import { writeFile } from "node:fs/promises";
import settings from "./settings.json" with { type: "json" };

for (const fixture of [
  { tab: "Security", paths: ["/waf", "/ssl/status"] },
  { tab: "Integrations", paths: ["/gst/settings"] },
  { tab: "Advanced", paths: ["/storage/settings"] },
]) {
  test(`${fixture.tab} detail reads abort on backgrounding and tab exit`, async ({
    page,
  }) => {
    await mockServer(page);
    let release!: () => void;
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    const requested: Record<string, number> = {};
    const failed: Record<string, number> = {};
    page.on("requestfailed", (request) => {
      const path = new URL(request.url()).pathname;
      if (fixture.paths.includes(path)) failed[path] = (failed[path] ?? 0) + 1;
    });
    for (const path of fixture.paths) {
      await page.route(`**${path}`, async (route) => {
        requested[path] = (requested[path] ?? 0) + 1;
        await held;
        await route.fulfill({ json: {} }).catch(() => {});
      });
    }
    try {
      await page.goto("/#/settings");
      await page.getByRole("tab", { name: fixture.tab, exact: true }).click();
      for (const path of fixture.paths)
        await expect.poll(() => requested[path]).toBe(1);
      await page.evaluate(() => {
        Object.defineProperty(document, "hidden", {
          configurable: true,
          value: true,
        });
        document.dispatchEvent(new Event("visibilitychange"));
      });
      for (const path of fixture.paths)
        await expect.poll(() => failed[path]).toBe(1);
      await page.evaluate(() => {
        Object.defineProperty(document, "hidden", {
          configurable: true,
          value: false,
        });
        document.dispatchEvent(new Event("visibilitychange"));
      });
      for (const path of fixture.paths)
        await expect.poll(() => requested[path]).toBe(2);
      await page.getByRole("tab", { name: "General", exact: true }).click();
      for (const path of fixture.paths)
        await expect.poll(() => failed[path]).toBe(2);
      await expect(page.getByRole("dialog")).toHaveCount(0);
    } finally {
      release();
    }
  });
}

test("certificate drafts survive tabs, block navigation and use revision checks", async ({
  page,
}) => {
  await mockServer(page);
  const status = {
    enabled: true,
    port: "8091",
    http_port: "8090",
    http_enabled: true,
    force_https: false,
    http_media: false,
    read_only: false,
    cert_from_flags: false,
    revision: "fixture-revision-1",
    cert: { source: "self-signed", trusted: false },
  };
  await page.route("**/ssl/status", (route) => route.fulfill({ json: status }));
  const changes: { revision?: string; body: unknown }[] = [];
  await page.route("**/ssl/paths", (route) => {
    changes.push({
      revision: route.request().headers()["if-match"],
      body: route.request().postDataJSON(),
    });
    return route.fulfill({ json: status });
  });
  await page.goto("/#/settings");
  await page.getByRole("tab", { name: "Security", exact: true }).click();
  await page
    .getByRole("textbox", { name: "Certificate path on server", exact: true })
    .fill("fixture.crt");
  await page
    .getByRole("textbox", { name: "Private key path on server", exact: true })
    .fill("fixture.key");
  await page.getByRole("tab", { name: "General", exact: true }).click();
  await page.getByRole("tab", { name: "Security", exact: true }).click();
  await expect(
    page.getByRole("textbox", {
      name: "Certificate path on server",
      exact: true,
    }),
  ).toHaveValue("fixture.crt");
  await page
    .getByRole("link", { name: "Dashboard", exact: true })
    .first()
    .click();
  await expect(
    page.getByRole("dialog", { name: "Unsaved changes" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Continue editing" }).click();
  await page
    .getByRole("button", { name: "Use existing files", exact: true })
    .click();
  await expect(
    page.getByRole("textbox", {
      name: "Certificate path on server",
      exact: true,
    }),
  ).toHaveValue("");
  expect(changes).toEqual([
    {
      revision: "fixture-revision-1",
      body: { cert: "fixture.crt", key: "fixture.key" },
    },
  ]);
  await page
    .getByRole("link", { name: "Dashboard", exact: true })
    .first()
    .click();
  await expect(
    page.getByRole("heading", { name: "Dashboard", exact: true }),
  ).toBeVisible();
});

test("legacy zero-cache settings remain accessible for explicit repair", async ({
  page,
}) => {
  await mockServer(page);
  await page.route("**/settings", (route) => {
    if (new URL(route.request().url()).pathname !== "/settings")
      return route.fallback();
    if (route.request().postDataJSON().action === "state")
      return route.fulfill({
        json: {
          revision: "fixture-zero",
          saved: { ...settings, CacheSize: 0 },
          effective: { ...settings, CacheSize: 0 },
          recovery: { source: "settings", schema_version: 1 },
          data_path: "fixture-state",
          executable: "fixture-server",
        },
      });
    return route.fulfill({ json: { ...settings, CacheSize: 0 } });
  });
  await page.goto("/#/settings");
  await expect(
    page.getByRole("heading", { name: "Settings", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("spinbutton", { name: "Cache Size", exact: true }),
  ).toHaveValue("0");
  await expect(
    page.getByText("The server could not complete the request.", {
      exact: true,
    }),
  ).toHaveCount(0);
});

test("active playback remains visible when Flow metrics are unavailable", async ({
  page,
}) => {
  await mockServer(page, { active: true });
  await page.route("**/torrents", (route) =>
    route.fulfill({ json: [{ ...torrent, active_readers: 1 }] }),
  );
  await page.route("**/flow/status/*", (route) =>
    route.fulfill({ json: { hash, sessions: [] } }),
  );
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Example series" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "No current playback" }),
  ).toHaveCount(0);
  await expect(page.getByText("Unknown", { exact: true })).toBeVisible();
});

test("polling is deduplicated, bounded and stops when hidden", async ({
  page,
}, info) => {
  test.setTimeout(45000);
  await mockServer(page, { active: true });
  await page.route("**/torrents", (route) =>
    route.fulfill({ json: [{ ...torrent, active_readers: 1 }] }),
  );
  const counts: Record<string, number> = {};
  const bytes: Record<string, number> = {};
  page.on("response", async (response) => {
    const path = new URL(response.url()).pathname;
    if (
      path.startsWith("/flow/") ||
      path === "/torrents" ||
      path === "/runtime/status"
    ) {
      try {
        bytes[path] = (bytes[path] || 0) + (await response.body()).length;
      } catch {
        /* cancelled in-flight response */
      }
    }
  });
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (
      path.startsWith("/flow/") ||
      path === "/torrents" ||
      path === "/runtime/status"
    )
      counts[path] = (counts[path] || 0) + 1;
  });
  await page.goto("/");
  await expect(page.getByText("Healthy", { exact: true })).toBeVisible();
  await page.waitForTimeout(11000);
  expect(counts["/flow/network"]).toBeLessThanOrEqual(4);
  expect(counts["/runtime/status"]).toBeLessThanOrEqual(4);
  expect(counts["/flow/status/" + hash]).toBeGreaterThanOrEqual(8);
  expect(counts["/flow/status/" + hash]).toBeLessThanOrEqual(14);
  expect(counts["/torrents"]).toBeLessThanOrEqual(14);
  const measurement = info.outputPath("polling-measurement.json");
  await writeFile(
    measurement,
    JSON.stringify(
      { visible_seconds: 11, counts, bytes, synthetic_payloads: true },
      null,
      2,
    ),
  );
  await info.attach("polling-measurement", {
    path: measurement,
    contentType: "application/json",
  });
  await page.evaluate(() => {
    Object.defineProperty(document, "hidden", {
      configurable: true,
      get: () => true,
    });
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await page.waitForTimeout(500);
  const before = { ...counts };
  await page.waitForTimeout(6000);
  expect(counts).toEqual(before);
  await page.evaluate(() => {
    Object.defineProperty(document, "hidden", {
      configurable: true,
      get: () => false,
    });
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await page.waitForTimeout(2000);
  expect(counts["/flow/status/" + hash]).toBeGreaterThan(
    before["/flow/status/" + hash]!,
  );
});
const hash = "0123456789012345678901234567890123456789";
test("next episode warmup requires a chosen file and handles rejected changes", async ({
  page,
}) => {
  await mockServer(page, { active: true });
  await page.route("**/flow/status/*", (route) =>
    route.fulfill({
      json: {
        hash,
        sampled_at: new Date().toISOString(),
        sessions: [
          {
            group: "phone",
            buffer_sampled_at_ms: Date.now(),
            file_index: 1,
            state: "PLAYING",
            active_readers: 1,
            buffer_ahead_seconds: 40,
            buffer_ahead_bytes: 1000,
            playback_consumption_rate: 500,
            download_rate: 600,
            sustainability_ratio: 1.2,
            piece_wait_p95_ms: 0,
            seek_count: 0,
            seek_recovery_ms: 0,
            buffer_warning: false,
          },
        ],
        next_episode_warmup: {
          enabled: true,
          current_file_index: 1,
          file_index: 0,
          automatic: false,
          state: "idle",
          reason: "MANUAL_SELECTION_REQUIRED",
          budget_bytes: 0,
          verified_bytes: 0,
        },
      },
    }),
  );
  await page.route("**/torrents", (route) =>
    route.request().postDataJSON().action === "get"
      ? route.fulfill({
          json: {
            ...torrent,
            file_stats: [
              ...torrent.file_stats,
              { id: 2, path: "Season 01/Show.S01E02.mkv", length: 1000 },
              { id: 3, path: "readme.txt", length: 20 },
            ],
          },
        })
      : route.fallback(),
  );
  const actions: unknown[] = [];
  await page.route("**/flow/warmup/*", (route) => {
    actions.push(route.request().postDataJSON());
    return route.fulfill({
      status: 409,
      json: { error: "fixture rejected change" },
    });
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Diagnostics", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await expect(
    dialog.getByRole("heading", { name: "Next episode warmup", exact: true }),
  ).toBeVisible();
  await dialog
    .getByRole("button", { name: "Choose file", exact: true })
    .click();
  const choice = dialog.getByRole("combobox", {
    name: "Choose file",
    exact: true,
  });
  await expect(choice.locator("option")).toHaveCount(2);
  await choice.selectOption("2");
  await dialog
    .getByRole("button", { name: "Warm selected file", exact: true })
    .click();
  await expect(dialog.getByRole("alert")).toBeVisible();
  expect(actions).toEqual([{ file_index: 2, action: "select" }]);
  await dialog
    .getByRole("button", { name: "Cancel warmup", exact: true })
    .click();
  await expect.poll(() => actions.length).toBe(2);
  expect(actions[1]).toEqual({ file_index: 0, action: "cancel" });
});

test("incident timeline and explicit player import preserve unknown evidence", async ({
  page,
}) => {
  await mockServer(page, { active: true });
  const session = "ABCDEFGHIJKLMNOPQRSTUV";
  let uploads = 0;
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/flow/"))
      uploads++;
  });
  await page.route("**/flow/status/*", (route) =>
    route.fulfill({
      json: {
        hash,
        sampled_at: "2026-10-10T00:00:00Z",
        sessions: [
          {
            group: "phone",
            session_id: session,
            file_index: 1,
            state: "PLAYING",
            active_readers: 1,
            buffer_ahead_seconds: 40,
            buffer_ahead_bytes: 1000,
            playback_consumption_rate: 500,
            download_rate: 600,
            sustainability_ratio: 1.2,
            piece_wait_p95_ms: 0,
            seek_count: 0,
            seek_recovery_ms: 0,
            buffer_warning: false,
          },
        ],
        timeline: {
          source: "server",
          dropped: 12,
          events: [
            {
              type: "seek",
              elapsed_ms: 1000,
              operation_ms: 250,
              file: 1,
              time: "2026-10-10T00:00:00Z",
            },
          ],
        },
      },
    }),
  );
  await page.goto("/");
  await page.getByRole("button", { name: "Diagnostics", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await expect(
    dialog.getByRole("heading", { name: "Incident timeline" }),
  ).toBeVisible();
  await expect(dialog.getByText("+1.00 s", { exact: true })).toBeVisible();
  await dialog.getByText("Player measurements", { exact: true }).click();
  const report = {
    schema_version: 1,
    source: "flow-player-media3",
    clock: "player-elapsed-and-wall; server-clock-alignment-unknown",
    samples: [
      {
        elapsed_ms: 3000,
        recorded_at_ms: 1791590400000,
        startup_ms: null,
        rebuffer_events: 2,
        rebuffer_ms: 1200,
        seek_events: 1,
        seek_buffer_ms: 250,
        paused_ms: 5000,
        play_ms: 2000,
        buffered_ms: 10000,
        dropped_frames: 0,
        bandwidth_bps: 1000000,
        state: 3,
        playing: true,
        session_id: session,
      },
    ],
  };
  await dialog.getByLabel("Import player JSON report").setInputFiles({
    name: "report.json",
    mimeType: "application/json",
    buffer: Buffer.from(JSON.stringify(report)),
  });
  await expect(
    dialog.getByText("Report matches an observed session.", { exact: true }),
  ).toBeVisible();
  await expect(
    dialog
      .getByText("Player measurements", { exact: true })
      .locator("..")
      .locator("dd")
      .filter({ hasText: /^—$/ }),
  ).toHaveCount(1);
  await dialog.getByRole("button", { name: "Clear imported report" }).click();
  await dialog.getByLabel("Import player JSON report").setInputFiles({
    name: "invalid.json",
    mimeType: "application/json",
    buffer: Buffer.from(
      JSON.stringify({ ...report, url: "https://private.invalid/token" }),
    ),
  });
  await expect(dialog.getByText(/private.invalid/)).toHaveCount(0);
  await expect(
    dialog.getByText("Invalid or oversized player report.", { exact: true }),
  ).toBeVisible();
  expect(uploads).toBe(0);
});

test("settings search can navigate to a field group without discarding drafts", async ({
  page,
}) => {
  await mockServer(page);
  await page.goto("/#/settings");
  await page
    .getByRole("spinbutton", { name: "Cache Size", exact: true })
    .fill("128");
  await page.getByRole("searchbox").fill("Cache Size");
  await page.getByRole("button", { name: "General", exact: true }).click();
  await expect(page.getByRole("searchbox")).toHaveValue("");
  await expect(
    page.getByRole("spinbutton", { name: "Cache Size", exact: true }),
  ).toBeFocused();
  await expect(
    page.getByRole("spinbutton", { name: "Cache Size", exact: true }),
  ).toHaveValue("128");
});
test("trace history is fetched only while diagnostics are open", async ({
  page,
}) => {
  await mockServer(page, { active: true });
  let compactRequests = 0;
  let diagnosticRequests = 0;
  await page.route("**/flow/status/*", (route) => {
    const compact =
      new URL(route.request().url()).searchParams.get("traces") === "false";
    if (compact) compactRequests++;
    else diagnosticRequests++;
    return route.fulfill({
      json: {
        hash,
        sampled_at: new Date().toISOString(),
        sessions: [
          {
            group: "phone",
            buffer_sampled_at_ms: Date.now(),
            file_index: 1,
            state: "PLAYING",
            active_readers: 1,
            buffer_ahead_seconds: 40,
            buffer_ahead_bytes: 1000,
            playback_consumption_rate: 500,
            download_rate: 600,
            sustainability_ratio: 1.2,
            piece_wait_p95_ms: 0,
            seek_count: 0,
            seek_recovery_ms: 0,
            buffer_warning: false,
            ...(compact
              ? {}
              : { traces: [{ classification: "diagnostic-trace-fixture" }] }),
          },
        ],
      },
    });
  });
  await page.goto("/");
  await expect(page.getByText("Healthy", { exact: true })).toBeVisible();
  expect(compactRequests).toBeGreaterThan(0);
  expect(diagnosticRequests).toBe(0);
  await page.getByRole("button", { name: "Diagnostics", exact: true }).click();
  await page
    .getByRole("dialog")
    .getByText("Technical details", { exact: true })
    .click();
  await page
    .getByRole("dialog")
    .getByText("Range request traces", { exact: true })
    .click();
  await expect(
    page.getByRole("dialog").getByText(/diagnostic-trace-fixture/),
  ).toBeVisible();
  expect(diagnosticRequests).toBeGreaterThan(0);
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Close", exact: true })
    .click();
  await page.waitForTimeout(300);
  const previous = diagnosticRequests;
  await page.waitForTimeout(3500);
  expect(diagnosticRequests).toBe(previous);
});
for (const width of [375, 1440]) {
  test(`buffering evidence is readable at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 1000 });
    await mockServer(page, { active: true });
    await page.route("**/flow/status/*", (route) =>
      route.fulfill({
        json: {
          hash,
          sampled_at: new Date().toISOString(),
          startup: {
            state: "PLAYING",
            file_index: 1,
            reserve: {
              known: true,
              reason: "SUPPLY_DEFICIT",
              base_bytes: 33554432,
              target_bytes: 67108864,
              maximum_bytes: 67108864,
              limited: true,
              demand_rate: 12500000,
              delivery_rate: 10000000,
              deficit_rate: 2500000,
              forecast_seconds: 45,
            },
          },
          sessions: [
            {
              group: "phone",
              buffer_sampled_at_ms: Date.now(),
              file_index: 1,
              state: "PLAYING",
              active_readers: 1,
              buffer_ahead_seconds: 0,
              buffer_ahead_bytes: 0,
              playback_consumption_rate: 12500000,
              download_rate: 10000000,
              sustainability_ratio: 0.8,
              piece_wait_p95_ms: 500,
              seek_count: 0,
              seek_recovery_ms: 0,
              buffer_warning: true,
              supply: {
                known: true,
                state: "SUPPLY_DEFICIT",
                demand_rate: 12500000,
                delivery_rate: 10000000,
                deficit_rate: 2500000,
              },
              blocker: {
                known: true,
                piece: 0,
                piece_length: 16777216,
                blocks: 1024,
                finished: 1022,
                requested: 2,
                unrequested: 0,
                writing: 0,
                oldest_request_age_ms: 4500,
                suppliers: 6,
                reason: "WAIT_BLOCKS",
              },
            },
          ],
          sparse: {
            known: true,
            sampled_at_ms: Date.now(),
            stream: {
              known: true,
              first_dial_ms: 0,
              first_connect_ms: 10,
              first_handshake_ms: 25,
              first_payload_ms: 100,
              connect_duration_ms: 10,
              handshake_duration_ms: 15,
              useful_bytes: 1048576,
              redundant_bytes: 16384,
              obsolete_bytes: 0,
              recovery_requests: 1,
              recovery_suppressed: 0,
              active_hedges: 0,
              peak_hedges: 1,
              active_hedge_bytes: 0,
              budget_changes: 2,
              frontiers: 1,
              recovery_scan_truncated: false,
              service: [],
              request_policy: {
                enabled: true,
                sampled_peers: 2,
                qualified_peers: 1,
                capacity_limited_peers: 0,
                desired_requests: 16,
                queued_requests: 12,
                capacity_requests: 64,
                horizon_min_ms: 4100,
                horizon_max_ms: 4100,
                truncated: false,
              },
              recovery_scan: {
                age_ms: 200,
                eligible: 1,
                uncertain_request: 1,
                stale_service: 0,
                already_duplicated: 0,
                not_late: 0,
                receiving: 0,
              },
            },
          },
        },
      }),
    );
    await page.goto("/");
    await page
      .getByRole("button", { name: "Diagnostics", exact: true })
      .click();
    const dialog = page.getByRole("dialog");
    await expect(
      dialog.getByRole("heading", { name: "Buffering evidence" }),
    ).toBeVisible();
    await expect(dialog.getByText(/Verified delivery 80.0 Mbps/)).toBeVisible();
    await expect(dialog.getByText(/1022\/1024 blocks finished/)).toBeVisible();
    await expect(
      dialog.getByText(/Startup reserve: 32.0 → 64.0 MiB/),
    ).toBeVisible();
    await expect(
      dialog.getByText(/reserve reached the configured/),
    ).toBeVisible();
    await dialog
      .getByText(/Measured request horizons: 4100/)
      .scrollIntoViewIfNeeded();
    await expect(
      dialog.getByText(/Measured request horizons: 4100/),
    ).toBeVisible();
    await expect(
      dialog.getByText(/Last recovery scan: 1 eligible/),
    ).toBeVisible();
    expect(
      await dialog.evaluate(
        (element) => element.scrollWidth <= element.clientWidth + 1,
      ),
    ).toBe(true);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: `../.tools/buffering-ui-${width}.png`,
      fullPage: true,
    });
  });
}
const torrent = {
  hash,
  title: "Example series",
  stat: 3,
  category: "Series",
  torrent_size: 12345678,
  file_stats: [{ id: 1, path: "Season 01/Show.S01E01.mkv", length: 12345678 }],
};
async function mockServer(
  page: Page,
  { status = 200, active = false }: { status?: number; active?: boolean } = {},
) {
  const saves: unknown[] = [];
  let saved = settings;
  let revision = "fixture-revision-1";
  await page.route("**/*", async (route) => {
    const path = new URL(route.request().url()).pathname;
    const json = (data: unknown, code = 200) =>
      route.fulfill({
        status: code,
        contentType: "application/json",
        body: JSON.stringify(
          code === 503 && status === 503
            ? {
                ...(data as object),
                error: "torrent engine is starting",
                code: "ENGINE_STARTING",
              }
            : data,
        ),
      });
    if (path === "/settings/metadata/v1")
      return route.fulfill({
        status: 404,
        json: { error: "older fixture server" },
      });
    if (path === "/echo")
      return route.fulfill({ body: "MatriX.145.Flow-test" });
    // Existing scenarios deliberately exercise the legacy-server fallback.
    // Projection-specific scenarios override these routes with real new DTOs.
    if (path === "/flow/active" || path === "/flow/library")
      return json(
        { error: "endpoint unavailable on legacy fixture" },
        status === 200 ? 404 : status,
      );
    if (path === "/mediabase")
      return json({ base: new URL(route.request().url()).origin });
    if (path === "/ssl/status")
      return json({
        enabled: false,
        http_port: "8090",
        http_enabled: true,
        force_https: false,
        http_media: false,
        read_only: false,
        cert_from_flags: false,
        revision,
        cert: { source: "none", trusted: false },
      });
    if (path === "/flow/tray")
      return json(
        {
          server_state: status === 503 ? "STARTING" : "RUNNING",
          active_torrent: active ? "Example series" : "",
          download_rate: 0,
          buffer_seconds: 0,
          peer_count: 0,
        },
        status,
      );
    if (path === "/flow/network")
      return json(
        {
          state: "ADDRESS_READY",
          connectivity: "DEGRADED",
          addresses: ["192.168.1.5", "10.0.0.5"],
          checked_at: "2026-09-28T00:00:00Z",
          changed_at: "2026-09-28T00:00:00Z",
          next_check_seconds: 15,
          reannounce_count: 0,
        },
        status,
      );
    if (path === "/runtime/status")
      return json(
        {
          dlna_enabled: false,
          bonjour_enabled: false,
          friendly_name: "Flow test",
          webdav_enabled: false,
          webdav_path: "/dav",
          fuse_path: "",
          fuse_enabled: false,
          bt: {
            active_streams: active ? 1 : 0,
            torrent_count: 1,
            torrents: [torrent],
            raw_stat: "Example runtime",
          },
        },
        status,
      );
    if (path === "/settings") {
      const body = route.request().postDataJSON();
      if (body.action === "get") return json(saved);
      if (body.action === "state")
        return json({
          revision,
          saved,
          effective: saved,
          applying: false,
          recovery: { source: "settings", schema_version: 1 },
          data_path: "fixture-state",
          executable: "fixture-server",
        });
      if (body.action === "plan")
        return json({ restart_required: false, active_work: active });
      saves.push(body);
      if (body.action === "set") {
        saved = body.sets;
        revision = "fixture-revision-2";
      }
      return route.fulfill({ body: "" });
    }
    if (path === "/torrents") {
      const body = route.request().postDataJSON();
      if (body.action === "list") return json([torrent], status);
      if (body.action === "add" || body.action === "get") return json(torrent);
      return route.fulfill({ body: "" });
    }
    if (path === "/viewed")
      return json([{ hash, file_index: 1, timecode: 90 }]);
    if (path === "/flow/preparation")
      return json({ jobs: [], quota_bytes: 67108864, reserved_bytes: 0 });
    if (path.startsWith("/flow/sources/"))
      return json({ private: false, sources: [] });
    if (path === "/cache")
      return json({
        Capacity: 100,
        Filled: 50,
        PiecesCount: 4,
        Pieces: { 0: { Id: 0, Completed: true } },
        Readers: [],
      });
    if (path.startsWith("/flow/status/"))
      return json({
        hash,
        sampled_at: new Date().toISOString(),
        sessions: active
          ? [
              {
                group: "phone",
                buffer_sampled_at_ms: Date.now(),
                file_index: 1,
                state: "PLAYING",
                active_readers: 1,
                buffer_ahead_seconds: 40,
                buffer_ahead_bytes: 1000,
                playback_consumption_rate: 500,
                download_rate: 600,
                sustainability_ratio: 1.2,
                piece_wait_p95_ms: 0,
                seek_count: 0,
                seek_recovery_ms: 0,
                buffer_warning: false,
                connected_peers: 5,
              },
            ]
          : [],
        trackers: [],
      });
    if (path === "/search/")
      return json([
        {
          Title: "Example series",
          Magnet: "magnet:?xt=urn:btih:" + hash,
          Seed: 10,
          Peer: 2,
          Size: "1 GiB",
        },
      ]);
    if (path === "/stream")
      return json({ ...torrent, preload_size: 100, preloaded_bytes: 100 });
    if (path === "/waf")
      return json({
        whitelist: "",
        blacklist: "",
        referers: "",
        default_referers: [],
        default_referers_enabled: true,
        read_only: false,
        warnings: [],
      });
    if (path === "/gst/settings") return json({ built_in: false });
    if (path === "/storage/settings")
      return json({ settings: "json", viewed: "bbolt" });
    return route.continue();
  });
  return saves;
}
for (const viewport of [
  { width: 375, height: 812 },
  { width: 412, height: 915 },
  { width: 768, height: 1024 },
  { width: 1440, height: 900 },
  { width: 2560, height: 1440 },
  { width: 915, height: 412 },
]) {
  test(`dashboard and navigation fit ${viewport.width}x${viewport.height}`, async ({
    page,
  }) => {
    await page.setViewportSize(viewport);
    await mockServer(page);
    await page.goto("/");
    await expect(
      page.getByRole("heading", { name: "Dashboard", exact: true }),
    ).toBeVisible();
    await expect(page.getByText("Degraded", { exact: true })).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page
      .getByRole("link", { name: "Torrents", exact: true })
      .last()
      .click();
    await expect(
      page.getByRole("heading", { name: "Example series" }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  });
}
test("explicit engine-starting 503 remains starting rather than offline", async ({
  page,
}) => {
  await mockServer(page, { status: 503 });
  await page.goto("/");
  await expect(page.getByText("Starting engine…").first()).toBeVisible();
});
test("settings cancel is inert, apply preserves Flow and unknown fields", async ({
  page,
}) => {
  const saves = await mockServer(page, { active: true });
  await page.goto("/#/settings");
  await page.getByRole("button", { name: "512 MiB", exact: true }).click();
  await page
    .getByRole("button", { name: "Apply settings", exact: true })
    .click();
  await expect(
    page.getByText(
      "These changes apply without restarting the torrent engine.",
    ),
  ).toBeVisible();
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  expect(saves).toHaveLength(0);
  await page
    .getByRole("button", { name: "Apply settings", exact: true })
    .click();
  await page.getByRole("button", { name: "Confirm", exact: true }).click();
  await expect.poll(() => saves.length).toBe(1);
  expect(saves[0]).toMatchObject({
    action: "set",
    sets: {
      CacheSize: 536870912,
      Flow: { Enabled: true, SwarmCustom: { ConnectionSpeed: 0 } },
    },
  });
});
test("quoted disk cache paths show a correction before any settings save", async ({
  page,
}) => {
  const saves = await mockServer(page);
  let plans = 0;
  page.on("request", (request) => {
    if (
      new URL(request.url()).pathname === "/settings" &&
      request.postDataJSON()?.action === "plan"
    )
      plans++;
  });
  await page.goto("/#/settings");
  await page.locator("#UseDisk").check();
  const path = page.locator("#TorrentsSavePath");
  await path.fill('"C:\\cache folder"');
  await page
    .getByRole("button", { name: "Apply settings", exact: true })
    .click();
  await expect(
    page.getByText(
      "Enter the folder path without surrounding quotation marks.",
      {
        exact: true,
      },
    ),
  ).toBeVisible();
  await expect(path).toBeFocused();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  expect(plans).toBe(0);
  expect(saves).toHaveLength(0);
  await path.fill("C:\\cache folder");
  await page
    .getByRole("button", { name: "Apply settings", exact: true })
    .click();
  await page.getByRole("button", { name: "Confirm", exact: true }).click();
  await expect.poll(() => saves.length).toBe(1);
  expect(saves[0]).toMatchObject({
    action: "set",
    sets: { UseDisk: true, TorrentsSavePath: "C:\\cache folder" },
  });
});

test("settings polling preserves draft, blocks a known conflict and reloads its revision", async ({
  page,
}) => {
  await mockServer(page);
  let revision = "draft-base";
  let submitted = "";
  await page.route("**/settings", (route) => {
    const body = route.request().postDataJSON();
    if (body.action === "state")
      return route.fulfill({
        json: {
          revision,
          saved: settings,
          effective: settings,
          recovery: { source: "settings", schema_version: 1 },
          data_path: "fixture-state",
          executable: "fixture-server",
        },
      });
    if (body.action === "set") {
      submitted = body.revision;
      return route.fulfill({
        status: 409,
        json: { error: "settings changed; reload before applying your draft" },
      });
    }
    return route.fallback();
  });
  await page.goto("/#/settings");
  await page.getByRole("button", { name: "512 MiB", exact: true }).click();
  revision = "external-change";
  await expect(page.getByText("external-change", { exact: true })).toHaveCount(
    1,
    {
      timeout: 10000,
    },
  );
  await expect(
    page.getByRole("spinbutton", { name: "Cache Size", exact: true }),
  ).toHaveValue("512");
  await expect(
    page.getByRole("button", { name: "Apply settings", exact: true }),
  ).toBeDisabled();
  expect(submitted).toBe("");
  await expect(
    page
      .getByRole("alert")
      .filter({ hasText: "Settings changed on the server" }),
  ).toBeVisible();
  await expect(
    page.getByRole("spinbutton", { name: "Cache Size", exact: true }),
  ).toHaveValue("512");
  await page
    .getByRole("button", {
      name: "Discard draft and reload saved settings",
      exact: true,
    })
    .click();
  await expect(
    page.getByRole("spinbutton", { name: "Cache Size", exact: true }),
  ).toHaveValue(String(settings.CacheSize / 1048576));
  await page.getByRole("button", { name: "512 MiB", exact: true }).click();
  await page
    .getByRole("button", { name: "Apply settings", exact: true })
    .click();
  await page.getByRole("button", { name: "Confirm", exact: true }).click();
  await expect.poll(() => submitted).toBe("external-change");
  await expect(page.getByRole("dialog").getByRole("alert")).toBeVisible();
});

test("settings advanced view and keyboard groups preserve hidden draft fields", async ({
  page,
}) => {
  const saves = await mockServer(page);
  await page.goto("/#/settings");
  await page.getByRole("tab", { name: "Network", exact: true }).click();
  await expect(page.locator("#ConnectionsLimit")).toHaveCount(0);
  await page.getByRole("checkbox", { name: "Show advanced settings" }).check();
  await page.locator("#ConnectionsLimit").fill("321");
  await page
    .getByRole("checkbox", { name: "Show advanced settings" })
    .uncheck();
  await expect(page.locator("#ConnectionsLimit")).toHaveCount(0);
  const network = page.getByRole("tab", { name: "Network", exact: true });
  await network.focus();
  await page.keyboard.press("Home");
  await expect(
    page.getByRole("tab", { name: "General", exact: true }),
  ).toBeFocused();
  await page.getByRole("button", { name: "512 MiB", exact: true }).click();
  await page
    .getByRole("button", { name: "Apply settings", exact: true })
    .click();
  await page.getByRole("button", { name: "Confirm", exact: true }).click();
  await expect.poll(() => saves.length).toBe(1);
  expect(saves[0]).toMatchObject({
    sets: { ConnectionsLimit: 321, CacheSize: 536870912 },
  });
});

test("required expiring playback links preserve the playlist session", async ({
  page,
}) => {
  await mockServer(page);
  await page.route("**/settings", (route) =>
    route.request().postDataJSON().action === "get"
      ? route.fulfill({
          json: {
            ...settings,
            Flow: { ...settings.Flow, RequirePlaybackToken: true },
          },
        })
      : route.fallback(),
  );
  let requested: unknown;
  await page.route("**/flow/playback-link", (route) => {
    requested = route.request().postDataJSON();
    return route.fulfill({
      json: {
        path: "/flow/play/fixture.signed",
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      },
    });
  });
  await page.goto("/#/torrents");
  await page.getByRole("button", { name: "Files / Play" }).click();
  await expect(
    page.getByRole("link", { name: "Open stream", exact: true }),
  ).toHaveAttribute("href", /\/flow\/play\/fixture\.signed\?ss=/);
  expect(requested).toEqual({ hash, index: 1 });
  await expect(page.getByText(/Valid until/)).toBeVisible();
});

test("deadline experiment is off by default and saves only after confirmation", async ({
  page,
}) => {
  const saves = await mockServer(page);
  await page.goto("/#/settings");
  await page.getByRole("tab", { name: "Flow", exact: true }).click();
  await page.getByRole("checkbox", { name: "Show advanced settings" }).check();
  const experiment = page.getByRole("checkbox", {
    name: "Rate-aware deadline experiment",
    exact: true,
  });
  await expect(experiment).not.toBeChecked();
  await expect(
    page.getByText(/Off by default: repeated tests showed mixed seek results/),
  ).toBeVisible();
  await experiment.check();
  expect(saves).toHaveLength(0);
  await page
    .getByRole("button", { name: "Apply settings", exact: true })
    .click();
  await page.getByRole("button", { name: "Confirm", exact: true }).click();
  await expect.poll(() => saves.length).toBe(1);
  expect(saves[0]).toMatchObject({
    action: "set",
    sets: { Flow: { RateAwareDeadlines: true } },
  });
});

test("dirty form blocks navigation; continuing keeps edits", async ({
  page,
}) => {
  await mockServer(page);
  await page.goto("/#/settings");
  await page.getByRole("button", { name: "512 MiB", exact: true }).click();
  await page
    .getByRole("link", { name: "Dashboard", exact: true })
    .first()
    .click();
  await expect(
    page.getByRole("dialog", { name: "Unsaved changes" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Continue editing" }).click();
  await expect(
    page.getByRole("heading", { name: "Settings", exact: true }),
  ).toBeVisible();
});
test("backup preview is inert until confirmation and support report downloads", async ({
  page,
}) => {
  await mockServer(page);
  const applies: unknown[] = [];
  await page.route("**/flow/backup/preview", (route) =>
    route.fulfill({
      json: {
        digest: "a".repeat(64),
        library_count: 1,
        settings_fields: ["CacheSize"],
        credentials_excluded: true,
        mode: "merge",
        restart_required: true,
      },
    }),
  );
  await page.route("**/flow/backup/apply", (route) => {
    applies.push(route.request().postDataJSON());
    return route.fulfill({
      json: { restored: true, recovery_backup: "before-restore-fixture.json" },
    });
  });
  await page.route("**/flow/support", (route) =>
    route.fulfill({ json: { schema_version: 1, privacy: "redacted" } }),
  );
  await page.goto("/#/settings");
  await page.getByRole("tab", { name: "Advanced", exact: true }).click();
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "Download support report" }).click();
  expect((await download).suggestedFilename()).toBe(
    "TorrServer-Flow-support.json",
  );
  const file = {
    name: "portable.json",
    mimeType: "application/json",
    buffer: Buffer.from(JSON.stringify({ fixture: true })),
  };
  await page
    .getByLabel("Preview backup import", { exact: true })
    .setInputFiles(file);
  await expect(
    page.getByRole("dialog", { name: "Restore preview" }),
  ).toBeVisible();
  expect(applies).toHaveLength(0);
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  expect(applies).toHaveLength(0);
  await page
    .getByLabel("Preview backup import", { exact: true })
    .setInputFiles(file);
  await page
    .getByRole("button", { name: "Restore backup", exact: true })
    .click();
  await expect(page.getByText(/Restored. Local recovery backup/)).toBeVisible();
  expect(applies).toEqual([
    { backup: { fixture: true }, digest: "a".repeat(64) },
  ]);
});

test("file view retains raw name, watched timecode and browser fallback", async ({
  page,
}) => {
  await mockServer(page);
  await page.goto("/#/torrents");
  await page.getByRole("button", { name: "Files / Play" }).click();
  await expect(
    page.getByText("Season 01/Show.S01E01.mkv", { exact: false }),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: /In progress/ })).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Open stream", exact: true }),
  ).toHaveAttribute("href", /index=1.*ss=/);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
});

test("episode preparation actions and readiness are visible without changing playback identity", async ({
  page,
}) => {
  await mockServer(page);
  let state = "";
  const actions: unknown[] = [];
  await page.route("**/flow/preparation", (route) => {
    if (route.request().method() === "POST") {
      const body = route.request().postDataJSON();
      actions.push(body);
      state = (
        {
          start: "downloading",
          pause: "paused",
          cancel: "cancelled",
          resume: "ready",
          remove: "",
        } as Record<string, string>
      )[body.action];
    }
    return route.fulfill({
      json: {
        jobs: state
          ? [
              {
                id: hash + ":1",
                hash,
                file_index: 1,
                state,
                length: 12345678,
                verified_bytes: state === "ready" ? 12345678 : 100,
                contiguous_bytes: state === "ready" ? 12345678 : 100,
                playback_ready: state === "ready",
              },
            ]
          : [],
        quota_bytes: 67108864,
        reserved_bytes: state ? 12582912 : 0,
      },
    });
  });
  await page.goto("/#/torrents");
  await page.getByRole("button", { name: "Files / Play" }).click();
  await page.locator("summary").filter({ hasText: "Prepare episode" }).click();
  await page
    .getByRole("button", { name: "Prepare episode", exact: true })
    .click();
  await page.getByRole("button", { name: "Pause preparation" }).click();
  await page
    .getByRole("button", { name: "Cancel and retain progress" })
    .click();
  await page.getByRole("button", { name: "Resume preparation" }).click();
  await expect(
    page.getByText(
      "The entire file is verified and retained, including seek data. Use the playback links above.",
    ),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Open stream", exact: true }),
  ).toHaveAttribute("href", /index=1.*ss=/);
  await page.getByRole("button", { name: "Remove prepared pieces" }).click();
  expect(actions).toEqual(
    ["start", "pause", "cancel", "resume", "remove"].map((action) => ({
      hash,
      file_index: 1,
      action,
    })),
  );
});

test("mirror management uses explicit LAN approval and hides private-torrent additions", async ({
  page,
}) => {
  await mockServer(page);
  let privateTorrent = false;
  const updates: unknown[] = [];
  await page.route("**/flow/sources/*", (route) => {
    if (route.request().method() === "POST") {
      updates.push(route.request().postDataJSON());
      privateTorrent = true;
    }
    return route.fulfill({ json: { private: privateTorrent, sources: [] } });
  });
  await page.goto("/#/torrents");
  await page.getByRole("button", { name: "Files / Play" }).click();
  await page.locator("summary").filter({ hasText: "HTTP mirrors" }).click();
  await page
    .getByLabel("Mirror URL", { exact: true })
    .fill("http://192.168.1.20/episode/");
  await page.getByLabel("Approve my LAN mirror").check();
  await page.getByRole("button", { name: "Add mirror", exact: true }).click();
  expect(updates).toEqual([
    { action: "add", url: "http://192.168.1.20/episode/", allow_local: true },
  ]);
  await expect(
    page.getByRole("button", { name: "Add mirror", exact: true }),
  ).toHaveCount(0);
});
test("pairing avoids loopback QR and has manual input", async ({ page }) => {
  await mockServer(page);
  await page.goto("/");
  await page.getByRole("button", { name: "Connect phone" }).click();
  await expect(page.getByLabel("Manual server address")).toHaveValue(
    "http://192.168.1.5:5173",
  );
  await page.getByLabel("Manual server address").fill("http://127.0.0.1:8090");
  await expect(page.getByText(/Enter an HTTP/)).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Copy Lampa address" }),
  ).toHaveCount(0);
});
test("search prepares selected file before offering playback", async ({
  page,
}) => {
  await mockServer(page);
  await page.goto("/#/add");
  await page
    .getByRole("textbox", { name: "Search", exact: true })
    .fill("Example");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await page.getByRole("button", { name: "Add and prepare" }).click();
  await page.getByLabel("Files / Play").selectOption("1");
  await expect(
    page.getByRole("link", { name: "Open stream", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "Prepare playback" }).click();
  await expect(
    page.getByRole("link", { name: "Open stream", exact: true }),
  ).toBeVisible();
});

test("connection failure retains library and recovers after retry", async ({
  page,
}) => {
  await mockServer(page);
  await page.goto("/#/torrents");
  await expect(
    page.getByRole("heading", { name: "Example series" }),
  ).toBeVisible();
  let fail = true;
  await page.route("**/torrents", (route) =>
    fail
      ? route.abort("connectionfailed")
      : route.fulfill({
          contentType: "application/json",
          body: JSON.stringify([torrent]),
        }),
  );
  await expect(page.getByText(/last received data/)).toBeVisible({
    timeout: 15000,
  });
  await expect(
    page.getByRole("heading", { name: "Example series" }),
  ).toBeVisible();
  fail = false;
  await page.getByRole("button", { name: "Retry", exact: true }).click();
  await expect(page.getByText(/last received data/)).toHaveCount(0);
});
test("authentication failure is actionable and skip link does not change routes", async ({
  page,
}) => {
  await mockServer(page, { status: 401 });
  await page.goto("/#/torrents");
  await expect(
    page.getByText("Authentication required", { exact: true }).first(),
  ).toBeVisible();
  await page.keyboard.press("Tab");
  await page.getByRole("link", { name: "Skip to content" }).press("Enter");
  await expect(page).toHaveURL(/#\/torrents$/);
  await expect(page.locator("main")).toBeFocused();
});
test("active playback view names the episode and renders measured health", async ({
  page,
}) => {
  await mockServer(page, { active: true });
  await page.goto("/");
  await expect(page.getByText("Healthy", { exact: true })).toBeVisible();
  await expect(
    page.getByRole("heading", { name: /Season 01\/Show.S01E01.mkv/ }),
  ).toBeVisible();
  await page.screenshot({
    path: "test-results/active-desktop.png",
    fullPage: true,
  });
});
test("pairing dialog traps keyboard focus and dismisses with Escape", async ({
  page,
}) => {
  await mockServer(page);
  await page.goto("/");
  await page.getByRole("button", { name: "Connect phone" }).click();
  for (let i = 0; i < 12; i++) {
    await page.keyboard.press("Tab");
    expect(
      await page
        .getByRole("dialog")
        .evaluate((el) => el.contains(document.activeElement)),
    ).toBe(true);
  }
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
});
test("mobile settings and pairing visual review", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await mockServer(page);
  await page.goto("/#/settings");
  await expect(
    page.getByRole("button", { name: "Apply settings", exact: true }),
  ).toBeVisible();
  await page.screenshot({
    path: "test-results/settings-mobile.png",
    fullPage: true,
  });
  await page.getByRole("button", { name: "Connect phone" }).click();
  await expect(page.getByLabel("Manual server address")).toBeVisible();
  await page.screenshot({
    path: "test-results/pairing-mobile.png",
    fullPage: true,
  });
});

test("engine controls reflect server pause and resume", async ({ page }) => {
  await mockServer(page);
  const actions: string[] = [];
  await page.route("**/flow/control", (route) => {
    const action = route.request().postDataJSON().action;
    actions.push(action);
    return route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        server_state: action === "pause" ? "PAUSED" : "RUNNING",
        active_torrent: "",
        download_rate: 0,
        buffer_seconds: 0,
        peer_count: 0,
      }),
    });
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Pause engine", exact: true }).click();
  await expect(page.getByText("Engine paused", { exact: true })).toBeVisible();
  await page
    .getByRole("button", { name: "Resume engine", exact: true })
    .click();
  await expect(page.getByText("Degraded", { exact: true })).toBeVisible();
  expect(actions).toEqual(["pause", "resume"]);
});
test("warm session is discoverable on a fresh dashboard", async ({ page }) => {
  await mockServer(page, { active: true });
  await page.route("**/torrents", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify([
        { ...torrent, active_readers: 0, warm_idle: true },
      ]),
    }),
  );
  await page.route("**/flow/status/*", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        hash,
        sampled_at: new Date().toISOString(),
        sessions: [
          {
            group: "phone",
            buffer_sampled_at_ms: Date.now(),
            file_index: 1,
            state: "WARM_IDLE",
            active_readers: 0,
            buffer_ahead_seconds: 40,
            buffer_ahead_bytes: 1000,
            playback_consumption_rate: 0,
            download_rate: 0,
            sustainability_ratio: 0,
            piece_wait_p95_ms: 0,
            seek_count: 0,
            seek_recovery_ms: 0,
            buffer_warning: false,
            connected_peers: 5,
          },
        ],
      }),
    }),
  );
  await page.goto("/");
  await expect(page.getByText("Warm idle", { exact: true })).toBeVisible();
});
for (const [code, heading] of Object.entries({
  ru: "Обзор",
  ua: "Огляд",
  bg: "Обзор",
  fr: "Tableau de bord",
  ro: "Panou de control",
  zh: "概览",
})) {
  test("localized phone layout " + code, async ({ page }) => {
    await mockServer(page);
    await page.setViewportSize({ width: 375, height: 812 });
    await page.goto("/");
    await page.locator("#language").selectOption(code);
    await expect(
      page.getByRole("heading", { name: heading, exact: true }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  });
}

test("incremental search keeps a fast result while another source is pending", async ({
  page,
}) => {
  await mockServer(page);
  await page.route("**/settings", async (route) => {
    if (route.request().postDataJSON()?.action !== "get")
      return route.fallback();
    await route.fulfill({
      json: {
        ...settings,
        EnableRutorSearch: false,
        EnableTorznabSearch: true,
        TorznabUrls: [
          { Host: "https://one.example", Key: "fixture" },
          { Host: "https://two.example", Key: "fixture" },
        ],
      },
    });
  });
  let finish!: () => void;
  const slow = new Promise<void>((resolve) => {
    finish = resolve;
  });
  await page.route("**/torznab/search/**", async (route) => {
    const index = new URL(route.request().url()).searchParams.get("index");
    if (index === "1") await slow;
    await route
      .fulfill({
        json: {
          items:
            index === "0"
              ? [{ Title: "Fast generated result", Hash: "a".repeat(40) }]
              : [],
          sources: [
            {
              source: "torznab",
              index: Number(index),
              state: index === "0" ? "ok" : "empty",
            },
          ],
        },
      })
      .catch(() => {});
  });
  try {
    await page.goto("/#/add");
    await page
      .getByRole("textbox", { name: "Search", exact: true })
      .fill("generated");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await expect(
      page.getByText("Fast generated result", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("list", { name: "Search sources" }),
    ).toContainText("Searching");
    finish();
    await expect(
      page.getByRole("list", { name: "Search sources" }),
    ).toContainText("No matches");
  } finally {
    finish();
  }
});

test("recent diagnostic history is fetched only on request and states incomplete coverage", async ({
  page,
}) => {
  await mockServer(page);
  let reads = 0;
  await page.route("**/flow/sessions", (route) => {
    reads++;
    return route.fulfill({
      json: {
        enabled: true,
        truncated: true,
        items: [
          {
            started: "2026-10-24T00:00:00Z",
            last_observed: "2026-10-24T00:01:00Z",
            samples: 1,
            bytes_served: 1024,
            peak_recent_read_stalls: 2,
            peak_recent_wait_p95_ms: 500,
            minimum_buffer_bytes: 0,
            maximum_buffer_bytes: 4096,
            range_requests: 3,
            range_cancels: 1,
            incomplete: true,
          },
        ],
      },
    });
  });
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Troubleshooting and recent sessions" }),
  ).toBeVisible();
  expect(reads).toBe(0);
  await page
    .getByRole("button", {
      name: "Troubleshooting and recent sessions",
      exact: true,
    })
    .click();
  await page.getByRole("button", { name: "Load recent sessions" }).click();
  await expect(
    page.getByText("Incomplete evidence", { exact: true }),
  ).toBeVisible();
  expect(reads).toBe(1);
  await page
    .getByText("Find where the problem starts", { exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Test connection" }),
  ).toBeVisible();
});

test("storage drafts retain their matching revision after configuration polling", async ({
  page,
}) => {
  await mockServer(page);
  let revision = "storage-draft-base";
  let submitted: unknown;
  await page.route("**/flow/active", (route) =>
    route.fulfill({
      json: { items: [], sampled_at: new Date().toISOString() },
    }),
  );
  await page.route("**/settings", (route) => {
    if (route.request().postDataJSON().action !== "state")
      return route.fallback();
    return route.fulfill({
      json: {
        revision,
        saved: settings,
        effective: settings,
        recovery: { source: "settings", schema_version: 1 },
        data_path: "fixture-state",
        executable: "fixture-server",
      },
    });
  });
  await page.route("**/storage/settings", (route) => {
    if (route.request().method() === "GET")
      return route.fulfill({
        json: {
          settings: "json",
          viewed: "bbolt",
          viewedCount: 12,
          revision: "storage-draft-base",
        },
      });
    submitted = route.request().postDataJSON();
    return route.fulfill({
      status: 409,
      json: { code: "SETTINGS_CONFLICT", error: "stale revision" },
    });
  });
  await page.goto("/#/settings?tab=advanced");
  await page
    .getByRole("combobox", { name: "Settings storage", exact: true })
    .selectOption("bbolt");
  revision = "external-change";
  await expect(page.getByText("external-change", { exact: true })).toHaveCount(
    1,
    { timeout: 10000 },
  );
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect
    .poll(() => submitted)
    .toEqual({
      settings: "bbolt",
      viewed: "bbolt",
      revision: "storage-draft-base",
    });
  await expect(
    page
      .getByText(/Settings changed on the server while this draft was open/)
      .last(),
  ).toBeVisible();
  await expect(
    page.getByRole("combobox", { name: "Settings storage", exact: true }),
  ).toHaveValue("bbolt");
});
