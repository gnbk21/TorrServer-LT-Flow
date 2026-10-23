"""Explicit 1.1.5 recovery: retain the tag and all successful non-macOS gates."""
import json
from pathlib import Path
import subprocess

from release_gates import validate_jobs

REPOSITORY = 'gnbk21/TorrServer-LT-Flow'
TAG = 'MatriX.145.Flow-v1.1.5'
SOURCE = '04cbb1ae08bfb87f25249876201066880c08a2d5'
RUN = 39982129539
SECURITY_RUN = 39982122481
ALLOWED = {
    'build/release.py', 'build/release_gates.py', 'build/recover_release.py',
    'build/recover_release_test.py', '.github/workflows/recover-release.yml',
    'RELEASE_1_1_5_NOTES.md', 'CI_PIPELINE.md', 'MACOS_RUNNER_INCIDENT.md',
    '.github/workflows/build-docker.yml',
}


def validate(run, security, jobs, annotations):
    for receipt, path in ((run, '.github/workflows/ts_release.yml'),
                          (security, '.github/workflows/codeql.yml')):
        if (receipt.get('head_sha') != SOURCE or receipt.get('path') != path
                or receipt.get('repository', {}).get('full_name') != REPOSITORY
                or receipt.get('status') != 'completed'):
            raise ValueError('Wrong source, workflow, repository or unfinished evidence')
    if (run.get('id') != RUN or run.get('event') != 'push' or run.get('head_branch') != TAG
            or run.get('conclusion') != 'failure' or security.get('id') != SECURITY_RUN
            or security.get('conclusion') != 'success'):
        raise ValueError('Unexpected tagged/security evidence')
    count = validate_jobs(jobs, 'essential', 'non-macos')
    latest = {row['name']: row for row in sorted(jobs, key=lambda row: row['id'])}
    for arch in ('arm64', 'amd64'):
        job = latest['macos / macos/' + arch]
        if (job.get('conclusion') != 'cancelled' or job.get('runner_id') != 0
                or job.get('steps') or job.get('labels') != ['macos-latest']):
            raise ValueError('macOS has a real build failure or an unexplained cancellation')
        messages = [row.get('message', '') for row in annotations[str(job['id'])]]
        if (not any('not acquired by Runner of type hosted' in message for message in messages)
                or not any('capacity constraints' in message for message in messages)):
            raise ValueError('Missing explicit GitHub scheduler/capacity diagnosis')
    return count


def api(path):
    return json.loads(subprocess.check_output(['gh', 'api', '--paginate', '--slurp',
                     f'repos/{REPOSITORY}/' + path], text=True))


if __name__ == '__main__':
    root = Path(__file__).resolve().parent.parent
    actual = subprocess.check_output(['git', 'rev-parse', TAG+'^{commit}'], cwd=root, text=True).strip()
    if actual != SOURCE:
        raise ValueError('Published source tag must not move')
    changed = subprocess.check_output(['git', 'diff', '--name-only', SOURCE, 'HEAD'], cwd=root, text=True).splitlines()
    if not set(changed) <= ALLOWED:
        raise ValueError('Recovery may change only reviewed packaging, its tests and release documents')
    run = api(f'actions/runs/{RUN}')[0]
    security = api(f'actions/runs/{SECURITY_RUN}')[0]
    jobs = [job for page in api(f'actions/runs/{RUN}/jobs?filter=all&per_page=100') for job in page['jobs']]
    latest = {row['name']: row for row in sorted(jobs, key=lambda row: row['id'])}
    annotations = {}
    for arch in ('arm64', 'amd64'):
        job = latest['macos / macos/'+arch]
        annotations[str(job['id'])] = [row for page in api(f'check-runs/{job["id"]}/annotations') for row in page]
    count = validate(run, security, jobs, annotations)
    report = dict(source_commit=SOURCE, tag=TAG, original_run=RUN, security_run=SECURITY_RUN,
                  required_roles_passed=count, platform_scope='non-macos',
                  packaging_workflow_commit=subprocess.check_output(['git','rev-parse','HEAD'], cwd=root, text=True).strip(),
                  run=run, security=security, jobs=jobs, scheduler_annotations=annotations)
    (root/'recovery-evidence.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    print(f'{count} actual successful roles; both omitted macOS roles failed before runner assignment.')
