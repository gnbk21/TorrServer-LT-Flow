import { describe, expect, it } from "vitest";
import { externalMediaURL, safeMediaBase } from "./mediaBase";

describe("external media origins", () => {
  const origin = "https://[fd00::1]:8091";
  it("rewrites only the origin and preserves capability and session queries", () => {
    expect(
      externalMediaURL(
        "/flow/play/token?ss=group&index=2",
        "http://[fd00::1]:8090",
        origin,
      ),
    ).toBe("http://[fd00::1]:8090/flow/play/token?ss=group&index=2");
  });
  it("preserves browser origin on older servers and unrelated links", () => {
    expect(externalMediaURL("/stream?play", undefined, origin)).toBe(
      origin + "/stream?play",
    );
    expect(
      externalMediaURL(
        "https://example.com/video",
        "http://[fd00::1]:8090",
        origin,
      ),
    ).toBe("https://example.com/video");
  });
  it("clears the old port when the media listener uses the default port", () => {
    expect(externalMediaURL("/stream?play", "http://[fd00::1]", origin)).toBe(
      "http://[fd00::1]/stream?play",
    );
  });
  it.each([
    "http://evil.test:8090",
    "http://user:password@[fd00::1]:8090",
    "ftp://[fd00::1]",
    "http://[fd00::1]/key",
    "http://[fd00::1]?secret=1",
  ])("rejects unsafe base %s", (value) => {
    expect(() => safeMediaBase(value, origin)).toThrow();
  });
});
