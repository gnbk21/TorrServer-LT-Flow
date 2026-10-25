# Code scanning review — 25 October 2026

## Scope and result

Reviewed `develop` at `62375927f08239675f2ec34f8a973b9d64317446`, the current
GitHub alert inventory, and native CodeQL analysis `2021682724` including its
SARIF data-flow traces. Runtime source is identical to the published 1.1.6 tag
source `cd8f1cceef7f11f9e87200be3426abc54fb33d9c`.

There are **30 open alerts: seven critical network-request findings, 22 high
path findings, and one high reflected-XSS finding**. This is not a clean security
bill of health. A successful CodeQL workflow means analysis completed; it does
not mean its findings were resolved.

The inventory changed even though its total did not: GitHub marked JacRed #49
and Torznab #54 fixed and opened #90 and #91 on 25 October at the corresponding
search requests. The new traces flow from `/settings` and backup restoration
through the settings snapshot to those configured destinations. This is not
evidence of two new search-term-to-host injection bugs. It also does not justify
automatically dismissing the replacement findings.

No alerts were dismissed, no scan rules were suppressed, and the running server,
personal configuration and release assets were not changed by this review.

## Security boundary that matters

Management clients are administrators of this process. They can configure
indexers, tracker-list sources, TMDB mirrors, certificate paths, disk cache roots
and MSX shared directories. Those operations intentionally access the server's
network and filesystem using the server account's privileges. They are unsafe
capabilities to give to an untrusted client, even when requests have timeouts or
response-size limits. Those limits prevent some resource abuse, not SSRF.

`auth.CheckAuth()` protects management routes **only when HTTP authentication is
enabled**. Disabling authentication is an explicit trusted-network operating
mode, not a separate authorization mechanism. `SearchWA` can expose search of
already configured indexers without exposing the endpoint-test handlers.
Legacy playback has its own compatibility and capability-token policies.

The compatible management-origin policy is permissive unless the restricted
profile or explicit `ManagementOrigins` is selected; the web server also permits
cross-origin API requests. Consequently, an unauthenticated installation can be
at risk from an untrusted reachable client or a website able to reach its LAN
API. Browser private-network protections must not be treated as Flow's access
control. The restricted origin policy helps browser-origin requests but still
allows originless native clients, so it does not replace authentication.

Portable backup imports do **not** accept indexer URLs, tracker URLs, certificate
paths or cache paths. They merge a bounded allowlist into existing host settings;
security-sensitive Flow fields are preserved. Several CodeQL backup-origin
traces conservatively taint the resulting complete settings object. The
`/settings` administrator path can actually change the corresponding values.

## Reconciliation of every open alert

Locations below are the current GitHub instances. “Owner capability” means a
real privileged operation remains, with the trust boundary described above;
it is not a claim that the operation is safe for public access.

| Alert | Location | Review |
| --- | --- | --- |
| #91 | `server/torznab/torznab.go:247` | Owner-configured indexer search. The host comes from `TorznabUrls`; query, categories and API key use encoded query values. Selecting an index chooses a saved entry. Timeout, response bound, concurrency limit and interface policy exist; private destinations and redirects are not prohibited. |
| #90 | `server/jacred/jacred.go:132` | Owner-configured JacRed search. `searchURL` retains the configured authority and encodes search/key values. Same administrator trust and redirect/private-network exposure as #91. |
| #55 | `server/torznab/torznab.go:373` | Capabilities and endpoint testing intentionally contact saved or administrator-supplied endpoints. `/torznab/test` is in the authorized group; `SearchWA` can expose saved capabilities. LAN Jackett/Prowlarr compatibility is intentional. |
| #50 | `server/jacred/jacred.go:219` | Administrator endpoint test accepts a supplied host. Authorized route, 30-second deadline, shared request bound, 1 MiB response bound and selected-interface transport; still a privileged network capability. |
| #53 | `server/torr/utils/webImageChecker.go:43` | Poster validation fetches an owner-supplied HTTP(S) URL. Credentials are rejected and decoding is bounded, but private addresses and redirect destinations are not filtered. The reported stream flow passes through the authenticated save path when HTTP auth is enabled. This is not a general public-only fetcher. |
| #52 | `server/torr/utils/torrent.go:231` | Owner-selected tracker-list URL is fetched with deadline, size limit and selected-interface transport. No private-destination restriction; do not expose settings writes to untrusted clients. |
| #51 | `server/torr/poster.go:178` | Configured TMDB API/mirror is contacted with encoded title query and bounded response. A torrent title does not select the API host; changing the configured host is an administrator capability. |
| #89 | `server/torr/apihelper.go:301` | Cache deletion uses a configured root plus a canonical storage hash, with hybrid identity validation, storage-drain wait and re-add fence. The root is owner-selected. These checks are not a filesystem sandbox against an attacker who can change that root or its contents. |
| #87 | `server/web/sslcerts/sslcerts.go:207` | Configured certificate read is bounded by `MaxPEMSize` and parsed as certificates. Selecting certificate paths is an administrator capability. |
| #86 | `server/web/msx/msx.go:233` | Authorized `/files` changes the shared-directory symlink to an explicitly selected existing directory. This deliberately grants file sharing of that directory. |
| #85 | `server/web/msx/msx.go:220` | `os.Stat` checks that same explicit directory before replacing the sharing entry. Invalid targets preserve the previous entry. It does not confine the owner to a fixed media root. |
| #80 | `server/torr/storage/torrstor/preparation.go:88` | Preparation migration compares the prior disk backend with a path under the configured root plus canonical hash. Owner-selected paths and directory aliases are supported. |
| #79 | `server/torr/storage/torrstor/preparation.go:63` | Preparation creates the configured root/hash directory. The hash is encoded from the binary identity; root authority remains with the administrator. |
| #78 | `server/torr/storage/torrstor/disk_handles.go:74` | Piece write creates parent directories for internally generated piece paths. Arbitrary torrent member names are not used as piece filenames. A malicious writable cache directory is outside the current protection boundary. |
| #76 | `server/server.go:85` | Startup enumerates the configured disk cache root. This reads directory entries rather than accepting a public request filename. Root selection is privileged. |
| #74 | `server/web/sslcerts/sslcerts.go:186` | Certificate ownership inspection stats the configured certificate. Ownership is also checked by certificate content before generated-pair replacement. |
| #73 | `server/web/sslcerts/sslcerts.go:166` | Key existence check for generated-pair ownership. A lone key is retained rather than overwritten. |
| #72 | `server/web/sslcerts/sslcerts.go:165` | Certificate existence check for the same ownership decision. Custom paths remain privileged configuration. |
| #71 | `server/web/ssl_api.go:102` | Authorized certificate download reads the configured certificate with a bound and returns only `CERTIFICATE` PEM blocks; it does not return private-key blocks or arbitrary raw file bytes. |
| #68 | `server/torr/storage/torrstor/diskpiece.go:33` | Disk resume stats a configured-root/canonical-hash/numeric-piece path and checks recorded completion and exact expected size. Resume integrity checks occur separately; stat/size alone is not authentication. |
| #67 | `server/torr/storage/torrstor/disk_handles.go:81` | File descriptor pool opens internally generated piece paths. It bounds descriptors and synchronizes eviction. It does not use a confined/no-follow filesystem root. |
| #66 | `server/torr/preparation.go:515` | Free-space lookup walks existing ancestors of the owner-selected preparation root. It does not read an arbitrary requested file's contents. |
| #63 | `server/server.go:132` | Startup cleanup removes the selected hash directory after enumerating its contents. Configured cache roots must be dedicated and trusted. |
| #62 | `server/server.go:130` | Startup cleanup removes enumerated child entries. It is nonrecursive for those children, but does not protect against a locally substituted ancestor symlink. |
| #61 | `server/server.go:124` | Startup cleanup reads a hash-named directory under the configured root. Hex validation prevents request hash traversal; the filesystem root and its directory entries must remain trusted. |
| #60 | `server/flow/resume.go:31` | Resume opens numeric piece filenames. Lstat, regular-file/size checks, same-file descriptor checks and SHA-1 verification precede advertising completion. |
| #59 | `server/flow/resume.go:26` | Same resume path's Lstat rejects individual symlink/nonregular pieces before opening. Ancestor directory substitution is a separate filesystem-hardening concern. |
| #58 | `server/flow/resume.go:19` | Resume checks the caller-selected cache directory exists. It is an owner cache path, not a public arbitrary-file response endpoint. |
| #57 | `server/diagnostics/private_file.go:11` | Secret file permission restriction operates on internal state/certificate/backup paths. Atomic secret writes create a protected temporary file before storing credentials. This does not grant untrusted clients a general chmod API. |
| #56 | `server/flow/response.go:49` | Generic response recorder forwards stream bytes. Its production caller `Torrent.Stream` installs explicit media/subtitle MIME types, `nosniff`, and binary attachment fallback for active/unknown formats before `ServeContent`. The recorder itself is not an HTML-rendering API. |

The native SARIF contains 36 results; six correspond to previously dismissed
instances on this branch. These were not counted as newly open alerts or
re-dismissed. Previous review approval does not authorize new dispositions.

## Remaining hardening priorities

1. **Separate management exposure from player compatibility.** Prefer an
   authenticated management listener/origin policy while retaining narrowly
   scoped playback access for Lampa/Just Player. Verify native-client behavior
   before promoting stricter defaults. Do not silently alter the owner's setup.
2. **Give outbound integrations explicit destination permissions.** Keep saved
   LAN indexers usable, but constrain public-only artwork/metadata fetches and
   validate redirects and DNS results at dial time. Merely parsing an HTTP URL
   or using the selected-interface transport does not prevent SSRF. Extending
   the existing `netpolicy.DestinationTransport` is preferable to duplicating
   its policy; LAN permissions need separate handling for intentional local
   services. Test environment-proxy behavior as part of this change.
3. **Constrain cache I/O relative to an opened filesystem root.** Canonical
   hashes and numeric pieces prevent string traversal, but ordinary `OpenFile`
   and startup enumeration can follow locally substituted directories. Cover
   Windows reparse points, Unix symlinks, migration aliases, deletion and
   concurrent replacement before claiming confinement. Keep explicit MSX
   sharing and owner-selected certificate paths as separate capabilities.

These are recommendations, not completed changes. Do not close the warnings
merely to make the dashboard green. Any later disposition should identify the
specific alert, trust assumptions, test evidence and remaining exposure.

## Verification and limits

- Fresh local `go test ./flow ./mimetype ./diagnostics -count=1 -timeout=120s`
  passed with isolated temporary state and `CGO_ENABLED=0`.
- An initial broader local invocation could not compile native-dependent
  packages with CGO disabled. No native test success is inferred from that
  failed invocation. This Windows environment lacks the required native build
  toolchain; no stub was substituted.
- Exact-commit [CodeQL run 40177696437](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40177696437)
  passed its actual native protocol/cache regression step, parser sanitizer and
  fuzz checks, Go/C++ extraction and scan, Windows guard scan, JavaScript/TS and
  Actions scans, and browser checks. These are existing CI results inspected
  during this review, not new local native runs.
- Reviewed the existing source gates for credential-free `/play` and `/stream`,
  configured search destinations, backup-field preservation, poster save flow,
  canonical removal paths, preparation state validation, certificate download,
  MSX sharing and stream MIME policy. No authentication bypass was established
  by these traces. This is not an exhaustive penetration test or a claim that
  every reachable network path is safe.
- No live exploit probes, playback interruptions, configuration changes or
  independent verification of the owner's firewall/exposure were performed.

Rule context: [CodeQL request forgery](https://codeql.github.com/codeql-query-help/go/go-request-forgery/)
and [CodeQL path injection](https://codeql.github.com/codeql-query-help/go/go-path-injection/).

Related earlier reasoning: `RELEASE_1_1_SECURITY_REVIEW.md` and
`RELIABILITY_USABILITY_SECURITY_REVIEW.md`. This dated inventory supersedes any
inference that an unchanged alert count means unchanged alert identities.
