import { useState, lazy, Suspense } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { ApiError, pollingDelay } from "../../api/client";
import { recentApi } from "../../api/recent";
import { torrentsApi } from "../../api/torrents";
import { viewedApi } from "../../api/viewed";
import { useVisibilityGate } from "../../hooks/queries";
import { formatDuration } from "../../lib/format";
import { Button } from "../common/Button";
import { Loading, RequestError } from "../common/RequestState";
import { PlaybackLinks } from "../common/PlaybackLinks";
const Modal = lazy(() =>
  import("../common/Modal").then((module) => ({ default: module.Modal })),
);
const Player = lazy(() => import("./VideoPlayer"));
export function RecentPlayback() {
  const { t } = useTranslation();
  const client = useQueryClient();
  const visible = useVisibilityGate(["recent"]);
  const recent = useQuery({
    queryKey: ["recent"],
    queryFn: ({ signal }) => recentApi.get(signal),
    enabled: visible,
    retry: false,
    refetchInterval: visible ? pollingDelay(15000) : false,
  });
  const [confirm, setConfirm] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>();
  const [player, setPlayer] = useState<{
    url: string;
    hash: string;
    index: number;
    path: string;
    title: string;
    startSeconds?: number;
  }>();
  const [resuming, setResuming] = useState<string>();
  const [limit, setLimit] = useState(10);
  const mutate = async (work: () => Promise<unknown>) => {
    setBusy(true);
    setError(undefined);
    try {
      await work();
      await client.invalidateQueries({ queryKey: ["recent"] });
      setConfirm(false);
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  const unsupported =
    recent.error instanceof ApiError && recent.error.status === 404;
  return (
    <section className="panel min-h-52 space-y-4">
      <h2 className="text-xl">{t("recent.title")}</h2>
      <p className="text-sm">{t("recent.hint")}</p>
      {recent.isPending && <Loading />}
      {unsupported && <p role="status">{t("recent.unavailable")}</p>}
      {recent.error && !unsupported && (
        <RequestError error={recent.error} retry={() => recent.refetch()} />
      )}
      {!!error && <RequestError error={error} />}
      {recent.data && !recent.data.enabled && <p>{t("recent.disabled")}</p>}
      {recent.data?.enabled && !recent.data.items.length && (
        <p>{t("recent.empty")}</p>
      )}
      {recent.data?.items.slice(0, limit).map((row) => {
        const key = `${row.hash}:${row.file_index}`;
        const position = row.viewed?.timecode ?? 0;
        return (
          <article key={key} className="panel space-y-3">
            <h3 className="font-semibold break-words">{row.title}</h3>
            <p className="font-mono text-sm break-all">{row.file_path}</p>
            <p>
              {t("recent.opened")}:{" "}
              {new Date(row.last_opened_at).toLocaleString()}
            </p>
            {position > 0 && (
              <p>
                {t(
                  row.viewed?.position_source === "legacy_saved" ||
                    !row.viewed?.position_source
                    ? "recent.legacyPosition"
                    : "recent.position",
                )}
                : {formatDuration(position)}
                {row.viewed?.position_updated_at
                  ? ` · ${new Date(row.viewed.position_updated_at).toLocaleString()}`
                  : ""}
              </p>
            )}
            {row.viewed?.completed !== undefined && (
              <p>
                {t(
                  row.viewed.completed
                    ? "recent.complete"
                    : "recent.notComplete",
                )}
              </p>
            )}
            {position > 0 && (
              <Button
                aria-pressed={resuming === key}
                onClick={() => setResuming(resuming === key ? undefined : key)}
              >
                {t("recent.resume")} · {formatDuration(position)}
              </Button>
            )}
            <PlaybackLinks
              url={torrentsApi.getStreamUrl(row.hash, row.file_index)}
              startSeconds={resuming === key ? position : undefined}
              onInternal={(url) =>
                setPlayer({
                  url,
                  hash: row.hash,
                  index: row.file_index,
                  path: row.file_path,
                  title: row.title,
                  startSeconds: resuming === key ? position : 0,
                })
              }
            />
            <p className="text-xs">{t("recent.handoff")}</p>
            <div className="actions">
              <Button
                disabled={busy || recent.data?.read_only}
                onClick={() =>
                  void mutate(() =>
                    viewedApi.complete(
                      row.hash,
                      row.file_index,
                      !row.viewed?.completed,
                    ),
                  )
                }
              >
                {t(
                  row.viewed?.completed
                    ? "recent.uncomplete"
                    : "recent.markComplete",
                )}
              </Button>
              <Button
                disabled={busy || recent.data?.read_only}
                onClick={() =>
                  void mutate(() => recentApi.remove(row.hash, row.file_index))
                }
              >
                {t("Delete")}
              </Button>
            </div>
          </article>
        );
      })}
      {recent.data && recent.data.items.length > limit && (
        <Button onClick={() => setLimit((value) => value + 10)}>
          {t("torrent.moreFiles")} ({recent.data.items.length - limit})
        </Button>
      )}
      {!!recent.data?.items.length && (
        <Button
          disabled={busy || recent.data.read_only}
          onClick={() => setConfirm(true)}
        >
          {t("recent.clear")}
        </Button>
      )}
      {confirm && (
        <Suspense fallback={<Loading />}>
          <Modal
            isOpen
            onClose={() => setConfirm(false)}
            title={t("recent.clear")}
          >
            <p>{t("recent.clearHint")}</p>
            <div className="actions">
              <Button
                disabled={busy}
                variant="danger"
                onClick={() => void mutate(recentApi.clear)}
              >
                {t("recent.clear")}
              </Button>
              <Button onClick={() => setConfirm(false)}>{t("Cancel")}</Button>
            </div>
          </Modal>
        </Suspense>
      )}
      {player && (
        <Suspense fallback={<Loading />}>
          <Player {...player} onClose={() => setPlayer(undefined)} />
        </Suspense>
      )}
    </section>
  );
}
