from pathlib import Path
import subprocess
import tempfile
import unittest

from release_checkout import verify


class CleanReleaseCheckout(unittest.TestCase):
    def test_release_rejects_source_changes_and_unknown_files_without_hiding_dev_state(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            def git(*args):
                subprocess.run(['git','-c','user.name=Flow fixture',
                    '-c','user.email=fixture@example.invalid','-C',str(root),*args],
                    check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
            git('init','--quiet')
            source = root/'source.go'
            source.write_text('committed source\n')
            git('add','source.go');git('commit','--quiet','-m','controlled source')
            self.assertTrue(verify('MatriX.148.Flow-v1.2.0',root))
            source.write_text('uncommitted change\n')
            with self.assertRaisesRegex(ValueError,'uncommitted'):
                verify('MatriX.148.Flow-v1.2.0',root)
            self.assertFalse(verify('MatriX.148.Flow-dev-local',root))
            git('checkout','--','source.go')
            (root/'unknown.go').write_text('unknown source\n')
            with self.assertRaisesRegex(ValueError,'unknown.go'):
                verify('MatriX.148.Flow-v1.2.0-rc.1',root)
