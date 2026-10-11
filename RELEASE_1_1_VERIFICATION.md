# Flow 1.1 verification

Release target: `MatriX.145.Flow-v1.1.0`. Publication is pending the final-source
gates; this document does not yet record a published or fully verified package.

## Completed investigation and local checks

- The new cold A/B/A cancellation regression passes the normal case and fails
  the deliberately late-completion case on the old runtime `78c437bd`. Failure
  includes positive race exercise and a real strict peer-cap violation.
- All 11 generated-peer socket regressions and both native patch application,
  hardlink-isolation/repeatability/drift tests passed before final security edits.
- Four inventory-schema/graph/bounded-file regressions passed. Local log, Flow
  and console package tests passed after sink-level log hardening.
- Earlier immutable-source controlled comparisons, player bundle verification,
  native profiling, WFP/service/update receipts and browser lab measurements are
  recorded in `UPGRADE_VERIFICATION.md` and `UPGRADE_MEASUREMENTS.md`. They do not
  replace final-source build or tagged package verification.

## Remaining publication checks

- Final native compilation and cancellation regression on Linux/Windows.
- Final CodeQL execution and recorded security-alert review.
- Native/race/fuzz/browser/platform/service/update/inventory/comparison gates.
- Merge of PR #5, exact tagged rebuild, reproducible packaging and provenance.
- Downloaded published asset hashes/source/version and public container identity.
- Eight-hour owned Adaptive Streaming convergence run, capacity experiment
  disabled: in progress, not passed. This uses the earlier `78c437bd` runtime;
  it is not exact-tag or new capacity-patch acceptance.

Physical Android/TV, public swarms and Windows boot/sleep/network/VPN remain
unfinished field checks under the owner's disclosed-publication policy.
Optional player independent decoder rebuild/redistribution compatibility and
complete native profiling attribution remain unverified; the APK is excluded
from this stable server release. Security scope is recorded separately in
`RELEASE_1_1_SECURITY_REVIEW.md`.

## Follow-up after the c02e7933 gates

Run `38133894530` failed the cold A/B/A cancellation check: late-completion
trial 1 never requested the cold target within the unchanged 10-second limit.
Owned Windows reproduction showed the target available and urgent but normal
ahead requests refilling the strict 64-block queue first. The new native patch
gives deferred deadline scheduling newly available credit before normal refill;
it changes only the opt-in capacity experiment. The corrected Windows executable
passed all six cold-seek trials; final-source platform gates remain separate.

The full CodeQL run `38133894549` passed scan execution, closed the corrected
log alerts, and produced additional findings reviewed in the security document.
The seven explicitly approved dispositions were recorded on GitHub. Additional
findings were not dismissed under that authorization.

Follow-up local checks: media MIME/byte-range, Flow, logging and console tests;
44 Python tests; actionlint; both real native patch application, repeatability,
hardlink isolation and drift tests passed. Packages importing libtorrent cannot
run on this host without a native compiler and remain assigned to CI.

The final host-tool follow-up passes all 45 Python tests, including an actual
Bash check that valid B2 is reused despite its nonzero version exit, and that
missing/wrong-host tools are bootstrapped. Native/race gates now explicitly
include root cache cleanup, MIME, Torznab and JacRed tests. New endpoint tests
exercise unavailable-interface routing, TMDB response limits, oversized
configured certificates and preservation of existing MSX sharing on bad input.

Cleanup review additionally found that the HTTP-side close signal preceded
native disk release. Removal now waits for actual storage ownership, preserves
bytes/records on timeout or failure, checks replacement ownership under the
engine lock and uses the hybrid's native storage digest. A held real file
descriptor regression checks that close cannot expose absent ownership early
and cannot remove a newly registered replacement. These native-linked checks
are pending the final-source CI run.

## Verified checkpoints before publication

- Runtime fixes are frozen at feature source
  `0b6da77a0c82d393ba12af2496aecdac4fb3347c`. Documentation updates do not
  change runtime code; the final tag still rebuilds the exact published tree.
- At checkpoint `8f7b5f4e`, Linux amd64 native units, race/fuzz/security and the
  complete controlled playback/resource-cycle step passed in
  [run 38136308932](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/38136308932).
  All six cross targets built. The obsolete overall workflow was cancelled
  after that job finished to release the final-source build queue; it is not
  recorded as a successful full workflow.
- The downloaded Windows executable from that checkpoint has SHA-256
  `b00e7934a02d835b8e75f5173deb6a6cb668114ba2f5e72721f819b1c37a0e6b`.
  Six fresh-state cold A/B/A trials passed: three ordinary and three with
  positively exercised late replies, zero peer queue overflows, and verified
  original Range bytes within the unchanged 10-second target limit.
- An additional owned Windows HTTP check against the same executable passed
  malformed/traversal removal rejection with an unrelated file preserved,
  plus HTML/SVG/unknown-format binary download headers, `nosniff`, and exact
  206 Range bytes. This predates the final disk-ownership/WebVTT edits and
  does not substitute for their final-source tests.
- Both macOS architectures and arm64 native unit tests passed at final runtime
  source in [run 38137745936](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/38137745936).
  Portable branch checks passed in
  [run 38137745870](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/38137745870).
  Optional player packaging passed in
  [run 38137745782](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/38137745782);
  independent decoder rebuilding remains unverified and its APK remains
  excluded from the stable server assets.
- Final runtime source CodeQL passed every category and the web gate in
  [run 38137745884](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/38137745884).
  The PR-ref query retains 30 reviewed open findings. Scan execution is not a
  clean-alert claim; their boundaries and the seven separately approved
  dispositions are recorded in `RELEASE_1_1_SECURITY_REVIEW.md`.
- Final-source Windows executable SHA-256:
  `ebebb25826b83055e90e38a94b1d383fda9b7cee2f10269c8676b30eb5811bf7`.
  An owned real-engine test passed invalid removal rejection, original-byte
  HTML/SVG/unknown-format Range downloads, native WebVTT Range MIME, `nosniff`,
  and successful disk-cache removal while preserving an unrelated file.
  Its receipt is `.tools/upgrade-0b6da-stream-boundaries/report.json`; this is
  controlled HTTP/storage evidence, not phone decoding or tagged identity.
