"""Check evidence limits and comparison gates, not simulated playback claims."""
import contextlib
import hashlib
import io
import json
from pathlib import Path
from types import SimpleNamespace
import tempfile
import unittest
from unittest.mock import patch

from high_bitrate_check import capture_go_profile, failure_evidence, tracker_evidence
import pgo_compare


class Profiling(unittest.TestCase):
    def test_tracker_failure_evidence_keeps_codes_without_identities(self):
        row = dict(protocol='http', status='ERROR', error_code=111,
                   host='private-host', id='private-id', url='private-url',
                   message='private-detail')
        result = tracker_evidence({'trackers': [row]*100})
        self.assertEqual(len(result), 64)
        self.assertEqual(result[0], dict(protocol='http', status='ERROR', error_code=111))
    def test_failure_evidence_omits_network_identities_and_player_traces(self):
        calls = []
        def query(path, timeout):
            calls.append((path, timeout))
            if path == '/flow/network':
                return {'listener_errors': 2, 'peer_tcp_port': 0,
                        'addresses': ['private-address'], 'last_error': 'private-detail'}
            return {'startup': {'state': 'BOOTSTRAP_PRELOAD'},
                    'hash': 'private-hash', 'traces': ['private-trace']}
        result = failure_evidence(SimpleNamespace(json=query), 'fixture')
        self.assertEqual(result, {'network': {'listener_errors': 2, 'peer_tcp_port': 0},
                                  'startup': {'state': 'BOOTSTRAP_PRELOAD'}})
        self.assertEqual(calls, [('/flow/network', 2), ('/flow/status/fixture?traces=false', 2)])

    def test_failure_evidence_preserves_unknown_and_handles_preupload_failure(self):
        def fail(path, timeout):
            raise TimeoutError('private-detail')
        server = SimpleNamespace(json=fail)
        self.assertEqual(failure_evidence(server, None), {'network_unavailable': 'TimeoutError'})
        self.assertEqual(failure_evidence(server, 'fixture'), {
            'network_unavailable': 'TimeoutError', 'startup_unavailable': 'TimeoutError'})

    def test_profile_is_bounded_and_records_exact_content(self):
        with tempfile.TemporaryDirectory() as root:
            server=SimpleNamespace(state=Path(root),profile_base='http://127.0.0.1:1234')
            with patch('high_bitrate_check.urllib.request.urlopen',return_value=io.BytesIO(b'profile')) as request:
                evidence=capture_go_profile(server,'profile',30)
            self.assertEqual(evidence['sha256'],hashlib.sha256(b'profile').hexdigest())
            self.assertEqual((server.state/'profile.pprof').read_bytes(),b'profile')
            self.assertEqual(request.call_args.kwargs['timeout'],45)
            class TooLarge(io.BytesIO):
                def read(self,limit): return bytes(limit)
            with patch('high_bitrate_check.urllib.request.urlopen',return_value=TooLarge()):
                with self.assertRaises(ValueError): capture_go_profile(server,'trace',5)
            self.assertFalse((server.state/'trace.pprof').exists())

    def comparison(self, fail=False):
        with tempfile.TemporaryDirectory() as root:
            base=Path(root)
            for name in ('baseline','candidate','fixture'): (base/name).write_bytes(name.encode())
            args=SimpleNamespace(output=base/'results',fixture=base/'fixture',baseline=base/'baseline',
                candidate=base/'candidate',mbps=120,seconds=60,rotations=5)
            calls=[]
            def workload(exe,path,fixture,profile,case,rate,seconds,cache,preload,**kwargs):
                calls.append((exe.name,case,rate,cache,kwargs))
                return {'case':case,'passed':not(fail and len(calls)==1), 'cpu_seconds':None,
                    'delivery':{'blocked_over_250ms_ms':0,'read_wait_p95_ms':2,'read_wait_p99_ms':3}}
            swarm=SimpleNamespace(__enter__=lambda self:self,__exit__=lambda *args:None)
            class Swarm:
                def __enter__(self): return swarm
                def __exit__(self,*args): pass
            with patch('pgo_compare.LocalSwarm',return_value=Swarm()), patch('pgo_compare.calibrate',return_value=200), \
                 patch('pgo_compare.run',side_effect=workload), patch('pgo_compare.gc.collect'), contextlib.redirect_stdout(io.StringIO()):
                passed=pgo_compare.compare(args)
            report=json.loads((args.output/'report.json').read_text())
            self.assertEqual(len(calls),40)
            self.assertEqual(calls[0][0],'baseline')
            self.assertEqual(calls[2][0],'candidate')
            self.assertTrue(all(call[2:4]==(15_000_000,512) and call[4]=={'startup_mb':32} for call in calls))
            self.assertFalse(report['adopted'])
            self.assertTrue(report['acceptance_repeats_met'])
            self.assertNotIn('cpu_seconds',report['summaries']['healthy/baseline'])
            return passed

    def test_rotations_match_reserves_keep_unknown_cpu_and_never_promote(self):
        self.assertTrue(self.comparison())

    def test_one_integrity_failure_fails_the_comparison_gate(self):
        self.assertFalse(self.comparison(fail=True))


if __name__=='__main__': unittest.main()
