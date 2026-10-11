#!/usr/bin/env python3
"""Interleaved native PGO comparison; no automatic adoption from synthetic data."""
import argparse
import gc
import hashlib
import json
from pathlib import Path
import statistics
from controlled_peer import LocalSwarm, PeerPlan
from high_bitrate_check import run, calibrate, MIB


def compare(args):
    args.output.mkdir(parents=True,exist_ok=False)
    fixture = args.fixture
    rate = int(args.mbps*1_000_000/8)
    with LocalSwarm([fixture],peers=[PeerPlan(rate=2*rate,request_queue=512)],
                    piece_length=4*MIB,high_throughput=True) as swarm:
        capacity = calibrate(swarm)
    del swarm
    gc.collect()
    if capacity < 1.3*args.mbps: raise RuntimeError('Peer capacity cannot sustain a fair PGO comparison')
    with fixture.open('rb') as data: fixture_hash = hashlib.file_digest(data,'sha256').hexdigest()
    report={'runs':[], 'summaries':{}, 'fixture_sha256':fixture_hash,
        'executable_sha256':{name:hashlib.sha256(path.read_bytes()).hexdigest() for name,path in (('baseline',args.baseline),('pgo',args.candidate))},
        'mbps':args.mbps, 'seconds':args.seconds, 'cache_mb':512, 'startup_reserve_mb':32,
        'piece_mb':4, 'rotations':args.rotations, 'peer_capacity_mbps':capacity,
        'acceptance_repeats_met':args.rotations>=5, 'synthetic_only':True, 'adopted':False,
        'decision':'Retain PGO off. This characterizes Go code only; native CPU and physical playback acceptance are separate.'}
    def record():
        (args.output/'report.json').write_text(json.dumps(report,indent=2)+'\n',encoding='utf-8')
    for case in ('healthy','outages','mixed-peers','bursts'):
        for iteration in range(args.rotations):
            order=[('baseline',args.baseline),('pgo',args.candidate)]
            if iteration%2: order.reverse()
            for name,exe in order:
                print(f'PGO {case} / repeat {iteration+1} / {name}',flush=True)
                result=run(exe,args.output/f'{case}-{iteration}-{name}',fixture,'adaptive',case,
                    rate,args.seconds,512,True,startup_mb=32)
                report['runs'].append({'build':name,'iteration':iteration,**result})
                record()
                gc.collect()
    for case in ('healthy','outages','mixed-peers','bursts'):
        for name in ('baseline','pgo'):
            runs=[r for r in report['runs'] if r['case']==case and r['build']==name]
            summary={'passed':sum(r['passed'] for r in runs),'total':len(runs)}
            for key in ('cpu_seconds','peak_observed_rss_bytes','preload_ready_ms','seek_ttfb_p95_ms','seek_ttfb_p99_ms','native_redundant_bytes'):
                values=[r[key] for r in runs if r.get(key) is not None]
                if values: summary[key]={'values':values,'median':statistics.median(values),'minimum':min(values),'maximum':max(values)}
            for key in ('blocked_over_250ms_ms','read_wait_p95_ms','read_wait_p99_ms'):
                values=[r['delivery'][key] for r in runs if r.get('delivery')]
                if values: summary[key]={'values':values,'median':statistics.median(values),'minimum':min(values),'maximum':max(values)}
            report['summaries'][f'{case}/{name}']=summary
    record()
    return all(r['passed'] for r in report['runs'])


if __name__=="__main__":
    p=argparse.ArgumentParser()
    for name in ('baseline','candidate','fixture','output'):p.add_argument('--'+name,type=Path,required=True)
    p.add_argument('--mbps',type=int,choices=(90,120),default=120)
    p.add_argument('--seconds',type=int,choices=(40,60,120),default=60)
    p.add_argument('--rotations',type=int,choices=(5,6),default=5)
    raise SystemExit(0 if compare(p.parse_args()) else 1)
