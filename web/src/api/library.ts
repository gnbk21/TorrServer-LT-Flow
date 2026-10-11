import { api, ApiError } from "./client";
import { torrentsApi } from "./torrents";
import { flowApi } from "./flow";
import { librarySchema, activeSchema } from "./schemas";
import type {
  ActiveSnapshot,
  LibraryFilter,
  LibraryPage,
} from "../types/library";

// Old servers can still host this UI. Only a missing endpoint selects the
// compatibility path; auth failures and malformed projections remain visible.
export const libraryApi = {
  page: async (
    filter: LibraryFilter,
    signal?: AbortSignal,
  ): Promise<LibraryPage> => {
    try {
      const query = new URLSearchParams(
        Object.entries(filter).map(([key, value]) => [key, String(value)]),
      );
      return librarySchema.parse(
        await api.get<unknown>(`/flow/library?${query}`, signal),
      ) as LibraryPage;
    } catch (error) {
      if (!(error instanceof ApiError) || error.status !== 404) throw error;
      const rows = await torrentsApi.list(signal);
      const matching = rows.filter(
        (row) =>
          (!filter.category ||
            (filter.category === "uncategorized"
              ? !row.category
              : `category:${row.category}` === filter.category)) &&
          `${row.title} ${row.name || ""}`
            .toLowerCase()
            .includes(filter.q.trim().toLowerCase()),
      );
      matching.sort(
        (a, b) =>
          (filter.sort === "title"
            ? a.title.toLowerCase().localeCompare(b.title.toLowerCase())
            : filter.sort === "size"
              ? (b.torrent_size || 0) - (a.torrent_size || 0)
              : (b.timestamp || 0) - (a.timestamp || 0)) ||
          a.hash.localeCompare(b.hash),
      );
      const page = Math.min(
        filter.page,
        Math.max(1, Math.ceil(matching.length / filter.limit)),
      );
      return {
        items: matching.slice((page - 1) * filter.limit, page * filter.limit),
        total: matching.length,
        library_total: rows.length,
        page,
        limit: filter.limit,
        categories: [
          ...new Set(rows.map((row) => row.category || "").filter(Boolean)),
        ].sort(),
        sampled_at: new Date().toISOString(),
      };
    }
  },
  active: async (signal?: AbortSignal): Promise<ActiveSnapshot> => {
    try {
      return activeSchema.parse(
        await api.get<unknown>("/flow/active", signal),
      ) as ActiveSnapshot;
    } catch (error) {
      if (!(error instanceof ApiError) || error.status !== 404) throw error;
      const rows = (await torrentsApi.list(signal)).filter(
        (row) =>
          (row.active_readers ?? (row.stat === 3 ? 1 : 0)) > 0 ||
          row.warm_idle ||
          row.stat === 1 ||
          row.stat === 2,
      );
      const items = await Promise.all(
        rows.map(async (torrent) => ({
          torrent,
          status: await flowApi.getStatus(torrent.hash, signal),
        })),
      );
      return { items, sampled_at: new Date().toISOString() };
    }
  },
};
