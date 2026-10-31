import json
from pathlib import Path
import tempfile
import unittest
from workspace_storage import inventory, apply_manifest, canonical_file


class WorkspaceStorageTests(unittest.TestCase):
    def test_preview_protection_changes_and_active_files(self):
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory).resolve()
            cache=root/'.tools/buffering-go-cache'
            cache.mkdir(parents=True)
            owned=cache/'owned-d';owned.write_bytes(b'cache')
            settings=cache/'settings.json';settings.write_text('{}')
            tool=root/'.tools/msys64/bin/gcc.exe';tool.parent.mkdir(parents=True);tool.write_bytes(b'compiler')
            evidence=root/'.tools/evidence-archive/manifest.json';evidence.parent.mkdir();evidence.write_text('{}')
            preview=inventory(root,active=[])
            self.assertTrue(owned.exists(), 'preview deleted a file')
            self.assertEqual([r['path'] for r in preview['files']], ['.tools/buffering-go-cache/owned-d'])
            results=apply_manifest(preview,root,active=[owned])
            self.assertEqual(results[0]['status'],'skipped')
            owned.write_bytes(b'changed')
            self.assertEqual(apply_manifest(preview,root,active=[])[0]['status'],'skipped')
            preview=inventory(root,active=[])
            self.assertEqual(apply_manifest(preview,root,active=[])[0]['status'],'removed')
            self.assertTrue(settings.exists() and tool.exists() and evidence.exists())

    def test_canonical_escape_and_tampered_manifest(self):
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory).resolve()
            (root/'settings.json').write_text('{}')
            for path in ('../outside', str(root/'settings.json')):
                with self.assertRaises(ValueError):canonical_file(root,path)
            forged={'schema_version':1,'workspace':str(root),'files':[{'eligible':True,'path':'settings.json'}]}
            self.assertEqual(apply_manifest(forged,root,active=[]),[])
            self.assertTrue((root/'settings.json').exists())
            other=Path(directory).parent
            try:(root/'link').symlink_to(other,target_is_directory=True)
            except OSError:return
            with self.assertRaises((ValueError,OSError)):canonical_file(root,'link/settings.json')
