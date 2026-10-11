import hashlib
import gzip
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest

from player_sources import package, source_tree, validate_lock, verify_archive, verify_bundle
from prepare_just_player import COMMIT, recipe_directory


class DecoderSourceTests(unittest.TestCase):
    def test_recompression_preserves_content_pin_but_source_or_mode_change_fails(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            _, lock, _, cache = self.fixture(root)
            item = lock['sources'][0]
            archive = cache / item['archive']
            item['source_tree'] = source_tree(archive, {'bytes': 1024, 'entries': 2})
            item['max_archive_bytes'] = 4096
            original = archive.read_bytes()
            archive.write_bytes(gzip.compress(gzip.decompress(original), mtime=12345))
            self.assertNotEqual(hashlib.sha256(archive.read_bytes()).hexdigest(), item['sha256'])
            receipt = verify_archive(item, archive)
            self.assertEqual(receipt['archive_sha256'], hashlib.sha256(archive.read_bytes()).hexdigest())
            # A changed executable bit is part of the source identity, not compression.
            for data, mode in [(b'original source', 0o755), (b'changed source!', 0o644)]:
                with tarfile.open(archive, 'w:gz') as output:
                    for name, content in [('codec/LICENSE', b'original component terms'), ('codec/source.c', data)]:
                        member = tarfile.TarInfo(name)
                        member.mode = mode if name.endswith('.c') else 0o644
                        member.size = len(content)
                        output.addfile(member, io.BytesIO(content))
                with self.assertRaisesRegex(ValueError, 'checksum changed'):
                    verify_archive(item, archive)

    def test_source_bundle_and_repository_recipe_discovery(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with self.assertRaisesRegex(ValueError, 'recipe is missing'):
                recipe_directory(root)
            published = root / 'flow-patch'
            published.mkdir()
            (published / 'flow-player.patch').write_text('published patch')
            self.assertEqual(recipe_directory(root), published)
            checkout = root / 'just-player'
            checkout.mkdir()
            (checkout / 'flow-player.patch').write_text('checkout patch')
            self.assertEqual(recipe_directory(root), checkout)

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
