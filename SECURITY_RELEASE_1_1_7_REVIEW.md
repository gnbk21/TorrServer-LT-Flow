# Flow 1.1.7 — additional full-branch findings

Reviewed 25 October 2026. Release source:
`87d1632e47d64adcfd98d7d916d80aebedbc4e8c`, tag `MatriX.145.Flow-v1.1.7`.
PR #10 is merged. This review initially held publication for the owner's
decision; the approved dispositions and resumed release are recorded below.
The installed server and personal settings are untouched.

## Why these findings appeared after the PR passed

The PR analysis was narrower than the full-branch analysis. Both used CodeQL
2.27.3 and the same security-extended suite, real Go/C++ extraction and 130 rules.
The PR action generated an augmented configuration for incremental analysis
(`exclude-from-incremental`) and reported six results. Its log explicitly says
file-coverage information is limited for pull requests. The full master scan
explicitly says it is **not** doing diff-informed analysis and reports 21 results.
This difference does not prove that a code change introduced 15 vulnerabilities.
It does mean that the PR's six-result/zero-open receipt cannot describe the
whole released branch.

- PR analysis: `2021864995`, tested merge
  `614a1dbdcd36f9a92a33c004e8450159acb3c910`,
  [run 40183731547](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40183731547).
- Full release-source analysis: `2022007836`,
  [run 40188062360](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40188062360).
  Its five jobs, expanded native regressions and analysis completed successfully.
  Successful execution does not resolve the findings.
- Of the 21 current results, six are the exact #92–#97 dispositions already
  approved, three have pre-existing dispositions (#1, #2 and #7), and **12 are
  open**: #52, #56, #57, #66, #71–#74, #85–#87 and #98.
- The full open set is **two critical network-request findings and ten high
  findings** (nine path findings and one reflected-XSS finding).

At the start of this review, only #92–#97 had been changed under the earlier
approval. That approval was not reused for the following 12. Their separate
owner decision and recorded states appear below.

## Exact proposed dispositions

Ten rows retain intentional administrator privileges with safeguards and use
GitHub's **won't fix** reason. Two use **false positive** because the production
call chain already constrains the operation that CodeQL flags. These are
documented decisions, not additional code fixes or a claim of zero vulnerabilities.

| Alert | Exact operation | Proposed reason, safeguards and remaining risk |
| --- | --- | --- |
| [#52](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/52) | `server/torr/utils/torrent.go:235`, tracker-list fetch | **Won't fix.** The URL comes from administrator tracker settings or pinned default mirrors. The new `FetchClient` validates/pins dial addresses, disables environment proxies and constrains redirects and explicit LAN grants. Deadline, bounded read and cached/local fallback remain. A trusted administrator can intentionally configure a permitted local tracker-list service. |
| [#98](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/98) | `server/torr/utils/webImageChecker.go:56`, artwork validation | **Won't fix.** Public candidates cannot acquire a LAN grant. Only an origin matching the administrator-configured TMDB image origin can use explicit literal/localhost LAN access. DNS/redirect policy, five-second deadline, 2 MiB header inspection and image-dimension bounds apply. Permanent policy rejection is distinguished from temporary errors. Retaining approved local artwork is an intentional administrator capability; unchanged artwork is not repeatedly fetched. |
| [#87](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/87) | `server/web/sslcerts/sslcerts.go:207`, certificate inspection | **Won't fix.** The administrator selects certificate paths. Leaf inspection reads at most `MaxPEMSize+1` (1 MiB plus the overflow byte), rejects oversize input and parses certificate PEM. It does not return arbitrary raw file bytes. Choosing a readable certificate path remains privileged. |
| [#74](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/74) | `server/web/sslcerts/sslcerts.go:186`, certificate ownership stat | **Won't fix.** Stats support a bounded ownership cache; a stat result alone does not grant replacement. Certificate content, self-signature and expected managed paths govern generated-pair replacement. Administrator-selected certificate paths remain supported. Same-account/privileged filesystem substitution is outside this protection boundary. |
| [#72](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/72) | `server/web/sslcerts/sslcerts.go:165`, managed certificate existence | **Won't fix.** This is an existence/ownership decision for the selected certificate pair. User certificates, invalid user files and a lone private key are preserved; the operation is not a public file-content read endpoint. Certificate location remains administrator configuration. |
| [#73](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/73) | `server/web/sslcerts/sslcerts.go:166`, managed key existence | **Won't fix.** The paired existence check prevents treating a lone user key as a newly owned generated pair. It does not serve key bytes. Explicit certificate/key configuration and the host filesystem remain trusted. |
| [#71](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/71) | `server/web/ssl_api.go:102`, certificate download | **Won't fix.** The management route reads the configured certificate with a size bound, extracts only `CERTIFICATE` PEM blocks and serves an attachment. Mixed PEM containing a private key does not return that key. This intentionally allows downloading the administrator-selected public certificate/chain; it does not restrict the administrator to one fixed path. |
| [#85](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/85) | `server/web/msx/msx.go:220`, shared-directory validation | **Won't fix.** Authorized `/files` deliberately chooses an existing directory for MSX sharing. Invalid or nondirectory selections return before replacing the existing entry. The ability to select a directory is retained; it must not be granted to an untrusted management client. |
| [#86](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/86) | `server/web/msx/msx.go:233`, shared-directory symlink | **Won't fix.** After validation, the explicit selected directory is exposed through the configured sharing entry. This is the actual file-sharing capability, separate from confined torrent-cache I/O. A trusted administrator can intentionally share sensitive files, so management access and the selected folder must remain trusted. |
| [#66](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/66) | `server/torr/preparation.go:516`, preparation free-space lookup | **Won't fix.** The function stats the configured preparation root or its existing ancestor to query disk capacity before creating a directory. It does not read or return arbitrary file contents. Preparation root selection remains an administrator privilege; descendant piece I/O/cleanup is separately confined. |
| [#57](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/57) | `server/diagnostics/private_file.go:11`, private-key permission restriction | **False positive.** The only production caller of `RestrictPrivateFile` is certificate setup, after `IsGenerated` checks the exact managed certificate/key paths under the CLI-selected state directory and generated-certificate ownership. A settings request cannot choose an arbitrary chmod target through this call. The function tightens permissions to 0600 and applies the Windows private ACL. This is not a general remote chmod API; malicious same-account filesystem changes remain outside the boundary. |
| [#56](https://github.com/gnbk21/TorrServer-LT-Flow/security/code-scanning/56) | `server/flow/response.go:49`, streaming observer write | **False positive.** Its only production constructor caller is `Torrent.Stream`. Before `ServeContent` writes through the observer, `SetStreamHeaders` chooses safe media/raster/subtitle types, sets `nosniff`, and forces HTML/SVG/unknown formats to `application/octet-stream` attachment. The observer forwards original bytes and Range behavior; it does not render them as HTML. This depends on keeping that production call chain/header policy intact. |

## Verification supporting this review

The exact release-source CodeQL workflow passed native protocol/cache checks,
sanitizer and fuzz checks and the expanded Go packages, including `netpolicy`,
`torr/utils`, `web`, `web/auth`, `web/msx`, `web/sslcerts` and API regressions.
The review inspected the actual full SARIF source-to-sink traces and production
callers rather than inferring safety from severity labels or the PR result count.

Relevant existing regression coverage includes:

- Explicit local destination grants, public-to-LAN rejection, redirect isolation,
  downgrade rejection and selected-interface enforcement.
- Tracker mirror fallback/timeouts/background refresh and poster image/type,
  timeout and interface behavior. A size limit is not treated as SSRF protection;
  destination validation is a separate enforced mechanism.
- Oversized certificate rejection; custom/invalid user certificate preservation;
  lone-key preservation; generated-key permission tightening; immutable/private
  uploads; stale-revision rejection and failed-change rollback.
- `TestCertificateAPIRequiresAuthRejectsStaleRevisionAndNeverDownloadsKey`,
  including a combined certificate/private-key PEM fixture.
- `TestInvalidDirectoryPreservesExistingFilesEntry` for MSX sharing.
- `TestUntrustedStreamContentAndRanges`, including HTML, SVG, extensionless and
  Windows-style names carrying script bytes, safe headers and byte-exact ranges.
- Actual recorder Range/progress/connection-reuse tests and Windows private-ACL
  tests in the accepted implementation's Windows portable run.

No live owner state or public swarm was probed. This is not an exhaustive
penetration test. Native management with HTTP authentication disabled still
trusts reachable originless clients; browser checks are not authentication.

## Publication hold and next action

Release run `40188065181` was cancelled before packaging/publication. All eight
platform builds and completed native/service/functional gates remain recorded;
the cancelled request group, combined evidence and publication must be rerun.
There is no 1.1.7 release yet, and the tag has not been moved or replaced.

If the owner approves these exact 12 dispositions, record each reason and a
specific comment, verify the saved alert state, and retry only the interrupted
release jobs while retaining successful earlier roles. The normal 25-role gate,
packaging, checksums, provenance, selected public downloads and container checks
still apply. Acceptance does not waive any required test.

## Owner decision and saved state

The owner explicitly approved these **12 specific dispositions** on
25 October 2026. Each alert's reviewed release-source instance, location and
open state were checked before mutation. Each saved dismissal reason/comment
was independently read back afterward. #56 and #57 were recorded as **false
positive**; the other ten were recorded as **won't fix**. No other alert was
changed under this approval.

The release's interrupted jobs were resumed using GitHub's failed-job retry,
retaining successful earlier roles. Publication and final public-download /
container checks still require their own successful receipts. Earlier six-alert
approval was not reused, no source/tag was moved, and no scan rule was suppressed.


## Completed publication

Flow 1.1.7 is now public and Latest. All 25 required exact-tag roles, selected
public download/provenance checks, 15 isolated security API checks and three
container architectures passed. The source/tag remained unchanged during
retry. See [the final release receipt](RELEASE_1_1_7_VERIFICATION.md) for exact
identities, actual job results and retained field/security limitations.
