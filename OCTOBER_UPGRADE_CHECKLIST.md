# October upgrade implementation

Authoritative backlog: [upgrade review](UPGRADE_REVIEW_2026_10.md). Keep the
existing engine/cache/API, original media bytes, opt-in experiments, public
metadata consent and owner settings. Deferred technologies remain deferred.

An item is complete only with implementation and relevant verification.

| Item | State | Evidence / remaining work |
| --- | --- | --- |
| F1 request-generation delivery state | Implemented; native gate | Current generation alone ends recovery; overlap/probe/close/cancel tests, admitted observer retirement across metrics toggles |
| F2 categorized native/tracker errors and safe retained evidence | Implemented; native gate | Closed category/code/operation/repeats through native, history, support and UI; category/transport/family aggregation remains separate |
| F3 integrity event/delta and refetch regression | Implemented; native gate | Fixed 60-second meter, native failure timeline/history, clear/refetch tests and actual corrupt-wire-block harness; no attribution |
| F4 common-anchor buffer measurement and freshness | Implemented; native gate | Same-offset readable/verified prefixes and sample time; holes/verification/seek tests; stale data cannot advertise readiness |
| F5 hot policy toggles and existing-task cleanup | Implemented; native gate | Urgent smoothing resets and existing index readers honor disable; re-enable and restart-bound capacity tests |
| F6 supported source-container build entry point | Implemented; build gate | Retired obsolete builder; authoritative native recipes plus verified runtime image, inventory and original-byte gates |
| F7 current status and running build identity | Implemented; exact delivery gate | Runtime/UI version, channel, source and modified identity; unstamped builds say development; historical checkpoints retained |
| S1 8 MiB delayed-block reproduction and bounded recovery experiment | Implemented; comparison gate | Final-block delay/reordering, reqq caps and five rotated stable/candidate/experimental runs; p95/p99, duplicate/RSS and integrity evidence; no new retry policy promoted without evidence |
| S2 seek generation/cancellation/fairness | Implemented; native gate | Obsolete readers cannot refresh deadlines; cold alternating seeks and two real complementary TCP/µTP seeds serving complete 8 MiB pieces to separate device windows |
| S3 bounded staggered metadata fallback and service cooldown/retry | Implemented; native gate | One-second hedge, two requests/torrent, four global, 16 MiB body bound; supplied tier first, policy-keyed bounded cooldown, explicit rate limit, consent/close/hash/redirect/coalescing and already-ready regressions |
| Web explanation, buffer confidence, accessible live status | Implemented; local checks passed | Seven locales; concise cause/action, age/confidence, verified/readable server buffer and integrity evidence; raw details collapsed; numeric polling outside live announcements |
| Web interaction/performance and accessibility checks | Local checks passed; device checks separate | 115 tests passed/one skipped, type/lint/build passed; 97 browser checks and three production lab repeats passed. Event Timing/import change-to-paint measured; keyboard/layout checked across seven locales/six viewports. Not field INP or spoken screen-reader acceptance |
| Player early correlation, monotonic order, session summary | Implemented; APK build passed on prior identical recipe | Header observed at open; v2 sequence/whole-session summary/minimum sampled buffer; bounded private manual export, v1 import retained. Patch repeat/drift tests and real APK build; new phone run remains field work |
| Network categorized recovery and transition checks | Implemented; native/Windows gates | Category/family/transport separation, existing interface generation cancellation and notification/guard regressions; no encryption/interface preference changes |
| Security branch/source reconciliation and boundary regressions | Reconciled; current scan gate | See OCTOBER_SECURITY_RECONCILIATION.md. Thirty retained findings identify older develop source; no new dismissals or zero-alert claim |
| Representative resource/profiling/endurance verification | Automated gate; physical checks separate | Existing ETW/Perfetto/Go tools plus exact-build two-hour RAM/disk endurance mode; interrupted runs are incomplete, two hours is not eight-hour acceptance |
| CI artifact/cache/retention and superseded-run policy | Implemented; policy/actionlint passed | Caller-scoped web concurrency; superseded PR runs cancel by explicit policy, manual/tag/release runs retain full gates; failed artifacts kept with bounded retention |
| Technology/research decisions | Documented | LT 2.1.2/web seeds retained; V2_VERIFICATION_RESEARCH.md records proof/cache prerequisites. UPSTREAM_MATRIX147_REVIEW.md records adapted sorting/User-Agent fixes and separate subtitle integration scope |
| Final reconciliation and exact executable delivery | Required final gate | Re-read backlog/diff, inspect exact package source/version/hash and rollback; stage separately without replacing the owner's active movie server |

Implementation and acceptance are separate: current-source results are in the
[PR checks](https://github.com/gnbk21/TorrServer-LT-Flow/pull/6/checks). Skipped or
interrupted checks are not passes. Physical boot/sleep/adapter transitions,
spoken screen-reader acceptance, broader real-phone/public-swarm playback and
an uninterrupted eight-hour run are not established by controlled fixtures.
Disruptive OS recovery tests cannot run during the owner's movie under the
agreed constraint. Stock Just Player does not gain Flow Player instrumentation
from a server update. No new stable release is claimed by this implementation.
