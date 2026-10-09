from pathlib import Path
from tempfile import TemporaryDirectory
import unittest

from runtime_notices import collect


class RuntimeNotices(unittest.TestCase):
    def test_windows_originals_and_referenced_license_text_are_preserved(self):
        with TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ('gcc-mingw-w64-base', 'mingw-w64-common'):
                (root / 'doc' / name).mkdir(parents=True)
                (root / 'doc' / name / 'copyright').write_text(name + ' original copyright', encoding='utf-8')
            (root / 'licenses').mkdir()
            (root / 'licenses/GPL-3').write_text('original GPL text', encoding='utf-8')
            notices = collect('windows-amd64', root / 'doc', root / 'licenses')
            self.assertIn('original GPL text', notices)
            self.assertIn('gcc-mingw-w64-base original copyright', notices)
            (root / 'doc/mingw-w64-common/copyright').unlink()
            with self.assertRaises(FileNotFoundError):
                collect('windows-amd64', root / 'doc', root / 'licenses')

    def test_android_requires_original_notice_and_keeps_ndk_identity(self):
        with TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'source.properties').write_text('Pkg.Revision = fixture', encoding='utf-8')
            with self.assertRaises(ValueError):
                collect('android-arm64', ndk_root=root)
            (root / 'NOTICE').write_text('original Android and LLVM runtime notices', encoding='utf-8')
            for target in ('android-arm64', 'android-armv7'):
                notices = collect(target, ndk_root=root)
                self.assertIn('Pkg.Revision = fixture', notices)
                self.assertIn('original Android and LLVM runtime notices', notices)


if __name__ == '__main__':
    unittest.main()
