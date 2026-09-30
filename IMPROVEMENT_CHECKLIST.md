# Preview 2 implementation and verification ledger

Scope: the improvement proposal accepted by the user on 30 September 2026.
Existing Flow and modern-interface specifications remain authoritative. Preserve
the libtorrent engine, one existing media cache, HTTP API compatibility, seven
languages and recoverable legacy sources. No change to the user's active server
or state directory is part of development verification.

An item is complete only after implementation and relevant verification.

## Distribution and consistency

- [x] Preview/stable tag classification, correct Docker channels, gated release publication.
- [x] Reproducible platform packages, checksums, dependency notices, build provenance.
- [x] Flow release manifest and installer with explicit channel selection and integrity verification.
- [x] PR/branch CI coverage for develop, native gates and dependency update PRs.
- [x] Correct quoted ETags, conditional GET/HEAD and upgrade/refresh regression tests.
- [x] One current status table reconciling historical audits and remaining requirements.

## Playback and observation

- [x] Time-bounded rolling download/wait/read-stall measurements separate from totals.
- [x] Hysteresis and bounded adaptive-window changes, preserving seek responsiveness.
- [x] Bounded per-file metadata result cache with identity checks and failure retry policy.
- [x] Explicit aggregate active/warm cache allocation accounting and native/process memory observation.
- [x] Truthful delivery-position/estimated-buffer/server-stall labels and troubleshooting explanations.
- [x] Network transition, readiness, retry and recovery measurements with bounded cancellation.

## Regression and measurement

- [x] Legal generated MP4/MKV fixtures, head/tail layouts and variable bitrate.
- [x] Controlled local peer bandwidth/delay/disconnect harness and result report.
- [x] Range overlap/cancellation/seek/reconnect/warm-expiry regressions.
- [x] Repeatable long resource-cycle runner recording RSS/cache/goroutines/handles.
- [x] Actual Go DTO/frontend validator contract tests.
- [x] Fuzz targets for Range, imports, settings and URL handling.
- [x] Opt-in loopback/authenticated profiling and execution-trace collection; native profiling guidance.
- [x] Conditional PGO builds and interleaved generated-fixture evaluation with documented results.
- [ ] Representative real-media PGO evaluation before any adoption.

## Product features

- [x] Credential-redacted bounded support-report download.
- [x] Library/settings backup export, validated import preview and pre-apply backup; credentials excluded by default.
- [x] Startup diagnostics for ports/auth/listeners and optional dependencies.
- [x] Idle-only verified update, preserved state, startup health check and rollback.
- [x] Swarm profile explanations beside settings in all maintained languages.
- [x] Playback troubleshooting panel based on known observations.
- [x] Measured large-library rendering/status optimization decision and implementation if warranted.
- [x] Restricted Windows service installation, explicit paths and boot/sleep/network recovery tests or exact external gates.
- [x] Artifact attestations; optional Authenticode remains conditional on a signing identity.
- [x] Measured bounded SSE evaluation, with polling fallback and implementation if warranted.

## Boundaries inherited from the proposal

Automatic swarm-profile selection, predictive episode prefetching, alternative
source providers and DoH remain conditional future work until tests demonstrate
their benefit and provider/account requirements are selected. Service-worker
installation needs a separate secure-context deployment design. These are not
silently enabled by this milestone.

Real phone/player traces and multi-hour acceptance on the user's media remain
external validation. Windows signing requires a publisher signing identity; do
not substitute a self-signed certificate or claim signed executable delivery.

## Evidence and outstanding gates

Native Linux runs `36747207599`, `36748117047` and `36750140703` verified the
rolling/adaptive/probe/network/cache observations, actual Go API contracts,
unit/vet/race/fuzz/security gates, maintenance and generated playback paths.
Final frontend CI verifies 42 distinct unit/component/actual-contract cases and 31 browser
scenarios. Windows native console updates passed under PowerShell 5.1 and 7.
See MEASUREMENTS.md for workloads, exact numbers and limitations.

Final tag `MatriX.145.Flow-v0.2.0-preview.5`, source `e0307c6d`, passed all eight
native targets, controlled playback, native maintenance and restricted Windows
service/update/rollback gates in run `36780598722`. Repeated packaging produced
identical checksum inventories. All 32 published assets were downloaded; all
31 checksum entries, 13 exact binary identities, eight fixed-date sorted ZIPs
and original dependency/runtime notices passed verification. Downloaded Windows
EXE/ZIP attestations verified the exact tag and source. Actual packaged scripts
passed live GitHub install/update with required provenance in PowerShell 5.1
and 7, preserving authentication, listeners, configuration and private backups.
Controlled playback passed Linux `36758891545`; extended profiling passed Linux
`36758885562` (256 initial switches plus 135 cycles) and Windows `2a12912c`
(256 plus 104 cycles). The PGO driver completed three interleaved pairs, with
both binary digests recorded and all playback checks passing. Normal builds
remain explicitly off; representative real-media adoption remains conditional.
Restricted service
update/rollback passed disposable Windows run `36753656101`, including restored
configuration, private migration quarantine and preserved account/listeners.
Actual packaged Windows browser navigation also verified that legacy zero-cache
settings open for explicit repair; Flow and Advanced maintenance controls are
wired into the page. No settings were applied; no console errors were observed.
All three container architectures passed actual runtime/version checks before
publication. Anonymous registry checks verified exact platform labels and the
same final digest for version and `preview` tags. Prerelease publication does
not update `latest`; stable publication remains gated by recorded acceptance.

The resource-cycle runner is verified with short samples. Checking that runner
item does not claim multi-hour representative playback acceptance. Physical
Windows boot/sleep/network and real Android checks remain exact external gates
in RELEASE_ACCEPTANCE.json and WEB_DEVICE_CHECKLIST.md. Stable approval remains
false. Authenticode still requires the maintainer's signing identity.
