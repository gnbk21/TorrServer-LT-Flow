import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { integrationsApi } from "../../api/integrations";
import { Button } from "../common/Button";
import { searchPosters } from "../../api/tmdb";
export function PosterSearch({
  title,
  onSelect,
}: {
  title: string;
  onSelect: (url: string) => void;
}) {
  const { t, i18n } = useTranslation();
  const [result, setResult] = useState<{ title: string; items: string[] }>();
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const request = useRef<AbortController>(undefined);
  useEffect(() => {
    const cancel = () => {
      if (document.hidden) request.current?.abort();
    };
    document.addEventListener("visibilitychange", cancel);
    return () => {
      request.current?.abort();
      document.removeEventListener("visibilitychange", cancel);
    };
  }, [title]);
  const run = async () => {
    request.current?.abort();
    const operation = new AbortController();
    request.current = operation;
    setBusy(true);
    setMessage("");
    try {
      const config = await integrationsApi.getTMDB(operation.signal);
      if (operation.signal.aborted) return;
      if (!config.APIKey) {
        setMessage(t("torrent.tmdbUnavailable"));
        return;
      }
      const result = await searchPosters(
        config,
        title,
        i18n.resolvedLanguage || "en",
        operation.signal,
      );
      if (operation.signal.aborted) return;
      setResult({ title, items: result });
      if (!result.length) setMessage(t("search.empty"));
    } catch {
      if (!operation.signal.aborted) setMessage(t("torrent.tmdbFailed"));
    } finally {
      if (request.current === operation) setBusy(false);
    }
  };
  return (
    <div className="space-y-2">
      <Button
        disabled={!title.trim()}
        isLoading={busy}
        onClick={() => void run()}
      >
        {t("torrent.findPoster")}
      </Button>
      {message && <p role="status">{message}</p>}
      <div className="flex flex-wrap gap-2">
        {(result?.title === title ? result.items : []).map((url, index) => (
          <button
            type="button"
            key={url}
            className="w-20 rounded overflow-hidden"
            aria-label={t("torrent.selectPoster", { index: index + 1 })}
            onClick={() => onSelect(url)}
          >
            <img
              loading="lazy"
              decoding="async"
              src={url}
              alt=""
              className="aspect-[2/3] object-cover"
            />
          </button>
        ))}
      </div>
    </div>
  );
}
