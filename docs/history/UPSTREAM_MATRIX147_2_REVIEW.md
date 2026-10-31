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
  Native CI run `41070115297` passed its complete unit-test and vet steps for
  runtime source `b63256934cd82961169e8ab492469523f5575887`; additional comparison
  and race jobs were still running at this checkpoint. CodeQL run `41070114825`
  completed successfully. These results do not certify later source changes.
- Following the owner's request, installed the official MSYS2 UCRT64 compiler
  environment locally, verified the stable installer SHA-256 against its release
  metadata, built pinned OpenSSL 3.5.7, WebRTC dependencies and patched libtorrent
  2.1.2, and compiled both standard and GST Windows executables. GCC 16.2.0 and
  Go 1.26.10 were used. `build/windows-native.sh` and `Enter-FlowBuild.ps1` retain
  the setup without machine PATH or global Git configuration changes.
- All three complete local native packages (`web/api`, `web/waf`, `torrfs`)
  passed on Windows (90 tests/subtests passed; the existing optional contract
  export was not requested). Local `go vet` passed for all three packages.
  An initial sandbox run blocked four existing HTTP/certificate fixtures; the
  unrestricted isolated-state rerun passed without adding skips or changing
  assertions. Private JSON test evidence is retained under
  `.tools/upstream1472/windows-native-tests-host.jsonl`.

No owner's server or settings were used for the tests. Phone playback, actual
GST transcoding and an upstream executable benchmark are not claimed by these
maintenance checks. The locally built executables are development artifacts;
they were not installed over the owner's server or published as a release.
