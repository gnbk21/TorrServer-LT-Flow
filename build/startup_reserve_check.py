#!/usr/bin/env python3
"""Short generated-media preload check: optional reserve versus required index."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import json
import math
from pathlib import Path
import subprocess
import time
import urllib.error

from controlled_peer import LocalSwarm, PeerPlan
from playback_harness import OwnedServer, upload, range_read
from fixture_ports import joint_loopback_port

MIB = 1048576


def check(executable, output, personal_settings=None):
    output.mkdir(parents=True, exist_ok=False)
    media = output/'generated.ts'
    subprocess.run(['ffmpeg', '-hide_banner', '-loglevel', 'error', '-n',
                    '-f', 'lavfi', '-i', 'testsrc2=size=320x180:rate=25', '-t', '12',
                    '-c:v', 'mpeg2video', '-b:v', '8M', '-minrate', '8M', '-maxrate', '8M',
                    '-bufsize', '16M', '-muxrate', '24M', '-f', 'mpegts', str(media)],
                   check=True, timeout=45)
    metadata = json.loads(subprocess.check_output([
        'ffprobe', '-v', 'error', '-show_format', '-of', 'json', str(media)], text=True))
    rate = float(metadata['format']['bit_rate'])/8
    # ffprobe's analysis must fit inside bootstrap. Announce the rest of the
    # ordinary head only after the real probe finishes, leaving enough paced
    # delivery to qualify supply independently of probe/runner timing.
    base = min(32*MIB, max(16*MIB, math.ceil(rate)))
    prefix, suffix = output/'000-padding.bin', output/'zzz-padding.bin'
    prefix.write_bytes(bytes(MIB//2))
    suffix.write_bytes(bytes(MIB))
    head_last = (prefix.stat().st_size+base-1)//MIB
    tail_last = (prefix.stat().st_size+media.stat().st_size-1)//MIB
    tail_first = tail_last-4
    assert tail_first > head_last+12, 'Generated file is too small for disjoint reserve/index evidence'
    flow = {'Enabled': True, 'AdaptiveStartup': True, 'SwarmProfile': 'adaptive',
            'BootstrapHeadMB': 8, 'ProbeGraceMs': 500, 'StartupBufferSeconds': 1,
            'StartupBufferMinMB': 16, 'StartupBufferMaxMB': 32, 'StartupSafetyFactorPct': 100,
            'MetricsEnabled': True, 'RangeTraceEnabled': True, 'CapacityAwareRequests': True,
            'AdaptiveUrgentHorizon': True, 'FrontierRecovery': True,
            'AdaptivePeerRequests': True, 'ContainerBurstHints': True}
    fixture = {'ForceEncrypt': False, 'DisableDHT': True, 'DisablePEX': True,
               'DisableUPNP': True, 'EnableLPD': False, 'EnableBonjour': False,
               'EnableDLNA': False, 'EnableRutorSearch': False, 'RetrackersMode': 0,
               'DefaultTrackers': '', 'Flow': flow}
    report = {'synthetic_transport_only': True, 'passed': False, 'media_rate': rate,
              'base_bytes': base, 'nonaligned_offset': prefix.stat().st_size,
              'port_allocation': {}}
    try:
        peer_port = joint_loopback_port(evidence=report['port_allocation'])
    except (OSError, RuntimeError) as error:
        report.update(failure_stage='fixture_port_allocation',
                      failure_type=type(error).__name__, error=str(error))
        (output/'report.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
        return report
    server = OwnedServer(executable, output/'state', extra_settings=(
        personal_settings if personal_settings is not None else fixture),
        extra_arguments=['--torrentaddr', f'127.0.0.1:{peer_port}'])
    try:
        server.ready()
        with server.request('/echo', timeout=5) as response:
            report['version'] = response.read().decode()
        if personal_settings is not None:
            effective = server.json('/settings', {'action': 'get'})
            def preserved(wanted, actual):
                for key, value in wanted.items():
                    if isinstance(value, dict):
                        preserved(value, actual.get(key, {}))
                    else:
                        assert actual.get(key) == value, key
            preserved(personal_settings, effective)
            report['personal_settings_verified'] = True
            for key, value in fixture.items():
                if key == 'Flow':
                    effective['Flow'].update(value)
                else:
                    effective[key] = value
            with server.request('/settings', {'action': 'set', 'sets': effective,
                                             'when': 'now'}, timeout=15) as response:
                response.read()
            report['fixture_only_settings_overrides'] = True
            server.ready()
        mandatory = set(range(head_last+1)) | set(range(tail_first, tail_last+1))
        bootstrap_last = (prefix.stat().st_size+8*MIB-1)//MIB
        bootstrap = set(range(bootstrap_last+1)) | set(range(tail_first, tail_last+1))
        plan = PeerPlan(pieces=bootstrap, rate=MIB, request_queue=128, strict_request_queue=True)
        with LocalSwarm([prefix, media, suffix], peers=[plan], piece_length=MIB,
                        high_throughput=True) as swarm:
            status = upload(server, swarm)
            info_hash = status['hash']
            index = next(f['id'] for f in status['file_stats'] if f['path'].endswith('generated.ts'))
            with ThreadPoolExecutor(max_workers=1) as pool:
                def preload():
                    with server.request(f'/stream?link={info_hash}&index={index}&preload&stat', timeout=60) as response:
                        return json.load(response)
                began = time.monotonic()
                loading = pool.submit(preload)
                samples, base_ready_at = [], None
                while not loading.done():
                    startup = server.json('/flow/status/'+info_hash+'?traces=false')['startup']
                    samples.append(startup)
                    if startup.get('probe_success') and not report.get('body_released_after_probe'):
                        plan.have_events = [(0, piece) for piece in sorted(mandatory-bootstrap)]
                        report['body_released_after_probe'] = True
                    p = startup.get('progress') or {}
                    if (p.get('head_verified_bytes', 0) >= (head_last+1)*MIB
                            and p.get('missing_tail_pieces') == 0 and not p.get('bootstrap', True)):
                        base_ready_at = base_ready_at or time.monotonic()
                    if time.monotonic()-began > 55:
                        raise TimeoutError('Optional reserve still blocks verified startup')
                    time.sleep(.2)
                loading.result(timeout=3)
            final = server.json('/flow/status/'+info_hash+'?traces=false')
            report['startup'] = final['startup']
            report['elapsed_seconds'] = time.monotonic()-began
            report['extra_wait_seconds'] = None if base_ready_at is None else time.monotonic()-base_ready_at
            report['samples'] = samples
            reserve = report['startup'].get('reserve') or {}
            assert report['startup']['state'] == 'READY'
            assert report['startup']['probe_success'], 'Real ffprobe metadata was not used'
            assert report.get('body_released_after_probe'), 'Fixture did not separate probe from supply measurement'
            assert any((s.get('reserve') or {}).get('requested_bytes', 0) > base for s in samples)
            assert any(s.get('startup_target_bytes', 0) > (head_last+1)*MIB for s in samples), 'Reserve extension not exercised'
            assert reserve.get('extra_wait_limited'), 'Optional wait limit not reported'
            assert report['extra_wait_seconds'] is not None and report['extra_wait_seconds'] < 11
            assert report['startup']['progress']['missing_head_pieces'] == 0
            assert report['startup']['progress']['missing_tail_pieces'] == 0
            assert not any(p['capacity_overflows'] for p in swarm.status()['peers'])
            assert final['integrity']['failures'] == 0
            content = media.read_bytes()
            report['head_range'] = range_read(server, info_hash, index, content, 0, 65535)
            report['tail_range'] = range_read(server, info_hash, index, content, len(content)-65536, len(content)-1)
            with server.request('/torrents', {'action': 'rem', 'hash': info_hash}, timeout=5) as response:
                response.read()
        # A missing required index must still hold startup even though optional
        # reserve can now be removed. Change padding identity for a fresh hash.
        prefix2 = output/'000-padding-2.bin'
        prefix2.write_bytes(bytes(MIB//2))
        with LocalSwarm([prefix2, media, suffix], peers=[PeerPlan(
                pieces=set(range(head_last+1)), rate=4*MIB)], piece_length=MIB,
                high_throughput=True) as swarm:
            status = upload(server, swarm)
            index = next(f['id'] for f in status['file_stats'] if f['path'].endswith('generated.ts'))
            h = status['hash']
            with ThreadPoolExecutor(max_workers=1) as pool:
                def missing():
                    with server.request(f'/stream?link={h}&index={index}&preload&stat', timeout=15) as response:
                        response.read()
                loading = pool.submit(missing)
                try:
                    until = time.monotonic()+5
                    observed_index = False
                    while time.monotonic() < until:
                        startup = server.json('/flow/status/'+h+'?traces=false')['startup']
                        assert startup['state'] != 'READY', 'Required index was bypassed'
                        observed_index |= (startup.get('progress') or {}).get('missing_tail_pieces', 0) > 0
                        assert not loading.done(), 'Missing index preload returned as ready'
                        time.sleep(.2)
                    assert observed_index
                    report['missing_index_held'] = True
                finally:
                    with server.request('/torrents', {'action': 'rem', 'hash': h}, timeout=5) as response:
                        response.read()
                    try:
                        loading.result(timeout=5)
                    except urllib.error.HTTPError as error:
                        if error.code not in (404, 500):
                            raise
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
    print(json.dumps({k: result.get(k) for k in ('failure_stage', 'port_allocation', 'passed', 'elapsed_seconds', 'extra_wait_seconds',
                                           'missing_index_held', 'owned_server_stopped')}))
