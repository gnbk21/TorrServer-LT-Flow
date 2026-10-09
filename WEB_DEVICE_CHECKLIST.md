# Modern interface device acceptance

Use the published Flow release identified in [STATUS.md](STATUS.md), recording
`/echo` and its SHA-256.
Development verification builds are identified separately in STATUS.md. Keep
the previous executable for rollback. Do not run two servers against one data directory.
The checks below are not marked passed by desktop emulation or fixture tests.

## Start

1. Close the previous server when its playback is finished, or use a different
   port and fresh directory for testing:

   ```powershell
   .\TorrServer-LT-windows-amd64.exe --port 8092 --path .\flow-modern-test
   ```

2. Open `http://127.0.0.1:8092` on the PC. Select **Connect phone**, check the LAN
   address, and open it on the phone over the same home network. Use that base
   address in Lampa. Allow private-network firewall access if needed.
3. Use media you are authorized to access. Start with one episode, then a large
   4K file and a torrent with slower peers. Record the build version, browser,
   player, cache budget and swarm profile with each result.

## Record results

| Check | Expected behavior | Result / evidence |
| --- | --- | --- |
| Phone dashboard and navigation | No horizontal page overflow; buttons usable in portrait and landscape | |
| Pairing | Correct LAN address/port; readable QR; manual choice and copy work over HTTP | |
| Browser compatibility | Test Chrome and Brave; Samsung Internet if available | |
| Just Player installed | File link opens the selected one-based file; audio and seeking work | |
| Just Player absent | HTTP Open/Copy remains usable; browser fallback behavior is recorded | |
| Lampa playback | Same server address works for episode launch; dashboard discovers the active session | |
| Search and preparation | Registration, metadata and prebuffer are distinct; player links appear after preparation | |
| Pause/background/resume | Warm state appears when applicable; reconnect resumes; zero download alone is not labelled stalled | |
| Seek forward/backward | Correct file continues; buffer and seek diagnostics update; no obsolete position overwrites the seek | |
| Network loss/recovery | UI shows failure/degraded state, retains useful data and recovers without request storms | |
| Settings | Cancel/discard makes no server change; Apply warns of restart, including an active stream | |
| Optional integrations | Unavailable TMDB/GST/search provider does not break library or direct playback | |
| Internal player | Test a browser-supported file; test GST/HLS only with a GST executable and working runtime | |
| Long session | Several hours of playback/UI; record server RSS, configured cache and browser memory at intervals | |
| Authentication | Management remains protected under configured HTTP auth/WAF; QR contains no credentials | |

Record failures with the exact action, file/container, browser/player version,
timestamp and a screenshot. Share logs only after removing private URLs and
credentials. Server delivery position and buffer estimates are not the player's
decoded presentation position or measured time to first frame.

The owner approved stable Latest publication after automated gates with these
unfinished field checks disclosed. Publication does not mark them passed.
Legacy removal remains held pending device acceptance. Current evidence and
limits are in [Flow 1.0 verification](RELEASE_1_0_VERIFICATION.md),
[STATUS.md](STATUS.md) and `RELEASE_ACCEPTANCE.json`.

## Adaptive acceptance

The current stable release includes these adaptive changes; historical tests
and exact development identities remain in
[ADAPTIVE_VERIFICATION.md](ADAPTIVE_VERIFICATION.md). Record `/echo`, SHA-256,
saved/effective settings and the test timestamp. Use a disposable state directory for failure
injection and keep your normal playback server separate.

| Check | Expected behavior | Result / evidence |
| --- | --- | --- |
| Cache Apply | Choose 2048 MiB, Apply, reload Settings and restart the server; saved/effective stay consistent | |
| Two settings tabs | Save from one tab; an older edited tab gets a conflict and retains its draft | |
| Idle restart | Queue a transport change while playing; effective settings stay unchanged until all active work ends; Cancel removes the intent | |
| Demand qualification | Full cache, intentional download pause, probe and preparation show their own modes; their zero traffic is not a measured supply outage | |
| Weak swarm | Fresh useful supply and media demand qualify risk; a sustained deficit suggests preparation or a smaller release | |
| Concurrent viewers/preparation | Independent seeks remain correct; preparation yields, then resumes without discarding verified data | |
| Capability expiry | Anonymous Range/HEAD works for the selected file; tampering, expiry and process restart reject later requests; renew before reopening | |
| Resource pressure | Record RSS, Go heap, handles, cache and available RAM separately; optional work yields and protected overcommit is explicit | |
| Service crash | In disposable service testing, record bounded SCM recovery and restored settings; slow swarm traffic alone never restarts the process | |

### Physical interface and recovery capture

Source-interface binding is not yet a verified VPN kill switch. Perform this
optional test only with controlled endpoints and media you can share. Do not
disable a physical adapter during unrelated remote work or important playback.

1. Record adapter names, IPv4/IPv6 addresses, routing and DNS configuration;
   select the intended torrent adapter explicitly. Capture both that adapter
   and the physical uplink with Wireshark or an equivalent OS packet capture.
2. Test peer TCP, peer UDP/uTP, DHT, UDP trackers, HTTP/HTTPS trackers and native
   web seeds separately against controlled endpoints. Include Go torrent URL
   imports and tracker-list mirrors. Keep canonical tracker hostnames intact.
3. Record source addresses and DNS requests for each path; test IPv4 and IPv6
   independently when enabled. Absence of traffic in a short capture is not
   proof that a protocol is protected; first confirm that path actually worked.
4. Disconnect the selected adapter, retaining the physical uplink. Check
   `WAIT_INTERFACE`, cancellation of pending torrent fetches and no automatic
   unbound torrent retry. Play already verified cached bytes locally.
5. Restore the adapter with the same address, then with a changed address;
   also test Windows sleep/resume. Check evidence invalidation, component
   recovery, bounded announce attempts and no private-torrent DHT activity.
6. Retain the capture and a per-protocol result with timestamps. System DNS
   is outside source binding; discovery, other applications, search, artwork
   and updater traffic must be identified separately. Any physical-uplink
   torrent/DNS escape blocks a stronger leak-protection claim and requires
   an explicit OS enforcement design before retesting.

Physical boot/sleep/network recovery, an eight-hour resource run, native C++
OS profiling and real-phone playback remain unperformed until their results
are recorded. Do not mark them passed from the generated loopback harness.
