import { useEffect, useState } from "react";
import { useQuery, useQueries, QueryClient } from "@tanstack/react-query";
import { api, ApiError } from "../api/client";
import { torrentsApi } from "../api/torrents";
import { flowApi } from "../api/flow";
import { runtimeApi } from "../api/runtime";
import { settingsApi } from "../api/settings";
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (count, error) =>
        !(
          error instanceof ApiError &&
          [400, 401, 403, 404].includes(error.status)
        ) && count < 1,
      staleTime: 1500,
      refetchOnWindowFocus: true,
      refetchIntervalInBackground: false,
      gcTime: 120_000,
    },
  },
});
export function useVisible() {
  const [visible, setVisible] = useState(!document.hidden);
  useEffect(() => {
    const update = () => setVisible(!document.hidden);
    document.addEventListener("visibilitychange", update);
    return () => document.removeEventListener("visibilitychange", update);
  }, []);
  return visible;
}
export function useLibrary() {
  const visible = useVisible();
  return useQuery({
    queryKey: ["torrents"],
    queryFn: ({ signal }) => torrentsApi.list(signal),
    refetchInterval: (query) =>
      !visible
        ? false
        : query.state.data?.some(
              (t) =>
                (t.active_readers ?? 0) > 0 || t.stat === 1 || t.stat === 2,
            )
          ? 1000
          : 5000,
  });
}
export function useRuntime() {
  const visible = useVisible();
  return useQuery({
    queryKey: ["runtime"],
    queryFn: ({ signal }) => runtimeApi.getStatus(signal),
    refetchInterval: visible ? 5000 : false,
  });
}
export function useNetwork() {
  const visible = useVisible();
  return useQuery({
    queryKey: ["network"],
    queryFn: ({ signal }) => flowApi.getNetwork(signal),
    refetchInterval: visible ? 5000 : false,
  });
}
export function useSettings() {
  return useQuery({
    queryKey: ["settings"],
    queryFn: ({ signal }) => settingsApi.get(signal),
    staleTime: 30_000,
  });
}
export function useFlow(hash?: string) {
  const visible = useVisible();
  return useQuery({
    queryKey: ["flow", hash],
    queryFn: ({ signal }) => flowApi.getStatus(hash!, signal),
    enabled: !!hash,
    refetchInterval: visible ? 1000 : false,
  });
}
export function useFlows(hashes: string[]) {
  const visible = useVisible();
  return useQueries({
    queries: hashes.map((hash) => ({
      queryKey: ["flow", hash],
      queryFn: ({ signal }: { signal: AbortSignal }) =>
        flowApi.getStatus(hash, signal),
      refetchInterval: visible ? 1000 : false,
    })),
  });
}
export function useVersion() {
  return useQuery({
    queryKey: ["version"],
    queryFn: ({ signal }) => runtimeApi.getEcho(signal),
    staleTime: 60_000,
  });
}
export interface CacheState {
  Capacity?: number;
  Filled?: number;
  PiecesLength?: number;
  PiecesCount?: number;
  Pieces?: Record<
    string,
    {
      Id: number;
      Length: number;
      Size: number;
      Completed: boolean;
      Priority: number;
    }
  >;
  Readers?: { Start: number; End: number; Reader: number }[];
}
export function useCache(hash: string, enabled: boolean) {
  const visible = useVisible();
  return useQuery({
    queryKey: ["cache", hash],
    queryFn: ({ signal }) =>
      api.post<{ action: string; hash: string }, CacheState>(
        "/cache",
        { action: "get", hash },
        signal,
      ),
    enabled,
    refetchInterval: visible ? 3000 : false,
  });
}
export async function refreshLibrary() {
  await queryClient.invalidateQueries({ queryKey: ["torrents"] });
  await queryClient.invalidateQueries({ queryKey: ["runtime"] });
}
