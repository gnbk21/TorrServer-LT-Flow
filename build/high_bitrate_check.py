#!/usr/bin/env python3
"""Matched real-engine high transport-bitrate tests on an original local fixture.

Owns fresh loopback state; never connects to public peers or a running server.
The generated MPEG-TS includes transport padding. Measurements concern verified
byte delivery, not decoded visual quality, Android playback or public swarms.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import http.client
import json
import os
from pathlib import Path
import socket
import struct
import subprocess
import time
from urllib.parse import urlencode

from controlled_peer import LocalSwarm, PeerPlan
from playback_harness import MIB, OwnedServer, range_read, upload


def cpu_seconds(process):
    """Owned process CPU only; unavailable platforms report unknown."""
    try:
        if os.name == 'nt':
            import ctypes
            from ctypes import wintypes
            values = [wintypes.FILETIME() for _ in range(4)]
            query = ctypes.WinDLL('kernel32', use_last_error=True).GetProcessTimes
            query.argtypes = [wintypes.HANDLE]+[ctypes.POINTER(wintypes.FILETIME)]*4
            query.restype = wintypes.BOOL
            if query(wintypes.HANDLE(int(process._handle)), *(ctypes.byref(v) for v in values)):
                return sum((v.dwHighDateTime << 32)+v.dwLowDateTime for v in values[2:])/10_000_000
        elif os.name == 'posix' and Path(f'/proc/{process.pid}/stat').exists():
            fields = Path(f'/proc/{process.pid}/stat').read_text().rsplit(')', 1)[1].split()
            return (int(fields[11])+int(fields[12]))/os.sysconf('SC_CLK_TCK')
    except (OSError, ValueError, AttributeError):
        pass
    return None


def calibrate(swarm):
    """Verify peer capacity directly, independently of the Flow scheduler."""
    peer = swarm.peers[0]
    with socket.create_connection(('127.0.0.1', peer.server_address[1]), timeout=10) as sock:
        sock.sendall(b'\x13BitTorrent protocol'+bytes(8)+swarm.info_hash+b'-FLTEST-123456789012')
        def receive(size):
            data = bytearray()
            while len(data) < size:
                chunk = sock.recv(size-len(data))
                if not chunk:
                    stats = swarm.status()
                    raise EOFError(f"Calibration peer disconnected after {stats['requests']} requests and {stats['sent_bytes']} sent bytes")
                data.extend(chunk)
            return bytes(data)
        receive(68)
        size = 16*MIB
        requests = b''.join(struct.pack('!IBIII', 13, 6, offset//swarm.piece_length,
                            offset % swarm.piece_length, 16384) for offset in range(0, size, 16384))
        started = time.monotonic()
        sock.sendall(requests)
        received = 0
        while received < size:
            length, = struct.unpack('!I', receive(4))
            packet = receive(length)
            if packet and packet[0] == 7:
                piece, offset = struct.unpack('!II', packet[1:9])
                start = piece*swarm.piece_length+offset
                if packet[9:] != swarm.data[start:start+len(packet)-9]:
                    raise AssertionError('Calibration bytes changed')
                received += len(packet)-9
        return received*8/(time.monotonic()-started)/1_000_000


def generate(path, seconds):
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists():
        raise ValueError('Fixture path must be new; use --fixture to reuse it')
    subprocess.run([
        'ffmpeg', '-hide_banner', '-loglevel', 'error', '-n', '-f', 'lavfi',
        '-i', 'testsrc2=size=640x360:rate=25', '-t', str(seconds),
        '-c:v', 'mpeg2video', '-b:v', '40M', '-minrate', '40M',
        '-maxrate', '40M', '-bufsize', '80M', '-muxrate', '120M',
        '-metadata', 'title=Flow original high transport bitrate test',
        '-f', 'mpegts', str(path)], check=True, timeout=180)


def paced(server, info_hash, index, source, rate, seconds, bursts):
    size = int(rate * seconds * (1.2 if bursts else 1))
    if size >= len(source):
        raise ValueError('Fixture is too short for the requested workload')
    connection = http.client.HTTPConnection('127.0.0.1', server.port, timeout=45)
    waits, received = [], 0
    started = time.monotonic()
    try:
        connection.request('GET', f'/play/{info_hash}/{index}?play&stat=high-bitrate',
                           headers={'Range': f'bytes=0-{size-1}'})
        response = connection.getresponse()
        if response.status != 206 or response.getheader('Content-Range') != f'bytes 0-{size-1}/{len(source)}':
            raise AssertionError('Incorrect Range status or bounds')
        first = response.read(1)
        if first != source[:1]:
            raise AssertionError('First byte mismatch')
        received = 1
        first_at = time.monotonic()
        due = first_at
        digest = hashlib.sha256(first)
        while received < size:
            if time.monotonic()-started > seconds+90:
                raise TimeoutError(f'Paced playback exceeded its bounded budget after {received}/{size} verified bytes')
            time.sleep(max(0, due-time.monotonic()))
            before = time.monotonic()
            chunk = response.read(min(65536, size-received))
            waits.append((time.monotonic()-before)*1000)
            if not chunk or chunk != source[received:received+len(chunk)]:
                raise AssertionError('Truncated or incorrect verified Range bytes')
            digest.update(chunk)
            received += len(chunk)
            # Alternate nominal and 1.5x demand in six-second transport periods.
            segment = int((received/rate)//6)
            current_rate = rate*(1.5 if bursts and segment % 2 else 1)
            due += len(chunk)/current_rate
        ordered = sorted(waits)
        return {'bytes': received, 'sha256': digest.hexdigest(),
                'ttfb_ms': (first_at-started)*1000,
                'delivery_elapsed_ms': (time.monotonic()-first_at)*1000,
                'scheduled_delivery_ms': (due-first_at)*1000,
                'behind_schedule_ms': max(0, time.monotonic()-due)*1000,
                'reads_over_250ms': sum(w >= 250 for w in waits),
                'read_wait_p99_ms': ordered[min(len(ordered)-1, int(len(ordered)*.99))],
                'read_wait_max_ms': max(waits), 'verified_bytes': True}
    finally:
        connection.close()


def run(executable, root, fixture, profile, case, rate, seconds, cache_mb, preload=False):
    report = {'profile': profile, 'case': case, 'cache_mb': cache_mb,
              'nominal_demand_mbps': rate*8/1_000_000,
              'piece_bytes': 4*MIB, 'requested_seconds': seconds,
              'preload': preload, 'http_delivery_only': True, 'samples': []}
    # Native time-critical work can exceed desired_queue_size. Advertise a
    # realistic capacity, with bounded fixture headroom for in-flight cancels.
    plans = [PeerPlan(rate=int(2*rate), request_queue=512,
                      outages=[(16, 19), (32, 35)] if case == 'outages' else [])]
    if case == 'mixed-peers':
        plans = [PeerPlan(rate=int(1.7*rate), request_queue=512)] + [PeerPlan(rate=rate//20, request_queue=512) for _ in range(7)]
    report['peer_advertised_request_queue'] = 512
    server = OwnedServer(executable, root, extra_settings={
        'CacheSize': cache_mb*MIB, 'PreloadCache': 10,
        'Flow': {'SwarmProfile': profile, 'BootstrapHeadMB': 4,
                 'ProbeGraceMs': 300, 'StartupBufferMinMB': 4,
                 'StartupBufferMaxMB': 32, 'StartupBufferSeconds': 2,
                 'WarmSessionTimeoutSec': 30, 'GlobalCacheBudgetMB': cache_mb,
                 'DiagnosticHistory': False}})
    swarm, cpu_start, playback_start = None, None, None
    try:
        report['ready_ms'] = server.ready()
        with server.request('/echo') as response:
            report['version'] = response.read().decode()
        with LocalSwarm([fixture], peers=plans, piece_length=4*MIB, high_throughput=True) as swarm:
            report['fixture_sha256'] = hashlib.sha256(swarm.data).hexdigest()
            info = upload(server, swarm)
            index = info['file_stats'][0]['id']
            info_hash = swarm.info_hash.hex()
            if preload:
                started = time.monotonic()
                server.json('/stream/generated?'+urlencode({'link': info_hash, 'index': index, 'preload': '', 'stat': ''}), timeout=90)
                deadline = started+90
                while True:
                    status = server.json('/torrents', {'action': 'get', 'hash': info_hash})
                    if status.get('preloaded_bytes', 0) >= status.get('preload_size', 1) > 0:
                        break
                    if time.monotonic() >= deadline:
                        raise TimeoutError('Explicit startup buffer did not complete')
                    time.sleep(.1)
                report['preload_ready_ms'] = (time.monotonic()-started)*1000
            cpu_start = cpu_seconds(server.process)
            playback_start = time.monotonic()
            with ThreadPoolExecutor(max_workers=1) as pool:
                pending = pool.submit(paced, server, info_hash, index, swarm.data,
                                      rate, seconds, case == 'bursts')
                while not pending.done():
                    status = server.json('/flow/status/'+info_hash, timeout=5)
                    sessions = status.get('sessions') or []
                    report['samples'].append({
                        'elapsed_s': time.monotonic()-swarm.started,
                        'cache_used': max((s['cache_used'] for s in sessions), default=0),
                        'cache_size': max((s['cache_size'] for s in sessions), default=0),
                        'sessions': sessions,
                        'urgent': status.get('sparse', {}).get('urgent'),
                        'urgent_truncated': status.get('sparse', {}).get('urgent_truncated')})
                    if len(report['samples']) % 5 == 1:
                        report['samples'][-1]['memory'] = server.json('/runtime/status')['memory']
                    time.sleep(1)
                report['delivery'] = pending.result()
            cpu_end = cpu_seconds(server.process)
            report['cpu_seconds'] = cpu_end-cpu_start if cpu_start is not None and cpu_end is not None else None
            report['playback_wall_seconds'] = time.monotonic()-playback_start
            report['peak_observed_rss_bytes'] = max((s.get('memory', {}).get('rss_bytes', 0) for s in report['samples']), default=0)
            report['peak_observed_cache_bytes'] = max(s['cache_used'] for s in report['samples'])
            report['qualified_consumption_observed'] = any(
                session.get('observed_confidence') == 'stable'
                for sample in report['samples'] for session in sample['sessions'])
            report['credible_media_observed'] = any(
                session.get('bitrate_estimate_confidence') in ('medium', 'high')
                for sample in report['samples'] for session in sample['sessions'])
            if profile == 'adaptive' and not report['qualified_consumption_observed']:
                raise AssertionError('Adaptive workload never qualified sequential consumption evidence')
            if report['peak_observed_cache_bytes'] > (cache_mb+8)*MIB:
                raise AssertionError('Observed cache exceeded budget plus two-piece concurrency allowance')
            report['ranges'] = []
            for start, cancel in ((len(swarm.data)//2, False), (0, False),
                                  (len(swarm.data)-65536, False), (len(swarm.data)//3, True), (0, False)):
                report['ranges'].append(range_read(server, info_hash, index,
                    swarm.data, start, min(len(swarm.data)-1, start+65535), cancel))
            deadline = time.monotonic()+5
            while True:
                status = server.json('/flow/status/'+info_hash)
                active = sum(s['active_readers'] for s in status.get('sessions') or [])
                if active == 0:
                    break
                if time.monotonic() >= deadline:
                    raise AssertionError('Readers leaked after cancellation')
                time.sleep(.2)
            report['active_readers_after_cancellation'] = active
            report['runtime'] = server.json('/runtime/status')
            report['swarm'] = swarm.status()
            report['passed'] = True
    except Exception as error:
        report['passed'], report['error'] = False, str(error)
    finally:
        # Keep failure evidence too: neither a source disconnect nor a timeout
        # should hide resource peaks and whether qualified demand was observed.
        if swarm is not None:
            report['swarm'] = swarm.status()
        if playback_start is not None:
            cpu_end = cpu_seconds(server.process)
            report.setdefault('cpu_seconds', cpu_end-cpu_start if cpu_start is not None and cpu_end is not None else None)
            report.setdefault('playback_wall_seconds', time.monotonic()-playback_start)
        if report['samples']:
            report['peak_observed_rss_bytes'] = max((s.get('memory', {}).get('rss_bytes', 0) for s in report['samples']), default=0)
            report['peak_observed_cache_bytes'] = max(s['cache_used'] for s in report['samples'])
            report['qualified_consumption_observed'] = any(
                session.get('observed_confidence') == 'stable'
                for sample in report['samples'] for session in sample['sessions'])
            report['credible_media_observed'] = any(
                session.get('bitrate_estimate_confidence') in ('medium', 'high')
                for sample in report['samples'] for session in sample['sessions'])
        server.close()
        (root/'report.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    return report


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--executable', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--fixture', type=Path)
    parser.add_argument('--profile', choices=('legacy', 'adaptive'), default='legacy')
    parser.add_argument('--cases', nargs='+', choices=('healthy', 'outages', 'mixed-peers', 'bursts'),
                        default=['healthy', 'outages', 'mixed-peers', 'bursts'])
    parser.add_argument('--mbps', type=float, default=90)
    parser.add_argument('--seconds', type=int, default=40)
    parser.add_argument('--cache-mb', type=int, default=128)
    parser.add_argument('--preload', action='store_true', help='Exercise the Lampa-style explicit bootstrap/probe path')
    args = parser.parse_args()
    if not 10 <= args.seconds <= 120 or not 10 <= args.mbps <= 200 or not 64 <= args.cache_mb <= 2048:
        parser.error('seconds 10..120, Mbps 10..200 and cache 64..2048 MiB required')
    if not args.executable.is_file():
        parser.error('Executable does not exist')
    args.output.mkdir(parents=True, exist_ok=False)
    fixture = args.fixture or args.output/'fixture'/'generated.ts'
    if args.fixture is None:
        generate(fixture, int(args.seconds*args.mbps/120*1.3)+10)
    results = {'executable_sha256': hashlib.sha256(args.executable.read_bytes()).hexdigest(),
               'generated_transport_fixture': True, 'public_discovery_disabled': True, 'cases': []}
    with LocalSwarm([fixture], peers=[PeerPlan(rate=int(2*args.mbps*1_000_000/8))],
                    piece_length=4*MIB, high_throughput=True) as swarm:
        results['peer_capacity_mbps'] = calibrate(swarm)
    if results['peer_capacity_mbps'] < 1.3*args.mbps:
        (args.output/'report.json').write_text(json.dumps(results, indent=2)+'\n', encoding='utf-8')
        raise RuntimeError('Local peer capacity is insufficient for a fair high-bitrate comparison')
    for case in args.cases:
        print(f'High bitrate: {args.profile} / {case}', flush=True)
        results['cases'].append(run(args.executable, args.output/case, fixture,
                                    args.profile, case, int(args.mbps*1_000_000/8), args.seconds, args.cache_mb, args.preload))
        (args.output/'report.json').write_text(json.dumps(results, indent=2)+'\n', encoding='utf-8')
    raise SystemExit(0 if all(case['passed'] for case in results['cases']) else 1)
