#!/usr/bin/env python3
"""Owned HTTP Range mirrors: approval, exact bytes, redirects and peer fallback."""
import argparse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import threading
import time
import urllib.error

from controlled_peer import LocalSwarm, PeerPlan, bencode
from playback_harness import OwnedServer, upload, range_read, MIB
from preparation_harness import restart_after_crash


class Mirror:
    def __init__(self, data, mode):
        self.requests = []
        owner = self
        class Handler(BaseHTTPRequestHandler):
            protocol_version = 'HTTP/1.1'
            def do_GET(self):
                owner.requests.append({'range': self.headers.get('Range'), 'redirect': self.path.startswith('/redirect/')})
                if self.path.startswith('/redirect/'):
                    self.send_response(302)
                    self.send_header('Location', '/data/'+self.path[len('/redirect/'):])
                    self.send_header('Content-Length', '0')
                    self.end_headers()
                    return
                if mode == 'unreachable':
                    self.send_response(503)
                    self.send_header('Retry-After', '30')
                    self.send_header('Content-Length', '0')
                    self.end_headers()
                    return
                raw = self.headers.get('Range', 'bytes=0-').removeprefix('bytes=').split('-')
                first, last = int(raw[0]), int(raw[1]) if raw[1] else len(data)-1
                last = min(last, len(data)-1)
                body = data[first:last+1]
                if mode == 'mismatch': body = bytes(b ^ 255 for b in body)
                self.send_response(206)
                self.send_header('Content-Range', f'bytes {first}-{last}/{len(data)}')
                self.send_header('Content-Length', str(len(body)))
                self.end_headers()
                try: self.wfile.write(body)
                except (BrokenPipeError, ConnectionResetError): pass
            def log_message(self, *_): pass
        self.server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        self.server.daemon_threads = True
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
    def __enter__(self):
        self.thread.start()
        return self
    def __exit__(self, *_):
        self.server.shutdown(); self.server.server_close(); self.thread.join(2)
    def url(self, redirect=False, hostname='127.0.0.1'):
        return f'http://{hostname}:{self.server.server_port}/'+('redirect/' if redirect else 'data/')


def run(executable, output):
    source = bytes((i*17+i//1024)%256 for i in range(3*MIB+19))
    episode = output/'episode.bin'; episode.write_bytes(source)
    report = {'cases':[], 'http_delivery_only':True}
    for mode in ('identical', 'redirect', 'mismatch', 'unreachable', 'blocked-dns'):
        case = {'case':mode}
        server = OwnedServer(executable, output/mode, extra_settings={'EnableDebug':True})
        try:
            server.ready()
            with Mirror(source, mode) as mirror, LocalSwarm([episode], peers=[PeerPlan(choke_until=6 if mode in ('mismatch','unreachable','blocked-dns') else 300)]) as swarm:
                # Imported local metadata cannot authorize reaching the LAN.
                tracker = f'http://127.0.0.1:{swarm.tracker.server_port}/announce'
                torrent = bencode({b'announce':tracker,b'info':swarm.info,b'url-list':[mirror.url(mode=='redirect', 'localhost' if mode=='blocked-dns' else '127.0.0.1')]})
                original = swarm.torrent
                swarm.torrent = lambda: torrent
                status = upload(server,swarm)
                swarm.torrent = original
                hash_text,index = status['hash'],status['file_stats'][0]['id']
                time.sleep(1.5)
                if mirror.requests: raise AssertionError('imported URL accessed local network without permission')
                endpoint='/flow/sources/'+hash_text
                try: server.json(endpoint,{'action':'add','url':mirror.url()})
                except urllib.error.HTTPError as error:
                    if error.code != 409: raise
                else: raise AssertionError('local URL accepted without approval')
                if mode!='blocked-dns':
                    server.json(endpoint,{'action':'add','url':mirror.url(mode=='redirect'),'allow_local':True})
                sources=server.json(endpoint)
                if any('/data' in s['origin'] or '/redirect' in s['origin'] for s in sources['sources']): raise AssertionError('source path leaked')
                case['head']=range_read(server,hash_text,index,source,0,65535)
                case['tail']=range_read(server,hash_text,index,source,len(source)-65536,len(source)-1)
                if mode=='blocked-dns':
                    if mirror.requests: raise AssertionError('DNS-resolved LAN mirror bypassed the destination guard')
                elif not mirror.requests: raise AssertionError('native mirror path was not exercised')
                if mode=='redirect' and not any(r['redirect'] for r in mirror.requests): raise AssertionError('redirect not exercised')
                if mode in ('mismatch','unreachable','blocked-dns') and swarm.status()['sent_bytes']==0: raise AssertionError('peer fallback not exercised')
                if len(mirror.requests)>24: raise AssertionError('unbounded mirror retries')
                case['mirror_requests']=mirror.requests
                case['swarm']=swarm.status()
                # API management exposes an opaque ID and origin only.
                for item in sources['sources']:
                    server.json(endpoint,{'action':'remove','id':item['id']})
                if not all(s['disabled'] for s in server.json(endpoint)['sources']): raise AssertionError('disable did not persist')
                restart_after_crash(server)
                if not all(s['disabled'] for s in server.json(endpoint)['sources']): raise AssertionError('imported source returned after restart')
                case['disabled_after_restart']=True
                case['passed']=True
        finally:
            server.close()
            report['cases'].append(case)
            (output/'report.json').write_text(json.dumps(report,indent=2),encoding='utf-8')
    report['passed']=True
    (output/'report.json').write_text(json.dumps(report,indent=2),encoding='utf-8')
    return report


if __name__=='__main__':
    parser=argparse.ArgumentParser()
    parser.add_argument('--executable',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    args=parser.parse_args();args.output.mkdir(parents=True,exist_ok=True)
    print(json.dumps(run(args.executable,args.output),indent=2))
