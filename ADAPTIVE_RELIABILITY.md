# Adaptive reliability and access controls

These changes are on the development branch in [PR #3](https://github.com/gnbk21/TorrServer-LT-Flow/pull/3).
Published Preview 2 `.5` predates them. They keep libtorrent, the existing piece
cache, HTTP Range behavior, free episode preparation and explicit swarm profiles.

## Useful delivery and buffering risk

The diagnostics distinguish **wire traffic** from newly verified pieces useful
to a reader's current forward window. Duplicate completion notifications,
unrelated downloads, probes and preparation do not inflate useful delivery.
Seeks and network recovery invalidate previous supply observations.

| Mode | Meaning |
| --- | --- |
| `DEMAND` | An external reader has missing forward data. |
| `FULL` | Its current forward window is contiguous and readable. |
| `IDLE` | There is no qualifying forward reader demand, or server downloads are intentionally paused. |
| `PROBE` | Only inspection work is relevant to the group. |
| `PREPARATION` | Background episode preparation is running without qualifying foreground demand. |
| `RECONNECT` | Local addresses or the selected torrent interface are unavailable. |

Rates use completed, wholly qualified one-second intervals: a short five-second
view and up to 59 seconds of history. At least three recent qualified intervals
are needed for medium confidence; five recent and ten total give high confidence.
Full, idle, mixed and stale intervals are excluded. Active demand with zero useful
delivery remains an outage sample. The numbers describe observed supply during
demand, not guaranteed swarm capacity.

Buffer risk combines contiguous forward bytes, credible media demand or stable
HTTP observations, conservative short/long useful delivery, variation, recent
piece waits and fresh known supplier counts. Missing evidence gives `UNKNOWN`.
Deficits, low buffer and long waits raise an explainable score; elevated/high
risk can raise the target to 90/120 seconds, bounded by configured limits and
existing cache windows. Smoothing prevents repeated window oscillation. Native
graded deadlines remain the default; the rate-aware/scarce experiments remain
opt-in. Persistent deficits suggest **Prepare episode** or a smaller release.

HTTP byte position and approximate buffer seconds are not the player's decoder
clock. A connected supplier is not a promise of future throughput. Unknown data
must not be interpreted as zero peers, healthy playback or a measured stall.

## Shared resource budget

Settings → Flow provides the following controls (MiB means 1,048,576 bytes):

| Setting | Default | Behavior |
| --- | --- | --- |
| `GlobalCacheBudgetMB` | `0` | Automatic RAM eviction budget: one quarter of system RAM, at most 4 GiB, at least 64 MiB. |
| `WarmCacheBudgetMB` | `512` | Warm/idle share, capped at one quarter of the global budget. `0` releases warm reservations. |
| `PreparationConcurrency` | `1` | Maximum concurrently scheduled background jobs, from 1 to 16. |
| `PreparationQuotaMB` | `4096` | Explicit disk reservation limit for prepared episodes. |

Active RAM caches share the foreground allocation; idle caches share the warm
allocation. Immediate reader, bootstrap and container reservations retain
priority. Their aggregate can exceed a soft budget: diagnostics explicitly show
protected overcommit and stop optional work. This is **not a process RSS hard
limit**. Native allocations, Go overhead, disk buffers and mandatory reservations
still consume memory. Disk-backed cache bytes are reported separately.

Available-system-memory pressure halves the budget, releases warm reservations
and limits background work. Process RSS, Go heap/system memory, handles where
supported, cache bytes and system memory are separate observations. macOS does
not currently supply available-system-memory pressure evidence.

Preparation yields to active streaming and memory/overcommit pressure. Scheduling
reasons include `PLAYBACK_PRIORITY`, `MEMORY_PRESSURE`, `QUOTA`, `DISK_SPACE` and
`QUEUED`. Jobs and verified bytes survive suspension. The disk guard retains at
least 64 MiB or two pieces of free-space margin. Remaining-byte forecasts sum
per-file progress and can conservatively count shared boundary pieces twice;
they do not reserve free space against other applications.

## Settings and recovery

Settings shows **draft**, **saved** and **effective** cache values, unsaved edits,
the configuration revision, executable and data directory. Polling preserves
dirty edits. A save from an outdated tab is rejected with HTTP 409; refresh or
discard edits before retrying. Older API clients may omit the revision for
compatibility. Read requests never reset settings.

Cache/read-ahead, buffer targets, runtime diagnostics and access policies apply
without restarting the engine. Native transport, interface, disk ownership,
swarm profile and integration changes require a restart. The confirmation offers
immediate Apply or **Apply when idle**. A pending change survives process restart;
its base revision is checked before applying. New playback is fenced briefly
while an idle restart is actually running. Pending changes can be cancelled.

The local `flow-settings-good.json` holds validated last-known-good settings.
Unreadable or unsupported settings recover in memory without overwriting the
source. Explicit Apply retains corrupt JSON as `flow-corrupt-settings-*.json`
or rejected settings as `flow-rejected-settings-*.json` before repair. Recovery
status identifies the source and any problem. A failed durable write does not
publish the new settings. `flow-settings-pending.json` stores queued intent.

These local files can contain integration secrets and host policy; keep the
data directory private. Portable backup remains redacted and retains the
destination host's credentials, interface and access policy during restore.

New Windows service installations use SCM crash recovery after 15, 30 and 60
seconds, then stop retrying until the 24-hour failure counter resets. Existing
service installations are not silently reconfigured. Slow torrents never trigger
a restart watchdog. A failed engine reconfiguration attempts the previous
settings and reports any rollback failure.

## Optional management and playback security

Compatibility defaults preserve existing Lampa/Just Player operation:

| Setting | Default | Behavior |
| --- | --- | --- |
| `SecurityProfile` | `compatible` | `restricted` checks browser origins on management endpoints. |
| `ManagementOrigins` | empty | Comma-separated exact HTTP(S) origins; the direct dashboard origin is always allowed. Supplying a list also enables checking in compatible mode. |
| `ManagementRateLimit` | `0` | Optional requests/minute per direct client IP; `0` is unlimited. |
| `RequirePlaybackToken` | `false` | Require authentication or a scoped capability for legacy playback routes. Requires HTTP authentication to enable. |
| `PlaybackTokenTTL` | `3600` | Capability lifetime in seconds, from 30 to 86,400. |

Origin checks supplement authentication; originless native clients still follow
normal authentication. A reverse proxy needs an explicitly allowed public origin
when it changes scheme/host. Rate identity uses the direct peer IP, not forwarded
headers: clients behind one proxy share its allowance. An overly low allowance
can throttle dashboard polling. HTTP 429 includes `Retry-After`.

**Files / Play** can create or renew an expiring link. It grants GET/HEAD/Range
access to one existing file, preserving the playback group; it cannot import,
list, manage or mint links. Tokens expire and are revoked by process restart.
An already open response can finish after expiry; subsequent Range requests need
a valid link. Choose a lifetime covering the episode/movie or renew before
reopening. Links are bearer secrets: anyone holding one can play that file.
Without HTTP authentication, compatible-mode link creation has the same LAN
access boundary as legacy playback. There is no per-link revocation list.

Requiring capabilities can break clients that depend on unauthenticated legacy
URLs or playlists. Enable it only after confirming the client's authenticated
requests or scoped-link workflow. Ordinary HTTP does not encrypt links. The
server remains a trusted-LAN application; access controls are not Internet
deployment approval. Request logging omits bodies/query strings and redacts
capability paths. Internal probe authorization uses a private process secret
and loopback; `stat=ffprobe` alone grants no access.

## Optional torrent interface binding

Settings → Network accepts an exact OS adapter name in `TorrentInterface`.
An empty name keeps OS routing. `RequireTorrentInterface` validates that a name
has been explicitly selected. Any selected adapter waits when unavailable;
there is no unbound torrent fallback. Native listeners and outgoing interfaces
use its eligible addresses. Disappearance pauses native networking, disables
discovery/transports and cancels Go metadata/tracker-list fetches. Recovery
reconciles the existing session. Selected binding disables UPnP/NAT-PMP/LSD and
cannot be combined with the existing command-line proxy.

Windows interface/route/address notifications are debounced, with polling as a
fallback and bounded jittered retry. Same-address recovery and sleep/delayed
checks invalidate observations and request recovery. Announce attempts are
globally spaced by at least 30 seconds; private torrents never force DHT
announces. A local address proves local readiness, while tracker replies provide
separate evidence of external connectivity. No engine restart is needed for
ordinary network changes.

**This is source-interface binding, not a verified VPN kill switch.** OS DNS
resolution is not bound by this feature. A source address alone is not an OS
firewall guarantee. Other applications and ordinary artwork/search/update
requests are outside this torrent policy. Exposure diagnostics deliberately
report DNS and leak-protection verification as false. Physical disconnect tests
covering TCP, UDP/uTP, DHT, HTTP/HTTPS/UDP trackers, DNS and mirrors are required
before making stronger claims. No router, firewall or VPN configuration is
changed automatically. Cached local playback is kept available.

## Measurement and acceptance

Upload diagnostics show utilization against an explicitly configured upload
cap. They do not infer WAN queue latency from LAN requests. If upload saturation
coincides with latency, compare a lower cap under the same workload. Router SQM
can be evaluated separately; Flow neither changes the router nor claims SQM is
installed or effective on this network.

For disposable loopback tests with a native build:

```powershell
python build/adaptive_harness.py --executable .\TorrServer-LT-windows-amd64.exe --output .\adaptive-results
python build/playback_harness.py --executable .\TorrServer-LT-windows-amd64.exe --fixtures .\fixtures --output .\profile-results --cases fast --duration 120 --profile
```

Generate fixtures with `python build/fixtures.py --help` and its documented
output argument, or omit `--fixtures` and let the harness generate them using
FFmpeg. Output directories must be new. The harness starts/stops only
its owned server and generated swarm; it does not reuse personal settings.
Profiles include Go CPU, heap/allocations, mutex/block waits, goroutines and a
short execution trace. Native C++ CPU requires an OS profiler (for example WPR/WPA
or Linux perf); Go pprof is not a complete native profile. Profiling is opt-in
and loopback-only. PGO remains off until representative comparisons justify it.

For an overnight resource run, use `--cases fast --duration 28800 --profile`
with a new output directory. Review hash failures, process/Go memory, handles,
goroutines and request latency over time. This is generated HTTP traffic, not
eight hours of decoded Android playback. Follow [MEASUREMENTS.md](MEASUREMENTS.md)
and [RELEASE_ACCEPTANCE.json](RELEASE_ACCEPTANCE.json) for phone/media acceptance:
startup, forward/backward seeks, pauses/lock, reconnect near warm timeout,
multiple readers, weak swarms, and Windows boot/sleep/network recovery.

See [the implementation checklist](ADAPTIVE_RELIABILITY_CHECKLIST.md) for exact
verification evidence and the unperformed environmental gates. No measured
speedup, stable release acceptance or VPN leak guarantee follows from compilation.
