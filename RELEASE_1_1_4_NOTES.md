# TorrServer-Flow 1.1.4

Version: `MatriX.145.Flow-v1.1.4`.

An update to Flow 1.1 with more reliable seek/metadata recovery, clearer playback
evidence and a more responsive web interface. Original media bytes, existing
clients, cache architecture and selected network/security preferences are retained.

## Changes since 1.1.3

- **Startup and seeking:** retain useful startup peers, schedule head/index phases
  without resetting their deadlines, and let continuing useful Adaptive startup
  make bounded progress. Obsolete range readers cannot renew playback deadlines
  or clear a newer seek's recovery state.
- **Automatic magnet recovery:** reuse saved verified metainfo, then try bounded
  supplied HTTPS sources. Optional free public caches require explicit consent.
  Slow sources can trigger a staggered second request; bounded service cooldowns
  and an explicit retry avoid serial timeout tails and repeated failed requests.
  Hash, DNS, redirect, interface, cancellation and concurrency protections remain
  enforced. Metadata recovery does not create video suppliers in an empty swarm.
- **More useful diagnostics:** distinguish native error categories, transport and
  address family; show recent integrity failures separately from redundant bytes.
  Readable/verified server buffers share an offset and sample timestamp. Seven
  languages explain the likely wait cause, evidence freshness and relevant action.
  Raw details remain available; exports omit raw signed URLs and peer identities.
- **Responsive web interface:** lazy settings panels retain visited drafts;
  navigation gives immediate feedback. Improved narrow-screen and enlarged-text
  headers, torrent cards, action buttons and dialogs. Live announcements follow
  meaningful state changes instead of continuously reading numeric telemetry.
- **Safer live policy changes:** urgent-horizon and container-burst flags can apply
  without restarting the torrent engine, with existing task cleanup/reset behavior.
  Native capacity and transport changes still require restart.
- **Build and upstream fixes:** Go 1.26.10 fixes the certificate-verification
  advisory found by the release gates. Retain libtorrent 2.1.2; retire the obsolete
  standalone builder in favor of the supported native recipes. Adapt MatriX.147's
  natural episode ordering and Torznab product/version identity fixes.
- **Optional player evidence:** earlier session correlation, monotonic event order
  and whole-session summaries in bounded manual reports. Stock Just Player works
  as before; it gains no instrumentation from a server update. The experimental
  companion APK is distributed separately from stable server assets.

## Defaults and upgrading

**Legacy remains the default.** Existing experiments and PGO remain opt-in/off;
profile comparisons are mixed and do not establish a universally fastest profile.
Saved cache/profile/encryption/interface settings are not automatically changed.
No paid provider, transcoding or full-episode download is required for direct
original-quality streaming. Actual delivery still depends on reachable suppliers
and sustained network throughput.

For Windows, download
`TorrServer-Flow-windows-amd64-MatriX.145.Flow-v1.1.4.zip`, extract it into a new
folder, finish playback and stop the old server. Keep the old executable and a
backup of its state for rollback. Launch the new standard executable with the
**same state path, port, authentication and HTTPS options** as before. For a fresh
installation:

```powershell
.\TorrServer-LT-windows-amd64.exe --path .\flow-data --port 8090
```

Open `http://127.0.0.1:8090` on the PC; Lampa/Just Player uses the PC's LAN address.
Allow private-network access if Windows prompts for the new executable location.
Do not run two versions against the same state directory. To roll back, close
the new version and restore the preserved executable/state if needed.
Managed installs can use the included authenticated idle-only updater with
checksum/provenance verification. See
[distribution and rollback](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/DISTRIBUTION.md).

## Verification and known limits

The implementation passed native unit/race/fuzz/vulnerability and platform gates,
Windows service/update/rollback, web production budgets, 115 frontend tests (one
existing skip), 104 browser checks and 122 generated-media streaming trials.
The straggler comparisons include 200 targets proven to need fresh peer data;
speed results were mixed, so they do not justify changing defaults. Exact tagged
builds, identity, inventories, reproducible packages and provenance gate this
publication. The original unpublished tag was repaired after CI package downloads
timed out. Essential checks were repeated on the corrected tag. Duplicate full
streaming comparisons and CPU capture reuse the accepted successful run
`39761266311`; a fail-closed source comparison verifies the runtime, web, native
patches, dependencies and build/distribution inputs are unchanged. Downloaded distribution results are recorded in
[1.1.4 verification](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/RELEASE_1_1_4_VERIFICATION.md).

This owner-authorized stable release discloses remaining field checks: broader
Android/TV/player and public-swarm playback, physical Windows boot/sleep/network
recovery and spoken screen-reader acceptance. Earlier two-hour RAM/disk runs
passed on an identified earlier binary; additional two-hour tests were skipped
at the owner's request. Exact-final endurance and the interrupted eight-hour run
are not passed. Controlled HTTP checks do not measure phone decoding or prove
ISP/VPN behavior. Native profiling does not establish complete C++ attribution;
the Windows interface guard does not guarantee protection of shared OS DNS.

Exact release-source CodeQL scans passed on master and develop. The same 30
reviewed findings remain open; a passing scan is not a zero-alert claim. No new dismissals were made. See [security scope](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/OCTOBER_SECURITY_RECONCILIATION.md).
Windows assets are not Authenticode signed. The optional player's independent
native decoder rebuild/redistribution audit remains unfinished; its APK is not
included in stable assets. MatriX.147 automatic sidecar subtitle conversion was
reviewed separately and is not included. Flow retains GPL-3.0 and upstream notices.

[Full changes since 1.1.3](https://github.com/gnbk21/TorrServer-LT-Flow/compare/MatriX.145.Flow-v1.1.3...MatriX.145.Flow-v1.1.4).
