# Modern interface device acceptance

Use the latest published Flow preview, recording `/echo` and its SHA-256.
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

Stable promotion and legacy removal remain gated on these results and the
remaining core Flow acceptance checks in [STATUS.md](STATUS.md) and
`RELEASE_ACCEPTANCE.json`.
