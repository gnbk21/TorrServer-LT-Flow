# TorrServer-Flow 1.1.6

Version: `MatriX.145.Flow-v1.1.6`.

## Streaming and connections

- Add two independent experiments in **Settings → Flow**: **Frontier recovery**
  requests a late required block from another eligible peer within strict
  duplicate budgets; **Adaptive peer requests** adjusts each qualified peer's
  request credit using observed service and local/remote limits. Both default
  to **off**, automatically require the existing capacity guard when enabled,
  and use the existing idle/restart configuration path.
- Rank fresh opt-in public peer hints by recent useful service and restore them
  in bounded staggered batches. Preserve private-torrent, interface and native
  TCP/uTP policy.
- Show startup phases, block service distributions, duplicate cost and recovery
  activity in diagnostics, support exports and opt-in history. Service time
  includes remote queues; it is not network RTT. Unknown observations remain
  unavailable rather than being presented as zero.

These changes refine the existing BitTorrent scheduler; they do not introduce
QUIC, HTTP/3 or a new peer protocol. Controlled tests show benefits in several
constrained cases and regressions in others. Neither experiment is promoted to
a default or guaranteed to increase public-swarm throughput.

## Reliability and web interface

- Return bounded partial search results with source outcomes, explicit retry,
  cancellation and exact torrent identity deduplication across Torznab/JacRed.
- Add opt-in anonymous recent-session summaries, guided troubleshooting and
  freshness handling that does not depend on the phone's clock matching the PC.
- Serialize settings/viewed storage migrations under an idle maintenance lease,
  with bounded recovery journaling, revision checks and verified rollback.
- Scope update backups to the versioned Flow state inventory, preserve unrelated
  shared-directory files, and bind required provenance/runtime checks to source.
- Honor management Retry-After without automatically replaying mutations;
  rotate logs with bounded archives while preserving native log descriptors.
- Integrate the new controls and evidence throughout all seven UI languages.

## Verification, platforms and updating

The release rebuilds the exact tagged source for Windows, Linux, Android and both
macOS architectures. Publication requires actual successful platform, native,
functional, Windows lifecycle, inventory, packaging and provenance gates.
Earlier source-identical full validation is reused only after receipt and
component checks: it includes 72 new matched streaming trials, ten existing
high-bitrate comparison jobs, independent viewers, late cancellation and
byte-exact jitter/loss/reordering/outage recovery. Those broad comparisons are
not claimed to run again on the tag. See
[verification](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/RELEASE_1_1_6_VERIFICATION.md).

Extract the platform ZIP and use the same state directory, port and authentication
options as before. Stop the previous server first; keep its executable and a
state backup for rollback. Existing settings are preserved. The optional GST
build still needs GStreamer libraries; standard builds do not.

Legacy remains the default; libtorrent 2.1.2, original media bytes and the
existing cache architecture are retained. Fresh phone/public-swarm playback,
physical Windows recovery and exact-final multi-hour acceptance remain field
checks. No further two-hour test was run. Existing reviewed security findings
remain disclosed; successful scans are not a zero-alert claim. Windows binaries
are not Authenticode signed. Personal settings and credentials are not included
in public release assets.
