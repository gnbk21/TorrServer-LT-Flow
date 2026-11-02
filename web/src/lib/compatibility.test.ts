import { describe, expect, it } from "vitest";
import { browserAdvice } from "./compatibility";
describe("browser-only advisory", () => {
  it("does not infer MKV compatibility or external-player capability", () =>
    expect(browserAdvice("matroska", "hevc", () => "probably")).toBe(
      "unknown",
    ));
  it("distinguishes reported family support from missing evidence", () => {
    expect(browserAdvice("mov,mp4", "h264", () => "probably")).toBe("possible");
    expect(browserAdvice("webm", "vp9", () => "")).toBe("unconfirmed");
    expect(browserAdvice("mov,mp4", undefined, () => "probably")).toBe(
      "unknown",
    );
  });
});
