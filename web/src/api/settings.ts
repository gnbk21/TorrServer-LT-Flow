import { api } from "./client";
import { settingsSchema } from "./schemas";
import { BTSettings } from "../types/settings";
import { z } from "zod";

const stateSchema = z.object({
  revision: z.string(),
  saved: settingsSchema,
  effective: settingsSchema.nullable(),
  pending: settingsSchema.optional(),
  error: z.string().optional(),
  recovery: z.object({
    source: z.string(),
    issue: z.string().optional(),
    schema_version: z.number(),
  }),
  data_path: z.string(),
  executable: z.string(),
});
export type ConfigurationState = z.infer<typeof stateSchema>;

export const settingsApi = {
  state: (signal?: AbortSignal) =>
    api
      .post("/settings", { action: "state" }, signal)
      .then((data) => stateSchema.parse(data)),
  plan: (sets: BTSettings) =>
    api.post<
      { action: string; sets: BTSettings },
      { restart_required: boolean; active_work: boolean }
    >("/settings", { action: "plan", sets }),
  apply: (sets: BTSettings, revision: string, when: "now" | "idle") =>
    api.post("/settings", { action: "set", sets, revision, when }),
  cancelPending: (revision: string) =>
    api.post("/settings", { action: "cancel_pending", revision }),
  get: (signal?: AbortSignal) =>
    api
      .post<{ action: string }, unknown>("/settings", { action: "get" }, signal)
      .then((data) => settingsSchema.parse(data) as BTSettings),

  set: (sets: BTSettings, signal?: AbortSignal) =>
    api.post<{ action: string; sets: BTSettings }, void>(
      "/settings",
      { action: "set", sets },
      signal,
    ),

  // This resets persisted settings and restarts the engine; never use for reads.
  reset: (signal?: AbortSignal) =>
    api.post<{ action: string }, void>("/settings", { action: "def" }, signal),
};
