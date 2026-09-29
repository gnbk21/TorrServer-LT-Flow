export function isAndroidDevice(): boolean {
  return /Android/i.test(navigator.userAgent);
}
export function isAppleDevice(): boolean {
  return /iPhone|iPad|iPod|Macintosh/i.test(navigator.userAgent);
}
export interface IntentUrls {
  justPlayerIntent?: string;
  vlcIntent?: string;
  webStreamUrl: string;
}
export function safeStreamUrl(value: string): URL {
  const url = new URL(value);
  if (
    !["http:", "https:"].includes(url.protocol) ||
    url.username ||
    url.password
  )
    throw new Error("Expected an HTTP(S) URL without credentials");
  url.hash = "";
  return url;
}
export function buildAndroidIntent(
  streamUrl: string,
  player: "just" | "vlc" = "just",
): string {
  const url = safeStreamUrl(streamUrl);
  const data = url.href.slice(url.protocol.length + 2).replace(/;/g, "%3B");
  return `intent://${data}#Intent;scheme=${url.protocol.slice(0, -1)};package=${player === "just" ? "com.brouken.player" : "org.videolan.vlc"};type=video/*;S.browser_fallback_url=${encodeURIComponent(url.href)};end`;
}
export function buildIntentUrls(streamUrl: string): IntentUrls {
  const url = safeStreamUrl(streamUrl).href;
  return {
    webStreamUrl: url,
    justPlayerIntent: isAndroidDevice() ? buildAndroidIntent(url) : undefined,
    vlcIntent: isAndroidDevice() ? buildAndroidIntent(url, "vlc") : undefined,
  };
}
