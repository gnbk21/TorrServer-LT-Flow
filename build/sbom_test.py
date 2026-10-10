import json
from pathlib import Path
from tempfile import TemporaryDirectory
import unittest
from unittest.mock import patch
from sbom import generate


class Inventory(unittest.TestCase):
    def test_exact_modules_native_pins_and_resolved_web_graph(self):
        with TemporaryDirectory() as directory:
            root = Path(directory)
            (root/'build').mkdir()
            (root/'build/_common.sh').write_text('BOOST_VERSION=${BOOST_VERSION:-1.85.0}\nOPENSSL_VERSION=${OPENSSL_VERSION:-3.5.7}\nLIBTORRENT_TAG=${LIBTORRENT_TAG:-v2.1.2}\n')
            web = root/'server/web/pages/template/pages'
            web.mkdir(parents=True)
            package = dict(key='react@19.0.0', name='react', version='19.0.0', license='MIT', notice_sha256='a'*64, dependencies=[])
            (web/'DEPENDENCIES.json').write_text(json.dumps(dict(schema_version=1,roots=['react@19.0.0'],packages=[package])))
            def output(command, **kwargs):
                if command[0] == 'git':
                    return ' '+'c'*40+' deps/libdatachannel\n'
                return 'fixture: go1.26.9\n\tdep\texample.org/module\tv1.2.3\th1:abc\n'
            with patch('sbom.subprocess.check_output', side_effect=output):
                first = generate(root, [Path('fixture')], 'release', 'b'*40, '2026-10-10T00:00:00Z', root/'native')
                second = generate(root, [Path('fixture')], 'release', 'b'*40, '2026-10-10T00:00:00Z', root/'native')
            self.assertEqual(first,second)
            self.assertEqual(first['bomFormat'],'CycloneDX')
            self.assertEqual({c['name'] for c in first['components']}, {'golang/go', 'example.org/module', 'boost', 'openssl', 'libtorrent-rasterbar', 'react', 'deps/libdatachannel'})
            refs = {c['bom-ref'] for c in first['components']} | {first['metadata']['component']['bom-ref']}
            self.assertEqual(len(refs),len(first['components'])+1)
            for edge in first['dependencies']:
                self.assertIn(edge['ref'],refs)
                self.assertTrue(set(edge['dependsOn']) <= refs)


if __name__ == '__main__': unittest.main()
