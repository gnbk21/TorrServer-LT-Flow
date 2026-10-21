# October upgrade implementation

Authoritative backlog: [upgrade review](UPGRADE_REVIEW_2026_10.md). Keep the
existing engine/cache/API, original media bytes, opt-in experiments, public
metadata consent and owner settings. Deferred technologies remain deferred.

An item is complete only with implementation and relevant verification.

| Item | State | Evidence / remaining work |
| --- | --- | --- |
| F1 request-generation delivery state | In progress | Overlapping seek/probe/cancel tests required |
| F2 categorized native/tracker errors and safe retained evidence | Pending | Native → history → export → UI |
| F3 integrity event/delta and refetch regression | Pending | Corrupt-block fixture; no attribution without evidence |
| F4 common-anchor buffer measurement and freshness | Pending | Holes, seeks, multiple readers and unknown rate |
| F5 hot policy toggles and existing-task cleanup | Pending | Keep native changes restart-bound |
| F6 supported source-container build entry point | Pending | Match authoritative native recipe |
| F7 current status and running build identity | Pending | Stable/development/source; preserve historical checkpoints |
| S1 8 MiB delayed-block reproduction and bounded recovery experiment | Pending | Repeated baseline/candidate, integrity, duplicate/RSS bounds |
| S2 seek generation/cancellation/fairness | Pending | Cold alternating seeks, probes, concurrent readers, limited queues |
| S3 bounded staggered metadata fallback and service cooldown/retry | Pending | Consent, network policy, cancellation, misses vs failures |
| Web explanation, buffer confidence, accessible live status | Pending | Seven locales, narrow layout, stale/unknown states |
| Web interaction/performance and accessibility checks | Pending | Diagnostics/settings/import/filter Event Timing and long tasks |
| Player early correlation, monotonic order, session summary | Pending | Optional player, bounded private export, no unsolicited transfer |
| Network categorized recovery and transition checks | Pending | Transport/family separation; preserve interface/encryption |
| Security branch/source reconciliation and boundary regressions | Pending | Preserve approved dispositions and exact evidence |
| Representative resource/profiling/endurance verification | Pending | Interrupted/fixture runs are not physical or phone acceptance |
| CI artifact/cache/retention and superseded-run policy | Pending | Preserve full release gates and failed evidence |
| Technology/research decisions | Pending | Retain LT 2.1.2/web seeds; document v2 verified-subpiece feasibility |
| Final reconciliation and exact executable delivery | Pending | Re-read backlog, diff, build identity and rollback |
