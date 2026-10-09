# TorrServer-Flow v0.2.0-preview.7

This preview integrates the streaming, recovery and diagnostic improvements
developed since Preview 2 `.5`. It streams original torrent bytes through the
existing rolling cache; no paid provider or whole-episode download is required.

## Changes

- Security maintenance: Go 1.26.9 and `golang.org/x/net` v0.60.0 replace the
  vulnerable previous pins, together with required transitive module updates.
  These versions address the 8 October Go HTTP/TLS/template disclosures.
- Experimental **Adaptive Streaming** profile with bounded reserves based on
  qualified delivery deficits. Partial slowdowns count as well as outages.
- Three independent, disabled-by-default experiments: peer request-capacity
  limits, demand-sized urgent horizons and resident MKV/MP4 container burst hints.
- Bounded asynchronous native storage I/O, reusable disk handles, ordered piece
  operations and shutdown/removal fences. Fixes include Windows flush access,
  portable storage errors and a resident verification/copy race.
- Useful-delivery, request-age, contiguous-frontier and storage-latency diagnostics;
  authenticated phone/LAN transfer testing and seven-language controls.
- Startup peer retention, reusable discovery state, sparse-source diagnostics,
  exact-byte web seeds and optional verified preparation.
- Recoverable settings, background-work admission, network recovery, organized
  console reporting and restricted Windows service recovery/update rollback.
- Disk-cache paths containing shell quotes are rejected with corrective help.

## Defaults and evidence

**Legacy remains the default.** Adaptive is optional; all three new experiments
remain off unless selected. Controlled performance results are mixed, so this
release makes no claim that Adaptive consistently beats Legacy.

The pre-release code passed all eight platform builds, native/race/fuzz/browser
checks, Windows service recovery, 72 high-bitrate trials with 120-second requested
workloads, final Windows recovery/LAN checks and 16 earlier matched Windows trials. Tagged binaries and
packages are independently checked by the release workflow before publication.
The user also reported one complete episode without stutters using the verified
development build with Adaptive, all three experiments and 2048 MiB RAM cache.
A subsequent logged episode reached preload readiness in 9.2 seconds; its
recorded first-byte responses were 0–3 ms. These are limited session observations,
not a universal playback or comparative-performance guarantee.

## Install and upgrade

For Windows, download
`TorrServer-Flow-windows-amd64-MatriX.145.Flow-v0.2.0-preview.7.zip`.
The standard executable works with stock Just Player through Lampa. Optional
`-gst` binaries require GStreamer. Platform packages include original dependency
notices; Windows also includes the tray and managed install/update scripts.

Stop playback and back up the existing state before upgrading. Keep executable
and state directories separate, use the same state path and launch arguments,
and retain the previous executable for rollback. Managed installations may use
`Update-Flow.ps1 -Channel preview -RequireAttestation`. Verify package SHA-256
against `SHA256SUMS`; `BUILDINFO.json` identifies the exact source and binaries.
GitHub build provenance is separate from Windows Authenticode publisher signing.

## Limits

- A subsequent LT 1.2.1 source comparison identified a remaining race risk in
  Flow's global engine-pointer publication during reconnect. It is not a
  demonstrated ordinary-playback failure; a targeted fix and regression are
  needed before stable acceptance. See the [evaluation](https://github.com/gnbk21/TorrServer-LT-Flow/blob/develop/UPSTREAM_LT_1_2_1_EVALUATION.md).
- Stable Flow / Modern Web 1.0 acceptance remains incomplete: broader real-phone
  playback, multi-hour resource convergence and physical boot/sleep/network
  recovery still need acceptance evidence.
- Configured media cache size is not a process-memory ceiling. RAM caching also
  requires native/Go buffers and allocator overhead; disk caching is available.
- Range traces are bounded in-memory diagnostics and disappear at shutdown.
  Opt-in diagnostic history persists startup/first-byte/sparse events, not every
  stream's complete byte-range history or Android decoder events.
- The separate experimental Flow Player remains a CI artifact, not an included
  stable Android app. Stock Just Player remains supported.
- Cross-swarm/v2 reuse is deferred pending identity, mapping and verification
  support. Router SQM/upload shaping are conditional operator procedures.
- Deploy on a trusted private LAN or VPN. Management authentication alone does
  not protect every inherited playback route; source-interface binding is not
  verified VPN leak protection. Executables remain unsigned.

See the [high bitrate guide](https://github.com/gnbk21/TorrServer-LT-Flow/blob/MatriX.145.Flow-v0.2.0-preview.7/HIGH_BITRATE_GUIDE.md),
[verification evidence](https://github.com/gnbk21/TorrServer-LT-Flow/blob/MatriX.145.Flow-v0.2.0-preview.7/HIGH_BITRATE_IMPLEMENTATION.md)
and [installation/rollback guide](https://github.com/gnbk21/TorrServer-LT-Flow/blob/MatriX.145.Flow-v0.2.0-preview.7/DISTRIBUTION.md).

Publication, downloaded-package, live updater, browser and container checks are
recorded in the [release verification](https://github.com/gnbk21/TorrServer-LT-Flow/blob/develop/PREVIEW_2_7_VERIFICATION.md).

The earlier `preview.6` tag failed the fresh reachable-vulnerability scan and
was not published. Its tag is retained as failure history; this corrected
`preview.7` is built and verified independently. Security scanning remains a
required publication gate. See the [Go release history](https://go.dev/doc/devel/release)
and [HTTP/2 advisory](https://pkg.go.dev/vuln/GO-2026-6617).
