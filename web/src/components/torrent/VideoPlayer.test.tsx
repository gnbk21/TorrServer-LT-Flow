import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, it, expect, vi } from "vitest";
import {
  render,
  waitFor,
  cleanup,
  fireEvent,
  act,
} from "@testing-library/react";
import { screen } from "@testing-library/react";
import VideoPlayer from "./VideoPlayer";
import { integrationsApi } from "../../api/integrations";
import { api } from "../../api/client";
import { viewedApi } from "../../api/viewed";
const preferences = vi.hoisted(() => ({ tracking: false }));
vi.mock("../../hooks/queries", () => ({
  useSettings: () => ({ data: { TrackTimecode: preferences.tracking } }),
}));
vi.mock("../../api/viewed", () => ({
  viewedApi: { report: vi.fn().mockResolvedValue(undefined) },
}));
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
  preferences.tracking = false;
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
  fireEvent.click(
    screen.getByRole("button", { name: "torrent.compatibility.gstMode" }),
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
  vi.unstubAllGlobals();
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
  fireEvent.click(
    screen.getByRole("button", { name: "torrent.compatibility.gstMode" }),
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
  expect(integrationsApi.getGst).not.toHaveBeenCalled();
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
  fireEvent.click(
    screen.getByRole("button", { name: "torrent.compatibility.gstMode" }),
  );
  view.unmount();
  resolve({ built_in: true });
  await Promise.resolve();
  await Promise.resolve();
  expect(engine.loadSource).not.toHaveBeenCalled();
  expect(api.get).not.toHaveBeenCalled();
});

it("uses an owned VTT blob for subtitles and revokes it when closed", async () => {
  vi.mocked(integrationsApi.getGst).mockResolvedValue({ built_in: false });
  const create = vi.fn<(blob: Blob) => string>(
    () => "blob:http://localhost/owned-subtitle",
  );
  const revoke = vi.fn();
  vi.stubGlobal(
    "URL",
    class extends URL {
      static override createObjectURL = create;
      static override revokeObjectURL = revoke;
    },
  );
  const view = render(
    <QueryClientProvider client={new QueryClient()}>
      <VideoPlayer {...props} />
    </QueryClientProvider>,
  );
  const file = new File(["<script>untrusted</script>"], "owned.vtt", {
    type: "text/html",
  });
  fireEvent.change(document.querySelector('input[type="file"]')!, {
    target: { files: [file] },
  });
  await waitFor(() => expect(create).toHaveBeenCalledOnce());
  expect(create.mock.calls[0]![0]).toBeInstanceOf(Blob);
  expect(create.mock.calls[0]![0].type).toBe("text/vtt");
  expect(document.querySelector("track")).toHaveAttribute(
    "src",
    "blob:http://localhost/owned-subtitle",
  );
  view.unmount();
  expect(revoke).toHaveBeenCalledWith("blob:http://localhost/owned-subtitle");
});

it("reports actual position every five seconds and on pause/close only with tracking enabled", async () => {
  preferences.tracking = true;
  vi.useFakeTimers();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const view = render(
    <QueryClientProvider client={client}>
      <VideoPlayer {...props} />
    </QueryClientProvider>,
  );
  const video = document.querySelector("video")!;
  Object.defineProperty(video, "readyState", { value: 1, configurable: true });
  Object.defineProperty(video, "paused", { value: false, configurable: true });
  video.currentTime = 42;
  await act(async () => {
    await vi.advanceTimersByTimeAsync(5000);
  });
  expect(viewedApi.report).toHaveBeenCalledWith(props.hash, 2, 42, false);
  video.currentTime = 46;
  await act(async () => {
    fireEvent.pause(video);
    await Promise.resolve();
  });
  expect(viewedApi.report).toHaveBeenCalledWith(props.hash, 2, 46, false);
  video.currentTime = 47;
  view.unmount();
  expect(viewedApi.report).toHaveBeenCalledWith(props.hash, 2, 47, true);
});
it("does not silently enable position reporting", async () => {
  const view = render(
    <QueryClientProvider client={new QueryClient()}>
      <VideoPlayer {...props} />
    </QueryClientProvider>,
  );
  const video = document.querySelector("video")!;
  Object.defineProperty(video, "readyState", { value: 1, configurable: true });
  video.currentTime = 60;
  fireEvent.pause(video);
  view.unmount();
  expect(viewedApi.report).not.toHaveBeenCalled();
});

it("retains the latest pause/close position behind a slow report without replaying it", async () => {
  preferences.tracking = true;
  let finish: () => void = () => {};
  vi.mocked(viewedApi.report).mockImplementationOnce(
    () =>
      new Promise<void>((resolve) => {
        finish = resolve;
      }),
  );
  const view = render(
    <QueryClientProvider client={new QueryClient()}>
      <VideoPlayer {...props} />
    </QueryClientProvider>,
  );
  const video = document.querySelector("video")!;
  Object.defineProperty(video, "readyState", { value: 1, configurable: true });
  video.currentTime = 42;
  fireEvent.pause(video);
  video.currentTime = 46;
  fireEvent.pause(video);
  video.currentTime = 47;
  view.unmount();
  expect(viewedApi.report).toHaveBeenCalledTimes(1);
  await act(async () => {
    finish();
  });
  expect(viewedApi.report).toHaveBeenCalledTimes(2);
  expect(viewedApi.report).toHaveBeenLastCalledWith(props.hash, 2, 47, true);
});
