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

- [ ] Publish the engine handle safely across reconnects; regression/race checks.
- [ ] Bounded HTTP read buffering, preserving logical playback progress,
  cancellation, partial availability, seeks and Range semantics; RAM/disk checks.
- [ ] Reconcile tracker/session and DNS improvements with existing Flow fixes.
- [ ] HTTPS certificate reload, renewal and authenticated management, preserving
  private-key permissions, user-owned certificates and revisioned settings.
- [ ] HTTPS-only / HTTP-media modes and bounded listener handling, including
  internal media URLs, redirect behavior and graceful shutdown.
- [ ] Modern Security certificate controls and all supported translations.
- [ ] GStreamer one-frame cue-boundary tolerance, including invalid/fractional
  rates and parser regression checks.
- [ ] GStreamer cold-source warmup and one retry, with cancellation, shutdown,
  shared probes and unavailable-runtime handling checked.
- [ ] README, release notes, version, source/license/provenance and acceptance
  record reconciled with the actual implementation.
- [ ] Automated web/native/platform/security/streaming/package gates pass.
- [ ] Published assets, installed package and container verified; full release
  is public, not a prerelease, and returned by GitHub's latest-release endpoint.

## Remaining field validation

Broader physical Android/player coverage, multi-hour resource measurements, and
physical Windows boot/sleep/network recovery remain separate field checks until
there is evidence for each. Previously reported smooth episodes are useful
playback evidence, but do not establish those broader results.
