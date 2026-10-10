import hashlib
import json
from pathlib import Path
import tempfile
import unittest
import zipfile

from player_sbom import generate


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
