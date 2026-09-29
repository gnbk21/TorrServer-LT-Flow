import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { Torrent } from "../../types/torrent";
import { humanizeBytes, humanizeSpeed } from "../../lib/format";
import { Button } from "../common/Button";
export interface TorrentCardProps {
  torrent: Torrent;
  busy?: boolean;
  isActiveStream?: boolean;
  onOpenFiles: () => void;
  onOpenDiagnostics?: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onDropCache: () => void;
  onCopyMagnet: () => void;
  onCopyPlaylist: () => void;
}
export function TorrentCard({
  torrent,
  busy,
  isActiveStream,
  onOpenFiles,
  onOpenDiagnostics,
  onEdit,
  onDelete,
  onDropCache,
  onCopyMagnet,
  onCopyPlaylist,
}: TorrentCardProps) {
  const { t } = useTranslation();
  const [broken, setBroken] = useState<string>();
  const [confirm, setConfirm] = useState(false);
  return (
    <article
      className={`panel space-y-4 ${isActiveStream ? "border-blue-400" : ""}`}
    >
      <div className="flex gap-4">
        <div className="w-20 aspect-[2/3] shrink-0 rounded-lg bg-slate-800 overflow-hidden flex items-center justify-center">
          {torrent.poster && torrent.poster !== broken ? (
            <img
              src={torrent.poster}
              onError={() => setBroken(torrent.poster)}
              alt=""
              loading="lazy"
              decoding="async"
              className="w-full h-full object-cover"
            />
          ) : (
            <span aria-hidden="true" className="text-2xl">
              ▶
            </span>
          )}
        </div>
        <div className="min-w-0 flex-1">
          <span className="text-sm text-blue-300">
            {isActiveStream
              ? t("torrent.playing")
              : t(`torrent.state.${torrent.stat}`, {
                  defaultValue: t("status.unknown"),
                })}
          </span>
          <h2 className="text-lg font-semibold break-words">
            <button className="text-left" onClick={onOpenFiles}>
              {torrent.title || torrent.name || torrent.hash}
            </button>
          </h2>
          <p className="text-sm text-slate-400">
            {torrent.category} · {humanizeBytes(torrent.torrent_size)}
          </p>
          {torrent.file_stats && (
            <p className="text-sm text-slate-400">
              {t("torrent.fileCount", { count: torrent.file_stats.length })}
            </p>
          )}
          {[1, 2, 3].includes(torrent.stat) && (
            <p className="text-sm">
              {humanizeSpeed(torrent.download_speed ?? 0)} ·{" "}
              {torrent.active_peers ?? 0} {t("flow.peers")}
            </p>
          )}
          {torrent.stat === 2 && (
            <p>
              {humanizeBytes(torrent.preloaded_bytes ?? 0)} /{" "}
              {humanizeBytes(torrent.preload_size)}
            </p>
          )}
          {isActiveStream &&
            (torrent.flow_playback?.length ? (
              torrent.flow_playback.map((session, index) => (
                <p
                  key={`${session.file_index}:${index}`}
                  className={
                    session.buffer_warning
                      ? "text-amber-300 text-sm"
                      : "text-sm"
                  }
                >
                  {t("flow.fileIndex", { index: session.file_index })} ·{" "}
                  {t("flow.buffer")}:{" "}
                  {session.buffer_seconds == null
                    ? "—"
                    : session.buffer_seconds.toFixed(1) + " s"}{" "}
                  · {t("flow.sustainability")}:{" "}
                  {session.sustainability == null
                    ? "—"
                    : session.sustainability.toFixed(2) + "×"}
                </p>
              ))
            ) : (
              <p className="text-sm">
                {t("flow.buffer")}: — · {t("flow.sustainability")}: —
              </p>
            ))}
        </div>
      </div>
      <div className="actions">
        <Button disabled={busy} onClick={onOpenFiles}>
          {t("torrent.files")}
        </Button>
        <Button disabled={busy} onClick={onCopyPlaylist}>
          {t("torrent.copyPlaylist")}
        </Button>
        <Button disabled={busy} onClick={onEdit}>
          {t("torrent.edit")}
        </Button>
        {onOpenDiagnostics && (
          <Button disabled={busy} onClick={onOpenDiagnostics}>
            {t("flow.diagnostics")}
          </Button>
        )}
      </div>
      <details>
        <summary>{t("Actions")}</summary>
        <div className="actions">
          <Button disabled={busy} onClick={onCopyMagnet}>
            {t("torrent.copyMagnet")}
          </Button>
          <Button disabled={busy} onClick={onDropCache}>
            {t("torrent.dropCache")}
          </Button>
          <Button
            disabled={busy}
            variant="danger"
            onClick={() => setConfirm(true)}
          >
            {t("torrent.remove")}
          </Button>
        </div>
      </details>
      {confirm && (
        <div className="border-t border-rose-800 pt-3">
          <p>{t("torrent.confirmRemove")}</p>
          <div className="actions">
            <Button disabled={busy} onClick={() => setConfirm(false)}>
              {t("Cancel")}
            </Button>
            <Button disabled={busy} variant="danger" onClick={onDelete}>
              {t("torrent.remove")}
            </Button>
          </div>
        </div>
      )}
    </article>
  );
}
