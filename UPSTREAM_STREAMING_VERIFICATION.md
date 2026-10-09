# LT maintenance and high bitrate streaming verification

Updated 8 October 2026. Development implementation; published Preview 2 `.5`
and the user's existing server executable/state are unchanged. Integration is in
[PR #3](https://github.com/gnbk21/TorrServer-LT-Flow/pull/3).

## Source and implementation

Reviewed upstream release [MatriX.145.LT-1.1.10](https://github.com/trinity-aml/TorrServer-LT/releases/tag/MatriX.145.LT-1.1.10), source
`c111484d084619b3fefa8c4e61f90f81c5099253`, published 2 October 2026.
The tested streaming revision used Go 1.26.8, libtorrent 2.1.2 and its reviewed
native patches. Preview 2 `.7` upgrades Go to 1.26.9 and x/net to v0.60.0 for
the 8 October security fixes; its tagged release gates verify those new pins.

| Area | Implemented behavior |
| --- | --- |
| HTTPS | Validate certificate/key without a listener; preserve user files; regenerate only managed self-signed pairs; atomic writes; restrict managed keys with Unix modes and Windows protected DACLs; reject partial CLI or saved pairs; support generated paths in read-only DB mode |
| Poster persistence | Unchanged artwork skips network work; transient failures preserve the prior poster; bounded image-header validation avoids full pixel allocation; reject invalid schemes, embedded credentials and invalid responses |
| Linux CA roots | Respect explicit root environment variables; discover existing system or Entware roots before outgoing TLS; warn if none are available |
| Optional HLS | AAC Main/SSR/LTP are re-encoded; compatible AAC remains eligible for copy; pipeline/profile regressions run with the `gst` build tag |
| macOS installer | Preserve existing launch context, port, authentication, logging and read-only flags during an upstream installer update |
| Engine reconnect | Remove an unconditional one-second sleep; retry only typed transient timeouts, at most three attempts with cancellable 200/400 ms waits; preserve settings rollback and errors |
| Native priorities | Remove outgoing deadlines before restoring the application vector; restore even an unchanged vector; reset deadlines before raising sequential preload priority; retain other device pipelines during a seek |
| Consumption evidence | Qualify sequential high bitrate HTTP progress without requiring probe metadata; detect individual-read jumps; expire idle rates and reset confidence on seek/idle transitions |
| Adaptive profile | Explicit opt-in, Legacy native peer settings, existing full deadline ramp, bounded VBR hints above credible metadata, recent qualified outage reserve and existing cache/time limits; no automatic profile switch |
| Diagnostics | Async native snapshot: at most 64 urgent pieces / 8,192 blocks, native priorities, requested/writing/finished/receiving states and duplicates; separate truncation; stale samples stay unknown; backend/Go/TypeScript contracts, seven-language UI and scalar retained history |

The HTTPS behavior intentionally differs from upstream's fallback: invalid
explicit HTTPS configuration fails before listeners open. It neither substitutes
a new identity nor silently downgrades to HTTP. Managed Windows keys reuse Flow's
existing secret-file protection mechanism.

## Bugs reproduced during verification

1. Native deadline removal demotes a piece to priority 1 independently of the
   cached Go vector. The actual native regression checks both parking at zero
   and keeping a preload reservation at its intended priority.
2. Without media metadata, the former 16 MiB **sample** jump threshold prevented
   sustained progress above about 64 Mbps from reaching the two-second rate
   sample. A jump check between individual reads permits normal sequential
   progress; unit regressions cover 90/120/200 Mbps. Fresh observations after an
   idle gap or seek no longer resurrect the previous rate/confidence.
3. The CLI ignored a certificate unless both paths were provided. A partial
   pair could generate another identity. The executable regression first
   reproduced that behavior, then verified clear failure and untouched files.
4. Adding the poster/utils package to race CI exposed old tracker test workers
   reading global fixtures after cleanup. Refresh workers now capture their
   interval and support stop/join before fixtures change. No race gate is
   suppressed, and production retains its process-wide refresh interval.

## Verification record

Runtime source `7c46fd4d693d163b3bb6412d59176f3efb457f5d` passed the full
[cross/native/service pipeline](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37752712232),
[macOS builds and native tests](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37752712319)
and [branch checks](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37752712457).
This includes native units, Go vet/race, reachable vulnerability scan, bounded
fuzzing, GStreamer-tagged pipeline tests, actual Go DTO validation, frontend
type/lint/build/unit/browser checks, controlled playback and Windows SCM
crash recovery/update/rollback. Optional native-layer comparison and container
publication are intentionally skipped on a pull request.

The tested Windows artifact stamps PR merge `8a7d740f95310ccb0c69c2dc8f6c79424dbce6c8`;
GitHub comparison verified no file differences from that runtime source. Its
SHA-256 is `d710ded95c65967c75581b94bfa4cb7aefd0a78c3b925564ff06eb499fd54eff`.
Subsequent documentation and development-harness edits do not alter server/web
runtime source. The requirement ledger is
[UPSTREAM_STREAMING_CHECKLIST.md](UPSTREAM_STREAMING_CHECKLIST.md).

Completed local checks include portable Go regressions, Windows protected-key
ACLs, standalone CA/poster tests, 15 Python socket/build tests, installer shell
syntax, frontend type checking/lint/build and 51 unit/component tests. The one
local contract-export skip is covered by the actual native Go DTO export in CI.
Generated HTTPS, invalid user files, partial CLI pairs, a lone default key and
read-only startup passed again on the final Windows executable above.

## Measurement method and limits

The real executable runs against a fresh local tracker and original generated
MPEG-TS, with 4 MiB torrent pieces. The transport includes padding: this measures
90/120 Mbps byte delivery and burst demand, not encoded visual complexity or
phone decoding. Public peer discovery is disabled. Calibration verifies the
controlled peer's direct byte capacity before each comparison.

The harness reports exact Range bytes, startup/TTFB, paced read waits, seeks,
cancellation, qualified consumption, media confidence, observed cache/RSS and
owned-process CPU when available. Explicit preload cases exercise the Lampa-style
bootstrap/probe path; direct Range cases exercise absent-metadata fallback.
Cache allowance is limited to the selected budget plus two pieces for concurrent
work. Reports include failures; a final cumulative catch-up does not erase a
long individual read wait.

The cache setting bounds media residency, not total process RSS. Go/native
metadata, sockets and allocation/GC headroom also consume memory. Observed RSS
and process CPU are reported separately; a two-minute check is not an endurance
or whole-process hard memory-limit guarantee.

The initial harness's fixed poll capped a peer below the requested high bitrate,
and its short socket send timeout could disconnect under ordinary backpressure.
Those uncalibrated cases are excluded from performance comparisons. Subsequent
calibrated intermediate builds are debugging evidence, not final-estimator
acceptance. In particular, the intermediate 120 Mbps / 2 GiB burst case exceeded
its time budget; the old baseline also timed out. Neither is called a pass.

Connection diagnostics then found a second source constraint in the long runs:
the fixture closed at its 2,048 pending-request ceiling without advertising a
capacity. Corrected high bitrate cases advertise `reqq=512` before unchoke,
retain bounded cancellation headroom and report queue peaks/closure reasons.
This uses the standard [BEP 10 request capacity](https://www.bittorrent.org/beps/bep_0010.html).
The pinned native scheduler may queue time-critical work beyond its ordinary
desired queue; the fixture correction changes no production timeout or peer cap.
Earlier calibrated runs remain recorded and cannot establish a profile ranking.

## Final executable comparisons

Scalar results, failed cases, source/fixture hashes and method limits are in
[HIGH_BITRATE_EVIDENCE.json](HIGH_BITRATE_EVIDENCE.json). `reqq-*` runs use the
corrected advertised source queue. Earlier runs are preserved as investigation
evidence. All comparisons run sequentially with the same generated source.

The direct 120 Mbps case alternates nominal demand and 180 Mbps bursts, delivers
2.16 GB over a scheduled 120 seconds and uses a 2 GiB media cache. No probe
metadata was available, so it also exercises high-rate consumption fallback.
Cold first-byte latency was 5.84 s for baseline Legacy, 6.32 s for final Legacy
and 6.29 s for final Adaptive; explicit-preload latency is measured separately.

| Executable/profile | Direct peer calibration, Mbps | Behind schedule, ms | Reads >=250 ms | Longest read, ms | CPU seconds | Peak observed RSS, GiB |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Baseline `99888e6f`, Legacy | 236.37 | 0.00 | 37 | 1,384.98 | 25.72 | 2.77 |
| Final `7c46fd4d`, Legacy | 236.07 | 0.67 | 62 | 1,121.18 | 27.42 | 2.70 |
| Final `7c46fd4d`, Adaptive | 186.67 | 3.72 | 62 | 1,175.54 | 31.11 | 3.01 |

All three delivered identical verified bytes, passed forward/backward/near-EOF
Ranges and cancellation/warm reads, returned to zero readers, stayed within the
media-cache allowance and used one source connection without queue-limit closure.
There are individual read waits despite cumulative catch-up. Calibration differed
between runs, CPU/RSS varied and each is one repetition: this is not evidence of
a speed ranking or Adaptive superiority.

The estimator correction is measurable independently of the ranking: after the
first 30 seconds, the old build's median observed consumption was 7.18 MB/s
(about 57 Mbps), while final Legacy reported 18.22 MB/s (about 146 Mbps) and
final Adaptive reported 18.67 MB/s (about 149 Mbps).
Actual scheduled byte demand averaged 18 MB/s (144 Mbps). These are HTTP demand
observations, not a decoded-frame bitrate. Unit regressions establish the exact
90/120/200 Mbps sequential-rate behavior without fixture timing variation.

The corrected explicit-preload outage case uses 90 Mbps, a 512 MiB media cache
and the same peer outages at 16–19 and 32–35 seconds from swarm start. Both runs
delivered the same 1.35 GB over a scheduled 120 seconds, passed all Range/reader
checks, qualified consumption and credible probe metadata, and stayed within the
cache allowance.

| Executable/profile | Preload ready, seconds | First byte after preload, ms | Reads >=250 ms | Longest read, seconds | CPU seconds |
| --- | ---: | ---: | ---: | ---: | ---: |
| Baseline `99888e6f`, Legacy | 4.60 | 1.08 | 12 | 9.57 | 16.33 |
| Final `7c46fd4d`, Adaptive | 5.03 | 1.00 | 9 | 15.87 | 13.11 |

Fewer long reads accompanied a worse longest wait in this pair. Both caught up
cumulatively, which does not erase that wait. The smaller cache also bounds how
much extra reserve the profile can retain. This result supports keeping Legacy
as default and Adaptive experimental; it does not establish smoother playback.

No consistent "better than Legacy" ranking is established. A dynamic bounded
urgent horizon remains research: prior rotated screens showed more blocked
reads, so it is not enabled. Real Android/Lampa/Just Player playback, public
swarms, GStreamer browser playback, physical network recovery, multi-hour
resources and a macOS service update remain separate acceptance. Source binding
is not verified VPN leak protection. No paid provider, router change, full
episode preparation or quality reduction is introduced.
