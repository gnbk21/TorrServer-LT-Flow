#!/usr/bin/env python3
"""Generated-media sparse-swarm acceptance and comparisons, using a real engine.

Each case owns its process, state and tracker; public discovery is disabled.
All returned Range bytes are checked. Timings are HTTP delivery observations,
not decoder first-frame timings or proof about any public swarm.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import json
import hashlib
import http.client
import math
from pathlib import Path
import threading
import time
from urllib.parse import quote, urlencode

from controlled_peer import LocalSwarm, PeerPlan
from fixtures import generate
from high_bitrate_check import cpu_seconds, observed_rss_bytes
from playback_harness import OwnedServer, upload, range_read, MIB


CASES = ('healthy', 'below', 'near', 'above', 'complementary', 'ten-suppliers', 'small-swarm', 'multiple-rare', 'choked-supplier', 'intermittent-supplier', 'late-have', 'diagnostic-seek', 'missing-seek', 'warm-reconnect', 'choked', 'outage', 'late-metadata', 'latency', 'variation')


def paced_read(server, hash_text, index, source, rate, seconds=8):
    """Measure server delivery to a paced client, without claiming decoder stalls."""
    size=min(len(source), int(rate*seconds))
    connection=http.client.HTTPConnection('127.0.0.1', server.port, timeout=120)
    started=time.monotonic()
    received=bytearray()
    waits=[]
    try:
        connection.request('GET',f'/play/{hash_text}/{index}?play&stat=paced',headers={'Range':f'bytes=0-{size-1}'})
        response=connection.getresponse()
        if response.status!=206: raise AssertionError('paced Range failed')
        while len(received)<size:
            due=started+len(received)/rate
            time.sleep(max(0,due-time.monotonic()))
            before=time.monotonic()
            chunk=response.read(min(65536,size-len(received)))
            waits.append((time.monotonic()-before)*1000)
            if not chunk: raise AssertionError('paced response truncated')
            received.extend(chunk)
        if received!=source[:size]: raise AssertionError('paced response bytes changed')
        return {'bytes':size,'elapsed_ms':(time.monotonic()-started)*1000,
                'client_read_wait_ms':sum(waits),'client_reads_over_500ms':sum(w>=500 for w in waits),
                'maximum_client_read_wait_ms':max(waits),'http_delivery_only':True}
    finally:
        connection.close()


def observe_missing_seek(server, swarm, index, source, piece, offset, supplier, report):
    status_path = '/flow/status/' + swarm.info_hash.hex()
    report['pending_seek_samples'] = []
    with ThreadPoolExecutor(max_workers=1) as pool:
        task = pool.submit(range_read, server, swarm.info_hash.hex(), index, source,
                           offset, min(len(source) - 1, offset + 65535), timeout=8)
        while not task.done():
            sample = server.json(status_path, timeout=2)
            report['pending_seek_samples'].append(sample)
            windows = sample.get('sparse', {}).get('windows', [])
            missing = any(w['first_piece'] == piece and w['availability'] and w['availability'][0] == 0
                          for w in windows)
            waiting = any(s.get('active_readers', 0) > 0 and s.get('wait_reason') == 'MISSING_CONNECTED'
                          and s.get('required_piece_suppliers') == 0
                          and s.get('playback_offset_bytes', 0) < offset for s in sample.get('sessions', []))
            if missing and waiting and 'pending_seek_missing_piece_observed' not in report:
                report['pending_seek_missing_piece_observed'] = True
                if any(piece in p['first_request_seconds'] for p in swarm.status()['peers']):
                    raise AssertionError('Absent seek piece was requested before the supplier was released')
                if supplier is not None:
                    report['supplier_release_seconds'] = time.monotonic() - swarm.started
                    supplier.set()
            time.sleep(.1)
        if not report.get('pending_seek_missing_piece_observed'):
            raise AssertionError('Pending seek never reported the unavailable target')
        if supplier is not None:
            return task.result()
        try:
            task.result()
        except TimeoutError:
            report['missing_seek_client_timeout'] = True
        else:
            raise AssertionError('Missing piece unexpectedly completed')
    deadline = time.monotonic() + 3
    while time.monotonic() < deadline:
        sample = server.json(status_path, timeout=2)
        if not any(s.get('active_readers', 0) for s in sample.get('sessions', [])):
            if any(piece in p['first_request_seconds'] for p in swarm.status()['peers']):
                raise AssertionError('Client requested a piece no peer advertised')
            report['missing_seek_reader_released'] = True
            return None
        time.sleep(.1)
    raise AssertionError('Timed-out seek retained an active reader')


def run(executable, root, fixture, label, media_rate, profile='custom', piece_length=256*1024, scarce=False, rate_aware=False):
    source = fixture.read_bytes()
    plen = piece_length
    count = math.ceil(len(source)/plen)
    urgent = count//2
    all_pieces = set(range(count))
    rate = max(64*1024, int(media_rate))
    supplier = threading.Event() if label == 'diagnostic-seek' else None
    paused = threading.Event() if label == 'warm-reconnect' else None
    if label in ('healthy', 'warm-reconnect'):
        plans = [PeerPlan(rate=8*rate, paused=paused)]
    elif label in ('below', 'near', 'above'):
        ratio = {'below':.6, 'near':1, 'above':2.5}[label]
        plans = [PeerPlan(rate=int(rate*ratio))]
    elif label == 'complementary':
        plans = [PeerPlan(pieces=set(range(i,count,2)), rate=2*rate) for i in range(2)]
    elif label == 'ten-suppliers':
        plans = [PeerPlan(pieces=all_pieces-{urgent}, rate=8*rate) for _ in range(9)]
        plans += [PeerPlan(pieces={0,urgent}, rate=max(32768,rate//4))]
    elif label in ('small-swarm','multiple-rare','choked-supplier','intermittent-supplier'):
        fast_count=3 if label=='small-swarm' else 9
        plans=[PeerPlan(pieces=all_pieces-{urgent},rate=8*rate) for _ in range(fast_count)]
        plans += [PeerPlan(pieces={0,urgent},rate=max(32768,rate//4),choke_until=6 if label=='choked-supplier' else 0,outages=[(1,6)] if label=='intermittent-supplier' else [])]
        if label=='multiple-rare': plans += [PeerPlan(pieces={urgent},rate=max(32768,rate//3))]
    elif label == 'late-have':
        plans = [PeerPlan(pieces=all_pieces-{urgent},rate=8*rate,have_events=[(5,urgent)])]
    elif label in ('diagnostic-seek', 'missing-seek'):
        plans = [PeerPlan(pieces=all_pieces-{urgent}, rate=8*rate,
                          gated_pieces={urgent: supplier} if supplier is not None else {})]
    elif label == 'choked':
        plans = [PeerPlan(rate=8*rate,choke_until=3)]
    elif label == 'outage':
        plans = [PeerPlan(rate=2*rate,outages=[(1,4)])]
    elif label == 'latency':
        plans = [PeerPlan(rate=2*rate,delay_ms=100)]
    elif label == 'variation':
        plans = [PeerPlan(rate=4*rate,rate_events=[(3,max(32768,rate//3)),(7,4*rate),(11,max(32768,rate//2)),(14,3*rate)])]
    else:
        plans = [PeerPlan(rate=8*rate,metadata_delay=3)]
    report = {'case':label, 'profile':profile, 'piece_length':plen, 'scarce_hints':scarce, 'rate_aware_deadlines':rate_aware, 'http_delivery_only':True, 'media_bytes_per_second':media_rate, 'fixture_sha256':hashlib.sha256(source).hexdigest(), 'ranges':[]}
    server = None
    resource_samples = []
    try:
        with LocalSwarm([fixture],peers=plans,piece_length=plen) as swarm:
            server = OwnedServer(executable, root/label, extra_settings={
                'CacheSize': 8*MIB, 'PreloadCache': 12, 'TrackersListURL': swarm.tracker_list_url,
                'Flow': {'Enabled': True, 'SwarmProfile': profile, 'SwarmCustom': {'MinReconnectTime': 1, 'PeerConnectTimeout': 5},
                         'ScarcePieceHints': scarce, 'RateAwareDeadlines': rate_aware, 'BootstrapHeadMB': 1,
                         'StartupBufferMinMB': 1, 'StartupBufferMaxMB': 8, 'StartupBufferSeconds': 1}})
            server.ready()
            started = time.monotonic()
            cpu_start = cpu_seconds(server.process)
            resource_samples.append(server.json('/runtime/status', timeout=3))
            if label == 'late-metadata':
                tracker = f'http://127.0.0.1:{swarm.tracker.server_port}/announce'
                magnet = 'magnet:?xt=urn:btih:'+swarm.info_hash.hex()+'&tr='+quote(tracker,safe='')
                status = server.json('/torrents',{'action':'add','link':magnet,'save_to_db':True},timeout=30)
                # A DB add may return before metadata. Explicit info lookup owns
                # metadata activation, as it does for a normal playlist request.
                deadline=time.monotonic()+30
                while not status.get('file_stats') and time.monotonic()<deadline:
                    status=server.json('/torrents',{'action':'get','hash':swarm.info_hash.hex()},timeout=30)
                    time.sleep(.1)
                if not status.get('file_stats'): raise AssertionError('metadata never supplied')
                report['metadata_ready_ms']=(time.monotonic()-started)*1000
                if report['metadata_ready_ms']<2500: raise AssertionError('metadata delay was not exercised')
            else:
                status=upload(server,swarm)
            if status['hash']!=swarm.info_hash.hex(): raise AssertionError('identity changed')
            index=status['file_stats'][0]['id']
            if label in ('below','near','above','latency','variation'):
                preload_started=time.monotonic()
                server.json('/stream/generated?'+urlencode({'link':swarm.info_hash.hex(),'index':index,'preload':'','stat':'controlled-preload'}),timeout=90)
                report['buffer_gate_response_ms']=(time.monotonic()-preload_started)*1000
            with ThreadPoolExecutor(max_workers=1) as pool:
                task=pool.submit(range_read,server,swarm.info_hash.hex(),index,source,0,min(len(source)-1,65535))
                report['samples']=[]
                while not task.done():
                    report['samples'].append(server.json('/flow/status/'+swarm.info_hash.hex()))
                    if time.monotonic()-started>120: raise TimeoutError('head request exceeded case budget')
                    time.sleep(.5)
                report['ranges'].append(task.result())
            resource_samples.append(server.json('/runtime/status', timeout=3))
            before=swarm.status()
            seek_start=time.monotonic()-swarm.started
            offset=urgent*plen
            report['rare_requested_before_seek']=any(urgent in p['first_request_seconds'] for p in before['peers'])
            if label in ('diagnostic-seek', 'missing-seek'):
                delivered = observe_missing_seek(server, swarm, index, source, urgent, offset, supplier, report)
                if delivered is not None:
                    report['ranges'].append(delivered)
            else:
                report['ranges'].append(range_read(server,swarm.info_hash.hex(),index,source,offset,min(len(source)-1,offset+65535)))
            if paused is not None:
                paused.set()
                deadline = time.monotonic() + 3
                while not swarm.status()['disconnects'] and time.monotonic() < deadline:
                    time.sleep(.05)
                if not swarm.status()['disconnects']:
                    raise AssertionError('Controlled peer did not disconnect')
                report['warm_during_outage'] = range_read(
                    server, swarm.info_hash.hex(), index, source, offset, min(len(source) - 1, offset + 65535), timeout=3)
                before_reconnect = swarm.status()['connections']
                paused.clear()
                deadline = time.monotonic() + 30
                while swarm.status()['connections'] <= before_reconnect and time.monotonic() < deadline:
                    time.sleep(.1)
                if swarm.status()['connections'] <= before_reconnect:
                    raise AssertionError('Controlled peer did not reconnect')
                report['warm_after_reconnect'] = range_read(
                    server, swarm.info_hash.hex(), index, source, offset, min(len(source) - 1, offset + 65535), timeout=3)
            report['ranges'].append(range_read(server,swarm.info_hash.hex(),index,source,0,65535))
            if label in ('below','near','above','latency','variation'):
                with ThreadPoolExecutor(max_workers=1) as pool:
                    task=pool.submit(paced_read,server,swarm.info_hash.hex(),index,source,media_rate)
                    while not task.done():
                        report['samples'].append(server.json('/flow/status/'+swarm.info_hash.hex()))
                        resource_samples.append(server.json('/runtime/status', timeout=3))
                        time.sleep(.5)
                    report['paced_delivery']=task.result()
            report['swarm']=swarm.status()
            if report['swarm']['tracker_list_requests'] < 1:
                raise AssertionError('Engine never loaded the loopback tracker list')
            requests=[p['first_request_seconds'][urgent] for p in report['swarm']['peers'] if urgent in p['first_request_seconds']]
            if supplier is not None and (not requests or min(requests) < report['supplier_release_seconds']):
                raise AssertionError('Late supplier did not serve the seek after release')
            report['urgent_first_request_ms_after_seek']=(min(requests)-seek_start)*1000 if requests else None
            sent=[p['first_sent_seconds'] for p in report['swarm']['peers'] if p['first_sent_seconds'] is not None]
            report['first_peer_payload_ms']=min(sent)*1000 if sent else None
            report['flow']=server.json('/flow/status/'+swarm.info_hash.hex())
            report['runtime']=server.json('/runtime/status')
            resource_samples.append(report['runtime'])
            rss = observed_rss_bytes(resource_samples)
            report['peak_observed_rss_bytes'] = max(rss) if rss else None
            report['rss_observations_available'] = len(rss)
            cpu_end = cpu_seconds(server.process)
            report['cpu_seconds'] = cpu_end - cpu_start if cpu_start is not None and cpu_end is not None else None
            report['elapsed_ms']=(time.monotonic()-started)*1000
            report['passed']=True
    except Exception as error:
        report['passed']=False
        report['error']=str(error)
        if 'swarm' in locals(): report['swarm']=swarm.status()
        if server is not None:
            try: report['flow']=server.json('/flow/status/'+swarm.info_hash.hex())
            except (OSError,ValueError): pass
        return report
    finally:
        if server is not None:
            server.close()
        (root/label).mkdir(parents=True, exist_ok=True)
        (root/label/'report.json').write_text(json.dumps(report,indent=2),encoding='utf-8')
    return report


if __name__=='__main__':
    parser=argparse.ArgumentParser()
    parser.add_argument('--executable',required=True,type=Path)
    parser.add_argument('--output',required=True,type=Path)
    parser.add_argument('--fixtures',type=Path)
    parser.add_argument('--cases',nargs='+',choices=CASES,default=list(CASES))
    parser.add_argument('--profile',choices=('legacy','adaptive','conservative','balanced','custom'),default='custom')
    parser.add_argument('--piece-length',type=int,default=256*1024)
    parser.add_argument('--scarce',action='store_true')
    parser.add_argument('--rate-aware',action='store_true')
    parser.add_argument('--record-failures',action='store_true',help='Characterize profiles; report failures without treating them as a passing gate')
    args=parser.parse_args()
    args.output.mkdir(parents=True,exist_ok=True)
    fixtures=args.fixtures or args.output/'fixtures'
    if not args.fixtures: generate(fixtures,seconds=12)
    manifest=json.loads((fixtures/'fixtures.json').read_text(encoding='utf-8'))
    entry=next(f for f in manifest['fixtures'] if f['name']=='variable.mp4')
    result={'executable':str(args.executable.resolve()),'executable_sha256':hashlib.sha256(args.executable.read_bytes()).hexdigest(),'generated_media':True,'cases':[]}
    try:
        for label in args.cases:
            print('Sparse case: '+label,flush=True)
            result['cases'].append(run(args.executable,args.output,fixtures/'variable.mp4',label,entry['bytes']/entry['duration'],args.profile,args.piece_length,args.scarce,args.rate_aware))
            (args.output/'report.json').write_text(json.dumps(result,indent=2)+'\n',encoding='utf-8')
            if not result['cases'][-1]['passed'] and not args.record_failures: raise AssertionError(result['cases'][-1]['error'])
        result['passed']=all(case['passed'] for case in result['cases'])
        result['characterization_only']=args.record_failures
    except Exception as error:
        result['error']=str(error)
        raise
    finally:
        (args.output/'report.json').write_text(json.dumps(result,indent=2)+'\n',encoding='utf-8')
