import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { flowApi } from "../../api/flow";
import { torrentsApi } from "../../api/torrents";
import { torrentSchema } from "../../api/schemas";
import type { FlowStatusResponse } from "../../types/flow";
import { Button } from "../common/Button";
import { RequestError } from "../common/RequestState";
import { humanizeBytes as bytes } from "../../lib/format";

export function NextEpisodeWarmup({ status }: { status: FlowStatusResponse }) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const [choosing, setChoosing] = useState(false);
  const [selected, setSelected] = useState("");
  const files = useQuery({
    queryKey: ["warmup-files", status.hash],
    queryFn: async ({ signal }) =>
      torrentSchema.parse(await torrentsApi.get(status.hash, signal)),
    enabled: choosing,
  });
  const change = useMutation({
    mutationFn: ({
      index,
      action,
    }: {
      index: number;
      action: "select" | "auto" | "cancel";
    }) => flowApi.warmup(status.hash, index, action),
    onSuccess: async () => {
      setChoosing(false);
      await client.invalidateQueries({ queryKey: ["flow", status.hash] });
      await client.invalidateQueries({
        queryKey: ["flow-diagnostics", status.hash],
      });
    },
  });
  const warmup = status.next_episode_warmup;
  if (!warmup) return null;
  const choices =
    files.data?.file_stats?.filter(
      (file) =>
        file.id !== warmup.current_file_index &&
        file.length > 0 &&
        /\.(mkv|mp4|m4v|avi|ts|webm)$/i.test(file.path),
    ) ?? [];
  return (
    <section className="panel space-y-3">
      <h3 className="font-semibold">{t("flow.warmupTitle")}</h3>
      <p className="text-sm text-slate-400">{t("flow.warmupHint")}</p>
      <p role="status">
        {!warmup.enabled
          ? t("flow.warmupDisabled")
          : t(`flow.warmupStates.${warmup.state}`)}
        {warmup.file_index > 0 && (
          <>
            {" "}
            · {t("flow.warmupFile", { index: warmup.file_index })} ·{" "}
            {bytes(warmup.verified_bytes)} / {bytes(warmup.budget_bytes)}
          </>
        )}
      </p>
      {warmup.reason && (
        <p className="text-sm text-slate-400">
          {t(`flow.warmupReasons.${warmup.reason}`, {
            defaultValue: t("flow.noMetrics"),
          })}
        </p>
      )}
      {warmup.enabled && warmup.current_file_index > 0 && (
        <div className="flex flex-wrap gap-2">
          <Button
            disabled={change.isPending}
            onClick={() => setChoosing(!choosing)}
          >
            {t("flow.warmupChoose")}
          </Button>
          <Button
            isLoading={change.isPending}
            onClick={() => change.mutate({ index: 0, action: "auto" })}
          >
            {t("flow.warmupAuto")}
          </Button>
          <Button
            disabled={change.isPending}
            onClick={() => change.mutate({ index: 0, action: "cancel" })}
          >
            {t("flow.warmupCancel")}
          </Button>
        </div>
      )}
      {choosing && (
        <div className="space-y-2">
          {files.error && (
            <RequestError error={files.error} retry={() => files.refetch()} />
          )}
          <label className="field">
            {t("flow.warmupChoose")}
            <select
              value={selected}
              onChange={(event) => setSelected(event.target.value)}
              disabled={files.isPending || change.isPending}
            >
              <option value="">{t("flow.warmupChoose")}</option>
              {choices.map((file) => (
                <option key={file.id} value={file.id}>
                  {file.path}
                </option>
              ))}
            </select>
          </label>
          <Button
            disabled={!selected || change.isPending}
            isLoading={change.isPending}
            onClick={() =>
              change.mutate({ index: Number(selected), action: "select" })
            }
          >
            {t("flow.warmupStart")}
          </Button>
        </div>
      )}
      {change.error && <RequestError error={change.error} />}
    </section>
  );
}
