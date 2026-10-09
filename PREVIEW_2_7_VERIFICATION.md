# Preview 2 `.7` release verification

Verified 9 October 2026. This record covers the published prerelease, rather than
only an untagged development executable.

## Identity and publication

- Version: `MatriX.145.Flow-v0.2.0-preview.7`.
- Source: `4a871114189f4b2939c50e5fd377edb0e727c803` on `develop`; [PR #3](https://github.com/gnbk21/TorrServer-LT-Flow/pull/3) is merged.
- [Tagged release pipeline](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37951384438): all required jobs passed. Optional PGO/layer-baseline evaluations are disabled; the development-only container path is skipped in favor of the gated release container job.
- [Published release](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.145.Flow-v0.2.0-preview.7) is explicitly a **prerelease**. Stable acceptance remains false.
- Go 1.26.9 and x/net v0.60.0 are present in the downloaded Windows executable's build metadata. Required transitive module updates are recorded in `server/go.mod` and `go.sum`.
- The preceding `.6` tag failed the fresh reachable-vulnerability scan and published no release. It remains retained as failure history. The security gate was preserved; see the [Go security release](https://go.dev/doc/devel/release) and [HTTP/2 advisory](https://pkg.go.dev/vuln/GO-2026-6617).

## Automated gates

The exact tagged revision passed web type checking, lint, component/unit and
browser checks; native patch isolation and upstream queue/deadline simulations;
Go native unit/race/vet/contracts and reachable vulnerability scanning; bounded
fuzzing; native sanitizer checks; generated-media playback, cache refetch,
storage/recovery and executable integration checks; Windows network notification
tests; and actual disposable Windows service crash/restart/update/rollback checks.

The tagged vulnerability scan reported zero reachable vulnerabilities and zero
affected imported packages. A separate module-only scan identified its one
remaining advisory as [GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932), for
unmaintained x/crypto/OpenPGP packages. Flow does not import those packages; no
fixed version exists for them. x/crypto remains required for its other packages.
This distinction is recorded rather than claiming every package in every
transitive module has no advisory.

All eight targets built: Linux amd64/arm64/armv7, Windows amd64, Android
arm64/armv7 and macOS amd64/arm64. Packaging checked **13 exact-identity binaries**
and **eight platform ZIPs**, dependency notices and checksums. Repeated packaging
from identical inputs produced identical checksum inventories. GitHub build
provenance was published before the container publication job ran.

Locally, 17 release/helper Python tests and the PowerShell channel selector
passed. Portable Go tests and module digest verification passed after the
security update. Duplicate native PR runs were cancelled; the full tagged
pipeline independently passed those gates.

## Repeated high bitrate checks

All **72/72** original-byte transport trials passed, at 90/120 Mbps, with healthy,
intermittent, mixed-peer and burst sources across RAM/disk configurations. Each
matrix case rotated Legacy, Adaptive and Adaptive with experiments three times.
Each trial requested a 120-second paced workload; preloading and blocked reads
can extend its wall time. Downloaded CI reports were reconciled against
their run counts and published Linux executable digest. Report digests below
identify the evidence inspected.

| Matrix case: Mbps / source / cache backend | Passed trials | Overall report SHA-256 |
| --- | --- | --- |
| 120-bursts-disk | 9/9 | `2ed0aba6dcbb66c864faa40c7e7689788eb6e0ab9eee93dae8428ee19f916d1b` |
| 120-healthy-disk | 9/9 | `8551f7bcc8c902004bbe168fc84e98341e34e06b84856c11647023e36ef1ab93` |
| 120-mixed-peers-disk | 9/9 | `2cb07bfe44e2ea537306d608200436fff7b1481ef0707d64436c58f76c3482ff` |
| 120-outages-ram | 9/9 | `ab97c16662faecb936f9d7a65e9e12e54d32076db42b9dbfdf9835d8197907df` |
| 90-bursts-ram | 9/9 | `4f8380f48c809da7f4533642caeec15cc14a03de572eec55543c0bad6f776576` |
| 90-healthy-ram | 9/9 | `18181857c0ab8ed5fee8199d0e4e2531527da311d7a3bb6fb96941813c56e4b6` |
| 90-mixed-peers-ram | 9/9 | `ff4df26d7db39fc123eb830c5d46d0e1d683bd807a8a6e138e6edbbdf8c4cb8e` |
| 90-outages-disk | 9/9 | `f547a03f288e5b7ecdaeebba0eec259452d2395fff4fbf160be04cca9af3c567` |

Measured performance varied by scenario and policy. These are server-read
timings, not Android stall counts or proof that Adaptive beats Legacy.
Legacy remains the default; the three new experiments
remain disabled by default. No profile promotion was made.

## Downloaded Windows package

- ZIP SHA-256: `a388afacb6522a25de86e4c03251fe51bae1301bc237bf46240243166e8ceaaf`.
- Standard executable SHA-256: `7e8a01cdaccfb7ebfd26cabe62b02fa341dc90b893622a35a22956ef8a7e5dec`.
- The downloaded Windows binary scan exited **3**, reporting only GO-2026-5932. The executable is stripped; govulncheck v1.8.0 falls back to module-level precision when it cannot extract symbols (`internal/vulncheck/binary.go`, lines 108-112). This is not a clean symbol/reachability scan. The separately resolved Windows import graph excludes OpenPGP, and the exact tagged native Linux source scan passed. A local Windows source scan could not complete without the native CGo build environment. These checks cover Go code, not the complete native dependency stack.
- Downloaded manifests matched the exact tag/source, asset sizes, public URLs and checksum entries for all eight packages and 13 binaries. The Windows ZIP, executable and metadata were locally downloaded and hashed; the ZIP and executable's GitHub attestations verified the exact source/tag.
- Actual packaged `Install-Flow.ps1` and `Update-Flow.ps1`, under **PowerShell 5.1 and 7**, downloaded the real preview assets with `-RequireAttestation`. Both installed and updated the verified development baseline `52d0ad20`, preserving the separate state path, loopback listener, authentication and private configuration. Protected recovery backups and installed executable digests passed.
- Headless Chromium opened the actual downloaded executable's embedded Dashboard, General (legacy cache size zero), Flow, Advanced, library and Search/Add pages without runtime errors or API fixtures. Legacy and disabled experiment defaults were checked. A 375-pixel viewport fitted Flow settings. Settings were not applied, and the owned server shut down cleanly. This is not physical phone acceptance.
- Tests used fresh owned state directories and loopback ports. The user's executable and playback state were not replaced.

## Public container

Anonymous registry access passed for all three platforms: Linux amd64, arm64 and
arm/v7. Each published image's source/version labels matched this tag and commit.
The version tag and `preview` alias share index digest
`sha256:91fa7b1dd47cbd1e475a955d6d869504487b7e3ed37581fa5fc3bb64e88674b9`.
The release pipeline also ran each container architecture and verified runtime
linkage and executable version before publication. Local verification inspected
registry manifests/configuration, without downloading or running container layers.

## Remaining acceptance

This is a verified preview distribution. Stable Flow / Modern Web 1.0 still
requires the representative real-phone playback matrix, multi-hour resource
acceptance and physical Windows boot/sleep/network recovery. The earlier
stutter-free episode report and logged subsequent session are partial acceptance
for development source `52d0ad20`, not a new phone test of this tagged binary.
Executables have GitHub build provenance but remain unsigned by Authenticode.
The subsequent [LT 1.2.1 source evaluation](UPSTREAM_LT_1_2_1_EVALUATION.md)
also identified a remaining engine-pointer publication race risk during
reconnect. A targeted fix and race regression are needed before stable
acceptance. Passing the recorded tests is not a claim that no defects remain.
See [release notes](PREVIEW_2_7_NOTES.md), [status](STATUS.md) and
[installation/rollback](DISTRIBUTION.md).
