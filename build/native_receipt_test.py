import json
import hashlib
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from native_receipt import artifacts, verify


class ReceiptTests(unittest.TestCase):
    def test_identity_and_installed_bytes_both_gate_cache_reuse(self):
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory)
            names=['lib/libtorrent-rasterbar.a','lib/libssl.a','lib/libcrypto.a','lib/libusrsctp.a',
                   'lib/libjuice-openssl.a','lib/pkgconfig/libtorrent-rasterbar.pc','include/libtorrent/version.hpp']
            for name in names:
                p=root/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_bytes(b'owned fixture')
            expected={'compiler':'fixture','recipe':'current'}
            (root/'.flow-native-receipt.json').write_text(json.dumps({'identity':expected,'artifacts':artifacts(root)}))
            verify(root,expected)
            with self.assertRaises(ValueError):verify(root,{'compiler':'changed'})
            (root/names[0]).write_bytes(b'tampered archive')
            with self.assertRaises(ValueError):verify(root,expected)


class SourceChecksumTests(unittest.TestCase):
    def check(self, content, expected):
        script=(Path(__file__).resolve().parent/'_fetch_sources.sh').read_text()
        code=script.split('# FLOW_OPENSSL_SOURCE_CHECK\n',1)[1].split('# END_FLOW_OPENSSL_SOURCE_CHECK',1)[0]
        with tempfile.TemporaryDirectory() as directory:
            archive=Path(directory)/'source.tar.gz'
            archive.write_bytes(content)
            with patch('sys.argv',['source-check',str(archive),expected]):
                exec(compile(code,'<OpenSSL source verification>','exec'),{})

    def test_reviewed_archive_is_accepted(self):
        self.check(b'reviewed source',hashlib.sha256(b'reviewed source').hexdigest())

    def test_changed_archive_is_rejected_before_extraction(self):
        with self.assertRaisesRegex(SystemExit,'checksum mismatch'):
            self.check(b'changed source',hashlib.sha256(b'reviewed source').hexdigest())

    def test_unreviewed_checksum_is_rejected(self):
        with self.assertRaisesRegex(SystemExit,'reviewed OPENSSL_SHA256'):
            self.check(b'source','unknown')
