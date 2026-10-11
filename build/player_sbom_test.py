import hashlib
import json
import io
from pathlib import Path
import tempfile
import tarfile
import unittest
import zipfile

from player_sbom import generate, source_manifest
from sbom_schema_check import check_generated_inventory


class PlayerInventoryTests(unittest.TestCase):
    def test_resolved_graph_bundled_native_hash_and_private_paths(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory) / 'lib-decoder.aar'
            with zipfile.ZipFile(archive, 'w') as output:
                output.writestr('jni/arm64-v8a/libdecoder.so', b'actual native bytes')
                output.writestr('classes.jar', b'not a native component')
            inventory = {
                'schema_version': 1, 'root': 'project :app',
                'components': [
                    {'id': 'project :app', 'dependencies': ['androidx.media3:media3-exoplayer:1.11.1']},
                    {'id': 'androidx.media3:media3-exoplayer:1.11.1', 'group': 'androidx.media3',
                     'name': 'media3-exoplayer', 'version': '1.11.1', 'dependencies': []}],
                'artifacts': [], 'bundled': [{'name': archive.name, 'file': str(archive),
                                             'sha256': hashlib.sha256(archive.read_bytes()).hexdigest()}]}
            report = generate(inventory)
            check_generated_inventory(report)
            self.assertEqual(report, generate(inventory))
            refs = {item['bom-ref'] for item in report['components']}
            refs.add(report['metadata']['component']['bom-ref'])
            for edge in report['dependencies']:
                self.assertIn(edge['ref'], refs)
                self.assertTrue(set(edge['dependsOn']) <= refs)
            native = [item for item in report['components'] if item['type'] == 'file']
            self.assertEqual(len(native), 1)
            self.assertEqual(native[0]['hashes'][0]['content'], hashlib.sha256(b'actual native bytes').hexdigest())
            self.assertNotIn(directory, json.dumps(report))
            archive.write_bytes(b'tampered')
            with self.assertRaisesRegex(ValueError, 'changed after resolution'):
                generate(inventory)

    def test_unknown_inventory_version_is_rejected(self):
        with self.assertRaisesRegex(ValueError, 'Unsupported'):
            generate({'schema_version': 2})

    def test_source_bundle_binds_recipe_licenses_and_built_aar(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ('UPSTREAM_LICENSE.txt', 'FLOW_PATCH_LICENSE.txt', 'prepare_just_player.py',
                         'player_patch_check.py', 'player_sbom.py', 'player_sources.py', 'BUILD_WORKFLOW.yml',
                         'flow-patch/inventory.gradle', 'flow-patch/SOURCE_BUNDLE.md'):
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text('original upstream license' if name == 'UPSTREAM_LICENSE.txt' else name)
            with tarfile.open(root / 'upstream.tar', 'w') as archive:
                for name, data in (('LICENSE', b'original upstream license'),
                                   ('app/libs/lib-decoder.aar', b'exact built bytes')):
                    member = tarfile.TarInfo(name)
                    member.size = len(data)
                    archive.addfile(member, io.BytesIO(data))
            inventory = {'bundled': [{'name': 'lib-decoder.aar', 'sha256': hashlib.sha256(b'exact built bytes').hexdigest()}]}
            commit = 'a' * 40
            (root / 'flow-patch/SOURCE_MANIFEST.json').write_text('nested file must be inventoried')
            (root / 'a-notice.txt').write_text('lowercase path')
            (root / 'Z-notice.txt').write_text('uppercase path')
            manifest = source_manifest(root, inventory, commit)
            names = [item['name'] for item in manifest['files']]
            self.assertEqual(names, sorted(names))
            self.assertEqual(manifest['flow_commit'], commit)
            self.assertFalse(manifest['native_corresponding_source_verified'])
            self.assertIn('flow-patch/SOURCE_MANIFEST.json', [item['name'] for item in manifest['files']])
            self.assertNotIn(directory, json.dumps(manifest))
            for item in manifest['files']:
                self.assertEqual(item['sha256'], hashlib.sha256((root / item['name']).read_bytes()).hexdigest())
            (root / 'SOURCE_MANIFEST.json').write_text(json.dumps(manifest))
            self.assertEqual(manifest, source_manifest(root, inventory, commit))
            inventory['bundled'][0]['sha256'] = '0' * 64
            with self.assertRaisesRegex(ValueError, 'differs from the built inventory'):
                source_manifest(root, inventory, commit)
            with self.assertRaisesRegex(ValueError, 'full lowercase Git'):
                source_manifest(root, inventory, 'unbound')
            (root / 'FLOW_PATCH_LICENSE.txt').unlink()
            with self.assertRaisesRegex(ValueError, 'missing a required'):
                source_manifest(root, inventory, commit)

    def test_source_bundle_rejects_mismatched_upstream_license(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ('UPSTREAM_LICENSE.txt', 'FLOW_PATCH_LICENSE.txt', 'prepare_just_player.py',
                         'player_patch_check.py', 'player_sbom.py', 'player_sources.py', 'BUILD_WORKFLOW.yml',
                         'flow-patch/inventory.gradle', 'flow-patch/SOURCE_BUNDLE.md'):
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text('preserved recipe')
            with tarfile.open(root / 'upstream.tar', 'w') as archive:
                member = tarfile.TarInfo('LICENSE')
                member.size = 7
                archive.addfile(member, io.BytesIO(b'changed'))
            with self.assertRaisesRegex(ValueError, 'license does not match'):
                source_manifest(root, {'bundled': []}, 'a' * 40)
