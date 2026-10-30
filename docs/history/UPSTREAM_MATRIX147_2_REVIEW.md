# YouROK MatriX.147.2 integration

Reviewed and adapted 30 October 2026. Upstream comparison:
`d84f41968780744d4b8643f7a73b22a84d2c3779` to
`8a69eadd28e6d243cea62ab6e37dbf55eb4eb77a` (10 commits, 15 files).
Flow baseline: `3d5fdb25bb167c24ddfe253d20bec07575a29195`.

Sources: [release](https://github.com/YouROK/TorrServer/releases/tag/MatriX.147.2),
[complete comparison](https://github.com/YouROK/TorrServer/compare/MatriX.147.1...MatriX.147.2).

## Requirement reconciliation

| Upstream fix | Flow disposition |
| --- | --- |
| #935 M3U external tracks require the full video name | Adapted Unicode-aware boundaries at both ends, including combining marks. Only audio/subtitle sidecars are attached; another video, poster, text file or playlist cannot become an external track. Parent-directory naming remains supported. |
| #938 Playlist ETag follows its content | Adapted using SHA-256 of the final playlist, hash and name. Conditional GET returns 304 only for the matching representation. Per-fetch playback session tokens remain independent, so otherwise identical newly generated playlists can still have different ETags. |
| #933 Duplicate torrent folders in WebDAV/FUSE | Adapted after category/active filtering. Unique sanitized names retain their paths. Duplicate names receive a hash prefix that expands past shared prefixes and literal-title collisions; a full-hash collision with a literal title receives a checked numeric suffix. Full v1/v2 identities are retained. Names are deterministic for a given visible set; adding/removing duplicates can change suffixed paths. |
| #944 Zoned link-local IPv6 clients | netip parses scoped IPv6 and exposes the numeric IP to existing ACLs. IPv4 with a bogus zone and malformed addresses remain rejected; forwarded headers do not select the ACL identity. |
| #943 Internationalized referer rules | Configured rules and incoming Referer/Origin hosts use the same IDNA lookup normalization. DNS label/hostname limits apply after conversion. Unicode and punycode rules deduplicate. The separate management-origin authorization and saved policy values are unchanged. |
| #942 Missing settings object | Already rejected by the settings endpoint before configuration application; its existing regression covers missing/null sets. |
| #941 Malformed hash panic | Flow uses its non-panicking InfoHash parser and retains full 40/64-character identities. No SHA-1-only upstream parser was imported. |
| #934 Services after settings reset | Flow already applies resets through its revision-aware configuration manager and integration callback after a successful restart-requiring change. Idle scheduling and rollback remain intact. |
| #937 Automatic subtitle matching | The upstream automatic subtitle module is absent from Flow. Its matching requirement remains relevant to a future separately integrated subtitle feature. Current M3U matching is fixed without claiming new GST/player subtitle support. |

No engine, scheduler, video quality, profile defaults or personal configuration
changes are part of this integration. This maintenance release supplies no
evidence of higher torrent throughput or a remedy for sustained supply deficits.

## Verification

- Before implementation, seven isolated source-derived checks reproduced the
  playlist, episode-name, sanitized-folder, IPv6 and IDN problems, confirmed the
  existing non-panicking v2-aware hash parser, and demonstrated that upstream's
  eight-hex suffix alone cannot guarantee unique paths.
- After implementation, the actual referer/parser source and its tests passed
  locally with Go 1.26.10, including IDN equivalence and invalid-host cases.
- New full-package regressions cover conditional playlist responses, separate
  fetch sessions, episode/Unicode/media-type boundaries, expanded hash prefixes,
  literal folder collisions, v1/v2 names and scoped IPv6 whitelist/blacklist rules.
- The native test, vet and race package lists now include `web/waf` and `torrfs`.
  Complete native CI results will be recorded after the candidate run finishes.

Local full-package compilation requires a C/C++ toolchain and patched libtorrent;
neither is available in this Windows checkout. No owner's server or settings were
used for the tests. Phone playback and an upstream executable benchmark are not
claimed by these maintenance checks.
