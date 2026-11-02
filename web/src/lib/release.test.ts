import { expect, it } from "vitest";
import { releaseLabels, matchesRelease } from "./release";
it("parses bounded release tokens without upgrading unknown facts", () => {
  expect(releaseLabels("Film.2160p.WEB-DL.HEVC.DV.TrueHD.RUS")).toMatchObject({
    resolution: "2160p",
    codec: "HEVC",
    hdr: "Dolby Vision",
    audio: "TRUEHD",
    language: "RUS",
  });
  expect(Object.values(releaseLabels("Sample name")).filter(Boolean)).toEqual(
    [],
  );
  expect(matchesRelease(releaseLabels("Film.1080p"), { codec: "HEVC" })).toBe(
    false,
  );
  expect(matchesRelease(releaseLabels("Film.1080p"), {})).toBe(true);
});
