"""Validate immutable source and successful roles before platform recovery.

This reusable path deliberately requires full validation. Historical essential
patch recovery remains documented separately; it is not silently generalized.
"""
import argparse
import json
from pathlib import Path
import re
import subprocess

from release import REPOSITORY, classify
from release_gates import validate_jobs


def validate(run, security, jobs, tag, source, omission_approved=False):
    classify(tag)
    if not re.fullmatch('[a-f0-9]{40}', source):
        raise ValueError('Expected immutable full source commit')
    for receipt, path in ((run, '.github/workflows/ts_release.yml'),
                          (security, '.github/workflows/codeql.yml')):
        if (receipt.get('head_sha') != source or receipt.get('path') != path
                or receipt.get('repository', {}).get('full_name') != REPOSITORY
                or receipt.get('status') != 'completed'):
            raise ValueError('Wrong source, workflow, repository or unfinished evidence')
    if run.get('event') != 'push' or run.get('head_branch') != tag or security.get('conclusion') != 'success':
        raise ValueError('Tagged build and successful source-identical security checks required')
    count = validate_jobs(jobs, 'full', 'non-macos')
    latest = {row['name']: row for row in sorted(jobs, key=lambda row: row['id'])}
    omitted = []
    for arch in ('arm64', 'amd64'):
        row = latest.get('macos / macos/'+arch, {})
        if row.get('conclusion') != 'success':
            omitted.append('darwin-'+arch)
            # A platform that ran and failed cannot be described as a queue loss.
            if row.get('runner_id') or row.get('steps') or row.get('conclusion') not in ('cancelled', 'failure'):
                raise ValueError('Omitted platform is not an unassigned hosted-runner failure')
    if omitted and not omission_approved:
        raise ValueError('Missing platforms require explicit approval and disclosure')
    return count, omitted


def api(path):
    return json.loads(subprocess.check_output(['gh', 'api', '--paginate', '--slurp',
                      f'repos/{REPOSITORY}/'+path], text=True))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--run', type=int, required=True)
    parser.add_argument('--security-run', type=int, required=True)
    parser.add_argument('--tag', required=True)
    parser.add_argument('--source', required=True)
    parser.add_argument('--allow-missing-macos', action='store_true')
    parser.add_argument('--reason', default='')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if args.run <= 0 or args.security_run <= 0:
        parser.error('Positive run IDs required')
    if args.allow_missing_macos and (not args.reason.strip() or len(args.reason)>1024):
        parser.error('An explicit bounded omission explanation is required')
    actual = subprocess.check_output(['git', 'rev-parse', args.tag+'^{commit}'], text=True).strip()
    if actual != args.source:
        raise ValueError('The release tag must retain the original binary source')
    # Recovery runs newer packaging against old source. Never let changes to
    # runtime, native build pins, guard, UI, dependencies or distributed scripts
    # masquerade as source-identical packaging.
    changed = subprocess.check_output(['git', 'diff', '--name-only', args.source, 'HEAD'], text=True).splitlines()
    allowed = {'build/release.py', 'build/platform_recovery.py', 'build/platform_recovery_test.py',
               'build/release_gates.py', '.github/workflows/release-recovery.yml'}
    if any(path not in allowed and not path.endswith('.md') for path in changed):
        raise ValueError('Recovery checkout contains non-packaging changes from binary source')
    run=api(f'actions/runs/{args.run}')[0]; security=api(f'actions/runs/{args.security_run}')[0]
    jobs=[job for page in api(f'actions/runs/{args.run}/jobs?filter=all&per_page=100') for job in page['jobs']]
    count,missing=validate(run,security,jobs,args.tag,args.source,args.allow_missing_macos)
    annotations={}
    latest={row["name"]:row for row in sorted(jobs,key=lambda row:row["id"])}
    for row in latest.values():
        if row.get('name','').startswith('macos / macos/') and row.get('conclusion')!='success':
            evidence=[item for page in api(f'check-runs/{row["id"]}/annotations') for item in page]
            if not any('not acquired by Runner of type hosted' in item.get('message','') for item in evidence):
                raise ValueError('Missing explicit runner-assignment failure evidence')
            annotations[str(row['id'])]=evidence
    args.output.write_text(json.dumps(dict(tag=args.tag,source_commit=args.source,run=run,security=security,
        jobs=jobs,missing=missing,reason=args.reason,required_roles_passed=count,annotations=annotations),indent=2)+'\n')
