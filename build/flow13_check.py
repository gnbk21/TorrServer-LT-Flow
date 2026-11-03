#!/usr/bin/env python3
"""Flow 1.3 integration on original fixtures and an isolated loopback server.

Tests metadata-only observation, exact subtitle scope/bytes, media delivery,
Recent/provenance and cached monitoring together. No personal state, public
swarm, decoded-player claim or long-duration test is involved.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
from pathlib import Path
import re
import time
import urllib.error
import urllib.request

from controlled_peer import LocalSwarm
from fixtures import generate
from playback_harness import OwnedServer, upload, range_read, MIB


def check(executable, output, fixtures=None):
    output.mkdir(parents=True, exist_ok=False)
    videos = generate(output/'fixtures') if fixtures is None else [
        fixtures/row['name'] for row in json.loads((fixtures/'fixtures.json').read_text(encoding='utf-8'))['fixtures']]
    sidecars = output/'sidecars'
    sidecars.mkdir()
    original = b'1\n00:00:01,000 --> 00:00:02,500\n<script>original fixture</script>\n'
    paths = []
    for name, data in [('head.en.srt', original), ('tail.ru.srt', original),
                       ('head.vtt', b'not a valid VTT'),
                       ('head.en.forced.srt', b'x'*(4*MIB+1))]:
        path = sidecars/name
        path.write_bytes(data)
        paths.append(path)
    report = {'schema_version': 1, 'executable_sha256': hashlib.sha256(executable.read_bytes()).hexdigest(),
              'synthetic_media_only': True, 'phone_acceptance': False, 'ranges': [], 'observations': []}
    server = OwnedServer(executable, output/'state', auth=True, extra_settings={
        'CacheSize': 128*MIB, 'TrackTimecode': True,
        'Flow': {'Enabled': True, 'SwarmProfile': 'legacy', 'MetricsEnabled': True,
                 'RecentPlaybackHistory': True, 'PrometheusEnabled': True,
                 'RangeTraceEnabled': True, 'MetadataFallback': False, 'PublicMetadataCache': False,
                 'WarmSessionTimeoutSec': 30, 'BootstrapHeadMB': 1,
                 'StartupBufferSeconds': 1, 'StartupBufferMinMB': 1, 'StartupBufferMaxMB': 8}})
    def status_without_auth(path, headers=None):
        try:
            with urllib.request.urlopen(urllib.request.Request(server.base+path, headers=headers or {}), timeout=12) as response:
                return response.status, response.read(), dict(response.headers)
        except urllib.error.HTTPError as error:
            return error.code, error.read(), dict(error.headers)
    def selected(path):
        deadline = time.monotonic()+12
        while True:
            try:
                with server.request(path, timeout=12) as response:
                    return response.read(), dict(response.headers)
            except urllib.error.HTTPError as error:
                body = error.read().decode('utf-8', errors='replace')
                assert json.loads(body).get('schema_version') == 1, 'Unversioned subtitle failure'
                report['observations'].append({'read_status':error.code,'error_body':body})
                if error.code != 503 or time.monotonic() >= deadline:
                    raise
                report['observations'].append({'read_deferred': True, 'status': error.code})
                time.sleep(.2)
    try:
        server.ready()
        with LocalSwarm(videos+paths) as swarm:
            info = upload(server, swarm)
            identity = info['hash']
            files = {Path(row['path']).name: row for row in info['file_stats']}
            video = files['head.mp4']['id']
            subtitle = files['head.en.srt']['id']
            initial_bytes = swarm.status()['sent_bytes']
            for _ in range(3):
                listing = server.json(f'/subs/{identity}/{video}')
                cached = server.json(f'/flow/media/{identity}/{video}')
                coverage = server.json(f'/flow/status/{identity}?coverage=true')
                server.json('/flow/search-status')
            assert swarm.status()['sent_bytes'] == initial_bytes, 'Observation downloaded media'
            assert not cached['known'], 'Observation launched an optional probe'
            assert len(coverage.get('sessions') or []) == 0, 'Observation created a session'
            matches = {row['file_index'] for row in listing['tracks']}
            assert subtitle in matches and files['tail.ru.srt']['id'] not in matches
            assert listing['schema_version'] == 1 and not listing['truncated']
            report['metadata_only_observation'] = True
            # Player/container probes deliver bytes but are not episode opens.
            # Keep their existing legacy Viewed marks independent from Recent.
            report['ranges'].append(range_read(server,identity,video,videos[0].read_bytes(),0,1023))
            index_source=next(path for path in videos if path.name=='variable.mp4')
            index_bytes=index_source.read_bytes()
            assert len(index_bytes)>8*MIB
            report['ranges'].append(range_read(server,identity,files[index_source.name]['id'],index_bytes,len(index_bytes)-65536,len(index_bytes)-1))
            time.sleep(3)  # allow the ordinary history worker to expose false opens
            assert not server.json('/flow/recent')['items'],'Header/index probes fabricated Recent openings'
            report['probes_excluded_from_recent']=True
            # Existing head/tail-index, VBR, seek, overlapping-reader and cancel
            # fixtures run with detailed coverage and metrics being observed.
            for path in videos:
                row = files[path.name]
                source = path.read_bytes()
                index = row['id']
                report['ranges'].append(range_read(server, identity, index, source, 0, min(len(source)-1, 3*MIB)))
                report['ranges'].append(range_read(server, identity, index, source, max(0,len(source)-65536),len(source)-1))
                with ThreadPoolExecutor(max_workers=2) as pool:
                    tasks = [pool.submit(range_read,server,identity,index,source,start,min(len(source)-1,start+65535))
                             for start in [len(source)//3,len(source)//2]]
                    report['ranges'].extend(task.result(timeout=30) for task in tasks)
                report['ranges'].append(range_read(server,identity,index,source,0,min(len(source)-1,131071),cancel=True))
                sample=server.json(f'/flow/status/{identity}?coverage=true')
                assert len(sample['coverage']['pieces']) <= 256 and len(sample['coverage']['frontiers']) <= 8
                with server.request('/metrics') as response:
                    assert b'flow_read_wait_seconds_total' in response.read()
            original_bytes, headers = selected(f'/subs/{identity}/{video}?track={subtitle}&format=original')
            assert original_bytes == original, 'Original subtitle bytes changed'
            assert 'no-store' in headers.get('Cache-Control','')
            converted, _ = selected(f'/subs/{identity}/{video}?track={subtitle}&format=vtt')
            assert converted.startswith(b'WEBVTT') and b'&lt;script&gt;' in converted and b'<script>' not in converted
            for file, expected in [('head.vtt',422),('head.en.forced.srt',413)]:
                try:
                    selected(f'/subs/{identity}/{video}?track={files[file]["id"]}&format=original')
                    raise AssertionError('Invalid subtitle accepted')
                except urllib.error.HTTPError as error:
                    assert error.code == expected, (file,error.code)
            old_link = server.json('/flow/playback-link', {'hash':identity,'index':video})
            assert status_without_auth(old_link['path']+'/media.srt')[0] == 404
            link = server.json('/flow/playback-link', {'hash':identity,'index':video,'subtitle_file_index':subtitle})
            status, data, _ = status_without_auth(link['subtitle_path'])
            assert status == 200 and data == original, 'Subtitle-scoped capability failed'
            status, data, _ = status_without_auth(link['path'], {'Range':'bytes=0-65535'})
            assert status == 206 and data == videos[0].read_bytes()[:65536], 'Named video alias changed bytes'
            assert status_without_auth(link['subtitle_path'].replace('media.srt','media.ass'))[0] == 404
            assert status_without_auth('/flow/recent')[0] == 401, 'Media capability acquired management authority'
            time.sleep(1.2)  # existing watcher drains first-byte history events
            recent = server.json('/flow/recent')
            assert {row['file_index'] for row in recent['items']} == {files[path.name]['id'] for path in videos}
            viewed = server.json('/viewed', {'action':'list','hash':identity})
            assert not any(row.get('completed') for row in viewed), 'Transferred bytes fabricated completion'
            with server.request('/viewed', {'action':'set','hash':identity,'file_index':video,'timecode':42.5,'position_source':'internal_player'}):
                pass
            saved = next(row for row in server.json('/viewed',{'action':'list','hash':identity}) if row['file_index']==video)
            assert saved['timecode'] == 42.5 and saved['position_source'] == 'internal_player' and saved['position_updated_at']
            with server.request('/viewed', {'action':'set','hash':identity,'file_index':video,'completed':True}):
                pass
            saved = next(row for row in server.json('/viewed',{'action':'list','hash':identity}) if row['file_index']==video)
            assert saved['timecode'] == 42.5 and saved['completed'] is True
            with server.request('/metrics') as response:
                metrics = response.read().decode()
            assert all(secret not in metrics for secret in [identity,'head.mp4','original fixture','127.0.0.1'])
            report['subtitle_scope_integrity'] = True
            report['recent_provenance'] = True
            # Detailed observation and metrics must not extend the warm lease.
            deadline = time.monotonic()+45
            while True:
                runtime=server.json('/runtime/status')
                server.json(f'/flow/status/{identity}?coverage=true')
                server.json(f'/subs/{identity}/{video}')
                server.json('/flow/recent')
                with server.request('/metrics') as response:
                    last_metrics=response.read().decode()
                # torrent_count includes retained library entries. Expiry drops
                # native/cache ownership, while the metadata record remains.
                row = next((row for row in runtime['bt']['torrents'] if row['hash'] == identity), None)
                if row is not None and row['stat'] in [4, 5] and runtime['resources']['ram_resident_bytes'] == 0:
                    break
                if time.monotonic()>=deadline:
                    raise AssertionError('Read-only observation prevented torrent expiry')
                time.sleep(1)
            metric=lambda text,name:float(re.search(r'^'+re.escape(name)+r' ([\d.eE+-]+)$',text,re.M).group(1))
            assert metric(last_metrics,'flow_cache_hit_bytes_total') >= metric(metrics,'flow_cache_hit_bytes_total')
            report['observation_did_not_prolong_torrent'] = True
            report['monotonic_metrics_after_removal'] = True
            report['peer'] = swarm.status()
        report['passed'] = True
    except Exception as error:
        report['error'] = f'{type(error).__name__}: {error}'
        report['passed'] = False
        raise
    finally:
        server.close()
        (output/'report.json').write_text(json.dumps(report,indent=2)+'\n',encoding='utf-8')
    return report


if __name__ == '__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--executable',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--fixtures',type=Path)
    args=parser.parse_args()
    check(args.executable,args.output,args.fixtures)
