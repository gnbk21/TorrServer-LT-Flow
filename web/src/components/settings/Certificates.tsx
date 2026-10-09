import { useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { certificatesApi } from "../../api/certificates";
import { Button } from "../common/Button";
import { Modal } from "../common/Modal";
import { Loading, RequestError } from "../common/RequestState";

export function Certificates({ disabled }: { disabled: boolean }) {
  const { t } = useTranslation();
  const cache = useQueryClient();
  const query = useQuery({
    queryKey: ["certificates"],
    queryFn: ({ signal }) => certificatesApi.status(signal),
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const [notice, setNotice] = useState(false);
  const [certPath, setCertPath] = useState("");
  const [keyPath, setKeyPath] = useState("");
  const [certFile, setCertFile] = useState<File>();
  const [keyFile, setKeyFile] = useState<File>();
  const [confirm, setConfirm] = useState<"regenerate" | "selfsigned">();
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => request.current?.abort(), []);
  const status = query.data;
  const locked =
    disabled ||
    busy ||
    !status?.enabled ||
    status.read_only ||
    status.cert_from_flags;
  const change = async (
    action: "paths" | "upload" | "selfsigned" | "regenerate",
  ) => {
    if (locked || !status) return;
    const controller = new AbortController();
    request.current = controller;
    setBusy(true);
    setError(undefined);
    setNotice(false);
    setConfirm(undefined);
    try {
      let body: FormData | { cert: string; key: string } | undefined;
      if (action === "upload") {
        if (
          !certFile ||
          !keyFile ||
          certFile.size > 1048576 ||
          keyFile.size > 1048576
        )
          throw new Error(t("settings.certificates.fileLimit"));
        body = new FormData();
        body.set("cert", certFile);
        body.set("key", keyFile);
      } else if (action === "paths")
        body = { cert: certPath.trim(), key: keyPath.trim() };
      const result = await certificatesApi.change(
        action,
        status.revision,
        body,
        controller.signal,
      );
      if (controller.signal.aborted) return;
      cache.setQueryData(["certificates"], result);
      await Promise.all([
        cache.invalidateQueries({ queryKey: ["configuration"] }),
        cache.invalidateQueries({ queryKey: ["settings"] }),
      ]);
      setNotice(true);
      setKeyFile(undefined);
      setCertFile(undefined);
    } catch (e) {
      if (!controller.signal.aborted) setError(e);
    } finally {
      if (!controller.signal.aborted) setBusy(false);
    }
  };
  return (
    <section className="panel space-y-4">
      <h2 className="font-semibold">{t("settings.certificates.title")}</h2>
      <p className="text-sm">{t("settings.certificates.startupHint")}</p>
      {query.isPending && <Loading />}
      {query.error && (
        <RequestError error={query.error} retry={() => query.refetch()} />
      )}
      {status && (
        <>
          <dl className="grid sm:grid-cols-2 gap-2 text-sm">
            <dt>{t("settings.certificates.mode")}</dt>
            <dd>
              {t(
                `settings.certificates.modes.${!status.enabled ? "http" : !status.http_enabled ? "only" : status.http_media ? "media" : status.force_https ? "redirect" : "both"}`,
              )}
            </dd>
            <dt>{t("settings.certificates.source")}</dt>
            <dd>{t(`settings.certificates.sources.${status.cert.source}`)}</dd>
            {status.cert.subject && (
              <>
                <dt>{t("settings.certificates.subject")}</dt>
                <dd className="break-all">{status.cert.subject}</dd>
              </>
            )}
            {status.cert.issuer && (
              <>
                <dt>{t("settings.certificates.issuer")}</dt>
                <dd className="break-all">{status.cert.issuer}</dd>
              </>
            )}
            {status.cert.not_after && (
              <>
                <dt>{t("settings.certificates.expires")}</dt>
                <dd>{new Date(status.cert.not_after).toLocaleString()}</dd>
              </>
            )}
            <dt>{t("settings.certificates.names")}</dt>
            <dd className="break-all">
              {[
                ...(status.cert.dns_names ?? []),
                ...(status.cert.ips ?? []),
              ].join(", ") || "—"}
            </dd>
            <dt>{t("settings.certificates.trust")}</dt>
            <dd>
              {t(
                `settings.certificates.${status.cert.trusted ? "trusted" : "untrusted"}`,
              )}
            </dd>
          </dl>
          {status.enabled && (
            <p className="text-sm">{t("settings.certificates.trustHint")}</p>
          )}
          {status.cert.error && <p role="alert">{status.cert.error}</p>}
          {status.enabled && (
            <a className="underline" href="/ssl/cert" download="torrserver.crt">
              {t("settings.certificates.download")}
            </a>
          )}
          {disabled && <p>{t("settings.certificates.draftHint")}</p>}
          {status.read_only && <p>{t("settings.certificates.readOnly")}</p>}
          {status.cert_from_flags && <p>{t("settings.certificates.flags")}</p>}
          <fieldset disabled={locked} className="space-y-3">
            <div className="grid sm:grid-cols-2 gap-3">
              <label className="field">
                {t("settings.certificates.certFile")}
                <input
                  type="file"
                  accept=".pem,.crt,.cer"
                  onChange={(e) => setCertFile(e.target.files?.[0])}
                />
              </label>
              <label className="field">
                {t("settings.certificates.keyFile")}
                <input
                  type="file"
                  accept=".pem,.key"
                  onChange={(e) => setKeyFile(e.target.files?.[0])}
                />
              </label>
            </div>
            <p className="text-sm">{t("settings.certificates.uploadHint")}</p>
            <Button
              disabled={!certFile || !keyFile}
              onClick={() => void change("upload")}
            >
              {t("settings.certificates.upload")}
            </Button>
            <div className="grid sm:grid-cols-2 gap-3">
              <label className="field">
                {t("settings.certificates.certPath")}
                <input
                  value={certPath}
                  onChange={(e) => setCertPath(e.target.value)}
                  autoComplete="off"
                />
              </label>
              <label className="field">
                {t("settings.certificates.keyPath")}
                <input
                  value={keyPath}
                  onChange={(e) => setKeyPath(e.target.value)}
                  autoComplete="off"
                />
              </label>
            </div>
            <Button
              disabled={!certPath.trim() || !keyPath.trim()}
              onClick={() => void change("paths")}
            >
              {t("settings.certificates.usePaths")}
            </Button>
            <div className="actions">
              <Button onClick={() => setConfirm("selfsigned")}>
                {t("settings.certificates.selfsigned")}
              </Button>
              <Button
                disabled={status.cert.source !== "self-signed"}
                onClick={() => setConfirm("regenerate")}
              >
                {t("settings.certificates.regenerate")}
              </Button>
            </div>
          </fieldset>
        </>
      )}
      {!!error && <RequestError error={error} retry={() => query.refetch()} />}
      {notice && <p role="status">{t("settings.certificates.saved")}</p>}
      <Modal
        isOpen={!!confirm}
        onClose={() => setConfirm(undefined)}
        title={t("settings.certificates.confirmTitle")}
      >
        <p>{t("settings.certificates.confirmHint")}</p>
        <div className="actions">
          <Button
            disabled={locked}
            onClick={() => confirm && void change(confirm)}
          >
            {t("Confirm")}
          </Button>
          <Button onClick={() => setConfirm(undefined)}>{t("Cancel")}</Button>
        </div>
      </Modal>
    </section>
  );
}
