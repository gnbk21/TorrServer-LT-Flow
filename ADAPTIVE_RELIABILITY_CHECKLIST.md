# Adaptive reliability implementation

Specification: the eight improvement areas accepted on 7 October 2026, together
with the original Flow and Modern Web specifications. Existing compatibility,
free-only sources, one authoritative piece cache and explicit device acceptance
remain requirements. No automatic router, firewall, personal configuration or
running-server changes are implied by implementation.

## 1. Qualified delivery
- [x] Count verified useful forward bytes independently of aggregate traffic.
- [x] Classify active demand, full/idle, probe, preparation and reconnect periods.
- [x] Expose short/long rates, freshness, confidence and invalidation after seeks.
- [x] Keep media demand distinct from bursty HTTP delivery; qualify sustainability
      and exhaustion predictions; preserve unknown evidence.

## 2. Risk controller
- [x] Explainable buffer risk from contiguous data, useful supply, variation,
      piece waits and fresh supplier evidence, with confidence.
- [x] Bounded startup/read-ahead adaptation, smoothing, seek invalidation and
      preparation/lower-release guidance; retain native graded deadlines.

## 3. Network recovery
- [x] Windows interface/route notifications plus portable polling fallback.
- [x] Sleep/resume and same-address transitions, debounce, bounded jittered retry.
- [x] Component-specific status/recovery without unnecessary engine restart or
      private-tracker announce storms.

## 4. Resources
- [x] Global active/warm cache and background-work budget, preserving immediate
      reader reservations and distinguishing disk from RAM.
- [x] Process/system memory, disk-space and handle observations; pressure limits
      optional work and releases idle reservations before disrupting readers.
- [x] Preparation scheduling, quota/free-space forecasts and explicit user quota.

## 5. Settings
- [x] Persistent unsaved indicator, accessible Apply, readable cache units.
- [x] Draft/saved/effective status, config revision/conflict rejection, data path
      and executable identity.
- [x] Safe hot application; explicit idle scheduling for disruptive changes.

## 6. Durability
- [x] Validated last-known-good settings, versioned migration/recovery report.
- [x] Bounded Windows service crash recovery, no slow-swarm restart watchdog.
- [x] Interrupted writes, corruption, disk/permission failure and preparation
      recovery tests preserve verified data.

## 7. Optional security
- [x] Configurable management origins/rate limits and compatibility profile.
- [x] Expiring playback capabilities with optional stronger access policy.
- [x] Exposure summary and optional torrent network/interface policy.
- [x] Missing-interface suspension with cached local playback preserved; no
      automatic unbound torrent fallback. Source binding is not a firewall.
- [ ] VPN disconnect enforcement and leak tests before any leak-protection claim;
      account for TCP/UDP/DHT/trackers/DNS/mirrors and preserve local playback.

## 8. Measurement and acceptance
- [x] Collect Go CPU/allocations/locks/trace profiles and retain the rule that
      optimization requires measured evidence. Native profiling guidance provided.
- [ ] Collect a native C++ OS profile under a representative streaming workload
      before attributing a native bottleneck or changing native tuning.
- [x] Reuse PGO comparison tooling; leave adoption gated on representative evidence.
- [x] Upload saturation/queue-delay diagnostics and router SQM guidance, no
      unsolicited router changes.
- [x] Integrate regressions for demand, seeks, reconnects, suppliers, multiple
      readers/preparation, resource bounds, crash/storage/network recovery.
- [x] Runnable overnight/endurance acceptance and Android/physical network
      checklist; retain exact unperformed environmental gates.
- [x] Frontend contracts/localization/build/embed, native/platform checks, final
      diff and original-specification reconciliation; runnable Windows artifact.

Deferred by the accepted proposal: engine replacement, QUIC, machine learning,
automatic swarm-profile switching, extra large media cache and unmeasured socket
tuning. Hardware-only acceptance and signing credentials are external gates.

## Evidence

See [ADAPTIVE_VERIFICATION.md](ADAPTIVE_VERIFICATION.md) for source revisions,
commands, CI results, local reports and the original-specification reconciliation.
Checked implementation items do not mark their separate physical/device gates as
passed. Final cross/native/service, both macOS targets and branch checks passed
at verification revision `0b6991f6`; the tested Windows runtime is `1e796532`.
These revisions have identical server/web source trees. Only the explicitly
unchecked physical/profile acceptance items remain; stable approval stays false.
