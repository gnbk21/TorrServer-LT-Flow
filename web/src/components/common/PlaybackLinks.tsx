import { useState } from "react";
import { useTranslation } from "react-i18next";
import { copyText } from "../../lib/clipboard";
import { buildIntentUrls, isAppleDevice } from "../../lib/intent";
import { Button } from "./Button";
export function PlaybackLinks({
  url,
  onInternal,
}: {
  url: string;
  onInternal?: () => void;
}) {
  const { t } = useTranslation();
  const [message, setMessage] = useState("");
  const intents = buildIntentUrls(url);
  const linkClass =
    "inline-flex items-center min-h-11 px-3 rounded-lg border border-slate-600 text-sm hover:bg-slate-800";
  return (
    <div className="actions">
      {intents.justPlayerIntent && (
        <a className={linkClass} href={intents.justPlayerIntent}>
          {t("torrent.justPlayer")}
        </a>
      )}
      <a className={linkClass} href={url} target="_blank" rel="noreferrer">
        {t("torrent.openStream")}
      </a>
      <Button
        onClick={() => {
          copyText(url)
            .then(() => setMessage(t("torrent.copied")))
            .catch(() => setMessage(t("pairing.copyFailed")));
        }}
      >
        {t("torrent.copyStream")}
      </Button>
      {onInternal && (
        <Button onClick={onInternal}>{t("torrent.internalPlayer")}</Button>
      )}
      <details>
        <summary>{t("torrent.otherPlayers")}</summary>
        <div className="actions">
          <a className={linkClass} href={intents.vlcIntent || `vlc://${url}`}>
            VLC
          </a>
          {isAppleDevice() && (
            <>
              <a
                className={linkClass}
                href={`iina://weblink?url=${encodeURIComponent(url)}`}
              >
                IINA
              </a>
              <a
                className={linkClass}
                href={`infuse://x-callback-url/play?url=${encodeURIComponent(url)}`}
              >
                Infuse
              </a>
              <a
                className={linkClass}
                href={`senplayer://x-callback-url/play?url=${encodeURIComponent(url)}`}
              >
                SenPlayer
              </a>
            </>
          )}
        </div>
      </details>
      {message && (
        <p role="status" className="text-sm">
          {message}
        </p>
      )}
    </div>
  );
}
