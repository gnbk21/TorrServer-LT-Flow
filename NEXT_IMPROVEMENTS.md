# Startup and retained diagnostics implementation checklist

Scope: the first four next-work items in `PERFORMANCE_AUDIT.md`. Preserve the
native engine, single cache, HTTP correctness, bounded probes, existing settings
migration and independent playback windows.

- [x] Opt-in structured history: bounded queue, nonblocking producers, fixed
  rotation/disk limits, private allowlisted fields, drop/error counts, bounded
  shutdown and retention across restarts; configuration and documentation.
- [x] DHT-only native session serialization: bounded validation, corrupt-file
  fallback, atomic private writes, periodic and graceful-stop saves, respect
  disabled DHT/read-only settings; round-trip and recovery verification.
- [x] Startup explanation: metadata/discovery/first useful data/head-index/
  prebuffer/probe-grace/ready/outcome timings; translated UI; unknown timings
  distinct from measured zero; no fixed player request sequence.
- [x] Early explicit preload response: release after readiness, preserve a
  bounded background warm handoff, coalesce concurrent same-file preloads,
  supersede other files safely, cancel/join on close, preserve reader priorities
  and reservation ownership; response timing and lifecycle regressions.
- [x] Native builds, meaningful unit/race/privacy/rotation/corruption tests,
  frontend checks, controlled startup and playback regressions; review diff and
  reconcile original specifications; provide a verified Windows executable.

Real phone request/first-frame comparison, public-swarm cold/warm discovery,
multi-hour playback and physical network/boot tests remain acceptance evidence,
not conclusions from deterministic local tests.

## Implementation decisions

- History defaults off. Its 256-event queue never waits for disk; four exact
  rotating file names retain at most 4 MiB. A strict event/stage allowlist
  excludes media identity, URLs, credentials and client addresses. Partial final
  records are repaired on restart. Shutdown waits at most one second for the
  writer; a still-running writer prevents a second writer opening the same file.
- DHT persistence defaults on only when Flow/DHT permit it. Native session
  parameters serialize only DHT nodes and IDs; user settings, plugins and proxy
  configuration remain authoritative. The 1 MiB file uses atomic private writes
  every five minutes and at graceful stop. Empty snapshots preserve useful old
  nodes. Libtorrent 2.1 can start DHT before UDP sockets exist; up to 32 native
  node hints per address family are reintroduced after listener readiness and
  subsequent changes through its public API. No new discovery engine is added.
- Startup reasons distinguish metadata, peers, first data, header/index,
  prebuffer and bounded probe grace. New unobserved timings are -1 (UI dash),
  distinct from measured zero. Metadata/DHT times start at registration; other
  startup times start at preload. Cache sampling is a server observation, not
  the Android decoded first frame. Labels/help are translated in all seven
  existing languages.
- One worker owns each torrent's preload. Same-file calls share readiness;
  switching files cancels/joins the old worker and tail-refinement owner before
  replacing reservations. Successful explicit preload releases its response at
  the gate while preserving the bounded eight-second warm handoff. Removal and
  shutdown cancel/join it. Non-probe calls retain synchronous cleanup. Reader
  takeover is scoped to the selected file, and priority cleanup reconciles live
  windows in the existing cache.

## Verified development executable

Runtime source: `6d5c9129d2e6fb9553034594235667ed53acad1a`.
Windows version: `MatriX.145.Flow-dev-3d31000157e2c675cb619871c6a585c072e83323`.
That identity is GitHub's synthetic PR merge; its tree
`329754f934c22cb636480938e0619193e93dd724` matches the runtime source tree exactly.

Local executable: `.tools/next-6d5c9129/windows/TorrServer-LT-windows-amd64.exe`.
SHA-256: `a71cb479712c67ea093a45b668082ca6b4908eb38eab882d881cb8a02ada3149`.
The downloaded archive matched artifact `11281234461`'s published SHA-256:
`f9940ab6f6672246916e29fea52dcdaab8cd35432360d3cca883e8112578607d`.
The published `.5` release is unchanged. Launch using the same state directory
and port as your usual instance; these checks used separate disposable state.

| Windows controlled case | Buffer ready | Explicit response | Concurrent same-file calls | Removal during handoff |
| --- | ---: | ---: | ---: | ---: |
| Legacy | 229 ms | 229 ms | 14 ms | 2.5 ms |
| Custom, 30-second reconnect | 232 ms | 232 ms | 3.2 ms | 2.4 ms |

Both retained their one healthy seeder connection. The preceding fixed-peer
build `4eed4944` already buffered in about 0.22 seconds but held the explicit
response for about 8.2 seconds. These matched local runs verify removing that
response wait, not a universal public-swarm or phone speedup.

Local reports are under `.tools/next-6d5c9129/{legacy-check,custom-check,discovery-check,ui-smoke-final,readonly-check}`.
Episode switching and exact live Range bytes passed; history retained the
engine-stop event without fixture names/hashes. Corrupt, oversized, valid and
disabled DHT files all reached local readiness with the expected restore state.
A real read-only launch restored valid hints while leaving history, routing and
settings files unchanged. The actual embedded UI showed both default values,
applied opt-in history through the existing confirmation flow, persisted it,
and started its writer without page errors.

## Verification gates

Runtime source `6d5c9129` passed:

- [Native run 37145629424](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37145629424):
  all six cross targets, native Go unit/vet/race/source-vulnerability/fuzz gates,
  actual Go/frontend contract validation, authenticated maintenance, console,
  controlled playback/resources, both startup profiles and state-recovery cases,
  plus Windows service/update/configuration rollback.
- [macOS run 37145629402](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37145629402):
  both architectures and native tests, including the real local DHT node query,
  verified routing snapshot and live UDP-listener recovery.
- [Portable run 37145629416](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37145629416).
- Frontend type checking, lint and production build; 42 unit cases plus the
  native-exported contract case (43 distinct cases total); all 31 browser
  scenarios. The contract case is intentionally skipped without its native
  fixture in the standalone web job and exercised by the native job.
- Local real-Windows history rotation/privacy/overflow/restart/partial-record,
  bounded-close and atomic/corrupt-state tests; actual-executable checks above.
  Sandbox-only temporary-file rename restrictions were resolved by rerunning
  with real Windows permissions. An empty successful torrent-removal response
  was corrected in the harness rather than changing the compatible API.

Python compilation, actionlint and final diff checks passed. All locally owned
test processes exited; no personal playback state was used or changed.

## Specification reconciliation

Reviewed the original core plan and modern-interface plan against this scope:
core §8–9 and §30–33, modern UI §23 and §32, plus the existing acceptance ledger.
The existing bootstrap, bounded asynchronous probe, one cache, reader windows,
settings migration, native engine and HTTP behavior remain integrated. New
fields are additive and optional in frontend types for older-server support.
No fixed Just Player Range sequence, arbitrary timeout reductions, duplicate
media cache or speculative resolver/provider stack was introduced.

No architecture deviation is required for these four improvements. The native
late-socket workaround implements useful DHT-state reuse using existing native
APIs. Public-swarm cold/warm benchmarks, actual phone request/frame comparison,
multi-hour representative-media and physical boot/sleep/network acceptance
remain outstanding; the stable gate in `RELEASE_ACCEPTANCE.json` stays false.
