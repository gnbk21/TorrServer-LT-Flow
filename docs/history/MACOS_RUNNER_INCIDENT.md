# macOS release runner incident â€” 23 October 2026

## Confirmed cause

Tagged 1.1.5 run [39982129539](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39982129539)
could not acquire GitHub-hosted macOS ARM machines on two attempts. Both macOS
jobs had `runner_id: 0` and no executed steps. GitHub's check annotations say
"The job was not acquired by Runner of type hosted even after multiple attempts"
and explicitly identify macOS ARM capacity constraints. They were cancelled by
the scheduler, not a failing compiler or Flow test.

Both Intel and ARM artifacts use `macos-latest` ARM machines. Intel code is
cross-built and version-probed through Rosetta, so both jobs share the capacity
bottleneck. [GitHub's runner reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)
lists that label as ARM and lists separate Intel runner labels. Changing the
label is not a proven cure for an ARM capacity shortage; an Intel machine also
cannot execute the native ARM test suite without changing the verification.

The same workflow successfully built both architectures earlier that day in
[39977825892](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/39977825892).
Repository Actions permissions are enabled. The evidence supports a hosted
runner allocation failure; GitHub's internal reason for the capacity shortage
is unavailable. The general status page showing Actions operational does not
contradict the job-specific capacity annotations.

## Release handling

The owner approved publishing available platforms after the wait. The explicit
`recover-release.yml` workflow preserves the original tag/source and consumes
only artifacts from its exact tagged run. It requires 23 successful non-macOS
roles, the exact-source security run, and scheduler evidence that both omitted
jobs executed no steps. It does not accept macOS build/test failures as capacity
failures. Default releases still require all 25 roles.

The distribution contains Windows, Linux and Android only. Manifest, checksums,
ZIPs, SBOM and BUILDINFO inventory only those assets and explicitly list both
macOS targets as pending. Provenance identifies the recovery packaging workflow
commit separately from the unchanged binary source commit. Neither the tag nor
the owner's installation/settings are replaced. Mac users retain 1.1.4 until
fresh 1.1.5 Mac builds and validation can be completed. No older development
Mac binary is relabelled as a release.

The frozen original tag workflow cannot be edited by a later master commit.
Any future macOS recovery must separately validate fresh exact-tag artifacts
and regenerate consistent manifest/checksum/inventory records; rerunning the
old publication job after publishing a partial distribution is not appropriate.


## Historical status after 1.1.7

This record describes the completed 1.1.5 recovery. The one-off workflow has
since been removed; current recovery uses the guarded
[generic platform workflow](../../.github/workflows/release-recovery.yml). Use
[the current stable release](../../RELEASE_1_1_7_NOTES.md), which includes both Mac
architectures. The old recommendation above was specific to that incident.
