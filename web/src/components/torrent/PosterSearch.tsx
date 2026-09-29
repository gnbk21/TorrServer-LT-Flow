import { useState } from "react";
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
  const [items, setItems] = useState<string[]>([]);
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const run = async () => {
    setBusy(true);
    setMessage("");
    try {
      const config = await integrationsApi.getTMDB();
      if (!config.APIKey) {
        setMessage(t("torrent.tmdbUnavailable"));
        return;
      }
      const result = await searchPosters(
        config,
        title,
        i18n.resolvedLanguage || "en",
      );
      setItems(result);
      if (!result.length) setMessage(t("search.empty"));
    } catch {
      setMessage(t("torrent.tmdbFailed"));
    } finally {
      setBusy(false);
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
        {items.map((url, index) => (
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
