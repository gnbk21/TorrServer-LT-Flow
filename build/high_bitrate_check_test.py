"""Reject cached or unknown data as evidence of a cold streaming seek."""
import unittest
from high_bitrate_check import select_cold_piece, range_cache_evidence, compact_cache_sample, observed_cache_turnover, observed_rss_bytes, MIB


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

    def test_churn_counts_observed_eviction_and_reappearance_not_initial_fill(self):
        sample=lambda *pieces:{'known':True,'verified_pieces':[[p,8*MIB] for p in pieces]}
        result=observed_cache_turnover([sample(1,2),sample(2,3),sample(1,3)])
        self.assertTrue(result['known'])
        self.assertEqual(result['evicted_verified_bytes'],16*MIB)
        self.assertEqual(result['reappeared_verified_bytes'],8*MIB)
        self.assertIn('lower_bound',result['method'])

    def test_unknown_churn_cannot_be_reported_as_zero(self):
        result=observed_cache_turnover([{'known':False},None])
        self.assertFalse(result['known'])
        self.assertIsNone(result['evicted_verified_bytes'])

    def test_unavailable_rss_is_not_a_measured_zero_or_peak(self):
        self.assertEqual(observed_rss_bytes([{}, {'memory': {'rss_bytes': 999}}]), [])
        self.assertEqual(observed_rss_bytes([
            {'memory': {'rss_available': False, 'rss_bytes': 999}},
            {'memory': {'rss_available': True, 'rss_bytes': 20}}]), [20])

    def test_partial_cache_blocks_are_not_verified_turnover(self):
        snapshot=self.snapshot()
        snapshot['Pieces']['0']['Completed']=False
        snapshot['Pieces']['1']['Size']=1
        result=compact_cache_sample(snapshot,128*MIB,8*MIB)
        self.assertTrue(result['known'])
        self.assertNotIn(0,dict(result['verified_pieces']))
        self.assertNotIn(1,dict(result['verified_pieces']))
        self.assertFalse(compact_cache_sample({},128*MIB,8*MIB)['known'])


if __name__ == '__main__':
    unittest.main()
