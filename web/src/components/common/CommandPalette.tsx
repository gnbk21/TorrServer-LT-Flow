import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Modal } from "./Modal";
import { Button } from "./Button";
export interface Command {
  id: string;
  label: string;
  run: () => void;
}
export function CommandPalette({
  open,
  onClose,
  commands,
}: {
  open: boolean;
  onClose: () => void;
  commands: Command[];
}) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const pending = useRef<Command | undefined>(undefined);
  useEffect(() => {
    if (open) setQuery("");
  }, [open]);
  const fold = (s: string) =>
    s.normalize("NFKD").replace(/\p{M}/gu, "").toLocaleLowerCase();
  const matches = commands.filter((c) =>
    fold(`${c.label} ${c.id}`).includes(fold(query.slice(0, 128))),
  );
  const activate = (command: Command) => {
    pending.current = command;
    onClose();
  };
  return (
    <Modal
      isOpen={open}
      onClose={onClose}
      title={t("app.expressive.commands")}
      onAfterClose={() => {
        const command = pending.current;
        pending.current = undefined;
        // Focus return and Radix teardown precede the next workflow.
        if (command) window.setTimeout(command.run, 0);
      }}
    >
      <label className="field">
        {t("Search")}
        <input
          value={query}
          maxLength={128}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && matches.length === 1) {
              activate(matches[0]!);
            }
          }}
        />
      </label>
      <div className="flow-command-results mt-4">
        {matches.map((c) => (
          <Button
            key={c.id}
            variant="ghost"
            onClick={() => {
              activate(c);
            }}
          >
            {c.label}
          </Button>
        ))}
      </div>
      {!matches.length && <p role="status">{t("app.expressive.noCommands")}</p>}
    </Modal>
  );
}
