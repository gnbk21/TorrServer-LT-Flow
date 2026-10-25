# Compatible parallel streaming controls

Included in Flow 1.1.6, these controls refine libtorrent 2.1.2's existing parallel block
scheduler. They keep ordinary torrent peers, original container bytes, Range
playback, verification and the existing cache. They do not add QUIC or HTTP/3.
The motivation and original measurements are in
[the evaluation](../history/PARALLEL_STREAMING_EVALUATION.md).

## Controls and defaults

In Settings → Flow, two independent experiments are available:

| Setting in BitTorr.Flow | Behavior | Default |
| --- | --- | --- |
| FrontierRecovery | Use recent block service observations to recover a late block at a blocked reader's current piece | false |
| AdaptivePeerRequests | Pace each qualified peer's request credit and retain bounded ordinary forward work | false |

Either switch enables the existing capacity guard as a prerequisite. Neither
changes the selected swarm profile. Legacy remains the default; disabling Flow
disables both native policies. Applying these engine settings uses the existing
idle/restart configuration path. Old configurations need no migration, and older
native engines report unavailable observations rather than invented timings.

## Late-block recovery

The existing deadline scheduler still chooses an alternate eligible supplier.
Go publishes at most eight currently blocked reader pieces. Publications replace
old targets and expire after three seconds, including on paused torrents. Existing
reader lifecycle, seek cancellation and piece priority reconciliation remain in
charge. Other reader pieces retain normal deadline scheduling.

The additional recovery decision requires eight unambiguous completed block
requests from the supplier and an observation within ten seconds. Delay includes
remote queueing and upload scheduling: it is not packet RTT. A smoothed service
mean and deviation set a late threshold bounded to 250–3000 ms. A response that
is at least three quarters received, or is receiving recent data, is protected
from this early retry.

With FrontierRecovery enabled, duplicate admission across native request paths
allows at most two logical copies of a block. Recovery credit is bounded to 16
requests/256 KiB per torrent and 64 requests/1 MiB across the session, both
concurrently and per one-second issue window. Shared ownership follows queued
requests through sends, cancels, rejects, disconnects and destruction. Cancel
messages cannot guarantee that a remote peer stops sending; late physical replies
can therefore outlive a logical request.

Scanning is limited to 8192 outstanding requests per pass, at most once per
100 ms, with 64 late candidates retained. Truncated, stale, ambiguous or missing
evidence falls back to existing scheduling. A bounded cancellation filter excludes
ambiguous non-FAST late replies from service estimates. Its conservative false
positives discard evidence; they do not reject useful payload.

## Per-peer request credit

AdaptivePeerRequests uses measured payload service, its variation, the peer's
advertised request limit and the local maximum. Its target horizon is bounded
to 250–2000 ms. Changes occur at most every 500 ms, with a deadband and steps of
roughly one quarter of current credit (at least two requests). Native snubbing,
slow-start and admission rules still apply. Missing or stale service evidence
retains native budgeting.

During time-critical downloading, ordinary forward work receives a bounded
share of available peer credit instead of an unconditional one-block allowance.
Urgent refill retains first opportunity; the capacity guard still enforces the
actual remaining local and remote slots.

## Connection reuse

Existing warm torrent connections and metadata reuse avoid unnecessary setup.
Opt-in public peer hints now retain a bounded ranking of recent useful suppliers,
then known lower service delay. At most 32 canonical endpoints are saved; legacy
unscored hint files remain readable. Restoration uses batches of four with
50–150 ms staggering and at most four restoration owners. Existing expiry,
network checks, private-torrent restrictions, interface policy and native dial
backoff remain effective.

Libtorrent already implements TCP/uTP fallback. This change does not add competing
transport handshakes, change owner transport preferences or claim transparent
connection migration.

## Diagnostics and interpretation

Live sparse observations expose bounded numeric data through the existing API,
the translated diagnostics drawer and redacted support export. Opt-in history
records a stream sample on the existing 30-second cadence. No peer or media
identity is added to these samples.

- Startup shows discovery, socket connection, protocol handshake, metadata,
  first payload, bootstrap, probe, reserve and first-byte observations. Missing
  values remain unknown; a reused probe is explicitly zero. Milestones use the
  displayed torrent/preload/operation clock bases and can overlap. They must not
  be added together. The handshake operation includes negotiation after socket
  connection; it does not split encryption into a separately measured phase.
- Block service P50/P95 values are lifetime histogram bucket upper bounds for
  each transport/address family. They describe completed, unambiguous requests,
  not network RTT or an independently timed ACK.
- Useful bytes count first-copy payload observations, not decoded, verified or
  necessarily player-required bytes. Existing verified-contiguous and reader
  metrics remain the evidence of playable frontier advancement.
- Redundant bytes include observed extra replies. Obsolete bytes are the subset
  explicitly marked no longer wanted; this is not an exhaustive attribution of
  every byte made unnecessary by a seek.
- Recovery requests/suppressions, current/peak duplicate credit, frontiers,
  scan truncation and lifetime request-budget changes explain policy activity.
  A missing or stale live snapshot is shown as unavailable.

Existing demand/reserve, preparation admission, memory budgets, four ordered
storage lanes, verified local reuse and web seeds remain integrated. More worker
threads cannot increase a remote seed's upload rate. Diagnose storage queue tails
and CPU before changing concurrency. A healthy verified buffer with phone-facing
delivery stalls still calls for LAN/player investigation; router SQM and phone
Wi-Fi conditions are outside an automatic server setting.

## Short isolated verification

The comparison harness uses generated, offset-distinct bytes and loopback peers,
fresh settings/state per trial, strict advertised queues and a calibrated supply.
Four policies share Legacy plus the existing capacity guard: capacity alone,
frontier alone, request credit alone and both. Rotated order limits order bias.

    python build/parallel_streaming_check.py --executable PATH --output NEW_DIRECTORY --case stragglers --rotations 3 --disk

CI covers healthy peers, delayed blocks, small queues, mixed peers, outages, RAM
and disk, including Windows. Healthy peers are tested at 100.66 Mbps; constrained
queue/latency cases use lower demand matched to their calibrated supply. A separate
owned Linux namespace checks jitter, configured packet loss, reordering and an
outage with both controls enabled, preserving qdisc observations before reset.
Reports retain failed trials, executable identity,
startup and seek timing, total/worst read waits, lateness, duplicate bytes, CPU,
RSS and integrity/capacity results. Native simulations cover clock freshness,
hysteresis, cancellation ambiguity and credit ownership. Existing independent
reader, seek, lifecycle, storage, network impairment and resource regressions
remain required.

These checks are byte-transport tests. They do not certify phone decoding,
public-swarm performance or multi-hour stability. No two-hour test is required,
and no experiment is promoted automatically by the harness. The owner's running
installation and settings are not used for these comparisons.
