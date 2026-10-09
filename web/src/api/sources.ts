import { z } from "zod";
import { api } from "./client";
export const sourcesSchema = z.object({
  private: z.boolean(),
  sources: z
    .array(
      z.object({
        id: z.string().regex(/^[a-f0-9]{32}$/),
        origin: z.string(),
        disabled: z.boolean(),
        allow_local: z.boolean(),
      }),
    )
    .max(16),
});
export const sourcesApi = {
  get: (hash: string, signal?: AbortSignal) =>
    api
      .get<unknown>(`/flow/sources/${encodeURIComponent(hash)}`, signal)
      .then((d) => sourcesSchema.parse(d)),
  update: (
    hash: string,
    value: {
      action: "add" | "remove";
      url?: string;
      id?: string;
      allow_local?: boolean;
    },
  ) =>
    api
      .post<object, unknown>(`/flow/sources/${encodeURIComponent(hash)}`, value)
      .then((d) => sourcesSchema.parse(d)),
};
