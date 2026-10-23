# YouROK MatriX.147.1 review

Reviewed 23 October 2026. Upstream tag `MatriX.147.1` resolves to
`d84f41968780744d4b8643f7a73b22a84d2c3779`; its comparison base is
`MatriX.147`, `a86d8fa7fd4e07605f240fe69bf299392f7cb023`.
Flow source inspected: `9fdfcea3` (documentation follow-up to the published
1.1.4 runtime at `1b1a92e0`). This is source review and a local query-decoding
reproduction, not an upstream executable benchmark or a change to Flow's runtime.

Sources: [release](https://github.com/YouROK/TorrServer/releases/tag/MatriX.147.1),
[six-commit comparison](https://github.com/YouROK/TorrServer/compare/MatriX.147...MatriX.147.1),
[engine dependency comparison](https://github.com/tsynik/torrent/compare/v1.2.32...v1.2.34).

## What actually changed

The release description lists three pull requests. The complete tag comparison
also includes a torrent-engine dependency update and a metadata-wait crash fix.
Generated legacy web assets account for much of the remaining file churn.

| Change | Effect | Assessment for Flow 1.1.4 |
| --- | --- | --- |
| `tsynik/torrent` 1.2.32 → 1.2.34 | An empty DHT lookup no longer holds the announcer for a fixed five minutes. Subsequent retries start at 15 seconds with bounded interval growth. | Useful cold-start idea, but Flow uses libtorrent 2.1.2, not this engine. Do not copy the dependency or claim the same delay exists in Flow. |
| Snapshot `t.Torrent` in `WaitInfo()` | Avoids a nil-pointer panic when removal/drop races a metadata waiter, including Lampa requests. | Flow already captures its atomic native handle once and has a close channel. The upstream pointer field and cache wiring do not exist in Flow. Add a direct concurrent metadata-wait/removal regression when extending lifecycle coverage; no blind cherry-pick. |
| Safe second query unescape, PR #928 | Keeps an already-decoded value containing a literal invalid percent escape instead of replacing it with an empty string. | A confirmed applicable defect: both stream handlers, Rutor, Torznab and Flow's JacRed handler discard this decoding error. Highest-priority small backport, extended to Flow-specific callers. |
| Complete Windows-1251 mapping, PR #923 | Corrects additional Cyrillic letters, punctuation and symbols in subtitle conversion, including Ukrainian text. | Flow does not have upstream's automatic torrent-sidecar conversion module. Preserve as a prerequisite for future external-subtitle work; it does not change Just Player's own decoder or fix streaming supply. |
| `follow-redirects` 1.16.0 → 1.16.1, PR #930 | Updates the legacy web dependency lockfile. | Flow's current package.json/yarn.lock contain neither axios nor follow-redirects. There is no corresponding dependency to bump. Keep the modern UI and lockfile. |

## Confirmed Flow query defect and upstream limits

Flow currently calls `c.Query(...)`, which already decodes URL query encoding,
then calls `url.QueryUnescape` again and discards its error. The inspected sites:

- `server/web/api/stream.go`: authenticated and compatibility playback handlers;
- `server/web/api/rutor.go`, `torznab.go` and `jacred.go`: searches;
- `server/web/api/utils/testlink.go`: a related pre-authentication link validator
  also decodes, but returns errors rather than discarding them.

A standalone Go 1.26.10 reproduction used `url.Values.Encode`, `url.ParseQuery`
and the exact second-unescape expression. It did not contact external services
or start the owner's server. Results:

| Logical input after first decode | Current second decode | Upstream helper |
| --- | --- | --- |
| `100% Pure` | Empty, error discarded | Preserved |
| `50%` | Empty, error discarded | Preserved |
| `%zz` | Empty, error discarded | Preserved |
| `C++` | `C  ` | Still `C  ` |
| `https://mirror.example/file.torrent?token=A%2BB` | Token changes to `A+B` | Still changes |
| `Caf%C3%A9` | `Café` | `Café`, retaining legacy double-encoding compatibility |

The token is fictional. Altering nested URL encoding can invalidate signatures
or change source identity; not every URL or title is affected. Upstream fixes
error handling, not the entire double-decoding ambiguity. A Flow adaptation
should retain required legacy client compatibility, avoid corrupting valid
URLs/plus signs, and validate the same canonical link that will be executed.
Exercise both stream paths and all three searches with normal, double-encoded,
literal-percent, plus-sign, Unicode, encoded-separator and authentication cases.
Do not remove compatibility decoding globally without checking actual callers.

## DHT relevance to startup and scarce swarms

The dependency comparison has two commits implementing the empty-swarm retry.
It can help upstream find peers sooner when a torrent is added before the DHT
routing table has useful contacts. It does not improve an established supplier's
upload rate or create missing pieces. Its tests use an immediately empty fake
DHT traversal and interval arithmetic; they are not public-swarm throughput data.

Flow already:

- forces initial tracker/DHT activity for new magnets;
- saves/restores bounded DHT routing state when enabled, including late-listener
  bootstrap handling in the native shim;
- configures a 60-second DHT interval for Legacy/Adaptive; other named profiles
  clear that override and retain native defaults;
- reannounces on qualified network recovery and preserves bounded metadata
  fallback with public-cache consent.

These measures do not prove every cold-start case is covered. A useful next
investigation is an empty-routing-table magnet followed by delayed bootstrap,
with actual lookup/peer-arrival timing recorded separately from metadata and
payload delivery. Only if a delay is reproduced should Flow add bounded,
readiness-aware retries through its existing native handle. Preserve private
torrent isolation, disabled-DHT/interface policy, cancellation, global request
limits and ordinary scheduling after peers appear. Do not add a second DHT stack
or repeatedly reset healthy swarms.

## Recommendation

1. Adapt the percent-query correctness fix, including JacRed and canonical
   validation/decoding regressions; do not copy the helper without plus/URL tests.
2. Measure cold-start DHT behavior before adopting an additional retry policy.
3. Extend metadata-wait/removal lifecycle regression coverage when changing that
   path. Existing Flow atomic-handle protection addresses the upstream panic's
   pointer-lifetime pattern, but is not proof of all possible lifetime races.
4. Keep complete CP1251 support with the separately scoped external-subtitle
   feature. No reason to replace Flow's engine, toolchain or modern frontend.

The release is primarily maintenance with one useful startup-discovery change.
It provides no measured high-bitrate playback improvement for Flow and no new
piece scheduler or remedy for insufficient sustained peer supply.

Evidence retained locally under `.tools/upstream1471`: release/compare API
responses, dependency diff, pinned libtorrent source and `query_repro.go`.
No Flow executable, settings, running session or published release was changed
by this review.
