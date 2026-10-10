import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useActive } from "../../hooks/queries";

export function ObservedPlayback() {
  const { t } = useTranslation();
  const activity = useActive();
  const rows =
    activity.data?.items.filter(
      (row) => (row.torrent.active_readers || 0) > 0 || row.torrent.warm_idle,
    ) || [];
  const row = rows[0];
  if (!row) return null;
  const session = row.status.sessions?.find(
    (session) => session.active_readers > 0,
  );
  return (
    <div
      className="observed-playback px-4 py-2 border-b border-slate-800 text-sm"
      role="status"
    >
      <Link
        to="/"
        className="flex flex-wrap gap-x-3 gap-y-1 min-h-11 items-center"
      >
        <strong>{t("flow.observedPlayback")}</strong>
        <span className="truncate max-w-72">
          {row.torrent.title || row.torrent.hash}
        </span>
        <span>
          {t(row.torrent.warm_idle ? "flow.warm" : "torrent.playing")}
        </span>
        <span>
          {t("flow.buffer")}:{" "}
          {session?.playback_consumption_rate &&
          session.playback_consumption_rate > 0
            ? `${session.buffer_ahead_seconds.toFixed(1)} s`
            : "—"}
        </span>
        {rows.length > 1 && <span>+{rows.length - 1}</span>}
        {activity.error && <span>{t("status.stale")}</span>}
      </Link>
      <p className="text-xs text-slate-400">{t("flow.serverObservation")}</p>
    </div>
  );
}
