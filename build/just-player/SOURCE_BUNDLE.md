# Experimental player source bundle

`upstream.tar` preserves the immutable Just Player application at the commit in
`SOURCE_MANIFEST.json`, including its original bundled decoder AARs. The manifest
checks those AAR bytes against the actual build inventory and hashes every
published recipe file. It identifies the Flow commit separately from the upstream
commit. `SHA256SUMS.txt` identifies the separate debug APK.

`UPSTREAM_LICENSE.txt` is the upstream application's original license.
`FLOW_PATCH_LICENSE.txt` is Flow's GPL-3.0 license for the additions in
`flow-patch/`, `prepare_just_player.py`, `player_patch_check.py` and
`player_sbom.py` and `player_sources.py`. These licenses do not replace third-party
component licenses. The source packager preserves their original notices separately.

To reproduce the application, check out the immutable player commit, apply
`prepare_just_player.py` with `flow-patch/` beside it, and follow the preserved
`BUILD_WORKFLOW.yml` toolchain and Gradle steps. The Gradle inventory script is in
`flow-patch/inventory.gradle`. This requires the Android SDK, Java and resolved
Gradle dependencies; the application archive alone is not an offline build kit.

## Native decoder provenance and notices

All six application AARs match the original `moneytoo/media` build at
`7286492527b8db7116f56abe836cba62002e3029`, run `34632899792`. The checksum lock
in `flow-patch/decoder-sources.json` identifies that build, all six AAR digests,
and the immutable FFmpeg, dav1d, cpu_features, IAMF and MPEG-H source revisions.
It resolves the original workflow's moving FFmpeg/dav1d branches to the exact
checkout identities recorded in the successful upstream log.
Most archives use an exact compressed-byte checksum. Gitiles regenerates the
FDK AAC archive with varying compressed bytes; its lock also pins all source
file paths, contents, executable bits and link targets. Packaging verifies that
425-entry source tree and records the actual downloaded archive checksum. Source
or notice changes still fail verification; compression metadata alone may vary.

`decoder-sources/archives/` preserves those source archives, the complete Media3
fork and IAMF's declared native dependency sources. `notices/` reproduces original
license, copying, notice and patent files. PFFFT's source header is preserved
intact because it contains that component's notice. No component is relicensed
by the application or Flow patch license. MPEG-H and FDK AAC contain their own
conditions and patent statements; their original terms remain authoritative.

`DECODER_SOURCE_REPORT.json` and the outer source manifest bind archives and
notices to the actual AAR inventory. A mismatched AAR, source checksum or notice
fails packaging. The source bundle is approximately 618 MiB before artifact
compression, mainly upstream test media; it is not an offline Android/Bazel
toolchain. It retains all four decoder extensions and both custom UI/runtime
libraries.

This proves source provenance and preserves notices. An independent native
rebuild, the complete resolved Bazel/toolchain closure and redistribution
compatibility remain open; `native_corresponding_source_verified` stays false.
A passing CycloneDX schema check does not settle those checks or validate phone
playback. The experimental player is not a Flow stable release asset.
