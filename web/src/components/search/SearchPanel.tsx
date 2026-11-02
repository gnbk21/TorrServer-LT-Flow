import { pollingDelay } from "../../api/client";
import {
  useFlow,
  useNetwork,
  useSettings,
  useVisibilityGate,
} from "../../hooks/queries";
import { useEffect, useRef, useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import {
  searchApi,
  deduplicateResults,
  type SearchSource,
  type SearchOutcome,
  type StructuredSearch,
} from "../../api/search";
import { SearchHealth } from "./SearchHealth";
import {
  releaseLabels,
  matchesRelease,
  type ReleaseLabels,
} from "../../lib/release";
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
  const settings = useSettings();
  const network = useNetwork();
  const [outcomes, setOutcomes] = useState<SearchOutcome[]>([]);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const [error, setError] = useState<unknown>();
  const [retryAction, setRetryAction] = useState<(() => void) | undefined>();
  const [sort, setSort] = useState("seeds");
  const [specification, setSpecification] = useState<StructuredSearch>({
    type: "search",
  });
  const [filters, setFilters] = useState<Partial<ReleaseLabels>>({});
  const [added, setAdded] = useState<Torrent>();
  const [file, setFile] = useState<number>();
  const [stage, setStage] = useState("");
  const [ready, setReady] = useState(false);
  const controller = useRef<AbortController>(undefined);
  const searchRequest = useRef<AbortController>(undefined);
  useEffect(
    () => () => {
      controller.current?.abort();
      searchRequest.current?.abort();
    },
    [],
  );
  const resetSearch = () => {
    searchRequest.current?.abort();
    setResults([]);
    setOutcomes([]);
    setLoading(false);
    setSearched(false);
    setError(undefined);
    setRetryAction(undefined);
  };

  const sourceGeneration = JSON.stringify([
    settings.data?.TorznabUrls,
    settings.data?.JacRedUrl,
    settings.data?.JacRedKey,
    enabledSources,
  ]);
  useEffect(() => {
    searchRequest.current?.abort();
    setResults([]);
    setOutcomes([]);
    setLoading(false);
    setSearched(false);
    setError(undefined);
    setRetryAction(undefined);
  }, [sourceGeneration]);

  const sources = (Object.keys(enabledSources) as SearchSource[]).filter(
    (key) => enabledSources[key],
  );
  const metadataVisible = useVisibilityGate(
    ["prepare", added?.hash],
    !!added && stage !== "PLAYABLE",
  );
  const info = useQuery({
    queryKey: ["prepare", added?.hash],
    queryFn: ({ signal }) => torrentsApi.get(added!.hash, signal),
    enabled: metadataVisible,
    refetchInterval: (q) =>
      metadataVisible && q.state.data?.stat !== 5 ? pollingDelay(1500) : false,
  });
  const flow = useFlow(stage === "PREPARING" ? added?.hash : undefined);
  const run = async () => {
    searchRequest.current?.abort();
    const request = new AbortController();
    searchRequest.current = request;
    setLoading(true);
    setError(undefined);
    setRetryAction(() => () => void run());
    setSearched(true);
    setResults([]);
    setOutcomes([]);
    try {
      const rows = await searchApi.search(
        query,
        source,
        undefined,
        request.signal,
        sources,
        (progress) => {
          if (!request.signal.aborted) {
            setResults(progress.items);
            setOutcomes(progress.sources);
          }
        },
        settings.data?.TorznabUrls?.map((_, index) => index),
        specification,
      );
      if (!request.signal.aborted) setResults(rows);
    } catch (e) {
      if (!request.signal.aborted) setError(e);
    } finally {
      if (!request.signal.aborted) setLoading(false);
    }
  };
  const retrySource = async (outcome: SearchOutcome) => {
    searchRequest.current?.abort();
    const request = new AbortController();
    searchRequest.current = request;
    setLoading(true);
    try {
      await searchApi.search(
        query,
        outcome.source,
        outcome.index,
        request.signal,
        [],
        (progress) => {
          if (request.signal.aborted) return;
          setOutcomes((previous) =>
            previous.map((row) =>
              row.source === outcome.source && row.index === outcome.index
                ? (progress.sources[0] ?? row)
                : row,
            ),
          );
          if (progress.items.length)
            setResults((previous) =>
              deduplicateResults([...previous, ...progress.items]),
            );
        },
        undefined,
        specification,
      );
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
  const parsedLabels = useMemo(
    () => new Map(results.map((item) => [item, releaseLabels(item.title)])),
    [results],
  );
  const filtered = results.filter((item) =>
    matchesRelease(parsedLabels.get(item) ?? {}, filters),
  );
  const sorted = [...filtered].sort((a, b) =>
    sort === "title"
      ? a.title.localeCompare(b.title)
      : sort === "peers"
        ? (b.leechers ?? -1) - (a.leechers ?? -1)
        : (b.seeders ?? -1) - (a.seeders ?? -1),
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
            required={
              !specification.imdbid &&
              !specification.tmdbid &&
              !specification.tvdbid
            }
            maxLength={1024}
            value={query}
            onChange={(e) => {
              resetSearch();
              setQuery(e.target.value);
            }}
          />
        </label>
        <div className="actions">
          <label>
            {t("search.source")}{" "}
            <select
              value={source}
              onChange={(e) => {
                resetSearch();
                setSource(e.target.value as SearchSource | "all");
              }}
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
        {!!enabledSources.torznab && (
          <details>
            <summary>{t("search.enriched.structured")}</summary>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mt-3">
              <label className="field">
                {t("search.enriched.type")}
                <select
                  value={specification.type}
                  onChange={(e) => {
                    resetSearch();
                    setSpecification((old) => ({
                      ...old,
                      type: e.target.value as StructuredSearch["type"],
                    }));
                  }}
                >
                  {["search", "movie", "tv"].map((mode) => (
                    <option key={mode} value={mode}>
                      {t(
                        `search.enriched.modes.${mode === "search" ? "generic" : mode}`,
                      )}
                    </option>
                  ))}
                </select>
              </label>
              {(
                ["imdbid", "tmdbid", "tvdbid", "year", "season", "ep"] as const
              ).map((key) => (
                <label className="field" key={key}>
                  {t(`search.enriched.${key}`)}
                  <input
                    maxLength={key === "imdbid" ? 14 : 12}
                    value={specification[key] ?? ""}
                    onChange={(e) => {
                      resetSearch();
                      setSpecification((old) => ({
                        ...old,
                        [key]: e.target.value,
                      }));
                    }}
                  />
                </label>
              ))}
            </div>
          </details>
        )}
      </form>
      <SearchHealth />
      <p className="text-sm text-slate-400">
        {t("search.serviceSeparation", {
          state:
            network.error || !network.data
              ? t("status.unknown")
              : t(
                  `status.${network.data.connectivity === "ONLINE" ? "online" : network.data.connectivity === "DEGRADED" ? "degraded" : "localOnly"}`,
                ),
        })}
      </p>
      {!!error && <RequestError error={error} retry={retryAction} />}
      {!!outcomes.length && (
        <ul
          className="panel text-sm space-y-2"
          aria-label={t("reliability.searchSources")}
        >
          {outcomes.map((row) => (
            <li key={`${row.source}:${row.index}`}>
              {row.source}
              {row.index >= 0 ? ` ${row.index + 1}` : ""}:{" "}
              {t(`reliability.searchStates.${row.state}`)}{" "}
              {row.elapsed_ms !== undefined && (
                <span> · {row.elapsed_ms} ms</span>
              )}
              {row.query_mode && (
                <span>
                  {" "}
                  ·{" "}
                  {t(`search.enriched.modes.${row.query_mode}`, {
                    defaultValue: t("status.unknown"),
                  })}
                </span>
              )}
              {!["pending", "ok", "empty", "cancelled"].includes(row.state) && (
                <Button
                  disabled={loading}
                  onClick={() => void retrySource(row).catch(setError)}
                >
                  {t("WAF.Retry")}
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}
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
      {!!results.length && (
        <section className="panel space-y-3">
          <p>{t("search.enriched.claims")}</p>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            {(
              [
                "resolution",
                "codec",
                "hdr",
                "release",
                "audio",
                "language",
              ] as const
            ).map((key) => (
              <label className="field" key={key}>
                {t(`search.enriched.${key}`)}
                <select
                  value={filters[key] || ""}
                  onChange={(e) =>
                    setFilters((old) => ({ ...old, [key]: e.target.value }))
                  }
                >
                  <option value="">{t("All")}</option>
                  {[
                    ...new Set(
                      results
                        .map((item) => parsedLabels.get(item)?.[key])
                        .filter((value): value is string => !!value),
                    ),
                  ]
                    .sort()
                    .map((value) => (
                      <option key={value} value={value}>
                        {value}
                      </option>
                    ))}
                </select>
              </label>
            ))}
          </div>
          <p role="status">
            {t("search.enriched.hidden", {
              count: results.length - filtered.length,
            })}
          </p>
          <Button onClick={() => setFilters({})}>
            {t("search.enriched.reset")}
          </Button>
        </section>
      )}
      {searched &&
        !loading &&
        !error &&
        !results.length &&
        outcomes.every((row) => row.state === "empty") && (
          <p className="panel">{t("search.empty")}</p>
        )}
      {sorted.map((item, index) => (
        <article
          key={`${item.magnet || item.link}-${index}`}
          className="panel space-y-3"
        >
          <h3 className="font-semibold break-words">{item.title}</h3>
          <div className="flex flex-wrap gap-2">
            {Object.entries(parsedLabels.get(item) ?? {})
              .filter(([, value]) => value)
              .map(([key, value]) => (
                <span key={key} className="flow-chip">
                  {t(`search.enriched.${key}`)}: {value}
                </span>
              ))}
          </div>
          <p className="text-sm text-slate-400">
            {(item.sources || (item.tracker ? [item.tracker] : [])).join(", ")}{" "}
            ·{" "}
            {typeof item.size === "number"
              ? humanizeBytes(item.size)
              : item.size || t("status.unknown")}{" "}
            · {item.seeders ?? t("status.unknown")} {t("search.sort.seeds")} ·{" "}
            {item.leechers ?? t("status.unknown")} {t("search.sort.peers")}
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
