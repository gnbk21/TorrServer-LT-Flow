# Flow 1.3 implementation and acceptance ledger

Authoritative scope: the owner's 2 November 2026 Flow 1.3 plan. This ledger
tracks execution, rather than replacing or reducing that plan. Baseline:
`MatriX.148.Flow-v1.2.0`, source
`aaf2bc583d196d376751dccf265242273eaad89d`. Baseline receipts and generated
evidence live in `_out/flow13`, outside source control. The running personal
installation, credentials, settings and reviewed standalone preview are protected.

| Stage | Concrete acceptance requirements | Status |
|---|---|---|
| 0 | Source/toolchain/API/settings/assets/screenshots and playback baseline receipts; complete parity inventory | Receipts/screenshots captured; final rotated comparison open |
| 1 | Generated semantic tokens, three palettes, light/dark/system, density, text scales, prepaint preference recovery | Implemented; token/contrast/bootstrap/unit checks pass |
| 1 | All four production destinations and existing workflows use expressive components; adaptive navigation and sheets | Integrated; full parity/browser lab verification open |
| 1 | Command palette, keyboard/focus, motion, reduced motion, forced colors, seven complete locales | Implemented; all 87 critical Chromium/Firefox/WebKit checks pass |
| 2 | Versioned bounded cache/native coverage; truthful freshness, verification, supply and index/frontier states | Implemented; actual refined preload/reader index pins reconciled; targeted race checks pass |
| 2 | Cache-only enriched media; advisory browser compatibility; explicit conversion choices | Implemented; targeted checks pass |
| 3 | Advertised structured Torznab queries, bounded capability cache, cold generic search and one fallback | Implemented; targeted planner/cache/cancellation checks pass |
| 3 | Bounded redacted endpoint health/history, timing/query mode, aggregate-source distinctions and setup guidance | Implemented; privacy/bounds checks pass |
| 3 | Parsed release badges, unknown values, identity-only source grouping, explainable filters and stable sorting | Implemented; unit checks pass |
| 4 | Metadata-only exact/episode subtitle matching and bounded candidate listing | Implemented; integration proves no media download/probe/session from listing |
| 4 | Auxiliary reader admission, pressure/startup protection, encoding/conversion limits and full cancellation cleanup | Implemented; unit/native v2/hybrid and combined integration checks pass |
| 4 | Scoped subtitle capability aliases, browser tracks, Just Player sibling discovery and manual fallbacks | Implemented; byte/scope/browser checks pass; physical player acceptance unavailable |
| 5 | Bounded shared Recent history, disable/remove/clear, compatible Viewed metadata and truthful provenance | Implemented; persistence/provenance/integration checks pass |
| 5 | Real internal-player position reporting, explicit saved-position handoff and existing episode/warmup reuse | Implemented; component/intent checks pass; physical handoff acceptance unavailable |
| 6 | Default-off protected Prometheus endpoint, fixed-label monotonic aggregates and cache-only scrapes | Implemented; authentication/privacy/monotonicity/integration checks pass |
| 7 | Regression/security/native/browser/locale/performance/distribution checks and complete documentation | In progress; performance flags not yet accepted |
| 7 | Isolated candidate package, rollback compatibility, final scope reconciliation and qualified release | Not published; required gates remain open |

## Production parity inventory

Retain Overview/active/warm streams, Library pagination/categories/filter/sort,
torrent details and progressive file/episode navigation, add magnet/hash/URL and
torrent upload, Rutor/JacRed/Torznab search, direct/external/internal playback,
M3U/copy links, viewed marks/timecodes, optional GStreamer/audio/embedded and
uploaded subtitles, metadata edits/poster lookup, remove/cache controls, Flow
pause/retry/warmup/preparation, verified reuse/web seeds, network/Range/timeline/
storage/resource diagnostics, private incident/support exports, LAN pairing,
backup preview/apply, runtime/developer storage views, complete settings metadata
and draft/saved/effective states, General/Flow/Network/Integrations/Security/
Advanced settings, WAF/HTTPS/auth, DLNA/Bonjour/WebDAV/FUSE configuration,
service/tray state and confirmed destructive actions. Backend state remains
authoritative; no server settings autosave or mutation replay.

## Defaults and acceptance boundaries

New useful presentation and local Recent history are enabled. Position tracking
retains the saved TrackTimecode value. Subtitles require selection. Prometheus
and conversion remain opt-in. Existing profiles, experimental controls, privacy
consents and trusted origins are unchanged. No paid provider, new engine,
automatic release/quality changes, companion app or default two-hour test.
Controlled native/browser results are separate from physical phone, public-swarm
and long-duration field acceptance. Missing acceptance is never called passed.

## Evidence and unresolved acceptance

- Baseline source/toolchain/API/settings/asset receipts, 1440/375-pixel screenshots
  and isolated calibration are under `_out/flow13/baseline`. The original published
  Windows executable SHA-256 is
  `447e488ac091ddd0f3dc5947295ff1392b68501925f215a854abd81a2952669f`.
- Targeted backend checks passed for settings, Flow, search status, JacRed,
  Torznab, cache storage, torrent lifecycle and HTTP APIs. Web iteration checks
  passed 163 tests with one existing skip, type checking, lint, token sync,
  bootstrap recovery and production build. These are not final tagged gates.
- `integration-r3/report.json` proves metadata-only observation, original and
  escaped subtitle bytes, selected token scope, authenticated Recent/provenance,
  process-monotonic metrics and unchanged idle expiry. The five existing MP4/MKV
  head/tail-index and variable-rate fixtures include seeks, cancellation and
  overlapping readers. Native TCP/uTP v2/hybrid auxiliary transfer also passed.
- Initial browser failures and traces were retained. Closed-details overflow and
  command-palette focus teardown were corrected. All six focused large-list and
  command workflows pass across Chromium, Firefox and WebKit. Earlier 81 passing
  critical checks do not replace the final complete rerun.
- The final critical rerun passed all 87 cases, including automated accessibility
  audits of Overview, Library, Search/Add and Settings in all six palette/mode
  combinations, all text scales, seven languages, Unicode/progressive files,
  focus return, reduced motion, forced colors and truthful Recent handoff.
  Navigation clearance was corrected for bottom bars versus side rails.
- The full native package inventory, vet and race runs passed. Subsequent targeted
  race checks cover actual refined index pins, auxiliary native completion and
  foreground eviction, Rutor unknown metadata and cold-query batch/fallback
  behavior. All 124 harness/workflow tests passed. Source-bound hosted checks
  remain required; the earlier Full script receipts retain their UI failures.
- All 150 ordinary browser scenarios passed. The production lab then caught an
  eager Recent confirmation-dialog import (215,055 gzip bytes) and loading-state
  layout shift. Dialog code is now lazy and Recent reserves consistent space,
  including an explicit unsupported-server state. Nine additional three-browser
  Recent checks pass. The final serial three-run lab records 201,896 gzip bytes,
  LCP 1,220/1,224/1,244 ms and zero CLS; input, import, diagnostics and settings
  budgets also pass. A prior slow-start outlier and all earlier failures remain
  preserved (`lab-startup-outlier`, `lab-layout-shift-failure`,
  `lab-initial-budget-failures`); no best-run-only claim is made.
- A native race test also closes the admission/shutdown gap: a reader published
  after a cache begins closing is rejected/cancelled, rather than waiting for
  the normal ten-second subtitle deadline. Active engine/cache close cannot
  retain an auxiliary reservation.
- The initial 40-trial comparison preserved zero byte failures/timeouts, but
  flagged seek latency/RSS. Review discovered that the old harness's odd-round
  reversal cancelled its two-policy rotation, always placing the baseline first.
  That defect is fixed and unit-tested. The old evidence remains preserved as
  `comparison-default-r2`; it does not establish performance acceptance. A custom
  same-toolchain baseline build is diagnostic only and cannot replace the pinned
  published baseline. No speed or memory improvement is claimed.
- Full local verification, final repeated performance review, hosted platform/
  security/distribution/provenance gates, visual parity and candidate packaging
  remain open. No default two-hour test has been introduced. Real Android/Lampa/
  Just Player subtitles, resume, reconnection and a manual screen-reader smoke
  check are not established by controlled browser/native tests.

## Security and compatibility review

New management endpoints inherit the existing authenticated/origin/rate-limited
groups. Subtitle aliases validate signed scope against metadata before activation;
old tokens grant no sidecar authority. Auxiliary reads have complete-piece
admission, deadline/pressure cancellation and private responses. Metrics determine
the socket peer directly, use fixed labels and read cached aggregates. Recent is
bounded shared local metadata; legacy marks are not completion evidence. No
trusted origin, scheduling default, privacy consent or TrackTimecode value is
changed. Historical alert dispositions do not authorize dismissing new findings.

The 2 November Go vulnerability scan found zero affected symbols or imported
packages. It also reports module-only
[GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932), covering the unmaintained
`golang.org/x/crypto/openpgp` packages (no fixed version). Flow does not import
those packages; it retains `x/crypto` for other dependencies. The full scanner
output is preserved in `_out/flow13/govulncheck-verbose.log`. This is a scoped
reachability result, not a claim that Flow is vulnerability-free.
