export interface FlowRangeTrace {
  timestamp: string;
  group: string;
  method: string;
  start: number;
  end: number;
  status: number;
  bytes_served: number;
  lifetime_ms: number;
  ttfb_ms: number;
  cancelled: boolean;
  classification: string;
}

export interface FlowStartup {
  wait_reason?: string;
  stage_started_ms?: number;
  elapsed_ms?: number;
  metadata_ready_ms?: number;
  first_dht_peer_ms?: number;
  first_peer_ms?: number;
  first_useful_block_ms?: number;
  file_index: number;
  state: string;
  bootstrap_head_target_bytes: number;
  bootstrap_tail_target_bytes: number;
  startup_target_bytes: number;
  bootstrap_complete_ms: number;
  probe_start_ms: number;
  probe_complete_ms: number;
  probe_success: boolean;
  startup_prebuffer_ms: number;
  time_to_first_byte_ms: number;
}

export interface FlowSession {
  readable_contiguous_bytes?: number;
  verified_contiguous_bytes?: number;
  frontier_growth_rate?: number;
  burst_hint_source?: string;
  supply_qualified?: boolean;
  delivery?: {
    mode: string;
    short_rate: number;
    long_rate: number;
    short_samples: number;
    samples: number;
    age_ms: number;
    confidence: string;
    variation: number;
    outage_seconds: number;
    recent_outage_seconds?: number;
    deficit_bytes?: number;
    deficit_samples?: number;
  };
  risk?: {
    level: string;
    reason: string;
    score: number;
    confidence: string;
    target_seconds: number;
    exhaustion_seconds?: number;
  };
  wait_reason?: string;
  required_piece_suppliers?: number;
  recent_window_seconds?: number;
  recent_cache_hit_bytes?: number;
  recent_cache_miss_bytes?: number;
  recent_piece_wait_count?: number;
  recent_piece_wait_p95_ms?: number;
  recent_server_read_stalls?: number;
  recent_download_rate?: number;
  download_rate_samples?: number;
  cache_hit_bytes: number;
  cache_miss_bytes: number;
  piece_wait_count: number;
  piece_wait_duration_ms: number;
  piece_wait_p50_ms: number;
  piece_wait_p95_ms: number;
  piece_wait_p99_ms: number;
  group: string;
  file_index: number;
  file_size: number;
  state: string;
  active_readers: number;
  playback_offset_bytes: number;
  playback_offset_seconds: number;
  buffer_ahead_bytes: number;
  buffer_ahead_seconds: number;
  buffer_exhaustion_seconds?: number | null;
  buffer_warning: boolean;
  estimated_media_bitrate: number;
  observed_playback_rate: number;
  observed_confidence: string;
  playback_consumption_rate: number;
  target_buffer_seconds: number;
  forward_window_pieces: number;
  bitrate_estimate_source: string;
  bitrate_estimate_confidence: string;
  download_rate: number;
  upload_rate: number;
  sustainability_ratio: number;
  cache_used: number;
  cache_size: number;
  connected_peers: number;
  range_request_count: number;
  range_cancel_count: number;
  seek_count: number;
  seek_recovery_ms: number;
  warm_reconnect_count: number;
  warm_reconnect_ttfb_ms: number;
  last_classification: string;
  last_ttfb_ms: number;
  traces?: FlowRangeTrace[];
}

export interface FlowNetworkStatus {
  peer_tcp_port?: number;
  peer_udp_port?: number;
  mapped_tcp_port?: number;
  mapped_udp_port?: number;
  mapping_successes?: number;
  mapping_errors?: number;
  listener_errors?: number;
  incoming_tcp?: number;
  incoming_utp?: number;
  incoming_ipv6?: number;
  state: string;
  connectivity: string;
  addresses: string[] | null;
  checked_at: string;
  changed_at: string;
  next_check_seconds: number;
  reannounce_count: number;
  last_tracker_reply?: string;
  last_tracker_error?: string;
  last_error?: string;
}

export interface FlowTrayStatus {
  server_state: string;
  active_torrent: string;
  download_rate: number;
  buffer_seconds: number;
  peer_count: number;
}

export interface FlowTrackerSummary {
  id: string;
  protocol: string;
  host: string;
  status: string;
  peers: number;
  error?: string;
  last_at: string;
}
export interface FlowStatusResponse {
  storage_io?: StorageIO;
  sparse?: SparseSnapshot;
  hash: string;
  startup?: FlowStartup;
  sessions?: FlowSession[] | null;
  trackers?: FlowTrackerSummary[] | null;
  network?: FlowNetworkStatus;
}

export interface StorageIO {
  queued_bytes: number;
  peak_queue_bytes: number;
  queued_jobs: number;
  rejected_jobs: number;
  completed_jobs: number;
  wait_p95_us: number;
  wait_max_us: number;
  callback_p95_us: number;
  callback_max_us: number;
}

export interface SparseSnapshot {
  request_age_truncated?: boolean;
  urgent_truncated?: boolean;
  urgent?: {
    piece: number;
    priority: number;
    blocks: number;
    unrequested: number;
    requested: number;
    writing: number;
    finished: number;
    duplicate_requests: number;
    verified: boolean;
    receiving_blocks: number;
    receiving_bytes: number;
    oldest_request_age_ms?: number;
  }[];
  request_timeouts?: number;
  requests_dropped?: number;
  known: boolean;
  private?: boolean;
  sampled_at_ms?: number;
  sampled_peers?: number;
  truncated?: boolean;
  useful_peers?: number;
  useful_downloading_peers?: number;
  choked_peers?: number;
  snubbed_peers?: number;
  pending_connections?: number;
  tracker_peers?: number;
  dht_peers?: number;
  pex_peers?: number;
  incoming_peers?: number;
  outstanding_bytes?: number;
  queued_blocks?: number;
  max_queue_ms?: number;
  failed_bytes?: number;
  redundant_bytes?: number;
  windows?: {
    first_piece: number;
    availability: number[];
    unchoked_suppliers: number;
  }[];
}
