# TorrServer-Flow 1.2.0

Version: `MatriX.148.Flow-v1.2.0`.

The upstream maintenance prefix advances to `148`; Flow's own version is 1.2.0.
Older `145` release identities remain supported by the new helpers. Existing
pre-1.2 Windows update helpers require installing the new package/helper once;
keep the existing state directory to preserve settings. Historical tags are not
renamed, and the prefix does not imply complete upstream feature parity.

## Playback diagnosis and setup

- Follow one playback journey from metadata and peer connection through required
  head/index data, optional reserve and HTTP delivery. See the blocker, evidence
  age, confidence and one read-only tuning suggestion together. Server delivery
  observations remain separate from the player's decoded buffer and frame stalls.
- Use guided phone setup with a five-minute, read-only connection proof and
  relevant authentication/origin evidence. The proof grants no management or
  playback authority. Flow does not grant Android permissions or trust origins
  automatically.
- See cached codec, audio/subtitle and bitrate information for the selected file,
  parser availability, verified exact-byte reuse eligibility/results and clearer
  metadata-recovery and next-episode warmup states. Search results are not all
  probed, and Flow does not silently select another release or reduce quality.

## Correctness, resources and the web interface

- Distinguish engine startup from generic unavailable-service errors. Manual
  pause/cancel can reach the server during polling cooldowns; mutations are never
  automatically replayed and server authorization/rate limits remain in force.
- Explain settings using versioned backend metadata, architecture-correct bounds,
  cache-allocation/application previews and older-server fallback. Preserve the
  Common/Advanced views and draft/saved/effective states. Remove the ineffective
  browser-profile selector while retaining its deprecated API/configuration field.
- Require three fresh five-second memory samples above 1.5 times the existing
  entry threshold before leaving pressure mode. Release warm/background work
  first and retain reader, preload and required-index protection, including
  clearly reported protected overcommit.
- Offer optional private Go flight recording: a 30-second target window, 32 MiB
  recorder hint, separate 64 MiB output limit, five-minute capture cooldown and
  at most two retained raw traces. It pauses under pressure and stays off by
  default. Raw traces require explicit selection and are excluded from ordinary
  support exports.
- Add bounded, read-only runtime-storage categories and preview-first developer
  cleanup tools with path, activity, lock and reviewed-manifest checks. Improve
  mobile layouts, long-value wrapping, keyboard behavior and all seven locales.
- Correct metadata cancellation being misreported as invalid data, Windows
  TCP/UDP fixture admission across excluded port ranges and large-piece VBR
  fixture sizing. Keep all playback/integrity assertions and failed evidence.

## Toolchain, dependencies and development

- Build with **Go 1.27.3** and **OpenSSL 3.5.9 LTS**. Verify the OpenSSL archive
  against its reviewed SHA-256 before extraction; include it in native receipts.
- Update Go networking/cryptography, HTTP/CORS, DNS, JSON/MIME, validation and
  protobuf dependency chains. Update purego to 0.11.2, including its macOS native
  profiling crash fix. Retain the reviewed Boost 1.85.0 and current libtorrent
  2.1.2 pins; no torrent-engine replacement is introduced.
- Patch Vite, Vitest, Zod, source-map-js and Undici. Scope the minimatch 11.0.1
  security resolution to typescript-estree and verify its consumed glob API.
  The two remaining Yarn audit entries concern ESLint's separate minimatch 3.1.5:
  the maintainer's advisory identifies the affected 10.2.6 preprocessing algorithm,
  which v3 does not contain. This development-only audit discrepancy is recorded,
  rather than described as a zero-finding audit.
- Add one local verification entry point, complete test-package inventory and
  compiler/dependency/patch receipts. Share a source-bound web build across PR
  consumers and let Linux validation proceed independently of other platforms.
- Include the reviewed MatriX.147.2 playlist, sidecar, filename, zoned-IPv6 and
  internationalized-referer compatibility fixes.
- Include both MatriX.148 fixes: canonical IP spelling for referer deny rules
  and correctly labelled binary units in Torznab results. Flow retains its
  stricter shared IDNA/hostname validation and separate management-origin policy.

## Compatibility and verification limits

Lampa → Flow → Just Player and original media bytes remain supported. Legacy and
existing experimental-control defaults are unchanged. Public packages contain
no personal configuration, and the optional recorder is not enabled by update.

Publication requires the exact tagged native/security/platform, controlled
streaming, Windows lifecycle, packaging, inventory and provenance gates. Compiler
comparison alone does not replace the eight-platform qualification. Measurements
remain variable and do not establish a general public-swarm streaming speed gain.

Representative final-version phone/public-swarm playback, physical Windows
boot/sleep/network recovery and multi-hour resource acceptance remain unfinished
and are disclosed under the owner's publication policy. No additional two-hour
test is added. Windows binaries are not Authenticode signed; passing scans and
retained security dispositions are not proof of zero vulnerabilities. Available
peer supply still limits sustainable playback bitrate.

Sources: [Go release history](https://go.dev/doc/devel/release),
[OpenSSL advisory](https://openssl-library.org/news/secadv/20260929.txt),
[purego 0.11.2](https://github.com/ebitengine/purego/releases/tag/v0.11.2),
[minimatch advisory](https://github.com/isaacs/minimatch/security/advisories/GHSA-g3hv-75wg-m249).
