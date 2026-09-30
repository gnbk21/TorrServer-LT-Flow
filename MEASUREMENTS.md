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
rejects maintenance. Live release download/attestation results are recorded
below. Authenticode requires the maintainer's signing identity.

## Final Preview 2 candidate: 1 October 2026

Tag `MatriX.145.Flow-v0.2.0-preview.5` identifies source commit
`e0307c6dc151ab88be7449c13080a42d344b0913`. The eight native targets and both
frontend builds passed in [tagged run 36780598722](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/36780598722).
Linux unit/vet/race tests, four bounded fuzz campaigns and reachable Go
vulnerability checks passed. The scan still reports one uncalled module
advisory; it does not certify the native C++ dependency stack.

Frontend verification passed 41 unit/component cases and a separate test of
actual Go DTOs (42 distinct cases), plus 31 browser scenarios. The corrected
build's entry JavaScript is 168.57 KB gzip, its lazy dashboard 3.94 KB gzip,
and its lazy HLS chunk 178.98 KB gzip. The minified-size warning remains visible.
A new regression accepts legacy `CacheSize: 0` when reading settings, so the
page can open for repair; Apply still requires a positive budget.

All three generated-media playback scenarios passed against the exact tagged
Linux binary, SHA-256
`cb8fb6a0208be85b1a741729ee9267bcef240018d5f0da83979c2b13d5072e20`.
The fast case completed 256 cross-file churn requests and four additional
resource cycles; slow and disconnect cases each completed four cycles. Warm
expiry was observed after 25.30 seconds. Native polling measured 33 requests /
126,239 response bytes over 12 seconds, with request p95 51.16 ms. Full status
was 69,619 bytes including 282 traces; compact status was 6,183 bytes. These are
generated-fixture observations, not decoded-frame or representative endurance
measurements. Authenticated maintenance passed exclusive ownership,
active-playback rejection, restore recovery and startup doctor checks.

The same tagged run passed restricted Windows service installation, restart and
clean shutdown. Both console and service update fixtures passed authenticated
update, corrupted-download and maintenance-owner rejection, preserved state and
listeners, protected backups, startup-failure rollback, restored configuration
and private quarantine of failed migration files. Physical boot/sleep/network
acceptance remains separate from these disposable CI lifecycle checks.

The prerelease published at `2026-09-30T21:58:31Z`. All 32 assets were downloaded
and verified: 31 checksum entries, 13 full binary identities, eight sorted ZIPs
with fixed timestamps/modes, manifest URLs/sizes/digests and original native,
Go, frontend and static compiler-runtime notices. CI packaged the same inputs
twice and obtained identical checksum inventories. This verifies reproducible
packaging, not arbitrary-toolchain native compilation reproducibility.

Windows standard executable SHA-256:
`15868e7b4d5c189ff1de1f117b149d499763862980e11c1e5e73c008de3d3179`.
Windows ZIP SHA-256:
`b5735a1aceea3109aa067c88c64a910770cbe713aca9dea737f4a1cb9bad7e30`.
GitHub CLI verified both downloaded attestations; source digest, tag reference,
workflow identity and artifact subject digests matched. This is build provenance,
not Authenticode publisher signing.

The actual packaged installation and update scripts passed live GitHub
release selection/download and required provenance under PowerShell 5.1 and 7.
Tests used separate fresh loopback state on ports 53342 and 59022, authenticated
management, the previously stress-tested `2a12912c` native baseline, and the
published `.5` candidate. Both preserved authentication, listeners and private
configuration, created protected recovery copies, returned exact-version healthy
servers and stopped their owned processes cleanly. The updater fixtures above
separately establish rejection and rollback behavior.

The same downloaded Windows package served its actual embedded UI on isolated
loopback port 56248. The in-app browser verified Dashboard, empty Torrents,
unconfigured Search/Add, General settings with the backend's legacy cache zero,
Flow settings and Advanced maintenance controls. No API responses were mocked,
no settings were applied, and no browser runtime errors were recorded. Clean
shutdown completed. This smoke establishes native page integration; it does not
replace real Android layout, Just Player launch or media acceptance.

The full tagged pipeline completed successfully, including actual `--version`
runtime/linkage checks for Linux amd64, arm64 and arm/v7 containers before push.
Anonymous registry reads verified all three platform configurations, exact source
and version labels, and identical version/`preview` index digest:
`sha256:d1490c9c148c822134aa247f6f2cec24c715faedd0a5956ac5b4e30368c1fdea`.
This prerelease publishes version and `preview` tags, not `latest`. Source branch
cross/native and macOS runs `36780597119` and `36780596874` also passed. PGO remains
off, stable approval remains false, and the three external hardware/media gates
in `RELEASE_ACCEPTANCE.json` are unchanged.
