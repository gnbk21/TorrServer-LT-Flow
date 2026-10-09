# TorrServer-Flow 1.0

The first full Flow release includes the modern interface, adaptive streaming,
Windows service/tray, diagnostic and recovery tools, plus the useful changes
from TorrServer-LT `MatriX.146.LT-1.2.1` and `MatriX.146.LT-1.2.2`.

## New LT integrations

- Safely published engine handle across reconnects and synchronized session access.
- Lazy 1 MiB HTTP transport buffering, separate logical consumption tracking,
  cancellation and Range/seek compatibility; the native piece cache remains the
  sole media cache. Fewer source reads are not a promise of faster public swarms.
- Live certificate reload, managed self-signed renewal, authenticated certificate
  status/public download/upload/path selection and explicit identity changes.
  Settings revision checks, immutable uploaded pairs and restricted private keys
  protect existing configuration and user-owned certificates.
- HTTPS-only and HTTP-media modes, plaintext redirection on the HTTPS port,
  bounded handshake handling and graceful listener shutdown. External player,
  playlist, phone pairing, DLNA and Bonjour addresses respect the selected mode;
  browser playback keeps its secure origin. Internal media access uses a private
  loopback listener and process secret, excluded from public responses/logs.
- Optional GStreamer: one-frame cue-boundary tolerance, including fractional
  rates; a bounded cold-source head/tail warmup with one retry; cancellation and
  cleanup for shared discovery processes. The standard binary needs no GStreamer.
- Modern Security certificate controls in all seven supported interface languages,
  draft preservation and navigation protection, and updated API documentation.

Flow retains its newer Go 1.26.9, libtorrent 2.1.2 and dependency pins. Existing
tracker/session and OS/VPN resolver fixes are retained; upstream runtime
downgrades and blocking startup DNS are not adopted. Invalid explicit HTTPS
configuration fails closed rather than silently falling back to HTTP.

## Defaults and installation

**Legacy remains the default swarm profile.** Adaptive Streaming and the three
high-bitrate scheduling experiments remain opt-in; no profile is proven best
for every network. Streaming retains the source video quality and does not
require full-episode downloading or a paid provider.

For Windows, extract the platform ZIP and run:

```powershell
.\TorrServer-LT-windows-amd64.exe --path .\flow-data --port 8090
```

Open `http://127.0.0.1:8090` on the PC or use its LAN address in Lampa/Just Player.
Stop the old server before replacing its executable; preserve its state and
previous binary. Managed install/update scripts now select the stable channel.
HTTPS deployments with CLI overrides use the documented manual upgrade path.
See [distribution and rollback](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/DISTRIBUTION.md).

Assets include eight platform ZIPs, standard and optional GST binaries,
SHA-256 manifests, exact source identities, original dependency notices and
GitHub provenance attestations. Windows binaries are not Authenticode signed.
The stable container alias is `ghcr.io/gnbk21/torrserver-lt-flow:latest`.

## Verification and remaining field checks

The tagged publication workflow requires native tests, race/vulnerability checks,
web/browser tests, platform/service/update gates, controlled streaming scenarios,
and reproducible packaging. Exact results and downloaded package/container
verification are recorded in [the release verification record](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/RELEASE_1_0_VERIFICATION.md).

**Published as Latest with the owner's approval and these disclosed limits:**
broader physical Android/player coverage, representative public-swarm A/B
comparisons, multi-hour resource measurements, and physical Windows
boot/sleep/network recovery checks remain unfinished. The user reported two
smooth episodes on an earlier build; this does not prove those broader checks
or the final release's behavior on every phone/network. Automated server waits
are not measurements of player-visible stutters. Source vulnerability scanning
and stripped-binary module scanning have different precision; consult the
verification record for any retained advisory.

Upstream attribution and adaptations: [LT integration ledger](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/UPSTREAM_LT_INTEGRATION.md).
Flow and its retained upstream sources are GPL-3.0.
