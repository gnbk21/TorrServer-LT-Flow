#!/usr/bin/env python3
"""Exercise two-phase startup with 16-MiB pieces and a nonaligned episode.

Uses only generated media, an owned process/state directory and loopback peers.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
from pathlib import Path
import time
import urllib.error
from urllib.parse import urlencode

from controlled_peer import LocalSwarm, PeerPlan
from playback_harness import OwnedServer, upload

MIB = 1 << 20


def run(executable, fixtures, output, case, assert_fixed):
    output.mkdir(parents=True, exist_ok=False)
    prefix, suffix = output / "000-padding.bin", output / "zzz-padding.bin"
    prefix.write_bytes(bytes(MIB))
    suffix.write_bytes(bytes(12 * MIB))
    media = fixtures / "variable.mp4"
    if media.stat().st_size < 64 * MIB:
        raise ValueError("Use a generated variable.mp4 of at least 64 MiB (build/fixtures.py --seconds 24)")
    settings = {"DisableUTP": True, "Flow": {
        "Enabled": True, "SwarmProfile": "adaptive", "BootstrapHeadMB": 1,
        "ProbeGraceMs": 0, "StartupBufferMinMB": 32, "StartupBufferMaxMB": 32,
        "DiagnosticHistory": True, "RangeTraceEnabled": False,
        "ScarcePieceHints": True, "RateAwareDeadlines": True,
        "CapacityAwareRequests": True, "AdaptiveUrgentHorizon": True,
        "ContainerBurstHints": True}}
    rate = 320 * 1024 if case == "progress" else 2 * MIB
    plan = PeerPlan(rate=rate, choke_until=1,
                    pieces={0, 1} if case == "missing-index" else None)
    server = OwnedServer(executable, output / "state", extra_settings=settings)
    report = {"case": case, "passed": False, "assert_fixed": assert_fixed,
              "executable_sha256": hashlib.sha256(executable.read_bytes()).hexdigest()}
    try:
        server.ready()
        with LocalSwarm([prefix, media, suffix], piece_length=16 * MIB, peers=[plan]) as swarm:
            upload(server, swarm)
            status = server.json("/torrents", {"action": "get", "hash": swarm.info_hash.hex()})
            index = next(f["id"] for f in status["file_stats"] if f["path"].endswith("variable.mp4"))
            path = "/stream/generated?" + urlencode({"link": swarm.info_hash.hex(), "index": index, "preload": "", "stat": ""})
            with ThreadPoolExecutor(max_workers=1) as pool:
                def preload():
                    with server.request(path, timeout=330) as response:
                        return json.load(response)
                started = time.monotonic()
                future = pool.submit(preload)
                samples = []
                removed = False
                while not future.done():
                    data = server.json("/flow/status/" + swarm.info_hash.hex() + "?traces=false")
                    samples.append(data["startup"])
                    if case == "missing-index" and time.monotonic() - started >= 12:
                        server.json("/torrents", {"action": "rem", "hash": swarm.info_hash.hex()})
                        removed = True
                        break
                    if time.monotonic() - started > 315:
                        raise TimeoutError("Startup exceeded test limit")
                    time.sleep(.2)
                try:
                    future.result(timeout=10)
                except urllib.error.HTTPError as error:
                    if not removed or error.code not in (404, 500):
                        raise
            elapsed = time.monotonic() - started
            report["elapsed_seconds"] = elapsed
            if case != "missing-index":
                data = server.json("/flow/status/" + swarm.info_hash.hex() + "?traces=false")
                samples.append(data["startup"])
            report["startup"] = samples[-1]
            stats = swarm.status()["peers"][0]
            report["first_piece_requests_seconds"] = stats["first_request_seconds"]
            report["sent_bytes"] = stats["sent_bytes"]
            report["history_status"] = server.json("/flow/network")["diagnostic_history"]
            premature = any(s.get("bootstrap_complete_ms", 0) == 0 and
                            1 in stats["first_request_seconds"] and
                            stats["first_request_seconds"][1] < s.get("elapsed_ms", 0) / 1000
                            for s in samples)
            report["phase2_requested_before_bootstrap"] = premature
            if case == "ordering" and assert_fixed and premature:
                raise AssertionError("Phase-2 head was requested before bootstrap completed")
            if case in ("ordering", "progress") and assert_fixed and report["startup"].get("state") != "READY":
                raise AssertionError("Useful progressing startup did not complete")
            if case == "progress" and assert_fixed and (elapsed <= 120 or not report["startup"].get("wait_extended") or report["startup"].get("wait_budget_ms", 0) > 300000):
                raise AssertionError("Progress-renewed startup budget was not exercised safely")
            if case == "missing-index":
                if any(s.get("state") == "READY" for s in samples):
                    raise AssertionError("Missing index was bypassed")
                evidence = [s.get("progress", {}).get("tail_supply") for s in samples]
                report["missing_index_supply_observed"] = any(e and e.get("suppliers") == 0 for e in evidence)
                if assert_fixed and not report["missing_index_supply_observed"]:
                    raise AssertionError("No-supplier index evidence missing")
            elif report["startup"].get("state") == "READY":
                play = "/play/" + swarm.info_hash.hex() + "/" + str(index) + "?play"
                with server.request(play, headers={"Range": "bytes=0-65535"}, timeout=10) as response:
                    if response.status != 206 or response.read() != media.read_bytes()[:65536]:
                        raise AssertionError("Media range content changed")
            if report["history_status"]["errors"] or report["history_status"]["dropped"]:
                raise AssertionError("History recorder lost evidence")
            report["passed"] = True
    finally:
        server.close()
        (output / "report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    return report


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--executable", type=Path, required=True)
    parser.add_argument("--fixtures", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--case", choices=("ordering", "progress", "missing-index"), required=True)
    parser.add_argument("--assert-fixed", action="store_true")
    args = parser.parse_args()
    print(json.dumps(run(args.executable, args.fixtures, args.output, args.case, args.assert_fixed), indent=2))
