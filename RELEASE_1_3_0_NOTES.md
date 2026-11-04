# Flow 1.3.0

Playback clarity, smarter search and a complete Material 3 Expressive interface.
Flow retains the PC → Lampa → external-player workflow and original media quality.
The [implementation and acceptance ledger](docs/development/FLOW_13.md) records
the controlled checks, performance findings and unavailable field checks.
The [publication verification](RELEASE_1_3_0_VERIFICATION.md) records the exact
public binaries, signatures, downloaded-package checks and container digest.

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

The release requires full native, security, browser, platform, packaging and
provenance gates. The pinned comparison baseline is published Flow 1.2.0; its
tag, source and assets remain unchanged. Individual failures and latency flags
are retained in the ledger rather than hidden by a favorable run. This upgrade
does not claim a general streaming-speed improvement or additional peer supply.

The exact tagged binary completed 80 rotated trials against immutable 1.2.0:
50.4 billion verified video bytes, 320 proven cold reads, zero native failed
bytes and no new timeouts. Two workloads flagged median per-trial cold-seek p95
increases of about 0.45 and 0.96 seconds; ordinary-seek flags also remain recorded.
Forced-outage cases recorded delivery deficits in both versions; these are
server-delivery observations, not decoded-player measurements.

Earlier cohorts retained other latency flags and about 152 MiB more median peak
RAM in one workload. The final tagged cohort had no RSS threshold flag, which
does not erase those earlier observations. All results are retained in the
ledger. These costs are not claimed fixed, their precise cause is not established,
and no general streaming-speed or memory improvement is claimed.

Physical Android/Lampa/Just Player launch, selected subtitles, saved-position
handoff, seeks, pause/resume and Wi-Fi reconnection have not been verified for
this candidate. Automated browser/native fixtures do not establish those checks
or manual screen-reader acceptance. No default two-hour test is included.

Use a separate candidate installation and state directory first. Preserve a
settings/database backup and the existing 1.2.0 installation for rollback; never
run two versions against the same state database. Private settings are not
included in public assets. Recent and Viewed additions use separate metadata
storage so older releases can still read legacy marks/timecodes.
