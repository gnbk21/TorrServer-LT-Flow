"""Fail closed when a test directory is missing from the explicit CI suites."""
import argparse
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def inventory(root=ROOT):
    suites = json.loads((root/'build/go-test-packages.json').read_text(encoding='utf-8'))
    actual = {p.parent.relative_to(root/'server').as_posix() for p in (root/'server').rglob('*_test.go')}
    listed = [p for suite in suites.values() for p in suite]
    if len(listed) != len(set(listed)) or set(listed) != actual:
        raise ValueError(f'Test inventory differs: missing={sorted(actual-set(listed))}; stale={sorted(set(listed)-actual)}')
    return suites


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--list', choices=('native', 'gst'))
    args = parser.parse_args()
    suites = inventory()
    if args.list:
        print(' '.join('./'+p if p != '.' else './' for p in suites[args.list]))
    else:
        print(f'All {sum(map(len, suites.values()))} Go test directories assigned to CI suites')
