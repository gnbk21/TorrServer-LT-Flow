# Console presentation verification

Verified 1 October 2026. Native development source:
[`2d3afaf6`](https://github.com/gnbk21/TorrServer-LT-Flow/commit/2d3afaf6e225394884a64bf3f9b4616baf39b35a).
The public Preview 2 `.5` release predates this feature.

## Implemented behavior

| Requirement | Implementation and evidence |
| --- | --- |
| Organized launch information | A grouped connection/configuration/runtime/help summary is printed after the listeners and engine are ready; native Windows/Linux output checked |
| Useful connection addresses | URLs use actual resolved ports, bind hosts and HTTP/HTTPS mode; IPv6, duplicate hosts, private LAN candidates and loopback-only behavior covered |
| Readable logs | Local timestamps, explicit lifecycle severity/component labels and prefix-based severity for legacy text; colors are reset after each styled span |
| Safe metadata display | Console control sequences, control characters and bidi formatting controls are removed or replaced; multiline logs retain timestamp/severity prefixes |
| Live observations | Actual stream requests, engine/network state, Flow pause, cache data, process RSS, Go heap and goroutines; five-second state sampling and configurable heartbeat |
| Limited observation overhead | State polling uses existing snapshots/atomics; cache/memory totals are collected only when a report is due; no torrent activation, native session dump, HTTP polling or parallel cache |
| Presentation controls | `--console auto/plain/off`; `--console-interval 0` disables reports, otherwise 5–3600 seconds; defaults auto/30 |
| Compatibility | Services and successful `--logpath` retain UTC file logging without the summary/heartbeat; `--version`, `--doctor` and existing settings/API/storage remain compatible |
| Cleanup and errors | The reporter is canceled/joined before exit, including API shutdown; concurrent cleanup restores terminal mode once; startup errors have actionable messages and close logs |
| Browser launch | `--ui` opens an actual bound URL after readiness, including the resolved HTTPS port |

Cache data measures cached piece extents in RAM/disk storage, not process RAM
or playable-buffer length. `INTERNET_WAIT` is unconfirmed tracker connectivity,
not proof of an Internet outage. Flow pause is the server's download pause
control; it does not infer whether Just Player has paused the episode.

## Verification

- [Native CI run 36837315657](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/36837315657)
  passed all six cross targets, frontend verification, native unit/parser tests,
  vet, console/log/engine race tests, reachable Go vulnerability scanning, fuzz
  campaigns, generated-peer playback/resource cycles and Windows restricted
  service/update/rollback regressions.
- [macOS run 36836929904](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/36836929904)
  passed both architectures at `5830e36b`. Its production Go/UI/build sources are
  identical to `2d3afaf6`; the intervening change corrects one Python harness
  assertion to match advertised LAN URL rows rather than help text.
- Seven real-server cases passed independently on Windows and Linux:
  automatic redirected output with heartbeat; plain output with reports off;
  legacy mode; UTC file logging; HTTPS/HTTP redirect configuration;
  machine-readable version/doctor commands; invalid options and occupied-port
  errors. All five owned Windows instances logged successful shutdown, exited
  with code zero, and left no owned process running.
- A Windows ConPTY probe used the actual console package and OS console handle.
  With process-local color opt-outs removed, auto mode enabled VT output
  (`3 → 7`), emitted INFO/WARN/ERROR colors with resets, and restored exactly `3`.
  Separate tests cover pipe fallback and `NO_COLOR`. The Codex shell's default
  `TERM=dumb`/`NO_COLOR` correctly suppresses colors.
- Portable Go tests, console/log vet, eight distribution Python tests,
  Python syntax, actionlint and final diff checks passed locally.

The first native console run caught an overly broad test assertion matching
the LAN help sentence. The assertion was corrected and both native platforms
passed the complete seven-case harness.

## Development artifact and reproduction

Windows executable from the final native CI run:

```text
TorrServer-LT-windows-amd64.exe
Version: MatriX.145.Flow-dev-2d3afaf6e225394884a64bf3f9b4616baf39b35a
SHA-256: 1e4c6c376395aeb9bc9f1f27233de157fb5cbcc65c842410e672d45fa5df2107
```

Local evidence is retained under `.tools/console-2d3afaf6/`: Windows and Linux
`*-console-check` / `windows-check` reports, captured output and
`terminal-report.json`. These ignored artifacts are not repository source.
Public Linux evidence is in the CI artifact `flow-console-check`.

Run the committed harness against a native executable with a fresh output
directory:

```powershell
python build/console_harness.py --executable .tools/console-2d3afaf6/windows/TorrServer-LT-windows-amd64.exe --output .tools/console-check-new
```

It creates isolated state and loopback listeners, then stops its owned
processes. It does not use the existing playback database.

This is a development build, not a new published release. Real phone/media,
multi-hour resource and physical Windows recovery acceptance remain the
separate stable-release gates in [STATUS.md](STATUS.md).
