export interface RankedAddress {
  ip: string;
  url: string;
  label: string;
  isRecommended: boolean;
  isLoopback: boolean;
}
export function isLoopback(host: string): boolean {
  return /^(localhost|127(?:\.\d+){3}|::1|\[::1\])$/i.test(host);
}
export function safePairingUrl(value: string): string {
  const url = new URL(value);
  if (
    !["http:", "https:"].includes(url.protocol) ||
    url.username ||
    url.password ||
    isLoopback(url.hostname) ||
    url.hostname === "0.0.0.0" ||
    url.hostname === "[::]"
  )
    throw new Error("pairing.invalidAddress");
  if (url.search || url.hash || url.pathname !== "/")
    throw new Error("pairing.invalidAddress");
  return url.origin;
}
export function rankServerAddresses(
  serverIps: string[] = [],
  currentPort?: string,
  origin = window.location.origin,
): RankedAddress[] {
  const current = new URL(origin);
  const port = currentPort ?? current.port;
  const hosts = [
    ...new Set(
      [current.hostname, ...serverIps].map((host) =>
        host.replace(/^\[|\]$/g, ""),
      ),
    ),
  ];
  return hosts
    .map((ip) => {
      const loopback = isLoopback(ip);
      const privateAddress =
        /^(10\.|192\.168\.|172\.(1[6-9]|2\d|3[01])\.|f[cd])/i.test(ip);
      const currentMatch = ip === current.hostname.replace(/^\[|\]$/g, "");
      const score = loopback
        ? 0
        : currentMatch
          ? 100
          : privateAddress
            ? 70
            : 30;
      const authority = ip.includes(":") ? `[${ip}]` : ip;
      return {
        ip,
        url: `${current.protocol}//${authority}${port ? `:${port}` : ""}`,
        label: loopback
          ? "pairing.loopback"
          : currentMatch
            ? "pairing.current"
            : privateAddress
              ? "pairing.private"
              : "pairing.other",
        isRecommended: false,
        isLoopback: loopback,
        score,
      };
    })
    .sort((a, b) => b.score - a.score)
    .map((address, index) => ({
      ...address,
      isRecommended: index === 0 && !address.isLoopback,
    }));
}
