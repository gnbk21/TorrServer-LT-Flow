# Flow 1.1.5 verification

23 October 2026. Published verification follows the retained preparation checkpoints below.

## Scope

The [147.1 review](UPSTREAM_MATRIX147_1_REVIEW.md) was reconciled against Flow.
Query handling is adapted across both stream paths, path-based playback and
Rutor/Torznab/JacRed. Canonical source validation shares the executed value.
Complete CP1251 conversion belongs to the separately scoped automatic subtitle
feature; Flow has no affected follow-redirects dependency. No engine replacement
or unmeasured discovery policy is added.

Runtime/test candidate: `a2542441569236bb73f85b5f832b74aed058148d`.
Local Go 1.26.10 tests passed all three top-level decoding/authorization tests
and ten named subtests; vet passed. Handler and native lifecycle/DHT checks use
the native CI toolchain. Workflow policy tests and actionlint passed locally.
Actual remote results are recorded below when completed.

## Focused release boundary

`RELEASE_ACCEPTANCE.json` identifies the exact candidate/native-security run and
earlier full engine run. The selector validates both sources, successful run
conclusions, workflow paths and repository identity. It rejects changed engine,
cache, scheduler, dependencies, native patches, build/distribution recipes or UI.
Only the explicitly reviewed API surface and tests can differ from the full
baseline; after the tested candidate only CI/top-level documents can differ.

The original full-platform plan required all 25 roles. The owner subsequently
approved six platforms after the Mac scheduler failures; the published scope
and all 23 applicable gates are recorded below. Full
comparisons/CPU capture are earlier evidence, not newly passed trials. Physical
and endurance limitations from [1.1.4](RELEASE_1_1_4_VERIFICATION.md) remain.
The owner's installed 1.1.4 and state are preserved during isolated verification.

## Accepted candidate evidence

- Candidate `a2542441`: [native/security workflow 39977825793](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39977825793) passed all jobs, including actual native protocol/cache/API regressions and CodeQL. [Branch checks](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39977825956) and [macOS](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39977825892) passed.
- Native job `120505747476` in build run `39977825877` passed unit/vet/race/vulnerability/fuzz gates. Its retained JSON test log confirms both real stream handlers preserved source URL components; Torznab/JacRed preserved query literals; concurrent metadata removal released all waiters.
- Empty native routing state with a deliberately unavailable local bootstrap node: first contact 1.043 ms, node available 1.001 s, first get_peers 4.002 s, peer result 4.003 s. No public bootstrap, production torrent or payload was involved. This case did not reproduce a multi-minute gap; no additional discovery retry policy is justified by it. The case also passed under the native race gates. These timings are controlled fixture evidence, not public-swarm performance.
- Reused full engine validation is successful run `39761266311`, source `d482e8c0`. The strict component and candidate diff checks protect reuse. It supplies the previously recorded full comparisons and CPU-capture evidence, not new 1.1.5 performance results.
- The owner's 1.1.4 executable hash and existing state fingerprints remained unchanged. No Jackett/RuTracker credentials, login or phone configuration were accessed.

Local evidence: `.tools/upstream1471/native-validation.log`,
`accepted-candidate-run.json`, `local-query-tests.jsonl`. The prior broad PR
workflow can continue to collect optional comparison data; it is not used to
pretend an unfinished comparison has passed.

## Patch-pipeline verification

79 local build/release/policy tests passed, including invalid receipt/source/
workflow/repository rejection, future patch-version selection, and full fallback
for undeclared plans. The restricted first attempt hit Windows loopback and
Git Bash permission errors; the authorized repeat with controlled temporary
paths passed. Actionlint passed. Live selection against both pinned successful
GitHub receipts returned `essential`, with UI run `39761266311`.

The cold-bootstrap native test passed and does not support adding an extra
15-second retry loop to Flow for this patch. Existing discovery behavior stays.

## Published Latest: six platforms, macOS pending

[1.1.5](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.145.Flow-v1.1.5) is stable
Latest for Windows, Linux and Android. Binary source remains `04cbb1ae08bfb87f25249876201066880c08a2d5` (PR #8
merged); the original tag was not moved. Intel and ARM macOS assets are absent,
explicitly pending in release notes, BUILDINFO and the update manifest. Mac
users should retain 1.1.4. This is an owner-approved platform omission, not
successful macOS validation.

Tagged run [39982129539](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39982129539)
passed all 23 required non-macOS roles: six targets, native queue simulation,
unit/vet/race/fuzz/reachable-vulnerability checks, eight functional groups,
Windows service/update/rollback and dependency inventory. Both attempts to
acquire Mac runners were cancelled before any build steps. GitHub explicitly
reported hosted runner acquisition failure and macOS ARM capacity constraints.
Both Mac targets share that pool; an earlier same-day candidate Mac run passed.
[Incident evidence and boundaries](MACOS_RUNNER_INCIDENT.md).

The independent 25-role default publication gate correctly rejected the missing
Mac results. An earlier progress update mistakenly treated disappearance from
the active queue as success and was corrected; no release was published on
that basis. Recovery run [39989260946](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39989260946)
separately checked all 23 applicable roles, exact tagged source and successful
security receipt, and explicit no-runner scheduler evidence. It reused the
already validated tag artifacts and passed reproducible subset packaging,
CycloneDX validation, provenance and container runtime checks. Recovery tooling
commit: `7a349d65d2718a08fe32f1478c4943495216f5aa`. Attestations identify this packaging workflow/source,
while manifest/Go build metadata retain the original binary source. These
identities are deliberately distinct. Default releases still require all 25
roles. No new two-hour comparison was run; unchanged-component full performance,
CPU and browser evidence is explicitly reused under the component policy.

- All **29 assets** are inventoried: six packages, nine Go binaries, optional
  Windows guard and distribution records. Five selected downloads passed four
  SHA256SUMS entries; 28 available GitHub
  asset digests agreed. Other platform bytes were not all downloaded locally.
- Windows executable and ZIP provenance passed exact recovery workflow commit,
  master source ref and GitHub-hosted-runner verification. Both standard/GST
  Go identities match the immutable binary source, version and Go 1.26.10.
  ZIP entry checksums, deterministic timestamps, bundled scripts and notices
  passed. No macOS binaries or links are advertised in the manifest.
- EXE SHA-256: `2476b1404055809da71b9cb140e1822a2922d3d80420ccf9a00126ee47de7f9b`.
  ZIP SHA-256: `aac99a0be55ee9170389893c914e3e852723e661b52e7739a046cad9daf19e97`.
- Published EXE bytes match the independently exercised tagged artifact. Actual
  Windows UI/runtime smoke passed (owned isolated readiness
  520.2 ms). Generated embedded assets honestly retain
  modified identity. Actual HTTP query/auth/source handling passed all
  12 loopback cases. These are not phone/public-swarm tests.
- Live GitHub install and 1.1.4-to-1.1.5 update passed in **PowerShell 5.1 and 7**,
  including required attestations, retained authentication/configuration,
  loopback listeners, modern UI and protected backup. Owned processes stopped;
  the owner's original installation/settings were not replaced.
- Anonymous GHCR reads verified three Linux architectures, matching source and
  version labels, and tag/latest index digest `sha256:6f5670042aebe998cd67c5f9c99f0a55efb5cc198baa62fc8ab68ff0be15caea`.
  CI ran exact-version/linkage checks on every architecture before pushing.
- Binary-source CodeQL and branch receipts passed. At that source, the same
  30 retained reviewed findings (23 high, seven critical) remained open with
  no new IDs/rules/paths or dismissals. This is not a zero-alert claim.
- Recovery publication took 3.2 minutes from its dispatch;
  container completion took 12.2 minutes. This excludes earlier
  failed Mac queue attempts and is not an equal-workload speed comparison.

Receipts: `.tools/flow115-published`, `.tools/flow115-release-runtime`,
`.tools/flow115-query-runtime`, `.tools/live-flow115-ps7`,
`.tools/live-flow115-ps51`, `.tools/upstream1471` and linked CI logs.
Broader phone/public-swarm, physical Windows recovery, exact-final endurance,
open security findings and macOS acceptance remain disclosed limitations.
