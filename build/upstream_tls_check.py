#!/usr/bin/env python3
"""Verify certificate maintenance in an owned real executable on loopback."""
import argparse
import hashlib
import json
from pathlib import Path
import socket
import ssl
import urllib.request

from playback_harness import OwnedServer


def unused_port():
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        return sock.getsockname()[1]


CASES = ('invalid-user', 'partial-pair', 'lone-default-key', 'generated', 'generated-read-only')


def run(executable, output, cases=CASES):
    output.mkdir(parents=True, exist_ok=False)
    results = {'executable_sha256': hashlib.sha256(executable.read_bytes()).hexdigest(), 'cases': []}
    invalid = b'original invalid user fixture; must remain unchanged\n'
    for case in cases:
        port = unused_port()
        arguments = ['--ssl', '--sslport', str(port)]
        seed = {}
        state = output/case
        if case == 'invalid-user':
            seed = {'custom.pem': invalid, 'custom.key': invalid}
            arguments += ['--sslcert', str((state/'custom.pem').resolve()),
                          '--sslkey', str((state/'custom.key').resolve())]
        elif case == 'partial-pair':
            seed = {'custom.pem': invalid}
            arguments += ['--sslcert', str((state/'custom.pem').resolve())]
        elif case == 'lone-default-key':
            seed = {'server.key': invalid}
        elif case == 'generated-read-only':
            # Read-only startup requires an existing, valid bbolt database.
            # Reuse the closed generated case's own database, never user state.
            seed = {'config.db': (output/'generated'/'config.db').read_bytes()}
            arguments += ['--rdb']
        server = OwnedServer(executable, state, seed_files=seed, extra_arguments=arguments)
        try:
            if case.startswith('generated'):
                server.ready()
                # Trust this fixture's generated cert and verify its IP SAN.
                context = ssl.create_default_context(cafile=str(state/'server.pem'))
                with urllib.request.urlopen(f'https://127.0.0.1:{port}/echo', context=context, timeout=5) as response:
                    if response.status != 200:
                        raise AssertionError('Generated HTTPS listener failed')
                pair = [(state/name).read_bytes() for name in ('server.pem', 'server.key')]
                if not all(pair):
                    raise AssertionError('Generated pair empty')
            else:
                server.process.wait(timeout=20)
                if server.process.returncode == 0:
                    raise AssertionError('Invalid HTTPS configuration reported success')
                for candidate in (server.port, port):
                    with socket.socket() as sock:
                        if sock.connect_ex(('127.0.0.1', candidate)) == 0:
                            raise AssertionError('Invalid HTTPS fell back to an open listener')
                for name, data in seed.items():
                    if (state/name).read_bytes() != data:
                        raise AssertionError('User certificate/key was overwritten')
                if (state/'server.pem').exists():
                    raise AssertionError('Invalid user configuration generated a replacement')
            results['cases'].append({'case': case, 'passed': True})
        finally:
            server.close()
            (output/'report.json').write_text(json.dumps(results, indent=2)+'\n', encoding='utf-8')
    return results


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--executable', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--cases', nargs='+', choices=CASES, default=list(CASES))
    args = parser.parse_args()
    if 'generated-read-only' in args.cases and ('generated' not in args.cases or args.cases.index('generated') > args.cases.index('generated-read-only')):
        parser.error('generated must precede generated-read-only to provide its owned database')
    print(json.dumps(run(args.executable, args.output, args.cases), indent=2))
