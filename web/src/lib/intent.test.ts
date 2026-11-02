import { expect, it } from "vitest";
import { buildAndroidIntent } from "./intent";
it("hands Just Player an explicit bounded integer position without invented subtitle extras", () => {
  const value = buildAndroidIntent(
    "https://example.invalid/flow/play/token/media.mkv?ss=abc",
    "just",
    42.9876,
  );
  expect(value).toContain("i.position=42987;");
  expect(value).toContain("ss=abc");
  expect(value).not.toContain("S.subs");
  for (const bad of [-1, NaN, Infinity, 3e6])
    expect(
      buildAndroidIntent("https://example.invalid/media.mkv", "just", bad),
    ).not.toContain("i.position");
  expect(
    buildAndroidIntent("https://example.invalid/media.mkv", "vlc", 42),
  ).not.toContain("i.position");
});
