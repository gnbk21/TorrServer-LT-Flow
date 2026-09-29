import { redactDiagnostic } from "../../lib/redact";
import { useTranslation } from "react-i18next";
import { Modal } from "../common/Modal";
import type { FlowStatusResponse } from "../../types/flow";
export interface FlowDiagnosticsDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  status?: FlowStatusResponse;
  torrentTitle?: string;
}
export function FlowDiagnosticsDrawer({
  isOpen,
  onClose,
  status,
  torrentTitle,
}: FlowDiagnosticsDrawerProps) {
  const { t } = useTranslation();
  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t("flow.diagnostics")}
      subtitle={torrentTitle}
      maxWidth="4xl"
    >
      {!status ? (
        <p>{t("flow.noMetrics")}</p>
      ) : (
        <div className="space-y-5">
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
                      {t(`metrics.${key}`, {
                        defaultValue: key.replaceAll("_", " "),
                      })}
                    </dt>
                    <dd className="font-mono break-all">
                      {redactDiagnostic(value)}
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
                        {t(`metrics.${key}`, {
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
