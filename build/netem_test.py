import unittest
from unittest.mock import Mock, patch

from netem_check import wait_stream_observations


class AsyncNetworkEvidence(unittest.TestCase):
    @patch('netem_check.time.sleep')
    def test_waits_for_real_native_snapshot(self, sleep):
        server = Mock()
        server.json.side_effect = [{}, {'sparse': {}},
                                   {'sparse': {'stream': {'known': True, 'frontiers': 2}}}]
        evidence = wait_stream_observations(server, 'owned')
        self.assertEqual(evidence['frontiers'], 2)
        self.assertEqual(server.json.call_count, 3)
        self.assertEqual(sleep.call_count, 2)

    @patch('netem_check.time.sleep')
    @patch('netem_check.time.monotonic', side_effect=[0, 6])
    def test_unavailable_evidence_still_fails(self, monotonic, sleep):
        server = Mock()
        server.json.return_value = {'sparse': {'stream': {'known': False}}}
        with self.assertRaisesRegex(AssertionError, 'observations unavailable'):
            wait_stream_observations(server, 'owned')
        sleep.assert_not_called()


if __name__ == '__main__':
    unittest.main()
