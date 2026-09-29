# TorrServer-Flow Preview 1

First public **prerelease** of this fork, including the modern web interface.
This is a development snapshot for testing; Flow 1.0 and Modern Web 1.0 acceptance
remain incomplete.

## Included

- Adaptive startup and read-ahead, warm mobile sessions, seek/Range diagnostics
  and the five documented swarm profiles.
- Modern dashboard, torrent library/file browser, add/search workflow and
  settings with explicit Apply/discard and restart warnings.
- LAN address selection and phone QR pairing, Just Player/VLC launch links,
  ordinary HTTP playback/copy fallbacks and seven interface languages.
- Windows service support and a separate Windows tray companion.
- Native server binaries for Windows x64, Linux x64/arm64/armv7, Android
  arm64/armv7 and macOS Intel/Apple Silicon. Optional GStreamer variants are
  included on the platforms supported by the build scripts.

## Download and run

On Windows, download **TorrServer-Flow-windows-amd64-preview.1.zip**, extract it,
then run the standard executable in PowerShell:

```powershell
.\TorrServer-LT-windows-amd64.exe --port 8090 --path .\flow-data
```

Open <http://127.0.0.1:8090> on the PC. On the same private LAN, use
`http://<PC-LAN-IP>:8090` on your phone or in Lampa, then launch Just Player.
**Connect phone** in the web interface can show the LAN address and QR code.
The standalone Windows executable is also available as a release asset.

Close an older instance before using its port. Back up its state before migrating;
keep concurrently running instances on different ports and data directories.
Changing the executable's folder does not automatically migrate your old library
or settings. Use Ctrl+F5 if the browser retains an older interface.

Use the standard executable for direct playback. The `-gst` variant requires
GStreamer installed separately at runtime and is supplied for Windows x64,
Linux x64/arm64 and both macOS targets. Installing ffprobe is optional; without
it, Flow uses its bootstrap and observed bitrate estimates. Unix downloads need executable
permission (`chmod +x TorrServer-LT-<platform>`). Android assets are native server
binaries, not an APK or a replacement for Just Player. The ZIPs include setup,
license and build-provenance documents; the Windows ZIP also includes FlowTray.ps1.

Use SHA256SUMS to verify downloaded assets. Windows PowerShell example:

```powershell
Get-FileHash .\TorrServer-Flow-windows-amd64-preview.1.zip -Algorithm SHA256
```

Keep the server on a trusted private LAN or VPN. It listens on all interfaces by
default. Inherited known-torrent playback routes are not fully protected by
management authentication; do not expose it directly to the Internet. Only use
torrents you are authorized to access.

## Build provenance and verification

The binaries are the unchanged successful CI artifacts from source commit
**847f6b4f37be32d49e5c9be1bdc30751d1836607**, which is the release tag's target.
They deliberately retain the exact development build identity:

`MatriX.145.Flow-dev-847f6b4f37be32d49e5c9be1bdc30751d1836607`

Preview 1 is a distribution label; the binaries were not restamped or rebuilt.
BUILDINFO.json records source/build links and individual binary hashes.

- [Cross-platform CI](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/36631696935)
  and [macOS CI](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/36631661693)
  passed for all eight targets.
- Type checking, lint, 40 frontend unit/component tests and 27 browser checks
  passed. Browser checks use controlled API fixtures.
- Native Linux unit tests, vet and race checks passed. Source govulncheck found
  no reachable vulnerability and one uncalled module advisory.
- The Windows executable passed an isolated test of the actual embedded UI and
  backend: startup, API/settings, mobile layouts, pairing, pause/resume and shutdown.
  No torrent or real phone was used in that smoke test.

## Known limits and source

Real Android/Just Player/Lampa validation of this modern build, large/weak-swarm
playback, network/background recovery and multi-hour resource stability remain
pending. No controlled evidence establishes that Flow is faster than upstream.
Optional source-provider and DNS-over-HTTPS roadmap work is unfinished.
The binaries are unsigned. Checksums verify file integrity, not publisher identity.
IPv6 discovery was not validated on this Windows host.

See the current [core audit](https://github.com/gnbk21/TorrServer-LT-Flow/blob/feature/modern-web/AUDIT.md),
[web audit](https://github.com/gnbk21/TorrServer-LT-Flow/blob/feature/modern-web/WEB_AUDIT.md)
and [device checklist](https://github.com/gnbk21/TorrServer-LT-Flow/blob/feature/modern-web/WEB_DEVICE_CHECKLIST.md).
Stable master/develop promotion and legacy source removal are still held.
No Docker image is published by this preview.

GPL-3.0 source is available through the release's Source code archives and
[the exact build commit](https://github.com/gnbk21/TorrServer-LT-Flow/tree/847f6b4f37be32d49e5c9be1bdc30751d1836607).
Build scripts pin the native dependency sources; frontend dependency notices
are included and embedded in the UI. Thanks to
[TorrServer-LT](https://github.com/trinity-aml/TorrServer-LT) and
[TorrServer](https://github.com/YouROK/TorrServer).
