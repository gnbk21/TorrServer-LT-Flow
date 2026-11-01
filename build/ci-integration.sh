#!/usr/bin/env bash
# Independent groups run on separate hosted runners, never alongside another
# policy comparison on the same host. Preserve every previous integration case.
set -euo pipefail
binary=_out/TorrServer-LT-linux-amd64
case "${1:-}" in
    playback)
        python3 build/playback_harness.py --executable "$binary" --output _out/controlled-playback --duration 20 --debug
        ;;
    startup)
        python3 build/startup_reserve_check.py --executable "$binary" --output _out/startup-reserve-check
        python3 build/fixtures.py --output _out/startup-fixtures
        python3 build/startup_harness.py --executable "$binary" --fixtures _out/startup-fixtures --output _out/startup-check --assert-fast
        python3 build/startup_harness.py --executable "$binary" --fixtures _out/startup-fixtures --output _out/startup-custom-check --profile custom --assert-fast
        mkdir -p _out/startup-piece-fixtures
        # VBR size depends on source complexity; two repetitions need not cover
        # the large-piece case's 64-MiB minimum. Remux to an 80-MiB byte target,
        # with a separate time bound. The harness still enforces its minimum.
        timeout 120s ffmpeg -hide_banner -loglevel error -stream_loop -1 -i _out/startup-fixtures/variable.mp4 -c copy -fs 83886080 -movflags +faststart _out/startup-piece-fixtures/variable.mp4
        python3 build/startup_piece_check.py --executable "$binary" --fixtures _out/startup-piece-fixtures --output _out/startup-piece-ordering --case ordering --assert-fixed
        python3 build/startup_piece_check.py --executable "$binary" --fixtures _out/startup-piece-fixtures --output _out/startup-piece-progress --case progress --assert-fixed
        python3 build/startup_piece_check.py --executable "$binary" --fixtures _out/startup-piece-fixtures --output _out/startup-piece-missing-index --case missing-index --assert-fixed
        ;;
    discovery)
        python3 build/discovery_harness.py --executable "$binary" --output _out/discovery-check
        python3 build/peerhint_harness.py --executable "$binary" --output _out/peerhint-check
        ;;
    preparation)
        python3 build/preparation_harness.py --executable "$binary" --output _out/preparation-check
        python3 build/webseed_harness.py --executable "$binary" --output _out/webseed-check
        ;;
    adaptive)
        python3 build/adaptive_harness.py --executable "$binary" --output _out/adaptive-check
        python3 build/integrity_harness.py --executable "$binary" --output _out/integrity-check
        ;;
    requests)
        python3 build/request_capacity_check.py --executable "$binary" --output _out/request-capacity
        python3 build/request_cancel_check.py --executable "$binary" --output _out/request-cancel
        python3 build/partial_block_recovery_check.py --executable "$binary" --output _out/partial-block-recovery
        ;;
    sparse)
        python3 build/fixtures.py --output _out/sparse-fixtures
        python3 build/sparse_harness.py --executable "$binary" --fixtures _out/sparse-fixtures --output _out/sparse-check
        python3 build/sparse_harness.py --executable "$binary" --fixtures _out/sparse-fixtures --output _out/sparse-rate-check --rate-aware --cases below complementary late-have
        ;;
    maintenance)
        python3 build/maintenance_harness.py --executable "$binary" --output _out/maintenance-check
        python3 build/upstream_tls_check.py --executable "$binary" --output _out/upstream-tls-check
        python3 build/console_harness.py --executable "$binary" --output _out/console-check
        ;;
    *)
        echo 'Expected integration group: playback, startup, discovery, preparation, adaptive, requests, sparse, maintenance' >&2
        exit 2
        ;;
esac
