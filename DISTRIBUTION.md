# Flow distribution and upgrades

Flow is a fork of TorrServer-LT. Standard binaries keep libtorrent and the
existing HTTP API; `-gst` binaries additionally require GStreamer libraries.
The included license and original Go, native and frontend notices accompany
every platform package. Dependencies are pinned in `build/_common.sh`,
`server/go.mod` and `web/yarn.lock`.

## Channels

- `MatriX.145.Flow-vX.Y.Z`: stable, after acceptance evidence is recorded in
  `RELEASE_ACCEPTANCE.json` and the tagged commit is present on `master`.
- `MatriX.145.Flow-vX.Y.Z-preview.N` (or alpha/beta/rc): prerelease. Preview
  builds pass CI but may have remaining real-device or long-session gates.
- Development branch artifacts are not published releases.

Docker `latest` follows stable releases; `preview` follows prereleases. A
version-specific image remains available. Choosing a preview is explicit.

## Integrity and provenance

Download a platform ZIP from this fork's GitHub Releases. Verify its SHA-256
against `SHA256SUMS`; `BUILDINFO.json` records the commit and exact binary
digests. The ZIPs use sorted inputs and fixed archive timestamps. This makes
packaging reproducible from identical CI inputs; native compilation is not
claimed to be bit-for-bit reproducible across different toolchains.

GitHub build attestations bind artifact digests to the build workflow. Verify
with `gh attestation verify <downloaded-file> --repo gnbk21/TorrServer-LT-Flow`.
These attestations are separate from Windows Authenticode publisher signing.
No publisher signing certificate is bundled or implied.

## Windows installation

Extract the ZIP. Keep all four PowerShell scripts together. Use an explicit
state directory separate from the executable directory. Existing state is
retained; installations are never inferred from whichever process owns a port.

```powershell
.\Install-Flow.ps1 -Channel preview -InstallDirectory C:\Flow\bin -StateDirectory C:\Flow\state -RequireAttestation
& C:\Flow\bin\TorrServer-LT-windows-amd64.exe --path C:\Flow\state --port 8090
```

Service installation requires an elevated terminal and `-AsService`. The SCM
installation uses an explicit executable and state path. Do not install from a
temporary or user-writable download folder on a shared computer.

## Updates and rollback

`Update-Flow.ps1` supports installations created by `Install-Flow.ps1`, running
on loopback HTTP with their recorded port/state paths. Stop watching first.
It stages and verifies the binary, requests atomic idle maintenance, closes
the owned server cleanly, retains a configuration backup and previous binary,
then checks the exact version and engine readiness. Failure restores the old
binary and rechecks health. Credentials are passed in memory via `-Credential`.

```powershell
.\Update-Flow.ps1 -InstallDirectory C:\Flow\bin -Channel preview -RequireAttestation
```

Installations using HTTPS redirects, additional CLI overrides, custom wrappers,
or Preview 1 (which lacks maintenance and `release.json`) use manual upgrades:
stop the server, back up its state, replace the binary, restart with the same
arguments, check playback, and retain the previous executable for rollback.
No update is performed while playback or detached preload is active.

## Other platforms

Extract the ZIP, make the standard binary executable if necessary, and start
it with `--path <state-directory>`. Linux/macOS/Android automation remains under
the host's service/package manager; the Windows scripts do not manage these
hosts. Phone/Just Player and multi-hour acceptance evidence is recorded
separately from compilation and controlled local tests.
