import { useLayoutEffect } from "react";
import { useTranslation } from "react-i18next";
import { usePreference } from "../../hooks/preferences";

export function Appearance() {
  const { t } = useTranslation();
  const [theme, setTheme] = usePreference(
    "theme",
    "system",
    (value): value is string =>
      typeof value === "string" && ["dark", "light", "system"].includes(value),
  );
  const [density, setDensity] = usePreference(
    "density",
    "comfortable",
    (value): value is string => value === "comfortable" || value === "compact",
  );
  const [palette, setPalette] = usePreference(
    "palette",
    "iris",
    (v): v is string =>
      typeof v === "string" && ["iris", "lagoon", "rose"].includes(v),
  );
  const [scale, setScale] = usePreference(
    "textScale",
    100,
    (v): v is number =>
      typeof v === "number" && [100, 125, 150, 200].includes(v),
  );
  const [motion, setMotion] = usePreference(
    "motion",
    "auto",
    (v): v is string => v === "auto" || v === "reduce",
  );
  useLayoutEffect(() => {
    const root = document.documentElement;
    root.dataset.palette = palette;
    root.dataset.motion = motion;
    root.style.fontSize = `${scale}%`;
  }, [palette, scale, motion]);
  useLayoutEffect(() => {
    const media = matchMedia("(prefers-color-scheme: dark)");
    const update = () => {
      document.documentElement.dataset.theme =
        theme === "system" ? (media.matches ? "dark" : "light") : theme;
    };
    update();
    if (media.addEventListener) {
      media.addEventListener("change", update);
      return () => media.removeEventListener("change", update);
    }
    media.addListener(update);
    return () => media.removeListener(update);
  }, [theme]);
  useLayoutEffect(() => {
    document.documentElement.dataset.density = density;
  }, [density]);
  return (
    <details className="relative" data-appearance>
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
        <label className="field">
          {t("app.expressive.palette")}
          <select value={palette} onChange={(e) => setPalette(e.target.value)}>
            {["iris", "lagoon", "rose"].map((v) => (
              <option key={v} value={v}>
                {t(`app.expressive.${v}`)}
              </option>
            ))}
          </select>
        </label>
        <label className="field">
          {t("app.expressive.textScale")}
          <select
            value={scale}
            onChange={(e) => setScale(Number(e.target.value))}
          >
            {[100, 125, 150, 200].map((v) => (
              <option key={v} value={v}>
                {v}%
              </option>
            ))}
          </select>
        </label>
        <label className="field">
          {t("app.expressive.motion")}
          <select value={motion} onChange={(e) => setMotion(e.target.value)}>
            <option value="auto">{t("app.system")}</option>
            <option value="reduce">{t("app.expressive.reduced")}</option>
          </select>
        </label>
      </div>
    </details>
  );
}
