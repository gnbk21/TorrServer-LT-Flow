import { afterEach, expect, test, vi } from "vitest";
import { api } from "./client";
import {
  searchApi,
  deduplicateResults,
  normalizeSearchItem,
  type SearchProgress,
} from "./search";
afterEach(() => vi.restoreAllMocks());
test("a successful source appears before a slow one and survives another source's failure", async () => {
  let finish: (rows: unknown) => void = () => {};
  vi.spyOn(api, "get").mockImplementation(async (path) => {
    if (path.startsWith("/jacred")) throw new Error("owned failure");
    if (path.includes("index=1"))
      return await new Promise((resolve) => {
        finish = resolve;
      });
    return [{ Title: "fixture", Hash: "a".repeat(40) }] as never;
  });
  const seen: SearchProgress[] = [];
  const result = searchApi.search(
    "fixture",
    "all",
    undefined,
    undefined,
    ["torznab", "jacred"],
    (s) => seen.push(s),
    [0, 1],
  );
  await vi.waitFor(() =>
    expect(
      seen.some(
        (s) =>
          s.items.length === 1 && s.sources.some((o) => o.state === "pending"),
      ),
    ).toBe(true),
  );
  finish([]);
  expect(await result).toHaveLength(1);
  expect(seen.at(-1)?.sources.map((s) => s.state)).toEqual([
    "ok",
    "empty",
    "unavailable",
  ]);
});
test("only matching authenticated torrent identities deduplicate, including v2 magnets", () => {
  const a = normalizeSearchItem({ Title: "same", Hash: "a".repeat(64) });
  expect(a.magnet).toContain("urn:btmh:1220");
  const b = normalizeSearchItem({ Title: "same", Hash: "b".repeat(64) });
  expect(deduplicateResults([a, a, b])).toHaveLength(2);
  expect(
    deduplicateResults([
      { ...a, magnet: "" },
      { ...a, magnet: "" },
    ]),
  ).toHaveLength(2);
});
test("cancellation cannot publish results from an obsolete search", async () => {
  const controller = new AbortController();
  vi.spyOn(api, "get").mockImplementation(
    async (_path, signal) =>
      await new Promise((_resolve, reject) =>
        signal?.addEventListener("abort", () => reject(signal.reason)),
      ),
  );
  const seen: SearchProgress[] = [];
  const result = searchApi.search(
    "fixture",
    "jacred",
    undefined,
    controller.signal,
    [],
    (s) => seen.push(s),
  );
  const before = seen.length;
  controller.abort(new DOMException("cancelled", "AbortError"));
  await expect(result).rejects.toMatchObject({ name: "AbortError" });
  expect(seen).toHaveLength(before);
});
