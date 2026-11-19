from pathlib import Path
import tempfile
import unittest
from unittest.mock import MagicMock, patch

from playback_harness import OwnedServer


class OwnedListenerPorts(unittest.TestCase):
    def test_http_and_profiler_refuse_peer_port_and_each_others_port(self):
        offered = (47000, 47001, 47001, 47000, 47002)
        reservations = []
        for port in offered:
            reservation = MagicMock()
            reservation.__enter__.return_value.getsockname.return_value = ("127.0.0.1", port)
            reservations.append(reservation)

        with tempfile.TemporaryDirectory() as directory, \
                patch("playback_harness.socket.socket", side_effect=reservations), \
                patch("playback_harness.subprocess.Popen") as launch:
            server = OwnedServer(Path("fixture"), Path(directory) / "state",
                                 profile=True, peer_port=47000)
            with server.log:
                self.assertEqual(server.port, 47001)
                self.assertEqual(server.profile_base, "http://127.0.0.1:47002")
                command = launch.call_args.args[0]
                self.assertEqual(command[command.index("--torrentaddr") + 1], "127.0.0.1:47000")
                self.assertEqual(command[command.index("--port") + 1], "47001")
