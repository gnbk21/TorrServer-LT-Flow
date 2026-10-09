# Flow 1.0 verification

This record distinguishes implementation, automated verification, published
artifact verification and unfinished physical field checks. Publication remains
blocked until the required automated gates pass. The owner authorized Latest
publication with the field limitations in `RELEASE_1_0_NOTES.md`.

## Local checks

- Web: TypeScript, ESLint, production build, 70 unit tests and 38 browser tests
  pass. One cross-language contract test runs only when CI exports native DTOs.
- Browser GST discovery allows the bounded cold-source retry to finish before
  starting HLS; regression checks cover the old timeout and closing mid-probe.
- Production HTTP wrapper: read/seek, failed seek, ordinary/suffix/open-ended/
  invalid/multipart Range, HEAD and buffered cancellation regressions pass.
- TLS splitter: first-byte routing preserves sniffed bytes and shutdown closes
  silent pending clients; isolated Windows loopback test passes.
- GST FFI: 17 isolated tests using the original production API/probe functions
  pass with strict pointer checking (`-d=checkptr=2`) on Windows. Typed pointers
  preserve provenance through errors, samples, callbacks and parsed segments;
  full native runtime/race validation remains a separate CI gate.
- Five-iteration 16 MiB memory/file microbenchmark: 513 source reads become 17;
  roughly 1 MiB additional transport allocation per active reader. This small
  synthetic test was slower with buffering, so it establishes call amortization,
  not a CPU or throughput improvement. Native streaming CI is the behavior gate.

## Native and release gates

Final native/platform, tagged packaging and downloaded-artifact evidence is
recorded here after those operations finish. Initial checkpoint `a8f91a9a`
passed native core/certificate tests but failed the GST command fixture lookup;
the fixture was corrected to use standalone command discovery via PATH.
That failed checkpoint is not release acceptance evidence.

## Unfinished field checks

Broader Android/player coverage, public-swarm comparisons, multi-hour resource
measurements and physical Windows boot/sleep/network recovery remain unfinished.
Existing smooth episode reports relate to source `52d0ad20`, not this final
tagged executable. Browser automation and controlled peers do not establish
phone decoder, Wi-Fi, ISP or every swarm's behavior.
