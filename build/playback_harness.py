#!/usr/bin/env python3
"""Exercise a real Flow executable with a controlled local swarm.

Owns a fresh state directory, loopback port and child process. It never attaches
to or restarts an existing server. Reports observations without treating HTTP
delivery as a measurement of Android decoder/player latency.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import http.client
import json
import os
from pathlib import Path
import socket
import subprocess
import time
import urllib.error
import urllib.request

from controlled_peer import LocalSwarm
from fixtures import generate

MIB = 1024 * 1024


class OwnedServer:
    def __init__(self, executable, state):
        self.executable, self.state = executable.resolve(), state.resolve()
        state.mkdir(parents=True, exist_ok=False)
        with socket.socket() as reservation:
            reservation.bind(("127.0.0.1", 0))
            self.port = reservation.getsockname()[1]
        self.base = f"http://127.0.0.1:{self.port}"
        config = {"CacheSize": 32*MIB, "ReaderReadAHead": 70, "PreloadCache": 25,
                  "ConnectionsLimit": 12, "DHTConnectionsLimit": 100, "PeersListenPort": 0,
                  "DisableDHT": True, "DisablePEX": True, "DisableUPNP": True,
                  "EnableLPD": False, "EnableBonjour": False, "EnableDLNA": False,
                  "RetrackersMode": 0, "DefaultTrackers": "", "StoreSettingsInJson": True,
                  "TorrentDisconnectTimeout": 30,
                  "Flow": {"Enabled": True, "BootstrapHeadMB": 1, "ProbeGraceMs": 300,
                           "StartupBufferSeconds": 1, "StartupBufferMinMB": 1, "StartupBufferMaxMB": 8,
                           "WarmSessionTimeoutSec": 30, "RangeTraceEnabled": True,
                           "SwarmProfile": "custom", "SwarmCustom": {"MinReconnectTime": 1, "PeerConnectTimeout": 5}}}
        (state / "settings.json").write_text(json.dumps({"BitTorr": config}), encoding="utf-8")
        self.log = (state / "server-output.log").open("wb")
        flags = subprocess.CREATE_NO_WINDOW if os.name == "nt" else 0
        self.process = subprocess.Popen([str(self.executable), "--path", str(self.state), "--port", str(self.port), "--ip", "127.0.0.1"],
                                        stdout=self.log, stderr=subprocess.STDOUT, creationflags=flags)
        self.started = time.monotonic()

    def request(self, path, data=None, headers=None, method=None, timeout=120):
        if data is not None and not isinstance(data, bytes):
            data = json.dumps(data).encode()
            headers = dict(headers or {}, **{"Content-Type": "application/json"})
        req = urllib.request.Request(self.base + path, data=data, headers=headers or {}, method=method)
        return urllib.request.urlopen(req, timeout=timeout)

    def json(self, path, data=None, timeout=10):
        with self.request(path, data, timeout=timeout) as response:
            return json.load(response)

    def ready(self):
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            if self.process.poll() is not None:
                raise RuntimeError(f"Owned server exited: {self.state / 'server-output.log'}")
            try:
                if self.json("/flow/tray", timeout=1)["server_state"] == "RUNNING":
                    return (time.monotonic() - self.started)*1000
            except (OSError, ValueError):
                pass
            time.sleep(.1)
        raise TimeoutError("Owned server startup timeout")

    def close(self):
        if self.process.poll() is None:
            try:
                with self.request("/shutdown", timeout=2):
                    pass
            except OSError:
                pass
            try:
                self.process.wait(30)
            except subprocess.TimeoutExpired:
                self.process.terminate()
                try:
                    self.process.wait(5)
                except subprocess.TimeoutExpired:
                    self.process.kill()
                    self.process.wait(5)
        self.log.close()


def upload(server, swarm):
    boundary = "flow-controlled-test-boundary"
    body = (f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"generated.torrent\"\r\nContent-Type: application/x-bittorrent\r\n\r\n".encode()
            + swarm.torrent() + f"\r\n--{boundary}\r\nContent-Disposition: form-data; name=\"save\"\r\n\r\ntrue\r\n--{boundary}--\r\n".encode())
    with server.request("/torrent/upload", body, {"Content-Type": f"multipart/form-data; boundary={boundary}"}) as response:
        result = json.load(response)
    if not result:
        raise RuntimeError("Generated torrent upload failed")
    return result[0] if isinstance(result, list) else result


def range_read(server, info_hash, index, source, start, end, cancel=False):
    connection = http.client.HTTPConnection("127.0.0.1", server.port, timeout=120)
    started = time.monotonic()
    try:
        connection.request("GET", f"/play/{info_hash}/{index}?play&stat=controlled", headers={"Range": f"bytes={start}-{end}"})
        response = connection.getresponse()
        first = response.read(1)
        ttfb = (time.monotonic()-started)*1000
        if response.status != 206 or first != source[start:start+1]:
            raise AssertionError(f"Range status/content mismatch: {response.status}, file {index}")
        if cancel:
            return {"file_index": index, "start": start, "cancelled": True, "ttfb_ms": ttfb}
        data = first + response.read()
        expected = source[start:end+1]
        if data != expected or response.getheader("Content-Range") != f"bytes {start}-{end}/{len(source)}":
            raise AssertionError("Range body/header mismatch")
        return {"file_index": index, "start": start, "bytes": len(data), "ttfb_ms": ttfb,
                "duration_ms": (time.monotonic()-started)*1000, "sha256": hashlib.sha256(data).hexdigest()}
    finally:
        connection.close()


def run_case(executable, directory, fixtures, label, rate, delay, disconnect, duration):
    server = OwnedServer(executable, directory / label)
    report = {"case": label, "http_delivery_only": True, "ranges": [], "samples": []}
    try:
        report["startup_ready_ms"] = server.ready()
        with server.request("/echo") as response:
            report["version"] = response.read().decode()
        with LocalSwarm(fixtures, rate, delay, disconnect) as swarm:
            status = upload(server, swarm)
            info_hash = swarm.info_hash.hex()
            if status["hash"] != info_hash:
                raise AssertionError("Torrent identity mismatch")
            for file in status["file_stats"]:
                source = next(p.read_bytes() for p in fixtures if file["path"].endswith(p.name))
                index = file["id"]
                # Full head, EOF index, and parallel overlapping Range requests.
                report["ranges"].append(range_read(server, info_hash, index, source, 0, min(len(source)-1,3*MIB)))
                report["ranges"].append(range_read(server, info_hash, index, source, max(0,len(source)-65536),len(source)-1))
                with ThreadPoolExecutor(max_workers=2) as pool:
                    tasks = [pool.submit(range_read, server, info_hash, index, source, start, min(len(source)-1,start+131071)) for start in (MIB, MIB+65536)]
                    report["ranges"].extend(task.result(timeout=120) for task in tasks)
                report["ranges"].append(range_read(server, info_hash, index, source, MIB, min(len(source)-1,MIB+2*MIB),cancel=True))
                report["ranges"].append(range_read(server, info_hash, index, source, MIB, min(len(source)-1,MIB+131071)))
                if len(source)>32*MIB:
                    report["ranges"].append(range_read(server, info_hash, index, source, 24*MIB,24*MIB+131071))
                    report["ranges"].append(range_read(server, info_hash, index, source, 3*MIB,3*MIB+131071))
            report["flow"] = server.json("/flow/status/"+info_hash)
            for session in report["flow"].get("sessions") or []:
                if session.get("active_readers", 0) != 0:
                    # Reader closes settle asynchronously after a cancelled socket.
                    time.sleep(.5)
                    report["flow"] = server.json("/flow/status/"+info_hash)
                    break
            deadline = time.monotonic() + duration
            cycle = 0
            last = status["file_stats"][-1]
            source = next(p.read_bytes() for p in fixtures if last["path"].endswith(p.name))
            while time.monotonic()<deadline:
                start = (cycle*65536) % max(1,len(source)-131072)
                range_read(server,info_hash,last["id"],source,start,start+65535)
                report["samples"].append({"elapsed_seconds": time.monotonic()-server.started, "runtime": server.json("/runtime/status")})
                cycle += 1
                time.sleep(5)
            report["cycles"] = cycle
            report["peer"] = swarm.status()
            if report["peer"]["sent_bytes"] <= 0 or report["peer"]["tracker_announces"] <= 0:
                raise AssertionError("The controlled native peer path was not exercised")
            if disconnect and report["peer"]["disconnects"] != 1:
                raise AssertionError("Disconnect scenario was not exercised")
            # This period contains no playback calls; observation must not keep
            # the warm reserve alive. Record actual expiry, not an assumed timer.
            if label == "fast":
                expiry_started = time.monotonic()
                deadline = expiry_started+45
                while time.monotonic()<deadline:
                    allocation = server.json("/runtime/status").get("cache_allocation")
                    if allocation is not None and allocation.get("warm_caches",0)==0:
                        report["warm_expiry_ms"] = (time.monotonic()-expiry_started)*1000
                        break
                    time.sleep(1)
                if allocation is not None and "warm_expiry_ms" not in report:
                    raise AssertionError("Warm cache did not expire")
            report["passed"] = True
    finally:
        server.close()
    return report


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--executable", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--fixtures", type=Path)
    parser.add_argument("--duration", type=int, default=0, help="additional resource cycling seconds per case; e.g. 7200 for endurance")
    parser.add_argument("--cases", nargs="+", choices=("fast","slow","disconnect"), default=["fast","slow","disconnect"])
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=False)
    paths = [args.fixtures / name for name in ("head.mp4","tail.mp4","seekable.mkv","variable.mp4")] if args.fixtures else generate(args.output / "fixtures")
    cases = {"fast": (0,0,0), "slow": (2*MIB,2,0), "disconnect": (4*MIB,1,100)}
    result = {"schema_version":1,"executable_sha256":hashlib.sha256(args.executable.read_bytes()).hexdigest(),"cases":[]}
    try:
        for name in args.cases:
            print(f"Controlled case: {name}", flush=True)
            result["cases"].append(run_case(args.executable,args.output,paths,name,*cases[name],args.duration))
            (args.output / "report.json").write_text(json.dumps(result,indent=2)+"\n",encoding="utf-8")
        print(json.dumps({"passed":True,"report":str(args.output / "report.json"),"cases":len(result["cases"])}))
    except Exception as error:
        result["error"] = str(error)
        (args.output / "report.json").write_text(json.dumps(result,indent=2)+"\n",encoding="utf-8")
        raise
