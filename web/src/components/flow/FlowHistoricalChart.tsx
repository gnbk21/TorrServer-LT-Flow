import { useTranslation } from "react-i18next";
export interface TelemetrySample {
  timestamp: number;
  bufferSeconds: number | null;
  downloadMbps: number;
  demandMbps: number | null;
}
export interface FlowHistoricalChartProps {
  samples: TelemetrySample[];
  height?: number;
}
export function FlowHistoricalChart({
  samples,
  height = 100,
}: FlowHistoricalChartProps) {
  const { t } = useTranslation();
  const bounded = samples
    .filter((sample) => sample.timestamp >= Date.now() - 120_000)
    .slice(-120);
  const first = bounded[0];
  const last = bounded.at(-1);
  if (!first || !last || bounded.length < 2)
    return <p className="panel text-slate-400">{t("flow.collecting")}</p>;
  const span = Math.max(1, last.timestamp - first.timestamp);
  const chart = (
    key: "bufferSeconds" | "downloadMbps" | "demandMbps",
    max: number,
  ) => {
    let gap = true;
    return bounded
      .map((sample) => {
        const value = sample[key];
        if (value === null || !Number.isFinite(value)) {
          gap = true;
          return "";
        }
        const point = `${gap ? "M" : "L"}${8 + ((sample.timestamp - first.timestamp) / span) * 384},${height - 8 - (Math.max(0, value) / max) * (height - 16)}`;
        gap = false;
        return point;
      })
      .join(" ");
  };
  const maxBuffer = Math.max(1, ...bounded.map((s) => s.bufferSeconds ?? 0));
  const maxRate = Math.max(
    1,
    ...bounded.flatMap((s) => [s.downloadMbps, s.demandMbps ?? 0]),
  );
  return (
    <section className="panel space-y-3">
      <h3>{t("flow.history")}</h3>
      <p className="text-sm text-emerald-300">
        {t("flow.buffer")} · 0–{maxBuffer.toFixed(0)} s
      </p>
      <svg
        viewBox={`0 0 400 ${height}`}
        className="w-full"
        role="img"
        aria-label={t("flow.buffer")}
      >
        <path
          d={chart("bufferSeconds", maxBuffer)}
          fill="none"
          stroke="#34d399"
          strokeWidth="2"
        />
      </svg>
      <p className="text-sm">
        <span className="text-blue-300">{t("flow.download")}</span> /{" "}
        <span className="text-amber-300">{t("flow.demand")} (---)</span> · 0–
        {maxRate.toFixed(1)} Mbps
      </p>
      <svg
        viewBox={`0 0 400 ${height}`}
        className="w-full"
        role="img"
        aria-label={t("flow.throughput")}
      >
        <path
          d={chart("downloadMbps", maxRate)}
          fill="none"
          stroke="#60a5fa"
          strokeWidth="2"
        />
        <path
          d={chart("demandMbps", maxRate)}
          fill="none"
          stroke="#fbbf24"
          strokeWidth="2"
          strokeDasharray="4 3"
        />
      </svg>
    </section>
  );
}
