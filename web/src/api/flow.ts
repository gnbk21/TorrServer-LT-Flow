import { api } from "./client";
import { flowSchema, networkSchema } from "./schemas";
import {
  FlowNetworkStatus,
  FlowStatusResponse,
  FlowTrayStatus,
} from "../types/flow";

export const flowApi = {
  getStatus: (hash: string, signal?: AbortSignal) =>
    api
      .get<unknown>(`/flow/status/${encodeURIComponent(hash)}`, signal)
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
