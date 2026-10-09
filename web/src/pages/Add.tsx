import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useSettings, refreshLibrary } from "../hooks/queries";
import { SearchPanel } from "../components/search/SearchPanel";
import { AddTorrentModal } from "../components/torrent/AddTorrentModal";
import { Button } from "../components/common/Button";
import { RequestError, Loading } from "../components/common/RequestState";
export default function Add() {
  const { t } = useTranslation();
  const settings = useSettings();
  const [open, setOpen] = useState(false);
  return (
    <div className="space-y-5">
      <div className="actions">
        <h1 className="text-2xl font-semibold mr-auto">{t("nav.add")}</h1>
        <Button onClick={() => setOpen(true)}>{t("Add")}</Button>
      </div>
      {settings.isPending ? (
        <Loading />
      ) : settings.error ? (
        <RequestError error={settings.error} retry={() => settings.refetch()} />
      ) : (
        <SearchPanel
          onTorrentAdded={() => void refreshLibrary()}
          enabledSources={{
            rutor: settings.data?.EnableRutorSearch,
            torznab:
              settings.data?.EnableTorznabSearch &&
              !!settings.data.TorznabUrls?.length,
            jacred:
              settings.data?.EnableJacRedSearch && !!settings.data.JacRedUrl,
          }}
        />
      )}
      {open && (
        <AddTorrentModal
          isOpen
          onClose={() => setOpen(false)}
          onSuccess={() => void refreshLibrary()}
        />
      )}
    </div>
  );
}
