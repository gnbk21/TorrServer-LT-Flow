# Flow 1.1.7 verification

Release target: `MatriX.145.Flow-v1.1.7`. Prepared 25 October 2026.
Publication, exact-tag identity, downloaded assets and container availability
remain pending until their receipts are recorded below.

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
