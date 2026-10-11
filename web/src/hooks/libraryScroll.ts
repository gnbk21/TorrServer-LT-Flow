import { useEffect, useRef } from "react";

// Restore after rows have been laid out. Until then a browser scroll event must
// not overwrite the saved offset with the height of the loading screen.
export function useLibraryScroll(ready: boolean) {
  const restored = useRef(false);
  useEffect(() => {
    if (!ready || restored.current) return;
    const frame = requestAnimationFrame(() => {
      let offset = 0;
      try {
        const stored = Number(sessionStorage.getItem("flow.library.scroll"));
        if (Number.isFinite(stored)) offset = stored;
      } catch {
        /* optional local preference */
      }
      restored.current = true;
      window.scrollTo(
        0,
        Math.max(
          0,
          Math.min(
            offset,
            document.documentElement.scrollHeight - window.innerHeight,
          ),
        ),
      );
    });
    // Marking restoration complete in the frame also survives StrictMode's
    // setup/cleanup/setup sequence and a cancelled initial render.
    return () => cancelAnimationFrame(frame);
  }, [ready]);
  useEffect(() => {
    const save = () => {
      if (!restored.current) return;
      try {
        sessionStorage.setItem("flow.library.scroll", String(window.scrollY));
      } catch {
        /* optional local preference */
      }
    };
    window.addEventListener("scroll", save, { passive: true });
    return () => window.removeEventListener("scroll", save);
  }, []);
}
