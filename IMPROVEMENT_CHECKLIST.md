# Preview 2 implementation and verification ledger

Scope: the improvement proposal accepted by the user on 30 September 2026.
Existing Flow and modern-interface specifications remain authoritative. Preserve
the libtorrent engine, one existing media cache, HTTP API compatibility, seven
languages and recoverable legacy sources. No change to the user's active server
or state directory is part of development verification.

An item is complete only after implementation and relevant verification.

## Distribution and consistency

- [ ] Preview/stable tag classification, correct Docker channels, gated release publication.
- [ ] Reproducible platform packages, checksums, dependency notices, build provenance.
- [ ] Flow release manifest and installer with explicit channel selection and integrity verification.
- [ ] PR/branch CI coverage for develop, native gates and dependency update PRs.
- [ ] Correct quoted ETags, conditional GET/HEAD and upgrade/refresh regression tests.
- [ ] One current status table reconciling historical audits and remaining requirements.

## Playback and observation

- [ ] Time-bounded rolling download/wait/read-stall measurements separate from totals.
- [ ] Hysteresis and bounded adaptive-window changes, preserving seek responsiveness.
- [ ] Bounded per-file metadata result cache with identity checks and failure retry policy.
- [ ] Explicit aggregate active/warm cache allocation accounting and native/process memory observation.
- [ ] Truthful delivery-position/estimated-buffer/server-stall labels and troubleshooting explanations.
- [ ] Network transition, readiness, retry and recovery measurements with bounded cancellation.

## Regression and measurement

- [ ] Legal generated MP4/MKV fixtures, head/tail layouts and variable bitrate.
- [ ] Controlled local peer bandwidth/delay/disconnect harness and result report.
- [ ] Range overlap/cancellation/seek/reconnect/warm-expiry regressions.
- [ ] Repeatable long resource-cycle runner recording RSS/cache/goroutines/handles.
- [ ] Actual Go DTO/frontend validator contract tests.
- [ ] Fuzz targets for Range, imports, settings and URL handling.
- [ ] Opt-in loopback/authenticated profiling and execution-trace collection; native profiling guidance.
- [ ] Representative PGO evaluation with documented results and conditional build support.

## Product features

- [ ] Credential-redacted bounded support-report download.
- [ ] Library/settings backup export, validated import preview and pre-apply backup; credentials excluded by default.
- [ ] Startup diagnostics for ports/auth/listeners and optional dependencies.
- [ ] Idle-only verified update, preserved state, startup health check and rollback.
- [ ] Swarm profile explanations beside settings in all maintained languages.
- [ ] Playback troubleshooting panel based on known observations.
- [ ] Measured large-library rendering/status optimization decision and implementation if warranted.
- [ ] Restricted Windows service installation, explicit paths and boot/sleep/network recovery tests or exact external gates.
- [ ] Artifact attestations; optional Authenticode signing when a signing identity is available.
- [ ] Measured bounded SSE evaluation, with polling fallback and implementation if warranted.

## Boundaries inherited from the proposal

Automatic swarm-profile selection, predictive episode prefetching, alternative
source providers and DoH remain conditional future work until tests demonstrate
their benefit and provider/account requirements are selected. Service-worker
installation needs a separate secure-context deployment design. These are not
silently enabled by this milestone.

Real phone/player traces and multi-hour acceptance on the user's media remain
external validation. Windows signing requires a publisher signing identity; do
not substitute a self-signed certificate or claim signed executable delivery.
