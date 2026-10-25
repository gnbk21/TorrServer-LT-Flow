# Large-piece startup fix — 18 October 2026

## Changes and constraints

The straggler loop previously refreshed all remaining gate deadlines every
200 ms, including phase-two pieces while bootstrap was unfinished. On a torrent
with a few large pieces this defeated the intended head/index order and delayed
the EOF region. Escalation now uses only the active phase, keeps existing
deadlines, and limits newly urgent stragglers to eight pieces. Libtorrent retains
peer selection, retries and end-game responsibility.

Adaptive can renew the normal 120-second wait in 30-second steps while required
bootstrap pieces are verified and gate bytes keep arriving. Each renewal requires
recent progress of at least 1 MiB and progress within ten seconds; the absolute
limit is 300 seconds. No-progress/trickle/unverified cases and other profiles
retain the ordinary limit. Required index, whole-piece verification, startup
buffer targets, cache settings and transport encryption are not relaxed.

The existing bounded native sampler now observes missing preload head/index
pieces without needing a player reader. Retained/live evidence distinguishes
sampled suppliers, unchoked suppliers and requested/unrequested blocks. Unknown,
stale and truncated evidence remain explicit. The web drawer and all seven
translations expose the observations and extended wait; the offline analyzer
includes the latest redacted startup snapshot.

## Exact tested Windows binary

- Source: `c0e695c8b3c1d116b47e55c8729fa6a3ee7ab011`.
- [Build 39150739678](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39150739678),
  artifact `TorrServer-LT-windows-amd64`, ID `12145620978`.
- CI merge revision: `63c18d8be54303efb5b890961a3bd7c3e53cbdc1`;
  parents are master `26f22bb3` and the source above.
- Executable SHA-256:
  `ba04bbc69490db5580a2c9fdf03a8fd6097bc21636e9838ee50a64d78c63b410`.
- Artifact ZIP matched GitHub's SHA-256:
  `3c4321b5fad31a2c203f73aa6c5ffdddd44ebed008b9035edf880b673db7f027`.
- Go 1.26.9 / libtorrent 2.1.2; development version uses the CI merge revision.
  `vcs.modified=true` reflects the generated embedded production UI.

The follow-up corrects test-harness handling of the intentionally empty removal
response, comments and documentation. It changes no executable behavior. That
corrected harness was used for the missing-index check below.

## Controlled comparison

`build/startup_piece_check.py` uses original generated media larger than 64 MiB,
16-MiB pieces, a nonaligned file start, a separate EOF piece and a loopback peer.
Each run owns fresh state and a process; no public torrent or user server is
involved. The baseline is the earlier diagnostic build `8a9587c6` (SHA-256
`e82febdb6ee07f283a918c9cd169501574546307cec6c0d97e098535e812264b`),
not an unmodified upstream build. These are individual controlled comparisons,
not a statistical public-swarm benchmark or a decoded-player measurement.

| Observation | Earlier diagnostic build | Fixed binary |
| --- | ---: | ---: |
| Bootstrap ready, ordering case | 63.6 s | 32.4 s |
| First EOF-piece request, ordering case | 46.05 s | 3.04 s |
| Later head requested before bootstrap | Yes | No |
| Entire configured buffer ready, ordering case | 63.74 s | 63.44 s |
| Slow progressing peer, 320 KiB/s fixture cap | Timeout at 120 s | Ready at 206.72 s |

The fixed slow run retained the complete 64-MiB gate, byte-verified the returned
media range and renewed its budget to 210.2 seconds. The unchanged source rate
still limits full-buffer time. This fix does not manufacture peer bandwidth.

The missing-index case advertised no EOF piece. It never reached READY and
reported zero sampled/unchoked suppliers, zero requested blocks and 1,024
unrequested blocks at priority 7. Removal cancelled the owned preload promptly;
the index gate was not bypassed. Each native case reported zero recorder errors
and zero dropped history records.

## Other verification

- Local Go flow/diagnostic/console/log/template tests and flow/diagnostic vet:
  passed; phase-demand selection tested independently of native linking.
- Native Linux unit and race/vulnerability gates passed in the source build.
  Local Windows race mode is unavailable without a C compiler; it was not
  represented as a local pass.
- Frontend: 107 passed, one existing skip; typecheck, lint, production build and
  dependency notices passed. English/Russian drawer checked at 375 × 812 with
  supplier/queue/wait fixtures: no overflow or browser errors.
- Python: 50 tests passed, including five history-analyzer tests.
- Actual Windows normal Legacy startup: buffer ready in 1,247.5 ms; concurrent
  preload, episode switch, removal, byte ranges and retained shutdown evidence
  passed.
- Actual Windows history test: periodic snapshots without an open browser or
  Range tracing, completed/cancelled responses, clean API exit and owned
  forced-stop/restart receipt passed, with zero drops/errors.
- macOS amd64/arm64 and the six other source-build platform targets compiled.
  Full workflow results remain separately visible on the PR; individual passed
  gates do not imply an unfinished/superseded workflow passed as a whole.

Ignored local reports: `.tools/startup-piece-baseline-order-large/report.json`,
`.tools/startup-piece-baseline-progress-large/report.json`,
`.tools/startup-piece-candidate-order-c0/report.json`,
`.tools/startup-piece-candidate-progress-c0/report.json`,
`.tools/startup-piece-candidate-missing-c0-v2/report.json`,
`.tools/startup-normal-candidate-c0/report.json` and
`.tools/startup-history-candidate-c0/report.json`.

## Reproduce

Generate legal fixtures with `python build/fixtures.py --output .tools/large-fixtures --seconds 24`
(requires ffmpeg/ffprobe), then run each case with a new output directory:

```powershell
python build/startup_piece_check.py --executable C:\Flow\TorrServer-LT-windows-amd64.exe --fixtures .tools/large-fixtures --output .tools/large-order --case ordering --assert-fixed
python build/startup_piece_check.py --executable C:\Flow\TorrServer-LT-windows-amd64.exe --fixtures .tools/large-fixtures --output .tools/large-progress --case progress --assert-fixed
python build/startup_piece_check.py --executable C:\Flow\TorrServer-LT-windows-amd64.exe --fixtures .tools/large-fixtures --output .tools/large-missing --case missing-index --assert-fixed
```

## Remaining field limits

The user's actual problematic torrent and Just Player have not yet been retried
with this fix. No phone, multi-hour or physical Windows power-cycle acceptance
is claimed. Peer availability and throughput may still be insufficient for the
video's bitrate. Strict transport encryption and the user's selected profile,
experiments and 2-GiB cache were not changed by this implementation.
