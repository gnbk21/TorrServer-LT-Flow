# TorrServer-Flow 1.1.7

Version: `MatriX.145.Flow-v1.1.7`. Security update for Flow 1.1.6.

## Security fixes

- Enforce same-origin or explicitly approved browser management requests in both
  legacy security profiles. Reject DNS-rebinding hostnames and cross-origin
  management navigations. Source imports and saving through legacy playback
  endpoints receive the same browser protection.
- Fail closed when HTTP authentication is enabled but the accounts file is
  missing, unreadable or malformed. Compare credentials in constant time.
- Validate outbound integration destinations and pin approved DNS answers when
  dialing. Recheck redirects, reject HTTPS downgrade and prohibit link-local metadata
  and selected reserved destinations. Environment proxies cannot
  bypass these checks. Explicit local service access is scoped to its configured
  scheme, host and port; public DNS names do not gain private-network access.
- Confine disk-cache I/O, resume verification, migration and cleanup beneath
  opened filesystem roots. Check actual piece descriptors for regular-file and
  single-hard-link ownership before truncation. Preserve bounded descriptor
  pooling, cache integrity and torrent removal/re-add synchronization.
- Explain the browser-origin setting in all seven interface languages.

## Updating and compatibility

Use the platform ZIP or existing managed updater, preserving your state
directory, port and authentication options. Keep the previous executable and a
private state backup for rollback. Public release assets contain no personal
settings or credentials. Standard builds do not require GStreamer.

**For web-based Lampa**, add its exact trusted `http(s)://host[:port]` origin in
**Settings → Management origins** from the local dashboard. Native clients
without browser headers keep their existing authentication path. Switching the
legacy security profile no longer grants arbitrary websites management access.

**For local indexers or mirrors**, configure a private IP address or `localhost`.
Other DNS names resolving to private addresses do not receive a LAN grant.
Local artwork requires its configured TMDB image origin. URL userinfo credentials
and environment-proxy routing are no longer supported for these fetches; use the
integration's existing API-key settings.

**If HTTP authentication is enabled**, a valid `accs.db` is required; invalid or
missing account storage now denies access. Browser checks do not replace
authentication. With authentication disabled, reachable originless native
management clients remain trusted administrators.

## Verification and limits

The accepted implementation passed all eight platform builds, the full native
and browser suites, six parallel-streaming cases, ten 90–120 Mb/s comparison
jobs and 15 isolated Windows API checks. The tagged release rebuilds every
platform and reruns native, functional, Windows lifecycle, inventory, packaging
and provenance gates. Earlier UI validation and full comparisons are reused
only after exact-source and successful-run checks; they are not claimed as
new comparisons of the tag. See the
[verification record](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/RELEASE_1_1_7_VERIFICATION.md).

The PR scan reported six results; full release-source analysis reports 21.
The owner explicitly approved six initial and twelve additional dispositions;
three pre-existing dispositions remain. These retain guarded administrator
capabilities and recognize existing permission/MIME safeguards. See the
[full-branch review](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/docs/security/SECURITY_RELEASE_1_1_7_REVIEW.md)
and [initial review](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/docs/security/SECURITY_DISPOSITIONS_PR10.md).
Trusted management access remains required. Accepted findings and successful
scans are not proof that Flow is vulnerability-free.

Legacy remains the default; libtorrent 2.1.2, original media bytes, existing
profiles and optional streaming controls are retained. Fresh representative
phone/public-swarm playback, physical Windows recovery and exact-final
multi-hour acceptance remain unverified. No further two-hour test was run.
Windows executables are not Authenticode signed.
