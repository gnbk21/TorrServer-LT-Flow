import { afterEach, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PosterSearch } from "./PosterSearch";
import { integrationsApi } from "../../api/integrations";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { resolvedLanguage: "en" },
  }),
}));
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

it("aborts artwork configuration work when the dialog closes", async () => {
  let incoming: AbortSignal | undefined;
  vi.spyOn(integrationsApi, "getTMDB").mockImplementation((signal) => {
    incoming = signal;
    return new Promise((_, reject) =>
      signal?.addEventListener(
        "abort",
        () => reject(new DOMException("Cancelled", "AbortError")),
        { once: true },
      ),
    );
  });
  const view = render(<PosterSearch title="Fixture" onSelect={() => {}} />);
  await userEvent.click(
    screen.getByRole("button", { name: "torrent.findPoster" }),
  );
  expect(incoming?.aborted).toBe(false);
  view.unmount();
  expect(incoming?.aborted).toBe(true);
});

it("aborts external artwork fetching when the title changes", async () => {
  vi.spyOn(integrationsApi, "getTMDB").mockResolvedValue({
    APIKey: "fixture",
    APIURL: "https://example.test",
  } as never);
  let incoming: AbortSignal | undefined;
  vi.spyOn(globalThis, "fetch").mockImplementation((_, options) => {
    incoming = options?.signal as AbortSignal;
    return new Promise((_, reject) =>
      incoming?.addEventListener(
        "abort",
        () => reject(new DOMException("Cancelled", "AbortError")),
        { once: true },
      ),
    );
  });
  const view = render(
    <PosterSearch title="First fixture" onSelect={() => {}} />,
  );
  await userEvent.click(
    screen.getByRole("button", { name: "torrent.findPoster" }),
  );
  await waitFor(() => expect(incoming).toBeDefined());
  view.rerender(<PosterSearch title="Second fixture" onSelect={() => {}} />);
  expect(incoming?.aborted).toBe(true);
  await waitFor(() =>
    expect(
      screen.getByRole("button", { name: "torrent.findPoster" }),
    ).not.toBeDisabled(),
  );
  expect(screen.queryByText("torrent.tmdbFailed")).not.toBeInTheDocument();
});
