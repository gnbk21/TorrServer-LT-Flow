# Flow upgrade implementation ledger

Authority: the owner-approved Reliability, Streaming Performance and Web Interface
Upgrade plan, 10 October 2026, supplements both original engineering specifications.
The selected interface direction is a balanced dashboard. Every unchecked item
remains required; implementation, automated evidence and physical acceptance are
tracked separately. Baseline: stable 1.0.1 / source 55c16dc3; integration c3dfe708.

## Maintenance
- [x] Close all MSX bodies, synchronize launcher configuration, bounded cancellation.
- [x] Propagate torrent request cancellation; bounded shared fetch with last-owner cleanup.
- [x] Separate credentials/network policy in shared fetch identity.
- [x] Redact sensitive values in logs, errors and exports.
- [x] Validate inherited proxy destinations and redirects; explicit intentional LAN policy.

## Web
- [x] Consistent design tokens, dark/light/system themes, comfortable/compact preferences.
- [x] Responsive navigation/sheets/action bars, keyboard/TV focus and reduced motion.
- [x] Active playback focus, concise idle state and persistent observed-playback summary.
- [x] Incident timeline, confidence/actions, stale timestamps and closed-detail cancellation.
- [x] Card/list modes, restored filters/page/scroll, episode/raw-name parity and per-row actions.
- [x] Search/add/preparation progress and retries; artwork independent of readiness.
- [x] Settings search, common/advanced grouping, units/effective values, changed-field/restart summaries.
- [x] Draft/conflict protection, pairing/trust/readiness/LAN guidance and optional availability.
- [x] Compact batched /flow/active and full-dataset paginated /flow/library projections.
- [x] Split identity/live subscriptions; visibility polling, lazy optional chunks and SVG graphs.
- [x] Seven-language parity and all existing API/player/HTTP fallback compatibility.

## Streaming and evidence
- [x] Synchronized startup/discovery/storage/HTTP/seek/recovery timeline and sparse explanations.
- [x] Optional Media3 startup/rebuffer/seek/buffer/drop telemetry; bounded redacted export/import.
- [x] Opaque session correlation; explicit unknown player evidence and server-estimate labels.
- [x] Native direct/64/256/1024 KiB buffer comparison and Go/ETW/Perfetto capture/decoding; partial native attribution disclosed.
- [x] Interruption/VBR reserves, recovery hysteresis, fair readers and bounded urgency refinements.
- [x] Duplicate traffic accounting, useful-slow-peer retention, bounded controller changes.
- [x] Representative generated-workload PGO evaluation; retained off after mixed comparative results.
- [x] One opt-in next-episode header/index warmup <=32 MiB in existing spare cache.
- [x] Warmup yields to waits/risk/pressure; cancels on changes/disablement/shutdown.

## Reliability/security
- [x] Slow/full disk, interrupted writes, crash, reconnect and concurrent-cache regressions.
- [x] Repeated controlled playback cycles; partial endurance observations recorded.
- [ ] Eight-hour resource convergence: interrupted after 15,784 seconds / 965 cycles; not passed.
- [x] CodeQL execution/review, release CycloneDX inventory, native parser/storage fuzz/sanitizers.
- [x] Exposure summary and isolated per-protocol interface-disconnect verification.
- [ ] Shared OS DNS and physical network enforcement acceptance.
- [x] Explicit reversible Windows WFP installation and scoped enforcement preserving LAN.
- [x] Authoritative release metadata; stable/preview transitions, source/notices/provenance.

## Protocol milestone
- [x] v2/hybrid parsing/identities/native callbacks/storage/resume/database/API validation.
- [x] Actual SHA-256/Merkle/native async_hash2 verification; preserved v1 identifiers.
- [x] Bounded verified identical-byte reuse, compatible mapping; private reuse disabled.

These protocol items passed the corrected `2caccb12` Linux native unit/race
and CodeQL regression gates, plus macOS arm64 unit tests. Later forced TCP/uTP
and final-source gates are recorded in UPGRADE_VERIFICATION.md; physical
acceptance remains open.

## Acceptance
- [x] Existing web/contracts/native/race/sanitizer/inventory/update rollback gates; tagged packaging checked at publication.
- [x] New maintenance ownership/cancellation/redaction/proxy/concurrency regressions.
- [x] Five rotated matched-reserve comparisons across healthy/intermittent/mixed/burst cases.
- [x] Startup/tail waits/seek/duplicates/CPU/RSS/integrity evidence; separate experiments.
- [x] UI 1/200/1000/5000 entries, seven languages, five widths plus phone landscape.
- [x] Auth/503/offline/stale/conflicts/fallback/keyboard and background polling browser checks; physical phone acceptance separate.
- [x] Documented lab bundle/LCP/interaction/CLS targets; real-device evidence separate.
- [ ] Physical Android, Windows boot/sleep/network and per-protocol enforcement acceptance.
- [x] Final source requirement reconciliation; unchanged defaults and recorded rollback paths.

Final runtime source `0b6da77a` passed full run `38137745954`, macOS
`38137745936`, portable `38137745870` and CodeQL `38137745884`.
Its downloaded reports confirm 72 high-bitrate, 80 profile and 80 transport
integrity trials. Those checkpoint results preceded exact tagging. Final v1.1.3
tagged publication/provenance and downloaded distribution gates passed;
see RELEASE_1_1_VERIFICATION.md. The owner approved stable publication with
unfinished field/resource acceptance disclosed; unchecked field items remain
unverified and are not represented as passes.

No second media cache, paid source, required transcoding, automatic profile
promotion, media/API service-worker caching or removal of legacy before physical
acceptance. WFP installation changes require the owner's explicit installation
choice; building and testing the implementation uses isolated state.

Corrected runtime comparisons use feature head `15faff93` / PR merge
`78c437bd`, with 160/160 integrity trials. PGO uses `ad41ffae`, 40/40 trials
after four training scenarios, and remains off. Exact hashes, medians and scope
are in UPGRADE_MEASUREMENTS.md. These checkmarks record controlled evidence;
complete native attribution, resource convergence and physical acceptance
remain open. Final v1.1.3 package/source verification passed. Later build-tool
changes still require their own gates.

Profiling decoder source `0df6eacb` passed both isolated jobs in targeted run
`38125387761`, using symbol fixtures from `38087989863`. The passing reports
retain their executable hashes and source-run identity; this is not a fresh
full-platform build. No default was promoted from a profiled measurement.

The optional player's six AARs now match the original upstream media build;
17 pinned source archives and 38 original notices pass real-archive checks.
Packaging preserves them with a checksum-bound source report. Independent
native rebuild, resolved Bazel/toolchain closure and redistribution compatibility
remain separate, so the source/notices gate is not marked fully complete.
