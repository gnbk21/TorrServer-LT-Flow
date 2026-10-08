#!/usr/bin/env python3
"""Actual Flow/BEP10 strict-cap integration; free loopback bytes, no public peers."""
import argparse
import gc
import json
from pathlib import Path
from high_bitrate_check import run, MIB


def check(args):
    args.output.mkdir(parents=True, exist_ok=False)
    fixture=args.output/'owned.bin'
    # Large enough to sustain the test and seeks, without a decoder dependency.
    with fixture.open('xb') as output:
        block=bytes(range(256))*4096
        for _ in range(64): output.write(block)
    results=[]
    for reqq in (32,64,250,512,2000):
        for latency in (0,80):
            print(f'Strict reqq={reqq}, pipelined latency={latency} ms', flush=True)
            result=run(args.executable,args.output/f'cap-{reqq}-{latency}',fixture,
                       'legacy','healthy',4*MIB,10,64,True,
                       reqq=reqq,strict=True,latency_ms=latency,piece_mb=1,
                       capacity_aware=True,startup_mb=4)
            peers=result.get('swarm',{}).get('peers',[])
            violations=[p for p in peers if p['peak_pending_requests']>min(reqq,1500)
                        or p['close_reasons'].get('queue-limit',0)]
            if not peers or violations:
                result['passed']=False
                result['capacity_error']='Native urgent/ordinary requests exceeded advertised or local cap'
            result['age_observed']=any(p.get('oldest_request_age_ms',-1)>0
                for sample in result['samples'] for p in sample.get('urgent') or [])
            results.append(result)
            (args.output/'report.json').write_text(json.dumps(results,indent=2)+'\n',encoding='utf-8')
            gc.collect()
    return all(r['passed'] for r in results)


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--executable',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    raise SystemExit(0 if check(parser.parse_args()) else 1)
