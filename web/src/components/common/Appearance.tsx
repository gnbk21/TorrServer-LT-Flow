import { useLayoutEffect } from "react";
import { useTranslation } from "react-i18next";
import { usePreference } from "../../hooks/preferences";

export function Appearance() {
  const { t } = useTranslation();
  const [theme, setTheme] = usePreference(
    "theme",
    "dark",
    (value): value is string =>
      typeof value === "string" && ["dark", "light", "system"].includes(value),
  );
  const [density, setDensity] = usePreference(
    "density",
    "comfortable",
    (value): value is string => value === "comfortable" || value === "compact",
  );
  useLayoutEffect(() => {
    const media = matchMedia("(prefers-color-scheme: dark)");
    const update = () => {
      document.documentElement.dataset.theme =
        theme === "system" ? (media.matches ? "dark" : "light") : theme;
    };
    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, [theme]);
  useLayoutEffect(() => {
    document.documentElement.dataset.density = density;
  }, [density]);
  return (
    <details className="relative">
      <summary>{t("app.appearance")}</summary>
      <div className="panel absolute right-0 top-full z-40 min-w-56 space-y-3">
        <label className="field">
          {t("app.theme")}
          <select
            value={theme}
            onChange={(event) => setTheme(event.target.value)}
          >
            {["dark", "light", "system"].map((value) => (
              <option key={value} value={value}>
                {t(`app.${value}`)}
              </option>
            ))}
          </select>
        </label>
        <label className="field">
          {t("app.density")}
          <select
            value={density}
            onChange={(event) => setDensity(event.target.value)}
          >
            {["comfortable", "compact"].map((value) => (
              <option key={value} value={value}>
                {t(`app.${value}`)}
              </option>
            ))}
          </select>
        </label>
      </div>
    </details>
  );
}
