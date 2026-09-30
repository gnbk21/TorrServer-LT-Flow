import { z } from "zod";
// Unknown fields survive validation so newer servers remain compatible.
export const torrentSchema = z
  .object({
    hash: z.string().min(1),
    title: z.string(),
    stat: z.number().int(),
    file_stats: z
      .array(
        z
          .object({
            id: z.number().int().positive(),
            path: z.string(),
            length: z.number().nonnegative(),
          })
          .passthrough(),
      )
      .nullable()
      .optional(),
  })
  .passthrough();
export const settingsSchema = z
  .object({
    // Persisted legacy/partial configurations can report zero. Read it without
    // preventing settings repair; Apply still validates a positive budget.
    CacheSize: z.number().nonnegative(),
    ReaderReadAHead: z.number(),
    PreloadCache: z.number(),
    Flow: z
      .object({
        Enabled: z.boolean(),
        SwarmProfile: z.string(),
        SwarmCustom: z.record(z.string(), z.unknown()),
      })
      .passthrough()
      .nullable()
      .optional(),
  })
  .passthrough();
export const flowSessionSchema = z
  .object({
    recent_window_seconds: z.number().int().nonnegative().optional(),
    recent_piece_wait_p95_ms: z.number().nonnegative().optional(),
    recent_server_read_stalls: z.number().int().nonnegative().optional(),
    recent_download_rate: z.number().nonnegative().optional(),
    download_rate_samples: z.number().int().nonnegative().optional(),
    group: z.string(),
    file_index: z.number().int(),
    state: z.string(),
    active_readers: z.number(),
    buffer_ahead_seconds: z.number(),
    buffer_ahead_bytes: z.number(),
    playback_consumption_rate: z.number(),
    download_rate: z.number(),
    sustainability_ratio: z.number(),
    piece_wait_p95_ms: z.number(),
    seek_count: z.number(),
    seek_recovery_ms: z.number(),
    buffer_warning: z.boolean(),
  })
  .passthrough();
export const flowSchema = z
  .object({
    hash: z.string(),
    sessions: z.array(flowSessionSchema).nullable().optional(),
    startup: z
      .object({ state: z.string(), file_index: z.number() })
      .passthrough()
      .optional(),
  })
  .passthrough();
export const networkSchema = z
  .object({
    state: z.string(),
    connectivity: z.string(),
    addresses: z.array(z.string()).nullable(),
    checked_at: z.string(),
    changed_at: z.string(),
    next_check_seconds: z.number(),
    reannounce_count: z.number(),
  })
  .passthrough();

export const runtimeSchema = z.object({
  dlna_enabled: z.boolean(), bonjour_enabled: z.boolean(), friendly_name: z.string(),
  webdav_enabled: z.boolean(), webdav_path: z.string(), fuse_path: z.string(), fuse_enabled: z.boolean(),
  memory: z.object({ rss_bytes: z.number().nonnegative(), rss_available: z.boolean(), go_heap_bytes: z.number().nonnegative(), goroutines: z.number().int().nonnegative() }).passthrough().optional(),
  cache_allocation: z.object({ resident_bytes: z.number().nonnegative(), active_readers: z.number().int().nonnegative(), warm_caches: z.number().int().nonnegative() }).passthrough().optional(),
  startup: z.object({ listeners_ready: z.boolean(), engine_ready: z.boolean() }).passthrough().optional(),
}).passthrough();
