import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from release import binary_identity_matches, classify, find_binary


class ReleaseClassification(unittest.TestCase):
    def test_channels(self):
        self.assertEqual(classify("MatriX.145.Flow-v1.0.0"), "stable")
        for tag in ("MatriX.145.Flow-preview.1", "MatriX.145.Flow-v0.2.0-preview.1", "MatriX.145.Flow-v1.0.0-rc.2"):
            self.assertEqual(classify(tag), "preview")

    def test_foreign_or_ambiguous_tags_fail_closed(self):
        for tag in ("latest", "v1.0.0", "MatriX.145", "MatriX.145.Flow-v1.0.0-beta", "MatriX.145.Flow-v1.0.0/../../bad", "MatriX.145.Flow-v١.٠.٠", "MatriX.145.Flow-v1.0.0\n"):
            with self.assertRaises(ValueError):
                classify(tag)


class ArtifactSelection(unittest.TestCase):
    def test_artifact_directory_with_binary_name_is_not_a_duplicate(self):
        with TemporaryDirectory() as directory:
            root = Path(directory)
            name = "TorrServer-LT-linux-amd64"
            artifact = root / name
            artifact.mkdir()
            binary = artifact / name
            binary.write_bytes(b"fixture")
            self.assertEqual(find_binary(root, name), binary)

    def test_actual_duplicate_or_missing_binary_is_rejected(self):
        with TemporaryDirectory() as directory:
            root = Path(directory)
            name = "TorrServer-LT-linux-amd64"
            with self.assertRaises(ValueError):
                find_binary(root, name)
            for part in ("one", "two"):
                artifact = root / part
                artifact.mkdir()
                (artifact / name).write_bytes(b"fixture")
            with self.assertRaises(ValueError):
                find_binary(root, name)


class BinaryIdentity(unittest.TestCase):
    def test_exact_go_build_settings(self):
        tag = "MatriX.145.Flow-v0.2.0-preview.4"
        commit = "a" * 40
        info = f'\tbuild\t-ldflags="-s -w -X server/version.Version={tag}"\n\tbuild\tvcs.revision={commit}\n'
        self.assertTrue(binary_identity_matches(info, tag, commit))
        self.assertTrue(binary_identity_matches(info.replace("-X ", "-X="), tag, commit))

    def test_prefixes_wrong_settings_and_malformed_flags_fail_closed(self):
        tag = "MatriX.145.Flow-v0.2.0-preview.4"
        commit = "a" * 40
        valid = f'\tbuild\t-ldflags="-s -w -X server/version.Version={tag}"\n\tbuild\tvcs.revision={commit}\n'
        for invalid in (valid.replace(tag, tag + "0"), valid.replace(commit, "b" * 40), valid.replace("-X ", ""), valid.replace('"\n', '\n', 1), valid + f'\tbuild\tvcs.revision={commit}\n'):
            self.assertFalse(binary_identity_matches(invalid, tag, commit))


if __name__ == "__main__":
    unittest.main()
