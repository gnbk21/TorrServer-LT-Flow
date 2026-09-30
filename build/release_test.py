import unittest
from release import classify


class ReleaseClassification(unittest.TestCase):
    def test_channels(self):
        self.assertEqual(classify("MatriX.145.Flow-v1.0.0"), "stable")
        for tag in ("MatriX.145.Flow-preview.1", "MatriX.145.Flow-v0.2.0-preview.1", "MatriX.145.Flow-v1.0.0-rc.2"):
            self.assertEqual(classify(tag), "preview")

    def test_foreign_or_ambiguous_tags_fail_closed(self):
        for tag in ("latest", "v1.0.0", "MatriX.145", "MatriX.145.Flow-v1.0.0-beta", "MatriX.145.Flow-v1.0.0/../../bad"):
            with self.assertRaises(ValueError):
                classify(tag)


if __name__ == "__main__":
    unittest.main()
