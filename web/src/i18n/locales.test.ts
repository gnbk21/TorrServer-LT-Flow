import { describe, it, expect } from "vitest";
import en from "./locales/en/translation.json";
const resources = import.meta.glob<{ default: Record<string, unknown> }>(
  "./locales/*/translation.json",
  { eager: true },
);
function flatten(
  value: Record<string, unknown>,
  prefix = "",
): Record<string, string> {
  return Object.fromEntries(
    Object.entries(value).flatMap(([key, v]) =>
      typeof v === "string"
        ? [[prefix + key, v]]
        : v && typeof v === "object"
          ? Object.entries(
              flatten(v as Record<string, unknown>, prefix + key + "."),
            )
          : [],
    ),
  );
}
describe("modern language coverage", () => {
  const english = flatten(en);
  const keys = Object.keys(english).filter((key) =>
    /^(nav|status|dashboard|flow|torrent|pairing|search|settings|runtime|app)\./.test(
      key,
    ),
  );
  for (const [name, module] of Object.entries(resources))
    it(name, () => {
      const translated = flatten(module.default);
      for (const key of keys) {
        expect(translated[key], key).toBeTruthy();
        expect(translated[key]?.match(/{{\w+}}/g) || [], key).toEqual(
          english[key]?.match(/{{\w+}}/g) || [],
        );
      }
    });
});
