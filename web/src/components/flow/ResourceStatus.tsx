import { useTranslation } from "react-i18next";
import type { RuntimeStatus } from "../../types/runtime";
import { humanizeBytes } from "../../lib/format";

export function ResourceStatus({ status }: { status?: RuntimeStatus }) {
  const { t } = useTranslation();
  if (!status) return null;
  const resources = status.resources;
  const exposure = status.exposure;
  return (
    <div className="space-y-3">
      {resources && (
        <>
          <dl className="grid grid-cols-2 lg:grid-cols-4 gap-3 text-sm">
            {[
              ["ramBudget", humanizeBytes(resources.budget_bytes)],
              ["warmBudget", humanizeBytes(resources.warm_budget_bytes)],
              ["ramCache", humanizeBytes(resources.ram_resident_bytes)],
              ["diskCache", humanizeBytes(resources.disk_cache_bytes)],
              [
                "systemFree",
                status.memory?.system_available
                  ? humanizeBytes(status.memory.system_available_bytes)
                  : "—",
              ],
            ].map(([key, value]) => (
              <div key={key}>
                <dt className="text-slate-400">{t(`runtime.${key}`)}</dt>
                <dd>{value}</dd>
              </div>
            ))}
          </dl>
          <p className="text-xs text-slate-400">{t("runtime.budgetHint")}</p>
          {resources.background_limited && (
            <p role="status">{t("runtime.backgroundLimited")}</p>
          )}
          {resources.protected_overcommit && (
            <p role="status">{t("runtime.overcommit")}</p>
          )}
        </>
      )}
      {status.bt?.upload_cap_busy && (
        <p role="status">{t("runtime.uploadBusy")}</p>
      )}
      {exposure && (
        <details className="text-sm space-y-2">
          <summary>{t("runtime.exposure")}</summary>
          <p>
            {t("runtime.accessSummary", {
              auth: t(
                exposure.http_auth
                  ? "runtime.protected"
                  : "runtime.unprotected",
              ),
              playback: t(
                exposure.legacy_playback
                  ? "runtime.compatible"
                  : "runtime.capabilities",
              ),
            })}
          </p>
          <p>{t("runtime.localExposure")}</p>
          {exposure.torrent_interface.name && (
            <>
              <p>
                {exposure.torrent_interface.name} ·{" "}
                {exposure.torrent_interface.state}
              </p>
              <p>{t("runtime.bindingLimit")}</p>
            </>
          )}
        </details>
      )}
    </div>
  );
}
