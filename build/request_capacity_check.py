#!/usr/bin/env python3
"""Actual Flow/BEP10 strict-cap integration; free loopback bytes, no public peers."""
import argparse
import gc
import json
from pathlib import Path
from high_bitrate_check import run, MIB

CASES = [(reqq, latency, 1) for reqq in (32, 64, 250, 512, 2000) for latency in (0, 250)]
# Large pieces also pressure the local 1500-block ceiling.
CASES += [(2000, latency, 16) for latency in (0, 250)]
CASES += [(reqq, 250, 8) for reqq in (32, 64, 512)]


def check(args):
    args.output.mkdir(parents=True, exist_ok=False)
    fixture=args.output/'owned.bin'
    # Twice the cache budget keeps the midpoint seek cold, so delayed urgent
    # request age is actually exercised rather than hoping playback is uncached.
    with fixture.open('xb') as output:
        block=bytes(range(256))*4096
        for _ in range(128): output.write(block)
    results=[]
    cases = list(dict.fromkeys(tuple(map(int, case.split(':'))) for case in args.case)) if args.case else CASES
    for trial, (reqq,latency,piece) in ((trial, case) for trial in range(args.repeat) for case in cases):
        print(f'Strict reqq={reqq}, latency={latency} ms, piece={piece} MiB, trial={trial}', flush=True)
        name = f'cap-{reqq}-{latency}-{piece}' + (f'-trial-{trial}' if args.repeat > 1 else '')
        result=run(args.executable,args.output/name,fixture,
                   'legacy','healthy',(MIB if latency else 4*MIB),10,64,True,
                   reqq=reqq,strict=True,latency_ms=latency,piece_mb=piece,
                   capacity_aware=True,startup_mb=4)
        peers=result.get('swarm',{}).get('peers',[])
        violations=[p for p in peers if p['peak_pending_requests']>min(reqq,1500)
                    or p['close_reasons'].get('queue-limit',0)]
        if not peers or violations:
            result['passed']=False
            result['capacity_error']='Native urgent/ordinary requests exceeded advertised or local cap'
        result['age_observed']=any(p.get('oldest_request_age_ms',-1)>0
            for sample in result['samples'] for p in sample.get('urgent') or []) or any(
                probe.get('oldest_request_age_ms', -1) > 0 for probe in result.get('request_age_probes', []))
        if latency and not result['age_observed']:
            result['passed']=False
            result['age_error']='Delayed urgent requests never exposed their actual outstanding age'
        results.append(result)
        (args.output/'report.json').write_text(json.dumps(results,indent=2)+'\n',encoding='utf-8')
        gc.collect()
        if args.repeat > 1 and not result['passed']:
            return False
    return all(r['passed'] for r in results)


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--executable',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--case', action='append', choices=[':'.join(map(str, case)) for case in CASES],
                        help='Narrow diagnostic case: reqq:latency-ms:piece-MiB; default checks the complete matrix')
    parser.add_argument('--repeat', type=int, choices=range(1, 11), default=1,
                        help='Repeat selected cases with fresh state; first failure retains evidence and stops')
    raise SystemExit(0 if check(parser.parse_args()) else 1)
