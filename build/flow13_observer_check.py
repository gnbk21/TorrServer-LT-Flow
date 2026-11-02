#!/usr/bin/env python3
"""Rotated high-bitrate observer overhead checks on fresh loopback state.

Uses the existing synthetic transport fixture and paced byte verifier. Compares
the same candidate/settings with presentation/monitoring/selected sidecars on
and off; it does not replace the published-1.2.0 baseline comparison.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
from pathlib import Path
import statistics
import time
import urllib.error

from controlled_peer import LocalSwarm, PeerPlan
from high_bitrate_check import paced
from playback_harness import OwnedServer, upload, MIB


def trial(executable, fixture, output, observed, case, seconds):
    output.mkdir(parents=True, exist_ok=False)
    text = b'1\n00:00:01,000 --> 00:00:02,000\n' + b'caption '*4096 + b'\n'
    subtitle = output/(fixture.stem+'.en.srt')
    subtitle.write_bytes(text)
    rate = 120_000_000//8
    settings = {'CacheSize':512*MIB, 'Flow':{
        'Enabled':True, 'SwarmProfile':'legacy', 'MetricsEnabled':True,
        'PrometheusEnabled':observed, 'RangeTraceEnabled':False,
        'MetadataFallback':False, 'PublicMetadataCache':False,
        'BootstrapHeadMB':4, 'StartupBufferMinMB':32, 'StartupBufferMaxMB':32,
        'StartupBufferSeconds':2, 'GlobalCacheBudgetMB':512,
        'StreamTransportBufferKiB':1024, 'WarmSessionTimeoutSec':30}}
    server = OwnedServer(executable,output/'state',extra_settings=settings)
    report={'observed':observed,'case':case,'seconds':seconds,'subtitles':[],
            'samples':[],'passed':False,'server_delivery_only':True}
    try:
        report['ready_ms']=server.ready()
        # Include identical metadata/bytes in both modes. Only explicit selected
        # reads and observational requests differ between the paired trials.
        with LocalSwarm([fixture,subtitle],peers=[PeerPlan(rate=2*rate)],
                        piece_length=4*MIB,high_throughput=True) as swarm:
            info=upload(server,swarm)
            video=next(row for row in info['file_stats'] if Path(row['path']).name==fixture.name)['id']
            track=next(row for row in info['file_stats'] if Path(row['path']).name==subtitle.name)['id']
            identity=info['hash']
            start=time.monotonic()
            server.json(f'/stream/generated?link={identity}&index={video}&preload=&stat=',timeout=90)
            report['preload_ms']=(time.monotonic()-start)*1000
            media=swarm.data[:fixture.stat().st_size]
            start=time.monotonic()
            with ThreadPoolExecutor(max_workers=2) as pool:
                video_task=pool.submit(paced,server,identity,video,media,rate,seconds,case=='bursts')
                sidecar_task=None
                next_subtitle,next_scrape=0,0
                while not video_task.done():
                    elapsed=time.monotonic()-start
                    sample=server.json('/runtime/status')
                    report['samples'].append({'elapsed_seconds':elapsed,'memory':sample['memory'],'resources':sample['resources']})
                    if observed:
                        coverage=server.json(f'/flow/status/{identity}?coverage=true')
                        assert len(coverage['coverage']['pieces'])<=256
                        if elapsed>=next_scrape:
                            with server.request('/metrics') as response:
                                assert b'flow_read_wait_seconds_total' in response.read()
                            next_scrape=elapsed+15
                        if sidecar_task is not None and sidecar_task.done():
                            try:
                                data=sidecar_task.result()
                                assert data==text,'Subtitle byte mismatch'
                                report['subtitles'].append({'elapsed_seconds':elapsed,'status':200})
                            except urllib.error.HTTPError as error:
                                assert error.code==503, error.code
                                report['subtitles'].append({'elapsed_seconds':elapsed,'status':503})
                            sidecar_task=None
                        if sidecar_task is None and elapsed>=next_subtitle:
                            def fetch():
                                with server.request(f'/subs/{identity}/{video}?track={track}&format=original',timeout=12) as response:
                                    return response.read()
                            sidecar_task=pool.submit(fetch)
                            next_subtitle=elapsed+5
                    time.sleep(1)
                report['delivery']=video_task.result()
                if sidecar_task is not None:
                    try:
                        assert sidecar_task.result()==text
                        report['subtitles'].append({'elapsed_seconds':time.monotonic()-start,'status':200,'after_video':True})
                    except urllib.error.HTTPError as error:
                        assert error.code==503
                        report['subtitles'].append({'status':503,'after_video':True})
            if observed:
                assert any(row['status']==200 and not row.get('after_video') for row in report['subtitles']), 'Every active subtitle fetch was deferred'
            report['peak_rss_bytes']=max(row['memory'].get('rss_bytes',0) for row in report['samples'])
            report['peer']=swarm.status()
            report['passed']=True
    except Exception as error:
        report['error']=f'{type(error).__name__}: {error}'
    finally:
        server.close()
        (output/'report.json').write_text(json.dumps(report,indent=2)+'\n',encoding='utf-8')
    return report


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--executable',required=True,type=Path)
    parser.add_argument('--fixture',required=True,type=Path)
    parser.add_argument('--output',required=True,type=Path)
    parser.add_argument('--seconds',type=int,default=30,choices=range(20,61))
    parser.add_argument('--rotations',type=int,default=2,choices=range(2,6))
    parser.add_argument('--cases',nargs='+',choices=['healthy','bursts'],default=['healthy','bursts'])
    args=parser.parse_args()
    args.output.mkdir(parents=True,exist_ok=False)
    report={'schema_version':1,'candidate_sha256':hashlib.sha256(args.executable.read_bytes()).hexdigest(),
            'fixture_sha256':hashlib.sha256(args.fixture.read_bytes()).hexdigest(),'rotations':args.rotations,'runs':[]}
    for case in args.cases:
        for repeat in range(args.rotations):
            for observed in ([False,True] if repeat%2==0 else [True,False]):
                print(case,repeat,'observed' if observed else 'control',flush=True)
                row=trial(args.executable,args.fixture,args.output/f'{case}-{repeat}-{int(observed)}',observed,case,args.seconds)
                report['runs'].append(row)
                (args.output/'report.json').write_text(json.dumps(report,indent=2)+'\n',encoding='utf-8')
    report['passed']=all(row['passed'] for row in report['runs'])
    report['medians']={case:{str(observed):{
        'peak_rss_bytes':statistics.median(row['peak_rss_bytes'] for row in report['runs'] if row['case']==case and row['observed']==observed and row['passed']),
        'read_wait_ms':statistics.median(row['delivery']['total_read_wait_ms'] for row in report['runs'] if row['case']==case and row['observed']==observed and row['passed'])}
        for observed in [False,True] if any(row['case']==case and row['observed']==observed and row['passed'] for row in report['runs'])} for case in args.cases}
    (args.output/'report.json').write_text(json.dumps(report,indent=2)+'\n',encoding='utf-8')
    return report['passed']


if __name__=='__main__':
    raise SystemExit(0 if main() else 1)
