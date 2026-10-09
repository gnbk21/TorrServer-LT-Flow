# Flow 1.0 verification

This record covers the downloaded, published stable release. The owner authorized
Latest publication after automated gates, with the field limitations below.
Those field checks remain unfinished; they have not been converted into passes.

## Identity and publication

- Version: `MatriX.145.Flow-v1.0.1`; public, **stable**, returned by GitHub's latest-release endpoint.
- Source: `55c16dc399b8f9b14bd5ae76b62450af027c6b81` on `master`; LT integration [PR #4](https://github.com/gnbk21/TorrServer-LT-Flow/pull/4) and Flow [PR #1](https://github.com/gnbk21/TorrServer-LT-Flow/pull/1) are merged.
- [Exact tagged pipeline](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37991885358): all required jobs passed. Optional PGO/layer-baseline evaluations remain off; the development container path is skipped in favor of gated release publication.
- 32 public assets: eight platform ZIPs, 13 native binaries, scripts, notices and manifests. Exact tag/source/version, asset sizes, URLs and checksum inventory were reconciled.
- Retained Go 1.26.9, libtorrent 2.1.2 and x/net v0.60.0; no upstream runtime downgrade.

The initial `MatriX.145.Flow-v1.0.0` tag did not publish: its release pipeline
[37987115381](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37987115381)
found a prerelease-only update fixture in the Windows service gate. Remaining
work was cancelled, and the tag was retained without rewriting its identity.
The fixture now derives each binary's actual stable/preview channel, retains
production identity/integrity checks, and tests channel transition and rollback.
The corrected tag repeats the full release pipeline independently.

## Automated source and platform gates

The tagged source passed web type checking/lint/build, unit/component tests and
38 browser scenarios; native units, race/vet/contracts and bounded fuzzing;
native sanitizer, queue/deadline simulations, generated-media streaming/cache
refetch/storage/recovery checks; Windows network notification tests; and actual
disposable Windows SCM crash/restart/update/configuration rollback checks.
All eight targets built: Linux amd64/arm64/armv7, Windows amd64, Android
arm64/armv7 and macOS amd64/arm64. Packaging twice from identical inputs produced
identical checksum inventories, with original dependency notices and provenance.

New regressions cover atomic engine publication, transport consumption versus
prefetch in RAM/disk, partial availability, cancellation, seek/Range/HEAD,
certificate ownership/reload/rollback, bounded listener shutdown, GST frame
arithmetic and shared/cancelled/cold probes. CI also exercised an actual generated
H.264/AAC MKV through GStreamer initialization/segment production.

Both standard and GST source vulnerability scans reported zero reachable
vulnerabilities. This covers Go source, not a complete native security audit.
Initial integration checkpoints caught GST command-fixture isolation, a Go
import alias conflict and FFI pointer provenance failures under checkptr. These
were corrected before the final gates and publication. Seventeen isolated FFI
tests also passed strict pointer checking on Windows; full native GST race and
actual-runtime checks passed in the tagged pipeline.

## Repeated high-bitrate streaming

**72/72** original-byte transport trials passed at 90/120 Mbps, across healthy,
intermittent, mixed-peer and burst sources in RAM/disk configurations. Each of
eight cases rotated Legacy, Adaptive and Adaptive with experiments three times.
Every trial requested a 120-second paced workload; preload/read waits can extend
wall time. Downloaded report digests and executable hashes were reconciled with
the published Linux binary.

| Matrix case: Mbps / source / cache backend | Passed trials | Report SHA-256 |
| --- | --- | --- |
| 120-bursts-disk | 9/9 | `1461f9a067d73375043c4b2806f8b5f676a2d77a1330cf0a7446a099c93118e6` |
| 120-healthy-disk | 9/9 | `0f8d7b22d3d76842056583bc1c5b6eab807f50ee04d2c67edb883d666407fea4` |
| 120-mixed-peers-disk | 9/9 | `ce813241efaa7bf445fcc3d0914b1d53f0ac7ade91290113e643f6b7f525e98c` |
| 120-outages-ram | 9/9 | `17c0f1b82cc47dcccc76be3e38ee4535ac0c9fe02eeabedbedbc4a45a55f8ce5` |
| 90-bursts-ram | 9/9 | `3c94139279c0bb70ca8502982185668c40d9ce3fe701c6228f244cd4fa8cea2d` |
| 90-healthy-ram | 9/9 | `c241406ef856b077c18cda00e32603dd31d310b1081d7fa141519d28b12a0f65` |
| 90-mixed-peers-ram | 9/9 | `a3ffbd5efd396100c8871e233e8c94f404468a6d56f7700d9290fcc012fca1d1` |
| 90-outages-disk | 9/9 | `7ad55f239d8cdac8100230edcfd93b6bf9ae5b23a5d86c1f8f239ee6ef3e0647` |

Performance is mixed. Server read waits are not Android stutter measurements.
Legacy remains the default and the three high-bitrate experiments remain off.
The local five-iteration 16 MiB memory/file microbenchmark reduced source reads
from 513 to 17 using approximately 1 MiB transport allocation per active reader.
It was slower in that synthetic workload, so this is call amortization evidence,
not a claim of universal CPU/throughput improvement.

## Downloaded Windows package

- ZIP SHA-256: `1a0f7f632826caa104851ddca86a487000d25cbe9df4fd884754499f3baf7166`.
- Standard executable SHA-256: `fb2c5ddb010085127a3f30058a8507309fc7160004642b494c8e7a75dc945420`.
- Downloaded ZIP and executable passed exact-source GitHub provenance verification, checksum verification, executable version and Go build metadata checks.
- Packaged install/update scripts passed real stable-channel GitHub download with `-RequireAttestation` under **PowerShell 5.1 and 7**, updating an owned baseline while preserving authentication, state paths, private configuration and protected recovery backups.
- Chromium opened the actual embedded Dashboard, General with legacy cache zero, Flow, Security certificates, Advanced maintenance, library and Search/Add pages without API fixtures or runtime errors. Legacy/disabled experiments and a 375-pixel layout were checked without applying settings; shutdown was clean.
- The actual published executable passed four owned HTTPS cases: HTTPS-only, HTTPS-only with the unused HTTP port occupied, forced HTTPS with HTTP media, and CLI certificate paths. Certificate upload, public-only download, stale revision rejection, invalid-path preservation, hot reload, failed-save identity rollback and private-key ACLs passed. This does not establish trust on physical Android clients.
- The stripped Windows binary scan exited **3**, retaining only [GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932), the unmaintained OpenPGP module advisory with no fixed version. govulncheck v1.8.0 falls back to module precision when it cannot extract symbols. The resolved Windows import graph excludes OpenPGP, and both exact tagged native Linux source variants passed. This is not a clean Windows symbol/reachability scan; local Windows native source scanning requires the CGo build environment.

Tests used fresh owned state and processes. The user's server and playback state
were not replaced. Source-branch CI uses a synthetic merge commit; its tree was
checked against the reviewed branch. Tagged assets use the exact master identity.

## Public container

Anonymous registry access verified Linux amd64, arm64 and arm/v7 manifests and
source/version labels. The release pipeline ran all three architectures and
verified executable version/runtime linkage. Local verification inspected
manifests/configuration, without running container layers. The version tag and
`latest` alias share index digest `sha256:abfeb6055e3a096808a6168207c5e44116f4b2c6152777533e420e00c1877484`.

## Unfinished field checks and decisions

Broader physical Android/player/browser coverage, representative public-swarm
comparisons, multi-hour resource measurements, and physical Windows boot/sleep/
network recovery remain unfinished. Earlier smooth episode reports relate to
source `52d0ad20`, not this tagged executable. Controlled peers and browser tests
do not establish phone decoding, Wi-Fi/ISP behavior or every swarm's performance.
The owner explicitly approved Latest publication with this disclosure; this is
the documented change to the original plans' physical acceptance gate.
Executables have GitHub provenance but are not Authenticode signed.

See [release notes](RELEASE_1_0_NOTES.md), [LT adaptations](UPSTREAM_LT_INTEGRATION.md),
[status](STATUS.md) and [installation/rollback](DISTRIBUTION.md).
