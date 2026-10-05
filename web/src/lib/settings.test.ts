import { describe, it, expect } from "vitest";
import { flattenSettings, mergeSettings, validateSettings } from "./settings";
import type { BTSettings } from "../types/settings";
describe("settings round trip and bounds", () => {
  const original = {
    CacheSize: 67108864,
    ReaderReadAHead: 95,
    PreloadCache: 10,
    TrustedProxies: ["127.0.0.1"],
    TorznabUrls: [
      { Host: "https://index.test", Key: "secret", Name: "Indexer" },
    ],
    FutureField: { Nested: ["preserve"] },
    Flow: {
      Enabled: false,
      RateAwareDeadlines: false,
      StartupBufferMinMB: 32,
      StartupBufferMaxMB: 128,
      BootstrapTailMode: "upstream-auto",
      SwarmCustom: { ConnectionSpeed: 0 },
    },
  } as unknown as BTSettings;
  it("preserves unknown fields, explicit false and custom zero values", () => {
    const flat = flattenSettings(original);
    flat.CacheSize = 134217728;
    const result = mergeSettings(original, flat);
    expect(result.FutureField).toEqual({ Nested: ["preserve"] });
    expect(result.Flow?.Enabled).toBe(false);
    expect(result.Flow?.RateAwareDeadlines).toBe(false);
    expect(result.Flow?.SwarmCustom.ConnectionSpeed).toBe(0);
    expect(result.TorznabUrls[0]?.Key).toBe("secret");
    expect(original.CacheSize).toBe(67108864);
  });
  it("saves an explicit deadline experiment without changing the original", () => {
    const flat = flattenSettings(original);
    flat["Flow.RateAwareDeadlines"] = true;
    expect(mergeSettings(original, flat).Flow?.RateAwareDeadlines).toBe(true);
    expect(original.Flow?.RateAwareDeadlines).toBe(false);
  });
  it("rejects non-integer, out-of-bound and cross-field values instead of normalizing", () => {
    const invalid = {
      ...flattenSettings(original),
      "Flow.StartupBufferMinMB": 256,
      "Flow.StartupBufferMaxMB": 128,
      "Flow.SwarmCustom.ConnectionSpeed": 501,
      ReaderReadAHead: NaN,
    };
    const errors = validateSettings(invalid);
    expect(errors["Flow.StartupBufferMaxMB"]).toBeDefined();
    expect(errors["Flow.SwarmCustom.ConnectionSpeed"]).toBeDefined();
    expect(errors.ReaderReadAHead).toBeDefined();
  });
});
