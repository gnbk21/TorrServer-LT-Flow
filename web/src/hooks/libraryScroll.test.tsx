import { StrictMode, type ReactNode } from "react";
import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { useLibraryScroll } from "./libraryScroll";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  sessionStorage.clear();
});

function frames() {
  const pending = new Map<number, FrameRequestCallback>();
  let id = 0;
  vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
    pending.set(++id, callback);
    return id;
  });
  vi.spyOn(window, "cancelAnimationFrame").mockImplementation((key) => {
    pending.delete(key);
  });
  vi.spyOn(document.documentElement, "scrollHeight", "get").mockReturnValue(
    5000,
  );
  const scroll = vi.spyOn(window, "scrollTo").mockImplementation(() => {});
  const flush = () =>
    act(() => {
      const callbacks = [...pending.values()];
      pending.clear();
      callbacks.forEach((callback) => callback(0));
    });
  return { scroll, flush };
}

it("restores cached rows after StrictMode cancels the initial frame", () => {
  const f = frames();
  sessionStorage.setItem("flow.library.scroll", "900");
  renderHook(() => useLibraryScroll(true), {
    wrapper: ({ children }: { children: ReactNode }) => (
      <StrictMode>{children}</StrictMode>
    ),
  });
  f.flush();
  expect(f.scroll).toHaveBeenCalledExactlyOnceWith(0, 900);
});

it("preserves the offset during loading, restores once and saves later scrolls", () => {
  const f = frames();
  sessionStorage.setItem("flow.library.scroll", "900");
  const hook = renderHook(({ ready }) => useLibraryScroll(ready), {
    initialProps: { ready: false },
  });
  act(() => window.dispatchEvent(new Event("scroll")));
  expect(sessionStorage.getItem("flow.library.scroll")).toBe("900");
  hook.rerender({ ready: true });
  f.flush();
  expect(f.scroll).toHaveBeenCalledExactlyOnceWith(0, 900);
  vi.spyOn(window, "scrollY", "get").mockReturnValue(1100);
  act(() => window.dispatchEvent(new Event("scroll")));
  expect(sessionStorage.getItem("flow.library.scroll")).toBe("1100");
  hook.rerender({ ready: false });
  hook.rerender({ ready: true });
  f.flush();
  expect(f.scroll).toHaveBeenCalledOnce();
  hook.unmount();
  act(() => window.dispatchEvent(new Event("scroll")));
  expect(sessionStorage.getItem("flow.library.scroll")).toBe("1100");
});
