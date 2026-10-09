import { describe, it, expect, vi, afterEach } from "vitest";
import { apiClient, ApiError } from "./client";
import { settingsApi } from "./settings";
import { runtimeApi } from "./runtime";
import { torrentsApi } from "./torrents";
import { integrationsApi } from "./integrations";
afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
describe("API contract", () => {
  it("allows cold GST discovery beyond the old deadline but bounds the full retry", async () => {
    vi.useFakeTimers();
    let signal: AbortSignal | undefined;
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation((_url, init) => {
        signal = init.signal;
        return new Promise((_resolve, reject) => {
          signal!.addEventListener("abort", () => reject(signal!.reason), {
            once: true,
          });
        });
      }),
    );
    const result = integrationsApi
      .getGstProbe("fixture", 1)
      .catch((error) => error);
    await vi.advanceTimersByTimeAsync(40_000);
    expect(signal?.aborted).toBe(false);
    await vi.advanceTimersByTimeAsync(80_000);
    expect(signal?.aborted).toBe(true);
    expect(await result).toMatchObject({ name: "TimeoutError" });
  });
  it("preserves a plain-text error body and status", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(new Response("engine starting", { status: 503 })),
    );
    await expect(apiClient("/torrents")).rejects.toMatchObject({
      status: 503,
      message: "engine starting",
      isStarting: true,
    });
  });
  it("does not offer a destructive defaults getter; explicit reset posts def", async () => {
    const fetcher = vi
      .fn()
      .mockImplementation(() => Promise.resolve(new Response("")));
    vi.stubGlobal("fetch", fetcher);
    expect(settingsApi).not.toHaveProperty("getDefaults");
    await settingsApi.reset();
    expect(JSON.parse(fetcher.mock.calls[0]![1].body)).toEqual({
      action: "def",
      revision: "",
    });
    await settingsApi.reset("saved-revision");
    expect(JSON.parse(fetcher.mock.calls[1]![1].body)).toEqual({
      action: "def",
      revision: "saved-revision",
    });
  });
  it("uses GET shutdown and the ss stream-group key", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(""));
    vi.stubGlobal("fetch", fetcher);
    await runtimeApi.shutdown();
    expect(fetcher.mock.calls[0]![1].method).toBe("GET");
    const url = new URL(torrentsApi.getStreamUrl("abc", 2));
    expect(url.searchParams.get("index")).toBe("2");
    expect(url.searchParams.has("ss")).toBe(true);
    expect(url.searchParams.has("session")).toBe(false);
  });
  it("forwards cancellation and prevents cross-origin API requests", async () => {
    const fetcher = vi.fn().mockImplementation((_url, init) => {
      expect(init.signal.aborted).toBe(true);
      return Promise.reject(init.signal.reason);
    });
    vi.stubGlobal("fetch", fetcher);
    const controller = new AbortController();
    controller.abort();
    await expect(
      apiClient("/torrents", { signal: controller.signal }),
    ).rejects.toBeDefined();
    await expect(apiClient("https://other.test/settings")).rejects.toThrow(
      "Cross-origin",
    );
  });
  it("recognizes authorization failures separately", () => {
    expect(new ApiError(403, "Forbidden", null).isUnauthorized).toBe(true);
  });
});
