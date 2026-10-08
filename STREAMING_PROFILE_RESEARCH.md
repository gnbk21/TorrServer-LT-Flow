# Improving high bitrate streaming

Updated 8 October 2026. This evaluates the streaming implementation and defines
the candidate profile. **Adaptive Streaming is implemented as an experimental
development setting; verification is in progress.** Legacy remains the active
retry profile on the user's existing executable. No new policy was applied to
that process during development.

## Objective and decision

Preserve original video quality and stream through Lampa and Just Player with a
bounded rolling cache. Full episode preparation, a lower bitrate release and a
paid provider are not prerequisites or the proposed solution. Receiving and
temporarily caching upcoming pieces is still necessary for streaming; retaining
the entire episode is not.

The profile is **Adaptive Streaming**, experimental and opt-in.
Its objective is better contiguous delivery than Legacy under variable peer
performance, not a larger connection count or a faster first response at the
expense of later interruptions. There is currently no evidence supporting a
claim that it outperforms Legacy. The first isolated policy comparisons rejected
two simple substitutions as ready defaults.

## What the actual incident established

| Observation | Supported conclusion | Not established |
| --- | --- | --- |
| Literal quotes in the disk-cache path, access denied, upload-only mode, empty verified cache; removing quotes restored writes | A local storage configuration failure prevented preloading | ISP blocking |
| Saved and effective cache size both 2048 MiB | The intended cache size was active during the retry | Two GiB of contiguous playable data |
| About 90 Mbps estimated media demand; useful delivery sometimes 7.4–9.1 MB/s against 11.2 MB/s demand | The measured useful supply sometimes failed to sustain demand | Whether all loss of useful supply originates in remote capacity or local scheduling |
| Urgent pieces had connected suppliers; nearly full cache contained gaps and older retained pieces | Total traffic, peer count and cache fullness are insufficient playback-health metrics | That a connected supplier is currently willing and able to send promptly |
| Pausing the player built a buffer and briefly restored smooth playback; stuttering returned | A reserve can absorb a temporary delivery deficit | A permanent fix from pausing |
| User-approved Balanced → Legacy retry helped initially but stutters returned | Legacy gave partial symptom relief in this session | A controlled comparative speed ranking |

These are server and user observations, not decoded-frame measurements. See
[the incident record](ADAPTIVE_VERIFICATION.md#live-disk-cache-incident-and-path-validation-8-october-2026).
Media names, torrent hashes, private paths and peer addresses are excluded here.

## Scheduling interaction reproduced

Flow already enables sequential download and adds native deadlines across each
forward window. Its Go priority vector grades nearby pieces above distant ones,
but calls `PrioritizePieces` before `SetPieceDeadline`.

In the pinned libtorrent 2.1.2 source, assigning or refreshing a deadline raises
an incomplete piece to priority 7. Removing the deadline lowers it to priority 1;
it does not restore the application's earlier priority. A paused synthetic
native probe reproduced both behaviors for all 32 pieces and for an eight-piece
subset, and explicit priority restoration recovered the intended vector.
Source: [libtorrent's implementation](https://github.com/arvidn/libtorrent/blob/v2.1.2/src/torrent.cpp).

This explains why the application's graded vector is not necessarily the
effective native vector. It **does not prove** that priority promotion caused
the observed stuttering: native deadlines still have ordered due times, and
normal sequential picking still prefers earlier pieces. Setting every deadline
to zero would remove useful ordering and is not a solution.

Native streaming already selects requests using estimated peer queues and has
adaptive duplicate requests for overdue pieces. Maintaining enough queued work
is essential, especially when many individually slow suppliers contribute.
Blindly adding aggressive retries or discarding slow peers can waste bandwidth.
See [libtorrent's streaming algorithm](https://www.libtorrent.org/streaming.html).

### Isolated comparison and its limits

The repeatable screen uses upstream libtorrent 2.1.2, original deterministic
bytes, 16 loopback suppliers, eight slow/eight faster peers, an outage, 4 MiB
pieces, a 128 MiB moving window and approximately 90 Mbps paced byte demand.
It compares:

1. **Full:** the existing priority/deadline ordering and full ascending ramp.
2. **Bounded eight:** deadlines for eight pieces, with ordinary forward work
   remaining eligible throughout the window.
3. **Graded after:** full deadlines followed by explicit graded priority
   restoration on each scheduler tick.

The bootstrap waits for two complete pieces; the paced phase lasts eight
seconds. Reported blocked polls are counts, not elapsed stall seconds. The
receiver checks the delivered prefix against the original source bytes.
Results and executable/script provenance are in
[STREAMING_POLICY_EVIDENCE.json](STREAMING_POLICY_EVIDENCE.json).

Three rotated runs per policy produced these screening observations:

| Policy | Median bootstrap, seconds | Median blocked polls | Exact prefixes |
| --- | ---: | ---: | ---: |
| Full ramp | 8.297 | 0 | 3/3 |
| Bounded eight | 3.797 | 63 | 3/3 |
| Graded after | 2.562 | 208 | 3/3 |

The alternatives' shortest delivered prefixes were 88.0 MB versus about 89.8 MB
for the control. An earlier nine-run screen included a 75.4 MB graded-after
prefix; its separate results are retained in the evidence JSON. This variation
also cautions against selecting a policy from one run. Exact prefixes establish
byte correctness, not sustained demand. Faster bootstrap changes the reserve
available when the outage occurs; this comparison does not isolate scheduling
from that startup tradeoff.

This is a short native scheduling screen, **not Flow's custom cache, HTTP Range
serving, Lampa or Just Player**. The upstream wheel lacks Flow's private native
patches. Both the two-piece bootstrap and complete-piece receiver differ from
Flow's startup and responsive block reads. A loopback comparison cannot predict
WAN congestion, Wi-Fi behavior or an episode's decoded stutter rate.
The native control uses selected Legacy-like settings, not the complete Flow
Legacy profile; the table is not a comparison of selectable production profiles.

The exploratory runs showed that shorter startup can accompany more blocked
reads, and that a fixed smaller urgent set is not consistently preferable.
Neither policy is accepted for production. The original Flow one-piece-only
deadline suppression previously starved the pipeline; it must not return.

## Candidate Adaptive Streaming behavior

Use the existing cache, group playheads, native scheduler, startup controller and
settings lifecycle. Avoid a second downloader or cache. Preserve all other
profiles and the current default until comparative evidence supports a change.

| Area | Candidate behavior | Required safeguard |
| --- | --- | --- |
| Native base | Start with Legacy as the control configuration; vary one setting at a time in experiments | Respect user peer/bandwidth limits, transport, proxies, private-torrent policy and interface binding |
| Urgent work | Keep the blocked read and nearest missing pieces first; adapt the urgent horizon using credible demand, contiguous reserve and native request progress | Never reduce a persistent active stream to one urgent piece; maintain enough forward work for all useful suppliers |
| Ordinary pipeline | Retain graded, nonzero priorities for the full allowed forward window | Missing diagnostics retain the established ramp; no zero-priority starvation or unbounded fetching |
| Priority reconciliation | Account for native promotion and demotion explicitly when deadlines are added, refreshed or removed | Test actual native priorities, cancellation and outstanding requests; do not assume the last Go vector is still effective |
| Recovery | Expand useful queued work when queues are underfilled; focus on the first incomplete frontier when later pieces progress but the frontier does not | Hysteresis, bounded work, no repeated global cancellation and no arbitrary peer bans |
| Reserves | Extend existing risk-aware startup/read-ahead decisions using recent deficit duration and delivery variance | Bounded by the same memory/disk budget; show the startup/resilience tradeoff and retain prompt seek handling |
| Unknown evidence | Keep demand/supply/queue health unknown when idle, stale or unqualified | Do not confuse a full idle cache with zero capacity or treat historical supply as current |
| Multiple readers | Reconcile independent devices, seeks, header/tail reads, warm state and preparation reservations together | One client must not cancel another client's work or monopolize the shared budget |
| Controls | Explicit selection with translated explanation, restart planning and rollback to Legacy | No automatic changes between user-selected profiles; retain experimental deadline switches independently |

The native session's queue and timeout settings are shared. A per-file controller
must not overwrite them based on one stream and destabilize another. First test
per-torrent scheduling improvements; consider session tuning separately after
request diagnostics identify a repeatable constraint.

An urgent horizon must account for demand in bytes per second, piece size,
available queue capacity and completion-time variation. A fixed piece count
represents very different playback durations across torrents. No horizon or
timeout value is asserted optimal from this screen.

## Improvements in priority order

### 1. Explain the first missing piece at block level

Extend the existing bounded asynchronous sparse snapshot with aggregate urgent
block states: unrequested, requested, receiving, waiting for writes/hash work,
duplicate requests and effective native priority. Include freshness, truncation
and request-progress information when actually available. Reconcile this with
cache residency and contiguous verified frontier growth.

Use native asynchronous snapshots and copy their data while owned by the alert
or native context. Do not expose native pointers to Go or synchronously enumerate
queues from HTTP handlers. Keep sampling, copied bytes and retained history
bounded; omit peer identities from ordinary diagnostics. The native
[download-queue API](https://www.libtorrent.org/reference-Torrent_Handle.html#torrent_handle)
provides block state, progress and duplicate counts, but not every desired
request-age metric; additional age instrumentation must be explicit and tested.

This distinguishes an absent request, a stalled supplier, too much distant work,
and local write/hash delay. These need different remedies; a generic "more
peers" button cannot distinguish them.

### 2. Benchmark coherent scheduling policies

Use those diagnostics to compare the full ramp, bounded urgent horizons and
priority restoration in the actual Flow build. Expand the controlled harness
with long high bitrate cases, 4 MiB pieces and real cache pressure. Measure
contiguous frontier growth, HTTP read waits, startup/seek tails, useful delivery,
duplicate payload, CPU/RSS and cache/refetch behavior together.

Queue tuning must consider both estimated delivery time and outstanding byte
capacity. Larger queues can improve utilization but delay recovery; smaller
queues can idle fast suppliers. Preserve native adaptive overdue handling before
considering any narrowly scoped patch. Official
[settings documentation](https://www.libtorrent.org/reference-Settings.html)
distinguishes queue-time targets, block-count caps and different timeout types;
they are not interchangeable speed controls.

### 3. Improve reserve and variable-demand awareness

Extend existing qualified media demand and useful-supply estimates with recent
delivery deficits and burst uncertainty. An average media bitrate does not
describe every scene. HTTP client read-ahead is also not a decoded play clock.
When precise timestamp demand is unavailable, report the uncertainty and use
bounded safety margins rather than inventing an instantaneous bitrate.

Avoid a new full-file scan or a startup-blocking probe. Keep the current probe
reuse and startup limits. A modest temporary reserve can absorb outages while
streaming continues; it is different from preparing the whole file.

### 4. Separate torrent delivery from delivery to the phone

Correlate native useful progress, server HTTP waits and local write/hash time.
If data is already contiguous but the phone still buffers, investigate the LAN,
Wi-Fi contention, decoder/container behavior and client buffering separately.
Compare TCP/uTP using controlled evidence; keep both enabled by default.
IPv6/incoming reachability can expose additional suppliers when supported, but
neither creates seed capacity. Do not change router/firewall configuration
without a specific measured need and authorization.

For a demonstrably saturated uplink, evaluate upload headroom and router SQM.
[FQ-CoDel](https://datatracker.ietf.org/doc/html/rfc8290) and
[CAKE](https://www.bufferbloat.net/projects/codel/wiki/Cake/)
manage bottleneck queues. They are optional network measures, not embedded Flow
features or established fixes for this incident. No router changes were made.

### 5. Use compatible free sources when they exist

Keep existing DHT/PEX/tracker recovery and warm peer retention, respecting
private torrents and announce cooldowns. Repeated forced announces and huge
tracker lists can add load without supplying the missing bytes.

Existing native HTTP mirrors/web seeds can supplement peers with an authorized
source containing the torrent's **exact bytes**. They do not need a paid account,
but no matching free mirror is guaranteed. Preserve native verification and
source restrictions. [BEP 19](https://www.bittorrent.org/beps/bep_0019.html)
describes the URL seed mechanism.

A different swarm of the same desired quality is also an optional source
choice, not a mandatory downgrade. A matching episode/title is insufficient for
cross-torrent piece reuse. Do not merge incompatible encodes or silently switch
the player's stream mid-episode.

## Implementation and acceptance order

1. Add bounded urgent-block diagnostics to the existing native snapshot and
   server/frontend contracts; test stale, empty, truncated and deleted-handle
   states. Keep production scheduling unchanged for this evidence step.
2. Add explicit candidate selection only with its complete backend validation,
   migration/export, native settings, scheduler wiring, seven-language UI,
   README and rollback behavior. Preserve existing profile choices; a missing
   profile value retains the Legacy default.
3. Compare actual Flow binaries with rotated repeated runs. Cover healthy,
   many-slow-peer, intermittent, complementary-partial, late-HAVE, disappearing
   supplier, high latency, insufficient aggregate supply and corrupt-source
   cases; use multi-minute 90–120 Mbps demand and 512 MiB/2 GiB caches.
4. Require exact Range bytes, working startup/forward/backward/near-EOF seeks,
   warm resume, no leaked readers, protected container data and bounded resource
   use. Include concurrent devices and background preparation interactions.
5. Accept a "better than Legacy" claim only with consistently lower HTTP wait
   time/tails and fewer interruptions at the same demand and startup constraints,
   without a material healthy-swarm, seek, integrity or resource regression.
   Record repetitions and distributions; do not select the best isolated run.
6. Validate representative Lampa/Just Player playback and a long session before
   promoting the profile. A source-capacity deficit must remain honestly reported;
   scheduling cannot guarantee uninterrupted playback when sustainable useful
   delivery stays below consumption after the reserve is exhausted.

The existing rate-aware/scarce experiments remain off: prior results were mixed.
An engine replacement, generic QUIC migration, machine-learning controller,
blanket short timeouts, unrestricted duplicate requests, arbitrary peer eviction
and an even larger cache are not justified by the present evidence. Upstream
[discussion 6272](https://github.com/arvidn/libtorrent/discussions/6272) describes
historical pipeline tradeoffs; it is not proof of a remaining 2.1.2 defect.

## Reproduce the native probes

These optional development tools use the official CPython-compatible libtorrent
2.1.2 binding, not a production dependency. The Windows investigation used the
official CPython 3.12 wheel; its release digest and provenance are recorded in
the evidence JSON. Do not add the wheel to the repository or global application
environment. Use a compatible Python and an isolated module directory.

```text
python build/native_priority_probe.py --binding-path <binding-directory> --output .tools/streaming-research/priority.json
python build/native_stream_screen.py --binding-path <binding-directory> --output .tools/streaming-research/screen.json --rotations 3
```

The priority probe disables incoming/outgoing peer transports. The screen uses
only manually connected loopback peers, disables public discovery/port mapping
and supplies no public tracker. It writes a 192 MiB original byte fixture and
temporary native storage under the report directory, then removes owned scratch
directories. Raw samples remain local. It does not use the user's media, server
process, cache path or settings. This work changes no stable release approval.
