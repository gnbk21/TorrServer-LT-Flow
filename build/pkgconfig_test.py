"""Regression for linker options embedded in native build directory names."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


class PkgConfigTests(unittest.TestCase):
    def test_private_libraries_are_whole_tokens(self):
        bash = shutil.which('bash')
        if os.name == 'nt':
            candidate = Path(r'C:\Program Files\Git\bin\bash.exe')
            if candidate.exists():
                bash = str(candidate)
        if bash is None:
            self.skipTest('Bash is unavailable')
        root = Path(__file__).resolve().parent.parent
        with tempfile.TemporaryDirectory() as directory:
            fixture = Path(directory)/'libtorrent-rasterbar.pc'
            fixture.write_text('Libs.private: -L/tmp/sparse-native-layers/lib -ltry_signal -L/tmp/-lBogus/lib -lboost_system\n')
            output = subprocess.check_output([bash, '-c', 'source build/_deps.sh; private_link_libraries "$1"', 'pkgconfig-test', fixture.as_posix()], cwd=root, text=True)
            self.assertEqual(output.split(), ['-ltry_signal', '-lboost_system'])


if __name__ == '__main__':
    unittest.main()
