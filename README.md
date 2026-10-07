# TorrServer-Flow

TorrServer-Flow is a fork of [TorrServer-LT](https://github.com/trinity-aml/TorrServer-LT) focused on streaming torrents from a Windows PC to an Android player over a local network. Its primary use case is direct playback in Just Player, including playback launched through Lampa. It keeps the libtorrent engine, existing media cache and HTTP API, and adds a modern embedded web interface.

> **Development status:** [Preview 2](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.145.Flow-v0.2.0-preview.5) is the current prerelease, built from `e0307c6d`. Its integration is tracked in [PR #3](https://github.com/gnbk21/TorrServer-LT-Flow/pull/3); see the [current status and exact remaining gates](STATUS.md). `develop` is the default integration branch; `master` is reserved for accepted stable releases. This preview is not a validated Flow / Modern Web 1.0 release.

## Run on Windows

1. Open [Preview 2](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.145.Flow-v0.2.0-preview.5), download **`TorrServer-Flow-windows-amd64-MatriX.145.Flow-v0.2.0-preview.5.zip`**, and extract it. This contains the modern interface, tray and managed install/update scripts. The standard Windows executable is also available separately.
2. In PowerShell, run the standard executable from the extracted folder, using a separate data directory for Flow:

   ```powershell
   .\TorrServer-LT-windows-amd64.exe --port 8090 --path .\flow-data
   ```

3. On the PC, open <http://127.0.0.1:8090>. On a phone on the same local network, use `http://<PC-LAN-IP>:8090` as the TorrServer address in your client (for example, Lampa), then play with Just Player. Allow the server through Windows Firewall on private networks if prompted.

Keep Flow and another TorrServer instance on **different ports and data directories**. Do not start them concurrently against the same `config.db`. The server listens on all interfaces by default. Keep it on a trusted private LAN or VPN. HTTP authentication (`--httpauth`, with an `accs.db` account file) protects management endpoints, but inherited playback routes can serve an existing torrent without credentials when its hash is known. Do not expose it directly to the Internet.

The artifact also contains a `-gst` executable for optional GStreamer HLS transcoding. Direct playback uses the standard executable and does not require GStreamer. The `-gst` variant needs GStreamer installed separately at runtime.

## Console display (development builds after Preview 2 `.5`)

The standard executable prints a grouped startup summary with web/LAN connection
addresses, state directory, cache budget, Flow/swarm settings and startup timing.
Logs have local timestamps and severity labels. Compatible terminals use subtle
colors; redirected output and `NO_COLOR` stay plain. Periodic status shows uptime,
engine/address/tracker state, active stream requests, cache data, process RSS,
Go heap and goroutines. State changes are sampled every five seconds; otherwise
the default heartbeat is 30 seconds. Cache/memory totals are collected only when
a status report is due. Cache data counts ordinary cache extents in the selected
RAM/disk store; retained preparation pieces have a separate disk reservation.
These totals are not process memory or a measure of playable buffer.
The web Dashboard provides detailed throughput and buffer
graphs. LAN candidates can include VPN adapters; tracker waiting does not prove
an Internet outage, especially when no torrent is active.

```powershell
.\TorrServer-LT-windows-amd64.exe --console plain --console-interval 60
```

Use `--console-interval 0` to disable status reports while keeping the startup
summary, or `--console off` for legacy log presentation. Intervals are 5–3600
seconds. Services and `--logpath` retain their existing UTC file logs without
the console summary or heartbeat. `--version` and `--doctor` remain machine
readable. For a clean shutdown, use **Settings → Advanced → Shut down** in the
web interface. The console does not accept interactive commands.
See the [console verification record](CONSOLE_AUDIT.md) for the exact native
source, executable identity and compatibility checks.

For a saved session log, add `--logpath .\flow.log` to your usual launch command.
This writes UTC file logs instead of the console panel. Console output alone is
not retained after exit. See [startup delay findings and improvements](PERFORMANCE_AUDIT.md)
for the peer-retention fix, controlled measurements and diagnostic guidance.

## What Flow adds

- **Adaptive startup:** schedules a bounded head buffer and the container index/tail; an optional `ffprobe` result can refine the startup target without blocking playback indefinitely.
- **Adaptive read-ahead:** sizes the forward cache window from media bitrate and observed playback, within the configured cache budget. It can be disabled independently of other Flow features.
- **Warm mobile sessions:** briefly protects data around the last playhead after a player disconnects, so a quick reconnect can resume from cached pieces.
- **Seek and Range diagnostics:** tracks startup, buffering, piece waits, seek recovery, and HTTP Range timing. A torrent card opens live diagnostics; detailed Range traces are opt-in.
- **Windows background operation:** includes a restricted service account and a separate tray companion. See [Flow development notes](FLOW.md) for service and tray commands.

Preview 2 also provides rolling 60-second health metrics, smoother adaptive
windows, bounded per-file probe reuse, process/cache memory observations,
redacted support downloads, portable backup/import, a startup doctor and an
idle-only Windows updater with integrity checks and rollback. Advanced settings
contain maintenance tools; importing settings requires preview/confirmation and
retains local credentials. The library renders 50 cards per page while searching
all entries. See [distribution guidance](DISTRIBUTION.md) and
[measurements](MEASUREMENTS.md) for usage and verification limits.

Flow settings are in the web interface's **Flow** tab and under `BitTorr.Flow` in `settings.json`. Flow is enabled by default. Published Preview 2 `.5` restarts the engine when settings are saved. Current development builds apply safe runtime changes immediately and offer **Apply when idle** for changes requiring an engine restart. `GET /flow/status/<torrent-hash>` exposes per-torrent diagnostics; `/flow/network` reports network readiness. Both follow the server's HTTP authentication setting. See [FLOW.md](FLOW.md) for defaults, endpoint behavior, and the comparison procedure.

## Adaptive reliability (current development branch)

Current development builds add verified useful-delivery rates, confidence and
buffer risk; shared active/warm RAM budgets; preparation scheduling; Windows
network notifications and recovery; configuration revisions, idle application
and last-known-good recovery. Settings shows unsaved, saved and effective values.
Optional origin/rate policies, expiring file links and torrent interface binding
are available without changing compatibility defaults. Interface binding is not
a verified VPN kill switch; OS DNS and physical disconnect tests remain separate.
See [the reliability guide](ADAPTIVE_RELIABILITY.md) for settings, behavior,
compatibility limits and test commands, and [the checklist](ADAPTIVE_RELIABILITY_CHECKLIST.md)
for verification and remaining device gates. These features are not in `.5`.

## Weak swarms and episode preparation (current development branch)

These additions are in PR #3 development builds, not published Preview 2 `.5`:

- **Availability diagnostics** distinguish metadata, discovery, connection,
  choke, missing connected-peer pieces, throughput and local cache waits.
  Samples are bounded and cached; unavailable or stale data is labelled unknown.
- **Prepare episode**, in **Files / Play**, retains the selected file in the
  existing piece store. Progress counts verified bytes; full-file readiness
  supports arbitrary seeks. The default disk quota is 4 GiB, separate from the
  evicting cache. Pause or cancel keeps progress; Remove releases it after readers
  close. Jobs survive server restarts and temporary source absence.
- **HTTP mirrors**, in the same dialog, use native Range requests, torrent
  hashes and peer fallback. A mirror must supply identical bytes. Imported URLs
  cannot access private destinations; a local mirror needs explicit approval.
  Torrents with mirrors wait for piece verification before serving their bytes.
- **Peer reachability** shows actual BitTorrent TCP/UDP ports, mapping results
  and incoming connections. New settings use port 51413; existing explicit port
  choices, including automatic port `0`, remain supported. HTTP port 8090 is separate.
- **Remember useful public peers** is optional and disabled by default. It keeps
  bounded private local hints for ten minutes and ignores them after network
  changes. Private torrents never restore those hints.
- **Rate-aware deadline experiment** is disabled by default after repeated
  tests showed mixed seek results. It uses piece size and qualified consumption
  estimates; the existing graded deadline ramp remains the default.
- **Scarce-piece scheduling experiment** is disabled by default. It advances a
  small amount of existing forward work using qualified, fresh evidence.

The native dependency is pinned to libtorrent v2.1.2 plus the reviewed queue-time
arithmetic fix. Delivery variation/outage feedback retains existing buffer
budgets and independent reader reconciliation. **Legacy
remains the default profile.** These changes cannot obtain bytes absent from all
accessible sources. Choose a smaller release explicitly when delivery stays below
consumption; different encodes cannot share pieces.

All of this works without a paid provider or account. See the [sparse streaming
guide](SPARSE_STREAMING.md) for settings, storage, privacy, endpoints and verification
limits, and the [implementation ledger](SPARSE_IMPLEMENTATION.md) for evidence.

## Swarm profiles

Select a profile in **Settings → Flow → Swarm profile**. `connection_speed` is outgoing connection attempts per second; `torrent_connect_boost` is the number of peers tried when a torrent is added. These are tuning choices, not measured speed rankings.

| Profile | What it does |
| --- | --- |
| **Legacy (default)** | Keeps TorrServer-LT's existing streaming-oriented libtorrent tuning, including 250 connection attempts/second, a 100-peer connect boost, and shorter peer/piece timeouts. Use it as the compatibility baseline. |
| **Conservative** | Removes those inherited swarm overrides and uses libtorrent defaults for them. Use it to compare with less modified swarm behavior. |
| **Balanced** | Starts from Conservative, then sets 50 connection attempts/second and a 50-peer connect boost. A moderate connection ramp to evaluate. |
| **Aggressive streaming** | Starts from Conservative, then sets 100 connection attempts/second and an 80-peer connect boost. It ramps up faster than Balanced, but less than Legacy on these two settings; it is not proven faster overall. |
| **Custom** | Starts from Conservative and applies only positive values entered for connection speed, connect boost, peer connect timeout, piece timeout, request queue time, and minimum reconnect time. `0` leaves that field at libtorrent's default. Use it for controlled experiments. |

All profiles retain your cache, proxy, upload, and active-torrent settings.
Development builds after Preview 2 `.5` also preserve healthy peers while a
lazy/warm torrent has no requested pieces; Conservative uses libtorrent defaults
for swarm tuning with these shared streaming lifecycle controls. Startup preload
sets piece priorities without pausing and disconnecting the swarm. Saving a
profile restarts the torrent engine and stops active streams. Keep **Legacy**
unless testing shows another profile works better for your network and torrents.

## Startup diagnostics in development builds

Current development builds return explicit preload requests when the buffer
is ready; an owned worker keeps the existing eight-second warm handoff in the
background. Same-file requests share that worker, and an episode switch or
removal cancels and joins the old owner. Reader windows continue to own playback
priorities. These changes are not yet in published Preview 2 `.5`.

**Settings → Flow → Retain diagnostic history** is opt-in and takes effect when
settings are applied (which restarts the engine). It saves `flow-history.jsonl`
in the server's state directory, with three rotating archives: at most 4 MiB
and a 256-event queue. Producers never wait for disk; dropped events and write
errors appear in Flow network diagnostics. Records contain UTC timestamps,
process-local torrent IDs, file indices, stage timings and numeric disconnect
codes. URLs, passkeys, credentials, media names/hashes and client addresses are
excluded. Existing console/file/access logging is separate and unchanged; raw
access logs can contain private URLs.

The diagnostics drawer explains metadata, peer discovery, first data,
header/index, prebuffer and bounded probe-grace waits. Metadata and first-DHT-peer
times start at torrent registration; other startup times start at preload.
Unobserved new timings show a dash. First media data is sampled from the existing
cache every 200 ms; a cached start may measure zero. These are server observations,
not the phone's decoded first frame or proof of Internet availability.

**Reuse DHT routing state** defaults on when Flow and DHT are enabled. Native
libtorrent saves only useful DHT nodes and node IDs to `flow-dht.bin`, every five
minutes and on graceful stop. State is bounded to 1 MiB, validated, and replaced
atomically; corrupt state is ignored without blocking local startup. Empty
routing snapshots do not erase previous useful nodes. Read-only mode prevents
writes; disabling DHT prevents loading and saving it. The binary routing file
contains network addresses and should not be included in shared diagnostic logs.
Persisted nodes are discovery hints. A bounded set (up to 32 per address family)
is reintroduced through libtorrent's public API after UDP listener readiness,
including listener changes; this handles late sockets without replacing native
discovery or network-change recovery.
Public-swarm and phone performance comparisons remain acceptance work.

## Build and project status

### Modern web interface

The modern interface was merged through [PR #2](https://github.com/gnbk21/TorrServer-LT-Flow/pull/2)
into **`develop`**, the default branch. The current preview packages the modern
build plus Preview 2 improvements. PR #3 tracks those changes into `develop`;
branch artifacts identify their exact development commit.
Legacy sources remain in `web-legacy/` until real device acceptance.

It provides a playback dashboard with bounded buffer/throughput history, a
torrent library and file browser, search with explicit preparation before
playback, and settings with Apply/discard and restart warnings. **Connect phone**
offers a selectable LAN address and QR code; playback links include Just Player,
VLC and ordinary HTTP open/copy fallbacks. Seven interface languages are retained.
Ordinary LAN HTTP works without installing a PWA or service worker.

The historical Preview 1 Windows build is available in
[CI run 36631696935](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/36631696935)
as `TorrServer-LT-windows-amd64`. It passed an isolated Windows UI/API smoke check;
real phone and long playback acceptance remain pending.

Frontend development requires Node.js 22.12 or newer and Yarn 1.22:

```powershell
cd web
yarn install --frozen-lockfile
yarn dev
```

Vite proxies server requests to `http://127.0.0.1:8090`; set `FLOW_DEV_SERVER` to
use another isolated server. Run `yarn typecheck`, `yarn lint`, `yarn test` and
`yarn test:e2e` for verification. Build with `yarn build`, then run
`go run gen_web.go` from the repository root before building the native server.
See [web acceptance record](WEB_AUDIT.md) and
[implementation ledger](MODERNIZATION_CHECKLIST.md) for verified behavior and
remaining device/release gates. Legacy sources remain in `web-legacy/`.

The [build workflow](.github/workflows/build.yml) compiles the web UI, server, libtorrent, and Windows dependencies, then uploads platform artifacts. A local Go-only build is insufficient; see [build instructions](build/README.md) for the CGo/C++ toolchain. Other platform targets remain in CI, but Windows x64 and Android playback are the primary Flow development path.

This fork retains the upstream API and settings format where possible. Upstream
install scripts, the root `release.json`, and upstream release links still refer
to **TorrServer-LT**. Use this fork's release assets or CI artifacts. Preview 2
generates its own channel-aware release manifest, checksums, original dependency
notices and GitHub build attestations. The managed scripts verify SHA-256 and
the exact executable version; `-RequireAttestation` also requires GitHub CLI
provenance verification. Executables remain unsigned by Authenticode. See
[distribution and recovery](DISTRIBUTION.md) and [Preview 2 notes](PREVIEW_2_NOTES.md).
The [draft stable integration PR](https://github.com/gnbk21/TorrServer-LT-Flow/pull/1)
tracks promotion from `develop` to `master` after the remaining acceptance.

Use torrents you are authorized to access. TorrServer-Flow retains the upstream [GPL-3.0 license](LICENSE) and acknowledges the work of [TorrServer-LT](https://github.com/trinity-aml/TorrServer-LT) and [TorrServer](https://github.com/YouROK/TorrServer).
