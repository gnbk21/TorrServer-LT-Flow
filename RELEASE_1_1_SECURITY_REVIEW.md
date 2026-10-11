# Flow 1.1 security review

CodeQL scan execution and alert disposition are separate checks. This review
does not claim that scanning proves the absence of vulnerabilities.

## Corrected findings

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
