# Reliability and usability milestone

Approved 24 October 2026. Preserve the native engine, original media bytes,
current defaults, existing state location, public-cache consent and seven locales.
Deliver a reviewed candidate/PR; publication is separate. No two-hour tests.

## Required work

- [x] Versioned Flow-owned update inventory; shared-directory/reparse safety; scoped rollback.
- [x] Latest selection, downgrade protection, source/workflow-bound provenance and runtime identity.
- [x] Clock-independent freshness across dashboard, strip and diagnostics; legacy compatibility.
- [x] Incremental partial search, bounded parallel Torznab and categorized outcomes; exact-hash deduplication.
- [x] Strict compatible storage requests, configuration serialization/revisions, idle gate and transactional recovery.
- [x] Opt-in bounded anonymous last-session summaries, restart recovery and support/UI integration.
- [x] Actionable integration tests, guided troubleshooting and user-started network checks.
- [x] Management polling honors Retry-After without retrying mutations or hiding stale observations.
- [x] Bounded redacted server/web log rotation, including shared sink and native descriptor handling.
- [x] Separate supported Intel/ARM Mac runners and caches; exact-source reusable platform recovery.
- [x] Short isolated netem network regression cases; byte integrity and resource/recovery measurements.
- [x] Contract/unit/browser/native/Windows verification, security boundary reconciliation and final diff review.
- [x] Final approved-plan reconciliation, accurate evidence/limits and reviewable PR.

## Explicitly later

Automatic same-torrent subtitles, expanded episode navigation, selective storage
cleanup and deeper symbolized C++ profiling remain follow-up candidates. No new
engine, automatic profile switching, verified-subpiece mode, PGO promotion,
paid provider or protocol replacement is part of this milestone.

## Verification receipts (24 October 2026)

Implementation head: `a9e1434deb67f8dbd37b2e149bec3af44fe4b6ec`, tested with
base `7925db49`; GitHub's tested merge is `8fa82df77f6a9845b023711a04ac46a14ee0cc21`.

- [Native build and regression run](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40113146808): passed, including all ten high-bitrate comparisons and both native profiling captures.
- [Native Mac run](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40113146712): both Intel and ARM builds and native version probes passed.
- [CodeQL](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40113146676)
  and [branch policy](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40113146770): passed.

| Requirement | Implementation and observed verification |
| --- | --- |
| Owned updater state | Explicit schema-1 inventory includes configuration, ACLs, certificates, preparation metadata, DHT state and Telegram language preferences. PS 5.1/7 fixtures preserve unrelated files, reject state/backup ancestor junctions and hard links, reject tampered/reused inventories, and quarantine only recognized new state. |
| Release identity | Latest/numeric channel fixtures passed; native Windows updater scenarios passed, including startup/migration rollback, authentication, source identity and restricted service operation. Required-attestation policy also passed against the actual published 1.1.5 executable using an isolated version-only download. |
| Freshness | Server-clock anchors plus monotonic receipt age, conservative request elapsed time and legacy handling cover dashboard/strip/drawer. Clock-skew/expiry tests and final browser checks passed. |
| Search | Native bounded-worker/outcome tests and frontend incremental, cancellation, aggregate compatibility, exact v1/v2 identity and partial-result browser regressions passed. Tests and searches share the global budget; malformed/null/oversized JacRed test responses have typed outcomes. |
| Storage | Native round-trip, namespace preservation, interrupted journal and post-write fault injection passed. Actual authenticated API migration preserved watched positions in both directions and rejected stale revisions/active playback; preparation integration rejected migration while downloading. Browser regression retained the matching storage draft revision through configuration polling and a rejected save. Telegram uses the same maintenance path. |
| History | Go tests cover bounded retention, restart reading, closure/partial evidence and identifier exclusion. Actual API checks verified authentication and disabled-history consent. Populated Go summaries passed frontend contract validators; browser tests verified manual loading and incomplete labels. |
| Troubleshooting | Seven-locale coverage, actionable saved-integration outcomes, parser/preload/playback guidance, deep linking and explicit generated-byte LAN test controls passed browser checks. |
| Retry-After | Numeric/server-Date cooldown, maximum delay, no mutation replay and expiration of existing observations passed unit/browser regressions. |
| Logs | Descriptor/rotation bounds and shared-sink tests passed locally and under native Linux validation. Concurrent cleanup is serialized; native race checks passed. |
| Mac/recovery | Both native architectures built and passed version probes on their assigned host pools, using per-host/target caches. Reusable recovery policy tests reject mismatched source, missing/skipped roles and executed code failures masquerading as queue failures. |
| Network regression | All three isolated netem cases passed byte-exact impaired and recovery Ranges. Durations: jitter 1.40 s, 1% loss 1.20 s, two-second outage 5.32 s. Peak RSS 84.2 MiB, 16 threads and 18 descriptors. The runner lacks fixed-seed support; the report explicitly records `seed_supported: false`. |

## Completed checks

- Frontend: **125 passed**, plus native exported-contract validation. All **19**
  exported DTO cases validated, including populated session rows. The ordinary
  frontend run skips only this native-input test; it ran in Linux validation.
- Browser: **107/107 passed** on the final source; all seven locale/layout cases
  and the storage polling conflict regression are included.
- Production lab: **3/3 passed**, with 5,000 library entries, a 412x915 viewport,
  4x CPU slowdown and 40 ms simulated latency. Initial JavaScript was **201,296
  bytes gzip** (196.6 KiB; unchanged 200 KiB limit). LCP 1,036–1,124 ms; CLS 0;
  input-to-paint 44.6–52.8 ms. These are controlled lab measurements.
- Python build/release policy suite: **88/88 passed**. TypeScript, ESLint,
  production build, actionlint and diff whitespace checks passed. The existing
  lazy HLS chunk-size warning remains; it is excluded from initial loading.
- Linux validation passed native unit/GStreamer/race checks, vet, reachable Go
  vulnerability scans, bounded fuzz campaigns and all netem cases. Structured
  unit output contains 648 passed test/subtest events and no failed events.
- Windows checks passed both PowerShell runtimes, actual restricted service
  lifecycle/crash policy, authenticated executable/service update and injected
  startup/migration rollback. Network-guard checks used an owned fixture only.
- All eight controlled integration groups passed, including strict request
  capacity/cancellation. All ten high-bitrate comparison jobs and both native
  profiling captures passed; no profile or experiment was promoted.
- All eight platform builds passed, including base/GStreamer version probes
  on native Intel/ARM Mac hosts. Required gates were not satisfied by skipped
  optional experiment jobs.

## Reconciliation and limits

The engine, native dependency pins, streaming defaults and public-cache consent
were preserved. The owner's running Flow 1.1.5 installation and settings were
not restarted or modified. No release was published by this milestone.
The implementation was reconciled against every approved checklist item. Final
receipt edits contain documentation only; the implementation and workflow trees
are the tested source identified above. [PR #9](https://github.com/gnbk21/TorrServer-LT-Flow/pull/9)
contains the completed candidate for review.

The final review fixed the migration request counting itself as busy, routed
Telegram through the same gate, included preparation work in the busy decision,
retained matching storage draft revisions, closed concurrent log-cleanup races,
preserved all persistent root metadata and kept the original UI payload gate.

See [security boundary reconciliation](RELIABILITY_USABILITY_SECURITY_REVIEW.md).
The repository retains 30 existing open CodeQL findings (23 high, 7 critical);
the final PR analyses reported zero incremental findings across native Go/C++,
Windows guard, JavaScript/TypeScript and Actions, with no analysis errors/warnings.
No findings or protections were disabled. Physical phone/public-swarm,
boot/sleep/network recovery and multi-hour acceptance remain field checks.
Copy/truncate native logging and unseeded hosted netem limitations are disclosed
in the implementation guide. These limits do not imply new streaming defaults
or promote a profile over Legacy.
