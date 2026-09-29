import { api } from "./client";
import { RuntimeStatus } from "../types/runtime";

export const runtimeApi = {
  getEcho: (signal?: AbortSignal) => api.get<string>("/echo", signal),

  getStatus: (signal?: AbortSignal) =>
    api.get<RuntimeStatus>("/runtime/status", signal),

  shutdown: (signal?: AbortSignal) => api.get<void>("/shutdown", signal),
};
