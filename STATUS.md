# Current project status

Updated 30 September 2026. This is the current status; dated audit sections
describe the revisions tested at those dates and do not override this table.

| Area | Current state | Remaining gate |
| --- | --- | --- |
| Public distribution | Preview 1 is published from exact commit `847f6b4f` | Preview 2 release packaging, download and attestation validation |
| Preview 2 integration | [Draft PR #3](https://github.com/gnbk21/TorrServer-LT-Flow/pull/3) | Final native playback stress and full CI gates |
| Modern UI | Type checking/lint/unit tests and 30 browser scenarios passed; large-library pagination, compact telemetry and all seven languages retained | Real Android layout and player launch acceptance |
| Portable maintenance | Native authenticated backup/restore/support, active-playback rejection and startup doctor passed | Hardware acceptance remains separate |
| Windows updater | Actual native install/update, corruption rejection, maintenance ownership, startup-failure rollback and protected backups passed in PowerShell 5.1 and 7 with controlled release transport | Live release transport and provenance validation |
| Restricted Windows service | Native SCM install/restart/stop/uninstall and authenticated update/configuration rollback passed in disposable CI | Physical boot/sleep/network acceptance remains separate |
| Cache refetch | Native complete/partial ownership checks and hole-safe completion passed Linux's three scenarios and extended Windows/Linux profiling stress | Final tagged build regression gate |
| Swarm profiles | Legacy, Conservative, Balanced, Aggressive streaming and bounded Custom; translated explanations | No comparative speed ranking established |
| Resource/performance evidence | Generated-media harness, resource runner, library optimization and six interleaved PGO comparison runs passed; PGO remains off | Multi-hour representative-media runs and real-media evidence before PGO adoption |
| Stable Flow / Modern Web 1.0 | Not approved; stable publication is gated by `RELEASE_ACCEPTANCE.json` | Representative phone/A-B evidence, multi-hour resources, Windows boot/sleep/network recovery |

The initial `MatriX.145.Flow-preview.2` tag failed its updater fixture gate in
run `36762628271`; no GitHub release was published. Strict channel selection
correctly rejected that fixture's development identity. The corrected attempt
uses `MatriX.145.Flow-v0.2.0-preview.1`; the failed tag is retained for traceability.
Container publication now follows all native, service, macOS, packaging and
provenance gates. The earlier failed attempt's container tag is not an accepted
release. Public package visibility remains to be checked with owner access.

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
