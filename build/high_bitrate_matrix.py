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

PINNED_BASELINE_HASHES = {
    '447e488ac091ddd0f3dc5947295ff1392b68501925f215a854abd81a2952669f',  # Windows amd64
    'f234d9d6934543480b41f4f11810499dc66bd5d7d8a7540b28416eceba7371c2',  # Linux amd64
}

def policy_order(policies, repeat):
    """Rotate the first policy; reversing an odd two-policy rotation cancels it."""
    offset = repeat % len(policies)
    return policies[offset:]+policies[:offset]

def regression_flags(summaries):
    flags=[]
    for key, candidate in summaries.items():
        case, policy=key.split('/',1)
        baseline=summaries.get(case+'/previous-'+policy)
        if baseline is None:
            continue
        failed = candidate.get('native_failed_bytes')
        if failed and failed['maximum'] > 0:
            flags.append(dict(case=case,policy=policy,metric='native_failed_bytes',
                              candidate_maximum=failed['maximum'],requires_diagnosis=True))
        for metric in ('preload_ready_ms','seek_ttfb_p95_ms','cold_seek_ttfb_p95_ms',
                       'ttfb_ms','read_wait_p95_ms','peak_observed_rss_bytes'):
            if metric not in candidate or metric not in baseline:
                continue
            before,after=baseline[metric]['median'],candidate[metric]['median']
            limit=16*MIB if metric.endswith('_bytes') else 250
            if after>before*1.10 and after-before>limit:
                flags.append(dict(case=case,policy=policy,metric=metric,baseline_median=before,
                                  candidate_median=after,absolute_increase=after-before,
                                  requires_diagnosis=True))
    return flags

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
    selected_profiles = getattr(args, 'profiles', ('legacy', 'adaptive'))
    policies = [policy for policy in policies if policy[2] in selected_profiles]
    if not policies:
        raise ValueError('No matching policy selected')
    with fixture.open('rb') as source:
        fixture_hash = hashlib.file_digest(source, 'sha256').hexdigest()
    baseline_hash=hashlib.sha256(args.baseline.read_bytes()).hexdigest() if args.baseline else None
    report = dict(peer_capacity_mbps=capacity, fixture_sha256=fixture_hash,
                  mbps=args.mbps, seconds=args.seconds, cache_mb=args.cache_mb,
                  startup_reserve_mb=32, piece_mb=args.piece_mb, disk=args.disk,
                  synthetic_transport_only=True, promoted=False,
                  verified_cold_seeks=getattr(args, 'verify_cold_seeks', False),
                  rotations=args.rotations, matched_stable_baseline=baseline_hash in PINNED_BASELINE_HASHES,
                  baseline=dict(sha256=baseline_hash,kind='published-Flow-1.2.0' if baseline_hash in PINNED_BASELINE_HASHES else 'custom-reference'),
                  acceptance_repeats_met=False,
                  expected_trials=len(args.cases)*args.rotations*len(policies),
                  binaries={name:hashlib.sha256(exe.read_bytes()).hexdigest() for name,exe,_,_ in policies},
                  runs=[], summaries={})
    for case in args.cases:
        for repeat in range(args.rotations):
            # The fixture and reserve are identical; first position rotates even
            # when comparing just the default profile across two binaries.
            order = policy_order(policies,repeat)
            for order_index,(name, exe, profile, options) in enumerate(order):
                print(f'{case} / repeat {repeat+1} / {name}', flush=True)
                result = run(exe, args.output/f'{case}-{repeat}-{name}', fixture,
                    profile, case, rate, args.seconds, args.cache_mb, True,
                    disk=args.disk, piece_mb=args.piece_mb, startup_mb=32,
                    verify_cold_seeks=getattr(args, 'verify_cold_seeks', False), **options)
                report['runs'].append(dict(policy=name, repeat=repeat, order_index=order_index, **result))
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
                            for key in ('preload_ready_ms','cpu_seconds','peak_observed_rss_bytes','peak_observed_cache_bytes','seek_ttfb_p95_ms','seek_ttfb_p99_ms','cold_seek_ttfb_p95_ms','cold_seek_ttfb_p99_ms','cold_total_first_byte_wait_ms','cold_first_byte_wait_over_250ms_ms','native_redundant_bytes','native_failed_bytes','cache_observed_evicted_bytes','cache_observed_refilled_bytes')})
            metrics['cold_seek_ttfb_ms'] = [item['ttfb_ms'] for result in results for item in result.get('cold_ranges', []) if item.get('cold_verified')]
            for key, values in metrics.items():
                if values:
                    summary[key] = dict(values=values, median=statistics.median(values),
                                        minimum=min(values), maximum=max(values),
                                        p95=sorted(values)[min(len(values)-1,int(len(values)*.95))],
                                        p99=sorted(values)[min(len(values)-1,int(len(values)*.99))],
                                        samples=len(values))
            report['summaries'][f'{case}/{name}'] = summary
    report['acceptance_repeats_met']=args.rotations>=5 and len(report['runs'])==report['expected_trials']
    report['regression_flags']=regression_flags(report['summaries'])
    report['review_required']=bool(report['regression_flags'])
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
    parser.add_argument('--profiles', nargs='+', choices=('legacy','adaptive'), default=['legacy','adaptive'],
                        help='Matched profiles to measure; defaults retain both existing policies')
    parser.add_argument('--transport-comparison', action='store_true', help='Compare direct/64/256/1024 KiB candidate transports separately')
    parser.add_argument('--verify-cold-seeks', action='store_true', help='Record mixed-seek cache evidence and require four verified uncached recovery reads')
    parser.add_argument('--cache-mb', type=int, choices=(512,2048), default=512)
    parser.add_argument('--piece-mb', type=int, choices=(1,4,8,16), default=4)
    parser.add_argument('--disk', action='store_true')
    parser.add_argument('--cases', nargs='+', choices=('healthy','outages','mixed-peers','bursts','stragglers'),
                        default=['healthy','outages','mixed-peers','bursts'])
    args=parser.parse_args()
    if not 30 <= args.seconds <= 120 or not 2 <= args.rotations <= 6:
        parser.error('seconds 30..120 and rotations 2..6 required')
    raise SystemExit(0 if compare(args) else 1)
