#!/usr/bin/env bash
# Isolate the dependency upgrade, queue backport and Flow runtime changes.
# Runs only in an explicitly requested benchmark job, never in a release build.
set -euo pipefail
project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
evaluation_root="$project_root/_out/sparse-native-layers"
mkdir -p "$evaluation_root"
original_revision=738649bb7e6145aeaf9d8014815dd2a5e96a2852
queue_revision=d7aafde53bf2b2d3dd9086df7f3bf1691a34ba97
git -C "$project_root" fetch --no-tags --depth=1 origin "$original_revision" "$queue_revision"

archive_baseline() {
    local revision="$1" destination="$2"
    mkdir -p "$destination"
    git -C "$project_root" archive "$revision" | tar -x -C "$destination"
    # Each baseline owns its sources and dependencies. A checkout/patch must
    # never mutate another variant's cached libtorrent tree or installed headers.
    cp -a "$project_root/_src" "$destination/_src"
    cp -a "$project_root/server/web/pages/template/." "$destination/server/web/pages/template/"
}

archive_baseline "$original_revision" "$evaluation_root/original"
archive_baseline "$queue_revision" "$evaluation_root/stable"
queue_patch="$evaluation_root/stable/build/native-patches/94bffc25272b-queue-time.patch"
cp "$queue_patch" "$evaluation_root/queue-time.patch"
rm "$queue_patch"

(cd "$evaluation_root/original" && TS_VERSION=MatriX.145.Flow-benchmark-original JOBS=2 bash build/linux-amd64.sh)
(cd "$evaluation_root/stable" && TS_VERSION=MatriX.145.Flow-benchmark-stable JOBS=2 bash build/linux-amd64.sh)
cp "$evaluation_root/original/_out/TorrServer-LT-linux-amd64" "$evaluation_root/original.bin"
cp "$evaluation_root/stable/_out/TorrServer-LT-linux-amd64" "$evaluation_root/stable.bin"
cp "$evaluation_root/queue-time.patch" "$queue_patch"
# cross_build's recipe guard deliberately rebuilds the native dependency when
# the exact patch changes. Installed libraries cannot silently survive this step.
(cd "$evaluation_root/stable" && TS_VERSION=MatriX.145.Flow-benchmark-queue JOBS=2 bash build/linux-amd64.sh)
cp "$evaluation_root/stable/_out/TorrServer-LT-linux-amd64" "$evaluation_root/queue.bin"

python3 "$project_root/build/sparse_layer_compare.py" \
    --original "$evaluation_root/original.bin" --stable "$evaluation_root/stable.bin" \
    --queue "$evaluation_root/queue.bin" --candidate "$project_root/_out/sparse-current/TorrServer-LT-linux-amd64" \
    --fixtures "$evaluation_root/fixtures" --output "$evaluation_root/results"
