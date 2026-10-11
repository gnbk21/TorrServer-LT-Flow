import { api } from "./client";
import { flowSchema, networkSchema } from "./schemas";
import {
  FlowNetworkStatus,
  FlowStatusResponse,
  FlowTrayStatus,
} from "../types/flow";

export const flowApi = {
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
      .then((data) => flowSchema.parse(data) as FlowStatusResponse),

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
