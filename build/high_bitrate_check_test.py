"""Reject cached or unknown data as evidence of a cold streaming seek."""
import unittest
from high_bitrate_check import select_cold_piece, range_cache_evidence, MIB


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

    def test_seek_evidence_checks_every_boundary_piece(self):
        snapshot = self.snapshot()
        snapshot['Pieces']['1']['Completed'] = False
        row = range_cache_evidence(snapshot,128*MIB,8*MIB,8*MIB-1,8*MIB)
        self.assertEqual(row['state'],'not_verified_resident')
        self.assertEqual([p['verified'] for p in row['pieces']],[True,False])
        row = range_cache_evidence(snapshot,128*MIB,8*MIB,0,65535)
        self.assertEqual(row['state'],'verified_resident')

    def test_missing_or_partial_data_is_not_verified(self):
        snapshot = self.snapshot()
        del snapshot['Pieces']['0']
        self.assertEqual(range_cache_evidence(snapshot,128*MIB,8*MIB,0,65535)['state'],'not_verified_resident')
        snapshot['Pieces']['0']={'Size':1,'Completed':True}
        self.assertEqual(range_cache_evidence(snapshot,128*MIB,8*MIB,0,65535)['state'],'not_verified_resident')

    def test_missing_geometry_cannot_claim_cache_knowledge(self):
        self.assertEqual(range_cache_evidence({},128*MIB,8*MIB,0,65535),{'state':'unknown','pieces':[]})


if __name__ == '__main__':
    unittest.main()
