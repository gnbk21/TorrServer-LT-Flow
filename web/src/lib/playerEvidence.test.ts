import { expect, test } from "vitest";
import { playerReportSchema, readPlayerReport } from "./playerEvidence";

const sample = {
  elapsed_ms: 5000,
  recorded_at_ms: 1_800_000_000_000,
  startup_ms: null,
  rebuffer_events: 0,
  rebuffer_ms: 0,
  seek_events: 1,
  seek_buffer_ms: 200,
  paused_ms: 3000,
  play_ms: 1000,
  buffered_ms: 10000,
  dropped_frames: 0,
  bandwidth_bps: 10_000_000,
  state: 3,
  playing: true,
  session_id: null,
};
const report = {
  schema_version: 1,
  source: "flow-player-media3",
  clock: "player-elapsed-and-wall; server-clock-alignment-unknown",
  samples: [sample],
};
test("unknown startup/correlation remains unknown and sensitive extra fields are rejected", () => {
  expect(playerReportSchema.parse(report).samples[0]?.startup_ms).toBeNull();
  for (const bad of [
    { ...report, url: "https://private/secret" },
    { ...report, samples: [{ ...sample, media_name: "private" }] },
    { ...report, samples: [{ ...sample, session_id: "https://private" }] },
    { ...report, samples: [{ ...sample, rebuffer_ms: -1 }] },
    { ...report, samples: Array(129).fill(sample) },
  ]) {
    expect(playerReportSchema.safeParse(bad).success).toBe(false);
  }
});
test("report size limit is checked before parsing", async () => {
  await expect(
    readPlayerReport(new File(["x".repeat(262145)], "private.json")),
  ).rejects.toThrow("flow.playerReportInvalid");
});
