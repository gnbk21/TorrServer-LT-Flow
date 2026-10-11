import hashlib
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

from player_sources import package, validate_lock, verify_bundle
from prepare_just_player import COMMIT


class DecoderSourceTests(unittest.TestCase):
    def fixture(self, root, entries=None):
        cache = root / 'cache'
        cache.mkdir()
        archive = cache / 'codec.tar.gz'
        with tarfile.open(archive, 'w:gz') as output:
            for name, data in entries or [('codec/LICENSE', b'original component terms'), ('codec/source.c', b'original source')]:
                member = tarfile.TarInfo(name)
                member.size = len(data)
                output.addfile(member, io.BytesIO(data))
        lock = {'schema_version': 1, 'player_commit': COMMIT,
                'upstream_run_url': 'https://github.com/moneytoo/media/actions/runs/34632899792',
                'bundled_aar_sha256': {'lib-decoder.aar': 'a' * 64},
                'sources': [{'name': 'codec', 'archive': archive.name,
                             'url': 'https://codeload.github.com/owner/codec/tar.gz/' + 'b' * 40,
                             'revision': 'b' * 40, 'required_notices': ['codec/LICENSE'],
                             'bytes': archive.stat().st_size,
                             'sha256': hashlib.sha256(archive.read_bytes()).hexdigest()}]}
        path = root / 'lock.json'
        path.write_text(json.dumps(lock), encoding='utf-8')
        inventory = {'bundled': [{'name': 'lib-decoder.aar', 'sha256': 'a' * 64}]}
        return path, lock, inventory, cache

    def test_real_archive_binding_original_notices_and_tampering(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            path, _, inventory, cache = self.fixture(root)
            output = root / 'bundle'
            report = package(path, inventory, output, cache)
            self.assertTrue(report['native_source_provenance_verified'])
            self.assertFalse(report['native_corresponding_source_verified'])
            self.assertEqual(report, verify_bundle(output, inventory))
            self.assertNotIn(directory, json.dumps(report))
            notice = output / 'notices/codec/codec/LICENSE'
            notice.write_text('changed terms')
            with self.assertRaisesRegex(ValueError, 'notice changed'):
                verify_bundle(output, inventory)
            # Altering both notice and report cannot defeat binding to the source archive.
            report['sources'][0]['notices'][0]['sha256'] = hashlib.sha256(notice.read_bytes()).hexdigest()
            (output / 'DECODER_SOURCE_REPORT.json').write_text(json.dumps(report))
            with self.assertRaisesRegex(ValueError, 'differs from the original'):
                verify_bundle(output, inventory)

    def test_mismatch_traversal_missing_notice_and_changed_cache_rejected(self):
        for entries, error in [([('../LICENSE', b'terms')], 'Unsafe'),
                               ([('C:/LICENSE', b'terms')], 'Unsafe'),
                               ([('codec/LICENSE', b'a'), ('codec/LICENSE', b'b')], 'duplicate'),
                               ([('codec/source.c', b'source')], 'missing a required')]:
            with self.subTest(error=error), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                path, _, inventory, cache = self.fixture(root, entries)
                with self.assertRaisesRegex(ValueError, error):
                    package(path, inventory, root / 'bundle', cache)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            path, lock, inventory, cache = self.fixture(root)
            inventory['bundled'][0]['sha256'] = 'c' * 64
            with self.assertRaisesRegex(ValueError, 'AAR bytes'):
                package(path, inventory, root / 'wrong', cache)
            inventory['bundled'][0]['sha256'] = 'a' * 64
            archive = cache / lock['sources'][0]['archive']
            archive.write_bytes(b'wrong' + archive.read_bytes()[5:])
            with self.assertRaisesRegex(ValueError, 'checksum changed'):
                package(path, inventory, root / 'tampered', cache)

    def test_lock_size_url_and_identity_limits(self):
        with tempfile.TemporaryDirectory() as directory:
            _, lock, _, _ = self.fixture(Path(directory))
            validate_lock(lock)
            lock['sources'][0]['url'] = 'https://example.com/source.tar.gz'
            with self.assertRaisesRegex(ValueError, 'URL'):
                validate_lock(lock)
            lock['sources'][0]['url'] = 'https://github.com/owner/codec'
            lock['sources'][0]['bytes'] = 1024 * 1024 * 1024
            with self.assertRaisesRegex(ValueError, 'size'):
                validate_lock(lock)
