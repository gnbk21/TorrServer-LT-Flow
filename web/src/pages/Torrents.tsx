import { useState, useMemo, lazy, Suspense } from "react";
import { useTranslation } from "react-i18next";
import { useLibrary, refreshLibrary } from "../hooks/queries";
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
const emptyLibrary: Torrent[] = [];
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
  const query = useLibrary();
  const [filter, setFilter] = useState("");
  const [category, setCategory] = useState("");
  const [sort, setSort] = useState("recent");
  const [page, setPage] = useState(1);
  const [add, setAdd] = useState(false);
  const [files, setFiles] = useState<Torrent>();
  const [diagnostic, setDiagnostic] = useState<Torrent>();
  const [edit, setEdit] = useState<Torrent>();
  const [error, setError] = useState<unknown>();
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const list = query.data ?? emptyLibrary;
  const shown = useMemo(
    () =>
      list
        .filter(
          (row) =>
            (!category ||
              (category === "uncategorized"
                ? !row.category
                : `category:${row.category}` === category)) &&
            `${row.title} ${row.name || ""}`
              .toLowerCase()
              .includes(filter.toLowerCase()),
        )
        .sort((a, b) =>
          sort === "title"
            ? a.title.localeCompare(b.title)
            : sort === "size"
              ? (b.torrent_size || 0) - (a.torrent_size || 0)
              : (b.timestamp || 0) - (a.timestamp || 0),
        ),
    [list, category, filter, sort],
  );
  const pages = Math.max(1, Math.ceil(shown.length / 50));
  const currentPage = Math.min(page, pages);
  const visibleRows = shown.slice((currentPage - 1) * 50, currentPage * 50);
  const act = async (fn: () => Promise<unknown>) => {
    setError(undefined);
    setBusy(true);
    try {
      await fn();
      await refreshLibrary();
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  const copy = (value: string) =>
    act(async () => {
      await copyText(value);
      setNotice(t("torrent.copied"));
    });
  const exportData = (format: string) => {
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
              <Button key={format} onClick={() => exportData(format)}>
                {format}
              </Button>
            ))}
          </div>
        </details>
      </div>
      <div className="grid sm:grid-cols-3 gap-3">
        <label className="field">
          {t("Search")}
          <input
            value={filter}
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
            {[...new Set(list.map((row) => row.category).filter(Boolean))].map(
              (value) => (
                <option key={value} value={`category:${value}`}>
                  {value}
                </option>
              ),
            )}
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
      {!query.isPending && !query.error && !shown.length && (
        <div className="panel">
          {t(list.length ? "torrent.noMatches" : "torrent.empty")}
        </div>
      )}
      <div className="grid xl:grid-cols-2 gap-4" aria-busy={busy}>
        {visibleRows.map((row) => (
          <TorrentCard
            busy={busy}
            key={row.hash}
            torrent={row}
            isActiveStream={(row.active_readers ?? 0) > 0}
            onOpenFiles={() => setFiles(row)}
            onOpenDiagnostics={() => setDiagnostic(row)}
            onEdit={() => setEdit({ ...row })}
            onDelete={() => act(() => torrentsApi.remove(row.hash))}
            onDropCache={() => act(() => torrentsApi.drop(row.hash))}
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
              void act(async () => {
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
            <Button type="submit" disabled={busy}>
              {t("Save")}
            </Button>
          </form>
        </Modal>
      )}
    </div>
  );
}
