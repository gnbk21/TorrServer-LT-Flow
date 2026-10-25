# Parallel streaming verification

Date: 2026-10-25. Final validation source: 9e99c8d0 on
improve/reliability-usability. Runtime implementation: e682e79e and its preceding
two commits; subsequent changes extend tests and documentation only. The installed
Flow 1.1.5 server was not changed.

## Early Windows comparison

The initial native candidate, 858c32e1, was downloaded from successful build
[40119196051](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40119196051).
Executable SHA-256:
a1802ada66010008b98a9c1de109610c80cdb83e65fec1259c1429f093f973bc.

The local delayed-block comparison used two rotated trials per policy, a generated
192 MiB byte fixture, 8 MiB pieces, 64 MiB disk cache, a 4 MiB startup reserve,
strict remote request queues and 33.55 Mbps paced demand. Calibration measured
67.09 Mbps fixture supply. Each trial streamed for 12 seconds and checked cold
seeks and cancellation. These are transport measurements, not decoded playback.

| Policy | Passed | Median preload ms | Median trial cold-seek P95 ms | Median redundant KiB | Median CPU seconds |
| --- | --- | ---: | ---: | ---: | ---: |
| Capacity prerequisite | 2/2 | 9221 | 1027 | 112 | 0.66 |
| Frontier recovery | 2/2 | 9020 | 554 | 176 | 0.73 |
| Adaptive request credit | 2/2 | 5415 | 373 | 56 | 1.29 |
| Both controls | 2/2 | 5418 | 380 | 64 | 0.70 |

No trial had a paced read lasting 250 ms. Peak observed RSS was 74–76 MiB.
Recovery counters were nonzero in both frontier trials (4 and 18 requests) and
both combined trials (3 and 5). Request-credit changes were nonzero in both
request-credit trials and both combined trials. Thus the policies actually ran.
The CPU variation and small trial count do not justify a new default.

Evidence: .tools/parallel-local-858c-stragglers/report.json. This earlier binary
predates the final accounting and strengthened admission changes; its results
must not be presented as measurements of the final executable.

With both controls enabled, the same candidate also passed:

- Two simultaneous independent session tokens, reading 12 MiB each from distant
  offsets of the same torrent, with byte-exact results and strict 64-slot peers.
  Both session groups were observed concurrently. First-byte times were 5.45 and
  6.06 seconds; maximum subsequent sampled read waits were below 1 ms.
- Cold A/B/A cancellation with normal replies and with non-FAST replies arriving
  after cancellation. The late-reply race was observed and capacity stayed valid.

Evidence: .tools/parallel-local-858c-viewers/report.json and
.tools/parallel-local-858c-cancel/report.json.

## Final Windows runtime checks

The final Windows runtime was downloaded from the successful platform build in
[40127402169](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40127402169).
Executable SHA-256:
c17a35c822a7569361639192c6c5fdd39d203706c998cd5db8363d394f76bef2.
The server and frontend source trees are identical between this e682e79e build
and final validation source 9e99c8d0; later commits change test coverage only.

Independent-viewer and ignored-cancellation checks passed again on this runtime.
Both viewers received 12 MiB of byte-exact data from distant positions of the
same torrent, with both session groups observed concurrently. First-byte times
were 5.51 and 6.33 seconds; the maximum subsequent sampled waits were 33.7 and
2.94 ms. Strict 64-slot peer queues were respected. Normal cancellation and
observed non-FAST late completion both passed the cold A/B/A regression.

Evidence: .tools/parallel-local-e682-viewers/report.json and
.tools/parallel-local-e682-cancel/report.json.

## Final source gates

The a718b68c comparison reports demonstrate the following medians. Each policy
has three rotated trials per case; the compiled runtime source is identical to
the final source. These values are observations, not promotion thresholds.

| Fixture | Policy | Preload ms | Trial cold-seek P95 ms | Read waits ≥250 ms, total ms | CPU seconds |
| --- | --- | ---: | ---: | ---: | ---: |
| Linux, 100.66 Mbps healthy | Capacity | 5004 | — | 0 | 2.45 |
| Linux, 100.66 Mbps healthy | Request credit | 4804 | — | 0 | 2.66 |
| Linux, delayed blocks | Capacity | 6206 | 649 | 0 | 1.22 |
| Linux, delayed blocks | Frontier | 5610 | 1298 | 0 | 1.16 |
| Linux, delayed blocks | Request credit | 5409 | 309 | 0 | 1.25 |
| Linux, delayed blocks | Both | 5409 | 302 | 0 | 1.22 |
| Windows, delayed blocks | Capacity | 8412 | 731 | 0 | 0.56 |
| Windows, delayed blocks | Both | 5216 | 358 | 0 | 2.00 |
| Linux, mixed peers | Capacity | 19405 | — | 1681 | 0.65 |
| Linux, mixed peers | Request credit | 11806 | — | 0 | 0.85 |
| Linux, mixed peers | Both | 11806 | — | 0 | 0.84 |

Frontier alone worsened the Linux cold-seek result in this fixture; the combined
Windows case consumed more CPU despite faster startup/seek. These costs prevent
a blanket claim that either switch is faster or cheaper. The ordinary healthy
case issued no extra recovery requests and recorded zero redundant bytes for all
four policies. The request-budget counters and delayed-block recovery counters
confirm that the new controls were exercised in the appropriate cases.

All twelve a718b68c small-queue trials passed. Its subsequent network check exposed
a harness race: a fast transfer ended before the asynchronous Go/native sampling
path published its first snapshot. 9e99c8d0 explicitly enables metrics and waits
up to five seconds before fault injection; missing observations still fail.
Two unit tests cover delayed publication and the unchanged failure condition.
The failed original report remains in .tools/parallel-final-evidence/small-queue-failure.

Final source 9e99c8d0 passed these completed runs:

- [Native build and regressions, 40130189531](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40130189531):
  all six non-Mac targets, native simulations, Linux unit/race/vet/fuzz and
  vulnerability checks, storage sanitizers, frontend/contracts/browser/lab gates,
  Windows service/update recovery, eight controlled functional groups, two native
  profiling captures, inventory, ten existing high-bitrate comparison jobs and
  all six new parallel comparison jobs passed.
- [macOS, 40130189500](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40130189500):
  Intel and ARM builds and their version probes passed.
- [CodeQL, 40130189507](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40130189507):
  all analysis categories passed. Its first native job reached a 10.063-second
  FuzzInfoHash context deadline after 222,278 executions, without a reported
  failing input. Only that failed job was retried; it then passed fuzzing and
  analysis without source changes. This does not establish the original timeout's
  root cause or clear existing repository security findings; see the existing
  RELIABILITY_USABILITY_SECURITY_REVIEW.md dispositions.
- [Branch policy, 40130189498](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40130189498):
  passed. Optional diagnostic/upgrade matrices and Docker publication were not
  requested and were skipped; they are not counted as passing checks.

The final six comparison reports contain 72/72 passing trials: three rotations
of each of four policies per case. Final representative medians are below;
neither these nor the earlier results select a default.

| Fixture | Policy | Preload ms | Trial cold-seek P95 ms | Read waits ≥250 ms, total ms | CPU seconds |
| --- | --- | ---: | ---: | ---: | ---: |
| Linux, 100.66 Mbps healthy | Capacity | 5004 | — | 0 | 2.43 |
| Linux, 100.66 Mbps healthy | Request credit | 4804 | — | 0 | 2.75 |
| Linux, delayed blocks | Capacity | 6208 | 478 | 0 | 1.24 |
| Linux, delayed blocks | Both | 5409 | 222 | 0 | 1.42 |
| Windows, delayed blocks | Capacity | 9208 | 778 | 0 | 0.97 |
| Windows, delayed blocks | Both | 5213 | 497 | 0 | 0.81 |
| Linux, mixed peers | Capacity | 19405 | — | 682 | 0.42 |
| Linux, mixed peers | Both | 12204 | — | 0 | 0.59 |
| Linux, outages | Capacity | 19604 | — | 24141 | 0.98 |
| Linux, outages | Both | 11804 | — | 0 | 1.06 |
| Linux, small queue | Capacity | 20603 | — | 0 | 0.22 |
| Linux, small queue | Frontier | 22604 | — | 0 | 0.22 |
| Linux, small queue | Both | 16604 | — | 0 | 0.23 |

Frontier alone slowed small-queue startup, and adaptive credit used more CPU in
several Linux cases. The earlier seek regression and Windows CPU variation remain
relevant evidence against a blanket improvement claim.

Final Linux and Windows cancellation reports each passed normal and observed
late-reply cancellation. The separate TCP namespace check passed byte-exact
impaired and recovered Ranges under jitter, configured 1% loss, reordering and a
two-second outage, with both controls enabled. Recorded transfer times were
502, 502, 402 and 3814 ms respectively. Fixed netem seeding was unavailable
(seed_supported=false). Qdisc counters can be cumulative; the outage can retain
reordering, so these are not claims of four independent identical network paths.

Final CI reports and executable SHA-256 identities are retained under
.tools/parallel-final-complete-9e99/; the compact aggregation is
.tools/parallel-final-receipt-summary.json. The per-trial reports retain resource,
integrity and duplicate-cost measurements, including failures from earlier runs.

Locally, 91 Python policy tests, the portable Go flow package, workflow lint and
embedded asset consistency checks passed. The embedded table has 54 existing
assets and contains both new settings. The frontend suite previously passed
129 tests with one skipped test, plus type checking, lint and production build;
the final CI repeats the frontend gates. The existing large HLS chunk warning
remains a build advisory.

The native simulation job for final source 9e99c8d0 passed. It includes service
freshness, clock rollback, paced credit/hysteresis, cancellation identity filters,
per-torrent and shared session credit, copied ownership and release behavior.

## Interpretation and remaining field validation

The comparison policies all share Legacy and the existing capacity guard so that
each new control can be varied independently. Reports retain failures and record
integrity, startup, cold seeks, total/worst read waits, lateness, duplicate cost,
CPU, RSS and storage observations. Safety gates do not automatically promote a
performance policy. A promotion threshold has not been selected, and neither
experiment is promoted.

Synthetic loopback, native and network-impairment checks cannot establish public
swarm availability, the owner's ISP capacity, phone Wi-Fi behavior or decoder
stutters. Fresh Lampa/Just Player field confirmation is still needed before
recommending either experiment for the owner's normal playback. No two-hour
test was started or required.
