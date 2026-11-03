import type { MouseEvent } from "react";
import type { NavigateFunction } from "react-router-dom";

// Commit route/loading feedback before a cold lazy destination finishes. Keep
// ordinary browser behavior for modified clicks, new windows and cancelled UI.
export function immediateNavigation(
  event: MouseEvent<HTMLAnchorElement>,
  to: string,
  navigate: NavigateFunction,
) {
  if (
    event.defaultPrevented ||
    event.button !== 0 ||
    event.metaKey ||
    event.ctrlKey ||
    event.shiftKey ||
    event.altKey
  )
    return;
  event.preventDefault();
  void navigate(to, { flushSync: true });
}
