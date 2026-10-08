"""Verify generated peers on real sockets independently of the server binary."""
import hashlib
from pathlib import Path
import socket
import struct
import tempfile
import time
import unittest

from controlled_peer import LocalSwarm, PeerPlan, bdecode, bencode


class WireTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.file = Path(self.directory.name) / 'episode.bin'
        self.file.write_bytes(bytes(range(256)) * 256)

    def connect(self, swarm, index=0):
        sock = socket.create_connection(swarm.peers[index].server_address, timeout=2)
        self.addCleanup(sock.close)
        sock.sendall(b'\x13BitTorrent protocol' + b'\0'*8 + swarm.info_hash + b'-TEST00-012345678901')
        self.assertEqual(self.read(sock, 68)[28:48], swarm.info_hash)
        return sock

    @staticmethod
    def read(sock, size):
        data = bytearray()
        while len(data) < size:
            chunk = sock.recv(size-len(data))
            if not chunk:
                raise EOFError()
            data.extend(chunk)
        return bytes(data)

    def message(self, sock):
        size, = struct.unpack('!I', self.read(sock, 4))
        return self.read(sock, size)

    @staticmethod
    def send(sock, packet):
        sock.sendall(struct.pack('!I', len(packet)) + packet)

    def initial(self, sock):
        return [self.message(sock) for _ in range(3)]

    def test_complementary_partial_peers_and_hashes(self):
        with LocalSwarm([self.file], piece_length=16384, peers=[PeerPlan(pieces={0, 2}), PeerPlan(pieces={1, 3})]) as swarm:
            data = bytearray()
            for piece in range(4):
                sock = self.connect(swarm, piece % 2)
                initial = self.initial(sock)
                self.assertEqual(initial[0], b'\x05' + (b'\xa0' if piece % 2 == 0 else b'\x50'))
                self.send(sock, b'\x06' + struct.pack('!III', piece, 0, 16384))
                packet = self.message(sock)
                self.assertEqual(packet[:9], b'\x07' + struct.pack('!II', piece, 0))
                block = packet[9:]
                self.assertEqual(hashlib.sha1(block).digest(), swarm.info[b'pieces'][piece*20:piece*20+20])
                data.extend(block)
            self.assertEqual(data, swarm.data)

    def test_late_have_and_unchoke(self):
        with LocalSwarm([self.file], piece_length=16384, peers=[PeerPlan(pieces={1}, choke_until=.15, have_events=[(.1, 0)])]) as swarm:
            sock = self.connect(swarm)
            initial = self.initial(sock)
            self.assertEqual(initial[0], b'\x05\x40')
            self.assertEqual(initial[1], b'\x00')
            self.assertEqual(self.message(sock), b'\x04' + struct.pack('!I', 0))
            self.assertEqual(self.message(sock), b'\x01')
            self.send(sock, b'\x06' + struct.pack('!III', 0, 0, 16384))
            self.assertEqual(self.message(sock)[9:], swarm.data[:16384])

    def test_metadata_extension_respects_delay(self):
        with LocalSwarm([self.file], peers=[PeerPlan(metadata_delay=.2)]) as swarm:
            sock = self.connect(swarm)
            initial = self.initial(sock)
            self.assertEqual(bdecode(initial[2][2:])[0][b'm'][b'ut_metadata'], 1)
            self.send(sock, b'\x14\x00' + bencode({b'm': {b'ut_metadata': 7}}))
            self.send(sock, b'\x14\x01' + bencode({b'msg_type': 0, b'piece': 0}))
            start = time.monotonic()
            packet = self.message(sock)
            self.assertGreater(time.monotonic()-start, .12)
            self.assertEqual(packet[:2], b'\x14\x07')
            header, offset = bdecode(packet[2:])
            self.assertEqual(header[b'total_size'], len(swarm.metadata))
            self.assertEqual(hashlib.sha1(packet[2+offset:]).digest(), swarm.info_hash)

    def test_cancel_and_interruption_reconnect(self):
        with LocalSwarm([self.file], peers=[PeerPlan(delay_ms=200, outages=[(.3, .45)])]) as swarm:
            sock = self.connect(swarm)
            self.initial(sock)
            request = struct.pack('!III', 0, 0, 16384)
            self.send(sock, b'\x06' + request)
            self.send(sock, b'\x08' + request)
            with self.assertRaises(EOFError):
                self.message(sock)
            self.assertEqual(swarm.status()['sent_bytes'], 0)
            time.sleep(.17)
            sock = self.connect(swarm)
            self.initial(sock)
            self.send(sock, b'\x06' + request)
            self.assertEqual(self.message(sock)[9:], swarm.data[:16384])

    def test_variable_rate_changes_real_queued_delivery(self):
        with LocalSwarm([self.file], peers=[PeerPlan(rate=16384, rate_events=[(.1,256*1024)])]) as swarm:
            sock=self.connect(swarm)
            self.initial(sock)
            for begin in range(0,65536,16384):
                self.send(sock,b'\x06'+struct.pack('!III',0,begin,16384))
            delivered=[]
            for begin in range(0,65536,16384):
                self.assertEqual(self.message(sock)[9:],swarm.data[begin:begin+16384])
                delivered.append(time.monotonic())
            slow=delivered[1]-delivered[0]
            fast=delivered[3]-delivered[2]
            self.assertGreater(slow,.7)
            self.assertLess(fast,slow/2)
            self.assertIsNotNone(swarm.status()['peers'][0]['first_sent_seconds'])

    def test_request_capacity_precedes_unchoke_and_close_is_recorded(self):
        with LocalSwarm([self.file], peers=[PeerPlan(request_queue=512)]) as swarm:
            sock = self.connect(swarm)
            initial = self.initial(sock)
            self.assertEqual(initial[1][:2], b'\x14\x00')
            self.assertEqual(bdecode(initial[1][2:])[0][b'reqq'], 512)
            self.assertEqual(initial[2], b'\x01')
            # Invalid bounds close this owned peer, with an explicit reason.
            self.send(sock, b'\x06'+struct.pack('!III', 0, 65536, 16384))
            with self.assertRaises(EOFError):
                self.message(sock)
            self.assertEqual(swarm.status()['peers'][0]['close_reasons'], {'invalid-request': 1})

    def test_strict_capacity_rejects_overflow(self):
        with LocalSwarm([self.file], peers=[PeerPlan(request_queue=2, strict_request_queue=True, delay_ms=500)]) as swarm:
            sock = self.connect(swarm)
            self.initial(sock)
            for begin in (0, 16384, 32768):
                self.send(sock, b'\x06'+struct.pack('!III', 0, begin, 16384))
            with self.assertRaises(EOFError): self.message(sock)
            self.assertEqual(swarm.status()['peers'][0]['peak_pending_requests'], 2)
            self.assertEqual(swarm.status()['peers'][0]['close_reasons'], {'queue-limit': 1})


if __name__ == '__main__':
    unittest.main()
