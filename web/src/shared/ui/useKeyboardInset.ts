import { useEffect } from "react";

/**
 * Keeps a CSS variable (--kb-inset) in sync with the on-screen keyboard height
 * using the VisualViewport API, so bottom-anchored UI (mobile nav, chat
 * composer) can lift above the keyboard instead of being hidden behind it.
 *
 * No-op on browsers without VisualViewport. Mounts once at the app shell.
 */
export function useKeyboardInset() {
  useEffect(() => {
    const vv = window.visualViewport;
    if (!vv) return;

    const root = document.documentElement;
    const update = () => {
      // Space between the visual viewport bottom and the layout viewport
      // bottom is the keyboard (plus any browser chrome) overlap.
      const inset = Math.max(
        0,
        window.innerHeight - vv.height - vv.offsetTop,
      );
      root.style.setProperty("--kb-inset", `${Math.round(inset)}px`);
    };

    update();
    vv.addEventListener("resize", update);
    vv.addEventListener("scroll", update);
    return () => {
      vv.removeEventListener("resize", update);
      vv.removeEventListener("scroll", update);
      root.style.removeProperty("--kb-inset");
    };
  }, []);
}
