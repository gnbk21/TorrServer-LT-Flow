#!/usr/bin/env python3
"""Authenticated, isolated native API/restore/doctor regression check."""
import argparse
import json
from pathlib import Path
import subprocess
import urllib.error
from playback_harness import OwnedServer


def expect_status(server, path, status, data=None, headers=None):
    try:
        with server.request(path, data, headers=headers) as response:
            actual = response.status
    except urllib.error.HTTPError as error:
        actual = error.code
    if actual != status:
        raise AssertionError(f"{path}: expected {status}, got {actual}")


def run(executable, output):
    server = OwnedServer(executable, output / "state", auth=True, extra_settings={"JacRedKey":"fixture-secret-key", "JacRedUrl":"https://example.test/private-token"})
    try:
        server.ready()
        expect_status(server,"/flow/support",401,headers={"Authorization":"Basic invalid"})
        support = server.json("/flow/support")
        text = json.dumps(support)
        if "fixture-secret-key" in text or "private-token" in text or "local-fixture-only" in text:
            raise AssertionError("Support report leaked credentials")
        backup = server.json("/flow/backup")
        text = json.dumps(backup)
        if "fixture-secret-key" in text or "private-token" in text:
            raise AssertionError("Portable backup leaked local secrets")
        preview = server.json("/flow/backup/preview",backup)
        expect_status(server,"/flow/maintenance",403,{"enabled":True},headers={"Origin":"https://foreign.example"})
        maintenance = server.json("/flow/maintenance",{"enabled":True})
        expect_status(server,"/torrents",503,{"action":"list"})
        expect_status(server,"/flow/backup/apply",409,{"backup":backup,"digest":preview["digest"]})
        expect_status(server,"/flow/maintenance",409,{"enabled":False,"token":"wrong"})
        server.json("/flow/maintenance",{"enabled":False,"token":maintenance["token"]})
        expect_status(server,"/flow/backup/apply",400,{"backup":backup,"digest":"0"*64})
        restored = server.json("/flow/backup/apply",{"backup":backup,"digest":preview["digest"]},timeout=30)
        server.ready()
        config = server.json("/settings",{"action":"get"})
        if config["JacRedKey"] != "fixture-secret-key":
            raise AssertionError("Restore replaced local credentials")
        recovery = server.state / "flow-backups" / restored["recovery_backup"]
        if not recovery.is_file() or "fixture-secret-key" not in recovery.read_text(encoding="utf-8"):
            raise AssertionError("Local protected recovery snapshot missing")
        doctor = subprocess.run([str(server.executable),"--doctor","--httpauth","--path",str(server.state),"--port",str(server.port),"--ip","127.0.0.1"],capture_output=True,timeout=10)
        checks = json.loads(doctor.stdout)
        if doctor.returncode != 1 or not any(c["name"]=="http_listener" and c["status"]=="error" for c in checks):
            raise AssertionError("Doctor missed occupied listener")
        if "fixture-secret-key" in doctor.stdout.decode() or "local-fixture-only" in doctor.stdout.decode():
            raise AssertionError("Doctor leaked credentials")
        result = {"passed":True,"authenticated":True,"maintenance_exclusive":True,"restore_recovery":True,"doctor_busy_listener":True,"support_bytes":len(json.dumps(support))}
        (output / "report.json").write_text(json.dumps(result,indent=2)+"\n",encoding="utf-8")
        print(json.dumps(result))
    finally:
        server.close()


if __name__ == "__main__":
    parser=argparse.ArgumentParser()
    parser.add_argument("--executable",type=Path,required=True)
    parser.add_argument("--output",type=Path,required=True)
    args=parser.parse_args()
    run(args.executable,args.output)
