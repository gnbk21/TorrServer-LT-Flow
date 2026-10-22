#!/usr/bin/env python3
"""Verify retained diagnostics with an owned server and generated loopback media.

No user's state, server, public torrent or player participates. HTTP byte delivery
is checked; this does not measure decoded playback or physical OS shutdown.
"""
import argparse
import hashlib
import json
from pathlib import Path
import time

from analyze_flow_history import summarize
from controlled_peer import LocalSwarm
from playback_harness import OwnedServer, upload


SETTINGS = {
    "DisableUTP": True,
    "Flow": {"Enabled": True, "DiagnosticHistory": True, "RangeTraceEnabled": False,
             "SwarmProfile": "legacy", "BootstrapHeadMB": 1, "ProbeGraceMs": 0,
             "StartupBufferMinMB": 1, "StartupBufferMaxMB": 8},
}


def records(state):
    path = state / "flow-history.jsonl"
    events = []
    for candidate in [Path(str(path) + f".{i}") for i in (3, 2, 1)] + [path]:
        if candidate.exists():
            data = candidate.read_bytes()
            if len(data) > 1 << 20:
                raise AssertionError("History exceeded its file size limit")
            for line in data.splitlines():
                if len(line) + 1 > 4096:
                    raise AssertionError("History exceeded its record size limit")
                events.append(json.loads(line))
    return events


def run(executable, fixtures, output):
    output.mkdir(parents=True, exist_ok=False)
    report = {"passed": False, "http_delivery_only": True,
              "executable_sha256": hashlib.sha256(executable.read_bytes()).hexdigest()}
    server = OwnedServer(executable, output / "playback", extra_settings=SETTINGS)
    try:
        server.ready()
        with server.request("/echo") as response:
            report["version"] = response.read().decode()
        source = fixtures / "variable.mp4"
        with LocalSwarm([source], rate=256 * 1024) as swarm:
            upload(server, swarm)
            # This response stays active across multiple ten-second samples.
            # No diagnostics drawer/status poller is opened to drive collection.
            path = f"/play/{swarm.info_hash.hex()}/1?play"
            started, received = time.monotonic(), bytearray()
            with server.request(path, headers={"Range": "bytes=0-"}, timeout=35) as response:
                if response.status != 206:
                    raise AssertionError("Expected real partial-content delivery")
                while time.monotonic() - started < 24:
                    part = response.read(65536)
                    if not part:
                        raise AssertionError("Fixture ended before periodic sampling")
                    received.extend(part)
            if not received or received != source.read_bytes()[:len(received)]:
                raise AssertionError("Delivered media bytes changed")
            # A complete cached response verifies completion independently of
            # the intentionally interrupted long response above.
            with server.request(path, headers={"Range": "bytes=0-65535"}, timeout=10) as response:
                if response.status != 206 or response.read() != received[:65536]:
                    raise AssertionError("Completed range outcome was corrupted")
            time.sleep(1)
            report["history_status"] = server.json("/flow/network")["diagnostic_history"]
            if report["history_status"]["errors"] or report["history_status"]["dropped"]:
                raise AssertionError("History lost evidence during the controlled run")
            report["verified_bytes"] = len(received)
            report["active_response_ms"] = round((time.monotonic() - started) * 1000)
            identity = swarm.info_hash.hex()
    finally:
        server.close()

    events = records(server.state)
    snapshots = [e["playback"] for e in events if e["type"] == "playback_sample"]
    outcomes = [e["response"] for e in events if e["type"] == "response"]
    if len(snapshots) < 2 or not any(s["active_readers"] > 0 and s["delivered_offset_bytes"] > 0 for s in snapshots):
        raise AssertionError("Periodic delivery evidence requires a browser or is missing")
    if not any(r["cancelled"] and r["bytes_served"] > 0 for r in outcomes):
        raise AssertionError("Interrupted response evidence missing")
    if not any(r["status"] == 206 and not r["cancelled"] and r["bytes_served"] == 65536 for r in outcomes):
        raise AssertionError("Completed response evidence missing")
    encoded = json.dumps(events)
    if any(secret in encoded for secret in (identity, "variable.mp4", "127.0.0.1", "Flow-generated-fixtures")):
        raise AssertionError("History retained media/client identity")
    summary = summarize(server.state / "flow-history.jsonl")
    if summary["closure"] != "ENGINE_STOP_RECORDED" or summary["shutdown_reason"] != "API_SHUTDOWN":
        raise AssertionError("Clean API exit evidence missing")
    report["playback_samples"] = len(snapshots)
    report["response_outcomes"] = len(outcomes)
    report["summary"] = summary

    # Only this test's owned process is killed. Copy its bounded retained history
    # into another fresh test state to exercise restart interpretation.
    interrupted = OwnedServer(executable, output / "interrupted", extra_settings=SETTINGS)
    try:
        interrupted.ready()
        time.sleep(.3)
        interrupted.process.kill()
        interrupted.process.wait(5)
    finally:
        interrupted.close()
    old = records(interrupted.state)
    if not any(e["type"] == "engine_started" for e in old) or any(e["type"] == "engine_stopped" for e in old):
        raise AssertionError("Forced-stop fixture did not preserve incomplete history")
    retained = (interrupted.state / "flow-history.jsonl").read_bytes()
    restarted = OwnedServer(executable, output / "restart", extra_settings=SETTINGS,
                            seed_files={"flow-history.jsonl": retained})
    try:
        restarted.ready()
    finally:
        restarted.close()
    restarted_events = records(restarted.state)
    if not any(e["type"] == "previous_run" and e.get("stage") == "UNCONFIRMED_TERMINATION" for e in restarted_events):
        raise AssertionError("Missing previous closure was not marked as uncertain")
    report["forced_stop_restart_receipt"] = True
    report["restart_summary"] = summarize(restarted.state / "flow-history.jsonl")
    report["passed"] = True
    (output / "report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    return report


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--executable", type=Path, required=True)
    parser.add_argument("--fixtures", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    print(json.dumps(run(args.executable, args.fixtures, args.output), indent=2))
