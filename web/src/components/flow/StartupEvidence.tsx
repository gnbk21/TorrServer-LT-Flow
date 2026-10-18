import { useTranslation } from "react-i18next";
import type { FlowStartup } from "../../types/flow";

export function StartupEvidence({ startup }: { startup: FlowStartup }) {
  const { t } = useTranslation();
  const progress = startup.progress;
  if (!progress) return null;
  const mib = (bytes: number) => (bytes / 1048576).toFixed(1);
  return (
    <section className="panel space-y-2" aria-live="polite">
      <h3 className="font-semibold">{t("flow.startupEvidenceTitle")}</h3>
      <p>{t(`flow.startupBlockers.${progress.blocked_on}`)}</p>
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
          </div>
        ))}
      </dl>
      <p className="text-sm text-slate-400">{t("flow.startupEvidenceHint")}</p>
    </section>
  );
}
