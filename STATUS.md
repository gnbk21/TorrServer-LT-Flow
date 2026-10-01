# Current project status

Updated 1 October 2026. This is the current status; dated audit sections
describe the revisions tested at those dates and do not override this table.

| Area | Current state | Remaining gate |
| --- | --- | --- |
| Public distribution | Preview 2 tag `MatriX.145.Flow-v0.2.0-preview.5`, source `e0307c6d`, is published; all eight targets, packages/provenance, live Windows transport and three container architectures passed | None for this prerelease distribution |
| Preview 2 integration | [PR #3](https://github.com/gnbk21/TorrServer-LT-Flow/pull/3), ready for review; implementation and final tagged verification complete | Merge into `develop`; stable acceptance is separate |
| Console presentation | Development source `2d3afaf6` adds grouped startup information, severity/color formatting and bounded status; Windows/Linux seven-case native checks and Windows ConPTY restoration passed; [verification record](CONSOLE_AUDIT.md) | Integration/next distribution; published `.5` predates this feature |
| Modern UI | Type checking/lint, 42 distinct frontend cases including actual Go contracts, 31 browser scenarios and actual packaged Windows UI smoke passed | Real Android layout and player launch acceptance |
| Portable maintenance | Native authenticated backup/restore/support, active-playback rejection and startup doctor passed | Hardware acceptance remains separate |
| Windows updater | Native failure/rollback regressions passed; final packaged scripts passed live GitHub download and required provenance under PowerShell 5.1 and 7 | None for managed distribution; physical acceptance remains separate |
| Restricted Windows service | Native SCM install/restart/stop/uninstall and authenticated update/configuration rollback passed in disposable CI | Physical boot/sleep/network acceptance remains separate |
| Cache refetch | Native complete/partial ownership checks and hole-safe completion passed the final tagged Linux scenarios and extended Windows/Linux profiling stress | Representative-media/endurance acceptance remains separate |
| Swarm profiles | Legacy, Conservative, Balanced, Aggressive streaming and bounded Custom; translated explanations | No comparative speed ranking established |
| Resource/performance evidence | Generated-media harness, resource runner, library optimization and six interleaved PGO comparison runs passed; PGO remains off | Multi-hour representative-media runs and real-media evidence before PGO adoption |
| Stable Flow / Modern Web 1.0 | Not approved; stable publication is gated by `RELEASE_ACCEPTANCE.json` | Representative phone/A-B evidence, multi-hour resources, Windows boot/sleep/network recovery |

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
