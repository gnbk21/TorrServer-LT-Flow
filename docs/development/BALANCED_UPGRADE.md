# Balanced upgrade implementation ledger

This is the implementation and verification record for the 30 October 2026
balanced roadmap. Historical release evidence remains in `docs/history`.

| Stage | Required behavior | Implementation |
|---|---|---|
| Correctness | Joint TCP/UDP admission, bounded retries, separate fixture failures, both harnesses | Implemented; rejected/occupied-port and cleanup tests, eight Windows straggler trials |
| Correctness | Explicit errors, polling/manual separation, no mutation replay | Implemented; 503/401/403/429 and manual pause/cancel regression checks |
| Correctness | Deprecate ineffective browser profile; preserve configuration compatibility | Implemented; retained API/config field and hidden editor control |
| Correctness | Complete test-package inventory, ordinary Rutor tests, documentation and locale reconciliation | Implemented; all 38 Go test directories assigned, full Rutor DB remains opt-in |
| Diagnostics | Unified evidence timeline and read-only single-change adviser | Implemented in playback overview; stale evidence, storage, source and phone distinctions |
| Resources | Three fresh five-second recovery samples above 1.5× entry threshold; protect foreground | Implemented; deterministic oscillation/recovery, protected reads, background recovery and sampler admission tests |
| Incidents | Optional private flight recorder, bounded output, cooldown, retention and pressure handling | Implemented; real trace, output overflow, private files, crash-partial cleanup, restart visibility and stop/resume tests |
| Setup | Authenticated creation/status, five-minute bounded read-only phone proof, permission guidance | Implemented through existing management policy; token isolation, expiry/capacity, seven-locale mobile checks |
| Settings | Backend metadata and old-server fallback, cache/application preview, accessible translated UI | Implemented; scalar registry shares enforcement, defaults/restart and migration tests, real Go/frontend contract check |
| Sources | Selected-media probe details, parser health, verified reuse and warmup/recovery evidence | Implemented; cache-only media reads, native verification counters, localized states and lazy-read checks |
| Storage | Separate runtime categories and safe developer preview inventory/explicit cleanup | Implemented; read-only bounded walks, RAM/disk distinction, canonical/active/locked/path-change safeguards |
| Development | Local verification and toolchain/cache receipts | Implemented; Quick/Full entry point and verified native compiler/dependency/recipe receipts |
| CI | Shared source-bound web artifact; independent Linux build/validation; retain all release gates | Implemented; source/trust-boundary tests, actual PR-run identity checked, workflow policy/actionlint checks |
| Verification | Timer policies, byte fixtures/index placement/VBR, repeated matched measurements | Implemented; results and limitations below |
| Toolchain | Separate Go 1.27.3 comparison; retain 1.26.10 until qualified | Evaluated on Windows; pin retained, no promotion |

Compatibility: preserve media bytes, Range behavior, torrent identity, saved
settings and existing defaults. No automatic profile/origin changes, paid
providers, mandatory preparation, quality reduction, companion app, SSE, QUIC,
new engine, ML tuning or unrestricted duplicate requests. No default two-hour
test. Controlled checks do not constitute phone/public-swarm acceptance.

## Verification record — 31 October 2026

The correctness milestone was completed before expanding behavior. Commit
`fb264991` passed PR #13's checks (55 successful, seven optional/skipped), and
the corrected Windows straggler fixture completed eight controlled trials.
That is separate from validation of the later changes. The final source's live
platform/security/distribution conclusions are attached to
[PR #13](https://github.com/gnbk21/TorrServer-LT-Flow/pull/13); a new release still
requires its applicable publication gates.

Local evidence is retained under `_out/verification` and `.tools/upstream1472`:

- `Verify-Flow.ps1 -Scope Quick`: web lint/typecheck/build, 144 web unit tests
  (one explicitly opt-in contract check skipped in the ordinary run), Python
  harness tests, embedding tests, native Windows build, all 37 ordinary native
  package test directories and vet. The separately exported actual Go contracts
  passed all six frontend contract checks.
- Python harness/policy tests: 114 passed. Package inventory: all 38 test
  directories, including the separate GStreamer suite. All six changed workflow
  files passed actionlint; policy tests retain native/security/platform/package/
  provenance gates.
- Browser suite: 118 scenarios passed across seven locales, mobile/landscape/
  desktop layouts, enlarged text, keyboard focus, long filenames, setup, settings,
  storage and lazy metadata reads. Two additional targeted scenarios passed for
  expired-proof renewal and localized metadata-recovery status.
- Production interface lab: three repeated runs passed the existing budgets
  after deferring unopened dialog code. Initial compressed JavaScript fell from
  206,552 bytes (over budget) to 194,059 bytes. LCP was 788/788/764 ms and CLS zero
  under the lab's 4× CPU slowdown and controlled network. These are browser-lab
  results, not physical-phone INP or playback measurements.
- Media fixtures: fast, slow-peer and disconnect cases passed 90 Range operations,
  including 15 cancellations, overlapping readers and byte hashes. Generated
  MP4 moov and Matroska cue placement are structurally checked at front/end;
  variable-rate media is verified from measured packet bytes. The fast case
  observed warm-reservation expiry rather than assuming it.
- Resource recovery: injected pressure preserves reader/preload/index protection
  even above a soft allocation, releases warm work first and restores background
  capacity. Repeated recovery samples do not churn the budget. The test records
  read waits, eviction markers, budget changes and RSS sample timestamps; it does
  not exhaust physical host memory or infer RSS changes from a cached sample.

### Performance observations and limits

`compare_streaming.py` preserves each attempt, calibration, configuration,
executable/fixture SHA-256 and individual measurements. Three alternating
baseline/candidate pairs at 90 Mbps completed byte/seek/cancellation checks.
Warm-cache residency varied, so ordinary seek timings cannot establish a general
speed improvement. A cheap admission check was restored before memory sampling
on native rate updates, with a regression test proving suppressed updates do not
invoke the memory sampler.

The first final-source CI run also exposed a test-only one-second history-file
drain assumption on a cold Windows runner (`TestHistoryNestedEnumsCannotRetainSecrets`).
Functional filesystem tests now allow a bounded five-second drain and check
previously ignored close results. Explicit blocked-writer deadlines and all
privacy/retention assertions remain intact. Portable compiler concurrency is
bounded to two packages. Production flush/shutdown deadlines are unchanged.
The source-bound web handoff manifest is ignored by Git so its presence cannot
mark an otherwise clean native build as modified; a policy test enforces this.

A 90 Mbps straggler comparison completed one pair but then failed its fixture
supply check (177.38 then 104.40 Mbps, below the required 1.3× margin). That
incomplete run is retained and is not a passing performance qualification.
A separate, disclosed 60 Mbps straggler comparison completed all three pairs;
every cold seek proved a new peer request and every body passed byte validation.
Cold-seek p95 observations, in milliseconds:

| Build | Run 1 | Run 2 | Run 3 |
|---|---:|---:|---:|
| Baseline | 605.1 | 793.6 | 604.6 |
| Candidate | 500.1 | 676.3 | 2787.6 |

The outlier remains part of the evidence. These variable results do not justify
promoting a profile or claiming a general streaming performance gain. The memory
policy is qualified by its deterministic transition/protection tests; Legacy
and all experimental defaults remain unchanged. No user media quality was
reduced, and no public torrent or owner playback cache was used.

The separate Go comparison passed builds and compatibility checks for 1.26.10
and 1.27.3 and retained 12 benchmark observations across three alternating
rotations. Measurements were mixed and variable. This was Windows qualification
of that comparison snapshot, not eight-platform qualification or authorization
to upgrade the runtime; `server/go.mod` stays at 1.26.10.

## Reconciliation and operational decisions

- Existing management authentication/origin/rate policy governs pairing creation,
  status and settings metadata. Disabling HTTP authentication retains the existing
  trusted-LAN behavior; a proof is never an authentication credential.
- Settings metadata describes scalar editor constraints and explicitly distinguishes
  historical permissive API fields and zero migration sentinels. Existing complex
  integration editors retain their own wire formats and handlers.
- Raw traces are separately selectable private downloads, never ordinary support
  exports. No recorder, experimental scheduler, trusted origin or disruptive
  setting is enabled/applied by the adviser or setup flow.
- Developer cleanup remains preview-only unless the owner supplies the exact
  reviewed manifest hash. No owner files, installed server or personal settings
  were deleted/replaced by this upgrade.
- No architectural feature was substituted. The lower cold-test demand is a
  fixture-admission adjustment with retained evidence, not a video-quality change.
  Phone/public-swarm and physical recovery acceptance remain explicitly separate;
  no new two-hour test was added.
