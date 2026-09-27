# TorrServer-Flow development state

This checkout tracks `trinity-aml/TorrServer-LT` on the `upstream` remote and
`gnbk21/TorrServer-LT-Flow` on `origin`. The `develop` branch contains Flow work.

## First implementation milestone

Flow settings live under `BitTorr.Flow` in the existing settings JSON and are
also returned by the existing `POST /settings` API. Existing settings files
without a `Flow` object receive the documented defaults. An older web client
that omits `Flow` when saving settings preserves the current Flow values.

The authenticated `GET /flow/status/{hash}` endpoint reports the current startup
stage and per-session playback, cache and Range diagnostics. It does not create
a new cache. Range traces are disabled by default; set
`BitTorr.Flow.RangeTraceEnabled` to `true` to keep the latest 128 requests per
logical session in the diagnostic response. `DebugFlow` additionally emits JSON
trace lines to the server log. A trace classification is a hint and never affects
HTTP responses.

The startup path uses the upstream preload's container-aware tail reservation
and MP4 `moov` refinement. It first schedules a configurable bootstrap head
(default 16 MiB) plus the upstream tail. It then fills at least the provisional
32 MiB head target. If ffprobe is available, it runs after bootstrap in a
separate goroutine. A credible bitrate can raise the head target, capped at
128 MiB by default. The player waits at most the configured 1500 ms probe grace
after the provisional gate is satisfied. A failed or unavailable probe leaves
the provisional target in place. The internal probe reader remains separate
from player session grouping.

The default warm timeout is 600 seconds. On the last player reader close, a
small region around its playhead remains protected in the existing piece cache
until reconnect or timeout. Status polling does not reset this deadline.
Setting `BitTorr.Flow.Enabled=false`
restores upstream's timeout and percentage-based preload behavior. The
`Connection: close` removal remains an HTTP compatibility fix in both modes.

## Adaptive buffering and seek follow-up

`BitTorr.Flow.AdaptiveReadAhead` defaults to true. The existing cache window
now chooses a bounded forward reach from probed/derived bitrate and, after
enough valid progress samples, observed playback consumption. Its starting
target is 45 seconds (`TargetBufferSeconds`) with a 180-second ceiling
(`MaxBufferSeconds`). Download sustainability and recent piece waits can raise
the target; a healthy swarm can lower it. The cache's configured budget and
upstream head/tail pins remain authoritative. Setting `AdaptiveReadAhead=false`
restores the upstream percentage-based window without disabling other Flow
features.

Flow diagnostics now include observed rate/confidence, chosen forward window,
target buffer seconds, first body-byte timing, and seek/warm Range TTFB. Buffer
ahead counts contiguous readable blocks, including a partial piece being served
responsively. These are server measurements; player-visible seek and resume
times still require the phone. A multi-file offset bug in the existing blocked
piece cleanup was corrected so a completed seek cannot leave an obsolete
piece forced indefinitely.

## Reproducible comparison protocol

The supplied `TorrServer-LT-windows-amd64.exe` is a MatriX.145.LT-1.1.9
baseline. Before claiming a playback improvement, run it and a Flow build with
separate `--path` directories, the same torrent corpus and cache settings, and
the same PC, LAN and Just Player version. Do not share the production
`config.db` between concurrent processes.

Record at least these classes: 5–10 Mbps 1080p, 15–40 Mbps 4K, 50–100 Mbps
REMUX, 100–150+ Mbps peaks, weak swarm, and seek-heavy playback. For each,
capture the Just Player Range pattern, time to first playable frame, warm
reconnect Range TTFB, seek recovery, stalls, peak RSS and CPU. Include MP4 with
front and rear `moov` and MKV with non-linear index access. Repeat runs and
compare p50/p95/p99 where the sample size permits. Record the torrent hashes
and file indexes privately so later runs reuse the same data. Do not publish
copyrighted media or credentials as fixtures.

The plan's later scheduler tuning, service/tray, custom DNS, and provider phases
depend on those measurements and a working Windows/libtorrent build. No
performance result is asserted by this document.

## Windows build and self-test

### Windows background service

From an elevated PowerShell window, the Windows binary supports:

```powershell
./TorrServer-LT-windows-amd64.exe --service install
./TorrServer-LT-windows-amd64.exe --service start
./TorrServer-LT-windows-amd64.exe --service stop
./TorrServer-LT-windows-amd64.exe --service restart
./TorrServer-LT-windows-amd64.exe --service uninstall
```

Install uses Automatic (Delayed Start), the wildcard HTTP listener, and
`%ProgramData%\TorrServer-Flow` for service data by default. Supply `--path`
and other ordinary server flags during install to retain them in the service
command line. Service logs default to `flow.log` in that data directory.
External DNS is no longer checked before the local server starts. The service
has no interactive window; open the HTTP UI from a browser. The tray companion
specified for a later phase is separate from the service.
The authenticated `/flow/network` endpoint reports local address readiness,
the next check, and reannounce attempts. `ADDRESS_READY` means a usable local
address exists; it does not certify Internet or tracker reachability. The
separate `connectivity` field stays `INTERNET_WAIT` until a tracker reply,
becomes `DEGRADED` after a later tracker error, and returns to `ONLINE` after
another reply. Flow checks periodically and retries with bounded backoff when
addresses are absent or a reannounce operation fails.
At startup the HTTP listener binds before libtorrent initializes; `/echo` and
`/flow/network` remain available during that brief initialization, while other
routes return 503. Service stop closes listeners and gives active HTTP requests
up to five seconds to finish before closing the torrent session and database.

### Windows tray companion

Run `tray/FlowTray.ps1` in Windows PowerShell with `-STA` in the user session.
The **TorrServer-Flow-Tray** CI artifact packages the script separately from
the server executable. For example:

```powershell
powershell.exe -NoProfile -STA -WindowStyle Hidden -ExecutionPolicy RemoteSigned -File .\FlowTray.ps1
```

The tray reads the local authenticated `/flow/tray` endpoint every five seconds
and shows server state, active torrent, download speed, playable buffer and peer
count. Its menu opens the UI, requests an elevated service restart, pauses or
resumes torrent activity, and exits the tray without stopping the service.
When HTTP authentication is enabled, pass a credential only for loopback use:

```powershell
powershell.exe -NoProfile -STA -WindowStyle Hidden -ExecutionPolicy RemoteSigned -File .\FlowTray.ps1 -Credential (Get-Credential)
```

### Swarm profiles

`BitTorr.Flow.SwarmProfile` in `settings.json` accepts `legacy`,
`conservative`, `balanced`, `aggressive`, or `custom`. Restart the server after
changing it. `legacy` preserves the fork's prior libtorrent values and remains
the default until a controlled benchmark supports changing the default.
`conservative` leaves swarm tuning near libtorrent defaults. `balanced` uses
50 connection attempts and a 50-peer connect boost; `aggressive` uses 100 and
80. These are experimental choices, not proven faster on every network.
`custom` applies only positive values supplied in `SwarmCustom` for
`ConnectionSpeed`, `TorrentConnectBoost`, `PeerConnectTimeout`, `PieceTimeout`,
`RequestQueueTime`, and `MinReconnectTime`; zero leaves that libtorrent value
untouched. Cache limits, proxy settings, upload choices, and active-torrent
queue protection are retained for every profile.

The authenticated `/flow/status/<hash>` response also includes a bounded
tracker event summary by protocol and host. Its identifiers hash the original
URLs; paths, queries, userinfo, and tracker passkeys are never returned. A
tracker reply is reported as success only after libtorrent emits that reply.

The web settings dialog has a **Flow** tab for the startup buffer, adaptive
read-ahead, warm mobile sessions, network retry bounds, swarm profile and
diagnostic switches. The main cache control provides 256, 512, 1024, 2048 and
4096 MB presets plus a custom size. Choose a preset according to available RAM;
none is presented as a universal recommendation. A torrent card opens live
Flow diagnostics during playback, including buffer seconds, measured download,
cache use, peer count, piece waits, seek recovery and tracker status. The
status endpoint is authenticated like the existing settings API. Saving
settings restarts the torrent engine and stops active streams; the newly
selected swarm profile applies during that restart.

The Windows executable depends on native libtorrent, Boost and OpenSSL. The
repository's `.github/workflows/build.yml` builds those dependencies on Ubuntu
with MinGW-w64 and uploads a `TorrServer-LT-windows-amd64` artifact. After
creating a GitHub fork, push this `develop` branch to it, then run the **build**
workflow with **Run workflow** and select `develop`. Download the Windows
artifact after it succeeds. This is the supported reproducible Windows build
route; a local Go-only build cannot link libtorrent.

For a local side-by-side test, start each executable from a separate PowerShell
window, with separate ports and fresh state directories. For example, from the
parent `torrs` directory (adjust executable names to match the artifact):

```powershell
./TorrServer-LT-windows-amd64.exe --port 8090 --path ./baseline-test
./TorrServer-Flow/TorrServer-LT-windows-amd64.exe --port 8092 --path ./flow-test
```

Do not point either process at the existing `torrs/config.db`. Configure both
with the same cache, peer, and other upstream settings. Import the same legal
test torrents in both instances. On the phone, open the respective LAN URLs
and play the same files in Just Player. Run each case more than once, alternating
the server order so swarm changes do not favor one build. Capture the Flow
diagnostics at `http://<PC-LAN-IP>:8092/flow/status/<torrent-hash>`; if HTTP
authentication is enabled, use the same credentials as the web UI. The Range
trace is opt-in in `flow-test/settings.json`: set
`BitTorr.Flow.RangeTraceEnabled` to `true` while the Flow process is stopped,
then restart it. Record Just Player version, startup to first frame, any stalls,
seek recovery, phone disconnect/reconnect delay, and server CPU/RAM for both.
The API's TTFB and startup timings are server observations; first playable frame
must be measured on the phone.

`buffer_exhaustion_seconds` estimates when the contiguous playable buffer will
run dry at the current measured download and consumption rates. A
`buffer_warning` is raised below 30 seconds during active playback; the field
is absent when the buffer is not draining or a rate cannot be estimated.
