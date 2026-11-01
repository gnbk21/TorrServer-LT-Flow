import type { FlowStatusResponse } from "../types/flow";

export function playbackOverview(status: FlowStatusResponse, now: number) {
  const age = now - Date.parse(status.sampled_at || "");
  const session =
    status.sessions?.find((s) => s.active_readers > 0) ?? status.sessions?.[0];
  const fresh = Number.isFinite(age) && age >= 0 && age < 10_000;
  const bufferAge = now - (session?.buffer_sampled_at_ms ?? 0);
  const bufferFresh =
    fresh &&
    !!session?.buffer_sampled_at_ms &&
    bufferAge >= 0 &&
    bufferAge < 10000;
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
      cause = bufferFresh ? "ready" : "unknown";
      action = bufferFresh ? "leaveOpen" : "inspect";
    } else if (
      (status.storage_io?.wait_p95_us ?? 0) > 250000 &&
      (status.storage_io?.queued_jobs ?? 0) > 0 &&
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
  } else if (
    status.startup?.progress &&
    status.startup.progress.blocked_on !== "NONE"
  ) {
    cause = "block";
    action = "waitBlock";
  } else if (status.startup?.state === "ADAPTIVE_PREBUFFER") {
    cause = "reserve";
    action = "waitBlock";
  } else if (
    status.startup?.state === "READY" ||
    status.startup?.state === "PLAYING"
  ) {
    cause = "phone";
    action = "setupPhone";
  }
  const confidence =
    !fresh || cause === "unknown"
      ? "unknown"
      : ["metadata", "idle", "seek", "ready", "reserve"].includes(cause)
        ? "high"
        : "medium";
  const adviser = !fresh
    ? "inspect"
    : cause === "storage"
      ? "inspectStorage"
      : cause === "phone"
        ? "setupPhone"
        : session?.buffer_warning
          ? "pausePlayer"
          : action;
  const startup = status.startup;
  const steps = [
    { key: "metadata", complete: (startup?.metadata_ready_ms ?? -1) >= 0 },
    {
      key: "connection",
      complete:
        (startup?.first_peer_ms ?? -1) >= 0 ||
        (session?.connected_peers ?? 0) > 0,
    },
    {
      key: "required",
      complete:
        ["READY", "PLAYING"].includes(startup?.state ?? "") ||
        ((startup?.bootstrap_complete_ms ?? 0) > 0 &&
          !["INDEX", "HEAD_AND_INDEX"].includes(
            startup?.progress?.blocked_on ?? "",
          )),
    },
    {
      key: "reserve",
      complete:
        (startup?.startup_prebuffer_ms ?? 0) > 0 ||
        ["READY", "PLAYING"].includes(startup?.state ?? ""),
    },
    {
      key: "http",
      complete:
        (session?.range_request_count ?? 0) > 0 &&
        (session?.last_ttfb_ms ?? -1) >= 0,
    },
  ];
  return {
    cause,
    action,
    age: Number.isFinite(age) && age >= 0 ? age : null,
    fresh,
    session,
    bufferFresh,
    confidence,
    adviser,
    steps,
  };
}
