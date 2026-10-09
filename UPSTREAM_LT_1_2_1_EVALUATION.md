# TorrServer-LT 1.2.1 evaluation

Evaluated 9 October 2026 against Flow preview `.7`, source `4a871114`.
This is an evaluation, not an upstream merge or a playback benchmark.

## Scope and identity

- [LT release](https://github.com/trinity-aml/TorrServer-LT/releases/tag/MatriX.146.LT-1.2.1): published 9 October 2026 at 12:38:29 UTC; stable, not a prerelease.
- Tagged source: `922d8370c5b99d1b8ef47b36e727600b709460bd`.
- [Changes from 1.2.0](https://github.com/trinity-aml/TorrServer-LT/compare/MatriX.146.LT-1.2.0...MatriX.146.LT-1.2.1): HTTP read buffering plus version updates. DNS, session and HTTPS changes belong to the preceding 1.2.0, and are included cumulatively here.
- Read the complete comparison and relevant streaming, session, DNS, settings, listener, certificate and playlist source at the tagged revision. Ran the five isolated upstream buffer tests with Go 1.26.9. Downloaded the standard Windows executable, inspected its Go build metadata and scanned it without launching it.
- Windows asset: 76,865,536 bytes; SHA-256 `0525eb029473bde80384966636cc8180b9c392460f4366600495b93b714334fb`, matching the GitHub asset digest. Other LT binaries and actual Lampa/Just Player playback were not tested.

## Streaming change

The [buffering patch](https://github.com/trinity-aml/TorrServer-LT/commit/32170776491adea2e2187726219544d6c85d8ead) wraps the HTTP stream reader in a **1 MiB** `bufio.Reader`. Relative seeks account for unread buffered bytes; successful seeks reset the buffer, while failed seeks retain unread data. The caller's comment still says 256 KiB, although the actual constant is 1 MiB.

Its five tests passed locally: read/seek behavior, failed-seek preservation, fewer underlying HTTP reads, ordinary/suffix/open/invalid ranges, and multipart/HEAD behavior. These exercise standard-library readers, not Flow's native cache.

The author's cached-file measurements report roughly 35–37% lower CPU per GB at unrestricted transfer speed. That is **upstream evidence**, not a measured gain in Flow or torrent download speed. The benefit at ordinary single-client playback rates is smaller. A buffer does not create missing pieces or increase a seeder's upload capacity.

**Recommendation:** evaluate a bounded HTTP buffer in Flow, especially for disk cache, fast transfers and multiple clients. First check:

- Read-ahead must not count as client consumption. Flow currently updates its progress estimator, playhead and scheduling inside the underlying reader's `Read`.
- Range probes, close/warm-tail handling and multi-reader protection must use the correct logical offset, despite the underlying reader being ahead.
- Preserve short reads at unavailable cache boundaries; do not wait for a whole buffer before sending available bytes. Flow's current reader already returns available bytes before a hole.
- Bound memory across concurrent requests: 1 MiB is per reader, additional to the media cache. Avoid unnecessary large reads for small range probes.
- Test cancellation, cold seeks, sparse holes, eviction and tiny ranges on real native RAM/disk caches. Compare CPU, storage calls, first-byte latency, throughput and memory against the same unbuffered executable.

No buffer has been added to Flow by this evaluation.

## Cumulative changes and Flow fit

| Area | LT behavior | Flow assessment |
| --- | --- | --- |
| Session access | Locks session reads and rechecks session existence when adding torrents | Flow already has these protections. Preserve its existing shutdown/recovery model. |
| Engine publication | Avoids repeatedly assigning the same global engine pointer during reconnect | **Remaining Flow gap:** `InitApiHelper` still assigns `bts` unconditionally. Unsynchronized readers can race with that write during reconnect. This is a source-level finding; the exact Flow race was not reproduced in this evaluation. Add a targeted reconnect/race regression and fix publication before stable acceptance. |
| Tracker refresh | Separates refresh work from session handling | Flow already owns a cancellable refresh lifecycle; preserve its cleanup and tests. |
| DNS | Keeps working system DNS; fallback tests actual DNS answers in parallel | Better than choosing a resolver from UDP dial success alone. However, LT still runs several bounded DNS stages synchronously at startup. Flow removed DNS as a startup prerequisite; retain that behavior. Any fallback should be explicit and respect OS/VPN DNS. Go resolver changes do not establish native libtorrent DNS behavior. |
| HTTPS | Certificate hot reload, managed self-signed renewal, certificate status/upload, HTTPS-only mode, HTTP media compatibility and internal loopback access | Useful optional deployment features. Adapt through Flow's revisioned settings and modern UI; do not copy the old UI/settings implementation wholesale. |
| Listeners | Header/idle timeouts, graceful shutdown, bounded TLS error reporting | Flow already has managed listeners and bounded startup/error handling. An idle timeout is a possible small improvement; do not impose a total write timeout on long streams. |

The cumulative [HTTPS change](https://github.com/trinity-aml/TorrServer-LT/commit/8a2173280ac75070e77708280273e2025282eb57) has sensible safeguards: validated certificate/key pairs, bounded uploads, authenticated management routes and no private-key download. Three adaptations matter:

1. **Fail closed for explicit HTTPS-only operation.** LT disables `settings.Ssl` on a bad certificate; `HTTPEnabled()` then returns true, even when HTTPS-only was requested. That can open plain HTTP unexpectedly. This follows the source control flow; no live LT failure reproduction was performed. Preserve Flow's current startup rejection for invalid configured certificates.
2. **Retain Windows key protection.** LT's atomic key writes use Unix modes and `Chmod`; these do not establish a restrictive Windows DACL. Flow's Windows private-file protection must remain in any port.
3. **Make HTTP media an explicit compatibility choice.** It serves media without TLS. Neither HTTPS to the phone nor these options hide BitTorrent peer traffic from the ISP. Self-signed certificates also require client trust; browser acceptance alone does not establish Just Player compatibility.

## Dependencies and security

The downloaded LT Windows binary confirms **Go 1.25.7**, `x/net` **v0.55.0**, `x/text` **v0.37.0**, and tagged source revision `922d8370`. Its source build configuration retains libtorrent **v2.1.0**; the actual native library version was not verified by running this executable. Flow preview `.7` confirms Go **1.26.9**, `x/net` **v0.60.0** and libtorrent **2.1.2.0**.

`govulncheck` v1.8.0 reported **56 advisory IDs** for the LT executable, including [HTTP/2 server crashes](https://pkg.go.dev/vuln/GO-2026-6617) and [unbounded Range parsing](https://pkg.go.dev/vuln/GO-2026-6609). The executable is stripped: when symbol extraction is unavailable, this scanner falls back to module-level precision. This is **not proof of 56 reachable or exploitable flaws**. Nevertheless, the old dependency/runtime versions are confirmed, and rebuilding with patched versions is warranted before adoption. The scan covers Go dependencies, not the complete native stack.

For comparison, Flow's stripped executable reports only [GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932), for unmaintained OpenPGP packages that Flow does not import. Its exact tagged native Linux source scan reports zero reachable vulnerabilities. See [Flow release verification](PREVIEW_2_7_VERIFICATION.md) for the distinction and local Windows source-scan limitation.

## Decision

Keep Flow's current architecture, updated dependencies and streaming defaults. Prioritize the engine-publication race fix; then evaluate HTTP buffering with Flow-specific measurements. Certificate hot reload and HTTPS-only deployment are useful later additions, with the safeguards above. Do not replace Flow with LT 1.2.1 or bulk-merge its source based on the release number or cached-file benchmark.

This release adds a useful I/O optimization and deployment features. It provides no demonstrated replacement for Flow's sparse-swarm/high-bitrate scheduling, and no guarantee of better playback on the user's torrents.
