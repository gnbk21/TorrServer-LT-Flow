// @vitest-environment node
import { afterEach, expect, test, vi } from "vitest";
import { measureLAN } from "./lan-test";

afterEach(() => vi.unstubAllGlobals());
function response(sizes: number[]) {
  return new Response(
    new ReadableStream({
      start(controller) {
        for (const size of sizes) controller.enqueue(new Uint8Array(size));
        controller.close();
      },
    }),
  );
}
test("LAN measurement uses bounded same-origin noncached transport", async () => {
  const fetch = vi.fn().mockResolvedValue(response([8 * 1048576, 8 * 1048576]));
  vi.stubGlobal("fetch", fetch);
  const controller = new AbortController();
  const result = await measureLAN(controller.signal);
  expect(result.bytes).toBe(16 * 1048576);
  expect(result.mbps).toBeGreaterThan(0);
  expect(fetch).toHaveBeenCalledWith("/flow/lan-test?mib=16", {
    signal: controller.signal,
    credentials: "same-origin",
    cache: "no-store",
  });
});
test.each([[1048576], [16 * 1048576, 1]])(
  "LAN measurement rejects incorrect byte budgets: %s",
  async (...sizes) => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(response(sizes)));
    await expect(measureLAN(new AbortController().signal)).rejects.toThrow(
      /truncated|budget/,
    );
  },
);
test("LAN cancellation reaches the transport", async () => {
  const controller = new AbortController();
  vi.stubGlobal(
    "fetch",
    vi.fn(
      (_url, options) =>
        new Promise((_resolve, reject) => {
          options.signal.addEventListener("abort", () =>
            reject(new DOMException("Aborted", "AbortError")),
          );
        }),
    ),
  );
  const result = measureLAN(controller.signal);
  controller.abort();
  await expect(result).rejects.toHaveProperty("name", "AbortError");
});
