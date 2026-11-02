# Flow 1.3.0 candidate

This is the candidate's release description. Publication remains gated by the
[implementation and acceptance ledger](docs/development/FLOW_13.md). Flow 1.2.0
remains the immutable published baseline until a qualified successor is released.

## Changes

- Material 3 Expressive throughout the existing interface: three palettes,
  light/dark/system appearance, density, text scaling, adaptive navigation,
  reduced-motion/forced-color support and a Ctrl/Cmd+K command palette. All seven
  existing languages are retained. Preferences apply before first paint.
- Bounded, selectable playback coverage separates resident/verified pieces,
  outstanding requests, sampled connected-peer supply, actual required header/
  index pins and server delivery frontiers. Age, unknowns and truncation are
  explicit. It does not measure decoded player buffering.
- Cache-only container/codec/track/HDR information and compatibility explanations.
  Direct playback comes first; optional GStreamer conversion requires selection.
- Advertised Torznab movie/TV identifiers, year and episode queries; cold title
  searches remain immediate, with bounded discovery and one applicable generic
  fallback. Redacted source health/history, filename-derived release badges,
  identity-based source grouping, unknown counts and explainable filters/sorting.
- Explicit torrent subtitle selection, bounded SRT/ASS/SSA/VTT retrieval and
  escaped WebVTT conversion. Scoped private aliases support Just Player sibling
  discovery, with copy/download/manual fallbacks. Auxiliary work yields to video
  startup, foreground reads and pressure; it creates no playback/history entry.
- Shared local Recent history (up to 200 files/30 days) with disable/remove/clear,
  saved-position provenance and explicit completion. Internal-player positions
  are reported only with existing TrackTimecode consent. Resume handoff is
  explicit; launching an external player does not provide return telemetry.
- Default-off, protected Prometheus monitoring on the existing listener, using
  cached aggregates, fixed labels and process-monotonic counters. No torrent
  identities, filenames, queries, addresses or credentials are exported.

Existing profiles, experimental scheduling, metadata-cache consent, trusted
origins, authentication and position-tracking settings retain their values.
Original media bytes, Range behavior and existing links/APIs remain compatible.
No paid service, quality reduction or compulsory full download is introduced.
See [limits, settings and API contracts](FLOW.md#flow-13-additions).

## Acceptance and upgrade notes

Final exact-source native/security/platform/packaging/provenance gates and the
corrected repeated comparison are still pending. Earlier mixed timing/RSS results
are preserved and are not evidence of a speed improvement. The initial harness's
two-policy ordering defect is fixed; no favorable run is selected to hide a
failure. The candidate is not yet an accepted performance replacement.

Physical Android/Lampa/Just Player launch, selected subtitles, saved-position
handoff, seeks, pause/resume and Wi-Fi reconnection have not been verified for
this candidate. Automated browser/native fixtures do not establish those checks
or manual screen-reader acceptance. No default two-hour test is included.

Use a separate candidate installation and state directory first. Preserve a
settings/database backup and the existing 1.2.0 installation for rollback; never
run two versions against the same state database. Private settings are not
included in public assets. Recent and Viewed additions use separate metadata
storage so older releases can still read legacy marks/timecodes.
