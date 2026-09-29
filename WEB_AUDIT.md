# Modern interface audit and acceptance record

Specification: supplied Modern Web Interface plan, revision 2.0, sections 1–100.
Integration branch: develop. Implementation began on feature/modern-web. This
is an implementation record, not a Modern Web 1.0 release approval.

## First public preview

On 30 September 2026, [Flow Preview 1](https://github.com/gnbk21/TorrServer-LT-Flow/releases/tag/MatriX.145.Flow-preview.1)
was published as a prerelease from the verified build commit 847f6b4f, reusing
unchanged CI binaries. The tag targets that exact commit; binaries retain their
development version identifier. Eight platform ZIPs, a standalone Windows
executable, tray script, setup instructions, build provenance, original dependency
notices and checksums are provided. All 17 uploaded asset digests matched the
local packages; the ZIP contents and every packaged binary were verified.
The inherited stable tag workflow was temporarily suspended during manual
publication and restored to active afterward; no preview Docker image was built.
This distribution does not satisfy or waive the device, playback, performance
and stable-release gates below. See [release notes](RELEASE_NOTES.md).

## Baseline and preservation

The audited legacy baseline is develop commit 757577be. Its native CI and
isolated Windows smoke results are recorded in AUDIT.md. The earlier draft's
API table contained incorrect methods and fields; the contract below replaces it.

Legacy sources are retained in web-legacy/ pending parity and device acceptance.
The local legacy archive and incoming draft backup are under .tools/audit/.
Existing user playback and state directories have not been used for these tests.

## Actual API contract

All server requests use same-origin credentials and bounded, abortable requests.
A 503 response means engine starting/unavailable; 401/403 means authentication or
access failure. An echo response alone never establishes Internet readiness.
Settings and torrent response validators retain unknown fields.

| Operation | Contract |
| --- | --- |
| Version | GET /echo, plain text |
| Library | POST /torrents with action list/get/add/set/rem/drop/wipe |
| Add | link, title, category, poster, data, save_to_db |
| Remove | action rem and hash; no invented with_files parameter |
| Upload | POST /torrent/upload, multipart file and optional save/title/poster/category |
| Cache map | POST /cache with action get and hash; Go DTO uses capitalized field names |
| Viewed | POST /viewed with action list/set/rem, hash and one-based file_index |
| Stream | GET /stream with link, one-based index, play and ss session group |
| Prepare | GET /stream with preload and stat; wait for successful prebuffer before offering playback |
| Playlist | stream m3u endpoint or /playlistall/all.m3u |
| Flow | GET /flow/status/:hash, /flow/network and /flow/tray |
| Flow control | POST /flow/control with pause/resume (existing server capability) |
| BT settings | POST /settings with get/set/def; def is a destructive reset, never a defaults getter |
| Shutdown | GET /shutdown, explicit confirmation |
| WAF | GET/POST /waf, independent saved draft |
| Storage | GET/POST /storage/settings, application restart notice |
| Search | GET /search/, /torznab/search/, /jacred/search/ with query parameter |
| Provider test | POST /torznab/test or /jacred/test with host and key |
| TMDB configuration | GET /tmdb/settings; save through nested BT settings |
| TMDB posters | Explicit external provider request using configured URL/key; optional and independent of library |
| GStreamer | GET/POST /gst/settings; built_in guards runtime-only routes |
| GST runtime | GET /gst/echo; reports found/available/works separately |
| GST playback | /gst/:hash/master.m3u8 with index and audio; probe and heartbeat routes |

The frontend does not change authentication policy. Known-torrent playback
continues to follow upstream routing rules. QR/intent URLs reject embedded
credentials; diagnostic URL rendering removes credentials, paths and query tokens.

## Implementation architecture

React 19, strict TypeScript, Vite 8, Tailwind 4, TanStack Query 5, Radix dialogs,
React Hook Form and i18next. Hash routing keeps refreshes compatible with Gin.
Four lazy pages: dashboard, library, search/add and settings.

Library polling uses 1 second for metadata/preload/active readers and 5 seconds
when idle. Network/runtime use 5 seconds. Flow uses 1 second only on relevant
dashboard/diagnostic/preparation views. Hidden-tab interval polling stops.
A bounded two-minute telemetry history belongs to each visible session.

Small additive backend projections support truthful observation:
active_readers, warm_idle and a compact flow_playback summary are emitted in
torrent status. Detailed traces stay in the existing Flow API. Flow observation
uses GetTorrentInfo instead of promoting a dormant torrent through GetTorrent.
Active playback stays visible with unknown health when telemetry is unavailable.

Settings retain unknown values, expose all current BT/Flow fields, validate
custom bounds and min/max relations, and require explicit Apply. WAF, GST and
storage save separately. Background focus/reconnect does not replace their
unsaved drafts. BT Apply/reset warns about engine restart and active playback.

Playback links always retain HTTP open/copy fallbacks. Android intents preserve
HTTPS and IPv6. The internal player loads HLS only when needed, uses GStreamer
for supported containers, preserves server audio preference, exposes audio and
subtitle selection, bounds error recovery, and tears down requests/timers/media.
Direct browser playback remains available when GST is absent.

Multi-add supports imported JSON/text and torrent files. Partial failures retain
only failed entries for retry, including their metadata.

## Verification recorded on 2026-09-30

- TypeScript type checking and ESLint pass.
- 40 unit/component tests passed in nine files: API behavior, settings bounds and
  merging, health, formats, episode parsing, intents, LAN URLs, GST validation,
  redaction, seven-language key parity, player teardown and batch retry.
- 27 Playwright checks passed against controlled API fixtures: five specified
  viewport sizes plus landscape, preparation gating, settings confirmation,
  dirty navigation, file indices, viewed progress, pairing, authentication,
  503, offline recovery, retained library state and keyboard focus. Additional
  checks cover pause/resume, fresh warm sessions, playback without telemetry,
  all seven languages and deduplicated visibility-aware polling.
- Screenshots inspected: active desktop, mobile settings and mobile pairing.
  The pairing select width was corrected after this inspection.
- A player component test exposed a real Radix portal timing defect; initialization
  now waits for the mounted video element. Direct and HLS lifecycle tests pass.
- Production Vite build passed. Measured entry chunk: 167.14 KB gzip; HLS:
  178.98 KB gzip, lazy. Dashboard and shared chunks add to first view; the entry
  chunk alone is not a total transfer figure. No production source maps emitted.
- Asset cache policy unit checks passed: HTML/manifests revalidate,
  hashed assets are immutable for one year, other assets use short caching.
- CI workflow migrated from CRA flags/tests to typecheck, lint, Vitest, Vite,
  Playwright and asset-generation tests. Final native CI passed (links below).
- Legacy executable 757577be was captured on isolated loopback port 18091 with
  fresh state. Desktop 1440x900 and mobile 375x812 screenshots are saved under
  .tools/audit/legacy-browser-baseline. No browser runtime exceptions occurred.
  The legacy mobile page measured 769 px wide at a 375 px viewport. Four legacy
  JavaScript assets total 635,052 bytes gzip; this is the asset sum, not a measured
  phone transfer. The baseline was reconstructed from the preserved audit build
  because the incoming modernization draft had already removed legacy sources.
- Registry audit of the modern frontend dependency tree reports zero advisories
  across 396 dependencies. This does not cover the retained legacy toolchain or
  the native C++/OpenSSL dependencies.
- Manifest icon dimensions were checked against the PNG headers: 180x180 and
  512x512. Build packaging retains original license notices for 78 production
  dependency packages, including lazy HLS/QR dependencies. One notice omitted by
  its npm package is preserved from upstream with provenance in web/licenses.
- Production Chromium sample: 375x812, 4x CPU throttling, controlled API fixtures,
  180 seconds. First-dashboard JavaScript asset sum was 186,769 bytes gzip (Node
  zlib measurement), including lazy dashboard/shared chunks but excluding CSS,
  images and media. Dashboard usable in 827 ms; library navigation took 156 ms.
  DOM count remained 424; post-GC heap grew from 4,971,928 to 5,123,576 bytes
  between 30 and 180 seconds. Script duration rose from 0.629 to 1.158 seconds;
  total task duration from 1.385 to 2.390 seconds. There were no browser errors.
  Counts: 75 library and 75 Flow requests, 28 each network/runtime/tray requests,
  one settings read. Browser throttling affects timer cadence; interval behavior
  is separately asserted in the bounded polling E2E check. This short sample
  does not establish mobile CPU use or multi-hour memory stability. Local results,
  screenshot and Chromium CPU profile are in .tools/audit/modern-performance.

Browser fixtures do not prove torrent throughput, media decoding or phone
intent handling. No real playback claims are derived from mocked tests.

## Remaining gates

The generated assets contain the modern interface. Native CI and isolated
executable checks passed; repeat checks affected by any later fixes.

## Feature parity reconciliation

| Legacy capability / plan requirement | Modern execution path |
| --- | --- |
| Magnet/hash/URL, uploads, multi-add, import | Library Add and Add/search page; partial failures remain retryable |
| Library/categories/filter/sort/poster | Torrents page, including uncategorized; lazy poster fallback |
| Edit metadata/category | Card Edit; preserves data; optional TMDB poster lookup |
| Export JSON/magnets/torrs, M3U | Library Export; card playlist; phone pairing playlist |
| Remove/remove-all/drop cache/cache map | Card secondary actions; Advanced confirmations; Files cache details |
| Details/files/episodes/viewed | Files modal; folder grouping, four naming patterns, raw paths, timecodes |
| Internal player/subtitles/GST | Lazy native video/HLS; embedded text tracks and external WebVTT; audio selection |
| External players | Files/dashboard links; Android Just Player/VLC and desktop platform links; HTTP fallback |
| Rutor/Torznab/JacRed | Configured providers on full Add/search page; add/metadata/prebuffer/playable stages |
| All BT/Flow settings | Six settings tabs; custom swarm bounds in Advanced; Apply/discard/restart confirmation |
| WAF/security/SSL | Security tab; independent WAF save and read-only/warning states; auth-file guidance |
| TMDB/provider keys | Integrations; masked secrets, saved-provider tests, optional external lookup |
| GStreamer/DLNA/Bonjour/storage/WebDAV/FUSE | Integrations/Advanced settings; runtime flags and optional GST health |
| Shutdown | Advanced explicit confirmation; existing GET contract |
| Flow dashboard/status/diagnostics/network | Dashboard and per-torrent diagnostics; backend-derived health and unknown values |
| Flow pause/resume | Header control with backend result, pending/error handling |
| Pairing/PWA/i18n/accessibility | Safe LAN/manual selection and SVG QR; manifest; seven languages; Radix focus/Escape |

The old About/provider shortcuts are represented by fork documentation and
provider settings rather than an additional route. The independent legacy speed
test is intentionally excluded: torrent/server rates remain available, and a
synthetic file download is not a measurement of playback sustainability. The
legacy interface and its dependencies remain preserved in web-legacy; they are
not bundled into the modern executable. Optional SSE and service-worker work
were not introduced, as the plan explicitly permits polling and ordinary HTTP.

Real Android/Just Player launch, seek, background/resume, network changes and
multi-hour memory acceptance require the user's device and representative media.
The user elected to perform phone testing. Retain legacy sources and avoid a
Modern Web 1.0 release claim until those gates are satisfied.
Use [WEB_DEVICE_CHECKLIST.md](WEB_DEVICE_CHECKLIST.md) to record those results.

At the user's request, [PR #2](https://github.com/gnbk21/TorrServer-LT-Flow/pull/2)
was merged from feature/modern-web into develop on 30 September 2026. Merge
commit a82087bb342467987f01228ec586f805a692bccf has the exact tree of the
verified feature head 59c17b48; changes since the tested build 847f6b4f are
documentation only. Develop is the fork's default branch. Stable master
promotion and legacy removal remain pending; integration does not waive the
acceptance checks above. Initial implementation: cc9d0ff3cb732c95b7df16920a6b1af850cddf41;
the subsequent packaging correction updates the manifest and license notices.
The initial cross/macOS matrices passed. Final packaging commit
847f6b4f37be32d49e5c9be1bdc30751d1836607 passed cross
[CI run 36631696935](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/36631696935)
and macOS [CI run 36631661693](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/36631661693).
All eight native targets passed. Linux unit tests, vet and race tests passed.
Source govulncheck reported zero reachable vulnerabilities, zero in imported
packages and one uncalled module advisory, as described in AUDIT.md. Docker
publication was intentionally skipped for the feature branch.

## Final Windows executable verification

Downloaded artifact: .tools/artifacts/windows-modern-847f6b4f/TorrServer-LT-windows-amd64.exe.
Version identifies the exact final packaging commit above and libtorrent 2.1.0.0.
SHA-256: A00CB418EFD3FBD4CCB345E5D10E417B6F086F907E6AA5B5CD4CC8C151D32C6A.

The executable ran on loopback port 18092, with fresh isolated state under
.tools/audit/modern-native-smoke/state. Chromium used the actual embedded UI and
actual backend, with no API fixtures for this check. The following passed:

- Modern HTML and local hashed JavaScript load; no legacy Firebase dependency.
- HTML/manifest no-cache and hashed JavaScript immutable cache headers.
- Correct icon dimensions and embedded notices for core/lazy dependencies.
- Echo, runtime, Flow tray/network, GST capability and empty-library reads.
- Missing settings object returns HTTP 400 without a restart.
- Settings Cancel leaves saved state unchanged; Apply persists FriendlyName and
  completes the real engine restart.
- Mobile Flow settings/pairing fit a 375 px viewport; screenshots inspected.
- Header pause/resume follows the actual backend response.
- No browser runtime exceptions; GET shutdown returns 200, process exits and
  the isolated listener closes.

Bonjour logged that no suitable IPv6 multicast interface was available on this
host; IPv4 advertisement continued. This environment limitation does not establish
IPv6 discovery compatibility. No torrent/media was loaded, no actual LAN phone
connection was tested, and no existing user process or state directory was changed.
