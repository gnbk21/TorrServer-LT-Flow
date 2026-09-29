import { api } from "./client";
import { settingsSchema } from "./schemas";
import { BTSettings } from "../types/settings";

export const settingsApi = {
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
