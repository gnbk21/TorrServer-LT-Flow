import { useState, useEffect, useRef, lazy, Suspense } from "react";
import { useTranslation } from "react-i18next";
import { useLibraryPage, useActive, refreshLibrary } from "../hooks/queries";
import { usePreference, isString, isPage } from "../hooks/preferences";
import { torrentsApi } from "../api/torrents";
import type { Torrent } from "../types/torrent";
import { TorrentCard } from "../components/torrent/TorrentCard";
import { AddTorrentModal } from "../components/torrent/AddTorrentModal";
import { TorrentFilesDialog } from "../components/torrent/TorrentFilesDialog";
import { Button } from "../components/common/Button";
import { Modal } from "../components/common/Modal";
import { Loading, RequestError } from "../components/common/RequestState";
import { copyText } from "../lib/clipboard";
import { useFlow } from "../hooks/queries";
const PosterSearch = lazy(() =>
  import("../components/torrent/PosterSearch").then((m) => ({
    default: m.PosterSearch,
  })),
);
const Diagnostics = lazy(() =>
  import("../components/flow/FlowDiagnosticsDrawer").then((m) => ({
    default: m.FlowDiagnosticsDrawer,
  })),
);
function DiagnosticsView({
  torrent,
  onClose,
}: {
  torrent: Torrent;
  onClose: () => void;
}) {
  const status = useFlow(torrent.hash);
  return (
    <Suspense fallback={<Loading />}>
      {status.error && <RequestError error={status.error} />}
      <Diagnostics
        isOpen
        onClose={onClose}
        status={status.data}
        torrentTitle={torrent.title}
      />
    </Suspense>
  );
}
export default function Torrents() {
  const { t } = useTranslation();
  const [filter, setFilter] = usePreference("library.filter", "", isString);
  const [category, setCategory] = usePreference(
    "library.category",
    "",
    isString,
  );
  const [sort, setSort] = usePreference(
    "library.sort",
    "recent",
    (v): v is string =>
      typeof v === "string" && ["recent", "title", "size"].includes(v),
  );
  const [page, setPage] = usePreference("library.page", 1, isPage);
  const [mode, setMode] = usePreference(
    "library.mode",
    "cards",
    (v): v is string => v === "cards" || v === "list",
  );
  const [search, setSearch] = useState(filter);
  useEffect(() => {
    const timer = setTimeout(() => setSearch(filter), 250);
    return () => clearTimeout(timer);
  }, [filter]);
  const query = useLibraryPage({ q: search, category, sort, page, limit: 50 });
  const activity = useActive();
  const restoredScroll = useRef(false);
  useEffect(() => {
    if (!query.data || restoredScroll.current) return;
    restoredScroll.current = true;
    let offset = 0;
    try {
      offset = Number(sessionStorage.getItem("flow.library.scroll")) || 0;
    } catch {
      /* optional local preference */
    }
    const frame = requestAnimationFrame(() =>
      window.scrollTo(
        0,
        Math.max(0, Math.min(offset, document.documentElement.scrollHeight)),
      ),
    );
    return () => cancelAnimationFrame(frame);
  }, [query.data]);
  useEffect(() => {
    const save = () => {
      try {
        sessionStorage.setItem("flow.library.scroll", String(window.scrollY));
      } catch {
        /* optional local preference */
      }
    };
    window.addEventListener("scroll", save, { passive: true });
    return () => window.removeEventListener("scroll", save);
  }, []);
  const [add, setAdd] = useState(false);
  const [files, setFiles] = useState<Torrent>();
  const [diagnostic, setDiagnostic] = useState<Torrent>();
  const [edit, setEdit] = useState<Torrent>();
  const [error, setError] = useState<unknown>();
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState<Set<string>>(new Set());
  const pages = Math.max(1, Math.ceil((query.data?.total || 0) / 50));
  const currentPage = query.data?.page || page;
  const visibleRows = (query.data?.items || []).map((row) => {
    const live = activity.data?.items.find(
      (item) => item.torrent.hash === row.hash,
    );
    return live
      ? {
          ...row,
          ...live.torrent,
          flow_playback: (live.status.sessions || [])
            .filter((s) => s.active_readers > 0)
            .map((s) => ({
              file_index: s.file_index,
              buffer_seconds:
                s.playback_consumption_rate > 0 ? s.buffer_ahead_seconds : null,
              sustainability:
                s.playback_consumption_rate > 0 ? s.sustainability_ratio : null,
              buffer_warning: s.buffer_warning,
            })),
        }
      : row;
  });
  const act = async (
    key: string,
    fn: () => Promise<unknown>,
    refresh = true,
  ) => {
    setError(undefined);
    setBusy((previous) => new Set(previous).add(key));
    try {
      await fn();
      if (refresh) await refreshLibrary();
    } catch (e) {
      setError(e);
    } finally {
      setBusy((previous) => {
        const next = new Set(previous);
        next.delete(key);
        return next;
      });
    }
  };
  const copy = (value: string) =>
    act(
      "clipboard",
      async () => {
        await copyText(value);
        setNotice(t("torrent.copied"));
      },
      false,
    );
  const exportData = async (format: string) => {
    const list = await torrentsApi.list();
    const body =
      format === "json"
        ? JSON.stringify(list, null, 2)
        : list
            .map((row) =>
              format === "magnets"
                ? `magnet:?xt=urn:btih:${row.hash}&dn=${encodeURIComponent(row.title)}`
                : `torrs://${row.torrs_hash || row.hash}`,
            )
            .join("\n");
    const url = URL.createObjectURL(
      new Blob([body], {
        type: format === "json" ? "application/json" : "text/plain",
      }),
    );
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `flow-library.${format === "json" ? "json" : "txt"}`;
    anchor.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  };
  return (
    <div className="space-y-5">
      <div className="actions">
        <h1 className="text-2xl font-semibold mr-auto">{t("nav.torrents")}</h1>
        <Button variant="primary" onClick={() => setAdd(true)}>
          {t("Add")}
        </Button>
        <details>
          <summary>{t("ExportLibrary")}</summary>
          <div className="actions">
            {["json", "magnets", "torrs"].map((format) => (
              <Button
                disabled={busy.has("export")}
                key={format}
                onClick={() =>
                  void act("export", () => exportData(format), false)
                }
              >
                {format}
              </Button>
            ))}
          </div>
        </details>
      </div>
      <label className="field max-w-xs">
        {t("torrent.view")}
        <select value={mode} onChange={(e) => setMode(e.target.value)}>
          <option value="cards">{t("torrent.cards")}</option>
          <option value="list">{t("torrent.list")}</option>
        </select>
      </label>
      <div className="grid sm:grid-cols-3 gap-3">
        <label className="field">
          {t("Search")}
          <input
            value={filter}
            maxLength={256}
            onChange={(e) => {
              setFilter(e.target.value);
              setPage(1);
            }}
          />
        </label>
        <label className="field">
          {t("Category")}
          <select
            value={category}
            onChange={(e) => {
              setCategory(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{t("All")}</option>
            <option value="uncategorized">{t("Uncategorized")}</option>
            {(query.data?.categories || []).map((value) => (
              <option key={value} value={`category:${value}`}>
                {value}
              </option>
            ))}
          </select>
        </label>
        <label className="field">
          {t("torrent.sort")}
          <select
            value={sort}
            onChange={(e) => {
              setSort(e.target.value);
              setPage(1);
            }}
          >
            {["recent", "title", "size"].map((value) => (
              <option key={value} value={value}>
                {t(`torrent.${value}`)}
              </option>
            ))}
          </select>
        </label>
      </div>
      {query.isPending && <Loading />}
      {query.error && (
        <RequestError
          error={query.error}
          retry={() => query.refetch()}
          stale={!!query.data}
        />
      )}{" "}
      {!!error && <RequestError error={error} />}{" "}
      {notice && <p role="status">{notice}</p>}
      {!query.isPending && !query.error && !visibleRows.length && (
        <div className="panel">
          {t(query.data?.library_total ? "torrent.noMatches" : "torrent.empty")}
        </div>
      )}
      <div
        className={mode === "cards" ? "grid xl:grid-cols-2 gap-4" : "space-y-3"}
      >
        {visibleRows.map((row) => (
          <TorrentCard
            busy={busy.has(row.hash)}
            compact={mode === "list"}
            key={row.hash}
            torrent={row}
            isActiveStream={(row.active_readers ?? 0) > 0}
            onOpenFiles={() => setFiles(row)}
            onOpenDiagnostics={() => setDiagnostic(row)}
            onEdit={() =>
              void act(
                row.hash,
                async () => setEdit(await torrentsApi.get(row.hash)),
                false,
              )
            }
            onDelete={() =>
              void act(row.hash, () => torrentsApi.remove(row.hash))
            }
            onDropCache={() =>
              void act(row.hash, () => torrentsApi.drop(row.hash))
            }
            onCopyMagnet={() => copy(`magnet:?xt=urn:btih:${row.hash}`)}
            onCopyPlaylist={() => copy(torrentsApi.getPlaylistUrl(row.hash))}
          />
        ))}
      </div>
      {pages > 1 && (
        <nav className="actions" aria-label={t("torrent.pagination")}>
          <Button
            disabled={currentPage === 1}
            onClick={() => setPage(currentPage - 1)}
          >
            {t("torrent.previousPage")}
          </Button>
          <span role="status">
            {t("torrent.pageCount", { page: currentPage, pages })}
          </span>
          <Button
            disabled={currentPage === pages}
            onClick={() => setPage(currentPage + 1)}
          >
            {t("torrent.nextPage")}
          </Button>
        </nav>
      )}
      {add && (
        <AddTorrentModal
          isOpen
          onClose={() => setAdd(false)}
          onSuccess={() => void refreshLibrary()}
        />
      )}{" "}
      {files && (
        <TorrentFilesDialog
          isOpen
          torrent={files}
          onClose={() => setFiles(undefined)}
        />
      )}{" "}
      {diagnostic && (
        <DiagnosticsView
          torrent={diagnostic}
          onClose={() => setDiagnostic(undefined)}
        />
      )}
      {edit && (
        <Modal
          isOpen
          onClose={() => setEdit(undefined)}
          title={t("torrent.edit")}
        >
          <form
            className="space-y-4"
            onSubmit={(e) => {
              e.preventDefault();
              void act(edit.hash, async () => {
                await torrentsApi.set({
                  hash: edit.hash,
                  title: edit.title,
                  poster: edit.poster,
                  category: edit.category,
                  data: edit.data,
                });
                setEdit(undefined);
              });
            }}
          >
            {(["title", "poster", "category"] as const).map((key) => (
              <label className="field" key={key}>
                {t(`torrent.${key}`)}
                <input
                  value={edit[key] || ""}
                  onChange={(e) => setEdit({ ...edit, [key]: e.target.value })}
                />
              </label>
            ))}
            <Suspense fallback={null}>
              <PosterSearch
                title={edit.title}
                onSelect={(poster) => setEdit({ ...edit, poster })}
              />
            </Suspense>
            <Button type="submit" disabled={busy.has(edit.hash)}>
              {t("Save")}
            </Button>
          </form>
        </Modal>
      )}
    </div>
  );
}
