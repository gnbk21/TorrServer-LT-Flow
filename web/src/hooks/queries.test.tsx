import { afterEach, expect, it, vi } from "vitest";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useFlowDiagnostics } from "./queries";
import { flowApi } from "../api/flow";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
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
