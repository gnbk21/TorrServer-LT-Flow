import errno
import socket
import unittest
from unittest.mock import patch
from fixture_ports import joint_loopback_port, PortAllocationError


class PortTests(unittest.TestCase):
    def factory(self, errors):
        sockets = []
        class Reservation:
            closed = False
            def __init__(self, family, kind):
                self.kind = kind
                sockets.append(self)
            def bind(self, address):
                if self.kind == socket.SOCK_DGRAM and errors:
                    error = errors.pop(0)
                    if error:
                        raise OSError(error, 'controlled admission failure')
            def getsockname(self):
                return ('127.0.0.1', 40000+len(sockets))
            def close(self):
                self.closed = True
        return Reservation, sockets

    def test_retries_admission_only_and_closes_every_socket(self):
        factory, sockets = self.factory([errno.EACCES, errno.EADDRINUSE, None])
        evidence = {}
        self.assertGreater(joint_loopback_port(socket_factory=factory, evidence=evidence), 0)
        self.assertEqual(evidence['attempts'], 3)
        self.assertEqual(len(evidence['admission_failures']), 2)
        self.assertTrue(all(s.closed for s in sockets))

    def test_bounded_failure(self):
        factory, sockets = self.factory([errno.EACCES]*4)
        with self.assertRaises(PortAllocationError) as failure:
            joint_loopback_port(attempts=4, socket_factory=factory)
        self.assertEqual(len(failure.exception.failures), 4)
        self.assertEqual(len(sockets), 8)
        self.assertTrue(all(s.closed for s in sockets))

    def test_unrelated_failure_is_not_retried(self):
        factory, sockets = self.factory([errno.ENOBUFS])
        with self.assertRaises(OSError):
            joint_loopback_port(socket_factory=factory)
        self.assertEqual(len(sockets), 2)
        self.assertTrue(all(s.closed for s in sockets))

    def test_real_joint_port_released(self):
        port = joint_loopback_port()
        with socket.socket() as tcp, socket.socket(type=socket.SOCK_DGRAM) as udp:
            tcp.bind(('127.0.0.1', port))
            udp.bind(('127.0.0.1', port))

    def test_contiguous_udp_exclusions_do_not_exhaust_tcp_selected_retries(self):
        sockets = []
        next_tcp_port = 53542

        class Reservation:
            closed = False

            def __init__(self, family, kind):
                self.kind = kind
                sockets.append(self)

            def bind(self, address):
                nonlocal next_tcp_port
                self.port = address[1]
                if self.kind == socket.SOCK_STREAM and not self.port:
                    self.port = next_tcp_port
                    next_tcp_port += 1
                if self.kind == socket.SOCK_DGRAM and 53542 <= self.port <= 53700:
                    error = OSError(errno.EACCES, 'controlled contiguous UDP exclusion')
                    error.winerror = 10013
                    raise error

            def getsockname(self):
                return ('127.0.0.1', self.port)

            def close(self):
                self.closed = True

        evidence = {}
        with patch('fixture_ports.random.SystemRandom.sample', return_value=[47000, 47001, 47002]):
            port = joint_loopback_port(attempts=4, socket_factory=Reservation, evidence=evidence)
        self.assertEqual(port, 47000)
        self.assertEqual(evidence['attempts'], 2)
        self.assertEqual(evidence['admission_failures'][0]['winerror'], 10013)
        self.assertEqual(len(sockets), 4)
        self.assertTrue(all(s.closed for s in sockets))


if __name__ == '__main__':
    unittest.main()
