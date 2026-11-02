import { z } from "zod";
import { api } from "./client";
const subtitleSchema = z.object({
  schema_version: z.literal(1),
  source: z.literal("torrent_metadata"),
  known: z.boolean(),
  truncated: z.boolean(),
  tracks: z
    .array(
      z.object({
        file_index: z.number().int().positive(),
        path: z.string().max(8192),
        size: z.number().positive(),
        format: z.enum(["srt", "ass", "ssa", "vtt"]),
        language: z.string().max(16).optional(),
        forced: z.boolean(),
        hearing_impaired: z.boolean(),
        match: z.enum(["exact", "episode", "ambiguous"]),
      }),
    )
    .max(100),
});
export const subtitlePath = (
  hash: string,
  index: number,
  track: number,
  format: "vtt" | "original" = "vtt",
) =>
  `/subs/${encodeURIComponent(hash)}/${index}?${new URLSearchParams({ track: String(track), format })}`;
export const subtitlesApi = {
  list: async (hash: string, index: number, signal?: AbortSignal) =>
    subtitleSchema.parse(
      await api.get(`/subs/${encodeURIComponent(hash)}/${index}`, signal),
    ),
  text: async (
    hash: string,
    index: number,
    track: number,
    signal?: AbortSignal,
  ) => {
    const value = await api.get<unknown>(
      subtitlePath(hash, index, track),
      signal,
      12000,
    );
    if (
      typeof value !== "string" ||
      value.length > 8 * 1024 * 1024 ||
      !value.startsWith("WEBVTT")
    )
      throw new Error("Invalid subtitle response");
    return value;
  },
};
