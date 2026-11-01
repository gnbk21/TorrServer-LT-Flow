import { afterEach, expect, test, vi } from "vitest";
import { getSettingsMetadata, type SettingsMetadata } from "./settingsMetadata";
import { validateSettings } from "../lib/settings";
import { resetManagementCooldown } from "./client";
afterEach(() => {
  vi.unstubAllGlobals();
  resetManagementCooldown();
});
test("only absent endpoint uses old-server constraints", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(() => Promise.resolve(new Response("missing", { status: 404 }))),
  );
  expect(await getSettingsMetadata()).toBeNull();
  vi.stubGlobal(
    "fetch",
    vi.fn(() => Promise.resolve(new Response("unauthorized", { status: 401 }))),
  );
  await expect(getSettingsMetadata()).rejects.toMatchObject({ status: 401 });
});
test("server metadata owns scalar bounds and discrete choices", () => {
  const metadata: SettingsMetadata = {
    schema_version: 1,
    fields: {
      "Flow.StreamTransportBufferKiB": {
        type: "integer",
        default: 1024,
        min: 0,
        max: 1024,
        choices: [0, 64, 256, 1024],
        restart_required: false,
        experimental: false,
        deprecated: false,
        read_only: false,
      },
    },
  };
  expect(
    validateSettings({ "Flow.StreamTransportBufferKiB": 65 }, metadata),
  ).toHaveProperty("Flow.StreamTransportBufferKiB");
  expect(
    validateSettings({ "Flow.StreamTransportBufferKiB": 64 }, metadata),
  ).toEqual({});
});
