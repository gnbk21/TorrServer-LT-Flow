# LT 1.2.1 / 1.2.2 integration and first full release

## Scope and acceptance

Integrate the useful upstream changes without replacing Flow's newer Go,
libtorrent, dependencies, startup policy, cache, or modern interface. Preserve
Legacy as the default swarm profile and keep experimental options disabled.
Upstream attribution and comparison are recorded in
`UPSTREAM_LT_1_2_1_EVALUATION.md`.

On 2026-10-09 the owner authorized publication as Latest after automated gates
pass, with unfinished phone, long-running resource, and physical Windows
recovery checks disclosed. These field checks must not be reported as passed.

## Implementation and verification checklist

- [x] Publish the engine handle safely across reconnects; regression/race checks.
- [x] Bounded HTTP read buffering, preserving logical playback progress,
  cancellation, partial availability, seeks and Range semantics; RAM/disk checks.
- [x] Reconcile tracker/session and DNS improvements with existing Flow fixes.
- [x] HTTPS certificate reload, renewal and authenticated management, preserving
  private-key permissions, user-owned certificates and revisioned settings.
- [x] HTTPS-only / HTTP-media modes and bounded listener handling, including
  internal media URLs, redirect behavior and graceful shutdown.
- [x] Modern Security certificate controls and all supported translations.
- [x] GStreamer one-frame cue-boundary tolerance, including invalid/fractional
  rates and parser regression checks.
- [x] GStreamer cold-source warmup and one retry, with cancellation, shutdown,
  shared probes and unavailable-runtime handling checked.
- [x] README, release notes, version, source/license/provenance and acceptance
  record reconciled with the actual implementation.
- [x] Automated web/native/platform/security/streaming/package gates pass.
- [ ] Published assets, installed package and container verified; full release
  is public, not a prerelease, and returned by GitHub's latest-release endpoint.

## Remaining field validation

Broader physical Android/player coverage, multi-hour resource measurements, and
physical Windows boot/sleep/network recovery remain separate field checks until
there is evidence for each. Previously reported smooth episodes are useful
playback evidence, but do not establish those broader results.

## Upstream source and adaptations

Reviewed the cumulative LT changes through tag `MatriX.146.LT-1.2.1`
(`922d8370c5b99d1b8ef47b36e727600b709460bd`) and the following
`MatriX.146.LT-1.2.2` tag (`9a08d66c0f23166895c13b930d2d01ee3c4b835e`).
Retained GPL-3.0 attribution and original notices. Relevant upstream changes:

- HTTP buffering: `32170776491adea2e2187726219544d6c85d8ead`, adapted to
  Flow's logical consumption, shared cache and reader scheduling.
- HTTPS modes and certificate management:
  `8a2173280ac75070e77708280273e2025282eb57`, adapted to revisioned settings,
  private-key ACLs, restricted process-secret loopback access and modern UI.
  HTTPS-only startup ignores the unused HTTP port; Telegram probes also use
  the internal listener. Explicit regeneration rolls back on a failed save,
  and the TLS loader cannot adopt an uncommitted identity.
- GST cue tolerance: `60217f6813fef0e4429460630a2c18ecb50fa063`, with
  checked arithmetic and fractional-frame regression coverage.
- GST cold-source retry: `fbff205942d75d8266350740f295e732da76daf0`, with
  request cancellation, shared-probe ownership and joined shutdown workers.
  Browser discovery waits for the bounded retry before HLS manifest loading;
  its 120-second request budget covers the server's 111-second maximum, while
  closing the player cancels the query. Actual-runtime and subprocess lifecycle
  fixtures are isolated so installed discovery commands cannot replace fixtures.
  Native FFI dereferences retain typed pointer provenance for race/checkptr
  validation; integer handles remain opaque and are not dereferenced.

Flow already protects native session lifetime and rechecks torrent addition
under the engine lock. The helper's engine pointer now uses atomic publication.
Its tracker refresh already has cancellation, generation fencing and bounded
network work; existing tests cover invalidation and failure preservation.
The OS/VPN resolver policy remains nonblocking. Upstream dependency downgrades,
legacy web components and generated assets were not copied into Flow.
