# Flow 1.1.6 verification

Prepared 25 October 2026. The immutable release tag and completed publication
receipts are added after the tagged workflow finishes; pending jobs are not
passing gates.

## Accepted implementation

PR #9 was merged into master after its final head
`2f261b7737060263291938d68e6e063762d16881` passed all required development gates.
Merge commit: `cd780a42` (identical implementation tree).

- [Full build 40132962712](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40132962712):
  six non-Mac targets, native queue simulations, frontend/contracts/browser/lab,
  Linux unit/race/vet/fuzz/vulnerability/storage checks, eight controlled
  functional groups, Windows service/update recovery, inventory, profiling,
  all ten existing high-bitrate jobs and six new parallel comparison jobs passed.
- [macOS 40132962595](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40132962595):
  Intel and ARM builds and version probes passed.
- [CodeQL 40132962678](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40132962678)
  and [branch checks 40132962685](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40132962685)
  completed successfully. This does not resolve existing reviewed findings.
- [Parallel-control measurements and retained failures](PARALLEL_STREAMING_VERIFICATION.md)
  record earlier source-identical comparisons and their mixed tradeoffs.
  [Reliability reconciliation](RELIABILITY_USABILITY_CHECKLIST.md) and
  [security boundaries](RELIABILITY_USABILITY_SECURITY_REVIEW.md) remain applicable.

## Reuse and exact-tag gates

The acceptance record pins the same fully validated implementation for both the
native/security candidate and full-build baseline. Only release metadata,
top-level documents and the release-note selection change afterward. The existing
selector rejects changes to runtime, UI, native patches, dependencies,
distribution scripts or build recipes. The compatibility mode name is retained;
no API/runtime difference from this baseline is permitted or needed here.

The tagged workflow still rebuilds all eight platforms and requires all 25
essential roles, including fresh native validation and all eight functional
groups, before packaging. It verifies binary tag/source identity, reproducible
packaging, CycloneDX inventory and GitHub provenance. Full comparisons and the
validated generated UI are explicitly reused; they are not newly passed tag
comparisons. Container publication follows the verified native release.

## Owner configuration and limits

The separately supplied development candidate was checked in a temporary
read-only loopback instance with copied owner configuration: 2048 MiB cache,
Adaptive profile, both new controls off and modern embedded UI. Original state
fingerprints remained unchanged and the temporary process stopped. This is not
verification of the forthcoming tagged executable; its identity and package are
checked separately after publication.

The owner's running installation/settings are not replaced by this release task.
Personal configuration is never published. Experiments remain optional, Legacy
remains the default for new installations, and existing configurations retain
their selected profile. Broader real-phone/public-swarm, physical Windows and
multi-hour field acceptance remains unverified. No further two-hour test is
required or started. Existing security findings remain disclosed, and Windows
executables are not Authenticode signed.
