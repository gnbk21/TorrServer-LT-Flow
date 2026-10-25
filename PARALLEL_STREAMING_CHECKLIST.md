# Parallel streaming implementation checklist

Scope: the compatible scheduler, connection lifecycle and measurement work in
`PARALLEL_STREAMING_EVALUATION.md`, sections A-D, integrated with existing E-H
mechanisms. HTTP/3 and a new peer protocol are outside this implementation.

- [x] A: bounded startup phase timing; unknown and overlapping phases explicit.
- [x] A: native request service distributions by transport/address family,
      useful/redundant/obsolete byte counters and recovery/budget observations.
- [x] A: API, history, support export and translated diagnostics UI integration.
- [x] B: opt-in native frontier recovery, using existing piece/deadline scheduler.
- [x] B: two-copy ceiling, per-torrent/session byte/request budgets, receiving
      progress, alternate suppliers, stale/missing-evidence fallback and seeks.
- [x] C: opt-in per-peer request budget with remote/local limits, service
      variation, paced adjustment/hysteresis and ordinary forward work.
- [x] D: improve ranked public peer hints and bounded connection reuse; retain
      native TCP/uTP/family fallback and owner binding/transport/private policy.
- [x] E-H: verify existing reserve/resource/background-work/verified-source
      mechanisms and document measured storage and LAN boundaries.
- [x] Settings/defaults, frontend contracts, English/Russian and other locales,
      configuration compatibility, documentation and generated embedded UI.
- [x] Deterministic policy/simulation tests, real native builds, strict remote
      queues, stragglers, cancellation, seek and independent-reader regression.
- [x] Matched isolated comparisons with one mechanism varied at a time; record
      startup/seek/tail waits, integrity, duplicate cost, CPU/RSS and limitations.
- [x] Final reconciliation against evaluation; no unproven default promotion,
      owner-server disruption or two-hour tests.

Implementation details and boundaries: [guide](PARALLEL_STREAMING_IMPLEMENTATION.md).
Final source 9e99c8d0 passed native regression, all eight platform builds,
72 matched comparison trials, four network impairment cases and CodeQL analysis.
Local Windows independent-viewer and ignored-cancellation checks also passed.
Evidence, the isolated fuzz retry and remaining phone/public-swarm limits are
recorded in [the verification record](PARALLEL_STREAMING_VERIFICATION.md).
Neither experiment is promoted; field validation is required before recommending
it for ordinary playback. No owner installation or settings were changed.
