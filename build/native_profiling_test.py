"""Check profiler scope, bounds, decoding gates and owned-session cleanup."""
import json
from pathlib import Path
from types import SimpleNamespace
import tempfile
import unittest
from unittest.mock import patch

import native_profiling as profiling


class NativeProfiling(unittest.TestCase):
    def test_refuses_host_capture_and_platform_mismatch(self):
        with patch.dict(profiling.os.environ, {}, clear=True):
            with self.assertRaises(ValueError): profiling.validate('etw')
            with self.assertRaises(ValueError): profiling.validate('perfetto')
        with patch.object(profiling, 'os', SimpleNamespace(name='nt', environ={
                'GITHUB_ACTIONS':'true', 'FLOW_ISOLATED_PROFILING':'github-hosted'})):
            profiling.validate('etw')
            with self.assertRaises(ValueError): profiling.validate('perfetto')

    def perfetto(self, csv='"samples","stacks","outside_samples"\n12,10,0\n'):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ('tracebox', 'trace_processor_shell'): (root/name).write_bytes(b'tool')
            server = SimpleNamespace(state=root, process=SimpleNamespace(pid=417, poll=lambda:None))
            commands=[]
            def command(arguments, log, timeout=45):
                commands.append(arguments)
                (root/'native-cpu.perfetto-trace').write_bytes(b'trace')
            def run(arguments, **kwargs):
                commands.append(arguments)
                kwargs['stdout'].write(csv.encode())
            with patch.object(profiling, 'validate'), patch.object(profiling, 'command', side_effect=command), \
                 patch.object(profiling.os, 'getuid', return_value=1001, create=True), \
                 patch.object(profiling.os, 'getgid', return_value=1001, create=True), \
                 patch.dict(profiling.os.environ, {'FLOW_PROFILE_SOURCE_RUN': '38087989863'}), \
                 patch.object(profiling.subprocess, 'run', side_effect=run):
                if '12,10,0' in csv:
                    result = profiling.capture(server, 'perfetto', 30, root)
                    self.assertTrue(result['passed'])
                    self.assertEqual(result['samples']['outside_samples'], 0)
                    self.assertEqual(result['source_run_id'], '38087989863')
                    config = (root/'native-cpu.pbtxt').read_text()
                    self.assertIn('target_pid: 417', config)
                    self.assertIn('duration_ms: 30000', config)
                    self.assertNotIn('process_stats', config)
                    self.assertNotIn('ftrace', config)
                    self.assertEqual(commands[0][:3], ['sudo','-n','--'])
                    self.assertEqual(commands[1], ['sudo', '-n', '--', 'chown', '--no-dereference',
                                                   '1001:1001', '--', str(root/'native-cpu.perfetto-trace')])
                else:
                    with self.assertRaises(ValueError): profiling.capture(server, 'perfetto', 30, root)
                    self.assertFalse(json.loads((root/'native-cpu.report.json').read_text())['passed'])

    def test_perfetto_pid_scope_and_decode(self): self.perfetto()

    def test_rejects_empty_unknown_or_foreign_samples(self):
        for row in ('0,0,0', '12,0,0', '12,10,1'):
            self.perfetto('samples,stacks,outside_samples\n'+row+'\n')

    def test_etw_stops_only_recording_it_started_even_on_process_exit(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            calls=[]
            server = SimpleNamespace(state=root, process=SimpleNamespace(pid=417, poll=lambda:None))
            fake_os=SimpleNamespace(name='nt',environ={'SystemRoot':str(root)})
            def command(arguments, log, timeout=45):
                calls.append(arguments)
                if '-stop' in arguments: (root/'native-cpu.etl').write_bytes(b'trace')
            with patch.object(profiling, 'validate'), patch.object(profiling, 'os', fake_os), \
                 patch.object(profiling, 'command', side_effect=command), \
                 patch.object(profiling.time, 'monotonic', return_value=0):
                server.process.poll=iter([None, 1]).__next__
                with self.assertRaises(RuntimeError): profiling.capture(server, 'etw', 1)
            self.assertEqual([call[1] for call in calls], ['-start', '-stop'])
            self.assertEqual(calls[0][-2:], calls[1][-2:])
            self.assertEqual(calls[0][-2], '-instancename')
            self.assertTrue(calls[0][2].endswith('flow_cpu.wprp!FlowCPU'))
            self.assertNotIn('-filemode', calls[0])
            self.assertFalse(json.loads((root/'native-cpu.report.json').read_text())['passed'])
            calls.clear()
            server.process.poll=lambda:None
            def reject_start(arguments, log, timeout=45):
                calls.append(arguments)
                raise RuntimeError('another recording exists')
            with patch.object(profiling, 'validate'), patch.object(profiling, 'os', fake_os), \
                 patch.object(profiling, 'command', side_effect=reject_start):
                with self.assertRaises(RuntimeError): profiling.capture(server, 'etw', 1)
            self.assertEqual([call[1] for call in calls], ['-start'])

    def test_output_bounds_and_process_liveness(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)/'trace'
            path.write_bytes(b'')
            with self.assertRaises(ValueError): profiling.evidence(path)
            path.write_bytes(b'x')
            self.assertEqual(profiling.evidence(path)['bytes'],1)
            with patch.object(profiling, 'LIMIT', 0):
                with self.assertRaises(ValueError): profiling.evidence(path)
            server=SimpleNamespace(process=SimpleNamespace(pid=417,poll=lambda:1))
            with patch.object(profiling, 'validate'):
                with self.assertRaises(ValueError): profiling.capture(server, 'perfetto', 30, directory)

    def test_invalid_source_run_is_rejected_before_any_capture(self):
        server = SimpleNamespace(process=SimpleNamespace(pid=417, poll=lambda:None))
        for identity in ('0', '-1', 'not-a-run', '01'):
            with patch.object(profiling, 'validate'), patch.object(profiling, 'command') as command, \
                 patch.dict(profiling.os.environ, {'FLOW_PROFILE_SOURCE_RUN': identity}):
                with self.assertRaisesRegex(ValueError, 'positive numeric identity'):
                    profiling.capture(server, 'perfetto', 30)
                command.assert_not_called()


if __name__ == '__main__': unittest.main()
