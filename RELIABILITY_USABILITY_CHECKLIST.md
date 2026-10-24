# Reliability and usability milestone

Approved 24 October 2026. Preserve the native engine, original media bytes,
current defaults, existing state location, public-cache consent and seven locales.
Deliver a reviewed candidate/PR; publication is separate. No two-hour tests.

## Required work

- [ ] Versioned Flow-owned update inventory; shared-directory/reparse safety; scoped rollback.
- [ ] Latest selection, downgrade protection, source/workflow-bound provenance and runtime identity.
- [ ] Clock-independent freshness across dashboard, strip and diagnostics; legacy compatibility.
- [ ] Incremental partial search, bounded parallel Torznab and categorized outcomes; exact-hash deduplication.
- [ ] Strict compatible storage requests, configuration serialization/revisions, idle gate and transactional recovery.
- [ ] Opt-in bounded anonymous last-session summaries, restart recovery and support/UI integration.
- [ ] Actionable integration tests, guided troubleshooting and user-started network checks.
- [ ] Management polling honors Retry-After without retrying mutations or hiding stale observations.
- [ ] Bounded redacted server/web log rotation, including shared sink and native descriptor handling.
- [ ] Separate supported Intel/ARM Mac runners and caches; exact-source reusable platform recovery.
- [ ] Short isolated netem network regression cases; byte integrity and resource/recovery measurements.
- [ ] Contract/unit/browser/native/Windows verification, security boundary reconciliation and final diff review.
- [ ] Final approved-plan reconciliation, accurate evidence/limits and reviewable PR.

## Explicitly later

Automatic same-torrent subtitles, expanded episode navigation, selective storage
cleanup and deeper symbolized C++ profiling remain follow-up candidates. No new
engine, automatic profile switching, verified-subpiece mode, PGO promotion,
paid provider or protocol replacement is part of this milestone.

## Evidence

Updated as each requirement is implemented and verified. A checked item requires
actual behavior and relevant verification, not simply related code.
## Verification progress (24 October 2026)

- Local frontend: 123 tests passed; one native-contract import awaits CI data.
- TypeScript and ESLint passed. Production Vite build passed (existing lazy HLS chunk warning).
- Browser suite: 102/106 passed in the initial run. The four failures exposed old
  timestamp-free fixtures. After making those fixtures represent fresh server
  measurements, all four focused regressions passed; complete rerun is pending.
- New partial-search and manually requested session-history browser checks passed.
- Build/release Python suite: all 88 tests passed. Actionlint and diff whitespace passed.
- Pure Go suites: flow, searchstatus, log and version passed, including recovery
  summaries, request bounds, native-descriptor-preserving log rotation and channels.
- Update-state fixtures passed in PowerShell 5.1 and 7: unrelated-file preservation,
  checksum rejection, scoped quarantine, junction rejection and hard-link rejection.
- Native engine compilation, exported Go/TypeScript contracts, storage fault
  injection, authenticated storage integration, Windows updater candidate and
  Linux netem evidence await the candidate CI run. No owner installation was changed.
- Production payload regression found and fixed by deferring optional diagnostics,
  history validation and inactive playback cards. The 200 KiB gate is unchanged.
- Freshness now conservatively includes request elapsed time, so a slow response
  cannot renew an already stale sample. Playback chart retention uses monotonic time.
- Native build startup exposed an existing reusable-workflow permission mismatch;
  both container callers now grant the callee's required artifact-read permission.
- Windows link-specific recovery fixtures are gated to Windows and wired into the
  actual Windows service job; platform-neutral inventory checks remain in tagging.
