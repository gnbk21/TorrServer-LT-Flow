# Adaptive reliability implementation

Specification: the eight improvement areas accepted on 7 October 2026, together
with the original Flow and Modern Web specifications. Existing compatibility,
free-only sources, one authoritative piece cache and explicit device acceptance
remain requirements. No automatic router, firewall, personal configuration or
running-server changes are implied by implementation.

## 1. Qualified delivery
- [ ] Count verified useful forward bytes independently of aggregate traffic.
- [ ] Classify active demand, full/idle, probe, preparation and reconnect periods.
- [ ] Expose short/long rates, freshness, confidence and invalidation after seeks.
- [ ] Keep media demand distinct from bursty HTTP delivery; qualify sustainability
      and exhaustion predictions; preserve unknown evidence.

## 2. Risk controller
- [ ] Explainable buffer risk from contiguous data, useful supply, variation,
      piece waits and fresh supplier evidence, with confidence.
- [ ] Bounded startup/read-ahead adaptation, smoothing, seek invalidation and
      preparation/lower-release guidance; retain native graded deadlines.

## 3. Network recovery
- [ ] Windows interface/route notifications plus portable polling fallback.
- [ ] Sleep/resume and same-address transitions, debounce, bounded jittered retry.
- [ ] Component-specific status/recovery without unnecessary engine restart or
      private-tracker announce storms.

## 4. Resources
- [ ] Global active/warm cache and background-work budget, preserving immediate
      reader reservations and distinguishing disk from RAM.
- [ ] Process/system memory, disk-space and handle observations; pressure limits
      optional work and releases idle reservations before disrupting readers.
- [ ] Preparation scheduling, quota/free-space forecasts and explicit user quota.

## 5. Settings
- [ ] Persistent unsaved indicator, accessible Apply, readable cache units.
- [ ] Draft/saved/effective status, config revision/conflict rejection, data path
      and executable identity.
- [ ] Safe hot application; explicit idle scheduling for disruptive changes.

## 6. Durability
- [ ] Validated last-known-good settings, versioned migration/recovery report.
- [ ] Bounded Windows service crash recovery, no slow-swarm restart watchdog.
- [ ] Interrupted writes, corruption, disk/permission failure and preparation
      recovery tests preserve verified data.

## 7. Optional security
- [ ] Configurable management origins/rate limits and compatibility profile.
- [ ] Expiring playback capabilities with optional stronger access policy.
- [ ] Exposure summary and optional torrent network/interface policy.
- [ ] VPN disconnect enforcement and leak tests before any leak-protection claim;
      account for TCP/UDP/DHT/trackers/DNS/mirrors and preserve local playback.

## 8. Measurement and acceptance
- [ ] Profile CPU/allocations/locks/native work; optimize only measured bottlenecks.
- [ ] Reuse PGO comparison tooling; adopt only with representative evidence.
- [ ] Upload saturation/queue-delay diagnostics and router SQM guidance, no
      unsolicited router changes.
- [ ] Integrate regressions for demand, seeks, reconnects, suppliers, multiple
      readers/preparation, resource bounds, crash/storage/network recovery.
- [ ] Runnable overnight/endurance acceptance and Android/physical network
      checklist; retain exact unperformed environmental gates.
- [ ] Frontend contracts/localization/build/embed, native/platform checks, final
      diff and original-specification reconciliation; runnable Windows artifact.

Deferred by the accepted proposal: engine replacement, QUIC, machine learning,
automatic swarm-profile switching, extra large media cache and unmeasured socket
tuning. Hardware-only acceptance and signing credentials are external gates.

## Evidence

Work in progress. Checked items require implementation and recorded verification.
