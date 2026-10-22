"""Publication requires actual successful gates, never a skipped dependency."""
import argparse
import json
from pathlib import Path

GROUPS = ('playback', 'startup', 'discovery', 'preparation', 'adaptive', 'requests', 'sparse', 'maintenance')
TARGETS = ('linux-amd64', 'linux-arm64', 'linux-armv7', 'windows-amd64', 'android-arm64', 'android-armv7')
REQUIRED = ('classify', 'web / web', 'cross / sources',
            'cross / windows-network-notifications', 'cross / Native queue and deadline simulations',
            'cross / linux-validation', 'cross / controlled-evidence',
            'cross / release-inventory', 'cross / windows-service',
            'macos / macos/arm64', 'macos / macos/amd64')


def validate_jobs(jobs, scope):
    if scope not in ('essential', 'full'):
        raise ValueError('Unknown validation scope')
    # Successful jobs from earlier attempts remain valid when failed jobs are
    # rerun. A later skipped/failed attempt for the same role supersedes success.
    latest = {}
    for row in sorted(jobs, key=lambda row: row['id']):
        latest[row['name']] = row
    required = [*REQUIRED, *('cross / '+t for t in TARGETS),
                *('cross / controlled-integration ('+g+')' for g in GROUPS)]
    errors = [name for name in required if latest.get(name, {}).get('conclusion') != 'success']
    if scope == 'full':
        for prefix, count in (('cross / high-bitrate-comparison (', 10),
                              ('cross / native-profiling / capture (', 2)):
            rows = [row for name, row in latest.items() if name.startswith(prefix)]
            if len(rows) != count or any(row.get('conclusion') != 'success' for row in rows):
                errors.append(prefix+'*')
    if errors:
        raise ValueError('Required release jobs missing, skipped or unsuccessful: '+', '.join(errors))
    return len(required)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('jobs', type=Path)
    parser.add_argument('--scope', choices=('full', 'essential'), required=True)
    args = parser.parse_args()
    pages = json.loads(args.jobs.read_text(encoding='utf-8'))
    count = validate_jobs([job for page in pages for job in page['jobs']], args.scope)
    print(f'{count} required roles passed; scope={args.scope}; skipped jobs cannot satisfy publication.')
