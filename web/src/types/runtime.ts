export interface TorrentStatusBrief {
  hash: string;
  title: string;
  name?: string;
  stat: number;
  stat_string: string;
  torrent_size: number;
  loaded_size: number;
  download_speed: number;
  upload_speed: number;
  active_peers: number;
  total_peers: number;
  connected_seeders: number;
  preloaded_bytes: number;
  preload_size: number;
}

export interface ClientStatus {
  upload_cap_utilization?: number;
  upload_cap_busy?: boolean;
  wan_queue_delay_known?: boolean;
  listen_port: number;
  active_streams: number;
  torrent_count: number;
  total_size: number;
  loaded_size: number;
  active_peers: number;
  total_peers: number;
  connected_seeders: number;
  bytes_read: number;
  bytes_written: number;
  download_speed: number;
  upload_speed: number;
  torrents: TorrentStatusBrief[];
  raw_stat: string;
}

export interface RuntimeStatus {
  resources?: {
    budget_bytes: number;
    warm_budget_bytes: number;
    memory_pressure: boolean;
    protected_overcommit: boolean;
    background_limited: boolean;
    ram_resident_bytes: number;
    disk_cache_bytes: number;
  };
  exposure?: {
    http_auth: boolean;
    legacy_playback: boolean;
    security_profile: string;
    management_rate_limit: number;
    explicit_origin_policy: boolean;
    http_port: string;
    https: boolean;
    torrent_interface: {
      name: string;
      state: string;
      addresses: string[];
      required: boolean;
      dns_verified: boolean;
      leak_protection_verified: boolean;
    };
    vpn_leak_protection_verified: boolean;
  };
  memory?: {
    rss_bytes: number;
    rss_available: boolean;
    go_heap_bytes: number;
    go_system_bytes: number;
    goroutines: number;
    handles: number;
    handles_available: boolean;
    sampled_at: string;
    system_total_bytes?: number;
    system_available_bytes?: number;
    system_available?: boolean;
    pressure?: boolean;
  };
  cache_allocation?: {
    caches: number;
    active_caches: number;
    idle_caches: number;
    warm_caches: number;
    resident_bytes: number;
    active_resident_bytes: number;
    idle_resident_bytes: number;
    warm_resident_bytes: number;
    effective_capacity_bytes: number;
    protected_bytes: number;
    active_readers: number;
  };
  startup?: {
    started_at: string;
    listeners_ready: boolean;
    engine_ready: boolean;
    listener_ready_ms: number;
    engine_ready_ms: number;
  };
  dlna_enabled: boolean;
  bonjour_enabled: boolean;
  friendly_name: string;
  webdav_enabled: boolean;
  webdav_path: string;
  fuse_path: string;
  fuse_enabled: boolean;
  bt?: ClientStatus | null;
}
