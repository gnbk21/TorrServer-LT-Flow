import { useEffect, useState, lazy, Suspense } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Modal } from "../common/Modal";
import { Button } from "../common/Button";
import { PlaybackLinks } from "../common/PlaybackLinks";
import { PrepareEpisode } from "./PrepareEpisode";
import { WebSeeds } from "./WebSeeds";
import { RequestError, Loading } from "../common/RequestState";
import { torrentsApi } from "../../api/torrents";
import { viewedApi } from "../../api/viewed";
import type { Torrent, TorrentFile } from "../../types/torrent";
import {
  humanizeBytes,
  parseEpisodeInfo,
  formatDuration,
} from "../../lib/format";
const CacheMap = lazy(() =>
  import("./CacheMap").then((m) => ({ default: m.CacheMap })),
);
const VideoPlayer = lazy(() => import("./VideoPlayer"));
export interface TorrentFilesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  torrent: Torrent;
}
export function TorrentFilesDialog({
  isOpen,
  onClose,
  torrent,
}: TorrentFilesDialogProps) {
  const { t } = useTranslation();
  const [cacheOpen, setCacheOpen] = useState(false);
  const [player, setPlayer] = useState<{
    url: string;
    title: string;
    hash: string;
    index: number;
    path: string;
  }>();
  const [error, setError] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  const info = useQuery({
    queryKey: ["torrent", torrent.hash],
    queryFn: ({ signal }) => torrentsApi.get(torrent.hash, signal),
    enabled: isOpen,
    refetchInterval: (q) =>
      isOpen && !document.hidden && !q.state.data?.file_stats?.length
        ? 2000
        : false,
  });
  const viewed = useQuery({
    queryKey: ["viewed", torrent.hash],
    queryFn: ({ signal }) => viewedApi.list(torrent.hash, signal),
    enabled: isOpen,
  });
  useEffect(() => {
    setError(undefined);
    setPlayer(undefined);
  }, [torrent.hash]);
  const files = info.data?.file_stats || torrent.file_stats || [];
  const groups = new Map<string, TorrentFile[]>();
  for (const file of files) {
    const directory = file.path.split(/[\\/]/).slice(0, -1).join("/") || "/";
    groups.set(directory, [...(groups.get(directory) || []), file]);
  }
  const toggle = async (id: number, marked: boolean) => {
    setBusy(true);
    try {
      if (marked) await viewedApi.remove(torrent.hash, id);
      else await viewedApi.set(torrent.hash, id);
      await viewed.refetch();
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={torrent.title || torrent.hash}
      maxWidth="4xl"
    >
      <div className="space-y-4">
        <a
          className="inline-flex min-h-11 items-center underline"
          href={torrentsApi.getPlaylistUrl(torrent.hash)}
        >
          {t("torrent.playlist")}
        </a>
        {(info.isPending || viewed.isPending) && <Loading />}
        {info.error && (
          <RequestError error={info.error} retry={() => info.refetch()} />
        )}{" "}
        {viewed.error && (
          <RequestError error={viewed.error} retry={() => viewed.refetch()} />
        )}{" "}
        {!!error && <RequestError error={error} />}
        {!!files.length && <WebSeeds hash={torrent.hash} />}
        {!info.isPending && !files.length && <p>{t("torrent.waitMetadata")}</p>}
        {[...groups].map(([directory, items]) => (
          <section key={directory} className="space-y-2">
            <h3 className="font-mono break-all text-slate-400">{directory}</h3>
            {items.map((file) => {
              const episode = parseEpisodeInfo(file.path);
              const mark = viewed.data?.find((v) => v.file_index === file.id);
              const url = torrentsApi.getStreamUrl(torrent.hash, file.id);
              return (
                <article key={file.id} className="panel space-y-3">
                  <div className="flex justify-between gap-3">
                    <div className="min-w-0">
                      {episode.code && (
                        <span className="text-blue-300">{episode.code} · </span>
                      )}
                      <strong className="break-words">
                        {episode.displayTitle}
                      </strong>
                      <p className="text-sm text-slate-400 break-all">
                        {file.path} · {humanizeBytes(file.length)}
                      </p>
                    </div>
                    <Button
                      disabled={busy || !!viewed.error}
                      onClick={() => toggle(file.id, !!mark)}
                    >
                      {t(
                        mark?.timecode
                          ? "torrent.inProgress"
                          : mark
                            ? "Viewed"
                            : "torrent.unwatched",
                      )}
                      {mark?.timecode
                        ? ` · ${formatDuration(mark.timecode)}`
                        : ""}
                    </Button>
                  </div>
                  <PlaybackLinks
                    url={url}
                    onInternal={(playbackURL) =>
                      setPlayer({
                        url: playbackURL,
                        title: episode.displayTitle,
                        hash: torrent.hash,
                        index: file.id,
                        path: file.path,
                      })
                    }
                  />
                  <PrepareEpisode hash={torrent.hash} index={file.id} />
                </article>
              );
            })}
          </section>
        ))}
        <details onToggle={(e) => setCacheOpen(e.currentTarget.open)}>
          <summary>{t("Cache")}</summary>
          {cacheOpen && (
            <Suspense fallback={<Loading />}>
              <CacheMap hash={torrent.hash} />
            </Suspense>
          )}
        </details>
      </div>
      {player && (
        <Suspense fallback={<Loading />}>
          <VideoPlayer {...player} onClose={() => setPlayer(undefined)} />
        </Suspense>
      )}
    </Modal>
  );
}
