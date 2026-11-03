import { z } from "zod";
import { api, ApiError } from "./client";
import { torrentMagnet } from "../lib/torrentLink";
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
  SeedKnown?: boolean;
  SizeKnown?: boolean;
}
export function normalizeSearchItem(raw: RawSearchItem): SearchResultItem {
  let magnet = raw.Magnet || "";
  if (!magnet && raw.Hash && /^(?:[a-f0-9]{40}|[a-f0-9]{64})$/i.test(raw.Hash))
    magnet = torrentMagnet(raw.Hash);
  return {
    title: raw.Title || raw.Name || raw.Hash || "",
    magnet,
    link: raw.Link,
    size: raw.SizeKnown === false ? null : (raw.Size ?? null),
    seeders: raw.SeedKnown === false ? null : (raw.Seed ?? null),
    leechers: raw.Peer ?? null,
    sources: raw.Tracker ? [raw.Tracker] : [],
    category: raw.Categories,
    date: raw.CreateDate,
    tracker: raw.Tracker,
  };
}
export type SearchState =
  | "pending"
  | "ok"
  | "empty"
  | "timeout"
  | "auth_required"
  | "unavailable"
  | "invalid_response"
  | "unsupported_query"
  | "cancelled";
export interface SearchOutcome {
  elapsed_ms?: number;
  query_mode?: string;
  source: SearchSource;
  index: number;
  state: SearchState;
}
export interface SearchProgress {
  items: SearchResultItem[];
  sources: SearchOutcome[];
}
const rawSearchSchema = z
  .object({
    Title: z.string().optional(),
    Name: z.string().optional(),
    Hash: z.string().optional(),
    Magnet: z.string().optional(),
    Link: z.string().optional(),
    Size: z.union([z.number(), z.string()]).optional(),
    Seed: z.number().optional(),
    Peer: z.number().optional(),
    Categories: z.string().optional(),
    CreateDate: z.string().optional(),
    Tracker: z.string().optional(),
    SeedKnown: z.boolean().optional(),
    SizeKnown: z.boolean().optional(),
  })
  .passthrough()
  .refine((row) =>
    [row.Title, row.Name, row.Hash].some((value) => typeof value === "string"),
  )
  .transform((row) => ({ ...row, Title: row.Title ?? "" }));
export const searchEnvelopeSchema = z.object({
  items: z.array(rawSearchSchema),
  sources: z.array(
    z.object({
      source: z.enum(["rutor", "torznab", "jacred"]),
      index: z.number().int(),
      elapsed_ms: z.number().nonnegative().optional(),
      query_mode: z.string().max(32).optional(),
      state: z.enum([
        "pending",
        "ok",
        "empty",
        "timeout",
        "auth_required",
        "unavailable",
        "invalid_response",
        "unsupported_query",
        "cancelled",
      ]),
    }),
  ),
});
function canonicalIdentity(xt: string): string {
  const text = xt.toLowerCase();
  if (!/^urn:btih:[a-z2-7]{32}$/.test(text)) return text;
  const alphabet = "abcdefghijklmnopqrstuvwxyz234567";
  let value = 0,
    bits = 0,
    hex = "";
  for (const letter of text.slice(9)) {
    value = (value << 5) | alphabet.indexOf(letter);
    bits += 5;
    if (bits >= 8) {
      bits -= 8;
      hex += ((value >>> bits) & 255).toString(16).padStart(2, "0");
    }
  }
  return `urn:btih:${hex}`;
}
export function deduplicateResults(
  rows: SearchResultItem[],
): SearchResultItem[] {
  const seen = new Map<string, SearchResultItem>();
  const grouped: SearchResultItem[] = [];
  for (const original of rows) {
    const row = {
      ...original,
      sources: [
        ...new Set([
          ...(original.sources ?? []),
          ...(original.tracker ? [original.tracker] : []),
        ]),
      ],
    };
    try {
      const ids = new URL(row.magnet).searchParams
        .getAll("xt")
        .map(canonicalIdentity)
        .filter((v) =>
          /^urn:btih:[a-f0-9]{40}$|^urn:btmh:1220[a-f0-9]{64}$/.test(v),
        );
      const duplicates = [
        ...new Set(
          ids
            .map((id) => seen.get(id))
            .filter((item): item is SearchResultItem => !!item),
        ),
      ];
      const existing = duplicates[0];
      if (existing) {
        existing.sources = [
          ...new Set([
            ...(existing.sources ?? []),
            ...row.sources,
            ...duplicates.flatMap((item) => item.sources ?? []),
          ]),
        ];
        for (const duplicate of duplicates.slice(1)) {
          const index = grouped.indexOf(duplicate);
          if (index >= 0) grouped.splice(index, 1);
          for (const [id, value] of seen)
            if (value === duplicate) seen.set(id, existing);
        }
        ids.forEach((id) => seen.set(id, existing));
        continue;
      }
      ids.forEach((id) => seen.set(id, row));
    } catch {
      /* Unknown identity must not merge similarly named releases. */
    }
    grouped.push(row);
  }
  return grouped;
}
export interface StructuredSearch {
  type?: "search" | "movie" | "tv";
  imdbid?: string;
  tmdbid?: string;
  tvdbid?: string;
  year?: string;
  season?: string;
  ep?: string;
}
export const searchApi = {
  search: async (
    query: string,
    source: SearchSource | "all" = "all",
    index?: number,
    signal?: AbortSignal,
    enabled: SearchSource[] = [],
    progress?: (value: SearchProgress) => void,
    indices?: number[],
    specification?: StructuredSearch,
  ): Promise<SearchResultItem[]> => {
    const sources = source === "all" ? enabled : [source];
    if (!sources.length) throw new Error("search.noSources");
    const jobs = sources.flatMap((item) =>
      item === "torznab" && indices?.length
        ? indices.map((i) => ({ source: item, index: i }))
        : [{ source: item, index: index ?? -1 }],
    );
    const results: SearchResultItem[][] = jobs.map(() => []);
    const outcomes: SearchOutcome[][] = jobs.map((job) => [
      {
        ...job,
        state: "pending",
      },
    ]);
    const controller = new AbortController();
    const abort = () => controller.abort(signal?.reason);
    if (signal?.aborted) abort();
    else signal?.addEventListener("abort", abort, { once: true });
    const timeout = setTimeout(
      () =>
        controller.abort(new DOMException("Search timed out", "TimeoutError")),
      30000,
    );
    const publish = () => {
      if (!signal?.aborted)
        progress?.({
          items: deduplicateResults(results.flat()),
          sources: outcomes.flat().map((row) => ({ ...row })),
        });
    };
    // Older servers ignore structured identifiers. Never turn an ID-only
    // request into their unrestricted empty-text search. Title searches remain
    // immediate; this presence check is shared only by ID-only endpoint jobs.
    let structuredSupport: Promise<boolean> | undefined;
    const supportsStructuredSearch = () =>
      (structuredSupport ??= api
        .get<unknown>("/flow/search-status", controller.signal)
        .then((data) => {
          if (
            !data ||
            typeof data !== "object" ||
            !("schema_version" in data) ||
            data.schema_version !== 1
          )
            throw new Error("search.invalidResponse");
          return true;
        })
        .catch((error: unknown) => {
          if (error instanceof ApiError && [404, 405].includes(error.status))
            return false;
          throw error;
        }));
    let next = 0;
    const worker = async () => {
      while (next < jobs.length) {
        const slot = next++;
        const job = jobs[slot]!;
        const item = job.source;
        if (!query.trim() && item !== "torznab") {
          outcomes[slot] = [{ ...job, state: "unsupported_query" }];
          publish();
          continue;
        }
        try {
          if (controller.signal.aborted) throw controller.signal.reason;
          if (!query.trim() && !(await supportsStructuredSearch())) {
            outcomes[slot] = [{ ...job, state: "unsupported_query" }];
            publish();
            continue;
          }
          const params = new URLSearchParams({
            query,
            index: String(job.index),
            details: "true",
          });
          if (item === "torznab")
            for (const [key, value] of Object.entries(specification ?? {}))
              if (value) params.set(key, value);
          const path = item === "rutor" ? "/search/" : `/${item}/search/`;
          const data = await api.get<
            | RawSearchItem[]
            | { items: RawSearchItem[]; sources: SearchOutcome[] }
          >(`${path}?${params}`, controller.signal);
          const parsed = Array.isArray(data)
            ? z.array(rawSearchSchema).safeParse(data)
            : searchEnvelopeSchema.safeParse(data);
          if (!parsed.success) throw new Error("search.invalidResponse");
          const valid = parsed.data;
          const rows = Array.isArray(valid) ? valid : valid.items;
          if (
            !Array.isArray(rows) ||
            rows.some(
              (row) =>
                !row ||
                typeof row !== "object" ||
                [row.Title, row.Name, row.Hash].every(
                  (value) => typeof value !== "string",
                ),
            )
          )
            throw new Error("search.invalidResponse");
          results[slot] = rows.map((raw) => {
            const result = normalizeSearchItem(raw);
            if (!result.sources?.length) result.sources = [item];
            return result;
          });
          const reported = !Array.isArray(valid)
            ? valid.sources.filter(
                (row) =>
                  row.source === item &&
                  (job.index === -1 || row.index === job.index),
              )
            : [];
          outcomes[slot] = reported.length
            ? reported
            : [{ ...job, state: rows.length ? "ok" : "empty" }];
        } catch (error) {
          const state: SearchState = controller.signal.aborted
            ? controller.signal.reason?.name === "TimeoutError"
              ? "timeout"
              : "cancelled"
            : error instanceof ApiError && [401, 403].includes(error.status)
              ? "auth_required"
              : error instanceof Error &&
                  error.message === "search.invalidResponse"
                ? "invalid_response"
                : "unavailable";
          outcomes[slot] = [{ ...job, state }];
        }
        publish();
      }
    };
    publish();
    try {
      await Promise.all(
        Array.from({ length: Math.min(4, jobs.length) }, worker),
      );
      if (signal?.aborted)
        throw signal.reason ?? new DOMException("Cancelled", "AbortError");
      return deduplicateResults(results.flat());
    } finally {
      clearTimeout(timeout);
      signal?.removeEventListener("abort", abort);
    }
  },
};
