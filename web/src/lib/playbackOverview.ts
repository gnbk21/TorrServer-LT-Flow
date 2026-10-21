import type { FlowStatusResponse } from "../types/flow";

export function playbackOverview(status: FlowStatusResponse, now: number) {
  const age = now - Date.parse(status.sampled_at || "");
  const session =
    status.sessions?.find((s) => s.active_readers > 0) ?? status.sessions?.[0];
  const fresh = Number.isFinite(age) && age >= 0 && age < 10_000;
  let cause = "unknown";
  let action = "inspect";
  if (!fresh) cause = "stale";
  else if (status.startup?.state === "METADATA") {
    cause = "metadata";
    action = "waitMetadata";
  } else if (session?.state === "WARM_IDLE") {
    cause = "idle";
    action = "leaveOpen";
  } else if (session && session.active_readers > 0) {
    if (session.state === "SEEK_RECOVERY") {
      cause = "seek";
      action = "waitBlock";
    } else if (session.buffer_ahead_bytes > 0) {
      cause = "ready";
      action = "leaveOpen";
    } else if (
      (status.storage_io?.wait_p95_us ?? 0) > 250000 &&
      (session.recent_server_read_stalls ?? 0) > 0
    ) {
      cause = "storage";
    } else if (
      ["medium", "high"].includes(session.wait_explanation?.confidence ?? "") &&
      session.required_piece_suppliers === 0 &&
      now - (status.sparse?.sampled_at_ms ?? 0) < 5000 &&
      now >= (status.sparse?.sampled_at_ms ?? 0)
    ) {
      cause = "missing";
      action = "waitSuppliers";
    } else if (
      session.wait_reason === "DISCOVERY" ||
      session.wait_reason === "CONNECTION"
    ) {
      cause = "discovery";
      action = "waitSuppliers";
    } else {
      cause = "block";
      action = "waitBlock";
    }
  }
  const bufferAge = now - (session?.buffer_sampled_at_ms ?? 0);
  const bufferFresh =
    fresh &&
    !!session?.buffer_sampled_at_ms &&
    bufferAge >= 0 &&
    bufferAge < 10000;
  return { cause, action, age: fresh ? age : null, session, bufferFresh };
}
