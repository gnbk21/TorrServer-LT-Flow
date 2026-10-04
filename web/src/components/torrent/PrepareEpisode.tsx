import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { preparationApi, type PreparationAction } from "../../api/preparation";
import { humanizeBytes } from "../../lib/format";
import { Button } from "../common/Button";
import { RequestError } from "../common/RequestState";

export function PrepareEpisode({
  hash,
  index,
}: {
  hash: string;
  index: number;
}) {
  const { t } = useTranslation();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const query = useQuery({
    queryKey: ["preparation"],
    queryFn: ({ signal }) => preparationApi.status(signal),
    refetchInterval: () => (document.hidden ? false : 2000),
  });
  const job = query.data?.jobs.find(
    (j) => j.hash === hash && j.file_index === index,
  );
  const act = async (action: PreparationAction) => {
    setBusy(true);
    setError(undefined);
    try {
      await preparationApi.control(hash, index, action);
      await query.refetch();
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <details className="rounded-lg border border-slate-700 p-3">
      <summary className="min-h-11 cursor-pointer">
        {t("preparation.title")}
        {job ? ` · ${t(`preparation.states.${job.state}`)}` : ""}
      </summary>
      <div className="space-y-3">
        <p className="text-sm text-slate-400">{t("preparation.help")}</p>
        {query.data && (
          <p className="text-xs text-slate-400">
            {t("preparation.quota", {
              used: humanizeBytes(query.data.reserved_bytes),
              total: humanizeBytes(query.data.quota_bytes),
            })}
          </p>
        )}
        {!!query.data?.error_code && (
          <p role="alert">
            {t(`preparation.errors.${query.data.error_code}`, {
              defaultValue: t("preparation.unavailable"),
            })}
          </p>
        )}
        {job && (
          <>
            <progress
              className="w-full"
              max={job.length}
              value={job.verified_bytes}
              aria-label={t("preparation.progress")}
            />
            <p>
              {humanizeBytes(job.verified_bytes)} / {humanizeBytes(job.length)}{" "}
              · {t(`preparation.states.${job.state}`)}
            </p>
            {job.playback_ready && (
              <p className="text-emerald-400">{t("preparation.readyHelp")}</p>
            )}
            {!!job.error_code && (
              <p role="alert">
                {t(`preparation.errors.${job.error_code}`, {
                  defaultValue: t("preparation.unavailable"),
                })}
              </p>
            )}
          </>
        )}
        <div className="flex flex-wrap gap-2">
          {!job && (
            <Button
              disabled={busy || query.isPending || !!query.data?.error_code}
              onClick={() => act("start")}
            >
              {t("preparation.title")}
            </Button>
          )}
          {job?.state === "downloading" && (
            <Button disabled={busy} onClick={() => act("pause")}>
              {t("preparation.pause")}
            </Button>
          )}
          {job && ["paused", "cancelled", "error"].includes(job.state) && (
            <Button disabled={busy} onClick={() => act("resume")}>
              {t("preparation.resume")}
            </Button>
          )}
          {job && ["downloading", "paused"].includes(job.state) && (
            <Button disabled={busy} onClick={() => act("cancel")}>
              {t("preparation.cancel")}
            </Button>
          )}
          {job && job.state !== "cleaning" && (
            <Button disabled={busy} onClick={() => act("remove")}>
              {t("preparation.remove")}
            </Button>
          )}
        </div>
      {!!(query.error || error) && (
          <RequestError
            error={error || query.error}
            retry={() => query.refetch()}
          />
        )}
        <p className="text-sm text-slate-400">{t("preparation.alternative")}</p>
      </div>
    </details>
  );
}
