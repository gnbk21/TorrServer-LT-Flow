"""Record the exact optional Android runtime graph, bundled AARs and native blobs."""
import argparse
import hashlib
import json
from pathlib import Path
from urllib.parse import quote
import zipfile

from prepare_just_player import COMMIT


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
    args = parser.parse_args()
    args.output.write_text(json.dumps(generate(json.loads(args.inventory.read_text())), indent=2)+'\n', encoding='utf-8')
