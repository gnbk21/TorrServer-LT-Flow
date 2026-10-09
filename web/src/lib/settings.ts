import type { BTSettings } from "../types/settings";
export const flowBounds: Record<string, readonly [number, number]> = {
  GlobalCacheBudgetMB: [0, 1048576],
  WarmCacheBudgetMB: [0, 65536],
  PreparationConcurrency: [1, 16],
  ManagementRateLimit: [0, 60000],
  PlaybackTokenTTL: [30, 86400],
  BootstrapHeadMB: [1, 128],
  ProbeGraceMs: [0, 10000],
  StartupBufferSeconds: [1, 60],
  StartupBufferMinMB: [1, 1024],
  StartupBufferMaxMB: [1, 2048],
  StartupSafetyFactorPct: [100, 300],
  TargetBufferSeconds: [10, 180],
  MaxBufferSeconds: [10, 600],
  PreparationQuotaMB: [64, 1048576],
  WarmSessionTimeoutSec: [30, 1800],
  NetworkRetryMinSec: [1, 60],
  NetworkRetryMaxSec: [1, 600],
  "SwarmCustom.ConnectionSpeed": [0, 500],
  "SwarmCustom.TorrentConnectBoost": [0, 500],
  "SwarmCustom.PeerConnectTimeout": [0, 120],
  "SwarmCustom.PieceTimeout": [0, 120],
  "SwarmCustom.RequestQueueTime": [0, 30],
  "SwarmCustom.MinReconnectTime": [0, 600],
};
export type SettingsValue = string | number | boolean;
export function flattenSettings(
  settings: BTSettings,
): Record<string, SettingsValue> {
  const flat: Record<string, SettingsValue> = {};
  const walk = (object: Record<string, unknown>, prefix = "") => {
    for (const [key, value] of Object.entries(object)) {
      const path = prefix + key;
      if (["string", "number", "boolean"].includes(typeof value))
        flat[path] = value as SettingsValue;
      else if (value && typeof value === "object" && !Array.isArray(value))
        walk(value as Record<string, unknown>, path + ".");
    }
  };
  walk(settings);
  flat.TrustedProxies = (settings.TrustedProxies || []).join("\n");
  flat.TorznabUrls = JSON.stringify(settings.TorznabUrls || []);
  return flat;
}
export function mergeSettings(
  original: BTSettings,
  values: Record<string, SettingsValue>,
): BTSettings {
  const result = structuredClone(original);
  for (const [path, value] of Object.entries(values)) {
    if (path === "TrustedProxies") {
      result.TrustedProxies = String(value)
        .split(/\r?\n/)
        .map((s) => s.trim())
        .filter(Boolean);
      continue;
    }
    if (path === "TorznabUrls") {
      result.TorznabUrls = JSON.parse(String(value));
      continue;
    }
    const parts = path.split(".");
    let target: Record<string, unknown> = result;
    for (const part of parts.slice(0, -1)) {
      if (!target[part] || typeof target[part] !== "object") target[part] = {};
      target = target[part] as Record<string, unknown>;
    }
    target[parts.at(-1)!] = value;
  }
  return result;
}
export function validateSettings(
  values: Record<string, SettingsValue>,
): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const [name, [min, max]] of Object.entries(flowBounds)) {
    const key = `Flow.${name}`;
    const value = values[key];
    if (
      value !== undefined &&
      (!Number.isInteger(value) || Number(value) < min || Number(value) > max)
    )
      errors[key] = `${min}–${max}`;
  }
  for (const [key, min, max] of [
    ["CacheSize", 1, 16 * 1024 * 1024 * 1024],
    ["ReaderReadAHead", 5, 100],
    ["PreloadCache", 0, 100],
    ["ConnectionsLimit", 1, 10000],
    ["DHTConnectionsLimit", 1, 100000],
    ["PeersListenPort", 0, 65535],
    ["SslPort", 0, 65535],
    ["RetrackersMode", 0, 3],
    ["TorrentDisconnectTimeout", 1, 86400],
    ["DownloadRateLimit", 0, Number.MAX_SAFE_INTEGER],
    ["UploadRateLimit", 0, Number.MAX_SAFE_INTEGER],
  ] as const) {
    const value = values[key];
    if (
      value !== undefined &&
      (!Number.isInteger(value) || Number(value) < min || Number(value) > max)
    )
      errors[key] = `${min}–${max}`;
  }
  for (const [min, max] of [
    ["StartupBufferMinMB", "StartupBufferMaxMB"],
    ["TargetBufferSeconds", "MaxBufferSeconds"],
    ["NetworkRetryMinSec", "NetworkRetryMaxSec"],
  ])
    if (Number(values[`Flow.${min}`]) > Number(values[`Flow.${max}`]))
      errors[`Flow.${max}`] = "settings.validationOrder";
  if (values.UseDisk && !String(values.TorrentsSavePath || "").trim())
    errors.TorrentsSavePath = "settings.pathRequired";
  const cachePath = String(values.TorrentsSavePath || "").trim();
  if (values.UseDisk && (cachePath.startsWith('"') || cachePath.endsWith('"')))
    errors.TorrentsSavePath = "settings.pathNoQuotes";
  if (
    values["Flow.BootstrapTailMode"] !== undefined &&
    values["Flow.BootstrapTailMode"] !== "upstream-auto"
  )
    errors["Flow.BootstrapTailMode"] = "settings.tailMode";
  return errors;
}
