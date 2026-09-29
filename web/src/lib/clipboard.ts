// Clipboard API is unavailable on ordinary LAN HTTP. Preserve a selectable
// fallback and report failure to the caller instead of pretending it copied.
export async function copyText(text: string): Promise<void> {
  try {
    if (navigator.clipboard) {
      await navigator.clipboard.writeText(text);
      return;
    }
  } catch {
    /* fall back for LAN HTTP */
  }
  const element = document.createElement("textarea");
  element.value = text;
  element.style.position = "fixed";
  element.style.opacity = "0";
  const previous = document.activeElement;
  document.body.append(element);
  element.select();
  try {
    if (!document.execCommand("copy")) throw new Error("pairing.copyFailed");
  } finally {
    element.remove();
    if (previous instanceof HTMLElement) previous.focus();
  }
}
