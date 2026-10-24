import { useEffect, useState, lazy, Suspense } from "react";
import { useTranslation } from "react-i18next";
import { FlowHealthPanel } from "./FlowHealthPanel";
import {
  FlowHistoricalChart,
  type TelemetrySample,
} from "./FlowHistoricalChart";
import type { FlowStatusResponse, FlowSession } from "../../types/flow";
import type { Torrent } from "../../types/torrent";
import { Button } from "../common/Button";
import { Loading } from "../common/RequestState";
import { PlaybackLinks } from "../common/PlaybackLinks";
import { torrentsApi } from "../../api/torrents";
const Files = lazy(() =>
  import("../torrent/TorrentFilesDialog").then((module) => ({
    default: module.TorrentFilesDialog,
  })),
);
const Diagnostics = lazy(() =>
  import("./FlowDiagnosticsDrawer").then((m) => ({
    default: m.FlowDiagnosticsDrawer,
  })),
);
export default function SessionCard({
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
    const now = performance.now();
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
