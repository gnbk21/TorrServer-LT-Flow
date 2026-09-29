import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { integrationsApi } from "../../api/integrations";
import { RequestError } from "../common/RequestState";

export default function GstRuntimeStatus() {
  const { t } = useTranslation();
  const gst = useQuery({
    queryKey: ["gst"],
    queryFn: ({ signal }) => integrationsApi.getGst(signal),
    staleTime: 30000,
  });
  const health = useQuery({
    queryKey: ["gst-health"],
    queryFn: ({ signal }) => integrationsApi.getGstHealth(signal),
    enabled: gst.data?.built_in === true,
    staleTime: 30000,
  });
  return (
    <section>
      {gst.error && (
        <RequestError error={gst.error} retry={() => gst.refetch()} />
      )}
      {gst.data && !gst.data.built_in && <p>{t("settings.gstUnavailable")}</p>}
      {health.error && (
        <RequestError error={health.error} retry={() => health.refetch()} />
      )}
      {health.data && (
        <dl className="grid sm:grid-cols-2 gap-3">
          {Object.entries(health.data).map(([key, value]) => (
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
    </section>
  );
}
