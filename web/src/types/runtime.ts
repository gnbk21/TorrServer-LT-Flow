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
  dlna_enabled: boolean;
  bonjour_enabled: boolean;
  friendly_name: string;
  webdav_enabled: boolean;
  webdav_path: string;
  fuse_path: string;
  fuse_enabled: boolean;
  bt?: ClientStatus | null;
}
