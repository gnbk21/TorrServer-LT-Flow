import { test, expect, type Page } from "@playwright/test";
import settings from "./settings.json" with { type: "json" };

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
}) => {
  test.setTimeout(45000);
  await mockServer(page, { active: true });
  await page.route("**/torrents", (route) =>
    route.fulfill({ json: [{ ...torrent, active_readers: 1 }] }),
  );
  const counts: Record<string, number> = {};
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
  await page.route("**/*", async (route) => {
    const path = new URL(route.request().url()).pathname;
    const json = (data: unknown, code = 200) =>
      route.fulfill({
        status: code,
        contentType: "application/json",
        body: JSON.stringify(data),
      });
    if (path === "/echo")
      return route.fulfill({ body: "MatriX.145.Flow-test" });
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
      if (body.action === "get") return json(settings);
      saves.push(body);
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
        sessions: active
          ? [
              {
                group: "phone",
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
test("503 remains starting rather than offline", async ({ page }) => {
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
  await expect(page.getByText(/Playback is active/)).toBeVisible();
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
    page.getByText("Authentication required or access denied").first(),
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
        sessions: [
          {
            group: "phone",
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
