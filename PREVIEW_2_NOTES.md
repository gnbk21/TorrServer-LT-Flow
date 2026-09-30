# TorrServer-Flow Preview 2

Preview 2 adds playback hardening, maintenance tools and a more efficient modern
interface. Stable Flow / Modern Web 1.0 acceptance remains incomplete.

## Changes

- Rolling health measurements, adaptive-window hysteresis, bounded metadata probe
  reuse and explicit process/cache memory accounting.
- Cache eviction/refetch fixes for completed torrents and rapid file switching;
  partial and complete eviction are serialized with native write/hash ownership;
  delayed alerts cannot mark replacement buffers with holes complete.
- Compact ordinary telemetry polling; optional Range history loads in the
  diagnostics drawer. Library pages render 50 cards while searching all entries.
- Redacted support download, portable settings/library export and import preview,
  protected recovery snapshots, and read-only startup diagnostics.
- Verified idle-only Windows installation/update, authenticated health checks,
  rollback of executable and configuration, and private failed-migration recovery.
- Restricted Windows service account, seven translated swarm-profile explanations,
  and playback troubleshooting based on server observations.
- Channel-aware platform packages, original dependency notices, checksums,
  exact build identity, GitHub provenance and controlled native regression gates.

## Use and limits

Use the standard executable for direct playback; optional `-gst` builds require
GStreamer. Windows packages include the tray and managed distribution scripts.
See [installation and recovery](https://github.com/gnbk21/TorrServer-LT-Flow/blob/MatriX.145.Flow-v0.2.0-preview.2/DISTRIBUTION.md),
[measurements](https://github.com/gnbk21/TorrServer-LT-Flow/blob/MatriX.145.Flow-v0.2.0-preview.2/MEASUREMENTS.md)
and [remaining stable acceptance gates](https://github.com/gnbk21/TorrServer-LT-Flow/blob/MatriX.145.Flow-v0.2.0-preview.2/RELEASE_ACCEPTANCE.json). Back up existing state and stop
playback before upgrading. Preview 1 requires a manual upgrade because it lacks
the managed update manifest and maintenance leases.

Real Android/Just Player A/B results, representative multi-hour resources and
physical Windows boot/sleep/network recovery remain external acceptance gates.
PGO stays disabled in normal builds; SSE, providers, DoH, predictive downloads
and automatic profile changes are not enabled without evidence. Executables
remain unsigned; GitHub provenance is separate from Authenticode signing.
Keep deployment on a trusted private LAN or VPN: inherited known-torrent playback
routes are not fully protected by management authentication. GPL-3.0 source is
available in this release's source archives and the commit recorded in BUILDINFO.
