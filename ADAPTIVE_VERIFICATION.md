# Adaptive reliability verification

Scope: the eight accepted improvement areas in
[ADAPTIVE_RELIABILITY_CHECKLIST.md](ADAPTIVE_RELIABILITY_CHECKLIST.md).
The implementation guide is [ADAPTIVE_RELIABILITY.md](ADAPTIVE_RELIABILITY.md).
Published Preview 2 `.5` is unchanged; development is in
[PR #3](https://github.com/gnbk21/TorrServer-LT-Flow/pull/3).

## Source and automated checks

Implementation starts after `c398bbbc`. Source `5dbf9afb` passed Windows network
notification registration/cancellation, frontend type/lint/build and 36 browser
scenarios, native streaming simulations, Linux unit/vet/race/vulnerability/fuzz
checks, Windows x64, both Android targets and Linux armv7. Both macOS targets and
branch checks passed separately. Its full cross run was superseded during
integration tests by `c7ac8bb5`; do not report that cancelled run as a full pass.

Code `c7ac8bb5` also invalidates useful-supply evidence after wake/route
changes retaining the same address, retains recovery hints during the announce
cooldown, and spaces failed as well as successful announce attempts. Its platform
builds, unit/race/fuzz/contracts passed; integration/service was superseded by
`0aab970a`, which additionally invalidates historical ONLINE connectivity after
wake/route changes while preserving a fresh concurrent tracker reply. That run's
platforms and unit/race/fuzz/contracts passed, but integration/service was
superseded by `0d0455c2` after a Windows HTTP rejection defect was reproduced.
`0d0455c2` passed 1,000 anonymous/authenticated request pairs on its actual Windows
executable, but the adaptive harness exposed the same unread-body reset on a
temporary maintenance 503 during queued restart. Final `1e796532` installs the
bounded rejection handling before all early-response middleware, covering
authentication, readiness, maintenance, origin/rate policy and WAF. Successful
responses and streaming bodies are untouched. Superseded full runs are not
reported as passing. Runtime source `1e796532` passed Linux native integration,
all eight platform builds, macOS and branch checks. Run `37672607699` failed its
Windows service test's display-text policy assertion before exercising a crash;
it is not an overall passing run. `1d224e2f` changes verification only: structured
SCM policy readback via the existing Windows library replaces text matching and
retains both the API result and `sc.exe` display. The `server/` source tree is
identical to `1e796532`. Its run `37676476761` was cancelled after Ubuntu's
Azure package mirror stalled for 20 minutes during browser dependency installation;
browser tests had not started. Both macOS targets and branch checks passed for
that revision. CI-only `0b6991f6` adds bounded package download retries/timeouts,
uses the official Ubuntu HTTPS mirror on the disposable runner and separates
browser installation from regressions. No tests were removed. Its `server/`
and `web/` trees are identical to `1e796532`. Final verification CI:

- [Cross/native/service](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37679502820): passed, including all six cross targets, native integration and Windows SCM/update/rollback.
- [macOS](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37679503229): passed, both architectures.
- [Branch checks](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37679502931): passed.

Documentation-only `ebe90299` retains those tested source trees; its cross/native/
service, macOS and branch reruns also passed. The later cache-path correction
below changes server/frontend source and has its own verification record.

The successful Linux `1e796532` reports are retained locally at
`.tools/adaptive-final-1e796532-linux/`: controlled playback, both startup
profiles, discovery, preparation, peer hints, web seeds, adaptive policies,
16 sparse cases and three experimental rate-aware cases. The rate-aware cases
remain opt-in; passing them does not change defaults. The portable SCM probe
build and PowerShell syntax checks passed locally. The final `0b6991f6` Linux
reports are retained at `.tools/adaptive-final-0b6991f6-linux/` and passed the
same scenarios. Windows runner evidence at
`.tools/adaptive-final-0b6991f6-service/` confirms the actual SCM policy: restart
after 15/30/60 seconds, then no action, with an 86,400-second reset period. A
forced process crash recovered under the restricted service account. Restart,
clean stop, uninstall and both console/service authenticated update, integrity
rejection, configuration/startup failure rollback and private migration quarantine
passed. Expected injected failure messages in the rollback reports are test
evidence, not unresolved update failures.

The separate optional `sparse-native-layers` experiment and Docker publication
jobs were skipped by this branch run's conditions; no new container or release
was published. The ordinary Linux gate did run its 16 sparse and three opt-in
rate-aware cases. These generated experiment results do not change defaults.

The Linux job runs actual Go DTO validation, unit/vet/race/fuzz/vulnerability
checks and generated playback/startup/maintenance/preparation/web-seed/adaptive
integration. The Windows service job checks SCM failure policy and actual crash
recovery, in addition to install/restart/stop/update/rollback/uninstall. Native
cache tests cover verified-completion deduplication, pause/resume qualification,
same-address evidence invalidation, reconnect and caches created during recovery.
The optional sparse-layer job records ABI, supplier, graded-deadline and default
versus experimental policy evidence; experiment results do not change defaults.

Local checks passed:

- Go 1.26.8 `go test -count=1 ./flow ./diagnostics ./netchange` on Windows.
- Frontend: 48 unit/component tests; the actual Go DTO contract is a separate
  native CI gate. Type checking, ESLint and production build passed.
- 36 Playwright scenarios: navigation, visibility, responsive sizes, dirty
  settings/revision conflicts, capability session grouping and prior workflows.
- 14 Python fixture/protocol/packaging tests; adaptive harness compilation.
- 500 real Windows split-header/body requests alternating 401/503 passed the bounded
  rejection helper regression; slow senders, large/unknown/chunked/Expect bodies
  and deadline-unavailable wrappers are covered separately. Gin wiring and both
  browser/API challenge behaviors are a native CI test.
- `git diff --check` and review of settings, cache, scheduling, network and access
  callers. All seven maintained locale files include the added strings.

Initial JS is 173.40 kB gzip; the optional HLS chunk is 178.98 kB gzip and remains
lazy. Vite's uncompressed chunk-size advisory remains visible; it is not a build
failure or a claim of real-phone CPU performance.

Failures found during verification were corrected: API package name collision,
reset contract expectations, reused consumed Response test fixture, native test
numeric types, unqualified demand confidence, pause classification, misleading
healthy headings, support-export host identifiers and route/announce recovery.

## Isolated Windows executable checks

The first Windows integration artifact was built from PR source `a64a871b`.
Its embedded identity is the CI merge commit
`MatriX.145.Flow-dev-032726dab2ce39e9afe3ca0d523f412ccb4b0f56`, libtorrent 2.1.2.0.
These are different identities by design; do not substitute the PR head for the
version returned by `/echo`.

Both commands passed with owned loopback servers and disposable settings:

```powershell
python build/adaptive_harness.py --executable .tools/artifacts/windows-adaptive-a64a871b/TorrServer-LT-windows-amd64.exe --output .tools/adaptive-a64a-windows
python build/playback_harness.py --executable .tools/artifacts/windows-adaptive-a64a871b/TorrServer-LT-windows-amd64.exe --fixtures .tools/controlled-fixtures --output .tools/adaptive-a64a-windows-profile --cases fast --duration 120 --profile
```

The adaptive report verified hot settings/conflict rejection, file-scoped
anonymous Range/HEAD and tamper rejection, strong playback policy, idle queue
cancel/process-crash recovery, token revocation on restart, unavailable adapter
suspension with verified local playback, origin/rate policy, corrupt JSON
last-known-good recovery, retained originals and explicit repair.

The playback report verified 256 initial cross-file requests, 107 profiling
cycles and 24 further cycles with 24 runtime samples. Server/API readiness
was 528 ms; this is neither playable-buffer time nor decoded time to first frame. Over the sampled interval
(80–197 seconds after test start), RSS changed from 152,027,136 to 156,393,472
bytes, handles from 470 to 479 and goroutines from 18 to 17; cache residence
remained about 31.8 MiB. These short observations do not establish leak absence.
Warm expiration was observed. Every owned server exited; personal server state,
router and firewall were not changed.

Go CPU/heap/allocs/mutex/block/goroutine profiles and a short execution trace are
retained in `.tools/adaptive-a64a-windows-profile/fast/`. The 30-second Windows Go
CPU profile attributes most samples to `runtime.cgocall` through the native
alert wait. Blocking native wait attribution is not evidence of 98% actual CPU
use or proof of a C++ bottleneck. The short mutex delta had no samples, which
does not establish absence of contention across the workload. No speculative
native tuning or PGO adoption follows from these profiles.

Intermediate source `c7ac8bb52a310033614e2bc6b23ddc9d69df0afb` Windows artifact:

- Local path: `.tools/artifacts/windows-adaptive-final-c7ac8bb5/TorrServer-LT-windows-amd64.exe`.
- Embedded version: `MatriX.145.Flow-dev-87c339db3e13c4fd17f404408364b7b31f416124`.
- Libtorrent: `2.1.2.0`; CI merge identity is `87c339db`, not PR head `c7ac8bb5`.
- SHA-256: `8ac1e21fbfa32abf4f653f42defa0bef68130ea2672cfa197349b2554ac663bc`.
- `build/adaptive_harness.py` passed again with this exact executable. Report:
  `.tools/adaptive-final-c7ac8bb5-windows/report.json`.
- A separate two-swarm native scheduling check passed concurrency one/queued
  admission, releasing the slot on pause and hot increasing concurrency to two.
  Holding a real third HTTP reader suspended both jobs with `PLAYBACK_PRIORITY`;
  closing it resumed both. The third job's verified data stayed ready. Local script:
  `.tools/validate-adaptive-scheduler.py`; final report:
  `.tools/adaptive-final-c7ac8bb5-scheduler-playback/report.json`.

The earlier profiling results above retain their own artifact identity. The new
verification executable is separate from the user's usual executable and state.

Superseded source `0aab970a2cc55b04dbf03437b3b47e095e5717ac` Windows artifact:

- Local path: `.tools/artifacts/windows-adaptive-final-0aab970a/TorrServer-LT-windows-amd64.exe`.
- Embedded version: `MatriX.145.Flow-dev-8e90a977f64b04614c4658247df0c956d5be88e5`.
- Libtorrent: `2.1.2.0`; embedded CI merge identity is `8e90a977`.
- SHA-256: `71e37a37ccc66f8a7604e8f15a98795d46686c47f61e242305505ce460739963`.
- The adaptive rerun failed with Windows socket error 10053 on an anonymous
  settings POST. The server stayed running and shut down cleanly. A focused
  replay reproduced error 10054 after 24 successful requests. A minimal standard
  Go HTTP server also reset two of 200 requests when closing with unread body
  bytes; consuming the small body passed 200 of 200. This is a reproduced
  response-delivery defect, not an unexplained host-transient claim.
- Source `0d0455c2` consumes only fixed, known bodies up to 4096 bytes before
  returning from an unauthorized close request, with a 250 ms read deadline.
  It does not consume uploads, unknown/chunked bodies or Expect bodies. Auth
  decisions/challenges are unchanged. Test helpers also close rejected responses.
  Local evidence is retained under `.tools/adaptive-final-0aab970a-windows/`,
  `.tools/adaptive-final-0aab970a-auth-diagnostic/` and the ignored
  `.tools/check-auth-baseline.py` / `.tools/auth-baseline.go` diagnostic fixture.

Intermediate `0d0455c2` artifact:

- Embedded version: `MatriX.145.Flow-dev-24507c8d8a3df902a6d7d4343d77fb821c1e09bc`.
- SHA-256: `93c3b2f5f52584a2a348269c7f52e3f4c779699208f4b0879a6d80165dfc8e94`.
- Authentication diagnostic passed 1,000 anonymous 401 requests interleaved with
  authenticated state reads. Report: `.tools/adaptive-final-0d0455c2-auth-diagnostic/report.json`.
- Full adaptive harness failed with socket reset 10054 on a state read during
  queued idle restart. Logs confirm maintenance/reconnect, not a process crash.
  Report: `.tools/adaptive-final-0d0455c2-windows/report.json`. Final `1e796532`
  covers this additional early-response path.

Final source `1e7965328d43fb2b1de84118303bf81ee7da53af` Windows artifact:

- Local path: `.tools/artifacts/windows-adaptive-final-1e796532/TorrServer-LT-windows-amd64.exe`.
- Embedded version: `MatriX.145.Flow-dev-562696283a2de1a0d4f7c090f2f6c888cd6a2b1e`.
- Libtorrent: `2.1.2.0`; embedded CI merge identity is `56269628`.
- SHA-256: `67d484d01f3cbb07cd6fd545767eb338288ff12dcb87412b1e6c4f7976d49c7a`.
- `build/adaptive_harness.py` passed with this exact executable. Report:
  `.tools/adaptive-final-1e796532-windows/report.json`. It covers hot settings and
  conflicts, capability scope/Range/HEAD/strong policy, queued idle apply/cancel
  and process-crash recovery, missing-interface cached local playback, origin/rate
  policy and corrupt settings preservation/repair.
- A separate exact-executable stress run passed 1,000 anonymous 401 POSTs,
  each followed by an authenticated state read, and 1,000 maintenance 503 POSTs.
  No connection errors occurred; the owned process stopped with exit zero.
  Script: `.tools/check-anonymous-settings.py`; report:
  `.tools/adaptive-final-1e796532-rejection-diagnostic/report.json`.

Reproduce the full acceptance with a new output directory:

```powershell
python build/adaptive_harness.py --executable .tools/artifacts/windows-adaptive-final-1e796532/TorrServer-LT-windows-amd64.exe --output .tools/adaptive-acceptance-new
```

## Live disk-cache incident and path validation (8 October 2026)

The active Windows server had literal surrounding quotation marks in its saved
disk-cache folder path. The console reported `Access is denied`, native status
showed upload-only mode and verified cache remained empty despite connected
peers. The intended unquoted folder existed and a temporary write/flush/delete
probe succeeded. Removing only those quotes through the revision-checked
settings API and applying the engine restart restored cache writes. Saved and
effective cache size remained 2048 MiB; the selected Balanced profile was retained.

The retry reached its first useful block in 410 ms, completed bootstrap/probing
in about nine seconds and its startup prebuffer in 24 seconds. These measure
server stages, not decoded time to first frame. The user confirmed playback
started. No further disk-write error or upload-only mode was observed.

Subsequent intermittent playback had a separate visible constraint: the probed
media bitrate was about 90 Mbps (11.2 MB/s), useful delivery sometimes fell below
that demand, contiguous data drained and server reads waited for pieces. Wire
traffic and connected-peer counts alone did not establish playable supply.
Pausing in Just Player left downloads enabled; the HTTP read position then had
about 74 seconds of contiguous data ahead. After resuming, the user reported
smooth playback; six samples over 25 seconds recorded no recent read stalls,
with contiguous ahead data ranging from 37 to 102 seconds. This short observation
does not establish sustained public-swarm throughput, eliminate future stalls
or measure the player's decoded buffer. Anonymous numeric samples are retained
locally in `.tools/live-preload-20261008/`.

The follow-up guard rejects active disk-cache paths beginning or ending with a
literal double quote, including quotes surrounded by whitespace. The API's
shared settings validator covers planning, saving and queued application; the
web form displays a field correction in all seven languages before sending a
plan or save. Inactive disk settings and existing unquoted absolute, UNC and
relative paths remain compatible. Paths are never silently rewritten. Existing
invalid stored configurations follow the established last-known-good recovery
and explicit repair policy.

Local verification passed the portable Go path cases, 49 frontend unit/component
tests, TypeScript, targeted ESLint and the production build with original
dependency notices. A focused Playwright regression verified the inline error,
field focus, no plan/save for quoted input and successful save after correction.
Initial JS is 173.44 kB gzip; HLS remains lazy. Full native settings integration
and the complete platform/browser suite require the follow-up commit's CI;
earlier passing CI does not verify this changed source. The running executable
has the repaired configuration but predates the new validation guard.

## Requirement reconciliation

The original Flow specification's architecture, two-stage startup, one existing
cache, HTTP Range compatibility, explicit swarm profiles, warm grouping and
native deadlines are preserved. New supply evidence qualifies sustainability
instead of treating aggregate wire traffic as playable supply. Risk targets are
bounded by the existing controller and configured limits. Immediate reader and
container reservations remain protected under the new soft global budget.

The Modern Web specification's server-authoritative settings, dirty protection,
restart warnings, visibility-aware polling, unknown evidence, seven languages,
relative API routes, accessibility layer and embedded/lazy assets remain in the
execution path. The new settings state/plan/apply API is additive; revision-less
legacy requests remain compatible. Capability links retain the existing `ss`
group and use the existing file/Range handler. Management access restrictions
and source binding are optional; defaults retain Lampa/Just Player behavior.

Free sources and verified episode preparation are retained. No paid provider,
replacement engine, second large media cache, automatic swarm switching, QUIC,
ML controller, arbitrary socket tuning or unconditional PGO was introduced.
No required implementation was replaced merely to simplify the architecture.

## Exact remaining acceptance gates

- A physical Windows/VPN disconnect trace covering TCP, UDP/uTP, DHT,
  HTTP/HTTPS/UDP trackers, DNS and mirrors has not been captured. Interface
  binding pauses when unavailable and has no unbound fallback, but OS DNS is
  not bound and source addresses are not firewall rules. Leak-protection and
  DNS-verification flags remain false. A stronger guarantee requires these
  measurements and, if needed, a separate explicit OS enforcement design.
- Representative Android/Just Player/Lampa playback, decoded startup, real
  forward/backward seeks, background/resume and weak public swarms need the
  user's device/media. Desktop fixture results do not pass those checks.
- Multi-hour representative resources and physical Windows boot/sleep/network
  recovery are unperformed. Runnable procedures remain in MEASUREMENTS.md,
  WEB_DEVICE_CHECKLIST.md and the adaptive guide; an overnight generated run
  uses `--duration 28800` and is distinct from decoded phone playback.
- Native C++ OS profiling needs a representative workload and WPR/WPA or perf
  capture. PGO remains off pending representative comparative evidence.
- Authenticode requires a publisher signing identity. No substitute signature
  or signed-binary claim is made.

`RELEASE_ACCEPTANCE.json` therefore remains `stable_approved: false`; legacy
sources remain available and stable/Modern Web 1.0 approval is not asserted.
