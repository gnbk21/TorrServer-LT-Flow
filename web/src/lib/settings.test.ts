import { describe, it, expect } from "vitest";
import {
  flattenSettings,
  mergeSettings,
  validateSettings,
  settingsUnit,
} from "./settings";
import type { BTSettings } from "../types/settings";
describe("settings round trip and bounds", () => {
  it("labels byte rates and durations using the native settings units", () => {
    expect(settingsUnit("DownloadRateLimit")).toBe("KiB/s");
    expect(settingsUnit("Flow.StreamTransportBufferKiB")).toBe("KiB");
    expect(settingsUnit("Flow.ManagementRateLimit")).toBe("requests/min");
    expect(settingsUnit("Flow.ProbeGraceMs")).toBe("ms");
    expect(settingsUnit("Flow.SwarmCustom.MinReconnectTime")).toBe("s");
    expect(settingsUnit("Flow.StartupSafetyFactorPct")).toBe("%");
    expect(settingsUnit("ConnectionsLimit")).toBe("");
  });
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
  it("keeps untouched null, absent and empty lists during a hot preference change", () => {
    for (const lists of [
      {},
      { TrustedProxies: null, TorznabUrls: null },
      { TrustedProxies: [], TorznabUrls: [] },
    ]) {
      const baseline = {
        CacheSize: 2147483648,
        ForceEncrypt: true,
        ...lists,
        Flow: { PublicMetadataCache: false },
      } as unknown as BTSettings;
      const flat = flattenSettings(baseline);
      flat["Flow.PublicMetadataCache"] = true;
      expect(mergeSettings(baseline, flat)).toEqual({
        ...baseline,
        Flow: { PublicMetadataCache: true },
      });
    }
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
  it("rejects quoted active disk paths while preserving unquoted and inactive paths", () => {
    for (const path of [
      '"C:\\cache folder"',
      '"C:\\cache',
      'C:\\cache"',
      '  "C:\\cache"  ',
    ]) {
      const values = { UseDisk: true, TorrentsSavePath: path };
      expect(validateSettings(values).TorrentsSavePath).toBe(
        "settings.pathNoQuotes",
      );
      expect(
        validateSettings({ ...values, UseDisk: false }).TorrentsSavePath,
      ).toBeUndefined();
      expect(values.TorrentsSavePath).toBe(path);
    }
    for (const path of [
      "C:\\cache folder",
      "\\\\server\\share\\cache",
      "/tmp/cache",
      "relative cache",
    ])
      expect(
        validateSettings({ UseDisk: true, TorrentsSavePath: path })
          .TorrentsSavePath,
      ).toBeUndefined();
  });
});
