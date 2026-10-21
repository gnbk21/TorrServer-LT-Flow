# October security evidence reconciliation

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
belong in the final October verification record.
