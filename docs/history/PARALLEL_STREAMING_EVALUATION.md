# Parallel streaming and connection performance evaluation

Date: 2026-10-24. Status: evaluation and proposed experiments; no runtime changes.

Source reviewed: `d5213eb51a0aa6edf57e65ce0dd77fcc23c109af`, with streaming implementation at `a9e1434deb67f8dbd37b2e149bec3af44fe4b6ec`. This evaluates the current development branch. It does not establish that the installed 1.1.5 executable includes every mechanism below.

## 1. Recommendation

Improve the existing native block scheduler and connection lifecycle. Borrow the principles of independent progress, avoiding repeated setup, adaptive recovery, and bounded flow control. Keep ordinary torrent peers compatible.

The highest priority is reducing time spent waiting for the next required bytes while other bytes arrive. Increasing aggregate download speed is useful only when it also advances the playable frontier or builds a usable reserve. A new default profile is not justified by the current comparisons.

## 2. Which QUIC ideas transfer?

QUIC is a transport protocol usable independently of HTTP/3. Its independent streams isolate ordered delivery: loss affecting one stream need not hold up unrelated streams. Its handshake integrates transport and security establishment; optional early data relies on previous connection state. These mechanisms require participating endpoints. [QUIC transport](https://www.rfc-editor.org/rfc/rfc9000.html#section-13), [QUIC handshake and early data](https://www.rfc-editor.org/rfc/rfc9001.html#section-4.6).

For Flow, the following are application-level analogies. They are not implementations of QUIC:

| Principle | Compatible application in Flow | Existing foundation | Further opportunity |
| --- | --- | --- | --- |
| Avoid repeated connection setup | Retain a bounded set of useful live peers between short playback gaps | Warm torrent lifecycle and retained seed connections | Measure avoidable reconnects and retain peers according to recent usefulness, idle cost, and network policy |
| Resume using remembered information | Reuse valid local metadata and optionally saved public peer hints | DHT persistence, metadata recovery, opt-in peer hints | Measure which startup phases these actually shorten; expire stale hints and preserve private-torrent restrictions |
| Let independent work progress | Request different blocks from multiple peers and store out-of-order arrivals | Native parallel downloading and independent reader windows | Keep late distant work from consuming request slots needed at the playback frontier |
| Recover promptly from missing progress | Request an urgently needed late block from a second capable peer | Native deadline-driven duplicate requests | Refine the conditions, budget, supplier choice, and cancellation for this existing mechanism |
| Bound work in flight | Match request credit to service capacity and receiver limits | Per-peer pipelines and optional capacity-aware native patch | Improve measurement and policy; avoid both empty pipelines and excessive remote queues |
| React to network changes | Reconnect while retaining useful cache and metadata | Retry/backoff and interface policy | Improve recovery evidence and hysteresis; reconnection is not transparent transport migration |

There are two different blocking problems:

1. **Transport ordering:** a TCP connection cannot deliver bytes after a missing earlier byte. Flow cannot change this contract unilaterally. A different peer connection can continue independently.
2. **Playback dependency:** a player needs particular bytes next. Even an unordered transport cannot make a missing required block available. Obtaining that block from another supplier can reduce the delay.

Putting the entire existing BitTorrent byte stream into one QUIC stream would retain ordering within that stream. Separating blocks into independent QUIC streams would require a compatible application protocol at both peers. A unilateral Flow patch cannot make ordinary public peers understand it.

QUIC's adaptive loss recovery uses acknowledgments, RTT estimates, and bounded probes. The useful analogy is to react to measured lack of progress. A torrent request's completion time also includes the remote client's queue and upload policy: it is not a clean RTT sample, and a late block does not prove packet loss. [QUIC recovery](https://www.rfc-editor.org/rfc/rfc9002.html#section-6.2).

## 3. Parallelism already present

BitTorrent supports pipelined requests; Flow's libtorrent engine already downloads blocks concurrently from multiple peers. `reqq` advertises a peer's supported outstanding request count. [BitTorrent wire protocol](https://www.bittorrent.org/beps/bep_0003.html), [extension handshake](https://www.bittorrent.org/beps/bep_0010.html).

Reviewed mechanisms:

- `server/torr/btserver.go`: Legacy uses a one-second request-queue target, local maximum of 1,500 outgoing requests per peer, connection ramp settings, both TCP/uTP according to owner settings, and native web-seed support. The default configured peer limit is 50 per torrent; a separate session limit is at least 200 when derived from that setting. These are ceilings and policies, not promises of useful active suppliers.
- `server/torr/swarm_profile.go`: Adaptive retains Legacy's session tuning. Other profiles deliberately replace selected overrides. The capacity-aware request experiment is independent of the named profile.
- `server/torr/storage/torrstor/cache.go`: concurrent reader windows, blocked-piece urgency, forward priorities, header/index pins, seek cancellation, and rolling eviction already exist.
- `server/flow/deadline.go` and `frontier.go`: demand-related deadlines, optional scarce-piece hints, a bounded urgent horizon, and verified frontier growth already exist.
- `build/native-patches/flow-request-capacity.patch`: optional enforcement of local/remote request capacity, plus actual request send-time observations.
- `flow-urgent-refill.patch`: with that experiment enabled, newly available request credit gives urgent scheduling an opportunity before ordinary refill.
- `flow-cancel-credit.patch`: handles ambiguous late replies after non-FAST cancellation within the capacity experiment.
- `94bffc25272b-queue-time.patch`: already corrects the units used to estimate queued delivery time. Re-proposing this upstream fix would duplicate existing work.
- `server/lt/lt_io_queue.hpp`: four worker lanes allow different pieces' storage operations to run concurrently while preserving per-piece order. Queue backpressure, admission limits, and lifecycle fences already exist.
- `server/torr/storage/torrstor/verified_reuse.go`: bounded local reuse of compatible, authenticated v2/hybrid file data already exists. It verifies bytes and falls back to peer fetching; it does not discover or combine arbitrary similarly named releases.
- `server/settings/flow.go`: startup/read-ahead controls, memory budgets, warm sessions, network retry controls, and optional scheduling experiments already exist.

Native deadline scheduling already chooses suppliers by estimated queue service and can issue duplicate requests for late pieces. The proposed work must refine that behavior rather than add a second independent retry system. [libtorrent streaming implementation](https://www.libtorrent.org/streaming.html).

## 4. Evidence reviewed

Downloaded existing artifacts from [build run 40113146808](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40113146808). No new movie, public-swarm, or endurance test was started.

Local evidence is under `.tools/parallel-performance-evaluation/ci-a9e1434/`, in each `high-bitrate-*` directory's `report.json`.

The reports contain **122 passing trials**:

- Eight ordinary cases: three policies, three rotations, 120 seconds per trial: 72 trials.
- Two straggler cases: five policies, five rotations, 30 seconds per trial: 50 trials, including immutable 1.1.3 baseline comparisons.
- The combined experiment enables capacity-aware requests, adaptive urgent horizon, and resident container burst hints together. These results do not attribute a benefit or regression to one switch.

### Median time spent in individual HTTP reads lasting at least 250 ms

Seconds; smaller is preferable. These are client-side byte-transport waits, not measured decoder stalls or watch time lost.

| Target / fixture / cache | Current Legacy | Current Adaptive | Combined experiments |
| --- | ---: | ---: | ---: |
| 90 Mbps, healthy, 512 MiB RAM / 1 MiB pieces | 9.02 | 14.69 | 0.00 |
| 90 Mbps, bursts, 512 MiB RAM / 4 MiB pieces | 0.00 | 0.00 | 0.00 |
| 90 Mbps, mixed peers, 2,048 MiB RAM / 4 MiB pieces | 0.35 | 0.52 | 1.07 |
| 90 Mbps, outages, 512 MiB disk / 4 MiB pieces | 26.81 | 28.51 | 17.73 |
| 120 Mbps, healthy, 2,048 MiB disk / 16 MiB pieces | 0.00 | 0.00 | 0.00 |
| 120 Mbps, bursts, 2,048 MiB disk / 4 MiB pieces | 42.19 | 47.75 | 0.00 |
| 120 Mbps, mixed peers, 512 MiB disk / 4 MiB pieces | 1.18 | 1.89 | 1.89 |
| 120 Mbps, outages, 512 MiB RAM / 4 MiB pieces | 13.54 | 12.92 | 14.29 |
| 90 Mbps, stragglers, 512 MiB RAM / 8 MiB pieces | 0.00 | 0.00 | 0.00 |
| 120 Mbps, stragglers, 512 MiB disk / 8 MiB pieces | 0.00 | 0.75 | 0.00 |

Important qualifications:

- Calibration showed roughly 171 Mbps for the 90 Mbps jobs and 224 Mbps for the 120 Mbps jobs. This verifies the local fixture supply, not the owner's ISP or phone Wi-Fi capacity.
- All trials passing means the harness's integrity, Range, cancellation, and resource gates passed. Some passing trials still contain substantial waits.
- The experimental combination has a strong result in one burst fixture and meaningful regressions elsewhere. In the 90 Mbps mixed-peer case, median native redundant bytes rose from 22.44 MiB with Legacy to 51.09 MiB with the combination.
- Median preload completion was approximately 5.4-6.6 seconds for ordinary Legacy cases. Some experimental cases were slower; the 90 Mbps healthy fixture was about 6.01 versus 7.01 seconds. Investigate the phase responsible before calling it a handshake delay.
- In the 120 Mbps straggler case, median across trials of cold-seek P95 first-byte time was about 150 ms for current Legacy and 334 ms for the combination. The corresponding earlier Legacy baseline was about 215 ms. Medians of trial P95s are not pooled P95s.
- For 512 MiB RAM ordinary cases, median observed peak RSS was around 1.14-1.18 GiB; the 2,048 MiB RAM mixed-peer case was around 2.19-2.24 GiB. Cache size is not a whole-process memory limit. These short runs do not establish an allocation leak or steady-state convergence.
- Read P99 alone can conceal rare long waits when the harness performs many small reads. Total blocked time, worst wait, lateness, and buffer depletion must accompany percentiles.
- The controlled stream is generated MPEG-TS byte transport. It does not establish HEVC/HDR decoding, phone playback, realistic WAN congestion, or public-swarm performance.

The existing netem report under `.tools/reliability-final-evidence/netem/report.json` also exercises jitter, packet loss, and an outage with byte-exact recovery. It is TCP-only and does not compare transport protocols. Its environment reports unsupported deterministic netem seeding, so exact impairment sequences are not guaranteed reproducible.

## 5. Prioritized development candidates

These are engineering recommendations and hypotheses, not measured new improvements.

### A. Attribute delays and wasted work before changing defaults

**Priority: first.** Extend the bounded existing evidence with startup phase durations, request completion distributions by transport/family, and frontier-specific waste accounting.

Distinguish discovery, socket connect, BitTorrent/security handshake, metadata, first payload, probe/index work, and startup reserve. Record unknown values where events are unavailable. Existing events and failure categories are a foundation, but the currently exported queue estimate is not network RTT.

Measure unique required bytes, redundant bytes, obsolete post-seek bytes, head-blocker age, and verified frontier advancement. Keep receiving progress separate from completed blocks and verified pieces. Use aggregate counters and bounded histograms; avoid per-block log spam or persistent peer identities.

This enables a useful explanation such as: "Peers are connected, but the next needed block is late; later blocks are arriving." It also prevents a 30-second metadata or preload delay from being mislabeled as a slow handshake.

### B. Refine late-block recovery at the playback frontier

**Priority: highest likely streaming benefit.** Integrate with the native deadline scheduler. When a required block is late relative to recent service behavior, the reserve is at risk, progress has stalled, and an alternate supplier is available, consider one bounded duplicate request.

Requirements for an experiment:

- Count existing native duplicates before granting new credit; candidate initial ceiling is two concurrent copies of a block across all recovery paths.
- Bound duplicated bytes and requests per torrent and across the session. Measure all late/cancelled replies; cancellation does not guarantee the remote peer stops sending.
- Use service-delay variation and actual request handoff age. Do not substitute the current queue estimate for RTT or blindly copy QUIC's packet-loss formula.
- Consider already receiving bytes and supplier availability. Avoid duplicating a nearly complete response solely because it is old.
- Cancel obsolete work on a seek, preserve generation/lifecycle handling, and preserve private/source/interface policy and verification behavior.
- Maintain useful forward work and fairness between independent viewers. An immediate-block focus must not collapse the pipeline to a single serial request.
- Disable the experiment or fall back to existing scheduling when evidence is missing or stale.

A fast alternate peer can work around a delayed peer connection. It cannot repair that connection's TCP transport, and no alternate supplier means no recovery shortcut.

### C. Improve per-peer request budgeting

**Priority: high, after A.** Make the optional capacity policy more responsive to observed service without simply raising caps.

Model request credit using measured useful delivery, remote capacity, outstanding work, receiving progress, and a bounded target horizon. Add hysteresis and paced adjustment to avoid oscillation and burst floods. Investigate whether the ordinary one-block fallback under time-critical capacity mode underfills useful peers in particular cases; this is a hypothesis requiring traces.

A bandwidth-delay product illustrates the reason for pipelines: at 100 Mbps over 100 ms, approximately 1.25 MB, or 77 blocks of 16 KiB, are needed in flight before accounting for processing or safety margin. Apply this to the relevant peer/path rate, not the sum of unrelated peers. Torrent service delay adds remote queuing and choking, so a pure network BDP formula is insufficient.

The code comment asserting that a one-second queue is above any real BDP is too broad. High-latency paths and remote scheduling delays exist. Test the assumption; it does not justify changing Legacy's target globally.

### D. Improve connection setup without inventing a peer protocol

**Priority: high for startup, conditional for steady playback.** Keep useful live connections through short gaps, reuse validated metadata, and score fresh public peer hints by actual results. All have existing foundations.

Before adding a dial race, inspect libtorrent's current TCP/uTP fallback and IPv4/IPv6 behavior. Test bounded staggered attempts only where the native behavior leaves a demonstrated delay. Deduplicate winners, limit pending attempts, apply backoff to failures, and preserve owner-selected transports and interface binding. More outgoing attempts can overload NAT state or recycle the same unreachable addresses.

A live connection can avoid a new handshake. Saved addresses and DHT state shorten discovery but do not eliminate a subsequent handshake. QUIC-style early data itself requires an agreed resumption mechanism; old peer addresses cannot serve as one.

### E. Balance reserve, work in flight, and competing viewers

**Priority: high.** Refine the existing demand controller and resource budget rather than allocate an unlimited buffer.

Use recent qualified supply deficits and variability to increase reserve within existing bounds; reduce it gradually after stable delivery. Preserve fast seeks and the ordinary fallback when rate evidence is unknown. Avoid optional next-episode or preparation work competing with a starving current stream. Test two viewers separately from two HTTP connections belonging to one viewer.

At 120 Mbps, 30 seconds of original container data is 450 MB; 60 seconds is 900 MB, before other memory overhead. A 2 GiB cache can represent about 143 seconds at that constant rate before pins, behind-data, other readers, and overhead. A larger cache absorbs temporary deficits; it cannot indefinitely compensate for supply below consumption.

### F. Optimize storage and memory only against measured bottlenecks

**Priority: evidence-dependent.** The existing four ordered lanes already provide parallel storage. More threads do not make a remote seed upload faster.

Use queue wait/callback tails, rejected jobs, CPU, allocations, and RSS to choose work. Candidate investigations are per-piece lock hold time, cgo transition frequency, redundant copies, and long hash jobs delaying urgent operations sharing a lane. Any lane/batching change must preserve write/hash/clear order and storage close fences. Do not blindly tune libtorrent's stock disk-thread settings: Flow uses a custom disk backend.

### G. Improve exact-byte supply where it exists

**Priority: opportunistic.** Native web seeds and authenticated local file reuse are already available. They can supplement a swarm where a matching source exists. Preserve exact byte identity, source policy, verification, quotas, and cancellation. Public metadata caches supply torrent descriptions, not a guaranteed video mirror or extra seed bandwidth.

### H. Check the PC-to-phone link independently

**Priority: diagnostic, not an automatic setting change.** A healthy torrent buffer with delivery stalls requires LAN/player investigation. Flow already offers a bounded phone-side LAN test and an experimental player buffer policy.

Measure actual phone-facing transfer under representative conditions, not just the PC's internet speed. A wired PC and a stable uncongested phone Wi-Fi link may improve headroom. Do not start competing bandwidth tests during a movie without the owner's intent.

If loaded latency rises substantially, evaluate upload headroom or router SQM using CAKE/FQ-CoDel. Router CPU and flow offload can limit the result; benchmark before/after. This is a network-specific option, not an application feature Flow can apply automatically. [OpenWrt SQM guidance](https://openwrt.org/docs/guide-user/network/traffic-shaping/sqm?s%5B%5D=packet).

## 6. Technologies with limited fit

| Idea | Assessment for this installation |
| --- | --- |
| QUIC principles at block-scheduler level | Strong fit for focused experiments A-D; does not require HTTP/3 |
| A new QUIC torrent transport | Requires compatible peers and application framing; unsuitable as a unilateral fix for existing public swarms |
| Switching every connection to UDP/uTP | uTP already exists and includes delay-based congestion control. UDP alone does not provide independent stream delivery; compare transports rather than assume a win. [BEP 29](https://www.bittorrent.org/beps/bep_0029.html) |
| MPTCP or multiple internet uplinks | Useful only with actual path diversity and suitable endpoint/OS support. MPTCP can fall back to TCP; it does not create capacity on one saturated shared uplink. Extra interfaces must respect binding policy. [MPTCP](https://www.rfc-editor.org/rfc/rfc8684.html) |
| Multiple HTTP range downloaders on the phone | Separate player work; can compete for cache, widen demand, and fetch obsolete data. No evidence this improves the current Just Player path |
| Whole-file sequential mode | Can assign an early block to a slow peer. Existing deadlines offer stronger control over streaming urgency |
| Many connections to the same peer | Subject to the remote upload limit and peer policy; not a substitute for independent suppliers |
| Arbitrary mixing of releases or new smaller torrent pieces | Requires matching authenticated bytes and existing torrent geometry. Names, episode number, resolution, and duration are insufficient |
| FEC / redundant encoded packets | Needs an encoder/source or negotiated peers and extra bandwidth; no unilateral recovery of unseen torrent bytes |
| BBR, io_uring, GPU hashing, or a different language | Investigate only for a demonstrated bottleneck and supported platform; no evidence they address the observed scheduler tradeoffs |

## 7. Verification and promotion criteria

1. Keep a measured baseline and vary one mechanism at a time before evaluating combinations. Keep startup reserve, file bytes, source capacity, cache backend, and piece geometry matched.
2. Use short deterministic native scheduling simulations for queue accounting, urgent-credit refill, late replies, ignored cancellations, choke/unchoke, missing suppliers, and seek generations.
3. Extend the existing controlled-peer tests with heterogeneity, strict small remote queues, realistic service latency, delayed last blocks, and false-positive duplicate triggers.
4. Reuse netem for loss, reordering, jitter, and outages. Record whether the host supports deterministic seeding and which transport is tested.
5. Compare startup phases, cold-seek first-byte time, total and worst read waits, buffer depletion, useful bytes, duplicate cost, CPU, memory, and storage tails. Do not promote on aggregate Mbps or P99 alone.
6. Cover both one viewer and independent viewers, RAM/disk, small/large pieces, disconnect/reconnect, cancellation, and bounded resource behavior. Byte integrity and lifecycle safety remain gates.
7. Require repeatable improvement in the target failure case with no material normal-path, startup, seek, or resource regression. Select numerical tolerances before implementation based on baseline variation; keep failures in the reports.
8. Use the owner's ordinary original-quality Lampa/Just Player playback as field confirmation when available. Controlled byte tests cannot certify visual playback. No two-hour test is required for this evaluation or proposed initial experiments.

## 8. What this evaluation establishes

Borrowing QUIC's design principles is technically useful, particularly independent progress, avoiding repeated setup, and adaptive bounded recovery. Flow already has many corresponding building blocks. The recommended contribution is to improve their decisions and measurement, especially the next missing block, rather than add generic parallel downloading.

No new transport, profile default, owner setting, running process, router rule, or release was changed. The practical gain of the proposed refinements remains to be demonstrated.
