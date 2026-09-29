import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import en from "./locales/en/translation.json";
const loaders: Record<string, () => Promise<{ default: unknown }>> = {
  ru: () => import("./locales/ru/translation.json"),
  ua: () => import("./locales/ua/translation.json"),
  bg: () => import("./locales/bg/translation.json"),
  fr: () => import("./locales/fr/translation.json"),
  ro: () => import("./locales/ro/translation.json"),
  zh: () => import("./locales/zh/translation.json"),
};
i18n
  .use({
    type: "backend" as const,
    init() {},
    read(
      language: string,
      _namespace: string,
      callback: (error: unknown, data: unknown) => void,
    ) {
      const loader = loaders[language];
      if (!loader) {
        callback(null, en);
        return;
      }
      loader()
        .then((module) => callback(null, module.default))
        .catch((error) => callback(error, null));
    },
  })
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: { en: { translation: en } },
    partialBundledLanguages: true,
    supportedLngs: ["en", "ru", "ua", "bg", "fr", "ro", "zh"],
    fallbackLng: "en",
    load: "languageOnly",
    interpolation: { escapeValue: false },
    react: { useSuspense: false },
    detection: {
      order: ["localStorage", "navigator"],
      caches: ["localStorage"],
      convertDetectedLanguage: (language) =>
        language.startsWith("uk") ? "ua" : language,
    },
  });
export default i18n;
