import type { Page } from "@playwright/test";
import settings from "./settings.json" with { type: "json" };

export async function upgradeFixture(page: Page, count = 200) {
  const sampled_at = new Date().toISOString();
  await page.route("**/*", (route) => {
    const url = new URL(route.request().url());
    const json = (payload: unknown) => route.fulfill({ json: payload });
    if (url.pathname === "/echo")
      return route.fulfill({ body: "MatriX.145.Flow-test" });
    if (url.pathname === "/flow/active") return json({ items: [], sampled_at });
    if (url.pathname === "/flow/library")
      return json({
        items: Array.from({ length: Math.min(50, count) }, (_, i) => ({
          hash: i.toString(16).padStart(40, "0"),
          title: `Fixture ${i}`,
          stat: 3,
          torrent_size: 12345678,
          category: "Fixtures",
        })),
        total: count,
        library_total: count,
        page: 1,
        limit: 50,
        categories: ["Fixtures"],
        sampled_at,
      });
    if (url.pathname === "/flow/tray")
      return json({
        server_state: "RUNNING",
        download_rate: 0,
        buffer_seconds: 0,
        peer_count: 0,
      });
    if (url.pathname === "/flow/network")
      return json({
        state: "ADDRESS_READY",
        connectivity: "ONLINE",
        addresses: [],
        checked_at: sampled_at,
        changed_at: sampled_at,
        next_check_seconds: 15,
        reannounce_count: 0,
      });
    if (url.pathname === "/runtime/status")
      return json({
        dlna_enabled: false,
        bonjour_enabled: false,
        friendly_name: "Flow",
        webdav_enabled: false,
        webdav_path: "/dav",
        fuse_path: "",
        fuse_enabled: false,
        bt: { active_streams: 0, torrent_count: 0, torrents: [] },
      });
    if (url.pathname === "/flow/preparation")
      return json({ jobs: [], quota_bytes: 67108864, reserved_bytes: 0 });
    if (url.pathname === "/settings") {
      const body = route.request().postDataJSON();
      return json(
        body.action === "state"
          ? {
              revision: "fixture",
              saved: settings,
              effective: settings,
              recovery: { source: "settings", schema_version: 1 },
              data_path: "fixture",
              executable: "fixture",
            }
          : settings,
      );
    }
    if (url.pathname === "/ssl/status")
      return json({
        enabled: false,
        http_port: "8090",
        http_enabled: true,
        force_https: false,
        http_media: false,
        read_only: false,
        cert_from_flags: false,
        revision: "fixture",
        cert: { source: "none", trusted: false },
      });
    return route.continue();
  });
}
