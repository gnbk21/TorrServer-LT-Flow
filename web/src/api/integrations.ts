import { api } from "./client";
import type { TMDBSettings } from "../types/settings";
export interface WafSettings {
  whitelist: string;
  blacklist: string;
  referers: string;
  default_referers_enabled: boolean;
  default_referers: string[];
  ip_enabled: boolean;
  referer_enabled: boolean;
  read_only: boolean;
  warnings: { list: string; line?: number; code: string; reason?: string }[];
}
export interface StorageSettings {
  settings: "json" | "bbolt";
  viewed: "json" | "bbolt";
}
export interface GstSettings {
  built_in: boolean;
  config?: Record<string, string | number | boolean>;
  defaults?: Record<string, string | number | boolean>;
}
export interface GstHealth {
  found: boolean;
  available: boolean;
  works: boolean;
  version?: string;
  error?: string;
}
export interface GstProbe {
  Tracks: {
    Index: number;
    Type: string;
    Title: string;
    Language: string;
    Codec: string;
  }[];
}
export const integrationsApi = {
  getGstMasterUrl: (hash: string, index: number, audio = -1) =>
    `${window.location.origin}/gst/${encodeURIComponent(hash)}/master.m3u8?${new URLSearchParams({ index: String(index), audio: String(audio) })}`,
  getGstHeartbeat: (hash: string, signal?: AbortSignal) =>
    api.get(`/gst/${encodeURIComponent(hash)}/heartbeat`, signal),
  getGstHealth: (signal?: AbortSignal) =>
    api.get<Record<string, GstHealth>>("/gst/echo", signal),
  getGstProbe: (hash: string, index: number, signal?: AbortSignal) =>
    api.get<GstProbe>(
      `/gst/${encodeURIComponent(hash)}/probe?index=${index}`,
      signal,
      // Server budget: two 33-second attempts plus 45-second cold warmup.
      120_000,
    ),
  testTorznab: (host: string, key = "", signal?: AbortSignal) =>
    api.post<
      { host: string; key: string },
      { success: boolean; error?: string }
    >("/torznab/test", { host, key }, signal),
  testJacred: (host: string, key = "", signal?: AbortSignal) =>
    api.post<
      { host: string; key: string },
      { success: boolean; error?: string }
    >("/jacred/test", { host, key }, signal),
  getWAF: (signal?: AbortSignal) => api.get<WafSettings>("/waf", signal),
  updateWAF: (
    data: Pick<
      WafSettings,
      "whitelist" | "blacklist" | "referers" | "default_referers_enabled"
    >,
    signal?: AbortSignal,
  ) => api.post<typeof data, WafSettings>("/waf", data, signal),
  getTMDB: (signal?: AbortSignal) =>
    api.get<TMDBSettings>("/tmdb/settings", signal),
  getStorage: (signal?: AbortSignal) =>
    api.get<StorageSettings>("/storage/settings", signal),
  setStorage: (data: StorageSettings) =>
    api.post<StorageSettings, { status: string }>("/storage/settings", data),
  getGst: (signal?: AbortSignal) =>
    api.get<GstSettings>("/gst/settings", signal),
  setGst: (config: NonNullable<GstSettings["config"]>) =>
    api.post("/gst/settings", { action: "set", config }),
};
