# YouROK MatriX.147 review

Reviewed 22 October 2026 against upstream tag `MatriX.147`, commit
`a86d8fa7fd`, and Flow development source `65954a39`. The release was published
16 October. This is a review of the release diff, not an upstream binary
benchmark or a claim that replacing Flow's engine improves streaming.

Sources: [release](https://github.com/YouROK/TorrServer/releases/tag/MatriX.147),
[complete comparison](https://github.com/YouROK/TorrServer/compare/MatriX.146.2...MatriX.147).

| Change | Flow assessment and action |
| --- | --- |
| Natural episode ordering, PR #913 | Applicable correctness fix: the common prefix can split a number, placing Episode 10 before Episode 1. Adapted the whole-number comparison and equal-number tie break, with a strict ordering regression. ASCII digit boundaries avoid treating individual UTF-8 bytes as Unicode digits. |
| Merged playlist file IDs, PR #914 | Flow already generates merged entries from `Torrent.Status()` and its saved file list, the same source used for streaming IDs. Upstream's separate anacrolix `statusFromSpec` implementation is absent. Keep this single source of IDs; the natural sort fix applies consistently to both paths. |
| Torznab API URL and User-Agent, PR #919 | Flow already normalizes `/api` and `/api/`, with Jackett/Prowlarr tests, cancellation, read bounds and selected-interface transport. Added the missing product/version User-Agent to both search and capabilities requests, with a local indexer regression. Do not replace the policy-aware client with upstream's default client. |
| Torznab hints/docs, PR #920 | Flow already has structured indexer entries and capabilities testing. Retain Flow's current UI/configuration names; upstream's legacy React hints are not compatible components. |
| External subtitles, PR #911 | Useful future feature: match sidecar SRT/ASS/SSA/VTT files in the torrent, convert to WebVTT, and expose tracks in GST HLS. Flow currently supports manual VTT upload and existing embedded tracks; it does **not** gain automatic torrent-sidecar matching from this upgrade. |
| Preload log spacing and migration adjustments | Preload only changes a console separator. Migration now skips existing target records even if their bytes differ; do not silently replace Flow's existing settings/library migration policy with that behavior. Neither change is a libtorrent deadline, request-capacity or high-bitrate scheduler fix. |
| README/contributors/version/generated legacy UI | Retain attribution; do not overwrite Flow's README, build identity or modern UI with upstream generated assets. |

## Subtitle integration boundary

The new upstream `ReadFile` takes an anacrolix file/reader and launches an
asynchronous bounded read. Flow uses libtorrent and generation-aware Go cache
readers. A direct cherry-pick would not compile or preserve its cancellation,
file boundaries, storage ownership and verification policies.

Upstream's GST loader synchronously loads external tracks while serving a
master playlist, uses a loopback HTTP request and retries failed tracks. A Flow
implementation should avoid adding subtitle wait time to video startup, carry
playback authorization through the internal boundary, use bounded memory/time,
and avoid logging titles or credential-bearing addresses. It also needs safe
text conversion, exact video/episode matching, language/forced-track handling,
reader cleanup, seven-locale UI and actual player compatibility checks. This is
a separate feature, outside the authorized October upgrade backlog; it was
reviewed, not silently marked implemented.

## Streaming conclusion

The reviewed diff contains no libtorrent upgrade or new high-bitrate peer
scheduler. MatriX.147 does not establish a remedy for missing suppliers or slow
blocks in an existing swarm. Flow retains libtorrent 2.1.2, its native patches,
original media bytes and current opt-in experiment defaults. Useful small
compatibility fixes above are incorporated without an engine replacement.
