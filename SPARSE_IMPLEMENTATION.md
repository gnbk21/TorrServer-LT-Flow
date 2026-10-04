# Sparse-swarm implementation ledger

Source specification: `../SPARSE_STREAMING_EVALUATION.md`, read in full, with the
original Flow and Modern Web plans as architectural constraints. Work starts
from `738649bb`. This ledger records implementation and evidence separately.

## Required work

- [ ] Pin native libtorrent v2.1.2 revision; deterministic queue-unit backport
  with upstream simulation tests; invalidate stale dependency caches.
- [ ] Reproduce ten-peer unique-supplier scheduling; retain eligible suppliers
  where the filtered deadline candidate set cannot supply urgent pieces.
- [ ] Extend generated peer fixtures: partial availability, dynamic HAVE,
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
  drift fails closed. Cross-platform compilation and simulation execution pending.
