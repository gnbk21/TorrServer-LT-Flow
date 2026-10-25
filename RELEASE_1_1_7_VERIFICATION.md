# Flow 1.1.7 verification

Published 25 October 2026. Tag: `MatriX.145.Flow-v1.1.7`.
Immutable binary source: `87d1632e47d64adcfd98d7d916d80aebedbc4e8c`.
The successful tagged workflow and public download receipts are recorded below.

## Accepted implementation

[PR #10](https://github.com/gnbk21/TorrServer-LT-Flow/pull/10) merged at
`48ee5e73f285d8562bdfcf23a8a3ddce7406c1d5`. Its implementation tree matches
accepted head `8ebe6caa7c089a8901336d2e40aa9ee7ce67ecf4`; GitHub tested merge
`614a1dbdcd36f9a92a33c004e8450159acb3c910`.

| Evidence | Result |
| --- | --- |
| [Full build 40183731558](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40183731558) | All 40 required roles passed: six non-Mac targets, native simulations, Windows lifecycle, Linux validation, eight functional groups, six parallel-streaming cases, ten high-bitrate comparison jobs, two profiling captures, inventory and combined evidence |
| [macOS 40183731551](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40183731551) | Intel and ARM plus web validation passed |
| [CodeQL 40183731547](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40183731547) | All five jobs passed, including expanded native security regressions and actual Go/C++ bridge extraction |
| [Branch checks 40183731541](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40183731541) | Four required Windows/Linux, interface-guard and inventory jobs passed; push-only web job intentionally skipped, actual web gates passed in the other workflows |

Native analysis `2021864995` reports six accepted administrator-operation
findings. The other three categories report zero; the PR merge ref has zero
open findings after the owner's specific approval. The recorded #92–#97
dispositions do not represent six additional vulnerability fixes. No rule,
severity or extraction coverage was suppressed. See
[security hardening](SECURITY_HARDENING.md) and
[accepted residual privileges](SECURITY_DISPOSITIONS_PR10.md).

The source-identical Windows implementation candidate passed 15 isolated API
checks: origin/rebinding protection, explicit local indexer access, redirect
isolation, compatible identity playback and missing-account fail-closed behavior.
Executable SHA-256:
`295e619d26348fae6e1df4fd678fde93310c1e9d6eb5860cfba9154d8af2cdb4`.
That candidate identifies earlier tested merge `b9044a7612b53365e947c07ed1f2f5dcfc725fff`;
the subsequent accepted head changes only review Markdown. This is separate
from verification of the eventual tagged binary. Both disposable instances stopped.

## Reuse and exact-tag gates

The acceptance record pins the same fully tested source for candidate CodeQL
and full-build baseline. Only top-level documents, acceptance metadata and the
release-note workflow selection differ afterward. The existing selector rejects
later runtime/UI/native/dependency/build/distribution changes and verifies both
exact successful GitHub receipts before reusing the generated UI and complete
streaming/profiling matrices. The `api_compatibility` mode name is retained;
there is no runtime difference from this accepted baseline.

The tag still rebuilds all eight platforms and requires all 25 essential
publication roles, including fresh native/race/vulnerability/fuzz tests, all
eight functional groups, Windows lifecycle, actual binary tag/source identity,
reproducible packaging, CycloneDX inventory and GitHub attestations. No skipped,
cancelled, missing or unfinished required role can satisfy publication. Containers
must pass version/linkage checks for all three Linux architectures.

## Owner installation and field limits

The running installation and its personal settings are not replaced by release
publication. Public assets contain no owner configuration or credentials.
Authentication/origin/local-integration compatibility changes are described in
[release notes](RELEASE_1_1_7_NOTES.md). Legacy defaults and disabled-by-default
streaming experiments are retained; owner-selected profiles remain settings.

Representative phone/public-swarm playback, physical Windows recovery and
exact-final multi-hour acceptance remain unverified. No further two-hour tests
are run. Accepted administrator privileges remain risks that require trusted
management access; this release is not an exhaustive penetration-test claim.
Windows executables remain unsigned by an Authenticode publisher certificate.


## Published Latest: all eight platforms

[Flow 1.1.7](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.145.Flow-v1.1.7) is public, stable
and Latest. [Tagged workflow 40188065181](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40188065181)
completed successfully, including all 25 actual required essential roles
and the three-architecture container job. Earlier source-identical full
comparisons and browser validation were reused; skipped optional comparisons
were not counted as new successful tag checks. No two-hour test was run.

- All 35 published assets are inventoried. The
  34 SHA256SUMS entries agree with GitHub's
  asset digests. 4 selected downloaded
  files passed their actual checksums. Other platform bytes were not all
  downloaded locally.
- The Windows ZIP passed content digests, fixed archive timestamps, the exact
  expected file allowlist and shipped-script comparisons against immutable Git
  blobs. Both standard and GST executables identify the exact tag, source and
  Go 1.26.10. Standard EXE SHA-256: `74cbf0d23a7a4d02db3a375a81df7b7ec3f5ad19351c9000d36bfa65d6ab3f35`.
  ZIP SHA-256: `7a405c512b7f7380b64758ee473faf508262badb61492f687d3234f91f5f0251`.
- EXE and ZIP provenance passed exact source digest, tag ref, release workflow
  and GitHub-hosted-runner enforcement. This is separate from Authenticode.
- The downloaded executable passed all 15 isolated security API checks using
  disposable settings and local fixtures. Both owned test instances stopped.
  No personal settings were accessed and no installed server was replaced.
- Anonymous GHCR metadata reads verified AMD64, ARM64 and ARMv7, matching source
  and version labels, and equal tag/latest index digest
  `sha256:48f0a0e9a1f14b7200f910fc77025ce66691bb87aa014e28057cb42ceaf3ec0f`. CI performed actual version/linkage execution
  on each architecture; local registry reads verify metadata.
- [Exact tagged-source CodeQL 40188062360](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40188062360)
  passed all five jobs. Its full native scan reports **21 results**: the six
  initial owner-approved dispositions, 12 additional dispositions approved after
  full-branch review, and three pre-existing dispositions. The other categories
  report zero. GitHub did not emit a separate PR aggregate results check for
  this master push; its actual workflow jobs and full scan/alert state were
  verified. The release source has zero open findings after these dispositions. Accepted findings are not
  additional vulnerability fixes or a vulnerability-free claim.

### Full-branch review and interrupted first attempt

The PR's diff-informed analysis reported only six results. Full master analysis
found twelve additional open findings outside that PR scope. Automatic release
run attempt 1 was cancelled before publication; no tag or source was replaced.
The exact additional findings, existing safeguards and risks were reviewed in
[SECURITY_RELEASE_1_1_7_REVIEW.md](SECURITY_RELEASE_1_1_7_REVIEW.md). The owner
explicitly approved ten retained administrator privileges and two false-positive
dispositions. Each saved state was verified separately; the prior six-alert
approval was not reused. The interrupted request group and dependent evidence /
publication were rerun while earlier successful roles were retained. The final
25-role gate verified actual latest attempts. No required check was waived.

Raw local receipts: `.tools/flow117-published/` and
`.tools/flow117-security-runtime/`. Field, physical recovery, unsigned Windows
publisher identity and exact-final endurance limitations remain as stated above.
