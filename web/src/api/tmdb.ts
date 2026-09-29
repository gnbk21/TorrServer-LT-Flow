import type { TMDBSettings } from "../types/settings";
// TMDB is the inherited, explicit external integration. It has no server proxy
// endpoint. Keep its credentials out of errors, logs, QR links and API caches.
export async function searchPosters(
  config: TMDBSettings,
  title: string,
  language: string,
): Promise<string[]> {
  const base = new URL(config.APIURL || "https://api.themoviedb.org");
  if (
    !["http:", "https:"].includes(base.protocol) ||
    base.username ||
    base.password
  )
    throw new Error("Invalid TMDB configuration");
  base.pathname =
    base.pathname.replace(/\/(?:3|search)(?:\/.*)?$/, "").replace(/\/$/, "") +
    "/3/search/multi";
  base.search = new URLSearchParams({
    api_key: config.APIKey,
    language: language === "ua" ? "uk" : language,
    query: title.slice(0, 100),
  }).toString();
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 15000);
  try {
    const response = await fetch(base, {
      signal: controller.signal,
      credentials: "omit",
    });
    if (!response.ok) throw new Error("TMDB request failed");
    const data: unknown = await response.json();
    if (
      !data ||
      typeof data !== "object" ||
      !("results" in data) ||
      !Array.isArray(data.results)
    )
      throw new Error("Invalid TMDB response");
    const imageBase = new URL(
      language === "ru" && config.ImageURLRu
        ? config.ImageURLRu
        : config.ImageURL || "https://image.tmdb.org",
    );
    if (
      !["http:", "https:"].includes(imageBase.protocol) ||
      imageBase.username ||
      imageBase.password
    )
      throw new Error("Invalid artwork configuration");
    return data.results
      .filter(
        (item: unknown): item is { poster_path: string } =>
          !!item &&
          typeof item === "object" &&
          "poster_path" in item &&
          typeof item.poster_path === "string" &&
          /^\/[\w.-]+$/.test(item.poster_path),
      )
      .slice(0, 20)
      .map((item) => `${imageBase.origin}/t/p/w300${item.poster_path}`);
  } finally {
    clearTimeout(timer);
  }
}
