// @vitest-environment node
import { storageSchema } from "./integrations";
import { settingsMetadataSchema } from "./settingsMetadata";
import { searchEnvelopeSchema } from "./search";
import { summariesSchema } from "./sessionHistory";
import { readFileSync } from "node:fs";
import { expect, test } from "vitest";
import {
  flowSchema,
  settingsSchema,
  torrentSchema,
  runtimeSchema,
  networkSchema,
  librarySchema,
  activeSchema,
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
test("bounded startup reserves retain older servers and reject impossible targets", () => {
  const reserve = {
    known: true,
    reason: "SUPPLY_DEFICIT",
    base_bytes: 32 * 1048576,
    target_bytes: 128 * 1048576,
    maximum_bytes: 128 * 1048576,
    limited: true,
    demand_rate: 12.5e6,
    delivery_rate: 10e6,
    deficit_rate: 2.5e6,
    forecast_seconds: 45,
  };
  const response = (value: unknown) => ({
    hash: "fixture",
    startup: { state: "ADAPTIVE_PREBUFFER", file_index: 1, reserve: value },
  });
  expect(flowSchema.safeParse({ hash: "fixture" }).success).toBe(true);
  expect(flowSchema.safeParse(response(reserve)).success).toBe(true);
  const bounded = {
    ...reserve,
    requested_bytes: reserve.maximum_bytes,
    extra_wait_budget_ms: 10000,
    extra_wait_limited: true,
    extra_wait_expired: true,
  };
  expect(flowSchema.parse(response(bounded)).startup?.reserve).toEqual(bounded);
  expect(
    flowSchema.safeParse(response({ ...bounded, extra_wait_budget_ms: -1 }))
      .success,
  ).toBe(false);
  expect(
    flowSchema.safeParse(response({ ...bounded, requested_bytes: 1 })).success,
  ).toBe(false);
  expect(
    flowSchema.safeParse(
      response({ ...reserve, target_bytes: reserve.maximum_bytes + 1 }),
    ).success,
  ).toBe(false);
  expect(
    flowSchema.safeParse(response({ ...reserve, demand_rate: Infinity }))
      .success,
  ).toBe(false);
});
test("native recovery and request telemetry reject oversized or inconsistent snapshots", () => {
  const zeros = Object.fromEntries(
    [
      "first_dial_ms",
      "first_connect_ms",
      "first_handshake_ms",
      "first_payload_ms",
      "connect_duration_ms",
      "handshake_duration_ms",
      "useful_bytes",
      "redundant_bytes",
      "obsolete_bytes",
      "recovery_requests",
      "recovery_suppressed",
      "active_hedges",
      "peak_hedges",
      "active_hedge_bytes",
      "budget_changes",
      "frontiers",
    ].map((key) => [key, 0]),
  );
  const base = {
    ...zeros,
    known: true,
    service: [],
    recovery_scan_truncated: false,
  };
  const policy = {
    enabled: true,
    sampled_peers: 2,
    qualified_peers: 1,
    capacity_limited_peers: 0,
    desired_requests: 16,
    queued_requests: 12,
    capacity_requests: 64,
    horizon_min_ms: 4000,
    horizon_max_ms: 4000,
    truncated: false,
  };
  const scan = {
    age_ms: 100,
    eligible: 1,
    uncertain_request: 1,
    stale_service: 0,
    already_duplicated: 0,
    not_late: 0,
    receiving: 0,
  };
  const response = (value: unknown) => ({
    hash: "fixture",
    sparse: { known: true, stream: value },
  });
  expect(flowSchema.safeParse(response(base)).success).toBe(true);
  expect(
    flowSchema.safeParse(
      response({ ...base, request_policy: policy, recovery_scan: scan }),
    ).success,
  ).toBe(true);
  expect(
    flowSchema.safeParse(
      response({
        ...base,
        request_policy: { ...policy, desired_requests: 65 },
      }),
    ).success,
  ).toBe(false);
  expect(
    flowSchema.safeParse(
      response({
        ...base,
        request_policy: { ...policy, horizon_max_ms: 6001 },
      }),
    ).success,
  ).toBe(false);
  expect(
    flowSchema.safeParse(
      response({ ...base, recovery_scan: { ...scan, not_late: 8192 } }),
    ).success,
  ).toBe(false);
});
test("storage diagnostics reject invalid counters while preserving new fields", () => {
  const storage = {
    queued_bytes: 0,
    peak_queue_bytes: 64,
    queued_jobs: 0,
    rejected_jobs: 0,
    completed_jobs: 1,
    wait_p95_us: 32,
    wait_max_us: 20,
    callback_p95_us: 128,
    callback_max_us: 100,
  };
  expect(
    flowSchema.safeParse({ hash: "fixture", storage_io: storage }).success,
  ).toBe(true);
  expect(
    flowSchema.safeParse({
      hash: "fixture",
      storage_io: { ...storage, queued_jobs: -1 },
    }).success,
  ).toBe(false);
  expect(
    flowSchema.safeParse({
      hash: "fixture",
      storage_io: { ...storage, wait_max_us: "unknown" },
    }).success,
  ).toBe(false);
});
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
    oldest_request_age_ms: -1,
  };
  const response = (urgent: unknown) => ({
    hash: "fixture",
    sparse: { known: true, urgent },
  });
  expect(flowSchema.safeParse(response([piece])).success).toBe(true);
  expect(
    flowSchema.safeParse(response([{ ...piece, oldest_request_age_ms: 250 }]))
      .success,
  ).toBe(true);
  expect(
    flowSchema.safeParse(response([{ ...piece, oldest_request_age_ms: -2 }]))
      .success,
  ).toBe(false);
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
      search: searchEnvelopeSchema,
      sessions: summariesSchema,
      storage: storageSchema,
      settings: settingsSchema,
      "settings-metadata": settingsMetadataSchema,
      torrent: torrentSchema,
      runtime: runtimeSchema,
      network: networkSchema,
      preparation: preparationSchema,
      sources: sourcesSchema,
      library: librarySchema,
      active: activeSchema,
    };
    const requiredFields = {
      flow: "hash",
      search: "items",
      sessions: "enabled",
      storage: "settings",
      settings: "CacheSize",
      "settings-metadata": "schema_version",
      torrent: "hash",
      runtime: "dlna_enabled",
      network: "state",
      preparation: "jobs",
      sources: "sources",
      library: "items",
      active: "items",
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
