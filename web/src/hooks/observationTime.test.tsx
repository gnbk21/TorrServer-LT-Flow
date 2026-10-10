import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { useObservationTime } from "./observationTime";

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  Object.defineProperty(document, "hidden", {
    configurable: true,
    value: false,
  });
});

it("does not schedule overflowing or far-future server timestamps", () => {
  vi.useFakeTimers();
  vi.setSystemTime(100_000);
  renderHook(() => useObservationTime(true, NaN, Number.MAX_SAFE_INTEGER));
  expect(vi.getTimerCount()).toBe(0);
});

it("owns one deadline timer, releases it while hidden or closed, and expires without polling", () => {
  vi.useFakeTimers();
  vi.setSystemTime(100_000);
  const { result, rerender } = renderHook(
    ({ enabled }) => useObservationTime(enabled, 110_000, 105_000),
    { initialProps: { enabled: true } },
  );
  expect(vi.getTimerCount()).toBe(1);
  act(() => {
    Object.defineProperty(document, "hidden", {
      configurable: true,
      value: true,
    });
    document.dispatchEvent(new Event("visibilitychange"));
  });
  expect(vi.getTimerCount()).toBe(0);
  expect(result.current).toBe(Infinity);
  act(() => {
    vi.advanceTimersByTime(6000);
    Object.defineProperty(document, "hidden", {
      configurable: true,
      value: false,
    });
    document.dispatchEvent(new Event("visibilitychange"));
  });
  expect(result.current).toBe(106_000);
  expect(vi.getTimerCount()).toBe(1);
  rerender({ enabled: false });
  expect(vi.getTimerCount()).toBe(0);
  rerender({ enabled: true });
  act(() => vi.advanceTimersByTime(4001));
  expect(result.current).toBe(110_001);
  expect(vi.getTimerCount()).toBe(0);
});
