#!/usr/bin/env bash
# Native Windows build under MSYS2 UCRT64. Reuses the pinned dependency recipe;
# its install tree is separate from the Linux-hosted Windows cross-build cache.
set -euo pipefail
[[ "${MSYSTEM:-}" == UCRT64 ]] || { echo 'Run in the MSYS2 UCRT64 environment.' >&2; exit 1; }

TARGET=windows-native-amd64
BINEXT=.exe
GOOS=windows GOARCH=amd64
CC=gcc
CXX=g++
B2_VARIANT=flowlocal
# Native b2 does not search the MSYS POSIX PATH like bash does.
B2_TOOLSET_CXX="\"$(cygpath -m "$(command -v g++)")\""
# Keep archive/response-file paths below Win32 tool limits in deep checkouts.
B2_FLAGS='--hash target-os=windows address-model=64 architecture=x86'
OPENSSL_CONFIG_TARGET=mingw64
EXTRA_CGO_LDFLAGS='-static -static-libgcc -static-libstdc++ -lws2_32 -liphlpapi -lmswsock -lsecur32 -lcrypt32 -lbcrypt -ladvapi32 -luser32 -lshell32 -lgdi32'
export CMAKE_GENERATOR='MSYS Makefiles'
# Older bundled WebRTC CMake projects predate CMake 4's policy floor.
CMAKE_CROSS_ARGS='-DCMAKE_C_COMPILER=gcc -DCMAKE_SYSTEM_NAME=Windows -DCMAKE_SYSTEM_PROCESSOR=x86_64 -DCMAKE_POLICY_VERSION_MINIMUM=3.5'
export TARGET BINEXT GOOS GOARCH CC CXX B2_VARIANT B2_TOOLSET_CXX B2_FLAGS \
    OPENSSL_CONFIG_TARGET EXTRA_CGO_LDFLAGS CMAKE_CROSS_ARGS

. "$(dirname "$0")/_deps.sh"
cross_build
