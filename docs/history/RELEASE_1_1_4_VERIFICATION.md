# Flow 1.1.4 release verification

Target: `MatriX.145.Flow-v1.1.4`, stable Latest, explicitly requested by the owner.
Publication is gated on an exact tagged-source rebuild and package verification.
Publication and downloaded distribution checks subsequently passed, as recorded below.

## Accepted implementation

Production implementation `f780398f7d9f607f4024504d79492c336080f7d0` and corrected
test harness `d482e8c06d0e7cce7575f5300f30cfc70290153b` passed all non-skipped gates.
The subsequent `5a7a6e7d` update changes only Markdown. Release preparation changes
notes, acceptance records and the release title/body selection, not runtime code.

- [Full native/platform/web/streaming/service/profiling build](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39761266311).
- [CodeQL](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39761266130),
  [branch checks](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39761266642),
  [macOS](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39761266143) and
  [optional player APK](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39761265933).
- 122 streaming trials, including 50 rotated straggler trials and 200 proven cold
  targets. All original-byte checks passed; mixed latency results retain defaults.
- Exact separately staged Windows candidate passed identity, embedded UI,
  corruption rejection/refetch and cold-seek smokes without touching owner state.

Full methodology, source identities, numeric comparisons and retained failures:
[October verification](OCTOBER_UPGRADE_VERIFICATION.md).

## Publication policy and outstanding field work

The owner requested 1.1.4 after reviewing the disclosed limitations. Required
automated gates remain mandatory; physical/device checks are disclosed rather
than relabelled as passes. No further two-hour tests will be started.

Unverified: exact-final endurance, eight-hour convergence, physical boot/sleep/
adapter recovery, broader real-phone/public-swarm playback and spoken screen-reader
acceptance. Earlier completed two-hour tests identify an earlier executable.
CodeQL source reconciliation preserves retained base findings; a green scan is
not a repository-wide zero-alert claim. Optional player decoder redistribution,
shared OS DNS protection and complete native profile attribution remain separate.

## Tagged and downloaded distribution evidence

Exact tagged run, package hashes/source/version, provenance, managed Windows
installation/update and public container identities will be recorded here after
their checks complete. The old installation remains unchanged during release work.

## CI repair before publication

Original unpublished source `a1c1f2db798fce90bef3b10560189f4aad44919f`, run
`39808722352`, encountered slow HTTP downloads from Azure's Ubuntu mirror. The
Linux profiler setup timed out before capture began; this was not a Flow runtime
failure. Earlier download delay/cancellation and this failed setup are retained.

The owner requested essential checks. The release repair uses the official HTTPS
Ubuntu archive, parallel validation and one shared web build. Every platform is
rebuilt and core correctness, security, functional, Windows lifecycle and release
provenance checks remain required. Only duplicate full comparisons/CPU capture
reuse successful accepted runtime run `39761266311` at `d482e8c0`. The classifier
checks both its identity/conclusion and a strict CI/document-only diff allowlist;
any runtime, web, native patch, dependency, build recipe or distribution change
requires full validation. No new two-hour tests run.

The tag had no published release/assets when repaired. The corrected source,
actual essential results and final distribution identities are recorded after
publication; the old source and failed evidence above remain historical.

The first CI repair at `c64e752c`, run `39828387868`, exposed skipped-ancestor
propagation from shared web jobs. Core downstream jobs were skipped; this was
caught and the run cancelled before publication. Windows identity/UI/integrity
smokes passed but do not replace those missing gates. The subsequent correction
keeps shared-web wrapper jobs successful and adds an independent actual-job
receipt gate before packaging. Local release/policy tests: 73 passed; actionlint
passed. This scheduling failure remains part of the evidence history.

The final essential repair also reuses the accepted run's identical generated web
assets and full web/unit/browser/laboratory results, after the strict source and
successful-run check. It copies them into the current run for every builder;
actual final native UI smoke and backend/frontend API contracts are checked
again. No change to web sources, dependency lockfiles or the embedding generator
is permitted by that eligibility check. The final publication gate requires 25
actual successful core roles; broader matrices are explicitly earlier evidence.

## Published Latest: corrected tagged and downloaded evidence

[Flow 1.1.4](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.145.Flow-v1.1.4)
is stable **Latest**, source `1b1a92e094bdfb35e4462a6b7e08f6b5d9aafc95`. PR #6 contains the runtime upgrade;
PR #7 and follow-up CI corrections repair the release pipeline. Both PRs are merged. The unpublished
preparation tag was repaired before publication; the previous source/run above are retained.

- [Corrected run 39830894745](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39830894745)
  passed all eight platform targets, source-bound shared web assets, native queue
  simulations, unit/vet/race/fuzz/reachable-vulnerability checks, actual frontend
  API contracts, all eight isolated functional groups, Windows service/update/
  rollback, inventory, reproducible packaging, provenance and container checks.
- The owner requested essential checks. Duplicate full streaming comparisons and
  CPU capture were not repeated on the final tag. Successful accepted full run
  `39761266311`, source `d482e8c0`, supplies those earlier results. A strict diff
  check and the accepted run's exact identity/conclusion protect that reuse.
  The identical generated web assets were copied into this run before compilation;
  actual final native UI smoke and backend/frontend API contracts ran again.
  Exact-source master/develop CodeQL workflows also repeated the full web checks.
  Downloaded earlier reports confirm 122 original-byte trials, including 50
  straggler trials and 200 proven-cold targets. This is not a claim that the final
  tag repeated those workloads. No further two-hour tests ran.
- Exact final-tag reports confirm 15 strict request-capacity cases, six cold
  cancellation/late-completion cases on each of Linux and Windows, integrity
  recovery, startup cases, restricted Windows SCM/update/migration rollback and
  four scoped WFP protocol checks with cleanup/routing restoration.
- All 35 assets are inventoried. The selected
  5 assets were downloaded; their four checksum
  entries passed and 34 available GitHub asset
  digests agree with SHA256SUMS. This does not claim all platform assets were
  separately downloaded locally. CI checked every packaged binary identity.
- Windows ZIP and executable attestations verified exact source/tag and
  ts_release.yml on GitHub-hosted runners. ZIP entries and deterministic metadata
  agree with checksums. Standard/GST identities report Go 1.26.10 and the exact
  source; the standard runtime reports libtorrent 2.1.2.0.
- Windows executable SHA-256: `a505c59d94e6fe57e70e0e122d1a304d21af4e3e7522bea49a845ee99ee21018`.
  Windows ZIP SHA-256: `e543bb830f73be70210aa995d8350ce9ead7d052aaad6e1581406e0f8ea27ea8`.
- Actual final Windows runtime and desktop/mobile embedded UI passed without API
  fixtures, browser errors or applying settings. CacheSize 0 compatibility,
  Legacy and disabled experiments remained usable. Runtime/API source and stable
  channel agree. Generated embedded assets honestly retain `modified: true`.
  Isolated ready time: 540.9 ms. A corrupt generated
  block was rejected and refetched into the original 8 MiB. These are controlled
  server checks, not phone-decoding or public-swarm performance claims.
- Live GitHub fresh installation and 1.1.3-to-1.1.4 update passed independently
  under PowerShell 5.1 and 7 with required provenance. Authentication, loopback
  listener, state/configuration and protected recovery backup survived. Owned
  test instances shut down; the owner's installation/settings were not replaced.
- Anonymous GHCR checks confirm Linux amd64, arm64 and arm/v7 with exact
  source/version labels and identical version/latest index digest:
  `sha256:4b1889402ebb4965df2a5c7572553a3c9a5f20a3dd728934b7d8e8ee728a3d2f`. CI checked actual version/linkage on all three
  architectures before pushing the container.
- All 25 required actual CI roles passed the independent publication gate.
  Exact-source master CodeQL `39830895466`, develop CodeQL `39830895639` and
  develop branch checks `39830895540` passed. The same 30 retained reviewed
  security findings remain open (23 high, seven critical), now identifying this
  source. No new alerts or dismissals were recorded; a passing scan is not a
  zero-alert result.
- Local pipeline verification passed 73 build/release/policy tests, actionlint,
  shell syntax checks and diff checks. The completed essential run took 18.4
  minutes to GitHub publication and 26.1 minutes including container completion.
  This is one essential run, not a measured speedup of the complete full suite.

Receipts: `.tools/flow114-published`, `.tools/flow114-final-ci-receipts`,
`.tools/flow114-release-runtime`, `.tools/flow114-release-ui-state`,
`.tools/flow114-release-integrity`, `.tools/live-flow114-ps7` and
`.tools/live-flow114-ps51`. Earlier failed/cancelled evidence remains retained.
The broader physical/device/endurance and security limits above still apply.
