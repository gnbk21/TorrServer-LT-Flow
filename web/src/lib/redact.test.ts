import { it, expect } from "vitest";
import { redactDiagnostic } from "./redact";
it("removes credentials and passkeys from diagnostic URLs and named fields", () => {
  const result = redactDiagnostic(
    "failed https://alice:secret@tracker.test/private/passkey?token=hidden token=other",
  );
  expect(result).toBe(
    "failed https://tracker.test/[redacted] token=[redacted]",
  );
});
it("retains numeric telemetry and non-sensitive state", () =>
  expect(redactDiagnostic(120)).toBe("120"));
it("redacts quoted fields, authorization values and opaque playback paths", () => {
  for (const value of [
    '{"password":"secret with spaces"}',
    "Authorization: Bearer secret",
    "probe_key=secret",
    "/flow/play/secret",
    "magnet:?xt=urn:btih:secret",
    "socks5h://alice:secret@proxy.test:1080",
    "wss://alice:secret@tracker.test/private-secret",
    "ss=secret TGToken=secret bot_token=secret session_token=secret client_token=secret",
  ]) {
    expect(redactDiagnostic(value)).not.toContain("secret");
  }
});
it("does not confuse harmless field suffixes with session tokens", () => {
  expect(redactDiagnostic("address=192.0.2.1 progress=20")).toBe(
    "address=192.0.2.1 progress=20",
  );
});
