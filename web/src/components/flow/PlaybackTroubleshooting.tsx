import { useTranslation } from "react-i18next";
import type { FlowSession } from "../../types/flow";

export function PlaybackTroubleshooting({
  session,
}: {
  session?: FlowSession;
}) {
  const { t } = useTranslation();
  const conditions = ["delivery"];
  if (
    session?.bitrate_estimate_confidence === "low" ||
    !session?.playback_consumption_rate
  )
    conditions.push("estimate");
  if (session?.buffer_warning) conditions.push("draining");
  if (session?.risk?.level === "HIGH") conditions.push("prepare");
  if ((session?.recent_server_read_stalls ?? 0) > 0) conditions.push("waiting");
  if (session?.state === "WARM_IDLE") conditions.push("idle");
  if (
    session?.active_readers &&
    session.connected_peers === 0 &&
    session.buffer_ahead_bytes === 0
  )
    conditions.push("peers");
  return (
    <details className="text-sm space-y-3">
      <summary>{t("flow.troubleshooting.title")}</summary>
      <ul className="list-disc pl-5 space-y-2">
        {conditions.map((key) => (
          <li key={key}>{t(`flow.troubleshooting.${key}`)}</li>
        ))}
      </ul>
    </details>
  );
}
