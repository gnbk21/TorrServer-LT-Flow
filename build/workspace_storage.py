"""Preview generated workspace storage; apply only an explicitly reviewed manifest.

No recursive deletion. Toolchains, dependencies, evidence, archives, installed
packages, settings, credentials and rollback binaries are inventory-only.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
DISPOSABLE = ('.tools/buffering-go-cache', '.tools/yarn-cache', 'web/node_modules/.vite')
INVENTORY = DISPOSABLE + ('_out', '_deps', '_src', '.tools/msys64', '.tools/gopath')
PROTECTED_NAMES = ('settings', 'config', 'credential', 'secret', 'token', 'backup', 'rollback', 'manifest', 'accs.db')


def canonical_file(root, relative):
    root = root.resolve(strict=True)
    supplied = Path(relative)
    if supplied.is_absolute() or '..' in supplied.parts or any(':' in p for p in supplied.parts):
        raise ValueError('path must stay inside the known workspace')
    path = root / supplied
    resolved = path.resolve(strict=True)
    resolved.relative_to(root)
    current = path
    while current != root:
        if current.is_symlink() or (hasattr(current, 'is_junction') and current.is_junction()):
            raise ValueError('links and junctions are protected')
        current = current.parent
    if not resolved.is_file():
        raise ValueError('only regular files can be selected')
    return resolved


def active_paths():
    if os.name == 'nt':
        command = "Get-CimInstance Win32_Process -Filter \"Name LIKE 'TorrServer%' OR Name LIKE 'Flow%' OR Name='node.exe' OR Name='go.exe' OR Name='compile.exe' OR Name='link.exe' OR Name='gcc.exe' OR Name='g++.exe' OR Name='cc1.exe' OR Name='cc1plus.exe'\" | Select-Object Name,ExecutablePath | ConvertTo-Json -Compress"
        result = subprocess.run(['powershell.exe', '-NoProfile', '-NonInteractive', '-Command', command], capture_output=True, text=True, timeout=15, check=True)
        rows = json.loads(result.stdout or '[]')
        rows = rows if isinstance(rows, list) else [rows]
        paths = [Path(p['ExecutablePath']).resolve() for p in rows if p and p.get('ExecutablePath')]
        if any(p and p['Name'].lower() in ('go.exe','compile.exe','link.exe','gcc.exe','g++.exe','cc1.exe','cc1plus.exe') for p in rows):
            paths.append(ROOT/'.tools/buffering-go-cache/active-build')
        if any(p and p['Name'].lower() == 'node.exe' for p in rows):
            paths.append(ROOT/'web/node_modules/.vite/active-web')
            paths.append(ROOT/'.tools/yarn-cache/active-node')
        return paths
    if not Path('/proc').is_dir():
        # No trustworthy process inventory on this host: preview remains useful,
        # but all disposable caches stay protected rather than assuming idle.
        return [ROOT/p/'unknown-process-inventory' for p in DISPOSABLE]
    paths = []
    for link in Path('/proc').glob('[0-9]*/exe'):
        try:
            paths.append(link.resolve(strict=True))
        except OSError:
            pass
    if any(p.name in ('go','compile','link','gcc','g++','cc1','cc1plus') for p in paths):
        paths.append(ROOT/'.tools/buffering-go-cache/active-build')
    if any(p.name == 'node' for p in paths):
        paths.append(ROOT/'web/node_modules/.vite/active-web')
        paths.append(ROOT/'.tools/yarn-cache/active-node')
    return paths


def locked(path):
    if os.name != 'nt':
        import fcntl
        try:
            with path.open('rb') as stream:
                fcntl.flock(stream, fcntl.LOCK_EX | fcntl.LOCK_NB)
                fcntl.flock(stream, fcntl.LOCK_UN)
            return False
        except OSError:
            return True
    import ctypes
    from ctypes import wintypes
    kernel = ctypes.WinDLL('kernel32', use_last_error=True)
    kernel.CreateFileW.argtypes = [wintypes.LPCWSTR, wintypes.DWORD, wintypes.DWORD, wintypes.LPVOID, wintypes.DWORD, wintypes.DWORD, wintypes.HANDLE]
    kernel.CreateFileW.restype = wintypes.HANDLE
    kernel.CloseHandle.argtypes = [wintypes.HANDLE]
    # Ask for DELETE permission without deleting. Existing handles that deny
    # delete-sharing, or inadequate permissions, conservatively exclude it.
    handle = kernel.CreateFileW(str(path), 0x10000, 7, None, 3, 0x80, None)
    if handle == wintypes.HANDLE(-1).value:
        return True
    kernel.CloseHandle(handle)
    return False


def disposable(relative):
    path = Path(relative)
    return any(path.is_relative_to(Path(prefix)) for prefix in DISPOSABLE) and not any(word in path.name.lower() for word in PROTECTED_NAMES) and path.suffix.lower() not in ('.exe', '.dll', '.zip', '.7z', '.bak', '.trace')


def digest(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def inventory(root=ROOT, active=None):
    root = root.resolve(strict=True)
    active = active_paths() if active is None else active
    categories, files = [], []
    prefixes = INVENTORY + tuple(p.relative_to(root).as_posix() for p in (root/'.tools').glob('*') if p.is_dir() and ('evidence' in p.name or 'archive' in p.name))
    for prefix in dict.fromkeys(prefixes):
        folder = root/prefix
        category = {'path': prefix, 'bytes': 0, 'files': 0, 'protected': prefix not in DISPOSABLE, 'incomplete': False}
        categories.append(category)
        if not folder.exists():
            continue
        if folder.is_symlink() or (hasattr(folder, 'is_junction') and folder.is_junction()):
            category['incomplete'] = True
            continue
        active_category = any(p == folder or p.is_relative_to(folder) for p in active)
        for parent, directories, names in os.walk(folder, followlinks=False):
            directories[:] = [d for d in directories if not (Path(parent)/d).is_symlink() and not (hasattr(Path(parent)/d, 'is_junction') and (Path(parent)/d).is_junction())]
            for name in names:
                if category['files'] >= 200000:
                    category['incomplete'] = True
                    directories.clear()
                    break
                path = Path(parent)/name
                relative = path.relative_to(root).as_posix()
                try:
                    path = canonical_file(root, relative)
                    st = path.stat()
                    category['files'] += 1
                    category['bytes'] += st.st_size
                    if disposable(relative):
                        eligible = not active_category and not locked(path)
                        files.append({'path': relative, 'bytes': st.st_size, 'mtime_ns': st.st_mtime_ns,
                                      'sha256': digest(path) if eligible else None,
                                      'eligible': eligible, 'reason': 'generated_cache' if eligible else 'active_or_locked'})
                except (OSError, ValueError):
                    category['incomplete'] = True
    return {'schema_version': 1, 'workspace': str(root), 'preview_only': True,
            'categories': categories, 'files': files,
            'protected': 'Source, toolchains, dependencies, installed/rollback binaries, settings, credentials, backups, raw traces, evidence archives and manifests are never selected.'}


def apply_manifest(manifest, root=ROOT, active=None):
    root = root.resolve(strict=True)
    if manifest.get('schema_version') != 1 or manifest.get('workspace') != str(root):
        raise ValueError('manifest belongs to a different workspace/schema')
    active = active_paths() if active is None else active
    results = []
    for row in manifest['files']:
        relative = row['path']
        if not row.get('eligible') or not disposable(relative):
            continue
        try:
            path = canonical_file(root, relative)
            st = path.stat()
            prefix = next(root/p for p in DISPOSABLE if path.is_relative_to(root/p))
            if any(p == path or p.is_relative_to(prefix) for p in active) or locked(path):
                raise ValueError('active or locked')
            if st.st_size != row['bytes'] or st.st_mtime_ns != row['mtime_ns'] or digest(path) != row['sha256']:
                raise ValueError('changed since preview')
            # Re-resolve immediately before deletion; never recurse or remove directories.
            if canonical_file(root, relative) != path:
                raise ValueError('path changed')
            path.unlink()
            results.append({'path': relative, 'status': 'removed'})
        except (OSError, ValueError) as error:
            results.append({'path': relative, 'status': 'skipped', 'reason': type(error).__name__})
    return results


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, help='Save preview manifest instead of printing it')
    parser.add_argument('--apply', type=Path, help='Reviewed preview manifest; no deletion without its exact SHA-256')
    parser.add_argument('--confirm-sha256', help='SHA-256 of the exact reviewed manifest')
    args = parser.parse_args()
    if args.apply:
        raw = args.apply.read_bytes()
        if hashlib.sha256(raw).hexdigest() != args.confirm_sha256:
            parser.error('exact reviewed manifest SHA-256 is required')
        result = apply_manifest(json.loads(raw))
    else:
        result = inventory()
    encoded = json.dumps(result, indent=2)+'\n'
    if args.output:
        args.output.write_text(encoded, encoding='utf-8')
    else:
        sys.stdout.write(encoded)
