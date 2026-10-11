import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { readPlayerReport, type PlayerReport } from "../../lib/playerEvidence";
import type { FlowStatusResponse } from "../../types/flow";
import { RequestError } from "../common/RequestState";
import { Button } from "../common/Button";

export function PlayerEvidence({ status }: { status: FlowStatusResponse }) {
  const { t } = useTranslation();
  const [report, setReport] = useState<PlayerReport>();
  const [error, setError] = useState<unknown>();
  const importGeneration = useRef(0);
  useEffect(
    () => () => {
      importGeneration.current++;
    },
    [],
  );
  const latest = report?.samples.at(-1);
  const matched =
    !!latest?.session_id &&
    status.sessions?.some(
      (session) => session.session_id === latest.session_id,
    );
  return (
    <details className="panel space-y-3">
      <summary>{t("flow.playerEvidence")}</summary>
      <p className="text-sm text-slate-400">{t("flow.playerEvidenceHint")}</p>
      <label className="field">
        {t("flow.playerImport")}
        <input
          type="file"
          accept="application/json,.json"
          onChange={async (event) => {
            const file = event.target.files?.[0];
            if (!file) return;
            const generation = ++importGeneration.current;
            event.target.value = "";
            setError(undefined);
            setReport(undefined);
            try {
              const imported = await readPlayerReport(file);
              if (generation === importGeneration.current) setReport(imported);
            } catch (failure) {
              if (generation === importGeneration.current) setError(failure);
            }
          }}
        />
      </label>
      {!!error && <RequestError error={error} />}
      {latest ? (
        <>
          <p>{t(matched ? "flow.playerMatched" : "flow.playerUnmatched")}</p>
          <p className="text-sm text-slate-400">
            {t("flow.playerClockUnknown")}
          </p>
          <dl className="grid grid-cols-2 gap-3">
            {(
              [
                "startup_ms",
                "rebuffer_events",
                "rebuffer_ms",
                "seek_events",
                "seek_buffer_ms",
                "buffered_ms",
                "dropped_frames",
                "paused_ms",
              ] as const
            ).map((key) => (
              <div key={key}>
                <dt>{t(`flow.playerMetrics.${key}`)}</dt>
                <dd className="font-mono">{latest[key] ?? "—"}</dd>
              </div>
            ))}
          </dl>
          <Button
            onClick={() => {
              importGeneration.current++;
              setReport(undefined);
              setError(undefined);
            }}
          >
            {t("flow.clearPlayerReport")}
          </Button>
        </>
      ) : (
        <p>{t("flow.playerUnknown")}</p>
      )}
    </details>
  );
}
