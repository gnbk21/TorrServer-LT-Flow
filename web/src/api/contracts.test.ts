// @vitest-environment node
import { readFileSync } from "node:fs";
import { expect, test } from "vitest";
import { flowSchema, settingsSchema, torrentSchema, runtimeSchema, networkSchema } from "./schemas";

const input = process.env.FLOW_CONTRACT_INPUT;
test.skipIf(!input)("actual Go DTOs satisfy frontend validators and reject incompatible field types", () => {
  const cases = JSON.parse(readFileSync(input!, "utf8")) as { kind: string; payload: Record<string, unknown> }[];
  const validators = { flow: flowSchema, settings: settingsSchema, torrent: torrentSchema, runtime: runtimeSchema, network: networkSchema };
  const requiredFields = { flow: "hash", settings: "CacheSize", torrent: "hash", runtime: "dlna_enabled", network: "state" };
  expect(cases.length).toBeGreaterThanOrEqual(7);
  for (const { kind, payload } of cases) {
    const key = kind as keyof typeof validators;
    expect(validators[key].safeParse(payload).success, kind).toBe(true);
    const incompatible = { ...payload, [requiredFields[key]]: { invalid: true } };
    expect(validators[key].safeParse(incompatible).success, `${kind} invalid type`).toBe(false);
  }
});
