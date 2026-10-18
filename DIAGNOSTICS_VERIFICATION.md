# Diagnostic evidence verification — 18 October 2026

## Candidate provenance

- Diagnostic source: `8a9587c646dc4363d368a530457c2ab9d18b6ce9`.
- CI merge revision: `2f1671688505e82e3d0f31fbf8472c6654537c2e`;
  parents are master `26f22bb3` and the diagnostic source above.
- [Native build 39137894655](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39137894655).
- Windows executable SHA-256:
  `e82febdb6ee07f283a918c9cd169501574546307cec6c0d97e098535e812264b`.
- Artifact ZIP matched GitHub's digest:
  `73bb8f660f92aa2d7b539a4fd38217f3ce087ff9befac3258e50b18fa447e8a1`.
- Development version: `MatriX.145.Flow-dev-2f1671688505e82e3d0f31fbf8472c6654537c2e`.
  Go reports `vcs.modified=true` because CI generates the embedded production web
  bundle before compilation. This is a development candidate, not a new stable tag.

The analyzer/test/documentation follow-up `c739c4a6` changes no compiled server,
native bridge or embedded web source. This page records that earlier diagnostic
candidate. The subsequent startup-demand fix has its own native comparisons and
must be verified separately; the checksum above does not identify that newer fix.

## Passed checks

- Cross-compilation: Windows amd64, Linux amd64/arm64/armv7 and Android arm64/armv7.
- [macOS builds](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39137894542): amd64/arm64.
- [CodeQL](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39137894550): native bridge, Go,
  JavaScript/TypeScript, Actions and Windows interface helper; PR check passed.
  Existing findings on other refs were not dismissed by this change.
- CI native unit, race, vet, reachable vulnerability and bounded fuzz checks;
  controlled playback/resource cycles, API contracts and authenticated diagnostics.
- Windows service/interface-helper lifecycle, cancellation, update and rollback.
- Eight controlled 90/120 Mbps high-bitrate cases, with rotated profile comparisons.
  These validate HTTP byte transport, not Android decoded playback.
- Frontend: 106 tests passed, one existing skip; typecheck, lint and production
  build passed. English/Russian drawer checked at 375 × 812 using mocked startup
  evidence: no horizontal overflow or browser errors.
- Local portable Go recorder, shutdown, console, log and embedded-template suites.
- Local Python suite: 49 tests passed, including four analyzer tests covering old
  logs, redaction, archive selection, idle demand and unknown piece evidence.

## Actual Windows executable checks

Both tests used generated local media, fresh state and a loopback seeder.

- `startup_harness.py`: buffer ready in **1,241.5 ms** with a healthy connected
  peer; verified media bytes, concurrent preload, episode switch and removal;
  six startup snapshots, actual HTTP 206 response and API shutdown cause retained.
- `diagnostic_history_check.py`: **4,915,200 bytes** verified over an approximately
  25-second request, three periodic playback snapshots without a diagnostics
  page or Range tracing, completed and cancelled responses retained.
- Both runs reported **zero dropped records and zero recorder errors**.
- Forced termination of an owned test process followed by restart produced
  `previous_run / UNCONFIRMED_TERMINATION`; the next clean API shutdown recorded
  both stop intent and completed engine teardown.
- History contained no generated media names, torrent hashes or client address.

Local machine-readable results are in ignored `.tools/diagnostic-native-startup-8a/report.json`,
`.tools/diagnostic-native-history-8a/report.json` and `.tools/diagnostic-ui-20261018/report.json`.

## Limits

No new real-phone episode, multi-hour session, physical power-loss or Windows
logoff/shutdown acceptance run was performed for this candidate. A missing final
record is uncertain evidence, not a crash diagnosis. Recording remains opt-in,
asynchronous and bounded; disk failures, queue drops and retention can remove
evidence. Server-side waits cannot identify exact decoded-player stutters.

User playback settings, cache size, swarm profile and scheduling behavior were
not changed. Stable 1.1.3 and its published release remain available.
