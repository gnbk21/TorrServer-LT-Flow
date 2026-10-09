#!/usr/bin/env python3
"""Interleaved native PGO comparison; no automatic adoption from synthetic data."""
import argparse
import hashlib
import json
from pathlib import Path
import statistics
from playback_harness import run_case


def compare(args):
    args.output.mkdir(parents=True,exist_ok=False)
    fixtures=[args.fixtures/name for name in ("head.mp4","tail.mp4","seekable.mkv","variable.mp4")]
    runs=[]
    for iteration in range(3):
        order=[("baseline",args.baseline),("pgo",args.candidate)]
        if iteration%2: order.reverse()
        for name,exe in order:
            result=run_case(exe,args.output,fixtures,f"{name}-{iteration}",4*1024*1024,1,0,0)
            timings=[r["ttfb_ms"] for r in result["ranges"] if not r.get("cancelled")]
            runs.append({"build":name,"iteration":iteration,"ready_ms":result["startup_ready_ms"],"ttfb_median_ms":statistics.median(timings),"ttfb_p95_ms":sorted(timings)[int((len(timings)-1)*.95)],"passed":result["passed"]})
    medians={name:statistics.median(r["ttfb_median_ms"] for r in runs if r["build"]==name) for name in ("baseline","pgo")}
    report={"runs":runs,"median_ttfb_ms":medians,"executable_sha256":{name:hashlib.sha256(path.read_bytes()).hexdigest() for name,path in (("baseline",args.baseline),("pgo",args.candidate))},"rate_limit_bytes_per_second":4*1024*1024,"block_delay_ms":1,"synthetic_only":True,"adopted":False,"decision":"Retain PGO off; real-media CPU/latency/resource acceptance is required before shipping a profile."}
    (args.output/"report.json").write_text(json.dumps(report,indent=2)+"\n",encoding="utf-8")
    print(json.dumps(report))


if __name__=="__main__":
    p=argparse.ArgumentParser()
    for name in ("baseline","candidate","fixtures","output"):p.add_argument("--"+name,type=Path,required=True)
    compare(p.parse_args())
