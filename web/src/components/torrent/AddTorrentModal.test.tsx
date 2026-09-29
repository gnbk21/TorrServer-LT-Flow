import { afterEach, it, expect, vi } from "vitest";
import {
  render,
  screen,
  waitFor,
  cleanup,
  fireEvent,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AddTorrentModal } from "./AddTorrentModal";
import { torrentsApi } from "../../api/torrents";
vi.mock("../../api/torrents", () => ({
  torrentsApi: { add: vi.fn(), upload: vi.fn() },
}));
vi.mock("./PosterSearch", () => ({ PosterSearch: () => null }));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { exists: () => false },
  }),
}));
afterEach(() => {
  cleanup();
  vi.resetAllMocks();
});
it("retries only failed batch entries while preserving their metadata", async () => {
  const add = vi.mocked(torrentsApi.add);
  add
    .mockResolvedValueOnce({ hash: "first" } as never)
    .mockRejectedValueOnce(new Error("network"))
    .mockResolvedValueOnce({ hash: "third" } as never)
    .mockResolvedValueOnce({ hash: "second" } as never);
  render(<AddTorrentModal isOpen onClose={() => {}} onSuccess={() => {}} />);
  const input = screen.getByLabelText("ImportLibraryHint");
  fireEvent.change(input, {
    target: {
      value: JSON.stringify([
        { hash: "first", title: "First" },
        { hash: "second", title: "Second" },
        { hash: "third", title: "Third" },
      ]),
    },
  });
  await userEvent.click(screen.getByRole("button", { name: "Add" }));
  await waitFor(() => expect(add).toHaveBeenCalledTimes(3));
  await waitFor(() =>
    expect((input as HTMLTextAreaElement).value).toContain("second"),
  );
  expect((input as HTMLTextAreaElement).value).not.toContain("first");
  expect((input as HTMLTextAreaElement).value).not.toContain("third");
  await userEvent.click(screen.getByRole("button", { name: "Add" }));
  await waitFor(() => expect(add).toHaveBeenCalledTimes(4));
  expect(add.mock.calls[3]?.[0]).toMatchObject({
    link: "second",
    title: "Second",
  });
});
