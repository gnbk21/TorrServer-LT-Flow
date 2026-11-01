import { test, expect, type Page } from "@playwright/test";
import { readFileSync } from "node:fs";
import settings from "./settings.json" with { type: "json" };
import { upgradeFixture } from "./upgrade-fixture";

async function balancedFixture(page: Page) {
  await upgradeFixture(page, 2);
  // This scenario exercises storage inventory, not preference editing. Never
  // let the Vite proxy consult the owner's port for an unrelated read.
  await page.route("**/storage/settings", (route) =>
    route.fulfill({
      status: 404,
      json: { error: "preference endpoint omitted by fixture" },
    }),
  );
  const value = {
    ...settings,
    DefaultTrackers: "",
    Flow: { ...settings.Flow, IncidentFlightRecorder: false },
  };
  let mutations = 0,
    proofs = 0;
  const rule = {
    type: "integer",
    default: 0,
    restart_required: false,
    experimental: false,
    deprecated: false,
    read_only: false,
  };
  await page.route("**/settings/metadata/v1", (route) =>
    route.fulfill({
      json: {
        schema_version: 1,
        fields: {
          CacheSize: {
            ...rule,
            default: 64 * 1048576,
            min: 1,
            max: 16 * 2 ** 30,
            unit: "bytes",
            invalid_policy: "reject",
          },
          "Flow.SwarmProfile": {
            ...rule,
            type: "string",
            default: "legacy",
            choices: ["legacy", "adaptive", "balanced"],
            restart_required: true,
          },
          "Flow.SecurityProfile": {
            ...rule,
            type: "string",
            default: "compatible",
            deprecated: true,
          },
          "Flow.IncidentFlightRecorder": {
            ...rule,
            type: "boolean",
            default: false,
          },
        },
      },
    }),
  );
  await page.route("**/settings", (route) => {
    if (new URL(route.request().url()).pathname !== "/settings")
      return route.fallback();
    const body = route.request().postDataJSON();
    if (body.action === "state")
      return route.fulfill({
        json: {
          revision: "balanced-fixture",
          saved: value,
          effective: value,
          recovery: { source: "settings", schema_version: 1 },
          data_path: "fixture",
          executable: "fixture",
        },
      });
    if (body.action === "get") return route.fulfill({ json: value });
    if (body.action === "plan")
      return route.fulfill({
        json: {
          restart_required: true,
          active_work: false,
          cache_preview: {
            budget_bytes: 512 * 1048576,
            warm_budget_bytes: 64 * 1048576,
            memory_pressure: false,
          },
        },
      });
    mutations++;
    return route.fulfill({
      status: 409,
      json: { error: "fixture forbids changes" },
    });
  });
  await page.route("**/mediabase", (route) =>
    route.fulfill({ json: { base: "http://192.168.50.10:8090" } }),
  );
  await page.route("**/flow/network", (route) =>
    route.fulfill({
      json: {
        state: "ADDRESS_READY",
        connectivity: "ONLINE",
        addresses: ["192.168.50.10"],
        checked_at: new Date().toISOString(),
        changed_at: new Date().toISOString(),
        next_check_seconds: 15,
        reannounce_count: 0,
      },
    }),
  );
  const proof = {
    id: "A".repeat(26),
    expires_at: new Date(Date.now() + 300000).toISOString(),
    different_client: false,
    evidence: [],
  };
  await page.route("**/flow/pairing", (route) => {
    proofs++;
    return route.fulfill({
      status: 201,
      json: {
        schema_version: 1,
        proof: { ...proof, token: "B".repeat(26) },
        startup: { listeners_ready: true, engine_ready: false },
        listener_addresses: [""],
        advertised_addresses: ["192.168.50.10"],
        auth_required: true,
      },
    });
  });
  await page.route("**/flow/pairing/status/*", (route) =>
    route.fulfill({
      json: {
        ...proof,
        different_client: true,
        reached_at: new Date().toISOString(),
        evidence: [
          {
            at: new Date().toISOString(),
            code: "MANAGEMENT_ORIGIN_DENIED",
            origin: "https://fixture.invalid",
          },
        ],
      },
    }),
  );
  await page.route("**/flow/storage", (route) =>
    route.fulfill({
      json: {
        schema_version: 1,
        preview_only: true,
        ram_cache_bytes: 1048576,
        categories: Object.fromEntries(
          ["preparation", "evicting_disk_cache", "diagnostics", "backups"].map(
            (name) => [
              name,
              { bytes: 1048576, files: 1, truncated: false, protected: true },
            ],
          ),
        ),
      },
    }),
  );
  return { mutations: () => mutations, proofs: () => proofs };
}

for (const language of ["en", "ru", "ua", "bg", "fr", "ro", "zh"]) {
  test(`balanced setup, metadata and storage mobile ${language}`, async ({
    page,
  }, info) => {
    const fixture = await balancedFixture(page);
    const labels = JSON.parse(
      readFileSync(
        new URL(
          `../src/i18n/locales/${language}/translation.json`,
          import.meta.url,
        ),
        "utf8",
      ),
    );
    await page.setViewportSize({ width: 375, height: 812 });
    await page.addInitScript(
      (language) => localStorage.setItem("i18nextLng", language),
      language,
    );
    await page.goto("/");
    await page
      .getByRole("button", { name: labels.pairing.connect, exact: true })
      .click();
    const dialog = page.getByRole("dialog");
    await expect(
      dialog.getByRole("heading", {
        name: labels.pairing.guide.title,
        exact: true,
      }),
    ).toBeVisible();
    expect(fixture.proofs()).toBe(0);
    await dialog
      .getByRole("button", { name: labels.pairing.guide.create, exact: true })
      .click();
    await expect(
      dialog.getByText(labels.pairing.guide.received, { exact: true }),
    ).toBeVisible();
    await expect(
      dialog.getByText(labels.pairing.guide.allInterfaces, { exact: true }),
    ).toBeVisible();
    await expect(
      dialog.getByText("https://fixture.invalid", { exact: false }),
    ).toBeVisible();
    expect(fixture.proofs()).toBe(1);
    expect(fixture.mutations()).toBe(0);
    await expect(dialog).not.toContainText("pairing.guide.");
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    if (language === "en")
      await page.screenshot({
        path: info.outputPath("guided-setup-mobile.png"),
        fullPage: true,
      });
    await page.keyboard.press("Escape");
    await expect(dialog).toHaveCount(0);
    await page.goto("/#/settings");
    await expect(
      page.getByText(labels.settings.metadata.fallback, { exact: true }),
    ).toHaveCount(0);
    await page
      .getByRole("tab", { name: labels.settings.tabs.flow, exact: true })
      .click();
    await page
      .getByRole("checkbox", {
        name: new RegExp(
          labels.settings.showAdvanced.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"),
        ),
      })
      .check();
    await expect(page.locator("#Flow__SecurityProfile")).toHaveCount(0);
    await expect(page.locator("#Flow__SwarmProfile option")).toHaveCount(3);
    await expect(
      page
        .getByText(labels.settings.metadata.restart, { exact: false })
        .first(),
    ).toBeVisible();
    await page
      .getByRole("tab", { name: labels.settings.tabs.advanced, exact: true })
      .click();
    await expect(
      page.locator("#Flow__IncidentFlightRecorder"),
    ).not.toBeChecked();
    await page
      .getByRole("button", {
        name: labels.settings.storageView.scan,
        exact: true,
      })
      .click();
    await expect(
      page.getByText(labels.settings.storageView.preparation, { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText(labels.settings.storageView.backups, { exact: true }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    expect(fixture.mutations()).toBe(0);
    if (language === "en")
      await page.screenshot({
        path: info.outputPath("storage-mobile.png"),
        fullPage: true,
      });
  });
}

test("settings preview describes allocation and remains inert until explicit apply", async ({
  page,
}) => {
  const fixture = await balancedFixture(page);
  await page.goto("/#/settings");
  await page.getByRole("button", { name: "512 MiB", exact: true }).click();
  await page
    .getByRole("button", { name: "Apply settings", exact: true })
    .click();
  const dialog = page.getByRole("dialog");
  await expect(
    dialog.getByText(/Shared eviction budget: 512 MiB/),
  ).toBeVisible();
  expect(fixture.mutations()).toBe(0);
  await page.keyboard.press("Escape");
  expect(fixture.mutations()).toBe(0);
});

test("selected-file metadata is lazy, bounded and handles long filenames on mobile", async ({
  page,
}) => {
  await balancedFixture(page);
  const hash = "0".repeat(40);
  const path = "Series/" + "Long episode filename ".repeat(18) + ".mkv";
  let mediaReads = 0,
    probes = 0;
  await page.route("**/torrents", (route) =>
    route.fulfill({
      json: {
        hash,
        title: "Fixture 0",
        stat: 3,
        torrent_size: 2000000,
        file_stats: [{ id: 1, path, length: 2000000 }],
      },
    }),
  );
  await page.route("**/viewed", (route) => route.fulfill({ json: [] }));
  await page.route("**/flow/sources/*", (route) =>
    route.fulfill({ json: { private: false, sources: [] } }),
  );
  await page.route("**/flow/media/*/*", (route) => {
    mediaReads++;
    return route.fulfill({
      json: {
        schema_version: 1,
        known: true,
        source: "existing_bounded_probe",
        media: {
          bitrate: "90000000",
          duration_seconds: 1200,
          truncated: false,
          probed_at: new Date().toISOString(),
          tracks: [
            { index: 0, type: "video", codec: "hevc" },
            { index: 1, type: "audio", codec: "eac3", language: "eng" },
            { index: 2, type: "subtitle", codec: "subrip", language: "rus" },
          ],
        },
      },
    });
  });
  await page.route("**/ffp/**", (route) => {
    probes++;
    return route.abort();
  });
  await page.setViewportSize({ width: 375, height: 812 });
  await page.goto("/#/torrents");
  await page
    .getByRole("button", { name: "Files / Play", exact: true })
    .first()
    .click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByText(path, { exact: false }).first()).toBeVisible();
  expect(mediaReads).toBe(0);
  await dialog
    .locator("summary")
    .filter({ hasText: "Media information" })
    .click();
  await expect(dialog.getByText(/90.00 Mbps · 11.25 MB\/s/)).toBeVisible();
  await expect(dialog.getByText(/eac3 · eng/)).toBeVisible();
  await expect(dialog.getByText(/subrip · rus/)).toBeVisible();
  expect(mediaReads).toBe(1);
  expect(probes).toBe(0);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});

test("expired pairing proof offers renewal without a misleading service error", async ({
  page,
}) => {
  await balancedFixture(page);
  await page.route("**/flow/pairing/status/*", (route) =>
    route.fulfill({ status: 410 }),
  );
  await page.goto("/");
  await page
    .getByRole("button", { name: "Connect phone", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Create connection proof", exact: true })
    .click();
  await expect(
    page.getByText("Proof expired. Create a new one.", { exact: true }),
  ).toBeVisible();
  await expect(page.getByRole("dialog").getByRole("alert")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Create connection proof", exact: true }),
  ).toBeEnabled();
});

test("playback overview describes the actual metadata recovery state", async ({
  page,
}) => {
  await balancedFixture(page);
  await page.route("**/flow/status/**", (route) =>
    route.fulfill({
      json: {
        hash: "0".repeat(40),
        sessions: [],
        sampled_at: new Date().toISOString(),
        startup: {
          state: "METADATA",
          file_index: 1,
          metadata_recovery: { state: "METADATA_HTTP", attempts: 2 },
        },
      },
    }),
  );
  const en = JSON.parse(
    readFileSync(
      new URL("../src/i18n/locales/en/translation.json", import.meta.url),
      "utf8",
    ),
  );
  await page.goto("/#/torrents");
  await page
    .getByRole("button", { name: en.flow.diagnostics, exact: true })
    .first()
    .click();
  await expect(
    page
      .getByRole("dialog")
      .getByText(
        "Metadata recovery: Retrieving metadata from an alternate source.; attempts 2",
        { exact: true },
      ),
  ).toBeVisible();
});

test("long metadata defaults wrap and initial pressure evidence remains unknown", async ({
  page,
}) => {
  await balancedFixture(page);
  const longDefault = "https://tracker.invalid/" + "a".repeat(400);
  await page.route("**/settings/metadata/v1", (route) =>
    route.fulfill({
      json: {
        schema_version: 1,
        fields: {
          DefaultTrackers: {
            type: "string",
            default: longDefault,
            restart_required: true,
            experimental: false,
            deprecated: false,
            read_only: false,
          },
        },
      },
    }),
  );
  await page.route("**/runtime/status", (route) =>
    route.fulfill({
      json: {
        dlna_enabled: false,
        bonjour_enabled: false,
        friendly_name: "Flow",
        webdav_enabled: false,
        webdav_path: "/dav",
        fuse_path: "",
        fuse_enabled: false,
        bt: { active_streams: 0, torrent_count: 0, torrents: [] },
        resources: {
          budget_bytes: 0,
          warm_budget_bytes: 0,
          memory_pressure: false,
          protected_overcommit: false,
          background_limited: false,
          ram_resident_bytes: 0,
          disk_cache_bytes: 0,
          pressure_state: "",
          recovery_samples: 0,
        },
      },
    }),
  );
  await page.setViewportSize({ width: 375, height: 812 });
  await page.goto("/#/settings");
  await page
    .getByRole("searchbox", { name: "Search all settings", exact: true })
    .fill("tracker");
  await expect(page.getByText(longDefault, { exact: false })).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page
    .getByRole("searchbox", { name: "Search all settings", exact: true })
    .fill("");
  await page.getByRole("tab", { name: "Advanced", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Runtime storage", exact: true }),
  ).toBeVisible();
  await expect(page.locator("main")).not.toContainText(
    "settings.storageView.pressureStates.",
  );
});
