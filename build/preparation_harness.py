#!/usr/bin/env python3
"""Real retained-piece preparation, cancellation, quota and crash recovery."""
import argparse
import json
import http.client
import os
from pathlib import Path
import subprocess
import socket
import time
import urllib.error

from controlled_peer import LocalSwarm
from playback_harness import OwnedServer, upload, range_read, MIB

def wait_job(server, state, timeout=120):
    end=time.monotonic()+timeout
    while time.monotonic()<end:
        status=server.json('/flow/preparation')
        if status.get('error_code'): raise AssertionError(status)
        jobs=status['jobs']
        if jobs and jobs[0]['state']==state: return jobs[0]
        if jobs and jobs[0]['state']=='error': raise AssertionError(jobs[0])
        time.sleep(.25)
    raise TimeoutError('preparation state '+state)

def restart_after_crash(server, settings_update=None):
    args=server.process.args
    server.process.kill();server.process.wait(10)
    if settings_update:
        path=server.state/'settings.json'
        config=json.loads(path.read_text(encoding='utf-8'))
        config['BitTorr'].update(settings_update)
        path.write_text(json.dumps(config),encoding='utf-8')
    flags=subprocess.CREATE_NO_WINDOW if os.name=='nt' else 0
    server.process=subprocess.Popen(args,stdout=server.log,stderr=subprocess.STDOUT,creationflags=flags)
    server.started=time.monotonic();server.ready()

def run(executable, output):
    source=bytes((i*17+i//1024)%256 for i in range(6*MIB+19))
    episode=output/'episode.bin';episode.write_bytes(source)
    server=OwnedServer(executable,output/'state',extra_settings={'CacheSize':MIB,'Flow':{'PreparationQuotaMB':64,'Enabled':True,'SwarmProfile':'conservative'}})
    report={}
    try:
        server.ready()
        with LocalSwarm([episode],rate=2*MIB) as swarm:
            status=upload(server,swarm);hash_text=status['hash'];index=status['file_stats'][0]['id']
            request={'hash':hash_text,'file_index':index,'action':'start'}
            server.json('/flow/preparation',request)
            server.json('/flow/preparation',{**request,'action':'pause'})
            wait_job(server,'paused')
            server.json('/flow/preparation',{**request,'action':'cancel'})
            wait_job(server,'cancelled')
            server.json('/flow/preparation',{**request,'action':'resume'})
            ready=wait_job(server,'ready');report['ready']=ready
            if ready['verified_bytes']!=len(source) or not ready['playback_ready']: raise AssertionError('premature readiness')
            # More than cache capacity retained in the same piece store.
            report['read']=range_read(server,hash_text,index,source,MIB,2*MIB-1)
            restart_after_crash(server)
            report['crash_recovery']=wait_job(server,'ready')
            server.process.kill();server.process.wait(10)
            piece=server.state/'flow-pieces'/hash_text/'0'
            if not piece.exists(): raise AssertionError('authoritative prepared piece missing')
            piece.write_bytes(b'corrupt')
            flags=subprocess.CREATE_NO_WINDOW if os.name=='nt' else 0
            server.process=subprocess.Popen(server.process.args,stdout=server.log,stderr=subprocess.STDOUT,creationflags=flags)
            server.started=time.monotonic();server.ready()
            report['corruption_repair']=wait_job(server,'ready')
            report['repaired_read']=range_read(server,hash_text,index,source,0,65535)
        # Verified data remains playable without the supplier or tracker.
        restart_after_crash(server)
        report['offline_resume']=wait_job(server,'ready')
        report['offline_read']=range_read(server,hash_text,index,source,len(source)-65536,len(source)-1)
        restart_after_crash(server,{'UseDisk':True,'TorrentsSavePath':str(server.state/'changed-disk-root')})
        report['storage_root_recovery']=wait_job(server,'ready')
        report['root_change_read']=range_read(server,hash_text,index,source,0,65535)
        # Hold the receive window small so the native reader remains owned while
        # explicit cleanup waits. These are our own connection and state files.
        player=http.client.HTTPConnection('127.0.0.1',server.port,timeout=10)
        try:
            player.connect();player.sock.setsockopt(socket.SOL_SOCKET,socket.SO_RCVBUF,4096)
            player.request('GET',f'/play/{hash_text}/{index}?play&stat=cleanup-test',headers={'Range':f'bytes=0-{len(source)-1}'})
            response=player.getresponse()
            if response.status!=206 or response.read(1)!=source[:1]: raise AssertionError('cleanup player failed')
            server.json('/flow/preparation',{**request,'action':'remove'})
            end=time.monotonic()+8
            while time.monotonic()<end:
                jobs=server.json('/flow/preparation')['jobs']
                if jobs and jobs[0].get('error_code')=='PLAYBACK_ACTIVE':break
                time.sleep(.1)
            else:raise AssertionError('cleanup did not preserve the active reader')
            if not (server.state/'flow-pieces'/hash_text/'0').exists():raise AssertionError('cleanup removed active playback data')
            try: server.json('/flow/preparation',{**request,'action':'resume'})
            except urllib.error.HTTPError as error:
                if error.code!=409:raise
            else:raise AssertionError('resume reversed native cleanup')
            report['active_cleanup_fenced']=True
        finally:
            player.close()
        deadline=time.monotonic()+20
        while server.json('/flow/preparation')['jobs'] and time.monotonic()<deadline: time.sleep(.25)
        if server.json('/flow/preparation')['jobs']: raise AssertionError('cleanup did not release reservation')
        if any((server.state/'flow-pieces'/hash_text).glob('[0-9]*')): raise AssertionError('cleanup left prepared files')
        oversized=output/'too-large.bin';oversized.write_bytes(source*11)
        with LocalSwarm([oversized]) as swarm:
            big=upload(server,swarm)
            try: server.json('/flow/preparation',{'hash':big['hash'],'file_index':big['file_stats'][0]['id'],'action':'start'})
            except urllib.error.HTTPError as err:
                if err.code!=409: raise
            else: raise AssertionError('disk quota accepted oversized preparation')
        report['quota_rejected']=True
        # A later job must migrate a populated ordinary disk cache to the
        # retained manager root, and library deletion must stop its job too.
        with LocalSwarm([episode]) as swarm:
            item=upload(server,swarm)
            request={'hash':item['hash'],'file_index':item['file_stats'][0]['id']}
            range_read(server,item['hash'],request['file_index'],source,0,65535)
            ordinary=server.state/'changed-disk-root'/item['hash']/'0'
            if not ordinary.exists():raise AssertionError('ordinary disk backend was not exercised')
            server.json('/flow/preparation',{**request,'action':'start'})
            wait_job(server,'ready')
            retained=server.state/'flow-pieces'/item['hash']/'0'
            if not retained.exists() or ordinary.exists():raise AssertionError('existing disk backend did not move to retained root')
            report['new_job_disk_migration']=True
            player=http.client.HTTPConnection('127.0.0.1',server.port,timeout=10)
            try:
                player.connect();player.sock.setsockopt(socket.SOL_SOCKET,socket.SO_RCVBUF,4096)
                player.request('GET',f"/play/{item['hash']}/{request['file_index']}?play&stat=library-cleanup",headers={'Range':f'bytes=0-{len(source)-1}'})
                response=player.getresponse()
                if response.status!=206 or response.read(1)!=source[:1]:raise AssertionError('library cleanup player failed')
                with server.request('/torrents',{'action':'rem','hash':item['hash']},timeout=10) as removed:
                    if removed.status!=200 or removed.read()!=b'':
                        raise AssertionError('library deletion response contract changed')
                end=time.monotonic()+8
                while time.monotonic()<end:
                    jobs=server.json('/flow/preparation')['jobs']
                    if jobs and jobs[0].get('error_code')=='PLAYBACK_ACTIVE':break
                    time.sleep(.1)
                else:raise AssertionError('library deletion bypassed preparation reader fence')
                if not retained.exists():raise AssertionError('library deletion removed active data')
            finally:player.close()
            end=time.monotonic()+20
            while server.json('/flow/preparation')['jobs'] and time.monotonic()<end:time.sleep(.25)
            if server.json('/flow/preparation')['jobs']:raise AssertionError('library cleanup retained jobs')
            time.sleep(2)
            if any(t['hash']==item['hash'] for t in server.json('/torrents',{'action':'list'})):raise AssertionError('deleted preparation resurrected its torrent')
            report['library_cleanup_fenced']=True
        broken=OwnedServer(executable,output/'disk-failure',seed_files={'flow-pieces':b'not a directory'})
        try:
            broken.ready()
            with LocalSwarm([episode]) as swarm:
                item=upload(broken,swarm)
                try:broken.json('/flow/preparation',{'hash':item['hash'],'file_index':item['file_stats'][0]['id'],'action':'start'})
                except urllib.error.HTTPError as error:
                    if error.code!=409:raise
                else:raise AssertionError('unwritable preparation storage accepted')
                job=broken.json('/flow/preparation')['jobs'][0]
                if job['state']!='error' or job['playback_ready']:raise AssertionError('disk failure claimed readiness')
                report['disk_failure_rejected']=True
        finally:broken.close()
        report['passed']=True
    except Exception as error:
        report['passed']=False
        report['error']=str(error)
        raise
    finally:
        server.close();(output/'report.json').write_text(json.dumps(report,indent=2),encoding='utf-8')
    return report

if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--executable',type=Path,required=True);parser.add_argument('--output',type=Path,required=True);args=parser.parse_args()
    args.output.mkdir(parents=True,exist_ok=True);print(json.dumps(run(args.executable,args.output),indent=2))
