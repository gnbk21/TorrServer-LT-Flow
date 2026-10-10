"""Reproducible CycloneDX inventory from exact release binaries and build inputs."""
import json
import re
import subprocess
from urllib.parse import quote


def component(name, version, ecosystem, license_name=None, properties=()):
    purl = f'pkg:{ecosystem}/{quote(name, safe="/")}@{quote(version, safe="")}'
    result = {'type': 'library', 'name': name, 'version': version, 'bom-ref': purl, 'purl': purl}
    if license_name:
        result['licenses'] = [{'license': {'name': license_name}}]
    if properties:
        result['properties'] = [{'name': key, 'value': str(value)} for key, value in properties]
    return result


def generate(root, binaries, tag, commit, timestamp, native_root):
    components, dependencies = {}, {}
    app_ref = f'pkg:github/gnbk21/TorrServer-LT-Flow@{commit}'
    app_dependencies = set()
    for binary in binaries:
        info = subprocess.check_output(['go', 'version', '-m', str(binary)], text=True)
        match = re.search(r':\s+(go\S+)', info.splitlines()[0])
        if not match:
            raise ValueError('Binary has no Go runtime identity')
        runtime = component('golang/go', match[1], 'generic', 'BSD-3-Clause')
        components[runtime['bom-ref']] = runtime
        app_dependencies.add(runtime['bom-ref'])
        for name, version in re.findall(r'^\s*dep\s+(\S+)\s+(\S+)', info, re.M):
            item = component(name, version, 'golang', properties=[('flow:inventory-evidence', 'go version -m')])
            components[item['bom-ref']] = item
            app_dependencies.add(item['bom-ref'])
    pins = (root/'build/_common.sh').read_text(encoding='utf-8')
    for name, variable, license_name in [('boost', 'BOOST_VERSION', 'BSL-1.0'), ('openssl', 'OPENSSL_VERSION', 'Apache-2.0'), ('libtorrent-rasterbar', 'LIBTORRENT_TAG', 'BSD-3-Clause')]:
        match = re.search(r'^'+variable+r'=\$\{'+variable+r':-([^}]+)\}', pins, re.M)
        if not match:
            raise ValueError(f'Missing native pin: {variable}')
        item = component(name, match[1], 'generic', license_name, [('flow:inventory-evidence', 'pinned native build; see BUILDINFO.json and NATIVE_AND_GO_NOTICES.txt')])
        components[item['bom-ref']] = item
        app_dependencies.add(item['bom-ref'])
    # The native source tree is fetched at immutable pins before packaging.
    # Record recursive dependencies including the WebRTC transport libraries;
    # distinguish source inventory from claims about linked machine code.
    submodules = subprocess.check_output(['git', '-C', str(native_root/'libtorrent'), 'submodule', 'status', '--recursive'], text=True)
    for line in submodules.splitlines():
        match = re.fullmatch(r' ([a-f0-9]{40}) (\S+)(?: \(.*\))?', line)
        if not match:
            raise ValueError('Native submodule is missing, modified or has an invalid identity')
        revision, path = match.groups()
        item = component(path, revision, 'generic', properties=[('flow:inventory-evidence', 'Immutable recursive native source revision; original license in NATIVE_AND_GO_NOTICES.txt'), ('flow:native-source-path', path)])
        components[item['bom-ref']] = item
        app_dependencies.add(item['bom-ref'])
    web = json.loads((root/'server/web/pages/template/pages/DEPENDENCIES.json').read_text(encoding='utf-8'))
    if web['schema_version'] != 1:
        raise ValueError('Unsupported web inventory')
    web_refs = {}
    for package in web['packages']:
        item = component(package['name'], package['version'], 'npm', package['license'], [('flow:original-notice-sha256', package['notice_sha256'])])
        components[item['bom-ref']] = item
        web_refs[package['key']] = item['bom-ref']
    for package in web['packages']:
        dependencies[web_refs[package['key']]] = sorted({web_refs[key] for key in package['dependencies']})
    app_dependencies.update(web_refs[key] for key in web['roots'])
    dependencies[app_ref] = sorted(app_dependencies)
    for ref in components:
        dependencies.setdefault(ref, [])
    return {'bomFormat': 'CycloneDX', 'specVersion': '1.6', 'version': 1,
            'metadata': {'timestamp': timestamp, 'component': {'type': 'application', 'name': 'TorrServer-Flow', 'version': tag, 'bom-ref': app_ref,
                'licenses': [{'license': {'id': 'GPL-3.0-only'}}], 'properties': [{'name': 'flow:source-commit', 'value': commit}]},
                'properties': [{'name': 'flow:inventory-scope', 'value': 'Exact Go modules and runtime, pinned native libraries and recursive source revisions, and resolved production web graph. Static toolchain notices remain in NATIVE_AND_GO_NOTICES.txt. Optional Flow Player has a separate resolved Android inventory.'}]},
            'components': [components[key] for key in sorted(components)],
            'dependencies': [{'ref': key, 'dependsOn': dependencies[key]} for key in sorted(dependencies)]}
