"""Fail closed before reusing full runtime evidence for a CI-only release repair."""
import argparse
from pathlib import PurePosixPath
import subprocess


CI_FILES = {
    'build/ci-apt.sh', 'build/ci-integration.sh', 'build/workflow_policy_test.py',
    'build/essential_release.py', 'build/essential_release_test.py',
    'build/release_gates.py', 'build/release_gates_test.py',
    'RELEASE_ACCEPTANCE.json',
}


def validate_changes(paths):
    for name in paths:
        path = PurePosixPath(name)
        if path.is_absolute() or '..' in path.parts or '\\' in name:
            raise ValueError(f'Invalid repository path: {name}')
        if (name in CI_FILES or name.startswith('.github/workflows/')
                or (len(path.parts) == 1 and path.suffix == '.md')):
            continue
        raise ValueError(f'Full validation required: runtime/build input changed: {name}')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('accepted_commit')
    args = parser.parse_args()
    changed = subprocess.check_output([
        'git', 'diff', '--no-renames', '--name-only', '-z', args.accepted_commit, 'HEAD',
    ]).decode('utf-8').split('\0')
    validate_changes([name for name in changed if name])
    print('Runtime, web, dependencies, native patches, build recipes and distribution inputs match the accepted full run.')
