"""Joint loopback TCP/UDP admission for owned native fixtures.

This is a preflight, not a reservation handed to libtorrent. The final bind can
still fail and must remain a server-start assertion, never a playback retry.
"""
import errno
import random
import socket


class PortAllocationError(RuntimeError):
    def __init__(self, failures):
        self.failures = failures
        super().__init__(f'Joint TCP/UDP admission failed after {len(failures)} attempts')


def joint_loopback_port(*, attempts=32, socket_factory=socket.socket, evidence=None):
    if not 1 <= attempts <= 128:
        raise ValueError('attempts must be between 1 and 128')
    failures = []
    # Windows can hand out sequential TCP ephemeral ports inside a large UDP
    # exclusion. Repeated bind(0) then makes no useful progress. Keep the initial
    # OS-selected port, then spread distinct candidates across unprivileged ports;
    # the real TCP and UDP binds remain the only admission authority.
    candidates = [0, *random.SystemRandom().sample(range(1024, 65536), attempts-1)]
    for attempt, candidate in enumerate(candidates):
        tcp = udp = None
        port = None
        try:
            tcp = socket_factory(socket.AF_INET, socket.SOCK_STREAM)
            udp = socket_factory(socket.AF_INET, socket.SOCK_DGRAM)
            tcp.bind(('127.0.0.1', candidate))
            port = tcp.getsockname()[1]
            udp.bind(('127.0.0.1', port))
            if evidence is not None:
                evidence.update(attempts=attempt+1, admission_failures=failures, port=port)
            return port
        except OSError as error:
            failure = {'attempt': attempt+1, 'candidate_port': candidate, 'port': port, 'errno': error.errno,
                       'winerror': getattr(error, 'winerror', None)}
            failures.append(failure)
            if evidence is not None:
                evidence.update(attempts=attempt+1, admission_failures=failures)
            if error.errno not in (errno.EADDRINUSE, errno.EACCES, 10013, 10048) and getattr(error, 'winerror', None) not in (10013, 10048):
                raise
        finally:
            for reservation in (udp, tcp):
                if reservation is not None:
                    reservation.close()
    raise PortAllocationError(failures)
