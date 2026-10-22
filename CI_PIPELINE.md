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
  remain sequential on their own runner. All ten scenarios and all rotations
  and duration budgets are retained.
- The combined `flow-controlled-playback` artifact keeps its previous paths.
  Per-group artifacts retain partial diagnostics if a group fails.

## Downloads and caches

Ubuntu jobs use the official HTTPS Ubuntu archive instead of the sometimes slow
Azure mirror, with bounded connection waits and retries. Repository signature
verification remains enabled. Playwright browser binaries are cached by OS,
architecture and the exact web dependency lockfile; system dependencies are
still checked on each runner.

No previous run's test result is substituted for a release gate. Source caches,
per-target native dependency caches, Go/Yarn caches and same-run artifacts serve
their existing roles. Release identity, checksum reproducibility, dependency
inventory and GitHub attestations still run before publication.

The already tagged 1.1.4 workflow uses its immutable original definition. These
scheduling changes apply to subsequent builds. Exact elapsed time depends on
runner availability, cache hits and network speed; a shorter duration must be
measured from a completed run, not inferred from the new graph.
