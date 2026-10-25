import unittest
from parallel_streaming_check import summarize, accepted_trial


class ComparisonReportTest(unittest.TestCase):
    def test_only_supplied_stock_horizon_timeouts_are_expected_controls(self):
        row = dict(passed=False, policy='capacity', failure_stage='preload',
                   failure_type='TimeoutError', swarm={'peers': [dict(sent_bytes=16384,
                   peak_pending_requests=2, capacity_overflows=[])]})
        self.assertTrue(accepted_trial('high-latency', row))
        self.assertFalse(accepted_trial('large-pieces', row))
        self.assertFalse(accepted_trial('high-latency', dict(row, policy='combined')))
        self.assertFalse(accepted_trial('high-latency', dict(row, policy='requests')))
        self.assertFalse(accepted_trial('high-latency', dict(row, policy_error='capacity exceeded')))
        self.assertFalse(accepted_trial('high-latency', dict(row, swarm={'peers': []})))
        self.assertFalse(accepted_trial('high-latency', dict(row, failure_stage='range_validation')))
        self.assertFalse(accepted_trial('high-latency', dict(row, failure_type='AssertionError')))
    def test_failed_and_incomplete_trials_remain_in_summary(self):
        result = summarize([
            {"policy": "capacity", "passed": True,
             "delivery": {"blocked_over_250ms_ms": 100, "read_wait_max_ms": 100,
                          "behind_schedule_ms": 0}},
            {"policy": "capacity", "passed": False, "preload_ready_ms": 500},
            {"policy": "frontier", "passed": False},
        ])
        self.assertEqual(result["capacity"]["passed"], 1)
        self.assertEqual(result["capacity"]["total"], 2)
        self.assertEqual(result["frontier"]["total"], 1)
        self.assertEqual(result["requests"]["total"], 0)


if __name__ == "__main__":
    unittest.main()
