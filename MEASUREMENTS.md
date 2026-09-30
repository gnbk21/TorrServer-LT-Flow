# Preview 2 measurements and validation

This record separates generated-fixture measurements from real-phone acceptance.
Measurements are observations of this workload, not claims that Flow is faster
than upstream for every torrent or that all defects have been eliminated.

## Large library

Chromium, 1,000 generated torrent entries, the same development host and route:

| Implementation | Rendered cards | DOM nodes | First usable list |
| --- | ---: | ---: | ---: |
| Original full rendering | 1,000 | 21,107 | 3,550 ms |
| Pagination | 50 | 1,161 | 846 ms |

These are individual runs, not statistical latency guarantees. Search and sorting
operate on the full library, including entries outside the current page. Pages
contain at most 50 cards. Libraries over 200 entries refresh every five seconds;
per-session playback telemetry retains its one-second interval. Hidden tabs stop
polling. `web/e2e/library.spec.ts` verifies search, page navigation, empty results
and the large-library request budget.

The production Vite build measured 168.58 KB gzip for the entry JavaScript,
about 16 KB gzip of shared preloads and 3.95 KB gzip for the lazy dashboard.
The HLS decoder is a separate 178.98 KB gzip lazy chunk; it is not an initial
preload. Vite reports decimal KB. Non-English language chunks load on selection. These are build sizes,
not measured download times. Vite's warning about chunks over 500 KiB minified
remains visible; the initial compressed JavaScript fits the specification's
preferred 150–200 KiB budget.

## Polling and SSE decision

The browser regression recorded 33 requests and 9,582 bytes of synthetic JSON
responses over 11 visible seconds: 12 Flow, 12 library and three each for runtime,
tray and network. Six hidden seconds produced no new status requests. This
establishes scheduling behavior; synthetic payload size is not a real swarm
measurement. `build/playback_harness.py` additionally records the response bytes
and request p95 of the same cadence against a real native server and generated
media in `polling_measurement`.

Native Linux run `36750140703` measured 33 requests / 126,290 response bytes
over 12 seconds, with request p95 51.7 ms. A full diagnostic response was 69,570
bytes including 282 Range trace records; compact status was 6,120 bytes. Normal
UI polling requests compact status; the diagnostics drawer fetches histories
only while open, every three visible seconds. Existing API callers still receive
the full response unless they explicitly request `?traces=false`. The earlier
full-history polling run used 887,133 bytes at the same cadence. These runs show
payload reduction, not a controlled CPU or latency comparison.

Retain bounded polling for Preview 2. The measured library bottleneck was card
rendering and repeated whole-library work; pagination and a slower large-library
cadence address it. No measured evidence currently justifies adding a persistent
SSE endpoint, credential/reconnect handling and another server subscription
lifecycle. Existing HTTP polling remains the compatibility path. Revisit SSE
using concurrent real clients if payload volume or request CPU becomes material.

## Controlled playback

`build/fixtures.py` creates original synthetic video and sine audio with FFmpeg:
MP4 with its index at the head, MP4 with its index at the tail, MKV and a larger
variable-bitrate MP4. It verifies streams, duration, MP4 layout and SHA-256.
No copyrighted sample library or external torrent is required.

On 30 September 2026, the production-only frontend dependency audit reported
zero known advisories across 78 dependencies. Native CI also runs reachable Go
vulnerability checks. These dated scans do not guarantee the absence of unknown
vulnerabilities or replace runtime and compatibility verification.

`build/controlled_peer.py` supplies an actual local BitTorrent peer and tracker.
The harness tests unthrottled, 2 MiB/s with block delay, and a one-time disconnect
at 4 MiB/s. Range responses are compared byte for byte to the generated source,
including overlapping requests, cancellation, backward/forward seek, reconnect
and observed warm-cache expiry. The fast case switches files 256 times to exceed
the media cache and exercise eviction/refetch after native completion.

Failed stress runs exposed premature EOF, stale completion/eviction races and
completed-torrent state that did not re-enter download mode. These scenarios
remain required gates; compiling the fix alone does not mark them passed.
Reports identify the exact executable digest and retain failure observations.

All three generated-media scenarios passed in Linux run `36750140703`, including
256 fast cross-file requests, overlap/cancellation/seek/reconnect, four additional
resource cycles per case and warm expiry after 25.3 seconds. A subsequent
extended profiling stress run exposed another race: stale incomplete pieces
were discarded while late blocks were acknowledged without storing their data.
The resulting hash mismatch banned the valid seed. Native serialized partial
pruning and normal retention of late blocks address that discard path. The
Windows `e351e7af` build then passed 256 initial switches plus 115 additional
switching cycles during CPU/trace collection, with no hash failures or peer bans.
Linux runs `36753657916` and `36754783800` still failed subsequent churn: complete
eviction and delayed completion alerts also needed native ownership checks.
Complete eviction is now serialized after native flush; stale markers do not
reset an in-flight download and delayed alerts cannot complete buffers with
holes. Exact metadata length covers the short final piece. Actual storage
failures are reported without fabricating corrupt hashes or acknowledging lost
writes. Linux run `36758891545` passed all three scenarios after these changes,
including 256 fast switches and four resource cycles per case. Windows
`2a12912c` passed 104 additional profiling cycles, four resource cycles and warm
expiry after 25.6 seconds. Linux profiling run `36758885562` passed 256 initial
switches plus 135 additional cycles without hash failures or peer bans.

The native 1,000-entry library upsert benchmark in run `36747207599` measured
319.6 ms / 11.65 MB / 81,099 allocations per whole-library rewrite versus
0.308 ms / 10.94 KB / 69 allocations per single-key update. Three iterations of
each implementation ran on the same Linux runner. This narrow persistence
benchmark is separate from browser rendering and does not establish general
throughput guarantees.

## Profiling and PGO

Profiling is opt-in on a separate numeric loopback address. It is never installed
on the public API listener. CPU profiles and traces are limited to 30 seconds per
request, and the profiling listener shuts down with its owned server.

```powershell
& .\TorrServer-LT-windows-amd64.exe --path .\profile-state --port 8091 --ip 127.0.0.1 --profile-address 127.0.0.1:8092
Invoke-WebRequest 'http://127.0.0.1:8092/debug/pprof/profile?seconds=30' -OutFile cpu.pprof
Invoke-WebRequest 'http://127.0.0.1:8092/debug/pprof/trace?seconds=5' -OutFile trace.out
go tool pprof -symbolize=none -top cpu.pprof
go tool trace trace.out
```

Profiles/traces can contain private runtime details. Keep them local unless
reviewed; the redacted support report does not include them. The earlier Windows
30-second CPU sample attributed about 97.1% of samples to `runtime.cgocall` through the
blocking alert wait. That does not establish that optimizing Go code will improve
playback or identify the C++ work underneath. Use native symbols and an OS CPU
profiler to investigate libtorrent/OpenSSL, separately from Go heap and scheduler
analysis. RSS includes native allocations; RSS minus Go heap is not an exact
native-memory measurement.

Use the runtime-recorded Go names with `-symbolize=none` for stripped release
binaries. Local ELF re-symbolization produced incorrect Go function labels in
the initial CI text summaries; the original profile still contained correct
runtime names and is what the PGO compiler consumed. Native C++ frames require
matching unstripped symbols and native profiling, separately from this Go view.

The manual build workflow's `evaluate_pgo` option trains on generated native
playback, records CPU/trace data, compiles with `TS_PGO_PROFILE`, and compares three
interleaved baseline and candidate runs. Normal builds explicitly use `-pgo=off`.
Generated-fixture results alone cannot authorize a shipped PGO profile; adoption
requires a representative real-media workload and repeatable improvement.

Linux run `36758885562` completed three interleaved baseline/PGO pairs at
4 MiB/s with 1 ms block delay; all six playback checks passed. Median-of-run
median TTFB was 1.225 ms without PGO versus 1.125 ms with PGO. Per-run p95 ranged
from 676–2,156 ms without PGO and 953–1,847 ms with PGO. Those variable tail
results and tiny median difference do not establish a repeatable viewing
benefit. The report records both binary digests, `synthetic_only: true` and
`adopted: false`; Preview 2 retains PGO off.

## Endurance and hardware gates

```powershell
python build/playback_harness.py --executable .\TorrServer-LT-windows-amd64.exe --output .\endurance-results --duration 7200
```

The duration is additional cycling **per scenario**, so the default three cases
take more than six hours. Use `--cases fast` for one two-hour controlled run.
Reports sample RSS, Go heap, goroutines, handle availability, resident cache and
active/warm allocations. Short CI cycles are regressions, not multi-hour proof.
Native cache capacity can expand for multiple protected reader windows; it is not
a hard cap on process RSS. Handle counts are reported only when the host supports
them.

Real Android/Just Player checks still require the user's phone and media:
launch through Lampa, startup, pause/resume, forward/backward seek, episode switch,
Wi-Fi loss/reconnect and background/resume. HTTP delivery time and server stalls
do not measure decoded frame time or player rebuffering. Retain the original
baseline and run the same file/network conditions for a controlled A/B claim.

The disposable Windows CI gate checks a restricted service account/SID, explicit
state path, authentication, readiness, restart and clean stop. Physical boot,
sleep/resume and network recovery require a host check: record `/flow/network`,
`/flow/tray` and startup diagnostics before/after each event, verify the listener
returns and launch the same legal media again. Do not change the live server or
reboot a user's host merely to satisfy a CI gate.

## Update regression boundaries

`build/test-windows-update.ps1` runs production scripts with actual native
executables, authenticated loopback HTTP, file replacement, protected recovery
backups and startup-failure rollback. Only GitHub release transport is replaced
by a controlled manifest/download fixture. It also checks corrupted downloads,
another maintenance owner, credential preservation and listener flags.
The rollback fixture also corrupts the owned configuration database after the
pre-update backup, then injects an actual candidate startup failure. The previous
server returns healthy with its private setting restored from the protected copy.
Disposable Windows run `36753656101` passed the service update variant as well:
restricted service identity, authenticated update, listener preservation, private
backups, restored configuration and quarantined candidate migration files. It
does not establish physical boot or sleep/resume behavior on the user's PC.
`build/maintenance_harness.py` separately verifies that actual active playback
rejects maintenance. Live release download/attestation verification remains a
distribution gate. Authenticode requires the maintainer's signing identity.
