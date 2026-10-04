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
- [ ] Controlled matrix: rates, complementary peers, rare suppliers, late
  availability, late metadata, network interruptions, media/seeks/multi-reader,
  private trackers, disk failure/recovery/quota and web seeds.
- [ ] Unit/race/native/frontend/browser/build gates, reviewed final diff,
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
| 5 — Diagnostics | Cached asynchronous bounded windows/peers, availability and useful suppliers, queues/failures/duplicates/sources/freshness; API/contracts, seven locales and numeric private history; no full bitfield or peer identity export. |
| 6 — Demand | Qualified bytes/rate deadlines preserve blocked zero, forward work, pins, independent readers and seek cleanup; bounded scarce hint remains opt-in; timing and duplicate observations are recorded without promising ordered completion. |
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
  and production build passed. Native privacy/storage/snapshot changes await CI.
- Diagnostics are bounded to one pending native request per torrent, once per
  two seconds, eight windows/256 pieces/512 peers. HTTP reads use cached aggregates;
  stale/truncated results never establish absence. No identities are emitted.
- Resume uses exact metadata SHA-1 hashes and lengths, with a shared verified
  bitmap for native and Go cache ownership. Unknown metadata never trusts files.
- Account-backed providers are omitted by explicit free-only preference. Native
  sources, HTTP mirrors and local preparation remain the intended product scope.

## Current real Windows evidence

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

## Remaining acceptance reconciliation

- The 64 KiB and 4 MiB comparisons passed late HAVE, unique and multiple rare
  suppliers (`532388c1`, three cases per size). Finish repeated bounded scarce
  experiment on/off checks. The experiment is disabled by default and has no
  established public-swarm speed benefit.
- Await final native CI gates and the explicit isolated-layer job: original
  v2.1.0 runtime, v2.1.2 alone, exact queue backport and current Flow runtime.
  The job repeats identical owned fixtures three times in varied order.
- `4fe814d2` closes the availability-to-copy race: HTTP reads recheck the mirror
  verification gate while holding the same piece lock as the copy. Native hash
  callbacks retain raw access to the failed backend. `0016c944` also contains
  peer-hint errors inside the native API boundary. Final native gates apply to
  these revisions; earlier Windows results do not establish this last fix.
- Reconcile the complete evaluation against final results and deliver the tested
  executable. No release or merge is inferred from this development task.
- Final integration review also added library-deletion cleanup, migration of
  existing ordinary disk backends to the retained root, empty-plan removal and
  same-file alias protection. Runtime coverage exercises deletion during playback
  and verifies that the durable job cannot resurrect its torrent.
- The first isolated-layer attempt reached the baseline link and exposed an
  inherited pkg-config parser matching `-l` inside a directory name. Whole-token
  parsing fixes that build error without changing baseline runtime/native source.
  Its regression passes locally; the comparison must be rerun before drawing a
  performance conclusion.
