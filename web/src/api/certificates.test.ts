import { afterEach, expect, test, vi } from "vitest";
import { certificatesApi, certificateStatusSchema } from "./certificates";

const fixture = {
  enabled: true,
  http_port: "8090",
  port: "8091",
  http_enabled: true,
  force_https: false,
  http_media: false,
  read_only: false,
  cert_from_flags: false,
  revision: "current",
  cert: { source: "self-signed", trusted: false },
};
afterEach(() => vi.unstubAllGlobals());
test("certificate status validates its contract and removes unexpected key material", () => {
  expect(
    certificateStatusSchema.parse({
      ...fixture,
      cert: { ...fixture.cert, private_key: "secret" },
    }).cert,
  ).not.toHaveProperty("private_key");
  expect(
    certificateStatusSchema.safeParse({
      ...fixture,
      cert: { source: "unknown" },
    }).success,
  ).toBe(false);
});
test("certificate mutation carries the draft revision and uses multipart for uploads", async () => {
  const fetch = vi.fn().mockResolvedValue(
    new Response(JSON.stringify(fixture), {
      headers: { "content-type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetch);
  const body = new FormData();
  body.set("cert", new Blob(["certificate"]), "certificate.pem");
  body.set("key", new Blob(["key"]), "key.pem");
  await certificatesApi.change("upload", "current", body);
  const request = fetch.mock.calls[0]![1];
  expect(request.headers.get("If-Match")).toBe("current");
  expect(request.headers.has("Content-Type")).toBe(false);
  expect(request.body).toBe(body);
});
