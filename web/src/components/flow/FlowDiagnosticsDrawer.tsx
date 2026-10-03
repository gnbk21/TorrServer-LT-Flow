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
                  .filter(([k]) => k !== "traces" && k !== "file_index")
                  .map(([key, value]) => (
                    <div key={key}>
                      <dt className="text-sm text-slate-400">
                        {t(`flow.metrics.${key}`, {
                          defaultValue: key.replaceAll("_", " "),
                        })}
                      </dt>
                      <dd className="font-mono break-all">
                        {key === "buffer_ahead_seconds" &&
                        s.playback_consumption_rate <= 0
                          ? "—"
                          : redactDiagnostic(value)}
                      </dd>
                    </div>
                  ))}
              </dl>
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
