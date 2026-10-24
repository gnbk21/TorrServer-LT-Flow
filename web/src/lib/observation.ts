import type { FlowStatusResponse } from "../types/flow";

export interface Observation {
  serverNow: number;
  receivedAt: number;
}
export interface Observed {
  observation?: Observation;
}

// Keep relative server sample times in the server's clock domain. Only elapsed
// receipt time uses the browser clock, which is monotonic across wall-clock edits.
export function observationNow(value?: Observed): number {
  const observation = value?.observation;
  return observation
    ? observation.serverNow +
        Math.max(0, performance.now() - observation.receivedAt)
    : Date.now();
}

export function observeFlow(
  status: FlowStatusResponse,
  anchor?: string,
  requestElapsed = 0,
): FlowStatusResponse {
  const serverNow = Date.parse(status.sampled_at ?? anchor ?? "");
  // Unknown server time cannot establish freshness on a clock-skewed client.
  const observation = {
    serverNow: serverNow + Math.max(0, requestElapsed),
    receivedAt: performance.now(),
  };
  return {
    ...status,
    sampled_at: status.sampled_at ?? anchor,
    observation,
    sessions: status.sessions?.map((session) => ({
      ...session,
      buffer_sampled_at_ms:
        session.buffer_sample_age_ms !== undefined && Number.isFinite(serverNow)
          ? serverNow - session.buffer_sample_age_ms
          : session.buffer_sampled_at_ms,
      observation,
    })),
  };
}
