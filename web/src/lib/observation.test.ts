import { afterEach, expect, test, vi } from "vitest";
import { observationNow, observeFlow } from "./observation";
import { playbackOverview } from "./playbackOverview";
import type { FlowStatusResponse } from "../types/flow";
afterEach(() => vi.restoreAllMocks());
test("server and client clock skew or wall clock edits do not expire a received sample", () => {
  let monotonic = 100;
  vi.spyOn(performance, "now").mockImplementation(() => monotonic);
  const server = Date.parse("2026-10-24T12:00:00Z");
  const input = {
    hash: "fixture",
    sampled_at: new Date(server).toISOString(),
    sessions: [
      {
        state: "PLAYING",
        active_readers: 1,
        buffer_ahead_bytes: 1024,
        buffer_sampled_at_ms: server,
      },
    ],
  } as FlowStatusResponse;
  const received = observeFlow(input);
  for (const skew of [-120000, 120000, 86400000]) {
    vi.spyOn(Date, "now").mockReturnValue(server + skew);
    expect(playbackOverview(received, observationNow(received)).cause).toBe(
      "ready",
    );
  }
  monotonic += 11000;
  expect(playbackOverview(received, observationNow(received)).cause).toBe(
    "stale",
  );
});
test("compact active responses use their envelope time; unknown age stays unknown", () => {
  const input = { hash: "fixture", sessions: [] } as FlowStatusResponse;
  expect(Number.isFinite(observationNow(observeFlow(input)))).toBe(false);
  expect(
    Number.isFinite(observationNow(observeFlow(input, "2026-10-24T12:00:00Z"))),
  ).toBe(true);
});

test("slow responses cannot renew stale server measurements", () => {
  const server = Date.parse("2026-10-24T12:00:00Z");
  const status = {
    hash: "fixture",
    sampled_at: new Date(server).toISOString(),
    sessions: [
      { buffer_sample_age_ms: 2000, buffer_sampled_at_ms: server - 2000 },
    ],
  } as FlowStatusResponse;
  const received = observeFlow(status, undefined, 9000);
  expect(
    observationNow(received) - received.sessions![0]!.buffer_sampled_at_ms!,
  ).toBeGreaterThanOrEqual(11000);
});
