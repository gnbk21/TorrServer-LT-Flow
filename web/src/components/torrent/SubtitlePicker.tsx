import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { subtitlesApi, subtitlePath } from "../../api/subtitles";
import { flowApi } from "../../api/flow";
import { ApiError } from "../../api/client";
import { useVisibilityGate } from "../../hooks/queries";
import { useExternalMediaBase } from "../../hooks/mediaBase";
import { externalMediaURL } from "../../lib/mediaBase";
import { copyText } from "../../lib/clipboard";
import { RequestError } from "../common/RequestState";
import { Button } from "../common/Button";
export function SubtitlePicker({
  hash,
  index,
  value,
  onChange,
}: {
  hash: string;
  index: number;
  value?: number;
  onChange: (value?: number) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [error, setError] = useState<unknown>();
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState(false);
  const enabled = useVisibilityGate(["subtitles", hash, index], open);
  const mediaBase = useExternalMediaBase();
  const query = useQuery({
    queryKey: ["subtitles", hash, index],
    queryFn: ({ signal }) => subtitlesApi.list(hash, index, signal),
    enabled,
    retry: false,
    staleTime: 30000,
  });
  const selected = query.data?.tracks.find(
    (track) => track.file_index === value,
  );
  const copy = async () => {
    if (!value) return;
    setBusy(true);
    setError(undefined);
    try {
      const link = await flowApi.playbackLink(hash, index, undefined, value);
      if (!link.subtitle_path) throw new Error("Subtitle link is unavailable");
      await copyText(
        externalMediaURL(
          new URL(link.subtitle_path, window.location.origin).href,
          mediaBase.data,
        ),
      );
      setCopied(true);
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <details onToggle={(e) => setOpen(e.currentTarget.open)}>
      <summary>{t("torrent.sidecars.title")}</summary>
      <p className="text-sm mt-2">{t("torrent.sidecars.hint")}</p>
      {query.error instanceof ApiError && query.error.status === 404 ? (
        <p>{t("status.unknown")}</p>
      ) : (
        query.error && (
          <RequestError error={query.error} retry={() => query.refetch()} />
        )
      )}
      <label className="field mt-2">
        {t("torrent.sidecars.select")}
        <select
          value={value ?? ""}
          onChange={(e) => {
            onChange(e.target.value ? Number(e.target.value) : undefined);
            setCopied(false);
          }}
        >
          <option value="">{t("status.disabled")}</option>
          {query.data?.tracks.map((track) => (
            <option
              key={track.file_index}
              value={track.file_index}
              disabled={track.size > 4 * 1024 * 1024}
            >
              {track.path} · {track.language || "—"} ·{" "}
              {t(`torrent.sidecars.${track.match}`)}
              {track.forced ? ` · ${t("torrent.sidecars.forced")}` : ""}
              {track.hearing_impaired ? " · SDH" : ""}
            </option>
          ))}
        </select>
      </label>
      {query.data?.known && !query.data.tracks.length && (
        <p>{t("torrent.sidecars.none")}</p>
      )}
      {query.data?.truncated && <p>{t("torrent.sidecars.truncated")}</p>}
      {selected && (
        <div className="actions mt-2">
          <a
            className="inline-flex min-h-11 items-center underline"
            href={subtitlePath(hash, index, selected.file_index, "original")}
            download={`subtitle.${selected.format}`}
          >
            {t("torrent.sidecars.download")}
          </a>
          <Button isLoading={busy} onClick={() => void copy()}>
            {t("torrent.sidecars.copy")}
          </Button>
        </div>
      )}
      {copied && <p role="status">{t("torrent.copied")}</p>}
      {!!error && <RequestError error={error} />}
      <p className="text-sm text-slate-400 mt-2">
        {t("torrent.sidecars.fallback")}
      </p>
    </details>
  );
}
