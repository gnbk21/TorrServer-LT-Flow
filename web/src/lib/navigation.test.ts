import { expect, it, vi } from "vitest";
import type { MouseEvent } from "react";
import { immediateNavigation } from "./navigation";

it("commits cold route feedback while preserving modified and cancelled links", () => {
  for (const modifiers of [
    {},
    { defaultPrevented: true },
    { button: 1 },
    { metaKey: true },
    { ctrlKey: true },
    { shiftKey: true },
    { altKey: true },
  ]) {
    const navigate = vi.fn();
    const preventDefault = vi.fn();
    const event = {
      defaultPrevented: false,
      button: 0,
      ...modifiers,
      preventDefault,
    } as unknown as MouseEvent<HTMLAnchorElement>;
    immediateNavigation(event, "/torrents", navigate);
    if (!Object.keys(modifiers).length) {
      expect(preventDefault).toHaveBeenCalledOnce();
      expect(navigate).toHaveBeenCalledWith("/torrents", { flushSync: true });
    } else {
      expect(preventDefault).not.toHaveBeenCalled();
      expect(navigate).not.toHaveBeenCalled();
    }
  }
});
