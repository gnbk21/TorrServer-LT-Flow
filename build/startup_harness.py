#!/usr/bin/env python3
"""Measure prebuffer startup with a healthy peer already connected.

Uses original generated fixtures, fresh state and a loopback seeder. Does not
attach to the user's server or access a public swarm.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
from pathlib import Path
import time
from urllib.parse import urlencode

from controlled_peer import LocalSwarm
from playback_harness import OwnedServer, upload


def run(executable, fixtures, output, assert_fast=False, profile="legacy"):
    output.mkdir(parents=True, exist_ok=False)
    server = OwnedServer(executable, output / "state", extra_settings={
        "EnableDebug": True, "DisableUTP": True,
        "Flow": {"Enabled": True, "BootstrapHeadMB": 1, "ProbeGraceMs": 0,
                 "StartupBufferMinMB": 1, "StartupBufferMaxMB": 8,
                 "SwarmProfile": profile, "SwarmCustom": {"MinReconnectTime": 30}}})
    report = {"scenario": "peer discovered before explicit preload", "profile": profile,
              "executable_sha256": hashlib.sha256(executable.read_bytes()).hexdigest(), "passed": False}
    try:
        server.ready()
        with server.request("/echo") as response:
            report["version"] = response.read().decode()
        with LocalSwarm([fixtures / "head.mp4"]) as swarm:
            upload(server, swarm)
            deadline = time.monotonic() + 15
            while swarm.status()["connections"] == 0:
                if time.monotonic() >= deadline:
                    raise TimeoutError("Seeder did not connect before preload")
                time.sleep(.05)
            # Allow the incoming handshake/bitfield to finish. No reader or
            # priority-bearing request has been created yet.
            time.sleep(.3)
            report["before"] = swarm.status()
            started = time.monotonic()
            path = "/stream/generated?" + urlencode({"link": swarm.info_hash.hex(), "index": 1, "preload": "", "stat": ""})
            # Lampa polls status after its preload request's read timeout. The
            # endpoint may retain an eight-second warm handoff grace, which is
            # distinct from the point where the buffer becomes playable.
            with ThreadPoolExecutor(max_workers=1) as pool:
                def preload():
                    with server.request(path, timeout=80) as response:
                        return json.load(response)
                future = pool.submit(preload)
                deadline = started + 65
                while True:
                    status = server.json("/torrents", {"action": "get", "hash": swarm.info_hash.hex()})
                    if status.get("preloaded_bytes", 0) >= status.get("preload_size", 1) > 0:
                        break
                    if time.monotonic() >= deadline:
                        raise TimeoutError("Preload buffer did not complete")
                    time.sleep(.05)
                report["buffer_ready_ms"] = (time.monotonic() - started) * 1000
                future.result(timeout=20)
            report["preload_response_ms"] = (time.monotonic() - started) * 1000
            report["after"] = swarm.status()
            report["startup"] = server.json("/flow/status/" + swarm.info_hash.hex())["startup"]
            if status.get("preloaded_bytes", 0) < status.get("preload_size", 1):
                raise AssertionError("Preload returned without completing its buffer")
            if assert_fast and report["buffer_ready_ms"] >= 5000:
                raise AssertionError("Healthy connected peer startup exceeded 5 seconds")
            if assert_fast and report["after"]["connections"] != report["before"]["connections"]:
                raise AssertionError("Preload discarded an already connected peer")
            report["passed"] = True
    finally:
        server.close()
        (output / "report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    return report


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--executable", type=Path, required=True)
    parser.add_argument("--fixtures", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--assert-fast", action="store_true")
    parser.add_argument("--profile", choices=("legacy", "custom"), default="legacy")
    args = parser.parse_args()
    print(json.dumps(run(args.executable, args.fixtures, args.output, args.assert_fast, args.profile), indent=2))
