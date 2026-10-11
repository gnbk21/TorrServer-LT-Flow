export function torrentMagnet(hash: string, title?: string): string {
  if (!/^(?:[a-f0-9]{40}|[a-f0-9]{64})$/i.test(hash)) {
    throw new Error("Invalid torrent identity");
  }
  const prefix = hash.length === 64 ? "urn:btmh:1220" : "urn:btih:";
  return `magnet:?xt=${prefix}${hash.toLowerCase()}${title ? `&dn=${encodeURIComponent(title)}` : ""}`;
}
