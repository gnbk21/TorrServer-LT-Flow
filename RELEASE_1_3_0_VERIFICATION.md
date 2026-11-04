# Flow 1.3.0 publication verification

Published 4 November 2026 as [TorrServer-Flow 1.3.0](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.148.Flow-v1.3.0), public, stable and **Latest**.
Immutable binary source: `dd7640ad6b59adcc4bfafd811bfd63e9f75dfd06`.
Release ID: `433672332`; publication time: `2026-11-04T08:44:43Z`.

## Source, security and distribution

- [PR #14](https://github.com/gnbk21/TorrServer-LT-Flow/pull/14) is merged. All 58 required preflight jobs passed at `a70d15f0`, including eight platform builds, native/security/browser checks and the corrected comparison. Subsequent application, native and web source remained unchanged; release metadata and the browser CI host/cache policy changed.
- The initial unpublished tagged attempt (`3eb1a844`, run `41791260374`) stopped at WebKit launch after Ubuntu's runner migration restored a browser cache requiring ICU 74 onto an ICU 78 host. All 33 WebKit cases failed before opening the application; 69 Chromium/Firefox cases passed. Those failures are retained, not counted as acceptance. The shared browser job now uses Ubuntu 24.04 and a distribution-specific cache key. All three browsers remain mandatory; 18 CI-policy tests pass. The unpublished tag was corrected using its exact old-object lease, with no release/assets to replace. Published 1.2.0 was untouched.
- [Final tagged release](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/41792926890) passed all 25 named required roles, ten high-bitrate matrix jobs, eight parallel-streaming jobs and two native captures. Additional shared-web jobs also passed. All eight platform builds, native runtime checks, race/fuzz checks, Windows service/guard checks, reproducible packaging and CycloneDX validation completed.
- [Exact-source CodeQL](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/41792850772) passed all five jobs; the subsequent master-branch query returned no open findings. This is scoped scan evidence, not a vulnerability-free claim or dismissal of new alerts. Existing documented administrator capabilities/risks remain unchanged.
- All **35 asset files were downloaded** and their SHA-256 values checked against GitHub metadata. All 34 payload digests match SHA256SUMS and the verified hosted-provenance statement. BUILDINFO and the Windows ZIP were independently verified using GitHub CLI with the exact source/tag, official release workflow and rejection of self-hosted runners.
- All eight ZIPs' members were checked for hashes, duplicate/unsafe names and symlinks. All **13 Go binaries** have the exact version/source, clean VCS state and expected GOOS/GOARCH (including ARMv7). Windows standard/GStreamer version probes pass. Source-matched distribution helpers, original licenses/notices, SBOM and the Windows guard are included.

| Windows artifact | SHA-256 |
|---|---|
| ZIP | `cfb45ed29afa535d59cb6f996c2a015f787502be34bed3b2e84448a3b9adc264` |
| Standard executable | `e471626701e5362c72c0e37a4321d75cfdc3e072b1de6f95f9c3f5d0d1e94b6f` |

## Downloaded-package integration

The real downloaded Windows package was extracted into a separate test installation with disposable loopback state. Its 27 byte-checked media ranges cover the existing five head/tail MP4/MKV and variable-rate fixtures, including seeks, cancellation and overlapping readers. The combined check passes:

- Metadata-only coverage/media/subtitle listing, without native activation or extended torrent lifetime.
- Original/escaped subtitle bytes, selected token authority, old video-only scope, malformed/oversized handling and private versioned failures.
- Successful video delivery in Recent, exclusion of probes/auxiliary work, position provenance and process-monotonic monitoring across torrent removal.
- Real desktop/mobile embedded UI, fresh metadata controls, Recent enabled by default and Prometheus disabled by default. Settled screenshots were inspected; first-paint screenshots remain separately retained. Browser tests cover all seven locales, palettes/modes, text scales, reduced motion, forced colors, focus and navigation. The initial JavaScript budget remains 200 KiB gzip.

The owner's installation/settings were not changed or uploaded. The owner declined further private-backup investigation. The existing synthetic rollback fixture proves 1.2.0 can read legacy settings/42.5-second timecodes; it is not a backup of the owner's state. The installed updater's backup mechanism remains required for an actual installation update.

## Measurements and retained costs

[Exact tagged-binary comparison](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/41795245168) used immutable published 1.2.0, identical per-trial settings and five rotated pairs for healthy, outage, mixed-peer and burst scenarios in Legacy/Adaptive: **80 trials, 50.4 billion verified video bytes, 320 proven cold reads, zero native failed bytes and no new timeouts**. The actual tagged Linux binary was compared because its native build host migrated to Ubuntu 26. All eight complete reports and hashes are retained.

Four threshold flags remain reviewed/disclosed: ordinary-seek p95 in mixed-peer/Adaptive and outage/Legacy, plus median per-trial cold-seek p95 increases of **448.46 ms** (712.25 → 1,160.72 ms) and **962.58 ms** (584.54 → 1,547.11 ms), respectively. Ordinary seeks mix different cached/nonresident observations; those states remain in the reports. The outage pooled cold-read median was 322.14 versus 298.27 ms, while its slower per-trial tails triggered the threshold. These are different summaries, not a fabricated global percentile. Forced-outage delivery deficits occurred in both versions and are not decoded-player measurements.

This final cohort had no RSS threshold flag. Earlier cohorts' different latency flags and approximately **152 MiB / 16.6%** extra median peak RSS remain recorded in the [ledger](docs/development/FLOW_13.md); the favorable final RSS result does not erase them. The precise causes are not established or called fixed. No universal streaming-speed/memory improvement, added peer supply or experiment promotion is claimed.

All **122 tagged high-bitrate trials** passed; the two five-repeat straggler matrices had no additional threshold flags. The separate combined observer check passed all eight trials. Earlier failed cold-admission evidence is retained; the corrected harness requires four proven cold reads within bounded distinct-target admission and never retries byte errors/timeouts. No default two-hour test was added.

## Container and field boundaries

[Stable container publication](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/41801952139) passed real runtime-linkage/version probes for Linux AMD64, ARM64 and ARMv7 before pushing. Anonymous registry verification confirms all three image configurations carry the exact source/version and that the version tag and `latest` resolve to the same index: `sha256:895398eae9d56347f43426d65c9bf4adfbd1c7d6f12702c4ec8937b521dc63a3`. No new listener or personal settings are included in these images.

Physical Android/Lampa/Just Player launch, selected-subtitle discovery, saved-position handoff, pause/resume, seeks and Wi-Fi reconnection remain unverified for 1.3. So do manual screen-reader smoke checks, representative public swarms, physical Windows recovery and extended resource acceptance. Windows binaries are not Authenticode signed. These limits are disclosed under the owner's publication authorization.

Legacy, experimental scheduling, metadata-cache consent, authentication/trusted origins and TrackTimecode retain their values. Direct playback and original media bytes remain the defaults; subtitles/conversion require explicit selection and Prometheus remains off. [Release notes](RELEASE_1_3_0_NOTES.md), [API/settings documentation](FLOW.md#flow-13-additions).
