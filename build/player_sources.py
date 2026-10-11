"""Preserve checksum-pinned decoder sources and original notices for Flow Player."""
import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import shutil
import tarfile
import time
from urllib.parse import urlsplit
from urllib.request import urlopen
import zipfile

from prepare_just_player import COMMIT, recipe_directory

MAX_ARCHIVE = 512 * 1024 * 1024
MAX_TOTAL = 1024 * 1024 * 1024
MAX_NOTICE = 2 * 1024 * 1024
HOSTS = {'codeload.github.com', 'github.com', 'android.googlesource.com', 'bitbucket.org'}


def digest(path):
    with path.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def validate_lock(lock):
    if lock.get('schema_version') != 1 or lock.get('player_commit') != COMMIT:
        raise ValueError('Decoder source lock does not match the pinned player')
    names, components = set(), set()
    total = 0
    for item in lock['sources']:
        name = item['archive']
        if (not re.fullmatch(r'[a-zA-Z0-9_.-]+\.(tar\.gz|zip)', name) or name in names
                or item['name'] in components
                or not re.fullmatch(r'[a-zA-Z0-9_-]+(?:\.[a-zA-Z0-9_-]+)*', item['name'])):
            raise ValueError('Unsafe or duplicate decoder source archive')
        names.add(name)
        components.add(item['name'])
        address = urlsplit(item['url'])
        if (address.scheme != 'https' or address.hostname not in HOSTS
                or address.username or address.password or address.port not in (None, 443)
                or address.query or address.fragment):
            raise ValueError('Unexpected decoder source URL')
        if not re.fullmatch(r'[0-9a-f]{64}', item['sha256']):
            raise ValueError('Decoder source checksum is missing')
        if not 0 < item['bytes'] <= MAX_ARCHIVE or not item['required_notices']:
            raise ValueError('Decoder source size or notice list is invalid')
        tree = item.get('source_tree')
        if tree is not None:
            if (not re.fullmatch(r'[0-9a-f]{64}', tree['sha256'])
                    or not 0 < tree['bytes'] <= MAX_TOTAL or not 0 < tree['entries'] <= 100000
                    or not item['bytes'] <= item['max_archive_bytes'] <= MAX_ARCHIVE
                    or not name.endswith('.tar.gz')):
                raise ValueError('Decoder source content pin or size limit is invalid')
        total += item.get('max_archive_bytes', item['bytes'])
    if total > MAX_TOTAL:
        raise ValueError('Decoder source bundle exceeds its size limit')


def validate_bundled(lock, inventory):
    actual = {item['name']: item['sha256'] for item in inventory['bundled']}
    if len(actual) != len(inventory['bundled']) or actual != lock['bundled_aar_sha256']:
        raise ValueError('Bundled AAR bytes do not match the traced upstream build')


def download(item, target):
    """Bound bytes and time; failed downloads never replace a verified archive."""
    temporary = target.with_suffix(target.suffix + '.partial')
    created = False
    deadline = time.monotonic() + 600
    try:
        with urlopen(item['url'], timeout=60) as response, temporary.open('xb') as output:
            created = True
            count = 0
            while chunk := response.read(1024 * 1024):
                count += len(chunk)
                if count > item.get('max_archive_bytes', item['bytes']) or time.monotonic() > deadline:
                    raise ValueError('Decoder source download exceeded its limit: ' + item['name'])
                output.write(chunk)
        verify_archive(item, temporary)
        temporary.replace(target)
    finally:
        if created:
            temporary.unlink(missing_ok=True)


def source_tree(path, limit):
    """Pin file contents, link targets and executable bits, excluding compression/time."""
    rows, seen, count = [], set(), 0
    with tarfile.open(path) as archive:
        for member in archive:
            if member.isdir():
                continue
            name = member.name
            relative = PurePosixPath(name)
            if relative.is_absolute() or '..' in relative.parts or '\\' in name or ':' in name or name in seen:
                raise ValueError('Unsafe or duplicate decoder source entry')
            seen.add(name)
            if len(seen) > limit['entries']:
                raise ValueError('Decoder source entry count exceeded its pin')
            if member.isfile():
                count += member.size
                if count > limit['bytes']:
                    raise ValueError('Decoder source expanded bytes exceeded their pin')
                with archive.extractfile(member) as source:
                    checksum = hashlib.file_digest(source, 'sha256').hexdigest()
                rows.append({'name': name, 'type': 'file', 'executable': bool(member.mode & 0o111),
                             'bytes': member.size, 'sha256': checksum})
            elif member.issym() or member.islnk():
                rows.append({'name': name, 'type': 'symlink' if member.issym() else 'hardlink',
                             'target': member.linkname})
            else:
                raise ValueError('Unsupported decoder source archive entry')
    data = json.dumps(sorted(rows, key=lambda row: row['name']), sort_keys=True, separators=(',', ':')).encode()
    return {'sha256': hashlib.sha256(data).hexdigest(), 'bytes': count, 'entries': len(rows)}


def verify_archive(item, path):
    if (path.is_symlink() or not path.is_file()
            or not 0 < path.stat().st_size <= item.get('max_archive_bytes', item['bytes'])):
        raise ValueError('Decoder source archive size or file type changed')
    actual = digest(path)
    if actual != item['sha256'] or path.stat().st_size != item['bytes']:
        try:
            expected = item.get('source_tree')
            if expected is None or source_tree(path, expected) != expected:
                raise ValueError('Decoder source archive checksum changed')
        except (tarfile.TarError, OSError) as error:
            raise ValueError('Decoder source archive checksum changed') from error
    return {'archive_sha256': actual, 'archive_bytes': path.stat().st_size}


def notices(item, path, output=None):
    """Read archives without extracting source paths or following archived links."""
    required = set(item['required_notices'])
    found, seen, report = set(), set(), []

    def preserve(name, size, read):
        relative = PurePosixPath(name)
        if (relative.is_absolute() or '..' in relative.parts or '\\' in name
                or ':' in name or name in seen):
            raise ValueError('Unsafe or duplicate decoder source entry')
        seen.add(name)
        base = relative.name.upper()
        if name not in required and not base.startswith(('LICENSE', 'COPYING', 'NOTICE', 'PATENTS')):
            return
        if not 0 < size <= MAX_NOTICE:
            raise ValueError('Decoder notice exceeds its size limit or is empty')
        data = read()
        if len(data) != size:
            raise ValueError('Decoder notice archive entry is truncated')
        # Keep source headers intact when a library (PFFFT) has no separate license.
        destination = PurePosixPath(item['name']) / relative
        if output is not None:
            target = output / destination
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(data)
        found.add(name)
        report.append({'source_entry': name, 'file': destination.as_posix(),
                       'sha256': hashlib.sha256(data).hexdigest(), 'bytes': size})

    if zipfile.is_zipfile(path):
        with zipfile.ZipFile(path) as archive:
            for member in archive.infolist():
                if not member.is_dir():
                    preserve(member.filename, member.file_size, lambda m=member: archive.read(m))
    else:
        with tarfile.open(path) as archive:
            for member in archive:
                if member.isfile():
                    def read(m=member):
                        with archive.extractfile(m) as source:
                            return source.read(MAX_NOTICE + 1)
                    preserve(member.name, member.size, read)
    if not required <= found:
        raise ValueError('Decoder source archive is missing a required notice')
    return report


def verify_bundle(directory, inventory):
    lock = json.loads((directory / 'decoder-sources.json').read_text(encoding='utf-8'))
    validate_lock(lock)
    validate_bundled(lock, inventory)
    report = json.loads((directory / 'DECODER_SOURCE_REPORT.json').read_text(encoding='utf-8'))
    if (report.get('schema_version') != 1 or report.get('player_commit') != COMMIT
            or report.get('upstream_run_url') != lock['upstream_run_url']
            or report.get('bundled_aar_sha256') != lock['bundled_aar_sha256']
            or report.get('native_source_provenance_verified') is not True
            or report.get('native_corresponding_source_verified') is not False
            or len(report['sources']) != len(lock['sources'])):
        raise ValueError('Decoder source report does not match its lock')
    for item, row in zip(lock['sources'], report['sources']):
        if any(row.get(key) != value for key, value in item.items()):
            raise ValueError('Decoder source report changed its pinned recipe')
        archive = directory / 'archives' / item['archive']
        if any(row.get(key) != value for key, value in verify_archive(item, archive).items()):
            raise ValueError('Decoder source report changed its actual archive checksum')
        if row['notices'] != notices(item, archive):
            raise ValueError('Decoder notice report differs from the original source')
        required = set(item['required_notices'])
        found = set()
        for notice in row['notices']:
            path = PurePosixPath(notice['file'])
            if path.is_absolute() or '..' in path.parts or '\\' in notice['file'] or ':' in notice['file']:
                raise ValueError('Unsafe preserved decoder notice path')
            target = directory / 'notices' / path
            if target.is_symlink() or not target.is_file() or digest(target) != notice['sha256']:
                raise ValueError('Preserved decoder notice changed')
            found.add(notice['source_entry'])
        if not required <= found:
            raise ValueError('Preserved decoder notice is missing')
    return report


def package(lock_path, inventory, output, cache=None):
    lock = json.loads(lock_path.read_text(encoding='utf-8'))
    validate_lock(lock)
    validate_bundled(lock, inventory)
    output.mkdir(parents=True, exist_ok=False)
    archives = output / 'archives'
    archives.mkdir()
    rows = []
    for item in lock['sources']:
        target = archives / item['archive']
        if cache is not None:
            cached = cache / item['archive']
            verify_archive(item, cached)
            shutil.copyfile(cached, target)
        else:
            download(item, target)
        rows.append({**item, **verify_archive(item, target),
                     'notices': notices(item, target, output / 'notices')})
    shutil.copyfile(lock_path, output / 'decoder-sources.json')
    report = {'schema_version': 1, 'player_commit': COMMIT,
              'upstream_run_url': lock['upstream_run_url'],
              'bundled_aar_sha256': lock['bundled_aar_sha256'], 'sources': rows,
              'native_source_provenance_verified': True,
              'native_corresponding_source_verified': False,
              'scope': 'Exact upstream AAR build matched; immutable decoder and declared dependency sources and original notices preserved. Independent rebuild, resolved Bazel/toolchain closure and redistribution compatibility remain separate.'}
    (output / 'DECODER_SOURCE_REPORT.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    return report


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('inventory', type=Path)
    parser.add_argument('output', type=Path)
    parser.add_argument('--lock', type=Path, default=recipe_directory() / 'decoder-sources.json')
    parser.add_argument('--cache', type=Path, help='Use only exact verified local source archives')
    args = parser.parse_args()
    report = package(args.lock, json.loads(args.inventory.read_text(encoding='utf-8')), args.output, args.cache)
    print(json.dumps({'archives': len(report['sources']), 'native_source_provenance_verified': True}))
