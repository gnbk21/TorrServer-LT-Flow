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
        RateAwareDeadlines: z.boolean().optional(),
        CapacityAwareRequests: z.boolean().optional(),
        AdaptiveUrgentHorizon: z.boolean().optional(),
        ContainerBurstHints: z.boolean().optional(),
        NextEpisodeWarmup: z.boolean().optional(),
        StreamTransportBufferKiB: z
          .union([z.literal(0), z.literal(64), z.literal(256), z.literal(1024)])
          .optional(),
        SwarmCustom: z.record(z.string(), z.unknown()),
      })
      .passthrough()
      .nullable()
      .optional(),
  })
  .passthrough();
export const flowSessionSchema = z
  .object({
    session_id: z
      .string()
      .regex(/^[A-Za-z0-9_-]{16,64}$/)
      .optional(),
    supply_qualified: z.boolean().optional(),
    readable_contiguous_bytes: z.number().int().nonnegative().optional(),
    verified_contiguous_bytes: z.number().int().nonnegative().optional(),
    frontier_growth_rate: z.number().nonnegative().optional(),
    burst_hint_source: z.string().optional(),
    delivery: z
      .object({
        mode: z.string(),
        short_rate: z.number().nonnegative(),
        long_rate: z.number().nonnegative(),
        short_samples: z.number().int().nonnegative(),
        samples: z.number().int().nonnegative(),
        age_ms: z.number(),
        confidence: z.string(),
        variation: z.number().nonnegative(),
        outage_seconds: z.number().int().nonnegative(),
        recent_outage_seconds: z.number().int().nonnegative().optional(),
        deficit_bytes: z.number().int().nonnegative().optional(),
        deficit_samples: z.number().int().nonnegative().optional(),
      })
      .passthrough()
      .optional(),
    risk: z
      .object({
        level: z.string(),
        reason: z.string(),
        score: z.number().min(0).max(100),
        confidence: z.string(),
        target_seconds: z.number().int().nonnegative(),
        exhaustion_seconds: z.number().nonnegative().optional(),
      })
      .optional(),
    wait_reason: z.string().optional(),
    required_piece_suppliers: z.number().int().nonnegative().optional(),
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
    next_episode_warmup: z
      .object({
        enabled: z.boolean(),
        current_file_index: z.number().int().nonnegative(),
        file_index: z.number().int().nonnegative(),
        automatic: z.boolean(),
        state: z.enum([
          "idle",
          "waiting",
          "warming",
          "ready",
          "evicted",
          "unavailable",
        ]),
        reason: z.string(),
        budget_bytes: z
          .number()
          .int()
          .min(0)
          .max(32 * 1024 * 1024),
        verified_bytes: z
          .number()
          .int()
          .min(0)
          .max(32 * 1024 * 1024),
      })
      .optional(),
    sampled_at: z.string().datetime({ offset: true }).optional(),
    timeline: z
      .object({
        source: z.literal("server"),
        dropped: z.number().int().nonnegative(),
        events: z
          .array(
            z.object({
              time: z.string().datetime({ offset: true }),
              elapsed_ms: z.number().int().nonnegative(),
              operation_ms: z.number().int(),
              type: z.string(),
              stage: z.string().optional(),
              file: z.number().int().optional(),
              bytes: z.number().int().optional(),
            }),
          )
          .max(128),
      })
      .optional(),
    storage_io: z
      .object({
        queued_bytes: z.number().int().nonnegative(),
        peak_queue_bytes: z.number().int().nonnegative(),
        queued_jobs: z.number().int().nonnegative(),
        rejected_jobs: z.number().int().nonnegative(),
        completed_jobs: z.number().int().nonnegative(),
        wait_p95_us: z.number().int().nonnegative(),
        wait_max_us: z.number().int().nonnegative(),
        callback_p95_us: z.number().int().nonnegative(),
        callback_max_us: z.number().int().nonnegative(),
      })
      .passthrough()
      .optional(),
    sparse: z
      .object({
        known: z.boolean(),
        urgent_truncated: z.boolean().optional(),
        request_age_truncated: z.boolean().optional(),
        urgent: z
          .array(
            z
              .object({
                piece: z.number().int().nonnegative(),
                priority: z.number().int().min(0).max(7),
                blocks: z.number().int().min(1).max(8192),
                unrequested: z.number().int().nonnegative(),
                requested: z.number().int().nonnegative(),
                writing: z.number().int().nonnegative(),
                finished: z.number().int().nonnegative(),
                duplicate_requests: z.number().int().nonnegative(),
                verified: z.boolean(),
                receiving_blocks: z.number().int().nonnegative(),
                receiving_bytes: z.number().int().nonnegative(),
                oldest_request_age_ms: z.number().int().min(-1).optional(),
              })
              .refine(
                (p) =>
                  p.unrequested + p.requested + p.writing + p.finished ===
                  p.blocks,
              ),
          )
          .max(64)
          .refine(
            (pieces) => pieces.reduce((sum, p) => sum + p.blocks, 0) <= 8192,
          )
          .nullable()
          .optional(),
        request_timeouts: z.number().int().nonnegative().optional(),
        requests_dropped: z.number().int().nonnegative().optional(),
        private: z.boolean().optional(),
        useful_peers: z.number().int().min(0).max(512).optional(),
        useful_downloading_peers: z.number().int().min(0).max(512).optional(),
        choked_peers: z.number().int().min(0).max(512).optional(),
        snubbed_peers: z.number().int().min(0).max(512).optional(),
        pending_connections: z.number().int().min(0).max(512).optional(),
        tracker_peers: z.number().int().min(0).max(512).optional(),
        dht_peers: z.number().int().min(0).max(512).optional(),
        pex_peers: z.number().int().min(0).max(512).optional(),
        incoming_peers: z.number().int().min(0).max(512).optional(),
        outstanding_bytes: z.number().int().nonnegative().optional(),
        queued_blocks: z.number().int().nonnegative().optional(),
        max_queue_ms: z.number().int().nonnegative().optional(),
        failed_bytes: z.number().int().nonnegative().optional(),
        redundant_bytes: z.number().int().nonnegative().optional(),
        sampled_at_ms: z.number().int().nonnegative().optional(),
        sampled_peers: z.number().int().min(0).max(512).optional(),
        truncated: z.boolean().optional(),
        windows: z
          .array(
            z.object({
              first_piece: z.number().int().nonnegative(),
              availability: z.array(z.number().int().min(0).max(512)).max(128),
              unchoked_suppliers: z.number().int().min(0).max(512),
            }),
          )
          .max(8)
          .refine(
            (windows) =>
              windows.reduce(
                (sum, window) => sum + window.availability.length,
                0,
              ) <= 256,
          )
          .nullable()
          .optional(),
      })
      .passthrough()
      .optional(),
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

export const librarySchema = z.object({
  items: torrentSchema.array(),
  total: z.number().int().nonnegative(),
  library_total: z.number().int().nonnegative(),
  page: z.number().int().positive(),
  limit: z.number().int().min(1).max(100),
  categories: z.array(z.string()),
  sampled_at: z.string().datetime({ offset: true }),
});
export const activeSchema = z.object({
  items: z.array(z.object({ torrent: torrentSchema, status: flowSchema })),
  sampled_at: z.string().datetime({ offset: true }),
});

export const runtimeSchema = z
  .object({
    dlna_enabled: z.boolean(),
    bonjour_enabled: z.boolean(),
    friendly_name: z.string(),
    webdav_enabled: z.boolean(),
    webdav_path: z.string(),
    fuse_path: z.string(),
    fuse_enabled: z.boolean(),
    memory: z
      .object({
        rss_bytes: z.number().nonnegative(),
        rss_available: z.boolean(),
        go_heap_bytes: z.number().nonnegative(),
        goroutines: z.number().int().nonnegative(),
      })
      .passthrough()
      .optional(),
    cache_allocation: z
      .object({
        resident_bytes: z.number().nonnegative(),
        active_readers: z.number().int().nonnegative(),
        warm_caches: z.number().int().nonnegative(),
      })
      .passthrough()
      .optional(),
    startup: z
      .object({ listeners_ready: z.boolean(), engine_ready: z.boolean() })
      .passthrough()
      .optional(),
  })
  .passthrough();
