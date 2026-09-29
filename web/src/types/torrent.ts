export interface TorrentFile {
  id: number;
  path: string;
  length: number;
  viewed?: boolean;
  timecode?: number;
}

export interface Torrent {
  hash: string;
  name?: string;
  torrs_hash?: string;
  title: string;
  poster?: string;
  category?: string;
  data?: string;
  stat: number;
  stat_string?: string;
  size?: number;
  torrent_size?: number;
  loaded_size?: number;
  preload_size?: number;
  preloaded_bytes?: number;
  download_speed?: number;
  upload_speed?: number;
  connected_seeders?: number;
  connected_peers?: number;
  active_peers?: number;
  active_readers?: number;
  warm_idle?: boolean;
  flow_playback?: {
    file_index: number;
    buffer_seconds: number | null;
    sustainability: number | null;
    buffer_warning: boolean;
  }[];
  total_peers?: number;
  total_seeders?: number;
  file_stats?: TorrentFile[];
  timestamp?: number;
}

export const TorrentState = {
  ADDED: 0,
  GETTING_INFO: 1,
  IN_DB: 5,
  PRELOAD: 2,
  WORKING: 3,
  CLOSED: 4,
} as const;

export type TorrentStateType = (typeof TorrentState)[keyof typeof TorrentState];
