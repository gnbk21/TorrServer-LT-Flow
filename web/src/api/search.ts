import { api } from "./client";
import type { SearchResultItem } from "../types/search";
export type SearchSource = "rutor" | "torznab" | "jacred";
export interface RawSearchItem {
  Title: string;
  Name?: string;
  Magnet?: string;
  Link?: string;
  Hash?: string;
  Size?: number | string;
  Seed?: number;
  Peer?: number;
  Categories?: string;
  CreateDate?: string;
  Tracker?: string;
}
export function normalizeSearchItem(raw: RawSearchItem): SearchResultItem {
  return {
    title: raw.Title || raw.Name || raw.Hash || "",
    magnet: raw.Magnet || (raw.Hash ? `magnet:?xt=urn:btih:${raw.Hash}` : ""),
    link: raw.Link,
    size: raw.Size ?? 0,
    seeders: raw.Seed ?? 0,
    leechers: raw.Peer ?? 0,
    category: raw.Categories,
    date: raw.CreateDate,
    tracker: raw.Tracker,
  };
}
export const searchApi = {
  search: async (
    query: string,
    source: SearchSource | "all" = "all",
    index?: number,
    signal?: AbortSignal,
    enabled: SearchSource[] = [],
  ): Promise<SearchResultItem[]> => {
    const sources = source === "all" ? enabled : [source];
    if (!sources.length) throw new Error("search.noSources");
    const results = await Promise.all(
      sources.map(async (item) => {
        const params = new URLSearchParams({
          query,
          index: String(index ?? -1),
        });
        const path = item === "rutor" ? "/search/" : `/${item}/search/`;
        const data = await api.get<RawSearchItem[]>(
          `${path}?${params}`,
          signal,
        );
        if (!Array.isArray(data)) throw new Error("search.invalidResponse");
        return data.map(normalizeSearchItem);
      }),
    );
    return results.flat();
  },
};
