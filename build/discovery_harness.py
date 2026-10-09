#!/usr/bin/env python3
"""Bounded-state recovery against a real executable in disposable state."""
import argparse
import json
from pathlib import Path
from playback_harness import OwnedServer


def run(executable, output):
    output.mkdir(parents=True, exist_ok=False)
    report = []
    native = b"d9:dht stated5:nodesl6:" + bytes([127,0,0,1,0x1a,0xe1]) + b"eee"
    for name, data, disabled, expected in [
        ("corrupt", b"not-bencode", False, False),
        ("oversized", bytes(1048577), False, False),
        ("restored", native, False, True),
        ("disabled", native, True, False),
    ]:
        server = OwnedServer(executable, output/name, extra_settings={
            "DisableDHT":disabled, "Flow":{"Enabled":True,"DHTStatePersistence":True,"DiagnosticHistory":True}},
            seed_files={"flow-dht.bin":data})
        try:
            server.ready()
            network = server.json("/flow/network")
            if network["dht_state_restored"] != expected:
                raise AssertionError(f"{name}: incorrect restoration outcome")
            report.append({"case":name,"restored":network["dht_state_restored"],"local_api_ready":True})
        finally:
            server.close()
        if name == "disabled" and (output/name/"flow-dht.bin").read_bytes() != data:
            raise AssertionError("Disabled DHT modified routing state")
    (output/"report.json").write_text(json.dumps(report,indent=2)+"\n",encoding="utf-8")
    return report

if __name__ == "__main__":
    p=argparse.ArgumentParser()
    p.add_argument("--executable",type=Path,required=True)
    p.add_argument("--output",type=Path,required=True)
    a=p.parse_args()
    print(json.dumps(run(a.executable,a.output),indent=2))
