import struct
import unittest
from fixtures import mp4_layout, mkv_layout


class ContainerFixtureTests(unittest.TestCase):
    def test_mp4_top_level_offsets_ignore_payload_lookalikes(self):
        data=struct.pack('>I4s',16,b'mdat')+b'moovfake'+struct.pack('>I4s',8,b'moov')
        self.assertEqual(mp4_layout(data),{'mdat':0,'moov':16})

    def test_matroska_top_level_cues_front_and_tail(self):
        cluster=bytes.fromhex('1f43b675')+b'\x84cues'
        cues=bytes.fromhex('1c53bb6b')+b'\x80'
        def segment(payload):return bytes.fromhex('18538067')+bytes([0x80|len(payload)])+payload
        front=mkv_layout(segment(cues+cluster))
        tail=mkv_layout(segment(cluster+cues))
        self.assertLess(front['cues'],front['first_cluster'])
        self.assertGreater(tail['cues'],tail['last_cluster'])
