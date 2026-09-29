import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import type Hls from "hls.js";
import { integrationsApi } from "../../api/integrations";
import { Modal } from "../common/Modal";

export default function VideoPlayer({
  url,
  title,
  hash,
  index,
  path,
  onClose,
}: {
  url: string;
  title: string;
  hash: string;
  index: number;
  path: string;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const [video, setVideo] = useState<HTMLVideoElement | null>(null);
  const hlsRef = useRef<Hls | null>(null);
  const [error, setError] = useState(false);
  const [subtitle, setSubtitle] = useState<string>();
  const [tracks, setTracks] = useState<{ id: number; name: string }[]>([]);
  const [selected, setSelected] = useState(-1);
  const [gstEnabled, setGstEnabled] = useState(false);
  const [audio, setAudio] = useState(-1);
  const resumeTime = useRef(0);
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
  useEffect(() => {
    const controller = new AbortController();
    let stopped = false;
    let retry: ReturnType<typeof setTimeout> | undefined;
    let heartbeat: ReturnType<typeof setInterval> | undefined;
    if (!video) return;
    setError(false);
    setTracks([]);
    setSelected(-1);
    const start = async () => {
      let source = url;
      try {
        const gst = await integrationsApi.getGst(controller.signal);
        if (stopped) return;
        if (
          gst.built_in &&
          (/\.(mkv|mk3d|webm)$/i.test(path) ||
            (/\.avi$/i.test(path) && gst.config?.TranscodeAVI === true))
        ) {
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
  }, [url, hash, index, path, video, audio]);
  useEffect(
    () => () => {
      if (subtitle) URL.revokeObjectURL(subtitle);
    },
    [subtitle],
  );
  return (
    <Modal isOpen onClose={onClose} title={title} maxWidth="4xl">
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
            if (file) setSubtitle(URL.createObjectURL(file));
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
