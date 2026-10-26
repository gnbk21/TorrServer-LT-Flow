# Flow 1.1.8 verification

## Accepted implementation

Source: `78d2ec2ebfef9bd6180cfaf3be13bf3fd018361c`. All referenced source-specific workflows completed successfully:

- [Full native build and runtime checks](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40294716178): six Windows/Linux/Android targets, native queue/deadline simulations, Go/cgo, race, API contracts, functional integration, Windows lifecycle/update/rollback, eight parallel-streaming cases, ten 90–120 Mb/s comparison jobs, profiling and inventory checks.
- [macOS builds](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40294716170): AMD64 and ARM64.
- [CodeQL and native regressions](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40294716243). Successful scans do not prove absence of vulnerabilities.
- [Portable branch checks](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40294716164).

The implementation also passed local native patch application/isolation checks,
97 portable Python tests, Windows PowerShell 5.1/7 sharing-lock retry tests,
Go policy/diagnostic tests and owned Windows runtime checks. The unchanged UI passed
140 frontend tests and 49 browser integration checks, plus type checking,
linting, locale validation and production embedding. Actual Go DTO contracts
are verified in native CI; local fixtures are not substituted for that export.

## Observed failures and corrections

- A generated 2.5-second-per-block peer exposed a two-request startup equilibrium.
  Added measured, paced, capacity-bounded growth and urgent-credit scheduling.
  An intermediate Windows build completed the generated preload in 38.6 seconds,
  where earlier candidates exceeded the 90-second test timeout. Byte checks and
  reader cleanup passed. The paced stress read still fell 13.1 seconds behind
  schedule; this result is not a claim of smooth high-latency playback.
- Windows executable replacement hit a transient sharing lock after shutdown.
  The updater now retries only sharing/lock errors for up to five seconds;
  permission errors remain immediate failures. Native update/rollback checks pass.
- The c03e1e6d Linux delayed-peer comparison completed preload but five adaptive
  playback trials exceeded their read budget. Trained request credit was being
  erased by application-limited rates between preload and a reader. The accepted
  source retains trained credit only while delayed service remains stable and
  fresh; empty queues cannot probe upward, and jitter, snubbing, expiration and
  local/remote queue caps still apply. A native transition regression covers this.
  A local Windows run of the intermediate 86de4321 implementation completed preload in
  38.4 seconds and verified a 24 MiB read with no leaked readers and no remote
  queue-cap breach. Its nominal 12-second read took 31.6 seconds, with 18 waits
  over 250 ms. This extreme generated case still stalls and is not presented
  as sustained real-time playback or a universal improvement over the earlier
  Windows observation.
- Source 86de4321 still failed two Linux adaptive playback trials: the stock
  inter-arrival timeout collapsed to two seconds within batches, then falsely
  snubbed peers whose measured per-request service was 2.5 seconds. The accepted
  source floors that timeout from fresh service evidence plus coarse-tick slack,
  within the configured ceiling and a 15-second measured bound. Disabled,
  untrained and stale cases keep the stock timeout; real stalls still time out.
- One 86de4321 Windows large-piece trial exhausted uncached seek targets in its
  twelve-piece fixture. Large-piece fixtures now contain at least 24 pieces;
  all four seeks still require observed non-residency and new peer requests.
  Cold-seek assertions were not relaxed.

- Two earlier Windows large-piece trials had no native peer listeners and no
  tracker announces. Transport comparisons now preflight a loopback TCP/UDP
  port and require both actual engine listeners before measuring the swarm.
- A management-rate test assumed readiness consumed exactly one request.
  Readiness can require multiple polls. The fixture now enables the two-request
  limit after readiness, requires two successful requests and then HTTP 429.
  Production rate enforcement was not relaxed.

High-latency stock-horizon control timeouts remain failed observations in the
comparison report; they are explicitly distinguished from adaptive candidates,
which must complete. Missing peers, corruption, capacity violations and candidate
failures are never accepted as expected controls.

The final implementation also passed a short owned Windows delayed-peer check:
39.6-second preload, then the complete verified 24 MiB read on its nominal
12-second schedule, zero read waits over 250 ms, maximum individual read wait
0.76 ms, peak request queue 512 against the peer's 512 cap, and zero readers
after cancellation. These are generated-byte transport observations, not a
phone or decoder acceptance claim.
Three additional publication-state tests cover drafts, published releases,
missing releases, API failures and malformed responses. The local portable
suite with those CI-only checks passed all 100 tests.

## Controlled large-piece comparison

The final-source Windows 16 MiB-piece case used a 384 MiB original fixture and
completed all twelve trials (four policies, three rotations each). Median
preload was 22.48 seconds with the capacity control and 11.19 seconds with
combined adaptive requests/recovery. Both groups had zero paced read waits
above 250 ms. Combined cold-seek p95 medians were 467 ms versus 1,199 ms for
the control. These generated transport measurements do not establish universal
improvement across torrents, public swarms, devices or decoders. The public
default remains unchanged.

The final-source Linux delayed-peer comparison completed all six adaptive
trials on their nominal delivery schedule. Each retained one or two individual
read waits over 250 ms. The six stock-horizon control preloads timed out and
remain failed observations. They are not counted as successful playback.

## Publication and private configuration

The tagged workflow verifies source identity and successful receipts before
reusing unchanged runtime/UI evidence. It rebuilds version-stamped binaries and
reruns native, API, Windows lifecycle, functional, packaging and provenance gates.
The release is staged as a draft while the owner-configured Windows package is
prepared and checked. Public downloads contain no private settings or state.
Publication as Latest and container publication followed the private package verification; receipts are recorded below.

## Limits

No fresh representative phone/player matrix, public-swarm acceptance, physical
Windows boot/sleep/network recovery or exact-final multi-hour acceptance is
claimed. Further two-hour tests were skipped at the owner's request. The local
delayed-peer check measured generated-byte transport, not decoded video.
Legacy remains the public default, new scheduling controls remain opt-in, and
1.1.7 security boundaries are retained. Existing owner-approved security
findings remain accepted risks rather than claimed fixes.


## Published Latest: exact tagged receipts

Published 26 October 2026 as [Flow 1.1.8](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.145.Flow-v1.1.8), stable and Latest.
Immutable binary source: `8907097ef3c5b177088a7cc7cd88d012f29f2094`. The release tag was not moved.

- [Tagged workflow 40319222101](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40319222101)
  passed all 25 actual required essential roles. The exact-source
  [security workflow 40319212479](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40319212479)
  also passed. Unchanged full comparisons, browser and macOS evidence came from
  the source-specific successful runs listed above; skipped jobs were not counted.
- All 35 published assets are inventoried; all 34 SHA256SUMS entries agree with
  GitHub asset digests. Four selected downloaded subjects (Windows ZIP,
  BUILDINFO.json, release.json and SBOM.cdx.json) passed their actual checksums.
  The checksum file itself is not a subject of its own inventory. Other platform
  bytes were not all downloaded locally; tagged CI verified their packaging.
- Downloaded BUILDINFO.json and the Windows ZIP passed GitHub attestation checks
  bound to the immutable source, tag ref and release workflow. Windows EXE
  SHA-256: `ef8b6a1b0f307d68c4a58c1cfce46f73c8af5976a8479f8bd7d9ccb2210c6930`.
  Windows ZIP SHA-256: `ac949f08877489cdfec482a5c0c4ddb00f24ef8a7b5fba32f5194e7435edce6e`.
- The private Windows package passed two isolated startups, exact version/source,
  effective settings and restart-persistence checks before publication. It retains
  the owner's 2048 MiB RAM cache, Adaptive profile, five enabled streaming controls,
  approved Lampa origin and diagnostic settings. Its executable matches the public
  package. The tray script differs only in equivalent Windows line endings.
  No private configuration or state was uploaded. No installed server was replaced.
- [Container workflow 40323777601](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40323777601)
  passed actual version/linkage execution on Linux AMD64, ARM64 and ARMv7 and
  published all three. Anonymous registry metadata checks verified their source
  and version labels and identical version/latest index digest:
  `sha256:69c067a97e7af43cba0e84b61c52dfb175c5ea96d05b7211c77d40fe9dc5c128`. These local checks read registry metadata;
  they do not claim a fresh local container playback test.
- The released master source has zero open CodeQL findings after the previously
  owner-approved dispositions. Those dispositions remain accepted risks, not
  additional fixes or proof that the application is vulnerability-free.

The field-validation limits above still apply. No fresh physical-phone, public
swarm or multi-hour acceptance is claimed, and no further two-hour test was run.
