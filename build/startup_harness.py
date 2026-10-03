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
                 "DiagnosticHistory": True, "SwarmProfile": profile, "SwarmCustom": {"MinReconnectTime": 30}}})
    report = {"scenario": "peer discovered before explicit preload", "profile": profile,
              "executable_sha256": hashlib.sha256(executable.read_bytes()).hexdigest(), "passed": False}
    try:
        server.ready()
        with server.request("/echo") as response:
            report["version"] = response.read().decode()
        with LocalSwarm([fixtures / "head.mp4", fixtures / "tail.mp4"]) as swarm:
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
            # endpoint returns at readiness while its owned background worker
            # retains the bounded warm handoff.
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
            if assert_fast and report["preload_response_ms"] > report["buffer_ready_ms"] + 1000:
                raise AssertionError("Preload response still waits for the warm grace")
            if report["startup"].get("first_useful_block_ms", -1) < 0:
                raise AssertionError("First useful media data was not observed")
            # Same-file requests coalesce with the already-ready warm owner.
            with ThreadPoolExecutor(max_workers=4) as pool:
                began = time.monotonic()
                responses = list(pool.map(lambda _: server.json(path), range(4)))
                report["concurrent_preload_ms"] = (time.monotonic()-began)*1000
                if any(r.get("preloaded_bytes",0) < r.get("preload_size",1) for r in responses):
                    raise AssertionError("Concurrent preload lost its buffer")
                if report["concurrent_preload_ms"] > 1000:
                    raise AssertionError("Concurrent preload waited for background handoff")
            # A new episode supersedes and joins the old scheduling owner.
            switch_path = "/stream/generated?" + urlencode({"link":swarm.info_hash.hex(),"index":2,"preload":"","stat":""})
            switched = server.json(switch_path, timeout=10)
            if switched.get("preloaded_bytes",0) < switched.get("preload_size",1):
                raise AssertionError("Episode switch did not complete its buffer")
            play_path = "/stream/generated?" + urlencode({"link":swarm.info_hash.hex(),"index":2,"play":""})
            with server.request(play_path,headers={"Range":"bytes=0-65535"},timeout=10) as response:
                if response.status != 206 or response.read() != (fixtures/"tail.mp4").read_bytes()[:65536]:
                    raise AssertionError("Warm handoff corrupted a live reader")
            history_status = server.json("/flow/network")["diagnostic_history"]
            if not history_status["enabled"] or history_status["errors"]:
                raise AssertionError("Diagnostic recorder failed during playback")
            report["history"] = history_status
            # Remove while a fresh explicit preload is still holding its warm
            # grace. Removal must join that owner without waiting eight seconds.
            server.json(path, timeout=10)
            removing = time.monotonic()
            with server.request("/torrents", {"action":"rem", "hash":swarm.info_hash.hex()}) as response:
                if response.status != 200:
                    raise AssertionError("Torrent removal failed")
                response.read()
            report["remove_during_handoff_ms"] = (time.monotonic()-removing)*1000
            if report["remove_during_handoff_ms"] > 2000:
                raise AssertionError("Removal waited for the warm grace")
            report["passed"] = True
    finally:
        server.close()
        history = output / "state" / "flow-history.jsonl"
        if report["passed"]:
            records = [json.loads(line) for line in history.read_text(encoding="utf-8").splitlines()]
            if not any(r["type"] == "engine_stopped" for r in records):
                raise AssertionError("Shutdown did not drain history")
            if "generated" in history.read_text(encoding="utf-8") or swarm.info_hash.hex() in history.read_text(encoding="utf-8"):
                raise AssertionError("History retained media identity")
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
