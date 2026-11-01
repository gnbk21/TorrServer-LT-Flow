package version

// Version is set at build time via -ldflags "-X server/version.Version=<tag>"
// (release builds stamp the git tag name). The leading number is significant:
// the Android client (ru.yourok.torrserve) derives the server's "MatriX
// version" from the FIRST digit run in this string and gates its settings UI on
// it — e.g. it only shows the PreloadCache field (and hides the dead legacy
// PreloadBuffer switch) when that number is > 131. Keep the first number >= 132
// so the client unlocks the modern controls. "148" identifies the reviewed
// upstream maintenance baseline; Flow retains its own feature set and semantic
// version. New releases use MatriX.148.Flow-vX.Y.Z. Historical MatriX.145.Flow
// identities remain recognized by the release and update helpers.
var Version = "MatriX.148.Flow-dev-local"
