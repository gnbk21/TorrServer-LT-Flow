"""Reject cached or unknown data as evidence of a cold streaming seek."""
import unittest
from high_bitrate_check import select_cold_piece, MIB


class ColdSeekEvidenceTests(unittest.TestCase):
    def snapshot(self):
        return {'PiecesLength': 8*MIB, 'PiecesCount': 16,
                'Pieces': {str(i): {'Size': 8*MIB, 'Completed': True} for i in range(16)}}

    def test_warm_and_unknown_maps_cannot_pass_as_cold(self):
        for snapshot in (self.snapshot(), {}, dict(self.snapshot(), Pieces=None)):
            with self.assertRaises(AssertionError):
                select_cold_piece(snapshot, 128*MIB, 8*MIB, False, set())

    def test_alternating_seeks_require_distinct_unavailable_pieces(self):
        snapshot = self.snapshot()
        for index in (0, 3, 4, 8, 12, 15):
            del snapshot['Pieces'][str(index)]
        low = select_cold_piece(snapshot, 128*MIB, 8*MIB, False, set())
        high = select_cold_piece(snapshot, 128*MIB, 8*MIB, True, {low}, low)
        self.assertNotIn(str(low), snapshot['Pieces'])
        self.assertNotIn(str(high), snapshot['Pieces'])
        self.assertGreaterEqual(abs(high-low), 2)
        self.assertNotIn(low, (0, 1, 14, 15))
        with self.assertRaises(AssertionError):
            select_cold_piece(snapshot, 128*MIB, 4*MIB, False, set())


if __name__ == '__main__':
    unittest.main()
