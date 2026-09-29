# Modern interface implementation ledger

Specification: revision 2.0, sections 1–100, in the supplied engineering plan.
Current branch: feature/modern-web. Audit build 757577be is verified separately
in AUDIT.md. Legacy remains recoverable on develop at that commit, with a local
archive/reference in .tools/audit. The incoming draft is separately archived.

Unchecked means implementation or verification remains; existence of a file is
not acceptance. Real device checks require the user's phone.

- [x] Read the complete specification; inspect draft and actual Go API contracts.
- [x] Finish prior audit CI and isolated Windows executable verification.
- [x] 1–9, 74–75: React 19, Vite 8, strict TS 5, Tailwind 4, Radix, RHF,
  TanStack Query 5, HashRouter shell, four pages, baseline and legacy preservation.
- [x] 6, 10–13, 68–70, 76: typed/validated API, authentication, timeouts,
  cancellation, structured errors, adaptive/visibility polling, complete proxies.
- [x] 14–23, 54, 79–80: dashboard idle/active, accurate unknown/health/starting
  states, authoritative buffer data, bounded charts, complete Flow/runtime metrics.
- [x] 24–30, 71, 77: library states/filter/sort/actions, metadata editing,
  multi-add/upload/import/export, cache map, file tree/episode parsing/viewed,
  playlists, internal player/subtitles, external players and confirmations.
- [x] 31–32, 61, 81: configured search providers, filters, surfaced errors,
  add/metadata/prepare/playable workflow, TMDB enhancement and poster fallback.
- [x] 33–40, 82–83: ranked safe LAN addresses/manual selection, IPv6/HTTPS,
  QR without credentials/loopback, LAN HTTP copy fallback, Just Player intents.
- [x] 41–44, 84: manifest/icons, basic LAN HTTP operation, truthful install help;
  no service worker required or media caching.
- [x] 45–53, 55, 78: all BT/Flow/custom fields and bounds, unknown preservation,
  no autosave, dirty leave/discard, restart confirmation, WAF independent save,
  integration/security/storage/GST settings, danger confirmations.
- [x] 56–60, 93–98: responsive slate UI, accessible dialogs/focus/touch/reduced
  motion, maintained languages, failure/retry/stale data and empty states.
- [x] 62–67, 85: build/embed integration, no source maps, cache headers,
  lazy chunks, initial compressed size and profiler/polling measurements.
- [x] 72–73: parity reconciliation; legacy retained until release gates below.
- [x] 87–90: meaningful unit/component/E2E tests and five responsive widths,
  landscape, keyboard/focus, offline/503/recovery, settings and player fallback.
- [ ] 91–92: real Windows/Android playback, seeking, background/network recovery,
  long session memory and performance acceptance; record measured evidence.

Incoming draft defects corrected during implementation: missing entry/pages; wrong stack majors; wrong torrent enum,
Flow/runtime DTOs, session query key and shutdown method; defaults read resets
settings; invented TMDB write route and file-deletion flag; swallowed search
errors; missing focus trap; hardcoded English and incomplete settings fields;
conditional chart hook; unknown metrics displayed as zero/healthy; unsafe pairing
selection and HTTP-only Android intents. These are not accepted functionality.

## Reconciliation status on 30 September 2026

- [x] 62–70: production build, modern asset generation, cache policy and proxy/auth
  compatibility are implemented and locally verified. Native executable gate
  remains pending below.
- [x] 66–67, 85, 92–93: measured first dashboard JS (186,769 bytes gzip), lazy
  player/language/auxiliary chunks, 180-second Chromium CPU/heap sample and bounded
  visibility-aware request counts. Real mobile CPU and multi-hour memory remain
  acceptance tests; no hardware performance claim is made.
- [x] 71–73: legacy feature matrix reconciled in WEB_AUDIT.md; preserved source,
  actual legacy screenshots/bundle baseline, feature branch and draft PR #2.
- [x] 75, 99 (build gates): final cross/macOS matrices, Linux unit/vet/race/security
  checks and actual Windows executable/UI/API smoke passed; recorded in WEB_AUDIT.md.
- [ ] 82–83, 91–92, 99–100: user-owned real-phone/Just Player, large-stream,
  network/background/seek and multi-hour acceptance, per WEB_DEVICE_CHECKLIST.md.
- [ ] 86: legacy removal intentionally held until the required acceptance gates.

No service worker or SSE was introduced; both are optional in the plan. The
legacy independent speed-test download is explicitly excluded with rationale in
WEB_AUDIT.md; actual torrent/runtime rates remain available. Baseline screenshots
were recovered after the incoming draft had removed legacy source, using the
preserved verified native artifact rather than an invented historical result.
