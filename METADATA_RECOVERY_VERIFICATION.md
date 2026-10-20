# Automatic metadata recovery verification

## Scope

Recover the same magnet metadata automatically, using the existing saved library,
bounded supplied HTTPS sources and optional free public caches. Public lookup is
owner-approved on this PC and remains off by default in the product. No paid
provider, replacement release, full-episode preparation or encryption downgrade.

## Implementation reconciliation

| Requirement | Implementation / evidence |
| --- | --- |
| Existing library metadata first | Existing AddTorrent / LoadTorrent backfill retained; restart reuse checked below. |
| Supplied exact-source URLs | xs then as, at most two distinct HTTPS sources, persisted with spec. |
| Optional free caches | Torrage / iTorrents; complete v1 hash only; explicit host consent, excluded from portable backups. |
| Automatic bounded attempt | One timer per torrent after 15 s; up to four sources, 8 s / 16 MiB each, four shared fetches globally. |
| Correct torrent only | Parsed identity plus all known native v1/v2 identities verified. Native hybrid mismatch rejection tested. |
| Preserve native session | SetMetadata installs info dictionary on same handle; native tracker retention verified on the wire. Outer trackers/web seeds ignored. |
| Security | DNS pinning, public destinations, HTTPS certificate verification, bounded safe redirects, no Referer query leakage or environment proxy bypass. |
| Cleanup | Removal, shutdown and peer success cancel recovery; concurrent waiters cannot multiply work. |
| Normal ready path | File lists, metadata waiters, lazy piece priorities and library persistence use existing signalGotInfo path. |
| Useful evidence | Live state/source/count/code, metadata elapsed clock, privacy-safe retained numeric classifications and startup drawer. |
| Configuration / UI | Hot controls, default opt-out, seven locales, production embedded assets rebuilt. |
| Compatibility | Existing endpoints and state retained; no paid service or forced content download. |

## Failure investigation

The first native Windows cache trial failed. Separate HTTPS/parser checks showed
valid metainfo; tracing the actual request established that itorrents.org
redirected to HTTP. Automatic recovery deliberately rejects that downgrade. The
canonical HTTPS host itorrents.net served the hash-verified metainfo directly.
The endpoint was corrected and redirect/privacy regression tests added. Torrage
returned an HTML page during testing, which was correctly rejected.

## Verification results

Metadata engine source: e00fd08900f23623af3035d81fc0004e50952da8.
Final source including the UI-only follow-up: af2dd0e62dd5acce03bbaf0ea7f426cb12e86944.
Build: https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39392438268
Security: https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39392438130
macOS: https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39392438010
Branch: https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39392438018

- Local frontend: 109 passed, one existing skip; typecheck, lint and production build/notices passed.
- Local Python: 52 passed. Portable Go Flow/diagnostics passed.
- Extracted exact HTTP-policy tests: HTTPS redirects, rejected HTTP/local/credential/fragment redirects, redirect loop, stripped Referer, compressed-body bound and cancellation passed.
- Native 39be7832 units and race/vulnerability gates passed; its superseded broad build was cancelled to let the endpoint fix build.
- Windows 39be7832 supplied HTTPS recovery: legal published Slackware metadata, DHT/PEX disabled, recovered at 15.51 s, three files, zero content bytes, recorder errors/drops zero.
- English/Russian mobile-width recovery drawer: visible without preload progress, no overflow or page errors (mock API visual check).

- e00fd089 native unit/vet/race and reachable-vulnerability gates passed. This superseded broad build was cancelled for the final UI follow-up.
- e00fd089 native public fallback: recovered in 15.988 s from the second public source, HASH_VERIFIED, three files. Retained codes identified Torrage HTML as INVALID_METADATA (4) and successful recovery as HASH_VERIFIED (8).
- e00fd089 restart reuse: saved metadata became available in 31.28 ms with external fallback disabled. Both controls applied through the native API without an engine restart and persisted.
- Actual embedded UI testing caught an existing null-list round-trip defect: changing one checkbox also changed untouched TorznabUrls and TrustedProxies from null to empty lists, forcing a restart. The frontend now preserves untouched list representations. New regression coverage and a real native-API/Vite check passed, including EN/RU 375px fields with no overflow or page errors.

## Delivered executable and final checks

Installed at `Flow-Diagnostics/TorrServer-LT-windows-amd64.exe`, launched by
`Flow-Diagnostics/Start-Flow.cmd` with the existing Downloads state on port 8090.

- Version: `MatriX.145.Flow-dev-213172eff1e59f83bff031ccd7190c777a5fb913`.
- Runtime/UI source: `af2dd0e62dd5acce03bbaf0ea7f426cb12e86944`.
- CI merge revision: `213172eff1e59f83bff031ccd7190c777a5fb913`.
- Artifact ID: `12250198344`; its ZIP SHA-256 was verified against GitHub:
  `9060c5c4f6399df7a7552f7c1508a8b99fb2b49e81227109c3259c8641092d49`.
- Executable SHA-256 after installation:
  `e67e5848f2ef6580f805c3caa967f5d2528179250d54f32e2ec890bb03fdfdd6`.
- Actual final public-cache recovery: **15.972 s**, two sources, `HASH_VERIFIED`,
  three file entries and **zero torrent payload bytes**.
- Saved metainfo reused after a native process restart in **6.03 ms**, with both
  external fallback controls disabled.
- The actual embedded UI applied the public-cache checkbox on/off without an
  engine restart and preserved unrelated fields. EN/RU controls were inspected
  at 375px, with no overflow or page errors. No mock API was used in this check.
- Product default opt-out was observed for 17 seconds with DHT/PEX disabled:
  metadata stayed unavailable and no external recovery event was recorded.
- Classified retained events distinguished invalid HTML (4) from verified
  recovery (8); no source URL/hash/title fields were retained.
- Final source native unit/vet/race/reachable-vulnerability gates passed.
  Windows x64, Android ARMv7/ARM64, Linux ARMv7/ARM64 and macOS builds passed.
  Native queue/deadline simulations, web browser regressions and branch checks
  passed. The final broad Linux integration/high-bitrate/service run and final
  native CodeQL repeat were still running at this checkpoint. Native engine code
  is byte-for-byte unchanged from e00fd089, whose complete CodeQL run passed;
  final JavaScript/Actions/Windows-helper security jobs passed separately.

### Owner configuration and rollback

The owner-approved `MetadataFallback` / `PublicMetadataCache` controls were
applied through the revision-checked settings API and verified in the durable
recovery snapshot. The **2 GiB cache, Adaptive profile, ForceEncrypt and all other
existing preferences were preserved**. The server was already stopped; only our
temporary configuration process ran and was stopped afterward. The original
stable executable in Downloads was not replaced.

`Start-Flow-before-metadata.cmd` launches the retained immediately previous
startup-fix executable. Its SHA-256 is
`ba04bbc69490db5580a2c9fdf03a8fd6097bc21636e9838ee50a64d78c63b410`.
The older `Start-Flow-previous.cmd` remains available too.

### Reproduction evidence

Ignored local reports under `.tools/` contain the sampled native observations:
`metadata-delivery-af2dd0e6/report.json`, `metadata-delivery-af2dd0e6/live-ui/report.json`,
`metadata-install-af2dd0e6/report.json`, and the corresponding owned process logs.
Native source tests are `server/torr/metadata_recovery_test.go`,
`server/lt/metadata_test.go`, and `server/settings/flow_test.go`; frontend round-trip
coverage is `web/src/lib/settings.test.ts`. Public-service tests require network
access and are evidence of this run, not an availability guarantee.


## Limits

Public cache availability is independent of Flow. A matching cache entry solves
metadata discovery; streaming still needs responsive peers or explicitly configured
web seeds. Tests used a legal published Linux-distribution torrent, not the user's
private viewing history. No real-phone retry, public-swarm bitrate guarantee,
multi-hour endurance or physical sleep/boot test is claimed by these checks.
This is a development executable on the existing PR, not a new stable release.
