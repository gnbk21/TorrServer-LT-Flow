# Sparse-swarm implementation ledger

Source specification: `../SPARSE_STREAMING_EVALUATION.md`, read in full, with the
original Flow and Modern Web plans as architectural constraints. Work starts
from `738649bb`. This ledger records implementation and evidence separately.

## Required work

- [x] Pin native libtorrent v2.1.2 revision; deterministic queue-unit backport
  with upstream simulation tests; invalidate stale dependency caches.
- [x] Reproduce ten-peer unique-supplier scheduling, smaller swarms and multiple
  rare suppliers. The baseline requested eligible suppliers promptly; the report
  makes a native filter patch conditional on reproducing a delay, so no speculative
  filter change was adopted.
- [x] Extend generated peer fixtures: partial availability, dynamic HAVE,
  choke/unchoke, independent suppliers, intermittent outages and metadata.
- [x] Bounded asynchronous peer/piece diagnostics, useful supplier counts,
  queues, failures, duplicates, source summaries, freshness and wait reasons.
- [x] Wire diagnostics into API, translated UI, private history and contracts.
- [x] Size/rate-aware graded deadlines, bounded urgent/ordinary work, scarce
  near-window hints, seek/probe/multiple-reader cleanup; correct comments.
- [x] Adapt buffering to delivery variation/outages/contiguous progress while
  respecting confidence, full-buffer idle, user caps and hysteresis.
- [x] Stable peer port with explicit override, native mapping/listener status,
  incoming connections and transport/route diagnosis; no dashboard exposure.
- [x] Native bounded useful-peer resume hints, expiry/backoff/network recovery,
  preserving private policy; no arbitrary slow-peer disconnection.
- [x] Private metadata guard, deferred public tracker injection for unknown
  metadata, tracker tier preservation and owned-tracker behavior tests.
- [x] Hash-verified exact-length disk resume, incomplete-write recovery.
- [x] Durable selected-file preparation using existing piece storage: quota,
  retained pieces, progress, separate job lifetime, restart recovery, cancellation,
  cleanup and playback readiness; API, UI and settings integration.
- [x] Native web-seed import and explicit management: exact-byte validation,
  Range/mismatch/redirect/retry/fallback tests, URL and secret protection.
- [x] Explicit alternative-release/lower-bitrate guidance and selection without
  mixing encodes or changing existing stream identity behind the player.
- [x] Upload-policy preservation/headroom guidance, metadata/discovery help and
  accurate descriptions of existing protocols and deferred technologies.
- [x] Controlled matrix: rates, complementary peers, rare suppliers, late
  availability, late metadata, network interruptions, media/seeks/multi-reader,
  private trackers, disk failure/recovery/quota and web seeds.
- [x] Unit/race/native/frontend/browser/build gates, reviewed final diff,
  platform builds, documentation, executable and specification reconciliation.

## Boundaries

The report evaluates but does not recommend introducing QUIC, application DoH,
AI prediction, transcoding or another engine without evidence. Full pure-v2
support is a separately scoped compatibility project. Optional account-backed
providers are excluded by the user's free-only requirement; no account is needed.
Physical NAT/IPv6, representative public swarms, Android first-frame and endurance
checks remain environmental acceptance and must not be reported as local tests.

## Specification reconciliation

| Evaluation section | Implemented behavior / evidence |
| --- | --- |
| 1 — Assessment | Piece-aware wait reasons and explicit connected-peer scope; complementary-peer fixture; no global-dead or missing-content recovery claim. |
| 2 — Existing mechanisms | Existing engine, cache, DHT/PEX/transports, warm sessions and preload handoff retained; existing playback/startup/resource gates remain in CI; Legacy remains default. |
| 3 — Upstream patches | Exact reviewed release revision plus upstream queue arithmetic/tests; source-drift and hardlink guards; native and CGo cache stamps include the patch recipe; existing storage extensions retained. |
| 4 — Supplier filtering | Owned ten-peer, small-swarm, multiple/choked/intermittent supplier cases. Ordinary native picking requested eligible rare suppliers promptly, so the conditional filter rewrite was not justified. |
| 5 — Diagnostics | Cached asynchronous bounded windows/peers, availability and useful suppliers, queues/failures/duplicates/sources/freshness; actual response demand tracks a blocked seek separately from delivered progress; API/contracts, seven locales and numeric private history; no full bitfield or peer identity export. |
| 6 — Demand | Qualified bytes/rate deadlines preserve blocked zero, forward work, pins, independent readers and seek cleanup. Mixed repeated results keep the graded ramp default and rate-aware timing opt-in; scarce hints remain independently opt-in. Timing and duplicate observations do not promise ordered completion. |
| 7 — Connectivity | Stable new-default peer port, saved/CLI overrides, native mapping/incoming/transport feedback, public discovery policy and bounded opt-in native useful-peer hints; profile comparisons retain native retry behavior. Physical routing remains a separate gate. |
| 8 — Private policy | Canonical tracker tiers, public injection deferred until known-public metadata, authorized native failover and no extra private peer restoration/sources; owned private tracker tests. |
| 9 — Buffer and preparation | Positive delivery variation, genuine blocked outages and full-buffer idle distinguished; existing caps/hysteresis; independent jobs, quota, verified retention/readiness, pause/cancel/resume/removal, root persistence, write/sync/cleanup errors and crash recovery. Upload limits and identity preserved. |
| 10 — Additional sources | Native exact-byte HTTP seeds, explicit URL management, destination/redirect checks and privacy protections; clear alternative-release guidance. Account providers excluded by free-only preference; speculative technologies remain deferred as the report specifies. |
| 11 — Verification | Generated partial peers, metadata, rate variation and wire checks; byte-checked media/seeks/multiple readers, private trackers, disk recovery/quota/failure, mirrors and hints; repeated isolated native layers/profiles/experiment. HTTP waits are labelled as server observations, not decoder stalls. |
| 12 — Order | Native baseline and reproducible supplier fixtures preceded runtime policy, connectivity, preparation and sources; no floating dependency, second media cache or separate peer killer was introduced. |

## Verification evidence

- Native pin: v2.1.2 / `6da363d2994f17c0b3c0450d124cf73a31a73847`.
- Exact queue patch applies and a second application makes no change.
- Two real patch regressions passed: shared hardlinks remain unchanged, source
  drift fails closed. Cross-platform compilation and simulation execution passed for `d7aafde5`.

- Baseline `d7aafde5`: upstream queue/transfer simulations, all six Linux/Windows/
  Android target builds, both macOS architectures and Windows service gates passed
  in runs `37197114818` and `37197114822`.
- Four real fixture wire tests passed locally: complementary partial suppliers,
  delayed HAVE/unchoke, delayed BEP 9 metadata, cancel/outage/reconnect.
- Exact-length/hash resume verifier and wait-reason tests passed with the full
  portable Flow suite. Frontend: 44 passed, one existing skip; typecheck, lint
  and production build passed at that early checkpoint. Later native privacy,
  storage and snapshot coverage passed in the complete `997e18f5` runtime gate.
- Diagnostics are bounded to one pending native request per torrent, once per
  two seconds, eight windows/256 pieces/512 peers. HTTP reads use cached aggregates;
  stale/truncated results never establish absence. No identities are emitted.
- Resume uses exact metadata SHA-1 hashes and lengths, with a shared verified
  bitmap for native and Go cache ownership. Unknown metadata never trusts files.
- Account-backed providers are omitted by explicit free-only preference. Native
  sources, HTTP mirrors and local preparation remain the intended product scope.

## Windows verification history

- `ba5631fd`: all 15 sparse cases passed with exact Range bytes: rates,
  complementary peers, unique/multiple/choked/intermittent suppliers, late HAVE,
  choke, outages, late metadata, latency and variable delivery. Generated fixture
  SHA-256 `9f882c068f1d1affd2128abb9d069a5fa041cbb29347eb0ce47c72ae5fe407bf`.
- Native mirrors: identical bytes, relative redirects, hash-mismatch rejection
  with peer fallback, 503 fallback and blocked DNS-to-LAN import all passed;
  disabled imported sources remained disabled after abrupt restart. The fixture
  uses distinct mirror and peer IPs so native corruption bans can isolate sources.
- Useful public peer hints restored a connection after restart with tracker
  advertisements, DHT and PEX disabled. Peer identities remain private opt-in state.
- `532388c1`: preparation passed pause/cancel/resume, verified retention beyond
  cache capacity, abrupt restart, corrupted-piece repair, source-free playback,
  retained root after disk-path change, active-reader cleanup fence, rejected
  cleanup reversal, explicit removal, quota rejection and disk-write failure.
- Baseline `d7aafde5` and candidate `532388c1` both passed actual prebuffer plus
  paced delivery for below/near/above consumption, latency and rate variation.
  Single-run timings are mixed; they do not establish an overall speed improvement.
- Portable Flow tests, 13 fixture tests, two real native patch application tests,
  workflow validation, frontend typecheck/lint/build, 45 frontend unit passes
  (one contract-export skip locally) and 33 browser cases passed. Native CI supplies
  the actual contract export, race, platform and runtime gates.

## Final acceptance reconciliation

- Small/large 64 KiB and 4 MiB cases passed late HAVE, unique and multiple rare
  suppliers (`532388c1`, three cases per size).
- Repeated native isolation completed at `997e18f5`: 72/72 byte-checked layer
  cases, 27 profile characterizations and 12/12 scarce on/off cases. Conservative
  and Balanced each failed all three intermittent-supplier cases at the HTTP
  waiting boundary; these are recorded as failures. Legacy remains the default.
  [Detailed comparison and policy decision](SPARSE_VERIFICATION.md#repeated-isolated-comparisons).
- The rate-aware deadline proposal remains implemented as an opt-in experiment
  after mixed repeated results. Settings migration, seven translated labels/help,
  frontend save/confirmation and native qualification tests cover that decision.
  Final-source native runtime passed at `3400e54a`; repeated Windows on/off
  revalidation passed 18/18 cases and 54 exact Range checks. Mixed HTTP delivery
  results support retaining the experiment off by default.
- `4fe814d2` closes the availability-to-copy race: HTTP reads hold the same piece
  lock for the mirror verification gate and byte copy. Native hashing retains raw
  access. `0016c944` contains native peer-hint errors. Complete native gates passed
  with these fixes at `997e18f5` and `fb04b171`.
- `72c38082` / `8e1478a8` synchronize handle publication and session lookup,
  preserve immutable native IDs, and join concurrent close calls. `fb04b171`
  rejects detached mirror updates instead of panicking. Native unit/race tests
  now include `./lt/`; the complete `fb04b171` native/platform/runtime/service
  gate passed. Final policy source `3400e54a` also passed complete native,
  macOS, portable and Windows service/update/rollback gates.
- Final storage integration covers library-deletion cleanup, migration of existing
  ordinary disk backends, empty-plan removal and same-file alias protection.
  Windows preparation and all five mirror cases passed again at `3400e54a`.
- `3400e54a` fixes pending-seek diagnostics using the actual response position,
  keeping delivered progress and probe/old-response isolation intact. The owned
  regression detects the old defect, then passes on final Linux and Windows.
  Final CI passed 16 default and three rate-aware sparse cases, 46 frontend
  tests, 34 browser scenarios, native API contracts, race/fuzz/vulnerability
  checks, upstream simulations and all eight platform targets.
- The final Windows executable was installed at the usual local path only while
  stopped; the archive digest, source tree, installed hash and version were
  verified, and the previous executable was preserved. All twelve evaluation
  sections were reread and reconciled against the implementation and evidence.
- Historical baseline linking required whole-token pkg-config parsing; the fixed
  isolation job passed without changing baseline native/runtime behavior.
- Physical NAT/IPv6, multi-hour public swarms, representative real media and
  Android decoded-frame measurements remain explicit environmental acceptance.
  No release or merge is inferred from this development task.
