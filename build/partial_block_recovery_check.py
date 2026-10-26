#!/usr/bin/env python3
"""Short real-wire regression for a nearly complete block that stops arriving."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
from pathlib import Path
import time

from controlled_peer import LocalSwarm, PeerPlan
from playback_harness import OwnedServer, upload, range_read


def check(executable, output, personal_settings=None, extra_arguments=None):
    output.mkdir(parents=True, exist_ok=False)
    content = hashlib.shake_256(b'Flow partial-block recovery fixture').digest(8*1048576)
    fixture = output/'generated.ts'
    fixture.write_bytes(content)
    fixture_settings = {
        'ForceEncrypt': False, 'EnableRutorSearch': False,
        'Flow': {'Enabled': True, 'MetricsEnabled': True, 'RangeTraceEnabled': True,
                 'SwarmProfile': 'adaptive', 'CapacityAwareRequests': True,
                 'AdaptiveUrgentHorizon': True, 'FrontierRecovery': True,
                 'AdaptivePeerRequests': True, 'ContainerBurstHints': True,
                 'BootstrapHeadMB': 8, 'ProbeGraceMs': 300,
                 'StartupBufferSeconds': 1, 'StartupBufferMinMB': 1,
                 'StartupBufferMaxMB': 8}}
    server = OwnedServer(executable, output/'state', extra_settings=(
        personal_settings if personal_settings is not None else fixture_settings),
        extra_arguments=extra_arguments)
    report = {'synthetic_transport_only': True, 'passed': False}
    try:
        report['ready_ms'] = server.ready()
        with server.request('/echo', timeout=5) as response:
            report['version'] = response.read().decode()
        if personal_settings is not None:
            effective = server.json('/settings', {'action': 'get'})
            def preserved(wanted, actual):
                for key, value in wanted.items():
                    if isinstance(value, dict):
                        preserved(value, actual.get(key, {}))
                    else:
                        assert actual.get(key) == value, 'Personal setting differs: '+key
            preserved(personal_settings, effective)
            report['personal_settings_verified'] = True
            # Only this disposable instance accepts the fixture's plain wire
            # handshake and disables discovery. The package keeps every setting.
            effective.update(ForceEncrypt=False, EnableRutorSearch=False,
                             RetrackersMode=0, DefaultTrackers='', DisableDHT=True,
                             DisablePEX=True, DisableUPNP=True, EnableLPD=False,
                             EnableBonjour=False, EnableDLNA=False)
            with server.request('/settings', {'action': 'set', 'sets': effective,
                                             'when': 'now'}, timeout=15) as response:
                response.read()
            report['fixture_only_settings_overrides'] = True
            server.ready()
        primary = PeerPlan(rate=8*1048576, request_queue=128,
                           strict_request_queue=True, partial_block_pause_ms=6000)
        alternate = PeerPlan(pieces=set(), choke_until=100000, rate=8*1048576,
                             request_queue=128, strict_request_queue=True)
        with LocalSwarm([fixture], peers=[primary, alternate], piece_length=1048576,
                        high_throughput=True) as swarm:
            status = upload(server, swarm)
            info_hash, index = status['hash'], status['file_stats'][0]['id']
            def preload():
                with server.request(f'/stream?link={info_hash}&index={index}&preload', timeout=30) as response:
                    response.read()
            with ThreadPoolExecutor(max_workers=1) as pool:
                loading = pool.submit(preload)
                deadline = time.monotonic()+15
                while True:
                    peer = swarm.status()['peers'][0]
                    if peer.get('partial_pauses'):
                        break
                    if loading.done():
                        loading.result()
                        raise AssertionError('Preload completed without the partial-block fault')
                    if time.monotonic() > deadline:
                        raise TimeoutError('Fixture did not reach its partial-block fault')
                    time.sleep(.01)
                target = peer['partial_pause_piece']
                # Offer just the stalled piece, after its original request is
                # demonstrably on the wire. This proves recovery uses another
                # supplier rather than winning the initial block assignment.
                alternate.have_events = [(0, target)]
                alternate.choke_until = time.monotonic()-swarm.started+.1
                # Ask for the withheld suffix itself. Reading an earlier block
                # can succeed from cache without exercising this recovery.
                start = target*1048576+peer['partial_pause_begin']+15360
                report['range'] = range_read(server, info_hash, index, content, start, start+1023)
                report['range_completed_seconds'] = time.monotonic()-swarm.started
                report['partial_pause_started_seconds'] = peer['partial_pause_started_seconds']
                report['partial_pause_deadline_seconds'] = peer['partial_pause_started_seconds']+6
                report['recovered_before_original_resumed'] = (
                    report['range_completed_seconds'] < report['partial_pause_deadline_seconds']-.25)
                loading.result(timeout=30)
            report['swarm'] = swarm.status()
            report['diagnostics'] = server.json('/flow/status/'+info_hash+'?traces=false')
            assert report['swarm']['peers'][0].get('partial_pauses') == 1
            assert not any(p['capacity_overflows'] for p in report['swarm']['peers'])
            assert report['diagnostics']['integrity']['failures'] == 0
            assert report['recovered_before_original_resumed'], 'Waited for the stalled original block'
            assert report['swarm']['peers'][1]['sent_bytes'] > 0
            report['passed'] = True
    finally:
        server.close()
        report['owned_server_stopped'] = server.process.poll() is not None
        (output/'report.json').write_text(json.dumps(report, indent=2)+'\n')
    return report


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--executable', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    result = check(args.executable, args.output)
    print(json.dumps({key: result[key] for key in (
        'version', 'passed', 'recovered_before_original_resumed', 'range',
        'range_completed_seconds', 'partial_pause_deadline_seconds', 'owned_server_stopped')}))
