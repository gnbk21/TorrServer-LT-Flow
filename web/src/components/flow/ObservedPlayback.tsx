import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useActive } from "../../hooks/queries";
import { useObservationTime } from "../../hooks/observationTime";

export function ObservedPlayback() {
  const { t } = useTranslation();
  const activity = useActive();
  const rows =
    activity.data?.items.filter(
      (row) => (row.torrent.active_readers || 0) > 0 || row.torrent.warm_idle,
    ) || [];
  const row = rows[0];
  const session = row?.status.sessions?.find(
    (session) => session.active_readers > 0,
  );
  const sampledAt = Date.parse(row?.status.sampled_at ?? "");
  const now = useObservationTime(
    !!row,
    sampledAt + 10000,
    (session?.buffer_sampled_at_ms ?? NaN) + 10000,
    row?.status,
  );
  const snapshotFresh =
    Number.isFinite(now) && now >= sampledAt && now - sampledAt < 10000;
  const fresh =
    snapshotFresh &&
    !!session?.buffer_sampled_at_ms &&
    now >= session.buffer_sampled_at_ms &&
    now - session.buffer_sampled_at_ms < 10000;
  if (!row) return null;
  return (
    <div className="observed-playback px-4 py-2 border-b border-slate-800 text-sm">
      <Link
        to="/"
        className="flex flex-wrap gap-x-3 gap-y-1 min-h-11 items-center"
      >
        <strong>{t("flow.observedPlayback")}</strong>
        <span className="truncate max-w-72">
          {row.torrent.title || row.torrent.hash}
        </span>
        <span role="status">
          {t(
            !snapshotFresh
              ? "status.stale"
              : row.torrent.warm_idle
                ? "flow.warm"
                : "torrent.playing",
          )}
        </span>
        <span>
          {t("flow.buffer")}:{" "}
          {session?.playback_consumption_rate &&
          session.playback_consumption_rate > 0 &&
          fresh
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
