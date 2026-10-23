# Flow 1.1.5 verification

23 October 2026. Preparation record; publication is not claimed at this checkpoint.

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

All 25 existing essential publication roles must still actually succeed. Full
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
