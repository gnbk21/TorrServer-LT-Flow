import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { api } from "../../api/client";
import { queryClient, useRuntime } from "../../hooks/queries";
import { humanizeBytes } from "../../lib/format";
import { Button } from "../common/Button";
import { Modal } from "../common/Modal";
import { RequestError } from "../common/RequestState";

const previewSchema = z.object({
  digest: z.string().regex(/^[a-f0-9]{64}$/),
  library_count: z.number().int().nonnegative().max(2000),
  settings_fields: z.array(z.string()),
  credentials_excluded: z.literal(true),
  mode: z.literal("merge"),
  restart_required: z.literal(true),
});
type Preview = z.infer<typeof previewSchema>;

function download(value: unknown, filename: string) {
  const url = URL.createObjectURL(
    new Blob([JSON.stringify(value, null, 2)], { type: "application/json" }),
  );
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export function MaintenancePanel({ dirty }: { dirty: boolean }) {
  const { t } = useTranslation();
  const runtime = useRuntime();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const [preview, setPreview] = useState<Preview>();
  const [backup, setBackup] = useState<unknown>();
  const [notice, setNotice] = useState("");
  const input = useRef<HTMLInputElement>(null);
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  const run = async (action: (signal: AbortSignal) => Promise<void>) => {
    request.current?.abort();
    const controller = new AbortController();
    request.current = controller;
    setBusy(true);
    setError(undefined);
    setNotice("");
    try {
      await action(controller.signal);
    } catch (e) {
      if (!controller.signal.aborted) setError(e);
    } finally {
      if (!controller.signal.aborted) setBusy(false);
    }
  };
  const select = (file?: File) => {
    setPreview(undefined);
    setBackup(undefined);
    if (!file) return;
    void run(async (signal) => {
      if (file.size > 16 * 1024 * 1024)
        throw new Error("settings.maintenance.tooLarge");
      const value: unknown = JSON.parse(await file.text());
      if (signal.aborted) return;
      const result = previewSchema.parse(
        await api.post("/flow/backup/preview", value, signal),
      );
      setBackup(value);
      setPreview(result);
    });
  };
  const memory = runtime.data?.memory;
  const allocation = runtime.data?.cache_allocation;
  return (
    <section className="panel space-y-4">
      <h2 className="font-semibold">{t("settings.maintenance.title")}</h2>
      <p className="text-sm text-slate-300">
        {t("settings.maintenance.exclusions")}
      </p>
      <div className="actions">
        <Button
          disabled={busy}
          onClick={() =>
            void run(async (signal) =>
              download(
                await api.get("/flow/support", signal),
                "TorrServer-Flow-support.json",
              ),
            )
          }
        >
          {t("settings.maintenance.support")}
        </Button>
        <Button
          disabled={busy}
          onClick={() =>
            void run(async (signal) =>
              download(
                await api.get("/flow/backup", signal),
                "TorrServer-Flow-backup.json",
              ),
            )
          }
        >
          {t("settings.maintenance.export")}
        </Button>
        <Button disabled={busy || dirty} onClick={() => input.current?.click()}>
          {t("settings.maintenance.import")}
        </Button>
        <input
          ref={input}
          type="file"
          accept="application/json,.json"
          className="hidden"
          aria-label={t("settings.maintenance.import")}
          onChange={(event) => {
            select(event.target.files?.[0]);
            event.target.value = "";
          }}
        />
      </div>
      {dirty && <p>{t("settings.maintenance.dirty")}</p>}
      {notice && <p role="status">{notice}</p>}
      {!!error && <RequestError error={error} />}
      {(memory || allocation) && (
        <dl className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <dt>{t("settings.maintenance.rss")}</dt>
            <dd>
              {memory?.rss_available ? humanizeBytes(memory.rss_bytes) : "—"}
            </dd>
          </div>
          <div>
            <dt>{t("settings.maintenance.heap")}</dt>
            <dd>{humanizeBytes(memory?.go_heap_bytes)}</dd>
          </div>
          <div>
            <dt>{t("settings.maintenance.activeCache")}</dt>
            <dd>{humanizeBytes(allocation?.active_resident_bytes)}</dd>
          </div>
          <div>
            <dt>{t("settings.maintenance.warmCache")}</dt>
            <dd>{humanizeBytes(allocation?.warm_resident_bytes)}</dd>
          </div>
        </dl>
      )}
      <p className="text-xs text-slate-400">
        {t("settings.maintenance.memoryHint")}
      </p>
      <Modal
        isOpen={!!preview}
        onClose={() => {
          if (!busy) {
            setPreview(undefined);
            setBackup(undefined);
          }
        }}
        title={t("settings.maintenance.preview")}
      >
        <p>
          {t("settings.maintenance.summary", {
            torrents: preview?.library_count,
            fields: preview?.settings_fields.length,
          })}
        </p>
        <p className="mt-3">{t("settings.maintenance.confirmHint")}</p>
        <div className="actions mt-4">
          <Button
            disabled={busy}
            onClick={() => {
              setPreview(undefined);
              setBackup(undefined);
            }}
          >
            {t("Cancel")}
          </Button>
          <Button
            disabled={busy || dirty}
            variant="danger"
            onClick={() =>
              void run(async (signal) => {
                if (!preview || !backup) return;
                const result = z
                  .object({
                    restored: z.literal(true),
                    recovery_backup: z.string(),
                  })
                  .parse(
                    await api.post(
                      "/flow/backup/apply",
                      { backup, digest: preview.digest },
                      signal,
                    ),
                  );
                setPreview(undefined);
                setBackup(undefined);
                await queryClient.invalidateQueries();
                setNotice(
                  t("settings.maintenance.restored", {
                    name: result.recovery_backup,
                  }),
                );
              })
            }
          >
            {t("settings.maintenance.restore")}
          </Button>
        </div>
      </Modal>
    </section>
  );
}
