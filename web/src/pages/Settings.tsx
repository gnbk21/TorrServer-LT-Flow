import { useEffect, useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import type { BTSettings } from "../types/settings";
import { useForm, type RegisterOptions } from "react-hook-form";
import { useTranslation } from "react-i18next";
import {
  useSettings,
  useRuntime,
  useVisible,
  queryClient,
} from "../hooks/queries";
import { useDirty } from "../hooks/dirty";
import { settingsApi } from "../api/settings";
import { torrentsApi } from "../api/torrents";
import { runtimeApi } from "../api/runtime";
import {
  flattenSettings,
  mergeSettings,
  validateSettings,
  flowBounds,
  type SettingsValue,
} from "../lib/settings";
import { Button } from "../components/common/Button";
import { Modal } from "../components/common/Modal";
import { Loading, RequestError } from "../components/common/RequestState";
import IntegrationSettings from "../components/settings/IntegrationSettings";
import { MaintenancePanel } from "../components/settings/MaintenancePanel";
import { LANTest } from "../components/settings/LANTest";
import { Certificates } from "../components/settings/Certificates";
const groups: Record<string, string[]> = {
  general: [
    "CacheSize",
    "ReaderReadAHead",
    "PreloadCache",
    "UseDisk",
    "TorrentsSavePath",
    "RemoveCacheOnDrop",
    "FriendlyName",
    "TrackTimecode",
    "MergeAllM3U",
  ],
  network: [
    "ForceEncrypt",
    "RetrackersMode",
    "TrackersListURL",
    "DefaultTrackers",
    "TorrentDisconnectTimeout",
    "EnableIPv6",
    "DisableTCP",
    "DisableUTP",
    "DisableUPNP",
    "DisableDHT",
    "DisablePEX",
    "DisableUpload",
    "DisableEndGame",
    "DownloadRateLimit",
    "UploadRateLimit",
    "ConnectionsLimit",
    "DHTConnectionsLimit",
    "PeersListenPort",
    "EnableLPD",
    "TrustedProxies",
    "Flow.TorrentInterface",
    "Flow.RequireTorrentInterface",
  ],
  integrations: [
    "EnableDLNA",
    "EnableBonjour",
    "EnableRutorSearch",
    "EnableTorznabSearch",
    "TorznabUrls",
    "EnableJacRedSearch",
    "JacRedUrl",
    "JacRedKey",
  ],
  security: [
    "SslPort",
    "Flow.ManagementOrigins",
    "Flow.ManagementRateLimit",
    "Flow.SecurityProfile",
    "Flow.RequirePlaybackToken",
    "Flow.PlaybackTokenTTL",
    "Flow.MSXAllowLAN",
  ],
  advanced: [
    "PadTailPartial",
    "EnableDebug",
    "ShowFSActiveTorr",
    "StoreSettingsInJson",
    "StoreViewedInJson",
  ],
};
export default function Settings() {
  const { t } = useTranslation();
  const query = useSettings();
  const runtime = useRuntime();
  const visible = useVisible();
  const configuration = useQuery({
    queryKey: ["configuration"],
    queryFn: ({ signal }) => settingsApi.state(signal),
    refetchInterval: visible ? 5000 : false,
  });
  const [draftRevision, setDraftRevision] = useState("");
  const [plan, setPlan] = useState<{
    restart_required: boolean;
    active_work: boolean;
  }>();
  const { setDirty } = useDirty();
  const [tab, setTab] = useState("general");
  const [search, setSearch] = useState("");
  const [error, setError] = useState<unknown>();
  const [saving, setSaving] = useState(false);
  const [confirm, setConfirm] = useState<
    "save" | "reset" | "wipe" | "shutdown"
  >();
  const [pending, setPending] = useState<Record<string, SettingsValue>>();
  const [notice, setNotice] = useState("");
  const [integrationDirty, setIntegrationDirty] = useState(false);
  const [certificateDirty, setCertificateDirty] = useState(false);
  const {
    register: rawRegister,
    handleSubmit,
    reset: rawReset,
    watch,
    setValue: rawSetValue,
    setError: rawFieldError,
    formState,
  } = useForm<Record<string, SettingsValue>>();
  const field = (key: string) => key.replaceAll(".", "__");
  const register = (
    key: string,
    options?: RegisterOptions<Record<string, SettingsValue>>,
  ) => rawRegister(field(key), options);
  const setValue = (
    key: string,
    value: SettingsValue,
    options: { shouldDirty: boolean },
  ) => rawSetValue(field(key), value, options);
  const fieldError = (key: string, error: { message: string }) =>
    rawFieldError(field(key), error);
  const decode = (data: Record<string, SettingsValue>) =>
    Object.fromEntries(
      Object.entries(data).map(([key, value]) => [
        key.replaceAll("__", "."),
        value,
      ]),
    );
  const reset = useCallback(
    (data: Record<string, SettingsValue>) =>
      rawReset(
        Object.fromEntries(
          Object.entries(data).map(([key, value]) => [
            key.replaceAll(".", "__"),
            value,
          ]),
        ),
      ),
    [rawReset],
  );
  const values = decode(watch());
  useEffect(() => {
    const saved = (configuration.data?.saved ?? query.data) as
      BTSettings | undefined;
    if (saved && !formState.isDirty) {
      reset(flattenSettings(saved));
      setDraftRevision(configuration.data?.revision ?? "");
    }
  }, [query.data, configuration.data, reset, formState.isDirty]);
  const dirty = formState.isDirty || integrationDirty || certificateDirty;
  useEffect(() => {
    setDirty(dirty);
    const before = (e: BeforeUnloadEvent) => {
      if (dirty) e.preventDefault();
    };
    window.addEventListener("beforeunload", before);
    return () => {
      window.removeEventListener("beforeunload", before);
      setDirty(false);
    };
  }, [dirty, setDirty]);
  if (query.isPending || configuration.isPending) return <Loading />;
  if (!query.data)
    return <RequestError error={query.error} retry={() => query.refetch()} />;
  const original = (configuration.data?.saved ?? query.data) as BTSettings;
  const submit = async (raw: Record<string, SettingsValue>) => {
    const data = decode(raw);
    const errors = validateSettings(data);
    if (Object.keys(errors).length) {
      for (const [name, message] of Object.entries(errors))
        fieldError(name, { message: t(message, { defaultValue: message }) });
      setNotice(t("settings.fixErrors"));
      const first = Object.keys(errors)[0]!;
      setTab(
        first.startsWith("Flow.SwarmCustom.")
          ? "advanced"
          : first.startsWith("Flow.")
            ? "flow"
            : Object.keys(groups).find((group) =>
                groups[group]?.includes(first),
              ) || "integrations",
      );
      requestAnimationFrame(() =>
        document.getElementById(field(first))?.focus(),
      );
      return;
    }
    setError(undefined);
    try {
      setPlan(await settingsApi.plan(mergeSettings(original, data)));
      setPending(data);
      setConfirm("save");
    } catch (e) {
      setError(e);
    }
  };
  const apply = async (when: "now" | "idle" = "now") => {
    setSaving(true);
    setError(undefined);
    try {
      if (confirm === "save" && pending) {
        await settingsApi.apply(
          mergeSettings(original, pending),
          draftRevision,
          when,
        );
        const fresh = await settingsApi.get();
        queryClient.setQueryData(["settings"], fresh);
        const state = await configuration.refetch();
        reset(flattenSettings((state.data?.saved ?? fresh) as BTSettings));
        setDraftRevision(state.data?.revision ?? "");
        setNotice(
          t(
            when === "idle" && plan?.restart_required
              ? "settings.queued"
              : "settings.saved",
          ),
        );
      }
      if (confirm === "reset") {
        await settingsApi.reset(draftRevision);
        const fresh = await settingsApi.get();
        queryClient.setQueryData(["settings"], fresh);
        reset(flattenSettings(fresh));
        await configuration.refetch();
      }
      if (confirm === "wipe") {
        await torrentsApi.wipe();
        await queryClient.invalidateQueries({ queryKey: ["torrents"] });
      }
      if (confirm === "shutdown") await runtimeApi.shutdown();
      setConfirm(undefined);
      await queryClient.invalidateQueries({ queryKey: ["runtime"] });
    } catch (e) {
      setError(e);
    } finally {
      setSaving(false);
    }
  };
  const labelFor = (key: string) =>
    t([`settings.fields.${key}`, `SettingsDialog.${key}`], {
      defaultValue: key
        .replace(/^Flow\.|^TMDBSettings\./, "")
        .replace(/([a-z])([A-Z])/g, "$1 $2"),
    });
  const hintFor = (key: string) =>
    [
      t(`SettingsDialog.${key}Hint`, { defaultValue: "" }),
      key === "Flow.MSXAllowLAN" ? t("flowHelp.MSXAllowLAN") : "",
      key.startsWith("Flow.")
        ? t(`settings.fieldHints.${key.slice(5)}`, { defaultValue: "" })
        : "",
      (
        {
          "Flow.DiagnosticHistory": t("settings.historyHint"),
          "Flow.DHTStatePersistence": t("settings.dhtHint"),
          "Flow.PeerResumeHints": t("settings.peerHintsHelp"),
          "Flow.ScarcePieceHints": t("settings.scarceHintsHelp"),
          "Flow.RateAwareDeadlines": t("settings.rateDeadlinesHelp"),
          "Flow.CapacityAwareRequests": t("settings.highBitrateExperimentHelp"),
          "Flow.AdaptiveUrgentHorizon": t("settings.highBitrateExperimentHelp"),
          "Flow.ContainerBurstHints": t("settings.highBitrateExperimentHelp"),
        } as Record<string, string>
      )[key] || "",
    ]
      .filter(Boolean)
      .join(" ");
  const originalFields = flattenSettings(original);
  const effectiveFields = configuration.data?.effective
    ? flattenSettings(configuration.data.effective as BTSettings)
    : {};
  const changed = Object.keys(values).filter(
    (key) => values[key] !== originalFields[key],
  );
  const keys = Object.keys(values).filter(
    (key) =>
      key !== "Flow.SchemaVersion" &&
      (search.trim()
        ? `${labelFor(key)} ${hintFor(key)}`
            .toLocaleLowerCase()
            .includes(search.trim().toLocaleLowerCase())
        : tab === "flow"
          ? key.startsWith("Flow.") &&
            key !== "Flow.SchemaVersion" &&
            !groups.security!.includes(key) &&
            !groups.network!.includes(key) &&
            !key.startsWith("Flow.SwarmCustom.")
          : tab === "advanced"
            ? groups.advanced!.includes(key) ||
              key.startsWith("Flow.SwarmCustom.")
            : tab === "integrations"
              ? groups.integrations!.includes(key) ||
                key.startsWith("TMDBSettings.")
              : groups[tab]?.includes(key)),
  );
  return (
    <div className="space-y-5">
      <h1 className="text-2xl font-semibold">{t("nav.settings")}</h1>
      <label className="field max-w-xl">
        {t("settings.search")}
        <input
          type="search"
          value={search}
          maxLength={256}
          onChange={(event) => setSearch(event.target.value)}
        />
      </label>
      <div className="panel space-y-2" aria-live="polite">
        <p>{t(dirty ? "settings.unsaved" : "settings.savedState")}</p>
        {!!changed.length && (
          <details>
            <summary>
              {t("settings.changedFields", { count: changed.length })}
            </summary>
            <ul className="list-disc pl-5">
              {changed.map((key) => (
                <li key={key}>{labelFor(key)}</li>
              ))}
            </ul>
          </details>
        )}
        <p className="text-sm">
          {t("settings.cacheState", {
            draft: (Number(values.CacheSize) / 1048576).toFixed(0),
            saved: (original.CacheSize / 1048576).toFixed(0),
            effective: configuration.data?.effective
              ? (
                  Number(configuration.data.effective.CacheSize) / 1048576
                ).toFixed(0)
              : "—",
          })}
        </p>
        {configuration.data && (
          <details className="text-xs break-all">
            <summary>{t("settings.identity")}</summary>
            <p>{configuration.data.executable}</p>
            <p>{configuration.data.data_path}</p>
            <p>{configuration.data.revision}</p>
          </details>
        )}
        {configuration.data?.recovery.issue && (
          <p role="alert">
            {t("settings.recovered", {
              source: configuration.data.recovery.source,
            })}
          </p>
        )}
        {configuration.data?.error && (
          <p role="alert">{configuration.data.error}</p>
        )}
        {configuration.data?.pending && (
          <div className="actions">
            <span>{t("settings.queued")}</span>
            <Button
              onClick={async () => {
                try {
                  await settingsApi.cancelPending(configuration.data!.revision);
                  await configuration.refetch();
                } catch (e) {
                  setError(e);
                }
              }}
            >
              {t("Cancel")}
            </Button>
          </div>
        )}
      </div>
      <div className="actions" role="tablist">
        {[
          "general",
          "flow",
          "network",
          "integrations",
          "security",
          "advanced",
        ].map((name) => (
          <Button
            role="tab"
            aria-selected={tab === name}
            key={name}
            variant={tab === name ? "primary" : "secondary"}
            onClick={() => {
              setTab(name);
              setSearch("");
            }}
          >
            {t(`settings.tabs.${name}`)}
          </Button>
        ))}
      </div>
      {query.error && (
        <RequestError error={query.error} stale retry={() => query.refetch()} />
      )}{" "}
      {!!error && <RequestError error={error} />}{" "}
      {configuration.error && (
        <RequestError
          error={configuration.error}
          retry={() => configuration.refetch()}
        />
      )}
      {configuration.data?.applying && (
        <p role="status">{t("settings.applying")}</p>
      )}
      {notice && <p role="status">{notice}</p>}
      <div hidden={tab !== "security"}>
        <Certificates
          visible={tab === "security"}
          onDirty={setCertificateDirty}
          disabled={
            formState.isDirty ||
            integrationDirty ||
            !!configuration.data?.pending ||
            !!configuration.data?.applying
          }
        />
      </div>
      <form onSubmit={handleSubmit(submit)} className="space-y-5">
        {tab === "flow" && (
          <p className="panel text-sm">{t("settings.flowHint")}</p>
        )}
        {tab === "network" && <LANTest />}
        {tab === "general" && (
          <div className="actions">
            {[256, 512, 1024, 2048, 4096].map((mib) => (
              <Button
                key={mib}
                onClick={() =>
                  setValue("CacheSize", mib * 1048576, { shouldDirty: true })
                }
              >
                {mib} MiB
              </Button>
            ))}
          </div>
        )}
        <div className="grid md:grid-cols-2 gap-4">
          {keys.map((key) => {
            const value = values[key];
            const label = labelFor(key);
            const bounds = flowBounds[key.replace("Flow.", "")];
            return (
              <div
                className={`field panel ${key === "DefaultTrackers" || key === "TorznabUrls" ? "md:col-span-2" : ""}`}
                key={key}
              >
                <label htmlFor={field(key)}>{label}</label>
                {typeof value === "number" && key !== "CacheSize" && (
                  <span className="text-xs text-slate-400">
                    {key.endsWith("MB")
                      ? "MiB"
                      : /Ms$/.test(key)
                        ? "ms"
                        : /(?:Seconds|Sec|Timeout|RequestQueueTime|ReconnectTime|PlaybackTokenTTL)$/.test(
                              key,
                            )
                          ? "s"
                          : /(?:Pct|ReadAHead|PreloadCache)$/.test(key)
                            ? "%"
                            : ""}
                  </span>
                )}
                {!!search.trim() && (
                  <span className="text-xs text-slate-400">
                    {t(
                      `settings.tabs.${Object.keys(groups).find((group) => groups[group]?.includes(key)) || (key.startsWith("TMDBSettings.") ? "integrations" : key.startsWith("Flow.SwarmCustom.") ? "advanced" : "flow")}`,
                    )}
                  </span>
                )}
                {!!hintFor(key) && (
                  <span className="text-xs text-slate-400">{hintFor(key)}</span>
                )}
                {typeof value === "number" && configuration.data?.effective && (
                  <span className="text-xs text-slate-400">
                    {t("settings.effectiveValue", {
                      value:
                        key === "CacheSize"
                          ? `${Number(configuration.data.effective.CacheSize) / 1048576} MiB`
                          : (effectiveFields[key] ?? "—"),
                    })}
                  </span>
                )}
                {key === "CacheSize" && <span>MiB</span>}
                {key === "PreloadCache" && <span>%</span>}
                {key === "CacheSize" ? (
                  <input
                    id={field(key)}
                    type="number"
                    min={1}
                    max={16384}
                    step={1}
                    value={Number(value) / 1048576}
                    onChange={(e) =>
                      setValue(key, Number(e.target.value) * 1048576, {
                        shouldDirty: true,
                      })
                    }
                  />
                ) : typeof value === "boolean" ? (
                  <input id={field(key)} type="checkbox" {...register(key)} />
                ) : key === "Flow.SwarmProfile" ? (
                  <>
                    <select id={field(key)} {...register(key)}>
                      {[
                        "legacy",
                        "adaptive",
                        "conservative",
                        "balanced",
                        "aggressive",
                        "custom",
                      ].map((profile) => (
                        <option key={profile} value={profile}>
                          {t(`settings.profiles.${profile}`)}
                        </option>
                      ))}
                    </select>
                    <span className="text-sm text-slate-400">
                      {t(`settings.profileHints.${String(values[key])}`)}
                    </span>
                  </>
                ) : key === "Flow.SecurityProfile" ? (
                  <select id={field(key)} {...register(key)}>
                    <option value="compatible">
                      {t("settings.compatible")}
                    </option>
                    <option value="restricted">
                      {t("settings.restricted")}
                    </option>
                  </select>
                ) : key === "Flow.BootstrapTailMode" ? (
                  <select id={field(key)} {...register(key)}>
                    <option value="upstream-auto">
                      {t("settings.tailMode")}
                    </option>
                  </select>
                ) : key === "DefaultTrackers" ? (
                  <TrackerRows
                    value={String(value)}
                    onChange={(next) =>
                      setValue(key, next, { shouldDirty: true })
                    }
                  />
                ) : key === "TorznabUrls" ? (
                  <IndexerRows
                    value={String(value)}
                    onChange={(next) =>
                      setValue(key, next, { shouldDirty: true })
                    }
                  />
                ) : ["TrustedProxies", "SslCert", "SslKey"].includes(key) ? (
                  <textarea id={field(key)} rows={4} {...register(key)} />
                ) : (
                  <input
                    id={field(key)}
                    type={
                      typeof value === "number"
                        ? "number"
                        : /Key$/.test(key) || key === "TrackersListURL"
                          ? "password"
                          : "text"
                    }
                    step={typeof value === "number" ? 1 : undefined}
                    min={bounds?.[0]}
                    max={bounds?.[1]}
                    {...register(key, {
                      valueAsNumber: typeof value === "number",
                    })}
                  />
                )}
                {bounds && (
                  <span className="text-xs text-slate-400">
                    {bounds[0]}–{bounds[1]}
                  </span>
                )}
                {formState.errors[field(key)] && (
                  <span role="alert" className="text-rose-300">
                    {formState.errors[field(key)]?.message}
                  </span>
                )}
              </div>
            );
          })}
        </div>
        <div className="actions sticky bottom-20 lg:bottom-3 panel">
          <Button
            type="submit"
            variant="primary"
            disabled={
              !formState.isDirty ||
              saving ||
              !draftRevision ||
              !!configuration.data?.applying
            }
          >
            {t("settings.apply")}
          </Button>
          <Button
            disabled={!formState.isDirty || saving}
            onClick={() => reset(flattenSettings(original))}
          >
            {t("settings.discard")}
          </Button>
          <p className="text-xs text-slate-400" role="status">
            {t(dirty ? "settings.unsaved" : "settings.savedState")}
          </p>
        </div>
      </form>
      <IntegrationSettings tab={tab} onDirty={setIntegrationDirty} />
      {tab === "advanced" && <MaintenancePanel dirty={dirty} />}
      {tab === "advanced" && (
        <section className="panel border-rose-900 space-y-3">
          <h2>{t("settings.danger")}</h2>
          <div className="actions">
            {(["reset", "wipe", "shutdown"] as const).map((action) => (
              <Button
                key={action}
                variant="danger"
                onClick={() => setConfirm(action)}
              >
                {t(`settings.${action}`)}
              </Button>
            ))}
          </div>
        </section>
      )}
      <Modal
        isOpen={!!confirm}
        onClose={() => {
          if (!saving) setConfirm(undefined);
        }}
        title={t(`settings.confirm.${confirm || "save"}`)}
      >
        <p>
          {t(
            confirm === "save" &&
              plan?.restart_required &&
              Number(runtime.data?.bt?.active_streams) > 0
              ? "settings.activeRestart"
              : "settings.confirmHint",
          )}
        </p>
        {confirm === "save" && (
          <p>
            {t(
              plan?.restart_required
                ? "settings.needsRestart"
                : "settings.hotApply",
            )}
          </p>
        )}
        <div className="actions mt-4">
          <Button onClick={() => setConfirm(undefined)} disabled={saving}>
            {t("Cancel")}
          </Button>
          <Button
            variant="danger"
            isLoading={saving}
            onClick={() => void apply()}
          >
            {t("settings.confirmAction")}
          </Button>
          {confirm === "save" && plan?.restart_required && (
            <Button isLoading={saving} onClick={() => void apply("idle")}>
              {t("settings.applyIdle")}
            </Button>
          )}
        </div>
        {!!error && <RequestError error={error} />}
      </Modal>
    </div>
  );
}
function TrackerRows({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  const { t } = useTranslation();
  const rows = value ? value.split(/\r?\n/) : [""];
  return (
    <div className="space-y-2">
      {rows.map((url, index) => (
        <label className="field" key={index}>
          {t("Tracker")} {index + 1}
          <div className="actions">
            <input
              className="min-w-0 flex-1"
              type="password"
              autoComplete="off"
              value={url}
              onChange={(event) =>
                onChange(
                  rows
                    .map((row, i) => (i === index ? event.target.value : row))
                    .join("\n"),
                )
              }
            />
            <Button
              onClick={() =>
                onChange(rows.filter((_row, i) => i !== index).join("\n"))
              }
            >
              {t("Delete")}
            </Button>
          </div>
        </label>
      ))}
      <Button onClick={() => onChange([...rows, ""].join("\n"))}>
        {t("Add")} {t("Tracker")}
      </Button>
    </div>
  );
}
function IndexerRows({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  const { t } = useTranslation();
  const rows: { Host: string; Key: string; Name: string }[] = JSON.parse(value);
  return (
    <div className="space-y-3">
      {rows.map((row, index) => (
        <div key={index} className="grid sm:grid-cols-3 gap-2">
          {(["Name", "Host", "Key"] as const).map((key) => (
            <label className="field" key={key}>
              {t(`settings.indexer.${key}`)}
              <input
                type={key === "Key" ? "password" : "text"}
                value={row[key]}
                onChange={(e) =>
                  onChange(
                    JSON.stringify(
                      rows.map((r, i) =>
                        i === index ? { ...r, [key]: e.target.value } : r,
                      ),
                    ),
                  )
                }
              />
            </label>
          ))}
          <Button
            onClick={() =>
              onChange(JSON.stringify(rows.filter((_r, i) => i !== index)))
            }
          >
            {t("torrent.remove")}
          </Button>
        </div>
      ))}
      <Button
        onClick={() =>
          onChange(JSON.stringify([...rows, { Name: "", Host: "", Key: "" }]))
        }
      >
        {t("settings.addIndexer")}
      </Button>
    </div>
  );
}
