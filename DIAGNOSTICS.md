# Diagnosing startup and delivery delays

Flow's web diagnostics show live server observations. Enable **Diagnostic
history** in Flow settings to retain evidence after playback ends. Detailed Range
tracing is independent and is not required for the periodic history below.

## Playback diagnosis and phone setup

The playback overview follows metadata discovery, peer connection, required
head/index bytes, optional reserve and HTTP delivery in one view. It shows the
current blocker, evidence age, confidence and one suggested reversible step.
Old or missing observations remain unknown. The adviser never changes settings,
profiles, trusted origins or quality. Server buffers and read waits do not measure
the player's decoded buffer or displayed frames. Search/parser availability is
shown separately in Search; a parser outage does not establish a torrent-network
fault. Full episode preparation remains an explicit optional operation.

**Connect phone → Guided phone setup** checks listener interfaces and advertised
LAN addresses, then creates a five-minute read-only connection proof. Scan that
proof's QR from the phone. A request from a different network address proves that
another client reached Flow; it does not identify the physical device. A PC-only
preview is labelled separately. Related authentication or browser-origin rejection
evidence appears after the visitor is observed. Unseen requests cannot be
diagnosed conclusively: check the chosen address, Wi-Fi/router isolation and the
app's local-network permission. Android behavior depends on its version, the app
and target SDK; Flow cannot grant another app's permissions.

Proof creation and status use the existing management authentication/origin/rate
policy. If HTTP authentication is disabled, the existing trusted-LAN access
policy still applies. Enable authentication when that is not appropriate. A
proof token grants no management or playback authority. At most 64 proofs and
128 short-lived access observations are held in memory; neither is persisted.
Setup remains available while the torrent engine starts. Creating a proof does
not add a trusted management origin. Add an origin only after explicitly deciding
to trust that exact application.

Expand **Media information** for a selected file to see metadata already obtained
by the bounded startup probe: codec, audio/subtitle tracks and available bitrate.
This read does not initiate a probe or inspect every search result. Unknown
information stays unknown. Direct playback preserves the media bytes; optional
GStreamer conversion is a separate explicitly selected path. Diagnostics also
show exact-byte reuse eligibility, accepted candidates and natively verified
bytes, plus metadata recovery and next-episode warmup state. A matching candidate
alone is not proof that bytes passed native verification.

## Optional incident traces and storage previews

**Settings → Advanced → Incident flight recorder** is
default-off. When enabled on a writable installation, Go's flight recorder targets
30 seconds of recent execution history with a 32 MiB size hint. That hint is not
a total-memory cap. A server read stall of at least 500 ms or startup failure can
save one asynchronous capture, at most once every five minutes. Each raw output
has a separate 64 MiB hard limit; incomplete/oversized output is discarded. At
most two raw captures are retained. Memory pressure pauses the recorder and it
resumes after pressure recovery. A conflicting runtime trace or storage failure
is reported without failing playback.

Raw traces can contain private process details. They are private files in
`flow-incidents`, excluded from ordinary support reports and portable exports.
Under **Settings → Advanced → Runtime storage**, explicitly select raw traces
before downloading one. Retained captures remain accessible with the recorder
disabled. ETW/Perfetto remain the tools for native C++ attribution.

The runtime storage scan is read-only and separates evicting disk cache, retained
preparation data, diagnostics, backups and resident RAM cache. It skips links and
uses a one-second/10,000-entry traversal limit; an incomplete category is labelled.
It does not delete files. Developer-workspace inventory and cleanup are separate;
see [the build guide](build/README.md).

## What is recorded

- `startup_sample`, at most once every two seconds plus a final snapshot:
  received and verified opening-buffer/end-of-file bytes, target sizes, missing
  piece counts and the first missing head/index piece. A received but unverified
  shared boundary piece remains missing. During `bootstrap`, the head target is
  the smaller probe head; the unscheduled phase-2 remainder is not blamed as a
  blocker. The live drawer shows head/index evidence separately.
  Fresh `head_supply` / `tail_supply` observations now include connected suppliers,
  unchoked suppliers for an exact one-piece sample, priority, requested/unrequested
  blocks and received/writing blocks. A missing observation is unknown; a truncated
  peer sample gives lower bounds. Preload blockers participate in the existing
  bounded native sampling even before a player connects.
- `playback_sample`, every ten seconds while clients are active or recently
  disconnected: response demand and delivery offsets, contiguous buffer, download
  and measured useful delivery rates, estimated consumption, local source
  availability, recent server read waits and request/cancellation totals.
  At most eight recent client sessions per torrent are retained per sample;
  `truncated` marks omitted sessions. Session numbers are process-local counters.
- `response`: completed external HTTP response status, bytes, first-byte delay,
  duration, cancellation and diagnostic classification. Internal media probes
  are excluded. `first_byte_observed` distinguishes a measured zero-millisecond
  first byte from a response that never delivered any body bytes. There are no
  request URLs, headers or client addresses.
- `peer_summary`: disconnect counts grouped by local torrent sequence,
  numeric error/operation and allowlisted reason, flushed every ten seconds and
  on recorder close. The aggregation holds at most 64 groups; overflow is counted
  in a marked `OTHER` group. Peer chatter does not fill the incident timeline.
- `shutdown_requested`: API, signal, service stop, server error or shutdown timeout.
  Exit paths request a bounded history sync. `engine_stopped` is recorded after
  native session destruction completes. Console Ctrl+C/close requests are handled
  on Windows as well as the existing Unix/Android signal paths.
- `previous_run / UNCONFIRMED_TERMINATION`: on recorder restart when retained
  history lacks a completed engine stop. This is **not** a crash diagnosis.

History remains opt-in, asynchronous and bounded to four files of at most 1 MiB
each. It excludes names, hashes, URLs, IPs, client tokens and arbitrary error text.
Each JSON line is limited to 4 KiB. Producers never wait for disk or queue space;
the recorder reports dropped records and write errors. No new media cache is used.

## Read a completed session

From this source checkout, with Python 3 installed:

```powershell
python build/analyze_flow_history.py "C:\path\to\Flow\flow-history.jsonl"
```

The analyzer reads only this file and its `.1`, `.2`, `.3` archives, selects the
latest recorder run, and prints an identity-free JSON summary. It supports older
logs; missing periodic evidence is left unknown. It reports exact median and
nearest-rank 95th percentile for **recorded** response/recovery starts, not video
frame stalls. The report includes cancellation/error counts and latest delivery
samples. It does not change or start the server.

## How to interpret the evidence

- `HEAD`, `INDEX`, or `HEAD_AND_INDEX` identifies the unverified portion holding
  the startup gate. Samples are spaced observations, not exact stage durations.
- `PROBE` means the required pieces are verified but the configured bounded
  probe grace is still active. Lowering the probe timeout will not fix missing
  startup pieces.
- A known zero supplier count means none of the sampled connected peers advertised
  this piece. Suppliers with no unchoked source indicate peer choking; a known
  queue with unrequested blocks exposes a different situation from a missing
  availability sample. These are observations, not automatic proof of the cause.
- Adaptive keeps the ordinary 120-second startup wait unless useful gate bytes
  are still arriving and some required bootstrap data is verified. At expiry,
  at least 1 MiB of progress over the recent observation window (up to 30 seconds)
  and progress within ten seconds allow another 30 seconds, up to a 300-second
  absolute maximum. Other profiles keep 120 seconds. This never bypasses required
  index data, piece verification or the configured startup-buffer target.
- Straggler escalation applies only to the active bootstrap/prebuffer phase and
  preserves existing deadlines. It no longer promotes unscheduled phase-two head
  pieces or rewrites the tail/head ordering every 200 ms.
- Periodic demand/delivery offsets describe the server response. The player can
  pause, probe, cancel, reconnect or buffer ahead; none proves its decoded position.
- `recent_read_stalls` and wait percentiles are shared per torrent (including
  internal cache activity), not separate per-player stall counters. Repeated
  snapshots overlap the rolling 60-second window; do not sum them.
- Low delivery while demand is idle is not evidence that the swarm cannot supply
  the video. Rate confidence, active demand and local supplier evidence matter.
- `NO_STOP_RECORD` can mean a still-running process, force termination, power loss,
  exhausted retention, disabled recording or a recorder failure. The next startup
  can mark missing closure; it cannot recover the precise cause or a missing stack
  trace. Native crashes and force-kills cannot reliably run shutdown handlers.

The stable 1.1.3 executable predates these additional records. A binary built from
the diagnostic changes is required to collect them; an analyzer cannot reconstruct
missing observations from old logs. These changes improve evidence collection,
not torrent availability or guaranteed startup/playback speed.

## Reproduce the controlled checks

With generated fixtures from `build/fixtures.py`, run against an absolute path to
the executable (each output directory must be new):

```powershell
python build/startup_harness.py --executable C:\Flow\TorrServer-LT-windows-amd64.exe --fixtures .tools/controlled-fixtures --output .tools/startup-check --assert-fast
python build/diagnostic_history_check.py --executable C:\Flow\TorrServer-LT-windows-amd64.exe --fixtures .tools/controlled-fixtures --output .tools/history-check
```

These checks own their processes, loopback seeder and fresh state. The second
holds a real byte-verified response across multiple sampling intervals with Range
tracing disabled, checks completion/cancellation evidence and privacy, then kills
only its own test process to check the next-run receipt. It never attaches to a
running user's server. Results are written to each output's `report.json`.

See [diagnostic verification](docs/history/DIAGNOSTICS_VERIFICATION.md) for the tested build,
results and remaining field-validation limits.
