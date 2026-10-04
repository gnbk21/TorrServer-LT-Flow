import { redactDiagnostic } from "../../lib/redact";
import { useTranslation } from "react-i18next";
import { Modal } from "../common/Modal";
import type { FlowStatusResponse } from "../../types/flow";
import { useFlowDiagnostics } from "../../hooks/queries";
import { RequestError } from "../common/RequestState";
export interface FlowDiagnosticsDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  status?: FlowStatusResponse;
  torrentTitle?: string;
}
export function FlowDiagnosticsDrawer({
  isOpen,
  onClose,
  status: summary,
  torrentTitle,
}: FlowDiagnosticsDrawerProps) {
  const { t } = useTranslation();
  const diagnostics = useFlowDiagnostics(summary?.hash, isOpen);
  const status = diagnostics.data ?? summary;
  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t("flow.diagnostics")}
      subtitle={torrentTitle}
      maxWidth="4xl"
    >
      {diagnostics.error && (
        <RequestError
          error={diagnostics.error}
          stale={!!status}
          retry={() => diagnostics.refetch()}
        />
      )}
      {!status ? (
        <p>{t("flow.noMetrics")}</p>
      ) : (
        <div className="space-y-5">
          {status.sparse && (
            <section className="panel" aria-live="polite">
              <h3 className="font-semibold">{t("flow.sparseTitle")}</h3>
              {!status.sparse.known ||
              !status.sparse.sampled_at_ms ||
              Date.now() - status.sparse.sampled_at_ms < 0 ||
              Date.now() - status.sparse.sampled_at_ms >= 5000 ? (
                <p>{t("flow.sparseUnknown")}</p>
              ) : (
                <div className="space-y-2 mt-2 text-sm">
                  <p>
                    {t("flow.sparsePeers", {
                      useful: status.sparse.useful_peers,
                      total: status.sparse.sampled_peers,
                      downloading: status.sparse.useful_downloading_peers,
                    })}
                  </p>
                  <p>
                    {t("flow.sparseChoke", {
                      choked: status.sparse.choked_peers,
                      snubbed: status.sparse.snubbed_peers,
                      pending: status.sparse.pending_connections,
                    })}
                  </p>
                  <p>
                    {t("flow.sparseSources", {
                      tracker: status.sparse.tracker_peers,
                      dht: status.sparse.dht_peers,
                      pex: status.sparse.pex_peers,
                      incoming: status.sparse.incoming_peers,
                    })}
                  </p>
                  <p>
                    {t("flow.sparseQueue", {
                      blocks: status.sparse.queued_blocks,
                      bytes: status.sparse.outstanding_bytes,
                      ms: status.sparse.max_queue_ms,
                    })}
                  </p>
                  <p>
                    {t("flow.sparseFailures", {
                      failed: status.sparse.failed_bytes,
                      redundant: status.sparse.redundant_bytes,
                    })}
                  </p>
                  <p>
                    {t("flow.sparseRequests", {
                      timeouts: status.sparse.request_timeouts ?? 0,
                      dropped: status.sparse.requests_dropped ?? 0,
                    })}
                  </p>
                  {status.sparse.private && <p>{t("flow.privatePolicy")}</p>}
                  {status.sparse.truncated && (
                    <p>{t("flow.sparseTruncated")}</p>
                  )}
                  <p className="text-slate-400">{t("flow.sparseHint")}</p>
                </div>
              )}
            </section>
          )}
          {status.startup?.wait_reason && (
            <section className="panel" aria-live="polite">
              <h3 className="font-semibold">{t("flow.startupWait")}</h3>
              <p>
                {t(`flow.startupStages.${status.startup.wait_reason}`, {
                  defaultValue: "—",
                })}
              </p>
              <p className="text-sm text-slate-400 mt-2">
                {t("flow.startupTimingHint")}
              </p>
            </section>
          )}
          {Object.entries({
            startup: status.startup,
            network: status.network,
          }).map(([group, data]) => (
            <section key={group}>
              <h3 className="font-semibold mb-3">{t(`flow.${group}`)}</h3>
              {group === "network" && (
                <p className="text-sm text-slate-400 mb-3">
                  {t("flow.connectivityHint")}
                </p>
              )}
              <dl className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {Object.entries(data || {}).map(([key, value]) => (
                  <div key={key}>
                    <dt className="text-sm text-slate-400">
                      {t(`flow.metrics.${key}`, {
                        defaultValue: key.replaceAll("_", " "),
                      })}
                    </dt>
                    <dd className="font-mono break-all">
                      {key.endsWith("_ms") && typeof value === "number"
                        ? value < 0
                          ? "—"
                          : `${value} ms`
                        : key === "wait_reason" && typeof value === "string"
                          ? t(`flow.startupStages.${value}`, {
                              defaultValue: "—",
                            })
                          : typeof value === "object" && value !== null
                            ? redactDiagnostic(JSON.stringify(value))
                            : redactDiagnostic(value)}
                    </dd>
                  </div>
                ))}
              </dl>
            </section>
          ))}
          {status.sessions?.map((s) => (
            <section className="panel" key={s.group}>
              <h3>
                {t("flow.session")} · {s.group} ·{" "}
                {t("flow.fileIndex", { index: s.file_index })}
              </h3>
              <dl className="grid grid-cols-1 sm:grid-cols-3 gap-3 mt-4">
                {Object.entries(s)
                  .filter(
                    ([k]) =>
                      k !== "traces" &&
                      k !== "file_index" &&
                      k !== "required_piece_suppliers",
                  )
                  .map(([key, value]) => (
                    <div key={key}>
                      <dt className="text-sm text-slate-400">
                        {t(`flow.metrics.${key}`, {
                          defaultValue: key.replaceAll("_", " "),
                        })}
                      </dt>
                      <dd className="font-mono break-all">
                        {key === "wait_reason"
                          ? t(`flow.sparseReasons.${value}`, {
                              defaultValue: t("flow.sparseUnknown"),
                            })
                          : key === "buffer_ahead_seconds" &&
                              s.playback_consumption_rate <= 0
                            ? "—"
                            : redactDiagnostic(value)}
                      </dd>
                    </div>
                  ))}
              </dl>
              {s.required_piece_suppliers !== undefined && (
                <p className="mt-3 text-sm">
                  {t("flow.sparseSuppliers", {
                    count: s.required_piece_suppliers,
                  })}
                </p>
              )}
              {s.traces && (
                <details>
                  <summary>{t("flow.traces")}</summary>
                  <pre className="overflow-auto text-xs">
                    {JSON.stringify(s.traces, null, 2)}
                  </pre>
                </details>
              )}
            </section>
          ))}
          <section>
            <h3>{t("flow.trackers")}</h3>
            {status.trackers?.map((tr) => (
              <div key={tr.id} className="panel my-2 break-all">
                {tr.protocol} · {tr.host} · {tr.status} · {tr.peers}{" "}
                {t("flow.peers")}
                {tr.error && <p>{redactDiagnostic(tr.error)}</p>}
              </div>
            ))}
          </section>
        </div>
      )}
    </Modal>
  );
}
