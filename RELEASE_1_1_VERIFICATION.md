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
- Eight-hour owned default-runtime convergence run: in progress, not passed.

Physical Android/TV, public swarms and Windows boot/sleep/network/VPN remain
unfinished field checks under the owner's disclosed-publication policy.
Optional player independent decoder rebuild/redistribution compatibility and
complete native profiling attribution remain unverified; the APK is excluded
from this stable server release. Security scope is recorded separately in
`RELEASE_1_1_SECURITY_REVIEW.md`.
