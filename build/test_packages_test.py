import json
from pathlib import Path
import tempfile
import unittest
from test_packages import inventory


class InventoryTests(unittest.TestCase):
    def test_current_inventory(self):
        self.assertIn('rutor', inventory()['native'])

    def test_unlisted_directory_is_rejected(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            (root/'build').mkdir()
            (root/'server/new').mkdir(parents=True)
            (root/'server/new/policy_test.go').touch()
            (root/'build/go-test-packages.json').write_text(json.dumps({'native': []}))
            with self.assertRaisesRegex(ValueError, 'missing='):
                inventory(root)
