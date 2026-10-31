import { useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { pairingApi } from "../../api/pairing";
import { ApiError, pollingDelay } from "../../api/client";
import { generateQrSvg } from "../../lib/qr";
import { copyText } from "../../lib/clipboard";
import { Button } from "../common/Button";
import { RequestError } from "../common/RequestState";

export function GuidedPhoneSetup({
  base,
  visible,
}: {
  base: string;
  visible: boolean;
}) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const [created, setCreated] =
    useState<Awaited<ReturnType<typeof pairingApi.create>>>();
  const [busy, setBusy] = useState(false),
    [expired, setExpired] = useState(false);
  const [error, setError] = useState<unknown>();
  const [qr, setQr] = useState("");
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  useEffect(() => {
    if (!visible) request.current?.abort();
  }, [visible]);
  const proofUrl =
    created && base ? `${base}/flow/pairing/proof/${created.proof.token}` : "";
  useEffect(() => {
    let disposed = false;
    setQr("");
    if (proofUrl && visible && !expired)
      void generateQrSvg(proofUrl)
        .then((svg) => {
          if (!disposed) setQr(svg);
        })
        .catch(() => {});
    return () => {
      disposed = true;
    };
  }, [proofUrl, visible, expired]);
  useEffect(() => {
    if (!created) return;
    const timer = setTimeout(() => setExpired(true), 300000);
    return () => clearTimeout(timer);
  }, [created]);
  const status = useQuery({
    queryKey: ["pairing-proof", created?.proof.id],
    queryFn: ({ signal }) => pairingApi.status(created!.proof.id, signal),
    enabled: visible && !!created && !expired,
    retry: false,
    refetchInterval: () => pollingDelay(2000),
    refetchIntervalInBackground: false,
    gcTime: 0,
  });
  useEffect(() => {
    if (status.error instanceof ApiError && status.error.status === 410)
      setExpired(true);
  }, [status.error]);
  useEffect(() => {
    const cancel = () =>
      void client.cancelQueries({
        queryKey: ["pairing-proof", created?.proof.id],
        exact: true,
      });
    if (!visible) cancel();
    return cancel;
  }, [client, visible, created?.proof.id]);
  const create = async () => {
    const controller = new AbortController();
    request.current?.abort();
    request.current = controller;
    setBusy(true);
    setError(undefined);
    try {
      const result = await pairingApi.create(controller.signal);
      if (!controller.signal.aborted) {
        setCreated(result);
        setExpired(false);
      }
    } catch (e) {
      if (!controller.signal.aborted) setError(e);
    } finally {
      setBusy(false);
    }
  };
  const current = status.data ?? created?.proof;
  return (
    <section className="panel space-y-3" aria-labelledby="guided-setup-title">
      <h3 id="guided-setup-title" className="font-semibold">
        {t("pairing.guide.title")}
      </h3>
      <ol className="list-decimal pl-5 space-y-3 text-sm">
        <li>
          {t("pairing.guide.listener")}{" "}
          {created &&
            t(
              created.startup.listeners_ready
                ? "pairing.guide.listenerReady"
                : "pairing.guide.listenerPending",
            )}
          {created && (
            <dl className="my-2 break-all">
              <dt>{t("pairing.guide.bindAddresses")}</dt>
              <dd>
                {created.listener_addresses
                  .map((a) => a || t("pairing.guide.allInterfaces"))
                  .join(", ")}
              </dd>
              <dt>{t("pairing.guide.lanAddresses")}</dt>
              <dd>
                {created.advertised_addresses?.join(", ") ||
                  t("status.unknown")}
              </dd>
              <dt>{t("pairing.guide.selectedAddress")}</dt>
              <dd>{base}</dd>
            </dl>
          )}
        </li>
        <li>
          {t("pairing.guide.proofHint")}
          <Button disabled={busy || !base} onClick={() => void create()}>
            {t("pairing.guide.create")}
          </Button>
          {expired ? (
            <p role="status">{t("pairing.guide.expired")}</p>
          ) : (
            created && (
              <>
                {qr && (
                  <div
                    className="w-48 max-w-full rounded-lg bg-white p-3 my-2"
                    dangerouslySetInnerHTML={{ __html: qr }}
                  />
                )}
                <Button onClick={() => void copyText(proofUrl).catch(setError)}>
                  {t("pairing.guide.copy")}
                </Button>
                <p role="status">
                  {t(
                    current?.different_client
                      ? "pairing.guide.received"
                      : current?.reached_at
                        ? "pairing.guide.sameClient"
                        : "pairing.guide.waiting",
                  )}
                </p>
              </>
            )
          )}
        </li>
        <li>
          {t("pairing.guide.access")}
          {created?.auth_required && <p>{t("pairing.authHint")}</p>}
          {current?.evidence.map((e, i) => (
            <p key={`${e.at}:${i}`} className="break-words">
              {t(`pairing.guide.${e.code}`)} {e.origin || ""}
            </p>
          ))}
        </li>
        <li>{t("pairing.guide.player")}</li>
      </ol>
      <p className="text-sm text-slate-400">
        {t("pairing.guide.android")}{" "}
        <a
          className="underline"
          href="https://developer.android.com/privacy-and-security/local-network-permission"
          target="_blank"
          rel="noreferrer"
        >
          Android
        </a>
      </p>
      <p className="text-sm text-slate-400">{t("pairing.guide.limits")}</p>
      {!!error && <RequestError error={error} />}
      {!expired && status.error && <RequestError error={status.error} />}
    </section>
  );
}
