"""Regression for linker options embedded in native build directory names."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


class PkgConfigTests(unittest.TestCase):
    def test_b2_banner_reuses_runnable_tool_with_nonzero_version_exit(self):
        bash = shutil.which('bash')
        if os.name == 'nt':
            candidate = Path(r'C:\Program Files\Git\bin\bash.exe')
            if candidate.exists():
                bash = str(candidate)
        if bash is None:
            self.skipTest('Bash is unavailable')
        root = Path(__file__).resolve().parent.parent
        with tempfile.TemporaryDirectory() as directory:
            fixture = Path(directory)
            tool = fixture/'b2'
            bootstrap = fixture/'bootstrap.sh'
            receipt = fixture/'bootstrapped'
            bootstrap.write_text('#!/usr/bin/env bash\nprintf x >> bootstrapped\n', newline='\n')
            bootstrap.chmod(0o755)
            command = [bash, '-c', 'source build/_deps.sh; BOOST_DIR="$1"; ensure_b2', 'b2-test', fixture.as_posix()]
            tool.write_text('#!/usr/bin/env bash\nprintf "B2 5.1.0\\n"\nexit 1\n', newline='\n')
            tool.chmod(0o755)
            subprocess.run(command, cwd=root, check=True, capture_output=True)
            self.assertFalse(receipt.exists(), 'valid cached B2 was rebuilt because of pipefail')
            tool.write_text('#!/usr/bin/env bash\nexit 126\n', newline='\n')
            subprocess.run(command, cwd=root, check=True, capture_output=True)
            self.assertEqual(receipt.read_text(), 'x', 'wrong-host tool did not bootstrap')
            tool.unlink()
            subprocess.run(command, cwd=root, check=True, capture_output=True)
            self.assertEqual(receipt.read_text(), 'xx', 'missing tool did not bootstrap')

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
