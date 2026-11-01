import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { api } from "../../api/client";
import { RequestError } from "../common/RequestState";

const schema = z.object({
  schema_version: z.literal(1),
  known: z.boolean(),
  source: z.literal("existing_bounded_probe"),
  media: z.object({
    bitrate: z.string().max(32),
    duration_seconds: z.number().nonnegative(),
    tracks: z
      .array(
        z.object({
          index: z.number().int(),
          type: z.enum(["video", "audio", "subtitle"]),
          codec: z.string().max(128),
          language: z.string().max(128).optional(),
          title: z.string().max(128).optional(),
        }),
      )
      .max(32)
      .nullable(),
    truncated: z.boolean(),
    probed_at: z.string(),
  }),
});
export function MediaInformation({
  hash,
  index,
}: {
  hash: string;
  index: number;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const query = useQuery({
    queryKey: ["cached-media", hash, index],
    queryFn: async ({ signal }) =>
      schema.parse(
        await api.get(
          `/flow/media/${encodeURIComponent(hash)}/${index}`,
          signal,
        ),
      ),
    enabled: open,
    retry: false,
    staleTime: 10000,
  });
  const media = query.data?.media;
  const bitrate = Number(media?.bitrate);
  return (
    <details
      onToggle={(e) => setOpen(e.currentTarget.open)}
      className="text-sm space-y-2"
    >
      <summary>{t("torrent.media.title")}</summary>
      <p className="text-slate-400">{t("torrent.media.hint")}</p>
      {query.error && (
        <RequestError error={query.error} retry={() => query.refetch()} />
      )}
      {!query.data?.known ? (
        <p>{t("torrent.media.unknown")}</p>
      ) : (
        <>
          <p>
            {t("torrent.media.bitrate")}:{" "}
            {Number.isFinite(bitrate) && bitrate > 0
              ? `${(bitrate / 1e6).toFixed(2)} Mbps · ${(bitrate / 8e6).toFixed(2)} MB/s`
              : t("status.unknown")}
          </p>
          <ul className="space-y-1">
            {media?.tracks?.map((track) => (
              <li key={track.index} className="break-words">
                {t(`torrent.media.${track.type}`)} {track.index}:{" "}
                {track.codec || t("status.unknown")} · {track.language || "—"}{" "}
                {track.title || ""}
              </li>
            ))}
          </ul>
          {media?.truncated && <p>{t("torrent.media.truncated")}</p>}
        </>
      )}
      <p className="text-slate-400">{t("torrent.media.direct")}</p>
    </details>
  );
}
