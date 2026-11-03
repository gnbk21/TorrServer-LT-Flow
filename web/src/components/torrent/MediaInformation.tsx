import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { api } from "../../api/client";
import { RequestError } from "../common/RequestState";
import { browserAdvice, type BrowserAdvice } from "../../lib/compatibility";
import { Button } from "../common/Button";
import { useVisibilityGate } from "../../hooks/queries";

const schema = z.object({
  schema_version: z.literal(1),
  known: z.boolean(),
  source: z.literal("existing_bounded_probe"),
  media: z.object({
    container: z.string().max(128).optional(),
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
          profile: z.string().max(128).optional(),
          width: z.number().nonnegative().optional(),
          height: z.number().nonnegative().optional(),
          bit_depth: z.number().nonnegative().optional(),
          channels: z.number().nonnegative().optional(),
          transfer: z.string().max(128).optional(),
          hdr: z.string().max(128).optional(),
          pixel_format: z.string().max(128).optional(),
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
  const [advice, setAdvice] = useState<BrowserAdvice>();
  const enabled = useVisibilityGate(["cached-media", hash, index], open);
  const query = useQuery({
    queryKey: ["cached-media", hash, index],
    queryFn: async ({ signal }) =>
      schema.parse(
        await api.get(
          `/flow/media/${encodeURIComponent(hash)}/${index}`,
          signal,
        ),
      ),
    enabled,
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
            {t("torrent.compatibility.container")}:{" "}
            {media?.container || t("status.unknown")}
          </p>
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
                {track.profile ? ` · ${track.profile}` : ""}
                {track.width && track.height
                  ? ` · ${track.width} × ${track.height}`
                  : ""}
                {track.bit_depth ? ` · ${track.bit_depth} bit` : ""}
                {track.transfer ? ` · ${track.transfer}` : ""}
                {track.type === "video" && (
                  <span> · HDR: {track.hdr || t("status.unknown")}</span>
                )}
                {track.pixel_format ? ` · ${track.pixel_format}` : ""}
                {track.channels ? ` · ${track.channels} ch` : ""}
              </li>
            ))}
          </ul>
          {media?.truncated && <p>{t("torrent.media.truncated")}</p>}
        </>
      )}
      <p className="text-slate-400">{t("torrent.media.direct")}</p>
      <section className="space-y-2">
        <h4>{t("torrent.compatibility.title")}</h4>
        <p>{t("torrent.compatibility.external")}</p>
        <ul className="list-disc pl-5">
          <li>{t("torrent.compatibility.direct")}</li>
          <li>{t("torrent.compatibility.remux")}</li>
          <li>{t("torrent.compatibility.audio")}</li>
          <li>{t("torrent.compatibility.video")}</li>
        </ul>
        <Button
          onClick={() =>
            setAdvice(
              browserAdvice(
                media?.container,
                media?.tracks?.find((track) => track.type === "video")?.codec,
                (mime) => document.createElement("video").canPlayType(mime),
              ),
            )
          }
        >
          {t("torrent.compatibility.check")}
        </Button>
        {advice && <p role="status">{t(`torrent.compatibility.${advice}`)}</p>}
        <p>{t("torrent.compatibility.advisory")}</p>
      </section>
    </details>
  );
}
