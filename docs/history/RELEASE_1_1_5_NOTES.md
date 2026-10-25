# TorrServer-Flow 1.1.5

Version: `MatriX.145.Flow-v1.1.5`.

**Platform availability:** Windows, Linux and Android are included. macOS Intel
and Apple Silicon assets are pending: GitHub could not assign macOS ARM runners
on two attempts, and neither Mac build executed. Mac users should retain 1.1.4.
The owner approved this partial platform release; this does not claim Mac checks
passed. [Runner diagnosis](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/docs/history/MACOS_RUNNER_INCIDENT.md).

A compatibility update adapting the applicable MatriX.147.1 query fix.

- Preserve literal `%` and `+` in playback metadata and searches, including
  JacRed, Rutor and Torznab. Retain one extra percent-decoding layer for legacy
  clients; a literal percent sequence that is valid encoding remains ambiguous
  in those legacy text fields.
- Keep already formed torrent/poster URLs and magnet components intact, so
  encoded path separators and signed query values are not decoded again.
  Validate the same canonical source used by playback parsing, including the
  credential-free compatibility routes.
- Add native metadata-wait/removal and delayed-bootstrap DHT regressions.
  These do not introduce a new DHT stack or change swarm/profile defaults.
- Add a [free RuTracker/Jackett setup guide for Lampa](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/LAMPA_RUTRACKER.md).
  Indexing is configured in Lampa's parser; Flow remains its streaming server.
- Make focused compatibility-patch validation reusable through an explicit
  acceptance record. It requires exact successful native/security evidence,
  unchanged engine/cache/scheduler/UI/dependency/build inputs and a pinned
  tested candidate. Other changes retain full validation.

## Verification and updating

The tagged release passed native unit/race/vulnerability/fuzz, source identity,
all eight functional groups, Windows lifecycle, platform builds, checksums,
reproducible packaging and provenance gates for the included platforms. The
recovery distribution uses the original tagged binary source and a separately
identified packaging workflow commit; BUILDINFO and GitHub attestations record
those distinct identities. Earlier full performance matrices
and generated UI are reused only after source/receipt checks; this is not a
claim that those performance comparisons were repeated for 1.1.5.
See [the verification record](https://github.com/gnbk21/TorrServer-LT-Flow/blob/master/docs/history/RELEASE_1_1_5_VERIFICATION.md).

For Windows, extract the new ZIP, stop the previous server, then launch the new
executable with the same state directory, port and authentication/HTTPS options.
Keep the previous executable and state backup for rollback. Do not run both
against the same state directory. Existing preferences are not reset.

Original media quality, libtorrent 2.1.2, Go 1.26.10 and the existing cache are
retained. This update does not establish greater public-swarm throughput.
Legacy stays the default; experiments and PGO remain off unless selected.
Automatic torrent-sidecar subtitles and the unrelated legacy web dependency
update are not included. Broader real-device/public-swarm, physical Windows
recovery and exact-final endurance acceptance remain disclosed field work.
The existing reviewed security findings remain open; passing scans are not a
zero-alert claim. Windows executables are not Authenticode signed.
