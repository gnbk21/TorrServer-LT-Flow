"""Identify the helper built from this checkout, without executing a Windows PE."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess


def sha256(path):
    with path.open('rb') as data:
        return hashlib.file_digest(data, 'sha256').hexdigest()


def generate(binary, root, compiler):
    with binary.open('rb') as data:
        if data.read(2) != b'MZ':
            raise ValueError('Interface guard is not a Windows executable')
    commit = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()
    version = subprocess.check_output([compiler, '-dumpfullversion'], text=True).strip()
    target = subprocess.check_output([compiler, '-dumpmachine'], text=True).strip()
    if target != 'x86_64-w64-mingw32':
        raise ValueError('Release guard requires the Windows amd64 MinGW toolchain')
    archives = []
    for name in ('libstdc++.a', 'libgcc.a', 'libgcc_eh.a', 'libmingw32.a', 'libmingwex.a'):
        path = Path(subprocess.check_output([compiler, '-print-file-name='+name], text=True).strip())
        if not path.is_file():
            raise ValueError('Missing static guard runtime archive: '+name)
        archives.append({'name':name,'sha256':sha256(path)})
    return {'schema_version': 1, 'name': binary.name, 'source_commit': commit,
            'executable_sha256': sha256(binary),
            'source_sha256': sha256(root/'build/windows-interface-guard.cpp'),
            'scope': 'Explicit optional per-executable Windows WFP; DNS protection unverified',
            'compiler': {'family':'GCC/MinGW-w64','version':version,'target':target},
            'runtime_archives': archives,
            'runtime': 'Static MinGW-w64/GCC/libstdc++; same notices as Windows native build'}


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--compiler', required=True)
    args = parser.parse_args()
    args.output.write_text(json.dumps(generate(args.binary, Path(__file__).resolve().parent.parent, args.compiler), indent=2)+'\n', encoding='utf-8')
