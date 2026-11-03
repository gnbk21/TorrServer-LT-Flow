import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { PieceCoverageSnapshot } from "../../api/coverage";
import { humanizeBytes } from "../../lib/format";
export function PieceCoverage({
  coverage,
  now,
}: {
  coverage: PieceCoverageSnapshot;
  now: number;
}) {
  const { t } = useTranslation();
  const [selected, setSelected] = useState<number>();
  const age = now - coverage.sampled_at_ms;
  const supplyAge = now - coverage.availability_sampled_at_ms;
  const fresh = coverage.known && age >= 0 && age < 5000;
  const supplyFresh = fresh && supplyAge >= 0 && supplyAge < 5000;
  const state = (p: PieceCoverageSnapshot["pieces"][number]) =>
    !fresh
      ? "unknown"
      : p.verified
        ? "verified"
        : p.resident
          ? "partial"
          : supplyFresh && p.queue_known && p.requested > 0
            ? "requested"
            : "missing";
  const piece = coverage.pieces.find((p) => p.piece === selected);
  return (
    <section className="panel space-y-3">
      <h3>{t("flow.coverage.title")}</h3>
      <p className="text-sm">{t("flow.coverage.hint")}</p>
      {(!fresh || !supplyFresh) && (
        <p role="status">{t("flow.sparseUnknown")}</p>
      )}
      <p className="text-sm">
        {t("flow.coverage.cacheAge", {
          seconds:
            coverage.sampled_at_ms > 0 && age >= 0
              ? (age / 1000).toFixed(1)
              : t("status.unknown"),
        })}
        {" · "}
        {t("flow.coverage.supplyAge", {
          seconds:
            coverage.availability_sampled_at_ms > 0 && supplyAge >= 0
              ? (supplyAge / 1000).toFixed(1)
              : t("status.unknown"),
        })}
      </p>
      {!coverage.pieces.length && <p>{t("flow.coverage.empty")}</p>}
      <div className="flex flex-wrap gap-2">
        {["verified", "partial", "requested", "missing", "unknown"].map(
          (key) => (
            <span className="flow-chip" key={key}>
              {t(`flow.coverage.${key}`)}
            </span>
          ),
        )}
      </div>
      <p className="text-sm">
        H: {t("flow.coverage.head")} · I: {t("flow.coverage.index")} · !:{" "}
        {t("flow.coverage.blocker")}
      </p>
      <div className="flow-coverage-grid">
        {coverage.pieces.map((p) => (
          <button
            type="button"
            key={p.piece}
            data-state={state(p)}
            data-blocker={p.blocker}
            aria-pressed={selected === p.piece}
            aria-label={`${p.piece}: ${t(`flow.coverage.${state(p)}`)}${p.required_head ? ` · ${t("flow.coverage.head")}` : ""}${p.required_index ? ` · ${t("flow.coverage.index")}` : ""}`}
            onClick={() => setSelected(p.piece)}
          >
            {p.piece}
            {p.blocker ? " !" : ""}
            {p.required_head ? " H" : ""}
            {p.required_index ? " I" : ""}
          </button>
        ))}
      </div>
      {piece && (
        <>
          {(!supplyFresh || piece.suppliers === undefined) && (
            <p>{t("flow.coverage.uncovered")}</p>
          )}
          <dl className="grid grid-cols-2 gap-2 text-sm">
            <dt>{t("flow.coverage.piece")}</dt>
            <dd>
              {piece.piece} · {t(`flow.coverage.${state(piece)}`)}
            </dd>
            <dt>{t("flow.coverage.suppliers")}</dt>
            <dd>
              {supplyFresh && piece.suppliers !== undefined
                ? piece.suppliers
                : "—"}
            </dd>
            <dt>{t("flow.coverage.requests")}</dt>
            <dd>{supplyFresh && piece.queue_known ? piece.requested : "—"}</dd>
            <dt>{t("flow.coverage.receiving")}</dt>
            <dd>
              {supplyFresh && piece.queue_known
                ? humanizeBytes(piece.receiving_bytes)
                : "—"}
            </dd>
          </dl>
        </>
      )}
      {coverage.frontiers.map((f, i) => (
        <p className="text-sm" key={i}>
          {t("flow.fileIndex", { index: f.file_index })} ·{" "}
          {t("flow.coverage.frontier")}: {f.piece} ·{" "}
          {t("flow.coverage.verified")}:{" "}
          {fresh && now - f.sampled_at_ms >= 0 && now - f.sampled_at_ms < 5000
            ? humanizeBytes(f.verified_bytes)
            : "—"}{" "}
          · {t("flow.upgrade.readableBuffer")}:{" "}
          {fresh && now - f.sampled_at_ms >= 0 && now - f.sampled_at_ms < 5000
            ? humanizeBytes(f.readable_bytes)
            : "—"}
        </p>
      ))}
      {coverage.truncated && <p>{t("flow.coverage.truncated")}</p>}
      <p className="text-sm text-slate-400">{t("flow.coverage.supplyHint")}</p>
    </section>
  );
}
