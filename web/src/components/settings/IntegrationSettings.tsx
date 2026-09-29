import { validateGst } from "../../lib/gstreamer";
import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import {
  integrationsApi,
  type WafSettings,
  type StorageSettings,
  type GstSettings,
} from "../../api/integrations";
import { useSettings, useRuntime } from "../../hooks/queries";
import { Button } from "../common/Button";
import { Loading, RequestError } from "../common/RequestState";
export default function IntegrationSettings({
  tab,
  onDirty,
}: {
  tab: string;
  onDirty: (dirty: boolean) => void;
}) {
  const { t } = useTranslation();
  const settings = useSettings();
  const runtime = useRuntime();
  const waf = useQuery({
    queryKey: ["waf"],
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    staleTime: Infinity,
    queryFn: ({ signal }) => integrationsApi.getWAF(signal),
    enabled: tab === "security",
  });
  const gst = useQuery({
    queryKey: ["gst"],
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    staleTime: Infinity,
    queryFn: ({ signal }) => integrationsApi.getGst(signal),
    enabled: tab === "integrations",
  });
  const storage = useQuery({
    queryKey: ["storage"],
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    staleTime: Infinity,
    queryFn: ({ signal }) => integrationsApi.getStorage(signal),
    enabled: tab === "advanced",
  });
  const gstHealth = useQuery({
    queryKey: ["gst-health"],
    queryFn: ({ signal }) => integrationsApi.getGstHealth(signal),
    enabled: tab === "integrations" && gst.data?.built_in === true,
    staleTime: 30000,
  });
  const [wafDraft, setWaf] = useState<WafSettings>();
  const [gstDraft, setGst] = useState<GstSettings["config"]>();
  const [storageDraft, setStorage] = useState<StorageSettings>();
  const [error, setError] = useState<unknown>();
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    if (waf.data) setWaf(waf.data);
  }, [waf.data]);
  useEffect(() => {
    if (gst.data) setGst(gst.data.config);
  }, [gst.data]);
  useEffect(() => {
    if (storage.data) setStorage(storage.data);
  }, [storage.data]);
  const dirty =
    (!!wafDraft && JSON.stringify(wafDraft) !== JSON.stringify(waf.data)) ||
    (!!gstDraft &&
      JSON.stringify(gstDraft) !== JSON.stringify(gst.data?.config)) ||
    (!!storageDraft &&
      JSON.stringify(storageDraft) !== JSON.stringify(storage.data));
  useEffect(() => {
    onDirty(dirty);
  }, [dirty, onDirty]);
  const run = async (action: () => Promise<unknown>) => {
    setBusy(true);
    setError(undefined);
    try {
      await action();
      setNotice(t("settings.saved"));
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  if (!["security", "integrations", "advanced"].includes(tab)) return null;
  return (
    <section className="panel space-y-4">
      {!!error && <RequestError error={error} />}{" "}
      {notice && <p role="status">{notice}</p>}
      {tab === "security" && (
        <>
          <h2 className="font-semibold">{t("WAF.Title")}</h2>
          <p>{t("WAF.SeparateSaveHint")}</p>
          <p className="text-sm text-slate-400">{t("settings.authHelp")}</p>
          {waf.isPending && <Loading />}
          {waf.error && (
            <RequestError error={waf.error} retry={() => waf.refetch()} />
          )}{" "}
          {wafDraft && (
            <>
              <p className="text-sm">{t("WAF.IPRulesHint")}</p>
              {(["whitelist", "blacklist", "referers"] as const).map((key) => (
                <label className="field" key={key}>
                  {t(
                    `WAF.${key === "whitelist" ? "Whitelist" : key === "blacklist" ? "Blacklist" : "Referers"}`,
                  )}
                  <textarea
                    rows={4}
                    disabled={wafDraft.read_only}
                    value={wafDraft[key]}
                    onChange={(e) =>
                      setWaf({ ...wafDraft, [key]: e.target.value })
                    }
                  />
                </label>
              ))}
              <label>
                <input
                  type="checkbox"
                  disabled={wafDraft.read_only}
                  checked={wafDraft.default_referers_enabled}
                  onChange={(e) =>
                    setWaf({
                      ...wafDraft,
                      default_referers_enabled: e.target.checked,
                    })
                  }
                />{" "}
                {t("WAF.DefaultReferers")}
              </label>
              <p className="text-sm break-all">
                {wafDraft.default_referers.join(", ")}
              </p>
              {wafDraft.read_only && <p>{t("WAF.ReadOnlyHint")}</p>}
              {wafDraft.warnings.map((warning, index) => (
                <p key={index} role="status">
                  {warning.list} {warning.line}:{" "}
                  {t(`WAF.WarningCodes.${warning.code}`, {
                    defaultValue: warning.code,
                  })}
                </p>
              ))}
              <Button
                disabled={
                  busy ||
                  wafDraft.read_only ||
                  JSON.stringify(wafDraft) === JSON.stringify(waf.data)
                }
                onClick={() =>
                  run(async () => {
                    const fresh = await integrationsApi.updateWAF(wafDraft);
                    setWaf(fresh);
                    await waf.refetch();
                  })
                }
              >
                {t("WAF.SaveLists")}
              </Button>
              <Button onClick={() => setWaf(waf.data)}>
                {t("settings.discard")}
              </Button>
            </>
          )}
        </>
      )}
      {tab === "integrations" && (
        <>
          <h2 className="font-semibold">{t("settings.integrationStatus")}</h2>
          <p>
            DLNA:{" "}
            {t(
              runtime.data?.dlna_enabled ? "status.enabled" : "status.disabled",
            )}{" "}
            · Bonjour:{" "}
            {t(
              runtime.data?.bonjour_enabled
                ? "status.enabled"
                : "status.disabled",
            )}{" "}
            · WebDAV:{" "}
            {t(
              runtime.data?.webdav_enabled
                ? "status.enabled"
                : "status.disabled",
            )}
          </p>
          {runtime.data?.webdav_enabled && (
            <a href={runtime.data.webdav_path} target="_blank" rel="noreferrer">
              WebDAV
            </a>
          )}
          <p className="text-sm">{t("settings.testSaved")}</p>
          <div className="actions">
            {settings.data?.TorznabUrls?.map((row, index) => (
              <Button
                key={index}
                disabled={busy}
                onClick={() =>
                  run(async () => {
                    const result = await integrationsApi.testTorznab(
                      row.Host,
                      row.Key,
                    );
                    if (!result.success)
                      throw new Error("settings.integrationFailed");
                  })
                }
              >
                {t("Torznab.Test")} {row.Name || `Torznab ${index + 1}`}
              </Button>
            ))}
            {settings.data?.EnableJacRedSearch && (
              <Button
                disabled={busy}
                onClick={() =>
                  run(async () => {
                    const result = await integrationsApi.testJacred(
                      settings.data!.JacRedUrl,
                      settings.data!.JacRedKey,
                    );
                    if (!result.success)
                      throw new Error("settings.integrationFailed");
                  })
                }
              >
                {t("Torznab.Test")} JacRed
              </Button>
            )}
          </div>
          <h3>GStreamer</h3>
          {gstHealth.error && (
            <RequestError
              error={gstHealth.error}
              retry={() => gstHealth.refetch()}
            />
          )}
          {gstHealth.data && (
            <dl className="grid sm:grid-cols-2 gap-3">
              {Object.entries(gstHealth.data).map(([key, value]) => (
                <div key={key}>
                  <dt>{key.replaceAll("_", " ")}</dt>
                  <dd>
                    {t(
                      value.works
                        ? "GStreamer.StatusWorks"
                        : value.available
                          ? "GStreamer.StatusAvailable"
                          : value.found
                            ? "GStreamer.StatusFound"
                            : "GStreamer.StatusMissing",
                    )}{" "}
                    {value.version}
                  </dd>
                </div>
              ))}
            </dl>
          )}
          {gst.isPending && <Loading />}
          {gst.error && (
            <RequestError error={gst.error} retry={() => gst.refetch()} />
          )}{" "}
          {gst.data && !gst.data.built_in && (
            <p>{t("settings.gstUnavailable")}</p>
          )}
          {gst.data?.built_in && gstDraft && (
            <>
              <div className="grid sm:grid-cols-2 gap-3">
                {Object.entries(gstDraft).map(([key, value]) => (
                  <label key={key} className="field">
                    {t(
                      [
                        `settings.gst.${key}`,
                        `GStreamer.${key === "GSTVersion" ? "Version" : key === "GSTPath" ? "Path" : key}`,
                      ],
                      {
                        defaultValue: key.replaceAll("_", " "),
                      },
                    )}
                    <input
                      type={
                        typeof value === "boolean"
                          ? "checkbox"
                          : typeof value === "number"
                            ? "number"
                            : "text"
                      }
                      checked={typeof value === "boolean" ? value : undefined}
                      value={typeof value !== "boolean" ? value : undefined}
                      onChange={(e) =>
                        setGst({
                          ...gstDraft,
                          [key]:
                            typeof value === "boolean"
                              ? e.target.checked
                              : typeof value === "number"
                                ? e.target.valueAsNumber
                                : e.target.value,
                        })
                      }
                    />
                  </label>
                ))}
              </div>
              <Button
                disabled={busy}
                onClick={() =>
                  run(async () => {
                    if (!validateGst(gstDraft))
                      throw new Error("settings.invalidGst");
                    await integrationsApi.setGst(gstDraft);
                    await gst.refetch();
                  })
                }
              >
                {t("GStreamer.SaveSettings")}
              </Button>
              <Button onClick={() => setGst(gst.data?.config)}>
                {t("settings.discard")}
              </Button>
            </>
          )}
        </>
      )}
      {tab === "advanced" && (
        <>
          <h2>{t("settings.storage")}</h2>
          <p>{t("settings.storageRestart")}</p>
          {storage.isPending && <Loading />}
          {storage.error && (
            <RequestError
              error={storage.error}
              retry={() => storage.refetch()}
            />
          )}{" "}
          {storageDraft && (
            <>
              <div className="grid sm:grid-cols-2 gap-3">
                {(["settings", "viewed"] as const).map((key) => (
                  <label className="field" key={key}>
                    {t(`settings.storageFields.${key}`)}
                    <select
                      value={storageDraft[key]}
                      onChange={(e) =>
                        setStorage({
                          ...storageDraft,
                          [key]: e.target.value as "json" | "bbolt",
                        })
                      }
                    >
                      <option value="json">JSON</option>
                      <option value="bbolt">BBolt</option>
                    </select>
                  </label>
                ))}
              </div>
              <Button
                disabled={busy}
                onClick={() =>
                  run(async () => {
                    await integrationsApi.setStorage(storageDraft);
                    await storage.refetch();
                  })
                }
              >
                {t("Save")}
              </Button>
              <Button onClick={() => setStorage(storage.data)}>
                {t("settings.discard")}
              </Button>
            </>
          )}
        </>
      )}
    </section>
  );
}
