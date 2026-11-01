# Security and supported versions

**Use the current stable Flow 1.2.0 release or newer.** Releases before 1.1.7 lack the authentication,
browser-origin, outbound-request and cache-confinement protections shipped in
[1.1.7](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.145.Flow-v1.1.7).
They are unsupported for new installations. See [the fixes and compatibility
notes](SECURITY_HARDENING.md) before updating an existing installation.

All earlier public releases were withdrawn to private drafts on 25 October
2026, and their old container versions were removed. Use the current maintained
stable release, [Flow 1.2.0](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.148.Flow-v1.2.0),
for new installations. See the [withdrawal record](docs/security/DISTRIBUTION_RETIREMENT_2026_10_25.md).

Only the current stable release receives security maintenance. Historical
source tags and engineering records are retained for auditability. Withdrawing
old downloads cannot remove copies already downloaded or third-party mirrors.

## Deployment boundaries

Keep Flow on a trusted LAN or VPN. Browser-origin checks are not authentication;
originless native clients retain their existing API access rules. Enable HTTP
authentication when the network is not fully trusted. Existing torrent playback
URLs can remain accessible to clients that know their identities. Do not expose
the listener directly to the Internet.

Administrator-selected integration endpoints, certificate paths, shared folders
and the cache root are privileged configuration. CodeQL severity labels include
these retained capabilities and conservative findings; they do not establish
that every flagged path is an exploitable vulnerability. The [recorded security
decisions](docs/security/SECURITY_RELEASE_1_1_7_REVIEW.md) document safeguards and
remaining risks. A passing scan is not proof that software is vulnerability-free.

## Reporting a vulnerability

Use the repository's **Security → Report a vulnerability** option if available.
Do not post credentials, private tracker URLs, diagnostic exports or exploit
details in a public issue. If private reporting is unavailable, open an issue
requesting a private reporting channel without sensitive details.
