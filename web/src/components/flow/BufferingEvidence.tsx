import { useTranslation } from "react-i18next";
import type { FlowSession } from "../../types/flow";
import { humanizeBytes } from "../../lib/format";

export function BufferingEvidence({
  session,
  fresh,
}: {
  session?: FlowSession;
  fresh: boolean;
}) {
  const { t } = useTranslation();
  if (
    !session ||
    session.active_readers < 1 ||
    (!session.supply && !session.blocker)
  )
    return null;
  const supply = session.supply;
  const blocker = session.blocker;
  return (
    <section className="space-y-2 text-sm">
      <h4 className="font-semibold">{t("flow.buffering.title")}</h4>
      {!fresh ? (
        <p>{t("flow.sparseUnknown")}</p>
      ) : (
        <>
          {supply?.known ? (
            <>
              <p>{t(`flow.buffering.states.${supply.state}`)}</p>
              <p>
                {t("flow.buffering.rates", {
                  delivery: ((supply.delivery_rate * 8) / 1e6).toFixed(1),
                  demand: ((supply.demand_rate * 8) / 1e6).toFixed(1),
                  deficit: ((supply.deficit_rate * 8) / 1e6).toFixed(1),
                })}
              </p>
            </>
          ) : (
            <p>{t("flow.buffering.unknown")}</p>
          )}
          {blocker?.known && blocker.reason !== "READY" && (
            <>
              <p>
                {t("flow.buffering.blocker", {
                  piece: blocker.piece,
                  size: humanizeBytes(blocker.piece_length),
                  finished: blocker.finished,
                  blocks: blocker.blocks,
                  requested: blocker.requested,
                  unrequested: blocker.unrequested,
                  writing: blocker.writing,
                })}
              </p>
              <p>
                {t("flow.buffering.blockerAge", {
                  age:
                    blocker.oldest_request_age_ms < 0
                      ? "—"
                      : blocker.oldest_request_age_ms,
                  suppliers: blocker.suppliers ?? "—",
                })}
              </p>
              {blocker.reason === "VERIFYING" && (
                <p>{t("flow.buffering.verifying")}</p>
              )}
            </>
          )}
        </>
      )}
      <p className="text-slate-400">{t("flow.buffering.hint")}</p>
    </section>
  );
}
