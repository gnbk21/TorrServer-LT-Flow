# PR 10 — six remaining security findings

## Review scope

Source branch: `bd4faf5b924bac345de2e0c48d13a8842214ec13`.
GitHub tested merge: `b9044a7612b53365e947c07ed1f2f5dcfc725fff`.
Native CodeQL analysis: `2021806975`, completed successfully in
[run 40181912977](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40181912977).

The native scan reports **five critical request-forgery findings and one high
path finding**. GitHub's separate CodeQL results check blocks this PR because
these are new alert instances. A successful analysis job is not a passing
results check. The default branch has not received this patch.

Preparing this review did not dismiss any finding. The owner explicitly approved
recording these exact six dispositions on **25 October 2026**. Prior approval
for seven different findings was not reused. The disposition below is an
explicit acceptance of retained administrator
capabilities, not a claim that all six operations were removed or made safe for
untrusted native clients.

## Implemented safeguards

- Browser management requires a same-origin request or an explicit exact origin.
  Rebinding hostnames and cross-site navigations are rejected. Native clients
  still require HTTP credentials when authentication is enabled. A missing or
  malformed accounts file no longer disables the playback source gate.
- `netpolicy.FetchClient` permits only HTTP(S) without userinfo or fragments.
  Resolver answers are checked and the approved literal IP is dialed. Environment
  proxies are disabled. Link-local/metadata, shared-address and selected reserved
  IPv6 transition ranges are blocked. HTTPS downgrade and unsafe redirects fail.
- LAN permission belongs only to an explicitly selected private/loopback literal
  or localhost at its configured scheme, host and port. Public DNS names do not
  gain LAN permission by resolving to a private address. A redirect does not
  acquire another local origin's permission. Integration requests continue to
  obey the selected torrent interface.
- Cache descendants use `os.Root` confinement. Actual piece handles must be
  regular files with one hard link before truncation. Resume hashes, drain/re-add
  fences, descriptor limits and owner-selected directory aliases remain intact.

These checks do not replace authentication. With authentication disabled,
originless clients able to reach the management listener remain trusted
administrators. The owner-selected cache root and its parent remain trusted.

## Exact proposed dispositions

For each row, the proposed GitHub reason is **won't fix**: retain the documented
privileged feature with the implemented guard, rather than remove that feature.
The code fixes remain; no scanner rule, severity or extraction setting changes.

| Alert | Operation | Reason and remaining risk |
| --- | --- | --- |
| [#92](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/92) | `server/jacred/jacred.go:135`, saved JacRed search | The authority comes from administrator settings; search terms are encoded query values. The fetch client enforces the destination and redirect policy above. Retaining local JacRed requires the explicit LAN-origin grant. An untrusted client with management access could choose a service, so management access must remain trusted/authenticated. |
| [#93](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/93) | `server/jacred/jacred.go:227`, administrator endpoint test | Testing an unsaved JacRed address is an intentional authorized management operation. It has the same network gate plus a 30-second deadline, 1 MiB response bound and shared concurrency limit. A trusted administrator can intentionally test a permitted local service; this is a retained privilege, not a public arbitrary-target proxy. |
| [#94](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/94) | `server/torr/poster.go:183`, configured TMDB API/mirror | Torrent titles select an encoded query, not the API authority. The administrator chooses the API/mirror, and the fetch client constrains its destination/redirects. Local mirrors remain possible via an explicit literal/localhost origin. A malicious administrator or compromised approved service is outside this protection boundary. |
| [#95](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/95) | `server/torznab/torznab.go:250`, saved Torznab search | Selecting an index chooses a saved entry; query/category/key use encoded parameters. The fetch gate checks actual dial addresses and scoped LAN permission. Local Jackett/Prowlarr remain supported. Settings writers are administrators and can intentionally select a local service. |
| [#96](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/96) | `server/torznab/torznab.go:381`, capabilities/endpoint test | Saved capabilities and testing an unsaved endpoint share the destination gate. Unsaved tests are authorized management operations; `SearchWA` exposes only search/capabilities of configured entries. Time/response/concurrency bounds remain. The explicit permission to contact a selected local indexer is retained. |
| [#97](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/97) | `server/cachefs/cachefs.go:18`, cache root creation | `MkdirAll` creates the administrator-selected root before it can be opened as an `os.Root`. Descendant I/O, cleanup and preparation resume are confined after opening. Selecting an arbitrary root is still privileged configuration; the root/parent and privileged mount changes are not sandboxed. Confining the root to itself before it exists cannot replace this administrator trust boundary. |

## Verification available for this decision

- Fresh native protocol/cache regressions, parser sanitizer checks, fuzz checks,
  and the expanded Go security regression packages passed in the exact-source
  CodeQL workflow. The real native bridge was extracted with a fresh Go cache.
- Windows/Linux portable branch checks passed. Local Windows cache confinement,
  integrity, MIME, diagnostic and real descriptor-pool tests passed.
- UI type checking, lint and formatting passed; 129 unit tests passed and one
  existing test was skipped. Browser acceptance passed in CI.
- The exact-run Windows executable was tested with disposable state and local
  ports: **15 checks passed**. These cover cross-origin/opaque/rebound-browser
  rejection, same-origin/approved-origin acceptance, ordinary playback origin
  handling, explicit local indexer access, redirect isolation, missing-account
  management denial, and a rejected playback source making no outbound request.
  Both owned test processes were stopped; personal settings were not accessed.
- Candidate executable SHA-256:
  `295e619d26348fae6e1df4fd678fde93310c1e9d6eb5860cfba9154d8af2cdb4`.
  Its version and VCS revision identify the tested GitHub merge above. Generated
  embedded assets make the normal build stamp `vcs.modified=true`; this is a CI
  candidate, not a separately published or reproducible-release claim.
- Broader platform/playback CI is still completing and must be checked before
  marking the PR ready. These security dispositions would not waive failed,
  cancelled or missing functional gates.

## Owner decision

The owner approved recording these **six specific** dispositions on GitHub.
Acceptance must not be reported as six additional vulnerability fixes or as
proof that Flow is vulnerability-free. Individual GitHub alert state must be
verified after recording the dispositions; the functional gates remain required.

See [implemented changes and compatibility](SECURITY_HARDENING.md) and the
[pre-patch 30-alert inventory](SECURITY_REVIEW_2026_10_25.md).
