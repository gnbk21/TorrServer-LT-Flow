# TorrServer-Flow 1.1

Version: `MatriX.145.Flow-v1.1.2`.

## Changes since 1.0.1

- Updated dashboard and torrent library: playback focus, incident timeline,
  card/list views, restored navigation state, paginated full-library search,
  compact live projections and less background polling.
- Consistent light/dark/system themes, density preferences, responsive controls,
  keyboard focus, settings search, draft/conflict protection and seven languages.
- Streaming resilience: bounded interruption/burst reserves, recovery hysteresis,
  fair concurrent readers, useful slow-peer retention and duplicate accounting.
  The capacity experiment reconciles late replies after seek cancellation and
  gives urgent cold pieces credit before normal ahead-of-playback refill.
- Opt-in next-episode head/index warmup, limited to 32 MiB in existing spare
  cache; it yields to current playback and cancels on changes or shutdown.
- Synchronized startup, discovery, storage, HTTP and recovery diagnostics;
  redacted export/import of optional player telemetry. Server waits are explicitly
  separated from measurements of player-visible buffering.
- Native v2/hybrid identities, SHA-256/Merkle verification, verified resume and
  bounded identical-byte reuse. Private torrents are excluded from shared reuse.
- Corrected uncertain-poster classification when client and request deadlines race.
- Improved fetch cancellation, response cleanup, launcher synchronization,
  credential isolation, proxy destination validation and log privacy.
- Validated cache-removal identifiers and cleanup directories, safe download
  types for non-media torrent files, bounded certificate/artwork lookup reads,
  and selected-interface routing for artwork and indexer requests.
- Optional reversible Windows interface guard, with isolated TCP/UDP IPv4/IPv6,
  LAN-preservation and uninstall tests. It is never installed automatically.
- Hardened release tooling: immutable Action pins, explicit token permissions,
  bounded inventory reads, complete CycloneDX dependency graphs and original
  dependency notices. Log entries escape injected line/terminal controls.

## Defaults and upgrading

**Legacy remains the default.** Adaptive Streaming, scheduling experiments,
next-episode warmup and PGO are not enabled automatically. Controlled comparison
results were mixed; no profile is claimed best for every swarm or network.
Streaming preserves the original video quality and needs no paid provider or
full-episode download. The existing piece store remains the only media cache.

On Windows, download and extract
`TorrServer-Flow-windows-amd64-MatriX.145.Flow-v1.1.2.zip`, stop the old server,
keep its executable and state for rollback, then run:

```powershell
.\TorrServer-LT-windows-amd64.exe --path .\flow-data --port 8090
```

Open `http://127.0.0.1:8090` on the PC. Lampa/Just Player uses the PC's LAN address.
Keep existing authentication, HTTPS, port and state-directory choices.
See [distribution and rollback](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/DISTRIBUTION.md).

## Verification scope

Publication requires the exact tagged native, platform, browser, security,
service/update, controlled streaming and reproducible-package gates.
[The verification record](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/RELEASE_1_1_VERIFICATION.md)
records actual results and any unfinished checks.

This owner-authorized stable release discloses unfinished physical Android/TV,
public-swarm and Windows boot/sleep/network/VPN acceptance. Long-run resource
acceptance is unfinished: the owned eight-hour attempt was interrupted after
about 4 hours and 965 cycles and is not a pass. The Windows
guard does not establish shared OS DNS leak protection. Native profiling has
partial call-stack attribution, and controlled workloads do not predict every
phone, peer or network. Preserved owner-controlled source inputs and reviewed
scanner findings are explained in [the security review](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/RELEASE_1_1_SECURITY_REVIEW.md).

The experimental Media3 companion APK is excluded from stable assets. Its
decoder sources/notices have traced provenance, but independent native rebuild
and redistribution compatibility are not established. Windows assets are not
Authenticode signed. Flow retains GPL-3.0 and its upstream attribution.
