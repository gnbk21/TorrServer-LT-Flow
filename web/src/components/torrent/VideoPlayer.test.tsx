import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, it, expect, vi } from "vitest";
import { render, waitFor, cleanup } from "@testing-library/react";
import VideoPlayer from "./VideoPlayer";
import { integrationsApi } from "../../api/integrations";
import { api } from "../../api/client";
const engine = vi.hoisted(() => ({
  loadSource: vi.fn(),
  attachMedia: vi.fn(),
  on: vi.fn(),
  destroy: vi.fn(),
}));
vi.mock("hls.js", () => ({
  default: class {
    static isSupported() {
      return true;
    }
    static Events = {
      ERROR: "error",
      SUBTITLE_TRACKS_UPDATED: "tracks",
      SUBTITLE_TRACK_SWITCH: "switch",
    };
    static ErrorDetails = { MANIFEST_LOAD_ERROR: "manifest" };
    static ErrorTypes = { NETWORK_ERROR: "network", MEDIA_ERROR: "media" };
    loadSource = engine.loadSource;
    attachMedia = engine.attachMedia;
    on = engine.on;
    destroy = engine.destroy;
  },
}));
vi.mock("../../api/integrations", async (importOriginal) => ({
  integrationsApi: {
    ...(await importOriginal<typeof import("../../api/integrations")>())
      .integrationsApi,
    getGst: vi.fn(),
    getGstProbe: vi.fn().mockResolvedValue({ Tracks: [] }),
  },
}));
vi.mock("../../api/client", () => ({
  api: { get: vi.fn().mockResolvedValue("OK") },
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
const props = {
  url: "http://localhost/stream?index=2",
  title: "Episode",
  hash: "abc",
  index: 2,
  path: "episode.mkv",
  onClose: vi.fn(),
};
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(integrationsApi.getGstProbe).mockResolvedValue({ Tracks: [] });
  vi.spyOn(HTMLMediaElement.prototype, "pause").mockImplementation(() => {});
  vi.spyOn(HTMLMediaElement.prototype, "load").mockImplementation(() => {});
  vi.spyOn(HTMLMediaElement.prototype, "canPlayType").mockReturnValue("");
});
it("waits for cold discovery and cancels it when the player closes", async () => {
  vi.mocked(integrationsApi.getGst).mockResolvedValue({ built_in: true });
  let finish!: (value: { Tracks: [] }) => void;
  let signal: AbortSignal | undefined;
  vi.mocked(integrationsApi.getGstProbe).mockImplementation(
    (_hash, _index, requestSignal) => {
      signal = requestSignal;
      return new Promise((resolve) => {
        finish = resolve;
      });
    },
  );
  const view = render(
    <QueryClientProvider client={new QueryClient()}>
      <VideoPlayer {...props} />
    </QueryClientProvider>,
  );
  await waitFor(() =>
    expect(integrationsApi.getGstProbe).toHaveBeenCalledOnce(),
  );
  expect(engine.loadSource).not.toHaveBeenCalled();
  view.unmount();
  expect(signal?.aborted).toBe(true);
  finish({ Tracks: [] });
  await Promise.resolve();
  await Promise.resolve();
  expect(engine.loadSource).not.toHaveBeenCalled();
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});
it("uses GST for MKV with the server audio preference and destroys HLS on close", async () => {
  vi.mocked(integrationsApi.getGst).mockResolvedValue({
    built_in: true,
    config: {},
  });
  const view = render(
    <QueryClientProvider client={new QueryClient()}>
      <VideoPlayer {...props} />
    </QueryClientProvider>,
  );
  await waitFor(() =>
    expect(engine.loadSource).toHaveBeenCalledWith(
      expect.stringContaining("/gst/abc/master.m3u8?index=2&audio=-1"),
    ),
  );
  view.unmount();
  expect(engine.destroy).toHaveBeenCalledOnce();
  expect(document.querySelector("video")).toBeNull();
});
it("retains direct playback when the executable has no GST", async () => {
  vi.mocked(integrationsApi.getGst).mockResolvedValue({ built_in: false });
  render(
    <QueryClientProvider client={new QueryClient()}>
      <VideoPlayer {...props} />
    </QueryClientProvider>,
  );
  await waitFor(() =>
    expect(document.querySelector("video")).toHaveAttribute("src", props.url),
  );
  expect(engine.loadSource).not.toHaveBeenCalled();
});
it("closing before GST discovery finishes cannot start a late player or heartbeat", async () => {
  let resolve!: (value: { built_in: boolean }) => void;
  vi.mocked(integrationsApi.getGst).mockImplementation(
    () =>
      new Promise((r) => {
        resolve = r;
      }),
  );
  const view = render(
    <QueryClientProvider client={new QueryClient()}>
      <VideoPlayer {...props} />
    </QueryClientProvider>,
  );
  view.unmount();
  resolve({ built_in: true });
  await Promise.resolve();
  await Promise.resolve();
  expect(engine.loadSource).not.toHaveBeenCalled();
  expect(api.get).not.toHaveBeenCalled();
});
