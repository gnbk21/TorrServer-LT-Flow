import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { api } from "../../api/client";
import type { RuntimeStatus } from "../../types/runtime";
import { humanizeBytes } from "../../lib/format";
import { Button } from "../common/Button";
import { RequestError } from "../common/RequestState";

const storageSchema = z.object({
  schema_version: z.literal(1),
  preview_only: z.literal(true),
  ram_cache_bytes: z.number().nonnegative(),
  categories: z.record(
    z.string(),
    z.object({
      bytes: z.number().nonnegative(),
      files: z.number().int().nonnegative(),
      truncated: z.boolean(),
      protected: z.literal(true),
    }),
  ),
});
export function StoragePanel({ runtime }: { runtime?: RuntimeStatus }) {
  const { t } = useTranslation();
  const [storage, setStorage] = useState<z.infer<typeof storageSchema>>();
  const [busy, setBusy] = useState(false),
    [raw, setRaw] = useState(false);
  const [error, setError] = useState<unknown>();
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  const scan = async () => {
    request.current?.abort();
    const controller = new AbortController();
    request.current = controller;
    setBusy(true);
    setError(undefined);
    try {
      setStorage(
        storageSchema.parse(
          await api.get("/flow/storage", controller.signal, 5000),
        ),
      );
    } catch (e) {
      if (!controller.signal.aborted) setError(e);
    } finally {
      setBusy(false);
    }
  };
  const incidents = runtime?.incidents;
  return (
    <section className="space-y-3 border-t border-slate-700 pt-4">
      <h3 className="font-semibold">{t("settings.storageView.title")}</h3>
      <p className="text-sm text-slate-400">
        {t("settings.storageView.preview")}
      </p>
      {runtime?.resources && (
        <p className="text-sm">
          {t("settings.storageView.pressure", {
            state: t(
              `settings.storageView.pressureStates.${runtime.resources.pressure_state ?? "unknown"}`,
            ),
            samples: runtime.resources.recovery_samples ?? 0,
          })}
          {runtime.resources.protected_overcommit &&
            ` · ${t("settings.storageView.overcommit")}`}
        </p>
      )}
      <Button disabled={busy} onClick={() => void scan()}>
        {t("settings.storageView.scan")}
      </Button>
      {storage && (
        <dl className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
          <div>
            <dt>{t("settings.storageView.ram")}</dt>
            <dd>{humanizeBytes(storage.ram_cache_bytes)}</dd>
          </div>
          {Object.entries(storage.categories).map(([key, row]) => (
            <div key={key}>
              <dt>{t(`settings.storageView.${key}`)}</dt>
              <dd>
                {humanizeBytes(row.bytes)} · {row.files}
                {row.truncated
                  ? ` · ${t("settings.storageView.incomplete")}`
                  : ""}
              </dd>
            </div>
          ))}
        </dl>
      )}
      {!!error && <RequestError error={error} />}
      <details>
        <summary>{t("settings.storageView.developer")}</summary>
        <p className="text-sm text-slate-400">
          {t("settings.storageView.developerHint")}
        </p>
        <code className="block break-all text-sm">
          python build/workspace_storage.py --output .tools/storage-preview.json
        </code>
      </details>
      <h3 className="font-semibold">{t("settings.storageView.incidents")}</h3>
      <p>
        {incidents
          ? t(
              `settings.storageView.incidentStates.${incidents.code || "DISABLED"}`,
              { defaultValue: incidents.code },
            )
          : t("status.unknown")}
      </p>
      <label className="flex items-start gap-3 text-sm">
        <input
          type="checkbox"
          checked={raw}
          onChange={(e) => setRaw(e.target.checked)}
        />
        {t("settings.storageView.raw")}
      </label>
      {raw && (
        <div className="space-y-2 text-sm">
          <p>{t("settings.fieldHints.IncidentFlightRecorder")}</p>
          {incidents?.files.length ? (
            incidents.files.map((file) => (
              <p key={file.name}>
                <a
                  className="underline min-h-11 inline-flex items-center break-all"
                  href={`/flow/incidents/${encodeURIComponent(file.name)}`}
                  download
                >
                  {file.created_at} · {humanizeBytes(file.bytes)}
                </a>
              </p>
            ))
          ) : (
            <p>{t("settings.storageView.noCaptures")}</p>
          )}
        </div>
      )}
    </section>
  );
}
