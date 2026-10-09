# High bitrate streaming in Flow development builds

These changes stream the original torrent through the existing rolling cache.
They require no paid account, whole-episode download or quality conversion.
Legacy remains the default. No profile can sustain playback indefinitely when
useful verified supply stays below the media's consumption rate.

## Controls and rollback

**Settings → Flow** exposes three independent experiments, all **off by default**:

| Setting | Behavior | Limitation |
| --- | --- | --- |
| Capacity-aware requests | Limits urgent and ordinary outstanding requests to the smaller of the peer's BEP 10 `reqq` and local cap. | A small cap on a distant peer can limit throughput. It cannot increase that peer's upload capacity. |
| Adaptive urgent horizon | With fresh qualified demand, gives deadlines to a smoothed near-term subset while keeping ordinary nonzero forward priorities. Each independent reader retains its own window. | Missing demand falls back to the existing full deadline ramp. Queue delivery estimates are not RTT measurements. |
| Container burst hints | Asynchronously reads already verified, resident MKV Cues or MP4 video chunk/timing indexes. Adaptive may raise its demand estimate, capped at twice the credible average. | Coarse file byte/time slopes include interleaving/overhead. Unsupported, missing, oversized or malformed indexes fall back immediately; no index fetch or full-file scan is started. |

Apply restarts the torrent engine. Test one switch at a time. To roll back, turn
the switches off and select Legacy, then Apply. Existing profile choices, clients,
cache layout and verified resume remain supported. Adaptive additionally learns
bounded reserve margins from recent qualified supply shortfalls, including
partial slowdowns; excluded/idle/probe intervals do not bridge the history.
This reserve stays within configured startup, read-ahead and cache limits.

Storage changes apply to all profiles: disk files reuse a bounded 64-handle pool;
native reads/writes/hashes run on four ordered worker lanes. The queue throttles
at 16 MiB, resumes below 8 MiB and rejects ordinary admission above 64 MiB or
8192 jobs with an error. Zero-byte piece clears and lifecycle fences remain
admissible so cleanup can finish under load. Posted read buffers have a separate
process-wide 64 MiB bound.
These are in-flight buffers, not a second persistent cache. Hashing uses a fixed
64 KiB worker buffer. Handles close before removal/migration and jobs retain
their storage until completions settle.

Container inspection checks a 150 ms deadline between bounded reads, with a
4 MiB inspection budget. This is not a hard timeout for an individual operating
system read; inspection runs asynchronously outside the streaming open path.

## Identify the bottleneck

Open the torrent's **Flow diagnostics** while the symptom occurs:

- **Readable contiguous bytes** can include responsive, not-yet-verified blocks.
  **Verified contiguous bytes** stop at the first hole. Both are file bounded.
- **Frontier growth** measures newly contiguous verified progress, independently
  of the reader moving forward. Unknown/stale evidence must not be called zero.
- **Delivery deficit** measures clean qualified demand shortfalls over recent
  intervals; it is not a decoded player clock or a forecast of future seeds.
- **Oldest urgent request age** starts when a request leaves native send buffering.
  `-1` means unavailable. The bounded peer/block scan can be truncated.
- **Storage I/O** shows active queue bytes/jobs and active-backend peaks/rejections.
  Completion counts and logarithmic wait/callback latency buckets are process
  lifetime aggregates. P95 is a bucket upper bound, not an exact percentile or
  per-torrent measurement. A hash callback includes reading and hashing.

High queue/callback time suggests local cache work; old urgent requests with
low useful progress suggest the supplier/pipeline. Healthy contiguous supply
with phone stalls calls for LAN/player/decoder measurements. Correlate timestamps;
one counter alone does not identify a cause.

### Phone/LAN transfer check

On the phone, open Flow's **Settings → Network → LAN transfer test**. Each run
fetches 16 MiB from this same server, with a 20-second browser timeout, a
15-second server write deadline and at most two simultaneous requests. Cancel
stops the transfer. Results stay in the page; no torrent or external URL is used.
Authentication and management-access policy still apply.

Repeat at the viewing location and compare with a wired client. Mbps includes
startup; read-gap P95/max include browser scheduling and coalescing. A short
successful burst does not prove sustained Wi-Fi capacity or decoder health.
For longer controlled LAN checks, an optional [iperf3](https://software.es.net/iperf/invoking.html)
server on the PC and `iperf3 -c <PC-LAN-IP> -R -t 60` on a compatible phone client
measure the server-to-phone direction. Bind the server to the LAN address and
allow access only from the LAN; stop it after testing. Flow does not install it
or change firewall rules. Start with an idle measurement. An optional comparison
during playback adds competing traffic and can itself cause stalls; record that
condition and do not treat test-induced stalls as the playback baseline.

## Optional Flow Player

Stock Just Player remains supported. The separate **Flow Player (experimental)**
APK patches immutable Just Player commit
`aa85148f6ccbfdf931fe207bb75ec77c478d8eb0` (v0.217, Media3 1.11.1).
Its distinct package `com.brouken.player.flow` installs alongside the stock app.
The workflow **optional-flow-player** builds a debug-signed
APK using the pinned source, recursive dependencies and source patch. Build
artifacts contain source/license material and checksums; this is not a stable
store release or an automatic player update.
Debug signing keys can differ between CI runs; replacing an installed build may
require uninstalling that separate experimental app first.

For HTTP(S) playback only, a memory policy chooses a byte target using maximum
heap, used heap and available system memory, capped at 384 MiB and 32 MiB on
low-memory devices. Media3 retains byte priority, 30–90 second time targets,
2.5-second startup, 7.5-second rebuffer target and no retained back buffer.
The byte target can stop buffering below the time target. Native decoder/surface
memory and later memory pressure are outside this startup-time estimate; the
policy is not a hard process-memory ceiling or an OOM guarantee. It does not
change resolution, bitrate, codecs or HTTP URLs.

Numeric local diagnostics are **off** unless the Android launch intent includes
boolean `flow_diagnostics=true`. For example, with an owned test URL:

```text
adb shell am start -a android.intent.action.VIEW -d "http://PC:8090/owned-test-url" -t "video/*" -p com.brouken.player.flow --ez flow_diagnostics true
adb logcat -s FlowPlayback:I
```

Only this added `FlowPlayback` tag omits URLs, titles, headers, credentials and
exception messages. Android/system/upstream logs are separate: do not share raw
logcat indiscriminately. Samples include buffered time, state, bandwidth estimate,
playback rebuffer events/time, seek count, dropped frames and numeric error codes.
Startup and seek buffering are excluded from supply-stall counts. Handler work
stops on player release. Lampa may omit the diagnostic extra; normal playback
still works. Select stock Just Player to roll back, or uninstall the separate app.

Reproduction requires Git, Python, JDK 21 and the pinned project's Android SDK/NDK:

```text
git clone --recurse-submodules https://github.com/moneytoo/Player.git player
git -C player checkout aa85148f6ccbfdf931fe207bb75ec77c478d8eb0
git -C player submodule update --init --recursive
git -C player lfs pull
python build/prepare_just_player.py player
```

Then use the SDK packages and Gradle target in
[the workflow](.github/workflows/just-player.yml). The preparation script rejects
other commits, checks patch context and is repeatable. Java policy checks do not
replace an APK compile, Android memory-pressure test or representative phone playback.

## Conditional network/source measures

**Upload headroom/SQM:** first record idle latency, latency during an upload,
upload rate, useful torrent delivery and phone stalls. If uplink saturation
coincides with increased latency, test a modest upload limit (start around
85–90% of measured sustainable upload). Re-measure rather than assuming one
percentage fits every connection. An unnecessarily low limit can reduce swarm
reciprocity. Record the old Flow upload setting and restore it if worse.

On a router you control, optional SQM using
[FQ-CoDel](https://www.rfc-editor.org/rfc/rfc8290.html) or
[CAKE](https://www.bufferbloat.net/projects/codel/wiki/Cake/) can manage bottleneck
queues. Back up router configuration, shape below the measured bottleneck, test
both directions and check router CPU and throughput. Hardware/offload interactions
depend on the router. Restore the backup if latency/throughput worsens. These are
conditional operator procedures; Flow changes no router, firewall, VPN or ISP setting.

**Free HTTP mirrors:** existing native web seeds can supplement peers only when
an authorized mirror contains the torrent's exact original bytes and layout.
Native verification and private/source restrictions remain in force. No matching
free mirror is guaranteed. Use the existing Sources controls; do not substitute
an arbitrary same-title encode or bypass access restrictions.

**Lossless remux:** [FFmpeg stream copy](https://ffmpeg.org/ffmpeg.html#Streamcopy)
can improve container compatibility without re-encoding media streams. However,
the resulting file has different bytes and torrent hashes. It needs its own
source/torrent and is not an in-place patch to an existing swarm. This optional
source-publishing technique is not required for Flow streaming and does not
silently convert or download the user's episode.

**Cross-swarm/v2 reuse:** deferred engineering candidate. Current Flow identities,
resume/storage verification and native callbacks require end-to-end v2/hybrid
support (including SHA-256 `async_hash2`), compatible piece/file mapping and
verified exact-byte reuse. Matching titles are insufficient. Enabling v2/QUIC,
an ML controller or unrestricted duplicate requests is not a supported speed fix.

## Reproduce comparisons

```text
python build/request_capacity_check.py --executable <Flow> --output <new-directory>
python build/high_bitrate_matrix.py --executable <Flow> --baseline <previous-Flow> --output <new-directory> --mbps 120 --seconds 120 --rotations 3 --cache-mb 512 --cases healthy outages mixed-peers bursts
```

`--baseline` is optional; without it the policies are current Legacy, current
Adaptive and Adaptive with the three switches. The matrix rotates policy order,
uses the same generated fixture and fixed 32 MiB startup min/max, calibrates peer
capacity, aligns outages to playback start, verifies every delivered byte and
checks forward/backward/EOF/cancel ranges and reader cleanup. It records startup,
total read waits, blocked waits, P95/P99/max, CPU, RSS, cache and storage metrics;
all repetitions and failures remain in reports. `--disk`, `--piece-mb 1|4|16`
and `--cache-mb 512|2048` exercise storage and piece boundaries. CI runs three
rotations of two-minute 90/120 Mbps cases across these configurations.

The generated MPEG-TS uses padding at 120 Mbps transport; it is not a real
high-bitrate HEVC/HDR decoder benchmark. Repeated local byte delivery, sparse
fixtures and races are gates, not proof of better public-swarm or phone playback.
Do not promote a policy from the best run: require consistent wait/tail gains
without startup, seek, integrity or resource regressions, followed by a long
original-quality Lampa/Just Player session on the actual device/network.

Implementation and remaining acceptance are tracked in
[HIGH_BITRATE_IMPLEMENTATION.md](HIGH_BITRATE_IMPLEMENTATION.md).
