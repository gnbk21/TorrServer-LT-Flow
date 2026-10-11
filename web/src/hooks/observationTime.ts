import { useEffect, useReducer } from "react";
import { useVisible } from "./queries";

// Expire displayed observations even when the server stops replying. Schedule
// only the next deadline; closed/hidden details own no timers or network work.
export function useObservationTime(
  enabled: boolean,
  sampleDeadline: number,
  sparseDeadline: number,
) {
  const visible = useVisible();
  const [revision, refresh] = useReducer((value: number) => value + 1, 0);
  useEffect(() => {
    if (!enabled || !visible) return;
    const now = Date.now();
    const future = [sampleDeadline, sparseDeadline].filter(
      (deadline) =>
        Number.isFinite(deadline) && deadline > now && deadline - now <= 10_000,
    );
    if (!future.length) return;
    const timer = setTimeout(refresh, Math.min(...future) - now + 1);
    return () => clearTimeout(timer);
  }, [enabled, visible, sampleDeadline, sparseDeadline, revision]);
  return enabled && visible ? Date.now() : Infinity;
}
