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
  ]) {
    expect(redactDiagnostic(value)).not.toContain("secret");
  }
});
