import { describe, it, expect } from "vitest";
import { torrentMagnet } from "./torrentLink";

describe("complete torrent identities", () => {
  it("keeps legacy links and emits BEP 52 multihashes for v2", () => {
    const v1 = "ab".repeat(20);
    const v2 = "cd".repeat(32);
    expect(torrentMagnet(v1)).toBe(`magnet:?xt=urn:btih:${v1}`);
    expect(torrentMagnet(v2.toUpperCase(), "show & episode")).toBe(
      `magnet:?xt=urn:btmh:1220${v2}&dn=show%20%26%20episode`,
    );
    expect(() => torrentMagnet("x".repeat(64))).toThrow();
    expect(() => torrentMagnet("cd".repeat(31))).toThrow();
  });
});
