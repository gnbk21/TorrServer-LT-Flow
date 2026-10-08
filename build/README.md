# Local cross-build (no Docker)

These scripts build **TorrServer-LT** for every supported target straight on a
Linux host — no Docker, no QEMU. Each one builds a static OpenSSL, the WebRTC
deps (usrsctp/libjuice, via cmake) and libtorrent (plus `boost_system` straight
from the Boost source tree, via Boost.Build/`b2`) into `_deps/<target>/`, then
links the Go binary against it through pkg-config. Final binaries land in
`_out/TorrServer-LT-<target>`. On the platforms a GStreamer runtime exists for
(linux amd64/arm64, windows amd64, macOS amd64/arm64) a second
`_out/TorrServer-LT-<target>-gst` is also built with `-tags gst` — the same
binary plus the GStreamer HLS transcoding feature (pure Go; GStreamer is
`dlopen`'d at runtime, so the deps above are unchanged).

libtorrent's own `Jamfile` emits a correct `libtorrent-rasterbar.pc` from its
`install` target, so the per-target Cflags/defines/Libs flow straight into cgo
(`server/lt` uses `#cgo pkg-config: libtorrent-rasterbar`). libtorrent is built
with `crypto=openssl` (static, built from source — no system OpenSSL needed)
and `webtorrent=on`: https trackers/web seeds plus WebTorrent (`wss://`
trackers, WebRTC browser peers) work on every target. libdatachannel's Jamfile
would build usrsctp/libjuice with the HOST compiler; `_deps.sh` cross-builds
them itself and patches that Jamfile to consume the prebuilt archives.

Flow also declares a narrow, non-virtual cache-refetch extension in the pinned
libtorrent headers. Its implementation is in `server/lt/lt_shim.cpp`: it clears
completed-state flags, corrects file progress and refreshes seed connect
candidates when previously downloaded media has been evicted. Native source and
installed headers receive identical declarations before compilation. Old cache
trees rebuild libtorrent once; `.flow-cache-extension-v3` identifies the new
recipe. The extension changes no object layout or swarm-profile settings. A
shared distro libtorrent without the internal API retains the documented limited
refetch fallback; use the supplied static builds for full Flow behavior.

```
build/
  _common.sh         paths + pinned versions (Boost 1.85.0, libtorrent v2.1.2, OpenSSL 3.5.7)
  _fetch_sources.sh  download Boost + OpenSSL + clone libtorrent into _src/  (idempotent)
  _deps.sh           shared engine: openssl + webrtc deps + b2 install libtorrent → go_build
  _osxcross.sh       locate OSXCross wrappers/SDK (darwin targets)
  all.sh             build every target whose toolchain is present
  <target>.sh        per-target toolchain setup + cross_build
```

`_src/`, `_deps/`, `_out/` are git-ignored.

## One command

```bash
build/all.sh                       # everything the host can build
TARGETS="linux-arm64" build/all.sh # a subset
```

`all.sh` never silently drops a target: each is reported `OK` / `FAIL` /
`SKIP` (toolchain missing).

## Prerequisites per target

The host needs `go`, `curl`, `git`, `cmake` (for the WebRTC deps) and a host
C++ compiler (to bootstrap Boost's `b2`).

| Target          | Install                                                                 |
|-----------------|-------------------------------------------------------------------------|
| `linux-amd64`   | nothing extra — uses the host gcc/g++ (libtorrent built static)         |
| `linux-arm64`   | `sudo apt install gcc-aarch64-linux-gnu g++-aarch64-linux-gnu`           |
| `linux-armv7`   | `sudo apt install gcc-arm-linux-gnueabihf g++-arm-linux-gnueabihf`       |
| `windows-amd64` | `sudo apt install gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64`             |
| `android-arm64` | Android NDK r26+ (tested r29), `export ANDROID_NDK_HOME=/path/to/android-ndk-r29` |
| `android-armv7` | same as android-arm64                                                   |
| `darwin-arm64`  | OSXCross, `export OSXCROSS_ROOT=/opt/osxcross` (see ⚠️ below)            |
| `darwin-amd64`  | same as darwin-arm64                                                     |

One apt line for the three apt-installable cross targets:

```bash
sudo apt install \
  gcc-aarch64-linux-gnu g++-aarch64-linux-gnu \
  gcc-arm-linux-gnueabihf g++-arm-linux-gnueabihf \
  gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64
```

> ⚠️ **macOS targets** need the Apple macOS SDK via OSXCross. Apple's licence
> only permits that SDK on Apple hardware, so cross-building macOS binaries on
> Linux is a legal grey area. For anything distributed, build on a real Mac or
> the macOS CI runner (`.github/workflows/build-macos.yml`). The scripts exist
> so "all architectures locally" is reproducible where the SDK is available.
>
> The darwin scripts accept either an OSXCross source build (`target/bin`) or a
> prebuilt `crazymax/osxcross` tree (`bin/` + SDK). To get a ready toolchain
> with an arm64-capable SDK (>= macOS 11) without building OSXCross — Docker is
> used once only, to extract; the TorrServer build itself stays Docker-free:
>
> ```bash
> docker create --name oxc crazymax/osxcross:13.1-ubuntu   # SDK 13.1, arm64+x86_64
> docker cp oxc:/osxcross /opt/osxcross
> docker cp oxc:/osxsdk   /opt/osxcross/SDK
> docker rm oxc
> sudo apt install -y clang        # wrappers drive the host clang
> export OSXCROSS_ROOT=/opt/osxcross
> ```

## Notes

- Every target — `linux-amd64` included — builds libtorrent from source and
  links it statically; the only dynamic deps are libc/libstdc++/libgcc (and on
  Windows nothing extra, those are linked static too).
- Boost/libtorrent are built once per target and cached in `_deps/<target>/`;
  re-running a script is a fast no-op for the deps and only relinks Go.
- Versions are pinned in `_common.sh`; override with e.g.
  `LIBTORRENT_TAG=<tag> LIBTORRENT_REV=<commit> build/linux-arm64.sh`.
  The checkout must match the revision and every reviewed patch must apply.

## Reviewed native fixes

Libtorrent v2.1.2 is pinned to `6da363d2994f17c0b3c0450d124cf73a31a73847`.
`native-patches/94bffc25272b-queue-time.patch` backports upstream commit
`94bffc25272b6833108d601fc0d824c344c48bc3`, including its queue arithmetic and
transfer simulations. That fix is not part of the release tag.
`flow-private-tracker-tiers.patch` makes private torrents use native tier
failover even when public torrents announce to all configured trackers.

`apply_native_patches.py` checks exact source context, breaks source hardlinks,
and accepts an already-applied patch. Its recipe digest participates in the
installed dependency stamp and CI cache key. The `native-streaming-tests` job
runs the upstream queue/transfer simulations before platform jobs proceed.
The Go/CGo stamp includes the native revision and patch recipe too: a backport
must relink the executable even when its pkg-config file is unchanged.
`flow-web-seed-destinations.patch` bounds imported sources and rechecks native
destination addresses, including redirects and cached resolutions.

The controlled peer fixture supports partial bitfields, delayed HAVE and
unchoke, metadata exchange, separate suppliers, rate limits, outages and
reconnects. `python -m unittest discover -s build -p '*_test.py'` checks the
fixture's real socket protocol independently of the production engine.

For an isolated comparison, dispatch `build.yml` with
`evaluate_sparse_baseline=true`. This opt-in job builds the original v2.1.0
runtime, v2.1.2 without the queue patch, the exact queue backport and the current
candidate. It repeats byte-checked generated fixtures three times in varied
order, then characterizes Legacy/Conservative/Balanced and the disabled-by-default
scarce-piece experiment. Its `flow-sparse-native-layers` artifact retains reports,
fixture metadata and executable hashes. Longer native reconnect waits may fail
a profile characterization case; those failures remain explicit in the report.
HTTP read wait is not the player's decoded-frame or stall time.

The recorded comparison and its individual failures are in
[SPARSE_VERIFICATION.md](../SPARSE_VERIFICATION.md) and
[SPARSE_COMPARISON.json](../SPARSE_COMPARISON.json). Rate-aware deadlines now
require `Flow.RateAwareDeadlines=true`; `sparse_harness.py --rate-aware` exercises
that policy separately. The default harness retains the graded ramp and includes
a blocked seek whose required piece arrives later. It checks the actual missing
piece diagnostic before checking the eventual Range bytes. The historical
comparison candidate had rate-aware deadlines automatically enabled.

## High transport-bitrate and certificate checks

`high_bitrate_check.py` owns a real server, loopback tracker and generated MPEG-TS
fixture. The transport includes padding; it tests byte delivery at a stated
rate, not visual complexity or decoded frames. Public discovery is disabled.
The peer fixture's opt-in high-throughput polling follows scheduled block times
and allows bounded socket backpressure. A direct, byte-checked peer calibration
must exceed 1.3 times nominal demand before a comparison runs.
High bitrate cases advertise a BEP 10 request capacity of 512 before unchoking,
and retain the fixture's bounded 2,048-request ceiling for cancellation headroom.
Reports retain queue peaks and closure reasons. Earlier cases without this
advertisement are diagnostic evidence, not clean source-capacity comparisons.

```text
python build/high_bitrate_check.py --executable <binary> --output <new-directory> --profile adaptive --mbps 90 --seconds 120 --cache-mb 512
python build/high_bitrate_check.py --executable <other-binary> --output <another-new-directory> --fixture <first-directory>/fixture/generated.ts --profile legacy --mbps 90 --seconds 120 --cache-mb 512
python build/upstream_tls_check.py --executable <binary> --output <new-directory>
```

Use sequential, rotated runs on the same machine and fixture. Reports include
exact Range bytes, waits, seeks, cancellation, observed cache/RSS and process CPU
where available. Add `--preload` to exercise the Lampa-style explicit bootstrap
and metadata probe before the same paced Range workload. Without it, direct
Range requests also exercise missing-metadata fallback. Keep startup mode equal
within each comparison and record startup time separately from playback waits.
Existing output/state is never overwritten. TLS checks cover
invalid custom files, partial CLI pairs, a lone default key, generated HTTPS
and read-only DB startup. They verify no fallback listener and no replacement of
user files. Windows ACLs also have a native platform regression test.
