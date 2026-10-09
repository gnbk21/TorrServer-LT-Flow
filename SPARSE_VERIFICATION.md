# Sparse-streaming verification

Updated 5 October 2026. These are development-build results, not a new release
or proof of public-swarm or Android decoder performance. The requirements and
remaining gates are tracked in [the implementation ledger](SPARSE_IMPLEMENTATION.md).

## Final Windows validation

The final `3400e54a` Windows artifact reports
`MatriX.145.Flow-dev-803f0502d92744ba664aa4385bb2df1f66969fa3`, libtorrent
`2.1.2.0`. GitHub's Git API confirms the synthetic PR merge's source tree matches
the tested branch exactly: `ec52ea496d6dfbd21d6647614db49c4b5bd931b1`.
Artifact `11336019881` passed CRC and GitHub SHA-256 verification:
`1f9f4092bdff820f17ff677dce76cba642a19e79f679d812c73c19656327199b`.
Executable SHA-256:
`06a20fca8986ff3a57f6513ea7c034e6a02201d6a995afff5cecfc8887ffc93e`.
This verifies source/artifact identity; it is not Authenticode signing.

Actual Windows native checks passed both deadline-setting values through the
settings API, missing-piece diagnostics during a blocked seek followed by exact
Range bytes, complete preparation retention/restart/corruption/offline/cleanup/
quota coverage and all five native mirror scenarios. The matrix used owned
processes, fresh state and loopback sources; no user library or playback session
was used. Raw reports: `.tools/windows-340-verification/verification.json`,
`seek/diagnostic-seek/report.json`, `preparation/report.json`, `mirrors/report.json`.

The usual `.tools/artifacts/windows/TorrServer-LT-windows-amd64.exe` was replaced
only after confirming it was stopped, then its hash and version were checked.
Previous `8e1478a8` executable backup:
`.tools/artifacts-backups/windows-0a763741-TorrServer-LT.exe`.

## Final deadline-policy revalidation

The final Windows build passed **18/18** cases: below consumption, complementary
partial peers and late HAVE, three repetitions per deadline setting, varied mode/
case order and fresh state. All **54 Range checks** returned exact bytes. No
duplicate payload was observed in these runs. Settings API checks confirmed both
actual flag values. Complete compact observations, fixture/executable identities
and the raw-report hash are in
[SPARSE_POLICY_VERIFICATION.json](SPARSE_POLICY_VERIFICATION.json).

Seek response medians, milliseconds, three observations per cell:

| Case | Default graded ramp | Rate-aware experiment |
| --- | ---: | ---: |
| Below consumption | 916.1 | 7019.0 |
| Complementary partial peers | 1337.5 | 1913.0 |
| Late HAVE | 2663.5 | 2371.6 |

The below-consumption paced client's total read-wait median was 15869.4 ms with
the default and 13245.7 ms with the experiment. These mixed HTTP delivery results
support retaining the experiment off by default. They are not decoder stall
times or a universal ranking. The Windows-generated fixture's SHA-256 is
`9f882c068f1d1affd2128abb9d069a5fa041cbb29347eb0ce47c72ae5fe407bf`;
it differs from the Linux comparison fixture, so settings are compared within
each fixture. Raw final policy report: `.tools/windows-340-rate-policy/report.json`.

## Final source gates

Source `3400e54aa413a40578edb2ea07db815c90b9013f` passed the complete
[native/platform/runtime/service run 37290710254](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37290710254),
[both macOS builds and native ARM tests](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37290710270),
and [portable branch checks](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37290710277).
This includes native unit/vet/race tests (including `./lt/`), actual Go API
contracts checked by frontend validators, four fuzz campaigns, the reachable
Go vulnerability scan, upstream queue/deadline simulations, all eight targets
and Windows service installation/update/failure rollback. The web gate passed
46 tests with one contract-export test skipped, and 34 browser scenarios; the
separate native contract gate supplies the actual backend export.

Final Linux runtime evidence passed 16 sparse cases with the graded default,
three explicitly rate-aware cases, fast/slow/disconnected playback and resource
cycles, cross-file churn, warm expiry, normal/Custom startup, discovery recovery,
preparation, useful-peer restart, five mirror cases and authenticated maintenance.
Artifact `11337372602` passed CRC and GitHub digest verification:
`017bf9be5280048fcee24b0374f63dd91332979f1dc24ae55bbe818c171f7a6f`.
Local copy: `.tools/sparse-340-linux-evidence`.

The final review found a blocked-seek diagnostic still sampling the previously
delivered position. It now uses the actual single-response HTTP start separately
from delivered-byte progress, with sequence protection for old responses and
probes. The response recorder covers full responses, suffix ranges, `If-Range`
fallback, multipart and errors. The old executable failed the new owned regression
while returning exact video bytes: 29 blocked observations sampled piece zero
and reported `READY`. The corrected Linux regression observed the sought missing
piece with `MISSING_CONNECTED` before late HAVE, then returned exact Range bytes.

The first opt-in-policy gate exposed an older test that still assumed automatic
rate-aware deadlines. Its setup now enables that experiment explicitly; the
default-off migration and independent qualification tests remain intact. Both
Linux and macOS suites passed the corrected final-source rerun.

## Earlier Windows executable verification

Source: `8e1478a87cba2fe0c284ed9148ac35393ad852e2`, built by
[run 37271894316](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37271894316).
The PR build uses synthetic merge commit `b73dff9fdf2e955c8938f2f26a9187849323fc68`.
The GitHub Git API confirms that its source tree is identical to `8e1478a8`:
`7f58f4a28d1001ad031b06ba9b7d91b8ac0496d9`. The archive passed CRC verification
and matched GitHub artifact digest
`sha256:51e46575b67be64d576489b5add4509f48e051b2b797bfa795fe9d55fbb9ba8b`.
This establishes artifact/source identity; it is not Authenticode signing.

Standard executable SHA-256:
`0a7637413b46cb38a4fc0910f220e9773ab3c5e9f8c9f3b0b541a1473490671a`.
Reported version: `MatriX.145.Flow-dev-b73dff9fdf2e955c8938f2f26a9187849323fc68`,
libtorrent `2.1.2.0` with the reviewed queue backport.

Each test used a fresh state directory, owned loopback ports and generated bytes.
No existing player session, library or server configuration was used.
The complete preparation and five-case mirror matrices below were rerun with
this `8e1478a8` binary. Peer-hint restart recovery passed with `052858ad`;
later changes concern diagnostics and handle lifecycle rather than hint policy.
Complementary, late-HAVE and choked sole-supplier cases additionally passed at
`4a8e5cb7`, with exact Range bytes and the actual diagnostics API.

| Actual Windows check | Result |
| --- | --- |
| Preparation pause/cancel/resume | Passed |
| Verified 6 MiB + 19 byte episode retained beyond a 1 MiB cache | Passed |
| Abrupt restart, corrupt-piece repair and hash-checked reads | Passed |
| Source-free playback and retained root after disk-path change | Passed |
| Explicit cleanup waits for active readers; reversal rejected | Passed |
| Populated ordinary disk backend migrates to preparation root | Passed |
| Library deletion waits for playback and does not resurrect the torrent | Passed |
| Quota rejection and invalid disk root never claim readiness | Passed |
| Identical-byte HTTP mirror and relative redirect | Passed |
| Corrupt mirror rejection and healthy independent-peer fallback | Passed |
| HTTP 503 fallback and unapproved DNS-to-LAN destination rejection | Passed |
| Imported-source disabling survives abrupt restart | Passed |
| Useful public peer recovered after restart without tracker advertisements, DHT or PEX | Passed |

Raw local reports: `.tools/preparation-8e-check/report.json`,
`.tools/webseed-8e-check/report.json`, `.tools/peerhint-052-check/report.json`.
Sparse diagnostics report: `.tools/sparse-4a8-check/report.json`.
The usual local executable was replaced only after confirming it was stopped;
its previous `4a8e5cb7` binary is retained as
`.tools/artifacts-backups/windows-7a620c27-TorrServer-LT.exe`.

## Verification failure repaired

The first complete build of this revision passed native compilation, unit/vet/
race checks, actual frontend API contracts, fuzz campaigns, the reachable Go
vulnerability scan and upstream queue/deadline simulations. Its runtime stage
failed when the new library-cleanup harness decoded an empty deletion response
as JSON. The endpoint's existing contract is HTTP 200 with an empty body. Commit
`997e18f5` checks that exact contract; the expanded Windows preparation test then
passed. The corrected complete Linux runtime rerun passed at `997e18f5`, as
recorded below.

Local portable Flow tests, 14 fixture/build-tool tests and workflow validation
also passed. Earlier media/sparse/boundary/browser coverage is recorded with its
actual tested revisions in the ledger.

## Complete Linux runtime gate

`997e18f5` passed all 15 sparse cases, the fast/slow/disconnect playback cases,
256 cross-file churn requests, warm expiry, normal/Custom preload checks, four
discovery-state cases, preparation, peer hints, five mirror cases and authenticated
maintenance. All six cross-platform targets, both macOS architectures and
Windows service/update/rollback checks passed. The isolated comparison job passed in [run 37267072705](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37267072705).

The final diagnostics correction prevents unsampled regions or truncated negative
unchoke counts from establishing a wait reason. Positive observed suppliers remain
usable evidence. Its portable regression suite passed; its final native runtime
rerun is tracked separately above. The comparison candidate remains `997e18f5`;
the later change does not modify any scheduling, buffering or native build policy.

## Final lifecycle review

`72c38082` publishes torrent handles atomically and keeps native session/torrent
IDs immutable during removal. Concurrent session-close calls now join the same
teardown; metadata, preload and cache callers capture one handle snapshot.
`8e1478a8` also reads the engine session under its registry lock and rechecks
connection state before registration. These changes address real concurrent
read/remove races; they do not change native scheduling policy.

The final native suite includes `./lt/` in race checks and exercises handle calls
during removal/session destruction plus concurrent torrent-close snapshots.
The `8e1478a8` native unit/vet/contract/race and reachable-vulnerability gates
passed, and final Windows preparation/mirror revalidation passed. Both macOS
architectures and portable checks passed. Its Linux runtime continuation was
cancelled by the final `fb04b171` source update, rather than a test failure.
That update additionally rejects a nil mirror handle when cleanup races
activation, with a native regression for the previously possible panic. The
complete `fb04b171` rerun passed in
[run 37273279837](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37273279837),
including the runtime, six target and Windows service/update/rollback gates.
Both macOS architectures and portable checks also passed at this revision.

## Repeated isolated comparisons

[Run 37267072705](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/37267072705)
compared original `738649bb` / v2.1.0, `d7aafde5` / v2.1.2 without the queue
patch, that same runtime with the exact queue backport, and candidate `997e18f5`.
Each variant used six identical cases, three repetitions in rotated/reversed
order and fresh process/state: **72/72 returned exact Range bytes**. The historical
baseline received only a linker-token parsing repair; its runtime stayed intact.
The original native supplier-filter case requested the scarce piece promptly,
so the conditional native filter rewrite remains unjustified.

Verified artifact: `11329781463`, SHA-256
`6fb048ee69dc5f5680e3cc7822dfed1f4d9b386fc4fdae60e0453994db3787c4`.
The archive passed CRC and GitHub digest verification. A compact record with
fixture identity, executable hashes, all per-case distributions and individual
profile failures is retained in [SPARSE_COMPARISON.json](SPARSE_COMPARISON.json).

Seek response medians, milliseconds, three observations per cell:

| Case | Original | Stable only | Queue fix | Candidate |
| --- | ---: | ---: | ---: | ---: |
| Below consumption | 3588.6 | 3585.3 | 7693.5 | 9086.3 |
| Near consumption | 1966.6 | 1973.5 | 4588.9 | 4593.7 |
| Above consumption | 1692.0 | 866.9 | 871.2 | 1043.9 |
| Complementary partial peers | 1184.2 | 2129.0 | 376.0 | 1901.8 |
| Ten peers, one scarce supplier | 115.6 | 114.8 | 116.4 | 147.7 |
| Late HAVE | 1186.4 | 1677.5 | 1530.3 | 1106.1 |

Buffer-gate medians were essentially unchanged: about 11.0/8.6/6.2 seconds for
below/near/above consumption in all four variants. Paced HTTP read waits were
mixed. Later lifecycle/diagnostic fixes do not change this comparison's policy.
The tested candidate had rate-aware deadlines automatically active. These
results do not establish a general speed improvement, and correctness of the
queue arithmetic does not imply every seek becomes faster.

**Policy decision:** retain the reviewed native arithmetic correction; retain
Legacy as the profile default. The new `Flow.RateAwareDeadlines` control keeps
the proposed byte/rate policy available as an explicit experiment, off by default,
with translated help and compatibility-preserving migration. This follows the
specification's requirement to test before selecting policy parameters. The
ordinary graded ramp remains the default; adaptive buffering remains independent.

Profile characterization: Legacy **9/9 passed**. Conservative **6/9 passed** and
Balanced **6/9 passed**; all three intermittent-supplier repetitions in each hit
the current HTTP reader's waiting boundary. Their longer native retry policy
remains intact. These six cases are failures, not successful playback results,
even though the comparison job itself completed. Preparation waits independently
of the HTTP reader, and another request can retry after native recovery.

Scarce-hint on/off: **12/12 passed** with three repetitions of variation and
intermittent-supplier cases per setting. No duplicate payload was observed in
these cases. Variation seek medians were 5092.8 ms off / 2935.0 ms on, while
intermittent-supplier medians were 1530.0 ms off / 1996.1 ms on. This mixed,
small fixture set does not justify enabling the experiment by default. These
historical scarce tests ran with the automatically active rate-aware policy;
current controls separate the experiments explicitly.

## Interpretation limits

Local controlled sources establish byte integrity and recovery behavior. They
cannot establish physical NAT/IPv6 reachability, multi-hour public-swarm resource
behavior, representative real-media acceptance or the phone's first decoded frame.
The rate-aware and scarce-piece experiments and useful-peer hints remain disabled
by default. These records describe development builds; they are not a stable
release approval or proof of public-swarm speed superiority.
