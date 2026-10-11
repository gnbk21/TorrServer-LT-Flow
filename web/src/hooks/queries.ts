import { useCallback, useEffect, useSyncExternalStore } from "react";
import {
  useQuery,
  useQueries,
  useQueryClient,
  QueryClient,
} from "@tanstack/react-query";
import { api, ApiError } from "../api/client";
import { torrentsApi } from "../api/torrents";
import { flowApi } from "../api/flow";
import { runtimeApi } from "../api/runtime";
import { settingsApi } from "../api/settings";
import { libraryApi } from "../api/library";
import type { ActiveSnapshot, LibraryFilter } from "../types/library";
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
const visibilitySubscribers = new Set<() => void>();
const publishVisibility = () =>
  visibilitySubscribers.forEach((notify) => notify());
function subscribeVisibility(notify: () => void) {
  if (!visibilitySubscribers.size)
    document.addEventListener("visibilitychange", publishVisibility);
  visibilitySubscribers.add(notify);
  return () => {
    visibilitySubscribers.delete(notify);
    if (!visibilitySubscribers.size)
      document.removeEventListener("visibilitychange", publishVisibility);
  };
}
export function useVisible() {
  return useSyncExternalStore(
    subscribeVisibility,
    () => !document.hidden,
    () => true,
  );
}
// Disabling polling does not abort an already running request. Read-only
// observers use this gate so hidden pages also release network work promptly.
export function useVisibilityGate(key: readonly unknown[], enabled = true) {
  const visible = useVisible();
  const client = useQueryClient();
  useEffect(() => {
    if (!visible || !enabled)
      void client.cancelQueries({ queryKey: key, exact: true });
  }, [client, key, enabled, visible]);
  return visible && enabled;
}
export function useLibrary() {
  const visible = useVisibilityGate(["torrents"]);
  return useQuery({
    queryKey: ["torrents"],
    queryFn: ({ signal }) => torrentsApi.list(signal),
    enabled: visible,
    refetchInterval: (query) =>
      !visible
        ? false
        : (query.state.data?.length ?? 0) > 200
          ? 5000
          : query.state.data?.some(
                (t) =>
                  (t.active_readers ?? 0) > 0 || t.stat === 1 || t.stat === 2,
              )
            ? 1000
            : 5000,
  });
}
export function useLibraryPage(filter: LibraryFilter) {
  const visible = useVisibilityGate(["library", filter]);
  return useQuery({
    queryKey: ["library", filter],
    queryFn: ({ signal }) => libraryApi.page(filter, signal),
    enabled: visible,
    refetchInterval: visible ? 5000 : false,
  });
}
export function useActive<T = ActiveSnapshot>(
  select?: (data: ActiveSnapshot) => T,
  poll = true,
) {
  const visible = useVisibilityGate(["active"]);
  return useQuery<ActiveSnapshot, Error, T>({
    queryKey: ["active"],
    queryFn: ({ signal }) => libraryApi.active(signal),
    enabled: visible,
    select,
    refetchOnMount: poll,
    refetchOnWindowFocus: poll,
    refetchOnReconnect: poll,
    refetchInterval: (query) =>
      !visible || !poll ? false : query.state.data?.items.length ? 1000 : 5000,
  });
}
export function useActiveTorrent(hash: string) {
  const select = useCallback(
    (data: ActiveSnapshot) =>
      data.items.find((item) => item.torrent.hash === hash),
    [hash],
  );
  // The page owns the polling interval; cards only select from its shared
  // cache. Giving every card a timer can duplicate fast completed requests.
  return useActive(select, false);
}
export function useRuntime() {
  const visible = useVisibilityGate(["runtime"]);
  return useQuery({
    queryKey: ["runtime"],
    queryFn: ({ signal }) => runtimeApi.getStatus(signal),
    enabled: visible,
    refetchInterval: visible ? 5000 : false,
  });
}
export function useNetwork() {
  const visible = useVisibilityGate(["network"]);
  return useQuery({
    queryKey: ["network"],
    queryFn: ({ signal }) => flowApi.getNetwork(signal),
    enabled: visible,
    refetchInterval: visible ? 5000 : false,
  });
}
export function useSettings() {
  const visible = useVisibilityGate(["settings"]);
  return useQuery({
    queryKey: ["settings"],
    queryFn: ({ signal }) => settingsApi.get(signal),
    enabled: visible,
    staleTime: 30_000,
  });
}
export function useConfiguration() {
  const visible = useVisibilityGate(["configuration"]);
  return useQuery({
    queryKey: ["configuration"],
    queryFn: ({ signal }) => settingsApi.state(signal),
    enabled: visible,
    refetchInterval: visible ? 5000 : false,
  });
}
export function useFlow(hash?: string) {
  const visible = useVisibilityGate(["flow", hash], !!hash);
  return useQuery({
    queryKey: ["flow", hash],
    queryFn: ({ signal }) => flowApi.getStatus(hash!, signal),
    enabled: visible,
    refetchInterval: visible ? 1000 : false,
  });
}
export function useFlows(hashes: string[]) {
  const visible = useVisible();
  const client = useQueryClient();
  useEffect(() => {
    if (!visible)
      for (const hash of hashes)
        void client.cancelQueries({ queryKey: ["flow", hash], exact: true });
  }, [client, hashes, visible]);
  return useQueries({
    queries: hashes.map((hash) => ({
      queryKey: ["flow", hash],
      queryFn: ({ signal }: { signal: AbortSignal }) =>
        flowApi.getStatus(hash, signal),
      enabled: visible,
      refetchInterval: visible ? 1000 : false,
    })),
  });
}
export function useFlowDiagnostics(hash: string | undefined, enabled: boolean) {
  const visible = useVisible();
  const client = useQueryClient();
  useEffect(() => {
    if (!hash) return;
    const cancel = () => {
      void client.cancelQueries({
        queryKey: ["flow-diagnostics", hash],
        exact: true,
      });
    };
    if (!enabled || !visible) cancel();
    return cancel;
  }, [client, hash, enabled, visible]);
  return useQuery({
    queryKey: ["flow-diagnostics", hash],
    queryFn: ({ signal }) => flowApi.getStatus(hash!, signal, true),
    enabled: enabled && visible && !!hash,
    refetchInterval: enabled && visible ? 3000 : false,
  });
}
export function useVersion() {
  const visible = useVisibilityGate(["version"]);
  return useQuery({
    queryKey: ["version"],
    queryFn: ({ signal }) => runtimeApi.getEcho(signal),
    enabled: visible,
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
  const visible = useVisibilityGate(["cache", hash], enabled);
  return useQuery({
    queryKey: ["cache", hash],
    queryFn: ({ signal }) =>
      api.post<{ action: string; hash: string }, CacheState>(
        "/cache",
        { action: "get", hash },
        signal,
      ),
    enabled: visible,
    refetchInterval: visible ? 3000 : false,
  });
}
export async function refreshLibrary() {
  await Promise.all(
    ["torrents", "library", "active", "runtime"].map((key) =>
      queryClient.invalidateQueries({ queryKey: [key] }),
    ),
  );
}
