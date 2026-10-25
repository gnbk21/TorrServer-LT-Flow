# October security evidence reconciliation

Publication follow-up: PR #6 and CI repair PR #7 are merged into stable
[Flow 1.1.4](../history/RELEASE_1_1_4_VERIFICATION.md). Older development identities and
pending-publication statements below are historical checkpoints. A passing
security scan does not mean retained reviewed findings have disappeared.

22 October 2026. GitHub API refresh on development source `65954a39` returned
30 open alerts: seven critical and 23 high. All latest instances identify
`refs/heads/develop`, source `26f22bb3`. These are retained findings, not 30
newly introduced defects in this upgrade. PR scan execution and final branch
analysis must be reported separately. No alert was dismissed by this work.

The seven owner-approved dispositions and the remaining retained capabilities
are documented in [the release security review](RELEASE_1_1_SECURITY_REVIEW.md).
That approval is not extended to new findings.

| Current alerts | Inspected boundary / regression evidence |
| --- | --- |
| 49, 50, 51, 53, 54, 55: HTTP sources | JacRed, posters, artwork and Torznab use the selected-interface transport. The new Torznab User-Agent preserves it. Owner-configured endpoints can intentionally be LAN services. Interface disappearance/cancellation tests remain required; this is not a public-only SSRF sandbox. |
| 52: remote tracker list | Owner-selected tracker-list URLs remain a management capability, with time/body bounds and interface policy. No automatic metadata fallback reuses that permissive fetcher. Metadata recovery instead uses its restricted destination/DNS/redirect client. |
| 56: response wrapper | `ResponseRecorder.Write` forwards original bytes; the caller supplies media/plain/binary Content-Type and nosniff behavior. Recorder does not render a supplied HTML template. Original-byte and MIME regressions remain required. |
| 57: private diagnostics | The filename is constructed by the bounded diagnostic writer under owner state, not from a media path. Private file permissions and bounded exports remain enforced. |
| 58–60: resume verification | Hex torrent identity, decimal bounded piece index, exact size/inode checks and symlink rejection remain in place. Corruption, holes, short-tail and v2 proof tests cover the data boundary. |
| 61–63, 76: server state/tool paths | Owner-selected state and local tool directories remain explicit capabilities. Do not reinterpret their existence as a general restricted-filesystem guarantee. |
| 66–68, 78–80, 89: preparation/cache/removal | Fixed hash bytes and numeric piece IDs construct storage names. Removal validates identity, drains owners and preserves data on failed cleanup. Existing migration, disk-handle and removal tests remain required. |
| 71–74, 87: TLS paths | Owner-selected certificate/key files remain supported; certificate generation/adoption/hot reload must preserve user files and permissions. Existing lone-key, custom-certificate and recovery regressions remain required. |
| 85, 86: MSX files | Authenticated static local files under the configured state directory are an intentional owner feature. Remote MSX requests separately use restricted destination/redirect checks. |

## New upgrade boundaries

- Native source and race checks on `69c78835` passed, but the reachable Go
  vulnerability gate correctly blocked that build with `GO-2026-6809`:
  `sslcerts.describe` reaches `crypto/x509.Certificate.Verify` in Go 1.26.9.
  The authoritative toolchain requirement is now Go **1.26.10**, the same-line
  security update. All CI targets read `server/go.mod`; the scan remains
  mandatory. [Official Go release history](https://go.dev/doc/devel/release#go1.26.0)
  records the 19 October certificate-verification security fix. Fresh native
  builds and reachable-vulnerability scans passed on `f780398f` and corrected
  test source `d482e8c0`; this finding was not dismissed or bypassed.

- Retained native errors contain a closed category/transport/family enum plus
  numeric code/operation/count. Signed mirror URLs and raw exceptions never
  enter the support/history payloads.
- Metadata requests remain consent-gated at launch and during execution, limited
  to two per torrent and four globally. Shared requests are separated by network
  policy; losers cancel. Native code verifies every known hash before installing
  metadata on the existing handle. A miss is not treated as a service outage.
- Player reports are manual local imports/exports, bounded in bytes/samples,
  strictly validated and not sent automatically. New monotonic sequencing is
  backward compatible with v1 reports.
- The upstream subtitle HTTP/reader feature was reviewed separately; no new
  unreviewed subtitle fetch endpoint was enabled.

This is source/alert reconciliation with automated boundary tests. It is not an
external penetration test, a zero-alert assertion, physical VPN leak acceptance
or proof of decoded phone playback. The exact current native scan/build results
are recorded in the [final October verification](../history/OCTOBER_UPGRADE_VERIFICATION.md).

Final refresh: all 30 retained open alerts still identify `develop` source
`26f22bb310a2bc6e9beef60216b8be7a33fdd401`. The four final PR analyses
(native Go/C++, Windows WFP, JavaScript/TypeScript and Actions) for merge revision
`1b8e6f6f995a84d4b06a94b9f98dc36dd9b9fa00` completed without execution errors
and each reports zero results. Those PR results do not erase the retained
base-branch alerts. No additional alert dispositions were recorded.

## Published 1.1.4 source reconciliation: 23 October 2026

Exact release source `1b1a92e094bdfb35e4462a6b7e08f6b5d9aafc95` passed master CodeQL run
`39830895466` and develop CodeQL run `39830895639`. The same 30 open
findings (23 high and seven critical) now identify this source; rule/path/ID
comparison found no new alerts. Earlier source references above are historical.
No new dismissals were made. Boundary capabilities and field limits remain
as documented; successful scan execution does not mean zero findings.

## Published 1.1.5 binary-source reconciliation

Source `04cbb1ae08bfb87f25249876201066880c08a2d5` passed exact-source master/develop CodeQL.
At that source the same 30 open findings (23 high, seven critical) remained;
ID/rule/path comparison found no new alerts. No additional dismissals were made.
The recovery packaging commit is distinct from the binary source.
See [release evidence](../history/RELEASE_1_1_5_VERIFICATION.md).
