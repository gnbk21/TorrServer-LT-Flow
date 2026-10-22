# Release pipeline

Patch releases use the same source identity, security, correctness, packaging
and provenance checks as other stable releases. Speed comes from scheduling
independent work together and reusing dependencies within their existing trust
boundaries.

## Scheduling

- A tagged release validates and builds the web interface once. Both cross and
  macOS builders download that artifact from the same workflow run. Standalone
  builds still validate their own interface.
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

## Essential checks for the unpublished 1.1.4 repair

The original unpublished tag's run failed while downloading Ubuntu dependencies;
the Linux profiler never started. The owner requested essential checks so the
release could proceed. The corrected tag rebuilds every platform and repeats
web validation, native simulations, native unit/race/vulnerability/fuzz checks,
all eight functional groups, Windows lifecycle checks and packaging/provenance.

Only duplicate full streaming comparisons and native CPU capture are omitted.
The full accepted runtime run `39761266311` already passed those checks, including
122 original-byte streaming trials. Before selecting essential scope, the
classifier verifies that run's exact commit and successful conclusion and rejects
any changed runtime, web, dependency, native patch, compiler recipe, packaging or
distribution input. Only an explicit list of CI files and top-level release
documents can differ. Other tags continue to use full validation.

The previous tag source and failed run remain documented; no published release
or owner installation was replaced. Exact elapsed time depends on runner
availability, cache hits and network speed. A shorter duration must be measured
from a completed run, not inferred from the new graph.
