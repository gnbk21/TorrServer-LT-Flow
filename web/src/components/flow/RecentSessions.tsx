import { useState, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import {
  sessionHistoryApi,
  type SessionSummaries,
} from "../../api/sessionHistory";
import { Button } from "../common/Button";
import { RequestError } from "../common/RequestState";
import { LANTest } from "../settings/LANTest";
import { humanizeBytes } from "../../lib/format";

export function RecentSessions() {
  const { t } = useTranslation();
  const [data, setData] = useState<SessionSummaries>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const controller = useRef<AbortController>(undefined);
  useEffect(() => () => controller.current?.abort(), []);
  const refresh = async () => {
    controller.current?.abort();
    const request = new AbortController();
    controller.current = request;
    setBusy(true);
    setError(undefined);
    try {
      setData(await sessionHistoryApi.getSessions(request.signal));
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="space-y-4">
      <details>
        <summary>{t("reliability.guide")}</summary>
        <div className="space-y-3 py-3">
          {(["parser", "preload", "playback"] as const).map((key) => (
            <p key={key}>{t(`reliability.guideSteps.${key}`)}</p>
          ))}
          <Link className="underline" to="/settings?tab=integrations">
            {t("settings.integrationStatus")}
          </Link>
          <LANTest />
        </div>
      </details>
      <p className="text-sm text-slate-400">{t("reliability.historyHelp")}</p>
      <Button disabled={busy} onClick={() => void refresh()}>
        {t("reliability.recentSessions")}
      </Button>
      {!!error && <RequestError error={error} retry={() => void refresh()} />}
      {data && !data.enabled && <p>{t("reliability.historyOff")}</p>}
      {data?.enabled && !data.items.length && (
        <p>{t("reliability.historyEmpty")}</p>
      )}
      {data?.truncated && (
        <p role="status">{t("reliability.historyTruncated")}</p>
      )}
      {!!data?.items.length && (
        <div className="overflow-auto">
          <table className="w-full text-sm">
            <thead>
              <tr>
                {(
                  [
                    "observed",
                    "delivery",
                    "buffer",
                    "stalls",
                    "coverage",
                  ] as const
                ).map((key) => (
                  <th className="text-left p-2" key={key}>
                    {t(`reliability.summary.${key}`)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.items.map((row, index) => (
                <tr key={index}>
                  <td className="p-2">
                    {new Date(row.last_observed).toLocaleString()}
                  </td>
                  <td className="p-2">{humanizeBytes(row.bytes_served)}</td>
                  <td className="p-2">
                    {row.samples > 0
                      ? `${humanizeBytes(row.minimum_buffer_bytes)} – ${humanizeBytes(row.maximum_buffer_bytes)}`
                      : "—"}
                  </td>
                  <td className="p-2">
                    {row.samples > 0
                      ? `${row.peak_recent_read_stalls} · ${Math.round(row.peak_recent_wait_p95_ms)} ms`
                      : "—"}
                  </td>
                  <td className="p-2">
                    {t(
                      row.incomplete
                        ? "reliability.incomplete"
                        : "reliability.observedEnd",
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
