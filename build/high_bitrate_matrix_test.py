import unittest
from high_bitrate_matrix import policy_order,regression_flags,MIB

class ComparisonTests(unittest.TestCase):
    def test_two_binary_order_really_rotates(self):
        self.assertEqual([policy_order(['baseline','candidate'],i)[0] for i in range(5)],
                         ['baseline','candidate','baseline','candidate','baseline'])
        self.assertEqual({tuple(policy_order(list(range(4)),i)) for i in range(4)},
                         {(0,1,2,3),(1,2,3,0),(2,3,0,1),(3,0,1,2)})

    def test_review_thresholds_require_relative_and_absolute_growth(self):
        summary=lambda latency,rss:{'seek_ttfb_p95_ms':{'median':latency},'peak_observed_rss_bytes':{'median':rss}}
        self.assertEqual(regression_flags({'healthy/previous-legacy':summary(100,100*MIB),
                                          'healthy/legacy':summary(200,110*MIB)}),[])
        rows=regression_flags({'healthy/previous-legacy':summary(100,100*MIB),
                               'healthy/legacy':summary(400,120*MIB)})
        self.assertEqual(len(rows),2)
        self.assertTrue(all(row['requires_diagnosis'] for row in rows))

    def test_rejected_fixture_bytes_always_require_review(self):
        rows=regression_flags({'healthy/previous-legacy':{},'healthy/legacy':{'native_failed_bytes':{'maximum':16384}}})
        self.assertEqual(len(rows),1)
        self.assertEqual(rows[0]['metric'],'native_failed_bytes')
        self.assertTrue(rows[0]['requires_diagnosis'])
