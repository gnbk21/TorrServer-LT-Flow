"""Real patch application, hardlink isolation, repeatability and drift checks."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

from apply_native_patches import apply, PATCH_DIR, recipe

SOURCE = Path(__file__).resolve().parent.parent / "_src/libtorrent"


class NativePatchTest(unittest.TestCase):
    def fixture(self, root):
        for patch in PATCH_DIR.glob("*.patch"):
            for line in patch.read_text().splitlines():
                if line.startswith("+++ b/"):
                    name = line[6:]
                    path = root / name
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_bytes(subprocess.check_output(
                        ["git", "-C", str(SOURCE), "show", f"v2.1.2:{name}"]))
        subprocess.run(["git", "init", "-q", str(root)], check=True)

    def test_apply_repeat_and_shared_source_isolation(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.fixture(root)
            source = root / "src/peer_connection.cpp"
            shared = root / "shared.cpp"
            os.link(source, shared)
            before = shared.read_bytes()
            apply(root)
            self.assertEqual(shared.read_bytes(), before)
            self.assertNotEqual(source.read_bytes(), before)
            patched = source.read_bytes()
            apply(root)
            self.assertEqual(source.read_bytes(), patched)
            self.assertEqual(len(recipe()), 64)

    def test_reject_source_drift(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.fixture(root)
            source = root / "src/peer_connection.cpp"
            source.write_text(source.read_text().replace("return milliseconds(", "return changed("))
            with self.assertRaises(RuntimeError):
                apply(root)


if __name__ == "__main__":
    unittest.main()
