/** Diagnostics show destinations without URL credentials, query tokens or private paths. */
export function redactDiagnostic(value: unknown): string {
  const text = Array.isArray(value) ? value.join(", ") : String(value ?? "—");
  return text
    .replace(
      /(?:https?|udp|wss?|socks(?:4a?|5h?)|torrs):\/\/[^\s"'<>]+|magnet:\?[^\s"'<>]+/gi,
      (raw) => {
        try {
          const url = new URL(raw);
          if (!url.host) return "[redacted URL]";
          return (
            url.protocol +
            "//" +
            url.host +
            (url.pathname !== "/" || url.search || url.hash
              ? "/[redacted]"
              : "/")
          );
        } catch {
          return "[redacted URL]";
        }
      },
    )
    .replace(/\/flow\/play\/[^\s/?"']+/g, "/flow/play/[redacted]")
    .replace(
      /(authorization["']?\s*[:=]\s*)(?:bearer|basic)\s+[^\s&,;"']+/gi,
      "$1[redacted]",
    )
    .replace(
      /\b((?:probe_key|passkey|password|passwd|api[_-]?key|(?:access|session|client|bot|tg)[_-]?token|token|ss|authorization|secret|capability)["']?\s*[:=]\s*)(?:"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\s&;,"']+)/gi,
      "$1[redacted]",
    );
}
