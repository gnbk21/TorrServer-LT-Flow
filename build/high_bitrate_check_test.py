"""Reject cached or unknown data as evidence of a cold streaming seek."""
import unittest
from unittest.mock import Mock, patch
from high_bitrate_check import select_cold_piece, range_cache_evidence, compact_cache_sample, observed_cache_turnover, observed_rss_bytes, measure_cold_seeks, MIB


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

    def cold_fixture(self):
        server = Mock()
        snapshot = {'PiecesLength': MIB, 'PiecesCount': 64, 'Pieces': {}}
        server.json.side_effect = lambda path, *args: snapshot if path == '/cache' else {'memory': {}}
        swarm = Mock(data=bytes(64*MIB))
        return server, swarm

    def test_inflight_arrival_is_retained_but_not_counted_as_a_cold_proof(self):
        server, swarm = self.cold_fixture()
        records, resources, admissions = [], [], []
        with patch('high_bitrate_check.piece_request_count', side_effect=[256,256]+[0,1]*4), \
                patch('high_bitrate_check.range_read', side_effect=lambda *args: {'ttfb_ms': 20}):
            measure_cold_seeks(server,swarm,'hash',1,MIB,records,resources,admissions)
        self.assertEqual(len(records),4)
        self.assertTrue(all(row['cold_verified'] for row in records))
        self.assertEqual(len(admissions),5)
        self.assertFalse(admissions[0]['cold_verified'])
        self.assertEqual(admissions[0]['admission'],'unproven_inflight_or_cache_race')
        self.assertNotIn(admissions[0]['piece'],[row['piece'] for row in records])
        self.assertEqual(len(resources),5)

    def test_cold_proof_admission_is_bounded_and_cannot_succeed_without_four_proofs(self):
        server, swarm = self.cold_fixture()
        records, resources, admissions = [], [], []
        with patch('high_bitrate_check.piece_request_count',return_value=256), \
                patch('high_bitrate_check.range_read',return_value={'ttfb_ms': 20}):
            with self.assertRaisesRegex(AssertionError,'four verified cold'):
                measure_cold_seeks(server,swarm,'hash',1,MIB,records,resources,admissions)
        self.assertEqual(records,[])
        self.assertEqual(len(admissions),12)

    def test_delivery_timeout_or_integrity_failure_is_never_retried(self):
        for error in (TimeoutError('read timeout'), AssertionError('byte mismatch')):
            with self.subTest(error=type(error).__name__):
                server, swarm = self.cold_fixture()
                with patch('high_bitrate_check.piece_request_count',return_value=0), \
                        patch('high_bitrate_check.range_read',side_effect=error) as read:
                    with self.assertRaises(type(error)):
                        measure_cold_seeks(server,swarm,'hash',1,MIB,[],[],[])
                self.assertEqual(read.call_count,1)


if __name__ == '__main__':
    unittest.main()
