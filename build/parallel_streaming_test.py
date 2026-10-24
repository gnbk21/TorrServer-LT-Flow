import unittest
from parallel_streaming_check import summarize


class ComparisonReportTest(unittest.TestCase):
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
