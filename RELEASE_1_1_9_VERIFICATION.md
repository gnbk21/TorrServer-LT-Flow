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
