# TorrServer-Flow

TorrServer-Flow is a fork of [TorrServer-LT](https://github.com/trinity-aml/TorrServer-LT) focused on streaming torrents from a Windows PC to an Android player over a local network. Its primary use case is direct playback in Just Player, including playback launched through Lampa. It keeps TorrServer-LT's libtorrent engine, web interface, and HTTP API.

> **Development status:** Flow is on `develop`; `master` is reserved for stable Flow releases. There is no tagged Flow release yet. The [Windows CI build](https://github.com/gnbk21/TorrServer-LT-Flow/actions/workflows/build.yml) is available as a development artifact. A successful build and a short phone playback check do not establish a performance advantage; controlled comparisons with TorrServer-LT are still pending.

## Run on Windows

1. Open a successful [`build` workflow run for `develop`](https://github.com/gnbk21/TorrServer-LT-Flow/actions/workflows/build.yml?query=branch%3Adevelop) and download the **`TorrServer-LT-windows-amd64`** artifact. Extract the ZIP.
2. In PowerShell, run the standard executable from the extracted folder, using a separate data directory for Flow:

   ```powershell
   .\TorrServer-LT-windows-amd64.exe --port 8090 --path .\flow-data
   ```

3. On the PC, open <http://127.0.0.1:8090>. On a phone on the same local network, use `http://<PC-LAN-IP>:8090` as the TorrServer address in your client (for example, Lampa), then play with Just Player. Allow the server through Windows Firewall on private networks if prompted.

Keep Flow and another TorrServer instance on **different ports and data directories**. Do not start them concurrently against the same `config.db`. The server listens on all interfaces by default. Keep it on a trusted private LAN or VPN. HTTP authentication (`--httpauth`, with an `accs.db` account file) protects management endpoints, but inherited playback routes can serve an existing torrent without credentials when its hash is known. Do not expose it directly to the Internet.

The artifact also contains a `-gst` executable for optional GStreamer HLS transcoding. Direct playback uses the standard executable and does not require GStreamer. The `-gst` variant needs GStreamer installed separately at runtime.

## What Flow adds

- **Adaptive startup:** schedules a bounded head buffer and the container index/tail; an optional `ffprobe` result can refine the startup target without blocking playback indefinitely.
- **Adaptive read-ahead:** sizes the forward cache window from media bitrate and observed playback, within the configured cache budget. It can be disabled independently of other Flow features.
- **Warm mobile sessions:** briefly protects data around the last playhead after a player disconnects, so a quick reconnect can resume from cached pieces.
- **Seek and Range diagnostics:** tracks startup, buffering, piece waits, seek recovery, and HTTP Range timing. A torrent card opens live diagnostics; detailed Range traces are opt-in.
- **Windows background operation:** includes a service mode and a separate tray companion. See [Flow development notes](FLOW.md) for service and tray commands.

Flow settings are in the web interface's **Flow** tab and under `BitTorr.Flow` in `settings.json`. Flow is enabled by default. Saving settings restarts the torrent engine and interrupts active streams. `GET /flow/status/<torrent-hash>` exposes per-torrent diagnostics; `/flow/network` reports network readiness. Both follow the server's HTTP authentication setting. See [FLOW.md](FLOW.md) for defaults, endpoint behavior, and the comparison procedure.

## Swarm profiles

Select a profile in **Settings → Flow → Swarm profile**. `connection_speed` is outgoing connection attempts per second; `torrent_connect_boost` is the number of peers tried when a torrent is added. These are tuning choices, not measured speed rankings.

| Profile | What it does |
| --- | --- |
| **Legacy (default)** | Keeps TorrServer-LT's existing streaming-oriented libtorrent tuning, including 250 connection attempts/second, a 100-peer connect boost, and shorter peer/piece timeouts. Use it as the compatibility baseline. |
| **Conservative** | Removes those inherited swarm overrides and uses libtorrent defaults for them. Use it to compare with less modified swarm behavior. |
| **Balanced** | Starts from Conservative, then sets 50 connection attempts/second and a 50-peer connect boost. A moderate connection ramp to evaluate. |
| **Aggressive streaming** | Starts from Conservative, then sets 100 connection attempts/second and an 80-peer connect boost. It ramps up faster than Balanced, but less than Legacy on these two settings; it is not proven faster overall. |
| **Custom** | Starts from Conservative and applies only positive values entered for connection speed, connect boost, peer connect timeout, piece timeout, request queue time, and minimum reconnect time. `0` leaves that field at libtorrent's default. Use it for controlled experiments. |

All profiles retain your cache, proxy, upload, and active-torrent settings. Saving a profile restarts the torrent engine and stops active streams. Keep **Legacy** unless testing shows another profile works better for your network and torrents.

## Build and project status

### Modern web interface

The modern interface is developed on **`feature/modern-web`**; `develop` retains
the audited legacy interface until real device acceptance. Choose that branch
in the build workflow to download an executable containing the modern UI.

It provides a playback dashboard with bounded buffer/throughput history, a
torrent library and file browser, search with explicit preparation before
playback, and settings with Apply/discard and restart warnings. **Connect phone**
offers a selectable LAN address and QR code; playback links include Just Player,
VLC and ordinary HTTP open/copy fallbacks. Seven interface languages are retained.
Ordinary LAN HTTP works without installing a PWA or service worker.

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

This fork retains the upstream API and settings format where possible. Upstream install scripts, `release.json`, and upstream release links still refer to **TorrServer-LT**, not a Flow release. Use this fork's CI artifact until a Flow release is published. The [draft integration PR](https://github.com/gnbk21/TorrServer-LT-Flow/pull/1) tracks promotion from `develop` to `master` after the remaining validation.

Use torrents you are authorized to access. TorrServer-Flow retains the upstream [GPL-3.0 license](LICENSE) and acknowledges the work of [TorrServer-LT](https://github.com/trinity-aml/TorrServer-LT) and [TorrServer](https://github.com/YouROK/TorrServer).
