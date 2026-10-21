"""Real corrupted wire block, native rejection, refetch and original-byte delivery."""
import argparse
from contextlib import closing
import json
from pathlib import Path
import time
from urllib.parse import urlencode
from controlled_peer import LocalSwarm,PeerPlan
from playback_harness import OwnedServer,upload,range_read,MIB

def check(executable,output):
    output.mkdir(parents=True,exist_ok=False)
    fixture=output/'owned.bin'
    fixture.write_bytes(bytes(range(256))*32768)
    report={'passed':False,'generated_bytes_only':True}
    try:
        with closing(OwnedServer(executable,output/'state',extra_settings={'CacheSize':64*MIB,'Flow':{'DiagnosticHistory':True,'BootstrapHeadMB':4,'StartupBufferMinMB':4,'StartupBufferMaxMB':4}})) as server:
            server.ready()
            with LocalSwarm([fixture],piece_length=MIB,peers=[PeerPlan(rate=8*MIB,corrupt_first_block=True),PeerPlan(rate=8*MIB,outages=[(0,2)])],high_throughput=True) as swarm:
                info=upload(server,swarm); hashed=swarm.info_hash.hex(); index=info['file_stats'][0]['id']
                server.json('/stream/generated?'+urlencode({'link':hashed,'index':index,'preload':'','stat':''}),timeout=90)
                deadline=time.monotonic()+90
                while time.monotonic()<deadline:
                    status=server.json('/flow/status/'+hashed)
                    tor=server.json('/torrents',{'action':'get','hash':hashed})
                    if status.get('integrity',{}).get('failures',0)>0 and tor.get('preloaded_bytes',0)>=tor.get('preload_size',1)>0: break
                    time.sleep(.1)
                else: raise AssertionError('Native failure/refetch was not observed')
                report['delivery']=range_read(server,hashed,index,swarm.data,0,len(swarm.data)-1)
                report['integrity']=status['integrity']
                report['corrupted_blocks']=sum(p.get('corrupted_blocks',0) for p in swarm.status()['peers'])
                if report['corrupted_blocks']!=1: raise AssertionError('Fixture did not corrupt exactly one block')
                report['passed']=True
    except Exception as error:
        report['error_type']=type(error).__name__
    finally:
        (output/'report.json').write_text(json.dumps(report,indent=2)+'\n',encoding='utf-8')
    return report['passed']

if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--executable',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    args=parser.parse_args()
    raise SystemExit(0 if check(args.executable,args.output) else 1)
