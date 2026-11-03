export type BrowserAdvice = "unknown" | "possible" | "unconfirmed";
// canPlayType is only an advisory for this browser, not the external player or
// exact codec profile/HDR/audio combination. It never starts a media probe.
export function browserAdvice(
  container: string | undefined,
  codec: string | undefined,
  canPlay: (type: string) => string,
): BrowserAdvice {
  const mime =
    container?.includes("mp4") || container?.includes("mov")
      ? "video/mp4"
      : container === "webm"
        ? "video/webm"
        : undefined;
  const codecs: Record<string, string> = {
    h264: "avc1",
    hevc: "hvc1",
    av1: "av01",
    vp9: "vp09",
    vp8: "vp8",
  };
  const value = codec && codecs[codec];
  if (!mime || !value) return "unknown";
  return canPlay(`${mime}; codecs="${value}"`) ? "possible" : "unconfirmed";
}
