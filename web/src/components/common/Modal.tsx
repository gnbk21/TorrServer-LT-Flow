import * as Dialog from "@radix-ui/react-dialog";
import { useRef, type ReactNode } from "react";
import { X } from "lucide-react";
import { useTranslation } from "react-i18next";
export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  subtitle?: string;
  children: ReactNode;
  maxWidth?: "sm" | "md" | "lg" | "xl" | "2xl" | "4xl";
  showCloseButton?: boolean;
  returnFocus?: HTMLElement | null;
}
export function Modal({
  isOpen,
  onClose,
  title,
  subtitle,
  children,
  maxWidth = "lg",
  showCloseButton = true,
  returnFocus,
}: ModalProps) {
  const { t } = useTranslation();
  const opener = useRef<HTMLElement | null>(null);
  const widths = {
    sm: "sm:max-w-sm",
    md: "sm:max-w-md",
    lg: "sm:max-w-lg",
    xl: "sm:max-w-xl",
    "2xl": "sm:max-w-2xl",
    "4xl": "sm:max-w-4xl",
  };
  return (
    <Dialog.Root
      open={isOpen}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/75" />
        <Dialog.Content
          onOpenAutoFocus={() => {
            // These controlled dialogs have no Radix Trigger. Remember the
            // actual opener before FocusScope moves focus into the content.
            const active = returnFocus ?? document.activeElement;
            opener.current =
              active instanceof HTMLElement &&
              active !== document.body &&
              active !== document.documentElement
                ? active
                : null;
          }}
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            const previous = opener.current;
            opener.current = null;
            const eligible =
              previous?.isConnected &&
              !previous.matches(":disabled") &&
              previous.getAttribute("aria-disabled") !== "true" &&
              !previous.closest("[inert]");
            if (eligible) {
              previous.focus({ preventScroll: true });
            }
            if (!eligible || document.activeElement !== previous) {
              // A route change can remove the row that opened the dialog.
              document.getElementById("main")?.focus({ preventScroll: true });
            }
          }}
          className={`fixed z-50 inset-0 sm:inset-auto sm:top-1/2 sm:left-1/2 sm:-translate-x-1/2 sm:-translate-y-1/2 w-full ${widths[maxWidth]} sm:max-h-[90dvh] flex flex-col bg-slate-900 sm:rounded-2xl border border-slate-700 shadow-xl`}
        >
          <header className="flex justify-between gap-3 p-5 border-b border-slate-700">
            <div>
              <Dialog.Title className="font-semibold text-lg">
                {title}
              </Dialog.Title>
              <Dialog.Description
                className={subtitle ? "text-sm text-slate-400" : "sr-only"}
              >
                {subtitle || title}
              </Dialog.Description>
            </div>
            {showCloseButton && (
              <Dialog.Close
                className="min-w-11 rounded-lg"
                aria-label={t("Close")}
              >
                <X className="mx-auto" />
              </Dialog.Close>
            )}
          </header>
          <div className="p-5 overflow-auto min-h-0">{children}</div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
