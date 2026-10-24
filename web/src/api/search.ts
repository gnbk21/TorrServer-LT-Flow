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
}
export function normalizeSearchItem(raw: RawSearchItem): SearchResultItem {
  let magnet = raw.Magnet || "";
  if (!magnet && raw.Hash && /^(?:[a-f0-9]{40}|[a-f0-9]{64})$/i.test(raw.Hash))
    magnet = torrentMagnet(raw.Hash);
  return {
    title: raw.Title || raw.Name || raw.Hash || "",
    magnet,
    link: raw.Link,
    size: raw.Size ?? 0,
    seeders: raw.Seed ?? 0,
    leechers: raw.Peer ?? 0,
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
  | "cancelled";
export interface SearchOutcome {
  source: SearchSource;
  index: number;
  state: SearchState;
}
export interface SearchProgress {
  items: SearchResultItem[];
  sources: SearchOutcome[];
}
const states: SearchState[] = [
  "pending",
  "ok",
  "empty",
  "timeout",
  "auth_required",
  "unavailable",
  "invalid_response",
  "cancelled",
];
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
      state: z.enum([
        "pending",
        "ok",
        "empty",
        "timeout",
        "auth_required",
        "unavailable",
        "invalid_response",
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
  const seen = new Set<string>();
  return rows.filter((row) => {
    try {
      const ids = new URL(row.magnet).searchParams
        .getAll("xt")
        .map(canonicalIdentity)
        .filter((v) =>
          /^urn:btih:[a-f0-9]{40}$|^urn:btmh:1220[a-f0-9]{64}$/.test(v),
        );
      if (ids.some((id) => seen.has(id))) return false;
      ids.forEach((id) => seen.add(id));
    } catch {
      /* Unknown identity must not merge similarly named releases. */
    }
    return true;
  });
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
  ): Promise<SearchResultItem[]> => {
    const sources = source === "all" ? enabled : [source];
    if (!sources.length) throw new Error("search.noSources");
    const jobs = sources.flatMap((item) =>
      item === "torznab" && indices?.length
        ? indices.map((i) => ({ source: item, index: i }))
        : [{ source: item, index: index ?? -1 }],
    );
    const results: SearchResultItem[][] = jobs.map(() => []);
    const outcomes: SearchOutcome[] = jobs.map((job) => ({
      ...job,
      state: "pending",
    }));
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
          sources: outcomes.map((row) => ({ ...row })),
        });
    };
    let next = 0;
    const worker = async () => {
      while (next < jobs.length) {
        const slot = next++;
        const job = jobs[slot]!;
        const item = job.source;
        try {
          if (controller.signal.aborted) throw controller.signal.reason;
          const params = new URLSearchParams({
            query,
            index: String(job.index),
            details: "true",
          });
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
          results[slot] = rows.map(normalizeSearchItem);
          const reported = !Array.isArray(valid)
            ? valid.sources?.[0]?.state
            : undefined;
          outcomes[slot] = {
            ...job,
            state:
              reported && states.includes(reported)
                ? reported
                : rows.length
                  ? "ok"
                  : "empty",
          };
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
          outcomes[slot] = { ...job, state };
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
