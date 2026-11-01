"""Fail before/after a release build if tracked or untracked source is dirty."""
import argparse
from pathlib import Path
import re
import subprocess

from release import classify


def verify(version, root):
    if not re.match(r'^MatriX\.[0-9]+\.Flow-(?:v[0-9]|preview\.)', version):
        return False  # Development/profiling builds retain honest dirty metadata.
    classify(version)
    dirty = subprocess.check_output(['git','status','--porcelain','--untracked-files=normal'],
                                    cwd=root, text=True)
    if dirty:
        raise ValueError('Release checkout has uncommitted changes; refusing dirty provenance:\n'+dirty)
    return True


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--version', required=True)
    args = parser.parse_args()
    checked = verify(args.version, Path(__file__).resolve().parents[1])
    print('Release source is clean.' if checked else 'Development build: original VCS state retained.')
