import { z } from "zod";
import { api } from "./client";

export const preparationSchema = z.object({
  jobs: z
    .array(
      z
        .object({
          id: z.string(),
          hash: z.string().regex(/^[a-f0-9]{40}$/),
          file_index: z.number().int().positive(),
          state: z.enum([
            "downloading",
            "ready",
            "paused",
            "cancelled",
            "error",
            "cleaning",
          ]),
          length: z.number().positive(),
          verified_bytes: z.number().nonnegative(),
          contiguous_bytes: z.number().nonnegative(),
          playback_ready: z.boolean(),
          error_code: z.string().optional(),
        })
        .refine(
          (j) =>
            j.verified_bytes <= j.length &&
            j.contiguous_bytes <= j.verified_bytes &&
            (!j.playback_ready || j.verified_bytes === j.length),
          "invalid preparation progress",
        ),
    )
    .max(16),
  quota_bytes: z.number().nonnegative(),
  reserved_bytes: z.number().nonnegative(),
  error_code: z.string().optional(),
});
export type PreparationStatus = z.infer<typeof preparationSchema>;
export type PreparationAction =
  "start" | "pause" | "resume" | "cancel" | "remove";
export const preparationApi = {
  status: (signal?: AbortSignal) =>
    api
      .get<unknown>("/flow/preparation", signal)
      .then((d) => preparationSchema.parse(d)),
  control: (hash: string, file_index: number, action: PreparationAction) =>
    api
      .post<object, unknown>("/flow/preparation", { hash, file_index, action })
      .then((d) => preparationSchema.parse(d)),
};
