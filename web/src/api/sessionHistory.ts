import { z } from "zod";
import { api } from "./client";
export const summariesSchema = z.object({
  enabled: z.boolean(),
  truncated: z.boolean(),
  items: z
    .array(
      z.object({
        started: z.string().datetime(),
        last_observed: z.string().datetime(),
        samples: z.number().nonnegative(),
        bytes_served: z.number().nonnegative(),
        peak_recent_read_stalls: z.number().nonnegative(),
        peak_recent_wait_p95_ms: z.number().nonnegative(),
        minimum_buffer_bytes: z.number().nonnegative(),
        maximum_buffer_bytes: z.number().nonnegative(),
        range_requests: z.number().nonnegative(),
        range_cancels: z.number().nonnegative(),
        incomplete: z.boolean(),
      }),
    )
    .max(64),
});
export type SessionSummaries = z.infer<typeof summariesSchema>;

export const sessionHistoryApi = {
  getSessions: (signal?: AbortSignal) =>
    api
      .get("/flow/sessions", signal)
      .then((value) => summariesSchema.parse(value)),
};
