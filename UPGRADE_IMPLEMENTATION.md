# Flow upgrade implementation ledger

Authority: the owner-approved Reliability, Streaming Performance and Web Interface
Upgrade plan, 10 October 2026, supplements both original engineering specifications.
The selected interface direction is a balanced dashboard. Every unchecked item
remains required; implementation, automated evidence and physical acceptance are
tracked separately. Baseline: stable 1.0.1 / source 55c16dc3; integration c3dfe708.

## Maintenance
- [ ] Close all MSX bodies, synchronize launcher configuration, bounded cancellation.
- [ ] Propagate torrent request cancellation; bounded shared fetch with last-owner cleanup.
- [ ] Separate credentials/network policy in shared fetch identity.
- [ ] Redact sensitive values in logs, errors and exports.
- [ ] Validate inherited proxy destinations and redirects; explicit intentional LAN policy.

## Web
- [ ] Consistent design tokens, dark/light/system themes, comfortable/compact preferences.
- [ ] Responsive navigation/sheets/action bars, keyboard/TV focus and reduced motion.
- [ ] Active playback focus, concise idle state and persistent observed-playback summary.
- [ ] Incident timeline, confidence/actions, stale timestamps and closed-detail cancellation.
- [ ] Card/list modes, restored filters/page/scroll, episode/raw-name parity and per-row actions.
- [ ] Search/add/preparation progress and retries; artwork independent of readiness.
- [ ] Settings search, grouping, units/effective values, changed-field/restart summaries.
- [ ] Draft/conflict protection, pairing/trust/readiness/LAN guidance and optional availability.
- [ ] Compact batched /flow/active and full-dataset paginated /flow/library projections.
- [ ] Split identity/live subscriptions; visibility polling, lazy optional chunks and SVG graphs.
- [ ] Seven-language parity and all existing API/player/HTTP fallback compatibility.

## Streaming and evidence
- [ ] Synchronized startup/discovery/storage/HTTP/seek/recovery timeline and sparse explanations.
- [ ] Optional Media3 startup/rebuffer/seek/buffer/drop telemetry; bounded redacted export/import.
- [ ] Opaque session correlation; explicit unknown player evidence and server-estimate labels.
- [ ] Native direct/64/256/1024 KiB buffer comparison and Go/ETW/Perfetto profiling.
- [ ] Interruption/VBR reserves, recovery hysteresis, fair readers and bounded urgency refinements.
- [ ] Duplicate traffic accounting, useful-slow-peer retention, bounded controller changes.
- [ ] Representative PGO evaluation; disabled unless comparative evidence supports adoption.
- [ ] One opt-in next-episode header/index warmup <=32 MiB in existing spare cache.
- [ ] Warmup yields to waits/risk/pressure; cancels on changes/disablement/shutdown.

## Reliability/security
- [ ] Slow/full disk, interrupted writes, crash, reconnect and concurrent-cache regressions.
- [ ] Eight-hour resource convergence and repeated playback-cycle evidence.
- [ ] CodeQL, release CycloneDX inventory, broader native parser/storage fuzz/sanitizers.
- [ ] Exposure summary and per-protocol interface-disconnect/DNS verification.
- [ ] Explicit reversible Windows WFP installation and scoped enforcement preserving LAN.
- [ ] Authoritative release metadata; stable/preview transitions, source/notices/provenance.

## Protocol milestone
- [ ] v2/hybrid parsing/identities/native callbacks/storage/resume/database/API validation.
- [ ] Actual SHA-256/Merkle/native async_hash2 verification; preserved v1 identifiers.
- [ ] Bounded verified identical-byte reuse, compatible mapping; private reuse disabled.

## Acceptance
- [ ] Existing web/contracts/native/race/sanitizer/package/update rollback gates.
- [ ] New maintenance ownership/cancellation/redaction/proxy/concurrency regressions.
- [ ] Five rotated matched-reserve comparisons across healthy/intermittent/mixed/burst cases.
- [ ] Startup/tail waits/seek/duplicates/CPU/RSS/integrity evidence; separate experiments.
- [ ] UI 1/200/1000/5000 entries, seven languages, five widths plus phone landscape.
- [ ] Auth/503/offline/stale/conflicts/fallback/keyboard and background polling checks.
- [ ] Documented lab bundle/LCP/interaction/CLS targets; real-device evidence separate.
- [ ] Physical Android, Windows boot/sleep/network and per-protocol enforcement acceptance.
- [ ] Final requirement reconciliation; unchanged defaults and recorded rollback paths.

No second media cache, paid source, required transcoding, automatic profile
promotion, media/API service-worker caching or removal of legacy before physical
acceptance. WFP installation changes require the owner's explicit installation
choice; building and testing the implementation uses isolated state.
