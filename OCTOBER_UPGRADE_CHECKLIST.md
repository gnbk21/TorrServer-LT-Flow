# October upgrade implementation

Authoritative backlog: [upgrade review](UPGRADE_REVIEW_2026_10.md). Keep the
existing engine/cache/API, original media bytes, opt-in experiments, public
metadata consent and owner settings. Deferred technologies remain deferred.

An item is complete only with implementation and relevant verification.

Final automated evidence: [October verification](OCTOBER_UPGRADE_VERIFICATION.md),
runtime `f780398f`, corrected harness `d482e8c0`. All non-skipped CI jobs passed.
Field acceptance and performance promotion remain separate, as detailed below.

| Item | State | Evidence / remaining work |
| --- | --- | --- |
| F1 request-generation delivery state | Implemented; native checks passed | Current generation alone ends recovery; overlap/probe/close/cancel tests, admitted observer retirement across metrics toggles |
| F2 categorized native/tracker errors and safe retained evidence | Implemented; native checks passed | Closed category/code/operation/repeats through native, history, support and UI; category/transport/family aggregation remains separate |
| F3 integrity event/delta and refetch regression | Implemented; native and exact Windows checks passed | Fixed 60-second meter, native failure timeline/history, clear/refetch tests and actual corrupt-wire-block harness; no attribution |
| F4 common-anchor buffer measurement and freshness | Implemented; native checks passed | Same-offset readable/verified prefixes and sample time; holes/verification/seek tests; stale data cannot advertise readiness |
| F5 hot policy toggles and existing-task cleanup | Implemented; native checks passed | Urgent smoothing resets and existing index readers honor disable; re-enable and restart-bound capacity tests |
| F6 supported source-container build entry point | Implemented; build/inventory gates passed | Retired obsolete builder; authoritative native recipes plus verified runtime image, inventory and original-byte gates |
| F7 current status and running build identity | Implemented; exact delivery checks passed | Runtime/UI version, channel, source and modified identity; unstamped builds say development; historical checkpoints retained |
| S1 8 MiB delayed-block reproduction and bounded recovery experiment | Comparisons complete; promotion criterion not established | Fifty five-rotation trials passed with 200 proven cold targets, exact bytes, bounded cache/resources and no reader leaks. Cold p95/p99 and blocked-time results are mixed; retain existing native retry and opt-in experiments. Earlier warm ranges are not cold-seek proof |
| S2 seek generation/cancellation/fairness | Implemented; native checks passed | Obsolete readers cannot refresh deadlines; cold alternating seeks and two real complementary TCP/µTP seeds serving complete 8 MiB pieces to separate device windows |
| S3 bounded staggered metadata fallback and service cooldown/retry | Implemented; native checks passed | One-second hedge, two requests/torrent, four global, 16 MiB body bound; supplied tier first, policy-keyed bounded cooldown, explicit rate limit, consent/close/hash/redirect/coalescing and already-ready regressions |
| Web explanation, buffer confidence, accessible live status | Implemented; local checks passed | Seven locales; concise cause/action, age/confidence, verified/readable server buffer and integrity evidence; raw details collapsed; numeric polling outside live announcements |
| Web interaction/performance and accessibility checks | Local and final CI checks passed | 115 tests passed/one skipped, type/lint/build passed; 104 browser checks including 200% text/polling. Enlarged header, poster, title and action sizing corrected. Three isolated production lab repeats and final Linux budgets passed. Not field INP or spoken screen-reader acceptance |
| Player early correlation, monotonic order, session summary | Implemented; final APK build passed | Header observed at open; v2 sequence/whole-session summary/minimum sampled buffer; bounded private manual export, v1 import retained. Patch repeat/drift tests and real APK build passed; new phone run remains field work |
| Network categorized recovery and transition checks | Implemented; native/Windows gates passed | Category/family/transport separation, existing interface generation cancellation and notification/guard regressions; no encryption/interface preference changes. Physical transitions remain unverified |
| Security branch/source reconciliation and boundary regressions | Reconciled; final scans passed | See OCTOBER_SECURITY_RECONCILIATION.md. Thirty retained findings identify older develop source. Go 1.26.10 security rebuild and reachable-vulnerability gate passed. Four final PR CodeQL analyses have zero results; retained base alerts remain open. No new dismissals |
| Representative resource/profiling/endurance verification | Automated captures passed; further two-hour tests waived | Earlier RAM/disk two-hour runs completed on the identified earlier executable. Owner requested no further two-hour tests; no exact-final endurance claim. ETW/Perfetto decode passed without full C++ attribution. Physical/eight-hour acceptance remains incomplete |
| CI artifact/cache/retention and superseded-run policy | Implemented; policy/actionlint passed | Caller-scoped web concurrency; superseded PR runs cancel by explicit policy, manual/tag/release runs retain full gates; failed artifacts kept with bounded retention |
| Technology/research decisions | Documented | LT 2.1.2/web seeds retained; V2_VERIFICATION_RESEARCH.md records proof/cache prerequisites. UPSTREAM_MATRIX147_REVIEW.md records adapted sorting/User-Agent fixes and separate subtitle integration scope |
| Final reconciliation and exact executable delivery | Completed with disclosed field limits | Backlog reread, final source/diff/evidence reconciled; exact Windows package/runtime/version/hash and corrupt-refetch smoke passed. Candidate staged separately; owner executable and settings preserved |

Implementation and acceptance are separate: current-source results are in the
[PR checks](https://github.com/gnbk21/TorrServer-LT-Flow/pull/6/checks). Skipped or
interrupted checks are not passes. Physical boot/sleep/adapter transitions,
spoken screen-reader acceptance, broader real-phone/public-swarm playback and
an uninterrupted eight-hour run are not established by controlled fixtures.
Disruptive OS recovery tests cannot run during the owner's movie under the
agreed constraint. Stock Just Player does not gain Flow Player instrumentation
from a server update. No new stable release is claimed by this implementation.
