import { useFlow } from "../../hooks/queries";
import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { searchApi, type SearchSource } from "../../api/search";
import { torrentsApi } from "../../api/torrents";
import type { Torrent } from "../../types/torrent";
import type { SearchResultItem } from "../../types/search";
import { Button } from "../common/Button";
import { RequestError } from "../common/RequestState";
import { PlaybackLinks } from "../common/PlaybackLinks";
import { humanizeBytes } from "../../lib/format";
export interface SearchPanelProps {
  onTorrentAdded?: () => void;
  enabledSources?: { rutor?: boolean; torznab?: boolean; jacred?: boolean };
}
export function SearchPanel({
  onTorrentAdded,
  enabledSources = {},
}: SearchPanelProps) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [source, setSource] = useState<SearchSource | "all">("all");
  const [results, setResults] = useState<SearchResultItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const [error, setError] = useState<unknown>();
  const [retryAction, setRetryAction] = useState<(() => void) | undefined>();
  const [sort, setSort] = useState("seeds");
  const [added, setAdded] = useState<Torrent>();
  const [file, setFile] = useState<number>();
  const [stage, setStage] = useState("");
  const [ready, setReady] = useState(false);
  const controller = useRef<AbortController>(undefined);
  useEffect(() => () => controller.current?.abort(), []);
  const sources = (Object.keys(enabledSources) as SearchSource[]).filter(
    (key) => enabledSources[key],
  );
  const info = useQuery({
    queryKey: ["prepare", added?.hash],
    queryFn: ({ signal }) => torrentsApi.get(added!.hash, signal),
    enabled: !!added,
    refetchInterval: (q) =>
      !document.hidden && q.state.data?.stat !== 5 && stage !== "PLAYABLE"
        ? 1500
        : false,
  });
  const flow = useFlow(stage === "PREPARING" ? added?.hash : undefined);
  const run = async () => {
    controller.current?.abort();
    const request = new AbortController();
    controller.current = request;
    setLoading(true);
    setError(undefined);
    setRetryAction(() => () => void run());
    setSearched(true);
    try {
      const rows = await searchApi.search(
        query,
        source,
        undefined,
        request.signal,
        sources,
      );
      if (!request.signal.aborted) setResults(rows);
    } catch (e) {
      if (!request.signal.aborted) setError(e);
    } finally {
      if (!request.signal.aborted) setLoading(false);
    }
  };
  const add = async (item: SearchResultItem) => {
    controller.current?.abort();
    const request = new AbortController();
    controller.current = request;
    setError(undefined);
    setRetryAction(() => () => void add(item));
    setAdded(undefined);
    setStage("ADDING");
    setReady(false);
    setFile(undefined);
    try {
      const row = await torrentsApi.add(
        {
          link: item.magnet || item.link || "",
          title: item.title,
          category: item.category,
          save_to_db: true,
        },
        request.signal,
      );
      if (request.signal.aborted) return;
      setAdded(row);
      setStage("METADATA");
      onTorrentAdded?.();
    } catch (e) {
      if (!request.signal.aborted) {
        setError(e);
        setStage("");
      }
    }
  };
  const prepare = async () => {
    if (!added || file === undefined) return;
    controller.current?.abort();
    const request = new AbortController();
    controller.current = request;
    setStage("PREPARING");
    setError(undefined);
    setRetryAction(() => () => void prepare());
    setReady(false);
    try {
      const result = await torrentsApi.prepare(
        added.hash,
        file,
        request.signal,
      );
      if (
        result.stat !== 3 ||
        ((result.preload_size || 0) > 0 &&
          (result.preloaded_bytes || 0) < (result.preload_size || 0))
      )
        throw new Error("search.prepareIncomplete");
      if (!request.signal.aborted) {
        setReady(true);
        setStage("PLAYABLE");
      }
    } catch (e) {
      if (!request.signal.aborted) {
        setError(e);
        setStage("METADATA");
      }
    }
  };
  const sorted = [...results].sort((a, b) =>
    sort === "title"
      ? a.title.localeCompare(b.title)
      : sort === "peers"
        ? b.leechers - a.leechers
        : b.seeders - a.seeders,
  );
  return (
    <div className="space-y-4">
      <form
        className="panel space-y-3"
        onSubmit={(e) => {
          e.preventDefault();
          void run();
        }}
      >
        <label className="field">
          {t("Search")}
          <input
            required
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </label>
        <div className="actions">
          <label>
            {t("search.source")}{" "}
            <select
              value={source}
              onChange={(e) =>
                setSource(e.target.value as SearchSource | "all")
              }
            >
              <option value="all">{t("All")}</option>
              {sources.map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
          </label>
          <Button
            type="submit"
            isLoading={loading}
            disabled={
              !sources.length || stage === "ADDING" || stage === "PREPARING"
            }
          >
            {t("Search")}
          </Button>
        </div>
        {!sources.length && <p>{t("search.noSources")}</p>}
      </form>
      {!!error && <RequestError error={error} retry={retryAction} />}
      {added && (
        <section className="panel space-y-3">
          <h2>{added.title}</h2>
          <p role="status">{t(`search.stage.${stage}`)}</p>
          {info.error && (
            <RequestError error={info.error} retry={() => info.refetch()} />
          )}
          {info.data?.file_stats?.length ? (
            <>
              <label className="field">
                {t("torrent.files")}
                <select
                  value={file ?? ""}
                  disabled={stage === "PREPARING"}
                  onChange={(e) => {
                    setFile(Number(e.target.value));
                    setReady(false);
                    setError(undefined);
                    setRetryAction(undefined);
                    setStage("METADATA");
                  }}
                >
                  <option value="" disabled>
                    {t("search.chooseFile")}
                  </option>
                  {info.data.file_stats.map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.path}
                    </option>
                  ))}
                </select>
              </label>
              <Button
                disabled={file === undefined || stage === "PREPARING"}
                onClick={() => void prepare()}
              >
                {t("search.prepare")}
              </Button>
              {stage === "PREPARING" && (
                <p>
                  {humanizeBytes(info.data.preloaded_bytes ?? 0)} /{" "}
                  {humanizeBytes(info.data.preload_size)}
                </p>
              )}
              {stage === "PREPARING" && flow.data?.startup && (
                <p role="status">
                  {t("search.startup")}: {flow.data.startup.state} ·{" "}
                  {humanizeBytes(flow.data.startup.startup_target_bytes)}
                </p>
              )}
              {ready && file !== undefined && (
                <PlaybackLinks
                  url={torrentsApi.getStreamUrl(added.hash, file)}
                />
              )}
            </>
          ) : (
            <p>{t("torrent.waitMetadata")}</p>
          )}
        </section>
      )}
      <label className="field">
        {t("torrent.sort")}
        <select value={sort} onChange={(e) => setSort(e.target.value)}>
          {["seeds", "peers", "title"].map((value) => (
            <option key={value} value={value}>
              {t(`search.sort.${value}`)}
            </option>
          ))}
        </select>
      </label>
      {searched && !loading && !error && !results.length && (
        <p className="panel">{t("search.empty")}</p>
      )}
      {sorted.map((item, index) => (
        <article
          key={`${item.magnet || item.link}-${index}`}
          className="panel space-y-3"
        >
          <h3 className="font-semibold break-words">{item.title}</h3>
          <p className="text-sm text-slate-400">
            {item.tracker} ·{" "}
            {typeof item.size === "number"
              ? humanizeBytes(item.size)
              : item.size}{" "}
            · {item.seeders} {t("search.sort.seeds")} · {item.leechers}{" "}
            {t("search.sort.peers")}
          </p>
          <Button
            disabled={stage === "ADDING" || stage === "PREPARING"}
            onClick={() => void add(item)}
          >
            {t("search.addPrepare")}
          </Button>
        </article>
      ))}
    </div>
  );
}
