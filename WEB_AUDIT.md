# Modern interface audit and acceptance record

Specification: supplied Modern Web Interface plan, revision 2.0, sections 1–100.
Working branch: feature/modern-web. This is an implementation record, not a
Modern Web 1.0 release approval.

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
  Playwright and asset-generation tests. Final native CI is pending.
- Legacy executable 757577be was captured on isolated loopback port 18091 with
  fresh state. Desktop 1440x900 and mobile 375x812 screenshots are saved under
  .tools/audit/legacy-browser-baseline. No browser runtime exceptions occurred.
  The legacy mobile page measured 769 px wide at a 375 px viewport. Four legacy
  JavaScript assets total 635,052 bytes gzip; this is the asset sum, not a measured
  phone transfer. The baseline was reconstructed from the preserved audit build
  because the incoming modernization draft had already removed legacy sources.

Browser fixtures do not prove torrent throughput, media decoding or phone
intent handling. No real playback claims are derived from mocked tests.

## Remaining gates

Final native Windows/Linux/macOS CI, isolated executable checks and measured
production-browser performance remain to be recorded. The generated assets now
contain the modern interface. Repeat checks affected by later fixes.

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
