#!/usr/bin/env python3
"""A local BitTorrent seeder/tracker for generated fixtures and failure tests.

Implements the ordinary v1 handshake, bitfield, unchoke and request/piece wire
messages. The server under test must download real hash-verified pieces through
libtorrent. No production mock cache or external swarm participates.
"""
import hashlib
from dataclasses import dataclass, field
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import socket
import socketserver
import select
import struct
import threading
import time
from urllib.parse import unquote_to_bytes


def bencode(value):
    if isinstance(value, int):
        return b"i" + str(value).encode() + b"e"
    if isinstance(value, str):
        value = value.encode()
    if isinstance(value, bytes):
        return str(len(value)).encode() + b":" + value
    if isinstance(value, list):
        return b"l" + b"".join(map(bencode, value)) + b"e"
    if isinstance(value, dict):
        return b"d" + b"".join(bencode(k) + bencode(v) for k, v in sorted(value.items())) + b"e"
    raise TypeError(type(value))


def receive(sock, length, stop):
    data = bytearray()
    while len(data) < length:
        if stop.is_set():
            raise EOFError("stopped")
        try:
            part = sock.recv(length - len(data))
        except socket.timeout:
            continue
        if not part:
            raise EOFError("peer disconnected")
        data.extend(part)
    return bytes(data)


def bdecode(data):
    """Decode a wire dictionary and return the offset of its binary suffix."""
    def decode(pos):
        token = data[pos:pos+1]
        if token == b"i":
            end = data.index(b"e", pos)
            return int(data[pos+1:end]), end+1
        if token in (b"l", b"d"):
            values, pos = [], pos+1
            while data[pos:pos+1] != b"e":
                value, pos = decode(pos)
                values.append(value)
            if token == b"d":
                return dict(zip(values[::2], values[1::2])), pos+1
            return values, pos+1
        end = data.index(b":", pos)
        size = int(data[pos:end])
        pos = end+1
        if size < 0 or pos+size > len(data):
            raise ValueError("truncated bencode")
        return data[pos:pos+size], pos+size
    return decode(0)


@dataclass
class PeerPlan:
    """Times are relative to swarm start, so reconnects do not reset outages."""
    pieces: set | None = None
    rate: int = 0
    delay_ms: int = 0
    choke_until: float = 0
    have_events: list = field(default_factory=list)  # (seconds, piece)
    outages: list = field(default_factory=list)  # (start, end) in seconds
    metadata_delay: float = 0
    disconnect_after: int = 0

    def available(self, count, elapsed):
        pieces = set(range(count)) if self.pieces is None else set(self.pieces)
        pieces.update(piece for when, piece in self.have_events if elapsed >= when)
        return pieces

    def online(self, elapsed):
        return not any(start <= elapsed < end for start, end in self.outages)


class LocalSwarm:
    def __init__(self, paths, rate=0, delay_ms=0, disconnect_after=0,
                 peers=None, piece_length=256*1024, private=False):
        self.paths = paths
        self.advertise_peers = True
        self.data = b"".join(p.read_bytes() for p in paths)
        if piece_length <= 0:
            raise ValueError("piece_length must be positive")
        self.piece_length = piece_length
        pieces = b"".join(hashlib.sha1(self.data[i:i+self.piece_length]).digest()
                          for i in range(0, len(self.data), self.piece_length))
        self.info = {b"name": b"Flow-generated-fixtures", b"piece length": self.piece_length,
                     b"pieces": pieces, b"files": [{b"length": p.stat().st_size, b"path": [p.name.encode()]} for p in paths]}
        if private:
            self.info[b"private"] = 1
        self.metadata = bencode(self.info)
        self.info_hash = hashlib.sha1(bencode(self.info)).digest()
        self.piece_count = len(pieces) // 20
        self.rate, self.delay = rate, delay_ms / 1000
        self.disconnect_after = disconnect_after
        self.plans = peers if peers is not None else [PeerPlan(rate=rate, delay_ms=delay_ms, disconnect_after=disconnect_after)]
        if not 1 <= len(self.plans) <= 32:
            raise ValueError("fixture requires 1..32 peers")
        for plan in self.plans:
            if plan.rate < 0 or plan.delay_ms < 0 or plan.choke_until < 0 or plan.metadata_delay < 0:
                raise ValueError("negative peer timing or rate")
            if any(p < 0 or p >= self.piece_count for p in plan.available(self.piece_count, float('inf'))):
                raise ValueError("piece outside generated torrent")
            if any(start < 0 or end <= start for start, end in plan.outages):
                raise ValueError("invalid outage interval")
        self.started = None
        self.peer_stats = [dict(requests=0, sent_bytes=0, connections=0, disconnects=0, requested_pieces=[], first_request_seconds={}) for _ in self.plans]
        self.stop = threading.Event()
        self.lock = threading.Lock()
        self.sent_bytes = self.connections = self.requests = self.disconnects = self.announces = 0
        owner = self

        class Peer(socketserver.BaseRequestHandler):
            def handle(self):
                sock = self.request
                sock.settimeout(.2)
                plan, index = self.server.plan, self.server.index
                stats = owner.peer_stats[index]
                elapsed = lambda: time.monotonic() - owner.started
                if not plan.online(elapsed()):
                    return
                with owner.lock:
                    owner.connections += 1
                    stats['connections'] += 1
                try:
                    handshake = receive(sock, 68, owner.stop)
                    if handshake[:20] != b"\x13BitTorrent protocol" or handshake[28:48] != owner.info_hash:
                        return
                    reserved = bytearray(8)
                    reserved[5] = 0x10  # BEP 10 extended handshake
                    sock.sendall(b"\x13BitTorrent protocol" + reserved + owner.info_hash + f"-FL0002-{index:012d}".encode())
                    def send(message):
                        sock.sendall(struct.pack("!I", len(message)) + message)
                    available = plan.available(owner.piece_count, elapsed())
                    bits = bytearray((owner.piece_count + 7) // 8)
                    for piece in available:
                        bits[piece // 8] |= 0x80 >> (piece % 8)
                    send(b"\x05" + bits)
                    choked = elapsed() < plan.choke_until
                    send(b"\x00" if choked else b"\x01")
                    send(b"\x14\x00" + bencode({b'm': {b'ut_metadata': 1}, b'metadata_size': len(owner.metadata)}))
                    incoming, pending, metadata_requests = bytearray(), [], []
                    metadata_id, next_send = 0, time.monotonic()
                    while not owner.stop.is_set():
                        if not plan.online(elapsed()):
                            with owner.lock:
                                owner.disconnects += 1
                                stats['disconnects'] += 1
                            return
                        new = plan.available(owner.piece_count, elapsed())
                        for piece in sorted(new - available):
                            send(b'\x04' + struct.pack('!I', piece))
                        available = new
                        if choked and elapsed() >= plan.choke_until:
                            choked = False
                            send(b'\x01')
                        if metadata_id and elapsed() >= plan.metadata_delay:
                            while metadata_requests:
                                piece = metadata_requests.pop(0)
                                block = owner.metadata[piece*16384:(piece+1)*16384]
                                header = {b'msg_type': 1 if block else 2, b'piece': piece}
                                if block:
                                    header[b'total_size'] = len(owner.metadata)
                                send(b'\x14' + bytes([metadata_id]) + bencode(header) + block)
                        now = time.monotonic()
                        if pending and not choked and now >= pending[0][0]:
                            _, piece, begin, count = pending.pop(0)
                            block = owner.data[piece*owner.piece_length+begin:piece*owner.piece_length+begin+count]
                            send(b'\x07' + struct.pack('!II', piece, begin) + block)
                            with owner.lock:
                                owner.sent_bytes += count
                                stats['sent_bytes'] += count
                        if not select.select([sock], [], [], .01)[0]:
                            continue
                        data = sock.recv(65536)
                        if not data:
                            return
                        incoming.extend(data)
                        while len(incoming) >= 4:
                            length, = struct.unpack('!I', incoming[:4])
                            if length > 128*1024:
                                return
                            if len(incoming) < length+4:
                                break
                            packet = bytes(incoming[4:4+length])
                            del incoming[:4+length]
                            if not packet:
                                continue
                            if packet[0] == 20 and len(packet) >= 3:
                                payload, _ = bdecode(packet[2:])
                                if packet[1] == 0:
                                    metadata_id = int(payload.get(b'm', {}).get(b'ut_metadata', 0))
                                    if not 0 <= metadata_id <= 255:
                                        return
                                elif packet[1] == 1 and payload.get(b'msg_type') == 0:
                                    if len(metadata_requests) >= 128:
                                        return
                                    piece = int(payload[b'piece'])
                                    if piece < 0:
                                        return
                                    metadata_requests.append(piece)
                            elif packet[0] in (6, 8) and len(packet) == 13:
                                piece, begin, count = struct.unpack('!III', packet[1:])
                                if packet[0] == 8:
                                    pending[:] = [r for r in pending if r[1:] != (piece, begin, count)]
                                    continue
                                if choked or piece not in available:
                                    continue
                                expected = min(owner.piece_length, len(owner.data)-piece*owner.piece_length)
                                if count == 0 or count > 32768 or begin+count > expected or len(pending) >= 2048:
                                    return
                                with owner.lock:
                                    owner.requests += 1
                                    stats['requests'] += 1
                                    if len(stats['requested_pieces']) < 2048:
                                        stats['requested_pieces'].append(piece)
                                    if piece not in stats['first_request_seconds'] and len(stats['first_request_seconds']) < 2048:
                                        stats['first_request_seconds'][piece] = elapsed()
                                    disconnect = plan.disconnect_after > 0 and stats['requests'] >= plan.disconnect_after and stats['disconnects'] == 0
                                    if disconnect:
                                        owner.disconnects += 1
                                        stats['disconnects'] += 1
                                if disconnect:
                                    return
                                next_send = max(next_send, time.monotonic()) + plan.delay_ms/1000 + (count/plan.rate if plan.rate else 0)
                                pending.append((next_send, piece, begin, count))
                except (OSError, EOFError, ValueError, KeyError, IndexError, TypeError):
                    pass

        class Tracker(BaseHTTPRequestHandler):
            def do_GET(self):
                query = self.path.partition("?")[2]
                values = dict(item.split("=", 1) for item in query.split("&") if "=" in item)
                if unquote_to_bytes(values.get("info_hash", "")) != owner.info_hash:
                    body = bencode({b"failure reason": b"Unknown generated torrent"})
                else:
                    with owner.lock:
                        owner.announces += 1
                    online = [p for p in owner.peers if p.plan.online(time.monotonic()-owner.started)]
                    peers = b''.join(socket.inet_aton(p.server_address[0]) + struct.pack('!H', p.server_address[1]) for p in online)
                    complete = sum(len(p.plan.available(owner.piece_count, time.monotonic()-owner.started)) == owner.piece_count for p in online)
                    body = bencode({b"interval": 5, b"min interval": 1, b"complete": complete, b"incomplete": len(online)-complete, b"peers": peers if owner.advertise_peers else b""})
                self.send_response(200)
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            def log_message(self, *_):
                pass

        class Peers(socketserver.ThreadingTCPServer):
            daemon_threads = True
            allow_reuse_address = True

        self.peers = []
        for index, plan in enumerate(self.plans):
            server = Peers((f'127.0.0.{index+1}', 0), Peer)
            server.index, server.plan = index, plan
            self.peers.append(server)
        self.peer = self.peers[0]  # compatibility with existing harnesses
        self.tracker = ThreadingHTTPServer(("127.0.0.1", 0), Tracker)
        self.threads = [threading.Thread(target=server.serve_forever, kwargs={'poll_interval': .05}, daemon=True) for server in (*self.peers, self.tracker)]

    def __enter__(self):
        self.started = time.monotonic()
        for thread in self.threads:
            thread.start()
        return self

    def __exit__(self, *_):
        self.stop.set()
        for peer in self.peers:
            peer.shutdown()
        self.tracker.shutdown()
        for peer in self.peers:
            peer.server_close()
        self.tracker.server_close()
        for thread in self.threads:
            thread.join(2)

    def torrent(self):
        url = f"http://127.0.0.1:{self.tracker.server_port}/announce"
        return bencode({b"announce": url, b"info": self.info})

    def status(self):
        with self.lock:
            return {"sent_bytes": self.sent_bytes, "connections": self.connections, "requests": self.requests,
                    "disconnects": self.disconnects, "tracker_announces": self.announces,
                    "peers": [dict(s, requested_pieces=list(s['requested_pieces']), first_request_seconds=dict(s['first_request_seconds'])) for s in self.peer_stats],
                    "rate_limit_bytes_per_second": self.rate, "block_delay_ms": self.delay * 1000}
