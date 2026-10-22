# Flow 1.1.4 release verification

Target: `MatriX.145.Flow-v1.1.4`, stable Latest, explicitly requested by the owner.
Publication is gated on an exact tagged-source rebuild and package verification.
This preparation checkpoint does not claim the tag is already published.

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
