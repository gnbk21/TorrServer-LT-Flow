"""Resolve/verify the single validated PR web artifact, without a privileged trigger."""
import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import time

MANIFEST = 'flow-web-source.json'


def expected_identity(env=os.environ):
    identity = {'schema_version': 1, 'repository': env['GITHUB_REPOSITORY'],
                'merge_sha': env['GITHUB_SHA'], 'head_sha': env.get('FLOW_PR_HEAD') or env['GITHUB_SHA'],
                'pull_request': int(env.get('FLOW_PR_NUMBER') or 0)}
    if not re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', identity['repository']):
        raise ValueError('invalid repository identity')
    if any(not re.fullmatch(r'[a-f0-9]{40}', identity[key]) for key in ('head_sha', 'merge_sha')):
        raise ValueError('invalid source identity')
    return identity


def verify(directory, expected):
    actual = json.loads((directory/MANIFEST).read_text(encoding='utf-8'))
    if actual != expected:
        raise ValueError('web artifact belongs to another repository, PR, head or merge source')


def trusted_run(run, expected):
    return (run.get('event') == 'pull_request' and run.get('path', '').split('@')[0] == '.github/workflows/checks.yml'
            and run.get('repository', {}).get('full_name') == expected['repository']
            and run.get('head_sha') == expected['head_sha']
            and any(p.get('number') == expected['pull_request'] and p.get('head', {}).get('sha') == expected['head_sha']
                    for p in run.get('pull_requests', [])))


def gh(path):
    return json.loads(subprocess.check_output(['gh', 'api', path], text=True, timeout=30))


def resolve(expected, timeout=900):
    deadline = time.monotonic()+timeout
    repository = expected['repository']
    while time.monotonic() < deadline:
        runs = gh(f'repos/{repository}/actions/workflows/checks.yml/runs?event=pull_request&head_sha={expected["head_sha"]}&per_page=20')['workflow_runs']
        candidates = sorted((r for r in runs if trusted_run(r, expected)), key=lambda r: r['id'], reverse=True)
        if candidates:
            run = candidates[0]
            jobs = gh(f'repos/{repository}/actions/runs/{run["id"]}/jobs?per_page=100')['jobs']
            web = [j for j in jobs if j['name'] == 'web / web']
            if web and web[0]['status'] == 'completed':
                if web[0]['conclusion'] != 'success':
                    raise ValueError('shared web validation failed; no consumer may substitute an unvalidated build')
                artifacts = gh(f'repos/{repository}/actions/runs/{run["id"]}/artifacts?per_page=100')['artifacts']
                if any(a['name'] == 'flow-web-pr' and not a['expired'] for a in artifacts):
                    return run['id']
            if run['status'] == 'completed' and not web:
                raise ValueError('shared source run did not execute its required web job')
        time.sleep(5)
    raise TimeoutError('shared PR web validation was not available within 15 minutes')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=('resolve', 'write', 'verify'))
    parser.add_argument('--directory', type=Path, default=Path('server/web/pages/template'))
    args = parser.parse_args()
    expected = expected_identity()
    if args.action == 'resolve':
        if expected['pull_request'] <= 0:
            parser.error('cross-run reuse is limited to the same pull request')
        run_id = resolve(expected)
        with Path(os.environ['GITHUB_OUTPUT']).open('a', encoding='utf-8') as output:
            output.write(f'run-id={run_id}\n')
        print('Shared validated PR web artifact ready:', run_id)
    elif args.action == 'verify':
        verify(args.directory, expected)
        print('Web repository, PR, head and merge-source identity verified')
    else:
        checkout = subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip()
        if checkout != expected['merge_sha']:
            raise ValueError('checkout differs from the triggering source')
        (args.directory/MANIFEST).write_text(json.dumps(expected, indent=2)+'\n', encoding='utf-8')
