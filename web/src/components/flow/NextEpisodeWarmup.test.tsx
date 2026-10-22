import { afterEach, expect, test, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NextEpisodeWarmup } from "./NextEpisodeWarmup";
import type { FlowStatusResponse } from "../../types/flow";
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
afterEach(cleanup);
test("warmup announces state changes without repeating polled byte counters", () => {
  const client = new QueryClient();
  const status = {
    hash: "fixture",
    next_episode_warmup: {
      enabled: true,
      state: "ready",
      automatic: false,
      reason: "",
      file_index: 2,
      current_file_index: 1,
      verified_bytes: 1024,
      budget_bytes: 4096,
    },
  } as FlowStatusResponse;
  const view = (value: FlowStatusResponse) => (
    <QueryClientProvider client={client}>
      <NextEpisodeWarmup status={value} />
    </QueryClientProvider>
  );
  const result = render(view(status));
  expect(screen.getByRole("status").textContent).toBe(
    "flow.warmupStates.ready",
  );
  result.rerender(
    view({
      ...status,
      next_episode_warmup: {
        ...status.next_episode_warmup!,
        verified_bytes: 2048,
      },
    }),
  );
  expect(screen.getByRole("status").textContent).toBe(
    "flow.warmupStates.ready",
  );
  expect(screen.getByText(/flow.warmupFile/).textContent).toContain(
    "flow.warmupFile",
  );
  result.rerender(
    view({
      ...status,
      next_episode_warmup: { ...status.next_episode_warmup!, state: "evicted" },
    }),
  );
  expect(screen.getByRole("status").textContent).toBe(
    "flow.warmupStates.evicted",
  );
  result.unmount();
  client.clear();
});
