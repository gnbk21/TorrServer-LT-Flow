import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Modal } from "../common/Modal";
import { Button } from "../common/Button";
import { generateQrSvg } from "../../lib/qr";
import { rankServerAddresses, safePairingUrl } from "../../lib/network";
import { copyText } from "../../lib/clipboard";
import { useExternalMediaBase } from "../../hooks/mediaBase";
import { useVisible } from "../../hooks/queries";
import { certificatesApi } from "../../api/certificates";
import { RequestError } from "../common/RequestState";
import { GuidedPhoneSetup } from "./GuidedPhoneSetup";
export interface PhonePairingModalProps {
  isOpen: boolean;
  onClose: () => void;
  serverIps?: string[];
}
export function PhonePairingModal({
  isOpen,
  onClose,
  serverIps,
}: PhonePairingModalProps) {
  const { t } = useTranslation();
  const visible = useVisible();
  const client = useQueryClient();
  const mediaBase = useExternalMediaBase(isOpen && visible);
  const addresses = rankServerAddresses(
    serverIps || [],
    undefined,
    mediaBase.data,
  );
  const [selected, setSelected] = useState("");
  const [manualSelection, setManualSelection] = useState(false);
  const [qr, setQr] = useState("");
  const [message, setMessage] = useState("");
  const initial = addresses.find((a) => !a.isLoopback)?.url || "";
  useEffect(() => {
    if (isOpen) {
      if (!manualSelection) setSelected(initial);
      setMessage("");
    }
  }, [isOpen, initial, manualSelection]);
  let safe = "";
  try {
    safe = safePairingUrl(selected);
  } catch {
    /* Invalid manual input must never become a QR. */
  }
  const https = safe.startsWith("https:");
  const certificate = useQuery({
    queryKey: ["pairing-certificates"],
    queryFn: ({ signal }) => certificatesApi.status(signal),
    enabled: isOpen && visible && https,
    staleTime: 30_000,
    retry: false,
  });
  useEffect(() => {
    const cancel = () =>
      void client.cancelQueries({
        queryKey: ["pairing-certificates"],
        exact: true,
      });
    if (!isOpen || !visible || !https) cancel();
    return cancel;
  }, [client, isOpen, visible, https]);
  useEffect(() => {
    let disposed = false;
    setQr("");
    if (safe && isOpen && visible)
      generateQrSvg(safe)
        .then((svg) => {
          if (!disposed) setQr(svg);
        })
        .catch(() => {
          if (!disposed) setQr("");
        });
    return () => {
      disposed = true;
    };
  }, [safe, isOpen, visible]);
  const copy = (value: string) =>
    copyText(value)
      .then(() => setMessage(t("torrent.copied")))
      .catch(() => setMessage(t("pairing.copyFailed")));
  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t("pairing.title")}
      maxWidth="lg"
    >
      <div className="space-y-4">
        <p className="text-sm text-slate-300">{t("pairing.hint")}</p>
        <div className="field">
          <label htmlFor="pairing-address">{t("pairing.address")}</label>
          <select
            id="pairing-address"
            value={addresses.some((a) => a.url === selected) ? selected : ""}
            onChange={(e) => {
              setManualSelection(true);
              setSelected(e.target.value);
            }}
          >
            <option value="">{t("pairing.manual")}</option>
            {addresses.map((a) => (
              <option key={a.url} value={a.url} disabled={a.isLoopback}>
                {a.url} — {t(a.label)}
              </option>
            ))}
          </select>
        </div>
        <label className="field">
          {t("pairing.manual")}
          <input
            value={selected}
            onChange={(e) => {
              setManualSelection(true);
              setSelected(e.target.value);
            }}
            placeholder="http://192.168.1.2:8090"
          />
        </label>
        {!safe && <p role="status">{t("pairing.invalidAddress")}</p>}
        <GuidedPhoneSetup base={safe} visible={isOpen && visible} />
        {safe && (
          <>
            <div className="panel space-y-2 text-sm">
              <p>{t(https ? "pairing.httpsTrust" : "pairing.httpPlaintext")}</p>
              <p>{t("pairing.reachability")}</p>
              {https && certificate.isPending && (
                <p role="status">{t("pairing.checkingCertificate")}</p>
              )}
              {https && certificate.error && (
                <RequestError
                  error={certificate.error}
                  retry={() => certificate.refetch()}
                />
              )}
              {https && certificate.data && (
                <p>
                  {t(
                    certificate.data.enabled
                      ? certificate.data.cert.trusted
                        ? "pairing.serverTrustOnly"
                        : "pairing.serverUntrusted"
                      : "pairing.tlsNotReported",
                  )}
                </p>
              )}
            </div>
            <div className="mx-auto w-48 bg-white rounded-xl p-3">
              {qr ? (
                <div dangerouslySetInnerHTML={{ __html: qr }} />
              ) : (
                <p className="text-slate-900">{t("pairing.qrUnavailable")}</p>
              )}
            </div>
            <p className="break-all font-mono text-sm">{safe}</p>
            <div className="actions">
              <Button onClick={() => copy(safe)}>
                {t("pairing.copyLampa")}
              </Button>
              <Button onClick={() => copy(`${safe}/playlistall/all.m3u`)}>
                {t("pairing.copyPlaylist")}
              </Button>
            </div>
          </>
        )}
        <p className="text-sm text-slate-400">{t("pairing.authHint")}</p>
        {message && <p role="status">{message}</p>}
      </div>
    </Modal>
  );
}
