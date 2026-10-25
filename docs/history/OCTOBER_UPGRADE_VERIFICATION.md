# October upgrade verification

Publication follow-up: PR #6 and CI repair PR #7 are merged into stable
[Flow 1.1.4](RELEASE_1_1_4_VERIFICATION.md). Older development identities and
pending-publication statements below are historical checkpoints. A passing
security scan does not mean retained reviewed findings have disappeared.

22 October 2026. This records the implementation of the complete
[October backlog](UPGRADE_REVIEW_2026_10.md), with a
[requirement checklist](OCTOBER_UPGRADE_CHECKLIST.md). Development work remains in
[PR #6](https://github.com/gnbk21/TorrServer-LT-Flow/pull/6); the published stable
release remains `MatriX.145.Flow-v1.1.3`. This work does not publish a release.

## Source and completed gates

Production implementation: `f780398f7d9f607f4024504d79492c336080f7d0`.
Corrected test harness: `d482e8c06d0e7cce7575f5300f30cfc70290153b`.
Between these commits, only Python test helpers and the Markdown checklist
changed; server, frontend, native patches and build recipes are identical.
Subsequent documentation changes do not change those tested components.

All non-skipped jobs completed successfully for `d482e8c0`:

| Gate | Completed run |
| --- | --- |
| Native, cross-platform, web, controlled streaming, ten high-bitrate jobs, Windows service/update/rollback, profiling and container inventory | [39761266311](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39761266311) |
| Native Go/C++, Windows WFP, JavaScript/TypeScript and Actions CodeQL | [39761266130](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39761266130) |
| Portable, inventory and Windows guard branch checks | [39761266642](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39761266642) |
| Both macOS targets | [39761266143](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39761266143) |
| Optional Flow Player APK | [39761265933](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39761265933) |

The earlier complete runtime build
[39750108840](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39750108840)
also passed. Its downloaded Windows executable was used for the exact-binary
local checks below. The optional branch-specific profile/transport sweeps were
skipped by their existing conditions; they are not passes. No PGO or new profile
default was promoted.

## Implemented behavior and verification

| Requirement | Implementation and evidence |
| --- | --- |
| F1 delivery state | Only the current real playback request can end seek recovery. Old responses, probes and unobserved requests cannot retire another reader or clear a newer seek. Overlap, cancellation, first-byte, metrics-toggle and last-reader tests passed. These are server delivery states. |
| F2 categorized failures | Closed native category, numeric code, operation and repeat count survive Go, history, support export and UI. DNS/TLS/HTTP interpretations respect categories; transport/address family remain separate. Old history and bounded redaction regressions passed. |
| F3 integrity evidence | Bounded recent hash-failure events and cumulative/recent counters distinguish corruption from redundant traffic. Actual corrupt-wire-block checks rejected bad data, recorded the failure, refetched and delivered original bytes. Delayed alerts do not invalidate the replacement a second time. |
| F4 buffer measurement | Readable and verified contiguous prefixes share the same anchor and sample time. Seek, stale sample, multiple-reader, cache-hole and unknown-bitrate regressions passed. Server cache is labelled separately from phone buffer. |
| F5 hot policy changes | Urgent-horizon changes reset smoothing; existing container-index readers/workers honor disable. Safe flags apply without engine restart; native capacity/transport changes still require restart. Toggle and cleanup tests passed. |
| F6 build consistency | Obsolete standalone builder is explicitly retired. Supported native recipes, runtime image, actual dependency inventory and original-byte delivery gates passed. LT remains pinned to 2.1.2. |
| F7 build identity | Runtime/API/UI distinguish stable and development builds and report source/modified status. Actual executable identity, generated embedded UI, archive digest and binary hash were verified. Dated historical failures remain recorded. |
| S1 stragglers | 8 MiB final-block delay/reordering, strict peer queue limits and five rotated baseline/candidate/experiment comparisons passed. Corrected cold-seek proof and mixed performance results are detailed below. No additional retry/picker patch is promoted without a consistent benefit. |
| S2 cancellation/fairness | Obsolete readers cannot renew deadlines. Strict-capacity, rapid cancellation/late completion, alternating cold targets and complementary TCP-only/µTP-only seeds serving complete pieces to separate device windows passed. |
| S3 metadata recovery | Supplied sources retain preference; a slow authorized source permits a one-second staggered second request. Limits are two requests per torrent, four globally and 16 MiB per body. Policy-keyed coalescing, bounded service cooldowns, explicit rate-limited retry, consent revocation, removal, redirects, all-known-hash validation and peer/local metadata winning passed. Missing metadata is distinct from service outage. Already-ready metadata triggers no unnecessary external fetch. |
| Web explanation and access | Seven locales show concise cause/action, evidence age/confidence and verified/readable server buffer; raw details are collapsed. Live announcements follow state changes. Visited settings panels retain drafts; header/cards/actions/dialogs work with enlarged text. 115 unit/component tests passed, one existing skip; type/lint/build and 104 browser checks passed. |
| Web responsiveness | Shared/visibility-aware polling is retained. Three isolated production-lab repeats met unchanged budgets: initial gzip JS 202,986 bytes, LCP 1048/1084/1152 ms, CLS 0.004754, input-to-paint 51.2/61.4/75.2 ms, import-to-paint 103.3/106.0/103.8 ms. Maximum Event Timing was 160/160/136 ms. Final Linux production-budget gates also passed after the dialog correction. This is not field INP. |
| Optional player | Response correlation is captured when the data source opens. Version 2 reports add monotonic order and whole-session statistics with bounded private manual export; version 1 import remains supported. Patch/drift tests and the real APK build passed. Stock Just Player is unchanged by the server update. |
| Network/mirrors | Categorized transport/family evidence, interface cancellation, Windows WFP and selected-policy regressions passed. Bounded mirror error records discard raw URLs and distinguish a last observed failure from current availability. Owner encryption/interface choices are preserved. |
| Engineering | Go 1.26.10 security rebuild, resource/profile tools, bounded artifact retention and explicit superseded-PR cancellation policy passed their gates. Manual/tag/release gates remain intact. Research/deferred technologies remain as specified. |

Controlled native coverage includes three playback cases with 26 original-byte
Range checks each, 15 strict request-capacity cases, six cancellation/late-completion
trials, 16 sparse cases plus three rate-aware cases, five web-seed cases, startup
ordering/progress/missing-index checks, maintenance/preparation/recovery,
peer-hint reuse, four DHT-state restoration cases and five certificate cases.
The final matrix also passed **122 trials**: 72 existing high-bitrate regression
trials plus 50 straggler trials. These are generated-media fixtures.

## Corrected cold-seek comparisons

The first matrix passed byte checks but its generic seek offsets were mostly
cached. It is retained as regression evidence, not cold-seek proof. The corrected
harness requires a zero-resident cache snapshot and a positive increase in wire
requests for the selected piece. Unknown geometry, unavailable counters or cached
targets fail the proof. Final duplicate counts and resource peaks include the
cold phase. All 59 Python fixture tests passed.

Final CI passed **50/50 straggler trials and 200/200 proven cold targets**, with
original-byte hashes, no remaining readers and cache usage within the configured
budget plus the explicit two-piece allowance. Five policies each ran five
rotations, with four proven cold seeks per run, at 90 Mb/s/RAM and 120 Mb/s/disk.
Both used 8 MiB pieces and a 512 MiB rolling cache. The stable Linux baseline
SHA-256 is `c2150033a8c101ef35bf4035a427733812baed6bc3a6c4015b5c15723dc368d4`;
candidate SHA-256 is
`3033519e6fd8e4e7409af24d8d5b5f728420418d62646f85d70315fe04b8c74e`.

| Policy | 90 Mb/s RAM cold median / reported p95–p99 | 120 Mb/s disk cold median / reported p95–p99 |
| --- | ---: | ---: |
| Stable Legacy | 221.1 / 3332.1 ms | 46.9 / 717.9 ms |
| Stable Adaptive | 224.9 / 1388.2 ms | 70.4 / 427.3 ms |
| Candidate Legacy | 174.1 / 1135.7 ms | 75.4 / 1116.2 ms |
| Candidate Adaptive | 259.7 / 3334.3 ms | 44.7 / 1119.3 ms |
| Candidate Adaptive with existing opt-in experiments | 110.0 / 294.2 ms | 147.1 / 334.6 ms |

Each cell summarizes 20 samples. The harness uses sorted index
`min(n-1, int(n*q))`; at this small sample count its reported p95 and p99 both
equal the maximum. This is not a population percentile estimate. Targets are
chosen from each run's actual cache, so offsets can differ across policies.
These are conditional cold-miss recovery comparisons, not identical phone seek
traces or a public-swarm speed ranking.

Median steady-delivery wait above 250 ms was zero for every RAM policy. Disk
medians were 754.8/482.0/312.5/0/271.3 ms respectively. Across these trials the
largest redundant-byte total was 3.05 MiB; observed peak RSS was at most
807.7 MiB for RAM and 88.4 MiB for disk. These observed bounds include sampled
cold-phase resources and do not constitute a general process-memory guarantee.
The experimental policy combines existing capacity-aware, urgent-horizon and
container-burst flags; it does not isolate the effect of each flag.

**Decision:** the complete improvement criterion (consistently lower cold tail
and blocked time without a scarce/healthy regression) is not established.
Keep Legacy as default and experiments opt-in. Existing native retry behavior
recovers these cases; the mixed results do not justify a new piece picker or
unconditional timeout/retry changes.

## Exact staged Windows candidate

A separate local `Flow-Upgrade-Candidate` folder contains the standard executable,
guarded launcher, build identity, license and native/web notices. Packaging did
not modify or restart the existing installation or owner settings.

- Runtime source branch: `f780398f7d9f607f4024504d79492c336080f7d0`.
- Reviewed merge/runtime revision: `5148a9fdf89aa87615ea9a0b8d5f5c03516ebeac`.
- Runtime version: `MatriX.145.Flow-dev-5148a9fdf89aa87615ea9a0b8d5f5c03516ebeac`.
- Development channel, `modified: true` because CI regenerated embedded assets;
  actual Go build metadata is Go 1.26.10, Windows amd64.
- Successful build `39750108840`, Windows artifact `12414040644`.
- Downloaded ZIP SHA-256, verified against GitHub's artifact digest:
  `deaace6c806c3042cc724cbbc7d3d5b81f610fec39ff0dab63c572e6f789129e`.
- Standard executable: 58,695,680 bytes, SHA-256
  `ae2d11b77db8d18bc0ecf902871c41ca6aeca7d87ce84a6846827cab44c319b1`.

This exact executable started in 515.6 ms, served the embedded HTML/574,445-byte
main script with a noncached index, and returned consistent runtime/echo identity.
It rejected a corrupt wire block, recorded one integrity event, refetched and
delivered the exact original 8 MiB SHA-256
`7d212b9c884f5c77896de960ae17cc341cda43b14d6a971f34ca29ebd4badf7f`.
Four proven cold targets also passed in separate Windows Adaptive/RAM and
Legacy/disk smokes; these single runs do not establish a speed ranking.
The follow-up changes only test helpers/docs, so the staged executable contains
the same production implementation verified by the corrected final matrix.
Digest verification is not a release-attestation claim.

Close the existing Flow server after finishing playback before launching this
candidate. Its launcher reuses existing state and refuses an occupied port.
Rollback is to close the candidate and launch the preserved previous folder.
No candidate has been automatically installed into the running movie server.

## Resource evidence and explicit test limits

The owner requested **no further two-hour tests** on 22 October. None were
started after that instruction. An earlier pair had already completed:
[39686916896](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39686916896),
source `febdb8d6`, executable SHA-256
`0b046867e4527f821ad27ecf8e2f936c351974796e80167971d7d69949b33744`.

| Backend | Duration / byte-checked cycles | Early → late median RSS | Handles | Goroutines |
| --- | --- | --- | --- | --- |
| RAM | 7201.16 s / 440 | 99,909,632 → 82,051,072 bytes | 462 → 366 | 18 → 18 |
| Disk | 7203.45 s / 474 | 39,284,736 → 38,354,944 bytes | 524 → 432 | 18 → 18 |

Both released all readers, cache reservations and protected bytes and met the
declared resource-convergence allowances. Subsequent production changes are the
Go security toolchain and web assets; these results nevertheless identify an
earlier executable, not exact-final endurance acceptance. Additional two-hour
testing is waived by the owner. The earlier interrupted eight-hour test remains
incomplete.

Linux Perfetto validation passed with 847 owned-process samples/stacks; Windows
ETW decoded 179,311 events with zero loss in the retained runtime-source capture.
Both explicitly report `native_attribution_known: false`: they do not establish
C++ function/allocation attribution. Final follow-up profiling jobs passed too.
PGO remains off. Actual disposable Windows SCM/WFP, authenticated update,
integrity rejection, failed-start/configuration rollback and state-preservation
gates passed. Physical boot, sleep, adapter/VPN transitions, spoken screen-reader
acceptance and new real-phone/public-swarm playback remain unverified. They
cannot be inferred from fixtures or performed disruptively during the owner's
movie. No universal performance or ISP diagnosis is claimed.

## Security and retained failures

- A reachable-vulnerability gate correctly blocked Go 1.26.9 with
  `GO-2026-6809` through certificate verification. Go 1.26.10 is now required;
  fresh native reachable-vulnerability gates passed. The finding was neither
  dismissed nor bypassed. See the
  [official Go release history](https://go.dev/doc/devel/release#go1.26.0).
- An earlier DHT rebind fixture reserved UDP without excluding a TCP collision.
  It now reserves a matching TCP/UDP pair and retains bounded failure evidence.
  Final repeats passed. This fixes ambiguity in the fixture; the exact historical
  failure cause is unproven, and no production DHT fix is claimed.
- Earlier Linux enlarged-dialog overflow failures were fixed with a shrinkable,
  wrapping title container; final browser repeats passed. Two locally contended
  Event Timing failures at 384/280 ms remain retained; isolated repeats used the
  same budgets and passed. The failed attempts are not relabelled as passes.
- Four completed CodeQL analyses on final PR merge
  `1b8e6f6f995a84d4b06a94b9f98dc36dd9b9fa00` report zero results and no execution
  errors. The repository still has 30 open retained alerts (seven critical,
  23 high), whose latest instances identify `develop` source `26f22bb3`.
  No new alert dismissals were made. See
  [source/boundary reconciliation](../security/OCTOBER_SECURITY_RECONCILIATION.md).

## MatriX.147 and reconciliation

The [upstream release](https://github.com/YouROK/TorrServer/releases/tag/MatriX.147)
and complete diff were reviewed. Natural episode ordering and Torznab
product/version User-Agent fixes were adapted with tests; relevant URL
normalization and playlist file-ID sharing already exist. Automatic torrent
sidecar subtitle conversion was reviewed as a separate feature requiring engine,
authorization, cancellation and player adaptation. It is not implemented or
claimed as part of this backlog. That release contains no libtorrent/high-bitrate
picker upgrade. See the [complete review](UPSTREAM_MATRIX147_REVIEW.md).

The backlog was reread and reconciled after implementation. Implemented behavior
and automated gates are complete for the identified source. The remaining field
checks, unestablished performance-promotion criterion and owner-waived additional
endurance are stated above. V2 per-block verification remains
[research](V2_VERIFICATION_RESEARCH.md); SSE/PWA, QUIC/WebRTC rewrites, ML,
paid providers and a second media cache remain deferred as specified.
