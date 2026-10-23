# Release pipeline

Patch releases use the same source identity, security, correctness, packaging
and provenance checks as other stable releases. Speed comes from scheduling
independent work together and reusing dependencies within their existing trust
boundaries.

## Scheduling

- A tagged release validates and builds the web interface once. Both cross and
  macOS builders download that artifact from the same workflow run. Standalone
  builds still validate their own interface. Shared-artifact wrapper jobs stay
  successful; only their duplicate validation steps are skipped, avoiding
  GitHub's propagation of skipped ancestor jobs.
- Native queue simulations and platform compilation run independently. A failed
  simulation still fails the reusable build workflow and blocks publication.
- Linux compilation uploads its ordinary binaries and exact native dependencies
  before unit, race, vulnerability, fuzz and integration checks run on separate
  runners. This also lets native dependency caches finish saving earlier.
- Eight isolated integration groups replace the former serial playback step:
  playback, startup, discovery, preparation, adaptive/integrity, request limits
  and cancellation, sparse supply, and maintenance/TLS/console.
- High bitrate comparisons, Windows service/update/rollback checks and native
  profiling can run alongside those groups. Policy trials within each comparison
  remain sequential on their own runner. Full validation retains all ten
  scenarios and all rotations and duration budgets.
- The combined `flow-controlled-playback` artifact keeps its previous paths.
  Per-group artifacts retain partial diagnostics if a group fails.

## Downloads and caches

Ubuntu jobs use the official HTTPS Ubuntu archive instead of the sometimes slow
Azure mirror, with bounded connection waits and retries. Repository signature
verification remains enabled. Playwright browser binaries are cached by OS,
architecture and the exact web dependency lockfile; system dependencies are
still checked on each runner.

Source caches, per-target native dependency caches, Go/Yarn caches and same-run
artifacts serve their existing roles. Release identity, checksum reproducibility,
dependency inventory and GitHub attestations still run before publication.

## Essential checks used for the 1.1.4 repair

The original unpublished tag's run failed while downloading Ubuntu dependencies;
the Linux profiler never started. The owner requested essential checks so the
release could proceed. The corrected tag rebuilds every platform and repeats
native simulations, native unit/race/vulnerability/fuzz checks,
all eight functional groups, Windows lifecycle checks and packaging/provenance.

The unchanged embedded UI is downloaded from that accepted full run after its
source/conclusion check, copied into the current run, and smoke-tested in the
final native package. Its earlier full browser/unit/laboratory results are reused.
Duplicate full streaming comparisons and native CPU capture are omitted.
The full accepted runtime run `39761266311` already passed those checks, including
122 original-byte streaming trials. Before selecting essential scope, the
classifier verifies that run's exact commit and successful conclusion and rejects
any changed runtime, web, dependency, native patch, compiler recipe, packaging or
distribution input. Only an explicit list of CI files and top-level release
documents can differ. Releases without an explicit accepted patch plan use full validation.

The previous tag source and failed run remain documented; no published release
or owner installation was replaced. Exact elapsed time depends on runner
availability, cache hits and network speed. A shorter duration must be measured
from a completed run, not inferred from the new graph.

Before packaging, publication independently reads actual job conclusions from
all run attempts. Every required platform, functional group and security/native
role must have a successful latest attempt. Skipped or missing jobs cannot pass
the gate, even if the reusable workflow's overall result looks successful.

## Completed 1.1.4 measurement

Run `39830894745`, source `1b1a92e0`, passed all 25 required publication
roles. GitHub published the release 18.4 minutes after the run started; the
container completion brought the total to 26.1 minutes. These are measured
essential-scope times for one run, not a measured speedup of the full suite.
Exact-source master/develop CodeQL workflows separately repeated full web
validation successfully. [Distribution verification](RELEASE_1_1_4_VERIFICATION.md)
records the final receipts and reused evidence.

## Reusable focused compatibility-patch validation

The original essential shortcut applied only to the 1.1.4 repair. Starting with
1.1.5, a stable patch can declare `patch_validation` in RELEASE_ACCEPTANCE.json.
The same selector works for subsequent patch versions without a workflow edit.
An absent or nonmatching plan chooses full validation; invalid matching plans fail.

The API-compatibility policy permits only a reviewed list of query/playback/search
files and named tests to differ from an exact successful full engine build. It
rejects changes to engine/cache/scheduler code, native patches, frontend sources,
dependencies and build/distribution recipes. A separate exact tested candidate
is pinned: every later runtime or test change is rejected. Both accepted GitHub
receipts must match their source, repository, expected workflow and successful
completed conclusion. The candidate must pass native CodeQL/protocol/cache/API
regressions; the baseline must be a full build run.

The tag still rebuilds all targets, runs native unit/race/vulnerability/fuzz and
all eight functional groups, checks Windows lifecycle and requires all 25 actual
publication roles plus checksums/reproducibility/provenance. Only unchanged UI
validation and earlier full performance/CPU matrices are reused. This is
component-specific evidence reuse, not a claim of identical entire runtime or
new performance acceptance. Broader changes continue through the full path.
