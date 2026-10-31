import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { FlowStatusResponse } from "../../types/flow";
import { playbackOverview } from "../../lib/playbackOverview";
import { humanizeBytes } from "../../lib/format";
import { flowApi } from "../../api/flow";
import { Button } from "../common/Button";
import { RequestError } from "../common/RequestState";
import { BufferingEvidence } from "./BufferingEvidence";

export function PlaybackOverview({
  status,
  now,
}: {
  status: FlowStatusResponse;
  now: number;
}) {
  const { t } = useTranslation();
  const view = playbackOverview(status, now);
  const [busy, setBusy] = useState(false);
  const [accepted, setAccepted] = useState(false);
  const [error, setError] = useState<unknown>();
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  useEffect(() => {
    if (!accepted) return;
    const timer = setTimeout(() => setAccepted(false), 30000);
    return () => clearTimeout(timer);
  }, [accepted]);
  const retry = async () => {
    const controller = new AbortController();
    request.current = controller;
    setBusy(true);
    setAccepted(false);
    setError(undefined);
    try {
      await flowApi.retryMetadata(status.hash, controller.signal);
      if (!controller.signal.aborted) setAccepted(true);
    } catch (e) {
      if (!controller.signal.aborted) setError(e);
    } finally {
      if (!controller.signal.aborted) setBusy(false);
    }
  };
  const session = view.session;
  const nativeAge = now - (status.sparse?.sampled_at_ms ?? NaN);
  return (
    <section className="panel space-y-3">
      <h3 className="font-semibold">{t("flow.upgrade.title")}</h3>
      <ol
        className="grid grid-cols-1 sm:grid-cols-5 gap-2 text-sm"
        aria-label={t("flow.timelineTitle")}
      >
        {view.steps.map((step) => (
          <li key={step.key} className="rounded-lg border border-slate-700 p-2">
            {t(`flow.journey.${step.key}`)}
            <br />
            <span className="text-slate-400">
              {t(
                !view.fresh
                  ? "status.unknown"
                  : step.complete
                    ? "flow.journey.observed"
                    : "flow.journey.unobserved",
              )}
            </span>
          </li>
        ))}
      </ol>
      <p role="status">{t(`flow.upgrade.causes.${view.cause}`)}</p>
      <p>{t(`flow.upgrade.actions.${view.action}`)}</p>
      <p className="text-sm text-slate-400">{t("flow.serverObservation")}</p>
      <dl className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
        <div>
          <dt>{t("flow.upgrade.verifiedBuffer")}</dt>
          <dd>
            {view.bufferFresh
              ? humanizeBytes(session?.verified_contiguous_bytes)
              : "—"}
          </dd>
        </div>
        <div>
          <dt>{t("flow.upgrade.readableBuffer")}</dt>
          <dd>
            {view.bufferFresh
              ? humanizeBytes(session?.readable_contiguous_bytes)
              : "—"}
          </dd>
        </div>
        <div>
          <dt>{t("flow.upgrade.age")}</dt>
          <dd>
            {view.age === null ? "—" : `${Math.floor(view.age / 1000)} s`}
          </dd>
        </div>
        <div>
          <dt>{t("flow.upgrade.confidence")}</dt>
          <dd>{t(`flow.confidence.${view.confidence}`)}</dd>
        </div>
      </dl>
      <aside className="rounded-lg border border-slate-700 p-3 space-y-2">
        <h4 className="font-semibold">{t("flow.adviser.title")}</h4>
        <p>{t(`flow.adviser.${view.adviser}`)}</p>
        <p className="text-sm text-slate-400">{t("flow.adviser.readOnly")}</p>
      </aside>
      {status.reuse && (
        <details className="text-sm space-y-2">
          <summary>{t("flow.reuse.title")}</summary>
          <p>
            {t(`flow.reuse.reasons.${status.reuse.reason}`, {
              defaultValue: t("status.unknown"),
            })}
          </p>
          <p>
            {t("flow.reuse.result", {
              candidates: status.reuse.candidates,
              accepted: status.reuse.native_accepted,
              verified: humanizeBytes(status.reuse.verified_bytes),
            })}
          </p>
          <p className="text-slate-400">{t("flow.reuse.hint")}</p>
        </details>
      )}
      {status.startup?.metadata_recovery && (
        <p className="text-sm">
          {t("flow.metadataRecoverySummary", {
            state: t(
              `flow.metadataRecovery.${status.startup.metadata_recovery.state}`,
              { defaultValue: t("status.unknown") },
            ),
            attempts: status.startup.metadata_recovery.attempts,
          })}
        </p>
      )}
      <BufferingEvidence
        session={session}
        fresh={view.bufferFresh}
        blockerFresh={
          view.bufferFresh &&
          status.sparse?.known === true &&
          nativeAge >= 0 &&
          nativeAge < 5000
        }
      />
      {status.integrity && (
        <p>
          {t("flow.upgrade.integrity", {
            count: status.integrity.recent_failures,
          })}
        </p>
      )}
      {view.cause === "metadata" && status.startup?.metadata_recovery && (
        <Button disabled={busy || accepted} onClick={() => void retry()}>
          {t("flow.upgrade.retry")}
        </Button>
      )}
      {accepted && <p role="status">{t("flow.upgrade.retryAccepted")}</p>}
      {!!error && <RequestError error={error} />}
    </section>
  );
}
