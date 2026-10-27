# Flow 1.1.9 verification

## Accepted implementation

Source: `49dbee114e28ef6f54ce16976cf2a908bdf5afac`.

- [Full native build and runtime checks](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40467760680): all six cross targets, native simulations, Go/cgo/race/API checks, eight functional groups, Windows service/update/rollback, ten high bitrate comparison jobs, eight parallel-streaming jobs, native profiling and inventory. Every required individual job was checked for actual success.
- [Both macOS architectures](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40467760653).
- [CodeQL and native regressions](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40467760747).
- [Windows/Linux branch checks](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40467760686).

The release receipt verifier requires unchanged runtime, embedded UI inputs,
native patches, dependencies and build recipes after this accepted source.
The tag rebuilds version-stamped binaries and repeats essential native,
functional, lifecycle, inventory, packaging and provenance checks. Long
comparisons and browser evidence are reused only after source and successful
workflow receipts are verified. The tagged rebuild remains required at this
preparation commit; publication receipts are recorded after it completes.

## Targeted regressions

Local checks passed 100 portable Python tests, Go policy tests and the actual
C++ queue/recovery policy regression under MSVC. Frontend type checking and
lint passed with 140 unit/component tests. The local contract-input-dependent
test was skipped; actual native contract export is checked by CI.

The generated Windows media fixture verified required head/index data, byte-exact
head/tail Range reads and optional-reserve expiry 10.03 seconds after the normal
gate became ready. A separate missing-index case stayed blocked. Total fixture
startup was 32.50 seconds; the ten-second allowance is not a total startup bound.
The stalled partial-block fixture fetched the missing suffix from an alternate
peer before the original peer resumed, without capacity overflow or byte errors.
These private-candidate observations are generated transport tests, not public
swarm or decoder acceptance. Their runtime changes are included in the accepted
source; final release integrations repeat these regressions.

On the accepted Linux source, the startup fixture measured 10.05 seconds of
extra wait with required-index protection intact. The stalled 1 KiB suffix
arrived in 834 ms, before the original peer resumed; all 15 strict request-
capacity cases passed. These are generated local transport observations.

## Defaults and limits

Legacy remains the default; optional streaming controls remain opt-in. The
startup reserve correction applies to Adaptive Streaming. The dashboard now
separates a wait-limited reserve from a cache-limited reserve in all seven
languages. Public artifacts exclude personal settings and playback state.
The owner's active installation is not replaced by this publication.

Broader representative final-version phone/public-swarm playback, physical
Windows boot/sleep/network recovery and multi-hour resource acceptance remain
unfinished under the owner's disclosed publication policy. No additional
two-hour test was run. Existing security dispositions remain accepted risks,
not additional fixes or proof of zero vulnerabilities. Windows executables
are not Authenticode signed.

## Unpublished tagged fixture correction

The first tagged attempt, source `b0c27bf00901445a7786233fc7e7f0fef5a21390`,
[run 40472788468](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40472788468),
failed the startup fixture's coverage assertion. Preload completed normally,
but the real probe finished after almost all ordinary head data had arrived;
supply correctly remained unknown and no optional reserve was requested.
This failed attempt was not published and is not counted as passing.

The fixture now supplies an 8 MiB probe head and the required index first,
then announces the remaining ordinary 16 MiB head only after real probe success.
The paced body interval qualifies actual supply before the normal gate completes.
The reserve-extension, expiry, capacity, byte-exact Range and missing-index
assertions remain enforced. The corrected fixture passed against the first
attempt's exact Windows binary: 36.86 seconds total, 10.17 seconds extra wait,
required index held, and the owned server stopped. Runtime, UI, native patches,
dependencies and build recipes did not change. Only this specific regression
fixture was added to the existing CI-only receipt allowlist; other runtime and
build-input changes still fail closed.

Because no release or assets existed, the unpublished tag was repaired using
an exact old-tag-object lease. Its failed source and run remain in history.
The corrected source must complete every required tagged gate before publication.

## Published Latest: exact tagged receipts

Published 27 October 2026 as [Flow 1.1.9](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.145.Flow-v1.1.9), stable and Latest.
Immutable binary source: `627dd3a9c673af507f04654014c1fd5ff7e09cb7`. This is the final tagged source. The unpublished tag repair is documented above; no published tag or asset was replaced.

- [Tagged build 40475629958](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40475629958) passed all 25 actual required essential roles. The unchanged runtime/UI proof reuses the full-source comparisons listed above; skipped jobs were not counted as passed.
- [Exact-source security workflow 40475583643](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40475583643) passed. There were zero open CodeQL alerts at publication, following the existing reviewed dispositions. No additional findings were dismissed for this release.
- All 35 public assets are inventoried; 34 SHA256SUMS entries match GitHub asset digests. Four selected downloaded subjects (Windows EXE/ZIP, BUILDINFO.json and release.json) passed actual checksums. Every Windows ZIP member matched its inventory, including source/version identities and original notices. Other platform bytes were verified by tagged packaging, not all downloaded locally.
- BUILDINFO.json and the Windows ZIP passed GitHub attestation checks bound to the exact source, tag ref and release workflow. Windows EXE SHA-256: `219985ee41a087b59b2871d9e8db91ae8790f624d83a0cbc8ce34a25e122f964`. ZIP SHA-256: `eaa48871f9d36c109136fc3158c8dd503c39d4baeb1d6c0090218c6428387e8f`.
- [Container publication 40478348687](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40478348687) passed actual version/linkage checks for Linux AMD64, ARM64 and ARMv7. Anonymous registry metadata confirmed their source/version labels and identical version/latest index digest: `sha256:7cbd15292569c0c695588073e9e2e798729a9372ec790108abedb6e7669f3e74`. No fresh local container playback is claimed.
- The corrected tagged startup fixture exercised real probe/supply separation, optional reserve expiry in 10.06 seconds after required readiness, byte-exact ranges and missing-index protection.
- The active owner installation and private settings were not replaced or uploaded. Public packages contain no personal state. Defaults and field-validation limits above remain in effect; no additional two-hour test was run.
