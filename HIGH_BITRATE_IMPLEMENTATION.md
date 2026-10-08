# High bitrate streaming implementation

Scope: implement the actionable recommendations from the 8 October research,
preserving original quality, free sources, one rolling cache, existing clients,
Legacy as the default and experimental scheduling behind explicit switches.

## Requirements and acceptance

- [ ] Storage: bounded reusable handles; close before deletion, migration and
  cache shutdown; unchanged disk format and verified resume semantics.
- [ ] Native I/O: owned write buffers, bounded workers/backpressure; read/write/
  hash/clear ordering; removal/deletion/shutdown fences; accurate storage errors.
- [ ] Observability: native queue/wait/callback latency, urgent outstanding age,
  readable versus verified contiguous reserve and frontier growth; no identities.
- [ ] Capacity experiment: remote `reqq` and local caps include urgent work;
  default remains unchanged; strict-cap peers and low/high-latency comparisons.
- [ ] Scheduling experiment: demand-sized urgent horizon, hysteresis, ordinary
  nonzero forward work and independent readers; no one-piece starvation or
  repeated global cancellation; retain baseline when evidence is unavailable.
- [ ] Deficit controller: credible demand and clean consecutive intervals only;
  cumulative shortfalls, expiry/reset and bounded startup/read-ahead integration.
- [ ] Burst hints: bounded asynchronous inspection of resident container indexes,
  coarse byte/time estimates, validation/fallback, no full-file scan or open gate.
- [ ] Phone/LAN diagnostics: bounded transfer test and reproducible measurements;
  distinguish HTTP supply, Wi-Fi gaps and decoder/rebuffer events.
- [ ] Optional Just Player changes: reproducible pinned source patch/build,
  memory-aware buffer budget and diagnostics; ordinary player remains supported.
- [ ] Conditional technologies: document measured SQM/upload-headroom procedure,
  existing exact-byte web seeds and lossless remux use; do not alter the router.
  Cross-swarm/v2 reuse is a longer-term candidate requiring end-to-end identity
  and hashing support, not a ready configuration change in this implementation.
- [ ] Integration: settings/defaults/validation, backend/frontend contracts,
  translated diagnostics/controls, documentation and rollback instructions.
- [ ] Verification: meaningful units/races/native lifecycle tests, strict-cap
  fixtures, matched repeated 90/120 Mbps and burst comparisons with equal reserve,
  integrity/seek/cancellation/resource checks, final diff and runnable artifact.

Performance promotion requires repeatable improvement in total blocked time and
read-delay tails without meaningful startup, seek, integrity or resource
regression. Synthetic byte transport is not Android decoding validation. Real
phone playback, physical LAN/router behavior and representative original-quality
media are environmental acceptance and must be recorded separately.

## Execution evidence

Pending. Checkboxes represent verified behavior, not merely code presence.
