# Automatic magnet metadata recovery

Flow first reuses verified metadata in its existing saved torrent library. A new
magnet normally obtains metadata from peers. If it is still missing after
15 seconds, recovery can obtain the same torrent's metadata over HTTPS without
replacing its native handle or changing the release.

## Sources and controls

- **Automatic magnet metadata recovery** (`Flow.MetadataFallback`, on by default)
  tries two distinct HTTPS `.torrent` URLs supplied in the magnet's `xs` and `as`
  parameters, in that order. URLs are retained with saved torrent specs.
- **Free public metadata caches** (`Flow.PublicMetadataCache`, off by default)
  additionally tries Torrage and iTorrents for stalled **v1** magnets. Enable this
  in Settings > Flow to permit disclosure of info-hashes to those services.
  No account, subscription or video download service is involved. Flow does not
  search by title or substitute another release.
- Public lookup is unavailable for v2-only magnets; supplied URLs support full
  v1/v2/hybrid verification. Known metainfo bypasses recovery entirely. For an
  unresolved magnet the private flag is unknown until metadata arrives; owners
  of private magnets should leave public lookup disabled. Supplied URLs are
  tried first, and success stops further lookups.
- Both controls apply to subsequent attempts without an engine restart. Public
  lookup permission is host-specific and excluded from portable backups. Existing
  installations do not acquire public lookup consent through migration/import.

Endpoints: [Torrage's documented hash lookup](https://torrage.info/) and
`https://itorrents.net/torrent/INFO_HASH.torrent`. These independent services may
miss a torrent or be unavailable. A TLS failure, challenge page or incorrect
response is never successful recovery. Flow does not upload torrent files.

## Bounds and correctness

Peer metadata retrieval continues during recovery. Each torrent owns one
operation; multiple waiting clients cannot multiply it. There are at most four
sources, each limited to eight seconds and a 16 MiB decoded response. Downloads
use the existing coalescing mechanism with a global limit of four concurrent
fetches. Peer success, removal and shutdown cancel outstanding work. Recovery
fits inside the existing metadata wait; it does not increase disconnect settings
or retry indefinitely.

Only HTTPS URLs without URL credentials/fragments are accepted. Certificates
remain verified. Redirects are bounded/revalidated. The existing destination
transport checks/pins DNS answers, blocks local/private/reserved destinations and
honors the selected torrent interface without an environment-proxy bypass.
BitTorrent transport encryption is unchanged.
Redirects to HTTP are refused, and source URLs are never sent in a Referer header.

Go and native code verify the payload identity; the native bridge checks every
known v1/v2 hash before installing the info dictionary. Outer tracker lists/web
seeds in the response are ignored, retaining original authorized tracker tiers.
Hash mismatch, HTML, empty/oversized bodies leave the magnet intact. Success uses
the ordinary metadata-ready path: lazy streaming priorities, file lists, waiting
clients and saved-library backfill all retain their existing mechanisms.

## Diagnostics and limits

The startup drawer reports waiting, HTTPS attempts, verified recovery, peer
success or exhausted sources. Retained `metadata_recovery` events and the incident
timeline use enum states and attempt counts, without source URLs, hashes, names,
arbitrary errors or peer addresses. The metadata-stage elapsed clock runs while
waiting. Recovery evidence remains visible through subsequent preloading.

Retained event `operation` identifies supplied HTTPS sources (1) or public caches
(2). Its `code` is a fixed classification: network failure (1), HTTP status/size
(2), invalid body (3), invalid metainfo (4), hash mismatch (5), native rejection
(6), concurrent-fetch capacity (7), verified identity (8), refused redirect (9).
The live startup response also exposes the corresponding symbolic code. Source
URLs and arbitrary upstream error strings are never retained.

Metadata removes the file-list bottleneck. Video still comes from peers or
separately configured web seeds. This feature cannot invent seeders or increase
their upload capacity. Public-cache availability and the user's actual release
need field validation separately from controlled fixture tests.

See the [verification record](METADATA_RECOVERY_VERIFICATION.md) for the tested
Windows executable, actual cache/restart/UI results and remaining field limits.
