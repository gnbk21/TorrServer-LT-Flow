"""Original dependency notices must survive Git/artifact round trips exactly."""
from pathlib import Path
import subprocess
import tempfile
import unittest


class NoticeBytes(unittest.TestCase):
    def test_original_mixed_line_endings_round_trip_without_dirty_checkout(self):
        attributes = (Path(__file__).resolve().parents[1]/'.gitattributes').read_bytes()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            def git(*args):
                return subprocess.check_output(['git','-c','core.autocrlf=false',
                    '-c','user.name=Flow fixture','-c','user.email=fixture@example.invalid',
                    '-C',str(root),*args],stderr=subprocess.DEVNULL)
            git('init','--quiet')
            (root/'.gitattributes').write_bytes(attributes)
            name = 'server/web/pages/template/pages/THIRD_PARTY_NOTICES.txt'
            path = root/name
            path.parent.mkdir(parents=True)
            original = b'Generated heading\n\nOriginal upstream notice\r\nSecond line\r\n'
            path.write_bytes(original)
            git('add','.gitattributes',name)
            self.assertEqual(git('show',':'+name),original)
            git('commit','--quiet','-m','controlled notice fixture')
            path.write_bytes(original)  # Simulate receiving the fresh CI artifact.
            self.assertEqual(git('status','--porcelain'),b'')
