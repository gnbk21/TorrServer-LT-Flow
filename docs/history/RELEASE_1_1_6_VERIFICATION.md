# Flow 1.1.6 verification

Published 25 October 2026. Immutable binary source: `cd8f1cceef7f11f9e87200be3426abc54fb33d9c`.
Tag: `MatriX.145.Flow-v1.1.6`. The completed tagged workflow and selected public downloads
were verified as recorded below.

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
  [security boundaries](../security/RELIABILITY_USABILITY_SECURITY_REVIEW.md) remain applicable.

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
fingerprints remained unchanged and the temporary process stopped. The exact
published executable was checked separately as recorded below.

The owner's running installation/settings are not replaced by this release task.
Personal configuration is never published. Experiments remain optional, Legacy
remains the default for new installations, and existing configurations retain
their selected profile. Broader real-phone/public-swarm, physical Windows and
multi-hour field acceptance remains unverified. No further two-hour test is
required or started. Existing security findings remain disclosed, and Windows
executables are not Authenticode signed.


## Published Latest: all eight platforms

[1.1.6](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.145.Flow-v1.1.6) is
public, stable and Latest. [Tagged workflow 40174251573](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40174251573)
completed successfully. All 25 required essential roles passed; optional
comparison/profiling jobs skipped under accepted component reuse are not counted
as newly passed checks. Both macOS architectures are included.

The first Mac ARM job compiled, passed native tests and version probes, then
failed artifact upload with `getaddrinfo ENOTFOUND` for GitHub's Azure artifact
storage hostname. GitHub refused an immediate job retry while the workflow was
still running. The failed job was retried after the other checks finished;
successful earlier roles were retained, and the final gate checked the latest
attempt for each role. No tag, source or gate was weakened or replaced.

- All 35 published assets are inventoried. The
  34 SHA256SUMS entries agree with GitHub
  asset digests. Four selected downloaded files passed their checksums; other
  platform bytes were not all downloaded locally.
- Windows ZIP entries passed content digests, fixed archive timestamps and
  shipped-script checks. Standard/GST Go identities match the immutable tag,
  source and Go 1.26.10. EXE SHA-256: `07d87be7d804b42a7f9c634d6deea99b1d7f9b513805635ceca55b987de6f8a1`.
  ZIP SHA-256: `12b0eb239fe9555637564b328d07dafa653983599c71e8b70a5b874b91ee5ffa`.
- Standard executable and ZIP attestations passed the exact source digest,
  tag ref, release workflow and GitHub-hosted-runner policy.
- The downloaded standard executable passed stable version/source identity,
  modern UI and an isolated read-only owner-settings snapshot check: readiness
  1066.2 ms, 2048 MiB cache,
  adaptive profile, new controls off. The owned process stopped.
  No owner state was written or running installation replaced.
- Public anonymous GHCR reads verify three Linux architectures, matching source
  and version labels, and the same tag/latest index digest
  `sha256:1ec7c84362db16e05f88240669f194e512678afad1a098fa6bebe01ad0cf6f0f`. CI executed exact-version/linkage checks on
  each architecture before publication; local checks inspect registry metadata.
- Exact tagged-source [CodeQL 40174248931](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40174248931)
  and branch checks passed. The rechecked repository inventory contains 30 open
  findings (23 high, seven critical); no dismissals or protection changes were
  made. Scan success is not a zero-alert claim.

An initial local shipped-script comparison encountered the checkout's CRLF line
endings. Comparing the package directly with the immutable tag's Git blobs
confirmed identical source bytes. The corrected verification harness passed;
no published asset was changed.

Raw local receipts: `.tools/flow116-published/` and
`.tools/flow116-runtime-config/`. Existing phone/public-swarm, physical recovery,
unsigned Windows publisher identity and exact-final endurance limits remain.
No further two-hour test was run. Personal configuration is absent from public
assets; the separate local launcher reuses the owner's existing state.
