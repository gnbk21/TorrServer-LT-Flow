#!/usr/bin/env python3
"""Short native TCP loss/jitter/reordering/outage regression in an owned namespace.

Requires Linux, iproute2 and CAP_SYS_ADMIN/CAP_NET_ADMIN (CI uses sudo).
No host qdisc, route, interface or public swarm is modified. Kernel scheduling
means a fixed PRNG seed is repeatable input, not bit-for-bit timing evidence.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import threading
import time

from controlled_peer import LocalSwarm, PeerPlan
from playback_harness import OwnedServer, upload, range_read, MIB


def command(*args):
    return subprocess.check_output(args, text=True, stderr=subprocess.STDOUT, timeout=5)


def resources(pid):
    status = dict(line.split(':', 1) for line in Path(f'/proc/{pid}/status').read_text().splitlines() if ':' in line)
    return dict(rss_bytes=int(status['VmRSS'].split()[0])*1024,
                threads=int(status['Threads']), descriptors=len(list(Path(f'/proc/{pid}/fd').iterdir())))


SEED_SUPPORTED = True

def shaping(*options):
    global SEED_SUPPORTED
    argv=['tc','qdisc','replace','dev','lo','root','netem','limit','10000',*options]
    if SEED_SUPPORTED:
        try:
            command(*argv,'seed','147115');return
        except subprocess.CalledProcessError:
            # Older hosted Ubuntu iproute2/kernel pairs have no seed option.
            # Require the same qdisc to work, and disclose the weaker evidence.
            command(*argv);SEED_SUPPORTED=False;return
    command(*argv)


def run(args):
    if (not args.parent_namespace or os.readlink('/proc/self/ns/net') == args.parent_namespace
            or os.readlink(f'/proc/{os.getppid()}/ns/net') != args.parent_namespace
            or os.readlink('/proc/self/ns/net') == os.readlink('/proc/1/ns/net')):
        raise RuntimeError('Refusing qdisc changes outside the newly owned namespace')
    command('ip', 'link', 'set', 'lo', 'up')
    args.output.mkdir(parents=True, exist_ok=False)
    # Generated exact bytes; direct Range integrity, not a decoder/media test.
    source = hashlib.shake_256(b'Flow owned network impairment fixture').digest(32*MIB)
    fixture = args.output/'generated.mkv'; fixture.write_bytes(source)
    report = dict(executable_sha256=hashlib.sha256(args.executable.read_bytes()).hexdigest(),
                  fixture_sha256=hashlib.sha256(source).hexdigest(), seed=147115,
                  namespace=os.readlink('/proc/self/ns/net'), tcp_only=True,
                  http_delivery_only=True, stream_controls=args.stream_controls,
                  passed=False, cases=[])
    try:
        for case in ('jitter', 'loss', 'reordering', 'outage'):
            server = OwnedServer(args.executable, args.output/case, extra_settings={
                'CacheSize':8*MIB,'DisableUTP':True,'DisableTCP':False,
                'Flow':{'Enabled':True,'SwarmProfile':'legacy' if args.stream_controls else 'adaptive',
                        'CapacityAwareRequests':args.stream_controls,
                        'FrontierRecovery':args.stream_controls,
                        'AdaptivePeerRequests':args.stream_controls,'PublicMetadataCache':False}})
            timer=None
            try:
                server.ready()
                peers = ([PeerPlan(rate=2*MIB, request_queue=64, strict_request_queue=True),
                          PeerPlan(rate=2*MIB, request_queue=64, strict_request_queue=True,
                                   request_latency_ms=100)] if args.stream_controls
                         else [PeerPlan(rate=4*MIB)])
                with LocalSwarm([fixture],peers=peers,high_throughput=True) as swarm:
                    status=upload(server,swarm);index=status['file_stats'][0]['id']
                    range_read(server,swarm.info_hash.hex(),index,source,0,65535)
                    before=resources(server.process.pid)
                    if case=='jitter': shaping('delay','20ms','10ms','distribution','normal')
                    elif case=='loss': shaping('delay','10ms','loss','random','1%')
                    elif case=='reordering': shaping('delay','10ms','reorder','25%','50%')
                    else:
                        shaping('loss','100%')
                        timer=threading.Timer(2,lambda:shaping('delay','5ms'));timer.start()
                    began=time.monotonic();samples=[]
                    with ThreadPoolExecutor(max_workers=1) as pool:
                        future=pool.submit(range_read,server,swarm.info_hash.hex(),index,source,16*MIB,17*MIB-1)
                        while not future.done():
                            samples.append(resources(server.process.pid))
                            if time.monotonic()-began>45: raise TimeoutError('netem recovery exceeded 45 seconds')
                            time.sleep(.1)
                        delivery=future.result()
                    elapsed=time.monotonic()-began
                    if case=='outage' and elapsed<1.5: raise AssertionError('outage did not exercise an uncached request')
                    impaired_qdisc=json.loads(command('tc','-j','-s','qdisc','show','dev','lo'))
                    shaping('delay','0ms')
                    # A second uncached range must still work after recovery.
                    recovered=range_read(server,swarm.info_hash.hex(),index,source,28*MIB,28*MIB+65535)
                    after=resources(server.process.pid)
                    peak={key:max(row[key] for row in [before,after,*samples]) for key in before}
                    if peak['rss_bytes']>MIB*1024 or peak['descriptors']>1024 or peak['threads']>256:
                        raise AssertionError('bounded fixture exceeded its resource envelope')
                    evidence=server.json('/flow/status/'+swarm.info_hash.hex()).get('sparse',{}).get('stream')
                    if args.stream_controls:
                        if not evidence or not evidence.get('known'):
                            raise AssertionError('Native stream control observations unavailable')
                        if (evidence['peak_hedges']>16 or evidence['active_hedge_bytes']>256*1024
                                or any(p['peak_pending_requests']>64 or p['capacity_overflows']
                                       for p in swarm.status()['peers'])):
                            raise AssertionError('Stream controls exceeded duplicate or peer capacity')
                    report['cases'].append(dict(case=case,elapsed_ms=elapsed*1000,delivery=delivery,
                        recovered=recovered,before=before,after=after,peak=peak,
                        stream_control=evidence,
                        qdisc=impaired_qdisc,passed=True))
            finally:
                try:
                    if timer is not None:timer.join(5)
                    shaping('delay','0ms')
                finally:
                    server.close()
        report['seed_supported']=SEED_SUPPORTED
        report['passed']=True
    finally:
        (args.output/'report.json').write_text(json.dumps(report,indent=2)+'\n')


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--executable',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--parent-namespace')
    parser.add_argument('--stream-controls',action='store_true')
    args=parser.parse_args()
    args.executable=args.executable.resolve();args.output=args.output.resolve()
    if sys.platform!='linux':parser.error('Linux network namespaces are required')
    if args.parent_namespace:
        run(args)
    else:
        argv=['unshare','--net',sys.executable,str(Path(__file__).resolve()),'--executable',str(args.executable),
              '--output',str(args.output),'--parent-namespace',os.readlink('/proc/self/ns/net')]
        if args.stream_controls: argv.append('--stream-controls')
        child=subprocess.Popen(argv,start_new_session=True)
        try:
            code=child.wait(180)
        except subprocess.TimeoutExpired:
            os.killpg(child.pid,signal.SIGKILL);child.wait();raise
        raise SystemExit(code)
