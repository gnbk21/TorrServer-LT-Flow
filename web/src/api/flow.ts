import { api } from "./client";
import { observeFlow } from "../lib/observation";
import { flowSchema, networkSchema } from "./schemas";
import {
  FlowNetworkStatus,
  FlowStatusResponse,
  FlowTrayStatus,
} from "../types/flow";

export const flowApi = {
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
  getStatus: async (
    hash: string,
    signal?: AbortSignal,
    includeTraces = false,
  ) => {
    const started = performance.now();
    const data = await api.get<unknown>(
      `/flow/status/${encodeURIComponent(hash)}${includeTraces ? "" : "?traces=false"}`,
      signal,
    );
    return observeFlow(
      flowSchema.parse(data) as FlowStatusResponse,
      undefined,
      performance.now() - started,
    );
  },

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
