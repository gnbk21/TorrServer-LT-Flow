import { api } from "./client";
import { RuntimeStatus } from "../types/runtime";
import { runtimeSchema } from "./schemas";

export const runtimeApi = {
  getEcho: (signal?: AbortSignal) => api.get<string>("/echo", signal),

  getStatus: async (signal?: AbortSignal) =>
    runtimeSchema.parse(await api.get<RuntimeStatus>("/runtime/status", signal)) as RuntimeStatus,

  shutdown: (signal?: AbortSignal) => api.get<void>("/shutdown", signal),
};
