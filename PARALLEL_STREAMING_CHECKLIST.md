# Parallel streaming implementation checklist

Scope: the compatible scheduler, connection lifecycle and measurement work in
`PARALLEL_STREAMING_EVALUATION.md`, sections A-D, integrated with existing E-H
mechanisms. HTTP/3 and a new peer protocol are outside this implementation.

- [ ] A: bounded startup phase timing; unknown and overlapping phases explicit.
- [ ] A: native request service distributions by transport/address family,
      useful/redundant/obsolete byte counters and recovery/budget observations.
- [ ] A: API, history, support export and translated diagnostics UI integration.
- [ ] B: opt-in native frontier recovery, using existing piece/deadline scheduler.
- [ ] B: two-copy ceiling, per-torrent/session byte/request budgets, receiving
      progress, alternate suppliers, stale/missing-evidence fallback and seeks.
- [ ] C: opt-in per-peer request budget with remote/local limits, service
      variation, paced adjustment/hysteresis and ordinary forward work.
- [ ] D: improve ranked public peer hints and bounded connection reuse; retain
      native TCP/uTP/family fallback and owner binding/transport/private policy.
- [ ] E-H: verify existing reserve/resource/background-work/verified-source
      mechanisms and document measured storage and LAN boundaries.
- [ ] Settings/defaults, frontend contracts, English/Russian and other locales,
      configuration compatibility, documentation and generated embedded UI.
- [ ] Deterministic policy/simulation tests, real native builds, strict remote
      queues, stragglers, cancellation, seek and independent-reader regression.
- [ ] Matched isolated comparisons with one mechanism varied at a time; record
      startup/seek/tail waits, integrity, duplicate cost, CPU/RSS and limitations.
- [ ] Final reconciliation against evaluation; no unproven default promotion,
      owner-server disruption or two-hour tests.
