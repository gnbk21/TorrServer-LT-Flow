// Never move a URL to another host or copy credentials into an external origin.
export function safeMediaBase(
  value: string,
  origin = window.location.origin,
): string {
  const base = new URL(value);
  const current = new URL(origin);
  if (
    !["http:", "https:"].includes(base.protocol) ||
    base.hostname !== current.hostname ||
    base.username ||
    base.password ||
    base.pathname !== "/" ||
    base.search ||
    base.hash
  )
    throw new Error("Invalid external media origin");
  return base.origin;
}

export function externalMediaURL(
  value: string,
  base?: string,
  origin = window.location.origin,
): string {
  const url = new URL(value, origin);
  if (!base || url.origin !== origin || url.username || url.password)
    return url.href;
  const target = new URL(safeMediaBase(base, origin));
  url.protocol = target.protocol;
  url.hostname = target.hostname;
  url.port = target.port;
  return url.href;
}
