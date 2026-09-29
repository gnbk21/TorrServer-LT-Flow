import type { FlowSession } from "../types/flow";
export type FlowHealthState =
  "HEALTHY" | "MARGINAL" | "BUFFER_RISK" | "STALLED" | "WARM_IDLE" | "UNKNOWN";
export interface HealthDerivation {
  state: FlowHealthState;
  label: string;
  description: string;
}
export function deriveFlowHealth(session?: FlowSession): HealthDerivation {
  let state: FlowHealthState = "UNKNOWN";
  if (session?.state === "WARM_IDLE") state = "WARM_IDLE";
  else if (
    session &&
    session.playback_consumption_rate > 0 &&
    Number.isFinite(session.buffer_ahead_seconds)
  ) {
    // Present the server's warning and measured rate. Do not infer a stall from
    // zero throughput: the player may be paused or reading entirely from cache.
    if (session.state === "STALLED") state = "STALLED";
    else if (session.buffer_warning) state = "BUFFER_RISK";
    else if (
      session.sustainability_ratio < 1 &&
      session.buffer_exhaustion_seconds != null
    )
      state = "MARGINAL";
    else if (session.buffer_ahead_seconds > 0) state = "HEALTHY";
  }
  return {
    state,
    label: `flow.health.${state}`,
    description: `flow.healthDescription.${state}`,
  };
}
