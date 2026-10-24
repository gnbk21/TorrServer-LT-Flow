import { z } from "zod";
import { api } from "./client";
import { observeFlow } from "../lib/observation";
import { flowSchema, networkSchema } from "./schemas";
import {
  FlowNetworkStatus,
  FlowStatusResponse,
  FlowTrayStatus,
} from "../types/flow";

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

export const flowApi = {
  getSessions: (signal?: AbortSignal) =>
    api
      .get("/flow/sessions", signal)
      .then((value) => summariesSchema.parse(value)),
  retryMetadata: (hash: string, signal?: AbortSignal) =>
    api.post(`/flow/metadata-retry/${encodeURIComponent(hash)}`, {}, signal),
  warmup: (
    hash: string,
    file_index: number,
    action: "select" | "cancel" | "auto",
    signal?: AbortSignal,
  ) =>
    api.post<object, NonNullable<FlowStatusResponse["next_episode_warmup"]>>(
      `/flow/warmup/${encodeURIComponent(hash)}`,
      { file_index, action },
      signal,
    ),
  playbackLink: async (hash: string, index: number, signal?: AbortSignal) => {
    const data = await api.post<object, { path: string; expires_at: string }>(
      "/flow/playback-link",
      { hash, index },
      signal,
    );
    if (
      !/^\/flow\/play\/[A-Za-z0-9_.-]+$/.test(data.path) ||
      !Number.isFinite(Date.parse(data.expires_at))
    )
      throw new Error("Invalid playback capability response");
    return data;
  },
  getStatus: (hash: string, signal?: AbortSignal, includeTraces = false) =>
    api
      .get<unknown>(
        `/flow/status/${encodeURIComponent(hash)}${includeTraces ? "" : "?traces=false"}`,
        signal,
      )
      .then((data) =>
        observeFlow(flowSchema.parse(data) as FlowStatusResponse),
      ),

  getNetwork: (signal?: AbortSignal) =>
    api
      .get<unknown>("/flow/network", signal)
      .then((data) => networkSchema.parse(data) as FlowNetworkStatus),

  getTray: (signal?: AbortSignal) =>
    api.get<FlowTrayStatus>("/flow/tray", signal),

  control: (action: "pause" | "resume", signal?: AbortSignal) =>
    api.post<{ action: string }, FlowTrayStatus>(
      "/flow/control",
      { action },
      signal,
    ),
};
