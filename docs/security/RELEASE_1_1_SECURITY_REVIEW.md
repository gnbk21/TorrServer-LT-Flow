# Flow 1.1 security review

CodeQL scan execution and alert disposition are separate checks. This review
does not claim that scanning proves the absence of vulnerabilities.

## Corrected findings

- Torrent removal validates and canonicalizes SHA-1/SHA-256 identifiers before
  constructing disk paths. Invalid input cannot reach the engine, database or
  recursive removal. Startup cleanup also rejects nonhexadecimal directories
  rather than recognizing them by length alone.
- Removal uses the actual hybrid storage identity, waits for native/cache disk
  ownership to drain, and checks re-addition under the engine lock. Timeouts or
  filesystem failures preserve the record and retained bytes for retry. Closing
  storage remains registered until in-use pooled disk handles are released.
- Torrent streaming assigns an explicit media/plain-subtitle/binary type and
  `nosniff`; HTML, SVG and unknown formats are downloads. Names never trigger
  local-file MIME sniffing. Byte-range responses retain their original bytes.
- GST allocation hints no longer add or multiply potentially large slice
  lengths. Certificate inspection and TMDB responses have actual read bounds.
- Artwork, TMDB, Torznab and JacRed HTTP clients use the existing selected
  interface transport, including disconnect cancellation and pooled-socket
  invalidation. Intentional owner-selected LAN endpoints remain supported.

- Log injection: convenience log functions and the shared native/standard log
  sink now escape embedded record separators, terminal controls and directional
  control characters. Original secret redaction and writer-consumption/error
  contracts are retained. Tests exercise native, access and convenience sinks.
- Inventory race: validation opens one descriptor, inspects that descriptor and
  reads at most 16 MiB plus one detection byte. Name replacement cannot switch
  the inspected file, and growth cannot remove the actual read bound.
- Workflow permissions: all workflows declare limited default token access;
  artifact retrieval, release, provenance and package publication receive only
  the permissions their jobs need. Reusable release calls grant required read
  access rather than silently disabling their fixture checks.
- Action supply chain: 19 existing action versions are pinned to commits
  resolved from their official repository tags, without changing major versions.
- Credential-free `/play` validates source syntax before parsing, matching
  `/stream`: torrent identities remain playable, local paths and remote fetches
  do not enter parsing through that unauthenticated path.
- Uploaded subtitle objects are created with `text/vtt` MIME, with ownership and
  revocation tests. The track source is an opaque browser-created blob URL;
  uploaded text is never assigned as an HTML string or a source URL.

## Reviewed retained behavior

The owner explicitly approved these seven documented dispositions on
11 October 2026. GitHub records alerts 1 and 7 as retained intentional
capabilities (`won't fix`) and alerts 2, 5, 6, 8 and 33 as reviewed false
positives. Corrected findings are resolved by code changes and fresh scans;
scan execution success alone is not a clean-alert result.

| Finding | Boundary and disposition |
| --- | --- |
| MSX request forgery | `remote` validates HTTP(S), rejects credentials/fragments, rechecks redirects, disables inherited proxies, validates all resolver answers and dials approved literal addresses. Private/loopback access requires explicit `MSXAllowLAN` and literal addresses. `CheckAuth` protects MSX when authentication is enabled. CodeQL does not model the custom transport/dial gate; reviewed as a false positive. |
| Merkle resume path expressions | The directory comes from owner-selected storage/state. Torrent metadata contributes only a validated bounded integer piece number formatted in decimal, never a filename/path component. Symlinks are rejected with Lstat, and the opened regular inode/size must match before hashing. Arbitrary metadata cannot inject path separators. Reviewed as false positives for this storage boundary. |
| Preparation cleanup path | The root is owner-selected persistent storage. Torrent identifiers are decoded into fixed hash bytes and re-encoded as hexadecimal; piece names are integers. Cleanup waits for reader/native disk ownership and preserves overlapping jobs. No arbitrary torrent filename reaches removal. Reviewed as a false positive. |
| Local `.torrent` source path | Reading an explicitly supplied owner-selected local `.torrent` path is a retained CLI/management feature. The parser reads at most 64 MiB and returns parsed metadata, not arbitrary raw files. This is an intentional path capability, not a claim of a restricted filesystem sandbox. Retained for compatibility. |
| `.torrent` HTTP source | Fetching an explicitly supplied URL, including owner-operated LAN sources and configured proxies, is a retained management feature. Requests have cancellation, bounded bodies, a 60-second timeout, a redirect limit and interface policy. Auth-required credential-free streaming rejects remote source links; management follows the existing authentication setting. This fetcher is not advertised as a public-only SSRF proxy. Retained for compatibility. |
| Subtitle track DOM sink | Only `URL.createObjectURL` of an owned VTT blob reaches the track source; file content cannot supply a javascript/data URL or HTML assignment. Reviewed as a false positive after the ownership/MIME regression. |

Retained source-input capabilities require trusted clients. Authentication is
optional for existing LAN compatibility; disabling it trusts clients with
management access. Do not expose the server directly to the public Internet.
The explicit restricted-origin/rate/playback-capability policies remain
available. These conclusions do not establish OS DNS leak protection, a general
VPN kill switch, an external penetration test or independent codec rebuilds.

## Additional findings from the full native scan

The `c02e7933` scan completed successfully but reported 49 additional findings.
Those findings were not covered by the owner's seven-alert dismissal approval;
none was dismissed using that approval. The fixes above require a new scan.

| Alerts | Review scope |
| --- | --- |
| 81, 82 | GST slice capacity arithmetic corrected. |
| 64, 65, 77 | Raw removal identifier corrected; canonical digest only. |
| 61–63, 76 | Cache roots are owner-selected. Cleanup now also validates directory hexadecimal syntax. |
| 56 | Generic response recorder is a passthrough sink; the real torrent response now supplies safe MIME/download headers before `ServeContent`. |
| 58–60, 66–68, 78–80 | State/cache roots are owner-selected; storage components are validated hexadecimal hashes and numeric piece indices. Resume opening checks the actual regular inode/size and verifies hashes; cleanup waits for native/reader ownership. |
| 57, 71–75 | Owner-configured diagnostics/state/TLS paths are intentional. Certificate downloads expose certificate PEM only, never keys; inspection and download bodies are bounded. |
| 69, 70 | Auth-protected MSX file sharing explicitly selects an owner directory; it is not a restricted filesystem sandbox. Authentication disabled means management clients are trusted. |
| 49–55 | Configured indexers, trackers, TMDB and poster URLs intentionally support owner-operated LAN services. Requests have time/body bounds and use the selected-interface transport. These clients are not public-only URL proxies; management authentication/exposure policy remains required. |
| 34–48 | Boost 1.85.0 **B2 build-engine** source, not linked or shipped server runtime. Findings include arithmetic, legacy format/scanner functions, writable build files and a suspected lifetime issue. B2 executes pinned, trusted build recipes on isolated CI runners; arbitrary client torrents do not become Jam recipes. The upstream tool is not claimed vulnerability-free. |

This review distinguishes retained capabilities and build-tool scope from
corrected runtime defects. Open findings and fresh scan results must remain
visible in the final release verification; scan success does not erase alerts.

The host-tool probe also had a build-cache defect: `pipefail` treated a valid
B2 version banner with a nonzero exit as a broken executable. The probe now
tests captured output independently. Real Bash regressions cover reusable,
wrong-host and missing tools. This avoids unnecessary bootstrap during native
bridge extraction; disappearance of a B2 finding in a later runtime scan would
not establish a fix in upstream B2 source.

## Final runtime source scan

[Run 38137745884](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/38137745884)
passed all four security categories and the web gate at feature source
`0b6da77a0c82d393ba12af2496aecdac4fb3347c`, checked out as PR merge
`dbc6774a16aef601550102391e83cc330c066be8`. The fresh PR-ref query retains
**30 open findings**: 22 path injection, seven request forgery and one generic
reflected-XSS sink. These are visible findings, not a clean-alert result.

The new removal flow is alert 89 at `apihelper.go`: its path uses a canonical
fixed-size storage digest under the configured cache root. Live hybrids use
their native storage identity; DB-only metadata must match the requested v1/v2
identity before selecting the corresponding storage digest. Native ownership
and replacement checks precede removal. The other findings retain the reviewed
owner-selected state/TLS/MSX/cache and source-URL boundaries above. Alert 56
remains on the generic response sink; the production stream caller supplies
the safe type/download/`nosniff` policy, including native WebVTT compatibility.

No additional alerts were dismissed. The omitted B2 bootstrap findings still
have the build-tool scope described above; the fresh extraction did not rebuild
that cached host tool and does not establish an upstream B2 security fix.

## Published 1.1.3 security verification

Exact source `baca8c41594e0fd6d3a17c14a6187482134d5135` passed master CodeQL run
`38148194003` and develop run `38148196346`. The final master query retains
30 open findings: 22 `go/path-injection`, seven `go/request-forgery` and one
`go/reflected-xss`. Alert IDs match the prior reviewed source and every latest
instance binds to this exact release commit. The approved seven dispositions
remain separate; no additional findings were dismissed. Retained owner-selected
paths/URLs and the response-sink boundary remain as documented above.

The checksum-bound Windows executable `266cec7297f8da63562a51f96269c28cca96de457145bb1afd74d87be3cd8540`
retains GO-2026-5932 in the stripped binary scan (exit 3, x/crypto v0.57.0,
OpenPGP unmaintained, no fixed version). Its resolved CGo-enabled Windows
import graph contains 511 packages, includes `server/lt` and excludes OpenPGP.
Both exact tagged Linux source variants report no vulnerabilities. This is
not a clean symbol-level Windows reachability result. Published ZIP/executable
provenance and all asset checksums passed; the optional Windows guard remains
explicitly installed and scoped, and shared OS DNS protection is unverified.
