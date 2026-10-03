# Startup and retained diagnostics implementation checklist

Scope: the first four next-work items in `PERFORMANCE_AUDIT.md`. Preserve the
native engine, single cache, HTTP correctness, bounded probes, existing settings
migration and independent playback windows.

- [ ] Opt-in structured history: bounded queue, nonblocking producers, fixed
  rotation/disk limits, private allowlisted fields, drop/error counts, bounded
  shutdown and retention across restarts; configuration and documentation.
- [ ] DHT-only native session serialization: bounded validation, corrupt-file
  fallback, atomic private writes, periodic and graceful-stop saves, respect
  disabled DHT/read-only settings; round-trip and recovery verification.
- [ ] Startup explanation: metadata/discovery/first useful data/head-index/
  prebuffer/probe-grace/ready/outcome timings; translated UI; unknown timings
  distinct from measured zero; no fixed player request sequence.
- [ ] Early explicit preload response: release after readiness, preserve a
  bounded background warm handoff, coalesce concurrent same-file preloads,
  supersede other files safely, cancel/join on close, preserve reader priorities
  and reservation ownership; response timing and lifecycle regressions.
- [ ] Native builds, meaningful unit/race/privacy/rotation/corruption tests,
  frontend checks, controlled startup and playback regressions; review diff and
  reconcile original specifications; provide a verified Windows executable.

Real phone request/first-frame comparison, public-swarm cold/warm discovery,
multi-hour playback and physical network/boot tests remain acceptance evidence,
not conclusions from deterministic local tests.
