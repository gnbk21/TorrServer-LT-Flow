import { z } from "zod";
import { api } from "./client";
const positionSchema = z.object({
  timecode: z.number().finite().nonnegative(),
  position_source: z.string().optional(),
  position_updated_at: z.string().optional(),
  completed: z.boolean().optional(),
});
export const recentSchema = z.object({
  schema_version: z.literal(1),
  enabled: z.boolean(),
  read_only: z.boolean(),
  items: z
    .array(
      z.object({
        hash: z.string().regex(/^(?:[a-f0-9]{40}|[a-f0-9]{64})$/i),
        file_index: z.number().int().positive(),
        last_opened_at: z.string(),
        file_path: z.string(),
        title: z.string(),
        viewed: positionSchema.optional(),
      }),
    )
    .max(200),
});
export const recentApi = {
  get: async (signal?: AbortSignal) =>
    recentSchema.parse(await api.get("/flow/recent", signal)),
  remove: (hash: string, file_index: number) =>
    api.post(
      "/flow/recent",
      { action: "remove", hash, file_index },
      undefined,
      true,
    ),
  clear: () => api.post("/flow/recent", { action: "clear" }, undefined, true),
};
