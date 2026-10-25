# Withdrawal of unsupported distributions — 25 October 2026

The owner approved withdrawing every Flow release before 1.1.7 and its older
container versions. Earlier builds lack the security boundaries shipped in
[1.1.7](../../SECURITY_HARDENING.md). This action does not mean every historical
CodeQL finding was a demonstrated exploit; the [full review](SECURITY_RELEASE_1_1_7_REVIEW.md)
distinguishes fixes, retained administrator privileges and conservative findings.

## Release downloads

Ten historical releases were converted to **private drafts**, retaining their
original assets, source tags and original notes below an upgrade notice:

- 1.1.6, 1.1.5, 1.1.4, 1.1.3 and 1.0.1.
- 0.2.0-preview.7, preview.5, preview.4 and preview.2.
- Flow-preview.1.

For each release, unauthenticated release-by-tag API access and one representative
asset download returned HTTP 404 after withdrawal. The public release list then
contained only 1.1.7. Authenticated checks verified every historical asset's ID,
name, size and digest remained unchanged in its draft. All remote tag references
were compared before and after and were identical. Drafts remain accessible to
repository writers; this is withdrawal, not permanent deletion of audit records.

**Flow 1.1.7 remains public, stable and Latest, with all 35 assets unchanged.**
The release-source tag remains `87d1632e47d64adcfd98d7d916d80aebedbc4e8c`.

## Containers

[Retirement run 40208494826](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/40208494826)
completed successfully and retained its JSON receipt as an artifact. The
one-time workflow and script are recorded in commit
`f621c662f84cc99354b803ec3072e0c7ce357476` and were removed after completion.

The job resolved all manifest graphs before deleting anything. It removed
**70 obsolete versions** reachable from ten old release indexes, including old
preview aliases. It protected the current index, all three architecture
manifests and their three attestation manifests. A final package-catalog read
contained exactly those **seven current versions**.

Both `MatriX.145.Flow-v1.1.7` and `latest` still resolve to:

`sha256:48f0a0e9a1f14b7200f910fc77025ce66691bb87aa014e28057cb42ceaf3ec0f`

Independent anonymous registry checks verified Linux AMD64, ARM64 and ARMv7
manifests and their exact 1.1.7 version/source labels after retirement. This was
registry verification, not a new runtime test on three physical machines.

## CI and limits

Future benchmarks now download the supported 1.1.7 baseline using a fixed
SHA-256 plus exact-tag/source/workflow provenance verification. A fresh download
passed both checks before withdrawal. Historical performance results retain
their original comparison versions; none were relabelled as 1.1.7 measurements.

Previously downloaded copies, local installations, third-party mirrors and
historical source tags cannot be recalled by withdrawing release downloads.
The owner's installations and settings were preserved. Update installed binaries
separately, following the [compatibility notes](../../SECURITY_HARDENING.md).
