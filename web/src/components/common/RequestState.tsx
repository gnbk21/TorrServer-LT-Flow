import { useTranslation } from "react-i18next";
import { ApiError } from "../../api/client";
import { Button } from "./Button";
export function RequestError({
  error,
  retry,
  stale = false,
}: {
  error: unknown;
  retry?: () => void;
  stale?: boolean;
}) {
  const { t, i18n } = useTranslation();
  const key =
    error instanceof ApiError
      ? error.isStarting
        ? "status.starting"
        : error.isUnauthorized
          ? "status.authRequired"
          : "status.requestFailed"
      : error instanceof Error && i18n.exists(error.message)
        ? error.message
        : error instanceof DOMException && error.name === "TimeoutError"
          ? "status.timeout"
          : error instanceof TypeError
            ? "status.offline"
            : "status.requestFailed";
  return (
    <div
      role="alert"
      className="panel border-amber-700 text-amber-200 space-y-2"
    >
      <p>{t(key)}</p>
      {stale && <p>{t("status.stale")}</p>}
      {retry && <Button onClick={retry}>{t("WAF.Retry")}</Button>}
    </div>
  );
}
export function Loading() {
  const { t } = useTranslation();
  return (
    <p role="status" className="panel">
      {t("Loading")}
    </p>
  );
}
