import type { Torrent } from "./torrent";
import type { FlowStatusResponse } from "./flow";

export interface LibraryFilter {
  q: string;
  category: string;
  sort: string;
  page: number;
  limit: number;
}
export interface LibraryPage {
  items: Torrent[];
  total: number;
  library_total: number;
  page: number;
  limit: number;
  categories: string[];
  sampled_at: string;
}
export interface ActiveSnapshot {
  items: { torrent: Torrent; status: FlowStatusResponse }[];
  sampled_at: string;
}
