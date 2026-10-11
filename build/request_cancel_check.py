#!/usr/bin/env python3
"""Real-engine cold A/B/A seek regression for non-FAST late disk completions."""
import argparse
import hashlib
import http.client
import json
from pathlib import Path
import struct
import time

from controlled_peer import LocalSwarm, PeerPlan
from playback_harness import OwnedServer, range_read, upload, MIB


def trial(executable, directory, fixture, late):
    directory.mkdir(parents=True, exist_ok=False)
    report = {'late_completion_enabled': late, 'passed': False}
    server = None
    swarm = None
    try:
        server = OwnedServer(executable, directory/'state', extra_settings={
            'EnableDebug': True, 'CacheSize': 64*MIB, 'PreloadCache': 10,
            'Flow': {'Enabled': True, 'RangeTraceEnabled': True,
                     'SwarmProfile': 'legacy', 'CapacityAwareRequests': True,
                     'BootstrapHeadMB': 4, 'ProbeGraceMs': 300,
                     'StartupBufferMinMB': 4, 'StartupBufferMaxMB': 4,
                     'GlobalCacheBudgetMB': 64}})
        server.ready()
        with LocalSwarm([fixture], peers=[PeerPlan(rate=2*MIB, request_queue=64,
                strict_request_queue=True, request_latency_ms=250,
                late_cancel_completion=late)], piece_length=MIB, high_throughput=True) as swarm:
            info = upload(server, swarm)
            index = info['file_stats'][0]['id']
            key = swarm.info_hash.hex()
            server.json(f'/stream/generated?link={key}&index={index}&preload&stat', timeout=90)
            # Cancel real cold readers only after their requests reach the peer.
            # Returning to A then collides with A's old submitted disk reads.
            for piece in (32, 100):
                before = swarm.status()['peers'][0]['requested_pieces'].count(piece)
                connection = http.client.HTTPConnection('127.0.0.1', server.port, timeout=10)
                try:
                    connection.request('GET', f'/play/{key}/{index}?play&stat=controlled',
                                       headers={'Range': f'bytes={piece*MIB}-'})
                    deadline = time.monotonic()+10
                    while swarm.status()['peers'][0]['requested_pieces'].count(piece) <= before:
                        if time.monotonic() > deadline:
                            raise TimeoutError(f'Cold target {piece} never reached the peer')
                        time.sleep(.01)
                finally:
                    connection.close()
                time.sleep(.1)
            report['range'] = range_read(server, key, index, swarm.data,
                                         32*MIB, 32*MIB+65535)
            peer = swarm.status()['peers'][0]
            report['capacity_passed'] = (peer['peak_pending_requests'] <= 64
                and not peer['close_reasons'].get('queue-limit', 0)
                and not peer['capacity_overflows'])
            report['race_exercised'] = peer['late_cancel_completions'] > 0
            report['passed'] = report['capacity_passed'] and (not late or report['race_exercised'])
            if not report['passed']:
                report['error'] = 'Strict capacity violated or late-completion race was not exercised'
    except Exception as error:
        report['error'] = str(error)
    finally:
        if swarm is not None:
            report['swarm'] = swarm.status()
        if server is not None:
            server.close()
        (directory/'report.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    return report


def check(args):
    args.output.mkdir(parents=True, exist_ok=False)
    fixture = args.output/'owned.ts'
    with fixture.open('xb') as output:
        # Each block carries its own offset: equal-looking pieces cannot hide
        # an incorrect Range offset. All bytes are generated and freely owned.
        suffix = bytes(range(256))*64
        for piece in range(128):
            for block in range(64):
                output.write(struct.pack('!II', piece, block)+suffix[8:])
    report = {'executable_sha256': hashlib.sha256(args.executable.read_bytes()).hexdigest(),
              'fixture_sha256': hashlib.sha256(fixture.read_bytes()).hexdigest(), 'trials': []}
    for late in (False, True):
        for count in range(args.repeat):
            result = trial(args.executable, args.output/f'late-{int(late)}-trial-{count}', fixture, late)
            report['trials'].append(result)
            report['passed'] = all(r['passed'] for r in report['trials'])
            (args.output/'report.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
            print(json.dumps({'late': late, 'trial': count, 'passed': result['passed'],
                              'race_exercised': result.get('race_exercised'),
                              'error': result.get('error')}), flush=True)
            if not result['passed']:
                return False
    return True


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--executable', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--repeat', type=int, choices=range(1, 11), default=3)
    raise SystemExit(0 if check(parser.parse_args()) else 1)
