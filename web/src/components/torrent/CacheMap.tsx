import { useCache } from "../../hooks/queries";
import { useTranslation } from "react-i18next";
import { RequestError, Loading } from "../common/RequestState";
import { humanizeBytes } from "../../lib/format";
export function CacheMap({ hash }: { hash: string }) {
  const { t } = useTranslation();
  const query = useCache(hash, true);
  if (query.isPending) return <Loading />;
  if (query.error)
    return <RequestError error={query.error} retry={() => query.refetch()} />;
  const state = query.data;
  const count = state?.PiecesCount || 0;
  const bins = Array.from({ length: Math.min(512, count) }, () => ({
    complete: 0,
    partial: 0,
    total: 0,
  }));
  for (const item of Object.values(state?.Pieces || {})) {
    const index = Math.min(
      bins.length - 1,
      Math.floor((item.Id / Math.max(1, count)) * bins.length),
    );
    const bin = bins[index];
    if (bin) {
      bin.total++;
      if (item.Completed) bin.complete++;
      else bin.partial++;
    }
  }
  return (
    <section className="space-y-3">
      <p>
        {t("Cache")}: {humanizeBytes(state?.Filled)} /{" "}
        {humanizeBytes(state?.Capacity)} · {count} {t("PiecesCount")}
      </p>
      <svg
        role="img"
        aria-label={t("torrent.cacheMap")}
        viewBox="0 0 512 32"
        className="w-full bg-slate-950 rounded"
      >
        {bins.map((bin, index) => (
          <rect
            key={index}
            x={(index * 512) / bins.length}
            y="0"
            width={512 / bins.length}
            height="32"
            fill={
              bin.partial ? "#fbbf24" : bin.complete ? "#34d399" : "#334155"
            }
          />
        ))}
      </svg>
      <p className="text-xs">{t("torrent.cacheLegend")}</p>
      <details>
        <summary>{t("torrent.cacheReaders")}</summary>
        <pre className="overflow-auto text-xs">
          {JSON.stringify(state?.Readers || [], null, 2)}
        </pre>
      </details>
    </section>
  );
}
