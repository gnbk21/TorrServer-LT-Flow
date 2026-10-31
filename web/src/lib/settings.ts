import type { BTSettings } from "../types/settings";
import type { SettingsMetadata } from "../api/settingsMetadata";
export const flowBounds: Record<string, readonly [number, number]> = {
  StreamTransportBufferKiB: [0, 1024],
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
// Tuning remains available through Advanced or search; choosing a view never
// changes the draft or omits hidden fields from the submitted configuration.
export const commonSettings = new Set([
  "CacheSize",
  "ReaderReadAHead",
  "PreloadCache",
  "UseDisk",
  "TorrentsSavePath",
  "RemoveCacheOnDrop",
  "FriendlyName",
  "TrackTimecode",
  "MergeAllM3U",
  "ForceEncrypt",
  "EnableIPv6",
  "DisableUpload",
  "DownloadRateLimit",
  "UploadRateLimit",
  "PeersListenPort",
  "Flow.TorrentInterface",
  "Flow.RequireTorrentInterface",
  "EnableDLNA",
  "EnableBonjour",
  "EnableRutorSearch",
  "EnableTorznabSearch",
  "TorznabUrls",
  "EnableJacRedSearch",
  "JacRedUrl",
  "JacRedKey",
  "SslPort",
  "Flow.RequirePlaybackToken",
  "Flow.Enabled",
  "Flow.AdaptiveStartup",
  "Flow.AdaptiveReadAhead",
  "Flow.SwarmProfile",
  "Flow.MetadataFallback",
  "Flow.PublicMetadataCache",
  "Flow.TargetBufferSeconds",
  "Flow.NextEpisodeWarmup",
  "Flow.MetricsEnabled",
  "Flow.DiagnosticHistory",
  "Flow.RangeTraceEnabled",
]);

export function settingsUnit(key: string): string {
  if (key === "CacheSize" || key.endsWith("MB")) return "MiB";
  if (key.endsWith("KiB")) return "KiB";
  if (key === "DownloadRateLimit" || key === "UploadRateLimit") return "KiB/s";
  if (key === "Flow.ManagementRateLimit") return "requests/min";
  if (/Ms$/.test(key)) return "ms";
  if (
    /(?:Seconds|Sec|Timeout|RequestQueueTime|ReconnectTime|PlaybackTokenTTL)$/.test(
      key,
    )
  )
    return "s";
  if (/(?:Pct|ReadAHead|PreloadCache)$/.test(key)) return "%";
  return "";
}
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
      // The server distinguishes nil and empty slices when planning a restart.
      // Preserve the original representation when this editor was not changed.
      if (String(value) === (original.TrustedProxies || []).join("\n"))
        continue;
      result.TrustedProxies = String(value)
        .split(/\r?\n/)
        .map((s) => s.trim())
        .filter(Boolean);
      continue;
    }
    if (path === "TorznabUrls") {
      if (String(value) === JSON.stringify(original.TorznabUrls || []))
        continue;
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
  metadata?: SettingsMetadata,
): Record<string, string> {
  const errors: Record<string, string> = {};
  if (metadata) {
    for (const [key, rule] of Object.entries(metadata.fields)) {
      const value = values[key];
      if (value === undefined || rule.read_only || rule.deprecated) continue;
      if (
        (rule.type === "integer" &&
          (!Number.isInteger(value) ||
            (rule.min !== undefined && Number(value) < rule.min) ||
            (rule.max !== undefined && Number(value) > rule.max))) ||
        (rule.choices && !rule.choices.includes(value as string | number))
      )
        errors[key] =
          rule.choices?.join(" / ") ?? `${rule.min ?? ""}–${rule.max ?? ""}`;
    }
  } else {
    if (
      values["Flow.StreamTransportBufferKiB"] !== undefined &&
      ![0, 64, 256, 1024].includes(
        Number(values["Flow.StreamTransportBufferKiB"]),
      )
    )
      errors["Flow.StreamTransportBufferKiB"] = "0 / 64 / 256 / 1024 KiB";
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
