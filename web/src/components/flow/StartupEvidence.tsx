import { useTranslation } from "react-i18next";
import type { FlowStartup } from "../../types/flow";

export function StartupEvidence({ startup }: { startup: FlowStartup }) {
  const { t } = useTranslation();
  const progress = startup.progress;
  const recovery = startup.metadata_recovery;
  if (!progress && !recovery) return null;
  const mib = (bytes: number) => (bytes / 1048576).toFixed(1);
  return (
    <section className="panel space-y-2">
      <h3 className="font-semibold">{t("flow.startupEvidenceTitle")}</h3>
      {recovery && (
        <div className="space-y-1 text-sm">
          <p>{t(`flow.metadataRecovery.${recovery.state}`)}</p>
          {recovery.attempts > 0 && (
            <p>{t("flow.metadataAttempts", { count: recovery.attempts })}</p>
          )}
          <p className="text-slate-400">{t("flow.metadataRecoveryHint")}</p>
        </div>
      )}
      {progress && (
        <>
          <p>{t(`flow.startupBlockers.${progress.blocked_on}`)}</p>
          {progress.wait_extended && (
            <p className="text-sm">
              {t("flow.startupWaitExtended", {
                seconds: Math.round((progress.wait_budget_ms ?? 0) / 1000),
              })}
            </p>
          )}
          <dl className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
            {(["head", "tail"] as const).map((part) => (
              <div key={part}>
                <dt>{t(`flow.startupEvidence.${part}`)}</dt>
                <dd>
                  {t("flow.startupVerified", {
                    ready: mib(progress[`${part}_verified_bytes`]),
                    target: mib(progress[`${part}_target_bytes`]),
                  })}
                </dd>
                <dd>
                  {t("flow.startupMissing", {
                    count:
                      progress[
                        part === "head"
                          ? "missing_head_pieces"
                          : "missing_tail_pieces"
                      ],
                  })}
                </dd>
                {progress[`${part}_blocker_piece`] !== undefined && (
                  <dd>
                    {t("flow.startupBlockerPiece", {
                      piece: progress[`${part}_blocker_piece`],
                    })}
                  </dd>
                )}
                {progress[`${part}_supply`] &&
                  (() => {
                    const supply = progress[`${part}_supply`]!;
                    return (
                      <>
                        <dd>
                          {t("flow.startupSuppliers", {
                            count: supply.suppliers,
                          })}
                        </dd>
                        {supply.unchoked_suppliers !== undefined && (
                          <dd>
                            {t("flow.startupUnchoked", {
                              count: supply.unchoked_suppliers,
                            })}
                          </dd>
                        )}
                        {supply.queue_known && (
                          <dd>
                            {t("flow.startupBlocks", {
                              pending: supply.requested_blocks,
                              waiting: supply.unrequested_blocks,
                              received:
                                supply.finished_blocks + supply.writing_blocks,
                            })}
                          </dd>
                        )}
                        {supply.truncated && (
                          <dd className="text-slate-400">
                            {t("flow.startupSupplyTruncated")}
                          </dd>
                        )}
                      </>
                    );
                  })()}
              </div>
            ))}
          </dl>
          <p className="text-sm text-slate-400">
            {t("flow.startupEvidenceHint")}
          </p>
          <p className="text-sm text-slate-400">
            {t("flow.startupSupplyHint")}
          </p>
        </>
      )}
    </section>
  );
}
