import { afterEach, expect, it, vi } from "vitest";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import {
  useFlowDiagnostics,
  useFlow,
  useActiveTorrent,
  useActive,
} from "./queries";
import { flowApi } from "../api/flow";
import { libraryApi } from "../api/library";
import type { ActiveSnapshot } from "../types/library";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

function fixture() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  let signal: AbortSignal | undefined;
  const request = vi
    .spyOn(flowApi, "getStatus")
    .mockImplementation((_, incoming) => {
      signal = incoming;
      return new Promise((_, reject) =>
        incoming?.addEventListener(
          "abort",
          () => reject(new DOMException("Cancelled", "AbortError")),
          { once: true },
        ),
      );
    });
  return { client, wrapper, request, signal: () => signal };
}

it("cancels a diagnostic request when its drawer closes", async () => {
  const f = fixture();
  const hook = renderHook(
    ({ enabled }) => useFlowDiagnostics("fixture", enabled),
    { wrapper: f.wrapper, initialProps: { enabled: true } },
  );
  await waitFor(() => expect(f.request).toHaveBeenCalledOnce());
  hook.rerender({ enabled: false });
  await waitFor(() => expect(f.signal()?.aborted).toBe(true));
  f.client.clear();
});

it("cancels on backgrounding and resumes only when visible", async () => {
  const f = fixture();
  let hidden = false;
  vi.spyOn(document, "hidden", "get").mockImplementation(() => hidden);
  renderHook(() => useFlowDiagnostics("fixture", true), { wrapper: f.wrapper });
  await waitFor(() => expect(f.request).toHaveBeenCalledOnce());
  act(() => {
    hidden = true;
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await waitFor(() => expect(f.signal()?.aborted).toBe(true));
  expect(f.request).toHaveBeenCalledOnce();
  act(() => {
    hidden = false;
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await waitFor(() => expect(f.request).toHaveBeenCalledTimes(2));
  f.client.clear();
});

it("aborts compact Flow work on backgrounding and starts no hidden request", async () => {
  const f = fixture();
  let hidden = true;
  vi.spyOn(document, "hidden", "get").mockImplementation(() => hidden);
  renderHook(() => useFlow("fixture"), { wrapper: f.wrapper });
  expect(f.request).not.toHaveBeenCalled();
  act(() => {
    hidden = false;
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await waitFor(() => expect(f.request).toHaveBeenCalledOnce());
  act(() => {
    hidden = true;
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await waitFor(() => expect(f.signal()?.aborted).toBe(true));
  f.client.clear();
});

it("isolates an idle card from another torrent's live updates using one batched query", async () => {
  const f = fixture();
  const snapshot = (bytes: number): ActiveSnapshot =>
    ({
      sampled_at: new Date().toISOString(),
      items: [
        {
          torrent: {
            hash: "playing",
            title: "Fixture",
            stat: 3,
            download_speed: bytes,
          },
          status: { hash: "playing" },
        },
      ],
    }) as ActiveSnapshot;
  f.client.setQueryData(["active"], snapshot(1));
  const request = vi.spyOn(libraryApi, "active").mockResolvedValue(snapshot(1));
  let idleRenders = 0;
  const idle = renderHook(
    () => {
      idleRenders++;
      return useActiveTorrent("idle").data;
    },
    { wrapper: f.wrapper },
  );
  const playing = renderHook(() => useActiveTorrent("playing").data, {
    wrapper: f.wrapper,
  });
  const complete = renderHook(() => useActive().data, { wrapper: f.wrapper });
  await waitFor(() => expect(request).toHaveBeenCalledOnce());
  const before = idleRenders;
  act(() => f.client.setQueryData(["active"], snapshot(2)));
  await waitFor(() =>
    expect(complete.result.current?.items[0]?.torrent.download_speed).toBe(2),
  );
  expect(playing.result.current?.torrent.download_speed).toBe(2);
  expect(idle.result.current).toBeUndefined();
  expect(idleRenders).toBe(before);
  expect(request).toHaveBeenCalledOnce();
  f.client.clear();
});

it("keeps one polling owner when multiple cards observe the shared snapshot", async () => {
  vi.useFakeTimers();
  const f = fixture();
  const data: ActiveSnapshot = {
    sampled_at: new Date().toISOString(),
    items: [
      {
        torrent: { hash: "playing", title: "Fixture", stat: 3 },
        status: { hash: "playing" },
      },
    ],
  };
  const request = vi.spyOn(libraryApi, "active").mockResolvedValue(data);
  renderHook(
    () => {
      useActive();
      useActiveTorrent("idle");
      useActiveTorrent("playing");
      useActiveTorrent("other");
    },
    { wrapper: f.wrapper },
  );
  await act(async () => {
    await vi.advanceTimersByTimeAsync(10);
  });
  expect(request).toHaveBeenCalledOnce();
  await act(async () => {
    await vi.advanceTimersByTimeAsync(3000);
  });
  expect(request).toHaveBeenCalledTimes(4);
  f.client.clear();
});
