import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Modal } from "../common/Modal";
import { Button } from "../common/Button";
import { generateQrSvg } from "../../lib/qr";
import { rankServerAddresses, safePairingUrl } from "../../lib/network";
import { copyText } from "../../lib/clipboard";
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
  const addresses = rankServerAddresses(serverIps || []);
  const [selected, setSelected] = useState("");
  const [qr, setQr] = useState("");
  const [message, setMessage] = useState("");
  const initial = addresses.find((a) => !a.isLoopback)?.url || "";
  useEffect(() => {
    if (isOpen) {
      setSelected((previous) => previous || initial);
      setMessage("");
    }
  }, [isOpen, initial]);
  let safe = "";
  try {
    safe = safePairingUrl(selected);
  } catch {
    /* Invalid manual input must never become a QR. */
  }
  useEffect(() => {
    let disposed = false;
    setQr("");
    if (safe)
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
  }, [safe]);
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
            onChange={(e) => setSelected(e.target.value)}
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
            onChange={(e) => setSelected(e.target.value)}
            placeholder="http://192.168.1.2:8090"
          />
        </label>
        {!safe && <p role="status">{t("pairing.invalidAddress")}</p>}
        {safe && (
          <>
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
