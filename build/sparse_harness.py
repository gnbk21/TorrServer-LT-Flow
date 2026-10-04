#!/usr/bin/env python3
"""Generated-media sparse-swarm acceptance and comparisons, using a real engine.

Each case owns its process, state and tracker; public discovery is disabled.
All returned Range bytes are checked. Timings are HTTP delivery observations,
not decoder first-frame timings or proof about any public swarm.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import json
import math
from pathlib import Path
import time
from urllib.parse import quote

from controlled_peer import LocalSwarm, PeerPlan
from fixtures import generate
from playback_harness import OwnedServer, upload, range_read, MIB


CASES = ('below', 'near', 'above', 'complementary', 'ten-suppliers', 'late-have', 'choked', 'outage', 'late-metadata')


def run(executable, root, fixture, label, media_rate):
    source = fixture.read_bytes()
    plen = 256*1024
    count = math.ceil(len(source)/plen)
    urgent = count//2
    all_pieces = set(range(count))
    rate = max(64*1024, int(media_rate))
    if label in ('below', 'near', 'above'):
        ratio = {'below':.6, 'near':1, 'above':2.5}[label]
        plans = [PeerPlan(rate=int(rate*ratio))]
    elif label == 'complementary':
        plans = [PeerPlan(pieces=set(range(i,count,2)), rate=2*rate) for i in range(2)]
    elif label == 'ten-suppliers':
        plans = [PeerPlan(pieces=all_pieces-{urgent}, rate=8*rate) for _ in range(9)]
        plans += [PeerPlan(pieces={0,urgent}, rate=max(32768,rate//4))]
    elif label == 'late-have':
        plans = [PeerPlan(pieces=all_pieces-{urgent},rate=8*rate,have_events=[(5,urgent)])]
    elif label == 'choked':
        plans = [PeerPlan(rate=8*rate,choke_until=3)]
    elif label == 'outage':
        plans = [PeerPlan(rate=2*rate,outages=[(1,4)])]
    else:
        plans = [PeerPlan(rate=8*rate,metadata_delay=3)]
    report = {'case':label, 'http_delivery_only':True, 'media_bytes_per_second':media_rate, 'ranges':[]}
    server = OwnedServer(executable,root/label,extra_settings={'CacheSize':8*MIB,'PreloadCache':12})
    try:
        server.ready()
        with LocalSwarm([fixture],peers=plans,piece_length=plen) as swarm:
            started = time.monotonic()
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
            with ThreadPoolExecutor(max_workers=1) as pool:
                task=pool.submit(range_read,server,swarm.info_hash.hex(),index,source,0,min(len(source)-1,65535))
                report['samples']=[]
                while not task.done():
                    report['samples'].append(server.json('/flow/status/'+swarm.info_hash.hex()))
                    if time.monotonic()-started>120: raise TimeoutError('head request exceeded case budget')
                    time.sleep(.5)
                report['ranges'].append(task.result())
            before=swarm.status()
            seek_start=time.monotonic()-swarm.started
            offset=urgent*plen
            report['rare_requested_before_seek']=any(urgent in p['first_request_seconds'] for p in before['peers'])
            report['ranges'].append(range_read(server,swarm.info_hash.hex(),index,source,offset,min(len(source)-1,offset+65535)))
            report['ranges'].append(range_read(server,swarm.info_hash.hex(),index,source,0,65535))
            report['swarm']=swarm.status()
            requests=[p['first_request_seconds'][urgent] for p in report['swarm']['peers'] if urgent in p['first_request_seconds']]
            report['urgent_first_request_ms_after_seek']=(min(requests)-seek_start)*1000 if requests else None
            report['flow']=server.json('/flow/status/'+swarm.info_hash.hex())
            report['elapsed_ms']=(time.monotonic()-started)*1000
            report['passed']=True
    finally:
        server.close()
    return report


if __name__=='__main__':
    parser=argparse.ArgumentParser()
    parser.add_argument('--executable',required=True,type=Path)
    parser.add_argument('--output',required=True,type=Path)
    parser.add_argument('--fixtures',type=Path)
    parser.add_argument('--cases',nargs='+',choices=CASES,default=list(CASES))
    args=parser.parse_args()
    args.output.mkdir(parents=True,exist_ok=True)
    fixtures=args.fixtures or args.output/'fixtures'
    if not args.fixtures: generate(fixtures,seconds=12)
    manifest=json.loads((fixtures/'fixtures.json').read_text(encoding='utf-8'))
    entry=next(f for f in manifest['fixtures'] if f['name']=='variable.mp4')
    result={'executable':str(args.executable.resolve()),'generated_media':True,'cases':[]}
    try:
        for label in args.cases:
            print('Sparse case: '+label,flush=True)
            result['cases'].append(run(args.executable,args.output,fixtures/'variable.mp4',label,entry['bytes']/entry['duration']))
            (args.output/'report.json').write_text(json.dumps(result,indent=2)+'\n',encoding='utf-8')
        result['passed']=True
    except Exception as error:
        result['error']=str(error)
        raise
    finally:
        (args.output/'report.json').write_text(json.dumps(result,indent=2)+'\n',encoding='utf-8')
