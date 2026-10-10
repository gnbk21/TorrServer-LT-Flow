import { afterEach, expect, it, vi } from "vitest";
import { api, ApiError } from "../api/client";
import { torrentsApi } from "../api/torrents";
import type { Torrent } from "../types/torrent";
import { parseImport } from "../components/torrent/AddTorrentModal";
import {
  formatLibraryExport,
  loadLibraryExport,
  portableLibraryRows,
} from "./libraryExport";

afterEach(() => vi.restoreAllMocks());
const fixture = {
  hash: "ab".repeat(20),
  title: "Episode",
  category: "series",
  stat: 5,
  data: '{"password":"secret"}',
  poster: "https://images.test/?token=secret",
  torrs_hash: "OpaquePrivateTrackerSecret",
  token: "secret",
  file_stats: [{ id: 1, path: "C:/secret", length: 4 }],
} as Torrent;

it("exports only portable fields and keeps JSON/magnets importable on legacy servers", () => {
  const rows = portableLibraryRows([
    fixture,
    { ...fixture, hash: "cd".repeat(32) },
  ]);
  const json = formatLibraryExport(rows, "json");
  for (const secret of [
    "secret",
    "Secret",
    "data",
    "poster",
    "file_stats",
    "token",
  ])
    expect(json).not.toContain(secret);
  expect(parseImport(json)).toEqual(
    rows.map((row) => ({
      link: row.hash,
      title: "Episode",
      category: "series",
      poster: "",
      data: "",
      save_to_db: true,
    })),
  );
  for (const format of ["magnets", "torrs"] as const) {
    const text = formatLibraryExport(rows, format);
    expect(text).toContain(`urn:btih:${fixture.hash}`);
    expect(text).toContain(`urn:btmh:1220${"cd".repeat(32)}`);
    expect(text).not.toContain("Secret");
    expect(parseImport(text)).toHaveLength(2);
  }
  expect(fixture.torrs_hash).toBe("OpaquePrivateTrackerSecret");
});

it("redacts recognizable credentials inside exported labels", () => {
  const rows = portableLibraryRows([
    {
      ...fixture,
      title:
        "Episode https://user:secret@tracker.test/private-secret?passkey=secret",
      category: "token=secret",
    },
  ]);
  expect(JSON.stringify(rows)).not.toContain("secret");
});

it("uses rebuilt compact tokens only from the dedicated server projection", async () => {
  vi.spyOn(api, "get").mockResolvedValue([
    {
      hash: fixture.hash,
      title: "Episode",
      category: "series",
      torrs_hash: "SafeToken",
      data: "secret",
    },
  ]);
  const rows = await loadLibraryExport();
  expect(formatLibraryExport(rows, "torrs")).toBe("torrs://SafeToken");
  expect(formatLibraryExport(rows, "json")).not.toContain("secret");
});

it("falls back only for a missing endpoint and excludes legacy compact tokens", async () => {
  const get = vi
    .spyOn(api, "get")
    .mockRejectedValue(new ApiError(404, "Missing", null));
  const list = vi.spyOn(torrentsApi, "list").mockResolvedValue([fixture]);
  expect(await loadLibraryExport()).toEqual(portableLibraryRows([fixture]));
  for (const error of [
    new ApiError(401, "Unauthorized", null),
    new ApiError(503, "Starting", null),
    new Error("Offline"),
  ]) {
    get.mockRejectedValue(error);
    await expect(loadLibraryExport()).rejects.toBe(error);
  }
  expect(list).toHaveBeenCalledTimes(1);
});

it("rejects malformed and oversized output without dropping entries silently", () => {
  expect(() =>
    portableLibraryRows([{ ...fixture, hash: "invalid" }]),
  ).toThrow();
  expect(() => portableLibraryRows(Array(5001).fill(fixture))).toThrow();
  expect(() =>
    formatLibraryExport(
      Array(5000).fill({
        hash: fixture.hash,
        title: "x".repeat(4096),
        category: "series",
      }),
      "json",
    ),
  ).toThrow();
  expect(formatLibraryExport([], "json")).toBe("[]");
});
