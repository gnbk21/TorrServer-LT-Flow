import type { GstSettings } from "../api/integrations";

/** Reject values the server would silently normalize, keeping Apply predictable. */
export function validateGst(
  config: NonNullable<GstSettings["config"]>,
): boolean {
  const positive = [
    "InactiveMinutes",
    "AACBitrateKbps",
    "SegmentSeconds",
    "VideoBitrate",
  ];
  const nonnegative = [
    "MaxTasks",
    "AACChannels",
    "AACSamplerate",
    "SegmentDiff",
  ];
  for (const key of [...positive, ...nonnegative]) {
    const value = config[key];
    if (
      typeof value !== "number" ||
      !Number.isSafeInteger(value) ||
      value < (positive.includes(key) ? 1 : 0)
    )
      return false;
  }
  return (
    typeof config.GSTVersion === "number" &&
    Number.isFinite(config.GSTVersion) &&
    config.GSTVersion >= 1.22 &&
    (config.Source === "play" || config.Source === "stream")
  );
}
