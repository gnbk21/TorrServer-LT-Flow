import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useSettings } from "../../hooks/queries";
import { flowApi } from "../../api/flow";
import { useTranslation } from "react-i18next";
import { copyText } from "../../lib/clipboard";
import { buildIntentUrls, isAppleDevice } from "../../lib/intent";
import { Button } from "./Button";
export function PlaybackLinks({
  url: originalURL,
  onInternal,
}: {
  url: string;
  onInternal?: (url: string) => void;
}) {
  const { t } = useTranslation();
  const [message, setMessage] = useState("");
  const [secure, setSecure] = useState(false);
  const settings = useSettings();
  const required = !!settings.data?.Flow?.RequirePlaybackToken;
  const input = new URL(originalURL, window.location.origin);
  const hash = input.searchParams.get("link") ?? "";
  const index = Number(input.searchParams.get("index"));
  const eligible =
    /^[a-f0-9]{40}$/i.test(hash) && Number.isInteger(index) && index > 0;
  const capability = useQuery({
    queryKey: ["playback-link", originalURL],
    queryFn: ({ signal }) => flowApi.playbackLink(hash, index, signal),
    enabled: eligible && (required || secure),
    refetchOnWindowFocus: false,
    retry: false,
    staleTime: 0,
  });
  const url =
    capability.data && (secure || required)
      ? new URL(capability.data.path, input.origin).href
      : originalURL;
  const blocked = required && !capability.data;
  const intents = buildIntentUrls(url);
  const linkClass =
    "inline-flex items-center min-h-11 px-3 rounded-lg border border-slate-600 text-sm hover:bg-slate-800";
  return (
    <div className="actions">
      {!blocked && intents.justPlayerIntent && (
        <a className={linkClass} href={intents.justPlayerIntent}>
          {t("torrent.justPlayer")}
        </a>
      )}
      {!blocked && (
        <a className={linkClass} href={url} target="_blank" rel="noreferrer">
          {t("torrent.openStream")}
        </a>
      )}
      <Button
        disabled={blocked}
        onClick={() => {
          copyText(url)
            .then(() => setMessage(t("torrent.copied")))
            .catch(() => setMessage(t("pairing.copyFailed")));
        }}
      >
        {t("torrent.copyStream")}
      </Button>
      {onInternal && (
        <Button disabled={blocked} onClick={() => onInternal(url)}>
          {t("torrent.internalPlayer")}
        </Button>
      )}
      {eligible && (
        <Button
          isLoading={capability.isFetching}
          onClick={() => {
            setSecure(true);
            if (secure || required) void capability.refetch();
          }}
        >
          {t("torrent.expiringLink")}
        </Button>
      )}
      {capability.data && (secure || required) && (
        <p role="status" className="text-xs">
          {t("torrent.linkExpires", {
            time: new Date(capability.data.expires_at).toLocaleString(),
          })}
        </p>
      )}
      {capability.error && <p role="alert">{t("torrent.linkFailed")}</p>}
      {!blocked && (
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
      )}
      {message && (
        <p role="status" className="text-sm">
          {message}
        </p>
      )}
    </div>
  );
}
