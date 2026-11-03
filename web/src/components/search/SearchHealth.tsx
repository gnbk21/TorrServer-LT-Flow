import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { api, ApiError } from "../../api/client";
import { useVisibilityGate } from "../../hooks/queries";
import { RequestError } from "../common/RequestState";
const capabilitySchema = z.object({
  available: z.string().max(16),
  supported_params: z.string().max(512),
});
const schema = z.object({
  schema_version: z.literal(1),
  items: z
    .array(
      z.object({
        source: z.string().max(32),
        index: z.number().int(),
        name: z.string().max(128),
        capabilities: z
          .object({
            search: capabilitySchema,
            movie: capabilitySchema,
            tv: capabilitySchema,
          })
          .nullable()
          .optional(),
        history: z.object({
          last_success: z.string().optional(),
          cooldown_until: z.string().optional(),
          attempts: z
            .array(
              z.object({
                state: z.string().max(32),
                elapsed_ms: z.number().nonnegative(),
                query_mode: z.string().max(32).optional(),
                at: z.string(),
              }),
            )
            .max(20),
        }),
      }),
    )
    .max(128),
});
export function SearchHealth() {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const enabled = useVisibilityGate(["search-health"], open);
  const result = useQuery({
    queryKey: ["search-health"],
    queryFn: async ({ signal }) =>
      schema.parse(await api.get("/flow/search-status", signal)),
    enabled,
    retry: false,
    staleTime: 5000,
  });
  return (
    <details className="panel" onToggle={(e) => setOpen(e.currentTarget.open)}>
      <summary>{t("search.enriched.health")}</summary>
      <p className="text-sm mt-3">{t("search.enriched.aggregate")}</p>
      <p className="text-sm mt-2">{t("search.enriched.setup")}</p>
      <a
        className="inline-flex min-h-11 items-center underline"
        href="#/settings"
      >
        {t("nav.settings")}
      </a>
      {result.error &&
        (!(result.error instanceof ApiError) ||
          result.error.status !== 404) && (
          <RequestError error={result.error} retry={() => result.refetch()} />
        )}
      {result.error instanceof ApiError && result.error.status === 404 && (
        <p>{t("status.unknown")}</p>
      )}
      {result.data?.items.map((row) => {
        const last = row.history.attempts.at(-1);
        return (
          <section
            className="panel mt-3 space-y-2"
            key={`${row.source}:${row.index}`}
          >
            <h3>
              {row.name} {row.index >= 0 ? row.index + 1 : ""}
            </h3>
            {row.capabilities && (
              <dl className="text-sm">
                {Object.entries(row.capabilities).map(([mode, cap]) => (
                  <div key={mode}>
                    <dt>
                      {t(
                        `search.enriched.modes.${mode === "search" ? "generic" : mode}`,
                      )}
                    </dt>
                    <dd>
                      {cap.available}:{" "}
                      {cap.supported_params || t("status.unknown")}
                    </dd>
                  </div>
                ))}
              </dl>
            )}
            <p>
              {last
                ? t(`reliability.searchStates.${last.state}`, {
                    defaultValue: t("status.unknown"),
                  })
                : t("status.unknown")}
              {last ? ` · ${last.elapsed_ms} ms` : ""}
            </p>
            {row.history.last_success && (
              <p>
                {t("search.enriched.lastSuccess")}:{" "}
                {new Date(row.history.last_success).toLocaleString()}
              </p>
            )}
            {row.history.cooldown_until &&
              Date.parse(row.history.cooldown_until) > Date.now() && (
                <p>
                  {t("search.enriched.cooldown")}:{" "}
                  {new Date(row.history.cooldown_until).toLocaleTimeString()}
                </p>
              )}
            <ul className="text-sm">
              {row.history.attempts
                .slice()
                .reverse()
                .map((item, i) => (
                  <li key={i}>
                    {new Date(item.at).toLocaleTimeString()} ·{" "}
                    {t(`reliability.searchStates.${item.state}`, {
                      defaultValue: t("status.unknown"),
                    })}{" "}
                    · {item.elapsed_ms} ms ·{" "}
                    {t(
                      `search.enriched.modes.${item.query_mode || "generic"}`,
                      { defaultValue: t("status.unknown") },
                    )}
                  </li>
                ))}
            </ul>
          </section>
        );
      })}
    </details>
  );
}
