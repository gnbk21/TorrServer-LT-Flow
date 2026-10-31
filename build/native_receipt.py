"""Record and verify native compiler, recipe and installed dependency identity."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shlex
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[1]
FLAGS = ('TARGET', 'GOOS', 'GOARCH', 'GOARM', 'B2_COMPILER', 'B2_VARIANT', 'B2_FLAGS',
         'CMAKE_CROSS_ARGS', 'CMAKE_GENERATOR', 'OPENSSL_CONFIG_TARGET', 'OPENSSL_CROSS_PREFIX',
         'OPENSSL_EXTRA_ARGS', 'EXTRA_CGO_LDFLAGS', 'BOOST_VERSION', 'LIBTORRENT_REV', 'OPENSSL_VERSION')


def sha(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def compiler(command):
    parts = shlex.split(command)
    if not parts:
        raise ValueError('compiler command is required')
    executable = shutil.which(parts[0])
    if not executable:
        raise ValueError('compiler executable is unavailable')
    version = subprocess.check_output(parts+['--version'], text=True, timeout=15).splitlines()
    return {'name': Path(executable).name, 'driver_sha256': sha(Path(executable)),
            'version': '\n'.join(version[:2]), 'arguments_sha256': hashlib.sha256(json.dumps(parts[1:]).encode()).hexdigest()}


def identity(driver, root=ROOT):
    if Path(driver).name != driver or not (root/'build'/driver).is_file():
        raise ValueError('unknown native build driver')
    recipes = [root/'build'/p for p in ('_common.sh', '_deps.sh', '_fetch_sources.sh',
               'native_receipt.py', 'apply_native_patches.py', 'patch_libtorrent_header.py', driver)]
    recipes += sorted((root/'build/native-patches').rglob('*'))
    # Store flag hashes instead of embedding private SDK/workspace paths.
    return {'schema_version': 1, 'driver': driver,
            'cc': compiler(os.environ['CC']), 'cxx': compiler(os.environ['CXX']),
            'flags': {k: hashlib.sha256(os.environ.get(k, '').encode()).hexdigest() for k in FLAGS},
            'recipes': {p.relative_to(root).as_posix(): sha(p) for p in recipes if p.is_file()}}


def artifacts(deps):
    deps = deps.resolve(strict=True)
    required = ['lib/libtorrent-rasterbar.a', 'lib/libssl.a', 'lib/libcrypto.a',
                'lib/libusrsctp.a', 'lib/libjuice-openssl.a', 'lib/pkgconfig/libtorrent-rasterbar.pc',
                'include/libtorrent/version.hpp']
    if any(not (deps/p).is_file() for p in required):
        raise ValueError('native installation is incomplete')
    found = {}
    for folder in ('include', 'lib'):
        for path in sorted((deps/folder).rglob('*')):
            if path.is_file():
                path.resolve(strict=True).relative_to(deps)
                found[path.relative_to(deps).as_posix()] = sha(path)
    return found


def verify(deps, expected):
    saved = json.loads((deps/'.flow-native-receipt.json').read_text(encoding='utf-8'))
    if saved.get('identity') != expected or saved.get('artifacts') != artifacts(deps):
        raise ValueError('native dependency/compiler/patch identity changed')
    return saved


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=('identity', 'write', 'verify'))
    parser.add_argument('--driver', required=True)
    parser.add_argument('--deps', type=Path)
    args = parser.parse_args()
    expected = identity(args.driver)
    if args.action == 'identity':
        print(hashlib.sha256(json.dumps(expected, sort_keys=True).encode()).hexdigest())
    elif args.action == 'verify':
        verify(args.deps, expected)
        print('Native dependency/compiler/patch receipt verified')
    else:
        result = {'identity': expected, 'artifacts': artifacts(args.deps),
                  'go_version': subprocess.check_output(['go', 'version'], text=True, timeout=15).strip()}
        temporary = args.deps/'.flow-native-receipt.pending'
        temporary.write_text(json.dumps(result, indent=2)+'\n', encoding='utf-8')
        temporary.replace(args.deps/'.flow-native-receipt.json')
        print('Native toolchain receipt written')
