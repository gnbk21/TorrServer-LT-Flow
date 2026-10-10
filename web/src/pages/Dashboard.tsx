import { redactDiagnostic } from "../lib/redact";
import { useEffect, useState, lazy, Suspense } from "react";
import { useTranslation } from "react-i18next";
import {
  useActive,
  useRuntime,
  useNetwork,
  useSettings,
} from "../hooks/queries";
import { FlowHealthPanel } from "../components/flow/FlowHealthPanel";
import { ResourceStatus } from "../components/flow/ResourceStatus";
import {
  FlowHistoricalChart,
  type TelemetrySample,
} from "../components/flow/FlowHistoricalChart";
import type { FlowStatusResponse, FlowSession } from "../types/flow";
import { Loading, RequestError } from "../components/common/RequestState";
import { humanizeBytes, humanizeSpeed } from "../lib/format";
import { Link } from "react-router-dom";
import { PlaybackLinks } from "../components/common/PlaybackLinks";
import { Button } from "../components/common/Button";
import { torrentsApi } from "../api/torrents";
import type { Torrent } from "../types/torrent";
const GstRuntime = lazy(() => import("../components/flow/GstRuntimeStatus"));
const Files = lazy(() =>
  import("../components/torrent/TorrentFilesDialog").then((module) => ({
    default: module.TorrentFilesDialog,
  })),
);
const Diagnostics = lazy(() =>
  import("../components/flow/FlowDiagnosticsDrawer").then((m) => ({
    default: m.FlowDiagnosticsDrawer,
  })),
);
function SessionCard({
  status,
  session,
  title,
  sampledAt,
  torrent,
}: {
  status: FlowStatusResponse;
  session: FlowSession;
  title: string;
  sampledAt: number;
  torrent?: Torrent;
}) {
  const { t } = useTranslation();
  const [samples, setSamples] = useState<TelemetrySample[]>([]);
  const [open, setOpen] = useState(false);
  const [filesOpen, setFilesOpen] = useState(false);
  useEffect(() => {
    const now = Date.now();
    setSamples((previous) =>
      [
        ...previous.filter((s) => s.timestamp >= now - 120000),
        {
          timestamp: now,
          bufferSeconds:
            session.playback_consumption_rate > 0
              ? session.buffer_ahead_seconds
              : null,
          downloadMbps: (session.download_rate * 8) / 1e6,
          demandMbps:
            session.playback_consumption_rate > 0
              ? (session.playback_consumption_rate * 8) / 1e6
              : null,
        },
      ].slice(-120),
    );
  }, [session, sampledAt]);
  return (
    <div className="space-y-4">
      <FlowHealthPanel
        session={session}
        title={title}
        onOpenDiagnostics={() => setOpen(true)}
      />
      <div className="panel actions">
        {torrent && (
          <Button onClick={() => setFilesOpen(true)}>
            {t("torrent.files")}
          </Button>
        )}
        {session.file_index > 0 && (
          <PlaybackLinks
            url={torrentsApi.getStreamUrl(status.hash, session.file_index)}
          />
        )}
      </div>
      <FlowHistoricalChart samples={samples} />
      {open && (
        <Suspense fallback={<Loading />}>
          <Diagnostics
            isOpen
            onClose={() => setOpen(false)}
            status={status}
            torrentTitle={title}
          />
        </Suspense>
      )}
      {filesOpen && torrent && (
        <Suspense fallback={<Loading />}>
          <Files isOpen torrent={torrent} onClose={() => setFilesOpen(false)} />
        </Suspense>
      )}
    </div>
  );
}
export default function Dashboard() {
  const { t } = useTranslation();
  const activity = useActive();
  const runtime = useRuntime();
  const network = useNetwork();
  const settings = useSettings();
  const [gstOpen, setGstOpen] = useState(false);
  const items = activity.data?.items || [];
  const active = items.flatMap(({ torrent, status }) =>
    (status.sessions || [])
      .filter(
        (session) =>
          session.active_readers > 0 || session.state === "WARM_IDLE",
      )
      .map((session) => ({
        status,
        session,
        torrent,
        sampledAt: activity.dataUpdatedAt,
        title: [
          torrent.title || status.hash,
          torrent.file_stats?.find((file) => file.id === session.file_index)
            ?.path,
        ]
          .filter(Boolean)
          .join(" · "),
      })),
  );
  const unmeasured = items
    .filter(
      ({ torrent, status }) =>
        (torrent.active_readers ?? 0) > 0 &&
        !status.sessions?.some((session) => session.active_readers > 0),
    )
    .map((item) => item.torrent);
  const [unmeasuredFiles, setUnmeasuredFiles] = useState<Torrent>();
  const bt = runtime.data?.bt;
  return (
    <div
      className="space-y-5"
      aria-busy={activity.isPending || runtime.isPending}
    >
      <h1 className="text-2xl font-semibold">{t("nav.dashboard")}</h1>
      {(activity.isPending || runtime.isPending) && (
        <span role="status" className="sr-only">
          {t("Loading")}
        </span>
      )}
      {activity.error && (
        <RequestError
          error={activity.error}
          stale={!!activity.data}
          retry={() => activity.refetch()}
        />
      )}{" "}
      {runtime.error && (
        <RequestError
          error={runtime.error}
          stale={!!runtime.data}
          retry={() => runtime.refetch()}
        />
      )}
      {active.length || unmeasured.length ? (
        <>
          {active.map((row) => (
            <SessionCard
              key={`${row.status.hash}:${row.session.group}:${row.session.file_index}`}
              {...row}
            />
          ))}
          {unmeasured.map((row) => (
            <section className="space-y-3" key={row.hash}>
              <FlowHealthPanel title={row.title} />
              <p>{t("flow.noMetrics")}</p>
              <Button onClick={() => setUnmeasuredFiles(row)}>
                {t("torrent.files")}
              </Button>
            </section>
          ))}
        </>
      ) : activity.isPending ? (
        <section className="panel min-h-52 flex items-center" role="status">
          {t("Loading")}
        </section>
      ) : (
        <section className="panel min-h-52 space-y-3">
          <h2 className="text-xl">{t("dashboard.idle")}</h2>
          <p className="text-slate-300">{t("dashboard.idleHint")}</p>
          <div className="actions">
            <Link
              className="inline-flex items-center min-h-11 px-4 bg-blue-600 rounded-xl"
              to="/add"
            >
              {t("Add")}
            </Link>
            <Link
              className="inline-flex items-center min-h-11 px-4 border rounded-xl"
              to="/torrents"
            >
              {t("nav.torrents")}
            </Link>
          </div>
        </section>
      )}
      <section className="panel space-y-4">
        <h2 className="font-semibold">{t("dashboard.server")}</h2>
        <ResourceStatus status={runtime.data} />
        <dl className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {[
            ["torrents", bt?.torrent_count],
            ["activeStreams", bt?.active_streams],
            ["download", humanizeSpeed(bt?.download_speed)],
            ["upload", humanizeSpeed(bt?.upload_speed)],
            ["loaded", humanizeBytes(bt?.loaded_size)],
            ["total", humanizeBytes(bt?.total_size)],
            ["capacity", humanizeBytes(settings.data?.CacheSize)],
            ["peers", bt?.active_peers],
            ["seeders", bt?.connected_seeders],
            ["listenPort", bt?.listen_port],
          ].map(([key, value]) => (
            <div key={key}>
              <dt className="text-sm text-slate-400">
                {key === "capacity"
                  ? t("SettingsDialog.CacheSize")
                  : t(`runtime.${key}`)}
              </dt>
              <dd className="font-mono">{value ?? "—"}</dd>
            </div>
          ))}
        </dl>
        {bt && (
          <details>
            <summary>{t("dashboard.fullRuntime")}</summary>
            <dl className="grid sm:grid-cols-2 gap-3">
              {Object.entries(bt)
                .filter(([key]) => key !== "raw_stat" && key !== "torrents")
                .map(([key, value]) => (
                  <div key={key}>
                    <dt>
                      {t(`metrics.${key}`, {
                        defaultValue: key.replaceAll("_", " "),
                      })}
                    </dt>
                    <dd>{String(value)}</dd>
                  </div>
                ))}
            </dl>
            <div className="overflow-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr>
                    <th>{t("torrent.title")}</th>
                    <th>{t("flow.download")}</th>
                    <th>{t("flow.peers")}</th>
                  </tr>
                </thead>
                <tbody>
                  {bt.torrents.map((row) => (
                    <tr key={row.hash}>
                      <td className="p-2">{row.title}</td>
                      <td>{humanizeSpeed(row.download_speed)}</td>
                      <td>{row.active_peers}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <details>
              <summary>{t("ShowRawStat")}</summary>
              <pre className="overflow-auto text-xs max-h-96">
                {redactDiagnostic(bt.raw_stat)}
              </pre>
            </details>
          </details>
        )}
      </section>
      <section className="panel space-y-3">
        <h2>{t("dashboard.network")}</h2>
        {network.error ? (
          <RequestError error={network.error} retry={() => network.refetch()} />
        ) : (
          <>
            <p>
              {network.data?.state} · {network.data?.connectivity}
            </p>
            <p className="break-all">{network.data?.addresses?.join(", ")}</p>
          </>
        )}
      </section>
      {runtime.data && (
        <section className="panel space-y-3">
          <h2>{t("settings.integrationStatus")}</h2>
          <dl className="grid sm:grid-cols-2 gap-3">
            {[
              ["DLNA", runtime.data.dlna_enabled],
              ["Bonjour", runtime.data.bonjour_enabled],
              ["WebDAV", runtime.data.webdav_enabled],
              ["FUSE", runtime.data.fuse_enabled],
            ].map(([name, enabled]) => (
              <div key={String(name)}>
                <dt>{name}</dt>
                <dd>{t(enabled ? "status.enabled" : "status.disabled")}</dd>
              </div>
            ))}
          </dl>
          {runtime.data.webdav_enabled && (
            <a
              className="inline-flex min-h-11 items-center underline"
              href={runtime.data.webdav_path}
            >
              WebDAV
            </a>
          )}
          {runtime.data.fuse_enabled && (
            <p className="break-all">{runtime.data.fuse_path}</p>
          )}
          <details onToggle={(event) => setGstOpen(event.currentTarget.open)}>
            <summary>GStreamer</summary>
            {gstOpen && (
              <Suspense fallback={<Loading />}>
                <GstRuntime />
              </Suspense>
            )}
          </details>
        </section>
      )}
      {unmeasuredFiles && (
        <Suspense fallback={<Loading />}>
          <Files
            isOpen
            torrent={unmeasuredFiles}
            onClose={() => setUnmeasuredFiles(undefined)}
          />
        </Suspense>
      )}
    </div>
  );
}
