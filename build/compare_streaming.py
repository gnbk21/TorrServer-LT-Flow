"""Repeated byte-checked baseline/candidate observations with identical settings.

Uses only the existing controlled peer and delivery harness. Every attempt,
including failures, is retained. These are server HTTP observations, not phone
decoding or public-swarm acceptance, and never qualify a toolchain promotion.
"""
import argparse
import gc
import hashlib
import json
from pathlib import Path
import statistics

from controlled_peer import LocalSwarm, PeerPlan
from high_bitrate_check import MIB, calibrate, run


def summary(rows):
    out = {}
    for name in ('baseline', 'candidate'):
        selected = [r for r in rows if r['build'] == name]
        metrics = {}
        for key in ('peak_observed_rss_bytes', 'cpu_seconds', 'seek_ttfb_p95_ms', 'cold_seek_ttfb_p95_ms', 'preload_ready_ms'):
            values = [r[key] for r in selected if r.get(key) is not None]
            if values:
                metrics[key] = {'values': values, 'median': statistics.median(values)}
        for key in ('blocked_over_250ms_ms', 'read_wait_max_ms', 'behind_schedule_ms'):
            values = [r['delivery'][key] for r in selected if key in (r.get('delivery') or {})]
            if values:
                metrics[key] = {'values': values, 'median': statistics.median(values)}
        out[name] = {'attempts': len(selected), 'passed': sum(bool(r.get('passed')) for r in selected),
                     'allocation_failures': sum(r.get('failure_stage') == 'fixture_port_allocation' for r in selected), 'metrics': metrics}
    return out


def compare(args):
    args.output.mkdir(parents=True, exist_ok=False)
    fixture = args.output/'owned.bin'
    with fixture.open('xb') as stream:
        for i in range(192):
            stream.write(hashlib.shake_256(i.to_bytes(4, 'big')).digest(MIB))
    builds = {'baseline': args.baseline.resolve(strict=True), 'candidate': args.candidate.resolve(strict=True)}
    report = {'schema_version': 1, 'passed': False, 'runs': [], 'calibrations_mbps': [],
              'configuration': {'profile': 'legacy', 'case': args.case, 'mbps': args.mbps, 'seconds': args.seconds, 'cache_mib': 128, 'piece_mib': 4, 'preload': False},
              'executable_sha256': {k: hashlib.sha256(v.read_bytes()).hexdigest() for k, v in builds.items()},
              'fixture_sha256': hashlib.sha256(fixture.read_bytes()).hexdigest(),
              'scope': 'Controlled HTTP byte delivery, repeated fixed configuration. Not player or public-swarm acceptance.'}
    rate = int(args.mbps*1e6/8)
    try:
        for rotation in range(args.rotations):
            with LocalSwarm([fixture], peers=[PeerPlan(rate=2*rate)], piece_length=4*MIB, high_throughput=True) as swarm:
                supply = calibrate(swarm)
            del swarm
            gc.collect()
            report['calibrations_mbps'].append(supply)
            if supply < 1.3*args.mbps:
                raise RuntimeError('Controlled fixture cannot sustain matched demand')
            for name in (('baseline', 'candidate') if rotation % 2 == 0 else ('candidate', 'baseline')):
                print(f'Rotation {rotation+1}: {name}', flush=True)
                result = run(builds[name], args.output/f'{rotation}-{name}', fixture, 'legacy', args.case, rate, args.seconds, 128)
                result.update(build=name, rotation=rotation)
                report['runs'].append(result)
                gc.collect()
                report['summary'] = summary(report['runs'])
                (args.output/'comparison.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
        report['passed'] = len(report['runs']) == args.rotations*2 and all(r.get('passed') for r in report['runs'])
    except Exception as error:
        report['error'] = str(error)
    finally:
        report['summary'] = summary(report['runs'])
        (args.output/'comparison.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    return report


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--baseline', required=True, type=Path)
    parser.add_argument('--candidate', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--rotations', type=int, choices=range(3, 11), default=3)
    parser.add_argument('--seconds', type=int, choices=range(10, 61), default=12)
    parser.add_argument('--mbps', type=float, default=90)
    parser.add_argument('--case', choices=('healthy', 'stragglers'), default='stragglers', help='Stragglers additionally proves each cold seek required a new peer request')
    args = parser.parse_args()
    if not 1 <= args.mbps <= 500:
        parser.error('--mbps must be between 1 and 500')
    result = compare(args)
    print(json.dumps({'passed': result['passed'], 'report': str(args.output/'comparison.json')}))
    raise SystemExit(0 if result['passed'] else 1)
