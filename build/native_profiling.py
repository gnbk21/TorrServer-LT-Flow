"""Opt-in native CPU evidence on disposable CI runners, never the user's host."""
import csv
import hashlib
import io
import json
import os
from pathlib import Path
import re
import subprocess
import time
import uuid
import xml.etree.ElementTree as ET

LIMIT = 128 * 1024 * 1024


def validate(kind):
    if kind not in ('etw', 'perfetto'):
        raise ValueError('Unknown native profiler')
    if os.environ.get('GITHUB_ACTIONS') != 'true' or os.environ.get('FLOW_ISOLATED_PROFILING') != 'github-hosted':
        raise ValueError('Native capture requires an explicitly isolated GitHub-hosted runner')
    if (kind == 'etw') != (os.name == 'nt'):
        raise ValueError('Native profiler does not match this platform')


def evidence(path):
    size = path.stat().st_size
    if not 0 < size <= LIMIT:
        raise ValueError('Native trace is empty or exceeds 128 MiB')
    with path.open('rb') as data:
        digest = hashlib.file_digest(data, 'sha256').hexdigest()
    return {'name': path.name, 'bytes': size, 'sha256': digest}


def command(arguments, log, timeout=45):
    with log.open('ab') as output:
        subprocess.run(arguments, stdout=output, stderr=subprocess.STDOUT, check=True, timeout=timeout)


def etw_events(path):
    evidence(path)
    root = ET.parse(path).getroot()
    events = root.find(".//Data[@name='events']")
    lost = root.find(".//Data[@name='lostEvents']")
    if root.tag != 'Report' or events is None or lost is None:
        raise ValueError('ETW decoding returned no event report')
    counts = {'events': int(events.text or '0'), 'lost_events': int(lost.text or '-1')}
    if counts['events'] <= 0 or counts['lost_events'] != 0:
        raise ValueError('ETW decoding is empty or reports lost events')
    return counts


def capture(server, kind, seconds, tools=None):
    validate(kind)
    if not 1 <= seconds <= 30 or server.process.poll() is not None or server.process.pid <= 0:
        raise ValueError('Native capture needs a live owned process and a duration of 1..30 seconds')
    report = {'schema_version': 1, 'kind': kind, 'owned_pid': server.process.pid,
              'duration_seconds': seconds, 'isolated_runner': True,
              'profiling_affects_measurement': True, 'passed': False}
    source_run = os.environ.get('FLOW_PROFILE_SOURCE_RUN')
    if source_run is not None:
        if not re.fullmatch(r'[1-9][0-9]*', source_run):
            raise ValueError('Native profile source run must be a positive numeric identity')
        report['source_run_id'] = source_run
    log = server.state / 'native-profiler.log'
    try:
        if kind == 'etw':
            # CPU profiling is system-scoped. Restrict it to a disposable runner
            # executing synthetic fixtures; never start/stop a workstation WPR.
            wpr = Path(os.environ['SystemRoot']) / 'System32/wpr.exe'
            tracerpt = wpr.with_name('tracerpt.exe')
            trace = server.state / 'native-cpu.etl'
            profile = Path(__file__).with_name('flow_cpu.wprp').resolve()
            if not profile.is_file():
                raise ValueError('Bounded native CPU recording profile is missing')
            instance = ['-instancename', 'FlowSynthetic-'+str(server.process.pid)+'-'+uuid.uuid4().hex[:12]]
            started = False
            try:
                try:
                    # Default CPU/file mode produced a 275 MiB trace on CI.
                    # Use only sampled CPU/module/thread evidence, with 64 MiB
                    # circular buffers. The saved trace retains its 128 MiB gate.
                    command([str(wpr), '-start', str(profile)+'!FlowCPU', *instance], log)
                except subprocess.TimeoutExpired:
                    # The start command may have created a session before its
                    # timeout. Cancel only our unique instance, never a default.
                    command([str(wpr), '-cancel', *instance], log)
                    raise
                started = True
                deadline = time.monotonic() + seconds
                while time.monotonic() < deadline:
                    if server.process.poll() is not None:
                        raise RuntimeError('Owned process exited during ETW capture')
                    time.sleep(max(0, min(.1, deadline-time.monotonic())))
            finally:
                if started:
                    try:
                        command([str(wpr), '-stop', str(trace), '-skipPdbGen', *instance], log, timeout=60)
                    except Exception:
                        command([str(wpr), '-cancel', *instance], log)
                        raise
            report['trace'] = evidence(trace)
            summary = server.state / 'etw-summary.txt'
            decoded = server.state / 'etw-report.xml'
            # -summary always emits text, regardless of the filename suffix.
            # -report with XML format is the structured decoding result.
            command([str(tracerpt), str(trace), '-summary', str(summary),
                     '-report', str(decoded), '-f', 'XML', '-o', 'NUL', '-y'], log)
            evidence(summary)
            counts = etw_events(decoded)
            report.update(scope='System CPU on disposable synthetic CI; filter owned PID in WPA',
                          decoder_validated=True, native_attribution_known=False,
                          decoded_events=counts['events'], lost_events=counts['lost_events'],
                          decoder_report=evidence(decoded),
                          circular_buffer_bytes=64*1024*1024,
                          trace_may_cover_recent_window_only=True,
                          recording_profile=evidence(profile))
        else:
            tools = Path(tools).resolve()
            tracebox, processor = tools / 'tracebox', tools / 'trace_processor_shell'
            for tool in (tracebox, processor):
                if not tool.is_file():
                    raise ValueError('Pinned Perfetto tools are missing')
            trace = server.state / 'native-cpu.perfetto-trace'
            config = server.state / 'native-cpu.pbtxt'
            config.write_text(f'''duration_ms: {seconds*1000}
max_file_size_bytes: {64*1024*1024}
buffers {{ size_kb: 32768 fill_policy: DISCARD }}
data_sources {{ config {{ name: "linux.perf" perf_event_config {{
timebase {{ counter: SW_CPU_CLOCK frequency: 99 timestamp_clock: PERF_CLOCK_MONOTONIC }}
callstack_sampling {{ scope {{ target_pid: {server.process.pid} }} kernel_frames: true }}
}} }} }}
''', encoding='utf-8')
            command(['sudo', '-n', '--', str(tracebox), '--txt', '-c', str(config), '-o', str(trace)],
                    log, timeout=seconds+45)
            if trace.is_symlink() or not trace.is_file():
                raise ValueError('Owned native trace is not a regular file')
            # tracebox runs with perf permissions and creates a root-owned 0600
            # file. Transfer only this owned fixture file to the invoking CI
            # account so decoding and artifact upload can read it.
            command(['sudo', '-n', '--', 'chown', '--no-dereference',
                     f'{os.getuid()}:{os.getgid()}', '--', str(trace)], log)
            report['trace'] = evidence(trace)
            query = f'''SELECT count(*) AS samples, count(s.callsite_id) AS stacks,
count(CASE WHEN p.pid != {server.process.pid} OR p.pid IS NULL THEN 1 END) AS outside_samples
FROM perf_sample s LEFT JOIN thread t ON t.utid=s.utid LEFT JOIN process p ON p.upid=t.upid
WHERE s.callsite_id IS NOT NULL;'''
            result = server.state / 'native-samples.csv'
            with result.open('wb') as output, log.open('ab') as errors:
                subprocess.run([str(processor), '-Q', query, str(trace)], stdout=output, stderr=errors,
                               check=True, timeout=30)
            evidence(result)
            rows = list(csv.DictReader(io.StringIO(result.read_text(encoding='utf-8'))))
            if len(rows) != 1:
                raise ValueError('Native sample query returned an invalid result')
            samples = {key: int(rows[0][key]) for key in ('samples', 'stacks', 'outside_samples')}
            if samples['samples'] <= 0 or samples['stacks'] != samples['samples'] or samples['outside_samples'] != 0:
                raise ValueError('Native capture is empty, lacks stacks or includes another process')
            report.update(scope='Owned process call stacks; per-CPU timer counters may include other processes; no scheduling, logs or process scan',
                          counter_samples_may_include_other_processes=True,
                          decoder_validated=True, samples=samples,
                          tools={tool.name: evidence(tool)['sha256'] for tool in (tracebox, processor)},
                          native_attribution_known=False)
        report['passed'] = True
        return report
    except Exception as error:
        report['failure_type'] = type(error).__name__
        raise
    finally:
        (server.state / 'native-cpu.report.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
