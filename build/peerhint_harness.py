#!/usr/bin/env python3
"""Recover a useful public peer with owned tracker discovery turned off."""
import argparse
import json
from pathlib import Path
import time
from controlled_peer import LocalSwarm
from playback_harness import OwnedServer, upload, range_read, MIB
from preparation_harness import restart_after_crash

def run(executable,output):
    source=bytes((i*17+i//1024)%256 for i in range(8*MIB+19))
    episode=output/'episode.bin';episode.write_bytes(source)
    server=OwnedServer(executable,output/'state',extra_settings={'Flow':{'Enabled':True,'PeerResumeHints':True,'SwarmProfile':'conservative','BootstrapHeadMB':1,'StartupBufferMinMB':1,'StartupBufferMaxMB':1,'StartupBufferSeconds':1}})
    report={}
    try:
        server.ready()
        with LocalSwarm([episode],rate=256*1024) as swarm:
            status=upload(server,swarm);hash_text,index=status['hash'],status['file_stats'][0]['id']
            report['first_read']=range_read(server,hash_text,index,source,0,65535)
            hints=server.state/'flow-peer-hints'/(hash_text+'.json')
            deadline=time.monotonic()+12
            while not hints.exists() and time.monotonic()<deadline: time.sleep(.25)
            if not hints.exists(): raise AssertionError('useful peer was not persisted')
            value=json.loads(hints.read_text(encoding='utf-8'))
            if not value['peers'] or len(value['peers'])>32: raise AssertionError('invalid hint bounds')
            swarm.advertise_peers=False
            before=swarm.status()['connections']
            restart_after_crash(server)
            report['restored_read']=range_read(server,hash_text,index,source,4*MIB,4*MIB+65535)
            if swarm.status()['connections']<=before: raise AssertionError('peer connection was not restored')
            report['new_connections']=swarm.status()['connections']-before
            report['passed']=True
    finally:
        server.close();(output/'report.json').write_text(json.dumps(report,indent=2),encoding='utf-8')
    return report

if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--executable',required=True,type=Path);parser.add_argument('--output',required=True,type=Path);args=parser.parse_args()
    args.output.mkdir(parents=True,exist_ok=True);print(json.dumps(run(args.executable,args.output),indent=2))
