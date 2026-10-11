import { useState, useEffect } from "react";

// Local preferences contain no credentials or server settings. Storage can be
// unavailable in private browsing; the interface must still work in memory.
export function usePreference<T>(
  key: string,
  fallback: T,
  validate: (value: unknown) => value is T,
) {
  const [value, setValue] = useState<T>(() => {
    try {
      const stored: unknown = JSON.parse(
        localStorage.getItem(`flow.ui.${key}`) || "null",
      );
      return validate(stored) ? stored : fallback;
    } catch {
      return fallback;
    }
  });
  useEffect(() => {
    try {
      localStorage.setItem(`flow.ui.${key}`, JSON.stringify(value));
    } catch {
      /* preferences remain usable in memory */
    }
  }, [key, value]);
  return [value, setValue] as const;
}
export const isString = (value: unknown): value is string =>
  typeof value === "string" && value.length <= 1024;
export const isPage = (value: unknown): value is number =>
  typeof value === "number" &&
  Number.isInteger(value) &&
  value > 0 &&
  value <= 1_000_000;
