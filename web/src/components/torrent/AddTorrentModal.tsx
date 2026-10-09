import { useState, lazy, Suspense } from "react";
import { useTranslation } from "react-i18next";
import { Modal } from "../common/Modal";
import { Button } from "../common/Button";
import { RequestError } from "../common/RequestState";
import { torrentsApi, type AddTorrentRequest } from "../../api/torrents";
const PosterSearch = lazy(() =>
  import("./PosterSearch").then((m) => ({ default: m.PosterSearch })),
);
export function parseImport(text: string): AddTorrentRequest[] {
  const trimmed = text.trim();
  if (!trimmed) return [];
  if (trimmed.startsWith("[") || trimmed.startsWith("{")) {
    const parsed: unknown = JSON.parse(trimmed);
    const items = Array.isArray(parsed) ? parsed : [parsed];
    return items.map((item: unknown) => {
      if (!item || typeof item !== "object")
        throw new Error("torrent.invalidImport");
      const row = item as Record<string, unknown>;
      const str = (key: string) =>
        typeof row[key] === "string" ? (row[key] as string) : "";
      const link =
        str("link") ||
        (str("torrs_hash") ? `torrs://${str("torrs_hash")}` : str("hash"));
      if (!link) throw new Error("torrent.invalidImport");
      return {
        link,
        title: str("title") || str("name"),
        poster: str("poster"),
        category: str("category"),
        data: str("data"),
        save_to_db: true,
      };
    });
  }
  return trimmed
    .split(/\r?\n/)
    .map((link) => ({ link: link.trim(), save_to_db: true }))
    .filter((row) => row.link);
}
export interface AddTorrentModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}
export function AddTorrentModal({
  isOpen,
  onClose,
  onSuccess,
}: AddTorrentModalProps) {
  const { t } = useTranslation();
  const [text, setText] = useState("");
  const [title, setTitle] = useState("");
  const [poster, setPoster] = useState("");
  const [category, setCategory] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [save, setSave] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const [result, setResult] = useState("");
  const chooseFiles = async (chosen: File[]) => {
    try {
      const imports = chosen.filter((file) => /\.(txt|json)$/i.test(file.name));
      const entries = (
        await Promise.all(
          imports.map(async (file) => parseImport(await file.text())),
        )
      ).flat();
      if (entries.length) setText(JSON.stringify(entries, null, 2));
      setFiles(chosen.filter((file) => /\.torrent$/i.test(file.name)));
    } catch (e) {
      setError(e);
    }
  };
  const submit = async () => {
    setBusy(true);
    setError(undefined);
    let count = 0;
    try {
      const entries = parseImport(text);
      if (!entries.length && !files.length)
        throw new Error("torrent.sourceRequired");
      const failedEntries: AddTorrentRequest[] = [];
      const failedFiles: File[] = [];
      for (const row of entries) {
        try {
          await torrentsApi.add({
            ...row,
            title: row.title || title,
            poster: row.poster || poster,
            category: row.category || category,
            save_to_db: save,
          });
          count++;
        } catch (e) {
          failedEntries.push(row);
          setError(e);
        }
      }
      for (const file of files) {
        try {
          const response = await torrentsApi.upload(
            file,
            save,
            title,
            poster,
            category,
          );
          if (Array.isArray(response) && !response.length)
            throw new Error("torrent.invalidUpload");
          count += Array.isArray(response) ? response.length : 1;
        } catch (e) {
          failedFiles.push(file);
          setError(e);
        }
      }
      // Keep only failures so retrying a partial batch never adds successes again.
      setText(
        failedEntries.length ? JSON.stringify(failedEntries, null, 2) : "",
      );
      setFiles(failedFiles);
      setResult(t("torrent.addedCount", { count }));
      onSuccess();
    } catch (e) {
      setError(e);
      setResult(t("torrent.addedCount", { count }));
      onSuccess();
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal
      isOpen={isOpen}
      onClose={() => {
        if (!busy) onClose();
      }}
      title={t("AddTorrent")}
      maxWidth="xl"
    >
      <form
        className="space-y-4"
        onSubmit={(e) => {
          e.preventDefault();
          void submit();
        }}
      >
        <label className="field">
          {t("ImportLibraryHint")}
          <textarea
            rows={4}
            value={text}
            onChange={(e) => setText(e.target.value)}
          />
        </label>
        <label
          className="field border border-dashed border-slate-500 p-4 rounded-xl"
          onDragOver={(e) => e.preventDefault()}
          onDrop={(e) => {
            e.preventDefault();
            if (!busy) void chooseFiles([...e.dataTransfer.files]);
          }}
        >
          {t("torrent.dropFiles")}
          <input
            type="file"
            disabled={busy}
            multiple
            accept=".torrent,.txt,.json"
            onChange={(e) => void chooseFiles([...(e.target.files || [])])}
          />
          {files.map((f) => (
            <span key={f.name}>{f.name}</span>
          ))}
        </label>
        <label className="field">
          {t("torrent.title")}
          <input value={title} onChange={(e) => setTitle(e.target.value)} />
        </label>
        <label className="field">
          {t("Category")}
          <input
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          />
        </label>
        <label className="field">
          {t("torrent.poster")}
          <input
            type="url"
            value={poster}
            onChange={(e) => setPoster(e.target.value)}
          />
        </label>
        <label>
          <input
            type="checkbox"
            checked={save}
            onChange={(e) => setSave(e.target.checked)}
          />{" "}
          {t("torrent.saveLibrary")}
        </label>
        <Suspense fallback={null}>
          <PosterSearch title={title} onSelect={setPoster} />
        </Suspense>
        {!!error && <RequestError error={error} />}{" "}
        {result && <p role="status">{result}</p>}
        <div className="actions">
          <Button onClick={onClose} disabled={busy}>
            {t("Close")}
          </Button>
          <Button type="submit" variant="primary" isLoading={busy}>
            {t("Add")}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
