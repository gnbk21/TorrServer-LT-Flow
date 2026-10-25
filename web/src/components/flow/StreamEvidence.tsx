import { useTranslation } from "react-i18next";
import { humanizeBytes } from "../../lib/format";
import type { StreamEvidence as Evidence } from "../../types/flow";

export function StreamEvidence({
  evidence,
  fresh,
}: {
  evidence?: Evidence;
  fresh: boolean;
}) {
  const { t } = useTranslation();
  if (!evidence) return null;
  return (
    <section className="panel space-y-2">
      <h3 className="font-semibold">{t("flow.stream.title")}</h3>
      {!fresh || !evidence.known ? (
        <p>{t("flow.sparseUnknown")}</p>
      ) : (
        <>
          <dl className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
            {(
              ["useful_bytes", "redundant_bytes", "obsolete_bytes"] as const
            ).map((key) => (
              <div key={key}>
                <dt>{t("flow.stream." + key)}</dt>
                <dd>{humanizeBytes(evidence[key])}</dd>
              </div>
            ))}
            {(
              [
                "recovery_requests",
                "recovery_suppressed",
                "active_hedges",
                "peak_hedges",
                "budget_changes",
              ] as const
            ).map((key) => (
              <div key={key}>
                <dt>{t("flow.stream." + key)}</dt>
                <dd>{evidence[key].toLocaleString()}</dd>
              </div>
            ))}
          </dl>
          {evidence.request_policy?.enabled && (
            <div className="text-sm space-y-1">
              <p>{t("flow.buffering.requests", evidence.request_policy)}</p>
              {evidence.request_policy.qualified_peers > 0 && (
                <p>{t("flow.buffering.horizons", evidence.request_policy)}</p>
              )}
              {evidence.request_policy.truncated && (
                <p>{t("flow.startupSupplyTruncated")}</p>
              )}
            </div>
          )}
          {evidence.recovery_scan &&
            evidence.frontiers > 0 &&
            evidence.recovery_scan.age_ms >= 0 &&
            evidence.recovery_scan.age_ms < 3000 && (
              <p className="text-sm">
                {t("flow.buffering.recovery", evidence.recovery_scan)}
              </p>
            )}
          {evidence.service.length > 0 && (
            <ul className="text-sm space-y-1">
              {evidence.service.map((row) => (
                <li key={row.transport + "/" + row.address_family}>
                  {row.transport.toUpperCase()} / {row.address_family}:{" "}
                  {t("flow.stream.service", {
                    p50: row.p50_ms,
                    p95: row.p95_ms,
                    samples: row.samples,
                  })}
                </li>
              ))}
            </ul>
          )}
          {evidence.recovery_scan_truncated && (
            <p>{t("flow.startupSupplyTruncated")}</p>
          )}
          <p className="text-sm text-slate-400">{t("flow.stream.hint")}</p>
        </>
      )}
    </section>
  );
}
