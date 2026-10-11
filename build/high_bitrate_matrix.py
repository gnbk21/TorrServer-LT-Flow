#!/usr/bin/env python3
"""Rotated matched comparisons; owns fresh state and never promotes a policy."""
import argparse
import gc
import hashlib
import json
from pathlib import Path
import statistics

from controlled_peer import LocalSwarm, PeerPlan
from high_bitrate_check import calibrate, generate, run, MIB


def compare(args):
    args.output.mkdir(parents=True, exist_ok=False)
    fixture = args.fixture or args.output/'fixture'/'generated.ts'
    if not args.fixture:
        generate(fixture, int(args.seconds*args.mbps/120*1.3)+10)
    rate = int(args.mbps*1_000_000/8)
    with LocalSwarm([fixture], peers=[PeerPlan(rate=2*rate, request_queue=512)],
                    piece_length=args.piece_mb*MIB, high_throughput=True) as swarm:
        capacity = calibrate(swarm)
    del swarm
    gc.collect()
    if capacity < 1.3*args.mbps:
        raise RuntimeError('Peer calibration cannot sustain a fair comparison')
    policies = [('legacy', args.executable, 'legacy', {}),
                ('adaptive', args.executable, 'adaptive', {})]
    if args.experiments:
        policies.append(('experiments', args.executable, 'adaptive',
                 dict(capacity_aware=True, urgent_horizon=True, burst_hints=True)))
    if args.baseline:
        policies.insert(0, ('previous-legacy', args.baseline, 'legacy', {}))
        policies.insert(1, ('previous-adaptive', args.baseline, 'adaptive', {}))
    if args.transport_comparison:
        policies = [(f'adaptive-transport-{size}', args.executable, 'adaptive', dict(transport_kib=size)) for size in (0, 64, 256, 1024)]
    with fixture.open('rb') as source:
        fixture_hash = hashlib.file_digest(source, 'sha256').hexdigest()
    report = dict(peer_capacity_mbps=capacity, fixture_sha256=fixture_hash,
                  mbps=args.mbps, seconds=args.seconds, cache_mb=args.cache_mb,
                  startup_reserve_mb=32, piece_mb=args.piece_mb, disk=args.disk,
                  synthetic_transport_only=True, promoted=False,
                  rotations=args.rotations, matched_stable_baseline=bool(args.baseline),
                  acceptance_repeats_met=args.rotations >= 5,
                  binaries={name:hashlib.sha256(exe.read_bytes()).hexdigest() for name,exe,_,_ in policies},
                  runs=[], summaries={})
    for case in args.cases:
        for repeat in range(args.rotations):
            # Rotate all policies; reverse alternating rounds to reduce fixed
            # ordering effects. The fixture and reserve are identical throughout.
            order = policies[repeat % len(policies):]+policies[:repeat % len(policies)]
            if repeat % 2: order = list(reversed(order))
            for name, exe, profile, options in order:
                print(f'{case} / repeat {repeat+1} / {name}', flush=True)
                result = run(exe, args.output/f'{case}-{repeat}-{name}', fixture,
                    profile, case, rate, args.seconds, args.cache_mb, True,
                    disk=args.disk, piece_mb=args.piece_mb, startup_mb=32, **options)
                report['runs'].append(dict(policy=name, repeat=repeat, **result))
                write_report(args.output, report)
                gc.collect()
    for case in args.cases:
        for name, *_ in policies:
            results = [r for r in report['runs'] if r['case']==case and r['policy']==name]
            summary = dict(passed=sum(r['passed'] for r in results), total=len(results))
            metrics = {key:[r['delivery'][key] for r in results if r.get('delivery')]
                       for key in ('ttfb_ms','total_read_wait_ms','blocked_over_250ms_ms',
                                   'read_wait_p95_ms','read_wait_p99_ms','read_wait_max_ms','behind_schedule_ms')}
            metrics.update({key:[r[key] for r in results if r.get(key) is not None]
                            for key in ('preload_ready_ms','cpu_seconds','peak_observed_rss_bytes','peak_observed_cache_bytes','seek_ttfb_p95_ms','seek_ttfb_p99_ms','native_redundant_bytes')})
            for key, values in metrics.items():
                if values:
                    summary[key] = dict(values=values, median=statistics.median(values),
                                        minimum=min(values), maximum=max(values),
                                        p95=sorted(values)[min(len(values)-1,int(len(values)*.95))],
                                        p99=sorted(values)[min(len(values)-1,int(len(values)*.99))],
                                        samples=len(values))
            report['summaries'][f'{case}/{name}'] = summary
    write_report(args.output, report)
    return all(r['passed'] for r in report['runs'])


def write_report(directory, report):
    (directory/'report.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')


if __name__ == '__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--executable', type=Path, required=True)
    parser.add_argument('--baseline', type=Path)
    parser.add_argument('--fixture', type=Path)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--mbps', type=int, choices=(90,120), required=True)
    parser.add_argument('--seconds', type=int, default=120)
    parser.add_argument('--rotations', type=int, default=5)
    parser.add_argument('--experiments', action='store_true', help='Separate opt-in experimental policy group')
    parser.add_argument('--transport-comparison', action='store_true', help='Compare direct/64/256/1024 KiB candidate transports separately')
    parser.add_argument('--cache-mb', type=int, choices=(512,2048), default=512)
    parser.add_argument('--piece-mb', type=int, choices=(1,4,16), default=4)
    parser.add_argument('--disk', action='store_true')
    parser.add_argument('--cases', nargs='+', choices=('healthy','outages','mixed-peers','bursts'),
                        default=['healthy','outages','mixed-peers','bursts'])
    args=parser.parse_args()
    if not 30 <= args.seconds <= 120 or not 2 <= args.rotations <= 6:
        parser.error('seconds 30..120 and rotations 2..6 required')
    raise SystemExit(0 if compare(args) else 1)
