import { afterEach, expect, test, vi } from "vitest";
import { api } from "./client";
import { pairingApi } from "./pairing";

vi.mock("./client", () => ({ api: { get: vi.fn(), post: vi.fn() } }));
afterEach(() => vi.resetAllMocks());

const proof = {
  id: "AbCdEfGhIjKlMn012789",
  expires_at: "2026-11-01T12:00:00Z",
  different_client: false,
  evidence: [],
};
const created = {
  schema_version: 1,
  proof: { ...proof, token: "7".repeat(64) },
  startup: { listeners_ready: true, engine_ready: true },
  advertised_addresses: null,
  listener_addresses: ["127.0.0.1:8090"],
  auth_required: true,
};

test("pairing creation accepts the existing mixed-case alphanumeric bounds", async () => {
  vi.mocked(api.post).mockResolvedValue(created);
  expect(await pairingApi.create()).toEqual(created);
  expect(api.post).toHaveBeenCalledExactlyOnceWith(
    "/flow/pairing",
    {},
    undefined,
    true,
  );
});

test("pairing status preserves valid proof IDs and encodes the lookup", async () => {
  vi.mocked(api.get).mockResolvedValue(proof);
  expect(await pairingApi.status(proof.id)).toEqual(proof);
  expect(api.get).toHaveBeenCalledExactlyOnceWith(
    `/flow/pairing/status/${proof.id}`,
    undefined,
  );
});

test("pairing creation rejects non-ASCII, punctuation and out-of-bounds tokens", async () => {
  for (const token of [
    "a".repeat(19),
    "a".repeat(65),
    "_".repeat(20),
    "é".repeat(20),
  ]) {
    vi.mocked(api.post).mockResolvedValue({
      ...created,
      proof: { ...created.proof, token },
    });
    await expect(pairingApi.create()).rejects.toThrow();
  }
});

test("pairing status rejects malformed IDs without accepting a partial match", async () => {
  for (const id of [
    "a".repeat(19),
    "a".repeat(65),
    "a".repeat(20) + "\n",
    "a".repeat(19) + "/",
  ]) {
    vi.mocked(api.get).mockResolvedValue({ ...proof, id });
    await expect(pairingApi.status(proof.id)).rejects.toThrow();
  }
});
