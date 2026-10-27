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
        FrontierRecovery: z.boolean().optional(),
        AdaptivePeerRequests: z.boolean().optional(),
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
    supply: z
      .object({
        known: z.boolean(),
        state: z.string(),
        demand_rate: z.number().nonnegative(),
        delivery_rate: z.number().nonnegative(),
        deficit_rate: z.number().nonnegative(),
        runway_seconds: z.number().nonnegative().optional(),
      })
      .optional(),
    blocker: z
      .object({
        known: z.boolean(),
        piece: z.number().int().nonnegative(),
        piece_length: z.number().int().nonnegative(),
        blocks: z.number().int().min(0).max(8192),
        unrequested: z.number().int().min(0).max(8192),
        requested: z.number().int().min(0).max(8192),
        writing: z.number().int().min(0).max(8192),
        finished: z.number().int().min(0).max(8192),
        oldest_request_age_ms: z.number().int().min(-1),
        suppliers: z.number().int().min(0).max(512).optional(),
        reason: z.string(),
      })
      .refine(
        (p) =>
          !p.known ||
          p.blocks === p.unrequested + p.requested + p.writing + p.finished,
      )
      .optional(),
    buffer_sample_age_ms: z.number().nonnegative().optional(),
    buffer_sampled_at_ms: z.number().int().nonnegative().optional(),
    buffer_offset_bytes: z.number().int().nonnegative().optional(),
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
    wait_explanation: z
      .object({
        reason: z.string(),
        confidence: z.enum(["unknown", "medium", "high"]),
        actions: z
          .array(
            z.enum([
              "CAPTURE_DIAGNOSTICS",
              "WAIT_METADATA",
              "CHECK_DISCOVERY",
              "CHECK_NETWORK",
              "BUILD_RESERVE",
              "WAIT_CONNECTION",
              "WAIT_SUPPLIERS",
              "WAIT_UNCHOKE",
              "WAIT_BLOCK",
            ]),
          )
          .max(3),
      })
      .optional(),
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
    integrity: z
      .object({
        failures: z.number().int().nonnegative(),
        recent_failures: z.number().int().nonnegative(),
        window_seconds: z.literal(60),
        last_failure_at_ms: z.number().int().nonnegative(),
      })
      .optional(),
    trackers: z
      .array(
        z.object({
          id: z.string().max(32),
          protocol: z.string().max(16),
          host: z.string().max(255),
          status: z.string().max(16),
          peers: z.number().int().nonnegative(),
          error: z.string().max(64).optional(),
          error_category: z
            .enum([
              "UNKNOWN",
              "SYSTEM",
              "GENERIC",
              "DNS",
              "ASIO",
              "TLS",
              "HTTP",
              "LIBTORRENT",
            ])
            .optional(),
          error_code: z.number().int().optional(),
          operation: z.number().int().optional(),
          failure_count: z.number().int().nonnegative().optional(),
          last_at: z.string().datetime({ offset: true }),
        }),
      )
      .max(64)
      .nullable()
      .optional(),
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
              operation_ms: z.number().int().nonnegative().nullable(),
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
        stream: z
          .object({
            request_policy: z
              .object({
                enabled: z.boolean(),
                sampled_peers: z.number().int().min(0).max(512),
                qualified_peers: z.number().int().min(0).max(512),
                capacity_limited_peers: z.number().int().min(0).max(512),
                desired_requests: z
                  .number()
                  .int()
                  .min(0)
                  .max(512 * 65535),
                queued_requests: z
                  .number()
                  .int()
                  .min(0)
                  .max(512 * 65535 * 2),
                capacity_requests: z
                  .number()
                  .int()
                  .min(0)
                  .max(512 * 65535),
                horizon_min_ms: z.number().int().min(0).max(6000),
                horizon_max_ms: z.number().int().min(0).max(6000),
                truncated: z.boolean(),
              })
              .refine(
                (p) =>
                  p.qualified_peers <= p.sampled_peers &&
                  p.capacity_limited_peers <= p.sampled_peers &&
                  p.desired_requests <= p.capacity_requests &&
                  p.horizon_min_ms <= p.horizon_max_ms &&
                  (p.qualified_peers === 0 || p.horizon_min_ms >= 250) &&
                  (p.qualified_peers > 0 || p.horizon_max_ms === 0),
              )
              .optional(),
            recovery_scan: z
              .object({
                age_ms: z.number().int().min(-1),
                eligible: z.number().int().min(0).max(8192),
                uncertain_request: z.number().int().min(0).max(8192),
                stale_service: z.number().int().min(0).max(8192),
                already_duplicated: z.number().int().min(0).max(8192),
                not_late: z.number().int().min(0).max(8192),
                receiving: z.number().int().min(0).max(8192),
              })
              .refine(
                (p) =>
                  p.eligible +
                    p.uncertain_request +
                    p.stale_service +
                    p.already_duplicated +
                    p.not_late +
                    p.receiving <=
                  8192,
              )
              .optional(),
            known: z.boolean(),
            first_dial_ms: z.number().int().min(-1),
            first_connect_ms: z.number().int().min(-1),
            first_handshake_ms: z.number().int().min(-1),
            first_payload_ms: z.number().int().min(-1),
            connect_duration_ms: z.number().int().min(-1),
            handshake_duration_ms: z.number().int().min(-1),
            useful_bytes: z.number().int().nonnegative(),
            redundant_bytes: z.number().int().nonnegative(),
            obsolete_bytes: z.number().int().nonnegative(),
            recovery_requests: z.number().int().nonnegative(),
            recovery_suppressed: z.number().int().nonnegative(),
            active_hedges: z.number().int().min(0).max(16),
            peak_hedges: z.number().int().min(0).max(16),
            active_hedge_bytes: z
              .number()
              .int()
              .min(0)
              .max(256 * 1024),
            budget_changes: z.number().int().nonnegative(),
            frontiers: z.number().int().min(0).max(8),
            recovery_scan_truncated: z.boolean(),
            service: z
              .array(
                z
                  .object({
                    transport: z.enum(["tcp", "utp", "unknown"]),
                    address_family: z.enum(["IPV4", "IPV6"]),
                    samples: z.number().int().nonnegative(),
                    p50_ms: z.number().int().min(0).max(300000),
                    p95_ms: z.number().int().min(0).max(300000),
                  })
                  .refine((row) => row.p95_ms >= row.p50_ms),
              )
              .max(6),
          })
          .optional(),
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
      .object({
        reserve: z
          .object({
            known: z.boolean(),
            reason: z.string(),
            base_bytes: z.number().int().nonnegative(),
            target_bytes: z.number().int().nonnegative(),
            maximum_bytes: z.number().int().nonnegative(),
            limited: z.boolean(),
            demand_rate: z.number().nonnegative(),
            delivery_rate: z.number().nonnegative(),
            deficit_rate: z.number().nonnegative(),
            forecast_seconds: z.number().int().min(0).max(60),
            additional_fill_seconds: z.number().nonnegative().optional(),
            requested_bytes: z.number().int().nonnegative().optional(),
            extra_wait_budget_ms: z.number().int().nonnegative().optional(),
            extra_wait_limited: z.boolean().optional(),
            extra_wait_expired: z.boolean().optional(),
          })
          .refine(
            (p) =>
              p.base_bytes <= p.target_bytes &&
              p.target_bytes <= p.maximum_bytes &&
              (p.requested_bytes === undefined ||
                p.target_bytes <= p.requested_bytes),
          )
          .optional(),
        state: z.string(),
        file_index: z.number(),
        phases: z
          .array(
            z.object({
              phase: z.enum([
                "DISCOVERY",
                "CONNECT",
                "HANDSHAKE",
                "METADATA",
                "FIRST_PAYLOAD",
                "BOOTSTRAP",
                "PROBE",
                "RESERVE",
                "FIRST_BYTE",
              ]),
              basis: z.enum([
                "NATIVE_TORRENT",
                "TORRENT",
                "PRELOAD",
                "OPERATION",
              ]),
              duration_ms: z.number().int().nonnegative().optional(),
            }),
          )
          .max(9)
          .optional(),
      })
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
    build: z
      .object({
        version: z.string().max(160),
        channel: z.enum(["stable", "preview", "development"]),
        source: z.string().max(64),
        modified: z.boolean(),
      })
      .optional(),
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
