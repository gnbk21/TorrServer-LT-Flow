import json
from pathlib import Path
import tempfile
import unittest
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
