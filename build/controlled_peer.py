#!/usr/bin/env python3
"""A local BitTorrent seeder/tracker for generated fixtures and failure tests.

Implements the ordinary v1 handshake, bitfield, unchoke and request/piece wire
messages. The server under test must download real hash-verified pieces through
libtorrent. No production mock cache or external swarm participates.
"""
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import socket
import socketserver
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


class LocalSwarm:
    def __init__(self, paths, rate=0, delay_ms=0, disconnect_after=0):
        self.paths = paths
        self.data = b"".join(p.read_bytes() for p in paths)
        self.piece_length = 256 * 1024
        pieces = b"".join(hashlib.sha1(self.data[i:i+self.piece_length]).digest()
                          for i in range(0, len(self.data), self.piece_length))
        self.info = {b"name": b"Flow-generated-fixtures", b"piece length": self.piece_length,
                     b"pieces": pieces, b"files": [{b"length": p.stat().st_size, b"path": [p.name.encode()]} for p in paths]}
        self.info_hash = hashlib.sha1(bencode(self.info)).digest()
        self.piece_count = len(pieces) // 20
        self.rate, self.delay = rate, delay_ms / 1000
        self.disconnect_after = disconnect_after
        self.stop = threading.Event()
        self.lock = threading.Lock()
        self.sent_bytes = self.connections = self.requests = self.disconnects = self.announces = 0
        owner = self

        class Peer(socketserver.BaseRequestHandler):
            def handle(self):
                sock = self.request
                sock.settimeout(1)
                with owner.lock:
                    owner.connections += 1
                try:
                    handshake = receive(sock, 68, owner.stop)
                    if handshake[:20] != b"\x13BitTorrent protocol" or handshake[28:48] != owner.info_hash:
                        return
                    sock.sendall(b"\x13BitTorrent protocol" + b"\0"*8 + owner.info_hash + b"-FL0002-012345678901")
                    bits = bytearray((owner.piece_count + 7) // 8)
                    for piece in range(owner.piece_count):
                        bits[piece // 8] |= 0x80 >> (piece % 8)
                    sock.sendall(struct.pack("!I", len(bits)+1) + b"\x05" + bits + b"\0\0\0\x01\x01")
                    while not owner.stop.is_set():
                        length, = struct.unpack("!I", receive(sock, 4, owner.stop))
                        if length > 128 * 1024:
                            return
                        packet = receive(sock, length, owner.stop)
                        if not packet or packet[0] != 6 or len(packet) != 13:
                            continue
                        piece, begin, count = struct.unpack("!III", packet[1:])
                        if piece >= owner.piece_count or count > 32768 or count == 0 or begin + count > owner.piece_length:
                            return
                        with owner.lock:
                            owner.requests += 1
                            disconnect = owner.disconnect_after > 0 and owner.requests >= owner.disconnect_after and owner.disconnects == 0
                            if disconnect:
                                owner.disconnects += 1
                        if disconnect:
                            return
                        offset = piece * owner.piece_length + begin
                        block = owner.data[offset:offset+count]
                        if len(block) != count:
                            return
                        if owner.stop.wait(owner.delay + (count / owner.rate if owner.rate > 0 else 0)):
                            return
                        sock.sendall(struct.pack("!I", len(block)+9) + b"\x07" + struct.pack("!II", piece, begin) + block)
                        with owner.lock:
                            owner.sent_bytes += count
                except (OSError, EOFError):
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
                    peers = socket.inet_aton("127.0.0.1") + struct.pack("!H", owner.peer.server_address[1])
                    body = bencode({b"interval": 5, b"min interval": 1, b"complete": 1, b"incomplete": 0, b"peers": peers})
                self.send_response(200)
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            def log_message(self, *_):
                pass

        class Peers(socketserver.ThreadingTCPServer):
            daemon_threads = True
            allow_reuse_address = True

        self.peer = Peers(("127.0.0.1", 0), Peer)
        self.tracker = ThreadingHTTPServer(("127.0.0.1", 0), Tracker)
        self.threads = [threading.Thread(target=server.serve_forever, daemon=True) for server in (self.peer, self.tracker)]

    def __enter__(self):
        for thread in self.threads:
            thread.start()
        return self

    def __exit__(self, *_):
        self.stop.set()
        self.peer.shutdown()
        self.tracker.shutdown()
        self.peer.server_close()
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
                    "rate_limit_bytes_per_second": self.rate, "block_delay_ms": self.delay * 1000}
