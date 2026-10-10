import { useTranslation } from "react-i18next";
import type { FlowStatusResponse } from "../../types/flow";

export function IncidentTimeline({ status }: { status: FlowStatusResponse }) {
  const { t, i18n } = useTranslation();
  if (!status.timeline) return null;
  return (
    <section className="panel space-y-3">
      <h3 className="font-semibold">{t("flow.timelineTitle")}</h3>
      <p className="text-sm text-slate-400">{t("flow.timelineHint")}</p>
      {status.sampled_at && (
        <p className="text-xs text-slate-400">
          {t("flow.sampledAt")}{" "}
          <time dateTime={status.sampled_at}>
            {new Date(status.sampled_at).toLocaleTimeString(
              i18n.language === "ua" ? "uk" : i18n.language,
            )}
          </time>
        </p>
      )}
      {status.timeline.dropped > 0 && (
        <p>{t("flow.timelineTruncated", { count: status.timeline.dropped })}</p>
      )}
      {!status.timeline.events.length && <p>{t("flow.noMetrics")}</p>}
      <ol className="max-h-80 overflow-auto space-y-2">
        {status.timeline.events.map((event, index) => (
          <li
            key={`${event.time}:${index}`}
            className="flex flex-wrap gap-2 text-sm"
          >
            <span className="font-mono min-w-20">
              +{(event.elapsed_ms / 1000).toFixed(2)} s
            </span>
            <span>
              {event.stage
                ? t(`flow.startupStages.${event.stage}`, {
                    defaultValue: t(`flow.timelineEvents.${event.type}`, {
                      defaultValue: t("status.unknown"),
                    }),
                  })
                : t(`flow.timelineEvents.${event.type}`, {
                    defaultValue: t("status.unknown"),
                  })}
            </span>
            {event.operation_ms >= 0 && (
              <span className="text-slate-400">{event.operation_ms} ms</span>
            )}
          </li>
        ))}
      </ol>
    </section>
  );
}
