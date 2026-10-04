import { expect, test } from "vitest";
import { preparationSchema } from "./preparation";
test("preparation does not claim readiness from partial or inconsistent data", () => {
  const job = { id: "job", hash: "0".repeat(40), file_index: 1, state: "downloading", length: 100, verified_bytes: 60, contiguous_bytes: 40, playback_ready: false };
  const state = { jobs: [job], quota_bytes: 4096, reserved_bytes: 100 };
  expect(preparationSchema.safeParse(state).success).toBe(true);
  for (const invalid of [{ playback_ready: true }, { contiguous_bytes: 70 }, { verified_bytes: 101 }, { state: "made-up" }, { file_index: 0 }])
    expect(preparationSchema.safeParse({ ...state, jobs: [{ ...job, ...invalid }] }).success).toBe(false);
});
