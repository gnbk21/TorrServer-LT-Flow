# Sparse-swarm implementation ledger

Source specification: `../SPARSE_STREAMING_EVALUATION.md`, read in full, with the
original Flow and Modern Web plans as architectural constraints. Work starts
from `738649bb`. This ledger records implementation and evidence separately.

## Required work

- [x] Pin native libtorrent v2.1.2 revision; deterministic queue-unit backport
  with upstream simulation tests; invalidate stale dependency caches.
- [ ] Reproduce ten-peer unique-supplier scheduling; retain eligible suppliers
  where the filtered deadline candidate set cannot supply urgent pieces.
- [x] Extend generated peer fixtures: partial availability, dynamic HAVE,
  choke/unchoke, independent suppliers, intermittent outages and metadata.
- [ ] Bounded asynchronous peer/piece diagnostics, useful supplier counts,
  queues, failures, duplicates, source summaries, freshness and wait reasons.
- [ ] Wire diagnostics into API, translated UI, private history and contracts.
- [ ] Size/rate-aware graded deadlines, bounded urgent/ordinary work, scarce
  near-window hints, seek/probe/multiple-reader cleanup; correct comments.
- [ ] Adapt buffering to delivery variation/outages/contiguous progress while
  respecting confidence, full-buffer idle, user caps and hysteresis.
- [ ] Stable peer port with explicit override, native mapping/listener status,
  incoming connections and transport/route diagnosis; no dashboard exposure.
- [ ] Native bounded useful-peer resume hints, expiry/backoff/network recovery,
  preserving private policy; no arbitrary slow-peer disconnection.
- [ ] Private metadata guard, deferred public tracker injection for unknown
  metadata, tracker tier preservation and owned-tracker behavior tests.
- [ ] Hash-verified exact-length disk resume, incomplete-write recovery.
- [ ] Durable selected-file preparation using existing piece storage: quota,
  retained pieces, progress, separate job lifetime, restart recovery, cancellation,
  cleanup and playback readiness; API, UI and settings integration.
- [ ] Native web-seed import and explicit management: exact-byte validation,
  Range/mismatch/redirect/retry/fallback tests, URL and secret protection.
- [ ] Explicit alternative-release/lower-bitrate guidance and selection without
  mixing encodes or changing existing stream identity behind the player.
- [ ] Upload-policy preservation/headroom guidance, metadata/discovery help and
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

## Evidence so far

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
