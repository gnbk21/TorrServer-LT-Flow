import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "../common/Button";
import { RequestError } from "../common/RequestState";
import { measureLAN, type TransferResult } from "../../lib/lan-test";

export function LANTest() {
  const { t } = useTranslation();
  const controller = useRef<AbortController | null>(null);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<TransferResult>();
  const [error, setError] = useState<unknown>();
  useEffect(() => () => controller.current?.abort(), []);
  const run = async () => {
    const current = new AbortController();
    controller.current = current;
    const timeout = window.setTimeout(() => current.abort(), 20000);
    setBusy(true);
    setError(undefined);
    setResult(undefined);
    try {
      setResult(await measureLAN(current.signal));
    } catch (e) {
      if (!current.signal.aborted) setError(e);
    } finally {
      window.clearTimeout(timeout);
      setBusy(false);
    }
  };
  return (
    <section className="panel space-y-3">
      <h2>{t("settings.lanTitle")}</h2>
      <p className="text-sm text-slate-400">{t("settings.lanHint")}</p>
      <div className="actions">
        <Button disabled={busy} onClick={() => void run()}>
          {t("settings.lanRun")}
        </Button>
        {busy && (
          <Button onClick={() => controller.current?.abort()}>
            {t("Cancel")}
          </Button>
        )}
      </div>
      {!!error && <RequestError error={error} />}
      {result && (
        <p role="status">
          {t("settings.lanResult", {
            rate: result.mbps.toFixed(1),
            ttfb: result.ttfbMs.toFixed(0),
            p95: result.p95ReadGapMs.toFixed(0),
            max: result.maxReadGapMs.toFixed(0),
          })}
        </p>
      )}
    </section>
  );
}
