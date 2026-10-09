import { afterEach, expect, test, vi } from "vitest";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import { LANTest } from "./LANTest";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { exists: () => false },
  }),
}));
vi.mock("../../lib/lan-test", () => ({
  measureLAN: (signal: AbortSignal) =>
    new Promise((_resolve, reject) => {
      signal.addEventListener("abort", () => reject(signal.reason));
    }),
}));
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

test("LAN timeout reports failure and allows another run", async () => {
  vi.useFakeTimers();
  render(<LANTest />);
  fireEvent.click(screen.getByText("settings.lanRun"));
  await act(async () => {
    await vi.advanceTimersByTimeAsync(20000);
  });
  expect(screen.getByRole("alert")).toHaveTextContent("status.timeout");
  expect(screen.getByText("settings.lanRun")).toBeEnabled();
});

test("explicit LAN cancellation stays quiet and releases busy state", async () => {
  render(<LANTest />);
  fireEvent.click(screen.getByText("settings.lanRun"));
  await act(async () => {
    fireEvent.click(screen.getByText("Cancel"));
  });
  expect(screen.queryByRole("alert")).toBeNull();
  expect(screen.getByText("settings.lanRun")).toBeEnabled();
});
