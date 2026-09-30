import { useTranslation } from "react-i18next";
import type { FlowSession } from "../../types/flow";
import { deriveFlowHealth } from "../../lib/health";
import { humanizeBytes, humanizeSpeed } from "../../lib/format";
import { Button } from "../common/Button";
import { PlaybackTroubleshooting } from "./PlaybackTroubleshooting";
export interface FlowHealthPanelProps {
  session?: FlowSession;
  title?: string;
  onOpenDiagnostics?: () => void;
  compact?: boolean;
}
export function FlowHealthPanel({
  session,
  title,
  onOpenDiagnostics,
}: FlowHealthPanelProps) {
  const { t } = useTranslation();
  const health = deriveFlowHealth(session);
  const known = !!session && session.playback_consumption_rate > 0;
  const values = [
    ["buffer", known ? `${session.buffer_ahead_seconds.toFixed(1)} s` : "—"],
    ["bufferBytes", humanizeBytes(session?.buffer_ahead_bytes)],
    [
      "download",
      humanizeSpeed(
        (session?.download_rate_samples ?? 0) > 0
          ? session?.recent_download_rate
          : session?.download_rate,
      ),
    ],
    ["demand", known ? humanizeSpeed(session.playback_consumption_rate) : "—"],
    [
      "sustainability",
      known ? `${session.sustainability_ratio.toFixed(2)}×` : "—",
    ],
    ["peers", session?.connected_peers ?? "—"],
    [
      "pieceWait",
      session
        ? `${(session.recent_piece_wait_p95_ms ?? session.piece_wait_p95_ms).toFixed(1)} ms`
        : "—",
    ],
    [
      "seek",
      session && session.seek_count > 0
        ? `${session.seek_recovery_ms} ms`
        : "—",
    ],
  ];
  return (
    <section className="panel space-y-4" aria-label={t("flow.title")}>
      <div className="flex justify-between flex-wrap gap-3">
        <h2 className="font-semibold">{title || t("flow.title")}</h2>
        <strong
          className={
            health.state === "HEALTHY"
              ? "text-emerald-400"
              : health.state === "BUFFER_RISK" || health.state === "STALLED"
                ? "text-rose-300"
                : "text-amber-300"
          }
        >
          {t(health.label)}
        </strong>
      </div>
      <p className="text-sm text-slate-300">{t(health.description)}</p>
      <dl className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {values.map(([key, value]) => (
          <div key={key}>
            <dt className="text-sm text-slate-400">{t(`flow.${key}`)}</dt>
            <dd className="font-mono text-lg">{value}</dd>
          </div>
        ))}
      </dl>
      <PlaybackTroubleshooting session={session} />
      {onOpenDiagnostics && (
        <Button onClick={onOpenDiagnostics}>{t("flow.diagnostics")}</Button>
      )}
    </section>
  );
}
