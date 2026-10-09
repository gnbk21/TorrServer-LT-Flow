# Console presentation verification

Verified 1 October 2026. Native development source:
[`034d4bf5`](https://github.com/gnbk21/TorrServer-LT-Flow/commit/034d4bf5ae32326b3be76ab103719db84ffbfdcd).
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

- [Native CI run 36842623470](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/36842623470)
  passed all six cross targets, frontend verification, native unit/parser tests,
  vet, console/log/engine race tests, reachable Go vulnerability scanning, fuzz
  campaigns, generated-peer playback/resource cycles and Windows restricted
  service/update/rollback regressions.
- [macOS run 36842629448](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/36842629448)
  passed both architectures at the same final source `034d4bf5`.
- Seven real-server cases passed independently on Windows and Linux:
  automatic redirected output with heartbeat; plain output with reports off;
  legacy mode; UTC file logging; HTTPS/HTTP redirect configuration;
  machine-readable version/doctor commands; invalid options and occupied-port
  errors. All five owned Windows instances logged successful shutdown, exited
  with code zero, and left no owned process running.
- A Windows ConPTY test launched the actual final native executable with HTTP
  access logging enabled. With process-local color opt-outs removed, auto mode
  enabled VT output (`3 → 7`), rendered the startup summary, exited successfully
  through API shutdown and restored exactly `3`; the access log remained usable.
  A separate real-console probe covered INFO/WARN/ERROR colors and resets.
  Tests cover pipe fallback and `NO_COLOR`. The Codex shell's default
  `TERM=dumb`/`NO_COLOR` correctly suppresses colors.
- Portable Go tests, console/log vet, eight distribution Python tests,
  Python syntax, actionlint and final diff checks passed locally.

The first native console run caught an overly broad test assertion matching
the LAN help sentence. The assertion was corrected and both native platforms
passed the complete seven-case harness.
Final review also found that the direct API exit must restore the console
without closing HTTP access loggers before other handlers finish. Console-only
cleanup now preserves those loggers until OS process exit; a real-file
regression test checks logging after cleanup. Standard log output is changed
before restoring console mode, joining any final colored write.

## Development artifact and reproduction

Windows executable from the final native CI run:

```text
TorrServer-LT-windows-amd64.exe
Version: MatriX.145.Flow-dev-034d4bf5ae32326b3be76ab103719db84ffbfdcd
SHA-256: d84ce3d4b0aebc835bc9ff11ca9a3992a0e3e61da106d1a365273df132b4030e
```

Local evidence is retained under `.tools/console-034d4bf5/`: `windows-check`,
`linux-console-check` and `native-terminal-check` reports and captured output.
The initial color probe is under `.tools/console-2d3afaf6/terminal-report.json`.
These ignored artifacts are not repository source.
Public Linux evidence is in the CI artifact `flow-console-check`.

Run the committed harness against a native executable with a fresh output
directory:

```powershell
python build/console_harness.py --executable .tools/console-034d4bf5/windows/TorrServer-LT-windows-amd64.exe --output .tools/console-check-new
```

It creates isolated state and loopback listeners, then stops its owned
processes. It does not use the existing playback database.

This is a development build, not a new published release. Real phone/media,
multi-hour resource and physical Windows recovery acceptance remain the
separate stable-release gates in [STATUS.md](STATUS.md).
