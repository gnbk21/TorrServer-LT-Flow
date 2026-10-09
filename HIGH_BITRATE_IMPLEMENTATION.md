# High bitrate streaming implementation

Scope: implement the actionable recommendations from the 8 October research,
preserving original quality, free sources, one rolling cache, existing clients,
Legacy as the default and experimental scheduling behind explicit switches.

## Requirements and acceptance

- [x] Storage: bounded reusable handles; close before deletion, migration and
  cache shutdown; unchanged disk format and verified resume semantics.
- [x] Native I/O: owned write buffers, bounded workers/backpressure; read/write/
  hash/clear ordering; removal/deletion/shutdown fences; accurate storage errors.
- [x] Observability: native queue/wait/callback latency, urgent outstanding age,
  readable versus verified contiguous reserve and frontier growth; no identities.
- [x] Capacity experiment: remote `reqq` and local caps include urgent work;
  default remains unchanged; strict-cap peers and low/high-latency comparisons.
- [x] Scheduling experiment: demand-sized urgent horizon, hysteresis, ordinary
  nonzero forward work and independent readers; no one-piece starvation or
  repeated global cancellation; retain baseline when evidence is unavailable.
- [x] Deficit controller: credible demand and clean consecutive intervals only;
  cumulative shortfalls, expiry/reset and bounded startup/read-ahead integration.
- [x] Burst hints: bounded asynchronous inspection of resident container indexes,
  coarse byte/time estimates, validation/fallback, no full-file scan or open gate.
- [x] Phone/LAN diagnostics: bounded transfer test and reproducible measurements;
  distinguish HTTP supply, Wi-Fi gaps and decoder/rebuffer events.
- [x] Optional Just Player changes: reproducible pinned source patch/build,
  memory-aware buffer budget and diagnostics; ordinary player remains supported.
- [x] Conditional technologies: document measured SQM/upload-headroom procedure,
  existing exact-byte web seeds and lossless remux use; do not alter the router.
  Cross-swarm/v2 reuse is a longer-term candidate requiring end-to-end identity
  and hashing support, not a ready configuration change in this implementation.
- [x] Integration: settings/defaults/validation, backend/frontend contracts,
  translated diagnostics/controls, documentation and rollback instructions.
- [x] Verification: meaningful units/races/native lifecycle tests, strict-cap
  fixtures, matched repeated 90/120 Mbps and burst comparisons with equal reserve,
  integrity/seek/cancellation/resource checks, final diff and runnable artifact.

Performance promotion requires repeatable improvement in total blocked time and
read-delay tails without meaningful startup, seek, integrity or resource
regression. Synthetic byte transport is not Android decoding validation. Real
phone playback, physical LAN/router behavior and representative original-quality
media are environmental acceptance and must be recorded separately.

## Execution evidence (9 October 2026)

The final code revision is `52d0ad2087df111611c58b7f917d1493babb9b70`.
All implementation gates passed in
[native/platform run 37825980737](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37825980737)
and [macOS run 37825985829](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37825985829).
The subsequent evidence update changes documentation only.

- Native patch checks and upstream queue/deadline simulations passed. Storage
  lifecycle fixtures passed ASan/UBSan: owned buffers, short I/O, hashing,
  piece-ordered clears, deletion/removal, concurrent shutdown and callbacks
  surviving backend destruction. Final review fixed portable error categories
  and clear ordering at the saturated ordinary job cap; regressions passed.
- Full Linux Go units, vet, actual backend/frontend contracts, race tests,
  reachable vulnerability scan and bounded fuzz campaigns passed. Resident
  MKV inspection reaches Adaptive without fetching missing media or creating
  readers; atomic resident verification/copy prevents the reviewed race.
- Frontend type checking, lint, build and browser gates passed. Local component
  tests passed 58 cases; CI supplies the actual Go export required by the one
  locally skipped contract test. LAN timeout/cancellation have regression tests.
- Cross builds passed Windows amd64, Linux amd64/arm64/armv7, Android arm64/armv7
  and macOS amd64/arm64. Windows notification/handle checks and disposable SCM
  install, crash restart, update failure and rollback gates passed.
- All 13 final Linux executable reports passed: playback (3 cases), discovery
  recovery (4), strict request capacity (12), sparse streaming (16), experimental
  rate-aware cases (3), exact-byte mirrors (5), upstream TLS (5), adaptive access,
  retained peer hints, maintenance, preparation and both startup profiles.
  All strict-cap cases delivered correct bytes, respected advertised capacity
  and left no readers after cancellation; every delayed case observed real
  outstanding urgent age. One immediately served cap-32 case did not need a
  positive age observation.
- The downloaded final Windows executable passed retained-cache crash recovery,
  corruption repair, offline resume/Range reads, storage-root migration,
  active-reader cleanup fencing, quota/disk rejection and authenticated LAN
  payload verification. Tests used fresh owned loopback state/processes and
  stopped them. SHA-256:
  `005601f1f750cd2488ce89779c98ebde3cd5f138950cf92a6c0328f269757d7c`.
- Local pure Go policies/parser/handle checks passed, including concurrent
  descriptor reuse, safe upgrades and Windows writable flush after read-only
  resume. Parser fuzzing completed 155,422 executions. Controlled wire fixtures
  passed 17 tests. Earlier Windows strict-cap checks passed 10 normal and two
  large-piece cases; a separate Conservative case reached and respected the
  actual local 500-request cap against a peer advertising 2000.
- Optional Flow Player APK builds passed on `8148da9e` and `e11ea26d`; final server
  corrections do not change player code. The downloaded `8148da9e` APK matches
  the CI checksum. Pinned patch repeatability/drift refusal and five Java memory
  policy checks passed. This verifies building, not Android playback.

### Repeated transport comparisons

The final Linux matrix passed **72/72 two-minute trials**: eight configurations,
three policies, three rotated repetitions, equal 32 MiB startup reserve and
identical generated bytes. Configurations cover 90/120 Mbps, healthy/mixed/
outage/burst suppliers, 512/2048 MiB RAM/disk caches and 1/4/16 MiB pieces.
Passing means byte integrity, ranges, seeks, cancellation and cache bounds
(configured budget plus the harness's two-piece boundary allowance);
it does not mean uninterrupted decoded playback.

Median total time spent in individual reads longer than 250 ms, seconds:

| Scenario | Cache / pieces | Legacy | Adaptive | Combined experiments |
| --- | --- | ---: | ---: | ---: |
| 90 Mbps healthy | 512 MiB RAM / 1 MiB | 8.0 | 12.1 | 0.0 |
| 90 Mbps bursts | 512 MiB RAM / 4 MiB | 0.0 | 0.0 | 1.1 |
| 90 Mbps mixed peers | 2048 MiB RAM / 4 MiB | 0.3 | 1.3 | 0.0 |
| 90 Mbps outages | 512 MiB disk / 4 MiB | 26.1 | 24.2 | 21.4 |
| 120 Mbps healthy | 2048 MiB disk / 16 MiB | 0.0 | 0.3 | 0.0 |
| 120 Mbps bursts | 2048 MiB disk / 4 MiB | 47.5 | 39.0 | 2.5 |
| 120 Mbps mixed peers | 512 MiB disk / 4 MiB | 0.8 | 1.5 | 0.0 |
| 120 Mbps outages | 512 MiB RAM / 4 MiB | 18.2 | 12.7 | 15.2 |

Results are mixed. Combined experiments reduce waits in several scenarios but
regress the 90 Mbps burst case; Adaptive alone is not consistently better than
Legacy. Outages still cause long waits. These results do not meet promotion
requirements. Synthetic HTTP read waits are not Just Player rebuffer seconds.

Median observed RSS is approximately 84–94 MiB in disk cases, 1144–1201 MiB in
512 MiB RAM-cache cases and 2249–2316 MiB in 2048 MiB RAM-cache cases. Configured
cache size is not a whole-process memory limit: native/Go buffers and allocator
retention cost additional memory. Two-minute samples cannot establish multi-hour
memory convergence. All resource distributions are retained, not just best runs.

Earlier matched Windows comparisons passed all 16 trials on `5378a444`:
two rotated repetitions of previous/current Legacy, Adaptive and combined
experiments, at 90 Mbps with 512 MiB RAM and 120 Mbps with 2048 MiB disk/
16 MiB pieces/outages. Every outage policy still had long reads. Those executable
hashes remain separate from the final source. Complete distributions, report
hashes, artifact provenance and limits are recorded in
[HIGH_BITRATE_OPTIMIZATION_EVIDENCE.json](HIGH_BITRATE_OPTIMIZATION_EVIDENCE.json).

## Remaining environmental acceptance and deferred work

- Real original-quality Lampa/Just Player playback, captured player Range traces,
  Android decoder/memory pressure, physical Wi-Fi/router measurements and
  multi-hour resource convergence remain environmental acceptance. Synthetic
  loopback delivery cannot establish those original-plan acceptance targets.
- Legacy stays default; the three experiments stay off. No superiority or
  uninterrupted public-swarm playback claim is made. Published Preview 2 `.5`
  remains unchanged; stable release approval remains false.
- Cross-swarm/v2 reuse is deferred until end-to-end identity, mapping, resume
  verification and SHA-256 callbacks exist. SQM/upload shaping and lossless
  remux are conditional operator procedures documented in the guide; this work
  changes no router and converts no media. No paid provider is required.
