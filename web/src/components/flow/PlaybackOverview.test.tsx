import { afterEach, expect, test, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { PlaybackOverview } from "./PlaybackOverview";
import type { FlowStatusResponse } from "../../types/flow";
const mocks = vi.hoisted(() => ({ retry: vi.fn() }));
vi.mock("../../api/flow", () => ({ flowApi: { retryMetadata: mocks.retry } }));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});
const now = Date.parse("2026-10-22T10:00:00Z");
const status = {
  hash: "fixture",
  sampled_at: new Date(now).toISOString(),
  startup: {
    state: "METADATA",
    file_index: 0,
    metadata_recovery: { state: "METADATA_MISS", attempts: 2 },
  },
} as FlowStatusResponse;
test("metadata retry is explicit and leaving the drawer aborts its observer", () => {
  mocks.retry.mockReturnValue(new Promise(() => {}));
  const view = render(<PlaybackOverview status={status} now={now} />);
  expect(mocks.retry).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "flow.upgrade.retry" }));
  expect(mocks.retry).toHaveBeenCalledTimes(1);
  const signal = mocks.retry.mock.calls[0]![1] as AbortSignal;
  view.unmount();
  expect(signal.aborted).toBe(true);
});
test("stale evidence does not offer actions as if it were current", () => {
  render(<PlaybackOverview status={status} now={now + 11000} />);
  expect(
    screen.queryByRole("button", { name: "flow.upgrade.retry" }),
  ).toBeNull();
  expect(screen.getByRole("status").textContent).toBe(
    "flow.upgrade.causes.stale",
  );
});
