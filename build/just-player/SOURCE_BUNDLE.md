# Experimental player source bundle

`upstream.tar` preserves the immutable Just Player application at the commit in
`SOURCE_MANIFEST.json`, including its original bundled decoder AARs. The manifest
checks those AAR bytes against the actual build inventory and hashes every
published recipe file. It identifies the Flow commit separately from the upstream
commit. `SHA256SUMS.txt` identifies the separate debug APK.

`UPSTREAM_LICENSE.txt` is the upstream application's original license.
`FLOW_PATCH_LICENSE.txt` is Flow's GPL-3.0 license for the additions in
`flow-patch/`, `prepare_just_player.py`, `player_patch_check.py` and
`player_sbom.py`. These licenses do not replace third-party component licenses.

To reproduce the application, check out the immutable player commit, apply
`prepare_just_player.py` with `flow-patch/` beside it, and follow the preserved
`BUILD_WORKFLOW.yml` toolchain and Gradle steps. The Gradle inventory script is in
`flow-patch/inventory.gradle`. This requires the Android SDK, Java and resolved
Gradle dependencies; the application archive alone is not an offline build kit.

The bundled decoder AARs contain native binary libraries. Their exact bytes are
inventoried, but the application archive does not establish the corresponding
native source, configuration or license mapping for those binaries. The upstream
`app/libs/README.md` gives general FFmpeg instructions, not a complete immutable
recipe for all bundled decoders. That source/notices audit remains open. A passing
CycloneDX schema check does not settle this audit or validate phone playback.
