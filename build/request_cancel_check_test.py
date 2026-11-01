from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from fixture_ports import PortAllocationError
from request_cancel_check import trial


class CancellationFixtureAdmission(unittest.TestCase):
    def test_exhausted_admission_never_starts_a_server(self):
        with tempfile.TemporaryDirectory() as directory, \
                patch('request_cancel_check.joint_loopback_port', side_effect=PortAllocationError([{'winerror': 10013}])), \
                patch('request_cancel_check.OwnedServer') as server:
            result = trial(Path('unused.exe'), Path(directory)/'trial', Path('unused.ts'), False)
        server.assert_not_called()
        self.assertFalse(result['passed'])
        self.assertEqual(result['failure_stage'], 'fixture_port_allocation')

    def test_startup_failure_is_not_retried_and_listener_is_explicit(self):
        with tempfile.TemporaryDirectory() as directory, \
                patch('request_cancel_check.joint_loopback_port', return_value=47000) as allocator, \
                patch('request_cancel_check.OwnedServer') as server:
            server.return_value.ready.side_effect = TimeoutError('controlled startup failure')
            result = trial(Path('unused.exe'), Path(directory)/'trial', Path('unused.ts'), True)
        allocator.assert_called_once()
        server.assert_called_once()
        self.assertEqual(server.call_args.kwargs['extra_arguments'], ['--torrentaddr', '127.0.0.1:47000'])
        server.return_value.close.assert_called_once()
        self.assertFalse(result['passed'])
        self.assertEqual(result['failure_stage'], 'server_ready')
