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
    CacheSize: z.number().positive(),
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
