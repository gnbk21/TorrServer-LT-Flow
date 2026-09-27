# Flow audit — 27 September 2026

## Assessment

Flow is a usable development fork, with working core playback features and
positive short phone-playback feedback. It is **not yet a validated Flow 1.0**.
Passing tests cannot establish absence of bugs, superiority over upstream, or
multi-hour stability. This review covers the Flow changes, their integration
points, settings/UI, build/release pipeline, dependency advisories, and a brief
live-process observation. It is not a line-by-line security audit of all inherited
TorrServer-LT, libtorrent, or third-party code.

The starting revision was `02a35ef7`. Upstream `master` was `f1eb076c` and was
already an ancestor of `develop`: 16 fork commits ahead, zero upstream commits
missing. No merge conflict or uncommitted change existed before the audit.

## Defects addressed

- **Stale delivery position:** diagnostics previously advanced the position only
  after an HTTP request ended, and omitted requests without a Range header.
  Position now advances after each successful body write using the actual HTTP
  response. Older requests cannot overwrite a newer seek; metadata probes,
  multipart framing and error bodies do not become media progress. This is the
  server delivery position, not Just Player's decoded presentation time.
- **Misleading diagnostics:** fixed the extra increment on the API's one-based
  file index, show unknown estimates explicitly, and cancel/serialize polling.
- **Settings failures:** a missing/null `sets` object now returns 400 before any
  engine restart. The settings dialog stays open and displays save errors.
- **Credential logging:** stopped logging complete CLI proxy URLs, proxy parse
  errors that can repeat those URLs, and torrent-add links containing passkeys.
  Other inherited access/debug logs can still contain private playback URLs;
  logs are not automatically safe to publish.
- **Service startup:** SCM readiness now follows this process's own bound
  listeners independently of engine initialization. Start/stop engine lifecycle
  operations are serialized. Telegram initialization follows the local listener.
  Installation resolves relative file arguments before SCM changes directories.
- **Artifact consistency:** every platform build consumes freshly generated web
  assets. Previously the independent web job could succeed while native binaries
  embedded an older committed bundle. Development versions identify Flow and the
  build revision while retaining the Android-compatible `MatriX.145` prefix.
  Tagged release packaging includes the tray companion.
- **Dependencies:** updated Go from 1.25.7 to 1.26.8 and patched affected
  `x/crypto`, `x/image`, `x/mod`, `x/net`, `x/text`, and `quic-go` dependencies,
  including required related module updates. CI includes source reachability
  scanning and race tests.
- **Documentation:** removed stale future-tense descriptions of implemented
  service/tray work and clarified inherited authentication limitations.

## Specification reconciliation

Sections refer to the supplied Final Engineering Specification & Development Plan.
“Implemented” describes code integration; it does not waive an outstanding
environmental acceptance test.

| Plan area | Current state and remaining work |
| --- | --- |
| Purpose, compatibility, scope and architecture (§1–6) | Existing libtorrent engine, cache, API and settings retained; Flow overlays startup/session/network state. No second cache, custom peer killer or assumed Just Player request sequence. Not every named state is exposed as a distinct transition. |
| Baseline and observability (§7–8) | Baseline binary and comparison procedure exist, but no reproducible A–F benchmark dataset/report. Most delivery, cache, Range, startup and reconnect counters exist. Missing dedicated container hint, active downloading peer count, underrun counter, resolver/announce timing, listener/network-ready latency and network retry count. Time to playable frame must be measured by a client. |
| Startup and estimation (§9–10) | Bootstrap head plus upstream tail/MP4 refinement, asynchronous bounded probing, provisional target, source/confidence and observed EWMA are integrated. Metadata is held for the active probed file; a persistent per-file metadata cache is absent. Controlled delayed/failed probe and container tests remain incomplete. |
| Cache, RAM and controller (§11–14) | One existing piece cache, bounded forward window, contiguous buffer calculation, presets, sustainability and buffer warning. Delivery-rate estimates remain approximations of player consumption, especially for bursty or parallel readers. Full VBR/weak-swarm validation is pending. |
| HTTP, warm sessions and seek (§15–20) | ServeContent retains Range/ETag semantics; no forced connection close. Cancellation, warm reservation/expiry, seek cleanup and server TTFB instrumentation exist. Synthetic unit tests and a short phone seek/resume passed historically. Actual-client trace fixtures, evicted backward seek and near-timeout reconnect matrix remain outstanding. |
| Peer tuning (§21–25) | Named profiles and bounded custom controls exist; no arbitrary slow-peer disconnection or new AIO tuning. Legacy remains default because profiles have not been comparatively benchmarked. |
| DNS, proxy and DHT (§26–30) | Existing system/libtorrent DNS and proxy mechanisms retained; bounded network retries and protocol/host tracker summaries added. No persisted hostname-to-IP rewriting. No application DoH resolver or UDP endpoint cache; these are evidence-dependent extensions. Transport fault tests are incomplete. |
| Windows service/network/tray (§31–34) | Service commands, delayed autostart, wildcard listener, polling/address-change reannounce and separate tray implemented. Polling is the fallback rather than native Windows address notifications. Boot-before-DHCP, suspend/resume, long service shutdown and interactive tray checks remain outstanding. SCM currently uses its default LocalSystem account; a dedicated restricted account/installer is future hardening. |
| Performance policy (§35–37) | No speculative pools or socket-buffer changes added. Profiling and long resource measurements are still needed before further tuning. |
| Providers (§38–42) | Optional provider/source abstraction and integrations are absent. P2P is the only Flow source. The provider roadmap phase is not complete. Credentials and account-dependent integrations were not invented. |
| Settings and UI (§43–47) | Flow controls, profiles, cache presets and live diagnostics wired to the API; migration/default normalization and feature switches exist. Flow labels remain English; provider controls and several advanced diagnostic fields are absent. |
| Acceptance and tests (§48–54) | Unit/HTTP/cache/network tests exist, but no complete controlled-peer harness, legal synthetic container corpus, captured phone traces, repeated multi-hour memory cycles or RW-01–18 results. Tests involving optional DoH/providers are inapplicable until those features exist. |
| CI, branches, upstream and license (§55–58) | Cross-platform CI, `develop` integration, stable `master`, upstream remote and GPL source retained. Draft integration PR remains open. Upstream installers/release manifest still point upstream; README directs users to fork development artifacts. No stable release is claimed. |
| Roadmap and success criteria (§59–74) | Code spans several milestones, but baseline/benchmark exit criteria were skipped earlier. Flow 1.0 acceptance is not satisfied. Post-1.0 predictive buffering, automatic tuning, history and Prometheus export remain candidates; rejected speculative features remain excluded. |

## Live process observation

The observed Windows process used roughly **4.16 GiB resident memory**, had four
reported peers, and reported about 657 seconds buffered with zero instantaneous
download. The old delivery-position bug makes that buffer reading unreliable.
A single sample cannot diagnose a memory leak or establish whether the cache
budget was appropriate. The saved configuration beside this executable specifies
a 4 GiB RAM cache, 95% read-ahead, Flow enabled and the Balanced profile; this is
consistent with a large cache footprint but is not a memory-stability test.
No matching TorrServer application-error event was found in the preceding eight
hours of the Windows Application log. The exit cause remains unknown.
The process was no longer running and port 8090 was
closed at the later check. The audit did not restart it, change its settings,
or stop the user's playback.

## Security and dependency findings

- `govulncheck` on the old stripped executable reported 42 advisories across six
  modules and the Go standard library. Binary matches are broader than a source
  reachability analysis; this does not prove 42 exploitable Flow vulnerabilities.
- After module updates, the module-only scan reports the unmaintained
  `golang.org/x/crypto/openpgp` advisory, which has no patched version.
  `go mod why golang.org/x/crypto/openpgp` reports that Flow does not need this
  package. The CI source scan is the more precise check.
- Yarn's original dependency tree reported 267 affected dependency occurrences.
  After updating axios's form-data to 4.0.6 and aligning Babel core at 7.29.6 to
  repair the broken test runner, 265 remain
  (10 critical, 142 high, 101 moderate, 12 low), not 265 independent exploitable
  runtime defects. Most paths are CRA 4/Webpack/Jest/ESLint build tooling. The
  remaining react-query/broadcast-channel paths require
  platform/reachability interpretation: Node-only code is not equivalent to
  browser runtime exposure. This audit does not claim a clean JavaScript audit.
  Replacing the obsolete build toolchain needs a separately verified migration,
  not blind transitive dependency overrides. The narrow Babel override resolves
  CRA's exact 7.12.3 pin conflicting with its installed newer syntax plugin;
  production compilation and component tests verify it. Yarn still warns about
  that exact-version override and inherited peer dependency declarations.
- Go scanning does not cover vulnerabilities in linked C++/OpenSSL code.
- Management authentication does not fully gate inherited known-torrent playback
  routes. Use a trusted LAN/VPN and private-network firewall rules. Restricting
  playback authentication needs an explicit compatibility design for Lampa and
  external players.

## Verification record

Local Flow/ffprobe tests, HTTP response progress regressions and service argument
tests pass. Web production build and lint were run; final CI and executable
verification results will be recorded after completion of this audit's build.

## Recommended next work, in order

1. **Controlled regression harness and recorded phone traces:** synthetic peers,
   MP4/MKV fixtures, cancellation/reconnect/seek, network interruption, and repeated
   cache/RSS measurements. This makes future performance changes defensible.
2. **Modernize the web build and dependency maintenance:** remove obsolete CRA 4
   tooling, preserve the embedded UI output/API, and add regular advisory review.
3. **Complete telemetry:** listener/network timings, active downloading peers,
   explicit underrun events, rolling rather than lifetime risk windows, and a
   credential-redacted diagnostics export. Keep server TTFB separate from phone
   time to first frame.
4. **Windows packaging and service hardening:** restricted service account,
   protected install directory, private-LAN firewall setup, signed/checksummed
   artifacts and a verified update/rollback path.
5. **UI quality of life:** translated Flow labels, inline profile explanations,
   clear per-setting validation and current build identity. Add DNS/provider
   features only after an observed need and controlled transport tests.

Do not increase cache sizes or choose more aggressive profiles simply because
their numbers are larger. The missing evidence is about playback continuity,
resource stability and recovery, not peak download speed.
