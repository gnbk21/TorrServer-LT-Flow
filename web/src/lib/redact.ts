/** Diagnostics show destinations without URL credentials, query tokens or private paths. */
export function redactDiagnostic(value: unknown): string {
  const text = Array.isArray(value) ? value.join(", ") : String(value ?? "—");
  return text
    .replace(/(?:https?|udp|wss?):\/\/[^\s"'<>]+/gi, (raw) => {
      try {
        const url = new URL(raw);
        return (
          url.protocol +
          "//" +
          url.host +
          (url.pathname !== "/" || url.search || url.hash ? "/[redacted]" : "/")
        );
      } catch {
        return "[redacted URL]";
      }
    })
    .replace(
      /\b(passkey|password|api[_-]?key|token|authorization)\s*[:=]\s*[^\s,;]+/gi,
      "$1=[redacted]",
    );
}
