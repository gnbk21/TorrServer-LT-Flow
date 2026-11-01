# Flow distribution and upgrades

Flow is a fork of TorrServer-LT. Standard binaries keep libtorrent and the
existing HTTP API; `-gst` binaries additionally require GStreamer libraries.
The included license and original Go, native and frontend notices accompany
every platform package. Dependencies are pinned in `build/_common.sh`,
`server/go.mod` and `web/yarn.lock`.
Windows MinGW/GCC and Android NDK static runtime notices are collected from
the toolchains used in CI; their digests are recorded in `BUILDINFO.json` and
their original text is included in `NATIVE_AND_GO_NOTICES.txt`.

`SBOM.cdx.json` records exact Go modules/runtime, pinned native sources, resolved
production web dependencies and the optional Windows helper's runtime archives.
Release and experimental-player workflows validate the generated inventory
against hash-checked, unmodified official CycloneDX 1.6 schemas and reject
duplicate identities, dangling dependencies or missing graph vertices before
publication. Source inventory does not imply every source component is linked.
For a local packaging check, install `web` build dependencies with
`yarn install --frozen-lockfile`, then run
`node build/validate_sbom.cjs <package-directory>/SBOM.cdx.json`.

The optional experimental player also preserves its original application source,
Flow patch license and exact build recipe. `SOURCE_MANIFEST.json` records both
upstream and Flow commits, checks the archived AAR bytes against the built
inventory, and hashes the source bundle. The archive contains prebuilt decoder
binaries; their corresponding native source/configuration/license mapping is
still an open audit and is not established by schema validation. See the
[source-bundle scope](build/just-player/SOURCE_BUNDLE.md). This experimental
artifact is separate from the stable server release.

## Channels

- `MatriX.148.Flow-vX.Y.Z`: stable, after acceptance evidence is recorded in
  `RELEASE_ACCEPTANCE.json` and the tagged commit is present on `master`.
  Flow stable releases follow the owner-approved policy: required automated gates pass;
  unfinished field checks are explicitly disclosed in the release notes.
- `MatriX.148.Flow-vX.Y.Z-preview.N` (or alpha/beta/rc): prerelease. Preview
  builds pass CI but may have remaining real-device or long-session gates.
- `MatriX.145.Flow-preview.N`: the initial numbered preview releases.
- Development branch artifacts are not published releases.

Flow 1.2.0 advances the reviewed upstream maintenance prefix from `145` to
`148`; the Flow semantic version remains independent. Current helpers recognize
both prefixes and compare the Flow version. Historical tags, hashes and download
links remain unchanged. Pre-1.2 installed update helpers recognize only `145`:
install the 1.2.0 Windows package/helper once to cross this boundary, retaining
your existing state/configuration. This naming change grants no new permission
and does not claim that Flow implements every upstream feature.

Docker `latest` follows stable releases; `preview` follows prereleases. A
version-specific image remains available. Choosing a preview is explicit.
Container publication follows native/service/macOS checks, verified packaging
and GitHub release provenance. Every container architecture is checked for
runtime linkage and its exact version before pushing.

GitHub package visibility is separate from repository visibility. If anonymous
GHCR pulls are denied, the owner must enable public visibility in the package's
settings before public container downloads can be verified. See
[GitHub package access guidance](https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility).

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

The optional [Windows interface guard](WINDOWS_INTERFACE_GUARD.md) accompanies
the Windows ZIP. It is never installed automatically. Its explicit preview,
Administrator install/remove and verification limits are documented separately.

Extract the ZIP. Keep all four PowerShell scripts together. Use an explicit
state directory separate from the executable directory. Existing state is
retained; installations are never inferred from whichever process owns a port.
The scripts support Windows PowerShell 5.1 and PowerShell 7. Install GitHub CLI
and ensure `gh` is on `PATH` before using `-RequireAttestation`; this option
requires successful provenance verification in addition to SHA-256 and version
checks. Running the standard executable directly does not require GitHub CLI.

```powershell
.\Install-Flow.ps1 -Channel stable -InstallDirectory C:\Flow\bin -StateDirectory C:\Flow\state -RequireAttestation
& C:\Flow\bin\TorrServer-LT-windows-amd64.exe --path C:\Flow\state --port 8090
```

For an existing `accs.db` account map, add `-HttpAuth`. For an explicit listener,
use `-ListenAddress 127.0.0.1,<PC-LAN-IP>` and repeat those `--ip` values when
starting manually. Managed updates retain both authentication and listener
choices. A restricted installation must include an IPv4 loopback listener for
local maintenance/health checks. With no listener override the upstream default
binds all interfaces; retain private-LAN/VPN deployment restrictions.

Service installation requires an elevated terminal and `-AsService`. The SCM
installation uses an explicit executable and state path. Do not install from a
temporary or user-writable download folder on a shared computer.
The service uses `NT SERVICE\TorrServer-Flow` with a restricted service SID;
its state directory receives Modify access and its executable receives
Read/Execute access for that service. Registration rolls back if these grants
fail. Keep the tray companion under the interactive user account.

## Updates and rollback

`Update-Flow.ps1` supports installations created by `Install-Flow.ps1`, running
on loopback HTTP with their recorded port/state paths. Stop watching first.
It stages and verifies the binary, requests atomic idle maintenance, closes
the owned server cleanly, retains a configuration backup and previous binary,
then checks the exact version and engine readiness. Failure restores the old
binary and rechecks health. Credentials are passed in memory via `-Credential`.
Rollback also restores the pre-update configuration snapshot. Files newly created
by a failed migration are retained privately under the recovery directory's
`failed-state` folder and removed from the restored server's active state path.

```powershell
.\Update-Flow.ps1 -InstallDirectory C:\Flow\bin -Channel stable -RequireAttestation
```

For authentication, add `-Credential (Get-Credential)`. The credential is used
only for the fixed loopback management endpoint and is not written to the install
record. Recovery directories are protected before configuration/account files
are copied, granting access only to the updating user, SYSTEM and Administrators.
Keep backups private. Maintenance uses an exclusive, expiring lease; an updater
cannot release another updater's lease. A crash before shutdown releases the
lease on expiry rather than leaving playback permanently blocked.

Export portable backups from **Settings → Advanced**. Exported settings/library
omit account files, provider keys, private tracker/source URLs and local paths.
Import requires a validated preview and explicit confirmation; it merges portable
preferences and library entries, retains local secrets, and saves a protected
full local recovery snapshot before the engine restarts. The recovery snapshot
contains private state and is not a shareable backup. The support download in the
same tab instead contains bounded, redacted observations.

Use `--doctor --path <state> --port <port> --ip <address>` (and `--httpauth` when
applicable) for a read-only startup check. It checks configured bind addresses,
port availability, account-file validity and optional FFmpeg/FFprobe availability.
An occupied port while your server is running is expected; this is not a server
health failure. See [measurement and acceptance guidance](docs/history/MEASUREMENTS.md).

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
# Source and container build consistency

The obsolete standalone `Dockerfile.builder` has been retired. It used separate
dependency defaults and omitted Flow's reviewed native patches. Source builds
use `build/linux-amd64.sh` (or the corresponding target script), which shares
the pins and patch recipe in `build/_common.sh` and `build/_deps.sh`. Build the
web assets with the existing web build step before the native executable.

Containers use `Dockerfile.runtime` with those verified native outputs under
`dist/amd64/torrserver`, `dist/arm64/torrserver` or `dist/armv7/torrserver`.
The release workflow verifies the actual version/linkage for all three
architectures before publishing. Do not build an unpatched second engine for
containers. See `.github/workflows/build-docker.yml` for the complete procedure.

## Staging a release before publication

When `publication_policy.stage_before_publish` in `RELEASE_ACCEPTANCE.json`
matches the exact release tag, the usual tagged gates and attestations run and
the resulting GitHub release is saved as a draft. Container publication waits.
This permits private installation checks before public download availability;
it does not bypass any release gate. Personal settings never enter CI assets.

After verifying the private installation, publish that existing draft and mark
it Latest if it is stable. Resume `build-docker.yml` manually with the published
tag, its exact source commit, channel and successful tagged release run ID.
The workflow verifies the published release and original successful source run
before using its binaries. Older drafts and development artifacts cannot
satisfy these checks.

A retry of an already published staged tag preserves its published visibility.
The classifier checks the existing release before deciding whether to stage;
a missing release or existing draft remains staged, while unexpected API errors
stop the workflow. This prevents a successful retry from withdrawing downloads.
