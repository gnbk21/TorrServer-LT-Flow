# Flow 1.3 implementation and acceptance ledger

Authoritative scope: the owner's 2 November 2026 Flow 1.3 plan. This ledger
tracks execution, rather than replacing or reducing that plan. Baseline:
`MatriX.148.Flow-v1.2.0`, source
`aaf2bc583d196d376751dccf265242273eaad89d`. Baseline receipts and generated
evidence live in `_out/flow13`, outside source control. The running personal
installation, credentials, settings and reviewed standalone preview are protected.

| Stage | Concrete acceptance requirements | Status |
|---|---|---|
| 0 | Source/toolchain/API/settings/assets/screenshots and playback baseline receipts; complete parity inventory | Receipts/screenshots captured; complete 80-trial comparison retained |
| 1 | Generated semantic tokens, three palettes, light/dark/system, density, text scales, prepaint preference recovery | Implemented; token/contrast/bootstrap/unit checks pass |
| 1 | All four production destinations and existing workflows use expressive components; adaptive navigation and sheets | Integrated; 154 existing browser scenarios and all three lab repeats pass |
| 1 | Command palette, keyboard/focus, motion, reduced motion, forced colors, seven complete locales | Implemented; all 102 critical Chromium/Firefox/WebKit checks pass |
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
| 7 | Regression/security/native/browser/locale/performance/distribution checks and complete documentation | Source checks recorded with a documented RSS limitation; corrected Mac source validation pending |
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
- Final subtitle admission spans conversion/response buffers, with a bounded
  write deadline that is reset on keep-alive reuse. Signed download/renewal links
  share the same selected capability. Metadata HEAD does not acquire fetch
  admission. Encoding checks include both UTF-16 byte orders and UTF-8 BOM.
  The query-preservation fixture now handles capability discovery separately.
  One prior history drain exceeded its existing one-second test budget; the
  unchanged test passed three isolated repeats, then the affected full packages
  passed serial race checks. Both original failures remain recorded in
  `subtitle-final-race.log`; they were not deleted or counted as passing runs.
- A cold Library shortcut could defer feedback until lazy code arrived (one
  observed interaction was 2,720 ms). Dashboard shortcuts and command navigation
  now reuse the navigation bar's immediate commit behavior. A held-module test
  proves loading appears before that module is released; modified clicks retain
  browser behavior. All 18 affected workflows pass across three engines and all
  three unchanged lab-budget runs pass after this correction. Earlier results
  remain in `lab-cold-shortcut-failure`. Preview as well as development test
  servers reject unmocked API paths locally; a direct HTTP check proves this.
- Combined observer trials reproduced auxiliary starvation: four healthy runs
  delivered every video byte correctly, but both observed runs deferred every
  selected sidecar. Explicit sidecars now receive bounded 5–9-second deadlines
  in the existing reconciler, behind urgent video demand, and admission checks
  verified required pins across active caches. Missing pins, waits and pressure
  still cancel/defer auxiliary work. Both corrected healthy observed runs fetch
  subtitles during playback; the interrupted burst trial and original failures
  are retained. The complete final observer comparison remains required.
- ID-only searches now check versioned structured-search support once per search
  before contacting older Torznab routes; absent support cannot become an empty
  broad query. Title searches retain their immediate path. Authentication and
  service failures keep their own explanations. Internal-player position reports
  coalesce the latest pause/close behind an in-flight report instead of losing
  that observation or replaying the earlier mutation. All 172 web unit tests,
  type/lint/build checks and 125 harness checks pass. Cache/torrent/API race
  checks pass. The local entry point bounds worker creation; earlier sandbox
  permission/worker-start failures remain separate, preserved evidence.
- Hosted checks of `4b8e98ca` passed CodeQL, all 99 critical browser cases, the
  web lab, native builds and most controlled suites. Linux's combined observer
  gate reproduced the same sidecar starvation. A mixed-peer trial failed before
  torrent upload when libtorrent bound the next port after successful preflight;
  it is retained as listener-admission evidence, not waived as streaming success.
  macOS Intel built successfully; the ARM job never received a runner. These
  earlier results do not replace final exact-source platform gates.
- Full local verification of clean source `14596076` passed all 19 steps
  (`_out/verification/20261103T092307448/receipt.json`): native build, package
  inventory, unit/vet/race, harness tests, 154 ordinary browser cases, 99 critical
  three-browser cases and all three production lab repeats. This supersedes,
  without deleting, the earlier incomplete Full receipts.
- All eight corrected combined high-bitrate trials passed with 450,000,000
  byte-checked video bytes each, zero delivery behind schedule and successful
  subtitle retrieval in every observed trial (13 successful active fetches).
  Healthy observer/control median peak RSS was 865,370,112 / 831,248,384 bytes;
  burst RSS was 861,009,920 / 839,315,456 bytes. These bounded observer costs do
  not cross both RSS regression thresholds. The two repeats per mode are an
  overhead check, not a speed claim or a replacement for the five-rotation
  immutable-baseline comparison (`observers-accepted-source/report.json`).
- The final Windows combined integration and isolated rollback check passed.
  Published 1.2.0 read the candidate's legacy 42.5-second saved position and
  settings without modifying the source fixture backup. An earlier invocation
  referenced a nonexistent fixture directory and failed before starting Flow;
  that invocation remains recorded separately. Actual candidate desktop/mobile
  screenshots and timing receipts are in `candidate/ui`. Review corrected two
  presentation defects: an unsampled resource snapshot appeared as a zero cache
  budget, and a QR placeholder could be white on white in light mode. Resource
  samples now distinguish unknown from measured zero; QR generation separates
  loading/failure and preserves black-on-white fallback text. The presentation
  changes pass 177 unit tests, all 99 critical browser cases and three lab runs.
  They change no backend scheduling, HTTP media or observation execution path.
- The owner declined further personal-settings backup investigation on
  3 November ("nevermind"). No personal installation/state was changed or copied.
  The separately staged candidate uses isolated state; the existing installer's
  verified state-snapshot/rollback path remains required before any real update.
- The first correctly rotated Linux comparison completed all 80 trials with
  exact bytes and no timeouts. It flagged mixed seek p95 in healthy/Legacy
  (46.1 to 879.6 ms) and mixed-peer/Adaptive (623.8 to 1,139.8 ms). No RSS flag
  occurred. Raw fixed-offset seeks include both header/index re-reads and remote
  positions; warm versus evicted state was not recorded. Those flags are open,
  not waived. The harness now records verified cache residency before each
  mixed seek and the matched workflow additionally requires four independently
  byte/request-verified cold recoveries per trial. Original mixed results and
  their flags remain visible; cold results do not overwrite them.
- A deeper acceptance check reproduced a separate Recent defect in rc.2:
  explicit player header/index probes could create history entries. Recent now
  reuses request-purpose evidence even when metrics/classification presentation
  is disabled. Whole short clips are classified as video rather than an entire
  file falling inside the eight-MiB tail window; partial headers and long-file
  indexes stay probes. HTTP Range/status/media bytes are unchanged. The old
  executable's failed reproduction is preserved in `player-probe-reproduction`;
  targeted Flow/torrent/API race checks and all 128 harness checks pass. Final
  integration and measured comparison of this correction remain required.
- rc.3's combined integration now proves player header/index probes stay out of
  Recent while actual delivery of all five media fixtures is retained. Four
  instrumented smoke trials also pass verified cold-read and byte checks. Their
  two rotations validate the measurement method, not final performance
  acceptance. The cache snapshots distinguish the last warm header re-read from
  earlier nonresident seeks. Final five-rotation flags remain subject to review.
- New subtitle, Recent and monitoring failures now include schema version 1 and
  no-store headers through the existing error mechanism. Legacy error maps stay
  string-only; status/code/text are preserved. HEAD failures remain body-free.
  Full API race checks and focused version/authentication/alias/admission tests
  pass. The combined integration additionally checks failure-envelope versions.
- Source `6cdc1d6f` passed hosted build run 41619859190, macOS run 41619859206,
  branch checks 41619859459, CodeQL 41619859453 and optional-player checks
  41619859188. Later UI/harness changes still require their own source-bound
  gates; these completed runs do not qualify an unbuilt later revision.
- Actual Chromium tab zoom at 100/200/400 percent exposed Settings actions
  obscuring the form at a 316-by-201 CSS-pixel viewport. The action bar now
  becomes static when it and compact navigation consume half the viewport.
  Twelve real tab-zoom checks pass, followed by all 102 critical cases across
  three browser engines. All 177 web unit tests pass (one existing skip), as do
  type/lint/build and three production lab repeats. Initial JS is 201,978 gzip
  bytes, below 204,800; LCP is 1,104/1,072/1,080 ms and CLS is zero in those
  controlled lab runs. Raw evidence is retained in `zoom-evidence-20261103` and
  `native-zoom-corrected`; earlier zoom failures remain intact.
- Comparison capture now retains native failed bytes, verified resident cache
  turnover, RSS availability and resource/pressure states. Cache turnover is an
  observed lower bound, not a claimed exact eviction count. Unavailable RSS is
  unknown rather than zero; no pressure recovery is claimed without a pressure
  observation. All 133 harness checks pass. Four collector smoke trials retain
  exact bytes, zero native failed bytes and complete resource measurements, but
  flag cold-seek p95 (two-run medians 314.9 to 642.7 ms). This remains open for
  repeated diagnosis, not accepted or hidden as a passing performance result.
  The workflow shards profiles and uses 40-second measured playback including
  both outage windows, retaining all five rotated pairs per profile/scenario
  (80 trials overall). The earlier 60-second workflow and its raw results are
  preserved; shorter feedback does not remove platform/security gates.
- The final source-bound comparison, run 41628583946 at `09c0b9c4`, completed
  all 80 rotated trials: 50.4 billion exact video bytes, 320 peer-request-verified
  cold reads, zero native failed bytes, zero timeouts and no latency threshold
  flags. All eight reports retain cache-turnover and resource observations.
  This is the complete comparison, not selection of the best trial. Earlier
  mixed seek flags remain preserved: their fixed offsets mixed warm/cache-race
  states with network fetches. The separate cold-read proof removes that
  ambiguity; it does not retroactively erase those earlier measurements.
- One final workload retains a resource flag: outage/Legacy median peak RSS
  increased from 929,497,088 to 1,080,672,256 bytes (about 144 MiB, 16.3%). Both
  versions kept resident cache within the configured 512 MiB allowance. Peak
  samples show allocated Go heap persisting through cache turnover/cold seeks,
  including after resident data contracts; the baseline also exhibits this
  pattern in individual trials. The longer 60-second comparison had no RSS
  flag. These observations do not establish a leak or prove a GC root cause.
  The higher peak is retained as a measured resource limitation, not called
  fixed or advertised as a memory improvement. No pressure episode occurred in
  this comparison, so it cannot establish live-pressure recovery performance.
- The source Linux race gate exposed a pre-existing wall-clock test assumption:
  demand and verification can straddle a second, yielding two averaging bins
  with one MiB total delivery. The failed result was 524,288 bytes/s over two
  samples, not duplicate bytes. The assertion now verifies total delivery and
  still rejects double accounting. A deterministic boundary test and both
  affected tests pass 100 race-detector repeats. The original CI failure is
  retained in `linux-validation-09c0b9c4.log`; the correction changes tests only.
- GitHub cancelled the source ARM Mac job without allocating a runner, with
  explicit hosted-capacity annotations. Release preparation uses the supported
  `macos-26` ARM image while retaining the Intel image. Previously verified ARM
  binaries report a macOS 15.0 minimum in their Mach-O load commands; compiler,
  Boost, CMake and final-link deployment flags now preserve that minimum.
  Native receipt identity includes those compiler/recipe changes, and the Mac
  gate checks both produced binaries' minimum OS versions before upload.
  Native ARM unit tests and execution probes remain mandatory. The alternative
  image is [generally available](https://github.blog/changelog/2026-02-26-macos-26-is-now-generally-available-for-github-hosted-runners/);
  this is not permission to omit the ARM platform or reuse an older binary.
  The first alternate Intel attempt rejected Boost 1.85's unsupported
  `macosx-version-min=15.0` property. The compiler-driver minimum remains
  explicit; the unsupported Boost property is removed. That failed attempt is
  preserved, and corrected native builds must pass before release approval.
- PR #14 merged as `182b3ac2`. The source six-platform build (41626944421)
  passes after its diagnosed fixture retry, including all ten high-bitrate,
  eight parallel-streaming and eight controlled-integration roles. The original
  failed attempt remains retained. Corrected release source `614f5066` passes
  CodeQL run 41675442575 (Actions, JavaScript/TypeScript, Windows guard, Go and
  real C++ bridge), with no open findings on that branch. This does not claim
  vulnerability-free software or waive existing documented limitations.
- Published 1.2.0 Intel and ARM binaries were downloaded independently, matched
  against their immutable BUILDINFO checksums, and both report macOS 15.0 in
  their load commands (`published-mac-minimum/receipt.json`). The deployment
  flags therefore preserve the actual published minimum. Corrected Mac run
  41675283140 still needs native success: GitHub again failed to allocate its
  ARM runner and explicitly annotated capacity constraints. That is not a
  completed ARM test. The owner chose to finish all source checks before
  changing release approval or starting the tagged build; approval remains off.
- Final metrics-policy review reproduced rejection of an authenticated socket
  peer with a scoped IPv6 address (`[fe80::1%en0]:port`). The peer parser now
  accepts standard scoped addresses and normalizes IPv4-mapped loopback;
  remote authentication, configured authentication on loopback, and rejection
  of forwarded-address spoofing remain enforced. The failing reproduction and
  three passing race-detector repeats are retained. This changes only the
  monitoring peer parser, not streaming, scheduling, cache allocation or UI.
  The corrected source still requires source security/platform validation.
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
