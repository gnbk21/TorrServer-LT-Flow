import { useId, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { sourcesApi } from "../../api/sources";
import { Button } from "../common/Button";
import { RequestError } from "../common/RequestState";

export function WebSeeds({ hash }: { hash: string }) {
  const { t } = useTranslation();
  const formId = useId();
  const [open, setOpen] = useState(false);
  const [url, setUrl] = useState("");
  const [local, setLocal] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const query = useQuery({
    queryKey: ["sources", hash],
    queryFn: ({ signal }) => sourcesApi.get(hash, signal),
    enabled: open,
  });
  const update = async (value: Parameters<typeof sourcesApi.update>[1]) => {
    setBusy(true);
    setError(undefined);
    try {
      await sourcesApi.update(hash, value);
      setUrl("");
      await query.refetch();
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <details className="panel" onToggle={(e) => setOpen(e.currentTarget.open)}>
      <summary className="min-h-11 cursor-pointer">
        {t("sources.title")}
      </summary>
      <div className="space-y-3">
        <p className="text-sm text-slate-400">{t("sources.help")}</p>
        {!!query.data?.private && <p>{t("flow.privatePolicy")}</p>}
        {query.data?.sources.map((s) => (
          <div key={s.id} className="flex flex-wrap items-center gap-2">
            <span className="font-mono break-all">{s.origin}</span>
            <span>
              {t(s.disabled ? "sources.disabled" : "sources.configured")}
            </span>
            {!s.disabled && (
              <Button
                disabled={busy}
                onClick={() => update({ action: "remove", id: s.id })}
              >
                {t("sources.remove")}
              </Button>
            )}
          </div>
        ))}
        {!query.data?.private && (
          <form
            className="space-y-3"
            onSubmit={(e) => {
              e.preventDefault();
              void update({ action: "add", url, allow_local: local });
            }}
          >
            <label htmlFor={formId}>{t("sources.url")}</label>
            <input
              id={formId}
              className="input w-full"
              type="url"
              required
              maxLength={8192}
              value={url}
              autoComplete="off"
              spellCheck={false}
              onChange={(e) => setUrl(e.target.value)}
            />
            <label className="flex min-h-11 items-center gap-2">
              <input
                type="checkbox"
                checked={local}
                onChange={(e) => setLocal(e.target.checked)}
              />
              {t("sources.local")}
            </label>
            <p className="text-sm text-slate-400">{t("sources.localHelp")}</p>
            <Button type="submit" disabled={busy || !query.data}>
              {t("sources.add")}
            </Button>
          </form>
        )}
        {!!(query.error || error) && (
          <RequestError
            error={error || query.error}
            retry={() => query.refetch()}
          />
        )}
      </div>
    </details>
  );
}
