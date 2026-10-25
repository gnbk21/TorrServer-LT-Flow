"""Fail closed before reusing unchanged-component evidence for a patch release."""
import argparse
import json
from pathlib import PurePosixPath
from pathlib import Path
import re
import subprocess


CI_FILES = {
    'build/ci-apt.sh', 'build/ci-integration.sh', 'build/workflow_policy_test.py',
    'build/essential_release.py', 'build/essential_release_test.py',
    'build/release_gates.py', 'build/release_gates_test.py',
    'RELEASE_ACCEPTANCE.json',
}

# Reviewed API compatibility surface. This is deliberately not a blanket API
# exemption: each release's acceptance record pins its independently tested
# candidate, after which only CI/document changes are allowed.
COMPATIBILITY_FILES = {
    'server/web/api/stream.go', 'server/web/api/play.go',
    'server/web/api/rutor.go', 'server/web/api/torznab.go',
    'server/web/api/jacred.go', 'server/web/api/utils/testlink.go',
    'server/web/api/utils/query.go', 'server/web/api/utils/query_test.go',
    'server/web/api/query_test.go', 'server/torr/handle_lifecycle_test.go',
    'server/lt/dht_state_test.go',
}


def validate_changes(paths):
    for name in paths:
        path = PurePosixPath(name)
        if path.is_absolute() or '..' in path.parts or '\\' in name:
            raise ValueError(f'Invalid repository path: {name}')
        if (name in CI_FILES or name.startswith('.github/workflows/')
                or (path.suffix == '.md' and (len(path.parts) == 1 or path.parts[0] == 'docs'))):
            continue
        raise ValueError(f'Full validation required: runtime/build input changed: {name}')


def validate_compatibility_changes(paths):
    for name in paths:
        if name in COMPATIBILITY_FILES:
            continue
        validate_changes([name])


def patch_plan(tag, acceptance):
    plan = acceptance.get('patch_validation')
    if not plan or plan.get('tag') != tag:
        return None  # new releases retain full validation unless explicitly scoped
    if not re.fullmatch(r'MatriX\.145\.Flow-v\d+\.\d+\.[1-9]\d*', tag):
        raise ValueError('Focused checks require a stable patch tag')
    if plan.get('mode') != 'api_compatibility':
        raise ValueError('Unsupported focused validation mode')
    for field in ('candidate_commit', 'baseline_commit'):
        if not isinstance(plan.get(field), str) or not re.fullmatch(r'[0-9a-f]{40}', plan[field]):
            raise ValueError(f'Invalid {field}')
    for field in ('candidate_run', 'baseline_run'):
        if type(plan.get(field)) is not int or plan[field] <= 0:
            raise ValueError(f'Invalid {field}')
    return plan


def validate_run_receipts(plan, repository, candidate, baseline):
    for run, source, workflow in (
        (candidate, plan['candidate_commit'], '.github/workflows/codeql.yml'),
        (baseline, plan['baseline_commit'], '.github/workflows/build.yml'),
    ):
        if (run.get('head_sha') != source or run.get('conclusion') != 'success'
                or run.get('status') != 'completed' or run.get('path') != workflow
                or run.get('repository', {}).get('full_name') != repository):
            raise ValueError('Accepted source/workflow/repository did not complete successfully')


def changed_since(commit):
    changed = subprocess.check_output([
        'git', 'diff', '--no-renames', '--name-only', '-z', commit, 'HEAD', '--',
    ]).decode('utf-8').split('\0')
    return [name for name in changed if name]


def select_release(tag, repository, acceptance):
    plan = patch_plan(tag, acceptance)
    if plan is None:
        return {'validation_scope': 'full', 'web_run_id': ''}
    if not re.fullmatch(r'[\w.-]+/[\w.-]+', repository):
        raise ValueError('Invalid repository')
    validate_compatibility_changes(changed_since(plan['baseline_commit']))
    validate_changes(changed_since(plan['candidate_commit']))
    runs = [json.loads(subprocess.check_output([
        'gh', 'api', f'repos/{repository}/actions/runs/{plan[field]}',
    ])) for field in ('candidate_run', 'baseline_run')]
    validate_run_receipts(plan, repository, *runs)
    return {'validation_scope': 'essential', 'web_run_id': str(plan['baseline_run'])}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('accepted_commit', nargs='?')
    parser.add_argument('--compatibility', action='store_true')
    parser.add_argument('--release-tag')
    parser.add_argument('--repository')
    parser.add_argument('--github-output', type=Path)
    args = parser.parse_args()
    if args.release_tag:
        if not args.repository or not args.github_output:
            parser.error('--release-tag requires --repository and --github-output')
        result = select_release(args.release_tag, args.repository, json.loads(Path('RELEASE_ACCEPTANCE.json').read_text()))
        with args.github_output.open('a', encoding='utf-8') as output:
            for name, value in result.items():
                output.write(f'{name}={value}\n')
        print(json.dumps(result))
    elif not args.accepted_commit:
        parser.error('Specify an accepted commit or --release-tag')
    elif args.compatibility:
        paths = changed_since(args.accepted_commit)
        validate_compatibility_changes(paths)
        print('Only reviewed API compatibility files, tests and CI/documents differ; engine, cache, scheduler, web, dependencies and build/distribution inputs match the accepted full run.')
    else:
        validate_changes(changed_since(args.accepted_commit))
        print('Runtime, web, dependencies, native patches, build recipes and distribution inputs match the accepted full run.')
