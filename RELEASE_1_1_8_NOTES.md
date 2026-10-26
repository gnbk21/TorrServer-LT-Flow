# TorrServer-Flow 1.1.8

Version: `MatriX.145.Flow-v1.1.8`.

## Buffering and recovery

- Use fresh measurements of each peer's block service time when scheduling
  urgent requests. The optional adaptive request policy can grow a constrained
  pipeline while respecting both the peer's advertised capacity and Flow's
  local request cap. Stalled-peer detection uses the same fresh service evidence,
  with bounded tick slack and the configured timeout ceiling, so healthy delayed
  replies are not judged by near-zero spacing inside a batch. Disabled or stale
  models retain the existing scheduling and timeout paths.
- Evaluate delayed blocks independently. An uncertain request in a large piece
  no longer prevents recovery of another independently measured block. Existing
  duplicate-byte, concurrency and request-rate limits remain enforced.
- Account for recent supply deficits when selecting startup and read-ahead
  reserves, within the configured cache and buffer limits. Require enough
  observation time for the torrent's piece size before interpreting gaps in
  whole-piece verification as weak supply. Healthy and unknown supply retain
  the ordinary startup floor.
- Show the governing startup reserve, supply trend, qualified peer queues and
  current blocking piece in the dashboard. Stale or incomplete observations are
  presented as unknown. Server read-ahead is not the player's decoder buffer.
- Retry transient Windows executable sharing locks for up to five seconds
  during managed updates and rollback. Permission errors still fail immediately;
  persistent locks still fail the update safely.

## Settings and compatibility

Legacy remains the default swarm profile. Optional capacity-aware requests,
adaptive urgent horizons, frontier recovery, adaptive peer requests and container
burst hints remain opt-in for public installations. Existing private settings
are preserved when updating. Public downloads contain no personal configuration.

All 1.1.7 security boundaries remain in place. Web-based Lampa requires its exact
trusted origin in **Settings → Management origins**. Management clients must
remain trusted; do not expose an unauthenticated server to the Internet.

The libtorrent 2.1.2 engine, original media bytes, Range API and single shared
cache are retained. This update cannot create missing seeders or sustain a media
bitrate above the useful supply available from the swarm.

## Verification limits

Publication requires successful automated build, native, API, browser, Windows
lifecycle, controlled transport, inventory and packaging checks. The new
high-latency comparison retains stock-horizon control timeouts as failed
observations and requires adaptive candidates to complete; these controls are
not reported as successful playback trials.

Controlled transport tests use generated local bytes. Fresh representative
Android/Just Player playback, public-swarm comparisons, physical Windows
boot/sleep/network recovery and exact-final multi-hour acceptance remain
unverified. No further two-hour test was run, as requested. Windows binaries
are not Authenticode signed. Accepted security dispositions are not fixes or
proof that Flow is vulnerability-free.
