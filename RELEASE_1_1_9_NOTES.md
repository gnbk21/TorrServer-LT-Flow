# TorrServer-Flow 1.1.9

Version: `MatriX.145.Flow-v1.1.9`.

## Preloading and streaming recovery

- Bound Adaptive Streaming's optional startup reserve by the estimated cost of
  whole torrent pieces. After the ordinary media head and required index are
  verified, this extra reserve may delay startup for at most ten more seconds
  (plus the scheduler tick). The normal bitrate-based startup target and required
  container data still have to be verified; total startup is not capped at ten
  seconds.
- Let the existing optional frontier recovery mechanism cover missing pieces in
  the active preload phase. Reader and preload demand have independent short
  leases, so finishing preload does not erase an active reader's recovery demand.
- Recover a nearly complete block when its original peer stops delivering
  payload. Partial completion alone no longer protects a stalled request
  indefinitely. Fresh service evidence, duplicate-byte budgets and request
  ownership limits still apply.
- Prevent adaptive request credit from repeatedly growing on already overloaded
  peers. Urgent bootstrap exceptions also respect a bounded estimated queue time;
  existing requests drain normally.
- Reject already-cancelled playback requests before attaching cache readers or
  recording playback activity. Improve startup reserve diagnostics with the
  requested size, extra-wait budget and whether that budget limited or expired.

## Compatibility and defaults

Legacy remains the default swarm profile. Capacity-aware requests, adaptive
urgent horizons, frontier recovery, adaptive peer requests and container burst
hints remain opt-in. The startup reserve correction applies to Adaptive
Streaming. Settings are preserved; public packages contain no personal config.

Flow retains libtorrent 2.1.2, original media bytes, the shared cache and its
existing Range API. All 1.1.7 security boundaries remain in place. Browser-based
Lampa needs its exact trusted origin in **Settings → Management origins**.

## Verification and limits

Publication requires the full automated release gates, including all eight
platforms, native queue/deadline simulations, security and API checks, generated
streaming comparisons, Windows lifecycle tests, reproducible packaging and
provenance. New regressions exercise stalled partial blocks, independent recovery
leases, optional-reserve expiry and missing required index data.

Controlled tests use generated local media and peers. They do not prove a speed
increase for every public swarm or player. Representative final-version phone
playback, physical boot/sleep/network recovery and multi-hour resource acceptance
remain unfinished and are disclosed under the owner's publication policy. No
additional two-hour test was run. Windows binaries are not Authenticode signed;
accepted security dispositions are not fixes or proof of zero vulnerabilities.
Flow cannot sustain playback above the useful supply available from its peers.
