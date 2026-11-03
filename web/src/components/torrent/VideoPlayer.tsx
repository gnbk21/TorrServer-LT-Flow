import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type Hls from "hls.js";
import { integrationsApi, type GstSettings } from "../../api/integrations";
import { Modal } from "../common/Modal";
import { Button } from "../common/Button";
import { subtitlesApi } from "../../api/subtitles";
import { viewedApi } from "../../api/viewed";
import { useSettings } from "../../hooks/queries";

export default function VideoPlayer({
  url,
  title,
  hash,
  index,
  path,
  onClose,
  startSeconds = 0,
  subtitleIndex,
}: {
  url: string;
  title: string;
  hash: string;
  index: number;
  path: string;
  onClose: () => void;
  startSeconds?: number;
  subtitleIndex?: number;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const settings = useSettings();
  const [video, setVideo] = useState<HTMLVideoElement | null>(null);
  const hlsRef = useRef<Hls | null>(null);
  const [error, setError] = useState(false);
  const [subtitle, setSubtitle] = useState<string>();
  const [tracks, setTracks] = useState<{ id: number; name: string }[]>([]);
  const [selected, setSelected] = useState(-1);
  const [gstEnabled, setGstEnabled] = useState(false);
  const [gstConfig, setGstConfig] = useState<GstSettings["config"]>();
  const [audio, setAudio] = useState(-1);
  const resumeTime = useRef(startSeconds);
  const [mode, setMode] = useState<"direct" | "gst">("direct");
  const probe = useQuery({
    queryKey: ["gst-probe", hash, index],
    queryFn: ({ signal }) => integrationsApi.getGstProbe(hash, index, signal),
    enabled: gstEnabled,
    retry: 3,
    retryDelay: 1500,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  });
  const audioTracks =
    probe.data?.Tracks?.filter((track) => track.Type === "audio") || [];
  const [subtitleError, setSubtitleError] = useState<unknown>();
  const [subtitleAttempt, setSubtitleAttempt] = useState(0);
  const [positionError, setPositionError] = useState(false);
  useEffect(() => {
    if (!video || !settings.data?.TrackTimecode) return;
    let pending = false,
      stopped = false;
    let queued: { position: number; closing: boolean } | undefined;
    const send = (sample: { position: number; closing: boolean }) => {
      if (
        queryClient.getQueryData<{ TrackTimecode?: boolean }>(["settings"])
          ?.TrackTimecode === false
      )
        return;
      pending = true;
      void viewedApi
        .report(hash, index, sample.position, sample.closing)
        .then(() => {
          if (!stopped) setPositionError(false);
        })
        .catch(() => {
          if (!stopped) setPositionError(true);
        })
        .finally(() => {
          pending = false;
          const next = queued;
          queued = undefined;
          if (next && (!stopped || next.closing)) send(next);
        });
    };
    const report = (closing = false) => {
      if (
        queryClient.getQueryData<{ TrackTimecode?: boolean }>(["settings"])
          ?.TrackTimecode === false ||
        video.readyState < 1 ||
        !Number.isFinite(video.currentTime) ||
        video.currentTime < 0
      )
        return;
      const sample = {
        position: video.currentTime,
        closing: closing || queued?.closing === true,
      };
      // Coalesce newly observed positions behind the one in flight. This is
      // not a retry: each explicit pause/close keeps its latest actual position,
      // and an old response cannot overwrite the final report out of order.
      if (pending) queued = sample;
      else send(sample);
    };
    const timer = setInterval(() => {
      if (!video.paused) report();
    }, 5000);
    const pause = () => report();
    const close = () => report(true);
    video.addEventListener("pause", pause);
    window.addEventListener("pagehide", close);
    return () => {
      clearInterval(timer);
      video.removeEventListener("pause", pause);
      window.removeEventListener("pagehide", close);
      report(true);
      stopped = true;
    };
  }, [video, hash, index, settings.data?.TrackTimecode, queryClient]);
  useEffect(() => {
    setSubtitle(undefined);
    if (!subtitleIndex) return;
    const controller = new AbortController();
    setSubtitleError(undefined);
    void subtitlesApi
      .text(hash, index, subtitleIndex, controller.signal)
      .then((text) => {
        if (!controller.signal.aborted)
          setSubtitle(
            URL.createObjectURL(new Blob([text], { type: "text/vtt" })),
          );
      })
      .catch((error) => {
        if (!controller.signal.aborted) setSubtitleError(error);
      });
    return () => controller.abort();
  }, [hash, index, subtitleIndex, subtitleAttempt]);
  useEffect(() => {
    const controller = new AbortController();
    let stopped = false;
    let retry: ReturnType<typeof setTimeout> | undefined;
    let heartbeat: ReturnType<typeof setInterval> | undefined;
    if (!video) return;
    setError(false);
    setGstEnabled(false);
    setTracks([]);
    setSelected(-1);
    const start = async () => {
      let source = url;
      try {
        if (mode === "gst") {
          const gst = await integrationsApi.getGst(controller.signal);
          if (stopped) return;
          if (gst.built_in) {
            setGstConfig(gst.config);
            await queryClient.fetchQuery({
              queryKey: ["gst-probe", hash, index],
              queryFn: ({ signal }) =>
                integrationsApi.getGstProbe(hash, index, signal),
              staleTime: Infinity,
              retry: false,
            });
            if (stopped) return;
            setGstEnabled(true);
            source = integrationsApi.getGstMasterUrl(hash, index, audio);
            let heartbeatPending = false;
            heartbeat = setInterval(() => {
              if (heartbeatPending) return;
              heartbeatPending = true;
              void integrationsApi
                .getGstHeartbeat(hash, controller.signal)
                .catch(() => {})
                .finally(() => {
                  heartbeatPending = false;
                });
            }, 30000);
          } else {
            setError(true);
          }
        }
      } catch {
        if (stopped)
          return; /* Direct playback remains available without GST. */
      }
      if (stopped) return;
      if (
        /\.m3u8(?:\?|$)/.test(source) &&
        !video.canPlayType("application/vnd.apple.mpegurl")
      ) {
        const { default: Hls } = await import("hls.js");
        if (stopped) return;
        if (!Hls.isSupported()) {
          setError(true);
          return;
        }
        const hls = new Hls({
          fragLoadPolicy: {
            default: {
              maxTimeToFirstByteMs: 30000,
              maxLoadTimeMs: 120000,
              timeoutRetry: {
                maxNumRetry: 4,
                retryDelayMs: 1000,
                maxRetryDelayMs: 8000,
              },
              errorRetry: {
                maxNumRetry: 6,
                retryDelayMs: 1000,
                maxRetryDelayMs: 8000,
              },
            },
          },
        });
        hlsRef.current = hls;
        let manifestRetries = 0;
        let recoveryAttempts = 0;
        hls.on(Hls.Events.SUBTITLE_TRACKS_UPDATED, (_event, data) =>
          setTracks(
            data.subtitleTracks.map((track) => ({
              id: track.id,
              name: track.name || track.lang || String(track.id + 1),
            })),
          ),
        );
        hls.on(Hls.Events.SUBTITLE_TRACK_SWITCH, (_event, data) =>
          setSelected(data.id),
        );
        hls.on(Hls.Events.ERROR, (_event, data) => {
          if (!data.fatal || stopped) return;
          if (
            data.details === Hls.ErrorDetails.MANIFEST_LOAD_ERROR &&
            data.response?.code === 502 &&
            manifestRetries++ < 15
          ) {
            clearTimeout(retry);
            retry = setTimeout(() => {
              if (!stopped) hls.loadSource(source);
            }, 2000);
            return;
          }
          if (recoveryAttempts++ < 3) {
            if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
              hls.startLoad();
              return;
            }
            if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
              hls.recoverMediaError();
              return;
            }
          }
          setError(true);
        });
        hls.loadSource(source);
        hls.attachMedia(video);
      } else video.src = source;
    };
    void start().catch(() => {
      if (!stopped) setError(true);
    });
    return () => {
      stopped = true;
      controller.abort();
      clearTimeout(retry);
      clearInterval(heartbeat);
      hlsRef.current?.destroy();
      hlsRef.current = null;
      video.pause();
      video.removeAttribute("src");
      video.load();
    };
  }, [url, hash, index, path, video, audio, queryClient, mode]);
  useEffect(
    () => () => {
      if (subtitle) URL.revokeObjectURL(subtitle);
    },
    [subtitle],
  );
  return (
    <Modal isOpen onClose={onClose} title={title} maxWidth="4xl">
      <p className="mb-3 text-sm">{t("torrent.compatibility.direct")}</p>
      <div className="actions mb-3">
        <Button
          variant={mode === "direct" ? "primary" : "outline"}
          onClick={() => {
            resumeTime.current = video?.currentTime || 0;
            setMode("direct");
          }}
        >
          {t("torrent.compatibility.directMode")}
        </Button>
        <Button
          variant={mode === "gst" ? "primary" : "outline"}
          onClick={() => {
            resumeTime.current = video?.currentTime || 0;
            setMode("gst");
          }}
        >
          {t("torrent.compatibility.gstMode")}
        </Button>
      </div>
      {mode === "gst" && (
        <p className="mb-3 text-sm">
          {t("torrent.compatibility.conversionWarning")}
        </p>
      )}
      {mode === "gst" && gstConfig && (
        <details className="mb-3">
          <summary>{t("GStreamer.Settings")}</summary>
          <dl>
            {[
              "AACBitrateKbps",
              "AACChannels",
              "TranscodeH264",
              "TranscodeH265",
              "TranscodeAV1",
              "TranscodeVP9",
              "TranscodeVP8",
              "TranscodeAVI",
              "HDRToSDR",
              "HardwareAcceleration",
              "VideoBitrate",
            ]
              .filter((key) => gstConfig[key] !== undefined)
              .map((key) => (
                <div key={key}>
                  <dt>{t(`GStreamer.${key}`, { defaultValue: key })}</dt>
                  <dd>
                    {typeof gstConfig[key] === "boolean"
                      ? t(gstConfig[key] ? "status.enabled" : "status.disabled")
                      : String(gstConfig[key])}
                  </dd>
                </div>
              ))}
          </dl>
        </details>
      )}
      <video
        ref={setVideo}
        controls
        playsInline
        className="w-full"
        onError={() => setError(true)}
        onLoadedMetadata={() => {
          if (video && resumeTime.current > 0) {
            video.currentTime = resumeTime.current;
            resumeTime.current = 0;
          }
        }}
      >
        {subtitle && (
          <track
            kind="subtitles"
            src={subtitle}
            label={t("torrent.subtitles")}
            default
          />
        )}
      </video>
      {error && <p role="alert">{t("torrent.codecFallback")}</p>}
      {!!subtitleError && (
        <div role="alert">
          <p>{t("torrent.sidecars.deferred")}</p>
          <Button onClick={() => setSubtitleAttempt((value) => value + 1)}>
            {t("WAF.Retry")}
          </Button>
        </div>
      )}
      {positionError && <p role="status">{t("recent.positionFailed")}</p>}
      {audioTracks.length > 1 && (
        <label className="field">
          {t("GStreamer.AudioTrack")}
          <select
            value={audio}
            onChange={(event) => {
              resumeTime.current = video?.currentTime || 0;
              setAudio(Number(event.target.value));
            }}
          >
            <option value={-1}>{t("GStreamer.AudioLangDefault")}</option>
            {audioTracks.map((track) => (
              <option key={track.Index} value={track.Index}>
                {[track.Title, track.Language, track.Codec]
                  .filter(Boolean)
                  .join(" · ") || track.Index + 1}
              </option>
            ))}
          </select>
        </label>
      )}
      {tracks.length > 0 && (
        <label className="field">
          {t("torrent.subtitles")}
          <select
            value={selected}
            onChange={(e) => {
              const value = Number(e.target.value);
              if (hlsRef.current) hlsRef.current.subtitleTrack = value;
              setSelected(value);
            }}
          >
            <option value={-1}>{t("status.disabled")}</option>
            {tracks.map((track) => (
              <option key={track.id} value={track.id}>
                {track.name}
              </option>
            ))}
          </select>
        </label>
      )}
      <label className="field mt-3">
        {t("torrent.subtitles")}
        <input
          type="file"
          accept=".vtt,text/vtt"
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file)
              setSubtitle(
                URL.createObjectURL(new Blob([file], { type: "text/vtt" })),
              );
          }}
        />
      </label>
      <a
        className="inline-flex min-h-11 items-center underline"
        href={url}
        target="_blank"
        rel="noreferrer"
      >
        {t("torrent.openStream")}
      </a>
    </Modal>
  );
}
