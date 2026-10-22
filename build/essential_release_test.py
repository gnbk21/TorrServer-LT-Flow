import unittest
from essential_release import validate_changes


class EssentialReleaseTests(unittest.TestCase):
    def test_ci_only_repair_and_empty_diff_are_eligible(self):
        validate_changes([])
        validate_changes(['.github/workflows/build.yml', 'build/ci-apt.sh',
                          'build/ci-integration.sh', 'build/workflow_policy_test.py',
                          'build/essential_release.py', 'build/essential_release_test.py',
                          'RELEASE_ACCEPTANCE.json', 'RELEASE_1_1_4_NOTES.md'])

    def test_runtime_and_build_input_changes_require_full_validation(self):
        for name in ('server/flow/session.go', 'server/go.mod', 'server/go.sum',
                     'web/yarn.lock', 'web/src/main.tsx', 'build/_common.sh',
                     'build/_deps.sh', 'build/linux-amd64.sh', 'build/release.py',
                     'build/native-patches/queue.patch', 'distribution/Update-Flow.ps1',
                     'Dockerfile.runtime', 'LICENSE', 'gen_web.go', 'web/README.md'):
            with self.subTest(name=name), self.assertRaises(ValueError):
                validate_changes([name])

    def test_path_tricks_fail_closed(self):
        for name in ('/README.md', '../README.md', '.github/workflows/../../server/a',
                     '.github\\workflows\\build.yml'):
            with self.subTest(name=name), self.assertRaises(ValueError):
                validate_changes([name])
