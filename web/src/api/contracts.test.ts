// @vitest-environment node
import { readFileSync } from "node:fs";
import { expect, test } from "vitest";
import {
  flowSchema,
  settingsSchema,
  torrentSchema,
  runtimeSchema,
  networkSchema,
} from "./schemas";
import { validateSettings } from "../lib/settings";
import { preparationSchema } from "./preparation";
import { sourcesSchema } from "./sources";

test("legacy zero cache is readable while Apply requires a positive budget", () => {
  const legacy = {
    CacheSize: 0,
    ReaderReadAHead: 5,
    PreloadCache: 0,
    LegacyOption: true,
  };
  expect(settingsSchema.parse(legacy)).toEqual(legacy);
  expect(settingsSchema.safeParse({ ...legacy, CacheSize: -1 }).success).toBe(
    false,
  );
  expect(validateSettings({ CacheSize: 0 }).CacheSize).toBeDefined();
  expect(
    validateSettings({ CacheSize: 64 * 1048576 }).CacheSize,
  ).toBeUndefined();
});

const input = process.env.FLOW_CONTRACT_INPUT;
test("urgent frontier bounds and block accounting are validated", () => {
  const piece = {
    piece: 1,
    priority: 7,
    blocks: 256,
    unrequested: 100,
    requested: 120,
    writing: 6,
    finished: 30,
    duplicate_requests: 0,
    verified: false,
    receiving_blocks: 1,
    receiving_bytes: 8192,
  };
  const response = (urgent: unknown) => ({
    hash: "fixture",
    sparse: { known: true, urgent },
  });
  expect(flowSchema.safeParse(response([piece])).success).toBe(true);
  expect(
    flowSchema.safeParse(response([{ ...piece, unrequested: 101 }])).success,
  ).toBe(false);
  expect(flowSchema.safeParse(response(Array(65).fill(piece))).success).toBe(
    false,
  );
  expect(
    flowSchema.safeParse(response(Array(64).fill({ ...piece, blocks: 256 })))
      .success,
  ).toBe(false);
});
test.skipIf(!input)(
  "actual Go DTOs satisfy frontend validators and reject incompatible field types",
  () => {
    const cases = JSON.parse(readFileSync(input!, "utf8")) as {
      kind: string;
      payload: Record<string, unknown>;
    }[];
    const validators = {
      flow: flowSchema,
      settings: settingsSchema,
      torrent: torrentSchema,
      runtime: runtimeSchema,
      network: networkSchema,
      preparation: preparationSchema,
      sources: sourcesSchema,
    };
    const requiredFields = {
      flow: "hash",
      settings: "CacheSize",
      torrent: "hash",
      runtime: "dlna_enabled",
      network: "state",
      preparation: "jobs",
      sources: "sources",
    };
    expect(cases.length).toBeGreaterThanOrEqual(7);
    for (const { kind, payload } of cases) {
      const key = kind as keyof typeof validators;
      expect(validators[key].safeParse(payload).success, kind).toBe(true);
      const incompatible = {
        ...payload,
        [requiredFields[key]]: { invalid: true },
      };
      expect(
        validators[key].safeParse(incompatible).success,
        `${kind} invalid type`,
      ).toBe(false);
    }
  },
);
