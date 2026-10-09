# Current project status

Updated 10 October 2026. This is the current status; dated audit sections
describe the revisions tested at those dates and do not override this table.

| Area | Current state | Remaining gate |
| --- | --- | --- |
| Public distribution | Stable **Flow 1.0**, tag `MatriX.145.Flow-v1.0.1`, source `55c16dc3`, is published as **Latest**; all eight targets, packages/provenance, live PowerShell 5.1/7 installation/update, packaged UI/HTTPS and three container architectures passed; [release verification](RELEASE_1_0_VERIFICATION.md) | Broader physical/player, public-swarm, endurance and Windows recovery validation remains disclosed field work |
| Preview 2 integration | [PR #3](https://github.com/gnbk21/TorrServer-LT-Flow/pull/3) is merged into `develop` and included in Flow 1.0. Streaming source `52d0ad20` is retained; `.7` upgrades Go to 1.26.9 and x/net to v0.60.0. The `.6` tag failed fresh vulnerability scanning and published no release; the full corrected `.7` pipeline passed | Broader device/endurance acceptance remains separate |
| Adaptive reliability and access | Qualified useful delivery/risk, shared soft RAM budget, background scheduling, draft/saved/effective settings with idle apply, recoverable settings, bounded service recovery, optional origins/rate/capabilities and source-interface policy; exact Windows adaptive and 2,000 rejection-response checks passed on `1e796532`, final native integration and actual SCM crash/update/rollback passed on `0b6991f6`, generated playback/profile checks on `a64a871b`; [guide](ADAPTIVE_RELIABILITY.md), [verification and limits](ADAPTIVE_VERIFICATION.md) | Real device/endurance/physical network checks and representative native OS profile. Source binding is not verified VPN leak protection |
| Console presentation | Included in `.7`: source `034d4bf5` adds grouped startup information, severity/color formatting and bounded status; Windows/Linux seven-case native checks and real-executable Windows ConPTY restoration passed; [verification record](CONSOLE_AUDIT.md) | None for publication; physical acceptance remains separate |
| Startup peer retention | Included in `.7`: source `db4f12dd` removes preload peer churn and preserves lazy/warm seeders across profiles; `4eed4944` also corrects translated profile help. Matched final Windows buffer readiness improved from 29.4 s to 0.22 s in Custom and 10.0 s to 0.22 s in Legacy; final native/macOS/portable gates passed; [investigation and next work](PERFORMANCE_AUDIT.md) | Representative episode comparisons remain separate |
| Retained history and startup handoff | Included in `.7`: source `6d5c9129` adds opt-in bounded history, DHT-only native persistence/recovery, translated wait stages and early explicit-preload responses. Matched Windows requests now return in 229/232 ms instead of the prior 8.2 s response grace; ownership, corruption/recovery, read-only and packaged-UI checks passed; [verification record](NEXT_IMPROVEMENTS.md) | Public-swarm/phone comparison remains separate |
| Sparse streaming | Source `3400e54a` passed native/race/fuzz/contracts, all eight targets, service rollback, 16 default and three experimental sparse cases. Repeated comparisons retain Legacy and the graded ramp by default; rate-aware timing, scarce hints and useful-peer history are opt-in. Verified preparation and native mirrors require no provider account; [guide](SPARSE_STREAMING.md), [results and individual failures](SPARSE_VERIFICATION.md) | Physical NAT/IPv6, representative public swarms, multi-hour resources and phone measurements remain separate |
| Modern UI | Final tagged source passed type checking/lint/build, frontend unit/component and native contract validation, and 38 browser scenarios; actual published Windows UI, desktop/375-pixel viewport checks and certificate controls passed | Broader real Android layout, certificate trust and player launch acceptance |
| Disk-cache path correction | Live quoted-path write failure repaired; backend/form guards reject shell quotes with seven-language help. Native/platform/browser/service/macOS CI passed. The guard is included in `.7`; the last verified development phone session used `52d0ad20` and 2048 MiB RAM cache. [Incident and limits](ADAPTIVE_VERIFICATION.md#live-disk-cache-incident-and-path-validation-8-october-2026) | Original disk incident is fixed; broader physical/endurance acceptance remains separate |
| Portable maintenance | Native authenticated backup/restore/support, active-playback rejection and startup doctor passed | Hardware acceptance remains separate |
| Windows updater | Native failure/rollback regressions passed; final packaged scripts passed live GitHub download and required provenance under PowerShell 5.1 and 7 | None for managed distribution; physical acceptance remains separate |
| Restricted Windows service | Native SCM install/restart/stop/uninstall, API readback of bounded recovery actions, actual crash restart and authenticated update/configuration rollback passed in final disposable CI | Physical boot/sleep/network acceptance remains separate |
| Cache refetch | Native complete/partial ownership checks and hole-safe completion passed the final tagged Linux scenarios and extended Windows/Linux profiling stress | Representative-media/endurance acceptance remains separate |
| Swarm profiles | Legacy, Conservative, Balanced, Aggressive streaming, bounded Custom and experimental opt-in Adaptive Streaming; backend/native settings, UI, export and seven-language explanations; Legacy remains the default | No comparative speed ranking established |
| LT maintenance | Included in `.7`: reviewed LT 1.1.10 certificate, poster, Linux CA, AAC and macOS installer fixes adapted while retaining newer Flow toolchain/native pins; bounded engine reconnect retries; final native/platform/race/GST/service gates and five actual Windows HTTPS cases passed on `7c46fd4d`; [verification](UPSTREAM_STREAMING_VERIFICATION.md) | Physical macOS service update and GStreamer browser acceptance remain separate |
| High bitrate streaming | Bounded asynchronous storage I/O, request-age/frontier diagnostics, cumulative delivery-deficit reserves, three disabled-by-default experiments and optional memory-aware Flow Player. Tagged `.7` integration and 72/72 trials with 120-second requested workloads passed; final Windows recovery/LAN and earlier 16 matched Windows trials passed. Results are mixed; [guide](HIGH_BITRATE_GUIDE.md), [implementation/evidence](HIGH_BITRATE_IMPLEMENTATION.md) | No promotion over Legacy. Public swarms, phone decoding, physical LAN and multi-hour resource acceptance remain separate |
| Resource/performance evidence | Generated-media harness, resource runner, library optimization and six interleaved PGO comparison runs passed; PGO remains off | Multi-hour representative-media runs and real-media evidence before PGO adoption |
| Stable Flow / Modern Web 1.0 | [PR #4](https://github.com/gnbk21/TorrServer-LT-Flow/pull/4) and [PR #1](https://github.com/gnbk21/TorrServer-LT-Flow/pull/1) are merged; `MatriX.145.Flow-v1.0.1` is public and Latest. Owner-approved publication policy and exact-source evidence are recorded in [release verification](RELEASE_1_0_VERIFICATION.md) | Broader phone/public-swarm, endurance and physical Windows checks remain unfinished; they were not marked passed |
| LT 1.2.1 / 1.2.2 integration | Safe engine publication, consumption-aware transport buffer, HTTPS/certificate/media addressing and GST cue/warmup/cancellation adaptations passed final native/race/security, actual GST media and published HTTPS checks. Flow retains newer dependencies and Legacy defaults. [Integration ledger](UPSTREAM_LT_INTEGRATION.md) | Physical GST/phone compatibility and representative public-swarm performance remain separate |

## Current stable release

Flow 1.0 is the first public stable release and is marked Latest. Its
[verification record](RELEASE_1_0_VERIFICATION.md) covers exact-source tagged CI,
72 controlled streaming trials, downloaded packages and provenance, live
PowerShell installation/update, actual embedded UI/HTTPS and public container
manifests. The owner approved publication with the remaining field checks
disclosed. Legacy remains the default; experimental scheduling stays off.
The stripped Windows scan retains an unused-module advisory and is not a clean
Windows symbol-level reachability result; both native Linux source scans passed.

## Previous published preview

Preview `.7` is verified and published. Its [verification record](PREVIEW_2_7_VERIFICATION.md) covers the exact tagged pipeline, 72 repeated native transport trials, downloaded Windows package, live install/update under both PowerShell versions, actual embedded UI and public container manifests. The source and binary vulnerability scans have different precision; the record retains the stripped-binary OpenPGP advisory and does not claim a clean Windows reachability scan.

## Earlier distribution history

The initial `MatriX.145.Flow-v1.0.0` tag reached the native/platform gates but
was blocked by a prerelease-only Windows update fixture. It published no
release; its identity is retained. The corrected `v1.0.1` repeated every gate
and is the first stable release. See [verification](RELEASE_1_0_VERIFICATION.md).

The initial `MatriX.145.Flow-preview.2` tag failed its updater fixture gate in
run `36762628271`; no GitHub release was published. Strict channel selection
correctly rejected that fixture's development identity. The next attempt,
`MatriX.145.Flow-v0.2.0-preview.1` in run `36766807632`, passed every native/service
gate but found a packager bug: artifact directories were counted as files.
Regression tests now distinguish directories from files and still reject real
duplicates. The `MatriX.145.Flow-v0.2.0-preview.2` release passed packaging and
provenance publication, but live installer validation found a PowerShell
JSON-array enumeration defect. Its native binaries are unaffected; its managed
scripts must be replaced. The selector and transport fixtures now preserve the
real API response shape. The `MatriX.145.Flow-v0.2.0-preview.3` publication was
cancelled during the final notice review: statically linked Windows/Android
toolchain runtimes also need their original notices. CI now collects those from
the actual compiler/NDK installation and packaging requires all three copies.
The `MatriX.145.Flow-v0.2.0-preview.4` package passed native, packaging,
provenance and live PowerShell 5.1/7 installation/update validation. The actual
browser check then found that a persisted zero cache budget prevented opening
Settings. The read validator now accepts the backend's legacy zero value;
Apply still requires a positive budget. Unit and browser regressions cover
explicit repair without autosave. The corrected
`MatriX.145.Flow-v0.2.0-preview.5` is published; earlier tags are retained for
traceability. Its actual packaged Windows UI opens General with cache zero,
Flow and Advanced maintenance controls, plus Dashboard, Torrents and Add/Search,
without browser runtime errors. This read-only smoke used fresh loopback state,
not phone emulation or API fixtures, and stopped its owned process cleanly.
The published `.5` package has 32 assets: 13 exact-identity native binaries,
eight platform archives, manifests, scripts and original dependency notices.
All 31 checksum entries passed verification after download; repeated CI
packaging produced identical checksum inventories. GitHub attestations for the
downloaded Windows executable and ZIP verified their exact source commit/tag.
Container publication now follows all native, service, macOS, packaging and
provenance gates. The earlier failed attempt's container tag is not an accepted
release. All three final container architectures passed actual runtime/version
checks before publication. Anonymous pulls verified that `preview` and the
version tag share digest
`sha256:d1490c9c148c822134aa247f6f2cec24c715faedd0a5956ac5b4e30368c1fdea`
and that every platform's source/version labels match `e0307c6d` and `.5`.
The full tagged pipeline and source branch cross/macOS pipelines passed.

Preview 2 adds bounded rolling measurements, adaptive-window hysteresis,
per-file probe result reuse, process/cache accounting, redacted support export,
portable backup/import, startup diagnostics, verified update/rollback and
restricted service operation. These features preserve the libtorrent engine,
one media cache, existing HTTP API and settings compatibility.

Optional PGO remains off in normal builds. SSE, automatic swarm-profile switching,
predictive episode downloading, provider integration and DoH are not activated
without workload/provider evidence. Service-worker installation needs a secure
deployment design. Authenticode requires a publisher signing identity. These
conditions are explicit, not silently completed requirements.

For test methods and limitations see [MEASUREMENTS.md](MEASUREMENTS.md); for
installation and recovery see [DISTRIBUTION.md](DISTRIBUTION.md). The per-feature
implementation ledger is [IMPROVEMENT_CHECKLIST.md](IMPROVEMENT_CHECKLIST.md).
