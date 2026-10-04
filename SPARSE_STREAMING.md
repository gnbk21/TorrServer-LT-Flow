# Sparse streaming in development builds

These changes are on `feature/preview-2-hardening` / PR #3. Published Preview 2
`.5` does not contain them. No provider subscription or account is required.
The [implementation ledger](SPARSE_IMPLEMENTATION.md) records build and test
evidence separately from environmental acceptance.

## Diagnose the wait

Open a torrent's diagnostics. The wait reason distinguishes metadata, peer
discovery, pending connections, choking, missing connected-peer pieces,
insufficient delivery, local cache/probe work and readiness. Zero observed
availability means **not available from connected peers**, not globally dead.
Complementary partial peers can supply a complete file without any full seeder.

Native snapshots run asynchronously at most once every two seconds, with one
pending request per torrent, eight windows, 256 pieces and 512 peers. HTTP reads
use cached aggregates. Samples older than five seconds, truncated samples and
missing evidence do not establish absence. Source counts can overlap because a
peer may have multiple discovery sources. Queue times are native estimates;
failed bytes are hash failures, redundant bytes are duplicate payload, and request
timeouts/drops are separate alert counters. These are server observations, not
the phone's first decoded frame.

Opt-in diagnostic history adds numeric sparse summaries no more than once every
30 seconds. Its existing 256-event queue and four 1 MiB files remain unchanged.
It excludes peer identities, bitfields, media hashes/names, URLs and credentials.

## Prepare a slow or intermittent episode

1. Open **Files / Play** and expand **Prepare episode** for the desired file.
2. Start preparation. The server downloads ordinary bounded work independently
   of the current player's requests; a missing source does not expire the job.
3. Wait for **Ready** for complete verified retention, including container and
   arbitrary-seek data. Use the existing playback links above the control.

You can play earlier through those links, but unprepared regions still need
sources. Pause stops preparation demand while keeping pieces. Cancel also keeps
progress; Resume continues. Playback can independently request the same pieces.
Remove stops preparation and waits for all readers of that torrent to close
before removing its native handle and unshared prepared piece files. Shared
boundary pieces remain until their last preparation reservation is removed.
Deleting a torrent from the library also schedules its preparation jobs for
this cleanup, so a durable job cannot silently re-add the deleted torrent.

There are at most 16 jobs. `BitTorr.Flow.PreparationQuotaMB` defaults to 4096 MiB,
with supported values 64–1048576 MiB. The reservation charges whole boundary
pieces once per torrent, so it can exceed the selected file's length. It is
separate from `CacheSize`. Lowering the quota pauses active jobs whose combined
reservations exceed it; remove existing jobs to release the reservation.
Disk write, sync, state and cleanup failures are reported without claiming readiness.

Preparation extends the existing authoritative piece cache; it creates no
parallel media copy. Pieces in reserved ranges survive ordinary LRU eviction.
Other pieces remain subject to normal streaming eviction budgets. Cache extent
metrics exclude retained preparation pieces; reservation/progress appears in
preparation status. Process RSS and Go heap retain their separate meanings.

Default storage is `<state directory>/flow-pieces/<hash>/<piece number>`. With
`UseDisk` and `TorrentsSavePath`, preparation initially shares that existing store.
The preparation manifest retains its chosen root across settings changes; new
jobs in that manager share it. Changing the disk cache path does not move prepared
data. Remove all jobs and restart before choosing a new preparation root. The
server migrates ordinary cached pieces to the retained root when starting a new
job; migration uses bounded scratch space and preserves aliases of the same file.
The private local `flow-preparation.json` contains original metadata, authorized
trackers and configured sources; it should not be shared as a diagnostic log.

Restart recovery verifies exact lengths and metadata SHA-1 hashes before trusting
piece files. Truncated, oversized, corrupt, zero-filled and symlink piece entries
do not establish availability. Native and Go readers share that verified result.
Full-file readiness also requires disk sync. The normal `UseDisk` cache alone is
still an evicting cache, not an offline archive.

## HTTP mirrors

Imported `.torrent` web seeds use libtorrent's native HTTP handling. **HTTP
mirrors** in the file dialog provides explicit URL addition and disabling.
Native HTTP Range, piece hashing, retries and peer fallback remain in the same
engine and cache. A matching title or filename is insufficient: every piece must
match the torrent's exact bytes. Torrents with mirrors use hash-verified reads
for their handle lifetime, including after disabling an in-flight mirror.
Ordinary torrents retain responsive block reads.

Limits: 16 configured/imported sources, four simultaneous native web connections,
two pipelined requests, 15-second inactivity timeout, 30-second retry delay and
60-second DNS retry interval. Native corruption handling and remote retry advice
also apply. A broken mirror does not disable ordinary peer discovery.
Native corruption bans can affect a peer sharing the corrupt mirror's IP;
use an independent healthy source when diagnosing that case.

Additional sources are rejected for private torrents. Canonical imported sources
remain governed by metadata and destination restrictions. Default destinations
must resolve to global addresses; literal IPs, resolved endpoints, cached addresses
and redirects are checked natively. Proxy-hidden destinations that cannot be
checked fail closed. Local access requires explicit **Approve my LAN mirror**
and a literal private/loopback IP, with no query arguments or URL credentials.
Cloud link-local destinations are prohibited.

User URLs require HTTP(S), no credentials/fragments/control characters and at
most 8192 characters. Public signed queries may be supplied but are stored only
in private local state. The UI/API exposes an opaque source ID and origin only;
paths and query strings are omitted, and native web-seed alerts redact URLs.
An origin is labelled **configured; availability unverified**, not healthy.
Disabling an imported source persists a tombstone so it does not return at restart.
Portable backup/support exports omit configured source URLs and peer hints.

## Reachability and useful peers

New default settings use BitTorrent port **51413**; saved and explicit settings
and CLI overrides remain authoritative. Port `0` still selects an automatic port.
The dashboard reports observed TCP/UDP ports separately from HTTP 8090, native
mapping successes/errors, listener errors and incoming TCP/uTP/IPv6 counts.
A successful router mapping is evidence of that mapping, not proof of end-to-end
reachability. Check private-network TCP/UDP firewall access, router forwarding,
CGNAT and available IPv6 routes. Dashboard exposure is unnecessary.

Keep TCP and uTP usable where supported. Native hole punching requires compatible
peers, a rendezvous peer and compatible NAT behavior. A forwarding-capable VPN
is an optional route workaround, not a free-source service or universal fix.
Uploads remain under your configured policy; avoid saturating the uplink and
delaying acknowledgements. Client identity and upload limits are not overridden.

`Flow.PeerResumeHints` is disabled by default. When enabled, recently downloading
public BitTorrent peers can be retained through the native peer-list resume source.
At most 32 endpoints per torrent, 128 private files and 8192 bytes per file are
allowed. Hints expire ten minutes after observation, require the same local
network fingerprint, and are injected once per new known-public handle after
250–2000 ms jitter. Native failure counters and reconnect backoff own retries.
Parole peers, web seeds, unknown privacy and private torrents are excluded.
Read-only mode prevents saving hints. `flow-peer-hints` contains peer addresses
and is separate from diagnostic history; do not share it. Disable the setting
to stop using it, and remove that directory while the server is stopped to erase
retained identities immediately. Expired files are pruned during later writes.

Private metadata preserves canonical tracker tiers, rejects public tracker
replacement/injection and additional peer restoration, announces through one
working tier, and clears previous discovery hints/connections on a private tracker
switch or when a magnet is established as private. Public injection is deferred
until metadata establishes public status. Tracker intervals remain native.

## Scheduling and profile comparisons

The dependency is pinned to libtorrent v2.1.2 revision
`6da363d2994f17c0b3c0450d124cf73a31a73847`, plus reviewed upstream queue-unit fix
`94bffc25272b6833108d601fc0d824c344c48bc3`. Patch recipes invalidate dependency
caches and fail on unexpected source drift.

Future deadlines use piece size and qualified media/consumption rates. The
blocked piece stays at deadline zero, with ordinary forward work and existing
container/probe reservations. Independent readers and obsolete seek work remain
reconciled. Native scheduling may hedge requests; strict completion order or zero
duplicates is not promised. Delivery variation and real blocked outages can grow
buffers within the existing cache/time caps and hysteresis; full-buffer idle does
not count as a delivery deficit.

`Flow.ScarcePieceHints` is an optional experiment, disabled by default. With a
qualified rate, at least five delivery samples showing variation, and a fresh
non-truncated availability sample, one of the next four pieces with one observed
supplier is advanced by at most one second. It adds no extra window or priority
and cannot overtake deadline zero. Public-swarm benefit is not established.

Legacy remains the default. Conservative restores native swarm timer defaults;
its longer reconnect backoff can exceed a single HTTP reader's 60-second wait
after a supplier disappears. Preparation keeps waiting independently. Balanced
changes connection initiation, not that backoff. Custom permits deliberate timer
comparisons. Do not rank profiles by their names or connection count.

The controlled harnesses cover partial suppliers, late HAVE/metadata, choking,
outages, rare suppliers, byte-checked seeks, preparation crash/offline/corruption/
quota recovery and mirror fallback. They own local peers and isolated processes.
Use the ledger for exact passing runs and remaining checks. Local timing is not
a public-swarm benchmark, physical NAT/IPv6 test, multi-hour endurance result or
Android decoded-frame measurement. Paid providers are excluded by the user's
free-only preference. Pure-v2 storage, QUIC, application DoH, AI prediction,
transcoding and another engine are not justified additions to this sparse-source work.
