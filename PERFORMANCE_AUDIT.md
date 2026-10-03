# Startup delay investigation and next improvements

Investigation date: 3 October 2026. Scope: the reported roughly 30-second wait
before media starts fetching; retain the existing engine, cache and API.

## Available evidence

No TorrServer process was running when inspected. No saved personal playback
log was found in the workspace, the known executable/state directories or the
default service location. The available captures are development/CI tests;
they cannot establish what happened during the user's last viewing session.
Console output is not automatically retained after a normal executable exits.

The old `.tools/artifacts/windows/TorrServer-LT-windows-amd64.exe` reports
`MatriX.145.LT-1.1.9`, without a Flow development identity. Verify `--version`
when comparing executables. The console development baseline is source
`034d4bf5`, not the published Preview 2 `.5` package.

## Confirmed implementation defects

1. Preload paused/resumed an otherwise healthy torrent to kick its picker.
   Libtorrent pause disconnects peers. On the controlled default-profile run,
   the log records `torrent paused` as the peer-disconnect reason and the
   buffer took 10.05 seconds to become ready. There was one connection before
   preload and two afterward. Piece priorities and deadlines can request the
   buffer without tearing down those connections.
2. Non-Legacy profiles restored `close_redundant_connections` to its default.
   Between metadata and playback, Flow deliberately sets piece priorities to
   zero. Libtorrent then considers the torrent finished and can disconnect
   seeders as an upload-to-upload connection. A controlled Custom run with
   a 30-second minimum reconnect time took 30.2 seconds to complete bootstrap.
   Keeping healthy peers is a shared lazy/warm streaming lifecycle requirement,
   alongside the existing active-torrent queue overrides. Peer caps and expiry
   still apply; download demand remains restricted to requested pieces.

The fixes remove the preload pause/resume and retain the existing
`close_redundant_connections=false` setting across every swarm profile. They
do not lower connection timeouts or add a second cache.

The `4eed4944` explicit preload endpoint also retains an eight-second handoff/prefetch
grace after the buffer is ready. Lampa normally polls the torrent state and can
launch playback during that grace. Endpoint response duration and playable
buffer readiness must therefore be measured separately. This grace does not
explain a wait before the first media blocks download.

## Reproducible verification

`build/startup_harness.py` runs a real executable against generated media and a
loopback tracker/seeder with a peer discovered before preload. It creates fresh
state, records version/digest, buffer-ready and endpoint-response timings,
connection counts and startup metrics, and stops its owned process. The fixed
build must retain the original connection and complete the small controlled
buffer in under five seconds. CI runs both Legacy and Custom (30-second
reconnect) cases alongside existing playback, cancellation, warm-session and
resource regressions. These are server measurements, not Android first-frame
measurements or a promise for arbitrary public swarms.

Windows measurements on identical generated media (one local healthy seeder,
TCP only, fresh state; no bandwidth or block-delay limit):

| Profile / reconnect policy | Before, source `034d4bf5` | Final build, source `4eed4944` | Connections before → after preload |
| --- | ---: | ---: | --- |
| Legacy / inherited 10-second minimum | 10,047 ms | 223 ms | Before fix: 1 → 2; after fix: 1 → 1 |
| Custom / explicit 30-second minimum | 29,411 ms | 221 ms | Before fix: 1 → 2; after fix: 1 → 1 |

Times measure buffer readiness from the preload request. The after-fix endpoint
response took about 8.2 seconds because it also retained the handoff grace.
The before Custom case dropped its peer as an upload-to-upload connection; the
before Legacy case recorded a pause disconnect. Neither disconnect occurs
during corrected startup. The controlled seeder closes its socket normally
when the test ends; that later EOF is not a startup failure.

Windows reports are retained locally under `.tools/startup-before-legacy`,
`.tools/startup-before-custom-final`, and `.tools/startup-db4f12dd/{legacy-check,custom-check}`.
An intermediate native build `db4f12dd` measured 285/259 ms. The final delivered
build reports are under `.tools/startup-4eed4944/{legacy-check,custom-check}`;
both passed the same five-second limit and retained the original connection.
The first Custom exploration under `.tools/startup-before-2` also enabled uTP,
so its 30.2-second result is not the matched TCP-only comparison above.
The native artifact's archive digest matched GitHub's published
`7f045d202f98233d5b5471c24a386e3d00a4cde4fd280f03fff4f597223972f4`.
Its standard executable SHA-256 is
`162a66a80860a5658682cc01636a570236c828f46dada9a0927755feab4e6db9`.
GitHub built the synthetic PR merge `d6dc4cc7`; its source tree
`711fb0d0d7ac3703e90082c3beb5c0122af59004` exactly matches `db4f12dd`.
The version string therefore correctly contains that merge identity.

[Native run 37136293890](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37136293890)
passed all six cross targets, native unit/vet/race/security/fuzz gates, both
startup regressions, controlled playback/resources, maintenance, console and
Windows service/update/rollback. The matching macOS run `37136293905` and
portable checks `37136293875` passed. The subsequent `4eed4944` revision changes
only profile help in all seven languages: Aggressive uses default timeouts,
not shorter ones. This is a development fix; published Preview 2 `.5` remains
unchanged. There is no captured phone session validating the reported incident.

The final code/help revision `4eed4944` also passed native build and regression
run [37136950582](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37136950582),
macOS `37136950601` and portable `37136950586`. Its PR merge identity is
`1cf66f83423a64b46100b0bfc869f2e1d0b44a6a`; the remote merge tree
`175c2f92f9c5deb6b63a4434aeaa06e654544ffb` matches the reviewed branch tree.
The delivered standard executable is
`.tools/startup-4eed4944/windows/TorrServer-LT-windows-amd64.exe`, SHA-256
`ce4074850e7bfe7989b109998cd46bd3cc105190df6c3da49864dbe92bfc964b`.
Its archive matched GitHub's artifact digest
`5c9093946f4b583710e4ad9839eb86f86ce0a0373528701e8b34a28ff18da55c`.
Use your usual state directory and port when launching it. Verification used
separate disposable directories and stopped every owned process; personal
playback state and the published release were not modified.

## Follow-up implementation

Development source `6d5c9129` implements the first four items below: opt-in
bounded history, native DHT routing-state reuse, translated startup explanations,
and an early explicit-preload response with an owned background handoff. See
[NEXT_IMPROVEMENTS.md](NEXT_IMPROVEMENTS.md) for implementation and verification.
The published `.5` release predates these changes. Public-swarm and real phone
comparisons remain separate acceptance evidence.

## Improvement priorities and remaining validation

| Improvement | Expected benefit | Validation needed |
| --- | --- | --- |
| Opt-in bounded diagnostic recorder | Retain redacted stage timings, first useful block, peer-disconnect reasons and startup outcome across restarts; make the next historical investigation possible | Fixed disk/queue limits, rotation, passkey/URL/token redaction, no blocking of playback on slow disks |
| DHT routing-state persistence | Reuse known nodes after restarting instead of rebuilding discovery from bootstrap alone | Use libtorrent's native session-parameter serialization for DHT state only; bounded/corrupt-file handling, atomic writes, cold/warm magnet comparisons and network-change recovery |
| Startup stage explanation | Show whether the wait is for metadata, useful peers, head/index pieces or a probe grace; distinguish a weak swarm from local work | Timestamp each stage and compare against captured Lampa/Just Player requests; no assumption about a fixed Range sequence |
| Early explicit-preload response with background handoff | Clients that wait for the preload response could launch earlier than the measured eight-second response grace | Keep bounded prefetch/reservation ownership until the reader takes over; cancellation, concurrent preload and shutdown tests; compare actual client behavior before changing the API timing |
| Request cancellation and duplicate-fetch coalescing | Stop abandoned remote `.torrent` requests and avoid downloading/parsing the same source for concurrent requests | Request contexts, bounded lifetime/cache, preserve per-request auth/URL identity and existing API errors; benchmark remote-link retries first |
| Risk-based buffer sizing | React to throughput variance and repeated waits before a stall, within the existing cache | Representative VBR/REMUX and weak-swarm tests; hysteresis, memory limits and independent clients; compare with the existing rolling controller |
| Targeted DNS/transport diagnostics | Identify DNS failure, tracker rejection and peer transport failures separately | Controlled DNS/UDP/TLS faults. Add application DoH only if system DNS is the measured failure; retain canonical hostnames and HTTPS identity |

DHT-state persistence uses an existing libtorrent facility:
[session-state tutorial](https://www.libtorrent.org/tutorial-ref.html).
Priorities/deadlines should continue to use its
[time-critical streaming picker](https://www.libtorrent.org/streaming.html).
The [settings reference](https://www.libtorrent.org/reference-Settings.html)
explains reconnect backoff. A bigger cache, more worker threads or a more
aggressive profile does not fix a forced peer disconnect.

Provider fallback remains conditional on an actual provider/account selection
and sustained source failure. PGO remains off until representative measurements
show a benefit. HTTP/3, a replacement torrent engine, transcoding and a second
media buffer are not justified by the observed startup problem.

## Capturing a future session

In the new development build, enable **Retain diagnostic history** before the
session to retain allowlisted startup events across process restarts. The
`flow-history.jsonl` files in the state directory have a combined 4 MiB limit;
records exclude URLs, names/hashes and credentials. Queue overflow and disk
errors are visible in network diagnostics. This setting is off by default and
cannot reconstruct an earlier session. Do not share the separate `flow-dht.bin`
routing file, which contains network addresses.

For a detailed short capture, use the executable's existing file logging with
the same state directory and
port you normally use, for example adding `--logpath .\flow.log`. This selects
UTC file logging instead of the console panel. For detailed short captures,
enable Flow Range tracing and Debug Flow in settings before reproducing the
issue, then disable them afterward. Settings changes restart the engine, so
apply them before starting playback. Access logs are optional and can contain
private URLs; they are not needed for an initial startup-stage capture.

Live diagnostics are runtime observations and disappear when the torrent
expires or the process exits. A saved support report during the problem is
useful, but cannot reconstruct an already-ended session unless retained history
was enabled beforehand. Retained stage events are not complete HTTP traces.
