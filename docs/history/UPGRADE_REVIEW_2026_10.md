# Flow upgrade review — 22 October 2026

## Scope and verified baseline

Review of source `0e8331a1`, current GitHub state, retained numeric playback
captures and primary technical documentation. This is an upgrade backlog,
not a claim that the proposed changes are implemented or that a full security
audit or new phone acceptance run has passed.

- [PR #6](https://github.com/gnbk21/TorrServer-LT-Flow/pull/6) is open and
  mergeable. Every non-skipped check currently reports success, including native,
  CodeQL, eight high-bitrate comparisons, Windows service, platform builds and
  ETW/Perfetto capture. Skipped optional workloads are not passes.
- Latest remains `MatriX.145.Flow-v1.1.3`, published 11 October. PR #6 contains
  newer development work; a successful PR check does not publish that work.
- Latest LT release returned by GitHub is still `MatriX.146.LT-1.2.2`.
  Flow's main native recipe already pins libtorrent **2.1.2**. Its relevant
  upstream fixes are not a new upgrade to propose again.
- No TorrServer process was returned by the process query during this review.
  Playback observations below come from retained 20 October captures.
- Fresh frontend verification: **109 tests passed, one skipped, 26 test files
  passed; type checking and lint passed**. The initial restricted test attempt
  failed on temporary-file rename permissions before running tests; the repeat
  with workspace-local temporary files and the unrestricted runner passed.

Existing functionality includes metadata recovery, native request-capacity and
cancellation experiments, adaptive deadlines/windows, container hints, verified
web seeds, resource budgets, revisioned settings, rollback, paginated library,
visibility-aware polling and optional Media3 player evidence. Extend these
mechanisms rather than add parallel implementations.

## 1. First fixes: correctness and useful evidence

| ID | Finding and source | Recommended change | Required verification |
| --- | --- | --- | --- |
| F1 | `server/torr/flow_metrics.go`: `flowStart` enters `SEEK_RECOVERY`; `flowFirstByte` records recovery time without clearing that state. `flowEnd` can also set `PLAYING` when an older overlapping response ends. | Make delivery state follow the current playback request generation. Current successful first byte ends server seek recovery; older responses and probes cannot clear a newer seek. Keep player state explicitly separate. | Overlapping seeks, old-response completion, cancellation before first byte, probes, current successful 200/206 response and last-reader close. No claim about decoded playback. |
| F2 | `server/lt/lt_shim.cpp::alert_json` retains peer error number and operation without its error category. Tracker errors lose native category/code/repeat count and become broad string-based classes in `tracker_diagnostics.go`. | Carry a bounded category enum, numeric code, operation and consecutive-failure count through Go, history, support export and UI. Interpret HTTP status only for the appropriate category. | Identical numeric values from different categories; DNS, timeout, refusal, TLS and HTTP failures; old history compatibility; no raw passkeys, URLs or exception strings. |
| F3 | Native `hash_failed_alert` includes a piece index, but `BTServer.handleAlert` has no corresponding case. Sparse status exposes cumulative failed bytes; retained evidence cannot explain when verification failures occurred. | Add bounded hash-failure events and recent deltas. Distinguish integrity failure, redundant bytes, refetch and transport failure. Inspect cache invalidation behavior as part of the regression. | Corrupt-block fixture, successful refetch, repeat failure, counter reset and session boundaries. Do not identify a malicious peer or faulty hardware without supporting evidence. |
| F4 | `FlowStatusWithTraces` combines cached group-window contiguous counts with an independently queried current-demand buffer. `FlowWindowStatus` exposes neither that count's anchor nor sample time. | Expose offsets and sample ages, or compute related counts from a common anchor. Label the server cache estimate separately from actual phone buffer. | Seek during sampling, multiple readers, stale group window, cache hole and absent bitrate. Never turn unknown data into zero or an optimistic safety score. |
| F5 | `NeedsEngineRestart` omits `AdaptiveUrgentHorizon` and `ContainerBurstHints` from its hot-policy allowlist, although their inspected consumers read current Go policy. | Verify disable/reset semantics for existing groups/index tasks, then permit safe changes without an engine restart. Keep native capacity/transport changes in the restart path. | Toggle each flag during active delivery, validate cache/index cleanup and ensure native settings still require restart. |
| F6 | `Dockerfile.builder` retains LT 2.1.0, Go 1.25 and an old product identity, and bypasses the main native patch recipe. Published containers instead use tested cross-build outputs via `Dockerfile.runtime`. | Align the standalone source builder with the authoritative recipe, or explicitly retire that build entry point and document the supported replacement. | Clean source build, native feature identity, dependency inventory and original-byte streaming check. This review did not build that Dockerfile and does not claim published containers are affected. |
| F7 | `STATUS.md` describes the October 11 release as the current upgrade; metadata delivery documents preserve a checkpoint with CI still pending. | Keep one dated current-status entry for PR #6 and link immutable checkpoints as history. Show stable versus development identity clearly. | GitHub commit/check identity, package source/version/hash and current UI identity agree. Preserve historical failures and field-test limits. |

F1–F4 should precede scheduler tuning: misleading states and mixed measurements
make it harder to tell whether a proposed optimization actually helps.

## 2. Streaming work with the highest expected value

### S1 — Reproduce and reduce delays from the final missing blocks

The reopened playback capture showed demanded 8 MiB pieces missing a small
number of blocks while later pieces were complete. The follow-up, after seeks
stopped, had one active reader, **64.5–79.4 seconds** of server buffer, no
read stalls in the preceding minute, and unchanged request/cancellation counts.
The user reported smooth playback then. This supports testing seek/straggler
recovery; it does not establish a persistent bandwidth or ISP problem.

Extend the existing native workload with **8 MiB pieces**, a delayed final block,
reordered completion, alternating cold seeks, limited advertised peer queues,
mixed TCP/µTP peers, concurrent readers and cancellation during recovery.
The current CLI accepts 1/4/16 MiB and does not cover that exact piece size.

First measure libtorrent's existing deadline retry behavior. If it leaves a
reproducible gap, test a bounded retry adjustment for old demanded blocks using
alternate eligible suppliers, respecting native request capacity and cancel
credit. Stop speculative work after useful delivery resumes. Do not introduce
a second piece picker or an unlimited duplicate-request policy.

Acceptance: matched repeated baseline/candidate runs; lower p95/p99 cold-seek
first-byte time and blocked-read time, with byte integrity, bounded duplicate
traffic/RSS and no regression in healthy or scarce-supplier cases. Keep opt-in
until comparative results justify promotion. [Libtorrent already prioritizes
time-critical pieces and can duplicate overdue requests](https://www.libtorrent.org/streaming.html).

### S2 — Make request cancellation and recovery generation-aware

Audit the complete reader → group window → deadline → native queue path after
rapid seeks. Cancel obsolete requests promptly while retaining useful near-term
verified cache. Prefer the latest real playback demand over old probes and
cancelled ranges; preserve fairness between devices. Existing cancellation and
capacity patches remain the foundation. Test this together with S1, not by
globally shortening every peer timeout or repeatedly reconnecting the engine.

### S3 — Shorten metadata fallback tails without flooding sources

`recoverMetadata` currently tries sources serially, with an eight-second timeout
per source after the fallback delay. Concurrent requests are coalesced, but
`FetchGroup` deliberately has no result cache or service-health cooldown.

Evaluate a staggered second request only after an authorized source is slow,
with a strict per-torrent and global concurrency/byte budget. Add short, bounded
service-failure cooldowns and distinguish a missing torrent from an unavailable
service; permit an explicit retry. Preserve supplied-source preference and make
any additional public-cache contact obey consent at execution time. Reuse
saved verified metadata first, cancel losers, check every known hash, and retain
DNS pinning, HTTPS/interface and redirect protections. A cache cannot supply
the video payload when there are no useful seeds.

Acceptance: slow/missing/invalid sources, consent revoked mid-operation,
same-source coalescing, different network policies, torrent removal, peer
metadata winning the race and recovery after cooldown. Compare tail latency
and external request count, not just a single successful cache lookup.

## 3. Web interface and daily use

| Upgrade | Existing foundation and proposed increment | Acceptance |
| --- | --- | --- |
| Clear playback explanation | Existing startup/timeline panels should show one concise cause, evidence age and appropriate action: metadata wait, no supplier, missing block, local storage or unknown player state. Put raw counters behind details. | EN/RU plus remaining locales, narrow phone layout, stale/disconnected states; no automatic engine restart or unfounded ISP diagnosis. |
| Honest buffer display | Show verified server buffer, its media-rate confidence and sample age. Actual phone buffer appears only with matching player evidence. Show recovery transitions from F1. | Unknown duration, bitrate bursts, mismatched player report and paused/closed player. |
| Responsive interactions | Retain pagination, shared active snapshots, selectors, lazy panels and hidden-page cancellation. Add measured interaction cases for opening diagnostics, settings, import and filtering; investigate large chart/JSON work only if profiling identifies it. | Existing production laboratory budgets plus Event Timing/long-task measurements and real-phone checks. The current two-frame input probe is not full field INP. [INP guidance](https://web.dev/articles/optimize-inp), [query render optimization](https://tanstack.com/query/latest/docs/framework/react/guides/render-optimizations). |
| Accessible live status | Announce meaningful state changes; avoid repeatedly reading rapidly changing numeric telemetry. Preserve keyboard focus, reduced motion and readable charts. | Screen-reader smoke, keyboard navigation and mobile zoom under active polling. |
| Trustworthy update/status view | Show stable/development channel, source identity and whether the running package includes a fix. Preserve draft/saved/effective settings and indicate why an apply requires restart. | Exact installed-binary check and restart-free policy changes; preserve rollback. |

SSE is a later measured option if polling cost becomes material. The current
visibility-aware/shared polling is already implemented; replacing it immediately
adds reconnect, authentication, backpressure and proxy complexity without an
established benefit. A PWA shell must not cache authenticated responses, settings,
media ranges or support exports.

## 4. Player, network and engineering improvements

- **Improve the existing optional Flow Player integration.** It already uses
  Media3 `PlaybackStatsListener`, memory-aware load control, bounded local JSON
  export and a server-session header. Correlation is currently learned only on
  load completion/cancellation/error, so a long open response may remain
  unmatched. Investigate capturing the header when the data source opens, add
  monotonic event ordering and a useful whole-session summary. Keep manual
  export/import working; any paired live transfer must be separately authorized
  and authenticated. Stock Just Player does not gain this instrumentation from
  a server update. [Media3 analytics](https://developer.android.com/media/media3/exoplayer/analytics).
- **Use evidence-based network recovery.** Extend the current network guard
  and diagnostic stages with categorized failures from F2 and controlled
  adapter/sleep/reconnect tests. Keep IPv4/IPv6 and TCP/µTP results separate.
  A successful port mapping does not prove incoming reachability; peer timeouts
  do not prove ISP interference. Preserve selected-interface and encryption
  preferences. Review source/interface behavior of every new remote client.
- **Review security alerts by affected branch and source.** GitHub currently
  returns 30 open alerts (seven critical, 23 high); their most recent instances
  identify `develop`, and the existing release security review documents these
  retained findings. These are not 30 newly confirmed vulnerabilities. Reconcile
  their exact paths/guards against current source and retain regression tests
  for redirects, owner-selected paths, malformed metadata and credential
  redaction. A passing CodeQL job is not a zero-alert result.
  [GitHub alert semantics](https://docs.github.com/en/code-security/how-tos/manage-security-alerts/manage-code-scanning-alerts/assess-alerts).
- **Finish representative endurance and recovery evidence.** Reuse the existing
  resource runner for uninterrupted multi-hour playback/idle/seek cycles and
  perform physical Windows boot/sleep/network checks. Track peak and post-idle
  RSS, native allocations, disk descriptors, goroutines, cache refetch and
  update rollback. Interrupted runs remain incomplete.
- **Optimize builds and diagnostics costs from profiles.** ETW/Perfetto, Go
  profiling and an optional PGO path already exist. Capture representative
  steady playback, seek and idle profiles before changing allocations or
  adopting PGO. Go PGO does not optimize native C++ by itself. Keep profiles
  private/bounded and prevent media identities from entering published evidence.
  [Go PGO guidance](https://go.dev/doc/pgo).
- **Make development runs cheaper to maintain.** Reuse dependency/patch-identity
  caches, cancel superseded branch runs only under an explicit workflow policy,
  retain failed workload evidence, and use bounded artifact retention. Preserve
  the complete release matrix and provenance gates. Consolidate overlapping
  current-status documents; retain dated evidence instead of deleting failures.

## 5. Technology decisions

- **Retain libtorrent 2.1.2 and the current architecture.** Its release includes
  priority, hash-picker, request and shutdown fixes. Flow's main recipe already
  includes that version. Review newer upstream changes individually with native
  patch compatibility and comparison tests. [Upstream release](https://github.com/arvidn/libtorrent/releases/tag/v2.1.2).
- **Retain native web seeds as an optional free additional source.** Improve
  their health/failure explanation where needed. They require a legitimate
  matching mirror, not an arbitrary same-named video. [BEP 19](https://www.bittorrent.org/beps/bep_0019.html).
- **Treat v2 block-verification work as research.** Merkle proofs may support
  finer verified availability, but libtorrent support alone does not mean Flow's
  cache/reader serves independently verified subpieces. Never expose unchecked
  bytes, combine different releases by name, or promise this improves existing
  v1 swarms. [BEP 52](https://www.bittorrent.org/beps/bep_0052.html).
- **Defer QUIC/WebRTC transport rewrites, ML scheduling, paid providers and a
  second media cache.** No evidence here justifies their compatibility and
  maintenance cost. HTTP/3 to the phone does not improve an unreachable seed's
  TCP/µTP connection. Keep high-quality original-byte streaming as the target;
  full-episode downloading and lower quality are not the required solution.

## Recommended delivery order

1. F1–F4: accurate state, categorized errors, integrity events and anchored metrics.
2. F5–F7: safe hot-policy changes, build consistency and current release identity.
3. S1/S2: exact 8 MiB straggler/seek reproduction, then measured native recovery.
4. S3 and the web explanation/performance work, each with bounded failure tests.
5. Optional player correlation improvements and representative endurance/recovery.

Do not advertise a new fastest profile or silently change production defaults
until the relevant repeated comparisons and device evidence support it.
