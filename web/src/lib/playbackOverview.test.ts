import { expect, test } from "vitest";
import { playbackOverview } from "./playbackOverview";
import type { FlowStatusResponse, FlowSession } from "../types/flow";

const now = Date.parse("2026-10-22T10:00:00Z");
const session = {
  active_readers: 1,
  state: "PLAYING",
  buffer_ahead_bytes: 0,
  required_piece_suppliers: 0,
  wait_explanation: { confidence: "high" },
  buffer_sampled_at_ms: now,
} as FlowSession;
const status = {
  hash: "fixture",
  sampled_at: new Date(now).toISOString(),
  sessions: [session],
  sparse: { known: true, sampled_at_ms: now },
} as FlowStatusResponse;
test("stale or future evidence cannot diagnose the swarm", () => {
  expect(playbackOverview(status, now + 11000).cause).toBe("stale");
  expect(playbackOverview(status, now - 1).cause).toBe("stale");
  expect(
    playbackOverview(
      { ...status, sparse: { known: true, sampled_at_ms: now - 6000 } },
      now,
    ).cause,
  ).toBe("block");
});
test("seeks, idle, supplier evidence and buffer are distinct", () => {
  expect(playbackOverview(status, now).cause).toBe("missing");
  expect(
    playbackOverview(
      { ...status, sessions: [{ ...session, state: "SEEK_RECOVERY" }] },
      now,
    ).cause,
  ).toBe("seek");
  expect(
    playbackOverview(
      {
        ...status,
        sessions: [{ ...session, state: "WARM_IDLE", active_readers: 0 }],
      },
      now,
    ).cause,
  ).toBe("idle");
  const view = playbackOverview(
    { ...status, sessions: [{ ...session, buffer_ahead_bytes: 1024 }] },
    now,
  );
  expect(view.cause).toBe("ready");
  expect(view.bufferFresh).toBe(true);
  for (const timestamp of [undefined, 0, now - 11000, now + 1000]) {
    const stale = playbackOverview(
      {
        ...status,
        sessions: [
          {
            ...session,
            buffer_ahead_bytes: 1024,
            buffer_sampled_at_ms: timestamp,
          },
        ],
      },
      now,
    );
    expect(stale.cause).toBe("unknown");
    expect(stale.bufferFresh).toBe(false);
  }
  expect(
    playbackOverview(
      {
        ...status,
        sessions: [{ ...session, buffer_sampled_at_ms: undefined }],
      },
      now,
    ).bufferFresh,
  ).toBe(false);
});
