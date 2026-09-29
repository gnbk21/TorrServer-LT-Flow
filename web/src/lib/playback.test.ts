import { describe, it, expect } from "vitest";
import { buildAndroidIntent } from "./intent";
import { rankServerAddresses, safePairingUrl } from "./network";
import { humanizeBytes, parseEpisodeInfo } from "./format";
describe("playback links", () => {
  it("preserves HTTPS, escapes intent delimiters and supplies a browser fallback", () => {
    const value = buildAndroidIntent(
      "https://[fd00::2]:8091/stream?link=a%26b&index=2&ss=abc;x",
    );
    expect(value).toContain("scheme=https;package=com.brouken.player;");
    expect(value).toContain("ss=abc%3Bx#Intent");
    expect(value).toContain("S.browser_fallback_url=https%3A%2F%2F");
  });
  it("rejects credentials and non-HTTP playback", () => {
    expect(() =>
      buildAndroidIntent("https://user:secret@host/stream"),
    ).toThrow();
    expect(() => buildAndroidIntent("javascript:alert(1)")).toThrow();
  });
});
describe("LAN pairing", () => {
  it("ranks current LAN origin ahead of other private adapters and keeps TLS", () => {
    const addresses = rankServerAddresses(
      ["192.168.1.2", "10.0.0.2", "::1"],
      undefined,
      "https://10.0.0.2:8091",
    );
    expect(addresses[0]?.url).toBe("https://10.0.0.2:8091");
    expect(addresses.at(-1)?.isLoopback).toBe(true);
  });
  it("formats IPv6 and preserves default ports", () => {
    expect(
      rankServerAddresses(["fd00::2"], undefined, "https://localhost")[0]?.url,
    ).toBe("https://[fd00::2]");
  });
  it("refuses loopback, URL credentials and token-bearing QR links", () => {
    for (const value of [
      "http://127.2.3.4:8090",
      "http://[::1]",
      "http://u:p@192.168.1.2",
      "http://192.168.1.2/?token=abc",
    ])
      expect(() => safePairingUrl(value)).toThrow();
  });
});
describe("media formatting", () => {
  it.each([
    "Show.S01E02.mkv",
    "Show.1x02.mkv",
    "Show Season 01 Episode 02.mkv",
  ])("parses %s", (name) =>
    expect(parseEpisodeInfo(name)).toMatchObject({ season: 1, episode: 2 }),
  );
  it("supports episode-only names but preserves uncertain/raw names", () => {
    expect(parseEpisodeInfo("E01.mkv").episode).toBe(1);
    expect(parseEpisodeInfo("Movie.2024.1080p.mkv")).toEqual({
      isEpisode: false,
      displayTitle: "Movie.2024.1080p.mkv",
    });
  });
  it("never formats invalid numbers as real telemetry", () => {
    expect(humanizeBytes(undefined)).toBe("—");
    expect(humanizeBytes(Infinity)).toBe("—");
    expect(humanizeBytes(0.5)).toBe("1 B");
  });
});
