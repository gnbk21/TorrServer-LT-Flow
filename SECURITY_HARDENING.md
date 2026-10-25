# Security boundary fixes

Implementation following [the 25 October review](SECURITY_REVIEW_2026_10_25.md).
That review describes the unmodified 1.1.6 source, not the protections added here.

## Changes

- Browser management requests require the dashboard origin or an exact entry in
  `Flow.ManagementOrigins` in both legacy security profiles. Cross-site browser
  navigations without Origin are rejected too. Browser management hostnames must
  be a literal IP, localhost, the computer's local name, or an explicitly approved
  origin host; this blocks an arbitrary DNS name rebinding to a LAN listener.
- Native clients without browser headers retain their HTTP authentication path.
  Ordinary identity-based playback/capability endpoints retain their access
  policy; importing a URL/local source or saving through a playback endpoint
  requires management-origin permission too. When HTTP
  authentication is enabled, a missing/unreadable/malformed accounts file now
  installs an empty, denying credential set instead of bypassing the playback
  source gate. Credential comparisons use constant-time comparison.
- JacRed, Torznab, tracker-list downloads, TMDB searches, torrent URL imports and
  poster validation use destination-checked clients. DNS answers are checked and
  pinned at dial time; environment proxies cannot bypass validation. Redirects
  are rechecked, HTTPS downgrade is refused, and cloud metadata, link-local,
  mapped private addresses and common IPv6 translation/transition ranges are
  blocked. A configured literal private/loopback endpoint or localhost grants
  access only to that original origin. It does not grant other local services.
  The selected torrent interface remains enforced for these integration clients.
- Poster URLs are public-only unless their origin exactly matches a configured
  TMDB image origin. Policy rejection is distinct from a temporary fetch failure.
- Cache piece reads, writes, stats, migration, resume verification, preparation
  removal, torrent deletion and startup cleanup use operations relative to an
  opened `os.Root`. A descendant symlink/reparse point cannot escape that root.
  Actual piece descriptors must be regular files with exactly one hard link;
  validation occurs before truncation. Existing descriptor pooling remains.
  Hash identities, piece layout, integrity verification, cancellation/drain
  fences and root-directory aliases are retained.
- CI runs the new filesystem tests on Windows and Linux and includes outbound,
  authentication, API, certificate and MSX regression packages in native checks.

## Compatibility and deployment

These changes are source changes, not an update to an already running binary.
The owner installation and its personal settings have not been modified.

For a trusted **web** Lampa client, add its exact `http(s)://host[:port]` origin
to Management origins from the local dashboard before using cross-origin
management. A native client with no browser-origin headers follows the existing
authentication rules. Arbitrary websites no longer receive management access by
selecting the compatible profile. The obsolete browser-profile selector is
removed; the serialized setting is retained for older API clients.

Use an explicit private IP or localhost for a local indexer/mirror. Other DNS
names resolving to private addresses do not receive an implicit LAN grant. If
artwork is hosted locally, explicitly configure its TMDB image origin. URL
userinfo credentials and environment-proxy routing are no longer accepted by
these fetches; configure integration API keys in their existing fields.

## Limits

Authentication-disabled installations still trust originless native clients that
can reach the management API. Browser checks are not authentication. Keep that
listener on a trusted network or enable HTTP authentication; these patches do
not silently invent credentials or change the owner's listener/firewall.

The owner-selected cache root and its parent remain trusted. Directory aliases
are allowed for that root; a privileged actor changing the selected root,
mounts or the process account is not contained. Hard-link checks describe the
descriptor at open time, not a sandbox against malicious same-account activity.
Explicit MSX shared directories and certificate paths remain administrator
capabilities rather than being silently confined to the torrent cache.

CodeQL can continue to report administrator-controlled URL/path flows through
custom policy code. No alert is automatically dismissed, no rule is disabled,
and a green scan alone must not be reported as zero findings.

## Verification

Windows local portable cache/resume/MIME/diagnostic tests and workflow lint have
passed during implementation. Native integration, browser and fresh CodeQL
results are required before this branch can be treated as ready to merge.

Design references: [Go os.Root](https://pkg.go.dev/os#Root) and
[OWASP SSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html).
