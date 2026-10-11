"""Record the exact optional Android runtime graph, bundled AARs and native blobs."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import tarfile
from urllib.parse import quote
import zipfile

from prepare_just_player import COMMIT
from player_sources import verify_bundle


def source_manifest(directory, inventory, flow_commit):
    """Bind the published recipe and bundled upstream bytes to both commits.

    A source archive containing decoder binaries is not corresponding decoder
    source. Preserve that distinction instead of inferring codec licenses.
    """
    if not re.fullmatch(r'[0-9a-f]{40}', flow_commit):
        raise ValueError('Flow source commit must be a full lowercase Git identity')
    required = ('upstream.tar', 'UPSTREAM_LICENSE.txt', 'FLOW_PATCH_LICENSE.txt',
                'prepare_just_player.py', 'player_patch_check.py', 'player_sbom.py', 'player_sources.py',
                'BUILD_WORKFLOW.yml', 'flow-patch/inventory.gradle',
                'flow-patch/SOURCE_BUNDLE.md')
    if any(not (directory / name).is_file() for name in required):
        raise ValueError('Player source bundle is missing a required recipe or license')
    with tarfile.open(directory / 'upstream.tar') as archive:
        members = archive.getmembers()
        names = [member.name for member in members]
        if len(names) != len(set(names)):
            raise ValueError('Duplicate upstream archive entries')
        license_file = archive.extractfile('LICENSE')
        if license_file is None or license_file.read() != (directory / 'UPSTREAM_LICENSE.txt').read_bytes():
            raise ValueError('Upstream source license does not match its archive')
        for artifact in inventory['bundled']:
            name = artifact['name']
            if Path(name).name != name or not name.startswith('lib-') or not name.endswith('.aar'):
                raise ValueError('Unexpected bundled Android artifact name')
            member = archive.getmember('app/libs/' + name)
            if not member.isfile():
                raise ValueError('Bundled Android artifact is not an archived regular file')
            with archive.extractfile(member) as source:
                if hashlib.file_digest(source, 'sha256').hexdigest() != artifact['sha256']:
                    raise ValueError('Archived bundled artifact differs from the built inventory')
    files = []
    # Windows Path ordering folds case; published Linux manifests must have
    # the same order when verified or regenerated on the owner's Windows PC.
    for path in sorted(directory.rglob('*'), key=lambda p: p.relative_to(directory).as_posix()):
        if path.is_symlink():
            raise ValueError('Player source bundle must not contain external symlinks')
        if not path.is_file() or path == directory / 'SOURCE_MANIFEST.json':
            continue
        with path.open('rb') as source:
            digest = hashlib.file_digest(source, 'sha256').hexdigest()
        files.append({'name': path.relative_to(directory).as_posix(),
                      'sha256': digest, 'bytes': path.stat().st_size})
    decoder = None
    if (directory / 'decoder-sources').exists():
        decoder = verify_bundle(directory / 'decoder-sources', inventory)
    return {'schema_version': 1, 'upstream_commit': COMMIT,
            'flow_commit': flow_commit, 'files': files,
            'native_source_provenance_verified': bool(decoder),
            'native_corresponding_source_verified': False,
            'scope': 'Pinned application archive, exact bundled AAR bytes, Flow patch and build recipe; matched decoder sources/notices when included. Independent native rebuild and redistribution compatibility remain separate.'}


def generate(inventory):
    if inventory['schema_version'] != 1:
        raise ValueError('Unsupported resolved Android inventory')
    root = f'pkg:github/moneytoo/Player@{COMMIT}#flow-patch'
    refs, components, edges = {}, {}, {}
    for item in inventory['components']:
        if item['id'] == inventory['root']:
            refs[item['id']] = root
            continue
        if item.get('group') and item['group'] != 'Player':
            ref = f"pkg:maven/{quote(item['group'], safe='')}/{quote(item['name'], safe='')}@{quote(item['version'], safe='')}"
        else:
            ref = f"flow-player-source:{quote(item['id'], safe='')}@{COMMIT}"
        refs[item['id']] = ref
        components[ref] = {'type': 'library', 'name': item['name'] or item['id'], 'version': item['version'] or COMMIT, 'bom-ref': ref}
        if ref.startswith('pkg:'):
            components[ref]['purl'] = ref
    for item in inventory['components']:
        edges[refs[item['id']]] = {refs[key] for key in item['dependencies']}
    edges.setdefault(root, set())
    for artifact in inventory['artifacts']:
        ref = refs.get(artifact['component'])
        if ref and ref != root:
            components[ref].setdefault('hashes', []).append({'alg': 'SHA-256', 'content': artifact['sha256']})
    for artifact in inventory['bundled']:
        path = Path(artifact['file'])
        if hashlib.sha256(path.read_bytes()).hexdigest() != artifact['sha256']:
            raise ValueError('Bundled artifact changed after resolution')
        ref = f"flow-player-bundled:{quote(artifact['name'], safe='')}@sha256:{artifact['sha256']}"
        components[ref] = {'type': 'library', 'name': artifact['name'], 'version': COMMIT, 'bom-ref': ref,
                           'hashes': [{'alg': 'SHA-256', 'content': artifact['sha256']}],
                           'properties': [{'name': 'flow:inventory-evidence', 'value': 'Exact bundled upstream AAR; version identifies the pinned source commit'}]}
        edges[root].add(ref)
        edges[ref] = set()
        with zipfile.ZipFile(path) as archive:
            for name in sorted(archive.namelist()):
                if not name.startswith('jni/') or not name.endswith('.so'):
                    continue
                digest = hashlib.sha256(archive.read(name)).hexdigest()
                native = f'{ref}/{quote(name, safe="/")}@sha256:{digest}'
                components[native] = {'type': 'file', 'name': name, 'bom-ref': native, 'hashes': [{'alg': 'SHA-256', 'content': digest}]}
                edges[ref].add(native)
                edges[native] = set()
    return {'bomFormat': 'CycloneDX', 'specVersion': '1.6', 'version': 1,
            'metadata': {'component': {'type': 'application', 'name': 'Flow Player (experimental)', 'version': COMMIT, 'bom-ref': root},
                         'properties': [{'name': 'flow:inventory-evidence', 'value': 'Resolved latestUniversalDebugRuntimeClasspath, bundled AARs and native library SHA-256 hashes. No inferred license claims.'}]},
            'components': [components[key] for key in sorted(components)],
            'dependencies': [{'ref': key, 'dependsOn': sorted(edges.get(key, set()))} for key in sorted(components.keys() | {root})]}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('inventory', type=Path)
    parser.add_argument('output', type=Path)
    parser.add_argument('--source-bundle', type=Path)
    parser.add_argument('--flow-commit')
    args = parser.parse_args()
    if bool(args.source_bundle) != bool(args.flow_commit):
        parser.error('--source-bundle and --flow-commit must be provided together')
    inventory = json.loads(args.inventory.read_text(encoding='utf-8'))
    report = generate(inventory)
    if args.source_bundle:
        if not re.fullmatch(r'[0-9a-f]{40}', args.flow_commit):
            parser.error('Flow source commit must be a full lowercase Git identity')
        verify_bundle(args.source_bundle / 'decoder-sources', inventory)
        report['metadata']['properties'].append({'name': 'flow:source-commit', 'value': args.flow_commit})
    args.output.write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    if args.source_bundle:
        manifest = source_manifest(args.source_bundle, inventory, args.flow_commit)
        (args.source_bundle / 'SOURCE_MANIFEST.json').write_text(json.dumps(manifest, indent=2)+'\n', encoding='utf-8')
