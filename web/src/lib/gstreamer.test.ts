import { describe, it, expect } from "vitest";
import { validateGst } from "./gstreamer";
const valid = {
  GSTVersion: 1.28,
  Source: "stream",
  InactiveMinutes: 5,
  AACBitrateKbps: 256,
  SegmentSeconds: 6,
  VideoBitrate: 10000,
  MaxTasks: 0,
  AACChannels: 0,
  AACSamplerate: 0,
  SegmentDiff: 20,
};
describe("GStreamer settings validation", () => {
  it("accepts valid configuration including automatic zero values", () =>
    expect(validateGst(valid)).toBe(true));
  it.each([
    ["SegmentSeconds", 0],
    ["AACBitrateKbps", NaN],
    ["MaxTasks", -1],
    ["AACChannels", 1.5],
    ["GSTVersion", 1.2],
    ["Source", "invalid"],
  ])("rejects %s=%s before sending settings", (key, value) =>
    expect(validateGst({ ...valid, [key as string]: value })).toBe(false),
  );
});
